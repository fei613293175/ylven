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
	ID                string    `json:"id"`
	Email             string    `json:"email"`
	Purpose           string    `json:"purpose"`
	TurnstileVerified bool      `json:"turnstile_verified"`
	OTPVerified       bool      `json:"otp_verified"`
	ExpiresAt         time.Time `json:"expires_at"`
	Consumed          bool      `json:"consumed"`
}

type OTP struct {
	ID          string    `json:"id"`
	DeliveryID  string    `json:"-"`
	ChallengeID string    `json:"challenge_id"`
	Email       string    `json:"email"`
	Digest      string    `json:"digest"`
	ExpiresAt   time.Time `json:"expires_at"`
	Attempts    int       `json:"attempts"`
	Consumed    bool      `json:"consumed"`
	CreatedAt   time.Time `json:"created_at"`
}

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"password_hash"`
	CreatedAt    time.Time `json:"created_at"`
	Status       string    `json:"status"`
}

type Session struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id"`
	AccessDigest     string    `json:"access_digest"`
	RefreshDigest    string    `json:"refresh_digest"`
	AccessExpiresAt  time.Time `json:"access_expires_at"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
	RefreshConsumed  bool      `json:"refresh_consumed"`
	DeviceID         string    `json:"device_id"`
	CreatedAt        time.Time `json:"created_at"`
	Revoked          bool      `json:"revoked"`
}

type UserView struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	Status    string    `json:"status"`
}

type Workspace struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type SessionView struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id"`
	AccessExpiresAt  time.Time `json:"access_expires_at"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
	DeviceID         string    `json:"device_id"`
	CreatedAt        time.Time `json:"created_at"`
	Revoked          bool      `json:"revoked"`
}

type AdminSessionView struct {
	ID          string    `json:"id"`
	AdminUserID string    `json:"admin_user_id"`
	ExpiresAt   time.Time `json:"expires_at"`
	CreatedAt   time.Time `json:"created_at"`
	Revoked     bool      `json:"revoked"`
}

type AuditEvent struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Email     string    `json:"email,omitempty"`
	SessionID string    `json:"session_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Role struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
type AdminUser struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"password_hash,omitempty"`
	RoleIDs      []string  `json:"role_ids"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}
