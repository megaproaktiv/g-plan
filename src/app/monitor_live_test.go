package main

import (
	"os"
	"testing"
)

// Live, read-only run of a real project file against the real external system.
// Skipped unless PTPM_LIVE_PTPLAN points at a .ptplan file (it is only read,
// never saved). Credentials come from ~/.pt-pm/config.yaml.
//
//	PTPM_LIVE_PTPLAN=~/Downloads/test-assignment.ptplan go test . -run LiveProjectFile -v
func TestLiveProjectFileSync(t *testing.T) {
	path := os.Getenv("PTPM_LIVE_PTPLAN")
	if path == "" {
		t.Skip("set PTPM_LIVE_PTPLAN to a .ptplan file to run against the live system")
	}
	a := NewApp()
	if _, err := a.OpenProject(path); err != nil {
		t.Fatal(err)
	}
	sum, err := a.SyncMonitor()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("range %s..%s, %d bookings fetched, project actual %s %s, unassigned %s (%d)",
		sum.From, sum.To, sum.Fetched, sum.Actual, a.project.Unit, sum.UnmatchedActual, sum.UnmatchedEntries)
	for _, s := range sum.Sections {
		for _, d := range s.Details {
			t.Logf("action %q ext=%q default=%v: total %s h (%s d) in %d | matched %s h (%s d) in %d | unassigned %s h (%s d) in %d",
				d.Name, d.ExternalID, d.IsDefault, d.ActualHours, d.ActualDays, d.Entries,
				d.MatchedHours, d.MatchedDays, d.MatchedEntries, d.DefaultHours, d.DefaultDays, d.DefaultEntries)
		}
	}
	for _, w := range toDTO(a.project).Packages {
		t.Logf("planned %q: %s %s = %s h = %s d", w.Name, w.Est, a.project.Unit, w.EstHours, w.EstDays)
	}
}
