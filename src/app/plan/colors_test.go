package plan

import (
	"path/filepath"
	"testing"
)

func TestColorsAndAnchorsRoundTrip(t *testing.T) {
	p := NewProject("D", UnitDays)
	s, _ := ParseDateStr("2026-01-01")
	e, _ := ParseDateStr("2026-02-01")
	p.Packages = []WorkPackage{
		{ID: "demo", Name: "demo", Desc: "d1", Color: "#ff0000", Start: s, End: e, Mode: ModeDerived},
		{ID: "ship", Name: "ship", Start: s, End: e, Mode: ModeDerived},
	}
	p.Deps = []Dependency{{From: "demo", To: "ship", Kind: "FS", FromAnchor: "b", ToAnchor: "t", Color: "#00aa00"}}
	p.SectionColors = map[string]string{"demo": "#123456"}

	dir := t.TempDir()
	base := filepath.Join(dir, "d.mmd")
	if _, err := p.Save(base); err != nil {
		t.Fatalf("Save: %v", err)
	}
	q, _, err := Open(base)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	wp := q.WorkPackage("demo")
	if wp.Color != "#ff0000" {
		t.Errorf("action color lost: %q", wp.Color)
	}
	if wp.Desc != "d1" {
		t.Errorf("action desc lost: %q", wp.Desc)
	}
	if len(q.Deps) != 1 || q.Deps[0].Color != "#00aa00" || q.Deps[0].FromAnchor != "b" || q.Deps[0].ToAnchor != "t" {
		t.Errorf("dep color/anchors lost: %+v", q.Deps)
	}
	if q.SectionColors["demo"] != "#123456" {
		t.Errorf("section color lost: %v", q.SectionColors)
	}
}
