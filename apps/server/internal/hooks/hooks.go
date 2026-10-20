// Package hooks is the server's extension point.
//
// There is no plugin system and there will not be one in 0.1: this is a
// self-hosted single binary, and hosting third-party code inside it would mean
// a sandbox, a lifecycle and a security surface that the product does not need
// yet. "Extensible" here means the server tells something else what happened.
//
// Hooks observe. They cannot change or reject a commit, by signature: a hook
// that could rewrite a changeset would put the LWW, version and sequence
// invariants in the hands of code the server does not control, which is a
// data-corruption path rather than a feature.
package hooks

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"
)

// Event types. The names are part of the webhook contract, so they are stable
// strings rather than derived from Go type names.
const (
	TypeCommitsApplied   = "commits_applied"
	TypeRowShared        = "row_shared"
	TypeDeviceRegistered = "device_registered"
	TypeAuthSucceeded    = "auth_succeeded"
	TypeRetentionSwept   = "retention_swept"
)

// AllTypes is every event a build can emit.
var AllTypes = []string{
	TypeCommitsApplied, TypeRowShared, TypeDeviceRegistered,
	TypeAuthSucceeded, TypeRetentionSwept,
}

// DefaultTypes is what a webhook receives unless the operator narrows it: the
// two events that say something about calendar data moving.
var DefaultTypes = []string{TypeCommitsApplied, TypeRowShared}

// Event is one notification.
//
// Data never carries a row payload, a token or a password: an event says that
// something changed and which row it was, not what the appointment says.
type Event struct {
	Type string `json:"type"`
	// At is unix milliseconds.
	At   int64 `json:"at"`
	Data any   `json:"data"`
}

// CommitsApplied is emitted once per accepted push batch.
type CommitsApplied struct {
	AccountID     string       `json:"accountId"`
	DeviceID      string       `json:"deviceId"`
	ServerSeqFrom int64        `json:"serverSeqFrom"`
	ServerSeqTo   int64        `json:"serverSeqTo"`
	Changes       []ChangedRow `json:"changes"`
	Conflicts     int          `json:"conflicts"`
}

// ChangedRow identifies a row that moved, without describing it.
type ChangedRow struct {
	Table     string  `json:"table"`
	RowID     string  `json:"rowId"`
	Op        string  `json:"op"`
	Version   int64   `json:"version"`
	ServerSeq int64   `json:"serverSeq"`
	TeamID    *string `json:"teamId,omitempty"`
}

// RowShared is emitted when a private row gains a team.
//
// It exists because it is the one change whose meaning the server understands:
// everything else is an opaque payload, but "this became visible to a team" is
// a fact worth acting on.
type RowShared struct {
	AccountID string `json:"accountId"`
	Table     string `json:"table"`
	RowID     string `json:"rowId"`
	TeamID    string `json:"teamId"`
	ServerSeq int64  `json:"serverSeq"`
}

// DeviceRegistered is emitted when a login creates or re-arms a device.
type DeviceRegistered struct {
	AccountID string `json:"accountId"`
	DeviceID  string `json:"deviceId"`
	Name      string `json:"name"`
}

// AuthSucceeded is emitted for a successful credential check.
type AuthSucceeded struct {
	AccountID string `json:"accountId"`
	DeviceID  string `json:"deviceId,omitempty"`
	// Method is "password" or "token".
	Method string `json:"method"`
	// Surface is "sync" or "dashboard".
	Surface string `json:"surface"`
}

// RetentionSwept is emitted after a sweep that removed something.
type RetentionSwept struct {
	DeletedCommits  int64 `json:"deletedCommits"`
	DeletedOps      int64 `json:"deletedOps"`
	DeletedSessions int64 `json:"deletedSessions"`
	OldestCommitSeq int64 `json:"oldestCommitSeq"`
}

// Emitter is what the rest of the server calls. The signature has no error and
// no return value on purpose: a caller must not be able to wait on a hook, and
// a hook must not be able to fail a request.
type Emitter interface {
	Emit(ctx context.Context, event Event)
}

// Sink delivers events somewhere. Implementations are expected to be slow and
// unreliable; the dispatcher insulates the request path from them.
type Sink interface {
	// Name appears in logs.
	Name() string
	Deliver(ctx context.Context, event Event) error
}

// Nop is the emitter a build without hooks uses.
type Nop struct{}

func (Nop) Emit(context.Context, Event) {}

// queueSize is how many events may wait for delivery. Past that the oldest are
// dropped: a slow webhook must never slow down a sync, and events are
// notifications, not a log.
const queueSize = 256

// Dispatcher fans events out to sinks on its own goroutine.
type Dispatcher struct {
	sinks  []Sink
	logger *slog.Logger
	queue  chan Event
	now    func() time.Time

	wg sync.WaitGroup

	mu      sync.Mutex
	dropped int64
	closed  bool
}

// NewDispatcher starts the worker. Close stops it.
func NewDispatcher(logger *slog.Logger, sinks ...Sink) *Dispatcher {
	if logger == nil {
		logger = slog.Default()
	}
	d := &Dispatcher{
		sinks:  sinks,
		logger: logger,
		queue:  make(chan Event, queueSize),
		now:    time.Now,
	}

	d.wg.Add(1)
	go d.run()
	return d
}

// Emit queues an event. It never blocks.
func (d *Dispatcher) Emit(_ context.Context, event Event) {
	if len(d.sinks) == 0 {
		return
	}
	if event.At == 0 {
		event.At = d.now().UnixMilli()
	}

	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return
	}
	d.mu.Unlock()

	select {
	case d.queue <- event:
	default:
		// Dropping the newest rather than blocking: the alternative is holding
		// up the push that produced it.
		d.mu.Lock()
		d.dropped++
		dropped := d.dropped
		d.mu.Unlock()
		d.logger.Warn("hook event dropped: queue full",
			slog.String("type", event.Type), slog.Int64("dropped_total", dropped))
	}
}

// Dropped reports how many events were discarded, for tests and for /health if
// it ever wants it.
func (d *Dispatcher) Dropped() int64 {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.dropped
}

// Close drains what is queued and stops the worker.
func (d *Dispatcher) Close() {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return
	}
	d.closed = true
	d.mu.Unlock()

	close(d.queue)
	d.wg.Wait()
}

func (d *Dispatcher) run() {
	defer d.wg.Done()

	for event := range d.queue {
		for _, sink := range d.sinks {
			// Each delivery gets its own deadline; one unreachable sink must not
			// stall the others or the queue behind them.
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := sink.Deliver(ctx, event); err != nil {
				d.logger.Warn("hook delivery failed",
					slog.String("sink", sink.Name()),
					slog.String("type", event.Type),
					slog.String("error", err.Error()))
			}
			cancel()
		}
	}
}

// Marshal renders an event as the JSON a sink sends. Exported so a sink and its
// tests agree on the bytes that get signed.
func Marshal(event Event) ([]byte, error) {
	return json.Marshal(event)
}
