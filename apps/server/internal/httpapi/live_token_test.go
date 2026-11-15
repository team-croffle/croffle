package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/team-croffle/croffle/apps/server/internal/protocol"
)

func (s *liveServer) tokenPair(t *testing.T, req protocol.TokenRequest) (int, protocol.TokenResponse, string) {
	t.Helper()
	res, body := s.post(t, "/auth/token", req, "")
	var parsed protocol.TokenResponse
	if res.StatusCode == http.StatusOK {
		if err := json.Unmarshal(body, &parsed); err != nil {
			t.Fatalf("decode token response: %v", err)
		}
	}
	return res.StatusCode, parsed, string(body)
}

func passwordGrant() protocol.TokenRequest {
	return protocol.TokenRequest{
		GrantType: protocol.GrantPassword, Username: testUsername, Password: testPassword,
		ClientID: testClientID, DeviceName: "laptop",
	}
}

func TestAuthTokenCoexistsWithTheDeviceToken(t *testing.T) {
	srv := startServer(t)
	deviceToken := srv.login(t)

	status, pair, body := srv.tokenPair(t, passwordGrant())
	if status != http.StatusOK {
		t.Fatalf("password grant: %d (%s)", status, body)
	}
	for _, field := range []string{"accessToken", "accessExpiresAt", "refreshToken", "refreshExpiresAt", "accountId", "teamId"} {
		if !strings.Contains(body, `"`+field+`"`) {
			t.Errorf("response lacks %s: %s", field, body)
		}
	}
	if pair.AccountID != "acct-1" || pair.AccessExpiresAt >= pair.RefreshExpiresAt {
		t.Errorf("pair = %+v", pair)
	}

	for name, token := range map[string]string{"device": deviceToken, "access": pair.AccessToken} {
		for _, path := range []string{"/auth/whoami", "/sync/pull", "/sync/push"} {
			if res, b := srv.post(t, path, nil, token); res.StatusCode == http.StatusUnauthorized {
				t.Errorf("%s token refused on %s: %s", name, path, b)
			}
		}
	}
}

func TestRejectRefreshTokenOnSync(t *testing.T) {
	srv := startServer(t)
	_, pair, _ := srv.tokenPair(t, passwordGrant())

	for _, path := range []string{"/auth/whoami", "/sync/push", "/sync/pull", "/sync/bootstrap"} {
		res, body := srv.post(t, path, nil, pair.RefreshToken)
		if res.StatusCode != http.StatusUnauthorized || !strings.Contains(string(body), `"unauthorized"`) {
			t.Errorf("refresh token on %s: %d (%s), want 401 unauthorized", path, res.StatusCode, body)
		}
	}
}

func TestRefreshReuseRevokesFamilyLive(t *testing.T) {
	srv := startServer(t)
	_, first, _ := srv.tokenPair(t, passwordGrant())

	status, second, body := srv.tokenPair(t, protocol.TokenRequest{
		GrantType: protocol.GrantRefreshToken, RefreshToken: first.RefreshToken,
	})
	if status != http.StatusOK {
		t.Fatalf("rotation: %d (%s)", status, body)
	}
	if second.AccessToken == first.AccessToken || second.RefreshToken == first.RefreshToken {
		t.Fatal("rotation returned the same tokens")
	}
	if res, _ := srv.post(t, "/auth/whoami", nil, second.AccessToken); res.StatusCode != http.StatusOK {
		t.Fatalf("rotated access token: %d", res.StatusCode)
	}

	status, _, body = srv.tokenPair(t, protocol.TokenRequest{
		GrantType: protocol.GrantRefreshToken, RefreshToken: first.RefreshToken,
	})
	if status != http.StatusUnauthorized || !strings.Contains(body, `"unauthorized"`) {
		t.Fatalf("replay: %d (%s), want 401 unauthorized", status, body)
	}
	for _, path := range []string{"/auth/whoami", "/sync/pull"} {
		if res, _ := srv.post(t, path, nil, second.AccessToken); res.StatusCode != http.StatusUnauthorized {
			t.Errorf("access token of the revoked family on %s: %d", path, res.StatusCode)
		}
	}
	if status, _, _ := srv.tokenPair(t, protocol.TokenRequest{
		GrantType: protocol.GrantRefreshToken, RefreshToken: second.RefreshToken,
	}); status != http.StatusUnauthorized {
		t.Errorf("refresh token of the revoked family: %d", status)
	}
}

func TestAuthTokenRejectsBadRequests(t *testing.T) {
	srv := startServer(t)
	cases := map[string]protocol.TokenRequest{
		"unknown grant":     {GrantType: "client_credentials"},
		"missing client id": {GrantType: protocol.GrantPassword, Username: testUsername, Password: testPassword},
	}
	for name, req := range cases {
		if status, _, body := srv.tokenPair(t, req); status != http.StatusBadRequest {
			t.Errorf("%s: %d (%s), want 400", name, status, body)
		}
	}
	wrong := passwordGrant()
	wrong.Password = "nope"
	if status, _, _ := srv.tokenPair(t, wrong); status != http.StatusUnauthorized {
		t.Errorf("wrong password: %d", status)
	}
}

