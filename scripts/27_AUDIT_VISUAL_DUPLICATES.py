#!/usr/bin/env python3
from __future__ import annotations
import importlib.util
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.dont_write_bytecode = True


def main() -> int:
    spec = importlib.util.spec_from_file_location('ylven_mockups', ROOT / 'scripts' / '24_GENERATE_MOCKUPS.py')
    if spec is None or spec.loader is None:
        raise RuntimeError('Could not load scripts/24_GENERATE_MOCKUPS.py')
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    result = module.audit_duplicates()
    print(json.dumps(result, ensure_ascii=False, indent=2))
    return 0 if result.get('result') == 'PASS' else 1


if __name__ == '__main__':
    raise SystemExit(main())
