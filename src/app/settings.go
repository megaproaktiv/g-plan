// settings.go — small, system-wide app settings persisted outside any project.
// Currently only the last-used directory (for Open / Save As pre-selection and
// as the fallback save location). Stored as JSON in the user's config dir.
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// appSettings is the on-disk shape of the global settings file.
type appSettings struct {
	LastDir string `json:"lastDir,omitempty"`
}

// settingsPath returns the path to the settings file in the OS user-config dir
// (e.g. ~/.config/pt-plan/settings.json on Linux, ~/Library/Application
// Support/pt-plan/... on macOS). Falls back to the home dir if unavailable.
func settingsPath() string {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		if home, herr := os.UserHomeDir(); herr == nil {
			dir = home
		} else {
			dir = "."
		}
	}
	return filepath.Join(dir, "pt-plan", "settings.json")
}

// loadSettings reads the settings file; missing/corrupt returns zero settings.
func loadSettings() appSettings {
	var s appSettings
	b, err := os.ReadFile(settingsPath())
	if err != nil {
		return s
	}
	_ = json.Unmarshal(b, &s) // ignore corrupt file -> defaults
	return s
}

// saveSettings writes the settings file, creating the directory as needed.
func saveSettings(s appSettings) {
	p := settingsPath()
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	if b, err := json.MarshalIndent(s, "", "  "); err == nil {
		_ = os.WriteFile(p, b, 0o644)
	}
}

// rememberDir persists dir as the last-used directory (best effort).
func rememberDir(dir string) {
	if dir == "" {
		return
	}
	s := loadSettings()
	if s.LastDir == dir {
		return
	}
	s.LastDir = dir
	saveSettings(s)
}

// lastDir returns the persisted last-used directory ("" if none).
func lastDir() string { return loadSettings().LastDir }

// downloadsDir returns the user's Downloads folder, or the home dir / cwd as a
// fallback. Used as the autosave location when a project has no path yet.
func downloadsDir() string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		dl := filepath.Join(home, "Downloads")
		if fi, serr := os.Stat(dl); serr == nil && fi.IsDir() {
			return dl
		}
		return home
	}
	if ld := lastDir(); ld != "" {
		return ld
	}
	return "."
}
