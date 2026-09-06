# TASK-260830-3bkz0c round 2 — prove-shared-environment-implementation-boundary

Rework of review-verdict-rev3 (B1, B2, F1, F2 blocking; B2b follow-up).
Worktree `.temp/STORY-260830-3drr2m/worktree`, base `1296ecc`.
Freeze on `internal/sessadapter` + `internal/dirnode` production lifted for exactly B2+F1.

## B2 — admit hole closed by delegation

Witness value (wire bytes): `\\ud800` + real `\udc00`
(an escaped backslash, literal text `ud800`, a real lone LOW surrogate).

| judge | entry | pre-fix | post-fix |
|---|---|---|---|
| environ | DecodeStrictObject | REFUSE lone-surrogate | REFUSE lone-surrogate |
| canonicaljson | Canonicalize | REFUSE lone low | REFUSE lone low |
| provhost | DecodeManifest | REFUSE lone-surrogate | REFUSE lone-surrogate |
| sessadapter | DecodeTuple (valid 6-member tuple) | ADMIT Version=`\\ud800`+U+FFFD, valid UTF-8 | REFUSE tuple lone-surrogate |
| dirnode | CheckScanRequest | ADMIT | REFUSE scan-request lone-surrogate |

Mechanism: the raw scan matched every `\u` byte pair. At the quoted
`\\ud800` text it read `ud800` as a HIGH surrogate and paired it with the
following real `\udc00`, so the real lone low was never seen — while
`encoding/json` silently rewrote it to U+FFFD (the exact harm
`environ/decode.go` names, SPEC.md:289 MUST violation).

Fix (`internal/sessadapter/decode.go`, `internal/dirnode/decode.go`):
`decodeStrictObject` delegates to `environ.DecodeStrictObject` and
translates `environ.Fault{Detail, Member}` into the package refusal
dialect; the `hasLoneSurrogateEscape` / `readUTF16Escape*` raw-scan copies
are deleted, not widened. Delegation was possible because the two
implementations were line-identical except the gate — the wrapper is
fault translation only, with zero reimplemented rules. Pinned
structurally by `TestDelegatingWrappersCallEnviron` (references
`environ`, builds no decoder/gate/measure, old helper names absent,
no surrogate literal in either file).

Corpus both-directions statement: the first battery sampled backslash-run
parity (runs 1..3) but never composed an even run with a FOLLOWING real
escape — green over the hole on the admit side, while the run-2 rows
proved only the over-strict side. The corpus now has 3 composed rows
(even run + real lone low = the witness; even run + real lone high;
even run + real pair) plus the 5 run-parity rows whose verdicts flipped
to member arms with the frame phrase asserted ABSENT (proving travel
past the frame gate). 7 frame rows flipped in total, each flip reviewed
as the recorded fix, not a silent re-baseline.

Import-cycle note: facades importing `environ` collides with the
in-package agreement batteries importing the facades. The six battery
files moved to the external `environ_test` package (textbook split);
`census_test.go` plus the new unit tests stay in-package. No public API
was exported to serve tests.

## B1 — shape census catches 7/7 live plants

Old name census caught 2/7. New two-layer census (name derivation +
shape derivation, both exact-equality against their ledgers):

Live plants into `internal/provider` (file since removed; full failure
log in evidence): all 9 derived as unregistered —

| plant | old census | new census |
|---|---|---|
| A fresh `decodeStrictObject` func | FAIL (name) | FAIL (name) |
| C3 `hasLoneSurrogateEscape` method | FAIL (name) | FAIL (name) |
| B1 `parseStrictObject` fresh decoder | pass (HOLE) | FAIL strict-decoder |
| B2 `scanForLoneSurrogate` fresh gate | pass (HOLE) | FAIL surrogate-gate |
| B3 `measureString` byte measure | pass (HOLE) | FAIL byte-measure |
| C1 `var stringLength = byteLength` | pass (HOLE) | FAIL string-measure + byte-measure |
| C1 target `byteLength` | pass (HOLE) | FAIL byte-measure |
| C2 `var decodeStrict` closure | pass (HOLE) | FAIL strict-decoder |
| fresh-name env-id grammar copy | pass (HOLE) | FAIL env-id-grammar |

