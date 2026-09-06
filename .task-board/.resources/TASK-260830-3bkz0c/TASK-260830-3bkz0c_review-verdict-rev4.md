# TASK-260830-3bkz0c review verdict — CR rev4 (`story_final`) — ACCEPTED

Reviewer run `RUN-260906-1c6cfb`. Worktree `.temp/STORY-260830-3drr2m/worktree`,
HEAD `82c38378fe79b3e2a3e9fa337c23643b856ba7ee`, tree
`08ad1875394e617b79bafaf2efb239934801b090`.

Every number below is the reviewer's own measurement, re-derived from production
source or produced by a probe this run executed. Where it disagrees with
`TASK-260830-3bkz0c_round2.md`, **this artifact is the authority**.

---

## 0. The `repository_delta=empty` flag is a CR-construction artifact

The CR header says the producer changed no repository file. That is not what
happened.

| Field | Value |
| --- | --- |
| rev4 `base_oid` | `82c3837` |
| rev4 candidate tree | `08ad187` |
| `git rev-parse 82c3837^{tree}` | `08ad187` |
| rev4 patch size | 0 bytes (`sha256:e3b0c442…` = sha256 of the empty string) |
| rev1/rev2/rev3 patches | 68 changed paths each, base `1cb6b93` (`LOGBOOK.md` hunk `f8211d8..e3fb383` = `1cb6b93..1296ecc`) |

The base was taken **at** the delivered commit, so the snapshot diffed the
commit against itself. The round-2 work is real and is commit `82c3837`:

```
git diff --stat 1296ecc 82c3837   →  20 files, +1706 / -435
  internal/sessadapter/decode.go  -134
  internal/dirnode/decode.go      -127
  internal/environ/shape_census_test.go  +1025 (new)
  internal/environ/decode_unit_test.go   +124 (new)
```

This is the known self-diff window, not an empty producer. I reviewed the
substance at `82c3837` and at the whole-story range `1cb6b93..82c3837`
(70 files, +33 996). Accepting an "empty" CR here means accepting a real
20-file delta whose snapshot base was mis-taken — the orchestrator should treat
the rev4 patch resource as unusable and the commit range as the review object.

---

## G-A (blocking) — did delegation close the class, or the witness?

### A1. Seven-shape control plant: **7 of 7 fire** (was 2 of 7 in round 1)

Re-planted all seven round-1 shapes into `internal/provider` (in census scope)
as one production file, then ran the census. Every one produced an
unregistered-copy failure:

| round-1 shape | round-1 census | rev4 census | firing layer |
| --- | --- | --- | --- |
| A `decodeStrictObject` fresh func | FAIL (name) | FAIL | name + shape |
| B1 `parseStrictObject` fresh decoder | pass (HOLE) | FAIL | shape `strict-decoder` |
| B2 `scanForLoneSurrogate` fresh gate | pass (HOLE) | FAIL | shape `surrogate-gate` |
| B3 `measureString` byte measure | pass (HOLE) | FAIL | shape `byte-measure` |
| C1 `var stringLength = byteLength` (+ target) | pass (HOLE) | FAIL ×2 | shape `byte-measure` + `string-measure` |
| C2 `var decodeStrict` closure | pass (HOLE) | FAIL | shape `strict-decoder` |
| C3 `hasLoneSurrogateEscape` method | FAIL (name) | FAIL | name + shape |
| fresh-name env-id grammar copy | pass (HOLE) | FAIL | grammar fresh-name |

The producer's B1 claim reproduces. Plants removed; tree verified.

### A2. Eighth plant: **two bypasses of shapes the census claims to derive**

Planted into `internal/provider` (in scope). Census **fully green** with both
present:

1. **Alias-import strict decoder.** `import stdjson "encoding/json"`, then
   `stdjson.NewDecoder(...)` + `dec.DisallowUnknownFields()` +
   `fmt.Errorf("duplicate member: %w", …)`. A complete strict-decoder copy with
   the duplicate signal. `bodyBuildsJSONDecoder` (`shape_census_test.go:~380`)
   hardcodes `identifier.Name != "json"` on the selector receiver, so any
   import alias walks through. The stated residue names only "a decoder that
   never touches `encoding/json`" — this one touches it.
2. **Two-result byte measure.** `func(s string) (int, error)` returning
   `len(s)`. `returnsSingleInt` requires exactly one `int` result. The stated
   residue names only "a byte measure over a non-string parameter type" — this
   one is over a string parameter.

