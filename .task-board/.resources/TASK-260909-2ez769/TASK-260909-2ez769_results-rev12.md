# TASK-260909-2ez769 rev12 outcome — call-site census + side-effect-free binding refusal

Producer: RUN-260917-39ee2b (muse-spark max), continuing the partial
rev12 candidate of RUN-260917-b6de1b (codex, killed by provider quota
before publishing; no candidate bytes discarded).

Base: managed `task-board worktree refresh-candidate` advanced the
Story worktree to protected trunk `62d446304391af187c417a88a2e14012b467956e`,
replaying the checkpointed peer/SSH leaves as `efe119a`/`5e54874`
(signatures verify). Candidate branch head `5e54874`, worktree
UNCOMMITTED. Candidate delta: 74 paths (17 tracked modifications +
57 untracked additions; the CR11 `__pycache__` bytecode is removed and
`internal/hosttrust/binding_rebind_test.go` is added).

## CR11 findings repaired

F1 (call-site admission): `writepath_census_types_test.go` now admits
an interface/type-parameter edge only for the exact `*types.Func`
object at an exact inventoried source site (file/line/column), with
top-level fail-fast hold-guard dominance for the four pair writers
(`replaceDurably`, `writeTempReplace`, `commitDocument`,
`commitConfigBinding`). The 119-site inventory is enumerated with
per-group production justifications in
`TASK-260909-2ez769_write-path-census-rev12.md`. The committed
regression `TestWritePathCensusRejectsUnlistedInterfaceCallSite` ports
the CR11 `reviewer_callsite_test.go` probe and extends it: plants
before AND after `requireHoldForConfig` must both rename real files
in the child witness (PASS) while the child census refuses both
unlisted sites (FAIL). The narrowing mutant
`writepath-interface-callsite-skip` preserves method identity while
dropping site/hold admission and is killed by that regression. The
former method-spelling interface admission predicate is removed;
remaining name-keyed maps are fail-closed concrete-call detectors
only (claim narrowed in the census doc). The 17 standalone CR11
plants remain killed.

F2 (refused rebind claims a target): `ensureConfigBindingLocked`
validates the existing store-side association and the target-side
association under the bootstrap lock BEFORE publishing either side; a
refused request leaves both the existing association and the
previously unclaimed target untouched. Committed RED-first regression
`TestReviewerRefusedRebindLeavesTargetUnclaimed` (RED log:
`rebind-red-rev12.log`), plus
`TestCompetingFirstBindingsPublishExactlyOneAssociation` (two handles
opened before binding; exactly one winner, loser side-effect free)
and `TestInterruptedFirstBindingIsRepairedAfterReopen` (target
durable + index missing is repaired on reopen; target never rebound).
The CR11 second-pair foreign-root refusal still holds.

Hygiene: no `__pycache__`/`.pyc` under `internal/`; all Python runs
with `PYTHONDONTWRITEBYTECODE=1`; README tool row describes the
all-indirect-call policy and its exact exceptions; evidence attached
as real binaries (verified below).

## AC coverage: 19 of 21 rows driven (rows 15, 21 stated bounds)

