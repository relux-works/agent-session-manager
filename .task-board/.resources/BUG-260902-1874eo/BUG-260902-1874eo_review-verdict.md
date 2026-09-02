# BUG-260902-1874eo — review verdict: CHANGES REQUESTED

Change Request `CR-BUG-260902-1874eo-1` revision 1.
Reviewed tree: `454c2db8640cc62f412f965a79183d675030df73` (worktree HEAD).

## On the empty `repository_delta`

The Change Request reports `repository_delta=empty` with base OID
`454c2db` and candidate tree `3a26ed7b`. That is a snapshot artifact, not an
empty delivery: `454c2db` **is** the producer's own commit
("BUG-260902-1874eo: pin-the-two-unpinned-normative-digests"), and
`git rev-parse 454c2db^{tree}` equals `3a26ed7b`, so the base was captured
after the leaf commit already landed on the story branch. The real reviewable
delta is the commit itself — 5 files, +429 lines — and it was reviewed as such.

```
git rev-parse 454c2db8640cc62f412f965a79183d675030df73^{tree}
3a26ed7bd6e3842aaa42e1c6293b929b8da141cb          # == candidate tree
```

This verdict therefore is NOT "the producer changed nothing". It is a verdict
on the reviewed commit content.

## Part 1 — pin the two unpinned normative digests: ACCEPTED on its own merits

Independently reproduced, not read:

| Check | Result |
| --- | --- |
| `sha256({"n":"9007199254740992"})` | `bb80eb37329e0a7e980fe3638c9722c44ac3184f7488f20c28cf67ae0b5f4f96` — equals the asserted value and the SPEC row quoted in the bug |
| `sha256({"n":"18446744073709551615"})` | `b0ec84c6bb6a7c030549f17dd482975d09c40ff9e5f83d4438ebeac12d3b6331` — equals the asserted value and the SPEC row quoted in the bug |
| Expected values | string literals quoted from `SPEC.md@28bf96d7`, whose SHA-256 the producer verified equals `internal/specpin.DocumentSHA256` (`562546d2...`, confirmed present at `internal/specpin/pin.go:30`). Not recomputed from the encoder. |
| Placement/shape | two subtests inside `TestCalculateObjectIdentityMatchesAXPublishedFixtures`, same `Canonicalize` + `scalar.SHA256Digest` shape as the pre-existing `NUM-SAFE-MAX` subtest |

Mutants re-run by the reviewer against the delivered tree:

| Mutant | Change | Observed |
| --- | --- | --- |
| A | `Canonicalize` returns `append(canonical, 0x20)` | `decimal_uint64 first unsafe integer`, `decimal_uint64 maximum`, `safe_integer_maximum`, `UTF-16 order` all RED |
| B2 | line 521 encoder input `...740992` -> `...740993` | only `decimal_uint64 first unsafe integer` RED |
| C2 | line 534 encoder input `...551615` -> `...551614` | only `decimal_uint64 maximum` RED |

Both new assertions are non-vacuous and individually decisive. No existing
assertion was weakened or deleted. Part 1's AC is fully met.

## Part 2 — the subsumed UTF-8 re-checks: one finding blocks acceptance

Confirmed good:

- All 8 `utf8.ValidString` sites in `closed_shapes.go` carry a subsumption
  comment naming `decodeStrict` (lines 300, 347, 463, 1827, 1849, 1955, 2089,
  2244) — a superset of the four the audit cited. Comment-only; no statement
  changed.
- Style matches the cited convention at `internal/localstore/paths.go:234-236`
  (inline `//` block directly above the guard, naming the subsuming function).
  Not a new style.
- Naming `decodeStrict` by function rather than by `canonical.go:311` is
  correct: `decodeStrict` is now at `canonical.go:324`, and this bug is itself
  an instance of a decayed line reference.
- Producer-claimed mutants D2, E3, F3, G2 all reproduce as RED against the
  delivered tree. The evidence table is honest.

### FINDING (blocking) — capability claim that does not reproduce

