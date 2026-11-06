package main

import (
	"testing"

	"github.com/megaproaktiv/g-plan/src/app/plan"
)

// --- Session 22: 7 dependency relationship rules + language/format config ---

// setSpan sets an action's start/end directly (bypassing clamps) for rule tests.
func depKindSpan(t *testing.T, a *App, id, start, end string) {
	t.Helper()
	if _, err := a.SetActionFields(id, "", "", "", start, end); err != nil {
		t.Fatalf("set span %s: %v", id, err)
	}
}

// TestAllRelationshipRules exercises each of the 7 kinds at a satisfied and a
// violated configuration, asserting DepViolated matches the spec:
//
//	FS: Start(B) >= End(A)
//	SS: Start(B) >= Start(A)
//	FF: End(B)   >= End(A)
//	SF: End(B)   >= Start(A)
//	MS: Start(B) >= Mid(A)
//	ME: End(B)   >= Mid(A)
//	MM: Start(A) <= Mid(B) <= End(A)
func TestAllRelationshipRules(t *testing.T) {
	// A is fixed at 2026-03-10 .. 2026-03-20 (mid = 2026-03-15).
	type tc struct {
		kind             string
		okStart, okEnd   string // B config that SATISFIES the rule
		badStart, badEnd string // B config that VIOLATES the rule
	}
	cases := []tc{
		// FS: Start(B) >= End(A)=03-20. ok: start 03-20 (touch). bad: start 03-19.
		{"FS", "2026-03-20", "2026-03-30", "2026-03-19", "2026-03-30"},
		// SS: Start(B) >= Start(A)=03-10. ok: start 03-10. bad: start 03-09.
		{"SS", "2026-03-10", "2026-03-30", "2026-03-09", "2026-03-30"},
		// FF: End(B) >= End(A)=03-20. ok: end 03-20. bad: end 03-19.
		{"FF", "2026-03-01", "2026-03-20", "2026-03-01", "2026-03-19"},
		// SF: End(B) >= Start(A)=03-10. ok: end 03-10. bad: end 03-09.
		{"SF", "2026-03-01", "2026-03-10", "2026-03-01", "2026-03-09"},
		// MS: Start(B) >= Mid(A)=03-15. ok: start 03-15. bad: start 03-14.
		{"MS", "2026-03-15", "2026-03-30", "2026-03-14", "2026-03-30"},
		// ME: End(B) >= Mid(A)=03-15. ok: end 03-15. bad: end 03-14.
		{"ME", "2026-03-01", "2026-03-15", "2026-03-01", "2026-03-14"},
		// MM: Start(A)=03-10 <= Mid(B) <= End(A)=03-20.
		//   ok: B 03-12..03-18 -> mid 03-15 (inside). bad: B 03-22..03-30 -> mid 03-26 (after End(A)).
		{"MM", "2026-03-12", "2026-03-18", "2026-03-22", "2026-03-30"},
	}
	for _, c := range cases {
		t.Run(c.kind, func(t *testing.T) {
			a := NewApp()
			if _, err := a.SetProjectRange("2026-01-01", "2026-12-31"); err != nil {
				t.Fatal(err)
			}
			A := mustWP(t, a, "A", "2026-03-10", "2026-03-20")
			B := mustWP(t, a, "B", "2026-03-10", "2026-03-20")
			if _, err := a.AddDependency(A, B, c.kind, "r", "l"); err != nil {
				t.Fatal(err)
			}
			// satisfied config
			depKindSpan(t, a, B, c.okStart, c.okEnd)
			if depViolated(a, A, B) {
				t.Fatalf("%s: B[%s..%s] should SATISFY the rule but was violated", c.kind, c.okStart, c.okEnd)
			}
			// violated config
			depKindSpan(t, a, B, c.badStart, c.badEnd)
			if !depViolated(a, A, B) {
				t.Fatalf("%s: B[%s..%s] should VIOLATE the rule but was ok", c.kind, c.badStart, c.badEnd)
			}
		})
	}
}

