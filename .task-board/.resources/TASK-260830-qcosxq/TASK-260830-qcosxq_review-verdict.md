# TASK-260830-qcosxq review round 2 — verdict: changes requested

CR `CR-TASK-260830-qcosxq-2` rev 2, base `de51363`, candidate tree
`81f7c566ffac70bae8ffb5391f27b882ee609640`, `repository_delta=present`.
Working tree verified byte-identical to the candidate OID before the battery
and after every mutant (harness restores from a `cp` backup and re-derives the
tree OID; it never runs `git checkout`).

## Headline

| Measure | Leaf 1 (r4) | This leaf (r1) | This leaf (r2) |
| --- | ---: | ---: | ---: |
| Mutants killed / measured | 80/83 | 44/66 | **89/106** |
| Confirmed equivalent | — | 4 | **6** |
| Real survivors | 3 | 18 | **11** |
| Resurrections of prior kills | 0 | 0 | **0** |

106 rows on a traversal I re-derived from production source this round (not
round 1's row set, so the ratio is not directly comparable to 44/66 — what is
comparable is the disposition of round 1's named survivors, below).

The rework is good and honest. Every one of round 1's **18** named real
survivors is closed: 16 killed, 2 producer-declared equivalent and confirmed
equivalent by me. Gates all green, coverage claim exact, `tracecheck`
unchanged. The blocking problem is the one the round was asked to answer.

## G-A (blocking) — the arm enumeration is still manual, and it is paying rent

**Answer: still manual.** The rev1→rev2 delta is test-only —
`internal/provhost/inventory_test.go` is byte-identical between the two
candidate trees, and no production file changed. Named findings got named
tests; nothing enumerates refusal branches from production.

Measured directly by planting a new refusal branch and asking whether anything
reddens:

| Planted branch | Shape | Suite |
| --- | --- | ---: |
| G2 — new `failProtocol` **call site** in `DecodeResponse` | new constructor call site | **KILLED** — audit: "refusal call sites without an exercised negative path: protocol.go:388" |
| G4 — new `failInvalid` **call site** in `encodeFrame` | new constructor call site | **KILLED** by the same audit |
| G1 — new `&frameFault{detail: "member is too long", member: "protocol_version"}` in `checkResponseMembers` | new **arm** through an existing site | **SURVIVED** |
| G3 — new `integrity("status state is too long", state)` in `DecodeStatusOutcome` | new **arm** through the single `failIntegrity` site | **SURVIVED** |

So the mechanism enumerates *constructor call sites*, exactly as round 1
described, and is still blind to arm-slide by construction: all 10 `frameFault`
literals funnel through two `failProtocol` sites, and all 18 `integrity(...)`
arms funnel through one `failIntegrity` site at `status.go:77`.

Arm coverage, as a measured ratio rather than prose: **51 of 52** production
refusal arms are named by at least one assertion. One is not (X19, below); the
`failMismatch` arm is identified by its `observed` detail rather than its human
detail, which is fine.

**But arm identity was never the whole gap.** `parseMajor` has five *rejection
branches*, and they are not refusal arms at all — they are classification
branches that route between two arms. No amount of member/detail assertion
reaches them; each needs its own fixture, enumerated by hand. Round 1 named
four. The producer closed exactly those four. The fifth is still open:

### F30 (blocking) — `parseMajor`'s non-digit **major** branch is deletable, and flips exit 13 → exit 6

`internal/provhost/protocol.go:346-352`. Narrowing `if digit < '0' || digit > '9'`
to a never-true condition leaves the whole suite green. Driven through
`DecodeResponse`:

| `protocol_version` | baseline | mutant |
| --- | --- | --- |
| `"a.0.0"` | `provider_protocol_error` exit **13** | `incompatible_protocol` exit **6** |
| `"2a.0.0"` | `provider_protocol_error` exit 13 | `incompatible_protocol` exit 6 |
| `"-1.0.0"` | `provider_protocol_error` exit 13 | `incompatible_protocol` exit 6 |
| `"+3.0.0"` | `provider_protocol_error` exit 13 | `incompatible_protocol` exit 6 |

Identical class and identical consequence to round 1's F21: a malformed version
promoted to a recognizable major, reported through the opposite Section 15.1
branch with the opposite exit code. It hides for the same reason F21 hid —
every fixture with a non-numeric first character (`v2`, `a.b.c`, `3.b.c`) also
has a non-numeric *rest*, so the `M22` branch catches it first and the
misclassification collapses back to the same code. `.0.0` covers the empty
major, nothing covers a non-empty non-numeric one.

This is the round adding an nth named row and the (n+1)th arm still paying
rent, which is precisely what G-A asked about.

### F31 (blocking) — `stdoutCap`'s probe byte is load-bearing and unpinned; a one-byte narrowing admits a frame Section 7.2 forbids

`internal/provhost/runner.go:20`, used only at `runner.go:85`. The comment at
`runner.go:15-19` states the `+2` exactly: "one maximal frame, its line
terminator, and one probe byte, so a maximal frame followed by any trailing
byte is still distinguishable from a clean end." Nothing tests it. Both
directions survive; the narrowing is not cosmetic. Measured through production
`ExecRunner` + `Host.Call`, with a real plugin emitting an 8 MiB frame, a
terminator, and one junk byte:

| `stdoutCap` | outcome |
| --- | --- |
| `MaxFrameBytes + 2` (as written) | `provider_protocol_error: stdout carries more than one frame` |
| `MaxFrameBytes + 1` | **accepted**, `err=nil`, body 8 388 472 bytes |
| `MaxFrameBytes * 4` | accepted-as-baseline; 4x the memory bound, suite green |

A gate that admits what it must reject, with the suite green: "Stdout MUST
contain protocol frames only" (§7.2, pinned by
`TestSection72FramingIsPinned`). The existing `TestCallOversizeStdoutRefuses`
and the `extra`-line row both drive `scriptRunner`, whose `Result.Stdout` is
handed over directly and never passes through `readCapped`, so no test reaches
this constant at all.

