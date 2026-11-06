package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/megaproaktiv/g-plan/src/app/plan"
)

func monitorApp(t *testing.T) (*App, []string) {
	a := NewApp()
	a.configPath = filepath.Join(t.TempDir(), ".pt-pm", "config.yaml")
	a.project.Unit = plan.UnitHours
	s, _ := plan.ParseDateStr("2026-01-01")
	e, _ := plan.ParseDateStr("2026-03-01")
	a.project.SetRange(s, e)
	var ids []string
	for _, n := range []struct{ name, cluster, est string }{{"Design", "S1", "10"}, {"Build", "S1", "20"}, {"Test", "S2", "5"}} {
		if _, err := a.AddWorkPackage(n.name, "2026-01-05", "2026-02-05"); err != nil {
			t.Fatal(err)
		}
		wp := &a.project.Packages[len(a.project.Packages)-1]
		wp.Cluster = n.cluster
		if _, err := a.SetWorkPackageEstimation(wp.ID, n.est); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, wp.ID)
	}
	return a, ids
}

func TestSetMonitorRules(t *testing.T) {
	a, _ := monitorApp(t)
	if _, err := a.SetMonitor(true, "", ""); err == nil {
		t.Error("enabling without system and project id must fail")
	}
	if _, err := a.SetMonitor(true, "aerion", ""); err == nil {
		t.Error("enabling without project id must fail")
	}
	if _, err := a.SetMonitor(true, "", "123"); err == nil {
		t.Error("enabling without system must fail")
	}
	if _, err := a.SetMonitor(false, "csv", "1"); err == nil {
		t.Error("csv cannot be selected: not implemented")
	}
	if _, err := a.SetMonitor(false, "aerion", "abc"); err == nil {
		t.Error("aerion project id must be numeric")
	}
	if a.project.Monitor.Enabled {
		t.Fatal("failed calls must not change state")
	}
	// storing system+id while off is allowed, then switching on works
	if dto, err := a.SetMonitor(false, "aerion", "42"); err != nil || dto.Monitor.Enabled || dto.Monitor.ProjectID != "42" {
		t.Fatalf("store while off: %+v %v", dto, err)
	}
	dto, err := a.SetMonitor(true, "aerion", "42")
	if err != nil || !dto.Monitor.Enabled || dto.Monitor.System != "aerion" {
		t.Fatalf("enable: %+v %v", dto, err)
	}
}

