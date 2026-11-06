package plan

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// TestSaveLoadRoundTrip proves the three-file save/open preserves the graphic
// content (incl. summary + dependencies via .mmd) and the roles/config/
// description (via .roles.json), with the new fields.
func TestSaveLoadRoundTrip(t *testing.T) {
	p := NewProject("Demo", UnitDays)
	p.Description = "a demo project"
	s, _ := ParseDateStr("2026-01-01")
	e, _ := ParseDateStr("2026-03-31")
	p.SetRange(s, e)

	p.Packages = []WorkPackage{{
		ID: "demo", Name: "demo", Start: s, End: e, Mode: ModeDerived, Summary: true,
		Details: []DetailPackage{
			{ID: "demo_build", Name: "Build", Desc: "do the build", Role: "Ian McKellen",
				Start: s, End: s.AddDate(0, 0, 10), Est: p.BaseFromWholeDisplay(5)},
		},
	}, {
		ID: "ship", Name: "ship", Start: s.AddDate(0, 1, 0), End: e, Mode: ModeFixed,
		Est: p.BaseFromWholeDisplay(3),
	}}
	p.Deps = []Dependency{{From: "demo", To: "ship", Kind: "FS"}}
	p.Roles = []Role{{Name: "Ian McKellen", Description: "lead",
		Assignments: []Assignment{{WorkPackageID: "demo", Percent: 100}}}}

	dir := t.TempDir()
	base := filepath.Join(dir, "demo.mmd")
	if _, err := p.Save(base); err != nil {
		t.Fatalf("Save: %v", err)
	}
	q, _, err := Open(base)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if q.Title != "Demo" {
		t.Errorf("title = %q", q.Title)
	}
	if q.Description != "a demo project" {
		t.Errorf("description = %q (roles.json round-trip)", q.Description)
	}
	wp := q.WorkPackage("demo")
	if wp == nil {
		t.Fatal("demo WP lost")
	}
	if !wp.Summary {
		t.Errorf("summary flag lost on round-trip")
	}
	if wp.Name != "demo" {
		t.Errorf("name = %q, want plain 'demo' (no WP prefix)", wp.Name)
	}
	if len(wp.Details) != 1 || wp.Details[0].Name != "Build" {
		t.Errorf("detail lost: %+v", wp.Details)
	}
	found := false
	for _, d := range q.Deps {
		if d.From == "demo" && d.To == "ship" && d.Kind == "FS" {
			found = true
		}
	}
	if !found {
		t.Errorf("dependency lost on round-trip: %+v", q.Deps)
	}
	if len(q.Roles) != 1 || q.Roles[0].Name != "Ian McKellen" {
		t.Errorf("roles lost: %+v", q.Roles)
	}
	_ = time.Now
}

// Feature: progress, Property 3: Project save/load round-trip preserves progress data
func TestProp_ProgressRoundTrip(t *testing.T) {
	dir := t.TempDir()
	rapid.Check(t, func(t *rapid.T) {
		p := NewProject("P", UnitDays)
		s, _ := ParseDateStr("2026-01-01")
		e, _ := ParseDateStr("2026-02-01")
		p.ShowProgress = rapid.Bool().Draw(t, "showProgress")
		n := rapid.IntRange(1, 5).Draw(t, "n")
		for i := 0; i < n; i++ {
			p.Packages = append(p.Packages, WorkPackage{
				ID: "a" + string(rune('a'+i)), Name: "a", Start: s, End: e, Mode: ModeFixed,
				Est:     Base(rapid.Int64Range(0, int64(MaxEffortMinutes)).Draw(t, "est")),
				RealEst: Base(rapid.Int64Range(0, int64(MaxEffortMinutes)).Draw(t, "real")),
				Contour: rapid.SampledFrom(allContours).Draw(t, "contour"),
			})
		}
		path := filepath.Join(dir, "rt.ptplan")
		if err := p.SaveJSON(path); err != nil {
			t.Fatalf("SaveJSON: %v", err)
		}
		q, err := OpenJSON(path)
		if err != nil {
			t.Fatalf("OpenJSON: %v", err)
		}
		if q.ShowProgress != p.ShowProgress {
			t.Fatalf("ShowProgress %v != %v", q.ShowProgress, p.ShowProgress)
		}
		if len(q.Packages) != len(p.Packages) {
			t.Fatalf("package count %d != %d", len(q.Packages), len(p.Packages))
		}
		for i := range p.Packages {
			a, b := p.Packages[i], q.Packages[i]
			if a.Est != b.Est || a.RealEst != b.RealEst || a.Contour != b.Contour {
				t.Fatalf("pkg %d: %+v != %+v", i, a, b)
			}
		}
	})
}

