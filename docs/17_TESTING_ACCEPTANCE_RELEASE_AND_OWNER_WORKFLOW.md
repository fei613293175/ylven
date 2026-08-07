# 17 测试、验收、发布与项目所有者工作流

## 1. 测试金字塔

### 1.1 单元测试

覆盖领域规则、推理参数映射、上下文裁剪、价格计算、额度桶、状态机、错误映射、权限和格式转换。不得只测试 getter 或框架默认行为。

### 1.2 集成测试

使用真实 PostgreSQL/Redis/NATS 容器，验证迁移、事务、Outbox、幂等、锁、队列和 API。Sub2API 通过契约 mock 和 staging 真实通道分别测试。

### 1.3 契约测试

- OpenAPI 与服务实现；
- Android 生成客户端与 API；
- YLVEN Adapter 与 Sub2API；
- Provider Adapter 与官方 API；
- Worker 输入/输出 schema；
- Feature Map 引用的接口、表和测试存在。

### 1.4 Android 测试

ViewModel/Reducer 单元测试、Compose UI 测试、网络/Room 集成、截图或语义回归和项目所有者物理 Android 手机测试。APK 构建和非设备重型测试在连接的线上服务器执行；APK 测试禁止 Android Emulator 和任何模拟器。至少覆盖小屏、常见中屏、字体放大、浅/深色、旋转/进程重建和弱网。

### 1.5 端到端测试

注册、登录、发消息、流式恢复、上传文件、识图、图片、PPT、充值测试流程、API Key 和后台操作。真实上游测试控制成本和频率；确定性 mock 只用于隔离测试，不能替代 staging 真实联调。

## 2. 测试清单不能伪造

`TESTS.md` 必须列出实际执行的命令、时间、环境、结果、失败和跳过理由。禁止写“应该通过”“已检查”而没有命令或证据。因缺少外部凭据而跳过的测试必须列入 `OWNER_ACTIONS.md` 和已知限制。

## 3. 每阶段完成条件

1. Feature Map 对应功能已实现；
2. 接口和数据库迁移存在；
3. 管理后台入口可访问；
4. 必要自动测试通过；
5. staging 健康检查通过；
6. Android staging APK 构建成功；
7. 交付目录文件完整；
8. `06_VERIFY_RELEASE.ps1` 返回 0；
9. 项目所有者完成真机/后台体验后，没有 P0/P1 缺陷；
10. 已知限制明确记录，不用占位功能冒充完成。

## 4. 自动交付目录

```text
dist/releases/Pxx/
├── YLVEN-Pxx-test.apk
├── FEATURES.md
├── TESTS.md
├── CHANGELOG.md
├── DEPLOYMENT.md
├── BUILD_INFO.json
└── SHA256SUMS.txt
```

脚本额外复制到 Windows 桌面 `YLVEN_交付/Pxx/`。仓库目录是正式来源，桌面是便捷副本。

## 5. 项目所有者验收

项目所有者只需：

1. 安装 APK；
2. 按 `APK_ACCEPTANCE_TEMPLATE.md` 的固定步骤测试；
3. 登录对应阶段后台；
4. 检查页面、配置和数据是否符合预期；
5. 对问题提供页面、操作、实际结果、预期结果和截图；
6. 确认通过后让 Codex启动下一阶段。

项目所有者不需要人工执行单元测试、生成报告或查找 APK。

## 6. 缺陷严重级别

- P0：数据泄漏、资金错误、账号接管、服务不可用、数据不可恢复；阻止发布。
- P1：核心功能无法完成、严重崩溃、重复扣费、文件丢失；阻止阶段通过。
- P2：有替代路径但明显影响使用；应在当前或紧随修复批处理。
- P3：视觉、文案、小交互；可集中精修但必须记录。

## 7. Codex 修复流程

收到真机反馈后，Codex先复现或建立可验证假设，定位 Feature ID，补回归测试，修复最小必要范围，运行相关测试，重新执行阶段发布。不得借一个 UI 问题重新规划整个项目，也不得只改文案掩盖后端错误。

## 8. 发布必需检查

- 格式、静态分析、依赖和 Secret 扫描；
- 线上服务器 Go/Android/Web 构建；
- 单元和集成测试；
- OpenAPI lint 和 breaking change 检查；
- 数据库迁移测试；
- Feature Map/阶段/测试引用校验；
- 线上服务器 APK 与 instrumentation APK 输出、下载 SHA-256 和 release contract 校验；
- 项目所有者物理手机的 ADB `device`、多设备自动择闲（全部占用时选择最短共享 FIFO 等待）、覆盖安装、启动、真机交互、截图、视觉比较和日志审查；
- 容器镜像扫描（生产阶段）。

GitHub Actions、Android Emulator 和其他模拟器不得用于本项目当前 APK 的构建或测试。

## 阶段 UI 证据固定目录

真实阶段发布前，Codex 必须从物理手机采集视觉证据并放在以下固定位置，不得临时改名或散落到桌面：

```text
build/ui-evidence/<Phase>/
├── screenshots/
│   ├── <StateID>.png
│   └── ...
└── VISUAL_DIFF_REPORT.md
```

- 每个绑定 State ID 必须恰好对应一张独立运行截图；文件名必须是 `<StateID>.png`。
- `VISUAL_DIFF_REPORT.md` 必须逐 State ID 记录已批准效果图 SHA、运行截图、设备/主题/字体缩放、差异和处理结果，并包含 `- Result: PASS`。
- `scripts/05_RELEASE_PHASE.ps1` 通过线上服务器构建后在物理手机采集并校验 `build/ui-evidence/<Phase>`；缺少任一状态、效果图未 APPROVED、SHA 不一致或存在未解释差异时，阶段发布失败。
- 桌面交付是发布脚本生成的副本，不是视觉证据的权威来源。
