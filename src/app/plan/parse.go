package plan

import (
	"os"
	"regexp"
	"strings"

	"github.com/megaproaktiv/gmaid/gmodel"
	"github.com/megaproaktiv/gmaid/gparser"
)

// ReadMMD parses a gmaid/Mermaid-Gantt file into the project's graphic content
// (milestones, phases, work packages, detail packages, dependencies). Roles are
// loaded separately from the .roles.json. The embedded gmaid parser resolves
// dates, durations, after/until, tags, summary bars and link directives
// (requirement R-2.1.5 — the .mmd is authoritative graphic source).
func (p *Project) ReadMMD(path string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return p.parseMMD(string(src))
}

func (p *Project) parseMMD(src string) error {
	c, err := gparser.Parse(src)
	if err != nil {
		return err
	}
	p.Title = c.Title
	if c.DateFormat != "" {
		p.DateFormat = c.DateFormat
	}
	if c.AxisFormat != "" {
		p.AxisFormat = c.AxisFormat
	}

	p.Milestones = nil
	p.Phases = nil
	p.Packages = nil
	p.Deps = nil

	for _, sec := range c.Sections {
		kind := classifySection(sec)
		switch kind {
		case secMilestones:
			for _, t := range sec.Tasks {
				p.Milestones = append(p.Milestones, Milestone{ID: t.ID, Name: t.Label, Date: t.Start})
			}
		case secPhases:
			for _, t := range sec.Tasks {
				p.Phases = append(p.Phases, Phase{ID: t.ID, Name: t.Label, Start: t.Start, End: t.End})
			}
		case secWorkPackage:
			p.Packages = append(p.Packages, workPackageFromSection(sec))
		}
	}

	for _, l := range c.Links {
		p.Deps = append(p.Deps, Dependency{From: l.From, To: l.To, Kind: linkKindString(l.Kind)})
	}
	p.EnsureRange() // every plan carries a project Start and End milestone
	return nil
}

type sectionKind int

const (
	secWorkPackage sectionKind = iota
	secMilestones
	secPhases
	secOther
)

func classifySection(sec *gmodel.Section) sectionKind {
	first := sec.Name
	for _, sep := range []string{"<br", "\n"} {
		first = strings.SplitN(first, sep, 2)[0]
	}
	name := strings.TrimSpace(strings.ToLower(first))
	switch {
	case name == "milestones":
		return secMilestones
	case name == "phases":
		return secPhases
	default:
		// A work-package section has a summary rollup bar as its first task.
		if len(sec.Tasks) > 0 && sec.Tasks[0].Summary {
			return secWorkPackage
		}
		// Fall back: any section with tasks that is not milestones/phases is
		// treated as a work package so hand-written plans still import.
		if len(sec.Tasks) > 0 {
			return secWorkPackage
		}
		return secOther
	}
}

// wpTitleRe extracts "WPn", name and estimation from a 3-line <br/> section
// title such as "WP4<br/>Pippin<br/>80 PD" (the PD/estimation line is advisory
// only; the authoritative estimate is set from the model/roles elsewhere).
var estLineRe = regexp.MustCompile(`([0-9]+(?:[.,][0-9]+)?)\s*\w*$`)

func workPackageFromSection(sec *gmodel.Section) WorkPackage {
	// gmaid normalizes <br/> to real newlines in section names; the WP name is
	// the first line of the title.
	norm := strings.ReplaceAll(strings.ReplaceAll(sec.Name, "<br/>", "\n"), "<br>", "\n")
	lines := strings.Split(norm, "\n")
	name := strings.TrimSpace(lines[0])
	// Legacy 3-line titles look like "WP1" / "Frodo" / "40 PD": if the first
	// line is an id-like token (WP<n>), the real name is the second line.
	if len(lines) >= 2 && isIDToken(name) {
		name = strings.TrimSpace(lines[1])
	}
	wp := WorkPackage{Name: name, Mode: ModeFixed}
	for i, t := range sec.Tasks {
		if i == 0 {
			// first task = the WP main bar (summary/rollup or normal)
			wp.ID = t.ID
			wp.Start = t.Start
			wp.End = t.End
			wp.Summary = t.Summary
			if name == "" {
				wp.Name = stripTrailingNumber(t.Label)
			}
			continue
		}
		wp.Details = append(wp.Details, DetailPackage{
			ID:    t.ID,
			Name:  stripTrailingNumber(t.Label),
			Start: t.Start,
			End:   t.End,
		})
	}
	return wp
}

// stripTrailingNumber removes a trailing estimation token from a detail label
// (e.g. "Tests 1,5" -> "Tests") so the name round-trips cleanly.
func stripTrailingNumber(label string) string {
	return strings.TrimSpace(estLineRe.ReplaceAllString(strings.TrimSpace(label), ""))
}

var idTokenRe = regexp.MustCompile(`^(?i)WP\d+$`)

func isIDToken(s string) bool { return idTokenRe.MatchString(strings.TrimSpace(s)) }

func linkKindString(k gmodel.LinkKind) string {
	switch k {
	case gmodel.LinkSS:
		return "SS"
	case gmodel.LinkFF:
		return "FF"
	case gmodel.LinkSF:
		return "SF"
	default:
		return "FS"
	}
}
