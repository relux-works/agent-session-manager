# TASK-260830-32jeti — round-7 review verdict: CHANGES REQUESTED

Reviewer run `RUN-260905-d579e4`. Candidate `CR-TASK-260830-32jeti-5` revision 5,
base `57afcc6d`, candidate tree `a72c56666ff528db446884c5708af94b3940fe7b`.
Tree OID re-derived from the working tree at review start, after every mutant
battery, and at the end — matched exactly each time. Every probe below was
planted and reverted.

**Verdict: changes requested (`to-dev`). Three blocking findings.**

Round-6's **F1 is closed, and closed structurally** — the three switch-shaped
vocabularies now die to their own derivations. Round-6's **F2 is closed for the
case it named and opens a worse one**: `Host.Call` now overruns its deadline
without bound whenever any descendant of the plugin holds the inherited stdout
pipe. **G-B is clean.** Of the three **G-C** advisories, one is answered
correctly and two are answered with bounds that are **false as written**.

---

## F1 (BLOCKING) — the F2 fix removes deadline enforcement whenever a descendant holds stdout

`runner.go` replaced `StdoutPipe`/`StderrPipe` with `os.Pipe` pairs owned by
`Run`. That fixes the drain-vs-`Wait` race, and it removes the only thing that
ever unblocked the drain. After `command.Wait()` returns with `ctx.Err() == nil`,
`runner.go:132-133` does an unconditional `<-stdoutDone` / `<-stderrDone`. The
drain ends at EOF, i.e. when **every** writer of that pipe is gone — and a
detached grandchild that inherited the fd is such a writer. `Wait` returns as
soon as the plugin exits, so `Run` then sits on the channel for as long as the
grandchild lives, with no context selection and nothing left to kill it
(`exec.CommandContext`'s cancel goroutine has already stopped).

`Host.Call` over `ExecRunner`, plugin `#!/bin/sh\nsleep 6 &\nprintf frame\nexit 0`,
request deadline **400 ms**, 12 consecutive calls:

| Run | Elapsed | Result |
| ---: | ---: | --- |
| 0 | 400 ms | `provider_timeout` |
| 1–11 | **6.01 – 6.08 s** | **`err = nil`, the frame accepted** |

**11 of 12 calls exceeded the deadline by ~15×, and elapsed tracks the
grandchild's lifetime, not the deadline** (`sleep 3` → 3.38 s; `sleep 25` →
25.4 s). A grandchild that outlives the host hangs `ax` indefinitely.

This is a **regression introduced by this revision**. The same probe against
rev3's `runner.go` (swapped in, sha256-verified restore) returns in **501 ms**
with `provider_timeout` — the frame was misclassified, but the deadline held.
`runner.go` is new in this Story (absent at base `57afcc6d`), so the defect ships
with the leaf.

