package plan

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Single-file project format (pt-plan/project@2).
//
// The project used to be stored as three sibling files (.mmd graphic source +
// .roles.json + generated .md). As the app has diverged from plain
// Gantt/Mermaid, the canonical on-disk format is now ONE self-contained JSON
// file that includes EVERYTHING: project config (unit, hours/day, zoom, view
// options, arrow style), the graphic content (milestones, phases, work
// packages with their detail sub-packages, dependencies), roles + assignments,
// section colors, and baselines.
//
// projectFile is the on-disk shape. Every field the app needs is explicit here
// (the Project struct tags most graphic slices json:"-" because they used to
// live in the .mmd, so we cannot just marshal Project directly).

const ProjectSchema = "pt-plan/project@2"

// ProjectFileExt is the extension for the single-file format. The file content
// is JSON, but the extension is the app's own ".ptplan" (not ".ptplan.json").
const ProjectFileExt = ".ptplan"

type projectFile struct {
	Schema string `json:"schema"`

	// --- project configuration ---
	Title           string `json:"title"`
	Description     string `json:"description"`
	Unit            Unit   `json:"unit"`
	HoursPerDay     int    `json:"hoursPerDay"`
	Language        string `json:"language,omitempty"`
	Zoom            string `json:"zoom,omitempty"`
	ArrowStyle      string `json:"arrowStyle,omitempty"`
	ShowDetails     bool   `json:"showDetails"`
	ShowRoleActions bool   `json:"showRoleActions"`
	// ShowRolePercent/ShowRoleDays are pointers so a file predating them keeps
	// the default (off = role sub-rows show the name only).
	ShowRolePercent *bool `json:"showRolePercent,omitempty"`
	ShowRoleDays    *bool `json:"showRoleDays,omitempty"`
	ShowPhases      bool  `json:"showPhases"`
	// ShowProgress is a pointer so a file predating it (key absent) keeps the
	// NewProject default (false).
	ShowProgress *bool `json:"showProgress,omitempty"`
	// Monitor is a pointer so files predating it keep the default (off).
	Monitor *Monitor `json:"monitor,omitempty"`
	// AutosaveEnabled/AutosaveMinutes/AutosaveOnExit are pointers so that a
	// project file predating these fields (key absent) keeps the NewProject
	// defaults (enabled, 5 minutes, on-exit true) instead of off/0/false.
	AutosaveEnabled *bool  `json:"autosaveEnabled,omitempty"`
	AutosaveMinutes *int   `json:"autosaveMinutes,omitempty"`
	AutosaveOnExit  *bool  `json:"autosaveOnExit,omitempty"`
	SaveVersions    *bool  `json:"saveVersions,omitempty"`
	VersionCount    *int   `json:"versionCount,omitempty"`
	ReportFormat    string `json:"reportFormat,omitempty"`
	ReportTodayLine *bool  `json:"reportTodayLine,omitempty"`
	ReportGrid      string `json:"reportGrid,omitempty"`
	ReportShowNotes *bool  `json:"reportShowNotes,omitempty"`
	ShowWeekNumbers bool   `json:"showWeekNumbers,omitempty"`

	// --- graphic content ---
	Milestones []Milestone   `json:"milestones"`
	Phases     []Phase       `json:"phases"`
	Packages   []WorkPackage `json:"packages"`
	Deps       []Dependency  `json:"deps"`

	// --- people ---
	Roles []Role `json:"roles"`

	// --- metadata ---
	SectionColors map[string]string `json:"sectionColors,omitempty"`
	SectionOrder  []string          `json:"sectionOrder,omitempty"`
	Baselines     []Baseline        `json:"baselines,omitempty"`
}

