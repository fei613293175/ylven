-- P03-W06: canonical multi-turn conversation history and traceable context builds.
CREATE TABLE IF NOT EXISTS conversation_branches (
  id UUID PRIMARY KEY,
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  parent_branch_id UUID REFERENCES conversation_branches(id),
  forked_from_message_id UUID,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS conversation_branches_conversation_idx
  ON conversation_branches (conversation_id, created_at);

ALTER TABLE conversations ADD COLUMN IF NOT EXISTS title_source TEXT NOT NULL DEFAULT 'AUTO_TEMP';
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS title_locked BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS active_branch_id UUID;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS summary_through_message_id UUID;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS temporary BOOLEAN NOT NULL DEFAULT FALSE;

INSERT INTO conversation_branches (id, conversation_id, created_at, updated_at)
SELECT id, id, created_at, updated_at FROM conversations
ON CONFLICT (id) DO NOTHING;
UPDATE conversations SET active_branch_id = id WHERE active_branch_id IS NULL;

ALTER TABLE messages ADD COLUMN IF NOT EXISTS branch_id UUID;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS sequence BIGINT;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS parent_message_id UUID;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS comparison_group_id UUID;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'completed';
ALTER TABLE messages ADD COLUMN IF NOT EXISTS completed_at TIMESTAMPTZ;

UPDATE messages SET branch_id = conversation_id WHERE branch_id IS NULL;
WITH ordered AS (
  SELECT id, ROW_NUMBER() OVER (PARTITION BY conversation_id, branch_id ORDER BY created_at, id) AS position
  FROM messages
  WHERE sequence IS NULL
)
UPDATE messages SET sequence = ordered.position
FROM ordered WHERE messages.id = ordered.id;
UPDATE messages SET completed_at = created_at WHERE completed_at IS NULL AND status = 'completed';

ALTER TABLE messages ALTER COLUMN branch_id SET NOT NULL;
ALTER TABLE messages ALTER COLUMN sequence SET NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS messages_branch_sequence_uidx
  ON messages (conversation_id, branch_id, sequence);
CREATE INDEX IF NOT EXISTS messages_branch_parent_idx
  ON messages (conversation_id, branch_id, parent_message_id);

CREATE TABLE IF NOT EXISTS message_parts (
  id UUID PRIMARY KEY,
  message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
  ordinal INTEGER NOT NULL,
  kind TEXT NOT NULL,
  text_content TEXT,
  object_key TEXT,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (message_id, ordinal)
);
INSERT INTO message_parts (id, message_id, ordinal, kind, text_content, created_at)
SELECT id, id, 0, 'TEXT', body, created_at FROM messages
ON CONFLICT (id) DO NOTHING;

ALTER TABLE message_runs ADD COLUMN IF NOT EXISTS branch_id UUID;
ALTER TABLE message_runs ADD COLUMN IF NOT EXISTS idempotency_key TEXT;
ALTER TABLE message_runs ADD COLUMN IF NOT EXISTS request_hash TEXT;
ALTER TABLE message_runs ADD COLUMN IF NOT EXISTS context_build_id UUID;
ALTER TABLE message_runs ADD COLUMN IF NOT EXISTS provider TEXT NOT NULL DEFAULT 'upstream';
ALTER TABLE message_runs ADD COLUMN IF NOT EXISTS started_at TIMESTAMPTZ;
ALTER TABLE message_runs ADD COLUMN IF NOT EXISTS completed_at TIMESTAMPTZ;
ALTER TABLE message_runs ADD COLUMN IF NOT EXISTS latency_ms BIGINT NOT NULL DEFAULT 0;
ALTER TABLE message_runs ADD COLUMN IF NOT EXISTS provider_continuation_used BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE message_runs ADD COLUMN IF NOT EXISTS provider_continuation_fallback BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE message_runs ADD COLUMN IF NOT EXISTS instance_id TEXT;
UPDATE message_runs SET branch_id = conversation_id WHERE branch_id IS NULL;
UPDATE message_runs SET started_at = created_at WHERE started_at IS NULL;
ALTER TABLE message_runs ALTER COLUMN branch_id SET NOT NULL;
ALTER TABLE message_runs ALTER COLUMN started_at SET NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS message_runs_user_idempotency_uidx
  ON message_runs (user_id, idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS message_runs_active_branch_uidx
  ON message_runs (conversation_id, branch_id)
  WHERE status IN ('queued', 'running', 'streaming');

ALTER TABLE run_events ADD COLUMN IF NOT EXISTS sequence BIGINT;
ALTER TABLE run_events ADD COLUMN IF NOT EXISTS payload JSONB NOT NULL DEFAULT '{}'::jsonb;
WITH ordered AS (
  SELECT id, ROW_NUMBER() OVER (PARTITION BY run_id ORDER BY id) AS position
  FROM run_events WHERE sequence IS NULL
)
UPDATE run_events SET sequence = ordered.position
FROM ordered WHERE run_events.id = ordered.id;
ALTER TABLE run_events ALTER COLUMN sequence SET NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS run_events_run_sequence_uidx ON run_events (run_id, sequence);

CREATE TABLE IF NOT EXISTS conversation_summaries (
  id UUID PRIMARY KEY,
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  branch_id UUID NOT NULL REFERENCES conversation_branches(id) ON DELETE CASCADE,
  source_first_sequence BIGINT NOT NULL,
  source_last_sequence BIGINT NOT NULL,
  source_first_message_id UUID NOT NULL REFERENCES messages(id),
  source_last_message_id UUID NOT NULL REFERENCES messages(id),
  structured_state JSONB NOT NULL DEFAULT '{}'::jsonb,
  summary_text TEXT NOT NULL,
  summary_model TEXT NOT NULL,
  summary_version TEXT NOT NULL,
  verified_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (conversation_id, branch_id, source_last_sequence, summary_version)
);

CREATE TABLE IF NOT EXISTS jobs (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL,
  job_type TEXT NOT NULL,
  resource_type TEXT NOT NULL,
  resource_id UUID,
  idempotency_key TEXT NOT NULL,
  request_hash TEXT NOT NULL,
  status TEXT NOT NULL,
  attempts INTEGER NOT NULL DEFAULT 0,
  max_attempts INTEGER NOT NULL DEFAULT 3,
  available_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  lease_owner TEXT,
  lease_expires_at TIMESTAMPTZ,
  last_error_code TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  completed_at TIMESTAMPTZ,
  UNIQUE (user_id, job_type, idempotency_key),
  CHECK (status IN ('queued', 'running', 'success', 'error', 'cancelled')),
  CHECK (attempts >= 0 AND max_attempts > 0)
);
CREATE INDEX IF NOT EXISTS jobs_claim_idx
  ON jobs (job_type, status, available_at, created_at);

CREATE TABLE IF NOT EXISTS context_compactions (
  id UUID PRIMARY KEY,
  job_id UUID NOT NULL UNIQUE REFERENCES jobs(id) ON DELETE CASCADE,
  user_id UUID NOT NULL,
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  branch_id UUID NOT NULL REFERENCES conversation_branches(id) ON DELETE CASCADE,
  idempotency_key TEXT NOT NULL,
  source_first_sequence BIGINT NOT NULL,
  source_last_sequence BIGINT NOT NULL,
  source_first_message_id UUID NOT NULL REFERENCES messages(id),
  source_last_message_id UUID NOT NULL REFERENCES messages(id),
  summary_id UUID REFERENCES conversation_summaries(id),
  summary_version TEXT NOT NULL,
  status TEXT NOT NULL,
  error_code TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  UNIQUE (user_id, conversation_id, idempotency_key),
  CHECK (status IN ('queued', 'running', 'success', 'error')),
  CHECK (source_first_sequence > 0 AND source_last_sequence >= source_first_sequence)
);
CREATE INDEX IF NOT EXISTS context_compactions_conversation_idx
  ON context_compactions (conversation_id, branch_id, created_at DESC);

CREATE TABLE IF NOT EXISTS model_capabilities (
  model_id TEXT PRIMARY KEY,
  context_limit_tokens INTEGER NOT NULL,
  output_reserve_tokens INTEGER NOT NULL,
  reasoning_reserve_tokens INTEGER NOT NULL,
  tool_reserve_tokens INTEGER NOT NULL,
  safety_margin_tokens INTEGER NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
INSERT INTO model_capabilities
  (model_id,context_limit_tokens,output_reserve_tokens,reasoning_reserve_tokens,tool_reserve_tokens,safety_margin_tokens)
VALUES
  ('ylven-default',128000,12000,20000,6000,6000),
  ('gpt-5.6',400000,16000,32000,8000,8000),
  ('gpt-5.5',400000,16000,32000,8000,8000),
  ('claude-opus',200000,16000,24000,8000,8000),
  ('grok-4',131072,12000,20000,6000,6000)
ON CONFLICT (model_id) DO NOTHING;

CREATE TABLE IF NOT EXISTS context_builds (
  id UUID PRIMARY KEY,
  run_id UUID NOT NULL REFERENCES message_runs(id) ON DELETE CASCADE,
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  branch_id UUID NOT NULL REFERENCES conversation_branches(id) ON DELETE CASCADE,
  model TEXT NOT NULL,
  context_limit_tokens INTEGER NOT NULL,
  input_budget_tokens INTEGER NOT NULL,
  estimated_input_tokens INTEGER NOT NULL,
  output_reserve_tokens INTEGER NOT NULL,
  reasoning_reserve_tokens INTEGER NOT NULL,
  tool_reserve_tokens INTEGER NOT NULL,
  safety_margin_tokens INTEGER NOT NULL,
  compaction_mode TEXT NOT NULL,
  continuation_mode TEXT NOT NULL,
  context_hash TEXT NOT NULL,
  compiler_version TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS context_builds_run_created_idx ON context_builds (run_id, created_at DESC);

CREATE TABLE IF NOT EXISTS context_build_items (
  id BIGSERIAL PRIMARY KEY,
  context_build_id UUID NOT NULL REFERENCES context_builds(id) ON DELETE CASCADE,
  ordinal INTEGER NOT NULL,
  item_type TEXT NOT NULL,
  source_id TEXT,
  role TEXT,
  content_redacted TEXT,
  estimated_tokens INTEGER NOT NULL DEFAULT 0,
  included BOOLEAN NOT NULL,
  exclusion_reason TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (context_build_id, ordinal)
);

CREATE TABLE IF NOT EXISTS provider_conversation_states (
  id UUID PRIMARY KEY,
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  branch_id UUID NOT NULL REFERENCES conversation_branches(id) ON DELETE CASCADE,
  provider TEXT NOT NULL,
  model TEXT NOT NULL,
  continuation_id TEXT NOT NULL,
  status TEXT NOT NULL,
  expires_at TIMESTAMPTZ,
  last_failure_code TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (conversation_id, branch_id, provider, model)
);

ALTER TABLE idempotency_records ADD COLUMN IF NOT EXISTS user_id UUID;
ALTER TABLE idempotency_records ADD COLUMN IF NOT EXISTS operation TEXT;
ALTER TABLE idempotency_records ADD COLUMN IF NOT EXISTS resource_type TEXT;
ALTER TABLE idempotency_records ADD COLUMN IF NOT EXISTS resource_id UUID;
ALTER TABLE idempotency_records ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idempotency_records_resource_idx
  ON idempotency_records (user_id, operation, resource_id);

CREATE TABLE IF NOT EXISTS conversation_locks (
  conversation_id UUID PRIMARY KEY REFERENCES conversations(id) ON DELETE CASCADE,
  holder_run_id UUID REFERENCES message_runs(id) ON DELETE SET NULL,
  lease_owner TEXT,
  lease_expires_at TIMESTAMPTZ,
  fencing_token BIGINT NOT NULL DEFAULT 0,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS conversation_title_jobs (
  id UUID PRIMARY KEY,
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  source_message_id UUID NOT NULL REFERENCES messages(id),
  status TEXT NOT NULL,
  attempts INTEGER NOT NULL DEFAULT 0,
  available_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  lease_owner TEXT,
  lease_expires_at TIMESTAMPTZ,
  last_error_code TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (conversation_id, source_message_id)
);

CREATE TABLE IF NOT EXISTS message_feedback (
  message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
  user_id UUID NOT NULL,
  value TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (message_id, user_id)
);

CREATE TABLE IF NOT EXISTS speech_jobs (
  id UUID PRIMARY KEY,
  message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
  user_id UUID NOT NULL,
  status TEXT NOT NULL,
  provider TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'conversations_title_source_check') THEN
    ALTER TABLE conversations ADD CONSTRAINT conversations_title_source_check
      CHECK (title_source IN ('AUTO_TEMP', 'AUTO_FINAL', 'USER'));
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'messages_role_check') THEN
    ALTER TABLE messages ADD CONSTRAINT messages_role_check
      CHECK (role IN ('system', 'user', 'assistant', 'tool'));
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'message_parts_kind_check') THEN
    ALTER TABLE message_parts ADD CONSTRAINT message_parts_kind_check
      CHECK (kind IN ('TEXT', 'IMAGE', 'FILE', 'AUDIO', 'TOOL_CALL', 'TOOL_RESULT', 'CITATION', 'ARTIFACT'));
  END IF;
END $$;
