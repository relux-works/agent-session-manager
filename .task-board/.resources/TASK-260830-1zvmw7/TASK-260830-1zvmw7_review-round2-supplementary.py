import subprocess, hashlib, json, re, os
os.chdir('/Users/iv/Developer/ReluxWorks/agent-session-manager/.temp/STORY-260830-3m2mw8/worktree')
SRC='internal/terminalbackend/manifest.go'
ORIG=open(SRC).read(); BASE=hashlib.sha256(ORIG.encode()).hexdigest()
M=[
 ("S1-GC-aliasing-positive-control","parseClaim returns the registry row slices (round-1 positive control)",
  '\t\tDependentOperations:  operations,\n\t\tEvidenceRequirements: requirements,\n',
  '\t\tDependentOperations:  row.dependentOperations,\n\t\tEvidenceRequirements: row.evidenceRequirements,\n'),
 ("S2-M37-reanchored","CheckOperation operation-vocabulary gate deleted",
  '\tif !operationVocabulary[operation] {\n\t\treturn mismatchf("operation vocabulary")\n\t}\n\tif operation == "manifest" || operation == "probe" {',
  '\tif operation == "manifest" || operation == "probe" {'),
 ("S3-F1-bypass-widen","F1 fix widened: create also bypasses the capability gate",
  '\tif operation == "manifest" || operation == "probe" {\n\t\treturn nil\n\t}\n',
  '\tif operation == "manifest" || operation == "probe" || operation == "create" {\n\t\treturn nil\n\t}\n'),
 ("S4-F1-bypass-total","F1 fix widened to every operation (total bypass)",
  '\tif operation == "manifest" || operation == "probe" {\n\t\treturn nil\n\t}\n\tif !admitted.HasOperation(operation) {\n\t\treturn &Error{Code: CodeCapabilityUnproven, Detail: "operation capability dependency"}\n\t}\n',
  '\treturn nil\n\tif !admitted.HasOperation(operation) {\n\t\treturn &Error{Code: CodeCapabilityUnproven, Detail: "operation capability dependency"}\n\t}\n'),
 ("S5-F1-divergence","CapabilitiesForOperation made to disagree: manifest gains a dependency",
  '\tif operation == "manifest" || operation == "probe" {\n\t\treturn []string{}, nil\n\t}\n',
  '\tif operation == "probe" {\n\t\treturn []string{}, nil\n\t}\n'),
 ("S6-static-without-manifest","checkClaimRelation static-claim-without-manifest arm deleted",
  '\t\tcase proved.Origin == OriginStatic && !exists:\n\t\t\treturn mismatchf("probe static claim without manifest")\n',
  '\t\tcase proved.Origin == OriginStatic && !exists:\n\t\t\t_ = static\n'),
 ("S7-evidence-platform","ParseEvidence platform-vocabulary gate narrowed",
  '\tplatform, err := scalar.ParsePlatform(platformName)\n\tif err != nil {\n\t\treturn Evidence{}, mismatchf("evidence platform")\n\t}\n',
  '\tplatform, _ := scalar.ParsePlatform(platformName)\n'),
 ("S8-claim-shape","parseClaim non-object guard narrowed",
  '\tobject, ok := raw.(map[string]any)\n\tif !ok {\n\t\treturn Claim{}, mismatchf("claim shape")\n\t}\n',
  '\tobject, _ := raw.(map[string]any)\n'),
 ("S9-admitted-has-any","Admitted.Has always true",
  None, None),
]
results=[]
for mid,desc,old,new in M:
    if old is None: continue
    n=ORIG.count(old)
    if n!=1:
        print(mid,"ANCHOR",n,flush=True); results.append({"id":mid,"desc":desc,"status":"ANCHOR-%d"%n}); continue
    open(SRC,'w').write(ORIG.replace(old,new,1))
    v=subprocess.run(['go','vet','./internal/terminalbackend/'],capture_output=True,text=True)
    if v.returncode!=0:
        open(SRC,'w').write(ORIG); print(mid,"COMPILE-FAIL",v.stderr.strip().splitlines()[-1:],flush=True)
        results.append({"id":mid,"desc":desc,"status":"COMPILE-FAIL","err":v.stderr[-300:]}); continue
    r=subprocess.run(['go','test','./internal/terminalbackend/','-count=1','-v'],capture_output=True,text=True)
    open(SRC,'w').write(ORIG)
    lines=r.stdout.splitlines()
    fails=[m.group(1) for m in (re.match(r'\s*--- FAIL: (\S+)', l) for l in lines) if m]
    msgs=[l.strip() for l in lines if '_test.go:' in l and ('error =' in l or 'refusal =' in l or 'want' in l)]
    st="KILLED" if r.returncode!=0 else "SURVIVED"
    results.append({"id":mid,"desc":desc,"status":st,"failing_tests":fails[:10],"messages":msgs[:6]})
    print(mid,st,"|",", ".join(fails[:6]),flush=True)
    for m in msgs[:4]: print("     ",m,flush=True)
assert hashlib.sha256(open(SRC,'rb').read()).hexdigest()==BASE,"REVERT FAILED"
print("\nreverted byte-identical")
json.dump(results,open('.temp/TASK-260830-1zvmw7/round2/supp.json','w'),indent=1)
