#!/usr/bin/env python3
from __future__ import annotations
import argparse,hashlib,json
from pathlib import Path
from PIL import Image, ImageChops, ImageStat

P01_SCREENSHOTS = [
 'P01-AUTH-LOGIN.png', 'P01-AUTH-REGISTER.png', 'P01-AUTH-REGISTER-OTP.png',
 'P01-AUTH-REGISTERED.png', 'P01-AUTH-ACCOUNT.png',
]

def sha(p:Path):
 h=hashlib.sha256();
 with p.open('rb') as f:
  for c in iter(lambda:f.read(1024*1024),b''):h.update(c)
 return h.hexdigest()

def main()->int:
 p=argparse.ArgumentParser(); p.add_argument('--phase',required=True); p.add_argument('--version',required=True); p.add_argument('--commit',required=True); a=p.parse_args()
 root=Path('build/owner-release'); apks=list(root.glob('*.apk')); errors=[]
 if len(apks)!=1: errors.append(f'exactly one APK required, got {len(apks)}')
 for name in ['AUTOMATED_TEST_REPORT.md','VISUAL_DIFF_REPORT.md']:
  if not (root/name).is_file(): errors.append(f'missing {name}')
 if a.phase.upper() == 'P01':
  images=[]
  for name in P01_SCREENSHOTS:
   path=root/'screenshots'/name
   if not path.is_file(): errors.append(f'missing P01 runtime screenshot: {name}'); continue
   shot=Image.open(path).convert('RGB'); images.append((name,shot))
   if shot.width < 720 or shot.height < 1280: errors.append(f'P01 runtime screenshot is too small: {name} {shot.size}')
   if all(high-low < 8 for low,high in ImageStat.Stat(shot).extrema): errors.append(f'P01 runtime screenshot is blank: {name}')
  for (left_name,left),(right_name,right) in zip(images,images[1:]):
   if left.size==right.size and sum(ImageStat.Stat(ImageChops.difference(left,right)).mean)/3 < 2:
    errors.append(f'P01 runtime screenshots are effectively identical: {left_name}, {right_name}')
 if errors: print('\n'.join(errors)); return 1
 info={'phase':a.phase,'version':a.version,'commit_sha':a.commit,'apk':apks[0].name,'apk_sha256':sha(apks[0])}
 (root/'CI_PROVENANCE.json').write_text(json.dumps(info,indent=2)+'\n',encoding='utf-8'); print('PASS'); return 0
if __name__=='__main__': raise SystemExit(main())
