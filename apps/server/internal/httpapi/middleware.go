package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
)

// HeaderRequestID is echoed on every response so a client log line and a server
// log line can be matched up.
const HeaderRequestID = "X-Request-Id"

type ctxKey int

const (
	ctxKeyRequestID ctxKey = iota
	ctxKeyRemoteIP
)

// RequestID returns the id assigned to r, or "" outside the middleware chain.
func RequestID(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyRequestID).(string)
	return v
}

// RemoteIP returns the caller address as resolved by the realIP middleware:
// X-Forwarded-For when TrustProxy is on, the socket address otherwise.
func RemoteIP(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyRemoteIP).(string)
	return v
}

// statusWriter records what the handler sent so the access log can report it.
type statusWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *statusWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
		w.ResponseWriter.WriteHeader(status)
	}
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

func (w *statusWriter) statusOrOK() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func recoverer(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				// http.ErrAbortHandler is the documented way to drop a
				// connection on purpose; do not turn it into a 500.
				if rec == http.ErrAbortHandler {
					panic(rec)
				}
				logger.ErrorContext(r.Context(), "panic in handler",
					slog.Any("panic", rec),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.String("request_id", RequestID(r.Context())),
				)
				writeError(w, http.StatusInternalServerError, "internal", "unexpected server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf [8]byte
		// rand.Read never fails on supported platforms (it panics instead).
		_, _ = rand.Read(buf[:])
		id := hex.EncodeToString(buf[:])

		w.Header().Set(HeaderRequestID, id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKeyRequestID, id)))
	})
}

// realIP resolves the caller address. Without trustProxy the forwarded headers
// are ignored entirely: any client can set them, so trusting them unconditionally
// would let a caller forge its own address in logs and rate limits.
func realIP(trustProxy bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := hostOnly(r.RemoteAddr)
		if trustProxy {
			if fwd := firstForwardedFor(r.Header.Get("X-Forwarded-For")); fwd != "" {
				ip = fwd
			}
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKeyRemoteIP, ip)))
	})
}

func hostOnly(addr string) string {
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return addr
}

// firstForwardedFor takes the left-most entry, i.e. the original client as seen
// by the outermost proxy.
func firstForwardedFor(value string) string {
	first, _, _ := strings.Cut(value, ",")
	first = strings.TrimSpace(first)
	if first == "" {
		return ""
	}
	// Some proxies send "[::1]:1234" or quoted IPv6 forms.
	if host, _, err := net.SplitHostPort(first); err == nil {
		return host
	}
	return strings.Trim(first, "[]\"")
}

// bodyLimit rejects oversized requests. Content-Length is checked first so a
// large upload is refused before it is read; the MaxBytesReader covers chunked
// requests, which only fail once a handler reads past the limit (see DecodeJSON).
func bodyLimit(limit int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > limit {
			writeError(w, http.StatusRequestEntityTooLarge, "body_too_large", "request body exceeds the limit")
			return
		}
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, limit)
		}
		next.ServeHTTP(w, r)
	})
}

// accessLog logs one line per request. Deliberately absent: headers (they carry
// the bearer token) and bodies (they carry calendar data).
func accessLog(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w}

		next.ServeHTTP(sw, r)

		level := slog.LevelInfo
		status := sw.statusOrOK()
		switch {
		// 501 is this build answering "not implemented yet" on a planned
		// endpoint, not a failure: logging it at error level would make a
		// half-built server look broken. Remove the case once every endpoint
		// has a real handler.
		case status == http.StatusNotImplemented:
			level = slog.LevelWarn
		case status >= 500:
			level = slog.LevelError
		case status >= 400:
			level = slog.LevelWarn
		}

		logger.Log(r.Context(), level, "request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", status),
			slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			slog.Int("bytes", sw.bytes),
			slog.String("remote_ip", RemoteIP(r.Context())),
			slog.String("request_id", RequestID(r.Context())),
		)
	})
}
