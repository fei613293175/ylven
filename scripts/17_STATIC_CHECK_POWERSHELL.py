#!/usr/bin/env python3
"""Deterministic, dependency-free static validation for bundled PowerShell scripts.

This checker is deliberately conservative. It validates UTF-8 BOM, line endings,
lexical termination of comments/strings/here-strings, balanced delimiters, a top-level
param block, and several high-value structural invariants. It does not claim to replace
Microsoft.PowerShell.Language.Parser AST validation; the generated report records that
boundary explicitly.
"""
from __future__ import annotations

import argparse
import json
import re
import sys
from dataclasses import asdict, dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Iterable

OPEN_TO_CLOSE = {"(": ")", "[": "]", "{": "}"}
CLOSE_TO_OPEN = {v: k for k, v in OPEN_TO_CLOSE.items()}


@dataclass
class Issue:
    code: str
    message: str
    line: int
    column: int


@dataclass
class FileResult:
    path: str
    passed: bool
    byte_length: int
    line_count: int
    utf8_bom: bool
    crlf_only: bool
    top_level_param_block: bool
    strict_mode: bool
    stop_on_error: bool
    delimiter_pairs: int
    issues: list[dict]


def location(text: str, index: int) -> tuple[int, int]:
    line = text.count("\n", 0, index) + 1
    prior = text.rfind("\n", 0, index)
    column = index + 1 if prior < 0 else index - prior
    return line, column


def issue(code: str, message: str, text: str, index: int) -> Issue:
    line, column = location(text, max(0, min(index, len(text))))
    return Issue(code=code, message=message, line=line, column=column)


def first_significant_token(text: str) -> str:
    """Strip BOM/comments and return first command-like token."""
    cleaned = re.sub(r"(?s)<#.*?#>", " ", text)
    cleaned = re.sub(r"(?m)^\s*#.*$", " ", cleaned)
    match = re.search(r"[A-Za-z_][A-Za-z0-9_-]*", cleaned)
    return match.group(0).lower() if match else ""


def lexical_validate(text: str) -> tuple[list[Issue], int]:
    issues: list[Issue] = []
    stack: list[tuple[str, int]] = []
    pairs = 0
    i = 0
    n = len(text)
    state = "normal"
    state_start = 0
    here_terminator = ""

    while i < n:
        ch = text[i]
        nxt = text[i + 1] if i + 1 < n else ""

        if state == "line_comment":
            if ch == "\n":
                state = "normal"
            i += 1
            continue

        if state == "block_comment":
            if ch == "#" and nxt == ">":
                state = "normal"
                i += 2
            else:
                i += 1
            continue

        if state == "single":
            if ch == "'":
                if nxt == "'":  # PowerShell single-quoted string escape.
                    i += 2
                else:
                    state = "normal"
                    i += 1
            else:
                i += 1
            continue

        if state == "double":
            if ch == "`":  # PowerShell escape character.
                i += 2 if i + 1 < n else 1
            elif ch == '"':
                state = "normal"
                i += 1
            else:
                i += 1
            continue

        if state in {"here_single", "here_double"}:
            line_start = i == 0 or text[i - 1] == "\n"
            if line_start:
                line_end = text.find("\n", i)
                if line_end < 0:
                    line_end = n
                line_text = text[i:line_end].rstrip("\r")
                if line_text == here_terminator:
                    state = "normal"
                    i = line_end
                    continue
            i += 1
            continue

        # Normal state.
        if ch == "#":
            state = "line_comment"
            state_start = i
            i += 1
            continue
        if ch == "<" and nxt == "#":
            state = "block_comment"
            state_start = i
            i += 2
            continue
        if ch == "@" and nxt in {"'", '"'}:
            after = i + 2
            # Here-string opener must be followed only by optional horizontal space and EOL.
            eol = text.find("\n", after)
            if eol < 0:
                eol = n
            if text[after:eol].strip(" \t\r") == "":
                state = "here_single" if nxt == "'" else "here_double"
                here_terminator = "'@" if nxt == "'" else '"@'
                state_start = i
                i = eol
                continue
        if ch == "'":
            state = "single"
            state_start = i
            i += 1
            continue
        if ch == '"':
            state = "double"
            state_start = i
            i += 1
            continue
        if ch in OPEN_TO_CLOSE:
            stack.append((ch, i))
            i += 1
            continue
        if ch in CLOSE_TO_OPEN:
            if not stack:
                issues.append(issue("UNEXPECTED_CLOSER", f"Unexpected closing delimiter {ch!r}.", text, i))
            else:
                opener, opener_index = stack.pop()
                expected = OPEN_TO_CLOSE[opener]
                if ch != expected:
                    issues.append(
                        issue(
                            "MISMATCHED_DELIMITER",
                            f"Delimiter {opener!r} opened here expects {expected!r}, but found {ch!r}.",
                            text,
                            opener_index,
                        )
                    )
                else:
                    pairs += 1
            i += 1
            continue
        i += 1

    if state == "block_comment":
        issues.append(issue("UNCLOSED_BLOCK_COMMENT", "Block comment <# ... #> is not terminated.", text, state_start))
    elif state == "single":
        issues.append(issue("UNCLOSED_SINGLE_QUOTE", "Single-quoted string is not terminated.", text, state_start))
    elif state == "double":
        issues.append(issue("UNCLOSED_DOUBLE_QUOTE", "Double-quoted string is not terminated.", text, state_start))
    elif state in {"here_single", "here_double"}:
        issues.append(issue("UNCLOSED_HERE_STRING", f"Here-string is missing terminator {here_terminator!r}.", text, state_start))

    while stack:
        opener, opener_index = stack.pop()
        issues.append(
            issue(
                "UNCLOSED_DELIMITER",
                f"Opening delimiter {opener!r} is missing closing delimiter {OPEN_TO_CLOSE[opener]!r}.",
                text,
                opener_index,
            )
        )
    return issues, pairs


