// Package main is the pt-plan Wails desktop application. The backend is a thin
// binding layer over the pure-Go ptplan/plan core; all plan logic lives in the
// core so it can be unit-tested without Wails.
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/megaproaktiv/g-plan/src/app/plan"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// App is the Wails-bound application state: the currently open project and its
// file paths.
type App struct {
	app            *application.App
	project        *plan.Project
	paths          plan.Paths
	base           string
	mode           string          // chat central-object mode: "action" | "role" | "person" | "section"
	current        string          // id/name of the current object in the active mode
	currentSection string          // default section for new actions
	undoStack      []*plan.Project // snapshots for Cmd-Z undo (deep copies)
	savedMark      string          // serialized project as last saved/opened (for dirty check)
	selKind        string          // current selection kind: "action" | "section" | ""
	selID          string          // current selection id (action id or section name)
	drillID        string          // action whose detail plane is open ("" = none); add in chat adds a detail here

	// monitoring (external system synchronization); see monitor.go
	configPath string // user config file with system credentials ("" = default)
	syncMu     sync.Mutex
	syncCancel context.CancelFunc // non-nil while a sync runs
	lastSync   *syncResult        // result of the last sync (for the current project)
}

// markSaved records the current project's serialized form as the clean
// baseline so IsDirty can detect subsequent edits.
func (a *App) markSaved() {
	a.savedMark = a.serializeMark()
}

// serializeMark returns a stable string representation of the project used only
// to detect unsaved changes (compared against savedMark).
func (a *App) serializeMark() string {
	if a.project == nil {
		return ""
	}
	if cp := deepCopyProject(a.project); cp != nil {
		// reuse the single-file JSON shape as the canonical comparison form
		b, _ := cp.MarshalProject()
		return string(b)
	}
	return ""
}

// IsDirty reports whether the project has unsaved changes since the last
// save/open/new. The UI checks this before Open/New to prompt the user.
func (a *App) IsDirty() bool {
	return a.serializeMark() != a.savedMark
}

// NewApp constructs the app with an empty days-mode project.
func NewApp() *App {
	return &App{project: plan.NewProject("Untitled", plan.UnitDays), mode: "action"}
}

// --- Project lifecycle bindings ---
//
// The canonical on-disk format is now a SINGLE self-contained JSON file
// (<name>.ptplan.json) that includes everything: config (unit, hours/day,
// zoom, view options), graphic content, roles, section colors and baselines.
// Legacy three-file projects (.mmd + .roles.json) still open for migration.

// OpenProject loads a project from a path. A single-file .ptplan.json (or any
// file whose bytes carry the project schema) loads directly; otherwise it
// falls back to the legacy three-file set so old projects still open.
func (a *App) OpenProject(path string) (*ProjectDTO, error) {
	if b, err := os.ReadFile(path); err == nil && plan.LooksLikeProjectJSON(b) {
		p, err := plan.OpenJSON(path)
		if err != nil {
			return nil, err
		}
		a.project, a.base, a.paths = p, path, plan.Paths{}
		a.markSaved()
		return toDTO(p), nil
	}
	// legacy three-file fallback
	p, paths, err := plan.Open(path)
	if err != nil {
		return nil, err
	}
	a.project, a.paths, a.base = p, paths, path
	a.markSaved()
	return toDTO(p), nil
}

// OpenDialog shows a native file picker and opens the chosen project. It
// pre-selects the last-used directory and remembers the directory of whatever
// is opened.
func (a *App) OpenDialog() (*ProjectDTO, error) {
	dlg := a.app.Dialog.OpenFile().
		SetTitle("Open project").
		AddFilter("pt-plan project (*.ptplan)", "*.ptplan").
		AddFilter("Legacy plan (*.mmd)", "*.mmd")
	if ld := lastDir(); ld != "" {
		dlg = dlg.SetDirectory(ld)
	}
	path, err := dlg.PromptForSingleSelection()
	if err != nil {
		return nil, err
	}
	if path == "" {
		return toDTO(a.project), nil // cancelled: keep current
	}
	dto, err := a.OpenProject(path)
	if err == nil {
		rememberDir(filepath.Dir(path))
	}
	return dto, err
}

