# TASK-260830-2z3se0 — reviewer verdict, CR revision 7

- Verdict: **accepted** → `accept_cr(TASK-260830-2z3se0, revision=7)`
- repeat-of: `none`
- Run: `RUN-260906-129580`. Candidate tree `070d52005f909ec92ec3b97d5d141860ee7d117b`,
  base `1cb6b93585749df4ef4755ce93e486bcde9e3a6b`, verified equal to the worktree
  before the first probe and after the last one
  (`git read-tree HEAD && git add -A && git write-tree`). Every mutant ran in a
  scratch copy of the module under `/tmp`, so the candidate tree was never
  written to.
- **451 mutant rows applied, 434 measured** (the 17 that cannot compile are
  replaced by the 17 hand-typed E rows, not counted as passes),
  **0 restore failures, 0 unmeasured rows.**

## Gate results

| Gate | Result |
| --- | --- |
| G-A — audit the enumeration, not the two shapes | **passes** |
| G-B — same treatment for the identity and constructor censuses | **passes** |
| G-C — re-run the battery; behavioural vs census-only; survivors; resurrections | **passes** |

Baseline before any probe: `go build ./...` 0, `go vet ./...` 0, `gofmt -l internal/`
clean, `go test ./... -count=1` **17/17 ok** (28.9s), `go test ./internal/sessadapter/
-cover` **82.4%**, `tracecheck` 0 (`acceptance_cases=88`,
`clauses_discharged=17/428`).

**Scope of the revision, established independently:** I materialised the rev6
candidate tree from its patch resource and hashed it file by file against rev7.
The delta is `bound_census_test.go`, `identity_census_test.go`,
`inventory_test.go`, `LOGBOOK.md` — **all nine production files byte-identical**,
README and the traceability files byte-identical. Test inventory 109 → 111
top-level, **0 removed**. The rework evidence's "test files only, zero
production bytes changed" is true as stated.

---

## G-A — the enumeration audited over its own claims

The claim under review moved up a level: not "these shapes are covered" but
"a bound in this package **can only** be carried by these AST shapes". Three
questions were put; here is what each measured.

### 1. Is the ten-shape list derived, or a hand enumeration?

It is a hand enumeration. Nothing in the package derives the shape space — the
census derives *sites within* the shapes, never the shape set. So the correct
test is not "is it derived" but "is every falsifiable claim inside it true".
I built the denominator independently of the census (`/tmp/residue.go`, a
`go/ast` scan of the nine production files that looks for each claimed-empty
shape directly) and checked all seven:

| header claim | independent check | result |
| --- | --- | --- |
| identity shape 3: `bytes.Equal` — **one** live row, `CheckContextEcho` | scan of every equality-ish selector call | **1**, `context.go:222` — exact |
| identity shape 8: other stdlib equality predicates — **none** in production | same scan (`DeepEqual`, `EqualFold`, `Compare`, `slices/maps.Equal`) | **0** |
| identity shape 2 / bound shape 6: package-level var initializers — **zero live rows** | comparison + ordering scan inside every package-level `var` initializer | **0 / 0** — including `protocol.go`'s seven refusal func literals, which hold no comparison |
| identity shapes 4-5: switch inventory | every `SwitchStmt` in production | **4**: 3 × `switch operation`, 1 tagless (`readUTF16Escape`) — exactly what the switch census allows |
| bound shape 5: direct `stringLength(...)` in a comparison — **no live instance** | every `stringLength` use | **2**, both bound to a variable first (`decode.go:254`, `:370`) |
| bound shape 10: `cap(...)` / `utf8.RuneCountInString(...)` outside `stringLength` — **no live instance** | `cap` builtin + selector-call length measures | **cap 0**; RuneCountInString **1**, inside `stringLength` (`decode.go:244`) |
| bound shape 4 residue: only a *direct* length binding counts | every binding whose RHS contains a length measure but is not a bare call/conversion | **9**, every one a `make([]T, 0, len(x))` capacity — no live bound |

**Every no-live-instance and live-count claim in both headers is true.** None
of them rests on the census's own report.

### 2. Shape 9 is a claim about a set — checked over the set

Shape 9 states the decoded-value comparison inside `checkUint53Bounds`
(`decode.go:269`) is "enforced at every caller row, and every caller row carries
a driver or a pinned exemption". Production has **15** `checkUint53Bounds` /
`requireUint53Bounds` call sites (5 `context.go`, 1 `probe.go`, 1 inside the
`requireUint53Bounds` forwarder, 6 forwarder callers, 2 `tuple.go`). The census
holds **15** rows for them: 14 name a driver, 1 is the forwarder's mechanism
exemption naming its rationale and its six caller rows. **15 of 15.** The set
claim holds over the set.

