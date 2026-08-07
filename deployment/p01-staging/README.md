# P01 staging deployment

This deployment is isolated from every other server project:

- project directory: `/opt/ylven-p01`
- API container: `ylven-p01-api`
- loopback port: `127.0.0.1:28201`
- web root: `/www/wwwroot/ai-admin.orbexa.cc/current`
- Nginx vhost: `/www/server/panel/vhost/nginx/ai-admin.orbexa.cc.conf`
- auth vhost: `/www/server/panel/vhost/nginx/auth.orbexa.cc.conf`

The staging environment uses YLVEN's own short arithmetic verification. Answers are stored only as salted digests, expire with the login or registration request, can be completed once, and lock after five wrong attempts. This is a basic abuse barrier rather than an advanced risk-control service. Production must switch `EMAIL_MODE=smtp`, disable `IDENTITY_DEBUG_OTP`, and provide mail secrets through mounted files.

Builds must use a commit-derived `YLVEN_IMAGE_TAG`. Keep the previous image and static release directory until health, public API, admin login and rollback probes have passed.
