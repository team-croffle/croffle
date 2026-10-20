package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/team-croffle/croffle/apps/server/internal/store"
)

// ErrDeviceOwnedByAnotherAccount is returned when a login presents a device id
// that already belongs to someone else. Device ids come from the client, so
// without this check a caller with valid credentials of their own could claim
// another account's device id and invalidate that device's token.
var ErrDeviceOwnedByAnotherAccount = errors.New("sqlite: device id belongs to another account")

const accountColumns = `id, username, passwordHash, createdAt, disabledAt`

func (db *DB) FindAccountByUsername(ctx context.Context, username string) (*store.Account, error) {
	return db.scanAccount(ctx,
		`SELECT `+accountColumns+` FROM account WHERE username = ?`, username)
}

func (db *DB) FindAccountByID(ctx context.Context, id string) (*store.Account, error) {
	return db.scanAccount(ctx,
		`SELECT `+accountColumns+` FROM account WHERE id = ?`, id)
}

func (db *DB) scanAccount(ctx context.Context, query string, args ...any) (*store.Account, error) {
	var (
		account    store.Account
		createdAt  int64
		disabledAt *int64
	)

	err := db.read.QueryRowContext(ctx, query, args...).Scan(
		&account.ID, &account.Username, &account.PasswordHash, &createdAt, &disabledAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("sqlite: read account: %w", err)
	}

	account.CreatedAt = fromMillis(createdAt)
	account.DisabledAt = fromMillisPtr(disabledAt)
	return &account, nil
}

func (db *DB) UpsertDevice(ctx context.Context, device store.Device) error {
	if device.ID == "" || device.AccountID == "" {
		return errors.New("sqlite: device id and account id are required")
	}

	return db.withWriteTx(ctx, func(tx *sql.Tx) error {
		// The WHERE on the conflict clause is the ownership check; it makes the
		// update a no-op for a foreign device, which RowsAffected then reports.
		res, err := tx.ExecContext(ctx, `
			INSERT INTO device (id, accountId, name, tokenHash, createdAt, lastSeenAt, revokedAt)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (id) DO UPDATE SET
				name       = excluded.name,
				tokenHash  = excluded.tokenHash,
				lastSeenAt = excluded.lastSeenAt,
				-- A fresh login re-arms a revoked device: revoking withdraws a
				-- token, it does not bar an account that can still prove itself.
				-- Disabling the account is what stops it for good.
				revokedAt  = NULL
			WHERE device.accountId = excluded.accountId`,
			device.ID, device.AccountID, device.Name, device.TokenHash,
			toMillis(device.CreatedAt), toMillisPtr(device.LastSeenAt), toMillisPtr(device.RevokedAt))
		if err != nil {
			return fmt.Errorf("sqlite: upsert device: %w", err)
		}

		affected, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("sqlite: upsert device rows: %w", err)
		}
		if affected == 0 {
			return ErrDeviceOwnedByAnotherAccount
		}
		return nil
	})
}

func (db *DB) FindDeviceByTokenHash(ctx context.Context, tokenHash []byte) (*store.Device, error) {
	if len(tokenHash) == 0 {
		return nil, store.ErrNotFound
	}

	var (
		device     store.Device
		createdAt  int64
		lastSeenAt *int64
		revokedAt  *int64
	)

	err := db.read.QueryRowContext(ctx, `
		SELECT id, accountId, name, tokenHash, createdAt, lastSeenAt, revokedAt
		FROM device WHERE tokenHash = ?`, tokenHash).Scan(
		&device.ID, &device.AccountID, &device.Name, &device.TokenHash,
		&createdAt, &lastSeenAt, &revokedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("sqlite: read device: %w", err)
	}

	device.CreatedAt = fromMillis(createdAt)
	device.LastSeenAt = fromMillisPtr(lastSeenAt)
	device.RevokedAt = fromMillisPtr(revokedAt)
	return &device, nil
}

func (db *DB) TouchDevice(ctx context.Context, deviceID string, at time.Time) error {
	return db.withWriteTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx,
			`UPDATE device SET lastSeenAt = ? WHERE id = ?`, toMillis(at), deviceID)
		if err != nil {
			return fmt.Errorf("sqlite: touch device: %w", err)
		}
		return nil
	})
}

func (db *DB) TeamsOf(ctx context.Context, accountID string) ([]string, error) {
	rows, err := db.read.QueryContext(ctx,
		`SELECT teamId FROM team_member WHERE accountId = ? ORDER BY teamId`, accountID)
	if err != nil {
		return nil, fmt.Errorf("sqlite: read team membership: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// Never nil: the caller builds an IN (...) clause from this and an
	// accidental nil is easier to mishandle than an empty slice.
	teams := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("sqlite: scan team id: %w", err)
		}
		teams = append(teams, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: iterate team membership: %w", err)
	}
	return teams, nil
}

// CreateAccount inserts a new login. The caller supplies an already hashed
// password: this layer never sees a plaintext one.
func (db *DB) CreateAccount(ctx context.Context, account store.Account) error {
	if account.ID == "" || account.Username == "" || account.PasswordHash == "" {
		return errors.New("sqlite: account id, username and password hash are required")
	}

	return db.withWriteTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO account (id, username, passwordHash, createdAt, disabledAt)
			VALUES (?, ?, ?, ?, ?)`,
			account.ID, account.Username, account.PasswordHash,
			toMillis(account.CreatedAt), toMillisPtr(account.DisabledAt))
		if err != nil {
			return fmt.Errorf("sqlite: create account: %w", err)
		}
		return nil
	})
}

// AddTeamMember makes an account part of a team, which is what makes that
// team's shared rows visible to it. Re-adding an existing member is a no-op.
func (db *DB) AddTeamMember(ctx context.Context, teamID, accountID, role string) error {
	if role == "" {
		role = "member"
	}
	return db.withWriteTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO team_member (teamId, accountId, role, createdAt)
			VALUES (?, ?, ?, ?)
			ON CONFLICT (teamId, accountId) DO NOTHING`,
			teamID, accountID, role, toMillis(timeNow()))
		if err != nil {
			return fmt.Errorf("sqlite: add team member: %w", err)
		}
		return nil
	})
}
