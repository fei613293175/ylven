package identity

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type API struct { Store *Store; TurnstileMode string; DebugOTP bool }

func NewAPI(store *Store) *API { mode := os.Getenv("TURNSTILE_MODE"); if mode == "" { mode = "external" }; return &API{Store:store, TurnstileMode:mode, DebugOTP:os.Getenv("IDENTITY_DEBUG_OTP")=="1"} }

func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/email/normalize", a.normalize)
	mux.HandleFunc("/api/v1/auth/register/challenge", a.challenge)
	mux.HandleFunc("/internal/v1/security/turnstile/verify", a.turnstile)
	mux.HandleFunc("/api/v1/auth/register/otp/send", a.otpSend)
	mux.HandleFunc("/api/v1/auth/register/otp/verify", a.otpVerify)
	mux.HandleFunc("/api/v1/auth/register/complete", a.complete)
	mux.HandleFunc("/api/v1/auth/login/challenge", a.loginChallenge)
	mux.HandleFunc("/api/v1/auth/login/otp/send", a.loginOTPSend)
	mux.HandleFunc("/api/v1/auth/login/otp/verify", a.loginOTPVerify)
	mux.HandleFunc("/api/v1/auth/sessions", a.sessions)
	mux.HandleFunc("/api/v1/auth/sessions/refresh", a.refreshSession)
	mux.HandleFunc("/api/v1/auth/logout", a.logout)
	mux.HandleFunc("/api/v1/auth/logout-all", a.logoutAll)
	mux.HandleFunc("/api/v1/account/devices", a.devices)
	mux.HandleFunc("/api/v1/account/devices/", a.deviceSession)
	mux.HandleFunc("/admin/v1/settings/email", a.adminSetting("email"))
	mux.HandleFunc("/admin/v1/settings/turnstile", a.adminSetting("turnstile"))
	mux.HandleFunc("/admin/v1/settings/otp-policy", a.adminSetting("otp-policy"))
	mux.HandleFunc("/admin/v1/users", a.adminUsers)
	mux.HandleFunc("/admin/v1/users/", a.adminUserDetail)
	return requestGuard(mux)
}

func requestGuard(next http.Handler) http.Handler { return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Header().Set("Content-Type","application/json"); next.ServeHTTP(w,r) }) }
func decode(r *http.Request, dst any) bool { return json.NewDecoder(io.LimitReader(r.Body,1<<20)).Decode(dst)==nil }
func writeJSON(w http.ResponseWriter, status int, value any) { w.WriteHeader(status); _ = json.NewEncoder(w).Encode(value) }
func writeError(w http.ResponseWriter, status int, code, message string) { writeJSON(w,status,map[string]any{"error":map[string]string{"code":code,"message":message}}) }

