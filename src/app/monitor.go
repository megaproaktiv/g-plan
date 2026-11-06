package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/megaproaktiv/g-plan/src/app/extsys"
	"github.com/megaproaktiv/g-plan/src/app/plan"
)

// Monitoring bindings: switch monitoring on for a project, configure the
// external system, and synchronize real effort from it. Everything here is
// read-only towards the external system.

// MonitorDTO is the project's monitoring setting as shown in the Monitor page.
type MonitorDTO struct {
	Enabled       bool   `json:"enabled"`
	System        string `json:"system"`
	ProjectID     string `json:"projectId"`
	DefaultAction string `json:"defaultAction"`
	IncludeBefore bool   `json:"includeBefore"`
	IncludeAfter  bool   `json:"includeAfter"`
	// SyncFrom / SyncTo are the dates a synchronization would fetch (project
	// period plus the chosen extra months); empty without a project period.
	SyncFrom string `json:"syncFrom"`
	SyncTo   string `json:"syncTo"`
}

// SystemDTO is one selectable external system type.
type SystemDTO struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Available bool   `json:"available"`
}

// SystemConfigDTO is the stored connection configuration WITHOUT the secrets:
// the frontend only learns whether a password / client secret is set.
type SystemConfigDTO struct {
	BaseURL         string `json:"baseUrl"`
	Username        string `json:"username"`
	ClientID        string `json:"clientId"`
	PasswordSet     bool   `json:"passwordSet"`
	ClientSecretSet bool   `json:"clientSecretSet"`
}

// ConnTestDTO is the result of a successful connection test.
type ConnTestDTO struct {
	ProjectName string `json:"projectName"`
	Status      int    `json:"status"`
	Start       string `json:"start"`
	End         string `json:"end"`
}

// SyncProgressDTO is the payload of the "monitor:progress" event.
type SyncProgressDTO struct {
	Fetched int `json:"fetched"`
	Total   int `json:"total"` // 0 until the system announced it
}

// MonitorActionDTO is the booked effort matched to one tracked action.
type MonitorActionDTO struct {
	ID         string `json:"id"`
	ExternalID string `json:"externalId"`
	Name       string `json:"name"`
	Actual     string `json:"actual"`
	Entries    int    `json:"entries"`
	// Est is the action's estimation; Remaining = Est - Actual (negative when
	// over budget) and RemainingPct its share of Est ("" without an estimation).
	Est          string `json:"est"`
	Remaining    string `json:"remaining"`
	RemainingPct string `json:"remainingPct"`
	// IsDefault marks the action that received unassigned bookings.
	IsDefault      bool   `json:"isDefault"`
	DefaultActual  string `json:"defaultActual"`
	DefaultEntries int    `json:"defaultEntries"`
	// Hours and days (project working days) of the total and of its two parts:
	// bookings that reference the External ID, and unassigned bookings credited
	// because this is the default action.
	ActualHours    string `json:"actualHours"`
	ActualDays     string `json:"actualDays"`
	MatchedHours   string `json:"matchedHours"`
	MatchedDays    string `json:"matchedDays"`
	MatchedEntries int    `json:"matchedEntries"`
	DefaultHours   string `json:"defaultHours"`
	DefaultDays    string `json:"defaultDays"`
}

// MonitorSectionDTO is one row (section = work package) of the result table.
type MonitorSectionDTO struct {
	Name       string `json:"name"`
	Actions    int    `json:"actions"`
	Tracked    int    `json:"tracked"`
	Est        string `json:"est"`        // estimated effort of all actions
	EstTracked string `json:"estTracked"` // estimated effort of tracked actions
	Actual     string `json:"actual"`
	Entries    int    `json:"entries"`
	// Remaining = EstTracked - Actual, RemainingPct its share of EstTracked.
	Remaining    string `json:"remaining"`
	RemainingPct string `json:"remainingPct"`
	// DefaultActual/DefaultEntries: unassigned bookings credited to this section
	// because it is the default action (included in Actual/Entries).
	DefaultActual  string             `json:"defaultActual"`
	DefaultEntries int                `json:"defaultEntries"`
	Details        []MonitorActionDTO `json:"details"`
}

