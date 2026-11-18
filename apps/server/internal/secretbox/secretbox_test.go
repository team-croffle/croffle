package secretbox

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"errors"
	"regexp"
	"strings"
	"testing"
)

func testKey(t *testing.T, fill byte) []byte {
	t.Helper()
	key := make([]byte, KeySize)
	for i := range key {
		key[i] = fill
	}
	return key
}

func TestSecretBoxSealOpenRoundTrips(t *testing.T) {
	box, err := New(testKey(t, 1))
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := box.Seal("acct-1", exampleSecret)
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^v1:[0-9a-f]{8}:[A-Za-z0-9+/]+=*$`).MatchString(sealed) {
		t.Fatalf("sealed value %q has the wrong shape", sealed)
	}
	if strings.Contains(sealed, exampleSecret) {
		t.Fatal("sealed value contains the plain secret")
	}
	if id, _ := KeyIDOf(sealed); id != box.KeyID() {
		t.Fatalf("key id %q, want %q", id, box.KeyID())
	}
	plain, err := box.Open("acct-1", sealed)
	if err != nil || plain != exampleSecret {
		t.Fatalf("Open = %q, %v", plain, err)
	}
}

func TestSecretBoxSealUsesAFreshNonce(t *testing.T) {
	box, _ := New(testKey(t, 1))
	a, _ := box.Seal("acct-1", "SAME")
	b, _ := box.Seal("acct-1", "SAME")
	if a == b {
		t.Fatal("two seals of the same secret are identical: the nonce is not random")
	}
}

func TestSecretBoxOpenRefusesAnotherAccount(t *testing.T) {
	box, _ := New(testKey(t, 1))
	sealed, _ := box.Seal("acct-1", exampleSecret)
	if _, err := box.Open("acct-2", sealed); !errors.Is(err, ErrDamaged) {
		t.Fatalf("Open with another account id: %v, want ErrDamaged", err)
	}
}

func TestSecretBoxOpenRefusesTampering(t *testing.T) {
	box, _ := New(testKey(t, 1))
	sealed, _ := box.Seal("acct-1", exampleSecret)
	raw, _ := base64.StdEncoding.DecodeString(sealed[len("v1:")+9:])
	raw[len(raw)-1] ^= 1
	tampered := sealed[:len("v1:")+9] + base64.StdEncoding.EncodeToString(raw)
	if _, err := box.Open("acct-1", tampered); !errors.Is(err, ErrDamaged) {
		t.Fatalf("Open of a tampered value: %v, want ErrDamaged", err)
	}
	if _, err := box.Open("acct-1", "v1:"+box.KeyID()+":not base64!"); !errors.Is(err, ErrDamaged) {
		t.Fatalf("Open of garbage: %v, want ErrDamaged", err)
	}
}

func TestSecretBoxMismatchedKeyIsNamed(t *testing.T) {
	one, _ := New(testKey(t, 1))
	two, _ := New(testKey(t, 2))
	sealed, _ := one.Seal("acct-1", exampleSecret)
	_, err := two.Open("acct-1", sealed)
	if !errors.Is(err, ErrWrongKey) || !strings.Contains(err.Error(), one.KeyID()) {
		t.Fatalf("Open with another key: %v, want ErrWrongKey naming %s", err, one.KeyID())
	}
}

func TestSecretBoxOpensPlainValues(t *testing.T) {
	box, _ := New(testKey(t, 1))
	if plain, err := box.Open("acct-1", exampleSecret); err != nil || plain != exampleSecret {
		t.Fatalf("Open(plain) = %q, %v", plain, err)
	}
}

func TestSecretBoxMissingKeyRefusesSealedValues(t *testing.T) {
	box, _ := New(testKey(t, 1))
	sealed, _ := box.Seal("acct-1", exampleSecret)
	if _, err := Plain.Open("acct-1", sealed); !errors.Is(err, ErrNoKey) {
		t.Fatalf("Plain.Open(sealed): %v, want ErrNoKey", err)
	}
	if stored, _ := Plain.Seal("acct-1", exampleSecret); stored != exampleSecret {
		t.Fatalf("Plain.Seal changed the value: %q", stored)
	}
	if Plain.KeyID() != "" {
		t.Fatal("Plain has a key id")
	}
}

func TestSecretKeyParse(t *testing.T) {
	key := testKey(t, 7)
	std := base64.StdEncoding.EncodeToString(key)
	for _, raw := range []string{std, "  " + std + "\n", strings.TrimRight(std, "=")} {
		got, err := ParseKey(raw)
		if err != nil || string(got) != string(key) {
			t.Errorf("ParseKey(%q) = %v, %v", raw, got, err)
		}
	}
}

func TestSecretKeyParseError(t *testing.T) {
	short := base64.StdEncoding.EncodeToString(make([]byte, 16))
	for _, raw := range []string{"", "   ", "not base64 !", short} {
		if _, err := ParseKey(raw); err == nil {
			t.Errorf("ParseKey(%q) succeeded", raw)
		}
	}
}

func TestSecretGenerateKey(t *testing.T) {
	a, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := GenerateKey()
	if a == b {
		t.Fatal("two generated keys are equal")
	}
	if key, err := ParseKey(a); err != nil || len(key) != KeySize {
		t.Fatalf("generated key does not parse: %v", err)
	}
}

// exampleSecret is a base32 TOTP secret made when the tests start: no
// secret-shaped literal sits in the tree (knowledge: no credential literals
// in tests).
var exampleSecret = func() string {
	buf := make([]byte, 10)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf)
}()
