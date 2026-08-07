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
	"math/big"
	"net/mail"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var citationURLPattern = regexp.MustCompile(`https?://[^\s)\]}>]+`)

func extractCitations(body string) []map[string]string {
	seen := map[string]bool{}
	out := make([]map[string]string, 0)
	for _, value := range citationURLPattern.FindAllString(body, -1) {
		value = strings.TrimRight(value, ".,")
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, map[string]string{"url": value, "title": value})
	}
	return out
}

const passwordIterations = 210000

var emailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

type Challenge struct {
	ID                   string    `json:"id"`
	Email                string    `json:"email"`
	Purpose              string    `json:"purpose"`
	TurnstileVerified    bool      `json:"turnstile_verified"`
	VerificationQuestion string    `json:"verification_question"`
	VerificationDigest   string    `json:"verification_digest"`
	VerificationAttempts int       `json:"verification_attempts"`
	OTPVerified          bool      `json:"otp_verified"`
	ExpiresAt            time.Time `json:"expires_at"`
	Consumed             bool      `json:"consumed"`
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
	SessionID        string    `json:"session_id"`
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

// Conversation, Message and HomeConfig are the P03 conversation-domain
// records. They intentionally live behind the authenticated Store boundary so
// callers can never address another user's records by guessing an ID.
type Conversation struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	Title      string     `json:"title"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	ArchivedAt *time.Time `json:"archived_at,omitempty"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
	Temporary  bool       `json:"temporary,omitempty"`
}

type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	UserID         string    `json:"user_id"`
	Role           string    `json:"role"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"created_at"`
}

type ModelCatalogEntry struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

