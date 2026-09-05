# TASK-260830-32jeti — round-6 review verdict: CHANGES REQUESTED

Reviewer run `RUN-260905-4b1178`. Candidate `CR-TASK-260830-32jeti-3` revision 3,
base `57afcc6d`, candidate tree `aa190969ba39fee5234fbb10bbff2af68441c5a0`.
Tree OID re-derived at the end of the review and matched exactly — every probe
below was planted and reverted; the candidate is unmodified.

**Verdict: changes requested (`to-dev`). Two blocking findings.**

Round 5's single blocking finding (F1r) **is closed** and closed well. What
blocks now is (F1) the residue the census cannot see, and (F2) a
nondeterministic production defect on `Host.Call`'s primary path that this
leaf's own harness does not detect.

---

## G-A — the census, attacked

### The census is genuinely load-bearing for the shapes it scans

Seven narrowing mutants planted **inside** the census itself; five killed.

| # | Narrowing | Result |
| --- | --- | ---: |
| C3 | scan skips `spawn.go` | KILLED (`registered derivation "spawnMembers" names no production vocabulary`) |
| C4 | drop the `ranged[name]` membership disjunct | KILLED |
| C5 | drop the `indexed[name]` disjunct | KILLED |
| C6 | drop the `unknownMember` disjunct | KILLED |
| C7 | `valueShape` stops recognising `map[string]string` | KILLED |
| C9 | plant an unregistered vocabulary in **build-tagged** `runner_windows.go` | KILLED |
| C10 | control: same plant in `runner_unix.go` | KILLED |

Both census directions are real, not decorative: C4–C7 redden through the
*orphaned-registration* loop, which is the direction that would otherwise pass
vacuously on a truncated scan.

**Build-tag question answered affirmatively, by construction rather than by
inheritance.** C9 plants `var bogusCensusVocab = map[string]bool{...}` plus a
package-level index expression into `runner_windows.go`, a file `//go:build
windows` excludes from the darwin build entirely. The census still finds it and
fails. `os.ReadDir` + `parser.ParseFile(…, 0)` ignore build constraints, so the
scan covers files the compiler never sees. C10 is the same plant in a compiled
file and behaves identically.

**Can `deriveMembershipVocabularies` return empty while `registered` is
non-empty?** Confirmed by forcing it — C3 is exactly that shape (one file's
vocabularies removed from `found`, registrations left standing) and the second
loop catches it. The zero-registration floor and the duplicate-registration
tripwire are correct but are not what carries this; the orphan loop is.

### The blind spot is not empty — ratio 19 of 22

Denominator derived **independently of the census**, by a separate AST walk of
every package-level `var`/`const`/`type` in the 15 non-test files
(`.temp/rev3-review/enum_vars.go`), plus a sweep for every `switch`, every
function-local `[]string`/`map[string]bool` literal, every `slices.Contains` /
`sort.Search` call, and every `init()`:

- 27 package-level declarations of the three scanned shapes exist.
- 8 are excluded by design, and **both exclusions verified independently**: each
  of the seven `*Required` lists has exactly one production use and it is a
  `missingMember` argument; `profileProviders` has zero production uses at all.
- 19 remain — the census finds 19 and registers 19. **19/19 for its own shapes.**
- **3 more closed vocabularies exist in production that the census cannot see**,
  all `switch`-shaped, all with real production call sites, all unguarded.
- (A fourth, `required := []string{…}` at `protocol.go:306`, is function-local
  and equally invisible, but belongs to the `*Required` order-list class that is
  excluded by design. Not counted.)

No chained-equality vocabularies, no `slices.Contains`, no `sort.Search`, no
`init()`-built sets, no `map[string]struct{}` — those predicted shapes are
genuinely absent. The three that exist are `switch`/`default`.

### F1 (BLOCKING) — three switch-shaped closed vocabularies admit an extra member with the suite green

