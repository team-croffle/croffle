package protocol

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Tables the server accepts. Anything else is a client that is ahead of this
// server, and is rejected rather than stored as an unknown table that nothing
// would ever pull.
var Tables = []string{
	"schedule",
	"tag",
	"schedule_tags",
	"schedule_reminder",
	"schedule_exception",
}

// Limits on one push. The desktop batches 100 commits, so these leave room
// while keeping a single request bounded.
// Paging limits live in the store package, which is what pages.
const (
	MaxCommitsPerPush = 500
	MaxIDLength       = 200
)

// Ops a commit may carry.
const (
	OpUpsert = "upsert"
	OpDelete = "delete"
)

// Row is one synchronised row on the wire: a flat map of primitives, exactly
// as the desktop's encodeRow produces it. Dates are unix milliseconds and
// booleans are 0/1, because that is what the client's SQLite columns hold.
type Row map[string]any

// Promoted keys are the columns the server reasons about. They travel as
// explicit commit fields and are therefore removed from the stored payload:
// keeping them in both places would mean two sources of truth for ownership
// and for the LWW comparison.
const (
	FieldVersion      = "version"
	FieldUpdatedAtMs  = "updatedAtMs"
	FieldDeletedAt    = "deletedAt"
	FieldLastWriterID = "lastWriterId"
	FieldOwnerID      = "ownerId"
	FieldTeamID       = "teamId"
)

var promotedFields = []string{
	FieldVersion, FieldUpdatedAtMs, FieldDeletedAt,
	FieldLastWriterID, FieldOwnerID, FieldTeamID,
}

// Commit is one change in a push.
type Commit struct {
	// OpID is the idempotency key. A retry must reuse it, which is the only
	// thing that stops a change being applied twice when a response is lost.
	OpID  string `json:"opId"`
	Table string `json:"table"`
	RowID string `json:"rowId"`
	Op    string `json:"op"`
	// Row is the full row. Nil is allowed for a delete whose tombstone fields
	// are carried by the commit itself.
	Row Row `json:"row,omitempty"`
	// BaseVersion is the version the client believes the server holds; 0 means
	// "this row is new".
	BaseVersion int64 `json:"baseVersion"`
	// Version must be greater than BaseVersion: row versions have to increase
	// for the conflict check to mean anything. A client that wins a conflict
	// therefore has to lift its own version above the server's before retrying.
	Version      int64  `json:"version"`
	UpdatedAtMs  int64  `json:"updatedAtMs"`
	LastWriterID string `json:"lastWriterId"`
}

// PushRequest is POST /sync/push.
type PushRequest struct {
	ClientID string   `json:"clientId"`
	Commits  []Commit `json:"commits"`
}

// Conflict reports a commit that was not applied because the server row had
// moved on. The server row rides along so the client can settle it with the
// same last-writer-wins rule both sides implement.
type Conflict struct {
	OpID                string `json:"opId"`
	Table               string `json:"table"`
	RowID               string `json:"rowId"`
	ServerRow           Row    `json:"serverRow,omitempty"`
	ServerVersion       int64  `json:"serverVersion"`
	ServerUpdatedAtMs   int64  `json:"serverUpdatedAtMs"`
	ServerLastWriterID  string `json:"serverLastWriterId,omitempty"`
	ServerRowWasDeleted bool   `json:"serverRowDeleted,omitempty"`
}

// PushResponse is the push result. A batch is routinely part accepted and part
// conflicted, which is why this is a 200 and not an error status.
type PushResponse struct {
	Accepted  []string   `json:"accepted"`
	Conflicts []Conflict `json:"conflicts"`
	// ServerSeq is the highest sequence this batch produced, or the current
	// highest when it produced none.
	ServerSeq int64 `json:"serverSeq"`
}

// Change is one entry of the commit log on the wire.
type Change struct {
	ServerSeq    int64  `json:"serverSeq"`
	Table        string `json:"table"`
	RowID        string `json:"rowId"`
	Op           string `json:"op"`
	Row          Row    `json:"row,omitempty"`
	Version      int64  `json:"version"`
	UpdatedAtMs  int64  `json:"updatedAtMs"`
	LastWriterID string `json:"lastWriterId,omitempty"`
}

// PullRequest is POST /sync/pull.
type PullRequest struct {
	LastServerSeq int64 `json:"lastServerSeq"`
	Limit         int   `json:"limit,omitempty"`
}

// PullResponse carries changes plus the cursor to store after applying them.
type PullResponse struct {
	Changes []Change `json:"changes"`
	// ServerSeq is the sequence of the last change in this response - never a
	// value the client has not seen, because a cursor moved past an unreturned
	// commit would lose it permanently.
	ServerSeq int64 `json:"serverSeq"`
	HasMore   bool  `json:"hasMore"`
}

