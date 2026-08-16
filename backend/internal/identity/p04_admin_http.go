package identity

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
)

func adminWriteP04Error(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	code := err.Error()
	switch {
	case strings.Contains(code, "version_conflict"):
		writeError(w, http.StatusConflict, "version_conflict", "The resource changed; reload before saving")
	case strings.Contains(code, "not_found"):
		writeError(w, http.StatusNotFound, code, "Requested resource was not found")
	case strings.Contains(code, "invalid"), strings.Contains(code, "mismatch"):
		writeError(w, http.StatusUnprocessableEntity, code, "The request is invalid")
	default:
		writeError(w, http.StatusInternalServerError, "p04_admin_unavailable", "P04 administration is unavailable")
	}
}

func adminP04Permission(method string) string {
	if method == http.MethodGet {
		return "models:read"
	}
	return "models:manage"
}

func (a *API) adminProviderChannels(w http.ResponseWriter, r *http.Request) {
	admin, ok := a.requireAdmin(w, r, adminP04Permission(r.Method))
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		items, err := a.Store.ListProviderChannels()
		if err != nil {
			adminWriteP04Error(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodPost:
		if !a.requireStepUp(w, r) {
			return
		}
		var input ProviderChannel
		if !decode(r, &input) {
			writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON")
			return
		}
		if strings.TrimSpace(input.ID) == "" {
			token, err := randomToken(12)
			if err != nil {
				adminWriteP04Error(w, err)
				return
			}
			input.ID = "channel-" + token
		}
		saved, err := a.Store.UpsertProviderChannel(input, admin.ID)
		if err != nil {
			adminWriteP04Error(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"channel": saved, "audited": true})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET or POST required")
	}
}

func (a *API) adminModelByID(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/admin/v1/models/"), "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		writeError(w, http.StatusBadRequest, "model_id_required", "Model ID required")
		return
	}
	modelID := strings.TrimSpace(parts[0])
	if _, ok := a.requireAdmin(w, r, adminP04Permission(r.Method)); !ok {
		return
	}
	if len(parts) == 1 {
		a.adminModelResource(w, r, modelID)
		return
	}
	if len(parts) != 2 {
		writeError(w, http.StatusNotFound, "not_found", "Endpoint not found")
		return
	}
	switch parts[1] {
	case "capabilities":
		a.adminModelCapabilities(w, r, modelID)
	case "reasoning-profiles":
		a.adminModelReasoningProfile(w, r, modelID)
	case "health-probe":
		a.adminModelHealthProbe(w, r, modelID)
	case "capability-probe":
		a.adminModelCapabilityProbe(w, r, modelID)
	default:
		writeError(w, http.StatusNotFound, "not_found", "Endpoint not found")
	}
}

func (a *API) adminModelResource(w http.ResponseWriter, r *http.Request, modelID string) {
	switch r.Method {
	case http.MethodGet:
		snapshot, err := a.Store.ModelCatalogSnapshot()
		if err != nil {
			adminWriteP04Error(w, err)
			return
		}
		for _, model := range snapshot.Models {
			if model.ID != modelID {
				continue
			}
			config, configErr := a.Store.GetModelCapabilityConfig(modelID)
			if configErr != nil {
				adminWriteP04Error(w, configErr)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"model": AdminModelCatalogEntry{ModelCatalogEntry: model, UpstreamModel: model.UpstreamModel}, "capability_config": config})
			return
		}
		writeError(w, http.StatusNotFound, "model_not_found", "Model not found")
	case http.MethodPut:
		if !a.requireStepUp(w, r) {
			return
		}
		admin, _ := a.Store.AuthorizeAdmin(bearer(r), "models:manage")
		var input struct {
			ModelCatalogEntry
			UpstreamModel string `json:"upstream_model"`
		}
		if !decode(r, &input) {
			writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON")
			return
		}
		input.ModelCatalogEntry.ID = modelID
		input.UpstreamModel = strings.TrimSpace(input.UpstreamModel)
		input.ModelCatalogEntry.UpstreamModel = input.UpstreamModel
		saved, err := a.Store.UpsertModelCatalogEntry(input.ModelCatalogEntry, admin.ID)
		if err != nil {
			writeModelCatalogError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"model": AdminModelCatalogEntry{ModelCatalogEntry: saved, UpstreamModel: saved.UpstreamModel}, "audited": true})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET or PUT required")
	}
}

