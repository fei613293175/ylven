# Tasks: [PHASE ID] [PHASE TITLE]

## Rules

- Every task includes one or more Feature IDs and concrete file paths or discoverable modules.
- Prefer end-to-end vertical slices over “build all backend” or hundreds of tiny file edits.
- A slice is not done until its UI/API/data/admin/observability/tests are aligned.
- External credentials create a mock/adapter task plus Owner Action; they do not block unrelated tasks.
- After two consecutive tasks without code, test or concrete blocker evidence, stop checking and identify the exact obstacle.

## Phase 0: Contract and Baseline Verification

- [ ] Confirm the current phase and exact Feature IDs; run contract validation once.
- [ ] Inspect only affected code and existing tests; capture current failing baseline.

## Phase 1+: Vertical Slices

Use this format for each slice:

- [ ] Txxx [P] `[FEATURE IDs]` Implement [user-visible slice] in `[paths/modules]`.
  - Android/web states:
  - API/events and errors:
  - Data/migration:
  - Admin/permissions/audit:
  - Jobs/provider behavior:
  - Tests and exact commands:
  - Completion evidence:

`[P]` is allowed only when tasks touch independent files/data and can be merged without hidden ordering.

## Final Integration and Release

- [ ] Update Feature Status with evidence or explicit deferred/external reasons.
- [ ] Run phase-relevant full tests and staging health checks.
- [ ] Run `05_RELEASE_PHASE.ps1` and `06_VERIFY_RELEASE.ps1`.
- [ ] Report APK, admin URL, tests, Owner Actions and limitations; stop before the next phase.

## UI 任务约束
每项 UI 任务必须绑定 Page/State/Interaction ID；效果图未批准时标记 BLOCKED_VISUAL，不得自由发挥。
