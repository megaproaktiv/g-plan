package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/megaproaktiv/g-plan/src/app/plan"
)

func parseDate(s string) (time.Time, error) {
	return time.Parse(plan.DateLayout, s)
}

// MoveWorkPackage sets a work package's start/end (Epic 1: drag in time and
// resize). Both dates are ISO strings. Duration is independent of estimation
// (R-4.4), so moving/resizing does not change the estimate here.
func (a *App) MoveWorkPackage(id, start, end string) (*ProjectDTO, error) {
	wp := a.project.WorkPackage(id)
	if wp == nil {
		return nil, fmt.Errorf("work package %q not found", id)
	}
	s, err := parseDate(start)
	if err != nil {
		return nil, fmt.Errorf("start: %w", err)
	}
	e, err := parseDate(end)
	if err != nil {
		return nil, fmt.Errorf("end: %w", err)
	}
	if e.Before(s) {
		return nil, fmt.Errorf("end before start")
	}
	if !wp.Start.Equal(s) || !wp.End.Equal(e) {
		a.pushUndo() // a move/resize is undoable (Cmd-Z)
	}
	wp.Start, wp.End = s, e
	return toDTO(a.project), nil
}

// SetWorkPackageEstimation sets a level-1 estimation from a display string
// (whole units only; R-4.1). Rejected if the project is fixed-mode and the
// value is not a whole display unit.
func (a *App) SetWorkPackageEstimation(id, display string) (*ProjectDTO, error) {
	wp := a.project.WorkPackage(id)
	if wp == nil {
		return nil, fmt.Errorf("work package %q not found", id)
	}
	if !wp.TracksEffort() {
		return nil, fmt.Errorf("action %q has the contour None: it has no estimation", wp.Name)
	}
	v, err := a.project.Parse(display)
	if err != nil {
		return nil, err
	}
	if v < 0 || v > plan.MaxEffortMinutes {
		return nil, fmt.Errorf("estimation out of range (0..%d minutes)", plan.MaxEffortMinutes)
	}
	// Typing an estimation on an action makes the level-1 value authoritative:
	// switch it to fixed mode so the value is held, flows into the project
	// total and is distributed over the details. (An action in derived mode
	// ignores wp.Est - the value would be stored but have no visible effect.)
	if !a.project.IsWholeDisplayUnit(v) {
		return nil, fmt.Errorf("level-1 estimation must be a whole %s", a.project.UnitSingular())
	}
	wp.Mode = plan.ModeFixed
	wp.Est = v
	return toDTO(a.project), nil
}

// SetWorkPackageRealEffort sets the Real_Effort from a display string. The
// value must be non-empty, parseable, and within 0..MaxEffortMinutes (R1.4,
// R1.5). On rejection an error is returned and the stored value is unchanged.
func (a *App) SetWorkPackageRealEffort(id, display string) (*ProjectDTO, error) {
	wp := a.project.WorkPackage(id)
	if wp == nil {
		return nil, fmt.Errorf("work package %q not found", id)
	}
	if !wp.TracksEffort() {
		return nil, fmt.Errorf("action %q has the contour None: it has no real effort", wp.Name)
	}
	if a.project.IsSynced(wp) {
		return nil, fmt.Errorf("the real effort of %q is synchronized from the external system and cannot be edited", wp.Name)
	}
	if strings.TrimSpace(display) == "" {
		return nil, fmt.Errorf("real effort must not be empty")
	}
	v, err := a.project.Parse(display)
	if err != nil {
		return nil, err
	}
	if v < 0 || v > plan.MaxEffortMinutes {
		return nil, fmt.Errorf("real effort out of range (0..%d minutes)", plan.MaxEffortMinutes)
	}
	wp.RealEst = v
	return toDTO(a.project), nil
}

// SetWorkPackageContour sets the Work_Contour. Only an exact match of one of
// the four contour kinds is accepted (R4.2, R4.3); otherwise an error is
// returned and the stored contour is unchanged.
func (a *App) SetWorkPackageContour(id, kind string) (*ProjectDTO, error) {
	wp := a.project.WorkPackage(id)
	if wp == nil {
		return nil, fmt.Errorf("work package %q not found", id)
	}
	if !plan.IsValidContour(kind) {
		return nil, fmt.Errorf("invalid contour %q", kind)
	}
	if plan.Contour(kind) == plan.ContourNone && wp.ExternalID != "" {
		return nil, fmt.Errorf("action %q has the External ID %q; an action with the contour None cannot carry one - remove the External ID first", wp.Name, wp.ExternalID)
	}
	if plan.Contour(kind) == plan.ContourNone && a.project.Monitor.DefaultAction == id {
		return nil, fmt.Errorf("action %q is the default action of the monitor and receives bookings; choose another default action first", wp.Name)
	}
	wp.Contour = plan.Contour(kind)
	return toDTO(a.project), nil
}

