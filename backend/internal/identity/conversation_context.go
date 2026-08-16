package identity

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const contextCompilerVersion = "p03-w06-v1"

type ProviderMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model               string            `json:"model"`
	Messages            []ProviderMessage `json:"messages"`
	ContinuationID      string            `json:"continuation_id,omitempty"`
	ReasoningParameters map[string]any    `json:"-"`
}

type ChatResponse struct {
	Content        string `json:"content"`
	ContinuationID string `json:"continuation_id,omitempty"`
}

type ModelCapability struct {
	ContextLimitTokens     int `json:"context_limit_tokens"`
	OutputReserveTokens    int `json:"output_reserve_tokens"`
	ReasoningReserveTokens int `json:"reasoning_reserve_tokens"`
	ToolReserveTokens      int `json:"tool_reserve_tokens"`
	SafetyMarginTokens     int `json:"safety_margin_tokens"`
}

type ContextBuildItem struct {
	Ordinal         int    `json:"ordinal"`
	ItemType        string `json:"item_type"`
	SourceID        string `json:"source_id,omitempty"`
	Role            string `json:"role,omitempty"`
	Content         string `json:"content,omitempty"`
	EstimatedTokens int    `json:"estimated_tokens"`
	Included        bool   `json:"included"`
	ExclusionReason string `json:"exclusion_reason,omitempty"`
}

type ConversationSummary struct {
	ID                   string         `json:"id"`
	ConversationID       string         `json:"conversation_id"`
	BranchID             string         `json:"branch_id"`
	SourceFirstSequence  int64          `json:"source_first_sequence"`
	SourceLastSequence   int64          `json:"source_last_sequence"`
	SourceFirstMessageID string         `json:"source_first_message_id"`
	SourceLastMessageID  string         `json:"source_last_message_id"`
	StructuredState      map[string]any `json:"structured_state"`
	SummaryText          string         `json:"summary_text"`
	SummaryModel         string         `json:"summary_model"`
	SummaryVersion       string         `json:"summary_version"`
}

type MessagePart struct {
	ID          string         `json:"id"`
	MessageID   string         `json:"message_id"`
	Ordinal     int            `json:"ordinal"`
	Kind        string         `json:"kind"`
	TextContent string         `json:"text_content,omitempty"`
	ObjectKey   string         `json:"object_key,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}

type ConversationContext struct {
	Conversation Conversation         `json:"conversation"`
	Branch       ConversationBranch   `json:"branch"`
	Messages     []Message            `json:"messages"`
	MessageParts []MessagePart        `json:"message_parts"`
	Summary      *ConversationSummary `json:"summary,omitempty"`
	Items        []ContextBuildItem   `json:"items"`
}

type ContextCompaction struct {
	ID                   string     `json:"id"`
	JobID                string     `json:"job_id"`
	ConversationID       string     `json:"conversation_id"`
	BranchID             string     `json:"branch_id"`
	IdempotencyKey       string     `json:"idempotency_key"`
	SourceFirstSequence  int64      `json:"source_first_sequence"`
	SourceLastSequence   int64      `json:"source_last_sequence"`
	SourceFirstMessageID string     `json:"source_first_message_id"`
	SourceLastMessageID  string     `json:"source_last_message_id"`
	SummaryID            string     `json:"summary_id,omitempty"`
	SummaryVersion       string     `json:"summary_version"`
	Status               string     `json:"status"`
	ErrorCode            string     `json:"error_code,omitempty"`
	Attempts             int        `json:"attempts"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
	StartedAt            *time.Time `json:"started_at,omitempty"`
	CompletedAt          *time.Time `json:"completed_at,omitempty"`
}

