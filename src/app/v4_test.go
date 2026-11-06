package main

import "testing"

// TestPhasesHasNoNumber verifies the top "Phases" section carries no WP number
// and other sections number from 1 in order.
func TestPhasesHasNoNumber(t *testing.T) {
	a := NewApp()
	a.AddSection("Phases")
	a.AddSection("Design")
	a.AddSection("Build")
	w1 := mustWP(t, a, "p1", "2026-01-01", "2026-01-10")
	w2 := mustWP(t, a, "d1", "2026-01-01", "2026-01-10")
	w3 := mustWP(t, a, "b1", "2026-01-01", "2026-01-10")
	a.SetActionCluster(w1, "Phases")
	a.SetActionCluster(w2, "Design")
	a.SetActionCluster(w3, "Build")
	d := a.GetProject()
	if _, ok := d.SectionNums["Phases"]; ok {
		t.Errorf("Phases must have no WP number, got %d", d.SectionNums["Phases"])
	}
	if d.SectionNums["Design"] != 1 {
		t.Errorf("Design WP num = %d, want 1", d.SectionNums["Design"])
	}
	if d.SectionNums["Build"] != 2 {
		t.Errorf("Build WP num = %d, want 2", d.SectionNums["Build"])
	}
	// a Phases action displays as P<k>, not 0.k
	for _, w := range d.Packages {
		if w.ID == w1 && w.DisplayID != "P1" {
			t.Errorf("Phases action DisplayID = %q, want P1", w.DisplayID)
		}
	}
}

// TestRoleDistinctColors verifies new roles get distinct default colors and
// SetRoleColor rejects a duplicate.
func TestRoleDistinctColors(t *testing.T) {
	a := NewApp()
	a.AddRole("A", "")
	a.AddRole("B", "")
	a.AddRole("C", "")
	cols := map[string]bool{}
	for _, r := range a.project.Roles {
		if r.Color == "" {
			t.Fatalf("role %q has no default color", r.Name)
		}
		if cols[r.Color] {
			t.Fatalf("duplicate default color %s", r.Color)
		}
		cols[r.Color] = true
	}
	// setting B to A's color must be rejected
	aColor := a.project.Roles[0].Color
	if _, err := a.SetRoleColor("B", aColor); err == nil {
		t.Fatal("expected rejection setting B to A's color")
	}
	// a free color is accepted
	if _, err := a.SetRoleColor("B", "#123456"); err != nil {
		t.Fatalf("set B color: %v", err)
	}
}

// TestSetModeRejectsPerson verifies "person" is gone.
func TestSetModeRejectsPerson(t *testing.T) {
	a := NewApp()
	if _, err := a.SetMode("person"); err == nil {
		t.Fatal("expected person mode to be rejected")
	}
	for _, m := range []string{"action", "role", "section"} {
		if _, err := a.SetMode(m); err != nil {
			t.Errorf("mode %q rejected: %v", m, err)
		}
	}
}

// TestShowPhasesPersists verifies ShowPhases round-trips (default true, saved
// false stays false).
func TestShowPhasesPersists(t *testing.T) {
	a := NewApp()
	if !a.project.ShowPhases {
		t.Fatal("ShowPhases should default true")
	}
	a.SetViewOptions(false, false, false, false) // turn Phases off
	dir := t.TempDir()
	base := dir + "/proj"
	saved, err := a.SaveProject(base)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	b := NewApp()
	if _, err := b.OpenProject(saved); err != nil {
		t.Fatalf("open: %v", err)
	}
	if b.project.ShowPhases {
		t.Fatal("ShowPhases should have persisted as false")
	}
}
