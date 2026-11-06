package extsys

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/megaproaktiv/g-plan/src/app/plan"
)

// Live test against the real Aerion system. READ-ONLY (GET requests plus the
// OAuth token request). Skipped unless PTPM_LIVE=1; uses the credentials in
// ~/.pt-pm/config.yaml. PTPM_LIVE_PROJECT selects the Aerion project id.
//
//	PTPM_LIVE=1 PTPM_LIVE_PROJECT=96998 go test ./extsys -run Live -v
func TestLiveAerionReadOnly(t *testing.T) {
	if os.Getenv("PTPM_LIVE") != "1" {
		t.Skip("set PTPM_LIVE=1 to run against the live Aerion system")
	}
	pid := os.Getenv("PTPM_LIVE_PROJECT")
	if pid == "" {
		pid = "96998"
	}
	path, err := DefaultConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	drv, err := NewDriver(plan.SystemAerion, path)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	info, err := drv.Check(ctx, pid)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	t.Logf("project %s: %q status=%d start=%s end=%s", info.ID, info.Name, info.Status, info.StartDate, info.EndDate)

	from, to := day("2026-01-01"), day("2026-11-04")
	var lastFetched, lastTotal int
	all, err := drv.Fetch(ctx, pid, from, to, func(f, tot int) { lastFetched, lastTotal = f, tot })
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	t.Logf("fetched %d bookings (progress reported %d of %d)", len(all), lastFetched, lastTotal)
	if len(all) == 0 {
		t.Fatal("expected bookings for the test project")
	}
	if lastFetched != len(all) || lastTotal != len(all) {
		t.Errorf("progress %d/%d does not match %d bookings", lastFetched, lastTotal, len(all))
	}

	// paging must not lose or duplicate entries: refetch with tiny pages
	old := aerionPageSize
	aerionPageSize = 250
	defer func() { aerionPageSize = old }()
	paged, err := drv.Fetch(ctx, pid, from, to, nil)
	if err != nil {
		t.Fatalf("paged Fetch: %v", err)
	}
	sum := func(bs []plan.Booking) (n int, sec int64, ids map[int64]bool) {
		ids = map[int64]bool{}
		for _, b := range bs {
			sec += int64(b.Seconds)
			ids[b.ID] = true
		}
		return len(bs), sec, ids
	}
	n1, s1, ids1 := sum(all)
	n2, s2, ids2 := sum(paged)
	if n1 != n2 || s1 != s2 || len(ids1) != n1 || len(ids2) != n2 {
		t.Errorf("paging mismatch: %d/%ds/%d unique vs %d/%ds/%d unique", n1, s1, len(ids1), n2, s2, len(ids2))
	}
	t.Logf("total tracked %.1f h in %d..%s", float64(s1)/3600, from.Year(), to.Format(plan.DateLayout))

	// match a token that really occurs in the booking texts, to prove the
	// text-matching path against real data (the token is derived, not printed)
	var tok string
	for _, b := range all {
		for _, w := range splitWords(b.Text) {
			if len(w) >= 4 && len(w) <= plan.MaxExternalIDLen {
				tok = w
				break
			}
		}
		if tok != "" {
			break
		}
	}
	p := plan.NewProject("live", plan.UnitHours)
	st, en := from, to
	p.SetRange(st, en)
	p.Packages = []plan.WorkPackage{{ID: "a", Name: "A", Cluster: "S", Start: st, End: en, Mode: plan.ModeFixed, ExternalID: tok}}
	res := p.MonitorSummary(all, "")
	t.Logf("token of %d chars matched %d of %d bookings (%d s)", len(tok), res.TotalEntries, res.Fetched, res.TotalSeconds)
	if tok == "" || res.TotalEntries == 0 || res.TotalEntries+res.UnmatchedEntries != res.Fetched {
		t.Errorf("matching against live data failed: token found=%v matched=%d", tok != "", res.TotalEntries)
	}
}

func splitWords(s string) []string {
	var out []string
	cur := []rune{}
	flush := func() {
		if len(cur) > 0 {
			out = append(out, string(cur))
			cur = cur[:0]
		}
	}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			cur = append(cur, r)
		} else {
			flush()
		}
	}
	flush()
	return out
}
