#!/usr/bin/env python3
from __future__ import annotations
import yaml
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
def main():
 d=yaml.safe_load((ROOT/'contracts/mockup-batches.yaml').read_text(encoding='utf-8')) or {}
 lines=['# YLVEN 效果图批次索引','']
 for b in d.get('batches') or []:
  lines += [f"## {b['batch_id']} — {b['surface']}（{b['count']} 张）",'']+[f"- `{x}`" for x in b['mockup_ids']]+['']
 (ROOT/'ui/MOCKUP_BATCH_INDEX_DETAILED.md').write_text('\n'.join(lines)+'\n',encoding='utf-8')
 print(len(d.get('batches') or []))
if __name__=='__main__': main()
