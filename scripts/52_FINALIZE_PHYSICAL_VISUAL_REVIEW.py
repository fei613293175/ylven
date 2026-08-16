#!/usr/bin/env python3
from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import json
import re
from pathlib import Path


EXPECTED_PAGES = {
    "YL-A-018", "YL-A-020", "YL-A-023", "YL-A-024",
    "YL-A-026", "YL-A-033", "YL-A-034",
}
CHECKS = {"layout", "font", "color", "spacing", "icons", "interaction_state"}


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as source:
        for chunk in iter(lambda: source.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--artifact-dir", required=True)
    parser.add_argument("--review-file", required=True)
    args = parser.parse_args()
    root = Path(args.artifact_dir).resolve()
    source_review = Path(args.review_file).resolve()
    errors: list[str] = []
    try:
        review = json.loads(source_review.read_text(encoding="utf-8-sig"))
        device = json.loads((root / "真机验收证据.json").read_text(encoding="utf-8-sig"))
    except (OSError, json.JSONDecodeError) as exc:
        print(exc)
        return 1

    if review.get("phase") != device.get("phase") or review.get("version") != device.get("version"):
        errors.append("visual review phase/version differs from device evidence")
    if review.get("reviewer") != "codex":
        errors.append("production visual review must be performed directly by Codex")
    if review.get("result") != "PASS":
        errors.append("production visual review result is not PASS")
    visual_report = root / "视觉差异报告.md"
    if not visual_report.is_file() or not re.search(
        r"(?im)^Result:\s*\*\*PASS[^\n]*\*\*\s*$",
        visual_report.read_text(encoding="utf-8-sig", errors="replace") if visual_report.is_file() else "",
    ):
        errors.append("server state-matrix visual comparison is not PASS")
    pages = review.get("pages") if isinstance(review.get("pages"), list) else []
    if {str(row.get("page_id")) for row in pages} != EXPECTED_PAGES:
        errors.append("production visual review does not cover the seven P03 production pages/overlay")
    for row in pages:
        page_id = str(row.get("page_id", ""))
        screenshot_name = str(row.get("screenshot", ""))
        expected_name = f"{page_id}-PRODUCTION.png"
        if screenshot_name != f"真实页面截图/{expected_name}":
            errors.append(f"{page_id}: unexpected production screenshot path")
            continue
        screenshot = root / "真实页面截图" / expected_name
        if not screenshot.is_file() or sha256(screenshot) != row.get("sha256"):
            errors.append(f"{page_id}: production screenshot hash mismatch")
        checks = row.get("checks") if isinstance(row.get("checks"), dict) else {}
        if set(checks) != CHECKS or any(value != "PASS" for value in checks.values()):
            errors.append(f"{page_id}: layout/font/color/spacing/icons/state checks are incomplete")
        if row.get("result") != "PASS":
            errors.append(f"{page_id}: review result is not PASS")
    if errors:
        print("\n".join(errors))
        return 1

    review["reviewed_at"] = review.get("reviewed_at") or dt.datetime.now(dt.timezone.utc).isoformat(timespec="seconds")
    destination = root / "真实页面视觉审查.json"
    destination.write_text(json.dumps(review, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    device["production_page_visual_review"] = "PASS"
    device["state_matrix_visual_compare"] = "PASS"
    device["state_matrix_visual_report"] = visual_report.name
    device["state_matrix_visual_report_sha256"] = sha256(visual_report)
    device["result"] = "PASS"
    device["visual_review_file"] = destination.name
    device["visual_review_sha256"] = sha256(destination)
    device["completed_at"] = dt.datetime.now(dt.timezone.utc).isoformat(timespec="seconds")
    (root / "真机验收证据.json").write_text(
        json.dumps(device, ensure_ascii=False, indent=2) + "\n", encoding="utf-8"
    )
    status = root / "test-results" / "server-post-processing.status"
    if status.parent.is_dir():
        status.write_text("PASS\n", encoding="utf-8")
    report = root / "自动化测试报告.md"
    text = report.read_text(encoding="utf-8-sig")
    text = text.replace(
        "- Result: PENDING_CODEX_PRODUCTION_VISUAL_REVIEW",
        "- Production-page visual review: PASS（Codex 逐页核对布局、字体、颜色、间距、图标和交互状态）\n- Result: PASS",
    )
    report.write_text(text, encoding="utf-8")
    print("PASS: direct Codex review finalized all seven P03 production-page screenshots")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
