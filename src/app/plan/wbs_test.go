package plan

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"pgregory.net/rapid"
)

func day(s string) time.Time { d, _ := ParseDateStr(s); return d }

// skynetLike mirrors the structure of the reported project: a Phases lane with
// five actions, three sections with one action each, and an EMPTY declared
// section that is numbered last.
func skynetLike() *Project {
	p := NewProject("Skynet", UnitDays)
	p.SetRange(day("2026-11-01"), day("2027-08-01"))
	add := func(id, name, cluster, start, end string) {
		p.Packages = append(p.Packages, WorkPackage{ID: id, Name: name, Cluster: cluster, Start: day(start), End: day(end), Mode: ModeDerived})
	}
	add("mob", "0 Mobilization", "Phases", "2026-11-07", "2027-01-29")
	add("found", "1 Foundation and pilot", "Phases", "2026-12-01", "2027-04-01")
	add("email", "2 Email and chat rollout", "Phases", "2027-05-02", "2027-07-01")
	add("voice", "3 Voice rollout", "Phases", "2027-05-15", "2027-06-30")
	add("zen", "4 Zendesk exit", "Phases", "2027-05-15", "2027-07-15")
	add("pm", "Project Management", "Section 1", "2026-11-04", "2027-07-31")
	add("aws", "AWS Landing Zone", "AWS landing zone", "2026-12-02", "2027-01-07")
	add("acc", "ACC Core configuration", "Connect core configuration", "2027-01-09", "2027-02-26")
	p.SectionColors = map[string]string{
		"AWS landing zone": "", "Additional Work Packages": "", "Connect core configuration": "", "Section 1": "",
	}
	return p
}

func TestWBSNumbersMatchTheScreen(t *testing.T) {
	w := skynetLike().WBS()
	wantSections := []string{"Section 1", "AWS landing zone", "Connect core configuration", "Additional Work Packages"}
	if !reflect.DeepEqual(w.Sections, wantSections) {
		t.Fatalf("sections %q, want %q", w.Sections, wantSections)
	}
	for name, n := range map[string]int{"Section 1": 1, "AWS landing zone": 2, "Connect core configuration": 3, "Additional Work Packages": 4} {
		if w.SectionNum[name] != n {
			t.Errorf("section %q is WP-%d, want WP-%d", name, w.SectionNum[name], n)
		}
	}
	if _, ok := w.SectionNum["Phases"]; ok {
		t.Error("the Phases lane has no WP number")
	}
	for id, want := range map[string]string{"pm": "1.1", "aws": "2.1", "acc": "3.1",
		"mob": "P1", "found": "P2", "email": "P3", "voice": "P4", "zen": "P5"} {
		if got := w.ActionDisplay[id]; got != want {
			t.Errorf("action %s is displayed %q, want %q", id, got, want)
		}
	}
	// the reported bug: "3.1" is the ACC action, "2.1" the AWS action
	if w.ActionByDisplay("3.1") != "acc" || w.ActionByDisplay("2.1") != "aws" || w.ActionByDisplay("1.1") != "pm" {
		t.Errorf("lookups: 3.1=%q 2.1=%q 1.1=%q", w.ActionByDisplay("3.1"), w.ActionByDisplay("2.1"), w.ActionByDisplay("1.1"))
	}
	if w.ActionByDisplay("p3") != "email" || w.ActionByDisplay(" P1 ") != "mob" {
		t.Error("Phases-lane numbers P1.. are found, case-insensitively")
	}
	if w.ActionByDisplay("4.1") != "" || w.ActionByDisplay("9.9") != "" || w.ActionByDisplay("x") != "" {
		t.Error("numbers that are not on the screen resolve to nothing")
	}
}

func TestWBSSharedSequenceWithMilestones(t *testing.T) {
	p := skynetLike()
	p.Milestones = append(p.Milestones, Milestone{ID: "m_go", Name: "Go-live", Date: day("2026-12-15"), Cluster: "AWS landing zone"})
	w := p.WBS()
	// in "AWS landing zone" the action starts 2026-12-02, the milestone is later
	if w.ActionDisplay["aws"] != "2.1" || w.MilestoneDisplay["m_go"] != "2.2" {
		t.Errorf("aws=%q m_go=%q", w.ActionDisplay["aws"], w.MilestoneDisplay["m_go"])
	}
	if w.MilestoneByDisplay("2.2") != "m_go" || w.MilestoneByDisplay("2.1") != "" {
		t.Error("a milestone code resolves to the milestone, an action code does not")
	}
	// the project Start/End milestones are not numbered
	if _, ok := w.MilestoneDisplay[MilestoneStartID]; ok {
		t.Error("m_start is not part of the numbering")
	}
}

