package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/team-croffle/croffle/apps/server/internal/store"
)

// Snapshot returns current state instead of history, for a device that is new
// or whose cursor fell outside the retention window.
//
// Every page is pinned to one instant. The first page picks it (the newest
// sequence at that moment) and reports it back; later pages must pass it in
// again. Re-reading the instant per page would leave changes made during
// pagination neither in the snapshot nor after it — invisible until something
// else touched the row.
func (db *DB) Snapshot(ctx context.Context, query store.SnapshotQuery) (store.SnapshotResult, error) {
	if query.AccountID == "" {
		return store.SnapshotResult{}, errors.New("sqlite: snapshot needs an account")
	}

	baseSeq := query.BaseSeq
	if baseSeq <= 0 {
		newest, err := db.maxServerSeq(ctx)
		if err != nil {
			return store.SnapshotResult{}, err
		}
		baseSeq = newest
	}

	limit := clampLimit(query.Limit, store.DefaultSnapshotLimit, store.MaxSnapshotLimit)

	where, args := visibility(query.AccountID, query.TeamIDs)
	args = append(args, baseSeq, query.AfterTable, query.AfterTable, query.AfterRowID, limit+1)

	// Keyset pagination on the primary key, not OFFSET: with OFFSET a row
	// inserted or removed between pages shifts every later page.
	rows, err := db.read.QueryContext(ctx, `
		SELECT tbl, rowKey, ownerId, teamId, version, updatedAtMs, lastWriterId, deletedAt, payload, serverSeq
		FROM sync_row
		WHERE (`+where+`)
		  AND serverSeq <= ?
		  AND (tbl > ? OR (tbl = ? AND rowKey > ?))
		ORDER BY tbl, rowKey
		LIMIT ?`, args...)
	if err != nil {
		return store.SnapshotResult{}, fmt.Errorf("sqlite: snapshot: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := store.SnapshotResult{Rows: []store.Row{}, ServerSeq: baseSeq}
	for rows.Next() {
		if len(result.Rows) == limit {
			result.HasMore = true
			break
		}

		var (
			row       store.Row
			deletedAt *int64
			payload   string
		)
		if err := rows.Scan(&row.Table, &row.RowID, &row.OwnerID, &row.TeamID, &row.Version,
			&row.UpdatedAtMs, &row.LastWriterID, &deletedAt, &payload, &row.ServerSeq); err != nil {
			return store.SnapshotResult{}, fmt.Errorf("sqlite: scan snapshot row: %w", err)
		}
		row.DeletedAt = deletedAt
		row.Payload = json.RawMessage(payload)

		// Tombstones are included: a client that never saw the row has to learn
		// that it is deleted, or a stale copy on another device would be
		// resurrected on its next push.
		result.Rows = append(result.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return store.SnapshotResult{}, fmt.Errorf("sqlite: iterate snapshot: %w", err)
	}
	return result, nil
}
