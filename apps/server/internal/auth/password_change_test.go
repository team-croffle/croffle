package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/team-croffle/croffle/apps/server/internal/store"
)

// Spec 005 S-12: the account changes its own password after proving the
// current one; the asking session is passed on to be kept.
func TestChangePasswordChecksTheCurrentPassword(t *testing.T) {
	svc, st, _ := newFixture(t, "hunter2")
	ctx := context.Background()
	keep := []byte("session-a")

	if err := svc.ChangePassword(ctx, "alice", "wrong", "", "new-pass-1", "ip", keep); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong current password: %v, want ErrInvalidCredentials", err)
	}
	for _, bad := range []string{"", strings.Repeat("x", MaxPassword+1)} {
		if err := svc.ChangePassword(ctx, "alice", "hunter2", "", bad, "ip", keep); !errors.Is(err, ErrInvalidNewPassword) {
			t.Fatalf("new password of %d bytes: %v, want ErrInvalidNewPassword", len(bad), err)
		}
	}
	if err := svc.ChangePassword(ctx, "alice", "hunter2", "", "new-pass-1", "ip", keep); err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}
	if ok, _ := VerifyPassword(st.account.PasswordHash, "new-pass-1"); !ok {
		t.Fatal("the stored hash is not the new password")
	}
	if string(st.keptSession) != "session-a" {
		t.Fatalf("kept session %q, want the asking one", st.keptSession)
	}
	if _, err := svc.CheckPassword(ctx, "alice", "hunter2", "", "ip"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("old password after the change: %v", err)
	}
}

// A wrong current password costs an attempt like a wrong login.
func TestChangePasswordIsRateLimited(t *testing.T) {
	svc, _, _ := newFixture(t, "hunter2")
	ctx := context.Background()
	var err error
	for range 20 {
		err = svc.ChangePassword(ctx, "alice", "wrong", "", "new-pass-1", "ip", nil)
		if errors.Is(err, ErrRateLimited) {
			break
		}
	}
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("after many wrong attempts: %v, want ErrRateLimited", err)
	}
}

func TestChangePasswordWithOTPNeedsACode(t *testing.T) {
	f := newOTPFixture(t)
	ctx := context.Background()
	if err := f.svc.ChangePassword(ctx, "alice", "hunter2", "", "new-pass-1", "ip", nil); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("without a code: %v, want ErrInvalidCredentials", err)
	}
	if err := f.svc.ChangePassword(ctx, "alice", "hunter2", f.codes[0], "new-pass-1", "ip", nil); err != nil {
		t.Fatalf("with a recovery code: %v", err)
	}
	if ok, _ := VerifyPassword(f.st.account.PasswordHash, "new-pass-1"); !ok {
		t.Fatal("the password was not changed")
	}
}

// staleStore is a store whose password changed between the check and the
// write (an operator reset).
type staleStore struct{ *fakeStore }

func (staleStore) ChangePassword(context.Context, string, string, string, []byte) error {
	return store.ErrNotFound
}

func TestChangePasswordRefusesAPasswordChangedMeanwhile(t *testing.T) {
	_, st, now := newFixture(t, "hunter2")
	svc := NewService(staleStore{st}, func() time.Time { return now })
	err := svc.ChangePassword(context.Background(), "alice", "hunter2", "", "new-pass-1", "ip", nil)
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("a reset in between: %v, want ErrInvalidCredentials", err)
	}
}
