package plan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFileAtomicReplacesAndLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.ptplan")
	if err := WriteFileAtomic(p, []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomic(p, []byte("two"), 0o644); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); string(b) != "two" {
		t.Fatalf("content %q", b)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("temp files left behind: %v", entries)
	}
}

func TestWriteFileAtomicKeepsPermissionsAndSymlink(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real.ptplan")
	link := filepath.Join(dir, "link.ptplan")
	if err := os.WriteFile(real, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, link); err != nil {
		t.Skip("symlinks not available")
	}
	if err := WriteFileAtomic(link, []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}
	if st, _ := os.Lstat(link); st.Mode()&os.ModeSymlink == 0 {
		t.Error("the symbolic link was replaced by a file")
	}
	if b, _ := os.ReadFile(real); string(b) != "y" {
		t.Errorf("target content %q", b)
	}
	if st, _ := os.Stat(real); st.Mode().Perm() != 0o600 {
		t.Errorf("permissions changed to %v", st.Mode().Perm())
	}
}

// A failed write (target directory missing / not writable) must leave the
// existing file untouched.
func TestFailedAtomicWriteKeepsTheOldFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.ptplan")
	os.WriteFile(p, []byte("good"), 0o644)
	if err := WriteFileAtomic(filepath.Join(dir, "missing", "a.ptplan"), []byte("bad"), 0o644); err == nil {
		t.Fatal("writing into a missing directory must fail")
	}
	if b, _ := os.ReadFile(p); string(b) != "good" {
		t.Fatalf("old file damaged: %q", b)
	}
}

func TestSaveJSONKeepsBackupOfThePreviousVersion(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "plan"+ProjectFileExt)
	pr := NewProject("First", UnitDays)
	if err := pr.SaveJSON(p); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p + BackupSuffix); err == nil {
		t.Error("the first save has nothing to back up")
	}
	first, _ := os.ReadFile(p)
	pr.Title = "Second"
	if err := pr.SaveJSON(p); err != nil {
		t.Fatal(err)
	}
	bak, err := os.ReadFile(p + BackupSuffix)
	if err != nil {
		t.Fatal(err)
	}
	if string(bak) != string(first) || !strings.Contains(string(bak), "First") {
		t.Errorf("backup is not the previous content: %.80s", bak)
	}
	now, _ := os.ReadFile(p)
	if !strings.Contains(string(now), "Second") {
		t.Error("the new content was not written")
	}
	// a backup never counts as a rotated version
	if v := ListVersions(p); len(v) != 0 {
		t.Errorf("backup listed as version: %v", v)
	}
	// and the file still loads
	if _, err := OpenJSON(p); err != nil {
		t.Fatal(err)
	}
}
