package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/team-croffle/croffle/apps/server/internal/store"
)

const (
	testAccount = "acct-1"
	testDevice  = "client-1"
)

func newSyncFixture(t *testing.T) (*DB, context.Context) {
	t.Helper()
	db := open(t, tempPath(t))
	ctx := context.Background()
	newAccount(t, db, testAccount, "alice")
	if err := db.AddTeamMember(ctx, store.DefaultTeamID, testAccount, "member"); err != nil {
		t.Fatalf("add team member: %v", err)
	}
	return db, ctx
}

func request(commits ...store.Commit) store.ApplyRequest {
	return store.ApplyRequest{
		AccountID: testAccount,
		DeviceID:  testDevice,
		TeamIDs:   []string{store.DefaultTeamID},
		Commits:   commits,
		Now:       time.UnixMilli(1_700_000_000_000),
	}
}

func upsert(opID, rowID string, base, version int64) store.Commit {
	return store.Commit{
		OpID: opID, Table: "schedule", RowID: rowID, Op: store.OpUpsert,
		BaseVersion: base, Version: version, UpdatedAtMs: 1_700_000_000_000,
		LastWriterID: "device:" + testDevice,
		Payload:      json.RawMessage(`{"title":"standup"}`),
	}
}

func readRow(t *testing.T, db *DB, table, rowID string) *store.Row {
	t.Helper()
	tx, err := db.write.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	row, err := currentRow(context.Background(), tx, table, rowID)
	if err != nil {
		t.Fatalf("read row: %v", err)
	}
	return row
}

func TestApplyCommitsInsertsNewRow(t *testing.T) {
	db, ctx := newSyncFixture(t)

	result, err := db.ApplyCommits(ctx, request(upsert("op-1", "row-1", 0, 1)))
	if err != nil {
		t.Fatalf("ApplyCommits: %v", err)
	}
	if len(result.Accepted) != 1 || result.Accepted[0] != "op-1" {
		t.Fatalf("accepted = %v", result.Accepted)
	}
	if len(result.Conflicts) != 0 {
		t.Errorf("conflicts = %+v, want none", result.Conflicts)
	}
	if result.ServerSeq != 1 {
		t.Errorf("serverSeq = %d, want 1", result.ServerSeq)
	}

	row := readRow(t, db, "schedule", "row-1")
	if row == nil {
		t.Fatal("row was not stored")
	}
	if row.Version != 1 || row.ServerSeq != 1 {
		t.Errorf("row = %+v", row)
	}
	// Ownership comes from the token, never from the commit.
	if row.OwnerID != testAccount {
		t.Errorf("ownerId = %q, want %q", row.OwnerID, testAccount)
	}
	if row.Deleted() {
		t.Error("a fresh upsert is a tombstone")
	}
	if string(row.Payload) != `{"title":"standup"}` {
		t.Errorf("payload = %s", row.Payload)
	}
}

// A push whose response was lost is retried with the same opIds; those commits
// must be reported as accepted without being applied a second time.
func TestApplyCommitsIsIdempotent(t *testing.T) {
	db, ctx := newSyncFixture(t)
	commit := upsert("op-1", "row-1", 0, 1)

	first, err := db.ApplyCommits(ctx, request(commit))
	if err != nil {
		t.Fatalf("first push: %v", err)
	}
	second, err := db.ApplyCommits(ctx, request(commit))
	if err != nil {
		t.Fatalf("retry: %v", err)
	}

	if len(second.Accepted) != 1 {
		t.Errorf("retry accepted = %v, want the opId", second.Accepted)
	}
	if second.ServerSeq != first.ServerSeq {
		t.Errorf("retry serverSeq = %d, want %d (no new commit)", second.ServerSeq, first.ServerSeq)
	}

	var commits int
	if err := db.read.QueryRow("SELECT COUNT(*) FROM commit_log").Scan(&commits); err != nil {
		t.Fatalf("count commits: %v", err)
	}
	if commits != 1 {
		t.Errorf("commit log entries = %d, want 1", commits)
	}
}