Each row: mutant widens the accepted set by one value; **whole `internal/provhost`
package**, no `-run` mask.

| Mutant | Vocabulary | Production call site | Suite |
| --- | --- | --- | ---: |
| C1 | envelope member set (6, §7.2) `protocol.go:328` | `checkResponseMembers` ← `DecodeResponse` `protocol.go:416` | **SURVIVED 12/12** |
| C2 | execution profile (2, §7.7) `profile.go:57` | `ProfileMapping` ← `checkSpawnProfileMapping` ← `DecodeSpawnPlan` `spawn.go:100` | **SURVIVED 12/12** |
| C8 | transaction state (4, §7.5) `status.go:25` | `validStatusState` ← `DecodeStatusOutcome` `status.go:126` | **SURVIVED 22/24** |

C8's two kills in 24 were **not** the status gate — both were
`TestExecRunnerReportsCrashExit`, i.e. finding F2 below. Per the map-iteration
rule I ran every survivor repeatedly rather than once; C8 is a clean survivor
and its occasional "kill" is the flake, which is exactly the coin-flip the rule
exists to catch.

**These are real acceptances, not no-ops.** A probe driving each production
entry point refuses all three on the shipped tree and accepts all three under
the mutant:

```
shipped:  REFUSED provider_protocol_error … unknown member
          REFUSED invalid_config … profile mapping names an unknown profile
          REFUSED integrity_failure … status state is not a registry member
mutant:   ACCEPTED: ProfileMapping(codex, bogus) = "--dangerously-bypass-approvals-and-sandbox"
          ACCEPTED: response envelope carrying "bogus" decoded without refusal
          ACCEPTED: DecodeStatusOutcome state = "bogus"
```

C2 is the sharpest: an unknown profile resolves to the **unrestricted flag**
rather than refusing, and `profile.go:5-10` states the opposite rule in the
file's own words — *"unknown providers and unknown profiles are refused, never
defaulted."* At `spawn.go:100` that widening makes `DecodeSpawnPlan` validate a
launch/resume plan's `profile_mapping` against the yolo flag for a profile the
contract never defined.

Why the existing tests miss all three: each gate **is** witnessed, but only for
one specific rejected value — `"diagnostics"` for the envelope
(`refusal_arm_inventory_test.go:461`), `{"", "YOLO", "yolo ", "unrestricted",
"standard-yolo"}` for the profile (`profile_test.go`). Witnessing a rejected
value proves the arm is reachable; it does not pin the **accepted** set. That is
precisely the round-2 finding class, one level up.

**The claim in the tree is stronger than what ships.**
`closed_vocabulary_census_test.go:18` opens *"This file closes the
closed-vocabulary class by construction."* It closes the class for
package-level `[]string` / `map[string]bool` / `map[string]string`. The comment
enumerates its out-of-scope items carefully (`*Required`, `profileProviders`,
`[]Operation`) but never names `switch`-shaped vocabularies, so the residue
reads as empty when it is three. Either shape counts as a fix: extend the
derivation to `switch` bodies with a refusing `default`, or register these three
explicitly — but the "by construction" sentence must not stand over an unstated
blind spot that has live instances in it.

---

## F2 (BLOCKING) — `ExecRunner` discards a judged result on a stdin `EPIPE`; valid response frames are lost

Found while stabilising C8. Not introduced by round 3 — this is leaf-2 code at
`8fbc052` — but it is this leaf's finding: the conformance harness exists to
exercise `Host.Call`, the defect is on `Host.Call`'s primary path, and the test
that touches it is the one that flakes.

`runner.go:132-134` and `141-143`:

```go
if writeErr != nil {
        return Result{}, writeErr
}
```

This fires **before stdout is ever considered**, whenever the stdin write races
the plugin's exit and takes `EPIPE`. `Host.Call` then reports
`provider_process_failed: runner reported no result`.

