package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/team-croffle/croffle/apps/server/internal/auth"
	"github.com/team-croffle/croffle/apps/server/internal/protocol"
	"github.com/team-croffle/croffle/apps/server/internal/store"
)

type fakeSyncer struct {
	result store.ApplyResult
	err    error
	got    store.ApplyRequest
}

func (f *fakeSyncer) ApplyCommits(_ context.Context, req store.ApplyRequest) (store.ApplyResult, error) {
	f.got = req
	return f.result, f.err
}

// syncHandler wires an authenticated caller that belongs to the default team.
func syncHandler(t *testing.T, syncer Syncer) http.Handler {
	t.Helper()
	return testHandler(t, func(o *Options) {
		o.Auth = &fakeAuth{identity: auth.Identity{
			AccountID: "acct-1", DeviceID: "client-1", TeamIDs: []string{store.DefaultTeamID},
		}}
		o.Sync = syncer
	})
}

func pushBody(commits ...protocol.Commit) string {
	data, err := json.Marshal(protocol.PushRequest{ClientID: "client-1", Commits: commits})
	if err != nil {
		panic(err)
	}
	return string(data)
}

func wireCommit() protocol.Commit {
	return protocol.Commit{
		OpID: "op-1", Table: "schedule", RowID: "row-1", Op: protocol.OpUpsert,
		BaseVersion: 0, Version: 1, UpdatedAtMs: 1_700_000_000_000,
		LastWriterID: "device:client-1",
		Row: protocol.Row{
			"id": "row-1", "title": "standup",
			"version": float64(1), "updatedAtMs": float64(1_700_000_000_000),
			"ownerId": "acct-1", "teamId": nil, "deletedAt": nil,
		},
	}
}

func TestPushPassesTheAuthenticatedIdentityToTheStore(t *testing.T) {
	syncer := &fakeSyncer{result: store.ApplyResult{Accepted: []string{"op-1"}, ServerSeq: 7}}

	rec := postJSON(t, syncHandler(t, syncer), "/sync/push", pushBody(wireCommit()), "tok")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}

	var body protocol.PushResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Accepted) != 1 || body.ServerSeq != 7 {
		t.Errorf("body = %+v", body)
	}

	// Ownership comes from the token; a row's owner is never taken from the
	// request body.
	if syncer.got.AccountID != "acct-1" || syncer.got.DeviceID != "client-1" {
		t.Errorf("store received %+v", syncer.got)
	}
	if len(syncer.got.Commits) != 1 {
		t.Fatalf("commits = %d, want 1", len(syncer.got.Commits))
	}

	// The promoted fields are stripped from the payload the store keeps.
	var payload map[string]any
	if err := json.Unmarshal(syncer.got.Commits[0].Payload, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if _, ok := payload["version"]; ok {
		t.Error("payload still carries the promoted version field")
	}
	if payload["title"] != "standup" {
		t.Errorf("payload lost the calendar fields: %v", payload)
	}
}

func TestPushRendersConflictsBackToTheWire(t *testing.T) {
	writer := "device:other"
	team := store.DefaultTeamID
	deletedAt := int64(1_700_000_002_000)

	syncer := &fakeSyncer{result: store.ApplyResult{
		Accepted: []string{},
		Conflicts: []store.Conflict{{
			OpID: "op-1", Table: "schedule", RowID: "row-1",
			ServerRow: &store.Row{
				Table: "schedule", RowID: "row-1", OwnerID: "acct-1", TeamID: &team,
				Version: 9, UpdatedAtMs: 1_700_000_002_000, LastWriterID: &writer,
				DeletedAt: &deletedAt, Payload: json.RawMessage(`{"title":"theirs"}`),
			},
		}},
		ServerSeq: 9,
	}}

	rec := postJSON(t, syncHandler(t, syncer), "/sync/push", pushBody(wireCommit()), "tok")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var body protocol.PushResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Conflicts) != 1 {
		t.Fatalf("conflicts = %+v, want 1", body.Conflicts)
	}

	conflict := body.Conflicts[0]
	if conflict.ServerVersion != 9 || conflict.ServerLastWriterID != writer {
		t.Errorf("conflict = %+v", conflict)
	}
	if !conflict.ServerRowWasDeleted {
		t.Error("the conflict does not report that the server row is a tombstone")
	}
	// The client needs the whole server row to settle the conflict with the
	// same rule the server would have used.
	if conflict.ServerRow["title"] != "theirs" {
		t.Errorf("serverRow = %v, want the payload fields", conflict.ServerRow)
	}
	if conflict.ServerRow["version"] != float64(9) {
		t.Errorf("serverRow.version = %v, want the promoted column", conflict.ServerRow["version"])
	}
}

