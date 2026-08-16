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
	ID                      string     `json:"id"`
	UserID                  string     `json:"user_id"`
	Title                   string     `json:"title"`
	TitleSource             string     `json:"title_source"`
	TitleLocked             bool       `json:"title_locked"`
	ActiveBranchID          string     `json:"active_branch_id"`
	SummaryThroughMessageID string     `json:"summary_through_message_id,omitempty"`
	Status                  string     `json:"status"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
	ArchivedAt              *time.Time `json:"archived_at,omitempty"`
	DeletedAt               *time.Time `json:"deleted_at,omitempty"`
	Temporary               bool       `json:"temporary,omitempty"`
	DefaultModelID          string     `json:"default_model_id,omitempty"`
	DefaultReasoningProfile string     `json:"default_reasoning_profile,omitempty"`
	AISettingsVersion       int64      `json:"ai_settings_version,omitempty"`
	AISettingsOverridden    bool       `json:"ai_settings_overridden"`
}

type Message struct {
	ID                string     `json:"id"`
	ConversationID    string     `json:"conversation_id"`
	UserID            string     `json:"user_id"`
	BranchID          string     `json:"branch_id"`
	Sequence          int64      `json:"sequence"`
	ParentMessageID   string     `json:"parent_message_id,omitempty"`
	ComparisonGroupID string     `json:"comparison_group_id,omitempty"`
	Role              string     `json:"role"`
	Body              string     `json:"body"`
	Status            string     `json:"status"`
	CreatedAt         time.Time  `json:"created_at"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
}

type HomeConfig struct {
	Announcement       string               `json:"announcement"`
	FeaturedProjectIDs []string             `json:"featured_project_ids"`
	ComposerTools      []ComposerToolConfig `json:"composer_tools"`
	ConsumerCopy       map[string]string    `json:"consumer_copy"`
	Version            int64                `json:"version"`
	UpdatedAt          time.Time            `json:"updated_at"`
}

type ComposerToolConfig struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Enabled bool   `json:"enabled"`
	Prompt  string `json:"prompt"`
}

func defaultComposerTools() []ComposerToolConfig {
	return []ComposerToolConfig{
		{ID: "camera", Label: "拍照", Enabled: false, Prompt: "请分析我接下来拍摄的内容："},
		{ID: "image", Label: "选择图片", Enabled: false, Prompt: "请分析我接下来选择的图片："},
		{ID: "file", Label: "上传文件", Enabled: false, Prompt: "请分析我接下来上传的文件："},
		{ID: "image-generation", Label: "生成图片", Enabled: false, Prompt: "请帮我生成图片："},
		{ID: "presentation", Label: "制作演示", Enabled: false, Prompt: "请帮我制作演示文稿："},
		{ID: "deep-research", Label: "深度研究", Enabled: false, Prompt: "请帮我深入研究："},
	}
}

func defaultConsumerCopy() map[string]string {
	return map[string]string{
		"thinking":          "正在思考",
		"tool_file_parse":   "正在阅读文件",
		"tool_image":        "正在生成图片",
		"tool_presentation": "正在制作演示文稿",
		"reconnecting":      "连接不稳定，正在恢复…",
		"rate_limited":      "当前请求较多，请稍后再试",
		"provider_error":    "暂时无法完成回答，请重试",
		"offline":           "当前网络不可用",
		"content_blocked":   "这个请求暂时无法处理，请调整后重试",
	}
}

func defaultHomeConfig() HomeConfig {
	return HomeConfig{
		ComposerTools: defaultComposerTools(), ConsumerCopy: defaultConsumerCopy(),
		Version: 1, UpdatedAt: time.Now().UTC(),
	}
}

