package plan

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"pgregory.net/rapid"
)

func TestMatchExternalIDs(t *testing.T) {
	ids := []string{"WP-1", "WP-12", "alpha"}
	cases := []struct {
		text string
		want []int
	}{
		{"worked on WP-1 today", []int{0}},
		{"worked on wp-1", []int{0}},           // case-insensitive
		{"WP-12 review", []int{1}},             // not mistaken for WP-1
		{"WP-123", nil},                        // token boundary
		{"xWP-1", nil},                         // preceded by a letter
		{"(WP-1)", []int{0}},                   // punctuation is a boundary
		{"WP-1/WP-12", []int{0, 1}},            // several references: all, in text order
		{"WP-12 and WP-1", []int{1, 0}},        // text order, not id order
		{"alpha WP-1 WP-1 alpha", []int{2, 0}}, // each id once
		{"Alpha: design", []int{2}},
		{"", nil},
		{"nothing here", nil},
	}
	for _, c := range cases {
		got := MatchExternalIDs(c.text, ids)
		if len(got) != len(c.want) {
			t.Errorf("MatchExternalIDs(%q) = %v, want %v", c.text, got, c.want)
			continue
		}
		for k := range got {
			if got[k] != c.want[k] {
				t.Errorf("MatchExternalIDs(%q) = %v, want %v", c.text, got, c.want)
			}
		}
	}
	if len(MatchExternalIDs("anything", []string{""})) != 0 {
		t.Error("empty id must never match")
	}
	// an id inside a longer matching id is not a reference of its own
	if got := MatchExternalIDs("Phase 1 done", []string{"Phase 1", "1"}); len(got) != 1 || got[0] != 0 {
		t.Errorf("'1' inside 'Phase 1' must not count: %v", got)
	}
	// ...but a separate occurrence of the short id does
	if got := MatchExternalIDs("Phase 1 and 1 more", []string{"Phase 1", "1"}); len(got) != 2 {
		t.Errorf("separate '1' must count: %v", got)
	}
}

// Feature: monitoring - an id embedded between non-id characters is always found.
func TestProp_MatchFindsEmbeddedID(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		id := rapid.StringMatching(`[A-Za-z][A-Za-z0-9-]{0,15}`).Draw(t, "id")
		pre := rapid.SampledFrom([]string{"", "x ", "(", "- ", "[", "Done: "}).Draw(t, "pre")
		post := rapid.SampledFrom([]string{"", " y", ")", ".", ", more", "]"}).Draw(t, "post")
		got := MatchExternalIDs(pre+id+post, []string{id})
		if len(got) != 1 || got[0] != 0 {
			t.Fatalf("id %q not found in %q: %v", id, pre+id+post, got)
		}
	})
}

func TestSpreadSeconds(t *testing.T) {
	for _, c := range []struct {
		sec, n int
		want   []int64
	}{
		{3600, 1, []int64{3600}},
		{3600, 2, []int64{1800, 1800}},
		{3601, 2, []int64{1801, 1800}}, // remainder to the first
		{10, 3, []int64{4, 3, 3}},
		{0, 3, []int64{0, 0, 0}},
	} {
		got := SpreadSeconds(c.sec, c.n)
		for k := range c.want {
			if got[k] != c.want[k] {
				t.Errorf("SpreadSeconds(%d,%d) = %v, want %v", c.sec, c.n, got, c.want)
				break
			}
		}
	}
	if SpreadSeconds(5, 0) != nil {
		t.Error("n <= 0 gives no shares")
	}
}

// Feature: monitoring - spreading never loses or invents time, and the shares
// differ by at most one second ("evenly").
func TestProp_SpreadSecondsConservesAndIsEven(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		sec := rapid.IntRange(0, 1_000_000).Draw(t, "sec")
		n := rapid.IntRange(1, 12).Draw(t, "n")
		shares := SpreadSeconds(sec, n)
		var sum, min, max int64
		min = shares[0]
		for _, x := range shares {
			sum += x
			if x < min {
				min = x
			}
			if x > max {
				max = x
			}
		}
		if sum != int64(sec) || max-min > 1 {
			t.Fatalf("SpreadSeconds(%d,%d) = %v", sec, n, shares)
		}
	})
}

