# TASK-260830-qcosxq round-3 review verdict — ACCEPTED

Reviewer run `RUN-260905-f6e0ac`. CR `CR-TASK-260830-qcosxq-3` revision 3,
base `de51363`, candidate tree `4e186ca`, `repository_delta=present`,
16 changed paths. Everything below was measured in this run, in the story
worktree, at that exact tree.

| | Round 1 | Round 2 | **Round 3** |
| --- | ---: | ---: | ---: |
| Mutants re-measured | 66 | 106 | **112 re-runs + 13 novel probes + 8 leaf-1 probes** |
| Killed | 44 | 89 | **102 / 112** |
| Survivors | 22 | 17 | **10** |
| — of them real (not equivalent) | 3 | 11 | **4** |
| Resurrections | — | 0 | **0** |
| New survivors introduced | — | 0 | **0** |

Verdict: **accepted**. Both round-2 blocking findings are closed by
production-driven tests I reddened myself, the derived arm inventory holds
against six of eight novel attack shapes, and the two shapes it does not see
are recorded below as measured, named bounds — not as prose.

---

## G-A — attacking the derived refusal-arm inventory

### The ratio, and how the denominator was established

**60 / 60.** The derivation reports `60 derived arms across 6 production
files`, and every one carries a witness.

The denominator was established **independently of the derivation under
test**, by grepping production source and counting the four shapes by hand:

| Shape | Independent count | Method |
| --- | ---: | --- |
| `&frameFault{...}` literals, distinct `(detail, member-source)` | 10 | 15 literals in `protocol.go` (lines 266–331), collapsed by hand to 10 distinct pairs (`not a JSON object\|""` appears 6×) |
| `integrity("literal")` calls | 17 | `grep -o 'integrity("[^"]*"' status.go \| sort -u` → 18 rows, minus the `"status body is "` wrapper |
| wrapper expansion over `decodeStrictObject` details | 3 | the three details reachable in `decodeStrictObject`: not a JSON object / duplicate member / trailing data after the object |
| `parseMajor` rejection branches | 5 | read directly from `protocol.go:373-399` |
| `failX("literal", …)` constructor calls, distinct `(ctor, detail)` | 25 | `grep -o 'fail\(Invalid\|Protocol\|Mismatch\|Process\|Timeout\|Integrity\)("[^"]*"' \| sort -u` |
| **total** | **60** | equals the derivation exactly |

So the coverage claim is not a number compared against itself: an independent
count of what production declares lands on the same 60.

### Does the walk see files added after it was written?

**Yes — measured, not reasoned.** The scan is a runtime `os.ReadDir` of the
package directory, and it already reports 6 production files (including
`runner_windows.go`, which the host build never compiles). I created a
seventh, `internal/provhost/planted_new_file.go`, holding one new
`&frameFault{detail: "planted new file rule"}`:

```
refusal arm coverage domain: 61 derived arms across 7 production files
derived refusal arm(s) with no witness at the production entry:
      frame|planted new file rule|""
```

File removed afterwards; package verified byte-identical to the candidate tree.

### The census floor — settled

`refusalArmCensusFloor = 60` is a hand-written literal, and the suspicion was
that it can be edited down to match a broken derivation.

- **`GA1` — floor 60 → 1, nothing else: SURVIVED.** The floor can drift below
  the real arm count and nothing fails. That much is true.
- **But the floor is not the defence.** I re-ran every truncation shape with
  the floor pinned at 1, so the floor could not fire:

| Probe (floor forced to 1) | Result | What caught it |
| --- | --- | --- |
| `GA2` skip `status.go` in the scan | RED | `TestWitnessedArmsAreAllDerived` — 20 orphaned integrity witnesses |
| `GA8` existing `failProtocol` literal → local variable | RED | orphan `ctor\|failProtocol\|frame is not UTF-8` |
| `GA10` existing site called through a constructor alias | RED | orphan `ctor\|failInvalid\|host has no runner` |
| `GA4` new unexercised `failIntegrity` site | RED | `inventory_test.go` site audit: `status.go:84` has no exercised negative path |

