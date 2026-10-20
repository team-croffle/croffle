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
	"github.com/team-croffle/croffle/apps/server/internal/protocol"
	"github.com/team-croffle/croffle/apps/server/internal/store"
)

// fakeAuth lets the HTTP layer be tested without argon2 or a database.
type fakeAuth struct {
	loginResult auth.LoginResult
	loginErr    error
	identity    auth.Identity
	authErr     error
	gotLogin    auth.LoginRequest
	gotToken    string

	account  *store.Account
	checkErr error
	gotCheck [3]string
}

func (f *fakeAuth) Login(_ context.Context, req auth.LoginRequest) (auth.LoginResult, error) {
	f.gotLogin = req
	return f.loginResult, f.loginErr
}

func (f *fakeAuth) Authenticate(_ context.Context, token string) (auth.Identity, error) {
	f.gotToken = token
	return f.identity, f.authErr
}

func (f *fakeAuth) CheckPassword(_ context.Context, username, password, remoteIP string) (*store.Account, error) {
	f.gotCheck = [3]string{username, password, remoteIP}
	if f.checkErr != nil {
		return nil, f.checkErr
	}
	if f.account != nil {
		return f.account, nil
	}
	return &store.Account{ID: "acct-1", Username: username}, nil
}

func authHandler(t *testing.T, fake *fakeAuth) http.Handler {
	t.Helper()
	return testHandler(t, func(o *Options) { o.Auth = fake })
}

func postJSON(t *testing.T, h http.Handler, path, body, bearer string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body ErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error body %q: %v", rec.Body.String(), err)
	}
	return body.Code
}

func TestLoginReturnsTokenAndTeam(t *testing.T) {
	fake := &fakeAuth{loginResult: auth.LoginResult{
		Token:    "tok-abc",
		Identity: auth.Identity{AccountID: "acct-1", DeviceID: "client-1", TeamIDs: []string{store.DefaultTeamID}},
	}}

	rec := postJSON(t, authHandler(t, fake), "/auth/login",
		`{"username":"alice","password":"hunter2","clientId":"client-1","deviceName":"laptop"}`, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}

	var body protocol.LoginResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Token != "tok-abc" || body.AccountID != "acct-1" {
		t.Errorf("body = %+v", body)
	}
	if body.TeamID == nil || *body.TeamID != store.DefaultTeamID {
		t.Errorf("teamId = %v, want %s", body.TeamID, store.DefaultTeamID)
	}
	// 0.1 never expires tokens: the desktop does not keep the password, so it
	// could not renew one without asking the user again.
	if body.ExpiresAt != nil {
		t.Errorf("expiresAt = %v, want absent", body.ExpiresAt)
	}

	if fake.gotLogin.Username != "alice" || fake.gotLogin.ClientID != "client-1" {
		t.Errorf("service received %+v", fake.gotLogin)
	}
	// The limiter keys on the caller address, so the handler has to pass it on.
	if fake.gotLogin.RemoteIP != "192.0.2.1" {
		t.Errorf("remote ip = %q, want the socket address", fake.gotLogin.RemoteIP)
	}
}

func TestLoginErrorMapping(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		status   int
		code     string
		hasRetry bool
	}{
		{name: "bad credentials", err: auth.ErrInvalidCredentials, status: http.StatusUnauthorized, code: "unauthorized"},
		{name: "disabled", err: auth.ErrAccountDisabled, status: http.StatusForbidden, code: "account_disabled"},
		{name: "otp", err: auth.ErrOTPRequired, status: http.StatusUnauthorized, code: "otp_required"},
		{name: "rate limited", err: auth.ErrRateLimited, status: http.StatusTooManyRequests, code: "too_many_attempts", hasRetry: true},
		{name: "missing client id", err: auth.ErrMissingClientID, status: http.StatusBadRequest, code: "invalid_request"},
		{name: "storage failure", err: errBoom, status: http.StatusInternalServerError, code: "internal"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := postJSON(t, authHandler(t, &fakeAuth{loginErr: tc.err}), "/auth/login",
				`{"username":"alice","password":"x","clientId":"c1"}`, "")

			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d", rec.Code, tc.status)
			}
			if got := errorCode(t, rec); got != tc.code {
				t.Errorf("code = %q, want %q", got, tc.code)
			}
			if tc.hasRetry && rec.Header().Get("Retry-After") == "" {
				t.Error("no Retry-After on a rate-limited response")
			}
			// An internal failure must not describe itself to the caller.
			if tc.code == "internal" && strings.Contains(rec.Body.String(), "boom") {
				t.Error("error body leaked the internal error")
			}
		})
	}
}