func TestFormatHoursAndDays(t *testing.T) {
	p := NewProject("x", UnitDays) // 8 h/day
	for _, c := range []struct {
		min         Base
		hours, days string
	}{
		{3150, "52,5", "6,56"},
		{240, "4", "0,5"},
		{19200, "320", "40"},
		{0, "0", "0"},
		{90, "1,5", "0,19"},
	} {
		if got := p.FormatHours(c.min); got != c.hours {
			t.Errorf("FormatHours(%d) = %q, want %q", c.min, got, c.hours)
		}
		if got := p.FormatDays(c.min); got != c.days {
			t.Errorf("FormatDays(%d) = %q, want %q", c.min, got, c.days)
		}
	}
	// the existing Format is unchanged
	if got := p.Format(19200); got != "40" {
		t.Errorf("Format = %q", got)
	}
}

func TestNormalizeExternalID(t *testing.T) {
	if v, err := NormalizeExternalID("  WP-1 "); err != nil || v != "WP-1" {
		t.Errorf("trim: %q %v", v, err)
	}
	if _, err := NormalizeExternalID(strings.Repeat("a", 16)); err != nil {
		t.Errorf("16 chars must be valid: %v", err)
	}
	if _, err := NormalizeExternalID(strings.Repeat("a", 17)); err == nil {
		t.Error("17 chars must be rejected")
	}
	if _, err := NormalizeExternalID(strings.Repeat("ä", 16)); err != nil {
		t.Errorf("limit counts characters, not bytes: %v", err)
	}
	if _, err := NormalizeExternalID("a\nb"); err == nil {
		t.Error("line break must be rejected")
	}
	if v, err := NormalizeExternalID(""); err != nil || v != "" {
		t.Errorf("empty is valid: %q %v", v, err)
	}
}

func TestMonitorValidate(t *testing.T) {
	if err := (Monitor{}).Validate(); err != nil {
		t.Errorf("off is always valid: %v", err)
	}
	if (Monitor{Enabled: true, ProjectID: "1"}).Validate() == nil {
		t.Error("enabled without system must fail")
	}
	if (Monitor{Enabled: true, System: SystemAerion}).Validate() == nil {
		t.Error("enabled without project id must fail")
	}
	if (Monitor{Enabled: true, System: SystemCSV, ProjectID: "1"}).Validate() == nil {
		t.Error("csv is not implemented")
	}
	if err := (Monitor{Enabled: true, System: SystemAerion, ProjectID: "1"}).Validate(); err != nil {
		t.Errorf("valid: %v", err)
	}
}

func monitorProject() *Project {
	p := NewProject("M", UnitHours)
	s, _ := ParseDateStr("2026-01-01")
	e, _ := ParseDateStr("2026-03-01")
	p.SetRange(s, e)
	h := func(n int64) Base { return p.BaseFromWholeDisplay(n) }
	p.Packages = []WorkPackage{
		{ID: "a", Name: "A", Cluster: "S1", Start: s, End: e, Mode: ModeFixed, Est: h(10), ExternalID: "WP-1"},
		{ID: "b", Name: "B", Cluster: "S1", Start: s, End: e, Mode: ModeFixed, Est: h(20)}, // not tracked
		{ID: "c", Name: "C", Cluster: "S2", Start: s, End: e, Mode: ModeFixed, Est: h(5), ExternalID: "WP-2"},
		{ID: "d", Name: "D", Cluster: "S3", Start: s, End: e, Mode: ModeFixed, Est: h(7)},
	}
	return p
}

