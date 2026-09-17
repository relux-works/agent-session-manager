#!/usr/bin/env python3
"""Run isolated behavioral mutants; never modify the managed Story checkout.
Usage: python3 internal/sessquery/testdata/mutate.py /absolute/evidence/directory
Every subprocess runs directly, logs its real exit, and has a bounded timeout.
The battery proves each gate by narrowing, never by deletion: every N-
mutant keeps its gate present and weakens it to admit exactly one member
of the class the gate must reject, and a named behavioral test must fail.
Grammar mutants preserve the searched-for token while changing behavior,
and the harness always executes the behavioral suites.
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
source = copy / 'internal/sessquery/query.go'
base = source.read_text()
selector_source = copy / 'internal/sessquery/selector.go'
selector_base = selector_source.read_text()
revalidate_source = copy / 'internal/sessquery/revalidate.go'
revalidate_base = revalidate_source.read_text()
plan_source = copy / 'internal/sessquery/plan.go'
plan_base = plan_source.read_text()
summary_source = copy / 'internal/sessquery/summary.go'
summary_base = summary_source.read_text()
lease_source = copy / 'internal/sessquery/lease.go'
lease_base = lease_source.read_text()
census_source = copy / 'internal/sessquery/rev11_regression_test.go'
census_base = census_source.read_text()
repo_source = copy / 'internal/sessrepo/store.go'
repo_base = repo_source.read_text()
results = []

IDA = '0198f4c8-3e70-7a11-8a2b-1234567890ab'
IDB = '0198f4c8-3e70-7a11-8a2b-1234567890ac'
HOSTA = '0198f4c8-4a10-7b22-8b3c-1234567890ab'
HOSTB = '0198f4c8-4a10-7b22-8b3c-1234567890ac'
HOSTC = '0198f4c8-4a10-7b22-8b3c-1234567890ad'
LEASEB = 'bbbbbbbb-cccc-4ddd-8eee-ffffffffffff'
LEASECYCLEB = 'dddddddd-eeee-4fff-8000-222222222222'

# Keep each mutant bounded to the smallest named behavioral suite that can
# observe its gate. This is still the same subprocess instrument and every
# selected test drives a production entry point; the control rows below run
# the complete packages. A missing mapping is a harness error, never an
# accidental empty test selection.
TEST_PATTERNS = {
    'N-collision-pair': r'^TestResolveExactNamesAndASCIICollisions$',
    'N-case-variant': r'^TestResolveExactNamesAndASCIICollisions$',
    'N-parked-id': r'^TestResolveExcludesTombstonedButListsIt$|^TestParkedReadRecoveryAndMissingVsMalformed$',
    'N-tombstoned-id': r'^TestResolveExcludesTombstonedButListsIt$',
    'N-unlisted-peer': r'^TestResolveAllowlistAndReadFailures$|^TestResolvePeerOrderAndReplicatedIdentity$',
    'N-duplicate-peer': r'^TestResolveAllowlistAndReadFailures$',
    'N-read-failure-fallback': r'^TestResolveAllowlistAndReadFailures$',
    'N-projection-failure-fallback': r'^TestListStatusCheckpointAndProjectionFailure$',
    'N-repository-case-variant': r'^TestResolveLocalNameAndUUID$',
    'N-first-at-split': r'^TestParseSelectorLiteralGrammar$',
    'N-local-prefix': r'^TestParseSelectorLiteralGrammar$',
    'N-id-case': r'^TestParseSelectorLiteralGrammar$',
    'N-peer-alias-fold': r'^TestSelectorConfigurationValidation$|^TestResolveQualifiedSourcesSelectOneIndex$',
    'N-id-through-names': r'^TestResolveQualifiedNameBeforeUUIDInSource$|^TestResolveBareIdentityUnion$',
    'N-explicit-fallback': r'^TestResolveQualifiedSourceNeverFallsBack$',
    'N-unknown-alias-local': r'^TestResolveExplicitSourceRefusals$',
    'N-disallowed-admit': r'^TestResolveExplicitSourceRefusals$',
    'N-dup-alias': r'^TestSelectorConfigurationValidation$',
    'N-identity-diverge': r'^TestResolveBareIdentityUnion$',
    'N-agreement-name': r'^TestResolveBareIdentityUnion$',
    'N-rev-revocation': r'^TestRevalidateDetectsEachFactChange$',
    'N-rev-config': r'^TestRevalidateDetectsEachFactChange$',
    'N-rev-binding': r'^TestRevalidateDetectsEachFactChange$',
    'N-rev-index': r'^TestRevalidateDetectsEachFactChange$',
    'N-rev-record': r'^TestRevalidateDetectsEachFactChange$',
    'N-rev-lease': r'^TestRevalidateDetectsEachFactChange$',
    'N-rev-heads': r'^TestRevalidateDetectsEachFactChange$',
    'N-rev-absent-substitute': r'^TestRevalidateDetectsEachFactChange$',
    'N-union-record': r'^TestRevalidateUnionLeaseDivergenceRefusesStale$|^TestReviewerOtherSourceDivergenceMustRefuse$',
    'N-union-lease': r'^TestRevalidateUnionLeaseDivergenceRefusesStale$',
    'N-union-tombstone': r'^TestRevalidateUnionTombstoneRefusesStale$',
    'N-plan-bootstrap': r'^TestBuildPlanRecordOnlySessionRefusesBootstrap$',
    'N-plan-source-host': r'^TestBuildPlanLocalSourceRequiresKnownHost$',
    'N-summary-bootstrap': r'^TestSummaryBootstrapRefusals$|^TestReviewerBootstrapSummaryMustRefuse$',
    'N-observation-host': r'^TestAuthoritativeMissingHostMetadataRefuses$',
    'N-observation-workspace': r'^TestAuthoritativeMissingWorkspaceRefuses$',
    'N-observation-capabilities': r'^TestAuthoritativeMissingCapabilitiesRefuses$',
    'N-observation-process': r'^TestAuthoritativeMissingProcessRefusesStatus$',
    'N-host-bound64': r'^TestAuthoritativeHostNameBound64$',
    'N-lease-missing': r'^TestBuildPlanRecordOnlySessionRefusesBootstrap$|^TestSummaryBootstrapRefusals$|^TestReviewerPlanRequiresLeaseRecord$',
    'N-chain-self': r'^TestSelfPredecessorWithCheckpointMustRefuse$|^TestRev4SelfPredecessorMustRefuse$',
    'N-checkpoint-placeholder': r'^TestRev4MissingCheckpointMustRefuse$|^TestRev5AncestorCheckpointAuthority$',
    'N-ancestor-checkpoint': r'^TestRev5AncestorCheckpointAuthority$|^TestRev5AncestorCheckpointBinding$',
    'N-checkpoint-holder': r'^TestRev5CheckpointCreatorMustBeHolder$',
    'N-ancestor-holder': r'^TestRev5AncestorCheckpointBinding$',
    'N-checkpoint-persistence': r'^TestRev6CheckpointPersistenceMustMatchSession$|^TestRev6TaskBoardPersistenceControl$|^TestRev6PersistenceMismatchIsObservationUnavailable$',
    'N-checkpoint-heads': r'^TestRev7CheckpointHeadAuthority$',
    'N-profile-first-source': r'^TestRev8ProfileSourceRefusals$|^TestRev8CheckpointProfileAuthority$',
    'N-profile-newest': r'^TestRev8CheckpointProfileAuthority$|^TestRev8ProfileSourceRefusals$',
    'N-profile-value': r'^TestRev8CheckpointProfileAuthority$|^TestRev8ProfileSourceRefusals$',
    'N-profile-change-direction': r'^TestRev8CheckpointProfileAuthority$|^TestRev8ProfileChangedPositiveControl$',
    'N-profile-fork-value': r'^TestReview9ForkLocalProfile$|^TestRev10ForkLocalProfile$',
    'N-profile-resume-missing': r'^TestRev10ResumeReferencedCheckpoint$|^TestRev11ReferencedCheckpointAdmissionRefusalClass$',
    'N-profile-resume-newest': r'^TestRev10ResumeReferencedCheckpoint$|^TestRev11ReferencedCheckpointAdmissionRefusalClass$',
    'N-profile-first-value': r'^TestRev8CheckpointProfileAuthority$|^TestRev8ProfileSourceRefusals$|^TestRev10ResumeReferencedCheckpoint$',
    'N-referenced-creator': r'^TestRev11ReferencedCheckpointAdmissionRefusalClass$|^TestRev11ReferencedCheckpointAdmissionOldPlan$',
    'N-referenced-variant': r'^TestRev11ReferencedCheckpointAdmissionRefusalClass$|^TestRev11ReferencedCheckpointAdmissionOldPlan$',
    'N-referenced-temporal': r'^TestReview11ReferencedTemporalAuthority$|^TestRev12ReferencedCheckpointLaterLease$',
    'N-parked-union-refusal': r'^TestRevalidateUnionParkedCopyRefuses$',
    'N-capability-name': r'^TestRev4UnknownCapabilityMustRefuse$',
    'N-census-alias-value-normalization': r'^TestRev15AliasAwareRecordConsumptionCensus$',
    'B-peer-order': r'^TestResolvePeerOrderAndReplicatedIdentity$',
    'B-session-order': r'^TestListStatusDerivedFactsAndStableOrder$',
    'B-tie-break': r'^TestResolvePeerOrderAndReplicatedIdentity$|^TestResolveBareIdentityUnion$',
    'C-harmless-comment': r'^TestRev14CapabilitySeal$',
    'C-not-applied': r'^TestRev14CapabilitySeal$',
    'C-compile-failure': r'^TestRev14CapabilitySeal$',
}

def test_command(name, path):
    pattern = TEST_PATTERNS.get(name)
    if pattern is None:
        raise SystemExit(f'no behavioral test mapping for {name}')
    packages = ['./internal/sessrepo'] if path == repo_source else ['./internal/sessquery']
    return ['go', 'test', *packages, '-run', pattern, '-count=1', '-v']

def run(name, path, old, new, narrow):
    original = path.read_text()
    if original.count(old) != 1:
        results.append(dict(mutant=name, classification='NOT_APPLIED', count=original.count(old), bound=narrow))
        return
    path.write_text(original.replace(old, new))
    command = test_command(name, path)
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

def run_census(name, old, new, narrow, pattern=r'^TestRev14RecordConsumptionCensusRejectsAlternatePaths$'):
    original = census_source.read_text()
    if original.count(old) != 1:
        results.append(dict(mutant=name, classification='NOT_APPLIED', count=original.count(old), bound=narrow))
        return
    census_source.write_text(original.replace(old, new))
    command = ['go', 'test', './internal/sessquery', '-run', pattern, '-count=1', '-v']
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
        census_source.write_text(original)
        assert hashlib.sha256(census_source.read_bytes()).digest() == hashlib.sha256(original.encode()).digest()

with (output / 'control-before.log').open('w') as log:
    before = subprocess.run(['go','test','./internal/sessquery','./internal/sessrepo','-count=1'],cwd=copy,stdout=log,stderr=subprocess.STDOUT,timeout=180)
results.append(dict(mutant='control-before', classification='CONTROL', exit=before.returncode))
if before.returncode:
    raise SystemExit('baseline failed')
run('N-collision-pair',source,'if len(ids) > 1 {','if len(ids) > 2 {','Allow exactly a two-identity folded bucket; larger buckets still refuse.')
run('N-case-variant',source,'if item.summary.Projection.Name == selector && exact == nil {','if (item.summary.Projection.Name == selector || selector == "az-09._") && exact == nil {','Allow the sole non-exact query az-09._ within its folded bucket.')
run('N-parked-id',source,'if row.Projection.Parked != nil || row.Projection.State == sessstate.StateTombstoned {','if (row.Projection.Parked != nil && row.Projection.SessionID != "0198f4c8-3e70-7a11-8a2b-1234567890ab") || row.Projection.State == sessstate.StateTombstoned {','Allow one parked session identity into live resolution.')
run('N-tombstoned-id',source,'if row.Projection.Parked != nil || row.Projection.State == sessstate.StateTombstoned {','if row.Projection.Parked != nil || (row.Projection.State == sessstate.StateTombstoned && row.Projection.SessionID != "0198f4c8-3e70-7a11-8a2b-1234567890ab") {','Allow one tombstoned session identity into live resolution.')
run('N-unlisted-peer',source,'ids := append([]string(nil), reader.AllowlistedPeerIDs...)\n\tsort.Strings(ids)\n\tvar out []candidate','ids := append([]string(nil), reader.AllowlistedPeerIDs...)\n\tids = append(ids, "0198f4c8-4a10-7b22-8b3c-1234567890ab")\n\tsort.Strings(ids)\n\tvar out []candidate','Read one peer outside the allowlist; all other peer identities still filtered.')
run('N-duplicate-peer',source,'if found {','if found && id != "0198f4c8-4a10-7b22-8b3c-1234567890ab" {','Permit duplicate sources for one peer identity, retain duplicate refusal for other identities.')
run('N-read-failure-fallback',source,'rows, err := repo.ListSessions()\n\tif err != nil {\n\t\treturn nil, err\n\t}', 'rows, err := repo.ListSessions()\n\tif err != nil {\n\t\tif errors.Is(err, sessrepo.ErrRepositoryPath) { return []Summary{}, nil }; return nil, err\n\t}', 'Treat repository-path read failure as empty, while retaining other read refusals.')
run('N-projection-failure-fallback',source,'projection, err := projector.Project(row.SessionID)\n\t\tif err != nil {\n\t\t\treturn nil, err\n\t\t}', 'projection, err := projector.Project(row.SessionID)\n\t\tif err != nil {\n\t\t\tif row.SessionID == "0198f4c8-3e70-7a11-8a2b-1234567890ab" { continue }; return nil, err\n\t\t}', 'Omit one identity on failed projection instead of propagating its failure.')
run('N-repository-case-variant',repo_source,'if len(folded) == 1 && folded[0].Name == query {','if len(folded) == 1 && (folded[0].Name == query || query == "Payments-API") {','Allow one non-exact query at the retained repository Resolve entry.')
run('N-first-at-split',selector_source,'key, source, qualified := strings.Cut(argument, "@")','key, source, qualified := argument, "", false\n\tif at := strings.LastIndexByte(argument, \'@\'); at >= 0 {\n\t\tkey, source, qualified = argument[:at], argument[at+1:], true\n\t}','Split at the last @ while keeping the @ token; an alias containing @ then misroutes.')
run('N-local-prefix',selector_source,'\tcase source == "local":','\tcase strings.HasPrefix(source, "local"):','Admit local-prefixed sources while keeping the local token; localX then selects local.')
run('N-id-case',selector_source,'\tif id, ok := strings.CutPrefix(key, "id:"); ok {\n\t\tif _, err := scalar.ParseUUIDv7(id); err != nil {','\tlowered := strings.ToLower(key)\n\tif id, ok := strings.CutPrefix(lowered, "id:"); ok {\n\t\tif _, err := scalar.ParseUUIDv7(id); err != nil {','Fold the id: key prefix while keeping its token; ID: then bypasses names.')
run('N-peer-alias-fold',selector_source,'if config.aliasOf[peer.HostID] == alias && alias != "" {','if asciiFold(config.aliasOf[peer.HostID]) == asciiFold(alias) && alias != "" {','Fold alias comparison while keeping exact mapping storage; WORK then selects Work.')
run('N-id-through-names',source,'\treturn selectIdentity(strings.TrimPrefix(parsed.Key, "id:"), live, config)','\tif match, found, err := matchName(live, strings.TrimPrefix(parsed.Key, "id:")); found || err != nil {\n\t\treturn match, err\n\t}\n\treturn selectIdentity(strings.TrimPrefix(parsed.Key, "id:"), live, config)','Route one explicit id: key through the name tier first; a UUID-shaped name then shadows the identity.')
run('N-explicit-fallback',source,'\treturn Selection{}, fmt.Errorf("%w: no live session named %q in source %s", ErrNotFound, parsed.Key, describeSource(parsed))','\treturn reader.resolveBare(Selector{Raw: parsed.Raw, Key: parsed.Key, UUIDShaped: parsed.UUIDShaped, Source: SourceBare}, config)','Fall back to the bare union on exactly an explicit-source miss; a locally present name then resolves.')
run('N-unknown-alias-local',source,'\t\t\treturn nil, "", fmt.Errorf("%w: no configured source for alias %q", ErrSourceNotFound, parsed.SourceValue)','\t\t\treturn reader.Local, "", nil','Treat exactly an unknown alias as the local source; a locally present name then resolves.')
run('N-disallowed-admit',source,'\t\t\treturn nil, "", fmt.Errorf("%w: peer %s is not allowlisted", ErrPeerNotAllowlisted, host)','\t\t\treturn reader.peerRepo(host)','Admit exactly one disallowed peer past the allowlist into its index read.')
run('N-dup-alias',selector_source,'\t\t\treturn config, fmt.Errorf("%w: duplicate source mapping for alias %q", ErrInvalidConfig, alias)','\t\t\tseenAliases[alias] = struct{}{}','Admit exactly duplicate alias mappings with last-write-wins instead of refusing them.')
run('N-identity-diverge',source,'\tfor _, item := range matches[1:] {\n\t\tif item.summary.Projection.RecordID != matches[0].summary.Projection.RecordID {','\tfor _, item := range matches[1:] {\n\t\tif item.summary.Projection.RecordID != matches[0].summary.Projection.RecordID && id.String() != "' + IDA + '" {','Admit exactly the idA cross-copy record disagreement in the identity tier; other identities still refuse integrity_failure.')
run('N-agreement-name',source,'\t\tif item.summary.Projection.RecordID != digest {','\t\tif item.summary.Projection.RecordID != digest && item.selection.SessionID != "' + IDA + '" {','Admit exactly the idA cross-copy record disagreement behind a name win; other identities still refuse integrity_failure.')
run('N-rev-revocation',revalidate_source,'\t\tif _, allowed := config.allowed[peerHost]; !allowed {','\t\tif _, allowed := config.allowed[peerHost]; !allowed && peerHost != "' + HOSTA + '" {','Admit exactly the revoked hostA peer past revocation; the refusal degrades to stale configuration.')
run('N-rev-config',revalidate_source,'\tif configuration != plan.ConfigurationDigest {','\tif configuration != plan.ConfigurationDigest && reader.Aliases["' + HOSTC + '"] != "ws" {','Admit exactly the ws-alias configuration for hostC; other configuration changes still refuse.')
run('N-rev-binding',revalidate_source,'\tif !bindingHolds(plan, config, reader) {','\tif !bindingHolds(plan, config, reader) && config.aliasFor("' + HOSTB + '") != "ws" {','Admit exactly the ws remap onto hostB; other binding changes still refuse.')
run('N-rev-index',revalidate_source,'\tif index != plan.SourceIndexDigest {','\tindexAdmitted := false\n\tfor _, row := range rows {\n\t\tif row.Projection.SessionID == "' + IDB + '" {\n\t\t\tindexAdmitted = true\n\t\t}\n\t}\n\tif index != plan.SourceIndexDigest && !indexAdmitted {','Admit exactly an index containing session idB; other index changes still refuse.')
run('N-rev-record',revalidate_source,'\tif recordID != plan.SessionRecordID {','\trecordParked := false\n\tfor _, row := range rows {\n\t\tif row.Projection.SessionID == plan.SessionID && row.Projection.Parked != nil {\n\t\t\trecordParked = true\n\t\t}\n\t}\n\tif recordID != plan.SessionRecordID && !recordParked {','Admit exactly the parked chain-mismatch record presented by the replacement fixture; the refusal degrades to the lease reason.')
run('N-rev-lease',revalidate_source,'\tif projection.Winner.Epoch != plan.LeaseEpoch || projection.Winner.LeaseID != plan.LeaseID {','\tif (projection.Winner.Epoch != plan.LeaseEpoch || projection.Winner.LeaseID != plan.LeaseID) && projection.Winner.LeaseID != "' + LEASEB + '" {','Admit exactly the leaseB successor at the plan source; the refusal degrades to the heads reason.')
run('N-rev-heads',revalidate_source,'\tif !equalStrings(heads, plan.AuthorityHeads) {','\tchain, _ := repo.ListEvents(plan.SessionID)\n\tif !equalStrings(heads, plan.AuthorityHeads) && len(chain) != 2 {','Admit exactly a two-event chain at the heads compare; the refusal degrades to the index reason.')
run('N-rev-absent-substitute',revalidate_source,'\tif absent {','\tif absent && len(rows) > 0 {\n\t\trecordID = rows[0].Projection.RecordID\n\t\tplan.SessionID = rows[0].Projection.SessionID\n\t}\n\tif absent && len(rows) == 0 {','Substitute the first listed sibling for exactly a deleted selection instead of reporting its absence.')
run('N-union-record',revalidate_source,'\t\tif row.Projection.State == sessstate.StateTombstoned {\n\t\t\treturn fmt.Errorf("%w: tombstone evidence changed", ErrPlanStale)\n\t\t}\n\t\tif row.Projection.RecordID != plan.SessionRecordID {','\t\tif row.Projection.State == sessstate.StateTombstoned {\n\t\t\treturn fmt.Errorf("%w: tombstone evidence changed", ErrPlanStale)\n\t\t}\n\t\tif row.Projection.RecordID != plan.SessionRecordID && row.Projection.SessionID != "' + IDB + '" {','Admit exactly the idB cross-source record disagreement in the union check; other contradictions still refuse.')
run('N-union-lease',revalidate_source,'\t\tif ordering > 0 {','\t\tif ordering > 0 && row.Projection.Winner.LeaseID != "' + LEASEB + '" {','Admit exactly the leaseB greater union successor; other greater tuples still refuse.')
run('N-union-tombstone',revalidate_source,'\t\t\treturn fmt.Errorf("%w: tombstone evidence changed", ErrPlanStale)','\t\tif row.Projection.SessionID != "' + IDB + '" {\n\t\t\treturn fmt.Errorf("%w: tombstone evidence changed", ErrPlanStale)\n\t\t}','Admit exactly the idB cross-source tombstone; other tombstone evidence still refuses.')
run('N-plan-bootstrap',plan_source,'\tif projection.Winner.IsZero() {\n\t\treturn SelectionPlan{}, fmt.Errorf("%w: complete authority read proves record %s without any valid lease", ErrBootstrapIncomplete, selected.SessionID)','\tif projection.Winner.IsZero() && selected.SessionID != "' + IDA + '" {\n\t\treturn SelectionPlan{}, fmt.Errorf("%w: complete authority read proves record %s without any valid lease", ErrBootstrapIncomplete, selected.SessionID)','Admit exactly record-only idA past the build bootstrap gate; the shape backstop still refuses it with invalid_arguments.')
run('N-plan-source-host',plan_source,'\t\tif config.localHost == "" {\n\t\t\treturn SelectionPlan{}, fmt.Errorf("%w: local host identity is required to bind a local plan source", ErrInvalidConfig)','\t\tif config.localHost == "" && selected.SessionID != "' + IDA + '" {\n\t\t\treturn SelectionPlan{}, fmt.Errorf("%w: local host identity is required to bind a local plan source", ErrInvalidConfig)','Admit exactly idA past the local-host gate; the shape backstop still refuses the empty source host with invalid_arguments.')
run('N-summary-bootstrap',summary_source,'\tif projection.Winner.IsZero() {\n\t\treturn fmt.Errorf("%w: complete authority read proves record %s without any valid lease", ErrBootstrapIncomplete, projection.SessionID)','\tif projection.Winner.IsZero() && projection.SessionID != "' + IDA + '" {\n\t\treturn fmt.Errorf("%w: complete authority read proves record %s without any valid lease", ErrBootstrapIncomplete, projection.SessionID)','Admit exactly record-only idA into list/status summaries; other interrupted prefixes still refuse.')
run('N-observation-host',summary_source,'\tmetadata, ok := reader.HostMetadata[lease.HolderHostID]\n\tif !ok {','\tmetadata, ok := reader.HostMetadata[lease.HolderHostID]\n\tif !ok && lease.HolderHostID != "' + HOSTA + '" {','Admit exactly owner hostA without host metadata; other missing owners still refuse observation_unavailable.')
run('N-observation-workspace',summary_source,'\tif !hasObservations || observations.WorkspaceStatus == nil {','\tif (!hasObservations || observations.WorkspaceStatus == nil) && projection.SessionID != "' + IDA + '" {','Admit exactly idA without workspace status; other missing workspaces still refuse observation_unavailable.')
run('N-observation-capabilities',summary_source,'\tif !hasObservations || observations.Capabilities == nil {','\tif (!hasObservations || observations.Capabilities == nil) && projection.SessionID != "' + IDA + '" {','Admit exactly idA without capabilities; other missing capabilities still refuse observation_unavailable.')
run('N-observation-process',summary_source,'\t\tif !hasObservations || observations.ProcessPresent == nil {','\t\tif (!hasObservations || observations.ProcessPresent == nil) && projection.SessionID != "' + IDA + '" {','Admit exactly idA without process liveness for status; other missing liveness still refuses observation_unavailable.')
run('N-host-bound64',summary_source,'\t\tif err := checkPrintableBound(metadata.DisplayName, 1, MaxDisplayNameRunes, "host display name"); err != nil {','\t\tif err := checkPrintableBound(metadata.DisplayName, 1, MaxDisplayNameRunes, "host display name"); err != nil && metadata.DisplayName != "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" {','Admit exactly the 65-character host name; other overlong names still refuse invalid_config.')
run('N-lease-missing',lease_source,'\tcandidates := bySession[sessionID]\n\tif len(candidates) == 0 {\n\t\treturn validatedLease{}, fmt.Errorf("%w: winning lease record for session %s cannot be established", ErrObservationUnavailable, sessionID)\n\t}','\tcandidates := bySession[sessionID]\n\tif len(candidates) == 0 && sessionID != "' + IDA + '" {\n\t\treturn validatedLease{}, fmt.Errorf("%w: winning lease record for session %s cannot be established", ErrObservationUnavailable, sessionID)\n\t}\n\tif len(candidates) == 0 {\n\t\treturn validatedLease{Digest: "sha256:0000000000000000000000000000000000000000000000000000000000000000", SessionID: sessionID, LeaseID: "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee", Epoch: 1, HolderHostID: "' + HOSTA + '"}, nil\n\t}','Admit exactly idA with no admitted lease record past the absence gate with a placeholder; the production test still refuses it.')
run('N-chain-self',lease_source,'\t\tparent, ok := byLease[current.Predecessor]\n\t\tif !ok {','\t\tparent, ok := byLease[current.Predecessor]\n\t\tif ok && parent.LeaseID == current.LeaseID {\n\t\t\treturn chain, parent, nil\n\t\t}\n\t\tif !ok {','Admit exactly the self-predecessor link while terminating the walk; skipped-epoch and cyclic links still refuse integrity_failure.')
run('N-checkpoint-placeholder',lease_source,'\tcheckpoint, ok := checkpoints[winner.Checkpoint]\n\tif !ok {\n\t\treturn fmt.Errorf("%w: validated checkpoint %s for session %s cannot be established", ErrObservationUnavailable, winner.Checkpoint, winner.SessionID)\n\t}\n','\tcheckpoint, ok := checkpoints[winner.Checkpoint]\n\tif !ok && winner.Checkpoint == "sha256:0000000000000000000000000000000000000000000000000000000000000000" {\n\t\treturn nil\n\t}\n\tif !ok {\n\t\treturn fmt.Errorf("%w: validated checkpoint %s for session %s cannot be established", ErrObservationUnavailable, winner.Checkpoint, winner.SessionID)\n\t}\n','Admit exactly the all-zero placeholder checkpoint reference; wrong-session and wrong-lease references still refuse observation_unavailable.')
run('N-ancestor-checkpoint',lease_source,'\tfor index := 1; index < len(chain); index++ {\n\t\tcurrent := chain[index]\n','\tfor index := 1; index < len(chain); index++ {\n\t\tcurrent := chain[index]\n\t\tif current.LeaseID == "' + LEASEB + '" {\n\t\t\tcontinue\n\t\t}\n','Admit exactly the leaseB ancestor past the winning-ancestry checkpoint gate; a missing, misbound, or wrong-creator ancestor checkpoint still refuses observation_unavailable.')
run('N-checkpoint-holder',lease_source,'\tif err := reader.checkWinnerCheckpoint(winner, predecessor, sessionKind, repo, chain); err != nil {\n\t\treturn validatedLease{}, err\n\t}\n','\tif err := reader.checkWinnerCheckpoint(winner, predecessor, sessionKind, repo, chain); err != nil && winner.SessionID != "' + IDA + '" {\n\t\treturn validatedLease{}, err\n\t}\n','Allow exactly the selected idA winner checkpoint admission error past the production boundary; the wrong-creator winner still reaches the summaries and fails this narrowing mutant.')
run('N-ancestor-holder',lease_source,'\t\tif err := checkCheckpointBinding(current, chain[index+1], checkpoints, sessionKind, repo, chain); err != nil {\n\t\t\treturn err\n\t\t}\n','\t\tif err := checkCheckpointBinding(current, chain[index+1], checkpoints, sessionKind, repo, chain); err != nil && current.LeaseID != "' + LEASEB + '" {\n\t\t\treturn err\n\t\t}\n','Allow exactly the leaseB necessary-ancestor checkpoint admission error past the production boundary; the invalid ancestor controls must fail this narrowing mutant.')
run('N-checkpoint-persistence',lease_source,'\tcase "direct":\n\t\tif !checkpoint.HasProviderManifest || checkpoint.HasTaskBoardBundle {','\tcase "direct":\n\t\tif (!checkpoint.HasProviderManifest || checkpoint.HasTaskBoardBundle) && sessionID != "' + IDA + '" {','Admit exactly the idA direct-session wrong persistence variant; task_board sessions and every other direct session still refuse observation_unavailable.')
run('N-checkpoint-heads',lease_source,'\tfor _, head := range checkpoint.EventHeads {\n\t\tsummary, ok := byID[head]\n\t\tif !ok {\n\t\t\treturn fmt.Errorf("%w: validated checkpoint %s names unknown event head %s for session %s", ErrObservationUnavailable, checkpoint.Digest, head, checkpoint.SessionID)\n\t\t}\n','\tfor _, head := range checkpoint.EventHeads {\n\t\tsummary, ok := byID[head]\n\t\tif !ok && head == "sha256:0000000000000000000000000000000000000000000000000000000000000000" {\n\t\t\tcontinue\n\t\t}\n\t\tif !ok {\n\t\t\treturn fmt.Errorf("%w: validated checkpoint %s names unknown event head %s for session %s", ErrObservationUnavailable, checkpoint.Digest, head, checkpoint.SessionID)\n\t\t}\n','Admit exactly the all-zero missing event head while preserving the searched unknown-head token; the missing-head negative must fail through the shared admission path.')
run('N-profile-first-source',lease_source,'if pair.hasSource != wantHasSource {','if pair.hasSource != wantHasSource && pair.source != "sha256:0000000000000000000000000000000000000000000000000000000000000000" {','Admit exactly the all-zero source where the creation pair wants null; other unexpected or missing sources still refuse integrity_failure.')
run('N-profile-newest',lease_source,'newestSource, newestProfile, hasNewest = summary.EventID, target, true','if !hasNewest {\n\t\t\tnewestSource, newestProfile, hasNewest = summary.EventID, target, true\n\t\t}','Keep the first change as authority instead of the newest; citations of a superseded source then admit while other source classes still refuse.')
run('N-profile-value',lease_source,'if pair.profile != wantProfile {','if pair.profile != wantProfile && pair.profile != "yolo" {','Admit exactly a yolo-valued pair past the effective-profile compare; every other stale value still refuses integrity_failure.')
run('N-profile-change-direction',lease_source,'target, err := leaseStringMember(payload, "to")\n\tif err != nil || (target != "standard" && target != "yolo") {','target, err := leaseStringMember(payload, "to")\n\ttarget, _ = leaseStringMember(payload, "from")\n\tif err != nil || (target != "standard" && target != "yolo") {','Derive the effective profile from the change source while still matching profile.changed and reading to; the P1/E1 positive controls must fail.')
run('N-profile-fork-value',lease_source,'if err := checkProfilePairSource(checkpoint, summary, pair, creation, "", false, byID, closure); err != nil {','if err := checkProfilePairSource(checkpoint, summary, pair, creation, "", false, byID, closure); err != nil && pair.profile != "standard" {','Admit exactly a standard-valued fork past the new-record compare; the fork negatives must fail by admission.')
run('N-profile-resume-missing',lease_source,'\tref, err := admission(checkpointID, authority.sessionID, summary)\n\tif err != nil {\n\t\treturn "", "", false, nil, err\n\t}','\tref, err := admission(checkpointID, authority.sessionID, summary)\n\tif err != nil {\n\t\treturn creation, "", false, map[string]bool{}, nil\n\t}','Admit exactly a missing-reference resume carrying the creation pair; the missing negative must fail by admission while other missing pairs still refuse.')
run('N-profile-resume-newest',lease_source,'refSource, refProfile, refHas = candidate.EventID, target, true','if !refHas {\n\t\trefSource, refProfile, refHas = candidate.EventID, target, true\n\t}','Keep the first referenced change as authority instead of the newest; the non-newest resume negative must fail by admission.')
run('N-profile-first-value',lease_source,'if err := checkProfilePairSource(checkpoint, summary, pair, wantProfile, wantSource, wantHasSource, byID, closure); err != nil {','if err := checkProfilePairSource(checkpoint, summary, pair, wantProfile, wantSource, wantHasSource, byID, closure); err != nil && (pair.profile != "standard" || pair.hasSource || wantProfile != "yolo" || wantHasSource) {','Admit exactly a standard-valued sourceless pair where the creation pair wants yolo/null; the first-profile negatives must fail by admission.')
run('N-referenced-creator',lease_source,'\tif raw.CreatorHostID != owner.HolderHostID {\n\t\treturn admittedCheckpoint{}, fmt.Errorf(\"%w: validated checkpoint %s is created by host %s, want owning lease holder %s\", ErrObservationUnavailable, raw.Digest, raw.CreatorHostID, owner.HolderHostID)\n\t}\n','\tif raw.CreatorHostID != owner.HolderHostID && raw.CreatorHostID != \"' + HOSTB + '\" {\n\t\treturn admittedCheckpoint{}, fmt.Errorf(\"%w: validated checkpoint %s is created by host %s, want owning lease holder %s\", ErrObservationUnavailable, raw.Digest, raw.CreatorHostID, owner.HolderHostID)\n\t}\n','Admit exactly a hostB-created checkpoint past the shared creator gate; the raw checkpoint tokens remain present while wrong_creator behavioral negatives must fail by admission.')
run('N-referenced-variant',lease_source,'\tif err := checkCheckpointPersistence(raw, sessionID, sessionKind); err != nil {\n\t\treturn admittedCheckpoint{}, err\n\t}\n','\tif err := checkCheckpointPersistence(raw, sessionID, sessionKind); err != nil && sessionID != \"' + IDA + '\" {\n\t\treturn admittedCheckpoint{}, err\n\t}\n','Admit exactly the idA wrong persistence variant past the shared admission gate; the raw checkpoint tokens remain present while wrong_variant behavioral negatives must fail by admission.')
run('N-referenced-temporal',lease_source,'if ownerIndex < resumeIndex {','if ownerIndex < resumeIndex && ownerIndex < 0 {','Admit exactly the future-owned referenced checkpoint past the owner-before-consumer temporal gate while preserving ownerIndex and resumeIndex; same-lease and later-consumer controls still refuse other invalid cases.')
run('N-parked-union-refusal',revalidate_source,'\t\tif row.Projection.Parked != nil {\n\t\t\treturn fmt.Errorf("%w: required authority for session %s is unknown: %s", ErrObservationUnavailable, plan.SessionID, row.Projection.Parked.BlockingReason)\n','\t\tif row.Projection.Parked != nil && row.Projection.RecordID == "" {\n\t\t\treturn fmt.Errorf("%w: required authority for session %s is unknown: %s", ErrObservationUnavailable, plan.SessionID, row.Projection.Parked.BlockingReason)\n','Admit parked union copies with a nonempty record; the agreeing-record member validates over unknown authority while the divergent member shifts to the record-agreement refusal.')
run('N-capability-name',summary_source,'\t\tif !admitted[name] {','\t\tif !admitted[name] && name != "invented_capability" {','Admit exactly the invented_capability name past the closed Section 7.3 vocabulary; every registry name and every other invented name still behave.')
run_census('N-census-raw-owner-object','\tif context.object != nil && owners[context.object] {\n\t\treturn true\n\t}','\tif context.object != nil && (owners[context.object] || context.name == "admitCheckpoint") {\n\t\treturn true\n\t}','Admit exactly a raw validatedCheckpoint consumer whose method is merely named admitCheckpoint; exact *types.Func identity remains required for every other consumer.')
run_census('N-census-unknown-callback','\tcallback, ok := callee.(*types.Var)\n\treturn ok && callbackParameters[context.object][callback]','\tcallback, ok := callee.(*types.Var)\n\treturn ok && (callbackParameters[context.object][callback] || callback.Name() == "review14UnknownAdmission")','Admit exactly the package-level unknown checkpointAdmission callback; the same census must reject the callback because its object is not the exact approved profile parameter.',r'^TestRev14CallbackProvenanceRejectsUnknownFunctions/unknown_callback.go$')
run_census('N-census-alias-value-normalization','func rev14UnaliasSealedValueType(typ types.Type) types.Type {\n\treturn types.Unalias(typ)\n}','func rev14UnaliasSealedValueType(typ types.Type) types.Type {\n\treturn typ\n}','Skip alias normalization on exactly the nested sealed-value recursion path; the container-of-alias census plant must fail.',r'^TestRev15AliasAwareRecordConsumptionCensus$')
run('B-peer-order',source,'\tsort.Strings(ids)\n\tvar out []candidate','\tsort.Sort(sort.Reverse(sort.StringSlice(ids)))\n\tvar out []candidate','Reverse peer-source tie ordering; behavioral determinism fixture must fail.')
run('B-session-order',source,'return out[i].Projection.SessionID < out[j].Projection.SessionID','return out[i].Projection.SessionID > out[j].Projection.SessionID','Reverse session output order; behavioral list fixture must fail.')
run('B-tie-break',source,') < holderHost(best.selection',') > holderHost(best.selection','Reverse the identical-copy host tie break; the deterministic selection fixture must fail.')
run('C-harmless-comment',lease_source,'// checkLeaseChain validates the winner','// checkLeaseChain validates the winner\n// classifier control: comment only, no behavior change.','Applied control: a behavior-preserving plant survives through the same instrument.')
run('C-not-applied',source,'TOKEN_THAT_IS_NOT_PRESENT','replacement','Application control: missing patch is not a killed mutant.')
run('C-compile-failure',source,'package sessquery','package sessquery\nthis is invalid Go','Compilation control: no named failed behavioral test is not a kill.')
with (output / 'control-after.log').open('w') as log:
    after = subprocess.run(['go','test','./internal/sessquery','./internal/sessrepo','-count=1'],cwd=copy,stdout=log,stderr=subprocess.STDOUT,timeout=180)
results.append(dict(mutant='control-after',classification='CONTROL', exit=after.returncode))
assert source.read_text() == base and selector_source.read_text() == selector_base and revalidate_source.read_text() == revalidate_base and plan_source.read_text() == plan_base and summary_source.read_text() == summary_base and lease_source.read_text() == lease_base and census_source.read_text() == census_base and repo_source.read_text() == repo_base
(output / 'mutants.json').write_text(json.dumps(results,indent=2)+'\n')
# Controls are reported separately from the applied behavioral denominator.
bad = [row for row in results if row['mutant'].startswith(('N-','B-')) and row['classification'] != 'KILLED']
raise SystemExit(1 if bad or after.returncode else 0)