`Host.Call`'s own doc says *"Deadline accounting is exact"*; `runner.go:124-127`
names this exact shape (*"a detached grandchild may hold the pipes open past
WaitDelay"*) but handles it **only** on the `ctx.Err() != nil` branch. Confirmed
independently by mutant `RN3`: dropping the parent's `stdoutWriter.Close()`
makes the package suite **hang past 540 s** instead of failing — same unbounded
drain, different route.

**Close by** bounding the post-`Wait` drain: select `stdoutDone`/`stderrDone`
against `ctx.Done()` and a `waitDrainDelay` timer, closing the read ends on
expiry, so a judged result is still returned and the deadline still governs. It
needs a negative test that fails when `Run` outlives the deadline — a plugin
that backgrounds anything is the shape, and the package already writes such
scripts.

---

## F2 (BLOCKING) — the surrogate pin is a hand-written list, and it misses three classes

The gate itself is good: over a **derived corpus of 201,548 vectors** (all 65,536
code units as a single `\uXXXX` in a value, lowercase and uppercase hex, plus
135,949 pair combinations) the local gate and `canonicaljson.Canonicalize` reach
**identical verdicts, 0 divergences**. Both directions, valid pairs included
(`😀` accepted by both). The escape half of the pin is real.

The **13-vector hand-written corpus** in `surrogate_test.go` does not carry that.
Two one-character narrowings of the gate survive the **whole package suite**,
6 of 6 runs each, and both are real acceptances, not no-ops:

| Mutant | Narrowing | Suite | Vectors the gate now admits that `canonicaljson` refuses |
| --- | --- | ---: | ---: |
| `SG1` | `unit <= 0xdfff` → `unit <= 0xdc00` | **SURVIVED 6/6** | **1023** lone low surrogates (`\udc01`–`\udfff`) |
| `SG5` | `b <= 'F'` → `b <= 'E'` in `readHexUnit` | **SURVIVED 6/6** | **473** uppercase-hex escapes (`\uDBFF` and kin) |

The corpus holds exactly one lone-low vector (`\udc00`, the one value `SG1`
keeps) and no uppercase-hex-digit vector at all. `TestSurrogateGateAgreesWithCanonicalJSON`
and `TestProductionEntriesRefuseLoneSurrogateEscapes` both stay green under both.
Narrowing mutants that *do* land in a covered vector die (`SG2`, `SG3`), and
removing the call site dies (`SG4`) — so the gate is reachable and partly
pinned, but the class is not closed.

**Third class, live on the shipped tree.** A **raw** lone surrogate (WTF-8
`\xed\xa0\x80`, not an escape) is where the two implementations actually
disagree today:

```
{"a":"\xed\xa0\x80"}   canonicaljson.Canonicalize -> refused: input is not valid UTF-8
                       decodeStrictObject         -> ACCEPTED
                       decoded member value        -> "���" (ef bf bd ×3)
```

That is exactly the silent U+FFFD substitution `surrogate.go:1-14` exists to
prevent, reached through the other encoding. Two statements in the tree are
false because of it:

- `surrogate.go:26-28` — *"non-UTF-8 bytes are not its verdict: they fall
  through to the JSON-syntax arms that already own them."* **No arm owns them.**
  `decodeStrictObject` has no `utf8.Valid` check; `encoding/json` transliterates
  and returns success.
- `identity.go:24-28` — *"the decoder half of that agreement is pinned by
  `TestSurrogateGateAgreesWithCanonicalJSON`."* It is pinned for escapes only.

I drove the G-C#1 advisory to a measurement: `provhost.CheckIdentity` vs
`canonicaljson.CalculateObjectIdentity` over a 32-vector §5.5 shape corpus
(kinds, bounds 128/256/512/1024, 32-vs-33 opaque keys, key grammar, absolute and
Windows paths, realm-fingerprint rule, subject≠session, schema/version, unknown
and duplicate members, non-UUIDv7, bad timestamp, non-digest `record_id`).
**31 of 32 agree.** The single divergence is this same raw-byte class:
`CheckIdentity` accepts an identity record whose `native_session_id` is silently
U+FFFD-mangled. So the recorded bound *"the two validators must agree rule for
rule"* is true for the shape rules and false at the decoder.

Not live on the wire today: `DecodeResponse` runs `utf8.Valid(frame)` before any
body decode (`protocol.go:402`), and every body decoder is downstream of it —
I checked all seven. The hole is on the exported entry points, which is where a
`cmd/ax` lift or a disk-read identity record will meet it.

**Close by** (a) deriving the pin corpus instead of listing it — the sweep above
kills `SG1` and `SG5` instantly and costs 0.3 s — and (b) one `utf8.Valid(data)`
in `decodeStrictObject`, which closes the raw class and makes both quoted
sentences true. Or keep the copy asymmetric and restate both bounds honestly.

---

## F3 (BLOCKING) — the v3 bound is false: only a v2-*shaped* 3.x envelope is `incompatible_protocol`

`doc.go` answers G-C#2 with: *"this host speaks major 2 only, classifies **any
recognizable 3.x envelope** as `incompatible_protocol` without trusting its
payload … A v3 plugin is therefore refused loudly rather than misread as v2."*

`DecodeResponse` orders the checks decode → `ok` → `checkResponseMembers` →
`protocol` → `protocol_version`. The v2 member vocabulary is applied **before**
the major is known:

| 3.0.0 envelope | Classified as |
| --- | --- |
| exactly v2-shaped | `incompatible_protocol` ✓ |
| plus any v3 member (`trace_id`) | `provider_protocol_error: unknown member` ✗ |
| without `body` (a plausible v3 restructure) | `provider_protocol_error: missing member` ✗ |
| with a non-UUIDv7 `request_id` | `incompatible_protocol` ✓ |

Same for `3.1.0`, `3.0.0-beta`, `3`, `3.0`, `999.0.0`. A real v3 plugin — one
whose envelope differs, which is what a major bump is for — is diagnosed as a
**broken v2 plugin**, which is the precise failure mode the bound claims not to
have, and the two codes carry different exit statuses and different operator
guidance. The tree's own pinned §7.2 quote scopes the rule it applies:
*"unknown members are protocol errors **under major version 2**"*
(`closed_vocabulary_census_test.go:442`).

**Close by** hoisting the `protocol` and `protocol_version` checks above
`checkResponseMembers`, with a negative test asserting `incompatible_protocol`
for a 3.x envelope carrying a foreign member; or restate the bound as *"any 3.x
envelope that is otherwise exactly v2-shaped"* and say what the others get.

---

## G-A — what the round asked, answered

| Question | Answer |
| --- | --- |
| Does the pin check **both** directions? | Yes, and a third way: each vector asserts a literal verdict *and* both implementations against it. Drift in either direction reddens. Not the weakness. |
| Derived or hand-written? | **Hand-written, 13 vectors**, and demonstrably too thin — `SG1` (1023 admissions) and `SG5` (473) survive it. F2. |
| Valid **pairs** covered? | Yes. `😀` and raw `😀` accepted by both; 135,949 derived pair vectors, 0 divergences. Not a one-sided gate. |

The gate's own walk is correct: escaped backslash handled (`"\\ud800"` is text
in both), unterminated strings and malformed escapes deferred to the syntax
arms, member names / nested objects / array elements all covered (63 position
vectors, 0 divergences), `\ud800𐀀` and `􏿿` both judged
correctly.

