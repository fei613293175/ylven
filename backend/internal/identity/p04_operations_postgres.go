package identity

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

func (p *postgresConversationStore) listProviderChannels() ([]ProviderChannel, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	rows, err := p.db.QueryContext(ctx, `SELECT id,provider_id,name,credential_reference,endpoint,enabled,priority,version,updated_at
		FROM provider_channels ORDER BY provider_id,priority,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ProviderChannel{}
	for rows.Next() {
		var item ProviderChannel
		if err := rows.Scan(&item.ID, &item.ProviderID, &item.Name, &item.CredentialRef, &item.Endpoint, &item.Enabled, &item.Priority, &item.Version, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (p *postgresConversationStore) upsertProviderChannel(input ProviderChannel, actorID string) (ProviderChannel, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return ProviderChannel{}, err
	}
	defer tx.Rollback()
	var providerExists bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM model_providers WHERE id=$1)`, input.ProviderID).Scan(&providerExists); err != nil {
		return ProviderChannel{}, err
	}
	if !providerExists {
		return ProviderChannel{}, errors.New("model_provider_not_found")
	}
	var current ProviderChannel
	err = tx.QueryRowContext(ctx, `SELECT id,provider_id,name,credential_reference,endpoint,enabled,priority,version,updated_at FROM provider_channels WHERE id=$1 FOR UPDATE`, input.ID).Scan(
		&current.ID, &current.ProviderID, &current.Name, &current.CredentialRef, &current.Endpoint, &current.Enabled, &current.Priority, &current.Version, &current.UpdatedAt)
	now := time.Now().UTC()
	action := "update"
	if errors.Is(err, sql.ErrNoRows) {
		if input.Version != 0 && input.Version != 1 {
			return ProviderChannel{}, errors.New("version_conflict")
		}
		input.Version, input.UpdatedAt = 1, now
		_, err = tx.ExecContext(ctx, `INSERT INTO provider_channels(id,provider_id,name,credential_reference,endpoint,enabled,priority,version,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			input.ID, input.ProviderID, input.Name, input.CredentialRef, input.Endpoint, input.Enabled, input.Priority, input.Version, input.UpdatedAt)
		action = "create"
	} else if err != nil {
		return ProviderChannel{}, err
	} else {
		if input.Version != current.Version {
			return ProviderChannel{}, errors.New("version_conflict")
		}
		input.Version, input.UpdatedAt = current.Version+1, now
		_, err = tx.ExecContext(ctx, `UPDATE provider_channels SET provider_id=$2,name=$3,credential_reference=$4,endpoint=$5,enabled=$6,priority=$7,version=$8,updated_at=$9 WHERE id=$1`,
			input.ID, input.ProviderID, input.Name, input.CredentialRef, input.Endpoint, input.Enabled, input.Priority, input.Version, input.UpdatedAt)
	}
	if err != nil {
		return ProviderChannel{}, err
	}
	if err = insertModelCatalogAudit(ctx, tx, actorID, action, "provider_channel", input.ID, current, input); err != nil {
		return ProviderChannel{}, err
	}
	return input, tx.Commit()
}

func (p *postgresConversationStore) listRoutingPolicies() ([]RoutingPolicy, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	rows, err := p.db.QueryContext(ctx, `SELECT id,model_id,primary_channel_id,fallback_channel_ids,max_attempts,enabled,version,updated_at FROM routing_policies ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []RoutingPolicy{}
	for rows.Next() {
		var item RoutingPolicy
		var raw []byte
		if err := rows.Scan(&item.ID, &item.ModelID, &item.Primary, &raw, &item.MaxAttempts, &item.Enabled, &item.Version, &item.UpdatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &item.Fallbacks); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (p *postgresConversationStore) upsertRoutingPolicy(input RoutingPolicy, actorID string) (RoutingPolicy, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return RoutingPolicy{}, err
	}
	defer tx.Rollback()
	var modelProviderID string
	err = tx.QueryRowContext(ctx, `SELECT provider_id FROM models WHERE id=$1`, input.ModelID).Scan(&modelProviderID)
	if errors.Is(err, sql.ErrNoRows) {
		return RoutingPolicy{}, errors.New("model_not_found")
	}
	if err != nil {
		return RoutingPolicy{}, err
	}
	channelIDs := append([]string{input.Primary}, input.Fallbacks...)
	seen := map[string]bool{}
	cleanFallbacks := []string{}
	for index, channelID := range channelIDs {
		channelID = strings.TrimSpace(channelID)
		if channelID == "" || seen[channelID] {
			return RoutingPolicy{}, errors.New("routing_policy_invalid")
		}
		seen[channelID] = true
		var channelProviderID string
		err = tx.QueryRowContext(ctx, `SELECT provider_id FROM provider_channels WHERE id=$1`, channelID).Scan(&channelProviderID)
		if errors.Is(err, sql.ErrNoRows) {
			return RoutingPolicy{}, errors.New("provider_channel_not_found")
		}
		if err != nil {
			return RoutingPolicy{}, err
		}
		if channelProviderID != modelProviderID {
			return RoutingPolicy{}, errors.New("routing_channel_provider_mismatch")
		}
		if index > 0 {
			cleanFallbacks = append(cleanFallbacks, channelID)
		}
	}
	input.Fallbacks = cleanFallbacks
	raw, err := json.Marshal(input.Fallbacks)
	if err != nil {
		return RoutingPolicy{}, err
	}
	var current RoutingPolicy
	var currentRaw []byte
	err = tx.QueryRowContext(ctx, `SELECT id,model_id,primary_channel_id,fallback_channel_ids,max_attempts,enabled,version,updated_at FROM routing_policies WHERE id=$1 FOR UPDATE`, input.ID).Scan(
		&current.ID, &current.ModelID, &current.Primary, &currentRaw, &current.MaxAttempts, &current.Enabled, &current.Version, &current.UpdatedAt)
	if err == nil {
		_ = json.Unmarshal(currentRaw, &current.Fallbacks)
	}
	now := time.Now().UTC()
	action := "update"
	if errors.Is(err, sql.ErrNoRows) {
		if input.Version != 0 && input.Version != 1 {
			return RoutingPolicy{}, errors.New("version_conflict")
		}
		input.Version, input.UpdatedAt = 1, now
		_, err = tx.ExecContext(ctx, `INSERT INTO routing_policies(id,model_id,primary_channel_id,fallback_channel_ids,max_attempts,enabled,version,updated_at) VALUES($1,$2,$3,$4::jsonb,$5,$6,$7,$8)`,
			input.ID, input.ModelID, input.Primary, string(raw), input.MaxAttempts, input.Enabled, input.Version, input.UpdatedAt)
		action = "create"
	} else if err != nil {
		return RoutingPolicy{}, err
	} else {
		if input.Version != current.Version {
			return RoutingPolicy{}, errors.New("version_conflict")
		}
		input.Version, input.UpdatedAt = current.Version+1, now
		_, err = tx.ExecContext(ctx, `UPDATE routing_policies SET model_id=$2,primary_channel_id=$3,fallback_channel_ids=$4::jsonb,max_attempts=$5,enabled=$6,version=$7,updated_at=$8 WHERE id=$1`,
			input.ID, input.ModelID, input.Primary, string(raw), input.MaxAttempts, input.Enabled, input.Version, input.UpdatedAt)
	}
	if err != nil {
		return RoutingPolicy{}, err
	}
	if err = insertModelCatalogAudit(ctx, tx, actorID, action, "routing_policy", input.ID, current, input); err != nil {
		return RoutingPolicy{}, err
	}
	return input, tx.Commit()
}

func (p *postgresConversationStore) providerRuntimePolicy(providerID string) (ProviderRuntimePolicy, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	var item ProviderRuntimePolicy
	err := p.db.QueryRowContext(ctx, `SELECT provider_id,max_concurrency,timeout_seconds,circuit_threshold,cooldown_seconds,enabled,version,updated_at FROM provider_runtime_policies WHERE provider_id=$1`, providerID).Scan(
		&item.ProviderID, &item.MaxConcurrency, &item.TimeoutSeconds, &item.CircuitThreshold, &item.CooldownSeconds, &item.Enabled, &item.Version, &item.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ProviderRuntimePolicy{}, errors.New("provider_runtime_policy_not_found")
	}
	return item, err
}

func (p *postgresConversationStore) providerRuntimePolicyForModel(modelID string) (ProviderRuntimePolicy, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	var item ProviderRuntimePolicy
	err := p.db.QueryRowContext(ctx, `SELECT m.provider_id,COALESCE(r.max_concurrency,8),COALESCE(r.timeout_seconds,90),COALESCE(r.circuit_threshold,3),COALESCE(r.cooldown_seconds,60),COALESCE(r.enabled,TRUE),COALESCE(r.version,1),COALESCE(r.updated_at,NOW()) FROM models m LEFT JOIN provider_runtime_policies r ON r.provider_id=m.provider_id WHERE m.id=$1`, modelID).Scan(
		&item.ProviderID, &item.MaxConcurrency, &item.TimeoutSeconds, &item.CircuitThreshold, &item.CooldownSeconds, &item.Enabled, &item.Version, &item.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		var catalogCount int
		_ = p.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM models`).Scan(&catalogCount)
		if catalogCount == 1 {
			return ProviderRuntimePolicy{ProviderID: "upstream", MaxConcurrency: 8, TimeoutSeconds: 90, CircuitThreshold: 3, CooldownSeconds: 60, Enabled: true, Version: 1}, nil
		}
		return ProviderRuntimePolicy{}, errors.New("model_not_found")
	}
	return item, err
}

func (p *postgresConversationStore) routingPlanForModel(modelID string) (RoutingPlan, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	var primary string
	var fallbackRaw []byte
	var maxAttempts int
	err := p.db.QueryRowContext(ctx, `SELECT primary_channel_id,fallback_channel_ids,max_attempts FROM routing_policies WHERE model_id=$1 AND enabled ORDER BY updated_at DESC,id DESC LIMIT 1`, modelID).Scan(&primary, &fallbackRaw, &maxAttempts)
	if errors.Is(err, sql.ErrNoRows) { return RoutingPlan{}, nil }
	if err != nil { return RoutingPlan{}, err }
	var fallbacks []string
	if err = json.Unmarshal(fallbackRaw, &fallbacks); err != nil { return RoutingPlan{}, err }
	plan := RoutingPlan{Configured:true, MaxAttempts:maxAttempts}
	if plan.MaxAttempts < 1 { plan.MaxAttempts = 1 }
	for _, channelID := range append([]string{primary}, fallbacks...) {
		if len(plan.Targets) >= plan.MaxAttempts { break }
		var channel ProviderChannel
		err = p.db.QueryRowContext(ctx, `SELECT id,provider_id,name,credential_reference,endpoint,enabled,priority,version,updated_at FROM provider_channels WHERE id=$1 AND enabled`, channelID).Scan(&channel.ID,&channel.ProviderID,&channel.Name,&channel.CredentialRef,&channel.Endpoint,&channel.Enabled,&channel.Priority,&channel.Version,&channel.UpdatedAt)
		if errors.Is(err, sql.ErrNoRows) { continue }
		if err != nil { return RoutingPlan{}, err }
		plan.Targets = append(plan.Targets, channel)
	}
	return plan, nil
}

func (p *postgresConversationStore) setRunProvider(userID, runID, provider string) error {
	ctx, cancel := databaseContext()
	defer cancel()
	result, err := p.db.ExecContext(ctx, `UPDATE message_runs SET provider=$3,updated_at=NOW() WHERE id=$1 AND user_id=$2 AND status='streaming'`, runID, userID, provider)
	if err != nil { return err }
	count, err := result.RowsAffected(); if err != nil { return err }
	if count == 0 { return errors.New("run_not_found") }
	return nil
}

func (p *postgresConversationStore) upsertProviderRuntimePolicy(input ProviderRuntimePolicy, actorID string) (ProviderRuntimePolicy, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return ProviderRuntimePolicy{}, err
	}
	defer tx.Rollback()
	var providerExists bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM model_providers WHERE id=$1)`, input.ProviderID).Scan(&providerExists); err != nil {
		return ProviderRuntimePolicy{}, err
	}
	if !providerExists {
		return ProviderRuntimePolicy{}, errors.New("model_provider_not_found")
	}
	var current ProviderRuntimePolicy
	err = tx.QueryRowContext(ctx, `SELECT provider_id,max_concurrency,timeout_seconds,circuit_threshold,cooldown_seconds,enabled,version,updated_at FROM provider_runtime_policies WHERE provider_id=$1 FOR UPDATE`, input.ProviderID).Scan(
		&current.ProviderID, &current.MaxConcurrency, &current.TimeoutSeconds, &current.CircuitThreshold, &current.CooldownSeconds, &current.Enabled, &current.Version, &current.UpdatedAt)
	now := time.Now().UTC()
	action := "update"
	if errors.Is(err, sql.ErrNoRows) {
		if input.Version != 0 && input.Version != 1 {
			return ProviderRuntimePolicy{}, errors.New("version_conflict")
		}
		input.Version, input.UpdatedAt = 1, now
		_, err = tx.ExecContext(ctx, `INSERT INTO provider_runtime_policies(provider_id,max_concurrency,timeout_seconds,circuit_threshold,cooldown_seconds,enabled,version,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`,
			input.ProviderID, input.MaxConcurrency, input.TimeoutSeconds, input.CircuitThreshold, input.CooldownSeconds, input.Enabled, input.Version, input.UpdatedAt)
		action = "create"
	} else if err != nil {
		return ProviderRuntimePolicy{}, err
	} else {
		if input.Version != current.Version {
			return ProviderRuntimePolicy{}, errors.New("version_conflict")
		}
		input.Version, input.UpdatedAt = current.Version+1, now
		_, err = tx.ExecContext(ctx, `UPDATE provider_runtime_policies SET max_concurrency=$2,timeout_seconds=$3,circuit_threshold=$4,cooldown_seconds=$5,enabled=$6,version=$7,updated_at=$8 WHERE provider_id=$1`,
			input.ProviderID, input.MaxConcurrency, input.TimeoutSeconds, input.CircuitThreshold, input.CooldownSeconds, input.Enabled, input.Version, input.UpdatedAt)
	}
	if err != nil {
		return ProviderRuntimePolicy{}, err
	}
	if err = insertModelCatalogAudit(ctx, tx, actorID, action, "provider_runtime_policy", input.ProviderID, current, input); err != nil {
		return ProviderRuntimePolicy{}, err
	}
	return input, tx.Commit()
}

func (p *postgresConversationStore) listProviderRuntimePolicies() ([]ProviderRuntimePolicy, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	rows, err := p.db.QueryContext(ctx, `SELECT provider_id,max_concurrency,timeout_seconds,circuit_threshold,cooldown_seconds,enabled,version,updated_at FROM provider_runtime_policies ORDER BY provider_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ProviderRuntimePolicy{}
	for rows.Next() {
		var item ProviderRuntimePolicy
		if err := rows.Scan(&item.ProviderID, &item.MaxConcurrency, &item.TimeoutSeconds, &item.CircuitThreshold, &item.CooldownSeconds, &item.Enabled, &item.Version, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (p *postgresConversationStore) modelCapabilityConfig(modelID string) (ModelCapabilityConfig, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	var item ModelCapabilityConfig
	item.ModelID = modelID
	err := p.db.QueryRowContext(ctx, `SELECT model_id,context_limit_tokens,output_reserve_tokens,reasoning_reserve_tokens,tool_reserve_tokens,safety_margin_tokens,updated_at FROM model_capabilities WHERE model_id=$1`, modelID).Scan(
		&item.ModelID, &item.ContextLimitTokens, &item.OutputReserveTokens, &item.ReasoningReserveTokens, &item.ToolReserveTokens, &item.SafetyMarginTokens, &item.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		var exists bool
		if err = p.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM models WHERE id=$1)`, modelID).Scan(&exists); err != nil {
			return ModelCapabilityConfig{}, err
		}
		if !exists {
			return ModelCapabilityConfig{}, errors.New("model_not_found")
		}
		return ModelCapabilityConfig{ModelID: modelID, ModelCapability: DefaultModelCapability(modelID)}, nil
	}
	return item, err
}

func (p *postgresConversationStore) upsertModelCapabilityConfig(input ModelCapabilityConfig, actorID string) (ModelCapabilityConfig, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return ModelCapabilityConfig{}, err
	}
	defer tx.Rollback()
	var exists bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM models WHERE id=$1)`, input.ModelID).Scan(&exists); err != nil {
		return ModelCapabilityConfig{}, err
	}
	if !exists {
		return ModelCapabilityConfig{}, errors.New("model_not_found")
	}
	before := ModelCapabilityConfig{ModelID: input.ModelID, ModelCapability: DefaultModelCapability(input.ModelID)}
	err = tx.QueryRowContext(ctx, `SELECT model_id,context_limit_tokens,output_reserve_tokens,reasoning_reserve_tokens,tool_reserve_tokens,safety_margin_tokens,updated_at FROM model_capabilities WHERE model_id=$1 FOR UPDATE`, input.ModelID).Scan(
		&before.ModelID, &before.ContextLimitTokens, &before.OutputReserveTokens, &before.ReasoningReserveTokens, &before.ToolReserveTokens, &before.SafetyMarginTokens, &before.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	if err != nil {
		return ModelCapabilityConfig{}, err
	}
	input.UpdatedAt = time.Now().UTC()
	_, err = tx.ExecContext(ctx, `INSERT INTO model_capabilities(model_id,context_limit_tokens,output_reserve_tokens,reasoning_reserve_tokens,tool_reserve_tokens,safety_margin_tokens,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT(model_id) DO UPDATE SET context_limit_tokens=EXCLUDED.context_limit_tokens,output_reserve_tokens=EXCLUDED.output_reserve_tokens,
		reasoning_reserve_tokens=EXCLUDED.reasoning_reserve_tokens,tool_reserve_tokens=EXCLUDED.tool_reserve_tokens,
		safety_margin_tokens=EXCLUDED.safety_margin_tokens,updated_at=EXCLUDED.updated_at`,
		input.ModelID, input.ContextLimitTokens, input.OutputReserveTokens, input.ReasoningReserveTokens, input.ToolReserveTokens, input.SafetyMarginTokens, input.UpdatedAt)
	if err != nil {
		return ModelCapabilityConfig{}, err
	}
	if err = insertModelCatalogAudit(ctx, tx, actorID, "update", "model_capability", input.ModelID, before, input); err != nil {
		return ModelCapabilityConfig{}, err
	}
	return input, tx.Commit()
}

func (p *postgresConversationStore) recordModelHealthAdmin(input ModelHealthStatus, actorID string) (ModelHealthStatus, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return ModelHealthStatus{}, err
	}
	defer tx.Rollback()
	var providerID string
	err = tx.QueryRowContext(ctx, `SELECT provider_id FROM models WHERE id=$1`, input.ModelID).Scan(&providerID)
	if errors.Is(err, sql.ErrNoRows) {
		return ModelHealthStatus{}, errors.New("model_not_found")
	}
	if err != nil {
		return ModelHealthStatus{}, err
	}
	if input.ProviderID != "" && input.ProviderID != providerID {
		return ModelHealthStatus{}, errors.New("model_provider_mismatch")
	}
	input.ProviderID = providerID
	capabilities, err := json.Marshal(input.Capabilities)
	if err != nil {
		return ModelHealthStatus{}, err
	}
	before := ModelHealthStatus{ModelID: input.ModelID, ProviderID: providerID}
	var beforeRaw []byte
	err = tx.QueryRowContext(ctx, `SELECT model_id,provider_id,status,latency_ms,last_probe_at,capabilities,COALESCE(error_code,'') FROM model_health_status WHERE model_id=$1 FOR UPDATE`, input.ModelID).Scan(
		&before.ModelID, &before.ProviderID, &before.Status, &before.LatencyMs, &before.LastProbeAt, &beforeRaw, &before.ErrorCode)
	if err == nil {
		_ = json.Unmarshal(beforeRaw, &before.Capabilities)
	} else if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	if err != nil {
		return ModelHealthStatus{}, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO model_health_status(model_id,provider_id,status,latency_ms,last_probe_at,capabilities,error_code)
		VALUES($1,$2,$3,$4,$5,$6::jsonb,NULLIF($7,''))
		ON CONFLICT(model_id) DO UPDATE SET provider_id=EXCLUDED.provider_id,status=EXCLUDED.status,latency_ms=EXCLUDED.latency_ms,
		last_probe_at=EXCLUDED.last_probe_at,capabilities=EXCLUDED.capabilities,error_code=EXCLUDED.error_code`,
		input.ModelID, input.ProviderID, input.Status, input.LatencyMs, input.LastProbeAt, string(capabilities), input.ErrorCode)
	if err != nil {
		return ModelHealthStatus{}, err
	}
	if err = insertModelCatalogAudit(ctx, tx, actorID, "record_health", "model_health", input.ModelID, before, input); err != nil {
		return ModelHealthStatus{}, err
	}
	return input, tx.Commit()
}