// jsonPathFor normalizes any chosen path to the single-file .ptplan path.
func jsonPathFor(path string) string {
	// strip legacy/ambiguous suffixes, then append the canonical extension
	for _, suf := range []string{plan.ProjectFileExt, ".ptplan.json", ".json", ".mmd", ".md"} {
		if strings.HasSuffix(path, suf) {
			path = strings.TrimSuffix(path, suf)
			break
		}
	}
	return path + plan.ProjectFileExt
}

// SaveDialog shows a native save picker and writes the single JSON file.
func (a *App) SaveDialog() (*ProjectDTO, error) {
	// Default filename is just the base name (NOT a full path) — passing a full
	// path to SetFilename makes the native dialog mangle it (e.g. ":Users:...").
	// If we have a current path, pre-select its directory too.
	defName := a.project.Title
	if defName == "" {
		defName = "Untitled"
	}
	defName += plan.ProjectFileExt
	dir := ""
	if a.base != "" {
		dir = filepath.Dir(a.base)
		defName = filepath.Base(a.base)
	} else if ld := lastDir(); ld != "" {
		dir = ld
	}
	dlg := a.app.Dialog.SaveFile().
		SetMessage("Save project").
		SetFilename(defName).
		AddFilter("pt-plan project (*.ptplan)", "*.ptplan")
	if dir != "" {
		dlg = dlg.SetDirectory(dir)
	}
	path, err := dlg.PromptForSingleSelection()
	if err != nil {
		return nil, err
	}
	if path == "" {
		return toDTO(a.project), nil // cancelled
	}
	path = jsonPathFor(path)
	if err := a.project.SaveJSON(path); err != nil {
		return nil, err
	}
	a.base = path
	a.markSaved()
	rememberDir(filepath.Dir(path))
	rememberRecent(path)
	return toDTO(a.project), nil
}

// SaveProject writes the single JSON file to base (or the current path).
func (a *App) SaveProject(base string) (string, error) {
	if base == "" {
		base = a.base
	}
	if base == "" {
		return "", fmt.Errorf("no path chosen")
	}
	base = jsonPathFor(base)
	if err := a.project.SaveJSON(base); err != nil {
		return "", err
	}
	a.base = base
	a.markSaved()
	rememberRecent(base)
	return base, nil
}

// SavePath returns the current project's save path ("" if never saved). Shown
// in the project dialog as the save directory/file.
func (a *App) SavePath() string { return a.base }

// Select marks an action or section as the current selection (shown with a
// border on the canvas). kind is "action" or "section"; ref may be #n / n /
// a WBS code / an internal id (action) or a section name. Returns the project.
func (a *App) Select(kind, ref string) (*ProjectDTO, error) {
	switch kind {
	case "action":
		id := a.resolveActionRef(ref)
		if id == "" {
			return nil, fmt.Errorf("action %q not found", ref)
		}
		a.selKind, a.selID = "action", id
	case "section":
		// section is referenced by name (or a section number via sectionNums)
		name := a.resolveSectionRef(ref)
		if name == "" {
			return nil, fmt.Errorf("section %q not found", ref)
		}
		a.selKind, a.selID = "section", name
		// Selecting a section also makes it the CURRENT section, so a new action
		// (chat `add`, the + button, AddWorkPackage) is created in it.
		a.currentSection = name
	case "milestone":
		id := a.resolveMilestoneRef(ref)
		if id == "" {
			return nil, fmt.Errorf("milestone %q not found", ref)
		}
		a.selKind, a.selID = "milestone", id
	default:
		return nil, fmt.Errorf("select kind must be action|section|milestone")
	}
	return toDTO(a.project), nil
}

// Deselect clears the current selection (Escape / Ctrl).
func (a *App) Deselect() *ProjectDTO {
	a.selKind, a.selID = "", ""
	return toDTO(a.project)
}

// SelectionDTO reports the current backend selection for the UI.
type SelectionDTO struct {
	Kind string `json:"kind"` // "action" | "section" | ""
	ID   string `json:"id"`
}

// Selection returns the current selection (chat `select` or a canvas click).
func (a *App) Selection() SelectionDTO {
	return SelectionDTO{Kind: a.selKind, ID: a.selID}
}

