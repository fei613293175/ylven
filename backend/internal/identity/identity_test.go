package identity

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNormalizeEmail(t *testing.T) {
	got, err := NormalizeEmail("  User@Example.COM ")
	if err != nil || got != "user@example.com" {
		t.Fatalf("normalize = %q, %v", got, err)
	}
	if _, err := NormalizeEmail("bad"); err == nil {
		t.Fatal("invalid email accepted")
	}
}

func TestRegistrationFlowAndOneTimeOTP(t *testing.T) {
	s, err := NewStore("")
	if err != nil {
		t.Fatal(err)
	}
	c, err := s.CreateChallenge("user@example.com", "register")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.VerifyTurnstile(c.ID, "test-pass", true); err != nil {
		t.Fatal(err)
	}
	o, err := s.CreateOTP(c.ID, "123456")
	if err != nil {
		t.Fatal(err)
	}
	if o.ID == "" {
		t.Fatal("missing OTP id")
	}
	if err := s.VerifyOTP(c.ID, "000000"); err == nil {
		t.Fatal("invalid OTP accepted")
	}
	if err := s.VerifyOTP(c.ID, "123456"); err != nil {
		t.Fatal(err)
	}
	u, err := s.CreateUser(c.ID, "USER@example.com", "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	if u.Email != "user@example.com" || u.PasswordHash == "correct horse battery" {
		t.Fatalf("bad user: %+v", u)
	}
	if _, err := s.CreateUser(c.ID, u.Email, "another password"); err == nil {
		t.Fatal("challenge reused")
	}
}

func TestSessionRefreshRotatesRefreshToken(t *testing.T) {
	s, _ := NewStore("")
	c, _ := s.CreateChallenge("user@example.com", "register")
	_ = s.VerifyTurnstile(c.ID, "test-pass", true)
	_, _ = s.CreateOTP(c.ID, "123456")
	_ = s.VerifyOTP(c.ID, "123456")
	_, _ = s.CreateUser(c.ID, "user@example.com", "correct horse battery")
	l, _ := s.CreateChallenge("user@example.com", "login")
	_ = s.VerifyTurnstile(l.ID, "test-pass", true)
	_, _ = s.CreateOTP(l.ID, "654321")
	_ = s.VerifyOTP(l.ID, "654321")
	first, access, refresh, err := s.CreateSession(l.ID, "user@example.com")
	if err != nil || first.ID == "" || access == "" || refresh == "" {
		t.Fatalf("session issue failed: %+v %v", first, err)
	}
	second, _, nextRefresh, err := s.RotateSession(refresh)
	if err != nil || second.ID == first.ID || nextRefresh == refresh {
		t.Fatalf("rotation failed: %+v %v", second, err)
	}
	if _, _, _, err := s.RotateSession(refresh); err == nil {
		t.Fatal("refresh token was reusable")
	}
	auditJSON, _ := json.Marshal(s.AuditSnapshot())
	if !strings.Contains(string(auditJSON), "user_login") || !strings.Contains(string(auditJSON), "session_refreshed") {
		t.Fatalf("missing auth audit events: %s", auditJSON)
	}
	for _, secret := range []string{access, refresh, nextRefresh, "654321"} {
		if strings.Contains(string(auditJSON), secret) {
			t.Fatalf("audit leaked secret material")
		}
	}
}

func TestSessionLogoutAndRateLimit(t *testing.T) {
	s, _ := NewStore("")
	ok, err := s.AllowAttempt("user@example.com", 1, time.Hour)
	if err != nil || !ok {
		t.Fatalf("first attempt blocked: %v", err)
	}
	ok, err = s.AllowAttempt("user@example.com", 1, time.Hour)
	if err != nil || ok {
		t.Fatalf("second attempt was not limited: %v", err)
	}
}

func TestAdminSettingsRejectRawSecrets(t *testing.T) {
	s, _ := NewStore("")
	if err := s.PutSetting("turnstile", map[string]string{"secret": "plain-text"}); err == nil {
		t.Fatal("raw secret accepted")
	}
	if err := s.PutSetting("turnstile", map[string]string{"site_key_reference": "cf-site", "secret_reference": "vault:turnstile"}); err != nil {
		t.Fatal(err)
	}
	value, ok := s.GetSetting("turnstile")
	if !ok || value["secret_reference"] != "vault:turnstile" {
		t.Fatalf("unexpected setting: %#v", value)
	}
}

