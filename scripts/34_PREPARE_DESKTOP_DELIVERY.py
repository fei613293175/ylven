#!/usr/bin/env python3
from __future__ import annotations
import argparse,hashlib,json,shutil
from pathlib import Path

def sha(p:Path):
 h=hashlib.sha256();
 with p.open('rb') as f:
  for c in iter(lambda:f.read(1024*1024),b''):h.update(c)
 return h.hexdigest()

def main()->int:
 p=argparse.ArgumentParser(); p.add_argument('--artifact-dir',required=True); p.add_argument('--phase',required=True); p.add_argument('--version',required=True); a=p.parse_args()
 src=Path(a.artifact_dir); root=Path(__file__).resolve().parents[1]; dest=Path.home()/'Desktop'/'YLVEN-Releases'/a.version; required=['AUTOMATED_TEST_REPORT.md','VISUAL_DIFF_REPORT.md','CI_PROVENANCE.json']
 apks=list(src.rglob('*.apk')); errors=[]
 if len(apks)!=1: errors.append(f'exactly one tested APK required, got {len(apks)}')
 for n in required:
  if not (src/n).is_file(): errors.append(f'missing {n}')
 if errors: print('\n'.join(errors)); return 1
 if dest.exists(): shutil.rmtree(dest)
 shutil.copytree(src,dest)
 evidence=dest/'P00-evidence'; evidence.mkdir()
 for path in sorted((root/'docs'/'evidence').glob('P00-W*.md')): shutil.copy2(path,evidence/path.name)
 shutil.copy2(root/'status'/'P00_FEATURE_STATUS.yaml', evidence/'P00_FEATURE_STATUS.yaml')
 shutil.copy2(root/'specs'/'001-p00-engineering-foundation'/'tasks.md', evidence/'P00-tasks.md')
 lines=[]
 for f in sorted(x for x in dest.rglob('*') if x.is_file() and x.name!='SHA256SUMS.txt'):
  lines.append(f'{sha(f)}  {f.relative_to(dest).as_posix()}')
 (dest/'SHA256SUMS.txt').write_text('\n'.join(lines)+'\n',encoding='utf-8'); print(dest); return 0
if __name__=='__main__': raise SystemExit(main())
