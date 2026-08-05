# P01 实施状态

- 阶段：P01 — 统一身份后端、邮箱认证与管理后台基础
- 当前状态：DOING（P01-W01）
- 最后更新：2026-08-05

## 已完成的纵向切片

P01-W01 已完成 Identity Service 的邮箱规范化、注册挑战、Turnstile 外部/Mock 边界、注册 OTP 和密码哈希链路，覆盖 P01-001 至 P01-006。

## 当前进行中

P01-W01：等待 GitHub Actions 对当前提交执行 Go 单元/API 构建验证，然后关闭工作包。

## 外部阻塞

无。缺少 DNS、R2、Turnstile、邮件或 Sub2API 凭据时，只记录真实联调阻塞，并继续可离线开发部分。

## 关键命令与证据

已执行：

- `gofmt`/`go test ./...`：本机未安装 Go，未执行；本机环境按合同由 GitHub Actions 承担重型构建。
- `P01-W01` 单元测试：`backend/internal/identity/identity_test.go`，由 CI 执行。
- 数据迁移：`backend/db/migrations/0002_p01_identity.sql`，含 `auth_challenges`、`email_otps`、`users`、`password_credentials`。
- 外部依赖：Turnstile 默认 `external` 返回明确 `turnstile_unconfigured`；本地/CI 可用 `TURNSTILE_MODE=mock` 和测试令牌验证适配器，不提交任何密钥。
