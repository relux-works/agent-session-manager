import subprocess, hashlib
SRC='internal/terminalbackend/manifest.go'
ORIG=open(SRC).read()
M={
 "M33-omission": ('\tfor capability := range manifest {\n\t\tif _, present := indexProbeCapability(probe, capability); !present {\n\t\t\treturn mismatchf("probe omission of manifest claim")\n\t\t}\n\t}\n',''),
 "M53-static-echo": ('\t\tcase proved.Origin == OriginStatic:\n\t\t\tif !equalClaim(proved, static) {\n\t\t\t\treturn mismatchf("probe static claim echo")\n\t\t\t}\n','\t\tcase proved.Origin == OriginStatic:\n'),
 "M35-protocol-membership": ('\tif !member {\n\t\treturn mismatchf("probe protocol membership")\n\t}\n','\t_ = member\n'),
 "M34-evidence-false-claim": ('\t\tif !present || !claim.Value {\n\t\t\treturn nil, mismatchf("evidence claim binding")\n\t\t}\n','\t\t_ = claim\n\t\tif !present {\n\t\t\treturn nil, mismatchf("evidence claim binding")\n\t\t}\n'),
 "M56-record-executable-digest": ('\tif manifest.ExecutableDigest != record.ExecutableDigest {\n\t\treturn &Error{Code: CodeUntrusted, BackendID: manifest.TerminalBackendID, Detail: "executable substitution"}\n\t}\n',''),
 "M24-expiry-upper-boundary": ('\tif observed.After(now) || !now.Before(expires) {','\tif observed.After(now) || now.After(expires) {'),
}
for mid,(old,new) in M.items():
    assert ORIG.count(old)==1, mid
    open(SRC,'w').write(ORIG.replace(old,new,1))
    r=subprocess.run(['go','test','./internal/terminalbackend/','-count=1','-run','TestReviewArms','-v'],capture_output=True,text=True)
    print("### "+mid)
    for line in r.stdout.splitlines():
        if 'armreview_test.go:58' in line: print("   "+line.strip())
    r2=subprocess.run(['go','test','./internal/terminalbackend/','-count=1'],capture_output=True,text=True)
    print("   shipped suite:", "KILLED" if r2.returncode!=0 else "SURVIVED")
    open(SRC,'w').write(ORIG)
assert hashlib.sha256(open(SRC,'rb').read()).hexdigest()=="d5a7b8cccf9670f98eb6e85cdbe76226fc8a293685177f685e0111fa74c7ab85"
print("reverted byte-identical")
