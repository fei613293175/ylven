#!/usr/bin/env python3
from __future__ import annotations
import argparse
import csv
import hashlib
import json
from collections import Counter
from pathlib import Path
from PIL import Image
import yaml

ROOT = Path(__file__).resolve().parents[1]
CANVAS = {
    'ANDROID': (1080, 2400), 'ADMIN': (1440, 1000), 'DEVELOPER': (1440, 1000),
    'WEB': (1878, 1000), 'DESIGN_SYSTEM': (1440, 1200),
}


def sha256(path: Path) -> str:
    h = hashlib.sha256()
    with path.open('rb') as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b''):
            h.update(chunk)
    return h.hexdigest()


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument('--require-approved', action='store_true')
    parser.add_argument('--json')
    args = parser.parse_args()
    errors: list[str] = []

    summary = json.loads((ROOT / 'UI_CONTRACT_BUILD_SUMMARY.json').read_text(encoding='utf-8'))
    with (ROOT / 'contracts' / 'mockup-manifest.csv').open(encoding='utf-8-sig', newline='') as handle:
        rows = list(csv.DictReader(handle))
    if len(rows) != summary['mockup_count']:
        errors.append(f"manifest count {len(rows)} != summary {summary['mockup_count']}")
    ids = [row['mockup_id'] for row in rows]
    if len(ids) != len(set(ids)):
        errors.append('duplicate mockup IDs')

    statuses: Counter[str] = Counter()
    surfaces: Counter[str] = Counter()
    for row in rows:
        statuses[row['status']] += 1
        surfaces[row['surface']] += 1
        path = ROOT / row['relative_path']
        if not path.is_file():
            errors.append(f"missing {row['relative_path']}")
            continue
        try:
            with Image.open(path) as image:
                if image.format != 'PNG':
                    errors.append(f"not PNG: {row['mockup_id']}")
                if image.size != CANVAS[row['surface']]:
                    errors.append(f"size {row['mockup_id']} {image.size} != {CANVAS[row['surface']]}")
        except Exception as exc:
            errors.append(f"cannot open {row['mockup_id']}: {exc}")
        if row.get('sha256') != sha256(path):
            errors.append(f"sha mismatch: {row['mockup_id']}")
        if row.get('duplicate_audit') != 'PASS':
            errors.append(f"duplicate audit not PASS: {row['mockup_id']}")
        if row['status'] not in {'GENERATED_PENDING_OWNER_REVIEW', 'APPROVED'}:
            errors.append(f"invalid status {row['mockup_id']}: {row['status']}")
        if args.require_approved and row['status'] != 'APPROVED':
            errors.append(f"not approved: {row['mockup_id']}")

    page_mockup_ids: list[str] = []
    for page_path in ROOT.glob('ui/pages/*/*.yaml'):
        page = yaml.safe_load(page_path.read_text(encoding='utf-8')) or {}
        page_mockup_ids.extend(state['mockup_id'] for state in page.get('states') or [])
    if set(page_mockup_ids) != set(ids):
        errors.append('page-contract mockup IDs differ from manifest')

    result = {
        'schema_version': '1.4', 'result': 'PASS' if not errors else 'FAIL',
        'mockup_count': len(rows), 'status_counts': dict(statuses),
        'surface_counts': dict(surfaces), 'errors': errors,
    }
    if args.json:
        Path(args.json).write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
    print(json.dumps(result, ensure_ascii=False, indent=2))
    return 0 if not errors else 1


if __name__ == '__main__':
    raise SystemExit(main())
