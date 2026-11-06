package main

import (
	"testing"
	"time"

	"github.com/megaproaktiv/g-plan/src/app/plan"
)

// TestFSDependencyViolation: an FS (end-start) link is violated when the
// successor starts before the predecessor ends; the DTO flags it and a warning
// is emitted.
func TestFSDependencyViolation(t *testing.T) {
	a := NewApp()
	a1 := mustWP(t, a, "a", "2026-03-02", "2026-03-20") // Mon..Fri window
	a2 := mustWP(t, a, "b", "2026-03-10", "2026-03-30")
	if _, err := a.AddDependency(a1, a2, "FS", "r", "l"); err != nil {
		t.Fatal(err)
	}
	d := a.GetProject()
	var dep *DepDTO
	for i := range d.Deps {
		if d.Deps[i].From == a1 && d.Deps[i].To == a2 {
			dep = &d.Deps[i]
		}
	}
	if dep == nil || !dep.Violated {
		t.Fatalf("expected FS dependency to be violated (b starts 03-10 before a ends 03-20)")
	}
	// moving b to start after a ends clears the violation
	if _, err := a.MoveWorkPackage(a2, "2026-03-23", "2026-04-10"); err != nil {
		t.Fatal(err)
	}
	d = a.GetProject()
	for i := range d.Deps {
		if d.Deps[i].From == a1 && d.Deps[i].To == a2 && d.Deps[i].Violated {
			t.Fatal("violation should clear once b starts after a ends")
		}
	}
}

// TestOverallocationWarning: effort exceeding the working days in the window
// produces an over-allocation warning and sets HasWarning on the action.
func TestOverallocationWarning(t *testing.T) {
	a := NewApp()
	// 20 days effort in a 2-week (10 workday) window => over-allocated
	w := mustWP(t, a, "big", "2026-03-02", "2026-03-13") // Mon..Fri x2 = 10 workdays
	if _, err := a.SetActionFields(w, "big", "", "20", "2026-03-02", "2026-03-13"); err != nil {
		t.Fatal(err)
	}
	ed, wd, over := a.project.Overallocated(a.project.WorkPackage(w))
	if !over {
		t.Fatalf("expected over-allocation: effortDays=%d workdays=%d", ed, wd)
	}
	if wd != 10 {
		t.Errorf("expected 10 workdays, got %d", wd)
	}
	d := a.GetProject()
	for _, p := range d.Packages {
		if p.ID == w && !p.HasWarning {
			t.Error("action should have HasWarning set when over-allocated")
		}
	}
	// a matching window (20 workdays) clears it
	if _, err := a.SetActionFields(w, "big", "", "20", "2026-03-02", "2026-03-27"); err != nil {
		t.Fatal(err)
	}
	if _, _, over := a.project.Overallocated(a.project.WorkPackage(w)); over {
		t.Error("20 effort over ~20 workdays should not be over-allocated")
	}
}

// TestWorkdaysBetween excludes weekends.
func TestWorkdaysBetween(t *testing.T) {
	// 2026-03-02 is a Monday; through 2026-03-06 Friday = 5 workdays
	s := mustDate(t, "2026-03-02")
	e := mustDate(t, "2026-03-08") // Sunday
	if got := plan.WorkdaysBetween(s, e); got != 5 {
		t.Errorf("Mon..Sun should be 5 workdays, got %d", got)
	}
}

// TestSameLinePersist: SameLine flag round-trips through save/load.
func TestSameLinePersist(t *testing.T) {
	a := NewApp()
	w1 := mustWP(t, a, "a1", "2026-03-02", "2026-03-10")
	w2 := mustWP(t, a, "a2", "2026-03-05", "2026-03-12")
	a.SetActionCluster(w1, "Design")
	a.SetActionCluster(w2, "Design")
	if _, err := a.SetWorkPackageSameLine(w2, true); err != nil {
		t.Fatal(err)
	}
	d := a.GetProject()
	for _, p := range d.Packages {
		if p.ID == w2 && !p.SameLine {
			t.Fatal("w2 should be sameLine in DTO")
		}
	}
	dir := t.TempDir()
	base := dir + "/proj"
	saved, err := a.SaveProject(base)
	if err != nil {
		t.Fatal(err)
	}
	b := NewApp()
	if _, err := b.OpenProject(saved); err != nil {
		t.Fatal(err)
	}
	found := false
	for i := range b.project.Packages {
		if b.project.Packages[i].SameLine {
			found = true
		}
	}
	if !found {
		t.Fatal("SameLine flag did not persist across save/load")
	}
}

func mustDate(t *testing.T, s string) time.Time {
	t.Helper()
	tm, err := parseDate(s)
	if err != nil {
		t.Fatalf("date %q: %v", s, err)
	}
	return tm
}

// TestPDFRenderSourceHasNoWarningStyling asserts that the .mmd (the source the
// PDF/PNG export renders from) carries NO warning-red border and no warning
// text, even when the project has an over-allocated action and a violated FS
// dependency. The red warning borders must live ONLY in the live visual editor
// (the frontend SVG), never in the export.
func TestPDFRenderSourceHasNoWarningStyling(t *testing.T) {
	a := NewApp()
	// over-allocated action: 20d effort in a ~10 workday window
	w1 := mustWP(t, a, "big", "2026-03-02", "2026-03-13")
	a.SetActionFields(w1, "big", "", "20", "2026-03-02", "2026-03-13")
	// violated FS dependency
	w2 := mustWP(t, a, "succ", "2026-03-05", "2026-03-20")
	a.AddDependency(w1, w2, "FS", "r", "l")

	// sanity: the project really does have warnings now
	if len(a.project.Validate()) == 0 {
		t.Fatal("expected validation warnings in this scenario")
	}

	mmd := a.project.MMD()
	// the warning red used by the frontend border must never appear in the .mmd
	if containsFold(mmd, "e03131") || containsFold(mmd, "over-allocated") ||
		containsFold(mmd, "violated") || containsFold(mmd, "warning") {
		t.Fatalf("the .mmd render source must not contain any warning styling/text; got:\n%s", mmd)
	}
}

func containsFold(s, sub string) bool {
	return len(sub) > 0 && len(s) >= len(sub) &&
		indexFold(s, sub) >= 0
}

func indexFold(s, sub string) int {
	ls, lsub := toLower(s), toLower(sub)
	for i := 0; i+len(lsub) <= len(ls); i++ {
		if ls[i:i+len(lsub)] == lsub {
			return i
		}
	}
	return -1
}

func toLower(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 'a' - 'A'
		}
	}
	return string(b)
}
