#!/usr/bin/env python3
from __future__ import annotations
import csv, re
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]

def main()->int:
 with (ROOT/'contracts/ui-interaction-map.csv').open(encoding='utf-8-sig',newline='') as f: rows=list(csv.DictReader(f))
 errors=[]; ids=[r.get('interaction_id','') for r in rows]
 if len(ids)!=len(set(ids)): errors.append('duplicate Interaction IDs')
 required=['interaction_id','feature_id','page_id','user_action','endpoint_or_event','functional_source']
 for i,r in enumerate(rows,2):
  for k in required:
   if not r.get(k): errors.append(f'row {i}: missing {k}')
 # Compose test tag convention is the Interaction ID itself; implementation tests must use it.
 out=ROOT/'contracts/interaction-test-bindings.csv'
 with out.open('w',encoding='utf-8-sig',newline='') as f:
  w=csv.DictWriter(f,fieldnames=['interaction_id','page_id','feature_id','compose_test_tag','required_test_id']); w.writeheader()
  for r in rows: w.writerow({'interaction_id':r['interaction_id'],'page_id':r['page_id'],'feature_id':r['feature_id'],'compose_test_tag':r['interaction_id'],'required_test_id':f"{r['interaction_id']}-UI"})
 if errors:
  print('\n'.join(errors[:100])); return 1
 print(f'PASS: {len(rows)} interaction contracts have unique IDs and required mappings'); return 0
if __name__=='__main__': raise SystemExit(main())
