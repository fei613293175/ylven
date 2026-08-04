# YLVEN Codex Project Instructions

1. `CURRENT_PHASE.yaml` is the only active phase. Do not start the next phase before owner acceptance.
2. Read `00_READ_ME_FIRST.md`, the active phase document, referenced design docs, contracts, and affected code. Do not repeatedly reread all history.
3. Use Spec Kit Lite for a major phase: `$speckit-specify` -> `$speckit-plan` -> `$speckit-tasks` -> `$speckit-implement`. Clarify/analyze only for material ambiguity or high-risk conflict.
4. Every implementation and test must reference Feature IDs from `contracts/feature-map.yaml`. Never silently remove scope.
5. Android calls only YLVEN APIs. Never put Sub2API, provider, database, R2, Turnstile, SMTP, payment, or admin secrets in the APK.
6. YLVEN owns identity, orders, ledger, products, entitlements, conversations, files, projects and artifacts. Sub2API is an isolated execution gateway with its own database.
7. Static success responses, fake balances, fake test reports, placeholder APKs and UI-only shells cannot be used to claim completion.
8. Implement first, then run affected tests. Do not repeat full-repository checks without code/config/test changes. Two no-output loops require a concrete blocker report.
9. Missing external credentials become explicit mock/feature-flag work plus `OWNER_ACTIONS.md`; they do not block unrelated implementation.
10. Money, credit, payment, retryable write, job and webhook operations require idempotency, auditability and deterministic state transitions.
11. Keep API, migrations, admin controls, metrics and Android states aligned with the Feature Map and error catalog.
12. A phase is not complete until `scripts/05_RELEASE_PHASE.ps1 -Phase Pxx` and `scripts/06_VERIFY_RELEASE.ps1 -Phase Pxx` succeed.
13. Release output contains a real Gradle APK, FEATURES, TESTS, CHANGELOG, DEPLOYMENT, OWNER_ACTIONS, OWNER_ACCEPTANCE, BUILD_INFO and SHA256 files in `dist/releases/Pxx/` and a desktop mirror when available.
14. Update `status/Pxx_FEATURE_STATUS.yaml` and `status/Pxx_IMPLEMENTATION_STATUS.md` continuously. Release statuses are only IMPLEMENTED, DEFERRED_WITH_REASON or BLOCKED_EXTERNAL.
15. Stop after reporting the current phase. Do not autonomously advance phases or introduce Governance V5.0, Kiro, Attempt or FAILED_BOUNDED state machines.

<!-- YLVEN_UI_AGENTS_V1_2 -->
## 强制 UI 与效果图合同

- 固定设计系统：`YL-DS-1.2.0`；数值权威为 `contracts/ui-design-tokens.yaml`。
- 每次开始工作包必须读取 `work-packets/<PacketID>.md`，其中重复了关键数值与绑定 Page ID。
- 页面必须读取 `ui/pages/<surface>/<PageID>.yaml`；不得凭聊天、记忆或相邻页面猜测。
- 在效果图状态未 `APPROVED` 前不得完成视觉实现；执行 `scripts/20_CHECK_UI_VISUAL_GATE.py`。
- 效果图只约束 UI；功能只来自 Feature ID。样例文案、金额、模型、头像和按钮不得转化为未登记功能。
- 禁止页面内新增硬编码颜色、字号、圆角、间距、高度、阴影和动画时长。
- 每个适用状态都要实现并截图，不能只做成功态。
- 新增页面、控件或交互必须先增加 Page ID、State ID、Interaction ID、Feature ID 和效果图合同。

<!-- YLVEN_V1_6_CONTINUITY_RULES -->
## V1.6 版本连续性、公开仓库与轻量本机规则

- 新会话、上下文压缩恢复、切换分支或拉取更新后，先运行 `.\ylven.ps1 resume`；不得根据聊天猜测下一个版本。
- `CURRENT_PHASE.yaml`、`CURRENT_WORK_PACKET.yaml` 与 `status/WORK_PACKET_STATUS.yaml` 是运行状态权威；`contracts/work-packet-map.yaml` 只定义静态范围。
- 工作包只能通过 `.\ylven.ps1 close-packet -Packet <ID>` 或有明确外部原因的 `defer-packet` 最终化。控制器验证 Feature、证据、测试、干净实现 Commit，追加哈希链历史，提交状态并选择下一包。
- 正式 APK 版本先通过 `.\ylven.ps1 release -Phase <Pxx>` 完成精确 CI Artifact 验收；项目所有者 `APPROVED` 后，才能运行 `.\ylven.ps1 close-release -Phase <Pxx>`，写入发布总账、创建不可变 Tag并推进阶段。
- 仓库公开是项目所有者确认的事实。不得要求改为 Private；必须在提交、推送和发布前通过公开仓库安全门禁，且不得提交任何 Secret、服务器详情、签名材料或一次性管理员凭据。
- 缺少 `.git`、远端为空或无默认分支时运行 `.\ylven.ps1 bootstrap`；这属于正常初始化，不是业务阻塞。
- 本机是轻量控制端，不因缺少 Java、Go、Gradle、ADB、Android SDK、Docker、Node 或系统 Python而停止。使用 uv 执行仓库脚本，GitHub Actions承担构建/模拟器/视觉回归，线上服务器承担 staging 与集成。
- 一次必要预检后直接实现；输入未变化时不得重复全仓库审计。真实代码、测试和运行结果优先于长篇检查报告。
