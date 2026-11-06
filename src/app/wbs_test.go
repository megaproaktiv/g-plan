package main

import (
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/megaproaktiv/g-plan/src/app/plan"
)

// skynetApp is an App holding a project shaped like the reported one.
func skynetApp(t testing.TB) *App {
	a := NewApp()
	p := plan.NewProject("Skynet", plan.UnitDays)
	p.SetRange(mustDay("2026-11-01"), mustDay("2027-08-01"))
	add := func(id, name, cluster, start, end string) {
		p.Packages = append(p.Packages, plan.WorkPackage{ID: id, Name: name, Cluster: cluster,
			Start: mustDay(start), End: mustDay(end), Mode: plan.ModeDerived, Contour: plan.ContourFlat})
	}
	add("mob", "0 Mobilization", "Phases", "2026-11-07", "2027-01-29")
	add("found", "1 Foundation and pilot", "Phases", "2026-12-01", "2027-04-01")
	add("pm", "Project Management", "Section 1", "2026-11-04", "2027-07-31")
	add("aws", "AWS Landing Zone", "AWS landing zone", "2026-12-02", "2027-01-07")
	add("acc", "ACC Core configuration", "Connect core configuration", "2027-01-09", "2027-02-26")
	p.SectionColors = map[string]string{"AWS landing zone": "", "Additional Work Packages": "", "Connect core configuration": "", "Section 1": ""}
	a.project = p
	return a
}

func mustDay(s string) (d time.Time) { d, _ = plan.ParseDateStr(s); return }

func sectionOrder(a *App) []string {
	d := toDTO(a.project)
	out := make([]string, len(d.SectionNums))
	for name, n := range d.SectionNums {
		out[n-1] = name
	}
	return out
}

// Error 2: "move 3.1 to ..." must move the action the screen calls 3.1.
func TestMoveByDisplayNumberMovesTheDisplayedAction(t *testing.T) {
	a := skynetApp(t)
	d := toDTO(a.project)
	disp := map[string]string{}
	for _, w := range d.Packages {
		disp[w.ID] = w.DisplayID
	}
	if disp["aws"] != "2.1" || disp["acc"] != "3.1" || disp["mob"] != "P1" {
		t.Fatalf("setup: %v", disp)
	}
	awsStart := a.project.WorkPackage("aws").Start
	if _, err := a.Chat("move 3.1 to 2026-12-01"); err != nil {
		t.Fatal(err)
	}
	if got := a.project.WorkPackage("acc").Start.Format("2006-01-02"); got != "2026-12-01" {
		t.Errorf("3.1 (ACC Core configuration) must start 2026-12-01, got %s", got)
	}
	if !a.project.WorkPackage("aws").Start.Equal(awsStart) {
		t.Errorf("2.1 (AWS Landing Zone) must not have moved, now starts %s", a.project.WorkPackage("aws").Start.Format("2006-01-02"))
	}
	// the Phases lane is addressed by the P-numbers shown on screen
	if _, err := a.Chat("move P2 to 2027-01-04"); err != nil {
		t.Fatal(err)
	}
	if got := a.project.WorkPackage("found").Start.Format("2006-01-02"); got != "2027-01-04" {
		t.Errorf("P2 must start 2027-01-04, got %s", got)
	}
}

// Every number the screen shows resolves to the action that shows it, in every
// command that takes an action reference.
func TestEveryDisplayedNumberResolvesToItsAction(t *testing.T) {
	a := skynetApp(t)
	for _, w := range toDTO(a.project).Packages {
		if got := a.resolveActionRef(w.DisplayID); got != w.ID {
			t.Errorf("resolveActionRef(%q) = %q, want %q", w.DisplayID, got, w.ID)
		}
		if got := a.resolveTaskRef(w.DisplayID); got != w.ID {
			t.Errorf("resolveTaskRef(%q) = %q, want %q", w.DisplayID, got, w.ID)
		}
	}
	// detail codes n.k.m follow the same numbering
	if _, err := a.AddDetail("acc", "Detail", "2027-01-10", "2027-01-20", "1", "", ""); err != nil {
		t.Fatal(err)
	}
	if wp, det := a.resolveWBSRef("3.1.1"); wp != "acc" || det == "" {
		t.Errorf("3.1.1 = %q/%q", wp, det)
	}
	if wp, _ := a.resolveWBSRef("2.1.1"); wp != "" {
		t.Errorf("AWS has no detail 1: %q", wp)
	}
	// sections: WP numbers as on the lane labels
	for name, n := range toDTO(a.project).SectionNums {
		if got := a.resolveSectionRef(strings.TrimSpace(itoa(n))); got != name {
			t.Errorf("section %d = %q, want %q", n, got, name)
		}
	}
	// milestone code
	if _, err := a.AddMilestone("Go-live", "2026-12-20", "AWS landing zone"); err != nil {
		t.Fatal(err)
	}
	var code string
	for _, m := range toDTO(a.project).Milestones {
		if m.Name == "Go-live" {
			code = m.DisplayID
		}
	}
	if code != "2.2" || a.resolveMilestoneRef(code) == "" {
		t.Errorf("milestone code %q must resolve", code)
	}
}

