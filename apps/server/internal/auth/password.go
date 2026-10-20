// Package auth turns credentials into a device token and back.
//
// There is exactly one credential the rest of the server understands: a bearer
// token bound to a device. A username and password is only a way to obtain one,
// which is why the desktop can offer "sign in" and "paste a token" as two
// screens over one mechanism.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2id parameters. They are written into every hash, so raising them later
// does not invalidate existing passwords: an old hash is still verified with
// the parameters it was created with.
//
// The values follow the OWASP minimum for argon2id (19 MiB, 2 iterations, 1
// lane), which a small self-hosted box can afford on an occasional login.
const (
	argonMemoryKiB = 19 * 1024
	argonTime      = 2
	argonThreads   = 1
	argonKeyLen    = 32
	argonSaltLen   = 16
)

// ErrMalformedHash means the stored hash is not something this package wrote.
var ErrMalformedHash = errors.New("auth: malformed password hash")

// HashPassword returns an encoded argon2id hash:
//
//	$argon2id$v=19$m=19456,t=2,p=1$<salt>$<hash>
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("auth: read salt: %w", err)
	}
	return encodeHash(password, salt, argonParams{
		memory:  argonMemoryKiB,
		time:    argonTime,
		threads: argonThreads,
		keyLen:  argonKeyLen,
	}), nil
}

// VerifyPassword reports whether password matches the encoded hash.
//
// A malformed hash is an error, not a mismatch: it means the database holds
// something this server cannot interpret, and treating that as "wrong
// password" would hide the problem behind a login failure.
func VerifyPassword(encoded, password string) (bool, error) {
	params, salt, want, err := decodeHash(encoded)
	if err != nil {
		return false, err
	}
	got := argon2.IDKey([]byte(password), salt, params.time, params.memory, params.threads, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

// dummyHash is verified against when the username does not exist, so a missing
// account and a wrong password cost the same time and cannot be told apart.
var dummyHash = func() string {
	h, err := HashPassword("not-a-real-password")
	if err != nil {
		// rand failing at init is unrecoverable and must not be papered over.
		panic("auth: cannot build dummy hash: " + err.Error())
	}
	return h
}()

// BurnPasswordTime runs the same work VerifyPassword would, and discards the
// result. Call it on the "no such user" path.
func BurnPasswordTime(password string) {
	_, _ = VerifyPassword(dummyHash, password)
}

type argonParams struct {
	memory  uint32
	time    uint32
	threads uint8
	keyLen  uint32
}

func encodeHash(password string, salt []byte, p argonParams) string {
	sum := argon2.IDKey([]byte(password), salt, p.time, p.memory, p.threads, p.keyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.memory, p.time, p.threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(sum))
}

func decodeHash(encoded string) (argonParams, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	// ["", "argon2id", "v=19", "m=...,t=...,p=...", salt, hash]
	if len(parts) != 6 || parts[0] != "" {
		return argonParams{}, nil, nil, ErrMalformedHash
	}
	if parts[1] != "argon2id" {
		return argonParams{}, nil, nil, fmt.Errorf("%w: algorithm %q", ErrMalformedHash, parts[1])
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return argonParams{}, nil, nil, ErrMalformedHash
	}
	if version != argon2.Version {
		return argonParams{}, nil, nil, fmt.Errorf("%w: argon2 version %d", ErrMalformedHash, version)
	}

	var p argonParams
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.memory, &p.time, &p.threads); err != nil {
		return argonParams{}, nil, nil, ErrMalformedHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return argonParams{}, nil, nil, ErrMalformedHash
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return argonParams{}, nil, nil, ErrMalformedHash
	}
	if len(salt) == 0 || len(want) == 0 {
		return argonParams{}, nil, nil, ErrMalformedHash
	}

	p.keyLen = uint32(len(want))
	return p, salt, want, nil
}
