package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strconv"

	"github.com/team-croffle/croffle/apps/server/internal/auth"
	"github.com/team-croffle/croffle/apps/server/internal/hooks"
	"github.com/team-croffle/croffle/apps/server/internal/protocol"
	"github.com/team-croffle/croffle/apps/server/internal/store"
)

// Syncer is the slice of the store the sync endpoints need. It grows as the
// handlers arrive (pull in #6, bootstrap in #7).
type Syncer interface {
	ApplyCommits(ctx context.Context, req store.ApplyRequest) (store.ApplyResult, error)
	PullSince(ctx context.Context, query store.PullQuery) (store.PullResult, error)
	Snapshot(ctx context.Context, query store.SnapshotQuery) (store.SnapshotResult, error)
	UncoveredTeams(ctx context.Context, deviceID string, teamIDs []string) ([]string, error)
	CoverTeams(ctx context.Context, deviceID string, memberships []store.Membership) error
	Memberships(ctx context.Context, accountID string) ([]store.Membership, error)
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

	// What each commit sent, for comparing against the server row. The first
	// commit with an opId wins, as in the store.
	sent := make(map[string]int, len(req.Commits))
	for i, wire := range req.Commits {
		if _, seen := sent[wire.OpID]; !seen {
			sent[wire.OpID] = i
		}
	}

	conflicts := make([]protocol.Conflict, 0, len(result.Conflicts))
	for _, conflict := range result.Conflicts {
		mapped, err := toWireConflict(conflict)
		if err != nil {
			s.writeSyncError(w, r, err)
			return
		}
		if i, ok := sent[conflict.OpID]; ok && mapped.ServerRow != nil {
			mapped.DifferentFields = protocol.DifferentFields(req.Commits[i].Row,
				req.Commits[i].Op == protocol.OpDelete, commits[i].DeletedAt, mapped.ServerRow)
		} else {
			mapped.DifferentFields = []string{}
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

	s.metrics.PushCommits(len(result.Applied), len(result.Conflicts), len(result.Accepted)-len(result.Applied))
	writeJSON(w, http.StatusOK, protocol.PushResponse{
		Accepted:  result.Accepted,
		Conflicts: conflicts,
		ServerSeq: result.ServerSeq,
	})

	// After the response: a hook must not be able to delay a sync, and the
	// dispatcher only queues anyway.
	s.emitApplied(r.Context(), identity, result)
}

// emitApplied turns an apply result into hook events. Payloads are left out on
// purpose: an event says which row moved, not what the appointment says.
func (s *server) emitApplied(ctx context.Context, identity auth.Identity, result store.ApplyResult) {
	if len(result.Applied) == 0 {
		return
	}

	changes := make([]hooks.ChangedRow, 0, len(result.Applied))
	from := result.Applied[0].ServerSeq
	for _, applied := range result.Applied {
		changes = append(changes, hooks.ChangedRow{
			Table: applied.Table, RowID: applied.RowID, Op: string(applied.Op),
			Version: applied.Version, ServerSeq: applied.ServerSeq, TeamID: applied.TeamID,
		})
	}

	s.hooks.Emit(ctx, hooks.Event{Type: hooks.TypeCommitsApplied, Data: hooks.CommitsApplied{
		AccountID:     identity.AccountID,
		DeviceID:      identity.DeviceID,
		ServerSeqFrom: from,
		ServerSeqTo:   result.ServerSeq,
		Changes:       changes,
		Conflicts:     len(result.Conflicts),
	}})

	for _, applied := range result.Applied {
		if !applied.BecameShared || applied.TeamID == nil {
			continue
		}
		// The one change whose meaning the server understands: everything else
		// is an opaque payload, but "this became visible to a team" is a fact
		// worth acting on.
		s.hooks.Emit(ctx, hooks.Event{Type: hooks.TypeRowShared, Data: hooks.RowShared{
			AccountID: identity.AccountID,
			Table:     applied.Table,
			RowID:     applied.RowID,
			TeamID:    *applied.TeamID,
			ServerSeq: applied.ServerSeq,
		}})
	}
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

	// A device whose account joined a team after the device was created has
	// not received the team's older commits. It has to take a snapshot, and
	// cursor_too_old is the answer desktop 1.3.x already turns into one. This
	// holds for a pull from 0 as well: the commit log may have been pruned, so
	// a pull from 0 is not the whole state (only a snapshot is).
	{
		missing, err := s.sync.UncoveredTeams(r.Context(), identity.DeviceID, identity.TeamIDs)
		if err != nil {
			s.writeSyncError(w, r, err)
			return
		}
		if len(missing) > 0 {
			s.logger.InfoContext(r.Context(), "pull needs a snapshot for a new team",
				slog.String("device_id", identity.DeviceID),
				slog.Any("teams", missing),
				slog.String("request_id", RequestID(r.Context())),
			)
			writeErrorDetails(w, http.StatusGone, "cursor_too_old",
				"this device has not received a team it is now in; bootstrap instead",
				map[string]string{"reason": "team_scope"})
			return
		}
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

	s.metrics.PullChanges(len(result.Changes))
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

func (s *server) bootstrap(w http.ResponseWriter, r *http.Request) {
	if s.sync == nil {
		s.notImplemented(w, r)
		return
	}
	identity, ok := IdentityOf(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	var req protocol.BootstrapRequest
	if !DecodeJSON(w, r, &req) {
		return
	}

	cursor, err := protocol.DecodeBootstrapCursor(req.Cursor)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_cursor", err.Error())
		return
	}
	// The memberships the snapshot started with travel in the cursor: a team
	// joined halfway through is missing from the earlier pages, and so is one
	// left and joined again, so neither may be recorded as covered when the
	// last page goes out (CoverTeams also compares the join time).
	snapshotTeams := cursor.Teams
	if req.Cursor == "" {
		memberships, err := s.sync.Memberships(r.Context(), identity.AccountID)
		if err != nil {
			s.writeSyncError(w, r, err)
			return
		}
		snapshotTeams = make([]protocol.CursorMembership, 0, len(memberships))
		for _, m := range memberships {
			snapshotTeams = append(snapshotTeams, protocol.CursorMembership{TeamID: m.TeamID, JoinedAtMs: m.JoinedAtMs})
		}
	}

	result, err := s.sync.Snapshot(r.Context(), store.SnapshotQuery{
		AccountID:  identity.AccountID,
		TeamIDs:    identity.TeamIDs,
		BaseSeq:    cursor.BaseSeq,
		AfterTable: cursor.Table,
		AfterRowID: cursor.RowID,
		Limit:      req.Limit,
	})
	if err != nil {
		s.writeSyncError(w, r, err)
		return
	}

	rows := make([]protocol.Change, 0, len(result.Rows))
	for _, row := range result.Rows {
		mapped, err := toWireSnapshotRow(row)
		if err != nil {
			s.writeSyncError(w, r, err)
			return
		}
		rows = append(rows, mapped)
	}

	response := protocol.BootstrapResponse{
		Rows:      rows,
		ServerSeq: result.ServerSeq,
		HasMore:   result.HasMore,
	}
	if result.HasMore && len(result.Rows) > 0 {
		last := result.Rows[len(result.Rows)-1]
		// The next cursor carries the same instant, so the following page stays
		// part of the same snapshot.
		next, err := protocol.EncodeBootstrapCursor(protocol.BootstrapCursor{
			BaseSeq: result.ServerSeq, Table: last.Table, RowID: last.RowID, Teams: snapshotTeams,
		})
		if err != nil {
			s.writeSyncError(w, r, err)
			return
		}
		response.Cursor = next
	}

	s.logger.DebugContext(r.Context(), "bootstrap served",
		slog.String("account_id", identity.AccountID),
		slog.Int("rows", len(rows)),
		slog.Int64("base_seq", result.ServerSeq),
		slog.Bool("has_more", result.HasMore),
		slog.String("request_id", RequestID(r.Context())),
	)

	if !result.HasMore {
		// CoverTeams keeps only the teams the account is still in, so a team
		// left halfway through is not recorded either.
		covered := make([]store.Membership, 0, len(snapshotTeams))
		for _, m := range snapshotTeams {
			covered = append(covered, store.Membership{TeamID: m.TeamID, JoinedAtMs: m.JoinedAtMs})
		}
		if err := s.sync.CoverTeams(r.Context(), identity.DeviceID, covered); err != nil {
			s.writeSyncError(w, r, err)
			return
		}
	}

	s.metrics.BootstrapRows(len(result.Rows))
	writeJSON(w, http.StatusOK, response)
}

// toWireSnapshotRow renders a stored row as a change. The op reflects the
// tombstone, so a client applies a deleted row as a delete rather than
// resurrecting it.
func toWireSnapshotRow(row store.Row) (protocol.Change, error) {
	wireRow, err := protocol.MergeRow(row.Payload, row.Version, row.UpdatedAtMs,
		row.LastWriterID, row.OwnerID, row.TeamID, row.DeletedAt)
	if err != nil {
		return protocol.Change{}, err
	}

	op := protocol.OpUpsert
	if row.Deleted() {
		op = protocol.OpDelete
	}

	out := protocol.Change{
		ServerSeq:   row.ServerSeq,
		Table:       row.Table,
		RowID:       row.RowID,
		Op:          op,
		Row:         wireRow,
		Version:     row.Version,
		UpdatedAtMs: row.UpdatedAtMs,
	}
	if row.LastWriterID != nil {
		out.LastWriterID = *row.LastWriterID
	}
	return out, nil
}