## Round 1's survivors — all 18 closed

| Round-1 finding | Mutants | Round-2 result |
| --- | --- | --- |
| F21 `parseMajor` | M19, M20, M21, M22 | **4/4 killed** — each reddens `TestDecodeResponseRefusals/foreign_*` by *code*, naming the promotion, not by landing somewhere |
| F22 token floor | `<16`, `<8`, `<48` | **3/3 killed**; I added `<31` and `<33` — both killed. Floor pinned two-sided at 31 refused / 32 accepted |
| F23 `readCapped` | R09 (limiter), R07 (truncation) | **2/2 killed** by `TestReadCappedStopsConsumingAtTheCap` / `TestReadCappedEnforcesCap` |
| F24 request bound | M06c halve | **killed**; I added widen-2x and `>=` (exact-boundary narrow) — both killed. Response side re-checked: both killed |
| F25 arm identity | M09, M09a, S02, S02a, M27 | **5/5 killed**, and they fail by *naming the slid arm*: "error = …: not a provider envelope, want detail containing \"missing member\"" |
| F25 | M09b (drop `"ok"`) | **equivalent, confirmed** — the missing-`ok` pre-check at `protocol.go:403` fires first on every input; the list entry is unreachable |
| F26 exit plumbing | R17 single-site | **equivalent, confirmed** — I cleared the *other* site and it also survives; each alone is masked by the other |
| F26 | R17 both sites | **killed** by `TestExecRunnerReportsCrashExit/crash_exit`, reproducing round 1's exact `process_failed` → `protocol_error` flip through production `ExecRunner` |
| F27 terminator | R14 | **killed** by `TestExecRunnerWritesOneTerminatedFrame` |
| F28 README paragraph | — | **fixed**; the discovery paragraph is back above `## Provider Plugin JSONL Protocol` at README:418-424, and the contradictory six/three constructor counts no longer sit adjacent |
| F29 evidence claim | — | **restated** to what is pinned (arm identities, 31/32 token bytes, both exact frame bounds). Each clause of the new sentence reproduces under mutation |

