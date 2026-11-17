package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/team-croffle/croffle/apps/server/internal/store"
)

// teamExists returns ErrNotFound for a team id that is not there, instead of
// the foreign-key error the insert would raise.
func teamExists(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, teamID string) error {
	var one int
	err := q.QueryRowContext(ctx, `SELECT 1 FROM team WHERE id = ?`, teamID).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: team %s", store.ErrNotFound, teamID)
	}
	if err != nil {
		return fmt.Errorf("sqlite: read team: %w", err)
	}
	return nil
}

func (db *DB) CreateTeam(ctx context.Context, team store.Team) error {
	if team.ID == "" || team.Name == "" {
		return errors.New("sqlite: team id and name are required")
	}
	return db.withWriteTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `
			INSERT INTO team (id, name, createdAt) VALUES (?, ?, ?)
			ON CONFLICT (id) DO NOTHING`, team.ID, team.Name, toMillis(team.CreatedAt))
		if err != nil {
			return fmt.Errorf("sqlite: create team: %w", err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return fmt.Errorf("%w: team %s", store.ErrDuplicate, team.ID)
		}
		return nil
	})
}

func (db *DB) RenameTeam(ctx context.Context, teamID, name string) error {
	if name == "" {
		return errors.New("sqlite: empty team name")
	}
	return db.updateOne(ctx, "team", teamID, `UPDATE team SET name = ? WHERE id = ?`, name, teamID)
}

func (db *DB) RemoveTeamMember(ctx context.Context, teamID, accountID string) error {
	return db.withWriteTx(ctx, func(tx *sql.Tx) error {
		if err := execOne(ctx, tx, "team member", teamID+"/"+accountID,
			`DELETE FROM team_member WHERE teamId = ? AND accountId = ?`, teamID, accountID); err != nil {
			return err
		}
		// Joining again has to catch up again: whatever the team wrote while
		// the account was out never reached its devices.
		if _, err := tx.ExecContext(ctx, `
			DELETE FROM device_scope
			WHERE teamId = ? AND deviceId IN (SELECT id FROM device WHERE accountId = ?)`, teamID, accountID); err != nil {
			return fmt.Errorf("sqlite: drop device scope: %w", err)
		}
		return nil
	})
}

func (db *DB) ListTeamMembers(ctx context.Context, teamID string) ([]store.TeamMember, error) {
	if err := teamExists(ctx, db.read, teamID); err != nil {
		return nil, err
	}
	rows, err := db.read.QueryContext(ctx, `
		SELECT m.accountId, a.username, m.role, m.createdAt
		FROM team_member m JOIN account a ON a.id = m.accountId
		WHERE m.teamId = ? ORDER BY a.username`, teamID)
	if err != nil {
		return nil, fmt.Errorf("sqlite: list team members: %w", err)
	}
	defer func() { _ = rows.Close() }()
	members := []store.TeamMember{}
	for rows.Next() {
		var (
			m         store.TeamMember
			createdAt int64
		)
		if err := rows.Scan(&m.AccountID, &m.Username, &m.Role, &createdAt); err != nil {
			return nil, fmt.Errorf("sqlite: scan team member: %w", err)
		}
		m.CreatedAt = fromMillis(createdAt)
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: iterate team members: %w", err)
	}
	return members, nil
}
