# V1.6 升级说明

- 增加 `CURRENT_WORK_PACKET.yaml`、唯一运行状态表、哈希链工作包历史和正式版本发布总账。
- 每个 Work Packet 关闭后自动创建状态 Commit、尝试推送并选择下一包；每个正式版本关闭后记录 Tag、CI Run、Artifact 与 APK SHA-256。
- 新 Codex 会话固定运行 `.\ylven.ps1 resume`，不再根据聊天猜测版本。
- 仓库 Public 是 Owner 明确决定；公开仓库安全扫描替代错误的 Private 阻塞。
- 缺少 `.git`、空远程和无默认分支由 Bootstrap 自动初始化。
- Owner 电脑改为轻量控制端；Java、Go、Gradle、ADB、Android SDK、Docker、Node 和系统 Python不再是本机阻塞。
- `PROJECT_CONTEXT.yaml` 与包版本统一为 `1.6.0`。
