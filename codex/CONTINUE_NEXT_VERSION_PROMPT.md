# “继续开发下个版本”的正确解释

先执行 `./ylven.ps1 resume`：
- current Work Packet 未关闭：继续该工作包；
- 阶段 READY_FOR_RELEASE：完成 CI 和项目所有者验收；
- release ledger 已关闭且状态已推进：开始新阶段的第一个 Work Packet；
- 项目 DONE：停止并要求批准新的版本计划。

聊天文字不能覆盖仓库状态。


## V1.6 mandatory state recovery

Before planning or editing, run `.\ylven.ps1 resume`. Use only `CURRENT_PHASE.yaml`, `CURRENT_WORK_PACKET.yaml`, `status/WORK_PACKET_STATUS.yaml`, the hash-chained Work Packet history and the Release Ledger. Never infer “next version” from this chat. The public empty repository and missing `.git` are Bootstrap cases, not blockers. The owner computer is a thin control client; missing Java, Go, Gradle, ADB, Android SDK, Docker or Node.js is not a blocker.
