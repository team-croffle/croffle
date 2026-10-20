package cli

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

type result struct {
	out string
	err string
}

type runner struct {
	t    *testing.T
	path string
}

func newRunner(t *testing.T) *runner {
	t.Helper()
	return &runner{t: t, path: filepath.Join(t.TempDir(), "server.db")}
}

// run executes a subcommand against the runner's database. stdin feeds
// --password-stdin.
func (r *runner) run(stdin string, args ...string) (result, error) {
	r.t.Helper()
	var out, errOut strings.Builder

	args = append(args, "--db", r.path)
	err := Execute(context.Background(), args,
		func(string) string { return "" }, strings.NewReader(stdin), &out, &errOut)

	return result{out: out.String(), err: errOut.String()}, err
}

func (r *runner) mustRun(stdin string, args ...string) result {
	r.t.Helper()
	res, err := r.run(stdin, args...)
	if err != nil {
		r.t.Fatalf("%v: %v (stderr: %s)", args, err, res.err)
	}
	return res
}

func TestUserCreateAndList(t *testing.T) {
	r := newRunner(t)

	res := r.mustRun("hunter2\n", "user", "create", "alice", "--password-stdin")
	if !strings.Contains(res.out, "created account alice") {
		t.Fatalf("output = %q", res.out)
	}
	// A new account joins the single team right away, otherwise it could not
	// receive a shared schedule without a second command.
	if !strings.Contains(res.out, "team-default") {
		t.Errorf("output does not mention the default team: %q", res.out)
	}

	res = r.mustRun("", "user", "list")
	if !strings.Contains(res.out, "alice") || !strings.Contains(res.out, "active") {
		t.Errorf("list output = %q", res.out)
	}
	// The hash has no operator use and should not be printable.
	if strings.Contains(res.out, "argon2") {
		t.Error("user list printed the password hash")
	}
}

// Regression: the flag package stops at the first non-flag argument, so
// `user create alice --db X` used to write to the default database instead.
func TestFlagsAfterPositionalArgumentsAreHonoured(t *testing.T) {
	r := newRunner(t)

	// --db is appended after the positional by the runner, which is exactly
	// the order that used to be dropped.
	r.mustRun("hunter2\n", "user", "create", "alice", "--password-stdin")

	res := r.mustRun("", "user", "list", "--json")
	var accounts []struct {
		Username string `json:"username"`
	}
	if err := json.Unmarshal([]byte(res.out), &accounts); err != nil {
		t.Fatalf("decode %q: %v", res.out, err)
	}
	if len(accounts) != 1 || accounts[0].Username != "alice" {
		t.Fatalf("accounts = %+v, want alice in the database named by --db", accounts)
	}
}

func TestUserCreateRejectsDuplicateUsername(t *testing.T) {
	r := newRunner(t)
	r.mustRun("hunter2\n", "user", "create", "alice", "--password-stdin")

	_, err := r.run("other\n", "user", "create", "alice", "--password-stdin")
	if err == nil {
		t.Fatal("creating a duplicate username succeeded")
	}
	// An operator should read "taken", not a constraint violation.
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error = %v, want a plain already-exists message", err)
	}
	if strings.Contains(err.Error(), "UNIQUE constraint") {
		t.Errorf("error leaks the raw constraint: %v", err)
	}
}

func TestUserCreateRequiresAPassword(t *testing.T) {
	r := newRunner(t)

	if _, err := r.run("", "user", "create", "alice"); err == nil {
		t.Error("creating an account without a password succeeded")
	}
	if _, err := r.run("", "user", "create", "alice", "--password-stdin"); err == nil {
		t.Error("creating an account with an empty stdin password succeeded")
	}
	if _, err := r.run("", "user", "create", "--password", "x"); err == nil {
		t.Error("creating an account without a username succeeded")
	}
}

func TestUserDisableAndEnable(t *testing.T) {
	r := newRunner(t)
	r.mustRun("hunter2\n", "user", "create", "alice", "--password-stdin")

	res := r.mustRun("", "user", "disable", "alice")
	if !strings.Contains(res.out, "disabled alice") {
		t.Errorf("output = %q", res.out)
	}
	if list := r.mustRun("", "user", "list"); !strings.Contains(list.out, "disabled") {
		t.Errorf("list does not show the state: %q", list.out)
	}

	r.mustRun("", "user", "enable", "alice")
	if list := r.mustRun("", "user", "list"); !strings.Contains(list.out, "active") {
		t.Errorf("list after enable = %q", list.out)
	}

	if _, err := r.run("", "user", "disable", "nobody"); err == nil {
		t.Error("disabling a missing account succeeded")
	}
}

