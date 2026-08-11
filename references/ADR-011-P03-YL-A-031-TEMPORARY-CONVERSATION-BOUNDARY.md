# ADR-011: Keep temporary conversation creation within the P03 functional boundary

- Status: Accepted for P03
- Date: 2026-08-11
- Owner approval: Approved by the project owner on 2026-08-11.
- Related Feature IDs: P03-028

## Context

`YL-A-031` is the approved P03 new-conversation form. Its sample content includes fields
for project, model, and conversation instructions, while the only P03 feature bound to
the page is P03-028: create a temporary conversation that does not enter history. Project
binding belongs to P06-007; per-conversation model configuration belongs to P04-008; and
no separate P03 feature authorizes persisted project membership or instructions. The P03
creation request only accepts a title, so the previous editable controls were misleading.

## Decision

The P03 production path creates a temporary conversation from its title only. The
production form removes project, model, and instruction inputs because none are submitted
or persisted by P03-028. The subtitle explicitly states the temporary retention behavior.
P04/P06 may introduce their own backed settings when their feature contracts exist.

## Alternatives considered

1. Persist project, model, or instruction data during P03. Rejected: it creates P04/P06
   behavior without the owning feature, API, data model, or ownership tests.
2. Retain disabled or read-only values. Rejected: there is no authoritative P03 value to
   display, so doing so would still imply a setting that does not exist.
3. Treat sample fields as functional P03 requirements. Rejected: page binding rules state
   that only registered Feature IDs authorize behavior.

## Consequences

The functional P03 temporary-conversation journey is truthful and covered by
instrumentation assertions. The project owner accepted the P03 title-only scope and its
documented visual difference on 2026-08-11. P04/P06 retain ownership of their settings.

## Security, data, billing and compatibility impact

This prevents premature project associations and their ownership implications. There is
no migration, billing, or compatibility change.

## Migration and rollback

When P06 project support is implemented, its own design and feature contract can enable
the relevant fields. Until then, retaining the P03 boundary is reversible with no data
cleanup.

## Validation evidence

- `ui/pages/android/YL-A-031.yaml` binds the page only to P03-028.
- `contracts/feature-map.yaml` assigns model configuration to P04 and project membership
  and instructions outside P03.
- `P03RealDeviceFlowTest` and `P03LiveStagingFlowTest` assert the removed labels are not
  rendered before creating a temporary conversation.
- Current physical-device representative screenshot: `YL-A-031-PRODUCTION.png`.
