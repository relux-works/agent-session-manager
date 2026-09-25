# TASK-260924-3n78rv results (rev5): projection planning and authority rules

Candidate: UNCOMMITTED tree in the STORY-260830-21bxa3 worktree
(`internal/cloneplanning/*`, 21 paths, plus one additive export
in `internal/clonefidelity/record.go`, +15/-0; no registry edit).
Change Request is `task_delta`. This revision answers CR rev4
verdict `effects-reason-set-coupling-missing` with a named
regression test driving the reviewer's exact cells, a rule x
entry table over every owner record rule, a disposition-row
census with reachability and no-fork guards, and eight mutant
rows. Trunk is still `6d3bff9` (`origin/main` verified equal),
so per the rework brief no refresh was run; the checkpoint
still descends from trunk.

Rework delta (rev4 -> rev5): new owner seam
`clonefidelity.ValidateDispositionRow` (pure delegation to the
existing row builder); `effects.go` rewritten to validate every
mapping through the seam (local vocabulary/sorted checks
removed; UTF-8 pre-gate kept); `plan.go` PlanItem self-checks
every emitted mapping; new `ownercensus_test.go` (census,
reachability, 2 guards, 279-cell evidence-ignore pin);
`session_test.go` gains the regression test, the 16-row rule
table with probe hygiene, the 556-mapping emitted-valid sweep,
and the seam-shape pin; harness `+5` rows with 4 effects rows
rewritten as delegation narrowings; one TRACEABILITY rule
(`R-OWNER`) plus axis/bound bullets. Outside-module change: the
single owner export, listed with its reason below.

## AC coverage, measured: 18 of 18 rows driven

Row 13/14 refusals now carry the owner's literal codes (rev4
measured 18/18 with local refusal texts; the behavior class the
reviewer found admitted is now refused and tabled). Each row
names its production call site; every refusal is literal-pinned
and every gate carries a killed narrowing (matrix names them).

| # | Row | Call site | Test |
|---|---|---|---|
| 1 | Strategy selection, 16 cells | SS | TestSelectStrategyOracle |
| 2 | Profile branch routing, 5 + invalids | SP | TestSelectProfileOracle |
| 3 | Item dispositions + reasons, 975 cells | PI | TestPlanItemWholeDomainOracle + 8 rule tests |
| 4 | Archive refusal with branch message | PI, SP | TestPlanItemArchiveRule, TestSelectProfileOracle |
| 5 | Continuation never exact | PI | TestPlanItemContinuationRule |
| 6 | Opaque reasons by protection/cause | PI | TestPlanItemOpaqueRule |
| 7 | Aborted tools are summarized history | PI, CI | TestPlanItemAbortRule, TestClassifyItemValidGrid |
| 8 | Importer tool loss + graph flattening | PI | TestPlanItemImporterRule |
| 9 | Strict/compact/messages overlays | PI | TestPlanItemStrictRule, TestPlanItemCompactRule, TestPlanItemMessagesRule |
| 10 | Visible authority always user_context | PV | TestProjectVisibleWholeDomain (6986 cells) |
| 11 | Escaping refuses invalid UTF-8 at BOTH exported entries, returns no U+FFFD value, and round-trips via encoding/json | EV, PV | TestEscapeVisibleTextRefusesInvalidUTF8 (5 classes + reviewer-exact FF), TestProjectVisibleTextRefusesInvalidUTF8, TestProjectVisibleWholeDomain, TestProjectVisibleEscapingOracle |
| 12 | Adversarial text never gains authority (every kind x every block type x every text class x every entry); ignored invalid bytes cannot change the item; ignored evidence reasons cannot change the item | CI, PI, TE, PV | TestClassifyItemIgnoresPayloadTextAllKinds (8432 sealed + 1054 hand-built + 1581 generated/long, downstream chain to zero, reviewer probes through PV); TestClassifyItemIgnoresInvalidPayloadBytes (620 cells); TestClassifyItemIgnoresEvidenceReasons (279 cells, rev5); TestClassifyItemReadsClosedFactsOnly (AST); TestClassifyItemVocabMatchesSpec (spec pin); TestProjectVisibleWholeDomain, TestProjectVisiblePositionIndependence; SS/SP/PI/PS/TE take no text by type |
| 13 | Tools inert; pending blocked; malformed disposition/reason pairs refuse with owner literals | TE, PI | TestPlanTargetEffectsZeroSweep, TestPlanTargetEffectsChained, TestPlanTargetEffectsRefusesMalformedReasonSets (rev5 regression), TestPlanTargetEffectsOwnerRecordRules (16 rows, rev5) |
| 14 | Source usage never target accounting | TE | TestPlanTargetEffectsZeroSweep (556 mappings + per-disposition) |
| 15 | Token classes (control-token text + usage counts) | PV, TE | rows 11 + 14 probes |
| 16 | Vendor importer strategy arm | PI | TestPlanItemImporterRule |
| 17 | Session counts + empty/1M/1M+1 edges | PS | TestPlanSessionCounts, TestPlanSessionRefusals, TestPlanSessionMillionEdge |
| 18 | Importer grid base vs candidate, moved classes named | — | importer-grid.log, byte-identity (231 identical + 1 additive export, moved none) |

