package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"errors"
	"fmt"
	"image/png"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"

	"github.com/team-croffle/croffle/apps/server/internal/store"
)

// One-time codes are standard TOTP (RFC 6238) so any authenticator app works:
// SHA-1, six digits, 30-second steps, one step of clock drift either way.
const (
	otpIssuer        = "Croffle"
	otpPeriod        = 30
	otpSkew          = 1
	recoveryCodes    = 10
	recoveryCodeSize = 10 // bytes: 80 bits, 16 base32 characters
	otpQRSize        = 200
)

var otpOpts = totp.ValidateOpts{
	Period: otpPeriod, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1,
}

var (
	// ErrOTPEnabled is returned when enrolment starts on an account that
	// already has one-time codes switched on.
	ErrOTPEnabled = errors.New("auth: one-time codes already enabled")
	// ErrOTPNotPending is returned when a confirmation arrives with no
	// enrolment waiting for one.
	ErrOTPNotPending = errors.New("auth: no one-time code enrolment in progress")
	// ErrInvalidCode is a wrong code during enrolment. At sign-in a wrong code
	// is ErrInvalidCredentials, the same answer as a wrong password.
	ErrInvalidCode = errors.New("auth: invalid one-time code")
)

// OTPEnrollment is what the dashboard shows to set up an authenticator app.
type OTPEnrollment struct {
	Secret string
	URI    string
	// QRPNG is a data: URL of the otpauth URI as a PNG, so the dashboard needs
	// no QR library of its own.
	QRPNG string
}

// OTPState is the one-word status: off, pending or enabled.
func (s *Service) OTPState(ctx context.Context, accountID string) (string, error) {
	state, err := s.store.OTPStatus(ctx, accountID)
	if err != nil {
		return "", fmt.Errorf("otp status: %w", err)
	}
	return state, nil
}

// BeginOTP starts (or restarts) an enrolment and returns the new secret.
func (s *Service) BeginOTP(ctx context.Context, accountID, username string) (OTPEnrollment, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer: otpIssuer, AccountName: username,
		Period: otpPeriod, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		return OTPEnrollment{}, fmt.Errorf("generate otp secret: %w", err)
	}

	err = s.store.StartOTP(ctx, accountID, key.Secret(), s.now())
	if errors.Is(err, store.ErrDuplicate) {
		return OTPEnrollment{}, ErrOTPEnabled
	}
	if err != nil {
		return OTPEnrollment{}, fmt.Errorf("start otp: %w", err)
	}

	img, err := key.Image(otpQRSize, otpQRSize)
	if err != nil {
		return OTPEnrollment{}, fmt.Errorf("render otp qr: %w", err)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return OTPEnrollment{}, fmt.Errorf("encode otp qr: %w", err)
	}
	return OTPEnrollment{
		Secret: key.Secret(),
		URI:    key.URL(),
		QRPNG:  "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()),
	}, nil
}

// ConfirmOTP switches one-time codes on once the first code matches, and
// returns the recovery codes. They exist only in this return value.
func (s *Service) ConfirmOTP(ctx context.Context, accountID, code string) ([]string, error) {
	enrolment, err := s.store.FindOTP(ctx, accountID)
	if errors.Is(err, store.ErrNotFound) {
		return nil, ErrOTPNotPending
	}
	if err != nil {
		return nil, fmt.Errorf("find otp: %w", err)
	}
	if enrolment.Enabled() {
		return nil, ErrOTPEnabled
	}

	step, ok := matchStep(enrolment.Secret, normaliseCode(code), s.now())
	if !ok {
		return nil, ErrInvalidCode
	}

	codes := make([]string, 0, recoveryCodes)
	hashes := make([][]byte, 0, recoveryCodes)
	for range recoveryCodes {
		raw := make([]byte, recoveryCodeSize)
		if _, err := rand.Read(raw); err != nil {
			return nil, fmt.Errorf("auth: read recovery code: %w", err)
		}
		recovery := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw)
		codes = append(codes, recovery)
		hashes = append(hashes, hashRecoveryCode(recovery))
	}

	if err := s.store.EnableOTP(ctx, accountID, enrolment.Secret, step, hashes, s.now()); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, ErrOTPNotPending
		}
		return nil, fmt.Errorf("enable otp: %w", err)
	}
	return codes, nil
}

// DisableOTP switches one-time codes off. It asks for the password and a
// current code (or a recovery code), so a session left open on a shared
// machine is not enough.
func (s *Service) DisableOTP(ctx context.Context, username, password, code, remoteIP string) error {
	// CheckPassword verifies the code too when codes are enabled; a missing
	// code is just a wrong answer here, not a prompt.
	account, err := s.CheckPassword(ctx, username, password, code, remoteIP)
	if errors.Is(err, ErrOTPRequired) {
		return ErrInvalidCredentials
	}
	if err != nil {
		return err
	}
	if err := s.store.DeleteOTP(ctx, account.ID); err != nil {
		return fmt.Errorf("delete otp: %w", err)
	}
	return nil
}

// verifyOTP is the sign-in check after the password matched. A missing code
// is ErrOTPRequired; a wrong or replayed one is ErrInvalidCredentials.
func (s *Service) verifyOTP(ctx context.Context, accountID, code string) error {
	enrolment, err := s.store.FindOTP(ctx, accountID)
	if errors.Is(err, store.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("find otp: %w", err)
	}
	if !enrolment.Enabled() {
		return nil
	}

	code = normaliseCode(code)
	if code == "" {
		return ErrOTPRequired
	}

	if isTOTPCode(code) {
		step, ok := matchStep(enrolment.Secret, code, s.now())
		if !ok {
			return ErrInvalidCredentials
		}
		advanced, err := s.store.AdvanceOTPStep(ctx, accountID, step)
		if err != nil {
			return fmt.Errorf("advance otp step: %w", err)
		}
		if !advanced {
			// The code is right but its step was already used: someone saw it.
			return ErrInvalidCredentials
		}
		return nil
	}

	used, err := s.store.UseRecoveryCode(ctx, accountID, hashRecoveryCode(code), s.now())
	if err != nil {
		return fmt.Errorf("use recovery code: %w", err)
	}
	if !used {
		return ErrInvalidCredentials
	}
	return nil
}

// matchStep returns the step whose code equals code, within the allowed skew.
func matchStep(secret, code string, now time.Time) (int64, bool) {
	if !isTOTPCode(code) {
		return 0, false
	}
	current := now.Unix() / otpPeriod
	for offset := int64(-otpSkew); offset <= otpSkew; offset++ {
		step := current + offset
		want, err := totp.GenerateCodeCustom(secret, time.Unix(step*otpPeriod, 0), otpOpts)
		if err != nil {
			return 0, false
		}
		if subtle.ConstantTimeCompare([]byte(want), []byte(code)) == 1 {
			return step, true
		}
	}
	return 0, false
}

func isTOTPCode(code string) bool {
	if len(code) != 6 {
		return false
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// normaliseCode accepts what people type: spaces and dashes are dropped and a
// recovery code is case-insensitive.
func normaliseCode(code string) string {
	code = strings.Map(func(r rune) rune {
		if r == ' ' || r == '-' {
			return -1
		}
		return r
	}, code)
	return strings.ToUpper(strings.TrimSpace(code))
}

func hashRecoveryCode(code string) []byte {
	sum := sha256.Sum256([]byte(normaliseCode(code)))
	return sum[:]
}
