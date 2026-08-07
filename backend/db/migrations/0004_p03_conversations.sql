-- P03-W01: owned conversations and searchable message bodies.
CREATE TABLE IF NOT EXISTS conversations (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL,
  title TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  archived_at TIMESTAMPTZ,
  deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS conversations_user_updated_idx ON conversations (user_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS conversations_user_status_idx ON conversations (user_id, status);

CREATE TABLE IF NOT EXISTS messages (
  id UUID PRIMARY KEY,
  conversation_id UUID NOT NULL REFERENCES conversations(id),
  user_id UUID NOT NULL,
  role TEXT NOT NULL,
  body TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS messages_conversation_created_idx ON messages (conversation_id, created_at DESC);

CREATE TABLE IF NOT EXISTS message_runs (
  id UUID PRIMARY KEY,
  conversation_id UUID NOT NULL REFERENCES conversations(id),
  user_id UUID NOT NULL,
  user_message_id UUID REFERENCES messages(id),
  assistant_message_id UUID REFERENCES messages(id),
  model TEXT NOT NULL,
  status TEXT NOT NULL,
  cursor BIGINT NOT NULL DEFAULT 0,
  error_code TEXT,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE IF NOT EXISTS run_events (
  id BIGSERIAL PRIMARY KEY,
  run_id UUID NOT NULL REFERENCES message_runs(id),
  event_type TEXT NOT NULL,
  delta TEXT,
  created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS run_events_run_id_idx ON run_events (run_id, id);
CREATE TABLE IF NOT EXISTS export_jobs (
  id UUID PRIMARY KEY,
  conversation_id UUID NOT NULL REFERENCES conversations(id),
  message_id UUID REFERENCES messages(id),
  user_id UUID NOT NULL,
  format TEXT NOT NULL,
  status TEXT NOT NULL,
  content TEXT,
  created_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE IF NOT EXISTS conversation_drafts (
  conversation_id UUID PRIMARY KEY REFERENCES conversations(id),
  user_id UUID NOT NULL,
  body TEXT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);
