#!/usr/bin/env python3
from __future__ import annotations
import re, sys
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
SKIP={'.git','build','dist','node_modules','.gradle','.specify','.agents','.ylven-local','.venv-tools','ui','__pycache__','.pytest_cache','.mypy_cache'}
BAD_EXT={'.pem','.key','.p12','.pfx','.jks','.keystore','.env'}
ALLOW_NAMES={'.env.example'}
secret=re.compile(r'(?i)(api[_-]?key|access[_-]?token|refresh[_-]?token|client[_-]?secret|password|private[_-]?key)\s*[:=]\s*["\']?([^\s"\']{16,})')
private_key=re.compile(r'-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----')
errors=[]
for p in ROOT.rglob('*'):
 if not p.is_file() or any(part in SKIP for part in p.relative_to(ROOT).parts): continue
 if p.name in ALLOW_NAMES: continue
 if p.suffix.lower() in BAD_EXT or p.suffix.lower() in {'.pyc','.pyo'}: errors.append(f'public repository forbids sensitive file: {p.relative_to(ROOT)}'); continue
 if p.stat().st_size>3_000_000 or p.suffix.lower() in {'.png','.jpg','.jpeg','.webp','.zip'}: continue
 try: text=p.read_text(encoding='utf-8',errors='ignore')
 except Exception: continue
 if private_key.search(text): errors.append(f'private key content in {p.relative_to(ROOT)}')
 for m in secret.finditer(text):
  value=m.group(2)
  nearby=text[max(0,m.start()-80):min(len(text),m.end()+80)].lower()
  if any(x in nearby for x in ['example','placeholder','secret://','ref=','_ref','_file','your_','<','__']): continue
  errors.append(f'possible real secret in {p.relative_to(ROOT)}: {m.group(1)}')
  break
required=['.env','.ylven-local/','*.jks','*.keystore','*.p12','*.pfx','*.pem']
gitignore=(ROOT/'.gitignore').read_text(encoding='utf-8',errors='ignore')
for item in required:
 if item not in gitignore: errors.append(f'.gitignore missing public-safety pattern: {item}')
if errors:
 print('\n'.join(errors[:200])); sys.exit(1)
print('PASS: repository content is safe for the owner-confirmed public repository baseline')
