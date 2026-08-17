package identity

import (
	"errors"
	"sort"
	"strings"
	"time"
)

func defaultAISelection() ModelSelection {
	return ModelSelection{CatalogModelID: "ylven-default", ProviderModel: "ylven-default", ReasoningProfile: "auto", ReasoningParameters: map[string]any{}}
}

func (s *Store) effectiveSelectionLocked(userID, conversationID, modelID, profileID string) (ModelSelection, error) {
	conversation, ok := s.data.Conversations[conversationID]
	if !ok || conversation.UserID != userID || conversation.DeletedAt != nil {
		return ModelSelection{}, errors.New("conversation_not_found")
	}
	if strings.TrimSpace(modelID) == "" && conversation.AISettingsOverridden {
		modelID = conversation.DefaultModelID
	}
	if strings.TrimSpace(profileID) == "" && conversation.AISettingsOverridden {
		profileID = conversation.DefaultReasoningProfile
	}
	if strings.TrimSpace(modelID) == "" || strings.TrimSpace(profileID) == "" {
		preference := s.data.AIPreferences[userID]
		if strings.TrimSpace(modelID) == "" {
			modelID = preference.ModelID
		}
		if strings.TrimSpace(profileID) == "" {
			profileID = preference.ReasoningProfile
		}
	}
	if strings.TrimSpace(modelID) == "" {
		modelID = "ylven-default"
	}
	if strings.TrimSpace(profileID) == "" {
		profileID = "auto"
	}
	return s.resolveModelSelectionLocked(modelID, profileID)
}

func (s *Store) effectiveSelection(access, conversationID, modelID, profileID string) (ModelSelection, error) {
	if s.conversationSQL != nil {
		return s.conversationSQL.effectiveSelection(access, conversationID, modelID, profileID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return ModelSelection{}, err
	}
	return s.effectiveSelectionLocked(user.ID, conversationID, modelID, profileID)
}

func (s *Store) defaultSelection(access, modelID, profileID string) (ModelSelection, error) {
	if s.conversationSQL != nil {
		return s.conversationSQL.defaultSelection(access, modelID, profileID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access)
	if err != nil {
		return ModelSelection{}, err
	}
	if strings.TrimSpace(modelID) == "" || strings.TrimSpace(profileID) == "" {
		preference := s.data.AIPreferences[user.ID]
		if strings.TrimSpace(modelID) == "" {
			modelID = preference.ModelID
		}
		if strings.TrimSpace(profileID) == "" {
			profileID = preference.ReasoningProfile
		}
	}
	return s.resolveModelSelectionLocked(modelID, profileID)
}

func (s *Store) resolveModelSelectionLocked(modelID, profileID string) (ModelSelection, error) {
	modelID = strings.TrimSpace(modelID)
	profileID = strings.TrimSpace(profileID)
	if modelID == "" { modelID = "ylven-default" }
	if profileID == "" { profileID = "auto" }
	// Preserve the P03 default runtime while the P04 catalog is empty. Once a
	// catalog exists, all non-default selections must come from that catalog.
	if modelID == "ylven-default" {
		hasDefaultCatalogEntry := false
		for _, raw := range s.data.ModelCatalog {
			if normalizeCatalogEntry(raw).ID == modelID { hasDefaultCatalogEntry = true; break }
		}
		if !hasDefaultCatalogEntry { return defaultAISelection(), nil }
	}
	providers := map[string]bool{}
	for _, provider := range s.data.ModelProviders { providers[provider.ID] = provider.Enabled }
	for _, raw := range s.data.ModelCatalog {
		item := normalizeCatalogEntry(raw)
		if item.ID != modelID { continue }
		if !item.Enabled || !providers[item.ProviderID] { return ModelSelection{}, errors.New("model_unavailable") }
		for _, profile := range item.ReasoningMappings {
			if profile.ProfileID == profileID && profile.Enabled {
				parameters := copyJSONMap(profile.UpstreamParameters)
				if err := validateUpstreamReasoningParameters(parameters); err != nil {
					return ModelSelection{}, err
				}
				return ModelSelection{CatalogModelID: item.ID, ProviderModel: item.UpstreamModel, ReasoningProfile: profileID, ReasoningParameters: parameters}, nil
			}
		}
		return ModelSelection{}, errors.New("reasoning_profile_unsupported")
	}
	if len(s.data.ModelCatalog) == 1 && normalizeCatalogEntry(s.data.ModelCatalog[0]).ID == "ylven-default" {
		// P03 accepted provider model IDs before the P04 catalog was populated.
		return ModelSelection{CatalogModelID: modelID, ProviderModel: modelID, ReasoningProfile: profileID, ReasoningParameters: map[string]any{}}, nil
	}
	return ModelSelection{}, errors.New("model_not_found")
}

func (s *Store) GetAIPreference(access string) (AIPreference, error) {
	if s.conversationSQL != nil { value, _, err := s.conversationSQL.getAIPreference(access); return value, err }
	s.mu.Lock(); defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access); if err != nil { return AIPreference{}, err }
	value, ok := s.data.AIPreferences[user.ID]
	if !ok { value = AIPreference{UserID: user.ID, ModelID: "ylven-default", ReasoningProfile: "auto", Version: 1, UpdatedAt: time.Now().UTC()} }
	return value, nil
}

func (s *Store) PutAIPreference(access string, input AIPreference) (AIPreference, error) {
	if s.conversationSQL != nil { return s.conversationSQL.putAIPreference(access, input) }
	s.mu.Lock(); defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access); if err != nil { return AIPreference{}, err }
	selection, err := s.resolveModelSelectionLocked(input.ModelID, input.ReasoningProfile); if err != nil { return AIPreference{}, err }
	if input.Version == 0 { input.Version = 1 }
	if current, ok := s.data.AIPreferences[user.ID]; ok && input.Version != current.Version { return AIPreference{}, errors.New("version_conflict") }
	input.UserID, input.ModelID, input.ReasoningProfile = user.ID, selection.CatalogModelID, selection.ReasoningProfile
	if current, ok := s.data.AIPreferences[user.ID]; ok { input.Version = current.Version + 1 }
	input.UpdatedAt = time.Now().UTC(); s.data.AIPreferences[user.ID] = input
	return input, s.persistLocked()
}

