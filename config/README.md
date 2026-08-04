# Environment Configuration

Configuration is layered and secret-free in the repository:

1. `base.yaml` contains non-secret defaults shared by all environments.
2. `environments/<name>.yaml` contains environment-specific overrides.
3. `.env` and secret references are injected by the runtime and are never
   committed. Local overrides belong in `.ylven-local/`.

Maps are merged recursively according to `schema.yaml`; an environment layer
overrides only the leaves it declares. The effective configuration is resolved
by the service that owns the setting and must contain every `required_paths`
entry after the merge.
No Android build may contain server credentials, provider keys, database
credentials, or Cloudflare secrets.
