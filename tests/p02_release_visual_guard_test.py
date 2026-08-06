from __future__ import annotations

import importlib.util
import unittest
from pathlib import Path

from PIL import Image


ROOT = Path(__file__).resolve().parents[1]
SPEC = importlib.util.spec_from_file_location(
    "verify_release", ROOT / "scripts" / "16_VERIFY_RELEASE.py"
)
assert SPEC and SPEC.loader
VERIFY_RELEASE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(VERIFY_RELEASE)


class P02ReleaseVisualGuardTest(unittest.TestCase):
    def test_keeps_p01_near_duplicate_rule(self) -> None:
        left = Image.new("RGB", (100, 100), "white")
        right = left.copy()
        right.putpixel((50, 50), (0, 0, 0))
        errors: list[str] = []

        VERIFY_RELEASE.validate_runtime_screenshot_distinctness(
            [("login.png", left), ("register.png", right)], "P01", errors
        )

        self.assertEqual(
            errors,
            ["CI runtime screenshots are effectively identical: login.png, register.png"],
        )

    def test_rejects_pixel_identical_screenshots(self) -> None:
        left = Image.new("RGB", (32, 32), "white")
        right = left.copy()
        errors: list[str] = []

        VERIFY_RELEASE.validate_runtime_screenshot_distinctness(
            [("loading.png", left), ("success.png", right)], "P02", errors
        )

        self.assertEqual(
            errors,
            ["CI runtime screenshots are pixel-identical: loading.png, success.png"],
        )

    def test_accepts_localized_but_meaningful_state_change(self) -> None:
        loading = Image.new("RGB", (100, 100), "white")
        success = loading.copy()
        for x in range(40, 60):
            for y in range(40, 60):
                success.putpixel((x, y), (24, 160, 88))
        errors: list[str] = []

        VERIFY_RELEASE.validate_runtime_screenshot_distinctness(
            [("loading.png", loading), ("success.png", success)], "P02", errors
        )

        self.assertEqual(errors, [])


if __name__ == "__main__":
    unittest.main()
