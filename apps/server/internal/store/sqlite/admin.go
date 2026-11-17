package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"

	"github.com/team-croffle/croffle/apps/server/internal/store"
)

// Compile-time check that the operator surface is complete too.
var _ store.Admin = (*DB)(nil)

func (db *DB) ListAccounts(ctx context.Context) ([]store.Account, error) {
	rows, err := db.read.QueryContext(ctx,
		`SELECT `+accountColumns+` FROM account ORDER BY username`)
	if err != nil {
		return nil, fmt.Errorf("sqlite: list accounts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	accounts := []store.Account{}
	for rows.Next() {
		account, err := scanAccountRow(rows)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, *account)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: iterate accounts: %w", err)
	}
	return accounts, nil
}

func (db *DB) SetAccountDisabled(ctx context.Context, accountID string, disabled, force bool) error {
	if !disabled {
		return db.updateOne(ctx, "account", accountID,
			`UPDATE account SET disabledAt = NULL WHERE id = ?`, accountID)
	}
	return db.withWriteTx(ctx, func(tx *sql.Tx) error {
		if !force {
			if err := guardLastOperator(ctx, tx, accountID); err != nil {
				return err
			}
		}
		return execOne(ctx, tx, "account", accountID,
			`UPDATE account SET disabledAt = COALESCE(disabledAt, ?) WHERE id = ?`, toMillis(timeNow()), accountID)
	})
}

func (db *DB) SetOperator(ctx context.Context, accountID string, operator, force bool) error {
	if operator {
		return db.updateOne(ctx, "account", accountID,
			`UPDATE account SET operatorAt = COALESCE(operatorAt, ?) WHERE id = ?`, toMillis(timeNow()), accountID)
	}
	return db.withWriteTx(ctx, func(tx *sql.Tx) error {
		if !force {
			if err := guardLastOperator(ctx, tx, accountID); err != nil {
				return err
			}
		}
		return execOne(ctx, tx, "account", accountID,
			`UPDATE account SET operatorAt = NULL WHERE id = ?`, accountID)
	})
}

// guardLastOperator returns ErrLastOperator when accountID is an active
// operator and no other active operator exists. It runs in the write
// transaction, so two requests cannot each remove "the other" operator.
func guardLastOperator(ctx context.Context, tx *sql.Tx, accountID string) error {
	var isActiveOperator bool
	err := tx.QueryRowContext(ctx,
		`SELECT operatorAt IS NOT NULL AND disabledAt IS NULL FROM account WHERE id = ?`, accountID).Scan(&isActiveOperator)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: account %s", store.ErrNotFound, accountID)
	}
	if err != nil {
		return fmt.Errorf("sqlite: read operator: %w", err)
	}
	if !isActiveOperator {
		return nil
	}
	var others int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM account
		WHERE operatorAt IS NOT NULL AND disabledAt IS NULL AND id <> ?`, accountID).Scan(&others); err != nil {
		return fmt.Errorf("sqlite: count operators: %w", err)
	}
	if others == 0 {
		return store.ErrLastOperator
	}
	return nil
}

// execOne runs an UPDATE that must hit exactly one row, inside tx.
func execOne(ctx context.Context, tx *sql.Tx, what, id, query string, args ...any) error {
	res, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("sqlite: update %s: %w", what, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("sqlite: update %s rows: %w", what, err)
	}
	if affected == 0 {
		return fmt.Errorf("%w: %s %s", store.ErrNotFound, what, id)
	}
	return nil
}

func (db *DB) SetAccountPassword(ctx context.Context, accountID, passwordHash string) error {
	if passwordHash == "" {
		return errors.New("sqlite: empty password hash")
	}
	return db.updateOne(ctx, "account", accountID,
		`UPDATE account SET passwordHash = ? WHERE id = ?`, passwordHash, accountID)
}

// RevokeDevice withdraws the device's token by clearing the hash as well as
// stamping revokedAt: a revoked row keeps no credential material at all. Its
// access and refresh tokens are revoked in the same transaction.
func (db *DB) RevokeDevice(ctx context.Context, deviceID string) error {
	return db.revokeDevice(ctx, deviceID,
		`UPDATE device SET revokedAt = ?, tokenHash = NULL WHERE id = ?`, deviceID)
}

// RevokeAccountDevice revokes a device only if it belongs to accountID. A
// device of another account is reported as ErrNotFound, the same as one that
// does not exist, so the dashboard cannot be used to probe device ids.
func (db *DB) RevokeAccountDevice(ctx context.Context, accountID, deviceID string) error {
	return db.revokeDevice(ctx, deviceID,
		`UPDATE device SET revokedAt = ?, tokenHash = NULL WHERE id = ? AND accountId = ?`,
		deviceID, accountID)
}

func (db *DB) revokeDevice(ctx context.Context, deviceID, query string, args ...any) error {
	now := toMillis(timeNow())
	return db.withWriteTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, query, append([]any{now}, args...)...)
		if err != nil {
			return fmt.Errorf("sqlite: revoke device: %w", err)
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("sqlite: revoke device rows: %w", err)
		}
		if affected == 0 {
			return fmt.Errorf("%w: device %q", store.ErrNotFound, deviceID)
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE auth_token SET revokedAt = ? WHERE deviceId = ? AND revokedAt IS NULL`,
			now, deviceID); err != nil {
			return fmt.Errorf("sqlite: revoke device tokens: %w", err)
		}
		return nil
	})
}

