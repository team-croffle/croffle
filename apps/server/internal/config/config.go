// Package config loads server configuration from environment variables and
// equally named flags.
//
// Env is the primary source because the server ships as a container image; the
// flags exist so that local runs and `docker exec croffle-server …` stay
// convenient. A flag, when given, overrides the environment.
//
// Invalid or empty values fail at startup instead of being silently defaulted:
// a server listening on the wrong port or writing to the wrong database is
// worse than a server that refuses to boot.
package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Environment variable names. Flags use the same name without the prefix and
// with dashes, e.g. CROFFLE_DB_PATH -> --db-path.
const (
	EnvListen     = "CROFFLE_LISTEN"
	EnvDBPath     = "CROFFLE_DB_PATH"
	EnvLogLevel   = "CROFFLE_LOG_LEVEL"
	EnvTrustProxy = "CROFFLE_TRUST_PROXY"
	EnvBasePath   = "CROFFLE_BASE_PATH"

	EnvRetainCommitsDays = "CROFFLE_RETAIN_COMMITS_DAYS"
	EnvRetainOpsDays     = "CROFFLE_RETAIN_OPS_DAYS"
)

// Defaults assume the container image. The README documents ./server.db for
// local runs; `make run` sets it.
const (
	DefaultListen   = ":8787"
	DefaultDBPath   = "/var/lib/croffle/server.db"
	DefaultLogLevel = "info"
	DefaultBasePath = "/"

	// Retention. 90 days of history is far longer than a device is normally
	// away, and 30 days of idempotency keys is far longer than any retry.
	DefaultRetainCommitsDays = 90
	DefaultRetainOpsDays     = 30
	// Below this, retention starts breaking the guarantees it supports: a
	// cursor would expire while a laptop is on holiday, and a late retry could
	// be applied twice.
	MinRetainDays = 7
)

// ErrVersionRequested is returned by Load when --version was passed. The caller
// prints the version and exits successfully.
var ErrVersionRequested = errors.New("version requested")

// Config is the fully validated configuration. Zero values never reach it.
type Config struct {
	// Listen is a host:port accepted by net.Listen, e.g. ":8787".
	Listen string
	// DBPath is the SQLite file. The store (work item #2) opens it.
	DBPath string
	// LogLevel is the minimum level of the JSON logger.
	LogLevel slog.Level
	// TrustProxy enables X-Forwarded-For / X-Forwarded-Proto. Only turn this on
	// when a reverse proxy is actually in front: otherwise any client can forge
	// its own address.
	TrustProxy bool
	// BasePath is the mount point, always starting with "/" and never ending
	// with one (except the root "/"). Set it when the proxy forwards a subpath
	// without stripping it.
	BasePath string
	// RetainCommits is how long the commit log is kept. A client whose cursor
	// falls outside it has to bootstrap again.
	RetainCommits time.Duration
	// RetainOps is how long idempotency keys are kept. A retry arriving later
	// than this could be applied a second time.
	RetainOps time.Duration
}

// Load parses args (without the program name) and the environment.
func Load(args []string, getenv func(string) string) (*Config, error) {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}

	fs := flag.NewFlagSet("croffle-server", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	listen := fs.String("listen", envOr(getenv, EnvListen, DefaultListen), "host:port to listen on")
	dbPath := fs.String("db-path", envOr(getenv, EnvDBPath, DefaultDBPath), "path to the SQLite database file")
	logLevel := fs.String("log-level", envOr(getenv, EnvLogLevel, DefaultLogLevel), "debug | info | warn | error")
	trustProxy := fs.String("trust-proxy", envOr(getenv, EnvTrustProxy, ""), "trust X-Forwarded-* headers (true/false)")
	basePath := fs.String("base-path", envOr(getenv, EnvBasePath, DefaultBasePath), "mount point, e.g. /croffle")
	retainCommits := fs.String("retain-commits-days", envOr(getenv, EnvRetainCommitsDays, ""),
		fmt.Sprintf("days of commit log to keep (default %d, minimum %d)", DefaultRetainCommitsDays, MinRetainDays))
	retainOps := fs.String("retain-ops-days", envOr(getenv, EnvRetainOpsDays, ""),
		fmt.Sprintf("days of idempotency keys to keep (default %d, minimum %d)", DefaultRetainOpsDays, MinRetainDays))
	showVersion := fs.Bool("version", false, "print version and exit")

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("parse flags: %w", err)
	}
	if *showVersion {
		return nil, ErrVersionRequested
	}
	if fs.NArg() > 0 {
		return nil, fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}

	cfg := &Config{}
	var err error

	if cfg.Listen, err = parseListen(*listen); err != nil {
		return nil, err
	}
	if cfg.DBPath, err = parseDBPath(*dbPath); err != nil {
		return nil, err
	}
	if cfg.LogLevel, err = ParseLogLevel(*logLevel); err != nil {
		return nil, err
	}
	if cfg.TrustProxy, err = parseBool(EnvTrustProxy, *trustProxy); err != nil {
		return nil, err
	}
	if cfg.BasePath, err = parseBasePath(*basePath); err != nil {
		return nil, err
	}
	if cfg.RetainCommits, err = parseRetainDays(EnvRetainCommitsDays, *retainCommits, DefaultRetainCommitsDays); err != nil {
		return nil, err
	}
	if cfg.RetainOps, err = parseRetainDays(EnvRetainOpsDays, *retainOps, DefaultRetainOpsDays); err != nil {
		return nil, err
	}

	return cfg, nil
}

