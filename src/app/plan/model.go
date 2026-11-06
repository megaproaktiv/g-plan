// Package plan is the pure-Go core of pt-plan: the project model, file I/O
// (.mmd graphic source + .roles.json), the generated .md projection, and the
// embedded gmaid rendering/reporting pipeline. It has no Wails dependency.
//
// Estimation arithmetic is integer-based (requirement R-4.3) to avoid any
// floating-point error. Values are stored in the SMALLEST natural sub-unit of
// the project's display unit, so every conversion is exact integer division:
//
//   - Day mode:  display unit = day,  smallest storable amount = 1 hour.
//     Estimations are stored as integer HOURS. 1 day == HoursPerDay
//     hours (project config, default 8). A whole-day level-1 value
//     is a multiple of HoursPerDay.
//   - Hour mode: display unit = hour, smallest storable amount = 1 minute.
//     Estimations are stored as integer MINUTES. 1 hour == 60 min.
//
// The stored integer type is Base (see base.go). Level-2 detail packages may
// carry two-decimal display fractions (R-4.2); those map onto whole base units
// (e.g. day mode 1,25 d = 10 h; hour mode 1,5 h = 90 min). Conversion to a
// decimal display string happens only at render/serialize time, always via
// integer math.
package plan

import (
	"fmt"
	"time"
)

// Unit is the project-wide estimation unit (requirement R-3.1).
type Unit string

const (
	UnitDays  Unit = "days"
	UnitHours Unit = "hours"
)

// Base is an estimation amount stored in MINUTES, always — independent of the
// project's display unit (see base.go). This makes switching days<->hours a
// pure display change. All estimation math is done on Base (minute) values.
type Base int64

// DefaultHoursPerDay is the working hours in one day used for day-mode storage
// when the project does not specify one.
const DefaultHoursPerDay = 8

// DefaultAutosaveMinutes is the default autosave interval. 0 means "after every
// change"; otherwise the allowed range is 5..99 (values 1..4 clamp up to 5).
const DefaultAutosaveMinutes = 5

// ClampAutosaveMinutes normalizes an autosave interval: negatives -> 0, 1..4
// -> 5, and anything above 99 -> 99. 0 (every change) and 5..99 pass through.
func ClampAutosaveMinutes(m int) int {
	if m <= 0 {
		return 0
	}
	if m < 5 {
		return 5
	}
	if m > 99 {
		return 99
	}
	return m
}

// DefaultVersionCount is the default number of rotated autosave versions kept.
const DefaultVersionCount = 5

// ClampVersionCount normalizes the kept-version count to the 5..10 range.
func ClampVersionCount(n int) int {
	if n < 5 {
		return 5
	}
	if n > 10 {
		return 10
	}
	return n
}

// NormalizeReportFormat returns "pdf" or "png" (default "pdf").
func NormalizeReportFormat(s string) string {
	if s == "png" {
		return "png"
	}
	return "pdf"
}

// NormalizeReportGrid returns "day" | "week" | "month" (default "month").
func NormalizeReportGrid(s string) string {
	switch s {
	case "day", "week", "month":
		return s
	}
	return "month"
}

// EstimationMode controls how a work package's level-1 estimation relates to
// its detail packages (requirement R-5.4).
type EstimationMode string

const (
	// ModeFixed: the level-1 estimation is authoritative and is distributed
	// down to the detail packages.
	ModeFixed EstimationMode = "fixed"
	// ModeDerived: the level-1 estimation is the sum of the detail-package
	// estimations (roll-up).
	ModeDerived EstimationMode = "derived"
)

// DateLayout is the storage/serialization date format (matches Mermaid
// dateFormat YYYY-MM-DD).
const DateLayout = "2006-01-02"

// Milestone is a zero-duration marker on the timeline. User milestones live in
// a section (Cluster) and may carry a color; the special project Start/End
// milestones (m_start/m_end) have no cluster and render in the Milestones view.
type Milestone struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Date    time.Time `json:"date"`
	Cluster string    `json:"cluster,omitempty"` // section this milestone belongs to
	Color   string    `json:"color,omitempty"`   // optional marker color (hex)
}

