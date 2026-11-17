package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/team-croffle/croffle/apps/server/internal/auth"
	"github.com/team-croffle/croffle/apps/server/internal/store"
)

// Admin backs the operator API under /dash/admin/: the same operations as the
// CLI, for an account that is an operator (requireOperator).
type Admin interface {
	store.Admin

	FindAccountByUsername(ctx context.Context, username string) (*store.Account, error)
	FindAccountByID(ctx context.Context, id string) (*store.Account, error)
	OTPStatus(ctx context.Context, accountID string) (string, error)
	DeleteOTP(ctx context.Context, accountID string) error
	UpsertDevice(ctx context.Context, device store.Device) error
}

// maxPassword matches the CLI's --password-stdin limit.
const maxPassword = 4096

// newAdminID returns prefix-<16 hex>, the shape the CLI generates.
func newAdminID(prefix string) string {
	var buf [8]byte
	// rand.Read panics rather than failing on supported platforms.
	_, _ = rand.Read(buf[:])
	return prefix + "-" + hex.EncodeToString(buf[:])
}

func (s *server) adminRoutes(mux *http.ServeMux) {
	route := func(method, path string, h http.HandlerFunc) {
		s.dashRoute(mux, method, "/dash/admin/"+path, s.requireOperator(h))
	}
	route(http.MethodGet, "accounts", s.adminAccounts)
	route(http.MethodPost, "accounts", s.adminCreateAccount)
	route(http.MethodPost, "accounts/{id}/disable", s.adminSetDisabled(true))
	route(http.MethodPost, "accounts/{id}/enable", s.adminSetDisabled(false))
	route(http.MethodPost, "accounts/{id}/password", s.adminSetPassword)
	route(http.MethodPost, "accounts/{id}/otp/disable", s.adminDisableOTP)
	route(http.MethodPost, "accounts/{id}/operator", s.adminSetOperator)
	route(http.MethodGet, "devices", s.adminDevices)
	route(http.MethodPost, "devices", s.adminIssueDevice)
	route(http.MethodPost, "devices/{id}/revoke", s.adminRevokeDevice)
	route(http.MethodGet, "teams", s.adminTeams)
	route(http.MethodPost, "teams", s.adminCreateTeam)
	route(http.MethodPost, "teams/{id}", s.adminRenameTeam)
	route(http.MethodGet, "teams/{id}/members", s.adminTeamMembers)
	route(http.MethodPost, "teams/{id}/members", s.adminAddTeamMember)
	route(http.MethodDelete, "teams/{id}/members/{accountId}", s.adminRemoveTeamMember)
	route(http.MethodGet, "info", s.adminInfo)
	route(http.MethodPost, "vacuum", s.adminVacuum)
	route(http.MethodGet, "commits", s.adminCommits)
}

// audit writes the one log line every operator action leaves: who, what, on
// what. Never a password or a token.
func (s *server) audit(r *http.Request, action, target string) {
	s.logger.InfoContext(r.Context(), "operator action",
		slog.String("operator_id", sessionOf(r.Context()).accountID),
		slog.String("action", action),
		slog.String("target", target),
		slog.String("request_id", RequestID(r.Context())),
	)
}

// adminError maps store errors onto the operator API's answers.
func (s *server) adminError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "no such account, device or team")
	case errors.Is(err, store.ErrDuplicate):
		writeError(w, http.StatusConflict, "duplicate", "that name or id is already taken")
	case errors.Is(err, store.ErrLastOperator):
		writeError(w, http.StatusConflict, "last_operator",
			"this is the last active operator; make another account an operator first")
	case errors.Is(err, store.ErrInvalidTeamName):
		writeError(w, http.StatusBadRequest, "invalid_name", "a team name has 1 to 64 characters")
	default:
		s.writeSyncError(w, r, err)
	}
}

func checkPassword(w http.ResponseWriter, password string) bool {
	if password == "" || len(password) > maxPassword {
		writeError(w, http.StatusBadRequest, "invalid_password", "the password must not be empty or longer than 4096 bytes")
		return false
	}
	return true
}

// resolveAccount finds an account by id, then by username, the way the CLI
// accepts either.
func (s *server) resolveAccount(ctx context.Context, idOrName string) (*store.Account, error) {
	idOrName = strings.TrimSpace(idOrName)
	if idOrName == "" {
		return nil, store.ErrNotFound
	}
	account, err := s.admin.FindAccountByID(ctx, idOrName)
	if errors.Is(err, store.ErrNotFound) {
		return s.admin.FindAccountByUsername(ctx, idOrName)
	}
	return account, err
}