func TestTokenIssueListAndRevoke(t *testing.T) {
	r := newRunner(t)
	r.mustRun("hunter2\n", "user", "create", "alice", "--password-stdin")

	issued := r.mustRun("", "token", "issue", "alice", "--device-name", "laptop")
	token := strings.TrimSpace(issued.out)
	if len(token) < 40 {
		t.Fatalf("token = %q, want the plaintext on stdout", token)
	}
	// The warning belongs on stderr so piping stdout to a file keeps it visible.
	if !strings.Contains(issued.err, "shown once") {
		t.Errorf("stderr = %q, want the one-time warning", issued.err)
	}

	listed := r.mustRun("", "token", "list", "alice", "--json")
	var devices []struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		State string `json:"state"`
	}
	if err := json.Unmarshal([]byte(listed.out), &devices); err != nil {
		t.Fatalf("decode %q: %v", listed.out, err)
	}
	if len(devices) != 1 {
		t.Fatalf("devices = %d, want 1", len(devices))
	}
	if devices[0].Name != "laptop" || devices[0].State != "pending" {
		t.Errorf("device = %+v, want a pending laptop", devices[0])
	}
	// Not even the hash: it is credential material.
	if strings.Contains(listed.out, "tokenHash") || strings.Contains(listed.out, token) {
		t.Error("token list exposed credential material")
	}

	revoked := r.mustRun("", "token", "revoke", devices[0].ID)
	if !strings.Contains(revoked.out, "revoked") {
		t.Errorf("output = %q", revoked.out)
	}
	// The difference from disabling an account is easy to get wrong, so the
	// command says it.
	if !strings.Contains(revoked.out, "user disable") {
		t.Errorf("revoke does not explain how to stop the account: %q", revoked.out)
	}

	if listed := r.mustRun("", "token", "list", "--json"); !strings.Contains(listed.out, `"state": "revoked"`) {
		t.Errorf("state after revoke = %q", listed.out)
	}
	if _, err := r.run("", "token", "revoke", "no-such-device"); err == nil {
		t.Error("revoking a missing device succeeded")
	}
}

func TestTokenIssueWithExplicitClientID(t *testing.T) {
	r := newRunner(t)
	r.mustRun("hunter2\n", "user", "create", "alice", "--password-stdin")

	issued := r.mustRun("", "token", "issue", "alice", "--client-id", "desktop-1")
	// With a client id given there is nothing to warn about.
	if strings.Contains(issued.err, "was generated") {
		t.Errorf("stderr = %q, want no generated-id note", issued.err)
	}

	listed := r.mustRun("", "token", "list", "--json")
	if !strings.Contains(listed.out, "desktop-1") {
		t.Errorf("device id = %q, want the supplied client id", listed.out)
	}
}

func TestTokenIssueRefusesDisabledAccount(t *testing.T) {
	r := newRunner(t)
	r.mustRun("hunter2\n", "user", "create", "alice", "--password-stdin")
	r.mustRun("", "user", "disable", "alice")

	if _, err := r.run("", "token", "issue", "alice"); err == nil {
		t.Error("issuing a token for a disabled account succeeded")
	}
}

func TestSetPasswordKeepsExistingTokens(t *testing.T) {
	r := newRunner(t)
	r.mustRun("hunter2\n", "user", "create", "alice", "--password-stdin")
	r.mustRun("", "token", "issue", "alice")

	res := r.mustRun("new-secret\n", "user", "set-password", "alice", "--password-stdin")
	// Tokens are separate credentials; `token revoke` is what withdraws them.
	if !strings.Contains(res.out, "still work") {
		t.Errorf("output = %q, want it to say tokens survive", res.out)
	}
	if listed := r.mustRun("", "token", "list", "--json"); strings.Contains(listed.out, "revoked") {
		t.Errorf("tokens were revoked by a password change: %q", listed.out)
	}
}

func TestTeamListAndAddMember(t *testing.T) {
	r := newRunner(t)
	r.mustRun("hunter2\n", "user", "create", "alice", "--password-stdin")

	res := r.mustRun("", "team", "list", "--json")
	var teams []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(res.out), &teams); err != nil {
		t.Fatalf("decode %q: %v", res.out, err)
	}
	if len(teams) != 1 || teams[0].ID != "team-default" {
		t.Fatalf("teams = %+v, want the seeded default team", teams)
	}

	// Already a member from user create: adding again must not fail.
	if _, err := r.run("", "team", "add-member", "team-default", "alice"); err != nil {
		t.Errorf("re-adding a member failed: %v", err)
	}
	if _, err := r.run("", "team", "add-member", "team-default", "nobody"); err == nil {
		t.Error("adding a missing account succeeded")
	}
}

