# TASK-260906-vmzk0y — own-provider-protocol-v3-envelope-and-error-lift (round 6)

Status: ready for review (handed off to review; board status `to-review`).

Candidate tree: `817b1a68d844f7f5b6734197526a6661682df164`
(verified by `git ls-tree -r`: the round-5 file
`internal/terminalbackend/digit_range_census_test.go` is absent, the new
`internal/terminalbackend/digit_guard_census_test.go` is in-tree, and
`git hash-object` of worktree `digit_guard_census_test.go`,
`descriptor_test.go`, `manifest_test.go`, and `surrogate_test.go`
byte-for-byte equals the tree blobs. Built via a detached index —
`GIT_INDEX_FILE` copy + `read-tree HEAD` + `add -A` + `write-tree` — so
the worktree index is untouched.)

Round 6 answers review-verdict-rev5: F1 (the `value > 100` pre-multiply
guard narrowed to its own arithmetic edge survives) and F2 (the census
lists its own sites and scopes out the accumulation bounds) — the fourth
consecutive round of one class. Production has been correct since round
2 and is untouched this round; the instrument is replaced, not patched:
`digit_range_census_test.go` (60-line header plus one test) is deleted,
and `digit_guard_census_test.go` derives all 9 digit-guard sites of
both shapes from production AST in one enumeration. Everything the
reviewer marked verified in rev5 stands and is not re-litigated here.

## F1 (blocking, closed) — pre-multiply guard pinned at its own edge

`descriptor.go:238` (`internal/terminalbackend/descriptor.go`). The
guard `value > 100` narrowed to `value > 922337203685477580`
(`floor(MaxInt/10)`) admitted
`columns`/`rows` = `9223372036854775808000000000000000001` (37 digits)
as 1 while every shipped vector still refused.

`TestParseProviderDescriptorGeometryOverflowVectors` gains the 37-digit
witness on both members asserting `descriptor geometry bound`, with a
doc comment stating the edge it pins. Measured: under the F1 mutant the
test FAILS (whole-package run: 2 failures — the witness test and the
census, which also sees the edited bound); on pristine code the suite is
green. The witness reddens thresholds in the dangerous window
(verified at `922337203685477580`, where it fails) and stays green at
the equivalents the reviewer named (`100`, `1000`, `10000`, `1e9`,
`92233720368547758` — the `1000` case is measured as R6-D06 below:
behavioural suite fully green, only the census reddens).

## F2 (blocking, closed) — one derived census over both shapes

`internal/terminalbackend/digit_guard_census_test.go`
(`TestDigitCensusCoversEveryLeafGuard` + kept adjacency test, now
`TestDigitCensusAdjacentsNeverReachGeometryGate`). It parses every
production file of `internal/terminalbackend` and `internal/provhost`
(directory-derived — a new file is scanned with no registration step)
and enumerates both shapes in one pass:

- char: an ordering comparison against a decimal digit rune
  (`c < '0' || c > '9'` and spellings, refuse- or accept-form,
  including the digit case/branch of a hex decoder);
- bound: an ordering comparison against a decimal accumulator — an
  identifier the same function accumulates through `* 10`
  (`value > N` / `major > N`, before or after the multiply).

Keys are structural (rune values, `* 10` accumulation in the same
function, printer-normalized chain text) — never identifier names, file
lists, or line numbers. The census fails closed in three directions: an
unregistered site, an orphan row (plus empty claims, cross-kind fields,
unknown kinds, rows with no test, rows naming a missing test), and an
unclassifiable site (equality against a digit rune, digit/accumulator
mixing, mixing with unrelated comparisons, accumulators on both sides,
negated digit/accumulator content). Zero-file scans, zero-site
derivations, and duplicate derived keys are fatal.

Derived denominator, 9/9 rowed across 19 production files:

| Site | Shape | Row declares |
|---|---|---|
| terminalbackend/descriptor.go `descriptorGeometry` `digit < '0' \|\| digit > '9'` | char | reachable rejected class `{-, +, ., e, E}` (json.Number domain; `/`/`:` die in the decoder, measured) |
| terminalbackend/manifest.go `readUTF16EscapeUnit` `digit >= '0' && digit <= '9'` | char | hex digit case admits exactly 0-9; non-hex reaches the refuse arm |
| provhost/protocol.go `parseMajor` `digit < '0' \|\| digit > '9'` | char | every non-digit byte; `/`/`:` witnessed in major position |
| provhost/protocol.go `parseMajor` `rest[i] < '0' \|\| rest[i] > '9'` | char | every non-digit byte; `/`/`:` witnessed in both rest positions |
| provhost/surrogate.go `readHexUnit` `b >= '0' && b <= '9'` | char | hex digit branch admits exactly 0-9; non-hex reaches the refuse arm |
| terminalbackend/descriptor.go `descriptorGeometry` `value > 100` | bound | threshold 100 (no wrap: value ≤ 100 at the multiply); witness the 37-digit literal |
| terminalbackend/descriptor.go `descriptorGeometry` `value < 1 \|\| value > 1000` | bound | lower 1 / upper 1000; witnesses 0/1, 1000/1001, 65535 |
| terminalbackend/terminalbackend.go `semverMajor` `major > (math.MaxInt-digit)/10` | bound | saturate-never-wrap edge; witness `922337203685477580801.0.0` + neighbours |
| provhost/protocol.go `parseMajor` `major > (math.MaxInt-step)/10` | bound | saturate-never-wrap edge; witness `922337203685477580802.0.0` + neighbours |

The two shapes share the enumeration: the brief's escape clause (derive
each separately with its own arms if they cannot share) was not needed —
one chain classifier places pure-digit chains as char, pure-accumulator
chains as bound, and fails everything mixed. No shape is scoped out in
prose: the hex letter cases carry no digit rune so the enumerator never
sees them (stated in the file, not a carve-out — a future digit-rune
letter check would derive and fail as unclassifiable).

## Control plants — all six redden (harness `plants_round6.py`)

| Plant | Arm attacked | Result |
|---|---|---|
| P1 plain new digit gate, new production file | unregistered site | REDDENED, `unregistered char site …`, only the census test fails |
| P2 same through a var binding (`octet := input[i]`) | unregistered site | REDDENED as `…|plantVarBindingTmp|char|octet < '0' \|\| octet > '9'` — the key follows the structure, not the name `digit` |
| P3 same through an import alias (`import js "encoding/json"`, gate on `js.Number`) | unregistered site | REDDENED — the scanner resolves no imports |
| P4 row with no site (test-file edit, production untouched) | orphan row | REDDENED |
| P5 `input[i] == '5'` | unclassifiable | REDDENED as `unclassifiable guard …` |
| P6 new `*10` accumulator with a guard | unregistered bound | REDDENED as `unregistered bound site …` |

Every plant is applied, the census run as a standalone process (exit 1,
signature asserted, only `TestDigitCensusCoversEveryLeafGuard` fails),
and reverted SHA-256-guarded; after all six the census is green again
and no `zzplant` file remains (`plants-round6.json` attached).

## In-round finding: the hex upper edge had no behavioural pin

The first battery run killed the hex condition edits (D15/D18) through
the census only, and the token-preserving inserts (D16/D19) SURVIVED the
whole suite: the canonical-JSON agreement sweeps never probe `:` inside
an escape. Same class, two more sites — closed in-round, not deferred:

- New `TestDocumentHexEscapeDigitBoundaryRefused` (terminalbackend) and
  `TestProductionEntriesRefuseHexDigitBoundaryEscapes` (provhost) drive
  `\ud80:` / `\udc0:` — malformed on real code (syntax arm), completing
  a surrogate unit under a `:`-admitting mutant (0xD80A/0xDC0A), which
  slides the refusal to the surrogate arm. Valid-pair controls prove the
  entries stay reachable. Both rows are now named in the S2/S5 census
  rows.
- Lower edge (`/`) deliberately has no row: `/` converts to 0xFFFF,
  which no surrogate prefix survives, so a widened lower bound is
  verdict-equivalent on every input — measured, not asserted, by
  R6-D32/R6-D33 (behavioural suite fully green, only the census
  reddens). The scan's only observable channel is the lone-surrogate
  verdict, and the garbage nibble never flips it.

## Round-6 mutation battery — 33 applied / 33 killed, 0 survivors

