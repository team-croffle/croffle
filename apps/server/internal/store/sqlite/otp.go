package sqlite

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/team-croffle/croffle/apps/server/internal/store"
)

func (db *DB) FindOTP(ctx context.Context, accountID string) (*store.OTP, error) {
	var (
		otp       store.OTP
		enabledAt *int64
		createdAt int64
	)
	err := db.read.QueryRowContext(ctx, `
		SELECT accountId, secret, enabledAt, lastStep, createdAt
		FROM account_otp WHERE accountId = ?`, accountID).Scan(
		&otp.AccountID, &otp.Secret, &enabledAt, &otp.LastStep, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("sqlite: read otp: %w", err)
	}
	otp.EnabledAt = fromMillisPtr(enabledAt)
	otp.CreatedAt = fromMillis(createdAt)
	if otp.Secret, err = db.secrets.Open(accountID, otp.Secret); err != nil {
		return nil, fmt.Errorf("sqlite: open otp secret: %w", err)
	}
	return &otp, nil
}

func (db *DB) OTPStatus(ctx context.Context, accountID string) (string, error) {
	var enabled bool
	err := db.read.QueryRowContext(ctx,
		`SELECT enabledAt IS NOT NULL FROM account_otp WHERE accountId = ?`, accountID).Scan(&enabled)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return "off", nil
	case err != nil:
		return "", fmt.Errorf("sqlite: read otp status: %w", err)
	case enabled:
		return "enabled", nil
	default:
		return "pending", nil
	}
}

func (db *DB) StartOTP(ctx context.Context, accountID, secret string, at time.Time) error {
	stored, err := db.secrets.Seal(accountID, secret)
	if err != nil {
		return fmt.Errorf("sqlite: seal otp secret: %w", err)
	}
	return db.withWriteTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `
			INSERT INTO account_otp (accountId, secret, createdAt) VALUES (?, ?, ?)
			ON CONFLICT (accountId) DO UPDATE SET
				secret = excluded.secret, createdAt = excluded.createdAt, lastStep = 0
			WHERE account_otp.enabledAt IS NULL`,
			accountID, stored, toMillis(at))
		if err != nil {
			return fmt.Errorf("sqlite: start otp: %w", err)
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("sqlite: start otp rows: %w", err)
		}
		if affected == 0 {
			return fmt.Errorf("%w: one-time codes are already enabled", store.ErrDuplicate)
		}
		return nil
	})
}

func (db *DB) EnableOTP(ctx context.Context, accountID, secret string, step int64, recoveryHashes [][]byte, at time.Time) error {
	now := toMillis(at)
	return db.withWriteTx(ctx, func(tx *sql.Tx) error {
		// The stored value is sealed with a fresh nonce, so it cannot be
		// compared with secret in SQL: open it here, in the same transaction,
		// and update only the row that was read.
		var stored string
		err := tx.QueryRowContext(ctx,
			`SELECT secret FROM account_otp WHERE accountId = ? AND enabledAt IS NULL`, accountID).Scan(&stored)
		if errors.Is(err, sql.ErrNoRows) {
			return store.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("sqlite: read pending otp: %w", err)
		}
		plain, err := db.secrets.Open(accountID, stored)
		if err != nil {
			return fmt.Errorf("sqlite: open otp secret: %w", err)
		}
		if subtle.ConstantTimeCompare([]byte(plain), []byte(secret)) != 1 {
			return store.ErrNotFound
		}
		res, err := tx.ExecContext(ctx, `
			UPDATE account_otp SET enabledAt = ?, lastStep = ?
			WHERE accountId = ? AND secret = ? AND enabledAt IS NULL`, now, step, accountID, stored)
		if err != nil {
			return fmt.Errorf("sqlite: enable otp: %w", err)
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("sqlite: enable otp rows: %w", err)
		}
		if affected == 0 {
			return store.ErrNotFound
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM otp_recovery_code WHERE accountId = ?`, accountID); err != nil {
			return fmt.Errorf("sqlite: clear recovery codes: %w", err)
		}
		for _, hash := range recoveryHashes {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO otp_recovery_code (codeHash, accountId, createdAt) VALUES (?, ?, ?)`,
				hash, accountID, now); err != nil {
				return fmt.Errorf("sqlite: store recovery code: %w", err)
			}
		}
		return nil
	})
}

// AdvanceOTPStep is the replay guard. The WHERE clause does the comparison, so
// two concurrent sign-ins with the same code cannot both pass.
func (db *DB) AdvanceOTPStep(ctx context.Context, accountID string, step int64) (bool, error) {
	var advanced bool
	err := db.withWriteTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx,
			`UPDATE account_otp SET lastStep = ? WHERE accountId = ? AND lastStep < ?`,
			step, accountID, step)
		if err != nil {
			return fmt.Errorf("sqlite: advance otp step: %w", err)
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("sqlite: advance otp step rows: %w", err)
		}
		advanced = affected == 1
		return nil
	})
	return advanced, err
}

func (db *DB) UseRecoveryCode(ctx context.Context, accountID string, codeHash []byte, at time.Time) (bool, error) {
	var used bool
	err := db.withWriteTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `
			UPDATE otp_recovery_code SET usedAt = ?
			WHERE codeHash = ? AND accountId = ? AND usedAt IS NULL`,
			toMillis(at), codeHash, accountID)
		if err != nil {
			return fmt.Errorf("sqlite: use recovery code: %w", err)
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("sqlite: use recovery code rows: %w", err)
		}
		used = affected == 1
		return nil
	})
	return used, err
}

func (db *DB) DeleteOTP(ctx context.Context, accountID string) error {
	return db.withWriteTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM otp_recovery_code WHERE accountId = ?`, accountID); err != nil {
			return fmt.Errorf("sqlite: delete recovery codes: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM account_otp WHERE accountId = ?`, accountID); err != nil {
			return fmt.Errorf("sqlite: delete otp: %w", err)
		}
		return nil
	})
}
