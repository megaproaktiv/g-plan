package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func helpText(p HelpPageDTO) string {
	var b strings.Builder
	b.WriteString(p.Intro + "\n")
	for _, s := range p.Sections {
		b.WriteString(s.Heading + "\n")
		for _, bl := range s.Blocks {
			b.WriteString(bl.Text + "\n" + strings.Join(bl.Items, "\n") + "\n" + strings.Join(bl.Head, "|") + "\n")
			for _, r := range bl.Rows {
				b.WriteString(strings.Join(r, "|") + "\n")
			}
		}
	}
	return b.String()
}

func blocksOfKind(s HelpSectionDTO, kind string) []HelpBlockDTO {
	var out []HelpBlockDTO
	for _, b := range s.Blocks {
		if b.Kind == kind {
			out = append(out, b)
		}
	}
	return out
}

func sectionByKey(p HelpPageDTO, key string) HelpSectionDTO {
	for _, s := range p.Sections {
		if s.Key == key {
			return s
		}
	}
	return HelpSectionDTO{}
}

func TestMonitoringHelpHasTheSameStructureInBothLanguages(t *testing.T) {
	en, de := monitoringHelp("en"), monitoringHelp("de")
	if len(en.Sections) != len(de.Sections) || len(en.Sections) < 4 {
		t.Fatalf("sections: en %d, de %d", len(en.Sections), len(de.Sections))
	}
	for i := range en.Sections {
		e, d := en.Sections[i], de.Sections[i]
		if e.Key != d.Key || len(e.Blocks) != len(d.Blocks) {
			t.Fatalf("section %s/%s: %d vs %d blocks", e.Key, d.Key, len(e.Blocks), len(d.Blocks))
		}
		if e.Heading == d.Heading {
			t.Errorf("%s: the heading is not translated", e.Key)
		}
		for j := range e.Blocks {
			eb, db := e.Blocks[j], d.Blocks[j]
			if eb.Kind != db.Kind || len(eb.Items) != len(db.Items) || len(eb.Rows) != len(db.Rows) || len(eb.Head) != len(db.Head) {
				t.Errorf("%s block %d differs in shape (%s %d/%d items, %d/%d rows)", e.Key, j, eb.Kind, len(eb.Items), len(db.Items), len(eb.Rows), len(db.Rows))
			}
			if eb.Kind != "table" && eb.Text == db.Text && eb.Text != "" {
				t.Errorf("%s block %d is not translated: %q", e.Key, j, eb.Text)
			}
		}
	}
	if en.Intro == de.Intro {
		t.Error("the intro is not translated")
	}
}

func TestMonitoringHelpTextIsWellFormed(t *testing.T) {
	for _, lang := range []string{"en", "de"} {
		page := monitoringHelp(lang)
		for _, s := range page.Sections {
			for _, b := range s.Blocks {
				texts := append([]string{b.Text}, b.Items...)
				texts = append(texts, b.Head...)
				for _, r := range b.Rows {
					texts = append(texts, r...)
				}
				for _, x := range texts {
					if strings.Count(x, "**")%2 != 0 || strings.Count(x, "`")%2 != 0 {
						t.Errorf("[%s/%s] unbalanced markup in %q", lang, s.Key, x)
					}
					if strings.Contains(x, "TODO") || strings.Contains(x, "%!") {
						t.Errorf("[%s/%s] unfinished text: %q", lang, s.Key, x)
					}
				}
				if (b.Kind == "p" || b.Kind == "note") && strings.TrimSpace(b.Text) == "" {
					t.Errorf("[%s/%s] empty %s block", lang, s.Key, b.Kind)
				}
				if b.Kind == "table" {
					for _, r := range b.Rows {
						if len(r) != len(b.Head) {
							t.Errorf("[%s/%s] row %v does not match the %d columns", lang, s.Key, r, len(b.Head))
						}
					}
				}
			}
		}
	}
}

