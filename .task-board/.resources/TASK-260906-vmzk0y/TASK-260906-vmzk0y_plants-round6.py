#!/usr/bin/env python3
"""Round-6 census control plants for TASK-260906-vmzk0y.

Each plant attacks one fail-closed arm of TestDigitCensusCoversEveryLeafGuard:
  P1 unregistered char site (plain new gate, new production file)
  P2 unregistered char site through a var binding (no identifier keying)
  P3 unregistered char site through an import alias (no import resolution)
  P4 orphan row (test-file edit, production untouched)
  P5 unclassifiable shape (equality against a digit rune)
  P6 unregistered bound site (new *10 accumulator with a guard)

Every plant is applied, the census is run as a standalone process (exit
code preserved, no pipe chain), the expected arm signature is asserted
in the output, and the tree is restored byte-identically (SHA-256
asserted). Results go to .temp/TASK-260906-vmzk0y/plants-round6.json.
"""
import hashlib
import json
import os
import subprocess
import sys

BACKUP = ".temp/TASK-260906-vmzk0y"
OUT = os.path.join(BACKUP, "plants-round6.json")

P1 = '''package terminalbackend

// Control plant P1 (temporary): an unregistered digit gate.
func plantUnregisteredGate(input string) bool {
\tfor i := 0; i < len(input); i++ {
\t\tif input[i] < '0' || input[i] > '9' {
\t\t\treturn false
\t\t}
\t}
\treturn true
}
'''

P2 = '''package terminalbackend

// Control plant P2 (temporary): digit gate through a var binding.
func plantVarBindingGate(input string) bool {
\tfor i := 0; i < len(input); i++ {
\t\toctet := input[i]
\t\tif octet < '0' || octet > '9' {
\t\t\treturn false
\t\t}
\t}
\treturn true
}
'''

P3 = '''package terminalbackend

import js "encoding/json"

// Control plant P3 (temporary): digit gate in a file using an import alias.
func plantAliasedGate(literal js.Number) bool {
\tdigits := literal.String()
\tfor i := 0; i < len(digits); i++ {
\t\tif digits[i] < '0' || digits[i] > '9' {
\t\t\treturn false
\t\t}
\t}
\treturn true
}
'''

P5 = '''package terminalbackend

// Control plant P5 (temporary): equality against a digit rune.
func plantUnclassifiableGate(input string) bool {
\tfor i := 0; i < len(input); i++ {
\t\tif input[i] == '5' {
\t\t\treturn false
\t\t}
\t}
\treturn true
}
'''

P6 = '''package terminalbackend

// Control plant P6 (temporary): unregistered accumulation bound.
func plantUnregisteredBound(digits string) int {
\tvalue := 0
\tfor i := 0; i < len(digits); i++ {
\t\tif digits[i] < '0' || digits[i] > '9' {
\t\t\treturn -1
\t\t}
\t\tif value > 100 {
\t\t\treturn -1
\t\t}
\t\tvalue = value*10 + int(digits[i]-'0')
\t}
\treturn value
}
'''

P4_OLD = ("\t{\n\t\tdir: \"provhost\", file: \"protocol.go\", function: \"parseMajor\",\n"
           "\t\tkind: \"char\", expr: `digit < '0' || digit > '9'`,")
P4_NEW = ("\t{\n\t\tdir: \"terminalbackend\", file: \"descriptor.go\", function: \"descriptorGeometry\",\n"
          "\t\tkind: \"char\", expr: `digit < '0' || digit > '0'`,\n"
          "\t\trejected: \"control plant P4 (temporary): no such gate\",\n"
          "\t\ttests: []string{\"TestParseProviderDescriptorValueRefusals\"},\n\t},\n" + P4_OLD)

PLANTS = [
    ("P1", "production-file", "internal/terminalbackend/zzplant_p1.go", P1,
     "unregistered char site"),
    ("P2", "production-file", "internal/terminalbackend/zzplant_p2.go", P2,
     "unregistered char site"),
    ("P3", "production-file", "internal/terminalbackend/zzplant_p3.go", P3,
     "unregistered char site"),
    ("P4", "test-edit", "internal/terminalbackend/digit_guard_census_test.go", (P4_OLD, P4_NEW),
     "orphan row"),
    ("P5", "production-file", "internal/terminalbackend/zzplant_p5.go", P5,
     "unclassifiable guard"),
    ("P6", "production-file", "internal/terminalbackend/zzplant_p6.go", P6,
     "unregistered bound site"),
]


def sha(path):
    with open(path, "rb") as handle:
        return hashlib.sha256(handle.read()).hexdigest()


def run(cmd):
    proc = subprocess.run(cmd, shell=True, capture_output=True, text=True)
    return proc.returncode, proc.stdout + proc.stderr


def tracked_files():
    _, out = run("git status --short")
    return out


def main():
    only = sys.argv[1:]
    pristine_tree = run("git stash list")
    results = []
    for pid, kind, path, payload, signature in PLANTS:
        if only and pid not in only:
            continue
        before = sha(path) if kind == "test-edit" else None
        existed = os.path.exists(path)
        try:
            if kind == "production-file":
                assert not existed, "%s already exists" % path
                with open(path, "w") as handle:
                    handle.write(payload)
            else:
                src = open(path).read()
                assert src.count(payload[0]) == 1, "P4 anchor sites=%d" % src.count(payload[0])
                open(path, "w").write(src.replace(payload[0], payload[1]))
            code, out = run("go test ./internal/terminalbackend/ -run 'TestDigitCensusCoversEveryLeafGuard' -count=1")
            failed = [l.split()[2] for l in out.splitlines() if l.strip().startswith("--- FAIL:")]
            ok = code != 0 and signature in out and failed == ["TestDigitCensusCoversEveryLeafGuard"]
            results.append({"id": pid, "arm": signature,
                            "disposition": "REDDENED" if ok else "NOT_REDDENED",
                            "exit": code, "failed": failed,
                            "signature_present": signature in out})
            print(json.dumps(results[-1]), flush=True)
        finally:
            if kind == "production-file":
                if os.path.exists(path):
                    os.remove(path)
            else:
                assert sha(path) != before or True
                # restore by reversing the replacement
                src = open(path).read()
                assert src.count(payload[1]) == 1, "P4 restore anchor missing"
                open(path, "w").write(src.replace(payload[1], payload[0]))
                assert sha(path) == before, "restore failed for %s" % path
    # confirm no plant file leaks and the worktree diff is plant-free
    leftovers = [p for _, _, p, _, _ in PLANTS if p.startswith("internal/") and os.path.exists(p)
                 and os.path.basename(p).startswith("zzplant")]
    status = tracked_files()
    results.append({"id": "CLEANUP", "leftovers": leftovers,
                    "zzplant_in_status": "zzplant" in status})
    print(json.dumps(results[-1]), flush=True)
    with open(OUT, "w") as handle:
        json.dump(results, handle, indent=2)
    assert not leftovers and "zzplant" not in status, "plant leak: %r" % (leftovers,)
    # census must be green again after all plants are reverted
    code, _ = run("go test ./internal/terminalbackend/ -run 'TestDigitCensus' -count=1")
    assert code == 0, "census not green after plant revert"
    print("ALL PLANTS REVERTED, CENSUS GREEN")


main()
