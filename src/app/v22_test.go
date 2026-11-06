package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/megaproaktiv/g-plan/src/app/plan"
)

// --- Session 29: report / export config ---

func TestReportConfigDefaults(t *testing.T) {
	a := NewApp()
	d := a.GetProject()
	if d.ReportFormat != "pdf" || !d.ReportTodayLine || d.ReportGrid != "month" || !d.ReportShowNotes {
		t.Fatalf("report defaults wrong: %+v", map[string]any{
			"format": d.ReportFormat, "today": d.ReportTodayLine, "grid": d.ReportGrid, "notes": d.ReportShowNotes,
		})
	}
}

func TestSetReportConfigAndPersist(t *testing.T) {
	a := NewApp()
	a.SetReportConfig("png", false, "week", false)
	d := a.GetProject()
	if d.ReportFormat != "png" || d.ReportTodayLine || d.ReportGrid != "week" || d.ReportShowNotes {
		t.Fatalf("report config not applied: %+v", d)
	}
	// invalid format/grid normalize
	a.SetReportConfig("xls", true, "fortnight", true)
	d = a.GetProject()
	if d.ReportFormat != "pdf" || d.ReportGrid != "month" {
		t.Fatalf("invalid report values should normalize, got format=%q grid=%q", d.ReportFormat, d.ReportGrid)
	}
	// round-trip
	a.SetReportConfig("png", false, "day", false)
	dir := t.TempDir()
	path := filepath.Join(dir, "r"+plan.ProjectFileExt)
	if _, err := a.SaveProject(path); err != nil {
		t.Fatal(err)
	}
	b := NewApp()
	if _, err := b.OpenProject(path); err != nil {
		t.Fatal(err)
	}
	if b.project.ReportFormat != "png" || b.project.ReportTodayLine || b.project.ReportGrid != "day" || b.project.ReportShowNotes {
		t.Fatalf("report config did not round-trip: %+v", b.project)
	}
}

// TestOldFileKeepsReportDefaults verifies a project file predating the report
// keys keeps the NewProject defaults (pdf, today on, month, notes on).
func TestOldFileKeepsReportDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "old"+plan.ProjectFileExt)
	json := `{"schema":"` + plan.ProjectSchema + `","title":"Old","unit":"days","hoursPerDay":8,` +
		`"milestones":[],"phases":[],"packages":[],"deps":[],"roles":[]}`
	if err := os.WriteFile(path, []byte(json), 0o644); err != nil {
		t.Fatal(err)
	}
	a := NewApp()
	if _, err := a.OpenProject(path); err != nil {
		t.Fatal(err)
	}
	if a.project.ReportFormat != "pdf" || !a.project.ReportTodayLine || a.project.ReportGrid != "month" || !a.project.ReportShowNotes {
		t.Fatalf("old file should keep report defaults, got %+v", a.project)
	}
}

