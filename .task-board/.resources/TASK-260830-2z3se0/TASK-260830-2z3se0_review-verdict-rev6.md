# TASK-260830-2z3se0 — reviewer verdict, CR revision 6

- Verdict: **changes requested** → `to-dev`
- repeat-of: `rev5/F1` **and** `rev5/F2` — same class, third consecutive round:
  the census's own unqualified sentence is false for a shape it does not derive.
  See "Routing note" — the next step is a bounded stated-bound fix, not another
  derivation widening.
- Candidate tree `289f5f9bf3ee2dd5f4be1f69950c49fb8510420b`, base
  `1cb6b93585749df4ef4755ce93e486bcde9e3a6b`, verified equal to the worktree
  before the first probe and after the last one
  (`git read-tree HEAD && git add -A && git write-tree`).
- Run: `RUN-260906-245a60`. **445 mutants applied** by byte offset, PRESENT-asserted,
  restored by byte-copy with a full-content comparison; **428 measured** (the 17
  that could not compile were re-typed by hand and re-measured, not counted as
  passes). **0 restore failures, 0 NOT-APPLIED, 0 unmeasured rows.**

## Gate results

| Gate | Result |
| --- | --- |
| G-A — settle `conditional` at `CheckTargetWriteGates` | **passes** |
| G-B — shapes covered over shapes that exist, denominator built independently | **fails** (F1) |
| G-C — re-run the battery; behavioural vs census-only; resurrections; survivors | **passes** |

Baseline before any mutant: `go build ./...` 0, `go vet ./...` 0,
`gofmt -l internal/` clean, `go test ./... -count=1` 17/17 ok (29.3s),
`tracecheck` 0 (`acceptance_cases=88`).

## G-A — settled, and settled in the direction the rev5 question left open

Round 5 asked whether the decode-side coherence gate makes `conditional`
unreachable at `CheckTargetWriteGates`, or whether a write gate admits a status
it must refuse. The producer answered the first half correctly — it does **not**
make it unreachable, because `CheckTargetWriteGates` and `CapabilityUsable` are
exported and take caller-built maps — and pinned the clause at every site. I did
not take that on report. Six targeted probes, each keeping the gate present and
admitting exactly one member of the rejected class:

| probe | narrows to | verdict | named failing test |
| --- | --- | --- | --- |
| GA1 `CapabilityUsable` status clause admits `capabilityStatuses[1]` | conditional+enabled usable | BEHAVIOURAL-KILL | `TestCapabilityNonAvailableEnabledIsNotUsable/conditional/enabled` |
| GA2 `capabilityMapUsable` same | conditional+enabled reaches the write gate | BEHAVIOURAL-KILL | `TestCheckTargetWriteGatesRefusesNonAvailableEnabled` (4 subtests) + the unit test |
| GA3 provider side admits conditional | conditional provider usable | BEHAVIOURAL-KILL | `…/conditional_provider` |
| GA4 decode coherence gate exempts `conditional` | decoded probe carries conditional+enabled | BEHAVIOURAL-KILL | `TestCapabilityStatusMatrix`, `TestEveryArmWitnessRefusesAtTheProductionEntry` |
| GA5 `CheckContextEcho` admits any same-length mismatch | rebuilt context of equal length | BEHAVIOURAL-KILL | `TestCheckContextEcho` |
| GA6 doctor required-capability check exempts conditional | doctor healthy with a conditional capability | BEHAVIOURAL-KILL | `TestCheckDoctorHealthyDrivesEveryRequiredCapability`, `TestDoctorResultRules` |

GA2 is the exact probe the rev5 verdict asked for and it fails at
`CheckTargetWriteGates`, not only at the unit predicate. GA4 shows the two
halves of the belt-and-braces pair are independently pinned: neither is carrying
the other. **G-A is closed. Do not re-open it.**

## G-C — 445 mutants applied / 428 measured, behavioural kills separated from census-only

