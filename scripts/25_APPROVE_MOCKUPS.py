#!/usr/bin/env python3
from __future__ import annotations
import argparse
import csv
import hashlib
from datetime import datetime, timezone
from pathlib import Path
import yaml

ROOT = Path(__file__).resolve().parents[1]


def sha256(path: Path) -> str:
    h = hashlib.sha256()
    with path.open('rb') as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b''):
            h.update(chunk)
    return h.hexdigest()


def main() -> int:
    parser = argparse.ArgumentParser()
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument('--all', action='store_true')
    group.add_argument('--page', action='append')
    group.add_argument('--mockup', action='append')
    parser.add_argument('--approved-by', required=True)
    args = parser.parse_args()

    manifest = ROOT / 'contracts' / 'mockup-manifest.csv'
    with manifest.open(encoding='utf-8-sig', newline='') as handle:
        rows = list(csv.DictReader(handle))
    pages = set(args.page or [])
    mockups = set(args.mockup or [])
    selected = [row for row in rows if args.all or row['page_id'] in pages or row['mockup_id'] in mockups]
    if not selected:
        raise SystemExit('No matching mockups selected')

    approved_at = datetime.now(timezone.utc).replace(microsecond=0).isoformat().replace('+00:00', 'Z')
    for row in selected:
        path = ROOT / row['relative_path']
        if not path.is_file():
            raise SystemExit(f"Missing {row['relative_path']}")
        actual = sha256(path)
        if row.get('sha256') != actual:
            raise SystemExit(f"SHA mismatch for {row['mockup_id']}")
        if row.get('duplicate_audit') != 'PASS':
            raise SystemExit(f"Duplicate audit is not PASS for {row['mockup_id']}")
        row['status'] = 'APPROVED'
        row['approved_by'] = args.approved_by
        row['approved_at'] = approved_at

    with manifest.open('w', encoding='utf-8-sig', newline='') as handle:
        writer = csv.DictWriter(handle, fieldnames=list(rows[0]))
        writer.writeheader()
        writer.writerows(rows)

    by_id = {row['mockup_id']: row for row in rows}
    for page_path in ROOT.glob('ui/pages/*/*.yaml'):
        page = yaml.safe_load(page_path.read_text(encoding='utf-8')) or {}
        changed = False
        for state in page.get('states') or []:
            row = by_id.get(state['mockup_id'])
            if row and row['status'] == 'APPROVED':
                state['mockup_status'] = 'APPROVED'
                state['mockup_sha256'] = row['sha256']
                state['approved_by'] = row['approved_by']
                state['approved_at'] = row['approved_at']
                changed = True
        if changed:
            page_path.write_text(yaml.safe_dump(page, allow_unicode=True, sort_keys=False, width=150), encoding='utf-8')

    print(f'Approved {len(selected)} mockups by {args.approved_by}.')
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
