package plan

import (
	"fmt"
	"os"
	"strings"
)

// WriteMarkdown regenerates the read-only <project>.md projection from the
// current model (requirement R-2.1.4). This file is OUTPUT ONLY — it is never
// parsed back. It exists so the plan can be read as plain text by humans
// (including screen-reader users) and by LLMs.
func (p *Project) WriteMarkdown(path string) error {
	return os.WriteFile(path, []byte(p.Markdown()), 0o644)
}

// Markdown returns the human-readable projection of the whole plan.
func (p *Project) Markdown() string {
	var b strings.Builder
	unit := p.unitWord()
	wbs := p.WBS()

	fmt.Fprintf(&b, "# %s\n\n", p.Title)
	b.WriteString("> Generated projection — do not edit. Regenerated from the `.mmd` graphic\n")
	b.WriteString("> source and the `.roles.json` on every save.\n\n")

	fmt.Fprintf(&b, "- **Unit:** %s", unit)
	if p.Unit == UnitDays {
		fmt.Fprintf(&b, " (stored as hours, %d h/day)", p.hpd())
	} else {
		b.WriteString(" (stored as minutes)")
	}
	b.WriteString("\n")
	fmt.Fprintf(&b, "- **Total estimation:** %s %s (calculated from work packages)\n",
		p.Format(p.TotalEst()), unit)
	fmt.Fprintf(&b, "- **Work packages:** %d · **Roles:** %d\n\n", len(p.Packages), len(p.Roles))

	// Milestones
	if len(p.Milestones) > 0 {
		b.WriteString("## Milestones\n\n| Milestone | Date |\n|---|---|\n")
		for _, m := range p.Milestones {
			fmt.Fprintf(&b, "| %s | %s |\n", m.Name, m.Date.Format(DateLayout))
		}
		b.WriteString("\n")
	}

	// Phases
	if len(p.Phases) > 0 {
		b.WriteString("## Phases\n\n| Phase | Start | End |\n|---|---|---|\n")
		for _, ph := range p.Phases {
			fmt.Fprintf(&b, "| %s | %s | %s |\n", ph.Name, ph.Start.Format(DateLayout), ph.End.Format(DateLayout))
		}
		b.WriteString("\n")
	}

	// Work packages
	b.WriteString("## Work packages\n\n")
	fmt.Fprintf(&b, "| # | Name | Start | End | Est (%s) | Mode | Cluster |\n", unit)
	b.WriteString("|---|---|---|---|--:|---|---|\n")
	for i := range p.Packages {
		wp := &p.Packages[i]
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %s |\n",
			wbs.ActionDisplay[wp.ID], wp.Name,
			wp.Start.Format(DateLayout), wp.End.Format(DateLayout),
			p.Format(wp.EffectiveEst()), wp.Mode, wp.Cluster)
	}
	fmt.Fprintf(&b, "| | **Total** | | | **%s** | | |\n\n", p.Format(p.TotalEst()))

	// Detail packages per work package
	for i := range p.Packages {
		wp := &p.Packages[i]
		if len(wp.Details) == 0 {
			continue
		}
		fmt.Fprintf(&b, "### %s — %s (details)\n\n", wbs.ActionDisplay[wp.ID], wp.Name)
		fmt.Fprintf(&b, "| Detail | Start | End | Est (%s) |\n|---|---|---|--:|\n", unit)
		for j := range wp.Details {
			d := &wp.Details[j]
			fmt.Fprintf(&b, "| %s | %s | %s | %s |\n",
				d.Name, d.Start.Format(DateLayout), d.End.Format(DateLayout), p.Format(d.Est))
		}
		if wp.Mode == ModeFixed {
			if diff := wp.Reconcile(); diff != 0 {
				fmt.Fprintf(&b, "\n> Fixed level-1 estimate %s vs detail sum %s (difference %s).\n",
					p.Format(wp.Est), p.Format(wp.DetailSum()), p.Format(diff))
			}
		}
		b.WriteString("\n")
	}

	// Roles
	if len(p.Roles) > 0 {
		b.WriteString("## Roles\n\n")
		for i := range p.Roles {
			r := &p.Roles[i]
			fmt.Fprintf(&b, "### %s\n\n", r.Name)
			if r.Description != "" {
				fmt.Fprintf(&b, "%s\n\n", r.Description)
			}
			fmt.Fprintf(&b, "Total load: **%s %s**\n\n", p.Format(p.RoleEst(r)), unit)
			if len(r.Assignments) > 0 {
				fmt.Fprintf(&b, "| Work package | %% | Share (%s) | Consumed |\n|---|--:|--:|--:|\n", unit)
				for _, a := range r.Assignments {
					wp := p.WorkPackage(a.WorkPackageID)
					name := a.WorkPackageID
					var share Base
					if wp != nil {
						name = strings.TrimSpace(wbs.ActionDisplay[wp.ID] + " " + wp.Name)
						share = Base(int64(wp.EffectiveEst()) * int64(a.Percent) / 100)
					}
					fmt.Fprintf(&b, "| %s | %d | %s | %s |\n", name, a.Percent, p.Format(share), p.AssignmentConsumed(a))
				}
				b.WriteString("\n")
			}
		}
	}

	// Dependencies
	if len(p.Deps) > 0 {
		b.WriteString("## Dependencies\n\n| From | Type | To |\n|---|---|---|\n")
		for _, d := range p.Deps {
			fmt.Fprintf(&b, "| %s | %s | %s |\n", d.From, d.Kind, d.To)
		}
		b.WriteString("\n")
	}

	// Consistency warnings
	if w := p.Validate(); len(w) > 0 {
		b.WriteString("## Consistency warnings\n\n")
		for _, line := range w {
			fmt.Fprintf(&b, "- %s\n", line)
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (p *Project) unitWord() string {
	if p.Unit == UnitHours {
		return "hours"
	}
	return "days"
}
