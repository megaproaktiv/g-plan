package main

import (
	"fmt"
	"strings"

	"github.com/megaproaktiv/g-plan/src/app/plan"
)

// DTOs are the JSON shapes crossing the Wails boundary to the frontend. Dates
// are ISO strings; estimations are pre-formatted display strings PLUS the raw
// integer base value so the UI can show and edit without re-implementing the
// unit arithmetic.

type ProjectDTO struct {
	Title           string            `json:"title"`
	Description     string            `json:"description"`
	Unit            string            `json:"unit"`
	HoursPerDay     int               `json:"hoursPerDay"`
	Language        string            `json:"language"`
	Start           string            `json:"start"`
	End             string            `json:"end"`
	TotalEst        string            `json:"totalEst"`
	Milestones      []MilestoneDTO    `json:"milestones"`
	Phases          []PhaseDTO        `json:"phases"`
	Packages        []WorkPackDTO     `json:"packages"`
	Deps            []DepDTO          `json:"deps"`
	Roles           []RoleDTO         `json:"roles"`
	SectionColors   map[string]string `json:"sectionColors"`
	SectionNums     map[string]int    `json:"sectionNums"`
	ArrowStyle      string            `json:"arrowStyle"`
	ShowDetails     bool              `json:"showDetails"`
	ShowRoleActions bool              `json:"showRoleActions"`
	ShowRolePercent bool              `json:"showRolePercent"`
	ShowRoleDays    bool              `json:"showRoleDays"`
	ShowPhases      bool              `json:"showPhases"`
	ShowProgress    bool              `json:"showProgress"`
	Monitor         MonitorDTO        `json:"monitor"`
	Zoom            string            `json:"zoom"`
	AutosaveEnabled bool              `json:"autosaveEnabled"`
	AutosaveMinutes int               `json:"autosaveMinutes"`
	AutosaveOnExit  bool              `json:"autosaveOnExit"`
	SaveVersions    bool              `json:"saveVersions"`
	VersionCount    int               `json:"versionCount"`
	ReportFormat    string            `json:"reportFormat"`
	ReportTodayLine bool              `json:"reportTodayLine"`
	ReportGrid      string            `json:"reportGrid"`
	ReportShowNotes bool              `json:"reportShowNotes"`
	ShowWeekNumbers bool              `json:"showWeekNumbers"`
	SavePath        string            `json:"savePath"`
	Warnings        []string          `json:"warnings"`
}

type MilestoneDTO struct {
	ID        string `json:"id"`
	DisplayID string `json:"displayId"` // "n.k" within its section (shared with actions)
	Name      string `json:"name"`
	Date      string `json:"date"`
	Cluster   string `json:"cluster"`
	Color     string `json:"color"`
}

type PhaseDTO struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Start string `json:"start"`
	End   string `json:"end"`
}

type DetailDTO struct {
	ID        string `json:"id"`
	DisplayID string `json:"displayId"` // "n.k.m"
	Name      string `json:"name"`
	Desc      string `json:"desc"`
	Role      string `json:"role"`
	Start     string `json:"start"`
	End       string `json:"end"`
	Est       string `json:"est"`
	Summary   bool   `json:"summary"`   // rollup/Sammelvorgang bar
	FontColor string `json:"fontColor"` // label color (hex); empty = auto-contrast
	TextPos   string `json:"textPos"`   // "" / auto | center | left | right | up | down
}

