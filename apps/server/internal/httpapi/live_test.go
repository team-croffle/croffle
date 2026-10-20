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
	"strconv"
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
		Sync:     db,
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

	// Still to come in its own work item, but the credential already carries
	// through to it.
	if res, _ := srv.post(t, "/sync/bootstrap", nil, token); res.StatusCode != http.StatusNotImplemented {
		t.Errorf("authenticated bootstrap status = %d, want 501", res.StatusCode)
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

// --- push (work item #5) ---

func livePush(t *testing.T, srv *liveServer, token string, commits ...protocol.Commit) protocol.PushResponse {
	t.Helper()
	res, body := srv.post(t, "/sync/push", protocol.PushRequest{ClientID: testClientID, Commits: commits}, token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("push status = %d (%s)", res.StatusCode, body)
	}

	var parsed protocol.PushResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("decode push: %v", err)
	}
	return parsed
}

func liveCommit(opID, rowID string, base, version int64) protocol.Commit {
	return protocol.Commit{
		OpID: opID, Table: "schedule", RowID: rowID, Op: protocol.OpUpsert,
		BaseVersion: base, Version: version, UpdatedAtMs: 1_700_000_000_000 + version,
		LastWriterID: "device:" + testClientID,
		Row: protocol.Row{
			"id": rowID, "title": "standup", "isAllDay": float64(0),
			"version": float64(version), "updatedAtMs": float64(1_700_000_000_000 + version),
			"ownerId": "acct-1", "teamId": nil, "deletedAt": nil,
		},
	}
}

func TestLivePushStoresRowsAndIsIdempotent(t *testing.T) {
	srv := startServer(t)
	token := srv.login(t)

	first := livePush(t, srv, token, liveCommit("op-1", "row-1", 0, 1), liveCommit("op-2", "row-2", 0, 1))
	if len(first.Accepted) != 2 || len(first.Conflicts) != 0 {
		t.Fatalf("response = %+v, want two accepted", first)
	}
	if first.ServerSeq != 2 {
		t.Errorf("serverSeq = %d, want 2", first.ServerSeq)
	}

	// Retrying the same batch, as a client does when a response is lost.
	retry := livePush(t, srv, token, liveCommit("op-1", "row-1", 0, 1), liveCommit("op-2", "row-2", 0, 1))
	if len(retry.Accepted) != 2 {
		t.Errorf("retry accepted = %v, want both", retry.Accepted)
	}
	if retry.ServerSeq != 2 {
		t.Errorf("retry serverSeq = %d, want 2 (no new commits)", retry.ServerSeq)
	}

	var commits int
	if err := queryRow(t, srv.path, "SELECT COUNT(*) FROM commit_log").Scan(&commits); err != nil {
		t.Fatalf("count commits: %v", err)
	}
	if commits != 2 {
		t.Errorf("commit log entries = %d, want 2", commits)
	}
}

func TestLivePushReturnsConflictsWithTheServerRow(t *testing.T) {
	srv := startServer(t)
	token := srv.login(t)

	livePush(t, srv, token, liveCommit("op-1", "row-1", 0, 1))

	// A second device that still believes the row is new.
	stale := livePush(t, srv, token, liveCommit("op-2", "row-1", 0, 2))
	if len(stale.Accepted) != 0 || len(stale.Conflicts) != 1 {
		t.Fatalf("response = %+v, want one conflict", stale)
	}

	conflict := stale.Conflicts[0]
	if conflict.ServerVersion != 1 {
		t.Errorf("serverVersion = %d, want 1", conflict.ServerVersion)
	}
	if conflict.ServerRow["title"] != "standup" {
		t.Errorf("serverRow = %v, want the stored payload", conflict.ServerRow)
	}

	// Settling it the way the desktop would: adopt the server's version as the
	// base and push again with a higher version.
	settled := livePush(t, srv, token, liveCommit("op-3", "row-1", conflict.ServerVersion, conflict.ServerVersion+1))
	if len(settled.Accepted) != 1 {
		t.Fatalf("settled = %+v, want it accepted", settled)
	}
}

func TestLivePushRefusesAForeignTeam(t *testing.T) {
	srv := startServer(t)
	token := srv.login(t)

	commit := liveCommit("op-1", "row-1", 0, 1)
	commit.Row["teamId"] = "team-not-mine"

	res, body := srv.post(t, "/sync/push",
		protocol.PushRequest{ClientID: testClientID, Commits: []protocol.Commit{commit}}, token)
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (%s)", res.StatusCode, body)
	}
	if !strings.Contains(string(body), "foreign_team") {
		t.Errorf("body = %s, want the foreign_team code", body)
	}
}

func TestLivePushRequiresAToken(t *testing.T) {
	srv := startServer(t)

	res, _ := srv.post(t, "/sync/push",
		protocol.PushRequest{ClientID: testClientID, Commits: []protocol.Commit{liveCommit("op-1", "row-1", 0, 1)}}, "")
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", res.StatusCode)
	}
}

