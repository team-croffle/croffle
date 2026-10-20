// This file is an external test package on purpose: it wires the real store,
// the real auth service and the real handler together through their public
// APIs, so it checks the assembly rather than a single layer.
package httpapi_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/team-croffle/croffle/apps/server/internal/auth"
	"github.com/team-croffle/croffle/apps/server/internal/httpapi"
	"github.com/team-croffle/croffle/apps/server/internal/protocol"
	"github.com/team-croffle/croffle/apps/server/internal/store"
	"github.com/team-croffle/croffle/apps/server/internal/store/sqlite"
)

const (
	testUsername = "alice"
	testPassword = "hunter2"
	testClientID = "client-1"
)

type liveServer struct {
	url  string
	db   *sqlite.DB
	path string
}

// startServer builds the whole stack over a temporary database, with one
// account that is a member of the seeded team.
func startServer(t *testing.T) *liveServer {
	t.Helper()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "server.db")

	db, err := sqlite.Open(ctx, path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	hash, err := auth.HashPassword(testPassword)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if err := db.CreateAccount(ctx, store.Account{
		ID: "acct-1", Username: testUsername, PasswordHash: hash, CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("create account: %v", err)
	}
	if err := db.AddTeamMember(ctx, store.DefaultTeamID, "acct-1", "member"); err != nil {
		t.Fatalf("add team member: %v", err)
	}

	handler := httpapi.NewHandler(httpapi.Options{
		Version:  "test",
		BasePath: "/",
		Logger:   slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError + 1})),
		Health:   db.Ping,
		Auth:     auth.NewService(db, time.Now),
	})

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	return &liveServer{url: srv.URL, db: db, path: path}
}

func (s *liveServer) post(t *testing.T, path string, body any, bearer string) (*http.Response, []byte) {
	t.Helper()

	var payload []byte
	switch v := body.(type) {
	case nil:
		payload = []byte(`{}`)
	case string:
		payload = []byte(v)
	default:
		var err error
		if payload, err = json.Marshal(v); err != nil {
			t.Fatalf("marshal body: %v", err)
		}
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, s.url+path, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	defer func() { _ = res.Body.Close() }()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return res, data
}

func (s *liveServer) login(t *testing.T) string {
	t.Helper()
	res, body := s.post(t, "/auth/login", protocol.LoginRequest{
		Username: testUsername, Password: testPassword,
		ClientID: testClientID, DeviceName: "laptop",
	}, "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d (%s)", res.StatusCode, body)
	}

	var parsed protocol.LoginResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	if parsed.Token == "" {
		t.Fatal("login returned no token")
	}
	return parsed.Token
}

func TestLiveLoginThenAuthenticatedCall(t *testing.T) {
	srv := startServer(t)
	token := srv.login(t)

	res, body := srv.post(t, "/auth/whoami", nil, token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("whoami status = %d (%s)", res.StatusCode, body)
	}

	var who protocol.WhoAmIResponse
	if err := json.Unmarshal(body, &who); err != nil {
		t.Fatalf("decode whoami: %v", err)
	}
	if who.AccountID != "acct-1" {
		t.Errorf("accountId = %q, want acct-1", who.AccountID)
	}
	if who.TeamID == nil || *who.TeamID != store.DefaultTeamID {
		t.Errorf("teamId = %v, want %s", who.TeamID, store.DefaultTeamID)
	}

	// The sync handlers are not built yet, but the credential already carries
	// through to them.
	res, _ = srv.post(t, "/sync/push", nil, token)
	if res.StatusCode != http.StatusNotImplemented {
		t.Errorf("authenticated push status = %d, want 501", res.StatusCode)
	}
}

func TestLiveWrongPasswordAndUnknownUserAreIndistinguishable(t *testing.T) {
	srv := startServer(t)

	wrongRes, wrongBody := srv.post(t, "/auth/login", protocol.LoginRequest{
		Username: testUsername, Password: "nope", ClientID: testClientID,
	}, "")
	unknownRes, unknownBody := srv.post(t, "/auth/login", protocol.LoginRequest{
		Username: "nobody", Password: "nope", ClientID: testClientID,
	}, "")

	if wrongRes.StatusCode != http.StatusUnauthorized || unknownRes.StatusCode != http.StatusUnauthorized {
		t.Fatalf("statuses = %d and %d, want 401 twice", wrongRes.StatusCode, unknownRes.StatusCode)
	}
	// Same status and same body: the client learns nothing about which
	// usernames exist.
	if string(wrongBody) != string(unknownBody) {
		t.Errorf("responses differ:\n wrong password: %s\n unknown user:  %s", wrongBody, unknownBody)
	}
}

