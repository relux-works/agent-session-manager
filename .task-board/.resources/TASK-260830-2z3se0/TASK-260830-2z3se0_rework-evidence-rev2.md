# TASK-260830-2z3se0 — rework evidence rev2 (answers review-verdict-rev1)

- Tree: worktree `task-board/story/STORY-260830-3drr2m`, all changes inside
  `internal/sessadapter/` (production + tests). Previously reviewed files
  (`LOGBOOK.md`, `README.md`, `internal/traceability/*`) untouched.
- Method from the verdict brief was followed: every mutant below was confirmed
  PRESENT (grep after patch) and compiling (`go vet` exit 0) before its
  red/green was believed; compile failures would not count as kills.
  Production restored after every mutant (full suite re-run green at the end).
- Battery log: `TASK-260830-2z3se0_mutant-battery-rev2.log` (27 lines, one per
  mutant, with compile + red flags).

## What was done per finding

### F1 — census blind to non-table vocabularies (blocking)
Production: all nine inline unions converted to table loops with `valid*`
consultants (refusal call sites and detail strings byte-identical, so the
323-arm inventory is unchanged):
`objectModes`/`validObjectMode` (context.go), `directionNames`/`validDirection`,
`tupleEntryStatuses`/`validTupleEntryStatus`, `evidenceResults`/
`validEvidenceResult`, `candidateKindNames`/`validCandidateKind`,
`contextFreeOperations`/`operationSkipsRequestContext` (tuple.go,
operations.go), `registryEntryStatuses`/`validRegistryEntryStatus` (probe.go),
`roleNames`/`validRole` (discovery.go). The last three (candidate kind, role,
context-free dispatch) are the same shape the detector found beyond the five
named in the verdict.
Census: `scanVocabularyTables` now resolves named slice/map types package-wide
(`type s []string` + `var t = s{...}` is a table; struct literals are not);
new `TestNoUnregisteredInlineVocabularies` forbids any boolean chain comparing
one operand against 2+ closed values (literals, constants, conversions);
header comment rewritten to state exactly what the scanner sees plus the
syntactic-scanner bound. Registrations 55 → 63.
Spec pins (`TestValueVocabulariesMatchSpec`): object modes, tuple entry
statuses, registry entry statuses, evidence results (both sentences),
context-free operations (derived no-context set), doctor directions, key
directions/roles/candidate kinds now pin the production tables (previously
literals where they existed at all).
Killed by: M01–M09 (table widenings + chain reintroduction).

### F2 — `var x = ctor` alias hole (blocking)
`collectFileArms` (inventory_test.go) now exempts only the definition's own
Name in its ValueSpec; a constructor in value position fails the file in all
three spellings. Doc comment states the exact rule. Regression:
`TestConstructorAliasSpellingsFailDerivation` (control + definition-exempt +
three alias probes on synthetic sources). 323/323 maintained.
Killed by: M11 (package-level var alias → derivation error
`constructor alias of failInvalid buries every arm it builds at ...:3:19`),
M12 (function-local var alias → derivation error), M13 control (direct-call
planted arm derives and is reported unwitnessed
`ctor|failInvalid|planted direct arm`).

### F3 — surrogate pairing bound (blocking)
New `TestSurrogatePairSweep`: every high surrogate (1024) × {lowLo−1, lowLo,
lowHi, lowHi+1} in upper hex, lowercase edge spot checks, full low-range sweep
for first/middle/last high. Verdicts from test-local range literals
(`lowLo=0xDC00, lowHi=0xDFFF`), never from production constants: 2058 refused
+ 5124 admitted.
Killed by: M14 (`second > 0xDFFF+1` admits `A\uD800\uE000B`), M15
(`second < 0xDBFF` admits `A\uD800\uDBFFB`).

### F4 — set-valued gates proven on a sample (blocking)
- `TestCheckDoctorHealthyDrivesEveryRequiredCapability` (probe_test.go):
  required set derived from `DoctorRequiredCapabilities` (both writer
  variants, count pinned at 4), whole-set pass + each member weakened alone
  refusing naming that member (8 subtests).
- `TestCheckValidateNullabilityDrivesEveryTarget`
  (operations_semantic_test.go): archive-carry and staged-miss per member
  through `CheckRequestBody` (8 refusal subtests + 2 passes). Obligation list
  written in-test so a narrowed production list reddens instead of shrinking
  the probe; names cross-pinned by the member census + operation table.
- `TestCheckValidateResultDrivesEveryApplicableCheck`: each of the three
  applicable checks false under valid=true refuses naming it, plus a garbage
  (`"yes"`) per-member type case.
