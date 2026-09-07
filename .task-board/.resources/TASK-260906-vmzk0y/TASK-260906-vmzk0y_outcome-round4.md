# TASK-260906-vmzk0y — own-provider-protocol-v3-envelope-and-error-lift (round 4)

Status: ready for review (handed off to review; board status `to-review`).

Candidate tree: `6c8e29de0765747dc93c8565a756a888c6609ca6`
(verified by `git ls-tree -r`: all four new files
`internal/provider/lift.go`, `internal/provider/lift_test.go`,
`internal/terminalbackend/descriptor.go`,
`internal/terminalbackend/descriptor_test.go` are in-tree; `git hash-object`
of the worktree `protocol.go`, `protocol_test.go`, `identity_test.go`, and
`LOGBOOK.md` byte-for-byte equals the tree blobs. Built via a detached index —
`GIT_INDEX_FILE` copy + `read-tree HEAD` + `add -A` + `write-tree` — so the
worktree index is untouched.)

Round 4 answers review-verdict-rev3: one blocking finding (F1a saturation-edge
+ F1b digit-range token-preserving, second consecutive round of the same
class) plus notes N4–N7. Everything else stands exactly as the reviewer
verified it — G-A (malformed giants pinned apart, migration reddens, peek
divergence gone on bare and full frames), G-B (round-2 sentence withdrawn as
classification evidence), G-C (N1/N3 closed), G-D battery denominator, F1/F2/F4/F5
classes, F3 bound, §7.A ownership, Lift fail-closed arms, (provider,3)→1.3.0
binding — and is not re-litigated here. No production change in round 4: the
`parseMajor` guard was already correct; the delta is three behavioural test
groups plus the N6 doc sentence.

## F1a (blocking, closed) — saturation guard pinned at its edge

The guard `major > (math.MaxInt-step)/10` was witnessed only far above its
edge (2^64+1, 2^64+2, 2^64, 41 nines), where saturation is sticky. The true
narrowing `major > math.MaxInt/10` (drops the `step` term) differs only when
the accumulator sits at exactly MaxInt/10 (922337203685477580) and the next
digit is 8 or 9. New `TestParseMajorSaturationEdge` drives five rows through
the production entry point `DecodeResponse`, each requiring parseMajor ==
MaxInt (not merely recognized), major != 2, and
`incompatible_protocol` / exit 6 with `observed` echoed:

| `protocol_version` | shipped | under M3 (`> MaxInt/10`) |
|---|---|---|
| `9223372036854775807.0.0` (MaxInt, largest admitted) | (MaxInt,true) → 6 | identical (no saturation on the exact fit) |
| `9223372036854775808.0.0` (MaxInt+1, first saturated) | (MaxInt,true) → 6 | (-9223372036854775808,true) → 6 (exit same, parse value kills) |
| `922337203685477580801.0.0` (aliasing neighbour, wraps to 1) | (MaxInt,true) → 6 | (1,true) → 6 (exit same, parse value kills) |
| `922337203685477580802.0.0` (aliasing witness, wraps to 2) | (MaxInt,true) → 6 | (2,true) → `provider_protocol_error` exit 13 |
| `922337203685477580803.0.0` (aliasing neighbour, wraps to 3) | (MaxInt,true) → 6 | (3,true) → 6 (exit same, parse value kills) |

The witness construction: `922337203685477580` walks the accumulator to
exactly MaxInt/10; the next `8` overflows int64 to -2^63; the following `0`
multiplies by 10 (0 mod 2^64), resetting to 0; the final `2` lands on 2 — a
foreign major aliasing native major 2, verbatim the invariant the `parseMajor`
doc comment asserts. The ±1 neighbours pin the fix at the guard, not a
literal blocklist of one string. Measured: M3 is KILLED by
`TestParseMajorSaturationEdge` alone
(`parseMajor("9223372036854775808.0.0") = -9223372036854775808, want
saturation to math.MaxInt`); `go test ./internal/provhost/ -count=1` passes on
shipped code.

## F1b (blocking, closed) — digit-range arms pinned behaviourally at `/` and `:`

