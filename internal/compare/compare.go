package compare

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
)

const MaxInput = 16 << 20

type Grant struct {
	Scope string `json:"scope"`
	Role  string `json:"role"`
}
type Identity struct {
	Label         string   `json:"label"`
	OrgID         string   `json:"orgId"`
	Grants        []Grant  `json:"grants"`
	AccessList    []string `json:"accessList"`
	RequireIPList *bool    `json:"requireIpList"`
}
type Input struct {
	SchemaVersion int      `json:"schemaVersion"`
	Scope         string   `json:"scope"`
	Before        Identity `json:"before"`
	After         Identity `json:"after"`
}
type NetworkReport struct {
	Removed []NetworkDelta `json:"removed"`
	Added   []NetworkDelta `json:"added"`
	Before  []NetworkDelta `json:"before"`
	After   []NetworkDelta `json:"after"`
}
type Report struct {
	SchemaVersion int            `json:"schemaVersion"`
	Status        string         `json:"status"`
	InputSHA256   string         `json:"inputSha256"`
	Scope         string         `json:"scope"`
	Before        string         `json:"before"`
	After         string         `json:"after"`
	RemovedGrants []Grant        `json:"removedGrants"`
	AddedGrants   []Grant        `json:"addedGrants"`
	Network       *NetworkReport `json:"network,omitempty"`
	Issues        []string       `json:"issues"`
	Limits        []string       `json:"limits"`
	NextSteps     []string       `json:"nextSteps"`
}

var idPattern = regexp.MustCompile(`^[a-fA-F0-9]{24}$`)
var rolePattern = regexp.MustCompile(`^(ORG|GROUP)_[A-Z_]+$`)

func Decode(data []byte) (Input, error) {
	var in Input
	if len(data) > MaxInput {
		return in, fmt.Errorf("input exceeds 16 MiB")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&in); err != nil {
		return in, fmt.Errorf("invalid comparison JSON: check the documented schema; unexpected fields are not accepted")
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return in, fmt.Errorf("input must contain exactly one JSON document")
	}
	return in, nil
}
func validate(i Identity) error {
	if len(i.Label) == 0 || len(i.Label) > 160 || strings.ContainsAny(i.Label, "\r\n\x00") {
		return fmt.Errorf("identity label must be 1-160 characters on one line")
	}
	if !idPattern.MatchString(i.OrgID) {
		return fmt.Errorf("orgId must be a 24-character hexadecimal Atlas organization ID")
	}
	if i.Grants == nil || i.AccessList == nil {
		return fmt.Errorf("grants and accessList must be explicit arrays, even when empty")
	}
	if len(i.Grants) > 10000 {
		return fmt.Errorf("grants exceed 10000 entries")
	}
	for j, g := range i.Grants {
		parts := strings.Split(g.Scope, ":")
		if len(parts) != 2 || !idPattern.MatchString(parts[1]) || !rolePattern.MatchString(g.Role) {
			return fmt.Errorf("grant %d requires org:ID or project:ID and an Atlas role name", j+1)
		}
		if parts[0] == "org" {
			if !strings.EqualFold(parts[1], i.OrgID) || !strings.HasPrefix(g.Role, "ORG_") {
				return fmt.Errorf("grant %d has inconsistent organization scope", j+1)
			}
		} else if parts[0] == "project" {
			if !strings.HasPrefix(g.Role, "GROUP_") {
				return fmt.Errorf("grant %d has inconsistent project scope", j+1)
			}
		} else {
			return fmt.Errorf("grant %d has unsupported scope", j+1)
		}
	}
	// Parse even when policy is unknown so malformed networks never disappear.
	_, err := networks(i.AccessList, true)
	return err
}
func grantSet(gs []Grant) map[Grant]bool {
	m := map[Grant]bool{}
	for _, g := range gs {
		g.Scope = strings.ToLower(g.Scope)
		m[g] = true
	}
	return m
}
func grantDifference(a, b map[Grant]bool) []Grant {
	out := []Grant{}
	for g := range a {
		if !b[g] {
			out = append(out, g)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Scope == out[j].Scope {
			return out[i].Role < out[j].Role
		}
		return out[i].Scope < out[j].Scope
	})
	return out
}
func Analyze(in Input) (Report, error) {
	r := Report{SchemaVersion: 1, Scope: in.Scope, Before: in.Before.Label, After: in.After.Label, Issues: []string{}, Status: "equal_in_scope", Limits: []string{
		"Compares exact scoped role assignments, not effective permissions or role inheritance.",
		"Only the provided configuration is compared; unmanaged Atlas resources and drift are not discovered.",
		"Does not validate secrets, token issuance, API operation support, or application behavior. Equality is not permission to retire a key.",
		"IPv6 is supported by the set engine; acceptance of an address by Atlas is not tested.",
	}, NextSteps: []string{"Review every removed and added role assignment and address range.", "Confirm organization IP policy and reconcile resources outside this snapshot.", "Test replacement authentication and representative API operations from real application egress addresses.", "Follow MongoDB's overlap migration procedure; retire the old key only after application verification."}}
	if in.SchemaVersion != 1 {
		return r, fmt.Errorf("unsupported comparison schemaVersion; expected 1")
	}
	if in.Scope != "provided-snapshot" && in.Scope != "terraform-managed" {
		return r, fmt.Errorf("scope must be provided-snapshot or terraform-managed")
	}
	if err := validate(in.Before); err != nil {
		return r, fmt.Errorf("before: %w", err)
	}
	if err := validate(in.After); err != nil {
		return r, fmt.Errorf("after: %w", err)
	}
	if !strings.EqualFold(in.Before.OrgID, in.After.OrgID) {
		return r, fmt.Errorf("cross-organization migration is unsupported")
	}
	b, _ := json.Marshal(in)
	hash := sha256.Sum256(b)
	r.InputSHA256 = hex.EncodeToString(hash[:])
	before, after := grantSet(in.Before.Grants), grantSet(in.After.Grants)
	r.RemovedGrants = grantDifference(before, after)
	r.AddedGrants = grantDifference(after, before)
	if len(r.RemovedGrants)+len(r.AddedGrants) > 0 {
		r.Status = "differences_found"
	}
	if in.Before.RequireIPList == nil || in.After.RequireIPList == nil {
		r.Status = "incomplete"
		r.Issues = append(r.Issues, "Organization Require IP Access List policy is unknown. Supply true or false for both identities; network equality is not evaluated.")
		return r, nil
	}
	old, _ := networks(in.Before.AccessList, *in.Before.RequireIPList)
	next, _ := networks(in.After.AccessList, *in.After.RequireIPList)
	r.Network = &NetworkReport{Removed: describe(difference(old, next)), Added: describe(difference(next, old)), Before: describe(old), After: describe(next)}
	if len(r.Network.Removed)+len(r.Network.Added) > 0 {
		r.Status = "differences_found"
	}
	return r, nil
}
