# BUG-260902-9uwnm7 — review verdict (round 2)

**Verdict: ACCEPTED.**

Change Request `CR-BUG-260902-9uwnm7-1` revision 1, base `422786cc`, candidate
tree `6b1134ed`. Verified locally: `git diff --quiet 6b1134ed` against the
worktree HEAD `8a0dced` succeeds, so the reviewed bytes are the committed bytes.
`git status --porcelain` was empty before, between every mutant, and at handoff;
every mutant was applied from a `/tmp` backup and reverted by file copy, never
by `git checkout`.

## What the acceptance criteria required, and where it is satisfied

| AC clause | Where |
| --- | --- |
| Each re-check carries a comment naming `decodeStrict`, or is deleted | `closed_shapes.go` 8 sites at :309, :356, :472, :1836, :1858, :1964, :2098, :2253, plus the shared note at :42-62, following the `internal/localstore/paths.go:234-236` block-comment convention |
| Clause sweep reports no unexplained survivor | `TestEveryUTF8RecheckDeclaresItsSubsumption`, which derives the sites from the AST rather than listing them |
| The `decodeStrict` dependency is stated explicitly | Stated in prose in the shared note **and** machine-checked by `TestNoExportedEntryPointHandsADecodedValueToAUTF8Recheck` + `TestDecodeStrictIsTheOnlyJSONDecodeInTheProductionPackage` |

The scope expansion from the four cited sites to all eight in the file is
correct and independently confirmed: the audit's line numbers (`:258`, `:1506`,
`:1525`, `:1760`) do not re-check UTF-8 on this tree, and no property separates
the cited subset from the other four.

## Gate attacked, not read

Every mutant below was applied to this tree and reverted; none is taken on the
producer's word.

| Mutant | Construction | Result |
| --- | --- | --- |
| H | exported `ValidateDecodedRecord(map[string]any)` reaching guards through `validateImmutableObjectShape` (the package's own function-value dispatch table) | **KILLED** — names all 7 guarded functions |
| I | the same body as an exported method on an exported type | **KILLED** |
| F3 (narrowing) | comment kept, but names `requireExactMembers` instead of `decodeStrict` | **KILLED** — `closed_shapes.go:2100` reported |
| G2 | second decoder: exported `[]byte` entry point calling `json.Unmarshal` | **KILLED** |
| M6 (mine) | new undocumented `utf8.ValidString` re-check added to `closed_shapes.go` | **KILLED by two tests** — `DeclaresItsSubsumption` at :2267 and the coverage ratio dropping to 7/8 with the new function named |
| M7 (mine) | new undocumented re-check as its own `return` in `core_records.go` | **KILLED** — by the pre-existing `TestEveryProductionRefusalGuardIsExecuted` |
| J2 (mine) | guard reached through a func-typed struct field | survives — **explicitly published as unmodelled** in both `closed_shapes.go` and `README.md` |

H and I are the two mutants that survived round 1; both now redden, so the
function-value and method edges are load-bearing and not decoration. F3 and M6
together show the documentation gate is not delete-only: narrowing it to name
the wrong subsuming validator fails just as a deletion does.

The coverage ratio is asserted rather than narrated —
`TestEveryUTF8RecheckIsCoveredByTheEntryPointPin` logs `7/7` and fails by name
if a guard falls out of reach of every exported entry point. I verified the
derived set independently: `nullableString`, `requireUTF8String`,
`validateExtensionValue`, `validateProviderIdentityRecord`,
`validateSessionBoardIdentity`, `validateSessionLaunchPlan`,
`validateSortedUniqueStrings`.

Absence handling is honest: all four new tests `t.Fatal` when their derivation
yields zero sites rather than passing vacuously, and the reused
`parsedProductionPackage` helper fatals on a parse failure instead of treating
an unreadable file as an empty one.

## Sibling scope in the same commit (BUG-260902-1874eo, part 1)

The two published digests are **not** recomputed from this encoder. Verified
independently, without the package:

```
printf '%s' '{"n":"9007199254740992"}'     | shasum -a 256
  -> bb80eb37329e0a7e980fe3638c9722c44ac3184f7488f20c28cf67ae0b5f4f96
printf '%s' '{"n":"18446744073709551615"}' | shasum -a 256
  -> b0ec84c6bb6a7c030549f17dd482975d09c40ff9e5f83d4438ebeac12d3b6331
```

Both equal the values asserted at `canonical_test.go:521` and `:536` and the
values quoted verbatim from `SPEC.md:303-304`. The pin is not tautological.

## Validation rerun by this review

| Command | Result |
| --- | --- |
| `gofmt -l .` | clean |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `go test ./... -count=1` | all 10 packages `ok`, exit 0 |

`internal/canonicaljson` under coverage reported 97.0% during the M6 mutant run
and returned to the clean tree afterwards.

## Findings recorded, not blocking

1. **`core_records.go:278` is a ninth re-check of the same class, still
   undocumented.** The producer disclosed this in `LOGBOOK.md` as
   `DISCLOSED, NOT FIXED` and scoped it out: `validateProviderIdentityRecord`
   is Section 2, while this bug's normative scope is §1.6. The disclosure is
   honest and the scoping is defensible, so it does not block acceptance — but
   the gap is real and I reproduced it. Mutant **M7b**, which adds
   `|| !utf8.ValidString(text)` as an extra *clause* (not a new return) inside
   `validateProviderIdentityRecord`, leaves the entire suite green.
   `TestEveryUTF8RecheckDeclaresItsSubsumption` reads only `closed_shapes.go`,
   while `utf8GuardedFunctions` derives package-wide, so a subsumed re-check
   clause added anywhere else in the package joins the survivor set silently.
   Follow-up: widen the documentation test to the production package and add the
   one comment at `core_records.go:278`.

2. **An exported package-level func-typed `var` is never enumerated as an entry
   point.** Mutant **J**,
   `var ValidateDecoded = func(object map[string]any) error { return validateSessionLaunchPlan(object) }`,
   hands a caller-supplied decoded map to a guard and survives the pin green:
   `exportedProductionEntryPoints` walks only `*ast.FuncDecl`, and
   `productionCallGraph` never walks a `FuncLit` bound to a package-level var.
   This is arguably already inside the published bound — an exported func var is
   a function value handed to another package and invoked there — and the
   guards are kept fail-closed precisely because the pin is bounded, so nothing
   in production is left unprotected. Naming this construction explicitly beside
   the other three would sharpen the claim.

Neither finding is a bypass of a live production gate: the re-checks were kept
rather than deleted, so an entry point the graph misses still hits a real
`utf8.ValidString` refusal. That fail-closed choice is what makes both findings
follow-up work instead of rework.
