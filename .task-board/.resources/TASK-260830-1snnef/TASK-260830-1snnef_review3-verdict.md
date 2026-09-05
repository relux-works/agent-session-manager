# TASK-260830-1snnef review verdict — round 3

**Verdict: changes requested → `to-dev`.** One blocking finding, one bound to
record. Everything else in this leaf and the whole Story re-measures clean.

CR `CR-TASK-260830-1snnef-3` rev 3, base `57afcc6`, candidate tree
`6dc5b00c22a7df4e21fece80a73c26e9651f9925`. A scratch-index `git write-tree`
returned `6dc5b00c…` before the first probe and again after the last of ~160
mutants, every one restored.

Round 2's four findings are closed for real — I killed each with my own mutants,
not by reading the rework note. **The conformance battery went 34/46 → 46/48
killed.** The Story's inherited batteries reproduce element for element with
zero resurrections.

The blocking finding is one I caused. My round-2 verdict told the producer that
C20 proved `legacyForward`'s closed-set property and that the P11 bound was
stale. **That was wrong**: C20's kill is a coin flip on Go map-iteration order,
and the property is not pinned. The producer did exactly what my verdict said,
so the tree now carries a claim that is false. Detail in H1.

---

## G-A — my own battery, replayed

**48 mutants applied, 46 killed, 2 survived** (round 2: 34/46 killed, 12
survived in four clusters). Every mutant preserves every refusal literal, so the
arm inventory cannot kill any of them by construction — only behavioural tests
score.

| Cluster | Round 2 | Round 3 |
| --- | --- | --- |
| G1 attachability (C13, C26–C29, C30) | 1 of 6 killed | **6 of 6** |
| G2 plain-error smuggling (V1–V4, V6, V7) | 2 of 6 killed | **6 of 6** |
| G3 replication table (C40, C40b, C40c, C40d) | 2 of 4 killed | **4 of 4** |
| G4 vocabularies (C31–C33, C43, C44) | 1 of 5 killed | **5 of 5** |
| Everything else (C01–C25 less C20, C34–C39, C40b/c/d, C08b, C13b) | killed | killed |

Survivors: **C20** — retired, it is a nondeterministic mutant, see H1 — and
**C45** (`allowedOperationErrors` widened with a non-catalogued code), which is
the honest unbounded-domain bound, already disclosed in-test. No new holes in
this set.

### Are the derivations over the production domain?

The brief asked whether G1 and G3 derive over production or over a hand-written
list. Both are sound, by different and appropriate mechanisms:

- **G1 derives over production.** `derivedParserVocabulary` parses
  `conformance.go`, finds the parser's single `switch`, and resolves every
  case-list identifier through the file's string const declarations. It fails
  closed on an unresolvable case, a case written as a conversion, zero consts, a
  missing or doubled switch, a missing default, a duplicate member, or an empty
  result — so a vocabulary the derivation cannot read is a red, not a vacuous
  pass. The `len(states) != 8` guard is a hard fatal, not a drifting floor;
  `TestDerivedVocabulariesAreExactlyTheAdmittedSets` pins the same set
  independently in both directions plus size.
- **G3 pins, it does not derive** — and that is the right instrument for a map
  with no production-side generator. `TestReplicationMembersAreExactlyTheClosedTable`
  reads the **live** `replicationMembers` map, checks `want ⊆ live`, `live ⊆ want`,
  and `len == 64`. A bidirectional exact mirror with a size pin is a pin; the
  round-2 failure was a one-directional iteration over a hand-written list, which
  is a sample. C40c (add a member) and C40 (inject one via `init()`) are both red
  there now.

### G2 — the third-walk sweep

The sibling walk is fixed: `TestPlainErrorsLiveOnlyInTheirTwoFunctions` now
scans every `FuncDecl` body (receiver-qualified) **plus** every package-level
var initializer, sharing `isPlainErrorConstruction` with the control path.

| # | Shape | Round 2 | Round 3 |
| --- | --- | --- | --- |
| V1 | CONTROL: `errors.New` in a `FuncDecl` body | RED | **RED** |
| V2 | package-level `var errPlanted = errors.New(…)` | GREEN | **RED** |
| V3 | package-level `var errPlanted = fmt.Errorf(…)` | GREEN | **RED** |
| V4 | package-level `var f = func() error { errors.New(…) }` | GREEN | **RED** |
| V6 | `errors.New` in a `*Registry` method body | — | **RED** |
| V7 | `errors.New` in a new `init()` | — | **RED** |
| V5 | **alias**: `var newPlain = errors.New`, called in a `FuncDecl` | — | **GREEN** |

I swept for a third walk across the story boundary. There is no third *closure*
walk: the other four `file.Decls` loops in `refusal_arm_inventory_test.go` are
declaration-targeted by design — the `Code*` const scan, the `Test`-prefix body
index, the `Is*` predicate resolver, and the `validatePlatforms` funnel — and
each fatals rather than passing vacuously when it cannot resolve its target.
`rejectConstructorAliases` inspects whole files already.

