---
name: ylven-contract-traceability
description: Validate and update YLVEN Feature ID traceability across phase documents, API inventory, OpenAPI, entity catalog, admin menus, tests and implementation status. Use when adding or changing product scope or when a release validation reports contract drift.
---

# YLVEN Contract Traceability

1. Treat `contracts/feature-map.yaml` as the source inventory for Feature IDs.
2. A scope change updates the Feature Map first, then generated CSV/API/OpenAPI manifests, entity/admin/test contracts, the phase document and tests.
3. Run `.\ylven.ps1 py scripts/12_REGENERATE_CONTRACTS.py`, then `.\ylven.ps1 py scripts/07_VALIDATE_CONTRACTS.py --phase Pxx`.
4. Do not delete an existing Feature ID to hide incomplete work. Use a documented deprecation or `DEFERRED_WITH_REASON` with owner-approved impact.
5. Shared endpoints preserve all `x-feature-ids`; do not overwrite earlier ownership.
6. Implementation evidence must be real code, migration, test, log or deployed endpoint paths.