Killed by: M16 (required[:1]), M17/M18 (targets[:3] both branches), M19
(applicable drop), M20 (nullable-bool drop, killed via arm slide
`not a boolean` → `failed applicable check`).

### F5 — same-length context forgery
`TestCheckContextEcho` gains a same-length case: one flipped hex digit in
`operation_id` (length asserted equal, `DecodeCallContext` asserted valid, so
the verdict isolates the echo gate), refused `byte-for-byte`.
Killed by: M21 (echo narrowed to length-only).

### F6 — 8 MiB edges
New `TestFrameBoundEdges`: frames of exactly `MaxFrameBytes` (must fail on
content, never on length) and `MaxFrameBytes+1` (must fail
`exceeds the 8 MiB bound`) through all three entry points
(`DecodeRequestFrame`, `CheckSuccessEnvelope`, `CheckFailureEnvelope`).
Killed by: M22 (`> MaxFrameBytes+1`), M23 (`>= MaxFrameBytes`).

### F7 — zero-value host facts
New `TestZeroValueHostFactsRefuse` (3 subtests) + `TestValidateResultRules/
zero_request_mode_fact`: each zero fact with a non-empty probe side refuses
through its own arm.
Killed by: M24–M27 (zero-guards on the four equalities; M24 kills via arm
slide `requested provider` → `verified manifest`).

### F8 — open-class gates
Recorded as a stated bound in the inventory header (unknown/duplicate-member
gates range over an unbounded name class; one witness per arm proves the gate
fires, not that every name is gated). No code change, as directed.

## Mutant battery (narrowing unless noted; 26 killed + 1 control-green / 27)

| Mutant | Narrows the gate to | Named test that fails | Result |
|---|---|---|---|
| M01 objectModes+reuse_sink | admit one extra mode | TestValueVocabulariesMatchSpec | KILLED |
| M02 tupleEntryStatuses+expired | admit one extra status | TestValueVocabulariesMatchSpec | KILLED |
| M03 evidenceResults+waived | admit one extra result | TestValueVocabulariesMatchSpec | KILLED |
| M04 directionNames+hybrid | admit one extra direction | TestValueVocabulariesMatchSpec | KILLED |
| M05 registryEntryStatuses+pending | admit one extra status | TestDoctorResultRules | KILLED |
| M06 roleNames+spectator | admit one extra role | TestValueVocabulariesMatchSpec | KILLED |
| M07 candidateKindNames+sidecar | admit one extra kind | TestValueVocabulariesMatchSpec | KILLED |
| M08 contextFreeOperations+OpDoctor | skip context for one more op | TestValueVocabulariesMatchSpec | KILLED |
| M09 inline `direction!="source_read"&&!=...` reintroduced (behavior identical) | shape gate, not behavior | TestNoUnregisteredInlineVocabularies | KILLED |
| M10 named-type table spelling | visibility control, must stay green | TestClosedVocabularyTablesAreRegistered | CONTROL-GREEN |
| M11 `var plantedFail=failInvalid` + arm | hide every arm built through it | TestDerivedRefusalArmsAreAllWitnessed (derivation error) | KILLED |
| M12 function-local `var localFail=failInvalid` + arm | same, local spelling | TestDerivedRefusalArmsAreAllWitnessed (derivation error) | KILLED |
| M13 direct-call planted arm | control: derivation must see it | TestDerivedRefusalArmsAreAllWitnessed (unwitnessed arm) | KILLED |
| M14 `second>0xDFFF`→`>0xDFFF+1` | admit U+E000 after high | TestSurrogatePairSweep | KILLED |
| M15 `second<0xDC00`→`<0xDBFF` | admit high-after-high | TestSurrogatePairSweep | KILLED |
| M16 doctor `required[:1]` | check 1 of 4 capabilities | TestCheckDoctorHealthyDrivesEveryRequiredCapability | KILLED |
| M17 nullability archive `targets[:3]` | check 3 of 4 targets | TestCheckValidateNullabilityDrivesEveryTarget | KILLED |
| M18 nullability staged `targets[:3]` | check 3 of 4 targets | TestCheckValidateNullabilityDrivesEveryTarget | KILLED |
| M19 applicable loop drops resume_surface_valid | admit false check | TestCheckValidateResultDrivesEveryApplicableCheck | KILLED |
| M20 nullable-bool loop drops resume_surface_valid | admit non-bool member | TestCheckValidateResultDrivesEveryApplicableCheck (arm slide) | KILLED |
| M21 echo `&& len!=len` | admit same-length forgery | TestCheckContextEcho | KILLED |
| M22 request `>MaxFrameBytes+1` | admit 8 MiB+1 frame | TestFrameBoundEdges | KILLED |
| M23 request `>=MaxFrameBytes` | refuse the exact edge | TestFrameBoundEdges | KILLED |
| M24 provider equality `&& ExpectedProviderID!=""` | admit on zero fact | TestZeroValueHostFactsRefuse (arm slide) | KILLED |
| M25 env equality `&& Manifest.EnvironmentID!=""` | admit on zero fact | TestZeroValueHostFactsRefuse | KILLED |
| M26 kind equality `&& expectedKind!=""` | admit on zero fact | TestZeroValueHostFactsRefuse | KILLED |
| M27 mode echo `&& ValidateMode!=""` | admit on zero fact | TestValidateResultRules/zero_request_mode_fact | KILLED |

