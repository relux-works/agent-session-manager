# TRACEABILITY — internal/clonereconcile

Final leaf of STORY-260830-21bxa3 (M4): reconciliation completeness
and the Story registry (TASK-260830-1esv6u). This package proves
that every captured candidate reconciles once to raw evidence or
exclusion, every raw item to a canonical item or normalization
disposition, and every canonical item to staged/live target
evidence or a target disposition, by reading back staged/live
target history through the sealed read-back owner and deriving
both reports through their owners.

Authority quotes below come from
`internal/specdoc/SPEC.v0.7.0.md` (pinned v0.7.0). Line numbers are
1-based.

## Why this package exists

Section 13.14.2 states the reconciliation spine (lines
10554-10556) but no landed package owns the cross-tier census:
`clonebundle` owns capture-plan/object reconciliation inside the
Capture Manifest, `clonefidelity` owns row shape and aggregate
reconciliation inside one report, and `clonereadback` owns the
validation report with its sibling-seal and valid rules — while
the read-back leaf explicitly defers tuple cross-match and
plan/projected/fidelity pairing to this final leaf. This package
owns only that composition: the three-tier link census over
caller-projected key sets plus the target-evidence pairing
against sealed reads. It mints no shape, parses no document, and
decides no valid bit of its own.

## Landed-owner reuse

| Scope | Owner | Use |
|---|---|---|
| Read-back decode | `internal/clonereadback` | `DecodeReadBackEvidenceManifest` is the only read path, both modes |
| Validation report + valid bit | `internal/clonereadback` | `BuildValidationReport` (attempted true, then false; exactly one seals) |
| Fidelity rows + report | `internal/clonefidelity` | `BuildFidelityReport`, `DecodeFidelityReport`; tier-3 censuses owner-decoded rows |
| Plan/projected pairing inputs | `internal/cloneplan` | `DecodeProjectionPlan`, `DecodeProjectedObjectManifest` |
| Exclusion vocabulary | `internal/clonebundle` | `ValidCaptureClass` at every tier-1 exclusion |
| Normalization vocabulary | `internal/clonefidelity` | `ValidDisposition` at every tier-2 normalization |
| Digests | `internal/scalar` | `ParseDigest` at every raw/canonical/expected digest |
| Tuples | `internal/sessadapter` | `DecodeTuple` for the validation target tuple |
| Character string measure | `internal/environ` | `StringLength` at the candidate bound |

No behavior in any reused package changed: this leaf adds a new
package only and edits no landed file (importer grid below).

## Axis inventory (written first)

Every gate axis the conformance matrix walks, with its domain:

| Axis | Domain | Oracle |
|---|---|---|
| A1 capture class | exact, opaque_preserved, excluded, normalized, lost (omitted) per candidate, N 0..6 | every class vector enumerated N=0..6 (19,531 vectors); each class pins its tier arms and row shape |
| A2 synthesized rows | 0..1 extra rows per cell | rotates per cell; synth rows carry no canonical and no captured source evidence; admitted with and without; the shard asserts both values ran |
| A3 read-back outcome | staged × live in present/absent/mismatched (9 cells) | every vector crosses all 9 pairs (175,779 cells); claimed evidence must resolve into the read of its side |
| A4 injected fault | none + 23 single-break faults (drop/dup/both/neither per tier, double-claims, bad vocabularies, row drops/swaps/strips, ghost/duplicate/malformed keys, plus the 4 O9 source-evidence relation breaks: swapped, foreign, missing, double-claimed) | rotating fault per cell; production refusal must name a broken rule (membership); each fault additionally pinned exactly by its gate row; the double-claim fault targets the last two rows |
| A5 validity variant | nominal, flipped check, diverged observed ID, error finding | rotated per cell; valid bit predicted from the spec valid rule, never from production |
| A6 candidate hygiene | UTF-8 validity, 1..512 characters, uniqueness | empty/513/duplicate/invalid-UTF-8 refuse; 512 ASCII and 512 é admit |
| A7 tier-1 arms | raw digest XOR capture-class exclusion per candidate | both/neither/unknown/missing/duplicate refuse; bad digest and bad class refuse |
| A8 tier-2 arms | canonical digest XOR non-synthesized disposition per raw | both/neither/unknown/missing/duplicate refuse; synthesized normalization refuses |
| A9 tier-3 coverage | every included candidate and canonical in exactly one row; each row's source evidence is its own candidate's tier-1 raw; no raw double-counted across rows | unknown/excluded key, null-vs-named canonical mismatch, swapped/foreign/missing/double-claimed source evidence, missing row refuse |
| A10 tier-3 evidence | non-loss rows carry ≥1 staged + ≥1 live resolving ID; loss rows carry none | missing side, phantom ID per side, loss-with-evidence refuse |
| A11 pairing | plan/projected IDs, native sessions, tuples, fidelity references across reads and reports | per-side mismatch literals; tuple cross-match both reports |
| A12 valid decision | owner-decided bit over 7 checks, ID equality, findings | true-then-false seal; all four validity shapes seal with the owner bit and decode back |

