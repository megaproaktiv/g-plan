package main

import (
	"reflect"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

func TestTokenizeQuotes(t *testing.T) {
	want := []string{"add", "action", "2 Email and chat rollout", "from", "2027-05-01", "to", "2027-06-30"}
	for name, line := range map[string]string{
		"ascii":      `add action "2 Email and chat rollout" from 2027-05-01 to 2027-06-30`,
		"smart":      "add action \u201c2 Email and chat rollout\u201d from 2027-05-01 to 2027-06-30",
		"german":     "add action \u201e2 Email and chat rollout\u201c from 2027-05-01 to 2027-06-30",
		"guillemets": "add action \u00ab2 Email and chat rollout\u00bb from 2027-05-01 to 2027-06-30",
	} {
		if got := tokenize(line); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: %q, want %q", name, got, want)
		}
	}
	// apostrophes are not quotes
	if got := tokenize("add Peter's plan"); !reflect.DeepEqual(got, []string{"add", "Peter's", "plan"}) {
		t.Errorf("apostrophe: %q", got)
	}
	if got := tokenize("add Peter\u2019s plan"); !reflect.DeepEqual(got, []string{"add", "Peter\u2019s", "plan"}) {
		t.Errorf("typographic apostrophe: %q", got)
	}
}

// The command from the report: the whole quoted text is the NAME, never the id.
func TestAddActionKeepsTheWholeName(t *testing.T) {
	const name = "2 Email and chat rollout"
	for label, line := range map[string]string{
		"ascii quotes":       `add action "2 Email and chat rollout" from 2027-05-01 to 2027-06-30`,
		"smart quotes":       "add action \u201c2 Email and chat rollout\u201d from 2027-05-01 to 2027-06-30",
		"german quotes":      "add action \u201e2 Email and chat rollout\u201c from 2027-05-01 to 2027-06-30",
		"without any quotes": `add action 2 Email and chat rollout from 2027-05-01 to 2027-06-30`,
		"with an estimation": `add action "2 Email and chat rollout" from 2027-05-01 to 2027-06-30 est 5`,
	} {
		a := NewApp()
		out, err := a.Chat(line)
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		if len(a.project.Packages) != 1 {
			t.Fatalf("%s: %d actions", label, len(a.project.Packages))
		}
		w := a.project.Packages[0]
		if w.Name != name {
			t.Errorf("%s: name %q, want %q", label, w.Name, name)
		}
		if w.Start.Format("2006-01-02") != "2027-05-01" || w.End.Format("2006-01-02") != "2027-06-30" {
			t.Errorf("%s: dates %s..%s", label, w.Start.Format("2006-01-02"), w.End.Format("2006-01-02"))
		}
		if !strings.Contains(out, name) {
			t.Errorf("%s: the reply should show the name: %q", label, out)
		}
		if strings.Contains(label, "estimation") && w.Est == 0 {
			t.Errorf("%s: estimation not applied", label)
		}
	}
}

// Feature: add action - however a name made of words and numbers is quoted (or
// not), it is the name, and the dates after it are the dates.
func TestProp_AddActionNameRoundTrips(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		words := rapid.SliceOfN(rapid.StringMatching(`[A-Za-z0-9]{1,8}`).Filter(func(s string) bool {
			l := strings.ToLower(s)
			return l != "from" && l != "to" && l != "est" && l != "after"
		}), 1, 5).Draw(t, "words")
		name := strings.Join(words, " ")
		form := rapid.SampledFrom([]string{`"%s"`, "\u201c%s\u201d", "\u201e%s\u201c", "%s"}).Draw(t, "form")
		a := NewApp()
		line := "add action " + strings.Replace(form, "%s", name, 1) + " from 2027-05-01 to 2027-06-30"
		if _, err := a.Chat(line); err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		if got := a.project.Packages[0].Name; got != name {
			t.Fatalf("%q: name %q, want %q", line, got, name)
		}
	})
}

func TestActionIDIsNeverJustANumber(t *testing.T) {
	a := NewApp()
	if _, err := a.Chat(`add action "2024" from 2027-05-01 to 2027-06-30`); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Chat(`add action "7" from 2027-05-01 to 2027-06-30`); err != nil {
		t.Fatal(err)
	}
	for _, w := range a.project.Packages {
		if strings.Trim(w.ID, "0123456789") == "" {
			t.Errorf("action %q got the purely numeric id %q", w.Name, w.ID)
		}
	}
	// "2" still means the second action, not an id
	if got := a.resolveActionRef("2"); got != a.project.Packages[1].ID {
		t.Errorf("'2' = %q, want the second action", got)
	}
}

