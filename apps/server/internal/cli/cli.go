// Package cli implements the operator subcommands of the croffle-server binary.
//
// Server 0.1 has no admin API: accounts, tokens and teams are managed with
// these commands, which talk to the database file directly. That is deliberate
// — whoever can run the binary against the file is already the operator — and
// it is why the dashboard (work item #10) stays read-only.
//
// In a container the usual form is:
//
//	docker exec <container> croffle-server user create alice --password-stdin
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/team-croffle/croffle/apps/server/internal/config"
	"github.com/team-croffle/croffle/apps/server/internal/store/sqlite"
)

// groups are the subcommand names. Anything else on the command line belongs to
// the server itself, so `croffle-server --listen :9000` keeps working.
var groups = map[string]func(*command) error{
	"user":  (*command).user,
	"token": (*command).token,
	"team":  (*command).team,
	"db":    (*command).database,
	"log":   (*command).logs,
	"help":  (*command).help,
}

// IsSubcommand reports whether args starts with one of the operator groups.
func IsSubcommand(args []string) bool {
	if len(args) == 0 {
		return false
	}
	_, ok := groups[args[0]]
	return ok
}

// Execute runs one subcommand. stdin is where --password-stdin reads from.
func Execute(ctx context.Context, args []string, getenv func(string) string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("no command given")
	}

	run, ok := groups[args[0]]
	if !ok {
		return fmt.Errorf("unknown command %q (try `croffle-server help`)", args[0])
	}

	cmd := &command{
		ctx:    ctx,
		group:  args[0],
		args:   args[1:],
		getenv: getenv,
		in:     stdin,
		out:    stdout,
		errOut: stderr,
	}
	defer cmd.close()

	return run(cmd)
}

// command is the shared state of one subcommand invocation.
type command struct {
	ctx    context.Context
	group  string
	args   []string
	getenv func(string) string
	in     io.Reader
	out    io.Writer
	errOut io.Writer

	db     *sqlite.DB
	asJSON bool
}

func (c *command) close() {
	if c.db != nil {
		_ = c.db.Close()
		c.db = nil
	}
}

// action pops the action name, e.g. "create" in `user create alice`.
func (c *command) action() string {
	if len(c.args) == 0 {
		return ""
	}
	action := c.args[0]
	c.args = c.args[1:]
	return action
}

// flags builds a flag set that always understands --db, plus --json when the
// action has something to print.
func (c *command) flags(action string, withJSON bool) *flag.FlagSet {
	fs := flag.NewFlagSet(c.group+" "+action, flag.ContinueOnError)
	fs.SetOutput(c.errOut)
	fs.String("db", "", "path to the database file (default "+config.EnvDBPath+")")
	if withJSON {
		fs.Bool("json", false, "print machine-readable JSON")
	}
	return fs
}

// parse reads the flags and opens the database.
//
// Opening applies pending migrations, the same as a server start would; when it
// does, it says so on stderr rather than changing the schema silently.
func (c *command) parse(fs *flag.FlagSet) error {
	positional, err := parseInterspersed(fs, c.args)
	if err != nil {
		return err
	}
	c.args = positional

	if jsonFlag := fs.Lookup("json"); jsonFlag != nil {
		c.asJSON = jsonFlag.Value.String() == "true"
	}

	path := fs.Lookup("db").Value.String()
	if path == "" {
		path = strings.TrimSpace(c.getenv(config.EnvDBPath))
	}
	if path == "" {
		path = config.DefaultDBPath
	}

	db, err := sqlite.Open(c.ctx, path)
	if err != nil {
		return err
	}
	c.db = db

	if db.AppliedMigrations > 0 {
		fmt.Fprintf(c.errOut, "applied %d migration(s) to %s\n", db.AppliedMigrations, path)
	}
	return nil
}

// parseInterspersed parses flags that appear before, after or between the
// positional arguments.
//
// The flag package stops at the first non-flag argument, so a plain Parse would
// silently ignore everything after it: `user create alice --db /tmp/x.db` would
// write to the default database instead of /tmp/x.db. Nobody writes their flags
// strictly first, so the loop pulls positionals out one at a time and keeps
// parsing the rest.
func parseInterspersed(fs *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		rest := fs.Args()
		if len(rest) == 0 {
			return positional, nil
		}
		// "--" ends flag parsing; keep the remainder verbatim.
		if rest[0] == "--" {
			return append(positional, rest[1:]...), nil
		}
		positional = append(positional, rest[0])
		args = rest[1:]
	}
}

// arg returns the nth positional argument after the flags.
func (c *command) arg(n int) string {
	if n >= len(c.args) {
		return ""
	}
	return c.args[n]
}

func (c *command) printf(format string, args ...any) {
	fmt.Fprintf(c.out, format, args...)
}

// table prints rows aligned, for a human reading the output.
func (c *command) table(header []string, rows [][]string) {
	tw := tabwriter.NewWriter(c.out, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, strings.Join(header, "\t"))
	for _, row := range rows {
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	_ = tw.Flush()
}

func (c *command) usage(lines ...string) error {
	return fmt.Errorf("usage:\n  %s", strings.Join(lines, "\n  "))
}

func (c *command) help() error {
	c.printf(`croffle-server - Croffle sync server

Usage:
  croffle-server [flags]                 start the server (default)
  croffle-server <command> <action> ...  operator commands

Server flags:
  --listen, --db-path, --log-level, --trust-proxy, --base-path, --version
  Each one also reads its CROFFLE_* environment variable.

Commands:
  user create <username> --password-stdin   create a login
  user list                                 list logins
  user disable <username>                   block authentication
  user enable <username>                    allow authentication again
  user set-password <username> --password-stdin
  token issue <username>                    issue a device token (printed once)
  token list [username]                     list devices
  token revoke <device-id>                  withdraw a device token
  team list                                 list teams
  team add-member <team-id> <username>      let an account see a team's rows
  db migrate                                apply pending migrations and exit
  db info                                   schema version, size and counts
  log dump                                  print the commit log as JSON lines

Every command takes --db to point at the database file, and the listing ones
take --json.
`)
	return nil
}
