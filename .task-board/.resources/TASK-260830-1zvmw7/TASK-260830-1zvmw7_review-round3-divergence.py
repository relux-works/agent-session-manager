import subprocess, hashlib, os, re
os.chdir('/Users/iv/Developer/ReluxWorks/agent-session-manager/.temp/STORY-260830-3m2mw8/worktree')
SRC='internal/terminalbackend/manifest.go'
ORIG=open(SRC).read(); BASE=hashlib.sha256(ORIG.encode()).hexdigest()
M=[
 ("S5r-registry-confers-manifest","a registry row confers the bypassed operation manifest (CheckOperation hardcodes the bypass; CapabilitiesForOperation derives it)",
  '\t"remote_attach": {\n\t\tgenerationVariable:   true,\n',
  None),
 ("S10-checkoperation-empty-capabilities","CheckOperation ignores capability set entirely for non-bypassed ops",
  None,None),
]
# resolve S5r anchor from the actual registry text
m = re.search(r'\t"remote_attach": \{\n(?:.*\n)*?\t\tdependentOperations:  \[\]string\{([^}]*)\},\n', ORIG)
assert m, "remote_attach row not found"
old = m.group(0)
new = old.replace('dependentOperations:  []string{%s},' % m.group(1),
                  'dependentOperations:  []string{"manifest", %s},' % m.group(1))
assert ORIG.count(old)==1
open(SRC,'w').write(ORIG.replace(old,new,1))
try:
    v=subprocess.run(['go','vet','./internal/terminalbackend/'],capture_output=True,text=True)
    print('vet exit',v.returncode, v.stderr[-300:] if v.returncode else '')
    r=subprocess.run(['go','test','./internal/terminalbackend/','-count=1','-v'],capture_output=True,text=True)
    print('S5r ->', 'KILLED' if r.returncode!=0 else 'SURVIVED')
    lines=r.stdout.splitlines()
    fails=[mm.group(1) for mm in (re.match(r'\s*--- FAIL: (\S+)', l) for l in lines) if mm]
    print('failing:', ', '.join(fails[:8]))
    for l in lines:
        if '_test.go:' in l and ('want' in l or 'error =' in l):
            print('   ', l.strip())
            break
finally:
    open(SRC,'w').write(ORIG)
    assert hashlib.sha256(open(SRC,'rb').read()).hexdigest()==BASE, "REVERT FAILED"
    print('reverted byte-identical')
