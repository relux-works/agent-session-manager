# Write-path census — config-plus-trust pair (TASK-260909-2ez769 rev6)

Producer: Muse (muse-spark), 2026-09-16. Managed base
`a89328ca85a613abde39660f4be749a8fcf57903`. This census closes the
CR3/CR4/CR5 crash/coherence family at the class level: every production
code path that creates, replaces, removes or renames `config.toml`, its
backups, `trust.json`, the pending joint-commit marker, the lock file or
the credential files is enumerated with the lock state it requires, the
intent revalidation it performs, its positive test, its interleaving
negative, and its narrowing plant.

Enforcement: `internal/config/writepath_census_test.go`
(`TestWritePathCensusGate`) statically fails when a pair mutation
appears outside these rows; `TestWritePathGateFlagsUnlockedCaller` is
the control plant proving the gate sees a new unlocked caller. The gate
runs in the normal `go test ./...` suite.

Conventions: "exclusive" is the host-channel authorization lock
(`fileLock.exclusive`), held across the whole listed sequence; "shared
fast path" is `lockForConvergedRead` (shared hold, valid only when no
marker is present, since staging a marker needs exclusive).

## Rows

### 1. Apply replacement — `replaceDurably` via `JointCommit`

- Production call site: `applyV4` (`internal/config/migration_v4_apply.go`)
  passes the replace closure to `store.JointCommit`; the closure calls
  `replaceDurably(filesystem, filename, backup, preview.SourceDocument,
  preview.Replacement)`.
- Required lock state: exclusive, held by `JointCommit` across leftover
  convergence, in-lock revalidation, marker staging, replace, bump and
  marker removal.
- Intent revalidation before writing: `revalidateApplyV4` inside the
  hold (current source bytes, source version, rendered preview bytes,
  generation, credential custody) plus the marker backstop
  (`marker.SourceGeneration == committed.Generation`) and the closed
  marker check.
- Positive test: `TestApplyV4HappyPath` (exact installed bytes, backup,
  generation+1).
- Interleaving negative: `TestLoadCoherentSerializesWithJointCommit`
  (coherent reader waits out a held joint commit);
  `TestRevalidateApplyV4StaleSource/StaleGeneration/PreviewMismatch`
  (concurrent change between fast-fail checks and the hold refuses).
- Narrowing plant: `preview-length` (same-length tampered preview
  admitted at the in-lock site; killed by
  `TestRevalidateApplyV4PreviewMismatch`), `apply-gen-direction`
  (newer generation admitted; killed by
  `TestRevalidateApplyV4StaleGeneration`).

### 2. Rollback replacement — `replaceDurably` via `JointCommit`

- Production call site: `rollbackV4` passes the replace closure to
  `store.JointCommit`; the closure calls `replaceDurably(filesystem,
  filename, preRollback, snapshot.Document(), backupBytes)`.
- Required lock state: exclusive, same hold as row 1.
- Intent revalidation: `revalidateRollbackV4` inside the hold (current
  document still the pinned source and still differs from the backup,
  backup still decodes below v4, generation pinned); an already-holding
  target returns `errRollbackConverged` before any marker is staged.
- Positive test: `TestRollbackV4` (restored bytes, generation+1,
  second rollback is a no-op).
- Interleaving negative: rollback stale-source/generation revalidation
  tests; the row-1 serialization shape covers the shared hold.
- Narrowing plant: shared with row 1 at the helper level
  (`replaceDurably` is only reachable through these two closures plus
  row 10); rollback-closure-specific:
  `compensate-rollback-source-swap` pins the restore closure (row 3).

### 3. In-hold compensation restore — `writeTempReplace` via `JointCommit`

- Production call site: `applyV4`/`rollbackV4` pass the restore closure
  (`writeTempReplace(filesystem, filename, source)`) to
  `store.JointCommit`; `compensateLocked`
  (`internal/hosttrust/joint.go`) runs it after a failed bump without
  releasing the hold. This is the CR5 F1 repair: the previous
  `restoreSourceAfterJointFailure` ran unlocked after `JointCommit`
  returned and is deleted.
