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
- 所有版本必须保持 `cc.orbexa.ylven` 和单调递增 `versionCode`。默认先安装上一版本，再执行同一正式签名的 `adb install -r` 覆盖安装，禁止卸载、清除数据或换包名绕过。唯一例外是项目所有者于 2026-08-08 明确批准的 P02 -> P03 一次性签名迁移，必须严格匹配 `contracts/signing-migrations/P03.properties`，如实记录卸载、旧本地数据/登录态丢失和新证书；P03 建立的新正式签名是 P04 及以后版本的固定升级基线。
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
- Resolve the current Android version only from `contracts/release-version-matrix.yaml`. Keep applicationId/signing stable and test `adb install -r` upgrade installation, except for the single owner-approved P03 migration recorded in `contracts/signing-migrations/P03.properties`.
- Generate `DNS_ACTION_REQUIRED.md` whenever `contracts/domain-delivery-map.yaml` assigns a domain to the current phase. Never guess DNS targets.
- Owner APKs must be built from the exact Git commit on the SSH-connected online server, downloaded with SHA-256 verification, and tested only on the project owner's physically connected Android phone. GitHub Actions, Android Emulator and every other simulator are forbidden for APK testing.
- Before any APK operation, `adb devices -l` must report the selected serial as exactly `device`. An unavailable device stops testing with no simulator fallback. When multiple eligible physical devices are connected, prefer an idle device; when all are occupied, choose the shortest shared FIFO queue and wait without interrupting another project. An explicit serial remains allowed.
- Real-device acceptance must install and launch the APK, inspect crash/ANR/log output, exercise every current-phase page with click/input/back/scroll and key business flows, capture every defined page state on the phone, compare it with the approved mockup, and record device/build/path/screenshots/issues. Any functional or visual failure blocks delivery.
- Work fast: one full preflight per packet, affected tests during implementation, full acceptance only at phase release. Repeated analysis without output is prohibited.

<!-- YLVEN_V1_6_CONTINUITY_RULES -->
## V1.6 版本连续性、公开仓库与轻量本机规则

- 新会话、上下文压缩恢复、切换分支或拉取更新后，先运行 `.\ylven.ps1 resume`；不得根据聊天猜测下一个版本。
- `CURRENT_PHASE.yaml`、`CURRENT_WORK_PACKET.yaml` 与 `status/WORK_PACKET_STATUS.yaml` 是运行状态权威；`contracts/work-packet-map.yaml` 只定义静态范围。
- 工作包只能通过 `.\ylven.ps1 close-packet -Packet <ID>` 或有明确外部原因的 `defer-packet` 最终化。控制器验证 Feature、证据、测试、干净实现 Commit，追加哈希链历史，提交状态并选择下一包。
- 正式 APK 版本先通过 `.\ylven.ps1 release -Phase <Pxx>` 完成线上服务器精确 Commit 构建、SHA-256 下载复核和项目所有者本机物理 Android 手机验收；项目所有者 `APPROVED` 后，才能运行 `.\ylven.ps1 close-release -Phase <Pxx>`，写入发布总账、创建不可变 Tag并推进阶段。
- 仓库公开是项目所有者确认的事实。不得要求改为 Private；必须在提交、推送和发布前通过公开仓库安全门禁，且不得提交任何 Secret、服务器详情、签名材料或一次性管理员凭据。
- 缺少 `.git`、远端为空或无默认分支时运行 `.\ylven.ps1 bootstrap`；这属于正常初始化，不是业务阻塞。
- 本机是轻量控制端，不因缺少 Java、Go、Gradle、Android SDK、Docker、Node 或系统 Python而停止。使用 uv 执行仓库脚本；线上服务器承担 Android 构建及重型自动测试，项目所有者本机的真实 Android 手机承担 APK 功能、日志和视觉验收。发布验收需要 ADB，可使用仓库忽略目录中的 platform-tools，不要求系统级安装。
- 本机 `git push` 失败时的固定备用路径：保持同一 Commit SHA、阶段分支和 `origin` 不变，通过已验证的 SSH/SFTP 将完整 Git Bundle 传到服务器，校验 SHA-256 与 `git bundle verify` 后由服务器推送 GitHub；不得把 Token、密钥或任何 Secret 放进仓库、Bundle、命令行或日志，推送后必须回读 GitHub 分支 SHA。
- 一次必要预检后直接实现；输入未变化时不得重复全仓库审计。真实代码、测试和运行结果优先于长篇检查报告。

<!-- YLVEN_DELIVERY_CONVERGENCE_V1 -->
## 交付收敛与效率规则（所有阶段强制）

