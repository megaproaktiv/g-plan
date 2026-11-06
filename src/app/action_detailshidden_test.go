package main

import (
	"path/filepath"
	"testing"

	"github.com/megaproaktiv/g-plan/src/app/plan"
)

func TestSetActionDetailsHidden(t *testing.T) {
	a := NewApp()
	a.AddWorkPackage("P", "2026-01-01", "2026-06-01")
	id := a.project.Packages[0].ID
	a.AddDetail(id, "d", "2026-01-05", "2026-02-01", "0", "", "")

	if _, err := a.SetActionDetailsHidden(id, true); err != nil {
		t.Fatal(err)
	}
	if !a.project.Packages[0].DetailsHidden {
		t.Fatal("not set")
	}
	var seen bool
	for _, w := range toDTO(a.project).Packages {
		if w.ID == id {
			seen = true
			if !w.DetailsHidden {
				t.Error("DTO detailsHidden false")
			}
		}
	}
	if !seen {
		t.Fatal("wp missing from DTO")
	}
	if _, err := a.SetActionDetailsHidden("nope", true); err == nil {
		t.Error("expected error for unknown action")
	}
	// round-trips
	path := filepath.Join(t.TempDir(), "p.ptplan")
	if err := a.project.SaveJSON(path); err != nil {
		t.Fatal(err)
	}
	p2, err := plan.OpenJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	if !p2.Packages[0].DetailsHidden {
		t.Error("detailsHidden lost in round-trip")
	}
}
