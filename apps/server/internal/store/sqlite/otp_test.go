package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/team-croffle/croffle/apps/server/internal/store"
)

func TestOTPLifecycle(t *testing.T) {
	db := open(t, tempPath(t))
	ctx := context.Background()
	newAccount(t, db, "acct-1", "alice")
	now := time.Now()

	if _, err := db.FindOTP(ctx, "acct-1"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("no enrolment: %v", err)
	}
	if err := db.StartOTP(ctx, "acct-1", "SECRET1", now); err != nil {
		t.Fatal(err)
	}
	// A pending enrolment can be restarted with a new secret.
	if err := db.StartOTP(ctx, "acct-1", "SECRET2", now); err != nil {
		t.Fatal(err)
	}
	otp, err := db.FindOTP(ctx, "acct-1")
	if err != nil || otp.Secret != "SECRET2" || otp.Enabled() {
		t.Fatalf("pending = %+v, %v", otp, err)
	}

	hashes := [][]byte{[]byte("h1"), []byte("h2")}
	// The secret the code was checked against has to be the pending one.
	if err := db.EnableOTP(ctx, "acct-1", "SECRET1", 100, hashes, now); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("enabling a replaced secret: %v", err)
	}
	if err := db.EnableOTP(ctx, "acct-1", "SECRET2", 100, hashes, now); err != nil {
		t.Fatal(err)
	}
	if err := db.EnableOTP(ctx, "acct-1", "SECRET2", 101, hashes, now); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("enabling twice: %v", err)
	}
	if err := db.StartOTP(ctx, "acct-1", "SECRET3", now); !errors.Is(err, store.ErrDuplicate) {
		t.Fatalf("restart while enabled: %v", err)
	}
	if got := countRows(t, db, "otp_recovery_code"); got != 2 {
		t.Fatalf("recovery codes = %d", got)
	}

	steps := []struct {
		step int64
		want bool
	}{{100, false}, {99, false}, {101, true}, {101, false}, {103, true}}
	for _, tc := range steps {
		got, err := db.AdvanceOTPStep(ctx, "acct-1", tc.step)
		if err != nil || got != tc.want {
			t.Errorf("advance to %d = %v, %v; want %v", tc.step, got, err, tc.want)
		}
	}

	if used, _ := db.UseRecoveryCode(ctx, "acct-1", []byte("h1"), now); !used {
		t.Error("first use of a recovery code refused")
	}
	if used, _ := db.UseRecoveryCode(ctx, "acct-1", []byte("h1"), now); used {
		t.Error("recovery code used twice")
	}
	if used, _ := db.UseRecoveryCode(ctx, "acct-1", []byte("nope"), now); used {
		t.Error("unknown recovery code accepted")
	}

	if err := db.DeleteOTP(ctx, "acct-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.FindOTP(ctx, "acct-1"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("after delete: %v", err)
	}
	if got := countRows(t, db, "otp_recovery_code"); got != 0 {
		t.Fatalf("recovery codes after delete = %d", got)
	}
}
