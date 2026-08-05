#!/usr/bin/env python3
"""Verify a phase release against RELEASE_CONTRACT.yaml and the locked UI contract."""
from __future__ import annotations

import argparse
import csv
import hashlib
import json
import re
import sys
import zipfile
from pathlib import Path

import yaml
from PIL import Image, ImageChops, ImageStat

ROOT = Path(__file__).resolve().parents[1]
FINAL_STATUSES = {"IMPLEMENTED", "DEFERRED_WITH_REASON", "BLOCKED_EXTERNAL"}
DESIGN_SYSTEM_VERSION = "YL-DS-1.2.0"
P01_RUNTIME_SCREENSHOTS = [
    "P01-AUTH-LOGIN.png", "P01-AUTH-REGISTER.png", "P01-AUTH-REGISTER-OTP.png",
    "P01-AUTH-REGISTERED.png", "P01-AUTH-ACCOUNT.png",
]


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def load_yaml(path: Path) -> dict:
    return yaml.safe_load(path.read_text(encoding="utf-8")) or {}


def phase_states(phase: str) -> list[dict[str, str]]:
    pages = (load_yaml(ROOT / "contracts" / "ui-page-catalog.yaml").get("pages") or [])
    selected_pages = {page["page_id"] for page in pages if phase in (page.get("phases") or [])}
    with (ROOT / "contracts" / "ui-state-catalog.csv").open(
        encoding="utf-8-sig", newline=""
    ) as handle:
        return [row for row in csv.DictReader(handle) if row.get("page_id") in selected_pages]


def mockups_by_state() -> dict[str, dict[str, str]]:
    with (ROOT / "contracts" / "mockup-manifest.csv").open(
        encoding="utf-8-sig", newline=""
    ) as handle:
        return {row["state_id"]: row for row in csv.DictReader(handle)}


def parse_sha_file(path: Path) -> dict[str, str]:
    result: dict[str, str] = {}
    for number, line in enumerate(path.read_text(encoding="utf-8").splitlines(), 1):
        if not line.strip():
            continue
        match = re.fullmatch(r"([0-9a-fA-F]{64})\s{2}(.+)", line.strip())
        if not match:
            raise ValueError(f"Invalid SHA256SUMS line {number}: {line!r}")
        name = match.group(2).replace("\\", "/")
        if name.startswith("/") or ".." in Path(name).parts:
            raise ValueError(f"Unsafe SHA256 path on line {number}: {name}")
        if name in result:
            raise ValueError(f"Duplicate SHA256 entry: {name}")
        result[name] = match.group(1).lower()
    return result