// BootstrapRequest asks for current state instead of history. It is what a new
// device calls, and what a device falls back to when its cursor is older than
// the retention window.
type BootstrapRequest struct {
	// Cursor is opaque: it pins the snapshot instant and the page position.
	// Pass back what the previous page returned.
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// BootstrapResponse is one page of the snapshot.
type BootstrapResponse struct {
	Rows []Change `json:"rows"`
	// ServerSeq is the instant the snapshot was taken and is identical on every
	// page. Store it as the cursor once the last page has been applied.
	ServerSeq int64  `json:"serverSeq"`
	Cursor    string `json:"cursor,omitempty"`
	HasMore   bool   `json:"hasMore"`
}

// SplitRow separates a wire row into the promoted values the server stores in
// columns and the opaque payload it keeps as JSON.
func SplitRow(row Row) (payload json.RawMessage, err error) {
	rest := make(Row, len(row))
	for key, value := range row {
		if slices.Contains(promotedFields, key) {
			continue
		}
		rest[key] = value
	}

	encoded, err := json.Marshal(rest)
	if err != nil {
		return nil, fmt.Errorf("encode row payload: %w", err)
	}
	return encoded, nil
}

// MergeRow rebuilds the wire row from the stored payload and the promoted
// columns, so a client receives the same shape it sent.
func MergeRow(payload json.RawMessage, version, updatedAtMs int64, lastWriterID *string, ownerID string, teamID *string, deletedAt *int64) (Row, error) {
	row := Row{}
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &row); err != nil {
			return nil, fmt.Errorf("decode row payload: %w", err)
		}
	}

	row[FieldVersion] = version
	row[FieldUpdatedAtMs] = updatedAtMs
	row[FieldOwnerID] = ownerID
	row[FieldDeletedAt] = nil
	if deletedAt != nil {
		row[FieldDeletedAt] = *deletedAt
	}
	row[FieldLastWriterID] = nil
	if lastWriterID != nil {
		row[FieldLastWriterID] = *lastWriterID
	}
	row[FieldTeamID] = nil
	if teamID != nil {
		row[FieldTeamID] = *teamID
	}
	return row, nil
}

// TeamIDOf reads the shared-team field of a wire row. A missing or null value
// means the row is private.
func TeamIDOf(row Row) (*string, error) {
	value, ok := row[FieldTeamID]
	if !ok || value == nil {
		return nil, nil
	}
	team, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("row field %s must be a string", FieldTeamID)
	}
	team = strings.TrimSpace(team)
	if team == "" {
		return nil, nil
	}
	return &team, nil
}

// DeletedAtOf reads the tombstone field of a wire row.
func DeletedAtOf(row Row) (*int64, error) {
	value, ok := row[FieldDeletedAt]
	if !ok || value == nil {
		return nil, nil
	}
	// JSON numbers decode to float64; milliseconds are well inside the range
	// where that is exact.
	switch typed := value.(type) {
	case float64:
		ms := int64(typed)
		return &ms, nil
	case int64:
		return &typed, nil
	default:
		return nil, fmt.Errorf("row field %s must be a number", FieldDeletedAt)
	}
}

// ErrInvalidCommit marks a commit the server will not store. The caller turns
// it into a 400: it is a client bug, not a conflict.
var ErrInvalidCommit = errors.New("invalid commit")

// ValidateCommit checks one commit's shape.
func ValidateCommit(c Commit) error {
	switch {
	case strings.TrimSpace(c.OpID) == "":
		return fmt.Errorf("%w: opId is required", ErrInvalidCommit)
	case len(c.OpID) > MaxIDLength:
		return fmt.Errorf("%w: opId is too long", ErrInvalidCommit)
	case strings.TrimSpace(c.RowID) == "":
		return fmt.Errorf("%w: rowId is required", ErrInvalidCommit)
	case len(c.RowID) > MaxIDLength:
		return fmt.Errorf("%w: rowId is too long", ErrInvalidCommit)
	case !slices.Contains(Tables, c.Table):
		return fmt.Errorf("%w: unknown table %q", ErrInvalidCommit, c.Table)
	case c.Op != OpUpsert && c.Op != OpDelete:
		return fmt.Errorf("%w: op must be %q or %q", ErrInvalidCommit, OpUpsert, OpDelete)
	case c.BaseVersion < 0:
		return fmt.Errorf("%w: baseVersion must not be negative", ErrInvalidCommit)
	case c.Version <= 0:
		return fmt.Errorf("%w: version must be positive", ErrInvalidCommit)
	case c.Version <= c.BaseVersion:
		// Row versions have to increase, otherwise baseVersion comparison stops
		// detecting anything. A client that wins a conflict must lift its
		// version above the server's before pushing again.
		return fmt.Errorf("%w: version (%d) must be greater than baseVersion (%d)",
			ErrInvalidCommit, c.Version, c.BaseVersion)
	case c.UpdatedAtMs <= 0:
		return fmt.Errorf("%w: updatedAtMs is required", ErrInvalidCommit)
	case c.Op == OpUpsert && c.Row == nil:
		return fmt.Errorf("%w: an upsert needs a row", ErrInvalidCommit)
	}
	return nil
}

// BootstrapCursor is the opaque cursor of a snapshot page. It travels as one
// base64 string so a client never has to understand or assemble it; the only
// contract is "send back what you were given".
type BootstrapCursor struct {
	// BaseSeq pins the instant. Every page of one snapshot carries the same
	// value, which is what keeps the pages consistent with each other.
	BaseSeq int64 `json:"s"`
	// Table and RowID are the keyset position: the last row of the page.
	Table string `json:"t,omitempty"`
	RowID string `json:"r,omitempty"`
}

// ErrInvalidCursor marks a cursor this server did not produce.
var ErrInvalidCursor = errors.New("invalid cursor")

// EncodeBootstrapCursor renders a cursor for the next page.
func EncodeBootstrapCursor(cursor BootstrapCursor) (string, error) {
	data, err := json.Marshal(cursor)
	if err != nil {
		return "", fmt.Errorf("encode cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

// DecodeBootstrapCursor reads a cursor a client sent back. An empty string is
// the first page.
func DecodeBootstrapCursor(encoded string) (BootstrapCursor, error) {
	if strings.TrimSpace(encoded) == "" {
		return BootstrapCursor{}, nil
	}

	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return BootstrapCursor{}, fmt.Errorf("%w: not base64", ErrInvalidCursor)
	}

	var cursor BootstrapCursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return BootstrapCursor{}, fmt.Errorf("%w: not a cursor", ErrInvalidCursor)
	}
	if cursor.BaseSeq < 0 {
		return BootstrapCursor{}, fmt.Errorf("%w: negative base", ErrInvalidCursor)
	}
	return cursor, nil
}
