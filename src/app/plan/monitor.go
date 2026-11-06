package plan

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"
)

// Monitoring: comparing the plan with real effort booked in an external time
// tracking system. This file holds the pure (Wails- and network-free) parts:
// the per-project settings, the action External ID rules, and the matching and
// aggregation of fetched bookings.

// MaxExternalIDLen is the maximum length (in characters) of an action's
// External ID.
const MaxExternalIDLen = 16

// External system identifiers.
const (
	SystemAerion = "aerion"
	SystemCSV    = "csv" // listed in the UI but not implemented yet
)

// IsSupportedSystem reports whether a system type can be used for monitoring.
// Only Aerion is implemented so far.
func IsSupportedSystem(s string) bool { return s == SystemAerion }

// Monitor is the per-project monitoring configuration. Credentials are NOT part
// of the project: they live in the user's ~/.pt-pm/config.yaml.
type Monitor struct {
	Enabled bool `json:"enabled"`
	// System is the external system type (see SystemAerion).
	System string `json:"system,omitempty"`
	// ProjectID is the project's identifier inside the external system.
	ProjectID string `json:"projectId,omitempty"`
	// DefaultAction is the ID of the action (work package) that receives all
	// bookings matching no action's External ID. Empty = none: such bookings are
	// only reported as unassigned.
	DefaultAction string `json:"defaultAction,omitempty"`
	// IncludeBefore / IncludeAfter widen the synchronized period by one month
	// before the project start / after the project end (bookings often start
	// early or run over).
	IncludeBefore bool `json:"includeBefore,omitempty"`
	IncludeAfter  bool `json:"includeAfter,omitempty"`
	// SyncedAt is when the real effort of the synchronized actions was last
	// taken from the external system (display text, e.g. "2026-11-04 16:11").
	SyncedAt string `json:"syncedAt,omitempty"`
}