// SetWorkPackageMode switches a work package between fixed and derived
// estimation (R-5.4).
func (a *App) SetWorkPackageMode(id, mode string) (*ProjectDTO, error) {
	wp := a.project.WorkPackage(id)
	if wp == nil {
		return nil, fmt.Errorf("work package %q not found", id)
	}
	switch plan.EstimationMode(mode) {
	case plan.ModeFixed, plan.ModeDerived:
		wp.Mode = plan.EstimationMode(mode)
	default:
		return nil, fmt.Errorf("invalid mode %q", mode)
	}
	return toDTO(a.project), nil
}

// AddWorkPackage appends a new work package with a derived mode and the given
// span. Returns the updated project.
func (a *App) AddWorkPackage(name, start, end string) (*ProjectDTO, error) {
	s, err := parseDate(start)
	if err != nil {
		return nil, err
	}
	e, err := parseDate(end)
	if err != nil {
		return nil, err
	}
	if e.Before(s) {
		return nil, fmt.Errorf("the end date %s is before the start date %s", end, start)
	}
	a.pushUndo() // adding an action is undoable (Cmd-Z)
	base := idFrom("", name)
	// An id made only of digits would be read as an action NUMBER ("2" = the
	// second action) in commands, so it gets a letter prefix.
	if base != "" && strings.Trim(base, "0123456789") == "" {
		base = "a" + base
	}
	id := a.uniqueID(base)
	if id == "" {
		id = a.uniqueID("wp")
	}
	a.project.Packages = append(a.project.Packages, plan.WorkPackage{
		ID: id, Name: name, Start: s, End: e, Mode: plan.ModeDerived, Cluster: a.currentSection,
		Contour: plan.ContourFlat,
	})
	return toDTO(a.project), nil
}

// MoveAction reorders an action within its OWN section/cluster. dir < 0 moves
// it up (toward n.1), dir > 0 moves it down. It swaps the action with its
// neighbour that shares the same cluster, preserving all other ordering. The
// n.k display ids are derived from this order, so this changes the visible
// position. Undoable (Cmd-Z).
func (a *App) MoveAction(id string, dir int) (*ProjectDTO, error) {
	idx := -1
	for i := range a.project.Packages {
		if a.project.Packages[i].ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("action %q not found", id)
	}
	cluster := a.project.Packages[idx].Cluster
	// find the neighbour index (previous/next) that shares the same cluster
	step := -1
	if dir > 0 {
		step = 1
	}
	swap := -1
	for j := idx + step; j >= 0 && j < len(a.project.Packages); j += step {
		if a.project.Packages[j].Cluster == cluster {
			swap = j
			break
		}
	}
	if swap < 0 {
		return toDTO(a.project), nil // already at an edge within its section
	}
	a.pushUndo()
	a.project.Packages[idx], a.project.Packages[swap] = a.project.Packages[swap], a.project.Packages[idx]
	return toDTO(a.project), nil
}

// RemoveWorkPackage deletes a work package by id. The deletion is undoable.
func (a *App) RemoveWorkPackage(id string) *ProjectDTO {
	a.pushUndo()
	out := a.project.Packages[:0]
	for _, wp := range a.project.Packages {
		if wp.ID != id {
			out = append(out, wp)
		}
	}
	a.project.Packages = out
	return toDTO(a.project)
}

// --- Drill-in: detail package edits (Epic 2) ---

// AddDetail adds a level-2 detail package to a work package, with optional
// description and role (role must be an existing project role or empty).
func (a *App) AddDetail(wpID, name, start, end, est, desc, role string) (*ProjectDTO, error) {
	wp := a.project.WorkPackage(wpID)
	if wp == nil {
		return nil, fmt.Errorf("work package %q not found", wpID)
	}
	s, err := parseDate(start)
	if err != nil {
		return nil, err
	}
	e, err := parseDate(end)
	if err != nil {
		return nil, err
	}
	v, err := a.project.Parse(est)
	if err != nil {
		return nil, err
	}
	if role != "" && !a.roleExists(role) {
		return nil, fmt.Errorf("role %q does not exist", role)
	}
	id := a.uniqueID(idFrom(wpID+"_", name))
	wp.Details = append(wp.Details, plan.DetailPackage{
		ID: id, Name: name, Desc: desc, Role: role, Start: s, End: e, Est: v,
	})
	return toDTO(a.project), nil
}

