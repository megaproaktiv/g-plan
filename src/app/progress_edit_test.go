package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/megaproaktiv/g-plan/src/app/plan"

	"pgregory.net/rapid"
)

func newAppWithAction(t testing.TB) (*App, string) {
	a := NewApp()
	if _, err := a.AddWorkPackage("demo", "2026-01-01", "2026-02-01"); err != nil {
		t.Fatal(err)
	}
	return a, a.project.Packages[0].ID
}

// New actions default to Flat contour and zero real effort (R4.4, R1.7).
func TestNewActionProgressDefaults(t *testing.T) {
	a, id := newAppWithAction(t)
	wp := a.project.WorkPackage(id)
	if wp.Contour != plan.ContourFlat {
		t.Errorf("Contour = %q, want Flat", wp.Contour)
	}
	if wp.RealEst != 0 {
		t.Errorf("RealEst = %d, want 0", wp.RealEst)
	}
}

// Feature: progress, Property 1: Accepting a valid effort stores exactly that value
func TestProp_ValidEffortStored(t *testing.T) {
	a, id := newAppWithAction(t)
	// Hour mode: display unit = hour, so 1 stored minute == 1/60 display unit.
	a.project.Unit = plan.UnitHours
	rapid.Check(t, func(t *rapid.T) {
		// whole hours so the estimated setter's whole-unit rule also accepts it
		hours := rapid.Int64Range(0, int64(plan.MaxEffortMinutes)/60).Draw(t, "hours")
		want := plan.Base(hours * 60)
		if _, err := a.SetWorkPackageEstimation(id, fmt.Sprint(hours)); err != nil {
			t.Fatalf("estimation %d rejected: %v", hours, err)
		}
		if got := a.project.WorkPackage(id).Est; got != want {
			t.Fatalf("Est = %d, want %d", got, want)
		}
		if _, err := a.SetWorkPackageRealEffort(id, fmt.Sprint(hours)); err != nil {
			t.Fatalf("real %d rejected: %v", hours, err)
		}
		if got := a.project.WorkPackage(id).RealEst; got != want {
			t.Fatalf("RealEst = %d, want %d", got, want)
		}
	})
}

// Feature: progress, Property 2: Invalid efforts are rejected and leave the stored value unchanged
func TestProp_InvalidEffortRejected(t *testing.T) {
	a, id := newAppWithAction(t)
	a.project.Unit = plan.UnitHours
	wp := a.project.WorkPackage(id)
	wp.Est, wp.RealEst = 120, 60
	rapid.Check(t, func(t *rapid.T) {
		overMax := fmt.Sprint(int64(plan.MaxEffortMinutes)/60 + 1 + rapid.Int64Range(0, 1_000_000).Draw(t, "over"))
		neg := fmt.Sprintf("-%d", rapid.Int64Range(1, 1_000_000).Draw(t, "neg"))
		bad := rapid.SampledFrom([]string{"", "   ", "abc", "1x", "1.2.3", overMax, neg}).Draw(t, "bad")
		if _, err := a.SetWorkPackageRealEffort(id, bad); err == nil {
			t.Fatalf("real %q accepted", bad)
		}
		if wp.RealEst != 60 {
			t.Fatalf("RealEst changed to %d on rejected %q", wp.RealEst, bad)
		}
		if strings.TrimSpace(bad) != "" { // empty estimation parses as 0 (existing behaviour)
			if _, err := a.SetWorkPackageEstimation(id, bad); err == nil {
				t.Fatalf("estimation %q accepted", bad)
			}
			if wp.Est != 120 {
				t.Fatalf("Est changed to %d on rejected %q", wp.Est, bad)
			}
		}
	})
}

// Feature: progress, Property 12: An invalid contour write is rejected and leaves the contour unchanged
func TestProp_InvalidContourRejected(t *testing.T) {
	a, id := newAppWithAction(t)
	wp := a.project.WorkPackage(id)
	wp.Contour = plan.ContourBack
	rapid.Check(t, func(t *rapid.T) {
		s := rapid.String().Draw(t, "s")
		if plan.IsValidContour(s) {
			t.Skip()
		}
		if _, err := a.SetWorkPackageContour(id, s); err == nil {
			t.Fatalf("contour %q accepted", s)
		}
		if wp.Contour != plan.ContourBack {
			t.Fatalf("contour changed to %q", wp.Contour)
		}
	})
}

func TestSetContourValid(t *testing.T) {
	a, id := newAppWithAction(t)
	for _, c := range []plan.Contour{plan.ContourFront, plan.ContourBack, plan.ContourDouble, plan.ContourFlat} {
		dto, err := a.SetWorkPackageContour(id, string(c))
		if err != nil {
			t.Fatalf("%s: %v", c, err)
		}
		if dto.Packages[0].Contour != string(c) {
			t.Errorf("dto contour = %q, want %q", dto.Packages[0].Contour, c)
		}
	}
}

func TestShowProgressViewOptionAndDTO(t *testing.T) {
	a, id := newAppWithAction(t)
	if toDTO(a.project).ShowProgress {
		t.Fatal("ShowProgress must default off")
	}
	if !a.SetViewOptions(false, false, true, true).ShowProgress {
		t.Fatal("ShowProgress not set")
	}
	a.project.Unit = plan.UnitHours
	if _, err := a.SetWorkPackageEstimation(id, "10"); err != nil {
		t.Fatal(err)
	}
	dto, err := a.SetWorkPackageRealEffort(id, "5")
	if err != nil {
		t.Fatal(err)
	}
	p := dto.Packages[0]
	if p.ProgressPercent != 50 || p.CompletionPos != 0.5 || p.RealEst != "5" {
		t.Errorf("dto progress = %d%% pos=%v real=%q", p.ProgressPercent, p.CompletionPos, p.RealEst)
	}
}

// TestSetRoleLabelOptions verifies the two "Show role actions" modifiers
// default off and are set + surfaced in the DTO.
func TestSetRoleLabelOptions(t *testing.T) {
	a, _ := newAppWithAction(t)
	d := toDTO(a.project)
	if d.ShowRolePercent || d.ShowRoleDays {
		t.Fatal("role label options must default off (names only)")
	}
	d = a.SetRoleLabelOptions(true, false)
	if !d.ShowRolePercent || d.ShowRoleDays {
		t.Fatalf("expected percent on, days off; got percent=%v days=%v", d.ShowRolePercent, d.ShowRoleDays)
	}
	d = a.SetRoleLabelOptions(true, true)
	if !d.ShowRolePercent || !d.ShowRoleDays {
		t.Fatalf("expected both on; got percent=%v days=%v", d.ShowRolePercent, d.ShowRoleDays)
	}
}
