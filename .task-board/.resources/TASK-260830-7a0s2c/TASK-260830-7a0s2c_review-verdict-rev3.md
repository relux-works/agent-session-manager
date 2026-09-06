# TASK-260830-7a0s2c — review verdict, CR rev3

**Verdict: ACCEPTED** (`accept_cr(TASK-260830-7a0s2c, revision=3)`).
`repeat-of: none` — both round-2 `repeat-of` classes (rev1/C1 finalize gate,
rev1/C2 skip report) are closed at the class, not patched at the vector.

- Run: `RUN-260906-e53010`, reviewer archetype, bound to rev3.
- Base `722ed1a8c3ac0e4169990f55318897f8e39247f0`, candidate tree
  `7be8309b01f900f0162f1bc525e98ca857fcc853`, 19 changed paths.
- Live worktree tree re-derived with a temporary index **before** probing and
  **after** every mutant: `7be8309b…` both times. Package manifest
  `c6018721…` identical throughout.
- All mutation work ran in `/tmp/axrev3`, a byte copy of the candidate,
  verified equal to a pristine backup afterwards and then deleted. The two
  G-A probes ran in the live worktree from a file backup and were restored
  byte-exactly (manifest re-verified).

---

## G-A (blocking) — the report, end to end: **MET**

Round 2's exact pair rerun on this tree. The failure is that the report named
only synthetic self-test rows while the real capability was absent. It no
longer does.

| Probe | Test outcome | `TestMain` final report line |
| --- | --- | --- |
| strict `SymlinkCapability` probe forced false | `--- FAIL: TestHostileSymlinkEscapeRefusedAtCommit`, exit 1, 245 PASS / 1 FAIL / 0 SKIP | `secconftest capabilities on darwin: skipped=0 "" failed=1 "symlink"` |
| non-strict `ModeBitsCapability` probe forced false | `--- SKIP: TestSkipModeBitsAndNonRootDenialsHold`, exit 0 | `secconftest capabilities on darwin: skipped=1 "mode-bits" failed=0 ""` |
| baseline (unmutated) | 246 PASS / 0 SKIP / 0 FAIL, exit 0 | `skipped=0 "" failed=0 ""` |

The real capability ID is named in both directions. The mid-suite
`skip_test.go:264` line still carries only its own synthetic rows — correct,
and the delivered docs no longer ask it to witness parallel verdicts.

**Ordering is not luck.** The concern behind round 2's finding was that a
non-parallel reader runs before the parallel writers. `TestMain` runs after
`m.Run()`, and the three counter-mutating tests restore via `t.Cleanup`
before any parallel body resumes. I did not accept that from the Go model:
the mode-bits probe survives `-shuffle=1..5`, printing
`skipped=1 "mode-bits"` on all five seeds.

**A `requireT` fake cannot satisfy the pin while the wiring is broken.**
The fake records `Skipf`/`Fatalf` only; the assertion is on the process-wide
counters, which only real `recordVerdict` writes. I planted two mutants the
fake cannot see — narrowing `recordVerdict`'s own `Skipped` and `Failed`
arms (S3r, S4r) — and both die to `TestRequireSkipArmRecordsVerdict` /
`TestRequireFailArmRecordsVerdict` *and* `TestSkipReportReadsRecordedVerdicts`.
M04/M05 reproduce as KILLED.

---

## G-B (blocking) — C1b's terminal neighbours: **MET**

Every neighbour driven through the public API in a probe test, asserting
error **identity**, not message text. Round 2's finding was that two refused
only by inheriting a raw filesystem `ENOENT`.

| Sequence | Error identity now | `errors.Is(err, fs.ErrNotExist)` |
| --- | --- | --- |
| rollback twice | `secconftest: rollback without prepare for operation "rr"` (`*errors.errorString`, unwrap nil) | **false** |
| commit twice | `ErrCommitPastFinalize` sentinel via `%w` | **false** |
| commit after rollback | `secconftest: commit without prepare for operation "cr"` (unwrap nil) | **false** |
| prepare after clean commit | `ErrPreparePastFinalize` sentinel | **false** |
| rollback after **crashed** commit | `ErrRollbackPastFinalize` sentinel; `CommittedBytes` still `"BODY"` | **false** |
| receipt present, staged bytes gone | names `commit without staged bytes` **then** wraps the real ENOENT | true, deliberately — it names the state first |

