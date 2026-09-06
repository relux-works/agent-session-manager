# TASK-260830-2z3se0 — reviewer verdict, CR-TASK-260830-2z3se0-4 revision 4

- Verdict: **changes requested** → `to-dev`
- repeat-of: `rev1/F7, rev2/N4, rev3/B4` (B7, fourth round on the zero-value /
  absent-fact class) and `rev2 (unmeasured), rev3/B5` (B8, second round on the
  bound class). Both are past the routing threshold, so both requested changes
  below are **censuses**, not site lists. B6's requested gate was delivered and
  works; it is not repeated.
- Reviewed tree: candidate `1469c6348e064454dd31a5a727e25486ed467b1b` (base
  `1cb6b93585749df4ef4755ce93e486bcde9e3a6b`), `repository_delta=present`, 32 paths.
- Working-tree OID recomputed before the first mutant, after every batch, and after
  the scratch probe file was deleted; unchanged at the end
  (`1469c6348e…`), `git status --short` identical to the handoff (6 modified, 1 untracked).

## Baseline reproduced by this run

| Check | Result |
| --- | --- |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `gofmt -l internal/` | empty |
| `go test ./... -count=1` | 17/17 packages ok |
| `go run ./internal/traceability/cmd/tracecheck` | ok, `acceptance_cases=88`, exit 0 |
| 7 session-adapter acceptance cases → named tests | **7 of 7 resolve to real `func Test…` declarations** |

Every gate the rework note claims is reproduced. The producer's numbers are honest.

## Correction to the brief's premise

The brief warned that "this round touched production in several gates, so
[resurrection] is a live risk." It did not. The rev3→rev4 delta is:

```
LOGBOOK.md                                 |  11 +
internal/sessadapter/bounds_edge_test.go   | 423 +
internal/sessadapter/import_census_test.go | 126 +
internal/sessadapter/inventory_test.go     |  53 +-
internal/sessadapter/zero_absent_test.go   | 196 +
```

**Zero production Go source changed.** The three new CR paths (29 → 32) are
exactly those three new test files. Resurrection was still swept for — 76
previously-killed mutants re-measured, see G-C — but the structural risk the
brief anticipated was not present.

## G-A — B6 is closed, and closed as a census

The fix is an import census derived from production, not a spelling list and not
a fourth site patch. `TestAxerrorImportsAreUnaliased` reads the package directory,
parses every non-test `.go` file, and requires **every** import of the error path
to bind the identifier `axerror`; anything else is a violation. It fails closed on
zero scanned files and on zero importers. There is no alias enumeration anywhere,
so there is no "fifth spelling".

Probed against production, not read:

| Probe | Result |
| --- | --- |
| A3 — second aliased import `axe …/axerror` in protocol.go + `axe.New` additive arm | **KILLED** — `TestAxerrorImportsAreUnaliased` |
| A4 — fresh aliased import in manifest.go (imports no axerror today) + `axe.New` arm | **KILLED** — same |
| A5 — dot import `. …/axerror` + bare `New(...)` arm | **KILLED** — same |
| A9 — alias present with **no** arm at all | **KILLED** — the census fires on the import alone |
| A7 control — unregistered 8th package-level constructor + additive arm | **KILLED** — `TestRefusalConstructorsMatchProduction` |
| A8 control — inline `axerror.New` refusal, no constructor | **KILLED** — same |

**6 of 7. Both rev3 controls still dead — no resurrection in the constructor census.**
B6 is resolved and must not be re-litigated.

The seventh is recorded below as N8: it is a different mechanism, not a fifth spelling.

## G-B — B4 and B5 closed as **sites**, not as classes

Both named survivor lists are 100% closed. Both classes, measured over a
denominator derived from production, are roughly half open.

### B4 — the 13 named sites: 13 of 13. The class: 22 of 48.

The 13 sites the rev3 verdict named all die now:

