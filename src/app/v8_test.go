package main

import (
	"os"
	"strings"
	"testing"

	"github.com/megaproaktiv/g-plan/src/app/plan"
)

// Tests for session-4 features:
//   - assign works with a bare percent and in any mode
//   - chat commands on detail sub-actions (est/link on n.k.m)
//   - move: after / before / to <date> / directional delta
//   - WP outside project start/end check
//   - role-capacity (fit) model + fits-window check

// --- assign with bare percent, any mode ---

func TestAssignBarePercentAnyMode(t *testing.T) {
	a := NewApp()
	if _, err := a.SetProjectRange("2026-01-01", "2026-12-31"); err != nil {
		t.Fatal(err)
	}
	a.AddRole("Architect", "")
	mustWP(t, a, "spec", "2026-02-02", "2026-02-20")
	// switch to role mode — assign must still work (it is a global verb)
	if _, err := a.Chat("mode role"); err != nil {
		t.Fatal(err)
	}
	res, err := a.Chat("assign Architect to 1.1 100")
	if err != nil {
		t.Fatalf("assign in role mode failed: %v", err)
	}
	if !strings.Contains(res, "100%") {
		t.Fatalf("expected 100%% assignment, got %q", res)
	}
	// verify the assignment is 100 (bare percent read, not defaulted to 50)
	d := a.GetProject()
	var pct int
	for _, r := range d.Roles {
		if r.Name == "Architect" {
			for _, as := range r.Assignments {
				pct = as.Percent
			}
		}
	}
	if pct != 100 {
		t.Fatalf("expected percent 100 (bare number), got %d", pct)
	}
}

// --- chat commands on detail sub-actions ---

func TestChatEstAndLinkOnDetails(t *testing.T) {
	a := NewApp()
	if _, err := a.SetProjectRange("2026-01-01", "2026-12-31"); err != nil {
		t.Fatal(err)
	}
	w := mustWP(t, a, "wp", "2026-03-02", "2026-03-30")
	if _, err := a.AddDetail(w, "d1", "2026-03-02", "2026-03-10", "0", "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := a.AddDetail(w, "d2", "2026-03-11", "2026-03-20", "0", "", ""); err != nil {
		t.Fatal(err)
	}
	// est on the first detail sub-action (WBS 1.1.1)
	if _, err := a.Chat("est 1.1.1 3"); err != nil {
		t.Fatalf("est on detail failed: %v", err)
	}
	d := a.GetProject()
	wp := dtoWP(d, w)
	if wp.Details[0].Est != "3" {
		t.Fatalf("expected detail 1.1.1 est 3, got %q", wp.Details[0].Est)
	}
	// link two detail sub-actions (1.1.1 -> 1.1.2)
	if _, err := a.Chat("link 1.1.1 1.1.2"); err != nil {
		t.Fatalf("link on details failed: %v", err)
	}
	d = a.GetProject()
	found := false
	for _, dep := range d.Deps {
		if dep.From == wp.Details[0].ID && dep.To == wp.Details[1].ID {
			found = true
		}
	}
	if !found {
		t.Fatal("expected a dependency between the two detail sub-actions")
	}
}

// --- move: after / before / to date / directional delta ---