func TestRevokeDeviceStopsAllTokens(t *testing.T) {
	srv := startServer(t)
	deviceToken := srv.login(t)
	_, pair, _ := srv.tokenPair(t, passwordGrant())

	if err := srv.db.RevokeDevice(context.Background(), testClientID); err != nil {
		t.Fatal(err)
	}
	for name, token := range map[string]string{"device": deviceToken, "access": pair.AccessToken} {
		if res, _ := srv.post(t, "/sync/pull", nil, token); res.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s token after revoke: %d", name, res.StatusCode)
		}
	}
	if status, _, _ := srv.tokenPair(t, protocol.TokenRequest{
		GrantType: protocol.GrantRefreshToken, RefreshToken: pair.RefreshToken,
	}); status != http.StatusUnauthorized {
		t.Errorf("refresh after revoke: %d", status)
	}
}

func TestRevokeDeviceStopsAllTokensFromTheDashboard(t *testing.T) {
	srv := startServer(t)
	deviceToken := srv.login(t)
	_, pair, _ := srv.tokenPair(t, passwordGrant())
	cookie := srv.dashLogin(t)

	req, err := http.NewRequest(http.MethodPost, srv.url+"/dash/devices/"+testClientID+"/revoke", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.AddCookie(cookie)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("dashboard revoke: %d", res.StatusCode)
	}

	for name, token := range map[string]string{"device": deviceToken, "access": pair.AccessToken} {
		if res, _ := srv.post(t, "/sync/pull", nil, token); res.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s token after dashboard revoke: %d", name, res.StatusCode)
		}
	}
	if status, _, _ := srv.tokenPair(t, protocol.TokenRequest{
		GrantType: protocol.GrantRefreshToken, RefreshToken: pair.RefreshToken,
	}); status != http.StatusUnauthorized {
		t.Errorf("refresh after dashboard revoke: %d", status)
	}
}

func TestConflictDifferentFields(t *testing.T) {
	srv := startServer(t)
	token := srv.login(t)

	livePush(t, srv, token, liveCommit("op-1", "row-1", 0, 1))
	moved := liveCommit("op-2", "row-1", 1, 2)
	moved.Row["title"] = "moved"
	livePush(t, srv, token, moved)

	stale := liveCommit("op-3", "row-1", 1, 3)
	stale.Row["title"] = "mine"
	stale.Row["location"] = "room 1"
	// A row-less delete is allowed by protocol 1; it compares only deletedAt.
	rowless := protocol.Commit{
		OpID: "op-4", Table: "schedule", RowID: "row-1", Op: protocol.OpDelete,
		BaseVersion: 1, Version: 3, UpdatedAtMs: 1_700_000_000_009, LastWriterID: "device:" + testClientID,
	}
	res, body := srv.post(t, "/sync/push", protocol.PushRequest{ClientID: testClientID,
		Commits: []protocol.Commit{stale, rowless}}, token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("push: %d %s", res.StatusCode, body)
	}
	// Raw JSON: the field must be present (an empty list is "[]", never absent).
	if strings.Count(string(body), `"differentFields":`) != 2 {
		t.Fatalf("differentFields missing: %s", body)
	}
	var parsed protocol.PushResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatal(err)
	}
	if len(parsed.Conflicts) != 2 {
		t.Fatalf("conflicts = %+v", parsed.Conflicts)
	}
	if got := strings.Join(parsed.Conflicts[0].DifferentFields, ","); got != "location,title" {
		t.Errorf("stale upsert differs in %q, want location,title", got)
	}
	if got := strings.Join(parsed.Conflicts[1].DifferentFields, ","); got != "deletedAt" {
		t.Errorf("row-less delete differs in %q, want deletedAt", got)
	}

	// Nothing was applied: the server row is still the second write.
	var title string
	if err := queryRow(t, srv.path, "SELECT json_extract(payload, '$.title') FROM sync_row WHERE rowKey = 'row-1'").Scan(&title); err != nil {
		t.Fatal(err)
	}
	if title != "moved" {
		t.Errorf("server row title = %q after conflicts, want moved", title)
	}
}

func TestLoginRateLimitPerUsername(t *testing.T) {
	srv := startServer(t)
	wrong := protocol.LoginRequest{Username: testUsername, Password: "nope", ClientID: testClientID}

	for i := 1; i <= 5; i++ {
		res, body := srv.post(t, "/auth/login", wrong, "")
		if res.StatusCode != http.StatusUnauthorized {
			t.Fatalf("attempt %d: %d %s, want 401 inside the burst", i, res.StatusCode, body)
		}
	}
	res, body := srv.post(t, "/auth/login", wrong, "")
	if res.StatusCode != http.StatusTooManyRequests || !strings.Contains(string(body), `"too_many_attempts"`) {
		t.Fatalf("sixth attempt: %d %s, want 429 too_many_attempts", res.StatusCode, body)
	}
	if got := res.Header.Get("Retry-After"); got != "30" {
		t.Errorf("Retry-After = %q, want 30", got)
	}
	// Another username from the same address has its own bucket.
	other := protocol.LoginRequest{Username: "bob", Password: "nope", ClientID: testClientID}
	if res, _ := srv.post(t, "/auth/login", other, ""); res.StatusCode != http.StatusUnauthorized {
		t.Errorf("another username: %d, want 401", res.StatusCode)
	}
}
