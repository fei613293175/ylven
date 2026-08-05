#!/usr/bin/env python3
from __future__ import annotations

import hashlib
import json
import tempfile
from pathlib import Path

from importlib.util import module_from_spec, spec_from_file_location


SCRIPT = Path(__file__).with_name("34_PREPARE_DESKTOP_DELIVERY.py")
SPEC = spec_from_file_location("desktop_delivery", SCRIPT)
assert SPEC and SPEC.loader
MODULE = module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


def main() -> int:
    root = SCRIPT.parents[1]
    with tempfile.TemporaryDirectory() as temporary:
        work = Path(temporary)
        artifact = work / "artifact"
        destination = work / "delivery"
        artifact.mkdir()
        apk = artifact / "YLVEN-1.1.0-P01.apk"
        apk.write_bytes(b"tested-apk")
        apk_hash = hashlib.sha256(apk.read_bytes()).hexdigest()
        (artifact / "AUTOMATED_TEST_REPORT.md").write_text("PASS\n", encoding="utf-8")
        (artifact / "VISUAL_DIFF_REPORT.md").write_text("PASS\n", encoding="utf-8")
        (artifact / "CI_PROVENANCE.json").write_text(
            json.dumps({"phase": "P01", "version": "1.1.0", "apk": apk.name, "apk_sha256": apk_hash}),
            encoding="utf-8",
        )

        MODULE.prepare_delivery(artifact, root, destination, "P01", "1.1.0")

        assert (destination / "P01-evidence" / "P01-W05.md").is_file()
        assert (destination / "P01-evidence" / "P01_FEATURE_STATUS.yaml").is_file()
        assert (destination / "OWNER_TEST_CHECKLIST.md").is_file()
        assert (destination / "FEATURES_ORIGINAL.md").is_file()
        assert (destination / "FEATURE_COMPLETION_COMPARISON.md").is_file()
        assert not (destination / "P00-evidence").exists()
        assert "P01-evidence/P01-W05.md" in (destination / "SHA256SUMS.txt").read_text(encoding="utf-8")
    print("PASS: phase-specific desktop delivery evidence")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