type MessageRun struct {
	ID                   string         `json:"id"`
	ConversationID       string         `json:"conversation_id"`
	UserID               string         `json:"user_id"`
	UserMessageID        string         `json:"user_message_id"`
	AssistantMessageID   string         `json:"assistant_message_id"`
	Model                string         `json:"model"`
	ProviderModel        string         `json:"provider_model,omitempty"`
	ReasoningProfile     string         `json:"reasoning_profile"`
	ReasoningParameters  map[string]any `json:"-"`
	BranchID             string         `json:"branch_id"`
	IdempotencyKey       string         `json:"idempotency_key,omitempty"`
	ContextBuildID       string         `json:"context_build_id,omitempty"`
	Status               string         `json:"status"`
	ErrorCode            string         `json:"error_code,omitempty"`
	Provider             string         `json:"provider,omitempty"`
	ContinuationUsed     bool           `json:"continuation_used"`
	ContinuationFallback bool           `json:"continuation_fallback"`
	StartedAt            time.Time      `json:"started_at"`
	CompletedAt          *time.Time     `json:"completed_at,omitempty"`
	LatencyMs            int64          `json:"latency_ms,omitempty"`
	Cursor               int64          `json:"cursor"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
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

type ConversationBranch struct {
	ID                  string    `json:"id"`
	ConversationID      string    `json:"conversation_id"`
	ParentBranchID      string    `json:"parent_branch_id,omitempty"`
	ForkedFromMessageID string    `json:"forked_from_message_id,omitempty"`
	Status              string    `json:"status"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type ProviderConversationState struct {
	ID              string     `json:"id"`
	ConversationID  string     `json:"conversation_id"`
	BranchID        string     `json:"branch_id"`
	Provider        string     `json:"provider"`
	Model           string     `json:"model"`
	ContinuationID  string     `json:"continuation_id"`
	Status          string     `json:"status"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	LastFailureCode string     `json:"last_failure_code,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type FirstMessageResult struct {
	Conversation Conversation `json:"conversation"`
	Run          MessageRun   `json:"run"`
	Created      bool         `json:"-"`
}

type idempotencyRecord struct {
	Operation   string    `json:"operation"`
	RequestHash string    `json:"request_hash"`
	ResourceID  string    `json:"resource_id"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// Runtime observability records are intentionally small and contain no access
// tokens, prompts beyond a bounded diagnostic preview, or provider secrets.
type ServiceInstance struct {
	InstanceID  string    `json:"instance_id"`
	ServiceName string    `json:"service_name"`
	Version     string    `json:"version,omitempty"`
	Status      string    `json:"status"`
	StartedAt   time.Time `json:"started_at"`
	LastSeenAt  time.Time `json:"last_seen_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type HealthCheck struct {
	ID         string    `json:"id"`
	InstanceID string    `json:"instance_id"`
	CheckName  string    `json:"check_name"`
	Status     string    `json:"status"`
	Detail     string    `json:"detail,omitempty"`
	CheckedAt  time.Time `json:"checked_at"`
}

type AIRuntimeHealth struct {
	Service    string            `json:"service"`
	Status     string            `json:"status"`
	InstanceID string            `json:"instance_id"`
	CheckedAt  time.Time         `json:"checked_at"`
	Instances  []ServiceInstance `json:"instances"`
	Checks     []HealthCheck     `json:"checks"`
}

type ContextDebugItem struct {
	Ordinal         int    `json:"ordinal"`
	ItemType        string `json:"item_type"`
	SourceID        string `json:"source_id,omitempty"`
	Role            string `json:"role,omitempty"`
	EstimatedTokens int    `json:"estimated_tokens"`
	Included        bool   `json:"included"`
	ExclusionReason string `json:"exclusion_reason,omitempty"`
	ContentLength   int    `json:"content_length"`
	ContentPreview  string `json:"content_preview,omitempty"`
}

type ContextDebugView struct {
	RunID                  string             `json:"run_id"`
	ConversationID         string             `json:"conversation_id"`
	BranchID               string             `json:"branch_id"`
	Status                 string             `json:"status"`
	Model                  string             `json:"model"`
	Provider               string             `json:"provider"`
	Cursor                 int64              `json:"cursor"`
	ContinuationUsed       bool               `json:"continuation_used"`
	ContinuationFallback   bool               `json:"continuation_fallback"`
	ContextBuildID         string             `json:"context_build_id"`
	ContextLimitTokens     int                `json:"context_limit_tokens"`
	InputBudgetTokens      int                `json:"input_budget_tokens"`
	EstimatedInputTokens   int                `json:"estimated_input_tokens"`
	OutputReserveTokens    int                `json:"output_reserve_tokens"`
	ReasoningReserveTokens int                `json:"reasoning_reserve_tokens"`
	ToolReserveTokens      int                `json:"tool_reserve_tokens"`
	SafetyMarginTokens     int                `json:"safety_margin_tokens"`
	CompactionMode         string             `json:"compaction_mode"`
	ContinuationMode       string             `json:"continuation_mode"`
	ContextHash            string             `json:"context_hash"`
	CompilerVersion        string             `json:"compiler_version"`
	Items                  []ContextDebugItem `json:"items"`
}

type DistributedTestRun struct {
	ID           string         `json:"id"`
	TestKey      string         `json:"test_key"`
	Status       string         `json:"status"`
	InstanceA    string         `json:"instance_a,omitempty"`
	InstanceB    string         `json:"instance_b,omitempty"`
	CursorBefore int64          `json:"cursor_before"`
	CursorAfter  int64          `json:"cursor_after"`
	ErrorCode    string         `json:"error_code,omitempty"`
	Details      map[string]any `json:"details,omitempty"`
	StartedAt    time.Time      `json:"started_at"`
	CompletedAt  *time.Time     `json:"completed_at,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// P04 records keep model choice, comparison and operational policy explicit.
type AIPreference struct {
	UserID           string    `json:"user_id"`
	ModelID          string    `json:"model_id"`
	ReasoningProfile string    `json:"reasoning_profile"`
	Version          int64     `json:"version"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type ComparisonCandidate struct {
	RunID            string `json:"run_id"`
	ModelID          string `json:"model_id"`
	ReasoningProfile string `json:"reasoning_profile"`
	Status           string `json:"status"`
	MessageID        string `json:"message_id,omitempty"`
	Body             string `json:"body,omitempty"`
	ErrorCode        string `json:"error_code,omitempty"`
}

type ComparisonGroup struct {
	ID             string                `json:"id"`
	ConversationID string                `json:"conversation_id"`
	UserID         string                `json:"user_id"`
	Prompt         string                `json:"prompt"`
	Status         string                `json:"status"`
	Candidates     []ComparisonCandidate `json:"candidates"`
	AdoptedRunID   string                `json:"adopted_run_id,omitempty"`
	SynthesisRunID string                `json:"synthesis_run_id,omitempty"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
}

type ProviderChannel struct {
	ID            string    `json:"id"`
	ProviderID    string    `json:"provider_id"`
	Name          string    `json:"name"`
	CredentialRef string    `json:"credential_reference"`
	Endpoint      string    `json:"endpoint"`
	Enabled       bool      `json:"enabled"`
	Priority      int       `json:"priority"`
	Version       int64     `json:"version"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type RoutingPolicy struct {
	ID          string   `json:"id"`
	ModelID     string   `json:"model_id"`
	Primary     string   `json:"primary_channel_id"`
	Fallbacks   []string `json:"fallback_channel_ids"`
	MaxAttempts int      `json:"max_attempts"`
	Enabled     bool     `json:"enabled"`
	Version     int64    `json:"version"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ProviderRuntimePolicy struct {
	ProviderID       string `json:"provider_id"`
	MaxConcurrency   int    `json:"max_concurrency"`
	TimeoutSeconds   int    `json:"timeout_seconds"`
	CircuitThreshold int    `json:"circuit_threshold"`
	CooldownSeconds  int    `json:"cooldown_seconds"`
	Enabled          bool   `json:"enabled"`
	Version          int64  `json:"version"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type ModelHealthStatus struct {
	ModelID      string    `json:"model_id"`
	ProviderID   string    `json:"provider_id"`
	Status       string    `json:"status"`
	LatencyMs    int64     `json:"latency_ms"`
	LastProbeAt  time.Time `json:"last_probe_at"`
	Capabilities []string  `json:"capabilities"`
	ErrorCode    string    `json:"error_code,omitempty"`
}

type UsageEvent struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	RunID           string    `json:"run_id"`
	ModelID         string    `json:"model_id"`
	InputTokens     int       `json:"input_tokens"`
	OutputTokens    int       `json:"output_tokens"`
	ReasoningTokens int       `json:"reasoning_tokens"`
	PriceVersionID  string    `json:"price_version_id"`
	CreatedAt       time.Time `json:"created_at"`
}

type PriceSnapshot struct {
	ID                  string    `json:"id"`
	ModelID             string    `json:"model_id"`
	InputPerMillion     float64   `json:"input_per_million"`
	OutputPerMillion    float64   `json:"output_per_million"`
	ReasoningPerMillion float64   `json:"reasoning_per_million"`
	EffectiveAt         time.Time `json:"effective_at"`
	Version             int64     `json:"version"`
}

// ModelCapabilityConfig is the operator-owned context budget for one catalog
// model. It is separate from ModelCapability, which is the compiler input.
type ModelCapabilityConfig struct {
	ModelID string `json:"model_id"`
	ModelCapability
	UpdatedAt time.Time `json:"updated_at"`
}

type state struct {
	Challenges             map[string]Challenge                 `json:"challenges"`
	OTPs                   map[string]OTP                       `json:"otps"`
	Users                  map[string]User                      `json:"users"`
	Sessions               map[string]Session                   `json:"sessions"`
	RateLimits             map[string][]time.Time               `json:"rate_limits"`
	Audit                  []AuditEvent                         `json:"audit"`
	Settings               map[string]map[string]string         `json:"settings"`
	Roles                  map[string]Role                      `json:"roles"`
	AdminUsers             map[string]AdminUser                 `json:"admin_users"`
	AdminSessions          map[string]AdminSession              `json:"admin_sessions"`
	StepUpChallenges       map[string]StepUpChallenge           `json:"step_up_challenges"`
	EmailTemplates         map[string]EmailTemplate             `json:"email_templates"`
	NotificationDeliveries []NotificationDelivery               `json:"notification_deliveries"`
	Workspaces             map[string]Workspace                 `json:"workspaces"`
	Conversations          map[string]Conversation              `json:"conversations"`
	ConversationBranches   map[string]ConversationBranch        `json:"conversation_branches"`
	Messages               map[string]Message                   `json:"messages"`
	MessageParts           map[string]MessagePart               `json:"message_parts"`
	ModelProviders         []ModelProvider                      `json:"model_providers"`
	ModelCatalog           []ModelCatalogEntry                  `json:"model_catalog"`
	CapabilityProbes       []CapabilityProbeResult              `json:"capability_probes"`
	ModelCatalogAudit      []ModelCatalogAuditEvent             `json:"model_catalog_audit"`
	HomeConfig             HomeConfig                           `json:"home_config"`
	Runs                   map[string]MessageRun                `json:"runs"`
	RunEvents              map[string][]RunEvent                `json:"run_events"`
	Exports                map[string]ExportJob                 `json:"exports"`
	Drafts                 map[string]Draft                     `json:"drafts"`
	Metrics                []map[string]any                     `json:"chat_metrics"`
	Feedback               map[string]MessageFeedback           `json:"message_feedback"`
	SpeechJobs             map[string]SpeechJob                 `json:"speech_jobs"`
	ContextBuilds          map[string]ContextBuild              `json:"context_builds"`
	ConversationSummaries  map[string]ConversationSummary       `json:"conversation_summaries"`
	ContextCompactions     map[string]ContextCompaction         `json:"context_compactions"`
	ProviderStates         map[string]ProviderConversationState `json:"provider_conversation_states"`
	Idempotency            map[string]idempotencyRecord         `json:"idempotency_records"`
	ServiceInstances       map[string]ServiceInstance           `json:"service_instances"`
	HealthChecks           []HealthCheck                        `json:"health_checks"`
	TestRuns               map[string]DistributedTestRun        `json:"test_runs"`
	AIPreferences          map[string]AIPreference               `json:"ai_preferences"`
	ComparisonGroups       map[string]ComparisonGroup            `json:"comparison_groups"`
	ProviderChannels       map[string]ProviderChannel            `json:"provider_channels"`
	RoutingPolicies        map[string]RoutingPolicy              `json:"routing_policies"`
	ProviderRuntimePolicies map[string]ProviderRuntimePolicy     `json:"provider_runtime_policies"`
	ModelHealth            map[string]ModelHealthStatus          `json:"model_health"`
	UsageEvents            map[string]UsageEvent                 `json:"usage_events"`
	PriceSnapshots         map[string]PriceSnapshot               `json:"price_snapshots"`
	ModelCapabilities      map[string]ModelCapabilityConfig       `json:"model_capabilities"`
}

type Store struct {
	mu              sync.Mutex
	path            string
	data            state
	conversationSQL *postgresConversationStore
	instanceID      string
}

func NewStore(path string) (*Store, error) {
	instanceSuffix, _ := randomToken(8)
	if instanceSuffix == "" {
		instanceSuffix = "local"
	}
	s := &Store{path: path, instanceID: "ai-runtime-" + instanceSuffix, data: state{Challenges: map[string]Challenge{}, OTPs: map[string]OTP{}, Users: map[string]User{}, Sessions: map[string]Session{}, RateLimits: map[string][]time.Time{}, Settings: defaultSettings(), Roles: defaultRoles(), AdminUsers: map[string]AdminUser{}, AdminSessions: map[string]AdminSession{}, StepUpChallenges: map[string]StepUpChallenge{}, EmailTemplates: defaultEmailTemplates(), NotificationDeliveries: []NotificationDelivery{}, Workspaces: map[string]Workspace{}, Conversations: map[string]Conversation{}, ConversationBranches: map[string]ConversationBranch{}, Messages: map[string]Message{}, MessageParts: map[string]MessagePart{}, Runs: map[string]MessageRun{}, RunEvents: map[string][]RunEvent{}, Exports: map[string]ExportJob{}, Drafts: map[string]Draft{}, Metrics: []map[string]any{}, Feedback: map[string]MessageFeedback{}, SpeechJobs: map[string]SpeechJob{}, ContextBuilds: map[string]ContextBuild{}, ConversationSummaries: map[string]ConversationSummary{}, ContextCompactions: map[string]ContextCompaction{}, ProviderStates: map[string]ProviderConversationState{}, Idempotency: map[string]idempotencyRecord{}, ServiceInstances: map[string]ServiceInstance{}, HealthChecks: []HealthCheck{}, TestRuns: map[string]DistributedTestRun{}, AIPreferences: map[string]AIPreference{}, ComparisonGroups: map[string]ComparisonGroup{}, ProviderChannels: map[string]ProviderChannel{}, RoutingPolicies: map[string]RoutingPolicy{}, ProviderRuntimePolicies: map[string]ProviderRuntimePolicy{}, ModelHealth: map[string]ModelHealthStatus{}, UsageEvents: map[string]UsageEvent{}, PriceSnapshots: map[string]PriceSnapshot{}, ModelCapabilities: map[string]ModelCapabilityConfig{}, ModelProviders: defaultModelProviders(), ModelCatalog: defaultModelCatalog(), CapabilityProbes: []CapabilityProbeResult{}, ModelCatalogAudit: []ModelCatalogAuditEvent{}, HomeConfig: defaultHomeConfig()}}
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
	if s.data.ConversationBranches == nil {
		s.data.ConversationBranches = map[string]ConversationBranch{}
	}
	if s.data.Messages == nil {
		s.data.Messages = map[string]Message{}
	}
	if s.data.MessageParts == nil {
		s.data.MessageParts = map[string]MessagePart{}
	}
	if len(s.data.ModelProviders) == 0 {
		s.data.ModelProviders = defaultModelProviders()
	}
	if s.data.ModelCatalog == nil {
		s.data.ModelCatalog = defaultModelCatalog()
	} else {
		for index := range s.data.ModelCatalog {
			s.data.ModelCatalog[index] = normalizeCatalogEntry(s.data.ModelCatalog[index])
		}
	}
	if s.data.CapabilityProbes == nil {
		s.data.CapabilityProbes = []CapabilityProbeResult{}
	}
	if s.data.ModelCatalogAudit == nil {
		s.data.ModelCatalogAudit = []ModelCatalogAuditEvent{}
	}
	if s.data.HomeConfig.UpdatedAt.IsZero() {
		s.data.HomeConfig = defaultHomeConfig()
	} else {
		if len(s.data.HomeConfig.ComposerTools) == 0 {
			s.data.HomeConfig.ComposerTools = defaultComposerTools()
		}
		if len(s.data.HomeConfig.ConsumerCopy) == 0 {
			s.data.HomeConfig.ConsumerCopy = defaultConsumerCopy()
		}
	}
	if s.data.ContextBuilds == nil {
		s.data.ContextBuilds = map[string]ContextBuild{}
	}
	if s.data.ConversationSummaries == nil {
		s.data.ConversationSummaries = map[string]ConversationSummary{}
	}
	if s.data.ContextCompactions == nil {
		s.data.ContextCompactions = map[string]ContextCompaction{}
	}
	if s.data.ProviderStates == nil {
		s.data.ProviderStates = map[string]ProviderConversationState{}
	}
	if s.data.Idempotency == nil {
		s.data.Idempotency = map[string]idempotencyRecord{}
	}
	if s.data.ServiceInstances == nil {
		s.data.ServiceInstances = map[string]ServiceInstance{}
	}
	if s.data.HealthChecks == nil {
		s.data.HealthChecks = []HealthCheck{}
	}
	if s.data.TestRuns == nil {
		s.data.TestRuns = map[string]DistributedTestRun{}
	}
	if s.data.AIPreferences == nil { s.data.AIPreferences = map[string]AIPreference{} }
	if s.data.ComparisonGroups == nil { s.data.ComparisonGroups = map[string]ComparisonGroup{} }
	if s.data.ProviderChannels == nil { s.data.ProviderChannels = map[string]ProviderChannel{} }
	if s.data.RoutingPolicies == nil { s.data.RoutingPolicies = map[string]RoutingPolicy{} }
	if s.data.ProviderRuntimePolicies == nil { s.data.ProviderRuntimePolicies = map[string]ProviderRuntimePolicy{} }
	if s.data.ModelHealth == nil { s.data.ModelHealth = map[string]ModelHealthStatus{} }
	if s.data.UsageEvents == nil { s.data.UsageEvents = map[string]UsageEvent{} }
	if s.data.PriceSnapshots == nil { s.data.PriceSnapshots = map[string]PriceSnapshot{} }
	if s.data.ModelCapabilities == nil { s.data.ModelCapabilities = map[string]ModelCapabilityConfig{} }
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

func (s *Store) authenticatedUser(access string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	return user, err
}

func (s *Store) CreateConversation(access, title string) (Conversation, error) {
	title = strings.TrimSpace(title)
	if len([]rune(title)) > 160 {
		return Conversation{}, errors.New("title_too_long")
	}
	if s.conversationSQL != nil {
		user, err := s.authenticatedUser(access)
		if err != nil {
			return Conversation{}, err
		}
		return s.conversationSQL.createConversation(user, title, false)
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
	titleSource := "USER"
	if title == "" {
		title = "新对话"
		titleSource = "AUTO_TEMP"
	}
	item := Conversation{ID: id, UserID: user.ID, Title: title, TitleSource: titleSource, TitleLocked: titleSource == "USER", ActiveBranchID: id, Status: "active", DefaultModelID: "ylven-default", DefaultReasoningProfile: "auto", AISettingsVersion: 1, CreatedAt: now, UpdatedAt: now}
	s.data.Conversations[id] = item
	s.data.ConversationBranches[id] = ConversationBranch{ID: id, ConversationID: id, Status: "active", CreatedAt: now, UpdatedAt: now}
	s.appendAuditLocked("conversation_created", user.Email, id)
	return item, s.persistLocked()
}

func (s *Store) CreateTemporaryConversation(access, title string) (Conversation, error) {
	if s.conversationSQL != nil {
		title = strings.TrimSpace(title)
		if len([]rune(title)) > 160 {
			return Conversation{}, errors.New("title_too_long")
		}
		user, err := s.authenticatedUser(access)
		if err != nil {
			return Conversation{}, err
		}
		return s.conversationSQL.createConversation(user, title, true)
	}
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
	if s.conversationSQL != nil {
		user, err := s.authenticatedUser(access)
		if err != nil {
			return nil, "", err
		}
		return s.conversationSQL.listConversations(user.ID, cursor, limit, includeArchived)
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
	if s.conversationSQL != nil {
		user, err := s.authenticatedUser(access)
		if err != nil {
			return nil, err
		}
		return s.conversationSQL.searchConversations(user.ID, query, limit)
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
	if s.conversationSQL != nil {
		user, err := s.authenticatedUser(access)
		if err != nil {
			return Message{}, err
		}
		return s.conversationSQL.appendMessage(user.ID, conversationID, role, body)
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
	sequence := s.nextMessageSequenceLocked(conversationID, conversation.ActiveBranchID)
	message := Message{ID: id, ConversationID: conversationID, UserID: user.ID, BranchID: conversation.ActiveBranchID, Sequence: sequence, Role: role, Body: body, Status: "completed", CreatedAt: now, CompletedAt: &now}
	s.data.Messages[id] = message
	s.data.MessageParts[id] = MessagePart{ID: id, MessageID: id, Ordinal: 0, Kind: "TEXT", TextContent: body, Metadata: map[string]any{}, CreatedAt: now}
	conversation.UpdatedAt = now
	s.data.Conversations[conversationID] = conversation
	s.appendAuditLocked("message_persisted:"+role, user.Email, id)
	return message, s.persistLocked()
}

func (s *Store) AppendCanonicalMessage(access, conversationID, role, body, idempotencyKey string) (Message, bool, error) {
	role = strings.TrimSpace(strings.ToLower(role))
	body = strings.TrimSpace(body)
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if role != "system" && role != "user" && role != "assistant" && role != "tool" {
		return Message{}, false, errors.New("message_role_invalid")
	}
	if body == "" {
		return Message{}, false, errors.New("message_body_required")
	}
	if len([]rune(body)) > 200000 {
		return Message{}, false, errors.New("message_body_too_long")
	}
	if idempotencyKey == "" || len(idempotencyKey) > 160 {
		return Message{}, false, errors.New("idempotency_key_invalid")
	}
	if s.conversationSQL != nil {
		user, err := s.authenticatedUser(access)
		if err != nil {
			return Message{}, false, err
		}
		return s.conversationSQL.appendCanonicalMessage(user.ID, conversationID, role, body, idempotencyKey)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return Message{}, false, err
	}
	conversation, ok := s.data.Conversations[conversationID]
	if !ok || conversation.UserID != user.ID || conversation.DeletedAt != nil {
		return Message{}, false, errors.New("conversation_not_found")
	}
	requestHash := conversationRequestHash(conversationID+"\x00"+body, role)
	recordKey := user.ID + ":context_message:" + idempotencyKey
	if record, exists := s.data.Idempotency[recordKey]; exists && record.ExpiresAt.After(time.Now().UTC()) {
		if record.RequestHash != requestHash {
			return Message{}, false, errors.New("idempotency_conflict")
		}
		message, found := s.data.Messages[record.ResourceID]
		if !found || message.ConversationID != conversation.ID {
			return Message{}, false, errors.New("idempotency_resource_missing")
		}
		return message, false, nil
	}
	id, err := randomToken(16)
	if err != nil {
		return Message{}, false, err
	}
	now := time.Now().UTC()
	message := Message{ID: id, ConversationID: conversation.ID, UserID: user.ID, BranchID: conversation.ActiveBranchID,
		Sequence: s.nextMessageSequenceLocked(conversation.ID, conversation.ActiveBranchID), Role: role, Body: body,
		Status: "completed", CreatedAt: now, CompletedAt: &now}
	s.data.Messages[id] = message
	s.data.MessageParts[id] = MessagePart{ID: id, MessageID: id, Ordinal: 0, Kind: "TEXT", TextContent: body, Metadata: map[string]any{}, CreatedAt: now}
	s.data.Idempotency[recordKey] = idempotencyRecord{Operation: "context_message", RequestHash: requestHash,
		ResourceID: id, CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour)}
	conversation.UpdatedAt = now
	s.data.Conversations[conversation.ID] = conversation
	s.appendAuditLocked("canonical_context_message_persisted:"+role, user.Email, id)
	return message, true, s.persistLocked()
}

func (s *Store) StartRun(access, conversationID, body, model string) (MessageRun, error) {
	key, err := randomToken(16)
	if err != nil {
		return MessageRun{}, err
	}
	return s.StartRunIdempotent(access, conversationID, body, model, key)
}

func (s *Store) StartRunIdempotent(access, conversationID, body, model, idempotencyKey string) (MessageRun, error) {
	run, _, err := s.StartRunIdempotentResult(access, conversationID, body, model, idempotencyKey)
	return run, err
}

func (s *Store) StartRunIdempotentResult(access, conversationID, body, model, idempotencyKey string) (MessageRun, bool, error) {
	selection, err := s.effectiveSelection(access, conversationID, model, "")
	if err != nil {
		return MessageRun{}, false, err
	}
	return s.startRunIdempotentResult(access, conversationID, body, idempotencyKey, selection)
}

func (s *Store) StartRunIdempotentResultWithProfile(access, conversationID, body, model, reasoningProfile, idempotencyKey string) (MessageRun, bool, error) {
	selection, err := s.effectiveSelection(access, conversationID, model, reasoningProfile)
	if err != nil {
		return MessageRun{}, false, err
	}
	return s.startRunIdempotentResult(access, conversationID, body, idempotencyKey, selection)
}

func (s *Store) startRunIdempotentResult(access, conversationID, body, idempotencyKey string, selection ModelSelection) (MessageRun, bool, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return MessageRun{}, false, errors.New("message_body_required")
	}
	if len([]rune(body)) > 200000 {
		return MessageRun{}, false, errors.New("message_body_too_long")
	}
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" || len(idempotencyKey) > 160 {
		return MessageRun{}, false, errors.New("idempotency_key_invalid")
	}
	if s.conversationSQL != nil {
		user, err := s.authenticatedUser(access)
		if err != nil {
			return MessageRun{}, false, err
		}
		return s.conversationSQL.startRun(user, conversationID, body, selection, idempotencyKey, "conversation_run")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return MessageRun{}, false, err
	}
	conversation, ok := s.data.Conversations[conversationID]
	if !ok || conversation.UserID != user.ID || conversation.DeletedAt != nil {
		return MessageRun{}, false, errors.New("conversation_not_found")
	}
	return s.startRunLocked(user, conversation, body, selection, idempotencyKey, "conversation_run")
}

func (s *Store) StartConversationFromFirstMessage(access, draftSessionID, body, model, idempotencyKey string, temporary bool) (FirstMessageResult, error) {
	selection, err := s.defaultSelection(access, model, "")
	if err != nil {
		return FirstMessageResult{}, err
	}
	return s.startConversationFromFirstMessage(access, draftSessionID, body, idempotencyKey, temporary, selection)
}

func (s *Store) StartConversationFromFirstMessageWithProfile(access, draftSessionID, body, model, reasoningProfile, idempotencyKey string, temporary bool) (FirstMessageResult, error) {
	selection, err := s.defaultSelection(access, model, reasoningProfile)
	if err != nil {
		return FirstMessageResult{}, err
	}
	return s.startConversationFromFirstMessage(access, draftSessionID, body, idempotencyKey, temporary, selection)
}

func (s *Store) startConversationFromFirstMessage(access, draftSessionID, body, idempotencyKey string, temporary bool, selection ModelSelection) (FirstMessageResult, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return FirstMessageResult{}, errors.New("message_body_required")
	}
	draftSessionID = strings.TrimSpace(draftSessionID)
	if draftSessionID == "" || len(draftSessionID) > 160 {
		return FirstMessageResult{}, errors.New("draft_session_id_invalid")
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		idempotencyKey = draftSessionID
	}
	if s.conversationSQL != nil {
		user, err := s.authenticatedUser(access)
		if err != nil {
			return FirstMessageResult{}, err
		}
		return s.conversationSQL.firstMessage(user, draftSessionID, body, selection, idempotencyKey, temporary)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return FirstMessageResult{}, err
	}
	requestHash := conversationSelectionRequestHash(body, selection)
	recordKey := user.ID + "|first_message|" + idempotencyKey
	if record, exists := s.data.Idempotency[recordKey]; exists {
		if record.RequestHash != requestHash {
			return FirstMessageResult{}, errors.New("idempotency_conflict")
		}
		run, runExists := s.data.Runs[record.ResourceID]
		conversation, conversationExists := s.data.Conversations[run.ConversationID]
		if runExists && conversationExists {
			return FirstMessageResult{Conversation: conversation, Run: run, Created: false}, nil
		}
		return FirstMessageResult{}, errors.New("idempotency_resource_missing")
	}
	conversationID, err := randomToken(16)
	if err != nil {
		return FirstMessageResult{}, err
	}
	now := time.Now().UTC()
	status := "active"
	if temporary {
		status = "temporary"
	}
	conversation := Conversation{ID: conversationID, UserID: user.ID, Title: TemporaryConversationTitle(body), TitleSource: "AUTO_TEMP", ActiveBranchID: conversationID, Status: status, DefaultModelID: "ylven-default", DefaultReasoningProfile: "auto", AISettingsVersion: 1, CreatedAt: now, UpdatedAt: now, Temporary: temporary}
	s.data.Conversations[conversationID] = conversation
	s.data.ConversationBranches[conversationID] = ConversationBranch{ID: conversationID, ConversationID: conversationID, Status: "active", CreatedAt: now, UpdatedAt: now}
	run, _, err := s.startRunLocked(user, conversation, body, selection, idempotencyKey, "first_message")
	if err != nil {
		delete(s.data.Conversations, conversationID)
		delete(s.data.ConversationBranches, conversationID)
		return FirstMessageResult{}, err
	}
	s.data.Idempotency[recordKey] = idempotencyRecord{Operation: "first_message", RequestHash: requestHash, ResourceID: run.ID, CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour)}
	s.appendAuditLocked("conversation_created_from_first_message", user.Email, conversationID)
	if err := s.persistLocked(); err != nil {
		return FirstMessageResult{}, err
	}
	return FirstMessageResult{Conversation: conversation, Run: run, Created: true}, nil
}

func (s *Store) startRunLocked(user User, conversation Conversation, body string, selection ModelSelection, idempotencyKey, operation string) (MessageRun, bool, error) {
	requestHash := conversationSelectionRequestHash(body, selection)
	recordKey := user.ID + "|" + operation + "|" + idempotencyKey
	if record, exists := s.data.Idempotency[recordKey]; exists {
		if record.RequestHash != requestHash {
			return MessageRun{}, false, errors.New("idempotency_conflict")
		}
		if run, ok := s.data.Runs[record.ResourceID]; ok {
			return run, false, nil
		}
		return MessageRun{}, false, errors.New("idempotency_resource_missing")
	}
	for _, existing := range s.data.Runs {
		if existing.ConversationID == conversation.ID && existing.BranchID == conversation.ActiveBranchID && existing.Status == "streaming" {
			return MessageRun{}, false, errors.New("conversation_busy")
		}
	}
	userMessageID, err := randomToken(16)
	if err != nil {
		return MessageRun{}, false, err
	}
	runID, err := randomToken(16)
	if err != nil {
		return MessageRun{}, false, err
	}
	now := time.Now().UTC()
	sequence := s.nextMessageSequenceLocked(conversation.ID, conversation.ActiveBranchID)
	userMessage := Message{ID: userMessageID, ConversationID: conversation.ID, UserID: user.ID, BranchID: conversation.ActiveBranchID, Sequence: sequence, Role: "user", Body: body, Status: "completed", CreatedAt: now, CompletedAt: &now}
	s.data.Messages[userMessageID] = userMessage
	s.data.MessageParts[userMessageID] = MessagePart{ID: userMessageID, MessageID: userMessageID, Ordinal: 0, Kind: "TEXT", TextContent: body, Metadata: map[string]any{}, CreatedAt: now}
	run := MessageRun{ID: runID, ConversationID: conversation.ID, UserID: user.ID, UserMessageID: userMessageID,
		Model: selection.CatalogModelID, ProviderModel: selection.ProviderModel, ReasoningProfile: selection.ReasoningProfile,
		ReasoningParameters: copyJSONMap(selection.ReasoningParameters), BranchID: conversation.ActiveBranchID,
		IdempotencyKey: idempotencyKey, Provider: "upstream", Status: "streaming", StartedAt: now, CreatedAt: now, UpdatedAt: now}
	s.data.Runs[runID] = run
	s.data.RunEvents[runID] = []RunEvent{}
	conversation.UpdatedAt = now
	s.data.Conversations[conversation.ID] = conversation
	s.data.Idempotency[recordKey] = idempotencyRecord{Operation: operation, RequestHash: requestHash, ResourceID: runID, CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour)}
	s.appendAuditLocked("message_run_started", user.Email, runID)
	return run, true, s.persistLocked()
}

func (s *Store) nextMessageSequenceLocked(conversationID, branchID string) int64 {
	var next int64 = 1
	for _, message := range s.data.Messages {
		if message.ConversationID == conversationID && message.BranchID == branchID && message.Sequence >= next {
			next = message.Sequence + 1
		}
	}
	return next
}

func conversationRequestHash(body, model string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(model) + "\x00" + strings.TrimSpace(body)))
	return hex.EncodeToString(digest[:])
}