// SetDetailFields updates a detail's title, description, estimation and role
// in one call (the drill-in editor). Empty est leaves it unchanged.
func (a *App) SetDetailFields(wpID, detID, name, desc, est, role string) (*ProjectDTO, error) {
	wp := a.project.WorkPackage(wpID)
	if wp == nil {
		return nil, fmt.Errorf("work package %q not found", wpID)
	}
	if role != "" && !a.roleExists(role) {
		return nil, fmt.Errorf("role %q does not exist", role)
	}
	for i := range wp.Details {
		if wp.Details[i].ID != detID {
			continue
		}
		if name != "" {
			wp.Details[i].Name = name
		}
		wp.Details[i].Desc = desc
		wp.Details[i].Role = role
		if est != "" {
			v, err := a.project.Parse(est)
			if err != nil {
				return nil, err
			}
			wp.Details[i].Est = v
		}
		return toDTO(a.project), nil
	}
	return nil, fmt.Errorf("detail %q not found", detID)
}

func (a *App) roleExists(name string) bool {
	for i := range a.project.Roles {
		if a.project.Roles[i].Name == name {
			return true
		}
	}
	return false
}

// SetActionFields edits an action's title, description, estimation and dates
// in one call (the action editor). Empty est/start/end leave those unchanged.
func (a *App) SetActionFields(id, name, desc, est, start, end string) (*ProjectDTO, error) {
	wp := a.project.WorkPackage(id)
	if wp == nil {
		return nil, fmt.Errorf("action %q not found", id)
	}
	if name != "" {
		wp.Name = name
	}
	wp.Desc = desc
	if est != "" && wp.TracksEffort() { // contour None: the (greyed-out) estimation is left alone
		v, err := a.project.Parse(est)
		if err != nil {
			return nil, err
		}
		// Setting an estimation directly on an action makes the level-1 value
		// authoritative. If the action has no details, switch it to fixed mode
		// so the typed value is held and flows into the project total (R-5.5/6).
		if len(wp.Details) == 0 {
			wp.Mode = plan.ModeFixed
		}
		if wp.Mode == plan.ModeFixed && !a.project.IsWholeDisplayUnit(v) {
			return nil, fmt.Errorf("level-1 estimation must be a whole %s", a.project.UnitSingular())
		}
		wp.Est = v
	}
	if start != "" && end != "" {
		s, err := parseDate(start)
		if err != nil {
			return nil, fmt.Errorf("start: %w", err)
		}
		e, err := parseDate(end)
		if err != nil {
			return nil, fmt.Errorf("end: %w", err)
		}
		if e.Before(s) {
			return nil, fmt.Errorf("cannot save: end date must be on or after start date")
		}
		wp.Start, wp.End = s, e
	}
	return toDTO(a.project), nil
}

// SetActionCluster moves an action into a different section/swimlane (the
// vertical drag gesture). An empty cluster means the default/ungrouped lane.
func (a *App) SetActionCluster(id, cluster string) (*ProjectDTO, error) {
	wp := a.project.WorkPackage(id)
	if wp == nil {
		return nil, fmt.Errorf("action %q not found", id)
	}
	wp.Cluster = cluster
	return toDTO(a.project), nil
}

// SetActionColor sets (or clears, when color=="") an action's bar color.
func (a *App) SetActionColor(id, color string) (*ProjectDTO, error) {
	wp := a.project.WorkPackage(id)
	if wp == nil {
		return nil, fmt.Errorf("action %q not found", id)
	}
	wp.Color = color
	return toDTO(a.project), nil
}

// SetActionFontColor sets (or clears, when color=="") an action's label/font
// color. Empty means auto-contrast (the renderer picks black or white against
// the bar background so the label stays readable). Use "#000000"/"#ffffff" to
// force black or white.
func (a *App) SetActionFontColor(id, color string) (*ProjectDTO, error) {
	wp := a.project.WorkPackage(id)
	if wp == nil {
		return nil, fmt.Errorf("action %q not found", id)
	}
	wp.FontColor = color
	return toDTO(a.project), nil
}

// SetDetailFontColor sets (or clears) a detail sub-action's label/font color.
// Empty means auto-contrast, matching SetActionFontColor for level-1 actions.
func (a *App) SetDetailFontColor(wpID, detID, color string) (*ProjectDTO, error) {
	wp := a.project.WorkPackage(wpID)
	if wp == nil {
		return nil, fmt.Errorf("work package %q not found", wpID)
	}
	for i := range wp.Details {
		if wp.Details[i].ID == detID {
			wp.Details[i].FontColor = color
			return toDTO(a.project), nil
		}
	}
	return nil, fmt.Errorf("detail %q not found", detID)
}

// validTextPos reports whether pos is an accepted text-position value. Empty
// and "auto" both mean auto-placement (inside if the label fits, else right).
// Positions: center | left (inside flush-left) | right / outright (outside, to
// the right of the bar) | outleft (outside, ending at the bar's left edge) |
// up | down. "right" is a legacy alias of "outright".
func validTextPos(pos string) bool {
	switch pos {
	case "", "auto", "center", "left", "right", "outright", "outleft", "up", "down":
		return true
	}
	return false
}

