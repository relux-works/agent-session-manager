# BUG-260902-1874eo / BUG-260902-9uwnm7 — rework round 1 results

Answers review round 1, which accepted part 1 and rejected part 2.
Delivered as an amend of the single leaf commit (one leaf = one commit past
checkpoint `422786c`); the signed head is `8a0dced`.

Changed paths: `internal/canonicaljson/utf8_subsumption_test.go`,
`internal/canonicaljson/closed_shapes.go` (comment only), `README.md`,
`LOGBOOK.md`. `internal/canonicaljson/canonical_test.go` is untouched — part 1
was accepted and was not edited (`git diff HEAD~1 -- canonical_test.go` empty
across the amend).

## What the review proved, restated

`productionCallGraph` recorded an edge only for `call.Fun.(*ast.Ident)` naming a
top-level declaration, and both derivations skipped `Recv != nil`. This package
dispatches through a function-value table: validators are registered as function
values in `mustBuildImmutableObjectShapeValidators` (`closed_shapes.go:66`) and
invoked as `validator(object)` in `validateImmutableObjectShape`. Neither the
registration nor the invocation was an edge, so coverage was 3 of 7 guarded
functions and mutants H and I survived.

## Fix 1 — strengthen the graph

`resolveCallee` now classifies every call site:

| Call shape | Modelled as |
| --- | --- |
| `f(...)`, `f` a package function | static edge to `f` |
| `x.M(...)`, `M` a method declared in this package | static edge to that method |
| `pkg.F(...)`, `pkg` an import | external, no edge |
| `f(...)`, `f` a builtin or predeclared type name | conversion/builtin, no edge |
| `f(...)`, `f` any other bare identifier | **function value** → edges to every address-taken function |
| `table[k](...)`, `g(a)(b)`, other computed callee | **function value** → edges to every address-taken function |
| `x.M(...)`, `M` not declared here | external method **or func-typed field — not modelled** |

"Address-taken" is derived: any identifier naming a package function or method
that appears anywhere in the package in a position other than a call callee or
its own declaration name. That covers `register(schema, validateSessionRecordV1, ...)`.

Methods are included in `utf8GuardedFunctions`, in the declaration set, and in
the exported entry-point set. `byteInputParameters` returns false for any method:
a receiver is an already-constructed Go value the caller supplies, so a method
can never satisfy the byte-input rule whatever its parameters look like.

### Coverage, as a ratio

`TestEveryUTF8RecheckIsCoveredByTheEntryPointPin` is new. It fails and names the
uncovered functions when a guarded function is reachable from no exported entry
point, because such a guard is asserted about vacuously.

| | Before | After |
| --- | ---: | ---: |
| Guarded functions reachable from an exported entry point | 3/7 | **7/7** |

Uncovered before, covered now: `validateSessionLaunchPlan`,
`validateSessionBoardIdentity`, `validateSortedUniqueStrings`,
`validateProviderIdentityRecord` — exactly the four the review named.

## Fix 2 — narrow the published claim

`closed_shapes.go` no longer ends "so the invariant cannot be broken silently".
`README.md` no longer says "Adding a second decoder or a map-taking entry point
reddens the pin" without qualification. Both now state what is modelled (direct
calls, methods declared in this package, dispatch through a function value) and
name what still evades it:

- a callee reached by reflection;
- a function value handed to another package and invoked there;
- **a func-typed struct field**: `x.M(...)` resolves only when `M` is a method
  declared in this package, so a dispatch table parked in a struct field and
  invoked as `table.validate(object)` produces no edge at all.

That last one is why the re-checks stay rather than being deleted: the pin
narrows how such a change can arrive unnoticed, it does not make it impossible.

## Mutants

Every mutant was confirmed present in the file before the run was believed, and
the tree was restored and verified clean (`git status --short`) after each.

