package httpapi_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/team-croffle/croffle/apps/server/internal/auth"
	"github.com/team-croffle/croffle/apps/server/internal/httpapi"
	"github.com/team-croffle/croffle/apps/server/internal/protocol"
	"github.com/team-croffle/croffle/apps/server/internal/store"
)

// lockedBuffer collects the server's JSON log while requests run.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

type adminServer struct {
	*liveServer
	log *lockedBuffer
}

// startAdminServer is the live stack with the operator API, alice as an
// operator and bob as a regular account.
func startAdminServer(t *testing.T) *adminServer {
	t.Helper()
	log := &lockedBuffer{}
	srv := startServerWith(t, func(o *httpapi.Options) {
		o.Admin = o.Sync.(httpapi.Admin)
		o.Logger = slog.New(slog.NewJSONHandler(log, &slog.HandlerOptions{Level: slog.LevelInfo}))
	})
	ctx := context.Background()
	if err := srv.db.SetOperator(ctx, "acct-1", true, false); err != nil {
		t.Fatal(err)
	}
	hash, _ := auth.HashPassword("bob-pass")
	if err := srv.db.CreateAccount(ctx, store.Account{ID: "acct-2", Username: "bob", PasswordHash: hash, CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	return &adminServer{liveServer: srv, log: log}
}

func (s *liveServer) do(t *testing.T, method, path string, body any, cookie *http.Cookie, headers map[string]string) (int, []byte) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(context.Background(), method, s.url+path, reader)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	data, _ := io.ReadAll(res.Body)
	return res.StatusCode, data
}

func (s *liveServer) dashLoginAs(t *testing.T, username, password string) *http.Cookie {
	t.Helper()
	res, body := s.postWithCookie(t, "/dash/login", map[string]string{"username": username, "password": password}, nil, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("dashboard login %s = %d %s", username, res.StatusCode, body)
	}
	for _, c := range res.Cookies() {
		if c.Name == "croffle_dash" {
			return c
		}
	}
	t.Fatal("no session cookie")
	return nil
}

func wantStatus(t *testing.T, what string, got, want int, body []byte) {
	t.Helper()
	if got != want {
		t.Fatalf("%s: status %d, want %d (%s)", what, got, want, body)
	}
}

func TestRequireOperatorRefusesRegularAccounts(t *testing.T) {
	s := startAdminServer(t)
	bob := s.dashLoginAs(t, "bob", "bob-pass")
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/dash/admin/accounts"},
		{http.MethodPost, "/dash/admin/accounts"},
		{http.MethodGet, "/dash/admin/devices"},
		{http.MethodGet, "/dash/admin/teams"},
		{http.MethodGet, "/dash/admin/info"},
		{http.MethodPost, "/dash/admin/vacuum"},
		{http.MethodGet, "/dash/admin/commits"},
		{http.MethodDelete, "/dash/admin/teams/team-default/members/acct-1"},
	} {
		var body any
		if c.method != http.MethodGet {
			body = map[string]string{}
		}
		code, data := s.do(t, c.method, c.path, body, bob, nil)
		if code != http.StatusForbidden || errorCodeOf(data) != "not_operator" {
			t.Errorf("%s %s as bob = %d %s, want 403 not_operator", c.method, c.path, code, data)
		}
	}
	// Without a session it is the session check that answers.
	if code, _ := s.do(t, http.MethodGet, "/dash/admin/accounts", nil, nil, nil); code != http.StatusUnauthorized {
		t.Errorf("no session = %d, want 401", code)
	}
	// Revoking takes effect on the next request, not at the next sign-in.
	alice := s.dashLoginAs(t, testUsername, testPassword)
	if err := s.db.SetOperator(context.Background(), "acct-2", true, false); err != nil {
		t.Fatal(err)
	}
	if err := s.db.SetOperator(context.Background(), "acct-1", false, false); err != nil {
		t.Fatal(err)
	}
	if code, _ := s.do(t, http.MethodGet, "/dash/admin/accounts", nil, alice, nil); code != http.StatusForbidden {
		t.Errorf("after revoke = %d, want 403", code)
	}
}

