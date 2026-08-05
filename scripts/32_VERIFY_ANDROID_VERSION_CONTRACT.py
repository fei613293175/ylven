#!/usr/bin/env python3
from __future__ import annotations
import argparse,re,yaml
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]

def main()->int:
 p=argparse.ArgumentParser(); p.add_argument('--phase',required=True); p.add_argument('--version',default='AUTO'); p.add_argument('--print-version',action='store_true'); a=p.parse_args()
 data=yaml.safe_load((ROOT/'contracts/release-version-matrix.yaml').read_text(encoding='utf-8')) or {}; phases=data.get('phases',[])
 phase='P00' if a.phase=='AUTO' else a.phase.upper(); row=next((x for x in phases if x['phase']==phase),None)
 if not row: print(f'unknown phase {phase}'); return 1
 version=row['version_name'] if a.version=='AUTO' else a.version
 if version!=row['version_name']: print(f'version mismatch: expected {row["version_name"]}, got {version}'); return 1
 if data.get('repository',{}).get('application_id')!='cc.orbexa.ylven': print('applicationId mismatch'); return 1
 print(version if a.print_version else f'PASS: {phase} -> {version} / {row["version_code"]}'); return 0
if __name__=='__main__': raise SystemExit(main())