| # | Mutant | Target | Result |
| --- | --- | --- | --- |
| H | exported `ValidateDecodedRecord(map[string]any)` → `validateImmutableObjectShape` (dispatch table only) | `TestNoExportedEntryPointHandsADecodedValueToAUTF8Recheck` | **RED**, names all 7 guards |
| I | same body as an exported method `DecodedRecordValidator.Validate(map[string]any)` | same | **RED**, `DecodedRecordValidator.Validate` |
| I2 | exported method with a `[]byte` parameter, decoded state on the receiver | same | **RED** — proves the receiver rule, not just the parameter rule |
| D2 | exported `ValidateExtensionsDirectly(any)` calling a guard by identifier | same | **RED** (regression check: the one shape the old graph could see) |
| G2 | exported `[]byte` entry point decoding with `json.Unmarshal` | `TestDecodeStrictIsTheOnlyJSONDecodeInTheProductionPackage` | **RED** |
| E3 | one subsumption comment deleted | `TestEveryUTF8RecheckDeclaresItsSubsumption` | **RED**, `closed_shapes.go:472` |
| F3 | narrowing: comment kept but names `requireExactMembers` | same | **RED**, `closed_shapes.go:474` |
| K | narrowing on the graph: the function-value (bare-identifier) edge removed | clean tree, then + H | clean **GREEN**; **H SURVIVES** — identifies that edge as the decisive addition |
| K2 | K plus the computed-callee edge removed | `TestEveryUTF8RecheckIsCoveredByTheEntryPointPin` | **RED**: `coverage is 3/7`, naming the same four functions the review measured |
| C2 | part 1: last hex digit of the `NUM-U64-MAX` expected value, SPEC quotation left intact | `TestCalculateObjectIdentityMatchesAXPublishedFixtures` | **RED** in `decimal_uint64_maximum` only; the other four subtests pass |

K is reported deliberately as a survivor: a mutant that leaves the clean tree
green but lets H through is what shows the new edge is load-bearing. A
delete-only mutant would have shown only that some code was executed.

## Validation, real exit codes, each run standalone

| Command | Exit | Result |
| --- | ---: | --- |
| `gofmt -l internal/` | 0 | no paths listed |
| `go build ./...` | 0 | |
| `go vet ./...` | 0 | |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | contracts=60 normative_sections=36 acceptance_cases=43 fixtures=30 compatibility_contracts=55 |
| `go generate ./internal/catalog` then `git diff --exit-code -- internal/catalog` | 0, 0 | no generated drift |
| `go test ./... -count=1` | 0 | all 10 packages ok |
| `go test ./... -cover -count=1` | 0 | `internal/canonicaljson` 97.2%, unchanged |
| `go test ./internal/canonicaljson -run='^$' -fuzz='^FuzzCanonicalizeRoundTrip$' -fuzztime=100x -parallel=1` | 0 | 100 execs, 0 new interesting |
| `git verify-commit HEAD` | 0 | good signature, `oparin@me.com` |

No existing assertion was weakened or deleted. No production logic changed;
`closed_shapes.go` is a comment-only edit.

## Found, not fixed

1. `internal/canonicaljson/core_records.go:278` in `validateProviderIdentityRecord`
   carries a ninth re-check of the same class, outside this bug's stated
   `closed_shapes.go` scope. It is now covered by the reachability pin (it is one
   of the 7) but still carries no subsumption comment, because
   `TestEveryUTF8RecheckDeclaresItsSubsumption` scans `closed_shapes.go` only.
2. Nothing derives the set of digest-publishing Section 1.6 fixtures from the
   pinned document (`SPEC.md` is not vendored), so a future published digest
   cannot be reported as unpinned — exactly how these two were missed.
3. The graph's bound is real and stated, not closed. A func-typed struct field
   is the shortest construction that still evades it. Closing it needs type
   information (`go/types`), which would make the pin depend on resolving this
   module's imports at test time; that tradeoff was not taken here and is left
   as a disclosed limit rather than an implied guarantee.
