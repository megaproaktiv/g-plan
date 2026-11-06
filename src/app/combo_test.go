package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/megaproaktiv/g-plan/src/app/plan"
)

// These tests drive the App binding layer (the exact surface the UI calls) and
// exercise feature COMBINATIONS end to end, per the "test until every
// combination works" directive. Each test builds a scenario through bindings,
// then asserts the resulting ProjectDTO is internally consistent.

func mustWP(t *testing.T, a *App, name, start, end string) string {
	t.Helper()
	if _, err := a.AddWorkPackage(name, start, end); err != nil {
		t.Fatalf("AddWorkPackage(%q): %v", name, err)
	}
	return a.project.Packages[len(a.project.Packages)-1].ID
}

// find a DTO package by internal id
func dtoWP(d *ProjectDTO, id string) *WorkPackDTO {
	for i := range d.Packages {
		if d.Packages[i].ID == id {
			return &d.Packages[i]
		}
	}
	return nil
}

// --- 1. WBS numbering across sections / actions / details ---

func TestComboWBSNumbering(t *testing.T) {
	a := NewApp()
	// two sections
	if _, err := a.AddSection("Alpha"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.AddSection("Beta"); err != nil {
		t.Fatal(err)
	}
	// put actions into each via cluster
	w1 := mustWP(t, a, "a1", "2026-01-01", "2026-01-10")
	w2 := mustWP(t, a, "a2", "2026-01-05", "2026-01-15")
	w3 := mustWP(t, a, "b1", "2026-02-01", "2026-02-10")
	if _, err := a.SetActionCluster(w1, "Alpha"); err != nil {
		t.Fatal(err)
	}
	a.SetActionCluster(w2, "Alpha")
	a.SetActionCluster(w3, "Beta")
	// details on w1 -> n.k.m
	a.AddDetail(w1, "d1", "2026-01-01", "2026-01-03", "0", "", "")
	a.AddDetail(w1, "d2", "2026-01-03", "2026-01-05", "0", "", "")

	d := a.GetProject()
	p1 := dtoWP(d, w1)
	p2 := dtoWP(d, w2)
	p3 := dtoWP(d, w3)
	// Alpha is WP after the two empty declared sections? Order: Alpha, Beta
	// declared first (AddSection), so Alpha=1, Beta=2.
	if p1.DisplayID != "1.1" {
		t.Errorf("w1 WBS = %q, want 1.1", p1.DisplayID)
	}
	if p2.DisplayID != "1.2" {
		t.Errorf("w2 WBS = %q, want 1.2", p2.DisplayID)
	}
	if p3.DisplayID != "2.1" {
		t.Errorf("w3 WBS = %q, want 2.1", p3.DisplayID)
	}
	if len(p1.Details) != 2 || p1.Details[0].DisplayID != "1.1.1" || p1.Details[1].DisplayID != "1.1.2" {
		t.Errorf("detail WBS wrong: %+v", p1.Details)
	}
}

// WBS cascades when a section is reordered.
func TestComboWBSCascadesOnReorder(t *testing.T) {
	a := NewApp()
	// two sections each with an action, so section order is well-defined by
	// package order: First (has w0) = WP-1, Second (has w) = WP-2.
	a.AddSection("First")
	a.AddSection("Second")
	w0 := mustWP(t, a, "f", "2026-01-01", "2026-01-05")
	w := mustWP(t, a, "x", "2026-01-06", "2026-01-10")
	a.SetActionCluster(w0, "First")
	a.SetActionCluster(w, "Second")
	d := a.GetProject()
	if dtoWP(d, w).DisplayID != "2.1" {
		t.Fatalf("pre-reorder WBS = %q, want 2.1", dtoWP(d, w).DisplayID)
	}
	// move Second up to position 1
	if _, err := a.MoveSection("Second", -1); err != nil {
		t.Fatal(err)
	}
	d = a.GetProject()
	if dtoWP(d, w).DisplayID != "1.1" {
		t.Errorf("post-reorder WBS = %q, want 1.1 (cascaded)", dtoWP(d, w).DisplayID)
	}
	if dtoWP(d, w0).DisplayID != "2.1" {
		t.Errorf("First should now be WP-2: %q", dtoWP(d, w0).DisplayID)
	}
}

// --- 2. Roles + estimation + total, combined ---

func TestComboRolesEstimationTotal(t *testing.T) {
	a := NewApp()
	w1 := mustWP(t, a, "A", "2026-01-01", "2026-01-31")
	w2 := mustWP(t, a, "B", "2026-02-01", "2026-02-28")
	// set estimations (detail-less -> fixed)
	if _, err := a.SetActionFields(w1, "", "", "10", "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetActionFields(w2, "", "", "5", "", ""); err != nil {
		t.Fatal(err)
	}
	d := a.GetProject()
	if d.TotalEst != "15" {
		t.Errorf("total = %q, want 15", d.TotalEst)
	}
	// up to 3 roles on w1
	if _, err := a.SetActionRoles(w1, []string{"Dev", "QA"}, []int{60, 40}); err != nil {
		t.Fatal(err)
	}
	d = a.GetProject()
	// Dev load = 60% of 10 = 6 days; QA = 4 days
	var dev, qa *RoleDTO
	for i := range d.Roles {
		if d.Roles[i].Name == "Dev" {
			dev = &d.Roles[i]
		}
		if d.Roles[i].Name == "QA" {
			qa = &d.Roles[i]
		}
	}
	if dev == nil || qa == nil {
		t.Fatalf("roles not created: %+v", d.Roles)
	}
	if dev.TotalEst != "6" {
		t.Errorf("Dev load = %q, want 6", dev.TotalEst)
	}
	if qa.TotalEst != "4" {
		t.Errorf("QA load = %q, want 4", qa.TotalEst)
	}
	// More than 3 role lines are now allowed (R-6.9: Add Role / × remove), and
	// the same role may appear more than once. A single assignment above 100%
	// is clamped to 100% (R-6.7); percents are NOT summed across lines (R-6.8).
	if _, err := a.SetActionRoles(w1, []string{"a", "b", "c", "d"}, []int{25, 25, 25, 25}); err != nil {
		t.Errorf("4 roles should be accepted now: %v", err)
	}
	// a single assignment over 100% is clamped
	if _, err := a.SetActionRoles(w1, []string{"Dev"}, []int{150}); err != nil {
		t.Fatal(err)
	}
	d = a.GetProject()
	for i := range d.Roles {
		if d.Roles[i].Name == "Dev" {
			for _, as := range d.Roles[i].Assignments {
				if as.WorkPackageID == w1 && as.Percent > 100 {
					t.Errorf("single assignment should be capped at 100%%, got %d", as.Percent)
				}
			}
		}
	}
	// the same role assigned twice to one action is allowed (two Developers)
	if _, err := a.SetActionRoles(w1, []string{"Dev", "Dev"}, []int{100, 100}); err != nil {
		t.Errorf("duplicate role on one action should be allowed: %v", err)
	}
	d = a.GetProject()
	devCount := 0
	for i := range d.Roles {
		if d.Roles[i].Name == "Dev" {
			for _, as := range d.Roles[i].Assignments {
				if as.WorkPackageID == w1 {
					devCount++
				}
			}
		}
	}
	if devCount != 2 {
		t.Errorf("expected 2 Dev assignments on the action, got %d", devCount)
	}
}

// --- 3. Derived vs fixed estimation with details ---

func TestComboDerivedRollup(t *testing.T) {
	a := NewApp()
	w := mustWP(t, a, "A", "2026-01-01", "2026-01-31") // starts derived
	a.AddDetail(w, "d1", "2026-01-01", "2026-01-10", "3", "", "")
	a.AddDetail(w, "d2", "2026-01-10", "2026-01-20", "2", "", "")
	d := a.GetProject()
	wp := dtoWP(d, w)
	if wp.Mode != "derived" {
		t.Errorf("mode = %q, want derived", wp.Mode)
	}
	if wp.Est != "5" {
		t.Errorf("derived est = %q, want 5 (3+2)", wp.Est)
	}
	if d.TotalEst != "5" {
		t.Errorf("total = %q, want 5", d.TotalEst)
	}
}

// --- 4. Dependencies + reorder keep links valid ---

func TestComboLinksSurviveReorder(t *testing.T) {
	a := NewApp()
	a.AddSection("S1")
	a.AddSection("S2")
	w1 := mustWP(t, a, "A", "2026-01-01", "2026-01-10")
	w2 := mustWP(t, a, "B", "2026-01-11", "2026-01-20")
	a.SetActionCluster(w1, "S1")
	a.SetActionCluster(w2, "S2")
	if _, err := a.AddDependency(w1, w2, "FS", "r", "l"); err != nil {
		t.Fatal(err)
	}
	a.MoveSection("S2", -1) // reorder
	d := a.GetProject()
	found := false
	for _, dep := range d.Deps {
		if dep.From == w1 && dep.To == w2 && dep.Kind == "FS" {
			found = true
		}
	}
	if !found {
		t.Errorf("dependency lost after reorder: %+v", d.Deps)
	}
}

// --- 5. Unit switch keeps absolute effort (the P1.5 bug class) ---

func TestComboUnitSwitchStable(t *testing.T) {
	a := NewApp()
	a.SetConfig("hours", 8)
	w := mustWP(t, a, "A", "2026-01-01", "2026-01-31")
	if _, err := a.SetActionFields(w, "", "", "12", "", ""); err != nil {
		t.Fatal(err)
	}
	// 12 hours -> switch to days (8h/day) -> 1,5 days
	a.SetConfig("days", 8)
	d := a.GetProject()
	if got := dtoWP(d, w).Est; got != "1,5" {
		t.Errorf("12h in day mode = %q, want 1,5", got)
	}
	// and back to hours -> 12
	a.SetConfig("hours", 8)
	d = a.GetProject()
	if got := dtoWP(d, w).Est; got != "12" {
		t.Errorf("round-trip to hours = %q, want 12", got)
	}
}

// --- 6. Baseline capture then edits then compare ---

func TestComboBaselineWorkflow(t *testing.T) {
	a := NewApp()
	w := mustWP(t, a, "A", "2026-01-01", "2026-01-10")
	a.SetActionFields(w, "", "", "5", "", "")
	if _, err := a.CaptureBaseline("v1"); err != nil {
		t.Fatal(err)
	}
	// slip + grow
	a.SetActionFields(w, "", "", "8", "2026-01-01", "2026-01-15")
	v, err := a.CompareBaseline("v1")
	if err != nil {
		t.Fatal(err)
	}
	var row *VarianceDTO
	for i := range v {
		if v[i].Kind == "wp" {
			row = &v[i]
		}
	}
	if row == nil {
		t.Fatal("no wp variance row")
	}
	if row.EndDays != 5 {
		t.Errorf("endDays = %d, want 5", row.EndDays)
	}
	if row.EstDelta != "3" {
		t.Errorf("estDelta = %q, want 3", row.EstDelta)
	}
}

// --- 7. Full kitchen-sink save/load round-trip ---

func TestComboFullRoundTrip(t *testing.T) {
	a := NewApp()
	a.SetProjectDetails("Kitchen", "desc", "2026-01-01", "2026-12-31")
	a.SetArrowStyle("square")
	a.AddSection("Design")
	w1 := mustWP(t, a, "Spec", "2026-01-01", "2026-01-31")
	w2 := mustWP(t, a, "Build", "2026-02-01", "2026-03-31")
	a.SetActionCluster(w1, "Design")
	a.SetActionCluster(w2, "Design")
	a.SetActionColor(w1, "#ff0000")
	a.SetWorkPackageSummary(w2, true)
	a.AddDetail(w1, "research", "2026-01-01", "2026-01-10", "2", "notes", "")
	a.SetActionRoles(w1, []string{"Dev"}, []int{100})
	a.AddDependency(w1, w2, "FS", "r", "l")
	a.CaptureBaseline("v1")

	dir := t.TempDir()
	base := filepath.Join(dir, "k")
	savedPath, err := a.SaveProject(base)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	// the single self-contained JSON file exists
	if _, err := os.Stat(savedPath); err != nil {
		t.Fatalf("missing project json %s: %v", savedPath, err)
	}
	if !strings.HasSuffix(savedPath, plan.ProjectFileExt) {
		t.Fatalf("expected a %s file, got %s", plan.ProjectFileExt, savedPath)
	}
	// reopen into a fresh app
	b := NewApp()
	d, err := b.OpenProject(savedPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if d.Title != "Kitchen" || d.Description != "desc" {
		t.Errorf("project meta lost: %q / %q", d.Title, d.Description)
	}
	if d.ArrowStyle != "square" {
		t.Errorf("arrow style lost: %q", d.ArrowStyle)
	}
	if len(d.Packages) != 2 {
		t.Fatalf("packages = %d, want 2", len(d.Packages))
	}
	sp := dtoWP(d, "spec")
	if sp == nil || sp.Color != "#ff0000" {
		t.Errorf("action color lost: %+v", sp)
	}
	if len(sp.Details) != 1 || sp.Details[0].Name != "research" {
		t.Errorf("detail lost: %+v", sp.Details)
	}
	bld := dtoWP(d, "build")
	if bld == nil || !bld.Summary {
		t.Errorf("summary flag lost")
	}
	if len(d.Deps) != 1 {
		t.Errorf("dependency lost: %+v", d.Deps)
	}
	if bl := b.ListBaselines(); len(bl) != 1 {
		t.Errorf("baseline lost: %+v", bl)
	}
	// WBS still correct after reload
	if sp.DisplayID != "1.1" || bld.DisplayID != "1.2" {
		t.Errorf("WBS after reload: %q / %q", sp.DisplayID, bld.DisplayID)
	}
}

// --- 8. Chat commands combine with the model ---

func TestComboChatFlow(t *testing.T) {
	a := NewApp()
	if _, err := a.Chat("add action Build"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Chat("add action Test :after #1"); err != nil {
		t.Fatalf("add after: %v", err)
	}
	d := a.GetProject()
	if len(d.Packages) != 2 {
		t.Fatalf("chat add: packages = %d, want 2", len(d.Packages))
	}
	// the :after must have created an FS link #1 -> #2
	if len(d.Deps) != 1 || d.Deps[0].Kind != "FS" {
		t.Errorf("add-after link missing: %+v", d.Deps)
	}
	// link by number
	a.Chat("add action Third")
	if _, err := a.Chat("link #1 #3 SS"); err != nil {
		t.Fatalf("link by number: %v", err)
	}
	d = a.GetProject()
	hasSS := false
	for _, dep := range d.Deps {
		if dep.Kind == "SS" {
			hasSS = true
		}
	}
	if !hasSS {
		t.Errorf("SS link not created: %+v", d.Deps)
	}
}

// --- 9. Detail clamp + drill numbering together ---

func TestComboDrillClampAndNumber(t *testing.T) {
	a := NewApp()
	w := mustWP(t, a, "A", "2026-01-10", "2026-01-20")
	a.AddDetail(w, "d1", "2026-01-10", "2026-01-12", "0", "", "")
	det := a.project.Packages[0].Details[0].ID
	// move detail outside parent -> clamp
	a.MoveDetail(w, det, "2026-01-01", "2026-02-01")
	d := a.GetProject()
	dp := dtoWP(d, w).Details[0]
	if dp.Start != "2026-01-10" || dp.End != "2026-01-20" {
		t.Errorf("detail not clamped: %s..%s", dp.Start, dp.End)
	}
	if dp.DisplayID != "1.1.1" {
		t.Errorf("detail WBS = %q, want 1.1.1", dp.DisplayID)
	}
}

// --- 10. Markdown projection reflects combined state ---

func TestComboMarkdownProjection(t *testing.T) {
	a := NewApp()
	w := mustWP(t, a, "Server", "2026-01-01", "2026-01-31")
	a.SetActionFields(w, "", "", "10", "", "")
	a.SetActionRoles(w, []string{"Dev"}, []int{100})
	md := a.Markdown()
	if !strings.Contains(md, "Server") {
		t.Errorf("markdown missing action name")
	}
	if !strings.Contains(md, "Dev") {
		t.Errorf("markdown missing role")
	}
}
