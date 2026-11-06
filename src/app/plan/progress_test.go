package plan

import (
	"math"
	"testing"

	"pgregory.net/rapid"
)

const eps = 1e-9

var allContours = []Contour{ContourFlat, ContourFront, ContourBack, ContourDouble}

func genContour() *rapid.Generator[Contour] { return rapid.SampledFrom(allContours) }

// genEffort draws efforts across negative, zero, in-range and above-max values.
func genEffort() *rapid.Generator[Base] {
	return rapid.Custom(func(t *rapid.T) Base {
		return Base(rapid.Int64Range(-10_000_000, 10_000_000).Draw(t, "effort"))
	})
}

// Feature: progress, Property 11: Normalized contour is always one of the four kinds
func TestProp_NormalizeContour(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := rapid.OneOf(rapid.String(), rapid.SampledFrom([]string{
			"Flat", "Front-Loaded", "Back-Loaded", "Double-Peak", "None", "", "flat", "none", "Flat ",
		})).Draw(t, "s")
		got := NormalizeContour(s)
		valid := false
		for _, c := range append(append([]Contour{}, allContours...), ContourNone) {
			if got == c {
				valid = true
			}
		}
		if !valid {
			t.Fatalf("NormalizeContour(%q) = %q, not one of the four kinds", s, got)
		}
		if IsValidContour(s) {
			if string(got) != s {
				t.Fatalf("valid %q normalized to %q", s, got)
			}
		} else if got != ContourFlat {
			t.Fatalf("invalid %q normalized to %q, want Flat", s, got)
		}
	})
}

// Feature: progress, Property 4: Progress percentage is always an integer within 0 through 100
func TestProp_ProgressBounds(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		w := WorkPackage{
			Est:     Base(rapid.Int64Range(-1_000_000_000_000, 1_000_000_000_000).Draw(t, "est")),
			RealEst: Base(rapid.Int64Range(-1_000_000_000_000, 1_000_000_000_000).Draw(t, "real")),
		}
		if p := w.ProgressPercent(); p < 0 || p > 100 {
			t.Fatalf("ProgressPercent(est=%d, real=%d) = %d out of 0..100", w.Est, w.RealEst, p)
		}
	})
}

// Feature: progress, Property 5: Progress percentage matches its defining formula and boundaries
func TestProp_ProgressFormula(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		e := genEffort().Draw(t, "est")
		r := genEffort().Draw(t, "real")
		w := WorkPackage{Est: e, RealEst: r}
		got := w.ProgressPercent()
		cr := r
		if cr < 0 {
			cr = 0
		}
		switch {
		case e <= 0:
			if got != 0 {
				t.Fatalf("est=%d: got %d, want 0", e, got)
			}
		case cr >= e:
			if got != 100 {
				t.Fatalf("est=%d real=%d: got %d, want 100", e, r, got)
			}
		default:
			if want := int(int64(cr) * 100 / int64(e)); got != want {
				t.Fatalf("est=%d real=%d: got %d, want %d", e, r, got, want)
			}
		}
		if r < 0 {
			z := WorkPackage{Est: e, RealEst: 0}
			if got != z.ProgressPercent() {
				t.Fatalf("negative real must equal real=0 result")
			}
		}
	})
}

// Feature: progress, Property 6: Completion position honours start/end boundaries for every contour
func TestProp_CompletionBoundaries(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		c := genContour().Draw(t, "contour")
		e := Base(rapid.Int64Range(1, int64(MaxEffortMinutes)).Draw(t, "est"))
		zero := WorkPackage{Est: e, RealEst: 0, Contour: c}
		if got := zero.CompletionPosition(); got != 0 {
			t.Fatalf("real=0 contour=%s: got %v, want 0", c, got)
		}
		over := WorkPackage{Est: e, RealEst: e + Base(rapid.Int64Range(0, 1_000_000).Draw(t, "extra")), Contour: c}
		if got := over.CompletionPosition(); got != 1 {
			t.Fatalf("real>=est contour=%s: got %v, want 1", c, got)
		}
	})
}