// Phase is a coarse program band (e.g. "1 Foundation and pilot").
type Phase struct {
	ID    string    `json:"id"`
	Name  string    `json:"name"`
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// DetailPackage is a level-2 task inside a work package (requirement R-5.1).
// Est is in Base and may carry two decimals (R-4.2). A detail has a title,
// an optional description, an estimation, and an optional role (chosen from
// the project roles).
type DetailPackage struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Desc      string    `json:"desc,omitempty"`
	Role      string    `json:"role,omitempty"`
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
	Est       Base      `json:"est"`                 // project base units (hours in day mode, minutes in hour mode)
	Summary   bool      `json:"summary,omitempty"`   // rollup/Sammelvorgang bar (same graphic as a summary action)
	FontColor string    `json:"fontColor,omitempty"` // optional label color (hex); empty = auto-contrast
	TextPos   string    `json:"textPos,omitempty"`   // "" / "auto" | center | left | right | up | down
}

// WorkPackage is a level-1 bar with a duration AND an estimation (R-5.2).
// Duration (Start..End) is independent of Est (R-4.4).
type WorkPackage struct {
	ID    string    `json:"id"`
	Name  string    `json:"name"`
	Desc  string    `json:"desc,omitempty"`
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Est   Base      `json:"est"` // level-1: whole days (multiple of HoursPerDay) in day mode
	// RealEst is the Real_Effort booked so far, in the same minute Base as Est.
	// A file without the key yields 0, which is the required default.
	RealEst Base `json:"realEst,omitempty"`
	// Contour is the Work_Contour. An absent key yields "", which
	// NormalizeContour maps to Flat; new actions are set to Flat explicitly.
	Contour Contour `json:"contour,omitempty"`
	// ExternalID identifies this action in the external time-tracking system:
	// bookings whose text contains it are counted as real effort of this
	// action. At most MaxExternalIDLen characters; unique within a project.
	ExternalID string         `json:"externalId,omitempty"`
	Mode       EstimationMode `json:"mode"`              // fixed | derived
	Summary    bool           `json:"summary,omitempty"` // rollup/Summiervorgang bar (different graphic)
	// DetailsHidden hides THIS action's level-3 detail bars in the main view even
	// when the project-wide Show Details is on (a per-action show/hide toggle).
	// Default false = details are shown when Show Details is on.
	DetailsHidden bool   `json:"detailsHidden,omitempty"`
	Color         string `json:"color,omitempty"` // optional bar color (hex); empty = default
	// FontColor is the optional label color (hex). Empty means auto-contrast:
	// the renderer picks black or white based on the bar background so the
	// label stays readable on dark colors.
	FontColor string `json:"fontColor,omitempty"`
	// TextPos controls where the label is drawn relative to the bar:
	// "" or "auto" = inside if it fits, else to the right; or an explicit
	// "center" | "left" | "right" | "up" | "down".
	TextPos string `json:"textPos,omitempty"`
	// NoteVisible, when true, draws the action's Description as a small yellow
	// sticky note near the bar, connected to the nearest action edge by a plain
	// (arrowless) line. NotePos places the note relative to the bar (same
	// vocabulary as TextPos; default "up").
	NoteVisible bool   `json:"noteVisible,omitempty"`
	NotePos     string `json:"notePos,omitempty"`
	Cluster     string `json:"cluster,omitempty"` // optional grouping (R-5.3)
	// SameLine, when true, lets this action share a swimlane row with the
	// previous action in the same cluster instead of occupying its own row.
	SameLine bool            `json:"sameLine,omitempty"`
	Details  []DetailPackage `json:"details,omitempty"`
}

// DetailSum returns the sum of the detail-package estimations (Base).
func (w *WorkPackage) DetailSum() Base {
	var s Base
	for i := range w.Details {
		s += w.Details[i].Est
	}
	return s
}

