package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// UncoveredTeams returns the teams of teamIDs the device has not received
// from the beginning (see migration 0007), in the order given.
func (db *DB) UncoveredTeams(ctx context.Context, deviceID string, teamIDs []string) ([]string, error) {
	if deviceID == "" || len(teamIDs) == 0 {
		return nil, nil
	}
	rows, err := db.read.QueryContext(ctx,
		`SELECT teamId FROM device_scope WHERE deviceId = ?`, deviceID)
	if err != nil {
		return nil, fmt.Errorf("sqlite: read device scope: %w", err)
	}
	defer func() { _ = rows.Close() }()
	covered := map[string]bool{}
	for rows.Next() {
		var team string
		if err := rows.Scan(&team); err != nil {
			return nil, fmt.Errorf("sqlite: scan device scope: %w", err)
		}
		covered[team] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: iterate device scope: %w", err)
	}
	var missing []string
	for _, team := range teamIDs {
		if !covered[team] {
			missing = append(missing, team)
		}
	}
	return missing, nil
}

// membershipGenerationKey is the sync_meta key of an account's membership
// generation: a counter that goes up in the same transaction as every change
// of the account's teams. A snapshot records it at its start; if it differs at
// the end, a membership changed while the pages were read, and the snapshot
// covers nothing (the next pull asks for another one). Unlike a timestamp it
// cannot collide when a membership is removed and added again.
func membershipGenerationKey(accountID string) string { return "membershipGen:" + accountID }

func bumpMembershipGeneration(ctx context.Context, tx *sql.Tx, accountID string) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO sync_meta (key, value) VALUES (?, 1)
		ON CONFLICT (key) DO UPDATE SET value = value + 1`, membershipGenerationKey(accountID)); err != nil {
		return fmt.Errorf("sqlite: bump membership generation: %w", err)
	}
	return nil
}

func membershipGeneration(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, accountID string) (int64, error) {
	var generation int64
	err := q.QueryRowContext(ctx, `SELECT value FROM sync_meta WHERE key = ?`,
		membershipGenerationKey(accountID)).Scan(&generation)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("sqlite: read membership generation: %w", err)
	}
	return generation, nil
}

// MembershipState returns the account's teams and its membership generation,
// read together, so a snapshot's visibility and what it may later cover come
// from the same instant (review 2 G-2).
func (db *DB) MembershipState(ctx context.Context, accountID string) ([]string, int64, error) {
	tx, err := db.read.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, 0, fmt.Errorf("sqlite: begin read: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	generation, err := membershipGeneration(ctx, tx, accountID)
	if err != nil {
		return nil, 0, err
	}
	rows, err := tx.QueryContext(ctx,
		`SELECT teamId FROM team_member WHERE accountId = ? ORDER BY teamId`, accountID)
	if err != nil {
		return nil, 0, fmt.Errorf("sqlite: read memberships: %w", err)
	}
	defer func() { _ = rows.Close() }()
	teams := []string{}
	for rows.Next() {
		var team string
		if err := rows.Scan(&team); err != nil {
			return nil, 0, fmt.Errorf("sqlite: scan membership: %w", err)
		}
		teams = append(teams, team)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("sqlite: iterate memberships: %w", err)
	}
	return teams, generation, nil
}

// CoverTeams records that the device has received these teams' rows from the
// beginning, if the account's membership generation is still the one the
// snapshot started with; otherwise a membership changed during the snapshot
// and it covers nothing. Only teams the account is in are recorded.
func (db *DB) CoverTeams(ctx context.Context, deviceID string, teamIDs []string, generation int64) error {
	if deviceID == "" || len(teamIDs) == 0 {
		return nil
	}
	now := toMillis(timeNow())
	return db.withWriteTx(ctx, func(tx *sql.Tx) error {
		var accountID string
		err := tx.QueryRowContext(ctx, `SELECT accountId FROM device WHERE id = ?`, deviceID).Scan(&accountID)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("sqlite: read device: %w", err)
		}
		current, err := membershipGeneration(ctx, tx, accountID)
		if err != nil {
			return err
		}
		if current != generation {
			return nil
		}
		for _, team := range teamIDs {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO device_scope (deviceId, teamId, coveredAt)
				SELECT ?, teamId, ? FROM team_member WHERE accountId = ? AND teamId = ?
				ON CONFLICT (deviceId, teamId) DO NOTHING`, deviceID, now, accountID, team); err != nil {
				return fmt.Errorf("sqlite: cover teams: %w", err)
			}
		}
		return nil
	})
}
