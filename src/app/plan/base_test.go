package plan

import "testing"

// Storage is ALWAYS minutes. Day mode (HoursPerDay=8): 1 day = 480 min.
func TestBaseDayMode(t *testing.T) {
	p := NewProject("t", UnitDays)
	cases := []struct {
		in   string
		base Base
		out  string
	}{
		{"40", 19200, "40"},   // 40 d = 40*480 min
		{"1,25", 600, "1,25"}, // 1.25 d = 600 min (480 + 120)
		{"0,5", 240, "0,5"},   // half a day = 240 min
		{"1", 480, "1"},       // 1 day = 480 min
		{"2,5", 1200, "2,5"},  // 2.5 d = 1200 min
	}
	for _, c := range cases {
		got, err := p.Parse(c.in)
		if err != nil {
			t.Fatalf("Parse(%q): %v", c.in, err)
		}
		if got != c.base {
			t.Errorf("Parse(%q) = %d base, want %d", c.in, got, c.base)
		}
		if s := p.Format(c.base); s != c.out {
			t.Errorf("Format(%d) = %q, want %q", c.base, s, c.out)
		}
	}
}

// Hour mode: 1 hour = 60 min.
func TestBaseHourMode(t *testing.T) {
	p := NewProject("t", UnitHours)
	cases := []struct {
		in   string
		base Base
		out  string
	}{
		{"1", 60, "1"},
		{"1,5", 90, "1,5"},
		{"0,25", 15, "0,25"},
		{"8", 480, "8"},
		{"0,5", 30, "0,5"},
	}
	for _, c := range cases {
		got, err := p.Parse(c.in)
		if err != nil {
			t.Fatalf("Parse(%q): %v", c.in, err)
		}
		if got != c.base {
			t.Errorf("Parse(%q) = %d base, want %d", c.in, got, c.base)
		}
		if s := p.Format(c.base); s != c.out {
			t.Errorf("Format(%d) = %q, want %q", c.base, s, c.out)
		}
	}
}

// THE P1.5 BUG: a value entered in hours must display as a day fraction after
// switching the project to day mode (storage is unit-independent minutes).
func TestUnitSwitchReinterpretsCorrectly(t *testing.T) {
	p := NewProject("t", UnitHours) // enter in hours
	v, _ := p.Parse("12")           // 12 h -> 720 min
	if v != 720 {
		t.Fatalf("12h parsed to %d min, want 720", v)
	}
	p.Unit = UnitDays // switch display to days (HoursPerDay=8)
	if got := p.Format(v); got != "1,5" {
		t.Errorf("12h shown in day mode = %q, want \"1,5\"", got)
	}
}

func TestWholeDisplayUnit(t *testing.T) {
	pd := NewProject("t", UnitDays)
	if !pd.IsWholeDisplayUnit(480) { // 1 d
		t.Error("480min should be whole days")
	}
	if pd.IsWholeDisplayUnit(600) { // 1.25 d
		t.Error("600min should NOT be whole days")
	}
	ph := NewProject("t", UnitHours)
	if !ph.IsWholeDisplayUnit(60) {
		t.Error("60min should be whole hours")
	}
	if ph.IsWholeDisplayUnit(90) {
		t.Error("90min should NOT be whole hours")
	}
}

func TestTotalAndRoleEstIntegerExact(t *testing.T) {
	p := NewProject("t", UnitDays)
	p.Packages = []WorkPackage{
		{ID: "wp1", Est: p.BaseFromWholeDisplay(40), Mode: ModeFixed},
		{ID: "wp2", Est: p.BaseFromWholeDisplay(15), Mode: ModeFixed},
	}
	if got := p.TotalEst(); got != p.BaseFromWholeDisplay(55) {
		t.Errorf("TotalEst = %d, want %d", got, p.BaseFromWholeDisplay(55))
	}
	r := Role{Name: "x", Assignments: []Assignment{{WorkPackageID: "wp1", Percent: 50}}}
	// 50% of 40 days = 20 days
	if got := p.RoleEst(&r); got != p.BaseFromWholeDisplay(20) {
		t.Errorf("RoleEst = %d, want %d", got, p.BaseFromWholeDisplay(20))
	}
}
