# Turnstile Setup

1. Create separate staging and production widgets for `auth.orbexa.cc` and approved local/test hostnames.
2. Store Site Key in server-delivered public configuration; store Secret only in server Secret Manager/environment reference.
3. Registration and login first create a server challenge with purpose, nonce, email hash, IP/device risk context and expiry.
4. Android opens a controlled WebView page hosted by YLVEN, receives only a one-time challenge result, and restricts navigation/JavaScript bridge.
5. The backend calls Siteverify with Secret, response token, remote IP where appropriate and an idempotency key; validate success, hostname/action and one-time use.
6. Only after server verification may the email-code endpoint accept the challenge ID. Never trust a client boolean such as `captchaPassed=true`.
7. Apply per email/IP/device/challenge rate limits and audit suspicious reuse without logging the raw token.