type adminAccount struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	CreatedAt   int64  `json:"createdAt"`
	DisabledAt  *int64 `json:"disabledAt,omitempty"`
	Operator    bool   `json:"operator"`
	OTP         string `json:"otp"`
	DeviceCount int    `json:"deviceCount"`
}

func (s *server) adminAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := s.admin.ListAccounts(r.Context())
	if err != nil {
		s.adminError(w, r, err)
		return
	}
	devices, err := s.admin.ListDevices(r.Context(), "")
	if err != nil {
		s.adminError(w, r, err)
		return
	}
	perAccount := map[string]int{}
	for _, d := range devices {
		perAccount[d.AccountID]++
	}
	out := make([]adminAccount, 0, len(accounts))
	for _, a := range accounts {
		otp, err := s.admin.OTPStatus(r.Context(), a.ID)
		if err != nil {
			s.adminError(w, r, err)
			return
		}
		out = append(out, adminAccount{
			ID: a.ID, Username: a.Username, CreatedAt: a.CreatedAt.UnixMilli(),
			DisabledAt: millisPtr(a.DisabledAt), Operator: a.Operator(),
			OTP: otp, DeviceCount: perAccount[a.ID],
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"accounts": out})
}

func (s *server) adminCreateAccount(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Operator bool   `json:"operator"`
	}
	if !DecodeJSON(w, r, &req) {
		return
	}
	username := strings.TrimSpace(req.Username)
	if username == "" {
		writeError(w, http.StatusBadRequest, "invalid_username", "the username must not be empty")
		return
	}
	if !checkPassword(w, req.Password) {
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		s.writeSyncError(w, r, err)
		return
	}
	now := s.now()
	account := store.Account{ID: newAdminID("acct"), Username: username, PasswordHash: hash, CreatedAt: now}
	if req.Operator {
		account.OperatorAt = &now
	}
	if err := s.admin.CreateAccount(r.Context(), account); err != nil {
		s.adminError(w, r, err)
		return
	}
	// As `user create` does: a new account is in the default team, so it can
	// receive what is shared there without a second step.
	if err := s.admin.AddTeamMember(r.Context(), store.DefaultTeamID, account.ID, "member"); err != nil {
		s.adminError(w, r, err)
		return
	}
	s.audit(r, "account create", account.ID)
	writeJSON(w, http.StatusCreated, map[string]any{"account": adminAccount{
		ID: account.ID, Username: account.Username, CreatedAt: now.UnixMilli(),
		Operator: req.Operator, OTP: "off",
	}})
}