- Required lock state: exclusive, same continuous hold as the replace
  and the failed bump. No surviving reader, writer or held boundary can
  converge the replacement between the failure and the restore.
- Intent revalidation before writing: the marker is re-read
  (custody-verified, closed decode) and compared pin-for-pin with the
  staged intent; the configuration hash must equal the replacement pin
  (source-present skips the restore; anything else refuses with the
  marker kept); after the restore the bytes are re-read and must equal
  the source pin before the marker aborts. Committed trust is
  deliberately never re-read: the fault that broke the bump must not
  break the abort, and an apparently durable bump is unproven exactly
  because its commit reported failure.
- Positive tests: `TestJointCommitCompensatesBumpFailure` (hosttrust),
  `TestApplyV4CompensatesBumpFailure` /
  `TestRollbackV4CompensatesBumpFailure` (production entries; clean
  abort, original bump refusal preserved).
- Interleaving negatives (deterministic, gated-restore rendezvous):
  `TestApplyV4CompensationCannotOverwriteConvergedCommit` (ported CR5
  probe; reader blocks, then observes the aborted pair),
  `TestRollbackV4CompensationCannotOverwriteConvergedCommit`,
  `TestApplyV4CompensationSerializesWithWriter` /
  `TestRollbackV4CompensationSerializesWithWriter` (revocation commits
  after the abort),
  `TestApplyV4CompensationSerializesWithBoundary` /
  `TestRollbackV4CompensationSerializesWithBoundary` (boundary admitted
  after the abort with the binding still current).
- Narrowing plants: `compensate-marker-skip` (admits markers differing
  only in non-generation pins; killed by
  `TestJointCommitCompensationRefusesTamperedMarker`),
  `compensate-restore-verify-skip` (admits restores leaving the
  replacement; killed by
  `TestJointCommitCompensationVerifiesRestore/noop_restore`),
  `compensate-restore-ignore` (ignores a failed restore; killed by
  `TestJointCommitCompensationFailsAfterDurableRestore`),
  `compensate-source-swap` / `compensate-rollback-source-swap`
  (restore the wrong bytes; killed by the `*CompensatesBumpFailure`
  entry tests). Preserve-token order swap
  `compensate-verify-before-restore` keeps every call and reorders
  verify-before-restore: the static gate still passes and only the
  behavioral suite fails (killed by
  `TestJointCommitCompensatesBumpFailure` among others).

### 4. Trust bump — `transactLocked` via `commitDocument`

- Production call site: `transactLocked`
  (`internal/hosttrust/transact.go`) encodes generation+1 and calls
  `commitDocument` (stage, fsync, 0600, ACL install, atomic rename,
  directory fsync).
- Required lock state: exclusive. Direct callers: `transact` (after
  `convergeLocked`), `JointCommit` (bump step),
  `recoverLocked`/`convergeLocked` (forward completion).
- Intent revalidation: generation ceiling (`MaxGeneration`), no caller
  generation change (`next.Generation == committed.Generation`), plus
  the caller-specific entry rules below.
- Positive tests: joint happy path, lifecycle tests, recovery tests.
- Interleaving negatives: `TestSharedSnapshotSerializesWithExclusive`,
  `TestMutationAuthorizationSerializesWithRevocation` (+ separate-store
  twin), `TestLoadCoherentSerializesWithJointCommit`.
- Narrowing plants: `exhaustion` (admits the ceiling generation;
  killed by `TestGenerationExhaustionRefusesMutation`),
  `recover-forward-no-bump` (completes forward without the bump;
  killed by `TestReadSnapshotConvergesInterruptedApply`).

### 5. Marker stage and removal — `commitDocument` / `removeMarkerLocked`

- Production call sites: `JointCommit` stages the marker after the
  backstop and removes it on replace failure and after the durable
  bump; `recoverLocked` removes it on forward completion and abort;
  `compensateLocked` removes it on clean abort and re-stages it in the
  sabotage remainder (absent marker plus diverged configuration: the
  in-memory intent still pins the decision, and the re-staged marker
  lets restart recovery resolve the pair instead of admitting it).
