package identity

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	runtimeServiceName   = "ai-runtime"
	healthLease          = 2 * time.Minute
	diagnosticPreviewSize = 240
)

func diagnosticPreview(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(value, "\r", " "), "\n", " "))
	if value == "" {
		return ""
	}
	// Context builds contain user-visible facts only. Keep this guard in the
	// diagnostic projection so a future compiler cannot expose hidden thought.
	if strings.Contains(strings.ToLower(value), "chain-of-thought") || strings.Contains(strings.ToLower(value), "chain of thought") {
		return "[redacted]"
	}
	runes := []rune(value)
	if len(runes) > diagnosticPreviewSize {
		return string(runes[:diagnosticPreviewSize]) + "..."
	}
	return string(runes)
}

func contextDebugItem(item ContextBuildItem) ContextDebugItem {
	return ContextDebugItem{
		Ordinal: item.Ordinal, ItemType: item.ItemType, SourceID: item.SourceID, Role: item.Role,
		EstimatedTokens: item.EstimatedTokens, Included: item.Included, ExclusionReason: item.ExclusionReason,
		ContentLength: len([]rune(item.Content)), ContentPreview: diagnosticPreview(item.Content),
	}
}

func contextDebugView(run MessageRun, build ContextBuild) ContextDebugView {
	items := make([]ContextDebugItem, 0, len(build.Items))
	for _, item := range build.Items {
		items = append(items, contextDebugItem(item))
	}
	return ContextDebugView{
		RunID: run.ID, ConversationID: build.ConversationID, BranchID: build.BranchID, Status: run.Status, Model: run.Model, Provider: run.Provider, Cursor: run.Cursor,
		ContinuationUsed: run.ContinuationUsed, ContinuationFallback: run.ContinuationFallback,
		ContextBuildID: build.ID, ContextLimitTokens: build.ContextLimitTokens, InputBudgetTokens: build.InputBudgetTokens,
		EstimatedInputTokens: build.EstimatedInputTokens, OutputReserveTokens: build.OutputReserveTokens,
		ReasoningReserveTokens: build.ReasoningReserveTokens, ToolReserveTokens: build.ToolReserveTokens,
		SafetyMarginTokens: build.SafetyMarginTokens, CompactionMode: build.CompactionMode,
		ContinuationMode: build.ContinuationMode, ContextHash: build.ContextHash, CompilerVersion: build.CompilerVersion,
		Items: items,
	}
}

