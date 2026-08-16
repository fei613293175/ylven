package identity

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/fei613293175/ylven/backend/db/migrations"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const conversationSQLTimeout = 15 * time.Second

type postgresConversationStore struct {
	db    *sql.DB
	store *Store
}

func (s *Store) EnablePostgresFromEnvironment(ctx context.Context) error {
	dsn, err := envOrFile("DATABASE_URL")
	if err != nil {
		return err
	}
	return s.EnablePostgres(ctx, dsn)
}

func (s *Store) EnablePostgres(ctx context.Context, dsn string) error {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return errors.New("DATABASE_URL is required for durable conversation storage")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open conversation database: %w", err)
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("connect conversation database: %w", err)
	}
	if err := migrations.Apply(ctx, db); err != nil {
		_ = db.Close()
		return err
	}
	persistence := &postgresConversationStore{db: db, store: s}
	if err := s.importLegacyConversations(ctx, persistence); err != nil {
		_ = db.Close()
		return fmt.Errorf("import legacy conversations: %w", err)
	}
	s.mu.Lock()
	legacyConfig := s.data.HomeConfig
	legacyModels := append([]ModelCatalogEntry(nil), s.data.ModelCatalog...)
	s.mu.Unlock()
	if err := persistence.syncProductConfiguration(ctx, legacyConfig, legacyModels); err != nil {
		_ = db.Close()
		return fmt.Errorf("sync product configuration: %w", err)
	}
	config, err := persistence.homeConfig()
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("load product configuration: %w", err)
	}
	models, err := persistence.modelCatalog()
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("load model catalog: %w", err)
	}
	s.mu.Lock()
	s.data.HomeConfig = config
	s.data.ModelCatalog = models
	s.mu.Unlock()
	s.conversationSQL = persistence
	return nil
}

func (s *Store) Close() error {
	if s.conversationSQL == nil {
		return nil
	}
	return s.conversationSQL.db.Close()
}

type sqlScanner interface {
	Scan(...any) error
}

const conversationColumns = `id::text, user_id::text, title, title_source, title_locked,
  COALESCE(active_branch_id::text, ''), COALESCE(summary_through_message_id::text, ''),
	status, created_at, updated_at, archived_at, deleted_at, temporary,
	COALESCE(default_model_id, 'ylven-default'), COALESCE(default_reasoning_profile, 'auto'),
	COALESCE(ai_settings_version, 1), COALESCE(ai_settings_overridden, FALSE)`

func scanConversation(scanner sqlScanner) (Conversation, error) {
	var item Conversation
	var archivedAt, deletedAt sql.NullTime
	err := scanner.Scan(&item.ID, &item.UserID, &item.Title, &item.TitleSource, &item.TitleLocked,
		&item.ActiveBranchID, &item.SummaryThroughMessageID, &item.Status, &item.CreatedAt, &item.UpdatedAt,
		&archivedAt, &deletedAt, &item.Temporary, &item.DefaultModelID, &item.DefaultReasoningProfile, &item.AISettingsVersion, &item.AISettingsOverridden)
	if archivedAt.Valid {
		item.ArchivedAt = &archivedAt.Time
	}
	if deletedAt.Valid {
		item.DeletedAt = &deletedAt.Time
	}
	return item, err
}

const messageColumns = `id::text, conversation_id::text, user_id::text, branch_id::text, sequence,
  COALESCE(parent_message_id::text, ''), COALESCE(comparison_group_id::text, ''), role, body,
  status, created_at, completed_at`

func scanMessage(scanner sqlScanner) (Message, error) {
	var item Message
	var completedAt sql.NullTime
	err := scanner.Scan(&item.ID, &item.ConversationID, &item.UserID, &item.BranchID, &item.Sequence,
		&item.ParentMessageID, &item.ComparisonGroupID, &item.Role, &item.Body, &item.Status,
		&item.CreatedAt, &completedAt)
	if completedAt.Valid {
		item.CompletedAt = &completedAt.Time
	}
	return item, err
}

const runColumns = `id::text, conversation_id::text, user_id::text, COALESCE(user_message_id::text, ''),
  COALESCE(assistant_message_id::text, ''), model, COALESCE(NULLIF(provider_model,''),model),
  COALESCE(NULLIF(reasoning_profile,''),'auto'), COALESCE(reasoning_parameters,'{}'::jsonb),
  branch_id::text, COALESCE(idempotency_key, ''),
  COALESCE(context_build_id::text, ''), status, COALESCE(error_code, ''), provider,
  provider_continuation_used, provider_continuation_fallback, started_at, completed_at, latency_ms,
  cursor, created_at, updated_at`

func scanRun(scanner sqlScanner) (MessageRun, error) {
	var item MessageRun
	var completedAt sql.NullTime
	var reasoningParameters []byte
	err := scanner.Scan(&item.ID, &item.ConversationID, &item.UserID, &item.UserMessageID,
		&item.AssistantMessageID, &item.Model, &item.ProviderModel, &item.ReasoningProfile, &reasoningParameters,
		&item.BranchID, &item.IdempotencyKey,
		&item.ContextBuildID, &item.Status, &item.ErrorCode, &item.Provider,
		&item.ContinuationUsed, &item.ContinuationFallback, &item.StartedAt, &completedAt,
		&item.LatencyMs, &item.Cursor, &item.CreatedAt, &item.UpdatedAt)
	if completedAt.Valid {
		item.CompletedAt = &completedAt.Time
	}
	if len(reasoningParameters) > 0 && err == nil {
		err = json.Unmarshal(reasoningParameters, &item.ReasoningParameters)
	}
	if item.ReasoningParameters == nil {
		item.ReasoningParameters = map[string]any{}
	}
	return item, err
}

func scanConversationSummary(scanner sqlScanner) (ConversationSummary, error) {
	var item ConversationSummary
	var structured []byte
	err := scanner.Scan(&item.ID, &item.ConversationID, &item.BranchID, &item.SourceFirstSequence,
		&item.SourceLastSequence, &item.SourceFirstMessageID, &item.SourceLastMessageID, &structured,
		&item.SummaryText, &item.SummaryModel, &item.SummaryVersion)
	if err == nil {
		item.StructuredState = map[string]any{}
		if len(structured) > 0 {
			err = json.Unmarshal(structured, &item.StructuredState)
		}
	}
	return item, err
}

func scanContextCompaction(scanner sqlScanner) (ContextCompaction, error) {
	var item ContextCompaction
	var startedAt, completedAt sql.NullTime
	err := scanner.Scan(&item.ID, &item.JobID, &item.ConversationID, &item.BranchID,
		&item.IdempotencyKey, &item.SourceFirstSequence, &item.SourceLastSequence,
		&item.SourceFirstMessageID, &item.SourceLastMessageID, &item.SummaryID,
		&item.SummaryVersion, &item.Status, &item.ErrorCode, &item.Attempts,
		&item.CreatedAt, &item.UpdatedAt, &startedAt, &completedAt)
	if startedAt.Valid {
		item.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		item.CompletedAt = &completedAt.Time
	}
	return item, err
}

const contextCompactionColumns = `cc.id::text,cc.job_id::text,cc.conversation_id::text,cc.branch_id::text,
  cc.idempotency_key,cc.source_first_sequence,cc.source_last_sequence,cc.source_first_message_id::text,
  cc.source_last_message_id::text,COALESCE(cc.summary_id::text,''),cc.summary_version,cc.status,
  COALESCE(cc.error_code,''),j.attempts,cc.created_at,cc.updated_at,cc.started_at,cc.completed_at`

func databaseContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), conversationSQLTimeout)
}

func newDatabaseUUID() (string, error) {
	raw, err := randomToken(16)
	if err != nil {
		return "", err
	}
	return raw[0:8] + "-" + raw[8:12] + "-" + raw[12:16] + "-" + raw[16:20] + "-" + raw[20:32], nil
}

func (p *postgresConversationStore) syncProductConfiguration(ctx context.Context, config HomeConfig, models []ModelCatalogEntry) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, model := range models {
		model = normalizeCatalogEntry(model)
		if strings.TrimSpace(model.ID) == "" || strings.TrimSpace(model.Name) == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO models (id,name,enabled,description,provider_id,upstream_model,purpose,speed_tier,sort_order,version,updated_at)
      VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) ON CONFLICT (id) DO NOTHING`, model.ID, model.Name, model.Enabled,
			model.Description, model.ProviderID, model.UpstreamModel, model.Purpose, model.SpeedTier, model.SortOrder, model.Version, model.UpdatedAt); err != nil {
			return err
		}
		profiles := model.ReasoningProfiles
		if len(profiles) == 0 {
			profiles = []string{"auto"}
		}
		for ordinal, profile := range profiles {
			label := map[string]string{"auto": "自动", "quick": "快速", "standard": "标准", "deep": "深度"}[profile]
			if _, err := tx.ExecContext(ctx, `INSERT INTO reasoning_profiles (model_id,profile_id,label,ordinal,enabled,upstream_parameters,version,updated_at)
        VALUES ($1,$2,$3,$4,TRUE,'{}'::jsonb,1,$5) ON CONFLICT (model_id,profile_id) DO NOTHING`, model.ID, profile, label, ordinal, model.UpdatedAt); err != nil {
				return err
			}
		}
	}
	if config.Version > 1 {
		value, err := json.Marshal(config)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE system_configs SET value=$1::jsonb,version=$2,updated_at=$3
      WHERE config_key='home_config' AND version < $2`, string(value), config.Version, config.UpdatedAt); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (p *postgresConversationStore) homeConfig() (HomeConfig, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	var raw []byte
	var version int64
	var updatedAt time.Time
	if err := p.db.QueryRowContext(ctx, `SELECT value,version,updated_at FROM system_configs WHERE config_key='home_config'`).Scan(&raw, &version, &updatedAt); err != nil {
		return HomeConfig{}, err
	}
	var config HomeConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return HomeConfig{}, err
	}
	config.Version = version
	config.UpdatedAt = updatedAt
	return normalizeHomeConfig(config)
}