- Required lock state: exclusive in all three functions.
- Intent revalidation: closed marker encode/decode (schema, operation,
  absolute config path, 64-hex pins, pins differ, uint53 generation);
  `JointCommit` refuses when a marker is already present after
  leftover convergence (`ErrJointIntervened`).
- Positive tests: `TestJointCommitHappyPath`,
  `TestRecoverCompletesForwardAfterCrash`,
  `TestRecoverAbortsWhenReplacementAbsent`.
- Interleaving negatives: `TestJointCommitConvergesLeftoverThenRefusesStale`,
  `TestJointCommitConvergesAbortedLeftoverThenCommits`,
  `TestJointCommitRefusesIntervenedLeftover`.
- Narrowing plants: `joint-marker-stale` (admits older pinned
  generations; killed by `TestJointCommitRefusesStaleMarkerGeneration`),
  `joint-converge-skip` (call-site removal paired with
  `recover-forward-no-bump`; killed by
  `TestJointCommitConvergesAbortedLeftoverThenCommits`),
  `marker-unreadable` (treats an unreadable marker as absent; killed by
  `TestReadSnapshotRefusesUnreadableMarker`).

### 6. Recovery and converged reads — `Recover` / `convergeLocked`

- Production call sites: `Store.Recover` (also automatic in `Open`),
  `convergeLocked` from every exclusive entry point
  (`JointCommit`, `transact`, `Initialize`,
  `WithMutationAuthorization`, `AuthorizeDispatch`) and from
  `lockForConvergedRead` (release-shared/acquire-exclusive upgrade
  when a marker is present, then converge under the exclusive hold).
- Required lock state: exclusive for convergence; the shared fast path
  applies only when no marker is present.
- Intent revalidation: configuration hash selects forward completion
  (replacement hash; source generation bumps, already-bumped only
  removes the marker, anything else refuses), abort (source hash) or
  refusal with the marker kept (anything else, or unreadable state).
- Positive tests: `TestRecoverCompletesForwardAfterCrash`,
  `TestOpenConvergesInterruptedCommit`,
  `TestRecoverConvergesAlreadyBumped`.
- Interleaving negatives: `TestSurvivingReaderConvergesInterruptedApply`
  / `...Rollback` (real child exit 77 at the rename seam),
  `TestSurvivingReaderRefusesDivergedConfig`,
  `TestApplyV4CrashConvergesGeneration`,
  `TestReadSnapshotConvergesInterruptedApply` /
  `TestTransactionConvergesPendingMarker` /
  `TestMutationAuthorizationRefusesStaleAfterConvergence` (barrier
  matrix), `TestReadSnapshotRefusesIntervention`.
- Narrowing plants: `recover-diverged` (completes forward under a
  diverged generation; killed by
  `TestRecoverRefusesDivergedGeneration`), `dance-converge-ignore` /
  `transact-converge-ignore` / `initialize-converge-ignore` /
  `authorize-converge-ignore` (each admits refused recovery into one
  entry class; killed by the matching `*RefusesIntervention` test).

### 7. Store setup — `Initialize`

- Production call site: `Store.Initialize` converges leftover intent,
  refuses when `trust.json` exists, encodes generation 1 and calls
  `commitDocument`.
- Required lock state: exclusive.
- Intent revalidation: leftover convergence first (setup never builds
  beside an unresolved marker); existing trust refuses (setup never
  overwrites committed authority).
- Positive test: `TestOpenCreatesOwnerOnlyLayout` and lifecycle setup.
- Interleaving negative: `TestInitializeRefusesIntervention`.
- Narrowing plant: `initialize-converge-ignore` (admits refused
  recovery into setup; killed by
  `TestInitializeRefusesIntervention`).

### 8. Enrollment, rotation, revocation, retirement — `transact`

- Production call sites: `Store.Enroll`, `Store.Rotate`,
  `Store.MarkRetiring`, `Store.Revoke` (and the admitting half of
  `Store.Issue`) mutate through `store.transact`, which converges
  leftover intent and commits via `transactLocked` under the exclusive
  hold.
