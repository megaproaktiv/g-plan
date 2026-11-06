package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/megaproaktiv/g-plan/src/app/plan"
)

// mdCell makes text safe for a markdown table cell.
func mdCell(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "|", "\\|")
	return strings.TrimSpace(strings.NewReplacer("\r", " ", "\n", " ").Replace(s))
}

// monitorMarkdown renders the synchronization result as markdown. Compact lists
// the work packages (sections) and the project sum; detailed adds the tracked
// actions below each work package.
func monitorMarkdown(p *plan.Project, d *MonitorSummaryDTO, detailed bool) string {
	var b strings.Builder
	title := strings.TrimSpace(p.Title)
	if title == "" {
		title = "Untitled"
	}
	kind := "compact"
	if detailed {
		kind = "detailed"
	}
	fmt.Fprintf(&b, "# Monitoring: %s (%s)\n\n", mdCell(title), kind)
	fmt.Fprintf(&b, "Real effort booked from %s to %s, synchronized %s. Values in %ss.\n\n",
		d.From, d.To, d.At, p.Unit)
	b.WriteString("| Work package / Action | External ID | Estimated (tracked) | Actual | Remaining | Remaining % | Entries |\n")
	b.WriteString("|---|---|--:|--:|--:|--:|--:|\n")
	row := func(name, ext, est, act, rem, pct string, entries int) {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %d |\n",
			name, mdCell(ext), est, act, rem, pct, entries)
	}
	for _, s := range d.Sections {
		row("**"+mdCell(s.Name)+"**", fmt.Sprintf("%d / %d tracked", s.Tracked, s.Actions),
			s.EstTracked, s.Actual, s.Remaining, s.RemainingPct, s.Entries)
		if !detailed {
			continue
		}
		for _, a := range s.Details {
			ext := a.ExternalID
			if a.IsDefault {
				if ext != "" {
					ext += " "
				}
				ext += "(default action)"
			}
			row("&nbsp;&nbsp;"+mdCell(a.Name), ext, a.Est, a.Actual, a.Remaining, a.RemainingPct, a.Entries)
		}
	}
	row("**Project**", fmt.Sprintf("%d / %d tracked", trackedTotal(d), actionsTotal(d)),
		d.EstTracked, d.Actual, d.Remaining, d.RemainingPct, d.Entries)
	if !(d.DefaultAction != "" && d.UnmatchedEntries == 0) {
		row("Bookings without a matching External ID", "", "", d.UnmatchedActual, "", "", d.UnmatchedEntries)
	}
	fmt.Fprintf(&b, "\nRemaining = estimated (tracked) - actual; negative means over budget. %d bookings fetched.\n", d.Fetched)
	return b.String()
}

func trackedTotal(d *MonitorSummaryDTO) (n int) {
	for _, s := range d.Sections {
		n += s.Tracked
	}
	return
}

func actionsTotal(d *MonitorSummaryDTO) (n int) {
	for _, s := range d.Sections {
		n += s.Actions
	}
	return
}

// ExportMonitorMarkdown writes the last synchronization result as two markdown
// files: "<name>-compact.md" (work packages only) and "<name>-detailed.md"
// (work packages and their actions). A save dialog picks the base name; the
// two paths are returned. Empty when the dialog was cancelled.
func (a *App) ExportMonitorMarkdown() ([]string, error) {
	d := a.monitorSummaryDTO()
	if d == nil {
		return nil, fmt.Errorf("nothing to export: synchronize first")
	}
	name := strings.TrimSpace(a.project.Title)
	if name == "" {
		name = "Untitled"
	}
	dlg := a.app.Dialog.SaveFile().
		SetMessage("Export monitoring as markdown (a -compact and a -detailed file are written)").
		SetFilename(name+"-monitoring.md").
		AddFilter("Markdown (*.md)", "*.md")
	if ld := lastDir(); ld != "" {
		dlg = dlg.SetDirectory(ld)
	}
	path, err := dlg.PromptForSingleSelection()
	if err != nil || path == "" {
		return nil, err
	}
	return a.writeMonitorMarkdown(path, d)
}

// writeMonitorMarkdown writes the two versions next to the chosen path.
func (a *App) writeMonitorMarkdown(path string, d *MonitorSummaryDTO) ([]string, error) {
	base := strings.TrimSuffix(path, filepath.Ext(path))
	var out []string
	for _, v := range []struct {
		suffix   string
		detailed bool
	}{{"-compact.md", false}, {"-detailed.md", true}} {
		f := base + v.suffix
		if err := os.WriteFile(f, []byte(monitorMarkdown(a.project, d, v.detailed)), 0o644); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	rememberDir(filepath.Dir(path))
	return out, nil
}
