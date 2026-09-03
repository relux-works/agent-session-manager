#!/usr/bin/env python3
"""Reviewer-independent mutation harness for CR rev5.

Each mutant: applied, verified present on disk carrying the MUTATED text,
compiled, then measured. Original restored afterwards.
"""
import json, subprocess, sys, os, hashlib

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "..", ".."))
os.chdir(ROOT)

MUTANTS = json.load(open(sys.argv[1]))
CMD = sys.argv[2].split() if len(sys.argv) > 2 else ["go", "test", "./...", "-count=1"]

def run(cmd, timeout=900):
    p = subprocess.run(cmd, capture_output=True, text=True, timeout=timeout)
    return p.returncode, (p.stdout + p.stderr)

results = []
for m in MUTANTS:
    path = m["file"]
    orig = open(path).read()
    if m["old"] not in orig:
        results.append({"id": m["id"], "status": "ANCHOR_MISSING"})
        continue
    if orig.count(m["old"]) != 1:
        results.append({"id": m["id"], "status": "ANCHOR_AMBIGUOUS", "count": orig.count(m["old"])})
        continue
    mutated = orig.replace(m["old"], m["new"])
    open(path, "w").write(mutated)
    try:
        on_disk = open(path).read()
        present = m["new"] in on_disk
        if not present:
            results.append({"id": m["id"], "status": "MUTATION_NOT_ON_DISK"})
            continue
        rc, out = run(["go", "build", "./..."])
        if rc != 0:
            results.append({"id": m["id"], "status": "DID_NOT_COMPILE", "output": out[-2000:]})
            continue
        rc, out = run(CMD)
        results.append({
            "id": m["id"],
            "file": path,
            "verified_present": True,
            "compiled": True,
            "status": "SURVIVED" if rc == 0 else "KILLED",
            "test_exit": rc,
            "failing": [l for l in out.splitlines() if l.startswith("--- FAIL") or l.startswith("FAIL")][:12],
        })
    finally:
        open(path, "w").write(orig)
        assert open(path).read() == orig

rc, out = run(CMD)
report = {"command": CMD, "mutants": results, "restored_suite_exit": rc}
print(json.dumps(report, indent=2))
