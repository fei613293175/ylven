#!/usr/bin/env python3
"""Safely rebuild V1.5 UI contracts while preserving valid generated/approved evidence."""
from __future__ import annotations
import argparse
import csv
import hashlib
import importlib.util
import json
import shutil
from datetime import datetime
from pathlib import Path
import yaml

ROOT = Path(__file__).resolve().parents[1]


def sha256(path: Path) -> str:
    h = hashlib.sha256()
    with path.open('rb') as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b''):
            h.update(chunk)
    return h.hexdigest()


def require_approved_visual_evidence(rows: dict[str, dict[str, str]]) -> None:
    """Reject reconciliation if the immutable approved visual baseline is absent."""
    failures: list[str] = []
    for mockup_id, row in rows.items():
        if row.get('status') != 'APPROVED':
            continue
        path = ROOT / row['relative_path']
        expected = row.get('sha256', '')
        if not expected:
            failures.append(f'{mockup_id}: approved entry has no SHA-256')
        elif not path.is_file():
            failures.append(f'{mockup_id}: approved PNG is missing ({path})')
        elif sha256(path) != expected:
            failures.append(f'{mockup_id}: approved PNG SHA-256 does not match the manifest')
    if failures:
        preview = '\n'.join(f'- {failure}' for failure in failures[:20])
        remainder = '' if len(failures) <= 20 else f'\n- ... and {len(failures) - 20} more'
        raise RuntimeError(
            'Refusing contract reconciliation because approved visual evidence is incomplete. '
            'Restore the approved assets before rerunning.\n' + preview + remainder
        )


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument('--confirm', action='store_true')
    args = parser.parse_args()
    if not args.confirm:
        raise SystemExit('Refusing UI contract regeneration without --confirm. A timestamped backup will be created.')

    manifest_path = ROOT / 'contracts' / 'mockup-manifest.csv'
    with manifest_path.open(encoding='utf-8-sig', newline='') as handle:
        old = {row['mockup_id']: row for row in csv.DictReader(handle)}
    require_approved_visual_evidence(old)

    stamp = datetime.now().strftime('%Y%m%d-%H%M%S')
    backup = ROOT / '.ylven-local' / 'ui-contract-backups' / stamp
    targets = [
        'contracts/ui-page-catalog.yaml', 'contracts/ui-state-catalog.csv',
        'contracts/mockup-manifest.csv', 'contracts/ui-page-resolved.csv',
        'contracts/feature-ui-map.yaml', 'contracts/ui-phase-binding.yaml',
        'contracts/work-packet-map.yaml', 'UI_CONTRACT_BUILD_SUMMARY.json', 'ui/pages',
    ]
    for relative in targets:
        source = ROOT / relative
        destination = backup / relative
        if source.is_dir():
            shutil.copytree(source, destination)
        elif source.is_file():
            destination.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(source, destination)

    spec = importlib.util.spec_from_file_location('ylven_mockups', ROOT / 'scripts' / '24_GENERATE_MOCKUPS.py')
    if spec is None or spec.loader is None:
        raise RuntimeError('Could not load the mockup generator')
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    # Contract reconciliation must not delete the approved PNGs before their
    # manifest hashes and approval metadata are checked below.  Full visual
    # regeneration remains available through the generator's explicit --all
    # flow, which deliberately resets the visual approval baseline.
    module.rebuild_contracts(preserve_visual_assets=True)

    with manifest_path.open(encoding='utf-8-sig', newline='') as handle:
        rows = list(csv.DictReader(handle))
    for row in rows:
        prior = old.get(row['mockup_id'])
        path = ROOT / row['relative_path']
        if prior and path.is_file() and prior.get('sha256') == sha256(path):
            for key in ['status', 'sha256', 'approved_by', 'approved_at', 'duplicate_audit']:
                row[key] = prior.get(key, '')
    with manifest_path.open('w', encoding='utf-8-sig', newline='') as handle:
        writer = csv.DictWriter(handle, fieldnames=list(rows[0]))
        writer.writeheader()
        writer.writerows(rows)

    by_id = {row['mockup_id']: row for row in rows}
    for page_path in ROOT.glob('ui/pages/*/*.yaml'):
        page = yaml.safe_load(page_path.read_text(encoding='utf-8')) or {}
        for state in page.get('states') or []:
            row = by_id.get(state['mockup_id'])
            if not row:
                continue
            state['mockup_status'] = row['status']
            state['mockup_sha256'] = row['sha256']
            if row.get('approved_by'):
                state['approved_by'] = row['approved_by']
                state['approved_at'] = row['approved_at']
        page_path.write_text(yaml.safe_dump(page, allow_unicode=True, sort_keys=False, width=150), encoding='utf-8')

    print(json.dumps({'result': 'PASS', 'backup': str(backup.relative_to(ROOT)), 'records': len(rows)}, ensure_ascii=False, indent=2))
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
