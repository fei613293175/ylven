#!/usr/bin/env python3
from __future__ import annotations
import argparse,re
from pathlib import Path
SOURCE_EXT={".kt",".kts",".java",".go",".py",".ts",".tsx",".js",".jsx",".vue"}
EXCLUDED={"build","dist","node_modules",".gradle",".git","contracts","docs","status","change-orders","references","scripts","test","tests","androidtest","debug","preview","generated","mockups"}
PATTERNS=[
    ("unimplemented_call",re.compile(r"\bTODO\s*\(|NotImplementedError|UnsupportedOperationException\s*\(\s*[\"'](?:TODO|not implemented)",re.I)),
    ("panic_placeholder",re.compile(r"panic\s*\(\s*[\"'](?:TODO|not implemented|stub)",re.I)),
    ("frontend_placeholder",re.compile(r"throw\s+new\s+Error\s*\(\s*[\"'](?:TODO|not implemented|stub)",re.I)),
    ("explicit_fake_runtime",re.compile(r"PRODUCTION[_ -]?(?:MOCK|STUB|FAKE)|USE[_ -]?(?:MOCK|STUB|FAKE)[_ -]?IN[_ -]?PRODUCTION",re.I)),
]
BAD_NAME=re.compile(r"(?:^|[_-])(mock|stub|fake|demo)(?:[_-]|$)",re.I)

def is_prod_source(root:Path,p:Path)->bool:
    if p.suffix.lower() not in SOURCE_EXT: return False
    rel=p.relative_to(root); parts={x.lower() for x in rel.parts}
    if parts & EXCLUDED: return False
    return any(x in parts for x in {"src","main","app","backend","server","service","services","worker","runtime","gateway","frontend","android"})

def main():
    ap=argparse.ArgumentParser(); ap.add_argument("--repo",required=True); a=ap.parse_args(); root=Path(a.repo).resolve()
    issues=[]; scanned=0
    for p in root.rglob("*"):
        if not p.is_file() or not is_prod_source(root,p): continue
        scanned+=1
        rel=p.relative_to(root)
        if BAD_NAME.search(p.stem): issues.append((rel,1,"production source filename contains mock/stub/fake/demo"))
        try: txt=p.read_text(encoding="utf-8")
        except Exception: continue
        for label,rx in PATTERNS:
            for m in rx.finditer(txt):
                line=txt.count("\n",0,m.start())+1
                issues.append((rel,line,label+": "+m.group(0)[:120]))
    if issues:
        for rel,line,msg in issues[:200]: print(f"{rel}:{line}: {msg}")
        raise SystemExit(f"Production placeholder policy failed: {len(issues)} finding(s) across {scanned} production source file(s)")
    print(f"Production placeholder policy PASS: scanned {scanned} production source file(s)")
if __name__=="__main__": main()