func (s *server) adminSetDisabled(disabled bool) http.HandlerFunc {
	action := "account enable"
	if disabled {
		action = "account disable"
	}
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if err := s.admin.SetAccountDisabled(r.Context(), id, disabled, false); err != nil {
			s.adminError(w, r, err)
			return
		}
		s.audit(r, action, id)
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *server) adminSetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
	}
	if !DecodeJSON(w, r, &req) || !checkPassword(w, req.Password) {
		return
	}
	id := r.PathValue("id")
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		s.writeSyncError(w, r, err)
		return
	}
	// A reset usually means the old password leaked: the new password and
	// signing every browser out happen in one transaction. Device tokens keep
	// working, as with `user set-password`; revoking them is a separate step.
	if err := s.admin.ResetPassword(r.Context(), id, hash); err != nil {
		s.adminError(w, r, err)
		return
	}
	s.audit(r, "account password", id)
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) adminDisableOTP(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := s.admin.FindAccountByID(r.Context(), id); err != nil {
		s.adminError(w, r, err)
		return
	}
	if err := s.admin.DeleteOTP(r.Context(), id); err != nil {
		s.adminError(w, r, err)
		return
	}
	s.audit(r, "account otp disable", id)
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) adminSetOperator(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Operator *bool `json:"operator"`
	}
	if !DecodeJSON(w, r, &req) {
		return
	}
	if req.Operator == nil {
		writeError(w, http.StatusBadRequest, "invalid_request", `send {"operator": true} or {"operator": false}`)
		return
	}
	id := r.PathValue("id")
	if err := s.admin.SetOperator(r.Context(), id, *req.Operator, false); err != nil {
		s.adminError(w, r, err)
		return
	}
	action := "operator revoke"
	if *req.Operator {
		action = "operator grant"
	}
	s.audit(r, action, id)
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) adminDevices(w http.ResponseWriter, r *http.Request) {
	accountID := ""
	if filter := r.URL.Query().Get("account"); filter != "" {
		account, err := s.resolveAccount(r.Context(), filter)
		if err != nil {
			s.adminError(w, r, err)
			return
		}
		accountID = account.ID
	}
	devices, err := s.admin.ListDevices(r.Context(), accountID)
	if err != nil {
		s.adminError(w, r, err)
		return
	}
	accounts, err := s.admin.ListAccounts(r.Context())
	if err != nil {
		s.adminError(w, r, err)
		return
	}
	names := map[string]string{}
	for _, a := range accounts {
		names[a.ID] = a.Username
	}

	type row struct {
		ID             string `json:"id"`
		AccountID      string `json:"accountId"`
		Username       string `json:"username"`
		Name           string `json:"name"`
		CreatedAt      int64  `json:"createdAt"`
		LastSeenAt     *int64 `json:"lastSeenAt,omitempty"`
		TokenExpiresAt *int64 `json:"tokenExpiresAt,omitempty"`
		State          string `json:"state"`
	}
	out := make([]row, 0, len(devices))
	for _, d := range devices {
		// No token hash, as on the account's own device list.
		out = append(out, row{
			ID: d.ID, AccountID: d.AccountID, Username: names[d.AccountID], Name: d.Name,
			CreatedAt: d.CreatedAt.UnixMilli(), LastSeenAt: millisPtr(d.LastSeenAt),
			TokenExpiresAt: millisPtr(d.TokenExpiresAt), State: d.State(s.now()),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"devices": out})
}

// adminIssueDevice is `token issue` for the dashboard. The token is in this
// response only.
func (s *server) adminIssueDevice(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username  string `json:"username"`
		Name      string `json:"name"`
		ExpiresIn string `json:"expiresIn"`
		NoExpiry  bool   `json:"noExpiry"`
	}
	if !DecodeJSON(w, r, &req) {
		return
	}
	account, err := s.resolveAccount(r.Context(), req.Username)
	if err != nil {
		s.adminError(w, r, err)
		return
	}
	if account.Disabled() {
		writeError(w, http.StatusConflict, "account_disabled", "enable the account before issuing a token")
		return
	}
	ttl := s.deviceTokenTTL
	switch {
	case req.NoExpiry:
		ttl = 0
	case strings.TrimSpace(req.ExpiresIn) != "":
		parsed, err := time.ParseDuration(strings.TrimSpace(req.ExpiresIn))
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, "invalid_expiry", "expiresIn is a positive duration such as 720h")
			return
		}
		ttl = parsed
	}

	token, hash, err := auth.NewToken()
	if err != nil {
		s.writeSyncError(w, r, err)
		return
	}
	now := s.now()
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "issued by dashboard"
	}
	device := store.Device{
		ID: newAdminID("client"), AccountID: account.ID, Name: name, TokenHash: hash, CreatedAt: now,
	}
	if ttl > 0 {
		expires := now.Add(ttl)
		device.TokenExpiresAt = &expires
	}
	if err := s.admin.UpsertDevice(r.Context(), device); err != nil {
		s.adminError(w, r, err)
		return
	}
	s.audit(r, "token issue", device.ID)
	writeJSON(w, http.StatusCreated, map[string]any{
		"deviceId":       device.ID,
		"accountId":      account.ID,
		"token":          token,
		"tokenExpiresAt": millisPtr(device.TokenExpiresAt),
	})
}

func (s *server) adminRevokeDevice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.admin.RevokeDevice(r.Context(), id); err != nil {
		s.adminError(w, r, err)
		return
	}
	s.audit(r, "token revoke", id)
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) adminTeams(w http.ResponseWriter, r *http.Request) {
	teams, err := s.admin.ListTeams(r.Context())
	if err != nil {
		s.adminError(w, r, err)
		return
	}
	type row struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		CreatedAt   int64  `json:"createdAt"`
		MemberCount int    `json:"memberCount"`
	}
	out := make([]row, 0, len(teams))
	for _, t := range teams {
		members, err := s.admin.ListTeamMembers(r.Context(), t.ID)
		if err != nil {
			s.adminError(w, r, err)
			return
		}
		out = append(out, row{ID: t.ID, Name: t.Name, CreatedAt: t.CreatedAt.UnixMilli(), MemberCount: len(members)})
	}
	writeJSON(w, http.StatusOK, map[string]any{"teams": out})
}

func (s *server) adminCreateTeam(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	}
	if !DecodeJSON(w, r, &req) {
		return
	}
	name, err := store.NormaliseTeamName(req.Name)
	if err != nil {
		s.adminError(w, r, err)
		return
	}
	id := strings.TrimSpace(req.ID)
	if id == "" {
		id = store.NewTeamID()
	} else if err := store.ValidateTeamID(id); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}
	now := s.now()
	if err := s.admin.CreateTeam(r.Context(), store.Team{ID: id, Name: name, CreatedAt: now}); err != nil {
		s.adminError(w, r, err)
		return
	}
	s.audit(r, "team create", id)
	writeJSON(w, http.StatusCreated, map[string]any{"team": map[string]any{
		"id": id, "name": name, "createdAt": now.UnixMilli(), "memberCount": 0,
	}})
}

