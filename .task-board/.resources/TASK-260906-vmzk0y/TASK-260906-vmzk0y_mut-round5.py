#!/usr/bin/env python3
"""Round-5 mutation harness for TASK-260906-vmzk0y.

Applies one mutant to a scratch copy of the candidate tree, runs up to
three bounded `go test` invocations as standalone processes (full package
suite for disposition, census-pair mask, targeted behavioural mask), and
restores the file byte-identically (SHA-256 asserted). Nothing is applied
to the candidate worktree itself.

Usage: MUT_ROOT=/tmp/vmzk0y-rev5-mut python3 mut_round5.py spec.json out.json
"""
import hashlib
import json
import os
import re
import subprocess
import sys

ROOT = os.environ.get("MUT_ROOT", "/tmp/vmzk0y-rev5-mut")


def sha(p):
    with open(p, "rb") as f:
        return hashlib.sha256(f.read()).hexdigest()


def run(pkg, mask, timeout=600):
    cmd = ["go", "test", pkg, "-count=1", "-v"]
    if mask:
        cmd += ["-run", mask]
    p = subprocess.run(cmd, cwd=ROOT, capture_output=True,
                       text=True, timeout=timeout)
    out = p.stdout + p.stderr
    fails = sorted(set(re.findall(r"^\s*--- FAIL: (\S+)", out, re.M)))
    passes = len(re.findall(r"^\s*--- PASS:", out, re.M))
    runs = len(re.findall(r"^=== RUN", out, re.M))
    return {"exit": p.returncode, "fails": fails,
            "pass_lines": passes, "run_lines": runs,
            "build_error": ("build failed" in out or "cannot use" in out
                            or "undefined:" in out or "[build failed]" in out
                            or "syntax error" in out)}


def main():
    spec = json.load(open(sys.argv[1]))
    results = []
    for m in spec:
        path, old, new = m["file"], m["old"], m["new"]
        full = os.path.join(ROOT, path)
        before = sha(full)
        src = open(full).read()
        n = src.count(old)
        row = {"id": m["id"], "class": m["class"], "file": path,
               "anchor_sites": n, "narrows_to": m.get("narrows_to", "")}
        if n != 1:
            row["disposition"] = "NOT_APPLIED"
            row["reason"] = "anchor occurs %d times, want 1" % n
            results.append(row)
            print(json.dumps(row), flush=True)
            continue
        open(full, "w").write(src.replace(old, new, 1))
        try:
            for r in m["runs"]:
                res = run(r["pkg"], r.get("mask"))
                row[r["name"]] = res
        finally:
            open(full, "w").write(src)
            after = sha(full)
            assert after == before, \
                "RESTORE MISMATCH %s: %s != %s" % (path, before, after)
        compile_fail = any(v.get("build_error") for k, v in row.items()
                           if isinstance(v, dict))
        kill_runs = [v for k, v in row.items() if isinstance(v, dict)]
        killed_by = sorted({f for v in kill_runs for f in v.get("fails", [])})
        row["killed_by"] = killed_by
        row["census_kill"] = row.get("census", {}).get("exit", 0) != 0
        row["behavioural_kill"] = row.get("behav", {}).get("exit", 0) != 0
        row["disposition"] = ("COMPILE_FAIL" if compile_fail
                              else ("KILLED" if any(
                                  v.get("exit", 0) != 0 for v in kill_runs)
                              else "SURVIVED"))
        results.append(row)
        print(json.dumps(row), flush=True)
    json.dump(results, open(sys.argv[2], "w"), indent=1)
    print("ALL RESTORED")


main()