func TestLivePushRowIsOwnedByTheTokensAccount(t *testing.T) {
	srv := startServer(t)
	token := srv.login(t)

	// The body claims someone else owns the row; the server must ignore that.
	commit := liveCommit("op-1", "row-1", 0, 1)
	commit.Row["ownerId"] = "acct-somebody-else"
	livePush(t, srv, token, commit)

	var owner string
	if err := queryRow(t, srv.path, "SELECT ownerId FROM sync_row WHERE rowKey = 'row-1'").Scan(&owner); err != nil {
		t.Fatalf("read owner: %v", err)
	}
	if owner != "acct-1" {
		t.Errorf("ownerId = %q, want the authenticated account", owner)
	}
}

// --- pull (work item #6) ---

func livePull(t *testing.T, srv *liveServer, token string, since int64, limit int) protocol.PullResponse {
	t.Helper()
	res, body := srv.post(t, "/sync/pull", protocol.PullRequest{LastServerSeq: since, Limit: limit}, token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("pull status = %d (%s)", res.StatusCode, body)
	}

	var parsed protocol.PullResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("decode pull: %v", err)
	}
	return parsed
}

func TestLivePushThenPullRoundTripsTheRow(t *testing.T) {
	srv := startServer(t)
	token := srv.login(t)

	sent := liveCommit("op-1", "row-1", 0, 1)
	livePush(t, srv, token, sent)

	pulled := livePull(t, srv, token, 0, 0)
	if len(pulled.Changes) != 1 {
		t.Fatalf("changes = %+v, want 1", pulled.Changes)
	}

	change := pulled.Changes[0]
	if change.ServerSeq != 1 || change.Table != "schedule" || change.RowID != "row-1" {
		t.Errorf("change = %+v", change)
	}

	// What comes back has to be the row that went in: the opaque payload plus
	// the promoted columns, with ownership as the server resolved it.
	for _, key := range []string{"id", "title", "isAllDay", "version", "updatedAtMs", "ownerId", "teamId", "deletedAt"} {
		if _, ok := change.Row[key]; !ok {
			t.Errorf("row is missing %q: %v", key, change.Row)
		}
	}
	if change.Row["title"] != sent.Row["title"] {
		t.Errorf("title = %v, want %v", change.Row["title"], sent.Row["title"])
	}
	if change.Row["ownerId"] != "acct-1" {
		t.Errorf("ownerId = %v, want the authenticated account", change.Row["ownerId"])
	}

	// The cursor is now current, so there is nothing left.
	again := livePull(t, srv, token, pulled.ServerSeq, 0)
	if len(again.Changes) != 0 {
		t.Errorf("changes after the cursor = %+v, want none", again.Changes)
	}
}

func TestLivePullPagesWithHasMore(t *testing.T) {
	srv := startServer(t)
	token := srv.login(t)

	commits := make([]protocol.Commit, 0, 3)
	for i := range 3 {
		id := strconv.Itoa(i)
		commits = append(commits, liveCommit("op-"+id, "row-"+id, 0, 1))
	}
	livePush(t, srv, token, commits...)

	first := livePull(t, srv, token, 0, 2)
	if len(first.Changes) != 2 || !first.HasMore || first.ServerSeq != 2 {
		t.Fatalf("first page = %+v", first)
	}

	second := livePull(t, srv, token, first.ServerSeq, 2)
	if len(second.Changes) != 1 || second.HasMore || second.ServerSeq != 3 {
		t.Fatalf("second page = %+v", second)
	}
}

func TestLivePullRejectsAnExpiredCursorWithGone(t *testing.T) {
	srv := startServer(t)
	token := srv.login(t)

	livePush(t, srv, token, liveCommit("op-1", "row-1", 0, 1))
	// Pretend retention swept past this client (work item #8 does this for
	// real).
	exec(t, srv.path, "UPDATE sync_meta SET value = 9 WHERE key = 'purgedThroughSeq'")

	res, body := srv.post(t, "/sync/pull", protocol.PullRequest{LastServerSeq: 1}, token)
	if res.StatusCode != http.StatusGone {
		t.Fatalf("status = %d, want 410 (%s)", res.StatusCode, body)
	}
	if !strings.Contains(string(body), "cursor_too_old") {
		t.Errorf("body = %s, want the cursor_too_old code", body)
	}
}

func TestLivePullIsScopedToTheAccount(t *testing.T) {
	srv := startServer(t)
	token := srv.login(t)
	livePush(t, srv, token, liveCommit("op-1", "row-1", 0, 1))

	// A second account on the same server must not see the first one's rows.
	ctx := context.Background()
	hash, err := auth.HashPassword("other-secret")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if err := srv.db.CreateAccount(ctx, store.Account{
		ID: "acct-2", Username: "bob", PasswordHash: hash, CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("create bob: %v", err)
	}

	res, body := srv.post(t, "/auth/login", protocol.LoginRequest{
		Username: "bob", Password: "other-secret", ClientID: "client-2",
	}, "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("bob's login = %d (%s)", res.StatusCode, body)
	}
	var login protocol.LoginResponse
	if err := json.Unmarshal(body, &login); err != nil {
		t.Fatalf("decode: %v", err)
	}

	pulled := livePull(t, srv, login.Token, 0, 0)
	if len(pulled.Changes) != 0 {
		t.Errorf("bob pulled %+v, want nothing", pulled.Changes)
	}
}
