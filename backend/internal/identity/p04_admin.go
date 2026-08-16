package identity

import (
	"errors"
	"sort"
	"strings"
	"time"
)

const p04AdminListLimit = 500

// ListComparisonGroups is an operations-only projection used to investigate
// asynchronous comparison runs. It remains separate from the user-scoped
// mobile read API and is bounded to protect the administration surface.
func (s *Store) ListComparisonGroups() ([]ComparisonGroup, error) {
	if s.conversationSQL != nil {
		return s.conversationSQL.listComparisonGroups()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]ComparisonGroup, 0, len(s.data.ComparisonGroups))
	for _, item := range s.data.ComparisonGroups {
		s.hydrateMemoryComparisonLocked(&item)
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].UpdatedAt.After(items[j].UpdatedAt) })
	if len(items) > p04AdminListLimit { items = items[:p04AdminListLimit] }
	return items, nil
}

// ListUsageEvents is deliberately an administrator-only projection. Mobile
// callers may record their own usage through the runtime, but must never be
// able to enumerate another user's records.
func (s *Store) ListUsageEvents() ([]UsageEvent, error) {
	if s.conversationSQL != nil {
		return s.conversationSQL.listUsageEvents()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]UsageEvent, 0, len(s.data.UsageEvents))
	for _, item := range s.data.UsageEvents {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	if len(items) > p04AdminListLimit {
		items = items[:p04AdminListLimit]
	}
	return items, nil
}

func (s *Store) ListPriceSnapshots() ([]PriceSnapshot, error) {
	if s.conversationSQL != nil {
		return s.conversationSQL.listPriceSnapshots()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]PriceSnapshot, 0, len(s.data.PriceSnapshots))
	for _, item := range s.data.PriceSnapshots {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].EffectiveAt.Equal(items[j].EffectiveAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].EffectiveAt.After(items[j].EffectiveAt)
	})
	if len(items) > p04AdminListLimit {
		items = items[:p04AdminListLimit]
	}
	return items, nil
}

func (s *Store) ListProviderRuntimePolicies() ([]ProviderRuntimePolicy, error) {
	if s.conversationSQL != nil {
		return s.conversationSQL.listProviderRuntimePolicies()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]ProviderRuntimePolicy, 0, len(s.data.ProviderRuntimePolicies))
	for _, item := range s.data.ProviderRuntimePolicies {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ProviderID < items[j].ProviderID })
	return items, nil
}

func (s *Store) UpsertPriceSnapshotAdmin(input PriceSnapshot, actorID string) (PriceSnapshot, error) {
	input.ModelID = strings.TrimSpace(input.ModelID)
	actorID = strings.TrimSpace(actorID)
	if input.ModelID == "" || actorID == "" || input.InputPerMillion < 0 || input.OutputPerMillion < 0 || input.ReasoningPerMillion < 0 {
		return PriceSnapshot{}, errors.New("price_snapshot_invalid")
	}
	if s.conversationSQL != nil {
		return s.conversationSQL.upsertPriceSnapshotAdmin(input, actorID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	modelFound := false
	for _, model := range s.data.ModelCatalog {
		if model.ID == input.ModelID {
			modelFound = true
			break
		}
	}
	if !modelFound {
		return PriceSnapshot{}, errors.New("model_not_found")
	}
	if input.ID == "" {
		id, err := randomToken(16)
		if err != nil {
			return PriceSnapshot{}, err
		}
		input.ID = "price-" + id
	}
	current, found := s.data.PriceSnapshots[input.ID]
	if found {
		if input.Version != current.Version {
			return PriceSnapshot{}, errors.New("version_conflict")
		}
		input.Version = current.Version + 1
	} else if input.Version != 0 && input.Version != 1 {
		return PriceSnapshot{}, errors.New("version_conflict")
	} else {
		input.Version = 1
	}
	if input.EffectiveAt.IsZero() {
		input.EffectiveAt = time.Now().UTC()
	}
	s.data.PriceSnapshots[input.ID] = input
	s.appendAuditLocked("p04_price_snapshot_updated", actorID, input.ID)
	return input, s.persistLocked()
}

func (s *Store) ListModelHealthStatuses() ([]ModelHealthStatus, error) {
	if s.conversationSQL != nil {
		return s.conversationSQL.listModelHealthStatuses()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]ModelHealthStatus, 0, len(s.data.ModelCatalog))
	for _, model := range s.data.ModelCatalog {
		if !model.Enabled {
			continue
		}
		item, found := s.data.ModelHealth[model.ID]
		if !found {
			item = ModelHealthStatus{ModelID: model.ID, ProviderID: model.ProviderID, Status: "available", Capabilities: []string{"text"}}
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ModelID < items[j].ModelID })
	return items, nil
}