## Resurrections: none

- 23 of 23 distinct provhost tests that killed a round-1 mutant still exist by
  name in the rev2 test files; the rewrite deleted no killer. The other 9
  round-1 killer tests live in `internal/provider`, which rev2 does not touch.
- The three inherited closers re-probed from scratch, not accepted from round 1:
  F18 — a production `init()` reassigning `ownerAttester` with the correct
  `func(os.FileInfo) (uint32, error)` signature turns
  `TestOwnerAttesterIsUnreassigned` red; F19 — `return nil,` → `return out,` at
  **all five** `Discover` refusal returns reddens
  `TestDiscoverRefusesRelativePluginDir`, `…RefusesDuplicates`,
  `…EnforcesTrustGatesAcrossSources`; F20 — a Windows-only type error passes
  `go build` (0), `go vet` (0) and `go test ./...` (0) and is caught only by
  `GOOS=windows go vet ./...` (1), the step this CR adds to CI.
- Every round-1 gate class I re-derived independently is killed. What I did not
  do is replay round 1's 66 rows by their original definitions — they were not
  recorded as a list. I re-derived a larger traversal instead and say so.

## Remaining real survivors (11 of 106; 6 more confirmed equivalent)

Beyond F30 and F31:

- **X19** — `integrity("status operation_id is not a string")`, `status.go:118`.
  The only production arm with no assertion anywhere. Narrowing it slides to the
  UUIDv7 arm, same code, different detail, suite green. `status_test.go:188`
  covers the malformed operation_id but not the non-string one.
- **AS6** — blanking the `status_state` detail on the token-floor arm survives.
  `status_state` is asserted on exactly **one** row (the unknown row,
  `status_test.go:90`); `requireLocalRefusal` does not check it, so the other
  17 integrity arms carry an unasserted detail.
- **S4b** — `stdoutCap` widened 4x (see F31).
- **S11** `WaitDelay = 0`, **S13** the `ctx.Err()` pipe-close block at
  `runner.go:101-112`, **S12** the short-write check at `runner.go:93`. All three
  guard detached-descendant and partial-write paths that no test constructs.
  Stated bounds, not defects.
- **U2** — removing the process-group `SIGKILL` `Cancel` from
  `runner_unix.go:19-21` (keeping `Setpgid`) leaves the suite green. The comment
  at `runner_test.go:484-487` claims the 300 ms timing proves the whole group
  died — it does not: with the group kill removed I measured **300 ms** on the
  same fixture, and on a `(sleep 2; touch marker) & sleep 30` plugin the
  grandchild died in **both** arms. What the test does pin is that `Setpgid` and
  `Kill(-pid)` are consistent (U1, `Setpgid: false`, is killed at 2.31 s). Whether
  the group kill buys anything beyond that is **unknown** on macOS from my
  probes — I could not construct a distinguishing case, and I am reporting that
  as unknown rather than inferring it either way. The comment's specific claim
  should be corrected to what it measures.

Confirmed equivalent (no observable outcome change, each checked rather than
assumed): M09b/`ok`-in-required, R17 single-site, the closing-delimiter arm at
`protocol.go:293` (unreachable after `More()` returns false),
`splitFrame`'s oversize flag on the newline branch (measured: the frame reaches
`DecodeResponse`'s identical bound and produces the identical arm —
`provider_protocol_error` / "frame exceeds 8 MiB" / member `""`), `isNull`
widened to empty raw, and the base64 error disjunct (`raw` is nil on error, so
the length arm refuses with the same detail).

## Producer's reported facts — independently re-verified