type AdminSession struct {
	ID           string    `json:"id"`
	AdminUserID  string    `json:"admin_user_id"`
	AccessDigest string    `json:"access_digest"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
	Revoked      bool      `json:"revoked"`
}
type StepUpChallenge struct {
	ID             string    `json:"id"`
	AdminSessionID string    `json:"admin_session_id"`
	TokenDigest    string    `json:"token_digest"`
	ExpiresAt      time.Time `json:"expires_at"`
	Consumed       bool      `json:"consumed"`
}
type EmailTemplate struct {
	Key       string    `json:"key"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	Version   int64     `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
}
type NotificationDelivery struct {
	ID          string    `json:"id"`
	TemplateKey string    `json:"template_key"`
	Recipient   string    `json:"recipient"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type state struct {
	Challenges             map[string]Challenge         `json:"challenges"`
	OTPs                   map[string]OTP               `json:"otps"`
	Users                  map[string]User              `json:"users"`
	Sessions               map[string]Session           `json:"sessions"`
	RateLimits             map[string][]time.Time       `json:"rate_limits"`
	Audit                  []AuditEvent                 `json:"audit"`
	Settings               map[string]map[string]string `json:"settings"`
	Roles                  map[string]Role              `json:"roles"`
	AdminUsers             map[string]AdminUser         `json:"admin_users"`
	AdminSessions          map[string]AdminSession      `json:"admin_sessions"`
	StepUpChallenges       map[string]StepUpChallenge   `json:"step_up_challenges"`
	EmailTemplates         map[string]EmailTemplate     `json:"email_templates"`
	NotificationDeliveries []NotificationDelivery       `json:"notification_deliveries"`
	Workspaces             map[string]Workspace          `json:"workspaces"`
}

type Store struct {
	mu   sync.Mutex
	path string
	data state
}

func NewStore(path string) (*Store, error) {
	s := &Store{path: path, data: state{Challenges: map[string]Challenge{}, OTPs: map[string]OTP{}, Users: map[string]User{}, Sessions: map[string]Session{}, RateLimits: map[string][]time.Time{}, Settings: defaultSettings(), Roles: defaultRoles(), AdminUsers: map[string]AdminUser{}, AdminSessions: map[string]AdminSession{}, StepUpChallenges: map[string]StepUpChallenge{}, EmailTemplates: defaultEmailTemplates(), NotificationDeliveries: []NotificationDelivery{}, Workspaces: map[string]Workspace{}}}
	if path == "" {
		return s, nil
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &s.data); err != nil {
		return nil, fmt.Errorf("identity store: %w", err)
	}
	if s.data.Challenges == nil {
		s.data.Challenges = map[string]Challenge{}
	}
	if s.data.OTPs == nil {
		s.data.OTPs = map[string]OTP{}
	}
	if s.data.Users == nil {
		s.data.Users = map[string]User{}
	}
	if s.data.Sessions == nil {
		s.data.Sessions = map[string]Session{}
	}
	if s.data.RateLimits == nil {
		s.data.RateLimits = map[string][]time.Time{}
	}
	if s.data.Settings == nil {
		s.data.Settings = defaultSettings()
	}
	if s.data.Roles == nil {
		s.data.Roles = defaultRoles()
	}
	if s.data.AdminUsers == nil {
		s.data.AdminUsers = map[string]AdminUser{}
	}
	if s.data.AdminSessions == nil {
		s.data.AdminSessions = map[string]AdminSession{}
	}
	if s.data.StepUpChallenges == nil {
		s.data.StepUpChallenges = map[string]StepUpChallenge{}
	}
	if s.data.EmailTemplates == nil {
		s.data.EmailTemplates = defaultEmailTemplates()
	}
	if s.data.NotificationDeliveries == nil {
		s.data.NotificationDeliveries = []NotificationDelivery{}
	}
	if s.data.Workspaces == nil {
		s.data.Workspaces = map[string]Workspace{}
	}
	return s, nil
}

func (s *Store) persistLocked() error {
	if s.path == "" {
		return nil
	}
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func NormalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if email == "" || !emailPattern.MatchString(email) {
		return "", errors.New("invalid email format")
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email {
		return "", errors.New("invalid email format")
	}
	return email, nil
}

func randomToken(bytes int) (string, error) {
	b := make([]byte, bytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *Store) appendAuditLocked(eventType, email, sessionID string) {
	id, _ := randomToken(8)
	s.data.Audit = append(s.data.Audit, AuditEvent{
		ID: id, Type: eventType, Email: email, SessionID: sessionID, CreatedAt: time.Now().UTC(),
	})
}

func (s *Store) CreateChallenge(email, purpose string) (Challenge, error) {
	normalized, err := NormalizeEmail(email)
	if err != nil {
		return Challenge{}, err
	}
	id, err := randomToken(16)
	if err != nil {
		return Challenge{}, err
	}
	c := Challenge{ID: id, Email: normalized, Purpose: purpose, ExpiresAt: time.Now().Add(10 * time.Minute)}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Challenges[id] = c
	s.appendAuditLocked("auth_challenge_created:"+purpose, normalized, id)
	return c, s.persistLocked()
}

func (s *Store) VerifyTurnstile(challengeID, token string, valid bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.data.Challenges[challengeID]
	if !ok {
		return errors.New("challenge_not_found")
	}
	if c.ExpiresAt.Before(time.Now()) {
		return errors.New("challenge_expired")
	}
	if c.Consumed {
		return errors.New("challenge_consumed")
	}
	if c.TurnstileVerified {
		return errors.New("turnstile_already_verified")
	}
	if !valid || strings.TrimSpace(token) == "" {
		return errors.New("turnstile_failed")
	}
	c.TurnstileVerified = true
	s.data.Challenges[challengeID] = c
	return s.persistLocked()
}

func (s *Store) CreateOTP(challengeID string, code string) (OTP, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.data.Challenges[challengeID]
	if !ok || c.ExpiresAt.Before(time.Now()) || c.Consumed {
		return OTP{}, errors.New("challenge_not_found")
	}
	if !c.TurnstileVerified {
		return OTP{}, errors.New("turnstile_required")
	}
	id, err := randomToken(16)
	if err != nil {
		return OTP{}, err
	}
	salt, err := randomToken(16)
	if err != nil {
		return OTP{}, err
	}
	now := time.Now().UTC()
	for _, existing := range s.data.OTPs {
		if existing.ChallengeID == challengeID && !existing.Consumed && now.Sub(existing.CreatedAt) < 60*time.Second {
			return OTP{}, errors.New("otp_cooldown")
		}
	}
	o := OTP{ID: id, ChallengeID: challengeID, Email: c.Email, Digest: hashSecret(code, salt), ExpiresAt: now.Add(10 * time.Minute), CreatedAt: now}
	// Store the salt with the digest; it is not secret and is required for verification.
	o.Digest = salt + ":" + o.Digest
	s.data.OTPs[id] = o
	deliveryID, err := randomToken(16)
	if err != nil {
		return OTP{}, err
	}
	o.DeliveryID = deliveryID
	templateKey := "register_otp"
	if c.Purpose == "login" {
		templateKey = "login_otp"
	}
	s.data.NotificationDeliveries = append(s.data.NotificationDeliveries, NotificationDelivery{ID: deliveryID, TemplateKey: templateKey, Recipient: c.Email, Status: "queued", CreatedAt: time.Now().UTC()})
	return o, s.persistLocked()
}

func (s *Store) ChallengeSnapshot(challengeID string) (Challenge, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	challenge, ok := s.data.Challenges[challengeID]
	return challenge, ok
}

func (s *Store) UpdateNotificationDelivery(deliveryID, status string) error {
	if status != "delivered" && status != "failed" && status != "mocked" {
		return errors.New("invalid_delivery_status")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.data.NotificationDeliveries {
		if s.data.NotificationDeliveries[index].ID == deliveryID {
			s.data.NotificationDeliveries[index].Status = status
			return s.persistLocked()
		}
	}
	return errors.New("delivery_not_found")
}

func (s *Store) EmailTemplateFor(key string) (EmailTemplate, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	template, ok := s.data.EmailTemplates[key]
	return template, ok
}

func (s *Store) VerifyOTP(challengeID, code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, otp := range s.data.OTPs {
		if otp.ChallengeID != challengeID || otp.Consumed {
			continue
		}
		if otp.ExpiresAt.Before(time.Now()) || otp.Attempts >= 5 {
			return errors.New("otp_expired_or_locked")
		}
		otp.Attempts++
		parts := strings.SplitN(otp.Digest, ":", 2)
		valid := len(parts) == 2 && subtle.ConstantTimeCompare([]byte(hashSecret(code, parts[0])), []byte(parts[1])) == 1
		if !valid {
			s.data.OTPs[id] = otp
			_ = s.persistLocked()
			return errors.New("otp_invalid")
		}
		otp.Consumed = true
		s.data.OTPs[id] = otp
		c := s.data.Challenges[challengeID]
		c.OTPVerified = true
		s.data.Challenges[challengeID] = c
		s.appendAuditLocked("otp_verified:"+c.Purpose, c.Email, challengeID)
		return s.persistLocked()
	}
	return errors.New("otp_not_found")
}

func (s *Store) CreateUser(challengeID, email, password string) (User, error) {
	normalized, err := NormalizeEmail(email)
	if err != nil {
		return User{}, err
	}
	if len(password) < 8 {
		return User{}, errors.New("password_too_short")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.data.Challenges[challengeID]
	if !ok || c.Email != normalized || !c.TurnstileVerified || !c.OTPVerified || c.ExpiresAt.Before(time.Now()) || c.Consumed {
		return User{}, errors.New("registration_not_verified")
	}
	if _, exists := s.data.Users[normalized]; exists {
		return User{}, errors.New("email_already_registered")
	}
	salt, err := randomToken(16)
	if err != nil {
		return User{}, err
	}
	uuid, err := randomToken(16)
	if err != nil {
		return User{}, err
	}
	u := User{ID: uuid, Email: normalized, PasswordHash: salt + ":" + hashSecret(password, salt), CreatedAt: time.Now().UTC(), Status: "active"}
	s.data.Users[normalized] = u
	c.Consumed = true
	s.data.Challenges[challengeID] = c
	s.appendAuditLocked("user_registered", normalized, challengeID)
	return u, s.persistLocked()
}

func (s *Store) EnsureWorkspace(userID string) (Workspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.data.Workspaces[userID]; ok {
		return existing, nil
	}
	id, err := randomToken(16)
	if err != nil {
		return Workspace{}, err
	}
	workspace := Workspace{ID: id, UserID: userID, Name: "个人工作区", CreatedAt: time.Now().UTC()}
	s.data.Workspaces[userID] = workspace
	s.appendAuditLocked("personal_workspace_initialized", "", userID)
	return workspace, s.persistLocked()
}

func (s *Store) CreateSessionForUser(email string) (Session, string, string, error) {
	normalized, err := NormalizeEmail(email)
	if err != nil {
		return Session{}, "", "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.data.Users[normalized]
	if !ok || u.Status != "active" {
		return Session{}, "", "", errors.New("login_not_verified")
	}
	return s.createSessionLocked(u, normalized)
}

func (s *Store) createSessionLocked(u User, email string) (Session, string, string, error) {
	id, err := randomToken(16)
	if err != nil { return Session{}, "", "", err }
	access, err := randomToken(32)
	if err != nil { return Session{}, "", "", err }
	refresh, err := randomToken(32)
	if err != nil { return Session{}, "", "", err }
	now := time.Now().UTC()
	session := Session{ID: id, UserID: u.ID, AccessDigest: digestToken(access), RefreshDigest: digestToken(refresh), AccessExpiresAt: now.Add(15 * time.Minute), RefreshExpiresAt: now.Add(30 * 24 * time.Hour), DeviceID: "device-" + id[:8], CreatedAt: now}
	s.data.Sessions[id] = session
	s.appendAuditLocked("user_login", email, id)
	return session, access, refresh, s.persistLocked()
}

func (s *Store) CreateSession(challengeID, email string) (Session, string, string, error) {
	normalized, err := NormalizeEmail(email)
	if err != nil {
		return Session{}, "", "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.data.Challenges[challengeID]
	if !ok || c.Email != normalized || c.Purpose != "login" || !c.TurnstileVerified || !c.OTPVerified || c.ExpiresAt.Before(time.Now()) || c.Consumed {
		return Session{}, "", "", errors.New("login_not_verified")
	}
	u, ok := s.data.Users[normalized]
	if !ok || u.Status != "active" {
		return Session{}, "", "", errors.New("login_not_verified")
	}
	session, access, refresh, err := s.createSessionLocked(u, normalized)
	if err != nil { return Session{}, "", "", err }
	c.Consumed = true
	s.data.Challenges[challengeID] = c
	return session, access, refresh, s.persistLocked()
}

func (s *Store) RotateSession(refreshToken string) (Session, string, string, error) {
	digest := digestToken(refreshToken)
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, current := range s.data.Sessions {
		if current.RefreshDigest != digest || current.RefreshConsumed || current.RefreshExpiresAt.Before(time.Now()) {
			continue
		}
		current.RefreshConsumed = true
		s.data.Sessions[id] = current
		access, err := randomToken(32)
		if err != nil {
			return Session{}, "", "", err
		}
		refresh, err := randomToken(32)
		if err != nil {
			return Session{}, "", "", err
		}
		newID, err := randomToken(16)
		if err != nil {
			return Session{}, "", "", err
		}
		now := time.Now().UTC()
		next := Session{ID: newID, UserID: current.UserID, AccessDigest: digestToken(access), RefreshDigest: digestToken(refresh), AccessExpiresAt: now.Add(15 * time.Minute), RefreshExpiresAt: now.Add(30 * 24 * time.Hour), DeviceID: current.DeviceID, CreatedAt: now}
		s.data.Sessions[newID] = next
		s.appendAuditLocked("session_refreshed", "", newID)
		return next, access, refresh, s.persistLocked()
	}
	return Session{}, "", "", errors.New("refresh_token_invalid")
}

func digestToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *Store) sessionByAccessLocked(access string) (Session, bool) {
	d := digestToken(access)
	for _, item := range s.data.Sessions {
		if item.AccessDigest == d && !item.Revoked && item.AccessExpiresAt.After(time.Now()) {
			return item, true
		}
	}
	return Session{}, false
}

func (s *Store) Logout(access string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.sessionByAccessLocked(access)
	if !ok {
		return errors.New("session_invalid")
	}
	item.Revoked = true
	s.data.Sessions[item.ID] = item
	s.data.Audit = append(s.data.Audit, AuditEvent{ID: item.ID, Type: "logout", SessionID: item.ID, CreatedAt: time.Now().UTC()})
	return s.persistLocked()
}
func (s *Store) LogoutAll(access string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.sessionByAccessLocked(access)
	if !ok {
		return errors.New("session_invalid")
	}
	for id, session := range s.data.Sessions {
		if session.UserID == item.UserID {
			session.Revoked = true
			s.data.Sessions[id] = session
		}
	}
	s.data.Audit = append(s.data.Audit, AuditEvent{ID: item.ID, Type: "logout_all", SessionID: item.ID, CreatedAt: time.Now().UTC()})
	return s.persistLocked()
}
func sessionView(session Session) SessionView {
	return SessionView{ID: session.ID, UserID: session.UserID, AccessExpiresAt: session.AccessExpiresAt, RefreshExpiresAt: session.RefreshExpiresAt, DeviceID: session.DeviceID, CreatedAt: session.CreatedAt, Revoked: session.Revoked}
}

func userView(user User) UserView {
	return UserView{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt, Status: user.Status}
}

func (s *Store) ListSessions(access string) ([]SessionView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.sessionByAccessLocked(access)
	if !ok {
		return nil, errors.New("session_invalid")
	}
	result := []SessionView{}
	for _, session := range s.data.Sessions {
		if session.UserID == item.UserID && !session.Revoked {
			result = append(result, sessionView(session))
		}
	}
	return result, nil
}

// CurrentAccount returns the authenticated account and its active session.
// The access token is only used for lookup and is never persisted or returned.
func (s *Store) CurrentAccount(access string) (UserView, SessionView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.sessionByAccessLocked(access)
	if !ok {
		return UserView{}, SessionView{}, errors.New("session_invalid")
	}
	var user User
	for _, candidate := range s.data.Users {
		if candidate.ID == item.UserID {
			user = candidate
			break
		}
	}
	if user.ID == "" || user.Status != "active" {
		return UserView{}, SessionView{}, errors.New("session_invalid")
	}
	return userView(user), sessionView(item), nil
}
func (s *Store) RevokeSession(access, targetID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.sessionByAccessLocked(access)
	if !ok {
		return errors.New("session_invalid")
	}
	target, exists := s.data.Sessions[targetID]
	if !exists || target.UserID != item.UserID {
		return errors.New("session_not_found")
	}
	target.Revoked = true
	target.RefreshConsumed = true
	s.data.Sessions[targetID] = target
	s.data.Audit = append(s.data.Audit, AuditEvent{ID: targetID, Type: "session_revoked", SessionID: targetID, CreatedAt: time.Now().UTC()})
	return s.persistLocked()
}
func (s *Store) AllowAttempt(key string, limit int, window time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	recent := []time.Time{}
	for _, when := range s.data.RateLimits[key] {
		if now.Sub(when) < window {
			recent = append(recent, when)
		}
	}
	if len(recent) >= limit {
		s.data.RateLimits[key] = recent
		s.appendAuditLocked("auth_rate_limited", key, "")
		_ = s.persistLocked()
		return false, nil
	}
	s.data.RateLimits[key] = append(recent, now)
	return true, s.persistLocked()
}
func (s *Store) AuditSnapshot() []AuditEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]AuditEvent(nil), s.data.Audit...)
}

func defaultSettings() map[string]map[string]string {
	return map[string]map[string]string{"email": {"provider": "", "sender": "", "secret_reference": ""}, "turnstile": {"site_key_reference": "", "secret_reference": ""}, "otp-policy": {"ttl_seconds": "600", "send_limit": "5", "cooldown_seconds": "60"}}
}
func (s *Store) GetSetting(name string) (map[string]string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.data.Settings[name]
	if !ok {
		return nil, false
	}
	copyValue := map[string]string{}
	for k, v := range value {
		copyValue[k] = v
	}
	return copyValue, true
}
func (s *Store) PutSetting(name string, value map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data.Settings[name]; !ok {
		return errors.New("setting_not_found")
	}
	for key := range value {
		if strings.Contains(strings.ToLower(key), "secret") && !strings.Contains(strings.ToLower(key), "reference") {
			return errors.New("raw_secret_forbidden")
		}
	}
	s.data.Settings[name] = value
	id, _ := randomToken(8)
	s.data.Audit = append(s.data.Audit, AuditEvent{ID: id, Type: "setting_updated:" + name, CreatedAt: time.Now().UTC()})
	return s.persistLocked()
}
func (s *Store) ListUsers() []UserView {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]UserView, 0, len(s.data.Users))
	for _, user := range s.data.Users {
		result = append(result, userView(user))
	}
	return result
}
func (s *Store) UserDetail(id string) (UserView, []SessionView, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, user := range s.data.Users {
		if user.ID == id {
			sessions := []SessionView{}
			for _, session := range s.data.Sessions {
				if session.UserID == id {
					sessions = append(sessions, sessionView(session))
				}
			}
			return userView(user), sessions, true
		}
	}
	return UserView{}, nil, false
}

func defaultRoles() map[string]Role {
	now := time.Now().UTC()
	return map[string]Role{
		"superadmin": {ID: "superadmin", Name: "Super Administrator", Permissions: []string{"*"}, CreatedAt: now, UpdatedAt: now},
		"support":    {ID: "support", Name: "Support", Permissions: []string{"users:read", "notifications:read"}, CreatedAt: now, UpdatedAt: now},
	}
}

func defaultEmailTemplates() map[string]EmailTemplate {
	now := time.Now().UTC()
	return map[string]EmailTemplate{
		"register_otp": {Key: "register_otp", Subject: "Verify your YLVEN account", Body: "Your verification code is {{code}}.", Version: 1, UpdatedAt: now},
		"login_otp":    {Key: "login_otp", Subject: "Your YLVEN sign-in code", Body: "Your sign-in code is {{code}}.", Version: 1, UpdatedAt: now},
	}
}

func verifyPassword(encoded, password string) bool {
	parts := strings.SplitN(encoded, ":", 2)
	return len(parts) == 2 && subtle.ConstantTimeCompare([]byte(hashSecret(password, parts[0])), []byte(parts[1])) == 1
}

func (s *Store) BootstrapAdmin(email, password string) (AdminUser, error) {
	normalized, err := NormalizeEmail(email)
	if err != nil {
		return AdminUser{}, err
	}
	if len(password) < 8 {
		return AdminUser{}, errors.New("admin_password_too_short")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.data.AdminUsers[normalized]; ok {
		if !verifyPassword(existing.PasswordHash, password) {
			salt, saltErr := randomToken(16)
			if saltErr != nil {
				return AdminUser{}, saltErr
			}
			existing.PasswordHash = salt + ":" + hashSecret(password, salt)
			s.data.AdminUsers[normalized] = existing
			if persistErr := s.persistLocked(); persistErr != nil {
				return AdminUser{}, persistErr
			}
		}
		existing.PasswordHash = ""
		return existing, nil
	}
	salt, err := randomToken(16)
	if err != nil {
		return AdminUser{}, err
	}
	id, err := randomToken(16)
	if err != nil {
		return AdminUser{}, err
	}
	admin := AdminUser{ID: id, Email: normalized, PasswordHash: salt + ":" + hashSecret(password, salt), RoleIDs: []string{"superadmin"}, Status: "active", CreatedAt: time.Now().UTC()}
	s.data.AdminUsers[normalized] = admin
	if err := s.persistLocked(); err != nil {
		return AdminUser{}, err
	}
	admin.PasswordHash = ""
	return admin, nil
}

func (s *Store) AdminLogin(email, password string) (AdminSession, string, error) {
	normalized, err := NormalizeEmail(email)
	if err != nil {
		return AdminSession{}, "", errors.New("admin_credentials_invalid")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	admin, ok := s.data.AdminUsers[normalized]
	if !ok || admin.Status != "active" || !verifyPassword(admin.PasswordHash, password) {
		return AdminSession{}, "", errors.New("admin_credentials_invalid")
	}
	id, err := randomToken(16)
	if err != nil {
		return AdminSession{}, "", err
	}
	token, err := randomToken(32)
	if err != nil {
		return AdminSession{}, "", err
	}
	now := time.Now().UTC()
	session := AdminSession{ID: id, AdminUserID: admin.ID, AccessDigest: digestToken(token), ExpiresAt: now.Add(8 * time.Hour), CreatedAt: now}
	s.data.AdminSessions[id] = session
	auditID, _ := randomToken(8)
	s.data.Audit = append(s.data.Audit, AuditEvent{ID: auditID, Type: "admin_login", Email: normalized, SessionID: id, CreatedAt: now})
	return session, token, s.persistLocked()
}

func (s *Store) adminByAccessLocked(access string) (AdminUser, AdminSession, bool) {
	digest := digestToken(access)
	for _, session := range s.data.AdminSessions {
		if session.AccessDigest != digest || session.Revoked || !session.ExpiresAt.After(time.Now()) {
			continue
		}
		for _, admin := range s.data.AdminUsers {
			if admin.ID == session.AdminUserID && admin.Status == "active" {
				return admin, session, true
			}
		}
	}
	return AdminUser{}, AdminSession{}, false
}

func permissionGranted(roles map[string]Role, roleIDs []string, permission string) bool {
	for _, roleID := range roleIDs {
		role, ok := roles[roleID]
		if !ok {
			continue
		}
		for _, item := range role.Permissions {
			if item == "*" || item == permission {
				return true
			}
		}
	}
	return false
}
func (s *Store) AuthorizeAdmin(access, permission string) (AdminUser, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	admin, _, ok := s.adminByAccessLocked(access)
	if !ok || !permissionGranted(s.data.Roles, admin.RoleIDs, permission) {
		return AdminUser{}, false
	}
	admin.PasswordHash = ""
	return admin, true
}

func (s *Store) AdminLogout(access string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	admin, session, ok := s.adminByAccessLocked(access)
	if !ok {
		return errors.New("admin_session_invalid")
	}
	session.Revoked = true
	s.data.AdminSessions[session.ID] = session
	auditID, _ := randomToken(8)
	s.data.Audit = append(s.data.Audit, AuditEvent{ID: auditID, Type: "admin_logout", Email: admin.Email, SessionID: session.ID, CreatedAt: time.Now().UTC()})
	return s.persistLocked()
}

func (s *Store) CreateStepUp(access, password string) (string, time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	admin, session, ok := s.adminByAccessLocked(access)
	if !ok || !verifyPassword(admin.PasswordHash, password) {
		return "", time.Time{}, errors.New("step_up_failed")
	}
	id, err := randomToken(16)
	if err != nil {
		return "", time.Time{}, err
	}
	token, err := randomToken(32)
	if err != nil {
		return "", time.Time{}, err
	}
	expires := time.Now().UTC().Add(5 * time.Minute)
	s.data.StepUpChallenges[id] = StepUpChallenge{ID: id, AdminSessionID: session.ID, TokenDigest: digestToken(token), ExpiresAt: expires}
	return token, expires, s.persistLocked()
}

func (s *Store) ConsumeStepUp(access, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, session, ok := s.adminByAccessLocked(access)
	if !ok {
		return errors.New("admin_session_invalid")
	}
	digest := digestToken(token)
	for id, item := range s.data.StepUpChallenges {
		if item.AdminSessionID == session.ID && item.TokenDigest == digest && !item.Consumed && item.ExpiresAt.After(time.Now()) {
			item.Consumed = true
			s.data.StepUpChallenges[id] = item
			return s.persistLocked()
		}
	}
	return errors.New("step_up_required")
}

func (s *Store) SetUserStatus(userID, status, actorID string) (User, error) {
	if status != "active" && status != "disabled" {
		return User{}, errors.New("invalid_user_status")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for email, user := range s.data.Users {
		if user.ID != userID {
			continue
		}
		user.Status = status
		s.data.Users[email] = user
		if status == "disabled" {
			for id, session := range s.data.Sessions {
				if session.UserID == userID {
					session.Revoked = true
					session.RefreshConsumed = true
					s.data.Sessions[id] = session
				}
			}
		}
		auditID, _ := randomToken(8)
		s.data.Audit = append(s.data.Audit, AuditEvent{ID: auditID, Type: "user_status:" + status, Email: user.Email, SessionID: actorID, CreatedAt: time.Now().UTC()})
		user.PasswordHash = ""
		return user, s.persistLocked()
	}
	return User{}, errors.New("user_not_found")
}

func (s *Store) ListRoles() []Role {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]Role, 0, len(s.data.Roles))
	for _, role := range s.data.Roles {
		role.Permissions = append([]string(nil), role.Permissions...)
		result = append(result, role)
	}
	return result
}
func (s *Store) PutRole(role Role, actorID string) (Role, error) {
	role.ID = strings.TrimSpace(role.ID)
	role.Name = strings.TrimSpace(role.Name)
	if role.ID == "" || role.Name == "" || len(role.Permissions) == 0 {
		return Role{}, errors.New("invalid_role")
	}
	for _, p := range role.Permissions {
		if strings.TrimSpace(p) == "" {
			return Role{}, errors.New("invalid_permission")
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if old, ok := s.data.Roles[role.ID]; ok {
		role.CreatedAt = old.CreatedAt
	} else {
		role.CreatedAt = now
	}
	role.UpdatedAt = now
	s.data.Roles[role.ID] = role
	auditID, _ := randomToken(8)
	s.data.Audit = append(s.data.Audit, AuditEvent{ID: auditID, Type: "role_updated:" + role.ID, SessionID: actorID, CreatedAt: now})
	return role, s.persistLocked()
}

func (s *Store) ListAdmins() []AdminUser {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]AdminUser, 0, len(s.data.AdminUsers))
	for _, admin := range s.data.AdminUsers {
		admin.PasswordHash = ""
		result = append(result, admin)
	}
	return result
}
func (s *Store) ListAdminSessions() []AdminSessionView {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]AdminSessionView, 0, len(s.data.AdminSessions))
	for _, session := range s.data.AdminSessions {
		result = append(result, AdminSessionView{ID: session.ID, AdminUserID: session.AdminUserID, ExpiresAt: session.ExpiresAt, CreatedAt: session.CreatedAt, Revoked: session.Revoked})
	}
	return result
}
func (s *Store) ListEmailTemplates() ([]EmailTemplate, []NotificationDelivery) {
	s.mu.Lock()
	defer s.mu.Unlock()
	templates := make([]EmailTemplate, 0, len(s.data.EmailTemplates))
	for _, item := range s.data.EmailTemplates {
		templates = append(templates, item)
	}
	deliveries := append([]NotificationDelivery(nil), s.data.NotificationDeliveries...)
	return templates, deliveries
}
func (s *Store) PutEmailTemplate(template EmailTemplate, actorID string) (EmailTemplate, error) {
	template.Key = strings.TrimSpace(template.Key)
	template.Subject = strings.TrimSpace(template.Subject)
	template.Body = strings.TrimSpace(template.Body)
	if template.Key == "" || template.Subject == "" || template.Body == "" {
		return EmailTemplate{}, errors.New("invalid_email_template")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, exists := s.data.EmailTemplates[template.Key]
	if exists && template.Version != current.Version {
		return EmailTemplate{}, errors.New("template_version_conflict")
	}
	template.Version = current.Version + 1
	if !exists {
		template.Version = 1
	}
	template.UpdatedAt = time.Now().UTC()
	s.data.EmailTemplates[template.Key] = template
	auditID, _ := randomToken(8)
	s.data.Audit = append(s.data.Audit, AuditEvent{ID: auditID, Type: "email_template_updated:" + template.Key, SessionID: actorID, CreatedAt: template.UpdatedAt})
	return template, s.persistLocked()
}

func hashSecret(secret, salt string) string {
	key := []byte(salt)
	block := []byte(secret)
	var result [32]byte
	for i := 1; i <= passwordIterations; i++ {
		h := hmac.New(sha256.New, key)
		h.Write(block)
		h.Write([]byte{byte(i >> 24), byte(i >> 16), byte(i >> 8), byte(i)})
		u := h.Sum(nil)
		for j := 0; j < len(result); j++ {
			result[j] ^= u[j]
		}
		block = u
	}
	return base64.RawStdEncoding.EncodeToString(result[:])
}
