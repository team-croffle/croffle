package metrics

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMetricsEndpointAndMethodLabel(t *testing.T) {
	m := New()
	m.ObserveRequest("POST /sync/push", "POST", 200, time.Millisecond)
	m.ObserveRequest("other", "MADE-UP-METHOD", 404, time.Millisecond)
	m.AuthAttempt("sync", "ok")

	srv := httptest.NewServer(m.Handler())
	defer srv.Close()
	res, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /metrics = %d", res.StatusCode)
	}
	text := string(body)
	for _, want := range []string{
		`croffle_http_requests_total{code="200",method="POST",route="POST /sync/push"} 1`,
		`croffle_http_requests_total{code="404",method="other",route="other"} 1`,
		`croffle_auth_attempts_total{result="ok",surface="sync"} 1`,
		`croffle_sync_push_commits_total{result="retry"} 0`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %s", want)
		}
	}
	if strings.Contains(text, "MADE-UP-METHOD") {
		t.Error("an unknown method became a label")
	}

	for _, path := range []string{"/", "/health", "/metrics/x"} {
		res, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		_ = res.Body.Close()
		if res.StatusCode != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404", path, res.StatusCode)
		}
	}
}
