# CR9 review verdict: changes requested

TASK-260909-2ez769 / CR-TASK-260909-2ez769-9 revision 9.
Reviewer RUN-260916-a52140. Base 305875134f8d344eb86ef926c7fbca3fed85c49d;
immutable candidate tree 5ed8cdd619684f023fd71e61bcd076c3b1764a2b.
Review-only: immutable archive under task scratch; no live product edits, index writes,
commits, checkpoints, branch operations or integration.

## F1 — P2: indirect-call census still fails open

Repeat-of: CR8 F1 / CR7 F1 / CR6 F3 (prevention-instrument completeness).
`internal/config/writepath_census_types_test.go`, visitCall, still conditions refusal
of types.Var callees on censusPotentialMutationObject. That helper accepts only
signatures beginning with (string, []byte). A func() error can mutate a captured
file equally well. Interface checks likewise use name/signature heuristics, and
censusIsInterfaceType does not handle *types.TypeParam. This is not the requested
fail-closed unresolved-target contract.

Independent production-position plants: generic method-set Execute() error,
an immediately invoked anonymous struct's function field, and a deferred function
parameter. Every plant compiles, TestWritePathCensusGate PASSES and
TestReviewerRoguePlantExecutes PASSES after replacing real file bytes without a hold.
Each combined subprocess exits 0. The test injects the writing callback into a
production func() error variable; the production indirect call is precisely the
unresolved edge the gate claims to refuse. This tests the callback boundary, not
a claim that the current product itself installs that callback. No compile error,
reflection, unsafe, deleted guard, or empty test selection is counted.

