package identity

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

func (p *postgresConversationStore) loadModelCatalog() ([]ModelCatalogEntry, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	rows, err := p.db.QueryContext(ctx, `SELECT m.id,m.name,m.enabled AND p.enabled,m.description,
    p.id,p.name,m.purpose,m.speed_tier,m.upstream_model,m.sort_order,m.version,m.updated_at
    FROM models m JOIN model_providers p ON p.id=m.provider_id
    ORDER BY p.sort_order,p.id,m.sort_order,m.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ModelCatalogEntry{}
	indexes := map[string]int{}
	for rows.Next() {
		var item ModelCatalogEntry
		if err := rows.Scan(&item.ID, &item.Name, &item.Enabled, &item.Description, &item.ProviderID, &item.ProviderName,
			&item.Purpose, &item.SpeedTier, &item.UpstreamModel, &item.SortOrder, &item.Version, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Capabilities = []ModelCapabilityTag{}
		item.ReasoningProfiles = []string{}
		indexes[item.ID] = len(items)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	profileRows, err := p.db.QueryContext(ctx, `SELECT model_id,profile_id,label,ordinal,enabled,upstream_parameters,version,updated_at
    FROM reasoning_profiles ORDER BY model_id,ordinal,profile_id`)
	if err != nil {
		return nil, err
	}
	for profileRows.Next() {
		var mapping ReasoningProfileMapping
		var raw []byte
		if err := profileRows.Scan(&mapping.ModelID, &mapping.ProfileID, &mapping.Label, &mapping.Ordinal, &mapping.Enabled, &raw, &mapping.Version, &mapping.UpdatedAt); err != nil {
			profileRows.Close()
			return nil, err
		}
		if err := json.Unmarshal(raw, &mapping.UpstreamParameters); err != nil {
			profileRows.Close()
			return nil, err
		}
		if index, ok := indexes[mapping.ModelID]; ok {
			items[index].ReasoningMappings = append(items[index].ReasoningMappings, mapping)
			if mapping.Enabled {
				items[index].ReasoningProfiles = append(items[index].ReasoningProfiles, mapping.ProfileID)
			}
		}
	}
	if err := profileRows.Close(); err != nil {
		return nil, err
	}
	capabilityRows, err := p.db.QueryContext(ctx, `SELECT latest.model_id,latest.capability_id,d.label,latest.probed_at
    FROM (
      SELECT DISTINCT ON (model_id,capability_id) model_id,capability_id,status,probed_at,expires_at,created_at
      FROM model_capability_probe_results
      ORDER BY model_id,capability_id,probed_at DESC,created_at DESC
    ) latest
    JOIN model_capability_definitions d ON d.id=latest.capability_id
    WHERE latest.status='passed' AND (latest.expires_at IS NULL OR latest.expires_at>NOW())
    ORDER BY latest.model_id,d.sort_order,d.id`)
	if err != nil {
		return nil, err
	}
	defer capabilityRows.Close()
	for capabilityRows.Next() {
		var modelID string
		var tag ModelCapabilityTag
		if err := capabilityRows.Scan(&modelID, &tag.ID, &tag.Label, &tag.ProbedAt); err != nil {
			return nil, err
		}
		if index, ok := indexes[modelID]; ok {
			items[index].Capabilities = append(items[index].Capabilities, tag)
		}
	}
	return items, capabilityRows.Err()
}

func (p *postgresConversationStore) reasoningProfiles(modelID string, mobile bool) ([]ReasoningProfileMapping, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	query := `SELECT r.model_id,r.profile_id,r.label,r.ordinal,r.enabled,r.upstream_parameters,r.version,r.updated_at
    FROM reasoning_profiles r JOIN models m ON m.id=r.model_id JOIN model_providers p ON p.id=m.provider_id
    WHERE r.model_id=$1`
	if mobile {
		query += ` AND r.enabled AND m.enabled AND p.enabled`
	}
	query += ` ORDER BY r.ordinal,r.profile_id`
	rows, err := p.db.QueryContext(ctx, query, strings.TrimSpace(modelID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	profiles := []ReasoningProfileMapping{}
	for rows.Next() {
		var item ReasoningProfileMapping
		var raw []byte
		if err := rows.Scan(&item.ModelID, &item.ProfileID, &item.Label, &item.Ordinal, &item.Enabled, &raw, &item.Version, &item.UpdatedAt); err != nil {
			return nil, err
		}
		if !mobile {
			if err := json.Unmarshal(raw, &item.UpstreamParameters); err != nil {
				return nil, err
			}
		}
		profiles = append(profiles, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(profiles) == 0 {
		return nil, errors.New("model_not_found")
	}
	return profiles, nil
}

func (p *postgresConversationStore) resolveModelSelection(modelID, profileID string) (ModelSelection, error) {
	modelID = valueOr(strings.TrimSpace(modelID), "ylven-default")
	profileID = valueOr(strings.TrimSpace(profileID), "auto")
	ctx, cancel := databaseContext()
	defer cancel()
	var selection ModelSelection
	var raw []byte
	err := p.db.QueryRowContext(ctx, `SELECT m.id,m.upstream_model,r.profile_id,r.upstream_parameters
    FROM models m JOIN model_providers p ON p.id=m.provider_id
    JOIN reasoning_profiles r ON r.model_id=m.id
    WHERE m.id=$1 AND r.profile_id=$2 AND m.enabled AND p.enabled AND r.enabled`, modelID, profileID).
		Scan(&selection.CatalogModelID, &selection.ProviderModel, &selection.ReasoningProfile, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		var modelExists bool
		_ = p.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM models WHERE id=$1)`, modelID).Scan(&modelExists)
		if !modelExists {
			if modelID == "ylven-default" {
				return defaultAISelection(), nil
			}
			var catalogCount int
			_ = p.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM models`).Scan(&catalogCount)
			if catalogCount == 1 {
				return ModelSelection{CatalogModelID: modelID, ProviderModel: modelID, ReasoningProfile: profileID, ReasoningParameters: map[string]any{}}, nil
			}
			return ModelSelection{}, errors.New("model_not_found")
		}
		return ModelSelection{}, errors.New("reasoning_profile_unsupported")
	}
	if err != nil {
		return ModelSelection{}, err
	}
	if err := json.Unmarshal(raw, &selection.ReasoningParameters); err != nil {
		return ModelSelection{}, err
	}
	if err := validateUpstreamReasoningParameters(selection.ReasoningParameters); err != nil {
		return ModelSelection{}, err
	}
	return selection, nil
}

func (p *postgresConversationStore) modelCatalogSnapshot() (ModelCatalogSnapshot, error) {
	models, err := p.loadModelCatalog()
	if err != nil {
		return ModelCatalogSnapshot{}, err
	}
	ctx, cancel := databaseContext()
	defer cancel()
	snapshot := ModelCatalogSnapshot{Models: models, Providers: []ModelProvider{}, CapabilityProbes: []CapabilityProbeResult{}, ReasoningMappings: []ReasoningProfileMapping{}, Audit: []ModelCatalogAuditEvent{}}
	providerRows, err := p.db.QueryContext(ctx, `SELECT id,name,enabled,sort_order,version,updated_at FROM model_providers ORDER BY sort_order,id`)
	if err != nil {
		return ModelCatalogSnapshot{}, err
	}
	for providerRows.Next() {
		var item ModelProvider
		if err := providerRows.Scan(&item.ID, &item.Name, &item.Enabled, &item.SortOrder, &item.Version, &item.UpdatedAt); err != nil {
			providerRows.Close()
			return ModelCatalogSnapshot{}, err
		}
		snapshot.Providers = append(snapshot.Providers, item)
	}
	if err := providerRows.Close(); err != nil {
		return ModelCatalogSnapshot{}, err
	}
	probeRows, err := p.db.QueryContext(ctx, `SELECT r.id::text,r.model_id,r.capability_id,d.label,r.status,r.probe_source,r.evidence_reference,
    r.probed_at,r.expires_at,r.recorded_by,r.created_at FROM model_capability_probe_results r
    JOIN model_capability_definitions d ON d.id=r.capability_id ORDER BY r.probed_at DESC,r.created_at DESC LIMIT 500`)
	if err != nil {
		return ModelCatalogSnapshot{}, err
	}
	for probeRows.Next() {
		var item CapabilityProbeResult
		var expiresAt sql.NullTime
		if err := probeRows.Scan(&item.ID, &item.ModelID, &item.CapabilityID, &item.CapabilityLabel, &item.Status,
			&item.ProbeSource, &item.EvidenceReference, &item.ProbedAt, &expiresAt, &item.RecordedBy, &item.CreatedAt); err != nil {
			probeRows.Close()
			return ModelCatalogSnapshot{}, err
		}
		if expiresAt.Valid {
			item.ExpiresAt = &expiresAt.Time
		}
		snapshot.CapabilityProbes = append(snapshot.CapabilityProbes, item)
	}
	if err := probeRows.Close(); err != nil {
		return ModelCatalogSnapshot{}, err
	}
	for _, model := range models {
		snapshot.ReasoningMappings = append(snapshot.ReasoningMappings, model.ReasoningMappings...)
	}
	auditRows, err := p.db.QueryContext(ctx, `SELECT id::text,actor_id,action,resource_type,resource_id,before_value,after_value,created_at
    FROM model_catalog_audit_events ORDER BY created_at DESC LIMIT 200`)
	if err != nil {
		return ModelCatalogSnapshot{}, err
	}
	defer auditRows.Close()
	for auditRows.Next() {
		var item ModelCatalogAuditEvent
		var beforeRaw, afterRaw []byte
		if err := auditRows.Scan(&item.ID, &item.ActorID, &item.Action, &item.ResourceType, &item.ResourceID, &beforeRaw, &afterRaw, &item.CreatedAt); err != nil {
			return ModelCatalogSnapshot{}, err
		}
		if len(beforeRaw) > 0 {
			_ = json.Unmarshal(beforeRaw, &item.BeforeValue)
		}
		if len(afterRaw) > 0 {
			_ = json.Unmarshal(afterRaw, &item.AfterValue)
		}
		snapshot.Audit = append(snapshot.Audit, item)
	}
	return snapshot, auditRows.Err()
}

func (p *postgresConversationStore) upsertModelProvider(input ModelProvider, actorID string) (ModelProvider, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return ModelProvider{}, err
	}
	defer tx.Rollback()
	var current ModelProvider
	err = tx.QueryRowContext(ctx, `SELECT id,name,enabled,sort_order,version,updated_at FROM model_providers WHERE id=$1 FOR UPDATE`, input.ID).
		Scan(&current.ID, &current.Name, &current.Enabled, &current.SortOrder, &current.Version, &current.UpdatedAt)
	now := time.Now().UTC()
	action := "update"
	if errors.Is(err, sql.ErrNoRows) {
		if input.Version != 0 && input.Version != 1 {
			return ModelProvider{}, errors.New("version_conflict")
		}
		input.Version = 1
		input.UpdatedAt = now
		_, err = tx.ExecContext(ctx, `INSERT INTO model_providers(id,name,enabled,sort_order,version,updated_at) VALUES ($1,$2,$3,$4,1,$5)`, input.ID, input.Name, input.Enabled, input.SortOrder, now)
		action = "create"
	} else if err != nil {
		return ModelProvider{}, err
	} else {
		if input.Version != current.Version {
			return ModelProvider{}, modelCatalogConflictError("provider", input.Version, current.Version)
		}
		input.Version = current.Version + 1
		input.UpdatedAt = now
		_, err = tx.ExecContext(ctx, `UPDATE model_providers SET name=$2,enabled=$3,sort_order=$4,version=$5,updated_at=$6 WHERE id=$1`, input.ID, input.Name, input.Enabled, input.SortOrder, input.Version, now)
	}
	if err != nil {
		return ModelProvider{}, err
	}
	if err := insertModelCatalogAudit(ctx, tx, actorID, action, "provider", input.ID, current, input); err != nil {
		return ModelProvider{}, err
	}
	return input, tx.Commit()
}

func (p *postgresConversationStore) upsertModelCatalogEntry(input ModelCatalogEntry, actorID string) (ModelCatalogEntry, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return ModelCatalogEntry{}, err
	}
	defer tx.Rollback()
	var providerExists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM model_providers WHERE id=$1)`, input.ProviderID).Scan(&providerExists); err != nil {
		return ModelCatalogEntry{}, err
	}
	if !providerExists {
		return ModelCatalogEntry{}, errors.New("model_provider_not_found")
	}
	var current ModelCatalogEntry
	err = tx.QueryRowContext(ctx, `SELECT id,name,enabled,description,provider_id,purpose,speed_tier,upstream_model,sort_order,version,updated_at FROM models WHERE id=$1 FOR UPDATE`, input.ID).
		Scan(&current.ID, &current.Name, &current.Enabled, &current.Description, &current.ProviderID, &current.Purpose, &current.SpeedTier, &current.UpstreamModel, &current.SortOrder, &current.Version, &current.UpdatedAt)
	now := time.Now().UTC()
	action := "update"
	if errors.Is(err, sql.ErrNoRows) {
		if input.Version != 0 && input.Version != 1 {
			return ModelCatalogEntry{}, errors.New("version_conflict")
		}
		input.Version = 1
		input.UpdatedAt = now
		_, err = tx.ExecContext(ctx, `INSERT INTO models(id,name,enabled,description,provider_id,purpose,speed_tier,upstream_model,sort_order,version,updated_at)
      VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,1,$10)`, input.ID, input.Name, input.Enabled, input.Description, input.ProviderID, input.Purpose, input.SpeedTier, input.UpstreamModel, input.SortOrder, now)
		if err == nil {
			_, err = tx.ExecContext(ctx, `INSERT INTO reasoning_profiles(model_id,profile_id,label,ordinal,enabled,upstream_parameters,version,updated_at)
        VALUES ($1,'auto','自动',0,TRUE,'{}'::jsonb,1,$2)`, input.ID, now)
		}
		action = "create"
	} else if err != nil {
		return ModelCatalogEntry{}, err
	} else {
		if input.Version != current.Version {
			return ModelCatalogEntry{}, modelCatalogConflictError("model", input.Version, current.Version)
		}
		input.Version = current.Version + 1
		input.UpdatedAt = now
		_, err = tx.ExecContext(ctx, `UPDATE models SET name=$2,enabled=$3,description=$4,provider_id=$5,purpose=$6,speed_tier=$7,
      upstream_model=$8,sort_order=$9,version=$10,updated_at=$11 WHERE id=$1`, input.ID, input.Name, input.Enabled, input.Description,
			input.ProviderID, input.Purpose, input.SpeedTier, input.UpstreamModel, input.SortOrder, input.Version, now)
	}
	if err != nil {
		return ModelCatalogEntry{}, err
	}
	if err := insertModelCatalogAudit(ctx, tx, actorID, action, "model", input.ID, current, input); err != nil {
		return ModelCatalogEntry{}, err
	}
	return input, tx.Commit()
}

