#!/usr/bin/env python3
"""Reviewer rev4 mutation harness: apply one mutant to a scratch copy of the
candidate tree, run a bounded go test invocation as a standalone process, and
restore the file byte-identically (SHA-256 asserted)."""
import hashlib, json, os, re, subprocess, sys

ROOT = os.environ.get("MUT_ROOT", "/tmp/vmzk0y-rev4-mut")

def sha(p):
    with open(p, "rb") as f:
        return hashlib.sha256(f.read()).hexdigest()

def run(pkg, mask, timeout=900):
    cmd = ["go", "test", pkg, "-count=1", "-v"]
    if mask:
        cmd += ["-run", mask]
    p = subprocess.run(cmd, cwd=ROOT, capture_output=True, text=True, timeout=timeout)
    out = p.stdout + p.stderr
    fails = sorted(set(re.findall(r"^\s*--- FAIL: (\S+)", out, re.M)))
    passes = len(re.findall(r"^\s*--- PASS:", out, re.M))
    runs = len(re.findall(r"^=== RUN", out, re.M))
    return {"exit": p.returncode, "fails": fails, "pass_lines": passes, "run_lines": runs,
            "build_error": ("build failed" in out or "cannot use" in out or "undefined:" in out
                            or "[build failed]" in out or "syntax error" in out)}

def mutate(path, old, new, occurrence=1):
    full = os.path.join(ROOT, path)
    before = sha(full)
    src = open(full).read()
    n = src.count(old)
    if n != occurrence:
        return None, before, n
    open(full, "w").write(src.replace(old, new, 1))
    return full, before, n

def restore(full, before, path):
    orig = subprocess.run(["git", "show", f"6c8e29de0765747dc93c8565a756a888c6609ca6:{path}"],
                          cwd=os.environ.get("REPO", os.getcwd()), capture_output=True)
    open(full, "wb").write(orig.stdout)
    after = sha(full)
    assert after == before, f"RESTORE MISMATCH {path}: {before} != {after}"

def main():
    spec = json.load(open(sys.argv[1]))
    results = []
    for m in spec:
        path, old, new = m["file"], m["old"], m["new"]
        full, before, n = mutate(path, old, new)
        row = {"id": m["id"], "file": path, "anchor_sites": n}
        if full is None:
            row["disposition"] = "NOT_APPLIED"
            row["reason"] = f"anchor occurs {n} times, want 1"
            results.append(row)
            print(json.dumps(row)); continue
        try:
            for r in m["runs"]:
                res = run(r["pkg"], r.get("mask"))
                row[r["name"]] = res
        finally:
            restore(full, before, path)
        compile_fail = any(v.get("build_error") for k, v in row.items() if isinstance(v, dict))
        killed = any(v.get("exit", 0) != 0 for k, v in row.items() if isinstance(v, dict))
        row["disposition"] = "COMPILE_FAIL" if compile_fail else ("KILLED" if killed else "SURVIVED")
        results.append(row)
        print(json.dumps(row))
    json.dump(results, open(sys.argv[2], "w"), indent=1)
    print("ALL RESTORED")

main()