Killers are classified the rev5 way: a failure whose only failing top-level
tests are the source-text censuses is `CENSUS-ONLY`, never a behavioural kill.
Census set: `TestIdentityGatesAreCensused`, `TestBoundGuardsAreCensused`,
`TestRefusalConstructorsMatchProduction`, `TestClosedVocabularyTablesAreRegistered`,
`TestNoUnregisteredInlineVocabularies`, `TestAxerrorImportsAreUnaliased`,
`TestAllProductionSwitchesAreClassified`, `TestClosedMemberSetsAreDerivedFromSpec`,
`TestValueVocabulariesMatchSpec`, `TestDerivedRefusalArmsAreAllWitnessed`,
`TestWitnessedArmsAreAllDerived`.

| Battery | applied | behavioural kills | survived | census-only |
| --- | ---: | ---: | ---: | ---: |
| A — helper bound edges, **WIDEN** (admits a member it must reject) | 101 | 93 | 8 | 0 |
| A — helper bound edges, **NARROW** (rejects a member it must admit) | 118 | 97 | 21 | 0 |
| C — `len` guards | 9 | 8 | 1 | 0 |
| D — identity gates, both operands, both spellings | 156 | 94 | 0 | 62 |
| E — hand-typed re-runs of D's 17 non-compiling rows | 17 | 10 | 0 | 7 |
| E2 — shape controls for 4 of E (no `len` introduced) | 4 | 0 | 0 | 4 |
| G-A — targeted capability/echo probes | 6 | 6 | 0 | 0 |
| G-B — census blind-spot plants and controls | 8 | 1 | 3 | 4 |
| R — the rev5 survivors my generator does not emit | 9 | 0 | 9 | 0 |
| **total measured** | **428** | **309** | **42** | **77** |

(The log carries 445 rows: the 428 above plus the 17 `COMPILE-FAIL` attempts that
battery E replaces.)

The battery is generated from production syntax, not from the census
registries: every INT literal argument of every one of the six bound helpers is
moved by one in **both** directions, every `len` comparison literal is moved
outward, and **both operands** of every derived identity gate are widened to
exempt their zero. D's 17 rows that could not compile (`go vet`'s
`redundant or`, or a non-string operand) were **re-typed by hand and re-measured**
rather than counted as passes — an unmeasured row is not a kill.

### Identity class: 173 measured rows, **0 survivors**

Every one of the 173 rows (156 generated + 17 hand-typed) either produces a
behavioural kill or is `CENSUS-ONLY` on a row that says exactly why. I checked
all 7 hand-typed `CENSUS-ONLY` results against their registry rows rather than
accepting the classification:

- 4 × `previous != ""` sentinels (`checkExcludedClasses`, `checkRequiredDispositions`,
  `checkContractsShape` ×2) — rows say "first-iteration sortedness sentinel, not a
  fact gate". A control re-shape (`&& previous != "a"`, introducing no `len` row)
  is likewise census-only. Consistent with the declared role.
- `decode.go|rawUint53|literal == ""` — row says defensive/unreachable. Correct:
  `json.Number.String` never yields an empty literal.
- `CheckCallBinding|context.Environment != admitted` — the *admitted* side dies
  (`TestCheckCallBindingZeroFactsRefuse`); the caller side is census-only, exactly
  as the row's "caller side unreachable, tuple always decoded" says.
- `CheckTupleAdmission|entry.Key.Environment != environment` — the *entry* side
  dies (`TestTupleAdmissionZeroFactsRefuse`); the probed side is the row's stated
  hand-built-`Tuple{}` bound.

**No unmeasured fact side. No unstated identity survivor.**

### Bound class, admit direction: 18 widening survivors, **all rostered**

