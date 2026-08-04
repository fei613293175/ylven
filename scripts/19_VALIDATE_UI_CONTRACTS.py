#!/usr/bin/env python3
from __future__ import annotations
import argparse
import csv
import json
import re
from collections import defaultdict
from pathlib import Path
import yaml

ROOT = Path(__file__).resolve().parents[1]
VISUAL_KINDS = {'PAGE', 'OVERLAY', 'COMPONENT_BOARD', 'SYSTEM_OVERLAY', 'SYSTEM_STATE_BOARD', 'DESIGN_BOARD'}
SURFACE_DIR = {'ANDROID': 'android', 'ADMIN': 'admin', 'DEVELOPER': 'developer', 'WEB': 'web', 'DESIGN_SYSTEM': 'design-system'}


def load(path: Path):
    return yaml.safe_load(path.read_text(encoding='utf-8')) or {}


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument('--mode', choices=['planning', 'implementation', 'release'], default='planning')
    parser.add_argument('--phase')
    parser.add_argument('--packet')
    args = parser.parse_args()
    errors: list[str] = []
    warnings: list[str] = []

    pages = load(ROOT / 'contracts' / 'ui-page-catalog.yaml').get('pages') or []
    features = load(ROOT / 'contracts' / 'feature-map.yaml').get('features') or []
    packets = load(ROOT / 'contracts' / 'work-packet-map.yaml').get('work_packets') or []
    known_features = {item['feature_id'] for item in features}
    known_packets = {item['work_packet_id'] for item in packets}
    page_ids = [item['page_id'] for item in pages]
    if len(page_ids) != len(set(page_ids)):
        errors.append('duplicate Page IDs')

    with (ROOT / 'contracts' / 'ui-state-catalog.csv').open(encoding='utf-8-sig', newline='') as handle:
        states = list(csv.DictReader(handle))
    with (ROOT / 'contracts' / 'mockup-manifest.csv').open(encoding='utf-8-sig', newline='') as handle:
        mockups = list(csv.DictReader(handle))
    state_ids = [item['state_id'] for item in states]
    mockup_ids = [item['mockup_id'] for item in mockups]
    if len(state_ids) != len(set(state_ids)):
        errors.append('duplicate State IDs')
    if len(mockup_ids) != len(set(mockup_ids)):
        errors.append('duplicate Mockup IDs')
    if set(state_ids) != set(mockup_ids):
        errors.append('State IDs and Mockup IDs differ')

    signatures: dict[str, list[str]] = defaultdict(list)
    normalized_names: dict[str, list[str]] = defaultdict(list)
    for page in pages:
        page_id = page['page_id']
        identity = page.get('visual_identity') or {}
        signatures[identity.get('screen_signature', '')].append(page_id)
        normalized = re.sub(r'[\s/·_-]+', '', page.get('name', '')).lower()
        normalized_names[normalized].append(page_id)
        if page.get('design_system_version') != 'YL-DS-1.2.0':
            errors.append(f'{page_id}: design-system version mismatch')
        if page.get('visual_kind') not in VISUAL_KINDS:
            errors.append(f'{page_id}: invalid visual_kind {page.get("visual_kind")}')
        if page.get('visual_kind') == 'OVERLAY' and not page.get('parent_page_id'):
            errors.append(f'{page_id}: OVERLAY requires parent_page_id')
        for feature_id in page.get('feature_ids') or []:
            if feature_id not in known_features:
                errors.append(f'{page_id}: unknown feature {feature_id}')
        for packet_id in page.get('work_packets') or []:
            if packet_id not in known_packets:
                errors.append(f'{page_id}: unknown packet {packet_id}')
        contract = ROOT / 'ui' / 'pages' / SURFACE_DIR[page['surface']] / f'{page_id}.yaml'
        if not contract.is_file():
            errors.append(f'missing page contract {page_id}')
        elif len((load(contract).get('states') or [])) != page.get('state_count'):
            errors.append(f'{page_id}: page contract state count differs from catalog')
    for signature, ids in signatures.items():
        if signature and len(set(ids)) > 1:
            errors.append(f'screen-signature conflict: {sorted(set(ids))}')
    for name, ids in normalized_names.items():
        if name and len(set(ids)) > 1:
            errors.append(f'normalized page-name conflict: {sorted(set(ids))}')

    selected = set(page_ids)
    if args.phase:
        selected = {page['page_id'] for page in pages if args.phase in (page.get('phases') or [])}
    if args.packet:
        packet = next((item for item in packets if item['work_packet_id'] == args.packet), None)
        if not packet:
            errors.append(f'unknown packet {args.packet}')
        else:
            selected = set(packet.get('page_ids') or [])

    if args.mode in {'implementation', 'release'}:
        for mockup in mockups:
            if mockup['page_id'] not in selected:
                continue
            if mockup['status'] != 'APPROVED':
                errors.append(f"{mockup['mockup_id']}: status {mockup['status']} not APPROVED")
            path = ROOT / mockup['relative_path']
            if not path.is_file() or not mockup.get('sha256'):
                errors.append(f"{mockup['mockup_id']}: missing file or SHA-256")

    result = {
        'schema_version': '1.4', 'mode': args.mode,
        'page_count': len(pages), 'state_count': len(states), 'mockup_count': len(mockups),
        'selected_page_count': len(selected), 'errors': errors, 'warnings': warnings,
        'status': 'PASS' if not errors else 'FAIL',
    }
    print(json.dumps(result, ensure_ascii=False, indent=2))
    return 0 if not errors else 1


if __name__ == '__main__':
    raise SystemExit(main())