func TestApplyCommitsReportsConflictWithServerRow(t *testing.T) {
	db, ctx := newSyncFixture(t)

	if _, err := db.ApplyCommits(ctx, request(upsert("op-1", "row-1", 0, 1))); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// The client thinks the server is still at version 0.
	stale := upsert("op-2", "row-1", 0, 2)
	result, err := db.ApplyCommits(ctx, request(stale))
	if err != nil {
		t.Fatalf("ApplyCommits: %v", err)
	}

	if len(result.Accepted) != 0 {
		t.Errorf("accepted = %v, want none", result.Accepted)
	}
	if len(result.Conflicts) != 1 {
		t.Fatalf("conflicts = %+v, want 1", result.Conflicts)
	}
	conflict := result.Conflicts[0]
	if conflict.OpID != "op-2" || conflict.ServerRow == nil {
		t.Fatalf("conflict = %+v, want the server row attached", conflict)
	}
	if conflict.ServerRow.Version != 1 {
		t.Errorf("server row version = %d, want 1", conflict.ServerRow.Version)
	}

	// Not recorded as applied: otherwise the client's corrected retry would be
	// treated as a duplicate and the change would be lost for good.
	var ops int
	if err := db.read.QueryRow("SELECT COUNT(*) FROM applied_op WHERE opId = 'op-2'").Scan(&ops); err != nil {
		t.Fatalf("count ops: %v", err)
	}
	if ops != 0 {
		t.Errorf("applied_op rows for the conflicted op = %d, want 0", ops)
	}

	// Retrying with the server's version applies.
	retry, err := db.ApplyCommits(ctx, request(upsert("op-3", "row-1", 1, 2)))
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if len(retry.Accepted) != 1 {
		t.Fatalf("retry accepted = %v", retry.Accepted)
	}
	if row := readRow(t, db, "schedule", "row-1"); row.Version != 2 {
		t.Errorf("row version = %d, want 2", row.Version)
	}
}

func TestApplyCommitsDeleteKeepsATombstone(t *testing.T) {
	db, ctx := newSyncFixture(t)

	if _, err := db.ApplyCommits(ctx, request(upsert("op-1", "row-1", 0, 1))); err != nil {
		t.Fatalf("seed: %v", err)
	}

	deletedAt := int64(1_700_000_001_000)
	remove := store.Commit{
		OpID: "op-2", Table: "schedule", RowID: "row-1", Op: store.OpDelete,
		BaseVersion: 1, Version: 2, UpdatedAtMs: deletedAt,
		LastWriterID: "device:" + testDevice, DeletedAt: &deletedAt,
		Payload: json.RawMessage(`{"title":"standup"}`),
	}
	if _, err := db.ApplyCommits(ctx, request(remove)); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// The row stays: "deleted" and "never received" have to stay
	// distinguishable for a client that is catching up.
	row := readRow(t, db, "schedule", "row-1")
	if row == nil {
		t.Fatal("the row was physically removed")
	}
	if !row.Deleted() || *row.DeletedAt != deletedAt {
		t.Errorf("deletedAt = %v, want %d", row.DeletedAt, deletedAt)
	}

	var op string
	if err := db.read.QueryRow("SELECT op FROM commit_log WHERE serverSeq = 2").Scan(&op); err != nil {
		t.Fatalf("read commit: %v", err)
	}
	if op != string(store.OpDelete) {
		t.Errorf("logged op = %q, want delete", op)
	}
}

func TestApplyCommitsStampsADeleteWithoutATimestamp(t *testing.T) {
	db, ctx := newSyncFixture(t)

	remove := store.Commit{
		OpID: "op-1", Table: "schedule", RowID: "row-1", Op: store.OpDelete,
		BaseVersion: 0, Version: 1, UpdatedAtMs: 1_700_000_000_000,
	}
	if _, err := db.ApplyCommits(ctx, request(remove)); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// Storing a delete without a tombstone would make it look live.
	row := readRow(t, db, "schedule", "row-1")
	if row == nil || !row.Deleted() {
		t.Fatalf("row = %+v, want a tombstone", row)
	}
}

func TestApplyCommitsSharesIntoOwnTeamOnly(t *testing.T) {
	db, ctx := newSyncFixture(t)

	team := store.DefaultTeamID
	shared := upsert("op-1", "row-1", 0, 1)
	shared.TeamID = &team
	if _, err := db.ApplyCommits(ctx, request(shared)); err != nil {
		t.Fatalf("share: %v", err)
	}
	if row := readRow(t, db, "schedule", "row-1"); row.TeamID == nil || *row.TeamID != team {
		t.Errorf("teamId = %v, want %s", row.TeamID, team)
	}

	foreign := "team-someone-else"
	bad := upsert("op-2", "row-2", 0, 1)
	bad.TeamID = &foreign
	_, err := db.ApplyCommits(ctx, request(bad))
	if !errors.Is(err, store.ErrForeignTeam) {
		t.Fatalf("error = %v, want ErrForeignTeam", err)
	}
}