func TestSectionOrderIncludesEmptySections(t *testing.T) {
	p := skynetLike()
	// an explicit order may place the EMPTY section anywhere
	p.SectionOrder = []string{"Section 1", "Additional Work Packages", "AWS landing zone", "Connect core configuration"}
	w := p.WBS()
	if w.SectionNum["Additional Work Packages"] != 2 || w.SectionNum["AWS landing zone"] != 3 || w.ActionDisplay["aws"] != "3.1" {
		t.Errorf("explicit order: %+v %q", w.SectionNum, w.ActionDisplay["aws"])
	}
	// names that no longer exist are ignored, sections it does not mention follow
	p.SectionOrder = []string{"Deleted", "Connect core configuration"}
	got := p.SectionNames()
	want := []string{"Connect core configuration", "Section 1", "AWS landing zone", "Additional Work Packages"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%q, want %q", got, want)
	}
	// duplicates in the stored order do not duplicate a section
	p.SectionOrder = []string{"Section 1", "Section 1", "AWS landing zone"}
	if n := len(p.SectionNames()); n != 4 {
		t.Errorf("%d sections, want 4", n)
	}
}

func TestSectionOrderIsSaved(t *testing.T) {
	p := skynetLike()
	p.SectionOrder = []string{"Additional Work Packages", "Section 1", "AWS landing zone", "Connect core configuration"}
	path := filepath.Join(t.TempDir(), "o.ptplan")
	if err := p.SaveJSON(path); err != nil {
		t.Fatal(err)
	}
	q, err := OpenJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(q.SectionOrder, p.SectionOrder) || q.WBS().SectionNum["Additional Work Packages"] != 1 {
		t.Errorf("order after reload: %q", q.SectionOrder)
	}
	// a file without the key keeps the natural order
	old := skynetLike()
	if err := old.SaveJSON(path); err != nil {
		t.Fatal(err)
	}
	o, _ := OpenJSON(path)
	if len(o.SectionOrder) != 0 || o.WBS().SectionNum["Section 1"] != 1 {
		t.Errorf("legacy file: %q", o.SectionOrder)
	}
}

// Feature: numbering - for any project the numbers are a bijection: every
// action has exactly one display id, every display id leads back to the same
// action, and the sections are numbered 1..N once each (Phases excluded).
func TestProp_WBSIsABijection(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		p := NewProject("x", UnitDays)
		p.SetRange(day("2026-01-01"), day("2026-12-31"))
		clusters := []string{"", "Phases", "A", "B", "C"}
		n := rapid.IntRange(0, 12).Draw(t, "n")
		for i := 0; i < n; i++ {
			start := day("2026-01-01").AddDate(0, 0, rapid.IntRange(0, 300).Draw(t, "d"))
			p.Packages = append(p.Packages, WorkPackage{
				ID: fmt.Sprintf("a%d", i), Name: "x", Start: start, End: start.AddDate(0, 0, 5), Mode: ModeDerived,
				Cluster: rapid.SampledFrom(clusters).Draw(t, "c"),
			})
		}
		if rapid.Bool().Draw(t, "declared") {
			p.SectionColors = map[string]string{"Empty1": "", "Empty2": ""}
		}
		p.SectionOrder = rapid.SliceOfN(rapid.SampledFrom(append(clusters, "Empty1", "Empty2", "Gone")), 0, 6).Draw(t, "order")
		w := p.WBS()
		seen := map[string]bool{}
		for _, wp := range p.Packages {
			d := w.ActionDisplay[wp.ID]
			if d == "" || seen[d] {
				t.Fatalf("action %s has display %q (duplicate or empty)", wp.ID, d)
			}
			seen[d] = true
			if got := w.ActionByDisplay(d); got != wp.ID {
				t.Fatalf("display %q leads to %q, want %q", d, got, wp.ID)
			}
		}
		for i, name := range w.Sections {
			if w.SectionNum[name] != i+1 || IsPhasesLane(name) {
				t.Fatalf("section %q numbered %d at position %d", name, w.SectionNum[name], i+1)
			}
		}
		if len(w.Sections) != len(w.SectionNum) {
			t.Fatalf("sections %q vs numbers %v", w.Sections, w.SectionNum)
		}
	})
}
