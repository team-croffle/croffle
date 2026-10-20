package config

import (
	"errors"
	"log/slog"
	"testing"
)

// envMap builds a getenv func, so tests never touch the real environment.
func envMap(kv map[string]string) func(string) string {
	return func(key string) string { return kv[key] }
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(nil, envMap(nil))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Listen != DefaultListen {
		t.Errorf("Listen = %q, want %q", cfg.Listen, DefaultListen)
	}
	if cfg.DBPath != DefaultDBPath {
		t.Errorf("DBPath = %q, want %q", cfg.DBPath, DefaultDBPath)
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Errorf("LogLevel = %v, want info", cfg.LogLevel)
	}
	if cfg.TrustProxy {
		t.Error("TrustProxy = true, want false by default (forged headers)")
	}
	if cfg.BasePath != "/" {
		t.Errorf("BasePath = %q, want /", cfg.BasePath)
	}
}

func TestLoadFromEnv(t *testing.T) {
	cfg, err := Load(nil, envMap(map[string]string{
		EnvListen:     "127.0.0.1:9000",
		EnvDBPath:     "./server.db",
		EnvLogLevel:   "DEBUG",
		EnvTrustProxy: "yes",
		EnvBasePath:   "/croffle/",
	}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Listen != "127.0.0.1:9000" {
		t.Errorf("Listen = %q", cfg.Listen)
	}
	if cfg.DBPath != "server.db" { // filepath.Clean drops the leading ./
		t.Errorf("DBPath = %q", cfg.DBPath)
	}
	if cfg.LogLevel != slog.LevelDebug {
		t.Errorf("LogLevel = %v, want debug", cfg.LogLevel)
	}
	if !cfg.TrustProxy {
		t.Error("TrustProxy = false, want true")
	}
	if cfg.BasePath != "/croffle" {
		t.Errorf("BasePath = %q, want /croffle (trailing slash trimmed)", cfg.BasePath)
	}
}

func TestFlagsOverrideEnv(t *testing.T) {
	cfg, err := Load(
		[]string{"--listen", ":9999", "--log-level", "error"},
		envMap(map[string]string{EnvListen: ":8787", EnvLogLevel: "info"}),
	)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Listen != ":9999" {
		t.Errorf("Listen = %q, want :9999", cfg.Listen)
	}
	if cfg.LogLevel != slog.LevelError {
		t.Errorf("LogLevel = %v, want error", cfg.LogLevel)
	}
}

func TestVersionRequested(t *testing.T) {
	_, err := Load([]string{"--version"}, envMap(nil))
	if !errors.Is(err, ErrVersionRequested) {
		t.Fatalf("err = %v, want ErrVersionRequested", err)
	}
}

func TestLoadRejectsInvalid(t *testing.T) {
	cases := []struct {
		name string
		args []string
		env  map[string]string
	}{
		{name: "listen without port", env: map[string]string{EnvListen: "abc"}},
		{name: "listen port not a number", env: map[string]string{EnvListen: "localhost:http"}},
		{name: "listen port out of range", env: map[string]string{EnvListen: ":99999"}},
		{name: "listen empty via flag", args: []string{"--listen", ""}},
		{name: "db path empty via flag", args: []string{"--db-path", ""}},
		{name: "unknown log level", env: map[string]string{EnvLogLevel: "trace"}},
		{name: "non boolean trust proxy", env: map[string]string{EnvTrustProxy: "maybe"}},
		{name: "base path without slash", env: map[string]string{EnvBasePath: "croffle"}},
		{name: "base path not normalised", env: map[string]string{EnvBasePath: "//croffle"}},
		{name: "base path with query", env: map[string]string{EnvBasePath: "/a?b"}},
		{name: "unexpected positional arg", args: []string{"serve"}},
		{name: "unknown flag", args: []string{"--nope"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := Load(tc.args, envMap(tc.env))
			if err == nil {
				t.Fatalf("Load succeeded with %+v, want an error", cfg)
			}
		})
	}
}

func TestParseLogLevel(t *testing.T) {
	cases := map[string]slog.Level{
		"debug":   slog.LevelDebug,
		"":        slog.LevelInfo,
		" info ":  slog.LevelInfo,
		"WARN":    slog.LevelWarn,
		"warning": slog.LevelWarn,
		"error":   slog.LevelError,
	}
	for in, want := range cases {
		got, err := ParseLogLevel(in)
		if err != nil {
			t.Errorf("ParseLogLevel(%q): %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("ParseLogLevel(%q) = %v, want %v", in, got, want)
		}
	}
}
