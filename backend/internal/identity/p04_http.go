package identity

import (
	"errors"
	"net/http"
	"strings"
	"time"
)

func (a *API) reserveProviderRun(run MessageRun) (ProviderRuntimePolicy, func(bool), error) {
	policy, err := a.Store.ProviderRuntimePolicyForModel(run.Model)
	if err != nil {
		return ProviderRuntimePolicy{}, nil, err
	}
	if !policy.Enabled {
		return ProviderRuntimePolicy{}, nil, errors.New("provider_runtime_disabled")
	}
	now := time.Now().UTC()
	a.providerMu.Lock()
	defer a.providerMu.Unlock()
	if openUntil := a.providerOpenUntil[policy.ProviderID]; now.Before(openUntil) {
		return ProviderRuntimePolicy{}, nil, errors.New("provider_circuit_open")
	}
	if a.providerRuns[policy.ProviderID] >= policy.MaxConcurrency {
		return ProviderRuntimePolicy{}, nil, errors.New("provider_concurrency_limited")
	}
	a.providerRuns[policy.ProviderID]++
	released := false
	release := func(success bool) {
		a.providerMu.Lock()
		defer a.providerMu.Unlock()
		if released {
			return
		}
		released = true
		if a.providerRuns[policy.ProviderID] > 0 {
			a.providerRuns[policy.ProviderID]--
		}
		if success {
			a.providerFailures[policy.ProviderID] = 0
			return
		}
		a.providerFailures[policy.ProviderID]++
		if a.providerFailures[policy.ProviderID] >= policy.CircuitThreshold {
			a.providerOpenUntil[policy.ProviderID] = time.Now().UTC().Add(time.Duration(policy.CooldownSeconds) * time.Second)
			a.providerFailures[policy.ProviderID] = 0
		}
	}
	return policy, release, nil
}

func writeP04MobileError(w http.ResponseWriter, err error) {
	if err == nil { return }
	code := err.Error()
	switch code {
	case "session_invalid":
		writeError(w, http.StatusUnauthorized, code, "Session is invalid")
	case "conversation_not_found", "message_not_found", "branch_not_found", "comparison_not_found", "model_not_found", "run_not_found":
		writeError(w, http.StatusNotFound, code, "Requested resource was not found")
	case "version_conflict", "conversation_busy", "idempotency_conflict", "usage_event_already_recorded":
		writeError(w, http.StatusConflict, code, "The request conflicts with the current state")
	default:
		writeError(w, http.StatusUnprocessableEntity, code, "The request cannot be completed")
	}
}

func (a *API) mobileAIPreference(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		value, err := a.Store.GetAIPreference(bearer(r))
		if err != nil { writeP04MobileError(w, err); return }
		writeJSON(w, http.StatusOK, value)
	case http.MethodPut:
		var input AIPreference
		if !decode(r, &input) { writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON"); return }
		value, err := a.Store.PutAIPreference(bearer(r), input)
		if err != nil { writeP04MobileError(w, err); return }
		writeJSON(w, http.StatusOK, value)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET or PUT required")
	}
}

func (a *API) mobileComparisons(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) { return }
	var input struct {
		ConversationID   string   `json:"conversation_id"`
		Prompt           string   `json:"prompt"`
		ModelIDs         []string `json:"model_ids"`
		ReasoningProfile string   `json:"reasoning_profile"`
	}
	if !decode(r, &input) { writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON"); return }
	if !a.chatRuntimeAvailableForAnyModel(input.ModelIDs) { writeError(w, http.StatusServiceUnavailable, "chat_runtime_unavailable", "AI provider runtime is not configured"); return }
	group, err := a.Store.CreateComparison(bearer(r), input.ConversationID, input.Prompt, input.ModelIDs, input.ReasoningProfile)
	if err != nil { writeP04MobileError(w, err); return }
	access := bearer(r)
	for _, candidate := range group.Candidates {
		if candidate.RunID == "" { continue }
		if run, _, runErr := a.Store.Run(access, candidate.RunID); runErr == nil { a.dispatchRun(run, access, false) }
	}
	writeJSON(w, http.StatusAccepted, group)
}

func (a *API) mobileComparisonByID(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/mobile/v1/comparisons/"), "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" { writeError(w, http.StatusBadRequest, "comparison_id_required", "Comparison ID required"); return }
	comparisonID := parts[0]
	if len(parts) == 1 {
		if !requireMethod(w, r, http.MethodGet) { return }
		group, err := a.Store.GetComparison(bearer(r), comparisonID)
		if err != nil { writeP04MobileError(w, err); return }
		writeJSON(w, http.StatusOK, group)
		return
	}
	if len(parts) != 2 { writeError(w, http.StatusNotFound, "not_found", "Endpoint not found"); return }
	switch parts[1] {
	case "adopt":
		if !requireMethod(w, r, http.MethodPost) { return }
		var input struct { RunID string `json:"run_id"` }
		if !decode(r, &input) || strings.TrimSpace(input.RunID) == "" { writeError(w, http.StatusBadRequest, "run_id_required", "Run ID required"); return }
		group, err := a.Store.AdoptComparison(bearer(r), comparisonID, input.RunID)
		if err != nil { writeP04MobileError(w, err); return }
		writeJSON(w, http.StatusOK, group)
	case "synthesize":
		if !requireMethod(w, r, http.MethodPost) { return }
		var input struct { RunIDs []string `json:"run_ids"` }
		if !decode(r, &input) { writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON"); return }
		access := bearer(r)
		group, err := a.Store.SynthesizeComparison(access, comparisonID, input.RunIDs)
		if err != nil { writeP04MobileError(w, err); return }
		if group.SynthesisRunID != "" { if run, _, runErr := a.Store.Run(access, group.SynthesisRunID); runErr == nil { a.dispatchRun(run, access, false) } }
		writeJSON(w, http.StatusAccepted, group)
	default:
		writeError(w, http.StatusNotFound, "not_found", "Endpoint not found")
	}
}

func (a *API) mobileModelServiceStatus(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) { return }
	items, err := a.Store.ServiceStatus(bearer(r))
	if err != nil { writeP04MobileError(w, err); return }
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// publicModelServiceStatus intentionally returns only the simplified health
// projection. It never exposes credentials, provider endpoints, prompts, or
// detailed runtime errors that are available to administrators.
func (a *API) publicModelServiceStatus(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) { return }
	items, err := a.Store.ListModelHealthStatuses()
	if err != nil { writeError(w, http.StatusServiceUnavailable, "service_status_unavailable", "Service status is temporarily unavailable"); return }
	for index := range items { items[index].ErrorCode = "" }
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