func envOr(getenv func(string) string, key, fallback string) string {
	if v := strings.TrimSpace(getenv(key)); v != "" {
		return v
	}
	return fallback
}

func parseListen(raw string) (string, error) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return "", fmt.Errorf("%s must not be empty (e.g. %q)", EnvListen, DefaultListen)
	}
	host, port, err := net.SplitHostPort(v)
	if err != nil {
		return "", fmt.Errorf("%s %q is not host:port (e.g. %q): %w", EnvListen, v, DefaultListen, err)
	}
	n, err := strconv.Atoi(port)
	if err != nil {
		// Named ports ("http") resolve differently per system; require a number
		// so the bound port is never a surprise.
		return "", fmt.Errorf("%s %q: port must be a number", EnvListen, v)
	}
	if n < 0 || n > 65535 {
		return "", fmt.Errorf("%s %q: port out of range", EnvListen, v)
	}
	return net.JoinHostPort(host, port), nil
}

func parseDBPath(raw string) (string, error) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return "", fmt.Errorf("%s must not be empty (e.g. %q)", EnvDBPath, DefaultDBPath)
	}
	return filepath.Clean(v), nil
}

// ParseLogLevel maps a configuration string to a slog level.
func ParseLogLevel(raw string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug, nil
	case "", "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("%s %q: want debug | info | warn | error", EnvLogLevel, raw)
	}
}

func parseBool(name, raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "0", "false", "no", "off":
		return false, nil
	case "1", "true", "yes", "on":
		return true, nil
	default:
		return false, fmt.Errorf("%s %q: want true or false", name, raw)
	}
}

// parseRetainDays refuses a retention that is too short rather than accepting
// it: a window below MinRetainDays silently breaks cursors and idempotency,
// and the symptom (a device re-bootstrapping, or a duplicate row) would show up
// far from the setting that caused it.
func parseRetainDays(name, raw string, fallback int) (time.Duration, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Duration(fallback) * 24 * time.Hour, nil
	}

	days, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s %q: want a number of days", name, raw)
	}
	if days < MinRetainDays {
		return 0, fmt.Errorf("%s %d: must be at least %d days, otherwise cursors and idempotency keys expire while they are still needed",
			name, days, MinRetainDays)
	}
	return time.Duration(days) * 24 * time.Hour, nil
}

func parseBasePath(raw string) (string, error) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return DefaultBasePath, nil
	}
	if !strings.HasPrefix(v, "/") {
		return "", fmt.Errorf("%s %q: must start with /", EnvBasePath, raw)
	}
	if strings.ContainsAny(v, "?# \t") {
		return "", fmt.Errorf("%s %q: must not contain spaces, ? or #", EnvBasePath, raw)
	}
	cleaned := filepath.ToSlash(filepath.Clean(v))
	if cleaned != v && cleaned+"/" != v {
		// Catches "//a", "/a/../b" and friends: the mount point has to be
		// something the router can prefix verbatim.
		return "", fmt.Errorf("%s %q: not a normalised path (did you mean %q?)", EnvBasePath, raw, cleaned)
	}
	return cleaned, nil
}
