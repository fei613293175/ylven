# P01 部署端点

- Phase: P01
- Version: 1.1.0
- Result: DEPLOYED_STAGING
- Checked at: 2026-08-06T03:35:02+08:00
- Image: `ylven-p01-api:9d7ec53`
- Server container port: `127.0.0.1:28201`

| Endpoint | Required result | Current evidence |
|---|---|---|
| `https://ai-admin.orbexa.cc/health` | YLVEN 管理后台/API 返回 200 | 公网 HTTPS 200；HTTP 自动跳转 HTTPS |
| `https://ai-admin.orbexa.cc/health/ready` | YLVEN Identity Service 返回 200 | 公网 HTTPS 200 |
| `https://auth.orbexa.cc/health/ready` | 独立认证域名返回 200 | 本版本未配置；认证 API 由 `ai-admin.orbexa.cc` 同源提供 |

本次只部署 YLVEN 专用 `ai-admin.orbexa.cc`，没有覆盖其他项目的域名或容器。证书有效期至 2026-11-03。