type WorkPackDTO struct {
	ID         string `json:"id"`
	Num        int    `json:"num"`       // global sequence (kept for compat)
	DisplayID  string `json:"displayId"` // "n.k" within its work package
	Name       string `json:"name"`
	Desc       string `json:"desc"`
	Start      string `json:"start"`
	End        string `json:"end"`
	Est        string `json:"est"` // effective estimation, display string
	RealEst    string `json:"realEst"`
	Contour    string `json:"contour"`
	ExternalID string `json:"externalId"` // id in the external time-tracking system
	// Workdays is the number of working days (Mon-Fri, no weekends) in the
	// inclusive start..end window.
	Workdays int `json:"workdays"`
	// Planned effort of the action in hours and in working days (the project
	// unit may be either; the action view shows both).
	EstHours string `json:"estHours"`
	EstDays  string `json:"estDays"`
	// Real effort in hours and days, and whether it is synchronized from the
	// external system (then read-only).
	RealHours string `json:"realHours"`
	RealDays  string `json:"realDays"`
	Synced    bool   `json:"synced"`
	SyncedAt  string `json:"syncedAt"`
	// The two consumed ratings in tenths of a percent (164 = 16,4 %): pure
	// (real/planned) and weighted by the contour (area under the curve).
	ProgressPermille int `json:"progressPermille"`
	ContourPermille  int `json:"contourPermille"`
	// ProgressPercent is the integer 0..100 progress; CompletionPos is the
	// 0..1 position along the duration after contour reshaping.
	ProgressPercent int     `json:"progressPercent"`
	CompletionPos   float64 `json:"completionPos"`
	Mode            string  `json:"mode"` // fixed | derived
	Summary         bool    `json:"summary"`
	DetailsHidden   bool    `json:"detailsHidden"` // per-action: hide my detail bars even when Show Details is on
	Color           string  `json:"color"`
	FontColor       string  `json:"fontColor"`   // label color (hex); empty = auto-contrast
	TextPos         string  `json:"textPos"`     // "" / auto | center | left | right | up | down
	NoteVisible     bool    `json:"noteVisible"` // draw Description as a yellow sticky note
	NotePos         string  `json:"notePos"`     // note placement (same vocab as textPos)
	Cluster         string  `json:"cluster"`
	SameLine        bool    `json:"sameLine"`
	HasWarning      bool    `json:"hasWarning"` // true if any validation warning involves this action
	// RoleFit describes whether the assigned roles' capacity covers the effort
	// and whether the effort fits the start..end window (capacity model).
	RoleFit *RoleFitDTO `json:"roleFit,omitempty"`
	Details []DetailDTO `json:"details"`
}

// RoleFitDTO is the read-only capacity summary shown below an action's roles.
type RoleFitDTO struct {
	Effort     string `json:"effort"`     // display string (e.g. "10")
	Capacity   string `json:"capacity"`   // assigned role capacity in the window
	PerDay     string `json:"perDay"`     // combined per-working-day capacity
	Fits       bool   `json:"fits"`       // capacity >= effort
	FitsWindow bool   `json:"fitsWindow"` // effort fits the start..end window at all
	Message    string `json:"message"`    // human-readable summary line
}

type DepDTO struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Kind       string `json:"kind"`
	FromAnchor string `json:"fromAnchor"`
	ToAnchor   string `json:"toAnchor"`
	Color      string `json:"color"`
	Violated   bool   `json:"violated"` // FS timing constraint breached
}

type AssignmentDTO struct {
	WorkPackageID string `json:"wp"`
	Percent       int    `json:"percent"`
	Consumed      string `json:"consumed"`
}

type RoleDTO struct {
	Name        string          `json:"name"`
	ID          int             `json:"id"` // stable role number (R<id>)
	Description string          `json:"description"`
	Color       string          `json:"color"`
	InUse       bool            `json:"inUse"`  // referenced by an assignment or a detail: cannot be deleted
	UsedBy      string          `json:"usedBy"` // human-readable description of the references
	TotalEst    string          `json:"totalEst"`
	Assignments []AssignmentDTO `json:"assignments"`
}