Committed proof (no throwaway): `TestShapeCensusCatchesControls`
(14 controls incl. split-decoder-via-callee, const-bound gate under
fresh names, verdict-style byte measure, local-alias inheritance,
negatives that must stay clean) drives `shapeSitesInPackage`, the same
pure extractor as production; `TestGrammarCensusCatchesFreshName`
classifies via the production `classifyGrammarCandidate`.

Shape ledger: 25 rows (5 decoders incl. 2 envelope decoders the name
census never named, 8 gates/callers incl. scalar's third spelling and
5 canonicaljson rune measures the name census never saw, 10 rune
measures, 2 byte gates). Stated residue: hand-rolled non-encoding/json
decoders, arithmetically-derived gate bounds under fresh names,
non-string byte measures, unknown grammar literals.

## F1 — the library has callers; dead code deleted

- `environ.DecodeStrictObject` + `HasLoneSurrogateEscape`: production
  callers in sessadapter + dirnode (the B2 wrappers). `boundary.md`
  row 1 now names the wrappers.
- `CheckUint53Bounds` / `rawUint53` / `parseUint53Literal` /
  `CheckSortedUniqueStrings` / `Fault.Error`: covered by
  `decode_unit_test.go` (bound edges incl. 2^53 magnitude, both
  sorted-unique halves, fault rendering both branches). Doc claim
  fixed to name this function's own battery.
- `validCapability` (zero references, false census claim): deleted.
- Coverage: environ 76.1% → 89.8%, zero functions at 0.0%.
- New hardening (found by the new test, fixed): `rawUint53`
  admitted `12a` as 12 (Decoder reads one value and stops); trailing
  check added, covered by M4. Sibling copies share the texture but are
  unreachable via production paths (frame gates first) → follow-up
  TASK-260906-33xcnc.

## F2 — 27 applied / 27 killed over the derived 89

Denominator re-derived from production: 31 `refuse()` arms + 58 boolean
`false` exits = 89 refusal exits (confirmed by count, not by sampling).

| mutant | what it narrows the gate to | named killing test |
|---|---|---|
| N1_lowsurrogate | admits lone lows | FrameAgreement/bare_low_escape |
| N2_bytelength | StringLength counts bytes | StringMeasureCountsRunes/environ_helper |
| N3_tuplextensions | tuple admits extensions | TupleAgreement/extensions_refused |
| N4_caplength | capabilities admit 9th key | Observation/ninth_capability_refused |
| N6_envunderscore | env-id admits underscore | SharedGrammarsAreOneLanguage |
| T1_rawscan | gate scans raw pairs, keeps name+texts | 7 escaped_backslash* rows; census green |
| M1_uint53floor | admits below-min | CheckUint53BoundsEdges |
| M2_uint53magnitude | admits exactly 2^53 | CheckUint53BoundsEdges |
| M4_uint53trailing | admits `12a` as 12 | CheckUint53BoundsEdges |
| M5_stringsdup | strings admit duplicates | SortedUniqueStringsHalves/sorted_duplicated |
| M6_stringsorder | strings admit unsorted | SortedUniqueStringsHalves/unsorted_unique |
| M7_digestbridge | digest admits non-digests | TupleAgreement/bad_fingerprint |
| M8_timestampbridge | timestamp admits non-timestamps | Observation/bad_timestamp |
| M9_tuplearch | tuple admits x86 | TupleAgreement/x86_refused |
| M10_tuplesemver | tuple admits `1.2` | TupleAgreement/short_version |
| M11_reasonavailable | admits available-with-reason | Observation/available_with_reason |
| M12_reasonconditional | admits conditional-without-reason | Observation/conditional_without_reason |
| M13_capabsent | presence tolerates missing directory_discovery | Observation/unknown_capability_key |
| M14a_evidencedup | evidence admits duplicates | Observation/duplicated_evidence |
| M14b_evidenceorder | evidence admits unsorted | Observation/unsorted_evidence |
| M15_framedup | frame admits duplicates | FrameAgreement/duplicate_member |
| M16_frametrailing | frame admits trailing data | FrameAgreement/trailing_data |
| M17_frameutf8 | frame admits non-UTF-8 | FrameAgreement/non-utf8_bytes |
| M18_highmispair | admits high-followed-by-non-low | FrameAgreement/high_followed_by_non-low |
| M19_extensions | extensions admit nodots | Observation/bad_extensions_key |
| M20_tupleunknown | tuple admits score | TupleAgreement/unknown_member |
| M21_obsschema | observation admits foreign schema | Observation/wrong_schema |

