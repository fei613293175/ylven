# P01 原功能与完成情况对比

- 阶段：P01
- 版本：1.1.0
- 原始基线：`docs/delivery/P01_FEATURES_ORIGINAL.md`
- 状态权威：`status/P01_FEATURE_STATUS.yaml`
- 判定规则：代码实现、自动测试、staging 部署和项目所有者验收分别记录，任一层未通过都不得被另一层替代。

| Feature ID | 原开发功能 | 当前状态 | 完成证据/差异 |
|---|---|---|---|
| P01-001 | 标准化邮箱地址并校验格式 | IMPLEMENTED | `NormalizeEmail`、HTTP 422 错误及 Go 单元测试；公网 API 待本次部署回归。 |
| P01-002 | 创建注册安全验证挑战 | IMPLEMENTED | 持久化 challenge、过期时间、用途校验和审计事件均有代码及测试。 |
| P01-003 | 服务端校验 Turnstile 一次性令牌 | IMPLEMENTED | 已实现 Cloudflare Siteverify、hostname/action 校验和 mock 合同测试；当前 staging 明确使用 mock，真实 Cloudflare Secret 联调未验收。 |
| P01-004 | 发送注册邮箱验证码 | IMPLEMENTED | 已实现 SMTP STARTTLS 适配器、模板渲染和投递状态；当前 staging 使用 debug OTP，真实邮箱投递未验收。 |
| P01-005 | 验证注册邮箱验证码 | IMPLEMENTED | OTP 仅存摘要、限次、过期和一次性消费均有 Go 测试及 API 路径。 |
| P01-006 | 创建用户并安全哈希密码 | IMPLEMENTED | 密码使用加盐迭代 HMAC-SHA256 保存，响应和管理接口均不返回摘要；并发由持久层互斥保护。 |
| P01-007 | 防止邮箱重复注册和竞态 | IMPLEMENTED | 规范化邮箱唯一键、互斥写入和 409 `email_already_registered` 路径已实现。 |
| P01-008 | 创建登录安全验证挑战 | IMPLEMENTED | 登录 purpose、限流、Turnstile 状态和过期时间均由真实 Store/API 处理。 |
| P01-009 | 发送登录邮箱验证码并防账号枚举 | IMPLEMENTED | 未知邮箱返回统一 accepted 路径；SMTP 适配器已实现，staging 仍为 debug OTP，真实投递未验收。 |
| P01-010 | 验证登录邮箱验证码 | IMPLEMENTED | 登录 OTP 的错误、次数、过期、一次性消费和审计路径已实现并测试。 |
| P01-011 | 签发访问令牌和轮换刷新令牌 | IMPLEMENTED | 随机 token 只以 SHA-256 摘要持久化，访问/刷新有效期与会话创建测试通过。 |
| P01-012 | 刷新登录会话 | IMPLEMENTED | canonical refresh 路由已接通，刷新 token 单次轮换，旧 token 复用测试失败。 |
| P01-013 | 退出当前设备 | IMPLEMENTED | 当前 access token 对应会话撤销并记录脱敏审计事件。 |
| P01-014 | 退出全部设备 | IMPLEMENTED | 同一用户全部会话撤销，Android 失败时不再误清本地会话。 |
| P01-015 | 查询用户设备和会话 | IMPLEMENTED | Bearer 隔离用户，只返回当前用户未撤销会话且清除 token 摘要字段。 |
| P01-016 | 撤销指定设备会话 | IMPLEMENTED | canonical DELETE 路由校验资源归属并撤销目标 access/refresh 会话。 |
| P01-017 | 执行认证频率限制与冷却 | IMPLEMENTED | 用户及管理员登录均使用持久化窗口限流，触发后写风险审计。 |
| P01-018 | 记录认证审计和风险事件 | IMPLEMENTED | challenge、OTP、注册、登录、刷新、退出、撤销、限流和后台高风险操作均写脱敏事件；测试拒绝 token/OTP 泄漏。 |
| P01-019 | 管理邮件服务商和发件人 | IMPLEMENTED | 管理 API/页面可读写 provider、sender 和 secret reference，原始 Secret 字段被拒绝并写审计。 |
| P01-020 | 管理 Turnstile 站点与密钥引用 | IMPLEMENTED | 管理 API/页面可读写 site key/secret reference，不接收或回显原始 Secret。 |
| P01-021 | 管理验证码时效和发送频率 | IMPLEMENTED | OTP policy 管理 API/页面可保存并回读 ttl、send limit 和 cooldown 配置。 |
| P01-022 | 查询用户列表和状态 | IMPLEMENTED | `/admin/v1/users` 使用真实持久数据，密码摘要和 token 字段不进入响应。 |
| P01-023 | 查询用户详情、身份与设备 | IMPLEMENTED | 用户详情 API 和 `YL-M-036` 入口返回归属会话的脱敏视图。 |
| P01-024 | 禁用或恢复用户账号 | IMPLEMENTED | 一次性 step-up 后变更状态；禁用立即撤销该用户全部会话并写审计。 |
| P01-025 | 建立管理员角色和细粒度权限 | IMPLEMENTED | Bearer RBAC、默认角色、角色创建/修改和无权限拒绝均有 API/Go 测试与 `YL-M-037` 页面。 |
| P01-026 | 建立后台登录和管理员会话 | IMPLEMENTED | 用户名别名 `admin`、8 小时管理员会话、会话列表及服务端退出撤销均已实现。 |
| P01-027 | 建立高风险配置二次确认 | IMPLEMENTED | step-up 绑定管理员会话、5 分钟有效、一次性消费；缺失/复用/跨会话均拒绝。 |
| P01-028 | 建立邮箱模板和发送记录 | IMPLEMENTED | 版本化模板、乐观版本冲突、投递 queued/mocked/delivered/failed 状态和后台页面均已实现。 |

## 已知限制与未冒充完成项

- staging 为可公开验证的测试环境：`TURNSTILE_MODE=mock`、`EMAIL_MODE=mock`、`IDENTITY_DEBUG_OTP=1`。它验证完整业务链路，但不等于真实 Cloudflare/SMTP 验收。
- `auth.orbexa.cc` 当前没有 DNS；P01 测试 API 暂与后台同源部署在 `ai-admin.orbexa.cc`。
- staging 运行时使用权限为 `0600` 的原子 JSON 状态文件；PostgreSQL migration 已交付，但生产 PostgreSQL runtime adapter 尚未切换。
- Android 中注册/登录/设备页是针对“P01 APK 仍显示 P00”的交付纠正，不把阶段状态推进到 P02。
- 精确 GitHub Actions Run、APK SHA-256、模拟器截图和项目所有者结论必须以最终交付目录为准；当前文档不宣称项目所有者已经 APPROVED。
