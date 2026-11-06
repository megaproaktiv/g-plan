package plan

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

func roleProject() *Project {
	p := NewProject("R", UnitDays)
	s, _ := ParseDateStr("2026-01-01")
	e, _ := ParseDateStr("2026-02-01")
	p.Packages = []WorkPackage{
		{ID: "a", Name: "Action A", Start: s, End: e, Mode: ModeFixed,
			Details: []DetailPackage{{ID: "a_d1", Name: "Detail 1", Role: "Dev", Start: s, End: e}}},
		{ID: "b", Name: "Action B", Start: s, End: e, Mode: ModeFixed},
	}
	p.Roles = []Role{
		{ID: 1, Name: "Dev", Assignments: []Assignment{{WorkPackageID: "a", Percent: 50}}},
		{ID: 2, Name: "Test"},
		{ID: 3, Name: "Ops"},
	}
	return p
}

func TestEnsureRoleIDs(t *testing.T) {
	p := NewProject("x", UnitDays)
	p.Roles = []Role{{Name: "A"}, {Name: "B"}, {Name: "C"}}
	p.EnsureRoleIDs()
	for i, r := range p.Roles {
		if r.ID != i+1 {
			t.Errorf("legacy role %d got ID %d, want %d", i, r.ID, i+1)
		}
	}
	// existing IDs are kept; missing and duplicate ones get fresh numbers
	p.Roles = []Role{{ID: 7, Name: "A"}, {Name: "B"}, {ID: 7, Name: "C"}, {ID: 2, Name: "D"}}
	p.EnsureRoleIDs()
	if p.Roles[0].ID != 7 || p.Roles[3].ID != 2 {
		t.Errorf("existing unique IDs must be kept: %+v", p.Roles)
	}
	seen := map[int]bool{}
	for _, r := range p.Roles {
		if r.ID <= 0 || seen[r.ID] {
			t.Errorf("IDs must be positive and unique: %+v", p.Roles)
		}
		seen[r.ID] = true
	}
}

// Feature: roles - IDs are always unique among the existing roles, and a
// surviving role never changes its ID when other roles are added or deleted.
func TestProp_RoleIDsUniqueAndStable(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		p := NewProject("x", UnitDays)
		ids := map[string]int{} // name -> ID at creation
		n := 0
		steps := rapid.IntRange(1, 30).Draw(t, "steps")
		for step := 0; step < steps; step++ {
			if len(p.Roles) > 0 && rapid.Bool().Draw(t, "del") {
				i := rapid.IntRange(0, len(p.Roles)-1).Draw(t, "i")
				name := p.Roles[i].Name
				if err := p.DeleteRole(name); err != nil {
					t.Fatal(err)
				}
				delete(ids, name)
				continue
			}
			n++
			name := fmt.Sprintf("role%d", n)
			p.Roles = append(p.Roles, Role{ID: p.NextRoleID(), Name: name})
			ids[name] = p.Roles[len(p.Roles)-1].ID
		}
		seen := map[int]bool{}
		for _, r := range p.Roles {
			if r.ID <= 0 || seen[r.ID] {
				t.Fatalf("IDs must be positive and unique: %+v", p.Roles)
			}
			seen[r.ID] = true
			if ids[r.Name] != r.ID {
				t.Fatalf("role %s changed its ID from %d to %d", r.Name, ids[r.Name], r.ID)
			}
		}
	})
}