func (p *postgresConversationStore) updateHomeConfig(config HomeConfig, actorID string) (HomeConfig, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return HomeConfig{}, err
	}
	defer tx.Rollback()
	var version int64
	if err := tx.QueryRowContext(ctx, `SELECT version FROM system_configs WHERE config_key='home_config' FOR UPDATE`).Scan(&version); err != nil {
		return HomeConfig{}, err
	}
	config.Version = version + 1
	config.UpdatedAt = time.Now().UTC()
	raw, err := json.Marshal(config)
	if err != nil {
		return HomeConfig{}, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE system_configs SET value=$1::jsonb,version=$2,updated_at=$3 WHERE config_key='home_config'`, string(raw), config.Version, config.UpdatedAt); err != nil {
		return HomeConfig{}, err
	}
	auditID, err := newDatabaseUUID()
	if err != nil {
		return HomeConfig{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO identity_audit_events (id,event_type,actor_id,details,created_at)
    VALUES ($1,'home_config_updated',CASE WHEN $2 ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$' THEN $2::uuid ELSE NULL END,
      jsonb_build_object('config_key','home_config','version',$3::bigint,'actor_reference',NULLIF($2,'')),$4)`, auditID, actorID, config.Version, config.UpdatedAt); err != nil {
		return HomeConfig{}, err
	}
	if err := tx.Commit(); err != nil {
		return HomeConfig{}, err
	}
	return config, nil
}

