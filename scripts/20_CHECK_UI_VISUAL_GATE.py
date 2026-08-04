#!/usr/bin/env python3
from __future__ import annotations
import argparse,csv,hashlib,sys
from pathlib import Path
import yaml
ROOT=Path(__file__).resolve().parents[1]
def load_yaml(p): return yaml.safe_load(p.read_text(encoding='utf-8')) or {}
def digest(p):
 h=hashlib.sha256(); h.update(p.read_bytes()); return h.hexdigest()
def main():
 p=argparse.ArgumentParser(); g=p.add_mutually_exclusive_group(required=True); g.add_argument('--phase'); g.add_argument('--packet'); a=p.parse_args()
 pages=(load_yaml(ROOT/'contracts/ui-page-catalog.yaml').get('pages') or [])
 packets=(load_yaml(ROOT/'contracts/work-packet-map.yaml').get('work_packets') or [])
 if a.phase: selected={x['page_id'] for x in pages if a.phase in (x.get('phases') or [])}
 else:
  found=next((x for x in packets if x['work_packet_id']==a.packet),None)
  if not found: print(f'ERROR: unknown packet {a.packet}',file=sys.stderr); return 2
  selected=set(found.get('page_ids') or [])
 with (ROOT/'contracts/mockup-manifest.csv').open(encoding='utf-8-sig',newline='') as f: rows=list(csv.DictReader(f))
 errors=[]
 for r in rows:
  if r['page_id'] not in selected: continue
  if r['status']!='APPROVED': errors.append(f"{r['mockup_id']}: status={r['status']}"); continue
  path=ROOT/r['relative_path']
  if not path.is_file(): errors.append(f"{r['mockup_id']}: file missing"); continue
  if not r.get('sha256') or digest(path)!=r['sha256'].lower(): errors.append(f"{r['mockup_id']}: SHA-256 mismatch")
 if errors:
  print('VISUAL GATE BLOCKED. Non-visual work may continue; visual implementation is forbidden.',file=sys.stderr)
  for e in errors: print('ERROR:',e,file=sys.stderr)
  return 1
 print(f'Visual gate passed for {a.phase or a.packet}: {len(selected)} pages.')
 return 0
if __name__=='__main__': raise SystemExit(main())
