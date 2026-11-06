package main

import (
	"path/filepath"
	"testing"

	"github.com/megaproaktiv/g-plan/src/app/plan"
)

// A detail (sub-action) can be marked as a Sammelvorgang (summary), just like a
// level-1 action, and it survives a save/load round-trip and the DTO.
func TestSetDetailSummary(t *testing.T) {
	a := NewApp()
	if _, err := a.AddWorkPackage("Parent", "2026-01-01", "2026-06-01"); err != nil {
		t.Fatal(err)
	}
	id := a.project.Packages[0].ID
	if _, err := a.AddDetail(id, "Phase", "2026-01-05", "2026-03-01", "0", "", ""); err != nil {
		t.Fatal(err)
	}
	det := a.project.Packages[0].Details[0].ID

	if _, err := a.SetDetailSummary(id, det, true); err != nil {
		t.Fatal(err)
	}
	if !a.project.Packages[0].Details[0].Summary {
		t.Fatal("summary not set on the model")
	}
	// DTO exposes it
	var found bool
	for _, w := range toDTO(a.project).Packages {
		for _, d := range w.Details {
			if d.ID == det {
				found = true
				if !d.Summary {
					t.Error("DTO detail.summary is false")
				}
			}
		}
	}
	if !found {
		t.Fatal("detail missing from DTO")
	}
	// toggle back off
	if _, err := a.SetDetailSummary(id, det, false); err != nil {
		t.Fatal(err)
	}
	if a.project.Packages[0].Details[0].Summary {
		t.Error("summary not cleared")
	}
	// unknown ids error
	if _, err := a.SetDetailSummary(id, "nope", true); err == nil {
		t.Error("expected error for unknown detail")
	}
	if _, err := a.SetDetailSummary("nope", det, true); err == nil {
		t.Error("expected error for unknown wp")
	}
}

// summary survives the JSON single-file round-trip.
func TestDetailSummaryRoundTrips(t *testing.T) {
	a := NewApp()
	a.AddWorkPackage("P", "2026-01-01", "2026-06-01")
	id := a.project.Packages[0].ID
	a.AddDetail(id, "Phase", "2026-01-05", "2026-03-01", "0", "", "")
	det := a.project.Packages[0].Details[0].ID
	a.SetDetailSummary(id, det, true)

	path := filepath.Join(t.TempDir(), "p.ptplan")
	if err := a.project.SaveJSON(path); err != nil {
		t.Fatal(err)
	}
	p2, err := plan.OpenJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	if !p2.Packages[0].Details[0].Summary {
		t.Error("summary lost in the JSON round-trip")
	}
}
