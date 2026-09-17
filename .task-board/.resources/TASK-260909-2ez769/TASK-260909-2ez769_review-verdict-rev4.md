# Revision 4 review: changes requested

TASK-260909-2ez769 / CR-TASK-260909-2ez769-4 revision 4.
Base: a89328ca85a613abde39660f4be749a8fcf57903.
Candidate tree: 2c8909c9c7425bb3a90c00b8db3762f43b4dbf39.
Review against an immutable archive; no managed-worktree product edits, commits, integration or landing.

## Blocking findings

### F1 — P1: surviving processes accept an interrupted config/trust pair

`internal/hosttrust/transact.go`, `WithSharedSnapshot`/`readSnapshotLocked`, and
`internal/hosttrust/authorize.go`, `readCommitted`, never inspect the pending
joint-commit marker. Recovery happens only in `Open` (or explicit `Recover`).
The exclusive OS lock disappears when the writer crashes, but other processes
with already-open Stores remain alive and can acquire it and read old trust
alongside the replaced configuration. Locking alone does not make that pair a
committed pair. The same omission reaches authorization and ordinary trust
transactions; the marker is not a general admission barrier.

Executed real child-process probes against the candidate:

- `TestReviewerCrashBetweenConfigAndGeneration`: production `applyV4` exits 77
  after the real config rename; surviving parent's production `LoadCoherent`
  succeeds with the replaced configuration and generation 3 (the old value).
  Parent test FAIL, go test exit 1.
- `TestReviewerRollbackCrashSurvivingReader`: first performs real `ApplyV4`, then
  production `rollbackV4` child exits 77 at the replacement seam; surviving
  `LoadCoherent` succeeds with generation 4 (the old value). FAIL, exit 1.
- `TestReviewerCrashReopenConverges`: a fresh `Open` after the apply crash does
  converge and passes with fatal assertions on read/open errors. This repair
  works for new openers, but does not cover surviving processes.

Repair all read/admission/transaction entry points under the authorization lock
so unresolved intent is recovered or refused before exposing a snapshot or
running a boundary. Cover both apply and rollback, surviving processes as well
as reopen, and marker cleanup/error ordering. Do not silently refresh a stale
request to the recovered generation. Section 11.10.3 requires coherent committed
reads across processes, not just after reopening a Store.

The shipped `TestApplyV4CrashConvergesGeneration` still returns success early on
Load, Configuration, Open and ReadSnapshot errors (migration_v4_apply_test.go,
roughly lines 419–436). Those branches cannot prove successful convergence.
Replace them with explicit expectations, distinguishing a tested refusal from
successful recovery. No shipped rollback child-crash test was found.

### F2 — P1: concurrent first-open replaces a held authorization lock

`internal/hosttrust/lock_unix.go`, `openLock`, uses Lstat-absent followed by
CreateTemp and replacing Rename. `lockInitMu` serializes only goroutines in one
process. Two processes can both observe absence; the delayed process renames
its staged lock over the first process's already-held lock. Subsequent locks
then attach to a different inode and no longer serialize.

`TestReviewerCrossProcessLockInitialization` deterministically pauses a real
child's production `openStore` after its absent-lock Lstat. Parent `Open` creates
the lock, initializes trust and holds an exclusive acquisition. Releasing the
child allows its `Open` to finish while the parent still holds the old lock.
The test verifies both facts: `returnedWhileHeld=true sameInode=false`.
FAIL, exit 1 on Darwin/arm64. Only the filesystem observation is synchronized;
creation, rename, Open and locking are production code against OS files.

Use atomic non-replacing lock initialization shared by all processes; never
replace a live lock inode. Exercise concurrent first-open and existing-lock
contention through production entry points. Inspect the analogous Windows
initialization path too. Do not infer a native Windows failure from this Darwin
execution.

### F3 — P2: remaining acceptance/evidence claims exceed their reach

