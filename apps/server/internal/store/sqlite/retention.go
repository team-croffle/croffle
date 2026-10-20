package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/team-croffle/croffle/apps/server/internal/store"
)

// sweepBatch is how many rows one DELETE removes. The write connection is
// shared with every push, so the sweep yields between batches instead of
// holding the lock for the whole range.
const sweepBatch = 5000

// Sweep drops commit log and idempotency entries older than the cut-offs.
//
// sync_row is never swept: it is the current state, and a snapshot is built
// from it. Only history and idempotency keys expire.
func (db *DB) Sweep(ctx context.Context, cfg store.SweepConfig) (store.SweepResult, error) {
	var result store.SweepResult

	if cfg.CommitsBefore.IsZero() || cfg.OpsBefore.IsZero() {
		return result, errors.New("sqlite: sweep needs both cut-offs")
	}

	commitsBefore := toMillis(cfg.CommitsBefore)
	opsBefore := toMillis(cfg.OpsBefore)

	// The highest sequence that is about to go. Pull compares a client cursor
	// against this to decide whether that client can still be served.
	var purgeThrough *int64
	if err := db.read.QueryRowContext(ctx,
		`SELECT MAX(serverSeq) FROM commit_log WHERE createdAt < ?`, commitsBefore).Scan(&purgeThrough); err != nil {
		return result, fmt.Errorf("sqlite: find purge boundary: %w", err)
	}

	deletedCommits, err := db.sweepTable(ctx,
		`DELETE FROM commit_log WHERE serverSeq IN (
			SELECT serverSeq FROM commit_log WHERE createdAt < ? ORDER BY serverSeq LIMIT ?)`,
		commitsBefore)
	if err != nil {
		return result, err
	}
	result.DeletedCommits = deletedCommits

	deletedOps, err := db.sweepTable(ctx,
		`DELETE FROM applied_op WHERE opId IN (
			SELECT opId FROM applied_op WHERE createdAt < ? ORDER BY createdAt LIMIT ?)`,
		opsBefore)
	if err != nil {
		return result, err
	}
	result.DeletedOps = deletedOps

	// Expired dashboard sessions go too. They cannot authenticate anyway
	// (FindSession checks the time), so this is only housekeeping.
	deletedSessions, err := db.sweepTable(ctx,
		`DELETE FROM dash_session WHERE tokenHash IN (
			SELECT tokenHash FROM dash_session WHERE expiresAt < ? ORDER BY expiresAt LIMIT ?)`,
		toMillis(cfg.Now()))
	if err != nil {
		return result, err
	}
	result.DeletedSessions = deletedSessions

	if purgeThrough != nil {
		if err := db.raisePurgedThrough(ctx, *purgeThrough); err != nil {
			return result, err
		}
	}

	if err := db.read.QueryRowContext(ctx,
		`SELECT COALESCE(MIN(serverSeq), 0) FROM commit_log`).Scan(&result.OldestCommitSeq); err != nil {
		return result, fmt.Errorf("sqlite: read oldest sequence: %w", err)
	}

	if result.DeletedCommits+result.DeletedOps+result.DeletedSessions > 0 {
		// A large delete inflates the WAL; truncating it here keeps the file
		// from growing after every sweep.
		if _, err := db.write.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
			return result, fmt.Errorf("sqlite: checkpoint after sweep: %w", err)
		}
	}
	return result, nil
}

// sweepTable deletes in batches until nothing matches, so one sweep cannot hold
// the write lock long enough to stall a push.
func (db *DB) sweepTable(ctx context.Context, query string, before int64) (int64, error) {
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}

		var removed int64
		err := db.withWriteTx(ctx, func(tx *sql.Tx) error {
			res, err := tx.ExecContext(ctx, query, before, sweepBatch)
			if err != nil {
				return fmt.Errorf("sqlite: sweep: %w", err)
			}
			removed, err = res.RowsAffected()
			if err != nil {
				return fmt.Errorf("sqlite: sweep rows: %w", err)
			}
			return nil
		})
		if err != nil {
			return total, err
		}

		total += removed
		if removed < sweepBatch {
			return total, nil
		}
	}
}

// raisePurgedThrough moves the marker forward only. It must never go backwards:
// a lower value would tell a client its cursor is still serviceable when the
// commits it needs are already gone.
func (db *DB) raisePurgedThrough(ctx context.Context, seq int64) error {
	return db.withWriteTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO sync_meta (key, value) VALUES (?, ?)
			ON CONFLICT (key) DO UPDATE SET value = MAX(sync_meta.value, excluded.value)`,
			metaPurgedThroughSeq, seq)
		if err != nil {
			return fmt.Errorf("sqlite: raise purge marker: %w", err)
		}
		return nil
	})
}

// PurgedThroughSeq reports how far retention has deleted. GET /health exposes
// it, because it is what explains a 410 to an operator.
func (db *DB) PurgedThroughSeq(ctx context.Context) (int64, error) {
	return db.metaInt(ctx, metaPurgedThroughSeq)
}

// OldestCommitSeq is the first sequence still in the log.
func (db *DB) OldestCommitSeq(ctx context.Context) (int64, error) {
	var seq int64
	if err := db.read.QueryRowContext(ctx,
		`SELECT COALESCE(MIN(serverSeq), 0) FROM commit_log`).Scan(&seq); err != nil {
		return 0, fmt.Errorf("sqlite: read oldest sequence: %w", err)
	}
	return seq, nil
}

// Vacuum rebuilds the database file, reclaiming the space a sweep freed.
//
// It is never run automatically: VACUUM holds an exclusive lock for as long as
// it takes to copy the database, which would stall every sync. `croffle-server
// db vacuum` exists so an operator can choose the moment.
func (db *DB) Vacuum(ctx context.Context) error {
	if _, err := db.write.ExecContext(ctx, `VACUUM`); err != nil {
		return fmt.Errorf("sqlite: vacuum: %w", err)
	}
	return nil
}
