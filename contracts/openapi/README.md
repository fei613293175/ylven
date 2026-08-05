# Canonical OpenAPI contract

`../openapi-skeleton.yaml` is the P00 canonical OpenAPI document. The generated
surface is intentionally checked by content hash so formatting or operation
drift cannot be hidden behind a regenerated client.

Run `python scripts/43_VERIFY_OPENAPI_DRIFT.py` from the repository root.
