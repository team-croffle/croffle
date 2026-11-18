package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/team-croffle/croffle/apps/server/internal/store"
)

// Spec 005 S-12: changing one's own password keeps the asking session, ends the
// others, and refuses when the stored hash is no longer the one checked.
func TestChangePasswordKeepsTheAskingSession(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	newAccount(t, db, "acct-1", "alice") // passwordHash 'hash'
	newAccount(t, db, "acct-2", "bob")
	now := time.Now()
	for _, s := range []struct{ token, account string }{{"a", "acct-1"}, {"b", "acct-1"}, {"other", "acct-2"}} {
		if err := db.CreateSession(ctx, store.Session{
			TokenHash: []byte(s.token), AccountID: s.account, CreatedAt: now, ExpiresAt: now.Add(time.Hour),
		}); err != nil {
			t.Fatal(err)
		}
	}

	if err := db.ChangePassword(ctx, "acct-1", "stale", "new-hash", []byte("a")); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("with a stale old hash: %v, want ErrNotFound", err)
	}
	if _, err := db.FindSession(ctx, []byte("b"), now); err != nil {
		t.Fatalf("a refused change ended a session: %v", err)
	}

	if err := db.ChangePassword(ctx, "acct-1", "hash", "new-hash", []byte("a")); err != nil {
		t.Fatal(err)
	}
	account, _ := db.FindAccountByID(ctx, "acct-1")
	if account.PasswordHash != "new-hash" {
		t.Fatal("the password was not changed")
	}
	if _, err := db.FindSession(ctx, []byte("a"), now); err != nil {
		t.Fatalf("the asking session ended: %v", err)
	}
	if _, err := db.FindSession(ctx, []byte("b"), now); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("the other session of the account: %v, want ErrNotFound", err)
	}
	if _, err := db.FindSession(ctx, []byte("other"), now); err != nil {
		t.Fatalf("another account's session ended: %v", err)
	}
}
