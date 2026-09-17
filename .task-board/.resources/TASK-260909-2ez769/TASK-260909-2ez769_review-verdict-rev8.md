# CR8 review verdict: changes requested

TASK-260909-2ez769 / CR-TASK-260909-2ez769-8 revision 8.
Reviewer RUN-260916-05f8c2. Base 305875134f8d344eb86ef926c7fbca3fed85c49d;
immutable candidate tree 2c81ffcae1411ac73e5c3fbc3fe8b231916cd409.
Review and probes only; no live product edits, staging, commits, checkpoints or integration.

## F1 — P2: typed census still admits an executable function-value writer

Repeat-of: CR7 F1 / CR6 F3, prevention-instrument completeness.
`internal/config/writepath_census_types_test.go:373` handles bare identifiers only
when their object is in the gated-helper map. OS/raw mutation value checks are
implemented only for SelectorExpr (`:388`). A dot-imported OS function used as a
value is a resolved types.Func, but escapes the first check; its later invocation
is a types.Var, which visitCall accepts without inspecting its target.

The following new production-position source compiles and evades the gate:

```go
import . "os"
var reviewerWrite = WriteFile
func reviewerRogue(path string, data []byte) error {
    return reviewerWrite(path, data, 0600)
}
```

`go test ./internal/config -count=1 -v -run
'^(TestWritePathCensusGate|TestReviewerRoguePlantExecutes)$'` exits **0**.
The gate PASSES and the witness PASSES after replacing real file bytes without
any hold. No unsafe, reflection, external sabotage, build failure, or empty test
selection is involved. See census-plants/dot_import_function_value/test.log and
its exact source/test files; probe-dot.py reproduces it.

The five CR7 shapes are repaired: direct control, map value, aliased os import,
factory-returned closure initializer, backend method value all yield real census
assertion failures (exit 1), with successful behavioral witnesses. Two additional
new shapes — promoted embedded backend method and package-level slice of writer
functions — also fail the gate while their witnesses execute successfully.
**7 of 8 independently planted writer shapes detected; 8 of 8 witnesses executed.**
The six older committed controls also pass in the clean package suite.

Required repair: classify resolved function objects consistently across identifier
and selector expression forms, including value escapes, rather than adding this
variable or import spelling to a list. Keep this compiling behavioral counterexample
and both new passing detection controls. The type loader walks config and hosttrust
production syntax for the current Go build context; it does not walk every platform
or build-tagged file in one invocation. TestWritePathCensusFailsClosedOnUnresolvedCallee
proves rejection of an ill-typed undefined callee, not sound target resolution of
well-typed indirect calls. State that distinction and the platform bound explicitly.
Backend definitions are package/receiver/method keyed, but isHoldEntryCall still
recognizes hold entries by method name alone; this is not an all-edges identity proof.

## F2 — P2: a foreign store can mint a path-bound token for the target configuration

Repeat-of: CR7 F2, wrong-resource capability in the mandatory writer guard.
`internal/hosttrust/hold.go:74-83` checks liveness and the config path only.
`WithExclusiveHoldForConfig` accepts any absolute config path and binds it without
relating the store root to that configuration's coordination resource. The config
writer's requireHoldForConfig (`internal/config/migration.go:313`) does not check
that root either. Trust-side requireHoldForRoot is effective, but it is not the
config writer's check.

The exact original CR7 probe now passes because an unbound token refuses. Change
only its production-position writer from `other.WithExclusiveHold(...)` to
`other.WithExclusiveHoldForConfig(filename, ...)` and the same test FAILS:

1. A separate goroutine holds the target store's exclusive lock and remains waiting.
2. Another, unrelated store obtains its own genuine live token bound to the target filename.
3. writeTempReplace accepts it and installs replacement bytes while the target lock is held.
4. The test reads those exact bytes before reporting failure.

The combined race-enabled run has **TestReviewerForeignHoldCannotWriteLockedPair FAIL**,
**TestWritePathCensusGate PASS**, exit **1**. See foreign-bound.log,
foreign-bound-writer.go and foreign-bound-test.go. The original unbound version,
zero and released controls all pass in foreign-original.log. This is an admitted
ordinary new caller, not a claim that the existing ApplyV4/RollbackV4 callers currently
choose the wrong store; severity remains P2 for the mandatory guard/instrument.

Required repair: make the mutation boundary verify the expected coordination resource
as well as the configuration path. Caller-supplied filename equality alone does not
prove that the right authorization lock is held. Keep the foreign-store/path-bound
probe, same-resource controls, and a narrowing mutant admitting exactly the wrong-root
case while preserving zero/released/wrong-path refusals.

Additional resource probes pass under -race (exit 0): right root / different
absolute config path refuses; a same-root separately reopened handle accepts the
live root capability; passing it to commitDocument for a different root refuses
and leaves target trust bytes unchanged; the scoped token refuses after release.
See resource-binding-probe.go and resource-bindings.log. The first version of
this probe attempted Open while holding the same lock in the same goroutine;
Open calls Recover and blocks as designed. That invalid nested-lock setup was
terminated with SIGQUIT (go test exit 1), retained in
resource-bindings-invalid-setup.log, and is not counted as a product failure or
passing evidence. The corrected probe opens the second handle before acquiring
the hold, then tests reuse against that same coordination root.

## Functional acceptance and bounds

