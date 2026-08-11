package identity

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type API struct {
	Store             *Store
	TurnstileMode     string
	TurnstileSiteKey  string
	TurnstileVerifier TurnstileVerifier
	EmailMode         string
	Mailer            OTPMailer
	DebugOTP          bool
	AdminEmail        string
	AdminUsername     string
	BootstrapError    error
	configurationErrs []string
	ChatRuntimeMode   string
	ChatResponder     ChatResponder
	runMu             sync.Mutex
	runCancels        map[string]context.CancelFunc
}

// Keep the run deadline above the provider HTTP deadline. This admits the
// observed slow but successful upstream responses while preserving a bound.
const chatRunTimeout = 95 * time.Second

func NewAPI(store *Store) *API {
	api := &API{Store: store, runCancels: map[string]context.CancelFunc{}}
	api.ChatRuntimeMode = strings.ToLower(strings.TrimSpace(os.Getenv("CHAT_RUNTIME_MODE")))
	if api.ChatRuntimeMode == "" {
		api.ChatRuntimeMode = "unconfigured"
	}
	if api.ChatRuntimeMode == "upstream" {
		endpoint, endpointErr := envOrFile("SUB2API_ENDPOINT")
		apiKey, apiKeyErr := envOrFile("SUB2API_API_KEY")
		defaultModel, defaultModelErr := envOrFile("SUB2API_DEFAULT_MODEL")
		_, endpointParseErr := openAIChatCompletionsEndpoint(endpoint)
		if endpointErr != nil || endpointParseErr != nil || apiKeyErr != nil || defaultModelErr != nil || endpoint == "" || apiKey == "" || defaultModel == "" {
			api.configurationErrs = append(api.configurationErrs, "upstream chat runtime is not configured")
			api.ChatRuntimeMode = "unconfigured"
		} else {
			api.ChatResponder = OpenAICompatibleResponder{Endpoint: endpoint, APIKey: apiKey, DefaultModel: defaultModel}
		}
	} else if api.ChatRuntimeMode != "unconfigured" {
		api.configurationErrs = append(api.configurationErrs, "invalid chat runtime mode")
		api.ChatRuntimeMode = "unconfigured"
	}
	api.TurnstileSiteKey = strings.TrimSpace(os.Getenv("TURNSTILE_SITE_KEY"))
	api.TurnstileMode = strings.ToLower(strings.TrimSpace(os.Getenv("TURNSTILE_MODE")))
	// P02 uses YLVEN's own short arithmetic check. The legacy fields and
	// endpoint remain readable for old data, but no hosted widget is loaded.
	if api.TurnstileMode == "" || api.TurnstileMode == "external" {
		api.TurnstileMode = "first_party"
	}
	switch api.TurnstileMode {
	case "first_party":
		// No external secret or browser widget is required.
	case "mock":
		mockToken, err := envOrFile("TURNSTILE_MOCK_TOKEN")
		if err != nil || mockToken == "" {
			api.configurationErrs = append(api.configurationErrs, "turnstile mock token is not configured")
		} else {
			api.TurnstileVerifier = MockTurnstileVerifier{Token: mockToken}
		}
	case "external":
		if api.TurnstileSiteKey == "" {
			api.configurationErrs = append(api.configurationErrs, "Turnstile site key is not configured")
		}
		secret, err := envOrFile("TURNSTILE_SECRET")
		if err != nil || secret == "" {
			api.configurationErrs = append(api.configurationErrs, "Turnstile secret is not configured")
		} else {
			api.TurnstileVerifier = CloudflareTurnstileVerifier{
				Secret:           secret,
				ExpectedHostname: strings.TrimSpace(os.Getenv("TURNSTILE_EXPECTED_HOSTNAME")),
				Endpoint:         strings.TrimSpace(os.Getenv("TURNSTILE_SITEVERIFY_URL")),
			}
		}
	default:
		api.configurationErrs = append(api.configurationErrs, "invalid Turnstile mode")
	}

	api.EmailMode = strings.ToLower(strings.TrimSpace(os.Getenv("EMAIL_MODE")))
	if api.EmailMode == "" {
		api.EmailMode = "smtp"
	}
	switch api.EmailMode {
	case "mock":
		// Mock delivery is permitted only in an explicitly labelled staging environment.
	case "smtp":
		password, err := envOrFile("SMTP_PASSWORD")
		port := 587
		if raw := strings.TrimSpace(os.Getenv("SMTP_PORT")); raw != "" {
			parsed, parseErr := strconv.Atoi(raw)
			if parseErr != nil {
				api.configurationErrs = append(api.configurationErrs, "invalid SMTP port")
			} else {
				port = parsed
			}
		}
		mailer := SMTPMailer{
			Host:     strings.TrimSpace(os.Getenv("SMTP_HOST")),
			Port:     port,
			Username: strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
			Password: password,
			From:     strings.TrimSpace(os.Getenv("EMAIL_FROM_ADDRESS")),
			FromName: strings.TrimSpace(os.Getenv("EMAIL_FROM_NAME")),
		}
		if err != nil || mailer.Host == "" || mailer.From == "" {
			api.configurationErrs = append(api.configurationErrs, "SMTP delivery is not configured")
		} else {
			api.Mailer = mailer
		}
	default:
		api.configurationErrs = append(api.configurationErrs, "invalid email mode")
	}
	api.DebugOTP = os.Getenv("IDENTITY_DEBUG_OTP") == "1" && api.EmailMode == "mock"

	api.AdminUsername = strings.TrimSpace(os.Getenv("OWNER_ADMIN_USERNAME"))
	if api.AdminUsername == "" {
		api.AdminUsername = "admin"
	}
	api.AdminEmail = strings.TrimSpace(os.Getenv("OWNER_ADMIN_EMAIL"))
	password, passwordErr := envOrFile("ADMIN_BOOTSTRAP_PASSWORD")
	if passwordErr != nil {
		api.BootstrapError = passwordErr
	} else if api.AdminEmail == "" || password == "" {
		api.BootstrapError = errors.New("administrator bootstrap is not configured")
	} else {
		_, api.BootstrapError = store.BootstrapAdmin(api.AdminEmail, password)
	}
	if api.BootstrapError != nil {
		api.configurationErrs = append(api.configurationErrs, "administrator bootstrap is invalid")
	}
	return api
}

func (a *API) ConfigurationOK() bool { return len(a.configurationErrs) == 0 }

