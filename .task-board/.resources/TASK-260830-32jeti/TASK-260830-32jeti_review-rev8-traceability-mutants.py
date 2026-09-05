import json, subprocess, sys, shutil, os, hashlib
ROOT="/Users/iv/Developer/ReluxWorks/agent-session-manager/.temp/STORY-260830-3jqsx1/worktree"
OWN=os.path.join(ROOT,"internal/traceability/ownership.v0.5.0.json")
GO=os.path.join(ROOT,"internal/traceability/traceability.go")
B=os.path.join(ROOT,".temp/TASK-260830-32jeti/mutants")
shutil.copy(OWN, B+"/own.bak"); shutil.copy(GO, B+"/go.bak")

def restore():
    shutil.copy(B+"/own.bak", OWN); shutil.copy(B+"/go.bak", GO)

def run(cmd):
    p=subprocess.run(cmd, shell=True, cwd=ROOT, capture_output=True, text=True)
    return p.returncode, (p.stdout+p.stderr)

def repin():
    # emulate a self-minter: recompute and update the pinned digest
    rc,out = run("go test ./internal/traceability/ -run TestVerifyRepositoryAcceptsExactOwnership 2>&1")
    import re
    m=re.search(r'projection digest ([0-9a-f]{64}) differs from reviewed', out)
    if not m: return None
    new=m.group(1)
    src=open(GO).read()
    import re as r2
    src2=r2.sub(r'reviewedOwnershipCanonicalSHA256 = "[0-9a-f]{64}"',
                f'reviewedOwnershipCanonicalSHA256 = "{new}"', src)
    assert src2!=src
    open(GO,"w").write(src2)
    return new

def mutate(fn):
    d=json.load(open(OWN))
    fn(d)
    json.dump(d, open(OWN,"w"), indent=2)
    open(OWN,"a").write("\n")

results=[]
def battery(name, desc, fn, do_repin):
    restore()
    mutate(fn)
    pin = repin() if do_repin else None
    rc,out = run("go test ./internal/traceability/... -count=1")
    tc_rc, tc_out = run("go run ./internal/traceability/cmd/tracecheck -root .")
    verdict = "KILLED" if (rc!=0 or tc_rc!=0) else "SURVIVED"
    first = [l for l in out.split("\n") if "FAIL" in l or "differs" in l or "ownership" in l or "acceptance" in l][:3]
    results.append((name, desc, "repinned" if do_repin else "pin-left", verdict, rc, tc_rc, " | ".join(first)[:260]))
    restore()

# baseline
restore()
rc,out=run("go test ./internal/traceability/... -count=1")
tc,_=run("go run ./internal/traceability/cmd/tracecheck -root .")
results.append(("M0","baseline unmutated","-", "GREEN" if rc==0 and tc==0 else "RED", rc, tc, ""))

def drop_sibling(d):
    d['acceptance_cases']=[e for e in d['acceptance_cases'] if e['id']!='terminal-lifecycle-conformance']
def repoint_test(d):
    for e in d['acceptance_cases']:
        if e['id']=='terminal-lifecycle-conformance':
            e['tests'][0]['declaration']='TestThisFunctionDoesNotExistAnywhere'
def repoint_prod(d):
    for e in d['acceptance_cases']:
        if e['id']=='terminal-lifecycle-conformance':
            e['production']['declaration']='NoSuchExportedSymbol'
def crossclaim(d):
    # sibling case claims THIS story's production file: a cross-package self-mint
    for e in d['acceptance_cases']:
        if e['id']=='terminal-lifecycle-conformance':
            e['production']['path']='internal/provhost/quiesce.go'
            e['production']['declaration']='DecodeQuiesceProof'
def narrow_tests(d):
    # keep the row, delete all but one test: a NARROWING, not a deletion
    for e in d['acceptance_cases']:
        if e['id']=='terminal-lifecycle-conformance':
            e['tests']=e['tests'][:1]
def reformat_only(d):
    pass  # json.dump reformat with no semantic change

battery("M1","drop one sibling acceptance case (deletion)",drop_sibling,True)
battery("M2","sibling row: test decl -> nonexistent (narrow)",repoint_test,True)
battery("M3","sibling row: production decl -> nonexistent (narrow)",repoint_prod,True)
battery("M4","sibling row cross-claims this story's production site",crossclaim,True)
battery("M5","sibling row: 12 tests -> 1 test (narrowing, row kept)",narrow_tests,True)
battery("M6","reformat only, NO semantic change, pin NOT updated",reformat_only,False)

restore()
print(f"{'ID':4} {'pin':10} {'verdict':9} {'gorc':5} {'tcrc':5} desc")
for r in results:
    print(f"{r[0]:4} {r[2]:10} {r[3]:9} {r[4]:<5} {r[5]:<5} {r[1]}")
    if r[6]: print(f"       -> {r[6]}")
print()
print("tree restored, sha check:")
