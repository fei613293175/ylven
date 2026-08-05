# 15 orbexa.cc 域名、Cloudflare、R2 与部署

## 1. 域名规划

| 域名 | 用途 | 公网策略 |
|---|---|---|
| `api.orbexa.cc` | Android Mobile BFF 与 YLVEN 原生 API | Cloudflare 橙云、WAF、限流 |
| `gateway.orbexa.cc` | 对外 OpenAI 兼容/公共 API | 橙云、严格 Key 限流 |
| `ai-admin.orbexa.cc` | YLVEN 专业管理后台 | 橙云，建议 Cloudflare Access |
| `developer.orbexa.cc` | 开发者中心 Web | 橙云 |
| `docs.orbexa.cc` | API 文档与帮助 | CDN |
| `auth.orbexa.cc` | Turnstile 移动验证页与未来 OIDC | 橙云，严格 CSP |
| `files.orbexa.cc` | 私有文件授权下载网关 | 橙云、短期签名 |
| `assets.orbexa.cc` | 公共品牌/静态资源 CDN | R2 自定义域名 |
| `download.orbexa.cc` | APK 下载与更新清单 | R2/CDN |
| `status.orbexa.cc` | 服务状态页 | 与主系统故障域尽量隔离 |
| `sub2api.orbexa.cc` | Sub2API 用户入口（如保留） | 受控开放 |
| `sub2api-admin.orbexa.cc` | Sub2API 管理入口 | Cloudflare Access/白名单 |

内部服务使用 Docker 网络 DNS，不暴露 `core-api`、`ai-runtime`、数据库、Redis、NATS 或 Worker 端口。

## 2. Cloudflare 配置基线

1. 所有公网域名启用代理并使用 Full (strict) TLS；
2. 源站安装 Cloudflare Origin Certificate 或受信证书；
3. 仅开放 80/443，数据库、Redis、NATS 不公网监听；
4. API 路径关闭不适当缓存；静态资源使用内容哈希和长期缓存；
5. SSE 路径验证代理缓冲、连接超时和响应压缩配置；
6. 管理后台叠加 Access，并保留 YLVEN 管理员 RBAC；
7. Turnstile sitekey 按环境分离，action 区分注册和登录；
8. WAF 对登录、验证码、文件上传、公共 API 和管理接口配置不同规则；
9. 使用 Cloudflare Rate Limiting 作为边缘第一层，应用内 Redis 作为业务第二层；
10. 日志和规则变更保留审计。

## 3. R2 桶设计

建议至少分为：

- `ylven-private-files-{env}`：用户上传原文件和解析产物，私有；
- `ylven-artifacts-{env}`：图片、PPT、PDF 和作品版本，私有；
- `ylven-public-assets-{env}`：品牌、发现页素材和公共模板缩略图；
- `ylven-releases-{env}`：APK、更新清单和校验和；
- `ylven-backups-{env}`：经加密的应用级备份（如采用）。

生产访问使用自定义域名或授权网关，不使用 `r2.dev` 作为正式分发入口。对象 key 由后端生成，例如：

```text
users/{user_id}/files/{attachment_id}/original
artifacts/{artifact_id}/versions/{version_id}/deck.pptx
releases/android/{version_code}/YLVEN.apk
```

数据库保存 bucket、key、etag、size、mime、sha256、owner、data classification 和 lifecycle state。

## 4. 环境隔离

至少建立：

- `local`：开发者本机和 mock upstream；
- `staging`：真实部署、测试数据、可选低额度 Sub2API 测试通道；
- `production`：未来商业流量和正式 API。

不同环境使用独立数据库、Redis namespace、R2 桶、Turnstile sitekey、Sub2API 用户/分组和密钥。禁止 staging 直接使用生产数据库。

## 5. 首期部署结构

推荐 Linux 服务器 + Docker Compose：

```text
nginx
core-api (2 replicas when needed)
ai-runtime (independent replicas)
developer-gateway
worker-email
worker-file
worker-image
worker-ppt
scheduler
admin-web
developer-web
postgresql
pgbouncer
redis
nats
observability stack
sub2api (independent compose/project)
```

Sub2API 保持独立 Compose、数据库和版本。YLVEN 通过私网地址访问，不将管理员端暴露给 Android。

## 6. 配置和 Secret

仓库只包含 `.env.example`。真实环境变量由服务器 Secret 文件、容器 Secret 或后续 Secret Manager 提供。Codex 需要真实值时生成 `OWNER_ACTIONS.md`，列出字段、用途、后台/Cloudflare 操作路径和验证命令，不得停下全部开发等待密钥；可使用明确 mock 完成其余代码和测试。

必需 Secret 示例：数据库、Redis/NATS、JWT/会话签名、R2 Access Key、Turnstile Secret、邮件供应商、Sub2API Base URL/Admin Credential、上游官方 API（未来）、支付渠道（后期）。

## 7. 发布与回滚

后端镜像使用不可变版本和 Git commit 标签，不使用 `latest`。发布流程：构建、测试、迁移预检、备份、部署 staging、冒烟、生产部署、健康检查和记录。数据库迁移必须向前兼容滚动发布；破坏性变更采用 expand-migrate-contract。

Android APK 放入 release 桶，生成签名下载 URL、SHA256、版本清单和更新说明。首期自有分发，不实现商店上架流程。

## 8. DNS 与所有者操作

Codex 可以生成 DNS 清单和验证脚本，但项目所有者需要在 Cloudflare 控制台完成最终域名和密钥操作。每次需要操作时必须写入 `OWNER_ACTIONS.md`，每项包含：目的、具体记录类型/主机名/目标值、代理状态、验证命令和完成勾选框。

## 9. 部署验收

- 所有公网端点使用 HTTPS；
- 源站无法绕过 Cloudflare 直接访问敏感服务；
- SSE 连续 10 分钟不被错误缓冲或截断；
- R2 私有文件未经授权无法访问；
- APK 可下载且 SHA256 一致；
- 管理后台受 Access/RBAC 保护；
- 数据库、Redis、NATS 无公网端口；
- 备份恢复至少在 staging 演练通过；
- 停掉任一无状态实例不影响整体服务。