Structural arguments. The capture-class × read-back product is
fully enumerated, not sampled: every shard asserts its executed
count equals 5^vectors × 9 computed from the class and outcome
vocabularies, asserts vector uniqueness (5^k distinct length-k
vectors over 5 symbols is the whole space), and asserts every
vector ran all 9 outcomes. The auxiliary axes rotate rather than
cross: synthesized rows alternate per cell (both values asserted
per shard), faults rotate one per cell (each fault applies
thousands of times across the product; the shard logs the
applied/skipped histogram), and validity variants rotate per cell
(the seal step runs last, independent of tiers) plus the dedicated
four-variant test. Tiers compose into an injective chain (I-chain
below): tier-1 raw keys are unique, tier-2 canonical targets are
unique, and each tier-3 row agrees with its candidate's chain.

## Interpretation record (decided, pinned)

Readings taken where the pinned text underdetermines the gate.
Each is pinned by a named test, not prose.

| # | Reading | Test |
|---|---|---|
| I-chain | Each tier link is injective: raw keys unique across tier-1 links and canonical targets unique across tier-2 links. Many-to-one at tier 2 cannot satisfy "exactly one non-synthesized disposition row" per canonical with one row per candidate, so it refuses as double-counted. | `TestReconcileGateRefusals/tier1-raw-twice`, `.../tier2-canon-twice` |
| I-loss | The two target dispositions are omitted and unrecoverable ("explicit loss"). Loss rows carry empty staged/live sets: evidence plus a loss disposition would reconcile the item twice. | `TestReconcileGateRefusals/tier3-loss-claims-staged`, `.../tier3-loss-claims-live` |
| I-norm | A normalization disposition is any valid disposition except synthesized: a raw item cannot normalize into a sourceless canonical. | `TestReconcileGateRefusals/tier2-synth-normalization` |
| I-synth | Synthesized rows never claim captured source evidence: their source-evidence IDs are disjoint from the tier-1 raw set. Claimed staged/live IDs still resolve. | `TestReconcileGateRefusals/tier3-synth-claims` |
| I-resolve | Non-synthesized rows trace to captured source evidence: every source-evidence ID binds to the row's own candidate's tier-1 raw (a swapped digest reports its owning candidate, a foreign digest reports the unreconciled claim). Global membership is not enough. | `TestReconcileSourceEvidenceChainBinding` (swapped/foreign), `TestReconcileGateRefusals/tier3-swapped-evidence`, `.../tier3-unreconciled-evidence` |
| I-missing | A non-synthesized row with no source evidence is missing its candidate's tier-1 raw. The check runs before the fidelity build so the tier-3 missing rule reports its own literal instead of the owner's shape refusal. | `TestReconcileGateRefusals/tier3-missing-evidence` |
| I-double | No raw item is double-counted across rows: the first digest claimed by two non-synthesized rows refuses with both rows. The census runs before the per-row chain checks so a shared digest reports the double claim, while a pure swap falls through to the chain check. | `TestReconcileGateRefusals/tier3-double-claimed-evidence` |
| I-order | Candidates need uniqueness, not sortedness: the census sorts internally for determinism. Duplicate declarations refuse. | `TestReconcileGateRefusals/candidate-duplicate` |
| I-dupkeys | A second row for one candidate refuses at the fidelity build (owner key uniqueness), wrapped in the reconciliation refusal. | `TestReconcileGateRefusals/tier3-duplicate-keys-refused-by-owner` |
| I-dupcanon | A second row for one canonical refuses at the agreement gate: tier-2 injectivity makes it disagree with the chain. | `TestReconcileGateRefusals/tier3-duplicate-canonical-refused-by-agreement` |
| I-canmiss | An uncovered canonical always coincides with an uncovered candidate (one chain, one row), so the candidate census reports the shape. | `TestReconcileGateRefusals/tier3-missing-row` |
| I-valid | Pairing mismatches refuse (evidence does not belong together); within-report check failures seal with valid=false through the owner. The entry never computes valid: it attempts true, then false. | `TestReconcileValidFollowsOwner` (4 variants) |
| I-scope | Reconciliation is target-only: archive scope refuses before any tier runs. | `TestReconcileGateRefusals/scope-archive` |