Stated bounds (not waived): operation synthesis; source/target
condition reasons outside the item axis; capability-gated
degradation past tool/subagent importer classes; per-kind
text-field extraction (visible entry takes extracted typed texts);
full Migration Checkpoint sealing; fact-name renames past the
grep pin; invalid UTF-8 in ignored payload members (accepted and
ignored, never decided or emitted); delegating outer UTF-8 checks
as defense in depth (inner gate measured); synthesized-canonical
and non-synthesized-evidence rules see scaffolding only (Mapping
carries neither member; owner suite pins both arms); the PlanItem
self-check arm is unreachable by measurement (bypass plant killed
by the reachability census). See TRACEABILITY.md.

## Mutant evidence (rev5 rows; full battery below)

| Mutant | What it narrows the gate to | Named test that fails (run alone) |
|---|---|---|
| N-effects-exact-reasons | Delegated gate skips the owner call for exactly the exact-with-operator_policy cell | TestPlanTargetEffectsRefusesMalformedReasonSets |
| N-effects-nonexact-bare | Delegated gate skips the owner call for exactly the bare semantic cell | TestPlanTargetEffectsRefusesMalformedReasonSets |
| N-effects-disposition (rewritten) | Delegated gate skips the owner call for exactly bogus_disposition | TestPlanTargetEffectsRefusals |
| N-effects-reason (rewritten) | Delegated gate skips the owner call for exactly single bogus_reason sets | TestPlanTargetEffectsRefusals |
| N-effects-order-skip (rewritten) | Delegated gate skips the owner call for exactly the unsorted unsafe/operator pair | TestPlanTargetEffectsRefusals |
| N-effects-reason-bounds | Delegated gate skips the owner call for exactly the empty reason | TestPlanTargetEffectsOwnerRecordRules |
| N-effects-reason-count | Delegated gate skips the owner call for exactly 129-reason sets | TestPlanTargetEffectsOwnerRecordRules |
| N-effects-nonzero (rewritten) | Zero fold funds one target input token per exact mapping | TestPlanTargetEffectsZeroSweep |
| N-planitem-owner-bypass | Presence plant: PlanItem self-check removed (behaviorally silent; arm proven unreachable by the 556-mapping emitted-valid sweep) | TestDelegationReachesOwner |

No survivors among narrowings. The neutral control
`C-doc-comment` (comment-only edit) SURVIVED as designed in all
three battery runs, proving the harness observes outcomes. Full
battery: 57 KILLED + 1 SURVIVED, 0 mismatches, 0 errors, three
runs on the final tree (`harness1/`, `harness2/`, `harness3/`
summaries; per-plant raw logs ship for the final run).

## Validation (real exit codes)

