package main

import (
	"fmt"
	"strconv"
	"strings"
)

// Session-stable numeric display IDs. Actions (and sections) get small integers
// 1..N assigned in order. They are NOT persisted and MAY change across saves,
// but are dense (reused after deletes) and stable while the app computes them
// from the current order. The internal string id remains the real key used by
// the .mmd / links.
//
// actionNum returns, for each action internal id, its 1-based number in
// package order.
func (a *App) actionNums() map[string]int {
	out := make(map[string]int, len(a.project.Packages))
	for i := range a.project.Packages {
		out[a.project.Packages[i].ID] = i + 1
	}
	return out
}

// actionByNum resolves a 1-based action number to its internal id ("" if none).
func (a *App) actionByNum(n int) string {
	if n >= 1 && n <= len(a.project.Packages) {
		return a.project.Packages[n-1].ID
	}
	return ""
}

// resolveActionRef accepts either a bare internal id, a "#n" / "n" numeric
// reference, or a dotted WBS reference "n.k" (work package n, action k), and
// returns the internal action id.
func (a *App) resolveActionRef(ref string) string {
	ref = strings.TrimSpace(ref)
	ref = strings.TrimPrefix(ref, "#")
	// the number shown on screen: "2.1" or, for the Phases lane, "P3"
	if id := a.project.WBS().ActionByDisplay(ref); id != "" {
		return id
	}
	// dotted WBS reference: "n.k" -> action, "n.k.m" -> the detail's parent.
	if strings.Contains(ref, ".") {
		if id, _ := a.resolveWBSRef(ref); id != "" {
			return id
		}
	}
	if n, err := strconv.Atoi(ref); err == nil {
		if id := a.actionByNum(n); id != "" {
			return id
		}
	}
	// fall back: maybe it is already an internal id
	if a.project.WorkPackage(ref) != nil {
		return ref
	}
	return ""
}

// resolveTaskRef resolves a reference to ANY task — a level-1/2 work package OR
// a level-3 detail (sub-action). Detail sub-actions are actions too, so chat
// commands (link, est, assign, move) accept them. It returns the task's real
// internal id: the DETAIL id when the ref is a detail WBS "n.k.m", otherwise
// the work-package id. Also accepts a bare internal detail id.
func (a *App) resolveTaskRef(ref string) string {
	ref = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(ref), "#"))
	// dotted WBS: n.k.m -> detail id; n.k -> work-package id
	if strings.Contains(ref, ".") {
		wpID, detID := a.resolveWBSRef(ref)
		if detID != "" {
			return detID
		}
		if wpID != "" {
			return wpID
		}
	}
	// a bare internal id that is a detail of some work package
	if d, _ := a.project.DetailByID(ref); d != nil {
		return ref
	}
	// otherwise fall back to the action resolver (#n / n / n.k / wp id)
	return a.resolveActionRef(ref)
}

// resolveWBSRef resolves a dotted display id to (actionID, detailID). For
// "n.k" it returns the action id and an empty detail id. For "n.k.m" it
// returns the parent action id and the detail id. Returns ("","") if not
// found. The dotted ids mirror dto.go's DisplayID computation: work packages
// are numbered per section in first-seen order, actions per section, details
// per action.
func (a *App) resolveWBSRef(ref string) (string, string) {
	ref = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(ref), "#"))
	parts := strings.Split(ref, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return "", ""
	}
	for _, p := range parts {
		if v, err := strconv.Atoi(strings.TrimSpace(p)); err != nil || v < 1 {
			return "", ""
		}
	}
	// "n.k" is looked up in the SAME numbering the screen shows (plan.WBS)
	id := a.project.WBS().ActionByDisplay(parts[0] + "." + parts[1])
	if id == "" {
		return "", ""
	}
	if len(parts) == 2 {
		return id, ""
	}
	// n.k.m -> the m-th detail of this action
	wp := a.project.WorkPackage(id)
	m, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
	if wp != nil && m >= 1 && m <= len(wp.Details) {
		return wp.ID, wp.Details[m-1].ID
	}
	return "", ""
}

// wpSectionNums returns the WP-n numbering EXACTLY as shown in the UI (the DTO
// SectionNums / lane labels): it is plan.WBS, the one shared numbering.
func (a *App) wpSectionNums() map[string]int { return a.project.WBS().SectionNum }

func mustAtoi(s string) (int, error) {
	s = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(s), "#"))
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("not a number: %q", s)
	}
	return n, nil
}

// roleNums returns, for each role name, its 1-based R-number in project order.
// The UI shows this as "R<n>" in front of the role.
func (a *App) roleNums() map[string]int {
	out := make(map[string]int, len(a.project.Roles))
	for i := range a.project.Roles {
		out[a.project.Roles[i].Name] = i + 1
	}
	return out
}

// resolveRoleRef accepts an "R<n>" / "<n>" numeric role reference, or a bare
// role name, and returns the role name ("" if not found). The number is the
// 1-based R-number shown in the UI.
func (a *App) resolveRoleRef(ref string) string {
	ref = strings.TrimSpace(ref)
	// strip an optional leading R/r (R1 -> 1)
	num := ref
	if len(num) >= 2 && (num[0] == 'R' || num[0] == 'r') {
		num = num[1:]
	}
	// R<n> is the role's stable ID (not its list position), so references
	// keep working after other roles are deleted.
	if n, err := strconv.Atoi(num); err == nil {
		if r := a.project.RoleByID(n); r != nil {
			return r.Name
		}
	}
	// fall back: a bare role name
	for i := range a.project.Roles {
		if a.project.Roles[i].Name == ref {
			return ref
		}
	}
	return ""
}

// --- shared action+milestone WBS numbering within a section ---
//
// Actions and milestones in the SAME section share one n.k sequence, ordered
// by date. The ordering lives in plan.Project.SectionSequence so both the DTO
// (display ids) and these resolvers agree.

// milestoneByWBS resolves a dotted WBS code "n.k" to a milestone id if the
// k-th item in work package n is a milestone, else "".
func (a *App) milestoneByWBS(ref string) string {
	return a.project.WBS().MilestoneByDisplay(ref)
}