func (a *API) adminModelCapabilities(w http.ResponseWriter, r *http.Request, modelID string) {
	if r.Method == http.MethodGet {
		value, err := a.Store.GetModelCapabilityConfig(modelID)
		if err != nil {
			adminWriteP04Error(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"capability_config": value})
		return
	}
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET or PUT required")
		return
	}
	if !a.requireStepUp(w, r) {
		return
	}
	admin, _ := a.Store.AuthorizeAdmin(bearer(r), "models:manage")
	var input ModelCapabilityConfig
	if !decode(r, &input) {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON")
		return
	}
	input.ModelID = modelID
	saved, err := a.Store.UpsertModelCapabilityConfig(input, admin.ID)
	if err != nil {
		adminWriteP04Error(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"capability_config": saved, "audited": true})
}

func (a *API) adminModelReasoningProfile(w http.ResponseWriter, r *http.Request, modelID string) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "PUT required")
		return
	}
	if !a.requireStepUp(w, r) {
		return
	}
	admin, _ := a.Store.AuthorizeAdmin(bearer(r), "models:manage")
	var input ReasoningProfileMapping
	if !decode(r, &input) {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON")
		return
	}
	input.ModelID = modelID
	saved, err := a.Store.UpsertReasoningProfile(input, admin.ID)
	if err != nil {
		writeModelCatalogError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reasoning_profile": saved, "audited": true})
}

func (a *API) adminModelHealthProbe(w http.ResponseWriter, r *http.Request, modelID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required")
		return
	}
	if !a.requireStepUp(w, r) {
		return
	}
	admin, _ := a.Store.AuthorizeAdmin(bearer(r), "models:manage")
	health, probeErr := a.executeModelProbe(r.Context(), modelID)
	if health.ModelID == "" {
		adminWriteP04Error(w, probeErr)
		return
	}
	saved, err := a.Store.RecordModelHealthAdmin(health, admin.ID)
	if err != nil {
		adminWriteP04Error(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"health": saved, "probe_succeeded": probeErr == nil})
}

func (a *API) adminModelCapabilityProbe(w http.ResponseWriter, r *http.Request, modelID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required")
		return
	}
	if !a.requireStepUp(w, r) {
		return
	}
	admin, _ := a.Store.AuthorizeAdmin(bearer(r), "models:manage")
	var input struct {
		CapabilityID string `json:"capability_id"`
	}
	if !decode(r, &input) || strings.TrimSpace(input.CapabilityID) == "" {
		writeError(w, http.StatusBadRequest, "capability_id_required", "Capability ID required")
		return
	}
	health, probeErr := a.executeModelCapabilityProbe(r.Context(), modelID, strings.TrimSpace(input.CapabilityID))
	if health.ModelID == "" {
		adminWriteP04Error(w, probeErr)
		return
	}
	if _, err := a.Store.RecordModelHealthAdmin(health, admin.ID); err != nil {
		adminWriteP04Error(w, err)
		return
	}
	status := "passed"
	if probeErr != nil {
		status = "failed"
	}
	probedAt := time.Now().UTC()
	probe, err := a.Store.RecordCapabilityProbe(CapabilityProbeResult{
		ModelID: modelID, CapabilityID: strings.TrimSpace(input.CapabilityID), Status: status,
		ProbeSource: "server_chat_probe", EvidenceReference: "probe://" + modelID + "/" + probedAt.Format("20060102T150405.000Z"), ProbedAt: probedAt,
	}, admin.ID)
	if err != nil {
		writeModelCatalogError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"probe": probe, "probe_succeeded": probeErr == nil})
}

