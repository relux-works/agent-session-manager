# TASK-260830-7a0s2c — review verdict, CR revision 1

**Verdict: changes_requested → `to-dev`.**
**repeat-of: none** (round 1).

Reviewer run `RUN-260906-bd510e`. Candidate tree `9ec5199c07a6fcf6f6dfa1ee8b682ec13037b3e4`
verified equal to the live worktree tree before and after every plant; base
`722ed1a8c3ac0e4169990f55318897f8e39247f0` (tree `5eb6d8a5…`). All evidence below was
produced by this run on darwin/arm64, go1.25.5.

This leaf ships an instrument, so the review question was "what would make it fail".
Four mechanisms answer "nothing".

---

## Blocking findings

### C1 — `Transactor.Rollback` claims a normative refusal it does not implement, and the same-named test never drives it

`internal/secconftest/crash.go` (`Rollback` doc): *"It refuses to run past finalization:
once Committed returns cleanly the operation is terminal, and rollback is forbidden
(AC-CLONE-005 shapes this rule; **the model enforces it**)."*

The cited rule is real. Pinned `SPEC.md:11964`, `AC-CLONE-005`: *"rollback remains
available through finalization intent and **is forbidden after Provider commit**"*.

The code contains no finalization state and no such refusal. `Rollback` checks only
"was this operation ever prepared". Measured:

```
PROBE Rollback after clean Commit returned: <nil>
PROBE committed file after rollback: statErr=<nil-path removed>   # durable bytes destroyed
```

`TestTransactorRefusesRollbackPastFinalize` drives exactly two cases —
rollback-without-prepare, and pre-commit rollback. It never calls `Commit` and never
reaches the case its name asserts. Directed mutant confirms what it actually measures:

| mutant | result |
| --- | --- |
| `Rollback` without-prepare refusal deleted | **KILLED** by `TestTransactorRefusesRollbackPastFinalize` |
| post-commit rollback (the named rule) | **no test exists**; behavior admits it |

Shape: a named refusal advertised as enforced, admitted in practice, with a test whose
name is the only thing standing in for the driver. Either implement the finalize gate and
drive it, or delete both the claim and the misleading test name. Do not keep the name.

Same comment block, minor: `crash.go` attributes `CR-CLONE-01..16` to the AC-CRASH-001
class list. `SPEC.md:11959` lists LAUNCH/SYNC/MAT/GRACE/FORCE/FORK/STOP/RESUME/RESTORE for
AC-CRASH-001; `CR-CLONE-01..16` belongs to AC-CLONE-005 (`SPEC.md:11964`).

### C2 — the capability skip report is structurally incapable of reporting anything but zero

`skip.go` `recordVerdict`/`SkippedIDs`/`LogReport` exist so "the skip count on the host is
evidence rather than prose". `TestSkipReportLogsHostCounts` asserts in its own comment
that "the counts come only from Require calls the suite really made, so the number is
evidence, not prose".

`LogReport` is called from the package's only **non-parallel** test. Every test that calls
`Require` is `t.Parallel()`. Go runs non-parallel tests to completion before resuming
paused parallel ones, so `LogReport` always reads the counters before anything can
increment them. Three independent proofs:

| plant | observed |
| --- | --- |
| symlink probe forced `false` (strict darwin → fail) | `--- FAIL: TestHostileSymlinkEscapeRefusedAtCommit`, report still printed `skipped=0 "" failed=0 ""` |
| symlink probe forced `false` + `StrictGOOS=["plan9"]` (→ skip) | `--- SKIP: TestHostileSymlinkEscapeRefusedAtCommit`, report still printed `skipped=0` |
| `recordVerdict` body replaced with `return` | **SURVIVED** — package fully green |

`skipped=0 failed=0` is cited as attached G-D evidence in
`TASK-260830-7a0s2c_secconftest.md`, in `README.md` ("the suite reports `skipped=0`") and
in the `LOGBOOK.md` entry. It is not evidence about this host; it is a constant. The one
genuine skip datum in the round is the `0 --- SKIP` lines in `go test -v`, which this run
reproduces repo-wide (`go test ./... -count=1 -v`: 0 SKIP lines) — that one is real, keep
it, and either wire the counter or delete it.

### C3 — 3 of the 4 capability probes gate nothing, and one of them does not probe

`Require` — the only function that turns a probe into a skip or a fail — has exactly one
call site in the repository:

```
internal/secconftest/skip.go:217:func Require(...)
internal/secconftest/hostile_test.go:252:	Require(t, SymlinkCapability())
```

`fifo`, `mode-bits` and `nonroot` are constructed and called only by
`TestSkipDefaultProbesTerminate` / `TestSkipProbesAreDeterministic`, which assert shape and
stability, never a decision. **1 of 4 probes decides any skip.**