### 3. Fourteen plants against the enumeration — 11 caught, 3 survive

Each plant is production code added to the package, then `go test
./internal/sessadapter/ -count=1` over the whole behavioural suite (not the
static checker alone), byte-restored after. Two are controls whose only job is
to make the probes fair.

| plant | shape | result |
| --- | --- | --- |
| `!bytes.Equal(a, b)` gate in a FuncDecl | identity 3 | **caught** — `TestIdentityGatesAreCensused` *(rev6 survivor, now closed)* |
| `var g = func(...) { if a != b …; if len(a) > 4096 … }` | identity 2 / bound 6 | **caught** — **both** censuses *(rev6 survivor, now closed)* |
| `length := len(v); if length < 1 \|\| length > 4096` | bound 4 | **caught** — `TestBoundGuardsAreCensused` *(rev6 survivor, now closed)* |
| `switch len(values) { case 0: … }` — a bound with no BinaryExpr at all | not in the bound list | **caught** — `TestAllProductionSwitchesAreClassified` (cross-census routing works; the bound header does not mention it, the identity header does) |
| `func (t T) zzCheck(v string) { if len(v) > 4096 }` — method receiver | bound 3 | **caught** |
| `func f() { check := func(v string){ if len(v) > 4096 } }` — nested closure | bound 3 | **caught** |
| `func init() { if len(values) > 4096 }` | bound 3 | **caught** |
| `var v = T{check: func(s string){ if len(s) > 4096 }}` — struct-field literal | bound 6 | **caught** |
| `if !received.zzEqualTo(sent)` — user-defined equality method | outside identity 8 ("stdlib") | **caught** — the method's own `t.raw == other.raw` body reddens the identity census |
| control: the same `!=` gate as a plain FuncDecl | — | **caught** |
| control: the same `len` bound as a plain FuncDecl | — | **caught** |
| `strings.EqualFold(a, b)` gate | identity 8 | **survives — stated** |
| `n := utf8.RuneCountInString(v); if n < 1 \|\| n > 4096` | bound 10, one word out | **survives — stated in substance** |
| `n := len(v) - 1; if n > 4095` | bound 4, one word out | **survives — stated elsewhere in the file** |

The two controls matter: identical code in a plain `FuncDecl` is caught, so a
survivor is a property of the shape, not of the plant being dead code.

**Verdict on G-A.** The enumeration is a hand list, and it is honest. The three
shapes rev6 demonstrated open are closed and I re-verified each; every
claimed-empty shape really is empty; the one set claim holds over its set; and
the nine further shapes I could invent are either derived, routed to a sibling
census, or land on a stated bound. No plant is both unstated and live.

### Carried forward, not charged (for the two following leaves)

Two of the three survivors sit one word outside the shape that means to cover
them, and the enumeration does not say of itself that it is hand-built:

- **bound shape 10** says "a **direct** `cap(...)` or `utf8.RuneCountInString(...)`
  comparison". `n := utf8.RuneCountInString(v); if n > 4096` is the same source
  through a variable and is not "direct"; shape 4's source list is `len` /
  `stringLength` only, so it does not pick it up either.
- **bound shape 4** says "bound from `len(...)` or `stringLength(...)`".
  `n := len(v) - 1` is an arithmetic combination. This *is* written down — in
  `lengthBoundValue`'s doc comment ("a length combined arithmetically before
  binding … binds no length variable") and in the rework evidence ("arithmetic
  combos are out") — just not inside the ten-shape list.
- The constructor header says *"The residue, stated here rather than left
  silent"* and names what would hide from it. The bound and identity headers
  carry "can only be carried by these AST shapes" with no equivalent sentence
  about the list itself.

Both survivors have **zero live instances**, proven above by a scan that does
not use the census. This is placement, not a false claim, and it is not worth an
eighth round on a leaf whose production bytes have not moved since rev4. The
next leaf should add one sentence per header — "this enumeration is a hand list;
a shape outside it is unmeasured" — and widen shape 4's source rule or shape
10's wording when it next touches them.

---

## G-B — the floor is even now

