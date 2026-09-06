# TASK-260830-2uowwk — review verdict, CR revision 1 (`story_final`)

**Verdict: changes_requested → `to-dev`.**
**repeat-of: none** (revision 1 of this element; F1 and F5 recur across earlier
leaves of this Story, see "Cross-leaf recurrence").

Reviewed tree: `67eea42b99fdfd5b40ccba311545d49ae50a1010` (CR candidate), base
`157a54cfe4e952a2638b574230967989178e9a05`. Worktree verified byte-identical to
the candidate tree before and after every probe below (`CLAUDE.md` reports a
false diff under content hashing — it is mode `120000` → `AGENTS.md`).

No gate in the new code admits what it must reject. I attacked all eleven
G-A/G-B/G-C surfaces and every one held. What does not hold is a set of five
advertised behaviours that do not reproduce — on a Story whose own AC row is
*no unsupported capability is advertised*. All five fixes are localized and
touch no gate logic.

---

## Blocking findings

### F1 — `capability-claims` advertises live probe verdicts it never reads

`.github/workflows/ci.yml:293-298`, step **"Record live probe verdicts"**.

The step runs `go test ./internal/secconftest -run 'TestSkipReportReadsRecordedVerdicts'`.
That test records two *synthetic* verdicts, asserts `Report()` moved, and restores
the counters in `t.Cleanup` (`internal/secconftest/skip_test.go:238-246`). It
invokes `Require` for no real capability. The line the step then greps is printed
by `TestMain` *after* the restore, so it is `skipped=0 "" failed=0 ""` by
construction on every host.

Reproduced:

| Command | Last `capabilities on` line |
| --- | --- |
| the step verbatim (`-run 'TestSkipReportReadsRecordedVerdicts'`) | `secconftest capabilities on darwin: skipped=0 "" failed=0 ""` |
| full suite (`go test ./internal/secconftest -v`) | `secconftest capabilities on darwin: skipped=0 "" failed=0 ""` |
| **nothing at all** (`-run='^$' -fuzz='^FuzzNoSuchTarget$'`) | `secconftest capabilities on darwin: skipped=0 "" failed=0 ""` |

Three runs, one of which executed zero tests, are indistinguishable. The line
cannot separate "every probe available" from "no probe ran". The step is also
non-gating: `grep ... | tail -n 1` takes `tail`'s status, so it exits 0
unconditionally and can never redden the job.

The frozen leaf already forbids exactly this reading, in `Report()`'s own doc
comment (`internal/secconftest/skip.go`): *"Citing a Report line as host evidence
is only honest for verdicts the suite really recorded; the per-test skip datum is
the `--- SKIP` lines under `go test -v`."* The `fixture-matrix` job does it the
honest way; this job does not.

The claim is then carried into two artifacts:
- `README.md` gate table: "README scanned against **live probe verdicts**";
- `TASK-260830-2uowwk_ci-gates.md` AC row 11: production entry point
  "`cigate.CheckAdvertisements` + **live** `secconftest` probes", evidence
  "probe-counter line in CI".

Neither reproduces. `ProbeState.Available` is computed from a real probe and then
discarded by every consumer in the repository: `ProbeStates()` is called only from
`CapabilityIDs()` (`internal/cigate/claims.go:139`), which returns IDs and drops
availability. Every `CheckAdvertisements` call site — production and test — passes
a forced state (`allUnavailable`/`availableStates`, `claims_test.go:246,261`).

To be explicit about merit: forcing all probes unavailable is *stronger* than
scanning against live verdicts, and host-independent. The design is right. Only
the description is wrong, in three places, and the CI step dressed as evidence
records none.

**Fix:** either make the step run a suite that actually probes and assert
something about the result, or delete it and stop citing a probe-counter line;
correct the README row and AC row 11 to say the gate scans with every probe
forced unavailable.

### F2 — outcome document names a tree that does not contain the work

`TASK-260830-2uowwk_ci-gates.md` line 3: `Candidate tree OID: 7be8309b01f900f0162f1bc525e98ca857fcc853`.

That is exactly `git rev-parse HEAD^{tree}` — the tree *before* this leaf. It
carries **zero** of the ten changed files and no `internal/cigate` entry at all
(`git ls-tree -r 7be8309b -- internal/cigate` → 0 rows; the CR tree has 7). The
document's own numbers are impossible on the tree it names: `cigate 90.5%`
cannot be measured in a tree with no `cigate` package. `git write-tree` returns
`7be8309b` here because `internal/cigate` is untracked, so the index tree omits it.

