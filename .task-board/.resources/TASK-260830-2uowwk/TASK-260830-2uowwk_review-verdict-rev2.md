# TASK-260830-2uowwk — review verdict, CR revision 2 (`story_final`)

**Verdict: accepted → `accept_cr` → `integrating`.**
**repeat-of: none.** All five rev1 blocking findings (F1–F5) and the verb-battery
observation are closed and independently re-measured. The residue below is four
mechanism-claim inaccuracies and one unmeasured region, none of which lets a gate
admit what its stated contract forbids; each is a one-line or one-test fix and is
itemised for a follow-up leaf.

Reviewed tree `8d8231ac0ae2ba557f4c02d0c88b9f905a19821a` (== CR candidate),
base `157a54cfe4e952a2638b574230967989178e9a05`. Tree OID re-derived from the
working tree through a temp `GIT_INDEX_FILE` **before and after** every probe:
both times equal to the record. 50 of the 60 CR paths carry 0 bytes of diff
against `HEAD`; this leaf is the 10 files below.

Environment: darwin/arm64, go1.25.5, in the Story worktree.

---

## G-A — did F1's deletion leave a hole? No.

| Probe | Result |
| --- | --- |
| `grep -c 'Record live probe verdicts' ci.yml` | **0** — step deleted, not reworded |
| `grep -c 'live probe' ci.yml` | **0** (the sole surviving mention is the comment at :298) |
| What now establishes AC row 11 | `capability-claims` job → `TestRealREADMECarriesNoPositiveClaim`, name-guarded by a `^--- PASS:` grep loop over all six selected tests. Ran the step verbatim: exit 0, **6 of 6 PASS guards** matched |
| README + doc.go describe the design | Yes — both say "with every probe forced unavailable", not live scanning |
| Is `ProbeState.Available` still dead? | **No.** `claims.go:235` reads it (`availability[state.ID] = state.Available`); both postures are driven — `TestRealREADMECarriesNoPositiveClaim` (all false) and `TestRealREADMEAdmitsWhenAllAvailable` (all true). What remains discarded is the **live probe outcome**: `ProbeStates()` has no caller but `CapabilityIDs()`, which drops availability. `doc.go` states this correctly; `ci.yml:298` and `LOGBOOK` 2249 do not — see finding R1 |

Gate-defeat probe on the real README (planted, then restored byte-identical):
a backticked flat positive claim is **REFUSED** with reason
`positive availability claim for an unavailable capability`.

## G-B — F3's wiring is real in both directions

`TestListedTargetsMatchDerivation` ties CI's `go test -list` leg to
`cigate.FuzzTargets()`. I mutated both sides of the tie:

| Mutant | Result |
| --- | --- |
| add `FuzzZZReviewProbeUnmapped` to `secconftest/fuzz_test.go`, no `FuzzTargetEntry` | **FAIL** — `listed targets = [… FuzzZZReviewProbeUnmapped], derivation = […]` |
| remove `"FuzzDetectCaseCollision"` from `FuzzTargetEntry`, target still present | **FAIL** — same test, mirrored message |

Both restored byte-identical (`diff -r` against a pre-probe copy of
`internal/secconftest`; `git status` clean).

