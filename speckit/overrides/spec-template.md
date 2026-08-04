# Phase Specification: [PHASE ID] [PHASE TITLE]

**Source of truth**: `CURRENT_PHASE.yaml`, one matching `phases/[PHASE]_*.md`, `contracts/feature-map.yaml`  
**Status**: Draft  
**Created**: [DATE]

## 1. Objective and User Value

Describe the concrete user/operator outcome of this phase. Do not restate the whole YLVEN product and do not introduce later-phase scope.

## 2. Included Feature IDs

List every Feature ID in this phase. The list must exactly match the phase document. For each group, explain the end-to-end user or operator flow.

## 3. User Scenarios and Acceptance

For each priority scenario include: actor, preconditions, numbered actions, visible states, backend effects, success result, recoverable failures, permissions and test evidence.

## 4. Functional Requirements

Use numbered, testable requirements. Cover Android/admin/developer surfaces, API/event behavior, persistence, configuration, auditing, billing and asynchronous jobs where applicable.

## 5. Non-Functional Requirements

Specify security, privacy, concurrency, latency, availability, idempotency, observability, accessibility, localization, data retention and rollback requirements relevant to this phase.

## 6. Data and Ownership

List entities, ownership boundaries, lifecycle, indexes/uniqueness, deletion/retention and provider references. Do not make Sub2API/provider IDs authoritative.

## 7. Error and Recovery Matrix

Map normal, empty, loading, validation, timeout, rate-limit, permission, cancellation, partial success and external dependency failures to stable error codes and recovery actions.

## 8. Out of Scope

Explicitly identify later-phase or rejected work so Codex does not expand scope while implementing.

## 9. Completion Evidence

State exact automated tests, staging checks, admin visibility, release artifacts and owner acceptance required. “Implemented” without paths/logs is invalid.

## UI 页面与状态合同
列出 Page ID、Feature ID、State ID、效果图路径、功能与样例内容边界。
