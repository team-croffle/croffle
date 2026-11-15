package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/team-croffle/croffle/apps/server/internal/store"
)

func newDevice(t *testing.T, db *DB, id, accountID string, tokenHash []byte) {
	t.Helper()
	if err := db.UpsertDevice(context.Background(), store.Device{
		ID: id, AccountID: accountID, Name: id, TokenHash: tokenHash, CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("upsert device %s: %v", id, err)
	}
}

func pair(deviceID, family, suffix string, at time.Time) store.TokenPair {
	return store.TokenPair{
		Access: store.AuthToken{TokenHash: []byte("a" + suffix), DeviceID: deviceID, Kind: store.TokenAccess,
			FamilyID: family, CreatedAt: at, ExpiresAt: at.Add(15 * time.Minute)},
		Refresh: store.AuthToken{TokenHash: []byte("r" + suffix), DeviceID: deviceID, Kind: store.TokenRefresh,
			FamilyID: family, CreatedAt: at, ExpiresAt: at.Add(30 * 24 * time.Hour)},
	}
}

func TestTokenPairCreateAndFind(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	newAccount(t, db, "acct-1", "alice")
	newDevice(t, db, "dev-1", "acct-1", []byte("device-token"))
	now := time.Now()

	if err := db.CreateTokenPair(ctx, pair("dev-1", "fam-1", "1", now)); err != nil {
		t.Fatal(err)
	}
	access, err := db.FindAuthToken(ctx, []byte("a1"))
	if err != nil {
		t.Fatal(err)
	}
	if access.Kind != store.TokenAccess || access.DeviceID != "dev-1" || !access.Usable(now) {
		t.Fatalf("access = %+v", access)
	}
	if access.Usable(now.Add(16 * time.Minute)) {
		t.Fatal("an access token is usable past its expiry")
	}
	if _, err := db.FindAuthToken(ctx, []byte("nope")); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("unknown token: %v", err)
	}
	// The device token is untouched by a pair.
	if _, err := db.FindDeviceByTokenHash(ctx, []byte("device-token")); err != nil {
		t.Fatalf("device token lost: %v", err)
	}
}

func TestRefreshRotationAndReuse(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	newAccount(t, db, "acct-1", "alice")
	newDevice(t, db, "dev-1", "acct-1", nil)
	now := time.Now()
	if err := db.CreateTokenPair(ctx, pair("dev-1", "fam-1", "1", now)); err != nil {
		t.Fatal(err)
	}

	if err := db.RotateRefresh(ctx, []byte("r1"), now, pair("dev-1", "fam-1", "2", now)); err != nil {
		t.Fatalf("first rotation: %v", err)
	}
	if err := db.RotateRefresh(ctx, []byte("r1"), now, pair("dev-1", "fam-1", "3", now)); !errors.Is(err, store.ErrTokenUsed) {
		t.Fatalf("second rotation of r1: %v, want ErrTokenUsed", err)
	}
	// The failed rotation stored nothing.
	if _, err := db.FindAuthToken(ctx, []byte("a3")); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("a replay stored a pair: %v", err)
	}
	if err := db.RotateRefresh(ctx, []byte("unknown"), now, pair("dev-1", "fam-1", "4", now)); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("unknown refresh: %v", err)
	}
	// An access token cannot be spent as a refresh token.
	if err := db.RotateRefresh(ctx, []byte("a2"), now, pair("dev-1", "fam-1", "5", now)); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("access as refresh: %v", err)
	}
	// An expired refresh token cannot be spent.
	later := now.Add(31 * 24 * time.Hour)
	if err := db.RotateRefresh(ctx, []byte("r2"), later, pair("dev-1", "fam-1", "6", later)); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expired refresh: %v", err)
	}

	if err := db.RevokeTokenFamily(ctx, "fam-1", now); err != nil {
		t.Fatal(err)
	}
	for _, hash := range []string{"a1", "r1", "a2", "r2"} {
		token, err := db.FindAuthToken(ctx, []byte(hash))
		if err != nil {
			t.Fatal(err)
		}
		if token.Usable(now) {
			t.Errorf("%s still usable after the family was revoked", hash)
		}
	}
}

func TestRevokeDeviceRevokesItsTokens(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	newAccount(t, db, "acct-1", "alice")
	newDevice(t, db, "dev-1", "acct-1", []byte("dt1"))
	newDevice(t, db, "dev-2", "acct-1", []byte("dt2"))
	now := time.Now()
	if err := db.CreateTokenPair(ctx, pair("dev-1", "fam-1", "1", now)); err != nil {
		t.Fatal(err)
	}
	if err := db.CreateTokenPair(ctx, pair("dev-2", "fam-2", "2", now)); err != nil {
		t.Fatal(err)
	}

	if err := db.RevokeAccountDevice(ctx, "acct-1", "dev-1"); err != nil {
		t.Fatal(err)
	}
	for _, hash := range []string{"a1", "r1"} {
		token, _ := db.FindAuthToken(ctx, []byte(hash))
		if token.Usable(now) {
			t.Errorf("%s survived the device revoke", hash)
		}
	}
	if token, _ := db.FindAuthToken(ctx, []byte("a2")); !token.Usable(now) {
		t.Error("another device's token was revoked")
	}

	if err := db.RevokeDevice(ctx, "dev-2"); err != nil {
		t.Fatal(err)
	}
	if token, _ := db.FindAuthToken(ctx, []byte("r2")); token.Usable(now) {
		t.Error("CLI revoke left the refresh token usable")
	}
}