func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/email/normalize", a.normalize)
	mux.HandleFunc("/api/v1/auth/register/challenge", a.challenge)
	mux.HandleFunc("/api/v1/auth/challenge/verify", a.verifyAnswer)
	mux.HandleFunc("/api/v1/auth/register/otp/send", a.otpSend)
	mux.HandleFunc("/api/v1/auth/register/otp/verify", a.otpVerify)
	mux.HandleFunc("/api/v1/auth/register/complete", a.complete)
	mux.HandleFunc("/api/v1/auth/login/challenge", a.loginChallenge)
	mux.HandleFunc("/api/v1/auth/login/otp/send", a.loginOTPSend)
	mux.HandleFunc("/api/v1/auth/login/otp/verify", a.loginOTPVerify)
	mux.HandleFunc("/api/v1/auth/password-policy", a.passwordPolicy)
	mux.HandleFunc("/api/v1/auth/sessions", a.sessions)
	mux.HandleFunc("/api/v1/onboarding/personal-workspace", a.personalWorkspace)
	mux.HandleFunc("/api/v1/account/session", a.accountSession)
	mux.HandleFunc("/api/v1/auth/sessions/refresh", a.refreshSession)
	mux.HandleFunc("/api/v1/auth/token/refresh", a.refreshSession)
	mux.HandleFunc("/api/v1/auth/logout", a.logout)
	mux.HandleFunc("/api/v1/auth/logout-all", a.logoutAll)
	mux.HandleFunc("/api/v1/account/devices", a.devices)
	mux.HandleFunc("/api/v1/account/devices/", a.deviceSession)
	mux.HandleFunc("/api/mobile/v1/home", a.mobileHome)
	mux.HandleFunc("/api/mobile/v1/conversations/search", a.mobileConversationSearch)
	mux.HandleFunc("/api/mobile/v1/conversations/from-first-message", a.mobileConversationFromFirstMessage)
	mux.HandleFunc("/api/mobile/v1/conversations/", a.mobileConversationByID)
	mux.HandleFunc("/api/mobile/v1/conversations", a.mobileConversations)
	mux.HandleFunc("/api/mobile/v1/runs/", a.mobileRunByID)
	mux.HandleFunc("/api/mobile/v1/messages/", a.mobileMessageByID)
	mux.HandleFunc("/internal/v1/conversations/", a.internalConversationByID)
	mux.HandleFunc("/internal/v1/context/build", a.internalContextBuild)
	mux.HandleFunc("/internal/v1/context/recompile", a.internalContextBuild)
	mux.HandleFunc("/internal/v1/provider-state/fallback", a.internalProviderFallback)
	mux.HandleFunc("/internal/metrics/chat", a.chatMetrics)
	mux.HandleFunc("/admin/v1/settings/email", a.adminSetting("email"))
	mux.HandleFunc("/admin/v1/settings/turnstile", a.adminSetting("turnstile"))
	mux.HandleFunc("/admin/v1/settings/otp-policy", a.adminSetting("otp-policy"))
	mux.HandleFunc("/admin/v1/users", a.adminUsers)
	mux.HandleFunc("/admin/v1/users/", a.adminUserDetail)
	mux.HandleFunc("/admin/v1/rbac/roles", a.adminRoles)
	mux.HandleFunc("/admin/v1/auth/login", a.adminLogin)
	mux.HandleFunc("/admin/v1/auth/logout", a.adminLogout)
	mux.HandleFunc("/admin/v1/auth/sessions", a.adminSessions)
	mux.HandleFunc("/admin/v1/security/step-up", a.adminStepUp)
	mux.HandleFunc("/admin/v1/notifications/email-templates", a.adminEmailTemplates)
	mux.HandleFunc("/admin/v1/content/home", a.adminHomeConfig)
	mux.HandleFunc("/admin/v1/conversations", a.adminConversations)
	mux.HandleFunc("/admin/v1/conversations/", a.adminConversationDetail)
	return requestGuard(mux)
}

func (a *API) mobileConversationFromFirstMessage(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if a.ChatRuntimeMode != "upstream" || a.ChatResponder == nil {
		writeError(w, http.StatusServiceUnavailable, "chat_runtime_unavailable", "AI runtime is unavailable")
		return
	}
	var in struct {
		DraftSessionID string `json:"draft_session_id"`
		Body           string `json:"body"`
		Model          string `json:"model"`
		IdempotencyKey string `json:"idempotency_key"`
		Temporary      bool   `json:"temporary"`
	}
	if !decode(r, &in) {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON")
		return
	}
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" {
		key = strings.TrimSpace(in.IdempotencyKey)
	}
	result, err := a.Store.StartConversationFromFirstMessage(bearer(r), in.DraftSessionID, in.Body, in.Model, key, in.Temporary)
	if err != nil {
		a.writeConversationRunError(w, err)
		return
	}
	if result.Created {
		a.dispatchRun(result.Run, bearer(r), true)
	}
	writeJSON(w, http.StatusAccepted, result)
}

func requestGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

func requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}
	w.Header().Set("Allow", method)
	writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", method+" required")
	return false
}

func decode(r *http.Request, dst any) bool {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst) == nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	if friendly, ok := map[string]string{
		"email_already_registered": "这个邮箱已经注册过了，可以直接登录",
		"invalid_email":            "请输入正确的邮箱地址",
		"otp_invalid":              "验证码不正确，请重新输入",
		"otp_expired_or_locked":    "验证码已过期，请重新获取",
		"auth_rate_limited":        "操作太频繁了，请稍后再试",
		"session_invalid":          "登录状态已失效，请重新登录",
		"email_delivery_failed":    "验证码暂时发送失败，请稍后再试",
		"turnstile_failed":         "验证没有完成，请再试一次",
		"verification_incorrect":   "答案不正确，请再试一次",
		"verification_locked":      "尝试次数过多，请重新开始",
		"challenge_expired":        "验证已过期，请重新开始",
	}[code]; ok {
		message = friendly
	}
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

var turnstilePageTemplate = template.Must(template.New("turnstile").Parse(`<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1,maximum-scale=1">
  <title>YLVEN 安全验证</title>
  <style nonce="{{.Nonce}}">
    :root { color-scheme: light; font-family: system-ui, sans-serif; color: #101828; background: #fff; }
    body { margin: 0; min-height: 100vh; display: grid; place-items: center; }
    main { width: min(100% - 32px, 360px); text-align: center; }
    h1 { margin: 0 0 12px; font-size: 22px; line-height: 30px; }
    p { margin: 0 0 24px; color: #475467; font-size: 14px; line-height: 20px; }
    .widget { min-height: 70px; display: grid; place-items: center; }
    #status { margin-top: 16px; min-height: 20px; }
    button { margin-top: 12px; min-height: 44px; padding: 0 20px; border: 1px solid #e4e7ec; border-radius: 12px; background: #fff; color: #475467; font-size: 14px; }
    [hidden] { display: none; }
  </style>
</head>
<body>
  <main>
    <h1>完成安全验证</h1>
    <p>验证令牌仅用于当前登录或注册请求。</p>
    <div class="widget">
      <div class="cf-turnstile" data-sitekey="{{.SiteKey}}" data-action="{{.Action}}" data-callback="turnstileSuccess" data-error-callback="turnstileError" data-expired-callback="turnstileExpired"></div>
    </div>
    <p id="status" role="status" aria-live="polite">正在加载安全验证...</p>
    <button id="retry" type="button" hidden>重新验证</button>
  </main>
  <script nonce="{{.Nonce}}">
    const statusNode = document.getElementById('status');
    const retryNode = document.getElementById('retry');
    window.turnstileSuccess = function(token) {
      statusNode.textContent = '验证已通过，正在返回应用...';
      retryNode.hidden = true;
      if (window.YlvenSecurity && typeof window.YlvenSecurity.onTurnstileToken === 'function') {
        window.YlvenSecurity.onTurnstileToken(token);
      } else {
        statusNode.textContent = '验证已通过，请返回 YLVEN 应用。';
      }
    };
    window.turnstileError = function() {
      statusNode.textContent = '安全验证暂时无法完成，请重试。';
      retryNode.hidden = false;
    };
    window.turnstileExpired = function() {
      statusNode.textContent = '验证已过期，请重新验证。';
      retryNode.hidden = false;
    };
    retryNode.addEventListener('click', function() {
      retryNode.hidden = true;
      statusNode.textContent = '正在重新加载安全验证...';
      if (window.turnstile) window.turnstile.reset();
    });
  </script>
  <script src="https://challenges.cloudflare.com/turnstile/v0/api.js"></script>
</body>
</html>`))

