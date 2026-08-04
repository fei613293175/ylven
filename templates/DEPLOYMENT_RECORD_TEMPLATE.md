# DEPLOYMENT EVIDENCE

- Phase: Pxx
- Deployment result: SUCCESS
- Health check result: PASS
- Rollback result: PASS
- Environment: staging
- Deployed at:
- Deployed by: Codex
- Git commit:
- Container/image digests:

## Configuration and secret references

List secret-reference names and configuration versions only. Never paste secret values.

## Database migrations

List migration IDs, execution result, compatibility window and rollback/forward-fix decision. Write `NOT_APPLICABLE` with a reason when the phase has no migration.

## Services changed

List each deployed service, version/image digest, replica count and affected public/internal routes.

## Health and smoke checks

Record exact commands/URLs, timestamps, HTTP results and relevant Feature IDs. A statement such as “checked successfully” is insufficient.

## Feature flags and model routes

Record changed flags/routes, old value, new value, rollout scope and rollback value. Write `NOT_APPLICABLE` with a reason when unchanged.

## Rollback evidence

Record rollback command or deployment revision, data compatibility assessment and whether a non-destructive dry run/simulation was performed. Use `- Rollback result: NOT_APPLICABLE` only with a concrete reason.

## Known limitations and Owner Actions

Reference `OWNER_ACTIONS.md` and the exact Feature IDs for any external configuration or accepted limitation.