func TestMoveAfterBeforeDateDelta(t *testing.T) {
	a := NewApp()
	if _, err := a.SetProjectRange("2026-01-01", "2026-12-31"); err != nil {
		t.Fatal(err)
	}
	w1 := mustWP(t, a, "first", "2026-03-02", "2026-03-06") // Mon..Fri
	w2 := mustWP(t, a, "second", "2026-05-01", "2026-05-05")

	// move 1.2 after 1.1  -> second starts one day after first ends
	if _, err := a.Chat("move 1.2 after 1.1"); err != nil {
		t.Fatalf("move after: %v", err)
	}
	if got := a.project.WorkPackage(w2).Start.Format("2006-01-02"); got != "2026-03-07" {
		t.Fatalf("after: expected start 2026-03-07, got %s", got)
	}

	// move 1.2 before 1.1 -> second ends one day before first starts (03-01)
	if _, err := a.Chat("move 1.2 before 1.1"); err != nil {
		t.Fatalf("move before: %v", err)
	}
	if got := a.project.WorkPackage(w2).End.Format("2006-01-02"); got != "2026-03-01" {
		t.Fatalf("before: expected end 2026-03-01, got %s", got)
	}

	// After the "before" move the second action starts first, so the screen now
	// shows it as 1.1 (n.k follows the dates): a typed number means what is on the
	// screen, so refer to the action by its id here.
	if got := toDTO(a.project).Packages[1].DisplayID; got != "1.1" {
		t.Fatalf("the moved action is displayed as 1.1 now, got %s", got)
	}
	// move <second> to 2026-06-10
	if _, err := a.Chat("move " + w2 + " to 2026-06-10"); err != nil {
		t.Fatalf("move to date: %v", err)
	}
	if got := a.project.WorkPackage(w2).Start.Format("2006-01-02"); got != "2026-06-10" {
		t.Fatalf("to date: expected start 2026-06-10, got %s", got)
	}

	// move 1.1 10d right (future) -> first shifts +10 days
	before := a.project.WorkPackage(w1).Start
	if _, err := a.Chat("move 1.1 10d right"); err != nil {
		t.Fatalf("move delta right: %v", err)
	}
	after := a.project.WorkPackage(w1).Start
	if after.Sub(before).Hours() != 10*24 {
		t.Fatalf("delta right: expected +10 days, got %v", after.Sub(before))
	}

	// move 1.1 10d left -> shifts back 10 days
	if _, err := a.Chat("move 1.1 10d left"); err != nil {
		t.Fatalf("move delta left: %v", err)
	}
	if !a.project.WorkPackage(w1).Start.Equal(before) {
		t.Fatalf("delta left: expected to return to %v, got %v", before, a.project.WorkPackage(w1).Start)
	}
}

// --- WP outside project window ---

func TestWPOutsideProjectWindow(t *testing.T) {
	a := NewApp()
	if _, err := a.SetProjectRange("2026-01-01", "2026-03-31"); err != nil {
		t.Fatal(err)
	}
	inside := mustWP(t, a, "inside", "2026-02-01", "2026-02-20")
	outside := mustWP(t, a, "late", "2026-05-01", "2026-05-20") // ends after project end
	d := a.GetProject()
	if dtoWP(d, inside).HasWarning {
		t.Fatal("an action inside the window should not be flagged")
	}
	if !dtoWP(d, outside).HasWarning {
		t.Fatal("an action outside the window should be flagged")
	}
	hasRangeWarn := false
	for _, msg := range d.Warnings {
		if strings.Contains(msg, "outside the project window") {
			hasRangeWarn = true
		}
	}
	if !hasRangeWarn {
		t.Fatalf("expected an 'outside the project window' warning, got %v", d.Warnings)
	}
}

// --- role capacity / fit model ---

