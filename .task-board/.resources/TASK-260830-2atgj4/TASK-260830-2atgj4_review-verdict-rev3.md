# Review verdict — TASK-260830-2atgj4 rev3: property-test the ownership reducer (final leaf, republish on trunk 7bf90af)

## Verdict: ACCEPT revision 3

`accept_cr(TASK-260830-2atgj4, revision=3, evidence=TASK-260830-2atgj4_review-verdict-rev3.md)`.

Change Request `CR-TASK-260830-2atgj4-3` (story_final, candidate tree
`01506b12a91c206d5dbfacc91c714b2bca465780` on trunk base
`7bf90affef6c880e243f6176e6e5f762a293e1a3`, 37 whole-Story paths, patch
sha256 `f18c1f847aa60b0b74b180e5b1b5b13fffc4a211c0e998e722077f8b25a6b85d`)
is accepted for integration. No P1, P2 or P3 findings. Reviewer: RUN-260917-19f291
(claude-opus-5 max). This review was scoped to the base reconciliation after
CR2 (accepted by RUN-260917-665e22, `TASK-260830-2atgj4_review-verdict-rev2.md`)
was demoted to stale by `integration_base_moved`; the rev2 deep review carries
over because the product tree is proven byte-identical (section 2). Every gate
was nevertheless re-attacked here with the producer battery and seven
reviewer-owned narrowing plants (section 5).

Method: read-only over the live Story worktree — no product edit, commit,
checkpoint, branch or index operation. Every mutant ran in an isolated copy
under `.temp/TASK-260830-2atgj4/rev3/` (gitignored; copies deleted after the
evidence was captured). The worktree hashed to the candidate tree OID
(`git read-tree HEAD && git add -A && git write-tree` under a scratch index)
before the review, after the test runs, and after every harness run:
`01506b12…` each time; HEAD stayed `995fe54`.

## 1. Immutable CR bytes and whole-Story diff — confirmed

- Patch resource digest `f18c1f84…` reproduced by
  `git diff 7bf90af 01506b12` (and `--binary`); the measured 37 changed paths
  equal the CR record's `changed_paths` set.
- Chain: `995fe54` (replayed 3g12yp) -> `bf21147` (replayed 2f5393) ->
  `7bf90af` (trunk). `git verify-commit` good on both (oparin@me.com, ECDSA
  key SHA256:V6JiKG…), author/committer Ivan Oparin.
- Replayed checkpoints vs originals (`0ebb7fa`, `1b8e75a`): path sets equal
  (12 and 22 paths). Every path trunk did not touch between `62d4463` and
  `7bf90af` carries a byte-identical blob (10 of 12 for 2f5393, 16 of 22 for
  3g12yp). The differing blobs are exactly the trunk-touched paths:
  - 2f5393: `README.md` equals the conflict-free 3-way merge
    (base `62d4463`, trunk `7bf90af`, checkpoint `0ebb7fa`) — `git merge-file`
    conflicts=0, output byte-equal; `LOGBOOK.md` is purely additive over trunk
    (0 removed / 38 added) and contains every one of the 38 lines the original
    checkpoint added.
  - 3g12yp: `LOGBOOK.md` purely additive over `bf21147` (0 removed / 43 added,
    all 43 original lines present); `README.md`, the registry, the digest pin
    and both figure-pin test files are the producer's 5 replay resolutions.
    The replayed checkpoint is internally consistent: in a `git archive`
    extraction of `995fe54`, its own `tracecheck` exits 0 with
    `acceptance_cases=128 … clauses_discharged=49/511` and
    `go test ./internal/traceability/... -count=1` is ok; the registry digest
    recomputed independently is `cc021c38…9430fdb` = its pinned value.
    (One log line in `09-replayed-checkpoint-995fe54.log` ran the LIVE tree's
    binary over the checkpoint's registry and reports a digest mismatch — an
    invocation artifact, annotated in the log, not a checkpoint defect.)
