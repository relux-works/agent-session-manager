# TASK-260830-1esv6u — implement-validation-report-and-reconciliation (developer handoff, rev3 rework)

Story-final leaf of STORY-260830-21bxa3. Rework answering CR rev2
verdict `TASK-260830-1esv6u_review-verdict-rev2.md` (three findings:
source-evidence-chain-mismatch P1, reconciliation-product-not-enumerated
P2, story-cases-have-no-clause-edges P2) per brief
`TASK-260830-1esv6u_rework-rev3.md`. Candidate left UNCOMMITTED in
the worktree. Trunk still `5b7876b`: no refresh (already current).

## What changed in this rework

1. **P1 — per-row source-evidence chain binding** (`internal/clonereconcile/reconcile.go`):
   `Reconcile` now binds every non-synthesized row's `SourceEvidenceIDs`
   digest to `rawByCandidate[row.SourceItemKey]` and refuses, each with a
   literal code and detail: swapped (`... claims source evidence %q of
   candidate %q`), foreign (kept `... claims unreconciled source evidence
   %q`), missing (`... has no source evidence`, scanned before the fidelity
   build so it reports the tier-3 literal instead of the owner shape
   refusal), double-claimed (`... is claimed by rows %q and %q`, censused
   before the per-row checks so a shared digest reports the double claim
   while a pure swap falls through to the chain check). The four relation
   breaks are the oracle rule O9, enumerated across the full product via
   four rotating faults (~7.2k applications each in LargeN), not hand-picked
   cells.
2. **P2 — full enumeration**: every capture-class vector N=0..6 (19,531
   vectors) × all nine read-back outcomes = 175,779 cells (1,404 N≤3 +
   174,375 N=4..6), each running the production `Reconcile` entry against
   the independent oracle. Every shard asserts its executed count equals
   5^vectors × 9 computed from the class/outcome vocabularies, asserts
   vector uniqueness (5^k distinct vectors over 5 symbols is the whole
   space), and asserts every vector ran all 9 outcomes. Synth count, fault
   (none + 23), and validity variant rotate per cell. New narrowing
   `N-t3-doublechain-prefix` (double-claim census over a three-row prefix)
   is invisible on every N≤3 input — the N≤3 subset passes on it
   (`N-t3-doublechain-prefix-subset.log`, exit 0) — and is killed by the
   N≥4 enumeration alone.
3. **P2 — registry clause edges (orchestrator decision)**: no clause edges
   fabricated. All five Story cases stay bound at the section level
   (`section:13.14.2` acceptance_cases, decoded and proven present); the
   zero extracted clauses is now MEASURED by running the production
   extractor over the pinned document (not a registry self-assertion);
   new test `TestStoryFinalFidelityClauseEdgesWhereRepresentable` fails
   when a section with ≥1 extracted clause owns a Story case without a
   clause edge (positive control on real 11.4 data + control plant); the
   clause-model limitation is a stated bound owned by the traceability
   clause extractor (`sectionClauseInventory` recognizes RFC 2119 keyword
   lines only) in TRACEABILITY.md, this note, and LOGBOOK. Registry bytes,
   digest (`ab7b66cc...`), tracecheck figures, and README pins unchanged —
   re-derived equal, pin tests green at all sites.
4. **Harness integrity (own finding during rework)**: four carried rows
   were misreported kills — `T-loss-branch` died at `go vet` (`suspect
   and`) and the three tuple narrowings were syntax errors
   (unparenthesized composite literal in an `if` condition), so their
   killer tests never executed. Fixed: `T-loss-branch` now misclassifies
   exact rows as loss (both tokens kept, behavioral kill on rows[0]);
   tuple narrowings parenthesized (behavioral kills, one full admission).
   All 72 logs re-verified free of `build failed`: every kill is a test
   failure.

## AC coverage ratio: 6 of 6 rows driven