| Named site | Killed by |
| --- | --- |
| 10 envelope echo gates (protocol.go 278/285/384/391/398/405/474/481/488/495) | `TestEnvelopeEchoEmptyStringRefuses/{request,success,failure}/…` — 10 of 10 |
| `capabilityMapUsable` absent ⇒ usable | `TestCapabilityUsableAbsentIsNotUsable` + `TestCheckTargetWriteGatesAbsentAdapterRefuses` |
| `CapabilityUsable` absent ⇒ usable | `TestCapabilityUsableAbsentIsNotUsable` |
| `CheckDoctorHealthy` exempts `""` | `TestCheckDoctorHealthyRefusesEmptyRequiredName` |
| provider-half absence (control) | `TestCheckTargetWriteGates` |

The class denominator is every `X != Y` refusal comparison in production, taken
by traversal. **48 applied, 22 killed, 25 survived (1 further mutant excluded as
equivalent, 1 unmeasured).**

| Survivor group | Sites | Survived |
| --- | ---: | ---: |
| `DecodeProbe` schema / schema_version echo (probe.go:257, :264) | 2 | 2 |
| `CheckProbe` identity facts (probe.go:392, :399, :406, :413, :420, :427) | 6 | 6 |
| `CheckProbeRequest` expected provider (probe.go:443) | 1 | 1 |
| doctor direction echo, doctor entry status (probe.go:610, :685) | 2 | 2 |
| `sealedMatchesFresh` seal arms (discovery.go:172–177) | 6 | 6 |
| `Discover` manifest provider (discovery.go:98) | 1 | 1 |
| `CheckTupleAdmission` key/status arms (tuple.go:1248, :1255, :1262, :1264, :1273) | 5 | 5 |
| call-context request digest (context.go:198) | 1 | 1 |
| `checkValidateResult` mode echo (operations.go:1091) | 1 | **excluded — equivalent** (the vocabulary gate at :1084 refuses `""` first) |
| capture-plan digest echo (operations.go:810) | 1 | unmeasured (`scalar.Digest` has no in-package zero literal) |

`DecodeProbe`'s schema echo is the headline, because it is the *identical* two
lines as `DecodeManifest`'s schema echo — which **is** pinned — and it is reachable
from raw bytes. Driven at the production entry, baseline vs the one-token narrowing:

```
baseline : DecodeProbe(schema="") -> session_adapter_protocol_error:
           adapter probe schema is not the session adapter probe
mutant   : DecodeProbe(schema="") -> err=<nil>
full package suite under the mutant: ok
```

A probe body that does not name the session adapter probe schema at all is fully
admitted, and nothing reddens.

Why the fix did not reach it: `TestEnvelopeEchoEmptyStringRefuses` derives its rows
from `requestRequired[:2]`, `successRequired[:4]`, `failureRequired[:4]` and pins
`rows != 10`. That is a derivation **inside a hand-picked domain of three
envelopes**. It cannot grow to a fourth decoder, and its own tripwire forbids it.
The `CheckProbe`/`sealedMatchesFresh`/`CheckTupleAdmission` rows are exported gates
taking caller-supplied structs — the same standard the already-fixed
`CheckCallBinding` rows were held to in rev2/N4 — but I did not drive each through
a decode path, so their reachability from bytes is **unknown**, not asserted.

### B5 — every `checkStringBounds` edge: 35 of 35. The bound class: 53 of 101.

The rev3 request was "pin every `checkStringBounds` maximum and every array cap at
the edge", and that is exactly and completely done:

| Family | Applied | Killed | Survived |
| --- | ---: | ---: | ---: |
| `checkStringBounds` maxima (all 18 production sites) | 18 | **18** | 0 |
| `checkStringBounds` minima (17) | 17 | **17** | 0 |
| explicit `len` caps + lower edges (dispositions, argv, events, strategies, fidelity limits, contracts, versions) | 11 | **11** | 0 |
| `MaxFrameBytes` ×3 | 3 | **3** | 0 |
| `checkUint53Bounds` finite maxima (context.go:746, :754) | 2 | **2** | 0 |
| **`requireStringBounds` maxima (operations.go, 13 sites)** | 13 | 0 | **13** |
| **`requireStringBounds` minima (13 sites)** | 13 | 1 | **12** |
| **`requireUint53Bounds` finite maxima (4 sites)** | 4 | 0 | **4** |
| **`checkSortedUniqueStrings` element + count maxima (14)** | 14 | 1 | **13** (2 of them equivalent) |
| **`checkSortedUniqueDigests` count maxima (4)** | 4 | 0 | **4** |
| **reverse-DNS extension-key bounds (decode.go:441, both edges)** | 2 | 0 | **2** |
| Total | **101** | **53** | **48** (46 real) |

