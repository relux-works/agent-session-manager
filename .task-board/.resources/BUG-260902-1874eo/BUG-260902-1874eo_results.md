# BUG-260902-1874eo + BUG-260902-9uwnm7 — implementation evidence

Delivered as one change, as the board note requires.

## Part 1 — pin the two unpinned normative digests

### Pinned-source provenance

`SPEC.md` was extracted from the local clone of `relux-works/agent-session-manager-spec`
at the pinned commit and its SHA-256 verified against `internal/specpin.DocumentSHA256`
before any line was read:

```
git -C <spec-clone> show 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c:SPEC.md > SPEC.pinned.md
shasum -a 256 SPEC.pinned.md
562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a  SPEC.pinned.md

internal/specpin/pin.go:
DocumentSHA256 = "562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a"
```

Lines 303 and 304, verbatim:

```
| <code>NUM-U64-STRING</code> | <code>{"n":"9007199254740992"}</code> | Accept only when <code>n</code> is typed <code>decimal_uint64</code>; SHA-256 <code>bb80eb37329e0a7e980fe3638c9722c44ac3184f7488f20c28cf67ae0b5f4f96</code> |
| <code>NUM-U64-MAX</code> | <code>{"n":"18446744073709551615"}</code> | Accept only when <code>n</code> is typed <code>decimal_uint64</code>; SHA-256 <code>b0ec84c6bb6a7c030549f17dd482975d09c40ff9e5f83d4438ebeac12d3b6331</code> |
```

Both reproduce independently of this implementation:

```
printf '%s' '{"n":"9007199254740992"}'     | shasum -a 256 -> bb80eb37329e0a7e980fe3638c9722c44ac3184f7488f20c28cf67ae0b5f4f96
printf '%s' '{"n":"18446744073709551615"}' | shasum -a 256 -> b0ec84c6bb6a7c030549f17dd482975d09c40ff9e5f83d4438ebeac12d3b6331
```

Confirmed absent before the change: `grep -rn 'bb80eb37\|b0ec84c6' --include='*.go' --include='*.json' --include='*.md' .`
matched only the board README for this bug.

### Change

`internal/canonicaljson/canonical_test.go`, two subtests added inside
`TestCalculateObjectIdentityMatchesAXPublishedFixtures`, in the same shape as the
existing `NUM-SAFE-MAX` assertion at that test's first subtest. Production subject is
`Canonicalize` + `scalar.SHA256Digest`, the same entry point the existing assertion drives.
Expected values are string literals quoted from the pinned lines, never recomputed.

### Decisive mutants

| Mutant | Change | Result |
| --- | --- | --- |
| A | `Canonicalize` appends one byte to its return value | both new subtests RED (and the two pre-existing digest subtests RED) |
| B2 | NUM-U64-STRING encoder input `...992` -> `...993` | only `decimal_uint64 first unsafe integer` RED |
| C2 | NUM-U64-MAX encoder input `...615` -> `...614` | only `decimal_uint64 maximum` RED |

B2 and C2 establish that each assertion is individually decisive; neither is carried by the other.

A first attempt at B/C mutated the verbatim SPEC quote in the comment instead of the
encoder input (`perl -0p s///` without `/g` hits the first occurrence, which is the comment)
and both stayed green. Recorded here because a green mutant that never reached production
code is the exact false-negative this evidence bar exists to catch.

## Part 2 — the subsumed UTF-8 re-checks

### The cited line numbers are stale

The bug names `closed_shapes.go:258, :1506, :1525, :1760`. None of those lines re-checks
UTF-8 on `main`, `origin/main`, or this branch. They match commit `7b94c9ad`
(five commits back on this file). Mapping to current lines:

| Audit line | Current line | Function |
| --- | --- | --- |
| 258 | 302 | `validateSessionLaunchPlan` argv element |
| 1506 | 1830 | `validateExtensionValue` string case |
| 1525 | 1852 | `validateExtensionValue` object key |
| 1760 | 2091 | `requireUTF8String` |

(Current lines are post-change, i.e. after the comments were inserted.)

The same commit carries EIGHT `utf8.ValidString` sites, not four, with no property
separating the cited subset from the rest. All eight are resolved, because the AC
("the clause sweep no longer reports them as unexplained survivors") is not met while
four identical survivors remain. The four not cited by the audit are
`validateSessionLaunchPlan` env_literals value, `validateSessionBoardIdentity` remote_url,
`nullableString`, and `validateSortedUniqueStrings`.

Similarly, `canonical.go:311` is now `canonical.go:328`. The comments name `decodeStrict`
by function rather than by line, deliberately: this bug is itself an instance of a
line-number reference decaying.

### Resolution: kept and documented, not deleted

The subsumption is real. `decodeStrict` refuses input that is not valid UTF-8 before any
value exists, and it is the only place in the package where bytes become Go values, so no
decoded string or key can reach a validator invalid.

They are kept because the argument is a package-internal invariant, not a property of the
eight validators:

- `CanonicalByteLength(value any)` already demonstrates the package can export a non-`[]byte`
  entry point. A future map-taking exported entry would make all eight comments false at once.
