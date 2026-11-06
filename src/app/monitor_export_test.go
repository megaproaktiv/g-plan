package main

import (
	"strings"
	"testing"

	"github.com/megaproaktiv/g-plan/src/app/plan"
)

func TestRemaining(t *testing.T) {
	rem, pct, ok := plan.Remaining(600, 3600*5) // 10 h estimated, 5 h booked
	if !ok || rem != 300 || pct != 50 {
		t.Fatalf("got %v %v %v", rem, pct, ok)
	}
	rem, pct, ok = plan.Remaining(600, 3600*15) // over budget
	if !ok || rem != -300 || pct != -50 {
		t.Fatalf("over budget: %v %v %v", rem, pct, ok)
	}
	if _, _, ok = plan.Remaining(0, 100); ok {
		t.Fatal("no estimation must give no percent")
	}
}

func TestMonitorMarkdownCompactAndDetailed(t *testing.T) {
	p := &plan.Project{Title: "Demo", Unit: plan.UnitHours}
	d := &MonitorSummaryDTO{
		From: "2026-01-01", To: "2026-02-01", At: "2026-02-02 10:00", Fetched: 3,
		EstTracked: "10", Actual: "4", Remaining: "6", RemainingPct: "60 %", Entries: 3,
		Sections: []MonitorSectionDTO{{
			Name: "Build", Actions: 2, Tracked: 1, EstTracked: "10", Actual: "4", Remaining: "6", RemainingPct: "60 %", Entries: 3,
			Details: []MonitorActionDTO{{Name: "A|B", ExternalID: "WP-1", Est: "10", Actual: "4", Remaining: "6", RemainingPct: "60 %", Entries: 3}},
		}},
	}
	c := monitorMarkdown(p, d, false)
	det := monitorMarkdown(p, d, true)
	if !strings.Contains(c, "Remaining %") || !strings.Contains(c, "**Build**") || strings.Contains(c, "WP-1") {
		t.Fatalf("compact wrong:\n%s", c)
	}
	if !strings.Contains(det, "A\\|B") || !strings.Contains(det, "WP-1") || !strings.Contains(det, "60 %") {
		t.Fatalf("detailed wrong:\n%s", det)
	}
}
