package identity

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"
)

func TestP03W06FirstMessageIsAtomicIdempotentAndTitleLocked(t *testing.T) {
	store, _ := NewStore("")
	createTestUser(t, store, "first-message@example.com")
	access := createAuthenticatedTestSession(t, store, "first-message@example.com")

	if _, err := store.StartConversationFromFirstMessage(access, "draft-empty", "  ", "", "same-key", false); err == nil || err.Error() != "message_body_required" {
		t.Fatalf("empty first message err=%v", err)
	}
	items, _, err := store.ListConversations(access, "", 20, false)
	if err != nil || len(items) != 0 {
		t.Fatalf("empty draft persisted a conversation: %+v err=%v", items, err)
	}

	first, err := store.StartConversationFromFirstMessage(access, "draft-1", "规划八月产品发布", "gpt-5.6", "first-key", false)
	if err != nil || !first.Created {
		t.Fatalf("first result=%+v err=%v", first, err)
	}
	duplicate, err := store.StartConversationFromFirstMessage(access, "draft-1", "规划八月产品发布", "gpt-5.6", "first-key", false)
	if err != nil || duplicate.Created || duplicate.Conversation.ID != first.Conversation.ID || duplicate.Run.ID != first.Run.ID {
		t.Fatalf("duplicate result=%+v err=%v", duplicate, err)
	}
	if _, err := store.StartConversationFromFirstMessage(access, "draft-1", "不同正文", "gpt-5.6", "first-key", false); err == nil || err.Error() != "idempotency_conflict" {
		t.Fatalf("changed duplicate err=%v", err)
	}

	if _, err := store.CompleteRun(access, first.Run.ID, "已整理发布步骤"); err != nil {
		t.Fatal(err)
	}
	renamed, err := store.UpdateConversation(access, first.Conversation.ID, "用户锁定标题")
	if err != nil || !renamed.TitleLocked || renamed.TitleSource != "USER" {
		t.Fatalf("renamed=%+v err=%v", renamed, err)
	}
	automatic, err := store.ApplyAutomaticConversationTitle(access, first.Conversation.ID, "自动标题不应覆盖", "AUTO_FINAL")
	if err != nil || automatic.Title != "用户锁定标题" || !automatic.TitleLocked {
		t.Fatalf("automatic overwrite=%+v err=%v", automatic, err)
	}
}

type titleRuleResponder struct {
	mu         sync.Mutex
	chatCalls  int
	titleCalls int
}

func (r *titleRuleResponder) Respond(_ context.Context, request ChatRequest) (ChatResponse, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(request.Messages) > 0 && strings.Contains(request.Messages[0].Content, "Return a title only") {
		r.titleCalls++
		return ChatResponse{Content: "八月发布计划"}, nil
	}
	r.chatCalls++
	return ChatResponse{Content: "已收到，我会继续结合前文回答。"}, nil
}

func (r *titleRuleResponder) counts() (int, int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.chatCalls, r.titleCalls
}