T1 (token-preserving: name, texts, structure kept; string walk
replaced by raw entry scan) reddens exactly 7 behavioral rows
(5 run-parity + plus_real_lone_low + plus_real_pair; plus_real_lone_high
agrees by construction) while `TestSharedImplementationsAreCensused` +
`TestSharedGrammarsAreOneLanguage` stay green — the census kill is
reported separately and is not counted in the narrowing ratio.
Delete-only mutants were deliberately not run (existence-only, not
evidence, per standing orders).

Exit coverage: 22 of 89 refusal exits directly weakened-and-killed
(24.7%). N2 (measure value) and N6 (grammar literal) are narrowing
kills over rule values outside the exit denominator. Residue (67 exits)
by function — decode.go: readUTF16Escape 2, unknownMember/missingMember
2, rawString 1, rawUint53 3 (decode-fail, non-number, fraction/exponent/
sign set — M3 dropped: the digit ladder independently refuses, so no
single-condition mutant admits), parseUint53Literal non-digit 1,
CheckStringBounds 2, CheckUint53Bounds max 1, CheckDigest non-string 1,
CheckUUIDv7 2, CheckTimestamp non-string 1, CheckExtensions frame-fault 1,
decodeArray 6, CheckSortedUniqueStrings 3 (non-array, count, item),
CheckSortedUniqueDigests 3; tuple.go refuse arms 6 (missing, env-id,
version, platform x2, fingerprint); observation.go refuse arms 18
(missing, unknown, env-id, env-version, provider x2, platform x2, arch,
obs/host/install/realm/auth/runtime/timestamp/extensions/schema-version);
observation.go bools 14 (3 vocabularies, checkCapabilities 2,
checkCapabilityResult 9). Every residue exit is behaviorally covered by
its named refusal row; none has a run narrowing mutant — that is the
stated bound, and extending the battery over it is in TASK-260906-33xcnc.

Harness: `/tmp/mutbattery/run.py` (27 exact-match appliers, full
`go test ./internal/environ/` gate per mutant, kill requires the named
test in the failure output, baseline copy-back restore; tree verified
identical after the battery). Two mutants were corrected during the run
(M4's first condition still refused syntax errors; M13's target row was
the swap row, not the seven row — the count arm fires first), both
corrections recorded here, final battery 27/27 from a clean tree.

## AC mapping — 7 of 7 rows driven

1. Shared strict-frame decoder: frame battery drives all five judges
   incl. the witness row (production: environ.DecodeStrictObject,
   sessadapter.DecodeTuple, dirnode.CheckScanRequest,
   provhost.DecodeManifest, canonicaljson.Canonicalize).
2. Shared tuple model: tuple battery incl. new-member/unknown/semver/
   arch rows through environ.DecodeTuple + sessadapter.DecodeTuple.
3. Shared observation model: observation battery through
   environ.DecodeEnvironmentObservation.
4. Shared measure: rune-vs-byte rows incl. multibyte vectors through
   all four measurers + 2 byte gates.
5. Shared identity/grammars: identity battery + grammar one-language
   incl. fresh-name rule.
6. Negative/refusal coverage: every row above asserts refusal text or
   absence; narrowing battery ties 26 gates to failing tests.
7. No unsupported capability: no new public API (external test package
   split instead of exports); `validCapability` deleted rather than
   advertised; tracecheck unchanged (94 cases).

Production call sites are named per row; the ratio is 7 of 7, not prose.

## B2b — follow-up on the board

Residue survives (provhost/canonicaljson/scalar copies, remaining
per-helper copies, sibling rawUint53 texture, 67-exit mutant residue):
TASK-260906-33xcnc `unify-remaining-environment-copies` created under
STORY-260830-3drr2m and referenced from the updated boundary.md.

## Candidate

Tree OID: 08ad1875394e617b79bafaf2efb239934801b090 (commit 82c38378fe79b3e2a3e9fa337c23643b856ba7ee, signed, verified; PR #35 head updated)
Gates: `go test ./... -count=1 -p 2` exit 0 (19 pkgs); environ `-race`
exit 0; cover 89.8%; `go vet ./...` + `GOOS=windows go vet` exit 0;
gofmt clean; `tracecheck` exit 0 (94, unchanged); `go build` exit 0.
Change Request publication is an orchestrator step; the signed feature
head is pushed and the PR updated.
