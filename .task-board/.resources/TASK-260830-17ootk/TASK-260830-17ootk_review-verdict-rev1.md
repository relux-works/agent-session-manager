# Review verdict — TASK-260830-17ootk rev1 (CR-TASK-260830-17ootk-1): ACCEPT

Reviewer run: RUN-260917-be21ad. Method: isolated immutable probes only;
the live Story worktree, index, branch, and HEAD were never mutated by this
review. All product evidence below was reproduced by the reviewer in the
live tree or in an isolated candidate copy (`/tmp/iso17`, since removed),
never accepted from producer logs alone.

Change Request: `CR-TASK-260830-17ootk-1` rev1, candidate tree
`25b762c5e4dbd97a49d0ef687c082d5941060781` over trunk base
`e4e3e8834675cf3814effd3b2673b4931dfbf311`.
Normative authority: pinned `internal/specdoc/SPEC.v0.6.0.md` §§5.4,
10.5–10.6, 13.12–13.13.

## Verdict: ACCEPT (route to `integrating` via `accept_cr`, evidence = this file)

## 1. Whole-Story diff — PASS

- `git verify-commit bde093e` and `8a757e0`: Good signatures (oparin@me.com).
- `git diff --name-only e4e3e88..25b762c` = 38 paths: the 27 committed
  checkpoint paths (sessckpt/matjournal/traceability/README/LOGBOOK) plus
  this leaf's 11 `internal/crashgate/` files and 6 leaf-modified files.
  The leaf touches no sessckpt/matjournal source: live `git status` shows
  exactly `M LOGBOOK.md README.md` + 4 traceability files and
  `?? internal/crashgate/`.
- Live worktree == candidate tree: the 6 tracked files show zero diff
  against `25b762c`, and `diff -r` of `internal/crashgate/` against
  `git archive 25b762c` is byte-identical.
- `task-board.config.json` byte-identical to trunk (`cmp` clean).
- Hygiene: no `__pycache__`/`.pyc` under crashgate/sessckpt/matjournal;
  `gofmt -l internal` empty; `go vet` clean on all four packages (exit 0);
  `cataloggen -adopted -output internal/catalog/catalog_gen.go -check`
  exit 0 after `go generate ./...` with no worktree side effects.

## 2. Registry derivation — PASS

- Independent probe (module-internal, git-ignored scratch, since removed)
  re-parsed Section 13.13 with its own range scanner: parsed=94
  (12 table ranges + 1 prose range), registered=94, missing=[], extra=[].
- Renamed-ID sensitivity proved in the isolated copy: `CR-MAT-01` →
  `CR-MAT-0X` makes `TestRegistryDerivesFromPinnedSpec` FAIL with
  `missing ["CR-MAT-01"], extra ["CR-MAT-0X"]` (exit 1).
- N/A audit by script: 48 reachable paths / 31 IDs, 121 N/A paths /
  63 IDs — exactly the claimed counts — and NO checkpoint/journal-class
  boundary is left without a driver.
- 10 N/A spot-checks against the landed code: no `clone`/`sync`/`bridge`/
  `lifecycle` packages exist; `terminalbackend` has no creation runtime
  (manifest/probe/conformance only); `provider` has no process-start or
  persistence runtime; `localstore`/`sessrepo`-adjacent rows are
  `WriteExternal` with disclosed out-of-story-boundary notes; no `doctor`
  surface exists. No row whose durable fact is a checkpoint capture or
  journal transition is marked N/A.

## 3. Conformance records — PASS, plus 5 independent reviewer probes

- Producer tarball holds 74 `conformance/*.json` records; all 74 validate
  mechanically (required fields present, `selected` in the 3-outcome enum,
  boundary ID in the registry).
- Live reruns (exit 0): full `crashgate` package, `TestRealSIGKILLSeams`
  twice (checkpoint_capture + journal_create), `TestCrashGateRejections`,
  `TestCrashGateMutualExclusion`, `TestCrashGateSection1312` (8 subtests).
