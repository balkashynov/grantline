// Package terraform imports only the overlap-phase state representation from
// terraform show -json. Plans and raw .tfstate files are deliberately rejected.
package terraform

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"regexp"
	"strings"

	"github.com/balkashynov/grantline/internal/compare"
)

type resource struct {
	Address  string                     `json:"address"`
	Mode     string                     `json:"mode"`
	Type     string                     `json:"type"`
	Provider string                     `json:"provider_name"`
	Schema   *int                       `json:"schema_version"`
	Values   map[string]json.RawMessage `json:"values"`
}
type module struct {
	Resources []resource `json:"resources"`
	Children  []module   `json:"child_modules"`
}
type state struct {
	Format string `json:"format_version"`
	Values *struct {
		Root *module `json:"root_module"`
	} `json:"values"`
	Planned json.RawMessage `json:"planned_values"`
	Changes json.RawMessage `json:"resource_changes"`
}

func str(r resource, k string) (string, error) {
	var s string
	if json.Unmarshal(r.Values[k], &s) != nil || s == "" {
		return "", fmt.Errorf("missing or unresolved %s", k)
	}
	return s, nil
}
func optional(r resource, k string) string {
	var s string
	_ = json.Unmarshal(r.Values[k], &s)
	return s
}
func roles(r resource, k string) ([]string, error) {
	var a []string
	if json.Unmarshal(r.Values[k], &a) != nil || a == nil {
		return nil, fmt.Errorf("missing or unresolved %s", k)
	}
	return a, nil
}
func compatible(r resource) error {
	if r.Provider != "registry.terraform.io/mongodb/mongodbatlas" {
		return fmt.Errorf("unsupported provider for relevant resource")
	}
	if r.Schema == nil || *r.Schema != 0 {
		return fmt.Errorf("unsupported resource schema version; expected 0")
	}
	return nil
}
func flatten(m module, out *[]resource, depth int) error {
	if depth > 64 {
		return fmt.Errorf("module nesting exceeds 64")
	}
	*out = append(*out, m.Resources...)
	if len(*out) > 100000 {
		return fmt.Errorf("state exceeds 100000 resources")
	}
	for _, c := range m.Children {
		if err := flatten(c, out, depth+1); err != nil {
			return err
		}
	}
	return nil
}
func Import(data []byte, from, to string, required *bool) (compare.Input, error) {
	var s state
	var in compare.Input
	if len(data) > compare.MaxInput {
		return in, fmt.Errorf("state exceeds 16 MiB")
	}
	if json.Unmarshal(data, &s) != nil {
		return in, fmt.Errorf("invalid Terraform JSON; input contents are omitted from errors")
	}
	if len(s.Planned) > 0 || len(s.Changes) > 0 {
		return in, fmt.Errorf("plan exports are unsupported; use terraform show -json on overlap-phase STATE after both identities exist")
	}
	if !regexp.MustCompile(`^1\.[0-9]+$`).MatchString(s.Format) || s.Values == nil || s.Values.Root == nil {
		return in, fmt.Errorf("expected terraform show -json state format 1.x with values.root_module")
	}
	var all []resource
	if err := flatten(*s.Values.Root, &all, 0); err != nil {
		return in, err
	}
	selected := map[string]resource{}
	seen := map[string]bool{}
	for _, r := range all {
		if r.Mode != "managed" {
			continue
		}
		if r.Address == "" || seen[r.Address] {
			return in, fmt.Errorf("missing or duplicate managed resource address")
		}
		seen[r.Address] = true
		selected[r.Address] = r
	}
	old, ok := selected[from]
	if !ok {
		return in, fmt.Errorf("source resource address not found among managed resources")
	}
	next, ok := selected[to]
	if !ok {
		return in, fmt.Errorf("target resource address not found among managed resources")
	}
	if old.Type != "mongodbatlas_api_key" || next.Type != "mongodbatlas_service_account" {
		return in, fmt.Errorf("select an organization mongodbatlas_api_key and mongodbatlas_service_account; project-created identity resources are unsupported")
	}
	before, err := identity(old, all, "api_key_id", "role_names", required)
	if err != nil {
		return in, fmt.Errorf("source: %w", err)
	}
	after, err := identity(next, all, "client_id", "roles", required)
	if err != nil {
		return in, fmt.Errorf("target: %w", err)
	}
	in = compare.Input{SchemaVersion: 1, Scope: "terraform-managed", Before: before, After: after}
	if _, err := compare.Analyze(in); err != nil {
		return in, err
	}
	return in, nil
}
func identity(primary resource, all []resource, idKey, roleKey string, required *bool) (compare.Identity, error) {
	i := compare.Identity{Label: primary.Address, Grants: []compare.Grant{}, AccessList: []string{}, RequireIPList: required}
	if err := compatible(primary); err != nil {
		return i, err
	}
	id, err := str(primary, idKey)
	if err != nil {
		return i, err
	}
	org, err := str(primary, "org_id")
	if err != nil {
		return i, err
	}
	i.OrgID = org
	rs, err := roles(primary, roleKey)
	if err != nil {
		return i, err
	}
	for _, r := range rs {
		i.Grants = append(i.Grants, compare.Grant{Scope: "org:" + org, Role: r})
	}
	for _, r := range all {
		if r.Mode != "managed" || r.Address == primary.Address {
			continue
		}
		relevantType := strings.Contains(r.Type, "api_key") || strings.Contains(r.Type, "service_account")
		if !relevantType {
			continue
		}
		rid := optional(r, idKey)
		if rid == "" {
			// An unresolved supported reference could belong to this identity. Do not
			// silently turn it into absence and return a reassuring equality result.
			expected := (idKey == "api_key_id" && (r.Type == "mongodbatlas_api_key_project_assignment" || r.Type == "mongodbatlas_access_list_api_key")) || (idKey == "client_id" && (r.Type == "mongodbatlas_service_account_project_assignment" || r.Type == "mongodbatlas_service_account_access_list_entry" || r.Type == "mongodbatlas_project_service_account_access_list_entry"))
			if expected {
				return i, fmt.Errorf("an access resource has a missing or unresolved identity reference")
			}
			continue
		}
		if rid != id {
			continue
		}
		if err := compatible(r); err != nil {
			return i, err
		}
		switch r.Type {
		case "mongodbatlas_api_key_project_assignment", "mongodbatlas_service_account_project_assignment":
			p, err := str(r, "project_id")
			if err != nil {
				return i, err
			}
			rr, err := roles(r, "roles")
			if err != nil {
				return i, err
			}
			for _, role := range rr {
				i.Grants = append(i.Grants, compare.Grant{Scope: "project:" + p, Role: role})
			}
		case "mongodbatlas_access_list_api_key", "mongodbatlas_service_account_access_list_entry":
			o, err := str(r, "org_id")
			if err != nil || !strings.EqualFold(o, org) {
				return i, fmt.Errorf("access-list resource has inconsistent organization")
			}
			cidr, ip := optional(r, "cidr_block"), optional(r, "ip_address")
			if cidr == "" && ip == "" {
				return i, fmt.Errorf("access-list resource is missing its address")
			}
			// Atlas may populate both computed fields for a single-host entry.
			// Accept only equivalent singleton forms; ambiguous dual fields fail.
			if cidr != "" && ip != "" {
				p, pe := netip.ParsePrefix(cidr)
				a, ae := netip.ParseAddr(ip)
				if pe != nil || ae != nil || p.Bits() != p.Addr().BitLen() || p.Addr() != a {
					return i, fmt.Errorf("access-list resource has ambiguous CIDR and IP fields; only equivalent single-host forms are supported")
				}
			}
			if cidr != "" {
				i.AccessList = append(i.AccessList, cidr)
			} else {
				i.AccessList = append(i.AccessList, ip)
			}
		case "mongodbatlas_service_account_secret", "mongodbatlas_project_service_account_secret":
			// Secrets and their lifecycle are explicitly outside the comparison.
		default:
			return i, fmt.Errorf("an additional identity/access resource references the selected principal but its type is unsupported")
		}
	}
	return i, nil
}
