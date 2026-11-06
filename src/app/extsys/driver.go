package extsys

import (
	"context"
	"fmt"
	"time"

	"github.com/megaproaktiv/g-plan/src/app/plan"
)

// Progress reports how many bookings have been fetched so far and the total
// the system announced.
type Progress func(fetched, total int)

// ProjectInfo describes a project in the external system (used to verify the
// configured project id).
type ProjectInfo struct {
	ID        string
	Name      string
	Status    int
	StartDate string
	EndDate   string
}

// Driver fetches data from one external system.
type Driver interface {
	// Check verifies credentials and that the project exists. Read-only.
	Check(ctx context.Context, projectID string) (ProjectInfo, error)
	// Fetch returns every single-line booking of the project between from and
	// to (inclusive). Read-only.
	Fetch(ctx context.Context, projectID string, from, to time.Time, onProgress Progress) ([]plan.Booking, error)
}

// SystemInfo describes a selectable system type in the UI.
type SystemInfo struct {
	ID        string
	Label     string
	Available bool
}

// Systems lists the known system types. CSV is shown but not yet implemented.
func Systems() []SystemInfo {
	return []SystemInfo{
		{ID: plan.SystemAerion, Label: "Aerion", Available: true},
		{ID: plan.SystemCSV, Label: "CSV (not yet implemented)", Available: false},
	}
}

// NewDriver creates the driver for a system type using the credentials in the
// config file at cfgPath.
func NewDriver(system, cfgPath string) (Driver, error) {
	switch system {
	case plan.SystemAerion:
		cfg, err := LoadAerion(cfgPath)
		if err != nil {
			return nil, err
		}
		if err := cfg.Complete(); err != nil {
			return nil, err
		}
		return NewAerion(cfg), nil
	case plan.SystemCSV:
		return nil, fmt.Errorf("the CSV system is not implemented yet")
	default:
		return nil, fmt.Errorf("unknown system %q", system)
	}
}
