#!/usr/bin/env python3
"""Compatibility wrapper. All runtime transitions are delegated to scripts/37_PROJECT_STATE.py."""
from __future__ import annotations
import argparse, subprocess, sys
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
def main():
 p=argparse.ArgumentParser(); p.add_argument('command',choices=['show','resume','validate','advance']); p.add_argument('--phase'); a=p.parse_args()
 mapping={'show':'show','resume':'resume','validate':'validate','advance':'close-release'}; cmd=[sys.executable,str(ROOT/'scripts/37_PROJECT_STATE.py'),mapping[a.command]]
 if a.command=='advance' and a.phase: cmd += ['--phase',a.phase]
 return subprocess.call(cmd,cwd=ROOT)
if __name__=='__main__': raise SystemExit(main())
