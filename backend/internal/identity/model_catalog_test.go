package identity

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func seedP04ModelCatalog(t *testing.T, store *Store) string {
	t.Helper()
	createTestUser(t, store, "p04-catalog@example.com")
	access := createAuthenticatedTestSession(t, store, "p04-catalog@example.com")
	now := time.Now().UTC()
	store.mu.Lock()
	store.data.ModelProviders = []ModelProvider{
		{ID: "openai", Name: "OpenAI", Enabled: true, SortOrder: 1, Version: 1, UpdatedAt: now},
		{ID: "anthropic", Name: "Anthropic", Enabled: true, SortOrder: 2, Version: 1, UpdatedAt: now},
	}
	store.data.ModelCatalog = []ModelCatalogEntry{
		{
			ID: "reasoner", Name: "Reasoner", Enabled: true, ProviderID: "openai", ProviderName: "OpenAI",
			Purpose: "complex reasoning", SpeedTier: "deliberate", UpstreamModel: "reasoner-v2", Version: 1, UpdatedAt: now,
			ReasoningProfiles: []string{"auto", "deep"},
			ReasoningMappings: []ReasoningProfileMapping{
				{ModelID: "reasoner", ProfileID: "auto", Label: "Automatic", Enabled: true, UpstreamParameters: map[string]any{}, Version: 1, UpdatedAt: now},
				{ModelID: "reasoner", ProfileID: "deep", Label: "Deep", Ordinal: 1, Enabled: true, UpstreamParameters: map[string]any{"reasoning_effort": "high"}, Version: 1, UpdatedAt: now},
			},
		},
		{ID: "writer", Name: "Writer", Enabled: true, ProviderID: "anthropic", ProviderName: "Anthropic", Purpose: "writing", SpeedTier: "fast", UpstreamModel: "writer-v1", Version: 1, UpdatedAt: now},
	}
	store.data.CapabilityProbes = []CapabilityProbeResult{
		{ID: "probe-vision-pass", ModelID: "reasoner", CapabilityID: "vision", CapabilityLabel: "Vision", Status: "passed", ProbedAt: now.Add(-2 * time.Hour), CreatedAt: now.Add(-2 * time.Hour)},
		{ID: "probe-vision-fail", ModelID: "reasoner", CapabilityID: "vision", CapabilityLabel: "Vision", Status: "failed", ProbedAt: now.Add(-time.Hour), CreatedAt: now.Add(-time.Hour)},
		{ID: "probe-tools-pass", ModelID: "reasoner", CapabilityID: "tools", CapabilityLabel: "Tools", Status: "passed", ProbedAt: now.Add(-time.Minute), CreatedAt: now.Add(-time.Minute)},
		{ID: "probe-expired", ModelID: "reasoner", CapabilityID: "files", CapabilityLabel: "Files", Status: "passed", ProbedAt: now.Add(-3 * time.Hour), ExpiresAt: timePointer(now.Add(-time.Hour)), CreatedAt: now.Add(-3 * time.Hour)},
	}
	store.mu.Unlock()
	return access
}

func timePointer(value time.Time) *time.Time { return &value }