func (s *Store) ContextDebugAdmin(runID string) (ContextDebugView, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return ContextDebugView{}, errors.New("context_debug_not_found")
	}
	if s.conversationSQL != nil {
		return s.conversationSQL.contextDebugAdmin(runID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.data.Runs[runID]
	if !ok || run.ContextBuildID == "" {
		return ContextDebugView{}, errors.New("context_debug_not_found")
	}
	build, ok := s.data.ContextBuilds[run.ContextBuildID]
	if !ok {
		return ContextDebugView{}, errors.New("context_debug_not_found")
	}
	return contextDebugView(run, build), nil
}

func runtimeStatus(runtimeMode string) string {
	if strings.EqualFold(strings.TrimSpace(runtimeMode), "upstream") {
		return "healthy"
	}
	return "degraded"
}

func aggregateRuntimeHealth(runtimeMode string, instances []ServiceInstance) string {
	if len(instances) == 0 {
		return "error"
	}
	status := runtimeStatus(runtimeMode)
	for _, instance := range instances {
		if instance.ExpiresAt.Before(time.Now().UTC()) && instance.Status == "healthy" {
			return "degraded"
		}
		if instance.Status == "error" {
			return "error"
		}
		if instance.Status == "degraded" {
			status = "degraded"
		}
	}
	return status
}

func normalizeInstanceLiveness(instances []ServiceInstance, now time.Time) {
	for index := range instances {
		if instances[index].ExpiresAt.Before(now) && instances[index].Status == "healthy" {
			instances[index].Status = "degraded"
		}
	}
}

func (s *Store) AIHealth(access, runtimeMode string) (AIRuntimeHealth, error) {
	if _, err := s.authenticatedUser(access); err != nil {
		return AIRuntimeHealth{}, err
	}
	if s.conversationSQL != nil {
		return s.conversationSQL.aiHealth(s.instanceID, runtimeMode)
	}
	now := time.Now().UTC()
	instance := ServiceInstance{InstanceID: s.instanceID, ServiceName: runtimeServiceName, Version: "p03-w08", Status: runtimeStatus(runtimeMode), StartedAt: now, LastSeenAt: now, ExpiresAt: now.Add(healthLease)}
	s.mu.Lock()
	if previous, ok := s.data.ServiceInstances[s.instanceID]; ok {
		instance.StartedAt = previous.StartedAt
	}
	s.data.ServiceInstances[s.instanceID] = instance
	checkID, _ := randomToken(16)
	s.data.HealthChecks = append(s.data.HealthChecks, HealthCheck{ID: checkID, InstanceID: s.instanceID, CheckName: "runtime_configuration", Status: instance.Status, Detail: "runtime configuration observed", CheckedAt: now})
	if len(s.data.HealthChecks) > 500 {
		s.data.HealthChecks = s.data.HealthChecks[len(s.data.HealthChecks)-500:]
	}
	instances := make([]ServiceInstance, 0, len(s.data.ServiceInstances))
	for _, item := range s.data.ServiceInstances {
		instances = append(instances, item)
	}
	sort.Slice(instances, func(i, j int) bool { return instances[i].InstanceID < instances[j].InstanceID })
	normalizeInstanceLiveness(instances, now)
	checks := append([]HealthCheck(nil), s.data.HealthChecks...)
	if err := s.persistLocked(); err != nil {
		s.mu.Unlock()
		return AIRuntimeHealth{}, err
	}
	s.mu.Unlock()
	if len(checks) > 50 {
		checks = checks[len(checks)-50:]
	}
	return AIRuntimeHealth{Service: runtimeServiceName, Status: aggregateRuntimeHealth(runtimeMode, instances), InstanceID: s.instanceID, CheckedAt: now, Instances: instances, Checks: checks}, nil
}

func validateDistributedTestRun(item DistributedTestRun) error {
	item.TestKey = strings.TrimSpace(item.TestKey)
	if item.TestKey == "" {
		return errors.New("test_key_required")
	}
	if item.Status != "running" && item.Status != "pass" && item.Status != "fail" {
		return errors.New("test_run_status_invalid")
	}
	if item.CursorBefore < 0 || item.CursorAfter < 0 || (item.Status == "pass" && item.CursorAfter < item.CursorBefore) {
		return errors.New("test_run_cursor_invalid")
	}
	return nil
}

func sameDistributedTestRun(left, right DistributedTestRun) bool {
	leftDetails, _ := json.Marshal(left.Details)
	rightDetails, _ := json.Marshal(right.Details)
	return left.TestKey == right.TestKey && left.Status == right.Status && left.InstanceA == right.InstanceA && left.InstanceB == right.InstanceB && left.CursorBefore == right.CursorBefore && left.CursorAfter == right.CursorAfter && left.ErrorCode == right.ErrorCode && string(leftDetails) == string(rightDetails)
}

func canAdvanceDistributedTestRun(existing, incoming DistributedTestRun) bool {
	return existing.Status == "running" && incoming.TestKey == existing.TestKey && incoming.InstanceA == existing.InstanceA && incoming.InstanceB == existing.InstanceB && incoming.CursorBefore == existing.CursorBefore && incoming.CursorAfter >= existing.CursorAfter && (incoming.Status == "running" || incoming.Status == "pass" || incoming.Status == "fail")
}

func stableDistributedTestRunID(raw string) string {
	raw = strings.TrimSpace(raw)
	if len(raw) == 36 && raw[8] == '-' && raw[13] == '-' && raw[18] == '-' && raw[23] == '-' {
		return raw
	}
	digest := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", digest[0:4], digest[4:6], digest[6:8], digest[8:10], digest[10:16])
}

func (s *Store) RecordDistributedTestRun(item DistributedTestRun) (DistributedTestRun, bool, error) {
	if item.TestKey == "" {
		item.TestKey = "p03-distributed-chat"
	}
	if err := validateDistributedTestRun(item); err != nil {
		return DistributedTestRun{}, false, err
	}
	if strings.TrimSpace(item.ID) == "" {
		return DistributedTestRun{}, false, errors.New("test_run_id_required")
	}
	item.ID = stableDistributedTestRunID(item.ID)
	if item.Details == nil {
		item.Details = map[string]any{}
	}
	now := time.Now().UTC()
	if item.StartedAt.IsZero() {
		item.StartedAt = now
	}
	if item.Status != "running" && item.CompletedAt == nil {
		item.CompletedAt = &now
	}
	item.UpdatedAt = now
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	if s.conversationSQL != nil {
		return s.conversationSQL.recordDistributedTestRun(item)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.data.TestRuns[item.ID]; ok {
		if !sameDistributedTestRun(existing, item) {
			if !canAdvanceDistributedTestRun(existing, item) {
				return DistributedTestRun{}, false, errors.New("idempotency_conflict")
			}
			item.StartedAt = existing.StartedAt
			item.CreatedAt = existing.CreatedAt
			s.data.TestRuns[item.ID] = item
			if err := s.persistLocked(); err != nil {
				return DistributedTestRun{}, false, err
			}
			return item, false, nil
		}
		return existing, false, nil
	}
	s.data.TestRuns[item.ID] = item
	if err := s.persistLocked(); err != nil {
		return DistributedTestRun{}, false, err
	}
	return item, true, nil
}

func scanServiceInstance(scanner sqlScanner) (ServiceInstance, error) {
	var item ServiceInstance
	err := scanner.Scan(&item.InstanceID, &item.ServiceName, &item.Version, &item.Status, &item.StartedAt, &item.LastSeenAt, &item.ExpiresAt)
	return item, err
}

func scanHealthCheck(scanner sqlScanner) (HealthCheck, error) {
	var item HealthCheck
	err := scanner.Scan(&item.ID, &item.InstanceID, &item.CheckName, &item.Status, &item.Detail, &item.CheckedAt)
	return item, err
}

func scanDistributedTestRun(scanner sqlScanner) (DistributedTestRun, error) {
	var item DistributedTestRun
	var details []byte
	var completedAt sql.NullTime
	err := scanner.Scan(&item.ID, &item.TestKey, &item.Status, &item.InstanceA, &item.InstanceB, &item.CursorBefore, &item.CursorAfter, &item.ErrorCode, &details, &item.StartedAt, &completedAt, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	item.Details = map[string]any{}
	if len(details) > 0 {
		err = json.Unmarshal(details, &item.Details)
	}
	if completedAt.Valid {
		item.CompletedAt = &completedAt.Time
	}
	return item, err
}
