package httpapi_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"

	"github.com/team-croffle/croffle/apps/server/internal/protocol"
)

func (s *liveServer) postWithCookie(t *testing.T, path string, body any, cookie *http.Cookie, headers map[string]string) (*http.Response, []byte) {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, s.url+path, strings.NewReader(string(payload)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	data, _ := io.ReadAll(res.Body)
	return res, data
}

func liveCode(t *testing.T, secret string, at time.Time) string {
	t.Helper()
	code, err := totp.GenerateCodeCustom(secret, at, totp.ValidateOpts{
		Period: 30, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		t.Fatal(err)
	}
	return code
}

type enrolled struct {
	secret   string
	recovery []string
}

// enrolOTP switches one-time codes on for the test account through the
// dashboard, as a user would.
func (s *liveServer) enrolOTP(t *testing.T) enrolled {
	t.Helper()
	cookie := s.dashLogin(t)

	res, body := s.postWithCookie(t, "/dash/otp/enroll", map[string]string{}, cookie, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("enroll: %d (%s)", res.StatusCode, body)
	}
	var enrolment struct {
		Secret     string `json:"secret"`
		OTPAuthURI string `json:"otpauthUri"`
		QRPNG      string `json:"qrPng"`
	}
	if err := json.Unmarshal(body, &enrolment); err != nil {
		t.Fatal(err)
	}
	if enrolment.Secret == "" || !strings.HasPrefix(enrolment.OTPAuthURI, "otpauth://totp/") ||
		!strings.HasPrefix(enrolment.QRPNG, "data:image/png;base64,") {
		t.Fatalf("enrolment = %s", body)
	}
	if got := res.Header.Get("Cache-Control"); got != "no-store" {
		t.Errorf("enroll Cache-Control = %q", got)
	}

	// Still pending: the password alone signs in.
	res, _ = s.post(t, "/auth/login", protocol.LoginRequest{
		Username: testUsername, Password: testPassword, ClientID: "client-pending",
	}, "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("login while pending: %d", res.StatusCode)
	}

	res, body = s.postWithCookie(t, "/dash/otp/confirm",
		map[string]string{"code": liveCode(t, enrolment.Secret, time.Now())}, cookie, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("confirm: %d (%s)", res.StatusCode, body)
	}
	var confirmed struct {
		RecoveryCodes []string `json:"recoveryCodes"`
	}
	if err := json.Unmarshal(body, &confirmed); err != nil {
		t.Fatal(err)
	}
	if len(confirmed.RecoveryCodes) != 10 {
		t.Fatalf("recovery codes = %v", confirmed.RecoveryCodes)
	}

	res, body = s.getWithCookie(t, "/dash/me", cookie)
	if res.StatusCode != http.StatusOK || !strings.Contains(string(body), `"otpEnabled":true`) {
		t.Fatalf("me after confirm: %d %s", res.StatusCode, body)
	}
	return enrolled{secret: enrolment.Secret, recovery: confirmed.RecoveryCodes}
}

func errorCodeOf(body []byte) string {
	var parsed struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(body, &parsed)
	return parsed.Code
}

func TestOTPRequiredAtEveryDoor(t *testing.T) {
	doors := map[string]func(srv *liveServer, otp string) (*http.Response, []byte){
		"/auth/login": func(srv *liveServer, otp string) (*http.Response, []byte) {
			return srv.post(t, "/auth/login", protocol.LoginRequest{
				Username: testUsername, Password: testPassword, OTP: otp, ClientID: testClientID,
			}, "")
		},
		"/auth/token": func(srv *liveServer, otp string) (*http.Response, []byte) {
			req := passwordGrant()
			req.OTP = otp
			return srv.post(t, "/auth/token", req, "")
		},
		"/dash/login": func(srv *liveServer, otp string) (*http.Response, []byte) {
			return srv.post(t, "/dash/login", map[string]string{
				"username": testUsername, "password": testPassword, "otp": otp,
			}, "")
		},
	}
	// One server per door: the rate limit is shared by (ip, username) across
	// doors, and this test makes more attempts than a person would.
	for name, door := range doors {
		t.Run(strings.TrimPrefix(name, "/"), func(t *testing.T) {
			srv := startServer(t)
			codes := srv.enrolOTP(t)

			res, body := door(srv, "")
			if res.StatusCode != http.StatusUnauthorized || errorCodeOf(body) != "otp_required" {
				t.Errorf("without code: %d %s, want 401 otp_required", res.StatusCode, body)
			}
			res, body = door(srv, "000000")
			if res.StatusCode != http.StatusUnauthorized || errorCodeOf(body) != "unauthorized" {
				t.Errorf("with a wrong code: %d %s, want 401 unauthorized", res.StatusCode, body)
			}
			if res, body := door(srv, codes.recovery[0]); res.StatusCode != http.StatusOK {
				t.Errorf("with a recovery code: %d %s", res.StatusCode, body)
			}
			if res, _ := door(srv, codes.recovery[0]); res.StatusCode != http.StatusUnauthorized {
				t.Errorf("accepted a recovery code twice: %d", res.StatusCode)
			}
			if res, body := door(srv, codes.recovery[1]); res.StatusCode != http.StatusOK {
				t.Errorf("an unused recovery code after a reused one: %d %s", res.StatusCode, body)
			}
		})
	}
}

func TestOTPStepReplayLive(t *testing.T) {
	srv := startServer(t)
	codes := srv.enrolOTP(t)

	// The confirmation spent the current step. A code for the next step is
	// inside the allowed drift and works once.
	next := liveCode(t, codes.secret, time.Now().Add(30*time.Second))
	login := func(otp string) int {
		res, _ := srv.post(t, "/auth/login", protocol.LoginRequest{
			Username: testUsername, Password: testPassword, OTP: otp, ClientID: testClientID,
		}, "")
		return res.StatusCode
	}
	if status := login(next); status != http.StatusOK {
		t.Fatalf("next-step code: %d", status)
	}
	if status := login(next); status != http.StatusUnauthorized {
		t.Errorf("same code twice: %d", status)
	}
	if status := login(liveCode(t, codes.secret, time.Now())); status != http.StatusUnauthorized {
		t.Errorf("current-step code after a later one: %d", status)
	}
}

func TestOTPDisableAndCrossOrigin(t *testing.T) {
	srv := startServer(t)
	codes := srv.enrolOTP(t)

	res, body := srv.post(t, "/dash/login", map[string]string{
		"username": testUsername, "password": testPassword, "otp": codes.recovery[0],
	}, "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("dash login with a recovery code: %d %s", res.StatusCode, body)
	}
	var cookie *http.Cookie
	for _, c := range res.Cookies() {
		if c.Name == "croffle_dash" {
			cookie = c
		}
	}

	cross := map[string]string{"Origin": "https://evil.example"}
	for _, path := range []string{"/dash/otp/enroll", "/dash/otp/confirm", "/dash/otp/disable"} {
		if res, _ := srv.postWithCookie(t, path, map[string]string{}, cookie, cross); res.StatusCode != http.StatusForbidden {
			t.Errorf("cross-origin %s: %d, want 403", path, res.StatusCode)
		}
	}
	if res, body := srv.postWithCookie(t, "/dash/otp/enroll", map[string]string{}, cookie, nil); res.StatusCode != http.StatusConflict || errorCodeOf(body) != "otp_enabled" {
		t.Errorf("enroll while enabled: %d %s", res.StatusCode, body)
	}

	if res, _ := srv.postWithCookie(t, "/dash/otp/disable",
		map[string]string{"password": testPassword, "code": ""}, cookie, nil); res.StatusCode != http.StatusUnauthorized {
		t.Errorf("disable without a code: %d", res.StatusCode)
	}
	if res, body := srv.postWithCookie(t, "/dash/otp/disable",
		map[string]string{"password": testPassword, "code": codes.recovery[1]}, cookie, nil); res.StatusCode != http.StatusNoContent {
		t.Fatalf("disable: %d %s", res.StatusCode, body)
	}
	res, _ = srv.post(t, "/auth/login", protocol.LoginRequest{
		Username: testUsername, Password: testPassword, ClientID: testClientID,
	}, "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("password alone after disable: %d", res.StatusCode)
	}
}