func (a *API) turnstilePage(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	challengeID := strings.TrimSpace(r.URL.Query().Get("challenge_id"))
	action := strings.TrimSpace(r.URL.Query().Get("action"))
	challenge, ok := a.Store.ChallengeSnapshot(challengeID)
	if !ok || challenge.ExpiresAt.Before(time.Now()) || challenge.Consumed {
		writeError(w, http.StatusNotFound, "challenge_not_found", "Security challenge is unavailable")
		return
	}
	if (action != "login" && action != "register") || action != challenge.Purpose {
		writeError(w, http.StatusUnprocessableEntity, "challenge_action_mismatch", "Security challenge action is invalid")
		return
	}
	if challenge.TurnstileVerified {
		writeError(w, http.StatusConflict, "turnstile_already_verified", "Security challenge was already verified")
		return
	}
	if a.TurnstileSiteKey == "" {
		writeError(w, http.StatusServiceUnavailable, "turnstile_unconfigured", "Security verification is unavailable")
		return
	}
	nonceBytes := make([]byte, 18)
	if _, err := rand.Read(nonceBytes); err != nil {
		writeError(w, http.StatusServiceUnavailable, "turnstile_unavailable", "Security verification is unavailable")
		return
	}
	nonce := fmt.Sprintf("%x", nonceBytes)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'nonce-"+nonce+"' https://challenges.cloudflare.com; style-src 'nonce-"+nonce+"'; frame-src https://challenges.cloudflare.com; connect-src https://challenges.cloudflare.com; img-src data:; base-uri 'none'; frame-ancestors 'none'; form-action 'none'")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
	if err := turnstilePageTemplate.Execute(w, map[string]string{"SiteKey": a.TurnstileSiteKey, "Action": action, "Nonce": nonce}); err != nil {
		return
	}
}

func (a *API) normalize(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var in struct {
		Email string `json:"email"`
	}
	if !decode(r, &in) {
		writeError(w, 400, "invalid_json", "Invalid JSON")
		return
	}
	email, err := NormalizeEmail(in.Email)
	if err != nil {
		writeError(w, 422, "invalid_email", err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"email": email, "valid": true})
}

func (a *API) verifyTurnstile(r *http.Request, challengeID, token, action string) error {
	if strings.TrimSpace(token) == "" {
		return errors.New("turnstile_token_required")
	}
	if a.TurnstileVerifier == nil {
		return errors.New("turnstile_unconfigured")
	}
	if err := a.TurnstileVerifier.Verify(r.Context(), token, "", action); err != nil {
		return err
	}
	return a.Store.VerifyTurnstile(challengeID, token, true)
}

func (a *API) challenge(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var in struct {
		Email          string `json:"email"`
		Purpose        string `json:"purpose"`
		TurnstileToken string `json:"turnstile_token"`
	}
	if !decode(r, &in) {
		writeError(w, 400, "invalid_json", "Invalid JSON")
		return
	}
	if in.Purpose == "" {
		in.Purpose = "register"
	}
	if in.Purpose != "register" {
		writeError(w, 422, "invalid_purpose", "Invalid challenge purpose")
		return
	}
	challenge, err := a.Store.CreateChallenge(in.Email, in.Purpose)
	if err != nil {
		writeError(w, 422, "invalid_email", err.Error())
		return
	}
	if in.TurnstileToken != "" {
		if err := a.verifyTurnstile(r, challenge.ID, in.TurnstileToken, "register"); err != nil {
			writeError(w, 422, "turnstile_failed", "Security verification failed")
			return
		}
	}
	writeJSON(w, 201, map[string]any{"challenge_id": challenge.ID, "email": challenge.Email, "verification_question": challenge.VerificationQuestion, "expires_at": challenge.ExpiresAt.UTC()})
}

func (a *API) verifyAnswer(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var in struct {
		ChallengeID string `json:"challenge_id"`
		Answer      string `json:"answer"`
	}
	if !decode(r, &in) {
		writeError(w, 400, "invalid_json", "请求内容不完整，请再试一次")
		return
	}
	if strings.TrimSpace(in.Answer) == "" {
		writeError(w, 422, "verification_incorrect", "请输入答案")
		return
	}
	if err := a.Store.VerifyAnswer(in.ChallengeID, in.Answer); err != nil {
		status := 422
		if err.Error() == "challenge_not_found" || err.Error() == "challenge_expired" {
			status = 410
		}
		if err.Error() == "verification_locked" {
			status = 429
		}
		message := map[string]string{"verification_incorrect": "答案不正确，请再试一次", "verification_locked": "尝试次数过多，请重新开始", "challenge_expired": "验证已过期，请重新开始", "challenge_consumed": "验证已完成，请继续操作"}[err.Error()]
		if message == "" {
			message = "验证没有完成，请再试一次"
		}
		writeError(w, status, err.Error(), message)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"verified": true})
}

func (a *API) turnstile(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var in struct {
		ChallengeID string `json:"challenge_id"`
		Token       string `json:"token"`
	}
	if !decode(r, &in) {
		writeError(w, 400, "invalid_json", "Invalid JSON")
		return
	}
	challenge, ok := a.Store.ChallengeSnapshot(in.ChallengeID)
	if !ok {
		writeError(w, 404, "challenge_not_found", "Challenge not found")
		return
	}
	if err := a.verifyTurnstile(r, challenge.ID, in.Token, challenge.Purpose); err != nil {
		status := 422
		if strings.Contains(err.Error(), "unconfigured") || strings.Contains(err.Error(), "unavailable") {
			status = 503
		}
		writeError(w, status, "turnstile_failed", "Security verification failed")
		return
	}
	writeJSON(w, 200, map[string]any{"verified": true})
}

func (a *API) otpSend(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var in struct {
		ChallengeID string `json:"challenge_id"`
	}
	if !decode(r, &in) {
		writeError(w, 400, "invalid_json", "Invalid JSON")
		return
	}
	codeBytes := make([]byte, 4)
	if _, err := rand.Read(codeBytes); err != nil {
		writeError(w, 500, "randomness_failed", "Unable to create OTP")
		return
	}
	code := fmt.Sprintf("%06d", (int(codeBytes[0])<<16|int(codeBytes[1])<<8|int(codeBytes[2]))%1000000)
	otp, err := a.Store.CreateOTP(in.ChallengeID, code)
	if err != nil {
		status := 422
		if err.Error() == "turnstile_required" {
			status = 403
		}
		if err.Error() == "otp_cooldown" {
			status = 429
		}
		writeError(w, status, err.Error(), "Challenge is not ready for email verification")
		return
	}
	challenge, ok := a.Store.ChallengeSnapshot(in.ChallengeID)
	if !ok {
		writeError(w, 422, "challenge_invalid", "Challenge not found")
		return
	}
	templateKey := "register_otp"
	if challenge.Purpose == "login" {
		templateKey = "login_otp"
	}
	if a.EmailMode == "mock" {
		if err := a.Store.UpdateNotificationDelivery(otp.DeliveryID, "mocked"); err != nil {
			writeError(w, 500, "delivery_state_failed", "Unable to record delivery")
			return
		}
	} else {
		template, found := a.Store.EmailTemplateFor(templateKey)
		if !found || a.Mailer == nil {
			_ = a.Store.UpdateNotificationDelivery(otp.DeliveryID, "failed")
			writeError(w, 503, "email_unconfigured", "Email delivery is not configured")
			return
		}
		body := strings.ReplaceAll(template.Body, "{{code}}", code)
		if err := a.Mailer.SendOTP(r.Context(), otp.Email, template.Subject, body, challenge.Purpose); err != nil {
			_ = a.Store.UpdateNotificationDelivery(otp.DeliveryID, "failed")
			writeError(w, 503, "email_delivery_failed", "Unable to deliver verification email")
			return
		}
		if err := a.Store.UpdateNotificationDelivery(otp.DeliveryID, "delivered"); err != nil {
			writeError(w, 500, "delivery_state_failed", "Unable to record delivery")
			return
		}
	}
	out := map[string]any{"delivery_id": otp.DeliveryID, "expires_at": otp.ExpiresAt.UTC(), "resend_after_seconds": 60}
	if a.DebugOTP {
		out["debug_code"] = code
	}
	writeJSON(w, 202, out)
}

