package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/team-croffle/croffle/apps/server/internal/auth"
)

// Recorder is what the HTTP layer reports to. internal/metrics implements it;
// nopRecorder stands in when metrics are off.
type Recorder interface {
	ObserveRequest(route, method string, code int, elapsed time.Duration)
	PushCommits(accepted, conflicts, retries int)
	PullChanges(n int)
	BootstrapRows(n int)
	AuthAttempt(surface, result string)
	RefreshReuse()
}

type nopRecorder struct{}

func (nopRecorder) ObserveRequest(string, string, int, time.Duration) {}
func (nopRecorder) PushCommits(int, int, int)                         {}
func (nopRecorder) PullChanges(int)                                   {}
func (nopRecorder) BootstrapRows(int)                                 {}
func (nopRecorder) AuthAttempt(string, string)                        {}
func (nopRecorder) RefreshReuse()                                     {}

// observe records one request under its route pattern. Requests the
// catch-all answers (the dashboard's files, unknown paths) share the label
// "other", so arbitrary paths cannot grow the label set.
func (s *server) observe(next http.Handler) http.Handler {
	if _, off := s.metrics.(nopRecorder); off {
		return next
	}
	catchAll := s.prefix("/")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(sw, r)

		route := r.Pattern
		if route == "" || route == catchAll || strings.HasSuffix(route, " "+catchAll) {
			route = "other"
		}
		s.metrics.ObserveRequest(route, r.Method, sw.statusOrOK(), time.Since(start))
	})
}

// authResult maps a sign-in outcome to the result label.
func authResult(err error) string {
	switch {
	case err == nil:
		return "ok"
	case errors.Is(err, auth.ErrOTPRequired):
		return "otp_required"
	case errors.Is(err, auth.ErrAccountDisabled):
		return "disabled"
	case errors.Is(err, auth.ErrRateLimited):
		return "rate_limited"
	case errors.Is(err, auth.ErrInvalidCredentials), errors.Is(err, auth.ErrRefreshReused),
		errors.Is(err, auth.ErrMissingClientID):
		return "invalid"
	default:
		return "error"
	}
}