func (p *postgresConversationStore) homeConfigAudit() ([]AuditEvent, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	rows, err := p.db.QueryContext(ctx, `SELECT id::text,event_type,COALESCE(details->>'actor_reference',''),created_at
    FROM identity_audit_events WHERE event_type='home_config_updated' ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []AuditEvent{}
	for rows.Next() {
		var event AuditEvent
		if err := rows.Scan(&event.ID, &event.Type, &event.SessionID, &event.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (p *postgresConversationStore) modelCatalog() ([]ModelCatalogEntry, error) {
	return p.loadModelCatalog()
}

func (s *Store) importLegacyConversations(ctx context.Context, persistence *postgresConversationStore) error {
	s.mu.Lock()
	conversations := make([]Conversation, 0, len(s.data.Conversations))
	for _, item := range s.data.Conversations {
		conversations = append(conversations, item)
	}
	messages := make([]Message, 0, len(s.data.Messages))
	for _, item := range s.data.Messages {
		messages = append(messages, item)
	}
	runs := make([]MessageRun, 0, len(s.data.Runs))
	for _, item := range s.data.Runs {
		runs = append(runs, item)
	}
	events := make(map[string][]RunEvent, len(s.data.RunEvents))
	for runID, values := range s.data.RunEvents {
		events[runID] = append([]RunEvent(nil), values...)
	}
	drafts := make([]Draft, 0, len(s.data.Drafts))
	for _, item := range s.data.Drafts {
		drafts = append(drafts, item)
	}
	s.mu.Unlock()
	if len(conversations) == 0 {
		return nil
	}
	tx, err := persistence.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, conversation := range conversations {
		branchID := valueOr(conversation.ActiveBranchID, conversation.ID)
		titleSource := conversation.TitleSource
		if titleSource == "" {
			if conversation.Title == "" || conversation.Title == "新对话" {
				titleSource = "AUTO_TEMP"
			} else {
				titleSource = "USER"
			}
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO conversations
      (id,user_id,title,title_source,title_locked,active_branch_id,summary_through_message_id,status,created_at,updated_at,archived_at,deleted_at,temporary)
      VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) ON CONFLICT (id) DO NOTHING`,
			conversation.ID, conversation.UserID, valueOr(conversation.Title, "新对话"), titleSource,
			conversation.TitleLocked || titleSource == "USER", branchID, nullIfEmpty(conversation.SummaryThroughMessageID),
			valueOr(conversation.Status, "active"), conversation.CreatedAt, conversation.UpdatedAt,
			conversation.ArchivedAt, conversation.DeletedAt, conversation.Temporary)
		if err == nil {
			_, err = tx.ExecContext(ctx, `INSERT INTO conversation_branches (id,conversation_id,status,created_at,updated_at)
        VALUES ($1,$2,'active',$3,$4) ON CONFLICT (id) DO NOTHING`, branchID, conversation.ID, conversation.CreatedAt, conversation.UpdatedAt)
		}
		if err != nil {
			return err
		}
	}
	sort.SliceStable(messages, func(i, j int) bool {
		if messages[i].ConversationID == messages[j].ConversationID {
			return messages[i].CreatedAt.Before(messages[j].CreatedAt)
		}
		return messages[i].ConversationID < messages[j].ConversationID
	})
	nextSequence := map[string]int64{}
	for _, message := range messages {
		branchID := valueOr(message.BranchID, message.ConversationID)
		key := message.ConversationID + "|" + branchID
		sequence := message.Sequence
		if sequence <= 0 {
			nextSequence[key]++
			sequence = nextSequence[key]
		} else if sequence > nextSequence[key] {
			nextSequence[key] = sequence
		}
		status := valueOr(message.Status, "completed")
		completedAt := message.CompletedAt
		if completedAt == nil && status == "completed" {
			value := message.CreatedAt
			completedAt = &value
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO messages
      (id,conversation_id,user_id,branch_id,sequence,parent_message_id,comparison_group_id,role,body,status,created_at,completed_at)
      VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) ON CONFLICT (id) DO NOTHING`, message.ID,
			message.ConversationID, message.UserID, branchID, sequence, nullIfEmpty(message.ParentMessageID),
			nullIfEmpty(message.ComparisonGroupID), message.Role, message.Body, status, message.CreatedAt, completedAt)
		if err == nil {
			_, err = tx.ExecContext(ctx, `INSERT INTO message_parts (id,message_id,ordinal,kind,text_content,created_at)
        VALUES ($1,$1,0,'TEXT',$2,$3) ON CONFLICT (id) DO NOTHING`, message.ID, message.Body, message.CreatedAt)
		}
		if err != nil {
			return err
		}
	}
	for _, run := range runs {
		startedAt := run.StartedAt
		if startedAt.IsZero() {
			startedAt = run.CreatedAt
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO message_runs
      (id,conversation_id,user_id,user_message_id,assistant_message_id,model,status,cursor,error_code,created_at,updated_at,
       branch_id,idempotency_key,context_build_id,provider,started_at,completed_at,latency_ms,provider_continuation_used,provider_continuation_fallback,
       reasoning_profile,reasoning_parameters,provider_model)
      VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22::jsonb,$23)
      ON CONFLICT (id) DO NOTHING`, run.ID, run.ConversationID, run.UserID, nullIfEmpty(run.UserMessageID),
			nullIfEmpty(run.AssistantMessageID), valueOr(run.Model, "ylven-default"), valueOr(run.Status, "completed"),
			run.Cursor, nullIfEmpty(run.ErrorCode), run.CreatedAt, run.UpdatedAt, valueOr(run.BranchID, run.ConversationID),
			nullIfEmpty(run.IdempotencyKey), nullIfEmpty(run.ContextBuildID), valueOr(run.Provider, "upstream"),
			startedAt, run.CompletedAt, run.LatencyMs, run.ContinuationUsed, run.ContinuationFallback,
			valueOr(run.ReasoningProfile, "auto"), func() string { raw, _ := json.Marshal(copyJSONMap(run.ReasoningParameters)); return string(raw) }(), valueOr(run.ProviderModel, valueOr(run.Model, "ylven-default")))
		if err != nil {
			return err
		}
		for _, event := range events[run.ID] {
			sequence := event.ID
			if sequence <= 0 {
				continue
			}
			_, err = tx.ExecContext(ctx, `INSERT INTO run_events (run_id,event_type,delta,created_at,sequence,payload)
        VALUES ($1,$2,NULLIF($3,''),$4,$5,'{}'::jsonb) ON CONFLICT (run_id,sequence) DO NOTHING`, run.ID, event.Type, event.Delta, event.CreatedAt, sequence)
			if err != nil {
				return err
			}
		}
	}
	for _, draft := range drafts {
		_, err = tx.ExecContext(ctx, `INSERT INTO conversation_drafts (conversation_id,user_id,body,updated_at)
      VALUES ($1,$2,$3,$4) ON CONFLICT (conversation_id) DO NOTHING`, draft.ConversationID, draft.UserID, draft.Body, draft.UpdatedAt)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (p *postgresConversationStore) createConversation(user User, title string, temporary bool) (Conversation, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	id, err := newDatabaseUUID()
	if err != nil {
		return Conversation{}, err
	}
	now := time.Now().UTC()
	title = strings.TrimSpace(title)
	titleSource := "USER"
	locked := true
	status := "active"
	if title == "" {
		title = "新对话"
		titleSource = "AUTO_TEMP"
		locked = false
	}
	if temporary {
		status = "temporary"
	}
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return Conversation{}, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO conversations
    (id,user_id,title,title_source,title_locked,active_branch_id,status,created_at,updated_at,temporary)
    VALUES ($1,$2,$3,$4,$5,$1,$6,$7,$7,$8)`, id, user.ID, title, titleSource, locked, status, now, temporary)
	if err == nil {
		_, err = tx.ExecContext(ctx, `INSERT INTO conversation_branches
      (id,conversation_id,status,created_at,updated_at) VALUES ($1,$1,'active',$2,$2)`, id, now)
	}
	if err != nil {
		return Conversation{}, err
	}
	if err := tx.Commit(); err != nil {
		return Conversation{}, err
	}
	return Conversation{ID: id, UserID: user.ID, Title: title, TitleSource: titleSource, TitleLocked: locked, ActiveBranchID: id, Status: status, DefaultModelID: "ylven-default", DefaultReasoningProfile: "auto", AISettingsVersion: 1, CreatedAt: now, UpdatedAt: now, Temporary: temporary}, nil
}

func (p *postgresConversationStore) listConversations(userID, cursor string, limit int, includeArchived bool) ([]Conversation, string, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	args := []any{userID, limit + 1}
	query := `SELECT ` + conversationColumns + ` FROM conversations
    WHERE user_id=$1 AND deleted_at IS NULL AND ($3 OR archived_at IS NULL)`
	args = append(args, includeArchived)
	if cursor != "" {
		query += ` AND (updated_at,id) < ((SELECT updated_at FROM conversations WHERE id=$4 AND user_id=$1),$4)`
		args = append(args, cursor)
	}
	query += ` ORDER BY updated_at DESC,id DESC LIMIT $2`
	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]Conversation, 0, limit+1)
	for rows.Next() {
		item, scanErr := scanConversation(rows)
		if scanErr != nil {
			return nil, "", scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	next := ""
	if len(items) > limit {
		next = items[limit-1].ID
		items = items[:limit]
	}
	return items, next, nil
}

func (p *postgresConversationStore) searchConversations(userID, queryText string, limit int) ([]Conversation, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	pattern := "%" + strings.ToLower(strings.TrimSpace(queryText)) + "%"
	rows, err := p.db.QueryContext(ctx, `SELECT `+conversationColumns+` FROM conversations c
    WHERE c.user_id=$1 AND c.deleted_at IS NULL AND
      (LOWER(c.title) LIKE $2 OR EXISTS (SELECT 1 FROM messages m WHERE m.conversation_id=c.id AND LOWER(m.body) LIKE $2))
    ORDER BY c.updated_at DESC,c.id DESC LIMIT $3`, userID, pattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Conversation{}
	for rows.Next() {
		item, scanErr := scanConversation(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (p *postgresConversationStore) startRun(user User, conversationID, body string, selection ModelSelection, idempotencyKey, operation string) (MessageRun, bool, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return MessageRun{}, false, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, conversationID); err != nil {
		return MessageRun{}, false, err
	}
	requestHash := conversationSelectionRequestHash(body, selection)
	namespace := user.ID + ":" + operation
	if existing, found, err := p.idempotentRun(ctx, tx, namespace, idempotencyKey, requestHash); err != nil {
		return MessageRun{}, false, err
	} else if found {
		return existing, false, tx.Commit()
	}
	conversation, err := scanConversation(tx.QueryRowContext(ctx, `SELECT `+conversationColumns+` FROM conversations WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL FOR UPDATE`, conversationID, user.ID))
	if errors.Is(err, sql.ErrNoRows) {
		return MessageRun{}, false, errors.New("conversation_not_found")
	}
	if err != nil {
		return MessageRun{}, false, err
	}
	var activeRunID string
	err = tx.QueryRowContext(ctx, `SELECT id::text FROM message_runs WHERE conversation_id=$1 AND branch_id=$2 AND status IN ('queued','running','streaming') LIMIT 1`, conversation.ID, conversation.ActiveBranchID).Scan(&activeRunID)
	if err == nil {
		return MessageRun{}, false, errors.New("conversation_busy")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return MessageRun{}, false, err
	}
	run, err := p.insertRun(ctx, tx, user, conversation, body, selection, idempotencyKey)
	if err != nil {
		return MessageRun{}, false, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO idempotency_records
    (namespace,idempotency_key,request_hash,created_at,user_id,operation,resource_type,resource_id,expires_at)
    VALUES ($1,$2,$3,NOW(),$4,$5,'message_run',$6,NOW()+INTERVAL '24 hours')`, namespace, idempotencyKey, requestHash, user.ID, operation, run.ID)
	if err != nil {
		return MessageRun{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return MessageRun{}, false, err
	}
	return run, true, nil
}

func (p *postgresConversationStore) idempotentRun(ctx context.Context, tx *sql.Tx, namespace, key, requestHash string) (MessageRun, bool, error) {
	var resourceID, recordedHash string
	err := tx.QueryRowContext(ctx, `SELECT resource_id::text,request_hash FROM idempotency_records
    WHERE namespace=$1 AND idempotency_key=$2 AND (expires_at IS NULL OR expires_at>NOW())`, namespace, key).Scan(&resourceID, &recordedHash)
	if errors.Is(err, sql.ErrNoRows) {
		return MessageRun{}, false, nil
	}
	if err != nil {
		return MessageRun{}, false, err
	}
	if recordedHash != requestHash {
		return MessageRun{}, false, errors.New("idempotency_conflict")
	}
	run, err := scanRun(tx.QueryRowContext(ctx, `SELECT `+runColumns+` FROM message_runs WHERE id=$1`, resourceID))
	return run, err == nil, err
}

func (p *postgresConversationStore) insertRun(ctx context.Context, tx *sql.Tx, user User, conversation Conversation, body string, selection ModelSelection, idempotencyKey string) (MessageRun, error) {
	messageID, err := newDatabaseUUID()
	if err != nil {
		return MessageRun{}, err
	}
	runID, err := newDatabaseUUID()
	if err != nil {
		return MessageRun{}, err
	}
	var sequence int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence),0)+1 FROM messages WHERE conversation_id=$1 AND branch_id=$2`, conversation.ID, conversation.ActiveBranchID).Scan(&sequence); err != nil {
		return MessageRun{}, err
	}
	now := time.Now().UTC()
	_, err = tx.ExecContext(ctx, `INSERT INTO messages
    (id,conversation_id,user_id,branch_id,sequence,role,body,status,created_at,completed_at)
    VALUES ($1,$2,$3,$4,$5,'user',$6,'completed',$7,$7)`, messageID, conversation.ID, user.ID, conversation.ActiveBranchID, sequence, body, now)
	if err == nil {
		_, err = tx.ExecContext(ctx, `INSERT INTO message_parts (id,message_id,ordinal,kind,text_content,created_at)
      VALUES ($1,$1,0,'TEXT',$2,$3)`, messageID, body, now)
	}
	parameters, marshalErr := json.Marshal(copyJSONMap(selection.ReasoningParameters))
	if marshalErr != nil {
		return MessageRun{}, marshalErr
	}
	if err == nil {
		_, err = tx.ExecContext(ctx, `INSERT INTO message_runs
      (id,conversation_id,user_id,user_message_id,model,status,cursor,created_at,updated_at,branch_id,idempotency_key,request_hash,provider,started_at,
       reasoning_profile,reasoning_parameters,provider_model)
      VALUES ($1,$2,$3,$4,$5,'streaming',0,$6,$6,$7,$8,$9,'upstream',$6,$10,$11::jsonb,$12)`, runID, conversation.ID, user.ID,
			messageID, selection.CatalogModelID, now, conversation.ActiveBranchID, idempotencyKey, conversationSelectionRequestHash(body, selection),
			selection.ReasoningProfile, string(parameters), selection.ProviderModel)
	}
	if err == nil {
		_, err = tx.ExecContext(ctx, `UPDATE conversations SET updated_at=$2 WHERE id=$1`, conversation.ID, now)
	}
	if err == nil {
		_, err = tx.ExecContext(ctx, `INSERT INTO conversation_locks (conversation_id,holder_run_id,lease_owner,lease_expires_at,fencing_token,updated_at)
      VALUES ($1,$2,$3,NOW()+INTERVAL '2 minutes',1,NOW())
      ON CONFLICT (conversation_id) DO UPDATE SET holder_run_id=EXCLUDED.holder_run_id,lease_owner=EXCLUDED.lease_owner,
      lease_expires_at=EXCLUDED.lease_expires_at,fencing_token=conversation_locks.fencing_token+1,updated_at=NOW()`, conversation.ID, runID, instanceID())
	}
	if err != nil {
		return MessageRun{}, err
	}
	return MessageRun{ID: runID, ConversationID: conversation.ID, UserID: user.ID, UserMessageID: messageID,
		Model: selection.CatalogModelID, ProviderModel: selection.ProviderModel, ReasoningProfile: selection.ReasoningProfile,
		ReasoningParameters: copyJSONMap(selection.ReasoningParameters), BranchID: conversation.ActiveBranchID,
		IdempotencyKey: idempotencyKey, Status: "streaming", Provider: "upstream", StartedAt: now, CreatedAt: now, UpdatedAt: now}, nil
}

func (p *postgresConversationStore) firstMessage(user User, draftSessionID, body string, selection ModelSelection, idempotencyKey string, temporary bool) (FirstMessageResult, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return FirstMessageResult{}, err
	}
	defer tx.Rollback()
	namespace := user.ID + ":first_message"
	requestHash := conversationSelectionRequestHash(body, selection)
	if existing, found, err := p.idempotentRun(ctx, tx, namespace, idempotencyKey, requestHash); err != nil {
		return FirstMessageResult{}, err
	} else if found {
		conversation, loadErr := scanConversation(tx.QueryRowContext(ctx, `SELECT `+conversationColumns+` FROM conversations WHERE id=$1 AND user_id=$2`, existing.ConversationID, user.ID))
		if loadErr != nil {
			return FirstMessageResult{}, loadErr
		}
		return FirstMessageResult{Conversation: conversation, Run: existing, Created: false}, tx.Commit()
	}
	conversationID, err := newDatabaseUUID()
	if err != nil {
		return FirstMessageResult{}, err
	}
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, user.ID+":"+draftSessionID); err != nil {
		return FirstMessageResult{}, err
	}
	now := time.Now().UTC()
	status := "active"
	if temporary {
		status = "temporary"
	}
	conversation := Conversation{ID: conversationID, UserID: user.ID, Title: TemporaryConversationTitle(body), TitleSource: "AUTO_TEMP", ActiveBranchID: conversationID, Status: status, DefaultModelID: "ylven-default", DefaultReasoningProfile: "auto", AISettingsVersion: 1, CreatedAt: now, UpdatedAt: now, Temporary: temporary}
	_, err = tx.ExecContext(ctx, `INSERT INTO conversations
    (id,user_id,title,title_source,title_locked,active_branch_id,status,created_at,updated_at,temporary)
    VALUES ($1,$2,$3,'AUTO_TEMP',FALSE,$1,$4,$5,$5,$6)`, conversationID, user.ID, conversation.Title, status, now, temporary)
	if err == nil {
		_, err = tx.ExecContext(ctx, `INSERT INTO conversation_branches (id,conversation_id,status,created_at,updated_at)
      VALUES ($1,$1,'active',$2,$2)`, conversationID, now)
	}
	if err != nil {
		return FirstMessageResult{}, err
	}
	run, err := p.insertRun(ctx, tx, user, conversation, body, selection, idempotencyKey)
	if err != nil {
		return FirstMessageResult{}, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO idempotency_records
    (namespace,idempotency_key,request_hash,created_at,user_id,operation,resource_type,resource_id,expires_at)
    VALUES ($1,$2,$3,NOW(),$4,'first_message','message_run',$5,NOW()+INTERVAL '24 hours')`, namespace, idempotencyKey, requestHash, user.ID, run.ID)
	if err == nil {
		jobID, tokenErr := newDatabaseUUID()
		if tokenErr != nil {
			return FirstMessageResult{}, tokenErr
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO conversation_title_jobs
      (id,conversation_id,source_message_id,status,available_at,created_at,updated_at)
      VALUES ($1,$2,$3,'queued',NOW(),NOW(),NOW()) ON CONFLICT (conversation_id,source_message_id) DO NOTHING`, jobID, conversationID, run.UserMessageID)
	}
	if err != nil {
		return FirstMessageResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return FirstMessageResult{}, err
	}
	return FirstMessageResult{Conversation: conversation, Run: run, Created: true}, nil
}

func instanceID() string {
	return valueOr(strings.TrimSpace(os.Getenv("YLVEN_INSTANCE_ID")), "single-instance")
}

func (p *postgresConversationStore) completeRun(userID, runID, assistantBody string) (MessageRun, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return MessageRun{}, err
	}
	defer tx.Rollback()
	run, err := scanRun(tx.QueryRowContext(ctx, `SELECT `+runColumns+` FROM message_runs WHERE id=$1 AND user_id=$2 FOR UPDATE`, runID, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return MessageRun{}, errors.New("run_not_found")
	}
	if err != nil {
		return MessageRun{}, err
	}
	if run.Status == "cancelled" {
		return run, errors.New("run_cancelled")
	}
	if run.Status != "streaming" {
		return run, errors.New("run_not_streaming")
	}
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, run.ConversationID); err != nil {
		return MessageRun{}, err
	}
	assistantID, err := newDatabaseUUID()
	if err != nil {
		return MessageRun{}, err
	}
	var sequence int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence),0)+1 FROM messages WHERE conversation_id=$1 AND branch_id=$2`, run.ConversationID, run.BranchID).Scan(&sequence); err != nil {
		return MessageRun{}, err
	}
	now := time.Now().UTC()
	_, err = tx.ExecContext(ctx, `INSERT INTO messages
    (id,conversation_id,user_id,branch_id,sequence,parent_message_id,comparison_group_id,role,body,status,created_at,completed_at)
    VALUES ($1,$2,$3,$4,$5,$6,(SELECT comparison_group_id FROM messages WHERE id=$6),'assistant',$7,'completed',$8,$8)`, assistantID, run.ConversationID, userID, run.BranchID, sequence, run.UserMessageID, assistantBody, now)
	if err == nil {
		_, err = tx.ExecContext(ctx, `INSERT INTO message_parts (id,message_id,ordinal,kind,text_content,created_at)
      VALUES ($1,$1,0,'TEXT',$2,$3)`, assistantID, assistantBody, now)
	}
	if err != nil {
		return MessageRun{}, err
	}
	cursor := run.Cursor
	for _, chunk := range runeChunks(assistantBody, 48) {
		cursor++
		if _, err := tx.ExecContext(ctx, `INSERT INTO run_events (run_id,event_type,delta,created_at,sequence,payload)
      VALUES ($1,'delta',$2::text,$3,$4,jsonb_build_object('delta',$2::text))`, run.ID, chunk, now, cursor); err != nil {
			return MessageRun{}, err
		}
	}
	cursor++
	if _, err := tx.ExecContext(ctx, `INSERT INTO run_events (run_id,event_type,created_at,sequence,payload)
    VALUES ($1,'completed',$2,$3,'{}'::jsonb)`, run.ID, now, cursor); err != nil {
		return MessageRun{}, err
	}
	latency := now.Sub(run.StartedAt).Milliseconds()
	_, err = tx.ExecContext(ctx, `UPDATE message_runs SET assistant_message_id=$2,status='completed',cursor=$3,
    completed_at=$4,latency_ms=$5,updated_at=$4 WHERE id=$1`, run.ID, assistantID, cursor, now, latency)
	if err == nil {
		_, err = tx.ExecContext(ctx, `UPDATE conversation_locks SET holder_run_id=NULL,lease_owner=NULL,lease_expires_at=NULL,updated_at=NOW() WHERE conversation_id=$1 AND holder_run_id=$2`, run.ConversationID, run.ID)
	}
	if err == nil {
		err = p.updateComparisonStatusForRunTx(ctx, tx, run.ID, now)
	}
	if err != nil {
		return MessageRun{}, err
	}
	if err := tx.Commit(); err != nil {
		return MessageRun{}, err
	}
	run.AssistantMessageID = assistantID
	run.Status = "completed"
	run.Cursor = cursor
	run.CompletedAt = &now
	run.LatencyMs = latency
	run.UpdatedAt = now
	return run, nil
}

