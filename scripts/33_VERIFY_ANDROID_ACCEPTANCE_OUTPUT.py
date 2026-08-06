#!/usr/bin/env python3
from __future__ import annotations
import argparse,hashlib,json
from pathlib import Path
from PIL import Image, ImageChops, ImageStat

P01_SCREENSHOTS = [
 '登录页.png', '注册页.png', '注册验证码页.png',
 '注册完成页.png', '账户与设备页.png',
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
 for name in ['自动化测试报告.md','视觉差异报告.md','覆盖安装证据.md']:
  if not (root/name).is_file(): errors.append(f'missing {name}')
 if a.phase.upper() == 'P01':
  images=[]
  for name in P01_SCREENSHOTS:
   path=root/'截图'/name
   if not path.is_file(): errors.append(f'missing P01 runtime screenshot: {name}'); continue
   shot=Image.open(path).convert('RGB'); images.append((name,shot))
   if shot.width < 720 or shot.height < 1280: errors.append(f'P01 runtime screenshot is too small: {name} {shot.size}')
   if all(high-low < 8 for low,high in ImageStat.Stat(shot).extrema): errors.append(f'P01 runtime screenshot is blank: {name}')
  for (left_name,left),(right_name,right) in zip(images,images[1:]):
   if left.size==right.size and sum(ImageStat.Stat(ImageChops.difference(left,right)).mean)/3 < 2:
    errors.append(f'P01 runtime screenshots are effectively identical: {left_name}, {right_name}')
 if errors: print('\n'.join(errors)); return 1
 info={'phase':a.phase,'version':a.version,'commit_sha':a.commit,'apk':apks[0].name,'apk_sha256':sha(apks[0])}
 (root/'CI来源证明.json').write_text(json.dumps(info,indent=2)+'\n',encoding='utf-8'); print('PASS'); return 0
if __name__=='__main__': raise SystemExit(main())