// TestWritePDFReportWithImage verifies the PDF builder accepts a caller image
// (the frontend-rendered PNG) and writes a non-empty file.
func TestWritePDFReportWithImage(t *testing.T) {
	a := NewApp()
	mustWP(t, a, "task", "2026-02-01", "2026-02-20")
	// a small real PNG (fpdf's parser rejects degenerate 1x1 streams)
	img := image.NewRGBA(image.Rect(0, 0, 400, 200))
	for x := 0; x < 400; x++ {
		for y := 0; y < 200; y++ {
			img.Set(x, y, color.RGBA{240, 240, 250, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	pngBytes := buf.Bytes()
	dir := t.TempDir()
	out := filepath.Join(dir, "report.pdf")
	if err := a.project.WritePDFReportWithImage(out, pngBytes); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Size() < 400 {
		t.Fatalf("pdf looks too small: %d bytes", fi.Size())
	}
}

// testPNG builds a small real PNG (fpdf's parser rejects degenerate 1x1 streams).
func testPNG(t *testing.T, r, g, b uint8) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 400, 200))
	for x := 0; x < 400; x++ {
		for y := 0; y < 200; y++ {
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// TestWritePDFReportWithViews verifies the report embeds the overview image PLUS
// one page per detail view (so the PDF has strictly more pages / bytes than the
// overview-only report built from the same image).
func TestWritePDFReportWithViews(t *testing.T) {
	a := NewApp()
	mustWP(t, a, "task", "2026-02-01", "2026-02-20")
	overview := testPNG(t, 240, 240, 250)
	views := []plan.ReportImage{
		{Caption: "1.1 Alpha", PNG: testPNG(t, 250, 240, 240)},
		{Caption: "1.2 Beta", PNG: testPNG(t, 240, 250, 240)},
	}

	dir := t.TempDir()
	withViews := filepath.Join(dir, "views.pdf")
	if err := a.project.WritePDFReportWithViews(withViews, overview, views); err != nil {
		t.Fatal(err)
	}
	overviewOnly := filepath.Join(dir, "overview.pdf")
	if err := a.project.WritePDFReportWithViews(overviewOnly, overview, nil); err != nil {
		t.Fatal(err)
	}

	biViews, err := os.Stat(withViews)
	if err != nil {
		t.Fatal(err)
	}
	biOverview, err := os.Stat(overviewOnly)
	if err != nil {
		t.Fatal(err)
	}
	if biViews.Size() < 400 {
		t.Fatalf("report with views too small: %d bytes", biViews.Size())
	}
	// The report with detail views carries extra Gantt pages, so it must be larger.
	if biViews.Size() <= biOverview.Size() {
		t.Fatalf("report with views (%d) should exceed overview-only report (%d)",
			biViews.Size(), biOverview.Size())
	}
}

// TestWritePDFReportWithViewsNoDetails verifies passing no detail views degrades
// to the overview-only behavior.
func TestWritePDFReportWithViewsNoDetails(t *testing.T) {
	a := NewApp()
	mustWP(t, a, "task", "2026-02-01", "2026-02-20")
	dir := t.TempDir()
	out := filepath.Join(dir, "report.pdf")
	if err := a.project.WritePDFReportWithViews(out, testPNG(t, 240, 240, 250), nil); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Size() < 400 {
		t.Fatalf("pdf looks too small: %d bytes", fi.Size())
	}
}

// TestNewProjectResetsCurrentSection verifies that New clears the per-session
// cursors (notably the last-used section) so a new project's actions land in
// the default lane instead of a stale section from the previous project.
func TestNewProjectResetsCurrentSection(t *testing.T) {
	a := NewApp()
	if _, err := a.AddSection("Mountain"); err != nil {
		t.Fatal(err)
	}
	mustWP(t, a, "dragon", "2026-01-01", "2026-01-10")
	if _, err := a.SetActionCluster("dragon", "Mountain"); err != nil {
		t.Fatal(err)
	}
	// SetActionCluster / Select set currentSection; adding here would land in Mountain.
	if _, err := a.Select("section", "Mountain"); err != nil {
		t.Fatal(err)
	}

	if _, err := a.NewProject(false); err != nil {
		t.Fatal(err)
	}
	if a.currentSection != "" || a.selKind != "" || a.selID != "" || a.drillID != "" || a.current != "" {
		t.Fatalf("New did not reset session cursors: section=%q selKind=%q selID=%q drill=%q current=%q",
			a.currentSection, a.selKind, a.selID, a.drillID, a.current)
	}
	// a fresh action must land in the default (empty) cluster
	mustWP(t, a, "breakfast", "2026-01-01", "2026-01-10")
	wp := a.project.WorkPackage("breakfast")
	if wp == nil {
		t.Fatal("breakfast not added")
	}
	if wp.Cluster != "" {
		t.Fatalf("new action landed in stale section %q, want default lane", wp.Cluster)
	}
}