type ContextBuild struct {
	ID                     string               `json:"id"`
	RunID                  string               `json:"run_id"`
	ConversationID         string               `json:"conversation_id"`
	BranchID               string               `json:"branch_id"`
	Model                  string               `json:"model"`
	ContextLimitTokens     int                  `json:"context_limit_tokens"`
	InputBudgetTokens      int                  `json:"input_budget_tokens"`
	EstimatedInputTokens   int                  `json:"estimated_input_tokens"`
	OutputReserveTokens    int                  `json:"output_reserve_tokens"`
	ReasoningReserveTokens int                  `json:"reasoning_reserve_tokens"`
	ToolReserveTokens      int                  `json:"tool_reserve_tokens"`
	SafetyMarginTokens     int                  `json:"safety_margin_tokens"`
	CompactionMode         string               `json:"compaction_mode"`
	ContinuationMode       string               `json:"continuation_mode"`
	ContextHash            string               `json:"context_hash"`
	CompilerVersion        string               `json:"compiler_version"`
	Messages               []ProviderMessage    `json:"messages"`
	Items                  []ContextBuildItem   `json:"items"`
	Summary                *ConversationSummary `json:"summary,omitempty"`
}

type ContextBuildInput struct {
	BuildID            string
	RunID              string
	ConversationID     string
	BranchID           string
	Model              string
	SystemInstruction  string
	ModelInstruction   string
	UserInstruction    string
	ProjectInstruction string
	Messages           []Message
	ExistingSummary    *ConversationSummary
	Capability         ModelCapability
	ContinuationMode   string
}

type ConversationContextCompiler struct{}

func DefaultModelCapability(model string) ModelCapability {
	lower := strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.Contains(lower, "gpt-5.6"), strings.Contains(lower, "gpt-5.5"):
		return ModelCapability{ContextLimitTokens: 400000, OutputReserveTokens: 16000, ReasoningReserveTokens: 32000, ToolReserveTokens: 8000, SafetyMarginTokens: 8000}
	case strings.Contains(lower, "claude"):
		return ModelCapability{ContextLimitTokens: 200000, OutputReserveTokens: 16000, ReasoningReserveTokens: 24000, ToolReserveTokens: 8000, SafetyMarginTokens: 8000}
	case strings.Contains(lower, "grok"):
		return ModelCapability{ContextLimitTokens: 131072, OutputReserveTokens: 12000, ReasoningReserveTokens: 20000, ToolReserveTokens: 6000, SafetyMarginTokens: 6000}
	default:
		return ModelCapability{ContextLimitTokens: 128000, OutputReserveTokens: 12000, ReasoningReserveTokens: 20000, ToolReserveTokens: 6000, SafetyMarginTokens: 6000}
	}
}

