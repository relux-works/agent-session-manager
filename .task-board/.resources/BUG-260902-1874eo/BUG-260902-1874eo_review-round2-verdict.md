# BUG-260902-1874eo + BUG-260902-9uwnm7 — Review round 2

**Verdict: CHANGES REQUESTED → `to-dev`.**
Change Request `CR-BUG-260902-1874eo-2` revision 2. Reviewer run `RUN-260902-79a17a`.

## The empty delta is a snapshot artifact, not an empty delivery

`repository_delta=empty` because the CR base OID **is** the leaf commit:

```
base OID                8a0dcedf7d1d4a932eae335df2772effe3923cee
8a0dced^{tree}          6b1134ed361b7617fda0f6a48cbc9ab3aba0458a
candidate tree          6b1134ed361b7617fda0f6a48cbc9ab3aba0458a   (identical)
git rev-list --count main..HEAD = 1                                (one commit past checkpoint 422786c)
```

Same artifact as round 1. The reviewable content is commit `8a0dced`: 5 files, +732 lines
(`LOGBOOK.md`, `README.md`, `canonical_test.go`, `closed_shapes.go`, `utf8_subsumption_test.go`),
reviewed in full. No repository change was the correct outcome *for the CR snapshot*; the work
itself is in the commit and was judged there.

## PART 1 — ACCEPTED. Do not touch it in rework.

Independently re-verified end to end, not taken from the producer's report.

| Check | Result |
| --- | --- |
| Pinned SPEC document SHA-256 | `562546d240f0…ac484a` == `internal/specpin.DocumentSHA256` ✅ |
| SPEC.md:303 / :304 at commit `28bf96d7` | contain NUM-U64-STRING / NUM-U64-MAX with the quoted digests ✅ |
| Comment quotation | verbatim against the pinned lines ✅ |
| `shasum -a 256` of `{"n":"9007199254740992"}` | `bb80eb37329e0a7e980fe3638c9722c44ac3184f7488f20c28cf67ae0b5f4f96` ✅ |
| `shasum -a 256` of `{"n":"18446744073709551615"}` | `b0ec84c6bb6a7c030549f17dd482975d09c40ff9e5f83d4438ebeac12d3b6331` ✅ |
| Style vs. existing NUM-SAFE-MAX at canonical_test.go:496 | identical shape ✅ |

Non-vacuity reproduced by this reviewer, mutant present in the file confirmed before each measurement:

- **A** — `Canonicalize` appends one byte: `safe_integer_maximum`, `decimal_uint64_first_unsafe_integer`,
  `decimal_uint64_maximum` and `UTF-16_order` all FAIL. Both new subtests redden on encoder drift.
- **C2** — one byte of the NUM-U64-MAX fixture (`…615` → `…614`): **exactly** `decimal_uint64_maximum`
  fails, the sibling subtest stays green. Individually decisive; neither new assertion is carried by the other.

The expectations are literals quoted from the pinned document, so they cannot drift with the encoder.
AC met.

## PART 2 — the round-1 blocker IS fixed, and three new survivors of the same class are not

### Round-1 findings: confirmed fixed

Reproduced by this reviewer, not accepted from the report:

| Mutant | Construction | Round 1 | Now |
| --- | --- | --- | --- |
| H | exported `ValidateDecodedRecord(map[string]any)` via the `immutableObjectShapeValidators` dispatch table | SURVIVED | **RED**, names all 7 guards |
| I | exported method taking a decoded map | SURVIVED | **RED** |
| I2 | exported method, `[]byte` param, decoded receiver | — | **RED** |
| E3 | one subsumption comment deleted | — | **RED** (`closed_shapes.go:472`) |
| F3 | comment kept but names `requireExactMembers` | — | **RED** (`closed_shapes.go:474`) |
| O | func-typed struct field | — | SURVIVES **as disclosed** — the bound is stated honestly |

Coverage is now genuinely `7/7` and asserted (`utf8_subsumption_test.go:465`). The function-value and
method edges are real fixes and the disclosed bound for mutant O is accurate.

### FINDING 1 (blocking) — a byte-taking entry point makes a documented-subsumed guard LIVE, all pins green

`byteInputParameters` checks the **shape of the parameters**, not that those bytes ever reach
`decodeStrict`. A `[]byte → string` conversion is not a JSON decode, so it passes both halves of the pin.

```go
// mutant J, in the production package
func ValidateSortedNames(raw [][]byte) error {
	values := make([]any, 0, len(raw))
	for _, item := range raw {
		values = append(values, string(item))   // bytes become a Go value, no decodeStrict
	}
	return validateSortedUniqueStrings(values, "names")
}
```

Measured:

- `TestNoExportedEntryPointHandsADecodedValueToAUTF8Recheck` — **green**
- `TestEveryUTF8RecheckIsCoveredByTheEntryPointPin` — **green**, still reports 7/7
- `TestDecodeStrictIsTheOnlyJSONDecodeInTheProductionPackage` — **green**
- `TestEveryUTF8RecheckDeclaresItsSubsumption` — **green**
- **full package suite `ok` (18.5s)**

And the guard is not merely reachable in theory — it is the term that refuses. Probe driven through
the exported entry point:

```
guard fired from exported entry point:
  invalid canonical object identity: member names[0] must be a UTF-8 string
```

So the comment on `validateSortedUniqueStrings` — "subsumed by decodeStrict" — is **false** under a
change the pin signs off on. This is not one of the three disclosed unmodelled constructions
(reflection, cross-package function value, func-typed struct field); it is the *ordinary* shape in
this package. The published text is specific and it is what fails:

- `README.md`: "no exported function or method may hand an already-decoded value to one of those
  re-checks, and `decodeStrict` must remain the only place in the package where bytes become Go
  values… That subsumption is machine-checked, not only asserted in prose."
- `closed_shapes.go`: "an exported entry point that accepted an already-decoded map, or a second JSON
  decoder anywhere in the package, would make every one of them false again. **Both conditions are
  machine-checked**."

"the only place where bytes become Go values" is the right invariant. Nothing checks it.

### FINDING 2 (blocking) — a second JSON decoder does not redden the pin

`TestDecodeStrictIsTheOnlyJSONDecodeInTheProductionPackage` matches `selector.X` only when the
identifier is literally `json`. It resolves no import path.

```go
// mutant L2, in the production package
import stdjson "encoding/json"

func decodeLoose(input []byte) any {
	var value any
	if stdjson.Unmarshal(input, &value) != nil {
		return looseDecodeSink
	}
	return value
}
```

Decoder pin **green**; **full package suite `ok` (17.8s)**. The same hole admits any decoder from a
different package (`encoding/json/v2`, a third-party decoder) — the realistic long-term case.

README states flatly: "Adding a second decoder reddens the pin." It does not reproduce.

(An earlier variant, mutant L, did fail — but via `TestEveryProductionRefusalGuardIsExecuted`,
an unrelated pre-existing gate that fired on the uncovered refusal return, not on the decoder.
L2 removes that accident and the decoder pin still sees nothing.)

### FINDING 3 (minor, fix or narrow) — a new re-check written as `utf8.Valid` is invisible

`utf8GuardedFunctions` and `TestEveryUTF8RecheckDeclaresItsSubsumption` both key on `ValidString`
with the package identifier spelled `utf8`. A brand-new, entirely undocumented re-check added to
`closed_shapes.go` as `utf8.Valid([]byte(text))` (**mutant M**) leaves every pin green.

`utf8.Valid` is not hypothetical here — `decodeStrict` itself uses it (`canonical.go:328`). The file
claims at `utf8_subsumption_test.go:106`: "Deriving them means a new re-check is covered by this pin
the moment it is written." A re-check written that way silently joins the undocumented survivors,
which is the exact condition this Story exists to remove. An aliased `unicode/utf8` import is the
same class.

## What rework must do

Part 1 is closed. For part 2, either strengthen or narrow — but the published text and the machine
check must agree, which is the standing bar this round and round 1 both apply.

1. **Finding 1.** Preferred: extend the pin so a byte-taking exported entry point must actually route
   its bytes through `decodeStrict`, and land mutant J as its decisive mutant.
   *Caveat, and the reason this is not prescribed as a one-liner:* the producer already removed a
   transitive-`decodeStrict`-reachability arm because mutant G survived it — `validateExtensionsObject`
   reaches `decodeStrict` via `canonicalByteBound` → `CanonicalByteLength` → `Canonicalize`, so plain
   transitive reachability is satisfied by a call that never touches the caller's bytes. That
   disclosure is correct and a naive re-add will not work. If no check can be made to hold, the
   minimum acceptable outcome is to **name the `string(bytes)` construction in the bound** alongside
   reflection / cross-package function values / func-typed fields, and drop the claim that
   "`decodeStrict` must remain the only place where bytes become Go values" is machine-checked.
2. **Finding 2.** Resolve the import *path* (`encoding/json` and any other JSON decoder) rather than
   the identifier `json`, and land mutant L2 red. If the check stays identifier-based, delete
   "Adding a second decoder reddens the pin" from `README.md` and the matching sentence in
   `closed_shapes.go`.
3. **Finding 3.** Either derive `utf8.Valid`/`utf8.ValidRune` and aliased `unicode/utf8` imports too,
   or narrow the `utf8_subsumption_test.go:106` comment to say the derivation covers `utf8.ValidString`
   only.
4. Do not weaken or delete any existing assertion. Do not touch `canonical_test.go`. Confirm every
   mutant is present in the file before believing a green or a red.

## Found, not requiring a fix here

- `core_records.go:278` (`validateProviderIdentityRecord`) carries a ninth re-check of the same class,
  outside `closed_shapes.go` scope and outside `TestEveryUTF8RecheckDeclaresItsSubsumption`'s file
  filter. The producer disclosed it. It is now the only undocumented `ValidString` survivor in the
  package. Correctly out of scope; worth its own board item.
- The brief's line numbers (`closed_shapes.go:258/1506/1525/1760`, `canonical.go:311`) are stale, and
  `internal/canonicaljson/projection.go` — cited as a style precedent — does not exist on this tree.
  The producer re-derived rather than trusting the list, which was the right call. Any sibling item
  from the same audit needs its references re-derived before work starts.
- The coarse "indirect call reaches every address-taken function" edge is an **over**-approximation,
  which is the safe direction for this property. It does mean the 7/7 ratio is partly carried by that
  blanket edge rather than by seven distinct concrete paths. Not a defect; noted so the ratio is not
  over-read.
- `parsedProductionPackage` / `packageProductionFiles` fail closed on a parse error and on a zero-file
  derivation (`t.Fatal`). Checked, per the orchestrator's question. `byteInputParameters` also fails
  closed on named types, generics and variadics — an unrecognised parameter type yields `false`, i.e.
  an error, not a pass.

## Reviewer validation, clean tree

Tree restored and verified byte-identical to the candidate after **every** mutant
(`git rev-parse HEAD^{tree}` = `6b1134ed361b7617fda0f6a48cbc9ab3aba0458a`, `git status --short` empty).

| Command | Result |
| --- | --- |
| `gofmt -l internal/` | clean |
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `go test ./... -count=1` | all 10 packages `ok` |

The suite being green is not the finding. The finding is that three changes which the shipped text
says would redden it do not.