Survivors: none. Every survivor class from rev1 is measured above and killed.
Method bounds (what this battery cannot see): no in-repo caller sequences
`Discover → CheckProbe → CheckTupleAdmission → CheckCallBinding` (package has
no in-repo importers); delegated packages (`canonicaljson`, `scalar`,
`axerror`) not attacked; runtime-built sets would evade the syntactic census
(none exist — every table is a package-level composite the scan pins);
open-class member gates per the F8 bound; map-iteration determinism is held by
single-unknown fixtures, and the suite was run `-count=3` green.

## AC coverage: 10 of 11 rows driven through production entry points

| AC row | Production call site | Named committed test |
|---|---|---|
| discovery | Discover (discovery.go) | TestDiscoverBindsManifestToCandidate (+ role table via TestValueVocabulariesMatchSpec/roles) |
| manifest | DecodeManifest (manifest.go) | TestDecodeManifestAcceptsFixture |
| probe | DecodeProbe/CheckProbe (probe.go) | TestDecodeProbeAcceptsFixture, TestCheckProbe*, TestZeroValueHostFactsRefuse |
| closed operations | CheckRequestBody/CheckSuccessBody (operations.go) | TestEveryOperationHasContractVectors (+ per-op semantic tests) |
| limits | DecodeResourceLimits (context.go) | TestDecodeResourceLimitsBounds |
| idempotency | VerifyRequestDigest + CheckContextEcho (context.go) | TestRequestDigestFixpoint, TestCheckContextEcho (now with same-length forgery) |
| tuple gates | CheckTupleAdmission + entry/key decoders (tuple.go) | TestCheckTupleAdmissionRefusals (+ entry-status/result/dir pins) |
| exact fixtures | fixture acceptance tests | TestDecodeManifestAcceptsFixture, TestDecodeProbeAcceptsFixture, contract vectors |
| negative/refusal | all refusal constructors | TestEveryArmWitnessRefusesAtTheProductionEntry (323/323) + per-gate negatives above |
| no unsupported capability | CheckTargetWriteGates, DoctorRequiredCapabilities | TestCheckTargetWriteGates, TestDoctorRequiredCapabilities, TestCheckDoctorHealthyDrivesEveryRequiredCapability |
| crash/idempotency-durable-state evidence | N/A by stated bound | package mutates no durable state (doc.go); no crash-evidence row applicable |

## Anomaly found during F4 (behavior preserved, reported, not redesigned)

`checkValidateNullability` is shared by request and success bodies, but
validate success bodies carry no target members and the member set is exact,
so an archive-mode success is ALWAYS refused (`carries a target member` on
absent members — reproduced with a scratch test, then removed), while a
staged/live success always passes nullability vacuously. The success-side
call is therefore a mode gate in disguise, not a nullability check. F4
coverage is driven request-side (both branches × 4 members) where the rule is
well-defined. Fix options: (a) scope the rule to request bodies; (b) treat
absent as null on success (breaks staged successes — needs a wider redesign);
(c) declare archive success impossible. Needs a product/spec decision — left
for review, recorded in LOGBOOK.md. No behavior changed.

## Gates (exit codes observed directly, no pipes)

- `go test ./... -count=1`: 17/17 packages ok, exit 0
- `go test ./internal/sessadapter/ -v -count=1`: 904 PASS, 0 FAIL, exit 0
- `go test ./internal/sessadapter/ -count=3`: ok, exit 0
- `go test -race ./internal/sessadapter/ -count=1`: ok, exit 0
- `go test ./internal/sessadapter/ -cover`: 81.4% of statements (was 81.1%)
- `go vet ./internal/sessadapter/`: exit 0; `GOOS=windows go vet`: exit 0
- `gofmt -l internal/sessadapter/`: empty
- Mutant battery `/tmp/mutant-battery.py`: 26 killed + 1 control-green / 27,
  every mutant grep-confirmed present and compiling before its verdict
