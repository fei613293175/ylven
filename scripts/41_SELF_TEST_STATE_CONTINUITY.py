#!/usr/bin/env python3
from __future__ import annotations
import hashlib, json, shutil, subprocess, sys, tempfile, zipfile
from pathlib import Path
import yaml
ROOT=Path(__file__).resolve().parents[1]
CONTROLLER=ROOT/'scripts/37_PROJECT_STATE.py'
def run(root:Path,*args:str,ok=True):
 cp=subprocess.run([sys.executable,str(root/'scripts/37_PROJECT_STATE.py'),*args],cwd=root,text=True,capture_output=True)
 if ok and cp.returncode: raise SystemExit(f"{' '.join(args)} failed:\n{cp.stdout}{cp.stderr}")
 if not ok and cp.returncode==0: raise SystemExit(f"{' '.join(args)} unexpectedly passed")
 return cp
def wy(path:Path,data): path.parent.mkdir(parents=True,exist_ok=True); path.write_text(yaml.safe_dump(data,sort_keys=False),encoding='utf-8')
def commit(root:Path,msg:str):
 subprocess.run(['git','add','.'],cwd=root,check=True,capture_output=True)
 subprocess.run(['git','commit','-m',msg],cwd=root,check=True,capture_output=True)
def feature(path:Path,phase:str,ids:list[str]):
 wy(path/'status'/f'{phase}_FEATURE_STATUS.yaml',{'phase':phase,'features':[{'feature_id':x,'status':'IMPLEMENTED','reason':'','evidence':['test:PASS']} for x in ids]})
def chain_init(path:Path,event:str,**extra):
 rec={'event':event,'timestamp':'2026-08-05T00:00:00+08:00','previous_hash':None,**extra}
 body=json.dumps(rec,ensure_ascii=False,sort_keys=True,separators=(',',':')).encode(); rec['record_hash']=hashlib.sha256(body).hexdigest()
 path.write_text(json.dumps(rec,ensure_ascii=False,sort_keys=True)+'\n',encoding='utf-8')
