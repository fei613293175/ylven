package identity

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestP04PreferencesBranchesComparisonsAndSynthesisUseRealRuns(t *testing.T) {
	store, err := NewStore("")
	if err != nil {
		t.Fatal(err)
	}
	access := seedP04ModelCatalog(t, store)
	preference, err := store.PutAIPreference(access, AIPreference{ModelID: "reasoner", ReasoningProfile: "deep", Version: 1})
	if err != nil || preference.Version != 1 || preference.ModelID != "reasoner" {
		t.Fatalf("preference=%+v err=%v", preference, err)
	}
	conversation, err := store.CreateConversation(access, "P04 state")
	if err != nil {
		t.Fatal(err)
	}
	if conversation.AISettingsVersion != 1 || conversation.AISettingsOverridden {
		t.Fatalf("unexpected new conversation settings: %+v", conversation)
	}
	conversation, err = store.PutConversationAISettings(access, conversation.ID, "writer", "auto", conversation.AISettingsVersion)
	if err != nil || !conversation.AISettingsOverridden || conversation.AISettingsVersion != 2 {
		t.Fatalf("conversation settings=%+v err=%v", conversation, err)
	}
	if _, err = store.PutConversationAISettings(access, conversation.ID, "reasoner", "deep", 1); err == nil || err.Error() != "version_conflict" {
		t.Fatalf("stale conversation settings err=%v", err)
	}

	first, created, err := store.StartRunIdempotentResultWithProfile(access, conversation.ID, "保留这条旧回答", "writer", "auto", "p04-original")
	if err != nil || !created {
		t.Fatalf("first=%+v created=%v err=%v", first, created, err)
	}
	completed, err := store.CompleteRun(access, first.ID, "旧答案")
	if err != nil {
		t.Fatal(err)
	}
	rerun, err := store.RerunMessage(access, completed.AssistantMessageID, "reasoner", "deep")
	if err != nil || rerun.BranchID == first.BranchID || rerun.Model != "reasoner" || rerun.ReasoningProfile != "deep" {
		t.Fatalf("rerun=%+v first=%+v err=%v", rerun, first, err)
	}
	if _, err = store.CompleteRun(access, rerun.ID, "新答案"); err != nil {
		t.Fatal(err)
	}
	oldRun, _, err := store.Run(access, first.ID)
	if err != nil || oldRun.Status != "completed" {
		t.Fatalf("old run was not preserved: %+v err=%v", oldRun, err)
	}

	group, err := store.CreateComparison(access, conversation.ID, "比较这两个方案", []string{"reasoner", "writer"}, "auto")
	if err != nil || len(group.Candidates) != 2 || group.Status != "running" {
		t.Fatalf("group=%+v err=%v", group, err)
	}
	for _, candidate := range group.Candidates {
		if _, err = store.CompleteRun(access, candidate.RunID, candidate.ModelID+" answer"); err != nil {
			t.Fatal(err)
		}
	}
	group, err = store.GetComparison(access, group.ID)
	if err != nil || group.Status != "completed" || group.Candidates[0].Body == "" || group.Candidates[1].Body == "" {
		t.Fatalf("completed comparison=%+v err=%v", group, err)
	}
	selected := group.Candidates[0]
	group, err = store.AdoptComparison(access, group.ID, selected.RunID)
	if err != nil || group.Status != "adopted" || group.AdoptedRunID != selected.RunID {
		t.Fatalf("adopted=%+v err=%v", group, err)
	}
	synthesized, err := store.SynthesizeComparison(access, group.ID, nil)
	if err != nil || synthesized.SynthesisRunID == "" || synthesized.Status != "synthesizing" {
		t.Fatalf("synthesis=%+v err=%v", synthesized, err)
	}
	if _, err = store.CompleteRun(access, synthesized.SynthesisRunID, "综合答案"); err != nil {
		t.Fatal(err)
	}
	group, err = store.GetComparison(access, group.ID)
	if err != nil || group.Status != "synthesized" {
		t.Fatalf("synthesis completion=%+v err=%v", group, err)
	}
}