func TestDeleteRoleOnlyWithoutReferences(t *testing.T) {
	p := roleProject()
	err := p.DeleteRole("Dev")
	if err == nil {
		t.Fatal("role with an assignment and a detail must not be deleted")
	}
	for _, want := range []string{"Action A", "Detail 1", "cannot be deleted"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %q: %v", want, err)
		}
	}
	if len(p.Roles) != 3 || len(p.RoleByName("Dev").Assignments) != 1 {
		t.Fatal("a refused delete must change nothing")
	}
	// only the detail reference left -> still refused
	p.RoleByName("Dev").Assignments = nil
	if p.DeleteRole("Dev") == nil {
		t.Error("detail reference must block deletion")
	}
	// a 0% assignment is still a reference
	p.Packages[0].Details[0].Role = ""
	p.RoleByName("Dev").Assignments = []Assignment{{WorkPackageID: "b", Percent: 0}}
	if p.DeleteRole("Dev") == nil {
		t.Error("a 0% assignment is still a reference")
	}
	// free it, then it can go; the others keep their IDs
	p.RoleByName("Dev").Assignments = nil
	if err := p.DeleteRole("Dev"); err != nil {
		t.Fatal(err)
	}
	if p.RoleByName("Dev") != nil || len(p.Roles) != 2 || p.Roles[0].ID != 2 || p.Roles[1].ID != 3 {
		t.Errorf("remaining roles must keep their IDs: %+v", p.Roles)
	}
	if err := p.DeleteRole("Dev"); err == nil {
		t.Error("deleting an unknown role is an error")
	}
	// unreferenced role deletes straight away
	if err := p.DeleteRole("Ops"); err != nil {
		t.Errorf("unreferenced: %v", err)
	}
}

func TestRenameRoleKeepsReferences(t *testing.T) {
	p := roleProject()
	if err := p.RenameRole("Dev", "Developer"); err != nil {
		t.Fatal(err)
	}
	r := p.RoleByName("Developer")
	if r == nil || r.ID != 1 || len(r.Assignments) != 1 {
		t.Fatalf("rename must keep ID and assignments: %+v", r)
	}
	if p.Packages[0].Details[0].Role != "Developer" {
		t.Errorf("detail reference not updated: %q", p.Packages[0].Details[0].Role)
	}
	if p.RoleByName("Dev") != nil {
		t.Error("old name must be gone")
	}
	if !p.RoleUsage("Developer").InUse() || p.RoleUsage("Dev").InUse() {
		t.Error("usage must follow the new name")
	}
}

func TestRenameRoleRules(t *testing.T) {
	p := roleProject()
	if p.RenameRole("Dev", "test") == nil {
		t.Error("duplicate name (case-insensitive) must be rejected")
	}
	if p.RenameRole("Dev", "Test") == nil {
		t.Error("duplicate name must be rejected")
	}
	if p.RenameRole("Dev", "   ") == nil {
		t.Error("empty name must be rejected")
	}
	if p.RenameRole("Nope", "X") == nil {
		t.Error("unknown role")
	}
	if p.RoleByName("Dev") == nil || p.Packages[0].Details[0].Role != "Dev" {
		t.Error("rejected renames must change nothing")
	}
	if err := p.RenameRole("Dev", "DEV"); err != nil {
		t.Errorf("changing only the case of the own name is fine: %v", err)
	}
	if err := p.RenameRole("DEV", "  Lead  "); err != nil || p.RoleByName("Lead") == nil {
		t.Errorf("trim: %v", err)
	}
}

func TestRoleIDPersistsAndLegacyFilesGetIDs(t *testing.T) {
	p := roleProject()
	path := filepath.Join(t.TempDir(), "r.ptplan")
	if err := p.SaveJSON(path); err != nil {
		t.Fatal(err)
	}
	q, err := OpenJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range q.Roles {
		if r.ID != p.Roles[i].ID {
			t.Errorf("role %s: ID %d != %d", r.Name, r.ID, p.Roles[i].ID)
		}
	}
	// a file written before IDs existed
	for i := range p.Roles {
		p.Roles[i].ID = 0
	}
	old := filepath.Join(t.TempDir(), "old.ptplan")
	if err := p.SaveJSON(old); err != nil {
		t.Fatal(err)
	}
	o, err := OpenJSON(old)
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range o.Roles {
		if r.ID != i+1 {
			t.Errorf("legacy role %s got ID %d, want %d", r.Name, r.ID, i+1)
		}
	}
}
