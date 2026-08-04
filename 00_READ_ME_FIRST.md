# YLVEN AI 商业系统开发计划包 V1.6.0

> 核验日期：2026-08-04  
> 目标读者：项目所有者、Codex、Android/后端/前端开发人员、测试与运维人员。  
> 用途：作为 YLVEN Android AI 软件从零开发的唯一产品与工程基线。本包不是聊天摘要；其中功能、架构、数据权威、验收和交付规则均用于实际编码。

## 已固定的决策

1. 品牌 YLVEN，首期仅 Android，不开发 iOS，暂不考虑应用商店上架。
2. 一级导航：首页、工作、发现、我的。
3. YLVEN 自建业务后端、数据库、管理后台和开发者中心；Android 不直连 Sub2API。
4. 当前测试使用已部署 Sub2API 和个人 ChatGPT/Claude/Grok 订阅；生产架构可切官方 API。
5. 主域名 `orbexa.cc`；Cloudflare 提供橙云、WAF、Turnstile、CDN 和 R2。
6. 注册：邮箱、登录密码、确认登录密码；Turnstile 后发送邮箱验证码并确认注册。
7. 登录：邮箱 + 邮箱验证码；发送验证码前先完成 Turnstile。
8. YLVEN 是用户、订单、账本、套餐、权益和产品数据权威；Sub2API 是上游调度/计量执行层。
9. 不使用 Governance V5.0，不使用 Kiro；使用 Codex + Spec Kit Lite + 机器合同 + 脚本/CI。
10. 每个阶段必须机械生成 APK、功能/测试/变更/部署清单、构建信息和校验和，并复制到桌面。


## 包内规模与追踪能力

- 362 个前端—后端—后台—数据—测试 Feature ID；
- 14 个阶段（P00～P13）；
- 1,086 个测试 ID；
- 201 个数据/基础设施实体；
- 225 个 OpenAPI operation，支持共享端点绑定多个 Feature ID；
- 详细内容索引见 `PACKAGE_CONTENTS.md`。

## 本开发包的最终核验状态

- 机器合同再生成和交叉校验：通过；
- P00 隔离发布控制自测：13 个步骤全部通过，包括缺少发布目录时拒绝完成、所有者未批准时拒绝推进；
- PowerShell 脚本静态检查：8/8 通过，覆盖 UTF-8 BOM、CRLF、字符串、注释、here-string、括号和严格模式；
- Spec Kit/Codex 自动集成合同：通过，固定 `v0.15.2`，目标目录 `.agents/skills`，调用格式 `$speckit-*`；
- 合成 APK 仅用于隔离测试发布校验器，测试后已删除，不包含在最终 ZIP；
- 本包是开发计划与仓库引导包，不是已经完成的 YLVEN App，因此不包含真实应用 APK。真实 APK 从 Codex 执行 P00 后由 Gradle 构建。

详细证据：`PACKAGE_SELF_TEST_REPORT.md`、`POWERSHELL_SYNTAX_REPORT.md`、`SPECKIT_INTEGRATION_VALIDATION.md`、`PACKAGE_VALIDATION_REPORT.md`。

## 先读哪些文件

1. `YLVEN_MASTER_DEVELOPMENT_PLAN.md`；
2. `docs/01_PROJECT_CHARTER_AND_FIXED_DECISIONS.md`；
3. `docs/16_PHASED_DEVELOPMENT_PLAN_P00_TO_P13.md`；
4. `contracts/FEATURE_MATRIX.md`；
5. 当前 `phases/Pxx_*.md`；
6. `codex/START_HERE_PROMPT.md`。

## 项目所有者只需做什么

- 首次将 ZIP 解压到新仓库并运行引导脚本；
- 按 `OWNER_ACTIONS.md` 完成确实必须人工配置的 DNS、R2、Turnstile、邮件和 Sub2API 密钥；
- 每阶段安装桌面交付 APK 并体验后台；
- 用验收模板反馈问题；
- 确认通过后发继续阶段提示。

日常代码、自动测试、构建、部署和交付文档由 Codex 完成。

