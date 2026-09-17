# TASK-260909-2ez769 rework results rev5 — CR4 findings repaired

Producer rework after CR4 changes-requested verdict (RUN-260910-faeff7).
Managed base a89328ca85a613abde39660f4be749a8fcf57903 (v0.6.0 adoption trunk
in place; no base move needed). Worktree left UNCOMMITTED on
task-board/story/STORY-260830-1kiyj6 for handoff snapshot; no producer
commit, no main push. Producer: Muse (muse-spark), 2026-09-16.

RED-first: the exact reviewer probes were ported onto pre-fix source and
reproduced FAIL (exit 1 each, logs in evidence tarball); the committed
tests below keep the same behavior under producer names so a future
reviewer drop-in cannot collide on test/type names.

Port mapping (reviewer probe -> committed test):
- TestReviewerCrashBetweenConfigAndGeneration -> TestSurvivingReaderConvergesInterruptedApply
- TestReviewerRollbackCrashSurvivingReader -> TestSurvivingReaderConvergesInterruptedRollback
- TestReviewerCrashReopenConverges -> behavior kept by fixed TestApplyV4CrashConvergesGeneration
- TestReviewerCrossProcessLockInitialization -> TestFirstOpenPreservesHeldLock
- TestReviewerSeparateStoreRevocationDuringBoundary -> TestMutationAuthorizationSerializesWithSeparateStoreRevocation
- TestReviewerStaleMutation / TestReviewerRevocationDuringBoundary -> kept passing; covered by TestWithMutationAuthorizationRefusesStaleGeneration / TestMutationAuthorizationSerializesWithRevocation

## Preservation (re-verified this run)

- 4 tracked RPC paths CLEAN vs HEAD; no UNRESOLVED_QUESTIONS.md or
  internal/rpcwire remnants in the worktree; none of those paths in the
  candidate diff.
- Control-root backup .temp/TASK-260830-z1yxg9/preserved-before-credentials-260910:
  15/15 files match manifest SHA256.
- Parked delta .temp/TASK-260909-2ez769/parked-rpc-delta-260910: 11/11
  untracked files match the control manifest SHA256.
- Checkpointed SSH/peer bytes untouched: candidate touches only config,
  hosttrust, secprim/census_test.go, LOGBOOK.md and the carried
  peeridentity/v4_interop_test.go; zero sshtransport changes.

## CR4 finding resolutions (production mechanism, not assertions)

F1 (P1, surviving crash readers) — the pending joint-commit marker is now a
general admission barrier (`internal/hosttrust/joint.go`: `markerPresent`,
`convergeLocked`; `transact.go`: `lockForConvergedRead`):
- Exclusive entry points (transact, JointCommit, WithMutationAuthorization,
  Initialize) converge pending intent before reading; JointCommit converges
  leftovers first (forward completion moves the generation so stale markers
  still refuse; refused leftovers fail the commit with the marker kept).
- Shared readers (ReadSnapshot, WithSharedSnapshot, AuthorizeDispatch) take
  the shared fast path only when no marker is present; otherwise they
  release shared, acquire exclusive (never an in-place upgrade), converge
  or refuse, then read. Marker-absent under a shared hold implies a
  coherent pair, since staging a marker needs exclusive.
- Stale-request refusal preserved: bindings compare against the converged
  generation; nothing refreshes a stale number. Recovery itself is the
  single convergence implementation (no duplicated logic).
- Tests: real child-process (exit 77 at the real rename seam)
  TestSurvivingReaderConvergesInterruptedApply (4.0.0/gen+1, marker gone),
  TestSurvivingReaderConvergesInterruptedRollback (3.0.0/gen+1, marker
  gone), TestSurvivingReaderRefusesDivergedConfig (ErrJointIntervened,
  marker kept, Open refuses); hosttrust barrier matrix
  (TestReadSnapshotConvergesInterruptedApply/AbortsUnreplacedCommit/
  RefusesIntervention/RefusesUnreadableMarker,
  TestSharedSnapshotConvergesInterruptedApply,
  TestTransactionConvergesPendingMarker/RefusesIntervention,
  TestJointCommitConvergesLeftoverThenRefusesStale/
  ConvergesAbortedLeftoverThenCommits/RefusesIntervenedLeftover,
  TestInitializeRefusesIntervention,
  TestMutationAuthorizationRefusesStaleAfterConvergence/RefusesIntervention,
  TestAuthorizeDispatchStaleAfterConvergence/RefusesIntervention).
