package httpapi_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/team-croffle/croffle/apps/server/internal/protocol"
)

// Spec 005 S-12 / V-14: a regular account changes its own password from the
// dashboard. The asking session stays, the other browser is signed out, the
// device token keeps working, and the log holds no password.
func TestDashChangePassword(t *testing.T) {
	srv := startAdminServer(t)
	a := srv.dashLoginAs(t, "bob", "bob-pass")
	b := srv.dashLoginAs(t, "bob", "bob-pass")
	res, body := srv.post(t, "/auth/login", protocol.LoginRequest{
		Username: "bob", Password: "bob-pass", ClientID: "bob-client", DeviceName: "laptop",
	}, "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("device login: %d %s", res.StatusCode, body)
	}
	token := tokenOf(t, body)

	change := func(current, next string, headers map[string]string) (int, []byte) {
		res, body := srv.postWithCookie(t, "/dash/me/password",
			map[string]string{"currentPassword": current, "newPassword": next}, a, headers)
		return res.StatusCode, body
	}
	if code, body := change("wrong", "bob-new-pass", nil); code != http.StatusUnauthorized {
		t.Fatalf("wrong current password: %d %s", code, body)
	}
	if code, body := change("bob-pass", "", nil); code != http.StatusBadRequest {
		t.Fatalf("empty new password: %d %s", code, body)
	}
	if code, body := change("bob-pass", "bob-new-pass", map[string]string{"Origin": "https://evil.example"}); code != http.StatusForbidden || errorCodeOf(body) != "cross_origin" {
		t.Fatalf("cross-origin: %d %s", code, body)
	}
	if code, body := change("bob-pass", "bob-new-pass", nil); code != http.StatusNoContent {
		t.Fatalf("change: %d %s", code, body)
	}

	if res, _ := srv.getWithCookie(t, "/dash/me", a); res.StatusCode != http.StatusOK {
		t.Errorf("the asking session after the change: %d, want 200", res.StatusCode)
	}
	if res, _ := srv.getWithCookie(t, "/dash/me", b); res.StatusCode != http.StatusUnauthorized {
		t.Errorf("the other session after the change: %d, want 401", res.StatusCode)
	}
	if res, _ := srv.post(t, "/auth/whoami", nil, token); res.StatusCode != http.StatusOK {
		t.Errorf("the device token after the change: %d, want 200", res.StatusCode)
	}
	if res, _ := srv.postWithCookie(t, "/dash/login", map[string]string{"username": "bob", "password": "bob-pass"}, nil, nil); res.StatusCode != http.StatusUnauthorized {
		t.Errorf("the old password: %d, want 401", res.StatusCode)
	}
	srv.dashLoginAs(t, "bob", "bob-new-pass")

	logs := srv.log.String()
	if !strings.Contains(logs, `"msg":"password changed"`) {
		t.Error("no password changed log line")
	}
	for _, secret := range []string{"bob-pass", "bob-new-pass"} {
		if strings.Contains(logs, secret) {
			t.Errorf("the log holds %q", secret)
		}
	}
}

// With one-time codes on, the change needs a code too.
func TestDashChangePasswordWithOTP(t *testing.T) {
	srv := startServer(t)
	codes := srv.enrolOTP(t)
	res, body := srv.post(t, "/dash/login", map[string]string{
		"username": testUsername, "password": testPassword, "otp": codes.recovery[0],
	}, "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("dash login: %d %s", res.StatusCode, body)
	}
	var cookie *http.Cookie
	for _, c := range res.Cookies() {
		if c.Name == "croffle_dash" {
			cookie = c
		}
	}
	req := map[string]string{"currentPassword": testPassword, "newPassword": "alice-new-pass"}
	if res, body := srv.postWithCookie(t, "/dash/me/password", req, cookie, nil); res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("without a code: %d %s", res.StatusCode, body)
	}
	// A recovery code: the enrolment just spent this step's TOTP code, and a
	// second one in the same step is a replay.
	req["code"] = codes.recovery[1]
	if res, body := srv.postWithCookie(t, "/dash/me/password", req, cookie, nil); res.StatusCode != http.StatusNoContent {
		t.Fatalf("with a code: %d %s", res.StatusCode, body)
	}
}

func tokenOf(t *testing.T, body []byte) string {
	t.Helper()
	var login protocol.LoginResponse
	if err := json.Unmarshal(body, &login); err != nil || login.Token == "" {
		t.Fatalf("login response %s: %v", body, err)
	}
	return login.Token
}