The record's tree is `67eea42b…`; `git diff 7be8309b 67eea42b` is the entire leaf
delta, 10 files / +1474.

The measurements themselves are sound — I reproduced them below on the correct
tree. The defect is the provenance line, and this is the check the round brief
named as having failed in two earlier leaves of this programme.

**Fix:** `resource update` the document with `67eea42b99fdfd5b40ccba311545d49ae50a1010`.

### F3 — `cigate.FuzzTargets` claims a CI wiring that does not exist

`internal/cigate/targets.go:12`: *"CI iterates this derivation rather than a
retyped copy."* CI does not. `fuzz-smoke` derives from `go test -list 'Fuzz'`
(`ci.yml:243`); `FuzzTargets` appears nowhere in `ci.yml` and has no non-test
caller in the repository. It is reachable only from `TestFuzzTargetsAreDerivedSorted`.

`doc.go:17-18` repeats the claim as one of the package's three stated purposes.

The `-list` derivation is arguably the better one (it reads the compiled binary),
so no gate is missing — but a guard whose doc asserts a production wiring it does
not have is the shape this Story exists to refuse.

**Fix:** either wire `fuzz-smoke` through `FuzzTargets`, or drop the function and
its two doc claims.

---

## Unstated bounds (fix by stating, not by widening)

### F4 — the advertisement scanner only sees backticked IDs

`CheckAdvertisements` matches ``"`"+id+"`"`` (`claims.go:241`). With every probe
forced unavailable, all of these are **admitted**:

| Planted sentence | Findings |
| --- | ---: |
| ``The `fifo` capability is available on Windows.`` | 1 (refused — gate works) |
| `FIFO creation is available and supported on Windows.` | 0 |
| `The fifo capability is available on Windows.` | 0 |
| ``There is no doubt that `fifo` is available on Windows.`` | 0 |
| ``It is not false that `fifo` is supported on Windows.`` | 0 |
| ``Only good news: `fifo` is available on Windows.`` | 0 |

