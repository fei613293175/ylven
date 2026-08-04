# P00 Validation Quickstart

## Prerequisites

- PowerShell, Git and uv on the local control client.
- Existing GitHub `origin` and existing staging SSH profile.
- External provider credentials are optional; their absence must produce `NOT_CONFIGURED`, not fake success.

## Resume and scope

```powershell
Set-ExecutionPolicy -Scope Process Bypass
.\ylven.ps1 resume
```

Expected: current phase `P00` and only the controller-selected `P00-Wxx` packet.

## Packet validation

```powershell
.\ylven.ps1 py -Script scripts/19_VALIDATE_UI_CONTRACTS.py --mode planning
.\ylven.ps1 py -Script scripts/20_CHECK_UI_VISUAL_GATE.py --packet P00-Wxx
.\ylven.ps1 py -Script scripts/07_VALIDATE_CONTRACTS.py --phase P00
.\ylven.ps1 py -Script scripts/40_PUBLIC_REPOSITORY_SAFETY.py
```

Run only affected Go, Web, Android and integration tests during a packet. Update `status/P00_FEATURE_STATUS.yaml` with real evidence, commit the implementation, then close the exact current packet through the controller.

## Phase acceptance

After all five packets are final, run the repository P00 release command. The release workflow must build and test Android version `1.0.0` (`1000000`), execute emulator interaction and screenshot comparison, verify provenance and deliver the same Artifact to `~/Desktop/YLVEN-Releases/1.0.0`.

Expected final state for this request: P00 Artifact delivered, owner acceptance pending, no `close-release` invocation and no P01 changes.
