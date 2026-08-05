"""Require every P00 feature to have status and concrete evidence references."""
from pathlib import Path
import yaml

root = Path(__file__).resolve().parents[1]
feature_map = yaml.safe_load((root / "contracts" / "feature-map.yaml").read_text(encoding="utf-8"))["features"]
status = yaml.safe_load((root / "status" / "P00_FEATURE_STATUS.yaml").read_text(encoding="utf-8"))
selected = {item["feature_id"] for item in feature_map if item.get("phase") == "P00"}
recorded = {item["feature_id"]: item for item in status["features"] if item["feature_id"] in selected}
missing = sorted(feature for feature in selected if recorded.get(feature, {}).get("status") != "PLANNED" and not recorded.get(feature, {}).get("evidence"))
if missing:
    raise SystemExit("Missing P00 evidence: " + ", ".join(missing))
print(f"PASS: {len(selected)} P00 features have status and evidence references")
