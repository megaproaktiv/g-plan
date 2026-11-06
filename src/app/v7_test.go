package main

import (
	"testing"
)

// Tests for the behaviors added to satisfy the new test-case checks:
//   - chat `est <wbs>` resolves a WBS ref (P10.6a)
//   - SS / FF / FS-one-unit dependency validation (P8.8, P8.9, P8.10)
//   - per-line-only role 100% rule + duplicate roles (P6.4a/b/c)
//   - undo of delete link / action / section (P8b)

// --- chat est by WBS id (P10.6a) ---

func TestChatEstByWBS(t *testing.T) {
	a := NewApp()
	// one section with one action => WBS 1.1
	if _, err := a.AddSection("Build"); err != nil {
		t.Fatal(err)
	}
	w := mustWP(t, a, "compile", "2026-01-01", "2026-01-10")
	if _, err := a.Chat("est 1.1 10"); err != nil {
		t.Fatalf("est 1.1 10: %v", err)
	}
	d := a.GetProject()
	wp := dtoWP(d, w)
	if wp == nil || wp.Est != "10" {
		t.Fatalf("expected action 1.1 effort 10, got %+v", wp)
	}
	// unknown WBS id must be rejected, leaving effort unchanged
	if _, err := a.Chat("est 9.9 5"); err == nil {
		t.Fatal("expected error for unknown WBS id 9.9")
	}
	d = a.GetProject()
	if dtoWP(d, w).Est != "10" {
		t.Fatal("effort changed after a rejected est command")
	}
}

// --- SS / FF / FS dependency validation (P8.8, P8.9, P8.10) ---

func TestSSDependencyValidation(t *testing.T) {
	a := NewApp()
	// SS rule: Start(B) >= Start(A). B starting at/after A's start is OK.
	x := mustWP(t, a, "x", "2026-02-02", "2026-02-10")
	y := mustWP(t, a, "y", "2026-02-02", "2026-02-20")
	if _, err := a.AddDependency(x, y, "SS", "l", "l"); err != nil {
		t.Fatal(err)
	}
	if depViolated(a, x, y) {
		t.Fatal("SS with equal starts should NOT be violated")
	}
	// B starting AFTER A's start still satisfies SS (Start(B) >= Start(A))
	if _, err := a.MoveWorkPackage(y, "2026-02-05", "2026-02-20"); err != nil {
		t.Fatal(err)
	}
	if depViolated(a, x, y) {
		t.Fatal("SS with B starting after A should NOT be violated")
	}
	// B starting BEFORE A's start violates SS
	if _, err := a.MoveWorkPackage(y, "2026-01-30", "2026-02-20"); err != nil {
		t.Fatal(err)
	}
	if !depViolated(a, x, y) {
		t.Fatal("SS with B starting before A should be violated")
	}
}

func TestFFDependencyValidation(t *testing.T) {
	a := NewApp()
	// FF rule: End(B) >= End(A). B ending at/after A's end is OK.
	x := mustWP(t, a, "x", "2026-02-02", "2026-02-20")
	y := mustWP(t, a, "y", "2026-02-10", "2026-02-20")
	if _, err := a.AddDependency(x, y, "FF", "r", "r"); err != nil {
		t.Fatal(err)
	}
	if depViolated(a, x, y) {
		t.Fatal("FF with equal ends should NOT be violated")
	}
	// B ending AFTER A's end still satisfies FF
	if _, err := a.MoveWorkPackage(y, "2026-02-10", "2026-02-25"); err != nil {
		t.Fatal(err)
	}
	if depViolated(a, x, y) {
		t.Fatal("FF with B ending after A should NOT be violated")
	}
	// B ending BEFORE A's end violates FF
	if _, err := a.MoveWorkPackage(y, "2026-02-10", "2026-02-15"); err != nil {
		t.Fatal(err)
	}
	if !depViolated(a, x, y) {
		t.Fatal("FF with B ending before A should be violated")
	}
}

func TestFSTouchingAllowed(t *testing.T) {
	a := NewApp() // day mode by default
	x := mustWP(t, a, "x", "2026-03-02", "2026-03-06")
	// FS rule: Start(B) >= End(A). Successor starting the SAME day x ends is OK
	// (touching endpoints satisfy the >= rule).
	y := mustWP(t, a, "y", "2026-03-06", "2026-03-20")
	if _, err := a.AddDependency(x, y, "FS", "r", "l"); err != nil {
		t.Fatal(err)
	}
	if depViolated(a, x, y) {
		t.Fatal("FS with B starting exactly when A ends should NOT be violated")
	}
	// starting BEFORE x ends violates FS
	if _, err := a.MoveWorkPackage(y, "2026-03-04", "2026-03-20"); err != nil {
		t.Fatal(err)
	}
	if !depViolated(a, x, y) {
		t.Fatal("FS with B starting before A ends should be violated")
	}
}