Harness `mut_round6.py` (attached) with results
`battery-round6-all.json` (attached, 37 rows: 33 production + C01 + X1
+ X2 + mask calibration). Denominator follows the census enumeration:
every one of the 9 sites carries a narrowing and an arm-deletion, every
char site a token-preserving narrowing, every bound site a
threshold-edge narrowing. Masks, run separately per mutant with real
exit codes: census = `-run 'TestDigitCensus'` (2 RUN lines, green
pristine); behavioural = full package suite with `-skip
'TestDigitCensus'` (460 RUN terminalbackend / 671 RUN provhost, green
pristine). Split BEHAVIOURAL-ONLY means census exit 0 with a non-empty
census mask — the split is derived per mutant, not asserted.

Applied 33 / killed 33 / survived 0. Narrowing 16/16 (incl. the F1 edge
R6-D04 and both saturation narrowings R6-D12/R6-D29);
narrowing/token-preserving 8/8, all BEHAVIORAL-ONLY with census exit 0
(R6-D02, D10, D14, D16, D19, D23, D26, D31); arm-deletion 9/9.
CENSUS-ONLY kills (behavioural suite green, census red): R6-D06 (the
`value > 1000` equivalence probe — the post-gate still decides every
exactly-accumulated literal), R6-D32/R6-D33 (lower-edge
verdict-equivalence bounds above), R6-C01 (census-only row edit,
production untouched). Controls: R6-X1 NOT_APPLIED (absent `>=`
anchor), R6-X2 COMPILE_FAIL (`math` unused) — distinct rows, not
counted as applied.

| Mutant | What it narrows the gate to | Named failing test(s) | Split |
|---|---|---|---|
| R6-D01 S1 `digit < '-'` | admits `-`/`.` (both reachable) | TestParseProviderDescriptorValueRefusals (+fraction/negative) | BOTH |
| R6-D02 S1 token-preserving `e`→5 insert | admits `e` as digit 5 | …ValueRefusals/columns_exponent | BEHAVIORAL-ONLY |
| R6-D03 S1 `if false` | digit gate never fires | …ValueRefusals (+7 subrows) | BOTH |
| R6-D04 S6 F1 edge `> 922337203685477580` | admits the 37-digit witness as 1 | …GeometryOverflowVectors | BOTH |
| R6-D05 S6 `> MaxInt` | guard never fires, wraps again | …GeometryOverflowVectors | BOTH |
| R6-D06 S6 `> 1000` equivalence probe | behaviourally equivalent (post-gate decides) | census only | CENSUS-ONLY |
| R6-D07 S6 block deleted | pre-multiply guard gone | …OverflowVectors + refusal-arm bijection | BOTH |
| R6-D08 S7 `> 1001` | admits exactly 1001 | …ValueRefusals/columns_over_bound (+OverflowVectors) | BOTH |
| R6-D09 S7 `< 0` | admits exactly 0 | …ValueRefusals/columns_zero+rows_zero | BOTH |
| R6-D10 S7 token-preserving admit-1001 | admits exactly 1001, text unchanged | …ValueRefusals/columns_over_bound | BEHAVIORAL-ONLY |
| R6-D11 S7 `if false` | range never fires | …ValueRefusals + …OverflowVectors | BOTH |
| R6-D12 S8 `> MaxInt/10` | drops the digit term; `…801` aliases 1 | TestSemverMajorSaturationEdge | BOTH |
| R6-D13 S8 `if false` | saturation never fires | …SaturationEdge + …ProtocolVersionOverflowVectors | BOTH |
| R6-D14 S8 token-preserving force-1 | forces major 1 at the aliasing step | TestSemverMajorSaturationEdge | BEHAVIORAL-ONLY |
| R6-D15 S2 `<= ':'` | admits `:` as nibble 10 | TestDocumentHexEscapeDigitBoundaryRefused | BOTH |
| R6-D16 S2 token-preserving `case digit == ':'` | admits `:`; case text unchanged | TestDocumentHexEscapeDigitBoundaryRefused | BEHAVIORAL-ONLY |
| R6-D17 S2 digit case deleted | digit escapes refused | TestDocumentSurrogateEscapeRefused + sweep + verdicts | BOTH |
| R6-D32 S2 `>= '/'` lower-edge probe | verdict-equivalent (0xFFFF) | census only | CENSUS-ONLY |
| R6-D18 S5 `<= ':'` | admits `:` as nibble 10 | TestProductionEntriesRefuseHexDigitBoundaryEscapes | BOTH |
| R6-D19 S5 token-preserving `b == ':'` | admits `:`; branch text unchanged | TestProductionEntriesRefuseHexDigitBoundaryEscapes | BEHAVIORAL-ONLY |
| R6-D20 S5 `if false` | digit branch never fires | lone-surrogate entries + sweep (13 tests) | BOTH |
| R6-D33 S5 `>= '/'` lower-edge probe | verdict-equivalent (garbage nibble) | census only | CENSUS-ONLY |
| R6-D21 S3 `digit > ':'` | major admits exactly `:` | TestParseMajorDigitBoundariesRefuseAtEntry + inventory pair | BOTH |
| R6-D22 S3 `digit < '/'` | major admits exactly `/` | same | BOTH |
| R6-D23 S3 token-preserving `== ':'` skip | admits `:`; text unchanged | TestParseMajorDigitBoundariesRefuseAtEntry | BEHAVIORAL-ONLY |
| R6-D27 S3 `if false` | major gate never fires | TestDecodeResponseRefusals + inventory + boundaries | BOTH |
| R6-D24 S4 `rest > ':'` | rest admits exactly `:` | boundaries + inventory pair | BOTH |
| R6-D25 S4 `rest < '/'` | rest admits exactly `/` | same | BOTH |
| R6-D26 S4 token-preserving `== '/'` skip | admits `/`; text unchanged | TestParseMajorDigitBoundariesRefuseAtEntry | BEHAVIORAL-ONLY |
| R6-D28 S4 `if false` | rest gate never fires | refusals + inventory + saturation-still-validates | BOTH |
| R6-D29 S9 `> MaxInt/10` | drops the step term; `…802` aliases 2 | TestParseMajorSaturationEdge | BOTH |
| R6-D30 S9 `if false` | saturation never fires | …SaturationEdge + TestParseMajorNeverWraps | BOTH |
| R6-D31 S9 token-preserving force-2 | forces major 2 at the aliasing step | TestParseMajorSaturationEdge | BEHAVIORAL-ONLY |

