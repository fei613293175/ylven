# P00 实施状态

- 阶段：P00 — 工程基座、部署骨架与上游能力证明
- 当前状态：IN_PROGRESS（P00-W01 已关闭；P00-W02 已实现，等待控制器关闭）
- 最后更新：2026-08-05

## 已完成的纵向切片

- P00-001：仓库模块与所有权边界。
- P00-002：base/local/staging/production 递归配置分层与 Secret 引用。
- P00-003：可构建的 Compose 应用壳和稳定合同测试标识。
- P00-004：YL-DS-1.2.0 浅色/深色语义 Token 与集中尺寸 Token。
- P00-005：首页、工作、发现、我的四栏导航与保存状态恢复。
- P00-006：Core API 进程身份、健康/就绪/版本端点。
- P00-007：AI Runtime 进程身份、健康/就绪/版本端点。
- P00-008：Developer Gateway 进程身份、健康/就绪/版本端点。
- P00-009：Worker 幂等入队、有限重试、毒消息归档和取消状态。
- P00-010：Scheduler 租约锁、过期接管、单调 fencing token 和重复执行保护。

## 当前进行中

P00-W01 已通过轻量控制端合同测试和门禁并由控制器关闭。P00-W02 已完成 Go 运行时实现和确定性测试；本机未安装 Go，真实执行由新增 GitHub Actions `backend-process-tests` job 负责。

## 外部阻塞

无。缺少 DNS、R2、Turnstile、邮件或 Sub2API 凭据时，只记录真实联调阻塞，并继续可离线开发部分。

## 关键命令与证据

- `docs/evidence/P00-W01.md`
- `docs/evidence/P00-W02.md`
