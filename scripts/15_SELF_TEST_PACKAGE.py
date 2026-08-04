#!/usr/bin/env python3
from __future__ import annotations
import argparse, datetime as dt, json, subprocess, sys
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
def run(args):
    cp=subprocess.run([sys.executable,'-B',*args],cwd=ROOT,text=True,capture_output=True)
    out=(cp.stdout+cp.stderr).strip()
    if cp.returncode: raise RuntimeError(f"{' '.join(args)} failed:\n{out}")
    return out
def main():
    ap=argparse.ArgumentParser(); ap.add_argument('--report-json',default=str(ROOT/'PACKAGE_SELF_TEST_REPORT.json')); ap.add_argument('--report-md',default=str(ROOT/'PACKAGE_SELF_TEST_REPORT.md')); a=ap.parse_args()
    started=dt.datetime.now(dt.timezone.utc).isoformat(); checks=[
      (['scripts/12_REGENERATE_CONTRACTS.py'],'regenerate_contracts'),
      (['scripts/07_VALIDATE_CONTRACTS.py','--phase','P00'],'validate_contracts'),
      (['scripts/38_VALIDATE_STATE_CONTINUITY.py'],'validate_state_continuity'),
      (['scripts/41_SELF_TEST_STATE_CONTINUITY.py'],'exercise_packet_and_release_closure'),
      (['scripts/36_VALIDATE_RELEASE_CHAIN.py'],'validate_repeated_version_rules'),
      (['scripts/40_PUBLIC_REPOSITORY_SAFETY.py'],'public_repository_secret_gate'),
      (['scripts/17_STATIC_CHECK_POWERSHELL.py'],'powershell_static_check'),
      (['scripts/18_VALIDATE_SPECKIT_BOOTSTRAP.py'],'speckit_bootstrap_contract'),
      (['scripts/19_VALIDATE_UI_CONTRACTS.py','--mode','planning'],'ui_contracts'),
      (['scripts/26_VALIDATE_GENERATED_MOCKUPS.py'],'mockup_integrity'),
      (['scripts/27_AUDIT_VISUAL_DUPLICATES.py'],'visual_duplicate_audit'),
      (['scripts/31_VALIDATE_INTERACTION_TEST_COVERAGE.py'],'interaction_test_coverage'),
      (['scripts/36_VALIDATE_RELEASE_CHAIN.py'],'release_chain_final')]
    rows=[]
    try:
      for cmd,name in checks: rows.append({'check':name,'result':'PASS','evidence':run(cmd)[-2000:]})
      result='PASS'; error=None
    except Exception as exc:
      result='FAIL'; error=str(exc); rows.append({'check':'failure','result':'FAIL','evidence':error})
    report={'schema_version':'1.6.0','result':result,'started_at_utc':started,'completed_at_utc':dt.datetime.now(dt.timezone.utc).isoformat(),'checks':rows,'error':error}
    Path(a.report_json).write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
    lines=['# YLVEN PACKAGE SELF-TEST REPORT V1.6','',f'- Result: **{result}**','', '## Checks','']
    lines.extend(f"- `{x['check']}`: **{x['result']}**" for x in rows)
    if error: lines += ['', '## Error','', '```text',error,'```']
    Path(a.report_md).write_text('\n'.join(lines)+'\n',encoding='utf-8')
    print(json.dumps(report,ensure_ascii=False,indent=2)); return 0 if result=='PASS' else 1
if __name__=='__main__': raise SystemExit(main())
