#!/usr/bin/env python3
from pathlib import Path
import yaml
ROOT=Path(__file__).resolve().parents[1]
errors=[]
required=['contracts/state-transition-contract.yaml','contracts/execution-environment.yaml','contracts/repository-policy.yaml','contracts/android-phase-acceptance.yaml','CURRENT_PHASE.yaml','CURRENT_WORK_PACKET.yaml','status/WORK_PACKET_STATUS.yaml','status/WORK_PACKET_HISTORY.jsonl','status/RELEASE_LEDGER.jsonl','scripts/29_BUILD_ANDROID_ON_SERVER.sh','scripts/49_BUILD_ANDROID_ONLINE_SERVER.ps1','scripts/50_RUN_PHYSICAL_DEVICE_ACCEPTANCE.ps1','scripts/51_VERIFY_SERVER_ANDROID_BUILD.py']
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
if (ROOT/'.github/workflows/android-phase-acceptance.yml').exists(): errors.append('forbidden GitHub Actions Android workflow still exists')
acceptance=yaml.safe_load((ROOT/'contracts/android-phase-acceptance.yaml').read_text(encoding='utf-8')) or {}
forbidden=acceptance.get('forbidden_execution') or {}
if any(forbidden.get(k) is not True for k in ('github_actions','android_emulator','every_other_simulator')): errors.append('forbidden Android execution modes are not canonical')
if errors:
 print('\n'.join(errors)); raise SystemExit(1)
print('PASS: V1.6 release, environment and continuity chain')
