package extsys

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeAerion serves just enough of the Aerion API for the driver and records
// every request so tests can assert the driver is strictly read-only.
type fakeAerion struct {
	entries  []map[string]any
	auths    atomic.Int32
	expire   atomic.Bool // next time-entries call returns 401 once
	throttle atomic.Int32

	mu      sync.Mutex
	methods []string
	wheres  []string
}

func (f *fakeAerion) record(r *http.Request) {
	f.mu.Lock()
	f.methods = append(f.methods, r.Method+" "+r.URL.Path)
	f.mu.Unlock()
}

func (f *fakeAerion) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		f.record(r)
		_ = r.ParseForm()
		if r.PostForm.Get("grant_type") != "password" || r.PostForm.Get("client_secret") != "sec" {
			http.Error(w, `{"error":"invalid_client"}`, http.StatusUnauthorized)
			return
		}
		n := f.auths.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "tok" + strconv.Itoa(int(n))})
	})
	mux.HandleFunc("/v1/projects/", func(w http.ResponseWriter, r *http.Request) {
		f.record(r)
		if r.URL.Path != "/v1/projects/42" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"project": map[string]any{
			"id": 42, "name": "Demo", "status": 1, "startDate": "2026-01-01T00:00:00.000Z", "endDate": "2026-12-31T00:00:00.000Z"}})
	})
	mux.HandleFunc("/v1/time-entries", func(w http.ResponseWriter, r *http.Request) {
		f.record(r)
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer tok") {
			http.Error(w, "no auth", http.StatusUnauthorized)
			return
		}
		if f.expire.CompareAndSwap(true, false) {
			http.Error(w, "expired", http.StatusUnauthorized)
			return
		}
		if f.throttle.Add(-1) >= 0 {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "slow down", http.StatusTooManyRequests)
			return
		}
		f.mu.Lock()
		f.wheres = append(f.wheres, r.URL.Query().Get("where"))
		f.mu.Unlock()
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		skip, _ := strconv.Atoi(r.URL.Query().Get("skip"))
		end := skip + limit
		if end > len(f.entries) {
			end = len(f.entries)
		}
		page := []map[string]any{}
		if skip < len(f.entries) {
			page = f.entries[skip:end]
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"timeEntries": page, "meta": map[string]any{"total": len(f.entries)}})
	})
	return mux
}

func newFake(n int) (*fakeAerion, *Aerion, func()) {
	f := &fakeAerion{}
	for i := 0; i < n; i++ {
		f.entries = append(f.entries, map[string]any{
			"id": i + 1, "day": "2026-02-02", "duration": 600, "comment": fmt.Sprintf("WP-%d work", i%3),
			"user": 7, "project": 42, "externalId": nil,
		})
	}
	srv := httptest.NewServer(f.handler())
	// plain http is accepted only for localhost; httptest listens on 127.0.0.1
	a := NewAerion(AerionConfig{BaseURL: srv.URL, Username: "u", Password: "p", ClientID: "c", ClientSecret: "sec"})
	return f, a, srv.Close
}

func day(s string) time.Time { d, _ := time.Parse("2006-01-02", s); return d }

