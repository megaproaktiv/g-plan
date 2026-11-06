package plan

import "time"

// ParseDateStr parses a YYYY-MM-DD date string (DateLayout).
func ParseDateStr(s string) (time.Time, error) { return time.Parse(DateLayout, s) }

// The project has an overall START and END. These are represented as two
// milestones with the stable ids below, so they round-trip through the .mmd
// graphic source like any other milestone and show up in the Milestones
// section. The visible timeline window is anchored on them.
const (
	MilestoneStartID = "m_start"
	MilestoneEndID   = "m_end"
)

// Start returns the project start milestone date (zero time if absent).
func (p *Project) Start() (time.Time, bool) { return p.milestoneDate(MilestoneStartID) }

// End returns the project end milestone date (zero time if absent).
func (p *Project) End() (time.Time, bool) { return p.milestoneDate(MilestoneEndID) }

func (p *Project) milestoneDate(id string) (time.Time, bool) {
	for i := range p.Milestones {
		if p.Milestones[i].ID == id {
			return p.Milestones[i].Date, true
		}
	}
	return time.Time{}, false
}

// SetRange sets (creating if needed) the project Start and End milestones.
// These bound the visible window and every plan carries them.
func (p *Project) SetRange(start, end time.Time) {
	p.setMilestone(MilestoneStartID, "Start", start)
	p.setMilestone(MilestoneEndID, "End", end)
}

func (p *Project) setMilestone(id, name string, date time.Time) {
	for i := range p.Milestones {
		if p.Milestones[i].ID == id {
			p.Milestones[i].Date = date
			if p.Milestones[i].Name == "" {
				p.Milestones[i].Name = name
			}
			return
		}
	}
	// Start goes first, End goes last.
	m := Milestone{ID: id, Name: name, Date: date}
	if id == MilestoneStartID {
		p.Milestones = append([]Milestone{m}, p.Milestones...)
	} else {
		p.Milestones = append(p.Milestones, m)
	}
}

// EnsureRange guarantees a brand-new/empty project has Start/End milestones so
// the visible window is well-defined, defaulting to today .. today+90d. It only
// acts when the project has NO milestones at all; a loaded plan that already
// carries its own milestones (even custom start/end ids) is left untouched.
func (p *Project) EnsureRange() {
	if len(p.Milestones) > 0 {
		return
	}
	now := time.Now().Truncate(24 * time.Hour)
	p.setMilestone(MilestoneStartID, "Start", now)
	p.setMilestone(MilestoneEndID, "End", now.AddDate(0, 0, 90))
}
