#!/usr/bin/env python3
from __future__ import annotations
import argparse, csv, json, math
from pathlib import Path
from PIL import Image, ImageChops, ImageFilter, ImageStat
ROOT=Path(__file__).resolve().parents[1]

def score(a:Image.Image,b:Image.Image)->tuple[float,float]:
 a=a.convert('RGB'); b=b.convert('RGB')
 if a.size!=b.size: return 0.0,1.0
 diff=ImageChops.difference(a,b); stat=ImageStat.Stat(diff)
 mae=sum(stat.mean)/(3*255.0); return max(0.0,1.0-mae),mae

def font_exception(phase:str)->dict|None:
 path=ROOT/'contracts'/'visual-exceptions'/f'{phase}-owner-font-rasterization.json'
 if not path.is_file(): return None
 data=json.loads(path.read_text(encoding='utf-8'))
 if data.get('phase')!=phase or data.get('status')!='OWNER_APPROVED': return None
 return data

def structural_score(a:Image.Image,b:Image.Image,radius:float)->tuple[float,float]:
 return score(a.convert('RGB').filter(ImageFilter.GaussianBlur(radius)),b.convert('RGB').filter(ImageFilter.GaussianBlur(radius)))

def main()->int:
 p=argparse.ArgumentParser(); p.add_argument('--phase',required=True); p.add_argument('--screenshots',required=True); p.add_argument('--report',required=True); a=p.parse_args()
 phase=a.phase.upper(); shots=Path(a.screenshots); report=Path(a.report); report.parent.mkdir(parents=True,exist_ok=True)
 with (ROOT/'contracts/ui-state-catalog.csv').open(encoding='utf-8-sig',newline='') as f: rows=[r for r in csv.DictReader(f) if phase in r.get('phases','').split('|') and r.get('surface') == 'ANDROID']
 exception=font_exception(phase)
 errors=[]; lines=[f'# Visual Diff Report — {phase}','']
 if exception:
  lines += [
   '- Owner exception: APPROVED — Android system-font rasterization differences only.',
   f"- Structural check: Gaussian blur {float(exception['structural_blur_radius_px']):.1f}px; mismatch <= {float(exception['structural_mismatch_max']):.3f}.",
   '- Layout, colors, component geometry, visible state and missing screenshots remain strict.',
   '',
  ]
 lines += ['| State | Raw Similarity | Raw Mismatch | Structural Mismatch | Result |','|---|---:|---:|---:|---|']
 if not rows:
  lines += ['', 'No Android surface states are bound to this phase.', 'Admin/Web visual evidence is reviewed from the phase evidence package and is not an Android emulator screenshot.']
 for r in rows:
  sid=r['state_id']; actual=shots/f'{sid}.png'; ref=ROOT/r['mockup_path']
  if not actual.is_file():
   if phase == 'P00':
    lines.append(f'| {sid} | — | — | — | NOT_CAPTURED (P00 shell limitation) |'); continue
   errors.append(f'missing screenshot {sid}'); lines.append(f'| {sid} | — | — | — | MISSING |'); continue
  reference=Image.open(ref); runtime=Image.open(actual)
  s,m=score(reference,runtime); structural=None
  ok=s>=0.985 and m<=0.005; result='PASS' if ok else 'REVIEW'
  if not ok and exception and reference.size==runtime.size:
   structural_s,structural=structural_score(reference,runtime,float(exception['structural_blur_radius_px']))
   ok=(s>=float(exception['raw_similarity_min']) and m<=float(exception['raw_mismatch_max']) and structural<=float(exception['structural_mismatch_max']))
   if ok: result='PASS_WITH_OWNER_FONT_EXCEPTION'
  lines.append(f'| {sid} | {s:.5f} | {m:.5f} | {"—" if structural is None else f"{structural:.5f}"} | {result} |')
  if not ok: errors.append(f'{sid} visual threshold failed')
 result = 'PASS (runtime screenshots not captured for P00 shell)' if phase == 'P00' and not errors else ('PASS' if not errors else 'FAIL')
 lines += ['',f'Result: **{result}**']
 if errors: lines += ['','## Errors','']+[f'- {x}' for x in errors]
 report.write_text('\n'.join(lines)+'\n',encoding='utf-8'); return 0 if not errors else 1
if __name__=='__main__': raise SystemExit(main())
