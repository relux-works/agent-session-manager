# TASK-260830-2xdt8t — Review round 3 verdict

- Run: `RUN-260904-27510d` (reviewer, Opus 5)
- Change Request: `CR-TASK-260830-2xdt8t-3` revision 3
- Verdict: **ACCEPTED**
- Date: 2026-09-05

## 1. Repository delta — the `empty` flag is a snapshot artifact, not an empty leaf

The CR reports `repository_delta=empty`. Measured:

| Fact | Value |
| --- | --- |
| CR base OID | `9481a20393408eff023349ad988f5aed3e4559b2` |
| CR candidate tree OID | `0411a188a90226b7464da8ce1737750605165457` |
| `git rev-parse HEAD` | `9481a203…` |
| `git rev-parse HEAD^{tree}` | `0411a188…` |
| `git rev-parse 9481a20^{tree}` | `0411a188…` |

The base OID **is** the delivered commit, so the patch diffs the commit against
its own tree and necessarily has zero paths. This is the same artifact recorded
at rounds 1 and 2 of this task (skill-project-management #113), not a producer
who changed nothing.

The real delta is commit `9481a20` itself:

```
internal/terminalbackend/internal_pin_test.go    | 30 ++++++++++
internal/terminalbackend/terminalbackend.go      |  7 ++-
internal/terminalbackend/terminalbackend_test.go | 75 ++++++++++++++++------
3 files changed, 92 insertions(+), 20 deletions(-)
```

`git verify-commit 9481a20` → `Good "git" signature for oparin@me.com with ECDSA
key SHA256:V6JiKG7J29mjsvikcLoSVp0bLa77VTsFy12gnLO81cM`. Parent is the reviewed
round-2 head `9e31f14`. Worktree clean, not pushed.

**Accepting is therefore not accepting an empty change.** I am accepting commit
`9481a20`, whose content I drove. The `empty` flag was resolved by measurement
before any verdict reasoning, not assumed away.

**Delivery-shape fact for the orchestrator, not a review finding:** the branch
now stands **three** commits past checkpoint `57afcc6` (`b5cf5b5`, `9e31f14`,
`9481a20`). The task notes ask for exactly one commit past the checkpoint; the
extra two are the two rework rounds, and the checkpoint has not advanced because
acceptance had not happened. The checkpoint step owns this.

## 2. Baseline, measured by running

| Check | Result |
| --- | --- |
| `go test ./... -count=1` | exit 0, 14 packages |
| `go vet ./...` | exit 0 |
| `gofmt -l internal/` | empty |
| `tracecheck` | exit 0 — `acceptance_cases=74`, `clauses_discharged=17/403`, `bindings=49 full=1 partial=3 sliver=1` |

The producer's reported figures reproduce exactly. Figures unmoved from round 2,
as claimed.

## 3. The three round-2 blocking findings — all closed, each verified by running

### G1 — the subtest that could not fail

The producer took **both** options offered: the inert black-box subtest
`TestRegistryCopiesSlicesAcrossItsBoundary/built-ins do not share a protocol array`
is deleted, and a white-box pin
`TestBuiltinsHoldIndependentProtocolArrays` (internal_pin_test.go:91) compares
the first-element addresses of the two **stored** records, bypassing the cloning
accessor.

Measured, mutant presence-checked (`count == 1`, mutant text present and original
absent after substitution, byte-identical restore):

- **R03** (both `New` clones reverted → built-ins share one backing array):
  **KILLED**, and killed by exactly one test —
  `internal_pin_test.go:110: ax.tmux and ax.conpty share one protocol backing array`.
  Round 2 measured R03 as a **survivor**; it is now dead.
- Under the same R03 mutant, my own black-box probe through `Resolve`
  (`TestProbeCrossBuiltinAliasingIsUnobservableThroughExports`) **PASSES**. That
  is the direct confirmation that the deleted subtest was structurally unfailable
  and that the white-box pin is the only construction that can observe the
  property.

**Correction to the orchestrator's stated pre-verification.** The round-3 brief
records "I reverted the built-in clone and the suite stayed green… That is now
the CORRECT outcome, because no test claims the property any more." That reading
does not match the shipped code. A test *does* claim the property, and the
two-site revert reddens the suite. The green the orchestrator observed reproduces
only for a **single-site** revert (X01/X02 below), which is a genuinely
equivalent mutant. The correct reading is: R03 is killed.

### G2 — the one-directional platform bound

`TestRegisterExternalAdmitsFullPlatformSet` (terminalbackend_test.go:281) drives
a four-platform record through the production admission entry point
`Registry.RegisterExternal`, then asserts `len(admitted.Platforms) == 4` **and**
member-for-member equality against `[linux macos windows wsl2]` — admission with
the set intact, not merely absence of an error.

| Mutant | Direction | Result | Killed by |
| --- | --- | --- | --- |
| X05 / N01 | `>4` → `>3` (last admitted value) | **KILLED** | `TestRegisterExternalAdmitsFullPlatformSet` |
| X06 | `>4` → `>2` | **KILLED** | 3 tests |
| X07 | `>4` → `>1` | **KILLED** | 3 tests |
| R11 / M22 | `>4` → `>5` (first refused value) | **KILLED** | `TestRegisterExternalRefusesPlatformViolations`, `TestValidatePlatformsReportsTheBoundFirst` |
| N02 | `>4` → `>6` | **KILLED** | same |

N01 was a round-2 survivor. The bound is now driven at the last admitted value
and the first refused one, and across three narrowing steps.

### G3 — the reserved-namespace guard hidden behind its predecessor

`TestRegisterExternalRefusesReservedNamespaceDistinguishingValue`
(terminalbackend_test.go:625) drives `ax.conpty` — the value `mustParseID`
admits and only the `ax.`-prefix bar in `TrustEntry.validate` refuses — and
asserts the **clause** (`external_trust reserved namespace`) rather than merely a
refusal, which is what makes the fall-through to a drift refusal detectable.

I verified the distinguishing domain rather than accepting it: `ParseID` admits
exactly `{ax.tmux, ax.conpty}` of the `ax.*` space (everything else is
`terminal_backend_not_found` / reserved namespace). So the guard's distinguishing
set has exactly two members, and both are now driven:

| Mutant | Result | Killed by |
| --- | --- | --- |
| X08 / N16 — bar narrowed to `== BuiltinTmux` | **KILLED** | `TestRegisterExternalRefusesReservedNamespaceDistinguishingValue` (the `ax.conpty` case) |
| X09 — bar narrowed to `== BuiltinConpty` | **KILLED** | `TestRegisterExternalRefusals` (the `ax.tmux` case) |
| X11 — bar deleted entirely | **KILLED** | both |
| X10 — bar narrowed to `== BuiltinTmux \|\| == BuiltinConpty` | SURVIVED — **confirmed equivalent** | n/a |

N16 was a round-2 survivor. Coverage is 2 of 2 distinguishing values, with the
equivalence of the exact-set form established rather than left ambiguous.

## 4. Attack point 1 — "no exported path can reach these arrays"

The claim at `terminalbackend.go:388-392` was checked by **driving every exported
symbol**, not by reading. I wrote a five-case external-package probe against the
real exported API. All five pass:

| Probe | What it drives | Result |
| --- | --- | --- |
| P1 | `Resolve` and `RequireRestoreBinding` egress, both directions | pass |
| P2 | `New` retaining the caller's `protocolVersions` argument array | pass |
| P3 | `IDs` handing out interior state | pass |
| P4 | cross-built-in aliasing observable through exports (expected inert) | pass — inert, as designed |
| P5 | `RegisterExternal` ingress + egress-mutation → drift still reported as drift | pass |

Exported surface enumerated from `go doc -all`: `New`, `Registry.Resolve`,
`Registry.RequireRestoreBinding`, `Registry.RegisterExternal`, `Registry.IDs`,
`ParseID`, `DefaultForPlatform`, `CheckVersionTuple`, `CheckProviderDescriptor`,
`DigestFile`, the six `Is*` predicates, and the `Error` / `Registration` /
`TrustEntry` / `InstanceBinding` / `Kind` types. Only `Resolve` and
`RequireRestoreBinding` return package-owned slices; `IDs` builds a fresh slice;
`Registry` has no exported fields. **The claim holds.**

## 5. Attack point 2 — the white-box pins and their reachability bounds

`internal_pin_test.go` pins three arms no production entry point can deliver an
input to. Each bound checked against the source, not assumed:

| Pinned arm | Reachability bound | Verified |
| --- | --- | --- |
| `parseKind` vocabulary refusal | `RegisterExternal` refuses any non-external kind as untrusted **before** `observed.validate()` runs, and `New` constructs only `builtin_go` | yes — the kind gate is at terminalbackend.go:434, `validate()` at :437 |
| `executable_digest must be null` | `New` builds null digests; `RegisterExternal` refuses non-external kinds first | yes |
| `validatePlatforms` length-arm precedence | a >4 list over a 4-member vocabulary must also fail a later rule, so the unit level is the only place precedence is observable | yes |

The reachability statement is per-arm in the doc comments of two of the three
tests and in the file header for the third (`parseKind`), which the header names
explicitly along with the black-box test that covers the production path
(`TestRegisterExternalRefusesUnknownKindAsUntrusted`). Substantively satisfied.

## 6. Attack point 5 — regression over the round-2 baseline

Round 2 measured 59 mutants at 52 killed / 7 survived. I re-ran **the whole
round-2 definition set** plus 11 new round-3 mutants.

| Metric | Value |
| --- | --- |
| Defined | 71 |
| Not applied (no measurement) | 2 — N12, T01 (target text absent; both pre-existing) |
| Build error (no measurement) | 1 — M32 (a mutant that does not compile is not a red) |
| **Measured** | **68** |
| **Killed** | **59** |
| **Survived** | **9** |
| Confirmed-equivalent survivors | 4 — N14, X01, X02, X10 |
| Effective measured / effective survivors | 64 / 5 |

**Every mutant the round-2 reviewer killed is still killed** — 50 of 50
re-applied and re-killed. Nothing was weakened to close a survivor. The two
round-2 survivors that the rework targeted (R03, N01) and the third (N16) are all
dead.

Harness discipline: byte-copy backup of every touched file, occurrence count
`== 1` before substitution, mutant-present / original-absent check after,
byte-equality restore assertion between mutants, `git status --porcelain` clean
after every batch.

## 7. Findings — none blocking

### NB1 — the egress enumeration in the comment is short by one accessor

`terminalbackend.go:388-392` supports "no exported path can reach these arrays"
with the list "(Resolve clones, RegisterExternal clones on ingress, IDs returns
strings)". `Registry.RequireRestoreBinding` is a fourth exported accessor that
returns a `Registration` and it is not in the list. The **conclusion is true** —
I verified it by running (probe P1/P5) — because `RequireRestoreBinding`'s single
return path is `return registry.Resolve(bound)` and Resolve's clone is pinned
twice (R01 wholesale, R01b member-wise half-fix). But the enumeration reads as
complete and is not. One clause.