def validate_ui(release_dir: Path, phase: str, fixture: bool, errors: list[str]) -> None:
    states = phase_states(phase)
    expected_ids = {row["state_id"] for row in states}
    index_path = release_dir / "UI_SCREENSHOT_INDEX.csv"
    rows: list[dict[str, str]] = []
    if index_path.is_file():
        try:
            with index_path.open(encoding="utf-8-sig", newline="") as handle:
                rows = list(csv.DictReader(handle))
        except Exception as exc:
            errors.append(f"cannot parse UI_SCREENSHOT_INDEX.csv: {exc}")
    actual_ids = {row.get("state_id", "") for row in rows}
    if actual_ids != expected_ids:
        errors.append(
            "UI_SCREENSHOT_INDEX state set differs: "
            f"missing={sorted(expected_ids-actual_ids)[:20]}, extra={sorted(actual_ids-expected_ids)[:20]}"
        )

    report_path = release_dir / "UI_CONTRACT_REPORT.json"
    ui_report: dict = {}
    if report_path.is_file():
        try:
            ui_report = json.loads(report_path.read_text(encoding="utf-8"))
        except Exception as exc:
            errors.append(f"invalid UI_CONTRACT_REPORT.json: {exc}")
    if ui_report:
        if ui_report.get("phase") != phase:
            errors.append("UI_CONTRACT_REPORT phase mismatch")
        if ui_report.get("design_system_version") != DESIGN_SYSTEM_VERSION:
            errors.append("UI_CONTRACT_REPORT design system mismatch")
        if int(ui_report.get("expected_state_count", -1)) != len(states):
            errors.append("UI_CONTRACT_REPORT expected state count mismatch")

    diff_path = release_dir / "VISUAL_DIFF_REPORT.md"
    diff_text = diff_path.read_text(encoding="utf-8", errors="replace") if diff_path.is_file() else ""
    if f"- Phase: {phase}" not in diff_text:
        errors.append("VISUAL_DIFF_REPORT phase marker is missing")
    if f"- Design system: {DESIGN_SYSTEM_VERSION}" not in diff_text:
        errors.append("VISUAL_DIFF_REPORT design-system marker is missing")

    if fixture:
        if any(row.get("status") != "SELF_TEST_ONLY" for row in rows):
            errors.append("self-test UI screenshot index must use SELF_TEST_ONLY status")
        if ui_report.get("result") != "SELF_TEST_ONLY":
            errors.append("self-test UI_CONTRACT_REPORT result must be SELF_TEST_ONLY")
        if int(ui_report.get("screenshot_count", -1)) != 0:
            errors.append("self-test UI_CONTRACT_REPORT must not claim screenshots")
        if not re.search(r"(?im)^-\s*Result:\s*SELF_TEST_ONLY\s*$", diff_text):
            errors.append("self-test VISUAL_DIFF_REPORT marker is missing")
        return

    mockups = mockups_by_state()
    screenshots_dir = release_dir / "screenshots"
    if not screenshots_dir.is_dir():
        errors.append("real release is missing screenshots/ directory")
    declared_screenshots: set[str] = set()
    for row in rows:
        state_id = row.get("state_id", "")
        if row.get("status") != "PASS":
            errors.append(f"{state_id}: screenshot status must be PASS")
        relative = row.get("runtime_screenshot_path", "").replace("\\", "/")
        if relative.startswith("/") or ".." in Path(relative).parts:
            errors.append(f"{state_id}: unsafe runtime screenshot path")
            continue
        declared_screenshots.add(relative)
        screenshot = release_dir / relative
        if not screenshot.is_file():
            errors.append(f"{state_id}: runtime screenshot is missing")
        elif row.get("runtime_sha256", "").lower() != sha256(screenshot):
            errors.append(f"{state_id}: runtime screenshot SHA-256 mismatch")
        manifest = mockups.get(state_id)
        if not manifest:
            errors.append(f"{state_id}: missing mockup manifest row")
            continue
        if manifest.get("status") != "APPROVED":
            errors.append(f"{state_id}: mockup status is not APPROVED")
        mockup_path = ROOT / manifest.get("relative_path", "")
        if not mockup_path.is_file():
            errors.append(f"{state_id}: approved mockup file is missing")
        elif not manifest.get("sha256") or sha256(mockup_path) != manifest["sha256"].lower():
            errors.append(f"{state_id}: approved mockup SHA-256 mismatch")
        if row.get("mockup_sha256", "").lower() != manifest.get("sha256", "").lower():
            errors.append(f"{state_id}: screenshot index mockup SHA differs from manifest")
        if state_id not in diff_text:
            errors.append(f"VISUAL_DIFF_REPORT is missing {state_id}")
    actual_screenshots = {
        path.relative_to(release_dir).as_posix()
        for path in screenshots_dir.glob("*.png")
    } if screenshots_dir.is_dir() else set()
    if actual_screenshots != declared_screenshots:
        errors.append(
            "runtime screenshot file set differs from index: "
            f"missing={sorted(declared_screenshots-actual_screenshots)[:20]}, "
            f"extra={sorted(actual_screenshots-declared_screenshots)[:20]}"
        )
    if ui_report.get("result") != "PASS":
        errors.append("real release UI_CONTRACT_REPORT result must be PASS")
    if int(ui_report.get("screenshot_count", -1)) != len(states):
        errors.append("real release UI screenshot count mismatch")
    if int(ui_report.get("approved_mockup_count", -1)) != len(states):
        errors.append("real release approved mockup count mismatch")
    if int(ui_report.get("unexplained_difference_count", -1)) != 0:
        errors.append("unexplained visual differences are forbidden")
    if not re.search(r"(?im)^-\s*Result:\s*PASS\s*$", diff_text):
        errors.append("real release VISUAL_DIFF_REPORT result must be PASS")


