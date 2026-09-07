import json,os,subprocess,sys,shutil,time

ROOT=os.getcwd()
BK=os.path.join(ROOT,".temp/TASK-260906-vmzk0y/rev5-review/backup")

MUTANTS=[
 # id, file, old, new, package, class
 ("TB-01","internal/terminalbackend/terminalbackend.go",
  "if major > (math.MaxInt-digit)/10 {","if major > math.MaxInt/10 {",
  "./internal/terminalbackend/","narrowing"),
 ("TB-02","internal/terminalbackend/terminalbackend.go",
  "if major > (math.MaxInt-digit)/10 {","if major > (math.MaxInt-digit)/10+1 {",
  "./internal/terminalbackend/","narrowing"),
 ("TB-03","internal/terminalbackend/descriptor.go",
  "if digit < '0' || digit > '9' {","if digit < '0' || digit > 'E' {",
  "./internal/terminalbackend/","narrowing"),
 ("TB-04","internal/terminalbackend/descriptor.go",
  "if digit < '0' || digit > '9' {","if digit < '0' || digit > ':' {",
  "./internal/terminalbackend/","narrowing"),
 ("TB-05","internal/terminalbackend/descriptor.go",
  "if digit < '0' || digit > '9' {","if digit < '/' || digit > '9' {",
  "./internal/terminalbackend/","narrowing"),
 ("TB-06","internal/terminalbackend/descriptor.go",
  "if digit < '0' || digit > '9' {","if digit < '.' || digit > '9' {",
  "./internal/terminalbackend/","narrowing"),
 ("TB-07","internal/terminalbackend/descriptor.go",
  "if digit < '0' || digit > '9' {","if digit < '-' || digit > '9' {",
  "./internal/terminalbackend/","narrowing"),
 ("TB-08","internal/terminalbackend/descriptor.go",
  "if digit < '0' || digit > '9' {","if (digit < '0' || digit > '9') && digit != 'E' {",
  "./internal/terminalbackend/","token-preserving"),
 ("TB-09","internal/terminalbackend/descriptor.go",
  "if digit < '0' || digit > '9' {","if (digit < '0' || digit > '9') && digit != 'e' {",
  "./internal/terminalbackend/","token-preserving"),
 ("TB-10","internal/terminalbackend/descriptor.go",
  "if value > 100 {","if value > 1000 {",
  "./internal/terminalbackend/","narrowing"),
 ("TB-11","internal/terminalbackend/descriptor.go",
  "if value < 1 || value > 1000 {","if value < 1 || value > 1001 {",
  "./internal/terminalbackend/","narrowing"),
 ("TB-12","internal/terminalbackend/descriptor.go",
  "if !semverPattern.MatchString(protocolVersion) || semverMajor(protocolVersion) != 1 {",
  "if !semverPattern.MatchString(protocolVersion) || semverMajor(protocolVersion) > 3 {",
  "./internal/terminalbackend/","narrowing"),
 ("TB-13","internal/terminalbackend/descriptor.go",
  "if len(object) != len(descriptorMembers) {","if len(object) < len(descriptorMembers) {",
  "./internal/terminalbackend/","narrowing"),
 ("TB-14","internal/terminalbackend/descriptor.go",
  "if _, err := scalar.ParseUUIDv7(instanceID); err != nil {",
  "if _, err := scalar.ParseUUIDv7(instanceID); err != nil && len(instanceID) != 4 {",
  "./internal/terminalbackend/","narrowing"),
 ("TB-15","internal/terminalbackend/descriptor.go",
  "\tif err := CheckProviderDescriptor(descriptor.Binding(), binding); err != nil {",
  "\tif err := CheckProviderDescriptor(descriptor.Binding(), binding); err != nil && descriptor.TerminalBackendID != binding.BackendID {",
  "./internal/terminalbackend/","narrowing"),
 ("TB-16","internal/terminalbackend/descriptor.go",
  "if _, err := scalar.ParseDigest(bindingID); err != nil {",
  "if _, err := scalar.ParseDigest(bindingID); err != nil && bindingID != \"\" {",
  "./internal/terminalbackend/","narrowing"),
 ("PH-01","internal/provhost/protocol.go",
  "if major > (math.MaxInt-step)/10 {","if major > math.MaxInt/10 {",
  "./internal/provhost/","narrowing"),
 ("PH-02","internal/provhost/protocol.go",
  "\t\tif digit < '0' || digit > '9' {","\t\tif digit < '0' || digit > ':' {",
  "./internal/provhost/","narrowing"),
 ("PH-03","internal/provhost/protocol.go",
  "\tobservedMajor, _ := parseMajor(version)","\tobservedMajor := ProtocolMajor",
  "./internal/provhost/","equivalence-probe"),
 ("PH-04","internal/provhost/protocol.go",
  "\t\tif major, recognized := parseMajor(version); recognized && major != ProtocolMajor {\n\t\t\tfailure, err := failMismatch(\"foreign protocol major\", version)",
  "\t\tif major, recognized := parseMajor(version); recognized && major > ProtocolMajor {\n\t\t\tfailure, err := failMismatch(\"foreign protocol major\", version)",
  "./internal/provhost/","narrowing"),
 ("PR-01","internal/provider/lift.go",
  "\t\tDetails: axerror.Details{},","\t\tDetails: axerror.Details{\"detail\": failure.detail},",
  "./internal/provider/","leak"),
 ("PR-02","internal/provider/lift.go",
  "\t\tMessage: message,","\t\tMessage: failure.Error(),",
  "./internal/provider/","leak"),
 ("PR-03","internal/provider/lift.go",
  "\t\tif _, err := axerror.ExitCodeFor(axerror.Version100, axerror.Code(failure.code)); err != nil {\n\t\t\treturn nil, err\n\t\t}\n\t\treturn nil, failure",
  "\t\treturn nil, failure",
  "./internal/provider/","arm-deletion"),
 ("PR-04","internal/provider/lift.go",
  "\t\tCause:   failure,","\t\tCause:   nil,",
  "./internal/provider/","narrowing"),

 ("TB-17","internal/terminalbackend/descriptor.go",
  "if value > 100 {","if value > 922337203685477580 {",
  "./internal/terminalbackend/","narrowing"),
 ("TB-18","internal/terminalbackend/descriptor.go",
  "\tif value < 1 || value > 1000 {","\tif value < 0 || value > 1000 {",
  "./internal/terminalbackend/","narrowing"),
 ("PH-05","internal/provhost/protocol.go",
  "\t\tif digit < '0' || digit > '9' {","\t\tif digit < '/' || digit > '9' {",
  "./internal/provhost/","narrowing"),
 ("PH-06","internal/provhost/protocol.go",
  "\t\tif rest[i] < '0' || rest[i] > '9' {","\t\tif rest[i] < '0' || rest[i] > ':' {",
  "./internal/provhost/","narrowing"),
]

