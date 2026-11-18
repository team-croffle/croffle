package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/team-croffle/croffle/apps/server/internal/store"
)

func TestPasswordRoundTrip(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	if !strings.HasPrefix(hash, "$argon2id$v=19$m=") {
		t.Errorf("hash = %q, want an encoded argon2id string", hash)
	}
	// The parameters travel with the hash so they can be raised later without
	// invalidating what is already stored.
	if !strings.Contains(hash, "t=2,p=1") {
		t.Errorf("hash = %q, want the cost parameters inline", hash)
	}

	ok, err := VerifyPassword(hash, "correct horse battery staple")
	if err != nil || !ok {
		t.Errorf("VerifyPassword(correct) = %v, %v; want true, nil", ok, err)
	}

	ok, err = VerifyPassword(hash, "wrong")
	if err != nil || ok {
		t.Errorf("VerifyPassword(wrong) = %v, %v; want false, nil", ok, err)
	}
}

func TestPasswordHashesAreSalted(t *testing.T) {
	a, _ := HashPassword("same")
	b, _ := HashPassword("same")
	if a == b {
		t.Error("two hashes of the same password are identical, salt is missing")
	}
}

func TestVerifyPasswordWithOldParameters(t *testing.T) {
	// A hash written with cheaper parameters must still verify, otherwise
	// raising the cost would lock every existing user out.
	salt := []byte("0123456789abcdef")
	old := encodeHash("secret", salt, argonParams{memory: 8 * 1024, time: 1, threads: 1, keyLen: 32})

	ok, err := VerifyPassword(old, "secret")
	if err != nil || !ok {
		t.Fatalf("VerifyPassword(old params) = %v, %v; want true, nil", ok, err)
	}
}

func TestVerifyPasswordRejectsMalformedHash(t *testing.T) {
	for _, bad := range []string{
		"", "plaintext", "$argon2i$v=19$m=1,t=1,p=1$c2FsdA$aGFzaA",
		"$argon2id$v=18$m=1,t=1,p=1$c2FsdA$aGFzaA",
		"$argon2id$v=19$m=1,t=1,p=1$!!!$aGFzaA",
	} {
		// A hash the server cannot read is an operational problem, not a wrong
		// password: reporting "false, nil" would hide it behind a login failure.
		if _, err := VerifyPassword(bad, "x"); err == nil {
			t.Errorf("VerifyPassword(%q) returned no error", bad)
		}
	}
}

func TestNewTokenIsRandomAndOnlyStoredHashed(t *testing.T) {
	token, hash, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}
	if len(token) < 40 {
		t.Errorf("token = %q, want at least 40 chars of base64", token)
	}
	if len(hash) != 32 {
		t.Errorf("hash length = %d, want 32", len(hash))
	}
	if strings.Contains(string(hash), token) {
		t.Error("stored hash contains the plaintext token")
	}

	other, _, _ := NewToken()
	if token == other {
		t.Error("two tokens are identical")
	}

	if got := HashToken(token); string(got) != string(hash) {
		t.Error("HashToken does not reproduce the stored hash")
	}
	// Whitespace from a copy-paste must not change the identity.
	if got := HashToken("  " + token + "\n"); string(got) != string(hash) {
		t.Error("HashToken is sensitive to surrounding whitespace")
	}
}

func TestBearerToken(t *testing.T) {
	cases := map[string]string{
		"Bearer abc":    "abc",
		"bearer abc":    "abc",
		"BEARER  abc  ": "abc",
		"Basic abc":     "",
		"Bearer":        "",
		"":              "",
		"abc":           "",
	}
	for header, want := range cases {
		if got := BearerToken(header); got != want {
			t.Errorf("BearerToken(%q) = %q, want %q", header, got, want)
		}
	}
}

func TestLimiterAllowsBurstThenRefills(t *testing.T) {
	now := time.Now()
	limiter := NewLimiter(func() time.Time { return now })

	for i := range loginBurst {
		if !limiter.Allow("k") {
			t.Fatalf("attempt %d denied inside the burst", i)
		}
	}
	if limiter.Allow("k") {
		t.Fatal("attempt past the burst was allowed")
	}

	now = now.Add(loginRefill)
	if !limiter.Allow("k") {
		t.Error("attempt after a refill interval was denied")
	}

	// A successful login clears the record so a user who mistyped once is not
	// throttled afterwards.
	limiter.Reset("k")
	for i := range loginBurst {
		if !limiter.Allow("k") {
			t.Fatalf("attempt %d denied after Reset", i)
		}
	}
}

