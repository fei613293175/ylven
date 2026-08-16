from __future__ import annotations

import importlib.util
import tempfile
import unittest
from pathlib import Path


SCRIPT = Path(__file__).resolve().parents[1] / "scripts" / "20_CHECK_UI_VISUAL_GATE.py"


def load_visual_gate():
    spec = importlib.util.spec_from_file_location("visual_gate", SCRIPT)
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(module)
    return module


class VisualGateChangeOrderTest(unittest.TestCase):
    def setUp(self):
        self.visual_gate = load_visual_gate()
        self.temp_dir = tempfile.TemporaryDirectory()
        self.root = Path(self.temp_dir.name)
        self.visual_gate.ROOT = self.root

    def tearDown(self):
        self.temp_dir.cleanup()

    def write_approved_change_order(self):
        path = self.root / "change-orders" / "CO-P03-001" / "CHANGE_ORDER.yaml"
        path.parent.mkdir(parents=True)
        path.write_text("status: APPROVED_FOR_IMPLEMENTATION\n", encoding="utf-8")

    def test_accepts_regular_approval(self):
        self.assertTrue(
            self.visual_gate.is_implementation_approved({"status": "APPROVED"})
        )

    def test_accepts_change_order_approval_with_matching_provenance(self):
        self.write_approved_change_order()
        self.assertTrue(
            self.visual_gate.is_implementation_approved(
                {
                    "status": "APPROVED_FOR_CHANGE_ORDER_IMPLEMENTATION",
                    "approved_by": "OWNER_REQUESTED_CO_P03_001",
                }
            )
        )

    def test_rejects_change_order_approval_with_mismatched_provenance(self):
        self.write_approved_change_order()
        self.assertFalse(
            self.visual_gate.is_implementation_approved(
                {
                    "status": "APPROVED_FOR_CHANGE_ORDER_IMPLEMENTATION",
                    "approved_by": "UNVERIFIED_APPROVER",
                }
            )
        )


if __name__ == "__main__":
    unittest.main()