## Clause map (spec sentence → gate → test)

| # | Pinned clause | Gate | Production site | Test |
|---|---|---|---|---|
| C1 | "Every captured candidate reconciles once to raw evidence or exclusion" (10554-10555) | tier-1 census | `checkTier1` | `TestReconciliationProperty` (O1 cells), `TestReconcileGateRefusals/tier1-*` (8 rows) |
| C2 | "every raw item to a canonical item or normalization disposition" (10555) | tier-2 census | `checkTier2` | property (O2 cells), `TestReconcileGateRefusals/tier2-*` (10 rows) |
| C3 | "every canonical item to staged/live target evidence or a target disposition" (10555-10556) | tier-3 census | `checkTier3`, `checkRowEvidenceResolves` | property (O3/O5 cells), `TestReconcileGateRefusals/tier3-*` (19 rows) |
| C3b | "Every captured candidate reconciles once to raw evidence" (10554-10555): each row's source evidence is its own candidate's raw | per-row chain binding, double-claim census, missing scan | `checkTier3` binding loop + census, `checkRowsCarrySourceEvidence` | property (O9 cells, 4 relation faults × ~7.2k applications), `TestReconcileSourceEvidenceChainBinding`, `TestReconcileGateRefusals/tier3-{swapped,unreconciled,missing,double-claimed}-evidence` |
| C4 | "a synthesized row never claims source evidence" (derived: synthesized requires no source canonical object; 10602-10603) | synth disjointness | `checkTier3` synth arm | `TestReconcileGateRefusals/tier3-synth-claims`, property synth axis |
| C5 | "Clone Validation Report 1.0.0 aggregates both reads" (10581, 10755) | sealed-pair order, sibling references by construction | `checkHistoryPair`, `sealValidationReport` | `TestReadBackHistoryAdmitsSealedPair`, `TestReconcileAdmitsValidTargetHistory` |
| C6 | "every applicable check must pass" (10582) + "valid=true requires every boolean true, matching native IDs/tuple, and no error finding" (10776-10777) | valid decided by the report owner | `sealValidationReport` (true-then-false) | `TestReconcileValidFollowsOwner`, property (O8 cells) |
| C7 | "equal expected and observed native Session IDs" (10741-10742) across reads, plan, projected, and report | native pairing | `checkNativePairing`, `checkValidationNativePairing` | pairing rows (4 literals) |
| C8 | observed/target tuple agreement across reads and both reports (10742, 10768; deferred by the read-back leaf) | tuple cross-match | `checkFidelityPairing`, `checkReportTuplePairing` | pairing rows (3 literals) |
| C9 | plan/projected/fidelity reference pairing (10740-10741, 10760-10765; deferred by the read-back leaf) | digest pairing | `checkPlanPairing`, `checkProjectedPairing`, fidelity binding, expected-fidelity agreement | pairing rows (8 literals) |
| C10 | "Modes cannot be relabeled" (10580) | authority-derived modes bind the pair | `ReadBackHistory` via the read-back owner | `TestReadBackHistoryRefusals` |

## Gate reachability matrix

Every production refusal site derived mechanically from the
`refuse("...")` call sites (`TestGateReachabilityMatrixDerivedFromRefusalSites`
fails on an unlisted site). Each row breaks only its gate and
asserts its code and detail at the named entry.