- TestApplyV4CrashConvergesGeneration repaired: every Load/Configuration/
  Open/ReadSnapshot error is now a fatal expectation (no silent early
  return), plus a marker-cleanup assertion.
- Two joint-test setups reworked (assertions unchanged): already-bumped and
  diverged states are now staged without transacting beside a marker,
  because transactions correctly converge first.

F2 (P1, cross-process lock init) — `openLock` (unix + windows) now creates
the lock atomically with O_CREAT|O_EXCL/CREATE_NEW (`FileSystem.CreateExclusive`):
a loser of the creation race opens the existing inode/file instead of
renaming over a live lock. Existing-file verification retries within a
500ms budget (creation-to-secure window); non-regular files refuse
immediately. Windows path applies the same rule; Windows runtime execution
stays unverified (no runner).
- Tests: TestFirstOpenPreservesHeldLock (deterministic barrier port:
  returnedWhileHeld=false, sameInode=true), TestConcurrentOpenAttachesToExistingLock
  (8-way same-inode attach + Open blocks on held exclusive).

F3 (P2, evidence reach) —
- Native Windows tests added (`internal/hosttrust/custody_windows_test.go`,
  7 tests: owner-only acceptance, other-principal refusal, inherited-grant
  refusal, null-DACL refusal, reparse refusal, secured-directory
  inheritance, Open end-to-end) driven through verifyOwnerFile/
  verifyOwnerDir/secureStaged/Open. GOOS=windows vet + test-compile pass
  (7.6MB test binary built, exit 0); runtime execution unverified. Row 15
  counts as implementation present, NOT driven.
- Backup classification: `replaceDurably` documents that backups are
  byte-exact configuration copies (no key material; keys live only under
  host-channel/credentials/), staged 0600 and matched by
  ExcludedConfigDirName; Windows backup DACLs inherit profile ACLs as a
  stated bound, not custody evidence. Test
  TestApplyV4BackupIsClassifiedConfigCopy pins bytes/mode/exclusion and the
  absence of credential files under the config dir.
- Row-21 handoff persisted as outcome TASK-260909-2ez769_exclusion-handoff.md
  (per-surface API + test obligation for 7 surfaces) with board notes on
  all 7 owned tasks (TASK-260830-13bbo0, 1ybn3u, 1kj7ae, 24z2b3, 355og8,
  1ifuz2, 1b162e — notes appended via task-board); the diagnostic-bundle
  surface is explicitly UNOWNED (no board task exists). Row 21 counts as
  implemented-matcher-only, NOT consumer-driven.

