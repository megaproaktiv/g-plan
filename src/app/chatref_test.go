package main

import "testing"

// TestChatUseResolvesRef verifies "use #n" (and WBS) resolves to a real action
// id so relative commands act on it.
func TestChatUseResolvesRef(t *testing.T) {
	a := NewApp()
	w1 := mustWP(t, a, "aa", "2026-03-01", "2026-03-20")
	mustWP(t, a, "bb", "2026-04-01", "2026-04-20")
	if _, err := a.Chat("use #1"); err != nil {
		t.Fatalf("use #1: %v", err)
	}
	if a.current != w1 {
		t.Fatalf("current = %q, want %q", a.current, w1)
	}
	// a relative move must now succeed (acts on current)
	if _, err := a.Chat("move 3d"); err != nil {
		t.Fatalf("move 3d after use: %v", err)
	}
}

// TestChatAssignResolvesRef verifies "assign <role> to #n" stores the
// assignment against the resolved action id, not the literal "#n".
func TestChatAssignResolvesRef(t *testing.T) {
	a := NewApp()
	w1 := mustWP(t, a, "aa", "2026-03-01", "2026-03-20")
	a.AddRole("Dev", "")
	if _, err := a.Chat("assign Dev to #1 50%"); err != nil {
		t.Fatalf("assign: %v", err)
	}
	var dev *struct{}
	ok := false
	for _, r := range a.project.Roles {
		if r.Name != "Dev" {
			continue
		}
		for _, as := range r.Assignments {
			if as.WorkPackageID == w1 && as.Percent == 50 {
				ok = true
			}
			if as.WorkPackageID == "#1" {
				t.Fatalf("assignment stored literal #1 instead of %q", w1)
			}
		}
	}
	_ = dev
	if !ok {
		t.Fatalf("Dev not assigned to %q at 50%%", w1)
	}
}
