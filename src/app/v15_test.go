package main

import "testing"

// --- Session 11: detail reorder + graphical drill linking support ---

// TestMoveDetailOrder verifies a detail sub-action can be reordered within its
// parent work package (the backend behind the detail panel's ▲/▼ controls),
// mirroring MoveAction for level-1 actions.
func TestMoveDetailOrder(t *testing.T) {
	a := NewApp()
	if _, err := a.SetProjectRange("2026-01-01", "2026-12-31"); err != nil {
		t.Fatal(err)
	}
	wp := mustWP(t, a, "Build", "2026-02-01", "2026-02-28")
	if _, err := a.AddDetail(wp, "spec", "2026-02-01", "2026-02-05", "0", "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := a.AddDetail(wp, "code", "2026-02-06", "2026-02-20", "0", "", ""); err != nil {
		t.Fatal(err)
	}
	p := a.project.WorkPackage(wp)
	if len(p.Details) != 2 {
		t.Fatalf("expected 2 details, got %d", len(p.Details))
	}
	spec, code := p.Details[0].ID, p.Details[1].ID
	// move "code" up -> it should come first
	if _, err := a.MoveDetailOrder(wp, code, -1); err != nil {
		t.Fatal(err)
	}
	p = a.project.WorkPackage(wp)
	if p.Details[0].ID != code || p.Details[1].ID != spec {
		t.Fatalf("after move up, order should be [code, spec], got [%s, %s]", p.Details[0].ID, p.Details[1].ID)
	}
	// move "code" down -> back to [spec, code]
	if _, err := a.MoveDetailOrder(wp, code, 1); err != nil {
		t.Fatal(err)
	}
	p = a.project.WorkPackage(wp)
	if p.Details[0].ID != spec || p.Details[1].ID != code {
		t.Fatalf("after move down, order should be [spec, code], got [%s, %s]", p.Details[0].ID, p.Details[1].ID)
	}
	// moving the top detail up again is a no-op
	if _, err := a.MoveDetailOrder(wp, spec, -1); err != nil {
		t.Fatal(err)
	}
	p = a.project.WorkPackage(wp)
	if p.Details[0].ID != spec {
		t.Fatal("moving the top detail up should be a no-op")
	}
}

func TestMoveDetailOrderErrors(t *testing.T) {
	a := NewApp()
	if _, err := a.MoveDetailOrder("nope", "x", -1); err == nil {
		t.Fatal("expected error for missing work package")
	}
	wp := mustWP(t, a, "Build", "2026-02-01", "2026-02-28")
	if _, err := a.MoveDetailOrder(wp, "ghost", 1); err == nil {
		t.Fatal("expected error for missing detail")
	}
}

// TestDetailToDetailLinkAnchors verifies a detail->detail dependency with
// explicit anchors (what the new graphical drill-plane anchor-drag emits) is
// accepted and round-trips through the DTO with its anchors, so the drill
// plane can render SS/FF/OV links between detail bars.
func TestDetailToDetailLinkAnchors(t *testing.T) {
	a := NewApp()
	if _, err := a.SetProjectRange("2026-01-01", "2026-12-31"); err != nil {
		t.Fatal(err)
	}
	wp := mustWP(t, a, "Build", "2026-02-01", "2026-02-28")
	if _, err := a.AddDetail(wp, "spec", "2026-02-01", "2026-02-05", "0", "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := a.AddDetail(wp, "code", "2026-02-06", "2026-02-20", "0", "", ""); err != nil {
		t.Fatal(err)
	}
	p := a.project.WorkPackage(wp)
	spec, code := p.Details[0].ID, p.Details[1].ID
	// a start-to-start link (left/left anchors) between the two details
	if _, err := a.AddDependency(spec, code, "SS", "l", "l"); err != nil {
		t.Fatal(err)
	}
	d := a.GetProject()
	var found bool
	for _, dep := range d.Deps {
		if dep.From == spec && dep.To == code {
			found = true
			if dep.Kind != "SS" {
				t.Fatalf("expected kind SS, got %q", dep.Kind)
			}
			if dep.FromAnchor != "l" || dep.ToAnchor != "l" {
				t.Fatalf("expected anchors l/l, got %s/%s", dep.FromAnchor, dep.ToAnchor)
			}
		}
	}
	if !found {
		t.Fatal("detail->detail dependency not found in DTO")
	}
}

