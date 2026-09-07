#!/usr/bin/env python3
"""Round-6 mutation battery for TASK-260906-vmzk0y.

Denominator: the 9 digit-guard sites of TestDigitCensusCoversEveryLeafGuard
(5 char + 4 bound across internal/terminalbackend + internal/provhost),
one enumeration, no listed extras. Every production mutant narrows,
token-preservingly narrows, or deletes exactly one enumerated site.

Masks (run separately per mutant, exit codes preserved, no pipe chains):
  census     = go test ./internal/terminalbackend/ -run 'TestDigitCensus'
  behavioral = go test <mutant package> -skip 'TestDigitCensus'
A mutant is KILLED when either mask exits non-zero; killed_by names the
failing tests per mask so the behavioural-versus-census split is derived,
not asserted. Classes narrowing / narrowing-token-preserving /
arm-deletion / census-only are separate; NOT_APPLIED and COMPILE_FAIL are
distinct dispositions outside the killed-over-applied denominator.

Each mutant is applied to the pristine file and restored byte-identically
afterwards (SHA-256 asserted). Usage: mut_round6.py [ID ...] runs only the
named mutants; no args runs all. Results append to battery-round6-<tag>.json.
"""
import hashlib
import json
import os
import shutil
import subprocess
import sys

BACKUP = ".temp/TASK-260906-vmzk0y"
DESC = "internal/terminalbackend/descriptor.go"
MANI = "internal/terminalbackend/manifest.go"
TBE = "internal/terminalbackend/terminalbackend.go"
PROTO = "internal/provhost/protocol.go"
SURR = "internal/provhost/surrogate.go"
CENSUS = "internal/terminalbackend/digit_guard_census_test.go"
TB = "./internal/terminalbackend/"
PH = "./internal/provhost/"

TARGETS = [DESC, MANI, TBE, PROTO, SURR, CENSUS]
PRISTINE = {p: os.path.join(BACKUP, "round6pristine_" + p.replace("/", "_")) for p in TARGETS}
for path, pristine in PRISTINE.items():
    shutil.copyfile(path, pristine)


def sha(path):
    with open(path, "rb") as handle:
        return hashlib.sha256(handle.read()).hexdigest()


ORIG_SHA = {p: sha(p) for p in TARGETS}