| census | shape space | disposition |
| --- | --- | --- |
| identity (`identity_census_test.go:33`) | **8 shapes** | 7 derived, 1 stated bound (other stdlib predicates), verified empty |
| bound (`bound_census_test.go:35`) | **10 shapes** | 6 derived, 2 stated-pinned, 2 stated bounds, all live counts verified |
| constructor/arm (`inventory_test.go:83`) | **7 shapes** | 7 derived-or-refused, each with a synthetic proof, **plus an explicit out-of-list residue** (frameFault via type alias, non-Go construction, out-of-dir files) |

The **import census** (`import_census_test.go`) has no shape space of its own,
and does not need one: it is shape 6 of the constructor space, it pins exactly
one thing — the `ImportSpec` name binding the identifier-matched censuses rely
on — and it proves that gate over all five spellings (default, identical alias,
renamed, blank, dot) with a zero-importer fail-closed guard. Complete for its
one claim.

All three censuses also gained the thing that keeps a scanner honest: the two
new shapes were added to the **canary lists**
(`decode.go|checkStringBounds|len|length < minimum`,
`context.go|CheckContextEcho|bytes.Equal`), so a scanner that goes blind to
either fails closed instead of reporting a short denominator as complete.

Registry growth is exactly the newly-derived live rows and nothing else:
bound 103 → 107 (+4, the shared character-count enforcements in
`checkStringBounds` and `checkSortedUniqueStrings`, exempt as mechanism because
they compare against parameters, each still naming a driver), identity 99 → 100
(+1, `CheckContextEcho|bytes.Equal`, **not** exempt, driver `TestCheckContextEcho`).
No row was dropped, retitled, or converted to an exemption.

---

## G-C — 451 applied / 434 measured, behavioural kills separated from census-only

Battery regenerated from rev7 production syntax by the same generator as rev6
(`gen.go`): every INT literal argument of every one of the six bound helpers
moved by one in both directions, every `len` comparison literal moved outward,
both operands of every derived identity gate widened to exempt their zero.
Mutant set came out **byte-identical to rev6's** — which is itself a check, since
production is byte-identical. Applied by byte offset, PRESENT-asserted,
byte-restored with a full-content comparison after every row.

| battery | applied | behavioural | census-only | survived | compile-fail |
| --- | ---: | ---: | ---: | ---: | ---: |
| A — helper bound edges, literal −1 | 101 | 89 | 0 | 12 | 0 |
| A — helper bound edges, literal +1 | 118 | 101 | 0 | 17 | 0 |
| C — `len` guard literals moved outward | 9 | 8 | 0 | 1 | 0 |
| D — identity gates, both operands, both spellings | 173 | 94 | 62 | 0 | 17 |
| E — the 17 D rows that cannot compile, re-typed | 17 | 10 | 7 | 0 | — |
| E2 — shape controls for 4 of E | 4 | 0 | 4 | 0 | — |
| GA — rev6 capability/echo probes, re-anchored | 6 | 6 | 0 | 0 | 0 |
| R — the 9 prior survivors the generator does not emit | 9 | 0 | 0 | 9 | 0 |
| P — my 14 enumeration plants and controls | 14 | 0 | 11 | 3 | 0 |
| **total measured** | **434** | **308** | **84** | **42** | — |

**A note on my own instrument, since the same standard applies to it.** My first
pass anchored the six GA probes with a Python text-mode `str.find`, which returns
*character* offsets, while the harness writes at *byte* offsets — the production
comments contain em dashes, so five of the six drifted and came back
`NOT-APPLIED`. An unmeasured row is not a pass. I re-anchored all five on bytes
and re-ran them: **6 of 6 behavioural kills**, including GA2 failing
`TestCheckTargetWriteGatesRefusesNonAvailableEnabled` at the write gate and GA5
failing `TestCheckContextEcho`. Zero unmeasured rows in the table above.

### Survivors: 42 rows, and the composition moved the right way

| | rev6 | rev7 |
| --- | ---: | ---: |
| measured | 428 | 434 |
| behavioural kills | 309 | 308 |
| census-only | 77 | 84 |
| survivor rows | 42 | 42 |
| distinct survivor IDs | 36 | 33 |

- **30 bound survivors — the identical set**, every one on a committed row or a
  declared class-level bound (the `platforms` set behind `scalar.ParsePlatform`,
  the five 65536 digest ceilings with their 65537-element stated bound, the
  `len(name) < 3` reverse-DNS guard, the 8 `maxUint53` maxima refused at parse
  time by `parseUint53Literal`, the `bytesTrimSpace` plumbing guard).
- **9 R survivors — the identical set**, same verified rationale.
- **3 plant survivors — a different three.** rev6's three (bytes.Equal,
  package-literal, len-via-variable) are now census kills; the three that remain
  are mine (EqualFold, rune-count-via-variable, arithmetic-length), all stated
  and all with zero live instances.