| Claim | Verified |
| --- | ---: |
| `go test ./... -count=1` exit 0, 15/15 | yes |
| `-race` on provhost exit 0 | yes |
| `go vet` and `GOOS=windows go vet` exit 0 | yes |
| `go build` on host / linux / windows exit 0 | yes |
| `gofmt -l` clean | yes |
| `tracecheck` exit 0, 17/403 clauses, 74 acceptance cases | yes, exact |
| provhost coverage 88.1% | yes, exact |
| rev2 is test-only, production untouched | yes — `git diff 2c7022b 81f7c56` touches only LOGBOOK, README, and the three `_test.go` files |
| the sweep no longer restores with `git checkout` | yes — `TASK-260830-qcosxq_round2-verify-sweep.sh` restores by `cp` from a backup, asserts the mutant was present, and byte-verifies the restore |
| 18 killed / 2 declared equivalent in the producer sweep | consistent with mine; both equivalence declarations hold |

## Standing bounds — unchanged, and one correction stands from round 1

- **`internal/provhost` does not import `internal/provider`.** Unchanged: the
  only reference is the comment at `runner.go:44`. My round-1 brief's
  expectation that this leaf would be the first production caller was wrong;
  leaf 1's bound is untouched.
- **`internal/provhost` has no importer anywhere in the tree**, and there is no
  `cmd/`. Two orphaned packages, no entry point. Every `Host.Call` assertion is
  made from a test.
- **Windows behaviour is compile-verified, not executed** — now actually
  enforced in CI by the `GOOS=windows go vet` step this CR adds.
- **The `Inspect`→`ReadFile` TOCTOU window is unprobed.** Unchanged.
- `doc.go`'s non-retryability claim is now asserted on every refusal row through
  both helpers — that round-1 bound is closed.

## What this method cannot see

Rows are derived by traversing production source, so a §7.2 duty with no code
leaves no row and is invisible here; `tracecheck` reports 17/403 clauses
discharged, and this review measures the quality of what was built, not the
size of what was not. Concurrency beyond `-race` on the existing tests, and real
Windows execution, are also outside it.

## Verdict

**changes requested → `to-dev`.**

Blocking: **F30** (a fifth `parseMajor` branch deletable with an exit-code flip
— the round-1 F21 defect on the arm round 1 did not name) and **F31** (a
one-byte narrowing of `stdoutCap` that makes the host accept a frame followed
by junk, measured through the production entry point).

Structural (G-A): the enumeration is still manual. Two rounds have now each
missed a row by hand. Rather than adding a sixth named `parseMajor` fixture,
mechanise the enumeration the way the call-site audit already is: derive the
`frameFault` literals and `integrity(...)` details from the AST and require
each to have been *asserted*, not merely constructed — G1 and G3 are the two
mutants that must go red. `parseMajor`'s branches need their own enumeration
(they are classification branches, not arms), or a restructure that makes them
enumerable.

Also in the same round, all small: **X19**, **AS6** (fold `status_state` into
`requireLocalRefusal`), and the `runner_test.go:484-487` comment correction.
**S11/S12/S13/S4b** are acceptable as stated bounds if written down as such.

Re-run bar for round 3: F30 and F31 killed; G1 and G3 red; no resurrection
among the rows this verdict records as killed; and the report given as a
killed/measured ratio over a production-derived traversal with equivalences
argued, not asserted.

## Harness

`.temp/TASK-260830-qcosxq/mut2/` — `drive.py` (apply exact-once / full-package
run / restore / re-derive the tree OID), `mut.py` (fails closed unless the
pattern matches exactly once), `restore.sh` (`cp` from a pristine backup, never
`git checkout`, and refuses to report OK unless the tree hashes back to
`81f7c566`), `batch{1..4}.json` (the row set), `logs/` (one per mutant). Every
run is full-package: `inventory_test.go`'s audit only fires when `-run` is
empty, and it alone killed S3, S6, G2, and G4.
