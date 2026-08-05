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
FINAL_STATUSES = {"IMPLEMENTED", "DEFERRED_WITH_REASON", "BLOCKED_EXTERNAL"}


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


def table_rows(text: str, phase: str) -> dict[str, list[str]]:
    rows: dict[str, list[str]] = {}
    for line in text.splitlines():
        if not line.lstrip().startswith("|"):
            continue
        cells = [cell.strip().strip("`") for cell in line.strip().strip("|").split("|")]
        if cells and re.fullmatch(rf"{re.escape(phase)}-\d{{3}}", cells[0]):
            rows[cells[0]] = cells
    return rows


def status_by_feature(phase: str) -> dict[str, str]:
    path = ROOT / "status" / f"{phase}_FEATURE_STATUS.yaml"
    if not path.is_file():
        return {}
    data = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
    return {
        str(item.get("feature_id")): str(item.get("status"))
        for item in data.get("features", [])
        if item.get("feature_id")
    }


def validate(phase: str, version: str) -> list[str]:
    ids = expected_feature_ids(phase)
    canonical_statuses = status_by_feature(phase)
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
        if kind in {"original", "comparison"}:
            rows = table_rows(text, phase)
            missing_rows = sorted(ids - set(rows))
            if missing_rows:
                errors.append(f"{kind} table has no individual row for: {', '.join(missing_rows)}")
            for feature_id, cells in rows.items():
                if kind == "original" and (len(cells) < 2 or len(cells[1]) < 4):
                    errors.append(f"original row {feature_id} has no concrete planned function")
                if kind == "comparison":
                    if len(cells) < 4:
                        errors.append(f"comparison row {feature_id} must include function, status and evidence/difference")
                        continue
                    status = cells[2]
                    if status not in FINAL_STATUSES:
                        errors.append(f"comparison row {feature_id} has invalid final status: {status}")
                    expected_status = canonical_statuses.get(feature_id)
                    if expected_status and status != expected_status:
                        errors.append(f"comparison row {feature_id} status differs from status YAML: {status} != {expected_status}")
                    if len(cells[3]) < 12:
                        errors.append(f"comparison row {feature_id} has no concrete evidence or difference")
        if kind == "checklist":
            checkboxes = re.findall(r"(?m)^\s*-\s*\[(?: |x|X)\]\s+.+$", text)
            if len(checkboxes) < 10:
                errors.append("checklist must contain at least 10 executable owner checks")
            for marker in ("进入路径", "操作步骤", "正确预期", "失败", "恢复", "已知限制"):
                if marker not in text:
                    errors.append(f"checklist omits required section or field: {marker}")
    missing_statuses = sorted(ids - set(canonical_statuses))
    if missing_statuses:
        errors.append(f"feature status YAML omits: {', '.join(missing_statuses)}")
    invalid_statuses = sorted(
        feature_id for feature_id, status in canonical_statuses.items()
        if feature_id in ids and status not in FINAL_STATUSES
    )
    if invalid_statuses:
        errors.append(f"features are not in a final release status: {', '.join(invalid_statuses)}")
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
