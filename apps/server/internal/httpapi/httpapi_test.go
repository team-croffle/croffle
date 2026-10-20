package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var fixedNow = time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)

func quietLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError + 1}))
}

func testHandler(t *testing.T, mutate func(*Options)) http.Handler {
	t.Helper()
	o := Options{
		Version:  "1.2.3",
		Commit:   "abc1234",
		BasePath: "/",
		Logger:   quietLogger(),
		Now:      func() time.Time { return fixedNow },
	}
	if mutate != nil {
		mutate(&o)
	}
	return NewHandler(o)
}

func doRequest(t *testing.T, h http.Handler, method, target string, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, target, body))
	return rec
}

func decodeHealth(t *testing.T, rec *httptest.ResponseRecorder) healthResponse {
	t.Helper()
	var got healthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode health body %q: %v", rec.Body.String(), err)
	}
	return got
}

func TestHealthWithoutStore(t *testing.T) {
	rec := doRequest(t, testHandler(t, nil), http.MethodGet, "/health", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want JSON", ct)
	}

	got := decodeHealth(t, rec)
	if got.Server != "1.2.3" || got.Commit != "abc1234" {
		t.Errorf("server/commit = %q/%q", got.Server, got.Commit)
	}
	if got.Protocol.Min != 1 || got.Protocol.Max != 1 {
		t.Errorf("protocol = %+v, want {1 1}", got.Protocol)
	}
	// No store wired yet: that is not an error, so the endpoint stays 200.
	if got.DB != dbStatusUnconfigured {
		t.Errorf("db = %q, want %q", got.DB, dbStatusUnconfigured)
	}
	if got.Time != fixedNow.UnixMilli() {
		t.Errorf("time = %d, want %d", got.Time, fixedNow.UnixMilli())
	}
}

func TestHealthWithHealthyStore(t *testing.T) {
	h := testHandler(t, func(o *Options) {
		o.Health = func(context.Context) error { return nil }
	})

	rec := doRequest(t, h, http.MethodGet, "/health", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := decodeHealth(t, rec); got.DB != dbStatusOK {
		t.Errorf("db = %q, want %q", got.DB, dbStatusOK)
	}
}

func TestHealthWithBrokenStoreIsUnavailable(t *testing.T) {
	h := testHandler(t, func(o *Options) {
		o.Health = func(context.Context) error { return errors.New("disk on fire") }
	})

	rec := doRequest(t, h, http.MethodGet, "/health", nil)
	// 503 so a container HEALTHCHECK and a proxy both notice.
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	if got := decodeHealth(t, rec); got.DB != dbStatusError {
		t.Errorf("db = %q, want %q", got.DB, dbStatusError)
	}
	if strings.Contains(rec.Body.String(), "disk on fire") {
		t.Error("health body leaked the internal error text")
	}
}

// A container HEALTHCHECK may use HEAD; net/http matches it on a GET pattern.
func TestHealthAnswersHead(t *testing.T) {
	rec := doRequest(t, testHandler(t, nil), http.MethodHead, "/health", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestPlannedEndpointsReturnNotImplemented(t *testing.T) {
	h := testHandler(t, nil)
	for _, path := range []string{"/auth/login", "/auth/whoami", "/sync/push", "/sync/pull", "/sync/bootstrap"} {
		t.Run(path, func(t *testing.T) {
			rec := doRequest(t, h, http.MethodPost, path, strings.NewReader(`{}`))
			if rec.Code != http.StatusNotImplemented {
				t.Fatalf("status = %d, want 501", rec.Code)
			}
			var body ErrorBody
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body.Code != "not_implemented" {
				t.Errorf("code = %q, want not_implemented", body.Code)
			}
		})
	}
}

// A wrong method on an existing path must not look like a wrong URL.
func TestWrongMethodIsMethodNotAllowed(t *testing.T) {
	h := testHandler(t, nil)
	cases := []struct {
		method, path, allow string
	}{
		{method: http.MethodGet, path: "/sync/push", allow: http.MethodPost},
		{method: http.MethodPost, path: "/health", allow: http.MethodGet},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			rec := doRequest(t, h, tc.method, tc.path, nil)
			if rec.Code != http.StatusMethodNotAllowed {
				t.Fatalf("status = %d, want 405", rec.Code)
			}
			if got := rec.Header().Get("Allow"); got != tc.allow {
				t.Errorf("Allow = %q, want %q", got, tc.allow)
			}
			var body ErrorBody
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body.Code != "method_not_allowed" {
				t.Errorf("code = %q, want method_not_allowed", body.Code)
			}
		})
	}
}

func TestUnknownPathIsJSONNotFound(t *testing.T) {
	rec := doRequest(t, testHandler(t, nil), http.MethodGet, "/nope", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	var body ErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("404 body is not JSON (%q): %v", rec.Body.String(), err)
	}
	if body.Code != "not_found" {
		t.Errorf("code = %q, want not_found", body.Code)
	}
}

func TestBasePathMountsEverythingBelowIt(t *testing.T) {
	h := testHandler(t, func(o *Options) { o.BasePath = "/croffle" })

	if rec := doRequest(t, h, http.MethodGet, "/croffle/health", nil); rec.Code != http.StatusOK {
		t.Errorf("GET /croffle/health = %d, want 200", rec.Code)
	}
	if rec := doRequest(t, h, http.MethodPost, "/croffle/sync/push", strings.NewReader(`{}`)); rec.Code != http.StatusNotImplemented {
		t.Errorf("POST /croffle/sync/push = %d, want 501", rec.Code)
	}
	// Without the prefix nothing is served, which is what makes a proxy that
	// strips the prefix and one that does not two different configurations.
	if rec := doRequest(t, h, http.MethodGet, "/health", nil); rec.Code == http.StatusOK {
		t.Error("GET /health answered 200 although the server is mounted at /croffle")
	}
}

func TestRequestIDIsEchoed(t *testing.T) {
	rec := doRequest(t, testHandler(t, nil), http.MethodGet, "/health", nil)
	if id := rec.Header().Get(HeaderRequestID); len(id) != 16 {
		t.Errorf("%s = %q, want 16 hex chars", HeaderRequestID, id)
	}
}

func TestOversizedContentLengthIsRejectedBeforeReading(t *testing.T) {
	h := testHandler(t, nil)
	rec := httptest.NewRecorder()

	req := httptest.NewRequest(http.MethodPost, "/sync/push", strings.NewReader(`{}`))
	// Claim a huge body without sending one: the fast path must refuse on the
	// header alone, otherwise a large upload is read before being thrown away.
	req.ContentLength = MaxBodyBytes + 1
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", rec.Code)
	}
	var body ErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Code != "body_too_large" {
		t.Errorf("code = %q, want body_too_large", body.Code)
	}
}