func TestFetchPagesUntilTotalAndIsReadOnly(t *testing.T) {
	old := aerionPageSize
	aerionPageSize = 4
	defer func() { aerionPageSize = old }()
	f, a, closeSrv := newFake(10)
	defer closeSrv()

	var last [2]int
	calls := 0
	got, err := a.Fetch(context.Background(), "42", day("2026-01-01"), day("2026-03-01"), func(fe, to int) {
		last = [2]int{fe, to}
		calls++
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 10 {
		t.Fatalf("fetched %d, want 10", len(got))
	}
	if calls != 3 || last != [2]int{10, 10} {
		t.Errorf("progress calls=%d last=%v", calls, last)
	}
	if got[0].Text != "WP-0 work" || got[0].Seconds != 600 || got[0].Day != "2026-02-02" || got[0].ID != 1 {
		t.Errorf("mapping: %+v", got[0])
	}
	for _, m := range f.methods {
		if !strings.HasPrefix(m, "GET ") && m != "POST /oauth2/token" {
			t.Errorf("unexpected request %q: driver must be read-only", m)
		}
	}
	s := f.wheres[0]
	for _, want := range []string{`"project":42`, `"trackingType":"WORK"`, `"2026-01-01"`, `"2026-03-01"`} {
		if !strings.Contains(s, want) {
			t.Errorf("where %s lacks %s", s, want)
		}
	}
}

func TestFetchEmptyAndBadProject(t *testing.T) {
	_, a, closeSrv := newFake(0)
	defer closeSrv()
	got, err := a.Fetch(context.Background(), "42", day("2026-01-01"), day("2026-03-01"), nil)
	if err != nil || len(got) != 0 {
		t.Errorf("empty: %v %v", got, err)
	}
	for _, bad := range []string{"", "abc", "-3", "0", "4 2"} {
		if _, err := a.Fetch(context.Background(), bad, day("2026-01-01"), day("2026-03-01"), nil); err == nil {
			t.Errorf("project id %q must be rejected", bad)
		}
	}
}

func TestReauthenticatesAfter401(t *testing.T) {
	f, a, closeSrv := newFake(3)
	defer closeSrv()
	f.expire.Store(true)
	got, err := a.Fetch(context.Background(), "42", day("2026-01-01"), day("2026-03-01"), nil)
	if err != nil || len(got) != 3 {
		t.Fatalf("got %d, err %v", len(got), err)
	}
	if f.auths.Load() != 2 {
		t.Errorf("authenticated %d times, want 2 (initial + after 401)", f.auths.Load())
	}
}

func TestRetriesOn429(t *testing.T) {
	f, a, closeSrv := newFake(2)
	defer closeSrv()
	f.throttle.Store(1)
	start := time.Now()
	got, err := a.Fetch(context.Background(), "42", day("2026-01-01"), day("2026-03-01"), nil)
	if err != nil || len(got) != 2 {
		t.Fatalf("got %d, err %v", len(got), err)
	}
	if time.Since(start) < 900*time.Millisecond {
		t.Error("expected to wait for Retry-After")
	}
}

func TestFetchHonoursCancellation(t *testing.T) {
	_, a, closeSrv := newFake(5)
	defer closeSrv()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := a.Fetch(ctx, "42", day("2026-01-01"), day("2026-03-01"), nil); err == nil {
		t.Error("cancelled context must abort")
	}
}

func TestAuthFailureDoesNotLeakSecrets(t *testing.T) {
	_, a, closeSrv := newFake(1)
	defer closeSrv()
	a.cfg.ClientSecret = "wrong-secret-value"
	a.cfg.Password = "my-password-value"
	_, err := a.Check(context.Background(), "42")
	if err == nil {
		t.Fatal("expected auth failure")
	}
	for _, secret := range []string{"wrong-secret-value", "my-password-value"} {
		if strings.Contains(err.Error(), secret) {
			t.Errorf("error leaks a secret: %v", err)
		}
	}
}

func TestCheckProject(t *testing.T) {
	_, a, closeSrv := newFake(0)
	defer closeSrv()
	info, err := a.Check(context.Background(), "42")
	if err != nil {
		t.Fatal(err)
	}
	if info.Name != "Demo" || info.StartDate != "2026-01-01" || info.EndDate != "2026-12-31" {
		t.Errorf("%+v", info)
	}
	if _, err := a.Check(context.Background(), "99"); err == nil {
		t.Error("unknown project must fail")
	}
}

// ---- config ----

func TestConfigRoundTripPermissionsAndOtherKeys(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".pt-pm")
	path := filepath.Join(dir, "config.yaml")
	if c, err := LoadAerion(path); err != nil || c != (AerionConfig{}) {
		t.Fatalf("missing file: %+v %v", c, err)
	}
	cfg := AerionConfig{BaseURL: "https://x.example", Username: "u", Password: "p", ClientID: "c", ClientSecret: "s"}
	if err := SaveAerion(path, cfg); err != nil {
		t.Fatal(err)
	}
	if st, _ := os.Stat(path); st.Mode().Perm() != 0o600 {
		t.Errorf("file mode %v, want 0600", st.Mode().Perm())
	}
	if st, _ := os.Stat(dir); st.Mode().Perm() != 0o700 {
		t.Errorf("dir mode %v, want 0700", st.Mode().Perm())
	}
	got, err := LoadAerion(path)
	if err != nil || got != cfg {
		t.Fatalf("round trip: %+v %v", got, err)
	}
	b, _ := os.ReadFile(path)
	if err := os.WriteFile(path, append(b, []byte("other:\n  keep: me\n")...), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg.Username = "u2"
	if err := SaveAerion(path, cfg); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(path)
	if !strings.Contains(string(b), "keep: me") {
		t.Error("other top-level keys must be preserved")
	}
	if got, _ := LoadAerion(path); got.Username != "u2" {
		t.Errorf("update lost: %+v", got)
	}
	if st, _ := os.Stat(path); st.Mode().Perm() != 0o600 {
		t.Errorf("mode after update %v, want 0600", st.Mode().Perm())
	}
}

func TestConfigYamlKeysMatchReportingTool(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.yaml")
	y := "thresholds:\n  x: 1\naerion:\n  base_url: \"https://a.example\"\n  username: \"u\"\n  password: \"p\"\n  client_id: \"ci\"\n  client_secret: \"cs\"\n"
	if err := os.WriteFile(path, []byte(y), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := LoadAerion(path)
	if err != nil || c.BaseURL != "https://a.example" || c.ClientID != "ci" || c.ClientSecret != "cs" || c.Password != "p" {
		t.Errorf("%+v %v", c, err)
	}
}

func TestCompleteAndBaseURL(t *testing.T) {
	err := AerionConfig{BaseURL: "https://a.example"}.Complete()
	if err == nil || !strings.Contains(err.Error(), "username") || !strings.Contains(err.Error(), "client secret") {
		t.Errorf("missing fields should be named: %v", err)
	}
	for _, ok := range []string{"https://a.example", "http://localhost:8080", "http://127.0.0.1:9"} {
		if ValidateBaseURL(ok) != nil {
			t.Errorf("%s should be valid", ok)
		}
	}
	for _, bad := range []string{"", "a.example", "http://a.example", "ftp://a.example", "https://"} {
		if ValidateBaseURL(bad) == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

func TestNewDriver(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.yaml")
	if _, err := NewDriver("aerion", path); err == nil {
		t.Error("unconfigured aerion must fail")
	}
	if _, err := NewDriver("csv", path); err == nil || !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("csv: %v", err)
	}
	if _, err := NewDriver("nope", path); err == nil {
		t.Error("unknown system must fail")
	}
	s := Systems()
	if len(s) != 2 || !s[0].Available || s[1].ID != "csv" || s[1].Available {
		t.Errorf("systems: %+v", s)
	}
}