type HomeConfig struct {
	Announcement       string    `json:"announcement"`
	FeaturedProjectIDs []string  `json:"featured_project_ids"`
	Version            int64     `json:"version"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type MessageRun struct {
	ID                 string     `json:"id"`
	ConversationID     string     `json:"conversation_id"`
	UserID             string     `json:"user_id"`
	UserMessageID      string     `json:"user_message_id"`
	AssistantMessageID string     `json:"assistant_message_id"`
	Model              string     `json:"model"`
	Status             string     `json:"status"`
	ErrorCode          string     `json:"error_code,omitempty"`
	Provider           string     `json:"provider,omitempty"`
	StartedAt          time.Time  `json:"started_at"`
	CompletedAt        *time.Time `json:"completed_at,omitempty"`
	LatencyMs          int64      `json:"latency_ms,omitempty"`
	Cursor             int64      `json:"cursor"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type RunEvent struct {
	ID        int64     `json:"id"`
	RunID     string    `json:"run_id"`
	Type      string    `json:"type"`
	Delta     string    `json:"delta,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type ExportJob struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	MessageID      string    `json:"message_id,omitempty"`
	UserID         string    `json:"user_id"`
	Format         string    `json:"format"`
	Status         string    `json:"status"`
	Content        string    `json:"content,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type Draft struct {
	ConversationID string    `json:"conversation_id"`
	UserID         string    `json:"user_id"`
	Body           string    `json:"body"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type MessageFeedback struct {
	MessageID string    `json:"message_id"`
	UserID    string    `json:"user_id"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
}

type SpeechJob struct {
	ID        string    `json:"id"`
	MessageID string    `json:"message_id"`
	UserID    string    `json:"user_id"`
	Status    string    `json:"status"`
	Provider  string    `json:"provider"`
	CreatedAt time.Time `json:"created_at"`
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
	Workspaces             map[string]Workspace         `json:"workspaces"`
	Conversations          map[string]Conversation      `json:"conversations"`
	Messages               map[string]Message           `json:"messages"`
	ModelCatalog           []ModelCatalogEntry          `json:"model_catalog"`
	HomeConfig             HomeConfig                   `json:"home_config"`
	Runs                   map[string]MessageRun        `json:"runs"`
	RunEvents              map[string][]RunEvent        `json:"run_events"`
	Exports                map[string]ExportJob         `json:"exports"`
	Drafts                 map[string]Draft             `json:"drafts"`
	Metrics                []map[string]any             `json:"chat_metrics"`
	Feedback               map[string]MessageFeedback   `json:"message_feedback"`
	SpeechJobs             map[string]SpeechJob         `json:"speech_jobs"`
}

type Store struct {
	mu   sync.Mutex
	path string
	data state
}

func NewStore(path string) (*Store, error) {
	s := &Store{path: path, data: state{Challenges: map[string]Challenge{}, OTPs: map[string]OTP{}, Users: map[string]User{}, Sessions: map[string]Session{}, RateLimits: map[string][]time.Time{}, Settings: defaultSettings(), Roles: defaultRoles(), AdminUsers: map[string]AdminUser{}, AdminSessions: map[string]AdminSession{}, StepUpChallenges: map[string]StepUpChallenge{}, EmailTemplates: defaultEmailTemplates(), NotificationDeliveries: []NotificationDelivery{}, Workspaces: map[string]Workspace{}, Conversations: map[string]Conversation{}, Messages: map[string]Message{}, Runs: map[string]MessageRun{}, RunEvents: map[string][]RunEvent{}, Exports: map[string]ExportJob{}, Drafts: map[string]Draft{}, Metrics: []map[string]any{}, Feedback: map[string]MessageFeedback{}, SpeechJobs: map[string]SpeechJob{}, ModelCatalog: []ModelCatalogEntry{{ID: "ylven-default", Name: "YLVEN 默认模型", Enabled: true}}, HomeConfig: HomeConfig{Version: 1, UpdatedAt: time.Now().UTC()}}}
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
	if s.data.Conversations == nil {
		s.data.Conversations = map[string]Conversation{}
	}
	if s.data.Messages == nil {
		s.data.Messages = map[string]Message{}
	}
	if s.data.ModelCatalog == nil {
		s.data.ModelCatalog = []ModelCatalogEntry{{ID: "ylven-default", Name: "YLVEN 默认模型", Enabled: true}}
	}
	if s.data.HomeConfig.UpdatedAt.IsZero() {
		s.data.HomeConfig = HomeConfig{Version: 1, UpdatedAt: time.Now().UTC()}
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

func randomInt(max int) (int, error) {
	if max <= 0 {
		return 0, errors.New("invalid random range")
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, err
	}
	return int(n.Int64()), nil
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
	left, err := randomInt(8)
	if err != nil {
		return Challenge{}, err
	}
	right, err := randomInt(8)
	if err != nil {
		return Challenge{}, err
	}
	answer := left + right
	salt, err := randomToken(16)
	if err != nil {
		return Challenge{}, err
	}
	c := Challenge{ID: id, Email: normalized, Purpose: purpose, ExpiresAt: time.Now().Add(10 * time.Minute), VerificationQuestion: fmt.Sprintf("%d + %d = ?", left, right), VerificationDigest: salt + ":" + hashSecret(strconv.Itoa(answer), salt)}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Challenges[id] = c
	s.appendAuditLocked("auth_challenge_created:"+purpose, normalized, id)
	return c, s.persistLocked()
}

// VerifyAnswer checks the first-party arithmetic verification. It is short-lived,
// single-use and locked after five wrong answers. The answer is never persisted.
func (s *Store) VerifyAnswer(challengeID, answer string) error {
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
		return errors.New("verification_already_verified")
	}
	if c.VerificationAttempts >= 5 {
		return errors.New("verification_locked")
	}
	c.VerificationAttempts++
	parts := strings.SplitN(c.VerificationDigest, ":", 2)
	valid := len(parts) == 2 && subtle.ConstantTimeCompare([]byte(hashSecret(strings.TrimSpace(answer), parts[0])), []byte(parts[1])) == 1
	if !valid {
		s.data.Challenges[challengeID] = c
		_ = s.persistLocked()
		return errors.New("verification_incorrect")
	}
	c.TurnstileVerified = true
	s.data.Challenges[challengeID] = c
	s.appendAuditLocked("verification_completed:"+c.Purpose, c.Email, challengeID)
	return s.persistLocked()
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

func (s *Store) CreateSessionForUser(email string, deviceIDs ...string) (Session, string, string, error) {
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
	return s.createSessionLocked(u, normalized, deviceIDs...)
}

func (s *Store) createSessionLocked(u User, email string, deviceIDs ...string) (Session, string, string, error) {
	id, err := randomToken(16)
	if err != nil {
		return Session{}, "", "", err
	}
	access, err := randomToken(32)
	if err != nil {
		return Session{}, "", "", err
	}
	refresh, err := randomToken(32)
	if err != nil {
		return Session{}, "", "", err
	}
	now := time.Now().UTC()
	deviceID := ""
	if len(deviceIDs) > 0 {
		deviceID = strings.TrimSpace(deviceIDs[0])
		if len(deviceID) > 128 {
			deviceID = deviceID[:128]
		}
	}
	if deviceID == "" {
		deviceID = "device-" + id[:8]
	}
	session := Session{ID: id, UserID: u.ID, AccessDigest: digestToken(access), RefreshDigest: digestToken(refresh), AccessExpiresAt: now.Add(15 * time.Minute), RefreshExpiresAt: now.Add(30 * 24 * time.Hour), DeviceID: deviceID, CreatedAt: now}
	s.data.Sessions[id] = session
	s.appendAuditLocked("user_login", email, id)
	return session, access, refresh, s.persistLocked()
}

func (s *Store) CreateSession(challengeID, email string, deviceIDs ...string) (Session, string, string, error) {
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
	session, access, refresh, err := s.createSessionLocked(u, normalized, deviceIDs...)
	if err != nil {
		return Session{}, "", "", err
	}
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
	return SessionView{ID: session.ID, SessionID: session.ID, UserID: session.UserID, AccessExpiresAt: session.AccessExpiresAt, RefreshExpiresAt: session.RefreshExpiresAt, DeviceID: session.DeviceID, CreatedAt: session.CreatedAt, Revoked: session.Revoked}
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
	latest := map[string]Session{}
	for _, session := range s.data.Sessions {
		if session.UserID != item.UserID || session.Revoked {
			continue
		}
		previous, exists := latest[session.DeviceID]
		if !exists || (previous.RefreshConsumed && !session.RefreshConsumed) ||
			(previous.RefreshConsumed == session.RefreshConsumed && session.CreatedAt.After(previous.CreatedAt)) {
			latest[session.DeviceID] = session
		}
	}
	result := make([]SessionView, 0, len(latest))
	for _, session := range latest {
		view := sessionView(session)
		view.ID = session.DeviceID
		result = append(result, view)
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

func (s *Store) authenticatedUserLocked(access string) (User, Session, error) {
	item, ok := s.sessionByAccessLocked(access)
	if !ok {
		return User{}, Session{}, errors.New("session_invalid")
	}
	for _, user := range s.data.Users {
		if user.ID == item.UserID && user.Status == "active" {
			return user, item, nil
		}
	}
	return User{}, Session{}, errors.New("session_invalid")
}

func (s *Store) CreateConversation(access, title string) (Conversation, error) {
	title = strings.TrimSpace(title)
	if len([]rune(title)) > 160 {
		return Conversation{}, errors.New("title_too_long")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return Conversation{}, err
	}
	id, err := randomToken(16)
	if err != nil {
		return Conversation{}, err
	}
	now := time.Now().UTC()
	if title == "" {
		title = "新对话"
	}
	item := Conversation{ID: id, UserID: user.ID, Title: title, Status: "active", CreatedAt: now, UpdatedAt: now}
	s.data.Conversations[id] = item
	s.appendAuditLocked("conversation_created", user.Email, id)
	return item, s.persistLocked()
}

func (s *Store) CreateTemporaryConversation(access, title string) (Conversation, error) {
	item, err := s.CreateConversation(access, title)
	if err != nil {
		return Conversation{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item.Temporary = true
	item.Status = "temporary"
	item.UpdatedAt = time.Now().UTC()
	s.data.Conversations[item.ID] = item
	s.appendAuditLocked("conversation_temporary_created", item.UserID, item.ID)
	return item, s.persistLocked()
}

func (s *Store) ListConversations(access string, cursor string, limit int, includeArchived bool) ([]Conversation, string, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return nil, "", err
	}
	items := make([]Conversation, 0)
	for _, item := range s.data.Conversations {
		if item.UserID != user.ID || item.DeletedAt != nil || (!includeArchived && item.ArchivedAt != nil) {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	start := 0
	if cursor != "" {
		for i, item := range items {
			if item.ID == cursor {
				start = i + 1
				break
			}
		}
	}
	if start > len(items) {
		start = len(items)
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}
	next := ""
	if end < len(items) {
		next = items[end-1].ID
	}
	return items[start:end], next, nil
}

func (s *Store) SearchConversations(access, query string, limit int) ([]Conversation, error) {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil, errors.New("query_required")
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return nil, err
	}
	matched := make([]Conversation, 0)
	for _, item := range s.data.Conversations {
		if item.UserID != user.ID || item.DeletedAt != nil {
			continue
		}
		hit := strings.Contains(strings.ToLower(item.Title), query)
		if !hit {
			for _, m := range s.data.Messages {
				if m.ConversationID == item.ID && strings.Contains(strings.ToLower(m.Body), query) {
					hit = true
					break
				}
			}
		}
		if hit {
			matched = append(matched, item)
		}
	}
	sort.Slice(matched, func(i, j int) bool { return matched[i].UpdatedAt.After(matched[j].UpdatedAt) })
	if len(matched) > limit {
		matched = matched[:limit]
	}
	return matched, nil
}

// AppendMessage persists a user or assistant message after checking that the
// caller owns the conversation. It is the storage boundary used by the chat
// runtime and makes message-body search operate on real persisted records.
func (s *Store) AppendMessage(access, conversationID, role, body string) (Message, error) {
	role = strings.TrimSpace(strings.ToLower(role))
	body = strings.TrimSpace(body)
	if role != "user" && role != "assistant" {
		return Message{}, errors.New("message_role_invalid")
	}
	if body == "" {
		return Message{}, errors.New("message_body_required")
	}
	if len([]rune(body)) > 200000 {
		return Message{}, errors.New("message_body_too_long")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return Message{}, err
	}
	conversation, ok := s.data.Conversations[conversationID]
	if !ok || conversation.UserID != user.ID || conversation.DeletedAt != nil {
		return Message{}, errors.New("conversation_not_found")
	}
	id, err := randomToken(16)
	if err != nil {
		return Message{}, err
	}
	now := time.Now().UTC()
	message := Message{ID: id, ConversationID: conversationID, UserID: user.ID, Role: role, Body: body, CreatedAt: now}
	s.data.Messages[id] = message
	conversation.UpdatedAt = now
	s.data.Conversations[conversationID] = conversation
	s.appendAuditLocked("message_persisted:"+role, user.Email, id)
	return message, s.persistLocked()
}

func (s *Store) StartRun(access, conversationID, body, model string) (MessageRun, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		model = "ylven-default"
	}
	userMessage, err := s.AppendMessage(access, conversationID, "user", body)
	if err != nil {
		return MessageRun{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return MessageRun{}, err
	}
	runID, err := randomToken(16)
	if err != nil {
		return MessageRun{}, err
	}
	now := time.Now().UTC()
	run := MessageRun{ID: runID, ConversationID: conversationID, UserID: user.ID, UserMessageID: userMessage.ID, Model: model, Provider: "upstream", Status: "streaming", StartedAt: now, CreatedAt: now, UpdatedAt: now}
	s.data.Runs[runID] = run
	s.data.RunEvents[runID] = []RunEvent{}
	s.appendAuditLocked("message_run_started", user.Email, runID)
	return run, s.persistLocked()
}

func (s *Store) CompleteRun(access, runID, assistantBody string) (MessageRun, error) {
	assistantBody = strings.TrimSpace(assistantBody)
	if assistantBody == "" {
		return MessageRun{}, errors.New("chat_provider_empty_response")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return MessageRun{}, err
	}
	run, ok := s.data.Runs[runID]
	if !ok || run.UserID != user.ID {
		return MessageRun{}, errors.New("run_not_found")
	}
	if run.Status == "cancelled" {
		return run, errors.New("run_cancelled")
	}
	if run.Status != "streaming" {
		return run, errors.New("run_not_streaming")
	}
	now := time.Now().UTC()
	assistantID, err := randomToken(16)
	if err != nil {
		return MessageRun{}, err
	}
	assistant := Message{ID: assistantID, ConversationID: run.ConversationID, UserID: user.ID, Role: "assistant", Body: assistantBody, CreatedAt: now}
	s.data.Messages[assistantID] = assistant
	run.AssistantMessageID = assistantID
	events := []RunEvent{}
	for i, runeValue := range []rune(assistantBody) {
		eventID := int64(i + 1)
		events = append(events, RunEvent{ID: eventID, RunID: runID, Type: "delta", Delta: string(runeValue), CreatedAt: now})
	}
	events = append(events, RunEvent{ID: int64(len(events) + 1), RunID: runID, Type: "completed", CreatedAt: now})
	run.Cursor = int64(len(events))
	run.Status = "completed"
	run.UpdatedAt = time.Now().UTC()
	completedAt := run.UpdatedAt
	run.CompletedAt = &completedAt
	run.LatencyMs = completedAt.Sub(run.StartedAt).Milliseconds()
	s.data.Runs[runID] = run
	s.data.RunEvents[runID] = events
	s.appendAuditLocked("message_run_completed", user.Email, runID)
	return run, s.persistLocked()
}

func (s *Store) FailRun(access, runID, code string) (MessageRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return MessageRun{}, err
	}
	run, ok := s.data.Runs[runID]
	if !ok || run.UserID != user.ID {
		return MessageRun{}, errors.New("run_not_found")
	}
	if run.Status == "cancelled" {
		return run, nil
	}
	_ = code // provider details are intentionally not persisted or exposed
	run.Status = "failed"
	run.ErrorCode = "chat_provider_error"
	run.UpdatedAt = time.Now().UTC()
	run.Cursor++
	s.data.Runs[runID] = run
	s.data.RunEvents[runID] = append(s.data.RunEvents[runID], RunEvent{ID: run.Cursor, RunID: runID, Type: "failed", CreatedAt: run.UpdatedAt})
	s.appendAuditLocked("message_run_failed", user.Email, runID)
	return run, s.persistLocked()
}

func (s *Store) CreateRun(access, conversationID, body, model, assistantBody string) (MessageRun, error) {
	run, err := s.StartRun(access, conversationID, body, model)
	if err != nil {
		return MessageRun{}, err
	}
	return s.CompleteRun(access, run.ID, assistantBody)
}

func (s *Store) Run(access, runID string) (MessageRun, []Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return MessageRun{}, nil, err
	}
	run, ok := s.data.Runs[runID]
	if !ok || run.UserID != user.ID {
		return MessageRun{}, nil, errors.New("run_not_found")
	}
	messages := []Message{}
	if m, ok := s.data.Messages[run.UserMessageID]; ok {
		messages = append(messages, m)
	}
	if m, ok := s.data.Messages[run.AssistantMessageID]; ok {
		messages = append(messages, m)
	}
	return run, messages, nil
}

func (s *Store) Events(access, runID string, after int64) (MessageRun, []RunEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return MessageRun{}, nil, err
	}
	run, ok := s.data.Runs[runID]
	if !ok || run.UserID != user.ID {
		return MessageRun{}, nil, errors.New("run_not_found")
	}
	events := []RunEvent{}
	for _, event := range s.data.RunEvents[runID] {
		if event.ID > after {
			events = append(events, event)
		}
	}
	return run, events, nil
}

func (s *Store) CancelRun(access, runID string) (MessageRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return MessageRun{}, err
	}
	run, ok := s.data.Runs[runID]
	if !ok || run.UserID != user.ID {
		return MessageRun{}, errors.New("run_not_found")
	}
	if run.Status == "streaming" {
		run.Status = "cancelled"
		run.UpdatedAt = time.Now().UTC()
		run.Cursor++
		s.data.Runs[runID] = run
		s.data.RunEvents[runID] = append(s.data.RunEvents[runID], RunEvent{ID: run.Cursor, RunID: runID, Type: "cancelled", CreatedAt: run.UpdatedAt})
		_ = s.persistLocked()
	}
	return run, nil
}

func (s *Store) SaveDraft(access, conversationID, body string) (Draft, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return Draft{}, err
	}
	c, ok := s.data.Conversations[conversationID]
	if !ok || c.UserID != user.ID || c.DeletedAt != nil {
		return Draft{}, errors.New("conversation_not_found")
	}
	draft := Draft{ConversationID: conversationID, UserID: user.ID, Body: body, UpdatedAt: time.Now().UTC()}
	s.data.Drafts[conversationID] = draft
	return draft, s.persistLocked()
}
func (s *Store) GetDraft(access, conversationID string) (Draft, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return Draft{}, err
	}
	d, ok := s.data.Drafts[conversationID]
	if !ok || d.UserID != user.ID {
		return Draft{}, errors.New("draft_not_found")
	}
	return d, nil
}
func (s *Store) ExportConversation(access, conversationID, messageID string) (ExportJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return ExportJob{}, err
	}
	c, ok := s.data.Conversations[conversationID]
	if !ok || c.UserID != user.ID || c.DeletedAt != nil {
		return ExportJob{}, errors.New("conversation_not_found")
	}
	content := "# " + c.Title + "\n\n"
	for _, m := range s.data.Messages {
		if m.ConversationID == conversationID && (messageID == "" || m.ID == messageID) {
			content += "## " + m.Role + "\n\n" + m.Body + "\n\n"
		}
	}
	id, _ := randomToken(16)
	job := ExportJob{ID: id, ConversationID: conversationID, MessageID: messageID, UserID: user.ID, Format: "markdown", Status: "ready", Content: content, CreatedAt: time.Now().UTC()}
	s.data.Exports[id] = job
	s.appendAuditLocked("conversation_export_created", user.Email, id)
	return job, s.persistLocked()
}

