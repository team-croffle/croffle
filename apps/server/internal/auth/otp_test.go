package auth

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"

	"github.com/team-croffle/croffle/apps/server/internal/store"
)

// otpFixture is a service with a movable clock and an enrolled account.
type otpFixture struct {
	svc     *Service
	st      *fakeStore
	clock   *time.Time
	secret  string
	codes   []string
	request LoginRequest
}

func codeAt(t *testing.T, secret string, at time.Time) string {
	t.Helper()
	code, err := totp.GenerateCodeCustom(secret, at, otpOpts)
	if err != nil {
		t.Fatal(err)
	}
	return code
}

func newOTPFixture(t *testing.T) *otpFixture {
	t.Helper()
	_, st, now := newFixture(t, "hunter2")
	clock := now
	svc := NewService(st, func() time.Time { return clock })
	ctx := context.Background()

	enrolment, err := svc.BeginOTP(ctx, "acct-1", "alice")
	if err != nil {
		t.Fatalf("BeginOTP: %v", err)
	}
	codes, err := svc.ConfirmOTP(ctx, "acct-1", codeAt(t, enrolment.Secret, clock))
	if err != nil {
		t.Fatalf("ConfirmOTP: %v", err)
	}
	return &otpFixture{svc: svc, st: st, clock: &clock, secret: enrolment.Secret, codes: codes, request: validLogin()}
}

func TestTOTPEnrolment(t *testing.T) {
	_, st, now := newFixture(t, "hunter2")
	svc := NewService(st, func() time.Time { return now })
	ctx := context.Background()

	enrolment, err := svc.BeginOTP(ctx, "acct-1", "alice")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(enrolment.URI, "otpauth://totp/Croffle:alice?") ||
		!strings.Contains(enrolment.URI, "secret="+enrolment.Secret) {
		t.Errorf("uri = %q", enrolment.URI)
	}
	if !strings.HasPrefix(enrolment.QRPNG, "data:image/png;base64,") {
		t.Errorf("qr = %.40q", enrolment.QRPNG)
	}
	if _, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(enrolment.Secret); err != nil {
		t.Errorf("secret is not base32: %v", err)
	}

	// Pending: the password alone still signs in.
	if state, _ := svc.OTPState(ctx, "acct-1"); state != "pending" {
		t.Fatalf("state = %q, want pending", state)
	}
	if _, err := svc.Login(ctx, validLogin()); err != nil {
		t.Fatalf("login during enrolment: %v", err)
	}

	if _, err := svc.ConfirmOTP(ctx, "acct-1", "000000"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("wrong confirm code: %v", err)
	}
	codes, err := svc.ConfirmOTP(ctx, "acct-1", codeAt(t, enrolment.Secret, now))
	if err != nil {
		t.Fatal(err)
	}
	if len(codes) != 10 {
		t.Fatalf("recovery codes = %d", len(codes))
	}
	for _, code := range codes {
		if len(code) != 16 {
			t.Errorf("recovery code %q is not 16 characters", code)
		}
		if _, ok := st.recovery[string(hashRecoveryCode(code))]; !ok {
			t.Errorf("recovery code %q not stored as its hash", code)
		}
		for hash := range st.recovery {
			if strings.Contains(hash, code) {
				t.Error("a recovery code is stored in plain text")
			}
		}
	}
	if state, _ := svc.OTPState(ctx, "acct-1"); state != "enabled" {
		t.Fatalf("state = %q, want enabled", state)
	}
	if _, err := svc.BeginOTP(ctx, "acct-1", "alice"); !errors.Is(err, ErrOTPEnabled) {
		t.Fatalf("enrolling twice: %v", err)
	}
	if _, err := svc.ConfirmOTP(ctx, "acct-1", codeAt(t, enrolment.Secret, now)); !errors.Is(err, ErrOTPEnabled) {
		t.Fatalf("confirming twice: %v", err)
	}
}

func TestTOTPRequiredAtEveryDoor(t *testing.T) {
	f := newOTPFixture(t)
	ctx := context.Background()

	// No code: otp_required at all three doors.
	if _, err := f.svc.Login(ctx, f.request); !errors.Is(err, ErrOTPRequired) {
		t.Errorf("login without code: %v", err)
	}
	if _, err := f.svc.IssuePair(ctx, f.request); !errors.Is(err, ErrOTPRequired) {
		t.Errorf("token without code: %v", err)
	}
	if _, err := f.svc.CheckPassword(ctx, "alice", "hunter2", "", "192.0.2.1"); !errors.Is(err, ErrOTPRequired) {
		t.Errorf("dashboard without code: %v", err)
	}

	// Wrong code: the same answer as a wrong password.
	wrong := f.request
	wrong.OTP = "000000"
	if _, err := f.svc.Login(ctx, wrong); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("wrong code: %v", err)
	}

	// A wrong password with a right code is still a wrong password.
	*f.clock = f.clock.Add(30 * time.Second)
	badPassword := f.request
	badPassword.Password = "nope"
	badPassword.OTP = codeAt(t, f.secret, *f.clock)
	if _, err := f.svc.Login(ctx, badPassword); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("wrong password with a code: %v", err)
	}
}

