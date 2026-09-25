# TASK-260830-1jmmqn results: projection-plan schemas (rework rev4)

Candidate: UNCOMMITTED tree in the STORY-260830-21bxa3 worktree.
Scope: `internal/cloneplan` (new files only; no landed file edited,
no registry edit). Change Request is `task_delta`. Base trunk
`0ca3e4c` is an ancestor of the checkpoint (`git merge-base
--is-ancestor 0ca3e4c HEAD` exits 0; HEAD `547ea28`): no refresh
needed.

This revision answers the rev3 finding
(`plan-constant-second-value-uncounted`). Every rev1 finding stays
closed; every rev1 production line is intact (plans.go still
`sha256:c2f7eb93506291635218beb49a589d8920bf32459d2906d73c9c3c74e311f02f`,
the reviewer's pinned blob). The diff is tests + harness +
TRACEABILITY, plus one new test file
(`constants_closed_test.go`).

## What changed per the finding

The rev3 finding: `TestPlanConstants` probed three invalid
`materialization_intent` strings (`prepare`, `clone2`, `Clone`),
so a gate widened to admit `copy` survived the submitted suite on
both sides. The invalid domain of a closed constant is every other
string, which is unbounded; this revision closes it BY
CONSTRUCTION, exactly per the rework brief:

1. Named regression (`TestPlanConstantCopyRegression`): `copy`
   refuses at every closed string gate (4 nested constants +
   modes + 4 schema/version literals) on both entries, 14
   subtests, literal refusal tokens. Each subtest carries sibling
   probes (the literal admits, the first derived neighbor
   refuses), so the kill lands on the `copy-refuses` probe only.
2. Generated invalid values (`TestPlanConstantGeneratedSweep`,
   `TestPlanConstantDerivedValues`, `TestPlanConstantBooleans`,
   `TestPlanConstantNonBooleanJSON`): per string literal at both
   entries, every edit-distance-1 neighbor over
   lowercase+digit+`_`, every case variant class (reusing the
   census `caseVariantsOf`), a fixed-seed 200-string sample, the
   `copy` probe, empty, whitespace paddings, and the 5-value
   non-string JSON grid at decode; per boolean, the other boolean
   both entries plus all 5 non-boolean JSON types at decode;
   modes adds per-element neighbor/case sweeps, the order swap,
   lengths 0..4, non-string elements, and non-array JSON. The
   whole sweep runs in ~2s.
3. Structural guard (`TestClosedConstantGateShape`): the gate
   LIST derives mechanically (reflection over the four nested
   plan input types, maps skipped by kind; build/decode functions
   found by type signature; the modes checker found by call
   shape), cross-checked both directions against the spec oracle
   (SPEC.v0.7.0.md:10703-10718, 10654, 168-169). Each gate must
   be one equality against the oracle literal; the
   comparison-literal set per file must equal the oracle set
   exactly; no switch, no prefix/fold/contains/normalize call.
   Control plants: the reviewer's build-side copy widening
   reddens it (exit 1), a two-case switch reddens it (exit 1),
   and a bonus decode-side copy widening reddens it (exit 1);
   pristine passes (exit 0). Production restored byte-identical
   after each plant (sha re-verified).
4. Mutants: 13 copy-admitting narrowings (`N-copy-*`, one per
   string gate per side plus the shared modes gate and the 4
   decode-only schema/version literals) killed by the named
   copy-regression subtests ALONE, and 7 null-admitting decode
   narrowings (`N-null-*`, one per closed boolean gate) killed by
   the named non-boolean subtests ALONE. Battery is now 145 rows:
   143 narrowing + 1 marked arm-delete + 1 neutral control
   SURVIVED.

## AC coverage, measured: 18 of 18 leaf-schema rows driven

Denominator: the 18 leaf-schema rows (planning semantics excluded
per the orchestrator split). Each row names its production call
site; every refusal below is literal-pinned and every gate carries
a killed narrowing (matrix names them). Rows 8-11 are now carried
by the construction above; all other rows unchanged from rev2.

| # | Row | Call site | Test |
|---|---|---|---|
| 1 | Plan 36-member set + schema/version literals | D | Derived census + UnknownMissing + Miscased(6 classes) + WrongTypes + Duplicated(every member) + NoLiteralList + SchemaLiterals + CopyRegression + DerivedValues + GeneratedSweep + GateShape |
| 2 | `projection_plan_id` omit-self identity | B, D | TestPlanIdentityIndependent |
| 3 | Mapping shape + disposition/reason vocabs | B, D | Census + ExpectedDisposition/MappingReason oracles |
| 4 | Operation shape + sequence order | B, D | Census + DAGRefusalLiterals |
| 5 | Dependency DAG vs oracle | B, D (shared gate) | WholeDomainCensus + ValidFive + Locality + SingleEdge + RefusalLiterals |
| 6 | Resource branch-exact nullability | B, D (shared gate) | ResourceNullabilityGrid (12/12 full) + BranchLiterals |
| 7 | Synth event shape + purposes | B, D | Census + SynthesizedPurposeOracle |
| 8-11 | Transaction/ReadBack/Resume/Rollback constants | B, D | TestPlanConstants + CopyRegression + DerivedValues + GeneratedSweep + Booleans + NonBooleanJSON + GateShape |
| 12 | ContractRequirement (id, SemVer, no ext) | B, D | TestContractSemver + TestContractNoExtensions |
| 13 | Manifest 10-member set + identity | MB, MD | Derived census + SchemaLiterals + ManifestIdentityIndependent + CopyRegression + DerivedValues + GeneratedSweep + GateShape |
| 14 | Entry branch-exact member sets | MB, MD | EntryNullabilityGrid (48/48 full) + GridNeverPanics + EntryBranchLiterals |
| 15 | Sorting/uniqueness on every ordered array | B, D, MB, MD | TestSortedUniqueSweep + TestRowOrderSweep |
| 16 | Every bound at edge and edge+1 | B, D, MB, MD | CountEdges + MillionEntryLevel + FactoredCountGates + String/Number edges |
| 17 | Policy maps | B, D | TestRequiredDispositions + TestSecurityExclusions |
| 18 | Scalar delegation | B, D, MB, MD | TestScalarGates + TestManifestScalarGates + TestNumberModel |

Stated bounds (not waived): the unbounded remainder past the
generated invalid set is closed by the structural guard (any
second value changes the gate shape or the comparison-literal
set), not by sampling; 5-op DAG coverage is structural (33M
graphs infeasible); the literal-list guard trips at census scale
(10+/40%+); JSON-level member probes run at decode entries while
build runs the per-field zero census. Cross-artifact couplings,
`total_bytes` derivation, and synthesized order stay bounded as in
rev1 (TRACEABILITY).

## Out-of-contract rows (declared, not silent)

- "Plan pair-neutral target projection, checkpoint message
  authority, inactive tools/instructions, token metadata, and
  vendor importer strategies" as PLANNING SEMANTICS: sibling leaf
  TASK-260924-3n78rv (orchestrator split); this leaf gates shapes
  only. No planning-semantics code was added.
- Cross-artifact couplings, `total_bytes` derivation, synthesized
  array order, policy key closure: stated bounds in TRACEABILITY.md
  (spec-silent or sibling-input scope), each witnessed by a named
  test.

## Brief gap (handoff precondition 1)

The brief carries no surface table, so there is no surface row to
map. Coverage map substitute: the clause map in
`internal/cloneplan/TRACEABILITY.md` maps every pinned clause to
its production entries and named tests, and the conformance matrix
(`TASK-260830-1jmmqn_conformance-matrix.md`) maps every gate to
its entries, killer tests, and killed narrowings (143 narrowings +
1 marked arm-delete + 1 control).

## Validation (real exit codes)

| Command | Exit | Evidence |
|---|---|---|
| `go test -count=1 ./internal/cloneplan/` | 0 (49s) | `package.log` |
| `go test -count=1 -cover ./internal/cloneplan/` | 0, 96.8% statements | `cover.log` |
| `go test -count=1 ./...` (full configured suite, once) | 0, 46 packages ok | `full.log` |
| `go test -count=3` (new constant tests + TestPlanConstants) | 0 (5s) | `determinism.log` |
| `go vet ./internal/cloneplan/` | 0 | `vet.log` |
| `GOOS=windows GOARCH=amd64 go vet ./...` | 0 | `winvet.log` |
| `gofmt -l internal/cloneplan/` | 0, no files | `gofmt.log` |
| Mutant battery, 145 rows, run 1 | 144 KILLED + control SURVIVED | `mut-run1-verdicts.txt`, `mut-logs-run1/` |
| Mutant battery, 145 rows, run 2 | 144 KILLED + control SURVIVED | `mut-run2-verdicts.txt`, `mut-logs-run2/` |
| Guard pristine pass | 0 | `guard-controls/guard-pristine.log` |
| Guard copy-widen control (must redden) | 1, gate named | `guard-controls/guard-copy-widen.log` |
| Guard two-case-switch control (must redden) | 1, gate named | `guard-controls/guard-two-case-switch.log` |
| Guard decode copy-widen control (must redden) | 1, gate named | `guard-controls/guard-decode-copy-widen.log` |
| Owner suites, candidate (6 pkgs) | 0 | `owners-candidate.log` |
| Owner suites, base archive (6 pkgs) | 0 | `owners-base.log` |
| Owner byte-identity base vs candidate | identical, 171 files | `byte-identity.txt` |
| Scratch-index `git diff --cached --check` (full tree) | 0 | `diff-check.log` |
| `task-board.config.json` vs base | byte-identical | `cmp` (see notes) |

Notes: the 20 new rows were also run as a group before the full
batteries (20/20 KILLED; superseded by run1/run2 logs above). A
stray `__pycache__` written by a syntax check was deleted before
hygiene; the candidate holds 25 paths. `task-board.config.json`
compares byte-identical via `cmp` against `git show
HEAD:task-board.config.json` (exit 0). Production files are
untouched since rev3 (plans.go
`c2f7eb93506291635218beb49a589d8920bf32459d2906d73c9c3c74e311f02f`).

## Rework diff bound

The diff touches only `internal/cloneplan/*` (25 paths: 24 rev3
paths plus the new `constants_closed_test.go`; still uncommitted;
real index untouched). No file outside it is listed: rev1/rev3
production semantics are unchanged, and this revision adds no
planning-semantics code, no registry edit, and no landed-file edit.