// updateOne runs a single-row update and turns "no such row" into ErrNotFound,
// so a CLI typo reports a missing account instead of silently succeeding.
func (db *DB) updateOne(ctx context.Context, what, id, query string, args ...any) error {
	return db.withWriteTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("sqlite: update %s: %w", what, err)
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("sqlite: update %s rows: %w", what, err)
		}
		if affected == 0 {
			return fmt.Errorf("%w: %s %q", store.ErrNotFound, what, id)
		}
		return nil
	})
}

func (db *DB) ListDevices(ctx context.Context, accountID string) ([]store.Device, error) {
	query := `SELECT ` + deviceColumns + ` FROM device`
	var args []any
	if accountID != "" {
		query += ` WHERE accountId = ?`
		args = append(args, accountID)
	}
	query += ` ORDER BY accountId, createdAt`

	rows, err := db.read.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("sqlite: list devices: %w", err)
	}
	defer func() { _ = rows.Close() }()

	devices := []store.Device{}
	for rows.Next() {
		device, err := scanDevice(rows)
		if err != nil {
			return nil, err
		}
		devices = append(devices, *device)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: iterate devices: %w", err)
	}
	return devices, nil
}

func (db *DB) ListTeams(ctx context.Context) ([]store.Team, error) {
	rows, err := db.read.QueryContext(ctx,
		`SELECT id, name, createdAt FROM team ORDER BY createdAt, id`)
	if err != nil {
		return nil, fmt.Errorf("sqlite: list teams: %w", err)
	}
	defer func() { _ = rows.Close() }()

	teams := []store.Team{}
	for rows.Next() {
		var (
			team      store.Team
			createdAt int64
		)
		if err := rows.Scan(&team.ID, &team.Name, &createdAt); err != nil {
			return nil, fmt.Errorf("sqlite: scan team: %w", err)
		}
		team.CreatedAt = fromMillis(createdAt)
		teams = append(teams, team)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: iterate teams: %w", err)
	}
	return teams, nil
}

func (db *DB) Info(ctx context.Context) (store.Info, error) {
	info := store.Info{Path: db.path}

	if stat, err := os.Stat(db.path); err == nil {
		info.SizeBytes = stat.Size()
	}

	if err := db.read.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&info.SchemaVersion); err != nil {
		return info, fmt.Errorf("sqlite: read schema version: %w", err)
	}

	counts := []struct {
		query string
		into  *int64
	}{
		{`SELECT COUNT(*) FROM account`, &info.Accounts},
		{`SELECT COUNT(*) FROM device`, &info.Devices},
		{`SELECT COUNT(*) FROM team`, &info.Teams},
		{`SELECT COUNT(*) FROM sync_row`, &info.Rows},
		{`SELECT COUNT(*) FROM commit_log`, &info.Commits},
		{`SELECT COALESCE(MIN(serverSeq), 0) FROM commit_log`, &info.OldestSeq},
		{`SELECT COALESCE(MAX(serverSeq), 0) FROM commit_log`, &info.NewestSeq},
		{`SELECT COALESCE((SELECT value FROM sync_meta WHERE key = 'purgedThroughSeq'), 0)`, &info.PurgedThroughSeq},
	}
	for _, c := range counts {
		if err := db.read.QueryRowContext(ctx, c.query).Scan(c.into); err != nil {
			return info, fmt.Errorf("sqlite: %s: %w", c.query, err)
		}
	}
	var err error
	if info.OTPSecretsPlain, info.OTPSecretsEncrypted, err = db.secretCounts(ctx); err != nil {
		return info, err
	}
	// The box this handle uses: the server's after PrepareSecrets, the CLI's
	// own (Plain) otherwise. The CLI overrides it from the configuration.
	info.SecretKey = "missing"
	if db.secrets.KeyID() != "" {
		info.SecretKey = "set"
	}
	return info, nil
}

func (db *DB) DumpCommitLog(ctx context.Context, sinceSeq int64, limit int) ([]store.Change, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := db.read.QueryContext(ctx, `
		SELECT serverSeq, tbl, rowKey, op, ownerId, teamId, version, updatedAtMs, lastWriterId, deletedAt, payload
		FROM commit_log
		WHERE serverSeq > ?
		ORDER BY serverSeq
		LIMIT ?`, sinceSeq, limit)
	if err != nil {
		return nil, fmt.Errorf("sqlite: dump commit log: %w", err)
	}
	defer func() { _ = rows.Close() }()

	changes := []store.Change{}
	for rows.Next() {
		change, err := scanChange(rows)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: iterate commit log: %w", err)
	}
	return changes, nil
}