func TestDashAdminAccounts(t *testing.T) {
	s := startAdminServer(t)
	alice := s.dashLoginAs(t, testUsername, testPassword)

	code, body := s.do(t, http.MethodPost, "/dash/admin/accounts",
		map[string]any{"username": "carol", "password": "carol-pass-1"}, alice, nil)
	wantStatus(t, "create carol", code, http.StatusCreated, body)
	var created struct {
		Account struct {
			ID string `json:"id"`
		} `json:"account"`
	}
	_ = json.Unmarshal(body, &created)
	carolID := created.Account.ID
	if teams, _ := s.db.TeamsOf(context.Background(), carolID); len(teams) != 1 || teams[0] != store.DefaultTeamID {
		t.Fatalf("carol's teams = %v, want team-default", teams)
	}
	code, body = s.do(t, http.MethodPost, "/dash/admin/accounts", map[string]any{"username": "carol", "password": "x"}, alice, nil)
	wantStatus(t, "duplicate", code, http.StatusConflict, body)
	code, body = s.do(t, http.MethodPost, "/dash/admin/accounts", map[string]any{"username": "dave", "password": ""}, alice, nil)
	wantStatus(t, "empty password", code, http.StatusBadRequest, body)

	// The list.
	code, body = s.do(t, http.MethodGet, "/dash/admin/accounts", nil, alice, nil)
	wantStatus(t, "list", code, http.StatusOK, body)
	var list struct {
		Accounts []struct {
			Username    string `json:"username"`
			Operator    bool   `json:"operator"`
			OTP         string `json:"otp"`
			DeviceCount int    `json:"deviceCount"`
		} `json:"accounts"`
	}
	if err := json.Unmarshal(body, &list); err != nil || len(list.Accounts) != 3 {
		t.Fatalf("list = %s", body)
	}
	if a := list.Accounts[0]; a.Username != testUsername || !a.Operator || a.OTP != "off" {
		t.Fatalf("alice in the list = %+v", a)
	}

	// Password reset signs carol's browser out; her device token stays.
	carolCookie := s.dashLoginAs(t, "carol", "carol-pass-1")
	res, loginBody := s.post(t, "/auth/login", protocol.LoginRequest{
		Username: "carol", Password: "carol-pass-1", ClientID: "carol-laptop", DeviceName: "laptop",
	}, "")
	wantStatus(t, "carol device login", res.StatusCode, http.StatusOK, loginBody)
	var login protocol.LoginResponse
	_ = json.Unmarshal(loginBody, &login)
	code, body = s.do(t, http.MethodPost, "/dash/admin/accounts/"+carolID+"/password", map[string]any{"password": "carol-pass-2"}, alice, nil)
	wantStatus(t, "password", code, http.StatusNoContent, body)
	if code, _ := s.do(t, http.MethodGet, "/dash/me", nil, carolCookie, nil); code != http.StatusUnauthorized {
		t.Fatalf("carol's session after the reset = %d, want 401", code)
	}
	if res, body := s.post(t, "/auth/whoami", nil, login.Token); res.StatusCode != http.StatusOK {
		t.Fatalf("carol's device token after the reset = %d %s", res.StatusCode, body)
	}

	// Disable stops the token; enable lets it through again.
	code, body = s.do(t, http.MethodPost, "/dash/admin/accounts/"+carolID+"/disable", map[string]any{}, alice, nil)
	wantStatus(t, "disable", code, http.StatusNoContent, body)
	if res, _ := s.post(t, "/auth/whoami", nil, login.Token); res.StatusCode == http.StatusOK {
		t.Fatal("a disabled account's token still works")
	}
	code, body = s.do(t, http.MethodPost, "/dash/admin/accounts/"+carolID+"/enable", map[string]any{}, alice, nil)
	wantStatus(t, "enable", code, http.StatusNoContent, body)

	// Operator flag and the last-operator rule.
	code, body = s.do(t, http.MethodPost, "/dash/admin/accounts/acct-1/operator", map[string]any{"operator": false}, alice, nil)
	if code != http.StatusConflict || errorCodeOf(body) != "last_operator" {
		t.Fatalf("revoke the last operator = %d %s", code, body)
	}
	code, body = s.do(t, http.MethodPost, "/dash/admin/accounts/acct-1/disable", map[string]any{}, alice, nil)
	if code != http.StatusConflict || errorCodeOf(body) != "last_operator" {
		t.Fatalf("disable the last operator = %d %s", code, body)
	}
	code, body = s.do(t, http.MethodPost, "/dash/admin/accounts/"+carolID+"/operator", map[string]any{"operator": true}, alice, nil)
	wantStatus(t, "grant carol", code, http.StatusNoContent, body)
	code, body = s.do(t, http.MethodPost, "/dash/admin/accounts/"+carolID+"/operator", map[string]any{}, alice, nil)
	wantStatus(t, "operator without a value", code, http.StatusBadRequest, body)
	code, body = s.do(t, http.MethodPost, "/dash/admin/accounts/acct-missing/disable", map[string]any{}, alice, nil)
	wantStatus(t, "missing account", code, http.StatusNotFound, body)

	// OTP off by the operator.
	if err := s.db.StartOTP(context.Background(), carolID, exampleSecret(), time.Now()); err != nil {
		t.Fatal(err)
	}
	code, body = s.do(t, http.MethodPost, "/dash/admin/accounts/"+carolID+"/otp/disable", map[string]any{}, alice, nil)
	wantStatus(t, "otp disable", code, http.StatusNoContent, body)
	if state, _ := s.db.OTPStatus(context.Background(), carolID); state != "off" {
		t.Fatalf("otp after disable = %s", state)
	}

	// The audit log names actions and never a secret.
	logged := s.log.String()
	for _, action := range []string{`"account create"`, `"account password"`, `"account disable"`, `"operator grant"`, `"account otp disable"`} {
		if !strings.Contains(logged, `"msg":"operator action"`) || !strings.Contains(logged, action) {
			t.Errorf("audit log lacks %s", action)
		}
	}
	for _, secret := range []string{"carol-pass-1", "carol-pass-2", login.Token} {
		if strings.Contains(logged, secret) {
			t.Errorf("the log contains a secret: %s", secret)
		}
	}
}

