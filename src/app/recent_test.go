package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func isolateSettings(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
}

func TestPushRecent(t *testing.T) {
	var l []string
	for _, p := range []string{"/a", "/b", "/c", "/d", "/e", "/f"} {
		l = pushRecent(l, p, 5)
	}
	if want := []string{"/f", "/e", "/d", "/c", "/b"}; !reflect.DeepEqual(l, want) {
		t.Fatalf("got %v want %v", l, want)
	}
	l = pushRecent(l, "/d", 5) // duplicate moves to front
	if want := []string{"/d", "/f", "/e", "/c", "/b"}; !reflect.DeepEqual(l, want) {
		t.Fatalf("got %v want %v", l, want)
	}
	if got := pushRecent(l, "", 5); !reflect.DeepEqual(got, l) {
		t.Fatalf("empty path must not change the list: %v", got)
	}
}

func TestRecentOnlyAfterSave(t *testing.T) {
	isolateSettings(t)
	a := NewApp()
	dir := t.TempDir()
	p := filepath.Join(dir, "one.ptplan")
	if _, err := a.SaveProject(p); err != nil {
		t.Fatal(err)
	}
	r := a.RecentFiles()
	if len(r) != 1 || r[0].Path != p || !r[0].Exists || r[0].Name != "one.ptplan" {
		t.Fatalf("after save: %+v", r)
	}
	// opening (without saving) must not add or reorder
	b := NewApp()
	p2 := filepath.Join(dir, "two.ptplan")
	if _, err := b.SaveProject(p2); err != nil {
		t.Fatal(err)
	}
	if _, err := b.OpenRecent(p); err != nil {
		t.Fatal(err)
	}
	if r := b.RecentFiles(); len(r) != 2 || r[0].Path != p2 {
		t.Fatalf("open must not reorder: %+v", r)
	}
}

func TestOpenRecentMissingIsDropped(t *testing.T) {
	isolateSettings(t)
	a := NewApp()
	p := filepath.Join(t.TempDir(), "gone.ptplan")
	if _, err := a.SaveProject(p); err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(p)
	if r := a.RecentFiles(); len(r) != 1 || r[0].Exists {
		t.Fatalf("expected one missing entry: %+v", r)
	}
	if _, err := a.OpenRecent(p); err == nil {
		t.Fatal("expected error for missing file")
	}
	if r := a.RecentFiles(); len(r) != 0 {
		t.Fatalf("missing entry must be dropped: %+v", r)
	}
}

func TestReopenLast(t *testing.T) {
	isolateSettings(t)
	a := NewApp()
	p := filepath.Join(t.TempDir(), "last.ptplan")
	if _, err := a.SaveProject(p); err != nil {
		t.Fatal(err)
	}
	if NewApp().OpenLastIfEnabled() {
		t.Fatal("must not reopen while the setting is off")
	}
	a.SetReopenLast(true)
	if !a.ReopenLastEnabled() {
		t.Fatal("setting not stored")
	}
	b := NewApp()
	if !b.OpenLastIfEnabled() || b.SavePath() != p {
		t.Fatalf("expected reopen of %s, got %q", p, b.SavePath())
	}
	a.ClearRecent()
	if NewApp().OpenLastIfEnabled() {
		t.Fatal("nothing to reopen after clear")
	}
}