// TestMMBelowStartViolated covers the LOWER bound of MM (Mid(B) < Start(A)).
func TestMMBelowStartViolated(t *testing.T) {
	a := NewApp()
	if _, err := a.SetProjectRange("2026-01-01", "2026-12-31"); err != nil {
		t.Fatal(err)
	}
	A := mustWP(t, a, "A", "2026-03-10", "2026-03-20")
	B := mustWP(t, a, "B", "2026-03-10", "2026-03-20")
	if _, err := a.AddDependency(A, B, "MM", "t", "b"); err != nil {
		t.Fatal(err)
	}
	// B entirely before A: mid(B) < Start(A) -> violated
	depKindSpan(t, a, B, "2026-02-01", "2026-02-10")
	if !depViolated(a, A, B) {
		t.Fatal("MM with Mid(B) before Start(A) should be violated")
	}
}

// TestOVAliasesMM verifies the legacy "OV" kind behaves exactly like MM.
func TestOVAliasesMM(t *testing.T) {
	a := NewApp()
	if _, err := a.SetProjectRange("2026-01-01", "2026-12-31"); err != nil {
		t.Fatal(err)
	}
	A := mustWP(t, a, "A", "2026-03-10", "2026-03-20")
	B := mustWP(t, a, "B", "2026-03-12", "2026-03-18") // mid 03-15 inside
	if _, err := a.AddDependency(A, B, "OV", "t", "b"); err != nil {
		t.Fatal(err)
	}
	if depViolated(a, A, B) {
		t.Fatal("OV (=MM) with Mid(B) inside A should NOT be violated")
	}
}

// TestAllKindsAccepted verifies AddDependency accepts every documented kind and
// stores it (not silently coerced to FS).
func TestAllKindsAccepted(t *testing.T) {
	for _, k := range []string{"FS", "SS", "FF", "SF", "MS", "ME", "MM"} {
		a := NewApp()
		A := mustWP(t, a, "A", "2026-03-10", "2026-03-20")
		B := mustWP(t, a, "B", "2026-03-12", "2026-03-18")
		if _, err := a.AddDependency(A, B, k, "r", "l"); err != nil {
			t.Fatalf("kind %s rejected: %v", k, err)
		}
		d := a.GetProject()
		found := false
		for _, dep := range d.Deps {
			if dep.From == A && dep.To == B {
				found = true
				if dep.Kind != k {
					t.Fatalf("kind %s stored as %q", k, dep.Kind)
				}
			}
		}
		if !found {
			t.Fatalf("kind %s: dependency not stored", k)
		}
	}
}

// TestUnknownKindCoercedToFS verifies an unknown kind falls back to FS.
func TestUnknownKindCoercedToFS(t *testing.T) {
	a := NewApp()
	A := mustWP(t, a, "A", "2026-03-10", "2026-03-20")
	B := mustWP(t, a, "B", "2026-03-20", "2026-03-30")
	if _, err := a.AddDependency(A, B, "ZZ", "r", "l"); err != nil {
		t.Fatal(err)
	}
	d := a.GetProject()
	for _, dep := range d.Deps {
		if dep.From == A && dep.To == B && dep.Kind != "FS" {
			t.Fatalf("unknown kind should coerce to FS, got %q", dep.Kind)
		}
	}
}

// --- Language / format config ---

func TestLanguageConfigTable(t *testing.T) {
	en := plan.LangByCode("en")
	if en.ISO != "en-US" || en.Decimal != "." || en.DateFormat != "2006-01-02" {
		t.Fatalf("english config wrong: %+v", en)
	}
	de := plan.LangByCode("de")
	if de.ISO != "de-DE" || de.Decimal != "," || de.DateFormat != "02.01.2006" {
		t.Fatalf("german config wrong: %+v", de)
	}
	// unknown -> english
	if plan.LangByCode("xx").Code != "en" {
		t.Fatal("unknown language should default to english")
	}
}

func TestLanguagePersistAndDTO(t *testing.T) {
	a := NewApp()
	if a.project.Language != "en" {
		t.Fatalf("default language should be en, got %q", a.project.Language)
	}
	a.SetLanguage("de")
	if a.GetProject().Language != "de" {
		t.Fatal("SetLanguage(de) not reflected in DTO")
	}
	// round-trip
	dir := t.TempDir()
	path := dir + "/p" + plan.ProjectFileExt
	if _, err := a.SaveProject(path); err != nil {
		t.Fatal(err)
	}
	b := NewApp()
	if _, err := b.OpenProject(path); err != nil {
		t.Fatal(err)
	}
	if b.project.Language != "de" {
		t.Fatalf("language did not round-trip, got %q", b.project.Language)
	}
	// unknown language normalizes to en
	a.SetLanguage("zz")
	if a.project.Language != "en" {
		t.Fatalf("unknown language should normalize to en, got %q", a.project.Language)
	}
}

