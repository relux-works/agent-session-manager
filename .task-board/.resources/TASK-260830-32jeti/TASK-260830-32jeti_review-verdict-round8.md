# TASK-260830-32jeti — round-8 review verdict: CHANGES REQUESTED

- Change Request: `CR-TASK-260830-32jeti-6` revision 6
- Base `57afcc6d…` → candidate tree `f926d104…`, `repository_delta=present`, 60 paths
- Worktree content verified byte-identical to the candidate tree before the
  first mutant and after the last one (`git write-tree` over a temporary index:
  `f926d104ab88f04d6b8a5c96931eb7518e1245a8`, three times).

One blocking finding. It is the same class as round-7 F2, so the note under
`repeat-of` matters more than the two vectors it names.

---

## F1 (blocking) — the derived surrogate sweep does not enumerate its pair
## dimension, and both boundary narrowings of the low-surrogate range survive

`internal/provhost/surrogate_test.go:70` claims:

> Here the inputs are enumerated from the space and the verdicts come from the
> independent implementation, so a narrowed bound diverges from the oracle on
> every vector it newly admits

and lists, under "Dimensions, each fully enumerated", "every high surrogate
paired with boundary lows and non-low seconds".

The second unit of a pair is **not** enumerated. It is a hand-written
five-element list:

```go
for _, second := range []int{0xdc00, 0xdc01, 0xdfff, 0x0041, 0xd800} {
```

The two values adjacent to the low-surrogate range — `0xdbff`, immediately
below `0xdc00`, and `0xe000`, immediately above `0xdfff` — are both absent.
They are the only seconds that separate the correct bound from an off-by-one
narrowing of it, so both off-by-one narrowings of that exact bound survive the
**whole** `internal/provhost` package, not merely the sweep:

| mutant | `surrogate.go:71` becomes | result |
| --- | --- | ---: |
| `SGC-lowCeilingToE000` | `low > 0xe000` | **SURVIVED** (`go test ./internal/provhost/ -count=1`, exit 0) |
| `SGJ-lowFloorToDBFF` | `low < 0xdbff` | **SURVIVED** (same, exit 0) |

Both are real production divergences, not equivalent mutants. Measured
directly through `decodeStrictObject` against the `canonicaljson` oracle:

```
clean    {"a":"\ud800\ue000"}  canonical.refuse=true  gate.refuse=true
SGC      {"a":"\ud800\ue000"}  canonical.refuse=true  gate.refuse=false
```

Under `SGC` the host accepts a body whose lone high surrogate `encoding/json`
then replaces with U+FFFD — the exact Section 1.6 class this gate exists to
close — and 239,630 swept vectors say nothing about it, because none of them
is a high surrogate followed by `\ue000`.

The five other narrowings I planted on this gate are killed
(`unit >= 0xd801`, `unit <= 0xdffe`, `'A'..'E'`, `'b'..'f'`, and the UTF-8 gate
disabled), and round-7's `SG1`/`SG5` are killed now, so the escape-unit and
raw-UTF-8 dimensions are genuinely derived. Only the pair dimension is not.

**Fix direction.** Do not add `0xdbff` and `0xe000` to the list. That closes
the two vectors I happened to name and leaves the class exactly where it is —
see `repeat-of`. Derive the second unit the way the other dimensions are
derived: either enumerate `second` over the full BMP for a sampled set of
highs, or compute the boundary set from the range endpoints (`lo-1, lo, lo+1,
hi-1, hi, hi+1`) so the corpus follows the bound instead of a literal list.
Then re-run `SGC`/`SGJ` from `reanchor.json`; both must redden.

