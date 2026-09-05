import subprocess, hashlib, json, sys, re, os
os.chdir('/Users/iv/Developer/ReluxWorks/agent-session-manager/.temp/STORY-260830-3m2mw8/worktree')
SRC='internal/terminalbackend/manifest.go'
ORIG=open(SRC).read()
BASE=hashlib.sha256(ORIG.encode()).hexdigest()

M=[
 ("G1-omission","checkClaimRelation omission loop",
  '\tfor capability := range manifest {\n\t\tif _, present := indexProbeCapability(probe, capability); !present {\n\t\t\treturn mismatchf("probe omission of manifest claim")\n\t\t}\n\t}\n',''),
 ("G2-static-echo","probe static claim echo equality",
  '\t\tcase proved.Origin == OriginStatic:\n\t\t\tif !equalClaim(proved, static) {\n\t\t\t\treturn mismatchf("probe static claim echo")\n\t\t\t}\n','\t\tcase proved.Origin == OriginStatic:\n'),
 ("G3-protocol-membership","checkProbeMembership protocol half",
  '\tif !member {\n\t\treturn mismatchf("probe protocol membership")\n\t}\n','\t_ = member\n'),
 ("G4-evidence-false-claim","checkEvidenceSet !claim.Value half",
  '\t\tif !present || !claim.Value {\n\t\t\treturn nil, mismatchf("evidence claim binding")\n\t\t}\n','\t\t_ = claim\n\t\tif !present {\n\t\t\treturn nil, mismatchf("evidence claim binding")\n\t\t}\n'),
 ("F3-record-digest","checkManifestRecordBinding executable digest (AdmitProbe)",
  '\tif manifest.ExecutableDigest != record.ExecutableDigest {\n\t\treturn &Error{Code: CodeUntrusted, BackendID: manifest.TerminalBackendID, Detail: "executable substitution"}\n\t}\n',''),
 ("F3b-probe-digest","checkProbeIdentity executable digest (probe<->manifest)",
  '\tif probe.ExecutableDigest != manifest.ExecutableDigest {\n\t\treturn &Error{Code: CodeUntrusted, BackendID: probe.TerminalBackendID, Detail: "executable substitution"}\n\t}\n',''),
 ("F4-expiry-boundary","checkEvidenceLiveness upper bound narrowing",
  '\tif observed.After(now) || !now.Before(expires) {','\tif observed.After(now) || now.After(expires) {'),
 ("F5-M01","ParseProbe schema URN",
  '\tif schema, err := stringMember(object, "schema"); err != nil || schema != SchemaProbe {\n\t\treturn Probe{}, mismatchf("probe schema")\n\t}\n',''),
 ("F5-M02","ParseProbe schema_version",
  '\tif version, err := stringMember(object, "schema_version"); err != nil || version != SchemaVersion100 {\n\t\treturn Probe{}, mismatchf("probe schema version")\n\t}\n',''),
 ("F5-M03","ParseProbe ParseID(backendID)",
  '\tif _, err := ParseID(backendID); err != nil {\n\t\treturn Probe{}, mismatchf("probe backend identity")\n\t}\n',''),
 ("F5-M04","ParseProbe implementation_kind",
  '\tkind, err := parseKind(kindName)\n\tif err != nil {\n\t\treturn Probe{}, mismatchf("probe implementation kind")\n\t}\n','\tkind, _ := parseKind(kindName)\n'),
 ("F5-M05","ParseProbe digest<->kind consistency",
  '\tif needsDigest(kind) == (digest == "") {\n\t\treturn Probe{}, mismatchf("probe executable digest")\n\t}\n',''),
 ("F5-M06","ParseEvidence schema URN",
  '\tif schema, err := stringMember(object, "schema"); err != nil || schema != SchemaCapabilityEvidence {\n\t\treturn Evidence{}, mismatchf("evidence schema")\n\t}\n',''),
 ("F5-M07","ParseEvidence schema_version",
  '\tif version, err := stringMember(object, "schema_version"); err != nil || version != SchemaVersion100 {\n\t\treturn Evidence{}, mismatchf("evidence schema version")\n\t}\n',''),
 ("F5-M08","ParseEvidence ParseID(backendID)",
  '\tif _, err := ParseID(backendID); err != nil {\n\t\treturn Evidence{}, mismatchf("evidence backend identity")\n\t}\n',''),
 ("F5-M09","ParseEvidence protocol major 1",
  '\tif semverMajor(protocolVersion) != 1 {\n\t\treturn Evidence{}, mismatchf("evidence protocol major 1")\n\t}\n',''),
 ("F5-M10","ParseManifest implementation_kind",
  '\tkind, err := parseKind(kindName)\n\tif err != nil {\n\t\treturn Manifest{}, mismatchf("manifest implementation kind")\n\t}\n','\tkind, _ := parseKind(kindName)\n'),
("H1-static-claim-without-manifest","checkClaimRelation fifth arm: static claim absent from manifest",
  '\t\tcase proved.Origin == OriginStatic && !exists:\n\t\t\treturn mismatchf("probe static claim without manifest")\n',
  '\t\tcase proved.Origin == OriginStatic && !exists:\n\t\t\t_ = static\n'),
 ("H2-evidence-platform","ParseEvidence platform vocabulary",
  '\tplatform, err := scalar.ParsePlatform(platformName)\n\tif err != nil {\n\t\treturn Evidence{}, mismatchf("evidence platform")\n\t}\n',
  '\tplatform, _ := scalar.ParsePlatform(platformName)\n'),
 ("H3-claim-shape","parseClaim non-object element guard",
  '\tobject, ok := raw.(map[string]any)\n\tif !ok {\n\t\treturn Claim{}, mismatchf("claim shape")\n\t}\n',
  '\tobject, _ := raw.(map[string]any)\n'),
]

results=[]
for mid,desc,old,new in M:
    n=ORIG.count(old)
    if n!=1:
        results.append({"id":mid,"desc":desc,"status":"ANCHOR-%d"%n}); print(mid,"ANCHOR",n,flush=True); continue
    open(SRC,'w').write(ORIG.replace(old,new,1))
    v=subprocess.run(['go','vet','./internal/terminalbackend/'],capture_output=True,text=True)
    if v.returncode!=0:
        open(SRC,'w').write(ORIG)
        results.append({"id":mid,"desc":desc,"status":"COMPILE-FAIL","out":v.stderr[:400]}); print(mid,"COMPILE-FAIL",flush=True); continue
    r=subprocess.run(['go','test','./internal/terminalbackend/','-count=1','-v'],capture_output=True,text=True)
    open(SRC,'w').write(ORIG)
    killed = r.returncode!=0
    lines=r.stdout.splitlines()
    fails=[m.group(1) for m in (re.match(r'\s*--- FAIL: (\S+)', l) for l in lines) if m]
    msgs=[l.strip() for l in lines if '_test.go:' in l and ('error = ' in l or 'refusal = ' in l)]
    results.append({"id":mid,"desc":desc,"status":"KILLED" if killed else "SURVIVED",
                    "failing_tests":fails[:12],"messages":msgs[:12]})
    print(mid, "KILLED" if killed else "SURVIVED", "| tests:", ", ".join(fails[:6]),flush=True)
    for m in msgs[:6]: print("      ", m,flush=True)

assert hashlib.sha256(open(SRC,'rb').read()).hexdigest()==BASE, "REVERT FAILED"
print("\nreverted byte-identical to", BASE)
json.dump(results, open('.temp/TASK-260830-1zvmw7/round3/armshift3.json','w'), indent=1)