func TestMonitorSummary(t *testing.T) {
	p := monitorProject()
	bookings := []Booking{
		{ID: 1, Seconds: 3600, Text: "WP-1 design"},
		{ID: 2, Seconds: 1800, Text: "wp-1: review"},
		{ID: 3, Seconds: 7200, Text: "WP-2"},
		{ID: 4, Seconds: 900, Text: "meeting without id"},
		{ID: 5, Seconds: 600, Text: "WP-12 other"}, // must not count for WP-1
	}
	s := p.MonitorSummary(bookings, "")
	if s.Fetched != 5 || s.TotalEntries != 3 || s.UnmatchedEntries != 2 {
		t.Fatalf("counts: %+v", s)
	}
	if s.TotalSeconds != 3600+1800+7200 || s.UnmatchedSeconds != 900+600 {
		t.Fatalf("seconds: %+v", s)
	}
	if len(s.Sections) != 3 || s.Sections[0].Name != "S1" {
		t.Fatalf("sections: %+v", s.Sections)
	}
	s1 := s.Sections[0]
	if s1.Actions != 2 || s1.Tracked != 1 || s1.Seconds != 5400 || s1.Entries != 2 {
		t.Errorf("S1: %+v", s1)
	}
	if s1.Est != p.BaseFromWholeDisplay(30) || s1.EstTracked != p.BaseFromWholeDisplay(10) {
		t.Errorf("S1 est: %d / %d", s1.Est, s1.EstTracked)
	}
	if s.Sections[2].Tracked != 0 || len(s.Sections[2].Details) != 0 {
		t.Errorf("S3 should have nothing tracked: %+v", s.Sections[2])
	}
	if s.TotalEst != p.BaseFromWholeDisplay(42) || s.TotalEstTracked != p.BaseFromWholeDisplay(15) {
		t.Errorf("totals est: %d / %d", s.TotalEst, s.TotalEstTracked)
	}
}

// Feature: monitoring - every fetched booking is either matched or unmatched,
// and section sums add up to the project sum.
func TestProp_SummaryConservesBookings(t *testing.T) {
	p := monitorProject()
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(0, 40).Draw(t, "n")
		var bs []Booking
		var total int64
		for i := 0; i < n; i++ {
			sec := rapid.IntRange(0, 40000).Draw(t, "sec")
			txt := rapid.SampledFrom([]string{"WP-1 x", "WP-2", "none", "WP-12", "", "WP-1 and WP-2", "WP-2 WP-1"}).Draw(t, "txt")
			bs = append(bs, Booking{ID: int64(i), Seconds: sec, Text: txt})
			total += int64(sec)
		}
		s := p.MonitorSummary(bs, "")
		if s.TotalSeconds+s.UnmatchedSeconds != total {
			t.Fatalf("seconds not conserved: %d+%d != %d", s.TotalSeconds, s.UnmatchedSeconds, total)
		}
		if s.TotalEntries+s.UnmatchedEntries != n {
			t.Fatalf("entries not conserved")
		}
		var sec int64
		for _, sec2 := range s.Sections {
			sec += sec2.Seconds
		}
		if sec != s.TotalSeconds {
			t.Fatalf("section sums %d != total %d", sec, s.TotalSeconds)
		}
	})
}

func TestBookingWithSeveralReferencesIsSpreadEvenly(t *testing.T) {
	p := monitorProject() // a: WP-1 (S1), c: WP-2 (S2)
	s := p.MonitorSummary([]Booking{
		{ID: 1, Seconds: 7200, Text: "WP-1 and WP-2 pairing session"},
		{ID: 2, Seconds: 3601, Text: "WP-2 / WP-1 review"}, // odd second: goes to the first mentioned (WP-2)
		{ID: 3, Seconds: 600, Text: "WP-1 alone"},
	}, "")
	a, c := s.Sections[0].Details[0], s.Sections[1].Details[0]
	if a.ID != "a" || a.Seconds != 3600+1800+600 || a.MatchedEntries != 3 {
		t.Errorf("a: %+v", a)
	}
	if c.ID != "c" || c.Seconds != 3600+1801 || c.MatchedEntries != 2 {
		t.Errorf("c: %+v", c)
	}
	if s.TotalSeconds != 7200+3601+600 || s.TotalEntries != 3 || s.UnmatchedEntries != 0 {
		t.Errorf("project totals count each booking once: %+v", s)
	}
	if s.Sections[0].Seconds+s.Sections[1].Seconds != s.TotalSeconds {
		t.Error("section sums must add up to the project total")
	}
}