func (ConversationContextCompiler) Compile(input ContextBuildInput) ContextBuild {
	capability := input.Capability
	if capability.ContextLimitTokens <= 0 {
		capability = DefaultModelCapability(input.Model)
	}
	budget := capability.ContextLimitTokens - capability.OutputReserveTokens - capability.ReasoningReserveTokens - capability.ToolReserveTokens - capability.SafetyMarginTokens
	if budget < 1024 {
		budget = 1024
	}
	build := ContextBuild{
		ID: input.BuildID, RunID: input.RunID, ConversationID: input.ConversationID, BranchID: input.BranchID, Model: input.Model,
		ContextLimitTokens: capability.ContextLimitTokens, InputBudgetTokens: budget,
		OutputReserveTokens: capability.OutputReserveTokens, ReasoningReserveTokens: capability.ReasoningReserveTokens,
		ToolReserveTokens: capability.ToolReserveTokens, SafetyMarginTokens: capability.SafetyMarginTokens,
		CompactionMode: "full_history", ContinuationMode: valueOr(input.ContinuationMode, "local_rebuild"), CompilerVersion: contextCompilerVersion,
	}

	appendInstruction := func(kind, content string) {
		content = strings.TrimSpace(content)
		if content == "" {
			return
		}
		build.Messages = append(build.Messages, ProviderMessage{Role: "system", Content: content})
		build.Items = append(build.Items, ContextBuildItem{Ordinal: len(build.Items), ItemType: kind, Role: "system", Content: content, EstimatedTokens: EstimateTokens(content), Included: true})
	}
	appendInstruction("system_instruction", input.SystemInstruction)
	appendInstruction("model_instruction", input.ModelInstruction)
	appendInstruction("user_instruction", input.UserInstruction)
	appendInstruction("project_instruction", input.ProjectInstruction)

	history := append([]Message(nil), input.Messages...)
	sort.SliceStable(history, func(i, j int) bool { return history[i].Sequence < history[j].Sequence })
	instructionTokens := 0
	for _, item := range build.Items {
		instructionTokens += item.EstimatedTokens
	}
	historyBudget := budget - instructionTokens
	if historyBudget < 512 {
		historyBudget = 512
	}
	totalHistoryTokens := 0
	for _, message := range history {
		totalHistoryTokens += EstimateTokens(message.Body)
	}

	selectedStart := 0
	if totalHistoryTokens*100 >= historyBudget*60 {
		build.CompactionMode = "summary_and_recent_turns"
		recentBudget := historyBudget * 55 / 100
		used := 0
		selectedStart = len(history)
		for selectedStart > 0 {
			turnStart := previousTurnStart(history, selectedStart)
			turnTokens := 0
			for _, message := range history[turnStart:selectedStart] {
				turnTokens += EstimateTokens(message.Body)
			}
			if used > 0 && used+turnTokens > recentBudget {
				break
			}
			selectedStart = turnStart
			used += turnTokens
		}
		if selectedStart > 0 {
			summary := input.ExistingSummary
			if summary == nil || summary.SourceLastSequence < history[selectedStart-1].Sequence {
				generated := BuildTraceableSummary(input.ConversationID, input.BranchID, history[:selectedStart])
				summary = &generated
			}
			fittedSummary := *summary
			fittedSummary.SummaryText = fitTextToTokenBudget(summary.SummaryText, historyBudget*30/100)
			summary = &fittedSummary
			build.Summary = summary
			summaryContent := "Earlier conversation summary (source messages remain authoritative):\n" + summary.SummaryText
			build.Messages = append(build.Messages, ProviderMessage{Role: "system", Content: summaryContent})
			build.Items = append(build.Items, ContextBuildItem{Ordinal: len(build.Items), ItemType: "conversation_summary", SourceID: summary.ID, Role: "system", Content: summaryContent, EstimatedTokens: EstimateTokens(summaryContent), Included: true})
		}
	}

	for index, message := range history {
		included := index >= selectedStart
		item := ContextBuildItem{Ordinal: len(build.Items), ItemType: "message", SourceID: message.ID, Role: message.Role, Content: message.Body, EstimatedTokens: EstimateTokens(message.Body), Included: included}
		if included {
			build.Messages = append(build.Messages, ProviderMessage{Role: normalizeProviderRole(message.Role), Content: message.Body})
		} else {
			item.Content = ""
			item.ExclusionReason = "replaced_by_traceable_summary"
		}
		build.Items = append(build.Items, item)
	}
	for _, message := range build.Messages {
		build.EstimatedInputTokens += EstimateTokens(message.Content)
	}
	hasher := sha256.New()
	for _, message := range build.Messages {
		_, _ = hasher.Write([]byte(message.Role))
		_, _ = hasher.Write([]byte{0})
		_, _ = hasher.Write([]byte(message.Content))
		_, _ = hasher.Write([]byte{0xff})
	}
	build.ContextHash = hex.EncodeToString(hasher.Sum(nil))
	return build
}

func previousTurnStart(messages []Message, end int) int {
	if end <= 0 {
		return 0
	}
	start := end - 1
	for start > 0 && messages[start].Role != "user" {
		start--
	}
	return start
}

var summarySignalPattern = regexp.MustCompile(`(?i)(叫|姓名|名字|金额|预算|价格|日期|时间|期限|决定|选择|方案|必须|不得|不能|约束|纠正|改为|不是|应为|\d|[$¥￥])`)