func TestExternalIDRules(t *testing.T) {
	a, ids := monitorApp(t)
	if _, err := a.SetActionExternalID(ids[0], "WP-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetActionExternalID(ids[1], "wp-1"); err == nil {
		t.Error("duplicate (case-insensitive) must be rejected")
	}
	if _, err := a.SetActionExternalID(ids[0], "WP-1"); err != nil {
		t.Errorf("re-saving the own id must be allowed: %v", err)
	}
	if _, err := a.SetActionExternalID(ids[1], strings.Repeat("x", 17)); err == nil {
		t.Error("more than 16 characters must be rejected")
	}
	if a.project.WorkPackage(ids[1]).ExternalID != "" {
		t.Error("rejected value must not be stored")
	}
	if _, err := a.SetActionExternalID("nope", "X"); err == nil {
		t.Error("unknown action")
	}
	// editable while monitoring is on, too (it clears the last result)
	if _, err := a.SetMonitor(true, "aerion", "42"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetActionExternalID(ids[1], "WP-2"); err != nil {
		t.Fatalf("the external id must stay editable while monitoring is on: %v", err)
	}
	if _, err := a.SetMonitor(false, "aerion", "42"); err != nil {
		t.Fatal(err)
	}
	dto, err := a.SetActionExternalID(ids[1], "WP-2")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range dto.Packages {
		if p.ID == ids[1] && p.ExternalID != "WP-2" {
			t.Errorf("dto external id = %q", p.ExternalID)
		}
	}
	// clearing is allowed
	if _, err := a.SetActionExternalID(ids[1], ""); err != nil {
		t.Errorf("clear: %v", err)
	}
}

func TestSystemConfigNeverReturnsSecretsAndKeepsThem(t *testing.T) {
	a, _ := monitorApp(t)
	if err := a.SaveSystemConfig("aerion", "http://x.example", "u", "p", "c", "s"); err == nil {
		t.Error("plain http to a non-local host must be rejected")
	}
	if err := a.SaveSystemConfig("aerion", "https://x.example", "user", "pw-secret", "cid", "cs-secret"); err != nil {
		t.Fatal(err)
	}
	if st, err := os.Stat(a.configPath); err != nil || st.Mode().Perm() != 0o600 {
		t.Fatalf("config file must exist with mode 0600: %v %v", st, err)
	}
	cfg, err := a.GetSystemConfig("aerion")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(cfg)
	if strings.Contains(string(b), "pw-secret") || strings.Contains(string(b), "cs-secret") {
		t.Fatalf("secrets leaked to the frontend: %s", b)
	}
	if cfg.BaseURL != "https://x.example" || cfg.Username != "user" || cfg.ClientID != "cid" || !cfg.PasswordSet || !cfg.ClientSecretSet {
		t.Errorf("%+v", cfg)
	}
	// saving with empty secrets keeps the stored ones
	if err := a.SaveSystemConfig("aerion", "https://y.example", "user2", "", "cid2", ""); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(a.configPath)
	if !strings.Contains(string(raw), "pw-secret") || !strings.Contains(string(raw), "cs-secret") || !strings.Contains(string(raw), "y.example") {
		t.Errorf("secrets must be kept and other fields updated:\n%s", raw)
	}
	if err := a.SaveSystemConfig("csv", "https://x.example", "", "", "", ""); err == nil {
		t.Error("csv has no configuration yet")
	}
}

// lastWhere records the where filter of the most recent time-entries request.
var lastWhere atomic.Value

// fakeAerion is a minimal Aerion API: 2 pages of time entries for project 42.
func fakeAerionServer(t *testing.T, texts []string, seconds []int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/oauth2/token":
			_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "tok"})
		case r.URL.Path == "/v1/projects/42":
			_ = json.NewEncoder(w).Encode(map[string]any{"project": map[string]any{"id": 42, "name": "Demo Project", "status": 1}})
		case r.URL.Path == "/v1/time-entries":
			if r.Method != http.MethodGet {
				t.Errorf("non-GET request %s to %s", r.Method, r.URL.Path)
			}
			lastWhere.Store(r.URL.Query().Get("where"))
			skip, _ := strconv.Atoi(r.URL.Query().Get("skip"))
			var page []map[string]any
			if skip < len(texts) {
				for i := skip; i < len(texts); i++ {
					page = append(page, map[string]any{"id": i + 1, "day": "2026-02-02", "duration": seconds[i], "comment": texts[i]})
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"timeEntries": page, "meta": map[string]any{"total": len(texts)}})
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestSyncMonitorEndToEnd(t *testing.T) {
	a, ids := monitorApp(t)
	srv := fakeAerionServer(t,
		[]string{"WP-1 design work", "wp-1 review", "WP-2 build", "no id here", "WP-12 other"},
		[]int{3600, 1800, 7200, 900, 600})
	defer srv.Close()
	if err := a.SaveSystemConfig("aerion", srv.URL, "u", "p", "c", "s"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SyncMonitor(); err == nil {
		t.Error("sync with monitoring off must fail")
	}
	for id, ext := range map[string]string{ids[0]: "WP-1", ids[1]: "WP-2"} {
		if _, err := a.SetActionExternalID(id, ext); err != nil {
			t.Fatal(err)
		}
	}
	info, err := a.TestSystemConnection("aerion", "42")
	if err != nil || info.ProjectName != "Demo Project" {
		t.Fatalf("connection test: %+v %v", info, err)
	}
	if _, err := a.SetMonitor(true, "aerion", "42"); err != nil {
		t.Fatal(err)
	}
	if a.GetMonitorSummary() != nil {
		t.Error("no summary before the first sync")
	}
	sum, err := a.SyncMonitor()
	if err != nil {
		t.Fatal(err)
	}
	if sum.Fetched != 5 || sum.Entries != 3 || sum.UnmatchedEntries != 2 {
		t.Fatalf("counts: %+v", sum)
	}
	if sum.From != "2026-01-01" || sum.To != "2026-03-01" {
		t.Errorf("range = %s..%s", sum.From, sum.To)
	}
	if len(sum.Sections) != 2 {
		t.Fatalf("sections: %+v", sum.Sections)
	}
	s1, s2 := sum.Sections[0], sum.Sections[1]
	// hour mode: WP-1 = 5400s = 1.5h, WP-2 = 7200s = 2h; estimates 10+20 and 5
	if s1.Name != "S1" || s1.Actual != "3,5" || s1.Est != "30" || s1.EstTracked != "30" || s1.Tracked != 2 || s1.Entries != 3 {
		t.Errorf("S1: %+v", s1)
	}
	if s2.Name != "S2" || s2.Actual != "0" || s2.Tracked != 0 {
		t.Errorf("S2: %+v", s2)
	}
	if sum.Actual != "3,5" || len(s1.Details) != 2 || s1.Details[0].Actual != "1,5" || s1.Details[1].Actual != "2" {
		t.Errorf("actual=%s details=%+v", sum.Actual, s1.Details)
	}
	if sum.UnmatchedActual != "0,42" {
		t.Errorf("unmatched = %s", sum.UnmatchedActual)
	}
	// the last result is available later, and invalid for another project
	if a.GetMonitorSummary() == nil {
		t.Error("summary should be kept")
	}
	a.project = plan.NewProject("other", plan.UnitDays)
	if a.GetMonitorSummary() != nil {
		t.Error("summary must not leak into another project")
	}
}

func TestSyncMonitorErrors(t *testing.T) {
	a, _ := monitorApp(t)
	if _, err := a.SetMonitor(true, "aerion", "42"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SyncMonitor(); err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Errorf("unconfigured: %v", err)
	}
	a.project.Monitor = plan.Monitor{Enabled: true, System: "csv", ProjectID: "1"}
	if _, err := a.SyncMonitor(); err == nil {
		t.Error("csv sync must fail")
	}
	// the running flag is released after a failed sync
	if a.syncCancel != nil {
		t.Error("sync lock leaked after error")
	}
}

func TestDefaultActionSettingRules(t *testing.T) {
	a, ids := monitorApp(t) // Design, Build (S1), Test (S2)
	if _, err := a.SetMonitorDefaultAction("nope"); err == nil {
		t.Error("unknown action must be rejected")
	}
	if _, err := a.SetMonitorDefaultAction("S1"); err == nil {
		t.Error("a section name is not an action")
	}
	a.project.WorkPackage(ids[0]).Cluster = "Phases"
	if _, err := a.SetMonitorDefaultAction(ids[0]); err == nil {
		t.Error("an action in the Phases lane is not a work package")
	}
	a.project.WorkPackage(ids[0]).Cluster = "S1"
	dto, err := a.SetMonitorDefaultAction(ids[2])
	if err != nil || dto.Monitor.DefaultAction != ids[2] {
		t.Fatalf("set: %+v %v", dto, err)
	}
	// SetMonitor must keep the default action
	dto, err = a.SetMonitor(true, "aerion", "42")
	if err != nil || dto.Monitor.DefaultAction != ids[2] {
		t.Fatalf("SetMonitor lost the default action: %+v %v", dto, err)
	}
	// can be changed while monitoring is on, and cleared
	if dto, err = a.SetMonitorDefaultAction(ids[0]); err != nil || dto.Monitor.DefaultAction != ids[0] {
		t.Fatalf("change while on: %+v %v", dto, err)
	}
	if dto, err = a.SetMonitorDefaultAction(""); err != nil || dto.Monitor.DefaultAction != "" {
		t.Fatalf("clear: %+v %v", dto, err)
	}
	// persisted with the project
	if _, err := a.SetMonitorDefaultAction(ids[2]); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "p.ptplan")
	if err := a.project.SaveJSON(path); err != nil {
		t.Fatal(err)
	}
	q, err := plan.OpenJSON(path)
	if err != nil || q.Monitor.DefaultAction != ids[2] {
		t.Fatalf("reload: %+v %v", q.Monitor, err)
	}
}

