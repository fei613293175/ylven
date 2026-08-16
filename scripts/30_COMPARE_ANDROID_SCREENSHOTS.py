#!/usr/bin/env python3
from __future__ import annotations
import argparse, csv, json, math
from pathlib import Path
from PIL import Image, ImageChops, ImageFilter, ImageStat
ROOT=Path(__file__).resolve().parents[1]
P03_DEPRECATED_PAGES={"YL-A-019","YL-A-031"}

def applies_to_phase(row:dict[str,str],phase:str,packet:str|None=None)->bool:
 if row.get('surface')!='ANDROID': return False
 phases=row.get('phases','').split('|')
 if packet and packet not in row.get('work_packets','').split('|'): return False
 if phase!='P03': return phase in phases
 work_packets=row.get('work_packets','').split('|')
 return row.get('page_id') not in P03_DEPRECATED_PAGES and ('P03' in phases or 'P03-W07' in work_packets)

def score(a:Image.Image,b:Image.Image)->tuple[float,float]:
 a=a.convert('RGB'); b=b.convert('RGB')
 if a.size!=b.size: return 0.0,1.0
 diff=ImageChops.difference(a,b); stat=ImageStat.Stat(diff)
 mae=sum(stat.mean)/(3*255.0); return max(0.0,1.0-mae),mae

def normalize_runtime(reference:Image.Image,runtime:Image.Image)->tuple[Image.Image,str]:
 if runtime.size==reference.size: return runtime,'native-size'
 original=runtime.size
 normalized=runtime.resize(reference.size,Image.Resampling.LANCZOS)
 return normalized,f'native {original[0]}x{original[1]} -> comparison {reference.width}x{reference.height}'

def font_exception(phase:str)->dict|None:
 path=ROOT/'contracts'/'visual-exceptions'/f'{phase}-owner-font-rasterization.json'
 if not path.is_file(): return None
 data=json.loads(path.read_text(encoding='utf-8'))
 if data.get('phase')!=phase or data.get('status')!='OWNER_APPROVED': return None
 return data

def semantic_state_exception(phase:str)->dict|None:
 path=ROOT/'contracts'/'visual-exceptions'/f'{phase}-owner-semantic-state-review.json'
 if not path.is_file(): return None
 data=json.loads(path.read_text(encoding='utf-8'))
 if data.get('phase')!=phase or data.get('status')!='OWNER_APPROVED': return None
 if data.get('scope')!='P03 Android runtime state matrix only': return None
 return data


def packet_semantic_state_exception(phase:str,packet:str|None)->dict|None:
 if not packet: return None
 path=ROOT/'contracts'/'visual-exceptions'/f'{packet}-owner-semantic-state-review.json'
 if not path.is_file(): return None
 data=json.loads(path.read_text(encoding='utf-8'))
 if data.get('phase')!=phase or data.get('packet')!=packet or data.get('status')!='OWNER_APPROVED': return None
 if data.get('scope')!=f'{packet} Android runtime state matrix only': return None
 return data

def structural_score(a:Image.Image,b:Image.Image,radius:float)->tuple[float,float]:
 return score(a.convert('RGB').filter(ImageFilter.GaussianBlur(radius)),b.convert('RGB').filter(ImageFilter.GaussianBlur(radius)))

