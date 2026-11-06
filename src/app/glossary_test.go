package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The terms the project owner asked to define (keys).
var requiredGlossaryKeys = []string{
	"section", "workpackage", "action", "role", "milestone", "estimation", "work",
	"duration", "durationcomplete", "workcomplete", "taskcomplete",
}

func TestGlossaryHasEveryRequiredTermInBothLanguages(t *testing.T) {
	en, de := glossaryTerms("en"), glossaryTerms("de")
	if len(en) != len(de) {
		t.Fatalf("en has %d terms, de has %d", len(en), len(de))
	}
	for i := range en {
		if en[i].Key != de[i].Key || en[i].Group != de[i].Group {
			t.Errorf("entry %d differs in key/group: %s/%s vs %s/%s", i, en[i].Key, en[i].Group, de[i].Key, de[i].Group)
		}
	}
	have := map[string]bool{}
	for _, e := range en {
		have[e.Key] = true
	}
	for _, k := range requiredGlossaryKeys {
		if !have[k] {
			t.Errorf("required term %q missing", k)
		}
	}
	// the required terms come first, in the requested order, as the core group
	for i, k := range requiredGlossaryKeys {
		if en[i].Key != k || en[i].Group != GlossaryCore {
			t.Errorf("position %d: %s (%s), want %s (core)", i, en[i].Key, en[i].Group, k)
		}
	}
}

func TestGlossaryEntriesAreCompleteAndTranslated(t *testing.T) {
	en, de := glossaryTerms("en"), glossaryTerms("de")
	seen := map[string]bool{}
	for i := range en {
		for _, e := range []GlossaryTermDTO{en[i], de[i]} {
			if strings.TrimSpace(e.Term) == "" || len(strings.TrimSpace(e.Definition)) < 20 {
				t.Errorf("%s: term/definition missing: %+v", e.Key, e)
			}
		}
		if seen[en[i].Key] {
			t.Errorf("duplicate key %s", en[i].Key)
		}
		seen[en[i].Key] = true
		if en[i].Definition == de[i].Definition || en[i].Term == de[i].Term {
			t.Errorf("%s: the German entry is not translated", en[i].Key)
		}
		if (en[i].Example == "") != (de[i].Example == "") || (en[i].Note == "") != (de[i].Note == "") {
			t.Errorf("%s: example/note present in only one language", en[i].Key)
		}
		// the German term keeps the English one in brackets so it can be found in the (mostly English) GUI
		if !strings.Contains(de[i].Term, "(") && en[i].Key != "role" {
			t.Errorf("%s: German term %q should carry the English term in brackets", en[i].Key, de[i].Term)
		}
	}
	if got := glossaryTerms("fr"); got[0].Term != en[0].Term {
		t.Error("an unknown language falls back to English")
	}
}

// The definitions must say what the project owner defined (guards against edits
// that quietly change the meaning).
func TestGlossaryDefinitionsKeepTheirMeaning(t *testing.T) {
	by := map[string]GlossaryTermDTO{}
	for _, e := range glossaryTerms("en") {
		by[e.Key] = e
	}
	// Section and Work Package are the same thing and name each other.
	if !strings.Contains(by["section"].AlsoCalled, "Work Package") || !strings.Contains(by["workpackage"].AlsoCalled, "Section") {
		t.Error("Section and Work Package must name each other as synonyms")
	}
	if !strings.Contains(by["workpackage"].Definition, "contains actions") {
		t.Error("a work package contains actions")
	}
	if !strings.Contains(by["milestone"].AlsoCalled, "Gateway") {
		t.Error("milestones are also called gateways")
	}
	// Work Complete is a percentage of the estimation (not an hours count), with the worked example.
	wc := by["workcomplete"]
	if !strings.Contains(wc.AlsoCalled, "Effort Progress") || !strings.Contains(wc.Definition, "percentage of the estimation") ||
		!strings.Contains(wc.Example, "32 / 80 = 40 %") {
		t.Errorf("work complete: %+v", wc)
	}
	// Duration Complete is the elapsed share of the duration, with the worked example.
	dc := by["durationcomplete"]
	if !strings.Contains(dc.AlsoCalled, "Duration Progress") || !strings.Contains(dc.Definition, "percentage of the duration") ||
		!strings.Contains(dc.Example, "2 / 10 = 20 %") {
		t.Errorf("duration complete: %+v", dc)
	}
	// Task Complete is a definition only.
	if !strings.Contains(by["taskcomplete"].Definition, "not shown in the program") || !strings.Contains(by["taskcomplete"].Definition, "tested against its requirements") {
		t.Errorf("task complete: %+v", by["taskcomplete"])
	}
	// Work is hours or days; Estimation is planned work; Duration differs from the estimation.
	if !strings.Contains(by["work"].Definition, "hours or days") || !strings.Contains(by["estimation"].Definition, "planned amount of work") ||
		!strings.Contains(by["duration"].Definition, "usually differs from the estimation") {
		t.Error("work / estimation / duration definitions changed meaning")
	}
}

func TestHelpLabelsExistInBothLanguages(t *testing.T) {
	en, de := guiStrings("en"), guiStrings("de")
	for _, k := range []string{"help.glossary", "help.tab.glossary", "glossary.intro", "glossary.group.core",
		"glossary.group.screen", "glossary.also", "glossary.example", "glossary.note"} {
		if en[k] == "" || de[k] == "" {
			t.Errorf("label %s missing", k)
		}
		if en[k] == de[k] {
			t.Errorf("label %s is not translated: %q", k, de[k])
		}
	}
}

// glossaryMarkdown renders the glossary as the documentation page.
func glossaryMarkdown(lang string) string {
	t := guiStrings(lang)
	title := map[string]string{"en": "Glossary", "de": "Glossar"}[lang]
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n> Generated from `src/app/glossary.go` (the in-program help shows the same text). Do not edit by hand:\n> run `UPDATE_GLOSSARY=1 go test ./... -run GlossaryDoc` after changing the glossary.\n\n%s\n", title, t["glossary.intro"])
	for _, g := range []string{GlossaryCore, GlossaryScreen} {
		fmt.Fprintf(&b, "\n## %s\n", t["glossary.group."+g])
		for _, e := range glossaryTerms(lang) {
			if e.Group != g {
				continue
			}
			fmt.Fprintf(&b, "\n### %s\n\n", e.Term)
			if e.AlsoCalled != "" {
				fmt.Fprintf(&b, "*%s:* %s\n\n", t["glossary.also"], e.AlsoCalled)
			}
			fmt.Fprintf(&b, "%s\n", e.Definition)
			if e.Example != "" {
				fmt.Fprintf(&b, "\n*%s:* %s\n", t["glossary.example"], e.Example)
			}
			if e.Note != "" {
				fmt.Fprintf(&b, "\n*%s:* %s\n", t["glossary.note"], e.Note)
			}
		}
	}
	return b.String()
}

// The documentation pages are generated from the glossary, so they cannot drift.
func TestGlossaryDocIsUpToDate(t *testing.T) {
	for lang, file := range map[string]string{"en": "glossary.md", "de": "glossary.de.md"} {
		path := filepath.Join("..", "..", "documentation", "manual", file)
		want := glossaryMarkdown(lang)
		if os.Getenv("UPDATE_GLOSSARY") == "1" {
			if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s missing: run UPDATE_GLOSSARY=1 go test . -run GlossaryDoc", file)
		}
		if string(got) != want {
			t.Errorf("%s is out of date: run UPDATE_GLOSSARY=1 go test . -run GlossaryDoc", file)
		}
	}
}
