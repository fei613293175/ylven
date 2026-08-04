# Version Pin Policy

## Package-time pins

| Component | Pin/constraint | Reason |
|---|---|---|
| Spec Kit | v0.15.2 | Pinned for deterministic project workflow; prevents workflow/template drift during a phase |
| Python for scripts | 3.11+; bootstrap installs 3.12 on Windows | Spec Kit requirement and typed validation scripts |
| Java | 21 LTS target for Android/Gradle baseline | stable build tooling baseline; P00 records actual AGP compatibility |
| PostgreSQL | 17 major | current project architecture baseline; patch version updated by image digest |
| Redis | 7 major | compatible cache/rate-limit baseline |
| NATS | 2 major | JetStream task/event baseline |

Application dependencies, Docker images and GitHub Actions must be pinned through lockfiles, version catalogs or immutable digests. Do not deploy `latest`. Upgrades occur between phases on a branch with contract, migration, build and rollback evidence.