func TestSyncAssignsUnmatchedToDefaultAction(t *testing.T) {
	a, ids := monitorApp(t) // ids: Design(S1), Build(S1), Test(S2)
	srv := fakeAerionServer(t,
		[]string{"WP-1 design", "no id here", "WP-12 other"},
		[]int{3600, 1800, 1800})
	defer srv.Close()
	if err := a.SaveSystemConfig("aerion", srv.URL, "u", "p", "c", "s"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetActionExternalID(ids[0], "WP-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetMonitor(true, "aerion", "42"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetMonitorDefaultAction(ids[2]); err != nil { // "Test" in S2, no External ID
		t.Fatal(err)
	}
	sum, err := a.SyncMonitor()
	if err != nil {
		t.Fatal(err)
	}
	if sum.UnmatchedEntries != 0 || sum.DefaultAction != ids[2] || sum.DefaultEntries != 2 || sum.DefaultActual != "1" {
		t.Fatalf("summary: %+v", sum)
	}
	s1, s2 := sum.Sections[0], sum.Sections[1]
	if s1.Actual != "1" || s1.Tracked != 1 || len(s1.Details) != 1 || s1.Details[0].IsDefault {
		t.Errorf("S1: %+v", s1)
	}
	if s2.Actual != "1" || s2.Entries != 2 || s2.Tracked != 1 || len(s2.Details) != 1 {
		t.Errorf("S2: %+v", s2)
	}
	d := s2.Details[0]
	if d.ID != ids[2] || !d.IsDefault || d.ExternalID != "" || d.Actual != "1" || d.DefaultActual != "1" || d.DefaultEntries != 2 {
		t.Errorf("default action row: %+v", d)
	}
	if sum.Actual != "2" || sum.Entries != 3 {
		t.Errorf("project total must include the default bookings: %+v", sum)
	}
	// changing the default invalidates the stale result
	if _, err := a.SetMonitorDefaultAction(""); err != nil {
		t.Fatal(err)
	}
	if a.GetMonitorSummary() != nil {
		t.Error("changing the default action must clear the last result")
	}
}

func TestSyncFailsWhenDefaultActionVanished(t *testing.T) {
	a, ids := monitorApp(t)
	if _, err := a.SetMonitorDefaultAction(ids[2]); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetMonitor(true, "aerion", "42"); err != nil {
		t.Fatal(err)
	}
	a.project.Packages = a.project.Packages[:2] // the default action is deleted
	if _, err := a.SyncMonitor(); err == nil || !strings.Contains(err.Error(), "default action") {
		t.Errorf("expected a clear error, got %v", err)
	}
}

func TestSyncRangeOptionsWidenTheFetchedPeriod(t *testing.T) {
	a, _ := monitorApp(t) // project 2026-01-01 .. 2026-03-01
	srv := fakeAerionServer(t, []string{"x"}, []int{60})
	defer srv.Close()
	if err := a.SaveSystemConfig("aerion", srv.URL, "u", "p", "c", "s"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetMonitor(true, "aerion", "42"); err != nil {
		t.Fatal(err)
	}
	where := func() string {
		lastWhere.Store("")
		sum, err := a.SyncMonitor()
		if err != nil {
			t.Fatal(err)
		}
		w, _ := lastWhere.Load().(string)
		_ = sum
		return w
	}
	// default: exactly the project period
	w := where()
	if !strings.Contains(w, "2026-01-01") || !strings.Contains(w, "2026-03-01") {
		t.Errorf("default period: %s", w)
	}
	// one month before only
	dto, err := a.SetMonitorRange(true, false)
	if err != nil || !dto.Monitor.IncludeBefore || dto.Monitor.SyncFrom != "2025-12-01" || dto.Monitor.SyncTo != "2026-03-01" {
		t.Fatalf("before: %+v %v", dto.Monitor, err)
	}
	if a.GetMonitorSummary() != nil {
		t.Error("changing the period must clear the stale result")
	}
	w = where()
	if !strings.Contains(w, "2025-12-01") || !strings.Contains(w, "2026-03-01") || strings.Contains(w, "2026-04-01") {
		t.Errorf("before only: %s", w)
	}
	// both
	dto, _ = a.SetMonitorRange(true, true)
	if dto.Monitor.SyncFrom != "2025-12-01" || dto.Monitor.SyncTo != "2026-04-01" {
		t.Fatalf("both: %+v", dto.Monitor)
	}
	w = where()
	if !strings.Contains(w, "2025-12-01") || !strings.Contains(w, "2026-04-01") {
		t.Errorf("both: %s", w)
	}
	sum := a.GetMonitorSummary()
	if sum.From != "2025-12-01" || sum.To != "2026-04-01" {
		t.Errorf("the result must report the range actually fetched: %s..%s", sum.From, sum.To)
	}
	// other monitor changes keep the period options
	if dto, err = a.SetMonitor(true, "aerion", "43"); err != nil || !dto.Monitor.IncludeBefore || !dto.Monitor.IncludeAfter {
		t.Errorf("SetMonitor must keep the options: %+v %v", dto.Monitor, err)
	}
	// persisted
	path := filepath.Join(t.TempDir(), "p.ptplan")
	if err := a.project.SaveJSON(path); err != nil {
		t.Fatal(err)
	}
	q, err := plan.OpenJSON(path)
	if err != nil || !q.Monitor.IncludeBefore || !q.Monitor.IncludeAfter {
		t.Errorf("reload: %+v %v", q.Monitor, err)
	}
	// and off again
	dto, _ = a.SetMonitorRange(false, false)
	if dto.Monitor.SyncFrom != "2026-01-01" || dto.Monitor.SyncTo != "2026-03-01" {
		t.Errorf("off: %+v", dto.Monitor)
	}
}

// The scenario of test-assignment.ptplan: one action (40 days planned) with the
// External ID "WP1" that is also the default action. One booking of 4 h
// references WP1, the other bookings carry no reference.
func TestActionShowsPlannedAndActualEffort(t *testing.T) {
	a := NewApp()
	a.configPath = filepath.Join(t.TempDir(), ".pt-pm", "config.yaml")
	s, _ := plan.ParseDateStr("2026-11-17")
	e, _ := plan.ParseDateStr("2027-07-31")
	a.project.SetRange(s, e)
	if _, err := a.AddWorkPackage("Project Management", "2026-11-17", "2027-07-31"); err != nil {
		t.Fatal(err)
	}
	id := a.project.Packages[0].ID
	if _, err := a.SetWorkPackageEstimation(id, "40"); err != nil { // 40 days
		t.Fatal(err)
	}
	if _, err := a.SetActionExternalID(id, "WP1"); err != nil {
		t.Fatal(err)
	}
	texts := []string{"WP1 kickoff and planning"}
	secs := []int{4 * 3600}
	for i := 0; i < 13; i++ { // 13 bookings without a reference: 48,5 h in total
		texts = append(texts, "general project work")
		if i < 12 {
			secs = append(secs, 13400)
		} else {
			secs = append(secs, 174600-12*13400)
		}
	}
	srv := fakeAerionServer(t, texts, secs)
	defer srv.Close()
	if err := a.SaveSystemConfig("aerion", srv.URL, "u", "p", "c", "s"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetMonitor(true, "aerion", "42"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetMonitorDefaultAction(id); err != nil {
		t.Fatal(err)
	}
	sum, err := a.SyncMonitor()
	if err != nil {
		t.Fatal(err)
	}
	d := sum.Sections[0].Details[0]
	if d.ActualHours != "52,5" || d.ActualDays != "6,56" || d.Entries != 14 {
		t.Errorf("total: %+v", d)
	}
	if d.MatchedHours != "4" || d.MatchedDays != "0,5" || d.MatchedEntries != 1 {
		t.Errorf("assigned to WP1: %+v", d)
	}
	if d.DefaultHours != "48,5" || d.DefaultEntries != 13 || !d.IsDefault {
		t.Errorf("unassigned: %+v", d)
	}
	// planned effort on the action itself, in hours and days
	w := toDTO(a.project).Packages[0]
	if w.Est != "40" || w.EstHours != "320" || w.EstDays != "40" {
		t.Errorf("planned: est=%s h=%s d=%s", w.Est, w.EstHours, w.EstDays)
	}
}

func TestBookingWithTwoReferencesIsSpreadThroughTheBinding(t *testing.T) {
	a, ids := monitorApp(t) // Design, Build (S1), Test (S2)
	srv := fakeAerionServer(t, []string{"WP-1 and WP-2 pairing", "WP-1 alone"}, []int{7200, 3600})
	defer srv.Close()
	if err := a.SaveSystemConfig("aerion", srv.URL, "u", "p", "c", "s"); err != nil {
		t.Fatal(err)
	}
	for id, ext := range map[string]string{ids[0]: "WP-1", ids[2]: "WP-2"} {
		if _, err := a.SetActionExternalID(id, ext); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := a.SetMonitor(true, "aerion", "42"); err != nil {
		t.Fatal(err)
	}
	sum, err := a.SyncMonitor()
	if err != nil {
		t.Fatal(err)
	}
	wp1, wp2 := sum.Sections[0].Details[0], sum.Sections[1].Details[0]
	// WP-1: half of the shared 2 h booking (1 h) + the 1 h booking = 2 h in 2 bookings
	if wp1.MatchedHours != "2" || wp1.MatchedEntries != 2 {
		t.Errorf("WP-1: %+v", wp1)
	}
	if wp2.MatchedHours != "1" || wp2.MatchedEntries != 1 {
		t.Errorf("WP-2 gets half of the shared booking: %+v", wp2)
	}
	if sum.Entries != 2 || sum.Actual != "3" || sum.UnmatchedEntries != 0 {
		t.Errorf("project counts each booking once: %+v", sum)
	}
}

// Synchronized actions take their real effort from the external system and
// cannot be edited; all other actions stay manual and editable.
func TestSyncFeedsRealEffortAndLocksTheField(t *testing.T) {
	a, ids := monitorApp(t) // Design, Build (S1), Test (S2); hour mode
	// Design: tracked by External ID WP-1; Build: untracked, manual value; Test: default action
	if _, err := a.SetActionExternalID(ids[0], "WP-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetWorkPackageRealEffort(ids[1], "3"); err != nil { // manual, monitoring still off
		t.Fatal(err)
	}
	if _, err := a.SetWorkPackageContour(ids[0], "Double-Peak"); err != nil {
		t.Fatal(err)
	}
	srv := fakeAerionServer(t,
		[]string{"WP-1 design", "unreferenced work"}, []int{3600, 1800})
	defer srv.Close()
	if err := a.SaveSystemConfig("aerion", srv.URL, "u", "p", "c", "s"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetMonitor(true, "aerion", "42"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetMonitorDefaultAction(ids[2]); err != nil {
		t.Fatal(err)
	}
	// before the first sync the synced actions are already locked
	if _, err := a.SetWorkPackageRealEffort(ids[0], "9"); err == nil || !strings.Contains(err.Error(), "synchronized") {
		t.Errorf("a synchronized action must reject manual real effort: %v", err)
	}
	if _, err := a.SyncMonitor(); err != nil {
		t.Fatal(err)
	}
	dto := toDTO(a.project)
	byID := map[string]WorkPackDTO{}
	for _, w := range dto.Packages {
		byID[w.ID] = w
	}
	d, b, tst := byID[ids[0]], byID[ids[1]], byID[ids[2]]
	if !d.Synced || b.Synced || !tst.Synced {
		t.Errorf("synced flags: design=%v build=%v test=%v", d.Synced, b.Synced, tst.Synced)
	}
	// Design: 1 h booked of 10 h planned
	if d.RealHours != "1" || d.ProgressPermille != 100 || d.SyncedAt == "" {
		t.Errorf("design: real=%s h, pure=%d, syncedAt=%q", d.RealHours, d.ProgressPermille, d.SyncedAt)
	}
	// Double-Peak at 10 %: the marker is behind the linear position (the start holds a lot of the effort)
	if d.ContourPermille >= d.ProgressPermille || d.ContourPermille != int(d.CompletionPos*1000+0.5) {
		t.Errorf("design contour rating %d should be below the pure %d and equal the marker position %v", d.ContourPermille, d.ProgressPermille, d.CompletionPos)
	}
	// the default action got the 30 min without a reference: 0,5 h of 5 h planned
	if tst.RealHours != "0,5" || tst.ProgressPermille != 100 || tst.ContourPermille != 100 { // Flat: same
		t.Errorf("default action: real=%s h pure=%d contour=%d", tst.RealHours, tst.ProgressPermille, tst.ContourPermille)
	}
	// the untracked action kept its manual value and stays editable
	if b.RealHours != "3" {
		t.Errorf("manual value must survive the sync: %s", b.RealHours)
	}
	if _, err := a.SetWorkPackageRealEffort(ids[1], "4"); err != nil {
		t.Errorf("an untracked action stays editable: %v", err)
	}
	if _, err := a.SetWorkPackageRealEffort(ids[0], "9"); err == nil {
		t.Error("the synchronized action must stay locked after the sync")
	}
	if got := a.project.WorkPackage(ids[0]).RealEst; got != 60 {
		t.Errorf("stored real effort = %d min, want 60", got)
	}
	// the values persist with the project, so reopening shows them without a sync
	path := filepath.Join(t.TempDir(), "p.ptplan")
	if err := a.project.SaveJSON(path); err != nil {
		t.Fatal(err)
	}
	q, err := plan.OpenJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	if q.WorkPackage(ids[0]).RealEst != 60 || q.Monitor.SyncedAt == "" {
		t.Errorf("reload: real=%d syncedAt=%q", q.WorkPackage(ids[0]).RealEst, q.Monitor.SyncedAt)
	}
	// switching monitoring off frees the fields (values stay)
	if _, err := a.SetMonitor(false, "aerion", "42"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SetWorkPackageRealEffort(ids[0], "2"); err != nil {
		t.Errorf("with monitoring off the field is editable again: %v", err)
	}
}
