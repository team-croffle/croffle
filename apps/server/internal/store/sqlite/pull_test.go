package sqlite

import (
	"errors"
	"strconv"
	"testing"

	"github.com/team-croffle/croffle/apps/server/internal/store"
)

func pullQuery(since int64, limit int) store.PullQuery {
	return store.PullQuery{
		AccountID: testAccount,
		TeamIDs:   []string{store.DefaultTeamID},
		SinceSeq:  since,
		Limit:     limit,
	}
}

func TestPullOnAnEmptyLog(t *testing.T) {
	db, ctx := newSyncFixture(t)

	result, err := db.PullSince(ctx, pullQuery(0, 0))
	if err != nil {
		t.Fatalf("PullSince: %v", err)
	}
	if len(result.Changes) != 0 || result.HasMore {
		t.Errorf("result = %+v, want nothing", result)
	}
	// Never a sequence the client has not seen.
	if result.ServerSeq != 0 {
		t.Errorf("serverSeq = %d, want 0", result.ServerSeq)
	}
}

func TestPullReturnsOwnChangesIncludingThisDevices(t *testing.T) {
	db, ctx := newSyncFixture(t)

	if _, err := db.ApplyCommits(ctx, request(
		upsert("op-1", "row-1", 0, 1),
		upsert("op-2", "row-2", 0, 1),
	)); err != nil {
		t.Fatalf("push: %v", err)
	}

	result, err := db.PullSince(ctx, pullQuery(0, 0))
	if err != nil {
		t.Fatalf("PullSince: %v", err)
	}
	// The pushing device's own changes are returned too: filtering them would
	// hide a change from that account's other windows, and the client marks
	// applied rows as server-written anyway.
	if len(result.Changes) != 2 {
		t.Fatalf("changes = %d, want 2", len(result.Changes))
	}
	if result.ServerSeq != 2 {
		t.Errorf("serverSeq = %d, want 2", result.ServerSeq)
	}
	if result.Changes[0].ServerSeq != 1 || result.Changes[1].ServerSeq != 2 {
		t.Errorf("order = %d, %d; want ascending", result.Changes[0].ServerSeq, result.Changes[1].ServerSeq)
	}

	first := result.Changes[0]
	if first.OwnerID != testAccount || string(first.Payload) != `{"title":"standup"}` {
		t.Errorf("change = %+v", first)
	}

	// Pulling again with the advanced cursor yields nothing.
	empty, err := db.PullSince(ctx, pullQuery(result.ServerSeq, 0))
	if err != nil {
		t.Fatalf("second pull: %v", err)
	}
	if len(empty.Changes) != 0 {
		t.Errorf("changes after the cursor = %d, want 0", len(empty.Changes))
	}
}

func TestPullHidesOtherAccountsAndShowsSharedRows(t *testing.T) {
	db, ctx := newSyncFixture(t)
	newAccount(t, db, "acct-2", "bob")

	// Bob pushes one private row and one shared with the team.
	team := store.DefaultTeamID
	shared := upsert("op-shared", "row-shared", 0, 1)
	shared.TeamID = &team
	bob := store.ApplyRequest{
		AccountID: "acct-2", DeviceID: "client-2", TeamIDs: []string{team},
		Commits: []store.Commit{upsert("op-private", "row-private", 0, 1), shared},
	}
	if err := db.AddTeamMember(ctx, team, "acct-2", "member"); err != nil {
		t.Fatalf("add bob: %v", err)
	}
	if _, err := db.ApplyCommits(ctx, bob); err != nil {
		t.Fatalf("bob's push: %v", err)
	}

	result, err := db.PullSince(ctx, pullQuery(0, 0))
	if err != nil {
		t.Fatalf("PullSince: %v", err)
	}
	if len(result.Changes) != 1 {
		t.Fatalf("changes = %d, want only the shared row: %+v", len(result.Changes), result.Changes)
	}
	if result.Changes[0].RowID != "row-shared" {
		t.Errorf("row = %q, want the shared one", result.Changes[0].RowID)
	}

	// Without the membership the shared row is invisible as well.
	outsider := pullQuery(0, 0)
	outsider.TeamIDs = nil
	alone, err := db.PullSince(ctx, outsider)
	if err != nil {
		t.Fatalf("PullSince without teams: %v", err)
	}
	if len(alone.Changes) != 0 {
		t.Errorf("changes = %+v, want none", alone.Changes)
	}
}

