package compare

import (
	"os"
	"testing"
)

func example(t *testing.T, name string) Input {
	t.Helper()
	b, e := os.ReadFile("../../examples/" + name + ".json")
	if e != nil {
		t.Fatal(e)
	}
	in, e := Decode(b)
	if e != nil {
		t.Fatal(e)
	}
	return in
}
func TestScenarios(t *testing.T) {
	for name, want := range map[string]string{"equivalent": "equal_in_scope", "lost-access": "differences_found", "empty-list": "differences_found", "unknown-policy": "incomplete", "role-change": "differences_found", "ipv6-gap": "differences_found"} {
		t.Run(name, func(t *testing.T) {
			r, e := Analyze(example(t, name))
			if e != nil || r.Status != want {
				t.Fatal(r, e)
			}
		})
	}
}
func TestExactScopedRoles(t *testing.T) {
	in := example(t, "equivalent")
	in.After.Grants[1].Scope = "project:555555555555555555555555"
	r, e := Analyze(in)
	if e != nil || len(r.RemovedGrants) != 1 || len(r.AddedGrants) != 1 {
		t.Fatal(r, e)
	}
}
func TestDuplicateRolesAreASet(t *testing.T) {
	in := example(t, "equivalent")
	in.After.Grants = append(in.After.Grants, in.After.Grants...)
	r, e := Analyze(in)
	if e != nil || r.Status != "equal_in_scope" {
		t.Fatal(r, e)
	}
}
func TestNoRoleHierarchyAssumption(t *testing.T) {
	in := example(t, "role-change")
	r, e := Analyze(in)
	if e != nil || len(r.RemovedGrants) != 1 || len(r.AddedGrants) != 1 {
		t.Fatal(r, e)
	}
}
func TestUnknownFieldsAndTrailingJSONRejected(t *testing.T) {
	for _, s := range []string{`{"private_key":"secret"}`, `{} {}`} {
		if _, e := Decode([]byte(s)); e == nil {
			t.Fatal("invalid JSON accepted")
		}
	}
}
func TestInputValidation(t *testing.T) {
	for _, mutate := range []func(*Input){func(i *Input) { i.Before.AccessList = nil }, func(i *Input) { i.After.OrgID = "555555555555555555555555" }, func(i *Input) { i.SchemaVersion = 2 }, func(i *Input) { i.Before.Grants[0].Role = "GROUP_OWNER" }} {
		in := example(t, "equivalent")
		mutate(&in)
		if _, e := Analyze(in); e == nil {
			t.Fatal("invalid comparison accepted")
		}
	}
}