**They do not share a traversal helper.** `deriveRefusalArms` and
`TestPlainErrorsLiveOnlyInTheirTwoFunctions` now hold two copies of the same
`FuncDecl` + `GenDecl`/`ValueSpec` loop. Two walks that agree today is weaker
than one walk both call. Not blocking; worth doing when the file is next touched.

---

## H1 (blocking) — the §4.E forward map is unpinned, and the tree says it is pinned

`legacyForward` (`internal/terminalbackend/conformance.go:952`) is a two-row
production table. Nothing pins its size or contents.

| Mutant | Result |
| --- | --- |
| **L3** — `legacyForward` gains `"screen": "vendor.screen"` (fresh canonical) | **GREEN, 5/5 runs** |
| L4 — gains `"screen": "vendor.term"` | RED, 5/5 |
| L2 — CONTROL, delete the `"conpty"` row | RED |
| C20 — gains `"screen": BuiltinTmux` | **RED 5 / GREEN 7 over 12 runs** |

A third historical escape name translates forward, with `legacy_unreported`
version and generation, and the suite says nothing. L4 is red only incidentally:
`TestLegacyReverseProjectionExistsOnlyForThePair` asserts
`ProjectToLegacy("vendor.term")` is incompatible, so the mutant collides with a
string that test happens to name. Any other canonical walks straight through.
The narrowing direction (L2) is pinned; only widening is open.

`TestHistoricalTranslationMapsOnlyTheImmutablePair` refuses a **10-string
sample** under a closure claim — "everything else is incompatible, never
fallback" — over a bounded, enumerable, two-row table sitting 40 lines away.
That is the family that blocked round 1 (F2) and round 2 (G1/G3/G4), on the
surface the AC names verbatim ("historical translation"), and on the production
entry point the ownership registry's new `terminal-lifecycle-conformance`
acceptance case cites.

### Why my round 2 said otherwise

`ProjectToLegacy` does `for legacy, canonical := range legacyForward`. C20 maps
`"screen"` onto `BuiltinTmux`, which `"tmux"` already owns, so
`ProjectToLegacy(ax.tmux)` returns `"screen"` or `"tmux"` by map-iteration
randomness. I measured one run, got RED, and reported the property pinned. It is
not. Two consequences:

1. **The tree now carries the false claim.** LOGBOOK: "P11 (legacy escape name)
   is no longer declined: C20 … is killed by
   `TestLegacyReverseProjectionExistsOnlyForThePair`, so the closed-set property
   is pinned", and the round-2 entry's "P11 stale bound dropped (C20 killed)".
   Both must go.
2. **C20 must leave the shared battery.** It is nondeterministic; the next
   replay gets a random answer. L3 is its deterministic replacement.

### Fix

Pin `legacyForward` in `internal_pin_test.go` with the exact idiom written for
`replicationMembers` this round — live map, both directions, size — which kills
L3 and L4 and makes C20 unnecessary. While there, assert the map's values are
**distinct**: `ProjectToLegacy`'s doc comment says "deterministically maps", and
that is true only because `legacyForward` happens to be injective today. Nothing
enforces it, and C20's coin flip is what a violation looks like.

Then either restore the P11 bound or, better, delete it as genuinely closed —
but by the pin, not by a flaky mutant.

## H2 (bound to record, not blocking) — plain-error constructor aliases

V5: `var newPlain = errors.New` at package level, `newPlain("smuggled")` inside a
`FuncDecl`. GREEN, while the identical text via a direct `errors.New` call (V1)
is red. `rejectConstructorAliases` exists for exactly this shape on
`mismatchf`/`integrityFailure`; the plain-error half has no equivalent, and the
stated bounds cover only *bespoke error types*, not an alias of a recognised
spelling. Severity is below H1 — a plain error carries no wire code — so: either
extend `rejectConstructorAliases` to `errors.New`/`fmt.Errorf` references
outside direct-call position (a few lines, same file), or add the bound line.

---

## G-B — the whole Story, re-measured

Production files are byte-identical to rev 2 (`conformance.go` `500d55d0…`,
`manifest.go` `97edf447…`, `terminalbackend.go` `f8b93b58…`), so every number
below measures the test delta.

### Leaf 1 — TASK-260830-2xdt8t

The review-2 (60) and review-3 (71) definition sets are **not two batteries**:
review-2 is an exact id-for-id, edit-for-edit subset of review-3. I ran all 71
once and report both slices from it rather than pretending to two runs.

| Slice | Killed | Survived | Not applied | Build error |
| --- | ---: | ---: | ---: | ---: |
| Full review-3 set (71) | 60 | 8 | 2 (N12, T01) | 1 (M32) |
| review-2 subset (60) | 53 | 4 | 2 | 1 |

Survivors reproduce round 2 element for element: **N09, N13, N14, N15, X01,
X02, X04, X10** — round 2's nine minus T01. **Zero resurrections.**