// fakeStore is the slice of store.Store the auth service touches.
type fakeStore struct {
	store.Store
	account  *store.Account
	device   *store.Device
	teams    []string
	upserted []store.Device
	touched  []string
	findErr  error

	registered []store.Device
	tokens     map[string]*store.AuthToken

	otp      *store.OTP
	recovery map[string]bool // hash -> used

	keptSession []byte // ChangePassword's keepSession
}

func (f *fakeStore) ChangePassword(_ context.Context, accountID, oldHash, newHash string, keepSession []byte) error {
	if f.account == nil || f.account.ID != accountID || f.account.PasswordHash != oldHash {
		return store.ErrNotFound
	}
	f.account.PasswordHash = newHash
	f.keptSession = keepSession
	return nil
}

func (f *fakeStore) FindAccountByUsername(_ context.Context, username string) (*store.Account, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	if f.account == nil || f.account.Username != username {
		return nil, store.ErrNotFound
	}
	return f.account, nil
}

func (f *fakeStore) FindAccountByID(_ context.Context, id string) (*store.Account, error) {
	if f.account == nil || f.account.ID != id {
		return nil, store.ErrNotFound
	}
	return f.account, nil
}

func (f *fakeStore) UpsertDevice(_ context.Context, d store.Device) error {
	f.upserted = append(f.upserted, d)
	f.device = &d
	return nil
}

func (f *fakeStore) FindDeviceByTokenHash(_ context.Context, hash []byte) (*store.Device, error) {
	if f.device == nil || string(f.device.TokenHash) != string(hash) {
		return nil, store.ErrNotFound
	}
	return f.device, nil
}

func (f *fakeStore) RegisterDevice(_ context.Context, d store.Device) error {
	if f.device != nil && f.device.ID == d.ID {
		f.device.Name = d.Name
		f.device.RevokedAt = nil
		return nil
	}
	f.registered = append(f.registered, d)
	f.device = &d
	return nil
}

func (f *fakeStore) RegisterDeviceWithPair(ctx context.Context, d store.Device, pair store.TokenPair) error {
	if err := f.RegisterDevice(ctx, d); err != nil {
		return err
	}
	return f.CreateTokenPair(ctx, pair)
}

func (f *fakeStore) FindDeviceByID(_ context.Context, id string) (*store.Device, error) {
	if f.device == nil || f.device.ID != id {
		return nil, store.ErrNotFound
	}
	return f.device, nil
}

func (f *fakeStore) tokenMap() map[string]*store.AuthToken {
	if f.tokens == nil {
		f.tokens = map[string]*store.AuthToken{}
	}
	return f.tokens
}

func (f *fakeStore) CreateTokenPair(_ context.Context, pair store.TokenPair) error {
	if f.device == nil || f.device.ID != pair.Access.DeviceID || f.device.Revoked() {
		return store.ErrNotFound
	}
	for _, token := range []store.AuthToken{pair.Access, pair.Refresh} {
		token := token
		f.tokenMap()[string(token.TokenHash)] = &token
	}
	return nil
}

func (f *fakeStore) FindAuthToken(_ context.Context, hash []byte) (*store.AuthToken, error) {
	token, ok := f.tokenMap()[string(hash)]
	if !ok {
		return nil, store.ErrNotFound
	}
	copied := *token
	return &copied, nil
}

func (f *fakeStore) RotateRefresh(ctx context.Context, oldHash []byte, at time.Time, next store.TokenPair) error {
	token, ok := f.tokenMap()[string(oldHash)]
	if !ok || token.Kind != store.TokenRefresh || token.RevokedAt != nil || !at.Before(token.ExpiresAt) {
		return store.ErrNotFound
	}
	if token.UsedAt != nil {
		return store.ErrTokenUsed
	}
	token.UsedAt = &at
	return f.CreateTokenPair(ctx, next)
}

func (f *fakeStore) RevokeDeviceTokens(_ context.Context, deviceID string, at time.Time) error {
	for _, token := range f.tokenMap() {
		if token.DeviceID == deviceID && token.RevokedAt == nil {
			token.RevokedAt = &at
		}
	}
	return nil
}