| Command | Exit | Evidence |
|---|---|---|
| `go test -count=1 -v ./internal/cloneplanning/` | 0, 60 PASS | `package.log` |
| `go test -count=1 -cover` on cloneplanning + clonefidelity | 0, 97.8% / 97.3% statements | `cover.log` |
| `go test -count=3` on the 13 new/changed tests | 0 | `determinism.log` |
| `go test -race -count=1` on cloneplanning + clonefidelity | 0 | `race-package.log` |
| `go test -p 4 -count=1` on the 9 owner packages | 0, 9 ok | `owners.log` |
| `go test -p 2 -count=1` full suite, 4 shards | 0, 48 ok, 0 FAIL | `full-shard1.log` .. `full-shard4.log` |
| 13 CI-derived `go test -fuzz ... -fuzztime=100x` smokes | 0 all, elapsed lines present | `fuzz.log`, `fuzz-targets.clean` |
| cigate contract gates (6) | 0, all PASS | `cigate-contracts.log` |
| cigate capability-claim gates (6) | 0, all PASS | `cigate-claims.log` |
| cigate fuzz-target derivation gates (2) | 0, all PASS | `cigate-targets.log` |
| `go vet ./...` | 0 | `vet.log` |
| `GOOS=windows GOARCH=amd64 go vet ./...` | 0 | `winvet.log` |
| gofmt scan (838 files) | 0, no files listed | `gofmt.log` |
| `go build ./...` | 0 | `build.log` |
| `GOOS=linux GOARCH=amd64 go build ./...` | 0 | `build-linux.log` |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 | `build-windows.log` |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | `tracecheck.log` |
| CI catalog check (`go generate` + `git diff --exit-code`) | 0, no diff | `cataloggen.log` |
| JSON validity sweep (141 tracked files) | 0, 0 invalid | `jsoncheck.log` |
| `task-board validate` | 0 (254 pre-existing board warnings, none on this task) | `board-validate.log` |
| `git diff --check` | 0 | `diffcheck.log` |
| Scratch-index `git diff --cached --check` (22 paths) | 0 | `hygiene.log`, `scope.log` |
| Mutant battery, 58 rows, run 1 | 57 KILLED + control SURVIVED, 0 mismatches | `harness1-summary.log` |
| Mutant battery, 58 rows, run 2 | 57 KILLED + control SURVIVED, 0 mismatches | `harness2-summary.log` |
| Mutant battery, 58 rows, run 3 (final tree) | 57 KILLED + control SURVIVED, 0 mismatches | `harness3-summary.log`, `harness3/` |
| Delegation census plant: new row entry (must redden) | 1, list linkage named | `deleg-control-newentry.log` |
| Delegation census revert (must pass, byte-identical) | 0 | `deleg-control-newentry-revert.log` |
| Reachability plant: self-check removed (must redden) | 1, seam named | `deleg-control-bypass.log` |
| Reachability revert (must pass, byte-identical) | 0 | `deleg-control-bypass-revert.log` |
| No-fork plant: vocabulary call re-added (must redden) | 1, call named | `deleg-control-fork.log` |
| No-fork revert (must pass, byte-identical) | 0 | `deleg-control-fork-revert.log` |
| No-switch plant: closure switch added (must redden) | 1, function named | `deleg-control-switch.log` |
| No-switch revert (must pass, byte-identical) | 0 | `deleg-control-switch-revert.log` |
| Reviewer cells on pre-fix shape (authentic failure) | 1, admission named | `prefix-demo-fail.log` |
| Reviewer cells on fixed shape | 0 | `prefix-demo-pass.log` |
| Owner byte-identity base vs candidate | 231 identical, 1 additive (+15/-0) | `importer-grid.log` |
| `task-board.config.json` base vs candidate | byte-identical blob | (hashes compared) |
| `git status` candidate scope | `M record.go` + `?? internal/cloneplanning/` | (worktree state) |
| `go test ./... -cover -count=1` (full-cover) | NOT RUN | exceeds the bounded-call budget; changed packages measured at 97.8%/97.3%; rerun by CR validation |
| `go test ./... -race -count=1 -timeout 25m` (full-race) | NOT RUN | exceeds the bounded-call budget; changed packages race-green; rerun by CR validation |

