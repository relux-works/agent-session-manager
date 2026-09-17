# CR10 review verdict: changes requested

TASK-260909-2ez769 / CR-TASK-260909-2ez769-10 revision 10.
Reviewer RUN-260917-da6f01. Base 305875134f8d344eb86ef926c7fbca3fed85c49d;
immutable candidate tree bfd4993dc1dfe7739d7515f1998ead5a4788f14d.
Patch SHA-256: 78e170e66f981f8e62b52410e65250350a8718e4257ac8d90beee1dbdde7a56b.
Review-only immutable archive and isolated probes; no live product, index, branch,
HEAD, checkpoint or integration changes. All review scratch is under
`.temp/review-rev10/` in the assigned Story worktree.

## F1 — P2: interface/type-parameter call admission still uses method-name guessing

Repeat-of: CR9 F1 / CR8 F1 / CR7 F1 / CR6 F3.
`internal/config/writepath_census_types_test.go:649` calls
`callUsesInterfaceMutationMethod`; lines 775–780 classify an indirect interface
call only when its name occurs in `censusIsMutationFunctionName`. `Execute() error`
is still silently admitted. A `*types.Func` interface method declaration is not
a resolved concrete implementation. The exception map additionally grants
`interfaceMethod: true` to whole enclosing functions, rather than enumerating
the precise interface-method object at each allowed edge. That is not the
required heuristic-free contract.

The eleven CR9 plants now fail with census assertions and passing executable
witnesses. However, the old generic plant is killed by the new refusal of the
secondary `reviewerEffect()` function variable, not by its generic Execute edge.
Two independent replacements isolate the actual boundary: a production generic
`reviewerInvoke[T reviewerAction](f T) { return f.Execute() }`, and the equivalent
non-generic interface call. The test supplies a concrete implementation that
writes a real temporary file. Both compile, both runtime witnesses pass, and
`TestWritePathCensusGate` passes: actual subprocess exit 0 in both cases.
This tests the unresolved production callback boundary, not a claim that the
current application itself installs that particular callback.

**13 of 15 independent plants detected; 15 of 15 runtime witnesses passed.**
The two requested new shapes (type-parameter method value and channel callback)
are detected. The two direct Execute-boundary shapes survive. See
`census-results.json`, `probes.py`, and `census-plants/*/test.log`; each plant's
exact production and witness sources are included. No compile error or empty
selection is counted as a kill.

Required repair: classify all interface/type-parameter calls as unresolved
concrete targets regardless of method spelling, including parenthesized callee
forms. Allow only explicit required edges by precise object identity, not a
boolean on the containing definition. Audit remaining initializer/function-field
and local-variable exemptions and justify each real production necessity.
Keep these direct boundary witnesses so an unrelated rejected call cannot mask
this gap. Do not describe the current gate as heuristic-free.

## F2 — P2: caller-supplied StateRoot still mints wrong-resource authority

Repeat-of: CR9 F2 / CR8 F2 / CR7 F2.
The durable sidecar is an improvement, but `EnsureConfigBinding(configPath,
stateDir)` (`internal/hosttrust/custody.go:154–170`) verifies the supplied stateDir
against the store root, not an authoritative association of the target document.
`config.requireHoldForConfig` (`migration.go:316–326`) repeats that comparison
using another caller-supplied stateDir. The attacker/caller still chooses the tuple.

Adaptation of the prior exact probe to CR10:

1. Hold the legitimate fixture Store's exclusive lock in another goroutine.
2. Open a foreign Store, then call
   `other.EnsureConfigBinding(fixture.filename, filepath.Dir(other.Root()))`.
3. Inside `other.WithExclusiveHold`, call
   `writeTempReplace(hold, filepath.Dir(other.Root()), osMigrationFileSystem{}, fixture.filename, replacement)`.

Binding succeeds; writeTempReplace returns nil; the target actually contains the
foreign replacement while its proper lock is still held. Race-enabled
`TestReviewerConfiguredForeignStore` fails its expected typed
ErrExclusiveHoldResourceMismatch assertion, exit 1. Exact source and
`foreign-final.log` are attached. This uses normal production binding APIs; no
sidecar forgery is needed. It demonstrates prevention-instrument failure, not
an external attacker privilege escalation or an assertion about an RPC caller.

Six committed zero/released/foreign/wrong-path controls pass alongside the
counterexample. Independent directly forged foreign-sidecar probe refuses when
passed the honest target stateDir; removing a legitimate binding then reopening
refuses until explicit re-establishment; symlink identity control passes.
`binding.log` records those three controls, exit 0. These controls establish
that the vulnerability is the caller-chosen association, not missing liveness
or canonicalization checks.

