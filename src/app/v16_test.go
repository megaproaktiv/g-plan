package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/megaproaktiv/g-plan/src/app/plan"
)

// --- Session 16: open/close/save-safe redesign ---

// TestAutosaveConfigClampAndPersist verifies SetAutosaveConfig clamps the
// interval (0 passes; 1..4 -> 5; >99 -> 99) and that the value + on-exit flag
// round-trip through a save/open of the single-file .ptplan.
func TestAutosaveConfigClampAndPersist(t *testing.T) {
	a := NewApp()
	// default on a fresh project
	if a.project.AutosaveMinutes != 5 || !a.project.AutosaveOnExit {
		t.Fatalf("defaults wrong: minutes=%d onExit=%v", a.project.AutosaveMinutes, a.project.AutosaveOnExit)
	}
	a.SetAutosaveConfig(true, 3, false, false, 5) // 3 -> 5
	if a.project.AutosaveMinutes != 5 {
		t.Fatalf("1..4 should clamp to 5, got %d", a.project.AutosaveMinutes)
	}
	a.SetAutosaveConfig(true, 150, true, false, 5) // 150 -> 99
	if a.project.AutosaveMinutes != 99 {
		t.Fatalf(">99 should clamp to 99, got %d", a.project.AutosaveMinutes)
	}
	a.SetAutosaveConfig(true, 0, true, false, 5) // 0 = every change, valid
	if a.project.AutosaveMinutes != 0 {
		t.Fatalf("0 should pass through, got %d", a.project.AutosaveMinutes)
	}

	// round-trip: set to 42 / onExit false, save, reopen
	a.SetAutosaveConfig(true, 42, false, false, 5)
	dir := t.TempDir()
	path := filepath.Join(dir, "p"+plan.ProjectFileExt)
	if _, err := a.SaveProject(path); err != nil {
		t.Fatal(err)
	}
	b := NewApp()
	if _, err := b.OpenProject(path); err != nil {
		t.Fatal(err)
	}
	if b.project.AutosaveMinutes != 42 || b.project.AutosaveOnExit {
		t.Fatalf("round-trip lost autosave config: minutes=%d onExit=%v", b.project.AutosaveMinutes, b.project.AutosaveOnExit)
	}
}

// TestOldFileKeepsAutosaveDefaults verifies a project file predating the
// autosave keys loads with the NewProject defaults (5 min, on-exit true)
// instead of 0/false.
func TestOldFileKeepsAutosaveDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "old"+plan.ProjectFileExt)
	// minimal valid project JSON with the right schema but NO autosave keys
	json := `{"schema":"` + plan.ProjectSchema + `","title":"Old","unit":"days","hoursPerDay":8,` +
		`"milestones":[],"phases":[],"packages":[],"deps":[],"roles":[]}`
	if err := os.WriteFile(path, []byte(json), 0o644); err != nil {
		t.Fatal(err)
	}
	a := NewApp()
	if _, err := a.OpenProject(path); err != nil {
		t.Fatal(err)
	}
	if a.project.AutosaveMinutes != 5 || !a.project.AutosaveOnExit || !a.project.AutosaveEnabled {
		t.Fatalf("old file should keep defaults, got minutes=%d onExit=%v enabled=%v", a.project.AutosaveMinutes, a.project.AutosaveOnExit, a.project.AutosaveEnabled)
	}
}

// TestSaveSmartUsesKnownPath verifies SaveSmart writes to the current path when
// one is set, and clears the dirty flag.
func TestSaveSmartUsesKnownPath(t *testing.T) {
	a := NewApp()
	dir := t.TempDir()
	path := filepath.Join(dir, "known"+plan.ProjectFileExt)
	if _, err := a.SaveProject(path); err != nil {
		t.Fatal(err)
	}
	mustWP(t, a, "task", "2026-02-01", "2026-02-10")
	if !a.IsDirty() {
		t.Fatal("adding a work package should make the project dirty")
	}
	saved, err := a.SaveSmart()
	if err != nil {
		t.Fatal(err)
	}
	if saved != path {
		t.Fatalf("SaveSmart should use the known path %q, got %q", path, saved)
	}
	if a.IsDirty() {
		t.Fatal("SaveSmart should clear the dirty flag")
	}
}