| # | AC row | Production call site | Named committed test |
|---|---|---|---|
| 1 | Read back staged/live target history | `clonereconcile.ReadBackHistory` | `TestReadBackHistoryAdmitsSealedPair`, `TestReadBackHistoryRefusals`, `TestReadBackHistoryEmptyInput` |
| 2 | Prove native evidence → canonical item → target evidence or explicit loss for every item | `clonereconcile.Reconcile` | `TestReconciliationProperty` (1,404 cells), `TestReconciliationPropertyLargeN` (174,375 cells), `TestReconcileAdmitsValidTargetHistory`, `TestReconcileSourceEvidenceChainBinding` |
| 3 | Exact contract fixtures pass | `Reconcile` (owner-sealed fixtures) | `TestReconcileAdmitsValidTargetHistory` (independent fidelity-ID recompute, owner decode-back of both derived reports) |
| 4 | Negative/refusal cases pass | `Reconcile`, `ReadBackHistory` | `TestReconcileGateRefusals` (68 rows, every refusal literal exact) |
| 5 | Crash/idempotency evidence when the operation mutates durable state | N/A — pure entries, no durable write | Vacuous-with-proof: `TestReconcileDeterminism` (10× byte-identical), `TestReconcilePurity` (inputs untouched) |
| 6 | No unsupported capability advertised | no new advertisement surface | Importer grid (TRACEABILITY.md): new package only, zero landed edits, all owner suites green; README/registry claim only measured coverage |

## Mutant evidence (67 rows, harness `internal/clonereconcile/testdata/mutant_harness.py`)

Every narrowing weakens one gate to admit exactly one member of the
class it must reject; the killer runs ALONE through a production entry
and must fail. Token-preserving rows additionally run the full package
suite. One raw log per plant per run rides the evidence tar (72 logs:
67 + 4 suite runs + the N≤3 subset-survival log). Every log verified
free of `build failed`: every kill is a behavioral test failure.

Verdict summary: 62 narrowings KILLED, 4 token-preserving KILLED (alone
and under suite), 1 control SURVIVED, 0 ERROR. Full table:

| Mutant | Narrows the gate to | Named failing test |
|---|---|---|
| N-scope | admit archive scope only | `TestReconcileGateRefusals/scope-archive` |
| N-history-staged | admit a live read in the staged slot only | `.../staged-not-staged` |
| N-history-live | admit a staged read in the live slot only | `.../live-not-live` |
| N-wrap-staged | admit the empty staged document only | `TestReadBackHistoryEmptyInput/empty-staged` |
| N-wrap-live | admit the empty live document only | `TestReadBackHistoryEmptyInput/empty-live` |
| N-wrap-plan | admit empty plan bytes only | `TestReconcileGateRefusals/plan-empty` |
| N-wrap-projected | admit empty projected bytes only | `.../projected-empty` |
| N-wrap-tuple | admit the empty tuple only | `.../validation-tuple-empty` |
| N-wrap-expected-parse | admit the empty expected digest only | `.../expected-fidelity-empty` |
| N-wrap-seal | admit the not-json findings member only | `.../validation-malformed` |
| N-cand-utf8 | admit the bad-ff key only | `.../candidate-utf8` |
| N-cand-bounds | admit length 513 only | `.../candidate-too-long` |
| N-cand-dup | admit the cand-0 duplicate only | `.../candidate-duplicate` |
| N-t1-undeclared | admit the ghost link only | `.../tier1-undeclared` |
| N-t1-twice | admit the cand-0 second link only | `.../tier1-twice` |
| N-t1-missing | excuse the cand-0 link only | `.../tier1-missing` |
| N-t1-both | admit the durable_payload both-arms link only | `.../tier1-both` |
| N-t1-neither | admit the cand-0 neither-arms link only | `.../tier1-neither` |
| N-t1-raw-digest | admit the bogus tier-1 raw only | `.../tier1-raw-not-digest` |
| N-t1-raw-twice | admit the raw-0 second claim only | `.../tier1-raw-twice` |
| N-t1-exclusion | admit the frobnicate exclusion only | `.../tier1-bad-exclusion` |
| N-t2-raw-digest | admit the bogus tier-2 raw only | `.../tier2-raw-not-digest` |
| N-t2-unknown | admit the phantom raw link only | `.../tier2-unknown-raw` |
| N-t2-twice | admit the raw-0 second link only | `.../tier2-twice` |
| N-t2-both | admit the omitted both-arms link only | `.../tier2-both` |
| N-t2-neither | admit the raw-0 neither-arms link only | `.../tier2-neither` |
| N-t2-canon-digest | admit the bogus canonical only | `.../tier2-canon-not-digest` |
| N-t2-canon-twice | admit the canon-0 second claim only | `.../tier2-canon-twice` |
| N-t2-normalization | admit the frobnicate normalization only | `.../tier2-bad-normalization` |
| N-t2-synth | admit synthesized for raw-2 only | `.../tier2-synth-normalization` |
| N-t2-missing | excuse the raw-0 link only | `.../tier2-missing` |
| N-t3-synth | admit the raw-0 synth claim only | `.../tier3-synth-claims` |
| N-t3-unknown | admit the cand-0x row only | `.../tier3-unknown-key` |
| N-t3-needs-canon | skip the cand-0 null-canonical row only | `.../tier3-needs-canonical` |
| N-t3-agreement | admit the (canon-1, canon-0) pair only | `.../tier3-wrong-canonical` |
| N-t3-under-norm | admit the phantom normalized canonical only | `.../tier3-canon-under-normalization` |
| N-t3-unreconciled | admit the phantom source claim only | `.../tier3-unreconciled-evidence` |
| N-t3-swapped | admit the raw-1 swap member only (refusal moves to the mirror row) | `.../tier3-swapped-evidence` |
| N-t3-no-source | admit the cand-0 empty source list only (owner shape refusal changes the literal) | `.../tier3-missing-evidence` |
| N-t3-double | admit the raw-0 double claim only (chain check then reports the swap literal) | `.../tier3-double-claimed-evidence` |
| N-t3-doublechain-prefix | census only the first three rows; invisible at N≤3, killed by the N≥4 enumeration alone | `TestReconciliationPropertyLargeN` (alone); N≤3 subset passes (`N-t3-doublechain-prefix-subset.log`) |
| N-t3-loss-staged | admit the single staged-0 loss claim only | `.../tier3-loss-claims-staged` |
| N-t3-loss-live | admit the single live-0 loss claim only | `.../tier3-loss-claims-live` |
| N-t3-no-staged | excuse the cand-0 staged gap only | `.../tier3-no-staged` |
| N-t3-no-live | excuse the cand-0 live gap only | `.../tier3-no-live` |
| N-t3-missing | excuse the cand-0 row only | `.../tier3-missing-row` |
| N-t3-staged-outside | admit the phantom staged claim only | `.../tier3-staged-outside` |
| N-t3-live-outside | admit the phantom live claim only | `.../tier3-live-outside` |
| N-plan-staged | admit the (plan-A, plan-B) pair only | `.../staged-plan-mismatch` |
| N-plan-live | admit the (plan-A, plan-B) pair only | `.../live-plan-mismatch` |
| N-projected-staged | admit the (proj-A, proj-B) pair only | `.../staged-projected-mismatch` |
| N-projected-live | admit the (proj-A, proj-B) pair only | `.../live-projected-mismatch` |
| N-native-sessions | admit the other-session live read only | `.../reads-differ-sessions` |
| N-plan-expects | admit other-session reads only | `.../plan-expects-other` |
| N-projected-expects | admit other-session reads only | `.../projected-expects-other` |
| N-fid-staged-bind | admit the phantom staged binding only | `.../fidelity-staged-unbound` |
| N-fid-live-bind | admit the phantom live binding only | `.../fidelity-live-unbound` |
| N-fid-tuple-staged | admit the relux.other fidelity target only | `.../fidelity-tuple-staged` |
| N-fid-tuple-live | admit the relux.other live observation only | `.../fidelity-tuple-live` |
| N-validation-tuple | admit the relux.other report target only | `.../validation-tuple-staged` |
| N-validation-native | admit the other-session expectation only | `.../validation-expects-other` |
| N-expected-fidelity | admit the phantom expectation only | `.../expected-fidelity-mismatch` |
| T-mode-staged | TOKEN KEPT (`staged`): reports the live detail | `.../staged-not-staged` (alone + suite) |
| T-loss-branch | TOKENS KEPT (`omitted`,`unrecoverable`): exact rows misclassified as loss, gate test fails on rows[0] | `.../tier3-loss-claims-staged` (alone + suite) |
| T-tier2-synth | TOKEN KEPT (`synthesized`): unreachable arm | `.../tier2-synth-normalization` (alone + suite) |
| T-tier3-synth | TOKEN KEPT (`synthesized`): unreachable arm | `.../tier3-synth-claims` (alone + suite) |
| C-doc-comment | SURVIVED (control) | bound: comment-only edit changes no behavior; survival proves the harness observes outcomes instead of hard-coding kills |

Non-gate sites without mutants (documented in TRACEABILITY.md): the two
`fidelity report invalid` forwarding wraps decide nothing (the owner
decides); any weakening still refuses with the identical literal via the
decode fallback, so no verdict change is expressible. Covered by the owner
suite plus the wrap tests.

## Out-of-contract rows (silence claims coverage; these do not)

