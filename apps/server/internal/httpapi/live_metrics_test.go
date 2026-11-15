package httpapi_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/team-croffle/croffle/apps/server/internal/httpapi"
	"github.com/team-croffle/croffle/apps/server/internal/metrics"
	"github.com/team-croffle/croffle/apps/server/internal/protocol"
)

func scrape(t *testing.T, m *metrics.Metrics) string {
	t.Helper()
	srv := httptest.NewServer(m.Handler())
	defer srv.Close()
	res, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("scrape: %d", res.StatusCode)
	}
	return string(body)
}

func metricValue(t *testing.T, body, series string) string {
	t.Helper()
	re := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(series) + ` (\S+)$`)
	m := re.FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("series %s not found", series)
	}
	return m[1]
}

func TestMetricsCountWhatHappened(t *testing.T) {
	m := metrics.New()
	srv := startServerWith(t, func(o *httpapi.Options) { o.Metrics = m })

	// Sign-ins: one ok, one wrong password.
	token := srv.login(t)
	srv.post(t, "/auth/login", protocol.LoginRequest{Username: testUsername, Password: "nope", ClientID: testClientID}, "")
	// Push: one applied, then the same op again (a retry), then a conflict.
	livePush(t, srv, token, liveCommit("op-1", "row-1", 0, 1))
	livePush(t, srv, token, liveCommit("op-1", "row-1", 0, 1))
	livePush(t, srv, token, liveCommit("op-2", "row-1", 0, 2))
	livePull(t, srv, token, 0, 10)
	liveBootstrap(t, srv, token, "", 10)
	// A path value and an unknown path must not become labels.
	cookie := srv.dashLogin(t)
	srv.postWithCookie(t, "/dash/devices/dev-secret-777/revoke", map[string]string{}, cookie, nil)
	_, _ = http.Get(srv.url + "/no-such-route/alice/acct-1")
	// Any token is a valid method; a made-up one must not become a label.
	if req, err := http.NewRequest("ALICEACCT1", srv.url+"/health", nil); err == nil {
		if res, err := http.DefaultClient.Do(req); err == nil {
			_ = res.Body.Close()
		}
	}

	body := scrape(t, m)
	for series, want := range map[string]string{
		`croffle_auth_attempts_total{result="ok",surface="sync"}`:                       "1",
		`croffle_auth_attempts_total{result="invalid",surface="sync"}`:                  "1",
		`croffle_auth_attempts_total{result="ok",surface="dashboard"}`:                  "1",
		`croffle_sync_push_commits_total{result="accepted"}`:                            "1",
		`croffle_sync_push_commits_total{result="retry"}`:                               "1",
		`croffle_sync_push_commits_total{result="conflict"}`:                            "1",
		`croffle_sync_pull_changes_total`:                                               "1",
		`croffle_sync_bootstrap_rows_total`:                                             "1",
		`croffle_http_requests_total{code="200",method="POST",route="POST /sync/push"}`: "3",
	} {
		if got := metricValue(t, body, series); got != want {
			t.Errorf("%s = %s, want %s", series, got, want)
		}
	}
	for _, want := range []string{
		`route="POST /dash/devices/{id}/revoke"`,
		`route="other"`,
		"croffle_http_request_duration_seconds_bucket",
		"croffle_auth_refresh_reuse_total",
		"go_goroutines",
		"process_",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("metrics lack %s", want)
		}
	}
	if !strings.Contains(body, `method="other"`) {
		t.Error("an unknown method was not folded into other")
	}
	for _, leak := range []string{"acct-1", "alice", "ALICEACCT1", "dev-secret-777", "client-1", "127.0.0.1", "no-such-route"} {
		if strings.Contains(body, leak) {
			t.Errorf("metrics contain %q", leak)
		}
	}
}

func TestMetricsAreNotOnTheAPIListener(t *testing.T) {
	m := metrics.New()
	srv := startServerWith(t, func(o *httpapi.Options) { o.Metrics = m })
	res, err := http.Get(srv.url + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if strings.Contains(string(body), "# HELP croffle_") {
		t.Fatal("the API listener serves metrics")
	}

	// The metrics handler serves /metrics and nothing else.
	metricsSrv := httptest.NewServer(m.Handler())
	defer metricsSrv.Close()
	res, err = http.Get(metricsSrv.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("metrics listener /health: %d, want 404", res.StatusCode)
	}
}

func TestRefreshReuseIsCounted(t *testing.T) {
	m := metrics.New()
	srv := startServerWith(t, func(o *httpapi.Options) { o.Metrics = m })
	_, pair, _ := srv.tokenPair(t, passwordGrant())
	refresh := protocol.TokenRequest{GrantType: protocol.GrantRefreshToken, RefreshToken: pair.RefreshToken}
	srv.tokenPair(t, refresh)
	srv.tokenPair(t, refresh)

	body := scrape(t, m)
	if got := metricValue(t, body, "croffle_auth_refresh_reuse_total"); got != "1" {
		t.Errorf("refresh reuse = %s", got)
	}
	if got := metricValue(t, body, `croffle_auth_attempts_total{result="ok",surface="token"}`); got != "2" {
		t.Errorf("token ok = %s", got)
	}
}