func (s *Store) RecordMetric(name string, value float64, errCode string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Metrics = append(s.data.Metrics, map[string]any{"name": name, "value": value, "error_code": errCode, "created_at": time.Now().UTC()})
	if len(s.data.Metrics) > 1000 {
		s.data.Metrics = s.data.Metrics[len(s.data.Metrics)-1000:]
	}
	_ = s.persistLocked()
}
func (s *Store) MetricsSnapshot() []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]map[string]any(nil), s.data.Metrics...)
}

func (s *Store) UpdateConversation(access, id, title string) (Conversation, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Conversation{}, errors.New("title_required")
	}
	if len([]rune(title)) > 160 {
		return Conversation{}, errors.New("title_too_long")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return Conversation{}, err
	}
	item, ok := s.data.Conversations[id]
	if !ok || item.UserID != user.ID || item.DeletedAt != nil {
		return Conversation{}, errors.New("conversation_not_found")
	}
	item.Title = title
	item.UpdatedAt = time.Now().UTC()
	s.data.Conversations[id] = item
	s.appendAuditLocked("conversation_renamed", user.Email, id)
	return item, s.persistLocked()
}

func (s *Store) ArchiveConversation(access, id string) (Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return Conversation{}, err
	}
	item, ok := s.data.Conversations[id]
	if !ok || item.UserID != user.ID || item.DeletedAt != nil {
		return Conversation{}, errors.New("conversation_not_found")
	}
	now := time.Now().UTC()
	item.ArchivedAt = &now
	item.Status = "archived"
	item.UpdatedAt = now
	s.data.Conversations[id] = item
	s.appendAuditLocked("conversation_archived", user.Email, id)
	return item, s.persistLocked()
}

