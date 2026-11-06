package main

import "testing"

// The first action named "demo" must get the id "demo" (not "demo0"/"demo2");
// a second "demo" gets "demo2".
func TestAddActionIDs(t *testing.T) {
	a := NewApp()
	if _, err := a.AddWorkPackage("demo", "2026-01-01", "2026-02-01"); err != nil {
		t.Fatal(err)
	}
	if got := a.project.Packages[0].ID; got != "demo" {
		t.Errorf("first demo id = %q, want \"demo\"", got)
	}
	if got := a.project.Packages[0].Name; got != "demo" {
		t.Errorf("name = %q, want \"demo\"", got)
	}
	if _, err := a.AddWorkPackage("demo", "2026-01-01", "2026-02-01"); err != nil {
		t.Fatal(err)
	}
	if got := a.project.Packages[1].ID; got != "demo2" {
		t.Errorf("second demo id = %q, want \"demo2\"", got)
	}
}
