package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/team-croffle/croffle/apps/server/internal/store"
)

func (db *DB) CreateTokenPair(ctx context.Context, pair store.TokenPair) error {
	return db.withWriteTx(ctx, func(tx *sql.Tx) error {
		return insertPair(ctx, tx, pair)
	})
}

func (db *DB) FindAuthToken(ctx context.Context, tokenHash []byte) (*store.AuthToken, error) {
	if len(tokenHash) == 0 {
		return nil, store.ErrNotFound
	}
	var (
		token                store.AuthToken
		kind                 string
		createdAt, expiresAt int64
		usedAt, revokedAt    *int64
	)
	err := db.read.QueryRowContext(ctx, `
		SELECT tokenHash, deviceId, kind, familyId, createdAt, expiresAt, usedAt, revokedAt
		FROM auth_token WHERE tokenHash = ?`, tokenHash).Scan(
		&token.TokenHash, &token.DeviceID, &kind, &token.FamilyID,
		&createdAt, &expiresAt, &usedAt, &revokedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("sqlite: read auth token: %w", err)
	}
	token.Kind = store.TokenKind(kind)
	token.CreatedAt = fromMillis(createdAt)
	token.ExpiresAt = fromMillis(expiresAt)
	token.UsedAt = fromMillisPtr(usedAt)
	token.RevokedAt = fromMillisPtr(revokedAt)
	return &token, nil
}

// RotateRefresh is the one place a refresh token is spent. The UPDATE's WHERE
// clause is the check: only an unused, unrevoked, unexpired refresh token
// matches, so two concurrent exchanges of the same token cannot both succeed.
func (db *DB) RotateRefresh(ctx context.Context, oldHash []byte, at time.Time, next store.TokenPair) error {
	now := toMillis(at)
	return db.withWriteTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `
			UPDATE auth_token SET usedAt = ?
			WHERE tokenHash = ? AND kind = 'refresh'
			  AND usedAt IS NULL AND revokedAt IS NULL AND expiresAt > ?`,
			now, oldHash, now)
		if err != nil {
			return fmt.Errorf("sqlite: spend refresh token: %w", err)
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("sqlite: spend refresh token rows: %w", err)
		}
		if affected == 1 {
			return insertPair(ctx, tx, next)
		}

		// Not spendable: tell a replay apart from an unknown token.
		var usedAt *int64
		err = tx.QueryRowContext(ctx,
			`SELECT usedAt FROM auth_token WHERE tokenHash = ? AND kind = 'refresh'`, oldHash).Scan(&usedAt)
		if errors.Is(err, sql.ErrNoRows) {
			return store.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("sqlite: read refresh token: %w", err)
		}
		if usedAt != nil {
			return store.ErrTokenUsed
		}
		return store.ErrNotFound
	})
}

func (db *DB) RevokeTokenFamily(ctx context.Context, familyID string, at time.Time) error {
	return db.withWriteTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx,
			`UPDATE auth_token SET revokedAt = ? WHERE familyId = ? AND revokedAt IS NULL`,
			toMillis(at), familyID); err != nil {
			return fmt.Errorf("sqlite: revoke token family: %w", err)
		}
		return nil
	})
}

func (db *DB) RevokeDeviceTokens(ctx context.Context, deviceID string, at time.Time) error {
	return db.withWriteTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx,
			`UPDATE auth_token SET revokedAt = ? WHERE deviceId = ? AND revokedAt IS NULL`,
			toMillis(at), deviceID); err != nil {
			return fmt.Errorf("sqlite: revoke device tokens: %w", err)
		}
		return nil
	})
}

// insertPair refuses a device that is missing or revoked. The check runs in
// the writer's transaction, so it is ordered against every revocation: a pair
// is either stored before a revoke (and revoked by it) or not stored at all.
func insertPair(ctx context.Context, tx *sql.Tx, pair store.TokenPair) error {
	var live int
	err := tx.QueryRowContext(ctx,
		`SELECT 1 FROM device WHERE id = ? AND revokedAt IS NULL`, pair.Access.DeviceID).Scan(&live)
	if errors.Is(err, sql.ErrNoRows) {
		return store.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("sqlite: check device: %w", err)
	}
	if pair.Refresh.DeviceID != pair.Access.DeviceID {
		return errors.New("sqlite: a pair belongs to one device")
	}
	for _, token := range []store.AuthToken{pair.Access, pair.Refresh} {
		if len(token.TokenHash) == 0 || token.DeviceID == "" || token.FamilyID == "" {
			return errors.New("sqlite: token hash, device and family are required")
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO auth_token (tokenHash, deviceId, kind, familyId, createdAt, expiresAt)
			VALUES (?, ?, ?, ?, ?, ?)`,
			token.TokenHash, token.DeviceID, string(token.Kind), token.FamilyID,
			toMillis(token.CreatedAt), toMillis(token.ExpiresAt)); err != nil {
			return fmt.Errorf("sqlite: insert %s token: %w", token.Kind, err)
		}
	}
	return nil
}
