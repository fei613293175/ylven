# Visual Diff Report — P01-W05

- Design system: `YL-DS-1.2.0`
- Runtime canvas: `1440×1000 CSS px`, light theme, font scale `1.0`
- Result: `CAPTURED_FOR_OWNER_REVIEW`

| Page | Runtime states captured | Review |
|---|---|---|
| YL-M-027 | S02_POPULATED | Shell, filters, table and real user status action captured |
| YL-M-037 | S02_POPULATED | Shell, filters, role table and permission editor captured |
| YL-M-038 | S02_POPULATED | Shell, filters, administrator/session table captured |
| YL-M-039 | S02_POPULATED | High-risk approval cards and confirmation entry captured |
| YL-M-040 | S07_SAVE_SUCCESS | Versioned template panel, step-up modal and save feedback captured |

The approved mockups contain illustrative rows and labels. Runtime evidence uses API-returned data and therefore intentionally does not copy those sample values. The five captured states are the owner-review baseline; the remaining contract states are exercised by the loading, permission, error and write-state branches in `App.js` and must be included in the exact CI visual acceptance run.
