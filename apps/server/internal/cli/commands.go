package cli

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/team-croffle/croffle/apps/server/internal/auth"
	"github.com/team-croffle/croffle/apps/server/internal/store"
)

// maxPasswordStdin caps what --password-stdin will read, so a wrong pipe does
// not turn into an unbounded allocation.
const maxPasswordStdin = 4096

func (c *command) user() error {
	switch action := c.action(); action {
	case "create":
		return c.userCreate()
	case "list":
		return c.userList()
	case "disable":
		return c.userSetDisabled(action, true)
	case "enable":
		return c.userSetDisabled(action, false)
	case "set-password":
		return c.userSetPassword()
	default:
		return c.usage(
			"croffle-server user create <username> --password-stdin",
			"croffle-server user list [--json]",
			"croffle-server user disable <username>",
			"croffle-server user enable <username>",
			"croffle-server user set-password <username> --password-stdin")
	}
}

func (c *command) userCreate() error {
	fs := c.flags("create", false)
	password := fs.String("password", "", "password (prefer --password-stdin: a flag lands in the shell history)")
	fromStdin := fs.Bool("password-stdin", false, "read the password from stdin")
	id := fs.String("id", "", "account id (generated when empty)")
	if err := c.parse(fs); err != nil {
		return err
	}

	username := strings.TrimSpace(c.arg(0))
	if username == "" {
		return errors.New("usage: croffle-server user create <username> --password-stdin")
	}

	secret, err := readPassword(*password, *fromStdin, c.in)
	if err != nil {
		return err
	}

	hash, err := auth.HashPassword(secret)
	if err != nil {
		return err
	}

	accountID := strings.TrimSpace(*id)
	if accountID == "" {
		accountID = newID("acct")
	}

	account := store.Account{
		ID: accountID, Username: username, PasswordHash: hash, CreatedAt: time.Now(),
	}
	if err := c.db.CreateAccount(c.ctx, account); err != nil {
		if errors.Is(err, store.ErrDuplicate) {
			return fmt.Errorf("an account named %q already exists", username)
		}
		return err
	}

	// 0.1 has a single team and sharing only works for its members, so a new
	// account joins it right away; otherwise every account would need a second
	// command before it could receive a shared schedule.
	if err := c.db.AddTeamMember(c.ctx, store.DefaultTeamID, accountID, "member"); err != nil {
		return fmt.Errorf("add %s to the default team: %w", username, err)
	}

	c.printf("created account %s (%s) and added it to %s\n", username, accountID, store.DefaultTeamID)
	c.printf("next: croffle-server token issue %s\n", username)
	return nil
}

func (c *command) userList() error {
	fs := c.flags("list", true)
	if err := c.parse(fs); err != nil {
		return err
	}

	accounts, err := c.db.ListAccounts(c.ctx)
	if err != nil {
		return err
	}

	if c.asJSON {
		// Deliberately not the password hash: there is no operator task that
		// needs it, and it should not end up in a pasted terminal log.
		type row struct {
			ID        string `json:"id"`
			Username  string `json:"username"`
			CreatedAt string `json:"createdAt"`
			Disabled  bool   `json:"disabled"`
		}
		out := make([]row, 0, len(accounts))
		for _, a := range accounts {
			out = append(out, row{a.ID, a.Username, formatTime(&a.CreatedAt), a.Disabled()})
		}
		return c.writeJSON(out)
	}

	rows := make([][]string, 0, len(accounts))
	for _, a := range accounts {
		state := "active"
		if a.Disabled() {
			state = "disabled"
		}
		rows = append(rows, []string{a.Username, a.ID, formatTime(&a.CreatedAt), state})
	}
	c.table([]string{"USERNAME", "ID", "CREATED", "STATE"}, rows)
	return nil
}

