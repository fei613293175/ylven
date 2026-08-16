from __future__ import annotations

import importlib.util
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
VERIFY = ROOT / "scripts" / "33_VERIFY_ANDROID_ACCEPTANCE_OUTPUT.py
FINALIZE = ROOT / "scripts" / "52_FINALIZE_PHYSICAL_VISUAL_REVIEW.py


def load_module(name: str, path: Path):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(module)
    return module


class P04W01AcceptanceScopeTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.verify = load_module("verify_android_acceptance", VERIFY)
        cls.finalize = load_module("finalize_physical_visual_review", FINALIZE)

    def test_packet_scope_is_exactly_the_twelve_w01_android_states(self):
        screenshots = self.verify.phase_android_screenshots("P04", "P04-W01")

        self.assertEqual(12, len(screenshots))
        self.assertEqual(
            {"YL-A-033", "YL-A-034"},
            {screenshot.split("-S", 1)[0] for screenshot in screenshots},
        )

    def test_packet_scope_maps_to_its_two_real_production_pages(self):
        expected = {
            "YL-A-033": "P04-MODEL-SELECTOR.png",
            "YL-A-034": "P04-REASONING-PROFILE.png",
        }

        self.assertEqual(expected, self.verify.expected_production_pages("P04", "P04-W01"))
        self.assertEqual(expected, self.finalize.expected_pages("P04", "P04-W01"))

    def test_phase_acceptance_does_not_gain_packet_only_production_pages(self):
        self.assertEqual({}, self.verify.expected_production_pages("P04", None))
        self.assertEqual({}, self.finalize.expected_pages("P04", None))


if __name__ == "__main__":
    unittest.main()
