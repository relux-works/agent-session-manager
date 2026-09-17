# TASK-260909-2ez769 rework results rev7 — CR6 findings repaired at the type level

Producer rework after CR6 changes-requested verdict (RUN-260916-c3e675,
Astra low). Managed base `a89328ca85a613abde39660f4be749a8fcf57903`
(v0.6.0 adoption trunk in place; no base move needed). Worktree left
UNCOMMITTED on `task-board/story/STORY-260830-1kiyj6` for handoff
snapshot; no producer commit, no main push. Producer: Muse
(muse-spark), 2026-09-16. This brief (rework-rev6) governs; superseded
briefs are scope context only.

RED-first: the three CR6 probes were ported onto pre-fix source and
reproduced FAIL (exit 1 each: legacy overwrite of admitted Config4 at
unchanged generation 4; failed apply admitting Config4 at old
generation 3; failed rollback admitting Config3 at old generation 4;
log in evidence tarball). The seven original reviewer probes passed
pre-fix and re-pass on the final source.

## Preservation (re-verified this run)

- Worktree implementation bytes verified identical to CR6 candidate
  tree `b0bb3d9d` on all 55 patch paths (scratch
  `git archive a89328c` + rev6 patch application + per-path `cmp`)
  before any edit. The only additions were the two RED-port test
  files from the cancelled run, reused here.
- 4 tracked RPC paths CLEAN vs HEAD; no `UNRESOLVED_QUESTIONS.md` or
  `internal/rpcwire` remnants; none of those paths in the candidate
  diff.
- Control-root backup
  `.temp/TASK-260830-z1yxg9/preserved-before-credentials-260910`:
  15/15 files match manifest SHA256 (recomputed).
- Parked delta `.temp/TASK-260909-2ez769/parked-rpc-delta-260910`:
  11/11 untracked files match the control manifest SHA256.
- Checkpointed SSH/peer bytes untouched: rev7 touches only
  `internal/hosttrust/{hold,joint,transact,authorize,errors}.go` +
  tests, `internal/config/{migration,migration_v4_apply,
  migration_v4_helpers}.go` + tests, the two mutation harnesses, and
  `LOGBOOK.md`. Zero `sshtransport` changes; `peeridentity` delta
  still only `v4_interop_test.go`.

## CR6 F1 resolution — legacy migrate inside the authorization transaction

`migrate` (`internal/config/migration.go`) now opens the coordinating
store (`openLegacyStore`, which converges interrupted joint intent and
fails closed when the store cannot open) and runs source revalidation
plus the durable replacement inside `store.WithExclusiveHold`
(`internal/hosttrust/hold.go`): the exclusive hold serializes the
legacy writer with joint commits and other legacy migrations, and
`revalidateLegacySource` under the hold refuses `ErrMigrationStaleSource`
when the source moved — including a completed Config4 commit. The rev6
unlocked exemption (census row 10) is gone. Pre-hold refusals
(unknown target, v4-explicit, downgrade, missing source, disclosure
choice, encode failure) return before the store opens and write
nothing. Opening creates the owner-only store skeleton when absent
(documented side effect; required so the bootstrap window closes too).

The repair serializes, so the reviewer's synchronous probe shape (a
nested Migrate+Apply running inside the outer rename) now blocks on
the held lock instead of overwriting: the post-fix log shows the
timeout with the `WithExclusiveHold` stack (see the evidence
tarball) — the CR5 restore-probe precedent for a repair that
introduces blocking. The committed port
`TestLegacyMigrateCannotOverwriteCommittedV4` stages the same race as
a goroutine against the gated rename: the nested migrate serializes
behind the held lock (proven blocked 250ms), then refuses stale
after the outer commits, and fresh joint work commits Config4 with
the coherent reader admitting exactly that pair. Symbol names
deliberately differ from the review probes so the reviewer can drop
the original probe files beside this candidate without a
redeclaration.

## CR6 F2 resolution — replace errors resolve against the pins

`JointCommit` routes every replace-closure error through the new
`resolveFailedReplace` (`internal/hosttrust/joint.go`) under the
same hold instead of unconditionally removing the marker. Only
configuration still equal to the source pin aborts with the marker
removed and the original error (the old behavior, now proven rather
than assumed); a durable replacement, diverged bytes, or unreadable
state runs the common compensation protocol, which restores the
source when it can be proven and otherwise keeps the marker so
readers refuse or converge. Covers apply and rollback, surviving
readers and reopen: `TestApplyV4FailedReplaceKeepsIntent` /
`TestRollbackV4FailedReplaceKeepsIntent` (ported CR6 probes: durable
replacement with a working restore converges to a clean abort),
`TestApplyV4FailedReplaceKeepsMarkerWhenRestoreFails` /
`TestRollbackV4FailedReplaceKeepsMarkerWhenRestoreFails` (restore
also fails: marker kept, reopen converges forward, surviving reader
observes the converged pair), `TestResolveFailedReplaceKeepsMarker`
(hosttrust-level mechanism).