// validNotePos accepts the text-position values PLUS four diagonal corners for
// the sticky note: ul (upper-left), ur (upper-right), ll (lower-left),
// lr (lower-right), so the connector to the note can run at ~45 degrees.
func validNotePos(pos string) bool {
	switch pos {
	case "ul", "ur", "ll", "lr":
		return true
	}
	return validTextPos(pos)
}

// SetActionTextPos sets where an action's label is drawn. "" or "auto" =
// auto-placement (inside when it fits, otherwise to the right); an explicit
// value is one of center | left | right | up | down.
func (a *App) SetActionTextPos(id, pos string) (*ProjectDTO, error) {
	wp := a.project.WorkPackage(id)
	if wp == nil {
		return nil, fmt.Errorf("action %q not found", id)
	}
	if !validTextPos(pos) {
		return nil, fmt.Errorf("invalid text position %q", pos)
	}
	if pos == "auto" {
		pos = ""
	}
	wp.TextPos = pos
	return toDTO(a.project), nil
}

// SetActionNote toggles whether the action's Description is drawn as a yellow
// sticky note near the bar, and where (notePos uses the same vocabulary as the
// text position; "auto"/"" is normalized to the default "up"). The note text is
// the action's Description.
func (a *App) SetActionNote(id string, visible bool, pos string) (*ProjectDTO, error) {
	wp := a.project.WorkPackage(id)
	if wp == nil {
		return nil, fmt.Errorf("action %q not found", id)
	}
	if !validNotePos(pos) {
		return nil, fmt.Errorf("invalid note position %q", pos)
	}
	if pos == "auto" || pos == "" {
		pos = "up"
	}
	wp.NoteVisible = visible
	wp.NotePos = pos
	return toDTO(a.project), nil
}

// SetDetailTextPos sets a detail sub-action's label position, same values and
// meaning as SetActionTextPos.
func (a *App) SetDetailTextPos(wpID, detID, pos string) (*ProjectDTO, error) {
	wp := a.project.WorkPackage(wpID)
	if wp == nil {
		return nil, fmt.Errorf("work package %q not found", wpID)
	}
	if !validTextPos(pos) {
		return nil, fmt.Errorf("invalid text position %q", pos)
	}
	if pos == "auto" {
		pos = ""
	}
	for i := range wp.Details {
		if wp.Details[i].ID == detID {
			wp.Details[i].TextPos = pos
			return toDTO(a.project), nil
		}
	}
	return nil, fmt.Errorf("detail %q not found", detID)
}

// SetDependencyColor sets (or clears) a link's color.
func (a *App) SetDependencyColor(from, to, color string) *ProjectDTO {
	for i := range a.project.Deps {
		if a.project.Deps[i].From == from && a.project.Deps[i].To == to {
			a.project.Deps[i].Color = color
		}
	}
	return toDTO(a.project)
}

// SetSectionColor sets (or clears) a section/cluster color by name.
func (a *App) SetSectionColor(section, color string) *ProjectDTO {
	if a.project.SectionColors == nil {
		a.project.SectionColors = map[string]string{}
	}
	if color == "" {
		delete(a.project.SectionColors, section)
	} else {
		a.project.SectionColors[section] = color
	}
	return toDTO(a.project)
}

// SetWorkPackageSummary toggles whether a work package renders as a summary
// (Summiervorgang) rollup bar.
func (a *App) SetWorkPackageSummary(id string, summary bool) (*ProjectDTO, error) {
	wp := a.project.WorkPackage(id)
	if wp == nil {
		return nil, fmt.Errorf("work package %q not found", id)
	}
	wp.Summary = summary
	return toDTO(a.project), nil
}

// SetActionDetailsHidden toggles whether THIS action's detail bars are hidden in
// the main view while the project-wide Show Details is on (a per-action show/hide
// detail toggle). It has no effect when Show Details is off (no details drawn at
// all then).
func (a *App) SetActionDetailsHidden(id string, hidden bool) (*ProjectDTO, error) {
	wp := a.project.WorkPackage(id)
	if wp == nil {
		return nil, fmt.Errorf("action %q not found", id)
	}
	wp.DetailsHidden = hidden
	return toDTO(a.project), nil
}

// SetDetailSummary toggles whether a detail (sub-action) renders as a summary
// (Sammelvorgang) rollup bar on the detail plane, mirroring an action's summary.
func (a *App) SetDetailSummary(wpID, detID string, summary bool) (*ProjectDTO, error) {
	wp := a.project.WorkPackage(wpID)
	if wp == nil {
		return nil, fmt.Errorf("work package %q not found", wpID)
	}
	for i := range wp.Details {
		if wp.Details[i].ID == detID {
			wp.Details[i].Summary = summary
			return toDTO(a.project), nil
		}
	}
	return nil, fmt.Errorf("detail %q not found", detID)
}

