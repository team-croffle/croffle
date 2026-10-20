// Package retention keeps the commit log and the idempotency table from growing
// without bound.
//
// Only history expires. The current state (sync_row) is never swept: a snapshot
// is built from it, so dropping a row there would lose data rather than
// reclaim space.
package retention

import (
	"context"
	"log/slog"
	"time"

	"github.com/team-croffle/croffle/apps/server/internal/store"
)

// interval is how often the sweep runs. Retention is measured in months, so
// once a day is frequent enough and keeps the cost invisible.
const interval = 24 * time.Hour

// Sweeper is the slice of the store this package needs.
type Sweeper interface {
	Sweep(ctx context.Context, cfg store.SweepConfig) (store.SweepResult, error)
}

// Service runs the sweep on a schedule.
type Service struct {
	store         Sweeper
	logger        *slog.Logger
	retainCommits time.Duration
	retainOps     time.Duration
	now           func() time.Time
}

func New(st Sweeper, logger *slog.Logger, retainCommits, retainOps time.Duration) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		store:         st,
		logger:        logger,
		retainCommits: retainCommits,
		retainOps:     retainOps,
		now:           time.Now,
	}
}

// Run sweeps once at startup and then daily, until ctx is cancelled.
//
// The first sweep is at startup rather than after the first interval, so a
// server that is restarted often still collects its garbage.
func (s *Service) Run(ctx context.Context) {
	s.SweepOnce(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.SweepOnce(ctx)
		}
	}
}

// SweepOnce runs one sweep and logs the outcome. A failure is logged, not
// propagated: retention falling behind is not a reason to stop serving sync.
func (s *Service) SweepOnce(ctx context.Context) {
	now := s.now()
	started := time.Now()

	result, err := s.store.Sweep(ctx, store.SweepConfig{
		CommitsBefore: now.Add(-s.retainCommits),
		OpsBefore:     now.Add(-s.retainOps),
	})
	if err != nil {
		if ctx.Err() != nil {
			// Shutdown cancelled the sweep; the next start picks it up.
			return
		}
		s.logger.ErrorContext(ctx, "retention sweep failed", slog.String("error", err.Error()))
		return
	}

	if result.DeletedCommits == 0 && result.DeletedOps == 0 {
		s.logger.DebugContext(ctx, "retention sweep found nothing to remove")
		return
	}

	// oldest_commit_seq is what explains a 410 to an operator, so it is logged
	// even though the deletion counts are the headline.
	s.logger.InfoContext(ctx, "retention sweep",
		slog.Int64("deleted_commits", result.DeletedCommits),
		slog.Int64("deleted_ops", result.DeletedOps),
		slog.Int64("oldest_commit_seq", result.OldestCommitSeq),
		slog.Int64("duration_ms", time.Since(started).Milliseconds()),
	)
}
