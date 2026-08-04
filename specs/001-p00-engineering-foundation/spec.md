# Feature Specification: P00 Engineering Foundation and Capability Proof

**Feature Branch**: `phase/p00-engineering-foundation`

**Created**: 2026-08-05

**Status**: Approved for implementation

**Input**: User description: "完成 P00 的开发、验证和交付；P00 完成交付后停止，不推进后续阶段。"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 可验证的工程与 Android 基线 (Priority: P1)

开发与验收人员可以从公开仓库恢复一致的模块、配置和 Android 工程，并在应用中看到遵循固定设计合同的四栏基础导航。

**Why this priority**: 这是所有后续服务、页面和发布工作的共同起点，必须先证明仓库与客户端基线可重复构建。

**Independent Test**: 从干净检出验证仓库边界和配置分层，构建应用并逐一切换首页、工作、发现、我的四个入口，同时核对浅色和深色主题。

**Acceptance Scenarios**:

1. **Given** 一个干净检出，**When** 执行基线校验，**Then** 模块边界、三层环境配置和秘密引用均可机械验证且不含明文 Secret。
2. **Given** 应用已启动，**When** 用户依次选择四个底部入口并切换系统主题，**Then** 当前入口状态被保留且所有颜色、排版和间距来自锁定设计合同。

---

### User Story 2 - 可诊断的运行服务骨架 (Priority: P2)

运维人员可以启动核心 API、AI Runtime、开发者网关、Worker 和 Scheduler 骨架，并从健康状态区分正常、依赖失败和不可用状态。

**Why this priority**: 后续业务纵向切片依赖稳定的服务边界、任务消费和定时调度入口。

**Independent Test**: 在无外部凭据的本地配置下启动各进程，检查健康/就绪结果、稳定错误码、队列消费幂等入口和 Scheduler 锁竞争行为。

**Acceptance Scenarios**:

1. **Given** 核心依赖可用，**When** 启动五类进程，**Then** 每个进程报告唯一身份、版本、健康和就绪状态。
2. **Given** 一个依赖不可用或锁已被占用，**When** 服务检查依赖或 Scheduler 尝试执行，**Then** 系统返回可恢复的稳定失败状态且不会重复执行任务。

---

### User Story 3 - 可替换的数据与对象存储基础设施 (Priority: P3)

开发和运维人员可以通过统一适配器使用数据库迁移、缓存限流、事件流和对象存储，并在缺少外部凭据时使用明确标记的本地实现完成非外部测试。

**Why this priority**: 数据权威、事件和对象存储必须先有确定边界，才能安全承载后续身份、会话、文件和计费功能。

**Independent Test**: 执行迁移往返、缓存/限流、事件发布消费和对象上传读取测试，并验证重复请求、超时和依赖失败的确定状态。

**Acceptance Scenarios**:

1. **Given** 本地依赖已启动，**When** 运行基础设施集成场景，**Then** 迁移、缓存、限流、事件和对象读写均可追踪且结果确定。
2. **Given** 外部对象存储凭据缺失，**When** 运行不依赖真实外部服务的测试，**Then** 明确的本地适配器工作且发布证据不会声称外部联调通过。

---

### User Story 4 - 可审计的合同、可观测性与发布链 (Priority: P4)

开发和验收人员可以机械验证 Feature、API、数据、测试和 UI 合同，CI 对精确提交执行构建与测试，并只交付同一受测 Artifact。

**Why this priority**: P00 是否完成必须由机器证据和精确 Artifact 证明，不能依赖聊天或手工报告。

**Independent Test**: 对精确提交运行合同、日志/指标/追踪、CI 和发布验证，核对 Build Info、测试记录、APK 哈希和桌面镜像来源一致。

**Acceptance Scenarios**:

1. **Given** 任一 Feature 映射、API 或测试证据漂移，**When** 运行合同门禁，**Then** 发布被拒绝并指出具体 ID。
2. **Given** 所有门禁通过，**When** 触发 P00 发布，**Then** 精确 CI Artifact 被验证、下载并与桌面交付的 SHA-256 一致。