// SetWorkPackageSameLine toggles whether an action may share a swimlane row
// with another action in the same cluster (the "same line" option).
func (a *App) SetWorkPackageSameLine(id string, sameLine bool) (*ProjectDTO, error) {
	wp := a.project.WorkPackage(id)
	if wp == nil {
		return nil, fmt.Errorf("work package %q not found", id)
	}
	wp.SameLine = sameLine
	return toDTO(a.project), nil
}

// AddDependency creates a typed link between two tasks (the L-key + click or
// anchor-drag gesture). Endpoints may be actions OR details (detail links are
// drawn in the drill plane). kind is FS|SS|FF|SF (default FS). fromAnchor/
// toAnchor (l|r|t|b) record where the user drew from so the arrow attaches
// there.
func (a *App) AddDependency(from, to, kind, fromAnchor, toAnchor string) (*ProjectDTO, error) {
	if from == "" || to == "" || from == to {
		return nil, fmt.Errorf("need two distinct tasks to link")
	}
	if !a.project.TaskByID(from) || !a.project.TaskByID(to) {
		return nil, fmt.Errorf("both tasks must exist")
	}
	// Phases-row milestones (and the project Start/End markers) are full-height
	// markers and cannot be link endpoints.
	if a.isUnlinkableMilestone(from) || a.isUnlinkableMilestone(to) {
		return nil, fmt.Errorf("a phases-row milestone cannot be linked")
	}
	switch kind {
	case "FS", "SS", "FF", "SF", "MS", "ME", "MM", "OV":
	default:
		kind = "FS"
	}
	for i := range a.project.Deps {
		if a.project.Deps[i].From == from && a.project.Deps[i].To == to {
			a.project.Deps[i].Kind = kind
			a.project.Deps[i].FromAnchor = fromAnchor
			a.project.Deps[i].ToAnchor = toAnchor
			return toDTO(a.project), nil
		}
	}
	a.project.Deps = append(a.project.Deps, plan.Dependency{
		From: from, To: to, Kind: kind, FromAnchor: fromAnchor, ToAnchor: toAnchor,
	})
	return toDTO(a.project), nil
}

// RemoveDependency deletes a link between two actions. The deletion is undoable.
func (a *App) RemoveDependency(from, to string) *ProjectDTO {
	a.pushUndo()
	out := a.project.Deps[:0]
	for _, d := range a.project.Deps {
		if !(d.From == from && d.To == to) {
			out = append(out, d)
		}
	}
	a.project.Deps = out
	return toDTO(a.project)
}

// MoveDetail sets a detail package's span (drag/resize inside a work package).
func (a *App) MoveDetail(wpID, detID, start, end string) (*ProjectDTO, error) {
	wp := a.project.WorkPackage(wpID)
	if wp == nil {
		return nil, fmt.Errorf("work package %q not found", wpID)
	}
	s, err := parseDate(start)
	if err != nil {
		return nil, err
	}
	e, err := parseDate(end)
	if err != nil {
		return nil, err
	}
	for i := range wp.Details {
		if wp.Details[i].ID == detID {
			// clamp the detail span to the parent action's window
			if s.Before(wp.Start) {
				s = wp.Start
			}
			if e.After(wp.End) {
				e = wp.End
			}
			if e.Before(s) {
				e = s
			}
			if !wp.Details[i].Start.Equal(s) || !wp.Details[i].End.Equal(e) {
				a.pushUndo()
			}
			wp.Details[i].Start, wp.Details[i].End = s, e
			return toDTO(a.project), nil
		}
	}
	return nil, fmt.Errorf("detail %q not found", detID)
}

// SetDetailEstimation sets a detail estimate (two-decimal fractions allowed).
func (a *App) SetDetailEstimation(wpID, detID, display string) (*ProjectDTO, error) {
	wp := a.project.WorkPackage(wpID)
	if wp == nil {
		return nil, fmt.Errorf("work package %q not found", wpID)
	}
	v, err := a.project.Parse(display)
	if err != nil {
		return nil, err
	}
	for i := range wp.Details {
		if wp.Details[i].ID == detID {
			wp.Details[i].Est = v
			return toDTO(a.project), nil
		}
	}
	return nil, fmt.Errorf("detail %q not found", detID)
}

// RemoveDetail deletes a detail package.
func (a *App) RemoveDetail(wpID, detID string) (*ProjectDTO, error) {
	wp := a.project.WorkPackage(wpID)
	if wp == nil {
		return nil, fmt.Errorf("work package %q not found", wpID)
	}
	a.pushUndo()
	out := wp.Details[:0]
	for _, d := range wp.Details {
		if d.ID != detID {
			out = append(out, d)
		}
	}
	wp.Details = out
	return toDTO(a.project), nil
}

