# P00 功能交付对比清单

## 阅读说明

P00 是“工程基座、部署骨架与上游能力证明”版本，不是登录、聊天、文件工作台或商业化版本。对比基线是 P00 开发前仓库中没有可验证、可安装的本阶段实现；“已完成”只表示 P00 范围内的实现和 CI 证据已存在，不表示后续阶段功能提前交付。

| Feature | 交付前基线 | P00 交付结果 | 证据 | Owner 验收重点 |
|---|---|---|---|---|
| P00-001 | 无稳定模块边界 | 已完成：仓库模块和所有权边界 | `repository/modules.yaml`; W01 | 检查仓库结构和模块归属 |
| P00-002 | 无分层环境合同 | 已完成：base/local/staging/production 配置分层和 Secret 引用 | `config/schema.yaml`; W01 | 确认示例不含真实 Secret |
| P00-003 | 无可安装 Android 壳 | 已完成：Compose 应用壳、Manifest 和稳定测试标识 | `android/app`; W01; Android CI | 安装 APK 并启动 |
| P00-004 | 无统一视觉 Token | 已完成：YL-DS-1.2.0 色彩、尺寸、字体和主题 Token | `android/app/.../ui/theme`; W01 | 检查浅色/深色和布局稳定性 |
| P00-005 | 无稳定主导航 | 已完成：首页、工作、发现、我的四栏和状态恢复 | `YlvenApp.kt`; Android UI tests; W01 | 切换四栏、旋转/重启后检查状态 |
| P00-006 | 无 Core API 进程合同 | 已完成：身份、健康、就绪、版本端点 | `backend/cmd/core-api`; W02 | 用 curl 检查端点和版本 |
| P00-007 | 无 AI Runtime 进程合同 | 已完成：AI Runtime 壳和端点 | `backend/cmd/ai-runtime`; W02 | 检查进程隔离和端点 |
| P00-008 | 无 Developer Gateway 合同 | 已完成：Gateway 壳和端点 | `backend/cmd/developer-gateway`; W02 | 检查端点和错误边界 |
| P00-009 | 无可靠 Worker 语义 | 已完成：幂等、有限重试、毒消息归档、取消/停机 | `backend/internal/platform/jobs`; W02 | 重复任务只执行一次 |
| P00-010 | 无 Scheduler 租约保护 | 已完成：租约、过期接管、fencing token、重复执行保护 | `backend/internal/platform/scheduling`; W02 | 并发租约和旧 token 被拒 |
| P00-011 | 无迁移保护 | 已完成：PostgreSQL checksum、顺序和回滚边界 | `backend/db/migrations`; W03 | 执行迁移校验和回滚演练 |
| P00-012 | 无缓存策略 | 已完成：Redis TTL 和限流边界 | `backend/internal/platform/cache`; W03 | 检查 TTL、限流和恢复 |
| P00-013 | 无事件去重 | 已完成：事件总线去重和稳定语义 | `backend/internal/platform/events`; W03 | 重放同一事件不重复生效 |
| P00-014 | 无对象存储边界 | 已完成：R2/LocalFS adapter 边界 | `backend/internal/platform/storage`; W03 | 未配置 R2 时必须明确失败 |
| P00-015 | 无 OpenAPI 漂移门禁 | 已完成：Canonical OpenAPI 和 hash 漂移检查 | `contracts/openapi`; W03 | 运行漂移脚本 |
| P00-016 | 无 Feature 追踪链 | 已完成：Feature、任务、代码、测试、证据可追溯 | `scripts/44_VALIDATE_P00_TRACEABILITY.py`; W04 | 按 Feature 查证据 |
| P00-017 | 无可审计观测 | 已完成：脱敏日志、指标和 `/internal/metrics` | `backend/internal/platform/telemetry`; W04 | 检查敏感字段不出现在日志 |
| P00-018 | 无精确 CI 门禁 | 已完成：合同、后端、Web、Android、安全和 Artifact 门禁 | `.github/workflows`; W04/W05 | 只接受同一 SHA 的 run |
| P00-019 | 无精确发布控制 | 已完成：版本校验、Artifact provenance、SHA 和桌面交付脚本 | `scripts/05`, `33`, `34`, `35`; W04 | 核对 APK SHA |
| P00-020 | 无 Sub2API 文本/流式证明 | 已完成：文本和流式探测，未配置时明确 `UNCONFIGURED` | `backend/.../sub2api`; W04 | 不把未配置当成功 |
| P00-021 | 无 Sub2API 视觉/文件证明 | 已完成：vision/file/image 探测和未配置状态 | `backend/.../sub2api/probe.go`; W05 | 检查 503/UNCONFIGURED 边界 |
| P00-022 | 无公开仓库秘密边界 | 已完成：env 示例、忽略规则和 Secret 扫描 | `.env.example`; `scripts/45`; W05 | 扫描不得发现凭据 |
| P00-023 | 无独立 Admin 壳 | 已完成：Vue Admin、RBAC、拒绝态、审计入口和 P00 状态 | `web/admin`; W05 | 无权限路由必须拒绝 |
| P00-024 | 无独立 Developer Portal 壳 | 已完成：Vue Developer Portal 的加载/空/错/成功态 | `web/developer`; W05 | 检查四类状态和导航 |

## 明确未交付

- P00 不包含登录、注册、邮箱验证码、会话聊天、模型路由、文件工作台、图片/PPT、钱包计费或真实 Sub2API/R2/DNS 凭据接入；这些属于 P01 及以后阶段。
- 当前 APK 的 Android 自动验收通过；P00 shell 尚未产出与全部 mockup state 一一对应的运行时截图，`VISUAL_DIFF_REPORT.md` 已明确记录该限制，不能视为视觉验收通过。
