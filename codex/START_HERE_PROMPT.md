# Codex 首次启动提示（V1.6）

从当前解压目录开始，不要因为没有 `.git`、远端为空、仓库公开或本机缺少 Java/Go/Gradle/ADB/Docker/Python 而停止。

1. 读取 `PROJECT_CONTEXT.yaml`、`AGENTS.md`、`contracts/contract-authority-map.yaml`、`contracts/state-transition-contract.yaml`、`contracts/execution-environment.yaml`。
2. 在 Windows 执行 `./ylven.ps1 bootstrap`。它负责初始化 Git、绑定 `https://github.com/fei613293175/ylven.git`、首次推送公开空仓库、安装 uv 与 Spec Kit，并建立当前状态。本机不部署重型开发环境。
3. 执行 `./ylven.ps1 resume`。只允许开发输出中的 current Work Packet；首次应为 `P00-W01`。
4. 项目首次只执行一次 `$speckit-constitution`；每阶段首次执行一次 `$speckit-specify`、`$speckit-plan`、`$speckit-tasks`。随后立即实现，不重复全项目规划。
5. Android 构建、重型自动测试、staging Docker 与集成放在已连接线上服务器，通过现有 SSH 配置访问；APK 功能、日志和逐状态视觉验收只在项目所有者本机已连接的物理 Android 手机上执行。设备必须处于精确 ADB `device` 状态；多台合格真机在线时优先选择空闲设备，全部占用时选择最短共享 FIFO 排队等待，也可显式指定序列号；禁止 GitHub Actions、Android Emulator 和任何模拟器。首次服务器操作运行 `./ylven.ps1 inventory`，只读盘点。
6. 每个 Work Packet 完成后更新 Feature 状态、证据和测试，形成提交，然后运行 `./ylven.ps1 close-packet -Packet <当前ID>`。脚本自动记录并选择下一包。
7. 全部工作包关闭后运行 `./ylven.ps1 release -Phase <当前阶段>`。只交付线上服务器从精确 Commit 构建、下载后 SHA-256 一致且真机验收完整通过的 APK。项目所有者批准后运行 `./ylven.ps1 close-release -Phase <当前阶段>`，再进入下一版本。
8. 仓库公开是项目所有者确认的决策，不得要求改为 Private；但每次推送前必须通过 `scripts/40_PUBLIC_REPOSITORY_SAFETY.py`。
9. 每个循环优先产出代码、测试、迁移、可运行部署或具体阻塞；禁止反复检查未变化输入。


## V1.6 mandatory state recovery

Before planning or editing, run `.\ylven.ps1 resume`. Use only `CURRENT_PHASE.yaml`, `CURRENT_WORK_PACKET.yaml`, `status/WORK_PACKET_STATUS.yaml`, the hash-chained Work Packet history and the Release Ledger. Never infer “next version” from this chat. The public empty repository and missing `.git` are Bootstrap cases, not blockers. The owner computer is a thin control client; missing Java, Go, Gradle, ADB, Android SDK, Docker or Node.js is not a blocker.
