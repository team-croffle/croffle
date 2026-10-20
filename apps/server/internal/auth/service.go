package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/team-croffle/croffle/apps/server/internal/store"
)

// lastSeenInterval is how stale device.lastSeenAt may get before a request
// refreshes it. Writing it on every request would push each sync through the
// single write connection for no benefit.
const lastSeenInterval = 5 * time.Minute

var (
	// ErrInvalidCredentials covers "no such user", "wrong password" and "bad
	// token" on purpose: the client learns nothing about which it was.
	ErrInvalidCredentials = errors.New("auth: invalid credentials")
	// ErrAccountDisabled is reported separately from a wrong password because
	// the operator needs to be able to tell them apart in the logs.
	ErrAccountDisabled = errors.New("auth: account disabled")
	// ErrOTPRequired asks the client to collect a one-time code and retry.
	ErrOTPRequired = errors.New("auth: otp required")
	// ErrRateLimited means too many failed attempts for this ip and username.
	ErrRateLimited = errors.New("auth: too many attempts")
	// ErrMissingClientID is returned when a login arrives without the device
	// identity the desktop is supposed to send.
	ErrMissingClientID = errors.New("auth: missing clientId")
)

// Identity is who a request belongs to, resolved from a token.
type Identity struct {
	AccountID string
	DeviceID  string
	// TeamIDs is what makes shared rows visible. Server 0.1 seeds one team;
	// the slice shape is already here for multi-team in 0.3.
	TeamIDs []string
}

// LoginRequest is the credential half of POST /auth/login.
type LoginRequest struct {
	Username string
	Password string
	// OTP is accepted now so the desktop's flow exists end to end; 0.1 has no
	// way to enrol a TOTP secret, so it is only validated once that lands.
	OTP string
	// ClientID is the desktop's own clientId and becomes the device id, which
	// is also what appears in a row's lastWriterId.
	ClientID   string
	DeviceName string
	// RemoteIP is used for rate limiting only.
	RemoteIP string
}

// LoginResult carries the only plaintext copy of the token.
type LoginResult struct {
	Token    string
	Identity Identity
}

// Service implements login and token authentication over a store.
type Service struct {
	store   store.Store
	now     func() time.Time
	limiter *Limiter
}

func NewService(st store.Store, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{store: st, now: now, limiter: NewLimiter(now)}
}

// CheckPassword verifies a username and password without issuing anything.
//
// The dashboard uses it: a browser login must not create a device or a sync
// token. Rate limiting and the equal-time failure path are shared with Login,
// because an attacker should not be able to pick the cheaper door.
func (s *Service) CheckPassword(ctx context.Context, username, password, remoteIP string) (*store.Account, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	limitKey := remoteIP + "|" + username
	if !s.limiter.Allow(limitKey) {
		return nil, ErrRateLimited
	}

	account, err := s.store.FindAccountByUsername(ctx, username)
	if errors.Is(err, store.ErrNotFound) {
		// Spend the same time a real verification would, so the response time
		// does not reveal which usernames exist.
		BurnPasswordTime(password)
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("find account: %w", err)
	}

	ok, err := VerifyPassword(account.PasswordHash, password)
	if err != nil {
		// A hash this server cannot read is an operational problem, not a
		// wrong password; surface it instead of logging the user out silently.
		return nil, fmt.Errorf("verify password for %s: %w", account.ID, err)
	}
	if !ok {
		return nil, ErrInvalidCredentials
	}
	if account.Disabled() {
		return nil, ErrAccountDisabled
	}

	s.limiter.Reset(limitKey)
	return account, nil
}

// Login verifies a username and password and issues a device token.
//
// Logging in again from the same clientId replaces that device's token: the
// desktop keeps one credential per installation, and a replaced token stops
// working immediately.
func (s *Service) Login(ctx context.Context, req LoginRequest) (LoginResult, error) {
	clientID := strings.TrimSpace(req.ClientID)
	if clientID == "" {
		return LoginResult{}, ErrMissingClientID
	}

	account, err := s.CheckPassword(ctx, req.Username, req.Password, req.RemoteIP)
	if err != nil {
		return LoginResult{}, err
	}

	token, hash, err := NewToken()
	if err != nil {
		return LoginResult{}, err
	}

	now := s.now()
	device := store.Device{
		ID:         clientID,
		AccountID:  account.ID,
		Name:       deviceName(req.DeviceName),
		TokenHash:  hash,
		CreatedAt:  now,
		LastSeenAt: &now,
	}
	if err := s.store.UpsertDevice(ctx, device); err != nil {
		return LoginResult{}, fmt.Errorf("upsert device: %w", err)
	}

	teams, err := s.store.TeamsOf(ctx, account.ID)
	if err != nil {
		return LoginResult{}, fmt.Errorf("teams of %s: %w", account.ID, err)
	}

	return LoginResult{
		Token: token,
		Identity: Identity{
			AccountID: account.ID,
			DeviceID:  device.ID,
			TeamIDs:   teams,
		},
	}, nil
}

// Authenticate resolves a bearer token to an identity.
func (s *Service) Authenticate(ctx context.Context, token string) (Identity, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return Identity{}, ErrInvalidCredentials
	}

	device, err := s.store.FindDeviceByTokenHash(ctx, HashToken(token))
	if errors.Is(err, store.ErrNotFound) {
		return Identity{}, ErrInvalidCredentials
	}
	if err != nil {
		return Identity{}, fmt.Errorf("find device: %w", err)
	}
	if device.Revoked() {
		// Revoked and unknown are the same answer to the client; the log line
		// is where the two are distinguishable.
		return Identity{}, ErrInvalidCredentials
	}

	account, err := s.store.FindAccountByID(ctx, device.AccountID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return Identity{}, ErrInvalidCredentials
		}
		return Identity{}, fmt.Errorf("find account %s: %w", device.AccountID, err)
	}
	if account.Disabled() {
		return Identity{}, ErrAccountDisabled
	}

	teams, err := s.store.TeamsOf(ctx, account.ID)
	if err != nil {
		return Identity{}, fmt.Errorf("teams of %s: %w", account.ID, err)
	}

	s.touch(ctx, device)

	return Identity{AccountID: account.ID, DeviceID: device.ID, TeamIDs: teams}, nil
}

// touch refreshes lastSeenAt when it is stale enough to be worth a write.
// Failures are ignored: it is bookkeeping, not part of the request's result.
func (s *Service) touch(ctx context.Context, device *store.Device) {
	now := s.now()
	if device.LastSeenAt != nil && now.Sub(*device.LastSeenAt) < lastSeenInterval {
		return
	}
	_ = s.store.TouchDevice(ctx, device.ID, now)
}

func deviceName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "unnamed device"
	}
	if len(name) > 100 {
		return name[:100]
	}
	return name
}
