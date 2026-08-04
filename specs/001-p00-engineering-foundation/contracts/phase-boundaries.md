# P00 Interface and Boundary Contract

## Runtime interfaces

- Each backend process exposes `/healthz`, `/readyz` and `/version` using the canonical error envelope and Feature IDs assigned by `contracts/openapi-skeleton.yaml`.
- Core API owns mobile/admin operational reads; AI Runtime owns provider capability execution; Developer Gateway owns external API boundaries; Worker owns asynchronous consumption; Scheduler owns scheduled dispatch under a fenced distributed lock.
- Android, admin and developer clients call only YLVEN endpoints. Provider URLs, IDs and credentials never cross the client boundary.

## Infrastructure adapters

- Database, cache/rate-limit, event bus and object storage are interfaces with production and explicit test implementations.
- Test implementations cannot be enabled in production configuration and cannot be reported as external integration success.
- Retryable writes require an idempotency key and stable terminal state. Events require event IDs and duplicate handling.

## UI boundaries

- Page/State IDs and visual values come from canonical UI contracts; controls without Interaction IDs remain non-interactive.
- Admin routes require declared permissions and produce audit entries for mutations.
- The developer portal is isolated from admin routes and does not inherit administrator credentials.

## Release boundary

- Work Packets close only through `ylven.ps1 close-packet` after a clean implementation commit.
- P00 release uses `ylven.ps1 release -Phase P00` and the exact CI Artifact.
- After delivery the workflow stops for owner acceptance. `close-release` and all P01 work are out of scope until explicit owner approval.