func createTestUser(t *testing.T, s *Store, email string) User {
	t.Helper()
	c, err := s.CreateChallenge(email, "register")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.VerifyTurnstile(c.ID, "test-pass", true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateOTP(c.ID, "123456"); err != nil {
		t.Fatal(err)
	}
	if err := s.VerifyOTP(c.ID, "123456"); err != nil {
		t.Fatal(err)
	}
	u, err := s.CreateUser(c.ID, email, "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestAdminRBACStepUpAndUserStatus(t *testing.T) {
	s, _ := NewStore("")
	u := createTestUser(t, s, "member@example.com")
	admin, err := s.BootstrapAdmin("owner@example.com", "long admin password")
	if err != nil {
		t.Fatal(err)
	}
	_, access, err := s.AdminLogin("owner@example.com", "long admin password")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.AuthorizeAdmin(access, "users:status"); !ok {
		t.Fatal("superadmin permission missing")
	}
	stepUp, _, err := s.CreateStepUp(access, "long admin password")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ConsumeStepUp(access, stepUp); err != nil {
		t.Fatal(err)
	}
	if err := s.ConsumeStepUp(access, stepUp); err == nil {
		t.Fatal("step-up token was reusable")
	}
	updated, err := s.SetUserStatus(u.ID, "disabled", admin.ID)
	if err != nil || updated.Status != "disabled" {
		t.Fatalf("status update: %+v %v", updated, err)
	}
}

func TestEmailTemplateVersionAndDeliveryRecords(t *testing.T) {
	s, _ := NewStore("")
	createTestUser(t, s, "mail@example.com")
	templates, deliveries := s.ListEmailTemplates()
	if len(templates) != 2 || len(deliveries) != 1 {
		t.Fatalf("templates=%d deliveries=%d", len(templates), len(deliveries))
	}
	saved, err := s.PutEmailTemplate(EmailTemplate{Key: "register_otp", Subject: "Updated", Body: "Code: {{code}}", Version: 1}, "admin")
	if err != nil || saved.Version != 2 {
		t.Fatalf("template update: %+v %v", saved, err)
	}
	if _, err := s.PutEmailTemplate(EmailTemplate{Key: "register_otp", Subject: "Stale", Body: "Code", Version: 1}, "admin"); err == nil {
		t.Fatal("stale template version accepted")
	}
}

func requestJSON(t *testing.T, handler http.Handler, method, path string, body any, access, stepUp string) *httptest.ResponseRecorder {
	t.Helper()
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	if access != "" {
		req.Header.Set("Authorization", "Bearer "+access)
	}
	if stepUp != "" {
		req.Header.Set("X-Step-Up-Token", stepUp)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder
}

func TestAdminHTTPRequiresBearerAndStepUp(t *testing.T) {
	t.Setenv("OWNER_ADMIN_EMAIL", "owner@example.com")
	t.Setenv("ADMIN_BOOTSTRAP_PASSWORD", "long admin password")
	s, _ := NewStore("")
	_, err := s.BootstrapAdmin("owner@example.com", "long admin password")
	if err != nil {
		t.Fatal(err)
	}
	handler := NewAPI(s).Handler()
	legacy := httptest.NewRequest(http.MethodGet, "/admin/v1/rbac/roles", nil)
	legacy.Header.Set("X-Admin-Role", "superadmin")
	legacyRecorder := httptest.NewRecorder()
	handler.ServeHTTP(legacyRecorder, legacy)
	if legacyRecorder.Code != http.StatusForbidden {
		t.Fatalf("legacy header status=%d", legacyRecorder.Code)
	}
	login := requestJSON(t, handler, http.MethodPost, "/admin/v1/auth/login", map[string]string{"email": "owner@example.com", "password": "long admin password"}, "", "")
	if login.Code != http.StatusCreated {
		t.Fatalf("login status=%d body=%s", login.Code, login.Body.String())
	}
	var loginBody struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(login.Body.Bytes(), &loginBody); err != nil {
		t.Fatal(err)
	}
	roles := requestJSON(t, handler, http.MethodGet, "/admin/v1/rbac/roles", nil, loginBody.AccessToken, "")
	if roles.Code != http.StatusOK {
		t.Fatalf("roles status=%d", roles.Code)
	}
	withoutStepUp := requestJSON(t, handler, http.MethodPost, "/admin/v1/rbac/roles", Role{ID: "auditor", Name: "Auditor", Permissions: []string{"users:read"}}, loginBody.AccessToken, "")
	if withoutStepUp.Code != http.StatusForbidden {
		t.Fatalf("missing step-up status=%d", withoutStepUp.Code)
	}
	confirmation := requestJSON(t, handler, http.MethodPost, "/admin/v1/security/step-up", map[string]string{"password": "long admin password"}, loginBody.AccessToken, "")
	if confirmation.Code != http.StatusCreated {
		t.Fatalf("step-up status=%d body=%s", confirmation.Code, confirmation.Body.String())
	}
	var confirmationBody struct {
		Token string `json:"step_up_token"`
	}
	if err := json.Unmarshal(confirmation.Body.Bytes(), &confirmationBody); err != nil {
		t.Fatal(err)
	}
	created := requestJSON(t, handler, http.MethodPost, "/admin/v1/rbac/roles", Role{ID: "auditor", Name: "Auditor", Permissions: []string{"users:read"}}, loginBody.AccessToken, confirmationBody.Token)
	if created.Code != http.StatusCreated {
		t.Fatalf("create role status=%d body=%s", created.Code, created.Body.String())
	}
	loggedOut := requestJSON(t, handler, http.MethodPost, "/admin/v1/auth/logout", map[string]string{}, loginBody.AccessToken, "")
	if loggedOut.Code != http.StatusOK {
		t.Fatalf("admin logout status=%d body=%s", loggedOut.Code, loggedOut.Body.String())
	}
	afterLogout := requestJSON(t, handler, http.MethodGet, "/admin/v1/rbac/roles", nil, loginBody.AccessToken, "")
	if afterLogout.Code != http.StatusForbidden {
		t.Fatalf("revoked admin session status=%d", afterLogout.Code)
	}
}

func TestAdminHTTPAcceptsConfiguredUsername(t *testing.T) {
	t.Setenv("OWNER_ADMIN_USERNAME", "admin")
	t.Setenv("OWNER_ADMIN_EMAIL", "admin@ai-admin.orbexa.cc")
	t.Setenv("ADMIN_BOOTSTRAP_PASSWORD", "12345678")
	s, _ := NewStore("")
	handler := NewAPI(s).Handler()
	login := requestJSON(t, handler, http.MethodPost, "/admin/v1/auth/login", map[string]string{"username": "admin", "password": "12345678"}, "", "")
	if login.Code != http.StatusCreated {
		t.Fatalf("username login status=%d body=%s", login.Code, login.Body.String())
	}
}

func TestCanonicalRefreshAndDeviceRoutes(t *testing.T) {
	s, _ := NewStore("")
	createTestUser(t, s, "routes@example.com")
	c, _ := s.CreateChallenge("routes@example.com", "login")
	_ = s.VerifyTurnstile(c.ID, "test-pass", true)
	_, _ = s.CreateOTP(c.ID, "654321")
	_ = s.VerifyOTP(c.ID, "654321")
	session, access, refresh, err := s.CreateSession(c.ID, "routes@example.com")
	if err != nil {
		t.Fatal(err)
	}
	handler := NewAPI(s).Handler()
	rotated := requestJSON(t, handler, http.MethodPost, "/api/v1/auth/token/refresh", map[string]string{"refresh_token": refresh}, "", "")
	if rotated.Code != http.StatusOK {
		t.Fatalf("refresh status=%d body=%s", rotated.Code, rotated.Body.String())
	}
	revoked := requestJSON(t, handler, http.MethodDelete, "/api/v1/account/devices/"+session.ID+"/sessions", nil, access, "")
	if revoked.Code != http.StatusOK {
		t.Fatalf("device revoke status=%d body=%s", revoked.Code, revoked.Body.String())
	}
}

func TestP02TurnstileReplayIsRejected(t *testing.T) {
	s, _ := NewStore("")
	c, err := s.CreateChallenge("p02-turnstile@example.com", "login")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.VerifyTurnstile(c.ID, "token", true); err != nil {
		t.Fatal(err)
	}
	if err := s.VerifyTurnstile(c.ID, "token", true); err == nil || err.Error() != "turnstile_already_verified" {
		t.Fatalf("replay error=%v", err)
	}
}

func TestP02AccountSessionEndpointReturnsSafeAuthenticatedView(t *testing.T) {
	s, _ := NewStore("")
	createTestUser(t, s, "restore@example.com")
	c, _ := s.CreateChallenge("restore@example.com", "login")
	_ = s.VerifyTurnstile(c.ID, "token", true)
	_, _ = s.CreateOTP(c.ID, "123456")
	_ = s.VerifyOTP(c.ID, "123456")
	_, access, _, err := s.CreateSession(c.ID, "restore@example.com")
	if err != nil {
		t.Fatal(err)
	}
	handler := NewAPI(s).Handler()
	unauthorized := requestJSON(t, handler, http.MethodGet, "/api/v1/account/session", nil, "", "")
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status=%d", unauthorized.Code)
	}
	restored := requestJSON(t, handler, http.MethodGet, "/api/v1/account/session", nil, access, "")
	if restored.Code != http.StatusOK || strings.Contains(restored.Body.String(), "password_hash") {
		t.Fatalf("restore status=%d body=%s", restored.Code, restored.Body.String())
	}
}

func TestP02RegistrationReturnsSessionAndInitializesWorkspace(t *testing.T) {
	s, _ := NewStore("")
	c, err := s.CreateChallenge("workspace@example.com", "register")
	if err != nil { t.Fatal(err) }
	if err := s.VerifyTurnstile(c.ID, "token", true); err != nil { t.Fatal(err) }
	if _, err := s.CreateOTP(c.ID, "123456"); err != nil { t.Fatal(err) }
	if err := s.VerifyOTP(c.ID, "123456"); err != nil { t.Fatal(err) }
	handler := NewAPI(s).Handler()
	completed := requestJSON(t, handler, http.MethodPost, "/api/v1/auth/register/complete", map[string]string{"challenge_id": c.ID, "email": "workspace@example.com", "password": "correct horse battery"}, "", "")
	if completed.Code != http.StatusCreated || !strings.Contains(completed.Body.String(), "access_token") || !strings.Contains(completed.Body.String(), "个人工作区") {
		t.Fatalf("registration completion status=%d body=%s", completed.Code, completed.Body.String())
	}
	var body struct { AccessToken string `json:"access_token"` }
	if err := json.Unmarshal(completed.Body.Bytes(), &body); err != nil { t.Fatal(err) }
	workspace := requestJSON(t, handler, http.MethodPost, "/api/v1/onboarding/personal-workspace", nil, body.AccessToken, "")
	if workspace.Code != http.StatusOK || !strings.Contains(workspace.Body.String(), "initialized") { t.Fatalf("workspace status=%d body=%s", workspace.Code, workspace.Body.String()) }
	policy := requestJSON(t, handler, http.MethodGet, "/api/v1/auth/password-policy", nil, "", "")
	if policy.Code != http.StatusOK || !strings.Contains(policy.Body.String(), "min_length") { t.Fatalf("policy status=%d body=%s", policy.Code, policy.Body.String()) }
}

func TestP02OTPCooldownPreventsImmediateResend(t *testing.T) {
	s, _ := NewStore("")
	c, _ := s.CreateChallenge("cooldown@example.com", "login")
	_ = s.VerifyTurnstile(c.ID, "token", true)
	if _, err := s.CreateOTP(c.ID, "123456"); err != nil { t.Fatal(err) }
	if _, err := s.CreateOTP(c.ID, "654321"); err == nil || err.Error() != "otp_cooldown" { t.Fatalf("cooldown error=%v", err) }
}
