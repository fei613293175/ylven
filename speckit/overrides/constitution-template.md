# YLVEN Project Constitution

## 1. Product and Scope Stability

YLVEN is an Android-first multi-model AI work platform with its own backend, admin console, developer portal, database, files and billing. The primary navigation, identity flow, domain ownership, public domains and release contract cannot change silently.

## 2. Data Authority and Provider Isolation

YLVEN is authoritative for identity, conversations, projects, files, artifacts, orders, ledger and entitlements. Sub2API and official providers are adapters/execution channels. Provider IDs and credentials are never product primary keys or Android secrets.

## 3. Vertical Feature Completeness

A Feature ID is complete only when required UI states, API/event, persistence, admin control, authorization, observability and tests are implemented. Static shells, permanent mock success and fake reports are prohibited.

## 4. Security and Financial Correctness

All secrets stay server-side. Registration/login challenges are server verified and one-time. Money, credit, jobs, webhooks and retryable writes are idempotent and auditable. Ledger history is immutable; balance is a projection.

## 5. Performance and Operability

Services are stateless where practical, streaming and queues are isolated, model/provider limits have bulkheads and circuit breakers, and configuration is dynamic. Avoid premature microservices; extract only measured bottlenecks.

## 6. Tests and Release Evidence

Use unit, integration, contract, Android UI and end-to-end tests appropriate to risk. Every phase produces a real Gradle APK and machine-verified release package. Chat claims do not constitute completion.

## 7. Spec Kit Lite Discipline

One Constitution, one active phase, one phase spec/plan/tasks. Optional clarify/analyze/checklist/converge are used only when risk justifies them. Planning must not repeatedly displace implementation.

**Version**: 1.2.0 | **Ratified**: 2026-08-04 | **Last Amended**: 2026-08-04
