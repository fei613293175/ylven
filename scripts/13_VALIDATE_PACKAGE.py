#!/usr/bin/env python3
"""Validate the YLVEN V1.6 repository/distribution and visual identity contracts."""
from __future__ import annotations
import argparse
import ast
import csv
import hashlib
import json
import re
import subprocess
import sys
from pathlib import Path
from urllib.parse import unquote
import yaml

ROOT = Path(__file__).resolve().parents[1]
PHASES = [f'P{i:02d}' for i in range(14)]
FORBIDDEN = {'.git', 'build', 'dist', '__pycache__', '.pytest_cache', 'node_modules', '.gradle', '.specify', '.agents', '.venv-tools', '.ylven-local', '.ylven-bootstrap-backup'}
REQUIRED_ROOT = [
    '00_READ_ME_FIRST.md', 'AGENTS.md', 'CURRENT_PHASE.yaml', 'CURRENT_WORK_PACKET.yaml', 'OWNER_ACTIONS.md',
    'PACKAGE_CONTENTS.md', 'PROJECT_CONTEXT.yaml', 'RELEASE_CONTRACT.yaml',
    'UI_CONTRACT_BUILD_SUMMARY.json', 'UPGRADE_NOTES_V1.6.md',
    'VISUAL_DUPLICATION_AUDIT_REPORT.json', 'VISUAL_DUPLICATION_AUDIT_REPORT.md',
    'YLVEN_MASTER_DEVELOPMENT_PLAN.md', 'requirements-tools.txt',
]


def sha256(path: Path) -> str:
    h = hashlib.sha256()
    with path.open('rb') as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b''):
            h.update(chunk)
    return h.hexdigest()


def run_python(arguments: list[str]) -> tuple[int, str]:
    result = subprocess.run([sys.executable, '-B', *arguments], cwd=ROOT, text=True, capture_output=True)
    return result.returncode, (result.stdout + result.stderr).strip()


def check_links(errors: list[str]) -> int:
    checked = 0
    pattern = re.compile(r'(?<!!)\[[^\]]*\]\(([^)]+)\)')
    for path in ROOT.rglob('*.md'):
        if any(part in FORBIDDEN for part in path.relative_to(ROOT).parts):
            continue
        for raw in pattern.findall(path.read_text(encoding='utf-8', errors='replace')):
            target = raw.strip().split(' ', 1)[0].strip('<>')
            if not target or target.startswith(('#', 'http://', 'https://', 'mailto:')):
                continue
            if any(token in target for token in ('{', '}', '<', '>')):
                continue
            target = unquote(target).split('#', 1)[0]
            resolved = (path.parent / target).resolve()
            checked += 1
            try:
                resolved.relative_to(ROOT.resolve())
            except ValueError:
                errors.append(f'{path.relative_to(ROOT)}: link escapes package: {target}')
                continue
            if not resolved.exists():
                errors.append(f'{path.relative_to(ROOT)}: broken local link: {target}')
    return checked