def validate_file(path: Path, root: Path) -> FileResult:
    raw = path.read_bytes()
    utf8_bom = raw.startswith(b"\xef\xbb\xbf")
    try:
        text = raw.decode("utf-8-sig")
    except UnicodeDecodeError as exc:
        line = raw[: exc.start].count(b"\n") + 1
        return FileResult(
            path=path.relative_to(root).as_posix(),
            passed=False,
            byte_length=len(raw),
            line_count=raw.count(b"\n") + 1,
            utf8_bom=utf8_bom,
            crlf_only=False,
            top_level_param_block=False,
            strict_mode=False,
            stop_on_error=False,
            delimiter_pairs=0,
            issues=[asdict(Issue("INVALID_UTF8", str(exc), line, 1))],
        )

    issues, pairs = lexical_validate(text)
    crlf_only = b"\n" not in raw.replace(b"\r\n", b"")
    if not utf8_bom:
        issues.append(Issue("MISSING_UTF8_BOM", "PowerShell file must use UTF-8 BOM for Windows compatibility.", 1, 1))
    if not crlf_only:
        issues.append(Issue("NON_CRLF_LINE_ENDING", "PowerShell file must use CRLF-only line endings.", 1, 1))

    top_level_param = first_significant_token(text) == "param"
    if not top_level_param:
        issues.append(Issue("MISSING_TOP_LEVEL_PARAM", "The first significant token must be a top-level param block.", 1, 1))

    strict_mode = bool(re.search(r"(?im)^\s*Set-StrictMode\s+-Version\s+Latest\s*$", text))
    if not strict_mode:
        issues.append(Issue("MISSING_STRICT_MODE", "Set-StrictMode -Version Latest is required.", 1, 1))

    stop_on_error = bool(re.search(r"(?im)^\s*\$ErrorActionPreference\s*=\s*['\"]Stop['\"]\s*$", text))
    if not stop_on_error:
        issues.append(Issue("MISSING_STOP_ON_ERROR", "$ErrorActionPreference must be set to 'Stop'.", 1, 1))

    # High-value checks for accidental truncation or unresolved conflict markers.
    for marker in ("<<<<<<<", "=======", ">>>>>>>"):
        index = text.find(marker)
        if index >= 0:
            issues.append(issue("MERGE_CONFLICT_MARKER", f"Unresolved merge marker {marker!r} found.", text, index))

    # Detect ASCII NULs, a common sign of accidental UTF-16 conversion.
    nul = text.find("\x00")
    if nul >= 0:
        issues.append(issue("NUL_CHARACTER", "NUL character detected; file may have wrong encoding.", text, nul))

    if path.name == "50_RUN_PHYSICAL_DEVICE_ACCEPTANCE.ps1":
        unsafe_mkdir = re.search(r"Invoke-Adb\s+shell\s+run-as\s+\$PackageId\s+mkdir\s+-p\s+files", text)
        if unsafe_mkdir:
            issues.append(
                issue(
                    "UNQUOTED_ADB_MKDIR_FLAG",
                    "Quote '-p' so Windows PowerShell 5 does not bind it as the PipelineVariable common parameter.",
                    text,
                    unsafe_mkdir.start(),
                )
            )

    return FileResult(
        path=path.relative_to(root).as_posix(),
        passed=not issues,
        byte_length=len(raw),
        line_count=text.count("\n") + (0 if text.endswith("\n") else 1),
        utf8_bom=utf8_bom,
        crlf_only=crlf_only,
        top_level_param_block=top_level_param,
        strict_mode=strict_mode,
        stop_on_error=stop_on_error,
        delimiter_pairs=pairs,
        issues=[asdict(item) for item in issues],
    )


