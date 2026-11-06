package main

import (
	"strings"
	"testing"

	"github.com/megaproaktiv/g-plan/src/app/plan"
)

func TestContourNoneGreysOutEstimationAndReal(t *testing.T) {
	a, ids := monitorApp(t) // Design 10, Build 20, Test 5 (hours)
	id := ids[0]
	if _, err := a.SetWorkPackageRealEffort(id, "3"); err != nil {
		t.Fatal(err)
	}
	totalBefore := a.project.TotalEst()

	dto, err := a.SetWorkPackageContour(id, "None")
	if err != nil {
		t.Fatalf("None must be selectable: %v", err)
	}
	w := dto.Packages[0]
	if w.Contour != "None" || w.Est != "0" || w.EstHours != "0" || w.RealEst != "0" || w.RealHours != "0" {
		t.Errorf("a None action shows no estimation and no real effort: %+v", w)
	}
	if w.ProgressPermille != 0 || w.ContourPermille != 0 || w.CompletionPos != 0 || w.ProgressPercent != 0 {
		t.Errorf("a None action has no progress: %+v", w)
	}
	if a.project.TotalEst() >= totalBefore {
		t.Error("the project total must no longer include the None action")
	}
	// the greyed-out fields cannot be changed through the backend either
	if _, err := a.SetWorkPackageEstimation(id, "7"); err == nil || !strings.Contains(err.Error(), "None") {
		t.Errorf("estimation of a None action: %v", err)
	}
	if _, err := a.SetWorkPackageRealEffort(id, "7"); err == nil || !strings.Contains(err.Error(), "None") {
		t.Errorf("real effort of a None action: %v", err)
	}
	// "Apply" in the editor sends the displayed (0) estimation: it must not wipe
	// the hidden value, since the field is greyed out and not meant to be edited
	stored := a.project.WorkPackage(id).Est
	if _, err := a.SetActionFields(id, "Design", "", "0", "", ""); err != nil {
		t.Fatal(err)
	}
	if got := a.project.WorkPackage(id).Est; got != stored {
		t.Errorf("the hidden estimation must survive Apply: got %d, want %d", got, stored)
	}

	// switching back restores both values
	dto, err = a.SetWorkPackageContour(id, "Flat")
	if err != nil {
		t.Fatal(err)
	}
	w = dto.Packages[0]
	if w.RealHours != "3" || w.EstHours != "10" {
		t.Errorf("values must come back: est=%s real=%s", w.EstHours, w.RealHours)
	}
	if w.ProgressPermille != 300 {
		t.Errorf("3 of 10 h = 30 %%: %d", w.ProgressPermille)
	}
}