func toDTO(p *plan.Project) *ProjectDTO {
	if p == nil {
		return &ProjectDTO{Unit: string(plan.UnitDays)}
	}
	d := &ProjectDTO{
		Title:       p.Title,
		Description: p.Description,
		Unit:        string(p.Unit),
		HoursPerDay: p.HoursPerDay,
		Language:    plan.NormalizeLanguage(p.Language),
		TotalEst:    p.Format(p.TotalEst()),
		// initialize to empty (not nil) so JSON marshals [] not null — the
		// frontend iterates these unconditionally.
		Milestones:      []MilestoneDTO{},
		Phases:          []PhaseDTO{},
		Packages:        []WorkPackDTO{},
		Deps:            []DepDTO{},
		Roles:           []RoleDTO{},
		Warnings:        []string{},
		SectionColors:   map[string]string{},
		SectionNums:     map[string]int{},
		ArrowStyle:      p.ArrowStyle,
		ShowDetails:     p.ShowDetails,
		ShowRoleActions: p.ShowRoleActions,
		ShowRolePercent: p.ShowRolePercent,
		ShowRoleDays:    p.ShowRoleDays,
		ShowPhases:      p.ShowPhases,
		ShowProgress:    p.ShowProgress,
		Monitor:         monitorToDTO(p),
		Zoom:            p.Zoom,
		AutosaveEnabled: p.AutosaveEnabled,
		AutosaveMinutes: plan.ClampAutosaveMinutes(p.AutosaveMinutes),
		AutosaveOnExit:  p.AutosaveOnExit,
		SaveVersions:    p.SaveVersions,
		VersionCount:    plan.ClampVersionCount(p.VersionCount),
		ReportFormat:    plan.NormalizeReportFormat(p.ReportFormat),
		ReportTodayLine: p.ReportTodayLine,
		ReportGrid:      plan.NormalizeReportGrid(p.ReportGrid),
		ReportShowNotes: p.ReportShowNotes,
		ShowWeekNumbers: p.ShowWeekNumbers,
	}
	for k, v := range p.SectionColors {
		d.SectionColors[k] = v
	}
	// The numbers shown on screen (WP-n, n.k, Pk) come from the one shared
	// numbering, plan.WBS, which the command resolvers use as well.
	wbs := p.WBS()
	for name, n := range wbs.SectionNum {
		d.SectionNums[name] = n
	}
	if s, ok := p.Start(); ok {
		d.Start = s.Format(plan.DateLayout)
	}
	if e, ok := p.End(); ok {
		d.End = e.Format(plan.DateLayout)
	}
	if w := p.Validate(); len(w) > 0 {
		d.Warnings = w
	}
	for _, m := range p.Milestones {
		mc := MilestoneDTO{ID: m.ID, Name: m.Name, Date: m.Date.Format(plan.DateLayout), Cluster: m.Cluster, Color: m.Color}
		mc.DisplayID = wbs.MilestoneDisplay[m.ID]
		d.Milestones = append(d.Milestones, mc)
	}
	for _, ph := range p.Phases {
		d.Phases = append(d.Phases, PhaseDTO{ID: ph.ID, Name: ph.Name,
			Start: ph.Start.Format(plan.DateLayout), End: ph.End.Format(plan.DateLayout)})
	}
	for i := range p.Packages {
		wp := &p.Packages[i]
		displayID := wbs.ActionDisplay[wp.ID]
		// Progress is computed against the effective estimation (the sum of
		// details in derived mode) so it matches the displayed estimate.
		eff := *wp
		eff.Est = wp.EffectiveEst()
		w := WorkPackDTO{
			ID: wp.ID, Num: i + 1,
			DisplayID: displayID,
			Name:      wp.Name, Desc: wp.Desc,
			Start: wp.Start.Format(plan.DateLayout), End: wp.End.Format(plan.DateLayout),
			Est: p.Format(wp.EffectiveEst()), Mode: string(wp.Mode), Summary: wp.Summary, DetailsHidden: wp.DetailsHidden,
			RealEst:          p.Format(wp.EffectiveRealEst()),
			Contour:          string(plan.NormalizeContour(string(wp.Contour))),
			ExternalID:       wp.ExternalID,
			Workdays:         plan.WorkdaysBetween(wp.Start, wp.End),
			EstHours:         p.FormatHours(wp.EffectiveEst()),
			EstDays:          p.FormatDays(wp.EffectiveEst()),
			RealHours:        p.FormatHours(wp.EffectiveRealEst()),
			RealDays:         p.FormatDays(wp.EffectiveRealEst()),
			Synced:           p.IsSynced(wp),
			SyncedAt:         p.Monitor.SyncedAt,
			ProgressPermille: eff.ProgressPermille(),
			ContourPermille:  eff.ContourPermille(),
			ProgressPercent:  eff.ProgressPercent(),
			CompletionPos:    eff.CompletionPosition(),
			Color:            wp.Color, FontColor: wp.FontColor, TextPos: wp.TextPos,
			NoteVisible: wp.NoteVisible, NotePos: wp.NotePos,
			Cluster: wp.Cluster, SameLine: wp.SameLine,
			HasWarning: actionHasWarning(p, wp),
			Details:    []DetailDTO{},
		}
		// Role-capacity fit summary (shown read-only under the roles). Only
		// meaningful when the action has effort and at least one role.
		if !wp.Summary {
			effort, perDay, capacity, _, fits, fitsWindow := p.RoleCapacity(wp)
			hasRole := false
			for ri := range p.Roles {
				for _, as := range p.Roles[ri].Assignments {
					if as.WorkPackageID == wp.ID {
						hasRole = true
					}
				}
			}
			if effort > 0 && hasRole {
				msg := fmt.Sprintf("Assigned capacity %s of %s %s", p.Format(capacity), p.Format(effort), p.Unit)
				if fits {
					msg += " — effort is covered."
				} else {
					msg += " — NOT enough: increase role %, add a role, or widen the dates."
				}
				if !fitsWindow {
					msg += " The window is too short to hold the effort even at 100%."
				}
				w.RoleFit = &RoleFitDTO{
					Effort: p.Format(effort), Capacity: p.Format(capacity), PerDay: p.Format(perDay),
					Fits: fits, FitsWindow: fitsWindow, Message: msg,
				}
			}
		}
		for j := range wp.Details {
			dp := &wp.Details[j]
			w.Details = append(w.Details, DetailDTO{
				ID: dp.ID, DisplayID: fmt.Sprintf("%s.%d", w.DisplayID, j+1),
				Name: dp.Name, Desc: dp.Desc, Role: dp.Role,
				Start: dp.Start.Format(plan.DateLayout), End: dp.End.Format(plan.DateLayout),
				Est: p.Format(dp.Est), Summary: dp.Summary, FontColor: dp.FontColor, TextPos: dp.TextPos,
			})
		}
		d.Packages = append(d.Packages, w)
	}
	for _, dep := range p.Deps {
		d.Deps = append(d.Deps, DepDTO{
			From: dep.From, To: dep.To, Kind: dep.Kind,
			FromAnchor: dep.FromAnchor, ToAnchor: dep.ToAnchor, Color: dep.Color,
			Violated: p.DepViolated(dep),
		})
	}
	for i := range p.Roles {
		r := &p.Roles[i]
		use := p.RoleUsage(r.Name)
		rd := RoleDTO{Name: r.Name, ID: r.ID, InUse: use.InUse(), UsedBy: use.String(), Description: r.Description, Color: r.Color, TotalEst: p.Format(p.RoleEst(r)), Assignments: []AssignmentDTO{}}
		for _, a := range r.Assignments {
			rd.Assignments = append(rd.Assignments, AssignmentDTO{
				WorkPackageID: a.WorkPackageID, Percent: a.Percent, Consumed: p.AssignmentConsumed(a),
			})
		}
		d.Roles = append(d.Roles, rd)
	}
	return d
}

