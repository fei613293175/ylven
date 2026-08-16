-- P04-W01: provider-owned model catalog, verified capabilities and reasoning mappings.
CREATE TABLE IF NOT EXISTS model_providers (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  sort_order INTEGER NOT NULL DEFAULT 0,
  version BIGINT NOT NULL DEFAULT 1,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (id ~ '^[a-z0-9][a-z0-9._-]{0,63}$'),
  CHECK (BTRIM(name) <> '')
);

INSERT INTO model_providers (id,name,enabled,sort_order)
VALUES ('ylven','YLVEN',TRUE,0)
ON CONFLICT (id) DO NOTHING;

ALTER TABLE models ADD COLUMN IF NOT EXISTS provider_id TEXT REFERENCES model_providers(id);
ALTER TABLE models ADD COLUMN IF NOT EXISTS upstream_model TEXT;
ALTER TABLE models ADD COLUMN IF NOT EXISTS purpose TEXT NOT NULL DEFAULT '';
ALTER TABLE models ADD COLUMN IF NOT EXISTS speed_tier TEXT NOT NULL DEFAULT 'balanced';
ALTER TABLE models ADD COLUMN IF NOT EXISTS sort_order INTEGER NOT NULL DEFAULT 0;
ALTER TABLE models ADD COLUMN IF NOT EXISTS version BIGINT NOT NULL DEFAULT 1;
UPDATE models SET provider_id='ylven' WHERE provider_id IS NULL;
UPDATE models SET upstream_model=id WHERE upstream_model IS NULL OR BTRIM(upstream_model)='';
ALTER TABLE models ALTER COLUMN provider_id SET NOT NULL;
ALTER TABLE models ALTER COLUMN upstream_model SET NOT NULL;
ALTER TABLE models DROP CONSTRAINT IF EXISTS models_speed_tier_check;
ALTER TABLE models ADD CONSTRAINT models_speed_tier_check CHECK (speed_tier IN ('fast','balanced','deliberate'));
CREATE INDEX IF NOT EXISTS models_provider_sort_idx ON models(provider_id,sort_order,id);

CREATE TABLE IF NOT EXISTS model_capability_definitions (
  id TEXT PRIMARY KEY,
  label TEXT NOT NULL,
  sort_order INTEGER NOT NULL DEFAULT 0,
  CHECK (id ~ '^[a-z0-9][a-z0-9._-]{0,63}$'),
  CHECK (BTRIM(label) <> '')
);

INSERT INTO model_capability_definitions (id,label,sort_order) VALUES
  ('text','文本',0),
  ('vision','视觉',10),
  ('tools','工具调用',20),
  ('reasoning','推理',30)
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS model_capability_probe_results (
  id UUID PRIMARY KEY,
  model_id TEXT NOT NULL REFERENCES models(id) ON DELETE CASCADE,
  capability_id TEXT NOT NULL REFERENCES model_capability_definitions(id),
  status TEXT NOT NULL,
  probe_source TEXT NOT NULL,
  evidence_reference TEXT NOT NULL,
  probed_at TIMESTAMPTZ NOT NULL,
  expires_at TIMESTAMPTZ,
  recorded_by TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (status IN ('passed','failed')),
  CHECK (BTRIM(probe_source) <> ''),
  CHECK (BTRIM(evidence_reference) <> ''),
  CHECK (BTRIM(recorded_by) <> '')
);
CREATE INDEX IF NOT EXISTS model_capability_probe_latest_idx
  ON model_capability_probe_results(model_id,capability_id,probed_at DESC,created_at DESC);

ALTER TABLE reasoning_profiles ADD COLUMN IF NOT EXISTS label TEXT NOT NULL DEFAULT '';
ALTER TABLE reasoning_profiles ADD COLUMN IF NOT EXISTS enabled BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE reasoning_profiles ADD COLUMN IF NOT EXISTS upstream_parameters JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE reasoning_profiles ADD COLUMN IF NOT EXISTS version BIGINT NOT NULL DEFAULT 1;
ALTER TABLE reasoning_profiles ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
UPDATE reasoning_profiles SET label=CASE profile_id
  WHEN 'auto' THEN '自动' WHEN 'quick' THEN '快速' WHEN 'standard' THEN '标准' WHEN 'deep' THEN '深度' ELSE profile_id END
WHERE BTRIM(label)='';
ALTER TABLE reasoning_profiles DROP CONSTRAINT IF EXISTS reasoning_profiles_upstream_parameters_object_check;
ALTER TABLE reasoning_profiles ADD CONSTRAINT reasoning_profiles_upstream_parameters_object_check
  CHECK (jsonb_typeof(upstream_parameters)='object');

ALTER TABLE message_runs ADD COLUMN IF NOT EXISTS reasoning_profile TEXT NOT NULL DEFAULT 'auto';
ALTER TABLE message_runs ADD COLUMN IF NOT EXISTS reasoning_parameters JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE message_runs ADD COLUMN IF NOT EXISTS provider_model TEXT NOT NULL DEFAULT '';
UPDATE message_runs SET provider_model=model WHERE BTRIM(provider_model)='';
ALTER TABLE message_runs DROP CONSTRAINT IF EXISTS message_runs_reasoning_parameters_object_check;
ALTER TABLE message_runs ADD CONSTRAINT message_runs_reasoning_parameters_object_check
  CHECK (jsonb_typeof(reasoning_parameters)='object');

CREATE TABLE IF NOT EXISTS model_catalog_audit_events (
  id UUID PRIMARY KEY,
  actor_id TEXT NOT NULL,
  action TEXT NOT NULL,
  resource_type TEXT NOT NULL,
  resource_id TEXT NOT NULL,
  before_value JSONB,
  after_value JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (BTRIM(actor_id) <> ''),
  CHECK (BTRIM(action) <> ''),
  CHECK (BTRIM(resource_type) <> ''),
  CHECK (BTRIM(resource_id) <> '')
);
CREATE INDEX IF NOT EXISTS model_catalog_audit_created_idx ON model_catalog_audit_events(created_at DESC);
