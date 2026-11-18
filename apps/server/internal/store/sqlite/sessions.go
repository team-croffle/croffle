package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/team-croffle/croffle/apps/server/internal/store"
)

// Compile-time check for the dashboard surface.
var _ store.Sessions = (*DB)(nil)

func (db *DB) CreateSession(ctx context.Context, session store.Session) error {
	if len(session.TokenHash) == 0 || session.AccountID == "" {
		return errors.New("sqlite: session needs a hash and an account")
	}

	return db.withWriteTx(ctx, func(tx *sql.Tx) error {
		if session.PasswordHash == "" {
			_, err := tx.ExecContext(ctx, `
				INSERT INTO dash_session (tokenHash, accountId, createdAt, expiresAt)
				VALUES (?, ?, ?, ?)`,
				session.TokenHash, session.AccountID,
				toMillis(session.CreatedAt), toMillis(session.ExpiresAt))
			if err != nil {
				return fmt.Errorf("sqlite: create session: %w", err)
			}
			return nil
		}
		res, err := tx.ExecContext(ctx, `
			INSERT INTO dash_session (tokenHash, accountId, createdAt, expiresAt)
			SELECT ?, id, ?, ? FROM account
			WHERE id = ? AND passwordHash = ? AND disabledAt IS NULL`,
			session.TokenHash, toMillis(session.CreatedAt), toMillis(session.ExpiresAt),
			session.AccountID, session.PasswordHash)
		if err != nil {
			return fmt.Errorf("sqlite: create session: %w", err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return fmt.Errorf("%w: the password changed or the account was disabled during sign-in", store.ErrNotFound)
		}
		return nil
	})
}

// FindSession returns ErrNotFound for a session that does not exist or has
// expired. An expired row is left for the sweep to collect; checking the time
// here means a stale row can never authenticate even if the sweep is behind.
func (db *DB) FindSession(ctx context.Context, tokenHash []byte, now time.Time) (*store.Session, error) {
	if len(tokenHash) == 0 {
		return nil, store.ErrNotFound
	}

	var (
		session   store.Session
		createdAt int64
		expiresAt int64
	)
	err := db.read.QueryRowContext(ctx, `
		SELECT tokenHash, accountId, createdAt, expiresAt
		FROM dash_session WHERE tokenHash = ? AND expiresAt > ?`,
		tokenHash, toMillis(now)).Scan(&session.TokenHash, &session.AccountID, &createdAt, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("sqlite: read session: %w", err)
	}

	session.CreatedAt = fromMillis(createdAt)
	session.ExpiresAt = fromMillis(expiresAt)
	return &session, nil
}

func (db *DB) DeleteSession(ctx context.Context, tokenHash []byte) error {
	return db.withWriteTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM dash_session WHERE tokenHash = ?`, tokenHash); err != nil {
			return fmt.Errorf("sqlite: delete session: %w", err)
		}
		return nil
	})
}

// ChangePassword sets the account's own new password and signs out every other
// dashboard session, in one transaction; see store.Store.
func (db *DB) ChangePassword(ctx context.Context, accountID, oldHash, newHash string, keepSession []byte) error {
	if newHash == "" || oldHash == "" {
		return errors.New("sqlite: empty password hash")
	}
	return db.withWriteTx(ctx, func(tx *sql.Tx) error {
		if err := execOne(ctx, tx, "account", accountID,
			`UPDATE account SET passwordHash = ? WHERE id = ? AND passwordHash = ?`,
			newHash, accountID, oldHash); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM dash_session WHERE accountId = ? AND tokenHash <> ?`,
			accountID, keepSession); err != nil {
			return fmt.Errorf("sqlite: delete other sessions: %w", err)
		}
		return nil
	})
}

// ResetPassword sets a new password and signs out every dashboard session of
// the account, in one transaction. Device tokens are separate credentials and
// stay; revoking them is a separate step.
func (db *DB) ResetPassword(ctx context.Context, accountID, passwordHash string) error {
	if passwordHash == "" {
		return errors.New("sqlite: empty password hash")
	}
	return db.withWriteTx(ctx, func(tx *sql.Tx) error {
		if err := execOne(ctx, tx, "account", accountID,
			`UPDATE account SET passwordHash = ? WHERE id = ?`, passwordHash, accountID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM dash_session WHERE accountId = ?`, accountID); err != nil {
			return fmt.Errorf("sqlite: delete account sessions: %w", err)
		}
		return nil
	})
}
