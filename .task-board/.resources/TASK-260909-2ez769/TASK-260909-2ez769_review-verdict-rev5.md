# CR5 review verdict: changes requested

Task TASK-260909-2ez769; CR-TASK-260909-2ez769-5 revision 5.
Base a89328ca85a613abde39660f4be749a8fcf57903; immutable candidate tree a823eb20be497c3fa0510e7da4d3506ddb284431.
Reviewer RUN-260916-f8925e. Production review is read-only; all probes run in an extracted immutable tree. No commit, checkpoint, branch/index mutation or integration.

## F1 — P1: failed joint commit restores configuration outside the authorization lock

Repeat-of: CR4 F1 / CR3 crash-transaction family, newly exercised error-recovery interleaving (not a failure of the repaired original crash probes).

`internal/hosttrust/joint.go:197` releases the exclusive lock when JointCommit returns ErrJointReplacementDurable. Both applyV4 (migration_v4_apply.go:106) and rollbackV4 (:242) then call restoreSourceAfterJointFailure (:258). That function calls writeTempReplace BEFORE acquiring any authorization lock; only its later store.Recover acquires one. It also does not revalidate the pending marker, current bytes, or generation before restoring.

Executed `TestReviewerRestoreCannotOverwriteConvergedCommit`, through production applyV4 and LoadCoherent, exit 1:

1. Real config replacement installs Config4, with the durable pending marker.
2. The injected filesystem seam temporarily changes trust.json mode to 0644. Production custody validation refuses the subsequent trust commit, returning ErrJointReplacementDurable and releasing the lock.
3. Immediately before the error handler's source-restoration rename, the seam restores 0600 and runs the surviving Store's production LoadCoherent. It legitimately completes recovery, commits generation 4, removes the marker, and admits Config4/generation 4.
4. The failed writer resumes its unlocked source restore, installing Config3. Its Recover sees no marker. A fresh production LoadCoherent accepts Config3/generation 4.

Observed failure: `unlocked recovery replaced admitted Config4 with 3.0.0 without a generation change: before=4 after=4`.

This is a deterministic scheduling/fault seam against real disk state and production locking, not a fake recovery implementation. The transient custody fault models the failing commit branch; the race also follows any recoverable trust-write/read failure. The result violates sections 6.6 and 11.10.3: an already-admitted config is replaced outside the shared generation transaction, without invalidating its generation. Another legitimate writer or a held mutation boundary can also interleave with that unlocked write. The rollback caller uses the identical unsafe helper; the executable counterexample here drives apply, not a separately executed rollback-failure variant.

Repair error compensation inside the same authorization transaction/lock, or reacquire and revalidate exact intent/config/generation before choosing a safe converged outcome. Never blindly restore after another process completed or superseded the operation. Add deterministic production tests for apply and rollback trust-commit failures racing surviving readers and another writer/held boundary; retain original crash and stale-generation tests. Ordinary implementable rework, no external blocker.

## CR4 repair checks and remaining bounds

- All seven original reviewer probes pass on CR5: stale mutation, same/separate-store revocation contention, cross-process first-open, surviving apply crash, reopened apply crash, surviving rollback crash. Child exit 77 is explicitly asserted. Producer RED logs reproduce the prior two crash failures and inode replacement.
- The shared-to-exclusive transition rechecks the current marker via convergeLocked AFTER obtaining exclusive ownership. A second writer can intervene during the release/acquire gap, but its current intent is then converged/refused; there is no stale-marker observation reused by this path. The new F1 is a separate unlocked compensation path.
- Atomic exclusive creation prevents the original production first-open replacement. verifyLockFile's 500ms retry checks current path custody but does not pin an inode: an externally replaced regular owner-only file can pass a later retry. No remaining normal production initializer replaces the inode; this review does not claim protection against an owner externally replacing lock files. Windows CREATE_NEW path inspected; native execution unverified.
- Seven native Windows custody tests now exist and call production verifyOwnerFile/verifyOwnerDir/secureStaged/Open. The symlink test can skip without privilege. Cross-compilation does not turn row 15 into a pass; runtime evidence remains unverified.
- Backup bytes are classified as configuration copies, not private-key custody; TestApplyV4BackupIsClassifiedConfigCopy drives the backup path. Windows profile ACL inheritance is a stated bound, not owner-only proof.
- Verified the exact API/test handoff notes on all seven named downstream tasks. Diagnostic export remains explicitly unowned. Matcher-only row 21 is not consumer-driven; no RPC implementation requested or added.

## AC accounting

19 of 21 functional AC rows driven: 17 bounded passing rows and 2 failing rows (13 and 19). Rows 15 and 21 remain not driven at native/consumer entries. This functional decomposition is not a claim that every normative clause or every gate has an exhaustive mutant.

