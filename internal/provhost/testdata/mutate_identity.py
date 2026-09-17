#!/usr/bin/env python3
"""Run isolated behavioral mutants for provider-identity creation; never modify the managed Story checkout.
Usage: PYTHONDONTWRITEBYTECODE=1 python3 internal/provhost/testdata/mutate_identity.py /absolute/evidence/directory
Every subprocess runs directly, logs its real exit, and has a bounded timeout.
The battery proves each gate by narrowing, never by deletion: every N-
mutant keeps its gate present and weakens it to admit exactly one member
of the class the gate must reject, and a named behavioral test must fail.
Text-inspecting gates carry a token-preserving mutant, and the harness
always executes the behavioral suites, never only a static checker.
"""
import hashlib
import json
from pathlib import Path
import re
import shutil
import subprocess
import sys

root = Path(__file__).resolve().parents[3]
output = Path(sys.argv[1]).resolve()
output.mkdir(parents=True, exist_ok=True)
copy = output / 'mutation-source'
if copy.exists():
    raise SystemExit('refusing to overwrite an existing mutation source')
copy.mkdir()
shutil.copytree(root / 'internal', copy / 'internal')
for name in ('go.mod', 'go.sum', 'README.md', 'LOGBOOK.md'):
    shutil.copy2(root / name, copy / name)
create_source = copy / 'internal/provhost/identity_create.go'
create_base = create_source.read_text()
bind_source = copy / 'internal/provhost/identity_bind.go'
bind_base = bind_source.read_text()
results = []

REALM_A = 'sha256:' + 'a' * 63 + 'd'
REALM_B = 'sha256:' + 'b' * 64

# Keep each mutant bounded to the smallest named behavioral suite that can
# observe its gate. This is still the same subprocess instrument and every
# selected test drives a production entry point; the control rows run the
# complete package. A missing mapping is a harness error, never an
# accidental empty test selection.
TEST_PATTERNS = {
    'N-create-native-empty': r'^TestCreateIdentityRefusals$',
    'N-create-opaque-abs': r'^TestCreateIdentityRefusals$',
    'N-create-realm-required': r'^TestCreateIdentityRefusals$',
    'N-create-backstop': r'^TestCreateIdentityRefusals$',
    'N-create-kind': r'^TestCreateIdentityRefusals$',
    'N-create-session': r'^TestCreateIdentityRefusals$',
    'N-discovery-root': r'^TestDecodeNativeDiscoveryRefusals$',
    'N-discovery-platform': r'^TestDecodeNativeDiscoveryRefusals$',
    'N-store-unknown': r'^TestStoreRootForRefusals$',
    'N-store-home': r'^TestStoreRootForRefusals$',
    'N-store-qwen': r'^TestStoreRootForRefusals$',
    'N-resume-qwen': r'^TestCheckResumeTupleRefusals$',
    'N-resume-unknown': r'^TestCheckResumeTupleRefusals$',
    'N-resume-muse-version': r'^TestCheckResumeTupleRefusals$',
    'N-build-drift': r'^TestVerifyIdentityBuildRefusals$',
    'N-bind-native': r'^TestVerifyIdentityDiscoveryRefusals$',
    'N-bind-absent': r'^TestVerifyIdentityDiscoveryRefusals$',
    'N-bind-contains': r'^TestVerifyIdentityDiscoveryRefusals$',
    'N-bind-unresolved': r'^TestVerifyIdentityDiscoveryRefusals$',
    'N-bind-override-pi': r'^TestVerifyIdentityDiscoveryRefusals$',
    'N-bind-override-absolute': r'^TestVerifyIdentityDiscoveryRefusals$',
    'N-bind-root-missing': r'^TestVerifyIdentityDiscoveryRefusals$',
    'N-bind-realm-shape': r'^TestVerifyIdentityDiscoveryRefusals$',
    'N-bind-realm-mismatch': r'^TestVerifyIdentityDiscoveryRefusals$',
    'C-harmless-comment': None,
    'C-not-applied': None,
    'C-compile-failure': None,
}

def test_command(name):
    pattern = TEST_PATTERNS.get(name, 'MISSING')
    if pattern == 'MISSING':
        raise SystemExit(f'no behavioral test mapping for {name}')
    if pattern is None:
        return ['go', 'test', './internal/provhost', '-count=1', '-v']
    return ['go', 'test', './internal/provhost', '-run', pattern, '-count=1', '-v']