// A file written before the progress feature (no realEst/contour/showProgress
// keys) must load with the documented defaults.
func TestProgressBackwardCompat(t *testing.T) {
	legacy := `{
  "schema": "pt-plan/project@2",
  "title": "old", "unit": "days", "hoursPerDay": 8,
  "showDetails": false, "showRoleActions": false, "showPhases": true,
  "milestones": [], "phases": [],
  "packages": [{"id":"a","name":"a","start":"2026-01-01T00:00:00Z","end":"2026-02-01T00:00:00Z","est":480,"mode":"fixed"}],
  "deps": [], "roles": []
}`
	path := filepath.Join(t.TempDir(), "old.ptplan")
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	q, err := OpenJSON(path)
	if err != nil {
		t.Fatalf("OpenJSON: %v", err)
	}
	if q.ShowProgress {
		t.Errorf("ShowProgress should default to false")
	}
	wp := q.WorkPackage("a")
	if wp == nil {
		t.Fatal("package lost")
	}
	if wp.RealEst != 0 {
		t.Errorf("RealEst = %d, want 0", wp.RealEst)
	}
	if NormalizeContour(string(wp.Contour)) != ContourFlat {
		t.Errorf("contour = %q, want Flat", wp.Contour)
	}
}

// TestRoleLabelOptionsRoundTrip proves the two "Show role actions" modifiers
// (ShowRolePercent, ShowRoleDays) survive a SaveJSON/OpenJSON round-trip.
func TestRoleLabelOptionsRoundTrip(t *testing.T) {
	p := NewProject("Roles", UnitDays)
	p.ShowRoleActions = true
	p.ShowRolePercent = true
	p.ShowRoleDays = true
	path := filepath.Join(t.TempDir(), "r.ptplan")
	if err := p.SaveJSON(path); err != nil {
		t.Fatal(err)
	}
	q, err := OpenJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	if !q.ShowRolePercent || !q.ShowRoleDays {
		t.Fatalf("role label options lost: percent=%v days=%v", q.ShowRolePercent, q.ShowRoleDays)
	}
	// and false survives too (pointer persistence, not omitempty-dropped)
	p.ShowRolePercent, p.ShowRoleDays = false, false
	if err := p.SaveJSON(path); err != nil {
		t.Fatal(err)
	}
	q2, err := OpenJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	if q2.ShowRolePercent || q2.ShowRoleDays {
		t.Fatalf("role label options should be false, got percent=%v days=%v", q2.ShowRolePercent, q2.ShowRoleDays)
	}
}

// A file predating the role-label modifiers must default them to OFF (the
// role sub-row shows the name only).
func TestRoleLabelOptionsBackwardCompat(t *testing.T) {
	legacy := `{
  "schema": "pt-plan/project@2",
  "title": "old", "unit": "days", "hoursPerDay": 8,
  "showDetails": false, "showRoleActions": true, "showPhases": true,
  "milestones": [], "phases": [], "packages": [], "deps": [], "roles": []
}`
	path := filepath.Join(t.TempDir(), "old.ptplan")
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	q, err := OpenJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	if q.ShowRolePercent || q.ShowRoleDays {
		t.Fatalf("legacy file must default role label options OFF, got percent=%v days=%v", q.ShowRolePercent, q.ShowRoleDays)
	}
}
