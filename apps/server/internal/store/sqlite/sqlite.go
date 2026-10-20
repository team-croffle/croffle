// Package sqlite implements store.Store on a single SQLite file.
//
// Two handles are opened on the same file: a pooled reader and a writer limited
// to one connection. The single writer is not a performance compromise, it is
// how the serverSeq visibility contract in store.Store is met — commits can
// never interleave, so a cursor never skips a sequence number.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // database/sql driver, registered as "sqlite"

	"github.com/team-croffle/croffle/apps/server/internal/store"
)

const driverName = "sqlite"

// readPoolSize bounds concurrent readers. WAL lets them run while a write is in
// flight, so this only caps memory and file descriptors.
const readPoolSize = 8

// busyTimeoutMS is how long a connection waits for a lock before giving up.
// With a single writer this should never be reached from inside the server; it
// covers the CLI touching the same file as a running server.
const busyTimeoutMS = 5000

// DB is the SQLite-backed store.
type DB struct {
	path  string
	read  *sql.DB
	write *sql.DB
	// AppliedMigrations counts the migrations Open had to apply, so the caller
	// can log "upgraded the schema" instead of staying silent about it.
	AppliedMigrations int
	closed            bool
}

// Compile-time check that this package still satisfies the boundary.
var _ store.Store = (*DB)(nil)

// Open opens (and creates, if missing) the database at path and applies every
// pending migration.
func Open(ctx context.Context, path string) (*DB, error) {
	if path == "" {
		return nil, errors.New("sqlite: empty database path")
	}

	read, err := sql.Open(driverName, dsn(path, false))
	if err != nil {
		return nil, fmt.Errorf("open read handle: %w", err)
	}
	read.SetMaxOpenConns(readPoolSize)
	read.SetMaxIdleConns(readPoolSize)

	write, err := sql.Open(driverName, dsn(path, true))
	if err != nil {
		_ = read.Close()
		return nil, fmt.Errorf("open write handle: %w", err)
	}
	// One connection: this is the serialisation point.
	write.SetMaxOpenConns(1)
	write.SetMaxIdleConns(1)
	// A pooled connection that is recycled loses nothing (pragmas live in the
	// DSN), but keeping it alive avoids re-running them on every batch.
	write.SetConnMaxLifetime(0)

	db := &DB{path: path, read: read, write: write}

	if err := db.write.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("connect to %s (does %s exist and is it writable?): %w",
			path, filepath.Dir(path), err)
	}

	applied, err := db.migrate(ctx)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	db.AppliedMigrations = applied

	return db, nil
}

// dsn builds the connection string.
//
// Pragmas belong in the DSN, not in a post-connect Exec: database/sql pools
// connections, so an Exec'd pragma would apply to whichever connection
// happened to run it and silently not to the others.
//
// The writer additionally uses _txlock=immediate so a write transaction takes
// the write lock at BEGIN. Without it a transaction that reads first and writes
// later has to upgrade its lock mid-flight, which is the classic source of
// SQLITE_BUSY under concurrency.
func dsn(path string, writer bool) string {
	q := url.Values{}
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", fmt.Sprintf("busy_timeout(%d)", busyTimeoutMS))
	q.Add("_pragma", "foreign_keys(1)")
	// NORMAL is the documented companion of WAL: a crash can lose the last
	// transactions but cannot corrupt the file. FULL would fsync per commit.
	q.Add("_pragma", "synchronous(NORMAL)")
	if writer {
		q.Set("_txlock", "immediate")
	}
	return "file:" + filepath.ToSlash(path) + "?" + q.Encode()
}

// Path returns the database file path.
func (db *DB) Path() string { return db.path }

// Ping reports whether the file is reachable and readable.
func (db *DB) Ping(ctx context.Context) error {
	var one int
	if err := db.read.QueryRowContext(ctx, "SELECT 1").Scan(&one); err != nil {
		return fmt.Errorf("sqlite ping: %w", err)
	}
	return nil
}

// Close releases both handles.
func (db *DB) Close() error {
	if db.closed {
		return nil
	}
	db.closed = true
	return errors.Join(db.write.Close(), db.read.Close())
}

// write runs fn inside a single write transaction on the one write connection.
// Every mutation in this package goes through here; that is what keeps commit
// sequence numbers ordered.
func (db *DB) withWriteTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := db.write.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin write tx: %w", err)
	}
	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			return errors.Join(err, fmt.Errorf("rollback: %w", rbErr))
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit write tx: %w", err)
	}
	return nil
}

// timeNow is a package-level seam so bookkeeping writes can be made
// deterministic in tests without threading a clock through every call.
var timeNow = time.Now

// toMillis converts a time to the unix milliseconds used by every timestamp
// column. The zero time becomes 0 so a forgotten clock is obvious rather than
// a date in 1754.
func toMillis(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}

func fromMillis(ms int64) time.Time {
	return time.UnixMilli(ms).UTC()
}

func fromMillisPtr(ms *int64) *time.Time {
	if ms == nil {
		return nil
	}
	t := fromMillis(*ms)
	return &t
}

func toMillisPtr(t *time.Time) *int64 {
	if t == nil {
		return nil
	}
	ms := toMillis(*t)
	return &ms
}
