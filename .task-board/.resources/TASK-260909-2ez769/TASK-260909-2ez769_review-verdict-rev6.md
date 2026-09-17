# CR6 review verdict: changes requested

TASK-260909-2ez769 / CR-TASK-260909-2ez769-6 revision 6.
Reviewer RUN-260916-c3e675. Base a89328ca85a613abde39660f4be749a8fcf57903;
immutable candidate tree b0bb3d9d600d4c323aa12480b3285512f96167c7.
Read-only product review; probes run in an extracted candidate under task scratch.
No live product edits, index/branch/HEAD changes, commit, checkpoint, integration or commit_ack.

## F1 — P1: delayed legacy migration overwrites an already committed Config4

Repeat-of: CR5 F1 / CR4 F1 / CR3 config-trust transaction family; distinct production path, not a recurrence in the repaired compensation closure.

`internal/config/migration.go:182` (`migrate`) reads the source without the authorization lock, then `:236` calls `replaceDurably` without revalidating the source under that lock. The rev6 census row 10 explicitly exempts this writer; `writepath_census_test.go:225` admits it. Checking only a concurrently outstanding marker does not cover a writer that resumes after the marker has been committed and removed.

Executed TestReviewerLegacyCannotOverwriteCommittedV4 (exit 1):

1. Start a production legacy Config2-to-Config3 migration and pause at its replacement Rename seam, after it has read Config2 and prepared its backup/replacement.
2. A second production Migrate completes Config2-to-Config3. PreviewV4 and ApplyV4 then commit Config4, and LoadCoherent admits Config4/generation 4.
3. Resume the first legacy Rename. It succeeds, returns Changed=true with nil error, and overwrites Config4 with Config3.
4. LoadCoherent succeeds with Config3/generation 4. No acknowledged RollbackV4 occurred, and no generation invalidation occurred.

Observed: `legacy migration silently overwrote admitted Config4 with 3.0.0 at unchanged generation 4`.

This drives actual migration, disk rename, preview, apply and coherent-read code; the seam only orders legitimate concurrent operations. It violates pinned sections 6.6 (no unacknowledged legacy replacement) and 11.10.3 (coherent config/trust authorization transaction). The census's legacy fail-closed claim is false. Bring this writer into the common serialization/revalidation contract; an unlocked pre-write source check alone still has the race. Preserve ordinary standalone Config1/2/3 migration behavior. Add a production regression and narrowing evidence for delayed legacy writes spanning a completed Config4 commit.

## F2 — P1: failed replacement recovery deletes the intent needed to reject or converge partial state

Repeat-of: CR3/CR4/CR5 crash/coherence family; replace-failure branch distinct from the repaired trust-bump-failure branch.

`internal/hosttrust/joint.go:233-235` unconditionally removes the pending marker whenever replace() returns an error. But `internal/config/migration.go:353-379` can return ErrMigrationSync + ErrMigrationRecovery AFTER the replacement rename succeeded and the attempted source-restoration rename failed. Thus an error does not prove the config is unchanged.

Executed two production probes (both exit 1):

- TestReviewerFailedReplacementCannotLoseIntent: ApplyV4 installs Config4; injected post-rename directory-open failure triggers recovery; injected recovery Rename failure leaves Config4 on disk. JointCommit removes the marker. LoadCoherent admits Config4 at the OLD generation 3.
- TestReviewerFailedRollbackCannotLoseIntent: after successful ApplyV4, RollbackV4 installs Config3; the same failure pair leaves Config3 on disk and removes the marker. LoadCoherent admits Config3 at the OLD generation 4.

The original operation truthfully returns failure, but subsequent readers silently accept the incomplete config/trust pair. This violates sections 6.6 and 11.10.3. Do not discard intent based only on a replace error. Keep the failure outcome inside the common locked state-resolution protocol, verify actual source/replacement pins, and retain recoverable/refusing intent whenever rollback is uncertain. Cover both apply and rollback, including failure after a durable replacement and failure during restoration. The compensation fix currently runs only after transactLocked fails, so it does not protect these paths.

## F3 — P2: census gate misses executable unlocked writers

Repeat-of: the CR6 class-level census requirement intended to prevent further CR3/4/5 bypass paths; new evidence-instrument finding.

TestWritePathCensusGate detected **1 of 4** reviewer plants. Every plant compiled, ran TestReviewerRoguePlantExecutes, and replaced a real config file through a production-source caller without acquiring the authorization lock:

| Plant in new production file | Gate result | Test process exit |
|---|---|---:|
| Direct FuncDecl calling writeTempReplace (positive control) | detected | 1 |
| Package var bound to a function literal calling writeTempReplace | missed | 0 |
| FuncDecl assigning writeTempReplace to a local variable, then calling it | missed | 0 |
| Method receiver calling os.Rename, reached through a wrapper | missed | 0 |

