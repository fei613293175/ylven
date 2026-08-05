#!/usr/bin/env python3
from __future__ import annotations

import argparse
import hashlib
import json
import shutil
import subprocess
import sys
from pathlib import Path


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def copy_matches(source: Path, pattern: str, destination: Path) -> list[Path]:
    copied: list[Path] = []
    for path in sorted(source.glob(pattern)):
        if path.is_file():
            destination.mkdir(parents=True, exist_ok=True)
            target = destination / path.name
            shutil.copy2(path, target)
            copied.append(target)
    return copied


def prepare_delivery(src: Path, root: Path, dest: Path, phase: str, version: str) -> None:
    gate = root / "scripts" / "47_VALIDATE_OWNER_DELIVERY_MANIFEST.py"
    result = subprocess.run(
        [sys.executable, str(gate), "--phase", phase, "--version", version],
        cwd=root,
        capture_output=True,
        text=True,
    )
    if result.returncode:
        raise ValueError(result.stdout.strip() or result.stderr.strip())
    required = ["AUTOMATED_TEST_REPORT.md", "VISUAL_DIFF_REPORT.md", "CI_PROVENANCE.json"]
    apks = list(src.rglob("*.apk"))
    errors: list[str] = []
    if len(apks) != 1:
        errors.append(f"exactly one tested APK required, got {len(apks)}")
    for name in required:
        if not (src / name).is_file():
            errors.append(f"missing {name}")
    if errors:
        raise ValueError("\n".join(errors))

    provenance = json.loads((src / "CI_PROVENANCE.json").read_text(encoding="utf-8"))
    apk = apks[0]
    expected = {
        "phase": phase,
        "version": version,
        "apk": apk.name,
        "apk_sha256": sha256(apk),
    }
    mismatches = [
        f"provenance {key} mismatch: expected {value}, got {provenance.get(key)}"
        for key, value in expected.items()
        if provenance.get(key) != value
    ]
    if mismatches:
        raise ValueError("\n".join(mismatches))

    if dest.exists():
        shutil.rmtree(dest)
    shutil.copytree(src, dest)

    evidence = dest / f"{phase}-evidence"
    packet_evidence = copy_matches(root / "docs" / "evidence", f"{phase}-W*.md", evidence)
    if not packet_evidence:
        raise ValueError(f"no work-packet evidence found for {phase}")
    copy_matches(root / "docs" / "evidence", f"VISUAL_DIFF_REPORT_{phase}*.md", evidence)
    screenshot_index = root / "docs" / "evidence" / "UI_SCREENSHOT_INDEX.csv"
    if screenshot_index.is_file():
        shutil.copy2(screenshot_index, evidence / screenshot_index.name)
    screenshots_root = root / "docs" / "evidence" / "screenshots"
    for screenshot_dir in sorted(screenshots_root.glob(f"{phase}-*")):
        if screenshot_dir.is_dir():
            shutil.copytree(screenshot_dir, evidence / "screenshots" / screenshot_dir.name)

    for status_name in (f"{phase}_FEATURE_STATUS.yaml", f"{phase}_IMPLEMENTATION_STATUS.md"):
        status_path = root / "status" / status_name
        if status_path.is_file():
            shutil.copy2(status_path, evidence / status_name)
    copy_matches(root / "phases", f"{phase}_*.md", evidence)
    for spec_dir in sorted((root / "specs").glob(f"*-{phase.lower()}-*")):
        tasks = spec_dir / "tasks.md"
        if tasks.is_file():
            shutil.copy2(tasks, evidence / f"{phase}-tasks.md")

    delivery_docs = copy_matches(root / "docs" / "delivery", f"{phase}_*.md", dest / f"{phase}-delivery-docs")
    if not delivery_docs:
        empty_delivery_dir = dest / f"{phase}-delivery-docs"
        if empty_delivery_dir.exists():
            empty_delivery_dir.rmdir()
    shutil.copy2(root / "docs" / "delivery" / f"{phase}_OWNER_TEST_CHECKLIST.md", dest / "OWNER_TEST_CHECKLIST.md")
    shutil.copy2(root / "docs" / "delivery" / f"{phase}_FEATURES_ORIGINAL.md", dest / "FEATURES_ORIGINAL.md")
    shutil.copy2(root / "docs" / "delivery" / f"{phase}_FEATURE_COMPLETION_COMPARISON.md", dest / "FEATURE_COMPLETION_COMPARISON.md")

    lines = []
    for path in sorted(item for item in dest.rglob("*") if item.is_file() and item.name != "SHA256SUMS.txt"):
        lines.append(f"{sha256(path)}  {path.relative_to(dest).as_posix()}")
    (dest / "SHA256SUMS.txt").write_text("\n".join(lines) + "\n", encoding="utf-8")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--artifact-dir", required=True)
    parser.add_argument("--phase", required=True)
    parser.add_argument("--version", required=True)
    parser.add_argument("--destination")
    args = parser.parse_args()

    root = Path(__file__).resolve().parents[1]
    destination = Path(args.destination) if args.destination else Path.home() / "Desktop" / "YLVEN-Releases" / args.version
    try:
        prepare_delivery(Path(args.artifact_dir), root, destination, args.phase, args.version)
    except (OSError, ValueError, json.JSONDecodeError) as exc:
        print(exc)
        return 1
    print(destination)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
