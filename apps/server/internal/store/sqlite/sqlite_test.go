package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/team-croffle/croffle/apps/server/internal/store"
)

func tempPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "server.db")
}

func open(t *testing.T, path string) *DB {
	t.Helper()
	db, err := Open(context.Background(), path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestOpenCreatesSchemaAndIsIdempotent(t *testing.T) {
	path := tempPath(t)

	db := open(t, path)
	if err := db.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}

	want := []string{"account", "applied_op", "commit_log", "device", "schema_migrations", "sync_meta", "sync_row", "team", "team_member"}
	got := tableNames(t, db)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("tables = %v, want %v", got, want)
	}

	// Server 0.1 has exactly one team, seeded by the first migration.
	var teams int
	if err := db.read.QueryRow("SELECT COUNT(*) FROM team WHERE id = ?", store.DefaultTeamID).Scan(&teams); err != nil {
		t.Fatalf("count teams: %v", err)
	}
	if teams != 1 {
		t.Errorf("default team rows = %d, want 1", teams)
	}

	// Reopening must not re-apply anything.
	if err := db.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	again := open(t, path)
	applied, err := again.migrate(context.Background())
	if err != nil {
		t.Fatalf("migrate (second run): %v", err)
	}
	if applied != 0 {
		t.Errorf("second migrate applied %d migrations, want 0", applied)
	}
	if teams := countRows(t, again, "team"); teams != 1 {
		t.Errorf("team rows after reopen = %d, want 1 (seed ran twice?)", teams)
	}
}

func TestPragmasApplyToEveryPooledConnection(t *testing.T) {
	db := open(t, tempPath(t))

	// Pragmas live in the DSN precisely so that this holds for connections the
	// pool creates later, not just for the first one.
	for i := range readPoolSize + 2 {
		var journal string
		if err := db.read.QueryRow("PRAGMA journal_mode").Scan(&journal); err != nil {
			t.Fatalf("read journal_mode (%d): %v", i, err)
		}
		if !strings.EqualFold(journal, "wal") {
			t.Fatalf("journal_mode = %q on connection %d, want wal", journal, i)
		}

		var fk int
		if err := db.read.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
			t.Fatalf("read foreign_keys (%d): %v", i, err)
		}
		if fk != 1 {
			t.Fatalf("foreign_keys = %d on connection %d, want 1", fk, i)
		}
	}
}

func TestWALFilesExistOnDisk(t *testing.T) {
	path := tempPath(t)
	db := open(t, path)

	// Force a write so the WAL is materialised.
	if err := db.withWriteTx(context.Background(), func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO team (id, name, createdAt) VALUES ('t2', 'second', 1)")
		return err
	}); err != nil {
		t.Fatalf("write: %v", err)
	}

	if _, err := os.Stat(path + "-wal"); err != nil {
		t.Errorf("no -wal file: %v", err)
	}
}

func TestForeignKeysAreEnforced(t *testing.T) {
	db := open(t, tempPath(t))

	err := db.withWriteTx(context.Background(), func(tx *sql.Tx) error {
		_, err := tx.Exec(`INSERT INTO device (id, accountId, name, createdAt) VALUES ('d1', 'ghost', 'laptop', 1)`)
		return err
	})
	if err == nil {
		t.Fatal("inserting a device for a missing account succeeded, want a foreign key error")
	}
}

func TestStrictTablesRejectWrongTypes(t *testing.T) {
	db := open(t, tempPath(t))

	err := db.withWriteTx(context.Background(), func(tx *sql.Tx) error {
		_, err := tx.Exec(`INSERT INTO team (id, name, createdAt) VALUES ('t3', 'third', 'not-a-number')`)
		return err
	})
	if err == nil {
		t.Fatal("inserting text into an INTEGER column succeeded, want a STRICT type error")
	}
}