// Feature: progress, Property 7: Completion position advances monotonically with real effort
func TestProp_CompletionMonotonic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		c := genContour().Draw(t, "contour")
		e := Base(rapid.Int64Range(2, int64(MaxEffortMinutes)).Draw(t, "est"))
		r1 := Base(rapid.Int64Range(0, int64(e)-1).Draw(t, "r1"))
		r2 := Base(rapid.Int64Range(int64(r1)+1, int64(e)).Draw(t, "r2"))
		f1 := (&WorkPackage{Est: e, RealEst: r1, Contour: c}).CompletionPosition()
		f2 := (&WorkPackage{Est: e, RealEst: r2, Contour: c}).CompletionPosition()
		if f1 > f2+eps {
			t.Fatalf("contour=%s est=%d: pos(%d)=%v > pos(%d)=%v", c, e, r1, f1, r2, f2)
		}
	})
}

// Feature: progress, Property 8: Flat contour places position at the exact effort ratio
func TestProp_FlatExactRatio(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		e := Base(rapid.Int64Range(1, int64(MaxEffortMinutes)).Draw(t, "est"))
		r := Base(rapid.Int64Range(0, int64(e)-1).Draw(t, "real"))
		got := (&WorkPackage{Est: e, RealEst: r, Contour: ContourFlat}).CompletionPosition()
		if want := float64(r) / float64(e); math.Abs(got-want) > eps {
			t.Fatalf("flat est=%d real=%d: got %v want %v", e, r, got, want)
		}
	})
}

// Feature: progress, Property 9: effort is the area under the curve. A
// front-loaded curve needs much effort before the marker gets far, so (once past
// its initial ramp) the marker is BEHIND the linear position; a back-loaded
// curve is the opposite (until its final ramp).
func TestProp_FrontBackOrdering(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		e := Base(rapid.Int64Range(2, int64(MaxEffortMinutes)).Draw(t, "est"))
		r := Base(rapid.Int64Range(1, int64(e)-1).Draw(t, "real"))
		tr := float64(r) / float64(e)
		front := (&WorkPackage{Est: e, RealEst: r, Contour: ContourFront}).CompletionPosition()
		back := (&WorkPackage{Est: e, RealEst: r, Contour: ContourBack}).CompletionPosition()
		if tr >= 0.06 && front > tr+eps {
			t.Fatalf("front-loaded marker %v is ahead of the effort ratio %v", front, tr)
		}
		if tr <= 0.94 && back < tr-eps {
			t.Fatalf("back-loaded marker %v is behind the effort ratio %v", back, tr)
		}
		if tr >= 0.06 && tr <= 0.94 && front > back+eps {
			t.Fatalf("front %v must not be ahead of back %v", front, back)
		}
	})
}

// Feature: progress, Property 10: the double-peak curve holds most of its area
// at both ends, so the marker moves SLOWLY per unit of effort near the start and
// the end and quickly in the middle.
func TestProp_DoublePeakEndsSlower(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		const e = Base(1_000_000)
		d := Base(2 * rapid.Int64Range(500, 75_000).Draw(t, "halfStep"))
		pos := func(r Base) float64 {
			return (&WorkPackage{Est: e, RealEst: r, Contour: ContourDouble}).CompletionPosition()
		}
		start := pos(d) - pos(0)
		end := pos(e) - pos(e-d)
		mid := pos(e/2+d/2) - pos(e/2-d/2)
		if !(start < mid) || !(end < mid) {
			t.Fatalf("d=%d start=%v end=%v mid=%v: ends must advance slower than the middle", d, start, end, mid)
		}
	})
}