The major arm computes `digit < '0' || digit > '9'` and the rest arm computes
`rest[i] < '0' || rest[i] > '9'`. The corpus used `a`, `2a`, `-1`, `+3`, `b`,
`c` — never the two characters adjacent to the range: `/` (0x2F, one below
`'0'`) and `:` (0x3A, one above `'9'`). New
`TestParseMajorDigitBoundariesRefuseAtEntry` drives eight rows through
`DecodeResponse`, each requiring parseMajor unrecognized AND
`provider_protocol_error` / exit 13 with member `protocol_version` /
`unsupported protocol version`:

`3/.0.0`, `/.0.0` (`/` in major) · `3:.0.0`, `:.0.0` (`:` in major) ·
`3./.0`, `3.0./` (`/` in minor, patch) · `3.:.0`, `3.0.:` (`:` in minor, patch)

— so `/` and `:` are covered in the major and in each rest position, plus the
leading single-character majors, verifying one step away from any single
finding vector. Measured effect of the token-preserving inserts (condition
text unchanged, derived arm set byte-identical, bijection stays 167/167):

| mutant (production edit, condition text preserved) | shipped | under mutant |
|---|---|---|
| M17 `if digit == ':' { continue }` above the major check | (0,false) → 13 | (3,true) → 6 |
| M17b `if digit == '/' { continue }` above the major check | (0,false) → 13 | (3,true) → 6 |
| M18 `if rest[i] == '/' { continue }` above the rest check | (0,false) → 13 | (3,true) → 6 |
| M18b `if rest[i] == ':' { continue }` above the rest check | (0,false) → 13 | (3,true) → 6 |

All four are KILLED by `TestParseMajorDigitBoundariesRefuseAtEntry` with zero
census kill (the census cannot see them by construction). The same rows give
the condition-editing narrowings their first behavioural killers: M5
(`rest[i] < '/'`), M5b (`rest[i] > ':'`), M6 (`digit > ':'`), M6b
(`digit < '/'`) are now killed by the new behavioural test IN ADDITION to the
inventory bijection (previously census-only).

## Round-4 mutation battery (F1 gate)

Denominator re-derived for this revision from production: the nine
version-classification obligations reachable from `DecodeResponse` — P1
`len(parts) != 3`, P2 major-digit range, P3 saturation guard, P4
`len(parts[0]) == 0`, P5 `len(rest) == 0`, P6 rest-digit range, C1
failure-error contract major selection, K1 `foreignMajor` peek predicate, V1
version-gate mismatch predicate. Battery script
`.temp/TASK-260906-vmzk0y/mut_round4.py` (evidence
`.temp/TASK-260906-vmzk0y/battery-round4-all.json`, per-subset files
`battery-round4-*.json`); every `go test ./internal/provhost/ -count=1` ran as
a standalone process with its real exit code preserved (no pipe chain), and
each target file was restored byte-identically afterwards (SHA-256 asserted,
`ALL RESTORED`).

Applied 22 / killed 21 / survived 1 (declared equivalent, see bound below).
Controls NOT_APPLIED and COMPILE_FAIL are distinct rows, not counted as
applied. Killer split: BEHAVIOURAL = drives a production entry point
(`DecodeResponse`, `EncodeRequest`, `Host.Call`, `DecodeBound` wrappers);
CENSUS = source-text inventory only
(`TestDerivedRefusalArmsAreAllWitnessed`, `TestWitnessedArmsAreAllDerived`).
`TestEveryArmWitnessRefusesAtTheProductionEntry` drives the entry, so it
counts as behavioural.

