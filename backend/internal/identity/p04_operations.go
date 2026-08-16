package identity

import (
	"errors"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

var secretReferencePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,127}$`)

type RoutingPlan struct {
	Configured  bool
	Targets     []ProviderChannel
	MaxAttempts int
}

func validProviderChannel(input ProviderChannel) bool {
	endpoint, err := url.ParseRequestURI(input.Endpoint)
	return modelCatalogIDPattern.MatchString(input.ID) && modelCatalogIDPattern.MatchString(input.ProviderID) &&
		strings.TrimSpace(input.Name) != "" && secretReferencePattern.MatchString(input.CredentialRef) && err == nil &&
		(endpoint.Scheme == "https" || endpoint.Scheme == "http") && endpoint.Host != ""
}

func validRuntimePolicy(value ProviderRuntimePolicy) bool {
	return strings.TrimSpace(value.ProviderID) != "" && value.MaxConcurrency > 0 && value.MaxConcurrency <= 1000 &&
		value.TimeoutSeconds > 0 && value.TimeoutSeconds <= 600 && value.CircuitThreshold > 0 && value.CircuitThreshold <= 100 &&
		value.CooldownSeconds > 0 && value.CooldownSeconds <= 3600
}

func validCapabilityConfig(value ModelCapabilityConfig) bool {
	return strings.TrimSpace(value.ModelID) != "" && value.ContextLimitTokens >= 1024 &&
		value.OutputReserveTokens >= 0 && value.ReasoningReserveTokens >= 0 && value.ToolReserveTokens >= 0 &&
		value.SafetyMarginTokens >= 0 &&
		value.ContextLimitTokens > value.OutputReserveTokens+value.ReasoningReserveTokens+value.ToolReserveTokens+value.SafetyMarginTokens
}

func (s *Store) ListProviderChannels() ([]ProviderChannel, error) {
	if s.conversationSQL != nil { return s.conversationSQL.listProviderChannels() }
	s.mu.Lock(); defer s.mu.Unlock()
	items := make([]ProviderChannel, 0, len(s.data.ProviderChannels))
	for _, item := range s.data.ProviderChannels { items = append(items, item) }
	sort.Slice(items, func(i, j int) bool {
		if items[i].ProviderID == items[j].ProviderID && items[i].Priority == items[j].Priority { return items[i].ID < items[j].ID }
		if items[i].ProviderID == items[j].ProviderID { return items[i].Priority < items[j].Priority }
		return items[i].ProviderID < items[j].ProviderID
	})
	return items, nil
}

func (s *Store) UpsertProviderChannel(input ProviderChannel, actorID string) (ProviderChannel, error) {
	input.ID, input.ProviderID, input.Name = strings.TrimSpace(input.ID), strings.TrimSpace(input.ProviderID), strings.TrimSpace(input.Name)
	input.CredentialRef, input.Endpoint = strings.TrimSpace(input.CredentialRef), strings.TrimSpace(input.Endpoint)
	if !validProviderChannel(input) || strings.TrimSpace(actorID) == "" { return ProviderChannel{}, errors.New("provider_channel_invalid") }
	if s.conversationSQL != nil { return s.conversationSQL.upsertProviderChannel(input, actorID) }
	s.mu.Lock(); defer s.mu.Unlock()
	providerFound := false; for _, provider := range s.data.ModelProviders { if provider.ID == input.ProviderID { providerFound = true; break } }; if !providerFound { return ProviderChannel{}, errors.New("model_provider_not_found") }
	current, found := s.data.ProviderChannels[input.ID]
	if found {
		if input.Version != current.Version { return ProviderChannel{}, errors.New("version_conflict") }
		input.Version = current.Version + 1
	} else {
		if input.Version != 0 && input.Version != 1 { return ProviderChannel{}, errors.New("version_conflict") }
		input.Version = 1
	}
	input.UpdatedAt = time.Now().UTC(); s.data.ProviderChannels[input.ID] = input
	s.appendAuditLocked("p04_provider_channel_updated", actorID, input.ID)
	return input, s.persistLocked()
}

func (s *Store) ListRoutingPolicies() ([]RoutingPolicy, error) {
	if s.conversationSQL != nil { return s.conversationSQL.listRoutingPolicies() }
	s.mu.Lock(); defer s.mu.Unlock(); items := make([]RoutingPolicy, 0, len(s.data.RoutingPolicies)); for _, item := range s.data.RoutingPolicies { items = append(items, item) }; sort.Slice(items, func(i,j int) bool { return items[i].ID < items[j].ID }); return items, nil
}

func (s *Store) UpsertRoutingPolicy(input RoutingPolicy, actorID string) (RoutingPolicy, error) {
	input.ID, input.ModelID, input.Primary = strings.TrimSpace(input.ID), strings.TrimSpace(input.ModelID), strings.TrimSpace(input.Primary)
	if !modelCatalogIDPattern.MatchString(input.ID) || input.ModelID == "" || input.Primary == "" || input.MaxAttempts < 1 || input.MaxAttempts > 10 || strings.TrimSpace(actorID) == "" { return RoutingPolicy{}, errors.New("routing_policy_invalid") }
	if s.conversationSQL != nil { return s.conversationSQL.upsertRoutingPolicy(input, actorID) }
	s.mu.Lock(); defer s.mu.Unlock()
	modelProviderID := ""; for _, model := range s.data.ModelCatalog { if model.ID == input.ModelID { modelProviderID = model.ProviderID; break } }; if modelProviderID == "" { return RoutingPolicy{}, errors.New("model_not_found") }
	primary, found := s.data.ProviderChannels[input.Primary]; if !found { return RoutingPolicy{}, errors.New("provider_channel_not_found") }; if primary.ProviderID != modelProviderID { return RoutingPolicy{}, errors.New("routing_channel_provider_mismatch") }
	seen := map[string]bool{input.Primary:true}; cleanFallbacks := []string{}
	for _, channelID := range input.Fallbacks { channelID = strings.TrimSpace(channelID); if channelID == "" || seen[channelID] { return RoutingPolicy{}, errors.New("routing_policy_invalid") }; channel, found := s.data.ProviderChannels[channelID]; if !found { return RoutingPolicy{}, errors.New("provider_channel_not_found") }; if channel.ProviderID != modelProviderID { return RoutingPolicy{}, errors.New("routing_channel_provider_mismatch") }; seen[channelID] = true; cleanFallbacks = append(cleanFallbacks, channelID) }
	input.Fallbacks = cleanFallbacks; current, found := s.data.RoutingPolicies[input.ID]
	if found { if input.Version != current.Version { return RoutingPolicy{}, errors.New("version_conflict") }; input.Version = current.Version + 1 } else if input.Version != 0 && input.Version != 1 { return RoutingPolicy{}, errors.New("version_conflict") } else { input.Version = 1 }
	input.UpdatedAt = time.Now().UTC(); s.data.RoutingPolicies[input.ID] = input; s.appendAuditLocked("p04_routing_policy_updated", actorID, input.ID); return input, s.persistLocked()
}

func (s *Store) GetProviderRuntimePolicy(providerID string) (ProviderRuntimePolicy, error) {
	providerID = strings.TrimSpace(providerID)
	if s.conversationSQL != nil { return s.conversationSQL.providerRuntimePolicy(providerID) }
	s.mu.Lock(); defer s.mu.Unlock(); value, found := s.data.ProviderRuntimePolicies[providerID]; if !found { return ProviderRuntimePolicy{}, errors.New("provider_runtime_policy_not_found") }; return value, nil
}

func (s *Store) ProviderRuntimePolicyForModel(modelID string) (ProviderRuntimePolicy, error) {
	modelID = strings.TrimSpace(modelID)
	if s.conversationSQL != nil {
		return s.conversationSQL.providerRuntimePolicyForModel(modelID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	providerID := ""
	for _, item := range s.data.ModelCatalog {
		if item.ID == modelID {
			providerID = item.ProviderID
			break
		}
	}
	if providerID == "" {
		if len(s.data.ModelCatalog) == 1 && normalizeCatalogEntry(s.data.ModelCatalog[0]).ID == "ylven-default" {
			return ProviderRuntimePolicy{ProviderID: "upstream", MaxConcurrency: 8, TimeoutSeconds: 90, CircuitThreshold: 3, CooldownSeconds: 60, Enabled: true, Version: 1}, nil
		}
		return ProviderRuntimePolicy{}, errors.New("model_not_found")
	}
	if value, found := s.data.ProviderRuntimePolicies[providerID]; found {
		return value, nil
	}
	return ProviderRuntimePolicy{ProviderID: providerID, MaxConcurrency: 8, TimeoutSeconds: 90, CircuitThreshold: 3, CooldownSeconds: 60, Enabled: true, Version: 1}, nil
}

func (s *Store) RoutingPlanForModel(modelID string) (RoutingPlan, error) {
	modelID = strings.TrimSpace(modelID)
	if s.conversationSQL != nil { return s.conversationSQL.routingPlanForModel(modelID) }
	s.mu.Lock(); defer s.mu.Unlock()
	var policy RoutingPolicy
	found := false
	for _, candidate := range s.data.RoutingPolicies {
		if candidate.ModelID != modelID || !candidate.Enabled { continue }
		if !found || candidate.UpdatedAt.After(policy.UpdatedAt) || (candidate.UpdatedAt.Equal(policy.UpdatedAt) && candidate.ID > policy.ID) { policy, found = candidate, true }
	}
	if !found { return RoutingPlan{}, nil }
	plan := RoutingPlan{Configured:true, MaxAttempts:policy.MaxAttempts}
	if plan.MaxAttempts < 1 { plan.MaxAttempts = 1 }
	for _, channelID := range append([]string{policy.Primary}, policy.Fallbacks...) {
		channel, exists := s.data.ProviderChannels[channelID]
		if !exists || !channel.Enabled { continue }
		plan.Targets = append(plan.Targets, channel)
		if len(plan.Targets) >= plan.MaxAttempts { break }
	}
	return plan, nil
}

func (s *Store) SetRunProvider(access, runID, provider string) error {
	provider = strings.TrimSpace(provider)
	if provider == "" { return errors.New("run_provider_invalid") }
	if s.conversationSQL != nil { user, err := s.authenticatedUser(access); if err != nil { return err }; return s.conversationSQL.setRunProvider(user.ID, runID, provider) }
	s.mu.Lock(); defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access); if err != nil { return err }
	run, found := s.data.Runs[runID]; if !found || run.UserID != user.ID { return errors.New("run_not_found") }
	if run.Status != "streaming" { return errors.New("run_not_streaming") }
	run.Provider, run.UpdatedAt = provider, time.Now().UTC(); s.data.Runs[runID] = run
	return s.persistLocked()
}

func (s *Store) UpsertProviderRuntimePolicy(input ProviderRuntimePolicy, actorID string) (ProviderRuntimePolicy, error) {
	input.ProviderID = strings.TrimSpace(input.ProviderID); if !validRuntimePolicy(input) || strings.TrimSpace(actorID) == "" { return ProviderRuntimePolicy{}, errors.New("provider_runtime_policy_invalid") }
	if s.conversationSQL != nil { return s.conversationSQL.upsertProviderRuntimePolicy(input, actorID) }
	s.mu.Lock(); defer s.mu.Unlock(); current, found := s.data.ProviderRuntimePolicies[input.ProviderID]; if found { if input.Version != current.Version { return ProviderRuntimePolicy{}, errors.New("version_conflict") }; input.Version=current.Version+1 } else if input.Version != 0 && input.Version != 1 { return ProviderRuntimePolicy{}, errors.New("version_conflict") } else { input.Version=1 }; input.UpdatedAt=time.Now().UTC(); s.data.ProviderRuntimePolicies[input.ProviderID]=input; s.appendAuditLocked("p04_provider_runtime_policy_updated", actorID, input.ProviderID); return input,s.persistLocked()
}

func (s *Store) GetModelCapabilityConfig(modelID string) (ModelCapabilityConfig, error) {
	modelID = strings.TrimSpace(modelID); if s.conversationSQL != nil { return s.conversationSQL.modelCapabilityConfig(modelID) }
	s.mu.Lock(); defer s.mu.Unlock(); if value, found := s.data.ModelCapabilities[modelID]; found { return value,nil }; return ModelCapabilityConfig{ModelID:modelID, ModelCapability:DefaultModelCapability(modelID)},nil
}

func (s *Store) UpsertModelCapabilityConfig(input ModelCapabilityConfig, actorID string) (ModelCapabilityConfig, error) {
	input.ModelID = strings.TrimSpace(input.ModelID); if !validCapabilityConfig(input) || strings.TrimSpace(actorID)=="" { return ModelCapabilityConfig{}, errors.New("model_capability_invalid") }
	if s.conversationSQL != nil { return s.conversationSQL.upsertModelCapabilityConfig(input, actorID) }
	s.mu.Lock(); defer s.mu.Unlock(); found:=false; for _,model:=range s.data.ModelCatalog { if model.ID==input.ModelID { found=true; break } }; if !found{return ModelCapabilityConfig{},errors.New("model_not_found")}; input.UpdatedAt=time.Now().UTC();s.data.ModelCapabilities[input.ModelID]=input;s.appendAuditLocked("p04_model_capability_updated",actorID,input.ModelID);return input,s.persistLocked()
}

func (s *Store) RecordModelHealthAdmin(input ModelHealthStatus, actorID string) (ModelHealthStatus, error) {
	input.ModelID, input.ProviderID, input.Status = strings.TrimSpace(input.ModelID), strings.TrimSpace(input.ProviderID), strings.TrimSpace(input.Status)
	if input.Status != "available" && input.Status != "degraded" && input.Status != "unavailable" { return ModelHealthStatus{}, errors.New("health_status_invalid") }
	if input.LatencyMs < 0 || input.ModelID == "" || strings.TrimSpace(actorID)=="" { return ModelHealthStatus{}, errors.New("model_health_invalid") }
	if input.LastProbeAt.IsZero() { input.LastProbeAt=time.Now().UTC() }
	if s.conversationSQL != nil { return s.conversationSQL.recordModelHealthAdmin(input, actorID) }
	s.mu.Lock(); defer s.mu.Unlock(); if input.ProviderID=="" { for _,model:=range s.data.ModelCatalog { if model.ID==input.ModelID { input.ProviderID=model.ProviderID; break } } }; if input.ProviderID=="" { return ModelHealthStatus{},errors.New("model_not_found") }; input.Capabilities=append([]string(nil),input.Capabilities...);s.data.ModelHealth[input.ModelID]=input;s.appendAuditLocked("p04_model_health_recorded",actorID,input.ModelID);return input,s.persistLocked()
}