// The table that explains how the External ID is found must say what the
// program really does. The expected answers are written down independently here.
func TestExternalIDMatchTableIsTrue(t *testing.T) {
	want := map[string]bool{
		"WP1 kickoff meeting": true, "wp1: design": true, "(WP1)": true, "design / WP1 / review": true,
		"WP12 review": false, "xWP1 review": false, "WP-1 kickoff": true, "WP 1 kickoff": false, "Daily standup": false,
	}
	for lang, yes := range map[string]string{"en": "yes", "de": "ja"} {
		tbl := blocksOfKind(sectionByKey(monitoringHelp(lang), "extid"), "table")
		if len(tbl) != 1 || len(tbl[0].Rows) != len(want) {
			t.Fatalf("[%s] expected one table with %d rows", lang, len(want))
		}
		for _, r := range tbl[0].Rows {
			text := strings.Trim(r[0], "`")
			exp, ok := want[text]
			if !ok {
				t.Fatalf("[%s] unexpected row %q", lang, text)
			}
			if got := r[1] == yes; got != exp {
				t.Errorf("[%s] %q: the help says found=%v, the program finds=%v", lang, text, got, exp)
			}
		}
	}
}

// The worked example: the numbers are checked against hand calculation.
//
//	AWS Landing Zone (WP2): 3 h + 2 h + half of 4 h = 7 h of 80 h  -> 8.7 %
//	ACC Core configuration (WP3): half of 4 h + 5 h = 7 h of 96 h  -> 7.2 %
//	Project Management (WP1, default): the 1 h without an ID of 320 h -> 0.3 %
//	7 h = 7/8 day = 0,88 days (rounded to two decimals), 1 h = 0,13 days
func TestWorkedExampleNumbers(t *testing.T) {
	for _, c := range []struct {
		lang, dec, days string
	}{{"en", ".", "days"}, {"de", ",", "Tage"}} {
		page := monitoringHelp(c.lang)
		steps := sectionByKey(page, "steps")
		tables := blocksOfKind(steps, "table")
		if len(tables) != 2 {
			t.Fatalf("[%s] the steps section has %d tables, want the bookings and the result", c.lang, len(tables))
		}
		// bookings -> counted for
		wantSplit := []string{"WP2 3 h", "WP2 2 h", "WP2 2 h, WP3 2 h", "WP3 5 h", ": 1 h"}
		for i, r := range tables[0].Rows {
			if !strings.Contains(r[2], wantSplit[i]) {
				t.Errorf("[%s] booking %q counted as %q, want it to contain %q", c.lang, r[0], r[2], wantSplit[i])
			}
		}
		if !strings.Contains(tables[0].Rows[4][2], "Project Management") {
			t.Errorf("[%s] the booking without an ID goes to the default action Project Management: %q", c.lang, tables[0].Rows[4][2])
		}
		// result table
		want := map[string][3]string{
			"AWS Landing Zone (WP2)":       {"10 " + c.days + " (80 h)", "0,88 " + c.days + " (7 h)", "8" + c.dec + "7%"},
			"ACC Core configuration (WP3)": {"12 " + c.days + " (96 h)", "0,88 " + c.days + " (7 h)", "7" + c.dec + "2%"},
			"Project Management (WP1)":     {"40 " + c.days + " (320 h)", "0,13 " + c.days + " (1 h)", "0" + c.dec + "3%"},
		}
		if len(tables[1].Rows) != 3 {
			t.Fatalf("[%s] result rows: %v", c.lang, tables[1].Rows)
		}
		for _, r := range tables[1].Rows {
			w, ok := want[r[0]]
			if !ok {
				t.Errorf("[%s] unexpected result row %q", c.lang, r[0])
				continue
			}
			if r[1] != w[0] || r[2] != w[1] || r[3] != w[2] {
				t.Errorf("[%s] %s: got %q / %q / %q, want %q / %q / %q", c.lang, r[0], r[1], r[2], r[3], w[0], w[1], w[2])
			}
		}
	}
}

