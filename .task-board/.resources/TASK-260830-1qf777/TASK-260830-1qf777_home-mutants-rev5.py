"""Round-5 mutation sweep for CR rev4 finding B1.

Every mutant is applied to production code only. The gate is the WHOLE
internal/config package with -count=1, never a -run mask, so a mutant cannot
look dead because an unrelated case was excluded.
"""
import subprocess, os, sys

ROOT = os.getcwd()
LOADER = "internal/config/loader.go"
JOIN = "errors.Join(ErrPlatformDefaultUnavailable, inputs.homeDirError)"
DROP = "ErrPlatformDefaultUnavailable"
GATE = "go test ./internal/config -count=1"

CAPTURE = "\thome, homeErr := os.UserHomeDir()\n"

def macos_site(head, tail):
    return head + '\t\t\treturn "", %s\n\t\t}\n' % JOIN + tail

MUTANTS = []

# --- the two mutants that survived the rev4 suite -------------------------
MUTANTS.append(("H01", LOADER, "\t\thomeDirError: homeErr,\n", "\t\thomeDirError: nil,\n",
    "rev4 SURVIVOR 1: the captured os.UserHomeDir cause is dropped at the capture"))
MUTANTS.append(("H02", LOADER, CAPTURE, CAPTURE + '\thome = ""\n',
    "rev4 SURVIVOR 2: the captured home is emptied, so every home-derived default refuses"))

# --- other ways the capture can stop being the real capture ---------------
MUTANTS.append(("H03", LOADER, CAPTURE, CAPTURE + "\thome = os.TempDir()\n",
    "constant home: the capture is replaced by a plausible non-empty constant"))
MUTANTS.append(("H04", LOADER, "\t\thomeDirError: homeErr,\n",
    '\t\thomeDirError: errors.New("user home unavailable"),\n',
    "substituted cause: a self-minted error replaces the captured one"))
MUTANTS.append(("H05", LOADER, CAPTURE, CAPTURE + '\thome = home + "x"\n',
    "drifted home: the capture is corrupted rather than emptied"))

# --- per-site cause drops: macOS lane -------------------------------------
MACOS_SITES = [
    ("H06", "macos config-file",
     '\t\tbase := nonEmptyEnvironment(inputs, "XDG_CONFIG_HOME")\n\t\tif base == "" {\n\t\t\tif inputs.HomeDir == "" {\n\t\t\t\treturn "", %s\n\t\t\t}\n' % JOIN),
    ("H07", "macos data-root",
     '\t\t\treturn "", %s\n\t\t}\n\t\treturn join(inputs.Platform, inputs.HomeDir, "Library", "Application Support", "ax"), nil\n' % JOIN),
    ("H08", "macos state-root",
     '\t\t\treturn "", %s\n\t\t}\n\t\treturn join(inputs.Platform, inputs.HomeDir, "Library", "Application Support", "ax", "state"), nil\n' % JOIN),
    ("H09", "macos cache-root",
     '\t\t\treturn "", %s\n\t\t}\n\t\treturn join(inputs.Platform, inputs.HomeDir, "Library", "Caches", "ax"), nil\n' % JOIN),
]
# --- per-site cause drops: linux/WSL2 lane --------------------------------
LINUX_SITES = [
    ("H10", "linux config-file", "XDG_CONFIG_HOME"),
    ("H11", "linux data-root", "XDG_DATA_HOME"),
    ("H12", "linux state-root", "XDG_STATE_HOME"),
    ("H13", "linux cache-root", "XDG_CACHE_HOME"),
]
for mid, label, find in MACOS_SITES:
    MUTANTS.append((mid, LOADER, find, find.replace(JOIN, DROP, 1),
        "cause drop at the %s site: that class refuses without the operator's cause" % label))
for mid, label, variable in LINUX_SITES:
    find = '\t\tbase = nonEmptyEnvironment(inputs, "%s")\n\t\tif base == "" {\n\t\t\tif inputs.HomeDir == "" {\n\t\t\t\treturn "", %s\n\t\t\t}\n' % (variable, JOIN)
    MUTANTS.append((mid, LOADER, find, find.replace(JOIN, DROP, 1),
        "cause drop at the %s site: that class refuses without the operator's cause" % label))

# --- gate deletion and layout drift ---------------------------------------
MUTANTS.append(("H14", LOADER,
    '\t\tif inputs.HomeDir == "" {\n\t\t\treturn "", %s\n\t\t}\n\t\treturn join(inputs.Platform, inputs.HomeDir, "Library", "Application Support", "ax"), nil\n' % JOIN,
    '\t\treturn join(inputs.Platform, inputs.HomeDir, "Library", "Application Support", "ax"), nil\n',
    "gate deletion at macos data-root: an unavailable home yields a rooted path instead of a refusal"))
MUTANTS.append(("H15", LOADER,
    '\t\treturn join(inputs.Platform, inputs.HomeDir, "Library", "Caches", "ax"), nil\n',
    '\t\treturn join(inputs.Platform, inputs.HomeDir, "Library", "Application Support", "ax", "cache"), nil\n',
    "layout drift at macos cache-root: a home-derived class stops matching the Section 3.2 default"))
MUTANTS.append(("H16", LOADER,
    '\t\t\tbase = join(inputs.Platform, inputs.HomeDir, ".local", "share")\n',
    '\t\t\tbase = join(inputs.Platform, inputs.HomeDir, ".local", "data")\n',
    "layout drift at linux data-root: a home-derived class stops matching the Section 3.2 default"))

def run(cmd):
    return subprocess.run(cmd, shell=True, cwd=ROOT, capture_output=True, text=True)

def read(p):
    with open(p) as f: return f.read()

def write(p, s):
    with open(p, "w") as f: f.write(s)

print("=== POSITIVE CONTROL: unmutated tree, whole package ===")
r = run(GATE)
print(r.stdout.strip())
print("exit=%d (want 0)" % r.returncode)
if r.returncode != 0:
    print(r.stderr[-3000:])
    sys.exit("baseline is not green; aborting the sweep")

results = []
for mid, path, find, repl, note in MUTANTS:
    original = read(path)
    count = original.count(find)
    if count != 1:
        results.append((mid, "INVALID", note))
        print("\n%s INVALID  anchor occurrences=%d  %s" % (mid, count, note))
        continue
    write(path, original.replace(find, repl, 1))
    try:
        r = run(GATE)
        verdict = "RED" if r.returncode != 0 else "SURVIVOR"
        results.append((mid, verdict, note))
        print("\n%s %s exit=%d  %s" % (mid, verdict, r.returncode, note))
        failing = [l.strip() for l in r.stdout.splitlines() if l.strip().startswith("--- FAIL")]
        for line in failing[:8]:
            print("      killed by: %s" % line)
        if verdict == "SURVIVOR":
            print("      !!! SURVIVOR, reported as a survivor and not folded into the RED count")
    finally:
        write(path, original)

print("\n=== RESTORED-TREE VERIFICATION ===")
r = run(GATE)
print(r.stdout.strip())
print("exit=%d (want 0)" % r.returncode)

red = sum(1 for x in results if x[1] == "RED")
survivors = [x for x in results if x[1] == "SURVIVOR"]
invalid = [x for x in results if x[1] == "INVALID"]
print("\nSUMMARY: %d mutants, %d RED, %d SURVIVOR, %d INVALID" % (len(results), red, len(survivors), len(invalid)))
for mid, verdict, note in survivors + invalid:
    print("  %s %s  %s" % (mid, verdict, note))