def main()->int:
 p=argparse.ArgumentParser(); p.add_argument('--phase',required=True); p.add_argument('--packet'); p.add_argument('--screenshots',required=True); p.add_argument('--report',required=True); a=p.parse_args()
 phase=a.phase.upper(); packet=a.packet.upper() if a.packet else None
 if packet and not packet.startswith(f'{phase}-W'): p.error(f'--packet {packet} does not belong to --phase {phase}')
 shots=Path(a.screenshots); report=Path(a.report); report.parent.mkdir(parents=True,exist_ok=True)
 with (ROOT/'contracts/ui-state-catalog.csv').open(encoding='utf-8-sig',newline='') as f: rows=[r for r in csv.DictReader(f) if applies_to_phase(r,phase,packet)]
 exception=font_exception(phase)
 semantic_exception=semantic_state_exception(phase)
 packet_exception=packet_semantic_state_exception(phase,packet)
 scope=packet or phase
 errors=[]; lines=[f'# Visual Diff Report — {scope}','']
 if exception:
  lines += [
   '- Owner exception: APPROVED — Android system-font rasterization differences only.',
   f"- Structural check: Gaussian blur {float(exception['structural_blur_radius_px']):.1f}px; mismatch <= {float(exception['structural_mismatch_max']):.3f}.",
   '- Layout, colors, component geometry, visible state and missing screenshots remain strict.',
   '',
  ]
 if semantic_exception:
  lines += [
   '- Owner exception: APPROVED - P03 runtime-state semantic visual review only.',
   f"- Semantic guardrail: similarity >= {float(semantic_exception['raw_similarity_min']):.3f}; structural mismatch <= {float(semantic_exception['structural_mismatch_max']):.3f}.",
   '- Raw metrics remain visible. This exception does not apply to production-route screenshots or any later phase.',
   '',
  ]
 if packet_exception:
  lines += [
   f'- Owner exception: APPROVED - {packet} runtime-state semantic visual review only.',
   '- Strict baseline result: FAIL (normal similarity >= 0.985 and mismatch <= 0.005 were not met); this exception does not rewrite the baseline result.',
   f"- Semantic guardrail: similarity >= {float(packet_exception['raw_similarity_min']):.3f}; structural mismatch <= {float(packet_exception['structural_mismatch_max']):.3f}.",
   '- Raw metrics remain visible. This exception applies only to this work packet and does not change later P04 validation.',
   '',
  ]
 lines += ['Raw physical-device PNGs are retained unchanged. Size normalization occurs only in memory for comparison.','', '| State | Screenshot Size | Comparison Similarity | Comparison Mismatch | Structural Mismatch | Result |','|---|---|---:|---:|---:|---|']
 if not rows:
  lines += ['', 'No Android surface states are bound to this phase.', 'Admin/Web visual evidence is reviewed from the phase evidence package and is not a physical-device Android screenshot.']
 for r in rows:
  sid=r['state_id']; actual=shots/f'{sid}.png'; ref=ROOT/r['mockup_path']
  if not actual.is_file():
   if phase == 'P00':
    lines.append(f'| {sid} | — | — | — | — | NOT_CAPTURED (P00 shell limitation) |'); continue
   errors.append(f'missing screenshot {sid}'); lines.append(f'| {sid} | — | — | — | — | MISSING |'); continue
  reference=Image.open(ref); runtime=Image.open(actual)
  runtime,normalization=normalize_runtime(reference,runtime)
  s,m=score(reference,runtime); structural=None
  ok=s>=0.985 and m<=0.005; result='PASS' if ok else 'REVIEW'
  if not ok and exception:
   structural_s,structural=structural_score(reference,runtime,float(exception['structural_blur_radius_px']))
   ok=(s>=float(exception['raw_similarity_min']) and m<=float(exception['raw_mismatch_max']) and structural<=float(exception['structural_mismatch_max']))
   if ok: result='PASS_WITH_OWNER_FONT_EXCEPTION'
  if not ok and semantic_exception:
   structural_s,structural=structural_score(reference,runtime,float(semantic_exception['structural_blur_radius_px']))
   ok=(s>=float(semantic_exception['raw_similarity_min']) and m<=float(semantic_exception['raw_mismatch_max']) and structural<=float(semantic_exception['structural_mismatch_max']))
   if ok: result='PASS_WITH_OWNER_SEMANTIC_STATE_EXCEPTION'
  if not ok and packet_exception:
   structural_s,structural=structural_score(reference,runtime,float(packet_exception['structural_blur_radius_px']))
   ok=(s>=float(packet_exception['raw_similarity_min']) and m<=float(packet_exception['raw_mismatch_max']) and structural<=float(packet_exception['structural_mismatch_max']))
   if ok: result='PASS_WITH_OWNER_PACKET_SEMANTIC_STATE_EXCEPTION'
  lines.append(f'| {sid} | {normalization} | {s:.5f} | {m:.5f} | {"—" if structural is None else f"{structural:.5f}"} | {result} |')
  if not ok: errors.append(f'{sid} visual threshold failed')
 result = 'PASS (runtime screenshots not captured for P00 shell)' if phase == 'P00' and not errors else ('PASS' if not errors else 'FAIL')
 lines += ['',f'Result: **{result}**']
 if errors: lines += ['','## Errors','']+[f'- {x}' for x in errors]
 report.write_text('\n'.join(lines)+'\n',encoding='utf-8'); return 0 if not errors else 1
if __name__=='__main__': raise SystemExit(main())
