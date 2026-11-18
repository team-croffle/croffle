package sqlite

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/team-croffle/croffle/apps/server/internal/secretbox"
	"github.com/team-croffle/croffle/apps/server/internal/store"
)

func testBox(t *testing.T, fill byte) *secretbox.AEAD {
	t.Helper()
	key := make([]byte, secretbox.KeySize)
	for i := range key {
		key[i] = fill
	}
	box, err := secretbox.New(key)
	if err != nil {
		t.Fatal(err)
	}
	return box
}

func readSecret(t *testing.T, db *DB, accountID string) string {
	t.Helper()
	var value string
	if err := db.read.QueryRow(`SELECT secret FROM account_otp WHERE accountId = ?`, accountID).Scan(&value); err != nil {
		t.Fatal(err)
	}
	return value
}

// With a key, the lifecycle of TestOTPLifecycle still holds: the stored value
// is sealed, FindOTP opens it, and EnableOTP compares opened values (a sealed
// value changes with every nonce, so an SQL comparison would never match).
func TestOTPEncryptLifecycle(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	newAccount(t, db, "acct-1", "alice")
	box := testBox(t, 1)
	if _, err := db.PrepareSecrets(ctx, box); err != nil {
		t.Fatal(err)
	}
	now := time.Now()

	if err := db.StartOTP(ctx, "acct-1", exampleSecret, now); err != nil {
		t.Fatal(err)
	}
	stored := readSecret(t, db, "acct-1")
	if !strings.HasPrefix(stored, "v1:"+box.KeyID()+":") || strings.Contains(stored, exampleSecret) {
		t.Fatalf("stored secret %q is not sealed with the key", stored)
	}
	otp, err := db.FindOTP(ctx, "acct-1")
	if err != nil || otp.Secret != exampleSecret {
		t.Fatalf("FindOTP = %+v, %v", otp, err)
	}
	if err := db.EnableOTP(ctx, "acct-1", "OTHERSECRET", 10, nil, now); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("enabling with another secret: %v", err)
	}
	if err := db.EnableOTP(ctx, "acct-1", exampleSecret, 10, [][]byte{[]byte("h")}, now); err != nil {
		t.Fatalf("enabling the pending secret: %v", err)
	}
	if state, err := db.OTPStatus(ctx, "acct-1"); err != nil || state != "enabled" {
		t.Fatalf("OTPStatus = %q, %v", state, err)
	}
}