def main() -> int:
    parser = argparse.ArgumentParser()
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument('--repository-mode', action='store_true')
    group.add_argument('--distribution-mode', action='store_true')
    parser.add_argument('--report-json')
    parser.add_argument('--report-md')
    args = parser.parse_args()
    errors: list[str] = []
    warnings: list[str] = []
    checks: dict[str, object] = {}

    for relative in REQUIRED_ROOT:
        if not (ROOT / relative).is_file():
            errors.append(f'missing root file: {relative}')

    docs = {}
    for path in (ROOT / 'docs').glob('*.md'):
        match = re.match(r'^(\d{2})_', path.name)
        if match:
            number = int(match.group(1))
            if number in docs:
                errors.append(f'duplicate docs number {number:02d}')
            docs[number] = path
    if set(docs) != set(range(1, 37)):
        errors.append(f'docs must be exactly 01..36; found {sorted(docs)}')
    checks['detailed_document_count'] = len(docs)

    scripts = {}
    for path in (ROOT / 'scripts').iterdir():
        if not path.is_file():
            continue
        match = re.match(r'^(\d{2})_', path.name)
        if match:
            number = int(match.group(1))
            if number in scripts and number != 49:
                errors.append(f'duplicate script number {number:02d}')
            scripts[number] = path
    if set(scripts) != set(range(53)):
        errors.append(f'scripts must be exactly 00..52; found {sorted(scripts)}')
    checks['numbered_script_count'] = len(scripts)

    online_build_script = (ROOT / 'scripts' / '49_RUN_ONLINE_SERVER_BUILD.sh').read_text(encoding='utf-8')
    mount_destinations = re.findall(r'--mount "type=bind,src=[^,]+,dst=([^,"]+)', online_build_script)
    duplicate_mount_destinations = sorted({
        destination for destination in mount_destinations if mount_destinations.count(destination) > 1
    })
    if duplicate_mount_destinations:
        errors.append(f'online-server Docker mount destinations must be unique: {duplicate_mount_destinations}')
    if '/artifacts' in mount_destinations:
        errors.append('online-server build must not reuse coordinator-reserved Docker mount destination /artifacts')
    if 'container_artifact_dir="/ylven-artifacts"' not in online_build_script:
        errors.append('online-server build must isolate YLVEN release outputs under /ylven-artifacts')
    checks['online_server_mount_destinations'] = mount_destinations

    for phase in PHASES:
        if len(list((ROOT / 'phases').glob(f'{phase}_*.md'))) != 1:
            errors.append(f'{phase}: expected exactly one phase document')
        for suffix in ['FEATURE_STATUS.yaml', 'IMPLEMENTATION_STATUS.md']:
            if not (ROOT / 'status' / f'{phase}_{suffix}').is_file():
                errors.append(f'missing status/{phase}_{suffix}')
    if len(list((ROOT / 'work-packets').glob('P??-W??.md'))) != 64:
        errors.append('work packet document count must be 64')

    parse_counts = {'yaml': 0, 'json': 0, 'csv': 0, 'python': 0}
    for path in ROOT.rglob('*'):
        if not path.is_file() or any(part in FORBIDDEN for part in path.relative_to(ROOT).parts):
            continue
        if path.stat().st_size == 0:
            errors.append(f'empty file: {path.relative_to(ROOT)}')
            continue
        try:
            suffix = path.suffix.lower()
            if suffix in {'.yaml', '.yml'}:
                yaml.safe_load(path.read_text(encoding='utf-8'))
                parse_counts['yaml'] += 1
            elif suffix == '.json':
                json.loads(path.read_text(encoding='utf-8'))
                parse_counts['json'] += 1
            elif suffix == '.csv':
                with path.open(encoding='utf-8-sig', newline='') as handle:
                    list(csv.DictReader(handle))
                parse_counts['csv'] += 1
            elif suffix == '.py':
                ast.parse(path.read_text(encoding='utf-8'))
                parse_counts['python'] += 1
        except Exception as exc:
            errors.append(f'cannot parse {path.relative_to(ROOT)}: {exc}')
    checks['parsed_files'] = parse_counts

    feature_map = yaml.safe_load((ROOT / 'contracts' / 'feature-map.yaml').read_text(encoding='utf-8')) or {}
    contract_manifest = json.loads((ROOT / 'contracts' / 'CONTRACT_MANIFEST.json').read_text(encoding='utf-8'))
    ui_summary = json.loads((ROOT / 'UI_CONTRACT_BUILD_SUMMARY.json').read_text(encoding='utf-8'))
    expected = {
        'feature_count': 362, 'test_count': 1086, 'database_entity_count': 200,
        'openapi_operation_count': 226, 'ui_page_count': 332, 'ui_state_count': 2671,
        'ui_mockup_count': 2671, 'work_packet_count': 64,
    }
    actual = {'feature_count': len(feature_map.get('features') or [])}
    actual.update({key: contract_manifest.get(key) for key in expected if key != 'feature_count'})
    for key, expected_value in expected.items():
        if actual.get(key) != expected_value:
            errors.append(f'{key}: expected {expected_value}, got {actual.get(key)}')
    if (ui_summary.get('page_count'), ui_summary.get('state_count'), ui_summary.get('mockup_count')) != (332, 2671, 2671):
        errors.append(f'UI summary mismatch: {ui_summary}')
    checks.update(actual)

    packet_doc = yaml.safe_load((ROOT / 'contracts' / 'work-packet-map.yaml').read_text(encoding='utf-8')) or {}
    packets = packet_doc.get('work_packets') or []
    with (ROOT / 'contracts' / 'ui-state-catalog.csv').open(encoding='utf-8-sig', newline='') as handle:
        states = list(csv.DictReader(handle))
    state_by_packet: dict[str, set[str]] = {packet['work_packet_id']: set() for packet in packets}
    for state in states:
        for packet_id in filter(None, state.get('work_packets', '').split('|')):
            state_by_packet.setdefault(packet_id, set()).add(state['state_id'])
    for packet in packets:
        packet_id = packet['work_packet_id']
        declared = set(packet.get('state_ids') or [])
        if declared != state_by_packet.get(packet_id, set()):
            errors.append(f'{packet_id}: state_ids stale or incomplete')
        if packet.get('ui_state_count') != len(declared) or packet.get('mockup_count') != len(declared):
            errors.append(f'{packet_id}: state/mockup count mismatch')

    for command, label in [
        (['scripts/19_VALIDATE_UI_CONTRACTS.py', '--mode', 'planning'], 'UI contracts'),
        (['scripts/26_VALIDATE_GENERATED_MOCKUPS.py'], 'generated mockups'),
        (['scripts/27_AUDIT_VISUAL_DUPLICATES.py'], 'visual duplicate audit'),
        (['scripts/38_VALIDATE_STATE_CONTINUITY.py'], 'state continuity'),
        (['scripts/40_PUBLIC_REPOSITORY_SAFETY.py'], 'public repository safety'),
    ]:
        code, output = run_python(command)
        if code:
            errors.append(f'{label} failed: {output[-3000:]}')
        else:
            checks[label] = 'PASS'

    audit = json.loads((ROOT / 'VISUAL_DUPLICATION_AUDIT_REPORT.json').read_text(encoding='utf-8'))
    if audit.get('result') != 'PASS' or audit.get('invalid_duplicate_count') != 0:
        errors.append('visual duplication audit is not clean')
    checks['visual_duplicate_audit'] = {
        'result': audit.get('result'), 'exact_duplicate_groups': audit.get('exact_duplicate_groups'),
        'cross_page_exact_duplicates': len(audit.get('cross_page_exact_duplicates') or []),
        'same_page_exact_duplicates': len(audit.get('same_page_exact_duplicates') or []),
        'screen_signature_conflicts': len(audit.get('screen_signature_conflicts') or {}),
        'semantic_page_name_conflicts': len(audit.get('semantic_page_name_conflicts') or {}),
    }

    font_extensions = {'.ttf', '.otf', '.ttc', '.woff', '.woff2'}
    for path in ROOT.rglob('*'):
        if any(part in FORBIDDEN for part in path.relative_to(ROOT).parts):
            continue
        if path.is_file() and path.suffix.lower() in font_extensions:
            errors.append(f'font file must not be distributed: {path.relative_to(ROOT)}')
        if path.is_file() and path.suffix.lower() == '.apk':
            errors.append(f'APK must not be bundled in this planning package: {path.relative_to(ROOT)}')

    checks['markdown_local_links_checked'] = check_links(errors)

    if args.distribution_mode:
        for forbidden in FORBIDDEN:
            if any(item.name == forbidden for item in ROOT.rglob(forbidden)):
                errors.append(f'forbidden directory in distribution: {forbidden}')
        sums_path = ROOT / 'SHA256SUMS.txt'
        file_manifest = ROOT / 'FILE_MANIFEST.csv'
        package_manifest = ROOT / 'PACKAGE_MANIFEST.json'
        for path in [sums_path, file_manifest, package_manifest]:
            if not path.is_file():
                errors.append(f'missing distribution manifest: {path.name}')
        if sums_path.is_file():
            declared: dict[str, str] = {}
            for number, line in enumerate(sums_path.read_text(encoding='utf-8').splitlines(), 1):
                if not line.strip():
                    continue
                match = re.fullmatch(r'([0-9a-f]{64})  (.+)', line)
                if not match:
                    errors.append(f'invalid SHA256SUMS line {number}')
                    continue
                declared[match.group(2)] = match.group(1)
            expected_files = {
                path.relative_to(ROOT).as_posix() for path in ROOT.rglob('*')
                if path.is_file() and path.name != 'SHA256SUMS.txt'
                and not any(part in FORBIDDEN for part in path.relative_to(ROOT).parts)
            }
            if set(declared) != expected_files:
                errors.append(
                    f'SHA256SUMS file set mismatch: missing={len(expected_files-set(declared))}, '
                    f'extra={len(set(declared)-expected_files)}'
                )
            for relative, digest in declared.items():
                path = ROOT / relative
                if path.is_file() and sha256(path) != digest:
                    errors.append(f'SHA mismatch: {relative}')

    report = {
        'schema_version': '1.6.0',
        'mode': 'distribution' if args.distribution_mode else 'repository',
        'result': 'PASS' if not errors else 'FAIL', 'checks': checks,
        'warning_count': len(warnings), 'warnings': warnings,
        'error_count': len(errors), 'errors': errors,
    }
    if args.report_json:
        Path(args.report_json).write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
    if args.report_md:
        lines = [
            '# PACKAGE VALIDATION REPORT V1.6', '', f"- Mode: {report['mode']}",
            f"- Result: **{report['result']}**", f'- Errors: {len(errors)}',
            f'- Warnings: {len(warnings)}', '', '## Core counts', '',
        ]
        lines.extend(f'- {key}: {value}' for key, value in actual.items())
        if errors:
            lines.extend(['', '## Errors', ''])
            lines.extend(f'- {item}' for item in errors)
        Path(args.report_md).write_text('\n'.join(lines) + '\n', encoding='utf-8')
    print(json.dumps(report, ensure_ascii=False, indent=2))
    return 0 if not errors else 1


if __name__ == '__main__':
    raise SystemExit(main())
