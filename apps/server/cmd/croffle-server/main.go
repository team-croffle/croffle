// Command croffle-server is the optional self-hosted sync backend for Croffle.
//
// Croffle works fully offline without it; this server exists so that several
// devices of one account can exchange changesets, and so that a schedule can be
// shared with a team. TLS terminates in front of the process (Traefik, Caddy,
// nginx, an ingress — whatever the operator already runs), so the server speaks
// plain HTTP and only honours X-Forwarded-* when CROFFLE_TRUST_PROXY is set.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/team-croffle/croffle/apps/server/internal/auth"
	"github.com/team-croffle/croffle/apps/server/internal/cli"
	"github.com/team-croffle/croffle/apps/server/internal/config"
	"github.com/team-croffle/croffle/apps/server/internal/httpapi"
	"github.com/team-croffle/croffle/apps/server/internal/logging"
	"github.com/team-croffle/croffle/apps/server/internal/protocol"
	"github.com/team-croffle/croffle/apps/server/internal/retention"
	"github.com/team-croffle/croffle/apps/server/internal/store/sqlite"
	"github.com/team-croffle/croffle/apps/server/web"
)

// Injected via -ldflags; see the Makefile.
var (
	version = "dev"
	commit  = "unknown"
)

// Timeouts. ReadTimeout has to cover a slow client uploading a full-size push
// body, so it is generous; ReadHeaderTimeout stays short to shed idle sockets.
const (
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 60 * time.Second
	writeTimeout      = 60 * time.Second
	idleTimeout       = 120 * time.Second
	shutdownTimeout   = 10 * time.Second
)

func main() {
	if err := run(os.Args[1:], os.Getenv, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "croffle-server: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, getenv func(string) string, stdout, stderr io.Writer) error {
	// Operator commands share the binary so the container image has one
	// entrypoint: `docker exec … croffle-server user create …`.
	if cli.IsSubcommand(args) {
		return cli.Execute(context.Background(), args, getenv, os.Stdin, stdout, stderr)
	}
	// `serve` is the default, and also accepted explicitly.
	if len(args) > 0 && args[0] == "serve" {
		args = args[1:]
	}

	cfg, err := config.Load(args, getenv)
	if errors.Is(err, config.ErrVersionRequested) {
		fmt.Fprintf(stdout, "croffle-server %s (%s), sync protocol %d-%d\n", version, commit, protocol.Min, protocol.Max)
		return nil
	}
	if err != nil {
		return err
	}

	logger := logging.New(cfg.LogLevel, stdout)
	// Also the default, so an incidental slog call anywhere in the tree lands in
	// the same JSON stream at the configured level instead of stderr text.
	slog.SetDefault(logger)

	// Opening the store applies pending migrations, so a schema problem is a
	// startup failure rather than a surprise on the first request.
	db, err := sqlite.Open(context.Background(), cfg.DBPath)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("closing the database failed", slog.String("error", err.Error()))
		}
	}()
	if db.AppliedMigrations > 0 {
		logger.Info("schema migrated", slog.Int("applied", db.AppliedMigrations))
	}

	authService := auth.NewService(db, time.Now)

	// A binary without a dashboard build still serves the API; only the HTML is
	// missing, and the catch-all says so.
	assets, err := web.FS()
	if err != nil {
		return fmt.Errorf("read embedded dashboard: %w", err)
	}

	handler := httpapi.NewHandler(httpapi.Options{
		Version:    version,
		Commit:     commit,
		BasePath:   cfg.BasePath,
		TrustProxy: cfg.TrustProxy,
		Logger:     logger,
		Health:     db.Ping,
		Auth:       authService,
		Sync:       db,
		Dash:       db,
		Assets:     assets,
		Retention: func(ctx context.Context) (int64, int64, error) {
			oldest, err := db.OldestCommitSeq(ctx)
			if err != nil {
				return 0, 0, err
			}
			purged, err := db.PurgedThroughSeq(ctx)
			if err != nil {
				return 0, 0, err
			}
			return oldest, purged, nil
		},
	})

	// Listen before announcing: a bind failure should be the error the operator
	// sees, not a "started" line followed by a crash.
	ln, err := net.Listen("tcp", cfg.Listen)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.Listen, err)
	}

	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Sweeps once now and then daily; a server that restarts often still
	// collects its garbage.
	go retention.New(db, logger, cfg.RetainCommits, cfg.RetainOps).Run(ctx)

	serveErr := make(chan error, 1)
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
			return
		}
		serveErr <- nil
	}()

	logger.Info("server started",
		slog.String("version", version),
		slog.String("commit", commit),
		slog.String("addr", ln.Addr().String()),
		slog.String("base_path", cfg.BasePath),
		slog.String("db_path", cfg.DBPath),
		slog.Bool("trust_proxy", cfg.TrustProxy),
		slog.Duration("retain_commits", cfg.RetainCommits),
		slog.Duration("retain_ops", cfg.RetainOps),
		slog.Bool("dashboard", web.Built()),
		slog.Int("protocol_min", protocol.Min),
		slog.Int("protocol_max", protocol.Max),
	)

	select {
	case err := <-serveErr:
		if err != nil {
			return fmt.Errorf("serve: %w", err)
		}
		return nil
	case <-ctx.Done():
		logger.Info("shutdown requested", slog.Duration("grace", shutdownTimeout))
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		// In-flight requests did not finish within the grace period.
		return fmt.Errorf("shutdown: %w", err)
	}
	if err := <-serveErr; err != nil {
		return fmt.Errorf("serve: %w", err)
	}

	logger.Info("server stopped")
	return nil
}