func (s *Store) GetConversationAISettings(access, conversationID string) (Conversation, error) {
	if s.conversationSQL != nil { return s.conversationSQL.getConversationAISettings(access, conversationID) }
	s.mu.Lock(); defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access); if err != nil { return Conversation{}, err }
	item, ok := s.data.Conversations[conversationID]; if !ok || item.UserID != user.ID || item.DeletedAt != nil { return Conversation{}, errors.New("conversation_not_found") }
	return item, nil
}

func (s *Store) PutConversationAISettings(access, conversationID string, modelID, profileID string, version int64) (Conversation, error) {
	if s.conversationSQL != nil { return s.conversationSQL.putConversationAISettings(access, conversationID, modelID, profileID, version) }
	s.mu.Lock(); defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access); if err != nil { return Conversation{}, err }
	item, ok := s.data.Conversations[conversationID]; if !ok || item.UserID != user.ID || item.DeletedAt != nil { return Conversation{}, errors.New("conversation_not_found") }
	selection, err := s.resolveModelSelectionLocked(modelID, profileID); if err != nil { return Conversation{}, err }
	if item.AISettingsVersion <= 0 { item.AISettingsVersion = 1 }
	if version != 0 && version != item.AISettingsVersion { return Conversation{}, errors.New("version_conflict") }
	item.DefaultModelID, item.DefaultReasoningProfile = selection.CatalogModelID, selection.ReasoningProfile
	item.AISettingsVersion++
	item.AISettingsOverridden = true
	item.UpdatedAt = time.Now().UTC()
	s.data.Conversations[conversationID] = item
	return item, s.persistLocked()
}

