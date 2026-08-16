from __future__ import annotations

import importlib.util
import unittest
from pathlib import Path


SCRIPT = Path(__file__).resolve().parents[1] / "scripts" / "30_COMPARE_ANDROID_SCREENSHOTS.py"


def load_comparator():
    spec = importlib.util.spec_from_file_location("visual_compare", SCRIPT)
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(module)
    return module


class VisualComparePacketScopeTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.compare = load_comparator()

    def test_phase_scope_keeps_all_matching_phase_rows(self):
        row = {
            "surface": "ANDROID",
            "phases": "P03|P04",
            "work_packets": "P03-W07|P04-W02",
            "page_id": "YL-A-035",
        }
        self.assertTrue(self.compare.applies_to_phase(row, "P04"))

    def test_packet_scope_only_keeps_exact_packet_rows(self):
        row = {
            "surface": "ANDROID",
            "phases": "P03|P04",
            "work_packets": "P03-W07|P04-W01",
            "page_id": "YL-A-033",
        }
        self.assertTrue(self.compare.applies_to_phase(row, "P04", "P04-W01"))
        self.assertFalse(self.compare.applies_to_phase(row, "P04", "P04-W02"))

    def test_packet_scope_rejects_non_android_rows(self):
        row = {
            "surface": "ADMIN",
            "phases": "P04",
            "work_packets": "P04-W01",
            "page_id": "YL-M-051",
        }
        self.assertFalse(self.compare.applies_to_phase(row, "P04", "P04-W01"))


if __name__ == "__main__":
    unittest.main()
