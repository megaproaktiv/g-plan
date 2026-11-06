package extsys

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/megaproaktiv/g-plan/src/app/plan"
)

// Aerion is the driver for the Aerion (CoffeeCup) REST API. The authentication
// (OAuth2 password grant) and the request style follow the reporting tool's
// aerion/client.go; time entries are read from GET /v1/time-entries. Only GET
// requests are made against the API (plus the token request), so the driver
// never changes data in Aerion.
type Aerion struct {
	cfg  AerionConfig
	http *http.Client

	mu    sync.Mutex
	token string
}

// NewAerion creates a driver for the given connection settings.
func NewAerion(cfg AerionConfig) *Aerion {
	return &Aerion{cfg: cfg, http: &http.Client{Timeout: 60 * time.Second}}
}

// aerionPageSize is the page size for time entries (API maximum is 50000;
// smaller pages give smoother progress). A variable so tests can shrink it.
var aerionPageSize = 5000

const (
	aerionMaxRetries = 3
	maxErrBody       = 300
)

func (a *Aerion) baseURL() string { return strings.TrimSuffix(strings.TrimSpace(a.cfg.BaseURL), "/") }

func truncate(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > maxErrBody {
		s = s[:maxErrBody] + "..."
	}
	return s
}

// authenticate obtains a bearer token (OAuth2 password grant, form-encoded).
func (a *Aerion) authenticate(ctx context.Context) error {
	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("username", a.cfg.Username)
	form.Set("password", a.cfg.Password)
	form.Set("client_id", a.cfg.ClientID)
	form.Set("client_secret", a.cfg.ClientSecret)
	form.Set("companyurl", a.baseURL())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL()+"/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := a.http.Do(req)
	if err != nil {
		return fmt.Errorf("authenticate: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed (HTTP %d): %s", resp.StatusCode, truncate(body))
	}
	var res struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &res); err != nil || res.AccessToken == "" {
		return fmt.Errorf("authentication response has no access token")
	}
	a.mu.Lock()
	a.token = res.AccessToken
	a.mu.Unlock()
	return nil
}

func (a *Aerion) bearer() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.token
}

// get performs an authenticated GET. It authenticates on first use, retries
// once after a 401 (expired token), and backs off on 429/5xx.
func (a *Aerion) get(ctx context.Context, path string) ([]byte, error) {
	if a.bearer() == "" {
		if err := a.authenticate(ctx); err != nil {
			return nil, err
		}
	}
	reauthed := false
	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL()+path, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+a.bearer())
		resp, err := a.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request failed: %w", err)
		}
		body, rerr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if rerr != nil {
			return nil, fmt.Errorf("read response: %w", rerr)
		}
		switch {
		case resp.StatusCode >= 200 && resp.StatusCode < 300:
			return body, nil
		case resp.StatusCode == http.StatusUnauthorized && !reauthed:
			reauthed = true
			if err := a.authenticate(ctx); err != nil {
				return nil, err
			}
		case (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500) && attempt < aerionMaxRetries:
			wait := time.Duration(attempt+1) * 2 * time.Second
			if s, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && s > 0 && s <= 30 {
				wait = time.Duration(s) * time.Second
			}
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		default:
			return nil, fmt.Errorf("request failed (HTTP %d): %s", resp.StatusCode, truncate(body))
		}
	}
}

func parseProjectID(id string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(id))
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("Aerion project ID %q must be a positive number", id)
	}
	return n, nil
}

// Check verifies the credentials and that the project exists.
func (a *Aerion) Check(ctx context.Context, projectID string) (ProjectInfo, error) {
	pid, err := parseProjectID(projectID)
	if err != nil {
		return ProjectInfo{}, err
	}
	body, err := a.get(ctx, fmt.Sprintf("/v1/projects/%d", pid))
	if err != nil {
		return ProjectInfo{}, err
	}
	var res struct {
		Project struct {
			ID        int    `json:"id"`
			Name      string `json:"name"`
			Status    int    `json:"status"`
			StartDate string `json:"startDate"`
			EndDate   string `json:"endDate"`
		} `json:"project"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return ProjectInfo{}, fmt.Errorf("parse project: %w", err)
	}
	p := res.Project
	return ProjectInfo{ID: strconv.Itoa(p.ID), Name: p.Name, Status: p.Status,
		StartDate: dateOnly(p.StartDate), EndDate: dateOnly(p.EndDate)}, nil
}

func dateOnly(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

// timeEntry is the subset of the Aerion TimeEntry record that is used.
type timeEntry struct {
	ID       int64  `json:"id"`
	Day      string `json:"day"`
	Duration int    `json:"duration"` // seconds actually tracked
	Comment  string `json:"comment"`  // the booking text
}

type timeEntriesPage struct {
	TimeEntries []timeEntry `json:"timeEntries"`
	Meta        struct {
		Total int `json:"total"`
	} `json:"meta"`
}

// Fetch reads all WORK time entries of the project between from and to
// (inclusive), page by page (skip/limit) until meta.total is reached.
func (a *Aerion) Fetch(ctx context.Context, projectID string, from, to time.Time, onProgress Progress) ([]plan.Booking, error) {
	pid, err := parseProjectID(projectID)
	if err != nil {
		return nil, err
	}
	where := map[string]any{"and": []any{
		map[string]any{"project": pid},
		map[string]any{"trackingType": "WORK"},
		map[string]any{"day": map[string]string{
			">=": from.Format(plan.DateLayout),
			"<=": to.Format(plan.DateLayout),
		}},
	}}
	wj, err := json.Marshal(where)
	if err != nil {
		return nil, err
	}
	var out []plan.Booking
	total := -1
	for skip := 0; total < 0 || skip < total; {
		path := fmt.Sprintf("/v1/time-entries?limit=%d&skip=%d&sort=%s&where=%s",
			aerionPageSize, skip, url.QueryEscape("id ASC"), url.QueryEscape(string(wj)))
		body, err := a.get(ctx, path)
		if err != nil {
			return nil, err
		}
		var page timeEntriesPage
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("parse time entries: %w", err)
		}
		total = page.Meta.Total
		for _, e := range page.TimeEntries {
			out = append(out, plan.Booking{ID: e.ID, Day: e.Day, Seconds: e.Duration, Text: e.Comment})
		}
		skip += len(page.TimeEntries)
		if onProgress != nil {
			onProgress(len(out), total)
		}
		if len(page.TimeEntries) == 0 {
			break // defensive: never loop on an empty page
		}
	}
	return out, nil
}