func TestPullPagesAndAdvancesTheCursorExactly(t *testing.T) {
	db, ctx := newSyncFixture(t)

	commits := make([]store.Commit, 0, 5)
	for i := range 5 {
		commits = append(commits, upsert("op-"+strconv.Itoa(i), "row-"+strconv.Itoa(i), 0, 1))
	}
	if _, err := db.ApplyCommits(ctx, request(commits...)); err != nil {
		t.Fatalf("push: %v", err)
	}

	first, err := db.PullSince(ctx, pullQuery(0, 2))
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if len(first.Changes) != 2 || !first.HasMore {
		t.Fatalf("first page = %+v", first)
	}
	// The cursor is the last change actually returned, not the newest in the
	// log: anything else would skip commits the client never saw.
	if first.ServerSeq != 2 {
		t.Errorf("serverSeq = %d, want 2", first.ServerSeq)
	}

	second, err := db.PullSince(ctx, pullQuery(first.ServerSeq, 2))
	if err != nil {
		t.Fatalf("second page: %v", err)
	}
	if len(second.Changes) != 2 || !second.HasMore || second.ServerSeq != 4 {
		t.Fatalf("second page = %+v", second)
	}

	last, err := db.PullSince(ctx, pullQuery(second.ServerSeq, 2))
	if err != nil {
		t.Fatalf("last page: %v", err)
	}
	if len(last.Changes) != 1 || last.HasMore || last.ServerSeq != 5 {
		t.Fatalf("last page = %+v", last)
	}
}

func TestPullClampsTheLimit(t *testing.T) {
	db, ctx := newSyncFixture(t)

	// Asking for too much is clamped rather than refused: the client already
	// knows how to follow hasMore.
	if got := clampLimit(0, store.DefaultPullLimit, store.MaxPullLimit); got != store.DefaultPullLimit {
		t.Errorf("limit 0 -> %d, want the default", got)
	}
	if got := clampLimit(store.MaxPullLimit*10, store.DefaultPullLimit, store.MaxPullLimit); got != store.MaxPullLimit {
		t.Errorf("huge limit -> %d, want the maximum", got)
	}
	if got := clampLimit(7, store.DefaultPullLimit, store.MaxPullLimit); got != 7 {
		t.Errorf("limit 7 -> %d", got)
	}

	if _, err := db.PullSince(ctx, pullQuery(0, 1_000_000)); err != nil {
		t.Errorf("PullSince with an oversized limit: %v", err)
	}
}

func TestPullCarriesTombstones(t *testing.T) {
	db, ctx := newSyncFixture(t)

	if _, err := db.ApplyCommits(ctx, request(upsert("op-1", "row-1", 0, 1))); err != nil {
		t.Fatalf("seed: %v", err)
	}

	deletedAt := int64(1_700_000_005_000)
	remove := store.Commit{
		OpID: "op-2", Table: "schedule", RowID: "row-1", Op: store.OpDelete,
		BaseVersion: 1, Version: 2, UpdatedAtMs: deletedAt, DeletedAt: &deletedAt,
	}
	if _, err := db.ApplyCommits(ctx, request(remove)); err != nil {
		t.Fatalf("delete: %v", err)
	}

	result, err := db.PullSince(ctx, pullQuery(1, 0))
	if err != nil {
		t.Fatalf("PullSince: %v", err)
	}
	if len(result.Changes) != 1 {
		t.Fatalf("changes = %+v", result.Changes)
	}

	change := result.Changes[0]
	if change.Op != store.OpDelete {
		t.Errorf("op = %q, want delete", change.Op)
	}
	// The client applies the tombstone it was given; the server does not make
	// one up from updatedAtMs.
	if change.DeletedAt == nil || *change.DeletedAt != deletedAt {
		t.Errorf("deletedAt = %v, want %d", change.DeletedAt, deletedAt)
	}
}