`requireStringBounds` is 25 of the 46. It is the wrapper around `checkStringBounds`
declared 100 lines above the sites that were pinned, called from `CheckRequestBody`
and `CheckSuccessBody` — the operation-layer production entries. Driven, baseline vs
`+1`:

```
baseline : CheckSuccessBody(OpResumePlan, cwd_relative=4096 chars) -> nil
           CheckSuccessBody(OpResumePlan, cwd_relative=4097 chars) -> session_adapter_protocol_error:
                operation body string member is outside its bound
mutant   : requireStringBounds(members,"cwd_relative",1,4097) -> 4097 admitted, err=<nil>

baseline : CheckRequestBody(OpResumePlan, expected_target_native_session_id=513) -> refused
mutant   : requireStringBounds(...,1,513) -> 513 admitted, err=<nil>
full package suite under both mutants: ok
```

Two mutants excluded as **equivalent**, not counted as gaps: `platforms` element
max 16→17 and count 4→5 are both unreachable behind `scalar.ParsePlatform` and the
four-name closed set. Five survivors (`created_resource_keys` count and the four
`checkSortedUniqueDigests` counts, all 65536) need a 65537-element body and are
fair **stated-bound** candidates rather than required rows — say so explicitly
rather than leaving them in the denominator silently. The other 41 are cheap: a
513-character string, or the literal number 65537.

## G-C — the traversal, the ratio, and the resurrections

Denominators derived from production by traversal, not from the prior verdict.

| Class | Applied | Killed | Survived |
| --- | ---: | ---: | ---: |
| Vocabulary / member tables widened by one (all 63 census registrations) | 63 | **63** | 0 |
| Obligation loops + capability conjunction + validate applicable triple | 11 | **11** | 0 |
| Import / constructor census probes | 7 | 6 | 1 |
| Zero-value / absent-fact identity gates | 48 | 22 | 26 |
| Numeric / string bound guards, both edges | 101 | 53 | 48 |
| Malformed probe of my own (see below) | 1 | 0 | 1 |
| **Total** | **231** | **155** | **76** |

**155 narrowing mutants killed of 231 applied (67%).** Of the 76 survivors, one is
a malformed probe of mine — `GC-B1 restore defect` planted an arbitrary extra rule
rather than the actual rev3 B1 shape, so its survival measures nothing and is
discarded — and 3 are equivalent mutants named above. **72 real survivors.**

Round 3 was 96 of 131. The ratio moved because the traversal is wider, not because
the work regressed: rev3's 32-mutant bound denominator did not reach
`requireStringBounds`, `requireUint53Bounds`, `checkSortedUniqueStrings`, or
`checkSortedUniqueDigests`, and its 10-mutant echo denominator did not reach any
decoder outside the three protocol envelopes. Where the two traversals overlap,
this round is strictly better.

### Round-3 survivors: 35 of 35 addressed. Resurrections: none.

- All 35 rev3 survivors were re-probed. The 13 B4 sites and the 20 B5 bounds are
  killed; the 2 B6 alias probes are killed by the new import census.
- **Resurrection sweep: 76 previously-killed mutants re-measured, 0 resurrected.**
  All 63 vocabulary tables still die when widened by one member (`63 of 63`); the
  `DoctorRequiredCapabilities` and `CheckTargetWriteGates` obligation loops die on
  every drop (`9 of 9`); `checkValidateResult`'s applicable triple dies on both
  shrinks; both constructor-census controls still die.
