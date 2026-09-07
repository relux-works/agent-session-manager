# TASK-260907-9ny4xl review verdict — CR-TASK-260907-9ny4xl-1 rev 1 (`story_final`)

Verdict: **ACCEPTED**. Reviewer run `RUN-260907-c77ada` (not goal-bound).
`repeat-of: none` (round 1).

Every number below was produced by this run in the worktree
`.temp/STORY-260905-3t31e9/worktree`, not read from the producer's outcome.
Where I accepted a producer figure without re-running it, the row says so.

## Provenance (independently re-derived)

| Check | Method | Result |
| --- | --- | --- |
| Candidate tree OID | detached-index `read-tree HEAD` + `add -A` + `write-tree` | `1ed479a0c1228b651f06fa2f9c8a80350a304fe6` — **equals the record** |
| Changed paths | `git diff --name-only e8a2bb8 1ed479a0` vs the CR's 82, both directions | **82 = 82**, `diff` of the sorted lists is empty |
| Scratch files | `git status --porcelain -uall \| grep '^??'` | **0** (my own probes live under gitignored `.temp/`) |
| Leaves 1–5 untouched | `git diff --stat HEAD` | **7 files**, exactly this leaf's delta |
| Base = trunk | `git fetch origin main; git rev-parse origin/main` | `e8a2bb8…` — **equals the CR base** |
| Fast-forward | `git merge-base --is-ancestor origin/main HEAD` | clean; `main..HEAD` = 5, `HEAD..main` = **0** |
| Leaf signatures | `git verify-commit` ×5 | `1d97474 5da63ad 114a056 cd8591d 613cbd7` — all *Good "git" signature for oparin@me.com* |
| Tree after 26 mutants | re-derived after every restore | `1ed479a0…` at every checkpoint, including the last |

## AC coverage — 5 of 5 rows driven, 1 stated bound

| AC row | Production call site | Driving evidence produced by this run |
| --- | --- | --- |
| 1. mechanism claims in ci.yml / doc.go / README / LOGBOOK describe the code | `CheckAdvertisements` — `claims.go:236-239` (`availability[state.ID] = state.Available`), read at `claims.go:255` | `TestAvailabilityFieldDrivesVerdict` kills **N3** (`= false`) and **N4** (`= true`). R2's claim is about bash exit semantics and has **no Go test** — *stated bound*, re-measured by executing the step (below) |
| 2. marker census measures effect, fails closed on a non-classifying marker | `CheckAdvertisements` + `containsAnyFold`/`wordMatches` (`claims.go:100-116`, `261-282`) | `TestEveryNonClaimMarkerClassifiesCorpusSentence`; control plants **C1/C2/C3/C6** all redden; **C5** old-vs-new below |
| 3. sentence-splitter narrowing mutant dies | `splitSentences` — `claims.go:205-219`, reached from `CheckAdvertisements:241` | **N1** and **N2** killed by `TestSameBlockSentencesSplitOnTerminal` **with the census masked out** |
| 4. catalog-freshness step rejects a token-preserving rewrite | CI step `ci.yml:46-50`; pin reads `internal/catalog/catalog.go:183` | **N10/N11** killed by `TestGenerateDirectiveStaysRecognizedForm`; step re-executed (S1) |
| 5. each stated bound names its escape and survives direct attack | see the bound table | R5, R6, backtick, traceability-Contains and leaf-5 bounds 1–3 all attacked directly |

## G-A (blocking) — R3, effect rather than occurrence: **CLOSED**

The old census asked whether a marker *occurred*. The corpus derivation dumped
through the production splitter shows why that measured nothing: of the 22
probe-mentioning README sentences, the two that existed before this leaf
(`README.md:659`, `:993`) are **negation-admitted**, and `:659` alone fires 18
of the 20 markers. Occurrence was fully satisfied by a sentence that needed no
marker at all — markers classified **0 of 2**.

Measured distribution now (production predicates, cross-checked against the
production verdict): **22 mentions — negation-admitted 2, marker-admitted 20,
refused 0**, one sole-classifier per marker.

The new criterion is sound as a necessity test, not just a proxy: for a
non-refused, non-negated mention the positive arm did not fire, so admission
came from `containsAnyFold(negations ∪ markers)`; with `len(markers) == 1` the
one marker is exactly what is holding the sentence admitted.

Control plants — **both directions demanded by the brief redden**:

| Plant | Shape | Old (occurrence) census | New (effect) census |
| --- | --- | --- | --- |
| **C1** `"kernel"` | present in 2 mentions, never sole | **PASS** | **FAIL** |
| **C2** `"absence"` | present only inside an already-negated mention — *the exact R3 shape* | **PASS** | **FAIL** |
| **C3** `"quokka"` | absent from the corpus | FAIL | FAIL |
| **C4** `"driver"` | classifies the **wrong thing** — admits `Ship the \`fifo\` driver tomorrow.` | n/a | **FAIL** (census **and** `TestUnclassifiedMentionIsRefused`) |
| **C6** backticks stripped from every README capability ID | corpus derivation goes empty | n/a | **FAIL** — *"the marker census is vacuous"*, fails closed |

C1 and C2 are the decisive rows: the two classes the old census could not see
are precisely the two the new one catches.

### The other instruments in this leaf's reach

The brief asked whether any of them still counts presence. Answer, honestly:
**two do, and in both cases presence *is* the mechanism, backed by an
independent effect check I executed.**

- `TestGenerateDirectiveStaysRecognizedForm` and the `ci.yml:48` grep count
  occurrences of `^//go:generate`. That is not a proxy: the go tool matches the
  same literal prefix, and I confirmed the effect (S1 — the disarmed form makes
  `go generate` a no-op). The residue is a *command* rewrite (`//go:generate
  true`) which preserves both token and form; the freshness step is fooled by
  it, but `internal/cataloggen/generate_test.go:22-27` compares the committed
  `catalog_gen.go` against a fresh `Generate(metadata, lock)` and reddens —
  observed: *"generated catalog is stale; run go generate ./internal/catalog"*.
  Effect is covered; not a finding.
- the fuzz `^fuzz: elapsed` grep is a presence check whose limit — it does not
  witness the mutation phase — is now the **stated R5 bound**, and I measured it.
- `TestCIWorkflowInvokesTraceabilityGate` (`traceability_test.go:456`) is a pure
  `bytes.Contains` presence gate and **is** defeated. Recorded as pre-existing,
  unfixed, traceability scope — see the bound table.

## G-B (blocking) — R1, R2, R4: **CLOSED**

**R1.** `claims.go` does read `ProbeState.Available` (`236-239`, consumed at
`255`). `ci.yml:314-321` and `LOGBOOK.md:147` now say so and agree with
`doc.go:14-20`; the genuinely discarded thing — the live probe outcome, since
`ProbeStates` feeds only `CapabilityIDs`, which drops availability — is what all
three now describe. `grep` confirms `ProbeStates` has exactly one caller
(`CapabilityIDs`, `claims.go:143`). Pinned by `TestAvailabilityFieldDrivesVerdict`;
N3 and N4 kill it in both directions.

**R2.** I rebuilt the derive step and ran it against a real empty leg
(`secconftest` + `canonicaljson` + fuzzless `axerror`):

| Leg shape | grep in the pipeline | `test -s` | `-eq 13` count pin | step exit |
| --- | --- | --- | --- | --- |
| 3 real packages | — | PASS | PASS | 0 |
| one silently empty leg | **refuses nothing** (loop continues) | **PASS** (12 lines) | **REFUSES** | **1** |
| every leg empty | refuses nothing | REFUSES | not reached | 1 |

The corrected comment at `ci.yml:247-254` is exactly right and the old one was
wrong. Neighbouring **steps** audited for the same mistake — PASS-guard loops
(`ci.yml:64-66`, `278-280`, `330-332`), `grep -c` count guards (`129`, `130`,
`159`, `184`, `232`), gofmt (`91-100`), the target-match diff, the smoke
`elapsed` grep (`294`): every one either fails under `set -e` or is a count
substitution that evaluates to `0` and fails the `test`. **No step carries it.**
One job-level residue is named in the bound table.

**R4.** With the census masks separated:

| Mutant | behavioural mask | census mask | real-README mask |
| --- | --- | --- | --- |
| **N1** paragraph granularity | **FAIL** `TestSameBlockSentencesSplitOnTerminal` | FAIL | pass |
| **N2** dot-only terminals | **FAIL** `TestSameBlockSentencesSplitOnTerminal` | **pass** | pass |

N2 is a pure behavioural kill — the census does not see it at all. R4's
requirement (dies to a *behavioural* test, not folded into a census kill) is met
without ambiguity.