func (c *command) userSetDisabled(action string, disabled bool) error {
	fs := c.flags(action, false)
	if err := c.parse(fs); err != nil {
		return err
	}

	account, err := c.resolveAccount(c.arg(0))
	if err != nil {
		return err
	}
	if err := c.db.SetAccountDisabled(c.ctx, account.ID, disabled); err != nil {
		return err
	}

	if disabled {
		// Disabling is the hard stop: unlike revoking a token, the account
		// cannot log back in.
		c.printf("disabled %s; existing tokens stop working immediately\n", account.Username)
	} else {
		c.printf("enabled %s\n", account.Username)
	}
	return nil
}

func (c *command) userSetPassword() error {
	fs := c.flags("set-password", false)
	password := fs.String("password", "", "password (prefer --password-stdin)")
	fromStdin := fs.Bool("password-stdin", false, "read the password from stdin")
	if err := c.parse(fs); err != nil {
		return err
	}

	account, err := c.resolveAccount(c.arg(0))
	if err != nil {
		return err
	}

	secret, err := readPassword(*password, *fromStdin, c.in)
	if err != nil {
		return err
	}
	hash, err := auth.HashPassword(secret)
	if err != nil {
		return err
	}
	if err := c.db.SetAccountPassword(c.ctx, account.ID, hash); err != nil {
		return err
	}

	// Existing tokens keep working on purpose: they are separate credentials,
	// and `token revoke` is the tool for withdrawing them.
	c.printf("password updated for %s (existing device tokens still work; revoke them with `token revoke`)\n",
		account.Username)
	return nil
}

func (c *command) token() error {
	switch action := c.action(); action {
	case "issue":
		return c.tokenIssue()
	case "list":
		return c.tokenList()
	case "revoke":
		return c.tokenRevoke()
	default:
		return c.usage(
			"croffle-server token issue <username> [--device-name NAME] [--client-id ID]",
			"croffle-server token list [username] [--json]",
			"croffle-server token revoke <device-id>")
	}
}

func (c *command) tokenIssue() error {
	fs := c.flags("issue", false)
	deviceName := fs.String("device-name", "", "label for this device")
	clientID := fs.String("client-id", "", "the desktop's clientId (generated when empty)")
	if err := c.parse(fs); err != nil {
		return err
	}

	account, err := c.resolveAccount(c.arg(0))
	if err != nil {
		return err
	}
	if account.Disabled() {
		return fmt.Errorf("account %s is disabled; enable it before issuing a token", account.Username)
	}

	token, hash, err := auth.NewToken()
	if err != nil {
		return err
	}

	deviceID := strings.TrimSpace(*clientID)
	generated := deviceID == ""
	if generated {
		deviceID = newID("client")
	}

	if err := c.db.UpsertDevice(c.ctx, store.Device{
		ID: deviceID, AccountID: account.ID,
		Name: valueOr(*deviceName, "issued by cli"), TokenHash: hash, CreatedAt: time.Now(),
	}); err != nil {
		return err
	}

	// The only place the plaintext ever appears. Warn on stderr so that piping
	// stdout into a file does not bury the warning.
	fmt.Fprintf(c.errOut, "this token is shown once and is not recoverable; it will be in your shell history\n")
	c.printf("%s\n", token)

	if generated {
		fmt.Fprintf(c.errOut,
			"device %s was generated. The desktop uses its own clientId, so after pasting this token "+
				"its writes carry that id instead; pass --client-id to match them up.\n", deviceID)
	}
	return nil
}

func (c *command) tokenList() error {
	fs := c.flags("list", true)
	if err := c.parse(fs); err != nil {
		return err
	}

	accountID := ""
	if name := strings.TrimSpace(c.arg(0)); name != "" {
		account, err := c.resolveAccount(name)
		if err != nil {
			return err
		}
		accountID = account.ID
	}

	devices, err := c.db.ListDevices(c.ctx, accountID)
	if err != nil {
		return err
	}

	if c.asJSON {
		// Not even the hash: it is credential material and no operator task
		// needs to see it.
		type row struct {
			ID        string `json:"id"`
			AccountID string `json:"accountId"`
			Name      string `json:"name"`
			CreatedAt string `json:"createdAt"`
			LastSeen  string `json:"lastSeenAt,omitempty"`
			State     string `json:"state"`
		}
		out := make([]row, 0, len(devices))
		for _, d := range devices {
			out = append(out, row{d.ID, d.AccountID, d.Name,
				formatTime(&d.CreatedAt), formatTime(d.LastSeenAt), deviceState(d)})
		}
		return c.writeJSON(out)
	}

	rows := make([][]string, 0, len(devices))
	for _, d := range devices {
		rows = append(rows, []string{d.ID, d.AccountID, d.Name,
			formatTime(&d.CreatedAt), formatTime(d.LastSeenAt), deviceState(d)})
	}
	c.table([]string{"DEVICE", "ACCOUNT", "NAME", "CREATED", "LAST SEEN", "STATE"}, rows)
	return nil
}

