package identity

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCloudflareTurnstileVerifierChecksHostnameAndAction(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("secret") != "secret-ref" || r.Form.Get("response") != "valid-token" {
			t.Fatalf("unexpected siteverify payload: %#v", r.Form)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "hostname": "auth.orbexa.cc", "action": "register"})
	}))
	defer provider.Close()
	verifier := CloudflareTurnstileVerifier{
		Secret: "secret-ref", ExpectedHostname: "auth.orbexa.cc", Endpoint: provider.URL, Client: provider.Client(),
	}
	if err := verifier.Verify(context.Background(), "valid-token", "", "register"); err != nil {
		t.Fatal(err)
	}
	if err := verifier.Verify(context.Background(), "valid-token", "", "login"); err == nil {
		t.Fatal("action mismatch accepted")
	}
}

func TestOpenAICompatibleResponderMapsDefaultModelAndV1BaseURL(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected provider path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer provider-key" {
			t.Fatal("provider authorization header is missing")
		}
		var request struct {
			Model    string              `json:"model"`
			Messages []map[string]string `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.Model != "gpt-5.5" || len(request.Messages) != 1 || request.Messages[0]["content"] != "hello" {
			t.Fatalf("unexpected provider payload: %#v", request)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": "provider answer"}}}})
	}))
	defer provider.Close()

	responder := OpenAICompatibleResponder{
		Endpoint: provider.URL + "/v1", APIKey: "provider-key", DefaultModel: "gpt-5.5", Client: provider.Client(),
	}
	answer, err := responder.Respond(context.Background(), "ylven-default", "hello")
	if err != nil || answer != "provider answer" {
		t.Fatalf("answer=%q err=%v", answer, err)
	}
}

func TestOpenAICompatibleResponderRequiresConfiguredDefaultModel(t *testing.T) {
	responder := OpenAICompatibleResponder{Endpoint: "https://provider.example/v1", APIKey: "provider-key"}
	if _, err := responder.Respond(context.Background(), "", "hello"); err == nil || err.Error() != "chat_runtime_unavailable" {
		t.Fatalf("unexpected missing-model error: %v", err)
	}
}

func TestOpenAICompatibleResponderDefaultTimeoutFitsRunBudget(t *testing.T) {
	if upstreamChatHTTPTimeout != 90*time.Second {
		t.Fatalf("upstream timeout=%s", upstreamChatHTTPTimeout)
	}
	if upstreamChatHTTPTimeout >= chatRunTimeout {
		t.Fatalf("provider timeout %s must remain below run timeout %s", upstreamChatHTTPTimeout, chatRunTimeout)
	}
}

func TestNewAPIRejectsIncompleteUpstreamChatConfiguration(t *testing.T) {
	t.Setenv("TURNSTILE_MODE", "first_party")
	t.Setenv("EMAIL_MODE", "mock")
	t.Setenv("OWNER_ADMIN_EMAIL", "admin@ai-admin.orbexa.cc")
	t.Setenv("ADMIN_BOOTSTRAP_PASSWORD", "12345678")
	t.Setenv("CHAT_RUNTIME_MODE", "upstream")
	t.Setenv("SUB2API_ENDPOINT", "https://provider.example/v1")
	t.Setenv("SUB2API_API_KEY", "provider-key")
	t.Setenv("SUB2API_DEFAULT_MODEL", "")
	store, err := NewStore("")
	if err != nil {
		t.Fatal(err)
	}
	api := NewAPI(store)
	if api.ConfigurationOK() || api.ChatRuntimeMode != "unconfigured" || api.ChatResponder != nil {
		t.Fatalf("incomplete upstream configuration was accepted: mode=%s responder=%T", api.ChatRuntimeMode, api.ChatResponder)
	}
}

func TestNewAPIConfiguresUpstreamDefaultModel(t *testing.T) {
	t.Setenv("TURNSTILE_MODE", "first_party")
	t.Setenv("EMAIL_MODE", "mock")
	t.Setenv("OWNER_ADMIN_EMAIL", "admin@ai-admin.orbexa.cc")
	t.Setenv("ADMIN_BOOTSTRAP_PASSWORD", "12345678")
	t.Setenv("CHAT_RUNTIME_MODE", "upstream")
	t.Setenv("SUB2API_ENDPOINT", "https://provider.example/v1")
	t.Setenv("SUB2API_API_KEY", "provider-key")
	t.Setenv("SUB2API_DEFAULT_MODEL", "gpt-5.5")
	store, err := NewStore("")
	if err != nil {
		t.Fatal(err)
	}
	api := NewAPI(store)
	responder, ok := api.ChatResponder.(OpenAICompatibleResponder)
	if !api.ConfigurationOK() || api.ChatRuntimeMode != "upstream" || !ok || responder.DefaultModel != "gpt-5.5" {
		t.Fatalf("valid upstream configuration was rejected: ready=%t mode=%s responder=%#v", api.ConfigurationOK(), api.ChatRuntimeMode, api.ChatResponder)
	}
}

func TestHTTPRegistrationUsesExplicitStagingMocks(t *testing.T) {
	t.Setenv("TURNSTILE_MODE", "mock")
	t.Setenv("TURNSTILE_MOCK_TOKEN", "test-pass")
	t.Setenv("EMAIL_MODE", "mock")
	t.Setenv("IDENTITY_DEBUG_OTP", "1")
	t.Setenv("OWNER_ADMIN_EMAIL", "admin@ai-admin.orbexa.cc")
	t.Setenv("ADMIN_BOOTSTRAP_PASSWORD", "12345678")
	store, err := NewStore("")
	if err != nil {
		t.Fatal(err)
	}
	api := NewAPI(store)
	if !api.ConfigurationOK() {
		t.Fatal("explicit staging mock configuration was not ready")
	}
	handler := api.Handler()
	challenge := requestJSON(t, handler, http.MethodPost, "/api/v1/auth/register/challenge", map[string]string{
		"email": "staging@example.com", "purpose": "register", "turnstile_token": "test-pass",
	}, "", "")
	if challenge.Code != http.StatusCreated {
		t.Fatalf("challenge status=%d body=%s", challenge.Code, challenge.Body.String())
	}
	var challengeBody struct {
		ChallengeID string `json:"challenge_id"`
	}
	if err := json.Unmarshal(challenge.Body.Bytes(), &challengeBody); err != nil {
		t.Fatal(err)
	}
	delivery := requestJSON(t, handler, http.MethodPost, "/api/v1/auth/register/otp/send", map[string]string{"challenge_id": challengeBody.ChallengeID}, "", "")
	if delivery.Code != http.StatusAccepted {
		t.Fatalf("delivery status=%d body=%s", delivery.Code, delivery.Body.String())
	}
	var deliveryBody struct {
		DebugCode string `json:"debug_code"`
	}
	if err := json.Unmarshal(delivery.Body.Bytes(), &deliveryBody); err != nil {
		t.Fatal(err)
	}
	if len(deliveryBody.DebugCode) != 6 {
		t.Fatalf("debug OTP missing: %q", deliveryBody.DebugCode)
	}
	verified := requestJSON(t, handler, http.MethodPost, "/api/v1/auth/register/otp/verify", map[string]string{"challenge_id": challengeBody.ChallengeID, "code": deliveryBody.DebugCode}, "", "")
	if verified.Code != http.StatusOK {
		t.Fatalf("verify status=%d body=%s", verified.Code, verified.Body.String())
	}
}

func TestFirstPartyVerificationEndpointDoesNotExposeAnswer(t *testing.T) {
	store, err := NewStore("")
	if err != nil {
		t.Fatal(err)
	}
	api := &API{Store: store}
	handler := api.Handler()
	created := requestJSON(t, handler, http.MethodPost, "/api/v1/auth/login/challenge", map[string]string{"email": "answer@example.com"}, "", "")
	if created.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	var body struct {
		ChallengeID string `json:"challenge_id"`
		Question    string `json:"verification_question"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.ChallengeID == "" || body.Question == "" || strings.Contains(created.Body.String(), "verification_digest") {
		t.Fatalf("unsafe response: %s", created.Body.String())
	}
	challenge, ok := store.ChallengeSnapshot(body.ChallengeID)
	if !ok {
		t.Fatal("challenge was not persisted")
	}
	var left, right int
	if _, err := fmt.Sscanf(challenge.VerificationQuestion, "%d + %d = ?", &left, &right); err != nil {
		t.Fatal(err)
	}
	verified := requestJSON(t, handler, http.MethodPost, "/api/v1/auth/challenge/verify", map[string]string{"challenge_id": challenge.ID, "answer": fmt.Sprint(left + right)}, "", "")
	if verified.Code != http.StatusOK {
		t.Fatalf("verify status=%d body=%s", verified.Code, verified.Body.String())
	}
	legacy := httptest.NewRecorder()
	handler.ServeHTTP(legacy, httptest.NewRequest(http.MethodGet, "/security/turnstile?challenge_id="+challenge.ID, nil))
	if legacy.Code != http.StatusNotFound {
		t.Fatalf("legacy page still exposed: %d", legacy.Code)
	}
}

func TestFirstPartyVerificationLocksAfterFiveFailures(t *testing.T) {
	store, err := NewStore("")
	if err != nil {
		t.Fatal(err)
	}
	challenge, err := store.CreateChallenge("limited@example.com", "login")
	if err != nil {
		t.Fatal(err)
	}
	api := &API{Store: store}
	handler := api.Handler()
	for attempt := 0; attempt < 5; attempt++ {
		failed := requestJSON(t, handler, http.MethodPost, "/api/v1/auth/challenge/verify", map[string]string{"challenge_id": challenge.ID, "answer": "999"}, "", "")
		if failed.Code != http.StatusUnprocessableEntity {
			t.Fatalf("attempt %d status=%d", attempt+1, failed.Code)
		}
	}
	locked := requestJSON(t, handler, http.MethodPost, "/api/v1/auth/challenge/verify", map[string]string{"challenge_id": challenge.ID, "answer": "0"}, "", "")
	if locked.Code != http.StatusTooManyRequests {
		t.Fatalf("locked status=%d body=%s", locked.Code, locked.Body.String())
	}
}