| # | Row | Production call / named test | Result |
|---|---|---|---|
| 1 | Closed Config4 | config.Decode / TestDecodeConfiguration4Refusals | pass |
| 2 | Historical compat + explicit migration | config.Migrate / TestMigrateRefusesV4Target, TestMigrateRefusesV4Downgrade | pass |
| 3 | Closed trust document | hosttrust.DecodeTrust / TestDecodeTrustRefusals | pass |
| 4 | Missing/corrupt/unreadable trust | hosttrust ReadSnapshot / TestReadSnapshotMissingStore, TestCorruptTrustRefusedNeverEmpty, TestReadSnapshotRefusesUnreadableMarker | pass |
| 5 | Fresh issuance | hosttrust IssueCredential / TestIssueCredentialProfile | pass |
| 6 | Exact profile + time bounds | hosttrust VerifyProfile / TestVerifyProfileRefusals, TestVerifyProfileRefusesJustPastExpiry | pass |
| 7 | Explicit tuple enrollment | hosttrust Store.Enroll / TestEnrollRefusals | pass; OOB human verification remains operator act |
| 8 | Mapping uniqueness | hosttrust DecodeTrust+admitEntry / duplicate-root/key/digest rows | pass |
| 9 | Rotation/retirement bounds | hosttrust Rotate, MarkRetiring / TestRotateBoundedWindow, TestMarkRetiringCapsAtLeafExpiry | pass |
| 10 | Revocation tombstones | hosttrust Revoke, Enroll / TestRevokeTombstone, TestRevokedLeafNeverReenrolls | pass |
| 11 | Old-generation mutation refusal | hosttrust WithMutationAuthorization / TestWithMutationAuthorizationRefusesStaleGeneration, TestMutationAuthorizationRefusesStaleAfterConvergence | pass |
| 12 | Shared authorization serialization | hosttrust Open/openLock, WithMutationAuthorization+Revoke / TestFirstOpenPreservesHeldLock, TestConcurrentOpenAttachesToExistingLock, TestMutationAuthorizationSerializesWithRevocation, TestMutationAuthorizationSerializesWithSeparateStoreRevocation | pass on original CR4 probes |
| 13 | Coherent config+trust snapshot | config LoadCoherent / TestSurvivingReaderConvergesInterruptedApply, TestSurvivingReaderConvergesInterruptedRollback, TestSurvivingReaderRefusesDivergedConfig | FAIL new recovery race; original crash probes pass |
| 14 | Unix owner custody | hosttrust Open, ValidateCustody / TestCustodyRefusals, TestValidateCustodyBindsDirectory | pass on darwin |
| 15 | Windows equivalent ACLs | hosttrust verifyOwner/installOwnerOnlyACL/secureStaged / custody_windows_test.go (7 native tests) | NOT DRIVEN: code + tests present, vet + test-compile pass, runtime unverified (no runner) |
| 16 | Complete peers + selected credential | config PreviewV4 / TestPreviewV4PeerEnrollment, TestPreviewV4CredentialGates | pass |
| 17 | Exact preview + confirmation | config ApplyV4 / TestApplyV4PreviewMismatch, TestApplyV4ConfirmRequiredWithDrops | pass |
| 18 | Current source/generation apply | config applyV4+JointCommit / TestApplyV4StaleSource, TestApplyV4StaleGeneration, revalidate* tests | pass, incl. in-lock revalidation |
| 19 | Crash-durable config/trust pair | config applyV4/rollbackV4 + hosttrust JointCommit/Recover/converge / TestApplyV4CrashConvergesGeneration (fixed), surviving-reader crash tests, barrier matrix | FAIL unlocked error recovery; original crash probes pass |
| 20 | Explicit backup rollback | config RollbackV4 / TestRollbackV4*, TestPublishedBackupNamesStayExcluded, TestApplyV4BackupIsClassifiedConfigCopy | pass |
| 21 | Secret exclusion across export surfaces | hosttrust MatchExcludedFromReplication/ExcludedConfigDirName / matcher tests + persisted handoff + 7 owner notes; diagnostics unowned | NOT DRIVEN: matcher enforced, no consumer calls it yet |


## Validation and provenance

- Independently ran hosttrust/config/peeridentity `-count=1 -cover`, exit 0: 77.9% / 93.2% / 97.5%. Original reviewer probe suite exit 0; new error-recovery probe exit 1. All use Go 1.25.5 darwin/arm64.
- Verified producer source manifests against immutable candidate: 31 hosttrust and 32 config source/test hashes. Producer mutation evidence has behavioral failures rather than build errors/empty selection, with applied passing neutral controls. Config label-precision kill stays separately classified; hosttrust joint-converge-skip is a call-site removal, not independently counted as narrowing. Its documented backstop survivor is not a kill.
- Targeted reviewer mutation replay and exact outcomes are in mutants.log and mutants/table.md; probe-only files were moved out before replay. These greens do not override F1.
- Reused attached rev5 local full-suite/build/vet/race/fixture/fuzz and handoff validation evidence rather than repeating unrelated checks. Board validation reports issues despite exit 0; that is not a clean-board proof. No hosted CI and no native Windows runtime execution.
- Verified RPC backup 15/15 and parked untracked 11/11 SHA256 hashes; no overlap with the 53 candidate paths. SSH transport untouched; peeridentity delta only adds v4_interop_test.go. Base signature verifies. CR patch digest matches 4a6e3beb15acb6e67dbd628ea9139c75dd333254312375a93347e51af57bbc01.
- Initial restore-probe write used a wrong scratch-relative path, so the first command selected no tests; that invocation is invalid evidence. Corrected probe executes the named test and fails behaviorally as quoted above.
- Skills: project-management and Curator go-testing-tools; required architecture-diagrams instructions read, no diagram changes needed. No tool installation. Directives successfully read, none recorded.

## Verdict

Changes requested; route to to-dev. Do not accept revision 5. Preserve the managed checkpoint, all candidate work and RPC packet. No commit_ack or integration from this reviewer. Review evidence archive includes the exact probe sources, logs, source verification and mutation replay.