// MoveDetailOrder reorders a detail within its parent work package by swapping
// it with its neighbour. dir < 0 moves it up (earlier in the list / lower n.k.m
// number), dir > 0 moves it down. Edges are no-ops. Mirrors MoveAction for
// level-1 actions so a detail sub-action has the same reorder affordance.
func (a *App) MoveDetailOrder(wpID, detID string, dir int) (*ProjectDTO, error) {
	wp := a.project.WorkPackage(wpID)
	if wp == nil {
		return nil, fmt.Errorf("work package %q not found", wpID)
	}
	idx := -1
	for i := range wp.Details {
		if wp.Details[i].ID == detID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("detail %q not found", detID)
	}
	swap := idx - 1
	if dir > 0 {
		swap = idx + 1
	}
	if swap < 0 || swap >= len(wp.Details) {
		return toDTO(a.project), nil // already at an edge
	}
	a.pushUndo()
	wp.Details[idx], wp.Details[swap] = wp.Details[swap], wp.Details[idx]
	return toDTO(a.project), nil
}

// MoveSection reorders a section (work package swimlane) relative to the
// others. dir < 0 moves it up (toward WP-1), dir > 0 moves it down. Order is
// expressed by the order of packages; we reorder whole cluster blocks. The
// numeric WP-n labels are derived from this order (display-only).
func (a *App) MoveSection(cluster string, dir int) (*ProjectDTO, error) {
	// The "Phases" view lane is fixed (always first, no WP number) and is not
	// reorderable; moving it is a no-op.
	if plan.IsPhasesLane(cluster) {
		return toDTO(a.project), nil
	}
	// The numbered sections as the screen shows them, INCLUDING empty ones.
	order := a.project.WBS().Sections
	idx := -1
	for i, c := range order {
		if c == cluster {
			idx = i
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("section %q not found", cluster)
	}
	swap := idx - 1
	if dir > 0 {
		swap = idx + 1
	}
	if swap < 0 || swap >= len(order) {
		return toDTO(a.project), nil // already at an edge
	}
	a.pushUndo()
	order = append([]string(nil), order...)
	order[idx], order[swap] = order[swap], order[idx]
	// The new order is stored explicitly, so a section without actions can be
	// moved as well (it has no actions whose position could carry the order).
	a.project.SectionOrder = order
	// Keep the actions grouped in the same order (Phases pinned first).
	rank := map[string]int{"Phases": -1, "phases": -1}
	for i, c := range order {
		rank[c] = i
	}
	pkgs := make([]plan.WorkPackage, len(a.project.Packages))
	copy(pkgs, a.project.Packages)
	stableSortByRank(pkgs, rank)
	a.project.Packages = pkgs
	return toDTO(a.project), nil
}

func stableSortByRank(pkgs []plan.WorkPackage, rank map[string]int) {
	// simple stable insertion sort by cluster rank (small N)
	for i := 1; i < len(pkgs); i++ {
		j := i
		for j > 0 && rank[pkgs[j-1].Cluster] > rank[pkgs[j].Cluster] {
			pkgs[j-1], pkgs[j] = pkgs[j], pkgs[j-1]
			j--
		}
	}
}

// --- Milestones (zero-duration markers) ---

// AddMilestone creates a milestone in the given section (cluster) at a date.
// Milestones exist only at the main level (not inside a drill-down). The new
// milestone is undoable and becomes selectable. An empty cluster uses the
// current section. Returns the updated project.
func (a *App) AddMilestone(name, date, cluster string) (*ProjectDTO, error) {
	d, err := parseDate(date)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(name) == "" {
		name = "Milestone"
	}
	if cluster == "" {
		cluster = a.currentSection
	}
	a.pushUndo()
	id := a.uniqueMilestoneID(idFrom("m_", name))
	a.project.Milestones = append(a.project.Milestones, plan.Milestone{
		ID: id, Name: name, Date: d, Cluster: cluster,
	})
	a.selKind, a.selID = "milestone", id
	return toDTO(a.project), nil
}

// isUnlinkableMilestone reports whether the id is a milestone that renders as a
// full-height marker and therefore cannot be a dependency endpoint: the project
// Start/End markers (m_start/m_end), an unclustered milestone, or any milestone
// in the top "Phases" row.
func (a *App) isUnlinkableMilestone(id string) bool {
	m := a.project.MilestoneByID(id)
	if m == nil {
		return false
	}
	if m.ID == plan.MilestoneStartID || m.ID == plan.MilestoneEndID {
		return true
	}
	if strings.TrimSpace(m.Cluster) == "" {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(m.Cluster), "phases")
}

// MoveMilestone sets a milestone's date (drag by mouse). For the project
// Start/End milestones (m_start/m_end) this moves the project window bound.
// Undoable.
func (a *App) MoveMilestone(id, date string) (*ProjectDTO, error) {
	d, err := parseDate(date)
	if err != nil {
		return nil, err
	}
	a.pushUndo()
	for i := range a.project.Milestones {
		if a.project.Milestones[i].ID == id {
			a.project.Milestones[i].Date = d
			return toDTO(a.project), nil
		}
	}
	return nil, fmt.Errorf("milestone %q not found", id)
}

// SetMilestoneFields edits a milestone's name, date, and color.
func (a *App) SetMilestoneFields(id, name, date, color string) (*ProjectDTO, error) {
	for i := range a.project.Milestones {
		if a.project.Milestones[i].ID == id {
			if name != "" {
				a.project.Milestones[i].Name = name
			}
			if date != "" {
				d, err := parseDate(date)
				if err != nil {
					return nil, err
				}
				a.project.Milestones[i].Date = d
			}
			a.project.Milestones[i].Color = color
			return toDTO(a.project), nil
		}
	}
	return nil, fmt.Errorf("milestone %q not found", id)
}

// RemoveMilestone deletes a milestone. The project Start/End milestones
// (m_start/m_end) cannot be removed. Undoable.
func (a *App) RemoveMilestone(id string) (*ProjectDTO, error) {
	if id == plan.MilestoneStartID || id == plan.MilestoneEndID {
		return nil, fmt.Errorf("the project Start/End milestones cannot be deleted")
	}
	a.pushUndo()
	out := a.project.Milestones[:0]
	for _, m := range a.project.Milestones {
		if m.ID != id {
			out = append(out, m)
		}
	}
	a.project.Milestones = out
	if a.selKind == "milestone" && a.selID == id {
		a.selKind, a.selID = "", ""
	}
	return toDTO(a.project), nil
}

// uniqueMilestoneID returns an unused milestone id based on seed.
func (a *App) uniqueMilestoneID(seed string) string {
	if seed == "" || seed == "m_" {
		seed = "m_milestone"
	}
	taken := func(id string) bool {
		for i := range a.project.Milestones {
			if a.project.Milestones[i].ID == id {
				return true
			}
		}
		return false
	}
	if !taken(seed) {
		return seed
	}
	for n := 2; ; n++ {
		cand := fmt.Sprintf("%s%d", seed, n)
		if !taken(cand) {
			return cand
		}
	}
}

// --- Roles (Epic 3) ---

// AddSection creates a new named section (swimlane). A section is realized as a
// cluster; it exists as soon as an action references it, so to make an empty
// section visible we remember it in the section-color map with no color via a
// zero-width marker is not possible — instead we add a placeholder entry in
// SectionColors (empty value) so the lane renders. Actions can then be dragged
// into it or created with it as current section.
func (a *App) AddSection(name string) (*ProjectDTO, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("section name required")
	}
	a.pushUndo() // adding a section is undoable (Cmd-Z)
	if a.project.SectionColors == nil {
		a.project.SectionColors = map[string]string{}
	}
	if _, ok := a.project.SectionColors[name]; !ok {
		// A new section goes to the END (WP-n+1), never to the top: fix the
		// order of the existing sections first, then append the new one.
		order := a.project.SectionNames()
		a.project.SectionColors[name] = "" // placeholder so the lane shows
		a.project.SectionOrder = append(order, name)
	}
	a.currentSection = name
	return toDTO(a.project), nil
}