`README.md` publishes, and `closed_shapes.go` repeats, a guarantee that the
delivered pin does not provide:

> README.md: "Adding a second decoder or a map-taking entry point reddens the pin."
> closed_shapes.go: "Both conditions are machine-checked in utf8_subsumption_test.go,
> so the invariant cannot be broken silently."

**Mutant H — a map-taking exported entry point that does NOT redden the pin:**

```go
// appended to internal/canonicaljson/closed_shapes.go
func ValidateDecodedRecord(object map[string]any) error {
	schema, _ := object["schema"].(string)
	version, _ := object["schema_version"].(string)
	validator, ok := immutableObjectShapeValidators[schemaIdentityKey{schema: schema, version: version}]
	if !ok {
		return nil
	}
	return validator(object)
}
```

Result: `TestNoExportedEntryPointHandsADecodedValueToAUTF8Recheck` PASS,
`TestEveryUTF8RecheckDeclaresItsSubsumption` PASS,
`TestDecodeStrictIsTheOnlyJSONDecodeInTheProductionPackage` PASS, and
`go test ./internal/canonicaljson -count=1` **ok (19.1s)**. The mutant survives
the entire owning package suite. It is exactly the break the comments and the
README declare impossible: a caller-supplied, already-decoded map reaching
`validateSessionLaunchPlan`, `validateSessionBoardIdentity`,
`validateSortedUniqueStrings` and the rest with their UTF-8 guards documented
as unreachable.

**Root cause.** `productionCallGraph` records an edge only for
`call.Fun.(*ast.Ident)` naming a top-level declaration. The package's dominant
dispatch is the `immutableObjectShapeValidators` function-value table
(`closed_shapes.go:66`, invoked at `:182`), and registration passes the
validators as *arguments* to a local `register` closure. Neither is an edge, so
the graph cannot see the production path.

**Measured coverage of the pin (reviewer probe over the delivered tree):**

| Exported entry point | Guards reached by the graph | Byte-only params |
| --- | --- | ---: |
| `CalculateObjectIdentity` | *none* | true |
| `VerifyObjectIdentity` | *none* | true |
| `Canonicalize` | *none* | true |
| `CanonicalByteLength` | *none* | false |
| `ValidateObservationEvent` | nullableString, requireUTF8String, validateExtensionValue | true |
| `ValidateObservationStream` | nullableString, requireUTF8String, validateExtensionValue | true |

Derived guarded functions (7): `nullableString`, `requireUTF8String`,
`validateExtensionValue`, `validateProviderIdentityRecord`,
`validateSessionBoardIdentity`, `validateSessionLaunchPlan`,
`validateSortedUniqueStrings`.

Only 3 of the 7 are reachable from any exported entry point in the graph. The
other 4 — including 3 of the 8 `closed_shapes.go` sites this part exists to
resolve — are covered vacuously: no mutant on them can ever redden this test.
`CalculateObjectIdentity`, the package's principal identity entry point,
reaches zero guards in the graph.

Mutant D2 passed only because it called `validateExtensionValue` by direct
identifier — the one shape the graph can see. The mutant set never probed the
dispatch table, which is why the gap was not found.

This is not a live defect: production code is comment-only and the guards stay
fail-closed. It is a false published guarantee, in `README.md` and in the
comment block the "kept, not deleted" decision rests on.

### Required to accept

Either of these closes it; the first is preferred.

1. **Make the pin cover the dispatch path.** Add edges for function values
   assigned to or registered into package-level tables (an `*ast.Ident`
   appearing in argument or assignment position that names a top-level
   declaration is an edge, conservatively), so `ValidateDecodedRecord` above
   reddens `TestNoExportedEntryPointHandsADecodedValueToAUTF8Recheck`. Ship
   mutant H as the decisive mutant, and state the new guarded-entry-point count
   so the coverage is no longer implicit.