func TestDBMigrateAndInfo(t *testing.T) {
	r := newRunner(t)

	res := r.mustRun("", "db", "migrate")
	if !strings.Contains(res.out, "schema version") {
		t.Errorf("migrate output = %q", res.out)
	}
	// The first open applied it; the second has nothing to do.
	if second := r.mustRun("", "db", "migrate"); !strings.Contains(second.out, "0 migration(s) applied now") {
		t.Errorf("second migrate = %q, want it to report no work", second.out)
	}

	r.mustRun("hunter2\n", "user", "create", "alice", "--password-stdin")

	info := r.mustRun("", "db", "info", "--json")
	var parsed struct {
		SchemaVersion int64  `json:"schemaVersion"`
		Accounts      int64  `json:"accounts"`
		Teams         int64  `json:"teams"`
		Path          string `json:"path"`
	}
	if err := json.Unmarshal([]byte(info.out), &parsed); err != nil {
		t.Fatalf("decode %q: %v", info.out, err)
	}
	// Not pinned to a number: every new migration would otherwise break this.
	if parsed.SchemaVersion < 1 || parsed.Accounts != 1 || parsed.Teams != 1 {
		t.Errorf("info = %+v", parsed)
	}
	if parsed.Path != r.path {
		t.Errorf("path = %q, want %q", parsed.Path, r.path)
	}
}

func TestLogDumpOnAnEmptyLog(t *testing.T) {
	r := newRunner(t)

	res := r.mustRun("", "log", "dump")
	if strings.TrimSpace(res.out) != "" {
		t.Errorf("stdout = %q, want nothing to dump", res.out)
	}
	// The "nothing here" note goes to stderr so stdout stays pipeable into jq.
	if !strings.Contains(res.err, "no commits") {
		t.Errorf("stderr = %q", res.err)
	}
}

func TestUsageAndUnknownCommands(t *testing.T) {
	r := newRunner(t)

	for _, args := range [][]string{
		{"user"}, {"user", "nonsense"}, {"token"}, {"team"}, {"db"}, {"log"},
	} {
		res, err := r.run("", args...)
		if err == nil {
			t.Errorf("%v succeeded, want a usage error (out: %q)", args, res.out)
			continue
		}
		if !strings.Contains(err.Error(), "usage") {
			t.Errorf("%v error = %v, want usage text", args, err)
		}
	}

	var out, errOut strings.Builder
	err := Execute(context.Background(), []string{"nonsense"},
		func(string) string { return "" }, strings.NewReader(""), &out, &errOut)
	if err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("error = %v, want unknown command", err)
	}
}

func TestIsSubcommand(t *testing.T) {
	cases := map[string]bool{
		"user": true, "token": true, "team": true, "db": true, "log": true, "help": true,
		"--listen": false, "serve": false, "": false,
	}
	for arg, want := range cases {
		args := []string{arg}
		if arg == "" {
			args = nil
		}
		if got := IsSubcommand(args); got != want {
			t.Errorf("IsSubcommand(%q) = %v, want %v", arg, got, want)
		}
	}
}

func TestHelpListsEveryCommand(t *testing.T) {
	var out, errOut strings.Builder
	if err := Execute(context.Background(), []string{"help"},
		func(string) string { return "" }, strings.NewReader(""), &out, &errOut); err != nil {
		t.Fatalf("help: %v", err)
	}

	// help must not need a database: it is what an operator runs first.
	for _, want := range []string{"user create", "token issue", "token revoke", "team add-member", "db migrate", "log dump"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("help does not mention %q", want)
		}
	}
}

func TestDatabasePathComesFromEnvWhenFlagIsAbsent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "from-env.db")
	var out, errOut strings.Builder

	err := Execute(context.Background(), []string{"db", "info"},
		func(key string) string {
			if key == "CROFFLE_DB_PATH" {
				return path
			}
			return ""
		},
		strings.NewReader(""), &out, &errOut)
	if err != nil {
		t.Fatalf("db info: %v", err)
	}
	if !strings.Contains(out.String(), path) {
		t.Errorf("output = %q, want the env path", out.String())
	}
}
