package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/team-croffle/croffle/apps/server/internal/store"
)

// MaxPassword is the longest password the server takes, in bytes; the CLI's
// --password-stdin and the operator API use the same limit.
const MaxPassword = 4096

// ErrInvalidNewPassword is a new password that is empty or longer than
// MaxPassword.
var ErrInvalidNewPassword = errors.New("auth: new password is empty or too long")

// ChangePassword is an account changing its own password from the dashboard.
//
// It asks for the current password, and a code when one-time codes are on,
// through CheckPassword: the same rate limit as a login, so a session left open
// on a shared machine is not enough and cannot be used to guess the password.
// Every other dashboard session of the account ends; keepSession (the token
// hash of the session that asked) stays, and device tokens are not touched.
func (s *Service) ChangePassword(ctx context.Context, username, current, code, newPassword, remoteIP string, keepSession []byte) error {
	// Checked first, so a mistyped new password does not cost an attempt.
	if newPassword == "" || len(newPassword) > MaxPassword {
		return ErrInvalidNewPassword
	}
	account, err := s.CheckPassword(ctx, username, current, code, remoteIP)
	if errors.Is(err, ErrOTPRequired) {
		return ErrInvalidCredentials
	}
	if err != nil {
		return err
	}
	hash, err := HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	err = s.store.ChangePassword(ctx, account.ID, account.PasswordHash, hash, keepSession)
	if errors.Is(err, store.ErrNotFound) {
		// The password changed between the check and the write: the current
		// password the caller gave is no longer the current one.
		return ErrInvalidCredentials
	}
	if err != nil {
		return fmt.Errorf("change password: %w", err)
	}
	return nil
}
