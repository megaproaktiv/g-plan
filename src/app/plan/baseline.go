package plan

import "time"

// Baselines (Basislinien): a frozen snapshot of the schedule + effort plan at a
// point in time, for historical comparison. A baseline never changes when the
// live plan changes (requirement R-13.1).

// TaskSnapshot captures one task's schedule+effort at baseline time. ID is the
// work-package or detail id; for details the ID is the detail id.
type TaskSnapshot struct {
	ID    string    `json:"id"`
	Kind  string    `json:"kind"` // "wp" | "detail"
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Est   Base      `json:"est"`
}

// Baseline is a named, timestamped set of task snapshots.
type Baseline struct {
	Name      string         `json:"name"`
	CreatedAt time.Time      `json:"createdAt"`
	TotalEst  Base           `json:"totalEst"`
	Tasks     []TaskSnapshot `json:"tasks"`
}

// CaptureBaseline freezes the current plan into a new baseline with the given
// name (R-13.1.1/2). It captures every work package and detail.
func (p *Project) CaptureBaseline(name string) Baseline {
	b := Baseline{Name: name, CreatedAt: time.Now(), TotalEst: p.TotalEst()}
	for i := range p.Packages {
		wp := &p.Packages[i]
		b.Tasks = append(b.Tasks, TaskSnapshot{
			ID: wp.ID, Kind: "wp", Start: wp.Start, End: wp.End, Est: wp.EffectiveEst(),
		})
		for j := range wp.Details {
			d := &wp.Details[j]
			b.Tasks = append(b.Tasks, TaskSnapshot{
				ID: d.ID, Kind: "detail", Start: d.Start, End: d.End, Est: d.Est,
			})
		}
	}
	return b
}

// Variance describes the difference between the live value and a baseline
// snapshot for one task (positive end-days = slipped later; positive est =
// grew). Zero fields mean no change.
type Variance struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	StartDays int    `json:"startDays"` // live.start - base.start, in days
	EndDays   int    `json:"endDays"`   // live.end - base.end, in days
	EstDelta  Base   `json:"estDelta"`  // live.est - base.est (minutes)
	Missing   bool   `json:"missing"`   // task existed in baseline but not live
	Added     bool   `json:"added"`     // task is new since baseline
}

// CompareBaseline computes per-task variance between the live plan and a
// baseline (R-13.1.4).
func (p *Project) CompareBaseline(b *Baseline) []Variance {
	// index baseline snapshots
	snap := map[string]TaskSnapshot{}
	for _, t := range b.Tasks {
		snap[t.ID] = t
	}
	live := map[string]TaskSnapshot{}
	order := []TaskSnapshot{}
	for i := range p.Packages {
		wp := &p.Packages[i]
		s := TaskSnapshot{ID: wp.ID, Kind: "wp", Start: wp.Start, End: wp.End, Est: wp.EffectiveEst()}
		live[wp.ID] = s
		order = append(order, s)
		for j := range wp.Details {
			d := &wp.Details[j]
			ds := TaskSnapshot{ID: d.ID, Kind: "detail", Start: d.Start, End: d.End, Est: d.Est}
			live[d.ID] = ds
			order = append(order, ds)
		}
	}
	var out []Variance
	for _, s := range order {
		base, ok := snap[s.ID]
		if !ok {
			out = append(out, Variance{ID: s.ID, Kind: s.Kind, Added: true})
			continue
		}
		out = append(out, Variance{
			ID:        s.ID,
			Kind:      s.Kind,
			StartDays: daysBetween(base.Start, s.Start),
			EndDays:   daysBetween(base.End, s.End),
			EstDelta:  s.Est - base.Est,
		})
	}
	// tasks that vanished since the baseline
	for _, t := range b.Tasks {
		if _, ok := live[t.ID]; !ok {
			out = append(out, Variance{ID: t.ID, Kind: t.Kind, Missing: true})
		}
	}
	return out
}

func daysBetween(from, to time.Time) int {
	return int(to.Sub(from).Hours() / 24)
}