Rows 4–6 are inside the bound `doc.go` does state ("it classifies sentences, not
intent"). Rows 2–3 are not: the backtick requirement appears only in the `Finding`
struct comment, while `doc.go`'s **Stated bounds** paragraph omits it and the
README says "**mentions** resolve to probe, skip, gate, or test context" — which
reads as all mentions. An unbackticked capability advertisement is invisible.

### F5 — `fuzz-smoke` is scoped to one package, described as "derived targets"

The job derives 8 targets from `internal/secconftest` only. Five further real
targets exist outside it — `internal/canonicaljson` (4) and `internal/scalar` (1) —
and are never smoked. `doc.go:17` asserts "fuzz entry points stay in
internal/secconftest", which is false repo-wide. Story-scope coverage is complete
and their seed corpora do run under the `test` job, so this is a bound to state,
not a gap to close in this leaf.

---

## Non-blocking observation

**Verb-drop battery dies by bookkeeping, not behaviour.** `positiveAvailability`
has 5 members; the mutant table ships one drop mutant (N1, `available`). I
censused all five:

| Dropped verb | Suite | Killed by |
| --- | --- | --- |
| `available` | red | count guard **+** `TestFindingCarriesLine` (behavioural) |
| `supported` | red | count guard only |
| `enabled` | red | count guard only |
| `works` | red | count guard only |
| `passes` | red | count guard only |

`claims_test.go:57` `t.Fatalf`s on `len(claims)-1 != len(positiveAvailability)`,
which aborts the parent before any subtest runs — 0 of 5 behavioural subtests
fire under any drop. Every mutant is still killed and the DoD letter is met, and
in the green state all six subtests do drive their verb
(`--- PASS: TestPositiveClaimsAreRefused/{available,supported,enabled,works,passes,capital}`).
Only the kill *mechanism* is thin. Not blocking; worth a line in the table.

---

## What I attacked and could not break

| Probe | Result |
| --- | --- |
| **G-A** empty test selector (`-run TestDoesNotExist`) | `go test` exits **0**; the `--- PASS` grep loop fires → job red. Guard load-bearing. |
| **G-A** empty fuzz derivation (`-list 'ZZZNoSuchFuzz'`) | `grep` 1, `test -s` **1** → job red |
| **G-A** `-fuzz='^FuzzNoSuchTarget$'` | exits **0** ("no fuzz tests to fuzz"); `^fuzz: elapsed` guard exits 1 → job red. Real target: guard exits 0, engine ran. |
| **G-A** `needs` completeness | 10 jobs, 9 non-verdict, **all 9 in `needs`**, none extra |
| **G-A** timeouts | all 10 jobs carry `timeout-minutes` (5–15) |
| **G-A** skip surface | exactly one `if:` in the file, `always()` on `gates-verdict` |
| **G-A** aggregator | drove the `jq` over synthetic states: `success`→admit; `skipped`/`failure`/`cancelled`/`null`→**reject**. Empty `{}` admits, unreachable (`needs` is static and complete). |
| **G-B** version set derived, not listed | `checkReleaseRoots(specpin.ReleaseV050/V043, catalog.ReleaseV050/V043)`; no version literal in `ci.yml` |
| **G-B** alter a historical contract | dropped `2.0.0` from the v0.4.3 Mesh RPC override → `TestVerifyContractPreservationLive` + `TestDerivedSetsAreComplete` **FAIL**, `normative source pin mismatch: v0.4.3 version overrides drift`; 2 of 6 PASS guards fire |
| **G-B** stale generated catalog | drifted `catalog.v0.5.0.json` (`checkpoint_id`→`mutant_id`) → freshness step exits **1** |
| **G-B** token-preserving generator disarm | `//go:generate`→`// go:generate` + drifted input → freshness step exits 0. **Pre-existing at HEAD** (identical step in the old `verify` job), and this leaf's new gate *strengthens* it: with the directive disarmed, a contract-version drift in `catalog_gen.go` is still caught by `VerifyContractPreservation`. Not charged to this CR. |
| **G-C** vocabulary source | `CapabilityIDs`→`ProbeStates`→`secconftest.DefaultCapabilities()`. No second vocabulary. |
| **G-C** make it fail | planted ``The `fifo` capability is available on Windows.`` in the real README → `TestRealREADMECarriesNoPositiveClaim` **FAIL**, reason `positive availability claim for an unavailable capability`; README restored byte-clean |
| **G-D** coverage | no floor, and job name + comment + README all say so. Honest report, not a gate. |
| **G-D** windows | `GOOS=windows go build`/`go vet` only; nothing in the file executes a Windows binary; job name and comment both disclaim execution |
| **G-E** landability | `origin/main` == `157a54c` == CR base; `origin/main` is an ancestor of HEAD; 2 commits ahead |
| **G-E** signatures | `722ed1a`, `7f90e58` both `Good "git" signature for oparin@me.com` |
| **G-E** frozen leaves | `git diff 722ed1a…67eea42b -- internal/secprim` empty; `git diff 7f90e58…67eea42b -- internal/secconftest` empty |
| **G-E** first-push risk | `actions/checkout@v7` and `actions/setup-go@v7` both resolve (unchanged from HEAD); CI green on `main` |

## Suites re-run on the candidate tree (this session, macOS/darwin arm64, go1.25.5)

| Command | Exit | Evidence |
| --- | ---: | --- |
| `go test ./... -v -count=1` | 0 | 22 `^ok`, 0 `^FAIL`, 1352 `--- PASS`, 1 `--- SKIP` (`TestDumpSweepSites`) |
| `go test -race ./... -count=1` | 0 | 22 `^ok`, no `DATA RACE`, 125s |
| `go test ./... -cover -count=1` | 0 | 22 `coverage:` lines, 0 `no test files`, `cigate 90.5%` |
| `go build ./...`, `go vet ./...` | 0 | clean |
| `gofmt -l internal` | — | empty |
| `go test ./internal/cigate -v` | 0 | 20 PASS / 0 FAIL |

`-race`, `-cover` and package coverage match the producer's `verify.log` exactly.

## Cross-leaf recurrence

F2 is the third occurrence in this programme of "outcome document names a tree
the record does not carry". F1 is the second occurrence of "a counter that host
conditions cannot move, cited as host evidence" — the first was fixed in
TASK-260830-7a0s2c by adding `TestMain`, and this leaf re-introduces the same
constancy by narrowing `-run` so no writer runs. If a third arrives, the next
step is a gate, not another revision.
