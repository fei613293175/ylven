# YLVEN 最终版本发布、域名、真机验收、桌面交付与快速执行合同

## 1. 唯一权威链路

```text
contracts/release-version-matrix.yaml
→ contracts/android-phase-acceptance.yaml
→ 干净的当前 Commit 生成 Git archive
→ 连接的线上服务器共享构建队列
→ 服务器 Gradle 构建与来源证明
→ 本机下载并复核 SHA-256
→ 项目所有者物理 Android 手机共享 FIFO 队列
→ 真机功能、日志和逐状态视觉验收
→ scripts/35_DELIVER_ANDROID_RELEASE.ps1
→ 本机桌面 YLVEN-Releases/<version>/
→ 项目所有者最终验收
```

GitHub Actions、Android Emulator 和任何其他模拟器不得用于当前 APK 构建或测试。任何其他文档只可解释上述链路，不得创建第二套版本号、域名或发布门禁。

## 2. 固定仓库

- 仓库：`https://github.com/fei613293175/ylven.git`
- remote：`origin`
- 默认分支：`main`
- 阶段分支：`phase/Pxx-*`
- Android applicationId：`cc.orbexa.ylven`
- 禁止直接向 `main` 推送功能代码，禁止 force push，禁止提交 Secret、服务器标识或签名材料。

## 3. Owner-facing Android 版本

| 阶段 | versionName | versionCode | 主要交付 |
|---|---:|---:|---|
| P00 | `1.0.0` | `1000000` | 工程基座、能力验证与可运行壳 |
| P01 | `1.1.0` | `1010000` | 身份后端、邮箱认证与管理后台 |
| P02 | `1.2.0` | `1020000` | Android 注册登录、设备安全与首轮真实 APK |
| P03 | `1.3.0` | `1030000` | 会话、流式聊天与消息 |
| P04 | `1.4.0` | `1040000` | 多模型、推理路由与回答对比 |
| P05 | `1.5.0` | `1050000` | 文件、视觉与文档处理 |
| P06 | `1.6.0` | `1060000` | 项目、知识检索与引用 |
| P07 | `1.7.0` | `1070000` | 图片工作台、任务与作品版本 |
| P08 | `1.8.0` | `1080000` | PPT 工作台、Deck JSON 与渲染 |
| P09 | `1.9.0` | `1090000` | 工作、发现、我的与产品整合 |
| P10 | `1.10.0` | `1100000` | 套餐、钱包、账本与 Sub2API 权益投影 |
| P11 | `1.11.0` | `1110000` | 开发者中心、公共 API、API Key 与 SSO |
| P12 | `1.12.0` | `1120000` | 高并发、安全、可观测性与运维 |
| P13 | `1.13.0` | `1130000` | 最终回归、官方 API 迁移验证与商业验收 |

同一 applicationId、同一正式签名证书和单调递增 versionCode 是覆盖更新安装的硬条件。P01 起必须先安装上一 owner release，再使用 `adb install -r` 安装本版，并验证数据标记、数据库迁移和登录态恢复。不得卸载或清除数据绕过签名/迁移失败。

## 4. 线上服务器构建

每个 owner-facing APK 必须从当前干净 Git Commit 创建可校验的 Git archive，在已连接线上服务器的跨项目共享构建队列中执行。服务器使用隔离工具链，不盲目升级宿主机；至少运行 Android 单元测试、Lint、APK 和 instrumentation APK 构建。

服务器输出必须记录 Commit SHA、源 archive SHA-256、工具链版本、构建队列运行标识、applicationId、versionName、versionCode、签名证书摘要、两个 APK 的 SHA-256 和测试/Lint 结果。本机下载后逐文件复核 SHA-256。服务器别名、地址、目录、凭据、keystore 和密码只能存放在忽略目录或环境变量中。

## 5. 物理手机验收

APK 只能在项目所有者本机连接的物理 Android 手机测试：

1. `adb devices -l` 中所选序列号状态严格为 `device`，并验证不是 qemu/模拟器；
2. 设备不可用立即停止，禁止回退模拟器；多台合格真机在线时优先选择空闲设备，全部占用时选择队列最短的 `%USERPROFILE%/.codex/android-device-queue/<serial>/` 共享 FIFO 持续等待；允许显式指定序列号且不得中断其他项目；
3. 持锁覆盖上一版安装、数据/登录标记、`adb install -r`、启动、全部测试、截图和日志采集；
4. 对本阶段每个页面真实执行点击、输入、返回、滚动和关键成功/失败/取消/恢复流程；
5. 每个 Android Interaction ID 必须有 testTag/semantics 和真机测试；
6. 每个 Page State ID 必须由物理手机生成截图并与 APPROVED 效果图逐页比较；
7. 清空并保存 logcat，检查 crash、ANR、native crash、系统退出原因和无法解释的异常；
8. 记录设备、APK、Commit、测试路径、截图、问题和结果。

确定性 Fake 网关可验证 UI 交互，但不能冒充真实 staging API、持久化和审计联调。任一功能、视觉、日志或真实业务流程失败都禁止交付。

## 6. 管理后台交付

- P00-P13 每个版本都必须由 Codex 亲自打开公共管理后台并完成当前版本对应菜单和功能的真实操作；不能因为版本没有新增后台菜单而跳过。
- 每版必须验证公共 URL、交付账号、当前版本功能、真实 API 回读、持久化结果和审计记录。仅健康检查、静态页面、mock 或演示成功均不算通过。
- 后台打不开、登录失败、对应功能不可用或真实数据/审计证据缺失时，版本保持未交付。
- 管理员邮箱、密码和一次性凭据只能保存在本机或服务器私密变量及项目所有者桌面交付目录，不得进入 Git、构建来源证明、日志或聊天。

## 7. 域名提醒

域名清单和交付阶段只以 `contracts/domain-delivery-map.yaml` 为准。Codex 在相关阶段开始和发布前必须生成 `DNS_ACTION_REQUIRED.md`，列出准确记录类型、目标值、Cloudflare 代理状态、TLS 和健康检查；没有真实目标值时禁止猜测。

## 8. 本机桌面交付

真机全部门禁通过后，只能交付同一 Commit、同一服务器 SHA-256、同一真机受测 APK：

```text
~/Desktop/YLVEN-Releases/<version>/
├── YLVEN-<version>-<phase>.apk
├── 原功能清单.md
├── 功能完成对比清单.md
├── 完整测试清单.md
├── 自动化测试报告.md
├── 截图索引.csv
├── 截图/
├── 视觉差异报告.md
├── 部署证据.md
├── 域名DNS状态.md
├── 构建信息.json
├── 服务器构建来源证明.json
├── 本机下载校验证明.json
├── 真机验收证据.json
├── 真机日志审查.md
├── 管理后台实测证据.md
└── 校验文件_SHA256.txt
```

## 9. 快速开发模式

- 每工作包只进行一次完整预检；
- 开发中运行受影响测试，阶段发布才运行一次完整服务器构建和真机验收；
- 连续两轮没有实际代码、测试或可运行成果，停止泛化分析并处理具体阻塞；
- Spec Kit 每阶段规划一次，普通 Work Packet 直接实现；
- 不重复生成内容相同的检查报告；
- 以真实代码、真实测试、真实部署、真实 APK 和可复核证据为主要产出。