All shard, package, and battery logs ran on the final tree; no
source file changed after `full-shard1.log` except board
resources (outside the worktree).

## Out-of-contract rows (declared, not silent)

- Closed shapes (plan, manifest, fidelity report, capture,
  canonical): sibling leaves cloneplan/clonefidelity/clonebundle;
  this leaf decides populating values and invokes (never
  re-implements) the owner's row validator.
- G1 normalization and folds: clonesnap/cloneproject; consumed
  here by reference and source pin.
- Target-branch admission: clonesnap.AdmitForTarget (caller
  scope); this leaf routes the branch only.
- Transaction, lineage, read-back, validation (13.14.3-13.14.5):
  sibling/future scope.

## Brief gap (handoff precondition 1)

The brief carries no surface table, so there is no surface row to
map. Coverage map substitute: the clause map in
`internal/cloneplanning/TRACEABILITY.md` maps every derivation
rule to its production entries and named tests, and the
conformance matrix maps every gate to its entries, killer tests,
and killed narrowings (57 narrowings + 1 control).

## Reviewer tests kept

No previously committed reviewer test exists on the board (rev1,
rev3, and rev4 verdicts carried probe files inside their evidence
tars, not committed tests). The rev1 reviewer's exact
`subagent_started` plant and all six probe texts remain embedded
in the committed sweep and `N-classify-text-subagent`; the rev3
reviewer's exact FF probe remains embedded in
`TestEscapeVisibleTextRefusesInvalidUTF8`; the rev4 reviewer's
exact cells (semantic without reasons, exact with
operator_policy) are now embedded in the committed
`TestPlanTargetEffectsRefusesMalformedReasonSets` with
`N-effects-exact-reasons` and `N-effects-nonexact-bare`, so the
next cycle inherits all three.

## Rework diff bound

The rev4 -> rev5 diff touches `internal/cloneplanning/` (1 new
test file, 2 edited production files, 4 edited test files, 1
edited harness, TRACEABILITY + doc updates) plus exactly one
file outside it, listed here with its reason:

- `internal/clonefidelity/record.go` (+15/-0): the rework
  invariant requires every entry that accepts a disposition or
  reason set to call the owner's validator, but no exported
  row-level seam existed — only the whole-report Build, whose
  count reconciliation would have forced this leaf to duplicate
  the owner's count derivation, and the vocabulary predicates,
  which cannot express the coupling rule. The new export is
  pure delegation to the existing row builder with zero
  behavior change (owner suite green unchanged); the
  alternative (a local coupling check) is exactly the fork the
  finding forbids. No registry edit, no other landed-file edit.

## Logbook-relevant decisions

- The owner's row rules had no exported seam (only whole-report
  `BuildFidelityReport` with count reconciliation and the
  vocabulary predicates). Reusing Build would have forced
  count-derivation duplication; re-implementing the coupling
  would have forked the owner. The additive `ValidateDispositionRow`
  export (+15/-0) is the minimal seam; verified by the 231/232
  byte-identity grid and the green owner suite.
- The disposition-row census pairs a disposition member with a
  reason-set member because the seam validates the coupling,
  which needs both. ClassifyItem's event evidence (reasons
  without a disposition, validated by the landed decoder and
  ignored by the classifier) is excluded by that rule and the
  exclusion is measured (SourceEvidence shape pins + 279
  adversarial ignore cells).
- PlanSession's counting switch stays: it derives counts from
  owner-validated emitted values (derivation, not admission)
  and the 975-cell oracle pins its behavior. The no-switch
  guard therefore scopes to the validation closures, with the
  scoping itself tested (the guard test fails if the
  PlanSession switch ever disappears).
- `PlanTargetEffects` keeps its UTF-8 pre-gate ahead of the
  owner call: the rev4 held row requires the package literal
  for invalid bytes, while the owner would refuse invalid
  dispositions with its vocabulary text.
