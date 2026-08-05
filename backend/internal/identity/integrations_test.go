package identity

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