func TestDashAdminDevices(t *testing.T) {
	s := startAdminServer(t)
	alice := s.dashLoginAs(t, testUsername, testPassword)

	code, body := s.do(t, http.MethodPost, "/dash/admin/devices",
		map[string]any{"username": "bob", "name": "issued-by-web", "noExpiry": true}, alice, nil)
	wantStatus(t, "issue", code, http.StatusCreated, body)
	var issued struct {
		DeviceID string `json:"deviceId"`
		Token    string `json:"token"`
	}
	_ = json.Unmarshal(body, &issued)
	if issued.Token == "" || issued.DeviceID == "" {
		t.Fatalf("issue = %s", body)
	}
	if res, _ := s.post(t, "/auth/whoami", nil, issued.Token); res.StatusCode != http.StatusOK {
		t.Fatalf("the issued token = %d", res.StatusCode)
	}

	for _, path := range []string{"/dash/admin/devices", "/dash/admin/devices?account=bob", "/dash/admin/devices?account=acct-2"} {
		code, body = s.do(t, http.MethodGet, path, nil, alice, nil)
		wantStatus(t, path, code, http.StatusOK, body)
		if !strings.Contains(string(body), issued.DeviceID) || strings.Contains(string(body), issued.Token) ||
			strings.Contains(strings.ToLower(string(body)), "tokenhash") {
			t.Fatalf("%s = %s", path, body)
		}
	}
	code, body = s.do(t, http.MethodGet, "/dash/admin/devices?account=nobody", nil, alice, nil)
	wantStatus(t, "unknown account filter", code, http.StatusNotFound, body)

	code, body = s.do(t, http.MethodPost, "/dash/admin/devices", map[string]any{"username": "bob", "expiresIn": "720h"}, alice, nil)
	wantStatus(t, "issue with expiry", code, http.StatusCreated, body)
	if !strings.Contains(string(body), `"tokenExpiresAt":`) || strings.Contains(string(body), `"tokenExpiresAt":null`) {
		t.Fatalf("expiring token = %s", body)
	}
	code, body = s.do(t, http.MethodPost, "/dash/admin/devices", map[string]any{"username": "bob", "expiresIn": "soon"}, alice, nil)
	wantStatus(t, "bad expiry", code, http.StatusBadRequest, body)

	code, body = s.do(t, http.MethodPost, "/dash/admin/devices/"+issued.DeviceID+"/revoke", map[string]any{}, alice, nil)
	wantStatus(t, "revoke", code, http.StatusNoContent, body)
	if res, _ := s.post(t, "/auth/whoami", nil, issued.Token); res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("a revoked token = %d, want 401", res.StatusCode)
	}
	if strings.Contains(s.log.String(), issued.Token) {
		t.Fatal("the log contains the issued token")
	}
}