| Mutant | What it narrows the gate to | Named test(s) that fail | Behavioural killer? |
|---|---|---|---|
| M1 narrowing (P3): saturation `continue` back to early `return MaxInt, true` | giant-with-malformed-rest reads recognized (exit 6, not 13) | `TestParseMajorSaturationStillValidatesTheRest`, `TestDecodeResponseUnrecognizedVersionRunsMemberRules` | BEHAVIOURAL (no census) |
| M2 narrowing (P3): guard neutered to `major > math.MaxInt` (never fires) | accumulation wraps again | `TestParseMajorNeverWraps`, `TestParseMajorSaturationEdge` | BEHAVIOURAL (no census) |
| M3 TRUE NARROWING (P3): `major > (MaxInt-step)/10` → `major > MaxInt/10` | threshold off by one; `...802` aliases native 2 | `TestParseMajorSaturationEdge` | BEHAVIOURAL (no census) |
| M4 narrowing (P5): `len(rest) == 0` → `len(rest) < 0` (never fires) | empty rest admitted | `TestDecodeResponseRefusals` + subrow, `TestEveryArmWitness.../empty_minor`, `TestParseMajorSaturationStillValidatesTheRest` + census pair | BOTH |
| M5 narrowing (P6 condition-edit): `rest[i] < '/'` | rest admits exactly `/` | `TestParseMajorDigitBoundariesRefuseAtEntry` (new) + census pair | BOTH (was census-only) |
| M5b narrowing (P6 condition-edit mirror): `rest[i] > ':'` | rest admits exactly `:` | same as M5 | BOTH |
| M6 narrowing (P2 condition-edit): `digit > ':'` | major admits exactly `:` | `TestParseMajorDigitBoundariesRefuseAtEntry` (new) + census pair | BOTH (was census-only) |
| M6b narrowing (P2 condition-edit mirror): `digit < '/'` | major admits exactly `/` | same as M6 | BOTH |
| M7 narrowing (P1): `len(parts) != 3` → `len(parts) < 3` | over-long shape (4+ parts) admitted | `TestDecodeResponseRefusals` + subrow + census pair | BOTH |
| M8 narrowing (P4): `len(parts[0]) == 0` → `< 0` (never fires) | empty major admitted | `TestDecodeResponseRefusals` + subrow, `TestEveryArmWitness.../empty_major` + census pair | BOTH |
| M11 narrowing (C1): contract `observedMajor` → `observedMajor+1` | failure-error contract off the observed major | `TestDecodeResponseV2FailureWith130ErrorIsBoundTo100`, `TestDecodeResponseReturnsChildFailure`, `TestChildFailureCodesPassThrough*` | BEHAVIOURAL (no census) |
| M13b narrowing (K1): peek `recognized && ...` → `recognized \|\| ...` (widen) | unrecognized versions skip member rules (N4) | `TestDecodeResponseUnrecognizedVersionRunsMemberRules` (new) | BEHAVIOURAL (no census; was SURVIVED) |
| M14b narrowing (K1): peek `!= ProtocolMajor` → `> ProtocolMajor` | major 1 no longer a mismatch | `TestDecodeResponseForeignMajorPrecedesMemberRules` + subrow | BEHAVIOURAL (no census) |
| M16 narrowing (V1): gate `!= ProtocolMajor` → `> ProtocolMajor` | major 1 falls to unsupported-version | `TestDecodeResponseForeignMajorPrecedesMemberRules` + subrow, `TestDecodeResponseRecognizableMajorMismatch` | BEHAVIOURAL (no census) |
| M17 narrowing/token-preserving (P2): insert `if digit == ':' { continue }` | major admits exactly `:`; census byte-identical | `TestParseMajorDigitBoundariesRefuseAtEntry` | BEHAVIOURAL ONLY (zero census kill — the point) |
| M17b narrowing/token-preserving (P2): insert `if digit == '/' { continue }` | major admits exactly `/`; census byte-identical | same | BEHAVIOURAL ONLY |
| M18 narrowing/token-preserving (P6): insert `if rest[i] == '/' { continue }` | rest admits exactly `/`; census byte-identical | same | BEHAVIOURAL ONLY |
| M18b narrowing/token-preserving (P6): insert `if rest[i] == ':' { continue }` | rest admits exactly `:`; census byte-identical | same | BEHAVIOURAL ONLY |
| M9 arm-deletion (P5+P6): whole minor/patch validation deleted | every malformed rest admitted | `TestDecodeResponseRefusals` + subrows, `TestEveryArmWitness...` + new boundary test + census pair | BOTH |
| M10 arm-deletion (P1): three-part shape check deleted | 2-part and 4-part admitted | `TestDecodeResponseRefusals` + subrows, `TestEveryArmWitness...` | BOTH (census pair co-fires via TestWitnessedArms... — counted in full list) |
| M15 census-only: census derives zero parse arms (production untouched) | inventory must orphan declared parse witnesses | `TestDerivedRefusalArmsAreAllWitnessed`, `TestWitnessedArmsAreAllDerived` | CENSUS ONLY (mutates the census itself) |
| M12 equivalence-probe (C1): contract back to hardcoded `Major: 2` | no observable change (only 2.0.0 reaches the line) | SURVIVES — declared equivalent, bound N5 | survivor bound stated below |