func TestMatchedAndDefaultAreReportedApart(t *testing.T) {
	p := monitorProject()
	s := p.MonitorSummary([]Booking{
		{ID: 1, Seconds: 14400, Text: "WP-1 design"},
		{ID: 2, Seconds: 600, Text: "no reference"},
	}, "a") // the action with the id is also the default
	a := s.Sections[0].Details[0]
	if a.MatchedSeconds != 14400 || a.MatchedEntries != 1 || a.DefaultSeconds != 600 || a.DefaultEntries != 1 ||
		a.Seconds != 15000 || a.Entries != 2 || !a.IsDefault {
		t.Errorf("%+v", a)
	}
}

func TestSecondsToBase(t *testing.T) {
	for _, c := range []struct {
		sec  int64
		want Base
	}{{0, 0}, {29, 0}, {30, 1}, {60, 1}, {3600, 60}, {5400, 90}} {
		if got := SecondsToBase(c.sec); got != c.want {
			t.Errorf("SecondsToBase(%d) = %d, want %d", c.sec, got, c.want)
		}
	}
}

func TestMonitorPersistsAndOldFilesLoad(t *testing.T) {
	p := monitorProject()
	p.Monitor = Monitor{Enabled: true, System: SystemAerion, ProjectID: "96998"}
	path := filepath.Join(t.TempDir(), "m.ptplan")
	if err := p.SaveJSON(path); err != nil {
		t.Fatal(err)
	}
	q, err := OpenJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	if q.Monitor != p.Monitor {
		t.Errorf("monitor = %+v, want %+v", q.Monitor, p.Monitor)
	}
	if q.WorkPackage("a").ExternalID != "WP-1" || q.WorkPackage("b").ExternalID != "" {
		t.Error("external ids not round-tripped")
	}
	// a file without the monitor key keeps monitoring off
	legacy := filepath.Join(t.TempDir(), "old.ptplan")
	if err := NewProject("old", UnitDays).SaveJSON(legacy); err != nil {
		t.Fatal(err)
	}
	o, err := OpenJSON(legacy)
	if err != nil || o.Monitor.Enabled {
		t.Errorf("legacy: %+v %v", o.Monitor, err)
	}
}

func TestDefaultActionReceivesUnmatchedBookings(t *testing.T) {
	p := monitorProject() // actions: a(S1,WP-1) b(S1) c(S2,WP-2) d(S3)
	bs := []Booking{
		{ID: 1, Seconds: 3600, Text: "WP-1 design"},
		{ID: 2, Seconds: 900, Text: "meeting without id"},
		{ID: 3, Seconds: 600, Text: "WP-12 other"},
	}
	s := p.MonitorSummary(bs, "d")
	if s.UnmatchedEntries != 0 || s.UnmatchedSeconds != 0 {
		t.Errorf("with a default action nothing stays unassigned: %+v", s)
	}
	if s.DefaultAction != "d" || s.DefaultEntries != 2 || s.DefaultSeconds != 1500 {
		t.Errorf("default totals: %+v", s)
	}
	s3 := s.Sections[2] // d lives in S3
	if s3.Name != "S3" || s3.Seconds != 1500 || s3.Entries != 2 || s3.Tracked != 1 {
		t.Errorf("S3 must count the default action as tracked: %+v", s3)
	}
	if s3.EstTracked != p.BaseFromWholeDisplay(7) {
		t.Errorf("the default action's estimate must be in the tracked estimate: %d", s3.EstTracked)
	}
	if len(s3.Details) != 1 {
		t.Fatalf("details: %+v", s3.Details)
	}
	d := s3.Details[0]
	if d.ID != "d" || !d.IsDefault || d.ExternalID != "" || d.Seconds != 1500 || d.DefaultSeconds != 1500 || d.DefaultEntries != 2 {
		t.Errorf("default action row: %+v", d)
	}
	if s.TotalSeconds != 3600+1500 || s.TotalEntries != 3 {
		t.Errorf("project total must include the default bookings: %+v", s)
	}
	// matched bookings are never redirected to the default action
	a := s.Sections[0].Details[0]
	if a.ID != "a" || a.Seconds != 3600 || a.IsDefault || a.DefaultSeconds != 0 {
		t.Errorf("a: %+v", a)
	}
	// no default: unchanged behaviour
	n := p.MonitorSummary(bs, "")
	if n.UnmatchedEntries != 2 || n.DefaultAction != "" || n.TotalEntries != 1 {
		t.Errorf("no default: %+v", n)
	}
}

