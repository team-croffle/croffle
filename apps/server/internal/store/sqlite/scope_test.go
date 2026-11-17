package sqlite

import (
	"context"
	"database/sql"
	"reflect"
	"testing"
	"time"

	"github.com/team-croffle/croffle/apps/server/internal/store"
)

func scopeDevice(t *testing.T, db *DB, id, accountID string) {
	t.Helper()
	if err := db.UpsertDevice(context.Background(), store.Device{
		ID: id, AccountID: accountID, Name: "test", TokenHash: []byte("hash-" + id), CreatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
}

func uncovered(t *testing.T, db *DB, deviceID string, teams ...string) []string {
	t.Helper()
	missing, err := db.UncoveredTeams(context.Background(), deviceID, teams)
	if err != nil {
		t.Fatal(err)
	}
	return missing
}

// A device starts covered for the teams its account is in when it is created;
// a team joined afterwards is uncovered until CoverTeams records it.
func TestDeviceScopeNewDeviceAndLaterJoin(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	newAccount(t, db, "acct-1", "alice")
	if err := db.AddTeamMember(ctx, store.DefaultTeamID, "acct-1", ""); err != nil {
		t.Fatal(err)
	}
	scopeDevice(t, db, "dev-1", "acct-1")
	if got := uncovered(t, db, "dev-1", store.DefaultTeamID); got != nil {
		t.Fatalf("a new device is uncovered for %v", got)
	}

	if err := db.CreateTeam(ctx, store.Team{ID: "team-2", Name: "Two", CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if err := db.AddTeamMember(ctx, "team-2", "acct-1", ""); err != nil {
		t.Fatal(err)
	}
	if got := uncovered(t, db, "dev-1", store.DefaultTeamID, "team-2"); !reflect.DeepEqual(got, []string{"team-2"}) {
		t.Fatalf("after joining team-2: uncovered %v", got)
	}
	// Signing in again on the same device does not cover the new team.
	scopeDevice(t, db, "dev-1", "acct-1")
	if got := uncovered(t, db, "dev-1", "team-2"); len(got) != 1 {
		t.Fatalf("a repeated sign-in covered team-2: %v", got)
	}
	if err := db.CoverTeams(ctx, "dev-1", memberships(t, db, "acct-1")); err != nil {
		t.Fatal(err)
	}
	if got := uncovered(t, db, "dev-1", store.DefaultTeamID, "team-2"); got != nil {
		t.Fatalf("after CoverTeams: uncovered %v", got)
	}
}

func TestDeviceScopeCoverTeamsSkipsTeamsTheAccountLeft(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	newAccount(t, db, "acct-1", "alice")
	scopeDevice(t, db, "dev-1", "acct-1")
	if err := db.CreateTeam(ctx, store.Team{ID: "team-2", Name: "Two", CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	// Not a member of team-2: a snapshot that started while it was must not
	// mark it covered.
	if err := db.CoverTeams(ctx, "dev-1", []store.Membership{{TeamID: "team-2", JoinedAtMs: 1}}); err != nil {
		t.Fatal(err)
	}
	if err := db.AddTeamMember(ctx, "team-2", "acct-1", ""); err != nil {
		t.Fatal(err)
	}
	if got := uncovered(t, db, "dev-1", "team-2"); len(got) != 1 {
		t.Fatalf("CoverTeams recorded a team the account was not in: %v", got)
	}
}

func TestRemoveTeamMemberDropsScope(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	newAccount(t, db, "acct-1", "alice")
	if err := db.AddTeamMember(ctx, store.DefaultTeamID, "acct-1", ""); err != nil {
		t.Fatal(err)
	}
	scopeDevice(t, db, "dev-1", "acct-1")
	scopeDevice(t, db, "dev-2", "acct-1")
	if err := db.RemoveTeamMember(ctx, store.DefaultTeamID, "acct-1"); err != nil {
		t.Fatal(err)
	}
	if err := db.AddTeamMember(ctx, store.DefaultTeamID, "acct-1", ""); err != nil {
		t.Fatal(err)
	}
	for _, dev := range []string{"dev-1", "dev-2"} {
		if got := uncovered(t, db, dev, store.DefaultTeamID); len(got) != 1 {
			t.Fatalf("%s is still covered after leaving and joining again", dev)
		}
	}
}

// 0.1.1 -> 0.2.0: every device is covered for its account's teams, so no
// device takes a snapshot because of the upgrade.
func TestMigrate0007SeedsExistingDevices(t *testing.T) {
	path := tempPath(t)
	ctx := context.Background()
	raw, err := sql.Open(driverName, dsn(path, true))
	if err != nil {
		t.Fatal(err)
	}
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	legacy := &DB{write: raw}
	if err := legacy.ensureMigrationsTable(ctx); err != nil {
		t.Fatal(err)
	}
	for _, m := range migrations {
		if m.version > 5 { // 0.1.1 shipped migrations 0001-0005
			break
		}
		if err := legacy.applyMigration(ctx, m); err != nil {
			t.Fatal(err)
		}
	}
	for _, q := range []string{
		`INSERT INTO account (id, username, passwordHash, createdAt) VALUES ('acct-1', 'alice', 'h', 1)`,
		`INSERT INTO account (id, username, passwordHash, createdAt) VALUES ('acct-2', 'bob', 'h', 1)`,
		`INSERT INTO team_member (teamId, accountId, createdAt) VALUES ('team-default', 'acct-1', 1)`,
		`INSERT INTO device (id, accountId, name, createdAt) VALUES ('client-1', 'acct-1', 'laptop', 1)`,
		`INSERT INTO device (id, accountId, name, createdAt) VALUES ('client-2', 'acct-2', 'phone', 1)`,
	} {
		if _, err := raw.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}

	db := open(t, path)
	rows, err := db.read.Query(`SELECT deviceId, teamId FROM device_scope ORDER BY deviceId, teamId`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got [][2]string
	for rows.Next() {
		var pair [2]string
		if err := rows.Scan(&pair[0], &pair[1]); err != nil {
			t.Fatal(err)
		}
		got = append(got, pair)
	}
	if want := [][2]string{{"client-1", "team-default"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("device_scope after the upgrade = %v, want %v", got, want)
	}
	if missing := uncovered(t, db, "client-1", store.DefaultTeamID); missing != nil {
		t.Fatalf("client-1 uncovered after the upgrade: %v", missing)
	}
}

func memberships(t *testing.T, db *DB, accountID string) []store.Membership {
	t.Helper()
	m, err := db.Memberships(context.Background(), accountID)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// Review F-3: a snapshot that started while the account was in team-2, saw it
// leave and join again before the last page, must not cover team-2 — its pages
// may have skipped the team's rows while the account was out.
func TestDeviceScopeCoverTeamsSkipsARejoinedTeam(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	newAccount(t, db, "acct-1", "alice")
	scopeDevice(t, db, "dev-1", "acct-1")
	if err := db.CreateTeam(ctx, store.Team{ID: "team-2", Name: "Two", CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if err := db.AddTeamMember(ctx, "team-2", "acct-1", ""); err != nil {
		t.Fatal(err)
	}
	atStart := memberships(t, db, "acct-1")

	if err := db.RemoveTeamMember(ctx, "team-2", "acct-1"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Millisecond) // a new membership has a later join time
	if err := db.AddTeamMember(ctx, "team-2", "acct-1", ""); err != nil {
		t.Fatal(err)
	}
	if err := db.CoverTeams(ctx, "dev-1", atStart); err != nil {
		t.Fatal(err)
	}
	if got := uncovered(t, db, "dev-1", "team-2"); len(got) != 1 {
		t.Fatalf("a team left and joined again during the snapshot was covered")
	}
	// The membership as it is now is covered by the next snapshot.
	if err := db.CoverTeams(ctx, "dev-1", memberships(t, db, "acct-1")); err != nil {
		t.Fatal(err)
	}
	if got := uncovered(t, db, "dev-1", "team-2"); got != nil {
		t.Fatalf("after a clean snapshot: uncovered %v", got)
	}
}
