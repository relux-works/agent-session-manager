# TASK-260830-1r9wrr — review verdict, CR-TASK-260830-1r9wrr-1 rev1

**Verdict: changes requested → `to-dev`.**
**repeat-of: none** (first revision of this leaf).

Candidate re-derived at review time: detached-index `git read-tree HEAD` + add of
the 16 CR paths → tree `95b2f9e58f00ee9aea5aba32c548bcc1e44e5a34`, equal to the
recorded candidate tree; `git ls-tree -r` confirms all 14 new `internal/sessstate`
files are in that tree (not an index that silently dropped the untracked package).
Working tree restored to that exact OID after every plant and mutant below —
re-verified as the last action of this review.

The production code is largely sound and the division of authority is real. Four
findings block acceptance: two are derivation defects inside the AC's own scope
that no test pins, one is a census whose denominator is a single function name
(a behaviourally wrong plant walks through all three censuses and the whole
suite), and one is a wrong published number in the shipped `README.md`.

---

## What I drove myself (exit codes observed in this run)

| Gate | Command | Exit |
| --- | --- | --- |
| package baseline | `go test ./internal/sessstate/ -count=1` | 0 |
| repo-wide suite | `go test ./... -count=1` | 0 (25 packages ok, 0 FAIL) |
| build | `go build ./...` | 0 |
| vet | `go vet ./internal/sessstate/` | 0 |
| windows | `GOOS=windows go build ./...`, `GOOS=windows go vet ./...` | 0, 0 |
| format | `gofmt -l internal/sessstate/` | no output |
| tracecheck | `go run ./internal/traceability/cmd/tracecheck` | 0 |
| coverage | `go test ./internal/sessstate/ -cover` | 0 — 91.7% of statements (claim matches) |
| race | `go test -race ./internal/sessstate/ ./internal/sessrepo/ -count=1` | 0 |
| reviewer battery | 13 mutants, harness `.temp/TASK-260830-1r9wrr/review/rmutate.sh` | 10 killed, **3 survived** |
| census plants | 5 plants, harness `.temp/TASK-260830-1r9wrr/review/plants.sh` | 4 killed, **1 survived** |
| behavioural probes | 7 probes through `Reduce`/`Project` | 3 confirmed defects, 3 refuted, 1 bound |

**Not re-run by me, accepted as the producer reported it:** the 4-group `-race`
split over all 25 packages (I ran 2 of them), and the six pinned `cigate`
contract selections. The attached rev1 validation log is truncated (65 536 bytes,
~5.2 MB dropped) and was not used as evidence for anything above.

---

## F1 — `resolveWinner`: a duplicated off-chain union lease erases the
## `union_supersedes_chain` conflict and the `divergent_history` warning

`internal/sessstate/sessstate.go`, `resolveWinner`, the `case 0:` arm:

```go
case 0:
    if _, ok := onChain[lease.LeaseID]; !ok {
        offChainWinner = false
    }
```

`Compare(lease, winner) == 0` with `lease.LeaseID` **not** on the chain is only
reachable *after* `winner` has already been moved to an off-chain union lease by
an earlier iteration. So the arm's only reachable effect is to clear the flag
that a previous iteration correctly set.

Driven through the production `Reduce` entry (chain `session.created` +
`provider.launched`, head lease `(1, aaaaaaaa-…)`):

```
Union: [(3, cccccccc-…)]              -> winner (3, cccccccc-…), conflicts=1
                                          kind=union_supersedes_chain rule=authoritative-state-unchanged
Union: [(3, cccccccc-…), (3, cccccccc-…)]
                                      -> winner (3, cccccccc-…), conflicts=0, warnings=[]
```

Same winner, conflict and warning gone. AC row 3 requires a conflict where both
sides are individually valid to be "refused or resolved by a named rule"; here it
is silently dropped, and the `divergent_history` warning `emitWarnings` derives
from it disappears with it.

**Mutant R2** — delete the arm body (`case 0: _ = onChain`) — **SURVIVED** both
gates (`behavioral=PASS full=PASS`). The branch has zero test coverage; its only
observable behaviour is the defect.

## F2 — `applyLocalLease`: a local lease that WON the tuple rule is reported `stale`

```go
if Compare(local, fold.projection.Winner) == 0 {
    return
}
```