### NB2 — `RequireRestoreBinding` egress is unpinned (measured survivor X03)

Replacing `return registry.Resolve(bound)` with a direct map read that skips the
clone leaves the whole suite **green** (X03, presence-checked). This reproduces
round-1 F1 on the second exported record-returning accessor.

Stated bound, and why this is not blocking:
- The behaviour is **correct today** and I proved it by driving it, not by
  reading — probe P1 mutates what `RequireRestoreBinding` returns and the
  registry is unchanged; probe P5 shows a genuinely drifted re-registration is
  still refused as drift after that mutation.
- There is exactly **one** clone site on this path (`cloneRecord` in `Resolve`)
  and it is measured in both the wholesale and member-wise directions. X03 does
  not narrow a guard across its domain; it deletes a delegation and reimplements
  the callee. A mutation-adequacy bar that requires killing arbitrary function
  rewrites is unbounded.
- `RequireRestoreBinding` has **no production caller**. Verified: the only
  production importer of this package is `internal/config/validation.go`
  (`grep -rl "internal/terminalbackend" --include='*.go'`), and it calls only
  `ParseID`, at lines 660 and 694. The repository builds no `ax` binary.

The risk it leaves is a future refactor that inlines the map read to avoid the
double `ParseID` — plausible, and it would restore F1 silently. One assertion
closes it. Recorded for whichever leaf first gives this accessor a caller.