func (f *fakeStore) RevokeTokenFamily(_ context.Context, family string, at time.Time) error {
	for _, token := range f.tokenMap() {
		if token.FamilyID == family && token.RevokedAt == nil {
			token.RevokedAt = &at
		}
	}
	return nil
}

func (f *fakeStore) FindOTP(_ context.Context, _ string) (*store.OTP, error) {
	if f.otp == nil {
		return nil, store.ErrNotFound
	}
	copied := *f.otp
	return &copied, nil
}

func (f *fakeStore) OTPStatus(_ context.Context, _ string) (string, error) {
	switch {
	case f.otp == nil:
		return "off", nil
	case f.otp.Enabled():
		return "enabled", nil
	default:
		return "pending", nil
	}
}

func (f *fakeStore) StartOTP(_ context.Context, accountID, secret string, at time.Time) error {
	if f.otp != nil && f.otp.Enabled() {
		return store.ErrDuplicate
	}
	f.otp = &store.OTP{AccountID: accountID, Secret: secret, CreatedAt: at}
	return nil
}

func (f *fakeStore) EnableOTP(_ context.Context, _, secret string, step int64, hashes [][]byte, at time.Time) error {
	if f.otp == nil || f.otp.Enabled() || f.otp.Secret != secret {
		return store.ErrNotFound
	}
	f.otp.EnabledAt = &at
	f.otp.LastStep = step
	f.recovery = map[string]bool{}
	for _, h := range hashes {
		f.recovery[string(h)] = false
	}
	return nil
}

func (f *fakeStore) AdvanceOTPStep(_ context.Context, _ string, step int64) (bool, error) {
	if f.otp == nil || step <= f.otp.LastStep {
		return false, nil
	}
	f.otp.LastStep = step
	return true, nil
}

func (f *fakeStore) UseRecoveryCode(_ context.Context, _ string, hash []byte, _ time.Time) (bool, error) {
	used, ok := f.recovery[string(hash)]
	if !ok || used {
		return false, nil
	}
	f.recovery[string(hash)] = true
	return true, nil
}

func (f *fakeStore) DeleteOTP(context.Context, string) error {
	f.otp = nil
	f.recovery = nil
	return nil
}

func (f *fakeStore) TouchDevice(_ context.Context, deviceID string, _ time.Time) error {
	f.touched = append(f.touched, deviceID)
	return nil
}

func (f *fakeStore) TeamsOf(_ context.Context, _ string) ([]string, error) {
	return f.teams, nil
}

func newFixture(t *testing.T, password string) (*Service, *fakeStore, time.Time) {
	t.Helper()
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	st := &fakeStore{
		account: &store.Account{ID: "acct-1", Username: "alice", PasswordHash: hash, CreatedAt: now},
		teams:   []string{store.DefaultTeamID},
	}
	return NewService(st, func() time.Time { return now }), st, now
}

func validLogin() LoginRequest {
	return LoginRequest{
		Username: "alice", Password: "hunter2",
		ClientID: "client-1", DeviceName: "laptop", RemoteIP: "192.0.2.1",
	}
}

func TestLoginIssuesTokenAndRegistersDevice(t *testing.T) {
	svc, st, _ := newFixture(t, "hunter2")

	result, err := svc.Login(context.Background(), validLogin())
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	if result.Token == "" {
		t.Error("no token returned")
	}
	if result.Identity.AccountID != "acct-1" || result.Identity.DeviceID != "client-1" {
		t.Errorf("identity = %+v", result.Identity)
	}
	if len(st.upserted) != 1 {
		t.Fatalf("devices upserted = %d, want 1", len(st.upserted))
	}

	// The device row keeps the hash, never the token.
	stored := st.upserted[0]
	if string(stored.TokenHash) != string(HashToken(result.Token)) {
		t.Error("stored hash does not match the issued token")
	}
	if strings.Contains(string(stored.TokenHash), result.Token) {
		t.Error("device row contains the plaintext token")
	}
}

func TestLoginRejectsWrongPasswordAndUnknownUser(t *testing.T) {
	svc, _, _ := newFixture(t, "hunter2")
	ctx := context.Background()

	wrong := validLogin()
	wrong.Password = "nope"
	if _, err := svc.Login(ctx, wrong); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("wrong password error = %v, want ErrInvalidCredentials", err)
	}

	unknown := validLogin()
	unknown.Username = "bob"
	// Same error as above on purpose: the client must not learn which
	// usernames exist.
	if _, err := svc.Login(ctx, unknown); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("unknown user error = %v, want ErrInvalidCredentials", err)
	}
}