- 每个工作包开始时建立并冻结候选版本：记录 Commit、配置版本、待验证 Feature ID、必要证据和已知问题。候选版本未发生代码、配置或测试输入变化时，禁止重复构建、全量测试、全量截图或全量视觉矩阵。
- 验收发现的问题先登记为一份按交付阻断等级排序的问题清单。只处理会导致功能、数据安全、发布合同、真实设备流程或批准视觉状态不通过的项目；非阻断的优化、文案偏好和未受影响页面不得插入当前阶段发布路径。
- 每个问题必须遵循“证据 -> 明确根因 -> 最小改动 -> 受影响测试 -> 受影响真机回归”的单向流程。没有明确根因或可复现证据时，不得猜测性改代码、重新打包或扩大回归范围。
- 同一问题的一次定向复测失败后，立即保留请求 ID、日志、截图、版本和复现步骤，并判断责任层（客户端、YLVEN 服务、上游服务、设备或合同）。同一输入连续两次无新增证据时停止重复操作，向项目所有者报告明确阻断和所需外部条件。
- 一个候选版本在最终发布前最多进行一次完整真机验收；候选版本变更后仅回归受影响功能、页面和状态。只有影响跨模块合同、签名、安装升级、认证或发布工件的变更，才需要重新进行完整验收。
- 视觉验收应一次采集每个定义状态并直接审查。仅为遮挡、错误状态或缺失状态补采对应页面；不得因单张截图或非阻断视觉差异反复重跑整个矩阵。效果图与功能合同冲突时，功能合同优先，记录 ADR/合同修订待办，不得篡改正确功能或伪造视觉通过。
- 构建、服务器测试、设备验收和证据整理是独立步骤：构建完成后先确认交付阻断清单为空，再进入真机；真机结束后再整理交付物。不得因报告排版、历史证据整理或无关截图占用测试设备或阻塞候选版本结论。
- 设备队列只在实际需要安装、操作或采集手机证据时持有。完成该批操作后，校验 owner 仅释放本项目自己的锁和票据，立即让出设备；不得因待分析、待构建或待整理报告继续占用设备。
- 本机 C 盘工作区按证据类别最多保留最近两份已完成的历史目录。删除前必须核对当前候选和正式交付物不再唯一引用该目录；旧 APK、副本、压缩中间物、构建分块、临时截图和诊断日志均应优先清理，严禁为方便回溯无限保留。

<!-- YLVEN_OWNER_EXECUTION_RULES_V1 -->
## 项目所有者强制开发与真机验收规则

- 独立核验优先于迎合结论。每次工作先区分已证实事实、待验证推测与主观判断；发现逻辑跳跃、信息缺口、互相矛盾的状态、无来源数字或错误结论时，必须直接指出证据、风险和替代解释。
- 本机仅作为轻量控制端和真实 Android 设备接入端。未经项目所有者单次明确授权，禁止在本机安装、搭建或运行任何项目构建、部署、非设备测试、Android Emulator、其他模拟器或虚拟设备；Android 构建、非设备测试和部署一律在已连接服务器执行。不得以本机环境缺失为由改用模拟器。
- APK 测试只能使用项目所有者物理连接的 Android 手机。每次安装、启动、点击或截图前，必须重新确认目标序列号的 `adb devices -l` 状态严格为 `device`，并保持原始显示分辨率与密度，不得为测试改变设备显示设置。
- 多台手机同时可用时，检测空闲设备并选择可立即使用者；若设备被其他项目占用，登记并遵守共享 FIFO 队列，持续等待而不抢占、不删除或修改其他项目的锁和票据。只在实际设备操作期间持有本项目锁，结束后校验 owner 并立即只释放自己的锁和票据。设备不可用时停止设备测试，等待可用真机，不得切换为模拟器。
- 真机验收必须覆盖 APK 安装和启动、崩溃/ANR/应用日志、每个当前阶段页面的真实点击、输入、返回、滚动和关键业务路径。每个定义页面状态均须在真机采集截图，并逐页对照批准效果图检查布局、字体、颜色、间距、图标与交互状态；证据必须记录设备、APK SHA-256/版本、测试路径、截图和问题。
- 任一关键功能、关键视觉状态、定义业务流程或所需证据未通过/缺失，均不得报告“通过”、不得交付 APK、不得关闭阶段。效果图与功能合同冲突时，功能合同优先，并写明 ADR 或验收例外，不得伪造视觉通过。
- 为控制交付时间，候选冻结后按“证据 -> 明确根因 -> 最小改动 -> 受影响服务器测试 -> 受影响真机回归”推进；未形成根因链不得猜测性修复、重打包或重跑全量验收。重复操作未产生新增证据时必须停止并报告明确外部阻断。
- 本机 C 盘的每类历史构建、APK、副本、压缩中间物、临时截图、日志和验收目录最多保留最近两份已完成数据；删除前确认当前候选、失败根因证据与正式交付物不再唯一引用，随后及时清理更旧的无用数据。