// MonitorSummaryDTO is the synchronization result.
type MonitorSummaryDTO struct {
	Sections         []MonitorSectionDTO `json:"sections"`
	Est              string              `json:"est"`
	EstTracked       string              `json:"estTracked"`
	Actual           string              `json:"actual"`
	Entries          int                 `json:"entries"`
	Remaining        string              `json:"remaining"`
	RemainingPct     string              `json:"remainingPct"`
	UnmatchedActual  string              `json:"unmatchedActual"`
	UnmatchedEntries int                 `json:"unmatchedEntries"`
	DefaultAction    string              `json:"defaultAction"`
	DefaultActual    string              `json:"defaultActual"`
	DefaultEntries   int                 `json:"defaultEntries"`
	Fetched          int                 `json:"fetched"`
	From             string              `json:"from"`
	To               string              `json:"to"`
	At               string              `json:"at"`
}

// syncResult is the raw result of the last synchronization.
type syncResult struct {
	project  *plan.Project // the project it belongs to
	sum      plan.MonitorSummary
	from, to time.Time
	at       time.Time
}

// setLastSync / getLastSync guard the last result: bindings run on different
// goroutines (a sync can run while the page asks for the summary).
func (a *App) setLastSync(r *syncResult) {
	a.syncMu.Lock()
	a.lastSync = r
	a.syncMu.Unlock()
}

func (a *App) getLastSync() *syncResult {
	a.syncMu.Lock()
	defer a.syncMu.Unlock()
	return a.lastSync
}

func (a *App) cfgPath() string {
	if a.configPath != "" {
		return a.configPath
	}
	p, err := extsys.DefaultConfigPath()
	if err != nil {
		return ""
	}
	return p
}

func (a *App) emit(name string, data any) {
	if a.app != nil {
		a.app.Event.Emit(name, data)
	}
}

// MonitorSystems lists the external system types for the pull-down. CSV is
// included but not available.
func (a *App) MonitorSystems() []SystemDTO {
	var out []SystemDTO
	for _, s := range extsys.Systems() {
		out = append(out, SystemDTO{ID: s.ID, Label: s.Label, Available: s.Available})
	}
	return out
}

// SetMonitor stores the monitoring setting. Switching monitoring ON requires a
// supported system type and the project ID in that system; otherwise an error
// is returned and nothing changes. The system and ID may be stored while
// monitoring is off.
func (a *App) SetMonitor(enabled bool, system, projectID string) (*ProjectDTO, error) {
	system = strings.TrimSpace(system)
	projectID = strings.TrimSpace(projectID)
	if system != "" && !plan.IsSupportedSystem(system) {
		return nil, fmt.Errorf("system %q is not supported yet", system)
	}
	if system == plan.SystemAerion && projectID != "" {
		if n, err := strconv.Atoi(projectID); err != nil || n <= 0 {
			return nil, fmt.Errorf("the Aerion project ID must be a positive number")
		}
	}
	m := a.project.Monitor // keeps default section and period options
	m.Enabled, m.System, m.ProjectID = enabled, system, projectID
	if err := m.Validate(); err != nil {
		return nil, err
	}
	if m != a.project.Monitor {
		a.setLastSync(nil)
	}
	a.project.Monitor = m
	return toDTO(a.project), nil
}

// SetMonitorDefaultAction chooses the default action (work package) that
// receives all bookings matching no action's External ID. An empty id clears
// it. It can be changed while monitoring is on.
func (a *App) SetMonitorDefaultAction(id string) (*ProjectDTO, error) {
	if id != "" && !a.project.HasAction(id) {
		return nil, fmt.Errorf("%q is not an action of this project", id)
	}
	if wp := a.project.WorkPackage(id); wp != nil && !wp.TracksEffort() {
		return nil, fmt.Errorf("action %q has the contour None and takes no bookings; it cannot be the default action", wp.Name)
	}
	if wp := a.project.WorkPackage(id); wp != nil && strings.EqualFold(strings.TrimSpace(wp.Cluster), "phases") {
		return nil, fmt.Errorf("action %q is in the Phases lane, which is a view and not a work package", wp.Name)
	}
	if a.project.Monitor.DefaultAction != id {
		a.setLastSync(nil)
	}
	a.project.Monitor.DefaultAction = id
	return toDTO(a.project), nil
}

// SetMonitorRange chooses whether a synchronization also covers one month
// before the project start and/or one month after the project end.
func (a *App) SetMonitorRange(before, after bool) (*ProjectDTO, error) {
	m := &a.project.Monitor
	if m.IncludeBefore != before || m.IncludeAfter != after {
		a.setLastSync(nil)
	}
	m.IncludeBefore, m.IncludeAfter = before, after
	return toDTO(a.project), nil
}

