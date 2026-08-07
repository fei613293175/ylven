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
        formal = work / "formal"
        artifact.mkdir()
        apk = artifact / "YLVEN-1.1.0-P01.apk"
        apk.write_bytes(b"tested-apk")
        apk_hash = hashlib.sha256(apk.read_bytes()).hexdigest()
        (artifact / "自动化测试报告.md").write_text("PASS\n", encoding="utf-8")
        (artifact / "视觉差异报告.md").write_text("PASS\n", encoding="utf-8")
        (artifact / "覆盖安装证据.md").write_text("结果：PASS\n", encoding="utf-8")
        (artifact / "服务器构建来源证明.json").write_text(
            json.dumps({
                "phase": "P01", "version": "1.1.0", "commit_sha": "0" * 40,
                "apk": apk.name, "apk_sha256": apk_hash,
                "build_host_class": "connected_online_server", "build_run_id": "self-test",
            }),
            encoding="utf-8",
        )
        (artifact / "本机下载校验证明.json").write_text(
            json.dumps({"phase": "P01", "version": "1.1.0", "commit_sha": "0" * 40, "server_manifest_verified": True}),
            encoding="utf-8",
        )
        (artifact / "真机验收证据.json").write_text(
            json.dumps({
                "phase": "P01", "version": "1.1.0", "commit_sha": "0" * 40,
                "device": {"serial": "SELF_TEST_ONLY"}, "result": "PASS", "real_staging_business_flow": "PASS",
            }),
            encoding="utf-8",
        )
        (artifact / "真机日志审查.md").write_text("- 结果：PASS\n", encoding="utf-8")
        (artifact / "真实页面截图索引.csv").write_text("page_id,result\nSELF_TEST_ONLY,PASS\n", encoding="utf-8")
        (artifact / "真实页面视觉审查.json").write_text(
            json.dumps({"reviewer": "codex", "result": "PASS", "pages": []}), encoding="utf-8"
        )

        MODULE.prepare_delivery(artifact, root, destination, "P01", "1.1.0", formal_dest=formal)

        assert (destination / "P01-evidence" / "P01-W05.md").is_file()
        assert (destination / "P01-evidence" / "P01_FEATURE_STATUS.yaml").is_file()
        assert (destination / "完整测试清单.md").is_file()
        assert (destination / "原功能清单.md").is_file()
        assert (destination / "功能完成对比清单.md").is_file()
        assert (destination / "构建信息.json").is_file()
        assert (formal / "所有者验收.md").is_file()
        assert (formal / "服务器构建来源证明.json").is_file()
        assert not (destination / "P00-evidence").exists()
        assert "P01-evidence/P01-W05.md" in (destination / "校验文件_SHA256.txt").read_text(encoding="utf-8")
    print("PASS: phase-specific desktop delivery evidence")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