## G-C — R5, R6, and the carried defect

**R5 (bound, re-measured).** All 13 derived targets smoked at `-fuzztime=100x`:
**13/13** print `fuzz: elapsed`; `now fuzzing` appears for **8**. The five that
never reach the mutation phase are exactly the ones named in the README row and
`ci.yml:285-289`: `secconftest/FuzzRedactCorpus` and all four `canonicaljson`
targets. The stated escape (budget lengthening / corpus shrink moves it
silently) is correct.

**R6 (bound ANSWERED, verified independently).** `gh pr checks 36` +
`gh pr view 36`: PR #36 (`STORY-260830-1i3qu7`) merged **2026-09-06T20:12:41Z**;
`Conformance fixtures (ubuntu-latest, internal/secprim)` and
`(ubuntu-latest, internal/secconftest)` **pass in both runs**, macOS legs and all
other jobs green. `LOGBOOK.md:15` restates it as ANSWERED. `grep` finds no
residual "unverified"/"not verified" Linux phrasing in README, ci.yml, doc.go,
LOGBOOK or the probe packages — the bound does not read as a live gap.

**Carried defect — CLOSED, not merely stated.** Reproduced first, then fixed:

| Tree | old step (`go generate` + `git diff --exit-code`) | new step (with the `ci.yml:48` pin) |
| --- | --- | --- |
| clean | 0 | 0 |
| `// go:generate` (token preserved) | **0 — the defect** | **1** |
| `\t//go:generate` (indented, token preserved) | 0 | **1** |
| `//go:generate true` (token *and* form preserved) | 0 | 0 — see residue |

The third row is a residue in the same class, but it is **not exploitable**:
`internal/cataloggen/generate_test.go` reddens on a stale committed catalog
independently of the directive (observed). Recorded, not charged.

## G-D (blocking) — the Story

- **Landable on merit: yes.** `origin/main` is `e8a2bb8` = the CR base, it is an
  ancestor of `HEAD`, the worktree is **0 commits behind trunk**, and all five
  existing leaf commits carry a good signature for `oparin@me.com`. This leaf's
  commit does not exist yet by design (work left uncommitted for the CR).
- **Would anything redden `main` on first push? No evidence of it.** I ran the
  full repository myself: `go test ./... -count=1 -race` → **exit 0, 23 packages
  ok, 0 DATA RACE** (2m09s wall, full repo, not a scoped subset); `go build
  ./...` 0; `go vet ./...` 0; `GOOS=windows go vet ./...` 0; `gofmt -l` empty;
  `tracecheck` 0 (60 contracts / 98 cases); catalog freshness + pin 0;
  capability-claims step verbatim **6/6** PASS guards; contract-preservation step
  verbatim **6/6**; fuzz smoke 13/13. cigate coverage 90.5%.
  Residual platform risk: **none identified** — no file among the 82 changed
  paths contains `runtime.GOOS` or a build constraint, and the fixture-matrix
  packages (`internal/secprim`, `internal/secconftest`) are not in the delta at
  all. The hosted ubuntu legs are the only thing I cannot execute; that is a
  stated bound, and it is the same bound PR #36 already answered for those two
  packages.
- **Leaf-5's three bounds — all still accurate; nothing in this leaf changes
  them.** This leaf touches none of the files involved. Each attacked directly:

| Bound | Attack | Result |
| --- | --- | --- |
| delegation gate does not constrain delegated-call **arguments** | mutated `sessadapter.checkStringBounds` to `environ.CheckStringBounds(raw, minimum, maximum+1)` — shape intact | **bound holds**: `TestCheckHelpersDelegateToEnviron`, `TestDelegationPinsAreLoadBearing`, `TestSharedShapesAreLedgered` all **pass**. Caught in depth by `environ/tuple_agreement_test.go:194` and `sessadapter/probe_test.go:207` |
| rune-measure detector keyed to the utf8 selector only | planted a production `len([]rune(value)) > 64` bound in `sessadapter/decode.go` | **bound holds and is wider than phrased**: `bodyCountsRunes` (`shape_census_test.go:486`) also matches `RuneCount`, but is blind to *any* non-selector spelling; environ census **green**. Caught in depth by `sessadapter` `TestBoundGuardsAreCensused` |
| `TestEveryRefusalSiteIsExercised` fires only on a full package run | ran it under `-run '^TestEveryRefusalSiteIsExercised$'` | **bound holds exactly**: `TestMain` gates `auditRefusalSites()` on `fullPackageTestRun()` (`refusal_site_audit_test.go:70,80`), so any `-run` mask silently skips the audit. CI's `test`/`race`/`coverage` jobs use no mask, so it does fire there |

