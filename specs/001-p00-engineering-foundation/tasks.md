# Tasks: P00 Engineering Foundation and Capability Proof

## Rules

- Every task includes Feature IDs, concrete paths, tests, and completion evidence.
- Execute only the Work Packet named by `CURRENT_WORK_PACKET.yaml`; reread that packet immediately before implementation.
- A packet is not complete until its implementation and tests form a clean commit and `./ylven.ps1 close-packet` succeeds.
- UI work must use the packet Page/State IDs and approved mockups. Real screenshots and visual diffs are release evidence, not optional documentation.
- External credentials may produce an explicit `BLOCKED_EXTERNAL` result; they must never produce fabricated success evidence.
- P00 release stops at owner acceptance. Do not run `close-release` and do not start P01.

## Phase 0: Contract and Baseline Verification

- [x] T001 `[P00-001..P00-024]` Restore the controller-selected phase and packet with `./ylven.ps1 resume`; verify `CURRENT_PHASE.yaml`, `CURRENT_WORK_PACKET.yaml`, `status/WORK_PACKET_STATUS.yaml`, and the selected file under `work-packets/`.
- [x] T002 `[P00-001..P00-024]` Validate planning contracts and approved visual inputs with `scripts/19_VALIDATE_UI_CONTRACTS.py`, `scripts/20_CHECK_UI_VISUAL_GATE.py`, `scripts/26_VALIDATE_GENERATED_MOCKUPS.py`, and `scripts/27_AUDIT_VISUAL_DUPLICATES.py`.
- [x] T003 `[P00-001..P00-024]` Establish the single phase specification, plan, research, data model, contracts, quickstart, checklist, and this task list under `specs/001-p00-engineering-foundation/`.

## Work Packet P00-W01: Repository, Configuration, and Android Baseline

- [x] T004 `[P00-001]` Define and test the repository/module ownership boundary in `repository/modules.yaml`, `repository/README.md`, and `tests/p00_w01_contract_test.py`.
  - Admin states: `YL-M-001-S01_LOADING` through `YL-M-001-S08_SERVER_ERROR` remain contract-bound future UI states; this packet supplies their real engineering data source, not a duplicate admin shell.
  - Tests: `./ylven.ps1 py -Script tests/p00_w01_contract_test.py`.
  - Evidence: schema-backed module inventory with unique paths and owners.
- [x] T005 `[P00-002]` Implement layered public configuration and secret references in `config/base.yaml`, `config/environments/*.yaml`, `config/schema.yaml`, and `config/README.md`.
  - Admin states: `YL-M-002-S01_LOADING` through `YL-M-002-S11_SERVER_ERROR` remain contract-bound future UI states.
  - Tests: validate base/local/staging/production overlays, required keys, and absence of plaintext credentials in `tests/p00_w01_contract_test.py`.
  - Evidence: deterministic merged configuration contract and public-repository scan.
- [x] T006 `[P00-003]` Implement the buildable Compose application shell in root Gradle files and `android/app/`, including stable test tags and process recreation support.
  - Android states: `YL-A-001-S01_DEFAULT`; admin contract states `YL-M-003-S01_LOADING` through `YL-M-003-S08_SERVER_ERROR` are not implemented as an admin product in this packet.
  - Tests: `android/app/src/test/.../AppStateTest.kt`, `android/app/src/androidTest/.../AppShellTest.kt`, and CI Android build/emulator tests.
  - Evidence: exact CI build and emulator output for the committed revision.
- [x] T007 `[P00-004]` Implement `YL-DS-1.2.0` semantic light/dark tokens in `android/app/src/main/java/cc/orbexa/ylven/ui/theme/` without page-local visual constants.
  - Android/design states: `YL-A-002-S01_DEFAULT`, `YL-DS-001-S01_DEFAULT` through `YL-DS-013-S01_DEFAULT`.
  - Tests: token contract assertions plus light/dark screenshot cases in Android CI.
  - Evidence: token values match `contracts/ui-design-tokens.yaml`; screenshots are indexed by State ID.
- [x] T008 `[P00-005]` Implement four-tab Home/Work/Discover/Profile navigation and saved selection in `android/app/src/main/java/cc/orbexa/ylven/ui/YlvenApp.kt`.
  - Android state: `YL-A-003-S01_DEFAULT`; test interactions are limited to registered tab navigation.
  - Tests: state unit tests and Compose click/recreation tests for all four tabs.
  - Evidence: CI emulator screenshots and `UI_SCREENSHOT_INDEX.csv` entries for the packet's applicable Android states.
- [x] T009 `[P00-001..P00-005]` Finalize W01 feature evidence in `status/P00_FEATURE_STATUS.yaml`, run affected tests, the P00-W01 visual gate, and the public-repository safety gate; commit implementation and tests, ensure a clean worktree, then run `./ylven.ps1 close-packet -Packet P00-W01`.

## Work Packet P00-W02: Runtime Process Shells

- [x] T010 `[P00-006,P00-007,P00-008]` After controller selection and rereading `work-packets/P00-W02.md`, implement Core API, AI Runtime, and Developer Gateway process identities, versioned health/readiness endpoints, and stable error envelopes in `backend/cmd/` and `backend/internal/platform/health/`.
  - API: `/api/health`, `/runtime/health`, `/developer/health` and their readiness counterparts.
  - Tests: process startup, healthy/degraded/unready dependency cases, and response contract tests in `backend/tests/`.
  - Evidence: real command output and request/response fixtures; no static success placeholder.