`FifoCapability`'s probe never attempts a named pipe. On every non-Windows host it is
`return true, "non-Windows host: fifo fixtures build"` — no `Mkfifo`, no filesystem call.
On `StrictGOOS: {linux, darwin}` it can therefore never skip and never fail: it is a
constant. That contradicts the `Capability` doc's own contract in the same file — *"A skip
asserts 'this cannot run here': the probe must really attempt the capability"* — while
`README.md` advertises all four as probes that "skip only where the platform cannot
provide" and the outcome document reports "`fifo ok`". This is the AC's own last clause
(*"no unsupported capability is advertised"*) failing on the instrument's own report.

Positive control: the strict-platform arm is genuinely load-bearing where it is wired —
the forced-false symlink plant produced `capability "symlink" unavailable on strict
platform darwin` and a **FAIL**, not a quiet skip, and the `decide` mutant
`strict-absence → skip` was **KILLED** by `TestSkipDecideTable`.

### C4 — mutant residue: 6 survivors from 18 directed mutants inside `internal/secconftest`

The round reported "10 applied, 10 killed, 0 survivors" with the denominator given as
"all mutants in `internal/secconftest` non-test files" — that is a statement about where
the mutants live, not a denominator, and no residue was stated. G-E asked for exactly
this. Re-derived from production, targeting the claims the package makes in prose:

| # | mutant (narrowing / census, all applied and compiling; `go vet` clean) | result |
| --- | --- | --- |
| Ma | `Digest` drops the `Report` component | **SURVIVED** |
| Mb | `Digest` drops the `Name` component | **SURVIVED** |
| Mj | `Digest` reduced to call counts only (drops `Name` **and** `Report`) | **SURVIVED** |
| Md | `bidiOverrideRunes` drops `U+FEFF` | **SURVIVED** |
| Me | `bidiOverrideRunes` cut from 16 runes to `{U+202E}` | **SURVIVED** |
| Mf | `HostileClass.Gate` for `path-traversal` renamed to `secprim.Redact` | **SURVIVED** |
| Mg | `recordVerdict` never records (see C2) | **SURVIVED** |
| Mh | `Commit` without-prepare refusal deleted | **SURVIVED** |
| Mc | `Digest` drops the `Calls` component | KILLED — `TestRunnerDigestCoversAllFixtures` |
| Mi | `Rollback` without-prepare refusal deleted | KILLED — `TestTransactorRefusesRollbackPastFinalize` |
| Mk | `QuoteInSection` section check narrowed to a 2-char prefix (token preserved) | KILLED — `TestQuoteInSectionRefusals` |
| Ml | `QuoteInSection` section check dropped entirely (token preserved) | KILLED — `TestQuoteInSectionRefusals` |
| Mm | `QuoteInSection` treats a nil document as satisfied | KILLED — `TestQuoteInSectionRefusals` |
| Mn | `Members("render-controls")` drops the C1 range `0x80-0x9f` | KILLED — `TestHostileMemberCounts` |
| Mo | `decide` maps strict absence to skip instead of fail | KILLED — `TestSkipDecideTable` |
| Mp | `MaybeFail` keeps the arm (fires forever) | KILLED — `TestCrashPointFiresOnceWhenArmed`, `TestTransactorRecoversAfterFault` |
| Mq | `Prepare` idempotency digest compare removed | KILLED — `TestTransactorIdempotentRetry` |
| Mr | `Fake.Advance` negative-delta guard removed | KILLED — `TestFakeAdvanceAndSet` |
| Ms | roster quote → plausible near-miss `"path traversal attacks"` | KILLED — `TestHostileRosterQuotesResolve` |
| Mt | roster class `Section` moved `16.3 → 16.4` | KILLED — `TestHostileRosterQuotesResolve` |

**Rows: 20 applied, 0 NOT_APPLIED, 0 COMPILE_FAIL, 0 VET_FAIL, 14 KILLED, 6 SURVIVED.**
(Every mutant was built with `go build` exit 0 and `go vet` exit 0 before its colour was
read, per the round brief's compile-failure warning; each was reverted from a pristine
copy and the package re-confirmed green.)

Each survivor contradicts a claim in the shipped text:

- `runner.go`: *"any dropped, reordered, or altered fixture changes it"* — only the
  **call-count** third of the digest is witnessed. `TestRunnerDigestCoversAllFixtures`
  varies entry count and call count; nothing ever varies a `Report` at constant
  name/count, and `TestRunnerDigestIsDeterministic` only asserts equality, never
  inequality. The digest can be reduced to `Calls` alone and stay green.
- `hostile.go`: *"The static list is the point: a gate arm deleted with its test row still
  faces every rune here."* Nothing pins the list's contents or length. `BidiRunes()` feeds
  `assertTerminalInert`, the shared invariant behind every render test and two fuzz
  targets, and it can be silently emptied to one rune. The **corpus** is pinned
  (`TestHostileMemberCounts`); the **assertion set** is not.