Derivation scope, measured rather than read: `FuzzTargets()` derives
**secconftest only** (8, from the instrument's own map). `canonicaljson` (4) and
`scalar` (1) own no entry map and are derived per package by CI's own `-list`,
pinned by the total count. Live `-list` returns 8 + 4 + 1 = **13**, matching the
pin. A package CI cannot scan fails closed: `go test ./pkg -list` is a bare
command under `set -e`, so a compile or resolution failure aborts the step.

## G-C — F5's new packages, fail-closed and per-target budget

**Empty inner result.** The `grep '^Fuzz' … | sed … >> file` pipeline takes
`sed`'s status, so `set -e` does **not** fire on a package with zero targets. I
built a three-package replica: package B empty → loop completes, `test -s`
passes, and the run dies at `test "$(wc -l …)" -eq 13` with **exit 1**. So the
step is fail-closed — via the count pin, not via the grep. `ci.yml:240-241`
claims otherwise; see finding R2.

**Per-target budget, and no stdin theft.** Ran the smoke loop verbatim. The
`while read` body runs `go test`, which is the classic shape that eats the
remaining lines of the redirected list. It does not here: **13 of 13 iterations
executed**, 13 per-target logs produced, all 13 carrying `^fuzz: elapsed`,
18s total. Each target gets its own `-fuzztime=100x`.

**Stated-bound-worthy fact (R5).** In 5 of 13 targets the 100x budget is spent
entirely on baseline-corpus coverage and the engine never reaches the mutation
phase (`now fuzzing with N workers` absent): `secconftest/FuzzRedactCorpus`
(103-entry corpus) and all four `canonicaljson` targets (196 entries). The
README's claim for this job — "Every derived target executed its engine, proven
by its elapsed line" — is **accurate** and does not overclaim mutation. Worth
stating, not a defect.

## G-D — verb-drop census: kills are behavioural, bookkeeping is gone

Rev1's `claims_test.go:57` size `Fatalf` aborted the parent before any subtest.
Re-censused every row from scratch:

| Battery | Rows | Subtests that fire under each drop | Killed by its own subtest |
| --- | ---: | ---: | ---: |
| `positiveAvailability` | 5 | **7 of 7** every time | **5 of 5** |
| `negations` | 13 | **14 of 14** every time | **13 of 13**, one-to-one, no cross-cover |

Census direction, which is the half a drop battery cannot prove:

| Mutant | Result |
| --- | --- |
| add gate verb `operational` with no claim row | `battery-covers-gate` **FAIL** — `gate verb "operational" has no claim row` |
| add gate negation `except` with no row | `battery-covers-gate` **FAIL** — `gate negation "except" has no conditional row` |

Under a verb drop the census stays green and only the behavioural row reddens —
the kill mechanism is behaviour, not table size.

## G-E — the Story is landable on merit

| Check | Result |
| --- | --- |
| `origin/main` | `157a54cfe4e952a2638b574230967989178e9a05` — **equals the CR base** |
| fast-forward | `origin/main` is an ancestor of `HEAD`; **2 ahead, 0 behind** |
| signatures | `722ed1a` and `7f90e58` both `Good "git" signature for oparin@me.com` (ECDSA `SHA256:V6JiKG…`). The third leaf commit does not exist yet — this CR is uncommitted working-tree work by design |
| frozen leaves | `git diff 722ed1a -- internal/secprim` → **0 bytes**; `git diff 7f90e58 -- internal/secconftest` → **0 bytes** |
| branch protection on `main` | none (`404 Branch not protected`) — the old `verify` job's disappearance cannot strand a required check |
| action resolution | `actions/checkout@v7` + `actions/setup-go@v7` unchanged from `HEAD` and green on `main` at `157a54cf` today |
| first-run risk | **the workflow triggers on `pull_request`**, so the Story PR is the first hosted execution of all nine new jobs, not `main`. A macOS/Linux divergence reddens the PR, not trunk |
| workflow structure | 10 jobs; `gates-verdict.needs` covers **all 9** non-verdict jobs exactly (no missing, no extra); `timeout-minutes` on 10 of 10; exactly one `if:` in the file (`always()` on `gates-verdict`) |
| aggregator | drove the `jq` over synthetic states: `success`→admit; `skipped`/`failure`/`cancelled`→**reject**. `{}` admits, unreachable because `needs` is static and complete |

**Stated bound (R6): Linux execution is unverified, not inferred.** No container
runtime exists on this host (no docker/colima/podman; lima has no instance), so I
could not execute the ubuntu legs. `internal/secprim` and `internal/secconftest`
have never run on a hosted Linux runner — both leaf commits are unlanded. Static
mitigation only: `open_member_unix.go:84-97` handles both `ELOOP` (Linux) and
`ENOTDIR` (Darwin) for the same O_NOFOLLOW event, and `task-board.config.json`
already carries `GOOS=linux GOARCH=amd64 go build ./...`. I report this as
**unknown**; the PR run resolves it before any trunk push.

## Not charged to this CR — confirmed recorded, not silently fixed

- `//go:generate` → `// go:generate` defeats catalog freshness with the token
  preserved. `internal/catalog` carries **0 bytes** of diff against the CR base,
  and the freshness step is **byte-identical** to `HEAD`'s. Recorded in the
  outcome doc (bound 7) and `LOGBOOK` 2249. ✔
- **Additional inherited defeat, not previously named.**
  `TestCIWorkflowInvokesTraceabilityGate` (`internal/traceability/traceability_test.go`,
  outside this CR's 60 paths) is a `bytes.Contains` gate over `ci.yml`. I moved
  `go vet ./...` from its `run:` step into a YAML comment: the test still
  **PASSES**. It is doubly blind here — `GOOS=windows go vet ./...` contains the
  same substring, so deleting the plain Vet step entirely is invisible to it.
  Pre-existing at `HEAD` (both lines existed there); recording it beside the
  `go:generate` one.

## G-F — battery provenance

Document tree OID == record tree OID == my independent derivation. Producer
reports 30 applied / 0 NOT_APPLIED / 0 COMPILE_FAIL, 28 killed, 2 bounded
survivors, narrowing and arm-deletion tabled separately. **I re-ran 29 of the 30
myself** and every one reproduced:

| Class | Mutants | Result |
| --- | ---: | --- |
| verb drops | 5 | KILLED, each in its own subtest |
| negation drops | 13 | KILLED, each in its own subtest |
| N3 boundary-drop (substring match) | 1 | KILLED — `TestSubstringIsNotAMarker` |
| B1 backtick-widen | 1 | KILLED — `TestUnbacktickedProseIsOutsideTheScanner` + `TestRealREADMECarriesNoPositiveClaim` |
| N5 case-narrow | 1 | KILLED — `/capital` and `/no` |
| N4 version-order-blind | 1 | KILLED — `TestCheckAgreementRefusals/version_order_changed` |
| E1 entry-drop | 1 | KILLED — `TestListedTargetsMatchDerivation` |
| A1–A4, A6 arm deletions | 5 | KILLED |
| A5 target guards | 1 | my deletion variant COMPILE_FAILs (unused import/var); ran **A5c**, dropping the empty-derivation guard → KILLED (`TestSortedTargetsRefusals`) |
| C1 release order swap | 1 | SURVIVED — matches the stated order-is-presentation bound |

Ratio on the production-derived denominator: **28 killed / 30 applied**, the two
survivors bounded. Denominator is the production lists themselves (5 verbs +
13 negations), both matcher narrowings, case folding, version ordering, every
refusal arm — not a test-side inventory.

### Seven reviewer-authored mutants the battery did not contain

| Mutant | Result |
| --- | --- |
| X2 emit only the first finding per sentence | KILLED — `TestMixedAvailabilityFindsOnlyUnavailable` |
| A5c drop the empty-derivation guard | KILLED — `TestSortedTargetsRefusals` |
| **X1 collapse `splitSentences` to paragraph granularity** | **SURVIVED** — see R4 |
| X3 drop `nonClaimMarkers` from the unclassified arm | SURVIVED (stricter direction) — see R3 |
| X4 drop `sort.Strings(mentioned)` | SURVIVED — order/presentation, same class as C2 |
| X5 drop the empty-capability guard in `ProbeStates` | SURVIVED — defensive guard whose trigger is unreachable; unmeasured |

---

## Findings — none blocking; all itemised for a follow-up leaf

### R1 — `ci.yml:298` and `LOGBOOK` 2249 state a field is dead that the gate reads

Both say *"`ProbeState.Available` is consumed by no call site."* `claims.go:235`
consumes it, and it is the whole input to the availability map. The true and
useful statement is `doc.go`'s: *live probe outcomes* are read by no call site.
A maintainer trusting the comment could delete the field and break the gate.
**Fix:** one word — say "live probe outcomes", as `doc.go` already does.

### R2 — `ci.yml:240-241` names the wrong guard as fail-closed

*"An empty derivation fails the grep and the explicit size check alike."* The
grep sits in a pipeline, so its exit status is discarded and `set -e` never
fires — proved with a three-package replica where the empty package sailed past
both the grep and `test -s` and died only at the count pin. The step **is**
fail-closed; the comment misattributes it, and a maintainer "simplifying" away
the count pin would open exactly the silent-empty hole this job exists to close.
**Fix:** name the count pin as the guard.

### R3 — the marker census measures occurrence, not effect: markers classify 0 of 2

Measured on the real README with the production splitter and matchers:

| Datum | Value |
| --- | ---: |
| sentences mentioning a backticked capability ID | **2** |
| refused as a positive claim | 0 |
| admitted by the **negation** arm | **2** |
| admitted by the **marker** arm | **0** |
| refused as unclassified | 0 |

`nonClaimMarkers` — 20 words — classifies nothing, which X3 confirms
independently: removing the entire list from the gate changes no test outcome.
`TestEveryNonClaimMarkerOccursInCorpus` passes only because README line 659 is
one long sentence containing 18 of the 20 marker words, with line 968 supplying
`rejected` and `scan`. Its own comment says it exists so that "a marker that
fires on no real sentence" cannot survive as "an admission hole waiting for
future text" — that is precisely the current state, and the test does not see it.
This also makes the README gate row's *"every backticked capability mention
resolves to probe, skip, gate, or test context"* false as a mechanism claim:
0 of 2 resolve that way; 2 of 2 resolve via negations.
**Fix:** assert the classifying *arm* per mention, not word occurrence; correct
the README cell to say mentions resolve to a **negated or non-claim** context.

### R4 — a narrowing mutant of the sentence splitter survives the whole suite

X1 replaces the sentence-terminal boundary with paragraph granularity. The suite
stays green, yet the behaviour is load-bearing in production: the two-sentence
document *"The `fifo` capability is not available on Linux. The `fifo` capability
is available on Windows."* is **REFUSED** today and would be **ADMITTED** under
X1. `TestFindingCarriesLine` is the only splitter test and it uses two separate
blank-line blocks, so it cannot see the change.
**Fix:** one row pinning that exact two-sentence document.

### R5 / R6 — stated bounds

R5: 5 of 13 fuzz targets never leave baseline-corpus gathering at `-fuzztime=100x`
(measured above). The README does not overclaim; state the bound.
R6: Linux execution unverified on this host (measured absence of any container
runtime, reported as unknown rather than inferred).

### Adjudicated, not re-litigated

Five flat positive claims that also carry an incidental negation token are
admitted: `"The `fifo` capability is available on Windows with no extra setup."`
and the `only` / `without` / `not` / `never` variants (5 of 5 admitted; the same
sentence without the negation token is refused). This is the same class the rev1
verdict examined under F4 rows 4–6 and placed inside `doc.go`'s stated bound
*"it classifies sentences, not intent"*. I reproduce it with more mundane
sentences and record it here; I do not re-open it as a finding.

---

## Suites re-run on the candidate tree, this session

| Command | Exit | Evidence |
| --- | ---: | --- |
| `go test ./... -v -count=1` | 0 | 22 `^ok`, 0 `^FAIL`, **1354 `--- PASS`**, 1 `--- SKIP` (`TestDumpSweepSites`) |
| `go test -race ./... -count=1` | 0 | 22 `^ok`, **0 `DATA RACE`** |
| `go test ./... -cover -count=1` | 0 | 22 `coverage:` lines, 0 `no test files`; **cigate 90.5%**, secprim 94.4%, secconftest 92.6% |
| `go build ./...`, `go vet ./...` | 0 | clean |
| `GOOS=windows go vet ./...`, `GOOS=windows go build ./...` | 0 | clean |
| `gofmt -l internal` | 0 | 339 files scanned, empty list |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | contracts=60, compatibility=55, cases=94 — unchanged |
| `go generate ./internal/catalog` + `git diff --exit-code` | 0 | clean |
| contract-preservation step verbatim | 0 | **6 of 6** `--- PASS` guards matched |
| capability-claims step verbatim | 0 | **6 of 6** `--- PASS` guards matched |
| fuzz target-derivation step verbatim | 0 | **2 of 2** `--- PASS` guards matched |
| fixture matrix, darwin leg `internal/secprim` | 0 | 51 `--- PASS`, 0 `--- SKIP` |
| fixture matrix, darwin leg `internal/secconftest` | 0 | 59 `--- PASS`, 0 `--- SKIP` |
| fuzz-smoke derive + smoke, verbatim | 0 | 13 targets, 13 iterations, 13 `fuzz: elapsed` |

`-race`, `-cover` and per-package coverage match the producer's `verify-rev2.log`.

## Tree integrity

Every mutant was applied to a pre-probe copy and restored by copy-back, never by
`git checkout`. After the final probe: `diff -r` clean on `internal/cigate` and
`internal/secconftest`, `cmp` clean on `README.md` and `.github/workflows/ci.yml`,
`git status --short` back to the exact three `M` + one `??`, and the re-derived
tree OID equal to `8d8231ac0ae2ba557f4c02d0c88b9f905a19821a`.

## AC coverage

**11 of 11 rows driven**, each with a named committed test and a production call
site. I independently drove rows 1–2 (contract gates), 3 (vet/gofmt), 4 (race),
5 (coverage), 6 (fixture matrix darwin legs + fuzz derivation + smoke), 7 and 11
(capability-claims step plus a planted-claim defeat probe), and read rows 8–10
against their named tests in the frozen leaves.