func TestLoginRejectsUnknownFields(t *testing.T) {
	rec := postJSON(t, authHandler(t, &fakeAuth{}), "/auth/login",
		`{"username":"alice","password":"x","clientId":"c1","admin":true}`, "")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := errorCode(t, rec); got != "invalid_json" {
		t.Errorf("code = %q, want invalid_json", got)
	}
}

func TestWhoamiRequiresBearerToken(t *testing.T) {
	h := authHandler(t, &fakeAuth{})

	rec := postJSON(t, h, "/auth/whoami", `{}`, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if got := rec.Header().Get("WWW-Authenticate"); !strings.Contains(got, "Bearer") {
		t.Errorf("WWW-Authenticate = %q, want a Bearer challenge", got)
	}
}

func TestWhoamiReturnsIdentity(t *testing.T) {
	fake := &fakeAuth{identity: auth.Identity{
		AccountID: "acct-1", DeviceID: "client-1", TeamIDs: []string{store.DefaultTeamID},
	}}

	rec := postJSON(t, authHandler(t, fake), "/auth/whoami", `{}`, "tok-abc")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}

	var body protocol.WhoAmIResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.AccountID != "acct-1" {
		t.Errorf("accountId = %q", body.AccountID)
	}
	if fake.gotToken != "tok-abc" {
		t.Errorf("service received token %q", fake.gotToken)
	}
}

func TestWhoamiRejectsInvalidToken(t *testing.T) {
	rec := postJSON(t, authHandler(t, &fakeAuth{authErr: auth.ErrInvalidCredentials}), "/auth/whoami", `{}`, "nope")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if got := errorCode(t, rec); got != "unauthorized" {
		t.Errorf("code = %q, want unauthorized", got)
	}
}

// The sync routes have no handler yet, but authentication is already in front
// of them: an unauthenticated call must get the 401 it will always get.
func TestSyncRoutesAreAuthenticatedBeforeTheyExist(t *testing.T) {
	h := authHandler(t, &fakeAuth{identity: auth.Identity{AccountID: "acct-1"}})

	for _, path := range []string{"/sync/push", "/sync/pull", "/sync/bootstrap"} {
		t.Run(path, func(t *testing.T) {
			if rec := postJSON(t, h, path, `{}`, ""); rec.Code != http.StatusUnauthorized {
				t.Errorf("without a token: status = %d, want 401", rec.Code)
			}
			if rec := postJSON(t, h, path, `{}`, "tok"); rec.Code != http.StatusNotImplemented {
				t.Errorf("with a token: status = %d, want 501", rec.Code)
			}
		})
	}
}

// A build without an authenticator answers 501 rather than pretending every
// caller is anonymous.
func TestAuthRoutesWithoutAnAuthenticator(t *testing.T) {
	h := testHandler(t, nil)

	if rec := postJSON(t, h, "/auth/login", `{"username":"a","password":"b","clientId":"c"}`, ""); rec.Code != http.StatusNotImplemented {
		t.Errorf("login status = %d, want 501", rec.Code)
	}
	if rec := postJSON(t, h, "/auth/whoami", `{}`, "tok"); rec.Code != http.StatusNotImplemented {
		t.Errorf("whoami status = %d, want 501", rec.Code)
	}
}

func TestLoginDoesNotLogCredentials(t *testing.T) {
	var sink strings.Builder
	logger := newCapturingLogger(&sink)

	fake := &fakeAuth{loginResult: auth.LoginResult{
		Token:    "tok-secret",
		Identity: auth.Identity{AccountID: "acct-1", DeviceID: "client-1"},
	}}
	h := NewHandler(Options{Version: "1.2.3", BasePath: "/", Logger: logger, Auth: fake,
		Now: func() time.Time { return fixedNow }})

	rec := postJSON(t, h, "/auth/login", `{"username":"alice","password":"hunter2","clientId":"client-1"}`, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	out := sink.String()
	for _, secret := range []string{"hunter2", "tok-secret"} {
		if strings.Contains(out, secret) {
			t.Errorf("log leaked %q: %s", secret, out)
		}
	}
	if !strings.Contains(out, "login succeeded") {
		t.Errorf("no login line in the log: %s", out)
	}
}