func runeChunks(value string, size int) []string {
	if size <= 0 {
		size = 48
	}
	runes := []rune(value)
	chunks := make([]string, 0, (len(runes)+size-1)/size)
	for start := 0; start < len(runes); start += size {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[start:end]))
	}
	return chunks
}

func (p *postgresConversationStore) finishRunWithError(userID, runID, status string) (MessageRun, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return MessageRun{}, err
	}
	defer tx.Rollback()
	run, err := scanRun(tx.QueryRowContext(ctx, `SELECT `+runColumns+` FROM message_runs WHERE id=$1 AND user_id=$2 FOR UPDATE`, runID, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return MessageRun{}, errors.New("run_not_found")
	}
	if err != nil {
		return MessageRun{}, err
	}
	if run.Status != "streaming" {
		return run, tx.Commit()
	}
	now := time.Now().UTC()
	cursor := run.Cursor + 1
	errorCode := ""
	if status == "failed" {
		errorCode = "chat_provider_error"
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO run_events (run_id,event_type,created_at,sequence,payload)
    VALUES ($1,$2,$3,$4,'{}'::jsonb)`, run.ID, status, now, cursor)
	if err == nil {
		_, err = tx.ExecContext(ctx, `UPDATE message_runs SET status=$2,error_code=NULLIF($3,''),cursor=$4,
      completed_at=$5,latency_ms=$6,updated_at=$5 WHERE id=$1`, run.ID, status, errorCode, cursor, now, now.Sub(run.StartedAt).Milliseconds())
	}
	if err == nil {
		_, err = tx.ExecContext(ctx, `UPDATE conversation_locks SET holder_run_id=NULL,lease_owner=NULL,lease_expires_at=NULL,updated_at=NOW() WHERE conversation_id=$1 AND holder_run_id=$2`, run.ConversationID, run.ID)
	}
	if err == nil {
		err = p.updateComparisonStatusForRunTx(ctx, tx, run.ID, now)
	}
	if err != nil {
		return MessageRun{}, err
	}
	if err := tx.Commit(); err != nil {
		return MessageRun{}, err
	}
	run.Status = status
	run.ErrorCode = errorCode
	run.Cursor = cursor
	run.CompletedAt = &now
	run.LatencyMs = now.Sub(run.StartedAt).Milliseconds()
	run.UpdatedAt = now
	return run, nil
}

func (p *postgresConversationStore) run(userID, runID string) (MessageRun, []Message, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	run, err := scanRun(p.db.QueryRowContext(ctx, `SELECT `+runColumns+` FROM message_runs WHERE id=$1 AND user_id=$2`, runID, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return MessageRun{}, nil, errors.New("run_not_found")
	}
	if err != nil {
		return MessageRun{}, nil, err
	}
	rows, err := p.db.QueryContext(ctx, `SELECT `+messageColumns+` FROM messages WHERE id=$1 OR id=$2 ORDER BY sequence`, run.UserMessageID, nullIfEmpty(run.AssistantMessageID))
	if err != nil {
		return MessageRun{}, nil, err
	}
	defer rows.Close()
	messages := []Message{}
	for rows.Next() {
		message, scanErr := scanMessage(rows)
		if scanErr != nil {
			return MessageRun{}, nil, scanErr
		}
		messages = append(messages, message)
	}
	return run, messages, rows.Err()
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func (p *postgresConversationStore) events(userID, runID string, after int64) (MessageRun, []RunEvent, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	run, err := scanRun(p.db.QueryRowContext(ctx, `SELECT `+runColumns+` FROM message_runs WHERE id=$1 AND user_id=$2`, runID, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return MessageRun{}, nil, errors.New("run_not_found")
	}
	if err != nil {
		return MessageRun{}, nil, err
	}
	rows, err := p.db.QueryContext(ctx, `SELECT sequence,event_type,COALESCE(delta,''),created_at FROM run_events WHERE run_id=$1 AND sequence>$2 ORDER BY sequence`, runID, after)
	if err != nil {
		return MessageRun{}, nil, err
	}
	defer rows.Close()
	items := []RunEvent{}
	for rows.Next() {
		var item RunEvent
		item.RunID = runID
		if err := rows.Scan(&item.ID, &item.Type, &item.Delta, &item.CreatedAt); err != nil {
			return MessageRun{}, nil, err
		}
		items = append(items, item)
	}
	return run, items, rows.Err()
}

func (p *postgresConversationStore) updateTitle(userID, conversationID, title, source string, lock bool) (Conversation, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	query := `UPDATE conversations SET title=$3,title_source=$4,title_locked=$5,updated_at=NOW()
    WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL`
	if !lock {
		query += ` AND title_locked=FALSE AND title_source='AUTO_TEMP'`
	}
	query += ` RETURNING ` + conversationColumns
	item, err := scanConversation(p.db.QueryRowContext(ctx, query, conversationID, userID, title, source, lock))
	updated := err == nil
	if errors.Is(err, sql.ErrNoRows) && !lock {
		item, err = scanConversation(p.db.QueryRowContext(ctx, `SELECT `+conversationColumns+` FROM conversations WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL`, conversationID, userID))
	}
	if errors.Is(err, sql.ErrNoRows) {
		return Conversation{}, errors.New("conversation_not_found")
	}
	if err == nil && lock {
		_, err = p.db.ExecContext(ctx, `UPDATE conversation_title_jobs SET status='cancelled',updated_at=NOW()
      WHERE conversation_id=$1 AND status IN ('queued','running')`, conversationID)
	} else if err == nil && updated && source == "AUTO_FINAL" {
		_, err = p.db.ExecContext(ctx, `UPDATE conversation_title_jobs SET status='completed',updated_at=NOW()
      WHERE conversation_id=$1 AND status IN ('queued','running')`, conversationID)
	}
	return item, err
}

func (p *postgresConversationStore) needsFinalTitle(userID, conversationID string) (bool, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	var needed bool
	err := p.db.QueryRowContext(ctx, `SELECT title_source='AUTO_TEMP' AND title_locked=FALSE
    FROM conversations WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL`, conversationID, userID).Scan(&needed)
	if errors.Is(err, sql.ErrNoRows) {
		return false, errors.New("conversation_not_found")
	}
	return needed, err
}

func (p *postgresConversationStore) modelCapability(ctx context.Context, model string) (ModelCapability, error) {
	var capability ModelCapability
	err := p.db.QueryRowContext(ctx, `SELECT context_limit_tokens,output_reserve_tokens,reasoning_reserve_tokens,
    tool_reserve_tokens,safety_margin_tokens FROM model_capabilities WHERE LOWER(model_id)=LOWER($1)`, strings.TrimSpace(model)).Scan(
		&capability.ContextLimitTokens, &capability.OutputReserveTokens, &capability.ReasoningReserveTokens,
		&capability.ToolReserveTokens, &capability.SafetyMarginTokens)
	if errors.Is(err, sql.ErrNoRows) {
		return DefaultModelCapability(model), nil
	}
	if err != nil {
		return ModelCapability{}, err
	}
	if capability.ContextLimitTokens <= 0 || capability.OutputReserveTokens < 0 || capability.ReasoningReserveTokens < 0 || capability.ToolReserveTokens < 0 || capability.SafetyMarginTokens < 0 {
		return ModelCapability{}, errors.New("model_capability_invalid")
	}
	return capability, nil
}

func (p *postgresConversationStore) conversationContext(userID, conversationID string) (ConversationContext, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	tx, err := p.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return ConversationContext{}, err
	}
	defer tx.Rollback()
	conversation, err := scanConversation(tx.QueryRowContext(ctx, `SELECT `+conversationColumns+` FROM conversations
    WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL`, conversationID, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return ConversationContext{}, errors.New("conversation_not_found")
	}
	if err != nil {
		return ConversationContext{}, err
	}
	var branch ConversationBranch
	err = tx.QueryRowContext(ctx, `SELECT id::text,conversation_id::text,COALESCE(parent_branch_id::text,''),
    COALESCE(forked_from_message_id::text,''),status,created_at,updated_at FROM conversation_branches
    WHERE id=$1 AND conversation_id=$2`, conversation.ActiveBranchID, conversation.ID).Scan(
		&branch.ID, &branch.ConversationID, &branch.ParentBranchID, &branch.ForkedFromMessageID,
		&branch.Status, &branch.CreatedAt, &branch.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ConversationContext{}, errors.New("conversation_branch_not_found")
	}
	if err != nil {
		return ConversationContext{}, err
	}

	snapshot := ConversationContext{Conversation: conversation, Branch: branch, Messages: []Message{}, MessageParts: []MessagePart{}, Items: []ContextBuildItem{}}
	rows, err := tx.QueryContext(ctx, `SELECT `+messageColumns+` FROM messages
    WHERE conversation_id=$1 AND branch_id=$2 AND status='completed' ORDER BY sequence`, conversation.ID, branch.ID)
	if err != nil {
		return ConversationContext{}, err
	}
	for rows.Next() {
		message, scanErr := scanMessage(rows)
		if scanErr != nil {
			rows.Close()
			return ConversationContext{}, scanErr
		}
		snapshot.Messages = append(snapshot.Messages, message)
		snapshot.Items = append(snapshot.Items, ContextBuildItem{Ordinal: len(snapshot.Items), ItemType: "message", SourceID: message.ID, Role: message.Role, Content: message.Body, EstimatedTokens: EstimateTokens(message.Body), Included: true})
	}
	if err := rows.Close(); err != nil {
		return ConversationContext{}, err
	}

	partRows, err := tx.QueryContext(ctx, `SELECT mp.id::text,mp.message_id::text,mp.ordinal,mp.kind,
    COALESCE(mp.text_content,''),COALESCE(mp.object_key,''),mp.metadata,mp.created_at
    FROM message_parts mp JOIN messages m ON m.id=mp.message_id
    WHERE m.conversation_id=$1 AND m.branch_id=$2 AND m.status='completed'
    ORDER BY m.sequence,mp.ordinal`, conversation.ID, branch.ID)
	if err != nil {
		return ConversationContext{}, err
	}
	for partRows.Next() {
		var part MessagePart
		var metadata []byte
		if err := partRows.Scan(&part.ID, &part.MessageID, &part.Ordinal, &part.Kind, &part.TextContent, &part.ObjectKey, &metadata, &part.CreatedAt); err != nil {
			partRows.Close()
			return ConversationContext{}, err
		}
		part.Metadata = map[string]any{}
		if len(metadata) > 0 {
			if err := json.Unmarshal(metadata, &part.Metadata); err != nil {
				partRows.Close()
				return ConversationContext{}, err
			}
		}
		snapshot.MessageParts = append(snapshot.MessageParts, part)
	}
	if err := partRows.Close(); err != nil {
		return ConversationContext{}, err
	}

	summary, err := scanConversationSummary(tx.QueryRowContext(ctx, `SELECT id::text,conversation_id::text,branch_id::text,
    source_first_sequence,source_last_sequence,source_first_message_id::text,source_last_message_id::text,
    structured_state,summary_text,summary_model,summary_version FROM conversation_summaries
    WHERE conversation_id=$1 AND branch_id=$2 ORDER BY source_last_sequence DESC,created_at DESC LIMIT 1`, conversation.ID, branch.ID))
	if err == nil {
		snapshot.Summary = &summary
	} else if !errors.Is(err, sql.ErrNoRows) {
		return ConversationContext{}, err
	}
	if err := tx.Commit(); err != nil {
		return ConversationContext{}, err
	}
	return snapshot, nil
}

func saveConversationSummary(ctx context.Context, tx *sql.Tx, summary ConversationSummary) error {
	stateJSON, err := json.Marshal(summary.StructuredState)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO conversation_summaries
    (id,conversation_id,branch_id,source_first_sequence,source_last_sequence,source_first_message_id,
     source_last_message_id,structured_state,summary_text,summary_model,summary_version,verified_at)
    VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NOW())
    ON CONFLICT (conversation_id,branch_id,source_last_sequence,summary_version) DO UPDATE SET
      structured_state=EXCLUDED.structured_state,summary_text=EXCLUDED.summary_text,
      summary_model=EXCLUDED.summary_model,verified_at=NOW()`, summary.ID, summary.ConversationID,
		summary.BranchID, summary.SourceFirstSequence, summary.SourceLastSequence, summary.SourceFirstMessageID,
		summary.SourceLastMessageID, stateJSON, summary.SummaryText, summary.SummaryModel, summary.SummaryVersion)
	return err
}