- [x] T011 `[P00-009]` Implement the Worker job envelope, idempotency, bounded retry, and failure archive in `backend/cmd/worker/` and `backend/internal/platform/jobs/`.
  - Tests: duplicate job, retry exhaustion, poison message, and graceful shutdown.
  - Evidence: zero duplicate business side effects in the recorded test.
- [x] T012 `[P00-010]` Implement the Scheduler process, lease/lock abstraction, and duplicate-run protection in `backend/cmd/scheduler/` and `backend/internal/platform/scheduling/`.
  - Tests: lock contention, lease expiry, clock boundary, and single execution.
  - Evidence: deterministic contention test output.
- [x] T013 `[P00-006..P00-010]` Finalize W02 status/evidence, run process tests and public safety gate, commit cleanly, and close only `P00-W02` through the controller.

## Work Packet P00-W03: Data, Events, Storage, and OpenAPI

- [x] T014 `[P00-011]` After controller selection and rereading `work-packets/P00-W03.md`, implement versioned PostgreSQL migrations, checksums, uniqueness constraints, and rollback verification in `backend/db/migrations/` and `backend/internal/platform/database/`.
- [x] T015 `[P00-012,P00-013]` Implement Redis cache/rate-limit and NATS JetStream publish/consume adapters in `backend/internal/platform/cache/` and `backend/internal/platform/events/`, including timeout/degraded states and duplicate-message handling.
- [x] T016 `[P00-014]` Implement the R2-compatible object-store interface and explicitly selected local filesystem test adapter in `backend/internal/platform/storage/`; never claim external R2 success without real credentials and evidence.
- [x] T017 `[P00-015]` Establish the canonical OpenAPI document and deterministic generation/drift check in `contracts/openapi/`, generated clients, and `scripts/`.
- [x] T018 `[P00-011..P00-015]` Run migration, cache, rate-limit, event, object-store, OpenAPI, and safety tests; finalize evidence, commit cleanly, and close only `P00-W03` through the controller.

## Work Packet P00-W04: Traceability, Observability, CI, Release, and Text Probe

- [ ] T019 `[P00-016]` After controller selection and rereading `work-packets/P00-W04.md`, implement mechanical Feature/API/data/UI/test/evidence traceability checks in `scripts/`, `contracts/`, and `tests/`.
- [ ] T020 `[P00-017]` Implement structured redacted logging, metrics, trace/request correlation, and `/internal/metrics` in `backend/internal/platform/telemetry/` with process integration tests.
- [ ] T021 `[P00-018,P00-019]` Implement exact-revision GitHub Actions gates and exact-Artifact release verification in `.github/workflows/`, `scripts/`, and release evidence templates; include contract, backend, Web, Android, secret, provenance, hash, screenshot, and visual-diff gates.
- [ ] T022 `[P00-020]` Implement Sub2API text/stream capability probing and explicit unconfigured/failure states in `backend/internal/platform/providers/sub2api/` and the admin probe endpoint; record real external evidence only when configuration is genuinely available.
- [ ] T023 `[P00-016..P00-020]` Run traceability, telemetry, CI workflow, release-controller self-tests, probe tests, and public safety gate; finalize evidence, commit cleanly, and close only `P00-W04` through the controller.

## Work Packet P00-W05: Multimodal Probe, Secret Boundary, and Web Shells

- [ ] T024 `[P00-021]` After controller selection and rereading `work-packets/P00-W05.md`, implement Sub2API vision/file/image probes and explicit unconfigured/failure states in the provider adapter and admin endpoint, with redacted evidence.
- [ ] T025 `[P00-022]` Enforce the secret-storage boundary across `.gitignore`, config examples, CI environments, and safety scripts; test detection of credentials, private keys, signing material, server inventory, and accidental evidence leaks.
- [ ] T026 `[P00-023]` Implement the Vue admin application shell, RBAC route guards, permission-denied behavior, audit entry point, and the P00-bound admin states in `web/admin/`; add Vitest and browser/CI screenshot tests.
- [ ] T027 `[P00-024]` Implement the isolated Vue developer portal shell with stable routes and loading/empty/error/success states in `web/developer/`; add Vitest and browser/CI screenshot tests.
- [ ] T028 `[P00-021..P00-024]` Run provider, secret, Web, RBAC, browser screenshot, visual-diff, and safety tests; finalize evidence, commit cleanly, and close only `P00-W05` through the controller.

## P00 Integration and Delivery

- [ ] T029 `[P00-001..P00-024]` Confirm all P00 Feature statuses are final with concrete evidence or an allowed external reason; run phase contracts, backend, Web, Android, security, UI, migration, idempotency, and staging health gates for the exact clean revision.
- [ ] T030 `[P00-001..P00-024]` Run `./ylven.ps1 release -Phase P00`; wait for the exact GitHub Actions revision, verify the CI run and Artifact provenance, and download only that tested Artifact.
- [ ] T031 `[P00-001..P00-024]` Verify APK version/commit/SHA-256, copy the exact tested APK and P00 delivery evidence to the Desktop, and report tests, screenshots, endpoints, Owner Actions, and known limitations.
- [ ] T032 `[P00-001..P00-024]` Stop at P00 owner acceptance with P01 untouched. Do not run `close-release`; wait for the project owner.