func TestP04W02HTTPResolvesDefaultsBeforeRoutingAndPreservesRunProvenance(t *testing.T) {
	store, err := NewStore("")
	if err != nil {
		t.Fatal(err)
	}
	access := seedP04ModelCatalog(t, store)
	conversation, err := store.CreateConversation(access, "P04 preference precedence")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.PutAIPreference(access, AIPreference{ModelID: "reasoner", ReasoningProfile: "deep", Version: 1}); err != nil {
		t.Fatal(err)
	}
	channel, err := store.UpsertProviderChannel(ProviderChannel{
		ID: "reasoner-default-route", ProviderID: "openai", Name: "Reasoner default",
		CredentialRef: "P04_DEFAULT_ROUTE_KEY", Endpoint: "https://route.example.test", Enabled: true, Priority: 1,
	}, "admin-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.UpsertRoutingPolicy(RoutingPolicy{ID: "reasoner-default-policy", ModelID: "reasoner", Primary: channel.ID, MaxAttempts: 1, Enabled: true}, "admin-1"); err != nil {
		t.Fatal(err)
	}

	api := NewAPI(store)
	api.ChatRuntimeMode = "unconfigured"
	api.ChannelResponderFactory = func(ProviderChannel) (ChatResponder, error) {
		return p04ChannelResponder{content: "default route answer"}, nil
	}
	created := requestJSON(t, api.Handler(), "POST", "/api/mobile/v1/conversations/"+conversation.ID+"/runs", map[string]any{
		"body": "use my defaults", "idempotency_key": "p04-default-route",
	}, access, "")
	if created.Code != 202 {
		t.Fatalf("default run status=%d body=%s", created.Code, created.Body.String())
	}
	var run MessageRun
	if err = json.Unmarshal(created.Body.Bytes(), &run); err != nil {
		t.Fatal(err)
	}
	if run.Model != "reasoner" || run.ReasoningProfile != "deep" {
		t.Fatalf("default precedence was not exposed in run: %+v", run)
	}
	persisted, _, err := store.Run(access, run.ID)
	if err != nil || persisted.ReasoningParameters["reasoning_effort"] != "high" {
		t.Fatalf("default reasoning parameters were not persisted: run=%+v err=%v", persisted, err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		stored, _, runErr := store.Run(access, run.ID)
		if runErr == nil && stored.Status == "completed" {
			run = stored
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if run.Status != "completed" || run.AssistantMessageID == "" {
		t.Fatalf("default route run did not complete: %+v", run)
	}
	metadata := requestJSON(t, api.Handler(), "GET", "/api/mobile/v1/messages/"+run.AssistantMessageID+"/run-metadata", nil, access, "")
	if metadata.Code != 200 {
		t.Fatalf("run metadata status=%d body=%s", metadata.Code, metadata.Body.String())
	}
	var recovered MessageRun
	if err = json.Unmarshal(metadata.Body.Bytes(), &recovered); err != nil {
		t.Fatal(err)
	}
	if recovered.ID != run.ID || recovered.Model != run.Model || recovered.ReasoningProfile != run.ReasoningProfile {
		t.Fatalf("answer metadata diverged from persisted run: got=%+v want=%+v", recovered, run)
	}
}

func TestP04W02ExplicitOverrideWinsAndBranchSnapshotsOnlyHistoryBeforeFork(t *testing.T) {
	store, err := NewStore("")
	if err != nil {
		t.Fatal(err)
	}
	access := seedP04ModelCatalog(t, store)
	conversation, err := store.CreateConversation(access, "P04 explicit override")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.PutAIPreference(access, AIPreference{ModelID: "reasoner", ReasoningProfile: "deep", Version: 1}); err != nil {
		t.Fatal(err)
	}
	conversation, err = store.PutConversationAISettings(access, conversation.ID, "writer", "auto", conversation.AISettingsVersion)
	if err != nil {
		t.Fatal(err)
	}
	first, created, err := store.StartRunIdempotentResultWithProfile(access, conversation.ID, "keep before fork", "reasoner", "deep", "p04-before-fork")
	if err != nil || !created {
		t.Fatalf("first run=%+v created=%v err=%v", first, created, err)
	}
	if first.Model != "reasoner" || first.ReasoningProfile != "deep" {
		t.Fatalf("per-message override did not win: %+v", first)
	}
	first, err = store.CompleteRun(access, first.ID, "old answer stays on source branch")
	if err != nil {
		t.Fatal(err)
	}
	second, created, err := store.StartRunIdempotentResult(access, conversation.ID, "do not copy after fork", "", "p04-after-fork")
	if err != nil || !created {
		t.Fatalf("second run=%+v created=%v err=%v", second, created, err)
	}
	if _, err = store.CompleteRun(access, second.ID, "later answer"); err != nil {
		t.Fatal(err)
	}
	branch, err := store.CreateConversationBranch(access, conversation.ID, first.AssistantMessageID)
	if err != nil {
		t.Fatal(err)
	}
	messages, err := store.ConversationMessages(access, conversation.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 2 || messages[0].Body != "keep before fork" || messages[1].Body != "old answer stays on source branch" {
		t.Fatalf("branch %s did not snapshot exactly through the fork: %+v", branch.ID, messages)
	}
	for _, message := range messages {
		if message.BranchID != branch.ID {
			t.Fatalf("snapshot returned source branch message: %+v", message)
		}
	}
}

type p04ChannelResponder struct {
	content string
	err     error
}

func (r p04ChannelResponder) Respond(context.Context, ChatRequest) (ChatResponse, error) {
	if r.err != nil {
		return ChatResponse{}, r.err
	}
	return ChatResponse{Content: r.content}, nil
}

func TestP04RoutingPolicyUsesFallbackChannelAfterPrimaryFailure(t *testing.T) {
	store, err := NewStore("")
	if err != nil {
		t.Fatal(err)
	}
	access := seedP04ModelCatalog(t, store)
	conversation, err := store.CreateConversation(access, "Route fallback")
	if err != nil {
		t.Fatal(err)
	}
	primary, err := store.UpsertProviderChannel(ProviderChannel{ID: "primary", ProviderID: "openai", Name: "Primary", CredentialRef: "P04_PRIMARY_KEY", Endpoint: "https://primary.example.test", Enabled: true, Priority: 1}, "admin-1")
	if err != nil {
		t.Fatal(err)
	}
	backup, err := store.UpsertProviderChannel(ProviderChannel{ID: "backup", ProviderID: "openai", Name: "Backup", CredentialRef: "P04_BACKUP_KEY", Endpoint: "https://backup.example.test", Enabled: true, Priority: 2}, "admin-1")
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.UpsertRoutingPolicy(RoutingPolicy{ID: "reasoner-route", ModelID: "reasoner", Primary: primary.ID, Fallbacks: []string{backup.ID}, MaxAttempts: 2, Enabled: true}, "admin-1")
	if err != nil {
		t.Fatal(err)
	}
	api := NewAPI(store)
	api.ChatRuntimeMode = "upstream"
	api.ChatResponder = p04ChannelResponder{err: errors.New("global responder must not run")}
	var used []string
	api.ChannelResponderFactory = func(channel ProviderChannel) (ChatResponder, error) {
		used = append(used, channel.ID)
		if channel.ID == primary.ID {
			return p04ChannelResponder{err: errors.New("primary unavailable")}, nil
		}
		return p04ChannelResponder{content: "备用通道答案"}, nil
	}
	run, created, err := store.StartRunIdempotentResultWithProfile(access, conversation.ID, "请路由", "reasoner", "auto", "p04-route")
	if err != nil || !created {
		t.Fatalf("run=%+v created=%v err=%v", run, created, err)
	}
	api.dispatchRun(run, access, false)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		stored, _, runErr := store.Run(access, run.ID)
		if runErr == nil && stored.Status == "completed" {
			if stored.Provider != "openai" || len(used) != 2 || used[0] != primary.ID || used[1] != backup.ID {
				t.Fatalf("route provider=%s used=%v", stored.Provider, used)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	stored, _, _ := store.Run(access, run.ID)
	t.Fatalf("fallback run did not complete: %+v used=%v", stored, used)
}

func TestP04MobileRunUsesConfiguredRouteWithoutGlobalResponder(t *testing.T) {
	store, err := NewStore("")
	if err != nil {
		t.Fatal(err)
	}
	access := seedP04ModelCatalog(t, store)
	conversation, err := store.CreateConversation(access, "Configured route only")
	if err != nil {
		t.Fatal(err)
	}
	channel, err := store.UpsertProviderChannel(ProviderChannel{ID: "route-only", ProviderID: "openai", Name: "Route only", CredentialRef: "P04_ROUTE_ONLY_KEY", Endpoint: "https://route.example.test", Enabled: true, Priority: 1}, "admin-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.UpsertRoutingPolicy(RoutingPolicy{ID: "route-only-policy", ModelID: "reasoner", Primary: channel.ID, MaxAttempts: 1, Enabled: true}, "admin-1"); err != nil {
		t.Fatal(err)
	}
	api := NewAPI(store)
	api.ChatRuntimeMode = "unconfigured"
	api.ChatResponder = nil
	api.ChannelResponderFactory = func(ProviderChannel) (ChatResponder, error) {
		return p04ChannelResponder{content: "路由成功"}, nil
	}
	response := requestJSON(t, api.Handler(), "POST", "/api/mobile/v1/conversations/"+conversation.ID+"/runs", map[string]any{
		"body": "使用已配置通道", "model": "reasoner", "idempotency_key": "route-only-run",
	}, access, "")
	if response.Code != 202 {
		t.Fatalf("route-only run status=%d body=%s", response.Code, response.Body.String())
	}
	var run MessageRun
	if err = json.Unmarshal(response.Body.Bytes(), &run); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		stored, _, runErr := store.Run(access, run.ID)
		if runErr == nil && stored.Status == "completed" {
			if stored.Provider != "openai" {
				t.Fatalf("route-only provider=%s", stored.Provider)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	stored, _, _ := store.Run(access, run.ID)
	t.Fatalf("route-only run did not complete: %+v", stored)
}

func TestP04AvailabilityExcludesUnavailableFallbacksAndUsageIsIdempotent(t *testing.T) {
	store, err := NewStore("")
	if err != nil {
		t.Fatal(err)
	}
	access := seedP04ModelCatalog(t, store)
	if _, err = store.RecordModelHealth(access, "writer", "unavailable", "health_probe_failed", 1, nil); err != nil {
		t.Fatal(err)
	}
	availability, err := store.ModelAvailability(access, "reasoner")
	if err != nil {
		t.Fatal(err)
	}
	fallbacks := availability["fallback_models"].([]ModelCatalogEntry)
	if len(fallbacks) != 0 {
		t.Fatalf("unavailable model exposed as fallback: %+v", fallbacks)
	}
	event := UsageEvent{ID: "usage-p04", RunID: "run-p04", ModelID: "reasoner", InputTokens: 2, OutputTokens: 3}
	if _, err = store.RecordUsage(access, event); err != nil {
		t.Fatal(err)
	}
	if _, err = store.RecordUsage(access, event); err == nil || err.Error() != "usage_event_already_recorded" {
		t.Fatalf("duplicate usage err=%v", err)
	}
}

func TestP04PublicServiceStatusIsAnonymousAndDoesNotExposeRuntimeErrors(t *testing.T) {
	store, err := NewStore("")
	if err != nil { t.Fatal(err) }
	access := seedP04ModelCatalog(t, store)
	if _, err = store.RecordModelHealth(access, "reasoner", "degraded", "provider_timeout", 91, []string{"text"}); err != nil { t.Fatal(err) }
	response := requestJSON(t, NewAPI(store).Handler(), "GET", "/public/v1/service-status/models", nil, "", "")
	if response.Code != 200 { t.Fatalf("public status=%d body=%s", response.Code, response.Body.String()) }
	var payload struct { Items []ModelHealthStatus `json:"items"` }
	if err = json.Unmarshal(response.Body.Bytes(), &payload); err != nil { t.Fatal(err) }
	if len(payload.Items) == 0 { t.Fatal("expected public health projection") }
	for _, item := range payload.Items { if item.ErrorCode != "" { t.Fatalf("public status leaked error code: %+v", item) } }
}

func TestP04ComparisonOperationsProjectionIsBoundedAndRetainsCandidates(t *testing.T) {
	store, err := NewStore("")
	if err != nil { t.Fatal(err) }
	access := seedP04ModelCatalog(t, store)
	conversation, err := store.CreateConversation(access, "Operations comparison")
	if err != nil { t.Fatal(err) }
	group, err := store.CreateComparison(access, conversation.ID, "Which answer is better?", []string{"reasoner", "writer"}, "auto")
	if err != nil { t.Fatal(err) }
	items, err := store.ListComparisonGroups()
	if err != nil { t.Fatal(err) }
	if len(items) != 1 || items[0].ID != group.ID || len(items[0].Candidates) != 2 { t.Fatalf("operations comparison projection=%+v", items) }
}
