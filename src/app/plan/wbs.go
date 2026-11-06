package plan

import (
	"fmt"
	"sort"
	"strings"
)

// The numbers shown on screen: sections are WP-1, WP-2 ..., actions and
// milestones in a section share one n.k sequence (1.1, 1.2 ...), and the actions
// of the top "Phases" lane are P1, P2 ...
//
// WBS is the ONE place that computes these numbers. The DTO (what the user
// sees), the command resolvers (move 3.1 ..., select 2, ...) and the section
// ordering all use it, so a number typed in a command always means the same
// thing as the number on the screen.

// IsPhasesLane reports whether a section name is the Phases view lane. The lane
// is fixed at the top, has no WP number and is not a work package.
func IsPhasesLane(name string) bool { return strings.EqualFold(strings.TrimSpace(name), "phases") }

// WBS holds the display numbering of a project.
type WBS struct {
	// Sections are the numbered sections in display order (WP-1 first), without
	// the Phases lane.
	Sections []string
	// SectionNum maps a section name to its WP number.
	SectionNum map[string]int
	// ActionDisplay maps an action id to its display id: "n.k" or "Pk".
	ActionDisplay map[string]string
	// MilestoneDisplay maps a milestone id to "n.k" (milestones that belong to a
	// numbered section only).
	MilestoneDisplay map[string]string

	byDisplay          map[string]string // lower-case display id -> action id
	milestoneByDisplay map[string]string
}

// ActionByDisplay returns the id of the action with the given display id
// ("2.1", "P3"; case-insensitive), or "" if there is none.
func (w WBS) ActionByDisplay(display string) string {
	return w.byDisplay[strings.ToLower(strings.TrimSpace(display))]
}

// MilestoneByDisplay returns the id of the milestone with the given "n.k", or "".
func (w WBS) MilestoneByDisplay(display string) string {
	return w.milestoneByDisplay[strings.TrimSpace(display)]
}

// SectionNames returns every section of the project except the Phases lane, in
// display order: first the sections of the explicit SectionOrder (as far as they
// still exist), then the remaining ones that hold actions in the order they
// first appear, then the remaining empty declared sections alphabetically.
func (p *Project) SectionNames() []string {
	exists := map[string]bool{}
	var used []string
	for i := range p.Packages {
		c := p.Packages[i].Cluster
		if IsPhasesLane(c) || exists[c] {
			continue
		}
		exists[c] = true
		used = append(used, c)
	}
	var empty []string
	for name := range p.SectionColors {
		if IsPhasesLane(name) || exists[name] {
			continue
		}
		exists[name] = true
		empty = append(empty, name)
	}
	sort.Strings(empty)
	base := append(used, empty...)

	placed := map[string]bool{}
	out := make([]string, 0, len(base))
	for _, name := range p.SectionOrder { // explicit order first
		if exists[name] && !placed[name] {
			placed[name] = true
			out = append(out, name)
		}
	}
	for _, name := range base { // everything not mentioned keeps its natural position
		if !placed[name] {
			placed[name] = true
			out = append(out, name)
		}
	}
	return out
}

// WBS computes the display numbering of the project.
func (p *Project) WBS() WBS {
	w := WBS{
		Sections:           p.SectionNames(),
		SectionNum:         map[string]int{},
		ActionDisplay:      map[string]string{},
		MilestoneDisplay:   map[string]string{},
		byDisplay:          map[string]string{},
		milestoneByDisplay: map[string]string{},
	}
	for i, name := range w.Sections {
		w.SectionNum[name] = i + 1
	}
	// actions and milestones of a section share one sequence ordered by date
	for _, name := range w.Sections {
		n := w.SectionNum[name]
		for k, it := range p.SectionSequence(name) {
			display := fmt.Sprintf("%d.%d", n, k+1)
			if it.Kind == "milestone" {
				w.MilestoneDisplay[it.ID] = display
				w.milestoneByDisplay[display] = it.ID
			} else {
				w.ActionDisplay[it.ID] = display
				w.byDisplay[strings.ToLower(display)] = it.ID
			}
		}
	}
	// the Phases lane: P1, P2 ... in action order
	k := 0
	for i := range p.Packages {
		if IsPhasesLane(p.Packages[i].Cluster) {
			k++
			display := fmt.Sprintf("P%d", k)
			w.ActionDisplay[p.Packages[i].ID] = display
			w.byDisplay[strings.ToLower(display)] = p.Packages[i].ID
		}
	}
	return w
}
