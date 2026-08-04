# 项目所有者批准后关闭当前正式版本并恢复下一阶段

1. 先运行 `.\ylven.ps1 resume`，确认当前阶段为 `READY_FOR_RELEASE`，不要根据聊天猜测阶段。
2. 确认 `dist/releases/<phase>/OWNER_ACCEPTANCE.md` 存在精确行 `- Result: APPROVED`，且当前阶段发布校验仍通过。
3. 运行 `.\ylven.ps1 close-release -Phase <当前阶段>`。控制器必须先校验精确 CI Artifact、Commit、版本和 SHA-256，创建/校验不可变 Tag，追加 `RELEASE_LEDGER.jsonl`，提交状态并推进到相邻阶段第一个 Work Packet。
4. 再运行 `.\ylven.ps1 resume`。只开发新输出中的唯一 Work Packet。新阶段首次执行一次 Spec Kit Specify/Plan/Tasks，随后立即实现；不得重新规划整个产品，也不得手工修改状态。