func TestAddActionInActionMode(t *testing.T) {
	a := NewApp()
	if _, err := a.Chat("mode action"); err != nil {
		t.Fatal(err)
	}
	out, err := a.Chat("add action Aktionname")
	if err != nil {
		t.Fatal(err)
	}
	if len(a.project.Packages) != 1 || a.project.Packages[0].Name != "Aktionname" || !strings.Contains(out, "Aktionname") {
		t.Fatalf("add Aktionname: %q %+v", out, a.project.Packages)
	}
	// the new action is current and selected, and it is part of the project the UI fetches next
	if a.selKind != "action" || a.selID != a.project.Packages[0].ID {
		t.Errorf("selection: %s %s", a.selKind, a.selID)
	}
	if d := a.GetProject(); len(d.Packages) != 1 || d.Packages[0].Name != "Aktionname" {
		t.Errorf("GetProject must return the new action right away: %+v", d.Packages)
	}
	// a quoted name that starts with a number is the whole name
	if _, err := a.Chat(`add action "2 Email and chat rollout"`); err != nil {
		t.Fatal(err)
	}
	if got := a.project.Packages[1].Name; got != "2 Email and chat rollout" {
		t.Errorf("name %q", got)
	}
	if _, err := a.Chat(`add action 3 Email and chat`); err != nil {
		t.Fatal(err)
	}
	if got := a.project.Packages[2].Name; got != "3 Email and chat" {
		t.Errorf("unquoted: %q", got)
	}
}

func TestAddActionWithoutDatesAndInOtherModes(t *testing.T) {
	a := NewApp()
	if _, err := a.Chat("mode section"); err != nil {
		t.Fatal(err)
	}
	// "add section" makes a section, "add action" an action - whatever the mode
	if _, err := a.Chat("add section Sprint"); err != nil {
		t.Fatal(err)
	}
	if len(a.project.Packages) != 0 {
		t.Fatalf("add section must not create an action")
	}
	if _, err := a.Chat(`add action "Sprint planning"`); err != nil {
		t.Fatal(err)
	}
	if len(a.project.Packages) != 1 || a.project.Packages[0].Name != "Sprint planning" {
		t.Fatalf("add action <name> without dates: %+v", a.project.Packages)
	}
	if a.mode != "section" {
		t.Errorf("the mode must not change, got %q", a.mode)
	}
	if _, err := a.Chat(`add action Plain name too`); err != nil || a.project.Packages[1].Name != "Plain name too" {
		t.Errorf("unquoted without dates: %v %+v", err, a.project.Packages)
	}
}

func TestAddActionErrorsAreClear(t *testing.T) {
	a := NewApp()
	if _, err := a.Chat(`add action from 2027-05-01 to 2027-06-30`); err == nil || !strings.Contains(err.Error(), "name") {
		t.Errorf("missing name: %v", err)
	}
	if len(a.project.Packages) != 0 {
		t.Error("a failed add must create nothing")
	}
	if _, err := a.Chat(`add action "X" from 2027-06-30 to 2027-05-01`); err == nil {
		t.Error("end before start must fail")
	}
	// a name containing the word "from" works when quoted
	if _, err := a.Chat(`add action "Back from holiday" from 2027-05-01 to 2027-05-10`); err != nil || a.project.Packages[0].Name != "Back from holiday" {
		t.Errorf("quoted keyword in name: %v %+v", err, a.project.Packages)
	}
}

// `add section <name>` creates a section and must leave the UI in
// section mode. The UI follows the backend selection, so a stale action
// selection from earlier must not survive the add.
func TestAddInSectionModeSelectsTheNewSection(t *testing.T) {
	a := NewApp()
	if _, err := a.AddWorkPackage("Existing", "2027-01-01", "2027-01-31"); err != nil {
		t.Fatal(err)
	}
	actionID := a.project.Packages[0].ID
	if _, err := a.Chat("mode section"); err != nil {
		t.Fatal(err)
	}
	for _, stale := range []struct{ kind, id string }{{"", ""}, {"action", actionID}, {"milestone", "m_start"}} {
		a.selKind, a.selID = stale.kind, stale.id
		if _, err := a.Chat("add section Rollout phase"); err != nil {
			t.Fatal(err)
		}
		sel := a.Selection()
		if sel.Kind != "section" || sel.ID != "Rollout phase" {
			t.Errorf("after the add (stale selection %q): selection is %q %q, want section \"Rollout phase\"", stale.kind, sel.Kind, sel.ID)
		}
		if a.mode != "section" {
			t.Errorf("the mode must stay section, got %q", a.mode)
		}
		if a.currentSection != "Rollout phase" {
			t.Errorf("the new section is the current section for new actions, got %q", a.currentSection)
		}
		// the next bare add of an ACTION lands in the new section
		a.Chat("add section Rollout phase") // same name again must not break anything
	}
	if _, ok := a.project.SectionColors["Rollout phase"]; !ok {
		t.Error("the section was not created")
	}
}