| # | Acceptance row | Production call / named test evidence | Result |
|---:|---|---|---|
| 1 | Closed Config4 | `config.Decode`; `TestDecodeConfiguration4Refusals` | pass |
| 2 | Historical compatibility + explicit migration | `config.Migrate`/legacy migration; `TestMigrateRefusesV4Target`, `TestMigrateRefusesV4Downgrade`, `TestLegacyMigrateCannotOverwriteCommittedV4`, `TestLegacyMigrateConvergesUnreplacedMarkerThenCommits`, `TestLegacyMigrateRefusesAfterReplacementLanded` | pass |
| 3 | Closed trust document | `hosttrust.DecodeTrust`; `TestDecodeTrustRefusals` | pass |
| 4 | Missing/corrupt/unreadable trust | `hosttrust.ReadSnapshot`; `TestReadSnapshotMissingStore`, `TestCorruptTrustRefusedNeverEmpty`, `TestReadSnapshotRefusesUnreadableMarker` | pass |
| 5 | Fresh issuance | `hosttrust.IssueCredential`; `TestIssueCredentialProfile`, `TestIssueSelfEnrollsActive` | pass |
| 6 | Exact profile + time bounds | `hosttrust.VerifyProfile`; `TestVerifyProfileRefusals`, `TestVerifyProfileRefusesJustPastExpiry`, wrong-key-usage/EKU/SAN tests | pass |
| 7 | Explicit tuple enrollment | `hosttrust.Store.Enroll`; `TestEnrollRefusals`, `TestEnrollBoundsPerHost`; OOB human verification remains an operator act | pass within stated OOB boundary |
| 8 | Mapping uniqueness | `hosttrust.DecodeTrust` + admission; duplicate credential/root/key/digest cases in `TestDecodeTrustRefusals`, `TestEnrollBoundsPerHost`, lifecycle tests | pass |
| 9 | Rotation/retirement bounds | `hosttrust.Rotate`, `MarkRetiring`; `TestRotateBoundedWindow`, `TestRotateCapsAtLeafExpiry`, `TestMarkRetiringCapsAtLeafExpiry` | pass |
| 10 | Revocation tombstones | `hosttrust.Revoke`, `Enroll`; `TestRevokeTombstone`, `TestRevokedLeafNeverReenrolls`, `TestRevocationClosesAuthorization` | pass |
| 11 | Old-generation mutation refusal | `hosttrust.WithMutationAuthorization`; `TestWithMutationAuthorizationRefusesStaleGeneration`, `TestMutationAuthorizationRefusesStaleAfterConvergence` | pass |
| 12 | Shared authorization serialization | `hosttrust.Open`/lock, `WithMutationAuthorization`, `Revoke`; `TestFirstOpenPreservesHeldLock`, `TestConcurrentOpenAttachesToExistingLock`, `TestMutationAuthorizationSerializesWithRevocation`, `TestMutationAuthorizationSerializesWithSeparateStoreRevocation`; binding precondition `Store.EnsureConfigBinding` + `TestReviewerRefusedRebindLeavesTargetUnclaimed`, `TestCompetingFirstBindingsPublishExactlyOneAssociation`, `TestInterruptedFirstBindingIsRepairedAfterReopen` (rev12) | pass |
| 13 | Coherent config+trust snapshot | `config.LoadCoherent`; `TestSurvivingReaderConvergesInterruptedApply`, `TestSurvivingReaderConvergesInterruptedRollback`, `TestSurvivingReaderRefusesDivergedConfig`, barrier and revalidation tests | pass |
| 14 | Unix owner custody | `hosttrust.Open`, `ValidateCustody`; `TestCustodyRefusals`, `TestValidateCustodyBindsDirectory`, `TestOpenCreatesOwnerOnlyLayout` | pass on Darwin |
| 15 | Windows equivalent ACLs | `verifyOwner`, `installOwnerOnlyACL`, `secureStaged`; `custody_windows_test.go` seven native tests | stated bound: no Windows runtime runner; cross-build, test-compile, and vet pass |
| 16 | Complete peers + selected credential | `config.PreviewV4`; `TestPreviewV4PeerEnrollment`, `TestPreviewV4CredentialGates` | pass |
| 17 | Exact preview + confirmation | `config.ApplyV4`; `TestApplyV4PreviewMismatch`, `TestApplyV4ConfirmRequiredWithDrops`; write discipline `TestWritePathCensusRejectsUnlistedInterfaceCallSite` (rev12) | pass |
| 18 | Current source/generation apply | Config4 apply + `hosttrust.JointCommit`; `TestApplyV4StaleSource`, `TestApplyV4StaleGeneration`, revalidation and stale-marker tests | pass |
| 19 | Crash-durable config/trust pair | Config4 apply/rollback + `hosttrust.JointCommit`, `Recover`, convergence; `TestApplyV4CrashConvergesGeneration`, surviving-reader/barrier/compensation tests | pass |
| 20 | Explicit backup rollback | `config.RollbackV4`; `TestRollbackV4*`, `TestPublishedBackupNamesStayExcluded`, `TestApplyV4BackupIsClassifiedConfigCopy` | pass |
| 21 | Secret exclusion across export surfaces | `hosttrust.MatchExcludedFromReplication`, `ExcludedConfigDirName`; matcher and persisted handoff/owner-note evidence | stated bound: repository matcher only; downstream consumers and diagnostics are unowned |