// ToFile builds the on-disk representation from the live project.
func (p *Project) toFile() projectFile {
	return projectFile{
		Schema:          ProjectSchema,
		Title:           p.Title,
		Description:     p.Description,
		Unit:            p.Unit,
		HoursPerDay:     p.hpd(),
		Language:        NormalizeLanguage(p.Language),
		Zoom:            p.Zoom,
		ArrowStyle:      p.ArrowStyle,
		ShowDetails:     p.ShowDetails,
		ShowRoleActions: p.ShowRoleActions,
		ShowRolePercent: &p.ShowRolePercent,
		ShowRoleDays:    &p.ShowRoleDays,
		ShowPhases:      p.ShowPhases,
		ShowProgress:    &p.ShowProgress,
		Monitor:         &p.Monitor,
		AutosaveEnabled: &p.AutosaveEnabled,
		AutosaveMinutes: func() *int { m := ClampAutosaveMinutes(p.AutosaveMinutes); return &m }(),
		AutosaveOnExit:  &p.AutosaveOnExit,
		SaveVersions:    &p.SaveVersions,
		VersionCount:    func() *int { n := ClampVersionCount(p.VersionCount); return &n }(),
		ReportFormat:    NormalizeReportFormat(p.ReportFormat),
		ReportTodayLine: &p.ReportTodayLine,
		ReportGrid:      NormalizeReportGrid(p.ReportGrid),
		ReportShowNotes: &p.ReportShowNotes,
		ShowWeekNumbers: p.ShowWeekNumbers,
		Milestones:      p.Milestones,
		Phases:          p.Phases,
		Packages:        p.Packages,
		Deps:            p.Deps,
		Roles:           p.Roles,
		SectionColors:   p.SectionColors,
		SectionOrder:    p.SectionOrder,
		Baselines:       p.Baselines,
	}
}

// fromFile loads the on-disk representation into the live project.
func (p *Project) fromFile(f projectFile) {
	p.Title = f.Title
	p.Description = f.Description
	if f.Unit != "" {
		p.Unit = f.Unit
	}
	if f.HoursPerDay > 0 {
		p.HoursPerDay = f.HoursPerDay
	}
	if f.Language != "" {
		p.Language = NormalizeLanguage(f.Language)
	}
	p.Zoom = f.Zoom
	p.ArrowStyle = f.ArrowStyle
	p.ShowDetails = f.ShowDetails
	p.ShowRoleActions = f.ShowRoleActions
	if f.ShowRolePercent != nil {
		p.ShowRolePercent = *f.ShowRolePercent
	}
	if f.ShowRoleDays != nil {
		p.ShowRoleDays = *f.ShowRoleDays
	}
	p.ShowPhases = f.ShowPhases
	if f.ShowProgress != nil {
		p.ShowProgress = *f.ShowProgress
	}
	if f.Monitor != nil {
		p.Monitor = *f.Monitor
	}
	if f.AutosaveEnabled != nil {
		p.AutosaveEnabled = *f.AutosaveEnabled
	}
	if f.AutosaveMinutes != nil {
		p.AutosaveMinutes = ClampAutosaveMinutes(*f.AutosaveMinutes)
	}
	if f.AutosaveOnExit != nil {
		p.AutosaveOnExit = *f.AutosaveOnExit
	}
	if f.SaveVersions != nil {
		p.SaveVersions = *f.SaveVersions
	}
	if f.VersionCount != nil {
		p.VersionCount = ClampVersionCount(*f.VersionCount)
	}
	if f.ReportFormat != "" {
		p.ReportFormat = NormalizeReportFormat(f.ReportFormat)
	}
	if f.ReportTodayLine != nil {
		p.ReportTodayLine = *f.ReportTodayLine
	}
	if f.ReportGrid != "" {
		p.ReportGrid = NormalizeReportGrid(f.ReportGrid)
	}
	if f.ReportShowNotes != nil {
		p.ReportShowNotes = *f.ReportShowNotes
	}
	p.ShowWeekNumbers = f.ShowWeekNumbers
	p.Milestones = f.Milestones
	p.Phases = f.Phases
	p.Packages = f.Packages
	p.DropExternalIDsOfNone()
	p.Deps = f.Deps
	p.Roles = f.Roles
	p.EnsureRoleIDs()
	p.SectionColors = f.SectionColors
	p.SectionOrder = f.SectionOrder
	p.Baselines = f.Baselines
}

