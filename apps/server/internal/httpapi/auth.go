package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/team-croffle/croffle/apps/server/internal/auth"
	"github.com/team-croffle/croffle/apps/server/internal/hooks"
	"github.com/team-croffle/croffle/apps/server/internal/protocol"
	"github.com/team-croffle/croffle/apps/server/internal/store"
)

// retryAfterSeconds is what a rate-limited client is told to wait. It matches
// the limiter's refill interval.
const retryAfterSeconds = 30

// Authenticator is the slice of the auth service the HTTP layer needs.
type Authenticator interface {
	Login(ctx context.Context, req auth.LoginRequest) (auth.LoginResult, error)
	Authenticate(ctx context.Context, token string) (auth.Identity, error)
	// CheckPassword is the dashboard's door: it verifies a password without
	// issuing a device token, so a browser login cannot become a sync
	// credential.
	CheckPassword(ctx context.Context, username, password, remoteIP string) (*store.Account, error)
}

// IdentityOf returns the caller resolved by requireAuth. The second result is
// false on unauthenticated routes.
func IdentityOf(ctx context.Context) (auth.Identity, bool) {
	id, ok := ctx.Value(ctxKeyIdentity).(auth.Identity)
	return id, ok
}

func (s *server) login(w http.ResponseWriter, r *http.Request) {
	if s.auth == nil {
		s.notImplemented(w, r)
		return
	}

	var req protocol.LoginRequest
	if !DecodeJSON(w, r, &req) {
		return
	}

	result, err := s.auth.Login(r.Context(), auth.LoginRequest{
		Username:   req.Username,
		Password:   req.Password,
		OTP:        req.OTP,
		ClientID:   req.ClientID,
		DeviceName: req.DeviceName,
		RemoteIP:   RemoteIP(r.Context()),
	})
	if err != nil {
		s.writeAuthError(w, r, err, slog.String("username", req.Username))
		return
	}

	// The token is in the body and nowhere else: not in the log, not in a
	// header, and in the database only as a hash.
	writeJSON(w, http.StatusOK, protocol.LoginResponse{
		Token:     result.Token,
		AccountID: result.Identity.AccountID,
		TeamID:    firstTeam(result.Identity.TeamIDs),
	})

	s.logger.InfoContext(r.Context(), "login succeeded",
		slog.String("account_id", result.Identity.AccountID),
		slog.String("device_id", result.Identity.DeviceID),
		slog.String("request_id", RequestID(r.Context())),
	)

	s.hooks.Emit(r.Context(), hooks.Event{Type: hooks.TypeDeviceRegistered, Data: hooks.DeviceRegistered{
		AccountID: result.Identity.AccountID,
		DeviceID:  result.Identity.DeviceID,
		Name:      req.DeviceName,
	}})
	s.hooks.Emit(r.Context(), hooks.Event{Type: hooks.TypeAuthSucceeded, Data: hooks.AuthSucceeded{
		AccountID: result.Identity.AccountID,
		DeviceID:  result.Identity.DeviceID,
		Method:    "password",
		Surface:   "sync",
	}})
}

func (s *server) whoami(w http.ResponseWriter, r *http.Request) {
	identity, ok := IdentityOf(r.Context())
	if !ok {
		// requireAuth is what populates this; reaching here means the route was
		// wired without it.
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	writeJSON(w, http.StatusOK, protocol.WhoAmIResponse{
		AccountID: identity.AccountID,
		TeamID:    firstTeam(identity.TeamIDs),
	})
}

// requireAuth resolves the bearer token and rejects anything it cannot place.
func (s *server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.auth == nil {
			s.notImplemented(w, r)
			return
		}

		token := auth.BearerToken(r.Header.Get("Authorization"))
		if token == "" {
			w.Header().Set("WWW-Authenticate", `Bearer realm="croffle"`)
			writeError(w, http.StatusUnauthorized, "unauthorized", "bearer token required")
			return
		}

		identity, err := s.auth.Authenticate(r.Context(), token)
		if err != nil {
			s.writeAuthError(w, r, err)
			return
		}

		next(w, r.WithContext(context.WithValue(r.Context(), ctxKeyIdentity, identity)))
	}
}

// writeAuthError maps an auth failure to a status and a stable code. The client
// branches on the code, so the mapping is part of the contract: "otp_required"
// makes the desktop show an OTP field, "unauthorized" makes it ask for a new
// login.
func (s *server) writeAuthError(w http.ResponseWriter, r *http.Request, err error, extra ...slog.Attr) {
	attrs := append([]slog.Attr{
		slog.String("remote_ip", RemoteIP(r.Context())),
		slog.String("request_id", RequestID(r.Context())),
	}, extra...)

	switch {
	case errors.Is(err, auth.ErrInvalidCredentials):
		// Unknown user, wrong password and bad token are one answer on purpose.
		s.logAuth(r, slog.LevelInfo, "authentication failed", attrs)
		writeError(w, http.StatusUnauthorized, "unauthorized", "invalid credentials")

	case errors.Is(err, auth.ErrAccountDisabled):
		s.logAuth(r, slog.LevelWarn, "authentication refused: account disabled", attrs)
		writeError(w, http.StatusForbidden, "account_disabled", "this account is disabled")

	case errors.Is(err, auth.ErrOTPRequired):
		s.logAuth(r, slog.LevelInfo, "authentication needs otp", attrs)
		writeError(w, http.StatusUnauthorized, "otp_required", "a one-time code is required")

	case errors.Is(err, auth.ErrRateLimited):
		s.logAuth(r, slog.LevelWarn, "authentication rate limited", attrs)
		w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds))
		writeError(w, http.StatusTooManyRequests, "too_many_attempts", "too many attempts, try again later")

	case errors.Is(err, auth.ErrMissingClientID):
		writeError(w, http.StatusBadRequest, "invalid_request", "clientId is required")

	default:
		// Anything else is the server's fault (unreadable hash, storage error).
		s.logger.ErrorContext(r.Context(), "authentication error",
			slog.String("error", err.Error()),
			slog.String("request_id", RequestID(r.Context())),
		)
		writeError(w, http.StatusInternalServerError, "internal", "unexpected server error")
	}
}

func (s *server) logAuth(r *http.Request, level slog.Level, msg string, attrs []slog.Attr) {
	s.logger.LogAttrs(r.Context(), level, msg, attrs...)
}

// firstTeam adapts the internal slice to the wire's single teamId. Server 0.1
// seeds exactly one team; multi-team (0.3) changes the wire, not this code.
func firstTeam(teams []string) *string {
	if len(teams) == 0 {
		return nil
	}
	team := teams[0]
	return &team
}
