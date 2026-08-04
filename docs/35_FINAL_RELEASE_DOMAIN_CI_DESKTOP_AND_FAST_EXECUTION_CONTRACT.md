# YLVEN 最终版本发布、域名、自动验收、桌面交付与快速执行合同

## 1. 唯一权威链路

```text
contracts/release-version-matrix.yaml
→ contracts/android-phase-acceptance.yaml
→ .github/workflows/android-phase-acceptance.yml
→ 当前 commit SHA 的精确 CI Artifact
→ scripts/35_DELIVER_ANDROID_RELEASE.ps1
→ 本机桌面 YLVEN-Releases/<version>/
→ 项目所有者真机与后台验收
```

任何其他文档只可解释该链路，不得创建第二套版本号、域名或发布门禁。

## 2. 固定仓库

- 仓库：`https://github.com/fei613293175/ylven.git`
- remote：`origin`
- 默认分支：`main`
- 阶段分支：`phase/Pxx-*`
- Android applicationId：`cc.orbexa.ylven`
- 禁止直接向 `main` 推送功能代码，禁止 force push，禁止提交 Secret。

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


同一 applicationId、同一正式签名证书和单调递增 versionCode 是覆盖更新安装的硬条件。P01 起，CI 必须先安装上一版再使用 `adb install -r` 安装本版，并验证数据迁移和登录态恢复。

## 4. GitHub Actions 模拟真机验收

每个 owner-facing APK 必须在固定 Android 模拟器完成：

1. 构建、单元测试和 Lint；
2. 上一版到本版的覆盖升级安装；
3. 本阶段全部可交互控件测试；
4. 全部已发布 Interaction ID 的回归点击；
5. 本阶段全部状态截图与已批准效果图比较；
6. 旧页面主要 Golden 状态回归；
7. 网络错误、服务失败、操作失败、成功、权限、额度和恢复流程；
8. 上传 APK、自动测试报告、截图索引、视觉差异、功能完成清单和 SHA-256。

“点击每一个按钮”不是人工口号：每个可交互控件必须拥有 Interaction ID、Compose testTag/semantics 和至少一个自动测试。无测试映射即发布失败。

## 5. 管理后台交付

- P00 / App 1.0.0：交付后台工程壳、健康检查和部署基础，不交付可长期使用的默认密码。
- **P01 / App 1.1.0：首次交付可访问的 `admin.orbexa.cc`、管理员邮箱和一次性密码。**
- 管理员邮箱由本机或服务器私密变量 `OWNER_ADMIN_EMAIL` 提供。
- Codex 在 staging 生成高强度一次性密码，只写入项目所有者桌面交付目录的 `ADMIN_ACCESS_ONE_TIME.txt`；不得进入 Git、GitHub Artifact、日志或聊天。
- 首次登录强制修改密码，并使一次性密码失效。

## 6. 域名提醒

域名清单和交付阶段只以 `contracts/domain-delivery-map.yaml` 为准。Codex 在相关阶段开始和发布前必须生成 `DNS_ACTION_REQUIRED.md`，列出准确记录类型、目标值、Cloudflare 代理状态、TLS 和健康检查；没有真实目标值时禁止猜测。

## 7. 本机桌面交付

通过 CI 后，Codex 必须下载**同一 commit SHA** 的精确 Artifact，禁止本地重新编译另一个 APK。目录：

```text
~/Desktop/YLVEN-Releases/<version>/
├── YLVEN-<version>-<phase>.apk
├── FEATURES_PLANNED.md
├── FEATURES_COMPLETED.md
├── OWNER_TEST_CHECKLIST.md
├── AUTOMATED_TEST_REPORT.md
├── UI_SCREENSHOT_INDEX.csv
├── VISUAL_DIFF_REPORT.md
├── DEPLOYMENT_ENDPOINTS.md
├── DOMAIN_DNS_STATUS.md
├── BUILD_INFO.json
├── CI_PROVENANCE.json
└── SHA256SUMS.txt
```

P01 额外包含本地生成且不进入 Git/CI 的 `ADMIN_ACCESS_ONE_TIME.txt`。

## 8. 项目所有者测试清单

`OWNER_TEST_CHECKLIST.md` 必须逐项说明：

- 本版新增功能；
- 每项功能的进入路径；
- 具体操作步骤；
- 正确预期；
- 需要测试的失败和恢复路径；
- 后台需要核对的字段或记录；
- 覆盖更新安装步骤；
- 已知限制；
- 通过/不通过填写位置。

## 9. 快速开发模式

- 每工作包只进行一次完整预检；
- 开发中运行受影响测试，阶段发布才运行一次完整验收；
- 连续两轮没有实际代码、测试或可运行成果，停止泛化分析并处理具体阻塞；
- Spec Kit 每阶段规划一次，普通 Work Packet 直接实现；
- 不重复生成内容相同的检查报告；
- 以真实代码、真实测试、真实部署、真实 APK 为主要产出。
