# TASK-260830-kkh1an — producer results (rev4 rework)

Rework of CR revision 3 after RUN-260918-f6b44c (claude-opus-5 max):
CHANGES REQUESTED with no P1 — one P2 production finding (F3), one
P2 evidence finding (E3), five P3. Everything else from rev3 stands:
the reviewer confirmed F1/F2/E1/E2 closed with their own instruments
(including real SIGKILLs at two seams), the shipped harness 100/100
twice, the 84-of-84 ratio re-derived, the full suite green, and
hygiene clean. This run keeps all of it and closes every finding
below. Trunk base `c3aae73` (unchanged since rev3; no refresh needed —
`origin/main` still `c3aae73` at handoff), SPEC
`internal/specdoc/SPEC.v0.7.0.md`, same heading numbers (§13.7 added
to the cited scope for the cold force-takeover rule). The v0.7.0
adoption figures are unchanged. `internal/traceability` untouched per
the Story contract (final leaf TASK-260830-2056mm carries the
registry bindings and the re-pin); `internal/fencing` gained the
reviewer-sanctioned exported verdict plus its tests and one doc
sentence, nothing else. Candidate UNCOMMITTED in the Story worktree
for handoff snapshot.

Inherited state: this run started from the rev3 candidate tree in the
Story worktree (`M LOGBOOK.md`, `M README.md`,
`M internal/axpane/decide.go`, `M internal/terminalbackend/…` ×4,
`?? internal/terminstance/` — the reviewer-confirmed rev3 delta).
All rev3 production code, tests, and harness rows were kept except
the two rows reworked below; the changes are the only delta.

## Finding-by-finding table

