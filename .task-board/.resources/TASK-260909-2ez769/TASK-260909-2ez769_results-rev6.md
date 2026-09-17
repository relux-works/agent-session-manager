# TASK-260909-2ez769 rework results rev6 — CR5 finding repaired as a class

Producer rework after CR5 changes-requested verdict (RUN-260916-f8925e,
Astra low). Managed base `a89328ca85a613abde39660f4be749a8fcf57903`
(v0.6.0 adoption trunk in place; no base move needed). Worktree left
UNCOMMITTED on `task-board/story/STORY-260830-1kiyj6` for handoff
snapshot; no producer commit, no main push. Producer: Muse
(muse-spark), 2026-09-16. This brief governs; superseded briefs are
scope context only.

RED-first: the exact CR5 probe `TestReviewerRestoreCannotOverwriteConvergedCommit`
was copied onto pre-fix source and reproduced FAIL (exit 1,
`unlocked recovery replaced admitted Config4 with 3.0.0 without a
generation change: before=4 after=4`, log in evidence tarball). The 7
original reviewer probes passed pre-fix (baseline log) and re-pass on
the final source (final log).

## Preservation (re-verified this run)

- Worktree implementation bytes verified identical to CR5 candidate
  tree `a823eb20` on all 53 patch paths (scratch
  `git archive a89328c` + rev5 patch application + per-path `cmp`)
  before any edit.
- 4 tracked RPC paths CLEAN vs HEAD; no `UNRESOLVED_QUESTIONS.md` or
  `internal/rpcwire` remnants; none of those paths in the candidate
  diff.
- Control-root backup
  `.temp/TASK-260830-z1yxg9/preserved-before-credentials-260910`:
  15/15 files match manifest SHA256 (recomputed).
- Parked delta `.temp/TASK-260909-2ez769/parked-rpc-delta-260910`:
  11/11 untracked files match the control manifest SHA256.
- Checkpointed SSH/peer bytes untouched: rev6 touches only
  `internal/hosttrust/{joint,errors}.go`,
  `internal/config/{migration_v4_apply,migration_v4_helpers}.go`,
  test files, the two mutation harnesses, and `LOGBOOK.md` (manifest
  diff vs rev5 names exactly these files plus the 2 new test files).
  Zero `sshtransport` changes; `peeridentity` delta still only
  `v4_interop_test.go`.

## CR5 F1 resolution (production mechanism, not assertions)

`JointCommit` now takes the restore closure and compensates a failed
generation bump under the SAME exclusive hold (`compensateLocked` in
`internal/hosttrust/joint.go`): the marker is re-read
(custody-verified, closed decode) and compared pin-for-pin with the
staged intent; the configuration hash must equal the replacement pin
(source-present skips the restore; anything else refuses with the
marker kept); the restore runs; the restored bytes are re-read and
must equal the source pin before the marker aborts. Committed trust
is deliberately never re-read for the decision (the fault that broke
the bump must not break the abort; an apparently durable bump is
unproven exactly because its commit reported failure). The unlocked
`restoreSourceAfterJointFailure` is deleted; `ErrJointReplacementDurable`
now reports the clean abort (original bump refusal preserved) and the
new `ErrJointCompensationFailed` reports compensation that cannot
complete (restore failed, pins moved, marker unreadable), with the
marker kept for operator resolution and the replacement left for
forward recovery. `applyV4`/`rollbackV4` map compensation failure to
the existing `ErrMigrationSync`/`ErrMigrationRecovery` compound;
signatures of the production entries are unchanged.

Post-fix, the original probe shape cannot fail behaviorally: its
in-seam convergence attempt blocks in `flock LOCK_SH` on the held
exclusive lock (45s-timeout goroutine dump in evidence shows
`compensateLocked` → restore closure → `writeTempReplace` → seam →
`LoadCoherent` → `lockForConvergedRead` → `Flock`). The port
`TestApplyV4CompensationCannotOverwriteConvergedCommit` keeps the
reviewer's final-state assertion with the race staged as a goroutine
against the gated restore.

## Write-path census + gate

`TASK-260909-2ez769_write-path-census-rev6.md` (attached outcome)
enumerates all 12 write paths over `config.toml`, backups,
`trust.json`, the marker, the lock file and credential files, each
with production call site, required lock state, intent revalidation,
positive test, interleaving negative and narrowing plant. Notable
rows: legacy `migrate` (no lock, fail-closed, pinned by the new
`TestLegacyMigrateBesideMarkerRefusesConvergence`), credential files
(no lock, inert-until-admitted with unique fresh-key directories),
lock creation (atomic `O_CREAT|O_EXCL`/`CREATE_NEW`, no lock held),
and the sabotage remainder (vanished marker + diverged config
re-stages the marker so restart recovery resolves instead of
admitting).

