package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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
	// ErrRefreshReused means a refresh token was presented a second time. The
	// whole family has been revoked by the time this is returned; the client
	// sees the same 401 as for any bad credential.
	ErrRefreshReused = errors.New("auth: refresh token reused")
)

// Default token-pair lifetimes.
const (
	DefaultAccessTokenTTL  = 15 * time.Minute
	DefaultRefreshTokenTTL = 30 * 24 * time.Hour
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
	// OTP is the six-digit code, or a recovery code, for an account with
	// one-time codes enabled (0.1.1). Without one such an account gets
	// ErrOTPRequired, which desktop 1.3.0 answers by asking for the code.
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
	// ExpiresAt is nil for a token that never expires.
	ExpiresAt *time.Time
}

// Policy holds the token lifetimes. The zero value is server 0.1.0's
// behaviour: device tokens that never expire.
type Policy struct {
	// DeviceTokenTTL is the lifetime of a device token issued by Login. Zero
	// means no expiry.
	DeviceTokenTTL time.Duration
	// AccessTokenTTL and RefreshTokenTTL are the token-pair lifetimes; zero
	// means the defaults above. The refresh lifetime restarts at every
	// rotation, so a device that syncs at least once per period stays signed in.
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

func (p Policy) accessTTL() time.Duration {
	if p.AccessTokenTTL > 0 {
		return p.AccessTokenTTL
	}
	return DefaultAccessTokenTTL
}

func (p Policy) refreshTTL() time.Duration {
	if p.RefreshTokenTTL > 0 {
		return p.RefreshTokenTTL
	}
	return DefaultRefreshTokenTTL
}

// PairResult carries the only plaintext copies of an access/refresh pair.
type PairResult struct {
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
	Identity         Identity
}

// Service implements login and token authentication over a store.
type Service struct {
	store   store.Store
	now     func() time.Time
	limiter *Limiter
	policy  Policy
}

func NewService(st store.Store, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{store: st, now: now, limiter: NewLimiter(now)}
}

// WithPolicy sets the token lifetimes and returns the service for chaining.
func (s *Service) WithPolicy(p Policy) *Service {
	s.policy = p
	return s
}

// CheckPassword verifies a username and password without issuing anything.
//
// The dashboard uses it: a browser login must not create a device or a sync
// token. Rate limiting and the equal-time failure path are shared with Login,
// because an attacker should not be able to pick the cheaper door.
//
// otp is the one-time code (or recovery code) for an account that has them
// enabled; it is ignored otherwise. A correct password without a code is
// ErrOTPRequired. A wrong code costs a rate-limit attempt like a wrong password.
func (s *Service) CheckPassword(ctx context.Context, username, password, otp, remoteIP string) (*store.Account, error) {
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
	if err := s.verifyOTP(ctx, account.ID, otp); err != nil {
		return nil, err
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

	account, err := s.CheckPassword(ctx, req.Username, req.Password, req.OTP, req.RemoteIP)
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
	if s.policy.DeviceTokenTTL > 0 {
		// Only tokens issued from now on carry an expiry; the operator turning
		// the setting on does not cut off devices that are already signed in.
		expires := now.Add(s.policy.DeviceTokenTTL)
		device.TokenExpiresAt = &expires
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
		ExpiresAt: device.TokenExpiresAt,
	}, nil
}

// IssuePair verifies a username and password and issues an access/refresh
// pair for the device.
//
// Unlike Login it does not touch the device token: a device that is also
// signed in the 0.1 way (desktop 1.3.0) keeps that token working.
func (s *Service) IssuePair(ctx context.Context, req LoginRequest) (PairResult, error) {
	clientID := strings.TrimSpace(req.ClientID)
	if clientID == "" {
		return PairResult{}, ErrMissingClientID
	}

	account, err := s.CheckPassword(ctx, req.Username, req.Password, req.OTP, req.RemoteIP)
	if err != nil {
		return PairResult{}, err
	}

	now := s.now()
	if err := s.store.RegisterDevice(ctx, store.Device{
		ID:         clientID,
		AccountID:  account.ID,
		Name:       deviceName(req.DeviceName),
		CreatedAt:  now,
		LastSeenAt: &now,
	}); err != nil {
		return PairResult{}, fmt.Errorf("register device: %w", err)
	}

	family, err := newFamilyID()
	if err != nil {
		return PairResult{}, err
	}
	result, pair, err := s.newPair(clientID, family, now)
	if err != nil {
		return PairResult{}, err
	}
	if err := s.store.CreateTokenPair(ctx, pair); err != nil {
		return PairResult{}, fmt.Errorf("store token pair: %w", err)
	}

	teams, err := s.store.TeamsOf(ctx, account.ID)
	if err != nil {
		return PairResult{}, fmt.Errorf("teams of %s: %w", account.ID, err)
	}
	result.Identity = Identity{AccountID: account.ID, DeviceID: clientID, TeamIDs: teams}
	return result, nil
}

// Refresh exchanges a refresh token for a new pair. Each refresh token works
// once; presenting it again revokes every token of its family and returns
// ErrRefreshReused, because only a stolen copy is ever replayed.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (PairResult, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return PairResult{}, ErrInvalidCredentials
	}
	hash := HashToken(refreshToken)

	token, err := s.store.FindAuthToken(ctx, hash)
	if errors.Is(err, store.ErrNotFound) {
		return PairResult{}, ErrInvalidCredentials
	}
	if err != nil {
		return PairResult{}, fmt.Errorf("find refresh token: %w", err)
	}
	if token.Kind != store.TokenRefresh {
		return PairResult{}, ErrInvalidCredentials
	}

	now := s.now()
	if token.UsedAt != nil && token.RevokedAt == nil {
		return PairResult{}, s.reused(ctx, token.FamilyID, now)
	}
	if !token.Usable(now) {
		return PairResult{}, ErrInvalidCredentials
	}

	identity, err := s.identityOfDevice(ctx, token.DeviceID)
	if err != nil {
		return PairResult{}, err
	}

	result, pair, err := s.newPair(token.DeviceID, token.FamilyID, now)
	if err != nil {
		return PairResult{}, err
	}
	err = s.store.RotateRefresh(ctx, hash, now, pair)
	switch {
	case errors.Is(err, store.ErrTokenUsed):
		// Lost a race with another exchange of the same token: that is a
		// replay as well.
		return PairResult{}, s.reused(ctx, token.FamilyID, now)
	case errors.Is(err, store.ErrNotFound):
		return PairResult{}, ErrInvalidCredentials
	case err != nil:
		return PairResult{}, fmt.Errorf("rotate refresh token: %w", err)
	}

	result.Identity = identity
	return result, nil
}