The rule is documented three times as *lost*: the function comment ("a local
lease that lost the tuple rule stales the projection"), `Projector.Local`'s field
doc, and `README.md` ("a local lease that lost the tuple rule stales the local
projection the same way"). It is implemented as *not equal*.

Driven through `Reduce` (chain head `(1, aaaaaaaa-…)`, `Local: (9, cccccccc-…)`):

```
state="stale"  winner={Epoch:1 LeaseID:aaaaaaaa-…}  warnings=[stale_process]
```

The host holding the strictly greater lease is reported stale with a
`stale_process` warning — a wrong `SessionState` value under AC row 1 and a wrong
winner story under AC row 2.

**Mutant R1** — the candidate fix `Compare(...) >= 0` — **SURVIVED** both gates.
The suite does not distinguish the two, so nothing pins the current behaviour
either way. (Applying the candidate fix and seeing whether the suite notices is
the check that separates "intended" from "unmeasured": it is unmeasured.)

**F1 and F2 together refute the outcome's claim** "26 applied, 26 killed, 0
survivors. No surviving mutant exists, so no survival bound is owed." Both are
narrowing mutants on production gates inside AC rows 2 and 3; the battery's
12 narrowing rows touch `Compare`, the gap arm, the transition table, the retry
gates, the provider arms, the lease-linkage arms and the parked branch, but not
one of them reaches `resolveWinner`'s off-chain bookkeeping or `applyLocalLease`.
The denominator excluded the half of the winner derivation that carries the
defects.

## F3 — the event census's denominator is one function name; a second handling
## site with a wrong effect walks through everything

`deriveHandledEventTypes` (census_state_test.go) inspects only
`FuncDecl.Name == "effect"`. Plant **PA**: a new production file
`internal/sessstate/zz_plant.go` declaring

```go
func (fold *chainFold) effectExtra(event Event, what string) error {
    switch event.Type {
    case "session.reaped":
        return fold.step(StateIdle, what)
    }
    return nil
}
```

wired into the production switch by the one-line, line-count-preserving edit
`default: return nil` → `default: return fold.effectExtra(event, name)`.

Result: **SURVIVED** — `behavioral=PASS full=PASS`. An unknown v1 event type now
moves `running → idle`, directly contradicting the Section 5.2 inert rule this
package claims to enforce and its own
`TestReduceTreatsUnknownV1TypeAsInert` (which drives a different spelling,
`session.future_probe`). All three censuses stay green: the refusal census sees
no new `refuse` site, the state census sees no new spelling, and the event census
never looks outside `effect`.

So "24 handled types == the pinned v1 registry, both directions" is a claim about
one function, not about production. Fix by deriving the denominator from every
production event-type dispatch, or by failing closed when a second dispatch site
on `event.Type` exists.

The other four plants were correctly killed, so the fix is narrow:

| Plant | Shape | Outcome |
| --- | --- | --- |
| PA | fresh-spelling handler in a new production file, wired into `effect` | **SURVIVED** |
| PB | `var deny = refuse` in a new production file | KILLED (alias audit, `zz_plant.go:3:12`) |
| PC | `var parseDigest = scalar.ParseDigest` (alias-bound owner delegation) | KILLED (alias audit, `zz_plant.go:5:26`) |
| PD | `const StateReaped State = "reaped"` in a new production file | KILLED (`TestStateSpaceIsCensused`) |
| PE | direct `refuse(ErrIntegrity, …)` at a new production site | KILLED (`TestRefusalSitesAreCensused`) |

## F4 — the published refusal-census denominator is wrong in the shipped README

`README.md:1763-1764` and the outcome both state:

> the 67-site refusal inventory … (39 in `sessstate.go`, 26 in `decode.go`, 2 in
> `project.go`)

The roster in `census_state_test.go` has **69** rows — 39 / **28** / 2 — and
`TestRefusalSitesAreCensused` requires derived == roster in *both* directions and
passes, so the derived count is 69 and `decode.go` carries 28 sites, not 26.
This contradicts the outcome's own "every number below re-derived after the last
artifact". Re-derive the number in source and correct both `README.md` and the
outcome.

## F5 — neither harness control exercised either gate (instrument, not production)

```
X1: NOT_APPLIED (pattern count=0)
X2: COMPILE_FAIL
```

X1 is an intentionally absent pattern and X2 is a syntax break, so both exit
before `go test` runs. Nothing in the producer's evidence shows the harness can
report **SURVIVED** — the outcome that every one of the 26 KILLED rows is
measured against — and `README.md` advertises them as "plus NOT_APPLIED and
COMPILE_FAIL controls" without that distinction.

I supplied the missing control: **R0**, behaviour-neutral and line-count
preserving (`tailEventID` returning `"" + fold.tailID`) → **SURVIVED**. With
R3–R10 (8 compiling known-bad mutants, all `behavioral=FAIL`) the instrument is
sound in both directions — but that is my evidence, not the delivered evidence.
Ship a real pair: one neutral mutant that must survive, one known-bad that must
redden, both reaching the gates.

## F6 (minor) — `scalar.ParseUUIDv7` is an unmeasured audit row and an
## unsupported doc claim

`doc.go:36-37` and `README.md:1775` name `scalar.ParseUUIDv7` among the watched
owner delegations "for digest and UUID grammars". It has **zero** occurrences in
`internal/sessstate` production (`environ.CheckUUIDv7` does that work). The
alias-audit row can never fire, and the documentation states a delegation the
package does not make.

## F7 (minor) — an empty chain skips union validation entirely

`Reduce` returns `creating` before `resolveWinner` when `len(input.Events) == 0`,
so `Input.Union` and `Input.Local` are neither validated nor applied:

```
Union: [{4, "not-a-uuid"}, {}]  ->  state="creating"  winner={0 ""}  err=<nil>
```

The same union on a non-empty chain refuses `ErrInvalidEvent` at
`sessstate.go:920` / `:923`. Make it a gate or a stated bound in `doc.go`.

---

## Reviewer mutation battery (13 mutants; 10 killed, 3 survived)

| ID | Mutation | Result |
| --- | --- | --- |
| R0 | neutral: `tailEventID` returns `"" + fold.tailID` | **SURVIVED** (negative control, as intended) |
| R0b | `if found.Parked` → `if found == nil \|\| found.Parked` | SURVIVED — my error, the arm is unreachable (`found == nil` already returned); a second neutral control, not a positive one |
| R1 | `Compare(local, Winner) == 0` → `>= 0` | **SURVIVED** — F2 |
| R2 | delete the `case 0:` off-chain reset in `resolveWinner` | **SURVIVED** — F1 |
| R3 | `staleSources` admits `stopped` | KILLED — `TestProjectLeavesStoppedPastTakeoverUnstaled` |
| R4 | transition table admits `running → stopped` | KILLED — `TestReduceRefusesUnlistedTransitions/running-to-stopped` |
| R5 | `noteCheckpoint` keeps the oldest instead of the newest | KILLED — `TestProjectDerivesFullLifecycle` |
| R6 | provider version recorded even on record mismatch | KILLED — `TestProviderMismatchKeepsRecordPinned` |
| R7 | local role inverted | KILLED — `TestProjectDerivesFullLifecycle`, `TestProjectIsIdempotentAcrossReads` |
| R8 | `stale_process` warning suppressed | KILLED — `TestProjectDerivesLocalStale` |
| R9 | epoch-gap conflict only past a jump of 3 | KILLED — `TestWinnerResolutionGap` |
| R10 | losing-branch conflict dropped for epoch-1 rivals | KILLED — `TestWinnerResolutionTie`, `…TieOrdersBytewise`, `…PastEpochTie` |

Every row above compiles and reaches both gates, so the denominator is 13 of 13
applied; killed-over-applied is 10/13, with the three survivors named.

---

## Checked and clean (probed, refuted as findings — do not re-litigate)

- **`task_board.launched.state` and `session.stopped.closure_kind` are not admit
  holes.** Both `else` arms looked ungated on reading, but the canonical owner
  closes them upstream: `core_records.go:859` `enum("state","running","idle")`
  and `core_records.go:742` `requireEnum(…,"checkpointed","bootstrap_abort")`.
  Fixtures carrying `"tombstoned"` / `"crashed"` are rejected by
  `CalculateObjectIdentity` before they can reach `Reduce`. The `doc.go` claim
  that the payload members `Reduce` reads were admitted by the closed shape holds.
- **Multiple predecessors on a non-first event are correct, not a divergence.**
  Section 5.2 declares predecessors as "a sorted array of one or more digests"
  and `sessrepo/chain.go:191` applies the identical
  `containsDigest(predecessors, tail)` rule. `Reduce` mirrors its owner exactly.
- **AC row 9 crash points sit inside the window they claim.** All four
  `armCreateStep` tests assert `injector.RemainingArmed()` is empty after the
  faulted `CreateSession`, so the armed point provably fired; the four targets
  are the four `CreateStep` boundaries (`SessionDir`, `Record`, `EventsDir`,
  `Chain`), one per boundary, and each of the first three heals on the identical
  retry.
- **Purity holds one step out from the producer's own vectors.** On a 5-event
  chain with a forced-lease succession, a two-entry union and a local lease: 64
  concurrent `Reduce` calls all `reflect.DeepEqual` to the sequential result;
  `Input` (including `Events` and `Union`) is unmutated by the call; and
  tampering with the returned `Conflicts`/`Warnings` slices does not leak into a
  later call — no aliasing across invocations. `-race` on the package: exit 0.
- **The untouched-package claims hold.** The CR delta is exactly 16 paths
  (`LOGBOOK.md`, `README.md`, 14 files under `internal/sessstate/`); no
  `internal/sessrepo` or `internal/provhost` file is touched, and no package in
  the repository imports `internal/sessstate` yet.
- **AC-row coverage: 10 of 10 rows name a driving test that exists and is green**,
  including `TestNoProductionPathAttestsProviderIdentityBinding`
  (`internal/provhost/identity_test.go:95`) for row 10 and `tracecheck` exit 0.
  The ratio is honest; F1/F2 are holes *inside* rows 2 and 3, not missing rows.

---

## What to do next

1. Fix `resolveWinner`'s `case 0:` arm (F1) and `applyLocalLease`'s comparison
   (F2), and pin each with a test that fails on the current code — a duplicated
   off-chain union entry, and a local lease strictly greater than the winner.
   Verify one step away from the finding vector, not only at it.
2. Widen the event census denominator so a second production dispatch on
   `event.Type` cannot be invisible (F3), and keep PA as a control plant.
3. Re-derive and correct the 67/26 numbers in `README.md` and the outcome (F4).
4. Add a real pair of harness controls that reach the gates — one neutral mutant
   that must survive, one known-bad that must redden (F5).
5. Fix or drop the `scalar.ParseUUIDv7` claim (F6) and gate-or-bound the
   empty-chain union path (F7).
6. Extend the battery's narrowing rows to the union/local-lease derivation; the
   current denominator excludes it.
