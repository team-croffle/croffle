package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/team-croffle/croffle/apps/server/internal/store"
)

func TestSessionLifecycle(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	newAccount(t, db, "acct-1", "alice")

	now := time.UnixMilli(1_700_000_000_000).UTC()
	hash := []byte("session-hash")

	if err := db.CreateSession(ctx, store.Session{
		TokenHash: hash, AccountID: "acct-1",
		CreatedAt: now, ExpiresAt: now.Add(12 * time.Hour),
	}); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	session, err := db.FindSession(ctx, hash, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("FindSession: %v", err)
	}
	if session.AccountID != "acct-1" {
		t.Errorf("session = %+v", session)
	}

	// Expiry is checked on read, so a stale row can never authenticate even if
	// the sweep is behind.
	if _, err := db.FindSession(ctx, hash, now.Add(13*time.Hour)); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expired session error = %v, want ErrNotFound", err)
	}

	if err := db.DeleteSession(ctx, hash); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
	if _, err := db.FindSession(ctx, hash, now); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("deleted session error = %v, want ErrNotFound", err)
	}

	// Deleting something that is gone is not an error: logging out twice is
	// not a failure.
	if err := db.DeleteSession(ctx, hash); err != nil {
		t.Errorf("second DeleteSession: %v", err)
	}
}

func TestSessionRequiresHashAndAccount(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()

	if err := db.CreateSession(ctx, store.Session{AccountID: "acct-1"}); err == nil {
		t.Error("session without a hash was created")
	}
	if err := db.CreateSession(ctx, store.Session{TokenHash: []byte("x")}); err == nil {
		t.Error("session without an account was created")
	}
	if _, err := db.FindSession(ctx, nil, time.Now()); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("empty hash error = %v, want ErrNotFound", err)
	}
}

func TestSweepRemovesExpiredSessions(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	newAccount(t, db, "acct-1", "alice")

	now := time.Now()
	for i, expiry := range []time.Duration{-time.Hour, time.Hour} {
		if err := db.CreateSession(ctx, store.Session{
			TokenHash: []byte{byte(i)}, AccountID: "acct-1",
			CreatedAt: now.Add(-2 * time.Hour), ExpiresAt: now.Add(expiry),
		}); err != nil {
			t.Fatalf("CreateSession: %v", err)
		}
	}

	cutoff := now.Add(-90 * 24 * time.Hour)
	result, err := db.Sweep(ctx, store.SweepConfig{CommitsBefore: cutoff, OpsBefore: cutoff, At: now})
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if result.DeletedSessions != 1 {
		t.Errorf("deletedSessions = %d, want 1", result.DeletedSessions)
	}
	if rows := countRows(t, db, "dash_session"); rows != 1 {
		t.Errorf("sessions left = %d, want the live one", rows)
	}
}
