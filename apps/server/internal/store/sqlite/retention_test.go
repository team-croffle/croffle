package sqlite

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/team-croffle/croffle/apps/server/internal/store"
)

// seedCommits writes n commits and backdates them, so a sweep has something
// old to find.
func seedCommits(t *testing.T, db *DB, n int, age time.Duration) {
	t.Helper()
	ctx := context.Background()

	commits := make([]store.Commit, 0, n)
	for i := range n {
		id := strconv.Itoa(i)
		commits = append(commits, upsert("op-"+id, "row-"+id, 0, 1))
	}
	if _, err := db.ApplyCommits(ctx, request(commits...)); err != nil {
		t.Fatalf("seed commits: %v", err)
	}

	backdated := time.Now().Add(-age).UnixMilli()
	if _, err := db.write.Exec(`UPDATE commit_log SET createdAt = ?`, backdated); err != nil {
		t.Fatalf("backdate commits: %v", err)
	}
	if _, err := db.write.Exec(`UPDATE applied_op SET createdAt = ?`, backdated); err != nil {
		t.Fatalf("backdate ops: %v", err)
	}
}

func TestSweepRemovesOldHistoryAndKeepsState(t *testing.T) {
	db, ctx := newSyncFixture(t)
	seedCommits(t, db, 3, 100*24*time.Hour)

	cutoff := time.Now().Add(-90 * 24 * time.Hour)
	result, err := db.Sweep(ctx, store.SweepConfig{CommitsBefore: cutoff, OpsBefore: cutoff})
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}

	if result.DeletedCommits != 3 || result.DeletedOps != 3 {
		t.Errorf("result = %+v, want 3 commits and 3 ops", result)
	}
	if result.OldestCommitSeq != 0 {
		t.Errorf("oldestCommitSeq = %d, want 0 on an empty log", result.OldestCommitSeq)
	}

	// The current state is not history: a snapshot is built from it, so
	// sweeping it would lose data rather than reclaim space.
	if rows := countRows(t, db, "sync_row"); rows != 3 {
		t.Errorf("sync_row rows = %d, want 3 (state must survive a sweep)", rows)
	}
}

func TestSweepKeepsFreshEntries(t *testing.T) {
	db, ctx := newSyncFixture(t)
	seedCommits(t, db, 2, time.Hour)

	cutoff := time.Now().Add(-90 * 24 * time.Hour)
	result, err := db.Sweep(ctx, store.SweepConfig{CommitsBefore: cutoff, OpsBefore: cutoff})
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if result.DeletedCommits != 0 || result.DeletedOps != 0 {
		t.Errorf("result = %+v, want nothing removed", result)
	}
	if result.OldestCommitSeq != 1 {
		t.Errorf("oldestCommitSeq = %d, want 1", result.OldestCommitSeq)
	}
}

// After a sweep, a client whose cursor was inside the removed range has to be
// told to bootstrap; the marker is what pull compares against.
func TestSweepRaisesThePurgeMarkerAndPullRefusesOldCursors(t *testing.T) {
	db, ctx := newSyncFixture(t)
	seedCommits(t, db, 4, 100*24*time.Hour)

	// A newer commit that must survive. The apply time comes from the request
	// (the handler passes the server's clock), and the shared fixture pins it
	// to a date in the past, so this one sets it explicitly.
	fresh := request(upsert("op-fresh", "row-fresh", 0, 1))
	fresh.Now = time.Now()
	if _, err := db.ApplyCommits(ctx, fresh); err != nil {
		t.Fatalf("fresh commit: %v", err)
	}

	cutoff := time.Now().Add(-90 * 24 * time.Hour)
	result, err := db.Sweep(ctx, store.SweepConfig{CommitsBefore: cutoff, OpsBefore: cutoff})
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if result.DeletedCommits != 4 {
		t.Fatalf("deleted = %d, want the 4 old commits", result.DeletedCommits)
	}
	if result.OldestCommitSeq != 5 {
		t.Errorf("oldestCommitSeq = %d, want 5", result.OldestCommitSeq)
	}

	purged, err := db.PurgedThroughSeq(ctx)
	if err != nil {
		t.Fatalf("PurgedThroughSeq: %v", err)
	}
	if purged != 4 {
		t.Fatalf("purgedThroughSeq = %d, want 4", purged)
	}

	if _, err := db.PullSince(ctx, pullQuery(2, 0)); !errors.Is(err, store.ErrCursorTooOld) {
		t.Errorf("pull with a swept cursor = %v, want ErrCursorTooOld", err)
	}
	// A cursor at the boundary is still fine: everything after it survives.
	if _, err := db.PullSince(ctx, pullQuery(4, 0)); err != nil {
		t.Errorf("pull at the purge boundary: %v", err)
	}
}