---

### User Story 5 - 受控的上游能力与 Web 管理入口 (Priority: P5)

管理员可以查看真实或明确未配置的 Sub2API 能力探测结果和秘密状态，通过受 RBAC 保护的后台入口管理基础配置；开发者可以进入独立的开发者门户壳。

**Why this priority**: 能力与秘密边界必须在后续模型、文件和开发者 API 功能开始前被证明，但外部凭据缺失不能伪装成成功。

**Independent Test**: 在配置和未配置外部凭据两种模式下运行能力探测，验证后台路由权限、审计记录、错误恢复状态和开发者门户入口。

**Acceptance Scenarios**:

1. **Given** 合法的外部配置可用，**When** 管理员执行文本、流式、视觉、文件和图片能力探测，**Then** 每项结果包含时间、能力、延迟、错误和来源证据。
2. **Given** 外部配置缺失或用户无权限，**When** 访问探测/秘密/后台路由，**Then** 页面显示明确可恢复状态且不泄露 Secret 或资源存在性。

### Edge Cases

- 空仓库、远端无默认分支或本机缺少重型工具时，引导流程仍能在轻量控制端恢复到当前 Work Packet。
- 同一任务或迁移被重复提交时，系统不得产生重复副作用。
- PostgreSQL、Redis、NATS、R2 或 Sub2API 超时/不可用时，健康状态和错误码必须可区分且不能无限 Loading。
- 深色模式、320dp/384dp/411dp 宽度和 1.3 字体缩放下，Android 壳不得出现文字遮挡或触控区小于合同值。
- CI 返回的提交、Artifact、版本或哈希与请求不一致时，发布必须失败。
- 公开仓库扫描发现疑似 Secret、服务器详情或签名材料时，提交和发布必须停止。

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001 / P00-001**: 系统 MUST 提供唯一、可机械验证的仓库模块边界和所有权声明。
- **FR-002 / P00-002**: 系统 MUST 提供 base、local、staging、production 配置分层，只提交公开值和 Secret 引用。
- **FR-003 / P00-003**: Android 客户端 MUST 提供可构建、可启动并带稳定测试标识的 Compose 应用壳。
- **FR-004 / P00-004**: Android 客户端 MUST 实现 `YL-DS-1.2.0` 的浅色/深色语义 Token，页面不得引入未登记视觉值。
- **FR-005 / P00-005**: Android 客户端 MUST 提供首页、工作、发现、我的四栏导航、可恢复状态和进程重建后的选中态保存。
- **FR-006 / P00-006**: 平台 MUST 提供 Core API 进程、版本化健康/就绪接口和稳定错误封装。
- **FR-007 / P00-007**: 平台 MUST 提供隔离供应商细节的 AI Runtime 进程与能力接口。
- **FR-008 / P00-008**: 平台 MUST 提供独立的 Developer Gateway 进程与公开 API 边界。
- **FR-009 / P00-009**: 平台 MUST 提供 Worker、幂等任务入口、有限重试和失败归档路径。
- **FR-010 / P00-010**: 平台 MUST 提供 Scheduler、可测试的分布式锁和重复执行保护。
- **FR-011 / P00-011**: 平台 MUST 提供 PostgreSQL 迁移、版本记录、唯一约束和可验证回滚策略。
- **FR-012 / P00-012**: 平台 MUST 提供 Redis 缓存与限流适配器、超时和降级状态。
- **FR-013 / P00-013**: 平台 MUST 提供 NATS JetStream 事件发布/消费合同、确认和重复消息处理。
- **FR-014 / P00-014**: 平台 MUST 提供 Cloudflare R2 兼容对象存储适配器及明确的本地测试实现。
- **FR-015 / P00-015**: 仓库 MUST 提供唯一 OpenAPI 权威合同和确定性生成/漂移校验。
- **FR-016 / P00-016**: 仓库 MUST 机械校验所有 P00 Feature 与 API、数据、页面、测试和证据的追踪关系。
- **FR-017 / P00-017**: 所有运行进程 MUST 输出结构化日志、指标和跨进程追踪关联信息，且自动脱敏。
- **FR-018 / P00-018**: GitHub Actions MUST 对精确提交执行合同、后端、Web、Android、Secret 和 Artifact 门禁。
- **FR-019 / P00-019**: 发布流程 MUST 仅交付精确受测 CI Artifact，并生成版本、测试、部署、验收、截图和 SHA-256 证据。
- **FR-020 / P00-020**: 能力探测 MUST 对 Sub2API 文本与流式能力给出真实结果或明确未配置状态。
- **FR-021 / P00-021**: 能力探测 MUST 对 Sub2API 视觉、文件与图片能力给出真实结果或明确未配置状态。
- **FR-022 / P00-022**: Secret MUST 只存在于本地忽略目录、CI Environment 或服务器秘密存储中，仓库只保留示例和引用。
- **FR-023 / P00-023**: 管理后台 MUST 提供 Vue 应用壳、RBAC 路由、权限拒绝和审计入口。
- **FR-024 / P00-024**: 开发者中心 MUST 提供与管理后台隔离的应用壳、加载/空/错误/成功状态和稳定路由。
- **FR-025**: 每个 Work Packet MUST 按控制器选定顺序完成，绑定 Feature 有最终状态、真实证据和测试后才能关闭。
- **FR-026**: P00 发布后 MUST 停在项目所有者验收点；未经所有者批准不得关闭正式版本或推进 P01。

