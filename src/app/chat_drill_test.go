package main

import "testing"

// With a detail plane open (SetDrill), `add action ... from ... to ...` in the
// chat adds a DETAIL of the drilled action, not a new top-level action.
func TestAddInDrillAddsDetail(t *testing.T) {
	a := NewApp()
	if _, err := a.AddWorkPackage("Parent", "2026-01-01", "2026-06-01"); err != nil {
		t.Fatal(err)
	}
	id := a.project.Packages[0].ID
	top := len(a.project.Packages)

	a.SetDrill(id)
	if out, err := a.Chat(`add action "Phase one" from 2026-02-01 to 2026-03-01`); err != nil {
		t.Fatalf("add in drill: %v (%s)", err, out)
	}
	if len(a.project.Packages) != top {
		t.Errorf("a top-level action was created (%d, want %d)", len(a.project.Packages), top)
	}
	wp := a.project.WorkPackage(id)
	if len(wp.Details) != 1 || wp.Details[0].Name != "Phase one" {
		t.Fatalf("details: %+v", wp.Details)
	}

	// bare add (no dates) defaults to the parent's span
	if _, err := a.Chat(`add detail "Phase two"`); err != nil {
		t.Fatal(err)
	}
	d := a.project.WorkPackage(id).Details[1]
	if !d.Start.Equal(wp.Start) || !d.End.Equal(wp.End) {
		t.Errorf("bare add span = %s..%s, want parent %s..%s", d.Start, d.End, wp.Start, wp.End)
	}

	// leaving the drill plane -> `add` is a top-level action again
	a.SetDrill("")
	if _, err := a.Chat(`add action "Top" from 2026-01-01 to 2026-02-01`); err != nil {
		t.Fatal(err)
	}
	if len(a.project.Packages) != top+1 {
		t.Errorf("top-level add after leaving drill did not work: %d packages", len(a.project.Packages))
	}
}

// An explicit "add detail ... to <wp>" is NOT hijacked by the drill context.
func TestExplicitAddDetailStillTargetsNamedWP(t *testing.T) {
	a := NewApp()
	a.AddWorkPackage("A", "2026-01-01", "2026-06-01")
	a.AddWorkPackage("B", "2026-01-01", "2026-06-01")
	aID, bID := a.project.Packages[0].ID, a.project.Packages[1].ID
	a.SetDrill(aID)
	if _, err := a.Chat(`add detail "x" to ` + bID + ` from 2026-02-01 to 2026-03-01`); err != nil {
		t.Fatal(err)
	}
	if len(a.project.WorkPackage(bID).Details) != 1 || len(a.project.WorkPackage(aID).Details) != 0 {
		t.Errorf("explicit add detail went to the wrong wp: A=%d B=%d",
			len(a.project.WorkPackage(aID).Details), len(a.project.WorkPackage(bID).Details))
	}
}
