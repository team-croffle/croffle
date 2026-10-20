package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/team-croffle/croffle/apps/server/internal/store"
)

func newAccount(t *testing.T, db *DB, id, username string) {
	t.Helper()
	if err := db.withWriteTx(context.Background(), func(tx *sql.Tx) error {
		_, err := tx.Exec(
			`INSERT INTO account (id, username, passwordHash, createdAt) VALUES (?, ?, 'hash', ?)`,
			id, username, time.Now().UnixMilli())
		return err
	}); err != nil {
		t.Fatalf("insert account %s: %v", id, err)
	}
}

func TestFindAccount(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	newAccount(t, db, "acct-1", "alice")

	byName, err := db.FindAccountByUsername(ctx, "alice")
	if err != nil {
		t.Fatalf("FindAccountByUsername: %v", err)
	}
	if byName.ID != "acct-1" || byName.PasswordHash != "hash" {
		t.Errorf("account = %+v", byName)
	}
	if byName.Disabled() {
		t.Error("fresh account reports as disabled")
	}

	byID, err := db.FindAccountByID(ctx, "acct-1")
	if err != nil {
		t.Fatalf("FindAccountByID: %v", err)
	}
	if byID.Username != "alice" {
		t.Errorf("username = %q", byID.Username)
	}

	// A missing account is ErrNotFound, never a nil result with a nil error.
	if _, err := db.FindAccountByUsername(ctx, "nobody"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("missing account error = %v, want ErrNotFound", err)
	}
	if _, err := db.FindAccountByID(ctx, "nope"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("missing id error = %v, want ErrNotFound", err)
	}
}

func TestFindAccountReadsDisabledAt(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	newAccount(t, db, "acct-1", "alice")

	if _, err := db.write.Exec("UPDATE account SET disabledAt = ? WHERE id = 'acct-1'", 1_700_000_000_000); err != nil {
		t.Fatalf("disable: %v", err)
	}

	account, err := db.FindAccountByID(ctx, "acct-1")
	if err != nil {
		t.Fatalf("FindAccountByID: %v", err)
	}
	if !account.Disabled() {
		t.Fatal("account does not report as disabled")
	}
	if got := account.DisabledAt.UnixMilli(); got != 1_700_000_000_000 {
		t.Errorf("disabledAt = %d", got)
	}
}

func TestUpsertDeviceAndLookupByToken(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	newAccount(t, db, "acct-1", "alice")

	now := time.Now().UTC().Truncate(time.Millisecond)
	first := store.Device{
		ID: "client-1", AccountID: "acct-1", Name: "laptop",
		TokenHash: []byte("hash-one"), CreatedAt: now, LastSeenAt: &now,
	}
	if err := db.UpsertDevice(ctx, first); err != nil {
		t.Fatalf("UpsertDevice: %v", err)
	}

	got, err := db.FindDeviceByTokenHash(ctx, []byte("hash-one"))
	if err != nil {
		t.Fatalf("FindDeviceByTokenHash: %v", err)
	}
	if got.ID != "client-1" || got.AccountID != "acct-1" || got.Name != "laptop" {
		t.Errorf("device = %+v", got)
	}

	// Logging in again from the same installation replaces the token, and the
	// previous one stops working immediately.
	second := first
	second.Name = "laptop renamed"
	second.TokenHash = []byte("hash-two")
	if err := db.UpsertDevice(ctx, second); err != nil {
		t.Fatalf("UpsertDevice (replace): %v", err)
	}

	if _, err := db.FindDeviceByTokenHash(ctx, []byte("hash-one")); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("old token still resolves (err = %v)", err)
	}
	got, err = db.FindDeviceByTokenHash(ctx, []byte("hash-two"))
	if err != nil {
		t.Fatalf("FindDeviceByTokenHash (new): %v", err)
	}
	if got.Name != "laptop renamed" {
		t.Errorf("name = %q, want the updated one", got.Name)
	}

	if _, err := db.FindDeviceByTokenHash(ctx, nil); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("empty hash error = %v, want ErrNotFound", err)
	}
}