- Native Windows DACL read/install code now exists; CR3's owner-SID-only
  implementation criticism is no longer accurate. However `TestOwnerOnlyGrants`
  is a pure evaluator and cross-build/vet do not execute native owner-only,
  other-principal or reparse acceptance/refusal. There are no native Windows
  tests in the candidate. Row 15 is not production-driven. No observed leak is
  claimed. Add the native tests and actual platform evidence required by scope;
  accurately retain unavailable runtime evidence as unverified.
- The config writer stages backups using Chmod only; the producer expressly
  leaves Windows backup DACLs to profile inheritance. The direct 11.10.2
  owner-only clause governs credential custody, so I do not assert every plain
  configuration backup is a leaked private key. Still, inherited profile ACLs
  are not evidence of owner-only access. Classify the actual backup contents
  and applicable contract rather than claiming this bound proves custody.
- `MatchExcludedFromReplication` and `ExcludedConfigDirName` still have only
  test callers. The outcome supplies a generic future-consumer instruction,
  but no named owning downstream task or corresponding attached API/test
  handoff. Implemented matcher tests are useful and public enrollment remains
  separate; they do not establish exclusion through replication/snapshot/clone/
  Session Directory/diagnostic consumers. Persist the precise ownership handoff
  for unavailable consumers, as CR3 F5 permits, and count row 21 honestly.
- Config label-precision mutation accounting was corrected. Retain that fix.
  Three newly affected hosttrust narrowing plants fail the expected named tests,
  and the applied harmless control survives. These do not cover F1/F2.

## Independent AC accounting

**19 of 21 functional AC rows driven**, of which 16 have bounded passing
candidate evidence and 3 have failing reviewer counterexamples. Rows 15 and 21
are not driven at the required native/consumer entry points. This is the same
21-row functional decomposition used in CR3, not a claim of every normative
clause or every gate covered. A driven row can fail.

| # | Row | Production call / named test | Result |
|---|---|---|---|
|1|Closed Config4|Decode / TestDecodeConfiguration4Refusals|Pass|
|2|Historical compatibility, explicit migration|Migrate / TestMigrateRefusesV4Target, TestMigrateRefusesV4Downgrade|Pass|
|3|Closed trust document|DecodeTrust / TestDecodeTrustRefusals|Pass|
|4|Missing/corrupt trust|ReadSnapshot / TestReadSnapshotMissingStore, TestCorruptTrustRefusedNeverEmpty|Pass, ordinary paths|
|5|Fresh issuance|IssueCredential / TestIssueCredentialProfile|Pass|
|6|Profile/time bounds|VerifyProfile / TestVerifyProfileRefusals, TestVerifyProfileRefusesJustPastExpiry|Pass|
|7|Explicit enrollment|Store.Enroll / TestEnrollRefusals|Pass; OOB verification remains operator act|
|8|Unique mapping|DecodeTrust / TestDecodeTrustRefusals duplicate-root/key cases|Pass|
|9|Rotation/retirement|Rotate, MarkRetiring / TestRotateBoundedWindow, TestMarkRetiringCapsAtLeafExpiry|Pass|
|10|Revocation tombstones|Revoke, Enroll / TestRevokeTombstone, TestRevokedLeafNeverReenrolls|Pass, serial paths|
|11|Stale mutation refusal|WithMutationAuthorization / TestReviewerStaleMutation|Pass, repaired|
|12|Shared authorization serialization|Open/openLock / TestReviewerCrossProcessLockInitialization; WithMutationAuthorization/Revoke / TestReviewerRevocationDuringBoundary and TestReviewerSeparateStoreRevocationDuringBoundary|FAIL first-open; existing same/separate-Store contention passes|
|13|Coherent config/trust readers|LoadCoherent / TestReviewerCrashBetweenConfigAndGeneration, TestReviewerRollbackCrashSurvivingReader|FAIL surviving readers|
|14|Unix custody|Open, ValidateCustody / TestCustodyRefusals, TestValidateCustodyBindsDirectory|Pass on Darwin|
|15|Windows equivalent ACLs|verifyOwner/installOwnerOnlyACL; only TestOwnerOnlyGrants model test|Not native-driven|
|16|Complete peers/local credential|PreviewV4 / TestPreviewV4PeerEnrollment, TestPreviewV4CredentialGates|Pass|
|17|Exact preview/confirmation|ApplyV4 / TestApplyV4PreviewMismatch, TestApplyV4ConfirmRequiredWithDrops|Pass|
|18|Current source/generation apply|ApplyV4 / TestApplyV4StaleSource, TestApplyV4StaleGeneration|Pass; in-lock helper tests additionally present|
|19|Crash-durable pair/recovery|applyV4, rollbackV4, LoadCoherent / reviewer crash tests above|FAIL surviving readers; fresh apply reopen passes|
|20|Explicit rollback|RollbackV4 / TestRollbackV4, TestRollbackV4BackupGates|Pass ordinary/error paths; crash bound in row19|
|21|Secret exclusion across consumers|MatchExcludedFromReplication/ExcludedConfigDirName / helper tests only|Not consumer-driven; precise ownership handoff incomplete|

