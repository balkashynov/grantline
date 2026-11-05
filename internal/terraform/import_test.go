package terraform

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/balkashynov/grantline/internal/compare"
)

func fixture(t *testing.T) []byte {
	t.Helper()
	b, e := os.ReadFile("../../examples/terraform-overlap.synthetic.json")
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func load(t *testing.T, b []byte, p *bool) (compare.Input, error) {
	t.Helper()
	return Import(b, "mongodbatlas_api_key.legacy", "mongodbatlas_service_account.replacement", p)
}
func TestNestedModuleImportAndSecretOmission(t *testing.T) {
	p := true
	in, e := load(t, fixture(t), &p)
	if e != nil {
		t.Fatal(e)
	}
	r, e := compare.Analyze(in)
	if e != nil {
		t.Fatal(e)
	}
	if r.Status != "differences_found" || len(r.RemovedGrants) != 1 || r.RemovedGrants[0].Role != "GROUP_DATA_ACCESS_READ_ONLY" || r.Network.Removed[0].CIDR != "192.0.2.128/25" {
		t.Fatalf("unexpected %v", r)
	}
	b, _ := json.Marshal(in)
	report, _ := json.Marshal(r)
	for _, s := range []string{string(b), string(report)} {
		if strings.Contains(s, "SYNTHETIC_SECRET") || strings.Contains(s, "private_key") || strings.Contains(s, "client_secret") {
			t.Fatal("secret field leaked")
		}
	}
}
func TestUnknownPolicyCannotPass(t *testing.T) {
	in, e := load(t, fixture(t), nil)
	if e != nil {
		t.Fatal(e)
	}
	r, e := compare.Analyze(in)
	if e != nil || r.Status != "incomplete" || r.Network != nil {
		t.Fatal(r, e)
	}
}
func TestRejectMalformedImports(t *testing.T) {
	tests := []struct{ name, old, next string }{
		{"plan", "\"format_version\": \"1.0\"", "\"planned_values\": {}, \"format_version\": \"1.0\""},
		{"future format", "\"format_version\": \"1.0\"", "\"format_version\": \"2.0\""},
		{"future schema", "\"schema_version\": 0", "\"schema_version\": 1"},
		{"unknown id", "\"api_key_id\": \"333333333333333333333333\"", "\"api_key_id\": null"},
		{"unknown roles", "\"role_names\": [\n              \"ORG_MEMBER\"\n            ]", "\"role_names\": null"},
		{"bad cidr", "192.0.2.0/24", "SYNTHETIC_SECRET_MUST_NEVER_APPEAR_IN_OUTPUT"},
		{"wrong organization", "\"org_id\": \"111111111111111111111111\"", "\"org_id\": \"555555555555555555555555\""},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			b := string(fixture(t))
			if !strings.Contains(b, c.old) {
				t.Fatal("mutation did not match")
			}
			b = strings.Replace(b, c.old, c.next, 1)
			p := true
			_, e := load(t, []byte(b), &p)
			if e == nil {
				t.Fatal("invalid import accepted")
			}
			if strings.Contains(e.Error(), "SYNTHETIC_SECRET") {
				t.Fatal("error leaked secret")
			}
		})
	}
}

func TestComputedSingletonFields(t *testing.T) {
	b := string(fixture(t))
	b = strings.Replace(b, `"cidr_block": "192.0.2.0/24"`, `"cidr_block": "192.0.2.10/32"`, 1)
	b = strings.Replace(b, `"ip_address": null`, `"ip_address": "192.0.2.10"`, 1)
	p := true
	if _, e := load(t, []byte(b), &p); e != nil {
		t.Fatal(e)
	}
	b = strings.Replace(b, `"ip_address": "192.0.2.10"`, `"ip_address": "192.0.2.11"`, 1)
	if _, e := load(t, []byte(b), &p); e == nil {
		t.Fatal("ambiguous computed fields accepted")
	}
}