## G-E — mutation battery: 24 applied, **23 killed / 24 = 95.8%**, 1 survivor

Denominator derived from production, not from the test list: every arm of
`CheckAdvertisements` (2 refusal arms + 2 input refusals), every classifier list
(`positiveAvailability`, `negations`, `nonClaimMarkers` — one member drop each),
the splitter's two dimensions, `wordMatches`' anchoring, the backtick matcher,
the availability map, the `ProbeStates` derivation guard, and the catalog
directive form. Full-package suites per mutant; `cp`-aside/`cp`-back with the
tree OID re-derived after every restore.

| Class | Applied | Killed | Survived |
| --- | ---: | ---: | ---: |
| narrowing | 13 | 13 | 0 |
| arm-deletion | 6 | 5 | **1** |
| census-only (control plants) | 5 | 5 | 0 |
| audit-only | 0 | — | — |
| **total applied** | **24** | **23** | **1** |
| NOT_APPLIED (harness control H1) | — | — | — |
| COMPILE_FAIL (harness control H2) | — | — | — |

`audit-only` is 0 because this CR's delta contains no AST audit; the leaf-1–5
audits are unchanged and green under the full-repo `-race` run. NOT_APPLIED and
COMPILE_FAIL are distinct verdict classes in the harness and were demonstrated
by deliberate controls (H1: non-existent anchor → NOT_APPLIED; H2: arity break →
COMPILE_FAIL) so neither can masquerade as SURVIVED.

| Mutant | Class | Narrows to | Verdict | Killed by |
| --- | --- | --- | --- | --- |
| N1 splitter → one sentence per block | narrowing | negated first sentence covers positive second | KILLED | behavioural (mask-verified) |
| N2 splitter → `.` only | narrowing | `!`/`?` stop splitting | KILLED | **behavioural only** |
| N3 `availability = false` | narrowing | available posture refuses | KILLED | behavioural |
| N4 `availability = true` | narrowing | unavailable posture admits | KILLED | behavioural |
| N5 `wordMatches` drops `\b` | narrowing | substrings classify | KILLED | behavioural + census |
| N6 backtick → bare ID | narrowing | bare prose enters the scanner | KILLED | behavioural + real-README |
| N7 drop marker `"skip"` | narrowing | its note refused unclassified | KILLED | real-README + census |
| N8 drop negation `"only"` | narrowing | only-conditional refused | KILLED | behavioural |
| N9 drop verb `"works"` | narrowing | works-claims admitted | KILLED | behavioural |
| N10 `//go:generate` → `// go:generate` | **narrowing, token-preserving** | directive disarmed, diff clean | KILLED | Go test **and** CI step (exit 1) |
| N11 `//go:generate` indented | **narrowing, token-preserving** | same | KILLED | Go test **and** CI step (exit 1) |
| N12 positive arm drops the negation guard | narrowing | over-strict direction | KILLED | behavioural |
| N13 admission set drops negations | narrowing | markers alone admit | KILLED | behavioural |
| A1 positive arm deleted | arm-deletion | claims slide to unclassified | KILLED | behavioural |
| A2 unclassified arm deleted | arm-deletion | unclassified mentions admitted | KILLED | behavioural |
| A3 empty-probe-set refusal deleted | arm-deletion | gate passes on empty vocabulary | KILLED | behavioural |
| A4 empty-document refusal deleted | arm-deletion | gate passes on nothing scanned | KILLED | behavioural |
| **A5 `ProbeStates` empty-derivation refusal deleted** | arm-deletion | empty capability set treated as nothing to check | **SURVIVED** | — |
| A6 directive removed entirely | arm-deletion | delete-only control for N10/N11 | KILLED | behavioural |
| C1/C2/C3/C4/C6 | census-control | see G-A | KILLED | census (C4 also behavioural) |

**A5 — the one survivor.** Deleting `claims.go:123-125` fails nothing. It is
**pre-existing and outside this CR's delta** (`claims.go` changed only in
comments this leaf), it has no injectable seam —
`secconftest.DefaultCapabilities()` is a fixed constructor — and the same class
is covered downstream by `CheckAdvertisements`' `len(states) == 0` refusal,
which **is** pinned (A3, killed). Recorded as a bound, not charged to this leaf.

