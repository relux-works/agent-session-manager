import json, subprocess, sys, os, time
ROOT = "/tmp/r6scratch"
def run(cmd, timeout=600):
    p = subprocess.run(cmd, shell=True, cwd=ROOT, capture_output=True, text=True, timeout=timeout)
    return p.returncode, p.stdout, p.stderr
def measure(m):
    path, old, new = m["file"], m["old"], m["new"]
    full = os.path.join(ROOT, path); backup = open(full).read()
    if backup.count(old) != 1:
        return {"id": m["id"], "verdict": "NOT-APPLIED", "detail": "anchor count=%d" % backup.count(old)}
    open(full,"w").write(backup.replace(old,new))
    if new not in open(full).read():
        open(full,"w").write(backup); return {"id": m["id"], "verdict":"NOT-APPLIED","detail":"absent on disk"}
    try:
        rc,o,e = run("go build ./... 2>&1")
        if rc != 0: return {"id": m["id"], "verdict":"NOT-COMPILED","detail":(o+e)[:300]}
        rc,o,e = run("go test ./... -count=1 2>&1")
        fails=[l for l in (o+e).splitlines() if l.startswith("FAIL") or "--- FAIL" in l]
        return {"id": m["id"], "verdict": "KILLED" if rc!=0 else "SURVIVED", "fails": fails[:8]}
    finally:
        open(full,"w").write(backup)
if __name__=="__main__":
    res=[]
    for m in json.load(open(sys.argv[1])):
        t0=time.time(); r=measure(m); r["seconds"]=round(time.time()-t0,1); r["note"]=m.get("note",""); res.append(r)
        print(json.dumps(r), flush=True)
    json.dump(res, open(sys.argv[2],"w"), indent=1)
    rc,o,_=run("go test ./... -count=1 >/dev/null 2>&1; echo $?"); print("RESTORED-SUITE-EXIT="+o.strip())