func TestLoginRequiresClientID(t *testing.T) {
	svc, _, _ := newFixture(t, "hunter2")

	req := validLogin()
	req.ClientID = ""
	if _, err := svc.Login(context.Background(), req); !errors.Is(err, ErrMissingClientID) {
		t.Errorf("error = %v, want ErrMissingClientID", err)
	}
}

func TestLoginRefusesDisabledAccount(t *testing.T) {
	svc, st, now := newFixture(t, "hunter2")
	disabled := now
	st.account.DisabledAt = &disabled

	if _, err := svc.Login(context.Background(), validLogin()); !errors.Is(err, ErrAccountDisabled) {
		t.Errorf("error = %v, want ErrAccountDisabled", err)
	}
}

func TestLoginSurfacesUnreadableHash(t *testing.T) {
	svc, st, _ := newFixture(t, "hunter2")
	st.account.PasswordHash = "not-a-hash"

	_, err := svc.Login(context.Background(), validLogin())
	if err == nil || errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("error = %v, want an operational error rather than a credential failure", err)
	}
}

func TestLoginIsRateLimitedPerIPAndUsername(t *testing.T) {
	svc, _, _ := newFixture(t, "hunter2")
	ctx := context.Background()

	wrong := validLogin()
	wrong.Password = "nope"
	for range loginBurst {
		if _, err := svc.Login(ctx, wrong); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("unexpected error inside the burst: %v", err)
		}
	}
	if _, err := svc.Login(ctx, wrong); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("error = %v, want ErrRateLimited", err)
	}

	// Another username from the same address still gets its own budget.
	other := wrong
	other.Username = "carol"
	if _, err := svc.Login(ctx, other); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("error for a different username = %v, want ErrInvalidCredentials", err)
	}
}

func TestAuthenticateResolvesIdentity(t *testing.T) {
	svc, st, _ := newFixture(t, "hunter2")
	ctx := context.Background()

	result, err := svc.Login(ctx, validLogin())
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	identity, err := svc.Authenticate(ctx, result.Token)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if identity.AccountID != "acct-1" || identity.DeviceID != "client-1" {
		t.Errorf("identity = %+v", identity)
	}
	if len(identity.TeamIDs) != 1 || identity.TeamIDs[0] != store.DefaultTeamID {
		t.Errorf("teams = %v", identity.TeamIDs)
	}

	// lastSeenAt was just set by the login, so no second write is needed.
	if len(st.touched) != 0 {
		t.Errorf("TouchDevice called %d times for a fresh device, want 0", len(st.touched))
	}
}

func TestAuthenticateRejectsBadAndRevokedTokens(t *testing.T) {
	svc, st, now := newFixture(t, "hunter2")
	ctx := context.Background()

	if _, err := svc.Authenticate(ctx, "nonsense"); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("unknown token error = %v, want ErrInvalidCredentials", err)
	}
	if _, err := svc.Authenticate(ctx, ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("empty token error = %v, want ErrInvalidCredentials", err)
	}

	result, err := svc.Login(ctx, validLogin())
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	revoked := now
	st.device.RevokedAt = &revoked
	if _, err := svc.Authenticate(ctx, result.Token); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("revoked device error = %v, want ErrInvalidCredentials", err)
	}
}

func TestAuthenticateRefusesDisabledAccount(t *testing.T) {
	svc, st, now := newFixture(t, "hunter2")
	ctx := context.Background()

	result, err := svc.Login(ctx, validLogin())
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	disabled := now
	st.account.DisabledAt = &disabled
	if _, err := svc.Authenticate(ctx, result.Token); !errors.Is(err, ErrAccountDisabled) {
		t.Errorf("error = %v, want ErrAccountDisabled", err)
	}
}

func TestAuthenticateThrottlesLastSeenWrites(t *testing.T) {
	hash, err := HashPassword("hunter2")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	stale := now.Add(-2 * lastSeenInterval)

	token, tokenHash, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}

	st := &fakeStore{
		account: &store.Account{ID: "acct-1", Username: "alice", PasswordHash: hash},
		device:  &store.Device{ID: "client-1", AccountID: "acct-1", TokenHash: tokenHash, LastSeenAt: &stale},
		teams:   []string{store.DefaultTeamID},
	}
	svc := NewService(st, func() time.Time { return now })

	if _, err := svc.Authenticate(context.Background(), token); err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	// Writing lastSeenAt on every request would push each sync through the
	// single write connection; it is only refreshed once it is stale.
	if len(st.touched) != 1 {
		t.Errorf("TouchDevice called %d times for a stale device, want 1", len(st.touched))
	}
}