func TestPushRejectsMalformedCommits(t *testing.T) {
	cases := map[string]func(*protocol.Commit){
		"unknown table":     func(c *protocol.Commit) { c.Table = "secrets" },
		"missing opId":      func(c *protocol.Commit) { c.OpID = "" },
		"version not above": func(c *protocol.Commit) { c.BaseVersion = 3; c.Version = 3 },
		"upsert no row":     func(c *protocol.Commit) { c.Row = nil },
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			commit := wireCommit()
			mutate(&commit)

			rec := postJSON(t, syncHandler(t, &fakeSyncer{}), "/sync/push", pushBody(commit), "tok")
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (%s)", rec.Code, rec.Body.String())
			}
			if got := errorCode(t, rec); got != "invalid_commit" {
				t.Errorf("code = %q, want invalid_commit", got)
			}
		})
	}
}

func TestPushRejectsAnOversizedBatch(t *testing.T) {
	commits := make([]protocol.Commit, protocol.MaxCommitsPerPush+1)
	for i := range commits {
		commits[i] = wireCommit()
	}

	rec := postJSON(t, syncHandler(t, &fakeSyncer{}), "/sync/push", pushBody(commits...), "tok")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := errorCode(t, rec); got != "too_many_commits" {
		t.Errorf("code = %q, want too_many_commits", got)
	}
}

// Sharing into a team the account is not in would make the row visible to
// strangers, so it is refused before it reaches the store.
func TestPushRejectsAForeignTeam(t *testing.T) {
	commit := wireCommit()
	commit.Row["teamId"] = "team-someone-else"

	syncer := &fakeSyncer{}
	rec := postJSON(t, syncHandler(t, syncer), "/sync/push", pushBody(commit), "tok")

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (%s)", rec.Code, rec.Body.String())
	}
	if got := errorCode(t, rec); got != "foreign_team" {
		t.Errorf("code = %q, want foreign_team", got)
	}
	if syncer.got.AccountID != "" {
		t.Error("the commit reached the store despite naming a foreign team")
	}
}

func TestPushMapsStoreFailures(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "foreign team from the store", err: store.ErrForeignTeam, status: http.StatusForbidden, code: "foreign_team"},
		{name: "not implemented", err: store.ErrNotImplemented, status: http.StatusNotImplemented, code: "not_implemented"},
		{name: "anything else", err: errors.New("disk on fire"), status: http.StatusInternalServerError, code: "internal"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := postJSON(t, syncHandler(t, &fakeSyncer{err: tc.err}), "/sync/push", pushBody(wireCommit()), "tok")
			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d", rec.Code, tc.status)
			}
			if got := errorCode(t, rec); got != tc.code {
				t.Errorf("code = %q, want %q", got, tc.code)
			}
			if tc.code == "internal" && strings.Contains(rec.Body.String(), "disk on fire") {
				t.Error("the response leaked the internal error")
			}
		})
	}
}

func TestPushWithoutAStoreIsNotImplemented(t *testing.T) {
	h := testHandler(t, func(o *Options) {
		o.Auth = &fakeAuth{identity: auth.Identity{AccountID: "acct-1", DeviceID: "client-1"}}
	})

	rec := postJSON(t, h, "/sync/push", pushBody(wireCommit()), "tok")
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", rec.Code)
	}
}

func TestPushAcceptsAnEmptyBatch(t *testing.T) {
	syncer := &fakeSyncer{result: store.ApplyResult{Accepted: []string{}, ServerSeq: 4}}

	rec := postJSON(t, syncHandler(t, syncer), "/sync/push", pushBody(), "tok")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}

	// An empty push is how a client asks "where does the log stand?" without
	// having anything to send.
	var body protocol.PushResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.ServerSeq != 4 {
		t.Errorf("serverSeq = %d, want 4", body.ServerSeq)
	}
	if body.Accepted == nil || body.Conflicts == nil {
		t.Error("accepted and conflicts must be empty arrays, not null")
	}
}