func depViolated(a *App, from, to string) bool {
	d := a.GetProject()
	for i := range d.Deps {
		if d.Deps[i].From == from && d.Deps[i].To == to {
			return d.Deps[i].Violated
		}
	}
	return false
}

// --- per-line-only role rule: two different roles at 100% give no warning
// (P6.4b/c) ---

func TestTwoRolesAt100NoWarning(t *testing.T) {
	a := NewApp()
	// widen the project window so the action is inside it
	if _, err := a.SetProjectRange("2026-01-01", "2026-12-31"); err != nil {
		t.Fatal(err)
	}
	w := mustWP(t, a, "task", "2026-04-01", "2026-04-30")
	if _, err := a.SetActionFields(w, "task", "", "20", "2026-04-01", "2026-04-30"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetActionRoles(w, []string{"Developer", "Architect"}, []int{100, 100}); err != nil {
		t.Fatal(err)
	}
	d := a.GetProject()
	wp := dtoWP(d, w)
	if wp.HasWarning {
		t.Fatalf("two different roles each at 100%% must not warn (per-line rule); warnings=%v", d.Warnings)
	}
	for _, msg := range d.Warnings {
		if containsPct100(msg) {
			t.Fatalf("unexpected per-sum role warning: %q", msg)
		}
	}
}

func containsPct100(s string) bool {
	return len(s) > 0 && (indexOf(s, "total 200%") >= 0 || indexOf(s, "expected 100%") >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// --- undo of delete link / action / section (P8b) ---

func TestUndoDeleteDependency(t *testing.T) {
	a := NewApp()
	x := mustWP(t, a, "x", "2026-05-01", "2026-05-10")
	y := mustWP(t, a, "y", "2026-05-12", "2026-05-20")
	if _, err := a.AddDependency(x, y, "FS", "r", "l"); err != nil {
		t.Fatal(err)
	}
	a.RemoveDependency(x, y)
	if len(a.GetProject().Deps) != 0 {
		t.Fatal("dependency should be gone after remove")
	}
	if _, err := a.Undo(); err != nil {
		t.Fatal(err)
	}
	if len(a.GetProject().Deps) != 1 {
		t.Fatal("undo should restore the deleted dependency")
	}
}

func TestUndoDeleteAction(t *testing.T) {
	a := NewApp()
	x := mustWP(t, a, "x", "2026-05-01", "2026-05-10")
	before := len(a.GetProject().Packages)
	a.RemoveWorkPackage(x)
	if len(a.GetProject().Packages) != before-1 {
		t.Fatal("action should be removed")
	}
	if _, err := a.Undo(); err != nil {
		t.Fatal(err)
	}
	if len(a.GetProject().Packages) != before {
		t.Fatal("undo should restore the deleted action")
	}
	if dtoWP(a.GetProject(), x) == nil {
		t.Fatal("restored action should have its original id")
	}
}

func TestUndoDeleteSection(t *testing.T) {
	a := NewApp()
	if _, err := a.AddSection("Design"); err != nil {
		t.Fatal(err)
	}
	// put an action into the section
	if _, err := a.AddWorkPackage("ui", "2026-06-01", "2026-06-10"); err != nil {
		t.Fatal(err)
	}
	id := a.project.Packages[len(a.project.Packages)-1].ID
	if a.project.WorkPackage(id).Cluster != "Design" {
		t.Fatalf("new action should land in current section Design, got %q", a.project.WorkPackage(id).Cluster)
	}
	if _, err := a.RemoveSection("Design"); err != nil {
		t.Fatal(err)
	}
	if _, ok := a.project.SectionColors["Design"]; ok {
		t.Fatal("section color entry should be gone after remove")
	}
	if a.project.WorkPackage(id).Cluster != "" {
		t.Fatal("action should move to the default lane after its section is removed")
	}
	if _, err := a.Undo(); err != nil {
		t.Fatal(err)
	}
	if a.project.WorkPackage(id).Cluster != "Design" {
		t.Fatal("undo should restore the action's section membership")
	}
}