func TestLimiterKeysAreIndependent(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	limiter := NewLimiter(func() time.Time { return now })
	for i := 0; i < loginBurst; i++ {
		limiter.Allow("ip|alice")
	}
	if limiter.Allow("ip|alice") {
		t.Fatal("alice should be limited")
	}
	if !limiter.Allow("ip|bob") {
		t.Fatal("bob must not share alice's bucket")
	}
}

func TestLimiterEvictsIdleKeysWhenFull(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	limiter := NewLimiter(func() time.Time { return now })
	for i := 0; i < limiterMaxKeys; i++ {
		limiter.Allow(fmt.Sprintf("k%d", i))
	}
	now = now.Add(limiterEvictIdle + time.Second)
	limiter.Allow("fresh")
	if got := limiter.size(); got != 1 {
		t.Fatalf("idle keys not evicted: %d tracked", got)
	}
}

func TestDeviceTokenTTL(t *testing.T) {
	ctx := context.Background()

	// Default policy: no expiry, as in 0.1.0.
	svc, st, now := newFixture(t, "hunter2")
	result, err := svc.Login(ctx, validLogin())
	if err != nil {
		t.Fatal(err)
	}
	if result.ExpiresAt != nil || st.device.TokenExpiresAt != nil {
		t.Fatalf("default policy set an expiry: %v", result.ExpiresAt)
	}

	// With a TTL, a sign-in token expires and is then refused.
	clock := now
	svc = NewService(st, func() time.Time { return clock }).WithPolicy(Policy{DeviceTokenTTL: 2 * time.Second})
	result, err = svc.Login(ctx, validLogin())
	if err != nil {
		t.Fatal(err)
	}
	if result.ExpiresAt == nil || !result.ExpiresAt.Equal(now.Add(2*time.Second)) {
		t.Fatalf("ExpiresAt = %v", result.ExpiresAt)
	}
	if _, err := svc.Authenticate(ctx, result.Token); err != nil {
		t.Fatalf("fresh token refused: %v", err)
	}
	clock = now.Add(3 * time.Second)
	if _, err := svc.Authenticate(ctx, result.Token); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expired token: %v, want ErrInvalidCredentials", err)
	}
}

func TestIssuePairKeepsTheDeviceToken(t *testing.T) {
	svc, st, now := newFixture(t, "hunter2")
	ctx := context.Background()

	login, err := svc.Login(ctx, validLogin())
	if err != nil {
		t.Fatal(err)
	}
	pair, err := svc.IssuePair(ctx, validLogin())
	if err != nil {
		t.Fatalf("IssuePair: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" || pair.AccessToken == pair.RefreshToken {
		t.Fatalf("pair = %+v", pair)
	}
	if !pair.AccessExpiresAt.Equal(now.Add(DefaultAccessTokenTTL)) ||
		!pair.RefreshExpiresAt.Equal(now.Add(DefaultRefreshTokenTTL)) {
		t.Fatalf("expiries = %v / %v", pair.AccessExpiresAt, pair.RefreshExpiresAt)
	}
	if pair.Identity.AccountID != "acct-1" || pair.Identity.DeviceID != "client-1" {
		t.Fatalf("identity = %+v", pair.Identity)
	}

	// Both the 0.1 device token and the new access token authenticate.
	for name, token := range map[string]string{"device": login.Token, "access": pair.AccessToken} {
		if _, err := svc.Authenticate(ctx, token); err != nil {
			t.Errorf("%s token refused: %v", name, err)
		}
	}
	if string(st.device.TokenHash) != string(HashToken(login.Token)) {
		t.Error("IssuePair replaced the device token")
	}

	// A refresh token is not a sync credential.
	if _, err := svc.Authenticate(ctx, pair.RefreshToken); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("refresh token as bearer: %v", err)
	}

	bad := validLogin()
	bad.Password = "nope"
	if _, err := svc.IssuePair(ctx, bad); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("wrong password: %v", err)
	}
	missing := validLogin()
	missing.ClientID = ""
	if _, err := svc.IssuePair(ctx, missing); !errors.Is(err, ErrMissingClientID) {
		t.Errorf("missing clientId: %v", err)
	}
}