func (a *API) normalize(w http.ResponseWriter, r *http.Request) { var in struct{Email string `json:"email"`}; if !decode(r,&in){writeError(w,400,"invalid_json","Invalid JSON");return}; email,err:=NormalizeEmail(in.Email); if err!=nil{writeError(w,422,"invalid_email",err.Error());return}; writeJSON(w,200,map[string]any{"email":email,"valid":true}) }
func (a *API) challenge(w http.ResponseWriter, r *http.Request) { var in struct{Email string `json:"email"`; Purpose string `json:"purpose"`; TurnstileToken string `json:"turnstile_token"`}; if !decode(r,&in){writeError(w,400,"invalid_json","Invalid JSON");return}; if in.Purpose==""{in.Purpose="register"}; c,err:=a.Store.CreateChallenge(in.Email,in.Purpose); if err!=nil{writeError(w,422,"invalid_email",err.Error());return}; if in.TurnstileToken!="" { valid:=a.TurnstileMode=="mock"&&in.TurnstileToken=="test-pass"; if err:=a.Store.VerifyTurnstile(c.ID,in.TurnstileToken,valid); err!=nil {writeError(w,422,"turnstile_failed",err.Error());return}; c.TurnstileVerified=true }; writeJSON(w,201,map[string]any{"challenge_id":c.ID,"email":c.Email,"expires_at":c.ExpiresAt.UTC()}) }
func (a *API) turnstile(w http.ResponseWriter, r *http.Request) { var in struct{ChallengeID string `json:"challenge_id"`; Token string `json:"token"`}; if !decode(r,&in){writeError(w,400,"invalid_json","Invalid JSON");return}; valid:=a.TurnstileMode=="mock"&&in.Token=="test-pass"; if a.TurnstileMode!="mock" {writeError(w,503,"turnstile_unconfigured","Turnstile provider is not configured");return}; if err:=a.Store.VerifyTurnstile(in.ChallengeID,in.Token,valid);err!=nil{writeError(w,422,"turnstile_failed",err.Error());return}; writeJSON(w,200,map[string]any{"verified":true}) }
func (a *API) otpSend(w http.ResponseWriter, r *http.Request) { var in struct{ChallengeID string `json:"challenge_id"`}; if !decode(r,&in){writeError(w,400,"invalid_json","Invalid JSON");return}; codeBytes:=make([]byte,4); if _,err:=rand.Read(codeBytes);err!=nil{writeError(w,500,"randomness_failed","Unable to create OTP");return}; code:=fmt.Sprintf("%06d",(int(codeBytes[0])<<16|int(codeBytes[1])<<8|int(codeBytes[2]))%1000000); otp,err:=a.Store.CreateOTP(in.ChallengeID,code);if err!=nil{writeError(w,422,"challenge_invalid",err.Error());return}; out:=map[string]any{"delivery_id":otp.ID,"expires_at":otp.ExpiresAt.UTC()};if a.DebugOTP{out["debug_code"]=code};writeJSON(w,202,out) }
func (a *API) otpVerify(w http.ResponseWriter, r *http.Request) { var in struct{ChallengeID string `json:"challenge_id"`; Code string `json:"code"`};if !decode(r,&in){writeError(w,400,"invalid_json","Invalid JSON");return};if err:=a.Store.VerifyOTP(in.ChallengeID,in.Code);err!=nil{writeError(w,422,"otp_invalid",err.Error());return};writeJSON(w,200,map[string]any{"verified":true}) }
func (a *API) complete(w http.ResponseWriter, r *http.Request) { var in struct{ChallengeID string `json:"challenge_id"`; Email string `json:"email"`; Password string `json:"password"`};if !decode(r,&in){writeError(w,400,"invalid_json","Invalid JSON");return};u,err:=a.Store.CreateUser(in.ChallengeID,in.Email,in.Password);if err!=nil{status:=422;if strings.Contains(err.Error(),"already_registered"){status=409};writeError(w,status,err.Error(),err.Error());return};writeJSON(w,201,map[string]any{"user_id":u.ID,"email":u.Email}) }
func (a *API) loginChallenge(w http.ResponseWriter, r *http.Request) { var in struct{Email string `json:"email"`; TurnstileToken string `json:"turnstile_token"`}; if !decode(r,&in){writeError(w,400,"invalid_json","Invalid JSON");return}; normalized,err:=NormalizeEmail(in.Email); if err!=nil { writeJSON(w,202,map[string]any{"accepted":true}); return }; allowed,_:=a.Store.AllowAttempt(normalized,5,10*time.Minute); if !allowed {writeError(w,429,"auth_rate_limited","Try again later");return}; c,err:=a.Store.CreateChallenge(normalized,"login"); if err!=nil { writeJSON(w,202,map[string]any{"accepted":true}); return }; if in.TurnstileToken!="" { valid:=a.TurnstileMode=="mock"&&in.TurnstileToken=="test-pass"; if err:=a.Store.VerifyTurnstile(c.ID,in.TurnstileToken,valid);err!=nil{writeError(w,422,"turnstile_failed",err.Error());return} }; writeJSON(w,201,map[string]any{"challenge_id":c.ID,"accepted":true,"expires_at":c.ExpiresAt.UTC()}) }
func (a *API) loginOTPSend(w http.ResponseWriter, r *http.Request) { a.otpSend(w,r) }
func (a *API) loginOTPVerify(w http.ResponseWriter, r *http.Request) { a.otpVerify(w,r) }
func (a *API) sessions(w http.ResponseWriter, r *http.Request) { var in struct{ChallengeID string `json:"challenge_id"`; Email string `json:"email"`}; if !decode(r,&in){writeError(w,400,"invalid_json","Invalid JSON");return}; session,access,refresh,err:=a.Store.CreateSession(in.ChallengeID,in.Email);if err!=nil{writeError(w,401,"login_not_verified","Unable to authenticate");return};writeJSON(w,201,map[string]any{"session_id":session.ID,"access_token":access,"refresh_token":refresh,"access_expires_at":session.AccessExpiresAt,"refresh_expires_at":session.RefreshExpiresAt}) }
func (a *API) refreshSession(w http.ResponseWriter, r *http.Request) { var in struct{RefreshToken string `json:"refresh_token"`};if !decode(r,&in){writeError(w,400,"invalid_json","Invalid JSON");return};session,access,refresh,err:=a.Store.RotateSession(in.RefreshToken);if err!=nil{writeError(w,401,"refresh_token_invalid","Refresh token is invalid or expired");return};writeJSON(w,200,map[string]any{"session_id":session.ID,"access_token":access,"refresh_token":refresh,"access_expires_at":session.AccessExpiresAt,"refresh_expires_at":session.RefreshExpiresAt}) }
func bearer(r *http.Request) string { value:=strings.TrimSpace(r.Header.Get("Authorization")); if len(value)>7 && strings.EqualFold(value[:7],"Bearer "){ return strings.TrimSpace(value[7:]) }; return "" }
func (a *API) logout(w http.ResponseWriter, r *http.Request) { if r.Method!=http.MethodPost {writeError(w,405,"method_not_allowed","POST required");return}; if err:=a.Store.Logout(bearer(r));err!=nil{writeError(w,401,"session_invalid","Session is invalid");return};writeJSON(w,200,map[string]any{"logged_out":true}) }
func (a *API) logoutAll(w http.ResponseWriter, r *http.Request) { if r.Method!=http.MethodPost {writeError(w,405,"method_not_allowed","POST required");return};if err:=a.Store.LogoutAll(bearer(r));err!=nil{writeError(w,401,"session_invalid","Session is invalid");return};writeJSON(w,200,map[string]any{"logged_out_all":true}) }
func (a *API) devices(w http.ResponseWriter, r *http.Request) { if r.Method!=http.MethodGet {writeError(w,405,"method_not_allowed","GET required");return};items,err:=a.Store.ListSessions(bearer(r));if err!=nil{writeError(w,401,"session_invalid","Session is invalid");return};writeJSON(w,200,map[string]any{"devices":items}) }
func (a *API) deviceSession(w http.ResponseWriter, r *http.Request) { if r.Method!=http.MethodDelete {writeError(w,405,"method_not_allowed","DELETE required");return};id:=strings.TrimPrefix(r.URL.Path,"/api/v1/account/devices/");if id==""{writeError(w,400,"device_id_required","Device ID required");return};if err:=a.Store.RevokeSession(bearer(r),id);err!=nil{writeError(w,404,"session_not_found","Session not found");return};writeJSON(w,200,map[string]any{"revoked":true,"session_id":id}) }
func adminAllowed(r *http.Request) bool { return r.Header.Get("X-Admin-Role")=="superadmin" }
func (a *API) adminSetting(name string) http.HandlerFunc { return func(w http.ResponseWriter,r *http.Request){if !adminAllowed(r){writeError(w,403,"permission_denied","Administrator role required");return};if r.Method==http.MethodGet{value,ok:=a.Store.GetSetting(name);if !ok{writeError(w,404,"setting_not_found","Setting not found");return};writeJSON(w,200,map[string]any{"name":name,"value":value});return};if r.Method==http.MethodPut{value:=map[string]string{};if !decode(r,&value){writeError(w,400,"invalid_json","Invalid JSON");return};if err:=a.Store.PutSetting(name,value);err!=nil{writeError(w,422,err.Error(),err.Error());return};writeJSON(w,200,map[string]any{"name":name,"value":value,"audited":true});return};writeError(w,405,"method_not_allowed","GET or PUT required")} }
func (a *API) adminUsers(w http.ResponseWriter,r *http.Request){if !adminAllowed(r){writeError(w,403,"permission_denied","Administrator role required");return};if r.Method!=http.MethodGet{writeError(w,405,"method_not_allowed","GET required");return};writeJSON(w,200,map[string]any{"users":a.Store.ListUsers()})}
func (a *API) adminUserDetail(w http.ResponseWriter,r *http.Request){if !adminAllowed(r){writeError(w,403,"permission_denied","Administrator role required");return};if r.Method!=http.MethodGet{writeError(w,405,"method_not_allowed","GET required");return};id:=strings.TrimPrefix(r.URL.Path,"/admin/v1/users/");user,sessions,ok:=a.Store.UserDetail(id);if !ok{writeError(w,404,"user_not_found","User not found");return};writeJSON(w,200,map[string]any{"user":user,"sessions":sessions})}
