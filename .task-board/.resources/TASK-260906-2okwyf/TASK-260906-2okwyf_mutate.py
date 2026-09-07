#!/usr/bin/env python3
"""v8heil mutation battery: apply one textual mutant, run a census mask and a
behavioural mask separately, restore the file byte-for-byte, report per-mask
verdicts plus the kill split.

Verdicts per mask:
  KILLED       mutant applied, mask FAILED
  SURVIVED     mutant applied, mask PASSED
Overall:
  KILLED       at least one mask failed (killer masks listed)
  SURVIVED     both masks passed  <-- finding, must be accounted
  NOT_APPLIED  anchor not found exactly once (control row)
  COMPILE_FAIL build/vet broke (control row)
"""
import hashlib
import json
import subprocess
import sys

ROOT = "/Users/iv/Developer/ReluxWorks/agent-session-manager/.temp/STORY-260905-3t31e9/worktree"


def sha(path):
    with open(path, "rb") as handle:
        return hashlib.sha256(handle.read()).hexdigest()


def run_mask(pkg, run):
    # Empty run selects the whole package (no -run flag): provider's
    # TestMain audit only runs unscoped, so a scoped census mask would
    # pass vacuously. Non-empty masks stay scoped.
    cmd = ["go", "test", pkg, "-count=1", "-v"]
    if run:
        cmd += ["-run", run]
    res = subprocess.run(cmd, cwd=ROOT, capture_output=True, text=True)
    out = res.stdout + res.stderr
    ran = out.count("--- PASS") + out.count("--- FAIL")
    verdict = "KILLED" if res.returncode != 0 else "SURVIVED"
    evidence = [line for line in out.splitlines()
                if "FAIL" in line or "want" in line
                or "refusal-site audit" in line][:6]
    return {"verdict": verdict, "ran": ran, "evidence": evidence}


def run(mut):
    path = ROOT + "/" + mut["file"]
    with open(path, "r", encoding="utf-8") as handle:
        original = handle.read()
    before = hashlib.sha256(original.encode()).hexdigest()
    old, new = mut["old"], mut["new"]
    count = original.count(old)
    if count != 1:
        return {"id": mut["id"], "class": mut["class"],
                "verdict": "NOT_APPLIED",
                "detail": "anchor occurs %d times" % count}
    try:
        with open(path, "w", encoding="utf-8") as handle:
            handle.write(original.replace(old, new, 1))
        pkg = "./" + mut["file"].split("/")[0] + "/" + mut["file"].split("/")[1] + "/"
        vet = subprocess.run(["go", "vet", pkg], cwd=ROOT,
                             capture_output=True, text=True)
        if vet.returncode != 0:
            return {"id": mut["id"], "class": mut["class"],
                    "verdict": "COMPILE_FAIL",
                    "detail": vet.stderr.strip().splitlines()[:3]}
        census = run_mask(mut.get("census_pkg", pkg), mut["census_run"])
        behav = run_mask(mut.get("behav_pkg", pkg), mut["behav_run"])
        masks = [("census", census), ("behav", behav)]
        record = {"id": mut["id"], "class": mut["class"],
                  "expect": mut.get("expect"),
                  "census": census, "behav": behav}
        # Audit mask (round 2): a full-package run whose TestMain audit
        # compares the exercised site set against the derivation. Scoped
        # census/behav masks skip that audit by construction, so a mutant
        # the derivation cannot see and the witnesses cannot feel (the
        # P8 sibling-swallow shape) is green on both and red here.
        if "audit_run" in mut or "audit_pkg" in mut:
            audit = run_mask(mut.get("audit_pkg", pkg), mut.get("audit_run", ""))
            masks.append(("audit", audit))
            record["audit"] = audit
        killers = [name for name, res in masks if res["verdict"] == "KILLED"]
        record["verdict"] = "KILLED" if killers else "SURVIVED"
        record["killers"] = killers
        return record
    finally:
        with open(path, "w", encoding="utf-8") as handle:
            handle.write(original)
        if sha(path) != before:
            print("!!! RESTORE FAILED for %s" % path, file=sys.stderr)
            sys.exit(2)


if __name__ == "__main__":
    muts = json.load(open(sys.argv[1]))
    out = []
    for item in muts:
        record = run(item)
        out.append(record)
        print(json.dumps(record), flush=True)
    summary = {v: sum(1 for r in out if r["verdict"] == v)
               for v in ("KILLED", "SURVIVED", "NOT_APPLIED", "COMPILE_FAIL")}
    print("SUMMARY " + json.dumps(summary), flush=True)
    with open(sys.argv[2], "w") as handle:
        json.dump({"results": out, "summary": summary}, handle, indent=2)