func (s *Store) CompleteRun(access, runID, assistantBody string) (MessageRun, error) {
	assistantBody = strings.TrimSpace(assistantBody)
	if assistantBody == "" {
		return MessageRun{}, errors.New("chat_provider_empty_response")
	}
	if s.conversationSQL != nil {
		user, err := s.authenticatedUser(access)
		if err != nil {
			return MessageRun{}, err
		}
		return s.conversationSQL.completeRun(user.ID, runID, assistantBody)
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
	sequence := s.nextMessageSequenceLocked(run.ConversationID, run.BranchID)
	userMessage := s.data.Messages[run.UserMessageID]
	assistant := Message{ID: assistantID, ConversationID: run.ConversationID, UserID: user.ID, BranchID: run.BranchID, Sequence: sequence, ParentMessageID: run.UserMessageID, ComparisonGroupID: userMessage.ComparisonGroupID, Role: "assistant", Body: assistantBody, Status: "completed", CreatedAt: now, CompletedAt: &now}
	s.data.Messages[assistantID] = assistant
	s.data.MessageParts[assistantID] = MessagePart{ID: assistantID, MessageID: assistantID, Ordinal: 0, Kind: "TEXT", TextContent: assistantBody, Metadata: map[string]any{}, CreatedAt: now}
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
	s.refreshMemoryComparisonForRunLocked(run)
	s.appendAuditLocked("message_run_completed", user.Email, runID)
	return run, s.persistLocked()
}

func (s *Store) FailRun(access, runID, code string) (MessageRun, error) {
	if s.conversationSQL != nil {
		user, err := s.authenticatedUser(access)
		if err != nil {
			return MessageRun{}, err
		}
		return s.conversationSQL.finishRunWithError(user.ID, runID, "failed")
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
		return run, nil
	}
	_ = code // provider details are intentionally not persisted or exposed
	run.Status = "failed"
	run.ErrorCode = "chat_provider_error"
	run.UpdatedAt = time.Now().UTC()
	run.LatencyMs = run.UpdatedAt.Sub(run.StartedAt).Milliseconds()
	run.Cursor++
	s.data.Runs[runID] = run
	s.data.RunEvents[runID] = append(s.data.RunEvents[runID], RunEvent{ID: run.Cursor, RunID: runID, Type: "failed", CreatedAt: run.UpdatedAt})
	s.refreshMemoryComparisonForRunLocked(run)
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
	if s.conversationSQL != nil {
		user, err := s.authenticatedUser(access)
		if err != nil {
			return MessageRun{}, nil, err
		}
		return s.conversationSQL.run(user.ID, runID)
	}
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
	if s.conversationSQL != nil {
		user, err := s.authenticatedUser(access)
		if err != nil {
			return MessageRun{}, nil, err
		}
		return s.conversationSQL.events(user.ID, runID, after)
	}
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
	if s.conversationSQL != nil {
		user, err := s.authenticatedUser(access)
		if err != nil {
			return MessageRun{}, err
		}
		return s.conversationSQL.finishRunWithError(user.ID, runID, "cancelled")
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
	if s.conversationSQL != nil {
		user, err := s.authenticatedUser(access)
		if err != nil {
			return Draft{}, err
		}
		return s.conversationSQL.saveDraft(user.ID, conversationID, body)
	}
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
	if s.conversationSQL != nil {
		user, err := s.authenticatedUser(access)
		if err != nil {
			return Draft{}, err
		}
		return s.conversationSQL.getDraft(user.ID, conversationID)
	}
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
	if s.conversationSQL != nil {
		user, err := s.authenticatedUser(access)
		if err != nil {
			return ExportJob{}, err
		}
		return s.conversationSQL.exportConversation(user.ID, conversationID, messageID)
	}
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
	if s.conversationSQL != nil {
		user, err := s.authenticatedUser(access)
		if err != nil {
			return Conversation{}, err
		}
		return s.conversationSQL.updateTitle(user.ID, id, title, "USER", true)
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
	item.TitleSource = "USER"
	item.TitleLocked = true
	item.UpdatedAt = time.Now().UTC()
	s.data.Conversations[id] = item
	s.appendAuditLocked("conversation_renamed", user.Email, id)
	return item, s.persistLocked()
}

func (s *Store) ApplyAutomaticConversationTitle(access, id, title, source string) (Conversation, error) {
	if source != "AUTO_TEMP" && source != "AUTO_FINAL" {
		return Conversation{}, errors.New("title_source_invalid")
	}
	if source == "AUTO_FINAL" {
		title = NormalizeFinalConversationTitle(title, "")
	} else {
		title = TemporaryConversationTitle(title)
	}
	if title == "" {
		return Conversation{}, errors.New("title_required")
	}
	if s.conversationSQL != nil {
		user, err := s.authenticatedUser(access)
		if err != nil {
			return Conversation{}, err
		}
		return s.conversationSQL.updateTitle(user.ID, id, title, source, false)
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
	if item.TitleLocked || item.TitleSource == "USER" || (source == "AUTO_FINAL" && item.TitleSource != "AUTO_TEMP") {
		return item, nil
	}
	item.Title = title
	item.TitleSource = source
	item.UpdatedAt = time.Now().UTC()
	s.data.Conversations[id] = item
	s.appendAuditLocked("conversation_auto_title_updated", user.Email, id)
	return item, s.persistLocked()
}

func (s *Store) ConversationNeedsFinalTitle(access, id string) (bool, error) {
	if s.conversationSQL != nil {
		user, err := s.authenticatedUser(access)
		if err != nil {
			return false, err
		}
		return s.conversationSQL.needsFinalTitle(user.ID, id)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return false, err
	}
	item, ok := s.data.Conversations[id]
	if !ok || item.UserID != user.ID || item.DeletedAt != nil {
		return false, errors.New("conversation_not_found")
	}
	return !item.TitleLocked && item.TitleSource == "AUTO_TEMP", nil
}

func (s *Store) ConversationContext(access, conversationID string) (ConversationContext, error) {
	if s.conversationSQL != nil {
		user, err := s.authenticatedUser(access)
		if err != nil {
			return ConversationContext{}, err
		}
		return s.conversationSQL.conversationContext(user.ID, conversationID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return ConversationContext{}, err
	}
	conversation, ok := s.data.Conversations[conversationID]
	if !ok || conversation.UserID != user.ID || conversation.DeletedAt != nil {
		return ConversationContext{}, errors.New("conversation_not_found")
	}
	branch, ok := s.data.ConversationBranches[conversation.ActiveBranchID]
	if !ok || branch.ConversationID != conversation.ID {
		return ConversationContext{}, errors.New("conversation_branch_not_found")
	}
	snapshot := ConversationContext{Conversation: conversation, Branch: branch, Messages: []Message{}, MessageParts: []MessagePart{}, Items: []ContextBuildItem{}}
	for _, message := range s.data.Messages {
		if message.ConversationID == conversation.ID && message.BranchID == branch.ID && message.Status == "completed" {
			snapshot.Messages = append(snapshot.Messages, message)
		}
	}
	sort.SliceStable(snapshot.Messages, func(i, j int) bool { return snapshot.Messages[i].Sequence < snapshot.Messages[j].Sequence })
	sequenceByMessage := map[string]int64{}
	for _, message := range snapshot.Messages {
		sequenceByMessage[message.ID] = message.Sequence
		snapshot.Items = append(snapshot.Items, ContextBuildItem{Ordinal: len(snapshot.Items), ItemType: "message", SourceID: message.ID,
			Role: message.Role, Content: message.Body, EstimatedTokens: EstimateTokens(message.Body), Included: true})
		part, exists := s.data.MessageParts[message.ID]
		if !exists {
			part = MessagePart{ID: message.ID, MessageID: message.ID, Ordinal: 0, Kind: "TEXT", TextContent: message.Body, Metadata: map[string]any{}, CreatedAt: message.CreatedAt}
		}
		snapshot.MessageParts = append(snapshot.MessageParts, part)
	}
	for _, part := range s.data.MessageParts {
		if part.Ordinal == 0 && part.ID == part.MessageID {
			continue
		}
		if _, exists := sequenceByMessage[part.MessageID]; exists {
			snapshot.MessageParts = append(snapshot.MessageParts, part)
		}
	}
	sort.SliceStable(snapshot.MessageParts, func(i, j int) bool {
		left, right := sequenceByMessage[snapshot.MessageParts[i].MessageID], sequenceByMessage[snapshot.MessageParts[j].MessageID]
		if left == right {
			return snapshot.MessageParts[i].Ordinal < snapshot.MessageParts[j].Ordinal
		}
		return left < right
	})
	for _, summary := range s.data.ConversationSummaries {
		if summary.ConversationID == conversation.ID && summary.BranchID == branch.ID && (snapshot.Summary == nil || summary.SourceLastSequence > snapshot.Summary.SourceLastSequence) {
			copyValue := summary
			snapshot.Summary = &copyValue
		}
	}
	return snapshot, nil
}

func (s *Store) CompileRunContext(access, runID, continuationMode string) (ContextBuild, error) {
	if s.conversationSQL != nil {
		user, err := s.authenticatedUser(access)
		if err != nil {
			return ContextBuild{}, err
		}
		return s.conversationSQL.compileRunContext(user.ID, runID, continuationMode)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return ContextBuild{}, err
	}
	run, ok := s.data.Runs[runID]
	if !ok || run.UserID != user.ID {
		return ContextBuild{}, errors.New("run_not_found")
	}
	history := make([]Message, 0)
	for _, message := range s.data.Messages {
		if message.ConversationID == run.ConversationID && message.BranchID == run.BranchID && message.Status == "completed" {
			history = append(history, message)
		}
	}
	var existingSummary *ConversationSummary
	for _, summary := range s.data.ConversationSummaries {
		if summary.ConversationID == run.ConversationID && summary.BranchID == run.BranchID && (existingSummary == nil || summary.SourceLastSequence > existingSummary.SourceLastSequence) {
			copyValue := summary
			existingSummary = &copyValue
		}
	}
	buildID, err := randomToken(16)
	if err != nil {
		return ContextBuild{}, err
	}
	build := (ConversationContextCompiler{}).Compile(ContextBuildInput{
		BuildID: buildID, RunID: run.ID, ConversationID: run.ConversationID, BranchID: run.BranchID, Model: run.Model,
		SystemInstruction: "Follow YLVEN safety and product policy. Use only user-visible conversation facts; never reveal or request hidden chain-of-thought.",
		Messages:          history, ExistingSummary: existingSummary, ContinuationMode: continuationMode,
	})
	if build.Summary != nil {
		s.data.ConversationSummaries[build.Summary.ID] = *build.Summary
		conversation := s.data.Conversations[run.ConversationID]
		conversation.SummaryThroughMessageID = build.Summary.SourceLastMessageID
		s.data.Conversations[conversation.ID] = conversation
	}
	s.data.ContextBuilds[build.ID] = build
	run.ContextBuildID = build.ID
	s.data.Runs[run.ID] = run
	if err := s.persistLocked(); err != nil {
		return ContextBuild{}, err
	}
	return build, nil
}

func (s *Store) ContextBuildOwned(access, runID string) (ContextBuild, error) {
	if s.conversationSQL != nil {
		user, err := s.authenticatedUser(access)
		if err != nil {
			return ContextBuild{}, err
		}
		return s.conversationSQL.contextBuildOwned(user.ID, runID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return ContextBuild{}, err
	}
	run, ok := s.data.Runs[runID]
	if !ok || run.UserID != user.ID || run.ContextBuildID == "" {
		return ContextBuild{}, errors.New("context_build_not_found")
	}
	build, ok := s.data.ContextBuilds[run.ContextBuildID]
	if !ok {
		return ContextBuild{}, errors.New("context_build_not_found")
	}
	return build, nil
}

func (s *Store) QueueConversationCompaction(access, conversationID, idempotencyKey string) (ContextCompaction, bool, error) {
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" || len(idempotencyKey) > 160 {
		return ContextCompaction{}, false, errors.New("idempotency_key_invalid")
	}
	if s.conversationSQL != nil {
		user, err := s.authenticatedUser(access)
		if err != nil {
			return ContextCompaction{}, false, err
		}
		return s.conversationSQL.queueConversationCompaction(user.ID, conversationID, idempotencyKey)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return ContextCompaction{}, false, err
	}
	conversation, ok := s.data.Conversations[conversationID]
	if !ok || conversation.UserID != user.ID || conversation.DeletedAt != nil {
		return ContextCompaction{}, false, errors.New("conversation_not_found")
	}
	messages := []Message{}
	for _, message := range s.data.Messages {
		if message.ConversationID == conversation.ID && message.BranchID == conversation.ActiveBranchID && message.Status == "completed" {
			messages = append(messages, message)
		}
	}
	if len(messages) == 0 {
		return ContextCompaction{}, false, errors.New("conversation_context_empty")
	}
	sort.SliceStable(messages, func(i, j int) bool { return messages[i].Sequence < messages[j].Sequence })
	requestHash := conversationRequestHash(conversation.ID+"\x00"+conversation.ActiveBranchID+"\x00"+messages[len(messages)-1].ID, contextCompilerVersion)
	recordKey := user.ID + ":conversation_compaction:" + idempotencyKey
	if record, exists := s.data.Idempotency[recordKey]; exists && record.ExpiresAt.After(time.Now().UTC()) {
		if record.RequestHash != requestHash {
			return ContextCompaction{}, false, errors.New("idempotency_conflict")
		}
		item, found := s.data.ContextCompactions[record.ResourceID]
		if !found {
			return ContextCompaction{}, false, errors.New("idempotency_resource_missing")
		}
		return item, false, nil
	}
	id, err := randomToken(16)
	if err != nil {
		return ContextCompaction{}, false, err
	}
	jobID, err := randomToken(16)
	if err != nil {
		return ContextCompaction{}, false, err
	}
	now := time.Now().UTC()
	item := ContextCompaction{ID: id, JobID: jobID, ConversationID: conversation.ID, BranchID: conversation.ActiveBranchID,
		IdempotencyKey: idempotencyKey, SourceFirstSequence: messages[0].Sequence, SourceLastSequence: messages[len(messages)-1].Sequence,
		SourceFirstMessageID: messages[0].ID, SourceLastMessageID: messages[len(messages)-1].ID,
		SummaryVersion: contextCompilerVersion, Status: "queued", CreatedAt: now, UpdatedAt: now}
	s.data.ContextCompactions[id] = item
	s.data.Idempotency[recordKey] = idempotencyRecord{Operation: "conversation_compaction", RequestHash: requestHash,
		ResourceID: id, CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour)}
	s.appendAuditLocked("conversation_compaction_queued", user.Email, id)
	return item, true, s.persistLocked()
}

func (s *Store) ProcessConversationCompaction(access, compactionID string) (ContextCompaction, error) {
	if s.conversationSQL != nil {
		user, err := s.authenticatedUser(access)
		if err != nil {
			return ContextCompaction{}, err
		}
		return s.conversationSQL.processConversationCompaction(user.ID, compactionID)
	}
	s.mu.Lock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		s.mu.Unlock()
		return ContextCompaction{}, err
	}
	item, ok := s.data.ContextCompactions[compactionID]
	if !ok {
		s.mu.Unlock()
		return ContextCompaction{}, errors.New("context_compaction_not_found")
	}
	conversation, ok := s.data.Conversations[item.ConversationID]
	if !ok || conversation.UserID != user.ID {
		s.mu.Unlock()
		return ContextCompaction{}, errors.New("context_compaction_not_found")
	}
	if item.Status != "queued" {
		s.mu.Unlock()
		return item, nil
	}
	now := time.Now().UTC()
	item.Status = "running"
	item.Attempts++
	item.StartedAt = &now
	item.UpdatedAt = now
	s.data.ContextCompactions[item.ID] = item
	messages := []Message{}
	for _, message := range s.data.Messages {
		if message.ConversationID == item.ConversationID && message.BranchID == item.BranchID && message.Status == "completed" && message.Sequence >= item.SourceFirstSequence && message.Sequence <= item.SourceLastSequence {
			messages = append(messages, message)
		}
	}
	s.mu.Unlock()
	sort.SliceStable(messages, func(i, j int) bool { return messages[i].Sequence < messages[j].Sequence })
	if len(messages) == 0 || messages[0].ID != item.SourceFirstMessageID || messages[len(messages)-1].ID != item.SourceLastMessageID {
		s.mu.Lock()
		failedAt := time.Now().UTC()
		item.Status = "error"
		item.ErrorCode = "compaction_source_changed"
		item.CompletedAt = &failedAt
		item.UpdatedAt = failedAt
		s.data.ContextCompactions[item.ID] = item
		_ = s.persistLocked()
		s.mu.Unlock()
		return item, errors.New("compaction source range is incomplete")
	}
	summary := BuildTraceableSummary(item.ConversationID, item.BranchID, messages)
	s.mu.Lock()
	defer s.mu.Unlock()
	completedAt := time.Now().UTC()
	item.Status = "success"
	item.SummaryID = summary.ID
	item.ErrorCode = ""
	item.CompletedAt = &completedAt
	item.UpdatedAt = completedAt
	s.data.ContextCompactions[item.ID] = item
	s.data.ConversationSummaries[summary.ID] = summary
	conversation = s.data.Conversations[item.ConversationID]
	conversation.SummaryThroughMessageID = summary.SourceLastMessageID
	s.data.Conversations[conversation.ID] = conversation
	s.appendAuditLocked("conversation_compaction_completed", user.Email, item.ID)
	return item, s.persistLocked()
}

func (s *Store) ContextCompactionOwned(access, compactionID string) (ContextCompaction, error) {
	if s.conversationSQL != nil {
		user, err := s.authenticatedUser(access)
		if err != nil {
			return ContextCompaction{}, err
		}
		return s.conversationSQL.contextCompactionOwned(user.ID, compactionID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return ContextCompaction{}, err
	}
	item, ok := s.data.ContextCompactions[compactionID]
	conversation, found := s.data.Conversations[item.ConversationID]
	if !ok || !found || conversation.UserID != user.ID {
		return ContextCompaction{}, errors.New("context_compaction_not_found")
	}
	return item, nil
}

func (s *Store) ProviderState(access, conversationID, branchID, provider, model string) (ProviderConversationState, bool, error) {
	if s.conversationSQL != nil {
		user, err := s.authenticatedUser(access)
		if err != nil {
			return ProviderConversationState{}, false, err
		}
		return s.conversationSQL.providerState(user.ID, conversationID, branchID, provider, model)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return ProviderConversationState{}, false, err
	}
	conversation, ok := s.data.Conversations[conversationID]
	if !ok || conversation.UserID != user.ID {
		return ProviderConversationState{}, false, errors.New("conversation_not_found")
	}
	state, exists := s.data.ProviderStates[providerStateKey(conversationID, branchID, provider, model)]
	if !exists || state.Status != "active" || (state.ExpiresAt != nil && state.ExpiresAt.Before(time.Now().UTC())) {
		return ProviderConversationState{}, false, nil
	}
	return state, true, nil
}

func (s *Store) SaveProviderState(access string, state ProviderConversationState) error {
	if s.conversationSQL != nil {
		user, err := s.authenticatedUser(access)
		if err != nil {
			return err
		}
		return s.conversationSQL.saveProviderState(user.ID, state)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return err
	}
	conversation, ok := s.data.Conversations[state.ConversationID]
	if !ok || conversation.UserID != user.ID {
		return errors.New("conversation_not_found")
	}
	now := time.Now().UTC()
	if state.ID == "" {
		state.ID, err = randomToken(16)
		if err != nil {
			return err
		}
		state.CreatedAt = now
	}
	state.UpdatedAt = now
	if state.Status == "" {
		state.Status = "active"
	}
	s.data.ProviderStates[providerStateKey(state.ConversationID, state.BranchID, state.Provider, state.Model)] = state
	return s.persistLocked()
}

func (s *Store) MarkProviderContinuationFallback(access, runID, failureCode string) error {
	if s.conversationSQL != nil {
		user, err := s.authenticatedUser(access)
		if err != nil {
			return err
		}
		return s.conversationSQL.markContinuationFallback(user.ID, runID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return err
	}
	run, ok := s.data.Runs[runID]
	if !ok || run.UserID != user.ID {
		return errors.New("run_not_found")
	}
	run.ContinuationUsed = true
	run.ContinuationFallback = true
	s.data.Runs[runID] = run
	for key, state := range s.data.ProviderStates {
		if state.ConversationID == run.ConversationID && state.BranchID == run.BranchID && state.Model == run.Model {
			state.Status = "failed"
			state.LastFailureCode = stableProviderFailureCode(failureCode)
			state.UpdatedAt = time.Now().UTC()
			s.data.ProviderStates[key] = state
		}
	}
	return s.persistLocked()
}

func providerStateKey(conversationID, branchID, provider, model string) string {
	return strings.Join([]string{conversationID, branchID, strings.ToLower(provider), strings.ToLower(model)}, "|")
}

func stableProviderFailureCode(_ string) string { return "continuation_unavailable" }

func (s *Store) ArchiveConversation(access, id string) (Conversation, error) {
	if s.conversationSQL != nil {
		user, err := s.authenticatedUser(access)
		if err != nil {
			return Conversation{}, err
		}
		return s.conversationSQL.setConversationStatus(user.ID, id, "archive")
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
	now := time.Now().UTC()
	item.ArchivedAt = &now
	item.Status = "archived"
	item.UpdatedAt = now
	s.data.Conversations[id] = item
	s.appendAuditLocked("conversation_archived", user.Email, id)
	return item, s.persistLocked()
}

func (s *Store) DeleteConversation(access, id string) (Conversation, error) {
	if s.conversationSQL != nil {
		user, err := s.authenticatedUser(access)
		if err != nil {
			return Conversation{}, err
		}
		return s.conversationSQL.setConversationStatus(user.ID, id, "delete")
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
	now := time.Now().UTC().Add(30 * 24 * time.Hour)
	item.DeletedAt = &now
	item.Status = "recycle_pending"
	item.UpdatedAt = time.Now().UTC()
	s.data.Conversations[id] = item
	s.appendAuditLocked("conversation_recycle_scheduled", user.Email, id)
	return item, s.persistLocked()
}

func (s *Store) Home(access string) (map[string]any, error) {
	if s.conversationSQL != nil {
		items, _, err := s.ListConversations(access, "", 5, false)
		if err != nil {
			return nil, err
		}
		catalog, err := s.conversationSQL.modelCatalog()
		if err != nil {
			return nil, err
		}
		config, err := s.conversationSQL.homeConfig()
		if err != nil {
			return nil, err
		}
		return map[string]any{"conversations": items, "projects": []any{}, "model_catalog": catalog, "config": config}, nil
	}
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

func (s *Store) MobileModelCatalog(access string) ([]ModelCatalogEntry, error) {
	if _, err := s.authenticatedUser(access); err != nil {
		return nil, err
	}
	if s.conversationSQL != nil {
		return s.conversationSQL.modelCatalog()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	providers := map[string]ModelProvider{}
	for _, provider := range s.data.ModelProviders {
		providers[provider.ID] = provider
	}
	latestProbe := map[string]CapabilityProbeResult{}
	now := time.Now().UTC()
	for _, probe := range s.data.CapabilityProbes {
		key := probe.ModelID + "\x00" + probe.CapabilityID
		current, exists := latestProbe[key]
		if !exists || probe.ProbedAt.After(current.ProbedAt) || (probe.ProbedAt.Equal(current.ProbedAt) && probe.CreatedAt.After(current.CreatedAt)) {
			latestProbe[key] = probe
		}
	}
	items := make([]ModelCatalogEntry, 0, len(s.data.ModelCatalog))
	for _, raw := range s.data.ModelCatalog {
		copyItem := normalizeCatalogEntry(raw)
		provider, providerExists := providers[copyItem.ProviderID]
		copyItem.Enabled = copyItem.Enabled && providerExists && provider.Enabled
		copyItem.ProviderName = provider.Name
		copyItem.ReasoningProfiles = append([]string(nil), copyItem.ReasoningProfiles...)
		copyItem.ReasoningMappings = nil
		copyItem.Capabilities = []ModelCapabilityTag{}
		for key, probe := range latestProbe {
			if !strings.HasPrefix(key, copyItem.ID+"\x00") || probe.Status != "passed" || (probe.ExpiresAt != nil && !probe.ExpiresAt.After(now)) {
				continue
			}
			copyItem.Capabilities = append(copyItem.Capabilities, ModelCapabilityTag{ID: probe.CapabilityID, Label: valueOr(probe.CapabilityLabel, probe.CapabilityID), ProbedAt: probe.ProbedAt})
		}
		sort.Slice(copyItem.Capabilities, func(i, j int) bool { return copyItem.Capabilities[i].ID < copyItem.Capabilities[j].ID })
		items = append(items, copyItem)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].ProviderID == items[j].ProviderID && items[i].SortOrder == items[j].SortOrder {
			return false
		}
		if items[i].ProviderID == items[j].ProviderID {
			return items[i].SortOrder < items[j].SortOrder
		}
		return providers[items[i].ProviderID].SortOrder < providers[items[j].ProviderID].SortOrder
	})
	return items, nil
}

func conversationSelectionRequestHash(body string, selection ModelSelection) string {
	if valueOr(selection.ReasoningProfile, "auto") == "auto" && len(selection.ReasoningParameters) == 0 &&
		(selection.ProviderModel == "" || selection.ProviderModel == selection.CatalogModelID) {
		return conversationRequestHash(body, selection.CatalogModelID)
	}
	parameters, _ := json.Marshal(selection.ReasoningParameters)
	return conversationRequestHash(body, selection.CatalogModelID+"\x00"+selection.ProviderModel+"\x00"+selection.ReasoningProfile+"\x00"+string(parameters))
}

func (s *Store) ListAllConversations() []Conversation {
	if s.conversationSQL != nil {
		items, err := s.conversationSQL.listAllConversations()
		if err != nil {
			return []Conversation{}
		}
		return items
	}
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
	if s.conversationSQL != nil {
		return s.conversationSQL.conversationDetail(id)
	}
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

func (s *Store) ConversationMessages(access, id string) ([]Message, error) {
	if s.conversationSQL != nil {
		user, err := s.authenticatedUser(access)
		if err != nil {
			return nil, err
		}
		return s.conversationSQL.conversationMessages(user.ID, id)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return nil, err
	}
	conversation, ok := s.data.Conversations[id]
	if !ok || conversation.UserID != user.ID || conversation.DeletedAt != nil {
		return nil, errors.New("conversation_not_found")
	}
	messages := make([]Message, 0)
	for _, message := range s.data.Messages {
		if message.ConversationID == id && message.BranchID == conversation.ActiveBranchID {
			messages = append(messages, message)
		}
	}
	sort.Slice(messages, func(i, j int) bool { return messages[i].Sequence < messages[j].Sequence })
	return messages, nil
}
func (s *Store) MessageOwned(access, messageID string) (Message, error) {
	if s.conversationSQL != nil {
		user, err := s.authenticatedUser(access)
		if err != nil {
			return Message{}, err
		}
		return s.conversationSQL.messageOwned(user.ID, messageID)
	}
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
	if s.conversationSQL != nil {
		user, err := s.authenticatedUser(access)
		if err != nil {
			return SpeechJob{}, err
		}
		return s.conversationSQL.createSpeechJob(user.ID, messageID)
	}
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
	if s.conversationSQL != nil {
		user, err := s.authenticatedUser(access)
		if err != nil {
			return MessageRun{}, Message{}, err
		}
		return s.conversationSQL.runForMessage(user.ID, messageID)
	}
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

func (s *Store) SaveMessageFeedback(access, messageID, value string) error {
	if value != "up" && value != "down" {
		return errors.New("feedback_invalid")
	}
	if s.conversationSQL != nil {
		user, err := s.authenticatedUser(access)
		if err != nil {
			return err
		}
		return s.conversationSQL.saveFeedback(user.ID, messageID, value)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return err
	}
	message, ok := s.data.Messages[messageID]
	if !ok || message.UserID != user.ID {
		return errors.New("message_not_found")
	}
	s.data.Feedback[messageID] = MessageFeedback{MessageID: messageID, UserID: user.ID, Value: value, CreatedAt: time.Now().UTC()}
	s.appendAuditLocked("message_feedback", user.Email, messageID)
	return s.persistLocked()
}

func (s *Store) UpdateHomeConfig(value HomeConfig, actorID string) (HomeConfig, error) {
	var err error
	value, err = normalizeHomeConfig(value)
	if err != nil {
		return HomeConfig{}, err
	}
	if s.conversationSQL != nil {
		value, err = s.conversationSQL.updateHomeConfig(value, actorID)
		if err != nil {
			return HomeConfig{}, err
		}
		s.mu.Lock()
		s.data.HomeConfig = value
		s.mu.Unlock()
		return value, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	value.Version = s.data.HomeConfig.Version + 1
	value.UpdatedAt = time.Now().UTC()
	s.data.HomeConfig = value
	s.appendAuditLocked("home_config_updated", "", actorID)
	return value, s.persistLocked()
}

func normalizeHomeConfig(value HomeConfig) (HomeConfig, error) {
	if len(value.Announcement) > 500 {
		return HomeConfig{}, errors.New("announcement_too_long")
	}
	if len(value.ComposerTools) == 0 {
		value.ComposerTools = defaultComposerTools()
	}
	allowedTools := map[string]bool{"camera": true, "image": true, "file": true, "image-generation": true, "presentation": true, "deep-research": true}
	seenTools := map[string]bool{}
	for _, tool := range value.ComposerTools {
		if !allowedTools[tool.ID] || seenTools[tool.ID] || strings.TrimSpace(tool.Label) == "" || len(tool.Prompt) > 200 {
			return HomeConfig{}, errors.New("composer_tools_invalid")
		}
		seenTools[tool.ID] = true
	}
	defaults := defaultConsumerCopy()
	if len(value.ConsumerCopy) == 0 {
		value.ConsumerCopy = defaults
	}
	if len(value.ConsumerCopy) != len(defaults) {
		return HomeConfig{}, errors.New("consumer_copy_invalid")
	}
	for key := range value.ConsumerCopy {
		if _, ok := defaults[key]; !ok {
			return HomeConfig{}, errors.New("consumer_copy_invalid")
		}
	}
	text := strings.ToLower(strings.Join(mapValues(value.ConsumerCopy), " "))
	for _, forbidden := range []string{"streaming", "connecting", "provider error", "request id", "sub2api", "sse", "websocket", "debug", "demo", "mock"} {
		if strings.Contains(text, forbidden) {
			return HomeConfig{}, errors.New("consumer_copy_invalid")
		}
	}
	for _, copy := range value.ConsumerCopy {
		if strings.TrimSpace(copy) == "" || len(copy) > 80 {
			return HomeConfig{}, errors.New("consumer_copy_invalid")
		}
	}
	return value, nil
}

func mapValues(values map[string]string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	return result
}
func (s *Store) HomeConfigSnapshot() (HomeConfig, error) {
	if s.conversationSQL != nil {
		return s.conversationSQL.homeConfig()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data.HomeConfig, nil
}

func (s *Store) HomeConfigAuditSnapshot() ([]AuditEvent, error) {
	if s.conversationSQL != nil {
		return s.conversationSQL.homeConfigAudit()
	}
	return s.AuditSnapshot(), nil
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