Neither shape exists in the tree today (no aliased `encoding/json` import in any
production file), so both are **latent**, not live. They are now on the record
as stated bounds.

### A3. Census scope is a directory list, not derived

`censusPackages` (`census_test.go:69`) is a hardcoded 7-entry map over 17
packages under `internal/`. I planted a full set — strict decoder with duplicate
signal, surrogate gate with literal `0xD800`/`0xDC00` bounds, `RuneCountInString`
measure, `len()` byte measure — into `internal/localstore`. **Census fully
green.** A copy landing in a package the list does not mention is not caught.

The bound *is* declared in source (`census_test.go:67-68`: "A production file in
any other package is outside this story's jurisdiction") and it aligns with the
AC's facade scope, so this is a legitimate stated bound — but it is absent from
`round2.md`'s residue paragraph, which lists only texture residue.

### A4. Differential fuzz from a fresh angle: **0 divergence classes**

Two independent passes over the five judges (`environ.DecodeStrictObject`,
`sessadapter.DecodeTuple`, `dirnode.CheckScanRequest`, `provhost.DecodeManifest`,
`canonicaljson.Canonicalize`), classifying each verdict as *stopped at a
frame/string rule* vs *travelled past it*:

| pass | vectors | divergence classes |
| --- | ---: | ---: |
| random, seed **20260907** (not round 1's 20260906), 34-token escape alphabet | 200 000 | **0** |
| directed enumeration, 12-token escape alphabet, depth 5 | 271 452 | **0** |

Directed enumeration was necessary: the round-1 witness is a specific 4-token
sequence whose probability under random draws is ~1e-7 per vector, so a random
pass alone proves little about that shape.

**Instrument validated by control plant.** Restored the pre-fix
`internal/dirnode/decode.go` from `1296ecc` and re-ran the directed pass. It
reported **2 divergence classes**, in both directions:

| class | witness | behaviour |
| --- | --- | --- |
| admit hole (the round-1 finding) | `"\\ud800\udc00"` | dirnode ADMITS (travels to unknown-member); environ, sessadapter, provhost, canonicaljson all REFUSE lone-surrogate |
| over-strict (**not named in round 1**) | `"\\\\\\u"` | dirnode REFUSES lone-surrogate; all four others admit at the frame |

So the fuzzer would have caught the class, and it catches the over-strict
direction the round-1 review only asserted. At the delivered head both classes
are gone across 471 452 vectors. `internal/dirnode/decode.go` restored; tree
verified byte-identical.

**G-A verdict: the class is closed, not the witness.** Two latent census
bypasses and the package-scope bound are recorded above as stated bounds.

---

## G-B (blocking) — F2's real denominator

### Denominator re-derived by AST walk over `internal/environ/*.go` (non-test)

| exit kind | count | producer counted it? |
| --- | ---: | --- |
| `refuse(rule, member)` call sites (`tuple.go` 10, `observation.go` 21) | **31** | yes |
| bool-function `return false` | **58** | yes |
| `&Fault{…}` returns in `DecodeStrictObject` | **11** | **no** |
| bool-function `return <expr>` that can be false (`CheckEnvironmentID`, `CheckSemver`, `isNull`, `rawUint53`) | **4** | no |

The producer's 31 + 58 = 89 both reproduce exactly. **The denominator is still
understated: the 11 frame-decoder refusal exits are excluded, and they are the
exact gate this leaf exists for** (`FaultNotUTF8`, `FaultLoneSurrogate`,
`FaultNotObject` ×7, `FaultDuplicate`, `FaultTrailing`).

The exclusion is self-inconsistent: **five of the battery's own mutants**
(`N1_lowsurrogate`, `M15_framedup`, `M16_frametrailing`, `M17_frameutf8`,
`M18_highmispair`) weaken exits inside those 11, so the reported numerator
counts kills over exits the reported denominator does not contain.

**Corrected denominator: 100 refusal exits** (31 + 58 + 11), or **104** counting
the four expression returns.

### Battery re-run independently

Re-ran all 27 mutants from a **fresh baseline copied from the delivered tree**
(the producer's `/tmp/envbaseline-r2` is stale — it differs from the delivered
files in `frame_agreement_test.go` and `shape_census_test.go`, but **gofmt
alignment whitespace only**, semantically identical, so the original measurement
stands).

```
mutants  1..14   killed 14 of 14
mutants 15..27   killed 13 of 13
```

| metric | value |
| --- | ---: |
| applied | **27 of 27** (zero `NOT-APPLIED`) |
| killed / applied | **27 / 27** |
| exit coverage on the corrected denominator | **27 of 100 (27.0%)** — producer reported 22 of 89 (24.7%) |
| residue with no narrowing mutant | **73 exits** — stated bound |
| census-only kill, reported separately | `T1_rawscan` (token-preserving) |
| tree OID after the full battery | `08ad1875394e617b79bafaf2efb239934801b090` = candidate tree, `git status` clean |

**The mutants are genuinely narrowing, not deletions.** I read the bodies:

- `M5_stringsdup`: `values[i-1] >= values[i]` → `>` — the uniqueness half alone,
  sortedness arm left refusing. Killed by
  `TestCheckSortedUniqueStringsHalves/sorted_duplicated_refuses`.
- `M13_capabsent`: presence loop exempts exactly `directory_discovery`, count
  arm still refuses short maps.
- `M15_framedup`: duplicate still detected, re-keyed instead of refused.
- `M16_frametrailing`: `err != io.EOF` → `err != nil && err != io.EOF`.
- `M17_frameutf8`: `!utf8.Valid(data)` → `len(data) == 0 && !utf8.Valid(data)`.

Kill criterion is sound: the harness requires `exit != 0` **and** the named test
in the failure output, so a mutant cannot read KILLED off the exit code alone.

One harness gap: the `T1` census re-check runs
`-run TestSharedImplementationsAreCensused|TestSharedGrammarsAreOneLanguage`
only — `TestSharedShapesAreLedgered` is not in that gate, so "census green under
T1" is proven for two of the three census layers.

---

## G-C — F1 and the coverage claims

| round-1 F1 symbol | round-1 | rev4 coverage | production callers **now** |
| --- | --- | ---: | --- |
| `CheckUint53Bounds` | 0.0%, 0 callers | 100.0% | **still 0 anywhere** |
| `CheckSortedUniqueStrings` | 0.0%, 0 callers | 100.0% | **still 0 anywhere** |
| `rawUint53` | reachable only via the above | 100.0% | in `environ`, still only via `CheckUint53Bounds` |
| `parseUint53Literal` | reachable only via the above | 88.9% | same |
| `Fault.Error()` | 0.0%, 0 callers | 100.0% | **still 0** — the wrappers read `fault.Detail` / `fault.Member` and never call `Error()` |
| `validCapability` | dead everywhere | — | **deleted** ✓ (the surviving `dirnode.validCapability` is a different function with a live caller at `probe.go:190`) |

- environ coverage **76.1% → 89.8%**, **zero functions at 0.0%** ✓.
- `CheckSortedUniqueStrings`' doc no longer claims coverage it does not have: it
  names `TestCheckSortedUniqueStringsHalves` and states outright "The observation
  battery covers the digest twin, not this function." ✓
- **The uniqueness half is fed a duplicate — on both functions.** `M5_stringsdup`
  (`>=`→`>`) and `M6_stringsorder` both KILLED on `CheckSortedUniqueStrings`;
  `M14a_evidencedup` / `M14b_evidenceorder` both KILLED on the *live* twin
  `CheckSortedUniqueDigests` through `DecodeEnvironmentObservation`. This closes
  the gap this Story hit twice.

**Delegation created exactly one caller.** `grep` for `environ.` across all
production files outside the package returns two call sites, both of the same
symbol:

```
internal/sessadapter/decode.go:48   members, fault := environ.DecodeStrictObject(data)
internal/dirnode/decode.go:49       members, fault := environ.DecodeStrictObject(data)
```

`HasLoneSurrogateEscape` is reached transitively through it — a real production
path. Everything else in the library has no external caller:
`environ.DecodeTuple` and `environ.DecodeEnvironmentObservation` (boundary.md
rows 4 and 5) are driven by the batteries as library entry points but by nothing
in the product. `boundary.md` row 5's "provider-owned member sources (dirnode
`/validate/*`, provider registry) produce it" does not name a call site of the
library — no dirnode file references `environ`.

Consequence for the battery: **5 of the 27 mutants** (`M1`, `M2`, `M4`, `M5`,
`M6`) weaken code no production path reaches. They measure the library, not a
product path. The producer disclosed the unit-test-only coverage in the F1 body
("covered by `decode_unit_test.go`"), though the F1 header "the library has
callers" overstates its own body.

This is residue management, not concealment: the library is the migration
destination, the sibling copies are pinned behaviorally by the agreement
batteries, and the unification residue is tracked in `TASK-260906-33xcnc`.

---

## G-D — the Story, not the leaf

### Sibling leaves modified after their checkpoints

`TASK-260830-2z3se0` and `TASK-260830-ljkj8r` both read `integrating`. Their
accepted CRs described trees at `7c25ae0` and `d5ad5f6`; `82c3837` has since
removed 134 lines from `internal/sessadapter/decode.go` and 127 from
`internal/dirnode/decode.go`.

**This is correct, and here is the precise reading.** `integrationCheckpointed`
records that each leaf's own signed commit exists and verifies — it is not a
claim about the branch tip. The review object for the combined tree is this
`story_final` CR, whose rev1–rev3 patches carried all 68 story paths from
`1cb6b93`. Lifting the freeze and delivering the B2/F1 fix through this leaf,
rather than reopening two checkpointed leaves, put the change where the
whole-story review actually happens. The sibling acceptances should be read as
"this leaf's commit is delivered and signed", not "this tree is what lands"; the
tree that lands is the one judged here.

`TASK-260906-33xcnc` is parented to `STORY-260905-3t31e9`, so this Story has
exactly its three leaves ✓.

### Landing

| check | result |
| --- | --- |
| `origin/main` | `1cb6b93585749df4ef4755ce93e486bcde9e3a6b` — unchanged since round 1 ✓ |
| `git merge-base --is-ancestor origin/main HEAD` | clean — 4 ahead, **0 behind**, fast-forward lands |
| `git verify-commit` on all 4 introduced commits | Good signature, `oparin@me.com`, ECDSA `SHA256:V6JiKG7J29mjsvikcLoSVp0bLa77VTsFy12gnLO81cM` |
| `go test ./... -count=1` | **19 packages, 0 failures** |
| `go vet ./...` | exit 0 |
| `GOOS=windows go vet ./...` | exit 0 — the cross-Story break that surfaced at combination review once before does **not** reproduce |
| `GOOS=linux go vet ./...` | exit 0 |
| `gofmt -l` | clean |
| `go build ./...` | exit 0 |
| DoD checklist | 18 of 18 checked |

**Nothing reddens `main` on first push.** The Story is landable on merit.

---

## Verdict: ACCEPT

The two blocking gates are answered on the merits.

- **G-A**: delegation closed the *class*, not the witness — 7 of 7 round-1
  shapes now fire, and 471 452 vectors from two angles find zero divergence
  across five judges with an instrument proven, by control plant, to reproduce
  the round-1 witness **and** an over-strict class round 1 never named.
- **G-B**: the battery is real — 27 applied, 27 killed, zero NOT-APPLIED,
  re-run by me from a fresh baseline, genuinely narrowing bodies, tree OID
  unchanged after the run.

The residue is reporting precision and tracked architectural staging, not
behaviour. Recorded here as the authoritative numbers rather than sent back for
a third revision: the denominator finding is the second consecutive same-class
finding on F2, and the fix is three sentences in outcome artifacts against a
tree that is green, signed, and fast-forwardable.

**Corrections that supersede `TASK-260830-3bkz0c_round2.md`:**

1. Refusal-exit denominator is **100** (31 `refuse()` + 58 bool `return false` +
   11 `&Fault{}`), not 89. Exit coverage is **27 of 100 (27.0%)**, residue
   **73 exits**. Five of the battery's mutants target exits the reported
   denominator excluded.
2. Of the six round-1 F1 symbols, **one** gained a production caller
   (`DecodeStrictObject`, transitively `HasLoneSurrogateEscape`);
   `CheckUint53Bounds`, `CheckSortedUniqueStrings`, `rawUint53`,
   `parseUint53Literal` and `Fault.Error()` remain production-unreachable and
   are covered by `decode_unit_test.go` alone. `boundary.md` rows 4 and 5 name
   canonical owners, not production call sites.
3. Census stated bounds gain three entries: an **alias-imported**
   `encoding/json` strict decoder, a byte measure returning **`(int, error)`**,
   and any copy in a package outside the seven-entry `censusPackages` list —
   all three planted and confirmed green this run.
4. The rev4 patch resource is a **zero-byte self-diff**; the reviewable delta is
   `1296ecc..82c3837` (round 2) inside `1cb6b93..82c3837` (the Story).

`repeat-of:` F2/round-1 (denominator, second occurrence — recorded, not routed);
F1/round-1 (production callers, partially closed and disclosed).
