package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

const migrationsDir = "migrations"

// migration is one embedded .sql file, named <version>_<slug>.sql.
type migration struct {
	version int64
	name    string
	sql     string
	hash    string
}

// migrate applies every migration that is not recorded yet and returns how many
// were applied.
//
// Each file runs in its own transaction, so a failure leaves the database at
// the previous version instead of half-way through.
func (db *DB) migrate(ctx context.Context) (int, error) {
	migrations, err := loadMigrations()
	if err != nil {
		return 0, err
	}

	if err := db.ensureMigrationsTable(ctx); err != nil {
		return 0, err
	}

	applied, err := db.appliedMigrations(ctx)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, m := range migrations {
		if got, ok := applied[m.version]; ok {
			// An already applied migration whose content changed means the
			// database and the code disagree about what the schema is. Editing
			// a released migration is the mistake; refuse to guess.
			if got != m.hash {
				return count, fmt.Errorf(
					"migration %d (%s) was applied with a different content (recorded %s, found %s): migrations must never be edited, add a new one",
					m.version, m.name, short(got), short(m.hash))
			}
			continue
		}

		if err := db.applyMigration(ctx, m); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (db *DB) ensureMigrationsTable(ctx context.Context) error {
	const ddl = `CREATE TABLE IF NOT EXISTS schema_migrations (
		version   INTEGER NOT NULL PRIMARY KEY,
		name      TEXT    NOT NULL,
		hash      TEXT    NOT NULL,
		appliedAt INTEGER NOT NULL
	) STRICT`

	if _, err := db.write.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	return nil
}

func (db *DB) appliedMigrations(ctx context.Context) (map[int64]string, error) {
	rows, err := db.write.QueryContext(ctx, "SELECT version, hash FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("read schema_migrations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	applied := map[int64]string{}
	for rows.Next() {
		var (
			version int64
			hash    string
		)
		if err := rows.Scan(&version, &hash); err != nil {
			return nil, fmt.Errorf("scan schema_migrations: %w", err)
		}
		applied[version] = hash
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate schema_migrations: %w", err)
	}
	return applied, nil
}

func (db *DB) applyMigration(ctx context.Context, m migration) error {
	err := db.withWriteTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, m.sql); err != nil {
			return fmt.Errorf("apply migration %d (%s): %w", m.version, m.name, err)
		}
		_, err := tx.ExecContext(ctx,
			"INSERT INTO schema_migrations (version, name, hash, appliedAt) VALUES (?, ?, ?, ?)",
			m.version, m.name, m.hash, time.Now().UnixMilli())
		if err != nil {
			return fmt.Errorf("record migration %d: %w", m.version, err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func loadMigrations() ([]migration, error) {
	entries, err := fs.ReadDir(migrationsFS, migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("read embedded migrations: %w", err)
	}

	migrations := make([]migration, 0, len(entries))
	seen := map[int64]string{}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		version, err := parseVersion(entry.Name())
		if err != nil {
			return nil, err
		}
		if other, dup := seen[version]; dup {
			return nil, fmt.Errorf("duplicate migration version %d (%s and %s)", version, other, entry.Name())
		}
		seen[version] = entry.Name()

		body, err := migrationsFS.ReadFile(path.Join(migrationsDir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		sum := sha256.Sum256(body)

		migrations = append(migrations, migration{
			version: version,
			name:    entry.Name(),
			sql:     string(body),
			hash:    hex.EncodeToString(sum[:]),
		})
	}

	if len(migrations) == 0 {
		return nil, fmt.Errorf("no migrations embedded under %s/", migrationsDir)
	}

	slices.SortFunc(migrations, func(a, b migration) int {
		return int(a.version - b.version)
	})
	return migrations, nil
}

// parseVersion reads the leading number of "0001_init.sql".
func parseVersion(name string) (int64, error) {
	prefix, _, ok := strings.Cut(strings.TrimSuffix(name, ".sql"), "_")
	if !ok || prefix == "" {
		return 0, fmt.Errorf("migration %q: want <version>_<name>.sql", name)
	}
	version, err := strconv.ParseInt(prefix, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("migration %q: version %q is not a number", name, prefix)
	}
	if version <= 0 {
		return 0, fmt.Errorf("migration %q: version must be positive", name)
	}
	return version, nil
}

func short(hash string) string {
	if len(hash) <= 12 {
		return hash
	}
	return hash[:12]
}
