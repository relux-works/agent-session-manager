# TASK-260830-2z3se0 — reviewer mutation log, CR rev2

Method: each mutant keeps the gate present and weakens it to admit exactly one
member of the class it must reject. Every mutant was written by exact-string
replacement, grep-confirmed present, and `go vet`-clean before its verdict; a
compile failure was re-anchored and re-measured, never counted. The worktree was
restored from a pristine copy after every mutant and the tree OID recomputed
against `4291b80f56f30f399ace7b79a7b189e139ff0582` each time.

**121 narrowing mutants over 93 production gate sites: 110 killed, 11 survived.**

## Batch 1 — obligation loops (G-A): 16 of 16 killed

| ID | Narrows | Result | Named failing test |
|---|---|---|---|
| A1 | `DoctorRequiredCapabilities` gains a 5th member | KILLED | TestCheckDoctorHealthyDrivesEveryRequiredCapability, TestDoctorRequiredCapabilities |
| A2 | `CheckDoctorHealthy` usable loop drops the last member | KILLED | .../official=false/workspace_binding |
| A3 | `CheckDoctorHealthy` name loop drops the last member | KILLED | TestEveryArmWitnessRefusesAtTheProductionEntry |
| A4 | `checkValidateNullability` targets gains `fidelity_profile` | KILLED | TestCheckValidateNullabilityDrivesEveryTarget/archive_carries_each_target |
| A5 | archive branch drops the first target | KILLED | TestValidateResultRules/archive_request_carries_target |
| A6 | staged branch drops the first target | KILLED | TestValidateResultRules/staged_request_misses_target |
| A7 | applicable-check loop drops `identity_valid` | KILLED | TestCheckValidateResultDrivesEveryApplicableCheck/identity_valid |
| A8 | nullable-bool loop drops `identity_valid` | KILLED | .../non_boolean_check_member |
| A9 | target-write adapter loop drops `native_read_back` | KILLED | TestCheckTargetWriteGates |
| A10 | target-write adapter loop gains `raw_capture` | KILLED | TestTargetWriteComplementDerivesNonRequired |
| A11 | target-write provider loop drops `portable_store` | KILLED | TestCheckTargetWriteGates |
| A12 | `decodeCapabilities` checks 14 of 15 | KILLED | TestDecodeProbeAcceptsFixture |
| A13 | `missingMember` drops the last required member | KILLED | TestCheckSuccessEnvelopeRefusals/missing_body |
| A14 | `requestMemberSet` drops the last member | KILLED | TestRequestDigestFixpoint |
| A15 | `successMemberSet` drops the last member | KILLED | TestEveryArmWitnessRefusesAtTheProductionEntry |
| A16 | `validOperation` drops the last operation | KILLED | TestValueVocabulariesMatchSpec/operations_ordered |

## Batch 2 — census shape probes (G-B): 5 of 11 shapes seen

A four-member vocabulary equal to `capabilityStatuses`, consulted live from
`CapabilityUsable` (behaviour-neutral: the value is already validated upstream).

| Shape | Census |
|---|---|
| S1 package-level slice composite (control) | seen — TestClosedVocabularyTablesAreRegistered |
| S2 named slice type composite | seen |
| S3 inline `\|\|` chain | seen — TestNoUnregisteredInlineVocabularies |
| S4 `map[string]struct{}` | seen |
| S5 tagless switch | **blind** |
| S6 tagged switch on non-`operation` ident | seen — TestAllProductionSwitchesAreClassified |
| S7 function-local slice literal | **blind** |
| S8 `make()` + `init()` | **blind** (named in the stated bound) |
| S9 `const` string + `strings.Contains` | **blind** |
| S10 `regexp` alternation | **blind** |
| S11 multi-name package-level `var a, b = …, …` | **blind** |

## Batch 3 — refusal-arm inventory closure (G-C): 8 of 10 killed

| ID | Narrows | Result |
|---|---|---|
| C1 | control: direct-call arm replacing an existing one | KILLED |
| C1b | control: **additive** direct-call arm | KILLED |
| C2 | package-level `var alias = failInvalid` | KILLED |
| C3 | function-local `var alias = failInvalid` | KILLED |
| C4 | `alias := failInvalid` | KILLED |
| C5 | constructor stored in a struct field | KILLED |
| C6 | constructor stored in a slice literal | KILLED |
| C7 | 8th constructor, arm **replacing** an existing one | KILLED (via the 323 floor) |
| C8 | inline `axerror.New`, arm **replacing** an existing one | KILLED (via the 323 floor) |
| C9 | method value calling `failInvalid` with non-literal args | KILLED |
| **C7b** | 8th constructor `failQuota`, **additive** arm | **SURVIVED** |
| **C8b** | inline `axerror.New`, **additive** arm | **SURVIVED** |

C1/C1b are controls and are excluded from the ratio; C7/C8 are superseded by
C7b/C8b, which isolate the closure question from the floor tripwire.

## Batch 4 — every production vocabulary table widened by one member: 57 of 61 killed

61 package-level composites enumerated by parsing production independently of the
census registration table. Killed by `TestValueVocabulariesMatchSpec` (semantic
vocabularies) or `TestClosedMemberSetsAreDerivedFromSpec` (closed bodies).

