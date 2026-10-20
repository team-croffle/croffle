// Package store is the persistence boundary of the sync server.
//
// The boundary is cut per operation, not per SQL statement: there is no query
// builder and no dialect abstraction. The only implementation is
// store/sqlite; if a deployment ever needs Postgres, it adds one more
// implementation of this interface rather than rewriting the handlers.
//
// # Row storage
//
// Rows are stored hybrid: the columns the server has to reason about
// (ownership, LWW bookkeeping, tombstone) are real columns, and everything the
// server does not understand — the calendar fields — stays in one opaque JSON
// payload. That is what lets the desktop add a column (teamId in 1.3.1, kind in
// 1.3.2) without a server migration, while the server can still decide what a
// given account is allowed to pull.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// ErrNotImplemented marks an operation whose implementation is scheduled for a
// later work item. It is never returned once 0.1 is complete.
var ErrNotImplemented = errors.New("store: not implemented")

// ErrNotFound is returned by lookups instead of a nil result with a nil error,
// so a caller cannot forget to check.
var ErrNotFound = errors.New("store: not found")

// Op is the kind of change a commit carries. A delete keeps the row and sets
// its tombstone: "deleted" and "never received" have to stay distinguishable
// for a client that is catching up.
type Op string

const (
	OpUpsert Op = "upsert"
	OpDelete Op = "delete"
)

// DefaultTeamID is the single team seeded by the first migration. Server 0.1
// has exactly one team; multi-team arrives in 0.3.
const DefaultTeamID = "team-default"

// Account is a login. Disabling blocks authentication without touching rows.
type Account struct {
	ID           string
	Username     string
	PasswordHash string
	CreatedAt    time.Time
	DisabledAt   *time.Time
}

// Disabled reports whether the account may no longer authenticate.
func (a Account) Disabled() bool { return a.DisabledAt != nil }

// Device is one installation of the desktop app. ID is the client's own
// clientId, which is also what appears in a row's lastWriterId as
// "device:<clientId>", so the two sides agree on who wrote what.
//
// Tokens are per device: losing a laptop revokes that laptop.
type Device struct {
	ID         string
	AccountID  string
	Name       string
	TokenHash  []byte
	CreatedAt  time.Time
	LastSeenAt *time.Time
	RevokedAt  *time.Time
}

// Revoked reports whether the device's token has been withdrawn.
func (d Device) Revoked() bool { return d.RevokedAt != nil }

// Row is the current state of one synchronised row.
type Row struct {
	Table string
	RowID string
	// OwnerID is always the authenticated account: the server overwrites
	// whatever the client sent, otherwise a client could file its rows under
	// someone else's ownership.
	OwnerID string
	// TeamID is set when the row is shared (desktop 1.3.1). Nil means private.
	TeamID      *string
	Version     int64
	UpdatedAtMs int64
	// LastWriterID is "device:<clientId>", "<provider>:<accountId>" or
	// "server". Rows the server applied carry "server" so the client's own
	// outbox triggers stay quiet and no ping-pong starts.
	LastWriterID *string
	// DeletedAt is the tombstone, in unix milliseconds.
	DeletedAt *int64
	// Payload is the opaque remainder of the row.
	Payload json.RawMessage
	// ServerSeq is the commit that last changed this row.
	ServerSeq int64
}

// Deleted reports whether the row is a tombstone.
func (r Row) Deleted() bool { return r.DeletedAt != nil }

// Commit is one change a client wants applied.
type Commit struct {
	// OpID is the idempotency key. A retried push must reuse it; that is the
	// only defence against applying a change twice when a response is lost.
	OpID  string
	Table string
	RowID string
	Op    Op
	// BaseVersion is the version the client believed the server had. A
	// mismatch is a conflict, not an error.
	BaseVersion  int64
	Version      int64
	UpdatedAtMs  int64
	LastWriterID string
	TeamID       *string
	Payload      json.RawMessage
}

// Conflict reports a commit that was not applied because the server row had
// moved on. The server row rides along so the client can settle it with the
// same last-writer-wins rule both sides use.
type Conflict struct {
	OpID      string
	Table     string
	RowID     string
	ServerRow *Row
}

// ApplyRequest is one push batch, already authenticated.
type ApplyRequest struct {
	AccountID string
	DeviceID  string
	// TeamIDs the account belongs to; a commit naming any other team is
	// rejected before it reaches the store.
	TeamIDs []string
	Commits []Commit
	Now     time.Time
}

// ApplyResult mirrors the push response: a batch may be partly accepted and
// partly in conflict, which is normal rather than an error.
type ApplyResult struct {
	Accepted  []string
	Conflicts []Conflict
	ServerSeq int64
}