// resolveMilestoneRef resolves a milestone reference (internal id, a WBS code
// like 1.2, or a name) to the milestone id, or "" if not found.
func (a *App) resolveMilestoneRef(ref string) string {
	ref = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(ref), "#"))
	for i := range a.project.Milestones {
		if a.project.Milestones[i].ID == ref || a.project.Milestones[i].Name == ref {
			return a.project.Milestones[i].ID
		}
	}
	// WBS code n.k that maps to a milestone (shared numbering, see toDTO)
	if strings.Contains(ref, ".") {
		if id := a.milestoneByWBS(ref); id != "" {
			return id
		}
	}
	return ""
}

// resolveSectionRef resolves a section reference (a name, or a 1-based section
// number) to the section name, or "" if not found.
func (a *App) resolveSectionRef(ref string) string {
	ref = strings.TrimSpace(ref)
	// exact cluster name match
	for i := range a.project.Packages {
		if a.project.Packages[i].Cluster == ref {
			return ref
		}
	}
	if a.project.SectionColors != nil {
		if _, ok := a.project.SectionColors[ref]; ok {
			return ref
		}
	}
	// numeric section number — use the SAME WP-n numbering shown in the UI
	// (lane labels / DTO SectionNums) so "select 3" matches what the user sees.
	nums := a.wpSectionNums()
	for name, n := range nums {
		if fmt.Sprintf("%d", n) == ref {
			return name
		}
	}
	return ""
}

// SetZoom persists the canvas zoom level in the model so it is saved with the
// project and restored on reopen. Accepts days|weeks|months|year|window.
func (a *App) SetZoom(zoom string) (*ProjectDTO, error) {
	switch zoom {
	case "days", "weeks", "months", "year", "window", "":
		a.project.Zoom = zoom
		return toDTO(a.project), nil
	}
	return nil, fmt.Errorf("invalid zoom %q", zoom)
}

// NewProject starts a fresh empty project (the only "blank slate" action). The
// "must save, no discard" rule: if the current project has unsaved changes it
// is ALWAYS saved first — to its known path, or to a Downloads fallback when it
// was never saved — so nothing is lost and the UI needs no save/discard dialog.
// The `autosave` parameter is kept for signature/test compatibility; saving the
// dirty old project is now unconditional.
func (a *App) NewProject(autosave bool) (*ProjectDTO, error) {
	if a.IsDirty() {
		if _, err := a.SaveSmart(); err != nil {
			return nil, fmt.Errorf("could not save the current project before New: %w", err)
		}
	}
	p := plan.NewProject("Untitled", plan.UnitDays)
	p.EnsureRange()
	a.project = p
	a.base = ""
	a.paths = plan.Paths{}
	a.undoStack = nil
	// reset the per-session cursors so stale state from the previous project
	// (e.g. the last-used section) does not leak into the new one.
	a.current = ""
	a.currentSection = ""
	a.selKind = ""
	a.selID = ""
	a.drillID = ""
	a.markSaved() // a fresh project starts "clean"
	return toDTO(p), nil
}

// SetProjectRange sets the project Start and End milestones (requirement:
// the project has an overall start and end that bound the visible window).
func (a *App) SetProjectRange(start, end string) (*ProjectDTO, error) {
	s, err := parseDate(start)
	if err != nil {
		return nil, fmt.Errorf("start: %w", err)
	}
	e, err := parseDate(end)
	if err != nil {
		return nil, fmt.Errorf("end: %w", err)
	}
	if !e.After(s) {
		return nil, fmt.Errorf("end must be after start")
	}
	a.project.SetRange(s, e)
	return toDTO(a.project), nil
}

// SetProjectDetails sets the plan name (<=16 chars), description, and the
// Start/End range in one call (the project-details dialog).
func (a *App) SetProjectDetails(name, description, start, end string) (*ProjectDTO, error) {
	name = strings.TrimSpace(name)
	if name != "" {
		if len([]rune(name)) > MaxPlanName {
			return nil, fmt.Errorf("plan name must be at most %d characters", MaxPlanName)
		}
		a.project.Title = name
	}
	a.project.Description = description
	if start != "" && end != "" {
		s, err := parseDate(start)
		if err != nil {
			return nil, fmt.Errorf("start: %w", err)
		}
		e, err := parseDate(end)
		if err != nil {
			return nil, fmt.Errorf("end: %w", err)
		}
		if !e.After(s) {
			return nil, fmt.Errorf("end must be after start")
		}
		a.project.SetRange(s, e)
	}
	return toDTO(a.project), nil
}

// GetProject returns the current project as a DTO for the UI.
func (a *App) GetProject() *ProjectDTO { return toDTO(a.project) }

