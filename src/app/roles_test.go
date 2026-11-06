package main

import (
	"strings"
	"testing"

	"github.com/megaproaktiv/g-plan/src/app/plan"
)

func roleApp(t *testing.T) (*App, string) {
	a := NewApp()
	if _, err := a.AddWorkPackage("Work", "2026-01-01", "2026-02-01"); err != nil {
		t.Fatal(err)
	}
	id := a.project.Packages[0].ID
	a.AddRole("Dev", "developer")
	a.AddRole("Test", "tester")
	a.AddRole("Ops", "operations")
	return a, id
}

func TestAddRoleAssignsStableIDs(t *testing.T) {
	a, id := roleApp(t)
	d := toDTO(a.project)
	if d.Roles[0].ID != 1 || d.Roles[1].ID != 2 || d.Roles[2].ID != 3 {
		t.Fatalf("ids: %+v", d.Roles)
	}
	// roles auto-created from the action editor get an ID too
	if _, err := a.SetActionRoles(id, []string{"Auto"}, []int{10}); err != nil {
		t.Fatal(err)
	}
	if r := a.project.RoleByName("Auto"); r == nil || r.ID != 4 {
		t.Fatalf("auto-created role: %+v", r)
	}
	// IDs do not shift when an earlier role is deleted
	if _, err := a.RemoveRole("Dev"); err != nil {
		t.Fatal(err)
	}
	if r := a.project.RoleByName("Test"); r.ID != 2 {
		t.Errorf("Test changed its ID to %d", r.ID)
	}
	a.AddRole("New", "")
	if r := a.project.RoleByName("New"); r.ID != 5 {
		t.Errorf("new role should get the next number after the highest (5), got %d", r.ID)
	}
}

func TestRemoveRoleRefusedWhileReferenced(t *testing.T) {
	a, id := roleApp(t)
	if _, err := a.SetActionRoles(id, []string{"Dev"}, []int{50}); err != nil {
		t.Fatal(err)
	}
	_, err := a.RemoveRole("Dev")
	if err == nil || !strings.Contains(err.Error(), "Work") {
		t.Fatalf("assigned role must not be deletable, naming the action: %v", err)
	}
	dto := toDTO(a.project)
	var dev RoleDTO
	for _, r := range dto.Roles {
		if r.Name == "Dev" {
			dev = r
		}
	}
	if !dev.InUse || !strings.Contains(dev.UsedBy, "Work") {
		t.Errorf("dto must expose the usage: %+v", dev)
	}
	// unassign, then deletion works
	if _, err := a.SetActionRoles(id, nil, nil); err != nil {
		t.Fatal(err)
	}
	dto, err = a.RemoveRole("Dev")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range dto.Roles {
		if r.Name == "Dev" {
			t.Error("role still present")
		}
	}
	if _, err := a.RemoveRole("Dev"); err == nil {
		t.Error("deleting it twice must report an error")
	}
}

func TestRenameRoleViaBinding(t *testing.T) {
	a, id := roleApp(t)
	if _, err := a.SetActionRoles(id, []string{"Dev"}, []int{50}); err != nil {
		t.Fatal(err)
	}
	a.project.Packages[0].Details = append(a.project.Packages[0].Details,
		plan.DetailPackage{ID: "d1", Name: "Detail", Role: "Dev"})
	dto, err := a.SetRoleFields("Dev", "Developer", "new description")
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, r := range dto.Roles {
		if r.Name == "Developer" {
			found = true
			if r.ID != 1 || r.Description != "new description" || len(r.Assignments) != 1 {
				t.Errorf("renamed role: %+v", r)
			}
		}
		if r.Name == "Dev" {
			t.Error("old name still present")
		}
	}
	if !found {
		t.Fatal("renamed role missing")
	}
	if a.project.Packages[0].Details[0].Role != "Developer" {
		t.Error("detail reference not renamed")
	}
	if _, err := a.SetRoleFields("Developer", "Test", ""); err == nil {
		t.Error("renaming to an existing role name must fail")
	}
	if a.project.RoleByName("Developer") == nil {
		t.Error("failed rename must leave the role alone")
	}
	// empty new name keeps the name but still updates the description
	if _, err := a.SetRoleFields("Developer", "", "only desc"); err != nil || a.project.RoleByName("Developer").Description != "only desc" {
		t.Errorf("empty name should keep the name: %v", err)
	}
}

func TestChatRoleRefUsesStableID(t *testing.T) {
	a, _ := roleApp(t)
	if _, err := a.RemoveRole("Dev"); err != nil { // IDs now: Test=2, Ops=3
		t.Fatal(err)
	}
	if got := a.resolveRoleRef("R2"); got != "Test" {
		t.Errorf("R2 = %q, want Test (by ID, not list position)", got)
	}
	if got := a.resolveRoleRef("3"); got != "Ops" {
		t.Errorf("3 = %q, want Ops", got)
	}
	if got := a.resolveRoleRef("R1"); got != "" {
		t.Errorf("R1 was deleted and must not resolve to another role, got %q", got)
	}
	if got := a.resolveRoleRef("Ops"); got != "Ops" {
		t.Errorf("by name: %q", got)
	}
}
