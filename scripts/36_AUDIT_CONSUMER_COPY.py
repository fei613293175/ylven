#!/usr/bin/env python3
from __future__ import annotations
import argparse,re
from pathlib import Path
FORBIDDEN=[r"\bstreaming\b",r"\bconnecting\b",r"provider error",r"request[_ ]?id",r"\bSSE\b",r"\bSub2API\b",r"\bdebug\b",r"\bdemo\b",r"\bmock\b",r"生成状态",r"正在连接并接收回答",r"请输入智能推荐",r"请输入 YLVEN 默认模型"]
SOURCE_EXT={".kt",".kts",".xml",".ts",".tsx",".js",".jsx",".vue",".html"}
EXCLUDED={"build","dist","node_modules",".gradle",".git","contracts","docs","status","change-orders","references","scripts","test","tests","androidtest","admin","developer","visual-review","mockups"}
UI_HINTS={"ui","screen","screens","component","components","presentation","compose","view","views","page","pages"}
STRING_RE=re.compile(r"\"(?:\\.|[^\"\\])*\"|'(?:\\.|[^'\\])*'|>[^<>]+<",re.S)

def is_consumer_ui_source(root:Path,p:Path)->bool:
    if p.suffix.lower() not in SOURCE_EXT: return False
    rel=p.relative_to(root); parts={x.lower() for x in rel.parts}
    if parts & EXCLUDED: return False
    if p.suffix.lower()==".xml" and "res" in parts and ("values" in parts or "layout" in parts or "menu" in parts): return True
    if p.suffix.lower() in {".kt",".kts"} and parts & UI_HINTS: return True
    if p.suffix.lower() in {".ts",".tsx",".js",".jsx",".vue",".html"} and parts & UI_HINTS: return True
    return False

def visible_fragments(text:str):
    for m in STRING_RE.finditer(text):
        v=m.group(0)
        if v.startswith(">") and v.endswith("<"): v=v[1:-1]
        elif len(v)>=2: v=v[1:-1]
        yield v

def main():
    ap=argparse.ArgumentParser(); ap.add_argument("--repo",required=True); a=ap.parse_args(); root=Path(a.repo).resolve()
    regs=[re.compile(x,re.I) for x in FORBIDDEN]; issues=[]; scanned=0
    for p in root.rglob("*"):
        if not p.is_file() or not is_consumer_ui_source(root,p): continue
        try: txt=p.read_text(encoding="utf-8")
        except Exception: continue
        scanned+=1
        for frag in visible_fragments(txt):
            if any(r.search(frag) for r in regs):
                pos=txt.find(frag); line=txt.count("\n",0,max(0,pos))+1
                issues.append((p,line,frag.strip()))
    if issues:
        for p,line,frag in issues[:200]: print(f"{p}:{line}: {frag}")
        raise SystemExit(f"Consumer copy policy failed: {len(issues)} visible string finding(s) across {scanned} UI source file(s)")
    print(f"Consumer copy policy PASS: scanned {scanned} consumer UI source file(s)")
if __name__=="__main__": main()
