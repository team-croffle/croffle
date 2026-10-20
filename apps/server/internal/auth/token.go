package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
)

// tokenBytes is the entropy of a device token. 32 random bytes is far past
// guessable and keeps the token short enough to paste.
const tokenBytes = 32

// NewToken returns a fresh token and the hash to store.
//
// The plaintext exists exactly once, in the response that issues it: the
// database keeps only the hash, so a database copy cannot be turned into a
// working credential.
func NewToken() (token string, hash []byte, err error) {
	raw := make([]byte, tokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, fmt.Errorf("auth: read token bytes: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	return token, HashToken(token), nil
}

// HashToken maps a token to its stored form.
//
// A plain SHA-256 is right here, unlike for passwords: the token is 32 random
// bytes, so there is no low-entropy input to protect with a slow KDF, and
// hashing happens on every single request.
func HashToken(token string) []byte {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return sum[:]
}

// BearerToken extracts the token from an Authorization header value. It returns
// "" when the header is absent or not a bearer token.
func BearerToken(header string) string {
	const prefix = "bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}