MUTANTS = [
    # S1 descriptorGeometry char gate.
    ("R6-D01", "narrowing", DESC, TB,
     "\t\tif digit < '0' || digit > '9' {",
     "\t\tif digit < '-' || digit > '9' {",
     "S1: lower bound to '-': admits exactly '-' and '.' (both reachable)"),
    ("R6-D02", "narrowing/token-preserving", DESC, TB,
     "\t\tdigit := digits[i]\n\t\tif digit < '0' || digit > '9' {",
     "\t\tdigit := digits[i]\n\t\tif digit == 'e' {\n\t\t\tvalue = value*10 + 5\n\t\t\tcontinue\n\t\t}\n\t\tif digit < '0' || digit > '9' {",
     "S1 TOKEN-PRESERVING: 'e' folds to digit 5, condition text unchanged"),
    ("R6-D03", "arm-deletion", DESC, TB,
     "\t\tif digit < '0' || digit > '9' {",
     "\t\tif false {",
     "S1: digit gate never fires"),
    # S6 descriptorGeometry pre-multiply guard.
    ("R6-D04", "narrowing", DESC, TB,
     "\t\tif value > 100 {",
     "\t\tif value > 922337203685477580 {",
     "S6 TRUE NARROWING (round-5 F1): threshold to floor(MaxInt/10)"),
    ("R6-D05", "narrowing", DESC, TB,
     "\t\tif value > 100 {",
     "\t\tif value > 9223372036854775807 {",
     "S6: guard never fires, accumulation wraps again"),
    ("R6-D06", "narrowing", DESC, TB,
     "\t\tif value > 100 {",
     "\t\tif value > 1000 {",
     "S6 EQUIVALENCE PROBE: post-gate still decides every exactly-accumulated literal"),
    ("R6-D07", "arm-deletion", DESC, TB,
     '\t\tif value > 100 {\n\t\t\treturn 0, &Error{Code: CodeProtocolError, Detail: "descriptor geometry bound"}\n\t\t}\n',
     "",
     "S6: pre-multiply guard deleted"),
    # S7 descriptorGeometry post-loop range.
    ("R6-D08", "narrowing", DESC, TB,
     "\tif value < 1 || value > 1000 {",
     "\tif value < 1 || value > 1001 {",
     "S7: upper range admits exactly 1001"),
    ("R6-D09", "narrowing", DESC, TB,
     "\tif value < 1 || value > 1000 {",
     "\tif value < 0 || value > 1000 {",
     "S7: lower range admits exactly 0"),
    ("R6-D10", "narrowing/token-preserving", DESC, TB,
     "\tif value < 1 || value > 1000 {",
     "\tif value == 1001 {\n\t\treturn uint16(value), nil\n\t}\n\tif value < 1 || value > 1000 {",
     "S7 TOKEN-PRESERVING: admits exactly 1001, range text unchanged"),
    ("R6-D11", "arm-deletion", DESC, TB,
     "\tif value < 1 || value > 1000 {",
     "\tif false {",
     "S7: post-loop range never fires"),
    # S8 semverMajor saturation guard.
    ("R6-D12", "narrowing", TBE, TB,
     "\t\tif major > (math.MaxInt-digit)/10 {",
     "\t\tif major > math.MaxInt/10 {",
     "S8 TRUE NARROWING: threshold drops the digit term"),
    ("R6-D13", "arm-deletion", TBE, TB,
     "\t\tif major > (math.MaxInt-digit)/10 {",
     "\t\tif false {",
     "S8: saturation guard never fires"),
    ("R6-D14", "narrowing/token-preserving", TBE, TB,
     "\t\tdigit := int(version[i] - '0')\n\t\tif major > (math.MaxInt-digit)/10 {",
     "\t\tdigit := int(version[i] - '0')\n\t\tif major == 922337203685477580 && version[i] == 56 {\n\t\t\tmajor = 1\n\t\t\tcontinue\n\t\t}\n\t\tif major > (math.MaxInt-digit)/10 {",
     "S8 TOKEN-PRESERVING: forces major 1 at the aliasing step, guard text unchanged"),
    # S2 manifest hex digit case.
    ("R6-D15", "narrowing", MANI, TB,
     "\t\tcase digit >= '0' && digit <= '9':",
     "\t\tcase digit >= '0' && digit <= ':':",
     "S2: digit case admits exactly ':' as nibble 10"),
    ("R6-D16", "narrowing/token-preserving", MANI, TB,
     "\t\tcase digit >= '0' && digit <= '9':",
     "\t\tcase digit == ':':\n\t\t\tunit |= 10\n\t\tcase digit >= '0' && digit <= '9':",
     "S2 TOKEN-PRESERVING: ':' decodes as nibble 10, digit case text unchanged"),
    ("R6-D17", "arm-deletion", MANI, TB,
     "\t\tcase digit >= '0' && digit <= '9':\n\t\t\tunit |= uint16(digit - '0')\n",
     "",
     "S2: digit case deleted, digit escapes refused"),
    ("R6-D32", "narrowing", MANI, TB,
     "\t\tcase digit >= '0' && digit <= '9':",
     "\t\tcase digit >= '/' && digit <= '9':",
     "S2 LOWER-EDGE BOUND PROBE: '/' converts to 0xFFFF, verdict-equivalent on every input"),
    # S5 surrogate hex digit branch.
    ("R6-D18", "narrowing", SURR, PH,
     "\t\tif b >= '0' && b <= '9' {",
     "\t\tif b >= '0' && b <= ':' {",
     "S5: digit branch admits exactly ':' as nibble 10"),
    ("R6-D19", "narrowing/token-preserving", SURR, PH,
     "\t\tif b >= '0' && b <= '9' {",
     "\t\tif b == ':' {\n\t\t\tdigit = 10\n\t\t} else if b >= '0' && b <= '9' {",
     "S5 TOKEN-PRESERVING: ':' decodes as nibble 10, branch text unchanged"),
    ("R6-D20", "arm-deletion", SURR, PH,
     "\t\tif b >= '0' && b <= '9' {",
     "\t\tif false {",
     "S5: digit branch never fires"),
    ("R6-D33", "narrowing", SURR, PH,
     "\t\tif b >= '0' && b <= '9' {",
     "\t\tif b >= '/' && b <= '9' {",
     "S5 LOWER-EDGE BOUND PROBE: '/' converts to a garbage nibble, verdict-equivalent on every input"),
    # S3 parseMajor major gate.
    ("R6-D21", "narrowing", PROTO, PH,
     "\t\tif digit < '0' || digit > '9' {",
     "\t\tif digit < '0' || digit > ':' {",
     "S3: major admits exactly ':'"),
    ("R6-D22", "narrowing", PROTO, PH,
     "\t\tif digit < '0' || digit > '9' {",
     "\t\tif digit < '/' || digit > '9' {",
     "S3: major admits exactly '/'"),
    ("R6-D23", "narrowing/token-preserving", PROTO, PH,
     "\t\tdigit := parts[0][i]\n\t\tif digit < '0' || digit > '9' {",
     "\t\tdigit := parts[0][i]\n\t\tif digit == ':' {\n\t\t\tcontinue\n\t\t}\n\t\tif digit < '0' || digit > '9' {",
     "S3 TOKEN-PRESERVING: major admits exactly ':', condition text unchanged"),
    ("R6-D27", "arm-deletion", PROTO, PH,
     "\t\tif digit < '0' || digit > '9' {",
     "\t\tif false {",
     "S3: major gate never fires"),
    # S4 parseMajor rest gate.
    ("R6-D24", "narrowing", PROTO, PH,
     "\t\t\tif rest[i] < '0' || rest[i] > '9' {",
     "\t\t\tif rest[i] < '0' || rest[i] > ':' {",
     "S4: rest admits exactly ':'"),
    ("R6-D25", "narrowing", PROTO, PH,
     "\t\t\tif rest[i] < '0' || rest[i] > '9' {",
     "\t\t\tif rest[i] < '/' || rest[i] > '9' {",
     "S4: rest admits exactly '/'"),
    ("R6-D26", "narrowing/token-preserving", PROTO, PH,
     "\t\tfor i := 0; i < len(rest); i++ {\n\t\t\tif rest[i] < '0' || rest[i] > '9' {",
     "\t\tfor i := 0; i < len(rest); i++ {\n\t\t\tif rest[i] == '/' {\n\t\t\t\tcontinue\n\t\t\t}\n\t\t\tif rest[i] < '0' || rest[i] > '9' {",
     "S4 TOKEN-PRESERVING: rest admits exactly '/', condition text unchanged"),
    ("R6-D28", "arm-deletion", PROTO, PH,
     "\t\t\tif rest[i] < '0' || rest[i] > '9' {",
     "\t\t\tif false {",
     "S4: rest gate never fires"),
    # S9 parseMajor saturation guard.
    ("R6-D29", "narrowing", PROTO, PH,
     "\t\tif major > (math.MaxInt-step)/10 {",
     "\t\tif major > math.MaxInt/10 {",
     "S9 TRUE NARROWING: threshold drops the step term"),
    ("R6-D30", "arm-deletion", PROTO, PH,
     "\t\tif major > (math.MaxInt-step)/10 {",
     "\t\tif false {",
     "S9: saturation guard never fires"),
    ("R6-D31", "narrowing/token-preserving", PROTO, PH,
     "\t\tstep := int(digit - '0')\n\t\tif major > (math.MaxInt-step)/10 {",
     "\t\tstep := int(digit - '0')\n\t\tif major == 922337203685477580 && parts[0][i] == 56 {\n\t\t\tmajor = 2\n\t\t\tcontinue\n\t\t}\n\t\tif major > (math.MaxInt-step)/10 {",
     "S9 TOKEN-PRESERVING: forces major 2 at the aliasing step, guard text unchanged"),
    # Census-only: production untouched.
    ("R6-C01", "census-only", CENSUS, TB,
     '\t\tdir: "provhost", file: "protocol.go", function: "parseMajor",\n\t\tkind: "char", expr: `digit < \'0\' || digit > \'9\'`,',
     '\t\tdir: "provhost", file: "protocol.go", function: "parseMajorXX",\n\t\tkind: "char", expr: `digit < \'0\' || digit > \'9\'`,',
     "census row points at no function (production untouched)"),
]

