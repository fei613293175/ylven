#!/usr/bin/env python3
"""Deterministic Work Packet and owner-facing release state controller.

The controller deliberately separates immutable planning definitions from runtime state:
- contracts/work-packet-map.yaml: static scope only
- status/WORK_PACKET_STATUS.yaml: mutable packet lifecycle
- CURRENT_PHASE.yaml / CURRENT_WORK_PACKET.yaml: current pointers
- status/WORK_PACKET_HISTORY.jsonl / RELEASE_LEDGER.jsonl: append-only hash chains

Chat text never chooses the next packet or release. This script does.
"""
from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import json
import os
import re
import subprocess
import sys
import tempfile
from pathlib import Path
from typing import Any

import yaml

ROOT = Path(__file__).resolve().parents[1]
PHASE_PATH = ROOT / "CURRENT_PHASE.yaml"
PACKET_PATH = ROOT / "CURRENT_WORK_PACKET.yaml"
DEFINITION_PATH = ROOT / "contracts/work-packet-map.yaml"
RUNTIME_PATH = ROOT / "status/WORK_PACKET_STATUS.yaml"
MATRIX_PATH = ROOT / "contracts/release-version-matrix.yaml"
PACKET_HISTORY = ROOT / "status/WORK_PACKET_HISTORY.jsonl"
RELEASE_LEDGER = ROOT / "status/RELEASE_LEDGER.jsonl"
SUMMARY_PATH = ROOT / "status/PROJECT_STATE_SUMMARY.md"
LOCAL_DIR = ROOT / ".ylven-local"

PACKET_FINAL = {"CLOSED", "DEFERRED_WITH_REASON"}
PACKET_OPEN = {"PLANNED", "TODO", "DOING", "BLOCKED"}
FEATURE_FINAL = {"IMPLEMENTED", "DEFERRED_WITH_REASON", "BLOCKED_EXTERNAL"}
PHASE_OPEN = {"TODO", "DOING", "BLOCKED", "READY_FOR_RELEASE"}


def now() -> str:
    return dt.datetime.now(dt.timezone(dt.timedelta(hours=8))).isoformat(timespec="seconds")


def load_yaml(path: Path) -> dict[str, Any]:
    if not path.exists():
        return {}
    data = yaml.safe_load(path.read_text(encoding="utf-8-sig"))
    return data if isinstance(data, dict) else {}


