package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/team-croffle/croffle/apps/server/internal/store"
)

// metaPurgedThroughSeq is the highest commit sequence retention has deleted.
const metaPurgedThroughSeq = "purgedThroughSeq"

// PullSince returns the commits after a cursor that the account may see.
//
// Visibility is fixed: the account's own rows plus the rows shared with a team
// it belongs to. Clients do not send scopes in 0.1; a per-scope cursor arrives
// with multi-team support in 0.3.
func (db *DB) PullSince(ctx context.Context, query store.PullQuery) (store.PullResult, error) {
	if query.AccountID == "" {
		return store.PullResult{}, errors.New("sqlite: pull needs an account")
	}

	limit := clampLimit(query.Limit, store.DefaultPullLimit, store.MaxPullLimit)

	if err := db.checkCursor(ctx, query.SinceSeq); err != nil {
		return store.PullResult{}, err
	}

	where, args := visibility(query.AccountID, query.TeamIDs)
	args = append([]any{query.SinceSeq}, args...)
	// One row more than asked for: that is how hasMore is answered exactly,
	// without a second counting query.
	args = append(args, limit+1)

	rows, err := db.read.QueryContext(ctx, `
		SELECT serverSeq, tbl, rowKey, op, ownerId, teamId, version, updatedAtMs, lastWriterId, deletedAt, payload
		FROM commit_log
		WHERE serverSeq > ? AND (`+where+`)
		ORDER BY serverSeq
		LIMIT ?`, args...)
	if err != nil {
		return store.PullResult{}, fmt.Errorf("sqlite: pull: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := store.PullResult{Changes: []store.Change{}, ServerSeq: query.SinceSeq}
	for rows.Next() {
		if len(result.Changes) == limit {
			result.HasMore = true
			break
		}

		change, err := scanChange(rows)
		if err != nil {
			return store.PullResult{}, err
		}
		result.Changes = append(result.Changes, change)
		// Only ever a sequence that is actually in this response: a cursor
		// moved past an unreturned commit would lose it permanently.
		result.ServerSeq = change.ServerSeq
	}
	if err := rows.Err(); err != nil {
		return store.PullResult{}, fmt.Errorf("sqlite: iterate commits: %w", err)
	}
	return result, nil
}

// checkCursor reports whether the commits this cursor still needs are gone.
func (db *DB) checkCursor(ctx context.Context, sinceSeq int64) error {
	if sinceSeq <= 0 {
		// A fresh client asks from 0 and gets whatever is left; bootstrap is
		// what fills in the rest.
		return nil
	}

	purged, err := db.metaInt(ctx, metaPurgedThroughSeq)
	if err != nil {
		return err
	}
	if sinceSeq < purged {
		return fmt.Errorf("%w: cursor %d, retained from %d", store.ErrCursorTooOld, sinceSeq, purged+1)
	}

	newest, err := db.maxServerSeq(ctx)
	if err != nil {
		return err
	}
	if sinceSeq > max(newest, purged) {
		// A cursor past everything this server ever issued: usually a client
		// pointed at a different server, or a restored backup. Bootstrapping is
		// the only honest recovery.
		return fmt.Errorf("%w: cursor %d is beyond the newest commit %d",
			store.ErrCursorTooOld, sinceSeq, newest)
	}
	return nil
}

func (db *DB) metaInt(ctx context.Context, key string) (int64, error) {
	var value int64
	err := db.read.QueryRowContext(ctx, `SELECT value FROM sync_meta WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("sqlite: read meta %s: %w", key, err)
	}
	return value, nil
}

// visibility builds the ownership filter: own rows, plus rows shared with a
// team this account is in.
func visibility(accountID string, teamIDs []string) (string, []any) {
	args := []any{accountID}
	if len(teamIDs) == 0 {
		return `ownerId = ?`, args
	}

	placeholders := make([]string, len(teamIDs))
	for i, team := range teamIDs {
		placeholders[i] = "?"
		args = append(args, team)
	}
	// Built from the length of the slice, never from its contents, so the team
	// ids stay bound parameters.
	return `ownerId = ? OR teamId IN (` + strings.Join(placeholders, ", ") + `)`, args
}

func clampLimit(requested, fallback, maximum int) int {
	if requested <= 0 {
		return fallback
	}
	if requested > maximum {
		// Clamped rather than rejected: a client asking for too much gets a
		// smaller page and hasMore, which it already knows how to follow.
		return maximum
	}
	return requested
}

func scanChange(rows *sql.Rows) (store.Change, error) {
	var (
		change    store.Change
		op        string
		ownerID   string
		deletedAt *int64
		payload   *string
	)
	if err := rows.Scan(&change.ServerSeq, &change.Table, &change.RowID, &op, &ownerID,
		&change.TeamID, &change.Version, &change.UpdatedAtMs, &change.LastWriterID,
		&deletedAt, &payload); err != nil {
		return store.Change{}, fmt.Errorf("sqlite: scan commit: %w", err)
	}

	change.Op = store.Op(op)
	change.OwnerID = ownerID
	change.DeletedAt = deletedAt
	if payload != nil {
		change.Payload = json.RawMessage(*payload)
	}
	return change, nil
}
