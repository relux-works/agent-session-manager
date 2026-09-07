# TASK-260906-vmzk0y — own-provider-protocol-v3-envelope-and-error-lift (round 3)

Status: ready for review (handed off to review; board status `to-review`).

Candidate tree: `a8570e2f74f3e0cf458a941e985a6d19354881cc`
(verified by `git ls-tree -r`: all 15 changed paths below are in-tree;
`git hash-object` of the worktree `protocol.go`, `protocol_test.go`,
`lift.go`, and `descriptor.go` byte-for-byte equals the tree blobs.
Built via a detached index — `GIT_INDEX_FILE` copy + `read-tree HEAD` +
`add -A` + `write-tree` — so the worktree index is untouched. The four
new files are real blobs in the tree, not dropped-untracked.)

Round 3 answers review-verdict-rev2: one blocking finding (G1,
introduced by the round-2 F2 fix) plus notes N1–N3. Everything else
stands exactly as the reviewer verified it — F1 and F2 refuse before
accumulating (driven through `ParseProviderDescriptor`, `New`,
`CheckVersionTuple`), F4 fails closed with both sub-arms pinned by
different tests, F5's census scans every package file/const/var with
all four plants confirmed by neutering production behaviour, F3's
rewritten bound reproduces on every sub-claim, reviewer battery
23 applied / 23 killed / 0 survivors — and is not re-litigated here.

## G1 (blocking, closed) — saturation no longer short-circuits the shape check

`internal/provhost/protocol.go` `parseMajor`: the saturating
`return math.MaxInt, true` is now `major = math.MaxInt; continue`, so
an overflowing first component still saturates (F2 stays closed) but
the minor/patch validation always runs. Only a fully numeric X.Y.Z
reports a recognized major. The doc paragraph now says exactly that —
it speaks of an all-numeric giant and enforces exactly that class.

Measured through the production entry point `DecodeResponse`
(pre-fix probe vs post-fix probe, same eight vectors):

| `protocol_version` | pre-fix | post-fix |
|---|---|---|
| `2.0.0` (control) | ADMITTED | ADMITTED |
| `3.0.0` | `incompatible_protocol`, exit 6 | `incompatible_protocol`, exit 6 |
| `18446744073709551618.0.0` | `incompatible_protocol`, exit 6 | `incompatible_protocol`, exit 6 |
| `99999999999999999999.0.0` | `incompatible_protocol`, exit 6 | `incompatible_protocol`, exit 6 |
| `99999999999999999999.abc.def` | `incompatible_protocol`, exit 6 | `provider_protocol_error`, exit 13 |
| `99999999999999999999.0.x` | `incompatible_protocol`, exit 6 | `provider_protocol_error`, exit 13 |
| `18446744073709551618..0` | `incompatible_protocol`, exit 6 | `provider_protocol_error`, exit 13 |
| `2.abc.def` (control) | `provider_protocol_error`, exit 13 | `provider_protocol_error`, exit 13 |

The two classes are pinned apart by named tests sharing one frame
builder: all-numeric giants assert exit 6 in `TestParseMajorNeverWraps`
(F2, unchanged), giant-malformed versions assert exit 13 in the new
`TestParseMajorSaturationStillValidatesTheRest` (5 rows:
`<giant>.abc.def`, `<giant>.0.x`, `<giant>..0` over two giant
magnitudes, each asserting `parseMajor` unrecognized AND
`DecodeResponse` code + exit through the production entry). A future
short-circuit reddens the new test while the F2 rows stay green, and
vice versa.

## Correction of the round-2 "no gate was weakened" sentence

