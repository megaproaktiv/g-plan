package plan

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// WriteMMD serializes the project's graphic content to a gmaid/Mermaid-Gantt
// document (requirements R-2.1.2, R-2.2). The .mmd is the graphic source of
// truth; roles live in the sibling .roles.json and are NOT written here.
//
// Layout of sections: Milestones, Phases, then one section per work package
// (3-line <br/> title: "WPn", name, estimation), each with a summary bar and
// its detail-package bars. Dependencies are emitted as `link` directives.
func (p *Project) WriteMMD(path string) error {
	return os.WriteFile(path, []byte(p.MMD()), 0o644)
}

// MMD returns the project serialized as a gmaid Gantt document string.
func (p *Project) MMD() string {
	var b strings.Builder
	b.WriteString("gantt\n")
	fmt.Fprintf(&b, "    title %s\n", p.Title)
	fmt.Fprintf(&b, "    dateFormat %s\n", orDefault(p.DateFormat, "YYYY-MM-DD"))
	fmt.Fprintf(&b, "    axisFormat %s\n", orDefault(p.AxisFormat, "%b %Y"))
	if p.TickEvery != "" {
		fmt.Fprintf(&b, "    tickInterval %s\n", p.TickEvery)
	}

	if len(p.Milestones) > 0 {
		b.WriteString("\n    section Milestones\n")
		for _, m := range p.Milestones {
			fmt.Fprintf(&b, "    %s :milestone, %s, %s, 0d\n",
				m.Name, m.ID, m.Date.Format(DateLayout))
		}
	}

	if len(p.Phases) > 0 {
		b.WriteString("\n    section Phases\n")
		for _, ph := range p.Phases {
			fmt.Fprintf(&b, "    %s :%s, %s, %s\n",
				ph.Name, ph.ID, ph.Start.Format(DateLayout), ph.End.Format(DateLayout))
		}
	}

	for i := range p.Packages {
		wp := &p.Packages[i]
		// section title: name / estimation + unit (the id is the task id, not
		// shown in the label so a plain "demo" has no WP prefix)
		title := fmt.Sprintf("%s<br/>%s %s", wp.Name, p.Format(wp.EffectiveEst()), p.unitAbbrev())
		fmt.Fprintf(&b, "\n    section %s\n", title)
		// main bar: a summary/rollup (Summiervorgang) when flagged, else normal
		tag := ""
		if wp.Summary {
			tag = "summary, "
		}
		fmt.Fprintf(&b, "    %s :%s%s, %s, %s\n",
			wp.Name, tag, wp.ID, wp.Start.Format(DateLayout), wp.End.Format(DateLayout))
		// detail packages
		for j := range wp.Details {
			d := &wp.Details[j]
			label := fmt.Sprintf("%s %s", d.Name, p.Format(d.Est))
			fmt.Fprintf(&b, "    %s :%s, %s, %s\n",
				label, d.ID, d.Start.Format(DateLayout), d.End.Format(DateLayout))
		}
	}

	if len(p.Deps) > 0 {
		b.WriteString("\n")
		for _, d := range p.Deps {
			kind := d.Kind
			if kind == "" {
				kind = "FS"
			}
			fmt.Fprintf(&b, "    link %s %s %s\n", d.From, kind, d.To)
		}
	}
	return b.String()
}

func (p *Project) unitAbbrev() string {
	if p.Unit == UnitHours {
		return "h"
	}
	return "d"
}

func orDefault(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

// sortedPackagesByStart is a helper for callers that want a stable, time-
// ordered view without mutating the project.
func (p *Project) sortedPackagesByStart() []WorkPackage {
	cp := make([]WorkPackage, len(p.Packages))
	copy(cp, p.Packages)
	sort.SliceStable(cp, func(i, j int) bool { return cp[i].Start.Before(cp[j].Start) })
	return cp
}