No refusal is an inherited errno with a nicer message wrapped around it. The
two without-prepare refusals are produced by the receipts-map check, not by
the filesystem; `os.Remove` errors on that path are discarded, so no
filesystem outcome can reach the caller.

**P7 closed.** `TestTransactorCrashOutcomes` now arms all five points
including `PointRollbackEnter`; I drove it independently — `Arm` registers it
(`RemainingArmed = [CR-MAT-rollback-enter]`), `Rollback` raises the `*Fault`
with `explicit_rollback`, `RemainingArmed` is empty after the drive, and the
staged bytes survive. The producer's P7 mutant (`Arm` silently drops
`PointRollbackEnter`) reproduces as KILLED.

---

## The C1 decision — judged, and correct

Rule chosen: **a crashed commit counts as committed.** I accept it, and the
reasoning, on four grounds — the third is the producer's, the other three are
mine.

1. The fault fires strictly after `os.WriteFile(committed, …)` and
   `os.Remove(staged)`. There is no observable sense in which the operation
   is uncommitted: `CommittedBytes` returns the body.
2. `Classify(PointCommitApply)` is `recoverable_parked_state`, whose own
   definition in the same file says the operation "must never silently resume
   as a fresh identity". A `Rollback` that deletes `.committed` and forgets
   the receipt — after which a fresh `Prepare` restages — is exactly a silent
   resume as a fresh identity. **The alternative rule contradicts the
   vocabulary the same file defines.**
3. AC-CLONE-005 has two halves — rollback forbidden after Provider commit,
   and all bundle evidence survives. Admitting the rollback walks through
   both at once. The producer's argument is right.
4. The rule is applied consistently at every neighbour, not only at
   `Rollback`: `Prepare` and `Commit` past finalization are refused too, so
   "terminal" means the same thing in all three phases. A rule enforced in
   one phase and not the others would be the round-2 defect in new clothes.