func (a *API) otpVerify(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var in struct {
		ChallengeID string `json:"challenge_id"`
		Code        string `json:"code"`
	}
	if !decode(r, &in) {
		writeError(w, 400, "invalid_json", "Invalid JSON")
		return
	}
	if err := a.Store.VerifyOTP(in.ChallengeID, in.Code); err != nil {
		writeError(w, 422, "otp_invalid", err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"verified": true})
}

func (a *API) complete(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var in struct {
		ChallengeID string `json:"challenge_id"`
		Email       string `json:"email"`
		Password    string `json:"password"`
		DeviceID    string `json:"device_id"`
	}
	if !decode(r, &in) {
		writeError(w, 400, "invalid_json", "Invalid JSON")
		return
	}
	user, err := a.Store.CreateUser(in.ChallengeID, in.Email, in.Password)
	if err != nil {
		status := 422
		if strings.Contains(err.Error(), "already_registered") {
			status = 409
		}
		writeError(w, status, err.Error(), err.Error())
		return
	}
	workspace, workspaceErr := a.Store.EnsureWorkspace(user.ID)
	if workspaceErr != nil {
		writeError(w, http.StatusInternalServerError, "workspace_unavailable", "Workspace could not be initialized")
		return
	}
	session, access, refresh, sessionErr := a.Store.CreateSessionForUser(user.Email, in.DeviceID)
	if sessionErr != nil {
		writeError(w, http.StatusInternalServerError, "session_unavailable", "Session could not be created")
		return
	}
	writeJSON(w, 201, map[string]any{
		"user_id": user.ID, "email": user.Email, "workspace": workspace,
		"session_id": session.ID, "device_id": session.DeviceID, "access_token": access, "refresh_token": refresh,
		"access_expires_at": session.AccessExpiresAt, "refresh_expires_at": session.RefreshExpiresAt,
	})
}

func (a *API) passwordPolicy(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"min_length": 8, "requires_letter": true, "requires_digit": true})
}

func (a *API) personalWorkspace(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	user, _, err := a.Store.CurrentAccount(bearer(r))
	if err != nil {
		writeError(w, http.StatusUnauthorized, "session_invalid", "Session is invalid")
		return
	}
	workspace, err := a.Store.EnsureWorkspace(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "workspace_unavailable", "Workspace could not be initialized")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"workspace": workspace, "initialized": true})
}

func (a *API) loginChallenge(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var in struct {
		Email          string `json:"email"`
		TurnstileToken string `json:"turnstile_token"`
	}
	if !decode(r, &in) {
		writeError(w, 400, "invalid_json", "Invalid JSON")
		return
	}
	normalized, err := NormalizeEmail(in.Email)
	if err != nil {
		writeJSON(w, 202, map[string]any{"accepted": true})
		return
	}
	allowed, _ := a.Store.AllowAttempt(normalized, 5, 10*time.Minute)
	if !allowed {
		writeError(w, 429, "auth_rate_limited", "Try again later")
		return
	}
	challenge, err := a.Store.CreateChallenge(normalized, "login")
	if err != nil {
		writeJSON(w, 202, map[string]any{"accepted": true})
		return
	}
	if in.TurnstileToken != "" {
		if err := a.verifyTurnstile(r, challenge.ID, in.TurnstileToken, "login"); err != nil {
			writeError(w, 422, "turnstile_failed", "Security verification failed")
			return
		}
	}
	writeJSON(w, 201, map[string]any{"challenge_id": challenge.ID, "accepted": true, "verification_question": challenge.VerificationQuestion, "expires_at": challenge.ExpiresAt.UTC()})
}

func (a *API) loginOTPSend(w http.ResponseWriter, r *http.Request)   { a.otpSend(w, r) }
func (a *API) loginOTPVerify(w http.ResponseWriter, r *http.Request) { a.otpVerify(w, r) }

func (a *API) sessions(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		a.accountSession(w, r)
		return
	}
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var in struct {
		ChallengeID string `json:"challenge_id"`
		Email       string `json:"email"`
		DeviceID    string `json:"device_id"`
	}
	if !decode(r, &in) {
		writeError(w, 400, "invalid_json", "Invalid JSON")
		return
	}
	session, access, refresh, err := a.Store.CreateSession(in.ChallengeID, in.Email, in.DeviceID)
	if err != nil {
		writeError(w, 401, "login_not_verified", "Unable to authenticate")
		return
	}
	writeJSON(w, 201, map[string]any{"session_id": session.ID, "device_id": session.DeviceID, "access_token": access, "refresh_token": refresh, "access_expires_at": session.AccessExpiresAt, "refresh_expires_at": session.RefreshExpiresAt})
}

func (a *API) accountSession(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	user, session, err := a.Store.CurrentAccount(bearer(r))
	if err != nil {
		writeError(w, http.StatusUnauthorized, "session_invalid", "Session is invalid")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": user, "session": session, "authenticated": true})
}

func (a *API) refreshSession(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var in struct {
		RefreshToken string `json:"refresh_token"`
	}
	if !decode(r, &in) {
		writeError(w, 400, "invalid_json", "Invalid JSON")
		return
	}
	session, access, refresh, err := a.Store.RotateSession(in.RefreshToken)
	if err != nil {
		writeError(w, 401, "refresh_token_invalid", "Refresh token is invalid or expired")
		return
	}
	writeJSON(w, 200, map[string]any{"session_id": session.ID, "device_id": session.DeviceID, "access_token": access, "refresh_token": refresh, "access_expires_at": session.AccessExpiresAt, "refresh_expires_at": session.RefreshExpiresAt})
}

func bearer(r *http.Request) string {
	value := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(value) > 7 && strings.EqualFold(value[:7], "Bearer ") {
		return strings.TrimSpace(value[7:])
	}
	return ""
}

func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if err := a.Store.Logout(bearer(r)); err != nil {
		writeError(w, 401, "session_invalid", "Session is invalid")
		return
	}
	writeJSON(w, 200, map[string]any{"logged_out": true})
}

func (a *API) logoutAll(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if err := a.Store.LogoutAll(bearer(r)); err != nil {
		writeError(w, 401, "session_invalid", "Session is invalid")
		return
	}
	writeJSON(w, 200, map[string]any{"logged_out_all": true})
}

func (a *API) devices(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	items, err := a.Store.ListSessions(bearer(r))
	if err != nil {
		writeError(w, 401, "session_invalid", "Session is invalid")
		return
	}
	writeJSON(w, 200, map[string]any{"devices": items})
}

func (a *API) deviceSession(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodDelete) {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/account/devices/")
	id = strings.TrimSuffix(id, "/sessions")
	id = strings.Trim(id, "/")
	if id == "" {
		writeError(w, 400, "device_id_required", "Device ID required")
		return
	}
	if err := a.Store.RevokeSession(bearer(r), id); err != nil {
		writeError(w, 404, "session_not_found", "Session not found")
		return
	}
	writeJSON(w, 200, map[string]any{"revoked": true, "session_id": id})
}

func mobileAuthError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	if err.Error() == "session_invalid" {
		writeError(w, http.StatusUnauthorized, "session_invalid", "Session is invalid")
	} else {
		writeError(w, http.StatusNotFound, "conversation_not_found", "Conversation not found")
	}
	return true
}