// EffectiveEst returns the work package's estimation honoring its mode:
// derived => sum of details; fixed => the stored level-1 value (R-5.6).
func (w *WorkPackage) EffectiveEst() Base {
	if !w.TracksEffort() {
		return 0 // contour None: no estimation
	}
	if w.Mode == ModeDerived {
		return w.DetailSum()
	}
	return w.Est
}

// WBSItem is one entry (action or milestone) in a section's shared n.k
// numbering sequence.
type WBSItem struct {
	Kind string // "action" | "milestone"
	ID   string
	Date time.Time
}

// SectionSequence returns a cluster's actions and milestones ordered by date
// (stable), used to assign the shared n.k display ids. The project Start/End
// milestones (m_start/m_end) are excluded (they are the Milestones view).
func (p *Project) SectionSequence(cluster string) []WBSItem {
	var items []WBSItem
	for i := range p.Packages {
		if p.Packages[i].Cluster == cluster {
			items = append(items, WBSItem{Kind: "action", ID: p.Packages[i].ID, Date: p.Packages[i].Start})
		}
	}
	for i := range p.Milestones {
		m := &p.Milestones[i]
		if m.ID == MilestoneStartID || m.ID == MilestoneEndID {
			continue
		}
		if m.Cluster == cluster {
			items = append(items, WBSItem{Kind: "milestone", ID: m.ID, Date: m.Date})
		}
	}
	sortWBSByDate(items)
	return items
}

func sortWBSByDate(items []WBSItem) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j-1].Date.After(items[j].Date); j-- {
			items[j-1], items[j] = items[j], items[j-1]
		}
	}
}

// WorkdaysBetween returns the number of working days (Mon-Fri, weekends
// excluded) in the inclusive window [start, end]. No holiday calendar is
// applied yet (future version). Returns 0 if end is before start.
func WorkdaysBetween(start, end time.Time) int {
	start = start.Truncate(24 * time.Hour)
	end = end.Truncate(24 * time.Hour)
	if end.Before(start) {
		return 0
	}
	n := 0
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		switch d.Weekday() {
		case time.Saturday, time.Sunday:
			// skip weekends
		default:
			n++
		}
	}
	return n
}

// Overallocated reports whether a work package's effort (in display days)
// exceeds the working days available in its start..end window. Only meaningful
// in day mode; returns (0,0,false) otherwise or when there is no effort.
func (p *Project) Overallocated(w *WorkPackage) (effortDays int, workdays int, over bool) {
	if p.Unit != UnitDays {
		return 0, 0, false
	}
	eff := w.EffectiveEst()
	if eff <= 0 {
		return 0, 0, false
	}
	// effort in whole display days (round up partial days)
	perDay := p.minutesPerDisplay()
	if perDay <= 0 {
		return 0, 0, false
	}
	effortDays = int((int64(eff) + perDay - 1) / perDay)
	workdays = WorkdaysBetween(w.Start, w.End)
	return effortDays, workdays, effortDays > workdays
}

