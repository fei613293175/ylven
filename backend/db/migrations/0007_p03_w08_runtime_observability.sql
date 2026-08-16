-- P03-W08: shared runtime health, context diagnostics and distributed test evidence.
CREATE TABLE IF NOT EXISTS service_instances (
  instance_id TEXT PRIMARY KEY,
  service_name TEXT NOT NULL,
  version TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  expires_at TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '2 minutes'),
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  CHECK (status IN ('healthy', 'degraded', 'error'))
);
CREATE INDEX IF NOT EXISTS service_instances_liveness_idx
  ON service_instances (service_name, last_seen_at DESC);

CREATE TABLE IF NOT EXISTS health_checks (
  id UUID PRIMARY KEY,
  instance_id TEXT NOT NULL REFERENCES service_instances(instance_id) ON DELETE CASCADE,
  check_name TEXT NOT NULL,
  status TEXT NOT NULL,
  detail TEXT NOT NULL DEFAULT '',
  checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (status IN ('healthy', 'degraded', 'error'))
);
CREATE INDEX IF NOT EXISTS health_checks_instance_checked_idx
  ON health_checks (instance_id, checked_at DESC);

CREATE TABLE IF NOT EXISTS test_runs (
  id UUID PRIMARY KEY,
  test_key TEXT NOT NULL,
  status TEXT NOT NULL,
  instance_a TEXT NOT NULL DEFAULT '',
  instance_b TEXT NOT NULL DEFAULT '',
  cursor_before BIGINT NOT NULL DEFAULT 0,
  cursor_after BIGINT NOT NULL DEFAULT 0,
  error_code TEXT NOT NULL DEFAULT '',
  details JSONB NOT NULL DEFAULT '{}'::jsonb,
  started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  completed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (status IN ('running', 'pass', 'fail')),
  CHECK (cursor_before >= 0 AND cursor_after >= 0)
);
CREATE INDEX IF NOT EXISTS test_runs_key_created_idx
  ON test_runs (test_key, created_at DESC);