// One batch is one transaction: a failure must not leave half of it applied.
func TestApplyCommitsRollsBackTheWholeBatchOnError(t *testing.T) {
	db, ctx := newSyncFixture(t)

	foreign := "team-someone-else"
	bad := upsert("op-2", "row-2", 0, 1)
	bad.TeamID = &foreign

	if _, err := db.ApplyCommits(ctx, request(upsert("op-1", "row-1", 0, 1), bad)); err == nil {
		t.Fatal("the batch succeeded, want an error")
	}

	if row := readRow(t, db, "schedule", "row-1"); row != nil {
		t.Error("the first commit of a failed batch was kept")
	}
	var commits int
	if err := db.read.QueryRow("SELECT COUNT(*) FROM commit_log").Scan(&commits); err != nil {
		t.Fatalf("count: %v", err)
	}
	if commits != 0 {
		t.Errorf("commit log entries = %d, want 0", commits)
	}
}

func TestApplyCommitsMixesAcceptedAndConflicted(t *testing.T) {
	db, ctx := newSyncFixture(t)

	if _, err := db.ApplyCommits(ctx, request(upsert("op-1", "row-1", 0, 1))); err != nil {
		t.Fatalf("seed: %v", err)
	}

	result, err := db.ApplyCommits(ctx, request(
		upsert("op-2", "row-1", 0, 2), // stale: conflict
		upsert("op-3", "row-2", 0, 1), // new: accepted
	))
	if err != nil {
		t.Fatalf("ApplyCommits: %v", err)
	}

	// Part accepted and part conflicted is normal, which is why push is a 200.
	if len(result.Accepted) != 1 || result.Accepted[0] != "op-3" {
		t.Errorf("accepted = %v, want [op-3]", result.Accepted)
	}
	if len(result.Conflicts) != 1 || result.Conflicts[0].OpID != "op-2" {
		t.Errorf("conflicts = %+v, want op-2", result.Conflicts)
	}
	if result.ServerSeq != 2 {
		t.Errorf("serverSeq = %d, want 2 (the commit this batch wrote)", result.ServerSeq)
	}
}

func TestApplyCommitsReportsCurrentSequenceWhenNothingIsWritten(t *testing.T) {
	db, ctx := newSyncFixture(t)

	if _, err := db.ApplyCommits(ctx, request(upsert("op-1", "row-1", 0, 1))); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Entirely stale batch: nothing is written, but the client still learns
	// where the log stands.
	result, err := db.ApplyCommits(ctx, request(upsert("op-2", "row-1", 0, 2)))
	if err != nil {
		t.Fatalf("ApplyCommits: %v", err)
	}
	if result.ServerSeq != 1 {
		t.Errorf("serverSeq = %d, want 1", result.ServerSeq)
	}
}

// The server may have dropped its history (retention), so a commit whose row is
// missing is inserted rather than refused: refusing it would strand the client.
func TestApplyCommitsInsertsWhenTheRowIsGone(t *testing.T) {
	db, ctx := newSyncFixture(t)

	result, err := db.ApplyCommits(ctx, request(upsert("op-1", "row-1", 7, 8)))
	if err != nil {
		t.Fatalf("ApplyCommits: %v", err)
	}
	if len(result.Accepted) != 1 {
		t.Fatalf("accepted = %v, want the commit", result.Accepted)
	}
	if row := readRow(t, db, "schedule", "row-1"); row == nil || row.Version != 8 {
		t.Errorf("row = %+v, want version 8", row)
	}
}

func TestApplyCommitsRequiresAnAccountAndDevice(t *testing.T) {
	db, ctx := newSyncFixture(t)

	req := request(upsert("op-1", "row-1", 0, 1))
	req.AccountID = ""
	if _, err := db.ApplyCommits(ctx, req); err == nil {
		t.Error("apply without an account succeeded")
	}

	req = request(upsert("op-1", "row-1", 0, 1))
	req.DeviceID = ""
	if _, err := db.ApplyCommits(ctx, req); err == nil {
		t.Error("apply without a device succeeded")
	}
}

func TestApplyCommitsKeepsSequencesContiguous(t *testing.T) {
	db, ctx := newSyncFixture(t)

	commits := make([]store.Commit, 0, 10)
	for i := range 10 {
		commits = append(commits, upsert("op-"+strconv.Itoa(i), "row-"+strconv.Itoa(i), 0, 1))
	}
	result, err := db.ApplyCommits(ctx, request(commits...))
	if err != nil {
		t.Fatalf("ApplyCommits: %v", err)
	}
	if result.ServerSeq != 10 {
		t.Fatalf("serverSeq = %d, want 10", result.ServerSeq)
	}

	rows, err := db.read.Query("SELECT serverSeq FROM commit_log ORDER BY serverSeq")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer func() { _ = rows.Close() }()

	want := int64(1)
	for rows.Next() {
		var seq int64
		if err := rows.Scan(&seq); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if seq != want {
			t.Fatalf("sequence = %d, want %d (a gap loses that change for every client)", seq, want)
		}
		want++
	}
}