def atomic_yaml(path: Path, data: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.NamedTemporaryFile("w", encoding="utf-8", dir=path.parent, delete=False) as handle:
        yaml.safe_dump(data, handle, allow_unicode=True, sort_keys=False, width=180)
        temp = Path(handle.name)
    temp.replace(path)


def run(command: list[str], *, check: bool = True) -> subprocess.CompletedProcess[str]:
    result = subprocess.run(command, cwd=ROOT, text=True, capture_output=True)
    if check and result.returncode:
        detail = (result.stdout + result.stderr).strip()
        raise SystemExit(detail or f"Command failed: {' '.join(command)}")
    return result


def git(*arguments: str, check: bool = True) -> subprocess.CompletedProcess[str]:
    return run(["git", *arguments], check=check)


def has_git() -> bool:
    return (ROOT / ".git").exists() and git("rev-parse", "--is-inside-work-tree", check=False).returncode == 0


def head_sha() -> str | None:
    if not has_git():
        return None
    result = git("rev-parse", "HEAD", check=False)
    return result.stdout.strip() if result.returncode == 0 else None


def branch_name() -> str | None:
    if not has_git():
        return None
    result = git("branch", "--show-current", check=False)
    value = result.stdout.strip()
    return value or None


def require_clean_git() -> str:
    if not has_git():
        raise SystemExit("A real Git repository is required before closing a Work Packet or release. Run bootstrap first.")
    sha = head_sha()
    if not sha:
        raise SystemExit("At least one committed Git revision is required before closure.")
    dirty = git("status", "--porcelain", check=False).stdout.strip()
    if dirty:
        raise SystemExit(
            "Commit the implementation, tests, Feature evidence and current state before closure. "
            "The worktree must be clean.\n" + dirty
        )
    return sha


def canonical_hash(record: dict[str, Any]) -> str:
    body = {key: value for key, value in record.items() if key != "record_hash"}
    encoded = json.dumps(body, ensure_ascii=False, sort_keys=True, separators=(",", ":")).encode("utf-8")
    return hashlib.sha256(encoded).hexdigest()


def read_jsonl(path: Path) -> list[dict[str, Any]]:
    if not path.exists():
        return []
    rows: list[dict[str, Any]] = []
    for number, line in enumerate(path.read_text(encoding="utf-8-sig").splitlines(), 1):
        if not line.strip():
            continue
        try:
            row = json.loads(line)
        except json.JSONDecodeError as exc:
            raise SystemExit(f"{path.relative_to(ROOT)} line {number}: invalid JSON: {exc}") from exc
        if not isinstance(row, dict):
            raise SystemExit(f"{path.relative_to(ROOT)} line {number}: object required")
        rows.append(row)
    return rows


def append_chain(path: Path, record: dict[str, Any]) -> dict[str, Any]:
    rows = read_jsonl(path)
    record = dict(record)
    record["previous_hash"] = rows[-1].get("record_hash") if rows else None
    record["record_hash"] = canonical_hash(record)
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("a", encoding="utf-8") as handle:
        handle.write(json.dumps(record, ensure_ascii=False, sort_keys=True) + "\n")
    return record


def validate_chain(path: Path) -> None:
    previous: str | None = None
    for number, row in enumerate(read_jsonl(path), 1):
        if row.get("previous_hash") != previous:
            raise SystemExit(f"{path.relative_to(ROOT)} line {number}: previous_hash mismatch")
        if row.get("record_hash") != canonical_hash(row):
            raise SystemExit(f"{path.relative_to(ROOT)} line {number}: record_hash mismatch")
        previous = row.get("record_hash")


def packet_id(row: dict[str, Any]) -> str:
    value = row.get("work_packet_id") or row.get("id")
    if not value:
        raise SystemExit("A Work Packet definition is missing work_packet_id")
    return str(value)


def packet_phase(row: dict[str, Any]) -> str:
    value = row.get("phase") or row.get("phase_id")
    if not value:
        raise SystemExit(f"{packet_id(row)}: phase is missing")
    return str(value)


def packet_sequence(row: dict[str, Any]) -> int:
    return int(row.get("sequence", 999999))


def definitions() -> list[dict[str, Any]]:
    return list(load_yaml(DEFINITION_PATH).get("work_packets") or [])


def runtime_document() -> dict[str, Any]:
    return load_yaml(RUNTIME_PATH)


def runtime_rows(document: dict[str, Any]) -> list[dict[str, Any]]:
    return list(document.get("work_packets") or [])


def definition_index() -> dict[str, dict[str, Any]]:
    rows = definitions()
    index = {packet_id(row): row for row in rows}
    if len(index) != len(rows):
        raise SystemExit("contracts/work-packet-map.yaml contains duplicate Work Packet IDs")
    return index


def runtime_index(document: dict[str, Any]) -> dict[str, dict[str, Any]]:
    rows = runtime_rows(document)
    index = {packet_id(row): row for row in rows}
    if len(index) != len(rows):
        raise SystemExit("status/WORK_PACKET_STATUS.yaml contains duplicate Work Packet IDs")
    return index


def packets_for_phase(rows: list[dict[str, Any]], phase: str) -> list[dict[str, Any]]:
    return sorted([row for row in rows if packet_phase(row) == phase], key=packet_sequence)


def phase_order() -> list[str]:
    matrix = load_yaml(MATRIX_PATH)
    items = matrix.get("phases") or matrix.get("releases") or []
    result: list[str] = []
    for item in items:
        value = item.get("phase") or item.get("phase_id")
        if value:
            result.append(str(value))
    if not result:
        result = sorted({packet_phase(row) for row in definitions()})
    return result


def release_entry(phase: str) -> dict[str, Any]:
    matrix = load_yaml(MATRIX_PATH)
    for item in matrix.get("phases") or matrix.get("releases") or []:
        if str(item.get("phase") or item.get("phase_id")) == phase:
            return item
    raise SystemExit(f"contracts/release-version-matrix.yaml has no release for {phase}")


def project_code() -> str:
    context = load_yaml(ROOT / "PROJECT_CONTEXT.yaml")
    project = context.get("project") or {}
    value = project.get("code") or project.get("name") or context.get("project_code") or context.get("project_name") or "app"
    slug = re.sub(r"[^a-z0-9]+", "-", str(value).lower()).strip("-")
    return slug or "app"


def save_state(phase_state: dict[str, Any], packet_state: dict[str, Any], runtime: dict[str, Any]) -> None:
    phase_state["phase_id"] = phase_state.get("current_phase") or phase_state.get("phase_id")
    phase_state["current_phase"] = phase_state.get("phase_id")
    phase_state["current_work_packet"] = packet_state.get("work_packet_id")
    phase_state["updated_at"] = now()
    packet_state["updated_at"] = now()
    runtime["updated_at"] = now()
    atomic_yaml(PHASE_PATH, phase_state)
    atomic_yaml(PACKET_PATH, packet_state)
    atomic_yaml(RUNTIME_PATH, runtime)


def current_state() -> tuple[dict[str, Any], dict[str, Any], dict[str, Any]]:
    return load_yaml(PHASE_PATH), load_yaml(PACKET_PATH), runtime_document()


def state_files() -> list[Path]:
    return [PHASE_PATH, PACKET_PATH, RUNTIME_PATH, PACKET_HISTORY, RELEASE_LEDGER, SUMMARY_PATH]


def git_commit_state(message: str, *, push: bool = True, tag: str | None = None) -> str | None:
    if not has_git():
        return None
    relative = [str(path.relative_to(ROOT)) for path in state_files() if path.exists()]
    git("add", "--", *relative)
    staged = git("diff", "--cached", "--quiet", check=False)
    if staged.returncode == 1:
        git("commit", "-m", message)
    elif staged.returncode not in {0, 1}:
        raise SystemExit("Unable to inspect staged state changes")
    state_sha = head_sha()
    if not push:
        return state_sha
    remote = git("remote", "get-url", "origin", check=False)
    if remote.returncode != 0:
        return state_sha
    push_result = git("push", "origin", "HEAD", check=False)
    tag_result = None
    if tag:
        tag_result = git("push", "origin", tag, check=False)
    if push_result.returncode or (tag_result and tag_result.returncode):
        LOCAL_DIR.mkdir(parents=True, exist_ok=True)
        details = (push_result.stdout + push_result.stderr).strip()
        if tag_result and tag_result.returncode:
            details += "\n" + (tag_result.stdout + tag_result.stderr).strip()
        (LOCAL_DIR / "PUSH_PENDING.md").write_text(
            "# State Push Pending\n\n"
            f"- State commit: `{state_sha}`\n"
            f"- Branch: `{branch_name() or 'detached'}`\n"
            f"- Tag: `{tag or 'none'}`\n\n"
            "Authenticate Git and run `git push origin HEAD` (and the tag push if present).\n\n"
            f"```text\n{details}\n```\n",
            encoding="utf-8",
        )
        print("WARNING: state was closed locally but push is pending; see .ylven-local/PUSH_PENDING.md", file=sys.stderr)
    else:
        pending = LOCAL_DIR / "PUSH_PENDING.md"
        if pending.exists():
            pending.unlink()
    return state_sha


def feature_status_path(phase: str) -> Path:
    return ROOT / "status" / f"{phase}_FEATURE_STATUS.yaml"


def check_feature_closure(definition: dict[str, Any]) -> dict[str, int]:
    path = feature_status_path(packet_phase(definition))
    document = load_yaml(path)
    index = {str(item.get("feature_id")): item for item in document.get("features") or []}
    errors: list[str] = []
    counts: dict[str, int] = {}
    for feature_id in definition.get("feature_ids") or []:
        item = index.get(str(feature_id))
        if item is None:
            errors.append(f"{feature_id}: missing from {path.relative_to(ROOT)}")
            continue
        status = str(item.get("status"))
        counts[status] = counts.get(status, 0) + 1
        if status not in FEATURE_FINAL:
            errors.append(f"{feature_id}: status {status!r} is not final")
        if not item.get("evidence"):
            errors.append(f"{feature_id}: evidence is empty")
        if status in {"DEFERRED_WITH_REASON", "BLOCKED_EXTERNAL"} and not str(item.get("reason") or "").strip():
            errors.append(f"{feature_id}: a reason is required for {status}")
    if errors:
        raise SystemExit("Work Packet cannot close:\n" + "\n".join(errors))
    return counts


def owner_acceptance(phase: str, *, legacy_exception: bool = False) -> str | None:
    directory = ROOT / "dist" / "releases" / phase
    names = ["所有者验收.md"]
    if legacy_exception:
        names.append("OWNER_ACCEPTANCE.md")
    for name in names:
        path = directory / name
        if not path.exists():
            continue
        match = re.search(r"^-\s*Result:\s*(\S+)\s*$", path.read_text(encoding="utf-8", errors="replace"), re.M | re.I)
        if match:
            return match.group(1).upper()
    return None


def document_feature_ids(text: str) -> set[str]:
    found = set(re.findall(r"\bP\d{2}-\d{3}\b", text))
    for start, end in re.findall(r"\b(P\d{2}-\d{3})\s*\.\.\s*(P\d{2}-\d{3})\b", text):
        phase_a, number_a = start.split("-")
        phase_b, number_b = end.split("-")
        if phase_a == phase_b:
            found.update(f"{phase_a}-{number:03d}" for number in range(int(number_a), int(number_b) + 1))
    return found


def release_evidence(phase: str, *, legacy_exception: bool = False) -> tuple[Path, dict[str, Any], dict[str, Any]]:
    directory = ROOT / "dist" / "releases" / phase
    if legacy_exception:
        required = [
            "BUILD_INFO.json", "CI_PROVENANCE.json", "OWNER_ACCEPTANCE.md", "SHA256SUMS.txt",
            "FEATURES_ORIGINAL.md", "FEATURE_COMPLETION_COMPARISON.md", "OWNER_TEST_CHECKLIST.md",
            "DEPLOYMENT_ENDPOINTS.md", "DOMAIN_DNS_STATUS.md", "ADMIN_ACCESS.md",
        ]
        feature_documents = ("FEATURES_ORIGINAL.md", "FEATURE_COMPLETION_COMPARISON.md", "OWNER_TEST_CHECKLIST.md")
        deployment_documents = ("DEPLOYMENT_ENDPOINTS.md", "DOMAIN_DNS_STATUS.md")
        build_name = "BUILD_INFO.json"
        provenance_name = "CI_PROVENANCE.json"
    else:
        required = [
            "构建信息.json", "服务器构建来源证明.json", "本机下载校验证明.json",
            "真机验收证据.json", "真机日志审查.md", "真实页面截图索引.csv", "真实页面视觉审查.json",
            "所有者验收.md", "校验文件_SHA256.txt",
            "原功能清单.md", "功能完成对比清单.md", "完整测试清单.md",
            "部署证据.md", "域名DNS状态.md", "管理后台实测证据.md", "覆盖安装证据.md",
        ]
        feature_documents = ("原功能清单.md", "功能完成对比清单.md", "完整测试清单.md")
        deployment_documents = ("部署证据.md", "域名DNS状态.md")
        build_name = "构建信息.json"
        provenance_name = "服务器构建来源证明.json"
    missing = [name for name in required if not (directory / name).is_file()]
    if missing:
        raise SystemExit(f"{phase}: missing release evidence: {', '.join(missing)}")
    try:
        build = json.loads((directory / build_name).read_text(encoding="utf-8-sig"))
        provenance = json.loads((directory / provenance_name).read_text(encoding="utf-8-sig"))
    except json.JSONDecodeError as exc:
        raise SystemExit(f"{phase}: invalid JSON release evidence: {exc}") from exc
    expected_ids = {
        str(item.get("feature_id"))
        for item in load_yaml(ROOT / "status" / f"{phase}_FEATURE_STATUS.yaml").get("features", [])
    }
    for name in feature_documents:
        text = (directory / name).read_text(encoding="utf-8", errors="replace")
        missing_ids = sorted(expected_ids - document_feature_ids(text))
        if missing_ids:
            raise SystemExit(f"{phase}: {name} omits Feature IDs: {', '.join(missing_ids)}")
    for name in deployment_documents:
        text = (directory / name).read_text(encoding="utf-8", errors="replace")
        if not re.search(r"(?im)^-\s*Result:\s*(PASS|DEPLOYED_STAGING|SUCCESS)\s*$", text):
            raise SystemExit(f"{phase}: {name} must contain a successful Result marker")
    if not legacy_exception:
        download = json.loads((directory / "本机下载校验证明.json").read_text(encoding="utf-8-sig"))
        device = json.loads((directory / "真机验收证据.json").read_text(encoding="utf-8-sig"))
        if provenance.get("build_host_class") != "connected_online_server":
            raise SystemExit(f"{phase}: release provenance is not an online-server build")
        if download.get("server_manifest_verified") is not True:
            raise SystemExit(f"{phase}: local server-artifact SHA-256 verification is missing")
        if device.get("result") != "PASS" or device.get("real_staging_business_flow") != "PASS" or device.get("production_page_visual_review") != "PASS":
            raise SystemExit(f"{phase}: physical-device and real-staging acceptance must be PASS")
    return directory, build, provenance


def first_value(*values: Any) -> Any:
    return next((value for value in values if value not in (None, "")), None)


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def write_summary(phase_state: dict[str, Any], packet_state: dict[str, Any], next_action: str) -> None:
    releases = [row for row in read_jsonl(RELEASE_LEDGER) if row.get("event") == "RELEASE_CLOSED"]
    packets = [row for row in read_jsonl(PACKET_HISTORY) if row.get("event") in {"WORK_PACKET_CLOSED", "WORK_PACKET_DEFERRED"}]
    phase = phase_state.get("phase_id") or phase_state.get("current_phase")
    current = packet_state.get("work_packet_id")
    lines = [
        "# Project State Summary",
        "",
        f"- Generated: `{now()}`",
        f"- Current phase: `{phase}`",
        f"- Phase status: `{phase_state.get('status')}`",
        f"- Current Work Packet: `{current or 'NONE'}`",
        f"- Work Packet status: `{packet_state.get('status')}`",
        f"- Last closed Work Packet: `{packet_state.get('last_closed_work_packet') or 'none'}`",
        f"- Last closed phase: `{phase_state.get('last_closed_phase') or 'none'}`",
        f"- Last closed version: `{phase_state.get('last_closed_version') or 'none'}`",
        f"- Closed/deferred Work Packets recorded: `{len(packets)}`",
        f"- Closed releases recorded: `{len(releases)}`",
        f"- Git branch: `{branch_name() or 'not initialized'}`",
        f"- Git HEAD: `{head_sha() or 'not initialized'}`",
        "",
        "## Exact next action",
        "",
        next_action,
        "",
        "## New-conversation rule",
        "",
        "Run the repository state `resume` command before planning or editing. Chat memory and phrases such as “next version” are never authoritative.",
        "",
    ]
    SUMMARY_PATH.parent.mkdir(parents=True, exist_ok=True)
    SUMMARY_PATH.write_text("\n".join(lines), encoding="utf-8")
    print(
        json.dumps(
            {
                "phase": phase,
                "phase_status": phase_state.get("status"),
                "work_packet": current,
                "work_packet_status": packet_state.get("status"),
                "last_closed_phase": phase_state.get("last_closed_phase"),
                "last_closed_version": phase_state.get("last_closed_version"),
                "next_action": next_action,
            },
            ensure_ascii=False,
            indent=2,
        )
    )


def validate() -> tuple[dict[str, Any], dict[str, Any], dict[str, Any], dict[str, dict[str, Any]], dict[str, dict[str, Any]]]:
    required = [PHASE_PATH, PACKET_PATH, DEFINITION_PATH, RUNTIME_PATH, MATRIX_PATH, PACKET_HISTORY, RELEASE_LEDGER]
    missing = [str(path.relative_to(ROOT)) for path in required if not path.exists()]
    if missing:
        raise SystemExit("Missing state authorities: " + ", ".join(missing))
    validate_chain(PACKET_HISTORY)
    validate_chain(RELEASE_LEDGER)

    phase_state, packet_state, runtime = current_state()
    definitions_by_id = definition_index()
    runtime_by_id = runtime_index(runtime)
    if set(definitions_by_id) != set(runtime_by_id):
        raise SystemExit(
            "Work Packet runtime coverage differs from planning definitions; "
            f"missing={sorted(set(definitions_by_id)-set(runtime_by_id))}, "
            f"extra={sorted(set(runtime_by_id)-set(definitions_by_id))}"
        )
    for identifier, row in runtime_by_id.items():
        definition = definitions_by_id[identifier]
        if packet_phase(row) != packet_phase(definition) or packet_sequence(row) != packet_sequence(definition):
            raise SystemExit(f"{identifier}: runtime phase/sequence differs from planning definition")
        status = row.get("status")
        if status not in PACKET_OPEN | PACKET_FINAL:
            raise SystemExit(f"{identifier}: unsupported runtime status {status!r}")

    phase = str(phase_state.get("phase_id") or phase_state.get("current_phase") or "")
    if not phase:
        raise SystemExit("CURRENT_PHASE.yaml has no phase_id/current_phase")
    if phase_state.get("current_phase") not in {None, phase}:
        raise SystemExit("CURRENT_PHASE.yaml current_phase and phase_id differ")
    if packet_state.get("phase_id") != phase:
        raise SystemExit("CURRENT_PHASE.yaml and CURRENT_WORK_PACKET.yaml disagree on phase")
    current = packet_state.get("work_packet_id")
    if current:
        if current not in definitions_by_id:
            raise SystemExit(f"Current Work Packet {current} is not defined")
        if packet_phase(definitions_by_id[current]) != phase:
            raise SystemExit(f"Current Work Packet {current} does not belong to {phase}")
        if packet_state.get("status") != runtime_by_id[current].get("status"):
            raise SystemExit(f"Current Work Packet status differs from status/WORK_PACKET_STATUS.yaml for {current}")
    if phase_state.get("current_work_packet") != current:
        raise SystemExit("CURRENT_PHASE.yaml current_work_packet differs from CURRENT_WORK_PACKET.yaml")

    history = read_jsonl(PACKET_HISTORY)
    final_history: dict[str, dict[str, Any]] = {}
    for row in history:
        if row.get("event") in {"WORK_PACKET_CLOSED", "WORK_PACKET_DEFERRED"}:
            identifier = str(row.get("work_packet_id"))
            if identifier in final_history:
                raise SystemExit(f"{identifier}: more than one final Work Packet history record")
            final_history[identifier] = row
    for identifier, row in runtime_by_id.items():
        if row.get("status") in PACKET_FINAL and identifier not in final_history:
            raise SystemExit(f"{identifier}: final runtime status without append-only history")
        if row.get("status") not in PACKET_FINAL and identifier in final_history:
            raise SystemExit(f"{identifier}: history says final but runtime status is {row.get('status')}")
        if identifier in final_history and row.get("closed_commit") != final_history[identifier].get("commit"):
            raise SystemExit(f"{identifier}: runtime closed_commit differs from history")

    phase_runtime = packets_for_phase(list(runtime_by_id.values()), phase)
    unfinished = [row for row in phase_runtime if row.get("status") not in PACKET_FINAL]
    if phase_state.get("status") == "READY_FOR_RELEASE":
        if unfinished or current is not None or packet_state.get("status") != "PHASE_COMPLETE":
            raise SystemExit(f"{phase}: READY_FOR_RELEASE requires all packets final and no current packet")
    elif phase_state.get("status") == "DONE":
        if current is not None or packet_state.get("status") != "DONE":
            raise SystemExit("DONE project must have no current Work Packet")
    else:
        if phase_state.get("status") not in PHASE_OPEN:
            raise SystemExit(f"Unsupported phase status {phase_state.get('status')!r}")
        if not unfinished:
            raise SystemExit(f"{phase}: all packets are final but phase is not READY_FOR_RELEASE")
        expected = packet_id(unfinished[0])
        if current != expected:
            raise SystemExit(f"Current Work Packet must be first unfinished packet {expected}, not {current}")

    releases = [row for row in read_jsonl(RELEASE_LEDGER) if row.get("event") == "RELEASE_CLOSED"]
    seen_phases: set[str] = set()
    order = phase_order()
    last_index = -1
    for row in releases:
        release_phase = str(row.get("phase_id"))
        if release_phase in seen_phases:
            raise SystemExit(f"{release_phase}: duplicate final release ledger entry")
        seen_phases.add(release_phase)
        if release_phase not in order:
            raise SystemExit(f"{release_phase}: release ledger phase is not planned")
        index = order.index(release_phase)
        if index <= last_index:
            raise SystemExit("Release ledger phases are not strictly ordered")
        last_index = index
        matrix = release_entry(release_phase)
        if str(row.get("version_name")) != str(matrix.get("version_name")):
            raise SystemExit(f"{release_phase}: release ledger version differs from release matrix")
        if row.get("owner_result") != "APPROVED":
            raise SystemExit(f"{release_phase}: closed release lacks owner approval")

    return phase_state, packet_state, runtime, definitions_by_id, runtime_by_id


def init_state() -> None:
    phase_state, packet_state, runtime = current_state()
    definitions_by_id = definition_index()
    runtime_by_id = runtime_index(runtime)
    if set(definitions_by_id) != set(runtime_by_id):
        raise SystemExit("Initialize status/WORK_PACKET_STATUS.yaml from the Work Packet definitions before running init")
    phase = str(phase_state.get("phase_id") or phase_state.get("current_phase") or phase_order()[0])
    phase_state.setdefault("schema_version", "1.6.0")
    phase_state["phase_id"] = phase
    phase_state["current_phase"] = phase
    phase_rows = packets_for_phase(list(runtime_by_id.values()), phase)
    unfinished = [row for row in phase_rows if row.get("status") not in PACKET_FINAL]
    if not unfinished:
        packet_state = {
            "schema_version": "1.0",
            "phase_id": phase,
            "work_packet_id": None,
            "status": "PHASE_COMPLETE",
            "started_at": None,
            "started_commit": None,
            "last_closed_work_packet": packet_state.get("last_closed_work_packet"),
        }
        phase_state["status"] = "READY_FOR_RELEASE"
    else:
        first = unfinished[0]
        if first.get("status") == "PLANNED":
            first["status"] = "TODO"
        packet_state = {
            "schema_version": "1.0",
            "phase_id": phase,
            "work_packet_id": packet_id(first),
            "status": first.get("status"),
            "started_at": first.get("started_at"),
            "started_commit": first.get("started_commit"),
            "last_closed_work_packet": packet_state.get("last_closed_work_packet"),
        }
        phase_state.setdefault("status", "TODO")
    save_state(phase_state, packet_state, runtime)
    write_summary(phase_state, packet_state, f"Continue exactly `{packet_state.get('work_packet_id') or 'release closure'}`.")
    validate()


def resume() -> None:
    phase_state, packet_state, _runtime, _definitions, _runtime_by_id = validate()
    phase = phase_state.get("phase_id")
    current = packet_state.get("work_packet_id")
    if phase_state.get("status") == "READY_FOR_RELEASE":
        action = (
            f"All Work Packets in `{phase}` are final. Build the exact clean commit on the connected online server, verify the downloaded SHA-256, complete physical-device acceptance, "
            f"obtain `所有者验收.md: APPROVED`, then close `{phase}` through the release controller. Do not start the next phase yet."
        )
    elif phase_state.get("status") == "DONE":
        action = "All planned owner-facing releases are closed. Do not invent another version without an approved plan revision."
    elif current:
        action = (
            f"Continue exactly `{current}` using `work-packets/{current}.md` and the current phase specification. "
            "Do not reselect work from chat history."
        )
    else:
        action = "State is inconsistent because there is no current Work Packet. Run state validation and repair before editing."
    write_summary(phase_state, packet_state, action)


def start_packet(requested: str | None) -> None:
    phase_state, packet_state, runtime, definitions_by_id, runtime_by_id = validate()
    current = packet_state.get("work_packet_id")
    if not current:
        raise SystemExit("No current Work Packet can be started")
    if requested and requested != current:
        raise SystemExit(f"Cannot start {requested}; the state authority selected {current}")
    row = runtime_by_id[current]
    if row.get("status") not in {"TODO", "DOING", "BLOCKED"}:
        raise SystemExit(f"{current}: cannot start from status {row.get('status')}")
    timestamp = row.get("started_at") or now()
    sha = row.get("started_commit") or head_sha()
    row.update({"status": "DOING", "started_at": timestamp, "started_commit": sha, "reason": ""})
    packet_state.update({"status": "DOING", "started_at": timestamp, "started_commit": sha, "blocked_reason": None})
    phase_state.update({"status": "DOING", "started_at": phase_state.get("started_at") or timestamp, "blocked_reason": None})
    save_state(phase_state, packet_state, runtime)
    definition = definitions_by_id[current]
    write_summary(
        phase_state,
        packet_state,
        f"Implement `{current}` ({definition.get('title')}). Commit code, tests and Feature evidence before running close-packet.",
    )


def block_packet(reason: str) -> None:
    if not reason.strip():
        raise SystemExit("A concrete blocking reason is required")
    phase_state, packet_state, runtime, _definitions, runtime_by_id = validate()
    current = packet_state.get("work_packet_id")
    if not current:
        raise SystemExit("There is no current Work Packet to block")
    row = runtime_by_id[current]
    row.update({"status": "BLOCKED", "reason": reason.strip()})
    packet_state.update({"status": "BLOCKED", "blocked_reason": reason.strip()})
    phase_state.update({"status": "BLOCKED", "blocked_reason": reason.strip()})
    append_chain(
        PACKET_HISTORY,
        {
            "event": "WORK_PACKET_BLOCKED",
            "phase_id": packet_phase(row),
            "work_packet_id": current,
            "reason": reason.strip(),
            "commit": head_sha(),
            "recorded_at": now(),
        },
    )
    save_state(phase_state, packet_state, runtime)
    write_summary(
        phase_state,
        packet_state,
        f"`{current}` is blocked: {reason.strip()}. Continue unaffected work inside the same packet and create an Owner Action only for the external dependency.",
    )


def finalize_packet(requested: str | None, note: str, deferred_reason: str | None, no_push: bool) -> None:
    phase_state, packet_state, runtime, definitions_by_id, runtime_by_id = validate()
    current = packet_state.get("work_packet_id")
    if not current:
        raise SystemExit("No current Work Packet can be closed")
    if requested and requested != current:
        raise SystemExit(f"Cannot close {requested}; the state authority selected {current}")
    definition = definitions_by_id[current]
    counts = check_feature_closure(definition)
    implementation_commit = require_clean_git()
    timestamp = now()
    final_status = "DEFERRED_WITH_REASON" if deferred_reason else "CLOSED"
    event = "WORK_PACKET_DEFERRED" if deferred_reason else "WORK_PACKET_CLOSED"
    if deferred_reason is not None and not deferred_reason.strip():
        raise SystemExit("A non-empty deferred reason is required")
    record = append_chain(
        PACKET_HISTORY,
        {
            "event": event,
            "phase_id": packet_phase(definition),
            "work_packet_id": current,
            "result": final_status,
            "feature_ids": list(definition.get("feature_ids") or []),
            "feature_status_counts": counts,
            "commit": implementation_commit,
            "closed_at": timestamp,
            "reason": deferred_reason.strip() if deferred_reason else "",
            "note": note.strip(),
        },
    )
    row = runtime_by_id[current]
    row.update(
        {
            "status": final_status,
            "closed_at": timestamp,
            "closed_commit": implementation_commit,
            "result": final_status,
            "reason": deferred_reason.strip() if deferred_reason else "",
            "evidence": list(dict.fromkeys(list(row.get("evidence") or []) + [f"git:{implementation_commit}", f"history:{record['record_hash']}"])),
        }
    )
    phase = packet_phase(definition)
    phase_rows = packets_for_phase(list(runtime_by_id.values()), phase)
    unfinished = [item for item in phase_rows if item.get("status") not in PACKET_FINAL]
    if unfinished:
        next_row = unfinished[0]
        if next_row.get("status") == "PLANNED":
            next_row["status"] = "TODO"
        packet_state = {
            "schema_version": "1.0",
            "phase_id": phase,
            "work_packet_id": packet_id(next_row),
            "status": next_row.get("status"),
            "started_at": next_row.get("started_at"),
            "started_commit": next_row.get("started_commit"),
            "last_closed_work_packet": current,
        }
        phase_state.update({"status": "DOING", "blocked_reason": None})
        action = f"`{current}` is final at `{implementation_commit}`. Continue exactly `{packet_id(next_row)}`."
    else:
        packet_state = {
            "schema_version": "1.0",
            "phase_id": phase,
            "work_packet_id": None,
            "status": "PHASE_COMPLETE",
            "started_at": None,
            "started_commit": None,
            "last_closed_work_packet": current,
        }
        phase_state.update({"status": "READY_FOR_RELEASE", "blocked_reason": None})
        action = f"Every Work Packet in `{phase}` is final. Run online-server build and physical-device acceptance; do not start another phase."
    save_state(phase_state, packet_state, runtime)
    write_summary(phase_state, packet_state, action)
    state_commit = git_commit_state(f"chore(state): close {current}", push=not no_push)
    validate()
    print(json.dumps({"implementation_commit": implementation_commit, "state_commit": state_commit, "next": packet_state.get("work_packet_id")}, ensure_ascii=False, indent=2))


def close_release(requested_phase: str | None, no_tag: bool, no_push: bool, legacy_exception: bool) -> None:
    phase_state, packet_state, runtime, _definitions_by_id, runtime_by_id = validate()
    phase = str(phase_state.get("phase_id"))
    if requested_phase and requested_phase != phase:
        raise SystemExit(f"Cannot close {requested_phase}; current phase is {phase}")
    if phase_state.get("status") != "READY_FOR_RELEASE":
        raise SystemExit(f"{phase} is not READY_FOR_RELEASE")
    if any(row.get("status") not in PACKET_FINAL for row in packets_for_phase(list(runtime_by_id.values()), phase)):
        raise SystemExit(f"{phase}: all Work Packets must be CLOSED or DEFERRED_WITH_REASON")
    if legacy_exception and phase != "P01":
        raise SystemExit("--legacy-exception is a one-time owner-authorized closure path for P01 only")
    if owner_acceptance(phase, legacy_exception=legacy_exception) != "APPROVED":
        expected_file = "OWNER_ACCEPTANCE.md" if legacy_exception else "所有者验收.md"
        raise SystemExit(f"{phase}: {expected_file} must contain '- Result: APPROVED'")
    directory, build, provenance = release_evidence(phase, legacy_exception=legacy_exception)
    device = {}
    download = {}
    if not legacy_exception:
        device = json.loads((directory / "真机验收证据.json").read_text(encoding="utf-8-sig"))
        download = json.loads((directory / "本机下载校验证明.json").read_text(encoding="utf-8-sig"))
    if legacy_exception:
        release_commit = first_value(
            build.get("commit_sha"), build.get("git_commit"), build.get("source_commit"),
            provenance.get("commit_sha"), provenance.get("git_commit"), provenance.get("source_commit"),
        ) or head_sha()
        if not release_commit:
            raise SystemExit("P01 legacy closure requires an identifiable release commit")
    else:
        release_commit = require_clean_git()
    matrix = release_entry(phase)
    version_name = str(matrix.get("version_name"))
    version_code = matrix.get("version_code")
    evidence_commit = first_value(
        build.get("commit_sha"), build.get("git_commit"), build.get("source_commit"),
        provenance.get("commit_sha"), provenance.get("git_commit"), provenance.get("source_commit"),
    )
    if evidence_commit and str(evidence_commit) != release_commit:
        raise SystemExit(f"Release evidence commit {evidence_commit} differs from clean Git HEAD {release_commit}")
    if build.get("version_name") not in (None, "", version_name):
        raise SystemExit(f"构建信息 version_name {build.get('version_name')} differs from {version_name}")
    if build.get("version_code") not in (None, "", version_code):
        raise SystemExit(f"构建信息 version_code {build.get('version_code')} differs from {version_code}")
    apk_files = sorted(directory.glob("*.apk"))
    actual_apk_sha = sha256_file(apk_files[0]) if len(apk_files) == 1 else None
    declared_apk_sha = first_value(build.get("apk_sha256"), provenance.get("apk_sha256"), provenance.get("artifact_sha256"))
    if actual_apk_sha and declared_apk_sha and actual_apk_sha != declared_apk_sha:
        raise SystemExit("The delivered APK SHA-256 differs from release provenance")
    tag = f"{project_code()}-v{version_name}"
    if not no_tag:
        existing = git("rev-list", "-n", "1", tag, check=False).stdout.strip()
        if existing and existing != release_commit:
            raise SystemExit(f"Immutable tag {tag} already points to {existing}, not {release_commit}")
        if not existing:
            git("tag", "-a", tag, release_commit, "-m", f"Approved release {version_name}")

    append_chain(
        RELEASE_LEDGER,
        {
            "event": "RELEASE_CLOSED",
            "phase_id": phase,
            "version_name": version_name,
            "version_code": version_code,
            "status": "CLOSED",
            "commit": release_commit,
            "git_tag": tag,
            "server_build_run_id": provenance.get("build_run_id"),
            "source_archive_sha256": provenance.get("source_archive_sha256"),
            "server_manifest_verified": download.get("server_manifest_verified"),
            "physical_device_serial": (device.get("device") or {}).get("serial"),
            "physical_device_result": device.get("result"),
            "artifact_name": provenance.get("apk"),
            "artifact_sha256": provenance.get("apk_sha256"),
            "apk_sha256": actual_apk_sha or declared_apk_sha,
            "owner_result": "APPROVED",
            "owner_acceptance_file": str(
                (directory / ("OWNER_ACCEPTANCE.md" if legacy_exception else "所有者验收.md")).relative_to(ROOT)
            ),
            "closed_at": now(),
        },
    )
    if legacy_exception:
        rows = read_jsonl(RELEASE_LEDGER)
        rows[-1]["closure_mode"] = "OWNER_AUTHORIZED_LEGACY_EXCEPTION"
        rows[-1]["exception_scope"] = "P01 legacy English delivery; new Chinese delivery and rebuild gates deferred"
        rows[-1]["record_hash"] = canonical_hash(rows[-1])
        RELEASE_LEDGER.write_text(
            "\n".join(json.dumps(row, ensure_ascii=False, sort_keys=True) for row in rows) + "\n",
            encoding="utf-8",
        )

    order = phase_order()
    index = order.index(phase)
    phase_state.update(
        {
            "last_closed_phase": phase,
            "last_closed_version": version_name,
            "last_owner_approved_phase": phase,
            "blocked_reason": None,
        }
    )
    if index == len(order) - 1:
        phase_state.update({"status": "DONE", "current_phase": phase, "phase_id": phase})
        packet_state = {
            "schema_version": "1.0",
            "phase_id": phase,
            "work_packet_id": None,
            "status": "DONE",
            "started_at": None,
            "started_commit": None,
            "last_closed_work_packet": packet_state.get("last_closed_work_packet"),
        }
        action = f"Final planned release `{version_name}` is closed. The planned project is DONE."
    else:
        next_phase = order[index + 1]
        next_rows = packets_for_phase(list(runtime_by_id.values()), next_phase)
        if not next_rows:
            raise SystemExit(f"{next_phase}: no Work Packet runtime rows")
        next_row = next_rows[0]
        if next_row.get("status") == "PLANNED":
            next_row["status"] = "TODO"
        phase_state.update(
            {
                "current_phase": next_phase,
                "phase_id": next_phase,
                "status": "TODO",
                "started_at": None,
                "current_work_packet": packet_id(next_row),
            }
        )
        packet_state = {
            "schema_version": "1.0",
            "phase_id": next_phase,
            "work_packet_id": packet_id(next_row),
            "status": next_row.get("status"),
            "started_at": next_row.get("started_at"),
            "started_commit": next_row.get("started_commit"),
            "last_closed_work_packet": packet_state.get("last_closed_work_packet"),
        }
        action = f"Release `{version_name}` is closed. The exact next work is `{packet_id(next_row)}` in `{next_phase}`."
    save_state(phase_state, packet_state, runtime)
    write_summary(phase_state, packet_state, action)
    state_commit = git_commit_state(f"chore(release): close {phase} {version_name}", push=not no_push, tag=None if no_tag else tag)
    validate()
    print(json.dumps({"release_commit": release_commit, "state_commit": state_commit, "tag": None if no_tag else tag, "next_phase": phase_state.get("phase_id"), "next_work_packet": packet_state.get("work_packet_id")}, ensure_ascii=False, indent=2))


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    subparsers = parser.add_subparsers(dest="command", required=True)
    subparsers.add_parser("init")
    subparsers.add_parser("resume")
    subparsers.add_parser("show")
    subparsers.add_parser("validate")
    start = subparsers.add_parser("start-packet")
    start.add_argument("--packet")
    close = subparsers.add_parser("close-packet")
    close.add_argument("--packet")
    close.add_argument("--note", default="")
    close.add_argument("--no-push", action="store_true")
    defer = subparsers.add_parser("defer-packet")
    defer.add_argument("--packet")
    defer.add_argument("--reason", required=True)
    defer.add_argument("--note", default="")
    defer.add_argument("--no-push", action="store_true")
    block = subparsers.add_parser("block")
    block.add_argument("--reason", required=True)
    release = subparsers.add_parser("close-release")
    release.add_argument("--phase")
    release.add_argument("--no-tag", action="store_true")
    release.add_argument("--no-push", action="store_true")
    release.add_argument("--legacy-exception", action="store_true")
    arguments = parser.parse_args()

    if arguments.command == "init":
        init_state()
    elif arguments.command in {"resume", "show"}:
        resume()
    elif arguments.command == "validate":
        validate()
        print("PASS: state continuity and append-only ledgers are valid")
    elif arguments.command == "start-packet":
        start_packet(arguments.packet)
    elif arguments.command == "close-packet":
        finalize_packet(arguments.packet, arguments.note, None, arguments.no_push)
    elif arguments.command == "defer-packet":
        finalize_packet(arguments.packet, arguments.note, arguments.reason, arguments.no_push)
    elif arguments.command == "block":
        block_packet(arguments.reason)
    elif arguments.command == "close-release":
        close_release(arguments.phase, arguments.no_tag, arguments.no_push, arguments.legacy_exception)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
