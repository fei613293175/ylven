# 新会话继续开发提示（V1.6）

不要根据聊天中的“下个版本”猜测。先执行 `./ylven.ps1 resume`，读取生成的 `status/PROJECT_STATE_SUMMARY.md`，并继续其中唯一的 current Work Packet。若状态为 READY_FOR_RELEASE，则只完成 CI、交付和项目所有者验收；不得自行进入下一阶段。工作包和版本只能通过 `ylven.ps1` 关闭。


## V1.6 mandatory state recovery

Before planning or editing, run `.\ylven.ps1 resume`. Use only `CURRENT_PHASE.yaml`, `CURRENT_WORK_PACKET.yaml`, `status/WORK_PACKET_STATUS.yaml`, the hash-chained Work Packet history and the Release Ledger. Never infer “next version” from this chat. The public empty repository and missing `.git` are Bootstrap cases, not blockers. The owner computer is a thin control client; missing Java, Go, Gradle, ADB, Android SDK, Docker or Node.js is not a blocker.
