#!/usr/bin/env python3
"""Reviewer mutation battery for TASK-260906-vmzk0y CR rev3.

Each mutant is applied to a pristine copy of the target file, the named
test scope is run as a standalone process, and the file is restored
byte-identically afterwards (asserted).
"""
import hashlib, json, os, shutil, subprocess, sys

ROOT = os.getcwd()
PROTO = "internal/provhost/protocol.go"
INV = "internal/provhost/refusal_arm_inventory_test.go"
BACKUP = ".temp/TASK-260906-vmzk0y/review-rev3"

TARGETS = {PROTO: os.path.join(BACKUP, "protocol.go.pristine"),
           INV: os.path.join(BACKUP, "inventory_test.go.pristine")}

for path, pristine in TARGETS.items():
    shutil.copyfile(path, pristine)

def sha(path):
    return hashlib.sha256(open(path, "rb").read()).hexdigest()

PRISTINE_SHA = {p: sha(p) for p in TARGETS}

PEEK_OLD_UNUSED = '\t\t\t\tif rawVersion, present := members["protocol_version"]; present {\n\t\t\t\t\tvar version string\n\t\t\t\t\tif err := json.Unmarshal(rawVersion, &version); err == nil && version != ProtocolVersion {\n\t\t\t\t\t\tif major, recognized := parseMajor(version); recognized && major != ProtocolMajor {'
GATE_OLD = '''	if version != ProtocolVersion {
		if major, recognized := parseMajor(version); recognized && major != ProtocolMajor {'''

PEEK_OLD = "\t\t\t\t\t" + "if major, recognized := parseMajor(version); recognized && major != ProtocolMajor {"

MUTANTS = [
    # id, class, file, old, new, detail
    ("M1", "narrowing", PROTO,
     "\t\t\tmajor = math.MaxInt\n\t\t\tcontinue\n",
     "\t\t\treturn math.MaxInt, true\n",
     "P3: round-2 early-return shape; giant-with-malformed-rest reads recognized"),
    ("M2", "narrowing", PROTO,
     "if major > (math.MaxInt-step)/10 {",
     "if major > math.MaxInt {",
     "P3: guard never fires; accumulation wraps again"),
    ("M3", "narrowing", PROTO,
     "if major > (math.MaxInt-step)/10 {",
     "if major > math.MaxInt/10 {",
     "P3: threshold off-by-one (drops the step term)"),
    ("M4", "narrowing", PROTO,
     "\t\tif len(rest) == 0 {",
     "\t\tif len(rest) < 0 {",
     "P5: empty-rest arm never fires (token preserved)"),
    ("M5", "narrowing", PROTO,
     "\t\t\tif rest[i] < '0' || rest[i] > '9' {",
     "\t\t\tif rest[i] < '/' || rest[i] > '9' {",
     "P6: rest admits exactly one rejected member ('/')"),
    ("M6", "narrowing", PROTO,
     "\t\tif digit < '0' || digit > '9' {",
     "\t\tif digit < '0' || digit > ':' {",
     "P2: major admits exactly one rejected member (':')"),
    ("M7", "narrowing", PROTO,
     "\tif len(parts) != 3 {",
     "\tif len(parts) < 3 {",
     "P1: admits exactly the over-long shape (4+ parts)"),
    ("M8", "narrowing", PROTO,
     "\tif len(parts[0]) == 0 {",
     "\tif len(parts[0]) < 0 {",
     "P4: empty-major arm never fires (token preserved)"),
    ("M9", "arm-deletion", PROTO,
     "\tfor _, rest := range parts[1:] {\n\t\tif len(rest) == 0 {\n\t\t\treturn 0, false\n\t\t}\n\t\tfor i := 0; i < len(rest); i++ {\n\t\t\tif rest[i] < '0' || rest[i] > '9' {\n\t\t\t\treturn 0, false\n\t\t\t}\n\t\t}\n\t}\n",
     "",
     "P5+P6: whole minor/patch validation deleted"),
    ("M10", "arm-deletion", PROTO,
     "\tif len(parts) != 3 {\n\t\treturn 0, false\n\t}\n",
     "",
     "P1: three-part shape check deleted"),
    ("M11", "narrowing", PROTO,
     "axerror.ContainingContract{ID: ProtocolID, Major: observedMajor}",
     "axerror.ContainingContract{ID: ProtocolID, Major: observedMajor + 1}",
     "C1: failure-error contract shifted off the observed major"),
    ("M12", "equivalence-probe", PROTO,
     "\tobservedMajor, _ := parseMajor(version)\n\tchild, err := axerror.DecodeBound(axerror.ContainingContract{ID: ProtocolID, Major: observedMajor}, members[\"error\"])",
     "\tchild, err := axerror.DecodeBound(axerror.ContainingContract{ID: ProtocolID, Major: 2}, members[\"error\"])",
     "C1: revert to the hardcoded major-2 contract (expected equivalent today)"),
    ("M13b", "narrowing", PROTO,
     PEEK_OLD,
     PEEK_OLD.replace("recognized && major != ProtocolMajor", "recognized || major != ProtocolMajor"),
     "K1: foreignMajor peek widened to unrecognized versions (re-anchored)"),
    ("M14b", "narrowing", PROTO,
     PEEK_OLD,
     PEEK_OLD.replace("recognized && major != ProtocolMajor", "recognized && major > ProtocolMajor"),
     "K2: foreignMajor peek narrowed to majors above 2 (re-anchored)"),
    ("M16", "narrowing", PROTO,
     GATE_OLD,
     GATE_OLD.replace("recognized && major != ProtocolMajor", "recognized && major > ProtocolMajor"),
     "V1: version-gate mismatch arm narrowed to majors above 2 (re-anchored)"),
    ("M15", "census-only", INV,
     'if ok && function.Name.Name == "parseMajor" {',
     'if ok && function.Name.Name == "parseMajorXX" {',
     "census derives zero parse arms (production untouched)"),
]

