# ADR-010: Preserve P03 scope when the conversation menu mockup shows project assignment

- Status: Proposed - requires explicit P03 owner release decision
- Date: 2026-08-11
- Owner approval: No approval for this exception is recorded.
- Related Feature IDs: P03-005, P03-006, P03-007, P03-029, P06-007

## Context

`YL-A-022` is a P03 conversation-menu overlay, and its approved visual baseline includes
the row `移入项目`. The same page contract and the API inventory assign actual project
membership to P06-007. P03 has no projects endpoint, project data model, or P06 feature
authorization. The exact P03 APK therefore renders the row disabled with an explanation.

## Decision

For P03, retain the row to preserve the approved menu layout but keep it disabled. The
row becomes actionable only when P06-007 is implemented and its project contract is
available. This preserves the P03 feature boundary and avoids inventing a project action.

## Alternatives considered

1. Implement project assignment in P03. Rejected: it advances P06 functionality and
   violates the feature map.
2. Hide the row. Rejected: it changes the approved visual composition and leaves an
   unexplained visual difference.
3. Mark the disabled row as visually identical to an enabled control. Rejected: it would
   misrepresent an unavailable operation as usable.

## Consequences

The P03 screenshot has an intentional interaction-state difference from the sample. It
needs owner acceptance as a release exception, or a future approved P03 mockup that
shows the disabled state. No P03 code or data is changed by this ADR.

## Security, data, billing and compatibility impact

Avoiding an unauthorized project write prevents cross-phase and ownership errors. There
is no migration, billing, or compatibility change.

## Migration and rollback

P06-007 will replace the disabled state with its own tested implementation. Removing this
record without either a P03 mockup revision or P06 implementation restores an unresolved
contract conflict.

## Validation evidence

- `ui/pages/android/YL-A-022.yaml` lists both P03 menu features and P06-007.
- `contracts/api-inventory.csv` assigns `PUT /api/mobile/v1/conversations/{conversationId}/project` to P06-007.
- Current physical-device representative screenshot: `YL-A-022-PRODUCTION.png`.