// The default action may itself have an External ID: it then gets its matched
// bookings plus the unassigned ones, shown separately.
func TestDefaultActionWithOwnExternalID(t *testing.T) {
	p := monitorProject()
	s := p.MonitorSummary([]Booking{
		{ID: 1, Seconds: 3600, Text: "WP-1 design"},
		{ID: 2, Seconds: 600, Text: "no id"},
	}, "a")
	a := s.Sections[0].Details[0]
	if a.ID != "a" || a.ExternalID != "WP-1" || !a.IsDefault || a.Seconds != 4200 || a.Entries != 2 || a.DefaultSeconds != 600 || a.DefaultEntries != 1 {
		t.Errorf("%+v", a)
	}
	if s.Sections[0].Tracked != 1 {
		t.Errorf("the action must be counted once: %+v", s.Sections[0])
	}
}

func TestDefaultActionUnknownIsIgnored(t *testing.T) {
	p := monitorProject()
	s := p.MonitorSummary([]Booking{{ID: 1, Seconds: 600, Text: "x"}}, "gone")
	if s.DefaultAction != "" || s.UnmatchedEntries != 1 || s.TotalEntries != 0 {
		t.Errorf("an unknown default action must behave like none: %+v", s)
	}
}

func TestHasAction(t *testing.T) {
	p := monitorProject()
	for id, want := range map[string]bool{"a": true, "d": true, "nope": false, "": false} {
		if got := p.HasAction(id); got != want {
			t.Errorf("HasAction(%q) = %v, want %v", id, got, want)
		}
	}
}

// Feature: monitoring - with a default action every booking is counted on some
// action: nothing stays unassigned and no seconds are lost or invented.
func TestProp_DefaultActionConservesSeconds(t *testing.T) {
	p := monitorProject()
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(0, 30).Draw(t, "n")
		var bs []Booking
		var total int64
		for i := 0; i < n; i++ {
			sec := rapid.IntRange(0, 40000).Draw(t, "sec")
			txt := rapid.SampledFrom([]string{"WP-1 x", "WP-2", "none", "WP-12", "", "WP-1 and WP-2", "WP-2 WP-1"}).Draw(t, "txt")
			bs = append(bs, Booking{ID: int64(i), Seconds: sec, Text: txt})
			total += int64(sec)
		}
		s := p.MonitorSummary(bs, rapid.SampledFrom([]string{"a", "b", "c", "d"}).Draw(t, "def"))
		var sec int64
		var ent int
		for _, x := range s.Sections {
			sec += x.Seconds
			ent += x.Entries
		}
		if s.UnmatchedSeconds != 0 || s.UnmatchedEntries != 0 {
			t.Fatalf("nothing may stay unassigned: %+v", s)
		}
		// seconds are conserved exactly; the project counts each booking once,
		// rows count references (a booking naming two actions shows on both)
		if sec != total || s.TotalSeconds != total || s.TotalEntries != n || ent < n {
			t.Fatalf("not conserved: sections %d/%d total %d/%d want %d/%d", sec, ent, s.TotalSeconds, s.TotalEntries, total, n)
		}
	})
}