func (s *Store) CreateConversationBranch(access, conversationID, forkedFromMessageID string) (ConversationBranch, error) {
	if s.conversationSQL != nil { return s.conversationSQL.createConversationBranch(access, conversationID, forkedFromMessageID) }
	s.mu.Lock(); defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access); if err != nil { return ConversationBranch{}, err }
	conversation, ok := s.data.Conversations[conversationID]; if !ok || conversation.UserID != user.ID || conversation.DeletedAt != nil { return ConversationBranch{}, errors.New("conversation_not_found") }
	parentBranchID, throughMessageID := conversation.ActiveBranchID, strings.TrimSpace(forkedFromMessageID)
	if throughMessageID != "" {
		message, exists := s.data.Messages[throughMessageID]
		if !exists || message.ConversationID != conversationID || message.UserID != user.ID { return ConversationBranch{}, errors.New("message_not_found") }
		parentBranchID = message.BranchID
	} else {
		throughMessageID = s.lastBranchMessageIDLocked(conversationID, parentBranchID)
	}
	id, err := randomToken(16); if err != nil { return ConversationBranch{}, err }
	now := time.Now().UTC(); branch := ConversationBranch{ID: id, ConversationID: conversationID, ParentBranchID: parentBranchID, ForkedFromMessageID: forkedFromMessageID, Status: "active", CreatedAt: now, UpdatedAt: now}
	if err := s.cloneMemoryBranchHistoryLocked(conversationID, parentBranchID, id, throughMessageID); err != nil { return ConversationBranch{}, err }
	s.data.ConversationBranches[id] = branch; conversation.ActiveBranchID, conversation.UpdatedAt = id, now; s.data.Conversations[conversationID] = conversation
	return branch, s.persistLocked()
}

func (s *Store) ListConversationBranches(access, conversationID string) ([]ConversationBranch, error) {
	if s.conversationSQL != nil { return s.conversationSQL.listConversationBranches(access, conversationID) }
	s.mu.Lock(); defer s.mu.Unlock(); user, _, err := s.authenticatedUserLocked(access); if err != nil { return nil, err }
	conversation, ok := s.data.Conversations[conversationID]; if !ok || conversation.UserID != user.ID { return nil, errors.New("conversation_not_found") }
	items := []ConversationBranch{}; for _, branch := range s.data.ConversationBranches { if branch.ConversationID == conversationID { items = append(items, branch) } }
	sort.Slice(items, func(i,j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) }); return items, nil
}

func (s *Store) SetActiveConversationBranch(access, conversationID, branchID string) (Conversation, error) {
	if s.conversationSQL != nil { return s.conversationSQL.setActiveConversationBranch(access, conversationID, branchID) }
	s.mu.Lock(); defer s.mu.Unlock(); user, _, err := s.authenticatedUserLocked(access); if err != nil { return Conversation{}, err }
	item, ok := s.data.Conversations[conversationID]; branch, branchOK := s.data.ConversationBranches[branchID]
	if !ok || item.UserID != user.ID || !branchOK || branch.ConversationID != conversationID { return Conversation{}, errors.New("branch_not_found") }
	item.ActiveBranchID, item.UpdatedAt = branchID, time.Now().UTC(); s.data.Conversations[conversationID] = item; return item, s.persistLocked()
}

func (s *Store) RerunMessage(access, messageID, modelID, profileID string) (MessageRun, error) {
	if s.conversationSQL != nil { return s.conversationSQL.rerunMessage(access, messageID, modelID, profileID) }
	s.mu.Lock(); defer s.mu.Unlock()
	user, _, err := s.authenticatedUserLocked(access); if err != nil { return MessageRun{}, err }
	message, ok := s.data.Messages[messageID]; if !ok || message.UserID != user.ID { return MessageRun{}, errors.New("message_not_found") }
	conversation, ok := s.data.Conversations[message.ConversationID]; if !ok || conversation.DeletedAt != nil { return MessageRun{}, errors.New("conversation_not_found") }
	prompt, source := message.Body, message
	if message.Role == "assistant" {
		if message.ParentMessageID == "" { return MessageRun{}, errors.New("rerun_source_not_found") }
		parent, found := s.data.Messages[message.ParentMessageID]
		if !found || parent.UserID != user.ID || parent.Role != "user" { return MessageRun{}, errors.New("rerun_source_not_found") }
		prompt, source = parent.Body, parent
	}
	selection, err := s.effectiveSelectionLocked(user.ID, conversation.ID, modelID, profileID)
	if err != nil { return MessageRun{}, err }
	newBranchID, err := randomToken(16); if err != nil { return MessageRun{}, err }
	throughMessageID := s.previousBranchMessageIDLocked(conversation.ID, source.BranchID, source.Sequence)
	if err = s.cloneMemoryBranchHistoryLocked(conversation.ID, source.BranchID, newBranchID, throughMessageID); err != nil { return MessageRun{}, err }
	now := time.Now().UTC()
	s.data.ConversationBranches[newBranchID] = ConversationBranch{ID: newBranchID, ConversationID: conversation.ID, ParentBranchID: source.BranchID, ForkedFromMessageID: message.ID, Status: "active", CreatedAt: now, UpdatedAt: now}
	conversation.ActiveBranchID, conversation.UpdatedAt = newBranchID, now
	s.data.Conversations[conversation.ID] = conversation
	run, _, err := s.startRunLocked(user, conversation, prompt, selection, "rerun-"+newBranchID, "message_rerun")
	return run, err
}