- This is expected and confirmed: the round changed no production source.

## Blocking findings

### B7 (blocking) — the zero-value / absent-fact class is closed at the 13 named sites and open at 25 more, including a decoder whose twin is pinned

`repeat-of: rev1/F7 (4 sites, closed), rev2/N4 (5 sites, closed), rev3/B4 (13
sites, closed this round)`. **Fourth consecutive round, same class, new sites, same
shape of fix.** Evidence and the driven `DecodeProbe` proof are in G-B above.

The routing rule's threshold was passed two rounds ago, and B6 shows what clearing
it looks like: the producer built a census and the class closed permanently in one
round. B7 has not had that. `TestEnvelopeEchoEmptyStringRefuses` is a fixed
ten-row domain with a tripwire that forbids growth, so a fifth round of naming
sites produces a fifth fixed table.

### B8 (blocking) — the bound class is closed at `checkStringBounds` and open at 46 guards reached through five other helpers

`repeat-of: rev2 (claimed traversal, class not measured), rev3/B5 (20 sites, closed
this round)`. Evidence and two driven production-entry proofs in G-B above.

`TestStringBoundEdges` is a hand-written 17-row literal with `if len(rows) != 17`.
It pins precisely the rows the previous reviewer listed and, like the echo table,
its own tripwire prevents it from growing. `TestArrayBoundEdges` is four
hand-written subtests over the four caps rev3 named. Neither table can notice that
production has 31 more bound call sites through `requireStringBounds`,
`requireUint53Bounds`, `checkSortedUniqueStrings`, and `checkSortedUniqueDigests` —
all six helpers are enumerable with one grep.

## Non-blocking findings

### N8 — the constructor census still admits a package-level function-value indirection

The import census closed every alias spelling. It does not close a call that never
names `axerror` at the call site:

```go
var newFault = axerror.New                       // import census: green (unaliased)
...
shadow, err := newFault(axerror.Spec{...})       // constructor census: invisible
```

`constructorSitesInFile` matches a `*ast.CallExpr` whose `Fun` is a
`SelectorExpr` on the identifier `axerror`; `pairedFuncLit` returns nil for a
`SelectorExpr` value, so the definition creates no scope and the call creates no
site. Planted as an additive arm in `DecodeRequestFrame`, the full package suite is
green (probe GA-A6). The arm derivation misses it too, so a refusal added this way
ships unwitnessed.

This is **not** a fifth alias spelling and not a repeat of B6 — B6's requested gate
was built and works. It is the same prose-ahead-of-the-check habit as N6: the
census sentence says "every `axerror.New` **site** in production must sit inside one
registered constructor body", and the check reads call sites only. Either widen the
match to any `axerror.New` selector outside a registered body, or narrow the
sentence and declare the indirection as a stated bound. It needs no round of its
own; fold it into whichever round closes B7/B8.

### N9 — the battery is finding-scoped for the third consecutive round

`repeat-of: rev2 requested-change #6, rev3/N7.` The round-4 note reports **36 of 36
killed** with an accurate, honest method bound ("battery traverses the reviewer's 35
survivors + the N6 arm"). Every one of those 36 reproduces, and stating the bound
instead of implying a traversal is the right instinct — I am counting it in the
producer's favour, as rev3 did.

But a declared bound closes a *blind spot*; it does not close a *class*. The
difference between N5's declared blind spots (regexp alternation, `init()`-populated
maps — genuinely hard to see) and this one is that here the unmeasured surface is
36 call sites of six named helpers, enumerable with a single grep, in the same
class as the finding being fixed. 36 of 36 and 155 of 230 describe the same tree.

## What is correct and must not be re-litigated

- **B6 is closed as a census.** Production-derived, no spelling list, fail-closed on
  zero scanned files and zero importers, and it kills the alias, the fresh alias,
  the dot import, and a bare alias with no arm. Exactly the gate that was asked for.