- Required lock state: exclusive.
- Intent revalidation: per-entry rules — authorized-tuple reproduction
  and profile checks (`Enroll`), single-active-predecessor plus
  24h/leaf-expiry bounds (`Rotate`, `MarkRetiring`), existing
  non-revoked entry (`Revoke`), cross-entry uniqueness over live and
  tombstone entries (`admitEntry`, `checkMappingUnique`), digest
  reproduction (`decodeEntry`).
- Positive tests: `TestEnrollBoundsPerHost`, `TestRotateBoundedWindow`,
  `TestRevokeTombstone`, `TestMarkRetiringCapsAtLeafExpiry`, issue
  profile tests.
- Interleaving negatives: same/separate-store revocation contention
  tests (CR3 repairs, still green), stale-mutation refusal
  (`TestWithMutationAuthorizationRefusesStaleGeneration`).
- Narrowing plants: `per-host-bound`, `rotation-expiry`,
  `revoke-state`, `retire-bound`, `retire-expiry`,
  `stale-mutation-refresh`, `expiry-grace`, `digest-root-swap`,
  `unique-root-pair`, `custody-binding` (each killed by its named
  lifecycle/profile test).

### 9. Credential files — `writeCustodyFile` and uncommitted cleanup

- Production call sites: `Store.Issue` and `Store.Rotate` create the
  credential directory (`ensureOwnerDir`), install
  `certificate.pem`/`private-key.pem`/`root.pem` via
  `writeCustodyFile` (0600 staging, ACL install before key material is
  written, fsync, atomic rename, owner verification), then run the
  admitting `transact`. On any failure the deferred cleanup removes
  only that operation's own uncommitted directory.
- Required lock state: none — the exception row. Custody files are
  inert until the admitting trust commit: a crash can only leave
  files without authority, never authority without verifiable
  custody. Directory names are fresh-key digests, so concurrent
  issuers never share a directory, and cleanup only removes the
  directory the failed operation created (no other operation can
  reference its unknown digests).
- Intent revalidation: use-time `ValidateCustody` (owner-only regular
  files, PEM shape, private/public match, profile, host binding,
  contained-leaf-equals-directory-name) before every use
  (`revalidateApplyV4`, dispatch authorization).
- Positive tests: `TestIssueSelfEnrollsActive`, rotation happy paths.
- Interleaving negative: not applicable as a pair race (unique inert
  directories); misuse is refused at use time
  (`TestValidateCustodyBindsDirectory`, `TestCustodyRefusals`).
- Narrowing plants: `custody-0644` (admits world-readable files;
  killed by `TestCustodyRefusals/world_readable_trust_file`),
  `custody-binding` (admits root-digest directory names; killed by
  `TestValidateCustodyBindsDirectory`).

### 10. Legacy v1/v2/v3 migration — `replaceDurably` in `migrate`

- Production call site: `migrate`
  (`internal/config/migration.go`) calls `replaceDurably` directly.
  This path predates the joint barrier and holds no authorization
  lock.
- Required lock state: none — the legacy row. It stays fail-closed
  against joint intent in both directions: a joint commit's in-lock
  revalidation refuses a source the legacy path changed
  (`ErrMigrationV4StaleSource`), and a legacy replacement beside an
  outstanding marker diverges from the pins, so every converged
  reader refuses `ErrJointIntervened` instead of admitting the mix.
- Intent revalidation: `Load` validation of the selected source
  before anything durable is written; backup publication before the
  replacement; refusal leaves the directory exactly as found.
- Positive tests: `TestMigrateProductionEntryCreatesOwnerOnlyBackupAndCompleteVersion2`,
  `TestMigrateProductionEntryMapsLegacyTerminalToVersion3AndIsIdempotent`.
- Interleaving negative:
  `TestLegacyMigrateBesideMarkerRefusesConvergence` (legacy v1-to-v2
  replacement beside a staged marker; `LoadCoherent` and a fresh
  `Open` refuse with the marker kept).
- Narrowing plant: `v4-explicit-label` is label precision (marked as
  such); the semantic legacy gates predate this census and keep
  their existing refusal tests.

### 11. Backups and staging files

