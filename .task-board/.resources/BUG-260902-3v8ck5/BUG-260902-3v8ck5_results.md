# BUG-260902-3v8ck5 - measure declared byte bounds on canonical bytes

## What was wrong

Two declared byte bounds were measured on Go's HTML-escaped JSON encoding
instead of on canonical bytes:

| Site | Before | Effect |
| --- | --- | --- |
| `internal/canonicaljson/closed_shapes.go` Launch Plan argv | `len(json.Marshal(argv))` | up to 6x over-measurement |
| `internal/config/validation.go` extensions | `len(json.Marshal(extensions))` | same |

`encoding/json` rewrites each of the single bytes `<`, `>` and `&` into a
six-character escape, and U+2028/U+2029 from three canonical bytes into six.
The bound actually in force was therefore a property of `encoding/json`, not the
pinned 65,536, and a conforming document was refused. The third site,
`validateExtensionsObject`, already measured canonical bytes, so one repository
carried both the defect and its fix.

It failed closed: no oversized document was ever admitted.

## What changed

New `internal/canonicaljson/declared_byte_bounds.go`:

- `CanonicalByteLength(value any) (int, error)` - the single measurement. It
  encodes with `SetEscapeHTML(false)` and then runs the RFC 8785 transform.
  `SetEscapeHTML(false)` alone is **not** sufficient: `encoding/json` escapes
  U+2028 and U+2029 regardless of that setting, so the canonical transform is
  what makes the measurement independent of the encoder.
- `canonicalByteBound(name string, value any, maximum int) error` - the shared
  refusal gate both canonicaljson sites now go through.

Call sites:

- `validateSessionLaunchPlan` - `canonicalByteBound("Session Record Launch Plan argv", argv, 65_536)`
- `validateExtensionsObject` - `canonicalByteBound("extensions object", extensions, 65_536)`
- `internal/config/validation.go:validateExtensions` - `canonicaljson.CanonicalByteLength(extensions)`

`internal/config` now imports `internal/canonicaljson`; there is no cycle
(canonicaljson imports only `catalog` and `scalar`).

Because `canonicalByteBound` has the derived bound-helper shape
(`name string` + `maximum int`), both byte bounds became first-class derived
obligations in `declared_bounds_test.go`. Two `boundaryConstraintCase` entries in
`boundary_constraints_test.go` now claim them, so changing either literal
reddens the derivation gate as well as the new tests.

`refusal_guards_test.go` unreachable declarations were updated: the three
serialize/canonicalize guards at the old call sites are replaced by the three
guards inside the shared helper, all under the existing `decodedValueRefusal`
reason (the values reaching them were produced by `decodeStrict`).

## New tests

`internal/canonicaljson/canonical_byte_bound_measurement_test.go`

- `TestMeasurementFillerWidthsAreTheCanonicalOnes` - pins the canonical width of
  every fill character the fixtures use, so a wrong width cannot silently
  produce a fixture that misses its boundary.
- `TestLaunchPlanArgvByteBoundIsMeasuredOnCanonicalBytes`
- `TestExtensionsObjectByteBoundIsMeasuredOnCanonicalBytes`
- `TestCanonicalByteLengthIgnoresHTMLAndLineSeparatorEscaping`

`internal/config/extension_canonical_byte_bound_test.go`

- `TestConfigurationExtensionByteBoundIsMeasuredOnCanonicalBytes`
- `TestConfigurationAndIdentityBoundsShareOneMeasurement`

Both tables run six rows: `<`, `>`, `&`, U+2028, U+2029 and one
encoding-neutral ASCII control row. Every row asserts:

1. the fixture is **exactly** at the SPEC literal 65,536 canonical bytes,
   measured through `Canonicalize` rather than through the helper under test;
2. the accepted at-limit fixture is **over** the bound when measured the escaped
   way (`assertEscapedMeasurementWouldRefuse`) - this is the assertion that makes
   a mutant restoring escaped measurement fail rather than pass;
3. accept at the limit and refuse exactly one canonical byte past it, at the
   production entries;
4. for canonicaljson, that the one-past refusal came from the byte bound itself
   and named the canonical size it measured, not from some other gate.

The SPEC literals are local constants
(`specificationDeclaredByteMaximum`, `specificationExtensionCanonicalMaxBytes`),
never `maxConfigExtensionBytes` or the production literal.

