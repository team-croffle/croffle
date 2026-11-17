package sqlite

import (
	"context"
	"errors"
	"testing"

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
