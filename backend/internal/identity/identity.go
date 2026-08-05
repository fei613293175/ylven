package identity

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

const passwordIterations = 210000

var emailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

type Challenge struct {
	ID string `json:"id"`
	Email string `json:"email"`
	Purpose string `json:"purpose"`
	TurnstileVerified bool `json:"turnstile_verified"`
	OTPVerified bool `json:"otp_verified"`
	ExpiresAt time.Time `json:"expires_at"`
	Consumed bool `json:"consumed"`
}

type OTP struct {
	ID string `json:"id"`
	ChallengeID string `json:"challenge_id"`
	Email string `json:"email"`
	Digest string `json:"digest"`
	ExpiresAt time.Time `json:"expires_at"`
	Attempts int `json:"attempts"`
	Consumed bool `json:"consumed"`
}

type User struct {
	ID string `json:"id"`
	Email string `json:"email"`
	PasswordHash string `json:"password_hash"`
	CreatedAt time.Time `json:"created_at"`
}

type Session struct {
	ID string `json:"id"`
	UserID string `json:"user_id"`
	AccessDigest string `json:"access_digest"`
	RefreshDigest string `json:"refresh_digest"`
	AccessExpiresAt time.Time `json:"access_expires_at"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
	RefreshConsumed bool `json:"refresh_consumed"`
	DeviceID string `json:"device_id"`
	CreatedAt time.Time `json:"created_at"`
	Revoked bool `json:"revoked"`
}

type AuditEvent struct { ID string `json:"id"`; Type string `json:"type"`; Email string `json:"email,omitempty"`; SessionID string `json:"session_id,omitempty"`; CreatedAt time.Time `json:"created_at"` }

type state struct {
	Challenges map[string]Challenge `json:"challenges"`
	OTPs map[string]OTP `json:"otps"`
	Users map[string]User `json:"users"`
	Sessions map[string]Session `json:"sessions"`
	RateLimits map[string][]time.Time `json:"rate_limits"`
	Audit []AuditEvent `json:"audit"`
}

type Store struct {
	mu sync.Mutex
	path string
	data state
}

func NewStore(path string) (*Store, error) {
	s := &Store{path: path, data: state{Challenges: map[string]Challenge{}, OTPs: map[string]OTP{}, Users: map[string]User{}, Sessions: map[string]Session{}, RateLimits: map[string][]time.Time{}}}
	if path == "" { return s, nil }
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) { return s, nil }
	if err != nil { return nil, err }
	if err := json.Unmarshal(b, &s.data); err != nil { return nil, fmt.Errorf("identity store: %w", err) }
	if s.data.Challenges == nil { s.data.Challenges = map[string]Challenge{} }
	if s.data.OTPs == nil { s.data.OTPs = map[string]OTP{} }
	if s.data.Users == nil { s.data.Users = map[string]User{} }
	if s.data.Sessions == nil { s.data.Sessions = map[string]Session{} }
	if s.data.RateLimits == nil { s.data.RateLimits = map[string][]time.Time{} }
	return s, nil
}

func (s *Store) persistLocked() error {
	if s.path == "" { return nil }
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil { return err }
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0600); err != nil { return err }
	return os.Rename(tmp, s.path)
}

func NormalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if email == "" || !emailPattern.MatchString(email) { return "", errors.New("invalid email format") }
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email { return "", errors.New("invalid email format") }
	return email, nil
}

func randomToken(bytes int) (string, error) {
	b := make([]byte, bytes)
	if _, err := rand.Read(b); err != nil { return "", err }
	return hex.EncodeToString(b), nil
}

func (s *Store) CreateChallenge(email, purpose string) (Challenge, error) {
	normalized, err := NormalizeEmail(email); if err != nil { return Challenge{}, err }
	id, err := randomToken(16); if err != nil { return Challenge{}, err }
	c := Challenge{ID: id, Email: normalized, Purpose: purpose, ExpiresAt: time.Now().Add(10*time.Minute)}
	s.mu.Lock(); defer s.mu.Unlock(); s.data.Challenges[id] = c
	return c, s.persistLocked()
}

func (s *Store) VerifyTurnstile(challengeID, token string, valid bool) error {
	s.mu.Lock(); defer s.mu.Unlock()
	c, ok := s.data.Challenges[challengeID]
	if !ok || c.ExpiresAt.Before(time.Now()) || c.Consumed { return errors.New("challenge_not_found") }
	if !valid || strings.TrimSpace(token) == "" { return errors.New("turnstile_failed") }
	c.TurnstileVerified = true; s.data.Challenges[challengeID] = c
	return s.persistLocked()
}

func (s *Store) CreateOTP(challengeID string, code string) (OTP, error) {
	s.mu.Lock(); defer s.mu.Unlock()
	c, ok := s.data.Challenges[challengeID]
	if !ok || c.ExpiresAt.Before(time.Now()) || c.Consumed { return OTP{}, errors.New("challenge_not_found") }
	id, err := randomToken(16); if err != nil { return OTP{}, err }
	salt, err := randomToken(16); if err != nil { return OTP{}, err }
	o := OTP{ID:id, ChallengeID:challengeID, Email:c.Email, Digest: hashSecret(code, salt), ExpiresAt:time.Now().Add(10*time.Minute)}
	// Store the salt with the digest; it is not secret and is required for verification.
	o.Digest = salt + ":" + o.Digest
	s.data.OTPs[id] = o
	return o, s.persistLocked()
}

func (s *Store) VerifyOTP(challengeID, code string) error {
	s.mu.Lock(); defer s.mu.Unlock()
	for id, otp := range s.data.OTPs {
		if otp.ChallengeID != challengeID || otp.Consumed { continue }
		if otp.ExpiresAt.Before(time.Now()) || otp.Attempts >= 5 { return errors.New("otp_expired_or_locked") }
		otp.Attempts++
		parts := strings.SplitN(otp.Digest, ":", 2)
		valid := len(parts) == 2 && subtle.ConstantTimeCompare([]byte(hashSecret(code, parts[0])), []byte(parts[1])) == 1
		if !valid { s.data.OTPs[id] = otp; _ = s.persistLocked(); return errors.New("otp_invalid") }
		otp.Consumed = true; s.data.OTPs[id] = otp
		c := s.data.Challenges[challengeID]; c.OTPVerified = true; s.data.Challenges[challengeID] = c
		return s.persistLocked()
	}
	return errors.New("otp_not_found")
}

func (s *Store) CreateUser(challengeID, email, password string) (User, error) {
	normalized, err := NormalizeEmail(email); if err != nil { return User{}, err }
	if len(password) < 8 { return User{}, errors.New("password_too_short") }
	s.mu.Lock(); defer s.mu.Unlock()
	c, ok := s.data.Challenges[challengeID]
	if !ok || c.Email != normalized || !c.TurnstileVerified || !c.OTPVerified || c.ExpiresAt.Before(time.Now()) || c.Consumed { return User{}, errors.New("registration_not_verified") }
	if _, exists := s.data.Users[normalized]; exists { return User{}, errors.New("email_already_registered") }
	salt, err := randomToken(16); if err != nil { return User{}, err }
	uuid, err := randomToken(16); if err != nil { return User{}, err }
	u := User{ID:uuid, Email:normalized, PasswordHash:salt+":"+hashSecret(password,salt), CreatedAt:time.Now().UTC()}
	s.data.Users[normalized] = u; c.Consumed = true; s.data.Challenges[challengeID] = c
	return u, s.persistLocked()
}

func (s *Store) CreateSession(challengeID, email string) (Session, string, string, error) {
	normalized, err := NormalizeEmail(email)
	if err != nil { return Session{}, "", "", err }
	s.mu.Lock(); defer s.mu.Unlock()
	c, ok := s.data.Challenges[challengeID]
	if !ok || c.Email != normalized || c.Purpose != "login" || !c.OTPVerified || c.ExpiresAt.Before(time.Now()) || c.Consumed { return Session{}, "", "", errors.New("login_not_verified") }
	u, ok := s.data.Users[normalized]
	if !ok { return Session{}, "", "", errors.New("login_not_verified") }
	id, err := randomToken(16); if err != nil { return Session{}, "", "", err }
	access, err := randomToken(32); if err != nil { return Session{}, "", "", err }
	refresh, err := randomToken(32); if err != nil { return Session{}, "", "", err }
	now := time.Now().UTC()
	session := Session{ID:id, UserID:u.ID, AccessDigest:digestToken(access), RefreshDigest:digestToken(refresh), AccessExpiresAt:now.Add(15*time.Minute), RefreshExpiresAt:now.Add(30*24*time.Hour), DeviceID:"device-"+id[:8], CreatedAt:now}
	s.data.Sessions[id] = session; c.Consumed = true; s.data.Challenges[challengeID] = c
	return session, access, refresh, s.persistLocked()
}

func (s *Store) RotateSession(refreshToken string) (Session, string, string, error) {
	digest := digestToken(refreshToken)
	s.mu.Lock(); defer s.mu.Unlock()
	for id, current := range s.data.Sessions {
		if current.RefreshDigest != digest || current.RefreshConsumed || current.RefreshExpiresAt.Before(time.Now()) { continue }
		current.RefreshConsumed = true; s.data.Sessions[id] = current
		access, err := randomToken(32); if err != nil { return Session{}, "", "", err }
		refresh, err := randomToken(32); if err != nil { return Session{}, "", "", err }
		newID, err := randomToken(16); if err != nil { return Session{}, "", "", err }
		now := time.Now().UTC()
		next := Session{ID:newID, UserID:current.UserID, AccessDigest:digestToken(access), RefreshDigest:digestToken(refresh), AccessExpiresAt:now.Add(15*time.Minute), RefreshExpiresAt:now.Add(30*24*time.Hour), DeviceID:current.DeviceID, CreatedAt:now}
		s.data.Sessions[newID] = next
		return next, access, refresh, s.persistLocked()
	}
	return Session{}, "", "", errors.New("refresh_token_invalid")
}

func digestToken(token string) string { sum := sha256.Sum256([]byte(token)); return hex.EncodeToString(sum[:]) }

func (s *Store) sessionByAccessLocked(access string) (Session, bool) {
	d := digestToken(access)
	for _, item := range s.data.Sessions { if item.AccessDigest == d && !item.Revoked && item.AccessExpiresAt.After(time.Now()) { return item, true } }
	return Session{}, false
}

func (s *Store) Logout(access string) error { s.mu.Lock(); defer s.mu.Unlock(); item, ok := s.sessionByAccessLocked(access); if !ok { return errors.New("session_invalid") }; item.Revoked=true; s.data.Sessions[item.ID]=item; s.data.Audit=append(s.data.Audit, AuditEvent{ID:item.ID,Type:"logout",SessionID:item.ID,CreatedAt:time.Now().UTC()}); return s.persistLocked() }
func (s *Store) LogoutAll(access string) error { s.mu.Lock(); defer s.mu.Unlock(); item, ok := s.sessionByAccessLocked(access); if !ok { return errors.New("session_invalid") }; for id, session := range s.data.Sessions { if session.UserID==item.UserID { session.Revoked=true; s.data.Sessions[id]=session } }; s.data.Audit=append(s.data.Audit, AuditEvent{ID:item.ID,Type:"logout_all",SessionID:item.ID,CreatedAt:time.Now().UTC()}); return s.persistLocked() }
func (s *Store) ListSessions(access string) ([]Session, error) { s.mu.Lock(); defer s.mu.Unlock(); item, ok := s.sessionByAccessLocked(access); if !ok { return nil, errors.New("session_invalid") }; result:=[]Session{}; for _, session := range s.data.Sessions { if session.UserID==item.UserID && !session.Revoked { session.AccessDigest=""; session.RefreshDigest=""; result=append(result,session) } }; return result,nil }
func (s *Store) RevokeSession(access, targetID string) error { s.mu.Lock(); defer s.mu.Unlock(); item, ok := s.sessionByAccessLocked(access); if !ok { return errors.New("session_invalid") }; target, exists:=s.data.Sessions[targetID]; if !exists || target.UserID!=item.UserID { return errors.New("session_not_found") }; target.Revoked=true; target.RefreshConsumed=true; s.data.Sessions[targetID]=target; s.data.Audit=append(s.data.Audit, AuditEvent{ID:targetID,Type:"session_revoked",SessionID:targetID,CreatedAt:time.Now().UTC()}); return s.persistLocked() }
func (s *Store) AllowAttempt(key string, limit int, window time.Duration) (bool, error) { s.mu.Lock(); defer s.mu.Unlock(); now:=time.Now(); recent:=[]time.Time{}; for _, when:=range s.data.RateLimits[key] { if now.Sub(when)<window { recent=append(recent,when) } }; if len(recent)>=limit { s.data.RateLimits[key]=recent; _=s.persistLocked(); return false,nil }; s.data.RateLimits[key]=append(recent,now); return true,s.persistLocked() }
func (s *Store) AuditSnapshot() []AuditEvent { s.mu.Lock(); defer s.mu.Unlock(); return append([]AuditEvent(nil),s.data.Audit...) }

func hashSecret(secret, salt string) string {
	key := []byte(salt); block := []byte(secret); var result [32]byte
	for i := 1; i <= passwordIterations; i++ {
		h := hmac.New(sha256.New, key); h.Write(block); h.Write([]byte{byte(i>>24), byte(i>>16), byte(i>>8), byte(i)})
		u := h.Sum(nil); for j := 0; j < len(result); j++ { result[j] ^= u[j] }; block = u
	}
	return base64.RawStdEncoding.EncodeToString(result[:])
}