| survivor | registry disposition | verified |
| --- | --- | --- |
| `manifest.go\|DecodeManifest\|checkSortedUniqueStrings\|"platforms"` element-min 1→0, element-max 16→17, count-max 4→5 | exempt: "equivalent … unreachable behind `scalar.ParsePlatform` and the four-name closed set" | yes — every element runs through `scalar.ParsePlatform`; a fifth member does not exist. rev5 noted the element *minimum* is not spelled out in the row wording; it still is not. |
| 5 × `65536` count ceilings (`canonical_event_ids`, `canonical_event_candidate_ids`, `raw_reference_ids`, `required_source_object_ids`, `created_resource_keys`) | exempt: "stated bound: 65537 digests" | yes |
| `decode.go\|checkExtensions\|len(name) < 3` → `< 2` | exempt: "no 2-character key satisfies the reverse-DNS grammar" | yes |
| 8 × `maxUint53` maxima → `maxUint53+1` | census header class-level stated bound; `parseUint53Literal` refuses above `maxUint53` at parse time | yes — `TestUint53RepresentabilityCeiling` proves the mechanism |
| `protocol.go\|bytesTrimSpace\|len(trimmed) == 0` → `== 1` | rostered plumbing, a trim helper with no refusal | yes |

**No unstated widening survivor.** This reproduces rev5's finding on byte-identical
production.

### Bound class, accept direction — new measurement, reported not charged

I also ran the direction rev5 did not: 118 mutants that *tighten* a bound
(min→min+1, max→max−1). **21 survive**, meaning no test supplies a value at the
accepted edge itself — e.g. `context.go|DecodeReadAuthority|"root_handle_names"`
admits elements of length 1..128 and no test drives a 1-character or a 128-character
element. All 21 sit on rows that name a driver.

This is **not** a rework item: the DoD's mutant requirement is the admit
direction, and that direction passes 93 of 101 with every survivor rostered.
I record it because the bound census header claims its drivers drive
"min-1/min/max/max+1", and for these 21 sites they drive the refusal side only.
Fold it into the stated bound if you touch the header for F1; do not open a
round on it.

### Resurrections: none, and none possible

- `git diff c0174e87 289f5f9b -- '*.go' ':(exclude)*_test.go'` is **empty**.
  Production is byte-identical to rev5, and rev5 established it byte-identical
  to rev4. The three changed files are `bound_census_test.go`,
  `identity_census_test.go`, `probe_test.go`.
- Test inventory diffed across the two trees: **103 → 109 top-level tests, 6
  added, 0 removed.** `probe_test.go` is pure addition (+75/−0). A kill cannot
  resurrect against identical production and a strict superset of tests.
- rev5's 13 survivors: all 13 re-measured, **none moved**. My generator emits 10
  of them directly; the remaining 3 classes (`maxUint53` maxima, the
  `bytesTrimSpace` trim guard) were run as battery R and survive with the same
  verified rationale.

## AC coverage — 7 of 7

| AC row | production call site | named driving tests |
| --- | --- | --- |
| discovery | `internal/sessadapter/discovery.go` `Discover` | `TestDiscoverBindsManifestToCandidate`, `TestDiscoverRefusals` |
| manifest | `manifest.go` `DecodeManifest` | `TestDecodeManifestAcceptsFixture`, `TestDecodeManifestValueRules` |
| probe | `probe.go` `DecodeProbe` | `TestDecodeProbeAcceptsFixture`, `TestDecodeProbeClosedRules` |
| closed operations | `operations.go` `CheckRequestBody` | `TestEveryOperationHasContractVectors`, derived-vector suite |
| limits | `context.go` `DecodeResourceLimits` | `TestDecodeResourceLimitsBounds` |
| idempotency | `context.go` `VerifyRequestDigest` | `TestRequestDigestFixpoint` + echo suite |
| tuple gates | `tuple.go` `CheckTupleAdmission` | `TestCheckTupleAdmissionRefusals` |

`tracecheck` exit 0 validates every declaration and test path in the registry.
The AC's crash/idempotency clause is declared vacuous with a stated bound
(`doc.go`: the package mutates no durable state; the idempotency it enforces is
the request-digest binding, the byte-for-byte context echo, and the fresh-sink
rule) and the README says the Section 7.8 clause binding stays `unevidenced`
rather than claiming clause coverage. Honest — no unsupported claim.

---

## F1 — G-B was asked and not answered; three shapes are open, one with a live production instance

