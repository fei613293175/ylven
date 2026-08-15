package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

func TestP02FirstPartyVerificationIsHashedLimitedAndOneTime(t *testing.T) {
	s, _ := NewStore("")
	c, err := s.CreateChallenge("p02-answer@example.com", "login")
	if err != nil {
		t.Fatal(err)
	}
	if c.VerificationQuestion == "" || c.VerificationDigest == "" || strings.Contains(c.VerificationDigest, c.VerificationQuestion) {
		t.Fatalf("unsafe challenge: %+v", c)
	}
	var left, right int
	if _, err := fmt.Sscanf(c.VerificationQuestion, "%d + %d = ?", &left, &right); err != nil {
		t.Fatal(err)
	}
	if err := s.VerifyAnswer(c.ID, "999"); err == nil || err.Error() != "verification_incorrect" {
		t.Fatalf("wrong answer error=%v", err)
	}
	if err := s.VerifyAnswer(c.ID, fmt.Sprint(left+right)); err != nil {
		t.Fatal(err)
	}
	if err := s.VerifyAnswer(c.ID, fmt.Sprint(left+right)); err == nil || err.Error() != "verification_already_verified" {
		t.Fatalf("replay error=%v", err)
	}
}

func TestP02SessionRefreshKeepsSingleDevice(t *testing.T) {
	s, _ := NewStore("")
	createTestUser(t, s, "one-device@example.com")
	c, _ := s.CreateChallenge("one-device@example.com", "login")
	_ = s.VerifyTurnstile(c.ID, "token", true)
	_, _ = s.CreateOTP(c.ID, "123456")
	_ = s.VerifyOTP(c.ID, "123456")
	first, _, refresh, err := s.CreateSession(c.ID, "one-device@example.com")
	if err != nil {
		t.Fatal(err)
	}
	rotated, access, refresh, err := s.RotateSession(refresh)
	if err != nil {
		t.Fatal(err)
	}
	rotatedAgain, access, _, err := s.RotateSession(refresh)
	if err != nil {
		t.Fatal(err)
	}
	devices, err := s.ListSessions(access)
	if err != nil {
		t.Fatal(err)
	}
	if len(devices) != 1 {
		t.Fatalf("devices=%d want 1: %+v", len(devices), devices)
	}
	if devices[0].ID != first.DeviceID || devices[0].SessionID != rotatedAgain.ID || rotated.DeviceID != first.DeviceID {
		t.Fatalf("device continuity lost: %+v", devices[0])
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
	if err != nil {
		t.Fatal(err)
	}
	if err := s.VerifyTurnstile(c.ID, "token", true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateOTP(c.ID, "123456"); err != nil {
		t.Fatal(err)
	}
	if err := s.VerifyOTP(c.ID, "123456"); err != nil {
		t.Fatal(err)
	}
	handler := NewAPI(s).Handler()
	completed := requestJSON(t, handler, http.MethodPost, "/api/v1/auth/register/complete", map[string]string{"challenge_id": c.ID, "email": "workspace@example.com", "password": "correct horse battery"}, "", "")
	if completed.Code != http.StatusCreated || !strings.Contains(completed.Body.String(), "access_token") || !strings.Contains(completed.Body.String(), "个人工作区") {
		t.Fatalf("registration completion status=%d body=%s", completed.Code, completed.Body.String())
	}
	var body struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(completed.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	workspace := requestJSON(t, handler, http.MethodPost, "/api/v1/onboarding/personal-workspace", nil, body.AccessToken, "")
	if workspace.Code != http.StatusOK || !strings.Contains(workspace.Body.String(), "initialized") {
		t.Fatalf("workspace status=%d body=%s", workspace.Code, workspace.Body.String())
	}
	policy := requestJSON(t, handler, http.MethodGet, "/api/v1/auth/password-policy", nil, "", "")
	if policy.Code != http.StatusOK || !strings.Contains(policy.Body.String(), "min_length") {
		t.Fatalf("policy status=%d body=%s", policy.Code, policy.Body.String())
	}
}

func TestP02OTPCooldownPreventsImmediateResend(t *testing.T) {
	s, _ := NewStore("")
	c, _ := s.CreateChallenge("cooldown@example.com", "login")
	_ = s.VerifyTurnstile(c.ID, "token", true)
	if _, err := s.CreateOTP(c.ID, "123456"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateOTP(c.ID, "654321"); err == nil || err.Error() != "otp_cooldown" {
		t.Fatalf("cooldown error=%v", err)
	}
}

func TestP03ConversationOwnershipPaginationSearchAndRecycle(t *testing.T) {
	s, _ := NewStore("")
	createTestUser(t, s, "p03@example.com")
	login := createAuthenticatedTestSession(t, s, "p03@example.com")
	first, err := s.CreateConversation(login, "项目 Alpha")
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.CreateConversation(login, "周报")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdateConversation(login, first.ID, "项目 Alpha 重命名"); err != nil {
		t.Fatal(err)
	}
	// Messages are persisted by the next work packet. Insert a real stored
	// message here so this work packet verifies that P03-004 searches the
	// message body as well as the conversation title.
	if _, err := s.AppendMessage(login, second.ID, "user", "正文关键词 needle"); err != nil {
		t.Fatal(err)
	}
	items, cursor, err := s.ListConversations(login, "", 1, false)
	if err != nil || len(items) != 1 || cursor == "" {
		t.Fatalf("page=%+v cursor=%q err=%v", items, cursor, err)
	}
	next, _, err := s.ListConversations(login, cursor, 10, false)
	if err != nil || len(next) != 1 || next[0].ID == items[0].ID {
		t.Fatalf("next page=%+v err=%v", next, err)
	}
	if _, err := s.SearchConversations(login, "重命名", 10); err != nil {
		t.Fatal(err)
	}
	matched, err := s.SearchConversations(login, "needle", 10)
	if err != nil || len(matched) != 1 || matched[0].ID != second.ID {
		t.Fatalf("message-body search=%+v err=%v", matched, err)
	}
	if _, err := s.ArchiveConversation(login, second.ID); err != nil {
		t.Fatal(err)
	}
	visible, _, err := s.ListConversations(login, "", 10, false)
	if err != nil || len(visible) != 1 {
		t.Fatalf("archived still visible: %+v %v", visible, err)
	}
	deleted, err := s.DeleteConversation(login, first.ID)
	if err != nil || deleted.DeletedAt == nil || deleted.Status != "recycle_pending" {
		t.Fatalf("recycle=%+v err=%v", deleted, err)
	}
	other := createAuthenticatedTestSession(t, s, "other@example.com")
	if _, err := s.UpdateConversation(other, first.ID, "越权"); err == nil {
		t.Fatal("cross-user conversation access accepted")
	}
}

func TestP03W07MobileModelCatalogReturnsAuthenticatedStableIDs(t *testing.T) {
	s, _ := NewStore("")
	createTestUser(t, s, "model-catalog@example.com")
	access := createAuthenticatedTestSession(t, s, "model-catalog@example.com")
	s.mu.Lock()
	s.data.ModelCatalog = []ModelCatalogEntry{
		{ID: "gpt-5.6-sol", Name: "GPT-5.6 Sol", Enabled: true, Description: "complex work"},
		{ID: "disabled-model", Name: "Disabled", Enabled: false},
	}
	s.mu.Unlock()

	handler := NewAPI(s).Handler()
	response := requestJSON(t, handler, http.MethodGet, "/api/mobile/v1/models", nil, access, "")
	if response.Code != http.StatusOK {
		t.Fatalf("model catalog status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		Items []ModelCatalogEntry `json:"items"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 2 || body.Items[0].ID != "gpt-5.6-sol" || body.Items[0].Name != "GPT-5.6 Sol" || body.Items[1].Enabled {
		t.Fatalf("unexpected model catalog: %+v", body.Items)
	}
	if len(body.Items[0].ReasoningProfiles) != 1 || body.Items[0].ReasoningProfiles[0] != "auto" {
		t.Fatalf("unsupported reasoning capability advertised: %+v", body.Items[0].ReasoningProfiles)
	}
	unauthorized := requestJSON(t, handler, http.MethodGet, "/api/mobile/v1/models", nil, "", "")
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated model catalog status=%d body=%s", unauthorized.Code, unauthorized.Body.String())
	}
}

func TestP03RunEndpointRejectsUnconfiguredProvider(t *testing.T) {
	s, _ := NewStore("")
	createTestUser(t, s, "runtime-api@example.com")
	access := createAuthenticatedTestSession(t, s, "runtime-api@example.com")
	conversation, err := s.CreateConversation(access, "运行 API")
	if err != nil {
		t.Fatal(err)
	}
	api := NewAPI(s)
	api.ChatRuntimeMode = "unconfigured"
	response := requestJSON(t, api.Handler(), http.MethodPost, "/api/mobile/v1/conversations/"+conversation.ID+"/runs", map[string]string{"body": "hello"}, access, "")
	if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), "chat_runtime_unavailable") {
		t.Fatalf("unconfigured runtime response=%d body=%s", response.Code, response.Body.String())
	}
}

func TestP03RunEndpointPersistsProviderResponseAndResumesSSE(t *testing.T) {
	s, _ := NewStore("")
	createTestUser(t, s, "runtime-upstream@example.com")
	access := createAuthenticatedTestSession(t, s, "runtime-upstream@example.com")
	conversation, err := s.CreateConversation(access, "上游运行")
	if err != nil {
		t.Fatal(err)
	}
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("unexpected provider request %s %q", r.URL.Path, r.Header.Get("Authorization"))
		}
		var request struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.Model != "gpt-test" {
			t.Fatalf("unexpected provider model %q", request.Model)
		}
		_, _ = w.Write([]byte("{\"choices\":[{\"message\":{\"content\":\"来自受控上游的回答\"}}]}"))
	}))
	defer provider.Close()
	api := NewAPI(s)
	api.ChatRuntimeMode = "upstream"
	api.ChatResponder = OpenAICompatibleResponder{Endpoint: provider.URL, APIKey: "test-key", DefaultModel: "gpt-test", Client: provider.Client()}
	created := requestJSON(t, api.Handler(), http.MethodPost, "/api/mobile/v1/conversations/"+conversation.ID+"/runs", map[string]string{"body": "测试正文", "model": "ylven-default"}, access, "")
	if created.Code != http.StatusAccepted {
		t.Fatalf("run create status=%d body=%s", created.Code, created.Body.String())
	}
	var run MessageRun
	if err := json.Unmarshal(created.Body.Bytes(), &run); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 50; i++ {
		current, _, err := s.Run(access, run.ID)
		if err != nil {
			t.Fatal(err)
		}
		if current.Status == "completed" {
			run = current
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if run.Status != "completed" {
		t.Fatalf("run did not complete: %+v", run)
	}
	events := requestJSON(t, api.Handler(), http.MethodGet, "/api/mobile/v1/runs/"+run.ID+"/events?after=1", nil, access, "")
	if events.Code != http.StatusOK || !strings.Contains(events.Body.String(), "delta\":\"自") || !strings.Contains(events.Body.String(), "delta\":\"受") || strings.Contains(events.Body.String(), "id: 1\ndata:") {
		t.Fatalf("events status=%d body=%s", events.Code, events.Body.String())
	}
}

type blockingChatResponder struct {
	started chan struct{}
	release chan struct{}
}

func (r blockingChatResponder) Respond(ctx context.Context, _ ChatRequest) (ChatResponse, error) {
	close(r.started)
	select {
	case <-r.release:
		return ChatResponse{Content: "late answer"}, nil
	case <-ctx.Done():
		return ChatResponse{}, ctx.Err()
	}
}

func TestP03RunCancellationPreventsLateAssistantPersistence(t *testing.T) {
	s, _ := NewStore("")
	createTestUser(t, s, "cancel-runtime@example.com")
	access := createAuthenticatedTestSession(t, s, "cancel-runtime@example.com")
	conversation, err := s.CreateConversation(access, "取消运行")
	if err != nil {
		t.Fatal(err)
	}
	provider := blockingChatResponder{started: make(chan struct{}), release: make(chan struct{})}
	api := NewAPI(s)
	api.ChatRuntimeMode = "upstream"
	api.ChatResponder = provider
	created := requestJSON(t, api.Handler(), http.MethodPost, "/api/mobile/v1/conversations/"+conversation.ID+"/runs", map[string]string{"body": "stop me"}, access, "")
	if created.Code != http.StatusAccepted {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	var run MessageRun
	if err := json.Unmarshal(created.Body.Bytes(), &run); err != nil {
		t.Fatal(err)
	}
	select {
	case <-provider.started:
	case <-time.After(time.Second):
		t.Fatal("provider did not start")
	}
	cancelled := requestJSON(t, api.Handler(), http.MethodPost, "/api/mobile/v1/runs/"+run.ID+"/cancel", nil, access, "")
	if cancelled.Code != http.StatusOK {
		t.Fatalf("cancel status=%d body=%s", cancelled.Code, cancelled.Body.String())
	}
	close(provider.release)
	for i := 0; i < 50; i++ {
		current, _, _ := s.Run(access, run.ID)
		if current.Status != "streaming" {
			run = current
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if run.Status != "cancelled" || run.AssistantMessageID != "" {
		t.Fatalf("late completion changed run: %+v", run)
	}
	_, messages, err := s.Run(access, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 {
		t.Fatalf("messages after cancellation=%+v", messages)
	}
}

func TestP03RunProviderFailurePersistsStableFailedState(t *testing.T) {
	s, _ := NewStore("")
	createTestUser(t, s, "failure-runtime@example.com")
	access := createAuthenticatedTestSession(t, s, "failure-runtime@example.com")
	conversation, err := s.CreateConversation(access, "失败运行")
	if err != nil {
		t.Fatal(err)
	}
	api := NewAPI(s)
	api.ChatRuntimeMode = "upstream"
	api.ChatResponder = staticErrorChatResponder{}
	created := requestJSON(t, api.Handler(), http.MethodPost, "/api/mobile/v1/conversations/"+conversation.ID+"/runs", map[string]string{"body": "fail"}, access, "")
	if created.Code != http.StatusAccepted {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	var run MessageRun
	if err := json.Unmarshal(created.Body.Bytes(), &run); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 50; i++ {
		current, _, _ := s.Run(access, run.ID)
		if current.Status != "streaming" {
			run = current
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if run.Status != "failed" || run.ErrorCode == "" {
		t.Fatalf("failed run=%+v", run)
	}
	_, events, err := s.Events(access, run.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Type != "failed" {
		t.Fatalf("failure events=%+v", events)
	}
}

func TestP03FailedRunRecordsLatency(t *testing.T) {
	s, _ := NewStore("")
	createTestUser(t, s, "failure-latency@example.com")
	access := createAuthenticatedTestSession(t, s, "failure-latency@example.com")
	conversation, err := s.CreateConversation(access, "失败延迟")
	if err != nil {
		t.Fatal(err)
	}
	run, err := s.StartRun(access, conversation.ID, "fail", "ylven-default")
	if err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	stored := s.data.Runs[run.ID]
	stored.StartedAt = time.Now().UTC().Add(-25 * time.Millisecond)
	s.data.Runs[run.ID] = stored
	s.mu.Unlock()
	failed, err := s.FailRun(access, run.ID, "chat_provider_unavailable")
	if err != nil {
		t.Fatal(err)
	}
	if failed.LatencyMs < 25 {
		t.Fatalf("failed run latency was not recorded: %+v", failed)
	}
}

func TestP03CitationEndpointUsesPersistedAssistantContent(t *testing.T) {
	s, _ := NewStore("")
	createTestUser(t, s, "w03@example.com")
	access := createAuthenticatedTestSession(t, s, "w03@example.com")
	api := NewAPI(s)
	conversation, err := s.CreateConversation(access, "引用会话")
	if err != nil {
		t.Fatal(err)
	}
	message, err := s.AppendMessage(access, conversation.ID, "assistant", "参考 https://example.com/docs。")
	if err != nil {
		t.Fatal(err)
	}
	citations := requestJSON(t, api.Handler(), http.MethodGet, "/api/mobile/v1/messages/"+message.ID+"/citations", nil, access, "")
	if citations.Code != http.StatusOK || !strings.Contains(citations.Body.String(), "https://example.com/docs") {
		t.Fatalf("citations=%d %s", citations.Code, citations.Body.String())
	}
}

func TestP03ConversationMessagesAreOwnedAndOrdered(t *testing.T) {
	s, _ := NewStore("")
	createTestUser(t, s, "history@example.com")
	access := createAuthenticatedTestSession(t, s, "history@example.com")
	conversation, err := s.CreateConversation(access, "消息历史")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.AppendMessage(access, conversation.ID, "user", "第一条"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AppendMessage(access, conversation.ID, "assistant", "第二条"); err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	conversation = s.data.Conversations[conversation.ID]
	previousBranchID := conversation.ActiveBranchID
	activeBranchID := "active-history-branch"
	now := time.Now().UTC()
	s.data.ConversationBranches[activeBranchID] = ConversationBranch{
		ID: activeBranchID, ConversationID: conversation.ID, ParentBranchID: previousBranchID,
		Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	conversation.ActiveBranchID = activeBranchID
	s.data.Conversations[conversation.ID] = conversation
	s.mu.Unlock()
	if _, err := s.AppendMessage(access, conversation.ID, "user", "当前分支第一条"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AppendMessage(access, conversation.ID, "assistant", "当前分支第二条"); err != nil {
		t.Fatal(err)
	}
	api := NewAPI(s)
	response := requestJSON(t, api.Handler(), http.MethodGet, "/api/mobile/v1/conversations/"+conversation.ID+"/messages", nil, access, "")
	body := response.Body.String()
	if response.Code != http.StatusOK || !strings.Contains(body, "当前分支第一条") || !strings.Contains(body, "当前分支第二条") {
		t.Fatalf("messages=%d %s", response.Code, response.Body.String())
	}
	if strings.Contains(body, "\"body\":\"第一条\"") || strings.Contains(body, "\"body\":\"第二条\"") {
		t.Fatalf("inactive branch leaked into message history: %s", body)
	}
	if strings.Index(body, "当前分支第一条") > strings.Index(body, "当前分支第二条") {
		t.Fatalf("messages are not ordered by sequence: %s", body)
	}
	createTestUser(t, s, "history-other@example.com")
	other := createAuthenticatedTestSession(t, s, "history-other@example.com")
	denied := requestJSON(t, api.Handler(), http.MethodGet, "/api/mobile/v1/conversations/"+conversation.ID+"/messages", nil, other, "")
	if denied.Code != http.StatusNotFound {
		t.Fatalf("cross-user messages=%d %s", denied.Code, denied.Body.String())
	}
}

func TestP03HomeConfigComposerToolsDefaultAndValidation(t *testing.T) {
	s, _ := NewStore("")
	snapshot, err := s.HomeConfigSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	defaults := snapshot.ComposerTools
	if len(defaults) != 6 {
		t.Fatalf("default composer tools=%+v", defaults)
	}
	for _, tool := range defaults {
		if tool.Enabled {
			t.Fatalf("unsupported tool must default to disabled: %+v", tool)
		}
	}

	saved, err := s.UpdateHomeConfig(HomeConfig{}, "admin-test")
	if err != nil || len(saved.ComposerTools) != 6 {
		t.Fatalf("legacy home config default failed: config=%+v err=%v", saved, err)
	}

	valid := HomeConfig{ComposerTools: []ComposerToolConfig{
		{ID: "file", Label: "上传文件", Enabled: true, Prompt: "请分析我接下来上传的文件："},
	}}
	saved, err = s.UpdateHomeConfig(valid, "admin-test")
	if err != nil || len(saved.ComposerTools) != 1 || !saved.ComposerTools[0].Enabled {
		t.Fatalf("valid composer tool config failed: config=%+v err=%v", saved, err)
	}

	invalid := []HomeConfig{
		{ComposerTools: []ComposerToolConfig{{ID: "unknown", Label: "未知"}}},
		{ComposerTools: []ComposerToolConfig{{ID: "file", Label: "上传文件"}, {ID: "file", Label: "重复"}}},
		{ComposerTools: []ComposerToolConfig{{ID: "file", Label: " "}}},
	}
	for _, candidate := range invalid {
		if _, err := s.UpdateHomeConfig(candidate, "admin-test"); err == nil || err.Error() != "composer_tools_invalid" {
			t.Fatalf("invalid composer config accepted: config=%+v err=%v", candidate, err)
		}
	}

	copyCandidate := saved
	copyCandidate.ConsumerCopy = defaultConsumerCopy()
	copyCandidate.ConsumerCopy["thinking"] = "正在整理上下文"
	saved, err = s.UpdateHomeConfig(copyCandidate, "admin-test")
	if err != nil || saved.ConsumerCopy["thinking"] != "正在整理上下文" {
		t.Fatalf("consumer copy config failed: config=%+v err=%v", saved, err)
	}
	forbiddenCopy := saved
	forbiddenCopy.ConsumerCopy = defaultConsumerCopy()
	forbiddenCopy.ConsumerCopy["thinking"] = "provider error"
	if _, err := s.UpdateHomeConfig(forbiddenCopy, "admin-test"); err == nil || err.Error() != "consumer_copy_invalid" {
		t.Fatalf("forbidden consumer copy accepted: err=%v", err)
	}
}

func TestMobileAuthErrorDoesNotHideStorageFailuresAsMissingConversation(t *testing.T) {
	recorder := httptest.NewRecorder()
	if !mobileAuthError(recorder, errors.New("database unavailable")) {
		t.Fatal("storage error was not handled")
	}
	if recorder.Code != http.StatusInternalServerError || !strings.Contains(recorder.Body.String(), "service_unavailable") {
		t.Fatalf("storage error=%d %s", recorder.Code, recorder.Body.String())
	}
}

func TestHomeConfigSnapshotUsesExplicitErrorContract(t *testing.T) {
	s, err := NewStore("")
	if err != nil {
		t.Fatal(err)
	}
	config, err := s.HomeConfigSnapshot()
	if err != nil || config.Version != 1 {
		t.Fatalf("home config snapshot=%+v err=%v", config, err)
	}
}

func TestP03W04TemporaryFeedbackExportAndAsyncRegenerate(t *testing.T) {
	s, _ := NewStore("")
	createTestUser(t, s, "w04@example.com")
	access := createAuthenticatedTestSession(t, s, "w04@example.com")
	api := NewAPI(s)
	temporary := requestJSON(t, api.Handler(), http.MethodPost, "/api/mobile/v1/conversations?temporary=true", map[string]string{"title": "临时"}, access, "")
	if temporary.Code != http.StatusCreated || !strings.Contains(temporary.Body.String(), "temporary") {
		t.Fatalf("temporary=%d %s", temporary.Code, temporary.Body.String())
	}
	var conversation Conversation
	if err := json.Unmarshal(temporary.Body.Bytes(), &conversation); err != nil {
		t.Fatal(err)
	}
	message, err := s.AppendMessage(access, conversation.ID, "assistant", "回答")
	if err != nil {
		t.Fatal(err)
	}
	feedback := requestJSON(t, api.Handler(), http.MethodPost, "/api/mobile/v1/messages/"+message.ID+"/feedback", map[string]string{"value": "up"}, access, "")
	if feedback.Code != http.StatusOK || !strings.Contains(feedback.Body.String(), "saved") {
		t.Fatalf("feedback=%d %s", feedback.Code, feedback.Body.String())
	}
	export := requestJSON(t, api.Handler(), http.MethodPost, "/api/mobile/v1/messages/"+message.ID+"/exports", nil, access, "")
	if export.Code != http.StatusCreated || !strings.Contains(export.Body.String(), "markdown") {
		t.Fatalf("export=%d %s", export.Code, export.Body.String())
	}
	api.ChatRuntimeMode = "upstream"
	api.ChatResponder = staticChatResponder{value: "重答结果"}
	regenerated := requestJSON(t, api.Handler(), http.MethodPost, "/api/mobile/v1/messages/"+message.ID+"/regenerate", nil, access, "")
	if regenerated.Code != http.StatusAccepted {
		t.Fatalf("regenerate=%d %s", regenerated.Code, regenerated.Body.String())
	}
}

func TestP03W05SpeechOwnershipMetricsAndAdminDiagnostics(t *testing.T) {
	s, _ := NewStore("")
	createTestUser(t, s, "w05@example.com")
	access := createAuthenticatedTestSession(t, s, "w05@example.com")
	conversation, err := s.CreateConversation(access, "诊断")
	if err != nil {
		t.Fatal(err)
	}
	assistant, err := s.AppendMessage(access, conversation.ID, "assistant", "可朗读的回答")
	if err != nil {
		t.Fatal(err)
	}
	api := NewAPI(s)
	speech := requestJSON(t, api.Handler(), http.MethodPost, "/api/mobile/v1/messages/"+assistant.ID+"/speech", nil, access, "")
	if speech.Code != http.StatusAccepted || !strings.Contains(speech.Body.String(), "android_system_tts") {
		t.Fatalf("speech=%d %s", speech.Code, speech.Body.String())
	}
	other := createAuthenticatedTestSession(t, s, "other@example.com")
	if response := requestJSON(t, api.Handler(), http.MethodPost, "/api/mobile/v1/messages/"+assistant.ID+"/speech", nil, other, ""); response.Code != http.StatusNotFound {
		t.Fatalf("cross-user speech=%d", response.Code)
	}
	s.RecordMetric("chat.run", 0.12, "")
	if _, err := s.BootstrapAdmin("owner@example.com", "long admin password"); err != nil {
		t.Fatal(err)
	}
	_, adminToken, err := s.AdminLogin("owner@example.com", "long admin password")
	if err != nil {
		t.Fatal(err)
	}
	metrics := requestJSON(t, api.Handler(), http.MethodGet, "/internal/metrics/chat", nil, adminToken, "")
	if metrics.Code != http.StatusOK || !strings.Contains(metrics.Body.String(), "chat.run") {
		t.Fatalf("metrics=%d %s", metrics.Code, metrics.Body.String())
	}
	detail := requestJSON(t, api.Handler(), http.MethodGet, "/admin/v1/conversations/"+conversation.ID, nil, adminToken, "")
	if detail.Code != http.StatusOK || !strings.Contains(detail.Body.String(), conversation.ID) {
		t.Fatalf("detail=%d %s", detail.Code, detail.Body.String())
	}
}

type staticErrorChatResponder struct{}

func (staticErrorChatResponder) Respond(context.Context, ChatRequest) (ChatResponse, error) {
	return ChatResponse{}, errors.New("provider_secret_detail")
}

func TestP03GeneratedTitleUsesProviderAndPersists(t *testing.T) {
	s, _ := NewStore("")
	createTestUser(t, s, "title-runtime@example.com")
	access := createAuthenticatedTestSession(t, s, "title-runtime@example.com")
	conversation, err := s.CreateConversation(access, "")
	if err != nil {
		t.Fatal(err)
	}
	api := NewAPI(s)
	api.ChatRuntimeMode = "upstream"
	api.ChatResponder = staticChatResponder{value: "项目周报"}
	response := requestJSON(t, api.Handler(), http.MethodPost, "/internal/v1/conversations/"+conversation.ID+"/title", map[string]string{}, access, "")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "项目周报相关讨论") {
		t.Fatalf("title response=%d body=%s", response.Code, response.Body.String())
	}
	items, _, err := s.ListConversations(access, "", 10, false)
	if err != nil || len(items) != 1 || items[0].Title != "项目周报相关讨论" || len([]rune(items[0].Title)) < 6 || len([]rune(items[0].Title)) > 18 {
		t.Fatalf("persisted title=%+v err=%v", items, err)
	}
}

type staticChatResponder struct{ value string }

func (r staticChatResponder) Respond(context.Context, ChatRequest) (ChatResponse, error) {
	return ChatResponse{Content: r.value}, nil
}

func TestP03RunLifecycleEventsDraftAndExport(t *testing.T) {
	s, _ := NewStore("")
	createTestUser(t, s, "runtime@example.com")
	access := createAuthenticatedTestSession(t, s, "runtime@example.com")
	conversation, err := s.CreateConversation(access, "运行测试")
	if err != nil {
		t.Fatal(err)
	}
	run, err := s.CreateRun(access, conversation.ID, "请解释 SSE", "ylven-default", "SSE 会按事件 ID 恢复。")
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "completed" || run.AssistantMessageID == "" || run.Cursor < 2 {
		t.Fatalf("run=%+v", run)
	}
	_, events, err := s.Events(access, run.ID, 0)
	if err != nil || len(events) != int(run.Cursor) || events[len(events)-1].Type != "completed" {
		t.Fatalf("events=%+v err=%v", events, err)
	}
	_, resumed, err := s.Events(access, run.ID, events[0].ID)
	if err != nil || len(resumed) != len(events)-1 {
		t.Fatalf("resumed=%+v err=%v", resumed, err)
	}
	stored, messages, err := s.Run(access, run.ID)
	if err != nil || stored.ID != run.ID || len(messages) != 2 {
		t.Fatalf("run=%+v messages=%+v err=%v", stored, messages, err)
	}
	draft, err := s.SaveDraft(access, conversation.ID, "未发送内容")
	if err != nil || draft.Body != "未发送内容" {
		t.Fatalf("draft=%+v err=%v", draft, err)
	}
	loaded, err := s.GetDraft(access, conversation.ID)
	if err != nil || loaded.Body != draft.Body {
		t.Fatalf("loaded=%+v err=%v", loaded, err)
	}
	export, err := s.ExportConversation(access, conversation.ID, "")
	if err != nil || export.Status != "ready" || !strings.Contains(export.Content, "请解释 SSE") {
		t.Fatalf("export=%+v err=%v", export, err)
	}
}

func createAuthenticatedTestSession(t *testing.T, s *Store, email string) string {
	t.Helper()
	if email == "other@example.com" {
		createTestUser(t, s, email)
	}
	c, _ := s.CreateChallenge(email, "login")
	if err := s.VerifyTurnstile(c.ID, "token", true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateOTP(c.ID, "654321"); err != nil {
		t.Fatal(err)
	}
	if err := s.VerifyOTP(c.ID, "654321"); err != nil {
		t.Fatal(err)
	}
	_, access, _, err := s.CreateSession(c.ID, email)
	if err != nil {
		t.Fatal(err)
	}
	return access
}