func TestAddMonthsClamped(t *testing.T) {
	d := func(s string) time.Time { x, _ := time.Parse("2006-01-02", s); return x }
	for _, c := range []struct {
		in   string
		n    int
		want string
	}{
		{"2026-03-31", -1, "2026-02-28"}, // clamp, not 3 March
		{"2028-03-31", -1, "2028-02-29"}, // leap year
		{"2026-01-31", 1, "2026-02-28"},
		{"2026-05-31", 1, "2026-06-30"},
		{"2026-01-15", -1, "2025-12-15"}, // across the year
		{"2026-12-15", 1, "2027-01-15"},
		{"2026-01-01", -1, "2025-12-01"},
		{"2026-03-01", 1, "2026-04-01"},
		{"2026-10-31", 2, "2026-12-31"},
	} {
		if got := AddMonthsClamped(d(c.in), c.n).Format(DateLayout); got != c.want {
			t.Errorf("%s %+d months = %s, want %s", c.in, c.n, got, c.want)
		}
	}
}

// Feature: monitoring - one month before/after lands in the adjacent month,
// keeps the day when it exists there, and never moves the wrong way.
func TestProp_AddMonthsClamped(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		base := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, rapid.IntRange(0, 4000).Draw(t, "days"))
		n := rapid.SampledFrom([]int{-1, 1}).Draw(t, "n")
		got := AddMonthsClamped(base, n)
		wantMonth := time.Date(base.Year(), base.Month()+time.Month(n), 1, 0, 0, 0, 0, time.UTC)
		if got.Year() != wantMonth.Year() || got.Month() != wantMonth.Month() {
			t.Fatalf("%s %+d -> %s: wrong month", base.Format(DateLayout), n, got.Format(DateLayout))
		}
		lastDay := time.Date(wantMonth.Year(), wantMonth.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
		wantDay := base.Day()
		if wantDay > lastDay {
			wantDay = lastDay
		}
		if got.Day() != wantDay {
			t.Fatalf("%s %+d -> %s: day %d, want %d", base.Format(DateLayout), n, got.Format(DateLayout), got.Day(), wantDay)
		}
		if (n < 0) != got.Before(base) {
			t.Fatalf("moved the wrong way")
		}
	})
}

func TestMonitorSyncRange(t *testing.T) {
	d := func(s string) time.Time { x, _ := time.Parse("2006-01-02", s); return x }
	s, e := d("2026-03-31"), d("2026-08-31")
	for _, c := range []struct {
		m        Monitor
		from, to string
	}{
		{Monitor{}, "2026-03-31", "2026-08-31"},
		{Monitor{IncludeBefore: true}, "2026-02-28", "2026-08-31"},
		{Monitor{IncludeAfter: true}, "2026-03-31", "2026-09-30"},
		{Monitor{IncludeBefore: true, IncludeAfter: true}, "2026-02-28", "2026-09-30"},
	} {
		from, to := c.m.SyncRange(s, e)
		if from.Format(DateLayout) != c.from || to.Format(DateLayout) != c.to {
			t.Errorf("%+v: %s..%s, want %s..%s", c.m, from.Format(DateLayout), to.Format(DateLayout), c.from, c.to)
		}
	}
}

