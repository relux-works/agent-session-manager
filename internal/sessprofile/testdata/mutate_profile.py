#!/usr/bin/env python3
"""Run isolated behavioral mutants for profile resolution; never modify the managed Story checkout.
Usage: PYTHONDONTWRITEBYTECODE=1 python3 internal/sessprofile/testdata/mutate_profile.py /absolute/evidence/directory [PLANT ...]
Every subprocess runs directly, logs its real exit, and has a bounded timeout.
The battery proves each gate by narrowing, never by deletion: every N-
mutant keeps its gate present and weakens it to admit exactly one member
of the class the gate must reject, and a named behavioral test must fail.
No gate here inspects source text; the Pi-version plant additionally
preserves the pinned token while admitting one neighboring version, and
the harness always executes the behavioral suites, never only a static
checker. With PLANT arguments only those plants run (plus no controls);
without them the full battery runs with controls.
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
for name in ('go.mod', 'go.sum'):
    shutil.copy2(root / name, copy / name)
profile_source = copy / 'internal/sessprofile/profile.go'
profile_base = profile_source.read_text()
mint_source = copy / 'internal/sessprofile/mint.go'
mint_base = mint_source.read_text()
setprofile_source = copy / 'internal/sessprofile/setprofile.go'
setprofile_base = setprofile_source.read_text()
pairs_source = copy / 'internal/sessprofile/pairs.go'
pairs_base = pairs_source.read_text()
resolve_source = copy / 'internal/provhost/profile_resolve.go'
resolve_base = resolve_source.read_text()
results = []
only = set(sys.argv[2:])

# Keep each mutant bounded to the smallest named behavioral suite that can
# observe its gate. This is still the same subprocess instrument and every
# selected test drives a production entry point; the control rows run both
# complete packages. A missing mapping is a harness error, never an
# accidental empty test selection.
TEST_PATTERNS = {
    'N-derive-stale': ('./internal/sessprofile', r'^TestDeriveRefusesLosingLeaseAndAmbiguity$'),
    'N-derive-divergent': ('./internal/sessprofile', r'^TestDeriveRefusesLosingLeaseAndAmbiguity$'),
    'N-derive-repeat': ('./internal/sessprofile', r'^TestDeriveRefusesLosingLeaseAndAmbiguity$'),
    'N-change-fromto': ('./internal/sessprofile', r'^TestDeriveRefusesMalformedChangePayload$'),
    'N-change-confirmed': ('./internal/sessprofile', r'^TestDeriveRefusesMalformedChangePayload$'),
    'N-mint-unconfirmed': ('./internal/sessprofile', r'^TestMintChangeEventRefusals$'),
    'N-mint-fromto': ('./internal/sessprofile', r'^TestMintChangeEventRefusals$'),
    'N-mint-to-enum': ('./internal/sessprofile', r'^TestMintChangeEventRefusals$'),
    'N-setprofile-stale': ('./internal/sessprofile', r'^TestSetProfileRefusals$'),
    'N-setprofile-divergent': ('./internal/sessprofile', r'^TestSetProfileRefusals$'),
    'N-setprofile-replay-instant': ('./internal/sessprofile', r'^TestSetProfileRefusesNoOpChange$'),
    'N-check-pair-profile': ('./internal/sessprofile', r'^TestBundlePairProjection$'),
    'N-check-pair-source': ('./internal/sessprofile', r'^TestBundlePairProjection$'),
    'N-fork-provenance': ('./internal/sessprofile', r'^TestForkProjectionAndCheck$'),
    'N-finalize-dormant': ('./internal/sessprofile', r'^TestFinalizePair$'),
    'N-mapping-norow': ('./internal/provhost', r'^TestResolveMappingRefusals$'),
    'N-mapping-pi': ('./internal/provhost', r'^TestResolveMappingRefusals$'),
    'N-argv-standard-alias': ('./internal/provhost', r'^TestProjectLaunchArgvRefusals$'),
    'N-argv-bound-128': ('./internal/provhost', r'^TestProjectLaunchArgvBoundsBothDirections$'),
    'N-argv-mismatch': ('./internal/provhost', r'^TestProjectLaunchArgvRefusals$'),
    'C-harmless-comment': None,
    'C-not-applied': None,
    'C-compile-failure': None,
}

def test_command(name):
    pattern = TEST_PATTERNS.get(name, 'MISSING')
    if pattern == 'MISSING':
        raise SystemExit(f'no behavioral test mapping for {name}')
    if pattern is None:
        return ['go', 'test', './internal/sessprofile', './internal/provhost', '-count=1', '-v']
    package, run = pattern
    return ['go', 'test', package, '-run', run, '-count=1', '-v']

def run(name, path, old, new, narrow):
    if only and name not in only:
        return
    original = path.read_text()
    if original.count(old) != 1:
        results.append(dict(mutant=name, classification='NOT_APPLIED', count=original.count(old), bound=narrow))
        print(name, 'NOT_APPLIED', original.count(old), flush=True)
        return
    path.write_text(original.replace(old, new))
    command = test_command(name)
    try:
        with (output / (name + '.log')).open('w') as log:
            process = subprocess.run(command, cwd=copy, stdout=log, stderr=subprocess.STDOUT, timeout=300)
        text = (output / (name + '.log')).read_text()
        failures = re.findall(r'^\s*--- FAIL: ([^ ]+)', text, re.M)
        ran_tests = re.findall(r'^=== RUN ', text, re.M)
        classification = 'COMPILE_OR_HARNESS_FAILURE' if not ran_tests else ('SURVIVED' if process.returncode == 0 else ('KILLED' if failures else 'COMPILE_OR_HARNESS_FAILURE'))
        results.append(dict(mutant=name, classification=classification, exit=process.returncode, failed_tests=failures, bound=narrow, command=command))
        print(name, classification, process.returncode, ', '.join(failures), flush=True)
    finally:
        path.write_text(original)
        assert hashlib.sha256(path.read_bytes()).digest() == hashlib.sha256(original.encode()).digest()

DIVERGENT_BELOW = 'aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee'
PI_NEXT = '0.74.0'

if not only:
    with (output / 'control-before.log').open('w') as log:
        before = subprocess.run(['go', 'test', './internal/sessprofile', './internal/provhost', '-count=1'], cwd=copy, stdout=log, stderr=subprocess.STDOUT, timeout=600)
    results.append(dict(mutant='control-before', classification='CONTROL', exit=before.returncode))
    print('control-before', 'CONTROL', before.returncode, flush=True)
    if before.returncode:
        raise SystemExit('baseline failed')
run('N-derive-stale', profile_source, 'if a.Epoch < b.Epoch {', 'if a.Epoch+1 < b.Epoch {', 'Admit exactly the head-minus-one stale epoch past the lease comparison; deeper stale epochs still refuse.')
run('N-derive-divergent', profile_source, 'if lease.Epoch == current.Epoch {', 'if lease.Epoch == current.Epoch && lease.LeaseID != "cccccccc-dddd-4eee-8fff-000000000000" {', 'Admit exactly the fixture divergent lease past the divergent arm; the sequence rule still refuses it with a degraded code.')
run('N-derive-repeat', profile_source, 'if event.Sequence != tailSeq+1 {', 'if event.Sequence != tailSeq+1 && event.Sequence != tailSeq {', 'Admit exactly the tail-sequence repeat; earlier repeats and skips still refuse.')
run('N-change-fromto', profile_source, 'if from == target {', 'if from == target && from != "standard" {', 'Admit exactly the standard-to-standard change; every other equal-ended change still refuses.')
run('N-change-confirmed', profile_source, 'if _, ok := payloadBool(event.Payload, "confirmed"); !ok {', 'if _, ok := payloadBool(event.Payload, "confirmed"); !ok && event.Payload["confirmed"] != "yes" {', 'Admit exactly confirmed="yes" past the boolean arm; other non-boolean confirmations still refuse.')
run('N-mint-unconfirmed', mint_source, 'if params.To == ProfileYOLO && !params.Confirmed {', 'if params.To == ProfileYOLO && !params.Confirmed && params.From != "standard" {', 'Admit exactly the standard-to-yolo unconfirmed change; the mint then succeeds and the refusal disappears.')
run('N-mint-fromto', mint_source, 'if params.From == params.To {', 'if params.From == params.To && params.From != "yolo" {', 'Admit exactly the yolo-to-yolo change past the mint arm; the owner backstop still refuses it with a degraded sentinel.')
run('N-mint-to-enum', mint_source, 'if params.To != ProfileStandard && params.To != ProfileYOLO {', 'if params.To != ProfileStandard && params.To != ProfileYOLO && params.To != "turbo" {', 'Admit exactly turbo past the mint enum arm; the owner backstop still refuses it with a degraded sentinel.')
run('N-setprofile-stale', setprofile_source, 'if acting.Epoch < head.Epoch {', 'if acting.Epoch+1 < head.Epoch {', 'Admit exactly the head-minus-one acting epoch past the stale arm; the divergent arm still refuses it with a degraded sentinel.')
run('N-setprofile-divergent', setprofile_source, 'if acting.Epoch < head.Epoch {', 'if acting.Epoch < head.Epoch || acting.LeaseID == "' + DIVERGENT_BELOW + '" {', 'Admit exactly the fixture below-head lease past the divergent arm; the stale arm still refuses it with a degraded sentinel.')
run('N-setprofile-replay-instant', setprofile_source, 'if committed.CreatedByHostID != request.CreatedByHostID || committed.CreatedAt != request.CreatedAt {', 'if committed.CreatedByHostID != request.CreatedByHostID {', 'Admit exactly the mismatched-instant no-op as a replay; the author arm still refuses foreign authors.')
run('N-check-pair-profile', pairs_source, 'if observed.Profile != want.Profile {', 'if observed.Profile != want.Profile && context != "bundle" {', 'Admit exactly the stale bundle profile; every other context still refuses a diverged value.')
run('N-check-pair-source', pairs_source, 'if observed.HasSource != want.HasSource {', 'if observed.HasSource && !want.HasSource {', 'Admit exactly the missing source; a spurious source still refuses.')
run('N-fork-provenance', pairs_source, 'if wantHasSource && provenance != wantSource {', 'if wantHasSource && provenance != wantSource && provenance != "" {', 'Admit exactly the empty provenance string; every other swapped provenance still refuses.')
run('N-finalize-dormant', pairs_source, '\t\treturn Pair{}, nil', '\t\treturn checkpoint, nil', 'Carry the checkpoint pair under dormant validation; the null-pair assertion then fails.')
run('N-mapping-norow', resolve_source, 'if _, ok := profileYOLOMapping[providerID]; !ok {', 'if _, ok := profileYOLOMapping[providerID]; !ok && providerID != "qwen" {', 'Admit exactly qwen past the row arm; the tuple gate still refuses it with a degraded code.')
run('N-mapping-pi', resolve_source, 'if providerID == "pi" && tuple.ProviderVersion != piPinnedVersion {', 'if providerID == "pi" && tuple.ProviderVersion != piPinnedVersion && tuple.ProviderVersion != "' + PI_NEXT + '" {', 'Preserve the pinned 0.73.1 token while admitting exactly 0.74.0; every other Pi version still refuses.')
run('N-argv-standard-alias', resolve_source, 'if tokens[element] {', 'if tokens[element] && element != "--yolo" {', 'Admit exactly the --yolo word past the refused-token gate; every other table flag is still refused.')
run('N-argv-bound-128', resolve_source, 'if len(projected) > 128 {', 'if len(projected) > 129 {', 'Admit exactly the 129-element vector; longer vectors still refuse.')
run('N-argv-mismatch', resolve_source, 'if resolved.ProviderID != providerID || resolved.Profile != profile {', 'if (resolved.ProviderID != providerID || resolved.Profile != profile) && providerID != "codex" {', 'Admit exactly the codex resolution mismatch; the projection then succeeds and the refusal disappears.')
run('C-harmless-comment', setprofile_source, '// sameEnvelope reports whether the committed change is the request', '// sameEnvelope reports whether the committed change is the request\n// classifier control: comment only, no behavior change.', 'Applied control: a behavior-preserving plant survives through the same instrument.')
run('C-not-applied', setprofile_source, 'TOKEN_THAT_IS_NOT_PRESENT', 'replacement', 'Application control: missing patch is not a killed mutant.')
run('C-compile-failure', setprofile_source, 'package sessprofile', 'package sessprofile\nthis is invalid Go', 'Compilation control: no named failed behavioral test is not a kill.')
if not only:
    with (output / 'control-after.log').open('w') as log:
        after = subprocess.run(['go', 'test', './internal/sessprofile', './internal/provhost', '-count=1'], cwd=copy, stdout=log, stderr=subprocess.STDOUT, timeout=600)
    results.append(dict(mutant='control-after', classification='CONTROL', exit=after.returncode))
    print('control-after', 'CONTROL', after.returncode, flush=True)
    assert profile_source.read_text() == profile_base and mint_source.read_text() == mint_base and setprofile_source.read_text() == setprofile_base and pairs_source.read_text() == pairs_base and resolve_source.read_text() == resolve_base
    (output / 'mutants.json').write_text(json.dumps(results, indent=2) + '\n')
    # Controls are reported separately from the applied behavioral denominator.
    bad = [row for row in results if row['mutant'].startswith('N-') and row['classification'] != 'KILLED']
    raise SystemExit(1 if bad or after.returncode else 0)
(output / 'mutants.json').write_text(json.dumps(results, indent=2) + '\n')
