#!/usr/bin/env python3
"""Hard gate for the three owner-facing per-phase delivery documents."""
from __future__ import annotations

import argparse
import re
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[1]
DOCS = {
    "original": "{phase}_FEATURES_ORIGINAL.md",
    "comparison": "{phase}_FEATURE_COMPLETION_COMPARISON.md",
    "checklist": "{phase}_OWNER_TEST_CHECKLIST.md",
}


def expected_feature_ids(phase: str) -> set[str]:
    data = yaml.safe_load((ROOT / "contracts" / "feature-map.yaml").read_text(encoding="utf-8"))
    return {str(item["feature_id"]) for item in data.get("features", []) if item.get("phase") == phase}


def covered_ids(text: str) -> set[str]:
    found = set(re.findall(r"\bP\d{2}-\d{3}\b", text))
    for start, end in re.findall(r"\b(P\d{2}-\d{3})\s*\.\.\s*(P\d{2}-\d{3})\b", text):
        phase_a, number_a = start.split("-")
        phase_b, number_b = end.split("-")
        if phase_a == phase_b:
            found.update(f"{phase_a}-{number:03d}" for number in range(int(number_a), int(number_b) + 1))
    return found


def validate(phase: str, version: str) -> list[str]:
    ids = expected_feature_ids(phase)
    errors: list[str] = []
    if not ids:
        return [f"no canonical Feature IDs found for {phase}"]
    for kind, pattern in DOCS.items():
        path = ROOT / "docs" / "delivery" / pattern.format(phase=phase)
        if not path.is_file():
            errors.append(f"missing {kind} document: {path.relative_to(ROOT)}")
            continue
        text = path.read_text(encoding="utf-8", errors="replace")
        if len(text.strip()) < 300:
            errors.append(f"{kind} document is too short to be an owner deliverable")
        if "Pxx" in text or "TBD" in text or "TODO" in text:
            errors.append(f"{kind} document contains an unresolved placeholder")
        if phase not in text or version not in text:
            errors.append(f"{kind} document does not identify {phase} / {version}")
        missing = sorted(ids - covered_ids(text))
        if missing:
            errors.append(f"{kind} document omits Feature IDs: {', '.join(missing)}")
    return errors


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--phase", required=True)
    parser.add_argument("--version", required=True)
    args = parser.parse_args()
    if args.phase.upper() == "AUTO":
        print("SKIP: owner delivery manifest requires an explicit phase")
        return 0
    errors = validate(args.phase.upper(), args.version)
    if errors:
        print("\n".join(f"FAIL: {error}" for error in errors))
        return 1
    print(f"PASS: {args.phase.upper()} owner delivery manifest covers all canonical Feature IDs")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