- **All 13 B4 sites and all 35 `checkStringBounds` edges are genuinely pinned**, at
  the production entry, with the killer named per row. Nothing in B7/B8 disputes
  any of it.
- **All 63 vocabulary tables remain content-pinned**, 63 of 63, and the obligation
  loops, capability conjunction, and validate triple are 11 of 11. This is still
  the strongest evidence in the package.
- **Zero resurrections** across 76 re-measured mutants, and the round changed no
  production source.
- **N6 is closed** — the function-local registered-name case reports, pinned by a
  synthetic vector that says it is synthetic.
- **Traceability and README are honest.** `acceptance_cases=88`, all 7
  session-adapter cases resolve to real tests, and the README states plainly that
  the §7.8 clause-level binding stays `unevidenced` and the registry makes no
  clause-level claim. No unsupported capability is advertised.

## Requested changes

Two censuses, modelled on `TestClosedVocabularyTablesAreRegistered` — the one gate
in this package that has held at 63 of 63 across three rounds. Both are the same
shape: derive the denominator from production source, require a row per subject,
fail closed on an unrostered subject.

1. **B8 — a bound census.** Parse the production files, collect every call to
   `checkStringBounds`, `requireStringBounds`, `checkUint53Bounds`,
   `requireUint53Bounds`, `checkSortedUniqueStrings`, `checkSortedUniqueDigests`
   and every explicit `len(...) > N` / `len(...) == 0` guard. Require each site to
   carry an edge row driven at min-1/min/max/max+1 through its production entry, or
   an explicit exemption row saying why. A new bound with no row fails the census.
   The 46 real survivors are listed by site in G-B; the five 65536-count rows are a
   legitimate stated bound if you say so.
2. **B7 — an identity-gate census.** Same shape over every `X != Y` refusal
   comparison and every map-presence default in production. Require a zero/absent
   row driven at the exported entry, or a declared exemption naming the upstream
   guard that makes the zero unreachable — `checkValidateResult`'s mode echo is a
   real example of a legitimate exemption, and `platforms` is the equivalent-mutant
   analogue on the bound side. Replace the `rows != 10` tripwire with the census
   count so the table cannot go stale.
3. **N8** — widen the constructor census to any `axerror.New` selector outside a
   registered body, or narrow its sentence and declare the indirection.
4. **Evidence** — report the next battery as a ratio over the two censuses'
   denominators. Once they exist this is mechanical: the census *is* the
   denominator, which is the whole point of asking for one instead of a table.

After these two censuses exist, a new bound or a new identity gate cannot ship
unrostered, and this class stops consuming rounds. That is why the request is two
gates and not 71 rows.

## Stop-the-line

None. Both findings are ordinary, bounded, mechanical rework inside this package.
The censuses have a working model in the same test suite, the denominators are one
grep each, and no external constraint or human product/architecture/approval
decision is involved.

## Method and integrity

- 231 narrowing mutants applied by exact-string or exact-line replacement, each
  grep-confirmed present on disk, each `go vet`-clean before its verdict, each
  reverted immediately after; 8 further mutants recorded `NOT_APPLIED` with the
  reason (type-zero literals unavailable in the file, or a shifted anchor) and
  excluded from every ratio rather than counted as passes. One applied mutant
  (`GC-B1 restore defect`) was a malformed probe on my side and its survival is
  reported as measuring nothing rather than as a finding.
- 3 survivors re-examined and reclassified as **equivalent mutants** after finding
  the upstream guard that makes them unreachable; they are excluded from the
  finding, not from the denominator.
- One scratch test file (`zz_reviewer_probe_test.go`) drove baseline-vs-mutant
  behaviour at `DecodeProbe`, `CheckRequestBody`, `CheckSuccessBody`, and
  `DecodeManifest`; deleted in the same call.
- Working-tree OID recomputed after every batch and at the end:
  `1469c6348e064454dd31a5a727e25486ed467b1b`, equal to the candidate;
  `git status --short` unchanged.
- `go test ./... -count=1` re-run on the restored tree: 17/17 packages ok.