func (a *API) executeModelProbe(parent context.Context, modelID string) (ModelHealthStatus, error) {
	modelID = strings.TrimSpace(modelID)
	selection, err := a.Store.ResolveModelSelection(modelID, "auto")
	if err != nil {
		return ModelHealthStatus{ModelID: modelID, Status: "unavailable", ErrorCode: err.Error(), LastProbeAt: time.Now().UTC()}, err
	}
	started := time.Now()
	ctx, cancel := context.WithTimeout(parent, 20*time.Second)
	defer cancel()
	_, _, err = a.respondForModel(ctx, modelID, ChatRequest{Model: selection.ProviderModel, Messages: []ProviderMessage{{Role: "system", Content: "Health probe. Reply with OK only."}, {Role: "user", Content: "OK"}}, ReasoningParameters: copyJSONMap(selection.ReasoningParameters)})
	status := "available"
	errorCode := ""
	if err != nil {
		status = "degraded"
		errorCode = "chat_provider_error"
	}
	return ModelHealthStatus{ModelID: modelID, Status: status, LatencyMs: time.Since(started).Milliseconds(), LastProbeAt: time.Now().UTC(), Capabilities: []string{"text"}, ErrorCode: errorCode}, err
}

func (a *API) executeModelCapabilityProbe(parent context.Context, modelID, capabilityID string) (ModelHealthStatus, error) {
	if capabilityID == "text" {
		return a.executeModelProbe(parent, modelID)
	}
	selection, err := a.Store.ResolveModelSelection(modelID, "auto")
	if err != nil {
		return ModelHealthStatus{ModelID: modelID, Status: "unavailable", ErrorCode: err.Error(), LastProbeAt: time.Now().UTC()}, err
	}
	started := time.Now()
	ctx, cancel := context.WithTimeout(parent, 20*time.Second)
	defer cancel()
	err = a.probeModelCapability(ctx, modelID, selection.ProviderModel, capabilityID, copyJSONMap(selection.ReasoningParameters))
	status, code := "available", ""
	if err != nil {
		status, code = "degraded", "capability_probe_failed"
	}
	return ModelHealthStatus{ModelID: modelID, Status: status, LatencyMs: time.Since(started).Milliseconds(), LastProbeAt: time.Now().UTC(), Capabilities: []string{"text"}, ErrorCode: code}, err
}

func (a *API) probeModelCapability(ctx context.Context, modelID, providerModel, capabilityID string, parameters map[string]any) error {
	plan, err := a.Store.RoutingPlanForModel(modelID)
	if err != nil {
		return err
	}
	if !plan.Configured {
		if a.ChatRuntimeMode != "upstream" || a.ChatResponder == nil {
			return errors.New("chat_runtime_unavailable")
		}
		prober, supported := a.ChatResponder.(ModelCapabilityProber)
		if !supported {
			return errors.New("capability_probe_unsupported")
		}
		return prober.ProbeCapability(ctx, providerModel, capabilityID, parameters)
	}
	if len(plan.Targets) == 0 {
		return errors.New("model_route_unavailable")
	}
	lastErr := errors.New("model_route_unavailable")
	for _, channel := range plan.Targets {
		responder, responderErr := a.responderForChannel(channel)
		if responderErr != nil {
			lastErr = responderErr
			continue
		}
		prober, supported := responder.(ModelCapabilityProber)
		if !supported {
			lastErr = errors.New("capability_probe_unsupported")
			continue
		}
		if err = prober.ProbeCapability(ctx, providerModel, capabilityID, parameters); err == nil {
			return nil
		}
		lastErr = err
	}
	return lastErr
}

func (a *API) adminRoutingPolicyByID(w http.ResponseWriter, r *http.Request) {
	policyID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/admin/v1/routing-policies/"), "/")
	if policyID == "" {
		writeError(w, http.StatusBadRequest, "policy_id_required", "Policy ID required")
		return
	}
	admin, ok := a.requireAdmin(w, r, adminP04Permission(r.Method))
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		items, err := a.Store.ListRoutingPolicies()
		if err != nil {
			adminWriteP04Error(w, err)
			return
		}
		for _, item := range items {
			if item.ID == policyID {
				writeJSON(w, http.StatusOK, map[string]any{"policy": item})
				return
			}
		}
		writeError(w, http.StatusNotFound, "routing_policy_not_found", "Routing policy not found")
	case http.MethodPut:
		if !a.requireStepUp(w, r) {
			return
		}
		var input RoutingPolicy
		if !decode(r, &input) {
			writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON")
			return
		}
		input.ID = policyID
		saved, err := a.Store.UpsertRoutingPolicy(input, admin.ID)
		if err != nil {
			adminWriteP04Error(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"policy": saved, "audited": true})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET or PUT required")
	}
}