func (s *Store) DeleteConversation(access, id string) (Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return Conversation{}, err
	}
	item, ok := s.data.Conversations[id]
	if !ok || item.UserID != user.ID || item.DeletedAt != nil {
		return Conversation{}, errors.New("conversation_not_found")
	}
	now := time.Now().UTC().Add(30 * 24 * time.Hour)
	item.DeletedAt = &now
	item.Status = "recycle_pending"
	item.UpdatedAt = time.Now().UTC()
	s.data.Conversations[id] = item
	s.appendAuditLocked("conversation_recycle_scheduled", user.Email, id)
	return item, s.persistLocked()
}

func (s *Store) Home(access string) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return nil, err
	}
	conversations := make([]Conversation, 0)
	for _, item := range s.data.Conversations {
		if item.UserID == user.ID && item.DeletedAt == nil && item.ArchivedAt == nil {
			conversations = append(conversations, item)
		}
	}
	sort.Slice(conversations, func(i, j int) bool { return conversations[i].UpdatedAt.After(conversations[j].UpdatedAt) })
	if len(conversations) > 5 {
		conversations = conversations[:5]
	}
	return map[string]any{"conversations": conversations, "projects": []any{}, "model_catalog": s.data.ModelCatalog, "config": s.data.HomeConfig}, nil
}

