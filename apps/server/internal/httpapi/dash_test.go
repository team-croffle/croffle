package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/team-croffle/croffle/apps/server/internal/auth"
	"github.com/team-croffle/croffle/apps/server/internal/store"
)

type fakeDash struct {
	fakeSyncer

	sessions map[string]store.Session
	account  store.Account
	teams    []string
	devices  []store.Device
	teamList []store.Team
	created  []store.Session
	deleted  [][]byte
}

func newFakeDash() *fakeDash {
	return &fakeDash{
		sessions: map[string]store.Session{},
		account:  store.Account{ID: "acct-1", Username: "alice"},
		teams:    []string{store.DefaultTeamID},
		teamList: []store.Team{{ID: store.DefaultTeamID, Name: "Default team"}},
	}
}

func (f *fakeDash) CreateSession(_ context.Context, session store.Session) error {
	f.created = append(f.created, session)
	f.sessions[string(session.TokenHash)] = session
	return nil
}

func (f *fakeDash) FindSession(_ context.Context, hash []byte, now time.Time) (*store.Session, error) {
	session, ok := f.sessions[string(hash)]
	if !ok || !session.ExpiresAt.After(now) {
		return nil, store.ErrNotFound
	}
	return &session, nil
}

func (f *fakeDash) DeleteSession(_ context.Context, hash []byte) error {
	f.deleted = append(f.deleted, hash)
	delete(f.sessions, string(hash))
	return nil
}

func (f *fakeDash) FindAccountByID(_ context.Context, id string) (*store.Account, error) {
	if f.account.ID != id {
		return nil, store.ErrNotFound
	}
	account := f.account
	return &account, nil
}

func (f *fakeDash) TeamsOf(_ context.Context, _ string) ([]string, error) { return f.teams, nil }

func (f *fakeDash) ListDevices(_ context.Context, _ string) ([]store.Device, error) {
	return f.devices, nil
}

func (f *fakeDash) ListTeams(_ context.Context) ([]store.Team, error) { return f.teamList, nil }

func dashHandler(t *testing.T, dash *fakeDash) http.Handler {
	t.Helper()
	return testHandler(t, func(o *Options) {
		o.Auth = &fakeAuth{}
		o.Dash = dash
	})
}

// login performs a dashboard login and returns the session cookie.
func dashLogin(t *testing.T, h http.Handler) *http.Cookie {
	t.Helper()

	rec := postJSON(t, h, "/dash/login", `{"username":"alice","password":"hunter2"}`, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d (%s)", rec.Code, rec.Body.String())
	}

	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == sessionCookie {
			return cookie
		}
	}
	t.Fatal("login set no session cookie")
	return nil
}

