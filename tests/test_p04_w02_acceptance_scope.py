from __future__ import annotations

import importlib.util
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
VERIFY = ROOT / "scripts" / "33_VERIFY_ANDROID_ACCEPTANCE_OUTPUT.py"
FINALIZE = ROOT / "scripts" / "52_FINALIZE_PHYSICAL_VISUAL_REVIEW.py"


def load_module(name: str, path: Path):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(module)
    return module


class P04W02AcceptanceScopeTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.verify = load_module("verify_android_acceptance", VERIFY)
        cls.finalize = load_module("finalize_physical_visual_review", FINALIZE)

    def test_packet_scope_is_exactly_the_twenty_five_w02_android_states(self):
        screenshots = self.verify.phase_android_screenshots("P04", "P04-W02")

        self.assertEqual(25, len(screenshots))
        self.assertEqual(
            {"YL-A-030", "YL-A-035", "YL-A-036", "YL-A-037", "YL-A-038"},
            {screenshot.split("-S", 1)[0] for screenshot in screenshots},
        )

    def test_packet_scope_maps_to_its_five_real_production_pages(self):
        expected = {
            "YL-A-030": "P04-W02-MESSAGE-ACTIONS.png",
            "YL-A-035": "P04-W02-GLOBAL-DEFAULT.png",
            "YL-A-036": "P04-W02-CONVERSATION-DEFAULT.png",
            "YL-A-037": "P04-W02-PER-MESSAGE-SELECTOR.png",
            "YL-A-038": "P04-W02-PER-MESSAGE-PROVENANCE.png",
        }

        self.assertEqual(expected, self.verify.expected_production_pages("P04", "P04-W02"))
        self.assertEqual(expected, self.finalize.expected_pages("P04", "P04-W02"))

    def test_phase_acceptance_does_not_gain_packet_only_production_pages(self):
        self.assertEqual({}, self.verify.expected_production_pages("P04", None))
        self.assertEqual({}, self.finalize.expected_pages("P04", None))


if __name__ == "__main__":
    unittest.main()
