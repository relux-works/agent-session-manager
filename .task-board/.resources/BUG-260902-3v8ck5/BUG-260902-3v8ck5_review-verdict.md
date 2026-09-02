# BUG-260902-3v8ck5 — Review verdict: ACCEPTED

Change Request `CR-BUG-260902-3v8ck5-1` revision 1, base `f20aeec`, candidate tree
`d92be21c82010fd9962a1f19fa3c24678ceaf48c`, repository delta `present` (8 paths).
Reviewed in the story worktree at `.temp/STORY-260902-2rngvr/worktree`; the working
tree was verified byte-identical to the candidate tree before and after every
mutant (`git diff --name-status d92be21…` empty; blob hashes of the three
production files equal to `git ls-tree` of the candidate).

## What the change does

`internal/canonicaljson/declared_byte_bounds.go` introduces one measurement,
`CanonicalByteLength` (encode with `SetEscapeHTML(false)`, then `Canonicalize`),
and one refusal gate, `canonicalByteBound(name, value, maximum)`.
`closed_shapes.go:284` (Launch Plan argv) and `closed_shapes.go:1794` (extensions
object) call the gate; `internal/config/validation.go:764` calls the measurement.
Both previously-divergent sites and the already-correct third site now agree.

## Attacks run by the reviewer (independent of the producer's sweep)

All mutants were applied to the candidate tree, run, then reverted by restoring
a pre-mutation copy — never by `git checkout`.

| Mutant | Change | Result |
| --- | --- | --- |
| RM1 delete-only | remove the argv gate call entirely from `validateSessionLaunchPlan` | RED — 4 tests: `TestLaunchPlanArgvByteBoundIsMeasuredOnCanonicalBytes`, `TestDeclaredBoundaryConstraintsReachBothIdentityEntries`, `TestEveryDeclaredBoundCallSiteIsProvenInBothDirections`, `TestEveryProductionRefusalGuardIsExecuted` |
| RM2 delete-only | `validateExtensionsObject` returns `nil` instead of the gate | RED — 5 tests incl. `TestIdentityExtensionsEnforceCountDepthAndCanonicalSize` |
| RM3 neuter | config bound raised to `maxConfigExtensionBytes*99` (import kept live, so the mutant compiles) | RED — `TestConfigurationExtensionByteBoundIsMeasuredOnCanonicalBytes`, `TestCurrentEntriesProveRelationalCollectionAndNestedBoundsInBothDirections` |
| RM4 narrow | argv bound `65_536` → `65_535` | RED — accept-at-limit fails |
| RM5 neutrality | drop `encoder.SetEscapeHTML(false)` | GREEN, as designed — `Canonicalize` re-parses, so the flag is defence in depth, not the fix. Matches the code comment rather than contradicting it. |

The producer's own 8-mutant sweep (`BUG-260902-3v8ck5_mutation-sweep.log`) was
re-read and its narrow/widen entries spot-reproduced. Delete-only mutants are
present but are **not** the load-bearing evidence here — RM4 and the producer's
M2–M7 narrow/widen pairs pin where the bound sits, not merely that one exists.

## Completeness of the property claim — proved, not assumed

The AC asks that a document at the declared limit be accepted *regardless of how
many characters JSON escaping would expand*. The fixture table names five
fillers (`<`, `>`, `&`, U+2028, U+2029) plus an encoding-neutral control. A
table is only a property proof if that set is the whole divergence class, so the
reviewer enumerated it rather than trusting the comment: a temporary probe
compared `len(json.Marshal)` against `len(Canonicalize)` for every valid rune in
`U+0000..U+10FFFF` (surrogates skipped).

- divergent rune count = **5**, and the five are exactly the five in the table.
- No unexpected divergent rune exists. The table is exhaustive over Unicode, which
  is stronger than sampling.

The probe was run as `internal/canonicaljson/zz_reviewer_probe_test.go` and
deleted afterwards; it is preserved read-only at
`.temp/BUG-260902-3v8ck5/probe/divergence_probe_test.go`.