func TestLiveTokenIsStoredHashedOnly(t *testing.T) {
	srv := startServer(t)
	token := srv.login(t)

	// A copy of the database must not be convertible into a working credential.
	raw, err := os.ReadFile(srv.path)
	if err != nil {
		t.Fatalf("read database file: %v", err)
	}
	if bytes.Contains(raw, []byte(token)) {
		t.Error("the database file contains the plaintext token")
	}

	var count int
	if err := srv.db.Ping(context.Background()); err != nil {
		t.Fatalf("ping: %v", err)
	}
	row := queryRow(t, srv.path, "SELECT COUNT(*) FROM device WHERE tokenHash IS NOT NULL")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("scan device count: %v", err)
	}
	if count != 1 {
		t.Errorf("devices with a token = %d, want 1", count)
	}
}

func TestLiveSecondLoginReplacesTheToken(t *testing.T) {
	srv := startServer(t)

	first := srv.login(t)
	second := srv.login(t)
	if first == second {
		t.Fatal("the second login returned the same token")
	}

	// One credential per installation: the replaced token stops working at once.
	if res, _ := srv.post(t, "/auth/whoami", nil, first); res.StatusCode != http.StatusUnauthorized {
		t.Errorf("old token status = %d, want 401", res.StatusCode)
	}
	if res, _ := srv.post(t, "/auth/whoami", nil, second); res.StatusCode != http.StatusOK {
		t.Errorf("new token status = %d, want 200", res.StatusCode)
	}
}

func TestLiveRevokedTokenAndDisabledAccount(t *testing.T) {
	srv := startServer(t)
	token := srv.login(t)

	exec(t, srv.path, "UPDATE device SET revokedAt = 1 WHERE id = ?", testClientID)
	if res, _ := srv.post(t, "/auth/whoami", nil, token); res.StatusCode != http.StatusUnauthorized {
		t.Errorf("revoked token status = %d, want 401", res.StatusCode)
	}

	// Logging in again re-arms the device, because the credential is still
	// valid; disabling the account is what stops it for good.
	token = srv.login(t)
	if res, _ := srv.post(t, "/auth/whoami", nil, token); res.StatusCode != http.StatusOK {
		t.Fatalf("status after re-login = %d, want 200", res.StatusCode)
	}

	exec(t, srv.path, "UPDATE account SET disabledAt = 1 WHERE id = 'acct-1'")
	res, body := srv.post(t, "/auth/whoami", nil, token)
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("disabled account status = %d, want 403 (%s)", res.StatusCode, body)
	}
	if !strings.Contains(string(body), "account_disabled") {
		t.Errorf("body = %s, want the account_disabled code", body)
	}
	if res, _ := srv.post(t, "/auth/login", protocol.LoginRequest{
		Username: testUsername, Password: testPassword, ClientID: testClientID,
	}, ""); res.StatusCode != http.StatusForbidden {
		t.Errorf("login on a disabled account = %d, want 403", res.StatusCode)
	}
}

func TestLiveHealthReportsTheDatabase(t *testing.T) {
	srv := startServer(t)

	res, err := http.Get(srv.url + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer func() { _ = res.Body.Close() }()

	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d (%s)", res.StatusCode, body)
	}
	if !strings.Contains(string(body), `"db":"ok"`) {
		t.Errorf("health body = %s, want db ok", body)
	}
}

// Helpers that talk to the database file directly, to set up states the server
// has no endpoint for (revoking, disabling) until the CLI lands.

func openRaw(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatalf("open raw: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func exec(t *testing.T, path, query string, args ...any) {
	t.Helper()
	if _, err := openRaw(t, path).Exec(query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

func queryRow(t *testing.T, path, query string, args ...any) *sql.Row {
	t.Helper()
	return openRaw(t, path).QueryRow(query, args...)
}
