package webhook

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/team-croffle/croffle/apps/server/internal/hooks"
)

func quiet() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError + 1}))
}

func event() hooks.Event {
	return hooks.Event{
		Type: hooks.TypeRowShared,
		At:   1_700_000_000_000,
		Data: hooks.RowShared{AccountID: "acct-1", Table: "schedule", RowID: "row-1", TeamID: "team-default", ServerSeq: 7},
	}
}

func TestNewRequiresAUsableConfiguration(t *testing.T) {
	// No URL is the normal case: no sink, no error.
	sink, err := New(Config{}, quiet())
	if err != nil || sink != nil {
		t.Fatalf("New(empty) = %v, %v; want nil, nil", sink, err)
	}

	// A URL without a secret is refused rather than sent unsigned: a receiver
	// that cannot verify the sender has to trust anyone who knows the URL.
	if _, err := New(Config{URL: "https://example.test/hook"}, quiet()); err == nil {
		t.Error("a webhook without a secret was accepted")
	}

	for _, bad := range []string{"not-a-url", "ftp://example.test", "/relative"} {
		if _, err := New(Config{URL: bad, Secret: "s"}, quiet()); err == nil {
			t.Errorf("New(%q) was accepted", bad)
		}
	}

	if _, err := New(Config{URL: "https://example.test/hook", Secret: "s", Types: []string{"nonsense"}}, quiet()); err == nil {
		t.Error("an unknown event type was accepted")
	}
}

func TestDeliverSignsTheBody(t *testing.T) {
	var (
		gotSignature string
		gotEvent     string
		gotDelivery  string
		gotBody      []byte
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSignature = r.Header.Get(HeaderSignature)
		gotEvent = r.Header.Get(HeaderEvent)
		gotDelivery = r.Header.Get(HeaderDelivery)
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	sink, err := New(Config{URL: server.URL, Secret: "topsecret"}, quiet())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := sink.Deliver(context.Background(), event()); err != nil {
		t.Fatalf("Deliver: %v", err)
	}

	if gotEvent != hooks.TypeRowShared {
		t.Errorf("%s = %q", HeaderEvent, gotEvent)
	}
	if gotDelivery == "" {
		t.Errorf("%s is empty", HeaderDelivery)
	}
	// The receiver can tell the sender apart from anyone who guessed the URL.
	if !Verify("topsecret", gotBody, gotSignature) {
		t.Errorf("signature %q does not verify", gotSignature)
	}
	if Verify("wrong-secret", gotBody, gotSignature) {
		t.Error("the signature verifies with the wrong secret")
	}

	// The event says which row moved, not what it contains.
	var decoded map[string]any
	if err := json.Unmarshal(gotBody, &decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded["type"] != hooks.TypeRowShared {
		t.Errorf("body = %s", gotBody)
	}
	if strings.Contains(string(gotBody), "payload") {
		t.Errorf("the event carries a row payload: %s", gotBody)
	}
}

func TestDeliverOnlySendsSubscribedEvents(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sink, err := New(Config{URL: server.URL, Secret: "s", Types: []string{hooks.TypeRetentionSwept}}, quiet())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := sink.Deliver(context.Background(), event()); err != nil {
		t.Fatalf("Deliver: %v", err)
	}
	if calls.Load() != 0 {
		t.Errorf("calls = %d, want the unsubscribed event to be skipped", calls.Load())
	}

	if err := sink.Deliver(context.Background(), hooks.Event{Type: hooks.TypeRetentionSwept}); err != nil {
		t.Fatalf("Deliver(subscribed): %v", err)
	}
	if calls.Load() != 1 {
		t.Errorf("calls = %d, want 1", calls.Load())
	}
}

func TestDeliverRetriesServerErrorsButNotClientErrors(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sink, err := New(Config{URL: server.URL, Secret: "s"}, quiet())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	sink.sleep = func(time.Duration) {} // no waiting in a test

	if err := sink.Deliver(context.Background(), event()); err != nil {
		t.Fatalf("Deliver: %v", err)
	}
	if calls.Load() != 3 {
		t.Errorf("attempts = %d, want 3", calls.Load())
	}

	// A 4xx is the receiver saying "do not send me this", so it is final.
	calls.Store(0)
	rejecting := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer rejecting.Close()

	sink, err = New(Config{URL: rejecting.URL, Secret: "s"}, quiet())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	sink.sleep = func(time.Duration) {}

	if err := sink.Deliver(context.Background(), event()); err == nil {
		t.Error("a 400 was treated as success")
	}
	if calls.Load() != 1 {
		t.Errorf("attempts on a 400 = %d, want 1", calls.Load())
	}
}

func TestDeliverGivesUpAfterTheLastAttempt(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	sink, err := New(Config{URL: server.URL, Secret: "s"}, quiet())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	sink.sleep = func(time.Duration) {}

	if err := sink.Deliver(context.Background(), event()); err == nil {
		t.Error("Deliver reported success although every attempt failed")
	}
	if calls.Load() != maxAttempts {
		t.Errorf("attempts = %d, want %d", calls.Load(), maxAttempts)
	}
}