// Requests without a Content-Length only fail once a handler reads past the
// limit, so DecodeJSON has to turn that into the same 413.
func TestDecodeJSONMapsMaxBytesErrorTo413(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sync/push", strings.NewReader(`{"a":"`+strings.Repeat("x", 128)+`"}`))
	req.ContentLength = -1
	req.Body = http.MaxBytesReader(rec, req.Body, 16)

	var dst map[string]string
	if DecodeJSON(rec, req, &dst) {
		t.Fatal("DecodeJSON succeeded on an oversized body")
	}
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", rec.Code)
	}
}

func TestDecodeJSONRejectsBadJSONWithoutEchoingIt(t *testing.T) {
	rec := httptest.NewRecorder()
	secret := "private-appointment-title"
	req := httptest.NewRequest(http.MethodPost, "/sync/push", strings.NewReader(`{"title": "`+secret+`"`))

	var dst map[string]string
	if DecodeJSON(rec, req, &dst) {
		t.Fatal("DecodeJSON succeeded on malformed JSON")
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	// The decoder's own message quotes the offending input, which for sync
	// requests is calendar data.
	if strings.Contains(rec.Body.String(), secret) {
		t.Error("error body echoed the request payload")
	}
}

func TestRealIPHonoursTrustProxySetting(t *testing.T) {
	cases := []struct {
		name       string
		trustProxy bool
		forwarded  string
		want       string
	}{
		{name: "untrusted header is ignored", trustProxy: false, forwarded: "203.0.113.9", want: "192.0.2.1"},
		{name: "trusted header wins", trustProxy: true, forwarded: "203.0.113.9", want: "203.0.113.9"},
		{name: "left-most entry is the client", trustProxy: true, forwarded: "203.0.113.9, 10.0.0.1", want: "203.0.113.9"},
		{name: "host:port form is unwrapped", trustProxy: true, forwarded: "[2001:db8::1]:4321", want: "2001:db8::1"},
		{name: "empty header falls back to socket", trustProxy: true, forwarded: "", want: "192.0.2.1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got string
			h := realIP(tc.trustProxy, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				got = RemoteIP(r.Context())
			}))

			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			req.RemoteAddr = "192.0.2.1:1111"
			if tc.forwarded != "" {
				req.Header.Set("X-Forwarded-For", tc.forwarded)
			}
			h.ServeHTTP(httptest.NewRecorder(), req)

			if got != tc.want {
				t.Errorf("RemoteIP = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRecovererTurnsPanicIntoJSON500(t *testing.T) {
	h := recoverer(quietLogger(), http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	var body ErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Code != "internal" {
		t.Errorf("code = %q, want internal", body.Code)
	}
	if strings.Contains(rec.Body.String(), "boom") {
		t.Error("500 body leaked the panic value")
	}
}

func TestAccessLogDoesNotRecordHeadersOrBody(t *testing.T) {
	var sink strings.Builder
	logger := slog.New(slog.NewJSONHandler(&sink, &slog.HandlerOptions{Level: slog.LevelInfo}))

	h := NewHandler(Options{Version: "1.2.3", BasePath: "/", Logger: logger, Now: func() time.Time { return fixedNow }})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sync/push", strings.NewReader(`{"token":"super-secret"}`))
	req.Header.Set("Authorization", "Bearer super-secret")
	h.ServeHTTP(rec, req)

	out := sink.String()
	if !strings.Contains(out, `"status":501`) {
		t.Fatalf("access log did not record the status: %s", out)
	}
	if strings.Contains(out, "super-secret") {
		t.Errorf("access log leaked a credential: %s", out)
	}
	// A planned-but-unbuilt endpoint is not a server failure.
	if strings.Contains(out, `"level":"ERROR"`) {
		t.Errorf("501 was logged at error level: %s", out)
	}
}
