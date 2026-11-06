package main

import (
	"testing"

	"github.com/megaproaktiv/g-plan/src/app/plan"
)

// --- Session 23: milestone create-at-section, move, phases non-linkable,
// action->milestone before/after rules ---

// msID returns the id of the last-added milestone.
func lastMilestone(a *App) string {
	return a.project.Milestones[len(a.project.Milestones)-1].ID
}

// TestAddMilestoneInSectionAtDate verifies a milestone is created in the given
// section at the given date and is selected (immediately visible).
func TestAddMilestoneInSectionAtDate(t *testing.T) {
	a := NewApp()
	if _, err := a.SetProjectRange("2026-01-01", "2026-12-31"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.AddSection("Work"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.AddMilestone("Gate", "2026-05-20", "Work"); err != nil {
		t.Fatal(err)
	}
	id := lastMilestone(a)
	m := a.project.MilestoneByID(id)
	if m == nil || m.Cluster != "Work" {
		t.Fatalf("milestone should be in section Work, got %+v", m)
	}
	if m.Date.Format(plan.DateLayout) != "2026-05-20" {
		t.Fatalf("milestone date should be 2026-05-20, got %s", m.Date.Format(plan.DateLayout))
	}
	// immediately selected
	if a.Selection().Kind != "milestone" || a.Selection().ID != id {
		t.Fatalf("new milestone should be selected, got %+v", a.Selection())
	}
}

// TestMoveMilestone verifies the drag-by-mouse date change (MoveMilestone),
// including moving the project Start/End markers.
func TestMoveMilestone(t *testing.T) {
	a := NewApp()
	if _, err := a.SetProjectRange("2026-01-01", "2026-12-31"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.AddSection("Work"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.AddMilestone("Gate", "2026-05-20", "Work"); err != nil {
		t.Fatal(err)
	}
	id := lastMilestone(a)
	if _, err := a.MoveMilestone(id, "2026-06-01"); err != nil {
		t.Fatal(err)
	}
	if got := a.project.MilestoneByID(id).Date.Format(plan.DateLayout); got != "2026-06-01" {
		t.Fatalf("milestone not moved, got %s", got)
	}
	// moving the project Start marker moves the window bound
	if _, err := a.MoveMilestone(plan.MilestoneStartID, "2026-02-01"); err != nil {
		t.Fatal(err)
	}
	if s, ok := a.project.Start(); !ok || s.Format(plan.DateLayout) != "2026-02-01" {
		t.Fatalf("project start not moved, got %v ok=%v", s, ok)
	}
	// move is undoable
	if _, err := a.Undo(); err != nil {
		t.Fatal(err)
	}
	if s, _ := a.project.Start(); s.Format(plan.DateLayout) != "2026-01-01" {
		t.Fatal("undo should restore the project start")
	}
}

// TestPhasesMilestoneNotLinkable verifies milestones in the Phases row (and the
// project Start/End markers and unclustered milestones) cannot be link
// endpoints.
func TestPhasesMilestoneNotLinkable(t *testing.T) {
	a := NewApp()
	if _, err := a.SetProjectRange("2026-01-01", "2026-12-31"); err != nil {
		t.Fatal(err)
	}
	act := mustWP(t, a, "spec", "2026-02-01", "2026-02-10")

	// Phases-row milestone
	if _, err := a.AddMilestone("Kickoff", "2026-02-15", "Phases"); err != nil {
		t.Fatal(err)
	}
	ph := lastMilestone(a)
	if _, err := a.AddDependency(act, ph, "FS", "r", "l"); err == nil {
		t.Fatal("a Phases-row milestone must not be linkable")
	}
	// unclustered milestone (full-height)
	if _, err := a.AddMilestone("Floating", "2026-02-18", ""); err != nil {
		t.Fatal(err)
	}
	fl := lastMilestone(a)
	if _, err := a.AddDependency(act, fl, "FS", "r", "l"); err == nil {
		t.Fatal("an unclustered (full-height) milestone must not be linkable")
	}
	// project Start marker
	if _, err := a.AddDependency(act, plan.MilestoneStartID, "FS", "r", "l"); err == nil {
		t.Fatal("the project Start marker must not be linkable")
	}
}

// TestActionToMilestoneRules verifies action-edge -> milestone links follow the
// normal before/after (FS/SF) rules, not a forced kind. A milestone is a
// zero-duration point so Start=End=Date.
func TestActionToMilestoneRules(t *testing.T) {
	a := NewApp()
	if _, err := a.SetProjectRange("2026-01-01", "2026-12-31"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.AddSection("Work"); err != nil {
		t.Fatal(err)
	}
	act := mustWP(t, a, "spec", "2026-02-01", "2026-02-10") // end 02-10
	if _, err := a.AddMilestone("Gate", "2026-02-15", "Work"); err != nil {
		t.Fatal(err)
	}
	ms := lastMilestone(a)
	// FS action.end -> milestone: Start(ms)=02-15 >= End(act)=02-10 -> OK
	if _, err := a.AddDependency(act, ms, "FS", "r", "l"); err != nil {
		t.Fatal(err)
	}
	if depViolated(a, act, ms) {
		t.Fatal("FS action->milestone with milestone after action end should NOT be violated")
	}
	// move the milestone BEFORE the action end -> FS violated
	if _, err := a.MoveMilestone(ms, "2026-02-05"); err != nil {
		t.Fatal(err)
	}
	if !depViolated(a, act, ms) {
		t.Fatal("FS action->milestone with milestone before action end SHOULD be violated")
	}
	// SF action.start -> milestone: End(ms) >= Start(act)=02-01.
	// milestone at 02-05 -> 02-05 >= 02-01 -> OK
	if _, err := a.AddDependency(act, ms, "SF", "l", "r"); err != nil {
		t.Fatal(err)
	}
	if depViolated(a, act, ms) {
		t.Fatal("SF action->milestone with milestone after action start should NOT be violated")
	}
}