// RoleCapacity models whether the roles assigned to a work package can deliver
// its effort within its start..end window.
//
// A role's percentage is NOT a share of the effort — it is the fraction of the
// working day the role spends on this action. So a role at p% contributes
// (p/100) * HoursPerDay hours on every working day (Mon-Fri) in the window.
//
// It returns, all in stored MINUTES:
//   - effort:    the action's effective estimation
//   - perDay:    the combined role capacity per working day (sum of p% * H)
//   - capacity:  perDay * number of working days in the window
//   - oneRoleMax: H * workdays — the most a single 100% role could deliver in
//     the window (used for the "does the effort fit the window at all" check)
//   - fits:      capacity >= effort (the assigned roles cover the effort)
//   - fitsWindow: effort <= oneRoleMax (the effort can be done in the window at
//     100% by one role — i.e. the window is long enough to hold the effort)
func (p *Project) RoleCapacity(w *WorkPackage) (effort, perDay, capacity, oneRoleMax Base, fits, fitsWindow bool) {
	effort = w.EffectiveEst()
	workdays := int64(WorkdaysBetween(w.Start, w.End))
	minsPerHour := int64(60)
	hpd := int64(p.hpd())
	dayMins := hpd * minsPerHour // minutes in one working day
	// sum of role percentages assigned to this work package
	var pctSum int64
	for i := range p.Roles {
		for _, as := range p.Roles[i].Assignments {
			if as.WorkPackageID == w.ID {
				pctSum += int64(as.Percent)
			}
		}
	}
	// perDay capacity in minutes = (pctSum/100) * dayMins (integer math)
	perDay = Base(pctSum * dayMins / 100)
	capacity = Base(int64(perDay) * workdays)
	oneRoleMax = Base(dayMins * workdays)
	fits = effort <= 0 || int64(capacity) >= int64(effort)
	fitsWindow = effort <= 0 || int64(oneRoleMax) >= int64(effort)
	return
}

// held level-1 estimation and the sum of its details (positive => details
// under-allocate, negative => details over-allocate). For derived mode it is
// always zero by construction. Used by the UI to surface over/under allocation
// (R-5.6).
func (w *WorkPackage) Reconcile() Base {
	if w.Mode == ModeDerived {
		return 0
	}
	return w.Est - w.DetailSum()
}

// Dependency is a typed link between two tasks (requirement R-2.1.2 / gmaid
// link directive). Kind is one of FS, SS, FF, SF, MS, ME, MM (see DepViolated
// for the timing rules). FromAnchor/ToAnchor record which anchor (l/r/t/b) the
// user drew from, so the arrow starts/ends there.
type Dependency struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Kind       string `json:"kind"` // FS|SS|FF|SF|MS|ME|MM
	FromAnchor string `json:"fromAnchor,omitempty"`
	ToAnchor   string `json:"toAnchor,omitempty"`
	Color      string `json:"color,omitempty"`
}

// Assignment references a role's share of a work package (requirements R-6.2,
// R-6.4). Percent is an integer percent [0..100]. Consumed is in Centi.
type Assignment struct {
	WorkPackageID string `json:"wp"`
	Percent       int    `json:"percent"`
	Consumed      Base   `json:"consumed,omitempty"`
}