func (s *server) adminRenameTeam(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if !DecodeJSON(w, r, &req) {
		return
	}
	name, err := store.NormaliseTeamName(req.Name)
	if err != nil {
		s.adminError(w, r, err)
		return
	}
	id := r.PathValue("id")
	if err := s.admin.RenameTeam(r.Context(), id, name); err != nil {
		s.adminError(w, r, err)
		return
	}
	s.audit(r, "team rename", id)
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) adminTeamMembers(w http.ResponseWriter, r *http.Request) {
	members, err := s.admin.ListTeamMembers(r.Context(), r.PathValue("id"))
	if err != nil {
		s.adminError(w, r, err)
		return
	}
	type row struct {
		AccountID string `json:"accountId"`
		Username  string `json:"username"`
		Role      string `json:"role"`
		CreatedAt int64  `json:"createdAt"`
	}
	out := make([]row, 0, len(members))
	for _, m := range members {
		out = append(out, row{AccountID: m.AccountID, Username: m.Username, Role: m.Role, CreatedAt: m.CreatedAt.UnixMilli()})
	}
	writeJSON(w, http.StatusOK, map[string]any{"members": out})
}

func (s *server) adminAddTeamMember(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
	}
	if !DecodeJSON(w, r, &req) {
		return
	}
	account, err := s.resolveAccount(r.Context(), req.Username)
	if err != nil {
		s.adminError(w, r, err)
		return
	}
	teamID := r.PathValue("id")
	if err := s.admin.AddTeamMember(r.Context(), teamID, account.ID, "member"); err != nil {
		s.adminError(w, r, err)
		return
	}
	s.audit(r, "team add member", teamID+"/"+account.ID)
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) adminRemoveTeamMember(w http.ResponseWriter, r *http.Request) {
	teamID, accountID := r.PathValue("id"), r.PathValue("accountId")
	if err := s.admin.RemoveTeamMember(r.Context(), teamID, accountID); err != nil {
		s.adminError(w, r, err)
		return
	}
	s.audit(r, "team remove member", teamID+"/"+accountID)
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) adminInfo(w http.ResponseWriter, r *http.Request) {
	info, err := s.admin.Info(r.Context())
	if err != nil {
		s.adminError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func (s *server) adminVacuum(w http.ResponseWriter, r *http.Request) {
	before, err := s.admin.Info(r.Context())
	if err != nil {
		s.adminError(w, r, err)
		return
	}
	if err := s.admin.Vacuum(r.Context()); err != nil {
		s.adminError(w, r, err)
		return
	}
	after, err := s.admin.Info(r.Context())
	if err != nil {
		s.adminError(w, r, err)
		return
	}
	s.audit(r, "vacuum", "database")
	writeJSON(w, http.StatusOK, map[string]any{"sizeBefore": before.SizeBytes, "sizeAfter": after.SizeBytes})
}

// adminCommits pages through the whole commit log: the metadata of every
// commit, without the row payloads (the CLI's `log dump` has those).
func (s *server) adminCommits(w http.ResponseWriter, r *http.Request) {
	limit := intQuery(r, "limit", 50)
	if limit < 1 || limit > 500 {
		limit = 50
	}
	since := int64(intQuery(r, "sinceSeq", 0))
	changes, err := s.admin.DumpCommitLog(r.Context(), since, limit+1)
	if err != nil {
		s.adminError(w, r, err)
		return
	}
	hasMore := len(changes) > limit
	if hasMore {
		changes = changes[:limit]
	}
	type row struct {
		ServerSeq   int64   `json:"serverSeq"`
		Table       string  `json:"table"`
		RowID       string  `json:"rowId"`
		Op          string  `json:"op"`
		OwnerID     string  `json:"ownerId"`
		TeamID      *string `json:"teamId,omitempty"`
		Version     int64   `json:"version"`
		UpdatedAtMs int64   `json:"updatedAtMs"`
		Writer      string  `json:"lastWriterId,omitempty"`
	}
	out := make([]row, 0, len(changes))
	next := since
	for _, c := range changes {
		entry := row{
			ServerSeq: c.ServerSeq, Table: c.Table, RowID: c.RowID, Op: string(c.Op),
			OwnerID: c.OwnerID, TeamID: c.TeamID, Version: c.Version, UpdatedAtMs: c.UpdatedAtMs,
		}
		if c.LastWriterID != nil {
			entry.Writer = *c.LastWriterID
		}
		out = append(out, entry)
		next = c.ServerSeq
	}
	writeJSON(w, http.StatusOK, map[string]any{"commits": out, "hasMore": hasMore, "serverSeq": next})
}