Required repair: make the authoritative configuration-to-coordination association
independent of the values a pair-helper caller supplies. Persisting a caller-chosen
path and comparing a second caller-chosen stateDir does not achieve that. Retain
this exact counterexample and a narrowing mutant at the real writer boundary.
Preserve normative Config4's closed two-field host_channel table and configurable
path semantics; do not add a non-normative TOML root field or hide the mismatch by
renaming another tuple-taking API. Any objectively incompatible product constraint
must be evidenced and routed explicitly, not declared solved by self-consistency.

## Functional AC accounting

**19 of 21 functional AC rows driven**, retaining rows 15 and 21 as stated bounds.
This is the established functional decomposition, not exhaustive normative-clause
coverage and not discharge of F1/F2. Re-read pinned sections 6.6, 11.10.2 and
11.10.3; all 44 exact test names in the producer report resolve in the candidate
(`ac-test-presence.json`). Production call sites and tests were rechecked; clean
config/hosttrust/peeridentity suites pass after removing reviewer plants.

| # | Acceptance row | Production call / named test evidence | Result |
|---:|---|---|---|
| 1 | Closed Config4 | `config.Decode`; `TestDecodeConfiguration4Refusals` | pass |
| 2 | Historical compatibility + explicit migration | `config.Migrate`/legacy migration; `TestMigrateRefusesV4Target`, `TestMigrateRefusesV4Downgrade`, `TestLegacyMigrateCannotOverwriteCommittedV4`, `TestLegacyMigrateConvergesUnreplacedMarkerThenCommits`, `TestLegacyMigrateRefusesAfterReplacementLanded` | pass |
| 3 | Closed trust document | `hosttrust.DecodeTrust`; `TestDecodeTrustRefusals` | pass |
| 4 | Missing/corrupt/unreadable trust | `hosttrust.ReadSnapshot`; `TestReadSnapshotMissingStore`, `TestCorruptTrustRefusedNeverEmpty`, `TestReadSnapshotRefusesUnreadableMarker` | pass |
| 5 | Fresh issuance | `hosttrust.IssueCredential`; `TestIssueCredentialProfile`, `TestIssueSelfEnrollsActive` | pass |
| 6 | Exact profile + time bounds | `hosttrust.VerifyProfile`; `TestVerifyProfileRefusals`, `TestVerifyProfileRefusesJustPastExpiry`, wrong-key-usage/EKU/SAN tests | pass |
| 7 | Explicit tuple enrollment | `hosttrust.Store.Enroll`; `TestEnrollRefusals`, `TestEnrollBoundsPerHost`; OOB human verification remains an operator act | pass within stated OOB boundary |
| 8 | Mapping uniqueness | `hosttrust.DecodeTrust` + admission; duplicate credential/root/key/digest cases in `TestDecodeTrustRefusals`, `TestEnrollBoundsPerHost`, and lifecycle tests | pass |
| 9 | Rotation/retirement bounds | `hosttrust.Rotate`, `MarkRetiring`; `TestRotateBoundedWindow`, `TestRotateCapsAtLeafExpiry`, `TestMarkRetiringCapsAtLeafExpiry` | pass |
| 10 | Revocation tombstones | `hosttrust.Revoke`, `Enroll`; `TestRevokeTombstone`, `TestRevokedLeafNeverReenrolls`, `TestRevocationClosesAuthorization` | pass |
| 11 | Old-generation mutation refusal | `hosttrust.WithMutationAuthorization`; `TestWithMutationAuthorizationRefusesStaleGeneration`, `TestMutationAuthorizationRefusesStaleAfterConvergence` | pass |
| 12 | Shared authorization serialization | `hosttrust.Open`/lock, `WithMutationAuthorization`, `Revoke`; `TestFirstOpenPreservesHeldLock`, `TestConcurrentOpenAttachesToExistingLock`, `TestMutationAuthorizationSerializesWithRevocation`, `TestMutationAuthorizationSerializesWithSeparateStoreRevocation` | pass |
| 13 | Coherent config+trust snapshot | `config.LoadCoherent`; `TestSurvivingReaderConvergesInterruptedApply`, `TestSurvivingReaderConvergesInterruptedRollback`, `TestSurvivingReaderRefusesDivergedConfig`, barrier and revalidation tests | pass |
| 14 | Unix owner custody | `hosttrust.Open`, `ValidateCustody`; `TestCustodyRefusals`, `TestValidateCustodyBindsDirectory`, `TestOpenCreatesOwnerOnlyLayout` | pass on Darwin |
| 15 | Windows equivalent ACLs | `verifyOwner`, `installOwnerOnlyACL`, `secureStaged`; `custody_windows_test.go` seven native tests | stated bound: no Windows runtime runner; cross-build, test-compile, and vet pass |
| 16 | Complete peers + selected credential | `config.PreviewV4`; `TestPreviewV4PeerEnrollment`, `TestPreviewV4CredentialGates` | pass |
| 17 | Exact preview + confirmation | `config.ApplyV4`; `TestApplyV4PreviewMismatch`, `TestApplyV4ConfirmRequiredWithDrops` | pass |
| 18 | Current source/generation apply | Config4 apply + `hosttrust.JointCommit`; `TestApplyV4StaleSource`, `TestApplyV4StaleGeneration`, revalidation and stale-marker tests | pass |
| 19 | Crash-durable config/trust pair | Config4 apply/rollback + `hosttrust.JointCommit`, `Recover`, convergence; `TestApplyV4CrashConvergesGeneration`, surviving-reader/barrier/compensation tests | pass |
| 20 | Explicit backup rollback | `config.RollbackV4`; `TestRollbackV4*`, `TestPublishedBackupNamesStayExcluded`, `TestApplyV4BackupIsClassifiedConfigCopy` | pass |
| 21 | Secret exclusion across export surfaces | `hosttrust.MatchExcludedFromReplication`, `ExcludedConfigDirName`; matcher and persisted handoff/owner-note evidence | stated bound: repository matcher only; downstream consumers and diagnostics are unowned |


