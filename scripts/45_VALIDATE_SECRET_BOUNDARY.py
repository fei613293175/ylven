"""Check public secret-storage rules without inspecting ignored local values."""
from pathlib import Path

root = Path(__file__).resolve().parents[1]
ignore = (root / ".gitignore").read_text(encoding="utf-8")
required = [".env", "*.pem", "*.key", "*.jks", ".ylven-local/"]
missing = [item for item in required if item not in ignore]
if missing:
    raise SystemExit("Missing secret ignore rules: " + ", ".join(missing))
for path in root.rglob("*"):
    if path == Path(__file__) or not path.is_file() or ".git" in path.parts or "node_modules" in path.parts:
        continue
    try:
        text = path.read_text(encoding="utf-8")
    except (UnicodeDecodeError, OSError):
        continue
    private_marker = "-----BEGIN " + "PRIVATE KEY-----"
    if private_marker in text or ("A" + "KIA") in text:
        raise SystemExit(f"credential material detected in public file: {path.relative_to(root)}")
print("PASS: secret boundary rules and public placeholders are valid")
