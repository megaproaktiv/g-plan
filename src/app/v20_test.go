package main

import (
	"path/filepath"
	"testing"

	"github.com/megaproaktiv/g-plan/src/app/plan"
)

// --- Session 24: note epic (Description shown as a yellow sticky note) ---

// TestSetActionNote verifies the note toggle + position persist on the model
// and surface in the DTO, and that "auto"/empty normalizes to "up".
func TestSetActionNote(t *testing.T) {
	a := NewApp()
	w := mustWP(t, a, "task", "2026-02-01", "2026-02-20")
	// default: not visible
	if dtoWP(a.GetProject(), w).NoteVisible {
		t.Fatal("note should default to not visible")
	}
	if _, err := a.SetActionNote(w, true, "right"); err != nil {
		t.Fatal(err)
	}
	mw := a.project.WorkPackage(w)
	if !mw.NoteVisible || mw.NotePos != "right" {
		t.Fatalf("note not set: visible=%v pos=%q", mw.NoteVisible, mw.NotePos)
	}
	d := dtoWP(a.GetProject(), w)
	if !d.NoteVisible || d.NotePos != "right" {
		t.Fatalf("DTO note wrong: visible=%v pos=%q", d.NoteVisible, d.NotePos)
	}
	// empty/auto position -> default "up"
	if _, err := a.SetActionNote(w, true, "auto"); err != nil {
		t.Fatal(err)
	}
	if a.project.WorkPackage(w).NotePos != "up" {
		t.Fatalf("auto note pos should normalize to up, got %q", a.project.WorkPackage(w).NotePos)
	}
	// invalid position rejected
	if _, err := a.SetActionNote(w, true, "sideways"); err == nil {
		t.Fatal("invalid note position should be rejected")
	}
	// diagonal corner positions are accepted (ul/ur/ll/lr) so the connector
	// can run at ~45 degrees.
	for _, pos := range []string{"ul", "ur", "ll", "lr"} {
		if _, err := a.SetActionNote(w, true, pos); err != nil {
			t.Fatalf("diagonal note pos %q should be valid: %v", pos, err)
		}
		if a.project.WorkPackage(w).NotePos != pos {
			t.Fatalf("note pos %q not stored", pos)
		}
	}
}

// TestNotePersist verifies the note fields round-trip through save/open.
func TestNotePersist(t *testing.T) {
	a := NewApp()
	w := mustWP(t, a, "task", "2026-02-01", "2026-02-20")
	if _, err := a.SetActionFields(w, "task", "A short note.", "", "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetActionNote(w, true, "down"); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "n"+plan.ProjectFileExt)
	if _, err := a.SaveProject(path); err != nil {
		t.Fatal(err)
	}
	b := NewApp()
	if _, err := b.OpenProject(path); err != nil {
		t.Fatal(err)
	}
	mw := b.project.WorkPackage(w)
	if mw == nil || !mw.NoteVisible || mw.NotePos != "down" || mw.Desc != "A short note." {
		t.Fatalf("note did not round-trip: %+v", mw)
	}
}
