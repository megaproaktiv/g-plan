package plan

import (
	"os"
	"path/filepath"
	"testing"
)

// TestEndToEnd exercises the full core pipeline against real anonymized data:
// parse the .mmd, attach roles, render a PNG via embedded gmaid, build a PDF
// report, and save the three-file set.
func TestEndToEnd(t *testing.T) {
	p := NewProject("", UnitDays)
	if err := p.ReadMMD("../testdata/plan.mmd"); err != nil {
		t.Fatalf("ReadMMD: %v", err)
	}
	// Attach a couple of roles with assignments (what the .roles.json holds).
	p.Roles = []Role{
		{Name: "Ian McKellen", Description: "Project lead", Assignments: []Assignment{{WorkPackageID: "wp1", Percent: 100}}},
		{Name: "Orlando Bloom", Description: "Engineer", Assignments: []Assignment{
			{WorkPackageID: "wp2", Percent: 50}, {WorkPackageID: "wp4", Percent: 28},
		}},
	}

	dir := t.TempDir()

	// PNG render via embedded gmaid.
	png := filepath.Join(dir, "plan.png")
	if err := p.RenderPNG(png, "", 2); err != nil {
		t.Fatalf("RenderPNG: %v", err)
	}
	if fi, err := os.Stat(png); err != nil || fi.Size() < 1000 {
		t.Fatalf("rendered PNG missing or too small: err=%v", err)
	}

	// Combined PDF report (gantt image + people tables).
	pdf := filepath.Join(dir, "report.pdf")
	if err := p.WritePDFReport(pdf, ""); err != nil {
		t.Fatalf("WritePDFReport: %v", err)
	}
	if fi, err := os.Stat(pdf); err != nil || fi.Size() < 2000 {
		t.Fatalf("PDF report missing or too small: err=%v", err)
	}

	// Save the three-file set and reopen.
	base := filepath.Join(dir, "myplan.mmd")
	paths, err := p.Save(base)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	for _, f := range []string{paths.MMD, paths.Roles, paths.MD} {
		if _, err := os.Stat(f); err != nil {
			t.Fatalf("expected saved file %s: %v", f, err)
		}
	}
	// Reopen and verify roundtrip of packages + roles.
	p2, _, err := Open(base)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if len(p2.Packages) != len(p.Packages) {
		t.Errorf("reopened packages = %d, want %d", len(p2.Packages), len(p.Packages))
	}
	if len(p2.Roles) != 2 {
		t.Errorf("reopened roles = %d, want 2", len(p2.Roles))
	}
}
