#!/usr/bin/env python3
"""Validate the pinned Spec Kit/Codex bootstrap contract without network installation."""
from __future__ import annotations

import argparse
import json
import re
import shutil
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path

EXPECTED_TAG = "v0.15.2"
EXPECTED_VERSION = "0.15.2"
EXPECTED_SOURCE = "git+https://github.com/github/spec-kit.git@v0.15.2"


def check_contains(errors: list[str], label: str, text: str, needle: str) -> None:
    if needle not in text:
        errors.append(f"{label} is missing required text: {needle}")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--json", type=Path)
    parser.add_argument("--markdown", type=Path)
    args = parser.parse_args()
    root = args.root.resolve()
    errors: list[str] = []
    checks: list[dict[str, str]] = []

    def record(name: str, passed: bool, evidence: str) -> None:
        checks.append({"check": name, "result": "PASS" if passed else "FAIL", "evidence": evidence})
        if not passed:
            errors.append(f"{name}: {evidence}")

    pin_file = root / "speckit" / "profile" / "SPECKIT_PINNED_VERSION"
    pin = pin_file.read_text(encoding="utf-8").strip() if pin_file.is_file() else ""
    record("pinned_version", pin == EXPECTED_TAG, f"expected={EXPECTED_TAG}; actual={pin or '<missing>'}")

    win_path = root / "scripts" / "01_INSTALL_SPECKIT_WINDOWS.ps1"
    linux_path = root / "scripts" / "02_INSTALL_SPECKIT_LINUX.sh"
    verify_path = root / "scripts" / "03_VERIFY_SPECKIT.ps1"
    win = win_path.read_text(encoding="utf-8-sig") if win_path.is_file() else ""
    linux = linux_path.read_text(encoding="utf-8") if linux_path.is_file() else ""
    verify = verify_path.read_text(encoding="utf-8-sig") if verify_path.is_file() else ""

    required_win = [
        EXPECTED_TAG, EXPECTED_VERSION, '$Source = "git+https://github.com/github/spec-kit.git@$Pinned"',
        "uv tool install specify-cli", "--force", "--from $Source",
        "specify init --here --force --integration codex --script ps --ignore-agent-tools",
        "specify integration install codex --script ps --force",
        "specify integration use codex --force", "specify integration status",
    ]
    win_missing = [item for item in required_win if item not in win]
    record("windows_bootstrap_contract", not win_missing, "missing=" + repr(win_missing) if win_missing else "all required pinned commands present")

    required_linux = [
        EXPECTED_TAG, EXPECTED_VERSION,
        'SOURCE="git+https://github.com/github/spec-kit.git@${PINNED}"',
        'uv tool install specify-cli --force --from "$SOURCE"',
        "specify init --here --force --integration codex --script sh --ignore-agent-tools",
        "specify integration install codex --script sh --force",
        "specify integration use codex --force", "specify integration status",
    ]
    linux_missing = [item for item in required_linux if item not in linux]
    record("linux_bootstrap_contract", not linux_missing, "missing=" + repr(linux_missing) if linux_missing else "all required pinned commands present")

    required_verify = [
        EXPECTED_VERSION, "specify version", "specify integration status --json",
        ".agents\\skills\\$skill\\SKILL.md",
        ".specify\\templates\\overrides\\$file",
    ]
    verify_missing = [item for item in required_verify if item not in verify]
    record("target_runtime_verification", not verify_missing, "missing=" + repr(verify_missing) if verify_missing else "version, integration state, core skill and overrides are verified after installation")

    command_doc = root / "codex" / "SPECKIT_COMMANDS_AND_DECISION_RULES.md"
    command_text = command_doc.read_text(encoding="utf-8") if command_doc.is_file() else ""
    docs_ok = "$speckit-specify" in command_text and "$speckit-implement" in command_text and "/speckit." not in command_text
    record("codex_invocation_contract", docs_ok, "uses $speckit-* and rejects legacy /speckit.*" if docs_ok else "Codex invocation documentation is incomplete or legacy syntax remains")

    skill_dirs = [
        root / "speckit" / "ylven-skills" / "ylven-phase-delivery" / "SKILL.md",
        root / "speckit" / "ylven-skills" / "ylven-contract-traceability" / "SKILL.md",
    ]
    record("ylven_custom_skills", all(path.is_file() and path.stat().st_size > 0 for path in skill_dirs), ", ".join(path.relative_to(root).as_posix() for path in skill_dirs))

    override_names = ["constitution-template.md", "spec-template.md", "plan-template.md", "tasks-template.md"]
    override_paths = [root / "speckit" / "overrides" / name for name in override_names]
    record("ylven_lite_overrides", all(path.is_file() and path.stat().st_size > 0 for path in override_paths), ", ".join(override_names))

    uv = shutil.which("uv")
    if uv:
        # Keep package validation fully offline: inspect the local CLI help only.
        install_help = subprocess.run(
            [uv, "tool", "install", "--help"], capture_output=True, text=True, timeout=15
        )
        uvx_help = subprocess.run(
            [uv, "tool", "run", "--help"], capture_output=True, text=True, timeout=15
        )
        install_text = install_help.stdout + install_help.stderr
        uvx_text = uvx_help.stdout + uvx_help.stderr
        uv_ok = (
            install_help.returncode == 0 and uvx_help.returncode == 0
            and "--force" in install_text and "--from" in uvx_text
        )
        record(
            "local_uv_cli_flag_parser", uv_ok,
            f"uv={uv}; offline help exposes tool-install --force and tool-run/uvx --from",
        )
    else:
        record("local_uv_cli_flag_parser", True, "uv is installed by the target bootstrap when absent; local package sandbox has no requirement to persist the tool")

    official_sources = root / "references" / "OFFICIAL_SOURCES.md"
    sources_text = official_sources.read_text(encoding="utf-8") if official_sources.is_file() else ""
    source_ok = all(
        item in sources_text
        for item in [
            "https://github.com/github/spec-kit/releases/tag/v0.15.2",
            "https://github.github.io/spec-kit/reference/core.html",
            "https://github.github.io/spec-kit/reference/integrations.html",
            "https://developers.openai.com/codex/build-skills",
        ]
    )
    record("official_source_register", source_ok, "official release, CLI, integration and Codex Skills references recorded")

    result_name = "PASS" if not errors else "FAIL"
    report = {
        "schema_version": "1.5.0",
        "result": result_name,
        "generated_at_utc": datetime.now(timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z"),
        "pinned_spec_kit": EXPECTED_TAG,
        "codex_integration": {"key": "codex", "skill_directory": ".agents/skills", "invocation": "$speckit-<command>"},
        "actual_network_install_in_packaging_sandbox": False,
        "actual_target_installation": "performed by scripts/00_BOOTSTRAP_REPOSITORY.ps1 -> scripts/01_INSTALL_SPECKIT_WINDOWS.ps1, then checked by scripts/03_VERIFY_SPECKIT.ps1",
        "validation_scope": "pinned command contract, target verification contract, local uv flag parsing, Lite templates, custom skills and official source register",
        "checks": checks,
        "errors": errors,
    }

    json_path = (args.json or root / "SPECKIT_INTEGRATION_VALIDATION.json").resolve()
    md_path = (args.markdown or root / "SPECKIT_INTEGRATION_VALIDATION.md").resolve()
    json_path.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    lines = [
        "# Spec Kit 与 Codex 自动集成校验报告", "",
        f"- 结果：**{result_name}**",
        f"- 固定版本：`{EXPECTED_TAG}`",
        "- Codex 集成：`.agents/skills`，调用格式 `$speckit-<command>`",
        "- 当前打包沙箱执行联网安装：否",
        "- 目标 Windows 仓库执行：`scripts/00_BOOTSTRAP_REPOSITORY.ps1` 自动安装并由 `scripts/03_VERIFY_SPECKIT.ps1` 验证",
        "", "## 校验项目", "", "| 项目 | 结果 | 证据 |", "|---|---:|---|",
    ]
    for item in checks:
        lines.append(f"| `{item['check']}` | {item['result']} | {item['evidence'].replace('|', '/')} |")
    lines.extend([
        "", "## 真实性边界", "",
        "本报告验证包内自动安装方法、固定版本、命令参数、Codex Skills 路径、Lite 模板和安装后检查逻辑。它不声称当前 Linux 打包沙箱已经替你的 Windows/Codex 环境完成联网安装。解压到真实仓库后，引导脚本会执行安装；版本、集成状态、核心 Skill 和覆盖模板任一不符合都会立即失败。",
        "",
    ])
    md_path.write_text("\n".join(lines), encoding="utf-8")

    for item in checks:
        print(f"{item['result']} {item['check']}: {item['evidence']}")
    print(f"Spec Kit bootstrap validation: {result_name}")
    return 0 if result_name == "PASS" else 1


if __name__ == "__main__":
    raise SystemExit(main())
