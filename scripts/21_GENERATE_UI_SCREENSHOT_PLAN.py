#!/usr/bin/env python3
from __future__ import annotations
import argparse,csv
from pathlib import Path
import yaml
ROOT=Path(__file__).resolve().parents[1]
def load_yaml(p): return yaml.safe_load(p.read_text(encoding='utf-8')) or {}
def main():
 p=argparse.ArgumentParser(); p.add_argument('--phase',required=True); p.add_argument('--output'); a=p.parse_args()
 pages=(load_yaml(ROOT/'contracts/ui-page-catalog.yaml').get('pages') or [])
 selected={x['page_id'] for x in pages if a.phase in (x.get('phases') or [])}
 with (ROOT/'contracts/ui-state-catalog.csv').open(encoding='utf-8-sig',newline='') as f: rows=[r for r in csv.DictReader(f) if r['page_id'] in selected]
 out=Path(a.output) if a.output else ROOT/f'dist/ui-screenshot-plan-{a.phase}.csv'; out.parent.mkdir(parents=True,exist_ok=True)
 fields=['state_id','page_id','page_name','state_name','mockup_path','runtime_screenshot_path','device_width_dp','theme','font_scale','status','notes']
 with out.open('w',encoding='utf-8-sig',newline='') as f:
  w=csv.DictWriter(f,fieldnames=fields); w.writeheader()
  for r in rows: w.writerow({'state_id':r['state_id'],'page_id':r['page_id'],'page_name':r['page_name'],'state_name':r['state_name'],'mockup_path':r['mockup_path'],'runtime_screenshot_path':f"screenshots/{r['state_id']}.png",'device_width_dp':'360','theme':'light','font_scale':'1.0','status':'PENDING','notes':''})
 print(out)
if __name__=='__main__': main()