// SetConfig updates project-wide unit and hours/day (requirement R-3).
func (a *App) SetConfig(unit string, hoursPerDay int) *ProjectDTO {
	a.project.Unit = plan.Unit(unit)
	if hoursPerDay > 0 {
		a.project.HoursPerDay = hoursPerDay
	}
	return toDTO(a.project)
}

// SetLanguage sets the GUI language ("en"|"de"), persisted with the project.
// The CLI/chat language is unaffected (always English).
func (a *App) SetLanguage(code string) *ProjectDTO {
	a.project.Language = plan.NormalizeLanguage(code)
	return toDTO(a.project)
}

// LangOptionDTO is one entry of the language pulldown (code + display name).
type LangOptionDTO struct {
	Code string `json:"code"`
	ISO  string `json:"iso"`
	Name string `json:"name"`
}

// Languages returns the supported GUI languages for the settings pulldown.
func (a *App) Languages() []LangOptionDTO {
	out := []LangOptionDTO{}
	for _, l := range plan.Languages {
		out = append(out, LangOptionDTO{Code: l.Code, ISO: l.ISO, Name: l.Name})
	}
	return out
}

// RelationshipRuleDTO is one row of the relationship-rules help table, with the
// abbreviation + name in the requested GUI language.
type RelationshipRuleDTO struct {
	Kind    string `json:"kind"`    // canonical kind FS|SS|FF|SF|MS|ME|MM
	Abbr    string `json:"abbr"`    // language abbreviation (EA/AA/... for de)
	Name    string `json:"name"`    // localized relationship name
	Rule    string `json:"rule"`    // the timing rule, e.g. "Start(B) >= End(A)"
	Example string `json:"example"` // localized example
}

// RelationshipRules returns the 7 dependency relationship rules localized for
// the given GUI language (the abbreviation follows the language: English uses
// FS/SS/FF/SF/MS/ME/MM, German uses EA/AA/EE/AE/MA/ME/MM).
func (a *App) RelationshipRules(lang string) []RelationshipRuleDTO {
	return relationshipRules(plan.NormalizeLanguage(lang))
}

// GUIStrings returns the translated GUI label bundle for a language (used by
// the help pages and localized UI chrome). The CLI/chat is never translated.
func (a *App) GUIStrings(lang string) map[string]string {
	return guiStrings(plan.NormalizeLanguage(lang))
}

// CommandHelp returns the command (CLI) help. It is ALWAYS English because the
// command language is never translated; the `lang` arg is accepted for a
// uniform help API but ignored for the body (only an English header).
func (a *App) CommandHelp(lang string) string {
	return a.ChatHelp()
}

// SetAutosaveConfig sets the autosave on/off switch, the interval (minutes; 0 =
// every change, 5..99 periodic, 1..4 clamp up to 5), the autosave-on-exit flag,
// and the versioning options (keep rotated numbered copies; count clamped
// 5..10). All are persisted with the project (.ptplan). Returns the updated
// project.
func (a *App) SetAutosaveConfig(enabled bool, minutes int, onExit bool, saveVersions bool, versionCount int) *ProjectDTO {
	a.project.AutosaveEnabled = enabled
	a.project.AutosaveMinutes = plan.ClampAutosaveMinutes(minutes)
	a.project.AutosaveOnExit = onExit
	a.project.SaveVersions = saveVersions
	a.project.VersionCount = plan.ClampVersionCount(versionCount)
	return toDTO(a.project)
}

// fallbackSavePath derives a path for a never-saved project: <Downloads>/<title
// or Untitled>.ptplan. Used by SaveSmart and the quit hook so New/Open/exit can
// always save without a dialog (the "must save, no discard" rule).
func (a *App) fallbackSavePath() string {
	name := strings.TrimSpace(a.project.Title)
	if name == "" {
		name = "Untitled"
	}
	return filepath.Join(downloadsDir(), name+plan.ProjectFileExt)
}

// SaveSmart saves WITHOUT a dialog: to the current path if known, otherwise to
// a fallback location (Downloads/<title>.ptplan). This backs the New/Open
// "must save first" rule and the quick-save disk button once a path exists.
// Returns the path written.
func (a *App) SaveSmart() (string, error) {
	base := a.base
	if base == "" {
		base = a.fallbackSavePath()
	}
	base = jsonPathFor(base)
	if err := a.project.SaveJSON(base); err != nil {
		return "", err
	}
	a.base = base
	a.markSaved()
	rememberDir(filepath.Dir(base))
	rememberRecent(base)
	return base, nil
}