func BuildTraceableSummary(conversationID, branchID string, messages []Message) ConversationSummary {
	summary := ConversationSummary{ConversationID: conversationID, BranchID: branchID, SummaryModel: "ylven-deterministic-state-extractor", SummaryVersion: contextCompilerVersion, StructuredState: map[string]any{}}
	if len(messages) == 0 {
		return summary
	}
	summary.SourceFirstSequence = messages[0].Sequence
	summary.SourceLastSequence = messages[len(messages)-1].Sequence
	summary.SourceFirstMessageID = messages[0].ID
	summary.SourceLastMessageID = messages[len(messages)-1].ID
	lines := make([]string, 0, 20)
	for index := len(messages) - 1; index >= 0 && len(lines) < 20; index-- {
		message := messages[index]
		body := strings.TrimSpace(message.Body)
		if body == "" || (!summarySignalPattern.MatchString(body) && len(lines) >= 6) {
			continue
		}
		lines = append(lines, normalizeProviderRole(message.Role)+"["+message.ID+"]: "+truncateRunes(body, 240))
	}
	for left, right := 0, len(lines)-1; left < right; left, right = left+1, right-1 {
		lines[left], lines[right] = lines[right], lines[left]
	}
	summary.SummaryText = strings.Join(lines, "\n")
	summary.StructuredState["source_message_ids"] = []string{summary.SourceFirstMessageID, summary.SourceLastMessageID}
	summary.StructuredState["retention_policy"] = "latest_high_signal_facts_decisions_constraints"
	hash := sha256.Sum256([]byte(conversationID + "\x00" + branchID + "\x00" + summary.SummaryText))
	summary.ID = hex.EncodeToString(hash[:16])
	return summary
}

func EstimateTokens(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	tokens := 0
	latinRun := 0
	flushLatin := func() {
		if latinRun > 0 {
			tokens += (latinRun + 3) / 4
			latinRun = 0
		}
	}
	for _, valueRune := range value {
		if unicode.Is(unicode.Han, valueRune) || unicode.Is(unicode.Hiragana, valueRune) || unicode.Is(unicode.Katakana, valueRune) {
			flushLatin()
			tokens++
		} else if unicode.IsSpace(valueRune) || unicode.IsPunct(valueRune) || unicode.IsSymbol(valueRune) {
			flushLatin()
			tokens++
		} else {
			latinRun++
		}
	}
	flushLatin()
	if tokens == 0 {
		return 1
	}
	return tokens
}

func fitTextToTokenBudget(value string, budget int) string {
	if budget <= 0 || EstimateTokens(value) <= budget {
		return value
	}
	lines := strings.Split(value, "\n")
	kept := make([]string, 0, len(lines))
	used := 0
	for _, line := range lines {
		lineTokens := EstimateTokens(line)
		if used+lineTokens > budget {
			break
		}
		kept = append(kept, line)
		used += lineTokens
	}
	if len(kept) > 0 {
		return strings.Join(kept, "\n")
	}
	runes := []rune(value)
	for len(runes) > 0 && EstimateTokens(string(runes)) > budget {
		runes = runes[:len(runes)-1]
	}
	return string(runes)
}

func TemporaryConversationTitle(body string) string {
	value := strings.TrimSpace(body)
	value = strings.Trim(value, "\"'“”‘’。！？!?.,，；;：: \t\r\n")
	if value == "" {
		return "待继续"
	}
	return truncateRunes(value, 18)
}

func NormalizeFinalConversationTitle(title, fallback string) string {
	value := strings.TrimSpace(title)
	value = strings.Trim(value, "\"'“”‘’。！？!?.,，；;：: \t\r\n")
	if value == "" || value == "新对话" {
		value = TemporaryConversationTitle(fallback)
	}
	if !containsHan(value) {
		value = "关于" + TemporaryConversationTitle(fallback) + "的讨论"
	}
	if utf8.RuneCountInString(value) < 6 {
		value += "相关讨论"
		if utf8.RuneCountInString(value) < 6 {
			value = "关于" + value
		}
	}
	if utf8.RuneCountInString(value) > 18 {
		value = truncateRunes(value, 18)
	}
	return value
}

func IsLowInformationMessage(body string) bool {
	value := strings.ToLower(strings.TrimSpace(body))
	value = strings.Trim(value, "\"'“”‘’。！？!?.,，；;：: \t\r\n")
	switch value {
	case "", "你好", "您好", "嗨", "哈喽", "在吗", "测试", "试试", "hi", "hello", "test":
		return true
	default:
		return false
	}
}

func containsHan(value string) bool {
	for _, valueRune := range value {
		if unicode.Is(unicode.Han, valueRune) {
			return true
		}
	}
	return false
}

func truncateRunes(value string, maximum int) string {
	if maximum <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maximum {
		return value
	}
	return string(runes[:maximum])
}

func normalizeProviderRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "assistant", "system", "tool":
		return strings.ToLower(strings.TrimSpace(role))
	default:
		return "user"
	}
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