def run(name, path, old, new, narrow):
    original = path.read_text()
    if original.count(old) != 1:
        results.append(dict(mutant=name, classification='NOT_APPLIED', count=original.count(old), bound=narrow))
        print(name, 'NOT_APPLIED', original.count(old), flush=True)
        return
    path.write_text(original.replace(old, new))
    command = test_command(name)
    try:
        with (output / (name + '.log')).open('w') as log:
            process = subprocess.run(command, cwd=copy, stdout=log, stderr=subprocess.STDOUT, timeout=180)
        text = (output / (name + '.log')).read_text()
        failures = re.findall(r'^\s*--- FAIL: ([^ ]+)', text, re.M)
        ran_tests = re.findall(r'^=== RUN ', text, re.M)
        classification = 'COMPILE_OR_HARNESS_FAILURE' if not ran_tests else ('SURVIVED' if process.returncode == 0 else ('KILLED' if failures else 'COMPILE_OR_HARNESS_FAILURE'))
        results.append(dict(mutant=name, classification=classification, exit=process.returncode, failed_tests=failures, bound=narrow, command=command))
        print(name, classification, process.returncode, ', '.join(failures), flush=True)
    finally:
        path.write_text(original)
        assert hashlib.sha256(path.read_bytes()).digest() == hashlib.sha256(original.encode()).digest()

with (output / 'control-before.log').open('w') as log:
    before = subprocess.run(['go','test','./internal/provhost','-count=1'],cwd=copy,stdout=log,stderr=subprocess.STDOUT,timeout=300)
results.append(dict(mutant='control-before', classification='CONTROL', exit=before.returncode))
print('control-before', 'CONTROL', before.returncode, flush=True)
if before.returncode:
    raise SystemExit('baseline failed')
