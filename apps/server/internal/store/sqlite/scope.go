package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
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

// CoverTeams records that the device has received these teams' rows from the
// beginning. Teams the account has left since (no membership row) are not
// recorded, so a snapshot that started before a removal cannot mark a team the
// device may no longer read.
func (db *DB) CoverTeams(ctx context.Context, deviceID string, teamIDs []string) error {
	if deviceID == "" || len(teamIDs) == 0 {
		return nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?, ", len(teamIDs)), ", ")
	args := []any{toMillis(timeNow()), deviceID}
	for _, team := range teamIDs {
		args = append(args, team)
	}
	return db.withWriteTx(ctx, func(tx *sql.Tx) error {
		// Built from the length of the slice, never from its contents.
		_, err := tx.ExecContext(ctx, `
			INSERT INTO device_scope (deviceId, teamId, coveredAt)
			SELECT d.id, m.teamId, ?
			FROM device d JOIN team_member m ON m.accountId = d.accountId
			WHERE d.id = ? AND m.teamId IN (`+placeholders+`)
			ON CONFLICT (deviceId, teamId) DO NOTHING`, args...)
		if err != nil {
			return fmt.Errorf("sqlite: cover teams: %w", err)
		}
		return nil
	})
}