| Refusal literal | Entry | Gate test |
|---|---|---|
| `staged read-back invalid: %v` | `ReadBackHistory` | `TestReadBackHistoryRefusals` (garbage staged) |
| `live read-back invalid: %v` | `ReadBackHistory` | `TestReadBackHistoryRefusals` (garbage live) |
| `reconciliation staged read is not a staged manifest` | `ReadBackHistory`, `Reconcile` | `TestReadBackHistoryRefusals` (swapped), `TestReconcileGateRefusals/staged-not-staged`, `.../zero-history` |
| `reconciliation live read is not a live manifest` | `Reconcile` | `TestReconcileGateRefusals/live-not-live` |
| `reconciliation needs a target fidelity report` | `Reconcile` | `TestReconcileGateRefusals/scope-archive` |
| `projection plan invalid: %v` | `Reconcile` | `TestReconcileGateRefusals/plan-garbage`, `TestReconcilePlanDecodesThroughOwner` |
| `projected object manifest invalid: %v` | `Reconcile` | `TestReconcileGateRefusals/projected-garbage` |
| `fidelity report invalid: %v` | `Reconcile` | `TestReconcileGateRefusals/tier3-duplicate-keys-refused-by-owner`, property O7 cells |
| `validation target tuple invalid: %v` | `Reconcile` | `TestReconcileGateRefusals/validation-tuple-garbage` |
| `expected fidelity report is not a digest: %v` | `Reconcile` | `TestReconcileGateRefusals/expected-fidelity-garbage` |
| `expected fidelity report %q does not match the derived report %q` | `Reconcile` | `TestReconcileGateRefusals/expected-fidelity-mismatch` |
| `candidate key is not valid UTF-8` | `Reconcile` | `TestReconcileGateRefusals/candidate-utf8` |
| `candidate key is not a string[1..512]` | `Reconcile` | `TestReconcileGateRefusals/candidate-empty`, `.../candidate-too-long`, `TestReconcileMultibyteCandidateMeasure` |
| `candidates carry duplicate %q` | `Reconcile` | `TestReconcileGateRefusals/candidate-duplicate` |
| `tier-1 link names undeclared candidate %q` | `Reconcile` | `TestReconcileGateRefusals/tier1-undeclared` |
| `tier-1 candidate %q reconciles twice` | `Reconcile` | `TestReconcileGateRefusals/tier1-twice` |
| `tier-1 candidate %q claims both raw evidence and exclusion` | `Reconcile` | `TestReconcileGateRefusals/tier1-both` |
| `tier-1 candidate %q has neither raw evidence nor exclusion` | `Reconcile` | `TestReconcileGateRefusals/tier1-neither` |
| `tier-1 raw evidence %q is not a digest: %v` | `Reconcile` | `TestReconcileGateRefusals/tier1-raw-not-digest` |
| `tier-1 raw evidence %q is claimed twice` | `Reconcile` | `TestReconcileGateRefusals/tier1-raw-twice` |
| `tier-1 exclusion %q is outside the capture classes` | `Reconcile` | `TestReconcileGateRefusals/tier1-bad-exclusion` |
| `tier-1 candidate %q has no link to raw evidence or exclusion` | `Reconcile` | `TestReconcileGateRefusals/tier1-missing` |
| `tier-2 raw item %q is not a digest: %v` | `Reconcile` | `TestReconcileGateRefusals/tier2-raw-not-digest` |
| `tier-2 link names unreconciled raw item %q` | `Reconcile` | `TestReconcileGateRefusals/tier2-unknown-raw` |
| `tier-2 raw item %q reconciles twice` | `Reconcile` | `TestReconcileGateRefusals/tier2-twice` |
| `tier-2 raw item %q claims both a canonical item and a normalization disposition` | `Reconcile` | `TestReconcileGateRefusals/tier2-both` |
| `tier-2 raw item %q has neither a canonical item nor a normalization disposition` | `Reconcile` | `TestReconcileGateRefusals/tier2-neither` |
| `tier-2 canonical item %q is not a digest: %v` | `Reconcile` | `TestReconcileGateRefusals/tier2-canon-not-digest` |
| `tier-2 canonical item %q is claimed twice` | `Reconcile` | `TestReconcileGateRefusals/tier2-canon-twice` |
| `tier-2 normalization %q is outside the dispositions` | `Reconcile` | `TestReconcileGateRefusals/tier2-bad-normalization` |
| `tier-2 raw item %q normalizes to synthesized, which is sourceless` | `Reconcile` | `TestReconcileGateRefusals/tier2-synth-normalization` |
| `tier-2 raw item %q has no link to a canonical item or normalization disposition` | `Reconcile` | `TestReconcileGateRefusals/tier2-missing` |
| `tier-3 synthesized row %q claims source evidence %q` | `Reconcile` | `TestReconcileGateRefusals/tier3-synth-claims` |
| `tier-3 row %q covers no included candidate` | `Reconcile` | `TestReconcileGateRefusals/tier3-unknown-key`, `.../tier3-excluded-key` |
| `tier-3 row %q needs canonical %q from tier 2` | `Reconcile` | `TestReconcileGateRefusals/tier3-needs-canonical` |
| `tier-3 row %q names canonical %q, tier 2 derives %q` | `Reconcile` | `TestReconcileGateRefusals/tier3-wrong-canonical`, `.../tier3-duplicate-canonical-refused-by-agreement` |
| `tier-3 row %q names canonical %q under normalization` | `Reconcile` | `TestReconcileGateRefusals/tier3-canon-under-normalization` |
| `tier-3 row %q claims unreconciled source evidence %q` | `Reconcile` | `TestReconcileGateRefusals/tier3-unreconciled-evidence`, `TestReconcileSourceEvidenceChainBinding/foreign` |
| `tier-3 row %q claims source evidence %q of candidate %q` | `Reconcile` | `TestReconcileGateRefusals/tier3-swapped-evidence`, `TestReconcileSourceEvidenceChainBinding/swapped` |
| `tier-3 row %q has no source evidence` | `Reconcile` | `TestReconcileGateRefusals/tier3-missing-evidence`, `TestReconcileSourceEvidenceChainBinding/missing` |
| `tier-3 source evidence %q is claimed by rows %q and %q` | `Reconcile` | `TestReconcileGateRefusals/tier3-double-claimed-evidence`, `TestReconcileSourceEvidenceChainBinding/double-claimed` |
| `tier-3 loss row %q claims staged evidence %q` | `Reconcile` | `TestReconcileGateRefusals/tier3-loss-claims-staged` |
| `tier-3 loss row %q claims live evidence %q` | `Reconcile` | `TestReconcileGateRefusals/tier3-loss-claims-live` |
| `tier-3 row %q has no staged target evidence` | `Reconcile` | `TestReconcileGateRefusals/tier3-no-staged` |
| `tier-3 row %q has no live target evidence` | `Reconcile` | `TestReconcileGateRefusals/tier3-no-live` |
| `tier-3 candidate %q has no disposition row` | `Reconcile` | `TestReconcileGateRefusals/tier3-missing-row` |
| `tier-3 row %q claims staged evidence %q outside the staged read` | `Reconcile` | `TestReconcileGateRefusals/tier3-staged-outside` |
| `tier-3 row %q claims live evidence %q outside the live read` | `Reconcile` | `TestReconcileGateRefusals/tier3-live-outside` |
| `staged read names projection plan %q, want %q` | `Reconcile` | `TestReconcileGateRefusals/staged-plan-mismatch` |
| `live read names projection plan %q, want %q` | `Reconcile` | `TestReconcileGateRefusals/live-plan-mismatch` |
| `staged read names projected manifest %q, want %q` | `Reconcile` | `TestReconcileGateRefusals/staged-projected-mismatch` |
| `live read names projected manifest %q, want %q` | `Reconcile` | `TestReconcileGateRefusals/live-projected-mismatch` |
| `staged and live reads observe different native sessions` | `Reconcile` | `TestReconcileGateRefusals/reads-differ-sessions` |
| `projection plan expects native session %q, reads observe %q` | `Reconcile` | `TestReconcileGateRefusals/plan-expects-other` |
| `projected manifest expects native session %q, reads observe %q` | `Reconcile` | `TestReconcileGateRefusals/projected-expects-other` |
| `fidelity report does not bind the staged read` | `Reconcile` | `TestReconcileGateRefusals/fidelity-staged-unbound` |
| `fidelity report does not bind the live read` | `Reconcile` | `TestReconcileGateRefusals/fidelity-live-unbound` |
| `fidelity target tuple differs from the staged observed tuple` | `Reconcile` | `TestReconcileGateRefusals/fidelity-tuple-staged` |
| `fidelity target tuple differs from the live observed tuple` | `Reconcile` | `TestReconcileGateRefusals/fidelity-tuple-live` |
| `validation target tuple differs from the staged observed tuple` | `Reconcile` | `TestReconcileGateRefusals/validation-tuple-staged` |
| `validation report expects native session %q, reads observe %q` | `Reconcile` | `TestReconcileGateRefusals/validation-expects-other` |
| `validation report invalid: %v; %v` | `Reconcile` | `TestReconcileGateRefusals/validation-malformed` |
| `owner:fidelity-duplicate-row-keys` | `Reconcile` (delegated) | Duplicate row keys refuse at the fidelity build with the owner's literal, wrapped as `fidelity report invalid` (`TestReconcileGateRefusals/tier3-duplicate-keys-refused-by-owner`) |

