# TASK-260830-32jeti — round-9 review verdict: ACCEPTED

- Change Request: `CR-TASK-260830-32jeti-7` revision 7, state `ready`
- Base `57afcc6d…` → candidate tree `3817cef4…`, `repository_delta=present`, 60 paths
- Worktree content verified byte-identical to the candidate tree by
  `git write-tree` over a temporary index (`GIT_INDEX_FILE` + `read-tree HEAD`
  + `add -A`): `3817cef48c266f4158370d84db95936fa58dd4ab`, before the first
  mutant, after every battery, and after the last mutant. Seven times.
- Reviewer run `RUN-260905-47806f`.

Round-8 F1 is closed. One new advisory (A3) is recorded, measured, not
blocking. Verdict: accept.

---

## G-A — the derived pair dimension: closed, and the ride-along risk is disproved

The round-8 blocking finding was that the sweep's pair second-unit dimension
was a five-element literal (`0xdc00, 0xdc01, 0xdfff, 0x0041, 0xd800`) missing
both values adjacent to the low-surrogate range, so `SGC` and `SGJ` survived
the whole package.

rev7 derives it: `surrogate.go` now names `highSurrogateMin/Max` and
`lowSurrogateMin/Max` and uses them in `hasLoneSurrogateEscape`;
`surrogate_test.go:129-141` builds `pairSeconds` from those constants
(`lowMin-2, lowMin-1, lowMin, lowMin+1, mid, lowMax-1, lowMax, lowMax+1,
lowMax+2, highMin, highMax, highMin-1`) crossed with every high surrogate.

The rev6→rev7 delta is exactly three hunks: the runner comment, the surrogate
constants, and the derived `pairSeconds` plus its comment. Nothing else moved.

### The load-bearing question: does the fixture ride along with a mutated constant?

It does ride along — and that is why it kills, not why it hides. The swept
points track the gate; the **verdict** comes from `canonicaljson.Canonicalize`,
which does not. So moving a bound moves the probe onto the value the bound
newly admits or newly refuses, and the oracle disagrees there. Measured on all
four constants in both directions:

| mutant | `surrogate.go` const becomes | result |
| --- | --- | ---: |
| `M1-highMax-const-narrow` | `highSurrogateMax = 0xdbfe` | **KILLED** |
| `M2-highMin-const-narrow` | `highSurrogateMin = 0xd801` | **KILLED** |
| `M3-highMax-const-widen` | `highSurrogateMax = 0xdc00` | **KILLED** |
| `M4-highMin-const-widen` | `highSurrogateMin = 0xd7ff` | **KILLED** |
| `M5-lowMax-const-widen` | `lowSurrogateMax = 0xe000` | **KILLED** |
| `M6-lowMax-const-narrow` | `lowSurrogateMax = 0xdffe` | **KILLED** |
| `M7-lowMin-const-narrow` | `lowSurrogateMin = 0xdc01` | **KILLED** |
| `M8-lowMin-const-widen` | `lowSurrogateMin = 0xdbff` | **KILLED** |

8 of 8. The producer ran the `lowSurrogateMax` widening; the other seven are
mine. The high side answers G-A's first question empirically: an off-by-one on
`highSurrogateMax` or `highSurrogateMin` dies, in both directions, even though
the pair loop's own bounds move with them — the single-unit dimension (every
BMP code unit as an escape, both hex cases) covers the lone-high verdict that
the shifted bound changes.

### Narrowing the predicate itself

Round-8's two named survivors plus ten more shapes, all against the same test
pair (`TestSurrogateGateDerivedSweepAgreesWithCanonicalJSON`,
`TestProductionEntriesRefuseLoneSurrogateEscapes`):

| mutant | shape | result |
| --- | --- | ---: |
| `SGC-lowCeilingToE000` | `low > lowSurrogateMax` → `low > 0xe000` | **KILLED** |
| `SGJ-lowFloorToDBFF` | `low < lowSurrogateMin` → `low < 0xdbff` | **KILLED** |
| `SGK` | `low > lowSurrogateMax` → `low >= lowSurrogateMax` | **KILLED** |
| `SGL` | `low < lowSurrogateMin` → `low <= lowSurrogateMin` | **KILLED** |
| `SGM` | drop the `low > lowSurrogateMax` clause | **KILLED** |
| `SGN` | drop the `low < lowSurrogateMin` clause | **KILLED** |
| `SG3r` | `!ok \|\| …` → `ok && (…)` (accept lone high at end) | **KILLED** |
| `SGP` / `SGP2` | lone-low arm narrowed at each end | **KILLED** |
| `SGQ` / `SGQ2` | high arm narrowed at each end | **KILLED** |
| `SGR` / `SGS` | hex-digit range narrowed (`'B'..'F'`, `'a'..'e'`) | **KILLED** |
| `SGT` | UTF-8 gate in `decodeStrictObject` disabled | **KILLED** |
| `SGW` | lone-surrogate gate not called from `decodeStrictObject` | **KILLED** |

21 of 22 killed. Not one is delete-only: `SGW`/`SGT` are the two deletions and
they are the *pair* to sixteen narrowings, not a substitute for them.