// TestAutosaveTickGate verifies AutosaveTick saves only when dirty AND a path
// exists (never invents a location).
func TestAutosaveTickGate(t *testing.T) {
	a := NewApp()
	// no path yet: tick is a no-op even when dirty
	mustWP(t, a, "task", "2026-02-01", "2026-02-10")
	if saved, err := a.AutosaveTick(); err != nil || saved != "" {
		t.Fatalf("tick without a path should be a no-op, got %q err=%v", saved, err)
	}
	// give it a path, mark saved (clean) -> tick no-op
	dir := t.TempDir()
	path := filepath.Join(dir, "t"+plan.ProjectFileExt)
	if _, err := a.SaveProject(path); err != nil {
		t.Fatal(err)
	}
	if saved, _ := a.AutosaveTick(); saved != "" {
		t.Fatalf("clean project should not autosave, got %q", saved)
	}
	// mutate -> dirty -> tick saves to the known path
	mustWP(t, a, "task2", "2026-03-01", "2026-03-10")
	if saved, err := a.AutosaveTick(); err != nil || saved != path {
		t.Fatalf("dirty+path should autosave to %q, got %q err=%v", path, saved, err)
	}
	if a.IsDirty() {
		t.Fatal("AutosaveTick should clear dirty")
	}
}

// TestNewProjectSavesDirtyFirst verifies New saves a dirty project (to its
// known path) before blanking the slate (must-save, no-discard rule).
func TestNewProjectSavesDirtyFirst(t *testing.T) {
	a := NewApp()
	dir := t.TempDir()
	path := filepath.Join(dir, "cur"+plan.ProjectFileExt)
	if _, err := a.SaveProject(path); err != nil {
		t.Fatal(err)
	}
	mustWP(t, a, "unsaved work", "2026-02-01", "2026-02-10")
	if _, err := a.NewProject(true); err != nil {
		t.Fatal(err)
	}
	// the OLD project's change must have been flushed to its file
	b := NewApp()
	if _, err := b.OpenProject(path); err != nil {
		t.Fatal(err)
	}
	if b.project.WorkPackage("unsaved_work") == nil && len(b.project.Packages) == 0 {
		t.Fatal("New should have saved the dirty project before blanking")
	}
	// and the live project is now a clean, empty Untitled
	if a.base != "" || a.IsDirty() || len(a.project.Packages) != 0 {
		t.Fatalf("after New: base=%q dirty=%v packages=%d", a.base, a.IsDirty(), len(a.project.Packages))
	}
}

// TestSaveOnExitRespectsFlag verifies SaveOnExit only saves when the flag is on
// and the project is dirty.
func TestSaveOnExitRespectsFlag(t *testing.T) {
	a := NewApp()
	dir := t.TempDir()
	path := filepath.Join(dir, "exit"+plan.ProjectFileExt)
	if _, err := a.SaveProject(path); err != nil {
		t.Fatal(err)
	}
	mustWP(t, a, "task", "2026-02-01", "2026-02-10")
	a.SetAutosaveConfig(true, 5, false, false, 5) // on-exit OFF
	if got := a.SaveOnExit(); got != "" {
		t.Fatalf("on-exit off should not save, got %q", got)
	}
	a.SetAutosaveConfig(true, 5, true, false, 5) // on-exit ON
	if got := a.SaveOnExit(); got != path {
		t.Fatalf("on-exit on + dirty should save to %q, got %q", path, got)
	}
}

// TestAutosaveEnabledGate verifies the master on/off switch: when autosave is
// disabled, AutosaveTick does nothing even if dirty with a known path; when
// enabled it saves. Also checks the default is enabled.
func TestAutosaveEnabledGate(t *testing.T) {
	a := NewApp()
	if !a.project.AutosaveEnabled {
		t.Fatal("autosave should default to enabled")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "g"+plan.ProjectFileExt)
	if _, err := a.SaveProject(path); err != nil {
		t.Fatal(err)
	}
	mustWP(t, a, "task", "2026-02-01", "2026-02-10")
	// turn autosave OFF -> tick is a no-op even though dirty + path
	a.SetAutosaveConfig(false, 5, true, false, 5)
	if saved, _ := a.AutosaveTick(); saved != "" {
		t.Fatalf("autosave off should not save, got %q", saved)
	}
	if !a.IsDirty() {
		t.Fatal("autosave off should leave the project dirty")
	}
	// turn autosave ON -> tick saves
	a.SetAutosaveConfig(true, 5, true, false, 5)
	if saved, err := a.AutosaveTick(); err != nil || saved != path {
		t.Fatalf("autosave on + dirty should save to %q, got %q err=%v", path, saved, err)
	}
}

// TestAutosaveEnabledPersist verifies the enabled flag round-trips and that an
// old file (no key) defaults to enabled.
func TestAutosaveEnabledPersist(t *testing.T) {
	a := NewApp()
	a.SetAutosaveConfig(false, 10, true, false, 5)
	dir := t.TempDir()
	path := filepath.Join(dir, "e"+plan.ProjectFileExt)
	if _, err := a.SaveProject(path); err != nil {
		t.Fatal(err)
	}
	b := NewApp()
	if _, err := b.OpenProject(path); err != nil {
		t.Fatal(err)
	}
	if b.project.AutosaveEnabled {
		t.Fatal("autosave-enabled=false should round-trip")
	}
}