func TestNoneAndTheMonitorDefaultAction(t *testing.T) {
	a, ids := monitorApp(t)
	if _, err := a.SetWorkPackageContour(ids[2], "None"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetMonitorDefaultAction(ids[2]); err == nil || !strings.Contains(err.Error(), "None") {
		t.Errorf("a None action cannot be the default action: %v", err)
	}
	if _, err := a.SetWorkPackageContour(ids[2], "Flat"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetMonitorDefaultAction(ids[2]); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetWorkPackageContour(ids[2], "None"); err == nil || !strings.Contains(err.Error(), "default action") {
		t.Errorf("the default action cannot be switched to None: %v", err)
	}
	if a.project.WorkPackage(ids[2]).Contour != plan.ContourFlat {
		t.Error("a refused change must leave the contour alone")
	}
}

func TestSyncSkipsNoneActions(t *testing.T) {
	a, ids := monitorApp(t)
	srv := fakeAerionServer(t, []string{"WP-1 design", "WP-2 build"}, []int{3600, 7200})
	defer srv.Close()
	if err := a.SaveSystemConfig("aerion", srv.URL, "u", "p", "c", "s"); err != nil {
		t.Fatal(err)
	}
	// Design is a None action (cannot carry an ID, so nobody takes "WP-1"); Build has WP-2
	if _, err := a.SetWorkPackageContour(ids[0], "None"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetActionExternalID(ids[1], "WP-2"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetMonitor(true, "aerion", "42"); err != nil {
		t.Fatal(err)
	}
	sum, err := a.SyncMonitor()
	if err != nil {
		t.Fatal(err)
	}
	if sum.Entries != 1 || sum.UnmatchedEntries != 1 {
		t.Errorf("only the Build booking matches; the None action's booking is unassigned: %+v", sum)
	}
	w := toDTO(a.project)
	for _, p := range w.Packages {
		if p.ID == ids[0] && (p.Synced || p.RealHours != "0") {
			t.Errorf("the None action is not synchronized: %+v", p)
		}
	}
}

// A None action is not a real action: it cannot carry an External ID.
func TestNoneActionCannotCarryAnExternalID(t *testing.T) {
	a, ids := monitorApp(t)
	id := ids[0]
	if _, err := a.SetWorkPackageContour(id, "None"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetActionExternalID(id, "WP-9"); err == nil || !strings.Contains(err.Error(), "None") {
		t.Errorf("setting an External ID on a None action: %v", err)
	}
	if a.project.WorkPackage(id).ExternalID != "" {
		t.Error("a refused ID must not be stored")
	}
	// clearing is always fine
	if _, err := a.SetActionExternalID(id, ""); err != nil {
		t.Errorf("clearing: %v", err)
	}

	// an action WITH an ID cannot be switched to None until the ID is removed
	if _, err := a.SetWorkPackageContour(id, "Flat"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetActionExternalID(id, "WP-9"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetWorkPackageContour(id, "None"); err == nil ||
		!strings.Contains(err.Error(), "WP-9") || !strings.Contains(err.Error(), "remove the External ID first") {
		t.Errorf("switching an action with an External ID to None: %v", err)
	}
	if got := a.project.WorkPackage(id); got.Contour != plan.ContourFlat || got.ExternalID != "WP-9" {
		t.Errorf("a refused change must leave the action alone: %+v", got)
	}
	if _, err := a.SetActionExternalID(id, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetWorkPackageContour(id, "None"); err != nil {
		t.Errorf("without an ID the switch works: %v", err)
	}
}

// External IDs stay editable while monitoring is on; the change clears the old
// result and the next synchronization uses the new ID.
func TestExternalIDIsEditableWhileMonitoringIsOn(t *testing.T) {
	a, ids := monitorApp(t)
	srv := fakeAerionServer(t, []string{"WP-5 design work"}, []int{3600})
	defer srv.Close()
	if err := a.SaveSystemConfig("aerion", srv.URL, "u", "p", "c", "s"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetMonitor(true, "aerion", "42"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetActionExternalID(ids[0], "WP-1"); err != nil {
		t.Fatalf("must be editable while monitoring is on: %v", err)
	}
	sum, err := a.SyncMonitor()
	if err != nil {
		t.Fatal(err)
	}
	if sum.Entries != 0 || sum.UnmatchedEntries != 1 {
		t.Fatalf("WP-1 does not match the booking yet: %+v", sum)
	}
	if a.GetMonitorSummary() == nil {
		t.Fatal("setup: there is a result")
	}
	if _, err := a.SetActionExternalID(ids[0], "WP-5"); err != nil {
		t.Fatal(err)
	}
	if a.GetMonitorSummary() != nil {
		t.Error("changing an External ID must clear the stale result")
	}
	sum, err = a.SyncMonitor()
	if err != nil {
		t.Fatal(err)
	}
	if sum.Entries != 1 || sum.UnmatchedEntries != 0 {
		t.Errorf("the new ID is used by the next synchronization: %+v", sum)
	}
	// clearing while monitoring is on works as well, and frees the ID for others
	if _, err := a.SetActionExternalID(ids[0], ""); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetActionExternalID(ids[1], "WP-5"); err != nil {
		t.Errorf("a freed ID can be reused: %v", err)
	}
}
