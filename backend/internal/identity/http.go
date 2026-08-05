package identity

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
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
	return requestGuard(mux)
}

func requestGuard(next http.Handler) http.Handler { return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Header().Set("Content-Type","application/json"); if r.Method != http.MethodPost { writeError(w,http.StatusMethodNotAllowed,"method_not_allowed","POST required"); return }; next.ServeHTTP(w,r) }) }
func decode(r *http.Request, dst any) bool { return json.NewDecoder(io.LimitReader(r.Body,1<<20)).Decode(dst)==nil }
func writeJSON(w http.ResponseWriter, status int, value any) { w.WriteHeader(status); _ = json.NewEncoder(w).Encode(value) }
func writeError(w http.ResponseWriter, status int, code, message string) { writeJSON(w,status,map[string]any{"error":map[string]string{"code":code,"message":message}}) }

func (a *API) normalize(w http.ResponseWriter, r *http.Request) { var in struct{Email string `json:"email"`}; if !decode(r,&in){writeError(w,400,"invalid_json","Invalid JSON");return}; email,err:=NormalizeEmail(in.Email); if err!=nil{writeError(w,422,"invalid_email",err.Error());return}; writeJSON(w,200,map[string]any{"email":email,"valid":true}) }
func (a *API) challenge(w http.ResponseWriter, r *http.Request) { var in struct{Email string `json:"email"`; Purpose string `json:"purpose"`; TurnstileToken string `json:"turnstile_token"`}; if !decode(r,&in){writeError(w,400,"invalid_json","Invalid JSON");return}; if in.Purpose==""{in.Purpose="register"}; c,err:=a.Store.CreateChallenge(in.Email,in.Purpose); if err!=nil{writeError(w,422,"invalid_email",err.Error());return}; if in.TurnstileToken!="" { valid:=a.TurnstileMode=="mock"&&in.TurnstileToken=="test-pass"; if err:=a.Store.VerifyTurnstile(c.ID,in.TurnstileToken,valid); err!=nil {writeError(w,422,"turnstile_failed",err.Error());return}; c.TurnstileVerified=true }; writeJSON(w,201,map[string]any{"challenge_id":c.ID,"email":c.Email,"expires_at":c.ExpiresAt.UTC()}) }
func (a *API) turnstile(w http.ResponseWriter, r *http.Request) { var in struct{ChallengeID string `json:"challenge_id"`; Token string `json:"token"`}; if !decode(r,&in){writeError(w,400,"invalid_json","Invalid JSON");return}; valid:=a.TurnstileMode=="mock"&&in.Token=="test-pass"; if a.TurnstileMode!="mock" {writeError(w,503,"turnstile_unconfigured","Turnstile provider is not configured");return}; if err:=a.Store.VerifyTurnstile(in.ChallengeID,in.Token,valid);err!=nil{writeError(w,422,"turnstile_failed",err.Error());return}; writeJSON(w,200,map[string]any{"verified":true}) }
func (a *API) otpSend(w http.ResponseWriter, r *http.Request) { var in struct{ChallengeID string `json:"challenge_id"`}; if !decode(r,&in){writeError(w,400,"invalid_json","Invalid JSON");return}; codeBytes:=make([]byte,4); if _,err:=rand.Read(codeBytes);err!=nil{writeError(w,500,"randomness_failed","Unable to create OTP");return}; code:=fmt.Sprintf("%06d",(int(codeBytes[0])<<16|int(codeBytes[1])<<8|int(codeBytes[2]))%1000000); otp,err:=a.Store.CreateOTP(in.ChallengeID,code);if err!=nil{writeError(w,422,"challenge_invalid",err.Error());return}; out:=map[string]any{"delivery_id":otp.ID,"expires_at":otp.ExpiresAt.UTC()};if a.DebugOTP{out["debug_code"]=code};writeJSON(w,202,out) }
func (a *API) otpVerify(w http.ResponseWriter, r *http.Request) { var in struct{ChallengeID string `json:"challenge_id"`; Code string `json:"code"`};if !decode(r,&in){writeError(w,400,"invalid_json","Invalid JSON");return};if err:=a.Store.VerifyOTP(in.ChallengeID,in.Code);err!=nil{writeError(w,422,"otp_invalid",err.Error());return};writeJSON(w,200,map[string]any{"verified":true}) }
func (a *API) complete(w http.ResponseWriter, r *http.Request) { var in struct{ChallengeID string `json:"challenge_id"`; Email string `json:"email"`; Password string `json:"password"`};if !decode(r,&in){writeError(w,400,"invalid_json","Invalid JSON");return};u,err:=a.Store.CreateUser(in.ChallengeID,in.Email,in.Password);if err!=nil{status:=422;if strings.Contains(err.Error(),"already_registered"){status=409};writeError(w,status,err.Error(),err.Error());return};writeJSON(w,201,map[string]any{"user_id":u.ID,"email":u.Email}) }