def render_markdown(report: dict) -> str:
    lines = [
        "# PowerShell 静态语法与兼容性检查报告",
        "",
        f"- 结果：**{report['result']}**",
        f"- 检查时间（UTC）：`{report['generated_at_utc']}`",
        f"- PowerShell 文件：`{report['summary']['file_count']}` 个",
        f"- 通过：`{report['summary']['passed']}` 个",
        f"- 失败：`{report['summary']['failed']}` 个",
        "- 检查方法：依赖零外部包的确定性词法、字符串、注释、here-string、分隔符、编码与结构检查",
        "- 官方 PowerShell AST：**未在当前 Linux 沙箱执行**；正式 Windows 初始化时由 `scripts/03_VERIFY_SPECKIT.ps1`、`scripts/09_DOCTOR.ps1` 和实际 PowerShell 执行继续验证",
        "",
        "## 检查边界",
        "",
        "本报告可发现未闭合字符串、注释、here-string、括号、方括号、花括号、编码错误、混合换行、缺失严格模式、缺失 Stop-on-error、冲突标记和截断等高风险问题。它不冒充 `Microsoft.PowerShell.Language.Parser` 的完整 AST 解析结果。",
        "",
        "## 文件结果",
        "",
        "| 文件 | 结果 | 行数 | 分隔符对 | UTF-8 BOM | CRLF |",
        "|---|---:|---:|---:|---:|---:|",
    ]
    for item in report["files"]:
        lines.append(
            f"| `{item['path']}` | {'PASS' if item['passed'] else 'FAIL'} | {item['line_count']} | {item['delimiter_pairs']} | "
            f"{'是' if item['utf8_bom'] else '否'} | {'是' if item['crlf_only'] else '否'} |"
        )
        for found in item["issues"]:
            lines.append(
                f"\n> `{item['path']}:{found['line']}:{found['column']}` `{found['code']}`：{found['message']}\n"
            )
    lines.extend(
        [
            "",
            "## 结论",
            "",
            "所有脚本均通过本包内可执行的确定性静态检查。项目复制到 Windows 仓库后，首次运行 `scripts/00_BOOTSTRAP_REPOSITORY.ps1` 会以真实 PowerShell 解释器执行脚本，并在任何语法或运行时错误处立即停止。"
            if report["result"] == "PASS"
            else "存在失败项；不得构建最终分发 ZIP。",
            "",
        ]
    )
    return "\n".join(lines)


def main(argv: Iterable[str] | None = None) -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--json", type=Path)
    parser.add_argument("--markdown", type=Path)
    args = parser.parse_args(list(argv) if argv is not None else None)

    root = args.root.resolve()
    scripts = sorted((root / "scripts").glob("*.ps1"))
    if (root / "ylven.ps1").is_file():
        scripts.append(root / "ylven.ps1")
    if not scripts:
        print("FAIL: no PowerShell scripts found", file=sys.stderr)
        return 1

    results = [validate_file(path, root) for path in scripts]
    passed = sum(item.passed for item in results)
    report = {
        "schema_version": "1.0",
        "result": "PASS" if passed == len(results) else "FAIL",
        "generated_at_utc": datetime.now(timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z"),
        "method": "deterministic_static_lexical_delimiter_encoding_and_structure_validation",
        "official_powershell_ast_executed": False,
        "official_ast_note": "PowerShell runtime was not installed in the Linux packaging sandbox; Windows bootstrap remains the executable interpreter check.",
        "summary": {"file_count": len(results), "passed": passed, "failed": len(results) - passed},
        "files": [asdict(item) for item in results],
    }

    json_path = (args.json or (root / "POWERSHELL_SYNTAX_REPORT.json")).resolve()
    md_path = (args.markdown or (root / "POWERSHELL_SYNTAX_REPORT.md")).resolve()
    json_path.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    md_path.write_text(render_markdown(report), encoding="utf-8")

    for item in results:
        print(f"{'PASS' if item.passed else 'FAIL'} {item.path}")
        for found in item.issues:
            print(f"  {found['line']}:{found['column']} {found['code']}: {found['message']}")
    print(f"PowerShell static check: {report['result']} ({passed}/{len(results)})")
    return 0 if report["result"] == "PASS" else 1


if __name__ == "__main__":
    raise SystemExit(main())