## CR6 F3 resolution — the instrument at the type level

Every helper that creates, replaces, removes or renames the pair
takes a `hosttrust.HeldExclusive` first parameter: an unforgeable
capability only `fileLock.exclusiveHold` constructs, verified live
before every mutation and expired on unlock. An unlocked writer
cannot compile a genuine hold; its zero-token form fails closed at
runtime (`ErrExclusiveHoldRequired`). Threaded through
`commitDocument`, `removeMarkerLocked`, `transactLocked`,
`convergeLocked`, `recoverLocked`, `compensateLocked`,
`resolveFailedReplace`, `replaceDurably`, `writeTempReplace`, and
the `JointCommit` replace/restore closures (which now receive the
live hold); `store.lock.exclusive()` keeps its signature for the
read paths and existing probe compatibility. Exempt-by-proof rows:
credential files (inert until the admitting commit, unique
fresh-key directories, use-time custody) and lock creation (the
atomic primitive the hold comes from); staging (`writeTempFile`)
and ACL install (`secureStaged`) stay statically allowlisted with
the proofs in the census.

The static gate is the second line, now fail-closed: it scans
function declarations (any receiver), package variable
initializers, and nested literals; resolves local aliases
transitively; flags helper escapes as values (assignments,
arguments, returns, fields); restricts the backend exemption to
exact (receiver, method) pairs; and flags every direct
`os.Rename/Remove/RemoveAll/Create/CreateTemp/OpenFile/WriteFile/
MkdirTemp/Link/Symlink/Truncate` outside the backend definitions.
Hold-entry closures cover both `JointCommit` and
`WithExclusiveHold`; the legacy direct-call exemption is removed.
Six executable controls must fail the gate: the four CR6 reviewer
shapes verbatim plus a producer direct-`os.WriteFile` shape and a
producer backend-receiver rogue method. The two compilable shapes
execute against real temporary files and must change bytes; the
helper-calling shapes cannot forge a hold, so their backstop is
the zero-hold runtime refusal asserted in the same test.

## AC coverage: 19 of 21 rows driven through production entry points

| # | Row | Production call / named test | Result |
|---|---|---|---|
| 1 | Closed Config4 | config.Decode / TestDecodeConfiguration4Refusals | pass |
| 2 | Historical compat + explicit migration | config.Migrate/migrate / TestMigrateRefusesV4Target, TestMigrateRefusesV4Downgrade, TestLegacyMigrateCannotOverwriteCommittedV4, TestLegacyMigrateConvergesUnreplacedMarkerThenCommits, TestLegacyMigrateRefusesAfterReplacementLanded | pass (was CR6 FAIL) |
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
| 13 | Coherent config+trust snapshot | config LoadCoherent / surviving-reader crash tests, six compensation races, F1/F2 ports, marker-kept reopen tests | pass (was CR6 FAIL) |
| 14 | Unix owner custody | hosttrust Open, ValidateCustody / TestCustodyRefusals, TestValidateCustodyBindsDirectory | pass on darwin |
| 15 | Windows equivalent ACLs | hosttrust verifyOwner/installOwnerOnlyACL/secureStaged / custody_windows_test.go (7 native tests) | NOT DRIVEN: code + tests present, runtime unverified (no runner) |
| 16 | Complete peers + selected credential | config PreviewV4 / TestPreviewV4PeerEnrollment, TestPreviewV4CredentialGates | pass |
| 17 | Exact preview + confirmation | config ApplyV4 / TestApplyV4PreviewMismatch, TestApplyV4ConfirmRequiredWithDrops | pass |
| 18 | Current source/generation apply | config applyV4+JointCommit / TestApplyV4StaleSource, TestApplyV4StaleGeneration, revalidate* tests | pass, incl. in-lock revalidation |
| 19 | Crash-durable config/trust pair | config applyV4/rollbackV4 + hosttrust JointCommit/resolveFailedReplace/compensateLocked/Recover/converge / TestApplyV4CrashConvergesGeneration (fixed), surviving-reader crash tests, barrier matrix, 6 compensation race tests, F2 ports + marker-kept tests | pass (was CR6 FAIL) |
| 20 | Explicit backup rollback | config RollbackV4 / TestRollbackV4*, TestPublishedBackupNamesStayExcluded, TestApplyV4BackupIsClassifiedConfigCopy, TestRollbackV4FailedReplaceKeepsIntent, TestRollbackV4FailedReplaceKeepsMarkerWhenRestoreFails | pass (was CR6 FAIL) |
| 21 | Secret exclusion across export surfaces | hosttrust MatchExcludedFromReplication/ExcludedConfigDirName / matcher tests + persisted handoff + owner notes; diagnostics unowned | NOT DRIVEN: matcher enforced, no consumer calls it yet |

