package hooks

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"
)

type recordingSink struct {
	mu     sync.Mutex
	events []Event
	err    error
	block  chan struct{}
}

func (r *recordingSink) Name() string { return "recording" }

func (r *recordingSink) Deliver(_ context.Context, event Event) error {
	if r.block != nil {
		<-r.block
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
	return r.err
}

func (r *recordingSink) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.events)
}

func quiet() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError + 1}))
}

func TestDispatcherDeliversAndStampsTime(t *testing.T) {
	sink := &recordingSink{}
	d := NewDispatcher(quiet(), sink)

	d.Emit(context.Background(), Event{Type: TypeRowShared, Data: RowShared{RowID: "row-1"}})
	d.Close()

	if sink.count() != 1 {
		t.Fatalf("delivered = %d, want 1", sink.count())
	}
	if sink.events[0].At == 0 {
		t.Error("the event carries no timestamp")
	}
}

// A hook must not be able to hold up the request that produced it.
func TestEmitDoesNotBlockOnASlowSink(t *testing.T) {
	sink := &recordingSink{block: make(chan struct{})}
	d := NewDispatcher(quiet(), sink)

	done := make(chan struct{})
	go func() {
		for range queueSize + 50 {
			d.Emit(context.Background(), Event{Type: TypeCommitsApplied})
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Emit blocked while the sink was stuck")
	}

	// Past the queue the newest are dropped rather than queued without bound.
	if d.Dropped() == 0 {
		t.Error("nothing was dropped although the queue was overrun")
	}

	close(sink.block)
	d.Close()
}

// A failing sink is logged, not propagated: there is nothing a caller could do
// with the error, and a hook is not part of the request's result.
func TestDeliveryFailureIsSwallowed(t *testing.T) {
	sink := &recordingSink{err: errors.New("receiver down")}
	d := NewDispatcher(quiet(), sink)

	d.Emit(context.Background(), Event{Type: TypeAuthSucceeded})
	d.Close()

	if sink.count() != 1 {
		t.Errorf("attempts = %d, want the delivery to have been tried", sink.count())
	}
}

func TestEmitAfterCloseIsIgnored(t *testing.T) {
	sink := &recordingSink{}
	d := NewDispatcher(quiet(), sink)
	d.Close()

	// Must not panic on a closed channel.
	d.Emit(context.Background(), Event{Type: TypeRowShared})
	if sink.count() != 0 {
		t.Errorf("delivered = %d after Close, want 0", sink.count())
	}
}

func TestDispatcherWithoutSinksDoesNothing(t *testing.T) {
	d := NewDispatcher(quiet())
	d.Emit(context.Background(), Event{Type: TypeRowShared})
	d.Close()

	if d.Dropped() != 0 {
		t.Errorf("dropped = %d, want 0", d.Dropped())
	}
}

func TestNopEmitter(t *testing.T) {
	// The build-without-hooks path has to be safe to call.
	Nop{}.Emit(context.Background(), Event{Type: TypeRowShared})
}

// Hooks observe. If this ever compiles with a return value, the invariants of
// the commit log are no longer the server's alone.
func TestEmitterSignatureCannotAffectTheCaller(t *testing.T) {
	var emitter Emitter = Nop{}
	// Compile-time: Emit returns nothing, so no caller can branch on a hook.
	emitter.Emit(context.Background(), Event{Type: TypeCommitsApplied})
}