func TestRegisterDeviceKeepsTheDeviceToken(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	newAccount(t, db, "acct-1", "alice")
	newAccount(t, db, "acct-2", "bob")
	newDevice(t, db, "dev-1", "acct-1", []byte("dt1"))

	if err := db.RegisterDevice(ctx, store.Device{ID: "dev-1", AccountID: "acct-1", Name: "renamed", CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	device, err := db.FindDeviceByTokenHash(ctx, []byte("dt1"))
	if err != nil {
		t.Fatalf("device token lost: %v", err)
	}
	if device.Name != "renamed" {
		t.Errorf("name = %q", device.Name)
	}
	if err := db.RegisterDevice(ctx, store.Device{ID: "dev-1", AccountID: "acct-2", Name: "x", CreatedAt: time.Now()}); !errors.Is(err, ErrDeviceOwnedByAnotherAccount) {
		t.Fatalf("foreign register: %v", err)
	}
	if err := db.RegisterDevice(ctx, store.Device{ID: "dev-new", AccountID: "acct-1", Name: "new", CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	fresh, err := db.FindDeviceByID(ctx, "dev-new")
	if err != nil || len(fresh.TokenHash) != 0 {
		t.Fatalf("new device = %+v, %v", fresh, err)
	}
}

func TestDeviceTokenExpiryIsStored(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	newAccount(t, db, "acct-1", "alice")
	expires := time.UnixMilli(1_800_000_000_000)
	if err := db.UpsertDevice(ctx, store.Device{ID: "dev-1", AccountID: "acct-1", Name: "d",
		TokenHash: []byte("dt"), CreatedAt: time.Now(), TokenExpiresAt: &expires}); err != nil {
		t.Fatal(err)
	}
	device, err := db.FindDeviceByTokenHash(ctx, []byte("dt"))
	if err != nil || device.TokenExpiresAt == nil || !device.TokenExpiresAt.Equal(expires) {
		t.Fatalf("device = %+v, %v", device, err)
	}
	if !device.TokenExpired(expires) || device.TokenExpired(expires.Add(-time.Second)) {
		t.Error("TokenExpired boundary is wrong")
	}
	// Issuing again without an expiry clears it.
	if err := db.UpsertDevice(ctx, store.Device{ID: "dev-1", AccountID: "acct-1", Name: "d",
		TokenHash: []byte("dt2"), CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	devices, _ := db.ListDevices(ctx, "acct-1")
	if len(devices) != 1 || devices[0].TokenExpiresAt != nil {
		t.Fatalf("devices = %+v", devices)
	}
}

func TestSweepExpiredAuthTokens(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	newAccount(t, db, "acct-1", "alice")
	newDevice(t, db, "dev-1", "acct-1", nil)
	now := time.Now()
	if err := db.CreateTokenPair(ctx, pair("dev-1", "fam-1", "1", now)); err != nil {
		t.Fatal(err)
	}

	// 20 minutes on: the access token is expired, the refresh token is not.
	result, err := db.Sweep(ctx, store.SweepConfig{
		CommitsBefore: now.Add(-time.Hour), OpsBefore: now.Add(-time.Hour), At: now.Add(20 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.DeletedTokens != 1 {
		t.Fatalf("DeletedTokens = %d, want 1", result.DeletedTokens)
	}
	if _, err := db.FindAuthToken(ctx, []byte("r1")); err != nil {
		t.Fatalf("live refresh token swept: %v", err)
	}
}

// TestMigrateFrom010 opens a database built by the 0.1.0 migrations only, with
// a 0.1.0 device token, and checks that 0.1.1 upgrades it in place and the
// token keeps resolving.
func TestMigrateFrom010(t *testing.T) {
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
		if m.version > 3 { // 0.1.0 shipped migrations 0001-0003
			break
		}
		if err := legacy.applyMigration(ctx, m); err != nil {
			t.Fatal(err)
		}
	}
	for _, q := range []string{
		`INSERT INTO account (id, username, passwordHash, createdAt) VALUES ('acct-1', 'alice', 'hash', 1)`,
		`INSERT INTO device (id, accountId, name, tokenHash, createdAt) VALUES ('dev-1', 'acct-1', 'laptop', X'0102', 1)`,
	} {
		if _, err := raw.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}

	db := open(t, path)
	device, err := db.FindDeviceByTokenHash(ctx, []byte{1, 2})
	if err != nil {
		t.Fatalf("0.1.0 token no longer resolves: %v", err)
	}
	if device.TokenExpiresAt != nil || device.TokenExpired(time.Now()) {
		t.Fatalf("0.1.0 token picked up an expiry: %+v", device)
	}
	for _, table := range []string{"auth_token", "account_otp", "otp_recovery_code"} {
		if got := countRows(t, db, table); got != 0 {
			t.Fatalf("%s has %d rows after upgrade", table, got)
		}
	}
	// An upgraded account has no one-time codes, so its password alone works.
	if _, err := db.FindOTP(ctx, "acct-1"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("upgraded account has an enrolment: %v", err)
	}
}
