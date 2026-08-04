#!/usr/bin/env python3
from __future__ import annotations
import argparse, yaml
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]

def main()->int:
 p=argparse.ArgumentParser(); p.add_argument('--phase',required=True); p.add_argument('--output'); a=p.parse_args()
 phase=a.phase.upper(); data=yaml.safe_load((ROOT/'contracts/domain-delivery-map.yaml').read_text(encoding='utf-8')) or {}
 rows=[x for x in data.get('domains',[]) if x.get('required_by_phase')==phase]
 out=Path(a.output) if a.output else ROOT/'dist'/'releases'/phase/'DNS_ACTION_REQUIRED.md'; out.parent.mkdir(parents=True,exist_ok=True)
 lines=[f'# {phase} DNS ACTION REQUIRED','', '> 只有在 Codex 通过服务器盘点取得真实目标值后才能填写；禁止猜测 IP、CNAME、Tunnel ID 或 R2 目标。','']
 if not rows: lines += ['本阶段没有新增域名，但必须检查此前域名健康状态。']
 for x in rows:
  lines += [f"## {x['host']}",f"- 用途：{x['purpose']}",f"- 记录策略：{x['record_strategy']}",f"- Cloudflare：{x['cloudflare_proxy']}",f"- TLS：{x['tls']}",f"- 健康检查：`{x['health_check']}`",'- 真实目标值：`OWNER/CODEX_TO_FILL`','- 解析状态：`PENDING_OWNER_ACTION`','']
 out.write_text('\n'.join(lines)+'\n',encoding='utf-8'); print(out); return 0
if __name__=='__main__': raise SystemExit(main())
