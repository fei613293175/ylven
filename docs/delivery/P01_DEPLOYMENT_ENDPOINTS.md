# P01 部署端点

- Phase: P01
- Version: 1.1.0
- Result: BLOCKED_EXTERNAL
- Checked at: 2026-08-05T21:24:55+08:00

| Endpoint | Required result | Current evidence |
|---|---|---|
| `https://auth.orbexa.cc/health/ready` | YLVEN Identity Service 返回 200 | DNS 不存在，未部署 |
| `https://admin.orbexa.cc/health` | YLVEN 管理后台返回 200 | 当前返回“合伙云 Pro 管理后台”，不是 YLVEN |

不得把现有合伙云页面、mock server 或本机 Vite 页面作为 P01 正式部署通过证据。需要项目所有者决定新的 YLVEN 管理后台域名，或明确批准迁移现有 `admin.orbexa.cc`；在此之前禁止覆盖线上配置。