Separately confirmed out-of-tree that `SetEscapeHTML(false)` alone does **not**
close the defect: with it set, `["<"]` is 5 bytes but `[U+2028]` is still 10
against 7 canonical. The AC offered "reuse `Canonicalize` **or** an encoder with
`SetEscapeHTML(false)`"; the second alternative would have been insufficient on
its own for U+2028/U+2029, and the implementation correctly took the first.

## Bypass-path and drift checks

- `grep` for `len(json.Marshal(...))` as a size measurement across production
  code: **zero** remaining occurrences. The defect class is gone, not moved.
- Every extensions path funnels through one of the two shared entries:
  `validateExtensionsObject` (10 call sites in `closed_shapes.go`, one in
  `core_records.go`) and `validateExtensions` (3 call sites in `validation.go`,
  covering directory installations, enrichment profiles, and peer disclosure).
  There is no extensions surface that reaches a bound without the shared gate,
  so covering one config section is covering one function, not one of three
  divergent clauses.
- Production entries are driven, not helpers: the canonicaljson tests go through
  `CalculateObjectIdentity` **and** `VerifyObjectIdentity`; the config tests go
  through `EncodeCurrent` (writer) **and** `loadConfigDocument` (reader). The
  reader case had to be reached by editing an already-encoded document, because
  the writer refuses the over-limit configuration first — the test says so and
  guards that it edited exactly one line.
- The SPEC literal is a local `const` in each test file, independent of
  `maxConfigExtensionBytes` and of the inline `65_536` at the call sites; RM4 and
  M2–M7 confirm that moving the production literal reddens the proof.
- Refusal-guard inventory updated correctly: the three now-unreachable call-site
  refusals were replaced by the three inside the shared helper, and
  `TestEveryProductionRefusalGuardIsExecuted` is green and still reddens on RM1.

## Gates re-run by the reviewer at the candidate tree

| Gate | Result |
| --- | --- |
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `gofmt -l internal/` | empty |
| `go test ./... -cover -count=1` | all 10 packages ok |
| `go run ./internal/traceability/cmd/tracecheck` | 0 — `contracts=60 normative_sections=36 acceptance_cases=43 fixtures=30 compatibility_contracts=55` |
| `cataloggen -check` | 0 |

Coverage against the producer's detached-worktree baseline at the same commit:
identical in every package, `internal/canonicaljson` 97.2%, `internal/config`
94.7%. No regression. Log: `.temp/BUG-260902-3v8ck5/review-cover.log`.

## Observation, not a finding (reported as unknown)

`internal/config/validation.go:465` carries a third 65,536-byte bound —
`mesh.peers[i]` SSH argv bytes — measured as a raw sum of
`len(endpoint) + Σ len(argument)`. It never passed through `json.Marshal`, so it
is not an instance of the defect this bug fixes, and it is outside the stated
scope (§6 and §10.1). Whether the specification declares that particular bound on
a canonical encoding rather than on raw transport bytes could **not** be
established here: the repository pins contracts in
`internal/specpin/v0.5.0.lock.json`, which carries no `65536` literal and no spec
prose. Reporting it as unknown rather than inferring it from the neighbouring
bounds. Worth a separate board item if the spec text says "encoded".

## Definition of Done

- [x] Every declared byte bound in scope measured on canonical bytes by reusing `Canonicalize`
- [x] argv site and config extension site share one helper — verified by call graph, not by claim
- [x] Property test at the declared limit, using the SPEC literal, proved exhaustive over the divergence class
- [x] A mutant restoring escaped measurement reddens (producer M1/M8; delete and narrow mutants reproduced by the reviewer)
- [x] Accept-at-limit and refuse-one-past proven at `CalculateObjectIdentity`, `VerifyObjectIdentity`, `EncodeCurrent`, `loadConfigDocument`
- [x] Tests, vet, build, tracecheck, catalog check green; no coverage regression
- [x] Logbook entry present and accurate — every quantitative claim in it that the reviewer checked reproduced

## Verdict

**ACCEPTED.** No changes requested. Not committed by this run: a
reviewer-archetype run must not supply `commit_ack`. The orchestrator owns the
commit of this scope and the final `done` transition with
`commit_ack=scope_committed`.