- **Resurrections: 0.** Production byte-identical, 0 tests removed, and the
  distinct-survivor set strictly shrank 36 → 33 — the three that left are exactly
  the three shapes rev7 closed.

The 1-kill delta against rev6 is bookkeeping, not regression: rev6's plant set
included the retargeted-switch-arm mutant (a behavioural kill); mine spent that
slot on shape plants instead. The producer ran that exact mutant this round as
P7 and it fails at `TestEveryArmWitnessRefusesAtTheProductionEntry`.

### The producer's own 15-row table, checked rather than accepted

Narrowings, not deletions; every row names its failing test; the three survivors
are the three stated bounds and are labelled as such; the two census co-fires are
disclosed rather than scored as clean singles. My independent run reproduces its
three survivors (S1 plain-param, S2 `cap`, S3 `reflect.DeepEqual` — my EqualFold
plant is the same class as S3) and adds two shapes it did not plant, both stated.
No row in it over-claims.

---

## AC coverage — 7 of 7 driven at the production entry

| AC row | production call site | named driving tests |
| --- | --- | --- |
| discovery | `discovery.go` `Discover` | `TestDiscoverBindsManifestToCandidate`, `TestDiscoverRefusals` |
| manifest | `manifest.go` `DecodeManifest` | `TestDecodeManifestAcceptsFixture`, `TestDecodeManifestValueRules` |
| probe | `probe.go` `DecodeProbe` | `TestDecodeProbeAcceptsFixture`, `TestDecodeProbeClosedRules` |
| closed operations | `operations.go` `CheckRequestBody` / `CheckSuccessBody` | `TestEveryOperationHasContractVectors` + derived-vector suite |
| limits | `context.go` `DecodeResourceLimits` | `TestDecodeResourceLimitsBounds`, `TestResourceLimitsMinimumEdges` |
| idempotency | `context.go` `VerifyRequestDigest`, `CheckContextEcho` | `TestRequestDigestFixpoint`, `TestCheckContextEcho` |
| tuple gates | `tuple.go` `CheckTupleAdmission` | `TestCheckTupleAdmissionRefusals` |

Every named test exists in a committed `_test.go` file (verified by grep, not by
report). All seven acceptance cases are present in `ownership.v0.5.0.json` and
`tracecheck` exits 0 over them.

The AC's crash/idempotency clause is declared vacuous with a stated bound, and
the bound is true: production imports `bytes`, `encoding/json`, `io`, `regexp`,
`strings`, `time`, `unicode/utf8` and two internal packages — **no `os`, no
`net`, no `os/exec`, no filesystem** — and the single `io` use is `io.EOF` in
JSON token checks. The package mutates no durable state, so there is nothing to
prove crash-safe. README keeps the Section 7.8 clause binding at `unevidenced`
and reports `17/428` clauses discharged rather than claiming clause coverage.
No unsupported capability is advertised.

---

## Why accepted rather than a fourth round on this class

rev6 routed this class to a gate rather than an open-ended derivation-widening
round, and offered a bounded ask: extend the derivations *or* write the bound
down, plus qualify the two unqualified sentences. rev7 did all three. The two
sentences now read "A new … **in one of the derived shapes below** … fails the
census" and are true as written. The residue I found is placement inside an
otherwise honest document, on two shapes with zero live instances, one of which
is already written down 300 lines lower in the same file. Rejecting on that
would be the loop rev6 warned about, on a leaf whose production bytes have not
moved in four revisions and whose behavioural evidence is 308 kills over 434
measured mutants.

## Reproduction

```bash
# tree identity, before and after every probe
git read-tree HEAD && git add -A && git write-tree     # 070d5200…

# rev6 -> rev7 delta: 3 test files + LOGBOOK, production byte-identical
git archive 1cb6b935 | tar -x -C /tmp/rev6chk && git apply <rev6 patch>
for f in internal/sessadapter/*.go; do shasum -a256 "$f" /tmp/rev6chk/"$f"; done

# battery: regenerate, shard, run in scratch copies of the module
go run gen.go internal/sessadapter > battery_raw.json     # 401 mutants
python3 harness.py shard.json battery.log                  # per shard, in /tmp/saN

# the enumeration probes (14 plants + 2 controls) and the independent shape scan
python3 harness.py probes.json probes.log
go run /tmp/residue.go internal/sessadapter
```

Logs attached as `TASK-260830-2z3se0_review-battery-rev7-RUN-260906-129580.log`.
