"""Verify the canonical OpenAPI document has not drifted from its recorded hash."""
from hashlib import sha256
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SPEC = ROOT / "contracts" / "openapi-skeleton.yaml"
CHECKSUM = ROOT / "contracts" / "openapi" / "openapi.sha256"

expected = CHECKSUM.read_text(encoding="utf-8").split()[0].lower()
actual = sha256(SPEC.read_bytes()).hexdigest()
if actual != expected:
    raise SystemExit(f"OpenAPI drift: expected {expected}, got {actual}")
print(f"PASS: canonical OpenAPI SHA-256 {actual}")