// Feature: progress - the marker position is the exact inverse of the area under
// the curve: the area up to the marker is the effort share, for every contour.
func TestProp_PositionIsInverseOfAreaShare(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		c := genContour().Draw(t, "contour")
		share := rapid.Float64Range(0.0005, 0.9995).Draw(t, "share")
		pos := ContourPositionForShare(c, share)
		if pos < 0 || pos > 1 {
			t.Fatalf("position out of range: %v", pos)
		}
		if got := ContourAreaShare(c, pos); math.Abs(got-share) > 1e-9 {
			t.Fatalf("%s: area up to the marker %v, want %v", c, got, share)
		}
	})
}

// Reference values computed independently (exact piecewise integration in Python):
// the position on the bar reached for an effort share t.
func TestContourPositionReferenceValues(t *testing.T) {
	ref := map[Contour]map[float64]float64{
		ContourFlat:   {0.1: 0.1, 0.25: 0.25, 0.5: 0.5, 0.75: 0.75, 0.9: 0.9},
		ContourFront:  {0.1: 0.0859, 0.25: 0.1740, 0.5: 0.3417, 0.75: 0.5592, 0.9: 0.7493},
		ContourBack:   {0.1: 0.2507, 0.25: 0.4408, 0.5: 0.6583, 0.75: 0.8260, 0.9: 0.9141},
		ContourDouble: {0.1: 0.0875, 0.25: 0.1827, 0.5: 0.5, 0.75: 0.8173, 0.9: 0.9125},
	}
	for c, m := range ref {
		for share, want := range m {
			if got := ContourPositionForShare(c, share); math.Abs(got-want) > 2e-4 {
				t.Errorf("%s share %.2f: position %.5f, want %.4f", c, share, got, want)
			}
		}
	}
}

func TestNewActionContourDefaultIsZeroRealEst(t *testing.T) {
	w := WorkPackage{}
	if w.RealEst != 0 {
		t.Fatalf("zero-value RealEst = %d", w.RealEst)
	}
	if NormalizeContour(string(w.Contour)) != ContourFlat {
		t.Fatalf("zero-value contour must normalize to Flat")
	}
}

func TestProgressPermille(t *testing.T) {
	for _, c := range []struct {
		est, real Base
		want      int
	}{
		{19200, 3150, 164}, // 52,5 h of 320 h = 16,4 %
		{19200, 0, 0},
		{19200, -5, 0},
		{19200, 19200, 1000},
		{19200, 40000, 1000},
		{0, 100, 0},
		{3, 1, 333},
	} {
		w := WorkPackage{Est: c.est, RealEst: c.real}
		if got := w.ProgressPermille(); got != c.want {
			t.Errorf("est=%d real=%d: %d, want %d", c.est, c.real, got, c.want)
		}
	}
}

// Reference values computed independently (numeric integration in Python).
func TestContourAreaShareReferenceValues(t *testing.T) {
	ref := map[Contour]map[float64]float64{
		ContourFlat:   {0.1: 0.1, 0.25: 0.25, 0.5: 0.5, 0.75: 0.75, 0.9: 0.9},
		ContourFront:  {0.1: 0.1239, 0.25: 0.3695, 0.5: 0.6903, 0.75: 0.9004, 0.9: 0.9735},
		ContourBack:   {0.1: 0.0265, 0.25: 0.0996, 0.5: 0.3097, 0.75: 0.6305, 0.9: 0.8761},
		ContourDouble: {0.1: 0.1207, 0.25: 0.3357, 0.5: 0.5, 0.75: 0.6643, 0.9: 0.8793},
	}
	for c, m := range ref {
		for x, want := range m {
			if got := ContourAreaShare(c, x); math.Abs(got-want) > 1e-4 {
				t.Errorf("%s at %.2f: %.5f, want %.4f", c, x, got, want)
			}
		}
	}
	if ContourAreaShare(ContourFront, 0) != 0 || ContourAreaShare(ContourFront, 1) != 1 {
		t.Error("area share must run from 0 to 1")
	}
	if got := ContourAreaShare("bogus", 0.3); math.Abs(got-0.3) > 1e-12 {
		t.Errorf("unknown contour behaves like Flat: %v", got)
	}
}