func TestIsSyncedAndApplySyncedEffort(t *testing.T) {
	p := monitorProject() // a: WP-1 (S1), b untracked, c: WP-2 (S2), d untracked
	a, b := p.WorkPackage("a"), p.WorkPackage("b")
	if p.IsSynced(a) {
		t.Error("nothing is synchronized while monitoring is off")
	}
	p.Monitor = Monitor{Enabled: true, System: SystemAerion, ProjectID: "1"}
	if !p.IsSynced(a) || p.IsSynced(b) || p.IsSynced(nil) {
		t.Error("an action is synchronized when it has an External ID")
	}
	p.Monitor.DefaultAction = "d"
	if !p.IsSynced(p.WorkPackage("d")) || p.IsSynced(b) {
		t.Error("the default action is synchronized too")
	}
	b.RealEst = 77 // manual value of an untracked action
	a.RealEst = 5  // stale value, must be replaced

	s := p.MonitorSummary([]Booking{
		{ID: 1, Seconds: 3 * 3600, Text: "WP-1 work"},
		{ID: 2, Seconds: 1800, Text: "unreferenced"},
	}, "d")
	if n := p.ApplySyncedEffort(s); n != 3 { // a, c, d
		t.Errorf("updated %d actions, want 3", n)
	}
	if a.RealEst != 180 || p.WorkPackage("c").RealEst != 0 || p.WorkPackage("d").RealEst != 30 {
		t.Errorf("real effort: a=%d c=%d d=%d", a.RealEst, p.WorkPackage("c").RealEst, p.WorkPackage("d").RealEst)
	}
	if b.RealEst != 77 {
		t.Errorf("an untracked action must keep its manual real effort, got %d", b.RealEst)
	}
	// monitoring off again: nothing is synchronized, values stay and become editable
	p.Monitor.Enabled = false
	if p.IsSynced(a) || a.RealEst != 180 {
		t.Error("switching monitoring off keeps the values but frees them")
	}
}

func TestMonitorSummaryIgnoresPhasesLane(t *testing.T) {
	p := &Project{Packages: []WorkPackage{
		{ID: "ph", Name: "Phase", Cluster: "Phases", ExternalID: "WP-1", Est: 60},
		{ID: "a", Name: "A", Cluster: "Build", ExternalID: "WP-1", Est: 60},
	}}
	p.Monitor.Enabled = true
	sum := p.MonitorSummary([]Booking{{Day: "2026-01-01", Seconds: 3600, Text: "WP-1"}}, "ph")
	if len(sum.Sections) != 1 || sum.Sections[0].Name != "Build" {
		t.Fatalf("sections = %+v, want only Build", sum.Sections)
	}
	if sum.DefaultAction != "" || sum.TotalSeconds != 3600 {
		t.Fatalf("default=%q total=%d", sum.DefaultAction, sum.TotalSeconds)
	}
	if p.IsSynced(&p.Packages[0]) || !p.IsSynced(&p.Packages[1]) {
		t.Fatal("IsSynced wrong for Phases lane")
	}
}

func TestMatchExternalIDsIgnoresCaseAndHyphens(t *testing.T) {
	for _, id := range []string{"WP1", "wp1", "WP-1", "wp-1"} {
		for _, txt := range []string{"WP1 x", "wp1 x", "Wp1 x", "WP-1 x", "wp-1 x", "a/WP-1/b"} {
			if got := MatchExternalIDs(txt, []string{id}); len(got) != 1 {
				t.Errorf("id %q should match %q, got %v", id, txt, got)
			}
		}
		for _, txt := range []string{"WP 1 x", "WP12", "WP-12", "xWP1", "x-WP1y"} {
			// "x-WP1y": a letter touches the id; "WP 1": a space is not a hyphen
			if got := MatchExternalIDs(txt, []string{id}); len(got) != 0 {
				t.Errorf("id %q must not match %q, got %v", id, txt, got)
			}
		}
	}
	if ExternalIDKey("WP-1") != ExternalIDKey("wp1") || ExternalIDKey("WP 1") == ExternalIDKey("WP1") {
		t.Error("ExternalIDKey must ignore case and hyphens, not spaces")
	}
}

func TestExternalIDOwnerTreatsHyphenAndCaseAsSame(t *testing.T) {
	p := &Project{Packages: []WorkPackage{{ID: "a", Name: "A", ExternalID: "WP-1"}}}
	if p.ExternalIDOwner("wp1", "b") == nil {
		t.Fatal("wp1 would match the same bookings as WP-1: must count as used")
	}
	if p.ExternalIDOwner("WP 1", "b") != nil {
		t.Fatal("WP 1 is a different id")
	}
}
