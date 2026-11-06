package plan

import (
	"io"
	"os"
	"path/filepath"
)

// BackupSuffix is appended to a project file name for the copy of the previous
// version that SaveJSON keeps ("plan.ptplan" -> "plan.ptplan.bak").
const BackupSuffix = ".bak"

// WriteFileAtomic writes data to path so that a crash, a full disk or a power
// cut can never leave a half-written file: the data goes to a temporary file in
// the SAME directory (so the final rename stays on one file system), is flushed
// to disk, and only then replaces path with one rename. Either the old or the
// new content is on disk, never a mixture. An existing file keeps its
// permissions; a new one gets perm. A symbolic link is followed, so the link
// itself stays in place.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	return writeAtomic(path, data, perm, false)
}

// WriteFileAtomicBackup is WriteFileAtomic, and before the replacement it keeps
// the previous content as path+BackupSuffix (a hard link where the file system
// supports it, else a copy). The backup is therefore the last good version even
// if the new content turns out to be unwanted.
func WriteFileAtomicBackup(path string, data []byte, perm os.FileMode) error {
	return writeAtomic(path, data, perm, true)
}

func writeAtomic(path string, data []byte, perm os.FileMode, backup bool) error {
	if real, err := filepath.EvalSymlinks(path); err == nil {
		path = real
	}
	if st, err := os.Stat(path); err == nil {
		perm = st.Mode().Perm()
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			_ = tmp.Close()
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Chmod(perm); err != nil && !isChmodUnsupported(err) {
		return err
	}
	if err := tmp.Sync(); err != nil { // the data is on disk before it replaces the old file
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if backup {
		if _, err := os.Stat(path); err == nil {
			_ = keepBackup(path, path+BackupSuffix) // a missing backup must not stop the save
		}
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	ok = true
	syncDir(dir)
	return nil
}

// keepBackup makes bak a copy of src without ever touching src.
func keepBackup(src, bak string) error {
	_ = os.Remove(bak)
	if err := os.Link(src, bak); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(bak)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = os.Remove(bak)
		return err
	}
	return out.Close()
}

// syncDir flushes the directory entry of the rename where the platform allows
// it (not supported on Windows; errors are ignored on purpose).
func syncDir(dir string) {
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
}

// isChmodUnsupported reports the errors of file systems without permissions.
func isChmodUnsupported(err error) bool { return os.IsPermission(err) || os.IsNotExist(err) }