CONTROLS = [
    ("X1", "control-absent", PROTO,
     "if major >= (math.MaxInt-step)/10 {",
     "if major > (math.MaxInt-step)/10 {",
     "anchor uses >=, absent from source"),
    ("X2", "control-compile", PROTO,
     "\t\tif major > (math.MaxInt-step)/10 {\n\t\t\tmajor = math.MaxInt\n\t\t\tcontinue\n\t\t}\n",
     "",
     "guard deleted outright, math import unused"),
]

def run(cmd):
    p = subprocess.run(cmd, shell=True, capture_output=True, text=True)
    return p.returncode, p.stdout + p.stderr

def apply_and_test(mid, cls, path, old, new, detail):
    src = open(path).read()
    n = src.count(old)
    if n != 1:
        return {"id": mid, "class": cls, "disposition": "NOT_APPLIED",
                "detail": f"{detail}; anchor sites={n}, want 1"}
    open(path, "w").write(src.replace(old, new))
    try:
        bcode, bout = run("go build ./internal/provhost/")
        if bcode != 0:
            return {"id": mid, "class": cls, "disposition": "COMPILE_FAIL",
                    "detail": detail, "build": bout.strip().splitlines()[:3]}
        tcode, tout = run("go test ./internal/provhost/ -count=1")
        failed = sorted({l.split()[2] for l in tout.splitlines()
                         if l.strip().startswith("--- FAIL:")})
        return {"id": mid, "class": cls,
                "disposition": "KILLED" if tcode != 0 else "SURVIVED",
                "killed_by": failed, "detail": detail}
    finally:
        shutil.copyfile(TARGETS[path], path)
        assert sha(path) == PRISTINE_SHA[path], f"restore failed for {path}"

only = sys.argv[1:]
results = []
for m in MUTANTS + CONTROLS:
    if only and m[0] not in only:
        continue
    r = apply_and_test(*m)
    results.append(r)
    print(json.dumps(r), flush=True)

tag = "-".join(only) if only else "all"
json.dump(results, open(os.path.join(BACKUP, f"battery-{tag}.json"), "w"), indent=2)
for p in TARGETS:
    assert sha(p) == PRISTINE_SHA[p], f"final restore check failed for {p}"
print("ALL RESTORED")
