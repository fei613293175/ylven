#!/usr/bin/env python3
from pathlib import Path
import yaml
ROOT=Path(__file__).resolve().parents[1]
errors=[]
required=['contracts/state-transition-contract.yaml','contracts/execution-environment.yaml','contracts/repository-policy.yaml','CURRENT_PHASE.yaml','CURRENT_WORK_PACKET.yaml','status/WORK_PACKET_STATUS.yaml','status/WORK_PACKET_HISTORY.jsonl','status/RELEASE_LEDGER.jsonl','.github/workflows/android-phase-acceptance.yml']
for name in required:
 if not (ROOT/name).is_file(): errors.append(f'missing {name}')
for path in (ROOT/'work-packets').glob('P??-W??.md'):
 if 'YLVEN_V1_6_CONTINUITY_RULES' not in path.read_text(encoding='utf-8'): errors.append(f'{path.name}: missing V1.6 repeated continuity rules')
for path in (ROOT/'phases').glob('P??_*.md'):
 if 'YLVEN_V1_6_CONTINUITY_RULES' not in path.read_text(encoding='utf-8'): errors.append(f'{path.name}: missing V1.6 repeated continuity rules')
repo=yaml.safe_load((ROOT/'contracts/repository-policy.yaml').read_text(encoding='utf-8')) or {}
if repo.get('expected_visibility')!='public' or not repo.get('public_repository_is_allowed'): errors.append('owner-confirmed public repository is not encoded')
context=yaml.safe_load((ROOT/'PROJECT_CONTEXT.yaml').read_text(encoding='utf-8')) or {}
if str(context.get('package_version'))!='1.6.0': errors.append('PROJECT_CONTEXT package_version must be 1.6.0')
if (ROOT/'contracts/CURRENT_PHASE.yaml').exists() or (ROOT/'contracts/CURRENT_WORK_PACKET.yaml').exists(): errors.append('duplicate current-state mirrors are forbidden')
if errors:
 print('\n'.join(errors)); raise SystemExit(1)
print('PASS: V1.6 release, environment and continuity chain')
