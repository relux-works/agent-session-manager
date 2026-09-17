# Review verdict — TASK-260830-17ootk rev2 (CR-TASK-260830-17ootk-2): ACCEPT

Reviewer run: RUN-260917-f20a9b. Method: isolated immutable probes only;
the live Story worktree, index, branch, and HEAD were never mutated by this
review (no product edits, commits, checkpoints, or integration). All product
evidence below was reproduced by the reviewer in the live tree, in the
isolated copy `/tmp/iso17` (mutant harnesses; since removed), or in `/tmp`
scratch (record validation; since removed) — never accepted from producer
logs alone.

Change Request: `CR-TASK-260830-17ootk-2` rev2, candidate tree
`0ec666f3bd44fc7280a16f70f7d745441b6f59a4` over trunk base
`9e9fe5154f5e8d97151670c3f689821cc3431d79`.
Normative authority: pinned `internal/specdoc/SPEC.v0.6.0.md` §§5.4,
10.5–10.6, 13.12–13.13.

Scope note: rev1 (`25b762c`, accepted by RUN-260917-be21ad, verdict
`TASK-260830-17ootk_review-verdict-rev1.md`) was demoted to stale by
`integration_base_moved` and republished onto the new trunk. Per the rev2
brief this review is SCOPED to the reconciliation: the product tree was
proven byte-identical to the rev1 candidate outside `LOGBOOK.md`,
`README.md`, and the four traceability files, the refresh replays and every
stale-copy restoration were verified, the merged registry digest was
recomputed through its gates, and the suites were rerun. The rev1 deep
review is not redone where the bytes are identical; every claim reused from
rev1 is named as such below.

## Verdict: ACCEPT (route to `integrating` via `accept_cr`, evidence = this file)

## 1. Whole-Story diff — PASS

- `git verify-commit 8299fdc` and `813b05d`: Good signatures
  (oparin@me.com, ECDSA `SHA256:V6JiKG7J29mjsvikcLoSVp0bLa77VTsFy12gnLO81cM`).
  Parent chain: `813b05d → 8299fdc → 9e9fe51` (fresh trunk base).
- Per-delta invariant holds: `git diff 9e9fe51 8299fdc` shows only the
  14yo67 delta (10 files), `git diff 8299fdc 813b05d` only the 3k3e6m delta
  (19 files). (The republish brief's stricter
  `git diff <original> <replayed>` empty-except-LOGBOOK expectation does not
  apply here: trunk's STORY-260830-315721 changed the same traceability
  files, so the replayed trees necessarily carry merged figures — the
  per-delta check is the meaningful invariant and it holds.)
- Trunk entries preserved newest-first: worktree `LOGBOOK.md` heads
  `17ootk`, then trunk's `2zvo8m` (STORY-260830-315721), then `3lt2xv`,
  then the Story checkpoints.
- Leaf scope: live `git status` shows exactly `M LOGBOOK.md README.md` +
  4 traceability files and `?? internal/crashgate/` (11 files). No
  sessckpt/matjournal source change in the uncommitted diff
  (`git diff --name-only | grep sessckpt/matjournal` empty). The 27 tracked
  whole-Story paths + 11 crashgate files = the 38 CR paths.
- `task-board.config.json` byte-identical to trunk (`cmp` clean).
- Stale-copy audit: worktree status contains no trunk-landed package
  (`provhost`, `resumesmoke`, `sessprofile`, `environ`, `secprim`) and no
  `.task-board` artifact — nothing stale overlaid; all non-candidate paths
  equal HEAD by construction of the clean status.
- Hygiene: no `__pycache__`/`.pyc` under crashgate/sessckpt/matjournal;
  `gofmt -l internal` empty (exit 0); `go vet` on all four packages exit 0;
  `go build ./...` exit 0; `cataloggen -adopted -output
  internal/catalog/catalog_gen.go -check` exit 0.

## 2. Reconciliation byte-identity + registry derivation — PASS

- `diff -rq` of live `internal/crashgate/` against
  `git archive 25b762c` (rev1 candidate): identical, all 11 files.
