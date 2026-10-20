package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/team-croffle/croffle/apps/server/internal/auth"
	"github.com/team-croffle/croffle/apps/server/internal/hooks"
	"github.com/team-croffle/croffle/apps/server/internal/store"
)

// Dashboard session cookie and lifetime.
const (
	sessionCookie   = "croffle_dash"
	sessionLifetime = 12 * time.Hour
)

// Dash is what the dashboard needs from the store. It is read-only apart from
// the session rows themselves: the dashboard of server 0.1 shows state, and
// anything that changes state is a CLI command.
type Dash interface {
	store.Sessions

	FindAccountByID(ctx context.Context, id string) (*store.Account, error)
	TeamsOf(ctx context.Context, accountID string) ([]string, error)
	ListDevices(ctx context.Context, accountID string) ([]store.Device, error)
	ListTeams(ctx context.Context) ([]store.Team, error)
	// Rows and commits are read through the same visibility rules as a sync
	// client: the dashboard shows your own data, not everyone's. An operator
	// who needs everything has the CLI, which already requires access to the
	// database file.
	Snapshot(ctx context.Context, query store.SnapshotQuery) (store.SnapshotResult, error)
	PullSince(ctx context.Context, query store.PullQuery) (store.PullResult, error)
}

type dashIdentity struct {
	accountID string
	username  string
	teams     []string
}

func (s *server) dashRoutes(mux *http.ServeMux) {
	if s.dash == nil {
		return
	}
	s.route(mux, http.MethodPost, "/dash/login", s.dashLogin)
	s.route(mux, http.MethodPost, "/dash/logout", s.dashLogout)
	s.route(mux, http.MethodGet, "/dash/me", s.requireSession(s.dashMe))
	s.route(mux, http.MethodGet, "/dash/devices", s.requireSession(s.dashDevices))
	s.route(mux, http.MethodGet, "/dash/teams", s.requireSession(s.dashTeams))
	s.route(mux, http.MethodGet, "/dash/rows", s.requireSession(s.dashRows))
	s.route(mux, http.MethodGet, "/dash/commits", s.requireSession(s.dashCommits))
}

type dashLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type dashMeResponse struct {
	AccountID string   `json:"accountId"`
	Username  string   `json:"username"`
	Teams     []string `json:"teams"`
}

func (s *server) dashLogin(w http.ResponseWriter, r *http.Request) {
	var req dashLoginRequest
	if !DecodeJSON(w, r, &req) {
		return
	}

	// CheckPassword rather than Login: a browser login must not leave a device
	// row behind, and must not mint a token that can push. Rate limiting and
	// the equal-time failure path are shared with the device login.
	account, err := s.auth.CheckPassword(r.Context(), req.Username, req.Password, RemoteIP(r.Context()))
	if err != nil {
		s.writeAuthError(w, r, err, slog.String("surface", "dashboard"))
		return
	}

	token, hash, err := auth.NewToken()
	if err != nil {
		s.writeSyncError(w, r, err)
		return
	}

	now := s.now()
	if err := s.dash.CreateSession(r.Context(), store.Session{
		TokenHash: hash,
		AccountID: account.ID,
		CreatedAt: now,
		ExpiresAt: now.Add(sessionLifetime),
	}); err != nil {
		s.writeSyncError(w, r, err)
		return
	}

	s.setSessionCookie(w, r, token, sessionLifetime)
	s.logger.InfoContext(r.Context(), "dashboard login",
		slog.String("account_id", account.ID),
		slog.String("request_id", RequestID(r.Context())),
	)

	s.hooks.Emit(r.Context(), hooks.Event{Type: hooks.TypeAuthSucceeded, Data: hooks.AuthSucceeded{
		AccountID: account.ID,
		Method:    "password",
		Surface:   "dashboard",
	}})

	teams, _ := s.dash.TeamsOf(r.Context(), account.ID)
	writeJSON(w, http.StatusOK, dashMeResponse{
		AccountID: account.ID,
		Username:  account.Username,
		Teams:     orEmpty(teams),
	})
}

