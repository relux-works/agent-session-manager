# Review verdict — TASK-260830-2f5393 rev1 (implement lease record validation and CAS)

## Verdict: ACCEPT revision 1

Change Request `CR-TASK-260830-2f5393-1` revision 1 is accepted for
integration. One P2 finding (stale ownership-registry gap row, §6 below)
is recorded with an explicit story-level requirement; it does not
invalidate this leaf. No P1 deviation was found. The candidate was never
mutated: every probe ran in isolated copies under `/tmp/review-2f5393/`
or as read-only `go test`/`go vet` on the live tree. No commit,
checkpoint, or branch operation was performed.

- Candidate source integrity: `sha256 internal/sessrepo/lease_store.go =
  bdb89cb607268d1d06ac2eb235f4e1a22017d3da61899203be26d15153cac6c8`
  (equals the battery-time digest), `testdata/mutate.py =
  a9d795ce50ae4d17c746a9c66bc9256a37902b82f0a76d78a3e8d2ee07290e5c`.
- AC coverage: **8 of 8 AC rows driven** through the named production
  entries (`Repository.CreateLease`, `Repository.CompareAndSwapLease`,
  `Repository.VerifyFencingToken`, `CheckFencingExpiry`,
  `Repository.WinningLease`/`ListLeases`/`GetLease`, `AttestLeaseRecord`);
  public-CLI coverage 0 of 8 by stated bound (no `ax` command exists).

## 1. Spec fidelity