// The marker must never move backwards: a lower value would tell a client its
// cursor is serviceable when the commits it needs are already gone.
func TestPurgeMarkerOnlyMovesForward(t *testing.T) {
	db, ctx := newSyncFixture(t)

	if err := db.raisePurgedThrough(ctx, 10); err != nil {
		t.Fatalf("raise to 10: %v", err)
	}
	if err := db.raisePurgedThrough(ctx, 4); err != nil {
		t.Fatalf("raise to 4: %v", err)
	}

	purged, err := db.PurgedThroughSeq(ctx)
	if err != nil {
		t.Fatalf("PurgedThroughSeq: %v", err)
	}
	if purged != 10 {
		t.Errorf("purgedThroughSeq = %d, want it to stay at 10", purged)
	}
}

func TestSweepHandlesMoreRowsThanOneBatch(t *testing.T) {
	db, ctx := newSyncFixture(t)

	// Two batches' worth, written directly: going through ApplyCommits would
	// be needlessly slow for a paging test.
	backdated := time.Now().Add(-100 * 24 * time.Hour).UnixMilli()
	for i := range sweepBatch + 10 {
		if _, err := db.write.Exec(`
			INSERT INTO commit_log (tbl, rowKey, op, ownerId, version, updatedAtMs, deviceId, createdAt)
			VALUES ('schedule', ?, 'upsert', ?, 1, 1, 'dev', ?)`,
			"row-"+strconv.Itoa(i), testAccount, backdated); err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
	}

	cutoff := time.Now().Add(-90 * 24 * time.Hour)
	result, err := db.Sweep(ctx, store.SweepConfig{CommitsBefore: cutoff, OpsBefore: cutoff})
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if result.DeletedCommits != int64(sweepBatch+10) {
		t.Errorf("deleted = %d, want %d", result.DeletedCommits, sweepBatch+10)
	}
	if rows := countRows(t, db, "commit_log"); rows != 0 {
		t.Errorf("commit_log rows = %d, want 0", rows)
	}
}

// A push must not fail while a sweep is running: both go through the single
// write connection, and the sweep yields between batches.
func TestPushSucceedsDuringASweep(t *testing.T) {
	db, ctx := newSyncFixture(t)

	backdated := time.Now().Add(-100 * 24 * time.Hour).UnixMilli()
	for i := range sweepBatch * 2 {
		if _, err := db.write.Exec(`
			INSERT INTO commit_log (tbl, rowKey, op, ownerId, version, updatedAtMs, deviceId, createdAt)
			VALUES ('schedule', ?, 'upsert', ?, 1, 1, 'dev', ?)`,
			"old-"+strconv.Itoa(i), testAccount, backdated); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	done := make(chan error, 1)
	go func() {
		cutoff := time.Now().Add(-90 * 24 * time.Hour)
		_, err := db.Sweep(ctx, store.SweepConfig{CommitsBefore: cutoff, OpsBefore: cutoff})
		done <- err
	}()

	for i := range 20 {
		if _, err := db.ApplyCommits(ctx, request(upsert("live-"+strconv.Itoa(i), "row-"+strconv.Itoa(i), 0, 1))); err != nil {
			t.Fatalf("push during sweep: %v", err)
		}
	}
	if err := <-done; err != nil {
		t.Fatalf("sweep: %v", err)
	}
}

func TestSweepRequiresBothCutoffs(t *testing.T) {
	db, ctx := newSyncFixture(t)

	if _, err := db.Sweep(ctx, store.SweepConfig{CommitsBefore: time.Now()}); err == nil {
		t.Error("sweep without an ops cut-off succeeded")
	}
	if _, err := db.Sweep(ctx, store.SweepConfig{OpsBefore: time.Now()}); err == nil {
		t.Error("sweep without a commits cut-off succeeded")
	}
}

func TestVacuumRuns(t *testing.T) {
	db, ctx := newSyncFixture(t)
	seedCommits(t, db, 5, time.Hour)

	if err := db.Vacuum(ctx); err != nil {
		t.Fatalf("Vacuum: %v", err)
	}
	// The data is still there afterwards.
	if rows := countRows(t, db, "commit_log"); rows != 5 {
		t.Errorf("commit_log rows after vacuum = %d, want 5", rows)
	}
}