Stated bounds (not production-driven here): rows 15 and 21 as above,
retained exactly as CR5/CR6 accounted them; live TLS 1.3
launch/handshake/hello/dispatch and stream close-down belong to the
RPC transport task; peer-side distribution of enrollment/rotation/
revocation is operator duty per spec; power-loss durability beyond
fsync; config-dir backup DACLs on Windows (profile ACLs +
exclusion); `verifyLockFile` 500ms retry rechecks custody without
pinning an inode (no claim against external owner replacement of
lock files); legacy migration creates the owner-only store skeleton
when absent and fails closed when the store cannot open.

## Negative + mutation evidence (rerun this run, exact final source)

- hosttrust battery `.temp/TASK-260909-2ez769/mutations-hosttrust-rev7/`:
  32 narrowing mutants KILLED by named tests (29 retained incl. the
  hold-threaded converge/restore re-anchors, plus new
  `hold-require-skip`, `hold-no-lock`, `resolve-source-skip`) + 1
  call-site mutant `joint-converge-skip` KILLED (paired with
  `recover-forward-no-bump`, as in rev5/rev6) + 1 neutral SURVIVED
  control (exit 0). Harness exit 0. results.json + table.md +
  manifest.json (34 source digests, head a89328c).
- config battery `.temp/TASK-260909-2ez769/mutations-config-rev7/`:
  9 semantic narrowing KILLED (6 retained incl. the hold-threaded
  restore swaps, plus new `replace-require-skip`,
  `legacy-revalidate-skip`, `replace-restore-skip`) + 1
  label-precision KILLED (marked LABEL-PRECISION) + 1 neutral passed.
  Harness exit 0. results.json + table.md + manifest.json (39 source
  digests, head a89328c).
- Every reported exit comes from the real command status
  (subprocess returncode; no pipelines). All 43 plants pre-verified
  to match exactly once; no compile-failure, unapplied or
  empty-selection kills. Mutant sources, overlay.json, standalone
  test.logs and final counts preserved per probe.
- New plants this run: `hold-require-skip`, `hold-no-lock`,
  `resolve-source-skip` (hosttrust); `replace-require-skip`,
  `legacy-revalidate-skip`, `replace-restore-skip` (config). Each
  admits exactly one member of its rejected class (zero token but
  not released token; unserialized but tokened writes; durable
  replacement as intact; zero token on the config side;
  same-version byte changes with the version check kept;
  unrestored post-rename sync failures), with the kept half pinned
  by a still-passing test verified in the plant logs.
- `compensate-verify-before-restore` remains the preserve-token
  behavioral mutant for the static gate: it reorders
  verify-before-restore while keeping every call, so the call-site
  gate still passes and only the behavioral suite fails. No other
  gate searches source text for a token.

## Gates rerun this run (all exit 0)

gofmt exact-check clean; `go vet ./...`; `GOOS=windows go vet ./...`;
`go build ./...` darwin+linux+windows amd64; `go test ./... -count=1`
(26 ok, 0 FAIL); `-race ./...` all ok with no `DATA RACE`;
`-cover ./...` all ok (hosttrust 78.0% config 93.5% peeridentity
97.5% secprim 94.4%); 343 top-level PASS / 0 FAIL across the four
touched packages; fuzz smoke 13/13 with `fuzz: elapsed` each;
tracecheck ok (contracts=63 sections=36 cases=101); catalog v0.6.0
-check ok with no drift; JSON check ok; `git diff --check` clean.
`task-board validate` exits 0 with 201 board-wide issues, 0
referencing this task (exit 0 alongside listed issues is not proof
of a clean board). Exact reviewer probes on the final source: 7
original + 2 F2 sync probes PASS; the CR5 restore probe blocks on
flock (same bounded timeout shape as rev6); the CR6 F1 sync probe
blocks on the held lock (serialization proof; the committed async
port pins the final state); rev6 token-less census plants no longer
compile (vet type error captured). No hosted CI used; all gates
executed locally with a private GOCACHE after external build-cache
interference flaked two battery runs (different plants each run;
serial re-runs killed; final batteries ran uninterrupted).
