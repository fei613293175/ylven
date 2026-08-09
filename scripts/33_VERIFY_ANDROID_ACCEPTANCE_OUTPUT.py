#!/usr/bin/env python3
from __future__ import annotations

import argparse
import csv
import hashlib
import json
import re
import subprocess
import sys
from pathlib import Path

from PIL import Image, ImageStat

ROOT = Path(__file__).resolve().parents[1]


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as source:
        for chunk in iter(lambda: source.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def phase_android_screenshots(phase: str) -> list[str]:
    with (ROOT / "contracts" / "ui-state-catalog.csv").open(
        encoding="utf-8-sig", newline=""
    ) as source:
        return [
            f"{row['state_id']}.png"
            for row in csv.DictReader(source)
            if row.get("surface") == "ANDROID" and phase in row.get("phases", "").split("|")
        ]


def load_json(path: Path, errors: list[str]) -> dict:
    if not path.is_file():
        errors.append(f"missing {path.name}")
        return {}
    try:
        return json.loads(path.read_text(encoding="utf-8-sig"))
    except (OSError, json.JSONDecodeError) as exc:
        errors.append(f"invalid {path.name}: {exc}")
        return {}


def load_properties(path: Path, errors: list[str]) -> dict[str, str]:
    if not path.is_file():
        errors.append(f"missing {path.name}")
        return {}
    values: dict[str, str] = {}
    try:
        for line in path.read_text(encoding="utf-8").splitlines():
            if line and not line.startswith("#") and "=" in line:
                key, value = line.split("=", 1)
                if key in values:
                    errors.append(f"duplicate signing migration property: {key}")
                values[key] = value
    except OSError as exc:
        errors.append(f"invalid {path.name}: {exc}")
    return values


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--artifact-dir", default="build/owner-release")
    parser.add_argument("--phase", required=True)
    parser.add_argument("--version", required=True)
    parser.add_argument("--commit", required=True)
    args = parser.parse_args()
    phase = args.phase.upper()
    root = Path(args.artifact_dir).resolve()
    errors: list[str] = []

    apks = list(root.glob("*.apk"))
    if len(apks) != 1:
        errors.append(f"exactly one owner APK required, got {len(apks)}")
    apk = apks[0] if len(apks) == 1 else root / "missing.apk"
    for name in [
        "自动化测试报告.md", "视觉差异报告.md", "服务器构建来源证明.json",
        "本机下载校验证明.json", "真机验收证据.json", "真机日志审查.md",
        "覆盖安装证据.md", "截图索引.csv", "真实页面截图索引.csv", "真实页面视觉审查.json",
    ]:
        if not (root / name).is_file():
            errors.append(f"missing {name}")

    server = load_json(root / "服务器构建来源证明.json", errors)
    download = load_json(root / "本机下载校验证明.json", errors)
    device = load_json(root / "真机验收证据.json", errors)
    production_review = load_json(root / "真实页面视觉审查.json", errors)
    for proof_name, proof in (("server", server), ("download", download), ("device", device)):
        for key, expected in (("phase", phase), ("version", args.version), ("commit_sha", args.commit)):
            if proof.get(key) != expected:
                errors.append(f"{proof_name} proof {key} mismatch")
    if server.get("build_host_class") != "connected_online_server":
        errors.append("APK was not proven to come from the connected online server")
    if server.get("build_coordinator") != "shared_cross_project_fifo":
        errors.append("server build did not prove shared queue ownership")
    migration = load_properties(ROOT / "contracts" / "signing-migrations" / f"{phase}.properties", errors) if phase == "P03" else {}
    if phase == "P03":
        for key, expected_value in {
            "phase": "P03", "approved": "true", "install_mode": "one_time_uninstall_then_install",
            "data_preserved": "false", "future_upgrade_baseline": "P03",
        }.items():
            if migration.get(key) != expected_value:
                errors.append(f"P03 signing migration property {key} mismatch")
        if server.get("signing_migration") is not True:
            errors.append("server provenance does not record P03 signing migration")
        if server.get("signing_migration_install_mode") != migration.get("install_mode"):
            errors.append("server provenance signing migration mode mismatch")
        if server.get("signing_migration_data_preserved") is not False:
            errors.append("server provenance must record P03 app data was not preserved")
        if server.get("previous_signing_certificate_sha256") != migration.get("previous_certificate_sha256"):
            errors.append("P03 previous certificate does not match migration approval")
        if server.get("signing_certificate_sha256") != migration.get("new_certificate_sha256"):
            errors.append("P03 new certificate does not match migration approval")
    elif server.get("signing_migration") is not False or server.get("signing_certificate_sha256") != server.get("previous_signing_certificate_sha256"):
        errors.append("non-P03 release contains an unapproved signing migration")
    if apk.is_file() and server.get("apk_sha256") != sha256(apk):
        errors.append("owner APK SHA-256 differs from server provenance")
    downloaded = download.get("downloaded_files_sha256") or {}
    if apk.is_file() and downloaded.get(apk.name) != sha256(apk):
        errors.append("owner APK SHA-256 differs from local download proof")

    device_info = device.get("device") or {}
    if device_info.get("adb_state") != "device":
        errors.append("physical-device proof does not record exact ADB state device")
    if device_info.get("physical_device") is not True or str(device_info.get("ro_kernel_qemu")) == "1":
        errors.append("physical-device proof is missing or identifies a simulator")
    size_match = re.fullmatch(
        r"Physical size:\s*(\d+)x(\d+)",
        str(device_info.get("physical_size", "")).strip(),
        flags=re.IGNORECASE,
    )
    native_size = tuple(map(int, size_match.groups())) if size_match else None
    if native_size is None or min(native_size) <= 0:
        errors.append("physical-device proof does not contain a valid native physical size")
    density_match = re.fullmatch(
        r"Physical density:\s*(\d+)",
        str(device_info.get("density", "")).strip(),
        flags=re.IGNORECASE,
    )
    if density_match is None or int(density_match.group(1)) <= 0:
        errors.append("physical-device proof does not contain a valid native physical density")
    if device_info.get("display_size_override") is not False:
        errors.append("physical-device proof does not explicitly reject a display-size override")
    if device_info.get("density_override") is not False:
        errors.append("physical-device proof does not explicitly reject a display-density override")
    queue = device.get("queue") or {}
    if queue.get("type") != "shared_fifo" or queue.get("lock_held_for_entire_run") is not True:
        errors.append("physical-device shared FIFO ownership was not proven")
    if device.get("real_device_ui_flow") != "PASS":
        errors.append("real-device UI interaction flow did not pass")
    if device.get("real_staging_business_flow") != "PASS":
        errors.append("real staging business flow did not pass")
    if device.get("log_review") != "PASS" or device.get("result") != "PASS":
        errors.append("physical-device acceptance is not fully PASS")
    if device.get("state_matrix_visual_compare") != "PASS":
        errors.append("physical-device state-matrix visual comparison did not pass")
    if device.get("production_page_visual_review") != "PASS":
        errors.append("direct production-page visual review did not pass")
    transition = device.get("install_transition") or {}
    if phase == "P03":
        if transition.get("mode") != "one_time_uninstall_then_install" or transition.get("data_preserved") is not False:
            errors.append("P03 device evidence does not prove the approved destructive signing migration")
    elif transition and (transition.get("mode") != "adb_install_r" or transition.get("data_preserved") is not True):
        errors.append("non-P03 device evidence does not prove same-signature adb install -r")

    expected = phase_android_screenshots(phase)
    actual = {path.name for path in (root / "截图").glob("*.png")}
    if actual != set(expected):
        missing = sorted(set(expected) - actual)
        unexpected = sorted(actual - set(expected))
        if missing:
            errors.append(f"missing physical-device screenshots: {', '.join(missing)}")
        if unexpected:
            errors.append(f"unexpected physical-device screenshots: {', '.join(unexpected)}")
    digests: dict[str, str] = {}
    for name in expected:
        path = root / "截图" / name
        if not path.is_file():
            continue
        try:
            with Image.open(path) as source:
                shot = source.convert("RGB")
                if native_size is not None and shot.size != native_size:
                    errors.append(
                        f"physical-device screenshot is not native size: {name} "
                        f"{shot.size}, expected {native_size}"
                    )
                if all(high - low < 8 for low, high in ImageStat.Stat(shot).extrema):
                    errors.append(f"physical-device screenshot is blank: {name}")
        except Exception as exc:
            errors.append(f"invalid physical-device screenshot {name}: {exc}")
            continue
        digest = sha256(path)
        if digest in digests:
            errors.append(f"physical-device screenshots are byte-identical: {digests[digest]}, {name}")
        else:
            digests[digest] = name

    expected_production_pages = {
        "YL-A-018", "YL-A-019", "YL-A-020", "YL-A-021",
        "YL-A-022", "YL-A-023", "YL-A-031",
    }
    production_dir = root / "真实页面截图"
    production_files = {path.name for path in production_dir.glob("*.png")}
    expected_production_files = {f"{page_id}-PRODUCTION.png" for page_id in expected_production_pages}
    if production_files != expected_production_files:
        errors.append("production-page screenshot set differs from the seven required P03 pages/overlay")
    production_digests: dict[str, str] = {}
    for name in sorted(expected_production_files):
        path = production_dir / name
        if not path.is_file():
            continue
        try:
            with Image.open(path) as source:
                shot = source.convert("RGB")
                if native_size is not None and shot.size != native_size:
                    errors.append(
                        f"production-page screenshot is not native size: {name} "
                        f"{shot.size}, expected {native_size}"
                    )
                if all(high - low < 8 for low, high in ImageStat.Stat(shot).extrema):
                    errors.append(f"production-page screenshot is blank: {name}")
        except Exception as exc:
            errors.append(f"invalid production-page screenshot {name}: {exc}")
            continue
        digest = sha256(path)
        if digest in production_digests:
            errors.append(f"production-page screenshots are byte-identical: {production_digests[digest]}, {name}")
        else:
            production_digests[digest] = name
    if production_review.get("reviewer") != "codex" or production_review.get("result") != "PASS":
        errors.append("production-page screenshots lack a direct PASS review by Codex")
    reviewed_pages = production_review.get("pages") if isinstance(production_review.get("pages"), list) else []
    if {str(row.get("page_id")) for row in reviewed_pages} != expected_production_pages:
        errors.append("production-page visual review page set is incomplete")
    if (root / "真实页面视觉审查.json").is_file() and device.get("visual_review_sha256") != sha256(root / "真实页面视觉审查.json"):
        errors.append("device evidence visual-review hash mismatch")

    visual = (root / "视觉差异报告.md").read_text(encoding="utf-8-sig", errors="replace") if (root / "视觉差异报告.md").is_file() else ""
    if not re.search(r"(?im)^Result:\s*\*\*PASS[^\n]*\*\*\s*$", visual):
        errors.append("physical-device visual report is not PASS")
    logs = (root / "真机日志审查.md").read_text(encoding="utf-8-sig", errors="replace") if (root / "真机日志审查.md").is_file() else ""
    if not re.search(r"(?m)^- 结果：PASS\s*$", logs):
        errors.append("physical-device crash/ANR/log review is not PASS")

    if apk.is_file():
        content_gate = subprocess.run(
            [sys.executable, str(ROOT / "scripts" / "48_VALIDATE_APK_CONTENT.py"), "--apk", str(apk), "--phase", phase],
            capture_output=True, text=True,
        )
        if content_gate.returncode:
            errors.append((content_gate.stdout or content_gate.stderr).strip())
    if errors:
        print("\n".join(errors))
        return 1
    print(f"PASS: {phase} exact server APK passed complete physical-device and staging acceptance")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