func (p *postgresConversationStore) compileRunContext(userID, runID, continuationMode string) (ContextBuild, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	run, err := scanRun(p.db.QueryRowContext(ctx, `SELECT `+runColumns+` FROM message_runs WHERE id=$1 AND user_id=$2`, runID, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return ContextBuild{}, errors.New("run_not_found")
	}
	if err != nil {
		return ContextBuild{}, err
	}
	rows, err := p.db.QueryContext(ctx, `SELECT `+messageColumns+` FROM messages
    WHERE conversation_id=$1 AND branch_id=$2 AND status='completed' ORDER BY sequence`, run.ConversationID, run.BranchID)
	if err != nil {
		return ContextBuild{}, err
	}
	history := []Message{}
	for rows.Next() {
		message, scanErr := scanMessage(rows)
		if scanErr != nil {
			rows.Close()
			return ContextBuild{}, scanErr
		}
		history = append(history, message)
	}
	if err := rows.Close(); err != nil {
		return ContextBuild{}, err
	}
	var existing *ConversationSummary
	summary, err := scanConversationSummary(p.db.QueryRowContext(ctx, `SELECT id::text,conversation_id::text,branch_id::text,source_first_sequence,source_last_sequence,
    source_first_message_id::text,source_last_message_id::text,structured_state,summary_text,summary_model,summary_version
    FROM conversation_summaries WHERE conversation_id=$1 AND branch_id=$2 ORDER BY source_last_sequence DESC LIMIT 1`, run.ConversationID, run.BranchID))
	if err == nil {
		existing = &summary
	} else if !errors.Is(err, sql.ErrNoRows) {
		return ContextBuild{}, err
	}
	capability, err := p.modelCapability(ctx, run.Model)
	if err != nil {
		return ContextBuild{}, err
	}
	buildID, err := newDatabaseUUID()
	if err != nil {
		return ContextBuild{}, err
	}
	build := (ConversationContextCompiler{}).Compile(ContextBuildInput{
		BuildID: buildID, RunID: run.ID, ConversationID: run.ConversationID, BranchID: run.BranchID, Model: run.Model,
		SystemInstruction: "Follow YLVEN safety and product policy. Use only user-visible conversation facts; never reveal or request hidden chain-of-thought.",
		Messages:          history, ExistingSummary: existing, Capability: capability, ContinuationMode: continuationMode,
	})
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return ContextBuild{}, err
	}
	defer tx.Rollback()
	if build.Summary != nil {
		err = saveConversationSummary(ctx, tx, *build.Summary)
		if err == nil {
			_, err = tx.ExecContext(ctx, `UPDATE conversations SET summary_through_message_id=$2 WHERE id=$1`, run.ConversationID, build.Summary.SourceLastMessageID)
		}
	}
	if err == nil {
		_, err = tx.ExecContext(ctx, `INSERT INTO context_builds
      (id,run_id,conversation_id,branch_id,model,context_limit_tokens,input_budget_tokens,estimated_input_tokens,
       output_reserve_tokens,reasoning_reserve_tokens,tool_reserve_tokens,safety_margin_tokens,compaction_mode,
       continuation_mode,context_hash,compiler_version)
      VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`, build.ID, build.RunID,
			build.ConversationID, build.BranchID, build.Model, build.ContextLimitTokens, build.InputBudgetTokens,
			build.EstimatedInputTokens, build.OutputReserveTokens, build.ReasoningReserveTokens, build.ToolReserveTokens,
			build.SafetyMarginTokens, build.CompactionMode, build.ContinuationMode, build.ContextHash, build.CompilerVersion)
	}
	for _, item := range build.Items {
		if err != nil {
			break
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO context_build_items
      (context_build_id,ordinal,item_type,source_id,role,content_redacted,estimated_tokens,included,exclusion_reason)
      VALUES ($1,$2,$3,NULLIF($4,''),NULLIF($5,''),NULLIF($6,''),$7,$8,NULLIF($9,''))`, build.ID, item.Ordinal,
			item.ItemType, item.SourceID, item.Role, item.Content, item.EstimatedTokens, item.Included, item.ExclusionReason)
	}
	if err == nil {
		_, err = tx.ExecContext(ctx, `UPDATE message_runs SET context_build_id=$2,updated_at=NOW() WHERE id=$1`, run.ID, build.ID)
	}
	if err != nil {
		return ContextBuild{}, err
	}
	if err := tx.Commit(); err != nil {
		return ContextBuild{}, err
	}
	return build, nil
}

func (p *postgresConversationStore) contextBuildOwned(userID, runID string) (ContextBuild, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	var build ContextBuild
	err := p.db.QueryRowContext(ctx, `SELECT cb.id::text,cb.run_id::text,cb.conversation_id::text,cb.branch_id::text,cb.model,
    cb.context_limit_tokens,cb.input_budget_tokens,cb.estimated_input_tokens,cb.output_reserve_tokens,
    cb.reasoning_reserve_tokens,cb.tool_reserve_tokens,cb.safety_margin_tokens,cb.compaction_mode,
    cb.continuation_mode,cb.context_hash,cb.compiler_version
    FROM context_builds cb JOIN message_runs r ON r.id=cb.run_id
    WHERE cb.run_id=$1 AND r.user_id=$2 ORDER BY cb.created_at DESC LIMIT 1`, runID, userID).Scan(
		&build.ID, &build.RunID, &build.ConversationID, &build.BranchID, &build.Model, &build.ContextLimitTokens,
		&build.InputBudgetTokens, &build.EstimatedInputTokens, &build.OutputReserveTokens, &build.ReasoningReserveTokens,
		&build.ToolReserveTokens, &build.SafetyMarginTokens, &build.CompactionMode, &build.ContinuationMode,
		&build.ContextHash, &build.CompilerVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return ContextBuild{}, errors.New("context_build_not_found")
	}
	if err != nil {
		return ContextBuild{}, err
	}
	rows, err := p.db.QueryContext(ctx, `SELECT ordinal,item_type,COALESCE(source_id,''),COALESCE(role,''),
    COALESCE(content_redacted,''),estimated_tokens,included,COALESCE(exclusion_reason,'')
    FROM context_build_items WHERE context_build_id=$1 ORDER BY ordinal`, build.ID)
	if err != nil {
		return ContextBuild{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var item ContextBuildItem
		if err := rows.Scan(&item.Ordinal, &item.ItemType, &item.SourceID, &item.Role, &item.Content, &item.EstimatedTokens, &item.Included, &item.ExclusionReason); err != nil {
			return ContextBuild{}, err
		}
		build.Items = append(build.Items, item)
		if item.Included && item.Content != "" {
			build.Messages = append(build.Messages, ProviderMessage{Role: normalizeProviderRole(item.Role), Content: item.Content})
		}
	}
	return build, rows.Err()
}

func (p *postgresConversationStore) queueConversationCompaction(userID, conversationID, idempotencyKey string) (ContextCompaction, bool, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return ContextCompaction{}, false, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, conversationID); err != nil {
		return ContextCompaction{}, false, err
	}
	conversation, err := scanConversation(tx.QueryRowContext(ctx, `SELECT `+conversationColumns+` FROM conversations
    WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL FOR UPDATE`, conversationID, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return ContextCompaction{}, false, errors.New("conversation_not_found")
	}
	if err != nil {
		return ContextCompaction{}, false, err
	}
	var firstSequence, lastSequence int64
	var firstMessageID, lastMessageID string
	err = tx.QueryRowContext(ctx, `SELECT id::text,sequence FROM messages WHERE conversation_id=$1 AND branch_id=$2
    AND status='completed' ORDER BY sequence LIMIT 1`, conversation.ID, conversation.ActiveBranchID).Scan(&firstMessageID, &firstSequence)
	if errors.Is(err, sql.ErrNoRows) {
		return ContextCompaction{}, false, errors.New("conversation_context_empty")
	}
	if err != nil {
		return ContextCompaction{}, false, err
	}
	if err := tx.QueryRowContext(ctx, `SELECT id::text,sequence FROM messages WHERE conversation_id=$1 AND branch_id=$2
    AND status='completed' ORDER BY sequence DESC LIMIT 1`, conversation.ID, conversation.ActiveBranchID).Scan(&lastMessageID, &lastSequence); err != nil {
		return ContextCompaction{}, false, err
	}
	requestHash := conversationRequestHash(conversation.ID+"\x00"+conversation.ActiveBranchID+"\x00"+lastMessageID, contextCompilerVersion)
	var recordedHash string
	existing, existingErr := scanContextCompaction(tx.QueryRowContext(ctx, `SELECT `+contextCompactionColumns+`
    FROM context_compactions cc JOIN jobs j ON j.id=cc.job_id
    WHERE j.user_id=$1 AND j.job_type='conversation_compaction' AND j.idempotency_key=$2`, userID, idempotencyKey))
	if existingErr == nil {
		if err := tx.QueryRowContext(ctx, `SELECT request_hash FROM jobs WHERE id=$1`, existing.JobID).Scan(&recordedHash); err != nil {
			return ContextCompaction{}, false, err
		}
		if recordedHash != requestHash {
			return ContextCompaction{}, false, errors.New("idempotency_conflict")
		}
		return existing, false, tx.Commit()
	}
	if !errors.Is(existingErr, sql.ErrNoRows) {
		return ContextCompaction{}, false, existingErr
	}
	jobID, err := newDatabaseUUID()
	if err != nil {
		return ContextCompaction{}, false, err
	}
	compactionID, err := newDatabaseUUID()
	if err != nil {
		return ContextCompaction{}, false, err
	}
	now := time.Now().UTC()
	_, err = tx.ExecContext(ctx, `INSERT INTO jobs
    (id,user_id,job_type,resource_type,resource_id,idempotency_key,request_hash,status,created_at,updated_at)
    VALUES ($1,$2,'conversation_compaction','context_compaction',$3,$4,$5,'queued',$6,$6)`, jobID,
		userID, compactionID, idempotencyKey, requestHash, now)
	if err == nil {
		_, err = tx.ExecContext(ctx, `INSERT INTO context_compactions
      (id,job_id,user_id,conversation_id,branch_id,idempotency_key,source_first_sequence,source_last_sequence,
       source_first_message_id,source_last_message_id,summary_version,status,created_at,updated_at)
      VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,'queued',$12,$12)`, compactionID, jobID, userID,
			conversation.ID, conversation.ActiveBranchID, idempotencyKey, firstSequence, lastSequence,
			firstMessageID, lastMessageID, contextCompilerVersion, now)
	}
	if err != nil {
		return ContextCompaction{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return ContextCompaction{}, false, err
	}
	return ContextCompaction{ID: compactionID, JobID: jobID, ConversationID: conversation.ID,
		BranchID: conversation.ActiveBranchID, IdempotencyKey: idempotencyKey, SourceFirstSequence: firstSequence,
		SourceLastSequence: lastSequence, SourceFirstMessageID: firstMessageID, SourceLastMessageID: lastMessageID,
		SummaryVersion: contextCompilerVersion, Status: "queued", CreatedAt: now, UpdatedAt: now}, true, nil
}

func (p *postgresConversationStore) processConversationCompaction(userID, compactionID string) (ContextCompaction, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return ContextCompaction{}, err
	}
	defer tx.Rollback()
	compaction, err := scanContextCompaction(tx.QueryRowContext(ctx, `SELECT `+contextCompactionColumns+`
    FROM context_compactions cc JOIN jobs j ON j.id=cc.job_id
    WHERE cc.id=$1 AND cc.user_id=$2 FOR UPDATE OF cc,j`, compactionID, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return ContextCompaction{}, errors.New("context_compaction_not_found")
	}
	if err != nil {
		return ContextCompaction{}, err
	}
	if compaction.Status != "queued" {
		return compaction, tx.Commit()
	}
	now := time.Now().UTC()
	result, err := tx.ExecContext(ctx, `UPDATE jobs SET status='running',attempts=attempts+1,lease_owner=$2,
	lease_expires_at=NOW()+INTERVAL '2 minutes',updated_at=$3 WHERE id=$1 AND status='queued' AND attempts<max_attempts`,
		compaction.JobID, instanceID(), now)
	if err != nil {
		return ContextCompaction{}, err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return compaction, tx.Commit()
	}
	_, err = tx.ExecContext(ctx, `UPDATE context_compactions SET status='running',started_at=$2,updated_at=$2
    WHERE id=$1`, compaction.ID, now)
	if err != nil {
		return ContextCompaction{}, err
	}
	if err := tx.Commit(); err != nil {
		return ContextCompaction{}, err
	}

	rows, err := p.db.QueryContext(ctx, `SELECT `+messageColumns+` FROM messages
    WHERE conversation_id=$1 AND branch_id=$2 AND status='completed' AND sequence BETWEEN $3 AND $4
    ORDER BY sequence`, compaction.ConversationID, compaction.BranchID, compaction.SourceFirstSequence, compaction.SourceLastSequence)
	if err != nil {
		return p.failConversationCompaction(ctx, userID, compaction.ID, "compaction_query_failed", err)
	}
	messages := []Message{}
	for rows.Next() {
		message, scanErr := scanMessage(rows)
		if scanErr != nil {
			rows.Close()
			return p.failConversationCompaction(ctx, userID, compaction.ID, "compaction_query_failed", scanErr)
		}
		messages = append(messages, message)
	}
	if err := rows.Close(); err != nil {
		return p.failConversationCompaction(ctx, userID, compaction.ID, "compaction_query_failed", err)
	}
	if len(messages) == 0 || messages[0].ID != compaction.SourceFirstMessageID || messages[len(messages)-1].ID != compaction.SourceLastMessageID {
		return p.failConversationCompaction(ctx, userID, compaction.ID, "compaction_source_changed", errors.New("compaction source range is incomplete"))
	}
	summary := BuildTraceableSummary(compaction.ConversationID, compaction.BranchID, messages)
	tx, err = p.db.BeginTx(ctx, nil)
	if err != nil {
		return p.failConversationCompaction(ctx, userID, compaction.ID, "compaction_persist_failed", err)
	}
	defer tx.Rollback()
	if err := saveConversationSummary(ctx, tx, summary); err != nil {
		return p.failConversationCompaction(ctx, userID, compaction.ID, "compaction_persist_failed", err)
	}
	completedAt := time.Now().UTC()
	result, err = tx.ExecContext(ctx, `UPDATE context_compactions SET status='success',summary_id=$3,error_code=NULL,
    completed_at=$4,updated_at=$4 WHERE id=$1 AND user_id=$2 AND status='running'`, compaction.ID, userID, summary.ID, completedAt)
	if err == nil {
		_, err = tx.ExecContext(ctx, `UPDATE jobs SET status='success',resource_id=$2,lease_owner=NULL,lease_expires_at=NULL,
      last_error_code=NULL,completed_at=$3,updated_at=$3 WHERE id=$1`, compaction.JobID, compaction.ID, completedAt)
	}
	if err == nil {
		_, err = tx.ExecContext(ctx, `UPDATE conversations SET summary_through_message_id=$3,updated_at=GREATEST(updated_at,$4)
      WHERE id=$1 AND user_id=$2`, compaction.ConversationID, userID, summary.SourceLastMessageID, completedAt)
	}
	if err != nil {
		return p.failConversationCompaction(ctx, userID, compaction.ID, "compaction_persist_failed", err)
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return p.failConversationCompaction(ctx, userID, compaction.ID, "compaction_state_conflict", errors.New("compaction was not running"))
	}
	if err := tx.Commit(); err != nil {
		return p.failConversationCompaction(ctx, userID, compaction.ID, "compaction_persist_failed", err)
	}
	return p.contextCompactionOwned(userID, compaction.ID)
}

func (p *postgresConversationStore) failConversationCompaction(ctx context.Context, userID, compactionID, code string, cause error) (ContextCompaction, error) {
	_, _ = p.db.ExecContext(ctx, `UPDATE context_compactions cc SET status='error',error_code=$3,completed_at=NOW(),updated_at=NOW()
    FROM jobs j WHERE cc.id=$1 AND cc.user_id=$2 AND j.id=cc.job_id AND cc.status='running'`, compactionID, userID, code)
	_, _ = p.db.ExecContext(ctx, `UPDATE jobs j SET status='error',last_error_code=$3,lease_owner=NULL,lease_expires_at=NULL,
    completed_at=NOW(),updated_at=NOW() FROM context_compactions cc
    WHERE cc.id=$1 AND cc.user_id=$2 AND cc.job_id=j.id AND j.status='running'`, compactionID, userID, code)
	return ContextCompaction{}, cause
}

func (p *postgresConversationStore) contextCompactionOwned(userID, compactionID string) (ContextCompaction, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	item, err := scanContextCompaction(p.db.QueryRowContext(ctx, `SELECT `+contextCompactionColumns+`
    FROM context_compactions cc JOIN jobs j ON j.id=cc.job_id WHERE cc.id=$1 AND cc.user_id=$2`, compactionID, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return ContextCompaction{}, errors.New("context_compaction_not_found")
	}
	return item, err
}

func (p *postgresConversationStore) providerState(userID, conversationID, branchID, provider, model string) (ProviderConversationState, bool, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	var item ProviderConversationState
	var expiresAt sql.NullTime
	err := p.db.QueryRowContext(ctx, `SELECT pcs.id::text,pcs.conversation_id::text,pcs.branch_id::text,pcs.provider,pcs.model,
    pcs.continuation_id,pcs.status,pcs.expires_at,COALESCE(pcs.last_failure_code,''),pcs.created_at,pcs.updated_at
    FROM provider_conversation_states pcs JOIN conversations c ON c.id=pcs.conversation_id
    WHERE pcs.conversation_id=$1 AND pcs.branch_id=$2 AND pcs.provider=$3 AND pcs.model=$4 AND c.user_id=$5
      AND pcs.status='active' AND (pcs.expires_at IS NULL OR pcs.expires_at>NOW())`, conversationID, branchID, provider, model, userID).Scan(
		&item.ID, &item.ConversationID, &item.BranchID, &item.Provider, &item.Model, &item.ContinuationID,
		&item.Status, &expiresAt, &item.LastFailureCode, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ProviderConversationState{}, false, nil
	}
	if err != nil {
		return ProviderConversationState{}, false, err
	}
	if expiresAt.Valid {
		item.ExpiresAt = &expiresAt.Time
	}
	return item, true, nil
}

func (p *postgresConversationStore) saveProviderState(userID string, state ProviderConversationState) error {
	ctx, cancel := databaseContext()
	defer cancel()
	if state.ID == "" {
		var err error
		state.ID, err = newDatabaseUUID()
		if err != nil {
			return err
		}
	}
	result, err := p.db.ExecContext(ctx, `INSERT INTO provider_conversation_states
    (id,conversation_id,branch_id,provider,model,continuation_id,status,expires_at,last_failure_code,created_at,updated_at)
    SELECT $1,$2,$3,$4,$5,$6,$7,$8,NULLIF($9,''),NOW(),NOW()
    FROM conversations WHERE id=$2 AND user_id=$10
    ON CONFLICT (conversation_id,branch_id,provider,model) DO UPDATE SET continuation_id=EXCLUDED.continuation_id,
      status=EXCLUDED.status,expires_at=EXCLUDED.expires_at,last_failure_code=EXCLUDED.last_failure_code,updated_at=NOW()`,
		state.ID, state.ConversationID, state.BranchID, state.Provider, state.Model, state.ContinuationID,
		valueOr(state.Status, "active"), state.ExpiresAt, state.LastFailureCode, userID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return errors.New("conversation_not_found")
	}
	return nil
}

func (p *postgresConversationStore) markContinuationFallback(userID, runID string) error {
	ctx, cancel := databaseContext()
	defer cancel()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var conversationID, branchID, model string
	err = tx.QueryRowContext(ctx, `UPDATE message_runs SET provider_continuation_used=TRUE,provider_continuation_fallback=TRUE,updated_at=NOW()
    WHERE id=$1 AND user_id=$2 RETURNING conversation_id::text,branch_id::text,model`, runID, userID).Scan(&conversationID, &branchID, &model)
	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("run_not_found")
	}
	if err == nil {
		_, err = tx.ExecContext(ctx, `UPDATE provider_conversation_states SET status='failed',last_failure_code='continuation_unavailable',updated_at=NOW()
      WHERE conversation_id=$1 AND branch_id=$2 AND model=$3`, conversationID, branchID, model)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (p *postgresConversationStore) appendMessage(userID, conversationID, role, body string) (Message, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return Message{}, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, conversationID); err != nil {
		return Message{}, err
	}
	conversation, err := scanConversation(tx.QueryRowContext(ctx, `SELECT `+conversationColumns+` FROM conversations WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL FOR UPDATE`, conversationID, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return Message{}, errors.New("conversation_not_found")
	}
	if err != nil {
		return Message{}, err
	}
	messageID, err := newDatabaseUUID()
	if err != nil {
		return Message{}, err
	}
	var sequence int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence),0)+1 FROM messages WHERE conversation_id=$1 AND branch_id=$2`, conversation.ID, conversation.ActiveBranchID).Scan(&sequence); err != nil {
		return Message{}, err
	}
	now := time.Now().UTC()
	_, err = tx.ExecContext(ctx, `INSERT INTO messages
    (id,conversation_id,user_id,branch_id,sequence,role,body,status,created_at,completed_at)
    VALUES ($1,$2,$3,$4,$5,$6,$7,'completed',$8,$8)`, messageID, conversation.ID, userID, conversation.ActiveBranchID, sequence, role, body, now)
	if err == nil {
		_, err = tx.ExecContext(ctx, `INSERT INTO message_parts (id,message_id,ordinal,kind,text_content,created_at) VALUES ($1,$1,0,'TEXT',$2,$3)`, messageID, body, now)
	}
	if err == nil {
		_, err = tx.ExecContext(ctx, `UPDATE conversations SET updated_at=$2 WHERE id=$1`, conversation.ID, now)
	}
	if err != nil {
		return Message{}, err
	}
	if err := tx.Commit(); err != nil {
		return Message{}, err
	}
	return Message{ID: messageID, ConversationID: conversation.ID, UserID: userID, BranchID: conversation.ActiveBranchID, Sequence: sequence, Role: role, Body: body, Status: "completed", CreatedAt: now, CompletedAt: &now}, nil
}