### NB3 — a stale clause in a test docstring

`TestRegistryCopiesSlicesAcrossItsBoundary`'s doc comment
(terminalbackend_test.go:473-479) still reads "…and the two built-ins must not
share one protocol backing array." That subtest was moved out of this function to
`internal_pin_test.go` in this very commit. The property **is** covered, by a
named test that reddens under R03, so a coverage audit reaches the right
conclusion by the wrong attribution. Prose the function no longer supports.

### NB4 — R13's kill is a hang, not an assertion

Neutralising the `DigestFile` regular-file guard (`if !info.Mode().IsRegular() &&
false`) registers as KILLED, but the mechanism is a **timeout**: the FIFO subtest
blocks forever in `os.ReadFile` with no writer. I measured this directly — under
the default `go test` timeout the package failed after **249.5s** with `signal:
terminated`, not with a `--- FAIL:` assertion. The detection is real (the suite
goes red and the guard's removal is observable), but it is a hang. Reported for
accuracy so the number is not read as an assertion kill.

### Carried forward from round 2, unchanged and still disclosed

- **N09** `parseKind` admits one extra plausible kind (`builtin_rust`) — survives.
  Stated bound: a closed vocabulary cannot be pinned against its complement by
  sampling, and no pinned kind set exists in `internal/catalog` to cross-check.
  The narrowing direction is covered (N10, R12 killed).
- **N13** `CheckVersionTuple` membership widened to a prefix match — survives. The
  distinguishing input is a prerelease list member; no production caller.
- **N15** `RegisterExternal` external-kind gate widened to admit `native_runtime`
  — survives; still refused downstream by the digest binding, only the clause
  changes.
- **canonicaljson `requireTerminalBackendID`** still carries its own copy of the
  grammar without the `ax.` reservation — pre-existing, disclosed at README:577-582.

### Newly confirmed equivalent this round

- **X01 / X02** — reverting **one** of the two `New` clones. `protocols` is
  already a defensive copy of the caller's argument and never escapes `New`, so a
  single-site revert leaves the two built-ins independent and violates no claimed
  property. This is what the orchestrator's pre-verification observed.
- **X04** — `New` retaining the caller's `protocolVersions` argument. Survives,
  and the effect of removal is that `sort.Strings` mutates the **caller's** slice
  in place; both built-ins still receive independent clones, so no registry
  state, gate, or refusal is weakened. A caller-visible side effect, not an
  integrity property.
- **X10**, **N14** — see §3 and the results JSON.

## 8. Prior decisions

D1 (`required_capabilities` keeps its empty default) and D2 (§6.2 not extended to
third-party IDs) were adjudicated at round 1, upheld at round 2, and are not
relitigated. This commit does not touch either.

## 9. Ownership and §6.2

Commit `9481a20` touches no traceability or README file, so the round-2 F5
discharge is unchanged and was re-verified by running rather than assumed:
`tracecheck` exit 0 at `acceptance_cases=74 clauses_discharged=17/403`, the
declaration `TestDecodeRefusesNonConptyBackendOnNativeWindows` is registered at
`internal/traceability/ownership.v0.5.0.json:210`, README:1964-1994 names both
arms, and the negative arm is real — N18 (narrowing the Windows guard to WSL2)
kills exactly `TestDecodeRefusesNonConptyBackendOnNativeWindows`.

The leaf still claims no new ownership. Confirmed honest by the same measurement
round 1 used: `ParseID` at two live entry points is the only production reach,
and it is pinned (M25, M27, R15 all killed). Everything else in the package has
no production caller.

## 10. Why accept

- No production defect at any input I drove.
- All three round-2 blocking findings closed, each killed by a specific named
  test, with neighbours probed in both directions and equivalence controls
  established rather than assumed.
- 50 of 50 previously-killed mutants re-killed — nothing weakened to close a
  survivor.
- The central round-3 claim was verified by running across the whole exported
  surface, not read.
- What remains is two prose inaccuracies (NB1, NB3), one measured-but-bounded
  coverage gap on an accessor with no production caller (NB2), and one reporting
  accuracy note (NB4). None of them makes a gate admit what it must reject, and
  none of them is worth a fourth producer/reviewer cycle on a non-final leaf that
  will be checkpointed rather than integrated.

Non-final leaf: checkpoint, do not integrate. This reviewer supplies no
`commit_ack`; the commit-owning mover makes the `done` transition.

## 11. Reviewer hygiene

No production code was modified by this review. Every mutant and probe was
reverted, my probe file was deleted, and the tree was verified after every batch:
`git status --porcelain` empty, `HEAD` unchanged at `9481a20`, full suite green
(14 packages), `go vet` exit 0, `gofmt -l internal/` empty.

## Artifacts

- `TASK-260830-2xdt8t_review3-verdict.md` (this file)
- `TASK-260830-2xdt8t_review3-mutation-results.json` — 71 mutants, per-mutant status, kill mechanism and equivalence annotations
- `TASK-260830-2xdt8t_review3-mutant-definitions.json` — exact edits, replayable
- `TASK-260830-2xdt8t_review3-probe_test.go.txt` — the exported-surface aliasing probe
- `TASK-260830-2xdt8t_review3-baseline-test.log`, `TASK-260830-2xdt8t_review3-tracecheck.log`