func (s *Store) CreateComparison(access, conversationID, prompt string, models []string, profile string) (ComparisonGroup, error) {
	if s.conversationSQL != nil { return s.conversationSQL.createComparison(access, conversationID, prompt, models, profile) }
	s.mu.Lock(); defer s.mu.Unlock(); user, _, err := s.authenticatedUserLocked(access); if err != nil { return ComparisonGroup{}, err }
	conversation, ok := s.data.Conversations[conversationID]; if !ok || conversation.UserID != user.ID || conversation.DeletedAt != nil { return ComparisonGroup{}, errors.New("conversation_not_found") }
	prompt = strings.TrimSpace(prompt); if prompt == "" { return ComparisonGroup{}, errors.New("message_body_required") }; if len([]rune(prompt)) > 200000 { return ComparisonGroup{}, errors.New("message_body_too_long") }
	if len(models) == 0 { models = []string{"ylven-default"} }; if len(models) > 6 { return ComparisonGroup{}, errors.New("comparison_model_limit_exceeded") }
	uniqueModelCount := 0
	seenModels := map[string]bool{}
	for _, requestedModelID := range models {
		requestedModelID = strings.TrimSpace(requestedModelID)
		if requestedModelID != "" && !seenModels[requestedModelID] { seenModels[requestedModelID] = true; uniqueModelCount++ }
	}
	if uniqueModelCount < 2 { return ComparisonGroup{}, errors.New("comparison_requires_multiple_models") }
	id, err := randomToken(16); if err != nil { return ComparisonGroup{}, err }
	now := time.Now().UTC(); group := ComparisonGroup{ID:id, ConversationID:conversationID, UserID:user.ID, Prompt:prompt, Status:"queued", Candidates:[]ComparisonCandidate{}, CreatedAt:now, UpdatedAt:now}
	s.data.ComparisonGroups[id] = group
	seen := map[string]bool{}
	for _, requestedModelID := range models {
		requestedModelID = strings.TrimSpace(requestedModelID); if requestedModelID == "" || seen[requestedModelID] { continue }; seen[requestedModelID] = true
		selection, selectErr := s.resolveModelSelectionLocked(requestedModelID, profile)
		if selectErr != nil { group.Candidates = append(group.Candidates, ComparisonCandidate{ModelID:requestedModelID, ReasoningProfile:strings.TrimSpace(profile), Status:"unavailable", ErrorCode:selectErr.Error()}); continue }
		branchID, branchErr := randomToken(16); if branchErr != nil { return ComparisonGroup{}, branchErr }
		forkedFromMessageID := s.lastBranchMessageIDLocked(conversation.ID, conversation.ActiveBranchID)
		if branchErr = s.cloneMemoryBranchHistoryLocked(conversation.ID, conversation.ActiveBranchID, branchID, forkedFromMessageID); branchErr != nil { return ComparisonGroup{}, branchErr }
		s.data.ConversationBranches[branchID] = ConversationBranch{ID:branchID, ConversationID:conversation.ID, ParentBranchID:conversation.ActiveBranchID, ForkedFromMessageID:forkedFromMessageID, Status:"comparison", CreatedAt:now, UpdatedAt:now}
		candidateConversation := conversation; candidateConversation.ActiveBranchID = branchID
		run, _, runErr := s.startRunLocked(user, candidateConversation, prompt, selection, "comparison-"+id+"-"+selection.CatalogModelID, "comparison_run")
		if runErr != nil { return ComparisonGroup{}, runErr }
		s.data.Conversations[conversation.ID] = conversation
		message := s.data.Messages[run.UserMessageID]; message.ComparisonGroupID = id; s.data.Messages[message.ID] = message
		group.Candidates = append(group.Candidates, ComparisonCandidate{RunID:run.ID, ModelID:selection.CatalogModelID, ReasoningProfile:selection.ReasoningProfile, Status:run.Status, MessageID:run.UserMessageID})
	}
	group.Status = comparisonAggregateStatus(group.Candidates); group.UpdatedAt = time.Now().UTC(); s.data.Conversations[conversation.ID] = conversation; s.data.ComparisonGroups[id] = group
	return group, s.persistLocked()
}