// RenameSection renames a section/cluster across all actions and metadata.
func (a *App) RenameSection(oldName, newName string) (*ProjectDTO, error) {
	newName = strings.TrimSpace(newName)
	if newName == "" {
		return nil, fmt.Errorf("new section name required")
	}
	for i := range a.project.Packages {
		if a.project.Packages[i].Cluster == oldName {
			a.project.Packages[i].Cluster = newName
		}
	}
	if a.project.SectionColors != nil {
		if c, ok := a.project.SectionColors[oldName]; ok {
			delete(a.project.SectionColors, oldName)
			a.project.SectionColors[newName] = c
		}
	}
	for i, c := range a.project.SectionOrder {
		if c == oldName {
			a.project.SectionOrder[i] = newName // the section keeps its place
		}
	}
	if a.currentSection == oldName {
		a.currentSection = newName
	}
	return toDTO(a.project), nil
}

// SetActionRoles replaces the set of role assignments on an action. Each entry
// is a role name + percent of the action's effort. The SAME role may appear
// more than once (e.g. two Developers) — each entry is an independent
// assignment. Each assignment's percent is capped at 100% individually
// (R-6.7); percents are NOT summed across lines and need not total 100%
// (the 100% rule is per line only, R-6.8).
func (a *App) SetActionRoles(id string, roleNames []string, percents []int) (*ProjectDTO, error) {
	wp := a.project.WorkPackage(id)
	if wp == nil {
		return nil, fmt.Errorf("action %q not found", id)
	}
	// remove all existing assignments to this action
	for i := range a.project.Roles {
		r := &a.project.Roles[i]
		out := r.Assignments[:0]
		for _, as := range r.Assignments {
			if as.WorkPackageID != id {
				out = append(out, as)
			}
		}
		r.Assignments = out
	}
	// add the new ones
	for i, name := range roleNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		pct := 0
		if i < len(percents) {
			pct = percents[i]
		}
		// cap a single assignment at 100% (per-line rule); negatives clamp to 0
		if pct > 100 {
			pct = 100
		}
		if pct < 0 {
			pct = 0
		}
		if !a.roleExists(name) {
			a.project.Roles = append(a.project.Roles, plan.Role{ID: a.project.NextRoleID(), Name: name})
		}
		for j := range a.project.Roles {
			if a.project.Roles[j].Name == name {
				a.project.Roles[j].Assignments = append(a.project.Roles[j].Assignments,
					plan.Assignment{WorkPackageID: id, Percent: pct})
				break
			}
		}
	}
	return toDTO(a.project), nil
}