// The case of test-assignment.ptplan: 52,5 h of 320 h planned (16,4 %). The
// contour rating is how far along the planned curve that effort gets.
func TestContourPermilleForTheRealCase(t *testing.T) {
	for c, want := range map[Contour]int{ContourFlat: 164, ContourFront: 123, ContourBack: 342, ContourDouble: 126} {
		w := WorkPackage{Est: 19200, RealEst: 3150, Contour: c}
		if got := w.ContourPermille(); got < want-1 || got > want+1 {
			t.Errorf("%s: contour rating %d, want %d", c, got, want)
		}
		if w.ProgressPermille() != 164 {
			t.Errorf("pure rating must not depend on the contour")
		}
	}
}

// Feature: ratings - the contour rating is within 0..1000, is 0 without real
// effort and 1000 once the plan is consumed, grows with real effort for every
// contour, and for Flat agrees with the pure rating within rounding.
func TestProp_ContourPermille(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		c := genContour().Draw(t, "contour")
		e := Base(rapid.Int64Range(2, int64(MaxEffortMinutes)).Draw(t, "est"))
		r1 := Base(rapid.Int64Range(0, int64(e)).Draw(t, "r1"))
		r2 := Base(rapid.Int64Range(int64(r1), int64(e)).Draw(t, "r2"))
		w1 := WorkPackage{Est: e, RealEst: r1, Contour: c}
		w2 := WorkPackage{Est: e, RealEst: r2, Contour: c}
		p1, p2 := w1.ContourPermille(), w2.ContourPermille()
		if p1 < 0 || p1 > 1000 || p2 < 0 || p2 > 1000 {
			t.Fatalf("out of range: %d %d", p1, p2)
		}
		if p1 > p2 {
			t.Fatalf("not monotonic: real %d -> %d, real %d -> %d", r1, p1, r2, p2)
		}
		if r1 == 0 && p1 != 0 {
			t.Fatalf("no real effort must rate 0, got %d", p1)
		}
		if r2 == e && p2 != 1000 {
			t.Fatalf("plan consumed must rate 1000, got %d", p2)
		}
		pure := w1.ProgressPermille()
		if c == ContourFlat && (pure-p1 < -1 || pure-p1 > 1) {
			t.Fatalf("flat must equal the pure rating: pure %d contour %d", pure, p1)
		}
		if c == ContourFront && pure >= 80 && p1 > pure+1 || c == ContourBack && pure <= 920 && p1 < pure-1 {
			t.Fatalf("%s: pure %d contour %d", c, pure, p1)
		}
	})
}

// Consumed is derived from completed (real) effort, so it is non-zero once an
// action has real effort. Regression for the report showing "0" everywhere.
func TestAssignmentConsumedFromRealEffort(t *testing.T) {
	p := NewProject("t", UnitDays)
	// Project Management: planned 40 d (320 h = 19200 min), real 15 h (900 min).
	p.Packages = append(p.Packages, WorkPackage{ID: "pm", Name: "Project Management", Est: 19200, RealEst: 900, Contour: ContourFlat, Mode: ModeFixed})
	if got, want := p.AssignmentConsumed(Assignment{WorkPackageID: "pm", Percent: 100}), p.Format(900)+" (4.6%)"; got != want {
		t.Errorf("100%% assignment: %q, want %q", got, want)
	}
	// a 50 % assignment consumes half of the days; the action's rating is unchanged
	if got, want := p.AssignmentConsumed(Assignment{WorkPackageID: "pm", Percent: 50}), p.Format(450)+" (4.6%)"; got != want {
		t.Errorf("50%% assignment: %q, want %q", got, want)
	}
	if got := p.AssignmentConsumed(Assignment{WorkPackageID: "gone", Percent: 50}); got != p.Format(0)+" (0.0%)" {
		t.Errorf("unknown action: %q", got)
	}
}