func (a *API) mobileHome(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	value, err := a.Store.Home(bearer(r))
	if mobileAuthError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (a *API) mobileConversations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		items, next, err := a.Store.ListConversations(bearer(r), r.URL.Query().Get("cursor"), limit, r.URL.Query().Get("include_archived") == "true")
		if mobileAuthError(w, err) {
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items, "next_cursor": next})
	case http.MethodPost:
		var in struct {
			Title string `json:"title"`
		}
		if !decode(r, &in) {
			writeError(w, 400, "invalid_json", "Invalid JSON")
			return
		}
		var item Conversation
		var err error
		if r.URL.Query().Get("temporary") == "true" {
			item, err = a.Store.CreateTemporaryConversation(bearer(r), in.Title)
		} else {
			item, err = a.Store.CreateConversation(bearer(r), in.Title)
		}
		if err != nil {
			if err.Error() == "session_invalid" {
				writeError(w, 401, "session_invalid", "Session is invalid")
			} else {
				writeError(w, 422, err.Error(), "Conversation title is invalid")
			}
			return
		}
		writeJSON(w, http.StatusCreated, item)
	default:
		writeError(w, 405, "method_not_allowed", "GET or POST required")
	}
}

func (a *API) mobileConversationSearch(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := a.Store.SearchConversations(bearer(r), r.URL.Query().Get("q"), limit)
	if err != nil {
		if err.Error() == "session_invalid" {
			writeError(w, 401, "session_invalid", "Session is invalid")
		} else {
			writeError(w, 400, err.Error(), "Search query is required")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *API) mobileConversationByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/mobile/v1/conversations/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, 400, "conversation_id_required", "Conversation ID required")
		return
	}
	id := parts[0]
	if len(parts) > 1 && parts[1] == "runs" {
		if !requireMethod(w, r, http.MethodPost) {
			return
		}
		if a.ChatRuntimeMode != "upstream" || a.ChatResponder == nil {
			writeError(w, http.StatusServiceUnavailable, "chat_runtime_unavailable", "AI provider runtime is not configured")
			return
		}
		var in struct {
			Body           string `json:"body"`
			Model          string `json:"model"`
			IdempotencyKey string `json:"idempotency_key"`
		}
		if !decode(r, &in) {
			writeError(w, 400, "invalid_json", "Invalid JSON")
			return
		}
		access := bearer(r)
		key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
		if key == "" {
			key = strings.TrimSpace(in.IdempotencyKey)
		}
		if key == "" {
			key, _ = randomToken(16)
		}
		run, created, err := a.Store.StartRunIdempotentResult(access, id, in.Body, in.Model, key)
		if err != nil {
			a.writeConversationRunError(w, err)
			return
		}
		writeJSON(w, http.StatusAccepted, run)
		if created {
			a.dispatchRun(run, access, true)
		}
		return
	}
	if len(parts) > 1 && parts[1] == "exports" {
		if !requireMethod(w, r, http.MethodPost) {
			return
		}
		job, err := a.Store.ExportConversation(bearer(r), id, "")
		if err != nil {
			writeError(w, 404, "conversation_not_found", "Conversation not found")
			return
		}
		writeJSON(w, http.StatusCreated, job)
		return
	}
	if len(parts) > 1 && parts[1] == "draft" {
		if r.Method == http.MethodGet {
			draft, err := a.Store.GetDraft(bearer(r), id)
			if err != nil {
				writeError(w, http.StatusNotFound, "draft_not_found", "Draft not found")
				return
			}
			writeJSON(w, http.StatusOK, draft)
			return
		}
		if r.Method == http.MethodPut {
			var in struct {
				Body string `json:"body"`
			}
			if !decode(r, &in) {
				writeError(w, 400, "invalid_json", "Invalid JSON")
				return
			}
			draft, err := a.Store.SaveDraft(bearer(r), id, in.Body)
			if err != nil {
				writeError(w, 422, err.Error(), "Draft unavailable")
				return
			}
			writeJSON(w, http.StatusOK, draft)
			return
		}
		writeError(w, 405, "method_not_allowed", "GET or PUT required")
		return
	}
	if len(parts) > 1 && parts[1] == "archive" {
		if !requireMethod(w, r, http.MethodPost) {
			return
		}
		item, err := a.Store.ArchiveConversation(bearer(r), id)
		if mobileAuthError(w, err) {
			return
		}
		writeJSON(w, http.StatusOK, item)
		return
	}
	switch r.Method {
	case http.MethodPatch:
		var in struct {
			Title string `json:"title"`
		}
		if !decode(r, &in) {
			writeError(w, 400, "invalid_json", "Invalid JSON")
			return
		}
		item, err := a.Store.UpdateConversation(bearer(r), id, in.Title)
		if err != nil {
			if err.Error() == "session_invalid" {
				writeError(w, 401, "session_invalid", "Session is invalid")
			} else {
				writeError(w, 404, "conversation_not_found", "Conversation not found")
			}
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodDelete:
		item, err := a.Store.DeleteConversation(bearer(r), id)
		if mobileAuthError(w, err) {
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"conversation": item, "recycle_after": item.DeletedAt})
	default:
		writeError(w, 405, "method_not_allowed", "PATCH or DELETE required")
	}
}

func (a *API) writeConversationRunError(w http.ResponseWriter, err error) {
	switch err.Error() {
	case "session_invalid":
		writeError(w, http.StatusUnauthorized, "session_invalid", "Session is invalid")
	case "conversation_busy", "idempotency_conflict":
		writeError(w, http.StatusConflict, err.Error(), "The conversation changed; retry the pending message")
	case "conversation_not_found":
		writeError(w, http.StatusNotFound, "conversation_not_found", "Conversation not found")
	default:
		writeError(w, http.StatusUnprocessableEntity, err.Error(), "Unable to create run")
	}
}

func (a *API) dispatchRun(run MessageRun, accessToken string, optimizeTitle bool) {
	go func() {
		started := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), chatRunTimeout)
		a.runMu.Lock()
		a.runCancels[run.ID] = cancel
		a.runMu.Unlock()
		defer func() {
			cancel()
			a.runMu.Lock()
			delete(a.runCancels, run.ID)
			a.runMu.Unlock()
		}()

		continuationMode := "local_rebuild"
		var response ChatResponse
		var providerErr error
		if state, ok, stateErr := a.Store.ProviderState(accessToken, run.ConversationID, run.BranchID, "upstream", run.Model); stateErr == nil && ok {
			if continuationResponder, supported := a.ChatResponder.(ChatContinuationResponder); supported {
				continuationMode = "provider_continuation"
				build, buildErr := a.Store.CompileRunContext(accessToken, run.ID, continuationMode)
				if buildErr == nil {
					response, providerErr = continuationResponder.Continue(ctx, ChatRequest{Model: run.Model, Messages: build.Messages, ContinuationID: state.ContinuationID})
				} else {
					providerErr = buildErr
				}
				if providerErr != nil {
					_ = a.Store.MarkProviderContinuationFallback(accessToken, run.ID, providerErr.Error())
					continuationMode = "local_rebuild_after_continuation_failure"
				}
			}
		}
		if continuationMode != "provider_continuation" || providerErr != nil {
			build, buildErr := a.Store.CompileRunContext(accessToken, run.ID, continuationMode)
			if buildErr != nil {
				providerErr = buildErr
			} else {
				response, providerErr = a.ChatResponder.Respond(ctx, ChatRequest{Model: run.Model, Messages: build.Messages})
			}
		}
		if providerErr != nil {
			_, _ = a.Store.FailRun(accessToken, run.ID, providerErr.Error())
		} else {
			if response.ContinuationID != "" {
				expiresAt := time.Now().UTC().Add(24 * time.Hour)
				_ = a.Store.SaveProviderState(accessToken, ProviderConversationState{ConversationID: run.ConversationID, BranchID: run.BranchID, Provider: "upstream", Model: run.Model, ContinuationID: response.ContinuationID, Status: "active", ExpiresAt: &expiresAt})
			}
			_, providerErr = a.Store.CompleteRun(accessToken, run.ID, response.Content)
			if providerErr == nil && optimizeTitle {
				a.optimizeConversationTitle(accessToken, run)
			}
		}
		a.Store.RecordMetric("chat.run", time.Since(started).Seconds(), func() string {
			if providerErr != nil {
				return "chat_provider_error"
			}
			return ""
		}())
	}()
}

