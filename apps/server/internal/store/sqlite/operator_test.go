package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/team-croffle/croffle/apps/server/internal/store"
)

func TestOperatorGrantAndRevoke(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	newAccount(t, db, "acct-1", "alice")
	newAccount(t, db, "acct-2", "bob")

	account, _ := db.FindAccountByID(ctx, "acct-1")
	if account.Operator() {
		t.Fatal("a new account is an operator")
	}
	if err := db.SetOperator(ctx, "acct-1", true, false); err != nil {
		t.Fatal(err)
	}
	if err := db.SetOperator(ctx, "acct-2", true, false); err != nil {
		t.Fatal(err)
	}
	accounts, _ := db.ListAccounts(ctx)
	for _, a := range accounts {
		if !a.Operator() {
			t.Fatalf("%s is not an operator after grant", a.Username)
		}
	}
	// Two operators: one may go.
	if err := db.SetOperator(ctx, "acct-2", false, false); err != nil {
		t.Fatalf("revoking one of two: %v", err)
	}
	if err := db.SetOperator(ctx, "missing", true, false); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("grant to a missing account: %v", err)
	}
}

func TestLastOperatorIsProtected(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	newAccount(t, db, "acct-1", "alice")
	newAccount(t, db, "acct-2", "bob")
	if err := db.SetOperator(ctx, "acct-1", true, false); err != nil {
		t.Fatal(err)
	}

	if err := db.SetOperator(ctx, "acct-1", false, false); !errors.Is(err, store.ErrLastOperator) {
		t.Fatalf("revoke the last operator: %v, want ErrLastOperator", err)
	}
	if err := db.SetAccountDisabled(ctx, "acct-1", true, false); !errors.Is(err, store.ErrLastOperator) {
		t.Fatalf("disable the last operator: %v, want ErrLastOperator", err)
	}
	// A disabled operator does not count: bob as a disabled operator leaves
	// alice the last active one.
	if err := db.SetOperator(ctx, "acct-2", true, false); err != nil {
		t.Fatal(err)
	}
	if err := db.SetAccountDisabled(ctx, "acct-2", true, false); err != nil {
		t.Fatalf("disable one of two operators: %v", err)
	}
	if err := db.SetOperator(ctx, "acct-1", false, false); !errors.Is(err, store.ErrLastOperator) {
		t.Fatalf("revoke with only a disabled second operator: %v, want ErrLastOperator", err)
	}
	// Regular accounts are never blocked by the rule.
	if err := db.SetAccountDisabled(ctx, "acct-2", false, false); err != nil {
		t.Fatal(err)
	}
	// --force goes through.
	if err := db.SetOperator(ctx, "acct-2", false, false); err != nil {
		t.Fatal(err)
	}
	if err := db.SetOperator(ctx, "acct-1", false, true); err != nil {
		t.Fatalf("forced revoke: %v", err)
	}
	account, _ := db.FindAccountByID(ctx, "acct-1")
	if account.Operator() {
		t.Fatal("forced revoke left the account an operator")
	}
}

// Review F-4: the new password and the end of every browser session are one
// transaction.
func TestResetPasswordEndsSessions(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	newAccount(t, db, "acct-1", "alice")
	now := time.Now()
	for _, token := range []string{"s1", "s2"} {
		if err := db.CreateSession(ctx, store.Session{
			TokenHash: []byte(token), AccountID: "acct-1", CreatedAt: now, ExpiresAt: now.Add(time.Hour),
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.ResetPassword(ctx, "acct-1", "new-hash"); err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"s1", "s2"} {
		if _, err := db.FindSession(ctx, []byte(token), now); !errors.Is(err, store.ErrNotFound) {
			t.Fatalf("session %s after the reset: %v", token, err)
		}
	}
	account, _ := db.FindAccountByID(ctx, "acct-1")
	if account.PasswordHash != "new-hash" {
		t.Fatal("the password was not changed")
	}
	if err := db.ResetPassword(ctx, "missing", "x"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("reset a missing account: %v", err)
	}
}

// Review F-4: a sign-in checked against the old password cannot create its
// session after the reset, and a disabled account cannot either.
func TestCreateSessionRefusesAStalePassword(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	newAccount(t, db, "acct-1", "alice") // passwordHash 'hash'
	now := time.Now()
	session := func(token, hash string) error {
		return db.CreateSession(ctx, store.Session{
			TokenHash: []byte(token), AccountID: "acct-1", CreatedAt: now, ExpiresAt: now.Add(time.Hour),
			PasswordHash: hash,
		})
	}
	if err := session("ok", "hash"); err != nil {
		t.Fatalf("the current password: %v", err)
	}
	if err := db.ResetPassword(ctx, "acct-1", "new-hash"); err != nil {
		t.Fatal(err)
	}
	if err := session("late", "hash"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("a sign-in checked before the reset: %v, want ErrNotFound", err)
	}
	if _, err := db.FindSession(ctx, []byte("late"), now); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("the late session exists")
	}
	if err := db.SetAccountDisabled(ctx, "acct-1", true, false); err != nil {
		t.Fatal(err)
	}
	if err := session("disabled", "new-hash"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("a disabled account: %v, want ErrNotFound", err)
	}
}