---

## G-B — the traceability re-pin is clean

| Check | Result |
| --- | --- |
| Every registry row names code that exists | **0 broken references** across 78 acceptance cases and 58 ownership rows, each production and test `path`+`declaration` resolved independently of the gate |
| Now-false gap texts rewritten | §5.5, §7.2, §7.3, §7.5 rewritten; §7.3 no longer says *"no provider manifest is implemented anywhere in this repository"* |
| Any **other** now-false claim left standing | None. Swept all 58 gaps for "not implemented"/"nowhere"/"anywhere in this repository": the 15 hits are §1.6, §2.3, §2.4, §3.3, §5.1–5.6, §6.1, §6.5, §10.3, §10.4, §15.3, §18.1 — all outside this Story's surface and all still true (checked §6.1's *"provider host"* clause and §15.3's *"no RPC hello frame … anywhere"* explicitly) |
| Re-pin computed, not copied | Yes — a copied hash fails closed, and the suite is green at `680a78f3…` |
| Is the pin semantic, as claimed? | Yes, proven both ways: a one-word gap-text edit reddens with an exact digest mismatch; a full `json.dumps(indent=1, sort_keys=True)` reformat stays **green** |
| Report figures | `bindings 49→53`, `acceptance_cases 74→78`, `unevidenced 41→45`, `clauses 403→428`, `discharged 17` unchanged — consistent with 4 new unevidenced bindings over 4 newly-owned sections; README and both goldens updated to match |

Advisory: the §7.7 gap defers version probing and launch-plan equality to
*"later leaves"*. This is the Story's last leaf, and `STORY-260905-3t31e9`'s
acceptance criteria (A1, A3, A6, A7) do not cover it. Name where it actually
goes.