func TestMonitoringStepsAreTenAndNameRealControls(t *testing.T) {
	en, de := monitoringHelp("en"), monitoringHelp("de")
	stepsEN := blocksOfKind(sectionByKey(en, "steps"), "ol")
	stepsDE := blocksOfKind(sectionByKey(de, "steps"), "ol")
	if len(stepsEN) != 1 || len(stepsEN[0].Items) != 10 || len(stepsDE[0].Items) != 10 {
		t.Fatalf("the guide has 10 numbered steps in both languages")
	}
	// the names of buttons and fields used in the help exist on the screen
	var src strings.Builder
	for _, f := range []string{"monitor.ts", "main.ts"} {
		b, err := os.ReadFile(filepath.Join("frontend", "src", f))
		if err != nil {
			t.Fatal(err)
		}
		src.Write(b)
	}
	allEN, allDE := helpText(en), helpText(de)
	for _, label := range []string{"Configure System…", "Synchronize", "Test", "Monitor", "External ID", "Default action",
		"Monitoring on", "Show Progress", "also one month", "Connected: project"} {
		if !strings.Contains(src.String(), label) {
			t.Errorf("the UI no longer has %q, but the help names it", label)
		}
		if !strings.Contains(allEN, label) || !strings.Contains(allDE, label) {
			t.Errorf("the help should mention %q in both languages", label)
		}
	}
	// the configuration file the help names is the one the program writes
	if !strings.Contains(allEN, "~/.pt-pm/config.yaml") {
		t.Error("the help should name the configuration file")
	}
}

func TestMonitoringHelpLabelsExist(t *testing.T) {
	en, de := guiStrings("en"), guiStrings("de")
	for _, k := range []string{"help.monitoring", "help.tab.monitoring"} {
		if en[k] == "" || de[k] == "" {
			t.Errorf("label %s missing", k)
		}
	}
	if en["help.tab.monitoring"] == de["help.tab.monitoring"] {
		t.Error("the tab title is not translated")
	}
}

// --- documentation generated from the help ---------------------------------

var mdEscapeBar = regexp.MustCompile(`\|`)

func monitoringGuideMarkdown(lang string) string {
	return helpMarkdown(monitoringHelp(lang), "src/app/monitoringhelp.go", "GuideDoc")
}

// helpMarkdown renders a help page as the generated documentation file.
func helpMarkdown(page HelpPageDTO, source, testName string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n> Generated from `%s` (the in-program help shows the same text). Do not edit by hand:\n> run `UPDATE_GLOSSARY=1 go test . -run %s` after changing the help.\n\n%s\n", page.Title, source, testName, page.Intro)
	for _, s := range page.Sections {
		fmt.Fprintf(&b, "\n## %s\n", s.Heading)
		for _, bl := range s.Blocks {
			switch bl.Kind {
			case "p":
				fmt.Fprintf(&b, "\n%s\n", bl.Text)
			case "note":
				fmt.Fprintf(&b, "\n> %s\n", bl.Text)
			case "ul":
				b.WriteString("\n")
				for _, it := range bl.Items {
					fmt.Fprintf(&b, "- %s\n", it)
				}
			case "ol":
				b.WriteString("\n")
				for i, it := range bl.Items {
					fmt.Fprintf(&b, "%d. %s\n", i+1, it)
				}
			case "table":
				cell := func(x string) string { return mdEscapeBar.ReplaceAllString(x, `\|`) }
				b.WriteString("\n| " + strings.Join(mapStrings(bl.Head, cell), " | ") + " |\n")
				b.WriteString("|" + strings.Repeat("---|", len(bl.Head)) + "\n")
				for _, r := range bl.Rows {
					b.WriteString("| " + strings.Join(mapStrings(r, cell), " | ") + " |\n")
				}
			}
		}
	}
	return b.String()
}

func mapStrings(in []string, f func(string) string) []string {
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = f(s)
	}
	return out
}

func TestMonitoringGuideDocIsUpToDate(t *testing.T) {
	for lang, file := range map[string]string{"en": "monitoring-guide.md", "de": "monitoring-guide.de.md"} {
		path := filepath.Join("..", "..", "documentation", "manual", file)
		want := monitoringGuideMarkdown(lang)
		if os.Getenv("UPDATE_GLOSSARY") == "1" {
			if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s missing: run UPDATE_GLOSSARY=1 go test . -run GuideDoc", file)
		}
		if string(got) != want {
			t.Errorf("%s is out of date: run UPDATE_GLOSSARY=1 go test . -run GuideDoc", file)
		}
	}
}
