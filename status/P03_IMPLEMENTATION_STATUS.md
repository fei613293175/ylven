# P03 实施状态

- 阶段：P03 — 核心会话、流式聊天与消息数据模型
- 当前状态：READY_FOR_RELEASE（P03-W01 至 P03-W05 已由控制器关闭；等待线上服务器精确 Commit 构建、SHA-256 下载复核、物理真机、部署证据与所有者验收）
- 最后更新：2026-08-08

## 已完成的纵向切片

P03-W01 至 P03-W05 已实现会话实体、搜索与回收、provider-backed 异步 run/SSE/取消、消息持久化与元数据、Room 缓存、安全 Markdown、导出/反馈/重答/临时会话、系统语音、草稿、所有权隔离、聊天指标与后台诊断。补充修复了返回时草稿保存、单条消息导出路由、SSE 游标恢复/重试、窄屏消息操作栏和 29 个生产 Interaction ID；Android APK 仍待线上服务器从最终精确 Commit 编译。

## 发布前待完成

当前工作包：无。控制器记录为 `PHASE_COMPLETE`。必须提交当前实现，从该精确 Commit 在线上服务器构建，下载并复核 APK SHA-256；随后取得共享 FIFO 真机锁，完成 P02 到 P03 覆盖安装、真实 staging/交互/日志/93 状态视觉验收、部署、后台数据回读和项目所有者验收。

## 签名迁移

P02 owner APK 的签名证书 SHA-256 为 `1785147b6b61077d2c62ad9561036ece38f1290820e63301307310bc234965ad`；其 GitHub 托管 runner 临时 debug 私钥不可恢复。项目所有者于 2026-08-08 明确批准 P02 -> P03 一次性签名迁移。迁移合同为 `contracts/signing-migrations/P03.properties`：保持包名，卸载 P02 后安装 P03，如实记录旧本地数据和登录态不保留；P03 新证书 `dd5b4b6aea99358ff3e0bd8c64c1359b6e68247df36fd86c64aa0c8138088049` 是 P04 及以后版本的固定升级基线。

## 关键命令与证据

本机 Web Admin `npm test`、`npm run build`：PASS。后端 `go test ./...`：PASS。合同 `scripts/07_VALIDATE_CONTRACTS.py --phase P03`、视觉门禁 `scripts/20_CHECK_UI_VISUAL_GATE.py --packet P03-W05`、状态连续性：PASS。`scripts/31_VALIDATE_INTERACTION_TEST_COVERAGE.py --phase P03`：29/29 PASS。Android 必须由线上服务器从最终精确 Commit 构建，并只在项目所有者物理真机验收。