## Residues recorded, not charged

1. **`ci.yml:14-15`** — *"No job declares an `if:` condition, so no job can skip
   quietly"*. `gates-verdict` declares `if: always()` at `ci.yml:370`. The
   premise is literally false; the conclusion it draws is true (`always()` cannot
   skip). Present at trunk before either Story, untouched by this delta, and
   outside R2's step-level remit. The precise fix is one clause: *"No gate job
   declares an `if:` condition; `gates-verdict` runs `if: always()` so it cannot
   skip either."* Worth a line in the next Story, not a blocking finding here.
2. **`TestCIWorkflowInvokesTraceabilityGate`** (`traceability_test.go:456`) —
   re-attacked harder than the LOGBOOK records. Not just comment-blinding:
   **deleting the real `Vet` and `Build` steps outright leaves the gate green**,
   because `GOOS=windows go vet ./...` and `GOOS=windows go build ./...` carry
   both substrings. Pre-existing, traceability scope, named in `LOGBOOK.md:19`.
3. **`//go:generate true`** — preserves token and form, fools the freshness step,
   caught by `cataloggen/generate_test.go`. Defense in depth verified.
4. **The census's corpus was extended to satisfy the census.** Twenty README
   bullets were added so each marker sole-classifies. That is a legitimate
   reading of "a marker no sentence needs is dead weight", the producer states
   it as follow-up need #2, and the instrument still catches marker death in
   both directions (C1/C2). Named so the choice is visible, not charged.
5. **`LOGBOOK.md:10` cites `claims.go:235`**, which is exact for the pre-leaf
   tree (verified against `HEAD`) but is `claims.go:238` in the tree this entry
   ships with — this leaf adds three comment lines above it. The 2249
   correction at `LOGBOOK.md:147` cites the same number legitimately, since that
   entry describes the tree as it then was. A ±3 drift in a citation that
   self-evidently names the availability-map assignment; worth a one-character
   fix, not a blocking claim defect.
6. `TestEveryNonClaimMarkerClassifiesCorpusSentence`, `TestAvailabilityFieldDrivesVerdict`
   and `TestSameBlockSentencesSplitOnTerminal` are not in the
   `capability-claims` job's `-run` list (`ci.yml:325`). They execute in CI
   through the unmasked `test`/`race`/`coverage` jobs. Noted, not a gap.

## Gates re-run by this reviewer (not accepted from the producer)

| Command | Exit | Observed |
| --- | ---: | --- |
| `go test ./... -count=1 -race` | 0 | 23 packages ok, 0 DATA RACE, **full repository**, 2m09s |
| `go build ./...` | 0 | clean |
| `go vet ./...` | 0 | clean |
| `GOOS=windows go vet ./...` | 0 | clean |
| `gofmt -l .` | — | empty (excluding gitignored `.temp/`) |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | 60 contracts / 98 acceptance cases |
| catalog freshness step verbatim + pin | 0 | pin count 1, diff clean |
| capability-claims step verbatim | 0 | 6/6 `--- PASS` guards |
| contract-preservation step verbatim | 0 | 6/6 `--- PASS` guards |
| fuzz smoke, all 13 targets | 0 | 13/13 `fuzz: elapsed`, 8 `now fuzzing` |
| `go test ./internal/cigate -cover` | 0 | 90.5% |
| `gh pr checks 36` / `gh pr view 36` | 0 | merged 2026-09-06, ubuntu legs pass ×2 |

Accepted from the producer without re-running: the per-package coverage figures
for `secprim` (94.4%) and `secconftest` (92.6%) — I ran `-cover` only for
`cigate`. Nothing else.

## Verdict

The three blocking gates close. R3's defect is fixed by an instrument that now
measures effect and is control-planted in both failing directions, with the old
census demonstrated to pass exactly the plants the new one catches. R1 and R2
are corrected claims that reproduce under direct measurement. R4 dies to a
behavioural test with the census masked out. The carried `//go:generate` defect
is closed rather than merely stated, and its residue is covered in depth. The
Story is landable on merit, `origin/main` is the CR base, the fast-forward is
clean, and every leaf commit is signed. The single battery survivor is
pre-existing, out of delta, and covered downstream.

`accept_cr(TASK-260907-9ny4xl, revision=1)`.