func TestTOTPStepReplayIsRefused(t *testing.T) {
	f := newOTPFixture(t)
	ctx := context.Background()

	// The confirmation used the current step; the next step's code works once.
	*f.clock = f.clock.Add(30 * time.Second)
	next := f.request
	next.OTP = codeAt(t, f.secret, *f.clock)
	if _, err := f.svc.Login(ctx, next); err != nil {
		t.Fatalf("fresh code: %v", err)
	}
	if _, err := f.svc.IssuePair(ctx, next); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("same code again: %v", err)
	}
	earlier := f.request
	earlier.OTP = codeAt(t, f.secret, f.clock.Add(-30*time.Second))
	if _, err := f.svc.Login(ctx, earlier); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("earlier step's code: %v", err)
	}

	// Clock skew: a code one step behind the server's clock is accepted
	// when that step has not been used.
	*f.clock = f.clock.Add(90 * time.Second)
	behind := f.request
	behind.OTP = codeAt(t, f.secret, f.clock.Add(-30*time.Second))
	if _, err := f.svc.Login(ctx, behind); err != nil {
		t.Errorf("code one step behind: %v", err)
	}
	// Two steps away is outside the window.
	*f.clock = f.clock.Add(5 * time.Minute)
	far := f.request
	far.OTP = codeAt(t, f.secret, f.clock.Add(-2*time.Minute))
	if _, err := f.svc.Login(ctx, far); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("code far outside the window: %v", err)
	}
}

func TestRecoveryCodeWorksOnce(t *testing.T) {
	f := newOTPFixture(t)
	ctx := context.Background()

	withRecovery := f.request
	withRecovery.OTP = f.codes[0]
	if _, err := f.svc.IssuePair(ctx, withRecovery); err != nil {
		t.Fatalf("recovery code: %v", err)
	}
	if _, err := f.svc.Login(ctx, withRecovery); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("recovery code twice: %v", err)
	}
	// Typed with spaces, dashes and lower case, another code still works.
	typed := strings.ToLower(f.codes[1][:4] + "-" + f.codes[1][4:8] + " " + f.codes[1][8:])
	if _, err := f.svc.CheckPassword(ctx, "alice", "hunter2", typed, "192.0.2.1"); err != nil {
		t.Errorf("recovery code as typed: %v", err)
	}
}

func TestDisableOTPNeedsPasswordAndCode(t *testing.T) {
	f := newOTPFixture(t)
	ctx := context.Background()

	if err := f.svc.DisableOTP(ctx, "alice", "hunter2", "", "ip"); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("disable without code: %v", err)
	}
	if err := f.svc.DisableOTP(ctx, "alice", "nope", f.codes[0], "ip"); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("disable with a wrong password: %v", err)
	}
	if err := f.svc.DisableOTP(ctx, "alice", "hunter2", f.codes[0], "ip"); err != nil {
		t.Fatalf("disable: %v", err)
	}
	if state, _ := f.svc.OTPState(ctx, "acct-1"); state != "off" {
		t.Fatalf("state = %q", state)
	}
	if _, err := f.svc.Login(ctx, f.request); err != nil {
		t.Fatalf("password alone after disable: %v", err)
	}
}

func TestOTPFailuresCountTowardsTheRateLimit(t *testing.T) {
	f := newOTPFixture(t)
	ctx := context.Background()
	wrong := f.request
	wrong.OTP = "000000"
	for range loginBurst {
		_, _ = f.svc.Login(ctx, wrong)
	}
	if _, err := f.svc.Login(ctx, wrong); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("after %d wrong codes: %v, want ErrRateLimited", loginBurst, err)
	}
}

// racingStore swaps the pending secret right after confirmation has read it,
// the interleaving a concurrent "set up again" in another tab produces.
type racingStore struct {
	*fakeStore
	swapTo string
}

func (r *racingStore) FindOTP(ctx context.Context, accountID string) (*store.OTP, error) {
	enrolment, err := r.fakeStore.FindOTP(ctx, accountID)
	if err == nil && r.swapTo != "" {
		_ = r.fakeStore.StartOTP(ctx, accountID, r.swapTo, time.Time{})
		r.swapTo = ""
	}
	return enrolment, err
}

func TestConfirmIsBoundToTheSecretItChecked(t *testing.T) {
	_, st, now := newFixture(t, "hunter2")
	// Two fresh secrets, made while the test runs, whose codes differ at the
	// fixture time (two random secrets collide one time in a million).
	secretA, secretB := newTestSecret(t), newTestSecret(t)
	for codeAt(t, secretA, now) == codeAt(t, secretB, now) {
		secretB = newTestSecret(t)
	}
	racing := &racingStore{fakeStore: st, swapTo: secretB}
	svc := NewService(racing, func() time.Time { return now })
	ctx := context.Background()

	if err := st.StartOTP(ctx, "acct-1", secretA, now); err != nil {
		t.Fatal(err)
	}
	// The code is right for A, which confirmation read; B replaced it before
	// the enable. B must not be switched on with A's code.
	if _, err := svc.ConfirmOTP(ctx, "acct-1", codeAt(t, secretA, now)); !errors.Is(err, ErrOTPNotPending) {
		t.Fatalf("confirm across a swap: %v, want ErrOTPNotPending", err)
	}
	if st.otp.Secret != secretB || st.otp.Enabled() {
		t.Fatalf("enrolment = %+v, want B still pending", st.otp)
	}
}

func newTestSecret(t *testing.T) string {
	t.Helper()
	raw := make([]byte, 20)
	if _, err := rand.Read(raw); err != nil {
		t.Fatal(err)
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw)
}
