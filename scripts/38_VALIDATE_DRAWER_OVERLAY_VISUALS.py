#!/usr/bin/env python3
from __future__ import annotations
from pathlib import Path
from PIL import Image, ImageStat
import csv, sys

ROOT=Path(__file__).resolve().parents[1]
PAGE_ID="YL-A-020"
PANEL_W=912
STATUS_H=72
W,H=1080,2400

def main():
    folder=ROOT/"ui/mockups/android"/PAGE_ID
    files=sorted(folder.glob(f"{PAGE_ID}-*.png"))
    if len(files)!=9:
        raise SystemExit(f"Expected 9 {PAGE_ID} state images, found {len(files)}")
    hashes=set()
    for p in files:
        im=Image.open(p).convert("RGB")
        if im.size!=(W,H): raise SystemExit(f"Wrong size: {p} {im.size}")
        crop=im.crop((PANEL_W,STATUS_H,W,H))
        stat=ImageStat.Stat(crop)
        variance=sum(stat.var)/3
        mean=sum(stat.mean)/3
        unique=len(crop.resize((84,116)).getcolors(maxcolors=1000000) or [])
        if variance < 120 or unique < 40:
            raise SystemExit(f"Opaque/uniform outside rail detected: {p.name}; variance={variance:.1f}, unique={unique}")
        if mean < 35:
            raise SystemExit(f"Outside scrim is too dark: {p.name}; mean={mean:.1f}")
        import hashlib
        h=hashlib.sha256(p.read_bytes()).hexdigest()
        if h in hashes: raise SystemExit(f"Duplicate drawer state image: {p.name}")
        hashes.add(h)
    print(f"PASS: {len(files)} modal drawer states; parent remains visible under non-uniform scrim")
if __name__=="__main__": main()
