import subprocess, hashlib, os
os.chdir('/Users/iv/Developer/ReluxWorks/agent-session-manager/.temp/STORY-260830-3m2mw8/worktree')
SRC='internal/terminalbackend/manifest.go'
ORIG=open(SRC).read()
BASE=hashlib.sha256(ORIG.encode()).hexdigest()

OLD = '\tif len(names) == 0 || len(names) > 4 {\n\t\treturn nil, mismatchf("platforms bound")\n\t}\n'
NEW = OLD + '\tif len(names) > 8 {\n\t\treturn nil, mismatchf("platforms review probe bound")\n\t}\n'
assert ORIG.count(OLD)==1, ORIG.count(OLD)
open(SRC,'w').write(ORIG.replace(OLD,NEW,1))
try:
    for cmd in (['gofmt','-l','.'], ['go','vet','./...'], ['go','test','./...','-count=1']):
        r=subprocess.run(cmd,capture_output=True,text=True)
        print(' '.join(cmd), '-> exit', r.returncode)
        if r.returncode!=0:
            print((r.stdout+r.stderr)[-1500:])
        elif cmd[0]=='gofmt' and r.stdout.strip():
            print('gofmt listed:', r.stdout.strip())
    r=subprocess.run(['go','run','./internal/traceability/cmd/tracecheck','-root','.'],capture_output=True,text=True)
    print('tracecheck -> exit', r.returncode)
    print((r.stdout+r.stderr)[-800:])
finally:
    open(SRC,'w').write(ORIG)
    assert hashlib.sha256(open(SRC,'rb').read()).hexdigest()==BASE, "REVERT FAILED"
    print("\nreverted byte-identical to", BASE)
