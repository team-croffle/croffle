package sqlite

import (
	"strconv"
	"testing"

	"github.com/team-croffle/croffle/apps/server/internal/store"
)

func snapshotQuery(limit int) store.SnapshotQuery {
	return store.SnapshotQuery{
		AccountID: testAccount,
		TeamIDs:   []string{store.DefaultTeamID},
		Limit:     limit,
	}
}

func TestSnapshotReturnsCurrentStateWithAPinnedInstant(t *testing.T) {
	db, ctx := newSyncFixture(t)

	if _, err := db.ApplyCommits(ctx, request(
		upsert("op-1", "row-1", 0, 1),
		upsert("op-2", "row-2", 0, 1),
	)); err != nil {
		t.Fatalf("push: %v", err)
	}
	// A second version of row-1: the snapshot is state, not history, so only
	// the latest is returned.
	if _, err := db.ApplyCommits(ctx, request(upsert("op-3", "row-1", 1, 2))); err != nil {
		t.Fatalf("second push: %v", err)
	}

	result, err := db.Snapshot(ctx, snapshotQuery(0))
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if len(result.Rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(result.Rows))
	}
	if result.ServerSeq != 3 {
		t.Errorf("serverSeq = %d, want the newest sequence", result.ServerSeq)
	}
	if result.HasMore {
		t.Error("hasMore on a complete snapshot")
	}

	for _, row := range result.Rows {
		if row.RowID == "row-1" && row.Version != 2 {
			t.Errorf("row-1 version = %d, want the latest (2)", row.Version)
		}
	}
}

func TestSnapshotPagesWithAStableInstant(t *testing.T) {
	db, ctx := newSyncFixture(t)

	commits := make([]store.Commit, 0, 5)
	for i := range 5 {
		id := strconv.Itoa(i)
		commits = append(commits, upsert("op-"+id, "row-"+id, 0, 1))
	}
	if _, err := db.ApplyCommits(ctx, request(commits...)); err != nil {
		t.Fatalf("push: %v", err)
	}

	first, err := db.Snapshot(ctx, snapshotQuery(2))
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if len(first.Rows) != 2 || !first.HasMore {
		t.Fatalf("first page = %+v", first)
	}
	base := first.ServerSeq

	// A change arriving mid-pagination must not appear in the snapshot: it is
	// after the pinned instant, and the client picks it up with a pull.
	if _, err := db.ApplyCommits(ctx, request(upsert("op-late", "row-late", 0, 1))); err != nil {
		t.Fatalf("late push: %v", err)
	}

	seen := map[string]bool{}
	for _, row := range first.Rows {
		seen[row.RowID] = true
	}

	last := first.Rows[len(first.Rows)-1]
	for {
		next := snapshotQuery(2)
		next.BaseSeq = base
		next.AfterTable = last.Table
		next.AfterRowID = last.RowID

		page, err := db.Snapshot(ctx, next)
		if err != nil {
			t.Fatalf("page: %v", err)
		}
		if page.ServerSeq != base {
			t.Fatalf("serverSeq = %d, want the pinned %d on every page", page.ServerSeq, base)
		}
		for _, row := range page.Rows {
			if seen[row.RowID] {
				t.Errorf("row %s appeared twice", row.RowID)
			}
			seen[row.RowID] = true
		}
		if !page.HasMore {
			break
		}
		last = page.Rows[len(page.Rows)-1]
	}

	if len(seen) != 5 {
		t.Errorf("snapshot covered %d rows, want exactly the 5 that existed at the instant: %v", len(seen), seen)
	}
	if seen["row-late"] {
		t.Error("a row created during pagination leaked into the snapshot")
	}
}

func TestSnapshotIncludesTombstones(t *testing.T) {
	db, ctx := newSyncFixture(t)

	if _, err := db.ApplyCommits(ctx, request(upsert("op-1", "row-1", 0, 1))); err != nil {
		t.Fatalf("seed: %v", err)
	}
	deletedAt := int64(1_700_000_009_000)
	if _, err := db.ApplyCommits(ctx, request(store.Commit{
		OpID: "op-2", Table: "schedule", RowID: "row-1", Op: store.OpDelete,
		BaseVersion: 1, Version: 2, UpdatedAtMs: deletedAt, DeletedAt: &deletedAt,
	})); err != nil {
		t.Fatalf("delete: %v", err)
	}

	result, err := db.Snapshot(ctx, snapshotQuery(0))
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if len(result.Rows) != 1 {
		t.Fatalf("rows = %+v, want the tombstone", result.Rows)
	}
	// A client that never saw the row has to learn it is deleted, or a stale
	// copy elsewhere would resurrect it on the next push.
	if !result.Rows[0].Deleted() {
		t.Error("the tombstone was dropped from the snapshot")
	}
}

func TestSnapshotIsScopedLikePull(t *testing.T) {
	db, ctx := newSyncFixture(t)
	newAccount(t, db, "acct-2", "bob")
	team := store.DefaultTeamID
	if err := db.AddTeamMember(ctx, team, "acct-2", "member"); err != nil {
		t.Fatalf("add bob: %v", err)
	}

	shared := upsert("op-shared", "row-shared", 0, 1)
	shared.TeamID = &team
	if _, err := db.ApplyCommits(ctx, store.ApplyRequest{
		AccountID: "acct-2", DeviceID: "client-2", TeamIDs: []string{team},
		Commits: []store.Commit{upsert("op-private", "row-private", 0, 1), shared},
	}); err != nil {
		t.Fatalf("bob's push: %v", err)
	}

	result, err := db.Snapshot(ctx, snapshotQuery(0))
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if len(result.Rows) != 1 || result.Rows[0].RowID != "row-shared" {
		t.Errorf("rows = %+v, want only the shared row", result.Rows)
	}
}

func TestSnapshotOnAnEmptyDatabase(t *testing.T) {
	db, ctx := newSyncFixture(t)

	result, err := db.Snapshot(ctx, snapshotQuery(0))
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if len(result.Rows) != 0 || result.HasMore || result.ServerSeq != 0 {
		t.Errorf("result = %+v, want an empty snapshot at sequence 0", result)
	}
}

func TestSnapshotRequiresAnAccount(t *testing.T) {
	db, ctx := newSyncFixture(t)

	if _, err := db.Snapshot(ctx, store.SnapshotQuery{}); err == nil {
		t.Error("snapshot without an account succeeded")
	}
}
