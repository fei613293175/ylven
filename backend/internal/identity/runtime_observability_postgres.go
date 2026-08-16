package identity

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

func (p *postgresConversationStore) contextDebugAdmin(runID string) (ContextDebugView, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	run, err := scanRun(p.db.QueryRowContext(ctx, `SELECT `+runColumns+` FROM message_runs WHERE id::text=$1`, runID))
	if errors.Is(err, sql.ErrNoRows) {
		return ContextDebugView{}, errors.New("context_debug_not_found")
	}
	if err != nil {
		return ContextDebugView{}, err
	}
	var build ContextBuild
	err = p.db.QueryRowContext(ctx, `SELECT cb.id::text,cb.run_id::text,cb.conversation_id::text,cb.branch_id::text,cb.model,
    cb.context_limit_tokens,cb.input_budget_tokens,cb.estimated_input_tokens,cb.output_reserve_tokens,
    cb.reasoning_reserve_tokens,cb.tool_reserve_tokens,cb.safety_margin_tokens,cb.compaction_mode,
    cb.continuation_mode,cb.context_hash,cb.compiler_version
    FROM context_builds cb WHERE cb.run_id::text=$1 ORDER BY cb.created_at DESC LIMIT 1`, runID).Scan(
		&build.ID, &build.RunID, &build.ConversationID, &build.BranchID, &build.Model, &build.ContextLimitTokens,
		&build.InputBudgetTokens, &build.EstimatedInputTokens, &build.OutputReserveTokens, &build.ReasoningReserveTokens,
		&build.ToolReserveTokens, &build.SafetyMarginTokens, &build.CompactionMode, &build.ContinuationMode,
		&build.ContextHash, &build.CompilerVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return ContextDebugView{}, errors.New("context_debug_not_found")
	}
	if err != nil {
		return ContextDebugView{}, err
	}
	rows, err := p.db.QueryContext(ctx, `SELECT ordinal,item_type,COALESCE(source_id,''),COALESCE(role,''),
    COALESCE(content_redacted,''),estimated_tokens,included,COALESCE(exclusion_reason,'')
    FROM context_build_items WHERE context_build_id=$1 ORDER BY ordinal`, build.ID)
	if err != nil {
		return ContextDebugView{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var item ContextBuildItem
		if err := rows.Scan(&item.Ordinal, &item.ItemType, &item.SourceID, &item.Role, &item.Content, &item.EstimatedTokens, &item.Included, &item.ExclusionReason); err != nil {
			return ContextDebugView{}, err
		}
		build.Items = append(build.Items, item)
	}
	if err := rows.Err(); err != nil {
		return ContextDebugView{}, err
	}
	return contextDebugView(run, build), nil
}

func (p *postgresConversationStore) aiHealth(instanceID, runtimeMode string) (AIRuntimeHealth, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	now := time.Now().UTC()
	status := runtimeStatus(runtimeMode)
	startedAt := now
	var previousStarted time.Time
	if err := p.db.QueryRowContext(ctx, `SELECT started_at FROM service_instances WHERE instance_id=$1`, instanceID).Scan(&previousStarted); err == nil {
		startedAt = previousStarted
	}
	_, err := p.db.ExecContext(ctx, `INSERT INTO service_instances (instance_id,service_name,version,status,started_at,last_seen_at,expires_at,metadata)
    VALUES ($1,'ai-runtime','p03-w08',$2,$3,$4,$5,'{}'::jsonb)
    ON CONFLICT (instance_id) DO UPDATE SET status=EXCLUDED.status,last_seen_at=EXCLUDED.last_seen_at,
      expires_at=EXCLUDED.expires_at,version=EXCLUDED.version`, instanceID, status, startedAt, now, now.Add(healthLease))
	if err != nil {
		return AIRuntimeHealth{}, err
	}
	checkID, err := newDatabaseUUID()
	if err != nil {
		return AIRuntimeHealth{}, err
	}
	if _, err := p.db.ExecContext(ctx, `INSERT INTO health_checks (id,instance_id,check_name,status,detail,checked_at)
    VALUES ($1,$2,'runtime_configuration',$3,$4,$5)`, checkID, instanceID, status, "runtime configuration observed", now); err != nil {
		return AIRuntimeHealth{}, err
	}
	rows, err := p.db.QueryContext(ctx, `SELECT instance_id,service_name,version,status,started_at,last_seen_at,expires_at
    FROM service_instances WHERE service_name='ai-runtime' ORDER BY instance_id`)
	if err != nil {
		return AIRuntimeHealth{}, err
	}
	instances := []ServiceInstance{}
	for rows.Next() {
		item, scanErr := scanServiceInstance(rows)
		if scanErr != nil {
			rows.Close()
			return AIRuntimeHealth{}, scanErr
		}
		instances = append(instances, item)
	}
	if err := rows.Close(); err != nil {
		return AIRuntimeHealth{}, err
	}
	normalizeInstanceLiveness(instances, now)
	checksRows, err := p.db.QueryContext(ctx, `SELECT id::text,instance_id,check_name,status,detail,checked_at
    FROM health_checks WHERE checked_at > NOW() - INTERVAL '10 minutes' ORDER BY checked_at DESC LIMIT 50`)
	if err != nil {
		return AIRuntimeHealth{}, err
	}
	checks := []HealthCheck{}
	for checksRows.Next() {
		item, scanErr := scanHealthCheck(checksRows)
		if scanErr != nil {
			checksRows.Close()
			return AIRuntimeHealth{}, scanErr
		}
		checks = append(checks, item)
	}
	if err := checksRows.Close(); err != nil {
		return AIRuntimeHealth{}, err
	}
	return AIRuntimeHealth{Service: runtimeServiceName, Status: aggregateRuntimeHealth(runtimeMode, instances), InstanceID: instanceID, CheckedAt: now, Instances: instances, Checks: checks}, nil
}

func (p *postgresConversationStore) recordDistributedTestRun(item DistributedTestRun) (DistributedTestRun, bool, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	existing, err := scanDistributedTestRun(p.db.QueryRowContext(ctx, `SELECT id::text,test_key,status,instance_a,instance_b,cursor_before,cursor_after,
    error_code,details,started_at,completed_at,created_at,updated_at FROM test_runs WHERE id=$1`, item.ID))
	if err == nil {
		if !sameDistributedTestRun(existing, item) {
			if !canAdvanceDistributedTestRun(existing, item) {
				return DistributedTestRun{}, false, errors.New("idempotency_conflict")
			}
			item.StartedAt = existing.StartedAt
			item.CreatedAt = existing.CreatedAt
			details, marshalErr := json.Marshal(item.Details)
			if marshalErr != nil {
				return DistributedTestRun{}, false, marshalErr
			}
			_, updateErr := p.db.ExecContext(ctx, `UPDATE test_runs SET status=$2,cursor_after=$3,error_code=$4,details=$5::jsonb,completed_at=$6,updated_at=$7 WHERE id=$1`, item.ID, item.Status, item.CursorAfter, item.ErrorCode, string(details), item.CompletedAt, item.UpdatedAt)
			if updateErr != nil {
				return DistributedTestRun{}, false, updateErr
			}
			return item, false, nil
		}
		return existing, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return DistributedTestRun{}, false, err
	}
	details, err := json.Marshal(item.Details)
	if err != nil {
		return DistributedTestRun{}, false, err
	}
	_, err = p.db.ExecContext(ctx, `INSERT INTO test_runs
    (id,test_key,status,instance_a,instance_b,cursor_before,cursor_after,error_code,details,started_at,completed_at,created_at,updated_at)
    VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,$10,$11,$12,$13)`, item.ID, item.TestKey, item.Status, item.InstanceA, item.InstanceB,
		item.CursorBefore, item.CursorAfter, item.ErrorCode, string(details), item.StartedAt, item.CompletedAt, item.CreatedAt, item.UpdatedAt)
	if err != nil {
		return DistributedTestRun{}, false, err
	}
	return item, true, nil
}
