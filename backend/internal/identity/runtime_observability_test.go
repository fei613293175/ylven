package identity

import "testing"

func TestP03W08ContextDebugHealthAndDistributedRunAreDurableInStore(t *testing.T) {
	store, err := NewStore("")
	if err != nil {
		t.Fatal(err)
	}
	createTestUser(t, store, "w08-observability@example.com")
	access := createAuthenticatedTestSession(t, store, "w08-observability@example.com")
	first, err := store.StartConversationFromFirstMessage(access, "w08-draft", "请保留这条上下文事实", "gpt-5.6", "w08-first", false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CompileRunContext(access, first.Run.ID, "local_rebuild"); err != nil {
		t.Fatal(err)
	}
	debug, err := store.ContextDebugAdmin(first.Run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if debug.RunID != first.Run.ID || debug.ContextBuildID == "" || len(debug.Items) == 0 {
		t.Fatalf("unexpected context debug: %+v", debug)
	}
	if debug.Items[0].ContentLength == 0 && debug.Items[0].ContentPreview != "" {
		t.Fatalf("preview length metadata is inconsistent: %+v", debug.Items[0])
	}

	health, err := store.AIHealth(access, "unconfigured")
	if err != nil {
		t.Fatal(err)
	}
	if health.Status != "degraded" || len(health.Instances) != 1 || len(health.Checks) != 1 {
		t.Fatalf("unexpected runtime health: %+v", health)
	}

	testRun := DistributedTestRun{ID: "00000000-0000-0000-0000-000000000008", TestKey: "p03-distributed-chat", Status: "pass", InstanceA: "a", InstanceB: "b", CursorBefore: 2, CursorAfter: 4}
	saved, created, err := store.RecordDistributedTestRun(testRun)
	if err != nil || !created || saved.ID != testRun.ID {
		t.Fatalf("saved test run=%+v created=%v err=%v", saved, created, err)
	}
	duplicate, created, err := store.RecordDistributedTestRun(testRun)
	if err != nil || created || duplicate.ID != testRun.ID {
		t.Fatalf("duplicate test run=%+v created=%v err=%v", duplicate, created, err)
	}
	conflict := testRun
	conflict.CursorAfter = 5
	if _, _, err := store.RecordDistributedTestRun(conflict); err == nil || err.Error() != "idempotency_conflict" {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}
	progress := DistributedTestRun{ID: "00000000-0000-0000-0000-000000000009", TestKey: "p03-distributed-chat", Status: "running", InstanceA: "a", InstanceB: "b", CursorBefore: 2, CursorAfter: 2}
	if _, _, err := store.RecordDistributedTestRun(progress); err != nil {
		t.Fatal(err)
	}
	progress.Status = "pass"
	progress.CursorAfter = 4
	if updated, created, err := store.RecordDistributedTestRun(progress); err != nil || created || updated.Status != "pass" {
		t.Fatalf("expected running-to-pass transition: result=%+v created=%v err=%v", updated, created, err)
	}
}
