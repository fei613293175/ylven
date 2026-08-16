-- P04-W02..W05: durable concurrency control for AI settings and operator APIs.
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS ai_settings_version BIGINT NOT NULL DEFAULT 1;
ALTER TABLE comparison_groups ADD COLUMN IF NOT EXISTS synthesis_run_id TEXT;
ALTER TABLE routing_policies ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE provider_runtime_policies ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE INDEX IF NOT EXISTS provider_channels_provider_priority_idx
  ON provider_channels(provider_id, enabled DESC, priority ASC, id);
CREATE INDEX IF NOT EXISTS routing_policies_model_idx ON routing_policies(model_id);
CREATE INDEX IF NOT EXISTS ai_usage_events_run_idx ON ai_usage_events(run_id);
CREATE UNIQUE INDEX IF NOT EXISTS ai_usage_events_run_uidx
  ON ai_usage_events(run_id) WHERE run_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS model_price_snapshots_model_effective_idx
  ON model_price_snapshots(model_id, effective_at DESC);