def run(cmd,timeout=420):
    p=subprocess.run(cmd,shell=True,capture_output=True,text=True,timeout=timeout)
    return p.returncode,p.stdout+p.stderr

results=[]
only=sys.argv[1:] if len(sys.argv)>1 else None
for mid,path,old,new,pkg,cls in MUTANTS:
    if only and mid not in only: continue
    src=open(path).read()
    n=src.count(old)
    if n!=1:
        results.append(dict(id=mid,cls=cls,pkg=pkg,status="NOT_APPLIED",note=f"anchor count={n}"))
        continue
    open(path,'w').write(src.replace(old,new,1))
    try:
        rc,out=run(f"go build ./... 2>&1")
        if rc!=0:
            results.append(dict(id=mid,cls=cls,pkg=pkg,status="COMPILE_FAIL",note=out.strip().splitlines()[:3]))
        else:
            t0=time.time()
            rc,out=run(f"go test {pkg} 2>&1")
            fails=[l.strip() for l in out.splitlines() if l.strip().startswith("--- FAIL")]
            results.append(dict(id=mid,cls=cls,pkg=pkg,status="KILLED" if rc!=0 else "SURVIVED",
                                secs=round(time.time()-t0,1),fails=fails[:8],nfail=len(fails)))
    finally:
        shutil.copy(os.path.join(BK,os.path.basename(path)),path)
print(json.dumps(results,indent=1))
