#!/usr/bin/env python3
from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as source:
        for chunk in iter(lambda: source.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def parse_manifest(path: Path) -> dict[str, str]:
    values: dict[str, str] = {}
    for line in path.read_text(encoding="utf-8").splitlines():
        digest, separator, name = line.partition("  ")
        if not separator or len(digest) != 64 or not name:
            raise ValueError(f"invalid SHA256SUMS line: {line!r}")
        values[name] = digest.lower()
    return values


def parse_properties(path: Path) -> dict[str, str]:
    values: dict[str, str] = {}
    for line in path.read_text(encoding="utf-8").splitlines():
        if not line or line.startswith("#"):
            continue
        key, separator, value = line.partition("=")
        if not separator or not key or key in values:
            raise ValueError(f"invalid signing migration property: {line!r}")
        values[key] = value
    return values


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--artifact-dir", required=True)
    parser.add_argument("--phase", required=True)
    parser.add_argument("--version", required=True)
    parser.add_argument("--commit", required=True)
    parser.add_argument("--source-archive-sha256", required=True)
    args = parser.parse_args()
    root = Path(args.artifact_dir).resolve()
    provenance_path = root / "服务器构建来源证明.json"
    manifest_path = root / "SHA256SUMS.txt"
    errors: list[str] = []
    if not provenance_path.is_file():
        errors.append("missing 服务器构建来源证明.json")
        provenance: dict = {}
    else:
        try:
            provenance = json.loads(provenance_path.read_text(encoding="utf-8"))
        except (OSError, json.JSONDecodeError) as exc:
            errors.append(f"invalid server provenance: {exc}")
            provenance = {}
    expected = {
        "phase": args.phase,
        "version": args.version,
        "commit_sha": args.commit,
        "source_archive_sha256": args.source_archive_sha256,
        "build_host_class": "connected_online_server",
        "build_coordinator": "shared_cross_project_fifo",
        "application_id": "cc.orbexa.ylven",
        "android_unit_tests": "PASS",
        "android_lint": "PASS",
    }
    for key, value in expected.items():
        if provenance.get(key) != value:
            errors.append(f"server provenance {key} mismatch: expected {value!r}, got {provenance.get(key)!r}")
    if provenance.get("android_build_tools") != "36.0.0":
        errors.append("server provenance must record Android Build Tools 36.0.0")
    if str(provenance.get("gradle_version")) != "8.9":
        errors.append("server provenance must record Gradle 8.9")
    if not str(provenance.get("java_version", "")).startswith("21."):
        errors.append("server provenance must record Java 21")
    for forbidden in ("workflow_run_id", "github_actions", "emulator"):
        if forbidden in provenance:
            errors.append(f"forbidden current-flow provenance field: {forbidden}")

    migration_path = ROOT / "contracts" / "signing-migrations" / f"{args.phase}.properties"
    if migration_path.is_file():
        try:
            migration = parse_properties(migration_path)
        except (OSError, ValueError) as exc:
            errors.append(str(exc))
            migration = {}
        expected_migration = {
            "phase": args.phase,
            "approved": "true",
            "install_mode": "one_time_uninstall_then_install",
            "data_preserved": "false",
        }
        for key, value in expected_migration.items():
            if migration.get(key) != value:
                errors.append(f"signing migration {key} mismatch")
        if provenance.get("signing_migration") is not True:
            errors.append("server provenance does not record the approved signing migration")
        if provenance.get("signing_migration_install_mode") != migration.get("install_mode"):
            errors.append("server provenance signing migration install mode mismatch")
        if provenance.get("signing_migration_data_preserved") is not False:
            errors.append("server provenance must record that signing migration cannot preserve app data")
        if provenance.get("previous_signing_certificate_sha256") != migration.get("previous_certificate_sha256"):
            errors.append("server provenance previous signing certificate differs from migration approval")
        if provenance.get("signing_certificate_sha256") != migration.get("new_certificate_sha256"):
            errors.append("server provenance new signing certificate differs from migration approval")
    else:
        if provenance.get("signing_migration") is not False:
            errors.append("unapproved signing migration in server provenance")
        if provenance.get("signing_certificate_sha256") != provenance.get("previous_signing_certificate_sha256"):
            errors.append("owner signing certificate differs from the previous release")

    artifact_fields = {
        "apk": "apk_sha256",
        "instrumentation_apk": "instrumentation_apk_sha256",
    }
    downloaded: dict[str, str] = {}
    for name_field, digest_field in artifact_fields.items():
        name = str(provenance.get(name_field, ""))
        path = root / name
        if not name or not path.is_file() or path.parent != root:
            errors.append(f"missing server artifact declared by {name_field}: {name!r}")
            continue
        actual = sha256(path)
        downloaded[name] = actual
        if actual != provenance.get(digest_field):
            errors.append(f"downloaded {name} SHA-256 differs from server provenance")

    if not manifest_path.is_file():
        errors.append("missing server SHA256SUMS.txt")
    else:
        try:
            manifest = parse_manifest(manifest_path)
            expected_files = {
                path.name for path in root.iterdir()
                if path.is_file() and path.name not in {"SHA256SUMS.txt", "本机下载校验证明.json"}
            }
            if set(manifest) != expected_files:
                errors.append("server SHA256SUMS.txt file set differs from downloaded outputs")
            for name, declared in manifest.items():
                if (root / name).is_file() and sha256(root / name) != declared:
                    errors.append(f"server SHA256SUMS mismatch: {name}")
        except (OSError, ValueError) as exc:
            errors.append(str(exc))

    apk_name = str(provenance.get("apk", ""))
    if apk_name and (root / apk_name).is_file():
        result = subprocess.run(
            [sys.executable, str(ROOT / "scripts" / "48_VALIDATE_APK_CONTENT.py"), "--apk", str(root / apk_name), "--phase", args.phase],
            cwd=ROOT, capture_output=True, text=True,
        )
        if result.returncode:
            errors.append((result.stdout or result.stderr).strip())
    if errors:
        print("\n".join(errors))
        return 1

    verification = {
        "schema_version": "1.0",
        "phase": args.phase,
        "version": args.version,
        "commit_sha": args.commit,
        "source_archive_sha256": args.source_archive_sha256,
        "downloaded_files_sha256": downloaded,
        "server_manifest_verified": True,
        "verified_at": dt.datetime.now(dt.timezone.utc).isoformat(timespec="seconds"),
    }
    (root / "本机下载校验证明.json").write_text(
        json.dumps(verification, ensure_ascii=False, indent=2) + "\n", encoding="utf-8"
    )
    print("PASS: exact online-server Android artifacts downloaded with matching SHA-256")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