func (s *Store) GetComparison(access, comparisonID string) (ComparisonGroup, error) {
	if s.conversationSQL != nil { return s.conversationSQL.getComparison(access, comparisonID) }
	s.mu.Lock(); defer s.mu.Unlock(); user, _, err := s.authenticatedUserLocked(access); if err != nil { return ComparisonGroup{}, err }; group, ok := s.data.ComparisonGroups[comparisonID]; if !ok || group.UserID != user.ID { return ComparisonGroup{}, errors.New("comparison_not_found") }; s.hydrateMemoryComparisonLocked(&group); return group, nil
}

func (s *Store) AdoptComparison(access, comparisonID, runID string) (ComparisonGroup, error) {
	if s.conversationSQL != nil { return s.conversationSQL.adoptComparison(access, comparisonID, runID) }
	s.mu.Lock(); defer s.mu.Unlock(); user, _, err := s.authenticatedUserLocked(access); if err != nil { return ComparisonGroup{}, err }; group, ok := s.data.ComparisonGroups[comparisonID]; if !ok || group.UserID != user.ID { return ComparisonGroup{}, errors.New("comparison_not_found") }
	run, exists := s.data.Runs[runID]; if !exists || run.UserID != user.ID || run.ConversationID != group.ConversationID { return ComparisonGroup{}, errors.New("comparison_candidate_not_found") }
	found := false; for _, candidate := range group.Candidates { if candidate.RunID == runID { found = true; break } }; if !found { return ComparisonGroup{}, errors.New("comparison_candidate_not_found") }
	if run.Status != "completed" || run.AssistantMessageID == "" { return ComparisonGroup{}, errors.New("comparison_candidate_not_ready") }
	conversation := s.data.Conversations[group.ConversationID]; conversation.ActiveBranchID, conversation.UpdatedAt = run.BranchID, time.Now().UTC(); s.data.Conversations[conversation.ID] = conversation
	group.AdoptedRunID, group.Status, group.UpdatedAt = runID, "adopted", conversation.UpdatedAt; s.hydrateMemoryComparisonLocked(&group); s.data.ComparisonGroups[comparisonID] = group; return group, s.persistLocked()
}

func (s *Store) SynthesizeComparison(access, comparisonID string, runIDs []string) (ComparisonGroup, error) {
	if s.conversationSQL != nil { return s.conversationSQL.synthesizeComparison(access, comparisonID, runIDs) }
	s.mu.Lock(); defer s.mu.Unlock(); user, _, err := s.authenticatedUserLocked(access); if err != nil { return ComparisonGroup{}, err }; group, ok := s.data.ComparisonGroups[comparisonID]; if !ok || group.UserID != user.ID { return ComparisonGroup{}, errors.New("comparison_not_found") }
	s.hydrateMemoryComparisonLocked(&group)
	requested := map[string]bool{}; for _, runID := range runIDs { if id := strings.TrimSpace(runID); id != "" { requested[id] = true } }; if len(requested) == 0 { for _, candidate := range group.Candidates { requested[candidate.RunID] = true } }
	sections := []string{}
	for _, candidate := range group.Candidates { if !requested[candidate.RunID] { continue }; run, exists := s.data.Runs[candidate.RunID]; if !exists || run.Status != "completed" || run.AssistantMessageID == "" { return ComparisonGroup{}, errors.New("comparison_candidate_not_ready") }; answer := s.data.Messages[run.AssistantMessageID]; sections = append(sections, "["+candidate.ModelID+"]\n"+answer.Body) }
	if len(sections) == 0 { return ComparisonGroup{}, errors.New("comparison_candidate_not_found") }
	conversation := s.data.Conversations[group.ConversationID]; selection, err := s.effectiveSelectionLocked(user.ID, conversation.ID, "", ""); if err != nil { return ComparisonGroup{}, err }
	prompt := "Synthesize the candidate answers below into one accurate answer. State uncertainty where candidates disagree. Do not mention hidden reasoning.\n\nOriginal request:\n"+group.Prompt+"\n\nCandidates:\n"+strings.Join(sections, "\n\n")
	run, _, err := s.startRunLocked(user, conversation, prompt, selection, "synthesis-"+group.ID, "comparison_synthesis"); if err != nil { return ComparisonGroup{}, err }
	message := s.data.Messages[run.UserMessageID]; message.ComparisonGroupID = group.ID; s.data.Messages[message.ID] = message
	group.SynthesisRunID, group.Status, group.UpdatedAt = run.ID, "synthesizing", time.Now().UTC(); s.data.ComparisonGroups[group.ID] = group
	return group, s.persistLocked()
}