// AddRole creates a new role.
// AddRole creates a new role with a distinct default color.
func (a *App) AddRole(name, description string) *ProjectDTO {
	a.project.Roles = append(a.project.Roles, plan.Role{
		ID: a.project.NextRoleID(), Name: name, Description: description, Color: a.project.NextRoleColor(),
	})
	return toDTO(a.project)
}

// SetRoleColor sets a role's color, enforcing that each role has a DISTINCT
// color (rejects a color already used by a different role).
func (a *App) SetRoleColor(name, color string) (*ProjectDTO, error) {
	color = strings.TrimSpace(color)
	for i := range a.project.Roles {
		if a.project.Roles[i].Name != name && a.project.Roles[i].Color == color && color != "" {
			return nil, fmt.Errorf("color %s is already used by role %q; each role needs a distinct color", color, a.project.Roles[i].Name)
		}
	}
	for i := range a.project.Roles {
		if a.project.Roles[i].Name == name {
			a.project.Roles[i].Color = color
			return toDTO(a.project), nil
		}
	}
	return nil, fmt.Errorf("role %q not found", name)
}

// SetRoleFields renames a role and/or updates its description. A rename keeps
// every reference intact (the role's assignments move with it and detail
// packages that name the role are updated) and is rejected when another role
// already has that name. An empty newName keeps the current name.
func (a *App) SetRoleFields(oldName, newName, description string) (*ProjectDTO, error) {
	r := a.project.RoleByName(oldName)
	if r == nil {
		return nil, fmt.Errorf("role %q not found", oldName)
	}
	if strings.TrimSpace(newName) != "" {
		if err := a.project.RenameRole(oldName, newName); err != nil {
			return nil, err
		}
	}
	r.Description = description
	return toDTO(a.project), nil
}

// RemoveRole deletes a role, but only when nothing references it: it is
// refused while the role is assigned to an action or named by a detail package
// (the error says where). Nothing changes on refusal.
func (a *App) RemoveRole(name string) (*ProjectDTO, error) {
	if err := a.project.DeleteRole(name); err != nil {
		return nil, err
	}
	return toDTO(a.project), nil
}

// AssignRole sets (or updates) a role's percentage on a work package. percent 0
// removes the assignment.
func (a *App) AssignRole(roleName, wpID string, percent int) (*ProjectDTO, error) {
	for i := range a.project.Roles {
		r := &a.project.Roles[i]
		if r.Name != roleName {
			continue
		}
		// update existing
		for j := range r.Assignments {
			if r.Assignments[j].WorkPackageID == wpID {
				if percent <= 0 {
					r.Assignments = append(r.Assignments[:j], r.Assignments[j+1:]...)
				} else {
					r.Assignments[j].Percent = percent
				}
				return toDTO(a.project), nil
			}
		}
		if percent > 0 {
			r.Assignments = append(r.Assignments, plan.Assignment{WorkPackageID: wpID, Percent: percent})
		}
		return toDTO(a.project), nil
	}
	return nil, fmt.Errorf("role %q not found", roleName)
}

// RoleLoad returns, for one role, its planned load per month bucket across the
// project timeline (Epic 3: see a role's load over the whole timeline, R-6.5).
// Each work-package share is spread evenly over the package's calendar days and
// summed per month. Values are display strings.
func (a *App) RoleLoad(roleName string) (map[string]string, error) {
	var role *plan.Role
	for i := range a.project.Roles {
		if a.project.Roles[i].Name == roleName {
			role = &a.project.Roles[i]
		}
	}
	if role == nil {
		return nil, fmt.Errorf("role %q not found", roleName)
	}
	return a.project.RoleLoadByMonth(role), nil
}

func (a *App) uniqueID(base string) string {
	exists := func(id string) bool {
		for i := range a.project.Packages {
			if a.project.Packages[i].ID == id {
				return true
			}
			for _, d := range a.project.Packages[i].Details {
				if d.ID == id {
					return true
				}
			}
		}
		return false
	}
	id := base
	for n := 2; exists(id); n++ {
		id = fmt.Sprintf("%s%d", base, n)
	}
	return id
}
