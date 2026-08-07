# ADR-009: Align YL-A-020 classification with its approved full-page baseline

- Status: Proposed — pending explicit P03 owner release acceptance
- Date: 2026-08-07
- Owner approval: No separate approval for this metadata correction is recorded. Existing approved state baselines are retained; no PNG, SHA-256, approval actor, or approval time was modified.
- Related Feature IDs: P03-003

## Context

`YL-A-020` was declared as an `OVERLAY` in the generated UI metadata, while all nine
approved `YL-A-020` baselines were generated and approved as full-canvas pages with
standard top-bar chrome. The runtime renderer followed the metadata and drew a
drawer over `YL-A-018`; the P03 exact Android CI therefore reported a structural
visual mismatch for every state of this page.

The approved PNG files and their manifest hashes are the visual authority. The
metadata classification contradicted those unchanged artifacts.

## Decision

Classify `YL-A-020` as `PAGE`, while retaining `YL-A-018` as its navigation parent.
The P03 Android acceptance renderer draws the page chrome, chips, and list layout
that correspond to the approved baselines.

## Alternatives considered

1. Keep it as an overlay and weaken or mask the visual diff. Rejected: this would
   conceal a page-identity mismatch and violate the release visual gate.
2. Regenerate the approved PNGs as drawer overlays. Rejected: that would invalidate
   the existing owner-approved baselines and require a new owner visual approval.
3. Use approved PNGs directly in the APK acceptance renderer. Rejected: runtime
   screenshots must be Canvas-rendered by the APK, not substitute baseline assets.

## Consequences

The page relationship remains available to navigation, but the visual contract no
longer imposes an overlay composition that conflicts with the approved screen. This
does not add functionality, change Feature scope, or alter any approved image.

## Security, data, billing and compatibility impact

None. This is a visual metadata and deterministic acceptance-renderer correction.

## Migration and rollback

No data migration is required. Reverting this ADR restores the prior inconsistent
metadata and will reproduce the visual-regression failure.

## Validation evidence

- The nine `ui/mockups/android/YL-A-020/*.png` hashes remain equal to the approved
  entries in `contracts/mockup-manifest.csv`.
- UI contract, generated-mockup, duplicate-audit, and P03 Android CI checks must
  pass on the exact implementation commit before release.
