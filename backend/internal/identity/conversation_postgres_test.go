package identity

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestP03W06PostgresFirstMessageTransactionAndIdempotency(t *testing.T) {
	dsn := os.Getenv("YLVEN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("YLVEN_TEST_DATABASE_URL is required for PostgreSQL integration coverage")
	}
	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("ylven_p03_w06_%d", time.Now().UnixNano())
	if _, err := admin.ExecContext(context.Background(), `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.ExecContext(context.Background(), `DROP SCHEMA `+schema+` CASCADE`); err != nil {
			t.Errorf("drop test schema: %v", err)
		}
	}()
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()

	store, _ := NewStore("")
	createTestUser(t, store, "postgres-context@example.com")
	access := createAuthenticatedTestSession(t, store, "postgres-context@example.com")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := store.EnablePostgres(ctx, parsed.String()); err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	first, err := store.StartConversationFromFirstMessage(access, "postgres-draft", "我叫陈平", "gpt-5.6", "postgres-first", false)
	if err != nil || !first.Created {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	duplicate, err := store.StartConversationFromFirstMessage(access, "postgres-draft", "我叫陈平", "gpt-5.6", "postgres-first", false)
	if err != nil || duplicate.Created || duplicate.Run.ID != first.Run.ID {
		t.Fatalf("duplicate=%+v err=%v", duplicate, err)
	}
	if _, err := store.CompleteRun(access, first.Run.ID, "好的，我记住了。你叫陈平。"); err != nil {
		t.Fatal(err)
	}
	finalized, err := store.ApplyAutomaticConversationTitle(access, first.Conversation.ID, "姓名记忆测试会话", "AUTO_FINAL")
	if err != nil || finalized.TitleSource != "AUTO_FINAL" || finalized.TitleLocked {
		t.Fatalf("finalized=%+v err=%v", finalized, err)
	}
	unchanged, err := store.ApplyAutomaticConversationTitle(access, first.Conversation.ID, "不应二次覆盖标题", "AUTO_FINAL")
	if err != nil || unchanged.Title != finalized.Title {
		t.Fatalf("automatic title changed twice: %+v err=%v", unchanged, err)
	}

	var conversations, messages, runs, idempotency int
	for queryText, target := range map[string]*int{
		"SELECT COUNT(*) FROM conversations":       &conversations,
		"SELECT COUNT(*) FROM messages":            &messages,
		"SELECT COUNT(*) FROM message_runs":        &runs,
		"SELECT COUNT(*) FROM idempotency_records": &idempotency,
	} {
		if err := store.conversationSQL.db.QueryRowContext(context.Background(), queryText).Scan(target); err != nil {
			t.Fatal(err)
		}
	}
	if conversations != 1 || messages != 2 || runs != 1 || idempotency != 1 {
		t.Fatalf("unexpected durable counts conversations=%d messages=%d runs=%d idempotency=%d", conversations, messages, runs, idempotency)
	}
	var titleJobStatus string
	if err := store.conversationSQL.db.QueryRowContext(context.Background(), `SELECT status FROM conversation_title_jobs WHERE conversation_id=$1`, first.Conversation.ID).Scan(&titleJobStatus); err != nil {
		t.Fatal(err)
	}
	if titleJobStatus != "completed" {
		t.Fatalf("title job status=%s", titleJobStatus)
	}
	_, events, err := store.Events(access, first.Run.ID, 0)
	if err != nil || len(events) < 2 || events[len(events)-1].Type != "completed" {
		t.Fatalf("events=%+v err=%v", events, err)
	}

	if _, err := store.conversationSQL.db.ExecContext(context.Background(), `UPDATE model_capabilities SET
    context_limit_tokens=4096,output_reserve_tokens=512,reasoning_reserve_tokens=256,
    tool_reserve_tokens=128,safety_margin_tokens=128 WHERE model_id='gpt-5.6'`); err != nil {
		t.Fatal(err)
	}
	second, err := store.StartRunIdempotent(access, first.Conversation.ID, "请复述我的名字", "gpt-5.6", "postgres-second")
	if err != nil {
		t.Fatal(err)
	}
	build, err := store.CompileRunContext(access, second.ID, "local_rebuild")
	if err != nil {
		t.Fatal(err)
	}
	if build.ContextLimitTokens != 4096 || build.InputBudgetTokens != 3072 || !strings.Contains(providerMessagesText(build.Messages), "我叫陈平") {
		t.Fatalf("database capability/context build=%+v", build)
	}
	snapshot, err := store.ConversationContext(access, first.Conversation.ID)
	if err != nil || len(snapshot.Messages) != 3 || len(snapshot.MessageParts) != 3 {
		t.Fatalf("postgres context messages=%d parts=%d err=%v", len(snapshot.Messages), len(snapshot.MessageParts), err)
	}
	queued, created, err := store.QueueConversationCompaction(access, first.Conversation.ID, "postgres-compact")
	if err != nil || !created || queued.Status != "queued" {
		t.Fatalf("postgres queue=%+v created=%v err=%v", queued, created, err)
	}
	completed, err := store.ProcessConversationCompaction(access, queued.ID)
	if err != nil || completed.Status != "success" || completed.SummaryID == "" || completed.Attempts != 1 {
		t.Fatalf("postgres compaction=%+v err=%v", completed, err)
	}
	snapshot, err = store.ConversationContext(access, first.Conversation.ID)
	if err != nil || snapshot.Summary == nil || len(snapshot.Messages) != 3 {
		t.Fatalf("postgres compacted snapshot=%+v err=%v", snapshot, err)
	}
	for table, expectedMinimum := range map[string]int{"jobs": 1, "context_compactions": 1, "conversation_summaries": 1, "model_capabilities": 5} {
		var count int
		if err := store.conversationSQL.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM `+table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count < expectedMinimum {
			t.Fatalf("table %s count=%d expected >=%d", table, count, expectedMinimum)
		}
	}
}