The round-2 outcome said "no gate was weakened" and cited the
provhost parse-arm census ("saturation adds no `return 0, false`
branch") as evidence for classification stability. That sentence is
withdrawn as stability evidence: a branch census enumerates rejection
branches, so a change that moves inputs *between* existing branches is
invisible to it. The measured record, stated plainly:

- Intended arm shift (F2 saturation, still in effect, still measured):
  all-numeric giant majors move from the exit-13 unusable-frame arm to
  the exit-6 mismatch arm, witnessed by the `18446744073709551618.0.0`
  exit-6 row of `TestParseMajorNeverWraps`.
- Unintended arm shift (G1, introduced by the same fix, now reverted):
  giant majors with malformed rests moved 13 → 6 with no test able to
  see it — the whole provhost suite passed identically on both
  behaviors. Restored to exit 13 and pinned by
  `TestParseMajorSaturationStillValidatesTheRest`.
- The census claim survives only in its scoped form: saturation adds
  no `return 0, false` branch, so the derived parse-arm inventory
  needs no new row (bijection still 167/167 derived arms witnessed).
  It is inventory bookkeeping, not classification evidence.

## Round-3 mutation battery (G1 gate)

Denominator (production-derived): the 6 version-classification
obligations in `parseMajor` — the 5 `return 0, false` rejection
branches enumerated by `armParseBranches` plus the saturation guard.
Battery script
`.temp/TASK-260906-vmzk0y/mutG1_round3.py` (evidence
`.temp/TASK-260906-vmzk0y/mutsG1_round3.json`); every mutant is
restored byte-identical after its run (asserted by comparison, worktree
`git status` unchanged: 11 modified + 4 new). Each `go test` ran as a
standalone process; the battery was run twice (once piped, once
standalone with exit 0) with byte-identical JSON.

| Mutant | What it narrows the gate to | Named test that fails |
|---|---|---|
| G1-N1 narrowing: saturation `continue` back to early `return MaxInt, true` (the round-2 shape) | giant-with-malformed-rest admitted as recognized (exit 6, not 13) | `TestParseMajorSaturationStillValidatesTheRest` |
| G1-N2 narrowing: `len(rest) == 0` check deleted | exactly the empty-rest member (`<giant>..0`) reads as recognized | `TestParseMajorSaturationStillValidatesTheRest` |
| G1-N3 narrowing: rest loop covers `parts[1:2]` only | exactly the unchecked-patch member (`<giant>.0.x`) reads as recognized | `TestParseMajorSaturationStillValidatesTheRest` |
| G1-N4 narrowing: saturation guard neutered to `major > math.MaxInt` (never fires) | accumulation wraps; wrap-to-2 reads as native major 2 | `TestParseMajorNeverWraps` |
| G1-A1 arm-deletion: whole minor/patch validation loop deleted | every malformed rest with a numeric major reads as recognized | `TestParseMajorSaturationStillValidatesTheRest` |
| G1-A2 arm-deletion: `len(parts) != 3` check deleted | `3.0` / `3.0.0.0` read as recognized foreign majors | `TestDecodeResponseRefusals` |
| G1-C1 census-only: `isFalsePair` matches `"1"`, census derives zero parse arms (production untouched) | inventory must orphan its declared parse witnesses | `TestWitnessedArmsAreAllDerived` |
| G1-C2 census-only, token-preserving: all 5 rejection branches collapse to one `parse|parseMajor` key (every `return 0, false` token kept, production untouched) | witnesses must orphan; proves the harness runs behavior, not only the checker | `TestWitnessedArmsAreAllDerived` |

Applied 8, killed 8, survivors 0 — narrowing 4/4, arm-deletion 2/2,
census-only 2/2. Distinct control rows, not counted as applied:

| Control | Disposition |
|---|---|
| G1-X1: guard deleted outright (`math` import unused) | COMPILE_FAIL |
| G1-X2: anchor with `>=` absent from source (0 sites) | NOT_APPLIED |

There are no surviving mutants, so no survival bounds to state.

## Gates re-run by this round (nothing accepted from round 2)

| Gate | Result |
|---|---|
| `go build ./...` | exit 0 |
| `GOOS=windows go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `GOOS=windows go vet ./...` | exit 0 |
| `go test ./... -count=1` | exit 0, 22 packages ok, no FAIL |
| `go test -race ./internal/provider/ ./internal/provhost/ ./internal/terminalbackend/ -count=1` | exit 0 |
| `go test ./internal/provider/ ./internal/provhost/ ./internal/terminalbackend/ -cover -count=1` | exit 0 — 97.8% / 85.8% / 94.2%, reproduces round 2 exactly |
| `gofmt -l internal/` | clean |
| `go run ./internal/traceability/cmd/tracecheck` | ok — contracts=60, acceptance_cases=98, reproduces round 2 exactly |

## AC row coverage: 14 of 15 driven + 1 stated bound (N1)

Rows 1–13 and 15 are driven through the production entry point by the
named test; row 14 (F6, future caller recorded) is prose by nature and
is reported as a stated bound, not as driven — the DoD permits a
stated bound in place of a driving test. All driving tests below were
re-run green in this round's full suite.

| # | AC / DoD row | Production call site | Driving test / bound |
|---|---|---|---|
| 1 | v3 envelope has exactly one named owner, recorded in provhost stated bounds and the ownership registry | `internal/provhost/doc.go` (bounds prose), `ownership.v0.5.0.json` `section:7.A` | `tracecheck` (production gate over the registry): ok (60 contracts, 98 cases) |
| 2 | working v3 admission or explicit stated bound naming the deferral | `terminalbackend.AdmitProviderDescriptor` (admit); `provhost/doc.go` (transport deferral + follow-up) | `TestAdmitProviderDescriptorAdmitsMatchingBinding`, `TestAdmitProviderDescriptorRefusesMismatch` |
| 3 | v3 envelope reaching provhost → documented outcome, both directions | `provhost.DecodeResponse` (refuse); `terminalbackend.AdmitProviderDescriptor` (admit) | `TestDecodeResponseWellFormedV3SuccessIsMismatch`, `TestDecodeResponseV3FailureWithValid130ErrorIsMismatch`, admit tests above |
| 4 | Error 1.3.0 binding implemented-or-deferred; no silent major-2 binding for v3 | `provhost.DecodeResponse` (contract derived from observed major), `axerror.BindingFor`/`DecodeBound` (table) | `TestDecodeResponseV2FailureWith130ErrorIsBoundTo100`; the v3 test first decodes its embedded error through the production `(provider, 3)` → 1.3.0 binding |
| 5 | F1: geometry bound decided on digits, overflow refused, nothing fabricated | `terminalbackend.descriptorGeometry` (both bound sites) | `TestParseProviderDescriptorGeometryOverflowVectors` |
| 6 | F2+G1: majors never wrap; wrap-to-native refused; saturation validates the full shape, giant-malformed pinned to exit 13 | `terminalbackend.semverMajor`, `provhost.parseMajor` + `DecodeResponse` | `TestParseProviderDescriptorProtocolVersionOverflowVectors`, `TestParseMajorNeverWraps` (exit-6 pin), `TestParseMajorSaturationStillValidatesTheRest` (new exit-13 pin) |
| 7 | F4: registered-but-unlifted code produces no wire object | `provider.Lift` (default arm) | `TestLiftRefusesRegisteredCodeWithoutLiftArm`; `TestLiftRefusesUnregisteredCode` keeps the other sub-arm |
| 8 | discovery failure lifts into an axerror Structured Error through an accepted production path | `provider.Lift` (`internal/provider/lift.go`) | `TestLiftCarriesDiscoveryFailuresToTheWire` (4 real-failure vectors) |
| 9 | causal-leak shape refused (negative) | `axerror.New` (production gate) | `TestNaiveLiftShapeIsRefusedByTheWireGate` (message + details directions) |
| 10 | no machine-local path, provider ID, or owner identity on any wire message | `provider.Lift`, `terminalbackend` parse arms | wire-absence asserts in all 4 lift vectors (secrets first proven present locally); `TestParseProviderDescriptorRefusesWithoutEchoingLocalData` |
| 11 | CheckProviderDescriptor completed; registry owns 7.A | `terminalbackend.ParseProviderDescriptor` + `AdmitProviderDescriptor`; registry | 12 inventory arms + rows (bijection 210/210); `tracecheck` ok |
| 12 | F3: bound rests on true facts, both halves pinned | `provhost/identity.go` header (bound prose) | `TestSpecIdentityExampleVerifiesAgainstItsClaimedDigest`, `TestNoProductionPathAttestsProviderIdentityBinding` (0 production call sites over 125 files) |
| 13 | F5: census derives the whole package, every form, fail-closed | `loadProviderCodes` (`internal/provider/lift_test.go`) | `TestLiftCoversTheClosedCodeSet` + `TestDerivedCodesIncludesVarBinding`, `TestDerivedCodesScansEveryPackageFile`, `TestDerivedCodesIncludesGroupedConsts`, `TestDerivedCodesFailsClosedOnUnclassifiableBinding` |
| 14 | F6: future caller recorded | `terminalbackend/descriptor.go` (future-caller paragraph) | STATED BOUND (prose by nature; names the path, order, and two miswirings); the gate it will guard is driven by row 2 |
| 15 | mutation battery with narrowing / arm-deletion / census-only split | battery scripts `.temp/TASK-260906-vmzk0y/mutG1_round3.py` (round-2 scripts retained in history) | round-3 table above: 8 applied, 8 killed, 0 survivors; round-2 battery 23/23 killed stands |

## Notes N1–N3

- N1: answered above — the ratio is 14 of 15 driven + 1 stated bound.
- N2 (stated bound): the attestation tripwire
  (`TestNoProductionPathAttestsProviderIdentityBinding`) matches
  `CallExpr` only, so a function-value binding of the attestation
  symbol (e.g. `f := canonicaljson.VerifyObjectIdentity; f(x)`) would
  not be seen. No such shape exists today: outside the definition
  itself the only production mentions are doc comments
  (`canonical.go`, `provhost/identity.go` header), every other
  reference is a `_test.go` file (verified by grep this round); the
  scan is otherwise AST-based,
  comment-immune, test-file-excluded, fails closed on `scanned == 0`,
  and control-planted.
- N3: answered — the round-3 battery script and evidence live at
  repo-root `.temp/TASK-260906-vmzk0y/mutG1_round3.py` and
  `mutsG1_round3.json`, not in `/tmp`.

## Production changes in round 3 (3 files of the 15 paths)

- `internal/provhost/protocol.go`: G1 fix (`major = math.MaxInt;
  continue` instead of the saturating early return) + doc paragraph
  narrowed to the enforced class.
- `internal/provhost/protocol_test.go`: new
  `TestParseMajorSaturationStillValidatesTheRest` pinning the exit-13
  class apart from the exit-6 class.
- `LOGBOOK.md`: entry 2252 (G1 finding, correction, gates, handoff).

All other round-1/round-2 paths are byte-identical to the rev2
candidate; leaf-1 packages remain untouched.

