# P03 实施状态

- 阶段：P03 - 核心会话、流式聊天与消息数据模型
- 当前状态：DOING（W08 非视觉后端实现已补齐；W07 v11 真机视觉门禁失败，P03 不得发布或关闭）。
- 最后更新：2026-08-15

## 已完成事实

P03-W01 至 P03-W05 已由控制器关闭。最终生产 APK SHA-256 为 `48bd84d7c156ad5e3658e0372205359f751eddf0bf74fa024f34d6ae47158383`。Xiaomi `24094RAD4C` 和 vivo X21 的物理设备证据覆盖批准的 P02 -> P03 一次性签名迁移、93 个原生尺寸状态截图、真实 staging 注册与会话、真实上游完成态、导出、重命名、归档、临时会话、日志和崩溃/ANR 审查；两台设备均未使用显示尺寸或密度覆盖。

公共管理后台已通过真实管理员会话读取 105 个持久化会话、71 条聊天指标、完成态 run、消息和 1,570 条审计事件；未授权请求返回 403。实测发现并修复 `/internal/metrics/chat` 的 Nginx 精确代理路由，修复 Commit 为 `920267d`。

P03 的实现覆盖会话实体、搜索与回收、provider-backed 异步 run/SSE/取消、消息持久化和元数据、Room 缓存、安全 Markdown、导出/反馈/重答/临时会话、系统语音、草稿、所有权隔离、聊天指标与后台诊断。

## 明确待决项

1. ADR-010 已由项目所有者批准：P03 保持“移入项目”禁用，P06-007 实现后再启用。
2. ADR-011 已由项目所有者批准：P03 临时会话采用标题字段，不提前实现 P04/P06 设置。
3. 上一会话 v11 真机视觉门禁实际失败：原始像素 `5/8`、结构 `0/8`、平均 mismatch `0.01980333`；因此不得把旧的 `READY_FOR_RELEASE` 或 W08 代码实现解释为发布通过。
4. W08 已新增上下文诊断、共享游标恢复证据、AI Runtime 健康心跳和多实例测试结果持久化；需在用户连接的服务器上执行 Go/数据库/CI 验证，当前未在本机运行构建或测试。
5. 仍需从包含最终证据更新的精确 Git Commit 在线上服务器生成最终 APK 证明；若 APK SHA-256 不变，仅回归受影响的后台指标路由，不重复完整真机矩阵。
6. 最终交付包通过发布校验后，由项目所有者将 `所有者验收.md` 标记为 `APPROVED`，再执行 `close-release`。

在最终发布校验和所有者批准完成前，不执行 `close-release`，也不启动 P04。

<!-- CO_P03_001_IMPLEMENTATION_STATUS -->
## P03-W08 本次增量实现

- `backend/db/migrations/0007_p03_w08_runtime_observability.sql` 新增 `service_instances`、`health_checks`、`test_runs` 持久化表及状态/游标约束。
- `GET /admin/v1/ai-runs/{runId}/context-debug` 使用 `admin.ai_runs.read`，只返回上下文组成元数据和 240 字符以内预览，不返回 provider 消息集合或隐藏思维链。
- `GET /internal/v1/health/ai-runtime` 写入实例租约与健康检查，并返回 `healthy|degraded|error` 聚合状态。
- `POST /internal/v1/ci/p03-distributed-chat` 使用 `ci` 权限和稳定 `test_run_id`/`Idempotency-Key` 记录多实例恢复结果。
- 当前工作树尚未提交；W07 视觉阻断和服务器验证缺口保持显式记录。

## CO-P03-001 未关闭补充

P03新增 P03-W06、P03-W07、P03-W08；三者全部关闭、CI与项目所有者验收通过前，P03不得 close-release，不得推进P04。