| Finding | Change (production call site) | Test that fails without it | Evidence path |
| --- | --- | --- | --- |
| F3: grant-less stale members never fenced (`repeat-of: rev2 F2`) | New landed `fencing.StaleRelativeToWinner` (`internal/fencing/staleness.go`): the direction/tuple arms over a verified well-formed winner without the grant precondition; `ObserveFencing` (`fencing.go`) consults it on every non-park outcome and fences on decided-stale from a live source. No local tuple comparison; the relative question needs no fallback | `TestRV3F3_GrantLessStaleFences` (48 transitions; fails pre-fix: 48 no-transition — verified by reverting the composition: 49 `--- FAIL` lines) | `terminstance-test-v.log`, `mutants/pass{1,2}/N-fencing-verdict-*.log`, `N-fencing-verdict-composition.log` (12 rows) |
| E3: per-effect recheck pinned on the tuple axis only (`repeat-of: rev2 E2`) | No production change (the loop re-reads `Now()` per effect); committed witness + narrowing row. Both E1 expired members re-keyed onto the late-deadline instant so they kill on the receipt | `TestRV3W_M1_AuthExpiryRecheckedBeforeEachEffect` (fails under the entry-instant mutant: everything commits, `error = nil`) | `terminstance-test-v.log`, `mutants/pass{1,2}/N-engine-recheck-auth-expiry.log` |
| P3-1: evidence bound edge unwitnessed | No production change (shared `checkSortedUniqueEvidenceWhere`); committed witness + narrowing row | `TestRV3W_M2_EvidenceBoundEdge` (fails under the 257 mutant on both surfaces) | `mutants/pass{1,2}/N-result-evidence-bound.log` |
| P3-2: conditional disposition unpinned at engine | No production change (`DispositionFor` mapping existed); committed witness + narrowing row | `TestRV3W_M3_ConditionalCapabilityDisposition` (fails under the collapsed mutant: `status_first` ≠ `required_operator_action`) | `mutants/pass{1,2}/N-engine-conditional-disposition.log` |
| P3-3: last_operation_id grammar unwitnessed | No production change (the arm existed); committed witness + narrowing row | `TestRV3W_M8_StatusLastOperationGrammar` (fails under the empty-admitting mutant) | `mutants/pass{1,2}/N-status-last-operation-empty.log` |
| P3-4: proof-kind sibling admitted under narrowing | No production change (single `ParseProviderProofKind` arm); committed witness + narrowing row + sibling corpora on all three enum tests | `TestRV3W_M4_ProofKindSiblingNamesRefused` (fails under the sibling-admitting mutant at both entries) | `mutants/pass{1,2}/N-proof-sibling.log` |
| P3-5: partial-progress create member unstated | TRACEABILITY bound only: the interim-proven member refuses uncertain with the key parked; exit route `unavailable → terminate-stale` (final leaf); the absent-proven member resumes | — (bound, not behavior; characterized by the reviewer's `TestRV3_PartialCreateProgressIsNotResumable` probe) | TRACEABILITY.md stated bounds |
| Retired `N-fencing-epoch-only` | Harness-only: provably equivalent post-fix (proof in TRACEABILITY mutant-table intro) — a stale_owner park implies a decided-stale verdict over the same observation and both arms call `fenceLive` with the same refusal, so no test can distinguish the weakened park arm | — (retired; the mismatch fact stays measured by `N-fencing-verdict-winner` + the losing-lease members of `TestRV3F3_GrantLessStaleFences`) | TRACEABILITY.md mutant table |
| Re-keyed `N-fencing-operation` | Harness-only: same plant (direct question through `mutation`), new killer `TestObserveFencingNonLiveSourcesLeaveState` — the launch-class requirement now shows on the surfaced park vocabulary | `TestObserveFencingNonLiveSourcesLeaveState` (fails under the plant: hard refusal instead of the stale_owner park) | `mutants/pass{1,2}/N-fencing-operation.log` |

Reviewer-mutant mirrors: all five surviving rev3 reviewer narrowings
(RV3-M1/M2/M3/M4/M8) are mirrored as committed harness rows with
committed killers (`N-engine-recheck-auth-expiry`,
`N-result-evidence-bound`, `N-engine-conditional-disposition`,
`N-proof-sibling`, `N-status-last-operation-empty`); RV3-M6/M7 were
already killed by the committed suite. The three reviewer controls
map to the existing `N-auth-epoch-low` kill, build-break ERROR
detection, and the `C-control` SURVIVED control.

## F3: members enumerated and why the enumeration is complete

The stale class is factored into three questions: (1) which
observations REACH the verdict, (2) which tuples the verdict calls
stale, (3) which sources fence.

(1) The verdict is consulted on every non-park outcome of the direct
question. For a stale tuple over a verified well-formed winner, the
only `Authorize` outcomes that are not a stale/remote park are the
four grant-precondition refusals — no grant, no clock reading,
lapsed grant, unusable policy — because the ownership arms all pass
by construction and these are all four refusals the gate can surface
after them (gate arms at `gate.go:338–349`, in order). All four are
driven to fence from active/parked/quiescing under local AND remote
winners (the grant arms precede direction, so a remote winner with
no grant refuses grant-gated rather than parking remote — the
fallback covers it at the first question). The relative question
needs no fallback: it carries the direct observation's grant facts
unchanged and the grant arms do not read `LocalHostID`, so a direct
remote park implies the relative question passes the grant arms.
Completeness is structural: the verdict never consults
grant/clock/policy, so no fifth grant member can hide behind it.

(2) Decided-stale (older epoch, same-epoch lease mismatch) fences;
decided-not-stale (winning tuple, future epoch) and undecidable
(malformed presented/session/epoch/lease, session mismatch, no
winner, malformed winner, unverified, ambiguous, failed handoff —
the gate's pre-grant arms mirrored one by one) surface the refusal
with no transition. The failed-handoff park is deliberately
preserved: handoff history is ownership, not a grant precondition.

(3) Creating fences grant-less with the other live states;
absent/stopped/stale_fenced/unavailable surface the grant refusal
(local and remote).

## E3: members enumerated and why the enumeration is complete

The per-effect authorization recheck has three axes: kind binding,
expiry, and the lease tuple. The kind axis is fixed per operation
(no mid-operation rotation is expressible — the context carries one
kind); the tuple axis was pinned at positions 1–3 in rev3
(`N-engine-recheck-auth`, `-third`); the expiry axis is pinned here
at the second stop effect with the deadline still ahead
(`TestRV3W_M1`, killed by `N-engine-recheck-auth-expiry`, which keeps
the tuple recheck per-effect and freezes only the expiry instant).
The generation recheck has one axis and was pinned at the same
positions. The E1 `expired` members now kill on the receipt as well:
with the deadline past the expiry, an entry-exempting mutant binds
the receipt before the recheck refuses, so only the receipt
assertion distinguishes it (all three create subtests kill on
`Lookup found a receipt`, verified by hand-applying the mutant).

## AC coverage: 93 of 93 rows driven

Measured, not restated: 93 data rows counted in
`internal/terminstance/TRACEABILITY.md` (rows 1–84 keep rev3 numbers;
rows 85–93 are the rework rows), 109 cited top-level tests all
present via `go test -list` (107 in-package + the fencing agreement
test + the `terminalbackend` column test), every cited subtest
`--- PASS`, zero missing (see `ratio-check.log`); the full package
suite is green (`terminstance-test-v.log`: exit 0, 111 `--- PASS`,
0 `--- FAIL` — the 4 in-package tests not cited in the matrix are
supporting positive/store-level pins, as in rev3). Production call
sites are named per row in the matrix and per clause in TRACEABILITY.md.
The five `internal/fencing` verdict tests are the verdict's own
contract (`fencing-test-v.log`: exit 0, 47 `--- PASS`); every verdict
arm is additionally killed through `ObserveFencing`, so the
composition — not the helper — is measured.

## Negative and mutant evidence

- `mutant_harness.py`: 116/116 rows match on two full passes with
  identical verdict lists — 114 narrowing KILLED + 1 tightening edge
  KILLED + the SURVIVED control. The tightening row
  (`N-auth-epoch-high`) is unchanged and documented; the retired row
  (`N-fencing-epoch-only`) is documented with its equivalence proof.
  Per-plant raw logs with subprocess exits in `mutants/pass1/` and
  `mutants/pass2/` (116 plants each, `PYTHONDONTWRITEBYTECODE=1`, no
  `__pycache__`); production blobs verified byte-identical
  before/after/across passes (`blob-*.txt`, covering
  `internal/terminstance/*.go` and `internal/fencing/staleness.go`).
- Crash/idempotency: carried from rev3 (no new durable write — the
  verdict and the recheck are pure over their inputs, and the
  evidence-bound/disposition/grammar arms are pure gates). The rev3
  no-replace + fsync + hooks + real-SIGKILL evidence on both the
  quiesce and create seams stands unchanged.
- No token-preserving source-text mutant applies: no gate inspects
  source text. (`TestRV3_FencingNoLocalTupleComparison` is a
  composition pin, not a gate: it asserts the absence of tuple
  references in `fencing.go`, and the verdict rows measure the
  composed behavior through the entry.)

## Validation suite (configured commands, in order)

Every command below is quoted from `task-board.config.json`
`spawn.worktree_isolation.validation.commands` (30 commands) and was
executed once against the exact candidate tree; logs in `cmd/`.
Command 5 ran as 4 bounded chunks with identical flags (packages
1–10/11–20/21–30/31–40; each package exactly once, no flags
shortened) because a single shell call is time-bounded; see
`cmd05-summary.txt` and `cmd05-r1..r4.log`.

| # | Command | Exit |
| --- | --- | --- |
| 1 | `test -z "$(gofmt -l $(git ls-files --cached --others --exclude-standard -- '*.go'))"` | 0 |
| 2 | `go build ./...` | 0 |
| 3 | `go vet ./...` | 0 |
| 4 | `go test ./... -count=1 -v` | 0 (40/40 packages ok) |
| 5 | `go test ./... -race -count=1 -timeout 25m` | 0 (4 chunks, 40/40, 0 DATA RACE) |
| 6 | `go test ./... -cover -count=1` | 0 (40/40; terminstance 86.0%, fencing 97.8%) |
| 7–10 | rpcwire fuzz ×4 (`-fuzztime=100x`) | 0 ×4 |
| 11 | scalar fuzz ×1 | 0 |
| 12–15 | canonicaljson fuzz ×4 | 0 ×4 |
| 16–23 | secconftest fuzz ×8 | 0 ×8 |
| 24 | `go run ./internal/traceability/cmd/tracecheck` | 0 (contracts=64, cases=140, clauses 56/569 — trunk figures unchanged) |
| 25 | `go run ./internal/catalog/cmd/cataloggen -adopted -output internal/catalog/catalog_gen.go -check` | 0, tree unmodified |
| 26 | `GOOS=linux GOARCH=amd64 go build ./...` | 0 |
| 27 | `GOOS=windows GOARCH=amd64 go build ./...` | 0 |
| 28 | `git ls-files -z '*.json' \| xargs -0 -n1 python3 -c 'import json,sys;json.load(open(sys.argv[1]))'` | 0 |
| 29 | `task-board validate` | 0 (zero issues name this task or story) |
| 30 | `git diff --check` | 0 |

An extra `GOOS=windows go vet` of the touched packages also exits 0
(`cmd27b-windows-vet.log`; not a configured command). The `.go` tree
was verified byte-identical before and after the race gate
(`go-sha-before-race.txt` == `go-sha-after-race.txt`), so the race
ran against the exact candidate.

## Files changed (rev4 delta on top of the rev3 tree)

- `internal/fencing/staleness.go` (new): exported
  `StaleRelativeToWinner` verdict; `staleness_test.go` (new): tuple
  arms, grant-independence, remote direction, undecided members,
  gate agreement; `doc.go`: one verdict sentence. Nothing else in
  the landed package is touched.
- `internal/terminstance/fencing.go`: verdict fallback composition
  + doc; `fencing_grantless_test.go` (new): 48-transition regression
  test, decided-not-stale keeps, 9-member undecided keeps,
  non-live verdict-path pins; `review_witness_test.go` (new): the
  five reviewer witnesses verbatim + the no-local-tuple
  composition pin.
- `internal/terminstance/`: late-deadline fixture
  (`fixtureLateDeadline`, `fixturePastExpiry`,
  `testContextWithDeadline`, `testCreateContextWithDeadline`) with
  both E1 expired members re-keyed to kill on the receipt;
  sibling-vocabulary corpora on all three enum tests;
  `mutant_harness.py` 100 → 116 rows (17 new, 1 retired, 1
  re-keyed); `TRACEABILITY.md` 84 → 93 rows.
- `README.md`: grant-less fencing sentence, §13.7 scope, battery
  count 98 → 114 narrowing rows.
- `LOGBOOK.md`: rev4 rework entry (additive, newest-first).
- `internal/traceability` untouched per the Story contract.

## Stated bounds (no unsupported claims)

No `ax` command, doctor result, or runtime capability is added. All
rev3 bounds stand. Newly stated: the interim-proven create member
refuses uncertain with the key parked and exits only through the
final leaf's `unavailable → terminate-stale` path (P3-5); the
failed-handoff park is preserved by the verdict (ownership history,
not a grant precondition); the retired row's equivalence (a stale
park implies a decided-stale verdict) with its proof. The per-code
disposition mapping and the AX-side seam emission codes remain the
leaf's contract where the spec pins the vocabulary but not the
mapping.

## Handoff

Ready for review. Candidate UNCOMMITTED in the Story worktree;
checklist updated on the board; outcome resources attached:
`TASK-260830-kkh1an_results.md`,
`TASK-260830-kkh1an_conformance-matrix.md`,
`TASK-260830-kkh1an_producer-evidence.tar.gz`.