What the alternative would have bought — an operator-free cleanup path — is
given up explicitly and named ("recovery from parked is via `CommittedBytes`
and an operator"). That is a stated decision, not an omission. Picking
neither was the unacceptable answer and is no longer the state.

---

## G-C — C2's superseded numbers: **MET, with one named defect**

What `LOGBOOK.md` now says at the 2035 entry:

> CORRECTION (round-3): the mutant counts and the darwin skip citation first
> written here (10-row table, 10/10 killed, skipped=0) were disproven in
> round-1 review and are superseded by …

The disproven numbers are no longer **asserted**; they are named as
withdrawn. That is the honest form of an in-place correction — it does not
erase what was claimed, so a reader can see the claim and its retraction
together. History is not silently rewritten.

**Finding G-C/1 (non-blocking, documentation).** The correction redirects the
reader to round 2's `29 applied / 2 survivors with bounds`, and the 2105
entry still asserts `29 applied / 27 killed / … 2 survived with bounds`
uncorrected. Round-2 review independently measured that same tree at
`54 applied / 44 killed / 10 survived` with seven unbounded survivors
(M04, M05, M08, P7, M27, M02, M16). So the correction on one entry points at
a number that was itself disproven, and the entry holding it carries no note.

Why this is not blocking: nothing about the **current** tree is misstated.
The 2155 entry above both of them is accurate — I reproduced its
`38 applied / 36 killed / 2 survived` row-for-row — and it explicitly lists
M02/M08/M16/M27 as survivors closed and M04/M05/P7 as killed, which
contradicts "2 survived" for a top-down reader. Fix on the next touch of this
file: point the 2035 correction at the round-3 numbers, and annotate 2105
with what independent review measured.

Sweep of the delivered tree for superseded numbers (`10/10`, `0 survivors`,
`skipped=0`, `29 applied`, `27 killed`, `86.9`, `89.3`, `10-row`): every hit
outside `.task-board/` (not part of this CR) is in `LOGBOOK.md` and is either
the correction note itself or a historical gate reading of a tree that no
longer exists. `README.md`, `doc.go`, and `task-board.config.json` are clean.

---

## G-D — the manifest and the denominator: **MET**

**Manifest, verified independently.** The outcome doc states the formula
("sha256 of the sorted per-file sha256 manifest over `internal/secconftest/`").
I implemented it from that sentence, not from the producer's script, over the
16 files, and got
`c6018721bd0941b4ce29a7e36fd37108b625711c6024fb541ab9ef7cb22cfb0d` —
byte-exact. It reproduced again after the whole battery and again at the end
of this review. This is a real improvement on rev2, whose fingerprint did not
reproduce under four plausible readings; the difference is that rev3 states
the formula.

**Denominator, re-derived.** I did not accept the producer's list as the
denominator. I re-ran their 38-row battery in a sandbox copy — **identical
row-for-row**, `38 applied / 0 NOT_APPLIED / 0 COMPILE_FAIL / 0 VET_FAIL /
36 KILLED / 2 SURVIVED`, `MANIFEST IDENTICAL` — then enumerated the gates in
the delivered production files that their list does **not** attack, and built
a supplemental battery of 29. Every supplemental mutant uses the **full
behavioral suite** as its harness (no `-run` mask), so a mutant cannot be
scored against a test that never reaches it.

Combined, with the distinct rows kept distinct:

| Kind | applied | run | KILLED | SURVIVED | NOT_APPLIED | COMPILE_FAIL | VET_FAIL |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| narrowing | 54 | 53 | 43 | 10 | 0 | 1 | 0 |
| arm-deletion | 8 | 8 | 8 | 0 | 0 | 0 | 0 |
| census-only | 5 | 5 | 5 | 0 | 0 | 0 | 0 |
| **Total** | **67** | **66** | **56** | **10** | **0** | **1** | **0** |

Named explicitly, as required: **P7 KILLED** by `TestTransactorCrashOutcomes`;
**M04 KILLED** by `TestRequireFailArmRecordsVerdict`; **M05 KILLED** by
`TestRequireSkipArmRecordsVerdict`. All three survived in round 2.

The single COMPILE_FAIL is mine (S5r left `info` unused) and is reported as
its own row, not folded into the denominator as a pass.

Twenty of my 29 supplemental mutants died to the delivered suite, including
the gates the producer's list omitted entirely: `Runner.Run`'s
unreached-fixture refusal (R1), its fixture-error propagation (R2),
`SortedNames`' own sort (R4), `QuoteInSection`'s nil-document guard (H1/H2 —
the "a failed read is never an absence" rule), the render-controls corpus's
newline exception (H4), a corpus row admitting a legitimate member (H5),
`Fake.Advance`'s negative-delta no-op (K1), `Recording.Now`'s counter (K2),
`decide`'s strict arm (S2r), `strictOn` (S7r), both `recordVerdict` arms
(S3r/S4r), `Prepare`'s idempotency check narrowed rather than deleted (C1r),
`MaybeFail`'s unknown-point refusal (C2r), a `Classify` row other than
commit-apply (C3r/C6r), and `DriveMemberPath` admitting everything with its
arrival preserved (F3r).

### The 10 survivors, classified honestly

Two are the producer's, with bounds already stated:

1. **N-DriveRedactNoDelegate** — driver delegation read-verified, not
   mutant-pinned (stated bound N16).
2. **S1-ModeBitsFalse** — a non-strict probe reporting unavailable skips by
   construction; the `--- SKIP` line is the signal (stated bound).

Eight are mine. Four are equivalent or platform-conditional rather than holes,
and I say which is which rather than counting them all as findings:

3. **H3** `QuoteInSection` quote-absent guard narrowed — *equivalent for
   behavior*. The terminal fallthrough still refuses; only the diagnostic
   message changes.
4. **F1r** `DriveGuardResolve` construction-failure arm flipped to admit —
   *equivalent*. The fixed `/stage` root cannot fail construction, exactly as
   the code comment states.
5. **S6r** `nonroot` probe uid check narrowed — *platform-conditional*. On a
   non-root host the mutant is behaviorally identical; it is the same class
   as the producer's S1 bound.
6. **S1r** `Require`'s own malformed-capability gate `||`→`&&` — redundant
   with `decide`'s identical check, which still returns `decisionFail`. Worth
   naming because the mutant *panics* on `ID != "" && Probe == nil` and no
   test notices: `Require` is never driven with a malformed capability.
   Misuse guard in test infrastructure; no gate hole.

Four are real, unstated bounds — all instrument-internal, none in a gate the
AC names:

7. **R3** `Digest` field separators (`"%s\x00%s\x00%d\x00"` → `"%s%s%d"`)
   survives the full suite. Dropping/reordering/altering a fixture *is*
   pinned; what is not pinned is field-boundary ambiguity, so
   `{Name:"ab",Report:"c"}` and `{Name:"a",Report:"bc"}` would collide.
8. **F2r** `Driver.Count` returning a phantom 1 for a zero count survives.
   `TestFuzzDriversReachProduction` asserts `Count(entry) == 0` → fail, so the
   arrival evidence is not itself pinned. This *refines* the stated N16 bound:
   the bound says delegation is "read-verified with arrival counts", but the
   counts it leans on are forgeable by the accessor. Not a hole in practice —
   `Redact`'s behavior is pinned by `TestHostileRedactMembersScrubbed`, and
   F3r shows a driver that stops deciding *is* caught behaviorally.
9. **C5r** `Injector.Armed` hardcoded to `true` survives. Its one call site
   asserts only the positive direction; nothing asserts `Armed()` is false for
   a disarmed point.
10. **K3** `Fake.Now` leaking `time.Now()` when `set && current.IsZero()`
    survives. Reachable only via `Set(time.Time{})`, which nothing drives.

### One more unstated bound, from reading the C1 code

The finalize gate is **not atomic with the durable write**. In `Commit`,
`os.WriteFile(committed, …)` and `os.Remove(staged)` happen before
`MaybeFail(PointCommitApply)`, and `finalized` is set only after — under the
mutex, but after. A concurrent `Rollback` on the same operation ID in that
window would see `finalized == false`, `ok == true`, and delete
`.committed`. Nothing drives concurrent phases on one ID and `-race` is
clean, and `doc.go` already bounds the Transactor as "a test model … not the
product journal", so this is a bound to state rather than a defect to fix.
Worth writing down before anyone reads this model as a recovery reference.

### One census that is off by one

`doc.go` says "both `recordVerdict` call sites pinned". There are **three**
in `Require` (skip.go:251, 261, 266); two are pinned. The unpinned one is the
malformed-capability guard from survivor S1r. `2 of 3`, not `both`.
Non-blocking — the two arms that carry the round-2 finding are the pinned
ones — but a census stated as a count should match the count.

---

## G-E — provenance and the frozen leaf: **MET**

- `git diff 722ed1a 7be8309b -- internal/secprim` — empty. `git status
  --porcelain -- internal/secprim` — empty. No untracked additions there.
- The outcome document names base `722ed1a`, which is the CR's base OID and
  the worktree `HEAD`.
- Live worktree tree derived with a temporary index equals the candidate
  `7be8309b01f900f0162f1bc525e98ca857fcc853` — checked before probing, after
  the G-A live-tree mutants, and at the end of this review.

---

## AC coverage: 6 of 6 rows driven, spot-checked against the mutants

| # | AC row | Production call site | Checked by |
| --- | --- | --- | --- |
| 1 | deterministic fixture runners | `Runner.Run`, `Digest`, `SortedNames` | N-Digest{Report,Name,Calls}, A-RunSortDeleted, R1, R2, R4 all KILLED |
| 2 | fake clocks | `Fake.Now/Advance/Set`, `Recording.Now` | N-FakeWallClock, K1, K2 KILLED |
| 3 | crash points | `Injector.MaybeFail`, `Transactor.Prepare/Commit/Rollback` | C1a–C1f, P7, C1r, C2r, C3r, C6r, A-Finalize/CommitWithoutPrepare/Idempotency/RollbackForget KILLED; my six-neighbour identity probe |
| 4 | protocol fuzzing | 8 `Driver` entries, 8 `Fuzz*` targets | 22 arrivals at each of 8 entries (22 seeds × 1); all 8 registered in `task-board.config.json`; F3r, C-FuzzEntryDrops KILLED |
| 5 | hostile strings | `secprim.CheckMemberPath/Guard.{Resolve,Open}/DetectCaseCollision/CheckArgv/IsEnvName/EscapeForTerminal/Redact` | 18 roster Gates pinned **and driven** bidirectionally; N-Bidi{FEFF,Cut}, N-GateRename, N-TokenPreserving, H4, H5, H6 KILLED |
| 6 | platform capability skips | `Require`, four probes, `TestMain` | G-A above; M02, M04, M05, M08, S2r, S3r, S4r, S7r, N-FifoTrueOnFailure, C-CapsDrops KILLED |

The source-text gate requirement is met: **N-TokenPreserving** drops the
section check while keeping the `section` identifier read, and its harness is
the full behavioral suite — it reddens at `TestQuoteInSectionRefusals`.

**"No unsupported capability is advertised"** — `grep` for imports of
`internal/secconftest` outside the package itself returns nothing. The
instrument enables no platform lane or provider suite.

**Fuzz registration is load-bearing, verified by attack.** Deleting one of the
eight registrations from `task-board.config.json` in a sandbox reddens
`TestConfiguredValidationRunsEveryFuzzTargetWithFixedBudget`
(`contains … 0 times, want exactly once`).

---

## Gates I ran myself (real exit codes, this worktree)

| Gate | Exit | Result |
| --- | ---: | --- |
| `go test ./... -count=1` | 0 | 21 packages ok |
| `go test ./internal/secconftest -v -count=1` | 0 | 246 PASS / 0 SKIP / 0 FAIL |
| `go test ./internal/secconftest -race -count=1` | 0 | ok |
| `go test ./internal/secconftest ./internal/secprim -cover` | 0 | 92.6% / 94.4% — matches the outcome doc |
| `go vet ./...` | 0 | clean |
| `GOOS=windows go vet ./internal/secconftest/` | 0 | clean |
| `GOOS=windows go build ./...` | 0 | clean |
| `gofmt -l internal/` | 0 | clean |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | contracts=60 sections=36 cases=94 fixtures=30 compat=55 — unchanged |
| producer battery, sandbox re-run | 0 | 38/36/2, rows identical, `MANIFEST IDENTICAL` |
| reviewer supplemental battery, sandbox | 0 | 29 applied, 20 killed, 8 survived, 1 COMPILE_FAIL, `MANIFEST IDENTICAL` |

Accepted from attached evidence without rerunning, and named as such: the
8 × `-fuzz -fuzztime=100x` runs.

---

## Summary

Both `repeat-of` classes are closed at the class, not at the vector. The
report observes a real parallel verdict in both directions and survives
shuffling; every terminal neighbour of the finalize gate refuses by a
deliberately produced, named error rather than an inherited `ENOENT`; the
fifth crash point is driven and consumption-checked; the manifest reproduces
from its stated formula; and the denominator, re-derived at 67 applied with
28 gates the producer's list never touched, adds no gate hole — its ten
survivors are four equivalent/platform-conditional rows and six
instrument-internal bounds.

The C1 rule is the right one and the argument for it holds. Findings G-C/1
(the correction points at a superseded number), the `2 of 3` recordVerdict
census, the four unstated survivor bounds, and the non-atomic finalize window
are all recorded here for the next touch of this package; none of them is a
gate that admits what it must reject, and none justifies a fourth round.