Production entries driven: `CalculateObjectIdentity` and `VerifyObjectIdentity`
(canonicaljson), `EncodeCurrent` (writer) and `Load` (reader, via
`loadConfigDocument`) for configuration. The reader-side one-past case edits the
encoded TOML document, because the writer refuses the over-limit configuration
before it can produce one.

## Mutation sweep

Full transcript: `BUG-260902-3v8ck5_mutation-sweep.log`. Every mutant was applied
to the working tree, the named command run, and the tree restored.

| # | Mutant | Command exit | Reddened by |
| --- | --- | ---: | --- |
| M1 | `CanonicalByteLength` returns `len(json.Marshal(value))` | 1 (both packages) | all four canonicaljson tests; both config tests |
| M2 | argv bound narrowed to 65535 | 1 | at-limit acceptance + derived-bounds gate |
| M3 | argv bound widened to 65537 | 1 | one-past refusal + derived-bounds gate |
| M4 | extensions bound narrowed to 65535 | 1 | at-limit acceptance + derived-bounds gate |
| M5 | extensions bound widened to 65537 | 1 | one-past refusal + derived-bounds gate |
| M6 | config bound narrowed to 65535 | 1 | `EncodeCurrent` at-limit acceptance |
| M7 | config bound widened to 65537 | 1 | `EncodeCurrent` one-past refusal |
| M8 | config site restores `len(json.Marshal(extensions))` | 1 | `EncodeCurrent` at-limit acceptance |

Narrow/widen mutants matter here because a delete-only mutant would prove the
gate exists and say nothing about where it sits.

## Validation

Every command run as a standalone process in the Story worktree; exit codes are
the real ones.

| Command | Exit |
| --- | ---: |
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `gofmt -l internal/` (no output) | 0 |
| `go test ./... -cover -count=1` | 0 |
| `go run ./internal/traceability/cmd/tracecheck` | 0 |
| `go generate ./internal/catalog` | 0 |
| `cataloggen ... -check` | 0 |
| `go test ./internal/canonicaljson -fuzz=^FuzzCanonicalizeRoundTrip$ -fuzztime=100x` | 0 |
| `go test ./internal/canonicaljson -fuzz=^FuzzObjectIdentityRepresentationInvariant$ -fuzztime=100x` | 0 |
| `go test ./internal/canonicaljson -fuzz=^FuzzClosedIdentityShapeRefusal$ -fuzztime=100x` | 0 |
| `go test ./internal/canonicaljson -fuzz=^FuzzObservationEventRefusal$ -fuzztime=100x` | 0 |
| `go test ./internal/scalar -fuzz=^FuzzScalarProductionEntries$ -fuzztime=100x` | 0 |

tracecheck: `traceability ok: contracts=60 normative_sections=36 acceptance_cases=43 fixtures=30 compatibility_contracts=55 assigned_scopes=0`

### Coverage

Baseline measured in a detached worktree at `HEAD` (`f20aeec`), same commands.

| Package | Before | After |
| --- | ---: | ---: |
| canonicaljson | 97.2% | 97.2% |
| config | 94.7% | 94.7% |
| catalog | 97.6% | 97.6% |
| catalog/cmd/cataloggen | 79.3% | 79.3% |
| cataloggen | 83.9% | 83.9% |
| localstore | 85.6% | 85.6% |
| scalar | 90.1% | 90.1% |
| specpin | 85.1% | 85.1% |
| traceability | 85.0% | 85.0% |
| traceability/cmd/tracecheck | 87.5% | 87.5% |

Exact canonicaljson statement counts from the coverage profiles:

| | total | covered | uncovered | pct |
| --- | ---: | ---: | ---: | ---: |
| before (`HEAD`) | 2233 | 2170 | 63 | 97.179 |
| after | 2238 | 2175 | 63 | 97.185 |

An intermediate shape that kept the byte-bound refusal inline at both call sites
measured 64 uncovered statements (97.143). Folding the two call sites onto the
shared `canonicalByteBound` gate removed that extra unreachable guard, which is
why the final shape has the same uncovered count as the baseline.

## Notes for review

- `internal/config` gained a dependency on `internal/canonicaljson`. That is the
  point of the AC ("the two sites share one helper"), and it is the direction
  that has no cycle.
- The three unreachable-refusal declarations that moved are unreachable for the
  same reason as before: every value reaching them was produced by `decodeStrict`
  or by `validateExtensionValue`, so neither `encoding/json` nor the RFC 8785
  transform can fail on it.