// SetActionExternalID sets an action's External ID (up to 16 characters,
// unique in the project). It can be changed at any time, also while monitoring
// is on: the last synchronization result is then cleared and the new ID is used
// by the next synchronization. An action with the contour None is not a real
// action and cannot carry an External ID (clearing one is always allowed).
func (a *App) SetActionExternalID(id, value string) (*ProjectDTO, error) {
	wp := a.project.WorkPackage(id)
	if wp == nil {
		return nil, fmt.Errorf("work package %q not found", id)
	}
	v, err := plan.NormalizeExternalID(value)
	if err != nil {
		return nil, err
	}
	if v != "" && !wp.TracksEffort() {
		return nil, fmt.Errorf("action %q has the contour None and cannot carry an External ID", wp.Name)
	}
	if o := a.project.ExternalIDOwner(v, id); o != nil {
		return nil, fmt.Errorf("external id %q is already used by action %q", v, o.Name)
	}
	wp.ExternalID = v
	a.setLastSync(nil)
	return toDTO(a.project), nil
}

// GetSystemConfig returns the stored connection settings of a system without
// its secrets.
func (a *App) GetSystemConfig(system string) (*SystemConfigDTO, error) {
	if system != plan.SystemAerion {
		return nil, fmt.Errorf("system %q has no configuration", system)
	}
	c, err := extsys.LoadAerion(a.cfgPath())
	if err != nil {
		return nil, err
	}
	return &SystemConfigDTO{
		BaseURL: c.BaseURL, Username: c.Username, ClientID: c.ClientID,
		PasswordSet: c.Password != "", ClientSecretSet: c.ClientSecret != "",
	}, nil
}

// SaveSystemConfig stores the connection settings in the user's
// ~/.pt-pm/config.yaml. An empty password or client secret keeps the stored
// one, so the form can be saved without re-typing secrets.
func (a *App) SaveSystemConfig(system, baseURL, username, password, clientID, clientSecret string) error {
	if system != plan.SystemAerion {
		return fmt.Errorf("system %q has no configuration", system)
	}
	baseURL = strings.TrimSpace(baseURL)
	if err := extsys.ValidateBaseURL(baseURL); err != nil {
		return err
	}
	cur, err := extsys.LoadAerion(a.cfgPath())
	if err != nil {
		return err
	}
	cur.BaseURL = baseURL
	cur.Username = strings.TrimSpace(username)
	cur.ClientID = strings.TrimSpace(clientID)
	if password != "" {
		cur.Password = password
	}
	if clientSecret != "" {
		cur.ClientSecret = clientSecret
	}
	path := a.cfgPath()
	if path == "" {
		return fmt.Errorf("cannot determine the home directory")
	}
	return extsys.SaveAerion(path, cur)
}

// TestSystemConnection checks the credentials and that the project exists in
// the external system (read-only).
func (a *App) TestSystemConnection(system, projectID string) (*ConnTestDTO, error) {
	drv, err := extsys.NewDriver(system, a.cfgPath())
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	info, err := drv.Check(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &ConnTestDTO{ProjectName: info.Name, Status: info.Status, Start: info.StartDate, End: info.EndDate}, nil
}

// SyncMonitor fetches every single-line booking of the external project from
// the project's start to its end, matches them to the actions' External IDs and
// returns the per-section summary. Progress is reported through the
// "monitor:progress" event.
func (a *App) SyncMonitor() (*MonitorSummaryDTO, error) {
	a.syncMu.Lock()
	if a.syncCancel != nil {
		a.syncMu.Unlock()
		return nil, fmt.Errorf("a synchronization is already running")
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.syncCancel = cancel
	a.syncMu.Unlock()
	defer func() {
		a.syncMu.Lock()
		a.syncCancel = nil
		a.syncMu.Unlock()
		cancel()
	}()

	p := a.project
	if err := p.Monitor.Validate(); err != nil {
		return nil, err
	}
	if !p.Monitor.Enabled {
		return nil, fmt.Errorf("monitoring is switched off")
	}
	start, ok1 := p.Start()
	end, ok2 := p.End()
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("the project needs a start and an end date")
	}
	from, to := p.Monitor.SyncRange(start, end)
	def := p.Monitor.DefaultAction
	if wp := p.WorkPackage(def); def != "" && wp != nil && !wp.TracksEffort() {
		return nil, fmt.Errorf("the default action %q has the contour None and takes no bookings - choose another one on the Monitor page", wp.Name)
	}
	if def != "" && !p.HasAction(def) {
		return nil, fmt.Errorf("the default action %q no longer exists - choose another one on the Monitor page", def)
	}
	drv, err := extsys.NewDriver(p.Monitor.System, a.cfgPath())
	if err != nil {
		return nil, err
	}
	a.emit("monitor:progress", SyncProgressDTO{})
	bookings, err := drv.Fetch(ctx, p.Monitor.ProjectID, from, to, func(fetched, total int) {
		a.emit("monitor:progress", SyncProgressDTO{Fetched: fetched, Total: total})
	})
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("synchronization cancelled")
		}
		return nil, err
	}
	now := time.Now()
	res := p.MonitorSummary(bookings, def)
	// the synchronized actions take their real effort from the external system
	p.ApplySyncedEffort(res)
	p.Monitor.SyncedAt = now.Format("2006-01-02 15:04")
	a.setLastSync(&syncResult{project: p, sum: res, from: from, to: to, at: now})
	return a.monitorSummaryDTO(), nil
}

