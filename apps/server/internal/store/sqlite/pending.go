package sqlite

import (
	"context"
	"fmt"

	"github.com/team-croffle/croffle/apps/server/internal/store"
)

// Operations whose implementation belongs to a later work item. They exist so
// the compile-time store.Store check in sqlite.go holds from the moment the
// boundary is defined, and so a caller that arrives early fails loudly instead
// of reading an empty result.
//
// Each one is replaced in place by its work item; this file disappears when the
// last of them lands.

func (db *DB) PullSince(_ context.Context, _ store.PullQuery) (store.PullResult, error) {
	return store.PullResult{}, fmt.Errorf("%w: PullSince (work item server-0.1.0#6)", store.ErrNotImplemented)
}

func (db *DB) Snapshot(_ context.Context, _ store.SnapshotQuery) (store.SnapshotResult, error) {
	return store.SnapshotResult{}, fmt.Errorf("%w: Snapshot (work item server-0.1.0#7)", store.ErrNotImplemented)
}

func (db *DB) Sweep(_ context.Context, _ store.SweepConfig) (store.SweepResult, error) {
	return store.SweepResult{}, fmt.Errorf("%w: Sweep (work item server-0.1.0#8)", store.ErrNotImplemented)
}
