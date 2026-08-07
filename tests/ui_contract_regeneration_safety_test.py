from __future__ import annotations

import copy
import csv
import importlib.util
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
SPEC = importlib.util.spec_from_file_location(
    "regenerate_ui_contracts", ROOT / "scripts" / "23_REGENERATE_UI_CONTRACTS.py"
)
assert SPEC and SPEC.loader
REGENERATE_UI_CONTRACTS = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(REGENERATE_UI_CONTRACTS)


def approved_manifest_rows() -> dict[str, dict[str, str]]:
    with (ROOT / "contracts" / "mockup-manifest.csv").open(
        encoding="utf-8-sig", newline=""
    ) as source:
        return {row["mockup_id"]: row for row in csv.DictReader(source)}


class UiContractRegenerationSafetyTest(unittest.TestCase):
    def test_all_current_approved_visual_evidence_is_intact(self) -> None:
        # This validates the immutable baseline before the reconciliation command
        # can make any filesystem changes. It intentionally never calls main()
        # or rebuild_contracts().
        REGENERATE_UI_CONTRACTS.require_approved_visual_evidence(approved_manifest_rows())

    def test_missing_approved_asset_is_rejected_before_reconciliation(self) -> None:
        rows = approved_manifest_rows()
        mockup_id, row = next(
            (item for item in rows.items() if item[1].get("status") == "APPROVED")
        )
        mutated = copy.deepcopy(rows)
        mutated[mockup_id]["relative_path"] = "ui/mockups/android/missing-approved-baseline.png"

        with self.assertRaisesRegex(RuntimeError, "approved PNG is missing"):
            REGENERATE_UI_CONTRACTS.require_approved_visual_evidence(mutated)

    def test_changed_approved_hash_is_rejected_before_reconciliation(self) -> None:
        rows = approved_manifest_rows()
        mockup_id, row = next(
            (item for item in rows.items() if item[1].get("status") == "APPROVED")
        )
        mutated = copy.deepcopy(rows)
        mutated[mockup_id]["sha256"] = "0" * 64

        with self.assertRaisesRegex(RuntimeError, "SHA-256 does not match"):
            REGENERATE_UI_CONTRACTS.require_approved_visual_evidence(mutated)


if __name__ == "__main__":
    unittest.main()
