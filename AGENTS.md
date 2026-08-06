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
13. Release output contains a real Gradle APK plus the Chinese owner artifacts `原功能清单.md`、`功能完成对比清单.md`、`完整测试清单.md`、`部署证据.md`、`管理后台实测证据.md`、`截图/` and `校验文件_SHA256.txt` in `dist/releases/Pxx/` and a desktop mirror when available.
14. Update `status/Pxx_FEATURE_STATUS.yaml` and `status/Pxx_IMPLEMENTATION_STATUS.md` continuously. Release statuses are only IMPLEMENTED, DEFERRED_WITH_REASON or BLOCKED_EXTERNAL.
15. Stop after reporting the current phase. Do not autonomously advance phases or introduce Governance V5.0, Kiro, Attempt or FAILED_BOUNDED state machines.

## 全局发布硬门禁（P00-P13，任何版本不得绕过）

- 软件 Logo 固定使用 `android/app/src/main/res/drawable/ylven_logo.png`，启动图固定使用 `android/app/src/main/res/drawable-nodpi/ylven_splash.png`；资源来源和校验必须记录在版本证据中。
- 所有版本必须保持 `cc.orbexa.ylven`、正式签名证书和单调递增 `versionCode`，先安装上一版本，再执行 `adb install -r` 覆盖安装；禁止卸载、清除数据或换包名绕过。覆盖安装后必须验证数据标记、数据库迁移和登录态恢复，失败即阻断交付。
- 所有者交付目录中，原功能清单、功能完成对比清单、完整测试清单、部署证据、截图目录/截图索引和 SHA-256 校验文件必须使用中文文件名：`原功能清单.md`、`功能完成对比清单.md`、`完整测试清单.md`、`部署证据.md`、`截图/`、`截图索引.csv`、`校验文件_SHA256.txt`。英文别名不能作为正式交付物。
- 每个版本（包括没有新增后台菜单的版本）都必须由 Codex 亲自打开公共管理后台，使用交付账号登录，逐项操作当前版本对应功能，核对真实 API 数据、持久化结果和审计记录；只能健康检查、静态页面、mock 或无法回读真实数据时，不得交付。
- 以上规则同时由 `contracts/owner-delivery-contract.yaml`、`contracts/release-contract.yaml`、`contracts/release-version-matrix.yaml` 和 `contracts/android-phase-acceptance.yaml` 约束；脚本校验失败即视为未完成，不得在回复中冒充完成。

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

## V1.4 页面身份与重复图强制规则

1. 页面主体必须来自当前 Page ID 的 `visual_identity`，不得只改标题或文件夹名后复用其他页面。
2. `YL-A-011` 是注册安全验证 Overlay；`YL-A-012` 是独立注册验证码 Page。
3. `OVERLAY` 必须在 `parent_page_id` 指定父页面上实现；`COMPONENT_BOARD` 只定义组件。
4. 开发前运行 `scripts/26_VALIDATE_GENERATED_MOCKUPS.py` 与 `scripts/27_AUDIT_VISUAL_DUPLICATES.py`。
5. V1.6 包内正式效果图以 Manifest 的 `APPROVED` 状态为准；任何重新生成或修改后的 PNG 会回到 `GENERATED_PENDING_OWNER_REVIEW`，重新批准前不得作为最终视觉基线。



<!-- YLVEN_V1_5_CANONICAL_RELEASE -->
## V1.5 Canonical release and fast-execution rules

- Read `contracts/contract-authority-map.yaml` before planning. Do not create duplicate rule sources.
- Verify `origin` is exactly `https://github.com/fei613293175/ylven.git`; use a phase branch and PR, never direct feature pushes to `main`.
- Resolve the current Android version only from `contracts/release-version-matrix.yaml`. Keep applicationId/signing stable and test `adb install -r` upgrade installation.
- Generate `DNS_ACTION_REQUIRED.md` whenever `contracts/domain-delivery-map.yaml` assigns a domain to the current phase. Never guess DNS targets.
- Every owner APK must pass the GitHub Actions emulator acceptance and visual regression workflow. Deliver only the exact tested CI Artifact to the desktop.
- Work fast: one full preflight per packet, affected tests during implementation, full acceptance only at phase release. Repeated analysis without output is prohibited.

<!-- YLVEN_V1_6_CONTINUITY_RULES -->
## V1.6 版本连续性、公开仓库与轻量本机规则

- 新会话、上下文压缩恢复、切换分支或拉取更新后，先运行 `.\ylven.ps1 resume`；不得根据聊天猜测下一个版本。
- `CURRENT_PHASE.yaml`、`CURRENT_WORK_PACKET.yaml` 与 `status/WORK_PACKET_STATUS.yaml` 是运行状态权威；`contracts/work-packet-map.yaml` 只定义静态范围。
- 工作包只能通过 `.\ylven.ps1 close-packet -Packet <ID>` 或有明确外部原因的 `defer-packet` 最终化。控制器验证 Feature、证据、测试、干净实现 Commit，追加哈希链历史，提交状态并选择下一包。
- 正式 APK 版本先通过 `.\ylven.ps1 release -Phase <Pxx>` 完成精确 CI Artifact 验收；项目所有者 `APPROVED` 后，才能运行 `.\ylven.ps1 close-release -Phase <Pxx>`，写入发布总账、创建不可变 Tag并推进阶段。
- 仓库公开是项目所有者确认的事实。不得要求改为 Private；必须在提交、推送和发布前通过公开仓库安全门禁，且不得提交任何 Secret、服务器详情、签名材料或一次性管理员凭据。
- 缺少 `.git`、远端为空或无默认分支时运行 `.\ylven.ps1 bootstrap`；这属于正常初始化，不是业务阻塞。
- 本机是轻量控制端，不因缺少 Java、Go、Gradle、ADB、Android SDK、Docker、Node 或系统 Python而停止。使用 uv 执行仓库脚本，GitHub Actions承担构建/模拟器/视觉回归，线上服务器承担 staging 与集成。
- 本机 `git push` 失败时的固定备用路径：保持同一 Commit SHA、阶段分支和 `origin` 不变，通过已验证的 SSH/SFTP 将完整 Git Bundle 传到服务器，校验 SHA-256 与 `git bundle verify` 后由服务器推送 GitHub；不得把 Token、密钥或任何 Secret 放进仓库、Bundle、命令行或日志，推送后必须回读 GitHub 分支 SHA。
- 一次必要预检后直接实现；输入未变化时不得重复全仓库审计。真实代码、测试和运行结果优先于长篇检查报告。