func TestAddInRoleModeDropsAStaleActionSelection(t *testing.T) {
	a := NewApp()
	if _, err := a.AddWorkPackage("Existing", "2027-01-01", "2027-01-31"); err != nil {
		t.Fatal(err)
	}
	a.selKind, a.selID = "action", a.project.Packages[0].ID
	if _, err := a.Chat("mode role"); err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{"add role Reviewer", `add role "Tech Lead"`} {
		a.selKind, a.selID = "action", a.project.Packages[0].ID
		if _, err := a.Chat(line); err != nil {
			t.Fatal(err)
		}
		if sel := a.Selection(); sel.Kind != "" {
			t.Errorf("%q left the selection %q %q: the UI would jump to action mode", line, sel.Kind, sel.ID)
		}
	}
	if a.project.RoleByName("Reviewer") == nil || a.project.RoleByName("Tech Lead") == nil {
		t.Error("the roles were not created")
	}
}

func TestAddInActionAndMilestoneModeSelectTheirOwnObject(t *testing.T) {
	a := NewApp()
	if _, err := a.Chat("add action First"); err != nil {
		t.Fatal(err)
	}
	if sel := a.Selection(); sel.Kind != "action" {
		t.Errorf("a new action is selected: %+v", sel)
	}
	if _, err := a.Chat("mode milestone"); err != nil {
		t.Fatal(err)
	}
	a.selKind, a.selID = "action", a.project.Packages[0].ID
	if _, err := a.Chat("add milestone Go-live"); err != nil {
		t.Fatal(err)
	}
	if sel := a.Selection(); sel.Kind != "milestone" {
		t.Errorf("a new milestone is selected, not the stale action: %+v", sel)
	}
}

// "add action <name> from <d> to <d>" with a quoted or unquoted name: the name
// ends at the first reserved word, the dates are used.
func TestAddActionNameEndsAtReservedWords(t *testing.T) {
	a := NewApp()
	for _, line := range []string{
		`add action "Scheduled Callback" from 2026-11-06 to 2026-11-10`,
		`add action Scheduled Callback from 2026-11-06 to 2026-11-10`,
		`add action „Scheduled Callback“ from 2026-11-06 to 2026-11-10`,
		`add wp "Scheduled Callback" from 2026-11-06 to 2026-11-10`,
	} {
		n := len(a.project.Packages)
		if _, err := a.Chat(line); err != nil {
			t.Fatalf("%s: %v", line, err)
		}
		w := a.project.Packages[n]
		if w.Name != "Scheduled Callback" || w.Start.Format("2006-01-02") != "2026-11-06" || w.End.Format("2006-01-02") != "2026-11-10" {
			t.Errorf("%s -> %q %s..%s", line, w.Name, w.Start.Format("2006-01-02"), w.End.Format("2006-01-02"))
		}
	}
	// only a start: one week; with est
	if _, err := a.Chat(`add action Review from 2026-12-01 est 3`); err != nil {
		t.Fatal(err)
	}
	w := a.project.Packages[len(a.project.Packages)-1]
	if w.Name != "Review" || w.End.Format("2006-01-02") != "2026-12-08" {
		t.Errorf("start only: %q %s", w.Name, w.End.Format("2006-01-02"))
	}
	// a bad date is an error and creates nothing
	n := len(a.project.Packages)
	if _, err := a.Chat(`add action Bad from 2026-13-40`); err == nil || len(a.project.Packages) != n {
		t.Errorf("bad date: %v", err)
	}
}

// The short form "add <name>" is gone in every mode: it was mixed up too often.
func TestBareAddIsRejected(t *testing.T) {
	for _, mode := range []string{"action", "section", "role", "milestone"} {
		a := NewApp()
		a.Chat("mode " + mode)
		for _, line := range []string{`add Demo`, `add "Scheduled Callback" from 2026-11-06 to 2026-11-10`, `add`} {
			_, err := a.Chat(line)
			if err == nil || !strings.Contains(err.Error(), "add action <name>") {
				t.Errorf("mode %s, %q: want a hint to the full form, got %v", mode, line, err)
			}
		}
		if len(a.project.Packages) != 0 || len(a.project.SectionColors) != 0 || len(a.project.Roles) != 0 {
			t.Errorf("mode %s: a rejected add created something", mode)
		}
	}
}

func TestAddDetailToAction(t *testing.T) {
	a := NewApp()
	a.Chat(`add action Parent from 2026-01-01 to 2026-03-01`)
	if _, err := a.Chat(`add detail "Design" to 1.1 from 2026-01-10 to 2026-01-20 est 2`); err != nil {
		t.Fatal(err)
	}
	d := a.project.Packages[0].Details
	if len(d) != 1 || d[0].Name != "Design" || d[0].End.Format("2006-01-02") != "2026-01-20" {
		t.Fatalf("details: %+v", d)
	}
}
