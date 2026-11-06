package main

import "testing"

// The action view shows the number of working days (Mon-Fri, no weekends).
func TestDTOWorkdaysExcludeWeekends(t *testing.T) {
	a := NewApp()
	for _, c := range []struct {
		start, end string
		want       int
	}{
		{"2026-11-02", "2026-11-06", 5},  // Mon-Fri
		{"2026-11-02", "2026-11-15", 10}, // two full weeks incl. two weekends
		{"2026-11-07", "2026-11-08", 0},  // Sat-Sun only
		{"2026-11-04", "2026-11-04", 1},  // single Wednesday
		{"2026-11-06", "2026-11-09", 2},  // Fri-Mon over a weekend
	} {
		dto, err := a.AddWorkPackage("w", c.start, c.end)
		if err != nil {
			t.Fatal(err)
		}
		got := dto.Packages[len(dto.Packages)-1].Workdays
		if got != c.want {
			t.Errorf("%s..%s: workdays = %d, want %d", c.start, c.end, got, c.want)
		}
	}
}