At `writepath_census_test.go:128-138` the scanner skips non-FuncDecl declarations; at `:201-212` it follows only syntactic call names and ignores all os-qualified selectors. Consequently the three supplied controls cannot establish the claimed completeness. The normal method receiver is not itself invisible: this plant specifically exercises the blanket stdlib exemption inside it. The allowlist also explicitly lets F1's unlocked migrate pass.

Repair the gate's claimed domain using symbol-aware handling or explicit fail-closed restrictions for unresolved writer references and raw filesystem entry points. Restrict backend exemptions to their actual definitions, not every os call. Add these executable controls and retain the token-preserving behavioral test. Do not merely add these names to a list or describe unobserved paths as absent. The source walk is attached as write-calls.log; 12 prose rows exist, but row 10 is unsafe and rows 1/2/11 omit F2's partial-failure outcome. No complete census certification is given.

## What the CR5 repair does establish

The original trust-bump-failure compensation now runs inside JointCommit's continuous exclusive hold. Marker pins and replacement bytes are checked, restore is verified against source bytes, and failed compensation retains intent. The unlocked restoreSourceAfterJointFailure is gone.

All seven original reviewer probes pass (exit 0), including actual child exit-77 crash cases and cross-process first-open serialization. All six apply/rollback compensation races pass with -race (exit 0), exercising a surviving reader, another writer and an authorization boundary. The port TestApplyV4CompensationCannotOverwriteConvergedCommit checks final config bytes and unchanged generation after a clean abort.

The exact original TestReviewerRestoreCannotOverwriteConvergedCommit now blocks on flock inside the restore seam's recursive LoadCoherent. The bounded replay times out at 5 seconds (exit 1); its stack proves the exclusive hold is still held, **not** that this old synchronous test passes. The six asynchronous ports supply the actual passing final-state evidence.

Not re-reading committed trust during compensation can leave (source config, generation+1) if a trust rename landed but its sync reported failure. The bump transaction changes only the generation, not credential entries. With the lock held continuously and the error returned, the extra invalidation is fail-safe, not silent credential loss or admission of an uncommitted replacement. This does not excuse F2, where replacement remains installed at the old generation after its intent is removed. Power-loss guarantees still depend on actual fsync durability; no disk-honesty proof is claimed.

## AC accounting

**19 of 21 functional AC rows driven: 15 bounded passing rows, 4 failing rows (2, 13, 19, 20).** Rows 15 and 21 retain their CR5 bounds exactly. This is a functional decomposition of pinned sections 6.6 and 11.10.2/3, not an exhaustive normative-clause or all-gates mutation count. Names below were checked against candidate source; changed-package tests were executed independently.

| # | Row | Production call / named test | Result |
|---|---|---|---|
| 1 | Closed Config4 | config.Decode / TestDecodeConfiguration4Refusals | pass |
| 2 | Historical compat + explicit migration | config.Migrate/migrate / TestMigrateRefusesV4Target, TestMigrateRefusesV4Downgrade; TestReviewerLegacyCannotOverwriteCommittedV4 | FAIL concurrent legacy overwrite (F1); sequential controls pass |
| 3 | Closed trust document | hosttrust.DecodeTrust / TestDecodeTrustRefusals | pass |
| 4 | Missing/corrupt/unreadable trust | ReadSnapshot / TestReadSnapshotMissingStore, TestCorruptTrustRefusedNeverEmpty, TestReadSnapshotRefusesUnreadableMarker | pass |
| 5 | Fresh issuance | IssueCredential / TestIssueCredentialProfile | pass |
| 6 | Exact profile + time bounds | VerifyProfile / TestVerifyProfileRefusals, TestVerifyProfileRefusesJustPastExpiry | pass |
| 7 | Explicit tuple enrollment | Store.Enroll / TestEnrollRefusals | pass; authenticated OOB verification is the operator act |
| 8 | Mapping uniqueness | DecodeTrust + admitEntry / duplicate-root/key/digest refusal cases | pass |
| 9 | Rotation/retirement bounds | Rotate, MarkRetiring / TestRotateBoundedWindow, TestMarkRetiringCapsAtLeafExpiry | pass |
| 10 | Revocation tombstones | Revoke, Enroll / TestRevokeTombstone, TestRevokedLeafNeverReenrolls | pass |
| 11 | Old-generation mutation refusal | WithMutationAuthorization / TestWithMutationAuthorizationRefusesStaleGeneration, TestMutationAuthorizationRefusesStaleAfterConvergence | pass within its lock contract |
| 12 | Shared authorization serialization | Open/openLock, WithMutationAuthorization+Revoke / TestFirstOpenPreservesHeldLock, TestConcurrentOpenAttachesToExistingLock, TestMutationAuthorizationSerializesWithRevocation, TestMutationAuthorizationSerializesWithSeparateStoreRevocation | pass for these entries |
| 13 | Coherent config+trust snapshot | LoadCoherent / surviving-reader crash tests, six compensation races, new reviewer probes | FAIL F1/F2; original crash/compensation controls pass |
| 14 | Unix owner custody | Open, ValidateCustody / TestCustodyRefusals, TestValidateCustodyBindsDirectory | pass on darwin |
| 15 | Windows equivalent ACLs | verifyOwner/installOwnerOnlyACL/secureStaged / custody_windows_test.go (7 native tests) | NOT DRIVEN: code/tests present; runtime unverified; cross-build is not runtime evidence |
| 16 | Complete peers + selected credential | PreviewV4 / TestPreviewV4PeerEnrollment, TestPreviewV4CredentialGates | pass |
| 17 | Exact preview + confirmation | ApplyV4 / TestApplyV4PreviewMismatch, TestApplyV4ConfirmRequiredWithDrops | pass |
| 18 | Current source/generation apply | applyV4+JointCommit / TestApplyV4StaleSource, TestApplyV4StaleGeneration, revalidate tests | pass for in-lock apply revalidation |
| 19 | Crash-durable config/trust pair | applyV4/rollbackV4 + JointCommit/Recover / crash/barrier tests; TestReviewerFailedReplacementCannotLoseIntent | FAIL partial replace failure discards intent (F2) |
| 20 | Explicit backup rollback | RollbackV4 / TestRollbackV4*, TestPublishedBackupNamesStayExcluded, TestApplyV4BackupIsClassifiedConfigCopy; TestReviewerFailedRollbackCannotLoseIntent | FAIL partial rollback recovery (F2); happy-path/backup controls pass |
| 21 | Secret exclusion across export surfaces | MatchExcludedFromReplication/ExcludedConfigDirName / matcher tests and persisted consumer handoff | NOT DRIVEN: matcher-only; no consumer calls yet; diagnostic export unowned |

