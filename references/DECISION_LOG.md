# Fixed Decision Log

| ID | Decision | Rationale | Revisit trigger |
|---|---|---|---|
| D-001 | Android native Kotlin/Compose first; no iOS in current scope | focused delivery and existing owner preference | Android commercial baseline accepted and iOS business case approved |
| D-002 | Four tabs: Home, Work, Discover, Mine | stable information architecture | owner-approved product ADR |
| D-003 | Android -> YLVEN API -> gateway/provider; never Android -> Sub2API | secret safety, unified data and migration | never; only internal transport may change |
| D-004 | YLVEN owns identity, money, entitlements and product data | prevents dual authority | only formal migration ADR |
| D-005 | Subscription proxy is test-only | terms, account and continuity risk | production official/authorized channels available |
| D-006 | Modular core plus separate AI Runtime/Gateway/Worker | scale without premature microservices | measured bottleneck and extraction plan |
| D-007 | Spec Kit Lite with Codex; no Kiro/Governance V5.0 | avoid duplicate state and inspection overhead | development tool replacement decision |
| D-008 | Machine-enforced release package | prevents long-session delivery drift | never remove; tooling may improve |
