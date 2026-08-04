# Implementation Plan: [PHASE ID] [PHASE TITLE]

## 1. Context and Constraints

Reference the phase specification, Constitution, Feature Map, fixed technology stack and existing code. Record constraints; do not reopen fixed decisions.

## 2. Vertical Architecture Slices

Organize implementation by user-visible vertical slices. Each slice must include Feature IDs, Android/web UI, API/events, domain logic, migrations, admin controls, metrics/logs and tests.

## 3. Android Plan

List screens, routes, ViewModels/reducers, UI states, Room/cache behavior, network/SSE behavior, accessibility, small-screen and theme considerations. Model/config values come from backend, not hard-coded APK data.

## 4. Backend and API Plan

List modules, endpoints, schemas, permissions, idempotency, rate limits, provider adapters, streaming, retry and error mapping. Define concrete OpenAPI changes.

## 5. Data and Migration Plan

List tables/entities, keys, ownership, uniqueness, indexes, transactions, outbox/events, retention, backfill and rollback. Financial changes require ledger-safe migration.

## 6. Admin and Developer Controls

List menus, filters, actions, configuration versions, audit logs, approvals, diagnostics and rollback. An admin page cannot be only a read-only table if operations need control.

## 7. Jobs and External Integrations

Define queue, payload version, idempotency, progress, cancellation, retry/dead-letter, credential dependency and contract mocks. External secrets become Owner Actions, not permanent blockers.

## 8. Performance, Security and Observability

Set relevant load assumptions, timeouts, bulkheads, circuit breakers, cache rules, PII/secret handling, metrics, traces, logs and alerts.

## 9. Test Plan

Map Feature IDs to unit, integration, contract, Android UI, E2E and small real-upstream verification. Record exact commands and required environments.

## 10. Deployment and Rollback

List environment variables, migrations, feature flags, deployment order, health checks, rollback and data compatibility. Finish with release script and owner acceptance.

## UI 实施计划
引用设计系统版本、公共组件、布局 profile、视觉门禁、截图与像素差异验收。