- Deleting them rewrites refusal statements pinned by `refusal_guards_test.go`; two of the
  eight are already registered there as `invalidUTF8Refusal`.

The AC requires the dependency to be stated explicitly. It is stated in the shared note at
`closed_shapes.go` and, unusually, machine-checked rather than left as prose.

### Change

- `internal/canonicaljson/closed_shapes.go`: one shared "Subsumed UTF-8 re-checks" note after
  the package var block carrying the full reachability argument, and a short comment at each
  of the eight sites naming `decodeStrict` and stating which sibling terms remain reachable.
  Comment-only; no statement changed. Style follows the block-comment convention at
  `internal/localstore/paths.go:234-236`.
- `internal/canonicaljson/utf8_subsumption_test.go` (new): three derived pins.
  - `TestEveryUTF8RecheckDeclaresItsSubsumption` — every `utf8.ValidString` in
    `closed_shapes.go` must carry a preceding comment naming `decodeStrict`.
  - `TestNoExportedEntryPointHandsADecodedValueToAUTF8Recheck` — call graph over production
    sources; any exported function reaching a derived re-check must take `[]byte`/`[][]byte`.
  - `TestDecodeStrictIsTheOnlyJSONDecodeInTheProductionPackage` — `json.NewDecoder` and
    `json.Unmarshal` may appear only inside `decodeStrict`.

  The guarded functions and the decode sites are DERIVED, not listed, so a re-check or a
  decoder added later is covered the moment it is written.

### An arm that was removed because it proved nothing

The first version asserted "an exported entry point reaching a re-check must also reach
`decodeStrict`". Mutant G survived it: `validateExtensionsObject` -> `canonicalByteBound` ->
`CanonicalByteLength` -> `canonicalEncoding` -> `Canonicalize` -> `decodeStrict`, so
transitive reachability is satisfied by a call that never touched the caller's bytes.
Reaching a decoder downstream is a different fact from having been decoded by it. That arm
was replaced by the decoder-monopoly test, which kills the same mutant on a property it can
actually decide.

### Decisive mutants

| Mutant | Change | Result |
| --- | --- | --- |
| D2 | exported `ValidateDecodedExtensions(map[string]any)` reaching `validateExtensionValue` | `TestNoExportedEntryPointHandsADecodedValueToAUTF8Recheck` RED |
| G2 | exported `ValidateRawExtensions([]byte)` decoding with `json.Unmarshal` | `TestDecodeStrictIsTheOnlyJSONDecodeInTheProductionPackage` RED |
| E3 | one subsumption comment block deleted at `requireUTF8String` | `TestEveryUTF8RecheckDeclaresItsSubsumption` RED |
| F3 | narrowing — that comment kept but names `requireExactMembers`, never `decodeStrict` | `TestEveryUTF8RecheckDeclaresItsSubsumption` RED |

F3 is the narrowing mutant required by the standing criteria: it proves the pin checks WHICH
validator is named, not merely that a comment exists.

No existing assertion was weakened or deleted in either part.

## Validation

Every command run directly as a standalone process; real exit codes.

| Command | Exit |
| --- | ---: |
| `gofmt -l internal/` (no output) | 0 |
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `go run ./internal/traceability/cmd/tracecheck` | 0 |
| `go generate ./internal/catalog` + `git diff --exit-code -- internal/catalog/catalog_gen.go` | 0 |
| `go test ./... -count=1` | 0 |
| `go test ./... -cover -count=1` | 0 |
| `go test ./internal/canonicaljson -run=^$ -fuzz=^FuzzCanonicalizeRoundTrip$ -fuzztime=100x -parallel=1` | 0 |

`internal/canonicaljson` coverage 97.2% of statements, unchanged from before the change.
`tracecheck`: `contracts=60 normative_sections=36 acceptance_cases=43 fixtures=30
compatibility_contracts=55 assigned_scopes=0`.

## Found but not fixed

1. `internal/canonicaljson/core_records.go:278` (`validateProviderIdentityRecord`,
   `opaque_identity` values) carries a NINTH `utf8.ValidString` re-check of exactly the same
   class. Left alone because the bug scopes part 2 to `closed_shapes.go`. It is now the only
   undocumented survivor of this class in the package; `TestEveryUTF8RecheckDeclaresItsSubsumption`
   deliberately scopes to `closed_shapes.go` and does not report it. Worth its own board item.
2. Nothing derives the SET of digest-publishing Section 1.6 fixtures from the pinned document.
   `SPEC.md` is not vendored and `internal/specpin/v0.5.0.lock.json` pins three conformance
   fixtures, none of them the Section 1.6 boundary rows. So a future added published digest
   cannot be reported as unpinned by any test — exactly how these two were missed. A completeness
   gate would need the Section 1.6 fixture table extracted into `testdata/`, the way the Session
   Event payload members already are.
3. The audit's line numbers were five commits stale and its site count was half the real
   population. If the same audit produced other findings on this package, their line references
   and counts should be re-derived before they are worked.
