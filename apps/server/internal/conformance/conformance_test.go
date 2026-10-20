// Package conformance runs the shared protocol scenarios against the real
// server stack.
//
// The scenarios live in packages/sync-fixtures and are read by the desktop as
// well, so a protocol change that lands on only one side fails here instead of
// in somebody's calendar.
package conformance

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/team-croffle/croffle/apps/server/internal/auth"
	"github.com/team-croffle/croffle/apps/server/internal/httpapi"
	"github.com/team-croffle/croffle/apps/server/internal/store"
	"github.com/team-croffle/croffle/apps/server/internal/store/sqlite"
)

// fixturesDir is the shared scenario directory, relative to this package.
const fixturesDir = "../../../../packages/sync-fixtures/scenarios"

// Markers a scenario can use instead of a value.
const (
	markerAny    = "$any"
	markerAbsent = "$absent"
)

type scenario struct {
	Name        string    `json:"name"`
	Protocol    int       `json:"protocol"`
	Description string    `json:"description"`
	Accounts    []account `json:"accounts"`
	Devices     []device  `json:"devices"`
	Steps       []step    `json:"steps"`
	// RowCodec is read by the desktop runner only; the server has no row
	// encoder of its own to check.
	RowCodec json.RawMessage `json:"rowCodec,omitempty"`
}

type account struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Teams    []string `json:"teams"`
}

type device struct {
	ID        string `json:"id"`
	AccountID string `json:"accountId"`
	Revoked   bool   `json:"revoked"`
}

type step struct {
	Request *request     `json:"request,omitempty"`
	Expect  *expectation `json:"expect,omitempty"`
	Effect  string       `json:"effect,omitempty"`
	Value   int64        `json:"value,omitempty"`
}

type request struct {
	Endpoint string         `json:"endpoint"`
	As       string         `json:"as"`
	Body     map[string]any `json:"body"`
}

type expectation struct {
	Status int            `json:"status"`
	Body   map[string]any `json:"body"`
}

func TestScenarios(t *testing.T) {
	entries, err := os.ReadDir(fixturesDir)
	if err != nil {
		t.Fatalf("read %s: %v (are the shared fixtures checked out?)", fixturesDir, err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	if len(names) == 0 {
		t.Fatalf("no scenarios in %s", fixturesDir)
	}

	for _, name := range names {
		t.Run(strings.TrimSuffix(name, ".json"), func(t *testing.T) {
			sc := load(t, filepath.Join(fixturesDir, name))
			if len(sc.Steps) == 0 {
				// A scenario with only a rowCodec block is for the desktop
				// runner; nothing for the server to do. An empty scenario with
				// nothing for either side is a mistake, not a skip.
				if len(sc.RowCodec) == 0 {
					t.Fatalf("%s has neither request steps nor a rowCodec block", sc.Name)
				}
				t.Skipf("%s is a desktop-only fixture (rowCodec)", sc.Name)
			}
			runScenario(t, sc)
		})
	}
}

func load(t *testing.T, path string) scenario {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	var sc scenario
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&sc); err != nil {
		// Unknown fields are a failure on purpose: a scenario that uses a
		// feature this runner does not implement must not look like it passed.
		t.Fatalf("decode %s: %v", path, err)
	}
	if sc.Protocol != 1 {
		t.Fatalf("%s targets protocol %d, this server speaks 1", path, sc.Protocol)
	}
	return sc
}

type harness struct {
	url    string
	db     *sqlite.DB
	path   string
	tokens map[string]string
}

func runScenario(t *testing.T, sc scenario) {
	t.Helper()
	h := setup(t, sc)

	for i, s := range sc.Steps {
		switch {
		case s.Effect != "":
			h.applyEffect(t, i, s)
		case s.Request != nil:
			h.runRequest(t, i, s, sc.Description)
		default:
			t.Fatalf("step %d is neither a request nor an effect", i)
		}
	}
}

func setup(t *testing.T, sc scenario) *harness {
	t.Helper()
	ctx := context.Background()

	path := filepath.Join(t.TempDir(), "server.db")
	db, err := sqlite.Open(ctx, path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	hash, err := auth.HashPassword("conformance")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	for _, a := range sc.Accounts {
		if err := db.CreateAccount(ctx, store.Account{
			ID: a.ID, Username: a.Username, PasswordHash: hash, CreatedAt: time.Now(),
		}); err != nil {
			t.Fatalf("create account %s: %v", a.ID, err)
		}
		for _, team := range a.Teams {
			if err := db.AddTeamMember(ctx, team, a.ID, "member"); err != nil {
				t.Fatalf("add %s to %s: %v", a.ID, team, err)
			}
		}
	}

	tokens := map[string]string{}
	for _, d := range sc.Devices {
		token, tokenHash, err := auth.NewToken()
		if err != nil {
			t.Fatalf("new token: %v", err)
		}
		if err := db.UpsertDevice(ctx, store.Device{
			ID: d.ID, AccountID: d.AccountID, Name: "conformance",
			TokenHash: tokenHash, CreatedAt: time.Now(),
		}); err != nil {
			t.Fatalf("create device %s: %v", d.ID, err)
		}
		if d.Revoked {
			if err := db.RevokeDevice(ctx, d.ID); err != nil {
				t.Fatalf("revoke device %s: %v", d.ID, err)
			}
		}
		tokens[d.ID] = token
	}

	handler := httpapi.NewHandler(httpapi.Options{
		Version:  "conformance",
		BasePath: "/",
		Logger:   slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError + 1})),
		Health:   db.Ping,
		Auth:     auth.NewService(db, time.Now),
		Sync:     db,
	})

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	return &harness{url: srv.URL, db: db, path: path, tokens: tokens}
}

