package plan

import (
	"path/filepath"
	"testing"

	"pgregory.net/rapid"
)

func TestNoneIsAValidContour(t *testing.T) {
	if !IsValidContour("None") || NormalizeContour("None") != ContourNone {
		t.Error("None must be a valid contour that survives normalization")
	}
	if IsValidContour("none") || IsValidContour("NONE") {
		t.Error("contour names are exact: 'none' is not 'None'")
	}
	// an unset contour is still Flat, not None
	if NormalizeContour("") != ContourFlat {
		t.Error("an empty contour stays Flat")
	}
}

// Feature: contour None - whatever numbers are stored, the action has no
// estimation, no real effort and no progress of any kind.
func TestProp_NoneHasNoEffortAndNoProgress(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		w := WorkPackage{
			ID: "a", Mode: ModeFixed, Contour: ContourNone,
			Est:     Base(rapid.Int64Range(-100, int64(MaxEffortMinutes)).Draw(t, "est")),
			RealEst: Base(rapid.Int64Range(-100, int64(MaxEffortMinutes)).Draw(t, "real")),
		}
		if w.EffectiveEst() != 0 || w.EffectiveRealEst() != 0 {
			t.Fatalf("effective effort must be 0: est=%d real=%d", w.EffectiveEst(), w.EffectiveRealEst())
		}
		if w.ProgressPercent() != 0 || w.ProgressPermille() != 0 || w.ContourPermille() != 0 || w.CompletionPosition() != 0 {
			t.Fatalf("no progress for None: %d %d %d %v", w.ProgressPercent(), w.ProgressPermille(), w.ContourPermille(), w.CompletionPosition())
		}
		// the same numbers with any other contour do count (the guard is the contour)
		w.Contour = ContourFlat
		if w.Est > 0 && w.EffectiveEst() != w.Est {
			t.Fatalf("a Flat action keeps its estimation: %d vs %d", w.EffectiveEst(), w.Est)
		}
	})
}

func TestNoneActionsAddNothingToTotalsAndCapacity(t *testing.T) {
	p := roleProject() // two actions; Dev is assigned to "a"
	p.Packages[0].Mode, p.Packages[0].Est = ModeFixed, p.BaseFromWholeDisplay(10)
	p.Packages[1].Mode, p.Packages[1].Est = ModeFixed, p.BaseFromWholeDisplay(5)
	p.Packages[0].Details = nil
	before := p.TotalEst()
	if before != p.BaseFromWholeDisplay(15) {
		t.Fatalf("setup: %d", before)
	}
	p.Packages[0].Contour = ContourNone
	if got := p.TotalEst(); got != p.BaseFromWholeDisplay(5) {
		t.Errorf("a None action must drop out of the project total: %d", got)
	}
	if got := p.RoleEst(p.RoleByName("Dev")); got != 0 {
		t.Errorf("role load of a None action is 0, got %d", got)
	}
	if _, _, over := p.Overallocated(&p.Packages[0]); over {
		t.Error("a None action cannot be over-allocated")
	}
	// the stored number is kept: switching back restores everything
	p.Packages[0].Contour = ContourFlat
	if p.TotalEst() != before {
		t.Errorf("switching back must restore the estimation: %d vs %d", p.TotalEst(), before)
	}
}

func TestNoneSurvivesSaveAndLoadWithItsHiddenValues(t *testing.T) {
	p := monitorProject()
	p.Packages[0].Contour = ContourNone
	p.Packages[0].RealEst = 123
	path := filepath.Join(t.TempDir(), "n.ptplan")
	if err := p.SaveJSON(path); err != nil {
		t.Fatal(err)
	}
	q, err := OpenJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	w := q.WorkPackage("a")
	if w.Contour != ContourNone || w.RealEst != 123 || w.Est != p.Packages[0].Est {
		t.Errorf("%+v", w)
	}
}

func TestNoneTakesNoBookings(t *testing.T) {
	p := monitorProject() // a: WP-1 (S1), c: WP-2 (S2), d untracked
	p.Packages[0].Contour = ContourNone
	p.Monitor = Monitor{Enabled: true, System: SystemAerion, ProjectID: "1"}
	if p.IsSynced(p.WorkPackage("a")) {
		t.Error("a None action is not synchronized even with an External ID")
	}
	s := p.MonitorSummary([]Booking{
		{ID: 1, Seconds: 3600, Text: "WP-1 design"}, // names the None action: nobody takes it
		{ID: 2, Seconds: 1800, Text: "WP-2 build"},
	}, "")
	if s.TotalEntries != 1 || s.UnmatchedEntries != 1 || s.UnmatchedSeconds != 3600 {
		t.Errorf("a booking naming a None action is unassigned: %+v", s)
	}
	if s.Sections[0].Tracked != 0 || len(s.Sections[0].Details) != 0 {
		t.Errorf("a None action is not tracked: %+v", s.Sections[0])
	}
	// it can't be the default either: the bookings stay unassigned
	s2 := p.MonitorSummary([]Booking{{ID: 3, Seconds: 600, Text: "x"}}, "a")
	if s2.DefaultAction != "" || s2.UnmatchedEntries != 1 {
		t.Errorf("a None action as default must be ignored: %+v", s2)
	}
	// ApplySyncedEffort leaves its real effort alone
	p.Packages[0].RealEst = 99
	p.ApplySyncedEffort(s)
	if p.Packages[0].RealEst != 99 {
		t.Errorf("sync must not touch a None action, got %d", p.Packages[0].RealEst)
	}
}

// Files written before "None cannot carry an External ID" may have one on a None
// action (it never had an effect). They are repaired when loaded.
func TestLoadingDropsExternalIDsOfNoneActions(t *testing.T) {
	p := monitorProject()
	p.Packages[0].Contour = ContourNone // a: had WP-1
	path := filepath.Join(t.TempDir(), "old.ptplan")
	if err := p.SaveJSON(path); err != nil {
		t.Fatal(err)
	}
	q, err := OpenJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := q.WorkPackage("a").ExternalID; got != "" {
		t.Errorf("a None action must not keep an External ID, got %q", got)
	}
	if q.WorkPackage("c").ExternalID != "WP-2" {
		t.Error("other actions keep their External ID")
	}
	// the freed ID can be used by another action: the None action does not own it
	if o := p.ExternalIDOwner("WP-1", "c"); o != nil {
		t.Errorf("a None action does not own an External ID: %+v", o)
	}
	if n := p.DropExternalIDsOfNone(); n != 1 || p.WorkPackage("a").ExternalID != "" {
		t.Errorf("DropExternalIDsOfNone removed %d", n)
	}
	if n := p.DropExternalIDsOfNone(); n != 0 {
		t.Error("nothing left to drop")
	}
}
