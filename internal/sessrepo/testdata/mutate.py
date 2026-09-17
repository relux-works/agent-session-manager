#!/usr/bin/env python3
"""Run isolated behavioral mutants; never modify the managed Story checkout.
Usage: python3 internal/sessrepo/testdata/mutate.py /absolute/evidence/directory
Every subprocess runs directly, logs its real exit, and has a bounded timeout.
The battery proves each lease-store gate by narrowing, never by deletion:
every N-mutant keeps its gate present and weakens it to admit exactly one
member of the class the gate must reject, and a named behavioral test must
fail. The T-mutant preserves the searched-for timestamp token while changing
behavior, and the harness always executes the behavioral suites.
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
source = copy / 'internal/sessrepo/lease_store.go'
base = source.read_text()
results = []

FOREIGN = 'sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff'
HOSTA = '0198f4c8-4a10-7b22-8b3c-1234567890ab'
LEASEA = 'aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee'
LEASEC = 'cccccccc-dddd-4eee-8fff-111111111111'

# Keep each mutant bounded to the smallest named behavioral suite that can
# observe its gate. This is still the same subprocess instrument and every
# selected test drives a production entry point; the control rows below run
# the complete package. A missing mapping is a harness error, never an
# accidental empty test selection.
TEST_PATTERNS = {
    'N-create-token': r'^TestCreateLeaseRefusesInvalidIdentity$',
    'N-create-holder': r'^TestCreateLeaseRefusesInvalidIdentity$',
    'N-create-issuer': r'^TestCreateLeaseRefusesInvalidIdentity$',
    'N-create-time': r'^TestCreateLeaseRefusesInvalidIdentity$',
    'T-create-time-pad': r'^TestCreateLeaseRefusesInvalidIdentity$',
    'N-successor-reason': r'^TestCompareAndSwapRefusesInvalidSuccessor$',
    'N-successor-checkpoint': r'^TestCompareAndSwapRefusesInvalidSuccessor$',
    'N-create-once': r'^TestCreateLeaseRefusesSecondCreateWithDifferingBytes$',
    'N-cas-token-reuse': r'^TestCompareAndSwapRefusesFencingTokenReuse$',
    'N-cas-conflict': r'^TestCompareAndSwapRefusesUnknownExpectation$',
    'N-cas-stale': r'^TestCompareAndSwapRefusesStaleExpectation$',
    'N-cas-empty': r'^TestCompareAndSwapRequiresExistingLease$',
    'N-fence-token-epoch': r'^TestVerifyFencingTokenRefusals$',
    'N-fence-token-lease': r'^TestVerifyFencingTokenRefusals$',
    'N-fence-token-holder': r'^TestVerifyFencingTokenRefusals$',
    'N-fence-stale': r'^TestVerifyFencingTokenRefusals$',
    'N-fence-future': r'^TestVerifyFencingTokenRefusals$',
    'N-fence-loser': r'^TestVerifyFencingTokenRefusals$',
    'N-fence-holder': r'^TestVerifyFencingTokenRefusals$',
    'N-expiry-policy': r'^TestCheckFencingExpiry$',
    'N-expiry-lapsed': r'^TestCheckFencingExpiry$',
    'N-load-funnel': r'^TestStoredLeaseCorruptionRefuses$',
    'N-get-funnel': r'^TestStoredLeaseCorruptionRefuses$|^TestGetLeaseRefusals$',
    'N-install-length': r'^TestInstallDisagreeingBytesRefuses$',
    'N-get-malformed': r'^TestGetLeaseRefusals$',
    'N-get-unknown': r'^TestGetLeaseRefusals$',
    'N-winning-empty': r'^TestWinningLeaseRefusesEmptyStore$',
    'C-harmless-comment': r'^TestCompareLeaseTupleOrdersByGreatestTuple$',
    'C-not-applied': r'^TestCompareLeaseTupleOrdersByGreatestTuple$',
    'C-compile-failure': r'^TestCompareLeaseTupleOrdersByGreatestTuple$',
}

def test_command(name):
    pattern = TEST_PATTERNS.get(name)
    if pattern is None:
        raise SystemExit(f'no behavioral test mapping for {name}')
    return ['go', 'test', './internal/sessrepo', '-run', pattern, '-count=1', '-v']

def run(name, old, new, narrow):
    original = source.read_text()
    if original.count(old) != 1:
        results.append(dict(mutant=name, classification='NOT_APPLIED', count=original.count(old), bound=narrow))
        print(name, 'NOT_APPLIED', original.count(old), flush=True)
        return
    source.write_text(original.replace(old, new))
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
        source.write_text(original)
        assert hashlib.sha256(source.read_bytes()).digest() == hashlib.sha256(original.encode()).digest()

with (output / 'control-before.log').open('w') as log:
    before = subprocess.run(['go','test','./internal/sessrepo','-count=1'],cwd=copy,stdout=log,stderr=subprocess.STDOUT,timeout=300)
results.append(dict(mutant='control-before', classification='CONTROL', exit=before.returncode))
if before.returncode:
    raise SystemExit('baseline failed')
run('N-create-token','if _, err := scalar.ParseUUIDv4(input.LeaseID); err != nil {','if _, err := scalar.ParseUUIDv4(input.LeaseID); err != nil && input.LeaseID != "not-a-uuid" {','Admit exactly the not-a-uuid fencing token; other malformed tokens still refuse.')
run('N-create-holder','if _, err := scalar.ParseUUIDv7(input.HolderHostID); err != nil {','if _, err := scalar.ParseUUIDv7(input.HolderHostID); err != nil && input.HolderHostID != "not-a-uuid" {','Admit exactly the not-a-uuid holder; other malformed holders still refuse.')
run('N-create-issuer','if _, err := scalar.ParseUUIDv7(input.IssuedByHostID); err != nil {','if _, err := scalar.ParseUUIDv7(input.IssuedByHostID); err != nil && input.IssuedByHostID != "0198f4c8-4a10-7b22-8b3c-1234567890aQ" {','Admit exactly the Q-suffixed issuer; other malformed issuers still refuse.')
run('N-create-time','if _, err := scalar.ParseTimestamp(input.CreatedAt); err != nil {','if _, err := scalar.ParseTimestamp(input.CreatedAt); err != nil && input.CreatedAt != "not-a-timestamp" {','Admit exactly the not-a-timestamp value; padded and other malformed timestamps still refuse.')
run('T-create-time-pad','if _, err := scalar.ParseTimestamp(input.CreatedAt); err != nil {','if _, err := scalar.ParseTimestamp(strings.TrimSpace(input.CreatedAt)); err != nil {','Parse the same timestamp token whitespace-tolerantly; padded timestamps admit while unparseable text still refuses.')
run('N-successor-reason','case "create", "graceful_takeover", "force_takeover", "recovery":','case "create", "graceful_takeover", "force_takeover", "recovery", "bogus_reason":','Admit exactly the bogus_reason value past the enum; every other non-enum reason still refuses.')
run('N-successor-checkpoint','if _, err := scalar.ParseDigest(input.CheckpointID); err != nil {','if _, err := scalar.ParseDigest(input.CheckpointID); err != nil && input.CheckpointID != "not-a-digest" {','Admit exactly the not-a-digest checkpoint; absent and other malformed checkpoints still refuse.')
run('N-create-once','\tif identical, ref := findIdenticalLease(stored, digest); identical {\n\t\treturn ref, nil\n\t}\n\tif len(stored) > 0 {','\tif identical, ref := findIdenticalLease(stored, digest); identical {\n\t\treturn ref, nil\n\t}\n\tif len(stored) > 1 {','Admit exactly a second create at depth one; a third create past a succession still refuses.')
run('N-cas-token-reuse','if known.LeaseID == input.LeaseID {','if known.LeaseID == input.LeaseID && known.LeaseID != "' + LEASEA + '" {','Admit exactly the epoch-1 token reuse; every other bound token still refuses.')
run('N-cas-conflict','\tbasis, known := findExpectedLease(stored, expected.RecordID)\n\tif !known {','\tbasis, known := findExpectedLease(stored, expected.RecordID)\n\tif !known && expected.RecordID != "' + FOREIGN + '" {','Admit exactly the foreign-digest expectation; malformed and empty expectations still conflict.')
run('N-cas-stale','if basis.RecordID != head.RecordID {','if basis.RecordID != head.RecordID && basis.Epoch != 1 {','Admit exactly epoch-1 stale bases; epoch-2 stale bases still refuse.')
run('N-cas-empty','\tif err != nil {\n\t\treturn LeaseRef{}, err\n\t}\n\tif len(stored) == 0 {','\tif err != nil {\n\t\treturn LeaseRef{}, err\n\t}\n\tif len(stored) == 0 && expected.RecordID != "" {','Admit exactly the empty-expectation CAS on an empty store; a named expectation on an empty store still refuses.')
run('N-fence-token-epoch','if token.Epoch == 0 {','if token.Epoch == 0 && (token.LeaseID != "bbbbbbbb-cccc-4ddd-8eee-ffffffffffff" || token.HolderHostID != "0198f4c8-7d40-7e55-8e6f-1234567890ab") {','Admit exactly the suite epoch-zero vector; the refusal degrades to the stale class.')
run('N-fence-token-lease','if _, err := scalar.ParseUUIDv4(token.LeaseID); err != nil {','if _, err := scalar.ParseUUIDv4(token.LeaseID); err != nil && token.LeaseID != "zzz" {','Admit exactly the zzz token lease; the refusal degrades to the conflict class.')
run('N-fence-token-holder','if _, err := scalar.ParseUUIDv7(token.HolderHostID); err != nil {','if _, err := scalar.ParseUUIDv7(token.HolderHostID); err != nil && token.HolderHostID != "zzz" {','Admit exactly the zzz token holder; the refusal degrades to the mismatch class.')
run('N-fence-stale','if token.Epoch < winner.Epoch {','if token.Epoch < winner.Epoch && !(token.Epoch == 1 && token.LeaseID == "' + LEASEA + '") {','Admit exactly the epoch-1 token; the refusal degrades to the conflict class.')
run('N-fence-future','if token.Epoch > winner.Epoch {','if token.Epoch > winner.Epoch && token.Epoch != 9 {','Admit exactly the epoch-9 token; the empty-store epoch-1 token still refuses unknown.')
run('N-fence-loser','if token.LeaseID != winner.LeaseID {','if token.LeaseID != winner.LeaseID && token.LeaseID != "' + LEASEC + '" {','Admit exactly the leaseC loser past the tie break; the token verifies and the negative fails by success.')
run('N-fence-holder','if token.HolderHostID != winner.HolderHostID {','if token.HolderHostID != winner.HolderHostID && token.HolderHostID != "' + HOSTA + '" {','Admit exactly the hostA non-holder; the token verifies and the negative fails by success.')
run('N-expiry-policy','if policy.RefreshInterval <= 0 {','if policy.RefreshInterval < 0 {','Admit exactly the zero interval; a negative interval still refuses.')
run('N-expiry-lapsed','if now.Sub(grant.ValidatedAt) > policy.RefreshInterval {','if now.Sub(grant.ValidatedAt) > policy.RefreshInterval && grant.SessionID != "0198f4c8-3e70-7a11-8a2b-1234567890ab" {','Admit exactly the suite lapsed grant; the grant verifies and the negative fails by success.')
run('N-load-funnel','\tstored, err := scanLeaseBlobs(directory, sessionID)\n\tif err != nil {','\tstored, err := scanLeaseBlobs(directory, sessionID)\n\tif err != nil && !strings.Contains(err.Error(), "garbage.json") {','Admit exactly the garbage-name torn vector; truncated, cross-session, misnamed, and undecodable vectors still refuse.')
run('N-get-funnel','if err := verifyLeaseBlob(view.record.sessionID, recordID, blob); err != nil {','if err := verifyLeaseBlob(view.record.sessionID, recordID, blob); err != nil && recordID != "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {','Admit exactly the undecodable-bytes vector; the misnamed vector still refuses.')
run('N-install-length','func leaseBytesEqual(left, right []byte) bool {\n\treturn bytes.Equal(left, right)\n}','func leaseBytesEqual(left, right []byte) bool {\n\treturn len(left) == len(right) && bytes.Equal(left, left)\n}','Compare length only at the ledgered equality site; the same-length garbage arm reuses while the different-length arm still refuses.')
run('N-get-malformed','if _, err := scalar.ParseDigest(recordID); err != nil {','if _, err := scalar.ParseDigest(recordID); err != nil && recordID != "not-a-digest" {','Admit exactly the not-a-digest record id; the refusal degrades to the unknown class.')
run('N-get-unknown','\tblob, err := os.ReadFile(filepath.Join(view.directory, "leases", blobFileName(recordID)))\n\tif err != nil {','\tblob, err := os.ReadFile(filepath.Join(view.directory, "leases", blobFileName(recordID)))\n\tif err != nil && recordID != "' + FOREIGN + '" {','Admit exactly the foreign-digest read; the refusal degrades to the corruption class.')
run('N-winning-empty','\tif err != nil {\n\t\treturn LeaseSummary{}, err\n\t}\n\tif len(stored) == 0 {','\tif err != nil {\n\t\treturn LeaseSummary{}, err\n\t}\n\tif len(stored) == 0 && sessionID != "0198f4c8-3e70-7a11-8a2b-1234567890ab" {','Admit exactly the suite empty store with a zero summary; the negative fails by success.')
run('C-harmless-comment','// CompareLeaseTuple orders two stored lease summaries by the Section 5.3','// CompareLeaseTuple orders two stored lease summaries by the Section 5.3\n// classifier control: comment only, no behavior change.','Applied control: a behavior-preserving plant survives through the same instrument.')
run('C-not-applied','TOKEN_THAT_IS_NOT_PRESENT','replacement','Application control: missing patch is not a killed mutant.')
run('C-compile-failure','package sessrepo','package sessrepo\nthis is invalid Go','Compilation control: no named failed behavioral test is not a kill.')
with (output / 'control-after.log').open('w') as log:
    after = subprocess.run(['go','test','./internal/sessrepo','-count=1'],cwd=copy,stdout=log,stderr=subprocess.STDOUT,timeout=300)
results.append(dict(mutant='control-after',classification='CONTROL', exit=after.returncode))
assert source.read_text() == base
(output / 'mutants.json').write_text(json.dumps(results,indent=2)+'\n')
# Controls are reported separately from the applied behavioral denominator.
bad = [row for row in results if (row['mutant'].startswith('N-') or row['mutant'].startswith('T-')) and row['classification'] != 'KILLED']
raise SystemExit(1 if bad or after.returncode else 0)
