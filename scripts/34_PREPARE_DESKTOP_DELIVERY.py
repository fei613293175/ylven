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


def prepare_delivery(
    src: Path,
    root: Path,
    dest: Path,
    phase: str,
    version: str,
    run_id: int | None = None,
    formal_dest: Path | None = None,
) -> None:
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
    if run_id is not None:
        provenance["workflow_run_id"] = run_id
        (dest / "CI_PROVENANCE.json").write_text(json.dumps(provenance, indent=2) + "\n", encoding="utf-8")

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
    shutil.copy2(dest / "FEATURES_ORIGINAL.md", dest / "FEATURES_PLANNED.md")
    shutil.copy2(dest / "FEATURE_COMPLETION_COMPARISON.md", dest / "FEATURES_COMPLETED.md")
    for name in ("DEPLOYMENT_ENDPOINTS", "DOMAIN_DNS_STATUS", "ADMIN_ACCESS"):
        source = root / "docs" / "delivery" / f"{phase}_{name}.md"
        if source.is_file():
            shutil.copy2(source, dest / f"{name}.md")

    version_parts = [int(part) for part in version.split(".")]
    if len(version_parts) != 3:
        raise ValueError(f"invalid semantic version: {version}")
    build_info = {
        "phase": phase,
        "version_name": version,
        "version_code": version_parts[0] * 1_000_000 + version_parts[1] * 10_000 + version_parts[2] * 100,
        "git_commit": provenance.get("commit_sha"),
        "apk_name": apk.name,
        "apk_sha256": expected["apk_sha256"],
        "apk_size": apk.stat().st_size,
        "artifact_origin": "github-actions",
        "source_apk": f"github-actions-run-{run_id}/{apk.name}" if run_id is not None else f"github-actions/{apk.name}",
        "workflow_run_id": run_id,
        "design_system_version": "YL-DS-1.2.0",
        "self_test_fixture": False,
    }
    (dest / "BUILD_INFO.json").write_text(json.dumps(build_info, indent=2) + "\n", encoding="utf-8")
    acceptance = (root / "templates" / "OWNER_ACCEPTANCE_TEMPLATE.md").read_text(encoding="utf-8")
    acceptance = acceptance.replace("YLVEN-Pxx-test.apk", apk.name).replace("Pxx", phase)
    (dest / "OWNER_ACCEPTANCE.md").write_text(acceptance, encoding="utf-8")
    owner_actions = root / "OWNER_ACTIONS.md"
    if owner_actions.is_file():
        shutil.copy2(owner_actions, dest / "OWNER_ACTIONS.md")

    lines = []
    for path in sorted(
        item for item in dest.rglob("*")
        if item.is_file() and item.name not in {"SHA256SUMS.txt", "OWNER_ACCEPTANCE.md"}
    ):
        lines.append(f"{sha256(path)}  {path.relative_to(dest).as_posix()}")
    (dest / "SHA256SUMS.txt").write_text("\n".join(lines) + "\n", encoding="utf-8")
    if formal_dest is not None:
        preserved_acceptance: str | None = None
        old_provenance = formal_dest / "CI_PROVENANCE.json"
        old_acceptance = formal_dest / "OWNER_ACCEPTANCE.md"
        if old_provenance.is_file() and old_acceptance.is_file():
            try:
                old = json.loads(old_provenance.read_text(encoding="utf-8"))
                if old.get("commit_sha") == provenance.get("commit_sha") and old.get("apk_sha256") == expected["apk_sha256"]:
                    preserved_acceptance = old_acceptance.read_text(encoding="utf-8")
            except json.JSONDecodeError:
                pass
        if formal_dest.exists():
            shutil.rmtree(formal_dest)
        shutil.copytree(dest, formal_dest)
        if preserved_acceptance is not None:
            (formal_dest / "OWNER_ACCEPTANCE.md").write_text(preserved_acceptance, encoding="utf-8")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--artifact-dir", required=True)
    parser.add_argument("--phase", required=True)
    parser.add_argument("--version", required=True)
    parser.add_argument("--run-id", type=int)
    parser.add_argument("--destination")
    args = parser.parse_args()

    root = Path(__file__).resolve().parents[1]
    destination = Path(args.destination) if args.destination else Path.home() / "Desktop" / "YLVEN-Releases" / args.version
    try:
        prepare_delivery(
            Path(args.artifact_dir), root, destination, args.phase, args.version,
            run_id=args.run_id, formal_dest=root / "dist" / "releases" / args.phase,
        )
    except (OSError, ValueError, json.JSONDecodeError) as exc:
        print(exc)
        return 1
    print(destination)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
