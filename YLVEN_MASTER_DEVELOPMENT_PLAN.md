# YLVEN AI 商业系统总开发计划 V1.6.0

> 基线日期：2026-08-04  
> 项目域名：`orbexa.cc`  
> 首期平台：Android 原生 App + YLVEN 后端 + 管理后台 + 开发者中心  
> 开发方式：Codex + Spec Kit Lite + 机器可读合同 + 自动测试与交付脚本

## 1. 项目目标

YLVEN 将 ChatGPT、Claude、Grok 及未来其他 AI 的能力统一到一套自有产品中。用户在同一 Android App 内完成对话、识图、文件分析、图片生成/编辑、PPT 创作、项目知识管理和作品管理；每个会话或每条消息可选择模型和推理强度，并清楚看到实际模型来源。

系统同时为未来商业化建立专业管理后台、套餐/钱包/账本、统一账号、开发者 API 和 Sub2API 轻度兼容。当前内部测试使用已接入 Sub2API 的个人订阅通道；架构必须支持未来切换官方 API 而无需重写 Android App、用户体系、会话、文件或钱包。

## 2. 固定产品结构

### 首页

AI 对话中心：新建/历史会话、流式回答、模型和推理选择、单条换模型、多模型对比、文件/图片、语音输入、引用、复制/分享、项目关联。

### 工作

工具、项目、作品、任务：图片工作台、PPT 工作台、文件资料库、项目知识库、多模型工作流、异步任务和版本化作品。

### 发现

由后台动态配置的服务中心：模板、模型实验室、工作流、连接器、技巧、活动、公告和后续服务，不发布只有“敬请期待”的空白页面。

### 我的

头像、用户名、UID、套餐、钱包、用量、项目/文件/作品、AI 默认设置、自定义指令、记忆、设备、安全、开发者 API、通知、主题、数据和 App 更新。

## 3. 固定技术架构

- Android：Kotlin + Jetpack Compose + Material 3 + Room + OkHttp/Retrofit；
- Core API / AI Runtime / Developer Gateway / Worker / Scheduler：Go；
- 管理后台/开发者中心：Vue 3 + TypeScript；
- 数据：PostgreSQL、PgBouncer、Redis、NATS JetStream；
- 文件和 CDN：Cloudflare R2 + 自定义域名；
- 边缘：Cloudflare DNS 橙云、WAF、Turnstile、CDN；
- 可观测性：OpenTelemetry、Prometheus、Grafana、Loki、Tempo；
- 部署：首期 Linux + Docker Compose，按指标扩容，不提前复杂微服务化。

Android 只访问 YLVEN API，绝不直连 Sub2API。YLVEN 会话/文件/钱包数据不依赖上游供应商 ID。

## 4. 身份固定方案

注册字段：邮箱、登录密码、确认登录密码。点击注册后先完成 Turnstile 安全验证，再发送注册邮箱验证码；验证码通过后创建正式用户并自动登录。

普通登录：邮箱 + 邮箱验证码。请求验证码前先完成安全验证。注册密码首期不在普通登录页使用，但安全哈希保存，用于未来密码登录和高风险操作。YLVEN 是身份权威，未来通过 OIDC 让用户单点登录 Sub2API。

## 5. 商业与 Sub2API 兼容

YLVEN 管理用户、订单、支付、不可变账本、套餐和产品权益；Sub2API 管理上游账号池、路由、实时用量、API Key 和限流。两个系统独立数据库，不同步密码，不双向覆盖余额。

App 和外部 API 可共享总 AI 额度，但外部 API 有独立预算和单 Key 限制。所有充值入口进入 YLVEN Payment Center。当前订阅代理明确标记为测试通道，商业开放前切换官方/授权 API。

## 6. 后端专业能力

- 动态模型和能力注册；
- 供应商 Adapter、能力探测、路由、熔断和透明回退；
- 结构化会话、消息部件、Run、分支和多模型对比；
- 文件直传、解析、知识索引和引用；
- 图片/PPT 异步任务与版本化作品；
- 产品、套餐、权益、钱包、不可变账本和对账；
- 公共 API Key、预算、限流、日志和开发者门户；
- 专业后台的配置、运营、审计、可观测性和回滚；
- 高并发下的无状态扩容、SSE 恢复、队列隔离和数据分区。

## 7. 前后端一一映射

完整映射位于：

- `contracts/FEATURE_MATRIX.md`；
- `contracts/feature-map.csv`；
- `contracts/feature-map.yaml`。

每个功能必须有 Feature ID，并明确 Android/Admin/Developer 页面、用户动作、UI 状态、接口、服务、表、后台菜单、权限、计费、异步任务、错误和测试。未映射功能不得口头宣布完成。

## 8. 开发阶段

P00～P13 详见 `docs/16_PHASED_DEVELOPMENT_PLAN_P00_TO_P13.md` 和 `phases/`。阶段顺序覆盖工程基座、身份、聊天、多模型、文件、项目、图片、PPT、产品整合、商业化、开发者 API、性能安全和最终加固。

## 9. Codex 与 Spec Kit Lite

首次运行 `scripts/00_BOOTSTRAP_REPOSITORY.ps1` 自动安装 Spec Kit v0.15.2、Codex Skills 和项目模板。项目只执行一次 Constitution；每个大阶段使用 Specify、Plan、Tasks、Implement；可选检查仅用于高风险阶段。禁止每个小修复跑全套流程。

每个阶段使用新 Codex 会话，读取 CURRENT_PHASE、阶段文档、Feature Map 和 Spec Kit artifacts。Codex 不自行选择下一阶段。

## 10. 自动交付

阶段完成必须执行发布和校验脚本，生成：APK、FEATURES、TESTS、CHANGELOG、DEPLOYMENT、BUILD_INFO、SHA256，并复制到桌面。没有脚本成功结果，不得标记完成。