func TestP04W01MobileCatalogGroupsProvidersAndExposesOnlyVerifiedCapabilities(t *testing.T) {
	store, _ := NewStore("")
	access := seedP04ModelCatalog(t, store)
	handler := NewAPI(store).Handler()
	response := requestJSON(t, handler, http.MethodGet, "/api/mobile/v1/models?group=provider", nil, access, "")
	if response.Code != http.StatusOK {
		t.Fatalf("catalog status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		Items  []ModelCatalogEntry  `json:"items"`
		Groups []ModelProviderGroup `json:"groups"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Groups) != 2 || body.Groups[0].Provider.ID != "openai" || body.Groups[1].Provider.ID != "anthropic" {
		t.Fatalf("provider groups=%+v", body.Groups)
	}
	if len(body.Items) != 2 || body.Items[0].Purpose != "complex reasoning" || body.Items[0].SpeedTier != "deliberate" {
		t.Fatalf("catalog fields=%+v", body.Items)
	}
	if len(body.Items[0].Capabilities) != 1 || body.Items[0].Capabilities[0].ID != "tools" {
		t.Fatalf("unverified or expired capability exposed: %+v", body.Items[0].Capabilities)
	}

	detail := requestJSON(t, handler, http.MethodGet, "/api/mobile/v1/models/reasoner", nil, access, "")
	if detail.Code != http.StatusOK || !containsJSONField(detail.Body.Bytes(), "\"purpose\":\"complex reasoning\"") {
		t.Fatalf("detail status=%d body=%s", detail.Code, detail.Body.String())
	}
	profiles := requestJSON(t, handler, http.MethodGet, "/api/mobile/v1/models/reasoner/reasoning-profiles", nil, access, "")
	if profiles.Code != http.StatusOK || !containsJSONField(profiles.Body.Bytes(), "\"profile_id\":\"deep\"") {
		t.Fatalf("profiles status=%d body=%s", profiles.Code, profiles.Body.String())
	}
}

func containsJSONField(body []byte, expected string) bool {
	for index := 0; index+len(expected) <= len(body); index++ {
		if string(body[index:index+len(expected)]) == expected {
			return true
		}
	}
	return false
}

func TestP04W01ReasoningSelectionIsValidatedAndImmutablePerRun(t *testing.T) {
	store, _ := NewStore("")
	access := seedP04ModelCatalog(t, store)
	conversation, err := store.CreateConversation(access, "P04 mapping")
	if err != nil {
		t.Fatal(err)
	}
	run, created, err := store.StartRunIdempotentResultWithProfile(access, conversation.ID, "analyze", "reasoner", "deep", "p04-run")
	if err != nil || !created {
		t.Fatalf("run=%+v created=%v err=%v", run, created, err)
	}
	if run.ProviderModel != "reasoner-v2" || run.ReasoningProfile != "deep" || run.ReasoningParameters["reasoning_effort"] != "high" {
		t.Fatalf("resolved run mapping=%+v", run)
	}
	store.mu.Lock()
	store.data.ModelCatalog[0].ReasoningMappings[1].UpstreamParameters["reasoning_effort"] = "low"
	store.mu.Unlock()
	stored, _, err := store.Run(access, run.ID)
	if err != nil || stored.ReasoningParameters["reasoning_effort"] != "high" {
		t.Fatalf("run mapping drifted after catalog edit: run=%+v err=%v", stored, err)
	}
	if _, _, err := store.StartRunIdempotentResultWithProfile(access, conversation.ID, "analyze", "reasoner", "quick", "unsupported-profile"); err == nil || err.Error() != "reasoning_profile_unsupported" {
		t.Fatalf("unsupported profile err=%v", err)
	}
}

func TestP04W01OpenAIResponderInjectsAllowedReasoningParameters(t *testing.T) {
	var payload map[string]any
	requestErr := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewDecoder(r.Body).Decode(&payload)
		requestErr <- err
		if err != nil {
			http.Error(w, "invalid test request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"continuation-1","choices":[{"message":{"content":"done"}}]}`))
	}))
	defer server.Close()
	responder := OpenAICompatibleResponder{Endpoint: server.URL, APIKey: "secret", Client: server.Client()}
	_, err := responder.Respond(context.Background(), ChatRequest{
		Model: "reasoner-v2", Messages: []ProviderMessage{{Role: "user", Content: "analyze"}},
		ReasoningParameters: map[string]any{"reasoning_effort": "high", "thinking": map[string]any{"type": "enabled"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := <-requestErr; err != nil {
		t.Fatal(err)
	}
	if payload["model"] != "reasoner-v2" || payload["reasoning_effort"] != "high" || payload["thinking"] == nil {
		t.Fatalf("upstream payload=%+v", payload)
	}
	if _, err := responder.Respond(context.Background(), ChatRequest{Model: "reasoner-v2", Messages: []ProviderMessage{{Role: "user", Content: "analyze"}}, ReasoningParameters: map[string]any{"unsafe": true}}); err == nil || err.Error() != "reasoning_parameter_not_allowed" {
		t.Fatalf("unknown upstream parameter err=%v", err)
	}
}

func TestP04W01AdminWritesUseVersionsAndAppendEvidenceAudit(t *testing.T) {
	store, _ := NewStore("")
	provider, err := store.UpsertModelProvider(ModelProvider{ID: "openai", Name: "OpenAI", Enabled: true}, "admin-1")
	if err != nil || provider.Version != 1 {
		t.Fatalf("provider=%+v err=%v", provider, err)
	}
	updated, err := store.UpsertModelProvider(ModelProvider{ID: provider.ID, Name: "OpenAI Platform", Enabled: true, Version: provider.Version}, "admin-1")
	if err != nil || updated.Version != 2 {
		t.Fatalf("updated=%+v err=%v", updated, err)
	}
	if _, err := store.UpsertModelProvider(ModelProvider{ID: provider.ID, Name: "stale", Enabled: true, Version: 1}, "admin-1"); err == nil || err.Error() != "version_conflict" {
		t.Fatalf("stale provider update err=%v", err)
	}
	model, err := store.UpsertModelCatalogEntry(ModelCatalogEntry{ID: "reasoner", Name: "Reasoner", Enabled: true, ProviderID: provider.ID, UpstreamModel: "reasoner-v2", Purpose: "reasoning", SpeedTier: "deliberate"}, "admin-1")
	if err != nil {
		t.Fatal(err)
	}
	probe, err := store.RecordCapabilityProbe(CapabilityProbeResult{ModelID: model.ID, CapabilityID: "tools", Status: "passed", ProbeSource: "server-contract-test", EvidenceReference: "evidence://p04/tools", ProbedAt: time.Now().UTC()}, "admin-1")
	if err != nil || probe.ID == "" || probe.EvidenceReference == "" {
		t.Fatalf("probe=%+v err=%v", probe, err)
	}
	snapshot, err := store.ModelCatalogSnapshot()
	if err != nil || len(snapshot.CapabilityProbes) != 1 || len(snapshot.Audit) < 4 {
		t.Fatalf("snapshot=%+v err=%v", snapshot, err)
	}
}