// AutosaveTick is called periodically by the UI timer. It saves ONLY when the
// project is dirty AND already has a path (autosave never invents a location
// silently mid-session; the New/Open/exit gate covers the no-path case).
// Returns the path saved, or "" when nothing was written.
func (a *App) AutosaveTick() (string, error) {
	if !a.project.AutosaveEnabled || a.base == "" || !a.IsDirty() {
		return "", nil
	}
	// Versioning: before overwriting, rotate the PREVIOUS on-disk content into a
	// numbered sibling (newest = highest number), keeping at most VersionCount.
	if a.project.SaveVersions {
		if _, err := plan.RotateVersion(a.base, a.project.VersionCount); err != nil {
			return "", fmt.Errorf("version rotation failed: %w", err)
		}
	}
	if err := a.project.SaveJSON(a.base); err != nil {
		return "", err
	}
	a.markSaved()
	return a.base, nil
}

// SaveOnExit saves the project when it is dirty and autosave-on-exit is enabled.
// Used by the window-close / shutdown hook. Best-effort: uses the known path or
// the fallback. Returns the path written, or "" when nothing was saved.
func (a *App) SaveOnExit() string {
	if !a.project.AutosaveOnExit || !a.IsDirty() {
		return ""
	}
	path, err := a.SaveSmart()
	if err != nil {
		return ""
	}
	return path
}

// --- Baselines (R-13.1) ---

// BaselineInfoDTO is a lightweight baseline descriptor for the UI.
type BaselineInfoDTO struct {
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
	TotalEst  string `json:"totalEst"`
	Tasks     int    `json:"tasks"`
}