## AC row coverage: 14 of 15 driven + 1 stated bound (carried)

Rows 1–5 and 7–14 as in outcome-round5 (re-run green in this round's
full suite); row 6 gains the F1 witness, the derived 9-site census, and
the two hex-boundary tests; row 15 is this round's battery (33/33, zero
survivors, classes and controls split as above); row 14 stays the
standing stated bound. No production file changed, so no prior row can
have moved.

## Changes in round 6 (no production file touched)

- `internal/terminalbackend/digit_guard_census_test.go` (new, replaces
  deleted `digit_range_census_test.go`): the derived census + kept
  adjacency test.
- `internal/terminalbackend/descriptor_test.go`: the 37-digit witness on
  both members (+ doc paragraph).
- `internal/terminalbackend/manifest_test.go`: new
  `TestDocumentHexEscapeDigitBoundaryRefused`.
- `internal/provhost/surrogate_test.go`: new
  `TestProductionEntriesRefuseHexDigitBoundaryEscapes`.
- Census S2/S5 rows name the new boundary tests.
- `LOGBOOK.md`: round-6 entry (this work).

## Gates re-run by this round (nothing accepted from prior rounds)

| Gate | Result |
|---|---|
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `GOOS=windows go vet` (three touched packages) | exit 0 |
| `GOOS=windows go build ./...` | exit 0 |
| `gofmt -l internal/` | clean |
| `go test ./... -count=1` | exit 0, 22 packages ok, 0 FAIL |
| `go test ./internal/terminalbackend/ ./internal/provhost/ ./internal/provider/ -cover -count=1` | exit 0 — 94.3% / 86.0% / 97.8% |
| `go test -race ./internal/terminalbackend/ ./internal/provhost/ -count=1` | exit 0 (both) |
| `go run ./internal/traceability/cmd/tracecheck` | ok — contracts=60, normative_sections=36, acceptance_cases=98 |
| terminalbackend + provhost refusal-arm bijections | unchanged rows, suite green (no new production arm) |

## Stated bounds carried or added

- N4/N6/N7 carried unchanged (no production touched; deferral item
  still names no follow-up board item — orchestrator freeze).
- N9 (`+` equivalent by reachability) carried in the S1 row.
- New: hex lower edge (`/`) is verdict-equivalent (measured R6-D32/D33);
  non-digit rune guards and hex letter cases are outside the
  enumerator's domain (a digit-rune novelty still fails as
  unclassifiable); an accumulator with no guard has no bound to row
  (missing-guard class stays with the overflow-vector tests); no
  accumulator shadowing exists (a shadowed comparison would misresolve).
