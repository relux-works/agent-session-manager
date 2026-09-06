# TASK-260830-2z3se0 — mutant table rev7 (census-shape round)

Scoring rule (review rev6): a failure whose ONLY failing top-level tests
are source-text censuses is CENSUS-ONLY, never a behavioural kill.
Census set: TestIdentityGatesAreCensused, TestBoundGuardsAreCensused,
TestRefusalConstructorsMatchProduction, TestClosedVocabularyTablesAreRegistered,
TestNoUnregisteredInlineVocabularies, TestAxerrorImportsAreUnaliased,
TestAllProductionSwitchesAreClassified, TestClosedMemberSetsAreDerivedFromSpec,
TestValueVocabulariesMatchSpec, TestDerivedRefusalArmsAreAllWitnessed,
TestWitnessedArmsAreAllDerived. Every production edit byte-restored after
its row; no `zz_*` files remain.

## Plants — reviewer G-B shapes (dead code: census-only is the correct signal)

| mutant | what it narrows the gate to | named failing test(s) | class |
|---|---|---|---|
| P1: `func zzProbeByteIdentity` with `!bytes.Equal(received, sent)` | new unrostered bytes.Equal gate | TestIdentityGatesAreCensused (`zz_p1_probe.go\|zzProbeByteIdentity\|bytes.Equal(received, sent)\|0`) | census-only |
| P2: `var zzProbeLiteralGate = func...` with `!=` + `len() > 4096` | new unrostered package-literal gate and bound | TestIdentityGatesAreCensused + TestBoundGuardsAreCensused | census-only |
| P3: `length := len(value); if length < 1 \|\| length > 4096` | hidden length-variable bound | TestBoundGuardsAreCensused (`length < N`, `length > N`) | census-only |
| P4: `zzProbeChecker{bound: requireStringBounds}` struct-field indirection | bounds hidden behind alias | TestBoundGuardsAreCensused ("func-value indirection of requireStringBounds") | census-only |
| P5: `func ... { return axerror.New }` | construction hidden behind return | TestRefusalConstructorsMatchProduction ("func-value indirection of axerror.New") | census-only |
| P6: inline `axerror.New(Spec{...literal...})` in a function | construction outside registered bodies | TestRefusalConstructorsMatchProduction ("axerror.New outside any constructor") | census-only |

## Narrowings — gate stays present, admits a rejected member (behavioural)

| mutant | what it narrows the gate to | named failing test(s) | class |
|---|---|---|---|
| P7: `case OpDoctor:` → `case Operation("doctor-zz"):` (operations.go:506) | arm unreachable through the registry | TestEveryArmWitnessRefusesAtTheProductionEntry (`.../bad_direction`: "want refusal, got nil error") | BEHAVIOURAL (arm entry) |
| N1a: `!bytes.Equal(a, b)` → `bytes.Equal(a, b)` (polarity inversion) | admits every forged context | TestCheckContextEcho + 18 downstream behavioural suites; 0 census fires | BEHAVIOURAL-KILL |
| N1b: `!bytes.Equal(a, b)` → `!bytes.Equal(a, b) && len(a) != len(b)` | admits same-length rebuilt context (GA5 repeat) | TestCheckContextEcho ("admitted a same-length rebuilt context") + TestBoundGuardsAreCensused (new `len` site — expected: a length-spelled weakening IS a bound site) | BEHAVIOURAL-KILL |
| N2: `length > maximum` → `length > maximum+1` (checkStringBounds mechanism) | admits max+1 at every string bound | TestRequestScalarBoundEdges ("discover/cursor admitted length 1025 past the 1024 maximum"), TestSuccessScalarBoundEdges, TestStringBoundEdges, TestDecodeManifestValueRules, TestDecodeProbeClosedRules + TestBoundGuardsAreCensused (new `length > maximum+N` shape — expected: exact-render pin) | BEHAVIOURAL-KILL |
| N3: capabilityMapUsable admits `capabilityStatuses[1]` (probe.go:534) | conditional+enabled reaches the write gate | TestCheckTargetWriteGatesRefusesNonAvailableEnabled (conditional_writer_pair, conditional_native_read_back) + TestCapabilityNonAvailableEnabledIsNotUsable/conditional/enabled + TestIdentityGatesAreCensused (new `==` arm — expected: exact-render pin) | BEHAVIOURAL-KILL (G-A re-pin) |
| N4e: cursor `1, 1024` → `1, 1025` | admits 1025-char cursor | TestRequestScalarBoundEdges | BEHAVIOURAL-KILL |

## Survivors — every one a stated bound (green as the headers promise)

| mutant | bound it states | header |
|---|---|---|
| S1: `func zzPlainParamBound(count int)` with `count < 1 \|\| count > 4096` | decoded-value/plain-param comparisons are not derived; suite green | bound shape 9 |
| S2: `if cap(values) > 4096` | direct `cap()` is not a length source; suite green | bound shape 10 |
| S3: `reflect.DeepEqual(a, b)` | non-bytes.Equal stdlib predicates are not derived; suite green | identity shape 8 |

Totals: 15 rows — 6 behavioural kills, 6 census-only kills, 3 stated
survivors, 0 restore failures, 0 unmeasured rows. A mutant with no named
failing test would be an unstated survivor; there is none.
