package plan

import (
	"fmt"
	"math"
)

// Contour is the per-action effort-distribution shape (progress spec R4.1).
type Contour string

const (
	ContourFlat   Contour = "Flat"
	ContourFront  Contour = "Front-Loaded"
	ContourBack   Contour = "Back-Loaded"
	ContourDouble Contour = "Double-Peak"
	// ContourNone means the action has no estimation and no real effort. Its
	// effective estimation and real effort are 0 (the stored numbers are kept so
	// switching back restores them), it takes no part in booking synchronization
	// and it has no progress graphics.
	ContourNone Contour = "None"
)

// TracksEffort reports whether the action has an estimation and real effort at
// all, that is whether its contour is anything but None.
func (w *WorkPackage) TracksEffort() bool { return w.Contour != ContourNone }

// EffectiveRealEst is the real effort that counts: 0 for a contour of None.
func (w *WorkPackage) EffectiveRealEst() Base {
	if !w.TracksEffort() {
		return 0
	}
	return w.RealEst
}

// MaxEffortMinutes is the inclusive upper bound for Estimated_Effort and
// Real_Effort, in minutes (R1.1, R1.2).
const MaxEffortMinutes Base = 5_256_000

// NormalizeContour maps any input to a valid Contour; empty or unknown values
// become ContourFlat (R4.6, R8.8). This is the read/render default.
func NormalizeContour(s string) Contour {
	if IsValidContour(s) {
		return Contour(s)
	}
	return ContourFlat
}

// IsValidContour reports whether s exactly matches one of the four contour
// kinds. It is used to REJECT invalid writes (R4.3).
func IsValidContour(s string) bool {
	switch Contour(s) {
	case ContourFlat, ContourFront, ContourBack, ContourDouble, ContourNone:
		return true
	}
	return false
}

// ProgressPercent returns the integer completion percentage 0..100 (R5).
// Integer arithmetic only (R5.7).
func (w *WorkPackage) ProgressPercent() int {
	if !w.TracksEffort() {
		return 0
	}
	real := w.RealEst
	if real < 0 {
		real = 0 // R5.5
	}
	est := w.Est
	if est <= 0 {
		return 0 // R5.2
	}
	if real >= est {
		return 100 // R5.4
	}
	return int(int64(real) * 100 / int64(est)) // R5.1, truncating
}

// CompletionPosition returns the fraction 0.0..1.0 along the action's duration
// at which the progress marker sits (R6).
//
// The contour curve is the planned effort DENSITY over the duration, so the
// effort is the area under the curve. Real effort is a share t = real/planned
// of that area, and the marker sits where the area under the curve (from the
// start) equals t. A front-loaded curve holds a large part of its area at the
// start, so a lot of effort must be booked before the marker gets far: with few
// hours booked the marker is BEHIND the linear position. A back-loaded curve is
// the opposite. Flat is linear. Boundaries: no real effort -> 0.0; real effort at
// or above the planned effort -> 1.0; no planned effort -> 0.0.
func (w *WorkPackage) CompletionPosition() float64 {
	if !w.TracksEffort() {
		return 0
	}
	e := w.Est
	if e <= 0 {
		return 0
	}
	r := w.RealEst
	if r <= 0 {
		return 0 // R6.7 (also clamps negatives)
	}
	if r >= e {
		return 1 // R6.6, R6.8
	}
	return ContourPositionForShare(w.Contour, float64(r)/float64(e))
}

// ProgressPermille is the pure consumed rating in tenths of a percent (0..1000):
// real effort divided by planned effort, integer math only, truncated. 164 means
// 16,4 %. Same boundary rules as ProgressPercent.
func (w *WorkPackage) ProgressPermille() int {
	if !w.TracksEffort() {
		return 0
	}
	real := w.RealEst
	if real < 0 {
		real = 0
	}
	if w.Est <= 0 {
		return 0
	}
	if real >= w.Est {
		return 1000
	}
	return int(int64(real) * 1000 / int64(w.Est))
}

// profilePoint is a vertex of a contour's effort-density profile: position u
// along the action (0..1) and height h (higher = more planned effort there).
type profilePoint struct{ u, h float64 }

// contourProfile returns the piecewise-linear density profile of a contour.
// These are exactly the curves the Gantt draws inside the action bar
// (frontend/src/progress.ts: contourCurvePoints); keep both in sync.
func contourProfile(c Contour) []profilePoint {
	switch c {
	case ContourFront:
		return []profilePoint{{0, 0.4}, {0.1, 1}, {1, 0.1}}
	case ContourBack:
		return []profilePoint{{0, 0.1}, {0.9, 1}, {1, 0.4}}
	case ContourDouble:
		return []profilePoint{{0, 0.4}, {0.1, 1}, {0.5, 0.1}, {0.9, 1}, {1, 0.4}}
	default:
		return []profilePoint{{0, 0.5}, {1, 0.5}}
	}
}