// AddMonthsClamped adds n months to t and keeps the day of month, clamped to
// the last day of the target month (31 Mar - 1 month = 28 Feb, not 3 Mar).
func AddMonthsClamped(t time.Time, n int) time.Time {
	y, mo, d := t.Date()
	first := time.Date(y, mo+time.Month(n), 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
	last := time.Date(first.Year(), first.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if d > last {
		d = last
	}
	return time.Date(first.Year(), first.Month(), d, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}

// SyncRange returns the period to synchronize for a project running from start
// to end: the project period, extended by one month before and/or after when
// the options are set.
func (m Monitor) SyncRange(start, end time.Time) (from, to time.Time) {
	from, to = start, end
	if m.IncludeBefore {
		from = AddMonthsClamped(start, -1)
	}
	if m.IncludeAfter {
		to = AddMonthsClamped(end, 1)
	}
	return from, to
}

// Validate returns an error when monitoring is enabled without a supported
// system type and an external project id (both are mandatory).
func (m Monitor) Validate() error {
	if !m.Enabled {
		return nil
	}
	if strings.TrimSpace(m.System) == "" {
		return fmt.Errorf("monitoring needs an external system type")
	}
	if !IsSupportedSystem(m.System) {
		return fmt.Errorf("system %q is not supported yet", m.System)
	}
	if strings.TrimSpace(m.ProjectID) == "" {
		return fmt.Errorf("monitoring needs the project ID in the external system")
	}
	return nil
}

// NormalizeExternalID trims an External ID and validates it: at most
// MaxExternalIDLen characters and no control characters or line breaks. The
// empty string is valid (no external id).
func NormalizeExternalID(s string) (string, error) {
	s = strings.TrimSpace(s)
	if n := len([]rune(s)); n > MaxExternalIDLen {
		return "", fmt.Errorf("external id has %d characters (maximum %d)", n, MaxExternalIDLen)
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return "", fmt.Errorf("external id must not contain control characters")
		}
	}
	return s, nil
}

// ExternalIDOwner returns the action that already uses the given External ID
// (case-insensitive), skipping the action with id exceptID. Nil if free.
func (p *Project) ExternalIDOwner(extID, exceptID string) *WorkPackage {
	if extID == "" {
		return nil
	}
	for i := range p.Packages {
		w := &p.Packages[i]
		if w.ID != exceptID && w.TracksEffort() && !IsPhasesLane(w.Cluster) && ExternalIDKey(w.ExternalID) == ExternalIDKey(extID) {
			return w
		}
	}
	return nil
}

// HasAction reports whether an action with this ID exists.
func (p *Project) HasAction(id string) bool { return id != "" && p.WorkPackage(id) != nil }

// Booking is one single-line time booking fetched from an external system.
type Booking struct {
	ID      int64  // id in the external system
	Day     string // YYYY-MM-DD
	Seconds int    // tracked duration in seconds
	Text    string // the booking text (where the work package id is searched)
}

func isIDChar(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }

// MatchExternalIDs returns the indexes (into ids) of every External ID that is
// referenced in text, in the order of their first appearance in the text. An id
// matches case-insensitively, with optional hyphens (WP1 = wp1 = WP-1 = wp-1,
// but not "WP 1"), and only as a whole token: the characters directly before
// and after it must not be letters or digits, so "WP-1" does not match inside
// "WP-12". An occurrence that lies completely inside a longer matching
// id (for example "1" inside "Phase 1") is not a reference of its own. Each id
// is returned at most once; empty ids never match.
func MatchExternalIDs(text string, ids []string) []int {
	hay := []rune(strings.ToLower(text))
	type occ struct{ idx, pos, end int }
	var occs []occ
	for idx, id := range ids {
		needle := idMatchRunes(id)
		n := len(needle)
		if n == 0 || n > len(hay) {
			continue
		}
		for pos := 0; pos < len(hay); pos++ {
			end, ok := matchAt(hay, pos, needle)
			if !ok {
				continue
			}
			if pos > 0 && isIDChar(hay[pos-1]) && isIDChar(needle[0]) {
				continue
			}
			if end < len(hay) && isIDChar(hay[end]) && isIDChar(needle[n-1]) {
				continue
			}
			occs = append(occs, occ{idx, pos, end})
		}
	}
	// drop occurrences inside a longer occurrence of another id
	var kept []occ
	for _, o := range occs {
		inside := false
		for _, q := range occs {
			if q.idx != o.idx && q.pos <= o.pos && o.end <= q.end && q.end-q.pos > o.end-o.pos {
				inside = true
				break
			}
		}
		if !inside {
			kept = append(kept, o)
		}
	}
	// first appearance per id, ordered by position
	first := map[int]int{}
	for _, o := range kept {
		if p, ok := first[o.idx]; !ok || o.pos < p {
			first[o.idx] = o.pos
		}
	}
	var out []int
	for idx := range first {
		out = append(out, idx)
	}
	sort.Slice(out, func(a, b int) bool {
		if first[out[a]] != first[out[b]] {
			return first[out[a]] < first[out[b]]
		}
		return out[a] < out[b]
	})
	return out
}

// idMatchRunes is the form of an External ID used for matching: lower case,
// without hyphens. Hyphens inside an ID are optional, so WP1 and WP-1 are the
// same ID (a space is not: "WP 1" is two words).
func idMatchRunes(id string) []rune {
	var out []rune
	for _, r := range strings.ToLower(id) {
		if r != '-' {
			out = append(out, r)
		}
	}
	return out
}

// ExternalIDKey is the comparison key of an External ID: case and hyphens do
// not matter. Two IDs with the same key would match the same booking texts.
func ExternalIDKey(id string) string { return string(idMatchRunes(strings.TrimSpace(id))) }

// matchAt matches needle (lower case, no hyphens) in hay at pos, allowing
// hyphens between the characters of the needle. It returns the end position.
func matchAt(hay []rune, pos int, needle []rune) (int, bool) {
	i := pos
	for k, c := range needle {
		if k > 0 {
			for i < len(hay) && hay[i] == '-' {
				i++
			}
		}
		if i >= len(hay) || hay[i] != c {
			return 0, false
		}
		i++
	}
	return i, true
}

// SpreadSeconds splits seconds evenly over n shares. The remainder is handed out
// one second at a time to the first shares, so the shares always add up to
// exactly the input (no time is lost or invented).
func SpreadSeconds(seconds, n int) []int64 {
	if n <= 0 {
		return nil
	}
	out := make([]int64, n)
	base, rem := int64(seconds)/int64(n), int64(seconds)%int64(n)
	for i := range out {
		out[i] = base
		if int64(i) < rem {
			out[i]++
		}
	}
	return out
}

// ActionActual is the booked effort of one tracked action: one with an
// External ID and/or the monitor's default action.
type ActionActual struct {
	ID         string
	ExternalID string // may be empty for the default action
	Name       string
	Est        Base  // the action's estimated effort
	Seconds    int64 // MatchedSeconds plus DefaultSeconds
	Entries    int   // MatchedEntries plus DefaultEntries
	// MatchedSeconds/MatchedEntries are the bookings that reference this
	// action's External ID (a booking naming several actions is spread evenly:
	// each gets its share of the seconds and counts the booking once).
	MatchedSeconds int64
	MatchedEntries int
	// IsDefault marks the default action; DefaultSeconds/DefaultEntries are the
	// unassigned bookings credited to it (already included in Seconds/Entries).
	IsDefault      bool
	DefaultSeconds int64
	DefaultEntries int
}

// SectionSummary is one section of the monitoring table.
type SectionSummary struct {
	Name       string
	Actions    int   // number of actions in the section
	Tracked    int   // actions with an External ID, plus the default action
	Est        Base  // estimated effort of all actions
	EstTracked Base  // estimated effort of the tracked actions only
	Seconds    int64 // booked effort of the tracked actions
	Entries    int
	Details    []ActionActual // tracked actions only
}

// MonitorSummary is the result of matching fetched bookings to the plan.
type MonitorSummary struct {
	Sections         []SectionSummary
	TotalEst         Base
	TotalEstTracked  Base
	TotalSeconds     int64
	TotalEntries     int
	UnmatchedSeconds int64 // bookings assigned to no action at all
	UnmatchedEntries int
	DefaultAction    string // ID of the action that received unassigned bookings ("" = none)
	DefaultSeconds   int64
	DefaultEntries   int
	Fetched          int
}

// SecondsToBase converts booked seconds to the minute-based Base, rounding to
// the nearest minute. Sums are converted once at the end so rounding errors do
// not accumulate.
func SecondsToBase(sec int64) Base { return Base((sec + 30) / 60) }

// Remaining returns the effort left of an estimation after the booked seconds
// (negative when over budget) and that as a whole percent of the estimation.
// ok is false when there is no estimation to compare with (est <= 0).
func Remaining(est Base, seconds int64) (rem Base, pct int, ok bool) {
	rem = est - SecondsToBase(seconds)
	if est <= 0 {
		return rem, 0, false
	}
	n := int64(rem) * 100
	if n >= 0 {
		return rem, int((n + int64(est)/2) / int64(est)), true
	}
	return rem, -int((-n + int64(est)/2) / int64(est)), true
}

// MonitorSummary matches bookings to the project's actions by External ID and
// aggregates per section and for the whole project. Bookings that match no
// action go to the action defaultAction when it is set (they are then counted
// as that action's effort and flagged as DefaultSeconds); otherwise they are
// reported as unmatched. The default action counts as tracked even without an
// External ID, so its estimate is compared with what it received.
func (p *Project) MonitorSummary(bookings []Booking, defaultAction string) MonitorSummary {
	var ids []string
	var owner []int // ids index -> p.Packages index
	defIdx := -1
	for i := range p.Packages {
		if !p.Packages[i].TracksEffort() || IsPhasesLane(p.Packages[i].Cluster) {
			continue // contour None / Phases lane (a view): takes no bookings, neither by External ID nor as default
		}
		if p.Packages[i].ExternalID != "" {
			ids = append(ids, p.Packages[i].ExternalID)
			owner = append(owner, i)
		}
		if defaultAction != "" && p.Packages[i].ID == defaultAction {
			defIdx = i
		}
	}
	secs := make([]int64, len(p.Packages))
	cnt := make([]int, len(p.Packages))
	sum := MonitorSummary{Fetched: len(bookings)}
	for _, b := range bookings {
		if refs := MatchExternalIDs(b.Text, ids); len(refs) > 0 {
			// several references: the time is spread evenly over the actions
			for n, share := range SpreadSeconds(b.Seconds, len(refs)) {
				secs[owner[refs[n]]] += share
				cnt[owner[refs[n]]]++
			}
			sum.TotalSeconds += int64(b.Seconds)
			sum.TotalEntries++
		} else {
			sum.UnmatchedSeconds += int64(b.Seconds)
			sum.UnmatchedEntries++
		}
	}
	matchedSecs := append([]int64(nil), secs...) // before the default share is added
	matchedCnt := append([]int(nil), cnt...)
	var defSecs int64
	var defCnt int
	if defIdx >= 0 {
		defSecs, defCnt = sum.UnmatchedSeconds, sum.UnmatchedEntries
		secs[defIdx] += defSecs
		cnt[defIdx] += defCnt
		sum.DefaultAction = p.Packages[defIdx].ID
		sum.DefaultSeconds, sum.DefaultEntries = defSecs, defCnt
		sum.TotalSeconds += defSecs
		sum.TotalEntries += defCnt
		sum.UnmatchedSeconds, sum.UnmatchedEntries = 0, 0
	}
	idx := map[string]int{}
	for i := range p.Packages {
		w := &p.Packages[i]
		if IsPhasesLane(w.Cluster) {
			continue // the Phases lane is a view: not part of monitoring
		}
		si, ok := idx[w.Cluster]
		if !ok {
			si = len(sum.Sections)
			idx[w.Cluster] = si
			sum.Sections = append(sum.Sections, SectionSummary{Name: w.Cluster})
		}
		s := &sum.Sections[si]
		s.Actions++
		est := w.EffectiveEst()
		s.Est += est
		sum.TotalEst += est
		if (w.ExternalID != "" && w.TracksEffort()) || i == defIdx {
			s.Tracked++
			s.EstTracked += est
			sum.TotalEstTracked += est
			s.Seconds += secs[i]
			s.Entries += cnt[i]
			a := ActionActual{ID: w.ID, ExternalID: w.ExternalID, Name: w.Name, Est: est, Seconds: secs[i], Entries: cnt[i],
				MatchedSeconds: matchedSecs[i], MatchedEntries: matchedCnt[i]}
			if i == defIdx {
				a.IsDefault, a.DefaultSeconds, a.DefaultEntries = true, defSecs, defCnt
			}
			s.Details = append(s.Details, a)
		}
	}
	return sum
}

// IsSynced reports whether the action's real effort comes from the external
// system: monitoring is on and the action either has an External ID or is the
// default action. The real effort of such an action is read-only; every
// other action keeps its manually entered real effort.
func (p *Project) IsSynced(wp *WorkPackage) bool {
	if wp == nil || !p.Monitor.Enabled || !wp.TracksEffort() || IsPhasesLane(wp.Cluster) {
		return false // contour None: no real effort, nothing to synchronize
	}
	return wp.ExternalID != "" || (p.Monitor.DefaultAction != "" && wp.ID == p.Monitor.DefaultAction)
}

// ApplySyncedEffort stores the booked effort of a synchronization as the real
// effort of every synchronized action (see IsSynced) and returns how many
// actions were updated. Other actions are left untouched.
func (p *Project) ApplySyncedEffort(s MonitorSummary) int {
	n := 0
	for _, sec := range s.Sections {
		for _, d := range sec.Details {
			if wp := p.WorkPackage(d.ID); wp != nil && p.IsSynced(wp) {
				wp.RealEst = SecondsToBase(d.Seconds)
				n++
			}
		}
	}
	return n
}

// DropExternalIDsOfNone removes the External ID of every action with the
// contour None, which cannot carry one, and returns how many were removed. It
// repairs files written before that rule (such an ID never had an effect: the
// action takes no bookings).
func (p *Project) DropExternalIDsOfNone() int {
	n := 0
	for i := range p.Packages {
		if !p.Packages[i].TracksEffort() && p.Packages[i].ExternalID != "" {
			p.Packages[i].ExternalID = ""
			n++
		}
	}
	return n
}
