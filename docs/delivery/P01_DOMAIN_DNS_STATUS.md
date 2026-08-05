# P01 域名与 DNS 状态

- Phase: P01
- Version: 1.1.0
- Result: BLOCKED_EXTERNAL
- DNS provider: Cloudflare

| Host | Contract | Current state | Required owner decision/action |
|---|---|---|---|
| `auth.orbexa.cc` | CNAME/Tunnel、代理、Full strict、`/health/ready` | NXDOMAIN | 提供真实 Tunnel/CNAME 目标并创建记录 |
| `admin.orbexa.cc` | CNAME/Tunnel、代理、Full strict、`/health` | 已指向另一线上产品 | 决定迁移现有域名，或批准新的 YLVEN 专用子域名 |

禁止猜测 DNS 目标，禁止在没有所有者决定时覆盖现有站点。