func (a *API) optimizeConversationTitle(accessToken string, run MessageRun) {
	needed, err := a.Store.ConversationNeedsFinalTitle(accessToken, run.ConversationID)
	if err != nil || !needed {
		return
	}
	_, messages, err := a.Store.Run(accessToken, run.ID)
	if err != nil || len(messages) == 0 {
		return
	}
	var userBody string
	for _, message := range messages {
		if message.Role == "user" {
			userBody = message.Body
			break
		}
	}
	if IsLowInformationMessage(userBody) {
		return
	}
	prompt := "Create a concise Chinese conversation title of 6 to 18 characters. Return only the title.\n"
	for _, message := range messages {
		prompt += message.Role + ": " + message.Body + "\n"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	response, err := a.ChatResponder.Respond(ctx, ChatRequest{Model: run.Model, Messages: []ProviderMessage{{Role: "system", Content: "Return a title only; do not expose hidden reasoning."}, {Role: "user", Content: prompt}}})
	if err != nil {
		return
	}
	_, _ = a.Store.ApplyAutomaticConversationTitle(accessToken, run.ConversationID, NormalizeFinalConversationTitle(response.Content, userBody), "AUTO_FINAL")
}

func (a *API) internalConversationByID(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/internal/v1/conversations/"), "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" {
		writeError(w, 404, "not_found", "Endpoint not found")
		return
	}
	switch parts[1] {
	case "title":
		a.internalConversationTitle(w, r, parts[0])
	case "context":
		a.internalConversationContext(w, r, parts[0], false)
	case "context-items":
		a.internalConversationContext(w, r, parts[0], true)
	case "compact":
		a.internalConversationCompact(w, r, parts[0])
	default:
		writeError(w, 404, "not_found", "Endpoint not found")
	}
}

func (a *API) internalConversationTitle(w http.ResponseWriter, r *http.Request, id string) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var in struct {
		Title string `json:"title"`
	}
	if !decode(r, &in) {
		writeError(w, 400, "invalid_json", "Invalid JSON")
		return
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		if a.ChatRuntimeMode != "upstream" || a.ChatResponder == nil {
			writeError(w, http.StatusServiceUnavailable, "chat_runtime_unavailable", "AI provider runtime is not configured")
			return
		}
		generated, generateErr := a.ChatResponder.Respond(r.Context(), ChatRequest{Model: "ylven-default", Messages: []ProviderMessage{{Role: "system", Content: "Return only a concise Chinese title of 6 to 18 characters."}, {Role: "user", Content: "为会话生成标题：" + id}}})
		if generateErr != nil {
			writeError(w, http.StatusBadGateway, "chat_provider_unavailable", "Unable to generate title")
			return
		}
		title = NormalizeFinalConversationTitle(generated.Content, id)
	}
	item, err := a.Store.ApplyAutomaticConversationTitle(bearer(r), id, title, "AUTO_FINAL")
	if err != nil {
		if err.Error() == "session_invalid" {
			writeError(w, 401, "session_invalid", "Session is invalid")
		} else {
			writeError(w, 404, "conversation_not_found", "Conversation not found")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"conversation": item, "generated": true})
}

func (a *API) internalConversationContext(w http.ResponseWriter, r *http.Request, conversationID string, itemsOnly bool) {
	if r.Method == http.MethodGet {
		snapshot, err := a.Store.ConversationContext(bearer(r), conversationID)
		if err != nil {
			if err.Error() == "session_invalid" {
				writeError(w, http.StatusUnauthorized, "session_invalid", "Session is invalid")
			} else {
				writeError(w, http.StatusNotFound, "conversation_not_found", "Conversation not found")
			}
			return
		}
		if itemsOnly {
			writeJSON(w, http.StatusOK, map[string]any{
				"conversation_id": snapshot.Conversation.ID,
				"branch_id":       snapshot.Branch.ID,
				"status": func() string {
					if len(snapshot.Items) == 0 {
						return "empty"
					}
					return "success"
				}(),
				"items":         snapshot.Items,
				"message_parts": snapshot.MessageParts,
				"summary":       snapshot.Summary,
			})
			return
		}
		writeJSON(w, http.StatusOK, snapshot)
		return
	}
	if itemsOnly || !requireMethod(w, r, http.MethodPost) {
		if itemsOnly && r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", http.MethodGet+" required")
		}
		return
	}
	var in struct {
		Role           string `json:"role"`
		Body           string `json:"body"`
		IdempotencyKey string `json:"idempotency_key"`
	}
	if !decode(r, &in) {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON")
		return
	}
	if strings.TrimSpace(in.IdempotencyKey) == "" {
		in.IdempotencyKey = strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	}
	message, created, err := a.Store.AppendCanonicalMessage(bearer(r), conversationID, in.Role, in.Body, in.IdempotencyKey)
	if err != nil {
		switch err.Error() {
		case "session_invalid":
			writeError(w, http.StatusUnauthorized, "session_invalid", "Session is invalid")
		case "conversation_not_found":
			writeError(w, http.StatusNotFound, "conversation_not_found", "Conversation not found")
		case "idempotency_conflict":
			writeError(w, http.StatusConflict, "idempotency_conflict", "Idempotency key was already used for another request")
		default:
			writeError(w, http.StatusBadRequest, err.Error(), "Invalid conversation context message")
		}
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	writeJSON(w, status, map[string]any{"message": message, "created": created})
}

func (a *API) internalConversationCompact(w http.ResponseWriter, r *http.Request, conversationID string) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var in struct {
		IdempotencyKey string `json:"idempotency_key"`
	}
	if !decode(r, &in) {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON")
		return
	}
	if strings.TrimSpace(in.IdempotencyKey) == "" {
		in.IdempotencyKey = strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	}
	access := bearer(r)
	item, created, err := a.Store.QueueConversationCompaction(access, conversationID, in.IdempotencyKey)
	if err != nil {
		switch err.Error() {
		case "session_invalid":
			writeError(w, http.StatusUnauthorized, "session_invalid", "Session is invalid")
		case "conversation_not_found":
			writeError(w, http.StatusNotFound, "conversation_not_found", "Conversation not found")
		case "conversation_context_empty":
			writeError(w, http.StatusUnprocessableEntity, "conversation_context_empty", "Conversation has no context to compact")
		case "idempotency_conflict":
			writeError(w, http.StatusConflict, "idempotency_conflict", "Idempotency key was already used for another request")
		default:
			writeError(w, http.StatusBadRequest, err.Error(), "Unable to queue conversation compaction")
		}
		return
	}
	if created {
		go func() {
			_, _ = a.Store.ProcessConversationCompaction(access, item.ID)
		}()
	}
	writeJSON(w, http.StatusOK, map[string]any{"compaction": item, "created": created})
}