run('N-create-native-empty',create_source,'if runeLength(params.NativeSessionID) < 1 || runeLength(params.NativeSessionID) > 512 {','if runeLength(params.NativeSessionID) < 0 || runeLength(params.NativeSessionID) > 512 {','Admit exactly the empty native session ID; overlong IDs still refuse.')
run('N-create-opaque-abs',create_source,r'if strings.HasPrefix(value, "/") || strings.HasPrefix(value, `\\`) || windowsDrivePattern.MatchString(value) {',r'if strings.HasSuffix(value, "/") || strings.HasPrefix(value, `\\`) || windowsDrivePattern.MatchString(value) {','Preserve the searched-for "/" token while checking the wrong end; a leading-absolute opaque value is admitted while UNC and drive forms still refuse.')
run('N-create-realm-required',create_source,'if params.ProviderID == "antigravity" && params.IdentityKind == "backend_conversation_uuid" && params.BackendRealm == "" {','if params.ProviderID == "antigravity" && params.IdentityKind == "backend_conversation_uuid" && params.BackendRealm == "" && params.NativeSessionID != "11111111-2222-4333-8444-555555555555" {','Admit exactly the fixture native ID past the Antigravity null-realm arm; the owner backstop still refuses it with a degraded detail.')
run('N-create-backstop',create_source,'digest, _, err := canonicaljson.CalculateObjectIdentity(staged)\n\tif err != nil {','digest, _, err := canonicaljson.CalculateObjectIdentity(staged)\n\tif err != nil && params.SessionID != "0198f4c8-3e70-7a11-8a2b-1234567890ab" {','Admit exactly the fixture session past the owner backstop; the deep-extensions negative then succeeds with an empty record_id claim.')
run('N-create-kind',create_source,'if !isIdentityKind(params.IdentityKind) {','if !isIdentityKind(params.IdentityKind) && params.IdentityKind != "window_handle" {','Admit exactly window_handle past the kind arm; the owner backstop still refuses it with a degraded detail.')
run('N-create-session',create_source,'if !isUUIDv7(params.SessionID) {','if !isUUIDv7(params.SessionID) && params.SessionID != "not-a-uuid" {','Admit exactly not-a-uuid past the session arm; the owner backstop still refuses it with a degraded detail.')
run('N-discovery-root',bind_source,'if !ok || (!isNull && !isAbsoluteOn(root, scalar.Platform(platform))) {','if !ok || (!isNull && !isAbsoluteOn(root, scalar.Platform(platform)) && root != "sessions/admitted") {','Admit exactly the sessions/admitted relative root; every other relative root still refuses.')
run('N-discovery-platform',bind_source,'if !isProbePlatform(platform) {','if !isProbePlatform(platform) && platform != "plan9" {','Admit exactly the plan9 platform past the registry arm; the body then decodes and the refusal disappears.')
run('N-store-unknown',bind_source,'if !isResumeProvider(providerID) {','if !isResumeProvider(providerID) && providerID != "futuredesk" {','Admit exactly futuredesk past the store registry arm; it silently resolves the pi root.')
run('N-store-home',bind_source,'if !isBareAbsolute(home) {','if !isBareAbsolute(home) && home != "Users/iv" {','Admit exactly the Users/iv relative home; a relative store root is then resolved.')
run('N-store-qwen',bind_source,'if providerID == "qwen" {','if providerID == "qwen" && home != "/Users/iv" {','Admit exactly the fixture home past the qwen arm; the registry arm still refuses it with a degraded detail.')
run('N-resume-qwen',bind_source,'if tuple.ProviderID == "qwen" {','if tuple.ProviderID == "qwen" && tuple.Platform != "linux" {','Admit exactly the linux tuple past the qwen arm; the registry arm still refuses it with a degraded detail.')
run('N-resume-unknown',bind_source,'if !isResumeProvider(tuple.ProviderID) {','if !isResumeProvider(tuple.ProviderID) && tuple.ProviderID != "futuredesk" {','Admit exactly futuredesk past the resume registry arm; the tuple then passes the gate.')
run('N-resume-muse-version',bind_source,'if tuple.ProviderID == "muse" && tuple.Platform == "macos" && tuple.Architecture == "arm64" && tuple.ProviderVersion != "0.1.0" {','if tuple.ProviderID == "muse" && tuple.Platform == "macos" && tuple.Architecture == "arm64" && tuple.ProviderVersion != "0.1.0" && tuple.ProviderVersion != "0.2.1" {','Preserve the pinned 0.1.0 token while admitting exactly 0.2.1; every other unverified version still refuses.')
run('N-build-drift',bind_source,'if version != tuple.ProviderVersion {','if version != tuple.ProviderVersion && tuple.ProviderVersion != "0.148.0" {','Admit exactly the 0.148.0 drifted build; every other drifted version still refuses.')
run('N-bind-native',bind_source,'if proof.NativeSessionID != native {','if proof.NativeSessionID != native && proof.NativeSessionID != "22222222-2222-4333-8444-555555555555" {','Admit exactly the foreign fixture native ID; the proof then binds and the refusal disappears.')
run('N-bind-absent',bind_source,'\tif !proof.Discovered {','\tif !proof.Discovered && !proof.HasDiscoveryRoot {','Admit exactly an undiscovered proof that still carries a root; a rootless undiscovered proof still refuses.')
run('N-bind-contains',bind_source,'if !rootContains(expected, proof.DiscoveryRoot) {','if !rootContains(expected, proof.DiscoveryRoot) && proof.DiscoveryRoot != "/Users/iv/.codex/sessions-evil" {','Admit exactly the sibling-prefix root; every other foreign root still refuses.')
run('N-bind-unresolved',bind_source,'if !isNull && !proof.BackendResolved {','if !isNull && !proof.BackendResolved && realm != "' + REALM_A + '" {','Admit exactly the fixture realm past the unresolved arm; the proof then binds and the refusal disappears.')
run('N-bind-override-pi',bind_source,'if ctx.Build.ProviderID != "pi" {','if ctx.Build.ProviderID != "pi" && ctx.Build.ProviderID != "codex" {','Admit exactly codex past the pi-only arm; the root-containment arm still refuses it with a degraded detail.')
run('N-bind-override-absolute',bind_source,'if !isAbsoluteOn(ctx.StoreRootOverride, scalar.Platform(ctx.Build.Platform)) {','if !isAbsoluteOn(ctx.StoreRootOverride, scalar.Platform(ctx.Build.Platform)) && ctx.StoreRootOverride != "custom/relative" {','Admit exactly the custom/relative override past the absolute arm; the root-containment arm still refuses it with a degraded detail.')
run('N-bind-root-missing',bind_source,'\tif !proof.HasDiscoveryRoot {','\tif !proof.HasDiscoveryRoot && !proof.Discovered {','Admit exactly a discovered proof with a null root; the root-containment arm still refuses it with a degraded detail.')
run('N-bind-realm-shape',bind_source,'if !isDigest(wantRealm) {','if !isDigest(wantRealm) && wantRealm != "nope" {','Admit exactly the nope realm past the shape arm; the realm-equality arm still refuses it with a degraded detail.')
run('N-bind-realm-mismatch',bind_source,'if isNull || realm != wantRealm {','if isNull || realm != wantRealm && wantRealm != "' + REALM_B + '" {','Admit exactly the drifted fixture realm past the equality arm; the unresolved arm still refuses it with a degraded detail.')
run('C-harmless-comment',bind_source,'// resolveBindRoot selects the expected store root: the caller override','// resolveBindRoot selects the expected store root: the caller override\n// classifier control: comment only, no behavior change.','Applied control: a behavior-preserving plant survives through the same instrument.')
run('C-not-applied',bind_source,'TOKEN_THAT_IS_NOT_PRESENT','replacement','Application control: missing patch is not a killed mutant.')
run('C-compile-failure',bind_source,'package provhost','package provhost\nthis is invalid Go','Compilation control: no named failed behavioral test is not a kill.')
with (output / 'control-after.log').open('w') as log:
    after = subprocess.run(['go','test','./internal/provhost','-count=1'],cwd=copy,stdout=log,stderr=subprocess.STDOUT,timeout=300)
results.append(dict(mutant='control-after',classification='CONTROL', exit=after.returncode))
print('control-after', 'CONTROL', after.returncode, flush=True)
assert create_source.read_text() == create_base and bind_source.read_text() == bind_base
(output / 'mutants.json').write_text(json.dumps(results,indent=2)+'\n')
# Controls are reported separately from the applied behavioral denominator.
bad = [row for row in results if row['mutant'].startswith('N-') and row['classification'] != 'KILLED']
raise SystemExit(1 if bad or after.returncode else 0)
