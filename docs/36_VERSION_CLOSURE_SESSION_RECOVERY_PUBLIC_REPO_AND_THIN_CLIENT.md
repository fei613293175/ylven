# 36. 版本关闭、会话恢复、公开仓库与轻量本机执行合同

## 1. 目标

本合同解决四个长期开发风险：新会话不知道做到哪里；代码写完但版本没有正式关闭；公开仓库被错误当作阻塞；项目所有者电脑被要求安装完整 Android、后端和容器工具链。

## 2. 唯一运行状态

仓库只允许以下运行状态来源：

```text
CURRENT_PHASE.yaml
CURRENT_WORK_PACKET.yaml
status/WORK_PACKET_STATUS.yaml
status/WORK_PACKET_HISTORY.jsonl
status/RELEASE_LEDGER.jsonl
status/PROJECT_STATE_SUMMARY.md
```

`contracts/work-packet-map.yaml` 只描述静态范围，不保存运行状态。禁止再建立 `contracts/CURRENT_PHASE.yaml`、`contracts/CURRENT_WORK_PACKET.yaml`、第二份阶段历史或第二份版本总账。

## 3. 每个 Work Packet 的正式关闭

工作包不是在 Codex 口头说“完成”时关闭。必须依次满足：

1. 当前工作包由状态控制器选定；
2. 所有绑定 Feature ID 已达到最终状态并有证据；
3. 代码、测试、合同和 Feature 证据已提交，工作树干净；
4. 执行 `close-packet`；
5. 控制器追加带哈希链的历史记录；
6. 更新 `WORK_PACKET_STATUS.yaml`；
7. 自动选择同阶段第一个未关闭工作包；
8. 创建专用状态提交并尝试推送。

工作包历史只追加、不覆盖。记录至少包含阶段、工作包、结果、实现 Commit、Feature 列表、证据统计、时间、前一条记录哈希和本条记录哈希。

## 4. 每个 APK 版本的正式关闭

当一个阶段全部工作包关闭后，阶段进入 `READY_FOR_RELEASE`，此时禁止开始下一阶段。随后必须完成：

```text
GitHub Actions真实构建
→ 模拟器功能和视觉验收
→ 精确CI Artifact桌面交付
→ 项目所有者真机和后台验收
→ OWNER_ACCEPTANCE.md = APPROVED
→ close-release
```

`close-release` 校验版本号、Commit、CI Run、Artifact、APK SHA-256 和 Owner 批准，创建或核验不可变 Git Tag，追加 `RELEASE_LEDGER.jsonl`，再推进到下一阶段第一个工作包。

## 5. 新 Codex 对话恢复

任何新会话的第一条执行命令必须是：

```powershell
.\ylven.ps1 resume
```

Codex 必须按输出继续：

- 当前工作包未关闭：继续当前工作包；
- 当前阶段 `READY_FOR_RELEASE`：只完成发布与 Owner 验收，不进入下一阶段；
- 当前版本已进入 Release Ledger：从推进后的阶段和工作包继续；
- 项目 `DONE`：不得自行创造下一版本。

聊天中的“继续下个版本”只是意图，不是状态事实。

## 6. 公开仓库

仓库公开是项目所有者确认的事实，不得再次要求改为 Private。公开仓库的代价是更严格的 Secret 门禁：提交和推送前必须扫描密钥、密码、服务器信息、签名材料和一次性管理员凭据。只有引用、示例和 `.env.example` 可以提交。

空远程仓库、没有默认分支和本地没有 `.git` 都属于首次 Bootstrap 场景，不是产品开发阻塞。Bootstrap 应初始化 `main`、绑定唯一 `origin`、完成安全扫描、创建首次提交和首次推送。

## 7. 轻量本机

项目所有者电脑只承担仓库控制、Spec Kit、Codex、状态推进、CI 调度、Artifact 下载和桌面交付。默认只要求 Git、PowerShell、uv、Spec Kit、OpenSSH；GitHub CLI 在需要调度/下载 CI 时使用。

以下工具缺失不得阻塞开发：Java、Go、Gradle、ADB、Android SDK、Docker、Node.js、PostgreSQL、Redis、NATS。Android 构建、模拟器和视觉验收在 GitHub Actions；后端、Web 和集成测试优先在 GitHub Actions；staging 运行在已连接的线上服务器。服务器首次操作必须只读盘点，不允许盲目升级宿主机或直接编辑未提交源码。

## 8. 每个阶段和每个小版本的重复规则

每份 `phases/Pxx_*.md` 和 `work-packets/Pxx-Wxx.md` 都必须重复：当前版本号、当前/下一工作包、版本关闭条件、新会话恢复命令、公开仓库 Secret 门禁、本机轻量执行边界、CI 精确 Artifact 交付要求。此重复是有意的上下文恢复机制，不得删除为“去重”。