func (p *postgresConversationStore) appendCanonicalMessage(userID, conversationID, role, body, idempotencyKey string) (Message, bool, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return Message{}, false, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, conversationID); err != nil {
		return Message{}, false, err
	}
	namespace := userID + ":context_message"
	requestHash := conversationRequestHash(conversationID+"\x00"+body, role)
	var resourceID, recordedHash string
	err = tx.QueryRowContext(ctx, `SELECT resource_id::text,request_hash FROM idempotency_records
    WHERE namespace=$1 AND idempotency_key=$2 AND (expires_at IS NULL OR expires_at>NOW())`, namespace, idempotencyKey).Scan(&resourceID, &recordedHash)
	if err == nil {
		if recordedHash != requestHash {
			return Message{}, false, errors.New("idempotency_conflict")
		}
		message, scanErr := scanMessage(tx.QueryRowContext(ctx, `SELECT `+messageColumns+` FROM messages
      WHERE id=$1 AND conversation_id=$2 AND user_id=$3`, resourceID, conversationID, userID))
		if scanErr != nil {
			return Message{}, false, scanErr
		}
		return message, false, tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Message{}, false, err
	}
	conversation, err := scanConversation(tx.QueryRowContext(ctx, `SELECT `+conversationColumns+` FROM conversations
    WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL FOR UPDATE`, conversationID, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return Message{}, false, errors.New("conversation_not_found")
	}
	if err != nil {
		return Message{}, false, err
	}
	messageID, err := newDatabaseUUID()
	if err != nil {
		return Message{}, false, err
	}
	var sequence int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence),0)+1 FROM messages
    WHERE conversation_id=$1 AND branch_id=$2`, conversation.ID, conversation.ActiveBranchID).Scan(&sequence); err != nil {
		return Message{}, false, err
	}
	now := time.Now().UTC()
	_, err = tx.ExecContext(ctx, `INSERT INTO messages
    (id,conversation_id,user_id,branch_id,sequence,role,body,status,created_at,completed_at)
    VALUES ($1,$2,$3,$4,$5,$6,$7,'completed',$8,$8)`, messageID, conversation.ID, userID,
		conversation.ActiveBranchID, sequence, role, body, now)
	if err == nil {
		_, err = tx.ExecContext(ctx, `INSERT INTO message_parts
      (id,message_id,ordinal,kind,text_content,created_at) VALUES ($1,$1,0,'TEXT',$2,$3)`, messageID, body, now)
	}
	if err == nil {
		_, err = tx.ExecContext(ctx, `INSERT INTO idempotency_records
      (namespace,idempotency_key,request_hash,created_at,user_id,operation,resource_type,resource_id,expires_at)
      VALUES ($1,$2,$3,$4,$5,'context_message','message',$6,$4+INTERVAL '24 hours')`, namespace,
			idempotencyKey, requestHash, now, userID, messageID)
	}
	if err == nil {
		_, err = tx.ExecContext(ctx, `UPDATE conversations SET updated_at=$2 WHERE id=$1`, conversation.ID, now)
	}
	if err != nil {
		return Message{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return Message{}, false, err
	}
	return Message{ID: messageID, ConversationID: conversation.ID, UserID: userID, BranchID: conversation.ActiveBranchID,
		Sequence: sequence, Role: role, Body: body, Status: "completed", CreatedAt: now, CompletedAt: &now}, true, nil
}

func (p *postgresConversationStore) saveDraft(userID, conversationID, body string) (Draft, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	var draft Draft
	err := p.db.QueryRowContext(ctx, `INSERT INTO conversation_drafts (conversation_id,user_id,body,updated_at)
    SELECT id,$2,$3,NOW() FROM conversations WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL
    ON CONFLICT (conversation_id) DO UPDATE SET body=EXCLUDED.body,updated_at=NOW()
    RETURNING conversation_id::text,user_id::text,body,updated_at`, conversationID, userID, body).Scan(&draft.ConversationID, &draft.UserID, &draft.Body, &draft.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Draft{}, errors.New("conversation_not_found")
	}
	return draft, err
}

func (p *postgresConversationStore) getDraft(userID, conversationID string) (Draft, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	var draft Draft
	err := p.db.QueryRowContext(ctx, `SELECT conversation_id::text,user_id::text,body,updated_at FROM conversation_drafts WHERE conversation_id=$1 AND user_id=$2`, conversationID, userID).Scan(&draft.ConversationID, &draft.UserID, &draft.Body, &draft.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Draft{}, errors.New("draft_not_found")
	}
	return draft, err
}

func (p *postgresConversationStore) setConversationStatus(userID, conversationID, operation string) (Conversation, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	var query string
	switch operation {
	case "archive":
		query = `UPDATE conversations SET status='archived',archived_at=NOW(),updated_at=NOW() WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL RETURNING ` + conversationColumns
	case "delete":
		query = `UPDATE conversations SET status='recycle_pending',deleted_at=NOW()+INTERVAL '30 days',updated_at=NOW() WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL RETURNING ` + conversationColumns
	default:
		return Conversation{}, errors.New("conversation_operation_invalid")
	}
	item, err := scanConversation(p.db.QueryRowContext(ctx, query, conversationID, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return Conversation{}, errors.New("conversation_not_found")
	}
	return item, err
}

func (p *postgresConversationStore) messageOwned(userID, messageID string) (Message, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	item, err := scanMessage(p.db.QueryRowContext(ctx, `SELECT `+messageColumns+` FROM messages WHERE id=$1 AND user_id=$2`, messageID, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return Message{}, errors.New("message_not_found")
	}
	return item, err
}

func (p *postgresConversationStore) runForMessage(userID, messageID string) (MessageRun, Message, error) {
	message, err := p.messageOwned(userID, messageID)
	if err != nil {
		return MessageRun{}, Message{}, err
	}
	ctx, cancel := databaseContext()
	defer cancel()
	run, err := scanRun(p.db.QueryRowContext(ctx, `SELECT `+runColumns+` FROM message_runs WHERE user_id=$1 AND (user_message_id=$2 OR assistant_message_id=$2) ORDER BY created_at DESC LIMIT 1`, userID, messageID))
	if errors.Is(err, sql.ErrNoRows) {
		return MessageRun{}, message, errors.New("run_not_found")
	}
	return run, message, err
}

func (p *postgresConversationStore) createSpeechJob(userID, messageID string) (SpeechJob, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	id, err := newDatabaseUUID()
	if err != nil {
		return SpeechJob{}, err
	}
	now := time.Now().UTC()
	result, err := p.db.ExecContext(ctx, `INSERT INTO speech_jobs (id,message_id,user_id,status,provider,created_at)
    SELECT $1,id,$2,'accepted','android_system_tts',$3 FROM messages WHERE id=$4 AND user_id=$2 AND role='assistant'`, id, userID, now, messageID)
	if err != nil {
		return SpeechJob{}, err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return SpeechJob{}, errors.New("message_not_found")
	}
	return SpeechJob{ID: id, MessageID: messageID, UserID: userID, Status: "accepted", Provider: "android_system_tts", CreatedAt: now}, nil
}

func (p *postgresConversationStore) exportConversation(userID, conversationID, messageID string) (ExportJob, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	conversation, err := scanConversation(p.db.QueryRowContext(ctx, `SELECT `+conversationColumns+` FROM conversations WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL`, conversationID, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return ExportJob{}, errors.New("conversation_not_found")
	}
	if err != nil {
		return ExportJob{}, err
	}
	args := []any{conversationID}
	query := `SELECT ` + messageColumns + ` FROM messages WHERE conversation_id=$1`
	if messageID != "" {
		query += ` AND id=$2`
		args = append(args, messageID)
	}
	query += ` ORDER BY branch_id,sequence`
	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return ExportJob{}, err
	}
	content := "# " + conversation.Title + "\n\n"
	for rows.Next() {
		message, scanErr := scanMessage(rows)
		if scanErr != nil {
			rows.Close()
			return ExportJob{}, scanErr
		}
		content += "## " + message.Role + "\n\n" + message.Body + "\n\n"
	}
	if err := rows.Close(); err != nil {
		return ExportJob{}, err
	}
	id, err := newDatabaseUUID()
	if err != nil {
		return ExportJob{}, err
	}
	now := time.Now().UTC()
	_, err = p.db.ExecContext(ctx, `INSERT INTO export_jobs (id,conversation_id,message_id,user_id,format,status,content,created_at)
    VALUES ($1,$2,$3,$4,'markdown','ready',$5,$6)`, id, conversationID, nullIfEmpty(messageID), userID, content, now)
	if err != nil {
		return ExportJob{}, err
	}
	return ExportJob{ID: id, ConversationID: conversationID, MessageID: messageID, UserID: userID, Format: "markdown", Status: "ready", Content: content, CreatedAt: now}, nil
}

func (p *postgresConversationStore) listAllConversations() ([]Conversation, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	rows, err := p.db.QueryContext(ctx, `SELECT `+conversationColumns+` FROM conversations ORDER BY updated_at DESC,id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Conversation{}
	for rows.Next() {
		item, scanErr := scanConversation(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (p *postgresConversationStore) conversationDetail(id string) (Conversation, []Message, []MessageRun, bool) {
	ctx, cancel := databaseContext()
	defer cancel()
	conversation, err := scanConversation(p.db.QueryRowContext(ctx, `SELECT `+conversationColumns+` FROM conversations WHERE id=$1`, id))
	if err != nil {
		return Conversation{}, nil, nil, false
	}
	messageRows, err := p.db.QueryContext(ctx, `SELECT `+messageColumns+` FROM messages WHERE conversation_id=$1 ORDER BY branch_id,sequence`, id)
	if err != nil {
		return Conversation{}, nil, nil, false
	}
	messages := []Message{}
	for messageRows.Next() {
		item, scanErr := scanMessage(messageRows)
		if scanErr != nil {
			messageRows.Close()
			return Conversation{}, nil, nil, false
		}
		messages = append(messages, item)
	}
	if err := messageRows.Close(); err != nil {
		return Conversation{}, nil, nil, false
	}
	runRows, err := p.db.QueryContext(ctx, `SELECT `+runColumns+` FROM message_runs WHERE conversation_id=$1 ORDER BY created_at`, id)
	if err != nil {
		return Conversation{}, nil, nil, false
	}
	defer runRows.Close()
	runs := []MessageRun{}
	for runRows.Next() {
		item, scanErr := scanRun(runRows)
		if scanErr != nil {
			return Conversation{}, nil, nil, false
		}
		runs = append(runs, item)
	}
	return conversation, messages, runs, runRows.Err() == nil
}

func (p *postgresConversationStore) conversationMessages(userID, id string) ([]Message, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	var activeBranchID string
	err := p.db.QueryRowContext(ctx, `SELECT active_branch_id::text FROM conversations WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL`, id, userID).Scan(&activeBranchID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("conversation_not_found")
	}
	if err != nil {
		return nil, err
	}
	rows, err := p.db.QueryContext(ctx, `SELECT `+messageColumns+` FROM messages WHERE conversation_id=$1 AND branch_id=$2 ORDER BY sequence`, id, activeBranchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	messages := []Message{}
	for rows.Next() {
		message, scanErr := scanMessage(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		messages = append(messages, message)
	}
	return messages, rows.Err()
}

func (p *postgresConversationStore) saveFeedback(userID, messageID, value string) error {
	ctx, cancel := databaseContext()
	defer cancel()
	result, err := p.db.ExecContext(ctx, `INSERT INTO message_feedback (message_id,user_id,value,created_at)
    SELECT id,$2,$3,NOW() FROM messages WHERE id=$1 AND user_id=$2
    ON CONFLICT (message_id,user_id) DO UPDATE SET value=EXCLUDED.value,created_at=NOW()`, messageID, userID, value)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return errors.New("message_not_found")
	}
	return nil
}