func TestRefreshRotatesAndDetectsReuse(t *testing.T) {
	svc, _, now := newFixture(t, "hunter2")
	ctx := context.Background()

	first, err := svc.IssuePair(ctx, validLogin())
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.Refresh(ctx, first.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if second.AccessToken == first.AccessToken || second.RefreshToken == first.RefreshToken {
		t.Fatal("refresh returned the same tokens")
	}
	if _, err := svc.Authenticate(ctx, second.AccessToken); err != nil {
		t.Fatalf("rotated access token refused: %v", err)
	}
	// The refresh lifetime restarts at rotation.
	if !second.RefreshExpiresAt.Equal(now.Add(DefaultRefreshTokenTTL)) {
		t.Errorf("refresh expiry = %v", second.RefreshExpiresAt)
	}

	// Replaying the spent token revokes the whole family.
	if _, err := svc.Refresh(ctx, first.RefreshToken); !errors.Is(err, ErrRefreshReused) {
		t.Fatalf("replay: %v, want ErrRefreshReused", err)
	}
	if _, err := svc.Authenticate(ctx, second.AccessToken); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("access token survived the reuse: %v", err)
	}
	if _, err := svc.Refresh(ctx, second.RefreshToken); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("refresh token survived the reuse: %v", err)
	}

	// An access token cannot be used as a refresh token.
	third, _ := svc.IssuePair(ctx, validLogin())
	if _, err := svc.Refresh(ctx, third.AccessToken); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("access as refresh: %v", err)
	}
}

func TestAccessTokenExpiresAndFollowsTheDevice(t *testing.T) {
	_, st, now := newFixture(t, "hunter2")
	clock := now
	svc := NewService(st, func() time.Time { return clock }).WithPolicy(Policy{
		AccessTokenTTL: time.Minute, RefreshTokenTTL: time.Hour,
	})
	ctx := context.Background()

	pair, err := svc.IssuePair(ctx, validLogin())
	if err != nil {
		t.Fatal(err)
	}
	clock = now.Add(2 * time.Minute)
	if _, err := svc.Authenticate(ctx, pair.AccessToken); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expired access token: %v", err)
	}
	next, err := svc.Refresh(ctx, pair.RefreshToken)
	if err != nil {
		t.Fatalf("refresh after access expiry: %v", err)
	}

	revoked := clock
	st.device.RevokedAt = &revoked
	if _, err := svc.Authenticate(ctx, next.AccessToken); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("access token of a revoked device: %v", err)
	}
	if _, err := svc.Refresh(ctx, next.RefreshToken); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("refresh of a revoked device: %v", err)
	}

	st.device.RevokedAt = nil
	disabled := clock
	st.account.DisabledAt = &disabled
	if _, err := svc.Authenticate(ctx, next.AccessToken); !errors.Is(err, ErrAccountDisabled) {
		t.Errorf("access token of a disabled account: %v", err)
	}

	clock = now.Add(2 * time.Hour)
	st.account.DisabledAt = nil
	if _, err := svc.Refresh(ctx, next.RefreshToken); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expired refresh token: %v", err)
	}
}

func TestRefreshReuseRevokesEveryPairOfTheDevice(t *testing.T) {
	svc, _, _ := newFixture(t, "hunter2")
	ctx := context.Background()

	device, err := svc.Login(ctx, validLogin())
	if err != nil {
		t.Fatal(err)
	}
	first, _ := svc.IssuePair(ctx, validLogin())
	second, _ := svc.IssuePair(ctx, validLogin()) // another family, same device
	if _, err := svc.Refresh(ctx, first.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Refresh(ctx, first.RefreshToken); !errors.Is(err, ErrRefreshReused) {
		t.Fatalf("replay: %v", err)
	}
	if _, err := svc.Authenticate(ctx, second.AccessToken); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("another family's access token survived the reuse: %v", err)
	}
	if _, err := svc.Refresh(ctx, second.RefreshToken); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("another family's refresh token survived the reuse: %v", err)
	}
	// The device token is a different credential and stays.
	if _, err := svc.Authenticate(ctx, device.Token); err != nil {
		t.Errorf("device token cut off by a refresh reuse: %v", err)
	}
}