- Production call sites: backups (`.bak.<version>`,
  `.pre-rollback.<version>`) publish inside `replaceDurably` under
  the same hold as rows 1/2/10 (0600 staging, hard-link publication,
  existing-name byte comparison). Config staging
  (`.ax-config-*`, `.ax-config-restore-*`) and trust staging
  (`.trust-stage-*`, `.custody-stage-*`, `lock-stage-*`) are created
  and renamed only inside the row-1..9 helpers; readers never select
  them and failures remove them.
- Required lock state: as the owning row; leftover staging from a
  crash is inert by name.
- Intent revalidation: backup bytes must equal the pre-migration
  source exactly; classification pinned by
  `TestApplyV4BackupIsClassifiedConfigCopy` (byte-exact config copy,
  no key material, 0600, exclusion match, no credential files under
  the config dir).
- Positive tests: backup assertions in the apply/rollback happy
  paths, `TestPublishedBackupNamesStayExcluded`.
- Interleaving negative: a backup published by a failed replace
  (replace fails after backup publication) is inert and excluded;
  the next attempt reuses it only on byte equality.
- Narrowing plant: covered through the owning rows; backup-name
  matching is pinned by `TestPublishedBackupNamesStayExcluded` and
  the exclusion matcher tests.

### 12. Lock file creation — `CreateExclusive` in `openLock`

- Production call site: `openLock` (unix and windows) creates the
  lock file atomically (`O_CREAT|O_EXCL` / `CREATE_NEW`); a loser of
  the creation race opens the existing inode/file instead of
  replacing a live lock.
- Required lock state: none — creation is the atomic primitive, not
  a pair mutation. No production path renames over the lock file.
- Intent revalidation: `verifyLockFile` (owner-only custody;
  non-regular files refuse immediately; other failures retry within
  the 500ms creation-to-secure window).
- Positive tests: `TestFirstOpenPreservesHeldLock`
  (returnedWhileHeld=false, sameInode=true),
  `TestConcurrentOpenAttachesToExistingLock`.
- Interleaving negative: `TestFirstOpenPreservesHeldLock` is itself
  the deterministic barrier port (child paused after the absence
  observation ends on the same inode and waits out the held lock);
  `TestConcurrentOpenAttachesToExistingLock` covers 8-way same-inode
  attach plus open-blocks-on-held-exclusive.
- Narrowing plant: `lock-replacing-init` (publishes the lock with a
  replacing rename; killed with genuine
  `returnedWhileHeld=true sameInode=false` by
  `TestFirstOpenPreservesHeldLock`).
- STATED BOUND: `verifyLockFile` rechecks current path custody but
  does not pin an inode: an owner replacing the lock file externally
  between retries can pass a later retry. No production initializer
  replaces the inode; no protection against external owner
  replacement is claimed.

## Cross-row notes

- Spurious-bump corner: a bump whose rename landed but whose commit
  still reported failure aborts to (source, source+1) rather than
  completing forward. Forward completion would risk power-loss
  incoherence (unproven rename durability with the marker gone);
  the abort direction only invalidates stale bindings, which refuse
  safely. No test can distinguish this corner from a failed bump,
  and none claims to.
- Power loss beyond fsync is a stated bound everywhere: process
  crashes are covered by stage+fsync+rename+dirsync with real
  child-process tests; disk honesty under power loss is not
  established here.
- Windows runtime execution is unverified (no runner): the DACL
  implementation plus 7 native build-tagged tests are present with
  cross-build/vet evidence only. Config-dir backup DACLs inherit
  user-profile ACLs as a stated bound. Row 15 of the AC matrix
  counts as implementation present, NOT driven.
- The static gate scans non-test sources only: test files stage
  fixtures through direct writes by design. Production can never
  call test helpers (Go excludes `_test.go` from normal builds).
  Direct `ReadFile`/`Lstat`/`Stat` calls are reads, not pair
  mutations, and are intentionally outside the gate; `Chmod`/
  `MkdirAll` are custody setup covered by the custody tests.
- Diagnostic bundle export remains UNOWNED (no board task); the
  row-21 matcher handoff persists on the seven owning tasks.