func (a *API) internalContextBuild(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var in struct {
		RunID string `json:"run_id"`
	}
	if !decode(r, &in) || strings.TrimSpace(in.RunID) == "" {
		writeError(w, http.StatusBadRequest, "run_id_required", "Run ID required")
		return
	}
	mode := "local_rebuild"
	if strings.HasSuffix(r.URL.Path, "/recompile") {
		mode = "model_switch_recompile"
	}
	build, err := a.Store.CompileRunContext(bearer(r), in.RunID, mode)
	if err != nil {
		if err.Error() == "session_invalid" {
			writeError(w, http.StatusUnauthorized, "session_invalid", "Session is invalid")
		} else {
			writeError(w, http.StatusNotFound, err.Error(), "Run context not found")
		}
		return
	}
	writeJSON(w, http.StatusOK, build)
}

func (a *API) internalProviderFallback(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var in struct {
		RunID string `json:"run_id"`
	}
	if !decode(r, &in) || strings.TrimSpace(in.RunID) == "" {
		writeError(w, http.StatusBadRequest, "run_id_required", "Run ID required")
		return
	}
	if err := a.Store.MarkProviderContinuationFallback(bearer(r), in.RunID, "manual_fallback"); err != nil {
		writeError(w, http.StatusNotFound, err.Error(), "Run not found")
		return
	}
	build, err := a.Store.CompileRunContext(bearer(r), in.RunID, "local_rebuild_after_continuation_failure")
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error(), "Unable to rebuild context")
		return
	}
	writeJSON(w, http.StatusOK, build)
}

func (a *API) mobileRunByID(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/mobile/v1/runs/"), "/")
	parts := strings.Split(path, "/")
	if len(parts) < 1 || parts[0] == "" {
		writeError(w, 400, "run_id_required", "Run ID required")
		return
	}
	runID := parts[0]
	if len(parts) > 1 && parts[1] == "cancel" {
		if !requireMethod(w, r, http.MethodPost) {
			return
		}
		run, err := a.Store.CancelRun(bearer(r), runID)
		if err != nil {
			writeError(w, 404, "run_not_found", "Run not found")
			return
		}
		a.runMu.Lock()
		if cancel := a.runCancels[runID]; cancel != nil {
			cancel()
		}
		a.runMu.Unlock()
		writeJSON(w, http.StatusOK, run)
		return
	}
	if len(parts) > 1 && parts[1] == "events" {
		if !requireMethod(w, r, http.MethodGet) {
			return
		}
		after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
		run, events, err := a.Store.Events(bearer(r), runID, after)
		if err != nil {
			writeError(w, 404, "run_not_found", "Run not found")
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		for _, event := range events {
			payload, _ := json.Marshal(event)
			fmt.Fprintf(w, "id: %d\ndata: %s\n\n", event.ID, payload)
		}
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		_ = run
		return
	}
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	run, messages, err := a.Store.Run(bearer(r), runID)
	if err != nil {
		writeError(w, 404, "run_not_found", "Run not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"run": run, "messages": messages})
}

func (a *API) mobileMessageByID(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/mobile/v1/messages/"), "/")
	parts := strings.Split(path, "/")
	if len(parts) < 1 || parts[0] == "" {
		writeError(w, 400, "message_id_required", "Message ID required")
		return
	}
	m, err := a.Store.MessageOwned(bearer(r), parts[0])
	if err != nil {
		writeError(w, 404, "message_not_found", "Message not found")
		return
	}
	if len(parts) < 2 {
		writeJSON(w, http.StatusOK, m)
		return
	}
	switch parts[1] {
	case "citations":
		if !requireMethod(w, r, http.MethodGet) {
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"message_id": m.ID, "citations": extractCitations(m.Body)})
	case "run-metadata":
		if !requireMethod(w, r, http.MethodGet) {
			return
		}
		run, _, err := a.Store.RunForMessage(bearer(r), m.ID)
		if err != nil {
			writeError(w, http.StatusNotFound, "run_not_found", "Run metadata not found")
			return
		}
		writeJSON(w, http.StatusOK, run)
	case "exports":
		if !requireMethod(w, r, http.MethodPost) {
			return
		}
		job, err := a.Store.ExportConversation(bearer(r), m.ConversationID, m.ID)
		if err != nil {
			writeError(w, 422, err.Error(), "Export unavailable")
			return
		}
		writeJSON(w, http.StatusCreated, job)
	case "speech":
		if !requireMethod(w, r, http.MethodPost) {
			return
		}
		job, err := a.Store.CreateSpeechJob(bearer(r), m.ID)
		if err != nil {
			writeError(w, http.StatusNotFound, "message_not_found", "Assistant message not found")
			return
		}
		writeJSON(w, http.StatusAccepted, job)
	case "feedback":
		if !requireMethod(w, r, http.MethodPost) {
			return
		}
		var in struct {
			Value string `json:"value"`
		}
		if !decode(r, &in) || (in.Value != "up" && in.Value != "down") {
			writeError(w, 400, "feedback_invalid", "Feedback value must be up or down")
			return
		}
		if err := a.Store.SaveMessageFeedback(bearer(r), m.ID, in.Value); err != nil {
			writeError(w, http.StatusUnprocessableEntity, err.Error(), "Feedback unavailable")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"message_id": m.ID, "value": in.Value, "saved": true})
	case "regenerate":
		if !requireMethod(w, r, http.MethodPost) {
			return
		}
		if a.ChatRuntimeMode != "upstream" || a.ChatResponder == nil {
			writeError(w, http.StatusServiceUnavailable, "chat_runtime_unavailable", "AI provider runtime is not configured")
			return
		}
		access := bearer(r)
		key, _ := randomToken(16)
		run, _, err := a.Store.StartRunIdempotentResult(access, m.ConversationID, "重新生成："+m.Body, "ylven-default", key)
		if err != nil {
			writeError(w, 422, err.Error(), "Regeneration unavailable")
			return
		}
		writeJSON(w, http.StatusAccepted, run)
		a.dispatchRun(run, access, false)
	default:
		writeError(w, 404, "not_found", "Endpoint not found")
	}
}

func (a *API) requireAdmin(w http.ResponseWriter, r *http.Request, permission string) (AdminUser, bool) {
	admin, ok := a.Store.AuthorizeAdmin(bearer(r), permission)
	if !ok {
		writeError(w, 403, "permission_denied", "Administrator permission required")
		return AdminUser{}, false
	}
	return admin, true
}

func (a *API) requireStepUp(w http.ResponseWriter, r *http.Request) bool {
	if err := a.Store.ConsumeStepUp(bearer(r), r.Header.Get("X-Step-Up-Token")); err != nil {
		writeError(w, 403, "step_up_required", "A fresh high-risk confirmation is required")
		return false
	}
	return true
}

func (a *API) adminLogin(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if a.BootstrapError != nil {
		writeError(w, 503, "admin_bootstrap_invalid", "Administrator bootstrap configuration is invalid")
		return
	}
	var in struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decode(r, &in) {
		writeError(w, 400, "invalid_json", "Invalid JSON")
		return
	}
	login := strings.TrimSpace(in.Username)
	if login == "" {
		login = strings.TrimSpace(in.Email)
	}
	email := login
	if strings.EqualFold(login, a.AdminUsername) && a.AdminEmail != "" {
		email = a.AdminEmail
	}
	allowed, _ := a.Store.AllowAttempt("admin:"+strings.ToLower(login), 5, 10*time.Minute)
	if !allowed {
		writeError(w, 429, "auth_rate_limited", "Try again later")
		return
	}
	session, token, err := a.Store.AdminLogin(email, in.Password)
	if err != nil {
		writeError(w, 401, "admin_credentials_invalid", "Unable to authenticate")
		return
	}
	writeJSON(w, 201, map[string]any{"session_id": session.ID, "access_token": token, "expires_at": session.ExpiresAt, "username": a.AdminUsername})
}

func (a *API) adminSessions(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	if _, ok := a.requireAdmin(w, r, "admins:read"); !ok {
		return
	}
	writeJSON(w, 200, map[string]any{"admins": a.Store.ListAdmins(), "sessions": a.Store.ListAdminSessions()})
}

func (a *API) adminLogout(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if err := a.Store.AdminLogout(bearer(r)); err != nil {
		writeError(w, 401, "admin_session_invalid", "Administrator session is invalid")
		return
	}
	writeJSON(w, 200, map[string]any{"logged_out": true})
}

func (a *API) adminStepUp(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if _, ok := a.requireAdmin(w, r, "security:step-up"); !ok {
		return
	}
	var in struct {
		Password string `json:"password"`
	}
	if !decode(r, &in) {
		writeError(w, 400, "invalid_json", "Invalid JSON")
		return
	}
	token, expires, err := a.Store.CreateStepUp(bearer(r), in.Password)
	if err != nil {
		writeError(w, 401, "step_up_failed", "Unable to confirm administrator")
		return
	}
	writeJSON(w, 201, map[string]any{"step_up_token": token, "expires_at": expires})
}

func (a *API) adminSetting(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		permission := "settings:read"
		if r.Method == http.MethodPut {
			permission = "settings:write"
		}
		admin, ok := a.requireAdmin(w, r, permission)
		if !ok {
			return
		}
		if r.Method == http.MethodGet {
			value, found := a.Store.GetSetting(name)
			if !found {
				writeError(w, 404, "setting_not_found", "Setting not found")
				return
			}
			writeJSON(w, 200, map[string]any{"name": name, "value": value})
			return
		}
		if r.Method == http.MethodPut {
			if !a.requireStepUp(w, r) {
				return
			}
			value := map[string]string{}
			if !decode(r, &value) {
				writeError(w, 400, "invalid_json", "Invalid JSON")
				return
			}
			if err := a.Store.PutSetting(name, value); err != nil {
				writeError(w, 422, err.Error(), err.Error())
				return
			}
			writeJSON(w, 200, map[string]any{"name": name, "value": value, "audited": true, "actor_id": admin.ID})
			return
		}
		writeError(w, 405, "method_not_allowed", "GET or PUT required")
	}
}