- Pinned text (§5.3): "There is no time-expiring ownership lease in
  v0.5.0. Liveness is not authority." `lease_expired` occurs **0 times**
  in `internal/specdoc/SPEC.v0.6.0.md`, and the §15.3 taxonomy names
  `lease_conflict` (exit 10) but no expired-lease class. The
  grant-expiry-only design (`fencing_grant_expired`, lease never
  retires) is faithful; `TestLeaseNeverExpiresWithAge` pins it. A new
  operational code is permitted ("New error codes MAY be added in a
  compatible minor contract version"). Not a deviation.
- Task scope cites v0.5.0 numbering; the retained v0.6.0 headings (§2.2,
  §5.3, §13.6–13.10) exist verbatim in the pinned document. The only
  numbering difference observed is the retained sentence above, which
  still says "v0.5.0" inside the v0.6.0 text.
- `internal/sessquery/lease.go` sha256 is identical trunk-vs-candidate
  (`3ce19568…ec2f`); `winningLeaseFor` is byte-unchanged. The adoption
  test (`TestLeaseStoreRecordsAdmitWithoutBehavioralChange`) proves
  store-minted records admit there with equal digest/epoch/lease/holder.
- `CompareLeaseTuple` restates `sessstate.Compare` line-for-line for the
  comparison logic. The agreement corpus is **6 example pairs, not an
  exhaustive alphabet** (`lease_store_adopt_test.go:66-73`). Planting a
  reversed tie-break on the sessrepo side goes RED
  (`attacks/A4-compare-divergence.log`, exit 1).

## 2. CAS, epochs, holder, fencing (10/10 reviewer probes green)

Probes in `probes/reviewer_probe_test.go`, log
`probes/reviewer-probes.log` (exit 0). Findings:

- No API accepts an epoch: succession always derives `head.Epoch+1`
  (never skips, never decreases); three successions pin epochs 2–4 with
  a linked predecessor chain. A second epoch-1 create refuses
  `lease_exists` (`TestReviewerSecondCreateAtSameEpochRefuses`).
- Sequential race on one basis: first CAS wins; the loser's differing
  bytes refuse **`stale_lease_epoch`**, not `lease_conflict`
  (`TestReviewerSequentialRaceOnSameBasis`). Correction to the review
  brief: `lease_conflict` is returned only for expectations naming
  nothing known (malformed/empty/foreign); a superseded-but-known basis
  with differing bytes is stale. The behavior is spec-consistent
  (loser does not win) and covered by narrowing mutant N-cas-stale.
- Byte-identical replay of a committed CAS returns the same reference
  with the blob directory byte-identical (name+size map unchanged).
- Replay of an old successor after the head advanced further still
  answers the persisted successor (re-mint against the pre-commit basis
  hits); it does not refuse (`TestReviewerReplayAfterHeadAdvanced`).
  Correction to the review brief's "(refused)": replay-after-advance is
  a safe answer, refusal applies only to differing bytes (stale).
- Holder is host-UUIDv7 only: same-host re-presentation verifies; the
  stored record carries no `pid`/`process_id`/`start_time`/`boot_id`/
  `holder_token` field. Same host + different process verifies (no
  process binding exists to reject it); there is no pid/start/boot
  material to survive a restart — holder identity is the host UUID
  alone, per §2.2 inv 4–5.
- Older-epoch token after a newer lease → `stale_lease_epoch`.
  Same-epoch tampered lease ID with valid v4 grammar → `lease_conflict`;
  tampered version nibble (v5) → `invalid lease record`.
- `R-cas-epoch-skip` (head+1 → head+2) KILLED twice by the successor
  and monotonicity suites.

## 3. Durability

- Split brain probed by unioning two partitioned epoch-2 blobs:
  `ListLeases` holds 3 blobs (1, 2, 2′), `WinningLease` elects the
  bytewise-greater lease ID deterministically, the loser is preserved
  and its token no longer verifies
  (`TestReviewerSplitBrainUnionElectsGreaterLease`).
- Staged-but-unrenamed temp is ignored by load; an `AfterLeaseStage`
  fault leaves no final and the retry lands the exact epoch. A tampered
  final blob refuses `chain_corrupt` through both `ListLeases` and
  `GetLease` funnels.
- Injector crash suites plus real-SIGKILL rename-seam drills
  (create + cas) rerun **twice**, exit 0 both runs
  (`probes/crash-rerun-1.log`, `crash-rerun-2.log`).

## 4. Refusal census (26 sites, 7 new sentinels)

`grep -c "refuse(Err" lease_store.go` = 26, matching the 26 narrowing
plants (23 new-sentinel sites + 3 `ErrChainCorrupt` funnel sites).
Census markers: 7 new boundary markers; equality ledger: 3 rows.
Attacks, each exit 1 with logs in `attacks/`:

- A1 (remove `").VerifyFencingToken"` marker): RED — 7 sites reported
  "never reached beneath a production boundary entry".
- A2 (swap conditions of two adjacent same-class grammar arms,
  v4↔v7): RED — `TestCreateLeasePersistsEpochOneCreate` fails.
- A3 (one refuse call stretched across two lines): RED — the
  line-anchored equality ledger mismatches (`lease_store.go:650` →
  `:651`); the refusal-derivation gate additionally requires
  single-line calls.

## 5. Mutation

- Producer battery rerun in an isolated copy on the exact final source:
  exit 0 — **27/27 applied plants KILLED** (26 narrowing + 1
  token-preserving `T-create-time-pad`); harmless control SURVIVED;
  NOT_APPLIED and COMPILE_OR_HARNESS_FAILURE classified separately
  (`verify/producer-battery-rerun-mutants.json` + `verify/per-plant/`
  raw logs).
- Reviewer-owned mutants on producer-unmutated arms, each KILLED twice
  (`own-mutants/per-plant/`, runner `own-mutants/reviewer_mutate.py`):
  R-expiry-boundary (`>` → `>=`, boundary grant must stay current),
  R-cas-epoch-skip (head+1 → head+2), R-fence-lease-directional (tie
  break admits only lesser IDs; the greater-ID loser then verifies),
  R-fence-holder-prefix (holder weakened to 8-char prefix); applied
  harmless comment control SURVIVED twice; control-after full package
  green. Honest note: a first prefix variant for the lease tie-break
  SURVIVED because the suite's winner/loser IDs share no 8-char prefix
  (too narrow to admit — not counted); the directional replacement
  above is the accepted evidence.
- Fail-closed mint confirmed from `N-create-token.log`: the admitted
  `not-a-uuid` token dies in mint with a plain operational error
  ("identify lease record: invalid canonical object identity…"),
  changing the refusal class — mint failures are operational errors,
  never refusals.

## 6. Bounds, docs, registry

- Provhost 5-site attestation bound holds: `go test
  ./internal/provhost` green (includes
  `TestNoProductionPathAttestsProviderIdentityBinding`); lease_store.go
  contains no direct `VerifyObjectIdentity` call (one comment mention).
- Added docs (`README.md`, `doc.go`, `LOGBOOK.md`, `TRACEABILITY.md`)
  claim the §5.3 lifecycle as implemented while
  `internal/traceability/ownership.v0.6.0.json` section:5.3 still
  carries the gap "lease acquisition, renewal, expiry and fencing are
  not implemented". **P2** per the review brief. Requirement: a later
  leaf of STORY-260830-1oqfec must update the section:5.3 binding and
  re-pin `reviewedOwnershipCanonicalSHA256` before story_final
  (precedent: Stories 3drr2m, 3jqsx1, 3m2mw8, 2jylym). No CLI, doctor,
  or capability claim exists in the candidate (added text explicitly
  disclaims all three).

## 7. Hygiene and suites

- Changed paths are exactly the 12 candidate paths (6 modified + 6 new:
  `lease_store.go`, `lease_store_test.go`, `lease_crash_test.go`,
  `TRACEABILITY.md`, `lease_store_adopt_test.go`, `testdata/mutate.py`);
  no `__pycache__`/`.pyc`, no `task-board.config.json`, no stray
  plants; work left UNCOMMITTED.
- `go vet ./...` exit 0; `gofmt` clean; `tracecheck` exit 0
  (contracts=63 sections=36 cases=101 fixtures=32);
  `cataloggen -check` exit 0.
- Suites rerun by the reviewer, exit 0: `go test ./internal/sessrepo
  ./internal/provhost -count=1`; `sessquery -run Lease`;
  `sessrepo -race`; `provhost -race`. Full `-count=1` runs of
  sessrepo/sessquery/sessstate/provhost observed exit 0 (background
  run; logs not retained). Full `-race` over all 26 packages and the
  fuzz/cross-compile matrix accepted from the attached producer
  evidence (`full-race.log` in
  `TASK-260830-2f5393_producer-evidence.tar.gz`).
