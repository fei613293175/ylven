# YLVEN Deployment Templates

These files define the intended staging topology, environment variables and Cloudflare setup. They contain no real credentials and are not evidence that deployment has occurred. P00 must replace image/build placeholders with the actual repository modules, run health checks and record the result in the release DEPLOYMENT.md.

Public entry points: `api`, `auth`, `admin`, `developer`, `docs`, `gateway`, `files`, `assets`, `download`, `status`, `sub2api` and restricted `sub2api-admin` under `orbexa.cc`. PostgreSQL, Redis, NATS, workers and internal admin APIs stay on private networks.
