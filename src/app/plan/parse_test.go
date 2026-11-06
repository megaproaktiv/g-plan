package plan

import (
	"strings"
	"testing"
)

func TestParseWorkPackagePlan(t *testing.T) {
	p := NewProject("", UnitDays)
	if err := p.ReadMMD("../testdata/work-package-plan.mmd"); err != nil {
		t.Fatalf("ReadMMD: %v", err)
	}
	if !strings.Contains(p.Title, "Rohan") {
		t.Errorf("title = %q, want it to contain Rohan", p.Title)
	}
	if len(p.Milestones) != 4 {
		t.Errorf("milestones = %d, want 4", len(p.Milestones))
	}
	if len(p.Packages) != 13 {
		t.Errorf("packages = %d, want 13", len(p.Packages))
	}
	// WP1 should be Frodo and span the whole project.
	wp1 := p.WorkPackage("wp1")
	if wp1 == nil {
		t.Fatal("wp1 not found")
	}
	if wp1.Name != "Frodo" {
		t.Errorf("wp1 name = %q, want Frodo", wp1.Name)
	}
	// Dependency wp2 FS wp3 present.
	found := false
	for _, d := range p.Deps {
		if d.From == "wp2" && d.To == "wp3" && d.Kind == "FS" {
			found = true
		}
	}
	if !found {
		t.Errorf("missing dependency wp2 FS wp3; got %+v", p.Deps)
	}
}

func TestParsePlanWithDetails(t *testing.T) {
	p := NewProject("", UnitDays)
	if err := p.ReadMMD("../testdata/wp4-email.mmd"); err != nil {
		t.Fatalf("ReadMMD: %v", err)
	}
	// wp4 should have detail packages (Implementation/Tests/Stabilization + role bars).
	wp4 := p.WorkPackage("wp4")
	if wp4 == nil {
		t.Fatal("wp4 not found")
	}
	if len(wp4.Details) == 0 {
		t.Errorf("wp4 has no detail packages")
	}
}
