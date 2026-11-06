package main

import "testing"

// TestResolveWBSRef verifies dotted display-id resolution (n.k action, n.k.m
// detail) matches dto.go's DisplayID numbering, and that chat :after accepts it.
func TestResolveWBSRef(t *testing.T) {
	a := NewApp()
	if _, err := a.AddSection("Alpha"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.AddSection("Beta"); err != nil {
		t.Fatal(err)
	}
	w1 := mustWP(t, a, "a1", "2026-01-01", "2026-01-10")
	w2 := mustWP(t, a, "a2", "2026-01-05", "2026-01-15")
	w3 := mustWP(t, a, "b1", "2026-02-01", "2026-02-10")
	a.SetActionCluster(w1, "Alpha")
	a.SetActionCluster(w2, "Alpha")
	a.SetActionCluster(w3, "Beta")
	a.AddDetail(w1, "d1", "2026-01-01", "2026-01-03", "0", "", "")
	a.AddDetail(w1, "d2", "2026-01-03", "2026-01-05", "0", "", "")

	// 1.1 -> w1, 1.2 -> w2, 2.1 -> w3
	if got := a.resolveActionRef("1.1"); got != w1 {
		t.Errorf("1.1 => %q, want %q", got, w1)
	}
	if got := a.resolveActionRef("1.2"); got != w2 {
		t.Errorf("1.2 => %q, want %q", got, w2)
	}
	if got := a.resolveActionRef("2.1"); got != w3 {
		t.Errorf("2.1 => %q, want %q", got, w3)
	}
	// 1.1.2 -> parent w1, detail d2
	pid, did := a.resolveWBSRef("1.1.2")
	if pid != w1 {
		t.Errorf("1.1.2 parent => %q, want %q", pid, w1)
	}
	wp := a.project.WorkPackage(w1)
	if did != wp.Details[1].ID {
		t.Errorf("1.1.2 detail => %q, want %q", did, wp.Details[1].ID)
	}
}

// TestDetailLinking verifies AddDependency accepts detail ids as endpoints.
func TestDetailLinking(t *testing.T) {
	a := NewApp()
	w := mustWP(t, a, "wp", "2026-01-01", "2026-01-20")
	a.AddDetail(w, "d1", "2026-01-01", "2026-01-05", "0", "", "")
	a.AddDetail(w, "d2", "2026-01-06", "2026-01-10", "0", "", "")
	wp := a.project.WorkPackage(w)
	d1, d2 := wp.Details[0].ID, wp.Details[1].ID
	if _, err := a.AddDependency(d1, d2, "FS", "r", "l"); err != nil {
		t.Fatalf("link details: %v", err)
	}
	found := false
	for _, dep := range a.project.Deps {
		if dep.From == d1 && dep.To == d2 {
			found = true
		}
	}
	if !found {
		t.Fatal("detail dependency not stored")
	}
	// linking a non-existent task must fail
	if _, err := a.AddDependency(d1, "nope", "FS", "r", "l"); err == nil {
		t.Fatal("expected error linking to missing task")
	}
}

// TestChatAddAfterWBS verifies "add X :after 1.2" chains + links via WBS code.
func TestChatAddAfterWBS(t *testing.T) {
	a := NewApp()
	a.AddSection("Alpha")
	w1 := mustWP(t, a, "a1", "2026-01-01", "2026-01-10")
	w2 := mustWP(t, a, "a2", "2026-01-05", "2026-01-15")
	a.SetActionCluster(w1, "Alpha")
	a.SetActionCluster(w2, "Alpha")
	if _, err := a.Chat("add action running :after 1.2"); err != nil {
		t.Fatalf("add :after 1.2: %v", err)
	}
	// the new action must be FS-linked from w2 (which is 1.2)
	last := a.project.Packages[len(a.project.Packages)-1].ID
	found := false
	for _, dep := range a.project.Deps {
		if dep.From == w2 && dep.To == last {
			found = true
		}
	}
	if !found {
		t.Fatalf("no FS link from 1.2 (%s) to new action %s; deps=%v", w2, last, a.project.Deps)
	}
}
