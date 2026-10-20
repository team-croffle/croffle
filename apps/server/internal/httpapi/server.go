// Package httpapi assembles the HTTP surface: middleware chain, routing and the
// handlers that do not belong to a feature package yet.
//
// Routing uses the net/http pattern syntax (Go 1.22+), so the method is part of
// the pattern and a mismatch answers 405 without any per-handler check.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// HealthFunc reports whether the storage layer is reachable. It is nil until
// the store lands (work item #2); GET /health then reports "unconfigured".
type HealthFunc func(context.Context) error

// Options configures NewHandler. Logger is required; the rest has defaults.
type Options struct {
	// Version and Commit come from the main package's ldflags values and are
	// echoed by GET /health.
	Version string
	Commit  string
	// BasePath is the validated mount point from the config package.
	BasePath string
	// TrustProxy enables X-Forwarded-* handling.
	TrustProxy bool
	Logger     *slog.Logger
	Health     HealthFunc
	// Now is injectable so tests can assert on the health timestamp.
	Now func() time.Time
}

type server struct {
	version  string
	commit   string
	basePath string
	logger   *slog.Logger
	healthFn HealthFunc
	now      func() time.Time
}

// NewHandler builds the whole HTTP handler: middleware, routes and stubs.
func NewHandler(o Options) http.Handler {
	s := &server{
		version:  valueOr(o.Version, "dev"),
		commit:   o.Commit,
		basePath: normaliseBase(o.BasePath),
		logger:   o.Logger,
		healthFn: o.Health,
		now:      o.Now,
	}
	if s.logger == nil {
		s.logger = slog.Default()
	}
	if s.now == nil {
		s.now = time.Now
	}

	mux := http.NewServeMux()
	s.route(mux, http.MethodGet, "/health", s.health)

	// 0.1 endpoints. The contract (paths, request and response bodies) is fixed
	// by the desktop side; the implementations arrive in their own work items,
	// so until then the route exists and answers 501 instead of 404. That keeps
	// "not built yet" distinguishable from "wrong URL" while the client is
	// being written against this server.
	for _, p := range []string{"/auth/login", "/auth/whoami", "/sync/push", "/sync/pull", "/sync/bootstrap"} {
		s.route(mux, http.MethodPost, p, s.notImplemented)
	}

	// Catch-all so unknown paths get the same JSON error shape as everything
	// else instead of net/http's plain-text 404.
	mux.HandleFunc(s.prefix("/"), s.notFound)

	// Outermost first: a panic anywhere below still produces a logged 500, and
	// the access log sees the status that the body limit may have written.
	var h http.Handler = mux
	h = bodyLimit(MaxBodyBytes, h)
	h = accessLog(s.logger, h)
	h = realIP(o.TrustProxy, h)
	h = requestID(h)
	h = recoverer(s.logger, h)
	return h
}

// route registers a handler for one method and, on the same path, a 405 for
// every other method.
//
// The method-less pattern is needed because of the catch-all below: without it
// a wrong-method request matches "/" and answers 404, hiding the fact that the
// path exists. net/http picks the method-specific pattern first, so the extra
// registration only ever handles the mismatch.
func (s *server) route(mux *http.ServeMux, method, path string, h http.HandlerFunc) {
	mux.HandleFunc(method+" "+s.prefix(path), h)
	mux.HandleFunc(s.prefix(path), func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Allow", method)
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use "+method+" on this endpoint")
	})
}

func (s *server) notImplemented(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "this endpoint is not available in this build")
}

func (s *server) notFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "not_found", "no such endpoint")
}

// pattern renders a net/http route pattern below the base path.
func (s *server) pattern(method, path string) string {
	return method + " " + s.prefix(path)
}

func (s *server) prefix(path string) string {
	if s.basePath == "/" {
		return path
	}
	return s.basePath + path
}

func normaliseBase(base string) string {
	base = strings.TrimSpace(base)
	if base == "" || base == "/" {
		return "/"
	}
	return "/" + strings.Trim(base, "/")
}

func valueOr(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
