# CR revision 3 review — changes requested

Task TASK-260909-2ez769; CR-TASK-260909-2ez769-3 revision 3.
Base a89328ca85a613abde39660f4be749a8fcf57903; candidate tree
881fb127162ee55a0175313850b4704f41cb24e0. Review is read-only against the
managed worktree. Probes and overlays ran in an archive of the immutable tree
under .temp/TASK-260909-2ez769/review3. No commits or product edits.

## Blocking findings

1. **P1: stale mutation authorization is admitted.**
   `internal/hosttrust/authorize.go:191`, `WithMutationAuthorization`, replaces
   the request generation with current.Generation and never compares the
   original binding. `TestReviewerStaleMutation` opens an authorized fixture,
   commits a config authorization generation change, then submits the original
   request. It returns nil and invokes the side-effect callback. Exit 1.
   Section 11.10.3 expressly forbids refreshing a cached number to authorize
   old prepared work. Refuse stale requests before entering the boundary;
   require fresh connection/replanning through the RPC caller contract. Add a
   negative test and a narrowing mutant for this mutation path, not only the
   dispatch generation guard.

2. **P1: revocation succeeds during an authorized mutation boundary.**
   `internal/hosttrust/lock_unix.go:60-73` uses flock on the same open descriptor
   for all goroutines of a Store, without process-local serialization.
   `TestReviewerRevocationDuringBoundary` waits until the production mutation
   callback is held, then calls Revoke on that Store. Revoke returns success
   before the callback is released. Exit 1 on darwin/arm64.
   The existing `TestMutationAuthorizationSerializesWithRevocation` uses an
   immediate nonblocking select after starting the revocation goroutine; it
   does not give that goroutine time to attempt the lock and misses the bug.
   Implement safe local and cross-process locking, including nested reads and
   concurrent lock initialization; test deterministic boundary contention.
   Section 11.10.3 forbids this success ordering.

3. **P1: config replacement and generation commit are not crash-atomic.**
   `internal/config/migration_v4_apply.go:82` installs configuration inside
   `CommitConfigChange`; `internal/hosttrust/transact.go:87` commits trust only
   after the callback. Recovery is an in-process error handler, with no durable
   joint transaction/restart marker. `TestReviewerCrashBetweenConfigAndGeneration`
   drives production apply with an OS filesystem wrapper and terminates the
   child process with exit 77 immediately after the real config rename. On
   restart, production Load accepts Config4 and a freshly opened Store accepts
   the old generation (3). The parent test fails, exit 1. No synthetic trust
   bytes or fake decoder are used. This is a process-crash ordering test, not a
   power-loss filesystem test. Sections 6.6 and 11.10.3 require the coherent
   committed config/trust transaction. Add durable recovery/selection and
   corresponding apply/rollback crash tests. Also move current-preview/source/
   generation revalidation into the shared transaction: presently those checks
   precede lock acquisition, leaving a concurrent-change window. The latter
   window is code-inspection evidence, not a separately executed probe.

4. **P1: native Windows custody is not implemented to its required contract.**
   `internal/hosttrust/custody_windows.go:11-42` requests only owner SID and
   equates it with owner-only access. Ownership does not establish DACL grants;
   no DACL is read or installed. Common permission checks additionally apply
   Unix 0700/0600 tests before this function, without a Windows ACL abstraction.
   Sections 11.10.2 and the explicit native-Windows platform lanes require
   equivalent owner-only ACLs. A successful cross-build and an absent Windows
   runner cannot waive the implementation requirement. Implement and validate
   ACL custody (positive owner-only and negative other-principal access/reparse
   cases). No Windows execution was performed here; actual Windows runtime
   acceptance/refusal remains unverified, not claimed as observed leakage.

5. **P2: no-secret replication and complete negative-coverage claims exceed evidence.**
   `ExcludedFromReplication` has no production consumer; its only caller is
   `TestExcludedFromReplication`. `TestNoSecretReplication` walks the local
   custody directory and asserts file placement; it does not drive replication,
   snapshot, clone, Session Directory or diagnostic export. No actual secret
   leak is claimed, but the mandatory exclusion is not established at those
   entry points. Provide production enforcement/negative tests or an explicit
   scoped ownership handoff for unavailable surfaces; do not count a returned
   path list as end-to-end exclusion. The config `v4-explicit-label` mutant
   changes only the error identity while retaining refusal: its kill proves
   label precision, not a narrowed semantic admission gate. Correct the claim
   of four config narrowing kills to three semantic plants plus one label plant.

## Independently derived AC accounting

**18 of 21 functional AC rows driven**, including three rows with failing
reviewer counterexamples. This is a functional decomposition of sections 6.6,
11.10.2 and 11.10.3 within this leaf, not a claim of all normative clauses or
all gate mutants covered. Fifteen rows have bounded passing test evidence;
three rows below lack the required production proof. The producer's unlisted
21/21 does not establish complete acceptance.