CONTROLS = [
    ("R6-X1", "control-absent", PROTO, PH,
     "if major >= (math.MaxInt-step)/10 {",
     "if major > (math.MaxInt-step)/10 {",
     "anchor uses >=, absent from source"),
    ("R6-X2", "control-compile", PROTO, PH,
     "\t\tif major > (math.MaxInt-step)/10 {\n\t\t\tmajor = math.MaxInt\n\t\t\tcontinue\n\t\t}\n",
     "",
     "guard deleted outright, math import unused"),
]


def run(cmd):
    proc = subprocess.run(cmd, shell=True, capture_output=True, text=True)
    return proc.returncode, proc.stdout + proc.stderr


def failed_tests(output):
    return sorted({line.split()[2] for line in output.splitlines()
                   if line.strip().startswith("--- FAIL:")})


def run_count(output):
    return sum(1 for line in output.splitlines() if line.strip().startswith("=== RUN"))


def calibrate():
    cal = {}
    for label, cmd in [
        ("census_mask", "go test %s -run 'TestDigitCensus' -count=1 -v" % TB),
        ("behavioral_mask_tb", "go test %s -skip 'TestDigitCensus' -count=1 -v" % TB),
        ("behavioral_mask_ph", "go test %s -skip 'TestDigitCensus' -count=1 -v" % PH),
    ]:
        code, out = run(cmd)
        cal[label] = {"exit": code, "run_lines": run_count(out)}
    return cal