Class split: narrowing 14/14 killed (incl. the true P3 narrowing M3);
narrowing/token-preserving 4/4 killed, all behaviourally with zero census
kill; arm-deletion 2/2 killed; census-only 1/1 killed; equivalence-probe 0/1
(declared equivalent). No surviving mutant except M12.

Distinct control rows, NOT counted as applied mutants:

| Control | Disposition |
|---|---|
| X1: anchor with `>=` absent from source (0 sites) | NOT_APPLIED |
| X2: guard deleted outright (`math` import unused) | COMPILE_FAIL |

Survival bound: M12 survives because only `2.0.0` reaches the failure-error
contract line today (the version gate refuses every other version above), so
`Major: observedMajor` and `Major: 2` coincide on every reachable path; the
`protocol.go` comment states exactly this, and shifting the major OFF the
observed value (M11) is killed by eight tests. Recorded per N5 so it is not
mistaken for a hole later.

Per-arm behavioural-vs-census verdict: every production arm except the
census-itself probe M15 now carries at least one BEHAVIOURAL killer. Arms
whose only killers were the census pair in rev3 (P2/P6 condition-edits M5/M6)
now carry the new behavioural test alongside the census pair; the four
token-preserving narrowings carry a behavioural killer and explicitly no
census kill, which is what proves the inventory is blind to that shape.

## Notes N4–N7

- N4 (peek widening, FIXED not merely confirmed): M13b survived rev3 with a
  member-detail slide (`ok` → `protocol_version`) on bare frames, code and
  exit unchanged. New `TestDecodeResponseUnrecognizedVersionRunsMemberRules`
  pins the narrowing direction behaviourally (bare frame with `2.abc.def` and
  `99999999999999999999.abc.def` must report member `ok` / `missing member`),
  so M13b is now KILLED by that test alone. The narrowing direction was
  already killed (M14b); both directions are now behaviourally pinned.
- N5: M12 is the declared equivalent above, not a hole; M11 kills the
  off-observed shift.
- N6: the N2 CallExpr-only bound now lives on the test's doc comment
  (`TestNoProductionPathAttestsProviderIdentityBinding` carries the `f :=
  pkg.VerifyObjectIdentity; f(x)` sentence), not only in the outcome.
- N7: carried since rev2 and accepted there — the §7.A deferral loop names no
  follow-up board item; per the round-4 brief that attachment is the
  orchestrator's board-element freeze, not this leaf's.

## AC row coverage: 14 of 15 driven + 1 stated bound (N1, carried)

Rows 1–13 and 15 were driven through the production entry point by the named
test in rounds 1–3 and re-run green in this round's full suite; row 14 (F6,
future caller recorded) is prose by nature and stays a stated bound. Round 4
adds the F1a/F1b/N4 behavioural rows to row 6 and the N6 sentence to row 12;
no other row changes.