RPC handshake, actual dispatch, watchdog and operator distribution boundaries
remain unchanged. No transport work is requested from this leaf. Power-loss
persistence remains distinct from the executed process-crash tests.

## Verification, preservation and logs

- Candidate patch SHA256 matches 73975946c2f994f11b3434e7923c689e1eefe9439e94d5d3d0a2cc53fa9bbf28.
  Base signature verifies for oparin@me.com and contains adoption trunk 8cf4aaa.
- RPC backup 15/15 and parked files 11/11 match stored hashes. None of the 49
  candidate paths overlaps the RPC manifest. Preservation outcome is still on
  TASK-260830-z1yxg9. Existing checkpointed SSH/peer bytes are preserved.
- Independently reran immutable hosttrust/config/peeridentity package tests
  `-count=1 -cover`: exit 0, coverage 77.8% / 93.1% / 97.5%.
- Reviewer authorization probes: exit 0, all three named tests PASS. Crash
  probe group: exit 1, two named FAIL and fresh-reopen PASS. Cross-process lock
  initialization: exit 1. Child exit 77 is asserted, not inferred.
- Re-executed neutral, stale-mutation-refresh, joint-marker-stale, recover-diverged
  overlays: neutral exit 0, three plants exit 1 with their expected named
  failures; harness exit 0. Probe-only files were removed from discovery first.
- Producer source manifests match immutable candidate (29 hosttrust, 8 config
  files); retained producer mutation results report 18 hosttrust semantic kills
  plus neutral and 4 config semantic kills plus 1 label kill and neutral.
- Reused rev4 attached build/vet/full-suite/publication evidence instead of
  rerunning unrelated suites. Its bounded validation log shows board issues
  despite an exit-0 board command; that is not proof of a clean board. These
  checks cannot override reproduced failures. No native Windows execution.
- Initial setup mistakes are not evidence: first probe command selected no
  tests in the live tree; first stale probe used an already-issued host and
  failed fixture setup. Both were corrected and superseded by the recorded
  immutable candidate runs. No product files were edited by either attempt.
- Review scratch: .temp/TASK-260909-2ez769/review4. Attached evidence ZIP contains
  probe source, reproducible commands, source/provenance checks, logs and mutant
  results. Required skills loaded; Go skill resolved through main checkout's
  Curator adapter because this worktree lacks adapters. No tools installed.

## Verdict and routing

Changes requested. Route TASK-260909-2ez769 to to-dev; do not accept revision 4.
This is ordinary implementable rework, not Stop-The-Line. Preserve the managed
base, RPC packet and all accepted work. Do not supply commit_ack or integrate.