`repeat-of: rev5/F1` (a spelling the identity census cannot derive) and
`rev5/F2` (a declaration context neither census descends into).

The brief asked, for each extended census, "what is the shape after this one?"
and closed with "If any is genuinely out of scope, that is a bound to write
down, not a silence." The rework evidence discusses none of the three shapes it
named, reports no shapes ratio, and neither census header gained a stated bound.
Both headers still carry their unqualified sentence:

- `identity_census_test.go:21` — *"A new identity gate with no row fails the census."*
- `bound_census_test.go:25` — *"A new bound with no row fails the census."*

I built the denominator independently of the registries (a `go/ast` shape scan
of the eight production files) and probed each shape with a plant. **8 probes;
each plant added to production, then `go build ./...`, `go vet ./...`, and
`go test ./... -count=1` across all 17 packages:**

| shape | probe | result |
| --- | --- | --- |
| identity gate spelled `bytes.Equal` | `if !bytes.Equal(received, sent) { return errors.New(…) }` | **SURVIVED** — build 0, vet 0, whole repo green, unrostered |
| identity gate + `len` bound inside a **package-level func literal** | `var zzProbeLiteralGate = func(observed, expected string) error { if observed != expected {…}; if len(observed) > 4096 {…} }` | **SURVIVED** — invisible to *both* censuses |
| bound whose comparison holds no `len` call | `length := len(value); if length < 1 \|\| length > 4096 {…}` | **SURVIVED** |
| bound helper through a struct field | `zzProbeChecker{bound: requireStringBounds}` then `checker.bound(…)` | covered — `TestBoundGuardsAreCensused` reports the indirection |
| constructor through a function that returns it | `func f() func(axerror.Spec)… { return axerror.New }` | covered — `TestRefusalConstructorsMatchProduction` |
| tagged-switch arm made unreachable | `case OpDoctor:` → `case Operation("doctor-zz"):` | covered — `TestEveryArmWitnessRefusesAtTheProductionEntry` |
| control: the same `!=` gate as a `FuncDecl` | — | reddens `TestIdentityGatesAreCensused`, naming the site |
| control: the same code as a `FuncDecl` | — | reddens **both** censuses |

The two controls make the probes fair: identical code in a `FuncDecl` is caught;
the same code one declaration level up is not.

**Shapes covered over shapes that exist, denominator built independently:**

- identity-decision shapes: `==`/`!=` BinaryExpr ✓, tagged switch ✓ (switch
  census allows only `switch operation`), tagless switch ✓ (only
  `readUTF16Escape`), map-presence ✓ (`presenceRegistrations`), inline
  vocabulary chain ✓ (chain census), stdlib equality predicate ✗,
  package-level func-literal scope ✗ → **5 of 7**
- bound shapes: six helpers direct ✓, helper through any indirection ✓
  (var/`:=`/assignment/argument/return/struct field/method value), `len`
  comparison in any statement context ✓, cross-field relational — stated ✓,
  `maxUint53` maxima — stated ✓, length through a variable ✗, package-level
  func-literal scope ✗ → **5 of 7**
- constructor shapes: direct, alias, import alias, returned-by-function, inline
  outside a registered body → **5 of 5**

**15 of 19; 4 open; 3 of the 4 demonstrated open by a plant that survives the
entire repository suite.** (`utf8.RuneCountInString`/`cap` as a direct bound
source is the fourth; I did not plant it — its only live use is inside
`stringLength`, whose callers are rostered, so it has no live gap today.)

Two of the four are not hypothetical:

1. **`context.go:222` `CheckContextEcho`** is a live identity gate —
   `if !bytes.Equal(received.raw, sent.raw) { failProtocol("success context does
   not echo the request context byte-for-byte", "context") }` — doing exactly the
   echo-identity job this census was built for, in a spelling the census cannot
   derive, with no row and no stated bound. It is behaviourally covered today:
   GA5 narrows it to admit any same-length mismatch and
   `TestCheckContextEcho` fails. So this is a durability finding, not a live
   hole — the same standing rev5/F1's two `discovery.go` rows had.