**The one survivor, stated as a bound, not passed over.** `SGU-hex-len-narrow`
(`start+6 > len(input)` → `start+5 > len(input)` in `readHexUnit`) survives.
It is not a gate weakening: it changes nothing about which class the gate
admits, and only turns a truncated `\uXX` escape sitting in the final bytes of
the buffer from a clean `ok=false` into a slice-bounds panic. Every swept body
ends `"}` after the escape, so no vector reaches it. Bound: the sweep does not
enumerate escapes truncated at the exact end of input. The unmutated code
guards it correctly, so there is no production defect here.

### The oracle runs over the production reader

`compare` calls `decodeStrictObject(body)` — `protocol.go:268`, the real strict
object reader that all fourteen non-test call sites in `identity.go`,
`probe.go`, `manifest.go`, `quiesce.go`, `status.go`, `spawn.go` and
`protocol.go` read through, not an isolated predicate.
`TestProductionEntriesRefuseLoneSurrogateEscapes` additionally drives the eight
exported entries (`DecodeManifest`, `DecodeProbe`, `CheckIdentity`,
`DecodeIdentifyResult`, quiesce, `DecodeSpawnPlan`, `DecodeResponse`,
`DecodeStatusOutcome`). Sweep: 246,798 of 246,798 vectors agree.

**No dimension of the sweep is a hand-written boundary literal.** The remaining
literals are the malformed-byte patterns, three astral samples, the `+= 64`
sampling stride and the 200,000 floor — none of them a surrogate bound. The raw
WTF-8 branch tests `unit >= 0xd800 && unit <= 0xdfff` with literals rather than
the gate constants, which is correct: deriving *that* branch from the gate
would be the ride-along shape, since it decides how to build the body, not what
the verdict should be.

---

## G-B — full replay, 177 rows, zero resurrections

All sixteen batteries plus leaf 1's and leaf 2's, replayed against candidate
tree `3817cef4…` with the round-8 harness (`mutate.py`, which cp-restores and
sha256-verifies each file).

| battery | rows | killed | survived | not applied | vs round 8 |
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
| probe_widen | 14 | 12 | 2 (R1/R3) | 0 | identical |
| probe_failclosed | 4 | 4 | 0 | 0 | identical |
| census3 | 11 | 9 | 0 | 2 (re-anchored in reanchor) | identical |
| supp | 12 | 5 (+`RN3` by hang) | 3 | 3 (`SG1`/`SG2`/`SG3`) | anchors rewritten by rev7 |
| reanchor | 10 | 3 | 5 | 2 (`SGC`/`SGJ`) | `SGC`/`SGJ` moved SURVIVED → NOT_APPLIED |
| rn7a / rn89 | 3 | 1 (`RN8`) | 2 (`RN7`, `RN9`) | 0 | identical |

**Every `NOT_APPLIED` row was re-anchored, not counted as a pass.** rev7's
constant refactor rewrote five anchors. Re-anchored and re-measured above:
`SGC` → `low > lowSurrogateMax` → `low > 0xe000` **KILLED**; `SGJ` →
`low < lowSurrogateMin` → `low < 0xdbff` **KILLED**; `SG1` → `SGP`/`SGP2`
**KILLED**; `SG2` → `SGQ`/`SGQ2` **KILLED**; `SG3` → `SG3r` **KILLED**.
`PR2`/`PR4`/`K6`/`C1`/`C3` re-anchor exactly as in round 8.

No survivor from a previous round resurrected, and no previously-killed row
went green. The carried survivors are the same one-directional bounds round 8
recorded: `Q4b`, `C9b`, `R1`, `R3`, `SW2` (type switches invisible to the
switch census), `C1floor-r2`, `C3a` (equivalent), `RN1`, `RN2`, `RN4`, `RN5`,
`RN6`, `RN7`, `RN9`.

### The two carried advisories

**`RN1`/`RN2` — accurately stated, unchanged.** Both SURVIVED again. The
`writeErr` ordering half of the discarded-frame finding is still unpinned; the
structural `os.Pipe` half is still load-bearing (`RN3` kills by hanging the
suite to the 540s harness timeout). Carried, correctly described.

**`Process.Kill` backstop — the advisory is no longer accurately stated,
because rev7 fixed it.** Round 8 recorded that the justifying comment named a
scenario `Setpgid` structurally prevents. `runner.go:152-165` now says the
escaper is a grandchild-or-deeper that called `setsid` itself, that the direct
child is a group leader whose own `setsid` fails EPERM, and that the direct
kill exists to race exec's group `Cancel`. That matches `runner_unix.go`
(`SysProcAttr{Setpgid: true}` plus a `Cancel` that kills `-pid`) and matches
the measured behavior. `RN4`/`RN6`/`RN7`/`RN9` still survive, which is now a
disclosed bound rather than a false claim. Advisory closed.

---

## A3 (advisory, new, measured) — the derived refusal-arm inventory has an undisclosed aliased-constructor bypass

