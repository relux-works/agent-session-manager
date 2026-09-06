# TASK-260830-7a0s2c — CR rev2 review verdict

**Verdict: changes_requested → `to-dev`.**

**repeat-of: rev1/C1 (finalize gate) and rev1/C2 (skip report).** Both blocking
findings below are the round-1 findings one step over, not new classes. Two
consecutive same-class findings: the next step is a gate on this leaf, not a
third revision handed back the same way.

- Change Request `CR-TASK-260830-7a0s2c-2` revision 2, base `722ed1a`,
  candidate tree `3c8eb31ca785c53bc613b4df2cda19c8bb6a1939`.
- Reviewer run `RUN-260906-27f892`. All probing done in `/tmp/axrev2`, a byte
  copy of the candidate tree. The live worktree tree OID was verified equal to
  the candidate before and after every probe and is unmodified.

## What round 2 genuinely closed (verified by attack, not by reading)

| Round-1 finding | Probe I ran | Result |
| --- | --- | --- |
| C3 — 1 of 4 probes wired, fifo probes nothing | faulted each of the four probe **attempts** and observed the gated test | **fully closed**, see G-B below |
| C1 — rollback after a *clean* commit permitted | narrowed the finalize gate three ways instead of deleting it | **closed for the clean path**: M14 (`finalized && len(id) > 8`), M15 (`finalized` set only for an empty body), M19 (without-prepare narrowed) all redden `TestTransactorRefusesRollbackPastFinalize` |
| C4 — 6 unbounded survivors | re-derived independently, 54 mutants | the six named survivors are all killed now: N1/N2/M22 (digest components), M30 (U+FEFF), M31 (Gate rename), P1 (recordVerdict), M18 (commit-without-prepare) |
| bound — env/redact fuzz assert only no-op properties | read the added poles, ran D3/D4 | strengthened **and** stated; `FuzzIsEnvName` now dies to an admit-all driver (D3) |
| bound — crash vocabulary unpinned | M12: `Classify` admits `CR-MAT-01` | killed by `TestCrashPointVocabularyIsPinned` |
| bound — an armed point never reached | P2: `RemainingArmed` constant-empty | killed by `TestInjectorArmedPointIsConsumedByTheDrive` |
| AC row 6 "1 of 4 probes" / row 1 "1 of 3 digest components" | measured both | **both numbers are now true**: 4 of 4 probes wired, 3 of 3 digest components pinned |

The corrected numbers in the outcome document are real. That part of the round
is good work.

---

## BLOCKING

### C1 — the finalize gate has a two-line bypass: crash the commit and roll back

`Rollback` refuses after a *clean* `Commit`. It does not refuse after a
**crashed** one, and the crash point in question fires **after** the durable
write:

```
Prepare("x", "BODY")
inj.Arm(PointCommitApply)          // classified recoverable_parked_state
Commit("x")   -> Fault: CR-MAT-commit-apply requires recoverable_parked_state
CommittedBytes("x") -> "BODY"      // the durable rename already happened
Rollback("x") -> <nil>             // admitted
CommittedBytes("x") -> ENOENT      // x.committed deleted
```

`Commit` writes `x.committed`, removes the staged file, and only *then* calls
`MaybeFail(PointCommitApply)`; the fault returns before
`transactor.finalized[operationID] = true`. So the gate is present, correct, and
reachable around.

This is not a hypothetical ordering. The package's own
`TestTransactorCrashOutcomes` asserts, for exactly this point, that the durable
bytes survive the parked fault ("parked commit lost the durable bytes"). The
model therefore treats those bytes as committed — and then lets `Rollback`
delete them and return `nil`.

The spec text `crash.go` cites for this rule, SPEC.md:11964 (`AC-CLONE-005`),
reads: *"rollback remains available through finalization intent and is
forbidden after Provider commit; all bundle evidence survives."* The bypass
walks through both halves of that clause.

Nothing covers it: no test drives rollback after a crashed commit, `doc.go`
states no bound for it, and the README claim is scoped to "a clean `Commit`" so
it is not false — it is simply silent about the neighbour where the same
behaviour round 1 rejected still lives.

