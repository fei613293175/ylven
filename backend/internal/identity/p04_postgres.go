package identity

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

func (p *postgresConversationStore) readP04(key string, out any) (bool, error) {
	ctx, cancel := databaseContext(); defer cancel()
	var raw []byte
	err := p.db.QueryRowContext(ctx, `SELECT value FROM p04_records WHERE record_key=$1`, key).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) { return false, nil }; if err != nil { return false, err }
	return true, json.Unmarshal(raw, out)
}

func (p *postgresConversationStore) writeP04(key, userID, recordType string, value any, version int64) error {
	raw, err := json.Marshal(value); if err != nil { return err }
	ctx, cancel := databaseContext(); defer cancel()
	_, err = p.db.ExecContext(ctx, `INSERT INTO p04_records(record_key,user_id,record_type,value,version,updated_at)
      VALUES($1,$2,$3,$4::jsonb,$5,NOW()) ON CONFLICT(record_key) DO UPDATE SET value=EXCLUDED.value,version=EXCLUDED.version,updated_at=NOW()`, key, userID, recordType, string(raw), version)
	return err
}

func (p *postgresConversationStore) getAIPreference(access string) (AIPreference, bool, error) {
	user, err := p.storeUser(access); if err != nil { return AIPreference{}, false, err }
	var value AIPreference; found, err := p.readP04("preference:"+user.ID, &value); if err != nil { return AIPreference{}, false, err }
	if !found { return AIPreference{UserID:user.ID,ModelID:"ylven-default",ReasoningProfile:"auto",Version:1,UpdatedAt:time.Now().UTC()}, false, nil }
	return value, true, nil
}

func (p *postgresConversationStore) storeUser(access string) (User, error) {
	if p == nil || p.store == nil { return User{}, errors.New("session_invalid") }
	return p.store.authenticatedUser(access)
}

