package main

import "testing"

// Moving an action (drag, resize, chat `move`) must be undoable with Cmd-Z.
func TestMoveWorkPackageIsUndoable(t *testing.T) {
	a := NewApp()
	if _, err := a.AddWorkPackage("demo", "2026-01-01", "2026-02-01"); err != nil {
		t.Fatal(err)
	}
	id := a.project.Packages[0].ID
	before := a.project.WorkPackage(id).Start
	if _, err := a.MoveWorkPackage(id, "2026-03-01", "2026-04-01"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Undo(); err != nil {
		t.Fatal(err)
	}
	if got := a.project.WorkPackage(id).Start; !got.Equal(before) {
		t.Errorf("start after undo = %v, want %v", got, before)
	}
}

func TestMoveSectionAndDetailAreUndoable(t *testing.T) {
	a := NewApp()
	for _, n := range []string{"A", "B"} {
		if _, err := a.AddSection(n); err != nil {
			t.Fatal(err)
		}
	}
	first := a.project.WBS().Sections[0]
	if _, err := a.MoveSection(first, 1); err != nil {
		t.Fatal(err)
	}
	if a.project.WBS().Sections[0] == first {
		t.Fatal("move had no effect")
	}
	if _, err := a.Undo(); err != nil {
		t.Fatal(err)
	}
	if got := a.project.WBS().Sections[0]; got != first {
		t.Errorf("first section after undo = %q, want %q", got, first)
	}
}