func (s *Store) ListAllConversations() []Conversation {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Conversation, 0, len(s.data.Conversations))
	for _, item := range s.data.Conversations {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	return out
}
func (s *Store) ConversationDetail(id string) (Conversation, []Message, []MessageRun, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.data.Conversations[id]
	if !ok {
		return Conversation{}, nil, nil, false
	}
	messages := []Message{}
	for _, m := range s.data.Messages {
		if m.ConversationID == id {
			messages = append(messages, m)
		}
	}
	runs := []MessageRun{}
	for _, run := range s.data.Runs {
		if run.ConversationID == id {
			runs = append(runs, run)
		}
	}
	sort.Slice(messages, func(i, j int) bool { return messages[i].CreatedAt.Before(messages[j].CreatedAt) })
	sort.Slice(runs, func(i, j int) bool { return runs[i].CreatedAt.Before(runs[j].CreatedAt) })
	return c, messages, runs, true
}
func (s *Store) MessageOwned(access, messageID string) (Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return Message{}, err
	}
	m, ok := s.data.Messages[messageID]
	if !ok || m.UserID != user.ID {
		return Message{}, errors.New("message_not_found")
	}
	return m, nil
}

func (s *Store) CreateSpeechJob(access, messageID string) (SpeechJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return SpeechJob{}, err
	}
	message, ok := s.data.Messages[messageID]
	if !ok || message.UserID != user.ID || message.Role != "assistant" {
		return SpeechJob{}, errors.New("message_not_found")
	}
	id, err := randomToken(16)
	if err != nil {
		return SpeechJob{}, err
	}
	if s.data.SpeechJobs == nil {
		s.data.SpeechJobs = map[string]SpeechJob{}
	}
	job := SpeechJob{ID: id, MessageID: messageID, UserID: user.ID, Status: "accepted", Provider: "android_system_tts", CreatedAt: time.Now().UTC()}
	s.data.SpeechJobs[id] = job
	s.appendAuditLocked("speech_requested", user.Email, id)
	return job, s.persistLocked()
}