func (p *postgresConversationStore) putAIPreference(access string, input AIPreference) (AIPreference, error) {
	user, err := p.storeUser(access)
	if err != nil {
		return AIPreference{}, err
	}
	selection, err := p.resolveModelSelection(input.ModelID, input.ReasoningProfile)
	if err != nil {
		return AIPreference{}, err
	}
	ctx, cancel := databaseContext()
	defer cancel()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return AIPreference{}, err
	}
	defer tx.Rollback()
	key := "preference:" + user.ID
	var currentVersion int64
	err = tx.QueryRowContext(ctx, `SELECT version FROM p04_records WHERE record_key=$1 FOR UPDATE`, key).Scan(&currentVersion)
	if errors.Is(err, sql.ErrNoRows) {
		if input.Version != 0 && input.Version != 1 {
			return AIPreference{}, errors.New("version_conflict")
		}
		currentVersion = 0
	} else if err != nil {
		return AIPreference{}, err
	} else if input.Version != 0 && input.Version != currentVersion {
		return AIPreference{}, errors.New("version_conflict")
	}
	input.UserID = user.ID
	input.ModelID = selection.CatalogModelID
	input.ReasoningProfile = selection.ReasoningProfile
	input.Version = currentVersion + 1
	input.UpdatedAt = time.Now().UTC()
	raw, err := json.Marshal(input)
	if err != nil {
		return AIPreference{}, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO p04_records(record_key,user_id,record_type,value,version,updated_at)
		VALUES($1,$2,'ai_preference',$3::jsonb,$4,$5)
		ON CONFLICT(record_key) DO UPDATE SET value=EXCLUDED.value,version=EXCLUDED.version,updated_at=EXCLUDED.updated_at`,
		key, user.ID, string(raw), input.Version, input.UpdatedAt)
	if err != nil {
		return AIPreference{}, err
	}
	if err = tx.Commit(); err != nil {
		return AIPreference{}, err
	}
	return input, nil
}

func (p *postgresConversationStore) defaultSelection(access, modelID, profileID string) (ModelSelection, error) {
	preference, _, err := p.getAIPreference(access)
	if err != nil {
		return ModelSelection{}, err
	}
	if strings.TrimSpace(modelID) == "" {
		modelID = preference.ModelID
	}
	if strings.TrimSpace(profileID) == "" {
		profileID = preference.ReasoningProfile
	}
	return p.resolveModelSelection(modelID, profileID)
}

func (p *postgresConversationStore) effectiveSelection(access, conversationID, modelID, profileID string) (ModelSelection, error) {
	conversation, err := p.getConversationAISettings(access, conversationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ModelSelection{}, errors.New("conversation_not_found")
		}
		return ModelSelection{}, err
	}
	if conversation.AISettingsOverridden {
		if strings.TrimSpace(modelID) == "" {
			modelID = conversation.DefaultModelID
		}
		if strings.TrimSpace(profileID) == "" {
			profileID = conversation.DefaultReasoningProfile
		}
	}
	return p.defaultSelection(access, modelID, profileID)
}

func (p *postgresConversationStore) getConversationAISettings(access, conversationID string) (Conversation,error) {
	user, err := p.storeUser(access); if err != nil { return Conversation{},err }; ctx,cancel:=databaseContext(); defer cancel()
	return scanConversation(p.db.QueryRowContext(ctx,`SELECT `+conversationColumns+` FROM conversations WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL`,conversationID,user.ID))
}

func (p *postgresConversationStore) putConversationAISettings(access, conversationID, modelID, profileID string, version int64) (Conversation,error) {
	user, err := p.storeUser(access)
	if err != nil { return Conversation{}, err }
	selection, err := p.resolveModelSelection(modelID, profileID)
	if err != nil { return Conversation{}, err }
	ctx, cancel := databaseContext()
	defer cancel()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil { return Conversation{}, err }
	defer tx.Rollback()
	conversation, err := scanConversation(tx.QueryRowContext(ctx, `SELECT `+conversationColumns+` FROM conversations WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL FOR UPDATE`, conversationID, user.ID))
	if errors.Is(err, sql.ErrNoRows) { return Conversation{}, errors.New("conversation_not_found") }
	if err != nil { return Conversation{}, err }
	if version != 0 && version != conversation.AISettingsVersion { return Conversation{}, errors.New("version_conflict") }
	now := time.Now().UTC()
	conversation.DefaultModelID, conversation.DefaultReasoningProfile = selection.CatalogModelID, selection.ReasoningProfile
	conversation.AISettingsVersion++
	conversation.AISettingsOverridden = true
	conversation.UpdatedAt = now
	_, err = tx.ExecContext(ctx, `UPDATE conversations SET default_model_id=$2,default_reasoning_profile=$3,ai_settings_version=$4,ai_settings_overridden=TRUE,updated_at=$5 WHERE id=$1`, conversation.ID, conversation.DefaultModelID, conversation.DefaultReasoningProfile, conversation.AISettingsVersion, now)
	if err != nil { return Conversation{}, err }
	return conversation, tx.Commit()
}

func (p *postgresConversationStore) createConversationBranch(access, conversationID, forkedFromMessageID string) (ConversationBranch,error) {
	user, err := p.storeUser(access)
	if err != nil { return ConversationBranch{}, err }
	ctx, cancel := databaseContext()
	defer cancel()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil { return ConversationBranch{}, err }
	defer tx.Rollback()
	branch, err := p.createConversationBranchTx(ctx, tx, user, conversationID, forkedFromMessageID)
	if err != nil { return ConversationBranch{}, err }
	return branch, tx.Commit()
}

func (p *postgresConversationStore) createConversationBranchTx(ctx context.Context, tx *sql.Tx, user User, conversationID, forkedFromMessageID string) (ConversationBranch, error) {
	return p.createConversationBranchAtTx(ctx, tx, user, conversationID, forkedFromMessageID, forkedFromMessageID, "active")
}

// createConversationBranchAtTx snapshots source-branch history through copyThroughMessageID.
// The context compiler deliberately reads one branch at a time, so the snapshot
// makes the parent link navigable without leaking an incomplete history.
func (p *postgresConversationStore) createConversationBranchAtTx(ctx context.Context, tx *sql.Tx, user User, conversationID, forkedFromMessageID, copyThroughMessageID, status string) (ConversationBranch, error) {
	conversation, err := scanConversation(tx.QueryRowContext(ctx, `SELECT `+conversationColumns+` FROM conversations WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL FOR UPDATE`, conversationID, user.ID))
	if errors.Is(err, sql.ErrNoRows) { return ConversationBranch{}, errors.New("conversation_not_found") }
	if err != nil { return ConversationBranch{}, err }
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, conversation.ID); err != nil { return ConversationBranch{}, err }
	parentBranchID := conversation.ActiveBranchID
	if strings.TrimSpace(forkedFromMessageID) != "" {
		var messageConversationID string
		err = tx.QueryRowContext(ctx, `SELECT conversation_id::text,branch_id::text FROM messages WHERE id=$1 AND user_id=$2`, forkedFromMessageID, user.ID).Scan(&messageConversationID, &parentBranchID)
		if errors.Is(err, sql.ErrNoRows) || messageConversationID != conversation.ID { return ConversationBranch{}, errors.New("message_not_found") }
		if err != nil { return ConversationBranch{}, err }
	}
	if strings.TrimSpace(copyThroughMessageID) == "" {
		err = tx.QueryRowContext(ctx, `SELECT id::text FROM messages WHERE conversation_id=$1 AND branch_id=$2 ORDER BY sequence DESC LIMIT 1`, conversation.ID, parentBranchID).Scan(&copyThroughMessageID)
		if errors.Is(err, sql.ErrNoRows) {
			copyThroughMessageID = ""
		} else if err != nil {
			return ConversationBranch{}, err
		}
	}
	if copyThroughMessageID != "" {
		var copyConversationID, copyBranchID string
		err = tx.QueryRowContext(ctx, `SELECT conversation_id::text,branch_id::text FROM messages WHERE id=$1 AND user_id=$2`, copyThroughMessageID, user.ID).Scan(&copyConversationID, &copyBranchID)
		if errors.Is(err, sql.ErrNoRows) || copyConversationID != conversation.ID || copyBranchID != parentBranchID { return ConversationBranch{}, errors.New("message_not_found") }
		if err != nil { return ConversationBranch{}, err }
	}
	id, err := newDatabaseUUID()
	if err != nil { return ConversationBranch{}, err }
	now := time.Now().UTC()
	branch := ConversationBranch{ID:id, ConversationID:conversation.ID, ParentBranchID:parentBranchID, ForkedFromMessageID:forkedFromMessageID, Status:status, CreatedAt:now, UpdatedAt:now}
	_, err = tx.ExecContext(ctx, `INSERT INTO conversation_branches(id,conversation_id,parent_branch_id,forked_from_message_id,status,created_at,updated_at) VALUES($1,$2,NULLIF($3,'')::uuid,NULLIF($4,'')::uuid,$5,$6,$6)`, branch.ID, branch.ConversationID, branch.ParentBranchID, branch.ForkedFromMessageID, branch.Status, now)
	if err != nil { return ConversationBranch{}, err }
	if err = p.cloneBranchHistoryTx(ctx, tx, conversation.ID, parentBranchID, branch.ID, copyThroughMessageID); err != nil { return ConversationBranch{}, err }
	if _, err = tx.ExecContext(ctx, `UPDATE conversations SET active_branch_id=$2,updated_at=$3 WHERE id=$1`, conversation.ID, branch.ID, now); err != nil { return ConversationBranch{}, err }
	return branch, nil
}

func (p *postgresConversationStore) cloneBranchHistoryTx(ctx context.Context, tx *sql.Tx, conversationID, sourceBranchID, targetBranchID, throughMessageID string) error {
	if strings.TrimSpace(throughMessageID) == "" { return nil }
	var throughSequence int64
	err := tx.QueryRowContext(ctx, `SELECT sequence FROM messages WHERE id=$1 AND conversation_id=$2 AND branch_id=$3`, throughMessageID, conversationID, sourceBranchID).Scan(&throughSequence)
	if errors.Is(err, sql.ErrNoRows) { return errors.New("message_not_found") }
	if err != nil { return err }
	rows, err := tx.QueryContext(ctx, `SELECT `+messageColumns+` FROM messages WHERE conversation_id=$1 AND branch_id=$2 AND sequence <= $3 ORDER BY sequence`, conversationID, sourceBranchID, throughSequence)
	if err != nil { return err }
	sourceMessages := []Message{}
	for rows.Next() {
		message, scanErr := scanMessage(rows)
		if scanErr != nil { rows.Close(); return scanErr }
		sourceMessages = append(sourceMessages, message)
	}
	if err = rows.Err(); err != nil { rows.Close(); return err }
	if err = rows.Close(); err != nil { return err }
	messageIDs := map[string]string{}
	for _, message := range sourceMessages {
		newMessageID, idErr := newDatabaseUUID()
		if idErr != nil { return idErr }
		parentID := messageIDs[message.ParentMessageID]
		_, err = tx.ExecContext(ctx, `INSERT INTO messages(id,conversation_id,user_id,branch_id,sequence,parent_message_id,comparison_group_id,role,body,status,created_at,completed_at) VALUES($1,$2,$3,$4,$5,NULLIF($6,'')::uuid,NULLIF($7,'')::uuid,$8,$9,$10,$11,$12)`, newMessageID, message.ConversationID, message.UserID, targetBranchID, message.Sequence, parentID, message.ComparisonGroupID, message.Role, message.Body, message.Status, message.CreatedAt, message.CompletedAt)
		if err != nil { return err }
		partRows, partErr := tx.QueryContext(ctx, `SELECT ordinal,kind,text_content,object_key,metadata,created_at FROM message_parts WHERE message_id=$1 ORDER BY ordinal`, message.ID)
		if partErr != nil { return partErr }
		for partRows.Next() {
			var ordinal int; var kind string; var textContent, objectKey sql.NullString; var metadata []byte; var createdAt time.Time
			if partErr = partRows.Scan(&ordinal,&kind,&textContent,&objectKey,&metadata,&createdAt); partErr != nil { partRows.Close(); return partErr }
			partID, idErr := newDatabaseUUID(); if idErr != nil { partRows.Close(); return idErr }
			_, partErr = tx.ExecContext(ctx, `INSERT INTO message_parts(id,message_id,ordinal,kind,text_content,object_key,metadata,created_at) VALUES($1,$2,$3,$4,$5,$6,$7::jsonb,$8)`, partID,newMessageID,ordinal,kind,textContent,objectKey,string(metadata),createdAt)
			if partErr != nil { partRows.Close(); return partErr }
		}
		if partErr = partRows.Err(); partErr != nil { partRows.Close(); return partErr }
		if partErr = partRows.Close(); partErr != nil { return partErr }
		messageIDs[message.ID] = newMessageID
	}
	return nil
}

func (p *postgresConversationStore) listConversationBranches(access, conversationID string)([]ConversationBranch,error){
	user,err:=p.storeUser(access);if err!=nil{return nil,err};ctx,cancel:=databaseContext();defer cancel();rows,err:=p.db.QueryContext(ctx,`SELECT b.id::text,b.conversation_id::text,COALESCE(b.parent_branch_id::text,''),COALESCE(b.forked_from_message_id::text,''),b.status,b.created_at,b.updated_at FROM conversation_branches b JOIN conversations c ON c.id=b.conversation_id WHERE b.conversation_id=$1 AND c.user_id=$2 ORDER BY b.created_at`,conversationID,user.ID);if err!=nil{return nil,err};defer rows.Close();items:=[]ConversationBranch{};for rows.Next(){var v ConversationBranch;if err:=rows.Scan(&v.ID,&v.ConversationID,&v.ParentBranchID,&v.ForkedFromMessageID,&v.Status,&v.CreatedAt,&v.UpdatedAt);err!=nil{return nil,err};items=append(items,v)};return items,rows.Err()
}

func (p *postgresConversationStore) setActiveConversationBranch(access, conversationID, branchID string) (Conversation, error) {
	user, err := p.storeUser(access)
	if err != nil { return Conversation{}, err }
	ctx, cancel := databaseContext()
	defer cancel()
	conversation, err := scanConversation(p.db.QueryRowContext(ctx, `UPDATE conversations c SET active_branch_id=$2,updated_at=NOW()
		WHERE c.id=$1 AND c.user_id=$3 AND c.deleted_at IS NULL
		AND EXISTS (SELECT 1 FROM conversation_branches b WHERE b.id=$2 AND b.conversation_id=c.id)
		RETURNING `+conversationColumns, conversationID, branchID, user.ID))
	if errors.Is(err, sql.ErrNoRows) { return Conversation{}, errors.New("branch_not_found") }
	if err != nil { return Conversation{}, err }
	return conversation, nil
}

func (p *postgresConversationStore) rerunMessage(access,messageID,modelID,profileID string)(MessageRun,error){
	user, err := p.storeUser(access)
	if err != nil { return MessageRun{}, err }
	ctx, cancel := databaseContext()
	defer cancel()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil { return MessageRun{}, err }
	defer tx.Rollback()
	var conversationID, role, body, parentMessageID string
	err = tx.QueryRowContext(ctx, `SELECT conversation_id::text,role,body,COALESCE(parent_message_id::text,'') FROM messages WHERE id=$1 AND user_id=$2`, messageID, user.ID).Scan(&conversationID, &role, &body, &parentMessageID)
	if errors.Is(err, sql.ErrNoRows) { return MessageRun{}, errors.New("message_not_found") }
	if err != nil { return MessageRun{}, err }
	if role == "assistant" {
		if parentMessageID == "" { return MessageRun{}, errors.New("rerun_source_not_found") }
		err = tx.QueryRowContext(ctx, `SELECT body FROM messages WHERE id=$1 AND conversation_id=$2 AND user_id=$3 AND role='user'`, parentMessageID, conversationID, user.ID).Scan(&body)
		if errors.Is(err, sql.ErrNoRows) { return MessageRun{}, errors.New("rerun_source_not_found") }
		if err != nil { return MessageRun{}, err }
	}
	copyThroughMessageID := ""
	if parentMessageID != "" {
		var sourceBranchID string
		if err = tx.QueryRowContext(ctx, `SELECT branch_id::text FROM messages WHERE id=$1`, parentMessageID).Scan(&sourceBranchID); err != nil { return MessageRun{}, err }
		err = tx.QueryRowContext(ctx, `SELECT id::text FROM messages WHERE conversation_id=$1 AND branch_id=$2 AND sequence < (SELECT sequence FROM messages WHERE id=$3) ORDER BY sequence DESC LIMIT 1`, conversationID, sourceBranchID, parentMessageID).Scan(&copyThroughMessageID)
		if errors.Is(err, sql.ErrNoRows) { copyThroughMessageID = "" } else if err != nil { return MessageRun{}, err }
	} else {
		var sourceBranchID string
		if err = tx.QueryRowContext(ctx, `SELECT branch_id::text FROM messages WHERE id=$1`, messageID).Scan(&sourceBranchID); err != nil { return MessageRun{}, err }
		err = tx.QueryRowContext(ctx, `SELECT id::text FROM messages WHERE conversation_id=$1 AND branch_id=$2 AND sequence < (SELECT sequence FROM messages WHERE id=$3) ORDER BY sequence DESC LIMIT 1`, conversationID, sourceBranchID, messageID).Scan(&copyThroughMessageID)
		if errors.Is(err, sql.ErrNoRows) { copyThroughMessageID = "" } else if err != nil { return MessageRun{}, err }
	}
	branch, err := p.createConversationBranchAtTx(ctx, tx, user, conversationID, messageID, copyThroughMessageID, "active")
	if err != nil { return MessageRun{}, err }
	conversation, err := scanConversation(tx.QueryRowContext(ctx, `SELECT `+conversationColumns+` FROM conversations WHERE id=$1 AND user_id=$2 FOR UPDATE`, conversationID, user.ID))
	if err != nil { return MessageRun{}, err }
	conversation.ActiveBranchID = branch.ID
	selection, err := p.effectiveSelection(access, conversationID, modelID, profileID)
	if err != nil { return MessageRun{}, err }
	run, err := p.insertRun(ctx, tx, user, conversation, body, selection, "rerun-"+branch.ID)
	if err != nil { return MessageRun{}, err }
	if err = tx.Commit(); err != nil { return MessageRun{}, err }
	return run, nil
}

func (p *postgresConversationStore) createComparison(access,conversationID,prompt string,models []string,profile string)(ComparisonGroup,error){
	user, err := p.storeUser(access)
	if err != nil { return ComparisonGroup{}, err }
	prompt = strings.TrimSpace(prompt)
	if prompt == "" { return ComparisonGroup{}, errors.New("message_body_required") }
	if len([]rune(prompt)) > 200000 { return ComparisonGroup{}, errors.New("message_body_too_long") }
	if len(models) == 0 { models = []string{"ylven-default"} }
	if len(models) > 6 { return ComparisonGroup{}, errors.New("comparison_model_limit_exceeded") }
	ctx, cancel := databaseContext()
	defer cancel()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil { return ComparisonGroup{}, err }
	defer tx.Rollback()
	conversation, err := scanConversation(tx.QueryRowContext(ctx, `SELECT `+conversationColumns+` FROM conversations WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL FOR UPDATE`, conversationID, user.ID))
	if errors.Is(err, sql.ErrNoRows) { return ComparisonGroup{}, errors.New("conversation_not_found") }
	if err != nil { return ComparisonGroup{}, err }
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, conversation.ID); err != nil { return ComparisonGroup{}, err }
	id, err := newDatabaseUUID()
	if err != nil { return ComparisonGroup{}, err }
	now := time.Now().UTC()
	group := ComparisonGroup{ID:id, ConversationID:conversationID, UserID:user.ID, Prompt:prompt, Status:"queued", Candidates:[]ComparisonCandidate{}, CreatedAt:now, UpdatedAt:now}
	// Messages created for candidates reference comparison_groups. Persist the
	// group before creating those messages so the foreign key is valid.
	if _, err = tx.ExecContext(ctx, `INSERT INTO comparison_groups(id,conversation_id,user_id,prompt,status,candidates,created_at,updated_at) VALUES($1,$2,$3,$4,$5,'[]'::jsonb,$6,$6)`, group.ID, group.ConversationID, group.UserID, group.Prompt, group.Status, now); err != nil { return ComparisonGroup{}, err }
	seen := map[string]bool{}
	for _, requestedModelID := range models {
		requestedModelID = strings.TrimSpace(requestedModelID)
		if requestedModelID == "" || seen[requestedModelID] { continue }
		seen[requestedModelID] = true
		selection, selectionErr := p.resolveModelSelection(requestedModelID, profile)
		if selectionErr != nil {
			group.Candidates = append(group.Candidates, ComparisonCandidate{ModelID:requestedModelID, ReasoningProfile:strings.TrimSpace(profile), Status:"unavailable", ErrorCode:selectionErr.Error()})
			continue
		}
		branchID, branchErr := p.createComparisonBranchTx(ctx, tx, user, conversation, id)
		if branchErr != nil { return ComparisonGroup{}, branchErr }
		candidateConversation := conversation
		candidateConversation.ActiveBranchID = branchID
		run, runErr := p.insertRun(ctx, tx, user, candidateConversation, prompt, selection, "comparison-"+id+"-"+selection.CatalogModelID)
		if runErr != nil { return ComparisonGroup{}, runErr }
		if _, runErr = tx.ExecContext(ctx, `UPDATE messages SET comparison_group_id=$2 WHERE id=$1`, run.UserMessageID, group.ID); runErr != nil { return ComparisonGroup{}, runErr }
		group.Candidates = append(group.Candidates, ComparisonCandidate{RunID:run.ID, ModelID:selection.CatalogModelID, ReasoningProfile:selection.ReasoningProfile, Status:"streaming", MessageID:run.UserMessageID})
	}
	if len(group.Candidates) == 0 { group.Status = "unavailable" } else {
		ready := false
		for _, candidate := range group.Candidates { if candidate.RunID != "" { ready = true; break } }
		if ready { group.Status = "running" } else { group.Status = "unavailable" }
	}
	raw, err := json.Marshal(group.Candidates)
	if err != nil { return ComparisonGroup{}, err }
	_, err = tx.ExecContext(ctx, `UPDATE comparison_groups SET status=$2,candidates=$3::jsonb,updated_at=$4 WHERE id=$1`, group.ID, group.Status, string(raw), now)
	if err != nil { return ComparisonGroup{}, err }
	if err = tx.Commit(); err != nil { return ComparisonGroup{}, err }
	return group, nil
}

func (p *postgresConversationStore) createComparisonBranchTx(ctx context.Context, tx *sql.Tx, user User, conversation Conversation, groupID string) (string, error) {
	var forkMessageID string
	err := tx.QueryRowContext(ctx, `SELECT id::text FROM messages WHERE conversation_id=$1 AND branch_id=$2 ORDER BY sequence DESC LIMIT 1`, conversation.ID, conversation.ActiveBranchID).Scan(&forkMessageID)
	if errors.Is(err, sql.ErrNoRows) { forkMessageID = "" } else if err != nil { return "", err }
	branchID, err := newDatabaseUUID()
	if err != nil { return "", err }
	now := time.Now().UTC()
	_, err = tx.ExecContext(ctx, `INSERT INTO conversation_branches(id,conversation_id,parent_branch_id,forked_from_message_id,status,created_at,updated_at) VALUES($1,$2,$3,NULLIF($4,'')::uuid,'comparison',$5,$5)`, branchID, conversation.ID, conversation.ActiveBranchID, forkMessageID, now)
	if err != nil { return "", err }
	if err = p.cloneBranchHistoryTx(ctx, tx, conversation.ID, conversation.ActiveBranchID, branchID, forkMessageID); err != nil { return "", err }
	return branchID, nil
}

func (p *postgresConversationStore) getComparison(access, comparisonID string) (ComparisonGroup, error) {
	user, err := p.storeUser(access)
	if err != nil { return ComparisonGroup{}, err }
	ctx, cancel := databaseContext()
	defer cancel()
	var group ComparisonGroup
	var raw []byte
	err = p.db.QueryRowContext(ctx, `SELECT id::text,conversation_id::text,user_id::text,prompt,status,candidates,COALESCE(adopted_run_id,''),COALESCE(synthesis_run_id,''),created_at,updated_at FROM comparison_groups WHERE id=$1 AND user_id=$2`, comparisonID, user.ID).Scan(&group.ID, &group.ConversationID, &group.UserID, &group.Prompt, &group.Status, &raw, &group.AdoptedRunID, &group.SynthesisRunID, &group.CreatedAt, &group.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) { return ComparisonGroup{}, errors.New("comparison_not_found") }
	if err != nil { return ComparisonGroup{}, err }
	if err = json.Unmarshal(raw, &group.Candidates); err != nil { return ComparisonGroup{}, err }
	for index := range group.Candidates {
		candidate := &group.Candidates[index]
		if candidate.RunID == "" { continue }
		var assistantMessageID string
		err = p.db.QueryRowContext(ctx, `SELECT status,COALESCE(assistant_message_id::text,''),COALESCE(error_code,'') FROM message_runs WHERE id=$1 AND user_id=$2`, candidate.RunID, user.ID).Scan(&candidate.Status, &assistantMessageID, &candidate.ErrorCode)
		if err != nil { return ComparisonGroup{}, err }
		candidate.MessageID = assistantMessageID
		if assistantMessageID != "" { _ = p.db.QueryRowContext(ctx, `SELECT body FROM messages WHERE id=$1`, assistantMessageID).Scan(&candidate.Body) }
	}
	if group.SynthesisRunID != "" {
		var synthesisStatus string
		if err = p.db.QueryRowContext(ctx, `SELECT status FROM message_runs WHERE id=$1 AND user_id=$2`, group.SynthesisRunID, user.ID).Scan(&synthesisStatus); err == nil {
			switch synthesisStatus { case "completed": group.Status = "synthesized"; case "failed", "cancelled": group.Status = "synthesis_failed"; default: group.Status = "synthesizing" }
		}
	} else if group.AdoptedRunID == "" { group.Status = comparisonAggregateStatus(group.Candidates) }
	return group, nil
}

func (p *postgresConversationStore) listComparisonGroups() ([]ComparisonGroup, error) {
	ctx, cancel := databaseContext(); defer cancel()
	rows, err := p.db.QueryContext(ctx, `SELECT id::text,conversation_id::text,user_id::text,prompt,status,candidates,
      COALESCE(adopted_run_id::text,''),COALESCE(synthesis_run_id::text,''),created_at,updated_at
      FROM comparison_groups ORDER BY updated_at DESC,id DESC LIMIT $1`, p04AdminListLimit)
	if err != nil { return nil, err }
	defer rows.Close()
	items := []ComparisonGroup{}
	for rows.Next() {
		var item ComparisonGroup; var raw []byte
		if err = rows.Scan(&item.ID,&item.ConversationID,&item.UserID,&item.Prompt,&item.Status,&raw,&item.AdoptedRunID,&item.SynthesisRunID,&item.CreatedAt,&item.UpdatedAt); err != nil { return nil, err }
		if err = json.Unmarshal(raw, &item.Candidates); err != nil { return nil, err }
		items = append(items, item)
	}
	return items, rows.Err()
}

func comparisonAggregateStatus(candidates []ComparisonCandidate) string {
	if len(candidates) == 0 { return "unavailable" }
	active, completed := false, false
	for _, candidate := range candidates {
		switch candidate.Status {
		case "queued", "running", "streaming": active = true
		case "completed": completed = true
		}
	}
	if active { return "running" }
	if completed { return "completed" }
	return "failed"
}

func (p *postgresConversationStore) updateComparisonStatusForRunTx(ctx context.Context, tx *sql.Tx, runID string, now time.Time) error {
	var groupID string
	err := tx.QueryRowContext(ctx, `SELECT COALESCE(m.comparison_group_id::text,'') FROM message_runs r JOIN messages m ON m.id=r.user_message_id WHERE r.id=$1`, runID).Scan(&groupID)
	if errors.Is(err, sql.ErrNoRows) || groupID == "" { return nil }
	if err != nil { return err }
	var group ComparisonGroup
	var raw []byte
	err = tx.QueryRowContext(ctx, `SELECT id::text,conversation_id::text,user_id::text,prompt,status,candidates,COALESCE(adopted_run_id,''),COALESCE(synthesis_run_id,''),created_at,updated_at FROM comparison_groups WHERE id=$1 FOR UPDATE`, groupID).Scan(&group.ID, &group.ConversationID, &group.UserID, &group.Prompt, &group.Status, &raw, &group.AdoptedRunID, &group.SynthesisRunID, &group.CreatedAt, &group.UpdatedAt)
	if err != nil { return err }
	if err = json.Unmarshal(raw, &group.Candidates); err != nil { return err }
	for index := range group.Candidates {
		candidate := &group.Candidates[index]
		if candidate.RunID == "" { continue }
		var assistantMessageID string
		if err = tx.QueryRowContext(ctx, `SELECT status,COALESCE(assistant_message_id::text,''),COALESCE(error_code,'') FROM message_runs WHERE id=$1`, candidate.RunID).Scan(&candidate.Status, &assistantMessageID, &candidate.ErrorCode); err != nil { return err }
		candidate.MessageID = assistantMessageID
	}
	if group.SynthesisRunID == runID {
		var synthesisStatus string
		if err = tx.QueryRowContext(ctx, `SELECT status FROM message_runs WHERE id=$1`, runID).Scan(&synthesisStatus); err != nil { return err }
		switch synthesisStatus { case "completed": group.Status = "synthesized"; case "failed", "cancelled": group.Status = "synthesis_failed"; default: group.Status = "synthesizing" }
		group.UpdatedAt = now
		_, err = tx.ExecContext(ctx, `UPDATE comparison_groups SET status=$2,updated_at=$3 WHERE id=$1`, group.ID, group.Status, now)
		return err
	}
	if group.AdoptedRunID != "" { return nil }
	group.Status, group.UpdatedAt = comparisonAggregateStatus(group.Candidates), now
	raw, err = json.Marshal(group.Candidates)
	if err != nil { return err }
	_, err = tx.ExecContext(ctx, `UPDATE comparison_groups SET status=$2,candidates=$3::jsonb,updated_at=$4 WHERE id=$1`, group.ID, group.Status, string(raw), now)
	return err
}

func (p *postgresConversationStore) adoptComparison(access, comparisonID, runID string) (ComparisonGroup, error) {
	user, err := p.storeUser(access)
	if err != nil { return ComparisonGroup{}, err }
	ctx, cancel := databaseContext(); defer cancel()
	tx, err := p.db.BeginTx(ctx, nil); if err != nil { return ComparisonGroup{}, err }; defer tx.Rollback()
	var group ComparisonGroup; var raw []byte
	err = tx.QueryRowContext(ctx, `SELECT id::text,conversation_id::text,user_id::text,prompt,status,candidates,COALESCE(adopted_run_id,''),COALESCE(synthesis_run_id,''),created_at,updated_at FROM comparison_groups WHERE id=$1 AND user_id=$2 FOR UPDATE`, comparisonID, user.ID).Scan(&group.ID,&group.ConversationID,&group.UserID,&group.Prompt,&group.Status,&raw,&group.AdoptedRunID,&group.SynthesisRunID,&group.CreatedAt,&group.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) { return ComparisonGroup{}, errors.New("comparison_not_found") }; if err != nil { return ComparisonGroup{}, err }
	if err = json.Unmarshal(raw, &group.Candidates); err != nil { return ComparisonGroup{}, err }
	found := false
	for _, candidate := range group.Candidates { if candidate.RunID == runID { found = true; break } }
	if !found { return ComparisonGroup{}, errors.New("comparison_candidate_not_found") }
	var run MessageRun
	run, err = scanRun(tx.QueryRowContext(ctx, `SELECT `+runColumns+` FROM message_runs WHERE id=$1 AND user_id=$2 AND conversation_id=$3 FOR UPDATE`, runID, user.ID, group.ConversationID))
	if errors.Is(err, sql.ErrNoRows) { return ComparisonGroup{}, errors.New("comparison_candidate_not_found") }; if err != nil { return ComparisonGroup{}, err }
	if run.Status != "completed" || run.AssistantMessageID == "" { return ComparisonGroup{}, errors.New("comparison_candidate_not_ready") }
	now := time.Now().UTC()
	_, err = tx.ExecContext(ctx, `UPDATE conversations SET active_branch_id=$2,updated_at=$3 WHERE id=$1 AND user_id=$4`, group.ConversationID, run.BranchID, now, user.ID)
	if err != nil { return ComparisonGroup{}, err }
	group.AdoptedRunID, group.Status, group.UpdatedAt = run.ID, "adopted", now
	raw, err = json.Marshal(group.Candidates); if err != nil { return ComparisonGroup{}, err }
	_, err = tx.ExecContext(ctx, `UPDATE comparison_groups SET adopted_run_id=$2,status=$3,candidates=$4::jsonb,updated_at=$5 WHERE id=$1`, group.ID, group.AdoptedRunID, group.Status, string(raw), now)
	if err != nil { return ComparisonGroup{}, err }
	if err = tx.Commit(); err != nil { return ComparisonGroup{}, err }; return group, nil
}

func (p *postgresConversationStore) synthesizeComparison(access, comparisonID string, runIDs []string) (ComparisonGroup, error) {
	user, err := p.storeUser(access); if err != nil { return ComparisonGroup{}, err }
	ctx, cancel := databaseContext(); defer cancel()
	tx, err := p.db.BeginTx(ctx, nil); if err != nil { return ComparisonGroup{}, err }; defer tx.Rollback()
	var group ComparisonGroup; var raw []byte
	err = tx.QueryRowContext(ctx, `SELECT id::text,conversation_id::text,user_id::text,prompt,status,candidates,COALESCE(adopted_run_id,''),COALESCE(synthesis_run_id,''),created_at,updated_at FROM comparison_groups WHERE id=$1 AND user_id=$2 FOR UPDATE`, comparisonID, user.ID).Scan(&group.ID,&group.ConversationID,&group.UserID,&group.Prompt,&group.Status,&raw,&group.AdoptedRunID,&group.SynthesisRunID,&group.CreatedAt,&group.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) { return ComparisonGroup{}, errors.New("comparison_not_found") }; if err != nil { return ComparisonGroup{}, err }
	if err = json.Unmarshal(raw, &group.Candidates); err != nil { return ComparisonGroup{}, err }
	requested := map[string]bool{}
	for _, id := range runIDs { if strings.TrimSpace(id) != "" { requested[id] = true } }
	if len(requested) == 0 { for _, candidate := range group.Candidates { if candidate.RunID != "" { requested[candidate.RunID] = true } } }
	if len(requested) == 0 { return ComparisonGroup{}, errors.New("comparison_candidate_not_found") }
	sections := []string{}
	for _, candidate := range group.Candidates {
		if !requested[candidate.RunID] { continue }
		var run MessageRun
		run, err = scanRun(tx.QueryRowContext(ctx, `SELECT `+runColumns+` FROM message_runs WHERE id=$1 AND user_id=$2 AND conversation_id=$3`, candidate.RunID, user.ID, group.ConversationID))
		if errors.Is(err, sql.ErrNoRows) { return ComparisonGroup{}, errors.New("comparison_candidate_not_found") }; if err != nil { return ComparisonGroup{}, err }
		if run.Status != "completed" || run.AssistantMessageID == "" { return ComparisonGroup{}, errors.New("comparison_candidate_not_ready") }
		var body string; if err = tx.QueryRowContext(ctx, `SELECT body FROM messages WHERE id=$1`, run.AssistantMessageID).Scan(&body); err != nil { return ComparisonGroup{}, err }
		sections = append(sections, "["+candidate.ModelID+"]\n"+body)
	}
	if len(sections) == 0 { return ComparisonGroup{}, errors.New("comparison_candidate_not_found") }
	conversation, err := scanConversation(tx.QueryRowContext(ctx, `SELECT `+conversationColumns+` FROM conversations WHERE id=$1 AND user_id=$2 FOR UPDATE`, group.ConversationID, user.ID)); if err != nil { return ComparisonGroup{}, err }
	selection, err := p.effectiveSelection(access, group.ConversationID, "", "")
	if err != nil { return ComparisonGroup{}, err }
	prompt := "Synthesize the candidate answers below into one accurate answer. State uncertainty where candidates disagree. Do not mention hidden reasoning.\n\nOriginal request:\n"+group.Prompt+"\n\nCandidates:\n"+strings.Join(sections, "\n\n")
	run, err := p.insertRun(ctx, tx, user, conversation, prompt, selection, "synthesis-"+group.ID); if err != nil { return ComparisonGroup{}, err }
	if _, err = tx.ExecContext(ctx, `UPDATE messages SET comparison_group_id=$2 WHERE id=$1`, run.UserMessageID, group.ID); err != nil { return ComparisonGroup{}, err }
	now := time.Now().UTC(); group.SynthesisRunID, group.Status, group.UpdatedAt = run.ID, "synthesizing", now
	_, err = tx.ExecContext(ctx, `UPDATE comparison_groups SET synthesis_run_id=$2,status=$3,updated_at=$4 WHERE id=$1`, group.ID, group.SynthesisRunID, group.Status, now); if err != nil { return ComparisonGroup{}, err }
	if err = tx.Commit(); err != nil { return ComparisonGroup{}, err }; return group, nil
}

func (p *postgresConversationStore) modelAvailability(access, modelID string) (map[string]any, error) {
	if _, err := p.storeUser(access); err != nil {
		return nil, err
	}
	modelID = strings.TrimSpace(modelID)
	ctx, cancel := databaseContext()
	defer cancel()
	var health ModelHealthStatus
	var raw []byte
	err := p.db.QueryRowContext(ctx, `SELECT model_id,provider_id,status,latency_ms,last_probe_at,capabilities,COALESCE(error_code,'') FROM model_health_status WHERE model_id=$1`, modelID).Scan(
		&health.ModelID, &health.ProviderID, &health.Status, &health.LatencyMs, &health.LastProbeAt, &raw, &health.ErrorCode)
	if errors.Is(err, sql.ErrNoRows) {
		if err = p.db.QueryRowContext(ctx, `SELECT provider_id FROM models WHERE id=$1`, modelID).Scan(&health.ProviderID); errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("model_not_found")
		}
		if err != nil {
			return nil, err
		}
		health.ModelID, health.Status, health.Capabilities = modelID, "available", []string{"text"}
	} else if err != nil {
		return nil, err
	} else if err = json.Unmarshal(raw, &health.Capabilities); err != nil {
		return nil, err
	}
	catalog, err := p.loadModelCatalog()
	if err != nil { return nil, err }
	healthItems, err := p.listModelHealthStatuses()
	if err != nil { return nil, err }
	healthByModel := make(map[string]ModelHealthStatus, len(healthItems))
	for _, item := range healthItems { healthByModel[item.ModelID] = item }
	fallbacks := []ModelCatalogEntry{}
	for _, model := range catalog {
		if model.ID == modelID || !model.Enabled { continue }
		candidateHealth, found := healthByModel[model.ID]
		if found && candidateHealth.Status == "unavailable" { continue }
		fallbacks = append(fallbacks, model)
	}
	return map[string]any{"model_id":modelID, "status":health.Status, "health":health, "fallback_models":fallbacks}, nil
}

func (p *postgresConversationStore) recordModelHealth(access, modelID, status, errorCode string, latency int64, capabilities []string) (ModelHealthStatus, error) {
	if _, err := p.storeUser(access); err != nil {
		return ModelHealthStatus{}, err
	}
	return p.recordModelHealthAdmin(ModelHealthStatus{ModelID:modelID, Status:status, ErrorCode:errorCode, LatencyMs:latency, Capabilities:capabilities, LastProbeAt:time.Now().UTC()}, "runtime")
}

func (p *postgresConversationStore) serviceStatus(access string) ([]ModelHealthStatus, error) {
	if _, err := p.storeUser(access); err != nil {
		return nil, err
	}
	ctx, cancel := databaseContext()
	defer cancel()
	rows, err := p.db.QueryContext(ctx, `SELECT m.id,m.provider_id,COALESCE(h.status,'available'),COALESCE(h.latency_ms,0),h.last_probe_at,COALESCE(h.capabilities,'["text"]'::jsonb),COALESCE(h.error_code,'') FROM models m LEFT JOIN model_health_status h ON h.model_id=m.id WHERE m.enabled ORDER BY m.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ModelHealthStatus{}
	for rows.Next() {
		var item ModelHealthStatus
		var probeAt sql.NullTime
		var raw []byte
		if err = rows.Scan(&item.ModelID, &item.ProviderID, &item.Status, &item.LatencyMs, &probeAt, &raw, &item.ErrorCode); err != nil {
			return nil, err
		}
		if probeAt.Valid {
			item.LastProbeAt = probeAt.Time
		}
		if err = json.Unmarshal(raw, &item.Capabilities); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func (p *postgresConversationStore) recordUsage(access string, event UsageEvent) (UsageEvent, error) {
	user, err := p.storeUser(access)
	if err != nil {
		return UsageEvent{}, err
	}
	return p.recordUsageForUser(user.ID, event)
}

func (p *postgresConversationStore) recordUsageForUser(userID string, event UsageEvent) (UsageEvent, error) {
	if event.ID == "" {
		event.ID, _ = newDatabaseUUID()
	}
	event.ModelID = strings.TrimSpace(event.ModelID)
	if event.ID == "" || event.ModelID == "" || event.InputTokens < 0 || event.OutputTokens < 0 || event.ReasoningTokens < 0 {
		return UsageEvent{}, errors.New("usage_event_invalid")
	}
	event.UserID = userID
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	ctx, cancel := databaseContext()
	defer cancel()
	result, err := p.db.ExecContext(ctx, `INSERT INTO ai_usage_events(id,user_id,run_id,model_id,input_tokens,output_tokens,reasoning_tokens,price_version_id,created_at) VALUES($1,$2,NULLIF($3,'')::uuid,$4,$5,$6,$7,$8,$9) ON CONFLICT DO NOTHING`,
		event.ID, event.UserID, event.RunID, event.ModelID, event.InputTokens, event.OutputTokens, event.ReasoningTokens, event.PriceVersionID, event.CreatedAt)
	if err != nil {
		return UsageEvent{}, err
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return UsageEvent{}, err
	}
	if inserted == 0 {
		return UsageEvent{}, errors.New("usage_event_already_recorded")
	}
	return event, nil
}

func (p *postgresConversationStore) listUsageEvents() ([]UsageEvent, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	rows, err := p.db.QueryContext(ctx, `SELECT id::text,user_id::text,COALESCE(run_id::text,''),model_id,input_tokens,output_tokens,reasoning_tokens,price_version_id,created_at FROM ai_usage_events ORDER BY created_at DESC,id DESC LIMIT $1`, p04AdminListLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []UsageEvent{}
	for rows.Next() {
		var item UsageEvent
		if err := rows.Scan(&item.ID, &item.UserID, &item.RunID, &item.ModelID, &item.InputTokens, &item.OutputTokens, &item.ReasoningTokens, &item.PriceVersionID, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (p *postgresConversationStore) listPriceSnapshots() ([]PriceSnapshot, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	rows, err := p.db.QueryContext(ctx, `SELECT id,model_id,input_per_million,output_per_million,reasoning_per_million,effective_at,version FROM model_price_snapshots ORDER BY effective_at DESC,id ASC LIMIT $1`, p04AdminListLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []PriceSnapshot{}
	for rows.Next() {
		var item PriceSnapshot
		if err := rows.Scan(&item.ID, &item.ModelID, &item.InputPerMillion, &item.OutputPerMillion, &item.ReasoningPerMillion, &item.EffectiveAt, &item.Version); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (p *postgresConversationStore) upsertPriceSnapshotAdmin(value PriceSnapshot, actorID string) (PriceSnapshot, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return PriceSnapshot{}, err
	}
	defer tx.Rollback()
	var modelExists bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM models WHERE id=$1)`, value.ModelID).Scan(&modelExists); err != nil {
		return PriceSnapshot{}, err
	}
	if !modelExists {
		return PriceSnapshot{}, errors.New("model_not_found")
	}
	if value.ID == "" {
		id, idErr := newDatabaseUUID()
		if idErr != nil {
			return PriceSnapshot{}, idErr
		}
		value.ID = "price-" + id
	}
	if value.EffectiveAt.IsZero() {
		value.EffectiveAt = time.Now().UTC()
	}
	var before PriceSnapshot
	err = tx.QueryRowContext(ctx, `SELECT id,model_id,input_per_million,output_per_million,reasoning_per_million,effective_at,version FROM model_price_snapshots WHERE id=$1 FOR UPDATE`, value.ID).Scan(&before.ID, &before.ModelID, &before.InputPerMillion, &before.OutputPerMillion, &before.ReasoningPerMillion, &before.EffectiveAt, &before.Version)
	action := "update"
	if errors.Is(err, sql.ErrNoRows) {
		if value.Version != 0 && value.Version != 1 {
			return PriceSnapshot{}, errors.New("version_conflict")
		}
		value.Version = 1
		action = "create"
		_, err = tx.ExecContext(ctx, `INSERT INTO model_price_snapshots(id,model_id,input_per_million,output_per_million,reasoning_per_million,effective_at,version) VALUES($1,$2,$3,$4,$5,$6,$7)`, value.ID, value.ModelID, value.InputPerMillion, value.OutputPerMillion, value.ReasoningPerMillion, value.EffectiveAt, value.Version)
	} else if err != nil {
		return PriceSnapshot{}, err
	} else {
		if value.Version != before.Version {
			return PriceSnapshot{}, errors.New("version_conflict")
		}
		value.Version = before.Version + 1
		_, err = tx.ExecContext(ctx, `UPDATE model_price_snapshots SET model_id=$2,input_per_million=$3,output_per_million=$4,reasoning_per_million=$5,effective_at=$6,version=$7 WHERE id=$1`, value.ID, value.ModelID, value.InputPerMillion, value.OutputPerMillion, value.ReasoningPerMillion, value.EffectiveAt, value.Version)
	}
	if err != nil {
		return PriceSnapshot{}, err
	}
	if err = insertModelCatalogAudit(ctx, tx, actorID, action, "price_snapshot", value.ID, before, value); err != nil {
		return PriceSnapshot{}, err
	}
	if err = tx.Commit(); err != nil {
		return PriceSnapshot{}, err
	}
	return value, nil
}

func (p *postgresConversationStore) listModelHealthStatuses() ([]ModelHealthStatus, error) {
	ctx, cancel := databaseContext()
	defer cancel()
	rows, err := p.db.QueryContext(ctx, `SELECT m.id,m.provider_id,COALESCE(h.status,'available'),COALESCE(h.latency_ms,0),h.last_probe_at,COALESCE(h.capabilities,'["text"]'::jsonb),COALESCE(h.error_code,'') FROM models m LEFT JOIN model_health_status h ON h.model_id=m.id WHERE m.enabled ORDER BY m.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ModelHealthStatus{}
	for rows.Next() {
		var item ModelHealthStatus
		var probeAt sql.NullTime
		var raw []byte
		if err = rows.Scan(&item.ModelID, &item.ProviderID, &item.Status, &item.LatencyMs, &probeAt, &raw, &item.ErrorCode); err != nil {
			return nil, err
		}
		if probeAt.Valid {
			item.LastProbeAt = probeAt.Time
		}
		if err = json.Unmarshal(raw, &item.Capabilities); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (p *postgresConversationStore) upsertPriceSnapshot(access string, value PriceSnapshot) (PriceSnapshot, error) {
	if _, err := p.storeUser(access); err != nil {
		return PriceSnapshot{}, err
	}
	value.ModelID = strings.TrimSpace(value.ModelID)
	if value.ModelID == "" || value.InputPerMillion < 0 || value.OutputPerMillion < 0 || value.ReasoningPerMillion < 0 {
		return PriceSnapshot{}, errors.New("price_snapshot_invalid")
	}
	if value.ID == "" {
		value.ID = "price-" + value.ModelID + "-" + time.Now().UTC().Format("20060102150405")
	}
	if value.Version == 0 {
		value.Version = 1
	}
	if value.EffectiveAt.IsZero() {
		value.EffectiveAt = time.Now().UTC()
	}
	ctx, cancel := databaseContext()
	defer cancel()
	var exists bool
	if err := p.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM models WHERE id=$1)`, value.ModelID).Scan(&exists); err != nil {
		return PriceSnapshot{}, err
	}
	if !exists {
		return PriceSnapshot{}, errors.New("model_not_found")
	}
	_, err := p.db.ExecContext(ctx, `INSERT INTO model_price_snapshots(id,model_id,input_per_million,output_per_million,reasoning_per_million,effective_at,version) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(id) DO UPDATE SET input_per_million=EXCLUDED.input_per_million,output_per_million=EXCLUDED.output_per_million,reasoning_per_million=EXCLUDED.reasoning_per_million,effective_at=EXCLUDED.effective_at,version=EXCLUDED.version`,
		value.ID, value.ModelID, value.InputPerMillion, value.OutputPerMillion, value.ReasoningPerMillion, value.EffectiveAt, value.Version)
	return value, err
}
