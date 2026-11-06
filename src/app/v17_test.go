package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/megaproaktiv/g-plan/src/app/plan"
)

// --- Session 21: autosave versioning ---

// TestVersionCountClampAndPersist verifies the kept-version count clamps to
// 5..10 and that SaveVersions + VersionCount round-trip through a save/open.
func TestVersionCountClampAndPersist(t *testing.T) {
	a := NewApp()
	// default: versioning off, count 5
	if a.project.SaveVersions || a.project.VersionCount != 5 {
		t.Fatalf("defaults wrong: saveVersions=%v count=%d", a.project.SaveVersions, a.project.VersionCount)
	}
	a.SetAutosaveConfig(true, 5, true, true, 2) // 2 -> 5
	if a.project.VersionCount != 5 {
		t.Fatalf("count < 5 should clamp to 5, got %d", a.project.VersionCount)
	}
	a.SetAutosaveConfig(true, 5, true, true, 99) // 99 -> 10
	if a.project.VersionCount != 10 {
		t.Fatalf("count > 10 should clamp to 10, got %d", a.project.VersionCount)
	}
	a.SetAutosaveConfig(true, 5, true, true, 7)
	dir := t.TempDir()
	path := filepath.Join(dir, "p"+plan.ProjectFileExt)
	if _, err := a.SaveProject(path); err != nil {
		t.Fatal(err)
	}
	b := NewApp()
	if _, err := b.OpenProject(path); err != nil {
		t.Fatal(err)
	}
	if !b.project.SaveVersions || b.project.VersionCount != 7 {
		t.Fatalf("round-trip lost versioning: saveVersions=%v count=%d", b.project.SaveVersions, b.project.VersionCount)
	}
}

// TestAutosaveRotatesVersions verifies that, with versioning on, each autosave
// rotates the previous on-disk content into a numbered sibling, keeping at most
// VersionCount, with the highest number being the newest.
func TestAutosaveRotatesVersions(t *testing.T) {
	a := NewApp()
	dir := t.TempDir()
	path := filepath.Join(dir, "plan"+plan.ProjectFileExt)
	if _, err := a.SaveProject(path); err != nil {
		t.Fatal(err)
	}
	a.SetAutosaveConfig(true, 0, true, true, 5) // versioning on, keep 5

	// perform several autosave cycles: mutate -> AutosaveTick (rotates + writes)
	for i := 0; i < 3; i++ {
		mustWP(t, a, "wp"+string(rune('A'+i)), "2026-02-01", "2026-02-10")
		if saved, err := a.AutosaveTick(); err != nil || saved != path {
			t.Fatalf("cycle %d: autosave failed, saved=%q err=%v", i, saved, err)
		}
	}
	versions := plan.ListVersions(path)
	if len(versions) != 3 {
		t.Fatalf("expected 3 rotated versions, got %d: %v", len(versions), versions)
	}
	// names are <base>.NNN.ptplan, ascending; newest (highest) is last
	wantLast := filepath.Join(dir, "plan.003"+plan.ProjectFileExt)
	if versions[len(versions)-1] != wantLast {
		t.Fatalf("newest version should be %q, got %q", wantLast, versions[len(versions)-1])
	}
	// the earliest version holds the ORIGINAL (empty) project content
	if _, err := os.Stat(versions[0]); err != nil {
		t.Fatalf("oldest version missing: %v", err)
	}
}

// TestVersionPruneKeepsNewest verifies old versions are pruned to the kept
// count, retaining the newest ones.
func TestVersionPruneKeepsNewest(t *testing.T) {
	a := NewApp()
	dir := t.TempDir()
	path := filepath.Join(dir, "plan"+plan.ProjectFileExt)
	if _, err := a.SaveProject(path); err != nil {
		t.Fatal(err)
	}
	a.SetAutosaveConfig(true, 0, true, true, 5) // keep 5

	// 8 rotations -> only the newest 5 should remain (004..008)
	for i := 0; i < 8; i++ {
		mustWP(t, a, "wp"+string(rune('A'+i)), "2026-02-01", "2026-02-10")
		if _, err := a.AutosaveTick(); err != nil {
			t.Fatal(err)
		}
	}
	versions := plan.ListVersions(path)
	if len(versions) != 5 {
		t.Fatalf("expected 5 kept versions, got %d: %v", len(versions), versions)
	}
	wantFirst := filepath.Join(dir, "plan.004"+plan.ProjectFileExt)
	wantLast := filepath.Join(dir, "plan.008"+plan.ProjectFileExt)
	if versions[0] != wantFirst || versions[len(versions)-1] != wantLast {
		t.Fatalf("kept window should be 004..008, got %q..%q", versions[0], versions[len(versions)-1])
	}
}

// TestNoVersionsWhenOff verifies no version files are created when versioning
// is off.
func TestNoVersionsWhenOff(t *testing.T) {
	a := NewApp()
	dir := t.TempDir()
	path := filepath.Join(dir, "plan"+plan.ProjectFileExt)
	if _, err := a.SaveProject(path); err != nil {
		t.Fatal(err)
	}
	a.SetAutosaveConfig(true, 0, true, false, 5) // versioning OFF
	mustWP(t, a, "wp", "2026-02-01", "2026-02-10")
	if _, err := a.AutosaveTick(); err != nil {
		t.Fatal(err)
	}
	if v := plan.ListVersions(path); len(v) != 0 {
		t.Fatalf("no versions expected when off, got %v", v)
	}
}