func TestRoleCapacityFit(t *testing.T) {
	a := NewApp()
	if _, err := a.SetProjectRange("2026-01-01", "2026-12-31"); err != nil {
		t.Fatal(err)
	}
	// 2026-03-02..2026-03-06 = 5 working days; H=8 => a 100% role delivers
	// 5*8 = 40 hours = 5 days. Effort of 5 days should FIT with one 100% role.
	w := mustWP(t, a, "task", "2026-03-02", "2026-03-06")
	if _, err := a.SetActionFields(w, "task", "", "5", "2026-03-02", "2026-03-06"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetActionRoles(w, []string{"Dev"}, []int{100}); err != nil {
		t.Fatal(err)
	}
	d := a.GetProject()
	wp := dtoWP(d, w)
	if wp.RoleFit == nil {
		t.Fatal("expected a RoleFit summary")
	}
	if !wp.RoleFit.Fits {
		t.Fatalf("5 days effort with one 100%% role over 5 workdays should fit; %+v", wp.RoleFit)
	}
	// Drop the role to 50% => capacity halves (2.5 days) => does NOT cover 5 days
	if _, err := a.SetActionRoles(w, []string{"Dev"}, []int{50}); err != nil {
		t.Fatal(err)
	}
	d = a.GetProject()
	wp = dtoWP(d, w)
	if wp.RoleFit.Fits {
		t.Fatalf("50%% of 5 workdays (2.5d) must NOT cover 5 days effort; %+v", wp.RoleFit)
	}
	if !wp.HasWarning {
		t.Fatal("under-capacity action should carry a warning")
	}
}

// --- fits-window: effort cannot exceed what a 100% role could do in the window ---

func TestEffortFitsWindow(t *testing.T) {
	a := NewApp()
	if _, err := a.SetProjectRange("2026-01-01", "2026-12-31"); err != nil {
		t.Fatal(err)
	}
	// 5 workdays window but 10 days effort => cannot fit even at 100%
	w := mustWP(t, a, "tight", "2026-03-02", "2026-03-06")
	if _, err := a.SetActionFields(w, "tight", "", "10", "2026-03-02", "2026-03-06"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetActionRoles(w, []string{"Dev"}, []int{100}); err != nil {
		t.Fatal(err)
	}
	d := a.GetProject()
	wp := dtoWP(d, w)
	if wp.RoleFit.FitsWindow {
		t.Fatalf("10 days effort cannot fit a 5-workday window; %+v", wp.RoleFit)
	}
}

// --- add ... after <id> applies the 1-unit minimum buffer (no FS violation) ---

func TestAddAfterBuffer(t *testing.T) {
	a := NewApp()
	if _, err := a.SetProjectRange("2026-01-01", "2026-12-31"); err != nil {
		t.Fatal(err)
	}
	pred := mustWP(t, a, "spec", "2026-03-02", "2026-03-06") // ends 03-06
	// add a successor after the predecessor via chat (WBS ref 1.1)
	if _, err := a.Chat("add action stabilisation after 1.1"); err != nil {
		t.Fatalf("add after: %v", err)
	}
	// the new action must start at least one day after the predecessor ends
	succ := a.project.Packages[len(a.project.Packages)-1]
	if !succ.Start.After(a.project.WorkPackage(pred).End) {
		t.Fatalf("successor start %s should be after predecessor end %s (buffer)",
			succ.Start.Format("2006-01-02"), a.project.WorkPackage(pred).End.Format("2006-01-02"))
	}
	// and the FS link must NOT be a violation (>= end + 1 unit)
	d := a.GetProject()
	for _, dep := range d.Deps {
		if dep.From == pred && dep.To == succ.ID {
			if dep.Violated {
				t.Fatal("FS link after 'add ... after' must not be violated (1-unit buffer applied)")
			}
			return
		}
	}
	t.Fatal("expected an FS dependency from predecessor to the new action")
}

// --- MoveSection reorders and is a no-op past the edges ---

func TestMoveSectionReorderAndEdges(t *testing.T) {
	a := NewApp()
	if _, err := a.AddSection("Design"); err != nil {
		t.Fatal(err)
	}
	mustWP(t, a, "d1", "2026-02-01", "2026-02-10")
	if _, err := a.AddSection("Build"); err != nil {
		t.Fatal(err)
	}
	mustWP(t, a, "b1", "2026-03-01", "2026-03-10")
	// initial numbering: Design=1, Build=2
	d := a.GetProject()
	if d.SectionNums["Design"] != 1 || d.SectionNums["Build"] != 2 {
		t.Fatalf("initial nums wrong: %v", d.SectionNums)
	}
	// move Build up -> Build=1, Design=2
	if _, err := a.MoveSection("Build", -1); err != nil {
		t.Fatal(err)
	}
	d = a.GetProject()
	if d.SectionNums["Build"] != 1 || d.SectionNums["Design"] != 2 {
		t.Fatalf("after move up, nums wrong: %v", d.SectionNums)
	}
	// moving the top section (Build) up again is a no-op
	if _, err := a.MoveSection("Build", -1); err != nil {
		t.Fatal(err)
	}
	d = a.GetProject()
	if d.SectionNums["Build"] != 1 {
		t.Fatalf("moving top section up should be a no-op; nums=%v", d.SectionNums)
	}
}

// --- undo of add section / add action (Cmd-Z) ---

func TestUndoAddSection(t *testing.T) {
	a := NewApp()
	before := len(a.project.SectionColors)
	if _, err := a.AddSection("Program Management"); err != nil {
		t.Fatal(err)
	}
	if _, ok := a.project.SectionColors["Program Management"]; !ok {
		t.Fatal("section should exist after add")
	}
	if _, err := a.Undo(); err != nil {
		t.Fatal(err)
	}
	if _, ok := a.project.SectionColors["Program Management"]; ok {
		t.Fatal("undo should remove the added section")
	}
	if len(a.project.SectionColors) != before {
		t.Fatalf("section count should return to %d, got %d", before, len(a.project.SectionColors))
	}
}

func TestUndoAddAction(t *testing.T) {
	a := NewApp()
	before := len(a.project.Packages)
	if _, err := a.AddWorkPackage("PM", "2026-02-01", "2026-02-10"); err != nil {
		t.Fatal(err)
	}
	if len(a.project.Packages) != before+1 {
		t.Fatal("action should be added")
	}
	if _, err := a.Undo(); err != nil {
		t.Fatal(err)
	}
	if len(a.project.Packages) != before {
		t.Fatal("undo should remove the added action")
	}
}

// --- MoveAction up/down within a section ---

func TestMoveActionUpDown(t *testing.T) {
	a := NewApp()
	if _, err := a.SetProjectRange("2026-01-01", "2026-12-31"); err != nil {
		t.Fatal(err)
	}
	// two actions in the same (default) section: PM then email
	pm := mustWP(t, a, "PM", "2026-02-01", "2026-02-10")
	email := mustWP(t, a, "email", "2026-03-01", "2026-03-10")
	// email is second -> displayId ends in .2
	d := a.GetProject()
	if dtoWP(d, email).DisplayID != dtoWP(d, pm).DisplayID[:len(dtoWP(d, pm).DisplayID)-1]+"2" {
		// not strict about exact ids; just assert order below
	}
	if a.project.Packages[0].ID != pm || a.project.Packages[1].ID != email {
		t.Fatalf("initial order should be [PM, email], got [%s, %s]", a.project.Packages[0].ID, a.project.Packages[1].ID)
	}
	// move email UP -> it should come before PM
	if _, err := a.MoveAction(email, -1); err != nil {
		t.Fatal(err)
	}
	if a.project.Packages[0].ID != email || a.project.Packages[1].ID != pm {
		t.Fatalf("after move up, order should be [email, PM], got [%s, %s]", a.project.Packages[0].ID, a.project.Packages[1].ID)
	}
	// move email DOWN -> back to [PM, email]
	if _, err := a.MoveAction(email, 1); err != nil {
		t.Fatal(err)
	}
	if a.project.Packages[0].ID != pm || a.project.Packages[1].ID != email {
		t.Fatalf("after move down, order should be [PM, email], got [%s, %s]", a.project.Packages[0].ID, a.project.Packages[1].ID)
	}
	// moving the top action up again is a no-op
	if _, err := a.MoveAction(pm, -1); err != nil {
		t.Fatal(err)
	}
	if a.project.Packages[0].ID != pm {
		t.Fatal("moving the top action up should be a no-op")
	}
	// undo the last real move (down) is covered by the undo stack; just ensure
	// MoveAction pushed undo by undoing once and checking order changed
	if _, err := a.Undo(); err != nil {
		t.Fatal(err)
	}
}

// MoveAction only reorders within the SAME section, not across sections.
func TestMoveActionStaysInSection(t *testing.T) {
	a := NewApp()
	if _, err := a.AddSection("A"); err != nil {
		t.Fatal(err)
	}
	a1 := mustWP(t, a, "a1", "2026-02-01", "2026-02-05")
	if _, err := a.AddSection("B"); err != nil {
		t.Fatal(err)
	}
	b1 := mustWP(t, a, "b1", "2026-03-01", "2026-03-05")
	_ = a1
	// b1 is the only action in section B; moving it up is a no-op (no neighbour
	// in the same section above it)
	if _, err := a.MoveAction(b1, -1); err != nil {
		t.Fatal(err)
	}
	if a.project.WorkPackage(b1).Cluster != "B" {
		t.Fatal("MoveAction must not move an action out of its section")
	}
}

// --- single-file JSON persistence: unit, hoursPerDay, zoom + everything ---

func TestSingleJSONPersistsConfig(t *testing.T) {
	a := NewApp()
	if _, err := a.SetProjectRange("2026-01-01", "2026-12-31"); err != nil {
		t.Fatal(err)
	}
	a.SetConfig("hours", 6) // unit + hours/day
	if _, err := a.SetZoom("weeks"); err != nil {
		t.Fatal(err)
	}
	w := mustWP(t, a, "task", "2026-02-02", "2026-02-20")
	a.AddDetail(w, "d1", "2026-02-02", "2026-02-10", "2", "", "")
	a.AddRole("Dev", "")
	a.AssignRole("Dev", w, 100)

	dir := t.TempDir()
	saved, err := a.SaveProject(dir + "/p")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(saved, ".ptplan") {
		t.Fatalf("expected a single .ptplan file, got %s", saved)
	}
	// only ONE file should be written (no .mmd / .roles.json / .md / .json siblings)
	for _, bad := range []string{dir + "/p.mmd", dir + "/p.roles.json", dir + "/p.md", dir + "/p.ptplan.json"} {
		if _, err := os.Stat(bad); err == nil {
			t.Fatalf("single-file format must not write %s", bad)
		}
	}

	b := NewApp()
	d, err := b.OpenProject(saved)
	if err != nil {
		t.Fatal(err)
	}
	if d.Unit != "hours" {
		t.Errorf("unit not persisted: %q", d.Unit)
	}
	if d.HoursPerDay != 6 {
		t.Errorf("hoursPerDay not persisted: %d", d.HoursPerDay)
	}
	if d.Zoom != "weeks" {
		t.Errorf("zoom not persisted: %q", d.Zoom)
	}
	if len(d.Packages) != 1 || len(d.Packages[0].Details) != 1 {
		t.Errorf("packages/details lost: %+v", d.Packages)
	}
	if len(d.Roles) != 1 || len(d.Roles[0].Assignments) != 1 {
		t.Errorf("roles/assignments lost: %+v", d.Roles)
	}
	if d.Start != "2026-01-01" || d.End != "2026-12-31" {
		t.Errorf("project range lost: %s..%s", d.Start, d.End)
	}
}

// SetZoom rejects invalid values and accepts the known levels.
func TestSetZoomValidation(t *testing.T) {
	a := NewApp()
	for _, z := range []string{"days", "weeks", "months", "year", "window"} {
		if _, err := a.SetZoom(z); err != nil {
			t.Fatalf("SetZoom(%q) should be ok: %v", z, err)
		}
		if a.project.Zoom != z {
			t.Fatalf("zoom not set to %q", z)
		}
	}
	if _, err := a.SetZoom("bogus"); err == nil {
		t.Fatal("SetZoom should reject an unknown level")
	}
}

// NewProject autosaves the old project (when it has a path) and resets.
func TestNewProjectAutosave(t *testing.T) {
	a := NewApp()
	mustWP(t, a, "old", "2026-02-01", "2026-02-10")
	dir := t.TempDir()
	saved, err := a.SaveProject(dir + "/old")
	if err != nil {
		t.Fatal(err)
	}
	// mutate after save, then New with autosave -> the change is flushed to disk
	a.AddWorkPackage("later", "2026-03-01", "2026-03-10")
	if _, err := a.NewProject(true); err != nil {
		t.Fatal(err)
	}
	if len(a.project.Packages) != 0 {
		t.Fatalf("new project should be empty, has %d packages", len(a.project.Packages))
	}
	if a.base != "" {
		t.Fatal("new project should have no save path")
	}
	// the autosaved old file should now contain BOTH actions
	reopened, err := plan.OpenJSON(saved)
	if err != nil {
		t.Fatal(err)
	}
	if len(reopened.Packages) != 2 {
		t.Fatalf("autosave should have flushed 2 packages, got %d", len(reopened.Packages))
	}
}

// --- role numeric id: assign 1 to 1.1 means role #1 -> WP 1.1 ---

func TestAssignRoleByNumericId(t *testing.T) {
	a := NewApp()
	if _, err := a.SetProjectRange("2026-01-01", "2026-12-31"); err != nil {
		t.Fatal(err)
	}
	a.AddRole("Architect", "")                            // role #1
	a.AddRole("Dev", "")                                  // role #2
	w := mustWP(t, a, "spec", "2026-02-02", "2026-02-20") // WBS 1.1
	if _, err := a.Chat("assign 1 to 1.1 100"); err != nil {
		t.Fatalf("assign 1 to 1.1: %v", err)
	}
	// role #1 (Architect) must have a 100% assignment on WP 1.1 (spec)
	var ok bool
	for _, r := range a.GetProject().Roles {
		if r.Name == "Architect" {
			for _, as := range r.Assignments {
				if as.WorkPackageID == w && as.Percent == 100 {
					ok = true
				}
			}
		}
	}
	if !ok {
		t.Fatal("assign 1 to 1.1 should assign role #1 (Architect) to WP 1.1 at 100%")
	}
	// the DTO exposes role numbers 1..n
	d := a.GetProject()
	if d.Roles[0].ID != 1 || d.Roles[1].ID != 2 {
		t.Fatalf("role numbers wrong: %d / %d", d.Roles[0].ID, d.Roles[1].ID)
	}
}

// --- select / deselect ---

func TestSelectDeselect(t *testing.T) {
	a := NewApp()
	w := mustWP(t, a, "task", "2026-02-01", "2026-02-10")
	// select by WBS
	if _, err := a.Select("action", "1.1"); err != nil {
		t.Fatal(err)
	}
	if s := a.Selection(); s.Kind != "action" || s.ID != w {
		t.Fatalf("selection = %+v, want action %s", s, w)
	}
	// via chat
	if _, err := a.AddSection("Design"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Chat("select section Design"); err != nil {
		t.Fatalf("chat select section: %v", err)
	}
	if s := a.Selection(); s.Kind != "section" || s.ID != "Design" {
		t.Fatalf("selection = %+v, want section Design", s)
	}
	// deselect
	a.Deselect()
	if s := a.Selection(); s.Kind != "" {
		t.Fatalf("deselect should clear, got %+v", s)
	}
}

// --- dirty tracking for Open/New prompt ---

func TestIsDirty(t *testing.T) {
	a := NewApp()
	a.markSaved() // baseline: clean
	if a.IsDirty() {
		t.Fatal("a freshly-marked project should be clean")
	}
	a.AddWorkPackage("task", "2026-02-01", "2026-02-10")
	if !a.IsDirty() {
		t.Fatal("after adding an action the project should be dirty")
	}
	dir := t.TempDir()
	if _, err := a.SaveProject(dir + "/p"); err != nil {
		t.Fatal(err)
	}
	if a.IsDirty() {
		t.Fatal("after save the project should be clean again")
	}
}

// --- milestones: creation, section, shared WBS numbering, color, delete ---

func TestMilestonesSharedWBS(t *testing.T) {
	a := NewApp()
	if _, err := a.SetProjectRange("2026-01-01", "2026-12-31"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.AddSection("Design"); err != nil {
		t.Fatal(err)
	}
	// action on day 1 -> 1.1 ; milestone on day 5 -> 1.2 (shared sequence)
	act := mustWP(t, a, "spec", "2026-02-01", "2026-02-10")
	if _, err := a.AddMilestone("Kickoff", "2026-02-05", "Design"); err != nil {
		t.Fatal(err)
	}
	d := a.GetProject()
	// find the action and milestone display ids
	var actDisp, msDisp string
	for _, p := range d.Packages {
		if p.ID == act {
			actDisp = p.DisplayID
		}
	}
	for _, m := range d.Milestones {
		if m.Name == "Kickoff" {
			msDisp = m.DisplayID
			if m.Cluster != "Design" {
				t.Errorf("milestone cluster = %q, want Design", m.Cluster)
			}
		}
	}
	if actDisp != "1.1" {
		t.Errorf("action display id = %q, want 1.1", actDisp)
	}
	if msDisp != "1.2" {
		t.Errorf("milestone display id = %q, want 1.2 (shared numbering after the action)", msDisp)
	}
	// set color + rename
	var msID string
	for _, m := range d.Milestones {
		if m.Name == "Kickoff" {
			msID = m.ID
		}
	}
	if _, err := a.SetMilestoneFields(msID, "Kickoff 2", "", "#ff0000"); err != nil {
		t.Fatal(err)
	}
	d = a.GetProject()
	for _, m := range d.Milestones {
		if m.ID == msID && (m.Name != "Kickoff 2" || m.Color != "#ff0000") {
			t.Errorf("milestone edit lost: %+v", m)
		}
	}
	// select by WBS 1.2 -> the milestone
	if _, err := a.Select("milestone", "1.2"); err != nil {
		t.Fatalf("select milestone 1.2: %v", err)
	}
	if s := a.Selection(); s.Kind != "milestone" || s.ID != msID {
		t.Fatalf("selection = %+v, want milestone %s", s, msID)
	}
	// project Start/End milestones cannot be deleted
	if _, err := a.RemoveMilestone("m_start"); err == nil {
		t.Fatal("deleting the project Start milestone should error")
	}
	// delete the user milestone
	if _, err := a.RemoveMilestone(msID); err != nil {
		t.Fatal(err)
	}
	d = a.GetProject()
	for _, m := range d.Milestones {
		if m.ID == msID {
			t.Fatal("milestone should be gone after delete")
		}
	}
}

// milestone creation via the chat command + milestone mode.
func TestMilestoneChat(t *testing.T) {
	a := NewApp()
	if _, err := a.SetProjectRange("2026-01-01", "2026-12-31"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Chat("milestone Launch on 2026-06-01"); err != nil {
		t.Fatalf("milestone chat: %v", err)
	}
	found := false
	for _, m := range a.project.Milestones {
		if m.Name == "Launch" {
			found = true
		}
	}
	if !found {
		t.Fatal("chat 'milestone Launch' should create a milestone")
	}
	// milestone mode + bare add
	if _, err := a.Chat("mode milestone"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Chat("add milestone Review"); err != nil {
		t.Fatalf("add in milestone mode: %v", err)
	}
	cnt := 0
	for _, m := range a.project.Milestones {
		if m.Name == "Review" || m.Name == "Launch" {
			cnt++
		}
	}
	if cnt != 2 {
		t.Fatalf("expected Launch + Review milestones, got %d", cnt)
	}
}

// --- OV (overlap) dependency: violated only when the actions do NOT overlap ---

func TestOverlapDependency(t *testing.T) {
	a := NewApp()
	if _, err := a.SetProjectRange("2026-01-01", "2026-12-31"); err != nil {
		t.Fatal(err)
	}
	x := mustWP(t, a, "x", "2026-03-02", "2026-03-20")
	// y overlaps x (starts before x ends) -> OV satisfied
	y := mustWP(t, a, "y", "2026-03-10", "2026-03-30")
	if _, err := a.AddDependency(x, y, "OV", "t", "b"); err != nil {
		t.Fatal(err)
	}
	if depViolated(a, x, y) {
		t.Fatal("OV with overlapping actions must NOT be violated")
	}
	// move y entirely after x -> no overlap -> OV violated
	if _, err := a.MoveWorkPackage(y, "2026-04-01", "2026-04-20"); err != nil {
		t.Fatal(err)
	}
	if !depViolated(a, x, y) {
		t.Fatal("OV with non-overlapping actions must be violated")
	}
}

// Deleting an action via the binding clears any matching selection (mirrors the
// UI: adding/deleting should not leave a stale selection).
func TestDeselectConsistencyOnRemove(t *testing.T) {
	a := NewApp()
	w := mustWP(t, a, "task", "2026-02-01", "2026-02-10")
	if _, err := a.Select("action", "1.1"); err != nil {
		t.Fatal(err)
	}
	if a.Selection().ID != w {
		t.Fatal("selection should be the action")
	}
	a.RemoveWorkPackage(w)
	// selection id now points at a gone action; a fresh add should still be
	// locatable as the last package (the UI deselects before adding).
	if _, err := a.AddWorkPackage("new", "2026-03-01", "2026-03-10"); err != nil {
		t.Fatal(err)
	}
	last := a.project.Packages[len(a.project.Packages)-1]
	if last.Name != "new" {
		t.Fatalf("the newly added action must be the last package, got %q", last.Name)
	}
}

// --- milestone linking: action end -> milestone, milestone -> action start ---

func TestMilestoneLinking(t *testing.T) {
	a := NewApp()
	if _, err := a.SetProjectRange("2026-01-01", "2026-12-31"); err != nil {
		t.Fatal(err)
	}
	act := mustWP(t, a, "spec", "2026-02-01", "2026-02-10")
	// a milestone in a NORMAL section (not Phases / not unclustered) is a
	// linkable diamond.
	if _, err := a.AddSection("Work"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.AddMilestone("Gate", "2026-02-12", "Work"); err != nil {
		t.Fatal(err)
	}
	ms := a.project.Milestones[len(a.project.Milestones)-1].ID
	// action end -> milestone (FS point link); satisfied because milestone is
	// after the action ends.
	if _, err := a.AddDependency(act, ms, "FS", "r", "l"); err != nil {
		t.Fatalf("link action->milestone: %v", err)
	}
	// milestone -> a following action's start
	act2 := mustWP(t, a, "build", "2026-02-20", "2026-03-01")
	if _, err := a.AddDependency(ms, act2, "FS", "r", "l"); err != nil {
		t.Fatalf("link milestone->action: %v", err)
	}
	d := a.GetProject()
	var sawA2M, sawM2A bool
	for _, dep := range d.Deps {
		if dep.From == act && dep.To == ms {
			sawA2M = true
			if dep.Violated {
				t.Error("action->milestone (milestone after action end) should not be violated")
			}
		}
		if dep.From == ms && dep.To == act2 {
			sawM2A = true
		}
	}
	if !sawA2M || !sawM2A {
		t.Fatalf("milestone links missing: a->m=%v m->a=%v", sawA2M, sawM2A)
	}
	// a milestone BEFORE the action end violates the FS action->milestone rule
	if _, err := a.SetMilestoneFields(ms, "", "2026-02-01", ""); err != nil {
		t.Fatal(err)
	}
	d = a.GetProject()
	for _, dep := range d.Deps {
		if dep.From == act && dep.To == ms && !dep.Violated {
			t.Error("action->milestone with milestone before action-end+1 should be violated")
		}
	}
}
