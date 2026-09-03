#!/usr/bin/env python3
"""Pre-fix reproduction of the two round-4 command-agreement narrowings.

Each mutant is applied, verified PRESENT on disk carrying the MUTATED text,
compiled, and then measured against the WHOLE repository suite. The reviewer
measured SURVIVED for both; this reproduces that before a line of test code is
written, so the later KILLED is a delta and not a claim.
"""
import subprocess, sys, os, json

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "..", ".."))
GUARD = "\tif result.Command() != command {"

MUTANTS = [
    ("spare-the-covered-invocation-doctor", "internal/cliresult/client.go", GUARD,
     "\tif result.Command() != command && command != CommandDoctor {",
     "control: the one existing row drives (invoked doctor, document list), so this must be KILLED"),
    ("spare-a-different-invoked-command-list", "internal/cliresult/client.go", GUARD,
     "\tif result.Command() != command && command != CommandList {",
     "reviewer measured SURVIVED"),
    ("admit-any-document-claiming-takeover", "internal/cliresult/client.go", GUARD,
     "\tif result.Command() != command && result.Command() != CommandTakeover {",
     "reviewer measured SURVIVED against all 13 packages"),
]

def run(cmd):
    return subprocess.run(cmd, cwd=ROOT, capture_output=True, text=True)

results = []
for name, path, old, new, note in MUTANTS:
    full = os.path.join(ROOT, path)
    original = open(full).read()
    if original.count(old) != 1:
        results.append({"mutant": name, "status": "NOT-APPLIED",
                        "detail": f"marker occurs {original.count(old)} times"})
        continue
    mutated = original.replace(old, new, 1)
    open(full, "w").write(mutated)
    try:
        on_disk = open(full).read()
        if on_disk != mutated or new not in on_disk:
            results.append({"mutant": name, "status": "NOT-APPLIED",
                            "detail": "the mutated text is not the text on disk"})
            continue
        build = run(["go", "build", "./..."])
        if build.returncode != 0:
            results.append({"mutant": name, "status": "NOT-COMPILED",
                            "detail": build.stderr.strip()[:400]})
            continue
        measured = run(["go", "test", "./...", "-count=1"])
        results.append({
            "mutant": name, "gate": path, "note": note,
            "measured_against": "go test ./... -count=1",
            "status": "KILLED" if measured.returncode != 0 else "SURVIVED",
            "exit_code": measured.returncode,
        })
    finally:
        open(full, "w").write(original)

restored = run(["go", "test", "./internal/cliresult", "-count=1"])
report = {"commit": run(["git", "rev-parse", "HEAD"]).stdout.strip(),
          "results": results,
          "suite_restored_exit": restored.returncode}
print(json.dumps(report, indent=2))
