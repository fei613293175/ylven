# P01 staging deployment

This deployment is isolated from every other server project:

- project directory: `/opt/ylven-p01`
- API container: `ylven-p01-api`
- loopback port: `127.0.0.1:28201`
- web root: `/www/wwwroot/ai-admin.orbexa.cc/current`
- Nginx vhost: `/www/server/panel/vhost/nginx/ai-admin.orbexa.cc.conf`

The staging environment explicitly uses mock Turnstile verification and debug OTP delivery until the owner supplies dedicated Cloudflare and SMTP secrets. Production must switch to `TURNSTILE_MODE=external`, `EMAIL_MODE=smtp`, disable `IDENTITY_DEBUG_OTP`, and provide secrets through mounted files.

Builds must use a commit-derived `YLVEN_IMAGE_TAG`. Keep the previous image and static release directory until health, public API, admin login and rollback probes have passed.