Rows 15 and 21 remain NOT DRIVEN: no Windows runtime custody/DACL runner,
and no downstream replication/diagnostic consumer enforcement. Cross-compilation
is not Windows execution. No post-lock-release dispatch, mesh-wide revocation,
physical power-loss, or authenticated-human verification guarantee is inferred.
RPC transport/admission/stream behavior remains with its owning task.

## Validation and preservation

- Independently ran clean `GOPROXY=off go test ./internal/config ./internal/hosttrust
  ./internal/peeridentity -count=1 -cover`: exit 0; coverage 93.2%, 78.2%, 97.5%.
- Independently ran all 15 census plants, race-enabled foreign-root counterexample
  with six committed controls, and three binding/canonicalization controls.
- Independently ran hosttrust neutral + hold-state-root-skip: neutral exit 0;
  mutant exit 1 at TestHeldExclusiveRejectsForeignConfigStateRoot. This proves
  rejection of an honestly supplied foreign stateDir, not the caller-chosen tuple.
- Independently ran config neutral + replace-require-skip + writepath-method-value-skip:
  neutral exit 0; both mutants exit 1 with named failures. The token-preserving
  census mutant executes the behavioral package suite but is killed by its census
  expectation in TestWritePathGateFlagsExecutableRogues/backend_method_value;
  that assertion is not relabeled as a production runtime authorization refusal.
- Producer mutation archive inspected: 48 logged rows / 45 unique non-neutral
  names (11 config, 34 hosttrust). Raw failing test events and log digests are
  recorded in producer-mutation-audit.json. Full batteries were NOT rerun and
  their reported kills do not discharge the independent survivors.
- Offline `go mod tidy -diff` and builds for darwin/arm64, linux/arm64,
  windows/amd64 exit 0. x/tools is imported only by the census test. README's
  tools row still says writer-shaped indirect calls; broader prose in the
  producer census overstates the actual gate.
- Full-repository test/coverage/vet and complete handoff validation were accepted
  as attached producer evidence, NOT independently replayed or recertified.
  The bounded validation log reports 26/26 green command shards and 366 board
  diagnostics with exit 0. These are diagnostics, not a passing board-health claim.
- RPC backup 15/15 and parked delta 11/11 hashes match. Literal zero overlap is
  false: README.md overlaps the backup manifest, as previously recorded; the
  candidate contains documentation changes, no RPC implementation paths.
- All 69 delta paths match the immutable candidate; pinned SPEC digest matches
  commit 0cbdf100dbf84df50c64f792b1f940e3a67859a6. SSH/peer checkpoint directories
  are unchanged through replay; only scoped peeridentity/v4_interop_test.go is
  added. ef0b63b and 3058751 signatures verify, exit 0. HEAD remains 3058751;
  `git rev-list --count HEAD..main` is 0 (local-ref observation only).
- Review execution anomalies retained honestly: the first package run overlapped
  a source-reading census plant and failed on that plant (packages.log), so it
  is not clean-candidate evidence. The final sequentially stable package run
  passed (packages-clean.log). A foreign-probe preparation used a wrong relative
  cwd and failed before writing its file; that run executed only committed
  controls (foreign.log). The corrected exact counterexample is foreign-final.log.
- project-management and go-testing-tools skills read; explicitly supplied
  architecture-diagrams skill read but no diagram work needed. Missing local
  skill path resolved through installed global symlink. No installs. Directive
  reads succeeded with none recorded.

## Verdict and routing

**Changes requested → to-dev.** Two repeated P2 prevention defects remain.
Reopen checklist items 6, 11, 12 and 18: their broad completion statements do not
hold in the presence of these executable counterexamples. No external blocker
or human-only decision was established. Attach this verdict, evidence archive
and review logbook before routing. No accept_cr, commit_ack, commit or integration.