## AC coverage: 19 of 21 rows driven through production entry points

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
| 12 | Shared authorization serialization | hosttrust Open/openLock, WithMutationAuthorization+Revoke / TestFirstOpenPreservesHeldLock, TestConcurrentOpenAttachesToExistingLock, TestMutationAuthorizationSerializesWithRevocation, TestMutationAuthorizationSerializesWithSeparateStoreRevocation | pass (was CR4 FAIL) |
| 13 | Coherent config+trust snapshot | config LoadCoherent / TestSurvivingReaderConvergesInterruptedApply, TestSurvivingReaderConvergesInterruptedRollback, TestSurvivingReaderRefusesDivergedConfig | pass (was CR4 FAIL) |
| 14 | Unix owner custody | hosttrust Open, ValidateCustody / TestCustodyRefusals, TestValidateCustodyBindsDirectory | pass on darwin |
| 15 | Windows equivalent ACLs | hosttrust verifyOwner/installOwnerOnlyACL/secureStaged / custody_windows_test.go (7 native tests) | NOT DRIVEN: code + tests present, vet + test-compile pass, runtime unverified (no runner) |
| 16 | Complete peers + selected credential | config PreviewV4 / TestPreviewV4PeerEnrollment, TestPreviewV4CredentialGates | pass |
| 17 | Exact preview + confirmation | config ApplyV4 / TestApplyV4PreviewMismatch, TestApplyV4ConfirmRequiredWithDrops | pass |
| 18 | Current source/generation apply | config applyV4+JointCommit / TestApplyV4StaleSource, TestApplyV4StaleGeneration, revalidate* tests | pass, incl. in-lock revalidation |
| 19 | Crash-durable config/trust pair | config applyV4/rollbackV4 + hosttrust JointCommit/Recover/converge / TestApplyV4CrashConvergesGeneration (fixed), surviving-reader crash tests, barrier matrix | pass (was CR4 FAIL surviving) |
| 20 | Explicit backup rollback | config RollbackV4 / TestRollbackV4*, TestPublishedBackupNamesStayExcluded, TestApplyV4BackupIsClassifiedConfigCopy | pass |
| 21 | Secret exclusion across export surfaces | hosttrust MatchExcludedFromReplication/ExcludedConfigDirName / matcher tests + persisted handoff + 7 owner notes; diagnostics unowned | NOT DRIVEN: matcher enforced, no consumer calls it yet |

Stated bounds (not production-driven here): rows 15 and 21 as above; live
TLS 1.3 launch/handshake/hello/dispatch and stream close-down belong to the
RPC transport task; peer-side distribution of enrollment/rotation/
revocation is operator duty per spec; power-loss durability beyond fsync;
config-dir backup DACLs on Windows (profile ACLs + exclusion).

## Negative + mutation evidence (rerun this run, exact final source)

- hosttrust battery `.temp/TASK-260909-2ez769/mutations-hosttrust-rev5/`:
  25 narrowing mutants KILLED by named tests (incl. new
  dance/transact/initialize/authorize-converge-ignore, marker-unreadable,
  lock-replacing-init with genuine `returnedWhileHeld=true sameInode=false`
  kill, recover-forward-no-bump failing 8 named convergence tests) +
  1 call-site mutant joint-converge-skip KILLED (paired with the
  recover-forward-no-bump narrowing mutant; a converge-ignore variant
  survives here by the staged-marker backstop and is documented, not
  counted) + 1 neutral SURVIVED control (exit 0). Harness exit 0.
  results.json + table.md + manifest.json (31 source digests, head a89328c).
- config battery `.temp/TASK-260909-2ez769/mutations-config-rev5/`:
  4 semantic narrowing KILLED + 1 label-precision KILLED (marked
  LABEL-PRECISION) + 1 neutral passed. Harness exit 0. results.json +
  table.md + manifest.json (32 source digests, head a89328c).
- Every reported exit comes from the real command status
  (subprocess returncode; no pipelines). All 33 plants pre-verified to
  match exactly once; no compile-failure, unapplied or empty-selection
  kills. Mutant sources, overlay.json, standalone test.logs and final
  counts preserved per probe.
- No gate searches source text for a token (all gates operate on decoded
  values/DER/parsed TOML/file bytes/ACEs); the preserve-token
  behavioral-mutant clause is vacuous here, as previously stated.

## Gates rerun this run (all exit 0)

gofmt clean (internal/); `go vet ./...`; `GOOS=windows go vet ./...`;
`go build ./...` darwin+linux+windows; `go test ./... -count=1`
(26 ok, 0 FAIL, 1 pre-existing env-gated SKIP TestDumpSweepSites);
`-cover` hosttrust 77.9% config 93.2% peeridentity 97.5% secprim 94.4%;
`-race ./...` 26 ok; tracecheck ok; catalog freshness pin+generate+diff
clean; cigate contracts 6/6 + claims 6/6 + target derivation pass; fuzz
smoke 13/13 with `fuzz: elapsed` each; fixtures secprim 51 pass +
secconftest 59 pass with 0 skips; `git diff --check` clean.
`task-board validate` exits 0 with 201 board-wide issues, 0 referencing
this task (exit 0 alongside listed issues is not proof of a clean board).
No hosted CI used; all gates executed locally.
