package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/team-croffle/croffle/apps/server/internal/protocol"
)

// Database states reported by GET /health.
const (
	dbStatusOK = "ok"
	// dbStatusError means the store is configured but unreachable. The endpoint
	// answers 503 so a container HEALTHCHECK and a proxy both notice.
	dbStatusError = "error"
	// dbStatusUnconfigured means no store has been wired yet (work item #2).
	// The server is up and answers metadata, so this is not a failure.
	dbStatusUnconfigured = "unconfigured"
)

type protocolRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

type retentionStatus struct {
	OldestCommitSeq  int64 `json:"oldestCommitSeq"`
	PurgedThroughSeq int64 `json:"purgedThroughSeq"`
}

type healthResponse struct {
	Server   string        `json:"server"`
	Commit   string        `json:"commit,omitempty"`
	Protocol protocolRange `json:"protocol"`
	DB       string        `json:"db"`
	// Retention is what explains a 410 to an operator: a cursor below
	// purgedThroughSeq can no longer be served.
	Retention *retentionStatus `json:"retention,omitempty"`
	// Time is the server clock in unix milliseconds. The desktop compares it
	// with its own to show a clock-skew warning: the LWW comparison key is a
	// client wall clock, so a badly skewed device always wins or always loses.
	Time int64 `json:"time"`
}

// health is public and unauthenticated: it is what a proxy, a HEALTHCHECK and
// the desktop's "test connection" all call before any credential exists.
func (s *server) health(w http.ResponseWriter, r *http.Request) {
	status := http.StatusOK
	dbStatus := dbStatusUnconfigured

	if s.healthFn != nil {
		if err := s.healthFn(r.Context()); err != nil {
			s.logger.ErrorContext(r.Context(), "health check failed",
				slog.String("error", err.Error()),
				slog.String("request_id", RequestID(r.Context())),
			)
			dbStatus = dbStatusError
			status = http.StatusServiceUnavailable
		} else {
			dbStatus = dbStatusOK
		}
	}

	response := healthResponse{
		Server:   s.version,
		Commit:   s.commit,
		Protocol: protocolRange{Min: protocol.Min, Max: protocol.Max},
		DB:       dbStatus,
		Time:     s.now().UnixMilli(),
	}

	if s.retention != nil && dbStatus == dbStatusOK {
		oldest, purged, err := s.retention(r.Context())
		if err != nil {
			// Not fatal: the rest of the answer is still useful, and /health
			// must stay cheap and reliable.
			s.logger.WarnContext(r.Context(), "could not read retention status",
				slog.String("error", err.Error()))
		} else {
			response.Retention = &retentionStatus{OldestCommitSeq: oldest, PurgedThroughSeq: purged}
		}
	}

	writeJSON(w, status, response)
}
