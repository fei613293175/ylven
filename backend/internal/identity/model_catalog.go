package identity

import (
	"errors"
	"regexp"
	"sort"
	"strings"
	"time"
)

var modelCatalogIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

type ModelProvider struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Enabled   bool      `json:"enabled"`
	SortOrder int       `json:"sort_order"`
	Version   int64     `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ModelCapabilityTag struct {
	ID       string    `json:"id"`
	Label    string    `json:"label"`
	ProbedAt time.Time `json:"probed_at"`
}

type CapabilityProbeResult struct {
	ID                string     `json:"id"`
	ModelID           string     `json:"model_id"`
	CapabilityID      string     `json:"capability_id"`
	CapabilityLabel   string     `json:"capability_label,omitempty"`
	Status            string     `json:"status"`
	ProbeSource       string     `json:"probe_source"`
	EvidenceReference string     `json:"evidence_reference"`
	ProbedAt          time.Time  `json:"probed_at"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	RecordedBy        string     `json:"recorded_by"`
	CreatedAt         time.Time  `json:"created_at"`
}

type ReasoningProfileMapping struct {
	ModelID            string         `json:"model_id"`
	ProfileID          string         `json:"profile_id"`
	Label              string         `json:"label"`
	Ordinal            int            `json:"ordinal"`
	Enabled            bool           `json:"enabled"`
	UpstreamParameters map[string]any `json:"upstream_parameters"`
	Version            int64          `json:"version"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

type ModelCatalogEntry struct {
	ID                string                    `json:"id"`
	Name              string                    `json:"name"`
	Enabled           bool                      `json:"enabled"`
	Description       string                    `json:"description,omitempty"`
	ProviderID        string                    `json:"provider_id"`
	ProviderName      string                    `json:"provider_name"`
	Purpose           string                    `json:"purpose"`
	SpeedTier         string                    `json:"speed_tier"`
	Capabilities      []ModelCapabilityTag      `json:"capabilities"`
	ReasoningProfiles []string                  `json:"reasoning_profiles"`
	UpstreamModel     string                    `json:"-"`
	SortOrder         int                       `json:"sort_order"`
	Version           int64                     `json:"version"`
	UpdatedAt         time.Time                 `json:"updated_at"`
	ReasoningMappings []ReasoningProfileMapping `json:"-"`
}

type ModelProviderGroup struct {
	Provider ModelProvider       `json:"provider"`
	Models   []ModelCatalogEntry `json:"models"`
}

type ModelCatalogAuditEvent struct {
	ID           string         `json:"id"`
	ActorID      string         `json:"actor_id"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resource_type"`
	ResourceID   string         `json:"resource_id"`
	BeforeValue  map[string]any `json:"before_value,omitempty"`
	AfterValue   map[string]any `json:"after_value,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

type ModelCatalogSnapshot struct {
	Providers         []ModelProvider           `json:"providers"`
	Models            []ModelCatalogEntry       `json:"models"`
	CapabilityProbes  []CapabilityProbeResult   `json:"capability_probes"`
	ReasoningMappings []ReasoningProfileMapping `json:"reasoning_mappings"`
	Audit             []ModelCatalogAuditEvent  `json:"audit"`
}

type AdminModelCatalogEntry struct {
	ModelCatalogEntry
	UpstreamModel string `json:"upstream_model"`
}

func adminModelCatalogEntries(items []ModelCatalogEntry) []AdminModelCatalogEntry {
	result := make([]AdminModelCatalogEntry, 0, len(items))
	for _, item := range items {
		result = append(result, AdminModelCatalogEntry{ModelCatalogEntry: item, UpstreamModel: item.UpstreamModel})
	}
	return result
}

type ModelSelection struct {
	CatalogModelID      string
	ProviderModel       string
	ReasoningProfile    string
	ReasoningParameters map[string]any
}

func defaultModelProviders() []ModelProvider {
	now := time.Now().UTC()
	return []ModelProvider{{ID: "ylven", Name: "YLVEN", Enabled: true, Version: 1, UpdatedAt: now}}
}

func defaultModelCatalog() []ModelCatalogEntry {
	now := time.Now().UTC()
	return []ModelCatalogEntry{{
		ID: "ylven-default", Name: "YLVEN 默认模型", Enabled: true, ProviderID: "ylven", ProviderName: "YLVEN",
		UpstreamModel: "ylven-default", Purpose: "自动匹配当前可用能力", SpeedTier: "balanced",
		ReasoningProfiles: []string{"auto"}, Version: 1, UpdatedAt: now,
		ReasoningMappings: []ReasoningProfileMapping{{ModelID: "ylven-default", ProfileID: "auto", Label: "自动", Enabled: true, UpstreamParameters: map[string]any{}, Version: 1, UpdatedAt: now}},
	}}
}

func copyJSONMap(source map[string]any) map[string]any {
	if source == nil {
		return map[string]any{}
	}
	copyValue := make(map[string]any, len(source))
	for key, value := range source {
		copyValue[key] = value
	}
	return copyValue
}

func normalizeCatalogEntry(item ModelCatalogEntry) ModelCatalogEntry {
	item.ID = strings.TrimSpace(item.ID)
	item.Name = strings.TrimSpace(item.Name)
	item.Description = strings.TrimSpace(item.Description)
	item.ProviderID = strings.TrimSpace(item.ProviderID)
	item.ProviderName = strings.TrimSpace(item.ProviderName)
	item.Purpose = strings.TrimSpace(item.Purpose)
	item.UpstreamModel = strings.TrimSpace(item.UpstreamModel)
	if item.ProviderID == "" {
		item.ProviderID = "ylven"
	}
	if item.ProviderName == "" {
		item.ProviderName = item.ProviderID
	}
	if item.UpstreamModel == "" {
		item.UpstreamModel = item.ID
	}
	switch item.SpeedTier {
	case "fast", "balanced", "deliberate":
	default:
		item.SpeedTier = "balanced"
	}
	if item.Version <= 0 {
		item.Version = 1
	}
	if item.UpdatedAt.IsZero() {
		item.UpdatedAt = time.Now().UTC()
	}
	if len(item.ReasoningProfiles) == 0 {
		item.ReasoningProfiles = []string{"auto"}
	}
	if len(item.ReasoningMappings) == 0 {
		for ordinal, profile := range item.ReasoningProfiles {
			item.ReasoningMappings = append(item.ReasoningMappings, ReasoningProfileMapping{ModelID: item.ID, ProfileID: profile, Label: profile, Ordinal: ordinal, Enabled: true, UpstreamParameters: map[string]any{}, Version: 1, UpdatedAt: item.UpdatedAt})
		}
	}
	return item
}

func validateUpstreamReasoningParameters(parameters map[string]any) error {
	allowed := map[string]bool{"reasoning_effort": true, "reasoning": true, "thinking": true}
	for key := range parameters {
		if !allowed[key] {
			return errors.New("reasoning_parameter_not_allowed")
		}
	}
	return nil
}

func (s *Store) MobileModelDetail(access, modelID string) (ModelCatalogEntry, error) {
	items, err := s.MobileModelCatalog(access)
	if err != nil {
		return ModelCatalogEntry{}, err
	}
	for _, item := range items {
		if item.ID == strings.TrimSpace(modelID) {
			return item, nil
		}
	}
	return ModelCatalogEntry{}, errors.New("model_not_found")
}

func (s *Store) MobileReasoningProfiles(access, modelID string) ([]ReasoningProfileMapping, error) {
	if _, err := s.authenticatedUser(access); err != nil {
		return nil, err
	}
	if s.conversationSQL != nil {
		return s.conversationSQL.reasoningProfiles(modelID, true)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, raw := range s.data.ModelCatalog {
		item := normalizeCatalogEntry(raw)
		if item.ID != strings.TrimSpace(modelID) || !item.Enabled {
			continue
		}
		profiles := make([]ReasoningProfileMapping, 0, len(item.ReasoningMappings))
		for _, profile := range item.ReasoningMappings {
			if profile.Enabled {
				profile.UpstreamParameters = nil
				profiles = append(profiles, profile)
			}
		}
		return profiles, nil
	}
	return nil, errors.New("model_not_found")
}

func (s *Store) ResolveModelSelection(modelID, profileID string) (ModelSelection, error) {
	if s.conversationSQL != nil {
		return s.conversationSQL.resolveModelSelection(modelID, profileID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		modelID = "ylven-default"
	}
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		profileID = "auto"
	}
	providers := map[string]bool{}
	for _, provider := range s.data.ModelProviders {
		providers[provider.ID] = provider.Enabled
	}
	for _, raw := range s.data.ModelCatalog {
		item := normalizeCatalogEntry(raw)
		if item.ID != modelID {
			continue
		}
		if !item.Enabled || !providers[item.ProviderID] {
			return ModelSelection{}, errors.New("model_unavailable")
		}
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
	return ModelSelection{}, errors.New("model_not_found")
}

func (s *Store) ModelCatalogSnapshot() (ModelCatalogSnapshot, error) {
	if s.conversationSQL != nil {
		return s.conversationSQL.modelCatalogSnapshot()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	models := make([]ModelCatalogEntry, 0, len(s.data.ModelCatalog))
	mappings := []ReasoningProfileMapping{}
	for _, raw := range s.data.ModelCatalog {
		item := normalizeCatalogEntry(raw)
		mappings = append(mappings, item.ReasoningMappings...)
		item.ReasoningMappings = nil
		models = append(models, item)
	}
	return ModelCatalogSnapshot{
		Providers: append([]ModelProvider(nil), s.data.ModelProviders...), Models: models,
		CapabilityProbes:  append([]CapabilityProbeResult(nil), s.data.CapabilityProbes...),
		ReasoningMappings: mappings, Audit: append([]ModelCatalogAuditEvent(nil), s.data.ModelCatalogAudit...),
	}, nil
}

func groupModelsByProvider(items []ModelCatalogEntry, providers []ModelProvider) []ModelProviderGroup {
	providerByID := map[string]ModelProvider{}
	for _, provider := range providers {
		providerByID[provider.ID] = provider
	}
	groups := map[string][]ModelCatalogEntry{}
	providerOrder := make([]string, 0)
	for _, item := range items {
		if _, exists := groups[item.ProviderID]; !exists {
			providerOrder = append(providerOrder, item.ProviderID)
		}
		groups[item.ProviderID] = append(groups[item.ProviderID], item)
	}
	result := make([]ModelProviderGroup, 0, len(groups))
	for _, providerID := range providerOrder {
		models := groups[providerID]
		provider, ok := providerByID[providerID]
		if !ok {
			provider = ModelProvider{ID: providerID, Name: models[0].ProviderName, Enabled: true}
		}
		result = append(result, ModelProviderGroup{Provider: provider, Models: models})
	}
	if len(providers) == 0 {
		return result
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Provider.SortOrder == result[j].Provider.SortOrder {
			return result[i].Provider.ID < result[j].Provider.ID
		}
		return result[i].Provider.SortOrder < result[j].Provider.SortOrder
	})
	return result
}

func (s *Store) UpsertModelProvider(input ModelProvider, actorID string) (ModelProvider, error) {
	input.ID = strings.TrimSpace(input.ID)
	input.Name = strings.TrimSpace(input.Name)
	if !modelCatalogIDPattern.MatchString(input.ID) || input.Name == "" || strings.TrimSpace(actorID) == "" {
		return ModelProvider{}, errors.New("model_provider_invalid")
	}
	if s.conversationSQL != nil {
		return s.conversationSQL.upsertModelProvider(input, actorID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for index, current := range s.data.ModelProviders {
		if current.ID != input.ID {
			continue
		}
		if input.Version != current.Version {
			return ModelProvider{}, errors.New("version_conflict")
		}
		before := current
		input.Version++
		input.UpdatedAt = now
		s.data.ModelProviders[index] = input
		s.appendModelCatalogAuditLocked(actorID, "update", "provider", input.ID, map[string]any{"name": before.Name, "enabled": before.Enabled, "version": before.Version}, map[string]any{"name": input.Name, "enabled": input.Enabled, "version": input.Version})
		return input, s.persistLocked()
	}
	if input.Version != 0 && input.Version != 1 {
		return ModelProvider{}, errors.New("version_conflict")
	}
	input.Version = 1
	input.UpdatedAt = now
	s.data.ModelProviders = append(s.data.ModelProviders, input)
	s.appendModelCatalogAuditLocked(actorID, "create", "provider", input.ID, nil, map[string]any{"name": input.Name, "enabled": input.Enabled, "version": input.Version})
	return input, s.persistLocked()
}

func (s *Store) UpsertModelCatalogEntry(input ModelCatalogEntry, actorID string) (ModelCatalogEntry, error) {
	input = normalizeCatalogEntry(input)
	if !modelCatalogIDPattern.MatchString(input.ID) || input.Name == "" || !modelCatalogIDPattern.MatchString(input.ProviderID) || input.UpstreamModel == "" || strings.TrimSpace(actorID) == "" {
		return ModelCatalogEntry{}, errors.New("model_catalog_entry_invalid")
	}
	if s.conversationSQL != nil {
		return s.conversationSQL.upsertModelCatalogEntry(input, actorID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	providerFound := false
	for _, provider := range s.data.ModelProviders {
		providerFound = providerFound || provider.ID == input.ProviderID
	}
	if !providerFound {
		return ModelCatalogEntry{}, errors.New("model_provider_not_found")
	}
	now := time.Now().UTC()
	for index, current := range s.data.ModelCatalog {
		current = normalizeCatalogEntry(current)
		if current.ID != input.ID {
			continue
		}
		if input.Version != current.Version {
			return ModelCatalogEntry{}, errors.New("version_conflict")
		}
		input.Version++
		input.UpdatedAt = now
		input.ReasoningProfiles = current.ReasoningProfiles
		input.ReasoningMappings = current.ReasoningMappings
		s.data.ModelCatalog[index] = input
		s.appendModelCatalogAuditLocked(actorID, "update", "model", input.ID, map[string]any{"name": current.Name, "enabled": current.Enabled, "version": current.Version}, map[string]any{"name": input.Name, "enabled": input.Enabled, "version": input.Version})
		return input, s.persistLocked()
	}
	if input.Version != 0 && input.Version != 1 {
		return ModelCatalogEntry{}, errors.New("version_conflict")
	}
	input.Version = 1
	input.UpdatedAt = now
	input.ReasoningProfiles = []string{"auto"}
	input.ReasoningMappings = []ReasoningProfileMapping{{ModelID: input.ID, ProfileID: "auto", Label: "自动", Enabled: true, UpstreamParameters: map[string]any{}, Version: 1, UpdatedAt: now}}
	s.data.ModelCatalog = append(s.data.ModelCatalog, input)
	s.appendModelCatalogAuditLocked(actorID, "create", "model", input.ID, nil, map[string]any{"name": input.Name, "enabled": input.Enabled, "version": input.Version})
	return input, s.persistLocked()
}

func (s *Store) RecordCapabilityProbe(input CapabilityProbeResult, actorID string) (CapabilityProbeResult, error) {
	input.ModelID = strings.TrimSpace(input.ModelID)
	input.CapabilityID = strings.TrimSpace(input.CapabilityID)
	input.ProbeSource = strings.TrimSpace(input.ProbeSource)
	input.EvidenceReference = strings.TrimSpace(input.EvidenceReference)
	if input.Status != "passed" && input.Status != "failed" {
		return CapabilityProbeResult{}, errors.New("capability_probe_status_invalid")
	}
	if !modelCatalogIDPattern.MatchString(input.ModelID) || !modelCatalogIDPattern.MatchString(input.CapabilityID) || input.ProbeSource == "" || input.EvidenceReference == "" || input.ProbedAt.IsZero() || input.ProbedAt.After(time.Now().UTC().Add(5*time.Minute)) || strings.TrimSpace(actorID) == "" {
		return CapabilityProbeResult{}, errors.New("capability_probe_invalid")
	}
	if s.conversationSQL != nil {
		return s.conversationSQL.recordCapabilityProbe(input, actorID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	modelFound := false
	for _, model := range s.data.ModelCatalog {
		modelFound = modelFound || model.ID == input.ModelID
	}
	if !modelFound {
		return CapabilityProbeResult{}, errors.New("model_not_found")
	}
	id, err := randomToken(16)
	if err != nil {
		return CapabilityProbeResult{}, err
	}
	input.ID = id
	input.RecordedBy = actorID
	input.CreatedAt = time.Now().UTC()
	input.CapabilityLabel = input.CapabilityID
	s.data.CapabilityProbes = append(s.data.CapabilityProbes, input)
	s.appendModelCatalogAuditLocked(actorID, "record_probe", "capability", input.ModelID+":"+input.CapabilityID, nil, map[string]any{"status": input.Status, "evidence_reference": input.EvidenceReference, "probed_at": input.ProbedAt})
	return input, s.persistLocked()
}

func (s *Store) UpsertReasoningProfile(input ReasoningProfileMapping, actorID string) (ReasoningProfileMapping, error) {
	input.ModelID = strings.TrimSpace(input.ModelID)
	input.ProfileID = strings.TrimSpace(input.ProfileID)
	input.Label = strings.TrimSpace(input.Label)
	validProfile := input.ProfileID == "auto" || input.ProfileID == "quick" || input.ProfileID == "standard" || input.ProfileID == "deep"
	if !modelCatalogIDPattern.MatchString(input.ModelID) || !validProfile || input.Label == "" || strings.TrimSpace(actorID) == "" || validateUpstreamReasoningParameters(input.UpstreamParameters) != nil {
		return ReasoningProfileMapping{}, errors.New("reasoning_profile_invalid")
	}
	if s.conversationSQL != nil {
		return s.conversationSQL.upsertReasoningProfile(input, actorID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for modelIndex, raw := range s.data.ModelCatalog {
		model := normalizeCatalogEntry(raw)
		if model.ID != input.ModelID {
			continue
		}
		for profileIndex, current := range model.ReasoningMappings {
			if current.ProfileID != input.ProfileID {
				continue
			}
			if input.Version != current.Version {
				return ReasoningProfileMapping{}, errors.New("version_conflict")
			}
			input.Version++
			input.UpdatedAt = now
			input.UpstreamParameters = copyJSONMap(input.UpstreamParameters)
			model.ReasoningMappings[profileIndex] = input
			model.ReasoningProfiles = enabledReasoningProfileIDs(model.ReasoningMappings)
			s.data.ModelCatalog[modelIndex] = model
			s.appendModelCatalogAuditLocked(actorID, "update", "reasoning_profile", input.ModelID+":"+input.ProfileID, map[string]any{"enabled": current.Enabled, "version": current.Version}, map[string]any{"enabled": input.Enabled, "version": input.Version})
			return input, s.persistLocked()
		}
		if input.Version != 0 && input.Version != 1 {
			return ReasoningProfileMapping{}, errors.New("version_conflict")
		}
		input.Version = 1
		input.UpdatedAt = now
		input.UpstreamParameters = copyJSONMap(input.UpstreamParameters)
		model.ReasoningMappings = append(model.ReasoningMappings, input)
		model.ReasoningProfiles = enabledReasoningProfileIDs(model.ReasoningMappings)
		s.data.ModelCatalog[modelIndex] = model
		s.appendModelCatalogAuditLocked(actorID, "create", "reasoning_profile", input.ModelID+":"+input.ProfileID, nil, map[string]any{"enabled": input.Enabled, "version": input.Version})
		return input, s.persistLocked()
	}
	return ReasoningProfileMapping{}, errors.New("model_not_found")
}

func enabledReasoningProfileIDs(mappings []ReasoningProfileMapping) []string {
	copyMappings := append([]ReasoningProfileMapping(nil), mappings...)
	sort.Slice(copyMappings, func(i, j int) bool { return copyMappings[i].Ordinal < copyMappings[j].Ordinal })
	profiles := []string{}
	for _, mapping := range copyMappings {
		if mapping.Enabled {
			profiles = append(profiles, mapping.ProfileID)
		}
	}
	return profiles
}

func (s *Store) appendModelCatalogAuditLocked(actorID, action, resourceType, resourceID string, before, after map[string]any) {
	id, _ := randomToken(16)
	s.data.ModelCatalogAudit = append([]ModelCatalogAuditEvent{{ID: id, ActorID: actorID, Action: action, ResourceType: resourceType, ResourceID: resourceID, BeforeValue: before, AfterValue: after, CreatedAt: time.Now().UTC()}}, s.data.ModelCatalogAudit...)
	if len(s.data.ModelCatalogAudit) > 200 {
		s.data.ModelCatalogAudit = s.data.ModelCatalogAudit[:200]
	}
}
