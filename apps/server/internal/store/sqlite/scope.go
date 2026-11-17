package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/team-croffle/croffle/apps/server/internal/store"
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
// beginning. Only a membership that still exists unchanged is recorded: a team
// the account has left since is not, and neither is one it left and joined
// again (its join time differs), because the snapshot could have skipped that
// team's rows while the account was out.
func (db *DB) CoverTeams(ctx context.Context, deviceID string, memberships []store.Membership) error {
	if deviceID == "" || len(memberships) == 0 {
		return nil
	}
	now := toMillis(timeNow())
	return db.withWriteTx(ctx, func(tx *sql.Tx) error {
		for _, m := range memberships {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO device_scope (deviceId, teamId, coveredAt)
				SELECT d.id, tm.teamId, ?
				FROM device d JOIN team_member tm ON tm.accountId = d.accountId
				WHERE d.id = ? AND tm.teamId = ? AND tm.createdAt = ?
				ON CONFLICT (deviceId, teamId) DO NOTHING`, now, deviceID, m.TeamID, m.JoinedAtMs); err != nil {
				return fmt.Errorf("sqlite: cover teams: %w", err)
			}
		}
		return nil
	})
}

func (db *DB) Memberships(ctx context.Context, accountID string) ([]store.Membership, error) {
	rows, err := db.read.QueryContext(ctx,
		`SELECT teamId, createdAt FROM team_member WHERE accountId = ? ORDER BY teamId`, accountID)
	if err != nil {
		return nil, fmt.Errorf("sqlite: read memberships: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := []store.Membership{}
	for rows.Next() {
		var m store.Membership
		if err := rows.Scan(&m.TeamID, &m.JoinedAtMs); err != nil {
			return nil, fmt.Errorf("sqlite: scan membership: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