**Either rule is defensible** — parked means the operator rolls back, or the
durable rename means rollback is forbidden — but the leaf currently picks
neither. Pick one, drive it through `Prepare`/`Commit`/`Rollback`, and if the
answer is "rollback is the recovery from parked", say so where the AC-CLONE-005
claim is made, because as written the two sentences contradict each other.

### C1b — the other three neighbours are undecided too

Driven through the public API in a throwaway probe (removed afterwards):

| Sequence | Actual result | Covered? |
| --- | --- | --- |
| `Rollback` twice, pre-commit | `nil`, `nil` | no test |
| `Commit` twice, cleanly | 2nd = raw `open …/z.staged: no such file or directory`, zero-valued `Receipt` | no test |
| `Commit` after `Rollback` | raw `open …/w.staged: no such file or directory` | no test |
| `Prepare` after a clean `Commit`, identical body | **admitted**, `err=nil`, receipt returned with `Outcome: explicit_rollback` for a finalized operation | no test |

Two of these refuse only by inheriting a filesystem `ENOENT` — the exact shape
`TestTransactorCommitRequiresPrepare` declares unacceptable for its own case
("the refusal must name the missing prepare, not merely inherit a filesystem
error"). The standard the package sets for itself is not applied one function
over.

Related, from the battery: **P7 survives** — `Injector.Arm` can silently drop
`PointRollbackEnter` with the suite green. The fifth crash point is classified
and vocabulary-pinned but is never armed anywhere, so 4 of 5 points are driven
through the `Transactor` and the rollback-enter fault path has no drive test at
all. The `RemainingArmed` consumption bound ("every arm the suite sets") is
literally true and does not reach this point, because the suite sets no such arm.

### C2 — the LOGBOOK still ships the citation this round was sent to delete

`LOGBOOK.md` line 22 — a line **added by this Change Request** — reads:

> EVIDENCE: outcome `TASK-260830-7a0s2c_secconftest.md` (… 10-row mutant table:
> 4 narrowing + 4 arm-deletion + 2 census-only, **10/10 killed, 0 survivors**;
> … **darwin skip report skipped=0**; digest `5569f34caab734eb` x3).

Both numbers were disproven in round 1 (re-derived 14 of 20 killed; the
`skipped=0` is a structural constant, not a measurement), and the producer's own
round-2 document supersedes them with 29 applied / 2 survivors. The round-2
LOGBOOK entry above it says *"The round-1 `skipped=0` citation was a
constant"* — but the disproven entry lands in `main` intact, with its numbers,
in the same commit. A reader of `LOGBOOK.md` gets both. The brief asked to
confirm the fake citation is gone from the outcome document, the README **and
the LOGBOOK**; it is gone from the first two and present in the third.

### C2b — the report mechanism is still structurally unable to witness a real verdict

I ran both round-1 proofs again on the delivered tree.

| Probe | Gated test | Printed report line |
| --- | --- | --- |
| strict `symlink` probe forced false (attempt redirected into an absent parent) | `--- FAIL: TestHostileSymlinkEscapeRefusedAtCommit`, exit 1 | `skipped=1 "self-test-skip-probe" failed=1 "self-test-fail-probe"` — the *synthetic* verdicts only; `symlink` absent |
| non-strict `mode-bits` probe forced false (`chmod 000` → `chmod 600`) | `--- SKIP: TestSkipModeBitsAndNonRootDenialsHold`, exit 0, verbose SKIP count 0 → 1 | identical line; `mode-bits` absent |

Cause, straight from the verbose log: `TestSkipReportReadsRecordedVerdicts` is
non-parallel and prints at log line 90, **before the first `=== CONT` of any
parallel test**. All four `Require` call sites are inside `t.Parallel()` tests.
The printed report therefore cannot carry a real capability verdict under any
host condition — the round-1 structural finding is unchanged; only the citation
moved.

Two more things follow from the battery:

- **M04 and M05 both survive.** Delete `recordVerdict(verdict)` from `Require`'s
  fail arm, or from its skip arm, and the suite stays green. The round-2 test
  calls `recordVerdict` directly, so it pins the helper (P1 reproduces the
  producer's A4 kill) and leaves both production call sites unmeasured. The
  whole `recordVerdict` / `Report` / `SkippedIDs` / `LogReport` mechanism can be
  disconnected from production without a single test noticing.
- Two statements about it are still not true. The outcome document's S1 bound
  says the denial test *"emits a `--- SKIP` line and `recordVerdict` logs it"* —
  `recordVerdict` logs nothing, and nothing prints it afterwards. The same
  document's Windows/root paragraph offers *"the SKIP line plus the recorded
  verdict is the evidence in both lanes"* — the recorded verdict is unobservable
  in both lanes.

The honest reading is that the `--- SKIP` datum is the *only* skip evidence and
the counter mechanism witnesses nothing. That is a defensible thing to ship
(and `go test ./... -count=1 -v` in the configured validation matrix does emit
the lines) — but then say it plainly, or wire a reader that runs after the
parallel tests (`TestMain`, or a non-parallel final test) so the numbers mean
something.

---

## G-by-G

**G-A — not met.** The `--- SKIP` datum is genuine and reproduces (0 baseline,
1 under a forced non-strict false, exit 0). The fake citation is gone from the
README, `doc.go` and the round-2 outcome document. It is **not** gone from the
delivered `LOGBOOK.md` (C2). The report still prints only synthetic verdicts
under both forced-probe conditions (C2b).

**G-B — met in full.** Four probes, four attacks, all four move:

| Probe | Attempt faulted | Gated test | Strict here? | Result |
| --- | --- | --- | --- | --- |
| `symlink` | symlink into an absent parent | `TestHostileSymlinkEscapeRefusedAtCommit` | yes | `--- FAIL`, exit 1 |
| `fifo` | `mkfifo` into an absent parent | `TestSkipFifoFixtureExchangesBytes` (+ the honesty test) | yes | `--- FAIL`, exit 1 |
| `mode-bits` | `chmod 000` → `chmod 600` | `TestSkipModeBitsAndNonRootDenialsHold` | no | `--- SKIP`, exit 0 |
| `nonroot` | `Geteuid() == 0` → `>= 0` | `TestSkipModeBitsAndNonRootDenialsHold` | no | `--- SKIP`, exit 0 |

None of the four returns a constant. Each gates a test that is **not** vacuous:
defeating the body while leaving the probe intact reddens all three bodies
(V1 open the safe member instead of the escape → FAIL; V2 build a regular file
and relax the probe's `ModeNamedPipe` check → FAIL; V3 neuter the test's own
`chmod 000` → FAIL). The README and outcome-document capability claims match
exactly what the probes establish, and no capability is advertised.

**G-C — not met.** See C1 / C1b. The clean-commit gate is real and
narrowing-pinned; every neighbour is undecided. The test name does now match
what it drives.

**G-D — bounds honest, denominator larger than reported.**

| | applied | NOT_APPLIED | COMPILE_FAIL | VET_FAIL | KILLED | SURVIVED |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| narrowing | 44 | 0 | 0 | 1 | 36 | 8 |
| arm-deletion | 2 | 0 | 0 | 0 | 2 | 0 |
| census-only | 5 | 0 | 0 | 0 | 5 | 0 |
| body-vacuity | 3 | 0 | 0 | 0 | 3 | 0 |
| **total** | **54** | **0** | **0** | **1** | **44** | **10** |

The one VET_FAIL (M03, `goos == runtime.GOOS && goos == "plan9"` → vet's
"suspect and") was repaired to a vet-clean equivalent (M03b) and re-run: killed.
It is reported as its own row, not folded into the kills.

Both producer bounds were **attacked, not accepted**:

- **N16 (driver delegation).** I ran the delegation census across all eight
  entries rather than the one. 7 of 8 are mutant-pinned; only `DriveRedact`
  survives, because the redact fuzz properties call `secprim.Redact` directly.
  The bound as written ("driver delegation is read-verified, not mutant-pinned")
  is honest and in fact conservative — it should read *one of eight*.
- **S1 (survive-as-skip).** Reproduced twice, independently, on both non-strict
  probes. It is a real construction-level survivor, not a rationale: the suite
  is green, the `--- SKIP` line is present, and the verbose skip count moves
  0 → 1. The bound holds.

**Seven survivors carry no bound**, none of them gate-critical on their own but
all of them residue:

| Mutant | What silently becomes admissible |
| --- | --- |
| M04 | `Require`'s fail arm stops recording verdicts |
| M05 | `Require`'s skip arm stops recording verdicts |
| M08 | `SymlinkCapability` drops darwin from `StrictGOOS` — an unexpected symlink absence would skip instead of failing (`fifo`'s strictness *is* pinned) |
| P7 | `Injector.Arm` silently drops `PointRollbackEnter` |
| M27 | `QuoteInSection`'s empty-quote guard narrowed to `quote == ""`; a whitespace-only quote is then refused only downstream, by accident |
| M02 | `decide` drops the empty-ID disjunct |
| M16 | `Prepare`'s empty-operation-ID refusal |

**G-E — three of four bounds are present and honest; the fourth is thin.**
The fuzz poles are strengthened *and* stated. The crash vocabulary is pinned
(`CR-MAT-01`, `CR-MAT-08`, `CR-MAT-09`, `CR-MAT-prepare` all refused; M12
killed) and the instrument-local/registry distinction is stated. The AC numbers
are corrected and true — I re-measured both: 4 of 4 probes wired, 3 of 3 digest
components pinned (N1, N2, M22 all killed). The armed-point bound is stated and
`RemainingArmed` is pinned (P2), but it does not reach `PointRollbackEnter`,
which no test ever arms.

**G-F — provenance clean.** Live tree `3c8eb31c…` equals the candidate, verified
before probing and again after the 54-mutant battery. `git diff 722ed1a --
internal/secprim` is empty and `internal/secprim` has no untracked additions.
The outcome document names the same base commit `722ed1a`. Its per-file
"fingerprint" `5d64ba3b…` did **not** reproduce under four plausible readings of
"sha256 of the sorted per-file sha256 manifest" (hash+name lines, sorted hashes
with and without newlines, concatenated manifest) over the correct 15 files — an
unverifiable citation, superseded by the tree OID, worth either dropping or
shipping with its exact recipe.

## Gates I reran myself

- `go test ./... -count=1` in the worktree → **exit 0, 21 packages ok**; tree
  OID unchanged afterwards.
- `go test ./internal/secconftest -v -count=1` → exit 0, **233 `--- PASS`,
  0 `--- SKIP`, 0 `--- FAIL`**; runner digest `5569f34caab734eb`. This
  reproduces the README's "0 on darwin/arm64 non-root" claim exactly.
- `go vet ./...` → exit 0. `gofmt -l internal/` → clean.
- Accepted from attached evidence without rerunning: `-race`, `-cover`,
  `GOOS=windows` vet/build, `tracecheck`, the 8 fuzz `-fuzztime=100x` runs.

## What the next round needs

1. Decide the crashed-commit rule, drive it, and state it. Same for the
   `Rollback`/`Commit`/`Prepare` neighbours — three of the four currently
   refuse by inheriting an `ENOENT` the package elsewhere calls unacceptable.
2. Correct the round-1 `LOGBOOK.md` entry in place. It is new content in this
   CR; the numbers in it were disproven before it was written.
3. Either wire a report reader that runs after the parallel tests, or drop the
   counter mechanism to what it actually is and remove the two outcome-document
   sentences that credit it with evidence it never produces. Whichever way, pin
   both `Require → recordVerdict` call sites (M04, M05).
4. Arm `PointRollbackEnter` somewhere, or state it as the point the model does
   not drive.
5. Bound or close M08 — the symlink probe's strictness is the one capability
   invariant of the four that a mutant can quietly remove.
