# Review verdict — TASK-260830-2atgj4 rev2: property-test the ownership reducer (final leaf)

## Verdict: ACCEPT revision 2

`accept_cr(TASK-260830-2atgj4, revision=2, evidence=TASK-260830-2atgj4_review-verdict-rev2.md)`.

Change Request `CR-TASK-260830-2atgj4-2` rev 2 (candidate tree
`92d7da93a719d80810895e2ce5cfaf1c23ce68c7` on trunk base
`e4e3e8834675cf3814effd3b2673b4931dfbf311`, 37 whole-Story paths) is
accepted for integration. No P1, P2, or P3 findings. The live Story
worktree, index, branch, and HEAD were never mutated: every probe ran
in isolated copies under `/tmp/review-2atgj4/` or as read-only
`go test`/`go vet`/`go run` on the live tree. No commit, checkpoint,
or branch operation was performed. Scratch source copies
(`src-*`, `mutation-source-*`, `selfmint/`) are deleted after the
evidence bundle is sealed; only logs, JSONs, and the harness script
are retained in the attached tarball.

## 1. Whole-Story diff (story_final) — confirmed

- Chain: `ef71cef` (replayed 3g12yp) -> `67445fa` (replayed 2f5393) ->
  `e4e3e88` (trunk). `git verify-commit` good on both replayed commits
  (oparin@me.com).
- Blob comparison original-vs-replayed: all 2f5393 files SAME except
  `LOGBOOK.md` (resolution) and `README.md`, whose delta is exactly the
  trunk 3lt2xv `-adopted` cataloggen text (trunk delta, not a checkpoint
  change). All 3g12yp production files SAME (`gate.go`, `token.go`,
  `fixtures`, `mutate.py`, `traceability.*`, `lease_store_adopt_test.go`);
  only `LOGBOOK.md`/`README.md` carry the trunk delta.
- `task-board.config.json` byte-identical trunk-vs-HEAD (zero diff).
- This leaf beyond the checkpoints: exactly the 13 stated paths —
  9 modified (`LOGBOOK.md`, `README.md`, `fencing/gate_test.go`,
  `sessstate/census_state_test.go`, `sessstate/sessstate.go`,
  `traceability/cmd/tracecheck/main_test.go`,
  `traceability/ownership.v0.6.0.json`, `traceability/traceability.go`,
  `traceability/traceability_test.go`) + 4 new
  (`fencing/ownership_properties_test.go`,
  `sessrepo/lease_properties_test.go`,
  `sessstate/ownership_properties_test.go`,
  `sessstate/testdata/mutate_properties.py` + `mutate_properties.py`
  under it). `git status` shows only these; no `__pycache__`, no `.pyc`,
  no stray plants.
- LOGBOOK order newest-first verified:
  2atgj4, 3lt2xv (trunk), 3g12yp, 2f5393.

## 2. Defect and minimal fix — reconstructed, not read

- RED reproduced in an isolated copy (HEAD `sessstate.go` + candidate
  property tests): `TestOwnership*` -> 4 FAIL / 1 PASS, exit 1
  (`red-ownership-properties.log`). The winner-only test passes; the
  four evidence/order tests fail — exactly the claimed signature.
- GREEN on the live tree: 5/5 PASS, exit 0
  (`green-ownership-properties.log`).
- Fix is minimal: winner selection is still a single greatest-tuple
  scan (unchanged semantics); reporting moved to a second pass in
  ascending tuple order naming the final winner. No winner changes:
  full `sessstate`/`sessrepo`/`fencing`/`sessquery` suites green (below).
- Off-chain flag equivalence proven from chain monotonicity:
  `trackLease` advances `current` only on strict tuple increase, so
  `current` is the tuple-maximum over the chain; any union entry above
  it is necessarily off-chain. Old flag set implies final winner above
  `current` (hence off-chain and in-union, new flag true); new flag true
  implies the winning entry exceeded the running winner at arrival (old
  flag set). Equivalent in all cases, including the empty chain.

## 3. Four invariants as falsifiable properties — 11/11 green, all plants killed

Generators inspected in the three `ownership_properties_test.go`
files: closed alphabets as claimed (epochs 1..4, leases R/A/B/C,
6-UUID store alphabet, 4 Section-5.3 reasons, 6 presenters x
5 observations x 5 operations, 4 clock shifts, 4 ages x 4
translations). Recorded counts reproduced in my run: 63,050 Reduce
calls (2,907 exhaustive pairs + 2,000 seed-260830 random), 21,315
partition concatenations (24,222 calls), 289 Compare pairs / 4,913
triples vs the builtin-operator oracle, 21 exhaustive + 200
seed-260831 reason sequences, 150 gate authorizations, 80 translated
authorizations. 11/11 `TestOwnership*` PASS.

Reviewer-owned narrowing mutants (`own_mutate.py`, isolated copies,
per-plant raw logs, exits in `own-mutants.json`) — 4/4 KILLED by the
named property, harmless control SURVIVED:

| Mutant | Narrowing | Killed by |
| --- | --- | --- |
| R1-tiebreak-epoch1 | lease tie-break inverted only at epoch 1 | `TestOwnershipWinnerIsTupleMaximum`, exit 1 |
| R2-drop-last-loser | loser dropped only as last union entry | `TestOwnershipLoserHistoryPreserved`, exit 1 |
| R3-gate-clock-tie | absolute year refused only on epoch-tie path | `TestOwnershipGateClockNonAuthority`, exit 1 |
| R4-gate-epoch1-second-owner | second owner minted only for epoch 1 | `TestOwnershipGateSingleAuthorizedOwner`, exit 1 |
| C-harmless-comment | comment only, applied | SURVIVED, exit 0 |

Method note: R3 v1 (authorize-in-2027) SURVIVED with root cause — the
expiry arm upstream of the tie path masks authorization plants for
lapsed ages and fresh ages are translation-identical. R3 v2 refuses on
the tie path in 2027 and is killed. The masking arm itself is covered
by the producer's `N-expiry-absolute`.

Producer battery rerun twice by the reviewer on the exact tree, both
runs identical: 8/8 `N-` plants KILLED by their named properties,
`C-harmless-comment` SURVIVED (applied), `C-not-applied`
NOT_APPLIED, `C-compile-failure` COMPILE_OR_HARNESS_FAILURE, both
`TestOwnership` controls exit 0, harness exit 0
(`prod-run1/`, `prod-run2/` logs + `mutants-full.json`).

No source-text gate is shipped by this leaf (only `sort`/`Compare`
logic and table assertions), so no token-preserving `T-` mutant is
owed; the P2 property asserts both the searched-for token (loser named
in the detail) and the behavior (winner is the maximum), and every
harness executes the behavioral property, never a static check.

AC coverage: **18 of 18 rows driven** through the named production
entries (matrix spot-verified row-family by row-family: rows 1–10 by
the 11 green properties, 11–13 by the green package suites including
landed crash/idempotency tests, 14 by the claims posture + README
disclaimers, 15 by the green P3 vector, 16–17 by green tracecheck,
18 by the rerun batteries). Disclosed bounds (grant age authoritative,
first-malformed-entry naming, no `ax`/`doctor`/runtime claim) are
accurate and tested where behavioral. Crash/idempotency: the changed
derivation is pure (no new durable mutation); landed
`TestCrashBefore/AfterLease*` green inside the sessrepo package run.

## 4. Story-close items — all confirmed

- P3 vector `epoch_zero_other_lease` (`{session A, epoch 0, valid third
  lease}` -> `invalid_arguments`) present in `gate_test.go` and green
  across all five entries.
- section:17.2 gap reworded to the narrower reader-enum meaning
  (diff inspected; tracecheck green).
- Three acceptance cases (`lease-ownership-union/store/gate-properties`)
  bound to existing driven declarations (`sessstate.Reduce`,
  `sessrepo.WinningLease`, `fencing.Authorize`); attached to 5.3#6,
  5.3#7, 2.2#1, 2.2#2, 2.2#4 with unchanged ratios (5.3 7/8 with #5
  open, 2.2 4/22). Digest re-pinned to
  `63c83796…0e121`; `tracecheck` exit 0 reproduces
  `acceptance_cases=113`, `clauses_discharged=28/485`.
- Self-minted-claim probe: appending
  `reviewer-self-minted-claim` in an isolated copy fails tracecheck
  (exit 1, digest `5bbcf4ab…` vs reviewed `63c83796…`).
  Sampled clause lines (2.2#1@408, 2.2#2@410) match the pinned text.
- README: property-suite section, `ownership properties` harness row,
  110->113 figure, trunk `-adopted` text preserved; no CLI/doctor/
  capability claim (explicit disclaimers; claims posture unchanged).

## 5. Hygiene and gates — rerun by the reviewer

- `gofmt -l internal` empty; `go vet` (touched trees) exit 0;
  `git diff --check` exit 0; `cataloggen -adopted -check` exit 0.
- `go test -count=1`: sessstate ok (71s), sessrepo ok (48s), fencing
  ok, traceability/... ok (2.7s + 28.7s), sessquery ok (121s).
- `go test -race -count=1`: sessstate/sessrepo/fencing ok, no
  DATA RACE; traceability/... ok, no DATA RACE.
- Full 27-package `go test ./...` sweep (normal + race + cover)
  accepted from the attached CR rev2 validation log; every
  review-scope package was rerun here as listed above.

## Reran vs accepted

Reran myself: RED/GREEN pair, 4 own narrowing mutants + control, full
producer 8-plant battery twice, P3 vector, tracecheck (+ self-mint
refusal), gofmt/vet/diff-check/cataloggen, all review-scope package
suites (normal and -race), enumeration-count capture. Accepted from
attached evidence: full `./...` 27-package sweep, coverage figures,
fuzz smoke, and fixture matrices in the CR rev2 validation log
(touched-package equivalents rerun here).

No product edits made. Candidate left UNCOMMITTED as required.
