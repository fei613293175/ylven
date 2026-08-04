# Research Decisions: P00 Engineering Foundation

## Runtime topology

- **Decision**: Use the five deployment units already fixed in `docs/06_SYSTEM_ARCHITECTURE_AND_DEPLOYMENT_UNITS.md`: Core API, AI Runtime, Developer Gateway, Worker and Scheduler.
- **Rationale**: These boundaries isolate streaming/provider workloads and public API traffic while keeping domain code in a modular monolith.
- **Alternatives considered**: One process would couple failure domains; additional microservices would violate P00's anti-complexity constraint.

## Version authority drift

- **Decision**: Use the current architecture baseline (Go 1.26.5, PostgreSQL 18.4, Redis 8, NATS 2.14.x) and pinned Android catalog; treat older values in `references/VERSION_PINS.md` as stale explanatory material.
- **Rationale**: Machine contracts and current phase/architecture documents precede explanatory references. CI lockfiles and image digests remain the executable authority.
- **Alternatives considered**: PostgreSQL 17/Redis 7 were rejected because they conflict with the newer phase architecture snapshot; silently mixing majors was rejected.

## Local versus remote execution

- **Decision**: Keep the owner machine as a thin control client. GitHub Actions performs builds/emulator/visual regression; the existing server performs Docker and staging integration.
- **Rationale**: This is a canonical V1.6 project fact and avoids recreating established environments.
- **Alternatives considered**: Installing Java, Go, Node, Docker and Android SDK locally is unnecessary and explicitly not a blocker.

## External capability proof

- **Decision**: Implement real probe adapters with explicit `NOT_CONFIGURED`, `SUPPORTED`, `UNSUPPORTED` and `FAILED` results; use contract fakes only in automated tests.
- **Rationale**: Missing credentials must not block unrelated implementation, but fake success cannot be release evidence.
- **Alternatives considered**: Static success responses and hard-coded model capability tables were rejected.

## UI implementation and evidence

- **Decision**: Bind UI only to approved mockups and `YL-DS-1.2.0` tokens, producing one runtime screenshot per applicable State ID in CI.
- **Rationale**: The manifest contains 2,671 approved state images and the phase contract requires exact state-level evidence.
- **Alternatives considered**: Inferring states from one success screenshot or recreating images without approval was rejected.

## Release provenance

- **Decision**: Deliver only the exact Android Artifact tested by the required GitHub Actions workflow, with commit/version/run/artifact/hash provenance.
- **Rationale**: It prevents local rebuild drift and is required by the release controller.
- **Alternatives considered**: Local APK builds and manually assembled TESTS.md were rejected as non-authoritative.
