# P01 部署端点

- Phase: P01
- Version: 1.1.0
- Result: DEPLOYED_STAGING
- Deployment result: SUCCESS
- Health check result: PASS
- Rollback result: NOT_APPLICABLE
- Checked at: 2026-08-06T03:35:02+08:00
- Image: `ylven-p01-api:9d7ec53`
- Server container port: `127.0.0.1:28201`

| Endpoint | Required result | Current evidence |
|---|---|---|
| `https://ai-admin.orbexa.cc/health` | YLVEN 管理后台/API 返回 200 | 公网 HTTPS 200；HTTP 自动跳转 HTTPS |
| `https://ai-admin.orbexa.cc/health/ready` | YLVEN Identity Service 返回 200 | 公网 HTTPS 200 |
| `https://auth.orbexa.cc/health/ready` | 独立认证域名返回 200 | 本版本未配置；认证 API 由 `ai-admin.orbexa.cc` 同源提供 |

本次只部署 YLVEN 专用 `ai-admin.orbexa.cc`，没有覆盖其他项目的域名或容器。证书有效期至 2026-11-03。

回滚证据：P01 是该主机上的首次 YLVEN 隔离 staging 部署，部署前没有可回滚的上一版 YLVEN 镜像或静态发布目录，因此没有执行会影响线上服务的回滚操作。当前源代码、Nginx 配置和管理员交付凭据已备份至 `/opt/ylven-p01/backups/20260806T025222/`；后续版本必须保留上一版镜像并完成无损回滚探针后才可标记 `PASS`。
