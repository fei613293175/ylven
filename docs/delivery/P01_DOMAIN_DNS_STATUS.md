# P01 域名与 DNS 状态

- Phase: P01
- Version: 1.1.0
- Result: DEPLOYED_STAGING
- Checked at: 2026-08-06T03:35:02+08:00
- DNS provider: Cloudflare

| Host | Contract | Current state | Evidence |
|---|---|---|---|
| `ai-admin.orbexa.cc` | YLVEN 专用管理后台/API、代理、Full strict、`/health` | 已解析到 `103.96.149.219` | 公网 HTTPS 200，证书有效至 2026-11-03 |
| `auth.orbexa.cc` | 独立认证服务、`/health/ready` | 本版本未配置 | 认证 API 暂由 `ai-admin.orbexa.cc` 同源提供 |
| `admin.orbexa.cc` | 其他项目既有入口 | 保持原项目不变 | 本次未修改 |

本次没有覆盖或迁移其他项目域名；YLVEN 使用 `ai-admin.orbexa.cc`。