func TestDashAdminTeams(t *testing.T) {
	s := startAdminServer(t)
	alice := s.dashLoginAs(t, testUsername, testPassword)

	code, body := s.do(t, http.MethodPost, "/dash/admin/teams", map[string]any{"name": "  QA Team "}, alice, nil)
	wantStatus(t, "create", code, http.StatusCreated, body)
	var created struct {
		Team struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"team"`
	}
	_ = json.Unmarshal(body, &created)
	id := created.Team.ID
	if created.Team.Name != "QA Team" || !strings.HasPrefix(id, "team-") {
		t.Fatalf("created = %s", body)
	}
	code, body = s.do(t, http.MethodPost, "/dash/admin/teams/"+id, map[string]any{"name": "QA & Release"}, alice, nil)
	wantStatus(t, "rename", code, http.StatusNoContent, body)
	code, body = s.do(t, http.MethodPost, "/dash/admin/teams/"+id, map[string]any{"name": strings.Repeat("x", 65)}, alice, nil)
	wantStatus(t, "too long", code, http.StatusBadRequest, body)

	code, body = s.do(t, http.MethodPost, "/dash/admin/teams/"+id+"/members", map[string]any{"username": "bob"}, alice, nil)
	wantStatus(t, "add member", code, http.StatusNoContent, body)
	code, body = s.do(t, http.MethodGet, "/dash/admin/teams/"+id+"/members", nil, alice, nil)
	wantStatus(t, "members", code, http.StatusOK, body)
	if !strings.Contains(string(body), `"username":"bob"`) || !strings.Contains(string(body), `"accountId":"acct-2"`) {
		t.Fatalf("members = %s", body)
	}
	code, body = s.do(t, http.MethodGet, "/dash/admin/teams", nil, alice, nil)
	var teams struct {
		Teams []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			MemberCount int    `json:"memberCount"`
		} `json:"teams"`
	}
	if err := json.Unmarshal(body, &teams); err != nil || code != http.StatusOK || len(teams.Teams) != 2 {
		t.Fatalf("teams = %d %s", code, body)
	}
	if got := teams.Teams[1]; got.ID != id || got.Name != "QA & Release" || got.MemberCount != 1 {
		t.Fatalf("the new team in the list = %+v", got)
	}
	code, body = s.do(t, http.MethodDelete, "/dash/admin/teams/"+id+"/members/acct-2", nil, alice, nil)
	wantStatus(t, "remove member", code, http.StatusNoContent, body)
	code, body = s.do(t, http.MethodDelete, "/dash/admin/teams/"+id+"/members/acct-2", nil, alice, nil)
	wantStatus(t, "remove again", code, http.StatusNotFound, body)
	code, body = s.do(t, http.MethodPost, "/dash/admin/teams/team-missing/members", map[string]any{"username": "bob"}, alice, nil)
	wantStatus(t, "missing team", code, http.StatusNotFound, body)
}

func TestDashAdminServer(t *testing.T) {
	s := startAdminServer(t)
	alice := s.dashLoginAs(t, testUsername, testPassword)
	token := s.login(t)
	livePush(t, s.liveServer, token, liveCommit("op-1", "row-1", 0, 1))

	code, body := s.do(t, http.MethodGet, "/dash/admin/info", nil, alice, nil)
	wantStatus(t, "info", code, http.StatusOK, body)
	var info struct {
		SchemaVersion int64  `json:"schemaVersion"`
		SecretKey     string `json:"secretKey"`
		Plain         *int64 `json:"otpSecretsPlain"`
		Encrypted     *int64 `json:"otpSecretsEncrypted"`
	}
	if err := json.Unmarshal(body, &info); err != nil || info.SchemaVersion != 7 || info.SecretKey != "missing" ||
		info.Plain == nil || info.Encrypted == nil {
		t.Fatalf("info = %s", body)
	}
	code, body = s.do(t, http.MethodPost, "/dash/admin/vacuum", map[string]any{}, alice, nil)
	wantStatus(t, "vacuum", code, http.StatusOK, body)

	code, body = s.do(t, http.MethodGet, "/dash/admin/commits?sinceSeq=0&limit=10", nil, alice, nil)
	wantStatus(t, "commits", code, http.StatusOK, body)
	if !strings.Contains(string(body), `"rowId":"row-1"`) || !strings.Contains(string(body), `"ownerId":"acct-1"`) ||
		strings.Contains(string(body), "standup") {
		t.Fatalf("commits = %s (want metadata, no payload)", body)
	}
}

func TestDashAdminRefusesCrossOrigin(t *testing.T) {
	s := startAdminServer(t)
	alice := s.dashLoginAs(t, testUsername, testPassword)
	evil := map[string]string{"Origin": "https://evil.example", "Sec-Fetch-Site": "cross-site"}
	for _, c := range []struct{ method, path string }{
		{http.MethodPost, "/dash/admin/accounts"},
		{http.MethodPost, "/dash/admin/devices"},
		{http.MethodPost, "/dash/admin/vacuum"},
		{http.MethodDelete, "/dash/admin/teams/team-default/members/acct-1"},
	} {
		code, body := s.do(t, c.method, c.path, map[string]any{"username": "x", "password": "y"}, alice, evil)
		if code != http.StatusForbidden || errorCodeOf(body) != "cross_origin" {
			t.Errorf("%s %s cross-origin = %d %s", c.method, c.path, code, body)
		}
	}
	if teams, _ := s.db.TeamsOf(context.Background(), "acct-1"); len(teams) != 1 {
		t.Fatal("a cross-origin request changed a membership")
	}
}

// An operator's devices still see only what any account sees.
func TestOperatorSyncIsUnchanged(t *testing.T) {
	s := startAdminServer(t)
	res, body := s.post(t, "/auth/login", protocol.LoginRequest{
		Username: "bob", Password: "bob-pass", ClientID: "bob-laptop", DeviceName: "laptop",
	}, "")
	wantStatus(t, "bob login", res.StatusCode, http.StatusOK, body)
	var bob protocol.LoginResponse
	_ = json.Unmarshal(body, &bob)
	commit := liveCommit("op-b", "row-bob", 0, 1)
	commit.LastWriterID = "device:bob-laptop"
	commit.Row["ownerId"] = "acct-2"
	livePush(t, s.liveServer, bob.Token, commit)

	alice := s.login(t)
	res, body = s.post(t, "/sync/pull", map[string]any{"lastServerSeq": 0}, alice)
	if res.StatusCode != http.StatusOK || strings.Contains(string(body), "row-bob") {
		t.Fatalf("an operator's pull = %d %s", res.StatusCode, body)
	}
}

// A path with two methods answers 405 for a third and names both.
func TestDashAdminMethodNotAllowedNamesEveryMethod(t *testing.T) {
	s := startAdminServer(t)
	alice := s.dashLoginAs(t, testUsername, testPassword)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPut, s.url+"/dash/admin/accounts", nil)
	req.AddCookie(alice)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	if res.StatusCode != http.StatusMethodNotAllowed || res.Header.Get("Allow") != "GET, POST" {
		t.Fatalf("PUT = %d, Allow %q", res.StatusCode, res.Header.Get("Allow"))
	}
}

// exampleSecret returns a base32 TOTP secret made at run time (knowledge: no
// credential literals in tests).
func exampleSecret() string {
	buf := make([]byte, 10)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf)
}