// Role is a named person/skill with a description (requirement R-6.1). Color
// is an optional hex color used to tint the role's bars/sub-rows; each role
// should have a distinct color.
type Role struct {
	// ID is the role's stable number (shown as R<ID>). It is assigned once,
	// unique among the existing roles, and survives renames and the deletion of
	// other roles. Files predating it get IDs assigned on load (EnsureRoleIDs).
	ID          int          `json:"id,omitempty"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Color       string       `json:"color,omitempty"`
	Assignments []Assignment `json:"assignments,omitempty"`
}

// Project is a whole plan: graphic content (from the .mmd) plus roles (from the
// .roles.json). The .md projection is generated from this.
type Project struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Unit        Unit   `json:"unit"`
	HoursPerDay int    `json:"hoursPerDay"` // day-mode storage base (default 8)
	// Language is the GUI language ("en" | "de"). The CLI/chat stays English
	// regardless; this only affects GUI labels, help, and number/date display.
	Language   string        `json:"language,omitempty"`
	DateFormat string        `json:"dateFormat"`
	AxisFormat string        `json:"axisFormat"`
	TickEvery  string        `json:"tickInterval,omitempty"`
	Milestones []Milestone   `json:"-"`
	Phases     []Phase       `json:"-"`
	Packages   []WorkPackage `json:"-"`
	Deps       []Dependency  `json:"-"`
	Roles      []Role        `json:"roles"`
	// SectionOrder is the explicit order of the sections (WP-1 first). It lets
	// even an empty section be moved up and down; sections it does not mention
	// keep their natural position after it (see SectionNames).
	SectionOrder []string `json:"-"`
	// SectionColors maps a cluster/section name to a hex color (optional).
	SectionColors map[string]string `json:"-"`
	// ArrowStyle is the dependency-arrow routing style: "spline" (rounded,
	// default) or "square" (orthogonal elbows).
	ArrowStyle string `json:"-"`
	// ShowDetails controls whether level-2 detail bars are drawn as sub-rows
	// in the main canvas. Off by default (R-view). View preference.
	ShowDetails bool `json:"-"`
	// ShowRoleActions controls whether, in role mode, each action is expanded
	// in the main canvas with one sub-row per attached role showing that
	// role's share. Off by default. View preference.
	ShowRoleActions bool `json:"-"`
	// ShowRolePercent and ShowRoleDays are modifiers of ShowRoleActions that
	// control how much each role sub-row's label shows. By default (both off)
	// the label is the role NAME only; ShowRolePercent adds the assignment
	// percent; ShowRoleDays adds the role's effort share ("= N d"). View
	// preferences (both off by default).
	ShowRolePercent bool `json:"-"`
	ShowRoleDays    bool `json:"-"`
	// ShowPhases controls whether the top "Phases" section is drawn. Phases is
	// the first lane and carries no WP number. On by default. View preference.
	ShowPhases bool `json:"-"`
	// ShowProgress controls whether progress markers and contour curves are
	// drawn in the action bars. Off by default (NewProject leaves it false).
	// View preference.
	ShowProgress bool `json:"-"`
	// Monitor is the monitoring (external system synchronization) setting.
	Monitor Monitor `json:"-"`
	// Zoom is the persisted canvas zoom level ("days"|"weeks"|"months"|"year"|
	// "window"). Saved with the model so a reopened project restores its view.
	Zoom string `json:"-"`
	// AutosaveEnabled is the master on/off switch for periodic autosave. When
	// off, no automatic timed save happens (the interval below is ignored);
	// manual Save and the New/Open "must-save" rule still apply. Default true.
	AutosaveEnabled bool `json:"-"`
	// AutosaveMinutes is the autosave interval in minutes: 0 = save after every
	// change (debounced in the UI); 5..99 = periodic. Values 1..4 are clamped
	// up to 5. Default 5.
	AutosaveMinutes int `json:"-"`
	// AutosaveOnExit, when true, saves the project (to its known path, or a
	// fallback) on quit if there are unsaved changes. Default true.
	AutosaveOnExit bool `json:"-"`
	// SaveVersions, when true, keeps rotated numbered copies of the file on each
	// autosave: before autosave overwrites the main file, its previous content
	// (if changed) is rotated into <name>.NNN.ptplan, newest = highest number.
	SaveVersions bool `json:"-"`
	// VersionCount is how many rotated versions to keep (5..10, default 5).
	VersionCount int `json:"-"`
	// --- report / export configuration ---
	// ReportFormat is the export target: "pdf" (combined report, default) or
	// "png" (just the Gantt image).
	ReportFormat string `json:"-"`
	// ReportTodayLine draws a vertical "today" marker in the chart. Default on.
	ReportTodayLine bool `json:"-"`
	// ReportGrid is the gridline interval shown in the chart: "day" | "week" |
	// "month" (default). Applies to the live canvas and the export.
	ReportGrid string `json:"-"`
	// ReportShowNotes controls whether the yellow sticky notes are included in
	// the export. Default on.
	ReportShowNotes bool `json:"-"`
	// ShowWeekNumbers draws the calendar week (ISO 8601) numbers in the date
	// row of the live canvas when there is room. Default off.
	ShowWeekNumbers bool `json:"-"`
	// Baselines are frozen snapshots for historical comparison (R-13.1).
	Baselines []Baseline `json:"-"`
}

// ArrowStyle values.
const (
	ArrowSpline = "spline"
	ArrowSquare = "square"
)

// RolePalette is the set of distinct default role colors, assigned in order so
// each new role gets a different one (requirement: roles have distinct colors).
var RolePalette = []string{
	"#d62728", "#2ca02c", "#1f77b4", "#ff7f0e", "#9467bd",
	"#8c564b", "#e377c2", "#17becf", "#bcbd22", "#7f7f7f",
}

// NextRoleColor returns a palette color not yet used by an existing role; if
// all are taken it cycles back to the start by count.
func (p *Project) NextRoleColor() string {
	used := map[string]bool{}
	for i := range p.Roles {
		if p.Roles[i].Color != "" {
			used[p.Roles[i].Color] = true
		}
	}
	for _, c := range RolePalette {
		if !used[c] {
			return c
		}
	}
	return RolePalette[len(p.Roles)%len(RolePalette)]
}

// NewProject returns an empty project with sensible defaults, including the
// project Start/End milestones that bound the visible timeline.
func NewProject(title string, unit Unit) *Project {
	p := &Project{
		Title:           title,
		Unit:            unit,
		HoursPerDay:     DefaultHoursPerDay,
		Language:        DefaultLanguage,
		DateFormat:      "YYYY-MM-DD",
		AxisFormat:      "%b %Y",
		TickEvery:       "1month",
		ArrowStyle:      ArrowSpline, // rounded splines by default
		ShowPhases:      true,        // the Phases lane is shown by default
		AutosaveEnabled: true,
		AutosaveMinutes: DefaultAutosaveMinutes,
		AutosaveOnExit:  true,
		SaveVersions:    false,
		VersionCount:    DefaultVersionCount,
		ReportFormat:    "pdf",
		ReportTodayLine: true,
		ReportGrid:      "month",
		ReportShowNotes: true,
	}
	p.EnsureRange()
	return p
}

// hpd returns the effective hours-per-day, defaulting when unset.
func (p *Project) hpd() int {
	if p.HoursPerDay <= 0 {
		return DefaultHoursPerDay
	}
	return p.HoursPerDay
}

// WorkPackage returns the work package with the given id, or nil.
func (p *Project) WorkPackage(id string) *WorkPackage {
	for i := range p.Packages {
		if p.Packages[i].ID == id {
			return &p.Packages[i]
		}
	}
	return nil
}

// DetailByID returns the detail package with the given id and its parent work
// package, or (nil, nil) if not found. Used for detail-level dependencies.
func (p *Project) DetailByID(id string) (*DetailPackage, *WorkPackage) {
	for i := range p.Packages {
		for j := range p.Packages[i].Details {
			if p.Packages[i].Details[j].ID == id {
				return &p.Packages[i].Details[j], &p.Packages[i]
			}
		}
	}
	return nil, nil
}

// TaskByID resolves an id to either a work package or a detail (whichever
// matches). Returns true if the id is any known task. Used to validate
// dependency endpoints that may be actions, details, OR milestones.
func (p *Project) TaskByID(id string) bool {
	if p.WorkPackage(id) != nil {
		return true
	}
	if d, _ := p.DetailByID(id); d != nil {
		return true
	}
	return p.MilestoneByID(id) != nil
}

// MilestoneByID returns the milestone with the given id, or nil.
func (p *Project) MilestoneByID(id string) *Milestone {
	for i := range p.Milestones {
		if p.Milestones[i].ID == id {
			return &p.Milestones[i]
		}
	}
	return nil
}

// TotalEst returns the project total estimation: the sum of every work
// package's effective estimation (requirement R-5.5 — calculated, never
// entered directly).
func (p *Project) TotalEst() Base {
	var s Base
	for i := range p.Packages {
		s += p.Packages[i].EffectiveEst()
	}
	return s
}

// RoleEst returns the derived total estimation for a role: the sum over its
// assignments of percent% of the referenced work package's effective
// estimation (requirement R-6.6). Integer math: wpEst*percent/100.
func (p *Project) RoleEst(r *Role) Base {
	var s Base
	for _, a := range r.Assignments {
		wp := p.WorkPackage(a.WorkPackageID)
		if wp == nil {
			continue
		}
		s += Base(int64(wp.EffectiveEst()) * int64(a.Percent) / 100)
	}
	return s
}

// AssignmentPercentByWP returns, per work package id, the sum of assigned
// role percentages. The UI uses this to flag work packages whose roles do not
// total 100% (requirement R-11.3).
func (p *Project) AssignmentPercentByWP() map[string]int {
	out := map[string]int{}
	for i := range p.Roles {
		for _, a := range p.Roles[i].Assignments {
			out[a.WorkPackageID] += a.Percent
		}
	}
	return out
}

// Validate returns a list of human-readable consistency warnings (non-fatal).
func (p *Project) Validate() []string {
	var warn []string
	if p.Unit != UnitDays && p.Unit != UnitHours {
		warn = append(warn, fmt.Sprintf("project unit %q is not days|hours", p.Unit))
	}
	for i := range p.Packages {
		wp := &p.Packages[i]
		if wp.Mode == ModeFixed && !p.IsWholeDisplayUnit(wp.Est) {
			warn = append(warn, fmt.Sprintf("WP %s: level-1 estimation %s is not a whole %s", wp.ID, p.Format(wp.Est), p.unitSingular()))
		}
		if wp.Mode == ModeFixed {
			if d := wp.Reconcile(); d != 0 {
				warn = append(warn, fmt.Sprintf("WP %s: details %s vs fixed %s (diff %s)",
					wp.ID, p.Format(wp.DetailSum()), p.Format(wp.Est), p.Format(d)))
			}
		}
	}
	// Role percentages are validated PER LINE only (R-6.8): a single
	// assignment may not exceed 100%. Percentages are NOT summed across role
	// lines, so multiple roles each at 100% on one work package is valid.
	for i := range p.Roles {
		for _, a := range p.Roles[i].Assignments {
			if a.Percent > 100 {
				warn = append(warn, fmt.Sprintf("WP %s: role %s assignment is %d%% (a single assignment may not exceed 100%%)", a.WorkPackageID, p.Roles[i].Name, a.Percent))
			}
		}
	}
	// Dependency violations (R-dep): for a Finish-Start (FS / end-begin) link,
	// the successor must not start before the predecessor finishes.
	for _, dep := range p.Deps {
		if p.DepViolated(dep) {
			warn = append(warn, fmt.Sprintf("Dependency %s -> %s (%s) is violated: %s starts before %s ends",
				dep.From, dep.To, depKindLabel(dep.Kind), dep.To, dep.From))
		}
	}
	// Capacity (R-cap): effort must fit the working days (Mon-Fri) in the
	// action's start..end window. If effort exceeds available workdays the
	// action is over-allocated.
	for i := range p.Packages {
		wp := &p.Packages[i]
		if ed, wd, over := p.Overallocated(wp); over {
			warn = append(warn, fmt.Sprintf("WP %s: effort %dd exceeds %d working day(s) in the window (over-allocated)", wp.ID, ed, wd))
		}
	}
	// Project window (R-range): every work package must lie within the
	// project's overall Start..End. Flag any action that starts before the
	// project start or ends after the project end.
	if ps, ok1 := p.Start(); ok1 {
		if pe, ok2 := p.End(); ok2 {
			for i := range p.Packages {
				wp := &p.Packages[i]
				if wp.Start.Before(ps) || wp.End.After(pe) {
					warn = append(warn, fmt.Sprintf("WP %s: %s..%s is outside the project window %s..%s",
						wp.ID, wp.Start.Format(DateLayout), wp.End.Format(DateLayout),
						ps.Format(DateLayout), pe.Format(DateLayout)))
				}
			}
		}
	}
	// Role capacity (R-cap-role): for each action with effort and roles, the
	// combined role capacity (sum of percent * hours/day over working days)
	// must cover the effort, and the effort must fit the window at all.
	for i := range p.Packages {
		wp := &p.Packages[i]
		if wp.Summary {
			continue
		}
		effort, _, _, _, fits, fitsWindow := p.RoleCapacity(wp)
		if effort <= 0 {
			continue
		}
		hasRole := false
		for ri := range p.Roles {
			for _, as := range p.Roles[ri].Assignments {
				if as.WorkPackageID == wp.ID {
					hasRole = true
				}
			}
		}
		if hasRole && !fits {
			warn = append(warn, fmt.Sprintf("WP %s: assigned role capacity does not cover the effort", wp.ID))
		}
		if !fitsWindow {
			warn = append(warn, fmt.Sprintf("WP %s: effort does not fit the start..end window", wp.ID))
		}
	}
	return warn
}

// depKindLabel gives FS a friendly default label.
func depKindLabel(k string) string {
	if k == "" {
		return "FS"
	}
	return k
}

// DepViolated reports whether a dependency's timing constraint is breached.
// A = predecessor (From), B = successor (To). Mid(x) = start + (end-start)/2.
//
//	FS (finish-start, EA):   Start(B) >= End(A)
//	SS (start-start, AA):    Start(B) >= Start(A)
//	FF (finish-finish, EE):  End(B)   >= End(A)
//	SF (start-finish, AE):   End(B)   >= Start(A)
//	MS (middle-start, MA):   Start(B) >= Mid(A)
//	ME (middle-end, ME):     End(B)   >= Mid(A)
//	MM (middle-middle, MM):  Start(A) <= Mid(B) <= End(A)
//
// Touching endpoints satisfy the ">=" / "<=" rules (not a violation). "OV" is
// accepted as a legacy alias of MM.
//
// CHANGE LOCK (v0.2.0): these seven relationship rules and their semantics are
// a LOCKED, stable contract as of v0.2.0. Do NOT change a rule's comparison,
// add/rename/remove a kind, or alter the touching-endpoint semantics without an
// explicit, deliberate decision (and a matching update to
// documentation/relationships.md, the RelationshipRules binding, the i18n
// abbreviations, and TestAllRelationshipRules). The canonical reference is
// documentation/relationships.md.
func (p *Project) DepViolated(dep Dependency) bool {
	kind := dep.Kind
	if kind == "" {
		kind = "FS"
	}
	from := p.taskSpan(dep.From) // A
	to := p.taskSpan(dep.To)     // B
	if from == nil || to == nil {
		return false
	}
	switch kind {
	case "FS":
		return to.start.Before(from.end) // Start(B) < End(A)
	case "SS":
		return to.start.Before(from.start) // Start(B) < Start(A)
	case "FF":
		return to.end.Before(from.end) // End(B) < End(A)
	case "SF":
		return to.end.Before(from.start) // End(B) < Start(A)
	case "MS":
		return to.start.Before(from.mid()) // Start(B) < Mid(A)
	case "ME":
		return to.end.Before(from.mid()) // End(B) < Mid(A)
	case "MM", "OV":
		// Start(A) <= Mid(B) <= End(A): the successor's midpoint must fall
		// within the predecessor's span. "OV" is the legacy alias.
		mb := to.mid()
		return mb.Before(from.start) || mb.After(from.end)
	default:
		return false
	}
}

type span struct{ start, end time.Time }

// mid returns the midpoint of a span (start + half the duration).
func (s *span) mid() time.Time {
	return s.start.Add(s.end.Sub(s.start) / 2)
}

// taskSpan resolves a dependency endpoint id (work package, detail, OR
// milestone) to its start/end span. A milestone is a zero-duration point.
func (p *Project) taskSpan(id string) *span {
	if wp := p.WorkPackage(id); wp != nil {
		return &span{start: wp.Start, end: wp.End}
	}
	if d, _ := p.DetailByID(id); d != nil {
		return &span{start: d.Start, end: d.End}
	}
	if m := p.MilestoneByID(id); m != nil {
		return &span{start: m.Date, end: m.Date}
	}
	return nil
}
