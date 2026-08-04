#!/usr/bin/env python3
from __future__ import annotations
import importlib.util
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
spec=importlib.util.spec_from_file_location('project_state',ROOT/'scripts/37_PROJECT_STATE.py')
if spec is None or spec.loader is None: raise SystemExit('Cannot load state controller')
module=importlib.util.module_from_spec(spec); spec.loader.exec_module(module)
module.validate()
print('PASS: current phase, current Work Packet, runtime status, hash-chained history and release ledger are consistent')