func (s *Store) ModelAvailability(access, modelID string) (map[string]any, error) {
	if s.conversationSQL != nil { return s.conversationSQL.modelAvailability(access, modelID) }
	s.mu.Lock(); defer s.mu.Unlock(); if _, _, err := s.authenticatedUserLocked(access); err != nil { return nil, err }
	modelID = strings.TrimSpace(modelID); requested := false; health, ok := s.data.ModelHealth[modelID]
	fallbacks := []ModelCatalogEntry{}
	for _, raw := range s.data.ModelCatalog { model := normalizeCatalogEntry(raw); if model.ID == modelID { requested = true; if !ok { health = ModelHealthStatus{ModelID:model.ID, ProviderID:model.ProviderID, Status:"available", Capabilities:[]string{"text"}} }; continue }; candidateHealth, hasHealth := s.data.ModelHealth[model.ID]; if model.Enabled && (!hasHealth || candidateHealth.Status == "available" || candidateHealth.Status == "degraded") { fallbacks = append(fallbacks, model) } }
	if !requested { return nil, errors.New("model_not_found") }
	return map[string]any{"model_id":modelID, "status":health.Status, "health":health, "fallback_models":fallbacks}, nil
}

func (s *Store) RecordModelHealth(access, modelID, status, errorCode string, latency int64, capabilities []string) (ModelHealthStatus, error) {
	if s.conversationSQL != nil { return s.conversationSQL.recordModelHealth(access, modelID, status, errorCode, latency, capabilities) }
	s.mu.Lock(); defer s.mu.Unlock(); user, _, err := s.authenticatedUserLocked(access); if err != nil { return ModelHealthStatus{}, err }; _ = user; if status != "available" && status != "degraded" && status != "unavailable" { return ModelHealthStatus{}, errors.New("health_status_invalid") }; providerID := ""; for _, model := range s.data.ModelCatalog { if model.ID == modelID { providerID = model.ProviderID; break } }; value := ModelHealthStatus{ModelID:modelID, ProviderID:providerID, Status:status, ErrorCode:errorCode, LatencyMs:latency, Capabilities:append([]string(nil), capabilities...), LastProbeAt:time.Now().UTC()}; s.data.ModelHealth[modelID] = value; return value, s.persistLocked()
}

func (s *Store) RecordUsage(access string, event UsageEvent) (UsageEvent, error) {
	if s.conversationSQL != nil { return s.conversationSQL.recordUsage(access, event) }
	s.mu.Lock(); defer s.mu.Unlock(); user, _, err := s.authenticatedUserLocked(access); if err != nil { return UsageEvent{}, err }; event.UserID = user.ID; if event.ID == "" { event.ID, _ = randomToken(16) }; if event.CreatedAt.IsZero() { event.CreatedAt = time.Now().UTC() }; if _, exists := s.data.UsageEvents[event.ID]; exists { return UsageEvent{}, errors.New("usage_event_already_recorded") }; for _, existing := range s.data.UsageEvents { if event.RunID != "" && existing.RunID == event.RunID { return UsageEvent{}, errors.New("usage_event_already_recorded") } }; s.data.UsageEvents[event.ID] = event; return event, s.persistLocked()
}

func (s *Store) UpsertPriceSnapshot(access string, value PriceSnapshot) (PriceSnapshot, error) {
	if s.conversationSQL != nil { return s.conversationSQL.upsertPriceSnapshot(access, value) }
	s.mu.Lock(); defer s.mu.Unlock(); if _, _, err := s.authenticatedUserLocked(access); err != nil { return PriceSnapshot{}, err }; if value.ID == "" { value.ID, _ = randomToken(16) }; if value.Version == 0 { value.Version = 1 }; if value.EffectiveAt.IsZero() { value.EffectiveAt = time.Now().UTC() }; s.data.PriceSnapshots[value.ID] = value; return value, s.persistLocked()
}

