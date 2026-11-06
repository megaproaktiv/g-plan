package main

import (
	"path/filepath"
	"testing"

	"github.com/megaproaktiv/g-plan/src/app/plan"
)

// A new section goes to the END: with WP-1 there, "add section" gives WP-2 -
// also when empty sections exist whose names sort before the new one.
func TestAddSectionAppendsAtTheEnd(t *testing.T) {
	a, _ := newAppWithAction(t)
	first := a.project.SectionNames()
	if len(first) == 0 {
		t.Fatal("expected a section")
	}
	for _, name := range []string{"Zeta", "Alpha", "Section 10", "Section 2"} {
		d, err := a.AddSection(name)
		if err != nil {
			t.Fatal(err)
		}
		if got := d.SectionNums[name]; got != len(a.project.SectionNames()) {
			t.Fatalf("%q got WP-%d, want the last number %d (order %v)", name, got, len(a.project.SectionNames()), a.project.SectionNames())
		}
	}
	names := a.project.SectionNames()
	want := append(append([]string{}, first...), "Zeta", "Alpha", "Section 10", "Section 2")
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("order = %v, want %v", names, want)
		}
	}
	// undo removes the last one again
	a.Undo()
	if got := a.project.SectionNames(); len(got) != len(want)-1 {
		t.Fatalf("after undo: %v", got)
	}
}

func TestShowWeekNumbersPersists(t *testing.T) {
	a := NewApp()
	if a.GetProject().ShowWeekNumbers {
		t.Fatal("week numbers must be off by default")
	}
	if !a.SetShowWeekNumbers(true).ShowWeekNumbers {
		t.Fatal("not applied")
	}
	path := filepath.Join(t.TempDir(), "w"+plan.ProjectFileExt)
	if _, err := a.SaveProject(path); err != nil {
		t.Fatal(err)
	}
	b := NewApp()
	if _, err := b.OpenProject(path); err != nil {
		t.Fatal(err)
	}
	if !b.project.ShowWeekNumbers {
		t.Fatal("week numbers did not round-trip")
	}
}