**Survivors — all four are unknown-member gate maps whose content nothing pins:**

| Table | Site | Consequence |
|---|---|---|
| `successMembers` | protocol.go:370 | success envelope admits an extra member (**confirmed**) |
| `failureMembers` | protocol.go:460 | failure envelope admits an extra member |
| `manifestMembers` | manifest.go:119 | manifest admits an extra member; the census local shadows the package var |
| `doctorResultMembers` | probe.go:584 | doctor result admits an extra member |

Bypass probe: with `successMembers` widened by one name, a frame carrying
`ax.planted.member` is **ADMITTED** by `CheckSuccessEnvelope`; the unwidened
control refuses it (`PROBE-REFUSED`).

## Batch 5 — F3/F5/F6/F7 closure: 17 of 17 killed

| ID | Narrows | Result | Named failing test |
|---|---|---|---|
| F3a | `second > 0xDFFF` → `> 0xDFFF+1` | KILLED | TestSurrogatePairSweep |
| F3b | `second < 0xDC00` → `< 0xDBFF` | KILLED | TestSurrogatePairSweep |
| F3c | `second > 0xDFFF` → `>= 0xDFFF` (refuses a valid pair) | KILLED | TestSurrogatePairSweep |
| F3d | `second < 0xDC00` → `<= 0xDC00` (refuses a valid pair) | KILLED | TestSurrogatePairSweep |
| F3e | `first <= 0xDBFF` → `<= 0xDBFE` | KILLED | TestLoneSurrogateSweep, TestSurrogatePairSweep |
| F5a | echo narrowed to length-only | KILLED | TestCheckContextEcho |
| F5b | echo narrowed to a 32-byte prefix | KILLED | TestCheckContextEcho |
| F6a–f | frame bound `> Max+1` and `>= Max` on request, success, failure | KILLED ×6 | TestFrameBoundEdges/{request,success,failure}/{exact_bound,one_over} |
| F7a | provider equality exempts the zero fact | KILLED | TestZeroValueHostFactsRefuse |
| F7b | environment equality exempts the zero fact | KILLED | TestZeroValueHostFactsRefuse |
| F7c | candidate-kind equality exempts the zero fact | KILLED | TestZeroValueHostFactsRefuse |
| F7d | validate mode echo exempts the zero fact | KILLED | TestValidateResultRules/zero_request_mode_fact |

Excluded as equivalent: dropping `secondLength < 0` is subsumed by the following
range test (`second == 0 < 0xDC00`), so no input distinguishes it.

## Batch 6 — resurrection sweep over round-1 kills: 12 of 12 killed

| ID | Narrows | Result |
|---|---|---|
| R1–R8 | `CheckBindingEquality` drops each of the eight identity facts | KILLED ×8 (TestBindingEqualityFlipsEveryFact) |
| R9 | `checkStringBounds` max → `maximum+1` | KILLED |
| R10 | `checkStringBounds` min → `minimum-1` | KILLED |
| R11 | `checkUint53Bounds` max → `maximum+1` | KILLED |
| R12 | `checkUint53Bounds` min → `minimum+1` | KILLED |

## Batch 7 — `CheckCallBinding` zero-fact exemptions: 0 of 5 killed

| Gate | Result |
|---|---|
| `role != binding.Role` exempts `binding.Role == ""` | **SURVIVED** |
| `context.ProviderID != binding.ProviderID` exempts the zero fact | **SURVIVED** |
| `context.ManifestDigest != binding.AdapterManifestDigest` exempts the zero fact | **SURVIVED** |
| `context.ExecutableSHA256 != binding.ExecutableSHA256` exempts the zero fact | **SURVIVED** |
| `context.Environment != admitted` exempts the zero tuple | **SURVIVED** |

## Open-class bound (not counted in the ratio)

`unknownMember` exempting `"debug"` and exempting `"ax.planted.member"` both
SURVIVE. This is the F8 bound the inventory header records explicitly; the reject
class is unbounded, so no single narrowing mutant is decisive. Correctly declared,
not a finding.

## Behavioural probes (not mutants)

| Probe | Result |
|---|---|
| archive-mode validate success through `CheckSuccessBody` with the package's own fixtures | **REFUSED**: `validate archive mode carries a target member` (member `projection_plan_id`) — finding B1 |
| success envelope carrying an extra member, `successMembers` unwidened (control) | refused |
| success envelope carrying an extra member, `successMembers` widened by one | **ADMITTED** — finding B2 |

## Method bounds

- Measures gates reachable from this package's exported API. No in-repo caller
  sequences `Discover → CheckProbe → CheckTupleAdmission → CheckCallBinding`;
  nothing outside `internal/sessadapter` imports the package.
- Does not attack `internal/canonicaljson`, `internal/scalar`, or `internal/axerror`,
  which several gates delegate to.
- The census shape probes measure syntactic visibility, not reachability: a planted
  vocabulary is consulted live but its verdict is behaviour-neutral by construction,
  which is what isolates the census from the behavioural suite.
- Three of the four unpinned `*Members` maps are reported as unpinned by identical
  construction, not as four separate bypass probes; one was confirmed behaviourally.