All eight previous source-position shapes are detected, including the repaired
dot-import value: each exits 1 with a census assertion failure and a passing runtime
witness. **8 of 11 independent plants detected; 11 of 11 runtime witnesses passed.**
See census-dot-results.json, census-plants/*/{reviewer_rogue.go,reviewer_rogue_test.go,test.log}
and probes.py. The script ran twice; the second also records exact executed witness
bytes (the first archive-copy expression saved its template rather than generated
witness). Final files and logs are from the second run.

Required repair: remove mutation-signature guessing from unresolved-call admission.
Explicitly identify and justify allowed indirect edges in censused definitions;
ordinary unresolved calls must refuse irrespective of their argument shape or name.
Keep these three executable controls. Audit censused-definition exceptions rather
than claiming their bodies have a stronger rule merely because their names appear
in a list. This review does not certify that exception set as complete.

## F2 — P2: OpenForConfig moves the caller-controlled binding upstream

Repeat-of: CR8 F2 / CR7 F2 (wrong-resource capability).
`internal/hosttrust/custody.go:112-122` accepts independent stateDir and configPath,
canonicalizes the latter, then stores it without establishing a relationship to
the coordination root. bindConfigPath in hold.go checks that caller-supplied stored
path; ValidateConfigPath checks liveness/path but not target-root ownership.

The committed foreign-store test now passes because Open produces an unconfigured
Store. Change only that opening to
`hosttrust.OpenForConfig(otherStateDir, fixture.filename)` and the configured foreign
Store mints a valid token for the target. While the target Store lock remains held
by a separate goroutine, writeTempReplace returns nil and installs the foreign bytes.
The race-enabled reviewer test logs both nil write/read errors and actual replacement
bytes, then fails the expected ErrExclusiveHoldResourceMismatch assertion (exit 1).
See reviewer_foreign_test.go and foreign-exact-final.log.

Zero, released, generic foreign and configured-wrong-path controls all pass in that
same command. A symlink path resolving to the correct configuration also passes the
additional identity probe (symlink.log). Thus rejecting unconfigured Stores did not
repair wrong-root authority. The new hold-resource-skip mutant proves only the
unconfigured-Store refusal, not the required wrong-root case.

Required repair: establish the authoritative configuration/coordination resource
relationship in the model; do not accept an arbitrary tuple through a differently
named constructor. Enforce that relationship at the writer boundary and keep this
configured-foreign-root test plus a narrowing mutant admitting precisely wrong-root
while preserving zero/released/wrong-path refusals. Preserve documented configurable
path semantics; do not silently invent an incompatible fixed path layout.

## Functional AC accounting and bounds

**19 of 21 functional AC rows driven.** The table below retains the established
functional decomposition, rechecked against named tests, production source and pinned
sections 6.6 / 11.10.2 / 11.10.3. It is not exhaustive normative-clause coverage and
does not discharge F1/F2. Clean config/hosttrust/peeridentity package suites pass.

| # | Row | Production call / named test | Result |
|---|---|---|---|
| 1 | Closed Config4 | config.Decode / TestDecodeConfiguration4Refusals | pass |
| 2 | Historical compat + explicit migration | config.Migrate/migrate / TestMigrateRefusesV4Target, TestMigrateRefusesV4Downgrade, TestLegacyMigrateCannotOverwriteCommittedV4, TestLegacyMigrateConvergesUnreplacedMarkerThenCommits, TestLegacyMigrateRefusesAfterReplacementLanded | pass |
| 3 | Closed trust document | hosttrust.DecodeTrust / TestDecodeTrustRefusals | pass |
| 4 | Missing/corrupt/unreadable trust | hosttrust ReadSnapshot / TestReadSnapshotMissingStore, TestCorruptTrustRefusedNeverEmpty, TestReadSnapshotRefusesUnreadableMarker | pass |
| 5 | Fresh issuance | hosttrust IssueCredential / TestIssueCredentialProfile | pass |
| 6 | Exact profile + time bounds | hosttrust VerifyProfile / TestVerifyProfileRefusals, TestVerifyProfileRefusesJustPastExpiry | pass |
| 7 | Explicit tuple enrollment | hosttrust Store.Enroll / TestEnrollRefusals | pass; OOB human verification remains operator act |
| 8 | Mapping uniqueness | hosttrust DecodeTrust+admitEntry / duplicate-root/key/digest rows | pass |
| 9 | Rotation/retirement bounds | hosttrust Rotate, MarkRetiring / TestRotateBoundedWindow, TestMarkRetiringCapsAtLeafExpiry | pass |
| 10 | Revocation tombstones | hosttrust Revoke, Enroll / TestRevokeTombstone, TestRevokedLeafNeverReenrolls | pass |
| 11 | Old-generation mutation refusal | hosttrust WithMutationAuthorization / TestWithMutationAuthorizationRefusesStaleGeneration, TestMutationAuthorizationRefusesStaleAfterConvergence | pass |
| 12 | Shared authorization serialization | hosttrust Open/openLock, WithMutationAuthorization+Revoke / TestFirstOpenPreservesHeldLock, TestConcurrentOpenAttachesToExistingLock, TestMutationAuthorizationSerializesWithRevocation, TestMutationAuthorizationSerializesWithSeparateStoreRevocation | pass |
| 13 | Coherent config+trust snapshot | config LoadCoherent / TestSurvivingReaderConvergesInterruptedApply, TestSurvivingReaderConvergesInterruptedRollback, TestSurvivingReaderRefusesDivergedConfig | pass |
| 14 | Unix owner custody | hosttrust Open, ValidateCustody / TestCustodyRefusals, TestValidateCustodyBindsDirectory | pass on darwin |
| 15 | Windows equivalent ACLs | hosttrust verifyOwner/installOwnerOnlyACL/secureStaged / custody_windows_test.go (7 native tests) | NOT DRIVEN: code + tests present, runtime unverified (no runner) |
| 16 | Complete peers + selected credential | config PreviewV4 / TestPreviewV4PeerEnrollment, TestPreviewV4CredentialGates | pass |
| 17 | Exact preview + confirmation | config ApplyV4 / TestApplyV4PreviewMismatch, TestApplyV4ConfirmRequiredWithDrops | pass |
| 18 | Current source/generation apply | config applyV4+JointCommit / TestApplyV4StaleSource, TestApplyV4StaleGeneration, revalidate* tests | pass, incl. in-lock revalidation |
| 19 | Crash-durable config/trust pair | config applyV4/rollbackV4 + hosttrust JointCommit/resolveFailedReplace/compensateLocked/Recover/converge / TestApplyV4CrashConvergesGeneration, TestSurvivingReaderConvergesInterruptedApply, TestApplyV4CompensationCannotOverwriteConvergedCommit, TestRollbackV4CompensationCannotOverwriteConvergedCommit | pass |
| 20 | Explicit backup rollback | config RollbackV4 / TestRollbackV4*, TestPublishedBackupNamesStayExcluded, TestApplyV4BackupIsClassifiedConfigCopy, TestRollbackV4FailedReplaceKeepsIntent, TestRollbackV4FailedReplaceKeepsMarkerWhenRestoreFails | pass |
| 21 | Secret exclusion across export surfaces | hosttrust MatchExcludedFromReplication/ExcludedConfigDirName / matcher tests + persisted handoff + owner notes; diagnostics unowned | NOT DRIVEN: matcher enforced, no consumer calls it yet |


Row 15 remains NOT DRIVEN: Windows runtime DACLs and backup inheritance are unverified;
Windows cross-build is not custody execution. Row 21 remains NOT DRIVEN: exclusion
matcher exists, but consumer enforcement is not implemented; diagnostics remains
unowned. No post-lock-release dispatch, mesh-wide revocation, physical power-loss,
or authenticated operator verification guarantee is inferred. RPC remains its owner's
scope. The named source checks are in ac-test-presence.txt; abbreviated table references
remain families rather than additional independently counted rows.

## Validation and preservation

- Independently ran clean `GOPROXY=off go test ./internal/config ./internal/hosttrust
  ./internal/peeridentity -count=1 -cover`: exit 0; 93.5%, 77.9%, 97.5% respectively.
  This includes producer census controls and production behavior tests.
- Independently ran eight old and three new census plants, race-enabled foreign-root
  probe with five committed hold controls, and the symlink identity probe.
- Independently ran hosttrust neutral + hold-resource-skip, and config neutral +
  replace-require-skip + writepath-method-value-skip. Raw overlays, JSON output and
  subprocess exits are attached. A token-preserving static-gate mutation executes
  the behavioral package suite; its census assertion must not be relabeled as a
  production runtime refusal. Full historical mutation batteries were NOT rerun.
- Offline go mod tidy -diff and builds for darwin/arm64, linux/arm64, windows/amd64
  each exit 0. x/tools is test-only; README tools row is accurate within the stated
  census limitation. No independent full-repository test/coverage/vet/CI replay.
- Producer archive inventory and reports were read. Handoff validation advertises
  26/26 command shards but is bounded; omitted output is not independently certified.
  It also prints 366 board issues while its command exits 0. These are recorded as
  emitted diagnostics, not converted into a new passing board-health assertion.
- RPC backup 15/15 and parked delta 11/11 hashes match. Literal zero path overlap is
  false: README.md overlaps the backup manifest, as already documented in CR8; its
  candidate change is the test-tool row. No RPC implementation paths were absorbed.
- All 67 candidate delta paths match the immutable tree; patch SHA-256 is
  f72cba2bfde30ef120e26a348f873e840e5190f2679a03a79257d2f680a76006.
  Pinned SPEC digest matches source commit 0cbdf100dbf84df50c64f792b1f940e3a67859a6.
  SSH/peer checkpoint directories remain unchanged through replay; only the scoped
  peeridentity/v4_interop_test.go is added. Both checkpoint signatures verify (exit 0).
- An initial package command used the parent module path and failed setup (exit 1);
  corrected commands run inside the extracted candidate. A later cleanup snippet
  had a wrong relative cwd; the ensuing package run still included the foreign probe
  and correctly failed on it (packages-with-foreign-probe.log). The final clean run
  removed the reviewer probe and passed. Neither setup error is a product defect.
- Required skills read; canonical CLI used via configured wrapper; directive reads
  succeeded with none recorded. No installation or diagram changes.

## Verdict

**Changes requested → to-dev.** Two recurring P2 prevention defects remain.
No external blocker or human-only decision was established. Attach this verdict,
raw evidence and the logbook outcome before routing. No accept_cr or commit_ack.
