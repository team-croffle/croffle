package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/team-croffle/croffle/apps/server/internal/secretbox"
	"github.com/team-croffle/croffle/apps/server/internal/store/sqlite"
)

func TestSecretGenerate(t *testing.T) {
	r := newRunner(t)
	a := strings.TrimSpace(r.mustRun("", "secret", "generate").out)
	b := strings.TrimSpace(r.mustRun("", "secret", "generate").out)
	if a == b {
		t.Fatal("two generated keys are equal")
	}
	for _, key := range []string{a, b} {
		if raw, err := secretbox.ParseKey(key); err != nil || len(raw) != 32 {
			t.Fatalf("generated %q: %v", key, err)
		}
	}
	if _, err := os.Stat(r.path); !os.IsNotExist(err) {
		t.Fatal("secret generate created the database")
	}
}

// seedSecrets enrols alice and bob through the store, the way the dashboard
// would, with whatever box the server had at the time.
func seedSecrets(t *testing.T, r *runner, box secretbox.Box) {
	t.Helper()
	r.mustRun("pw-a\n", "user", "create", "alice", "--password-stdin")
	r.mustRun("pw-b\n", "user", "create", "bob", "--password-stdin")
	db, err := sqlite.Open(context.Background(), r.path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.PrepareSecrets(context.Background(), box); err != nil {
		t.Fatal(err)
	}
	accounts, err := db.ListAccounts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range accounts {
		if err := db.StartOTP(context.Background(), a.ID, "SECRET"+a.Username, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
}

func keyFile(t *testing.T, r *runner) (path, id string) {
	t.Helper()
	key := strings.TrimSpace(r.mustRun("", "secret", "generate").out)
	path = filepath.Join(t.TempDir(), "key")
	if err := os.WriteFile(path, []byte(key+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	raw, _ := secretbox.ParseKey(key)
	return path, secretbox.KeyIDFor(raw)
}

func TestSecretRotate(t *testing.T) {
	r := newRunner(t)
	seedSecrets(t, r, secretbox.Plain)
	file1, id1 := keyFile(t, r)
	file2, id2 := keyFile(t, r)

	// No current key: the plain secrets are sealed for the first time.
	res := r.mustRun("", "secret", "rotate", "--new-key-file", file1)
	if !strings.Contains(res.out, "sealed 2 one-time code secret(s) with key "+id1) {
		t.Fatalf("first rotate output = %q", res.out)
	}
	if !strings.Contains(res.err, "Stop the server before rotating") {
		t.Fatalf("first rotate printed no warning: %q", res.err)
	}

	// Key 1 -> key 2, the new key on stdin.
	r.env = map[string]string{"CROFFLE_SECRET_KEY_FILE": file1}
	raw2, _ := os.ReadFile(file2)
	res = r.mustRun(string(raw2), "secret", "rotate", "--new-key-stdin")
	if !strings.Contains(res.out, "with key "+id2) {
		t.Fatalf("second rotate output = %q", res.out)
	}

	// The server starts with key 2 and reads the secrets.
	db, err := sqlite.Open(context.Background(), r.path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	box2, _ := secretbox.New(mustKey(t, file2))
	if _, err := db.PrepareSecrets(context.Background(), box2); err != nil {
		t.Fatalf("start with key 2: %v", err)
	}
	info, _ := db.Info(context.Background())
	if info.OTPSecretsEncrypted != 2 || info.OTPSecretsPlain != 0 {
		t.Fatalf("after rotate: %+v", info)
	}
}

func mustKey(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	key, err := secretbox.ParseKey(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func TestSecretRotateRefuses(t *testing.T) {
	r := newRunner(t)
	seedSecrets(t, r, secretbox.Plain)
	file1, _ := keyFile(t, r)

	for name, args := range map[string][]string{
		"no new key": {"secret", "rotate"},
		"both flags": {"secret", "rotate", "--new-key-file", file1, "--new-key-stdin"},
	} {
		if _, err := r.run("", args...); err == nil {
			t.Errorf("%s: rotate succeeded", name)
		}
	}
	r.env = map[string]string{"CROFFLE_SECRET_KEY_FILE": file1}
	r.mustRun("", "db", "info") // the key is valid
	if _, err := r.run("", "secret", "rotate", "--new-key-file", file1); err == nil ||
		!strings.Contains(err.Error(), "is the current key") {
		t.Fatalf("rotate to the same key: %v", err)
	}
}

func TestDBInfoReportsSecretKey(t *testing.T) {
	r := newRunner(t)
	seedSecrets(t, r, secretbox.Plain)
	res := r.mustRun("", "db", "info")
	for _, want := range []string{"secret key", "missing", "otp secrets (plain)", "otp secrets (encrypted)"} {
		if !strings.Contains(res.out, want) {
			t.Errorf("db info lacks %q:\n%s", want, res.out)
		}
	}
	file1, _ := keyFile(t, r)
	r.env = map[string]string{"CROFFLE_SECRET_KEY_FILE": file1}
	var info struct {
		SecretKey string `json:"secretKey"`
		Plain     int64  `json:"otpSecretsPlain"`
	}
	if err := json.Unmarshal([]byte(r.mustRun("", "db", "info", "--json").out), &info); err != nil {
		t.Fatal(err)
	}
	if info.SecretKey != "set" || info.Plain != 2 {
		t.Fatalf("db info --json = %+v, want the configured key and 2 plain secrets", info)
	}
	// Seal the secrets, then lose the key: status and disable still work.
	r.env = nil
	r.mustRun("", "secret", "rotate", "--new-key-file", file1)
	if out := r.mustRun("", "user", "otp", "status", "alice").out; !strings.Contains(out, "pending") {
		t.Fatalf("otp status without the key = %q", out)
	}
	r.mustRun("", "user", "otp", "disable", "alice")
	if out := r.mustRun("", "user", "otp", "status", "alice").out; !strings.Contains(out, "off") {
		t.Fatalf("otp status after disable = %q", out)
	}
}
