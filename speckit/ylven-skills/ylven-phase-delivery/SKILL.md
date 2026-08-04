---
name: ylven-phase-delivery
description: Build and verify the current YLVEN phase release package after implementation, including a real APK, feature/test/deployment documents, owner actions, acceptance template, build metadata, checksums and desktop delivery. Use only when the active phase is ready for release.
---

# YLVEN Phase Delivery

1. Read `CURRENT_PHASE.yaml`, the phase status and `RELEASE_CONTRACT.yaml`.
2. Confirm all phase Feature IDs have a final release status and evidence/reason.
3. Run the project release command: `./scripts/05_RELEASE_PHASE.ps1 -Phase Pxx` on PowerShell.
4. Run `./scripts/06_VERIFY_RELEASE.ps1 -Phase Pxx` independently.
5. Never create a placeholder APK or manually fabricate TESTS.md. Release generation must consume the real Gradle artifact and real command log.
6. Return the authoritative `dist/releases/Pxx` path, desktop mirror, APK SHA-256, tests, limitations and Owner Actions.
7. Stop and wait for owner acceptance. Do not advance the phase.
