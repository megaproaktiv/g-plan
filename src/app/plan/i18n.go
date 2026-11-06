package plan

import (
	"fmt"
	"strings"
	"time"
)

// Internal language/format configuration. The GUI language is one of a small
// fixed set; each language maps to an ISO code, a human date format, and a
// number format (decimal + thousands separators). The CLI/chat language is
// always English and is NOT affected by this.

// DefaultLanguage is the GUI language a new project starts in.
const DefaultLanguage = "en"

// LangConfig describes a supported GUI language's formatting rules.
type LangConfig struct {
	Code       string `json:"code"`       // short language key, e.g. "en"
	ISO        string `json:"iso"`        // IETF/ISO locale tag, e.g. "en-US"
	Name       string `json:"name"`       // self-name shown in the pulldown
	DateFormat string `json:"dateFormat"` // Go reference layout for display dates
	Decimal    string `json:"decimal"`    // decimal separator for numbers
	Thousands  string `json:"thousands"`  // thousands separator for numbers
}

// Languages is the internal config table: the supported GUI languages and
// their ISO code + date/number formats. English uses ISO dates and a dot
// decimal; German uses D.M.Y dates and a comma decimal with dot thousands.
var Languages = []LangConfig{
	{Code: "en", ISO: "en-US", Name: "English", DateFormat: "2006-01-02", Decimal: ".", Thousands: ","},
	{Code: "de", ISO: "de-DE", Name: "Deutsch", DateFormat: "02.01.2006", Decimal: ",", Thousands: "."},
}

// LangByCode returns the config for a language code, defaulting to English for
// an unknown/empty code.
func LangByCode(code string) LangConfig {
	for _, l := range Languages {
		if l.Code == code {
			return l
		}
	}
	return Languages[0] // English
}

// NormalizeLanguage returns a supported language code, defaulting to English.
func NormalizeLanguage(code string) string {
	return LangByCode(code).Code
}

// langConfig returns the project's active language configuration.
func (p *Project) langConfig() LangConfig {
	return LangByCode(p.Language)
}

// FormatNumber renders a Base value as a decimal number of DISPLAY units in the
// project's language (decimal + thousands separators), up to two decimals, via
// integer math. This is the language-aware counterpart of Format (which always
// uses a comma). Used by the GUI; the CLI keeps Format's comma convention.
func (p *Project) FormatNumber(b Base) string {
	lc := p.langConfig()
	per := p.subUnitsPerDisplay()
	neg := b < 0
	v := int64(b)
	if neg {
		v = -v
	}
	whole := v / per
	rem := v % per
	frac2 := (rem*100 + per/2) / per
	if frac2 >= 100 {
		whole++
		frac2 -= 100
	}
	wholeStr := groupThousands(whole, lc.Thousands)
	var s string
	switch {
	case frac2 == 0:
		s = wholeStr
	case frac2%10 == 0:
		s = fmt.Sprintf("%s%s%d", wholeStr, lc.Decimal, frac2/10)
	default:
		s = fmt.Sprintf("%s%s%02d", wholeStr, lc.Decimal, frac2)
	}
	if neg {
		s = "-" + s
	}
	return s
}

// groupThousands inserts the thousands separator into a non-negative integer.
func groupThousands(n int64, sep string) string {
	s := fmt.Sprintf("%d", n)
	if sep == "" || len(s) <= 3 {
		return s
	}
	var b strings.Builder
	pre := len(s) % 3
	if pre > 0 {
		b.WriteString(s[:pre])
	}
	for i := pre; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteString(sep)
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

// FormatDate renders a time in the project's language date format. The storage
// / transport format stays ISO (DateLayout); this is for GUI display only.
func (p *Project) FormatDate(t time.Time) string {
	return t.Format(p.langConfig().DateFormat)
}

// FormatDateStr reparses an ISO (YYYY-MM-DD) date string and renders it in the
// project's language date format. Returns the input unchanged if it does not
// parse.
func (p *Project) FormatDateStr(iso string) string {
	t, err := time.Parse(DateLayout, iso)
	if err != nil {
		return iso
	}
	return p.FormatDate(t)
}