2. **Narrow the claim to what is checked.** Rewrite the `README.md` paragraph
   and the `closed_shapes.go` note to say the machine-check covers exported
   entry points that call a re-check through a statically resolvable call
   chain, and record that validators dispatched via
   `immutableObjectShapeValidators` are outside it. "Cannot be broken silently"
   must go.

Also re-run the reviewer probe and correct the results artifact: it reports
eight sites resolved in `closed_shapes.go` (true), but the derivation yields
seven guarded *functions* including `validateProviderIdentityRecord`, so the
"ninth re-check, undocumented" framing in `Found but not fixed` #1 is already
inside the derived set that `TestNoExportedEntryPointHandsADecodedValueToAUTF8Recheck`
consumes, even though the comment test scopes to `closed_shapes.go`. Worth one
sentence so the two counts stop disagreeing.

Part 1 needs no rework. Keep it exactly as delivered.

### FINDING (blocking, same root) — Mutant I: exported methods are skipped entirely

Both `utf8GuardedFunctions` and `productionCallGraph` skip any declaration with
`function.Recv != nil`. An exported *method* handing a decoded map to a guard is
therefore invisible on both sides of the pin — it is neither a derived guarded
function nor a derived entry point.

```go
// appended to internal/canonicaljson/closed_shapes.go
type DecodedValidator struct{}

func (DecodedValidator) ValidateExtensions(object map[string]any) error {
	return validateExtensionValue(object, 0)
}
```

Result: all three pins PASS, `go test ./internal/canonicaljson -count=1` **ok**.
This is the same root cause as Mutant H and should be closed by the same fix:
include methods in both derivations, or say in the note and the README that the
check covers package-level functions only.

## Orchestrator-requested probes: results

| Probe | Result |
| --- | --- |
| Does `productionCallGraph` miss call edges? | **Yes.** Function values registered into `immutableObjectShapeValidators` and method receivers are both unmodelled. Mutants H and I above. |
| Does `byteInputParameters` have false negatives? | Essentially no — it is over-strict, which is the safe direction. A named byte-slice type, `...byte`, or an extra `context.Context` parameter all return `false` and flag the entry point. The one hole is a zero-parameter exported function reading package state, which is contrived. Not a finding. |
| Is `utf8GuardedFunctions` complete and does it fail closed? | **Fails closed correctly.** It reuses the existing repo helpers `packageProductionFiles` / `parseProductionFile` (`constraint_inventory_test.go:83,109`), which `t.Fatal` on a read or parse error and on a zero-length file set; the test additionally `t.Fatal`s on an empty derived set. Good architectural fit — no new parsing convention was invented. Completeness is limited only by the `Recv != nil` skip above. |
| Highest-value untried mutant (orchestrator item 4) | **Ran it. The pin does not redden.** Mutants H and I are exactly that mutant in the two shapes this package actually uses, and both survive the full package suite. Per the orchestrator's own framing, "the subsumption comments are back to being unfalsifiable prose" for the validators reached through the dispatch table. |
| Digest provenance (orchestrator item 5) | Verified. Expected values are literals quoted from the pinned document, and both reproduce from `shasum` outside this implementation. Nothing was recomputed from the encoder. |
| Mutant discipline (orchestrator item 3) | Every mutant above was `grep`-confirmed present in the file before the run, and the tree was `git diff --stat`-confirmed clean after each revert. No `-run` filter was trusted without a matching `=== RUN` line. |

## Validation run by the reviewer on the delivered tree

| Command | Result |
| --- | --- |
| `git status --short` / `git diff --stat` after all mutants reverted | clean, identical to `454c2db` |
| `gofmt -l internal/ cmd/` | no output |
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `go test ./... -count=1` | all packages `ok` |
| `go test ./internal/canonicaljson -count=1` with mutant H applied | `ok` — **survivor** |
| `go test ./internal/canonicaljson -count=1` with mutant I applied | `ok` — **survivor** |

## Verdict

`changes_requested` -> `to-dev`. One bounded fix in part 2 closing both
Mutant H and Mutant I; part 1 is accepted as delivered and must not be touched.
