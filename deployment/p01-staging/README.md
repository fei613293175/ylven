# P01 staging deployment

This deployment is isolated from every other server project:

- project directory: `/opt/ylven-p01`
- API container: `ylven-p01-api`
- loopback port: `127.0.0.1:28201`
- web root: `/www/wwwroot/ai-admin.orbexa.cc/current`
- Nginx vhost: `/www/server/panel/vhost/nginx/ai-admin.orbexa.cc.conf`
- auth vhost: `/www/server/panel/vhost/nginx/auth.orbexa.cc.conf`

The staging environment uses Cloudflare's documented always-pass test widget and test secret through the real Siteverify API. This proves the WebView, hosted page, Cloudflare token, server verification and one-time challenge binding without using production credentials. Production must replace both test values with a dedicated widget for `auth.orbexa.cc`, switch `EMAIL_MODE=smtp`, disable `IDENTITY_DEBUG_OTP`, and provide secrets through mounted files.

Builds must use a commit-derived `YLVEN_IMAGE_TAG`. Keep the previous image and static release directory until health, public API, admin login and rollback probes have passed.
