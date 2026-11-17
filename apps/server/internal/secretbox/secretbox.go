// Package secretbox seals the small secrets the server has to read back, such
// as TOTP secrets, with a key the operator manages.
//
// A sealed value is "v1:<keyId>:<base64(nonce || ciphertext)>". The cipher is
// AES-256-GCM with a random 12-byte nonce, and the account id is the
// additional data, so a sealed secret copied onto another account's row does
// not open. keyId is the first 8 hex digits of SHA-256(key): it lets the
// server tell "sealed with another key" apart from "damaged" and name the key
// in the error.
//
// Plain values are what servers without a key store (0.1.1 and 0.2.0 without
// CROFFLE_SECRET_KEY). TOTP secrets are base32, which has no ':', so a stored
// value is sealed exactly when it starts with the version prefix.
package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// KeySize is the length of a key in bytes (AES-256).
const KeySize = 32

const prefix = "v1:"

var (
	// ErrNoKey means a sealed value was found but the box has no key.
	ErrNoKey = errors.New("secretbox: the value is sealed but no secret key is set")
	// ErrWrongKey means the value was sealed with a key whose id differs.
	ErrWrongKey = errors.New("secretbox: the value was sealed with another key")
	// ErrDamaged means the value claims the current key but does not open:
	// tampered with, moved to another account, or truncated.
	ErrDamaged = errors.New("secretbox: the sealed value does not open")
)

// Box seals and opens secrets bound to one account.
type Box interface {
	// Seal returns the value to store for plain.
	Seal(accountID, plain string) (string, error)
	// Open returns the plain secret for a stored value, sealed or not.
	Open(accountID, stored string) (string, error)
	// KeyID is the id of the key, or "" for a box without one.
	KeyID() string
}

// IsSealed reports whether a stored value is sealed.
func IsSealed(stored string) bool { return strings.HasPrefix(stored, prefix) }

// KeyIDOf returns the key id of a sealed value.
func KeyIDOf(stored string) (string, bool) {
	if !IsSealed(stored) {
		return "", false
	}
	rest := stored[len(prefix):]
	id, _, ok := strings.Cut(rest, ":")
	return id, ok && id != ""
}

// ParseKey decodes a key from its configured form: standard base64, padding
// optional, surrounding whitespace ignored. It has to be exactly KeySize bytes.
func ParseKey(raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("the secret key is empty")
	}
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		key, err = base64.RawStdEncoding.DecodeString(strings.TrimRight(raw, "="))
	}
	if err != nil {
		return nil, errors.New("the secret key is not base64 (make one with `croffle-server secret generate`)")
	}
	if len(key) != KeySize {
		return nil, fmt.Errorf("the secret key is %d bytes; it has to be %d (make one with `croffle-server secret generate`)", len(key), KeySize)
	}
	return key, nil
}

// GenerateKey returns a new random key in its configured (base64) form.
func GenerateKey() (string, error) {
	key := make([]byte, KeySize)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("secretbox: read random key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// KeyIDFor returns the id of a key.
func KeyIDFor(key []byte) string {
	sum := sha256.Sum256(key)
	return hex.EncodeToString(sum[:])[:8]
}

// Plain is the box of a server without a key: it stores secrets as they are
// and refuses to pretend it can open a sealed one.
var Plain Box = plainBox{}

type plainBox struct{}

func (plainBox) Seal(_, plain string) (string, error) { return plain, nil }

func (plainBox) Open(_, stored string) (string, error) {
	if IsSealed(stored) {
		return "", ErrNoKey
	}
	return stored, nil
}

func (plainBox) KeyID() string { return "" }

// AEAD is a box with a key.
type AEAD struct {
	gcm cipher.AEAD
	id  string
}

// New returns a box for key, which has to be KeySize bytes.
func New(key []byte) (*AEAD, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("secretbox: key is %d bytes, want %d", len(key), KeySize)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("secretbox: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("secretbox: %w", err)
	}
	return &AEAD{gcm: gcm, id: KeyIDFor(key)}, nil
}

// KeyID returns the id of the box's key.
func (b *AEAD) KeyID() string { return b.id }

// Seal encrypts plain for accountID.
func (b *AEAD) Seal(accountID, plain string) (string, error) {
	nonce := make([]byte, b.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("secretbox: read nonce: %w", err)
	}
	sealed := b.gcm.Seal(nonce, nonce, []byte(plain), []byte(accountID))
	return prefix + b.id + ":" + base64.StdEncoding.EncodeToString(sealed), nil
}

// Open decrypts a sealed value, or returns a plain one as it is: a server that
// has just been given a key still reads the rows it has not converted yet.
func (b *AEAD) Open(accountID, stored string) (string, error) {
	if !IsSealed(stored) {
		return stored, nil
	}
	id, ok := KeyIDOf(stored)
	if !ok {
		return "", ErrDamaged
	}
	if id != b.id {
		return "", fmt.Errorf("%w (id %s, the current key is %s)", ErrWrongKey, id, b.id)
	}
	raw, err := base64.StdEncoding.DecodeString(stored[len(prefix)+len(id)+1:])
	if err != nil || len(raw) < b.gcm.NonceSize() {
		return "", ErrDamaged
	}
	nonce, ciphertext := raw[:b.gcm.NonceSize()], raw[b.gcm.NonceSize():]
	plain, err := b.gcm.Open(nil, nonce, ciphertext, []byte(accountID))
	if err != nil {
		return "", ErrDamaged
	}
	return string(plain), nil
}
