package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/team-croffle/croffle/apps/server/internal/store"
)

func TestTeamCreate(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	if err := db.CreateTeam(ctx, store.Team{ID: "team-eng", Name: "Engineering", CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if err := db.CreateTeam(ctx, store.Team{ID: "team-eng", Name: "Again", CreatedAt: time.Now()}); !errors.Is(err, store.ErrDuplicate) {
		t.Fatalf("same id twice: %v, want ErrDuplicate", err)
	}
	teams, _ := db.ListTeams(ctx)
	if len(teams) != 2 || teams[0].ID != store.DefaultTeamID {
		t.Fatalf("teams = %+v, want team-default and team-eng", teams)
	}
}

func TestTeamRename(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	if err := db.RenameTeam(ctx, store.DefaultTeamID, "Everyone"); err != nil {
		t.Fatal(err)
	}
	teams, _ := db.ListTeams(ctx)
	if teams[0].Name != "Everyone" {
		t.Fatalf("name = %q", teams[0].Name)
	}
	if err := db.RenameTeam(ctx, "team-missing", "X"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("rename a missing team: %v", err)
	}
}

func TestTeamMembers(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	newAccount(t, db, "acct-1", "alice")
	newAccount(t, db, "acct-2", "bob")
	if err := db.CreateTeam(ctx, store.Team{ID: "team-eng", Name: "Engineering", CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"acct-2", "acct-1"} {
		if err := db.AddTeamMember(ctx, "team-eng", id, ""); err != nil {
			t.Fatal(err)
		}
	}
	members, err := db.ListTeamMembers(ctx, "team-eng")
	if err != nil || len(members) != 2 || members[0].Username != "alice" || members[0].Role != "member" {
		t.Fatalf("members = %+v, %v", members, err)
	}
	if _, err := db.ListTeamMembers(ctx, "team-missing"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("members of a missing team: %v", err)
	}
	if err := db.AddTeamMember(ctx, "team-missing", "acct-1", ""); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("add to a missing team: %v, want ErrNotFound", err)
	}
}

func TestTeamRemoveMember(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	newAccount(t, db, "acct-1", "alice")
	if err := db.AddTeamMember(ctx, store.DefaultTeamID, "acct-1", ""); err != nil {
		t.Fatal(err)
	}
	if err := db.RemoveTeamMember(ctx, store.DefaultTeamID, "acct-1"); err != nil {
		t.Fatal(err)
	}
	if teams, _ := db.TeamsOf(ctx, "acct-1"); len(teams) != 0 {
		t.Fatalf("still in %v", teams)
	}
	if err := db.RemoveTeamMember(ctx, store.DefaultTeamID, "acct-1"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("remove twice: %v, want ErrNotFound", err)
	}
}