def validate_ci_ui(release_dir: Path, phase: str, errors: list[str]) -> None:
    report = release_dir / "VISUAL_DIFF_REPORT.md"
    text = report.read_text(encoding="utf-8", errors="replace") if report.is_file() else ""
    if not re.search(r"(?im)^Result:\s*\*\*PASS[^\n]*\*\*\s*$", text):
        errors.append("CI VISUAL_DIFF_REPORT does not contain a PASS result")
    screenshot_dir = release_dir / "screenshots"
    names = P01_RUNTIME_SCREENSHOTS if phase == "P01" else [path.name for path in screenshot_dir.glob("*.png")]
    images: list[tuple[str, Image.Image]] = []
    for name in names:
        path = screenshot_dir / name
        if not path.is_file():
            errors.append(f"missing CI runtime screenshot: {name}")
            continue
        shot = Image.open(path).convert("RGB")
        images.append((name, shot))
        if shot.width < 720 or shot.height < 1280:
            errors.append(f"CI runtime screenshot is too small: {name} {shot.size}")
        if all(high - low < 8 for low, high in ImageStat.Stat(shot).extrema):
            errors.append(f"CI runtime screenshot is blank: {name}")
    for (left_name, left), (right_name, right) in zip(images, images[1:]):
        if left.size == right.size and sum(ImageStat.Stat(ImageChops.difference(left, right)).mean) / 3 < 2:
            errors.append(f"CI runtime screenshots are effectively identical: {left_name}, {right_name}")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--phase", required=True, choices=[f"P{i:02d}" for i in range(14)])
    parser.add_argument("--allow-self-test-fixture", action="store_true")
    args = parser.parse_args()

    phase = args.phase
    release_dir = ROOT / "dist" / "releases" / phase
    errors: list[str] = []
    if not release_dir.is_dir():
        print(f"ERROR: missing release directory {release_dir}", file=sys.stderr)
        return 1

    contract = load_yaml(ROOT / "RELEASE_CONTRACT.yaml")
    required = [name.format(phase=phase) for name in contract.get("required_outputs") or []]
    for name in required:
        path = release_dir / name
        if not path.is_file() or path.stat().st_size == 0:
            errors.append(f"missing or empty required output: {name}")

    apk_files = sorted(release_dir.glob("*.apk"))
    if len(apk_files) != 1:
        errors.append(f"release must contain exactly one APK, got {len(apk_files)}")
    apk = apk_files[0] if len(apk_files) == 1 else release_dir / f"missing-{phase}.apk"
    minimum = int((contract.get("apk_rules") or {}).get("minimum_bytes", 1048576))
    if apk.is_file():
        if apk.stat().st_size < minimum:
            errors.append(f"APK is below minimum size: {apk.stat().st_size} < {minimum}")
        try:
            with zipfile.ZipFile(apk) as archive:
                bad_member = archive.testzip()
                if bad_member:
                    errors.append(f"APK ZIP integrity failed at {bad_member}")
                names = set(archive.namelist())
                if "AndroidManifest.xml" not in names:
                    errors.append("APK does not contain AndroidManifest.xml")
                if not any(name == "classes.dex" or re.fullmatch(r"classes\d+\.dex", name) for name in names):
                    errors.append("APK does not contain classes.dex")
        except zipfile.BadZipFile as exc:
            errors.append(f"APK is not a valid ZIP/APK: {exc}")

    build_path = release_dir / "BUILD_INFO.json"
    build: dict = {}
    if build_path.is_file():
        try:
            build = json.loads(build_path.read_text(encoding="utf-8"))
        except Exception as exc:
            errors.append(f"invalid BUILD_INFO.json: {exc}")
        else:
            if build.get("phase") != phase:
                errors.append("BUILD_INFO phase mismatch")
            if build.get("apk_name") != apk.name:
                errors.append("BUILD_INFO APK name mismatch")
            if apk.is_file() and build.get("apk_sha256") != sha256(apk):
                errors.append("BUILD_INFO APK SHA-256 mismatch")
            if apk.is_file() and int(build.get("apk_size", -1)) != apk.stat().st_size:
                errors.append("BUILD_INFO APK size mismatch")
            if build.get("design_system_version") != DESIGN_SYSTEM_VERSION:
                errors.append("BUILD_INFO design system mismatch")
            fixture = bool(build.get("self_test_fixture"))
            if fixture and not args.allow_self_test_fixture:
                errors.append("self-test fixture is forbidden for a real release")
            if not fixture:
                source = str(build.get("source_apk", "")).replace("\\", "/").lower()
                origin = build.get("artifact_origin")
                source_ok = (origin == "gradle" and "/build/outputs/apk/" in source) or (origin == "github-actions" and "github-actions" in source)
                if not source_ok:
                    errors.append("real release BUILD_INFO does not prove Gradle or exact GitHub Actions APK origin")
                if not re.fullmatch(r"[0-9a-fA-F]{40,64}", str(build.get("git_commit", ""))):
                    errors.append("real release BUILD_INFO has no valid Git commit")

    provenance_path = release_dir / "CI_PROVENANCE.json"
    provenance: dict = {}
    if provenance_path.is_file():
        try:
            provenance = json.loads(provenance_path.read_text(encoding="utf-8"))
        except Exception as exc:
            errors.append(f"invalid CI_PROVENANCE.json: {exc}")
        else:
            if provenance.get("phase") != phase:
                errors.append("CI provenance phase mismatch")
            if provenance.get("apk") != apk.name:
                errors.append("CI provenance APK name mismatch")
            if apk.is_file() and provenance.get("apk_sha256") != sha256(apk):
                errors.append("CI provenance APK SHA-256 mismatch")
            if build and provenance.get("commit_sha") != build.get("git_commit"):
                errors.append("CI provenance commit differs from BUILD_INFO")

    sha_path = release_dir / "SHA256SUMS.txt"
    if sha_path.is_file():
        try:
            declared = parse_sha_file(sha_path)
            expected_files = {
                path.relative_to(release_dir).as_posix()
                for path in release_dir.rglob("*")
                if path.is_file() and path.name not in {"SHA256SUMS.txt", "OWNER_ACCEPTANCE.md"}
            }
            if set(declared) != expected_files:
                errors.append(
                    "SHA256SUMS file set differs: "
                    f"missing={sorted(expected_files-set(declared))[:20]}, "
                    f"extra={sorted(set(declared)-expected_files)[:20]}"
                )
            for name, expected_hash in declared.items():
                path = release_dir / name
                if path.is_file() and sha256(path) != expected_hash:
                    errors.append(f"SHA-256 mismatch for {name}")
        except Exception as exc:
            errors.append(f"SHA256SUMS validation failed: {exc}")

    feature_map = load_yaml(ROOT / "contracts" / "feature-map.yaml").get("features") or []
    expected_ids = {feature["feature_id"] for feature in feature_map if feature["phase"] == phase}
    status = load_yaml(ROOT / "status" / f"{phase}_FEATURE_STATUS.yaml")
    status_entries = status.get("features") or []
    if {entry.get("feature_id") for entry in status_entries} != expected_ids:
        errors.append("phase status Feature IDs differ from feature map")
    for entry in status_entries:
        current = entry.get("status")
        if current not in FINAL_STATUSES:
            errors.append(f"{entry.get('feature_id')}: non-final release status {current!r}")
        elif current == "IMPLEMENTED" and not (entry.get("evidence") or []):
            errors.append(f"{entry.get('feature_id')}: IMPLEMENTED lacks evidence")
        elif current != "IMPLEMENTED" and not str(entry.get("reason", "")).strip():
            errors.append(f"{entry.get('feature_id')}: non-implemented status lacks reason")

    features_path = release_dir / "FEATURE_COMPLETION_COMPARISON.md"
    features_text = features_path.read_text(encoding="utf-8", errors="replace") if features_path.is_file() else ""
    for feature_id in expected_ids:
        if feature_id not in features_text:
            errors.append(f"FEATURE_COMPLETION_COMPARISON.md is missing {feature_id}")

    tests_path = release_dir / "AUTOMATED_TEST_REPORT.md"
    tests_text = tests_path.read_text(encoding="utf-8", errors="replace") if tests_path.is_file() else ""
    if not re.search(r"(?im)^-\s*Result:\s*PASS\s*$", tests_text):
        errors.append("AUTOMATED_TEST_REPORT.md does not record PASS")

    deployment_path = release_dir / "DEPLOYMENT_ENDPOINTS.md"
    deployment_text = deployment_path.read_text(encoding="utf-8", errors="replace") if deployment_path.is_file() else ""
    checks = [
        ("Deployment result", r"(?im)^-\s*Deployment result:\s*(SUCCESS|NOT_APPLICABLE)\s*$"),
        ("Health check result", r"(?im)^-\s*Health check result:\s*(PASS|NOT_APPLICABLE)\s*$"),
        ("Rollback result", r"(?im)^-\s*Rollback result:\s*(PASS|NOT_APPLICABLE)\s*$"),
    ]
    for label, pattern in checks:
        match = re.search(pattern, deployment_text)
        if not match:
            errors.append(f"DEPLOYMENT_ENDPOINTS.md lacks valid {label} evidence marker")
        elif label == "Rollback result" and match.group(1) == "NOT_APPLICABLE":
            if not re.search(r"(?ims)^\s*(?:Rollback evidence|回滚证据)\s*[:：]\s*\S", deployment_text):
                errors.append("DEPLOYMENT_ENDPOINTS.md must explain a NOT_APPLICABLE rollback result")

    if build.get("artifact_origin") == "github-actions":
        validate_ci_ui(release_dir, phase, errors)
    else:
        validate_ui(release_dir, phase, bool(build.get("self_test_fixture")), errors)

    acceptance_text = (release_dir / "OWNER_ACCEPTANCE.md").read_text(encoding="utf-8", errors="replace") if (release_dir / "OWNER_ACCEPTANCE.md").is_file() else ""
    if f"- Phase: {phase}" not in acceptance_text or f"- APK: {apk.name}" not in acceptance_text:
        errors.append("OWNER_ACCEPTANCE.md phase/APK was not normalized")
    if not re.search(r"(?im)^-\s*Result:\s*(PENDING|APPROVED|REJECTED)\s*$", acceptance_text):
        errors.append("OWNER_ACCEPTANCE.md has no valid Result")

    if errors:
        for error in errors:
            print(f"ERROR: {error}", file=sys.stderr)
        return 1
    fixture_note = " (self-test fixture allowed)" if build.get("self_test_fixture") else ""
    print(
        f"Release verification passed for {phase}{fixture_note}: "
        f"{len(required)} required outputs and exact CI runtime evidence."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