Two doc comments in the same file state the opposite rule:

- `runner.go:123-126` — *"A stdin write failure after the process ran is still a
  result when the process produced one… Report write and close failures only
  when no result can be judged; otherwise the frame governs."*
- `runner.go:208-209` — *"a valid frame governs regardless of lateness or exit
  status."*

Measured at the production entry point (`Host{Runner: ExecRunner{}}.Call`),
two independent runs:

| Plugin shape | Correct outcome | Wrong outcome | Rate |
| --- | ---: | ---: | ---: |
| answers a **valid frame**, exits without reading stdin | 390 / 298 accepted | **10 / 2 frames discarded** | 0.7 – 2.5% |
| `cat >/dev/null; exit 3` (the shipped test's own script) | 368 / 294 `plugin exited without a response` | **32 / 6 `runner reported no result`** | 2 – 8% |

The first row is the severe one: a plugin returns a **well-formed, correct,
correlated §7.2 response** and the host throws it away and fails the call. The
second is a diagnostic misclassification of a crash.

Consequences:

- `go test ./internal/provhost/ -count=1` on the **clean candidate tree** failed
  **3 of 30** runs under load (`TestExecRunnerReportsCrashExit/crash_exit`:
  `error = provider_process_failed: … runner reported no result, want detail
  containing "plugin exited without a response"`). A further 20 runs were green
  and `go test ./...` was green 8/8 — the flake is load-sensitive, so a green
  suite is a scheduling outcome, not a property. 3 failures in 50 isolated runs.
- Every single-run KILL verdict in this task's rounds 4–6 evidence carries that
  same false-kill probability. It did not change any conclusion here — I re-ran
  every survivor — but it means single-run mutant results in the chain are
  weaker than they read.
- `LOGBOOK.md` 0926 records *"`go test ./...` 15/15 green"* as a property. It is
  a single observation of a nondeterministic suite. Fix F2 and the claim
  becomes true; until then it should not be stated flatly.

The fix is what the comment already specifies: when `command.ProcessState` is
available, judge the result — a non-empty frame governs, and an empty stdout
with a known exit code is `plugin exited without a response` — and surface
`writeErr` only when nothing can be judged. Whatever the fix, it needs a
negative test that fails when a valid frame is discarded; a plugin that answers
without reading stdin is a shape `ExecRunner.Run`'s own comment at
`runner.go:75-77` already anticipates, so it is in scope by the package's
own reckoning.

---

## G-B — full sweep, no resurrection

All nine round-4 batches replayed against rev3, whole-package runs,
`cp`-aside restore with sha256 verification after every row.

| Battery | Rows | Killed | Survived |
| --- | ---: | ---: | ---: |
| b1 | 15 | 15 | 0 |
| b1b | 20 | 19 | 1 (`Q4b-backgroundNull`) |
| b2 | 21 | 21 | 0 |
| b2r | 11 | 11 | 0 |
| b3 | 20 | 20 | 0 |
| b4 | 25 | 24 | 0 (`K6` NOT_APPLIED, stale path) |
| b5 | 7 | 7 | 0 |
| b6 | 2 | 2 | 0 |
| b7 | 1 | 0 | 1 (`C9b-newArmViaWrapperOnly`) |
| `k6fix` (corrected `K6` path) | 1 | 1 | 0 |
| **total** | **122** | **120** | **2** |

**120 of 122. The two survivors are `Q4b` and `C9b` — the two stated bounds,
name for name. Zero resurrections, zero new survivors.** Identical to round 5.

Round-5 pack replayed too:

- `probe_widen.json` — **12 of 14**. W1–W10 all KILLED, and each of W1–W9 is red
  **three independent ways**: its derivation test, its census subtest, and its
  `TestClosedMemberSetsRefuseExtraMembers` production-entry-point row. That is a
  real closure of F1r, not a one-witness patch. R2/R4 killed. R1/R3 survive —
  the two stated behaviour-preserving bounds.
- `probe_failclosed.json` — **4 of 4** killed (V1 blank window, V2 no code span,
  V3 bad marker, V4 joined-member collision; V4 now dies to the derivation alone
  thanks to the `reflect.DeepEqual` fix).

**Leaf 1 (`TASK-260830-2890sd`, `internal/provider`)** — subtree OID
`308a2f9f2622fbd645298717d9eb03bd0d12483a`, byte-identical to the round-5
measurement. Untouched; no resurrection is structurally possible.

**Leaf 2 (`TASK-260830-qcosxq`, provhost protocol)** — I reconstructed the rev2
tree from its own patch and got `ca45ddf611c18ca58d573e6e13f5238904f109a8`,
exactly the OID the round-5 probe pack records, so the comparison is exact. The
rev2→rev3 delta is **test-only**: `+688` new lines
(`closed_vocabulary_census_test.go`), a 9-line edit, and a LOGBOOK entry. **Zero
production bytes changed.** Leaf 2's 102/112 replay therefore stands unchanged,
and the 122-battery replay above confirms it empirically at the provhost level.

The 9-line edit is a **strengthening**, not a relaxation:
`requireVocabulary` moves from `fmt.Sprintf("%v", …)` string comparison to
`reflect.DeepEqual`, which is what kills V4.

---

## G-C — bounds carried by the tree, not only the board

| Bound | In the tree | Verified |
| --- | --- | --- |
| `*Required` last-entry drops behaviour-preserving | `closed_vocabulary_census_test.go:36-39` | R1/R3 survive; each `*Required` has exactly one production use, a `missingMember` argument — the drop leaves the body refusing |
| `profileProviders` out of scope | same comment block | zero production uses, confirmed by grep |
| new arm reachable only through a wrapper stays blind | `refusal_arm_inventory_test.go:128-133` | `C9b` survives, as recorded |
| census floor is a duplicate tripwire, not load-bearing | census both-direction loops | confirmed — C4–C7 die to the orphan loop, not the floor |
| `provhost` does not import `internal/provider` | — | `go list -deps`: catalog, scalar, canonicaljson, axerror only |
| `provhost` has no importers | — | no `internal/provhost` import outside the package |
| no top-level `cmd/` | — | absent |
| stderr cap, error-detail contents, receipt ownership, §8 cell values, §8.2 roots | `doc.go` "Stated bounds" | present and accurate |

The one bound **missing** from the tree is F1's: the census's blindness to
`switch`-shaped vocabularies, currently covered by a "closes the class by
construction" claim instead.

---

## Gates

| Gate | Result |
| --- | --- |
| `go vet ./...` | exit 0 |
| `gofmt -l internal/` | clean |
| `go test ./... -count=1` | green 8/8 runs |
| `go test ./internal/provhost/ -count=1` | **red 3 of 50 runs** (F2) |
| `go test ./internal/provhost/ -race -count=1` | green |
| coverage | provhost 86.5%, provider 97.0% |
| candidate tree after review | `aa190969…` unchanged |

---

## What closes this round

1. **F1** — bring the three `switch`-shaped vocabularies (`checkResponseMembers`
   envelope, `ProfileMapping` profile, `validStatusState` state) under a
   derivation, or register them explicitly; and correct the
   "closes the class by construction" sentence so the residue is stated. C1, C2
   and C8 must go red.
2. **F2** — stop discarding a judgeable `Result` on a stdin `EPIPE`, so a valid
   frame governs as both doc comments already promise; add a negative test that
   fails when a valid frame is discarded, and restate the LOGBOOK green-suite
   claim once the suite is deterministic.

Everything else in this leaf is in good shape. The census is well-built, F1r is
genuinely closed three ways over, and nothing regressed anywhere in the story.

Replay pack attached as `TASK-260830-32jeti_review-round6-probe-pack.tgz`.