| Row | AC clause making it so |
|---|---|
| §13.14.4 (Transaction and target Checkpoint) unbound | Scope names the section but no story leaf implements Materialization Plan 2 / Clone Projection; the AC's scoped deliverable is reconciliation per §13.14.2 — binding it would be an unsupported claim |
| Operation/bundle constancy across documents | Stated in §13.14.3, outside the §13.14.2 story boundary named in Scope |
| Plan-workspace vs read-workspace comparison | No pinned sentence requires the pairing; stated bound in TRACEABILITY.md |
| Parsed counts/heads/structural-digest derivation | Bounded non-derived by the read-back leaf; reconciliation pairs identities, not parsed content |
| Capture-manifest-internal reconciliation | Owned by `clonebundle.VerifyCaptureReconciliation`; this leaf censuses caller-projected key sets |
| Clause-level traceability edges for the five Story cases | Orchestrator decision: the traceability clause extractor recognizes RFC 2119 keyword lines only and §13.14.2 yields zero extracted clauses (measured); section-level binding proven + stricter rule enforced where representable + stated bound — no edge fabricated |

## Coverage map

The brief carries no surface table (`references/attack-surface-catalog.md`
rows are absent from the brief and the rework brief), reported here as a
brief gap per the handoff contract: there is no table row to map, so no
coverage-map.md row is owed. Gate coverage is instead measured by the
68-row gate table × 67-row mutant harness above.

## Rework diff (bounded)

Inside the leaf module + story-final registry test + LOGBOOK only:

- `internal/clonereconcile/reconcile.go` (chain binding, missing scan, double-claim census)
- `internal/clonereconcile/errors.go` (refusal comment)
- `internal/clonereconcile/gates_test.go` (3 gate rows + `TestReconcileSourceEvidenceChainBinding`)
- `internal/clonereconcile/property_test.go` (O9 + 4 relation faults + full N=0..6 × 9 enumeration + executable shard assertions)
- `internal/clonereconcile/TRACEABILITY.md` (axes, I-resolve/I-missing/I-double, C3b, matrix +3, clause-model bound, mutant counts)
- `internal/clonereconcile/testdata/mutant_harness.py` (4 new narrowings, 1 reworked T-row, 3 tuple paren fixes, 1 find disambiguation)
- `internal/traceability/registry_rederivation_test.go` (measured zero + stricter-rule test + control plant)
- `LOGBOOK.md` (rework entry, newest-first)

Unchanged by this rework (verified): `ownership.v0.7.0.json` bytes,
`reviewedOwnershipCanonicalSHA256`, tracecheck figures, README pins.
Everything outside the list above is byte-identical to the rev2
candidate: all eight owner packages diff-clean against checkpoint
`f416d54`, moved classes none.

## Verification (real exit codes, run in this session)

| Command | Exit |
|---|---|
| `go test ./internal/clonereconcile/ -v -count=1` | 0 |
| `go test ./internal/clonereconcile/ -cover -count=1` (99.2%) | 0 |
| `go test ./... -count=1` (51/51 packages, no FAIL) | 0 |
| importer + traceability suites (8 owner packages + traceability) | 0 |
| new gate rows + regression test `-count=3` | 0 |
| `go vet ./...` | 0 |
| `GOOS=windows GOARCH=amd64 go vet ./...` | 0 |
| `go build ./...` | 0 |
| `gofmt -l internal/` (empty) | 0 |
| `go run ./internal/traceability/cmd/tracecheck` | 0 |
| digest re-derivation probe (re-derived == reviewed `ab7b66cc...`) | equal |
| mutant harness (67 rows, batched; 62 KILLED + 4 T KILLED + 1 SURVIVED) | 0, all verdicts as expected |
| all 72 mutant logs free of `build failed` | clean |
| scratch-index exact-tree diff (8 modified + 12 new + known scratch dir, all intended) | clean |

Registry: digest `ab7b66cc6d85df8ed7eac2aa5cd7b7fa635db20a3e5092824490b435f94ed3f1`,
tracecheck `bindings=73 ... unmeasured=5 ... acceptance_cases=168`,
README fenced line byte-equal, pin tests green at five sites. Trunk rows
(STORY-260830-ub60id) carried untouched; rederivation suite green. Trunk
still `5b7876b`: no second refresh (refresh-candidate would refuse
`refresh_already_current`; not run on this rework).

Evidence tar `TASK-260830-1esv6u_producer-evidence.tar.gz` holds all logs
above plus the 72 mutant raw logs.