// Error 1: a section without actions can be ordered like any other.
func TestOrderAnEmptySection(t *testing.T) {
	a := skynetApp(t)
	want := []string{"Section 1", "AWS landing zone", "Connect core configuration", "Additional Work Packages"}
	if got := sectionOrder(a); !reflect.DeepEqual(got, want) {
		t.Fatalf("start: %q", got)
	}
	// "Order up" on WP4 (the empty section) must move it to WP3, then WP2 ...
	if _, err := a.MoveSection("Additional Work Packages", -1); err != nil {
		t.Fatal(err)
	}
	want = []string{"Section 1", "AWS landing zone", "Additional Work Packages", "Connect core configuration"}
	if got := sectionOrder(a); !reflect.DeepEqual(got, want) {
		t.Fatalf("after one Order up: %q", got)
	}
	if _, err := a.MoveSection("Additional Work Packages", -1); err != nil {
		t.Fatal(err)
	}
	want = []string{"Section 1", "Additional Work Packages", "AWS landing zone", "Connect core configuration"}
	if got := sectionOrder(a); !reflect.DeepEqual(got, want) {
		t.Fatalf("after two: %q", got)
	}
	// the actions' numbers follow: AWS is now WP-3
	for _, w := range toDTO(a.project).Packages {
		if w.ID == "aws" && w.DisplayID != "3.1" {
			t.Errorf("AWS Landing Zone is displayed %q, want 3.1", w.DisplayID)
		}
	}
	// ... and "Order down" moves it back
	if _, err := a.MoveSection("Additional Work Packages", 1); err != nil {
		t.Fatal(err)
	}
	want = []string{"Section 1", "AWS landing zone", "Additional Work Packages", "Connect core configuration"}
	if got := sectionOrder(a); !reflect.DeepEqual(got, want) {
		t.Fatalf("after Order down: %q", got)
	}
	// the edges stay put, the Phases lane cannot be moved
	a.MoveSection("Section 1", -1)
	a.MoveSection("Connect core configuration", 1)
	a.MoveSection("Phases", 1)
	if got := sectionOrder(a); !reflect.DeepEqual(got, want) {
		t.Errorf("edges/Phases must not change the order: %q", got)
	}
	if _, err := a.MoveSection("Nope", -1); err == nil {
		t.Error("an unknown section is an error")
	}
}

func TestOrderingAUsedSectionAmongEmptyOnes(t *testing.T) {
	a := skynetApp(t)
	// move a section that HAS an action past the empty one at the end
	if _, err := a.MoveSection("Connect core configuration", 1); err != nil {
		t.Fatal(err)
	}
	want := []string{"Section 1", "AWS landing zone", "Additional Work Packages", "Connect core configuration"}
	if got := sectionOrder(a); !reflect.DeepEqual(got, want) {
		t.Fatalf("%q", got)
	}
	// the canvas order of the actions' sections is consistent with the numbers
	var seen []string
	for _, w := range a.project.Packages {
		if !plan.IsPhasesLane(w.Cluster) && (len(seen) == 0 || seen[len(seen)-1] != w.Cluster) {
			seen = append(seen, w.Cluster)
		}
	}
	if !reflect.DeepEqual(seen, []string{"Section 1", "AWS landing zone", "Connect core configuration"}) {
		t.Errorf("package order: %q", seen)
	}
}

func TestSectionOrderSurvivesSaveRenameRemoveAndUndo(t *testing.T) {
	a := skynetApp(t)
	if _, err := a.MoveSection("Additional Work Packages", -1); err != nil {
		t.Fatal(err)
	}
	if _, err := a.MoveSection("Additional Work Packages", -1); err != nil {
		t.Fatal(err)
	}
	want := []string{"Section 1", "Additional Work Packages", "AWS landing zone", "Connect core configuration"}

	// saved with the project
	path := filepath.Join(t.TempDir(), "p.ptplan")
	if err := a.project.SaveJSON(path); err != nil {
		t.Fatal(err)
	}
	b := NewApp()
	if _, err := b.OpenProject(path); err != nil {
		t.Fatal(err)
	}
	if got := sectionOrder(b); !reflect.DeepEqual(got, want) {
		t.Errorf("after reload: %q", got)
	}

	// a rename keeps the section's place
	if _, err := a.RenameSection("Additional Work Packages", "Extras"); err != nil {
		t.Fatal(err)
	}
	if got := sectionOrder(a); !reflect.DeepEqual(got, []string{"Section 1", "Extras", "AWS landing zone", "Connect core configuration"}) {
		t.Errorf("after rename: %q", got)
	}
	// removing it takes it out of the numbering; undo brings it back in place
	if _, err := a.RemoveSection("Extras"); err != nil {
		t.Fatal(err)
	}
	if got := sectionOrder(a); !reflect.DeepEqual(got, []string{"Section 1", "AWS landing zone", "Connect core configuration"}) {
		t.Errorf("after remove: %q", got)
	}
	if _, err := a.Undo(); err != nil {
		t.Fatal(err)
	}
	if got := sectionOrder(a); !reflect.DeepEqual(got, []string{"Section 1", "Extras", "AWS landing zone", "Connect core configuration"}) {
		t.Errorf("after undo: %q", got)
	}
}

func itoa(n int) string { return strconv.Itoa(n) }