func (s *Store) RunForMessage(access, messageID string) (MessageRun, Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return MessageRun{}, Message{}, err
	}
	m, ok := s.data.Messages[messageID]
	if !ok || m.UserID != user.ID {
		return MessageRun{}, Message{}, errors.New("message_not_found")
	}
	for _, run := range s.data.Runs {
		if run.AssistantMessageID == messageID || run.UserMessageID == messageID {
			return run, m, nil
		}
	}
	return MessageRun{}, m, errors.New("run_not_found")
}

func (s *Store) UpdateHomeConfig(value HomeConfig, actorID string) (HomeConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(value.Announcement) > 500 {
		return HomeConfig{}, errors.New("announcement_too_long")
	}
	value.Version = s.data.HomeConfig.Version + 1
	value.UpdatedAt = time.Now().UTC()
	s.data.HomeConfig = value
	s.appendAuditLocked("home_config_updated", "", actorID)
	return value, s.persistLocked()
}
func (s *Store) HomeConfigSnapshot() HomeConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data.HomeConfig
}
func (s *Store) RevokeSession(access, targetID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.sessionByAccessLocked(access)
	if !ok {
		return errors.New("session_invalid")
	}
	target, exists := s.data.Sessions[targetID]
	if exists && target.UserID == item.UserID {
		target.Revoked = true
		target.RefreshConsumed = true
		s.data.Sessions[targetID] = target
		s.data.Audit = append(s.data.Audit, AuditEvent{ID: targetID, Type: "session_revoked", SessionID: targetID, CreatedAt: time.Now().UTC()})
		return s.persistLocked()
	}
	found := false
	for id, candidate := range s.data.Sessions {
		if candidate.UserID == item.UserID && candidate.DeviceID == targetID {
			candidate.Revoked = true
			candidate.RefreshConsumed = true
			s.data.Sessions[id] = candidate
			found = true
		}
	}
	if !found {
		return errors.New("session_not_found")
	}
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
