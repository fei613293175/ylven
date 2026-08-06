# P02 域名 DNS 状态

- 阶段：P02
- 版本：1.2.0
- ai-admin.orbexa.cc：解析与 HTTPS 正常；当前公网 API 回读 staging 版本 1.2.0-12f0bb5。
- auth.orbexa.cc：公共 DNS 已解析到 103.96.149.219；Let’s Encrypt 证书 SAN=auth.orbexa.cc；`/security/turnstile` 真实页面 200。
- assets.orbexa.cc：DNS NXDOMAIN，P02 静态资源域名外部阻塞。
- download.orbexa.cc：Cloudflare DNS 已解析，但未配置 YLVEN P02 APK 下载服务，下载路径 NOT RUN/外部阻塞。
- 结论：auth/ai-admin 公网入口 PASS；assets/download 仍为本阶段未配置外部项。