func TestOTPEncryptRefusesASecretMovedToAnotherAccount(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	newAccount(t, db, "acct-1", "alice")
	newAccount(t, db, "acct-2", "bob")
	if _, err := db.PrepareSecrets(ctx, testBox(t, 1)); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := db.StartOTP(ctx, "acct-1", "AAAA", now); err != nil {
		t.Fatal(err)
	}
	if err := db.StartOTP(ctx, "acct-2", "BBBB", now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.write.Exec(`UPDATE account_otp SET secret = ? WHERE accountId = 'acct-2'`, readSecret(t, db, "acct-1")); err != nil {
		t.Fatal(err)
	}
	if _, err := db.FindOTP(ctx, "acct-2"); !errors.Is(err, secretbox.ErrDamaged) {
		t.Fatalf("FindOTP of a moved secret: %v, want ErrDamaged", err)
	}
}

func TestMigratePlainOTPSealsOnStart(t *testing.T) {
	path := tempPath(t)
	db := open(t, path)
	ctx := context.Background()
	newAccount(t, db, "acct-1", "alice")
	now := time.Now()
	if err := db.StartOTP(ctx, "acct-1", exampleSecret, now); err != nil {
		t.Fatal(err)
	}
	if err := db.EnableOTP(ctx, "acct-1", exampleSecret, 42, nil, now); err != nil {
		t.Fatal(err)
	}

	// Without a key: plain, counted, nothing changes.
	report, err := db.PrepareSecrets(ctx, secretbox.Plain)
	if err != nil || report.Plain != 1 || report.Sealed != 0 {
		t.Fatalf("no key: %+v, %v", report, err)
	}
	if readSecret(t, db, "acct-1") != exampleSecret {
		t.Fatal("no key: the secret changed")
	}

	box := testBox(t, 1)
	report, err = db.PrepareSecrets(ctx, box)
	if err != nil || report.Sealed != 1 {
		t.Fatalf("with a key: %+v, %v", report, err)
	}
	if !strings.HasPrefix(readSecret(t, db, "acct-1"), "v1:"+box.KeyID()+":") {
		t.Fatal("the plain secret was not sealed")
	}
	var enabledAt *int64
	var lastStep int64
	if err := db.read.QueryRow(`SELECT enabledAt, lastStep FROM account_otp WHERE accountId = 'acct-1'`).Scan(&enabledAt, &lastStep); err != nil {
		t.Fatal(err)
	}
	if enabledAt == nil || lastStep != 42 {
		t.Fatalf("sealing changed the enrolment: enabledAt %v, lastStep %d", enabledAt, lastStep)
	}
	otp, err := db.FindOTP(ctx, "acct-1")
	if err != nil || otp.Secret != exampleSecret {
		t.Fatalf("FindOTP after sealing = %+v, %v", otp, err)
	}
	// A second start with the same key changes nothing.
	if report, err := db.PrepareSecrets(ctx, box); err != nil || report.Sealed != 0 {
		t.Fatalf("second start: %+v, %v", report, err)
	}

	info, err := db.Info(ctx)
	if err != nil || info.OTPSecretsEncrypted != 1 || info.OTPSecretsPlain != 0 || info.SecretKey != "set" {
		t.Fatalf("Info = %+v, %v", info, err)
	}
}

func TestSecretBoxMissingKeyStopsStart(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	newAccount(t, db, "acct-1", "alice")
	if _, err := db.PrepareSecrets(ctx, testBox(t, 1)); err != nil {
		t.Fatal(err)
	}
	if err := db.StartOTP(ctx, "acct-1", exampleSecret, time.Now()); err != nil {
		t.Fatal(err)
	}

	if _, err := db.PrepareSecrets(ctx, secretbox.Plain); !errors.Is(err, ErrSecretsNeedKey) {
		t.Fatalf("sealed secrets without a key: %v, want ErrSecretsNeedKey", err)
	}
	other := testBox(t, 2)
	_, err := db.PrepareSecrets(ctx, other)
	if !errors.Is(err, ErrSecretsOtherKey) || !strings.Contains(err.Error(), testBox(t, 1).KeyID()) {
		t.Fatalf("sealed with another key: %v, want ErrSecretsOtherKey naming the old key", err)
	}
	// The status does not need the key: an operator can still see and clear it.
	db.SetSecretBox(secretbox.Plain)
	if state, err := db.OTPStatus(ctx, "acct-1"); err != nil || state != "pending" {
		t.Fatalf("OTPStatus without the key = %q, %v", state, err)
	}
	if err := db.DeleteOTP(ctx, "acct-1"); err != nil {
		t.Fatalf("DeleteOTP without the key: %v", err)
	}
	if _, err := db.PrepareSecrets(ctx, secretbox.Plain); err != nil {
		t.Fatalf("start after clearing: %v", err)
	}
}

func TestSecretRotate(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	newAccount(t, db, "acct-1", "alice")
	newAccount(t, db, "acct-2", "bob")
	now := time.Now()
	for _, id := range []string{"acct-1", "acct-2"} {
		if err := db.StartOTP(ctx, id, "SECRET"+id, now); err != nil {
			t.Fatal(err)
		}
	}
	one, two := testBox(t, 1), testBox(t, 2)

	// Plain to the first key: turning sealing on.
	if n, err := db.RotateSecrets(ctx, secretbox.Plain, one); err != nil || n != 2 {
		t.Fatalf("plain -> key 1: %d, %v", n, err)
	}
	// Key 1 to key 2.
	if n, err := db.RotateSecrets(ctx, one, two); err != nil || n != 2 {
		t.Fatalf("key 1 -> key 2: %d, %v", n, err)
	}
	for _, id := range []string{"acct-1", "acct-2"} {
		if !strings.HasPrefix(readSecret(t, db, id), "v1:"+two.KeyID()+":") {
			t.Fatalf("%s is not sealed with key 2", id)
		}
	}
	if _, err := db.PrepareSecrets(ctx, two); err != nil {
		t.Fatalf("start with key 2: %v", err)
	}
	if otp, err := db.FindOTP(ctx, "acct-1"); err != nil || otp.Secret != "SECRETacct-1" {
		t.Fatalf("FindOTP with key 2 = %+v, %v", otp, err)
	}
}

func TestSecretRotateIsAllOrNothing(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	newAccount(t, db, "acct-1", "alice")
	newAccount(t, db, "acct-2", "bob")
	one, two := testBox(t, 1), testBox(t, 2)
	if _, err := db.PrepareSecrets(ctx, one); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	for _, id := range []string{"acct-1", "acct-2"} {
		if err := db.StartOTP(ctx, id, "SECRET"+id, now); err != nil {
			t.Fatal(err)
		}
	}
	damaged := "v1:deadbeef:" + base64.StdEncoding.EncodeToString([]byte("not a valid ciphertext"))
	if _, err := db.write.Exec(`UPDATE account_otp SET secret = ? WHERE accountId = 'acct-2'`, damaged); err != nil {
		t.Fatal(err)
	}
	before := readSecret(t, db, "acct-1")
	if _, err := db.RotateSecrets(ctx, one, two); err == nil {
		t.Fatal("rotate with a secret that does not open succeeded")
	}
	if readSecret(t, db, "acct-1") != before {
		t.Fatal("a failed rotate changed another secret")
	}
}

// exampleSecret is a base32 TOTP secret made when the tests start: no
// secret-shaped literal sits in the tree (knowledge: no credential literals
// in tests).
var exampleSecret = func() string {
	buf := make([]byte, 10)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf)
}()
