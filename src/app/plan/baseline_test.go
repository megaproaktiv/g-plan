package plan

import (
	"path/filepath"
	"testing"
)

func TestBaselineCaptureComparePersist(t *testing.T) {
	p := NewProject("D", UnitDays)
	s, _ := ParseDateStr("2026-01-01")
	e, _ := ParseDateStr("2026-01-31")
	p.Packages = []WorkPackage{
		{ID: "wp1", Name: "A", Start: s, End: e, Mode: ModeFixed, Est: p.BaseFromWholeDisplay(10)},
	}
	// capture a baseline
	b := p.CaptureBaseline("v1")
	p.Baselines = append(p.Baselines, b)
	if len(b.Tasks) != 1 || b.TotalEst != p.BaseFromWholeDisplay(10) {
		t.Fatalf("baseline capture wrong: %+v", b)
	}

	// change the live plan: slip end by 5 days, grow estimate to 15
	p.Packages[0].End = e.AddDate(0, 0, 5)
	p.Packages[0].Est = p.BaseFromWholeDisplay(15)

	v := p.CompareBaseline(&p.Baselines[0])
	if len(v) != 1 {
		t.Fatalf("variance rows = %d, want 1", len(v))
	}
	if v[0].EndDays != 5 {
		t.Errorf("endDays = %d, want 5", v[0].EndDays)
	}
	if v[0].EstDelta != p.BaseFromWholeDisplay(5) {
		t.Errorf("estDelta = %d, want %d", v[0].EstDelta, p.BaseFromWholeDisplay(5))
	}

	// persist + reload keeps the baseline frozen
	dir := t.TempDir()
	base := filepath.Join(dir, "d.mmd")
	if _, err := p.Save(base); err != nil {
		t.Fatalf("Save: %v", err)
	}
	q, _, err := Open(base)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if len(q.Baselines) != 1 || q.Baselines[0].Name != "v1" {
		t.Fatalf("baseline not persisted: %+v", q.Baselines)
	}
	if q.Baselines[0].Tasks[0].Est != p.BaseFromWholeDisplay(10) {
		t.Errorf("baseline est changed on reload: %d", q.Baselines[0].Tasks[0].Est)
	}
}
