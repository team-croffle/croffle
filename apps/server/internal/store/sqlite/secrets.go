package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/team-croffle/croffle/apps/server/internal/secretbox"
)

// SecretsReport is what PrepareSecrets found and did.
type SecretsReport struct {
	// Sealed is how many plain secrets were sealed with the key just now.
	Sealed int
	// Plain is how many secrets are stored plain afterwards (only without a
	// key).
	Plain int
}

// ErrSecretsNeedKey and ErrSecretsOtherKey stop the server from starting: it
// could not check any one-time code, so every account using one would be
// locked out with a confusing error instead of a clear start-up failure.
var (
	ErrSecretsNeedKey  = errors.New("stored one-time code secrets are encrypted but no secret key is set")
	ErrSecretsOtherKey = errors.New("stored one-time code secrets were sealed with another key")
)

type storedSecret struct {
	accountID string
	value     string
}

func (db *DB) storedSecrets(ctx context.Context, q interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}) ([]storedSecret, error) {
	rows, err := q.QueryContext(ctx, `SELECT accountId, secret FROM account_otp ORDER BY accountId`)
	if err != nil {
		return nil, fmt.Errorf("sqlite: read otp secrets: %w", err)
	}
	defer rows.Close()
	var out []storedSecret
	for rows.Next() {
		var s storedSecret
		if err := rows.Scan(&s.accountID, &s.value); err != nil {
			return nil, fmt.Errorf("sqlite: scan otp secret: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// PrepareSecrets checks the stored TOTP secrets against box and, when box has
// a key, seals the plain ones in one transaction. It runs before the server
// listens:
//
//   - with a key, a secret sealed with another key is ErrSecretsOtherKey;
//   - without a key (secretbox.Plain), any sealed secret is ErrSecretsNeedKey.
//
// It also installs box on db, so the server serves with the box it checked.
func (db *DB) PrepareSecrets(ctx context.Context, box secretbox.Box) (SecretsReport, error) {
	if box == nil {
		box = secretbox.Plain
	}
	var report SecretsReport
	err := db.withWriteTx(ctx, func(tx *sql.Tx) error {
		secrets, err := db.storedSecrets(ctx, tx)
		if err != nil {
			return err
		}
		keyID := box.KeyID()
		for _, s := range secrets {
			if !secretbox.IsSealed(s.value) {
				if keyID == "" {
					report.Plain++
					continue
				}
				sealed, err := box.Seal(s.accountID, s.value)
				if err != nil {
					return err
				}
				if _, err := tx.ExecContext(ctx,
					`UPDATE account_otp SET secret = ? WHERE accountId = ?`, sealed, s.accountID); err != nil {
					return fmt.Errorf("sqlite: seal otp secret: %w", err)
				}
				report.Sealed++
				continue
			}
			if keyID == "" {
				return ErrSecretsNeedKey
			}
			if id, _ := secretbox.KeyIDOf(s.value); id != keyID {
				return fmt.Errorf("%w (id %s); run secret rotate", ErrSecretsOtherKey, id)
			}
		}
		return nil
	})
	if err != nil {
		return SecretsReport{}, err
	}
	db.SetSecretBox(box)
	return report, nil
}

// RotateSecrets opens every stored secret with from and seals it with to, in
// one transaction: if a single secret does not open, nothing changes. from may
// be secretbox.Plain, which turns sealing on for a server that had no key.
// It reports how many secrets were sealed with to.
func (db *DB) RotateSecrets(ctx context.Context, from, to secretbox.Box) (int, error) {
	if from == nil {
		from = secretbox.Plain
	}
	if to == nil || to.KeyID() == "" {
		return 0, errors.New("sqlite: rotate needs a new key")
	}
	count := 0
	err := db.withWriteTx(ctx, func(tx *sql.Tx) error {
		secrets, err := db.storedSecrets(ctx, tx)
		if err != nil {
			return err
		}
		for _, s := range secrets {
			plain, err := from.Open(s.accountID, s.value)
			if err != nil {
				return fmt.Errorf("the secret of account %s does not open with the current key: %w", s.accountID, err)
			}
			sealed, err := to.Seal(s.accountID, plain)
			if err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx,
				`UPDATE account_otp SET secret = ? WHERE accountId = ?`, sealed, s.accountID); err != nil {
				return fmt.Errorf("sqlite: reseal otp secret: %w", err)
			}
			count++
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}

// secretCounts fills the secret fields of Info.
func (db *DB) secretCounts(ctx context.Context) (plain, sealed int64, err error) {
	secrets, err := db.storedSecrets(ctx, db.read)
	if err != nil {
		return 0, 0, err
	}
	for _, s := range secrets {
		if secretbox.IsSealed(s.value) {
			sealed++
		} else {
			plain++
		}
	}
	return plain, sealed, nil
}
