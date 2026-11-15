package protocol

// LoginRequest is POST /auth/login.
//
// The desktop offers two sign-in screens, "username and password" and "paste a
// token", but there is one mechanism underneath: this endpoint hands out the
// token that every later request carries.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	// OTP is reserved: the desktop sends it after a 401 with code
	// "otp_required". Server 0.1 has no way to enrol a secret, so nothing
	// requires it yet.
	OTP string `json:"otp,omitempty"`
	// ClientID is the desktop's own clientId. It becomes the device id, which
	// is the same value that appears in a row's lastWriterId.
	ClientID   string `json:"clientId"`
	DeviceName string `json:"deviceName,omitempty"`
}

// LoginResponse carries the only plaintext copy of the token.
type LoginResponse struct {
	Token     string `json:"token"`
	AccountID string `json:"accountId"`
	// TeamID is the team this account shares into. Server 0.1 has exactly one.
	TeamID *string `json:"teamId,omitempty"`
	// ExpiresAt is unix milliseconds, or absent for a token that does not
	// expire. It is absent unless the operator set CROFFLE_DEVICE_TOKEN_TTL
	// (server 0.1.1): desktop 1.3.0 does not store the password, so it could
	// not renew an expiring token silently. POST /auth/token is the renewable
	// alternative.
	ExpiresAt *int64 `json:"expiresAt,omitempty"`
}

// WhoAmIResponse is POST /auth/whoami. The desktop calls it once before
// storing a pasted token, so a typo fails at paste time instead of looking
// like a working connection that never syncs.
type WhoAmIResponse struct {
	AccountID string  `json:"accountId"`
	TeamID    *string `json:"teamId,omitempty"`
}