2. **`protocol.go:126-155`** already ships seven package-level func literals (the
   refusal constructors). The constructor census descends into them by design
   ("the set of package-level func-valued vars whose body constructs an error").
   The identity and bound censuses do not. The fix that was ported to one
   instrument was not ported to the other two — the rev5/F2 shape again.

### What closes this

Either extend the two derivations, or — better — **write the bound down**. Add to
each census header a "Stated bounds" clause naming what the derivation does not
see, and qualify the two unqualified sentences to the shapes actually derived.
Concretely: `bytes.Equal`/stdlib equality predicates, declaration contexts other
than `FuncDecl` bodies, and length values bound to a variable before comparison.
If you extend a derivation instead, the accompanying stated bound is still
required, because the next shape after the extension will exist too.

### Routing note for the orchestrator

This is the third consecutive round on one class: rev4/N8, rev5/F1+F2, now this.
Each round's extension closed the shape that was found and left the next one, and
I can demonstrate that directly — the shapes I plant today are the shapes the
rev5 fixes did not reach. Per the repeat-of rule this should be a **gate**, not an
open-ended seventh derivation-widening round. The bounded ask above (a stated
bound plus two qualified sentences, no new derivation required) is what I would
route; an equally defensible call is to accept the CR on a policy decision that
an unqualified census sentence is acceptable given the behavioural evidence, and
record the four open shapes on the Story so the two following leaves inherit them
explicitly. That call is the orchestrator's, not mine — but it should not be made
by silence, which is what the current revision does.

## Explicitly closed — do not re-litigate in round 7

- G-A, the capability-status class, at all three sites plus the decode coherence
  gate and the doctor required-capability path. Six probes, six behavioural kills.
- The identity class over its production-derived denominator: 173 rows, 0
  survivors, every exemption verified against the code.
- The bound class in the admit direction: 101+9 widening mutants, 18 survivors,
  every one on a committed row or the census's declared class-level stated bound.
- Resurrections: impossible by construction; production byte-identical to rev4
  and rev5, zero tests removed.
- AC coverage 7 of 7; README/tracecheck claims honest (`acceptance_cases=88`,
  Section 7.8 clause binding disclosed as `unevidenced`).
- The bound accept-direction survivors (21) — measured and reported above, not a
  rework item.

## Reproduction

```bash
# tree identity, before and after
git read-tree HEAD && git add -A && git write-tree     # 289f5f9b…

# rev5 -> rev6 production delta (empty)
git diff c0174e8761518432e55ed5bdcf915959da9bd9f4 289f5f9bf3ee2dd5f4be1f69950c49fb8510420b \
  -- '*.go' ':(exclude)*_test.go'

# G-B plants: add to internal/sessadapter/, then
go build ./... && go vet ./... && go test ./... -count=1     # all three stay green
#   zz_gb1.go  func zzProbeByteIdentity(received, sent []byte) error {
#                  if !bytes.Equal(received, sent) { return errors.New("…") }; return nil }
#   zz_gb2.go  var zzProbeLiteralGate = func(observed, expected string) error {
#                  if observed != expected { return errors.New("…") }
#                  if len(observed) > 4096 { return errors.New("…") }; return nil }
#   zz_gb3.go  func zzProbeLenVariable(value string) error {
#                  length := len(value)
#                  if length < 1 || length > 4096 { return errors.New("…") }; return nil }
# controls (must redden): the same bodies written as plain func declarations.

# G-A
#   probe.go CapabilityUsable / capabilityMapUsable:
#     value.Status == "available"
#       -> (value.Status == "available" || value.Status == capabilityStatuses[1])
go test ./internal/sessadapter/ -count=1     # red at
#     TestCapabilityNonAvailableEnabledIsNotUsable and
#     TestCheckTargetWriteGatesRefusesNonAvailableEnabled
```

Battery log attached as `TASK-260830-2z3se0_review-battery-rev6-RUN-260906-245a60.log`
(445 rows: mutant id, verdict, failing behavioural tests, failing census tests,
restore check, wall time).
