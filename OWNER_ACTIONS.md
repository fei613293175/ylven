# OWNER_ACTIONS

P01 code, isolated staging deployment and automated mock-mode integration can be completed without placing any Secret in Git. The following external production integrations remain owner-controlled and must not be represented as already verified.

| Phase / Feature | Goal | Current state | Owner-controlled input | Safe delivery method | Completion check |
|---|---|---|---|---|---|
| P01 / P01-003 | Real Cloudflare Turnstile siteverify | Adapter implemented; staging explicitly uses mock token | `TURNSTILE_SECRET`, site key, expected hostname/action | Mount `TURNSTILE_SECRET_FILE` on server; never commit it | Valid token passes once; invalid, wrong-host, wrong-action and replay fail |
| P01 / P01-004, P01-009 | Real registration/login email delivery | SMTP adapter implemented; staging explicitly returns debug OTP | SMTP host, port, username/password and verified sender | Mount `SMTP_PASSWORD_FILE`; set non-secret SMTP fields in deployment env | Test mailbox receives OTP; failed delivery is recorded; response never contains `debug_code` |
| P01 / auth domain | Dedicated authentication web origin | `auth.orbexa.cc` remains NXDOMAIN; API is temporarily co-hosted on `ai-admin.orbexa.cc` | Cloudflare DNS record and future Turnstile widget hostname | Configure in Cloudflare, then issue TLS certificate | DNS/TLS and `https://auth.orbexa.cc/health/ready` return the YLVEN service |
| P05 | R2 staging storage | Not required by P01 | R2 bucket and S3-compatible credentials | Server secret files / Secret Manager | Pre-signed upload, download and delete pass |
| P04 | Sub2API controlled integration | Not required by P01 | Base URL and minimum-scope integration credential | Server secret file / Secret Manager | Capability probe records text, stream, cancel, image and file results |

P01 staging limitations are repeated in `docs/delivery/P01_FEATURE_COMPLETION_COMPARISON.md` and `docs/delivery/P01_OWNER_TEST_CHECKLIST.md` so the release cannot be mistaken for production email or production Turnstile acceptance.
