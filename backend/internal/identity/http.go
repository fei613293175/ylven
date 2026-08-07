package identity

import (
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
}

func NewAPI(store *Store) *API {
	api := &API{Store: store}
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
	return requestGuard(mux)
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
	session, access, refresh, sessionErr := a.Store.CreateSessionForUser(user.Email)
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
	}
	if !decode(r, &in) {
		writeError(w, 400, "invalid_json", "Invalid JSON")
		return
	}
	session, access, refresh, err := a.Store.CreateSession(in.ChallengeID, in.Email)
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
