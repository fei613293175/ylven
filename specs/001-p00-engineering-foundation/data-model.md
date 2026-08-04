# P00 Data Model

P00 establishes operational metadata and adapter boundaries, not end-user identity, conversation or commerce data. Canonical field definitions remain in `contracts/database-entity-catalog.yaml` and migrations.

## Repository and configuration

- **repository_modules**: stable module ID, path, runtime, owner, public boundary and lifecycle state. Module ID and path are unique.
- **system_configs**: environment, key, public value or Secret reference, schema version, revision, updated time and actor. `(environment, key, revision)` is unique; changes are append-audited.
- **feature_flags**: stable flag key, environment, enabled state, revision and audit metadata.
- **design_tokens**: design system version, token key/value and immutable checksum.
- **app_builds**: application ID, version name/code, commit, CI run, Artifact ID, APK hash and build state.

## Runtime and jobs

- **service_instances**: process type, instance ID, version, commit, started time, heartbeat, health/readiness and dependency summary.
- **jobs**: job ID, type, idempotency key, payload reference, state, attempt count, next attempt and terminal error. `(type, idempotency_key)` is unique.
- **job_events**: append-only job transition records linked to a job ID.
- **scheduled_jobs**: schedule ID, task type, schedule expression, enabled state and last/next execution.
- **distributed_locks**: lock key, holder, fencing token and expiry. Fencing token is monotonic.

## Infrastructure and contracts

- **schema_migrations**: version, name, checksum, applied time and result; version/checksum cannot be silently rewritten.
- **cache_namespaces**: namespace, purpose, TTL policy and limit policy.
- **event_streams**: stream/subject, retention, duplicate window, consumer and acknowledgement policy.
- **storage_buckets**: logical bucket ID, adapter, public/private classification, region reference and lifecycle policy; credentials are never fields.
- **api_contracts**: API version, OpenAPI checksum, generated client checksum and publication state.
- **feature_contracts**: Feature ID and references to API/data/page/test/evidence records.

## Operations, release and providers

- **audit_logs**: append-only actor, action, resource type/ID, result, trace ID and redacted details.
- **service_metrics**: metric identity and label policy; time-series samples remain in the observability backend.
- **traces**: trace/span identifiers and redacted linkage metadata.
- **ci_runs**: run ID, workflow, commit, status and Artifact references.
- **release_manifests**: phase/version/commit/run/artifact/APK hash and evidence file checksums.
- **provider_channels**: opaque channel ID, adapter type, enabled state and Secret reference only.
- **capability_probe_runs**: probe ID, channel ID, capability set, start/end time and aggregate status.
- **capability_probe_results**: probe ID, capability, status, latency, redacted error and evidence reference.
- **model_capabilities**: channel capability projection derived from successful probes, with observed time and expiry.
- **secret_references**: logical key, environment, provider, existence/rotation status and last verification time; never Secret value.
- **admin_routes / developer_routes**: route ID, permission, feature binding, visibility and revision.

## State transitions

- Job: `QUEUED -> RUNNING -> SUCCEEDED|FAILED|CANCELLED`; retryable failure returns to `QUEUED` only below the fixed attempt limit.
- Probe: `QUEUED -> RUNNING -> SUCCEEDED|PARTIAL|FAILED|NOT_CONFIGURED`.
- CI run: `QUEUED -> RUNNING -> SUCCEEDED|FAILED|CANCELLED`.
- Release manifest: `DRAFT -> CI_VERIFIED -> DELIVERED -> OWNER_APPROVED -> CLOSED`; P00 stops at `DELIVERED` until owner approval.