- `task-board.config.json`: candidate blob `f89a9fe…` == `7bf90af` blob.
- This leaf beyond the checkpoints: the 13 stated paths (9 modified + 4 new
  including `internal/sessstate/testdata/mutate_properties.py`); `git status`
  shows exactly those; no `__pycache__`, `.pyc` or stray plant anywhere in the
  tree (`find` outside `.temp`).

## 2. Reconciliation against rev2 — product tree byte-identical

- rev2 candidate `92d7da93` -> rev3 candidate `01506b12` changes exactly 215
  paths, and that set equals `git diff --name-only e4e3e88 7bf90af` (the trunk
  advance). Of these, 209 lie outside the CR's 37 paths and every one carries
  the `7bf90af` blob (0 mismatches). The remaining 6 are the reconciled files:
  `LOGBOOK.md`, `README.md`, `internal/traceability/{ownership.v0.6.0.json,
  traceability.go, traceability_test.go, cmd/tracecheck/main_test.go}`.
- The other 31 CR paths (all of `internal/sessstate`, `internal/sessrepo`,
  `internal/fencing`, `internal/sessquery`, the harness and the property
  tests) are byte-identical between rev2 and rev3 (0 mismatches). The rev2
  RED/GREEN reconstruction, minimal-fix and generator-domain findings therefore
  stand unchanged.
