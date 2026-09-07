# TASK-260906-vmzk0y — own-provider-protocol-v3-envelope-and-error-lift (round 5)

Status: ready for review (handed off to review; board status `to-review`).

Candidate tree: `d6253f58fe72a84c0900329c686300f232374874`
(verified by `git ls-tree -r`: all six new files
`internal/provider/lift.go`, `internal/provider/lift_test.go`,
`internal/terminalbackend/descriptor.go`,
`internal/terminalbackend/descriptor_test.go`,
`internal/terminalbackend/semver_major_edge_test.go`,
`internal/terminalbackend/digit_range_census_test.go` are in-tree;
`git hash-object` of the worktree `semver_major_edge_test.go`,
`digit_range_census_test.go`, `descriptor_test.go`, and `LOGBOOK.md`
byte-for-byte equals the tree blobs. Built via a detached index —
`GIT_INDEX_FILE` copy + `read-tree HEAD` + `add -A` + `write-tree` — so
the worktree index is untouched.)

Round 5 answers review-verdict-rev4: two blocking findings (F1
`semverMajor` saturation guard pinned only far from its edge, F2
`descriptorGeometry` digit gate with no witness in the upper half of its
rejected class — third consecutive round of one class, both sites inside
this leaf's delta) plus the standing census item. Everything else stands
exactly as the reviewer verified it in rev4 — G-A (derived
behavioural-versus-census split), G-B (provhost saturation edge and
`/`/`:` digit boundaries), §7.A ownership, v3 envelope both directions,
the lift seam, N6 — and is not re-litigated here. No production change
in round 5: the `semverMajor` guard and the `descriptorGeometry` gate
were already correct; the delta is two new test files, two `1E2` rows in
the value-refusals table, and the logbook entry.

## F1 (blocking, closed) — semverMajor saturation guard pinned at its edge

The guard `major > (math.MaxInt-digit)/10`
(`internal/terminalbackend/terminalbackend.go:258`) was witnessed only
by literals that saturate under the weakened guard too
(`18446744073709551617` → MaxInt, `9223372036854775808` → −2^63, both
still `!= 1`, still refused). The true narrowing
`major > math.MaxInt/10` (S1, drops the `digit` term — the exact M3/F1a
shape) survived the whole terminalbackend suite and admits
`922337203685477580801.0.0` at `ParseProviderDescriptor` with `err =
nil` (semverMajor returns 1, aliasing native major 1). The aliasing
reaches every `semverMajor(...) != 1` site: `New`/`RegisterExternal`
(`terminalbackend.go:235`), `CheckVersionTuple` (`:556`),
`manifest.go:1072/1155/1301`, `descriptor.go:168`.

New `TestSemverMajorSaturationEdge` (internal, mirroring
`TestParseMajorSaturationEdge`) drives five rows — largest exact fit
`9223372036854775807.0.0`, first saturated `9223372036854775808.0.0`,
and the aliasing witness `922337203685477580801.0.0` with its ±1
neighbours `...800`/`...802` — each requiring semverMajor == MaxInt
(not merely refusal: the neighbours refuse under S1 too, so the value
pin carries those kills) AND refusal with the named arm at the
production entries: `descriptor protocol version` at
`ParseProviderDescriptor`, `protocol_version major 1` at
`CheckVersionTuple` (list carries the version itself, so only the major
arm can refuse), and the witness at `New` (`protocol_versions major
1`). Controls pin `1.0.0 → 1`, `2.0.0 → 2`, and the old-corpus witness
still saturating (documenting the blind region it lived in).

| `protocol_version` | shipped | under S1 (`> MaxInt/10`) |
|---|---|---|
| `9223372036854775807.0.0` (MaxInt, largest exact fit) | (MaxInt, refuse) | identical (exact fit, no saturation) |
| `9223372036854775808.0.0` (MaxInt+1, first saturated) | (MaxInt, refuse) | (−2^63, refuse — exit same, value kills) |
| `922337203685477580800.0.0` (neighbour, wraps to 0) | (MaxInt, refuse) | (0, refuse — value kills) |
| `922337203685477580801.0.0` (aliasing witness, wraps to 1) | (MaxInt, refuse) | (1, ADMITTED err=nil — entry kills) |
| `922337203685477580802.0.0` (neighbour, wraps to 2) | (MaxInt, refuse) | (2, refuse — value kills) |

Witness construction: `922337203685477580` walks the accumulator to
exactly MaxInt/10; the next `8` overflows int64 to −2^63; the following
`0` multiplies by 10 (0 mod 2^64), resetting to 0; the final digit lands
on 0/1/2. Measured: S1 is KILLED by `TestSemverMajorSaturationEdge`
alone (`semverMajor("9223372036854775808.0.0") = -9223372036854775808,
want saturation to math.MaxInt`), with zero census kill (census mask
exit 0, 2 RUN / 2 PASS, verified non-empty).

## F2 (blocking, closed) — geometry digit gate witnessed at its true edge

The gate `digit < '0' || digit > '9'`
(`internal/terminalbackend/descriptor.go:235`) was witnessed at `e`
(0x65) at best — 44 bytes above the true edge `9` (0x39). Every mutant
upper bound X with `E <= X < e` admits uppercase `E` and survived. Under
D2 (`digit > 'E'`), `columns: 1E2` — a valid JSON number naming 100 —
is admitted as Columns = 312 (`1`, then `E` valued at 0x45−0x30 = 21,
then `2`; 1·10+21 = 31, 31·10+2 = 312): a fabricated in-range value.

Two `1E2` rows added to `TestParseProviderDescriptorValueRefusals`
(columns and rows, asserting `descriptor geometry digits`); the rows row
proves the fix is at the shared gate, not at one call. Measured: D2 and
the token-preserving D1 (`if digit == 'E' { continue }`, condition text
byte-identical) are both KILLED by the new rows alone, with zero census
kill (census mask exit 0, 2 RUN / 2 PASS, verified non-empty).

## Item 3 — census of all three digit-range comparisons (recorded)

`internal/terminalbackend/digit_range_census_test.go` carries the file
header census; each row below names the committed witness that makes
"covered" checkable instead of asserted:

| Site | Condition | Reachable rejected class (domain) | Adjacent witnesses | Declared equivalents |
|---|---|---|---|---|
| provhost parseMajor major arm (`protocol.go:379`) | `digit < '0' \|\| digit > '9'` | every non-digit byte (JSON string) | `/` (0x2F), `:` (0x3A) in `TestParseMajorDigitBoundariesRefuseAtEntry` | letters/signs in the older corpus |
| provhost parseMajor rest arm (`protocol.go:397`) | `rest[i] < '0' \|\| rest[i] > '9'` | same | `/`/`:` in minor AND patch positions, same test | rest letters (control rows) |
| terminalbackend descriptorGeometry (`descriptor.go:235`) | `digit < '0' \|\| digit > '9'` | exactly `{-, +, ., e, E}` (json.Number) | `-` (`-80`/`-24`), `.` (`80.0`/`24.5`), `e` (`1e2`), `E` (`1E2`, this round) on both members | `+`: one line, not a row (N9) — only legal after e/E, which refuses first; `/`/`:` provably unreachable (see below) |

`TestDescriptorGeometryAdjacentsNeverReachTheGate` proves the `/`-`:`
unreachability claim executably: `1/2` and `1:2` on both members are
refused with the document syntax mismatch arm before the gate runs — the
inverse of the provhost rows, which is why the same adjacent characters
need opposite tests at the two domains. Deliberately out of scope (one
line in the header, not an oversight): the hex three-way splits in
`manifest.go`/`surrogate.go` predate this leaf and are a different
shape; the two saturation guards are accumulation bounds pinned by the
edge tests, not by adjacent characters.

## Round-5 mutation battery (F1/F2 gates + re-pinned census sites)

Denominator re-derived from production to cover the gates under review
rather than provhost alone: the three digit-range comparisons above
plus both saturation guards. Harness
`TASK-260906-vmzk0y_mut-round5.py` (spec
`TASK-260906-vmzk0y_spec-round5.json`, results
`TASK-260906-vmzk0y_battery-round5.json`, all attached); mutants applied
to a scratch copy at `/tmp/vmzk0y-rev5-mut`, never to the candidate
worktree (`ALL RESTORED`, SHA-256 asserted per file; worktree `git
status` confirms only the intended delta). Every `go test` ran as a
standalone process with its real exit code preserved (no pipe chain).
Killer split: BEHAVIOURAL = drives a production entry point;
CENSUS = source-text inventory only
(`TestDerivedRefusalArmsAreAllWitnessed` /
`TestWitnessedArmsAreAllDerived` in provhost,
`TestDerivedRefusalArmsAreAllDeclared` /
`TestDeclaredRefusalArmsAreAllDerived` in terminalbackend). Every mask
verified non-empty (census 2 RUN, behavioural ≥1 RUN).

Applied 9 / killed 8 / survived 1 (declared equivalent, bound below).
Controls NOT_APPLIED and COMPILE_FAIL are distinct rows, not counted as
applied.

| Mutant | What it narrows the gate to | Named test(s) that fail | Killer |
|---|---|---|---|
| P-M3 narrowing (provhost saturation): `> (MaxInt-step)/10` → `> MaxInt/10` | threshold off by one; `...802` aliases native 2 | `TestParseMajorSaturationEdge` | BEHAVIOURAL (census exit 0) |
| P-M6 narrowing (major digit arm): `digit > ':'` | major admits exactly `:` | `TestParseMajorDigitBoundariesRefuseAtEntry` + census pair | BOTH |
| P-M17 narrowing/token-preserving: insert `if digit == ':' { continue }` | major admits exactly `:`; census byte-identical | `TestParseMajorDigitBoundariesRefuseAtEntry` | BEHAVIOURAL ONLY (census exit 0 — the point) |
| T-S1 TRUE NARROWING (semverMajor saturation): `> (MaxInt-digit)/10` → `> MaxInt/10` | threshold off by one; `...801` aliases native 1 and is admitted | `TestSemverMajorSaturationEdge` | BEHAVIOURAL (census exit 0) |
| T-S2 delete-shaped (saturation): guard → `> math.MaxInt` (never fires) | accumulation wraps again; proves the gate exists, not its edge | `TestParseProviderDescriptorProtocolVersionOverflowVectors`, `TestSemverMajorSaturationEdge` | BEHAVIOURAL (census exit 0) |
| T-D1 narrowing/token-preserving: insert `if digit == 'E' { continue }` | geometry admits exactly `E`; census byte-identical | `TestParseProviderDescriptorValueRefusals` (+`columns_exponent_uppercase`, +`rows_exponent_uppercase`) | BEHAVIOURAL ONLY (census exit 0 — the point) |
| T-D2 narrowing (geometry bound shift): `digit > 'E'` | geometry admits `:`–`E`; `1E2` admitted as 312 | same rows | BEHAVIOURAL (census exit 0) |
| T-D3 narrowing/token-preserving: insert `if digit == '+' { continue }` | no reachable disposition changes | SURVIVES — declared equivalent, bound below | survivor bound stated |
| T-D4 arm-deletion: geometry digit-gate block deleted | non-digits fall to the bound arm or accumulate; digits arm gone | `TestParseProviderDescriptorValueRefusals` (+7 subrows) + census pair | BOTH (census co-fires by construction — the arm is removed) |

Class split: narrowing 4/4 killed (incl. the true semverMajor narrowing
T-S1); narrowing/token-preserving 2/3 killed-or-equivalent (P-M17, T-D1
killed behaviourally with zero census kill; T-D3 equivalent);
delete-shaped 1/1 killed; arm-deletion 1/1 killed; equivalence 0/1
(declared equivalent).

Distinct control rows, NOT counted as applied mutants:

| Control | Disposition |
|---|---|
| X1: anchor `>= math.MaxInt` absent from source (0 sites) | NOT_APPLIED |
| X2: saturation block deleted outright (`math` import unused) | COMPILE_FAIL |

Survival bound: T-D3 survives because `+` is only legal in a
json.Number after `e`/`E`, which the unchanged gate refuses first, so
the mutant changes no disposition on any reachable literal (full suite
461/461 green under it, behavioural mask 28/28 green). A lower-bound
narrowing admitting `+` admits `-` too (0x2B < 0x2D), which the `-80`
row kills — so `+` adds no narrowing power and gets the declared line,
not a row (N9). Recorded in the census header so it is not mistaken for
a hole later.

Per-arm verdict: every gate under review now carries at least one
BEHAVIOURAL killer. The two token-preserving narrowings that the
inventory is blind to by construction (P-M17, T-D1) carry a behavioural
killer and explicitly no census kill.

## Notes N4/N6/N7

- N4 (carried): the `foreignMajor` peek widening placement outside this
  delta was accepted in round 3; this round touches no production file
  at all (`git diff HEAD --name-only` shows only prior-round paths plus
  the two new test files and the `descriptor_test.go`/`LOGBOOK.md`
  edits), so the placement still holds by construction.
- N6 (carried): the N2 CallExpr-only bound sits on
  `TestNoProductionPathAttestsProviderIdentityBinding`'s doc comment;
  untouched this round.
- N7 (carried): the §7.A deferral still names no follow-up board item —
  per the round-4 brief this is the orchestrator's board-element freeze,
  not a producer omission.

## AC row coverage: 14 of 15 driven + 1 stated bound (carried)

Rows 1–5 and 7–13 were driven through the production entry point by the
named test in rounds 1–3 and re-run green in this round's full suite
(exit 0, 22 pkgs, 0 FAIL); rows 6 and 15 gain the round-5 rows below;
row 14 stays a stated bound.

| # | AC / DoD row | Production call site | Driving test / bound |
|---|---|---|---|
| 1 | v3 envelope has exactly one named owner, recorded in provhost stated bounds and the ownership registry | `internal/provhost/doc.go`, `ownership.v0.5.0.json` `section:7.A` | `tracecheck`: ok (60 contracts, 36 sections, 98 cases) |
| 2 | working v3 admission or explicit stated bound naming the deferral | `terminalbackend.AdmitProviderDescriptor`; `provhost/doc.go` | `TestAdmitProviderDescriptorAdmitsMatchingBinding`, `TestAdmitProviderDescriptorRefusesMismatch` |
| 3 | v3 envelope reaching provhost → documented outcome, both directions | `provhost.DecodeResponse`; `terminalbackend.AdmitProviderDescriptor` | `TestDecodeResponseWellFormedV3SuccessIsMismatch`, `TestDecodeResponseV3FailureWithValid130ErrorIsMismatch` |
| 4 | Error 1.3.0 binding implemented-or-deferred; no silent major-2 binding for v3 | `provhost.DecodeResponse` (observed-major contract), `axerror.BindingFor`/`DecodeBound` | `TestDecodeResponseV2FailureWith130ErrorIsBoundTo100`; the v3 test decodes its embedded error through production `(provider, 3)` → 1.3.0 |
| 5 | F1 (rev1): geometry bound decided on digits, overflow refused | `terminalbackend.descriptorGeometry` | `TestParseProviderDescriptorGeometryOverflowVectors` |
| 6 | majors never wrap; saturation validates the full shape; guard edges, digit edges, and the three-site census pinned | `terminalbackend.semverMajor`, `provhost.parseMajor` + `DecodeResponse`, `terminalbackend.descriptorGeometry` | prior rows (round 4) + NEW `TestSemverMajorSaturationEdge` (T-S1/T-S2), NEW `1E2` value-refusal rows (T-D1/T-D2), NEW `TestDescriptorGeometryAdjacentsNeverReachTheGate` + census header (item 3) |
| 7 | F4: registered-but-unlifted code produces no wire object | `provider.Lift` | `TestLiftRefusesRegisteredCodeWithoutLiftArm`, `TestLiftRefusesUnregisteredCode` |
| 8 | discovery failure lifts into an axerror Structured Error through an accepted production path | `provider.Lift` | `TestLiftCarriesDiscoveryFailuresToTheWire` |
| 9 | causal-leak shape refused (negative) | `axerror.New` | `TestNaiveLiftShapeIsRefusedByTheWireGate` |
| 10 | no machine-local path, provider ID, or owner identity on any wire message | `provider.Lift`, `terminalbackend` parse arms | wire-absence asserts in all 4 lift vectors; `TestParseProviderDescriptorRefusesWithoutEchoingLocalData` |
| 11 | CheckProviderDescriptor completed; registry owns 7.A | `terminalbackend.ParseProviderDescriptor` + `AdmitProviderDescriptor`; registry | inventory bijection (terminalbackend) + rows; `tracecheck` ok |
| 12 | F3: bound rests on true facts, both halves pinned | `provhost/identity.go` header | `TestSpecIdentityExampleVerifiesAgainstItsClaimedDigest`, `TestNoProductionPathAttestsProviderIdentityBinding` |
| 13 | F5: census derives the whole package, every form, fail-closed | `loadProviderCodes` | `TestLiftCoversTheClosedCodeSet` + plants |
| 14 | F6: future caller recorded | `terminalbackend/descriptor.go` | STATED BOUND (prose by nature; names path, order, two miswirings) |
| 15 | mutation battery with narrowing / token-preserving / arm-deletion / census-only split | `/tmp` harness (attached) | this round: 9 applied / 8 killed / 1 survived-equivalent + NOT_APPLIED + COMPILE_FAIL distinct; prior batteries stand |

## Production changes in round 5 (none)

- `internal/terminalbackend/semver_major_edge_test.go` (new):
  `TestSemverMajorSaturationEdge`.
- `internal/terminalbackend/digit_range_census_test.go` (new): three-site
  census record + `TestDescriptorGeometryAdjacentsNeverReachTheGate`.
- `internal/terminalbackend/descriptor_test.go`: two `1E2` rows.
- `LOGBOOK.md`: entry 2254 (this round).
- All other paths byte-identical to rev4; leaf-1 packages untouched.

## Gates re-run by this round (nothing accepted from prior rounds)

| Gate | Result |
|---|---|
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `gofmt -l internal/` | clean |
| `go test ./... -count=1` | exit 0, 22 packages ok, 0 FAIL |
| `go test ./internal/terminalbackend/ ./internal/provhost/ ./internal/provider/ -cover -count=1` | exit 0 — 94.2% / 85.8% / 97.8% |
| `go test -race ./internal/terminalbackend/ ./internal/provhost/ -count=1` | exit 0 (both) |
| `GOOS=windows go build ./...` | exit 0 |
| `GOOS=windows go vet` (three touched packages) | exit 0 |
| `go run ./internal/traceability/cmd/tracecheck` | ok — contracts=60, normative_sections=36, acceptance_cases=98 |
| terminalbackend refusal-arm bijection | unchanged rows, suite green (no new production arm) |