- `HostileClass.Gate` documents "the production entry point deciding the members" and the
  outcome table leans on those names, but nothing asserts a class is driven through its
  named gate. It is unchecked annotation.
- `Commit`-without-prepare is a real refusal in production with no test driving it.

---

## What held up (verified, not read)

- **Spec pinning of the hostile roster is real and load-bearing.** A near-miss quote (Ms)
  and a wrong section (Mt) both redden `TestHostileRosterQuotesResolve`. `QuoteInSection`
  resisted every token-preserving narrowing I could build (Mk/Ml/Mm all killed). The
  document itself is digest-pinned in `specdoc`, so the "class removed from the document"
  direction fails closed twice over.
- **Fuzz arrival is real, not a counter trick.** Defeat plants at the *production* entry
  (not the driver) — restored from a pristine `internal/secprim` copy afterwards:

  | plant at the secprim entry | target that must notice |
  | --- | --- |
  | `EscapeForTerminal` → identity | `FuzzEscapeForTerminal` **FAIL**, `FuzzRenderForTerminal` **FAIL** |
  | `CheckArgv` → admit all | `FuzzCheckArgv` **FAIL** |
  | `CheckMemberPath` → admit all | `FuzzCheckMemberPath` **FAIL**, `FuzzGuardResolve` **FAIL** |
  | `DetectCaseCollision` → always collide | `FuzzDetectCaseCollision` **FAIL** |
  | `IsEnvName` → admit all | `FuzzIsEnvName` **PASS** (package RED via `TestHostileEnvNamesRefused`) |
  | `Redact` → identity | `FuzzRedactCorpus` **PASS** (package RED via `TestHostileRedactMembersScrubbed`) |

  **6 of 8 targets** carry a property that reddens when their entry gate is fully
  defeated. Seed corpus 22 per target, 22 arrivals per entry, stable across 3 runs.
- **Determinism holds.** Digest `5569f34caab734eb` identical across **10 independent
  processes**; 5/5 package runs exit 0; arrival counts stable across 3 runs. No
  map-iteration flake.
- **The fuzz-registration gate is load-bearing.** Deleting the `FuzzGuardResolve` line from
  `task-board.config.json` reddens
  `TestConfiguredValidationRunsEveryFuzzTargetWithFixedBudget` with the exact missing
  command named. A target cannot be added unfuzzed-in-CI.
- **Provenance clean (G-F).** Live worktree tree == candidate tree
  `9ec5199c07a6fcf6f6dfa1ee8b682ec13037b3e4`; base tree `5eb6d8a5…` matches the outcome
  document; `git diff 722ed1a 9ec5199c -- internal/secprim` is **empty**. The document
  names the tree the record actually carries.
- **Gates reproduce.** `go test ./... -count=1` exit 0 (21 packages, 0 `--- SKIP` lines);
  `go vet ./...` exit 0; `gofmt -l internal/` clean; `GOOS=windows go build ./...` and
  `GOOS=windows go vet ./...` exit 0; `tracecheck` exit 0 (contracts=60 sections=36
  cases=94 fixtures=30 compat=55); `secconftest` coverage 86.9%.

## Bounds the round should state rather than leave implied

1. `FuzzIsEnvName` and `FuzzRedactCorpus` assert only properties a **no-op gate satisfies**
   — `BuildEnv` under an absent-parent lookup accepts anything `IsEnvName` admits, and
   identity is trivially idempotent. Their class is covered by the hostile roster, not by
   the fuzz target. State it or give each target an independent property.
2. The crash-point vocabulary is hand-written and **unpinned**, unlike the hostile roster.
   None of the five `CR-MAT-*` point names occurs in the pinned `SPEC.md` (the spec's
   registry is `CR-MAT-01..08`), and no test relates them. `Classify` rejects unknown
   names, so a new crash site is silently unarmable until someone hand-adds a constant.
3. An armed point that is **never reached** is noticed only incidentally, where a test
   happens to assert a `*Fault`. There is no "all arms consumed" assertion, so a future
   fixture that arms a point on a path it never takes passes.
4. AC row 6 ("platform capability skips") should be reported as **1 of 4 probes wired**,
   not as four probes deciding skips. Rows 1–5 are driven; row 1's digest witness is 1 of
   3 components (C4).

## Route

`to-dev`. C1 is decisive on its own: a normative refusal advertised as enforced, admitted
in practice, guarded by a test whose name is the only thing that looks like a driver — in
a leaf whose entire purpose is that a suite which cannot fail is the defect. C2 and C3
turn the round's G-D evidence into a constant. C4 supplies the residue G-E asked for.