// VarianceDTO is a per-task variance row (display strings).
type VarianceDTO struct {
	WBS       string `json:"wbs"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	StartDays int    `json:"startDays"`
	EndDays   int    `json:"endDays"`
	EstDelta  string `json:"estDelta"`
	Status    string `json:"status"` // "", "added", "missing"
}

// CaptureBaseline freezes the current plan as a named baseline.
func (a *App) CaptureBaseline(name string) ([]BaselineInfoDTO, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("baseline name required")
	}
	b := a.project.CaptureBaseline(name)
	a.project.Baselines = append(a.project.Baselines, b)
	return a.ListBaselines(), nil
}

// ListBaselines returns all baselines (descriptors only).
func (a *App) ListBaselines() []BaselineInfoDTO {
	out := []BaselineInfoDTO{}
	for i := range a.project.Baselines {
		b := &a.project.Baselines[i]
		out = append(out, BaselineInfoDTO{
			Name:      b.Name,
			CreatedAt: b.CreatedAt.Format("2006-01-02 15:04"),
			TotalEst:  a.project.Format(b.TotalEst),
			Tasks:     len(b.Tasks),
		})
	}
	return out
}

// DeleteBaseline removes a baseline by name.
func (a *App) DeleteBaseline(name string) []BaselineInfoDTO {
	out := a.project.Baselines[:0]
	for _, b := range a.project.Baselines {
		if b.Name != name {
			out = append(out, b)
		}
	}
	a.project.Baselines = out
	return a.ListBaselines()
}

// CompareBaseline returns per-task variance of the live plan vs a baseline,
// with WBS codes and names resolved for display.
func (a *App) CompareBaseline(name string) ([]VarianceDTO, error) {
	var bl *plan.Baseline
	for i := range a.project.Baselines {
		if a.project.Baselines[i].Name == name {
			bl = &a.project.Baselines[i]
		}
	}
	if bl == nil {
		return nil, fmt.Errorf("baseline %q not found", name)
	}
	// build id -> (wbs, name) lookup from the current DTO numbering
	dto := toDTO(a.project)
	wbs := map[string]string{}
	nm := map[string]string{}
	for _, w := range dto.Packages {
		wbs[w.ID] = w.DisplayID
		nm[w.ID] = w.Name
		for _, d := range w.Details {
			wbs[d.ID] = d.DisplayID
			nm[d.ID] = d.Name
		}
	}
	var out []VarianceDTO
	for _, v := range a.project.CompareBaseline(bl) {
		status := ""
		if v.Added {
			status = "added"
		} else if v.Missing {
			status = "missing"
		}
		out = append(out, VarianceDTO{
			WBS:       wbs[v.ID],
			Name:      nm[v.ID],
			Kind:      v.Kind,
			StartDays: v.StartDays,
			EndDays:   v.EndDays,
			EstDelta:  a.project.Format(v.EstDelta),
			Status:    status,
		})
	}
	return out, nil
}

// SetArrowStyle sets the dependency-arrow routing style: "spline" (rounded,
// default) or "square" (orthogonal).
func (a *App) SetArrowStyle(style string) (*ProjectDTO, error) {
	switch style {
	case plan.ArrowSpline, plan.ArrowSquare:
		a.project.ArrowStyle = style
	default:
		return nil, fmt.Errorf("arrow style must be spline|square")
	}
	return toDTO(a.project), nil
}

// SetViewOptions sets the main-canvas view preferences: whether to draw
// level-2 detail bars (showDetails), whether role mode expands each action
// into per-role sub-rows (showRoleActions), and whether the top "Phases" lane
// is shown (showPhases), and whether progress markers/contour curves are drawn
// (showProgress). showDetails/showRoleActions/showProgress default off;
// showPhases defaults on.
func (a *App) SetViewOptions(showDetails, showRoleActions, showPhases, showProgress bool) *ProjectDTO {
	a.project.ShowDetails = showDetails
	a.project.ShowRoleActions = showRoleActions
	a.project.ShowPhases = showPhases
	a.project.ShowProgress = showProgress
	return toDTO(a.project)
}

// SetRoleLabelOptions sets the two modifiers of "Show role actions" that
// control how much each role sub-row's label shows: showPercent appends the
// assignment percent, showDays appends the role's effort share ("= N d").
// With both off (the default) a role sub-row shows the role NAME only.
// Persisted with the project.
func (a *App) SetRoleLabelOptions(showPercent, showDays bool) *ProjectDTO {
	a.project.ShowRolePercent = showPercent
	a.project.ShowRoleDays = showDays
	return toDTO(a.project)
}

// SetReportConfig sets the export/report options: format ("pdf"|"png"), whether
// to draw the today line, the gridline interval ("day"|"week"|"month"), and
// whether notes are included. Persisted with the project. The today line, grid,
// and notes also affect the LIVE canvas so the export matches the screen.
func (a *App) SetReportConfig(format string, todayLine bool, grid string, showNotes bool) *ProjectDTO {
	a.project.ReportFormat = plan.NormalizeReportFormat(format)
	a.project.ReportTodayLine = todayLine
	a.project.ReportGrid = plan.NormalizeReportGrid(grid)
	a.project.ReportShowNotes = showNotes
	return toDTO(a.project)
}

// SetShowWeekNumbers switches the calendar week numbers (ISO 8601) in the date
// row on or off. Persisted with the project.
func (a *App) SetShowWeekNumbers(on bool) *ProjectDTO {
	a.project.ShowWeekNumbers = on
	return toDTO(a.project)
}

// MaxPlanName is the maximum length of a plan name/title (requirement: a plan
// must have a name, capped at 16 characters).
const MaxPlanName = 16

// SetTitle sets the plan name, enforcing the 16-character limit. Leading and
// trailing whitespace is trimmed; an over-long name is rejected so the caller
// can surface the limit.
func (a *App) SetTitle(name string) (*ProjectDTO, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("plan name must not be empty")
	}
	if len([]rune(name)) > MaxPlanName {
		return nil, fmt.Errorf("plan name must be at most %d characters", MaxPlanName)
	}
	a.project.Title = name
	return toDTO(a.project), nil
}

// --- Rendering / reporting bindings ---

// RenderPNG renders the current plan to outPath via embedded gmaid.
func (a *App) RenderPNG(outPath string, scale float64) error {
	if scale <= 0 {
		scale = 2
	}
	return a.project.RenderPNG(outPath, "", scale)
}

// ExportPDFReport writes the combined PDF report (gantt + people tables).
func (a *App) ExportPDFReport(outPath string) error {
	return a.project.WritePDFReport(outPath, "")
}

// reportDetailView is one per-work-package detail (drill) image supplied by the
// frontend for the report. Name is the WBS label + work-package name; PNG is
// the base64 data-URL payload (just the base64 after the comma).
type reportDetailView struct {
	Name string `json:"name"`
	PNG  string `json:"png"`
}

// ExportReport writes the report using the frontend-rendered Gantt images so
// the export matches the live canvas EXACTLY (notes, spline/square arrows, WP-n
// section headers, today line, grid). overviewBase64 is the whole-project
// overview image (details compacted); detailsJSON is a JSON array of
// {name, png} objects, one per work package that has detail packages, each
// holding that work package's detail (drill) view. The report ALWAYS prints the
// details: the overview on its own page, then each detail view on its own page.
// The format follows the project's ReportFormat ("pdf" embeds the images +
// tables; "png" writes the overview image alone). Shows a native save dialog.
// Returns the path.
func (a *App) ExportReport(overviewBase64, detailsJSON string) (string, error) {
	overview, details, err := decodeReportImages(overviewBase64, detailsJSON)
	if err != nil {
		return "", err
	}

	format := plan.NormalizeReportFormat(a.project.ReportFormat)
	ext, filter, filterName := ".pdf", "*.pdf", "PDF (*.pdf)"
	if format == "png" {
		ext, filter, filterName = ".png", "*.png", "PNG (*.png)"
	}
	name := strings.TrimSpace(a.project.Title)
	if name == "" {
		name = "Untitled"
	}
	dlg := a.app.Dialog.SaveFile().
		SetMessage("Export report").
		SetFilename(name+"-report"+ext).
		AddFilter(filterName, filter)
	if ld := lastDir(); ld != "" {
		dlg = dlg.SetDirectory(ld)
	}
	path, err := dlg.PromptForSingleSelection()
	if err != nil || path == "" {
		return "", err
	}
	if !strings.HasSuffix(strings.ToLower(path), ext) {
		path += ext
	}
	if werr := a.writeReport(path, format, overview, details); werr != nil {
		return "", werr
	}
	rememberDir(filepath.Dir(path))
	return path, nil
}

// decodeReportImages decodes the frontend-supplied overview + detail-view images
// (base64 payloads) into raw PNG bytes for the report builder.
func decodeReportImages(overviewBase64, detailsJSON string) ([]byte, []plan.ReportImage, error) {
	overview, err := base64.StdEncoding.DecodeString(overviewBase64)
	if err != nil {
		return nil, nil, fmt.Errorf("decode overview report image: %w", err)
	}
	var views []reportDetailView
	if strings.TrimSpace(detailsJSON) != "" {
		if jerr := json.Unmarshal([]byte(detailsJSON), &views); jerr != nil {
			return nil, nil, fmt.Errorf("decode report detail views: %w", jerr)
		}
	}
	details := make([]plan.ReportImage, 0, len(views))
	for _, v := range views {
		png, derr := base64.StdEncoding.DecodeString(v.PNG)
		if derr != nil {
			return nil, nil, fmt.Errorf("decode detail image %q: %w", v.Name, derr)
		}
		details = append(details, plan.ReportImage{Caption: v.Name, PNG: png})
	}
	return overview, details, nil
}

// writeReport writes the report to path in the given format ("pdf" embeds the
// images + tables; "png" writes the overview image alone).
func (a *App) writeReport(path, format string, overview []byte, details []plan.ReportImage) error {
	if format == "png" {
		// A single PNG cannot hold multiple views; write the overview.
		return os.WriteFile(path, overview, 0o644)
	}
	return a.project.WritePDFReportWithViews(path, overview, details)
}

// ExportPDFDialog shows a native save picker and writes the PDF report there.
// Legacy path (gmaid-rendered image); kept for compatibility. Prefer
// ExportReport which uses the live canvas image.
func (a *App) ExportPDFDialog() (string, error) {
	path, err := a.app.Dialog.SaveFile().
		SetMessage("Export PDF report").
		SetFilename(a.project.Title+"-report.pdf").
		AddFilter("PDF (*.pdf)", "*.pdf").
		PromptForSingleSelection()
	if err != nil || path == "" {
		return "", err
	}
	if err := a.project.WritePDFReport(path, ""); err != nil {
		return "", err
	}
	return path, nil
}

// Markdown returns the generated read-only projection (for preview in-app).
func (a *App) Markdown() string { return a.project.Markdown() }

// Warnings returns consistency warnings for the current project.
func (a *App) Warnings() []string { return a.project.Validate() }
