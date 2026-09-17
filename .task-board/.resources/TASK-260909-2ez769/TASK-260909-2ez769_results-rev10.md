# TASK-260909-2ez769 — producer outcome rev10

Status: ready for independent review; candidate remains uncommitted in the managed Story worktree.

## Scope and provenance

- Task: `TASK-260909-2ez769`, implementation of Host Trust Store 1.0.0, credential lifecycle, authorization generations, and explicit Configuration 4.0.0 migration for pinned `relux-works/agent-session-manager-spec@v0.6.0` commit `0cbdf100dbf84df50c64f792b1f940e3a67859a6` (sections 6.6, 11.10.2, and 11.10.3).
- Worktree: `/Users/iv/Developer/ReluxWorks/agent-session-manager/.temp/STORY-260830-1kiyj6/worktree`.
- Branch: `task-board/story/STORY-260830-1kiyj6`.
- Current protected base: `origin/main=8626fb361c1d6b62d5f95c7be4cb33c85d4aadf3`; worktree `HEAD=305875134f8d344eb86ef926c7fbca3fed85c49d`, ahead 2 and behind 0.
- Delivery rule: no candidate commit, rebase, merge, push, or main integration was performed; the Story handoff snapshots the uncommitted worktree.

## Implementation delivered

- `internal/hosttrust` provides closed Trust Store decoding, owner-only custody, Profile-1 issuance and verification, explicit enrollment, per-host credential mapping, fresh-key bounded rotation/retirement, revocation tombstones, current-generation mutation authorization, shared/exclusive lock serialization, atomic crash recovery, and export exclusion.
- `config-binding.json` is an owner-only, immutable sidecar. It binds the selected config path to the Host Trust Store and checks the config-side selected `StateRoot` against the canonical store root. `JointCommit`, `LoadCoherent`, Config4 apply/rollback, and legacy pair writers require the prior explicit binding; missing, partial, unreadable, ambiguous, rebound, downgraded, or foreign-root state refuses.
- Config4 keeps the normative closed `mesh.host_channel` table exact: only `version` and `credential_id`. No non-normative store-root field was added to TOML, and no implicit legacy fallback remains. Config1/2/3 readers and legacy migration behavior remain covered.
- Config4 migration is explicit-preview, exact-preview-confirmed, complete-peer-enrollment, durable apply, and explicit-backup rollback. Failed replacement compensates under the same exclusive hold, verifies restored bytes, and preserves the marker when safe recovery cannot be proven.
- Trust/private material is excluded from replication surfaces; only the public enrollment artifact is intended for out-of-band transfer. The repository has no RPC/TLS/hello/dispatch/stream consumer implementation in this task; that boundary remains with the owning RPC task.

## Acceptance matrix

The enumerated production acceptance coverage is **19 of 21 rows driven**. Rows 15 and 21 are explicit stated bounds, not passing claims.

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

## Negative and narrowing-mutant evidence

- Config4 exact-source mutation battery: 1 neutral control passed; 11 narrowing/precision mutants killed; 0 survivors. Exact result: `.temp/TASK-260909-2ez769/mutations-config-final/results.json` and `table.md`.
- Host Trust Store exact-source mutation battery: 1 neutral control passed; 34 narrowing/precision mutants killed; 0 survivors. Exact results: `.temp/TASK-260909-2ez769/mutations-hosttrust-rev10-01/` through `-06/`, `-comp-restore/`, `-order/`, and `-root/`.
- Mutants include refusal weakening for custody, binding, mapping, generation, stale source/direction, expiration, revocation, markers, locks, convergence, compensation, state-root binding, and authorization. The `writepath-method-value-skip` mutation preserves the searched-for token and runs the behavioral suite; it is killed by behavior rather than only by the static checker.
- Initial relative-`GOCACHE` mutation attempts are retained as setup failures and excluded from kill counts. Final batteries ran with task-scoped absolute caches, restored all mutated source byte-for-byte, and report actual subprocess exits.

## Preservation audit

- RPC control backup: 15 files, 0 mismatches.
- Parked enumerated RPC delta: 11 files, 0 mismatches.
- Checkpointed SSH/peer work preserved; no `internal/rpcwire` or `UNRESOLVED_QUESTIONS.md` implementation path was absorbed into this leaf.
- Evidence: `.temp/TASK-260909-2ez769/preservation-final-01.log`.

## Validation evidence

| Check | Result | Evidence |
|---|---|---|
| Repository tests | 26/26 packages passed | `.temp/TASK-260909-2ez769/go-test-all-02.log` |
| Repository coverage | exit 0; all 26 package rows passed | `.temp/TASK-260909-2ez769/go-cover-all-02.log` |
| Touched-package race | config, hosttrust, peeridentity, secprim passed | `.temp/TASK-260909-2ez769/go-race-touched-01.log` |
| Static analysis | `go vet ./...` passed | `.temp/TASK-260909-2ez769/go-vet-all-01.log` |
| Native build | `go build ./...` passed | `.temp/TASK-260909-2ez769/go-build-native-01.log` |
| Windows build | `GOOS=windows GOARCH=amd64 go build ./...` passed | `.temp/TASK-260909-2ez769/go-build-windows-01.log` |
| Windows test compilation/vet | config, hosttrust, secprim compile and touched vet passed | `windows-test-*-01.log`, `go-vet-windows-touched-01.log` |
| Module/catalog/spec checks | `go mod tidy -diff`, catalog `-check`, tracecheck passed | `go-mod-tidy-diff-01.log`, `catalog-check-01.log`, `tracecheck-01.log` |
| Formatting/diff | gofmt and `git diff --check` passed | `format-check-02.log` |

Coverage highlights from the complete run: config 93.2%, hosttrust 78.2%, peeridentity 97.5%, secprim 94.4%; no package failed.

## Explicit bounds

- Windows ACL/custody runtime was not available in this Darwin run; Windows build, test compilation, and vet are evidence only for those checks.
- Downstream replication consumers and diagnostics export are outside this repository/task and remain unowned; row 21 is not claimed as a repository-wide consumer guarantee.
- RPC TLS, hello, admission, dispatch, and stream lifecycle remain outside this task.
- Evidence covers the implemented fsync/atomic-replacement protocol and crash/intervention tests; it does not claim untested physical power-loss behavior beyond those mechanisms.
- Windows backup DACL inheritance and any platform-specific deployment custody need a Windows runtime owner.

## Handoff state

Outcome artifacts are ready for the developer-to-reviewer handoff. The worktree is intentionally uncommitted; no main integration or review acceptance is claimed here.