**T01 is not a survivor and never was measured by that harness.** Its edit is
additive (`old` is a prefix of `new`), so the harness's own post-substitution
check "original text still present" always trips. Round 2 listed it as a
survivor while separately verifying it killed by hand; this round the harness
labels it NOT-APPLIED, which is the honest status. I re-applied it by hand
against this leaf's `+57`-line ownership delta: **KILLED, 89 tests red**,
including the self-mint case, file restored byte-identical
(`22bf4a8eed2e6de3…`).

### Leaf 2 — TASK-260830-1zvmw7

| Battery | Killed | Survived | Non-compiling | Anchor bad |
| --- | ---: | ---: | ---: | ---: |
| battery-1 (60) | 46 | **0** | 12 | 2 |
| battery-2 (47) | 47 | **0** | 0 | 0 |

Identical to round 2 (`46 + 12 = 58` applied, 0 survived, 2 missing), with one
restatement: **12 of battery-1's "58 kills" are mutants that do not compile**
("declared and not used"). A non-compiling mutant is not a measurement — the
same correction I had to make to my own G2 method last round. It costs nothing
here: battery-2 re-runs all twelve (M12, M23, M25, M30, M31, M34, M35, M43,
M47, M51, M54, M59) as compiling variants and **kills every one**, so the gates
are proven. The headline should read 46 measured kills + 12 covered by battery 2.

---

## G-C — bounds, citations, tooling

| Item | Result |
| --- | --- |
| F2 citation on `conformanceOperations` | **Now true.** Cites `TestDerivedVocabulariesAreExactlyTheAdmittedSets`, which pins `conformanceOperations` to the derived operation set in both directions plus length. C32 red. |
| Runner restores everything it touches | **Yes.** `probe3.py` snapshots and hash-verifies any extension including `ownership.v0.5.0.json`, removes created files, asserts porcelain + watched hashes. Its JSON-restore self-test passes. I replayed it: **21/21 red, 0 survived, 0 anchor-missing.** |
| No T01 residue in the candidate | Confirmed independently — candidate tree OID unchanged across my ~160 mutants. |
| Round-1 bounds in the tree | Both present, `refusal_arm_inventory_test.go` lines 25–40 (attribution drift with the live `CheckTransition` example; kill inflation with the non-comparability warning), plus the three round-2 bounds (W11 DigestFile function-wide, W13 bespoke error types, `mismatchf` single-literal assertion). |
| Coverage | **90.0%** of statements, as claimed. |

Stale-bound status: **P11 must be restored or replaced by a real pin (H1).**
P13 (`ProjectToLegacy` on an empty ID refuses with a different wire code) stands
as a declined closure.

## Gates I ran myself

| Gate | Result |
| --- | --- |
| `go test ./... -count=1` | **exit 0**, 14/14 packages |
| `go test ./internal/terminalbackend/ ./internal/config/ ./internal/traceability/... -race -count=1` | **exit 0** |
| `go vet ./...` | exit 0 |
| `gofmt -l .` | clean |
| `go run ./internal/traceability/cmd/tracecheck` | exit 0, `acceptance_cases=76`, `bindings=49 … clauses_discharged=17/403` |
| `go test ./internal/terminalbackend/ -cover` | **90.0%** |
| Candidate tree identity, before first probe and after last | `6dc5b00c22a7df4e21fece80a73c26e9651f9925` ✓ |

## Architecture and advertised capability

Unchanged and sound. The harness holds no authority, does no I/O, leaves
capability gating in `manifest.go` and trust in `terminalbackend.go`. I
re-verified the round-1 bound on this tree: the only production importer of
`internal/terminalbackend` outside the package is `internal/config/validation.go`,
and it uses `ParseID` only — **no production caller exists for any conformance
entry point**. That is not concealed: `conformance.go`'s header states §4.B
verbatim ("M0 exposes only an internal semantic interface and conformance
harness … performs no I/O, launches no backend, and holds no authority"), so
"production entry point" here honestly means the harness's own exported surface
and no unsupported capability is advertised. The README addition claims the
harness surfaces and the inventory's both-direction property, both of which hold.

---

## What acceptance needs

1. **H1** — pin `legacyForward` (live map, both directions, size) in
   `internal_pin_test.go`, and assert its canonical values are distinct. Kill L3
   and L4.
2. **H1** — remove the false "closed-set property is pinned" / "P11 stale bound
   dropped (C20 killed)" statements from LOGBOOK; state the pin instead. Retire
   C20 from the shared battery in favour of L3 and note why: it collides on an
   existing canonical and `ProjectToLegacy` iterates a map, so it is a coin flip.
3. **H2** — extend `rejectConstructorAliases` to `errors.New`/`fmt.Errorf`, or
   record the alias bound.
4. Optional, and the stronger answer to the twin question: give
   `deriveRefusalArms` and `TestPlainErrorsLiveOnlyInTheirTwoFunctions` one
   shared declaration walk instead of two copies.

Round 2's four findings are genuinely closed, the whole Story re-measures with
zero resurrections, and every gate is green. H1 is one table, one pin, and one
LOGBOOK correction — and it is my error to hand back, not the producer's.