| # | Scoped row | Production call / named test | Result or bound |
|---|---|---|---|
|1|Closed Config4|Decode / TestDecodeConfiguration4Refusals|Driven, passes|
|2|Historical compatibility and explicit migration|Migrate / TestMigrateRefusesV4Target, TestMigrateRefusesV4Downgrade|Driven, passes|
|3|Closed trust document|DecodeTrust / TestDecodeTrustRefusals|Driven, passes|
|4|Missing/corrupt trust refuses|ReadSnapshot / TestReadSnapshotMissingStore, TestCorruptTrustRefusedNeverEmpty|Driven, passes|
|5|Fresh issuance|IssueCredential / TestIssueCredentialProfile|Driven, passes|
|6|Exact profile and time bounds|VerifyProfile / TestVerifyProfileRefusals, TestVerifyProfileRefusesJustPastExpiry|Driven, passes|
|7|Explicit tuple enrollment|Store.Enroll / TestEnrollRefusals|Driven, passes; OOB human verification remains operator act|
|8|Mapping uniqueness|DecodeTrust / TestDecodeTrustRefusals duplicate-root/key rows|Driven, passes|
|9|Rotation/retirement bounds|Rotate, MarkRetiring / TestRotateBoundedWindow, TestMarkRetiringCapsAtLeafExpiry|Driven, passes|
|10|Revocation tombstones|Revoke, Enroll / TestRevokeTombstone, TestRevokedLeafNeverReenrolls|Driven, passes serially|
|11|Old-generation mutation refusal|WithMutationAuthorization / TestReviewerStaleMutation|Driven, FAIL|
|12|Boundary/revoke serialization|WithMutationAuthorization, Revoke / TestReviewerRevocationDuringBoundary|Driven, FAIL|
|13|Coherent config plus trust snapshot|ReadSnapshot returns trust only; caller supplies allowlist|Not established; missing combined production proof|
|14|Unix owner custody|Open, ValidateCustody / TestCustodyRefusals, TestValidateCustodyBindsDirectory|Driven on Darwin, passes|
|15|Windows equivalent ACLs|verifyOwner|Not driven; implementation gap F4|
|16|Complete peers and selected credential|PreviewV4 / TestPreviewV4PeerEnrollment, TestPreviewV4CredentialGates|Driven, passes|
|17|Exact preview and confirmation|ApplyV4 / TestApplyV4PreviewMismatch, TestApplyV4ConfirmRequiredWithDrops|Driven, passes serially|
|18|Current source/generation apply|ApplyV4 / TestApplyV4StaleSource, TestApplyV4StaleGeneration|Driven, passes serially; transaction window remains|
|19|Crash-durable config/trust pair|applyV4, Load, ReadSnapshot / TestReviewerCrashBetweenConfigAndGeneration|Driven, FAIL|
|20|Explicit backup rollback|RollbackV4 / TestRollbackV4, TestRollbackV4BackupGates|Driven, passes normal/error paths; stopping channels belongs to RPC/operator boundary|
|21|Secret exclusion across export surfaces|ExcludedFromReplication, local walk only|Not production-driven across required surfaces|

Live TLS/handshake/dispatch admission and one-second stream closure remain the
RPC task's scope, not a newly invented guarantee of this leaf. Peer distribution
is operator duty. No blanket waiver for custody, generation or crash atomicity.

## Verification and provenance

- Reviewed supplied producer results, retry2, retry3, refresh and CR validation.
- Independently verified base includes adoption trunk 8cf4aaa; base signature
  verifies as oparin@me.com. Candidate archive contains the v0.6.0 adoption.
- Patch SHA256 matches a1ee1e7dd2120ce8bb883d55ad3092bd53c46432e7791ddc2c8bf30334266d81.
- RPC backups 15/15 and parked untracked files 11/11 match manifest hashes;
  none of those paths overlaps the 44-path CR. RPC preservation resource remains
  attached to TASK-260830-z1yxg9. No checkpointed SSH/peer changes were modified.
- Reran immutable config/hosttrust/peeridentity package tests with -count=1
  -cover: exit 0; coverage 93.2%, 77.0%, 97.5%. Initial live-tree run also exit 0;
  acceptance conclusions use the immutable archive run. An intermediate masked
  run passed but is superseded by the complete three-package rerun.
- Accepted whole-repository build/vet/full-suite and publication validation from
  the attached rev3 log rather than rerunning unrelated gates. Those green
  checks do not override the three new negative failures.
- Reran four semantic narrowing overlays: stale-direction, custody-0644,
  preview-length, confirm-drops, each exit 1 with the expected named failing
  test. Two neutral controls pass exit 0. Harnesses exit 0. Probe sources were
  moved out of package test discovery before these control/mutant runs.
- Independently checked producer mutation source SHA256 manifests against the
  immutable archive: 25 hosttrust files and 7 config files match. Retained
  producer results record 15 hosttrust kills, 3 config semantic kills plus
  1 label kill, and two passing controls. This is not every-gate coverage.
- New probes fail exit 1 (hosttrust two tests; config crash test). Child crash
  exit 77 is asserted by its parent. Evidence archive preserves source, logs,
  commands/exit accounting, mutation outputs and provenance verification script.
- Readiness: Go 1.25.5 darwin/arm64, git 2.50.1, Python 3; task-board mutation
  succeeded. Managed worktree lacks local skill adapters; required Curator Go
  skill was read from the main checkout's managed symlink. Initial missing-path
  and unknown resources-query attempts were corrected without installing tools.

## Routing

Request developer rework, status to-dev. No acceptance, commit_ack, integration,
commit or main push. Keep preservation and current managed base intact. Attach
this verdict and the reproduction evidence before ending the reviewer run.