func TestP03W06LowInformationTitleWaitsThenFinalizesOnce(t *testing.T) {
	store, _ := NewStore("")
	createTestUser(t, store, "title-policy@example.com")
	access := createAuthenticatedTestSession(t, store, "title-policy@example.com")
	responder := &titleRuleResponder{}
	api := NewAPI(store)
	api.ChatRuntimeMode = "upstream"
	api.ChatResponder = responder

	firstResponse := requestJSON(t, api.Handler(), http.MethodPost, "/api/mobile/v1/conversations/from-first-message", map[string]any{
		"draft_session_id": "title-draft", "body": "你好", "model": "gpt-5.6", "idempotency_key": "title-first",
	}, access, "")
	if firstResponse.Code != http.StatusAccepted {
		t.Fatalf("first response=%d %s", firstResponse.Code, firstResponse.Body.String())
	}
	var first FirstMessageResult
	if err := json.Unmarshal(firstResponse.Body.Bytes(), &first); err != nil {
		t.Fatal(err)
	}
	waitForRunStatus(t, store, access, first.Run.ID, "completed")
	items, _, err := store.ListConversations(access, "", 10, false)
	if err != nil || len(items) != 1 || items[0].Title != "你好" || items[0].TitleSource != "AUTO_TEMP" {
		t.Fatalf("low-information title=%+v err=%v", items, err)
	}
	if chatCalls, titleCalls := responder.counts(); chatCalls != 1 || titleCalls != 0 {
		t.Fatalf("low-information calls chat=%d title=%d", chatCalls, titleCalls)
	}

	secondResponse := requestJSON(t, api.Handler(), http.MethodPost, "/api/mobile/v1/conversations/"+first.Conversation.ID+"/runs", map[string]any{
		"body": "请制定八月产品发布计划", "model": "gpt-5.6", "idempotency_key": "title-second",
	}, access, "")
	if secondResponse.Code != http.StatusAccepted {
		t.Fatalf("second response=%d %s", secondResponse.Code, secondResponse.Body.String())
	}
	var second MessageRun
	if err := json.Unmarshal(secondResponse.Body.Bytes(), &second); err != nil {
		t.Fatal(err)
	}
	waitForRunStatus(t, store, access, second.ID, "completed")
	var finalized Conversation
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
		items, _, _ = store.ListConversations(access, "", 10, false)
		if len(items) == 1 && items[0].TitleSource == "AUTO_FINAL" {
			finalized = items[0]
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if finalized.TitleSource != "AUTO_FINAL" || utf8.RuneCountInString(finalized.Title) < 6 || utf8.RuneCountInString(finalized.Title) > 18 {
		t.Fatalf("finalized title=%+v", finalized)
	}

	thirdResponse := requestJSON(t, api.Handler(), http.MethodPost, "/api/mobile/v1/conversations/"+first.Conversation.ID+"/runs", map[string]any{
		"body": "继续补充风险", "model": "gpt-5.6", "idempotency_key": "title-third",
	}, access, "")
	var third MessageRun
	if err := json.Unmarshal(thirdResponse.Body.Bytes(), &third); err != nil {
		t.Fatal(err)
	}
	waitForRunStatus(t, store, access, third.ID, "completed")
	time.Sleep(20 * time.Millisecond)
	items, _, _ = store.ListConversations(access, "", 10, false)
	if chatCalls, titleCalls := responder.counts(); chatCalls != 3 || titleCalls != 1 || items[0].Title != finalized.Title {
		t.Fatalf("final title changed=%+v chat=%d title=%d", items[0], chatCalls, titleCalls)
	}

	for _, candidate := range []struct {
		provider string
		fallback string
	}{
		{"Plan", "制定八月产品发布计划"},
		{"计划", "短问"},
	} {
		title := NormalizeFinalConversationTitle(candidate.provider, candidate.fallback)
		if utf8.RuneCountInString(title) < 6 || utf8.RuneCountInString(title) > 18 || !containsHan(title) {
			t.Fatalf("invalid normalized title %q", title)
		}
	}
}

func waitForRunStatus(t *testing.T, store *Store, access, runID, expected string) {
	t.Helper()
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
		run, _, err := store.Run(access, runID)
		if err != nil {
			t.Fatal(err)
		}
		if run.Status == expected {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	run, _, _ := store.Run(access, runID)
	t.Fatalf("run %s status=%s, expected %s", runID, run.Status, expected)
}

func TestP03W06CanonicalHistorySupportsReferencesCorrectionsAndModelSwitch(t *testing.T) {
	store, _ := NewStore("")
	createTestUser(t, store, "context@example.com")
	access := createAuthenticatedTestSession(t, store, "context@example.com")
	conversation, err := store.CreateConversation(access, "上下文")
	if err != nil {
		t.Fatal(err)
	}
	turns := []struct {
		user      string
		assistant string
		model     string
	}{
		{"我叫陈平", "好的，我会在本次会话中记住你叫陈平。", "gpt-5.6"},
		{"给我两种发布方案：第一种周一，第二种周三", "第一种周一灰度；第二种周三全量。", "gpt-5.6"},
		{"我纠正一下，第二种方案改为周四", "已更新：第二种方案是周四全量。", "claude-opus"},
	}
	for index, turn := range turns {
		if _, err := store.CreateRun(access, conversation.ID, turn.user, turn.model, turn.assistant); err != nil {
			t.Fatalf("turn %d: %v", index, err)
		}
	}
	run, err := store.StartRunIdempotent(access, conversation.ID, "我叫什么名字？第二种方案是哪天？", "grok-4", "model-switch")
	if err != nil {
		t.Fatal(err)
	}
	build, err := store.CompileRunContext(access, run.ID, "model_switch_recompile")
	if err != nil {
		t.Fatal(err)
	}
	joined := providerMessagesText(build.Messages)
	for _, expected := range []string{"我叫陈平", "第二种周三", "第二种方案改为周四", "我叫什么名字"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("compiled context omitted %q: %s", expected, joined)
		}
	}
	if build.Model != "grok-4" || build.ContinuationMode != "model_switch_recompile" {
		t.Fatalf("neutral recompilation metadata=%+v", build)
	}
	_, messages, runs, ok := store.ConversationDetail(conversation.ID)
	if !ok || len(messages) != 7 || len(runs) != 4 {
		t.Fatalf("canonical history messages=%d runs=%d ok=%v", len(messages), len(runs), ok)
	}
	for index, message := range messages {
		if message.Sequence != int64(index+1) || message.BranchID != conversation.ActiveBranchID {
			t.Fatalf("message ordering at %d: %+v", index, message)
		}
	}
}

func TestP03W06CompactionRetainsAmountDateDecisionAndConstraint(t *testing.T) {
	messages := []Message{
		{ID: "m1", ConversationID: "c1", BranchID: "b1", Sequence: 1, Role: "user", Body: "预算金额是125000元，交付日期是2026年9月18日，决定采用第二种方案，约束是不得停机。", Status: "completed"},
		{ID: "m2", ConversationID: "c1", BranchID: "b1", Sequence: 2, Role: "assistant", Body: "已确认金额、日期、决定和约束。", Status: "completed"},
	}
	for index := 3; index <= 42; index++ {
		role := "user"
		if index%2 == 0 {
			role = "assistant"
		}
		messages = append(messages, Message{ID: "m" + strings.Repeat("x", index), ConversationID: "c1", BranchID: "b1", Sequence: int64(index), Role: role, Body: strings.Repeat("补充讨论内容", 20), Status: "completed"})
	}
	build := (ConversationContextCompiler{}).Compile(ContextBuildInput{
		BuildID: "build", RunID: "run", ConversationID: "c1", BranchID: "b1", Model: "small-model",
		Messages:   messages,
		Capability: ModelCapability{ContextLimitTokens: 1400, OutputReserveTokens: 200, ReasoningReserveTokens: 100, ToolReserveTokens: 50, SafetyMarginTokens: 50},
	})
	if build.Summary == nil || build.CompactionMode != "summary_and_recent_turns" {
		t.Fatalf("compaction did not run: %+v", build)
	}
	for _, expected := range []string{"125000", "2026年9月18日", "第二种方案", "不得停机"} {
		if !strings.Contains(build.Summary.SummaryText, expected) {
			t.Fatalf("summary omitted %q: %s", expected, build.Summary.SummaryText)
		}
	}
	if build.EstimatedInputTokens > build.InputBudgetTokens {
		t.Fatalf("compiled input exceeded budget: %d > %d", build.EstimatedInputTokens, build.InputBudgetTokens)
	}
}

func TestP03W06ConversationConcurrencyAndReconnectIdempotency(t *testing.T) {
	store, _ := NewStore("")
	createTestUser(t, store, "ordering@example.com")
	access := createAuthenticatedTestSession(t, store, "ordering@example.com")
	conversation, _ := store.CreateConversation(access, "顺序")
	first, created, err := store.StartRunIdempotentResult(access, conversation.ID, "第一条", "gpt-5.6", "same-run")
	if err != nil || !created {
		t.Fatalf("first=%+v created=%v err=%v", first, created, err)
	}
	duplicate, created, err := store.StartRunIdempotentResult(access, conversation.ID, "第一条", "gpt-5.6", "same-run")
	if err != nil || created || duplicate.ID != first.ID {
		t.Fatalf("duplicate=%+v created=%v err=%v", duplicate, created, err)
	}
	if _, _, err := store.StartRunIdempotentResult(access, conversation.ID, "并发第二条", "gpt-5.6", "other-run"); err == nil || err.Error() != "conversation_busy" {
		t.Fatalf("parallel run err=%v", err)
	}
	if _, err := store.CompleteRun(access, first.ID, "第一条回答"); err != nil {
		t.Fatal(err)
	}
	second, created, err := store.StartRunIdempotentResult(access, conversation.ID, "第二条", "gpt-5.6", "second-run")
	if err != nil || !created {
		t.Fatalf("second=%+v created=%v err=%v", second, created, err)
	}
	_, messages, _, _ := store.ConversationDetail(conversation.ID)
	for index, message := range messages {
		if message.Sequence != int64(index+1) {
			t.Fatalf("non-monotonic messages=%+v", messages)
		}
	}
}

type fallbackResponder struct {
	mu       sync.Mutex
	requests []ChatRequest
}

func (r *fallbackResponder) Continue(context.Context, ChatRequest) (ChatResponse, error) {
	return ChatResponse{}, errors.New("expired continuation")
}

func (r *fallbackResponder) Respond(_ context.Context, request ChatRequest) (ChatResponse, error) {
	r.mu.Lock()
	r.requests = append(r.requests, request)
	r.mu.Unlock()
	return ChatResponse{Content: "本地历史重建后的回答"}, nil
}

func TestP03W06ProviderContinuationFailureFallsBackToLocalHistory(t *testing.T) {
	store, _ := NewStore("")
	createTestUser(t, store, "fallback@example.com")
	access := createAuthenticatedTestSession(t, store, "fallback@example.com")
	conversation, _ := store.CreateConversation(access, "续聊")
	if _, err := store.CreateRun(access, conversation.ID, "前文事实是蓝色", "gpt-5.6", "已记住蓝色"); err != nil {
		t.Fatal(err)
	}
	expires := time.Now().UTC().Add(time.Hour)
	if err := store.SaveProviderState(access, ProviderConversationState{ConversationID: conversation.ID, BranchID: conversation.ActiveBranchID, Provider: "upstream", Model: "gpt-5.6", ContinuationID: "expired-id", Status: "active", ExpiresAt: &expires}); err != nil {
		t.Fatal(err)
	}
	responder := &fallbackResponder{}
	api := NewAPI(store)
	api.ChatRuntimeMode = "upstream"
	api.ChatResponder = responder
	response := requestJSON(t, api.Handler(), http.MethodPost, "/api/mobile/v1/conversations/"+conversation.ID+"/runs", map[string]string{"body": "前文是什么颜色？", "model": "gpt-5.6", "idempotency_key": "fallback-run"}, access, "")
	if response.Code != http.StatusAccepted {
		t.Fatalf("run response=%d %s", response.Code, response.Body.String())
	}
	var run MessageRun
	if err := json.Unmarshal(response.Body.Bytes(), &run); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
		current, _, err := store.Run(access, run.ID)
		if err != nil {
			t.Fatal(err)
		}
		if current.Status == "completed" {
			run = current
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if run.Status != "completed" || !run.ContinuationUsed || !run.ContinuationFallback {
		t.Fatalf("fallback run=%+v", run)
	}
	build, err := store.ContextBuildOwned(access, run.ID)
	if err != nil || build.ContinuationMode != "local_rebuild_after_continuation_failure" || !strings.Contains(providerMessagesText(build.Messages), "前文事实是蓝色") {
		t.Fatalf("fallback build=%+v err=%v", build, err)
	}
}

func providerMessagesText(messages []ProviderMessage) string {
	var builder strings.Builder
	for _, message := range messages {
		builder.WriteString(message.Role)
		builder.WriteString(":")
		builder.WriteString(message.Content)
		builder.WriteString("\n")
	}
	return builder.String()
}

func TestP03W06CanonicalContextAPIsEnforceOwnershipIdempotencyAndCompaction(t *testing.T) {
	store, _ := NewStore("")
	createTestUser(t, store, "context-api@example.com")
	createTestUser(t, store, "context-api-other@example.com")
	access := createAuthenticatedTestSession(t, store, "context-api@example.com")
	otherAccess := createAuthenticatedTestSession(t, store, "context-api-other@example.com")
	conversation, err := store.CreateConversation(access, "规范上下文")
	if err != nil {
		t.Fatal(err)
	}
	api := NewAPI(store)

	created := requestJSON(t, api.Handler(), http.MethodPost, "/internal/v1/conversations/"+conversation.ID+"/context", map[string]string{
		"role": "user", "body": "预算125000元，必须在2026年9月18日前交付", "idempotency_key": "context-write-1",
	}, access, "")
	if created.Code != http.StatusCreated || !strings.Contains(created.Body.String(), `"created":true`) {
		t.Fatalf("context create=%d %s", created.Code, created.Body.String())
	}
	duplicate := requestJSON(t, api.Handler(), http.MethodPost, "/internal/v1/conversations/"+conversation.ID+"/context", map[string]string{
		"role": "user", "body": "预算125000元，必须在2026年9月18日前交付", "idempotency_key": "context-write-1",
	}, access, "")
	if duplicate.Code != http.StatusOK || !strings.Contains(duplicate.Body.String(), `"created":false`) {
		t.Fatalf("context duplicate=%d %s", duplicate.Code, duplicate.Body.String())
	}
	conflict := requestJSON(t, api.Handler(), http.MethodPost, "/internal/v1/conversations/"+conversation.ID+"/context", map[string]string{
		"role": "user", "body": "不同请求", "idempotency_key": "context-write-1",
	}, access, "")
	if conflict.Code != http.StatusConflict || !strings.Contains(conflict.Body.String(), "idempotency_conflict") {
		t.Fatalf("context conflict=%d %s", conflict.Code, conflict.Body.String())
	}

	snapshot := requestJSON(t, api.Handler(), http.MethodGet, "/internal/v1/conversations/"+conversation.ID+"/context", nil, access, "")
	if snapshot.Code != http.StatusOK || !strings.Contains(snapshot.Body.String(), "125000") || !strings.Contains(snapshot.Body.String(), "message_parts") {
		t.Fatalf("context snapshot=%d %s", snapshot.Code, snapshot.Body.String())
	}
	items := requestJSON(t, api.Handler(), http.MethodGet, "/internal/v1/conversations/"+conversation.ID+"/context-items", nil, access, "")
	if items.Code != http.StatusOK || !strings.Contains(items.Body.String(), `"status":"success"`) || !strings.Contains(items.Body.String(), `"item_type":"message"`) {
		t.Fatalf("context items=%d %s", items.Code, items.Body.String())
	}
	denied := requestJSON(t, api.Handler(), http.MethodGet, "/internal/v1/conversations/"+conversation.ID+"/context", nil, otherAccess, "")
	if denied.Code != http.StatusNotFound {
		t.Fatalf("cross-user context=%d %s", denied.Code, denied.Body.String())
	}

	compact := requestJSON(t, api.Handler(), http.MethodPost, "/internal/v1/conversations/"+conversation.ID+"/compact", map[string]string{
		"idempotency_key": "compact-1",
	}, access, "")
	if compact.Code != http.StatusOK {
		t.Fatalf("compact=%d %s", compact.Code, compact.Body.String())
	}
	var queued struct {
		Compaction ContextCompaction `json:"compaction"`
		Created    bool              `json:"created"`
	}
	if err := json.Unmarshal(compact.Body.Bytes(), &queued); err != nil || !queued.Created || queued.Compaction.Status != "queued" {
		t.Fatalf("queued=%+v err=%v", queued, err)
	}
	var completed ContextCompaction
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
		completed, err = store.ContextCompactionOwned(access, queued.Compaction.ID)
		if err != nil {
			t.Fatal(err)
		}
		if completed.Status == "success" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if completed.Status != "success" || completed.SummaryID == "" || completed.Attempts != 1 {
		t.Fatalf("completed compaction=%+v", completed)
	}
	after, err := store.ConversationContext(access, conversation.ID)
	if err != nil || after.Summary == nil || len(after.Messages) != 1 || !strings.Contains(after.Summary.SummaryText, "125000") {
		t.Fatalf("compacted context=%+v err=%v", after, err)
	}
	replayed, createdAgain, err := store.QueueConversationCompaction(access, conversation.ID, "compact-1")
	if err != nil || createdAgain || replayed.ID != completed.ID || replayed.Status != "success" {
		t.Fatalf("compaction replay=%+v created=%v err=%v", replayed, createdAgain, err)
	}
	if _, _, err := store.QueueConversationCompaction(otherAccess, conversation.ID, "compact-other"); err == nil || err.Error() != "conversation_not_found" {
		t.Fatalf("cross-user compaction err=%v", err)
	}
}
