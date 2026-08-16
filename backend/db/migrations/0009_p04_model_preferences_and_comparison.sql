-- P04-W02..W05: user/session model choices, comparisons and operations.
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS default_model_id TEXT NOT NULL DEFAULT 'ylven-default';
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS default_reasoning_profile TEXT NOT NULL DEFAULT 'auto';

CREATE TABLE IF NOT EXISTS p04_records (
  record_key TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  record_type TEXT NOT NULL,
  value JSONB NOT NULL,
  version BIGINT NOT NULL DEFAULT 1,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (BTRIM(record_key) <> ''),
  CHECK (BTRIM(user_id) <> ''),
  CHECK (BTRIM(record_type) <> ''),
  CHECK (jsonb_typeof(value) = 'object')
);
CREATE INDEX IF NOT EXISTS p04_records_user_type_idx ON p04_records(user_id,record_type,updated_at DESC);

CREATE TABLE IF NOT EXISTS comparison_groups (
  id UUID PRIMARY KEY,
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  user_id UUID NOT NULL,
  prompt TEXT NOT NULL,
  status TEXT NOT NULL,
  candidates JSONB NOT NULL DEFAULT '[]'::jsonb,
  adopted_run_id TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (BTRIM(prompt) <> ''),
  CHECK (jsonb_typeof(candidates) = 'array')
);
CREATE INDEX IF NOT EXISTS comparison_groups_user_created_idx ON comparison_groups(user_id,created_at DESC);

CREATE TABLE IF NOT EXISTS provider_channels (
  id TEXT PRIMARY KEY, provider_id TEXT NOT NULL, name TEXT NOT NULL,
  credential_reference TEXT NOT NULL, endpoint TEXT NOT NULL DEFAULT '',
  enabled BOOLEAN NOT NULL DEFAULT TRUE, priority INTEGER NOT NULL DEFAULT 0,
  version BIGINT NOT NULL DEFAULT 1, updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (BTRIM(credential_reference) <> '')
);
CREATE TABLE IF NOT EXISTS routing_policies (
  id TEXT PRIMARY KEY, model_id TEXT NOT NULL, primary_channel_id TEXT NOT NULL,
  fallback_channel_ids JSONB NOT NULL DEFAULT '[]'::jsonb, max_attempts INTEGER NOT NULL DEFAULT 2,
  enabled BOOLEAN NOT NULL DEFAULT TRUE, version BIGINT NOT NULL DEFAULT 1,
  CHECK (jsonb_typeof(fallback_channel_ids) = 'array')
);
CREATE TABLE IF NOT EXISTS provider_runtime_policies (
  provider_id TEXT PRIMARY KEY, max_concurrency INTEGER NOT NULL DEFAULT 8,
  timeout_seconds INTEGER NOT NULL DEFAULT 90, circuit_threshold INTEGER NOT NULL DEFAULT 3,
  cooldown_seconds INTEGER NOT NULL DEFAULT 60, enabled BOOLEAN NOT NULL DEFAULT TRUE,
  version BIGINT NOT NULL DEFAULT 1
);
CREATE TABLE IF NOT EXISTS model_health_status (
  model_id TEXT PRIMARY KEY, provider_id TEXT NOT NULL, status TEXT NOT NULL,
  latency_ms BIGINT NOT NULL DEFAULT 0, last_probe_at TIMESTAMPTZ NOT NULL,
  capabilities JSONB NOT NULL DEFAULT '[]'::jsonb, error_code TEXT,
  CHECK (status IN ('available','degraded','unavailable')),
  CHECK (jsonb_typeof(capabilities) = 'array')
);
CREATE TABLE IF NOT EXISTS ai_usage_events (
  id UUID PRIMARY KEY, user_id UUID NOT NULL, run_id UUID, model_id TEXT NOT NULL,
  input_tokens INTEGER NOT NULL DEFAULT 0, output_tokens INTEGER NOT NULL DEFAULT 0,
  reasoning_tokens INTEGER NOT NULL DEFAULT 0, price_version_id TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS model_price_snapshots (
  id TEXT PRIMARY KEY, model_id TEXT NOT NULL, input_per_million NUMERIC NOT NULL,
  output_per_million NUMERIC NOT NULL, reasoning_per_million NUMERIC NOT NULL,
  effective_at TIMESTAMPTZ NOT NULL, version BIGINT NOT NULL DEFAULT 1
);
