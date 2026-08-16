# P04 实施状态

- 阶段：P04 — 多模型目录、推理强度、路由与对比回答
- 当前状态：P04-W01 已按项目所有者授权接受并停止本轮开发（P04-001 至 P04-006 已实现，服务器构建和真机功能回归通过；严格 Android 视觉门禁原始结果保留为 FAIL，未修改阈值或比较逻辑；P04-W02 未开始）。
- 最后更新：2026-08-16

## 已完成的纵向切片

P04-W01 已完成供应商/模型目录、能力探测、推理档位和上游参数映射的纵向切片。后端模型目录与持久化、管理端目录投影、Android 生产选择器和 API 合同均已实现。

- Feature：`P04-001` 至 `P04-006`。
- Android：`1.4.0` / `versionCode 1040000`；本轮服务器构建 APK SHA-256 为 `8dd37afc19d610261ace9e515abb898594cfe8a7ebae370ea8dc09545db59f81`。
- 真机：Redmi `24094RAD4C`（`CMZDEENVAESS89UC`，Android API 35）；`P03RealDeviceFlowTest` 3/3（47.127s）及 `P03ConversationStateUiTest` 1/1（69.775s）通过，并已审查本轮 logcat，无 `FATAL EXCEPTION`、应用进程崩溃或 ANR。
- 证据目录：`.ylven-local/p04-w01-contract-final-evidence/cmzdeenv-20260816/`，含安装记录、12 个 P04-W01 原始真机状态截图、APK/截图哈希、instrumentation 输出、logcat、ANR 审查及严格视觉报告。

## 本轮结论

项目所有者于 2026-08-16 明确接受本轮视觉差异并授权结束开发目标。该决定是针对本轮增量的验收例外，不等同于严格视觉门禁通过；原始截图、哈希和比较报告继续保留。

## 验收例外与控制器状态

P04-W01 功能实现不存在外部依赖阻塞，但严格 Android 视觉验收未通过：`YL-A-033` 和 `YL-A-034` 的 12 个要求状态均为 `REVIEW`，相似度为 `0.95639`–`0.97650`，像素失配为 `0.02350`–`0.04361`，未达到 `similarity >= 0.985` 且 `mismatch <= 0.005`。证据为 `.ylven-local/p04-w01-contract-final-evidence/cmzdeenv-20260816/P04-W01-strict-visual-diff.md`。本轮按项目所有者明确授权接受该差异；严格报告仍标记为 `FAIL`。仓库状态控制器仍显示 `P03/P03-W07/DOING`，因此未绕过控制器直接伪造 `P04-W01` 的机器关闭记录，也未开始 P04-W02。

## 关键命令与证据

本轮 Android `testDebugUnitTest`、`lintDebug`、`assembleDebug` 和 `assembleDebugAndroidTest` 均在连接服务器完成并通过（3m12s）；本轮真机安装、状态采集和交互回归均通过。严格视觉比较在连接服务器执行并保留失败结果；本轮交付结论依据项目所有者的明确验收例外。