## Windows 首次执行

```powershell
Set-ExecutionPolicy -Scope Process Bypass
.\ylven.ps1 bootstrap -CreateInitialCommit -PushInitialCommit
```

脚本会通过网络固定安装 Spec Kit v0.15.2、初始化 Codex Skills、部署 YLVEN 本地模板、复制短版 AGENTS、建立阶段状态并执行自检。版本、集成状态、核心 Skill 或覆盖模板任一不符合即停止。完成后重新打开 Codex，粘贴 `codex/START_HERE_PROMPT.md`。

## 禁止

- 不在一个超长会话连续开发全部阶段；
- 不为每个小改动运行完整 Spec Kit 流程；
- 不靠聊天记忆交付 APK；
- 不用假数据或静态壳冒充功能完成；
- 不让 App 保存 Sub2API/API 管理密钥；
- 不在个人订阅中转上正式对外商业销售 API；
- 不重新引入复杂 Attempt/FAILED_BOUNDED 治理状态机。

<!-- UI_PACKAGE_ENTRY_V1_2 -->

## 本版新增：页面与效果图合同

先阅读 `docs/26` 至 `docs/33`。当前已完成 **332 个页面/组件合同、2,671 个独立状态效果图和 64 个小版本绑定**。全部 PNG 已生成、通过精确重复审计并在 Manifest 中标记为 `APPROVED`；Codex 可据此实施 UI，但每个 APK 版本仍必须提交真实运行截图和视觉差异供最终验收。

- V1.5 update notes: `UPGRADE_NOTES_V1.5.md`.


## V1.6 连续开发与环境修正

- 公开空仓库是允许的首次状态；Bootstrap 自动初始化 `.git` 和 `main`。
- 本机不需要 Java、Go、Gradle、ADB、Android SDK、Docker 或系统 Python；工具 Python 由 uv 放入 `.venv-tools`。
- 新会话先运行 `.\ylven.ps1 resume`；工作包和正式版本都必须机械关闭并写入追加式历史。
- 详见 `docs/36_VERSION_WORK_PACKET_CLOSURE_LEDGER_AND_SESSION_RECOVERY.md` 与 `docs/36_VERSION_CLOSURE_SESSION_RECOVERY_PUBLIC_REPO_AND_THIN_CLIENT.md`。

<!-- YLVEN_V1_6_BOOTSTRAP -->
## V1.6 启动方式

即使解压目录没有 `.git`，也直接在包根目录运行 `./ylven.ps1 bootstrap`。脚本会初始化 Git、绑定公开空仓库、创建 main 首次提交和 P00 阶段分支，并安装 uv/Spec Kit。它不会要求本机安装 Java、Go、Gradle、ADB、Docker 或系统 Python。每次新 Codex 会话运行 `./ylven.ps1 resume`。


## V1.6 版本连续性与执行环境

- 新 Codex 会话第一条命令固定为 `.\ylven.ps1 resume`；不得根据聊天中的“下个版本”猜测。
- 每个 Work Packet 必须通过 `.\ylven.ps1 close-packet -Packet <ID>` 关闭；控制器验证 Feature 最终状态、理由、证据、测试和实现 Commit，追加哈希链历史，自动选择下一包，并自动提交状态变更。
- 每个正式 APK 版本只有在精确 CI Artifact、项目所有者 `APPROVED` 后，才能通过 `.\ylven.ps1 close-release -Phase <Pxx>` 关闭；控制器写入发布总账、创建不可变 Tag、自动提交状态并推进下一阶段。
- 仓库公开是项目所有者确认的合法事实；缺少 `.git`、远程为空或没有默认分支由 Bootstrap 自动修复，不再视为业务阻塞。
- 本机为轻量控制端，只要求 Git、PowerShell、uv、Spec Kit、OpenSSH，以及发布时使用的 GitHub CLI；Java、Go、Gradle、ADB、Android SDK、Docker、Node 和系统 Python均不是本机启动门槛。
- GitHub Actions承担构建、模拟器与视觉回归；线上服务器承担 Docker、数据库及 staging 集成。
