package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strconv"

	"github.com/team-croffle/croffle/apps/server/internal/auth"
	"github.com/team-croffle/croffle/apps/server/internal/protocol"
	"github.com/team-croffle/croffle/apps/server/internal/store"
)

// Syncer is the slice of the store the sync endpoints need. It grows as the
// handlers arrive (pull in #6, bootstrap in #7).
type Syncer interface {
	ApplyCommits(ctx context.Context, req store.ApplyRequest) (store.ApplyResult, error)
	PullSince(ctx context.Context, query store.PullQuery) (store.PullResult, error)
}

func (s *server) push(w http.ResponseWriter, r *http.Request) {
	if s.sync == nil {
		s.notImplemented(w, r)
		return
	}
	identity, ok := IdentityOf(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	var req protocol.PushRequest
	if !DecodeJSON(w, r, &req) {
		return
	}

	if len(req.Commits) > protocol.MaxCommitsPerPush {
		writeError(w, http.StatusBadRequest, "too_many_commits",
			"a push carries at most "+strconv.Itoa(protocol.MaxCommitsPerPush)+" commits")
		return
	}

	commits := make([]store.Commit, 0, len(req.Commits))
	for i, wire := range req.Commits {
		commit, err := toStoreCommit(wire, identity)
		if err != nil {
			s.writeSyncError(w, r, err, slog.Int("commit_index", i))
			return
		}
		commits = append(commits, commit)
	}

	// The body's clientId is informational: the token decides which device this
	// is. They legitimately differ when the token came from `token issue`,
	// where the server generated the device id before the desktop had one.
	result, err := s.sync.ApplyCommits(r.Context(), store.ApplyRequest{
		AccountID: identity.AccountID,
		DeviceID:  identity.DeviceID,
		TeamIDs:   identity.TeamIDs,
		Commits:   commits,
		Now:       s.now(),
	})
	if err != nil {
		s.writeSyncError(w, r, err)
		return
	}

	conflicts := make([]protocol.Conflict, 0, len(result.Conflicts))
	for _, conflict := range result.Conflicts {
		mapped, err := toWireConflict(conflict)
		if err != nil {
			s.writeSyncError(w, r, err)
			return
		}
		conflicts = append(conflicts, mapped)
	}

	s.logger.InfoContext(r.Context(), "push applied",
		slog.String("account_id", identity.AccountID),
		slog.String("device_id", identity.DeviceID),
		slog.Int("commits", len(commits)),
		slog.Int("accepted", len(result.Accepted)),
		slog.Int("conflicts", len(conflicts)),
		slog.Int64("server_seq", result.ServerSeq),
		slog.String("request_id", RequestID(r.Context())),
	)

	writeJSON(w, http.StatusOK, protocol.PushResponse{
		Accepted:  result.Accepted,
		Conflicts: conflicts,
		ServerSeq: result.ServerSeq,
	})
}

// toStoreCommit validates one wire commit and splits its row into the promoted
// columns and the opaque payload.
func toStoreCommit(wire protocol.Commit, identity auth.Identity) (store.Commit, error) {
	if err := protocol.ValidateCommit(wire); err != nil {
		return store.Commit{}, err
	}

	teamID, err := protocol.TeamIDOf(wire.Row)
	if err != nil {
		return store.Commit{}, errors.Join(protocol.ErrInvalidCommit, err)
	}
	if teamID != nil && !slices.Contains(identity.TeamIDs, *teamID) {
		// Sharing into a team the account is not in would make the row visible
		// to strangers.
		return store.Commit{}, store.ErrForeignTeam
	}

	deletedAt, err := protocol.DeletedAtOf(wire.Row)
	if err != nil {
		return store.Commit{}, errors.Join(protocol.ErrInvalidCommit, err)
	}

	payload, err := protocol.SplitRow(wire.Row)
	if err != nil {
		return store.Commit{}, err
	}

	return store.Commit{
		OpID:         wire.OpID,
		Table:        wire.Table,
		RowID:        wire.RowID,
		Op:           store.Op(wire.Op),
		BaseVersion:  wire.BaseVersion,
		Version:      wire.Version,
		UpdatedAtMs:  wire.UpdatedAtMs,
		LastWriterID: wire.LastWriterID,
		TeamID:       teamID,
		DeletedAt:    deletedAt,
		Payload:      payload,
	}, nil
}

func toWireConflict(conflict store.Conflict) (protocol.Conflict, error) {
	out := protocol.Conflict{
		OpID:  conflict.OpID,
		Table: conflict.Table,
		RowID: conflict.RowID,
	}
	if conflict.ServerRow == nil {
		return out, nil
	}

	row := conflict.ServerRow
	wireRow, err := protocol.MergeRow(row.Payload, row.Version, row.UpdatedAtMs,
		row.LastWriterID, row.OwnerID, row.TeamID, row.DeletedAt)
	if err != nil {
		return protocol.Conflict{}, err
	}

	out.ServerRow = wireRow
	out.ServerVersion = row.Version
	out.ServerUpdatedAtMs = row.UpdatedAtMs
	out.ServerRowWasDeleted = row.Deleted()
	if row.LastWriterID != nil {
		out.ServerLastWriterID = *row.LastWriterID
	}
	return out, nil
}

func (s *server) writeSyncError(w http.ResponseWriter, r *http.Request, err error, extra ...slog.Attr) {
	switch {
	case errors.Is(err, protocol.ErrInvalidCommit):
		writeError(w, http.StatusBadRequest, "invalid_commit", err.Error())

	case errors.Is(err, store.ErrForeignTeam):
		writeError(w, http.StatusForbidden, "foreign_team",
			"this account is not a member of that team")

	case errors.Is(err, store.ErrCursorTooOld):
		// Retrying the same cursor would fail forever; the client has to
		// bootstrap from a snapshot instead.
		writeError(w, http.StatusGone, "cursor_too_old",
			"the changes after this cursor are no longer retained; bootstrap instead")

	case errors.Is(err, store.ErrNotImplemented):
		writeError(w, http.StatusNotImplemented, "not_implemented",
			"this endpoint is not available in this build")

	default:
		attrs := append([]slog.Attr{
			slog.String("error", err.Error()),
			slog.String("request_id", RequestID(r.Context())),
		}, extra...)
		s.logger.LogAttrs(r.Context(), slog.LevelError, "sync failed", attrs...)
		writeError(w, http.StatusInternalServerError, "internal", "unexpected server error")
	}
}

func (s *server) pull(w http.ResponseWriter, r *http.Request) {
	if s.sync == nil {
		s.notImplemented(w, r)
		return
	}
	identity, ok := IdentityOf(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	var req protocol.PullRequest
	if !DecodeJSON(w, r, &req) {
		return
	}
	if req.LastServerSeq < 0 {
		writeError(w, http.StatusBadRequest, "invalid_cursor", "lastServerSeq must not be negative")
		return
	}

	result, err := s.sync.PullSince(r.Context(), store.PullQuery{
		AccountID: identity.AccountID,
		TeamIDs:   identity.TeamIDs,
		SinceSeq:  req.LastServerSeq,
		Limit:     req.Limit,
	})
	if err != nil {
		s.writeSyncError(w, r, err, slog.Int64("cursor", req.LastServerSeq))
		return
	}

	changes := make([]protocol.Change, 0, len(result.Changes))
	for _, change := range result.Changes {
		mapped, err := toWireChange(change)
		if err != nil {
			s.writeSyncError(w, r, err)
			return
		}
		changes = append(changes, mapped)
	}

	s.logger.DebugContext(r.Context(), "pull served",
		slog.String("account_id", identity.AccountID),
		slog.Int64("cursor", req.LastServerSeq),
		slog.Int("changes", len(changes)),
		slog.Bool("has_more", result.HasMore),
		slog.String("request_id", RequestID(r.Context())),
	)

	writeJSON(w, http.StatusOK, protocol.PullResponse{
		Changes:   changes,
		ServerSeq: result.ServerSeq,
		HasMore:   result.HasMore,
	})
}

// toWireChange rebuilds the row a client sent from the promoted columns and the
// opaque payload, so what comes back out has the shape that went in.
func toWireChange(change store.Change) (protocol.Change, error) {
	row, err := protocol.MergeRow(change.Payload, change.Version, change.UpdatedAtMs,
		change.LastWriterID, change.OwnerID, change.TeamID, change.DeletedAt)
	if err != nil {
		return protocol.Change{}, err
	}

	out := protocol.Change{
		ServerSeq:   change.ServerSeq,
		Table:       change.Table,
		RowID:       change.RowID,
		Op:          string(change.Op),
		Row:         row,
		Version:     change.Version,
		UpdatedAtMs: change.UpdatedAtMs,
	}
	if change.LastWriterID != nil {
		out.LastWriterID = *change.LastWriterID
	}
	return out, nil
}