## Stated bounds

What this package does not prove, named explicitly:

| Bound | Why |
|---|---|
| Clause-level traceability edges for the five Story cases do not exist: the traceability clause extractor (owner: `internal/traceability`, `sectionClauseInventory`) recognizes only RFC 2119 keyword lines, and the §13.14.2 reconciliation sentence is declarative, so the section yields zero extracted clauses | Orchestrator decision on CR rev2 finding story-cases-have-no-clause-edges: section-level binding is proven (`TestStoryFinalFidelityCasesAppearInDecodedBinding`), the stricter edge rule is enforced wherever representable (`TestStoryFinalFidelityClauseEdgesWhereRepresentable`), and no edge is fabricated |
| Operation/bundle constancy across reads, plan, projected, and reports is not cross-checked | Stated in Section 13.14.3, outside this story's 13.14.2 scope |
| Plan workspace vs read workspace bindings are not compared | No pinned sentence requires the pairing; the validation `workspace_binding_valid` check stays the owner's valid rule |
| Parsed event counts, head IDs, and structural digests are not derived from canonical events | The read-back leaf bounds them as non-derived; reconciliation pairs identities, not parsed content |
| Capture-manifest-internal reconciliation is not re-proven | Owned by `clonebundle` (`VerifyCaptureReconciliation`); this package censuses caller-projected key sets |
| Tier-1 exclusion reasons beyond the capture class are not modeled | The pinned sentence names exclusion without a reason vocabulary |

