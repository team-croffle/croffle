// Package metrics exposes Prometheus counters for operators.
//
// It is off unless CROFFLE_METRICS_LISTEN is set, and then served on that
// address only, never on the API listener: the numbers say how the server is
// used, which is nothing a sync client or the internet needs to read.
//
// No label carries an account, a username, a device or an address. Routes are
// labelled by their mux pattern ("POST /dash/devices/{id}/revoke"), so a path
// value never becomes a label either.
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics owns a private registry, so nothing registered globally by a
// dependency ends up on the endpoint by accident.
type Metrics struct {
	registry *prometheus.Registry

	requests      *prometheus.CounterVec
	duration      *prometheus.HistogramVec
	pushCommits   *prometheus.CounterVec
	pullChanges   prometheus.Counter
	bootstrapRows prometheus.Counter
	authAttempts  *prometheus.CounterVec
	refreshReuse  prometheus.Counter
}

// New builds the registry with the server's metrics plus the Go runtime and
// process collectors.
func New() *Metrics {
	m := &Metrics{
		registry: prometheus.NewRegistry(),
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "croffle_http_requests_total",
			Help: "HTTP requests by route pattern, method and status code.",
		}, []string{"route", "method", "code"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "croffle_http_request_duration_seconds",
			Help:    "HTTP request latency by route pattern and method.",
			Buckets: prometheus.DefBuckets,
		}, []string{"route", "method"}),
		pushCommits: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "croffle_sync_push_commits_total",
			Help: "Pushed commits by result: accepted (applied), conflict, retry (an opId already applied).",
		}, []string{"result"}),
		pullChanges: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "croffle_sync_pull_changes_total",
			Help: "Changes handed out by /sync/pull.",
		}),
		bootstrapRows: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "croffle_sync_bootstrap_rows_total",
			Help: "Rows handed out by /sync/bootstrap.",
		}),
		authAttempts: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "croffle_auth_attempts_total",
			Help: "Sign-in attempts by surface (sync, token, dashboard) and result.",
		}, []string{"surface", "result"}),
		refreshReuse: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "croffle_auth_refresh_reuse_total",
			Help: "Spent refresh tokens presented again; each revoked a token family.",
		}),
	}
	m.registry.MustRegister(
		m.requests, m.duration, m.pushCommits, m.pullChanges, m.bootstrapRows,
		m.authAttempts, m.refreshReuse,
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	// Label sets that exist from the start, so a dashboard shows a zero
	// instead of "no data" before the first event.
	for _, result := range []string{"accepted", "conflict", "retry"} {
		m.pushCommits.WithLabelValues(result)
	}
	return m
}

// Handler serves the registry in the Prometheus text format.
func (m *Metrics) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{}))
	return mux
}

func (m *Metrics) ObserveRequest(route, method string, code int, elapsed time.Duration) {
	m.requests.WithLabelValues(route, method, strconv.Itoa(code)).Inc()
	m.duration.WithLabelValues(route, method).Observe(elapsed.Seconds())
}

func (m *Metrics) PushCommits(accepted, conflicts, retries int) {
	m.pushCommits.WithLabelValues("accepted").Add(float64(accepted))
	m.pushCommits.WithLabelValues("conflict").Add(float64(conflicts))
	m.pushCommits.WithLabelValues("retry").Add(float64(retries))
}

func (m *Metrics) PullChanges(n int) { m.pullChanges.Add(float64(n)) }

func (m *Metrics) BootstrapRows(n int) { m.bootstrapRows.Add(float64(n)) }

func (m *Metrics) AuthAttempt(surface, result string) {
	m.authAttempts.WithLabelValues(surface, result).Inc()
}

func (m *Metrics) RefreshReuse() { m.refreshReuse.Inc() }
