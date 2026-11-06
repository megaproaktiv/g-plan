package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/megaproaktiv/g-plan/src/app/plan"
)

func TestObjectHelpHasTheSameStructureInBothLanguages(t *testing.T) {
	en, de := objectHelp("en"), objectHelp("de")
	if len(en.Sections) != 3 || len(de.Sections) != 3 {
		t.Fatalf("sections: en %d, de %d, want 3 (sections, actions, contour)", len(en.Sections), len(de.Sections))
	}
	for i := range en.Sections {
		e, d := en.Sections[i], de.Sections[i]
		if e.Key != d.Key || len(e.Blocks) != len(d.Blocks) || e.Heading == d.Heading {
			t.Fatalf("section %s/%s differs in key, blocks or heading", e.Key, d.Key)
		}
		for j := range e.Blocks {
			eb, db := e.Blocks[j], d.Blocks[j]
			if eb.Kind != db.Kind || len(eb.Items) != len(db.Items) || len(eb.Rows) != len(db.Rows) || len(eb.Head) != len(db.Head) {
				t.Errorf("%s block %d differs in shape", e.Key, j)
			}
		}
	}
}

// Every contour the program knows is explained, and no contour is explained that does not exist.
func TestObjectHelpExplainsEveryContour(t *testing.T) {
	for _, lang := range []string{"en", "de"} {
		var rows [][]string
		for _, b := range sectionByKey(objectHelp(lang), "contour").Blocks {
			if b.Kind == "table" && len(b.Head) == 3 {
				rows = b.Rows
			}
		}
		var documented []string
		for _, r := range rows {
			documented = append(documented, strings.Trim(r[0], "*"))
		}
		known := []string{string(plan.ContourFlat), string(plan.ContourFront), string(plan.ContourBack), string(plan.ContourDouble), string(plan.ContourNone)}
		if fmt.Sprint(documented) != fmt.Sprint(known) {
			t.Errorf("%s: help explains %v, program has %v", lang, documented, known)
		}
	}
}

// The help says a section WITHOUT actions can be ordered: that must be true.
func TestObjectHelpClaimEmptySectionCanBeMoved(t *testing.T) {
	a := NewApp()
	for _, n := range []string{"A", "B", "C"} {
		if _, err := a.AddSection(n); err != nil {
			t.Fatal(err)
		}
	}
	before := fmt.Sprint(a.project.WBS().Sections)
	if _, err := a.MoveSection("C", -1); err != nil {
		t.Fatal(err)
	}
	if after := fmt.Sprint(a.project.WBS().Sections); after == before {
		t.Errorf("an empty section did not move: %s", after)
	}
	for _, lang := range []string{"en", "de"} {
		txt := helpText(objectHelp(lang))
		if !strings.Contains(txt, "ohne Aktionen") && !strings.Contains(txt, "without actions") {
			t.Errorf("%s help must say that sections without actions can be moved", lang)
		}
	}
}

// Names of buttons and menu entries used in the help exist on the screen.
func TestObjectHelpNamesRealControls(t *testing.T) {
	var src strings.Builder
	for _, f := range []string{"main.ts", "ctxmenu.ts"} {
		b, err := os.ReadFile(filepath.Join("frontend", "src", f))
		if err != nil {
			t.Fatal(err)
		}
		src.Write(b)
	}
	for _, label := range []string{"New Section", "Edit Action", "Move Section Up", "Move Section Down", "▲ up", "▼ down", "Show Progress", "External ID"} {
		if !strings.Contains(src.String(), label) {
			t.Errorf("the UI no longer has %q, but the help names it", label)
		}
		for _, lang := range []string{"en", "de"} {
			if !strings.Contains(helpText(objectHelp(lang)), label) && label != "Show Progress" && label != "Move Section Down" {
				t.Errorf("%s help should mention %q", lang, label)
			}
		}
	}
}

func objectGuideMarkdown(lang string) string {
	page := objectHelp(lang)
	return helpMarkdown(page, "src/app/objecthelp.go", "ObjectGuideDoc")
}

func TestObjectGuideDocIsUpToDate(t *testing.T) {
	for lang, file := range map[string]string{"en": "plan-objects.md", "de": "plan-objects.de.md"} {
		path := filepath.Join("..", "..", "documentation", "manual", file)
		want := objectGuideMarkdown(lang)
		if os.Getenv("UPDATE_GLOSSARY") == "1" {
			if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s missing: run UPDATE_GLOSSARY=1 go test . -run ObjectGuideDoc", file)
		}
		if string(got) != want {
			t.Errorf("%s is out of date: run UPDATE_GLOSSARY=1 go test . -run ObjectGuideDoc", file)
		}
	}
}