- Reviewer probes (`reviewer_probe_test.go` in the evidence tarball, run
  in the isolated copy, 5/5 PASS, log attached):
  - CR-STOP-02 own AfterBlob crash → identical retry converges
    byte-identical, one checkpoint identity (safe_retry semantics).
  - CR-STOP-02 torn variant (disagreeing bytes at the digest path) →
    retry refuses `ErrCheckpointConflict`, torn bytes preserved, nothing
    installed.
  - CR-MAT-03 torn variant (truncated `journal.json`) → exactly
    `recoverable_parked_state` with durable `recovery.json`, frozen phase.
  - CR-MAT-06-path adopt crash + substituted native handle → exactly
    `recoverable_parked_state`, fail-closed, manager reference unchanged.
  - Rollback-facts + moved-lease ordering attack → exactly
    `recoverable_parked_state`, rollback NOT executed (phase stays
    staging, blocking `last_error` recorded).
- Two initial probe expectations were corrected during the run (both were
  probe errors, not product findings, reasoning recorded here): a torn
  journal cannot name operation IDs (quarantine evidence by construction),
  and the native-substitution gate applies only where a bound identity
  exists (`nativeRequired`: committed provider or adopted/resumed bridge),
  so substitution at the opened-but-unadopted stage is correctly out of
  that gate's scope — the harness proves substitution rejection where the
  gate applies (adopted binding).

## 4. Exclusivity and rejections — PASS

- `grep` over crashgate + matjournal sources and records: the only outcome
  strings are `safe_retry`, `explicit_rollback`, `recoverable_parked_state`
  (the constant definitions). No fourth outcome exists.
- The 10 shipped rejection rows pass live; the reviewer ordering attack
  above proves conflicting rollback/parked evidence parks without
  executing rollback.

## 5. Section 13.12 — PASS (7 of 19 driven, 8 subtests green)

- The 12 unlanded-owner rows verified against the tree (sync flow, lease
  arbitration, provider runtime/adapter, bridge, lifecycle orchestration,
  clone flow absent or partial-landed as claimed).

## 6. Mutation — PASS

- Producer harness reproduced in the isolated copy: 9 KILLED + 1 control
  SURVIVED, 0 not-applied, 0 anomalies (summary + per-plant logs in
  evidence). `N-registry-grace11` shows static pass + behavior fail,
  proving the token-preserving mutant runs the behavioral suite.
- Each of the 9 KILLED re-run twice more (named killer only): all exits
  `[1, 1]`, zero rerun failures.
- 4 reviewer-owned narrowing mutants on predicates the producer did not
  mutate (isolated copy, logs in evidence): `R-lease-forward`,
  `R-substitution-leaseactive`, `R-exhaustiveness-rollback-contradiction`,
  `R-registry-stop02` (static pass + behavior fail — second
  token-preserving proof) — all KILLED; `R-control-comment` SURVIVED.

## 7. Story-close items — PASS

- `section:13.13` partial 9/11 with the `#4`/`#5` journal-halves gap
  disclosed; `section:13.12` stays `unmeasured`, which is the honest and
  gate-required level (the scanner cannot measure table-shaped sections;
  claiming `partial` would be refused; the gap names the 7 driven rows and
  the harness case). 2 new acceptance cases present;
  `reviewedOwnershipCanonicalSHA256=a1b48479…` enforced by the passing
  `tracecheck` (107 cases, 26/489 clauses — reproduced exactly).
- Self-minted-claim attack: planting a bogus `full` binding for 13.13 in
  the isolated copy makes the traceability suite FAIL (refused).
- README crashgate section carries test commands and states no `ax`
  command, no `doctor` result, no capability claim; figures refreshed
  (107 cases, 26/489 — match the observed run).
- LOGBOOK newest-first: leaf entry on top, trunk `3lt2xv` entry preserved,
  both predecessor Story entries present.

## AC coverage: 4 of 4 rows driven

1. Crash after every durable boundary, exactly one outcome —
   `sessckpt.Store.Capture` + `matjournal.Store.Create`/updates/`Recover`:
   48/48 reachable paths (72 rows + 2 SIGKILL).
2. Fixtures and negative/refusal cases — `Recover` rejection gates: 10/10
   rejections + 12 exclusion rows, each failing closed.
3. Crash/idempotency evidence — 74 machine-readable conformance records.
4. No unsupported capability — no `cmd` surface, modeled probes only,
   six stated bounds in TRACEABILITY.md.

No product behavior was changed by this leaf (no sessckpt/matjournal
source edits); no findings remain open. No Stop-The-Line condition met.
