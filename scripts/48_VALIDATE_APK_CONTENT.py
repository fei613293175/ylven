#!/usr/bin/env python3
"""Validate that an owner APK contains the current Android auth experience."""
from __future__ import annotations

import argparse
import sys
import zipfile
from pathlib import Path


REQUIRED_P02_MARKERS = (
    "p02-security-answer",
    "请完成小验证",
    "轻松解决每天的小问题",
)
FORBIDDEN_LEGACY_MARKERS = (
    "继续验证",
    "受控安全页面",
    "安全、统一的多模型 AI 工作台",
    "使用邮箱和登录密码建立个人工作区",
    "/security/turnstile",
)


def validate(apk: Path, phase: str) -> list[str]:
    errors: list[str] = []
    if not apk.is_file():
        return [f"APK not found: {apk}"]
    try:
        with zipfile.ZipFile(apk) as archive:
            dex_names = sorted(name for name in archive.namelist() if name.endswith(".dex"))
            if not dex_names:
                return ["APK contains no DEX files"]
            payload = b"\n".join(archive.read(name) for name in dex_names)
    except (OSError, zipfile.BadZipFile, KeyError) as exc:
        return [f"cannot inspect APK: {exc}"]

    if phase.upper() == "P02":
        for marker in REQUIRED_P02_MARKERS:
            if marker.encode("utf-8") not in payload:
                errors.append(f"APK is missing current P02 marker: {marker}")
        for marker in FORBIDDEN_LEGACY_MARKERS:
            if marker.encode("utf-8") in payload:
                errors.append(f"APK still contains legacy auth marker: {marker}")
    return errors


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--apk", required=True)
    parser.add_argument("--phase", required=True)
    args = parser.parse_args()
    errors = validate(Path(args.apk), args.phase)
    if errors:
        print("\n".join(f"FAIL: {error}" for error in errors))
        return 1
    print(f"PASS: {args.phase.upper()} APK content matches the current source flow")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
