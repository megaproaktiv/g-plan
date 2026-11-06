package main

import "testing"

// Setting an estimation on a detail-less action must switch it to fixed mode
// so the value is stored and flows into the project total.
func TestSetEstimationFlowsToTotal(t *testing.T) {
	a := NewApp()
	if _, err := a.AddWorkPackage("demo", "2026-01-01", "2026-02-01"); err != nil {
		t.Fatal(err)
	}
	id := a.project.Packages[0].ID
	if _, err := a.SetActionFields(id, "", "", "5", "", ""); err != nil {
		t.Fatalf("SetActionFields: %v", err)
	}
	wp := a.project.WorkPackage(id)
	if wp.Mode != "fixed" {
		t.Errorf("mode = %q, want fixed after direct estimation", wp.Mode)
	}
	// 5 days -> project total should be 5 days (320 h stored at 8h/day = 5*64? no: 5*8*... )
	if got := a.project.Format(a.project.TotalEst()); got != "5" {
		t.Errorf("project total = %q, want 5", got)
	}
}

// `est 1 20` on an action that has details (derived mode) must take effect:
// the typed level-1 value becomes authoritative.
func TestChatEstOnActionWithDetails(t *testing.T) {
	a := NewApp()
	if _, err := a.AddWorkPackage("demo", "2026-01-01", "2026-03-01"); err != nil {
		t.Fatal(err)
	}
	id := a.project.Packages[0].ID
	if _, err := a.AddDetail(id, "d1", "2026-01-01", "2026-01-31", "3", "", ""); err != nil {
		t.Fatal(err)
	}
	a.project.WorkPackage(id).Mode = "derived"
	if _, err := a.Chat("est 1 20"); err != nil {
		t.Fatal(err)
	}
	if got := a.project.Format(a.project.TotalEst()); got != "20" {
		t.Errorf("project total = %q, want 20", got)
	}
}