RPC TLS launch/hello/stream shutdown/dispatch integration remains the RPC owner's scope. No invented guarantee after lock release, native Windows DACL execution, global revocation, or secret-exclusion consumer integration. Windows backup DACL inheritance remains a bound.

## Validation, provenance and evidence reuse

- Independently executed immutable-candidate config/hosttrust/peeridentity tests with -count=1 -cover: exit 0, coverage 93.3% / 77.9% / 97.5%. Go 1.25.5 darwin/arm64.
- Seven prior probes: exit 0. Six compensation race tests with -race: exit 0. New legacy and partial-failure regression probes: real behavioral failures, exit 1. Census controls: one detected / three missed, with executed disk-mutation witnesses.
- Verified 31 hosttrust and 34 config source/test SHA256 entries against candidate bytes. All 39 producer mutation plants match exactly once and their archived modified files equal the expected candidate overlay. All expected named failures are present in raw logs, with nonempty selections and no build-error kills. Recorded subprocess statuses: hosttrust 29 narrowing + 1 call-site kill, neutral exit 0; config 6 semantic kills + 1 separately classified label-precision kill, neutral exit 0. These batteries were audited from attached exact-source evidence, not rerun by this reviewer. Preserve-token reorder runs the behavioral suite. They do not cover F1/F2/F3.
- Reused attached rev6 full-suite/build/vet/race/fixture/fuzz and local handoff validation evidence; handoff reports 26/26 exact command shards green and test-case coverage unknown. No unrelated full replay. Producer board validation exit 0 with 201 reported issues is not clean-board proof. No hosted CI or native Windows run.
- Verified RPC backup 15/15 and parked delta 11/11 hashes, zero overlap with the 55 candidate paths. SSH transport unchanged; peeridentity change only v4_interop_test.go. Base signature verifies. Patch SHA256 matches 3d8057e3c4c2eb090eb70843440a68e3b1287218a109a551b7118ce6d7c7371f. Local SPEC.v0.6.0.md SHA256 matches the pin for commit 0cbdf100dbf84df50c64f792b1f940e3a67859a6.
- The first partial-replacement probe write used the wrong scratch-relative path, so that invocation selected no tests. Its preserved log is explicitly invalid evidence; the corrected two-test execution fails as above. The first prior-probe regex omitted two probes; original-seven.log reruns all seven explicitly and passes.
- Tool readiness/provenance logged; task-board wrapper resolves to the canonical Curator CLI. Required project-management, architecture-diagrams and Curator go-testing-tools instructions read; no diagram or tool installation needed. Directive reads succeeded with none recorded.

## Verdict and required rework

Changes requested; route to **to-dev**, do not accept CR6. Ordinary implementable rework, no external blocker or human decision required. Unify every actual config writer and partial-failure outcome under the authorization transaction, repair the census instrument, add these regressions and narrowing plants, then publish a new managed candidate. Preserve all candidate/SSH/peer/RPC bytes. Evidence archive contains runnable probes, plant sources, real logs and audit scripts; no credentials are included.