func (s *Service) reused(ctx context.Context, family string, now time.Time) error {
	if err := s.store.RevokeTokenFamily(ctx, family, now); err != nil {
		return fmt.Errorf("revoke token family after reuse: %w", err)
	}
	return ErrRefreshReused
}

func (s *Service) newPair(deviceID, family string, now time.Time) (PairResult, store.TokenPair, error) {
	access, accessHash, err := NewToken()
	if err != nil {
		return PairResult{}, store.TokenPair{}, err
	}
	refresh, refreshHash, err := NewToken()
	if err != nil {
		return PairResult{}, store.TokenPair{}, err
	}
	result := PairResult{
		AccessToken:      access,
		AccessExpiresAt:  now.Add(s.policy.accessTTL()),
		RefreshToken:     refresh,
		RefreshExpiresAt: now.Add(s.policy.refreshTTL()),
	}
	pair := store.TokenPair{
		Access: store.AuthToken{TokenHash: accessHash, DeviceID: deviceID, Kind: store.TokenAccess,
			FamilyID: family, CreatedAt: now, ExpiresAt: result.AccessExpiresAt},
		Refresh: store.AuthToken{TokenHash: refreshHash, DeviceID: deviceID, Kind: store.TokenRefresh,
			FamilyID: family, CreatedAt: now, ExpiresAt: result.RefreshExpiresAt},
	}
	return result, pair, nil
}

// Authenticate resolves a bearer token to an identity. It accepts a device
// token or an access token; a refresh token is never a sync credential.
func (s *Service) Authenticate(ctx context.Context, token string) (Identity, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return Identity{}, ErrInvalidCredentials
	}
	hash := HashToken(token)

	device, err := s.store.FindDeviceByTokenHash(ctx, hash)
	if errors.Is(err, store.ErrNotFound) {
		return s.authenticateAccess(ctx, hash)
	}
	if err != nil {
		return Identity{}, fmt.Errorf("find device: %w", err)
	}
	if device.Revoked() || device.TokenExpired(s.now()) {
		// Revoked, expired and unknown are the same answer to the client: it
		// has to sign in again either way.
		return Identity{}, ErrInvalidCredentials
	}
	return s.identityOf(ctx, device)
}

func (s *Service) authenticateAccess(ctx context.Context, hash []byte) (Identity, error) {
	token, err := s.store.FindAuthToken(ctx, hash)
	if errors.Is(err, store.ErrNotFound) {
		return Identity{}, ErrInvalidCredentials
	}
	if err != nil {
		return Identity{}, fmt.Errorf("find access token: %w", err)
	}
	if token.Kind != store.TokenAccess || !token.Usable(s.now()) {
		return Identity{}, ErrInvalidCredentials
	}
	return s.identityOfDevice(ctx, token.DeviceID)
}

func (s *Service) identityOfDevice(ctx context.Context, deviceID string) (Identity, error) {
	device, err := s.store.FindDeviceByID(ctx, deviceID)
	if errors.Is(err, store.ErrNotFound) {
		return Identity{}, ErrInvalidCredentials
	}
	if err != nil {
		return Identity{}, fmt.Errorf("find device %s: %w", deviceID, err)
	}
	if device.Revoked() {
		return Identity{}, ErrInvalidCredentials
	}
	return s.identityOf(ctx, device)
}

// identityOf finishes authentication for a device that passed its token
// check: the account must still be allowed in.
func (s *Service) identityOf(ctx context.Context, device *store.Device) (Identity, error) {
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

func newFamilyID() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("auth: read family id: %w", err)
	}
	return "fam_" + hex.EncodeToString(raw), nil
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