### Key Entities

- **Runtime Process**: 服务身份、版本、健康、就绪和依赖状态。
- **Configuration Layer**: 环境名、公开配置、功能开关、Secret 引用和版本。
- **Migration Record**: 迁移版本、校验和、应用时间和结果。
- **Job Envelope**: 任务 ID、幂等键、状态、重试次数和失败原因。
- **Capability Probe Result**: 能力类型、目标通道、开始/结束时间、结果、延迟和脱敏错误。
- **Release Evidence**: 阶段、版本、Commit、CI Run、Artifact、APK SHA-256、测试和视觉证据。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 24 个 P00 Feature ID 在阶段发布前 100% 具有最终状态及至少一条可验证证据或明确外部阻塞原因。
- **SC-002**: 从干净检出到 CI 产出 P00 APK 的流程无需本机 Java、Android SDK、Go、Node 或 Docker。
- **SC-003**: Android 四个主入口在受测设备上均可一次点击到达，选中态正确，所有合同状态无无限 Loading。
- **SC-004**: P00 绑定的每个适用 UI State 都有对应真实截图记录，关键几何误差为 0dp/0px，未批准差异数为 0。
- **SC-005**: 重复任务、重复事件和 Scheduler 锁竞争测试产生的重复业务副作用为 0。
- **SC-006**: 公开仓库安全扫描对 Secret、私钥、签名材料和服务器详情的未解决发现数为 0。
- **SC-007**: 桌面 APK 与受测 CI Artifact 的 Commit、版本和 SHA-256 匹配率为 100%。
- **SC-008**: P00 交付完成后，运行状态仍为 P00 等待所有者验收，P01 的实现变更数为 0。

## Assumptions

- 项目所有者已确认 GitHub 仓库公开，公开本身不是阻塞条件。
- 本机是轻量控制端；构建、模拟器和视觉回归由 GitHub Actions 完成，集成环境由既有服务器承担。
- 缺少外部凭据时只允许明确的本地适配器、Feature Flag 或 `BLOCKED_EXTERNAL`，不得生成静态成功结果。
- P00 版本和 Android 标识只以 `contracts/release-version-matrix.yaml` 为准。
- 本规格不改变 `contracts/feature-map.yaml`、Work Packet 定义或 P00 之后的任何范围。