---

## G-C — the three advisories

| Advisory | Answered? | Verdict |
| --- | --- | --- |
| A2 `CheckIdentity` duplicates `validateProviderIdentityRecord` | Recorded bound in `identity.go` | **Bound is right, its pin claim is wrong.** 31/32 differential agreement measured; the 1 divergence is F2's raw-byte class, and the sentence claiming the decoder half is pinned is false |
| A3 Provider Protocol 3.0.0 | Recorded bound in `doc.go` | **False as written.** F3 |
| A4 provider error model vs `refuseCausalLeak` | Recorded bound in `provider/doc.go` | **Accurate.** `refuseCausalLeak` exists at `axerror/details.go:204`, is called from `axerror.go:165`, and refuses verbatim containment of the cause and each wrapped link; *"no such call site exists yet"* verified — no `cmd/`, and no non-test importer of `internal/provider`. Only nit: it decides **verbatim** containment, not paraphrase — worth one clause so the future lift does not over-trust it |

---

## G-D — the full sweep, no resurrections

All nine round-4 batches plus the round-5 packs, whole-package runs,
`cp`-aside restore with sha256 verification after every row.

| Battery | Rows | Killed | Survived | Not applied |
| --- | ---: | ---: | ---: | ---: |
| b1 | 15 | 15 | 0 | 0 |
| b1b | 20 | 19 | 1 (`Q4b`) | 0 |
| b2 | 21 | 19 | 0 | 2 (`PR2`, `PR4`) |
| b2r | 11 | 11 | 0 | 0 |
| b3 | 20 | 20 | 0 | 0 |
| b4 | 25 | 24 | 0 | 1 (`K6`, stale path) |
| b5 | 7 | 6 | 0 | 1 (`C1-floorTo1`) |
| b6 | 2 | 2 | 0 | 0 |
| b7 | 1 | 0 | 1 (`C9b`) | 0 |
| `k6fix` | 1 | 1 | 0 | 0 |
| **total** | **123** | **117** | **2** | **4** |

**The two survivors are `Q4b` and `C9b` — the two stated bounds, name for name.
Zero resurrections, zero new survivors.** `probe_widen` replayed 12 of 14 with
`R1`/`R3` surviving as the stated behaviour-preserving bounds;
`probe_failclosed` 4 of 4.

The four `NOT_APPLIED` are anchors this revision rewrote. I re-anchored all
four rather than let the denominator drop:

| Re-anchored | Result |
| --- | ---: |
| `PR2r` standard returns the yolo mapping | KILLED (`TestProfileMappingMatchesSection77`) |
| `PR4r` unknown profile not refused | KILLED (`TestClosedMemberSetsRefuseExtraMembers`) |
| `K6` (already corrected in `k6fix`) | KILLED |
| `C1floor-r` floor `164 → 1` | **SURVIVED** — and correctly: the floor is a `<` lower bound, so lowering it is an equivalent mutant. Round-6's "kill" of this row was the F2 flake, exactly as that verdict warned. The scanner-shortness class is carried by `b5`'s `C2-skipIdentityFile`, which kills |

### Round-6 F1 is closed

The three widenings that survived 12/12 last round now die, each to its own
spec derivation:

| Mutant | Vocabulary | Result |
| --- | --- | ---: |
| `C1r` | `responseMembers` + `"bogus"` | **KILLED** (`TestResponseMembersAreDerivedFromSpec`) |
| `C2r` | `profileNames` + `"bogus"` | **KILLED** (`TestProfileNamesAreDerivedFromSpec`) |
| `C8r` | `statusStates` + `"bogus"` | **KILLED** (`TestStatusStatesAreDerivedFromSpec`) |

The census itself is still load-bearing: `C4`–`C7` (each disjunct, and the
`map[string]string` shape) killed through the orphan loop; `C9`/`C10` killed,
so build-tagged files are still scanned.

### The residue, measured independently of the census

Denominator taken by scanning the 15 production files directly:

