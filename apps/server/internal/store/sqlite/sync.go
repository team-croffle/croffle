package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/team-croffle/croffle/apps/server/internal/store"
)

// ApplyCommits applies one push batch.
//
// The whole batch is one transaction: partial success is expressed in the
// result (accepted plus conflicts), not by leaving half of it written. That
// also keeps the commit log contiguous, which the cursor depends on.
func (db *DB) ApplyCommits(ctx context.Context, req store.ApplyRequest) (store.ApplyResult, error) {
	result := store.ApplyResult{
		Accepted:  []string{},
		Conflicts: []store.Conflict{},
	}
	if req.AccountID == "" || req.DeviceID == "" {
		return result, errors.New("sqlite: apply needs an account and a device")
	}

	now := req.Now
	if now.IsZero() {
		now = timeNow()
	}
	nowMs := toMillis(now)

	err := db.withWriteTx(ctx, func(tx *sql.Tx) error {
		for _, commit := range req.Commits {
			if commit.TeamID != nil && !slices.Contains(req.TeamIDs, *commit.TeamID) {
				// The handler checks this too; refusing here as well means a
				// future caller cannot skip it.
				return fmt.Errorf("%w: %s", store.ErrForeignTeam, *commit.TeamID)
			}

			applied, err := alreadyApplied(ctx, tx, commit.OpID)
			if err != nil {
				return err
			}
			if applied {
				// A retry of a push whose response was lost. Report success
				// without touching the row: that is what the idempotency key
				// is for.
				result.Accepted = append(result.Accepted, commit.OpID)
				continue
			}

			current, err := currentRow(ctx, tx, commit.Table, commit.RowID)
			if err != nil {
				return err
			}

			if current != nil && current.Version != commit.BaseVersion {
				// The server moved on. Hand the row back and let the client
				// settle it; the server does not pick a winner in 0.1.
				result.Conflicts = append(result.Conflicts, store.Conflict{
					OpID:      commit.OpID,
					Table:     commit.Table,
					RowID:     commit.RowID,
					ServerRow: current,
				})
				// Deliberately not recorded in applied_op: recording a commit
				// that was not applied would make the client's retry a no-op
				// and lose the change for good.
				continue
			}

			seq, err := appendCommit(ctx, tx, req, commit, nowMs)
			if err != nil {
				return err
			}
			if err := upsertRow(ctx, tx, req.AccountID, commit, seq, nowMs); err != nil {
				return err
			}
			if err := recordOp(ctx, tx, commit.OpID, req.DeviceID, seq, nowMs); err != nil {
				return err
			}

			result.Accepted = append(result.Accepted, commit.OpID)
			result.ServerSeq = seq
		}
		return nil
	})
	if err != nil {
		return store.ApplyResult{}, err
	}

	if result.ServerSeq == 0 {
		// Nothing was written (everything conflicted or was a retry); report
		// where the log currently stands so the client can still advance.
		seq, err := db.maxServerSeq(ctx)
		if err != nil {
			return store.ApplyResult{}, err
		}
		result.ServerSeq = seq
	}
	return result, nil
}

func alreadyApplied(ctx context.Context, tx *sql.Tx, opID string) (bool, error) {
	var one int
	err := tx.QueryRowContext(ctx, `SELECT 1 FROM applied_op WHERE opId = ?`, opID).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("sqlite: look up op %s: %w", opID, err)
	}
	return true, nil
}

func currentRow(ctx context.Context, tx *sql.Tx, table, rowID string) (*store.Row, error) {
	var (
		row       store.Row
		payload   string
		deletedAt *int64
	)
	err := tx.QueryRowContext(ctx, `
		SELECT tbl, rowKey, ownerId, teamId, version, updatedAtMs, lastWriterId, deletedAt, payload, serverSeq
		FROM sync_row WHERE tbl = ? AND rowKey = ?`, table, rowID).Scan(
		&row.Table, &row.RowID, &row.OwnerID, &row.TeamID, &row.Version,
		&row.UpdatedAtMs, &row.LastWriterID, &deletedAt, &payload, &row.ServerSeq)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("sqlite: read row %s/%s: %w", table, rowID, err)
	}

	row.DeletedAt = deletedAt
	row.Payload = json.RawMessage(payload)
	return &row, nil
}

// appendCommit writes the log entry and returns its sequence. The sequence is
// allocated here, inside the batch transaction, which is what the visibility
// contract in store.Store requires.
func appendCommit(ctx context.Context, tx *sql.Tx, req store.ApplyRequest, commit store.Commit, nowMs int64) (int64, error) {
	res, err := tx.ExecContext(ctx, `
		INSERT INTO commit_log
			(tbl, rowKey, op, ownerId, teamId, version, updatedAtMs, lastWriterId, payload, deviceId, createdAt)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		commit.Table, commit.RowID, string(commit.Op), req.AccountID, commit.TeamID,
		commit.Version, commit.UpdatedAtMs, nullString(commit.LastWriterID),
		payloadOrNil(commit.Payload), req.DeviceID, nowMs)
	if err != nil {
		return 0, fmt.Errorf("sqlite: append commit %s: %w", commit.OpID, err)
	}

	seq, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("sqlite: read commit sequence: %w", err)
	}
	return seq, nil
}

func upsertRow(ctx context.Context, tx *sql.Tx, accountID string, commit store.Commit, seq, nowMs int64) error {
	deletedAt := commit.DeletedAt
	if commit.Op == store.OpDelete && deletedAt == nil {
		// A delete without a tombstone timestamp would be indistinguishable
		// from a live row; stamp it rather than storing something ambiguous.
		ms := nowMs
		deletedAt = &ms
	}

	payload := commit.Payload
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}

	_, err := tx.ExecContext(ctx, `
		INSERT INTO sync_row
			(tbl, rowKey, ownerId, teamId, version, updatedAtMs, lastWriterId, deletedAt, payload, serverSeq)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (tbl, rowKey) DO UPDATE SET
			-- ownerId is never taken from the client: a row must stay with the
			-- account that owns it.
			teamId       = excluded.teamId,
			version      = excluded.version,
			updatedAtMs  = excluded.updatedAtMs,
			lastWriterId = excluded.lastWriterId,
			deletedAt    = excluded.deletedAt,
			payload      = excluded.payload,
			serverSeq    = excluded.serverSeq`,
		commit.Table, commit.RowID, accountID, commit.TeamID, commit.Version,
		commit.UpdatedAtMs, nullString(commit.LastWriterID), deletedAt, string(payload), seq)
	if err != nil {
		return fmt.Errorf("sqlite: upsert row %s/%s: %w", commit.Table, commit.RowID, err)
	}
	return nil
}

func recordOp(ctx context.Context, tx *sql.Tx, opID, deviceID string, seq, nowMs int64) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO applied_op (opId, deviceId, serverSeq, createdAt) VALUES (?, ?, ?, ?)`,
		opID, deviceID, seq, nowMs)
	if err != nil {
		return fmt.Errorf("sqlite: record op %s: %w", opID, err)
	}
	return nil
}

func (db *DB) maxServerSeq(ctx context.Context) (int64, error) {
	var seq int64
	if err := db.read.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(serverSeq), 0) FROM commit_log`).Scan(&seq); err != nil {
		return 0, fmt.Errorf("sqlite: read max sequence: %w", err)
	}
	return seq, nil
}

func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func payloadOrNil(payload json.RawMessage) any {
	if len(payload) == 0 {
		return nil
	}
	return string(payload)
}