// The serverSeq visibility contract: concurrent writers must produce a gapless,
// strictly increasing sequence, because a reader that saw a later number would
// move its cursor past an earlier one and lose it for good.
func TestConcurrentWritesProduceOrderedSequence(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()

	seedAccount(t, db, "acct")

	const (
		writers = 4
		perTx   = 25
	)

	var wg sync.WaitGroup
	errs := make(chan error, writers)

	for w := range writers {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := range perTx {
				err := db.withWriteTx(ctx, func(tx *sql.Tx) error {
					_, err := tx.ExecContext(ctx,
						`INSERT INTO commit_log (tbl, rowKey, op, ownerId, version, updatedAtMs, deviceId, createdAt)
						 VALUES ('schedule', ?, 'upsert', 'acct', 1, ?, 'dev', ?)`,
						fmtRowID(w, i), time.Now().UnixMilli(), time.Now().UnixMilli())
					return err
				})
				if err != nil {
					errs <- err
					return
				}
			}
		}(w)
	}

	wg.Wait()
	close(errs)
	for err := range errs {
		// SQLITE_BUSY would mean the single-writer policy leaked.
		t.Fatalf("concurrent write failed: %v", err)
	}

	rows, err := db.read.QueryContext(ctx, "SELECT serverSeq FROM commit_log ORDER BY serverSeq")
	if err != nil {
		t.Fatalf("read commit_log: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var seqs []int64
	for rows.Next() {
		var seq int64
		if err := rows.Scan(&seq); err != nil {
			t.Fatalf("scan: %v", err)
		}
		seqs = append(seqs, seq)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate: %v", err)
	}

	if len(seqs) != writers*perTx {
		t.Fatalf("commits = %d, want %d", len(seqs), writers*perTx)
	}
	for i, seq := range seqs {
		if want := int64(i + 1); seq != want {
			t.Fatalf("seq[%d] = %d, want %d (gap or reuse in the commit log)", i, seq, want)
		}
	}
}

// Reads must not queue behind the single write connection; otherwise a long
// push would stall every /health and pull.
func TestReadsRunDuringAWrite(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()

	writing := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)

	go func() {
		done <- db.withWriteTx(ctx, func(tx *sql.Tx) error {
			if _, err := tx.Exec("INSERT INTO team (id, name, createdAt) VALUES ('t4', 'held', 1)"); err != nil {
				return err
			}
			close(writing)
			<-release
			return nil
		})
	}()

	<-writing
	readCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var count int
	err := db.read.QueryRowContext(readCtx, "SELECT COUNT(*) FROM team").Scan(&count)
	close(release)

	if err != nil {
		t.Fatalf("read during write: %v", err)
	}
	// The open transaction is not visible yet, so the seeded team is all there is.
	if count != 1 {
		t.Errorf("team count during write = %d, want 1 (uncommitted row leaked)", count)
	}
	if err := <-done; err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestEditedMigrationIsRefused(t *testing.T) {
	path := tempPath(t)
	db := open(t, path)

	// Simulate a released migration whose file was changed afterwards.
	if _, err := db.write.Exec("UPDATE schema_migrations SET hash = 'deadbeef' WHERE version = 1"); err != nil {
		t.Fatalf("tamper: %v", err)
	}

	_, err := db.migrate(context.Background())
	if err == nil {
		t.Fatal("migrate accepted a changed migration, want an error")
	}
	if !strings.Contains(err.Error(), "never be edited") {
		t.Errorf("error = %v, want it to explain that migrations must not be edited", err)
	}
}

// Pins the hazard that cost a debugging round: SQLite identifiers are
// case-insensitive, so a column named rowId would answer to `rowid` and
// `ORDER BY rowid` on the commit log would sort by text instead of sequence —
// reordering the pull feed. The column is called rowKey for that reason; this
// test fails if anyone renames it back.
func TestCommitLogRowidIsTheSequence(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	seedAccount(t, db, "acct")

	if err := db.withWriteTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.Exec(
			`INSERT INTO commit_log (tbl, rowKey, op, ownerId, version, updatedAtMs, deviceId, createdAt)
			 VALUES ('schedule', 'not-a-number', 'upsert', 'acct', 1, 1, 'dev', 1)`)
		return err
	}); err != nil {
		t.Fatalf("insert: %v", err)
	}

	var rowid int64
	if err := db.read.QueryRow("SELECT rowid FROM commit_log LIMIT 1").Scan(&rowid); err != nil {
		t.Fatalf("rowid does not resolve to the sequence any more: %v", err)
	}
	if rowid != 1 {
		t.Errorf("rowid = %d, want 1 (the serverSeq)", rowid)
	}
}

func TestOpenRejectsEmptyPath(t *testing.T) {
	if _, err := Open(context.Background(), ""); err == nil {
		t.Fatal("Open(\"\") succeeded, want an error")
	}
}

func TestPendingOperationsReportNotImplemented(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()

	_, err := db.Sweep(ctx, store.SweepConfig{})
	if !errors.Is(err, store.ErrNotImplemented) {
		t.Errorf("Sweep error = %v, want ErrNotImplemented", err)
	}
	if !strings.Contains(err.Error(), "server-0.1.0#8") {
		t.Errorf("error = %v, want it to name the work item", err)
	}
}

func TestParseVersion(t *testing.T) {
	good := map[string]int64{"0001_init.sql": 1, "12_more.sql": 12}
	for name, want := range good {
		got, err := parseVersion(name)
		if err != nil {
			t.Errorf("parseVersion(%q): %v", name, err)
			continue
		}
		if got != want {
			t.Errorf("parseVersion(%q) = %d, want %d", name, got, want)
		}
	}

	for _, name := range []string{"init.sql", "abc_init.sql", "0_init.sql", "-1_x.sql"} {
		if _, err := parseVersion(name); err == nil {
			t.Errorf("parseVersion(%q) succeeded, want an error", name)
		}
	}
}

func tableNames(t *testing.T, db *DB) []string {
	t.Helper()
	rows, err := db.read.Query(
		"SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan table name: %v", err)
		}
		names = append(names, name)
	}
	return names
}

func countRows(t *testing.T, db *DB, table string) int {
	t.Helper()
	var n int
	if err := db.read.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

func seedAccount(t *testing.T, db *DB, id string) {
	t.Helper()
	if err := db.withWriteTx(context.Background(), func(tx *sql.Tx) error {
		_, err := tx.Exec(
			"INSERT INTO account (id, username, passwordHash, createdAt) VALUES (?, ?, 'x', 1)", id, id)
		return err
	}); err != nil {
		t.Fatalf("seed account: %v", err)
	}
}

func fmtRowID(writer, i int) string {
	return "row-" + string(rune('a'+writer)) + "-" + strconv.Itoa(i)
}