- **1** `switch` in production (`DecodeStatusOutcome`), classified, and
  `SW1` (plant an unclassified switch) **KILLED**.
- **0** type switches, **0** `select`, **0** chained-equality vocabularies,
  **0** `slices.Contains`, **0** `sort.Search`, **0** `init()`-built sets.
- **5** function-local literals: four are allow-sets built from a package-level
  vocabulary (`quiesceBlockers`, `manifestPlatforms`, `identifyEvidence`) or
  from runtime input, so the census covers them transitively; the fifth is
  `protocol.go:323`'s order list, excluded by the stated `*Required` rule.
- Census: **22 found / 22 registered**, both directions.

**Ratio: 22 of 22 for the scanned shapes, and 0 unguarded instances outside
them.** Two stated bounds are missing from the tree:
`TestAllProductionSwitchesAreClassified` matches only `*ast.SwitchStmt`
(`SW2`, a planted type switch, **SURVIVED 6/6**) and keys on the enclosing
function name, so a second switch inside `DecodeStatusOutcome` is admitted
silently. Zero instances today; say so rather than leaving the header claim to
cover it.

### On the two new F2 tests

They are real but **probabilistic**. Against rev3's `runner.go`,
`TestExecRunnerKeepsValidFrameWhenPluginIgnoresStdin` and
`TestExecRunnerClassifiesCrashExitDeterministically` are red in **2 of 3** runs
(one run fully green; 4 of 200 frames discarded in the red run). And `RN2` —
restoring `return Result{}, writeErr` ahead of the judge, i.e. round-6 F2's
exact ordering — **SURVIVED 6 of 6**, including against a plugin that closes
its own stdin (`exec 0<&-`) to force the EPIPE: 0 discards in 1250 calls. What
actually carries F2 is the structural half (`os.Pipe`), which `RN3` shows is
load-bearing — by hanging. The `writeErr` ordering is unpinned; a fake-Runner
unit test over `Run`'s judging rule would pin it deterministically and cost
nothing.

### Leaf 1 and leaf 2

`internal/provider` subtree `308a2f9f…` → `b1f62223…`; the entire delta is an
**11-line package doc comment** in `doc.go`. No production byte changed, so
leaf 1's battery cannot resurrect and was not replayed — stated, not assumed.
Leaf 2 is `internal/provhost`, whose production **did** change this round; the
123-row replay above is that measurement.

---

## Gates

| Gate | Result |
| --- | --- |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `GOOS=windows go vet ./...` | exit 0 |
| `gofmt -l internal/` | clean |
| `go test ./... -count=1` | **15 packages green** |
| `go test ./internal/provhost/ -count=1` | **green 10 of 10** — round-6's flake is gone |
| `go test ./internal/provhost/ -race -count=1` | green |
| coverage | provhost 86.0%, provider 97.0%, traceability 86.6% |
| candidate tree after review | `a72c5666…` unchanged |

---

## What closes this round

1. **F1** — bound the post-`Wait` drain so a detached grandchild cannot hold
   `Host.Call` past its deadline, with a negative test that fails when `Run`
   outlives the deadline.
2. **F2** — derive the surrogate pin corpus (`SG1` and `SG5` must go red), and
   either add `utf8.Valid` to `decodeStrictObject` or correct the two false
   sentences in `surrogate.go` and `identity.go`.
3. **F3** — check `protocol`/`protocol_version` before the v2 member
   vocabulary, or restate `doc.go`'s v3 bound to match what ships.

Also worth a line each, not blocking: the `*ast.SwitchStmt`-only and
function-name-keyed bounds of the switch classifier; the arm-census floor being
a lower bound; the §7.7 gap's "later leaves"; and `refuseCausalLeak` deciding
verbatim containment only.

Everything else in this leaf is in good shape. The census closure is genuine,
the traceability re-pin is honest and correctly computed, the discovery and
protocol leaves are untouched, and nothing regressed anywhere in the battery.

Replay pack attached as `TASK-260830-32jeti_review-round7-probe-pack.tgz`.