func (s *Store) ServiceStatus(access string) ([]ModelHealthStatus, error) {
	if s.conversationSQL != nil { return s.conversationSQL.serviceStatus(access) }
	s.mu.Lock(); defer s.mu.Unlock(); if _, _, err := s.authenticatedUserLocked(access); err != nil { return nil, err }; result := make([]ModelHealthStatus, 0, len(s.data.ModelCatalog)); for _, model := range s.data.ModelCatalog { if !model.Enabled { continue }; value, ok := s.data.ModelHealth[model.ID]; if !ok { value = ModelHealthStatus{ModelID:model.ID, ProviderID:model.ProviderID, Status:"available", Capabilities:[]string{"text"}} }; result = append(result, value) }; sort.Slice(result, func(i,j int) bool { return result[i].ModelID < result[j].ModelID }); return result, nil
}

func (s *Store) lastBranchMessageIDLocked(conversationID, branchID string) string {
	var result string
	var sequence int64 = -1
	for _, message := range s.data.Messages { if message.ConversationID == conversationID && message.BranchID == branchID && message.Sequence > sequence { result, sequence = message.ID, message.Sequence } }
	return result
}

func (s *Store) previousBranchMessageIDLocked(conversationID, branchID string, beforeSequence int64) string {
	var result string
	var sequence int64 = -1
	for _, message := range s.data.Messages { if message.ConversationID == conversationID && message.BranchID == branchID && message.Sequence < beforeSequence && message.Sequence > sequence { result, sequence = message.ID, message.Sequence } }
	return result
}

func (s *Store) cloneMemoryBranchHistoryLocked(conversationID, sourceBranchID, targetBranchID, throughMessageID string) error {
	if strings.TrimSpace(throughMessageID) == "" { return nil }
	through, ok := s.data.Messages[throughMessageID]
	if !ok || through.ConversationID != conversationID || through.BranchID != sourceBranchID { return errors.New("message_not_found") }
	items := []Message{}
	for _, message := range s.data.Messages { if message.ConversationID == conversationID && message.BranchID == sourceBranchID && message.Sequence <= through.Sequence { items = append(items, message) } }
	sort.Slice(items, func(i, j int) bool { return items[i].Sequence < items[j].Sequence })
	mapping := map[string]string{}
	for _, source := range items {
		id, err := randomToken(16); if err != nil { return err }
		cloned := source; cloned.ID, cloned.BranchID, cloned.ParentMessageID = id, targetBranchID, mapping[source.ParentMessageID]
		s.data.Messages[id] = cloned
		if part, exists := s.data.MessageParts[source.ID]; exists { partID, partErr := randomToken(16); if partErr != nil { return partErr }; clonedPart := part; clonedPart.ID, clonedPart.MessageID = partID, id; s.data.MessageParts[id] = clonedPart }
		mapping[source.ID] = id
	}
	return nil
}

func (s *Store) hydrateMemoryComparisonLocked(group *ComparisonGroup) {
	for index := range group.Candidates { candidate := &group.Candidates[index]; run, ok := s.data.Runs[candidate.RunID]; if !ok { continue }; candidate.Status, candidate.ErrorCode, candidate.MessageID = run.Status, run.ErrorCode, run.AssistantMessageID; if run.AssistantMessageID != "" { candidate.Body = s.data.Messages[run.AssistantMessageID].Body } }
	if group.SynthesisRunID != "" { if run, ok := s.data.Runs[group.SynthesisRunID]; ok { switch run.Status { case "completed": group.Status = "synthesized"; case "failed", "cancelled": group.Status = "synthesis_failed"; default: group.Status = "synthesizing" }; return } }
	if group.AdoptedRunID == "" { group.Status = comparisonAggregateStatus(group.Candidates) }
}

func (s *Store) refreshMemoryComparisonForRunLocked(run MessageRun) {
	message, ok := s.data.Messages[run.UserMessageID]; if !ok || message.ComparisonGroupID == "" { return }
	group, ok := s.data.ComparisonGroups[message.ComparisonGroupID]; if !ok { return }
	s.hydrateMemoryComparisonLocked(&group); group.UpdatedAt = time.Now().UTC(); s.data.ComparisonGroups[group.ID] = group
}
