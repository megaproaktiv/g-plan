// recent.go — "Open Recent" (the last MaxRecent saved projects) and the
// system-wide "reopen the last project at startup" setting. The list lives in
// the global settings file (settings.go), never inside a project, and only
// grows after a successful SAVE (not on open and not on autosave).
package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// MaxRecent is the number of entries kept in the recent-files list.
const MaxRecent = 5

// RecentDTO is one entry of the Open Recent menu.
type RecentDTO struct {
	Path   string `json:"path"`
	Name   string `json:"name"`   // file name without directory
	Dir    string `json:"dir"`    // containing directory
	Exists bool   `json:"exists"` // false when the file vanished since
}

// pushRecent returns list with path moved/added to the front, de-duplicated
// and truncated to max entries. The input slice is not modified.
func pushRecent(list []string, path string, max int) []string {
	if path == "" || max <= 0 {
		return append([]string(nil), list...)
	}
	path = filepath.Clean(path)
	out := make([]string, 0, max)
	out = append(out, path)
	for _, p := range list {
		if len(out) >= max {
			break
		}
		if filepath.Clean(p) != path {
			out = append(out, p)
		}
	}
	return out
}

// dropRecent returns list without path.
func dropRecent(list []string, path string) []string {
	path = filepath.Clean(path)
	out := make([]string, 0, len(list))
	for _, p := range list {
		if filepath.Clean(p) != path {
			out = append(out, p)
		}
	}
	return out
}

// rememberRecent records path as the newest saved project (best effort).
func rememberRecent(path string) {
	if path == "" {
		return
	}
	s := loadSettings()
	s.RecentFiles = pushRecent(s.RecentFiles, path, MaxRecent)
	saveSettings(s)
}

// RecentFiles lists the recently saved projects, newest first.
func (a *App) RecentFiles() []RecentDTO {
	out := []RecentDTO{}
	for _, p := range loadSettings().RecentFiles {
		_, err := os.Stat(p)
		out = append(out, RecentDTO{
			Path: p, Name: filepath.Base(p), Dir: filepath.Dir(p), Exists: err == nil,
		})
	}
	return out
}

// OpenRecent opens one entry of the recent list. A file that no longer exists
// is removed from the list and reported as an error. Like Open, the caller is
// expected to save a dirty project first (the UI does).
func (a *App) OpenRecent(path string) (*ProjectDTO, error) {
	if _, err := os.Stat(path); err != nil {
		s := loadSettings()
		s.RecentFiles = dropRecent(s.RecentFiles, path)
		saveSettings(s)
		return nil, fmt.Errorf("%s no longer exists and was removed from the recent list", filepath.Base(path))
	}
	dto, err := a.OpenProject(path)
	if err == nil {
		rememberDir(filepath.Dir(path))
	}
	return dto, err
}

// ClearRecent empties the recent-files list.
func (a *App) ClearRecent() {
	s := loadSettings()
	s.RecentFiles = nil
	saveSettings(s)
}

// ReopenLastEnabled reports the "reopen last project at startup" setting.
func (a *App) ReopenLastEnabled() bool { return loadSettings().ReopenLast }

// SetReopenLast stores the "reopen last project at startup" setting.
func (a *App) SetReopenLast(on bool) {
	s := loadSettings()
	s.ReopenLast = on
	saveSettings(s)
}

// OpenLastIfEnabled opens the newest existing recent file when the setting is
// on and no file is open yet (the startup project has no path). Returns true when a project
// was opened. Called once by the UI at startup; failures are silent no-ops.
func (a *App) OpenLastIfEnabled() bool {
	s := loadSettings()
	if !s.ReopenLast || a.base != "" {
		return false
	}
	for _, p := range s.RecentFiles {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		if _, err := a.OpenProject(p); err == nil {
			return true
		}
	}
	return false
}
