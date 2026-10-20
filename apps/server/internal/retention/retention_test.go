package retention

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/team-croffle/croffle/apps/server/internal/store"
)

type fakeSweeper struct {
	mu     sync.Mutex
	calls  []store.SweepConfig
	result store.SweepResult
	err    error
}

func (f *fakeSweeper) Sweep(_ context.Context, cfg store.SweepConfig) (store.SweepResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, cfg)
	return f.result, f.err
}

func (f *fakeSweeper) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func quiet() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError + 1}))
}

func TestSweepOnceUsesTheConfiguredWindows(t *testing.T) {
	sweeper := &fakeSweeper{result: store.SweepResult{DeletedCommits: 2, OldestCommitSeq: 7}}
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)

	service := New(sweeper, quiet(), 90*24*time.Hour, 30*24*time.Hour)
	service.now = func() time.Time { return now }
	service.SweepOnce(context.Background())

	if sweeper.count() != 1 {
		t.Fatalf("sweeps = %d, want 1", sweeper.count())
	}
	cfg := sweeper.calls[0]
	if want := now.Add(-90 * 24 * time.Hour); !cfg.CommitsBefore.Equal(want) {
		t.Errorf("commitsBefore = %v, want %v", cfg.CommitsBefore, want)
	}
	// The two windows differ on purpose: history can expire long before the
	// idempotency keys may.
	if want := now.Add(-30 * 24 * time.Hour); !cfg.OpsBefore.Equal(want) {
		t.Errorf("opsBefore = %v, want %v", cfg.OpsBefore, want)
	}
}

// Retention falling behind is not a reason to stop serving sync.
func TestSweepOnceSwallowsFailures(t *testing.T) {
	sweeper := &fakeSweeper{err: errors.New("disk on fire")}
	service := New(sweeper, quiet(), 90*24*time.Hour, 30*24*time.Hour)

	service.SweepOnce(context.Background()) // must not panic
	if sweeper.count() != 1 {
		t.Errorf("sweeps = %d, want the attempt to have happened", sweeper.count())
	}
}

// The first sweep happens at startup, not after the first interval: a server
// that restarts often would otherwise never collect its garbage.
func TestRunSweepsImmediatelyAndStopsWithTheContext(t *testing.T) {
	sweeper := &fakeSweeper{}
	service := New(sweeper, quiet(), 90*24*time.Hour, 30*24*time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		service.Run(ctx)
		close(done)
	}()

	deadline := time.After(2 * time.Second)
	for sweeper.count() == 0 {
		select {
		case <-deadline:
			t.Fatal("no sweep happened at startup")
		case <-time.After(time.Millisecond):
		}
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after the context was cancelled")
	}
}