- Independent set proof (reviewer instrument, `/tmp` scratch): spec range
  tokens expand to exactly 94 unique IDs (78 table in 12 ranges + 16
  `CR-CLONE` prose); registry literals extract to 94 IDs; sets equal.
  Live `TestRegistryDerivesFromPinnedSpec` passes
  ("derived 78 table IDs in 12 ranges and 16 prose IDs in 1 ranges;
  registry matches exactly") and `TestRegistryStructure` passes
  ("registry holds 94 boundaries (31 reachable) over 48 reachable paths").
- Rename sensitivity: private renamed copy (`CR-MAT-01..08` →
  `CR-MAT-0X..08`) parses to 86 IDs with `CR-MAT-01..08` missing from the
  document side → REDDENS. (Go-level rename proof reused from rev1 on
  identical bytes: `CR-MAT-01` → `CR-MAT-0X` fails with
  `missing ["CR-MAT-01"], extra ["CR-MAT-0X"]`.)
- N/A audit: 54 distinct owner strings; 10+ spot-checks against the landed
  code — no `clone`/`bridge`/`lifecycle`/`doctor`/`sync` packages exist;
  `internal/provider` "holds no state and starts no process" (no
  process-start/persistence runtime); `internal/terminalbackend` has no
  Create/Launch/exec in non-test code (no creation runtime). Two
  script-flagged rows investigated and cleared (probe over-match, not a
  finding): `CR-GRACE-03` (quiescence proof generation writes no
  checkpoint/journal bytes) and `CR-STOP-04` (closure verification writes
  no checkpoint/journal bytes) — both honestly N/A with the exact unlanded
  owner named. No row whose durable fact IS a checkpoint capture or journal
  transition is marked N/A.

## 3. Conformance records — PASS, plus reviewer reruns

- Live reruns (all exit 0): full crashgate/sessckpt/matjournal/
  traceability suites `-count=1` (crashgate 15.3s incl. real-SIGKILL rows,
  sessckpt 4.0s, matjournal 16.7s, traceability 2.9s, tracecheck pkg
  31.6s); `TestRealSIGKILLSeams -count=2` green twice;
  `TestCrashGateRejections|TestCrashGateMutualExclusion|
  TestCrashGateSection1312|TestConformance` green; `-race` green on all
  three packages (crashgate 13.3s, sessckpt 4.7s, matjournal 15.7s).
- Reviewer-regenerated records (`CRASHGATE_CONFORMANCE_DIR=/tmp/cg-records`
  over live `TestCrashGateConformance`): 72 JSON records, 72/72
  mechanically valid — every record carries boundary, path, operation IDs,
  pre/post durable facts (`phase_before/after`, `receipt_present`),
  external effect + status probe, winning lease before/after, native
  identity/binding before/after, `selected`, reason/remediation; every
  boundary ID in the registry; `selected` in the 3-outcome enum with all
  three outcomes represented (safe_retry 60, recoverable_parked_state 11,
  explicit_rollback 1). The remaining 2 of the producer's 74 are the
  SIGKILL-seam records, covered by the twice-rerun SIGKILL rows above.
- Independent crash probes on identical bytes are reused from rev1 (5/5:
  STOP-02 hook crash convergence, STOP-02 torn bytes refusal, MAT-03 torn
  journal parking, MAT-06 adopt-crash substitution parking, rollback-facts
  + moved-lease ordering attack parking without executing rollback); fresh
  attack evidence on THIS tree is in §4/§6 (4 reviewer-owned narrowing
  mutants KILLED, producer harness reproduced).

## 4. Exclusivity and rejections — PASS

- `grep` over crashgate + matjournal non-test sources: the only outcome
  constructors are `OutcomeSafeRetry`, `OutcomeExplicitRollback`,
  `OutcomeRecoverableParked` (9/32/27 uses). No fourth outcome exists
  (`rollback_failed` is a resume-reason string at `recover.go:641`, not an
  outcome; `*_parks` strings are test names).
- The 10 shipped rejection rows pass live; reviewer mutant
  `R-two-auth-lease-match` (two-authorities gate skipped on matching lease)
  is KILLED by `TestCrashGateRejections/two_live_authorities`, proving the
  gate admits nothing when weakened by exactly one member class; mutant
  `R-rollback-contradiction-narrow` (RollingBack admitted, RolledBack kept)
  is KILLED by `contradictory_rollback_with_activation_parks`, proving
  conflicting rollback evidence parks instead of executing rollback.

## 5. Section 13.12 — PASS (7 of 19 driven, 8 subtests green)

- `TestCrashGateSection1312` green live (operator interrupt, owner-resume
  response loss, rename blocked incl. real fault, disk full ENOSPC +
  read-only, import/adopt lease failures). The 12 unlanded-owner rows name
  genuinely unlanded flows (verified absence in §2).

## 6. Mutation — PASS

- Producer harness reproduced in the isolated copy `/tmp/iso17`
  (PYTHONDONTWRITEBYTECODE=1): `mutants: 9 killed, 1 survived,
  0 not applied, anomalies=0` (exit 0). `N-registry-grace11` static-pass +
  behavior-fail re-observed in the summary (token-preserving proof).
- 4 reviewer-owned narrowing mutants on predicates the producer did not
  mutate (isolated copy, per-plant logs in evidence):
  `R-host-diskfull` (disk_full dropped, rename kept;
  killer `TestCrashGateSection1312/disk_full_enospc_at_seam`) — KILLED;
  `R-two-auth-lease-match` (two-authorities skipped on matching lease;
  killer `TestCrashGateRejections/two_live_authorities`) — KILLED;
  `R-rollback-contradiction-narrow` (RollingBack admitted;
  killer `.../contradictory_rollback_with_activation_parks`) — KILLED;
  `R-registry-stop04-driven` (CR-STOP-04 flipped N/A→reachable, token
  preserved; killer `TestCrashGateConformance` coverage) — KILLED with
  static derivation still passing (second token-preserving proof);
  `R-control-comment` (harmless) — SURVIVED. Precise denominators: 4/4
  narrowing KILLED, 1/1 control SURVIVED, 0 not-applied.

## 7. Story-close items — PASS

- `section:13.13` `partial` 9/11 with the `#4`/`#5` journal-halves gap
  disclosed; `section:13.12` stays `unmeasured` with the harness gap naming
  the 7 driven rows — concur with rev1 that `unmeasured` is the honest and
  gate-required level (assigned-scope admission requires `full`; the
  scanner cannot see table-shaped sections, and claiming `partial` would be
  self-minted evidence). 2 new acceptance cases present
  (`crash-gate-journal-conformance` → `matjournal/recover.go` `Recover`
  (`Store.Recover`, recover.go:246); `crash-gate-checkpoint-conformance` →
  `sessckpt/capture.go` `Capture` (`Store.Capture`, capture.go:114)); every
  named test exists and runs green (§3).
- Digest: `traceability.go` diff vs HEAD is exactly the one-line re-pin
  `89e9e8fd…` → `ba242a9d55e11da9299d3732197311f1e874be3f43e0bf11d0ec00de30937afd`;
  the gate is green in two independent binaries (package tests +
  `tracecheck`: `contracts=63 normative_sections=36 acceptance_cases=119
  ... clauses_discharged=38/489`, exit 0). Limit stated honestly: a third,
  fully independent reimplementation of the canonical projection was not
  built; the pin is verified through the two shipped gates, not by an
  outside reimplementation. A self-minted-claim probe is reused from rev1
  (gate logic unchanged — the only `traceability.go` delta is the pin).
- README crashgate section carries test commands and ends with the explicit
  non-claim ("adds no `ax` command, no `doctor` result, and no runtime
  capability claim"); removed README lines are figure-only. LOGBOOK is
  newest-first with all Story entries plus trunk's `2zvo8m` and `3lt2xv`.
- Conformance matrix `TASK-260830-17ootk_conformance-matrix-rev2.md`
  (rev1 matrix + refresh note) reviewed; counts re-proved live (94 IDs,
  48 reachable / 121 N/A paths via the structure test).

## 8. AC coverage — 4 of 4 AC rows driven through production entry points

Production call sites: `Store.Recover`
(`internal/matjournal/recover.go:246`) and `Store.Capture`
(`internal/sessckpt/capture.go:114`).

| # | AC row | Named committed test → production call site |
| - | ------ | ------------------------------------------- |
| 1 | Crash after every durable boundary proves exactly safe_retry, explicit_rollback, or recoverable_parked_state | `TestCrashGateConformance` → `Recover`/`Capture` (48 reachable paths, 72 records + 2 SIGKILL seams) |
| 2 | Exact contract fixtures and negative/refusal cases pass | `TestCrashGateRejections` (10 rows) + `TestCrashGateMutualExclusion` + `TestCrashGateSection1312` → `Recover` |
| 3 | Crash/idempotency evidence for durable mutations | `TestRealSIGKILLSeams` (×2 reruns) + per-row conformance records → `Capture`/`Recover` |
| 4 | No unsupported capability advertised | tracecheck gate + README non-claim line + `cataloggen -check` (all exit 0) |

No stated bounds beyond the disclosed ones (63 N/A IDs with owners, 13.13
9/11, 13.12 7/19 — each mapped in the conformance matrix).

## Attachments

- `TASK-260830-17ootk_review-verdict-rev2.md` (this file)
- `TASK-260830-17ootk_review-evidence-rev2.tar.gz` (per-plant mutant logs,
  producer-harness iso summary, suite/SIGKILL/tracecheck/cataloggen logs,
  record-validation report, diff fingerprints)

Live merged checklist: all 19 items already `done`; no change needed.
No `commit_ack` supplied (reviewer archetype). No Stop-The-Line: no
evidence-backed external blocker; no unresolved human-only decision.
