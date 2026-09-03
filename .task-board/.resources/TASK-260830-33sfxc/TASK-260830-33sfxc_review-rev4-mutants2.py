import json, os, subprocess, sys, shutil

def read(p): return open(p, encoding="utf-8").read()
def write(p, s): open(p, "w", encoding="utf-8").write(s)
def run(cmd):
    p = subprocess.run(cmd, shell=True, capture_output=True, text=True)
    return p.returncode, p.stdout + p.stderr

MUTANTS = [
 dict(name="command-agreement-narrowed-to-spare-the-covered-pair", file="internal/cliresult/client.go",
      old="	if result.Command() != command {",
      new="	if result.Command() != command && command != CommandDoctor {"),
 dict(name="command-agreement-narrowed-to-admit-a-forged-takeover-document", file="internal/cliresult/client.go",
      old="	if result.Command() != command {",
      new="	if result.Command() != command && result.Command() != CommandTakeover {"),
 dict(name="command-agreement-narrowed-to-admit-a-forged-doctor-document", file="internal/cliresult/client.go",
      old="	if result.Command() != command {",
      new="	if result.Command() != command && result.Command() != CommandDoctor {"),
]
results=[]
for m in MUTANTS:
    path=m["file"]; original=read(path)
    assert m["old"] in original, m["name"]
    write(path, original.replace(m["old"], m["new"], 1))
    assert m["new"] in read(path)
    code,out = run("go build ./... 2>&1")
    if code!=0:
        write(path, original); results.append(dict(mutant=m["name"], status="DID_NOT_COMPILE", output=out[-500:])); continue
    code,out = run("go test ./internal/cliresult/... ./internal/axerror/... ./internal/traceability/... -count=1 2>&1")
    write(path, original)
    fails = sorted({l.strip().split()[2] for l in out.splitlines() if l.strip().startswith("--- FAIL:")})
    results.append(dict(mutant=m["name"], status="KILLED" if code!=0 else "SURVIVED", exit_code=code, killed_by=fails[:8]))
    print(json.dumps(results[-1])); sys.stdout.flush()
code,out=run("go test ./internal/cliresult/... ./internal/axerror/... -count=1 2>&1")
print("restored:", code)
write(".temp/TASK-260830-33sfxc/review-rev4/independent-mutation-2.json", json.dumps(dict(results=results, restored=code), indent=1))