def main():
 with tempfile.TemporaryDirectory(prefix='state-continuity-') as td:
  r=Path(td); (r/'scripts').mkdir(); shutil.copy2(CONTROLLER,r/'scripts/37_PROJECT_STATE.py'); (r/'contracts').mkdir(); (r/'status').mkdir(); (r/'dist/releases').mkdir(parents=True)
  definitions=[{'work_packet_id':'P00-W01','phase':'P00','sequence':1,'title':'first','feature_ids':['P00-001']},{'work_packet_id':'P00-W02','phase':'P00','sequence':2,'title':'second','feature_ids':['P00-002']},{'work_packet_id':'P01-W01','phase':'P01','sequence':1,'title':'third','feature_ids':['P01-001']}]
  wy(r/'contracts/work-packet-map.yaml',{'work_packets':definitions}); wy(r/'contracts/release-version-matrix.yaml',{'phases':[{'phase':'P00','version_name':'1.0.0','version_code':1000000},{'phase':'P01','version_name':'1.1.0','version_code':1010000}]}); wy(r/'PROJECT_CONTEXT.yaml',{'project':{'code':'sample'}})
  rows=[{'work_packet_id':d['work_packet_id'],'phase':d['phase'],'sequence':d['sequence'],'status':'TODO' if i==0 else 'PLANNED','started_at':None,'closed_at':None,'started_commit':None,'closed_commit':None,'result':None,'reason':'','evidence':[]} for i,d in enumerate(definitions)]
  wy(r/'status/WORK_PACKET_STATUS.yaml',{'work_packets':rows}); wy(r/'CURRENT_PHASE.yaml',{'current_phase':'P00','phase_id':'P00','current_work_packet':'P00-W01','status':'TODO','last_closed_phase':None,'last_closed_version':None}); wy(r/'CURRENT_WORK_PACKET.yaml',{'phase_id':'P00','work_packet_id':'P00-W01','status':'TODO','last_closed_work_packet':None})
  chain_init(r/'status/WORK_PACKET_HISTORY.jsonl','STATE_INITIALIZED',phase_id='P00',work_packet_id='P00-W01'); chain_init(r/'status/RELEASE_LEDGER.jsonl','LEDGER_INITIALIZED',next_phase='P00',next_work_packet='P00-W01')
  feature(r,'P00',['P00-001','P00-002']); feature(r,'P01',['P01-001']); (r/'.gitignore').write_text('dist/\n.ylven-local/\n',encoding='utf-8')
  subprocess.run(['git','init','-b','main'],cwd=r,check=True,capture_output=True); subprocess.run(['git','config','user.email','test@example.invalid'],cwd=r,check=True); subprocess.run(['git','config','user.name','State Test'],cwd=r,check=True); commit(r,'initial')
  run(r,'validate'); run(r,'start-packet'); commit(r,'implement first'); run(r,'close-packet','--no-push')
  current=yaml.safe_load((r/'CURRENT_WORK_PACKET.yaml').read_text()); assert current['work_packet_id']=='P00-W02'
  run(r,'start-packet'); commit(r,'implement second'); run(r,'close-packet','--no-push')
  phase=yaml.safe_load((r/'CURRENT_PHASE.yaml').read_text()); assert phase['status']=='READY_FOR_RELEASE'
  release=r/'dist/releases/P00'; release.mkdir(parents=True); apk=release/'sample.apk'
  with zipfile.ZipFile(apk,'w') as z: z.writestr('AndroidManifest.xml',b'x'); z.writestr('classes.dex',b'x')
  sha=subprocess.check_output(['git','rev-parse','HEAD'],cwd=r,text=True).strip(); apksha=hashlib.sha256(apk.read_bytes()).hexdigest()
  (release/'构建信息.json').write_text(json.dumps({'commit_sha':sha,'version_name':'1.0.0','version_code':1000000,'apk_sha256':apksha}),encoding='utf-8')
  (release/'服务器构建来源证明.json').write_text(json.dumps({'commit_sha':sha,'build_host_class':'connected_online_server','build_run_id':'self-test','apk':'sample.apk','apk_sha256':apksha}),encoding='utf-8')
  (release/'本机下载校验证明.json').write_text(json.dumps({'commit_sha':sha,'server_manifest_verified':True}),encoding='utf-8')
  (release/'真机验收证据.json').write_text(json.dumps({'commit_sha':sha,'result':'PASS','real_staging_business_flow':'PASS','production_page_visual_review':'PASS','device':{'serial':'self-test'}}),encoding='utf-8')
  (release/'真机日志审查.md').write_text('- 结果：PASS\n',encoding='utf-8')
  (release/'真实页面截图索引.csv').write_text('page_id,result\nSELF_TEST_ONLY,PASS\n',encoding='utf-8')
  (release/'真实页面视觉审查.json').write_text(json.dumps({'reviewer':'codex','result':'PASS','pages':[]}),encoding='utf-8')
  (release/'所有者验收.md').write_text('- Result: APPROVED\n',encoding='utf-8'); (release/'校验文件_SHA256.txt').write_text(f'{apksha}  sample.apk\n',encoding='utf-8')
  manifest='- Phase: P00\n- Version: 1.0.0\nP00-001 P00-002\n'
  for name in ('原功能清单.md','功能完成对比清单.md','完整测试清单.md'): (release/name).write_text(manifest,encoding='utf-8')
  for name in ('部署证据.md','域名DNS状态.md'): (release/name).write_text('- Result: PASS\n',encoding='utf-8')
  (release/'管理后台实测证据.md').write_text('测试人：Codex\nURL：https://example.invalid\n真实 API 数据：PASS\n审计记录：PASS\n',encoding='utf-8')
  (release/'覆盖安装证据.md').write_text('结果：PASS\n',encoding='utf-8')
  run(r,'close-release','--phase','P00','--no-push'); run(r,'validate')
  phase=yaml.safe_load((r/'CURRENT_PHASE.yaml').read_text()); packet=yaml.safe_load((r/'CURRENT_WORK_PACKET.yaml').read_text()); assert phase['phase_id']=='P01' and packet['work_packet_id']=='P01-W01'

  definitions.append({'work_packet_id':'P00-W03','phase':'P00','sequence':3,'title':'change order','feature_ids':['P00-003']})
  wy(r/'contracts/work-packet-map.yaml',{'work_packets':definitions})
  runtime=yaml.safe_load((r/'status/WORK_PACKET_STATUS.yaml').read_text())
  runtime['work_packets'].append({'work_packet_id':'P00-W03','phase':'P00','sequence':3,'status':'PLANNED','started_at':None,'closed_at':None,'started_commit':None,'closed_commit':None,'result':None,'reason':'','evidence':[]})
  wy(r/'status/WORK_PACKET_STATUS.yaml',runtime); feature(r,'P00',['P00-001','P00-002','P00-003']); commit(r,'install change order')
  run(r,'reopen-release','--phase','P00','--change-order','CO-TEST-001','--reason','approved incremental scope','--no-push'); run(r,'validate')
  phase=yaml.safe_load((r/'CURRENT_PHASE.yaml').read_text()); packet=yaml.safe_load((r/'CURRENT_WORK_PACKET.yaml').read_text()); assert phase['phase_id']=='P00' and packet['work_packet_id']=='P00-W03'
  run(r,'start-packet'); commit(r,'implement change order'); run(r,'close-packet','--no-push')
  sha=subprocess.check_output(['git','rev-parse','HEAD'],cwd=r,text=True).strip()
  (release/'构建信息.json').write_text(json.dumps({'commit_sha':sha,'version_name':'1.0.0','version_code':1000000,'apk_sha256':apksha}),encoding='utf-8')
  (release/'服务器构建来源证明.json').write_text(json.dumps({'commit_sha':sha,'build_host_class':'connected_online_server','build_run_id':'self-test-r2','apk':'sample.apk','apk_sha256':apksha}),encoding='utf-8')
  (release/'本机下载校验证明.json').write_text(json.dumps({'commit_sha':sha,'server_manifest_verified':True}),encoding='utf-8')
  (release/'真机验收证据.json').write_text(json.dumps({'commit_sha':sha,'result':'PASS','real_staging_business_flow':'PASS','production_page_visual_review':'PASS','device':{'serial':'self-test'}}),encoding='utf-8')
  manifest='- Phase: P00\n- Version: 1.0.0\nP00-001 P00-002 P00-003\n'
  for name in ('原功能清单.md','功能完成对比清单.md','完整测试清单.md'): (release/name).write_text(manifest,encoding='utf-8')
  run(r,'close-release','--phase','P00','--no-push'); run(r,'validate')
  phase=yaml.safe_load((r/'CURRENT_PHASE.yaml').read_text()); assert phase['phase_id']=='P01'
  ledger=[json.loads(x) for x in (r/'status/RELEASE_LEDGER.jsonl').read_text().splitlines() if x.strip()]
  assert sum(x.get('event')=='RELEASE_CLOSED' and x.get('phase_id')=='P00' for x in ledger)==2
  assert sum(x.get('event')=='RELEASE_REOPENED' and x.get('phase_id')=='P00' for x in ledger)==1
  assert ledger[-1]['git_tag']=='sample-v1.0.0-co-test-001' and ledger[-1]['owner_result']=='APPROVED'
  hist=[json.loads(x) for x in (r/'status/WORK_PACKET_HISTORY.jsonl').read_text().splitlines() if x.strip()]; assert sum(x.get('event')=='WORK_PACKET_CLOSED' for x in hist)==3
 print('PASS: packet closure, release reopen, replacement immutable tag and next-session recovery')
if __name__=='__main__': raise SystemExit(main())