## Mutants

`testdata/mutant_harness.py` ships one narrowing row per gate
(62 narrowings, including one per O9 relation class) plus four
token-preserving attacks and the harmless `C-doc-comment`
SURVIVED control. Every narrowing weakens one gate to admit
exactly one member of the class it must reject, and every killer
drives a production entry alone. The four token-preserving rows
(`T-mode-staged`, `T-loss-branch`, `T-tier2-synth`,
`T-tier3-synth`) keep the searched-for token and change the
decision; their killers execute the behavioral gate tests, and
the full package suite re-runs on all four to prove the
always-on suite kills them too. `N-t3-doublechain-prefix`
censuses only the first three rows: it is invisible on every
N<=3 input (the N<=3 subset passes on it) and is killed by the
N>=4 enumeration alone. One raw log per plant per run rides the
evidence tar (67 rows + 4 suite runs + the subset-survival log).

## Importer outcome grid

This leaf adds `internal/clonereconcile` and edits no landed
file: every owner entry behaves identically before and after.
Grid keyed (package, entry, input class) → outcome:

| Package | Entry | Input class | Base | Candidate |
|---|---|---|---|---|
| `clonebundle` | `ValidCaptureClass` | 9 classes + neighbors | admit/refuse per class | unchanged (no edit; suite green) |
| `clonefidelity` | `BuildFidelityReport` | valid/invalid report candidates | seal/refuse | unchanged (no edit; suite green) |
| `clonefidelity` | `DecodeFidelityReport` | valid/invalid bytes | decode/refuse | unchanged (no edit; suite green) |
| `clonefidelity` | `ValidDisposition` | 7 dispositions + neighbors | admit/refuse | unchanged (no edit; suite green) |
| `cloneplan` | `DecodeProjectionPlan` | valid/invalid bytes | decode/refuse | unchanged (no edit; suite green) |
| `cloneplan` | `DecodeProjectedObjectManifest` | valid/invalid bytes | decode/refuse | unchanged (no edit; suite green) |
| `clonereadback` | `BuildValidationReport` | valid/invalid candidates + sealed siblings | seal/refuse | unchanged (no edit; suite green) |
| `clonereadback` | `DecodeReadBackEvidenceManifest` | valid/invalid bytes + authority | seal/refuse | unchanged (no edit; suite green) |
| `scalar` | `ParseDigest` | digests + malformed | parse/refuse | unchanged (no edit; suite green) |
| `sessadapter` | `DecodeTuple` | tuples + malformed | decode/refuse | unchanged (no edit; suite green) |
| `environ` | `StringLength` | strings | character count | unchanged (no edit; suite green) |

Moved classes: none. The candidate introduces no new import edge
into any landed package (all edges point from the new package to
the owners) and moves no declaration.

## Durability

This package holds no local state and performs no durable write:
reconciliation is pure bytes-in/bytes-out over memory, so crash
semantics are vacuous. The durability proof is determinism plus
purity: `TestReconcileDeterminism` (ten reconciliations
byte-identical) and `TestReconcilePurity` (entries never mutate
their inputs).