func (a *API) adminRoutingPolicies(w http.ResponseWriter, r *http.Request) {
	admin, ok := a.requireAdmin(w, r, adminP04Permission(r.Method))
	if !ok { return }
	switch r.Method {
	case http.MethodGet:
		items, err := a.Store.ListRoutingPolicies()
		if err != nil { adminWriteP04Error(w, err); return }
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodPost:
		if !a.requireStepUp(w, r) { return }
		var input RoutingPolicy
		if !decode(r, &input) { writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON"); return }
		if strings.TrimSpace(input.ID) == "" {
			token, err := randomToken(12)
			if err != nil { adminWriteP04Error(w, err); return }
			input.ID = "route-" + token
		}
		saved, err := a.Store.UpsertRoutingPolicy(input, admin.ID)
		if err != nil { adminWriteP04Error(w, err); return }
		writeJSON(w, http.StatusCreated, map[string]any{"policy": saved, "audited": true})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET or POST required")
	}
}

func (a *API) adminProviderRuntimePolicies(w http.ResponseWriter, r *http.Request) {
	admin, ok := a.requireAdmin(w, r, adminP04Permission(r.Method))
	if !ok { return }
	switch r.Method {
	case http.MethodGet:
		items, err := a.Store.ListProviderRuntimePolicies()
		if err != nil { adminWriteP04Error(w, err); return }
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodPost:
		if !a.requireStepUp(w, r) { return }
		var input ProviderRuntimePolicy
		if !decode(r, &input) { writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON"); return }
		saved, err := a.Store.UpsertProviderRuntimePolicy(input, admin.ID)
		if err != nil { adminWriteP04Error(w, err); return }
		writeJSON(w, http.StatusCreated, map[string]any{"policy": saved, "audited": true})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET or POST required")
	}
}

func (a *API) adminModelHealth(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) { return }
	if _, ok := a.requireAdmin(w, r, "models:read"); !ok { return }
	items, err := a.Store.ListModelHealthStatuses()
	if err != nil { adminWriteP04Error(w, err); return }
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *API) adminComparisons(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) { return }
	if _, ok := a.requireAdmin(w, r, "conversations:read"); !ok { return }
	items, err := a.Store.ListComparisonGroups()
	if err != nil { adminWriteP04Error(w, err); return }
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *API) adminProviderRuntimePolicyByID(w http.ResponseWriter, r *http.Request) {
	providerID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/admin/v1/provider-runtime-policies/"), "/")
	if providerID == "" {
		writeError(w, http.StatusBadRequest, "provider_id_required", "Provider ID required")
		return
	}
	admin, ok := a.requireAdmin(w, r, adminP04Permission(r.Method))
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		item, err := a.Store.GetProviderRuntimePolicy(providerID)
		if err != nil {
			adminWriteP04Error(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"policy": item})
	case http.MethodPut:
		if !a.requireStepUp(w, r) {
			return
		}
		var input ProviderRuntimePolicy
		if !decode(r, &input) {
			writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON")
			return
		}
		input.ProviderID = providerID
		saved, err := a.Store.UpsertProviderRuntimePolicy(input, admin.ID)
		if err != nil {
			adminWriteP04Error(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"policy": saved, "audited": true})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET or PUT required")
	}
}

func (a *API) adminUsageEvents(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	if _, ok := a.requireAdmin(w, r, "models:read"); !ok {
		return
	}
	items, err := a.Store.ListUsageEvents()
	if err != nil {
		adminWriteP04Error(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *API) adminPriceSnapshots(w http.ResponseWriter, r *http.Request) {
	admin, ok := a.requireAdmin(w, r, adminP04Permission(r.Method))
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		items, err := a.Store.ListPriceSnapshots()
		if err != nil {
			adminWriteP04Error(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodPut, http.MethodPost:
		if !a.requireStepUp(w, r) {
			return
		}
		var input PriceSnapshot
		if !decode(r, &input) {
			writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON")
			return
		}
		saved, err := a.Store.UpsertPriceSnapshotAdmin(input, admin.ID)
		if err != nil {
			adminWriteP04Error(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"snapshot": saved, "audited": true})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "GET, POST, or PUT required")
	}
}