// profileArea is the exact integral of the piecewise-linear profile from 0 to x.
func profileArea(p []profilePoint, x float64) float64 {
	var area float64
	for i := 0; i+1 < len(p); i++ {
		a, b := p[i], p[i+1]
		if x <= a.u {
			break
		}
		ux := math.Min(b.u, x)
		hx := a.h + (b.h-a.h)*(ux-a.u)/(b.u-a.u)
		area += (a.h + hx) / 2 * (ux - a.u)
	}
	return area
}

// ContourAreaShare returns the fraction (0..1) of the total area under the
// contour's curve that lies between the start and position x (also 0..1). For
// Flat it equals x.
func ContourAreaShare(c Contour, x float64) float64 {
	if x <= 0 {
		return 0
	}
	if x >= 1 {
		return 1
	}
	p := contourProfile(NormalizeContour(string(c)))
	return profileArea(p, x) / profileArea(p, 1)
}

// ContourPositionForShare is the inverse of ContourAreaShare: the position
// (0..1) along the duration at which the area under the contour's curve, from
// the start, reaches the given share (0..1) of the total area. Solved exactly:
// the cumulative area is piecewise quadratic, so each segment is a quadratic
// equation.
func ContourPositionForShare(c Contour, share float64) float64 {
	if share <= 0 {
		return 0
	}
	if share >= 1 {
		return 1
	}
	p := contourProfile(NormalizeContour(string(c)))
	need := share * profileArea(p, 1)
	for i := 0; i+1 < len(p); i++ {
		a, b := p[i], p[i+1]
		seg := (a.h + b.h) / 2 * (b.u - a.u)
		if need > seg {
			need -= seg
			continue
		}
		slope := (b.h - a.h) / (b.u - a.u)
		// area from a.u over distance d is a.h*d + slope*d*d/2 = need
		disc := math.Max(0, a.h*a.h+2*slope*need)
		d := 2 * need / (a.h + math.Sqrt(disc)) // numerically stable root
		return math.Min(1, a.u+d)
	}
	return 1
}

// ContourPermille is the contour-weighted consumed rating in tenths of a
// percent (0..1000): how far along the planned curve the real effort has got,
// that is the completion position as a percentage. With the Flat contour it
// equals ProgressPermille (up to the 0,1 % rounding). A front-loaded curve rates
// the same effort LOWER (the effort planned at the start is large), a
// back-loaded curve higher.
func (w *WorkPackage) ContourPermille() int {
	if !w.TracksEffort() || w.Est <= 0 || w.RealEst <= 0 {
		return 0
	}
	if w.RealEst >= w.Est {
		return 1000
	}
	return int(math.Round(w.CompletionPosition() * 1000))
}

// ConsumedPermille is the consumed rating of the action in tenths of a percent
// (0..1000): real effort divided by the EFFECTIVE estimation (so an action in
// derived mode counts against the sum of its details). Integer math, truncated,
// like ProgressPermille. 0 for a contour of None or without an estimation.
func (w *WorkPackage) ConsumedPermille() int {
	est := w.EffectiveEst()
	if !w.TracksEffort() || est <= 0 {
		return 0
	}
	real := w.RealEst
	if real <= 0 {
		return 0
	}
	if real >= est {
		return 1000
	}
	return int(int64(real) * 1000 / int64(est))
}

// FormatPermille renders tenths of a percent as "4.6%".
func FormatPermille(permille int) string {
	return fmt.Sprintf("%d.%d%%", permille/10, permille%10)
}

// AssignmentConsumed is what a role has consumed of one assignment: the real
// effort of the assigned action, scaled by the assignment percent, and the
// consumed rating of the action in percent, e.g. "1.88 (4.6%)" (text for the
// reports). The percent is derived from the completed (real) effort.
func (p *Project) AssignmentConsumed(a Assignment) string {
	wp := p.WorkPackage(a.WorkPackageID)
	if wp == nil {
		return p.Format(0) + " (0.0%)"
	}
	days := Base(int64(wp.EffectiveRealEst()) * int64(a.Percent) / 100)
	return fmt.Sprintf("%s (%s)", p.Format(days), FormatPermille(wp.ConsumedPermille()))
}