func (c *command) tokenRevoke() error {
	fs := c.flags("revoke", false)
	if err := c.parse(fs); err != nil {
		return err
	}

	deviceID := strings.TrimSpace(c.arg(0))
	if deviceID == "" {
		return errors.New("usage: croffle-server token revoke <device-id>")
	}
	if err := c.db.RevokeDevice(c.ctx, deviceID); err != nil {
		return err
	}

	// Say what revoking does and does not do, because the difference from
	// disabling an account is easy to get wrong.
	c.printf("revoked the token of device %s\n", deviceID)
	c.printf("the account can obtain a new one by signing in again; use `user disable` to stop it entirely\n")
	return nil
}

func (c *command) team() error {
	switch action := c.action(); action {
	case "list":
		return c.teamList()
	case "add-member":
		return c.teamAddMember()
	default:
		return c.usage(
			"croffle-server team list [--json]",
			"croffle-server team add-member <team-id> <username> [--role ROLE]")
	}
}

func (c *command) teamList() error {
	fs := c.flags("list", true)
	if err := c.parse(fs); err != nil {
		return err
	}

	teams, err := c.db.ListTeams(c.ctx)
	if err != nil {
		return err
	}

	if c.asJSON {
		return c.writeJSON(teams)
	}

	rows := make([][]string, 0, len(teams))
	for _, t := range teams {
		rows = append(rows, []string{t.ID, t.Name, formatTime(&t.CreatedAt)})
	}
	c.table([]string{"ID", "NAME", "CREATED"}, rows)
	return nil
}

func (c *command) teamAddMember() error {
	fs := c.flags("add-member", false)
	role := fs.String("role", "member", "role inside the team (not enforced in 0.1)")
	if err := c.parse(fs); err != nil {
		return err
	}

	teamID := strings.TrimSpace(c.arg(0))
	if teamID == "" {
		return errors.New("usage: croffle-server team add-member <team-id> <username>")
	}
	account, err := c.resolveAccount(c.arg(1))
	if err != nil {
		return err
	}

	if err := c.db.AddTeamMember(c.ctx, teamID, account.ID, *role); err != nil {
		return err
	}
	c.printf("added %s to team %s as %s\n", account.Username, teamID, *role)
	return nil
}

func (c *command) database() error {
	switch action := c.action(); action {
	case "migrate":
		return c.dbMigrate()
	case "info":
		return c.dbInfo()
	default:
		return c.usage(
			"croffle-server db migrate",
			"croffle-server db info [--json]")
	}
}

func (c *command) dbMigrate() error {
	fs := c.flags("migrate", false)
	// parse() opens the database, and opening is what applies migrations; this
	// command exists so that can be done without starting the server.
	if err := c.parse(fs); err != nil {
		return err
	}

	info, err := c.db.Info(c.ctx)
	if err != nil {
		return err
	}
	c.printf("database %s is at schema version %d (%d migration(s) applied now)\n",
		info.Path, info.SchemaVersion, c.db.AppliedMigrations)
	return nil
}