func TestLanguageNumberFormat(t *testing.T) {
	a := NewApp() // day mode, 8h/day
	// 12 hours = 1,5 days. English uses a dot, German a comma.
	v, err := a.project.Parse("1,5")
	if err != nil {
		t.Fatal(err)
	}
	a.project.Language = "en"
	if got := a.project.FormatNumber(v); got != "1.5" {
		t.Fatalf("english number format: want 1.5, got %q", got)
	}
	a.project.Language = "de"
	if got := a.project.FormatNumber(v); got != "1,5" {
		t.Fatalf("german number format: want 1,5, got %q", got)
	}
	// thousands separator: 2500 days
	big, _ := a.project.Parse("2500")
	a.project.Language = "en"
	if got := a.project.FormatNumber(big); got != "2,500" {
		t.Fatalf("english thousands: want 2,500, got %q", got)
	}
	a.project.Language = "de"
	if got := a.project.FormatNumber(big); got != "2.500" {
		t.Fatalf("german thousands: want 2.500, got %q", got)
	}
}

func TestLanguageDateFormat(t *testing.T) {
	a := NewApp()
	a.project.Language = "en"
	if got := a.project.FormatDateStr("2026-03-15"); got != "2026-03-15" {
		t.Fatalf("english date: want 2026-03-15, got %q", got)
	}
	a.project.Language = "de"
	if got := a.project.FormatDateStr("2026-03-15"); got != "15.03.2026" {
		t.Fatalf("german date: want 15.03.2026, got %q", got)
	}
}

// --- Relationship rules table (help content) ---

func TestRelationshipRulesTable(t *testing.T) {
	a := NewApp()
	en := a.RelationshipRules("en")
	if len(en) != 7 {
		t.Fatalf("expected 7 rules, got %d", len(en))
	}
	// order + canonical kinds
	wantKinds := []string{"FS", "SS", "FF", "SF", "MS", "ME", "MM"}
	for i, r := range en {
		if r.Kind != wantKinds[i] {
			t.Fatalf("rule %d kind: want %s, got %s", i, wantKinds[i], r.Kind)
		}
		if r.Abbr != wantKinds[i] { // english abbr == kind
			t.Fatalf("english abbr for %s should equal kind, got %s", r.Kind, r.Abbr)
		}
	}
	de := a.RelationshipRules("de")
	wantDeAbbr := []string{"EA", "AA", "EE", "AE", "MA", "ME", "MM"}
	for i, r := range de {
		if r.Kind != wantKinds[i] {
			t.Fatalf("de rule %d kind: want %s, got %s", i, wantKinds[i], r.Kind)
		}
		if r.Abbr != wantDeAbbr[i] {
			t.Fatalf("german abbr for %s: want %s, got %s", r.Kind, wantDeAbbr[i], r.Abbr)
		}
		if r.Name == "" || r.Example == "" || r.Rule == "" {
			t.Fatalf("german rule %s missing text: %+v", r.Kind, r)
		}
	}
}

func TestLanguagesBinding(t *testing.T) {
	a := NewApp()
	langs := a.Languages()
	if len(langs) != 2 || langs[0].Code != "en" || langs[1].Code != "de" {
		t.Fatalf("languages binding wrong: %+v", langs)
	}
}

func TestGUIStringsFallback(t *testing.T) {
	a := NewApp()
	de := a.GUIStrings("de")
	if de["help.close"] != "Schließen" {
		t.Fatalf("german help.close wrong: %q", de["help.close"])
	}
	// a key present in en must exist in de (fallback)
	if _, ok := de["links.title"]; !ok {
		t.Fatal("de bundle missing links.title")
	}
	en := a.GUIStrings("en")
	if en["help.close"] != "Close" {
		t.Fatalf("english help.close wrong: %q", en["help.close"])
	}
}
