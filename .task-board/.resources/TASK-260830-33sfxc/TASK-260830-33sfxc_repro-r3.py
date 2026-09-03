#!/usr/bin/env python3
"""Reproduce the two round-2 surviving narrowing mutants before any fix."""
import subprocess, os, json, sys

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))

MUTANTS = [
    ("F1-exit-equality-narrowed-to-registered-codes", "internal/cliresult/client.go",
     "\tif failure.ExitCode() != output.ExitStatus {",
     "\tif failure.CodeRegistered() && failure.ExitCode() != output.ExitStatus {"),
    ("F2-success-guard-narrowed-past-130", "internal/cliresult/client.go",
     "\tif output.ExitStatus != SuccessExitStatus {\n\t\treturn nil, fmt.Errorf(\n\t\t\t\"%w: stdout carries a CLI Result, which reports success, and the process exited %d\",",
     "\tif output.ExitStatus != SuccessExitStatus && output.ExitStatus != 130 {\n\t\treturn nil, fmt.Errorf(\n\t\t\t\"%w: stdout carries a CLI Result, which reports success, and the process exited %d\","),
]

def run(cmd):
    return subprocess.run(cmd, cwd=ROOT, capture_output=True, text=True)

out = []
for name, path, old, new in MUTANTS:
    full = os.path.join(ROOT, path)
    original = open(full).read()
    if original.count(old) != 1:
        out.append({"mutant": name, "status": "NOT-APPLIED", "detail": "marker not unique"})
        continue
    mutated = original.replace(old, new, 1)
    open(full, "w").write(mutated)
    try:
        on_disk = open(full).read()
        if on_disk != mutated or new not in on_disk:
            out.append({"mutant": name, "status": "NOT-APPLIED"})
            continue
        build = run(["go", "build", "./..."])
        if build.returncode != 0:
            out.append({"mutant": name, "status": "NOT-COMPILED", "detail": build.stderr[:400]})
            continue
        m = run(["go", "test", "./internal/cliresult", "./internal/axerror", "-count=1"])
        out.append({"mutant": name, "present_on_disk": True,
                    "suite": "./internal/cliresult ./internal/axerror",
                    "exit_code": m.returncode,
                    "status": "KILLED" if m.returncode != 0 else "SURVIVED"})
    finally:
        open(full, "w").write(original)

restored = run(["go", "test", "./internal/cliresult", "./internal/axerror", "-count=1"])
print(json.dumps({"results": out, "restored_suite_exit_code": restored.returncode}, indent=2))
