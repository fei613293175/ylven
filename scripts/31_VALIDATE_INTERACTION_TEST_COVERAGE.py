#!/usr/bin/env python3
from __future__ import annotations

import argparse
import csv
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SUPERSEDED_INTERACTIONS = {
    "YL-A-019-C-P03_002-01": "CO-P03-001-HOME-SEND",
}


def kotlin_text(root: Path) -> str:
    return "\n".join(
        path.read_text(encoding="utf-8", errors="replace")
        for path in sorted(root.rglob("*.kt"))
    )


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--phase")
    args = parser.parse_args()
    phase = args.phase.upper() if args.phase else None

    with (ROOT / "contracts" / "ui-interaction-map.csv").open(
        encoding="utf-8-sig", newline=""
    ) as source:
        rows = list(csv.DictReader(source))
    errors: list[str] = []
    ids = [row.get("interaction_id", "") for row in rows]
    if len(ids) != len(set(ids)):
        errors.append("duplicate Interaction IDs")
    required = [
        "interaction_id", "feature_id", "page_id", "surface", "user_action",
        "endpoint_or_event", "functional_source",
    ]
    for line, row in enumerate(rows, 2):
        for field in required:
            if not row.get(field):
                errors.append(f"row {line}: missing {field}")

    selected = [
        row for row in rows
        if row.get("surface") == "ANDROID"
        and (phase is None or row.get("feature_id", "").startswith(f"{phase}-"))
    ]
    implementation = kotlin_text(ROOT / "android" / "app" / "src" / "main")
    device_tests = kotlin_text(ROOT / "android" / "app" / "src" / "androidTest")
    interactions = {row["interaction_id"]: row for row in rows}
    for row in selected:
        interaction_id = row["interaction_id"]
        replacement_id = SUPERSEDED_INTERACTIONS.get(interaction_id)
        if replacement_id:
            replacement = interactions.get(replacement_id)
            if replacement is None:
                errors.append(f"{interaction_id}: missing replacement interaction {replacement_id}")
            elif replacement_id not in implementation or replacement_id not in device_tests:
                errors.append(f"{interaction_id}: replacement {replacement_id} lacks app/device coverage")
            continue
        if interaction_id not in implementation:
            errors.append(f"{interaction_id}: no matching Compose testTag/semantics in app source")
        if interaction_id not in device_tests:
            errors.append(f"{interaction_id}: no physical-device instrumentation reference")

    bindings_path = ROOT / "contracts" / "interaction-test-bindings.csv"
    if not bindings_path.is_file():
        errors.append("missing contracts/interaction-test-bindings.csv")
    else:
        with bindings_path.open(encoding="utf-8-sig", newline="") as source:
            bindings = {row.get("interaction_id"): row for row in csv.DictReader(source)}
        for row in selected:
            binding = bindings.get(row["interaction_id"])
            if binding is None:
                errors.append(f"{row['interaction_id']}: missing interaction-test binding")
            elif binding.get("compose_test_tag") != row["interaction_id"]:
                errors.append(f"{row['interaction_id']}: binding must use the Interaction ID as testTag")

    if errors:
        print("\n".join(errors[:200]))
        return 1
    scope = phase or "all phases"
    print(f"PASS: {len(selected)} Android interactions for {scope} have app semantics and instrumentation coverage")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