`TestWritePathCensusGate` (committed,
`internal/config/writepath_census_test.go`) statically fails any
pair mutation outside the censused rows (JointCommit-closure rule
for the config helpers, lock-held allowlists for trust/marker
helpers, custody-site allowlists, raw-rename/remove/create
allowlists); zero violations on this source.
`TestWritePathGateFlagsUnlockedCaller` is the control plant (3
rogue shapes flagged, censused shapes pass).

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
| 12 | Shared authorization serialization | hosttrust Open/openLock, WithMutationAuthorization+Revoke / TestFirstOpenPreservesHeldLock, TestConcurrentOpenAttachesToExistingLock, TestMutationAuthorizationSerializesWithRevocation, TestMutationAuthorizationSerializesWithSeparateStoreRevocation | pass |
| 13 | Coherent config+trust snapshot | config LoadCoherent / TestSurvivingReaderConvergesInterruptedApply, TestSurvivingReaderConvergesInterruptedRollback, TestSurvivingReaderRefusesDivergedConfig, TestApplyV4CompensationCannotOverwriteConvergedCommit, TestRollbackV4CompensationCannotOverwriteConvergedCommit | pass (was CR5 FAIL) |
| 14 | Unix owner custody | hosttrust Open, ValidateCustody / TestCustodyRefusals, TestValidateCustodyBindsDirectory | pass on darwin |
| 15 | Windows equivalent ACLs | hosttrust verifyOwner/installOwnerOnlyACL/secureStaged / custody_windows_test.go (7 native tests) | NOT DRIVEN: code + tests present, vet + test-compile pass, runtime unverified (no runner) |
| 16 | Complete peers + selected credential | config PreviewV4 / TestPreviewV4PeerEnrollment, TestPreviewV4CredentialGates | pass |
| 17 | Exact preview + confirmation | config ApplyV4 / TestApplyV4PreviewMismatch, TestApplyV4ConfirmRequiredWithDrops | pass |
| 18 | Current source/generation apply | config applyV4+JointCommit / TestApplyV4StaleSource, TestApplyV4StaleGeneration, revalidate* tests | pass, incl. in-lock revalidation |
| 19 | Crash-durable config/trust pair | config applyV4/rollbackV4 + hosttrust JointCommit/compensateLocked/Recover/converge / TestApplyV4CrashConvergesGeneration (fixed), surviving-reader crash tests, barrier matrix, 6 compensation race tests, 2 entry clean-abort tests | pass (was CR5 FAIL unlocked error recovery) |
| 20 | Explicit backup rollback | config RollbackV4 / TestRollbackV4*, TestPublishedBackupNamesStayExcluded, TestApplyV4BackupIsClassifiedConfigCopy | pass |
| 21 | Secret exclusion across export surfaces | hosttrust MatchExcludedFromReplication/ExcludedConfigDirName / matcher tests + persisted handoff + 7 owner notes; diagnostics unowned | NOT DRIVEN: matcher enforced, no consumer calls it yet |

Stated bounds (not production-driven here): rows 15 and 21 as above,
retained exactly as CR5 accounted them; live TLS 1.3
launch/handshake/hello/dispatch and stream close-down belong to the
RPC transport task; peer-side distribution of enrollment/rotation/
revocation is operator duty per spec; power-loss durability beyond
fsync; config-dir backup DACLs on Windows (profile ACLs +
exclusion); `verifyLockFile` 500ms retry rechecks custody without
pinning an inode (no claim against external owner replacement of
lock files); the spurious-bump corner aborts fail-safe.

## Negative + mutation evidence (rerun this run, exact final source)

- hosttrust battery `.temp/TASK-260909-2ez769/mutations-hosttrust-rev6/`:
  29 narrowing mutants KILLED by named tests (incl. new
  compensate-marker-skip, compensate-restore-verify-skip,
  compensate-restore-ignore, compensate-verify-before-restore) + 1
  call-site mutant joint-converge-skip KILLED (paired with
  recover-forward-no-bump, as in rev5) + 1 neutral SURVIVED control
  (exit 0). Harness exit 0. results.json + table.md + manifest.json
  (31 source digests, head a89328c).
- config battery `.temp/TASK-260909-2ez769/mutations-config-rev6/`:
  6 semantic narrowing KILLED (incl. new compensate-source-swap,
  compensate-rollback-source-swap) + 1 label-precision KILLED
  (marked LABEL-PRECISION) + 1 neutral passed. Harness exit 0.
  results.json + table.md + manifest.json (34 source digests, head
  a89328c).
- Every reported exit comes from the real command status
  (subprocess returncode; no pipelines). All 39 plants pre-verified
  to match exactly once; no compile-failure, unapplied or
  empty-selection kills. Mutant sources, overlay.json, standalone
  test.logs and final counts preserved per probe.
- `compensate-verify-before-restore` is the preserve-token
  behavioral mutant for the static gate: it reorders
  verify-before-restore while keeping every call, so the call-site
  gate still passes and only the behavioral suite fails (killed by
  `TestJointCommitCompensatesBumpFailure` among others; the
  harnesses run the full behavioral suites). No other gate searches
  source text for a token (all gates operate on decoded
  values/DER/parsed TOML/file bytes/ACEs).
- In-test finding: the new boundary race tests initially deadlocked
  by reading trust on the main goroutine after the replacement seam
  (lock held by the faulted op); the request now binds pre-seam with
  an in-code comment. Final runs are deadlock-free, including under
  `-race`.

## Gates rerun this run (all exit 0)

gofmt clean (413 files scanned); `go vet ./...`;
`GOOS=windows go vet ./...`; `go build ./...`
darwin+linux+windows; `go test ./... -count=1` (26 ok, 0 FAIL, 1
pre-existing env-gated SKIP TestDumpSweepSites); `-cover`
hosttrust 77.9% config 93.3% peeridentity 97.5% secprim 94.4%;
`-race ./...` 26 ok with no `DATA RACE`; tracecheck ok
(contracts=63 sections=36 cases=101); catalog freshness pin +
generate + clean diff (no collateral); cigate contracts 6/6 +
claims 6/6 + target derivation 2/2; fuzz smoke 13/13 with `fuzz:
elapsed` each; fixtures secprim 51 pass + secconftest 59 pass with
0 skips; `git diff --check` clean. `task-board validate` exits 0
with 201 board-wide issues, 0 referencing this task (exit 0
alongside listed issues is not proof of a clean board). 7 original
reviewer probes re-pass on the final source. No hosted CI used;
all gates executed locally.