| # | AC / DoD row | Production call site | Driving test / bound |
|---|---|---|---|
| 1 | v3 envelope has exactly one named owner, recorded in provhost stated bounds and the ownership registry | `internal/provhost/doc.go`, `ownership.v0.5.0.json` `section:7.A` | `tracecheck`: ok (60 contracts, 98 cases) |
| 2 | working v3 admission or explicit stated bound naming the deferral | `terminalbackend.AdmitProviderDescriptor`; `provhost/doc.go` | `TestAdmitProviderDescriptorAdmitsMatchingBinding`, `TestAdmitProviderDescriptorRefusesMismatch` |
| 3 | v3 envelope reaching provhost → documented outcome, both directions | `provhost.DecodeResponse`; `terminalbackend.AdmitProviderDescriptor` | `TestDecodeResponseWellFormedV3SuccessIsMismatch`, `TestDecodeResponseV3FailureWithValid130ErrorIsMismatch` |
| 4 | Error 1.3.0 binding implemented-or-deferred; no silent major-2 binding for v3 | `provhost.DecodeResponse` (observed-major contract), `axerror.BindingFor`/`DecodeBound` | `TestDecodeResponseV2FailureWith130ErrorIsBoundTo100`; the v3 test decodes its embedded error through production `(provider, 3)` → 1.3.0 |
| 5 | F1: geometry bound decided on digits, overflow refused | `terminalbackend.descriptorGeometry` | `TestParseProviderDescriptorGeometryOverflowVectors` |
| 6 | F2+G1+F1a+F1b+N4: majors never wrap; saturation validates the full shape; guard edge and digit edges pinned; peek directions pinned | `terminalbackend.semverMajor`, `provhost.parseMajor` + `DecodeResponse` | `TestParseProviderDescriptorProtocolVersionOverflowVectors`, `TestParseMajorNeverWraps` (exit-6), `TestParseMajorSaturationStillValidatesTheRest` (exit-13), NEW `TestParseMajorSaturationEdge` (M3), NEW `TestParseMajorDigitBoundariesRefuseAtEntry` (M5/M6/M17/M18 + mirrors), NEW `TestDecodeResponseUnrecognizedVersionRunsMemberRules` (M13b) |
| 7 | F4: registered-but-unlifted code produces no wire object | `provider.Lift` | `TestLiftRefusesRegisteredCodeWithoutLiftArm`, `TestLiftRefusesUnregisteredCode` |
| 8 | discovery failure lifts into an axerror Structured Error through an accepted production path | `provider.Lift` | `TestLiftCarriesDiscoveryFailuresToTheWire` |
| 9 | causal-leak shape refused (negative) | `axerror.New` | `TestNaiveLiftShapeIsRefusedByTheWireGate` |
| 10 | no machine-local path, provider ID, or owner identity on any wire message | `provider.Lift`, `terminalbackend` parse arms | wire-absence asserts in all 4 lift vectors; `TestParseProviderDescriptorRefusesWithoutEchoingLocalData` |
| 11 | CheckProviderDescriptor completed; registry owns 7.A | `terminalbackend.ParseProviderDescriptor` + `AdmitProviderDescriptor`; registry | 12 inventory arms + rows (bijection 210/210); `tracecheck` ok |
| 12 | F3: bound rests on true facts, both halves pinned | `provhost/identity.go` header | `TestSpecIdentityExampleVerifiesAgainstItsClaimedDigest`, `TestNoProductionPathAttestsProviderIdentityBinding` (0 production call sites; N2 bound now on the test) |
| 13 | F5: census derives the whole package, every form, fail-closed | `loadProviderCodes` | `TestLiftCoversTheClosedCodeSet` + var/grouped/fail-closed plants |
| 14 | F6: future caller recorded | `terminalbackend/descriptor.go` | STATED BOUND (prose by nature; names path, order, two miswirings); the gate it will guard is driven by row 2 |
| 15 | mutation battery with narrowing / arm-deletion / census-only split | `.temp/TASK-260906-vmzk0y/mut_round4.py` | this round: 22 applied / 21 killed / 1 survived-equivalent + NOT_APPLIED + COMPILE_FAIL distinct; prior batteries stand |

## Production changes in round 4 (no production change)

- `internal/provhost/protocol_test.go`: three new behavioural tests
  (`TestParseMajorSaturationEdge`, `TestParseMajorDigitBoundariesRefuseAtEntry`,
  `TestDecodeResponseUnrecognizedVersionRunsMemberRules`) + `math` import.
- `internal/provhost/identity_test.go`: N2 stated bound moved onto the
  tripwire's doc comment (N6).
- `LOGBOOK.md`: entry 2253 (this round).
- All other round-1/2/3 paths byte-identical; leaf-1 packages untouched
  (`git diff --name-only 1d97474 -- internal/canonicaljson internal/environ
  internal/terminalbackend/manifest.go` returns 0 files — re-verified this
  round). Same 15 paths (11 modified + 4 new) as rev3.

## Gates re-run by this round (nothing accepted from prior rounds)

| Gate | Result |
|---|---|
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `go test ./... -count=1` | exit 0, 22 packages ok, no FAIL |
| `go test ./internal/provider/ ./internal/provhost/ ./internal/terminalbackend/ -cover -count=1` | exit 0 — 97.8% / 85.8% / 94.2% |
| `gofmt -l internal/` | clean |
| `go run ./internal/traceability/cmd/tracecheck` | ok — contracts=60, acceptance_cases=98 |
| provhost refusal-arm bijection | 167/167 derived arms witnessed |

Not run in this round (stated, not implied): `GOOS=windows go build/vet` and
`go test -race` were green in rev3 and are unchanged by this test-only delta;
they were not re-run here for time. No gate is claimed on their behalf.