def apply_and_test(mid, cls, path, pkg, old, new, detail):
    src = open(path).read()
    n = src.count(old)
    if n != 1:
        return {"id": mid, "class": cls, "disposition": "NOT_APPLIED",
                "detail": "%s; anchor sites=%d, want 1" % (detail, n)}
    open(path, "w").write(src.replace(old, new))
    try:
        bcode, bout = run("go build %s" % pkg)
        if bcode != 0:
            return {"id": mid, "class": cls, "disposition": "COMPILE_FAIL",
                    "detail": detail, "build": bout.strip().splitlines()[:3]}
        ccode, cout = run("go test %s -run 'TestDigitCensus' -count=1" % TB)
        census_failed = failed_tests(cout)
        hcode, hout = run("go test %s -skip 'TestDigitCensus' -count=1" % pkg)
        behavioral_failed = failed_tests(hout)
        killed = (ccode != 0) or (hcode != 0)
        split = ("BEHAVIORAL-ONLY" if hcode != 0 and ccode == 0 else
                 "CENSUS-ONLY" if ccode != 0 and hcode == 0 else
                 "BOTH" if killed else "NONE")
        return {"id": mid, "class": cls,
                "disposition": "KILLED" if killed else "SURVIVED",
                "split": split,
                "census_exit": ccode, "census_killed_by": census_failed,
                "behavioral_exit": hcode, "behavioral_killed_by": behavioral_failed,
                "detail": detail}
    finally:
        shutil.copyfile(PRISTINE[path], path)
        assert sha(path) == ORIG_SHA[path], "restore failed for %s" % path


def main():
    only = sys.argv[1:]
    results = []
    if not only or "CAL" in only:
        cal = calibrate()
        print(json.dumps({"id": "CAL", "masks": cal}), flush=True)
        results.append({"id": "CAL", "masks": cal})
    for mutant in MUTANTS + CONTROLS:
        if only and mutant[0] not in only:
            continue
        record = apply_and_test(*mutant)
        results.append(record)
        print(json.dumps(record), flush=True)
    tag = "-".join(only) if only else "all"
    with open(os.path.join(BACKUP, "battery-round6-%s.json" % tag), "w") as handle:
        json.dump(results, handle, indent=2)
    for path in TARGETS:
        assert sha(path) == ORIG_SHA[path], "final restore check failed for %s" % path
    print("ALL RESTORED")


main()