// TestActionAndDetailFontColor verifies the font-color setters persist on the
// model and surface in the DTO (so the editor's Text-color control works), and
// that an empty value clears it (auto-contrast).
func TestActionAndDetailFontColor(t *testing.T) {
	a := NewApp()
	if _, err := a.SetProjectRange("2026-01-01", "2026-12-31"); err != nil {
		t.Fatal(err)
	}
	wp := mustWP(t, a, "Build", "2026-02-01", "2026-02-28")
	if _, err := a.SetActionFontColor(wp, "#000000"); err != nil {
		t.Fatal(err)
	}
	if got := a.project.WorkPackage(wp).FontColor; got != "#000000" {
		t.Fatalf("action font color: want #000000, got %q", got)
	}
	d := a.GetProject()
	if dtoWP(d, wp).FontColor != "#000000" {
		t.Fatalf("DTO action font color not surfaced: %q", dtoWP(d, wp).FontColor)
	}
	// clear -> auto-contrast
	if _, err := a.SetActionFontColor(wp, ""); err != nil {
		t.Fatal(err)
	}
	if got := a.project.WorkPackage(wp).FontColor; got != "" {
		t.Fatalf("action font color should clear, got %q", got)
	}

	if _, err := a.AddDetail(wp, "spec", "2026-02-01", "2026-02-05", "0", "", ""); err != nil {
		t.Fatal(err)
	}
	det := a.project.WorkPackage(wp).Details[0].ID
	if _, err := a.SetDetailFontColor(wp, det, "#ffffff"); err != nil {
		t.Fatal(err)
	}
	if got := a.project.WorkPackage(wp).Details[0].FontColor; got != "#ffffff" {
		t.Fatalf("detail font color: want #ffffff, got %q", got)
	}
}

func TestFontColorSettersErrors(t *testing.T) {
	a := NewApp()
	if _, err := a.SetActionFontColor("nope", "#000000"); err == nil {
		t.Fatal("expected error for missing action")
	}
	if _, err := a.SetDetailFontColor("nope", "x", "#000000"); err == nil {
		t.Fatal("expected error for missing work package")
	}
	wp := mustWP(t, a, "Build", "2026-02-01", "2026-02-28")
	if _, err := a.SetDetailFontColor(wp, "ghost", "#000000"); err == nil {
		t.Fatal("expected error for missing detail")
	}
}

// TestActionAndDetailTextPos verifies the text-position setters persist and
// surface in the DTO, that "auto" normalizes to "" (auto-placement), and that
// an invalid value is rejected.
func TestActionAndDetailTextPos(t *testing.T) {
	a := NewApp()
	if _, err := a.SetProjectRange("2026-01-01", "2026-12-31"); err != nil {
		t.Fatal(err)
	}
	wp := mustWP(t, a, "Build", "2026-02-01", "2026-02-28")
	if _, err := a.SetActionTextPos(wp, "right"); err != nil {
		t.Fatal(err)
	}
	if got := a.project.WorkPackage(wp).TextPos; got != "right" {
		t.Fatalf("action text pos: want right, got %q", got)
	}
	if dtoWP(a.GetProject(), wp).TextPos != "right" {
		t.Fatal("DTO action text pos not surfaced")
	}
	// "auto" normalizes to "" (auto-placement)
	if _, err := a.SetActionTextPos(wp, "auto"); err != nil {
		t.Fatal(err)
	}
	if got := a.project.WorkPackage(wp).TextPos; got != "" {
		t.Fatalf("auto should normalize to empty, got %q", got)
	}
	// invalid value rejected
	if _, err := a.SetActionTextPos(wp, "diagonal"); err == nil {
		t.Fatal("expected error for invalid text position")
	}

	if _, err := a.AddDetail(wp, "spec", "2026-02-01", "2026-02-05", "0", "", ""); err != nil {
		t.Fatal(err)
	}
	det := a.project.WorkPackage(wp).Details[0].ID
	if _, err := a.SetDetailTextPos(wp, det, "center"); err != nil {
		t.Fatal(err)
	}
	if got := a.project.WorkPackage(wp).Details[0].TextPos; got != "center" {
		t.Fatalf("detail text pos: want center, got %q", got)
	}
	if _, err := a.SetDetailTextPos(wp, det, "nope"); err == nil {
		t.Fatal("expected error for invalid detail text position")
	}
}

// TestOutsideTextPositions verifies the new outside-left/right positions are
// accepted and persisted (outleft ends at the bar's left edge; outright/right
// sit to the right of the bar).
func TestOutsideTextPositions(t *testing.T) {
	a := NewApp()
	wp := mustWP(t, a, "Build", "2026-02-01", "2026-02-28")
	for _, pos := range []string{"outleft", "outright", "right"} {
		if _, err := a.SetActionTextPos(wp, pos); err != nil {
			t.Fatalf("pos %q should be valid: %v", pos, err)
		}
		if got := a.project.WorkPackage(wp).TextPos; got != pos {
			t.Fatalf("text pos %q not stored, got %q", pos, got)
		}
	}
}
