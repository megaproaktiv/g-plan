package plan

import (
	"time"
)

// UnitSingular returns the singular display-unit word ("day"|"hour").
func (p *Project) UnitSingular() string { return p.unitSingular() }

// RoleLoadByMonth computes a role's planned load per calendar month across its
// assignments (requirement R-6.5). Each assignment's share
// (percent% of the work package's effective estimation, in Base units) is
// spread evenly over the inclusive calendar days of the work package and
// accumulated per month. The result maps "YYYY-MM" -> display string.
//
// Integer arithmetic: a share of S base units over N days contributes
// S*daysInMonth/N to each month (floor); any rounding remainder is dropped
// (sub-base-unit), which is acceptable for a planning view.
func (p *Project) RoleLoadByMonth(r *Role) map[string]string {
	acc := map[string]int64{} // month -> base units
	for _, a := range r.Assignments {
		wp := p.WorkPackage(a.WorkPackageID)
		if wp == nil {
			continue
		}
		share := int64(wp.EffectiveEst()) * int64(a.Percent) / 100
		if share == 0 {
			continue
		}
		days := daysInclusive(wp.Start, wp.End)
		if days <= 0 {
			continue
		}
		perDay := share / days
		rem := share % days
		cur := wp.Start
		for i := int64(0); i < days; i++ {
			key := cur.Format("2006-01")
			d := perDay
			if i < rem { // distribute the remainder across the first days
				d++
			}
			acc[key] += d
			cur = cur.AddDate(0, 0, 1)
		}
	}
	out := make(map[string]string, len(acc))
	for k, v := range acc {
		out[k] = p.Format(Base(v))
	}
	return out
}

func daysInclusive(start, end time.Time) int64 {
	if end.Before(start) {
		return 0
	}
	return int64(end.Sub(start).Hours()/24) + 1
}
