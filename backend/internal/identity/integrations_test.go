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