// applyEffect produces a state no endpoint can: so far only "retention has
// swept this far".
func (h *harness) applyEffect(t *testing.T, index int, s step) {
	t.Helper()

	switch s.Effect {
	case "setPurgedThroughSeq":
		// Written through a second handle rather than a test-only method on the
		// store: the production API should not grow an escape hatch for this.
		raw, err := sql.Open("sqlite", "file:"+h.path)
		if err != nil {
			t.Fatalf("step %d: open database: %v", index, err)
		}
		defer func() { _ = raw.Close() }()

		if _, err := raw.Exec(
			`UPDATE sync_meta SET value = ? WHERE key = 'purgedThroughSeq'`, s.Value); err != nil {
			t.Fatalf("step %d: %v", index, err)
		}
	default:
		// An unimplemented effect must fail loudly: skipping it would let a
		// scenario pass without checking what it was written for.
		t.Fatalf("step %d: unknown effect %q", index, s.Effect)
	}
}

func (h *harness) runRequest(t *testing.T, index int, s step, description string) {
	t.Helper()
	if s.Expect == nil {
		t.Fatalf("step %d has a request but no expectation", index)
	}

	body, err := json.Marshal(s.Request.Body)
	if err != nil {
		t.Fatalf("step %d: encode body: %v", index, err)
	}

	req, err := http.NewRequestWithContext(context.Background(),
		http.MethodPost, h.url+s.Request.Endpoint, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("step %d: build request: %v", index, err)
	}
	req.Header.Set("Content-Type", "application/json")

	if s.Request.As != "" {
		token, ok := h.tokens[s.Request.As]
		if !ok {
			t.Fatalf("step %d: no device %q in this scenario", index, s.Request.As)
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("step %d: POST %s: %v", index, s.Request.Endpoint, err)
	}
	defer func() { _ = res.Body.Close() }()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("step %d: read body: %v", index, err)
	}

	if res.StatusCode != s.Expect.Status {
		t.Fatalf("step %d (%s): status = %d, want %d\nbody: %s\nscenario: %s",
			index, s.Request.Endpoint, res.StatusCode, s.Expect.Status, raw, description)
	}
	if len(s.Expect.Body) == 0 {
		return
	}

	var actual map[string]any
	if err := json.Unmarshal(raw, &actual); err != nil {
		t.Fatalf("step %d: response is not a JSON object (%s): %v", index, raw, err)
	}

	if err := matchObject(s.Expect.Body, actual); err != nil {
		t.Fatalf("step %d (%s): %v\nfull response: %s\nscenario: %s",
			index, s.Request.Endpoint, err, raw, description)
	}
}

// matchObject compares expected against actual as a subset, so adding a field
// to a response does not break every scenario.
func matchObject(expected, actual map[string]any) error {
	for key, want := range expected {
		got, present := actual[key]

		if marker, ok := want.(string); ok {
			switch marker {
			case markerAbsent:
				if present && got != nil {
					return fmt.Errorf("%q should be absent but is %v", key, got)
				}
				continue
			case markerAny:
				if !present || got == nil {
					return fmt.Errorf("%q should be present with any value", key)
				}
				continue
			}
		}

		if !present {
			return fmt.Errorf("%q is missing", key)
		}
		if err := matchValue(want, got); err != nil {
			return fmt.Errorf("%q: %w", key, err)
		}
	}
	return nil
}

func matchValue(want, got any) error {
	switch expected := want.(type) {
	case map[string]any:
		actual, ok := got.(map[string]any)
		if !ok {
			return fmt.Errorf("want an object, got %T (%v)", got, got)
		}
		return matchObject(expected, actual)

	case []any:
		actual, ok := got.([]any)
		if !ok {
			return fmt.Errorf("want an array, got %T (%v)", got, got)
		}
		// Lengths must match: "the first two of many" is almost never what a
		// scenario means, and a silent extra element hides a bug.
		if len(expected) != len(actual) {
			return fmt.Errorf("want %d elements, got %d (%v)", len(expected), len(actual), actual)
		}
		for i := range expected {
			if err := matchValue(expected[i], actual[i]); err != nil {
				return fmt.Errorf("[%d]: %w", i, err)
			}
		}
		return nil

	default:
		if !sameScalar(want, got) {
			return fmt.Errorf("want %v (%T), got %v (%T)", want, want, got, got)
		}
		return nil
	}
}

// sameScalar compares after JSON round-tripping, so 1 and 1.0 are equal and a
// scenario does not have to know how Go decoded the number.
func sameScalar(want, got any) bool {
	wantJSON, err1 := json.Marshal(want)
	gotJSON, err2 := json.Marshal(got)
	if err1 != nil || err2 != nil {
		return false
	}
	return string(wantJSON) == string(gotJSON)
}