**repeat-of:** revision 5 finding F2 ("hand-written surrogate pin misses three
classes"). This is the second consecutive finding of the same class — a
surrogate-corpus dimension that is written by hand while the comment above it
says the space is enumerated. Two consecutive same-class findings mean the next
step is a gate, not another revision: the fix should make the sweep's own
comment checkable, e.g. by deriving every dimension from the range constants
`hasLoneSurrogateEscape` uses, so a hand-written dimension cannot reappear
without the derivation obviously disagreeing.

---

## Advisories (not blocking, recorded because they are measured)

**A1 — the `writeErr`-ordering half of round-6 F2 is still unpinned.**
`RN1-dropStdoutJudge` and `RN2-writeErrFirst` both SURVIVED again this round,
as in round 7. `RN2` restores round-6 F2's exact defective ordering and nothing
reddens. Round 7 already recommended a fake-`Runner` unit test over `Run`'s
judging rule; it was not added. What actually carries F2 is the structural
`os.Pipe` half, which `RN3` still shows is load-bearing by hanging the suite.
Carried, not new.

**A2 — the wait-select `ctx.Done()` branch and its `Process.Kill` backstop are
unreachable on unix and unexercised.** `runner.go:152-160` says "Cancel targets
the process group, which a self-detached child escapes". A *direct* child
cannot self-detach here: `SysProcAttr{Setpgid: true}` makes it a process-group
leader, so its own `setsid()` fails with `EPERM` and the group kill still
reaches it. Measured — a plugin that `exec`s `perl … setsid(); sleep(20)`
returns `provider_timeout` at 2.00s clean, at 2.00s with the backstop removed,
and at 2.00s with the backstop removed *and* `waitDrainDelay` raised to 600s:

| mutant | result |
| --- | ---: |
| `RN4` (no `Process.Kill` backstop) | SURVIVED |
| `RN6` (`waitDrainDelay` 2s → 600s) | SURVIVED |
| `RN7` (`RN4` + `RN6` together) | SURVIVED |
| `RN9` (whole `case <-ctx.Done():` arm made unreachable) | SURVIVED |

The branch is harmless defensive redundancy; the comment justifying it names a
scenario the `Setpgid` call structurally prevents. Either state it as a
Windows/non-`Setpgid` backstop or drop the unreachable justification.

The load-bearing half of the round-8 F1 fix **is** pinned:
`RN8-collectDrainsUnbounded` (pass `context.Background()` to `collectDrains`)
is KILLED by `TestExecRunnerDetachedDescendantCannotHoldCallPastDeadline` at
20.23s. That fix is real.

**A3 — carried one-directional bounds, re-measured, unchanged.** `Q4b`,
`C9b`, `probe_widen` R1/R3, `SW2` (type switches are invisible to the switch
census), and `C1floor-r2` (the arm-census floor 166 catches derivation
shortfall only, not a lowered floor). `C3a` — making the switch census skip
`spawn.go` — survives but is an equivalent mutant: `spawn.go` holds no switch,
and skipping the one file that does (`status.go`) is caught by the stale-
classification check.

---

## G-A — verdict on the pin: real in both directions, incomplete in one dimension

The sweep judges every vector by `canonicaljson.Canonicalize`, never by a shared
expectation, and it compares refusal in **both** directions (`(canonicalErr !=
nil) != (gateFault != nil)`), so an over-broad gate reddens as loudly as a
narrowed one. Valid surrogate pairs are covered — 1,024 highs × `0xdc00`,
`0xdc01`, `0xdfff` — and so are astral raw samples including U+1F600, so the
"rejects `\ud800` and also rejects 😀" failure cannot hide. The escape-unit,
hex-case, member-name, nested, array, raw-UTF-8 and malformed-byte dimensions
are genuinely enumerated. The pair second-unit dimension is not; that is F1.

Local copy vs shared package: the copy is deliberate and the reason given
(member-level fault attribution the shared decoder does not report) is accurate.
`internal/canonicaljson` is untouched — subtree unchanged from the base.

## G-B — the registry re-pin is honest and the gate fails closed

- All 14 production and test declarations named by the four new acceptance
  cases and four new section bindings exist at the paths given.
- The four false gap texts are **rewritten**, not left beside contradicting
  rows. §7.3 no longer says "no provider manifest is implemented anywhere in
  this repository"; §5.5, §7.2 and §7.5 now name the provhost implementation
  and keep their honest "no clause is enumerated against an acceptance case"
  qualifier and their unchanged `unevidenced`/`unmeasured` coverage labels.
- The re-pin is **computed, not copied**: the digest is `sha256` over
  `json.Marshal(registry)` of the parsed projection, so it cannot match a file
  it was not derived from. Attacked three ways, all fail closed:

| attack | result |
| --- | --- |
| forge an acceptance case pointing at `internal/provhost/nonexistent.go` | `read …: no such file or directory` |
| repoint an existing row's declaration to another real symbol in another file | `declaration "DecodeManifest" is absent from "internal/provhost/probe.go"` |
| prepend `TAMPERED PROSE.` to one `gap` string, nothing else | `projection digest e0cb1eac… differs from reviewed 680a78f3…` |

Gap prose is inside the pinned projection, so the documentation cannot drift
without a review of the pin. Registry→code is verified; code→registry is still
unscanned, which is the gate's own pre-existing bound and not this leaf's job.

## G-C — all three advisories are answered in the tree

| advisory | answer |
| --- | --- |
| `CheckIdentity` duplicating `canonicaljson.validateProviderIdentityRecord` | Deliberate, with the reason stated at `identity.go:20-30`: this package must mint Section 15.1 provider-stdio refusals with member attribution and the shared package exports no such entry. The 16-member required set matches `core_records.go:208-211` exactly. The decoder half of the agreement is pinned by the derived sweep; the shape half is stated as a same-change obligation, i.e. a stated bound, not a claimed gate. |
| Provider Protocol 3.0.0 pinned by the v0.5.0 catalog while provhost hard-pins major 2 | Stated bound in `doc.go`, and round 8 made it *true* rather than documented: `F3b-peekOnlyMajorAbove3` is KILLED by `TestDecodeResponseForeignMajorPrecedesMemberRules` (6 mismatch rows), so a 3.0.0 envelope carrying v3 members, or omitting `body`/`error`/`ok`, is now `incompatible_protocol` instead of "unknown member". Frames without our protocol identity or without a readable version keep the member-rule verdicts. |
| provider's error model vs `axerror.refuseCausalLeak` | Recorded with its composition point named: `internal/provider/doc.go` states that `Error()` appends the cause verbatim, that `refuseCausalLeak` refuses any message reproducing a local cause, and that the future `cmd/ax` lift must rebuild the message with dynamic facts in `Details` — and states plainly that no such call site exists yet. |

## G-D — full sweep, zero resurrections

164 battery rows replayed plus 13 new/re-anchored rows.

| battery | rows | killed | survived | not applied | vs round 7 |
| --- | ---: | ---: | ---: | ---: | --- |
| b1 | 15 | 15 | 0 | 0 | identical |
| b1b | 20 | 19 | 1 `Q4b` | 0 | identical |
| b2 | 21 | 19 | 0 | 2 (`PR2`/`PR4`, re-anchored in supp → KILLED) | identical |
| b2r | 11 | 11 | 0 | 0 | identical |
| b3 | 20 | 20 | 0 | 0 | identical |
| b4 | 25 | 24 | 0 | 1 (`K6` → k6fix KILLED) | identical |
| b5 | 7 | 6 | 0 | 1 (`C1` floor → re-anchored) | identical |
| b6 | 2 | 2 | 0 | 0 | identical |
| b7 | 1 | 0 | 1 `C9b` | 0 | identical |
| k6fix | 1 | 1 | 0 | 0 | identical |
| probe_widen | 14 | 12 | 2 (R1/R3 bounds) | 0 | identical |
| probe_failclosed | 4 | 4 | 0 | 0 | identical |
| census3 | 11 | 9 | 0 | 2 | `C1floor-r` anchor rewritten (166), re-anchored below |
| supp | 12 | 8 | 3 | 0 (+`RN3` kills by hang) | **`SG1` and `SG5` now KILLED — round-7 F2 closed** |
| reanchor (new) | 10 | 3 | 7 | 0 | new |
| RN7 / RN8 / RN9 (new) | 3 | 1 | 2 | 0 | new |

No survivor from a previous round resurrected. Every round-7 `NOT_APPLIED`
anchor was re-anchored rather than counted as a pass: `PR2`/`PR4` in
`supp.json` (both KILLED), `K6` in `k6fix.json` (KILLED), `C1floor` at its new
value 166 (SURVIVED, the same one-directional bound), and `C3` split into
`C3a`/`C3b` because the census walker is now duplicated — `C3b` KILLED, `C3a`
equivalent as explained in A3.

Map-iteration rule: the map-shaped batteries (`b1`, `b1b`, `b3`, census) were
run once each with identical results to round 7, and `go test
./internal/provhost/ -count=8` is green, so no verdict here rests on a single
lucky iteration order.

**Leaf 1 was not replayed, and that is measured, not assumed.** The
`internal/provider` subtree is byte-identical between the round-7 candidate
`a72c5666…` and this one: both `b1f62223bd5e0d496f5a44a1eeffcbb2ffd5a28f`. No
production byte changed, so leaf 1's battery cannot resurrect. Leaf 2 is
`internal/provhost`, whose production did change; the replay above is that
measurement.

## AC coverage

Every AC surface has a named driving production entry and named negative tests:
manifest (`DecodeManifest` / `TestDecodeManifestRefusals`), probe
(`DecodeProbe` / `TestDecodeProbeRefusals`,
`TestDecodeProbeEnabledRequiresAvailable`), capability gate
(`RequireCapability` / `TestRequireCapabilityRefusesUnprovenSurfaces`,
`TestCapabilityGatePrecedesCall`), profile mapping (`ProfileMapping` /
`TestProfileMappingRefusesUnknowns`), quiescence (`DecodeQuiesceProof` /
`TestDecodeQuiesceSafeLies`, `TestDecodeQuiesceRefusals`), identity
(`CheckIdentity` / `TestCheckIdentityRefusals`,
`TestCheckIdentityNamesAnotherProvider`), resume (`DecodeSpawnPlan` /
`TestResumeThroughCall`, `TestSpecResumePlanDecodes`,
`TestDecodeSpawnPlanRefusals`), idempotency (`IdempotencyKeyFor` /
`TestIdempotencyKeyForRefusesUnkeyed`, plus the cross-process
`TestLostPrepareReturnsByteIdentical` and `TestChangedBodyReturnsMismatch`),
and fail-closed status recovery (`DecodeStatusOutcome` /
`TestDecodeStatusOutcomeUnknownFailsClosed`, nullability matrix, foreign
transaction, entropy floor). 9 of 9 AC rows have driving tests. No AC row is
unchecked, and no mutant table in this round is deletion-only.

## Gates

| gate | result |
| --- | ---: |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `GOOS=windows go vet ./...` | exit 0 |
| `gofmt -l internal/` | clean |
| `go test ./... -count=1` | 15 packages ok |
| `go test ./internal/provhost/ -count=8` | exit 0 |
| `go test ./internal/provhost/ -race -count=1` | exit 0 |
| coverage | provhost 85.8%, provider 97.0%, traceability 86.6% |
| candidate tree after review | `f926d104…` unchanged |

## What closes this round

Close F1 by deriving the sweep's pair second-unit dimension from the same range
constants the gate uses, then show `SGC-lowCeilingToE000` and
`SGJ-lowFloorToDBFF` from `reanchor.json` both KILLED. A1 and A2 are recorded,
not required. Everything else in this CR — the traceability re-pin, the
foreign-major ordering, the raw-UTF-8 gate, the collectDrains deadline bound,
and the three G-C answers — measured clean.