func (c *command) dbInfo() error {
	fs := c.flags("info", true)
	if err := c.parse(fs); err != nil {
		return err
	}

	info, err := c.db.Info(c.ctx)
	if err != nil {
		return err
	}

	if c.asJSON {
		return c.writeJSON(info)
	}

	c.table([]string{"FIELD", "VALUE"}, [][]string{
		{"path", info.Path},
		{"schema version", strconv.FormatInt(info.SchemaVersion, 10)},
		{"size", formatBytes(info.SizeBytes)},
		{"accounts", strconv.FormatInt(info.Accounts, 10)},
		{"devices", strconv.FormatInt(info.Devices, 10)},
		{"teams", strconv.FormatInt(info.Teams, 10)},
		{"rows", strconv.FormatInt(info.Rows, 10)},
		{"commits", strconv.FormatInt(info.Commits, 10)},
		{"oldest seq", strconv.FormatInt(info.OldestSeq, 10)},
		{"newest seq", strconv.FormatInt(info.NewestSeq, 10)},
	})
	return nil
}

func (c *command) logs() error {
	switch action := c.action(); action {
	case "dump":
		return c.logDump()
	default:
		return c.usage("croffle-server log dump [--since-seq N] [--limit N]")
	}
}

// logDump always writes JSON lines: it is the first tool to reach for when a
// client and the server disagree, and that output is meant to be filtered with
// jq or diffed against the client's outbox.
func (c *command) logDump() error {
	fs := c.flags("dump", false)
	since := fs.Int64("since-seq", 0, "only commits after this serverSeq")
	limit := fs.Int("limit", 100, "maximum number of commits")
	if err := c.parse(fs); err != nil {
		return err
	}

	changes, err := c.db.DumpCommitLog(c.ctx, *since, *limit)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(c.out)
	for _, change := range changes {
		if err := enc.Encode(change); err != nil {
			return fmt.Errorf("encode commit %d: %w", change.ServerSeq, err)
		}
	}
	if len(changes) == 0 {
		fmt.Fprintf(c.errOut, "no commits after seq %d\n", *since)
	}
	return nil
}

// resolveAccount accepts a username or an account id, so an operator can copy
// either column out of `user list`.
func (c *command) resolveAccount(identifier string) (*store.Account, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return nil, errors.New("an account (username or id) is required")
	}

	account, err := c.db.FindAccountByUsername(c.ctx, identifier)
	if err == nil {
		return account, nil
	}
	if !errors.Is(err, store.ErrNotFound) {
		return nil, err
	}

	account, err = c.db.FindAccountByID(c.ctx, identifier)
	if errors.Is(err, store.ErrNotFound) {
		return nil, fmt.Errorf("no account %q", identifier)
	}
	if err != nil {
		return nil, err
	}
	return account, nil
}

func (c *command) writeJSON(v any) error {
	enc := json.NewEncoder(c.out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// readPassword prefers stdin: a password passed as a flag ends up in the shell
// history and in the process list.
func readPassword(flagValue string, fromStdin bool, stdin io.Reader) (string, error) {
	if fromStdin {
		data, err := io.ReadAll(io.LimitReader(stdin, maxPasswordStdin))
		if err != nil {
			return "", fmt.Errorf("read password from stdin: %w", err)
		}
		secret := strings.TrimRight(string(data), "\r\n")
		if secret == "" {
			return "", errors.New("empty password on stdin")
		}
		return secret, nil
	}

	if flagValue == "" {
		return "", errors.New("a password is required: pass --password-stdin (preferred) or --password")
	}
	return flagValue, nil
}

func newID(prefix string) string {
	var buf [8]byte
	// rand.Read panics rather than failing on supported platforms.
	_, _ = rand.Read(buf[:])
	return prefix + "-" + hex.EncodeToString(buf[:])
}

func deviceState(d store.Device) string {
	switch {
	case d.Revoked():
		return "revoked"
	case d.LastSeenAt == nil:
		// Issued but never used: a token waiting to be pasted into a desktop.
		return "pending"
	default:
		return "active"
	}
}

func formatTime(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func formatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return strconv.FormatInt(n, 10) + " B"
	}
	value := float64(n)
	for _, suffix := range []string{"KiB", "MiB", "GiB"} {
		value /= unit
		if value < unit {
			return fmt.Sprintf("%.1f %s", value, suffix)
		}
	}
	return fmt.Sprintf("%.1f TiB", value/unit)
}

func valueOr(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}