func (s *server) dashLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookie); err == nil && cookie.Value != "" {
		if err := s.dash.DeleteSession(r.Context(), auth.HashToken(cookie.Value)); err != nil {
			s.logger.WarnContext(r.Context(), "could not delete session",
				slog.String("error", err.Error()))
		}
	}
	// Clear the cookie even when there was no session: logging out must never
	// leave the browser holding something.
	s.setSessionCookie(w, r, "", -time.Hour)
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) dashMe(w http.ResponseWriter, r *http.Request) {
	identity := sessionOf(r.Context())
	writeJSON(w, http.StatusOK, dashMeResponse{
		AccountID: identity.accountID,
		Username:  identity.username,
		Teams:     orEmpty(identity.teams),
	})
}

func (s *server) dashDevices(w http.ResponseWriter, r *http.Request) {
	identity := sessionOf(r.Context())

	devices, err := s.dash.ListDevices(r.Context(), identity.accountID)
	if err != nil {
		s.writeSyncError(w, r, err)
		return
	}

	type row struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		CreatedAt  int64  `json:"createdAt"`
		LastSeenAt *int64 `json:"lastSeenAt,omitempty"`
		State      string `json:"state"`
	}

	out := make([]row, 0, len(devices))
	for _, device := range devices {
		// No token hash: it is credential material, and a dashboard has no use
		// for it.
		out = append(out, row{
			ID: device.ID, Name: device.Name,
			CreatedAt:  device.CreatedAt.UnixMilli(),
			LastSeenAt: millisPtr(device.LastSeenAt),
			State:      deviceState(device),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"devices": out})
}

func (s *server) dashTeams(w http.ResponseWriter, r *http.Request) {
	identity := sessionOf(r.Context())

	teams, err := s.dash.ListTeams(r.Context())
	if err != nil {
		s.writeSyncError(w, r, err)
		return
	}

	type row struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		IsMember bool   `json:"isMember"`
	}

	out := make([]row, 0, len(teams))
	for _, team := range teams {
		out = append(out, row{ID: team.ID, Name: team.Name, IsMember: contains(identity.teams, team.ID)})
	}
	writeJSON(w, http.StatusOK, map[string]any{"teams": out})
}

func (s *server) dashRows(w http.ResponseWriter, r *http.Request) {
	identity := sessionOf(r.Context())

	result, err := s.dash.Snapshot(r.Context(), store.SnapshotQuery{
		AccountID:  identity.accountID,
		TeamIDs:    identity.teams,
		AfterTable: r.URL.Query().Get("afterTable"),
		AfterRowID: r.URL.Query().Get("afterRowId"),
		Limit:      intQuery(r, "limit", 50),
	})
	if err != nil {
		s.writeSyncError(w, r, err)
		return
	}

	type row struct {
		Table       string  `json:"table"`
		RowID       string  `json:"rowId"`
		Version     int64   `json:"version"`
		UpdatedAtMs int64   `json:"updatedAtMs"`
		TeamID      *string `json:"teamId,omitempty"`
		Deleted     bool    `json:"deleted"`
		ServerSeq   int64   `json:"serverSeq"`
		// Payload is sent so a row can be inspected, but the dashboard keeps it
		// collapsed: it is the user's calendar content.
		Payload string `json:"payload"`
	}

	out := make([]row, 0, len(result.Rows))
	for _, stored := range result.Rows {
		out = append(out, row{
			Table: stored.Table, RowID: stored.RowID, Version: stored.Version,
			UpdatedAtMs: stored.UpdatedAtMs, TeamID: stored.TeamID,
			Deleted: stored.Deleted(), ServerSeq: stored.ServerSeq,
			Payload: string(stored.Payload),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"rows":      out,
		"hasMore":   result.HasMore,
		"serverSeq": result.ServerSeq,
	})
}

func (s *server) dashCommits(w http.ResponseWriter, r *http.Request) {
	identity := sessionOf(r.Context())

	result, err := s.dash.PullSince(r.Context(), store.PullQuery{
		AccountID: identity.accountID,
		TeamIDs:   identity.teams,
		SinceSeq:  int64(intQuery(r, "sinceSeq", 0)),
		Limit:     intQuery(r, "limit", 50),
	})
	if err != nil {
		s.writeSyncError(w, r, err)
		return
	}

	type row struct {
		ServerSeq   int64   `json:"serverSeq"`
		Table       string  `json:"table"`
		RowID       string  `json:"rowId"`
		Op          string  `json:"op"`
		Version     int64   `json:"version"`
		UpdatedAtMs int64   `json:"updatedAtMs"`
		Writer      string  `json:"lastWriterId,omitempty"`
		TeamID      *string `json:"teamId,omitempty"`
	}

	out := make([]row, 0, len(result.Changes))
	for _, change := range result.Changes {
		entry := row{
			ServerSeq: change.ServerSeq, Table: change.Table, RowID: change.RowID,
			Op: string(change.Op), Version: change.Version,
			UpdatedAtMs: change.UpdatedAtMs, TeamID: change.TeamID,
		}
		if change.LastWriterID != nil {
			entry.Writer = *change.LastWriterID
		}
		out = append(out, entry)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"commits":   out,
		"hasMore":   result.HasMore,
		"serverSeq": result.ServerSeq,
	})
}