func (p *postgresConversationStore) recordCapabilityProbe(input CapabilityProbeResult, actorID string) (CapabilityProbeResult, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return CapabilityProbeResult{}, err
	}
	defer tx.Rollback()
	var label string
	if err := tx.QueryRowContext(ctx, `SELECT d.label FROM model_capability_definitions d JOIN models m ON m.id=$1 WHERE d.id=$2`, input.ModelID, input.CapabilityID).Scan(&label); errors.Is(err, sql.ErrNoRows) {
		return CapabilityProbeResult{}, errors.New("model_or_capability_not_found")
	} else if err != nil {
		return CapabilityProbeResult{}, err
	}
	id, err := newDatabaseUUID()
	if err != nil {
		return CapabilityProbeResult{}, err
	}
	input.ID = id
	input.RecordedBy = actorID
	input.CapabilityLabel = label
	input.CreatedAt = time.Now().UTC()
	_, err = tx.ExecContext(ctx, `INSERT INTO model_capability_probe_results
    (id,model_id,capability_id,status,probe_source,evidence_reference,probed_at,expires_at,recorded_by,created_at)
    VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, input.ID, input.ModelID, input.CapabilityID, input.Status,
		input.ProbeSource, input.EvidenceReference, input.ProbedAt, input.ExpiresAt, actorID, input.CreatedAt)
	if err != nil {
		return CapabilityProbeResult{}, err
	}
	if err := insertModelCatalogAudit(ctx, tx, actorID, "record_probe", "capability", input.ModelID+":"+input.CapabilityID, nil, input); err != nil {
		return CapabilityProbeResult{}, err
	}
	return input, tx.Commit()
}

func (p *postgresConversationStore) upsertReasoningProfile(input ReasoningProfileMapping, actorID string) (ReasoningProfileMapping, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return ReasoningProfileMapping{}, err
	}
	defer tx.Rollback()
	parameters, err := json.Marshal(copyJSONMap(input.UpstreamParameters))
	if err != nil {
		return ReasoningProfileMapping{}, err
	}
	var current ReasoningProfileMapping
	var currentRaw []byte
	err = tx.QueryRowContext(ctx, `SELECT model_id,profile_id,label,ordinal,enabled,upstream_parameters,version,updated_at
    FROM reasoning_profiles WHERE model_id=$1 AND profile_id=$2 FOR UPDATE`, input.ModelID, input.ProfileID).
		Scan(&current.ModelID, &current.ProfileID, &current.Label, &current.Ordinal, &current.Enabled, &currentRaw, &current.Version, &current.UpdatedAt)
	if err == nil {
		_ = json.Unmarshal(currentRaw, &current.UpstreamParameters)
	}
	now := time.Now().UTC()
	action := "update"
	if errors.Is(err, sql.ErrNoRows) {
		var modelExists bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM models WHERE id=$1)`, input.ModelID).Scan(&modelExists); err != nil {
			return ReasoningProfileMapping{}, err
		}
		if !modelExists {
			return ReasoningProfileMapping{}, errors.New("model_not_found")
		}
		if input.Version != 0 && input.Version != 1 {
			return ReasoningProfileMapping{}, errors.New("version_conflict")
		}
		input.Version = 1
		input.UpdatedAt = now
		_, err = tx.ExecContext(ctx, `INSERT INTO reasoning_profiles(model_id,profile_id,label,ordinal,enabled,upstream_parameters,version,updated_at)
      VALUES ($1,$2,$3,$4,$5,$6::jsonb,1,$7)`, input.ModelID, input.ProfileID, input.Label, input.Ordinal, input.Enabled, string(parameters), now)
		action = "create"
	} else if err != nil {
		return ReasoningProfileMapping{}, err
	} else {
		if input.Version != current.Version {
			return ReasoningProfileMapping{}, modelCatalogConflictError("reasoning_profile", input.Version, current.Version)
		}
		input.Version = current.Version + 1
		input.UpdatedAt = now
		_, err = tx.ExecContext(ctx, `UPDATE reasoning_profiles SET label=$3,ordinal=$4,enabled=$5,upstream_parameters=$6::jsonb,
      version=$7,updated_at=$8 WHERE model_id=$1 AND profile_id=$2`, input.ModelID, input.ProfileID, input.Label, input.Ordinal, input.Enabled, string(parameters), input.Version, now)
	}
	if err != nil {
		return ReasoningProfileMapping{}, err
	}
	if err := insertModelCatalogAudit(ctx, tx, actorID, action, "reasoning_profile", input.ModelID+":"+input.ProfileID, current, input); err != nil {
		return ReasoningProfileMapping{}, err
	}
	return input, tx.Commit()
}

func insertModelCatalogAudit(ctx context.Context, tx *sql.Tx, actorID, action, resourceType, resourceID string, before, after any) error {
	id, err := newDatabaseUUID()
	if err != nil {
		return err
	}
	beforeRaw, err := json.Marshal(before)
	if err != nil {
		return err
	}
	afterRaw, err := json.Marshal(after)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO model_catalog_audit_events
    (id,actor_id,action,resource_type,resource_id,before_value,after_value,created_at)
    VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7::jsonb,NOW())`, id, actorID, action, resourceType, resourceID, string(beforeRaw), string(afterRaw))
	return err
}

func modelCatalogConflictError(resource string, expected, actual int64) error {
	return fmt.Errorf("%s_version_conflict: expected %d actual %d", resource, expected, actual)
}