// PullQuery asks for commits after a cursor, within one account's visibility.
type PullQuery struct {
	AccountID string
	TeamIDs   []string
	SinceSeq  int64
	Limit     int
}

// Change is one entry of the commit log as handed to a client.
type Change struct {
	ServerSeq    int64
	Table        string
	RowID        string
	Op           Op
	Version      int64
	UpdatedAtMs  int64
	LastWriterID *string
	TeamID       *string
	Payload      json.RawMessage
}

// PullResult carries the changes plus the cursor the client should store once
// it has applied them.
type PullResult struct {
	Changes []Change
	// ServerSeq is the sequence of the last change returned, or SinceSeq when
	// there was none. Never a value the client has not actually seen: moving
	// the cursor past an unreturned commit loses it forever.
	ServerSeq int64
	HasMore   bool
}

// SnapshotQuery asks for current state instead of history, for a device that
// is new or whose cursor fell outside the retention window.
type SnapshotQuery struct {
	AccountID string
	TeamIDs   []string
	// BaseSeq pins the snapshot to one instant. The first page reports the
	// instant it chose and every following page must pass it back, otherwise
	// changes made mid-pagination are neither in the snapshot nor after it.
	BaseSeq int64
	// AfterTable and AfterRowID are the keyset cursor within the snapshot.
	AfterTable string
	AfterRowID string
	Limit      int
}

// SnapshotResult is one page of the snapshot.
type SnapshotResult struct {
	Rows      []Row
	ServerSeq int64
	HasMore   bool
}

// SweepConfig is the retention cut-off for one sweep.
type SweepConfig struct {
	CommitsBefore time.Time
	OpsBefore     time.Time
}

// SweepResult reports what a sweep removed. OldestCommitSeq is what decides
// whether a given client cursor is still serviceable.
type SweepResult struct {
	DeletedCommits  int64
	DeletedOps      int64
	OldestCommitSeq int64
}

// Store is the persistence contract.
//
// # serverSeq visibility contract
//
// Every implementation must guarantee that a commit's sequence number is
// allocated inside the same transaction that writes the row, and that no
// reader can observe sequence N+1 before N.
//
// This is not a SQLite detail, it is what makes the cursor safe. If two
// transactions could commit out of order, a reader that saw N+1 would advance
// its cursor past N and never receive it: one appointment silently missing.
// SQLite with a single writer satisfies this for free. A Postgres
// implementation would not — its sequences leave gaps and do not order commits
// — and would have to serialise the append itself, with a counter row bumped
// inside the transaction or an advisory lock.
type Store interface {
	// Ping reports whether the storage is reachable. GET /health calls it.
	Ping(ctx context.Context) error
	Close() error

	// CreateAccount inserts a login whose password is already hashed; the
	// store never handles a plaintext password.
	CreateAccount(ctx context.Context, account Account) error
	// FindAccountByUsername returns ErrNotFound when there is no such account.
	FindAccountByUsername(ctx context.Context, username string) (*Account, error)
	// FindAccountByID is the lookup behind every authenticated request, since
	// a token resolves to a device and a device to its account.
	FindAccountByID(ctx context.Context, id string) (*Account, error)
	// UpsertDevice registers a device or replaces its token.
	//
	// It must refuse to move an existing device row to a different account:
	// device ids come from the client, so without that check a caller could
	// claim someone else's device id and invalidate their token.
	UpsertDevice(ctx context.Context, device Device) error
	// TouchDevice records that a device was seen. Callers throttle it; it is
	// bookkeeping and must not become a write per request.
	TouchDevice(ctx context.Context, deviceID string, at time.Time) error
	// FindDeviceByTokenHash resolves a bearer token to its device. Revoked
	// devices and disabled accounts are the caller's business to reject.
	FindDeviceByTokenHash(ctx context.Context, tokenHash []byte) (*Device, error)
	// TeamsOf lists the teams an account belongs to; it defines what the
	// account can pull beyond its own rows.
	TeamsOf(ctx context.Context, accountID string) ([]string, error)
	// AddTeamMember grants an account access to a team's shared rows.
	AddTeamMember(ctx context.Context, teamID, accountID, role string) error

	// ApplyCommits applies one push batch in a single transaction.
	ApplyCommits(ctx context.Context, req ApplyRequest) (ApplyResult, error)
	// PullSince returns commits after the cursor that the account may see.
	PullSince(ctx context.Context, query PullQuery) (PullResult, error)
	// Snapshot returns current state for a bootstrapping device.
	Snapshot(ctx context.Context, query SnapshotQuery) (SnapshotResult, error)

	// Sweep drops commit log and idempotency entries older than the cut-off.
	Sweep(ctx context.Context, cfg SweepConfig) (SweepResult, error)
}