项目所有者主要负责提供必要外部配置、最终 APK 真机测试和后台体验。Codex负责规划、编码、自动测试、部署、文档和交付。

## 11. 质量和风险底线

- 不用假数据或占位页冒充完成；
- 不把模型/价格/权益写死在 APK；
- 不保存上游内部思维链或伪造推理过程；
- 不把个人订阅通道作为商业生产 API；
- 不允许两套支付/余额相互覆盖；
- 不允许用户文件跨账号访问；
- 不允许密钥进入 Git/APK/日志；
- 不为高并发提前制造难以维护的微服务；
- 不允许治理检查长期代替实际开发。

## 12. 首次开始

1. 解压本包到新仓库根目录；
2. 运行 `scripts/00_BOOTSTRAP_REPOSITORY.ps1`；
3. 重新打开 Codex；
4. 粘贴 `codex/START_HERE_PROMPT.md`；
5. Codex从 P00 开始，阶段末交付 APK 和后台地址；
6. 项目所有者真机/后台验收后，用继续提示进入下一阶段。

## 13. 本包的可执行规模

本包将产品范围拆解为 362 个 Feature ID、14 个阶段、1,086 个测试 ID、201 个实体目录项和 225 个 OpenAPI operation。数量不是完成目标；每个条目必须通过机器合同追踪到真实代码、页面、接口、后台、数据和测试。共享接口使用 `x-feature-ids` 保存全部功能归属，防止后生成合同覆盖先前功能。

完整索引见 `PACKAGE_CONTENTS.md`，追踪规则见 `docs/23_TRACEABILITY_AND_COMPLETION_ENFORCEMENT.md`。

## 14. 开发包与真实产品交付的边界

本 ZIP 交付的是可执行开发基线、详细功能和架构合同、Codex/Spec Kit Lite 自动引导、阶段状态、测试目录和机械发布工具。包内发布控制已通过隔离自测，但不会以合成 APK 冒充产品 APK。Codex 在真实仓库执行 P00 后，只有 Gradle 生成且通过 `scripts/16_VERIFY_RELEASE.py` 的 APK 才能进入阶段交付目录；随后仍需项目所有者真机和后台验收，未批准不得进入下一阶段。

<!-- UI_CONTRACT_MASTER_V1_2 -->

## UI 视觉合同基线

项目使用 `YL-DS-1.2.0`、332 个 Page ID、2671 个 State/Mockup ID 和 64 个 Work Packet。任何页面在状态效果图未批准前不得由 Codex自由设计；功能以 Feature ID 为准，视觉以 Token 与批准效果图为准。


## V1.5 视觉身份重绑定基线

- 保留 332 个页面、弹层、组件板和设计板合同；状态从旧版 2,909 项收敛为 **2,671 项**。减少的 238 项是错误套用的通用状态或重复身份，不是遗漏功能。
- `YL-A-011` 绑定为注册安全验证弹层，父页面 `YL-A-010`；`YL-A-012` 绑定为独立注册邮箱验证码页面。
- V1.6 包内全部正式 PNG 在 `contracts/mockup-manifest.csv` 中为 `APPROVED`，可作为 Codex 实施基线；最终 APK 仍须通过真实截图、视觉差异与项目所有者验收。
- 精确 PNG 重复、跨 Page ID 重复、同页状态重复、页面身份签名冲突与规范化页面名称冲突均为 0。
- 管理后台允许共享专业 Shell，但主内容、标题、业务字段和页面身份必须按合同区分。


## V1.6 状态闭环入口

- 当前阶段：`CURRENT_PHASE.yaml`
- 当前小版本：`CURRENT_WORK_PACKET.yaml`
- 工作包状态：`contracts/work-packet-map.yaml`
- 工作包历史：`status/WORK_PACKET_HISTORY.jsonl`
- 正式版本总账：`status/RELEASE_LEDGER.jsonl`
- 新会话恢复：`.\ylven.ps1 resume`
- 本机/CI/服务器分工：`contracts/execution-environment.yaml`

<!-- YLVEN_V1_6_STATE -->
## V1.6 continuous-development closure model

Each of the 64 Work Packets closes into a hash-chained packet history. Each P00–P13 owner-facing version closes into a hash-chained release ledger containing version, commit, immutable tag, CI run, Artifact and APK SHA-256. A fresh Codex conversation restores the exact next action from these files before it may edit code.


## V1.6 版本连续性与执行环境

- 新 Codex 会话第一条命令固定为 `.\ylven.ps1 resume`；不得根据聊天中的“下个版本”猜测。
- 每个 Work Packet 必须通过 `.\ylven.ps1 close-packet -Packet <ID>` 关闭；控制器验证 Feature 最终状态、理由、证据、测试和实现 Commit，追加哈希链历史，自动选择下一包，并自动提交状态变更。
- 每个正式 APK 版本只有在精确 CI Artifact、项目所有者 `APPROVED` 后，才能通过 `.\ylven.ps1 close-release -Phase <Pxx>` 关闭；控制器写入发布总账、创建不可变 Tag、自动提交状态并推进下一阶段。
- 仓库公开是项目所有者确认的合法事实；缺少 `.git`、远程为空或没有默认分支由 Bootstrap 自动修复，不再视为业务阻塞。
- 本机为轻量控制端，只要求 Git、PowerShell、uv、Spec Kit、OpenSSH，以及发布时使用的 GitHub CLI；Java、Go、Gradle、ADB、Android SDK、Docker、Node 和系统 Python均不是本机启动门槛。
- GitHub Actions承担构建、模拟器与视觉回归；线上服务器承担 Docker、数据库及 staging 集成。