// CancelSync aborts a running synchronization.
func (a *App) CancelSync() {
	a.syncMu.Lock()
	defer a.syncMu.Unlock()
	if a.syncCancel != nil {
		a.syncCancel()
	}
}

// GetMonitorSummary returns the last synchronization result of the open
// project, or nil when there is none (or the project changed since).
func (a *App) GetMonitorSummary() *MonitorSummaryDTO {
	return a.monitorSummaryDTO()
}

func (a *App) monitorSummaryDTO() *MonitorSummaryDTO {
	r := a.getLastSync()
	if r == nil || r.project != a.project {
		return nil
	}
	p := a.project
	f := func(sec int64) string { return p.Format(plan.SecondsToBase(sec)) }
	hrs := func(sec int64) string { return p.FormatHours(plan.SecondsToBase(sec)) }
	days := func(sec int64) string { return p.FormatDays(plan.SecondsToBase(sec)) }
	d := &MonitorSummaryDTO{
		Sections:         []MonitorSectionDTO{},
		Est:              p.Format(r.sum.TotalEst),
		EstTracked:       p.Format(r.sum.TotalEstTracked),
		Actual:           f(r.sum.TotalSeconds),
		Remaining:        remStr(p, r.sum.TotalEstTracked, r.sum.TotalSeconds),
		RemainingPct:     pctStr(r.sum.TotalEstTracked, r.sum.TotalSeconds),
		Entries:          r.sum.TotalEntries,
		UnmatchedActual:  f(r.sum.UnmatchedSeconds),
		UnmatchedEntries: r.sum.UnmatchedEntries,
		DefaultAction:    r.sum.DefaultAction,
		DefaultActual:    f(r.sum.DefaultSeconds),
		DefaultEntries:   r.sum.DefaultEntries,
		Fetched:          r.sum.Fetched,
		From:             r.from.Format(plan.DateLayout),
		To:               r.to.Format(plan.DateLayout),
		At:               r.at.Format("2006-01-02 15:04"),
	}
	for _, s := range r.sum.Sections {
		sd := MonitorSectionDTO{
			Name: s.Name, Actions: s.Actions, Tracked: s.Tracked,
			Est: p.Format(s.Est), EstTracked: p.Format(s.EstTracked),
			Actual: f(s.Seconds), Entries: s.Entries,
			Remaining: remStr(p, s.EstTracked, s.Seconds), RemainingPct: pctStr(s.EstTracked, s.Seconds), Details: []MonitorActionDTO{},
		}
		for _, x := range s.Details {
			sd.Details = append(sd.Details, MonitorActionDTO{
				ID: x.ID, ExternalID: x.ExternalID, Name: x.Name, Actual: f(x.Seconds), Entries: x.Entries,
				Est: p.Format(x.Est), Remaining: remStr(p, x.Est, x.Seconds), RemainingPct: pctStr(x.Est, x.Seconds),
				IsDefault: x.IsDefault, DefaultActual: f(x.DefaultSeconds), DefaultEntries: x.DefaultEntries,
				ActualHours: hrs(x.Seconds), ActualDays: days(x.Seconds),
				MatchedHours: hrs(x.MatchedSeconds), MatchedDays: days(x.MatchedSeconds), MatchedEntries: x.MatchedEntries,
				DefaultHours: hrs(x.DefaultSeconds), DefaultDays: days(x.DefaultSeconds),
			})
		}
		d.Sections = append(d.Sections, sd)
	}
	return d
}

// remStr formats the remaining effort (estimation minus booked seconds).
func remStr(p *plan.Project, est plan.Base, seconds int64) string {
	rem, _, _ := plan.Remaining(est, seconds)
	return p.Format(rem)
}

// pctStr formats the remaining effort as a percent of the estimation ("" when
// there is no estimation).
func pctStr(est plan.Base, seconds int64) string {
	_, pct, ok := plan.Remaining(est, seconds)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%d %%", pct)
}
