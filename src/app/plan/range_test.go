package plan

import "testing"

func TestNewProjectHasStartEnd(t *testing.T) {
	p := NewProject("x", UnitDays)
	s, okS := p.Start()
	e, okE := p.End()
	if !okS || !okE {
		t.Fatal("new project must have Start and End milestones")
	}
	if !e.After(s) {
		t.Errorf("End %v must be after Start %v", e, s)
	}
	if len(p.Milestones) != 2 {
		t.Errorf("new project milestones = %d, want 2 (Start, End)", len(p.Milestones))
	}
	if p.Milestones[0].ID != MilestoneStartID {
		t.Errorf("first milestone id = %q, want %q", p.Milestones[0].ID, MilestoneStartID)
	}
}

func TestSetRangeUpdates(t *testing.T) {
	p := NewProject("x", UnitDays)
	s, _ := ParseDateStr("2026-01-01")
	e, _ := ParseDateStr("2026-06-30")
	p.SetRange(s, e)
	gs, _ := p.Start()
	ge, _ := p.End()
	if !gs.Equal(s) || !ge.Equal(e) {
		t.Errorf("range not applied: got %v..%v", gs, ge)
	}
	if len(p.Milestones) != 2 {
		t.Errorf("SetRange should not duplicate milestones, got %d", len(p.Milestones))
	}
}

func TestLoadedPlanKeepsOwnMilestones(t *testing.T) {
	p := NewProject("", UnitDays)
	if err := p.ReadMMD("../testdata/work-package-plan.mmd"); err != nil {
		t.Fatalf("ReadMMD: %v", err)
	}
	// testdata already has 4 milestones; EnsureRange must not add more.
	if len(p.Milestones) != 4 {
		t.Errorf("loaded plan milestones = %d, want 4 (unchanged)", len(p.Milestones))
	}
}