// actionHasWarning reports whether any validation concern involves this work
// package: over-allocation (effort > workdays), a violated FS dependency, or
// role percentages that do not total 100% while the action carries effort.
func actionHasWarning(p *plan.Project, wp *plan.WorkPackage) bool {
	if _, _, over := p.Overallocated(wp); over {
		return true
	}
	for _, dep := range p.Deps {
		if (dep.From == wp.ID || dep.To == wp.ID) && p.DepViolated(dep) {
			return true
		}
	}
	// Role percentages are checked PER LINE only (R-6.8): a single assignment
	// may not exceed 100%. Percentages are NOT summed across role lines, so
	// multiple roles (or multiple instances of a role) each at 100% on one
	// action is valid and raises no warning.
	for i := range p.Roles {
		for _, as := range p.Roles[i].Assignments {
			if as.WorkPackageID == wp.ID && as.Percent > 100 {
				return true
			}
		}
	}
	// Role capacity: assigned roles must cover the effort and the effort must
	// fit the window (the capacity model; R-cap-role).
	if !wp.Summary {
		effort, _, _, _, fits, fitsWindow := p.RoleCapacity(wp)
		if effort > 0 {
			hasRole := false
			for i := range p.Roles {
				for _, as := range p.Roles[i].Assignments {
					if as.WorkPackageID == wp.ID {
						hasRole = true
					}
				}
			}
			if (hasRole && !fits) || !fitsWindow {
				return true
			}
		}
	}
	// Outside the project Start..End window.
	if ps, ok1 := p.Start(); ok1 {
		if pe, ok2 := p.End(); ok2 {
			if wp.Start.Before(ps) || wp.End.After(pe) {
				return true
			}
		}
	}
	return false
}

// idFrom makes a lowercase id-safe token from a name (for new packages/details).
func idFrom(prefix, name string) string {
	var b strings.Builder
	b.WriteString(prefix)
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			b.WriteByte('_')
		}
	}
	return b.String()
}

// monitorToDTO converts the monitor setting, including the period a sync would
// fetch (project period plus the chosen extra months).
func monitorToDTO(p *plan.Project) MonitorDTO {
	m := p.Monitor
	d := MonitorDTO{Enabled: m.Enabled, System: m.System, ProjectID: m.ProjectID, DefaultAction: m.DefaultAction,
		IncludeBefore: m.IncludeBefore, IncludeAfter: m.IncludeAfter}
	if s, ok1 := p.Start(); ok1 {
		if e, ok2 := p.End(); ok2 {
			from, to := m.SyncRange(s, e)
			d.SyncFrom, d.SyncTo = from.Format(plan.DateLayout), to.Format(plan.DateLayout)
		}
	}
	return d
}
