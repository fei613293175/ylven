# Cloudflare DNS and Exposure Plan

| Host | Purpose | Proxy | Origin exposure |
|---|---|---|---|
| api.orbexa.cc | Android/Mobile API | Orange cloud | HTTPS only through reverse proxy/WAF |
| auth.orbexa.cc | login/registration/OIDC | Orange cloud | strict rate limit and bot protection |
| admin.orbexa.cc | management UI | Orange cloud | access policy/MFA recommended |
| developer.orbexa.cc | developer portal | Orange cloud | public HTTPS |
| docs.orbexa.cc | public API docs | Orange cloud | public HTTPS |
| gateway.orbexa.cc | public developer API | Orange cloud | API WAF, rate/budget enforcement |
| files.orbexa.cc | private/signed file delivery | Orange cloud/R2 custom domain | no directory listing |
| assets.orbexa.cc | public static assets | Orange cloud/R2 custom domain | immutable caching |
| download.orbexa.cc | APK download page | Orange cloud | signed/versioned downloads |
| status.orbexa.cc | service status | Orange cloud | read-only |
| sub2api.orbexa.cc | internal/test gateway user surface | Orange cloud | authenticated only |
| sub2api-admin.orbexa.cc | Sub2API admin | Orange cloud | Cloudflare Access/IP/MFA; never open broadly |

PostgreSQL, Redis, NATS, object-storage S3 credentials, worker ports, metrics exporters and internal admin APIs are not public DNS services. Use Full (strict) TLS, origin certificates or trusted certificates, HSTS after validation, and separate staging origins where possible.
