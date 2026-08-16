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