The reverse direction is an independent second measurement against a witness
set the derivation does not produce, so **the round-2-of-the-sibling-Story
defect (a ratio compared against itself) does not exist here.** The floor is a
redundant early tripwire. Lowering it costs nothing today because every
truncation I could build still reddens without it.

### Shapes the AST walk cannot see — two found, both measured

Six of my eight novel shapes reddened, including the two the brief named:

| Probe | Shape | Result |
| --- | --- | --- |
| `GA5` | `frameFault` built inside a helper from parameters | **RED** — lands as `frame\|expr:&frameFault{detail: detail, member: member}`, unwitnessed |
| `GA7` | detail assigned by a `switch` into a shared variable | **RED** — new `expr:` obligation *and* orphaned `frame\|unknown member\|name` witness; both directions |
| `GA2` | a production file the walk never visits | **RED** |
| `GA4` | new arm through `failIntegrity` (the 6th constructor, **absent from the arm walk's 5-name switch**) | **RED**, but by the *sibling* gate, not this one |
| `GA10` | existing arm re-routed through a constructor alias | **RED** |
| `GA8` | existing literal detail hoisted to a variable | **RED** |
| `GA9` | **alias-minted arm at a site no test exercises** | **SURVIVED** |
| `GA11` | **a one-line wrapper fronting a constructor** | **SURVIVED** |

**`GA11` is the real one.** I planted

```go
func refuseFrame(detail string) (Response, error) {
	failure, err := failProtocol(detail, "")
	...
}
```

routed the existing `"frame exceeds 8 MiB"` refusal through it (so the
wrapper's single constructor site is exercised and the arm key survives from
`runner.go`), and added a second, brand-new arm `refuseFrame("planted helper
family rule")` with no witness and no negative test. **Full suite green.**

Why both gates miss it: the arm walk keys constructor arms on a literal first
argument at a call whose `Fun` is one of five hard-coded idents; `refuseFrame`
is not one, and the wrapper's own call passes a non-literal. The site audit
derives *sites*, and the wrapper is one site — already exercised. So a wrapper
collapses N refusal rules into 1 required negative test.

`GA9` is the same hole reached through `alias := failX`, and only bites when
no test exercises the arm at all.

**Bound, stated plainly, because the tree's own comment does not state it:**
`refusal_arm_inventory_test.go`'s header says "Non-literal shapes land as
`expr:<source>` obligations rather than passing silently", and the round-3
evidence repeats it as "never silent". That is true for `frameFault` fields
and for `integrity(...)` — I reddened both — and **false for refusal
constructors**, where the inline comment 3 lines below correctly says the
opposite ("Non-literal first arguments are the fault conduits … there is no
separate obligation here"). The header generalises past what the code does.

I am **not blocking on this**, and I want the reasoning on record rather than
implied:

- No production shape in this tree triggers it. The 60/60 claim is true today,
  verified against an independently counted denominator.
- The README's public claim is correctly **scoped** — it says "every frameFault
  literal, integrity detail, and parseMajor classification branch", which is
  exactly the set that does fail closed. It does not overclaim to a reader.
- Behavioral coverage does not rest on this gate: 102 of 112 behavioral mutants
  die on entry-point tests independently of it.
- What is at risk is a **future** refactor that fronts refusals with a helper.
  That refactor is itself reviewable, and it now has a named probe (`GA11`)
  and a reproducible harness attached to reproduce the hole in one command.

The next producer to touch this file should either extend the derivation one
transitive step (treat a function that calls a constructor with a non-literal
first argument as a conduit, and derive literal-argument calls **to it** as
arms) or correct the header comment to name the conduit exception. Either is a
few lines; neither is worth a fourth round against a blocked sibling leaf.

---

## G-B — resurrections and the remaining survivors

All 112 round-1/round-2 battery rows re-planted at rev3 (exact-once plant,
presence check, targeted run, `cp` restore, sha256-verified byte-identical
restore; zero unrestored). Plus 8 leaf-1 probes against `internal/provider`.

**Zero resurrections.** Every mutant killed in rounds 1–2 is still killed.
**Zero new survivors.** The 10 that survive are a strict subset of round 2's 17.

### What moved

| Round-2 finding | Mutants | Round-3 result |
| --- | --- | --- |
| **F30** — 5th `parseMajor` branch, exit 13→6 | M23 `a.0.0`, M22 rest-digit | **both killed.** 4 rows added (`a.0.0`, `2a.0.0`, `-1.0.0`, `+3.0.0`) driving `DecodeResponse` and asserting `provider_protocol_error` + member + detail + **exit 13**, plus 4 arm witnesses keyed on the branch condition source, so narrowing `\|\|`→`&&` rewrites the key and orphans all four |
| **F31** — `stdoutCap` probe byte unpinned | S4 narrow, S4b wide | **both killed.** `TestExecRunnerRefusesMaximalFramePlusJunk` builds an exactly-`MaxFrameBytes` frame + `\n` + one junk byte through a real `sh` plugin and `Host.Call` over `ExecRunner`; at `+1` the host **accepts the forbidden frame** (`want a failure, got nil`). The widening direction is held by a constant pin, and the test comment discloses that the wide direction is not behaviorally observable |
| **X19** — non-string `operation_id`, the one unasserted arm | X19 | **killed.** New `status_test.go` row + inventory witness asserting `status_state == ""` |
| **AS6** — `status_state` asserted on exactly one integrity row | AS6 | **killed.** New `requireIntegrityRefusal` asserts code + detail + `status_state` + exit 9 + non-retryable; **21** call sites in the inventory |
| **U2 / the `runner_test.go:484-487` comment** | U2 | **comment corrected, unknown preserved** — see G-C |

### The 10 that survive

Six were **confirmed equivalent in round 2** and re-confirmed as unchanged
here: `A3` (`ok` in the required list, masked by the earlier pre-check),
`R17a` (single-site exit plumbing, each site masks the other), `D3` (closing
delimiter, unreachable after `More()`), `S1` (`splitFrame` oversize flag on the
newline branch — reaches `DecodeResponse`'s identical bound and identical arm),
`X16` (`isNull` widened), `X9b` (base64 error disjunct — `raw` is nil, so the
length arm refuses identically).

Four are **real, and all four were already stated bounds in round 2**:
`S11` (`WaitDelay = 0`), `S12` (short-write check), `S13` (the `ctx.Err()`
pipe-close block) — detached-descendant and partial-write paths no test
constructs — and `U2`, which is the honest unknown below.

**One anomaly, reported rather than smoothed over:** in the first pass, `S1`
came back RED once, failing `TestExecRunnerReportsCrashExit/clean_exit` with
`runner reported no result` — a fork/exec-shaped failure unrelated to the
mutation. I re-ran `S1` four more times (all SURVIVED) and ran
`TestExecRunner*` **15×** on the clean tree (15/15 green). One occurrence in
~130 process-spawning runs. I am recording it as an observed one-off, not as a
confirmed flake and not as a kill.

### Leaf 1 (`internal/provider`) — re-attacked, and strengthened by this CR

| Probe | Result |
| --- | --- |
| F18 — production `init()` reassigning `ownerAttester` to a uid-0 attester | **RED** on `TestOwnerAttesterIsUnreassigned` |
| F19 — `return nil,` → `return out,` at the plugin-dir refusal (single site) | **RED**: "returned 1 candidates alongside a refusal, want no partial set" |
| F19 — all five `Discover` refusal returns | **RED** on three tests |
| owner approval gate `&& false` | **RED** |
| `Canonicalize` skipped (symlink resolution bypass) | **RED** |
| attestation error swallowed in `OSSystem.Inspect` | **RED** |

This CR's two provider test edits are both **strengthenings**:
`TestDiscoverRefusesRelativePluginDir` now plants a distinct real candidate in
each absolute dir, so the no-partial-set promise is asserted against a
**non-empty** candidate set — which is what makes the single-site F19 mutant
die. `TestOwnerAttesterIsUnreassigned` is new. Nothing in leaf 1 was weakened.

---

## G-C — bounds, and the unknown that had to survive

**The unknown is intact.** `runner_test.go:537-542` now reads:

> What the group kill buys beyond `Setpgid`/`Kill` consistency is unknown on
> this platform: removing the `Cancel` alone still returns in ~300ms here,
> while disabling `Setpgid` delays past 2s — so this pins termination latency,
> not detached-descendant fate.

That is exactly what the measurement supports. No new test asserts
detached-descendant fate, and `U2` (removing the process-group `SIGKILL`
`Cancel`) correctly still **survives** — which is the honest outcome. `U1`
(`Setpgid: false`) is still killed. The rework did not replace the unknown
with a green test implying knowledge.

**Standing bounds — unchanged this round, re-verified here:**

- `go list -deps ./internal/provhost` shows `catalog`, `scalar`,
  `canonicaljson`, `axerror` — **no `internal/provider`**. The only mention is
  the comment at `runner.go:44`.
- **`provhost` has no importer** anywhere in the tree.
- **No top-level `cmd/`** (`ls cmd` → no such directory). The only `cmd`
  directories are `internal/catalog/cmd/cataloggen` and
  `internal/traceability/cmd/tracecheck`.
- Two orphaned packages; **every `Host.Call` claim is made from a test.**
  `Host.Call` is nonetheless driven as a real production entry point by 6
  witnesses and by the `ExecRunner` tests over real `sh` plugins.

---

## Producer's reported facts — independently re-run

| Claim | Verified |
| --- | ---: |
| production untouched this round | **yes** — per-path diff of the rev2 and rev3 patches: `protocol.go`, `runner.go`, `status.go`, `doc.go`, `runner_unix.go`, `runner_windows.go`, `ci.yml`, `LOGBOOK.md` and both provider test files are byte-identical; only `README.md` and 4 test files differ |
| `go build ./...` exit 0 | yes |
| `go vet ./...` / `GOOS=windows go vet ./...` exit 0 | yes |
| `GOOS=linux` / `GOOS=windows go build ./...` exit 0 | yes |
| `gofmt -l` clean | yes |
| `go test ./... -count=1` exit 0, 15/15 packages | yes |
| provhost coverage **88.4%** | yes, exact |
| provhost `-race` exit 0 | yes |
| `tracecheck` ok, **17/403** clauses, **74** acceptance cases | yes, exact |
| 60/60 derived arms witnessed | yes, and the denominator independently counted at 60 |
| 9 red-as-required in the producer's own sweep | consistent; I re-derived a larger traversal rather than replaying their rows |

## README and doc claims

`README.md`'s new pin paragraph is **correctly scoped** and every clause
reproduces: the arm-derivation sentence names the three shapes that do fail
closed; the `stdoutCap` sentence names the maximal-frame-plus-junk probe,
which I confirmed is behaviorally load-bearing; the six code/exit pins
(3, 13, 6, 13, 13, 9) are asserted against the catalog **and** the wire
`ExitCodeFor` in `TestStableCodesAreRegistered`. `doc.go` states its bounds
(1 MiB stderr cap, no idempotency receipts, non-retryability) without
overreaching.

One process note, non-blocking: **round 3 added no `LOGBOOK.md` entry.** The
leaf's rounds 1–2 entries are there; the round-3 mechanism decision (deriving
arms from source rather than listing them) and its stated bounds live only in
the board evidence artifact. Worth an entry when the orchestrator commits.

---

## Verdict

**Accepted.** `accept_cr(TASK-260830-qcosxq, revision=3)`. All 15 merged
checklist items are checked, both round-2 blocking findings are closed by
tests I reddened at the production entry, no round-1/2 kill resurrected, no
new survivor appeared, and the four remaining real survivors plus the two arm
inventory blind spots are recorded above as measured bounds rather than prose.

Element parks at `to-review`; the orchestrator commits the scope and makes the
`done` transition with `commit_ack=scope_committed`. This unblocks leaf 3.

## Reproducing this review

`TASK-260830-qcosxq_round3-review-battery.json` carries every row (id, file,
exit, verdict, restore check, seconds, failure tail) for the 112 re-runs, the
13 novel G-A probes, the 4 floor-forced combinations, the 8 leaf-1 probes and
the 4 `S1` repeats. `TASK-260830-qcosxq_round3-review-probes.tgz` carries the
harness and the probe definitions; each probe is a literal `old`/`new` source
pair, planted exactly once and restored byte-identically.