func withCookie(t *testing.T, h http.Handler, method, path string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestDashLoginSetsAnHttpOnlySessionCookie(t *testing.T) {
	dash := newFakeDash()
	h := dashHandler(t, dash)

	rec := postJSON(t, h, "/dash/login", `{"username":"alice","password":"hunter2"}`, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != sessionCookie {
		t.Fatalf("cookies = %+v", cookies)
	}
	cookie := cookies[0]

	// HttpOnly and SameSite=Strict: a script must not be able to read it, and
	// it must not ride along on a cross-site request.
	if !cookie.HttpOnly {
		t.Error("the session cookie is readable by scripts")
	}
	if cookie.SameSite != http.SameSiteStrictMode {
		t.Errorf("SameSite = %v, want Strict", cookie.SameSite)
	}

	// The value is a session, not a sync token: the store got its hash, and the
	// response body carries no token at all.
	if len(dash.created) != 1 {
		t.Fatalf("sessions created = %d, want 1", len(dash.created))
	}
	if string(dash.created[0].TokenHash) != string(auth.HashToken(cookie.Value)) {
		t.Error("the stored hash does not match the cookie")
	}
	if strings.Contains(rec.Body.String(), cookie.Value) {
		t.Error("the response body repeats the session value")
	}
}

// The browser must never hold a credential that can write, so a dashboard
// login issues no device token and leaves no device row.
//
// That the session value is also rejected as a bearer token needs the real auth
// service, so it is asserted in the end-to-end test instead: a fake
// authenticator here would accept anything.
func TestDashLoginIssuesNoSyncCredential(t *testing.T) {
	dash := newFakeDash()
	h := dashHandler(t, dash)

	dashLogin(t, h)

	if dash.fakeSyncer.got.AccountID != "" {
		t.Error("the dashboard login touched the sync store")
	}
	if len(dash.created) != 1 {
		t.Fatalf("sessions created = %d, want exactly the session", len(dash.created))
	}
}

func TestDashRoutesNeedASession(t *testing.T) {
	h := dashHandler(t, newFakeDash())

	for _, path := range []string{"/dash/me", "/dash/devices", "/dash/teams", "/dash/rows", "/dash/commits"} {
		t.Run(path, func(t *testing.T) {
			rec := withCookie(t, h, http.MethodGet, path, nil)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", rec.Code)
			}
			if got := errorCode(t, rec); got != "unauthorized" {
				t.Errorf("code = %q", got)
			}
		})
	}
}

func TestDashExpiredSessionClearsTheCookie(t *testing.T) {
	dash := newFakeDash()
	h := dashHandler(t, dash)
	cookie := dashLogin(t, h)

	// Expire it behind the browser's back.
	session := dash.sessions[string(auth.HashToken(cookie.Value))]
	session.ExpiresAt = fixedNow.Add(-time.Hour)
	dash.sessions[string(auth.HashToken(cookie.Value))] = session

	rec := withCookie(t, h, http.MethodGet, "/dash/me", cookie)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	// Clearing it stops the browser from sending something that can never work.
	for _, c := range rec.Result().Cookies() {
		if c.Name == sessionCookie && c.MaxAge >= 0 {
			t.Errorf("cookie MaxAge = %d, want it cleared", c.MaxAge)
		}
	}
}

func TestDashRefusesADisabledAccount(t *testing.T) {
	dash := newFakeDash()
	h := dashHandler(t, dash)
	cookie := dashLogin(t, h)

	disabled := fixedNow
	dash.account.DisabledAt = &disabled

	if rec := withCookie(t, h, http.MethodGet, "/dash/me", cookie); rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestDashLogoutDropsTheSession(t *testing.T) {
	dash := newFakeDash()
	h := dashHandler(t, dash)
	cookie := dashLogin(t, h)

	rec := withCookie(t, h, http.MethodPost, "/dash/logout", cookie)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if len(dash.deleted) != 1 {
		t.Errorf("deleted sessions = %d, want 1", len(dash.deleted))
	}
	if rec := withCookie(t, h, http.MethodGet, "/dash/me", cookie); rec.Code != http.StatusUnauthorized {
		t.Errorf("after logout: status = %d, want 401", rec.Code)
	}

	// Logging out without a session still clears the cookie rather than failing.
	if rec := withCookie(t, h, http.MethodPost, "/dash/logout", nil); rec.Code != http.StatusNoContent {
		t.Errorf("logout without a session = %d, want 204", rec.Code)
	}
}

func TestDashDevicesHideCredentialMaterial(t *testing.T) {
	dash := newFakeDash()
	seen := fixedNow
	dash.devices = []store.Device{
		{ID: "client-1", Name: "laptop", TokenHash: []byte("secret-hash"), CreatedAt: fixedNow, LastSeenAt: &seen},
		{ID: "client-2", Name: "pending one", TokenHash: []byte("other"), CreatedAt: fixedNow},
	}

	h := dashHandler(t, dash)
	cookie := dashLogin(t, h)

	rec := withCookie(t, h, http.MethodGet, "/dash/devices", cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if strings.Contains(body, "secret-hash") || strings.Contains(body, "tokenHash") {
		t.Errorf("the device list exposes credential material: %s", body)
	}

	var parsed struct {
		Devices []struct {
			ID    string `json:"id"`
			State string `json:"state"`
		} `json:"devices"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(parsed.Devices) != 2 {
		t.Fatalf("devices = %+v", parsed.Devices)
	}
	if parsed.Devices[0].State != "active" || parsed.Devices[1].State != "pending" {
		t.Errorf("states = %+v", parsed.Devices)
	}
}

func TestDashRowsAndCommitsAreScopedToTheSession(t *testing.T) {
	dash := newFakeDash()
	dash.snapshotResult = store.SnapshotResult{
		Rows: []store.Row{{
			Table: "schedule", RowID: "row-1", OwnerID: "acct-1", Version: 1,
			UpdatedAtMs: 1_700_000_000_000, Payload: json.RawMessage(`{"title":"standup"}`),
		}},
		ServerSeq: 1,
	}
	dash.pullResult = store.PullResult{
		Changes:   []store.Change{{ServerSeq: 1, Table: "schedule", RowID: "row-1", Op: store.OpUpsert}},
		ServerSeq: 1,
	}

	h := dashHandler(t, dash)
	cookie := dashLogin(t, h)

	if rec := withCookie(t, h, http.MethodGet, "/dash/rows", cookie); rec.Code != http.StatusOK {
		t.Fatalf("rows status = %d (%s)", rec.Code, rec.Body.String())
	}
	// Visibility comes from the session, exactly as it comes from the token on
	// the sync endpoints: a dashboard shows your own data, not everyone's.
	if dash.gotSnapshot.AccountID != "acct-1" || len(dash.gotSnapshot.TeamIDs) != 1 {
		t.Errorf("snapshot query = %+v", dash.gotSnapshot)
	}

	if rec := withCookie(t, h, http.MethodGet, "/dash/commits", cookie); rec.Code != http.StatusOK {
		t.Fatalf("commits status = %d", rec.Code)
	}
	if dash.gotPull.AccountID != "acct-1" {
		t.Errorf("pull query = %+v", dash.gotPull)
	}
}

// A build without a dashboard has no session surface at all.
func TestDashRoutesAbsentWithoutADashStore(t *testing.T) {
	h := testHandler(t, func(o *Options) { o.Auth = &fakeAuth{} })

	rec := postJSON(t, h, "/dash/login", `{"username":"alice","password":"x"}`, "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 (the route should not exist)", rec.Code)
	}
}
