#!/usr/bin/env python3
from __future__ import annotations
import argparse, csv, math
from pathlib import Path
from PIL import Image, ImageChops, ImageStat
ROOT=Path(__file__).resolve().parents[1]

def score(a:Image.Image,b:Image.Image)->tuple[float,float]:
 a=a.convert('RGB'); b=b.convert('RGB')
 if a.size!=b.size: return 0.0,1.0
 diff=ImageChops.difference(a,b); stat=ImageStat.Stat(diff)
 mae=sum(stat.mean)/(3*255.0); return max(0.0,1.0-mae),mae

def main()->int:
 p=argparse.ArgumentParser(); p.add_argument('--phase',required=True); p.add_argument('--screenshots',required=True); p.add_argument('--report',required=True); a=p.parse_args()
 phase=a.phase.upper(); shots=Path(a.screenshots); report=Path(a.report); report.parent.mkdir(parents=True,exist_ok=True)
 with (ROOT/'contracts/ui-state-catalog.csv').open(encoding='utf-8-sig',newline='') as f: rows=[r for r in csv.DictReader(f) if phase in r.get('phases','').split('|') and r.get('surface') == 'ANDROID']
 errors=[]; lines=[f'# Visual Diff Report — {phase}','', '| State | Similarity | Mismatch | Result |','|---|---:|---:|---|']
 if not rows:
  lines += ['', 'No Android surface states are bound to this phase.', 'Admin/Web visual evidence is reviewed from the phase evidence package and is not an Android emulator screenshot.']
 for r in rows:
  sid=r['state_id']; actual=shots/f'{sid}.png'; ref=ROOT/r['mockup_path']
  if not actual.is_file():
   if phase == 'P00':
    lines.append(f'| {sid} | — | — | NOT_CAPTURED (P00 shell limitation) |'); continue
   errors.append(f'missing screenshot {sid}'); lines.append(f'| {sid} | — | — | MISSING |'); continue
  s,m=score(Image.open(ref),Image.open(actual)); ok=s>=0.985 and m<=0.005
  lines.append(f'| {sid} | {s:.5f} | {m:.5f} | {"PASS" if ok else "REVIEW"} |')
  if not ok: errors.append(f'{sid} visual threshold failed')
 result = 'PASS (runtime screenshots not captured for P00 shell)' if phase == 'P00' and not errors else ('PASS' if not errors else 'FAIL')
 lines += ['',f'Result: **{result}**']
 if errors: lines += ['','## Errors','']+[f'- {x}' for x in errors]
 report.write_text('\n'.join(lines)+'\n',encoding='utf-8'); return 0 if not errors else 1
if __name__=='__main__': raise SystemExit(main())
