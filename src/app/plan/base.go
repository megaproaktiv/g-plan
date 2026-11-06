package plan

import (
	"fmt"
	"strings"
)

// Base storage model — ABSOLUTE and unit-independent.
//
// Every estimation is stored in MINUTES, regardless of the project's display
// unit. This way switching the display unit (days <-> hours) only changes how a
// value is rendered, never what it means. Conversions are exact integer math:
//
//   - Hour mode:  1 display unit (hour) == 60 minutes.
//   - Day mode:   1 display unit (day)  == 60 * HoursPerDay minutes.
//
// So 12 hours is always stored as 720 minutes; shown as "12" in hour mode and
// as "1,5" in day mode (720 / (60*8)).
//
// minutesPerDisplay returns how many stored minutes make up one display unit.
func (p *Project) minutesPerDisplay() int64 {
	if p.Unit == UnitHours {
		return 60
	}
	return 60 * int64(p.hpd())
}

// subUnitsPerDisplay is kept as an alias for existing call sites.
func (p *Project) subUnitsPerDisplay() int64 { return p.minutesPerDisplay() }

func (p *Project) unitSingular() string {
	if p.Unit == UnitHours {
		return "hour"
	}
	return "day"
}

// IsWholeDisplayUnit reports whether b represents a whole number of display
// units (no fractional part) — e.g. in day mode, a multiple of HoursPerDay.
func (p *Project) IsWholeDisplayUnit(b Base) bool {
	return int64(b)%p.subUnitsPerDisplay() == 0
}

// BaseFromWholeDisplay converts a whole count of display units to Base
// (e.g. 40 days -> 40*HoursPerDay hours). Used for level-1 entry (R-4.1).
func (p *Project) BaseFromWholeDisplay(displayUnits int64) Base {
	return Base(displayUnits * p.subUnitsPerDisplay())
}

// Format renders a Base value as a decimal number of DISPLAY units, using a
// COMMA decimal separator and up to two decimals, via integer math only
// (requirement R-4.3). Examples:
//
//	day mode, HoursPerDay=8:  10 (hours) -> "1,25" (days)
//	hour mode:               90 (minutes) -> "1,5" (hours)
func (p *Project) Format(b Base) string {
	return formatPer(b, p.subUnitsPerDisplay())
}

// FormatHours renders a Base (minutes) value in HOURS ("52,5"), independent of
// the project's display unit.
func (p *Project) FormatHours(b Base) string { return formatPer(b, 60) }

// FormatDays renders a Base (minutes) value in working DAYS of the project's
// hours per day ("6,56"), independent of the project's display unit.
func (p *Project) FormatDays(b Base) string { return formatPer(b, 60*int64(p.hpd())) }

// formatPer renders b as a decimal number of units of `per` minutes: comma
// separator, up to two decimals, exact integer math.
func formatPer(b Base, per int64) string {
	neg := b < 0
	v := int64(b)
	if neg {
		v = -v
	}
	whole := v / per
	rem := v % per // remaining sub-units
	// Convert the remainder to hundredths of a display unit with rounding:
	// frac2 = round(rem*100/per), exact integer arithmetic.
	frac2 := (rem*100 + per/2) / per
	if frac2 >= 100 { // rounding spilled into the whole part
		whole++
		frac2 -= 100
	}
	var s string
	switch {
	case frac2 == 0:
		s = fmt.Sprintf("%d", whole)
	case frac2%10 == 0:
		s = fmt.Sprintf("%d,%d", whole, frac2/10)
	default:
		s = fmt.Sprintf("%d,%02d", whole, frac2)
	}
	if neg {
		s = "-" + s
	}
	return s
}

// Parse converts a decimal string of DISPLAY units (comma or dot, up to two
// decimals) into a stored Base value, snapping to the nearest whole base unit
// via integer math only. Examples:
//
//	day mode, HoursPerDay=8:  "1,25" -> 10 (hours);  "1,20" -> 10 (nearest hour)
//	hour mode:                "1,5"  -> 90 (minutes)
func (p *Project) Parse(s string) (Base, error) {
	per := p.subUnitsPerDisplay()
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	}
	s = strings.Replace(s, ",", ".", 1)
	intPart, fracPart, _ := strings.Cut(s, ".")
	if intPart == "" {
		intPart = "0"
	}
	var whole int64
	for _, r := range intPart {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("invalid number %q", s)
		}
		whole = whole*10 + int64(r-'0')
	}
	// fractional hundredths of a display unit (two digits, padded/truncated)
	var frac2 int64
	for i := 0; i < 2; i++ {
		var d byte = '0'
		if i < len(fracPart) {
			d = fracPart[i]
			if d < '0' || d > '9' {
				return 0, fmt.Errorf("invalid number %q", s)
			}
		}
		frac2 = frac2*10 + int64(d-'0')
	}
	// base = whole*per + round(frac2/100 * per)
	base := whole*per + (frac2*per+50)/100
	if neg {
		base = -base
	}
	return Base(base), nil
}
