package main

import (
	"fmt"
	"testing"
)

// --- Session 25: `select <n>` in section mode matches the UI lane number ---

// TestSelectSectionByNumberMatchesUI verifies that the numeric ref used by the
// chat `select <n>` command resolves to the SAME section the UI labels as n
// (the DTO SectionNums). Previously resolveSectionRef used a different
// numbering (counting Phases + default lane) than the UI lane labels.
func TestSelectSectionByNumberMatchesUI(t *testing.T) {
	a := NewApp()
	// a Phases lane (no number) + two real sections
	w1 := mustWP(t, a, "a", "2026-02-01", "2026-02-10")
	a.SetActionCluster(w1, "Phases")
	w2 := mustWP(t, a, "b", "2026-03-01", "2026-03-10")
	a.SetActionCluster(w2, "Design")
	w3 := mustWP(t, a, "c", "2026-04-01", "2026-04-10")
	a.SetActionCluster(w3, "Build")
	if _, err := a.AddSection("Rollout"); err != nil {
		t.Fatal(err)
	}

	d := a.GetProject()
	// for every numbered section the UI shows, `select <n>` must select exactly
	// that section.
	a.SetMode("section")
	for name, n := range d.SectionNums {
		ref := fmt.Sprintf("%d", n)
		if got := a.resolveSectionRef(ref); got != name {
			t.Fatalf("select %s resolved to %q, UI labels section %q as %d", ref, got, name, n)
		}
		// and going through the chat `select <n>` sets the selection to it
		if _, err := a.Chat("select " + ref); err != nil {
			t.Fatalf("chat select %s: %v", ref, err)
		}
		sel := a.Selection()
		if sel.Kind != "section" || sel.ID != name {
			t.Fatalf("chat select %s -> selection %+v, want section %q", ref, sel, name)
		}
	}
	// Phases carries no number and must not be reachable by a stray number.
	if _, ok := d.SectionNums["Phases"]; ok {
		t.Fatal("Phases should have no WP number")
	}
}

// TestWpSectionNumsMatchesDTO guards that the App.wpSectionNums() mapping used
// by resolveSectionRef stays identical to the DTO SectionNums the UI renders.
func TestWpSectionNumsMatchesDTO(t *testing.T) {
	a := NewApp()
	mustWP2(t, a, "a", "Design")
	mustWP2(t, a, "b", "Build")
	mustWP2(t, a, "p", "Phases")
	if _, err := a.AddSection("ZEmpty"); err != nil {
		t.Fatal(err)
	}
	got := a.wpSectionNums()
	want := a.GetProject().SectionNums
	if len(got) != len(want) {
		t.Fatalf("size mismatch: wpSectionNums=%v dto=%v", got, want)
	}
	for name, n := range want {
		if got[name] != n {
			t.Fatalf("section %q: wpSectionNums=%d dto=%d", name, got[name], n)
		}
	}
}

// mustWP2 adds an action in a given cluster.
func mustWP2(t *testing.T, a *App, name, cluster string) string {
	t.Helper()
	id := mustWP(t, a, name, "2026-02-01", "2026-02-10")
	if _, err := a.SetActionCluster(id, cluster); err != nil {
		t.Fatal(err)
	}
	return id
}

// TestSelectSectionThenAddActionLandsInSection verifies that selecting a section
// (by number) makes it the current section, so a new action created afterwards
// lands in that section — both via the Select binding and the chat `select`.
func TestSelectSectionThenAddActionLandsInSection(t *testing.T) {
	a := NewApp()
	mustWP2(t, a, "d", "Design")
	mustWP2(t, a, "b", "Build")
	mustWP2(t, a, "r", "Rollout")
	// section numbers: Design=1, Build=2, Rollout=3
	nums := a.wpSectionNums()

	// via the Select binding (what a canvas click / selectSection uses)
	if _, err := a.Select("section", fmt.Sprintf("%d", nums["Build"])); err != nil {
		t.Fatal(err)
	}
	if _, err := a.AddWorkPackage("new1", "2026-05-01", "2026-05-10"); err != nil {
		t.Fatal(err)
	}
	n1 := a.project.Packages[len(a.project.Packages)-1]
	if n1.Cluster != "Build" {
		t.Fatalf("after Select section Build, new action should be in Build, got %q", n1.Cluster)
	}

	// via the chat `select <n>` in section mode
	a.SetMode("section")
	if _, err := a.Chat(fmt.Sprintf("select %d", nums["Rollout"])); err != nil {
		t.Fatal(err)
	}
	a.SetMode("action")
	if _, err := a.AddWorkPackage("new2", "2026-06-01", "2026-06-10"); err != nil {
		t.Fatal(err)
	}
	n2 := a.project.Packages[len(a.project.Packages)-1]
	if n2.Cluster != "Rollout" {
		t.Fatalf("after chat select Rollout, new action should be in Rollout, got %q", n2.Cluster)
	}
}

// TestChatAddSelectsNewActionNotSection verifies that after `select <section>`
// then `add action <name>`, the backend selection is the NEW ACTION, not
// the previously selected section — so the UI stays in action mode and focuses
// the new action instead of flipping back to section mode. (Regression: the
// section stayed selected, so the chat post-sync flipped the UI to section.)
func TestChatAddSelectsNewActionNotSection(t *testing.T) {
	a := NewApp()
	mustWP2(t, a, "d", "Design")
	mustWP2(t, a, "b", "Build")
	nums := a.wpSectionNums()

	a.SetMode("section")
	if _, err := a.Chat(fmt.Sprintf("select %d", nums["Build"])); err != nil {
		t.Fatal(err)
	}
	if a.Selection().Kind != "section" {
		t.Fatal("after select, a section should be selected")
	}
	a.SetMode("action")
	if _, err := a.Chat("add action demo"); err != nil {
		t.Fatal(err)
	}
	// the new action must now be the selection (not the section)
	sel := a.Selection()
	if sel.Kind != "action" {
		t.Fatalf("after `add`, selection should be the new action, got %+v", sel)
	}
	newAct := a.project.Packages[len(a.project.Packages)-1]
	if sel.ID != newAct.ID {
		t.Fatalf("selection %q is not the new action %q", sel.ID, newAct.ID)
	}
	// and it still landed in the selected section (Build)
	if newAct.Cluster != "Build" {
		t.Fatalf("new action should be in Build, got %q", newAct.Cluster)
	}
	if newAct.Name != "demo" {
		t.Fatalf("new action name = %q, want demo", newAct.Name)
	}
}