// SaveJSON writes the whole project to a single self-contained JSON file.
func (p *Project) SaveJSON(path string) error {
	b, err := p.MarshalProject()
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// MarshalProject returns the single-file JSON bytes for the project (used by
// SaveJSON and by the dirty-check comparison).
func (p *Project) MarshalProject() ([]byte, error) {
	b, err := json.MarshalIndent(p.toFile(), "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// OpenJSON reads a single-file project. It returns an error if the file is not
// a pt-plan project JSON.
func OpenJSON(path string) (*Project, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f projectFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("parse project json: %w", err)
	}
	if f.Schema != ProjectSchema {
		return nil, fmt.Errorf("not a pt-plan project file (schema %q)", f.Schema)
	}
	p := NewProject("", UnitDays)
	p.fromFile(f)
	return p, nil
}

// LooksLikeProjectJSON reports whether the bytes start with a pt-plan project
// schema marker (a cheap sniff used to pick the loader).
func LooksLikeProjectJSON(b []byte) bool {
	var probe struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal(b, &probe); err != nil {
		return false
	}
	return probe.Schema == ProjectSchema
}

// --- Autosave version rotation -------------------------------------------
//
// When "save versions" is on, autosave rotates the PREVIOUS on-disk content of
// the main file into a numbered sibling before overwriting it, keeping at most
// N versions. Version files are named "<base>.NNN<ext>" (e.g.
// "plan.001.ptplan"): the NNN is zero-padded so a lexical sort matches the
// numeric order, and the highest number is the NEWEST. Numbers increase
// monotonically; the oldest (lowest) files are pruned when the count exceeds N.

// versionRe matches a version sibling "<base>.NNN.ptplan" and captures NNN.
var versionRe = regexp.MustCompile(`\.(\d{3,})` + regexp.QuoteMeta(ProjectFileExt) + `$`)

// versionPrefix returns the "<base>." stem that version files share with the
// main file, i.e. the main path with the ProjectFileExt stripped plus a dot.
func versionPrefix(mainPath string) string {
	base := strings.TrimSuffix(mainPath, ProjectFileExt)
	return base + "."
}

// ListVersions returns the existing version files for a main path, sorted
// ASCENDING by version number (oldest first, newest last).
func ListVersions(mainPath string) []string {
	dir := filepath.Dir(mainPath)
	prefix := filepath.Base(versionPrefix(mainPath))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	type vf struct {
		path string
		num  int
	}
	var found []vf
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		m := versionRe.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		n, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		found = append(found, vf{filepath.Join(dir, name), n})
	}
	sort.Slice(found, func(i, j int) bool { return found[i].num < found[j].num })
	out := make([]string, len(found))
	for i := range found {
		out[i] = found[i].path
	}
	return out
}

// nextVersionNum returns the next (highest+1) version number for a main path.
func nextVersionNum(mainPath string) int {
	versions := ListVersions(mainPath)
	if len(versions) == 0 {
		return 1
	}
	last := versions[len(versions)-1]
	if m := versionRe.FindStringSubmatch(filepath.Base(last)); m != nil {
		if n, err := strconv.Atoi(m[1]); err == nil {
			return n + 1
		}
	}
	return len(versions) + 1
}

// RotateVersion copies the CURRENT on-disk main file into a new numbered
// version "<base>.NNN.ptplan" (NNN = highest existing + 1), then prunes the
// oldest versions so at most keep-1 remain afterwards plus the one just made
// (i.e. keep total). It is a no-op when the main file does not exist yet.
// `keep` is clamped to 5..10. Returns the version path written ("" if none).
func RotateVersion(mainPath string, keep int) (string, error) {
	keep = ClampVersionCount(keep)
	data, err := os.ReadFile(mainPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // nothing to rotate yet
		}
		return "", err
	}
	num := nextVersionNum(mainPath)
	vpath := fmt.Sprintf("%s%03d%s", versionPrefix(mainPath), num, ProjectFileExt)
	if err := os.WriteFile(vpath, data, 0o644); err != nil {
		return "", err
	}
	// prune oldest so the total kept does not exceed `keep`
	versions := ListVersions(mainPath)
	if len(versions) > keep {
		for _, old := range versions[:len(versions)-keep] {
			_ = os.Remove(old)
		}
	}
	return vpath, nil
}