// requireSession resolves the dashboard cookie. It deliberately does not accept
// a bearer token: the two credentials stay separate in both directions.
func (s *server) requireSession(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookie)
		if err != nil || cookie.Value == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized", "sign in to the dashboard first")
			return
		}

		session, err := s.dash.FindSession(r.Context(), auth.HashToken(cookie.Value), s.now())
		if errors.Is(err, store.ErrNotFound) {
			// Expired or unknown: clear the cookie so the browser stops
			// sending something that will never work again.
			s.setSessionCookie(w, r, "", -time.Hour)
			writeError(w, http.StatusUnauthorized, "unauthorized", "the dashboard session has expired")
			return
		}
		if err != nil {
			s.writeSyncError(w, r, err)
			return
		}

		account, err := s.dash.FindAccountByID(r.Context(), session.AccountID)
		if err != nil || account.Disabled() {
			s.setSessionCookie(w, r, "", -time.Hour)
			writeError(w, http.StatusUnauthorized, "unauthorized", "this account can no longer sign in")
			return
		}

		teams, err := s.dash.TeamsOf(r.Context(), session.AccountID)
		if err != nil {
			s.writeSyncError(w, r, err)
			return
		}

		identity := dashIdentity{accountID: account.ID, username: account.Username, teams: teams}
		next(w, r.WithContext(context.WithValue(r.Context(), ctxKeySession, identity)))
	}
}

func sessionOf(ctx context.Context) dashIdentity {
	identity, _ := ctx.Value(ctxKeySession).(dashIdentity)
	return identity
}

// setSessionCookie writes the session cookie.
//
// Secure is set when the request arrived over HTTPS, which behind a proxy is
// only visible through X-Forwarded-Proto - and that header is trusted only when
// the operator said a proxy is there. Marking it Secure unconditionally would
// make a plain-HTTP LAN deployment unable to log in at all, because the browser
// would refuse to store the cookie.
func (s *server) setSessionCookie(w http.ResponseWriter, r *http.Request, value string, maxAge time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    value,
		Path:     s.basePath,
		HttpOnly: true,
		Secure:   s.requestIsHTTPS(r),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(maxAge.Seconds()),
	})
}

func (s *server) requestIsHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	if s.trustProxy && r.Header.Get("X-Forwarded-Proto") == "https" {
		return true
	}
	return false
}

func deviceState(device store.Device) string {
	switch {
	case device.Revoked():
		return "revoked"
	case device.LastSeenAt == nil:
		return "pending"
	default:
		return "active"
	}
}

func intQuery(r *http.Request, name string, fallback int) int {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return fallback
	}
	return value
}

func millisPtr(t *time.Time) *int64 {
	if t == nil {
		return nil
	}
	ms := t.UnixMilli()
	return &ms
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func orEmpty(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
