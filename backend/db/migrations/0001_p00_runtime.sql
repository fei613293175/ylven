-- P00-011: operational metadata. Version and checksum are immutable.
CREATE TABLE IF NOT EXISTS schema_migrations (
  version BIGINT PRIMARY KEY,
  name TEXT NOT NULL,
  checksum TEXT NOT NULL,
  applied_at TIMESTAMPTZ NOT NULL,
  result TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS idempotency_records (
  namespace TEXT NOT NULL,
  idempotency_key TEXT NOT NULL,
  request_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (namespace, idempotency_key)
);