`internal/provhost/refusal_arm_inventory_test.go` derives constructor arms by
matching `call.Fun.(*ast.Ident)` against a fixed six-name set
(`integrity`, `failInvalid`, `failProtocol`, `failMismatch`, `failProcess`,
`failTimeout`). One line of aliasing escapes the derivation entirely.

Measured as a contrast pair — the *same* new production refusal arm, planted in
`CheckIdentity`, differing only in how the constructor is reached:

| mutant | how the arm calls the constructor | full `./internal/provhost/` suite |
| --- | --- | ---: |
| `AL2-directConstructorNewArm` | `failProtocol("identity body is preposterously large", "")` | **KILLED** — `TestDerivedRefusalArmsAreAllWitnessed` |
| `AL1-aliasedConstructorNewArm` | `var aliasedProtocol = failProtocol` then `aliasedProtocol(…)` | **SURVIVED**, exit 0 |

`AL1` is exactly the failure the file's own header names as its reason to
exist: a brand-new production refusal arm carrying a brand-new unasserted
detail shipping while `gofmt`, `go vet` and the whole suite stay green. It is
not in the file's stated bounds.

**Cross-Story.** The sibling Story landed as PR #32 and
`internal/terminalbackend/refusal_arm_inventory_test.go` — same family of gate,
built from the same model — closes this with `rejectConstructorAliases` (line
276) and states it at line 114: *"Constructor references outside direct-call
position (`refuse := mismatchf`) are rejected outright … an aliased refusal
cannot be inventoried."* The two packages diverge on a hole one of them
already knows how to close.

**Why this is advisory and not blocking.** The exposure is prospective, and I
measured that rather than assuming it: a scan of every non-direct-call
reference to the six constructors and to `integrity` across all provhost
production files returns only the six declaration sites themselves
(`protocol.go:121-137`, `status.go:84`). Nothing ships unwitnessed today. The
hole is in a test-completeness harness, not in a production gate; every
production gate named in the AC is attacked and killed above. This is the same
tier round 8 used for A1/A2 — one of which was a materially false claim in
production source.

**Fix direction, for whichever round next touches provhost refusals.** Either
port `rejectConstructorAliases` (adapting it to provhost's `var f = func(…)`
constructor declarations and to `inventory_test.go`'s deliberate test-time
reassignment), or state it in the header as a disclosed-not-closed bound
alongside the existing merged-identity bound. Both are honest; the current
unqualified claim is not. Note the same ident-matching shape covers
`frameFault` composite literals (matched by type-ident name) and `integrity`.

---

## AC coverage — 9 of 9 rows driven, 21 named tests all present

Verified by resolving each named test in the candidate tree, not by reading the
producer's table: all 21 exist at the claimed paths.

| AC surface | production call site | named driving tests |
| --- | --- | --- |
| manifests | `DecodeManifest` | `TestDecodeManifestRefusals` (manifest_test.go) |
| probes | `DecodeProbe` | `TestDecodeProbeRefusals`, `TestDecodeProbeEnabledRequiresAvailable` (probe_test.go) |
| capabilities | `RequireCapability` | `TestRequireCapabilityRefusesUnprovenSurfaces` (probe_test.go), `TestCapabilityGatePrecedesCall` (conformance_test.go) |
| profile mapping | `ProfileMapping` | `TestProfileMappingRefusesUnknowns` (profile_test.go) |
| quiescence | `DecodeQuiesceProof` | `TestDecodeQuiesceSafeLies`, `TestDecodeQuiesceRefusals` (quiesce_test.go) |
| identity | `CheckIdentity` | `TestCheckIdentityRefusals`, `TestCheckIdentityNamesAnotherProvider` (identity_test.go) |
| resume | `DecodeSpawnPlan` | `TestResumeThroughCall` (conformance_test.go), `TestSpecResumePlanDecodes`, `TestDecodeSpawnPlanRefusals` (spawn_test.go) |
| idempotency | `IdempotencyKeyFor` | `TestIdempotencyKeyForRefusesUnkeyed` (idempotency_test.go), `TestLostPrepareReturnsByteIdentical`, `TestChangedBodyReturnsMismatch` (conformance_test.go) |
| fail-closed tuples | `DecodeStatusOutcome`, `decodeStrictObject` | `TestDecodeStatusOutcomeUnknownFailsClosed` (status_test.go), `TestProductionEntriesRefuseLoneSurrogateEscapes`, `TestSurrogateGateDerivedSweepAgreesWithCanonicalJSON` (surrogate_test.go) |

No AC row is unchecked. No mutant table in this round is deletion-only.

---

## Gates, this round, run by me

| gate | result |
| --- | ---: |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `GOOS=windows go vet ./...` | exit 0 |
| `gofmt -l internal/` | clean |
| `go test ./... -count=1` | 15 packages ok, exit 0 |
| sweep vector count | 246,798 of 246,798 agree |
| candidate tree after review | `3817cef4…` unchanged |

## Verdict

Accept. Round-8 F1 is closed by derivation, not by adding the two vectors I
named — and the derivation survives the ride-along attack that would have made
it worthless. The full battery replays with zero resurrections and every
rewritten anchor re-anchored. A3 is recorded for the next round that touches
provhost refusals.