Rows 15/21 are NOT DRIVEN. No post-lock-release dispatch, mesh-wide
revocation, physical power-loss, or authenticated-human verification
guarantee is inferred. RPC transport/admission remains outside this leaf.

## Mutation batteries (exact final source, per-plant raw logs)

Reran in full on the post-refresh final tree with
`PYTHONDONTWRITEBYTECODE=1` (4 parallel streams, 9 bounded groups,
exits from subprocess return codes; per-mutant overlay, `test.log`,
`results.json`, `table.md` under
`.temp/TASK-260909-2ez769/mutations-{config,hosttrust}-rev12post-groupN`,
also archived in the evidence tarball):

- Config4 (`internal/config/mutations_v4.py`): 1 neutral control
  passed (exit 0) + 13 narrowing/precision mutants killed (exit 1
  with the expected named test failing), 0 survivors. Includes the
  new `writepath-interface-callsite-skip` (killed by
  `TestWritePathCensusRejectsUnlistedInterfaceCallSite`),
  `writepath-method-value-skip`, and `config-association-root-skip`.
- Host Trust Store (`internal/hosttrust/mutations.py`): 1 neutral
  control passed (exit 0) + 34 narrowing/precision mutants killed
  (exit 1 with the expected named test failing), 0 survivors.
  Includes `hold-state-root-skip` (killed by
  `TestHeldExclusiveRejectsForeignConfigStateRoot`) and the
  preserve-token `compensate-verify-before-restore` order swap
  (behavioral suite kill).
- Aggregate: 47 killed, 2 neutrals passed, 0 survivors; every kill
  verified from `results.json` (exit 1 + expected test in failing
  set).

## Local validation (exact final source)

- `gofmt -l` (repo, excluding `.temp`/`.task-board`): clean.
- `go vet ./...` (native): exit 0.
- `go mod tidy -diff`: exit 0 (offline, `GOFLAGS=-mod=mod`).
- `git diff --check`: clean (excluding board checkout noise).
- `go build ./...`: exit 0 native, darwin/arm64, linux/arm64, windows/amd64.
- `go test ./... -count=1`: exit 0, all packages ok (see `go-test-all-rev12post.log`).
- Coverage (`-count=1 -cover`): config 92.9%, hosttrust 76.3%,
  peeridentity 97.5%, secprim 94.4%, localstore 83.8% — exit 0.
- Windows test compilation (`-exec /usr/bin/true -run '^$'`) and
  `GOOS=windows go vet` on config+hosttrust: exit 0 (no Windows
  runtime claim; stated bound row 15).
- `-race` on hosttrust+config: exit 0.
- Key verbose runs: `TestWritePathCensusGate` +
  `TestWritePathCensusRejectsUnlistedInterfaceCallSite` pass
  (11.99s); F2 trio passes (0.81s).
  (`census-regression-rev12post.log`, `binding-tests-rev12post.log`).
- Independent RED: the F2 trio against unfixed CR11 code fails with
  exit 1 on `TestReviewerRefusedRebindLeavesTargetUnclaimed`
  (`rebind-red-rev12post.log`); adjacent pins pass pre/post fix.
- Pinned spec: `internal/specdoc/SPEC.v0.6.0.md` sha256 matches
  `internal/specpin/v0.6.0.lock.json` (`0cbdf100dbf84df50c64f792b1f940e3a67859a6`).

## Preservation

- RPC backup manifest 15/15 and parked delta 11/11 sha256 match
  (`preservation-rev12post.log`).
- Candidate delta 74 paths; literal RPC-manifest overlap is README.md
  only (documented host-credentials/Config4/tooling section; original
  RPC README bytes preserved in backup). No RPC implementation path
  absorbed.
- SSH/peer checkpoint directories byte-identical across the replay;
  `internal/peeridentity/v4_interop_test.go` is this leaf's only
  peeridentity path.
- Replayed checkpoints `efe119a`/`5e54874` verify against the
  configured author key.

## Bounds restated

Native Windows ACL runtime execution and downstream
replication/diagnostic consumers are explicit bounds, not passes. The
census executes for the active Go build context. No claim covers
dispatch after shared-lock release or Windows DACL runtime
validation beyond cross-compilation.

## Handoff

Ready for review. Candidate UNCOMMITTED in the Story worktree; no
main push or integration performed. Independent review follows.