func TestPullRejectsACursorThatWasSweptAway(t *testing.T) {
	db, ctx := newSyncFixture(t)

	if _, err := db.ApplyCommits(ctx, request(
		upsert("op-1", "row-1", 0, 1),
		upsert("op-2", "row-2", 0, 1),
	)); err != nil {
		t.Fatalf("push: %v", err)
	}

	// Retention has deleted everything up to sequence 5 (work item #8 is what
	// sets this in production).
	if _, err := db.write.Exec(`UPDATE sync_meta SET value = 5 WHERE key = ?`, metaPurgedThroughSeq); err != nil {
		t.Fatalf("set purged marker: %v", err)
	}

	_, err := db.PullSince(ctx, pullQuery(2, 0))
	if !errors.Is(err, store.ErrCursorTooOld) {
		t.Fatalf("error = %v, want ErrCursorTooOld", err)
	}

	// A cursor at or past the purge point is still serviceable.
	if _, err := db.PullSince(ctx, pullQuery(5, 0)); err != nil {
		t.Errorf("cursor at the purge point: %v", err)
	}
}

// A cursor past everything this server ever issued usually means the client was
// pointed at a different server or a restored backup; bootstrapping is the only
// honest recovery.
func TestPullRejectsACursorFromTheFuture(t *testing.T) {
	db, ctx := newSyncFixture(t)

	if _, err := db.ApplyCommits(ctx, request(upsert("op-1", "row-1", 0, 1))); err != nil {
		t.Fatalf("push: %v", err)
	}

	_, err := db.PullSince(ctx, pullQuery(99, 0))
	if !errors.Is(err, store.ErrCursorTooOld) {
		t.Errorf("error = %v, want ErrCursorTooOld", err)
	}
}

func TestPullFromZeroIsAlwaysAllowed(t *testing.T) {
	db, ctx := newSyncFixture(t)

	if _, err := db.write.Exec(`UPDATE sync_meta SET value = 42 WHERE key = ?`, metaPurgedThroughSeq); err != nil {
		t.Fatalf("set purged marker: %v", err)
	}

	// A fresh client asks from 0 and gets whatever is left; bootstrap fills in
	// the rest.
	if _, err := db.PullSince(ctx, pullQuery(0, 0)); err != nil {
		t.Errorf("PullSince from 0: %v", err)
	}
}

func TestPullRequiresAnAccount(t *testing.T) {
	db, ctx := newSyncFixture(t)

	if _, err := db.PullSince(ctx, store.PullQuery{}); err == nil {
		t.Error("pull without an account succeeded")
	}
}

func TestVisibilityBindsTeamIDsAsParameters(t *testing.T) {
	// The IN list is built from the length of the slice, never from its
	// contents, so a team id can never become SQL.
	where, args := visibility("acct-1", []string{"t1", "t2"})
	if where != `ownerId = ? OR teamId IN (?, ?)` {
		t.Errorf("where = %q", where)
	}
	if len(args) != 3 {
		t.Errorf("args = %v, want the account and both teams", args)
	}

	where, args = visibility("acct-1", nil)
	if where != `ownerId = ?` || len(args) != 1 {
		t.Errorf("without teams: where = %q, args = %v", where, args)
	}

	injected := "t1') OR 1=1 --"
	_, args = visibility("acct-1", []string{injected})
	if args[1] != injected {
		t.Errorf("args = %v, want the raw string bound as a parameter", args)
	}
}