- Six reconciled files, each checked with `git merge-file` (base `e4e3e88`,
  ours `7bf90af`, theirs rev2) and then against the exact rev3 blob:
  - `LOGBOOK.md`: purely additive over trunk (0 removed / 128 added) and over
    rev2 (1 removed / 244 added — the removed line is the leaf's own count
    sentence `110 to 113` rewritten `128 to 131`). Headings under 2026-09-17
    newest-first: 2atgj4, 17ootk, 2zvo8m, 3lt2xv, 3uzfyn, kp4zpu, 3k3e6m,
    14yo67, 3g12yp, 2f5393.
  - `README.md`: the naive 3-way merge has 6 conflicts; the rev3 blob resolves
    each by keeping both sides (trunk's Execution-profile/Native-resume sections
    contiguous, the Story's fencing and property sections inserted after) and
    rewriting the coverage figures. Trunk's `-adopted` cataloggen text intact.
    Prose enumeration cross-checked: full=2 (6.2, 2.4), partial=6 (13.13,
    14.2, 5.3, 15.1, 15.3, 7.7), sliver=4 (10.3, 5.5, 8, 2.2), unmeasured=4
    (7.3, 13.12, 13.14.5, 15.2), unevidenced=44, unowned=11; 2+6+4+4+44 = 60.
  - Registry `ownership.v0.6.0.json`: semantic union verified entry by entry.
    Acceptance cases: rev3 ids = trunk (119) ∪ rev2 (113) = 131; trunk added 18,
    Story added 12, no overlap, no id modified by both sides, every entry
    byte-equal to its source, both source orders preserved as subsequences.
    Ownership rows: 65 = trunk 64 + the Story's `section:2.2` binding; trunk
    modified 7.7/8/2.4/5.5/5.4, the Story modified 5.3/17.2 — disjoint; every
    row equals its source. Unowned: 11 = trunk's 12 minus `section:2.2`
    (now bound), entries byte-equal.
  - Digest: `reviewedOwnershipCanonicalSHA256 = 39087ef8a9696ea46d9408d29c159c732c3c9ab229949f3f6c7c38f867aab8c3`
    recomputed by an independent Go program (struct shapes copied from
    `traceability.go`, `json.Marshal` + sha256) over the rev3 registry:
    `39087ef8…` — match. Control: the same program reproduces trunk's
    `ba242a9d…` and rev2's `63c83796…` pins from their registries.
  - `traceability_test.go` / `main_test.go`: figures 131 / 60 / 2 / 6 / 4 /
    44 / 4 / 11 / 49/511 — exactly what `tracecheck` prints on the tree.

## 3. Gates rerun on the exact candidate tree (all real exits)

| Command | Result |
| --- | --- |
| `gofmt -l` (tracked+untracked .go) | empty, exit 0 |
| `go build ./...` / `go vet ./...` | exit 0 / exit 0 |
| `go run ./internal/traceability/cmd/tracecheck` | exit 0: `acceptance_cases=131 … bindings=60 full=2 partial=6 sliver=4 unevidenced=44 unmeasured=4 unowned=11 clauses_discharged=49/511` |
| `cataloggen -adopted -output internal/catalog/catalog_gen.go -check` | exit 0 |
| `go test ./internal/sessstate ./internal/sessrepo ./internal/fencing ./internal/sessquery ./internal/traceability/... -count=1 -v` | exit 0, 1145 PASS lines, 0 FAIL (2:08) |
| same packages `-race -count=1` | exit 0, all 6 ok, no DATA RACE (5:49) |
| `go test ./... -count=1` | exit 0, 32/32 packages ok (2:33) |
| `git diff --check` | exit 0 |

Accepted from the attached rev3 validation log (windowed capture, summary
`required=26 green=26 failed=0 missing=0`): the `-race` and `-cover` full
sweeps, the fuzz smokes (commands 7-19), the cross builds, JSON validation and
`task-board validate`. Every review-scope package was rerun here as listed.

## 4. Story-close items — confirmed on the rev3 tree

- P3 vector `epoch_zero_other_lease` (`{session A, epoch 0, fenceLeaseC}`)
  present in `internal/fencing/gate_test.go:45`; green in the suite run.
- `section:17.2` gap carries the narrower reader-enum wording ("discharged
  for the reader's own enums and silent for enums the reader never observes").
- Three acceptance cases `lease-ownership-{union,store,gate}-properties` bind
  `sessstate.Reduce`, `sessrepo.WinningLease`, `fencing.Authorize` to the 11
  `TestOwnership*` declarations; attached to 5.3#6/#7 (lines 1967/1968) and
  2.2#1/#2/#4 (lines 408/410/413); every pinned line matches
  `internal/specdoc/SPEC.v0.6.0.md` (sha256 `74504539…` = registry source pin).
- Self-minted claim: appending `reviewer-self-minted-claim-rev3` (real
  declarations, unreviewed) to the registry in an isolated copy makes
  `tracecheck -root <copy>` exit 1 (`projection digest ccd4d2e6… differs from
  reviewed 39087ef8…`); the unmodified copy is green (control).
- README: `## Ownership reducer properties` (line 2085) and the
  `ownership properties` harness tool row (line 3308) present; explicit
  "adds no `ax` command, no `doctor` result, and no runtime capability claim".

## 5. Gates attacked, not read

Producer battery `internal/sessstate/testdata/mutate_properties.py` rerun twice
on the exact tree (`05-battery-run1.log`, `05b-battery-run2.log`, harness exit 0
both, ~50s each; classifications, exits and failing tests identical across the
two runs): 8/8 `N-` plants
KILLED by their named property with exit 1 and the expected witness
(`N-compare-tiebreak`/`N-union-detail-first` -> `TestOwnershipUnionOrderIndependent`;
both loser-drop plants -> `TestOwnershipLoserHistoryPreserved`; `N-create-clock`
-> `TestOwnershipStoreClockNonAuthority`; `N-expiry-absolute` ->
`TestOwnershipGateClockNonAuthority`; both gate plants ->
`TestOwnershipGateSingleAuthorizedOwner`); `C-harmless-comment` SURVIVED (applied,
exit 0); `C-not-applied` NOT_APPLIED; `C-compile-failure`
COMPILE_OR_HARNESS_FAILURE; both `TestOwnership` controls exit 0. Per-plant raw
logs and `mutants-full.json` for both runs attached.

Reviewer-owned narrowing plants (`own_mutate_rev3.py`, isolated copy, per-plant
raw logs, exits in `own-mutants.json`), all distinct from the producer's and the
rev2 reviewer's plants — 7/7 KILLED, control SURVIVED:

| Plant | Narrowing (gate stays present, admits exactly one class member) | Killed by (exit 1) |
| --- | --- | --- |
| R3-P1-loser-order-epoch-only | loser evidence sorted by epoch only, so two same-epoch losers report in arrival order | `TestOwnershipUnionOrderIndependent` — permutation of `{1,R},{1,A},{1,B}` changed the conflict order |
| R3-P1-partition-second-half-dropped | a same-epoch loser arriving LAST in a 3-entry union loses its evidence (reconnect tail) | `TestOwnershipUnionPartitionStable` |
| R3-P2-loser-same-id-lower-epoch-dropped | a lower-epoch loser carrying the winner's own lease ID is dropped from the evidence | `TestOwnershipLoserHistoryPreserved` — `{1,R},{2,R}` |
| R3-P2-store-epoch1-hidden-in-long-chain | `scanLeaseBlobs` hides the epoch-1 loser once the chain holds ≥4 leases (`ListLeases`/`WinningLease` path) | `TestOwnershipStoreSingleAuthorityPerEpoch` — "3 leases stored, want 4" |
| R3-P3-store-skips-2020-created-at | the store consults `created_at`: a 2020-stamped blob is not loaded | `TestOwnershipStoreClockNonAuthority` — the 2020 shift's epoch-1 lease vanishes and the CAS refuses `unknown lease` |
| R3-P3-gate-refuses-input-in-2027 | `Authorize` refuses `input` when the wall clock reads 2027 (absolute position consulted) | `TestOwnershipGateClockNonAuthority` — +365d translation of `input` flips to `refused:invalid_arguments` |
| R3-P4-gate-admits-future-epoch-9 | the epoch arm is skipped for presented epoch 9, so the future presenter (winner's lease ID) is minted a token | `TestOwnershipGateSingleAuthorizedOwner` — `local/input/future` authorized |
| C-harmless-comment | comment only, applied | SURVIVED, exit 0 |
| C-not-applied | absent token | NOT_APPLIED |

No source-text gate is introduced by this leaf (the census line pins moved
in `census_state_test.go` were exercised by the green suite), so no
token-preserving mutant is owed beyond those the predecessor verdicts recorded.

AC coverage: 18 of 18 conformance-matrix rows driven through the named
production entries (`sessstate.Reduce`/`Compare`, `sessrepo.CreateLease`/
`CompareAndSwapLease`/`WinningLease`/`ListLeases`/`GetLease`/`VerifyFencingToken`,
`fencing.Authorize*`), per `TASK-260830-2atgj4_conformance-matrix-rev3.md` and
the 11 green `TestOwnership*` properties; the disclosed bounds (grant age
authoritative by design, first-malformed-entry naming, no `ax`/`doctor`/runtime
claim) are unchanged from rev2. Crash/idempotency: the leaf changes no durable
mutation (pure derivation); the landed `TestCrashBefore/AfterLease*` tests ran
green inside the sessrepo package run.

## 6. Landing note (not a finding)

`origin/main` advanced during this review from `7bf90af` to `4d0f467`
(`TASK-260917-tbh71p`, `task-board.config.json` only: reviewer routing). That
path is outside the CR's 37 changed paths and the 26-command validation suite
is byte-identical between the two configs (sha256 of the command list
`4d0ea0dd…` on both), so the integration should auto-reparent rather than
refuse `integration_base_moved`; if it refuses anyway, a refresh needs no
content reconciliation.

## Reran vs accepted

Reran myself: tree-identity hashes (3x), patch digest, replay signature and
blob/3-way checks, registry semantic union and independent digest recomputation
(with trunk/rev2 controls), tracecheck + self-mint refusal, cataloggen check,
gofmt/vet/build, `git diff --check`, review-scope suites normal and `-race`,
full `go test ./... -count=1`, producer battery twice, seven own plants plus
controls, replayed-checkpoint self-consistency (995fe54 tracecheck + tests).
Accepted from attached evidence: full `-race`/`-cover` sweeps, fuzz smokes,
cross builds, JSON validation, `task-board validate` (rev3 validation log) and
the rev2 deep review of the byte-identical product tree.

No product edits made. Candidate left UNCOMMITTED as required. The repo
LOGBOOK entry belongs to the producer and was not touched.