func (a *API) adminUsers(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	if _, ok := a.requireAdmin(w, r, "users:read"); !ok {
		return
	}
	writeJSON(w, 200, map[string]any{"users": a.Store.ListUsers()})
}

func (a *API) adminHomeConfig(w http.ResponseWriter, r *http.Request) {
	admin, ok := a.requireAdmin(w, r, "content:home")
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"config": a.Store.HomeConfigSnapshot(), "audit": a.Store.AuditSnapshot()})
	case http.MethodPut:
		if !a.requireStepUp(w, r) {
			return
		}
		var in HomeConfig
		if !decode(r, &in) {
			writeError(w, 400, "invalid_json", "Invalid JSON")
			return
		}
		saved, err := a.Store.UpdateHomeConfig(in, admin.ID)
		if err != nil {
			writeError(w, 422, err.Error(), "Home configuration is invalid")
			return
		}
		writeJSON(w, http.StatusOK, saved)
	default:
		writeError(w, 405, "method_not_allowed", "GET or PUT required")
	}
}

func (a *API) adminConversations(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	if _, ok := a.requireAdmin(w, r, "conversations:read"); !ok {
		return
	}
	// P03-W01 deliberately exposes an auditable read-only operations view.
	// It must not acknowledge a write that did not change a conversation.
	writeJSON(w, http.StatusOK, map[string]any{
		"conversations": a.Store.ListAllConversations(),
		"audit":         a.Store.AuditSnapshot(),
	})
}

func (a *API) adminConversationDetail(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	if _, ok := a.requireAdmin(w, r, "conversations:read"); !ok {
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/admin/v1/conversations/"), "/")
	conversation, messages, runs, ok := a.Store.ConversationDetail(id)
	if !ok {
		writeError(w, http.StatusNotFound, "conversation_not_found", "Conversation not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"conversation": conversation, "messages": messages, "runs": runs, "audit": a.Store.AuditSnapshot()})
}

func (a *API) chatMetrics(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	if _, ok := a.requireAdmin(w, r, "conversations:read"); !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"metrics": a.Store.MetricsSnapshot()})
}

func (a *API) adminUserDetail(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/admin/v1/users/"), "/")
	if strings.HasSuffix(path, "/status") {
		a.adminUserStatus(w, r, strings.TrimSuffix(path, "/status"))
		return
	}
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	if _, ok := a.requireAdmin(w, r, "users:read"); !ok {
		return
	}
	user, sessions, ok := a.Store.UserDetail(path)
	if !ok {
		writeError(w, 404, "user_not_found", "User not found")
		return
	}
	writeJSON(w, 200, map[string]any{"user": user, "sessions": sessions})
}

func (a *API) adminUserStatus(w http.ResponseWriter, r *http.Request, id string) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	admin, ok := a.requireAdmin(w, r, "users:status")
	if !ok {
		return
	}
	if !a.requireStepUp(w, r) {
		return
	}
	var in struct {
		Status string `json:"status"`
	}
	if !decode(r, &in) {
		writeError(w, 400, "invalid_json", "Invalid JSON")
		return
	}
	user, err := a.Store.SetUserStatus(id, in.Status, admin.ID)
	if err != nil {
		status := 422
		if err.Error() == "user_not_found" {
			status = 404
		}
		writeError(w, status, err.Error(), err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"user": userView(user), "audited": true})
}

func (a *API) adminRoles(w http.ResponseWriter, r *http.Request) {
	permission := "rbac:read"
	if r.Method == http.MethodPost {
		permission = "rbac:write"
	}
	admin, ok := a.requireAdmin(w, r, permission)
	if !ok {
		return
	}
	if r.Method == http.MethodGet {
		writeJSON(w, 200, map[string]any{"roles": a.Store.ListRoles()})
		return
	}
	if r.Method == http.MethodPost {
		if !a.requireStepUp(w, r) {
			return
		}
		var role Role
		if !decode(r, &role) {
			writeError(w, 400, "invalid_json", "Invalid JSON")
			return
		}
		saved, err := a.Store.PutRole(role, admin.ID)
		if err != nil {
			writeError(w, 422, err.Error(), err.Error())
			return
		}
		writeJSON(w, 201, map[string]any{"role": saved, "audited": true})
		return
	}
	writeError(w, 405, "method_not_allowed", "GET or POST required")
}

func (a *API) adminEmailTemplates(w http.ResponseWriter, r *http.Request) {
	permission := "notifications:read"
	if r.Method == http.MethodPut {
		permission = "notifications:write"
	}
	admin, ok := a.requireAdmin(w, r, permission)
	if !ok {
		return
	}
	if r.Method == http.MethodGet {
		templates, deliveries := a.Store.ListEmailTemplates()
		writeJSON(w, 200, map[string]any{"templates": templates, "deliveries": deliveries})
		return
	}
	if r.Method == http.MethodPut {
		if !a.requireStepUp(w, r) {
			return
		}
		var template EmailTemplate
		if !decode(r, &template) {
			writeError(w, 400, "invalid_json", "Invalid JSON")
			return
		}
		saved, err := a.Store.PutEmailTemplate(template, admin.ID)
		if err != nil {
			status := 422
			if err.Error() == "template_version_conflict" {
				status = 409
			}
			writeError(w, status, err.Error(), err.Error())
			return
		}
		writeJSON(w, 200, map[string]any{"template": saved, "audited": true})
		return
	}
	writeError(w, 405, "method_not_allowed", "GET or PUT required")
}