**19 of 21 functional AC rows driven** by the candidate's named production-entry
tests. I checked the pinned sections 6.6, 11.10.2 and 11.10.3 and rechecked the named
test definitions in source; clean config/hosttrust/peeridentity package suites passed.
This is the established functional decomposition, not exhaustive normative-clause
coverage or proof of the failed prevention instrument. It does not justify accepting
F1/F2. Rows 15 and 21 retain their prior accounting exactly.

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


Row 15: NOT DRIVEN. Windows equivalent ACL code and seven native tests exist;
cross-build/vet is not runtime DACL evidence. Windows backup ACL inheritance
remains a bound. Row 21: NOT DRIVEN. Matcher-only evidence and persisted consumer
handoff; no consumer enforcement call yet, diagnostics still unowned. RPC
TLS/hello/dispatch/watchdog/stream integration stays with the RPC owner. No
mesh-wide revocation, physical power-loss, post-lock-release dispatch guarantee
or externally authenticated operator action is inferred.

## Verification, evidence scope, and preservation

- Independently ran `GOPROXY=off go test ./internal/config ./internal/hosttrust
  ./internal/peeridentity -count=1 -cover`: exit 0; coverage 93.5%, 77.5%, 97.5%.
  The package run includes the committed six earlier census controls, negative
  hold tests, migration/revalidation, compensation races, crash child-process
  tests, trust lifecycle and credential-profile tests. It is not a rerun of all
  historical reviewer-only probes or the full repository CI matrix.
- Independently reran the three requested narrowing mutants against this candidate:
  hold-resource-skip -> TestHeldExclusiveRejectsUnboundConfigPath; replace-require-skip
  -> TestReplaceDurablyRefusesZeroHold and TestWriteTempReplaceRefusesZeroHold;
  writepath-method-value-skip -> TestWritePathGateFlagsExecutableRogues/backend_method_value.
  Each subprocess exited 1 with named test assertion failures; both package neutral
  controls exited 0. The token-preserving census mutant executes the full config
  behavioral suite, not only a static checker; its specific kill is the census
  assertion, with the executable witness/control suite run alongside. Do not label
  that static assertion itself a runtime mutation failure. Raw JSON logs, overlays,
  exact mutant source and results are attached. No compile-failure, unapplied or
  empty-selection kill is counted.
- Older rev7 mutation batteries are historical, NOT exact-source certification of
  CR8: 6/34 hosttrust source files and 5/39 config source files in their manifests
  differ (and CR8 adds typed census files). older-mutation-scope.json enumerates
  those differences. I reran the three requested mutants and neutral controls,
  not all earlier mutants. The producer's old evidence cannot be reused as if the
  entire source/test/dependency identity were unchanged.
- The new module is test-only (`internal/config/writepath_census_types_test.go`),
  pinned x/tools v0.47.0 with x/mod v0.37.0 and x/sync v0.21.0. Offline `go mod tidy`
  exits 0 and leaves go.mod/go.sum byte-identical. Offline `go build ./...` for
  darwin, linux, windows each exits 0. No Windows runtime execution is claimed.
  A pinned test-only Go dependency is consistent with the repository's testing
  approach; no policy forbids it. README's added tools row correctly identifies
  the test import, command and output location; its symbol-resolution description
  must not be read as a complete indirect-call proof in view of F1.
- The handoff validation log advertises 26/26 green shards but explicitly omits
  5,130,118 bytes. I do not certify the omitted raw commands from that aggregate.
  Full repository tests/coverage/vet and all CI shards were not independently
  rerun here; producer logs/claims remain separate from the commands above.
- All 66 candidate changed paths match the immutable tree. The attached patch's
  SHA-256 matches 0da174ee75dae1abf85ad32c79d869c606568a53bdf435cd581ce7f7aacaf327.
  Normative local SPEC.v0.6.0.md matches its pinned digest and source commit
  0cbdf100dbf84df50c64f792b1f940e3a67859a6. Source reads used this local pinned copy.
- RPC backup **15/15** and parked **11/11** hashes match. The requested literal
  zero path overlap is FALSE: README.md occurs in both the RPC backup manifest
  and this 66-path delta. The exact README delta is solely the go/packages tools
  row; no RPC implementation was absorbed. All other RPC paths have zero overlap.
  No backup or parked file was modified. Pre/post refresh manifests are retained
  and included as historical transition evidence, not substituted for final-tree
  provenance. SSH/peer checkpoint directories are byte-identical across old and
  refreshed checkpoint tips; candidate adds only peeridentity/v4_interop_test.go.
  Both replayed checkpoints ef0b63b and 3058751 verify their signatures (exit 0).
  `git rev-list --count HEAD..main` returned 0; all review evidence is bound to
  the immutable candidate rather than an assumed worktree revision.
- Required project-management, architecture-diagrams and Curator go-testing-tools
  skills read. PATH task-board wrapper resolves to the canonical Curator binary.
  Directive reads succeeded with none recorded. No installation or diagram work.

## Verdict and route

**Changes requested -> to-dev. Do not accept CR8.** Two P2 prevention defects
remain, both repeat-of the CR7 finding classes. Ordinary implementable rework;
no external blocker or human decision. Preserve existing product repairs and
foreign work, fix the classes, and publish a fresh managed candidate. Evidence
and this verdict are attached before routing. No accept_cr or commit_ack issued.
