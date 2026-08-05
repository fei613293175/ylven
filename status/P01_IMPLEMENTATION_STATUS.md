# P01 实施状态

- 阶段：P01 — 统一身份后端、邮箱认证与管理后台基础
- 当前工作包：P01-W05
- 状态：实现完成，等待工作包 CI 验收与阶段 release

## 已完成纵向切片

- P01-W01 / P01-001..006：邮箱规范化、注册挑战、Turnstile 边界、OTP、密码哈希。
- P01-W02 / P01-007..012：登录挑战、登录 OTP、防枚举、访问令牌和刷新轮换。
- P01-W03 / P01-013..018：退出、设备会话、认证限流和审计。
- P01-W04 / P01-019..023：邮件/Turnstile/OTP 设置和脱敏用户查询。
- P01-W05 / P01-024..028：禁用/恢复用户、RBAC 角色、管理员登录与会话、一次性 step-up、邮件模板和投递记录。

## P01-W05 实现

- `backend/internal/identity` 扩展了管理员账号/会话、角色权限、用户状态、step-up challenge、模板版本和投递记录的持久化模型。
- 管理端 API 只接受管理员 Bearer 会话；旧的 `X-Admin-Role` 临时头不再授权任何管理接口。
- 禁用用户撤销该用户全部设备会话；高风险写操作消费绑定当前管理员会话的 5 分钟 step-up 令牌。
- 管理员 bootstrap 仅读取运行时 `OWNER_ADMIN_EMAIL` 与 `ADMIN_BOOTSTRAP_PASSWORD`，仓库不含凭据。
- `web/admin` 已接入真实 API、RBAC 页面和 1440px 管理后台视觉壳；Vue 运行时模板编译器通过 `vite.config.js` 固定配置。

## 验证

- 本机：`web/admin` 的 Vitest 3 tests passed，Vite production build passed。
- 本机合同门禁：07 contracts、38 state continuity、40 public safety、43 OpenAPI drift、45 secret boundary 均通过。
- 浏览器运行验收：登录、用户列表、角色/管理员页面、step-up modal、邮件模板保存成功提示；截图索引见 `docs/evidence/UI_SCREENSHOT_INDEX.csv`。
- Go、Android、模拟器和完整阶段验收由 GitHub Actions 执行；本机不因缺少 Go/Android 工具链替代 CI 结果。
