# 24 外部配置、Owner Actions 与 Secret 管理

## 1. 项目所有者确实需要提供的资源

- `orbexa.cc` 的 Cloudflare DNS、WAF、Turnstile 和 R2 管理权限；
- staging/production R2 Bucket、S3 兼容访问凭据与自定义域名；
- 邮件发送服务、验证后的发件域名、API 或 SMTP 凭据；
- 已部署 Sub2API Base URL、受控管理员集成凭据和测试用户；
- Linux/Docker 部署主机访问方式；
- 未来正式支付渠道和官方 OpenAI/Anthropic/xAI API 凭据。

## 2. OWNER_ACTIONS.md 格式

每条待办必须包含：阶段、Feature ID、目标、项目所有者操作步骤、需要填写的变量名、最小权限、Codex 已完成部分、验证命令、截止影响和 Secret 传递方式。不能只写“请配置邮箱”“请提供密钥”。完成后把 Secret 放入部署环境/Secret Manager，不粘贴进仓库文档。

## 3. 环境分层

至少区分 local、test、staging、production。环境变量模板只包含名称和说明，不包含真实值。数据库、Redis、NATS、R2、邮件、Turnstile、Sub2API 和官方 API 的 Secret 使用引用方式注入；日志配置必须对 Authorization、Cookie、验证码、密码、API Key 和预签名 URL 脱敏。

## 4. 域名规划

首期使用 `api.orbexa.cc`、`auth.orbexa.cc`、`admin.orbexa.cc`、`developer.orbexa.cc`、`docs.orbexa.cc`、`gateway.orbexa.cc`、`files.orbexa.cc`、`assets.orbexa.cc`、`download.orbexa.cc`、`status.orbexa.cc`、`sub2api.orbexa.cc` 和受限的 `sub2api-admin.orbexa.cc`。外部暴露面必须最小化，数据库、Redis、NATS 和内部 Worker 不直接公网开放。

## 5. 验证方式

每项外部配置都有确定性验证：DNS/TLS/健康检查、Turnstile test token、邮件投递与退信、R2 预签名上传下载、Sub2API 文本/流式/取消/图片/文件能力探测。验证结果记录时间、环境和 request_id，不在日志保存完整 Secret。

## 6. 不阻塞原则

缺少外部配置时，Codex完成接口、模拟器、契约测试、错误处理、后台配置项和部署模板。只有真实联调 Feature 标记 BLOCKED_EXTERNAL；不把整个阶段交回项目所有者，也不反复询问已知信息。