// Device ids come from the client, so claiming someone else's id must fail:
// otherwise any account could invalidate another account's device token.
func TestUpsertDeviceRefusesForeignDevice(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	newAccount(t, db, "acct-1", "alice")
	newAccount(t, db, "acct-2", "bob")

	now := time.Now()
	if err := db.UpsertDevice(ctx, store.Device{
		ID: "client-1", AccountID: "acct-1", Name: "alice laptop",
		TokenHash: []byte("alice-token"), CreatedAt: now,
	}); err != nil {
		t.Fatalf("UpsertDevice (alice): %v", err)
	}

	err := db.UpsertDevice(ctx, store.Device{
		ID: "client-1", AccountID: "acct-2", Name: "bob steals it",
		TokenHash: []byte("bob-token"), CreatedAt: now,
	})
	if !errors.Is(err, ErrDeviceOwnedByAnotherAccount) {
		t.Fatalf("error = %v, want ErrDeviceOwnedByAnotherAccount", err)
	}

	// Alice's token must still work.
	if _, err := db.FindDeviceByTokenHash(ctx, []byte("alice-token")); err != nil {
		t.Errorf("alice's token stopped working: %v", err)
	}
}

func TestUpsertDeviceClearsRevocationOnFreshLogin(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	newAccount(t, db, "acct-1", "alice")

	now := time.Now()
	if err := db.UpsertDevice(ctx, store.Device{
		ID: "client-1", AccountID: "acct-1", Name: "laptop",
		TokenHash: []byte("t1"), CreatedAt: now,
	}); err != nil {
		t.Fatalf("UpsertDevice: %v", err)
	}
	if _, err := db.write.Exec("UPDATE device SET revokedAt = 1 WHERE id = 'client-1'"); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	// Revoking withdraws a token; an account that can still prove itself may
	// get a new one. Disabling the account is what stops it for good.
	if err := db.UpsertDevice(ctx, store.Device{
		ID: "client-1", AccountID: "acct-1", Name: "laptop",
		TokenHash: []byte("t2"), CreatedAt: now,
	}); err != nil {
		t.Fatalf("UpsertDevice (re-login): %v", err)
	}

	device, err := db.FindDeviceByTokenHash(ctx, []byte("t2"))
	if err != nil {
		t.Fatalf("FindDeviceByTokenHash: %v", err)
	}
	if device.Revoked() {
		t.Error("device is still revoked after a fresh login")
	}
}

func TestUpsertDeviceRequiresIDs(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()

	if err := db.UpsertDevice(ctx, store.Device{AccountID: "acct-1"}); err == nil {
		t.Error("UpsertDevice without an id succeeded")
	}
	if err := db.UpsertDevice(ctx, store.Device{ID: "client-1"}); err == nil {
		t.Error("UpsertDevice without an account id succeeded")
	}
}

func TestTouchDevice(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	newAccount(t, db, "acct-1", "alice")

	if err := db.UpsertDevice(ctx, store.Device{
		ID: "client-1", AccountID: "acct-1", Name: "laptop",
		TokenHash: []byte("t1"), CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("UpsertDevice: %v", err)
	}

	at := time.UnixMilli(1_700_000_000_000).UTC()
	if err := db.TouchDevice(ctx, "client-1", at); err != nil {
		t.Fatalf("TouchDevice: %v", err)
	}

	device, err := db.FindDeviceByTokenHash(ctx, []byte("t1"))
	if err != nil {
		t.Fatalf("FindDeviceByTokenHash: %v", err)
	}
	if device.LastSeenAt == nil || !device.LastSeenAt.Equal(at) {
		t.Errorf("lastSeenAt = %v, want %v", device.LastSeenAt, at)
	}

	// Touching a device that is gone is not worth an error: it is bookkeeping.
	if err := db.TouchDevice(ctx, "no-such-device", at); err != nil {
		t.Errorf("TouchDevice on a missing device = %v, want nil", err)
	}
}

func TestTeamsOf(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	newAccount(t, db, "acct-1", "alice")

	teams, err := db.TeamsOf(ctx, "acct-1")
	if err != nil {
		t.Fatalf("TeamsOf: %v", err)
	}
	// Empty rather than nil: the caller builds an IN (...) clause from it.
	if teams == nil {
		t.Error("TeamsOf returned nil, want an empty slice")
	}
	if len(teams) != 0 {
		t.Errorf("teams = %v, want none before membership", teams)
	}

	if _, err := db.write.Exec(
		`INSERT INTO team_member (teamId, accountId, role, createdAt) VALUES (?, 'acct-1', 'member', 1)`,
		store.DefaultTeamID); err != nil {
		t.Fatalf("add membership: %v", err)
	}

	teams, err = db.TeamsOf(ctx, "acct-1")
	if err != nil {
		t.Fatalf("TeamsOf: %v", err)
	}
	if len(teams) != 1 || teams[0] != store.DefaultTeamID {
		t.Errorf("teams = %v, want [%s]", teams, store.DefaultTeamID)
	}
}
