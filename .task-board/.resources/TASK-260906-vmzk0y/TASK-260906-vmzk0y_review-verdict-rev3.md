# TASK-260906-vmzk0y — review verdict, CR revision 3

Verdict: **changes requested** (one blocking finding, three notes).

repeat-of: `rev2/G1` — class only: *a `parseMajor` classification migration the
whole provhost suite is indifferent to*. rev2/G1's own remediation is
measurably closed (G-A below); the class is not. **This is the second
consecutive round with a finding of this class**, so the orchestrator's
"two consecutive same-class findings mean a gate, not another revision" rule
applies to routing.

Candidate tree `a8570e2f74f3e0cf458a941e985a6d19354881cc` — verified equal to
the worktree by a detached-index `read-tree HEAD` + `add -A` + `write-tree`,
re-verified byte-identical after every mutant below. `ls-tree` shows the four
new files (`provider/lift.go`, `provider/lift_test.go`,
`terminalbackend/descriptor.go`, `terminalbackend/descriptor_test.go`) are
in-tree. 15 changed paths, matching the record.

rev2 → rev3 is exactly 3 files (`git diff be5b2844… a8570e2f…`:
`LOGBOOK.md`, `provhost/protocol.go`, `provhost/protocol_test.go`, 51+/6-).
Everything the rev2 verdict accepted is byte-identical and is **not**
re-litigated here.

## What I ran myself (nothing accepted from the outcome document)

| Gate | Result |
| --- | --- |
| `go build ./...` / `GOOS=windows go build ./...` | exit 0 / exit 0 |
| `go vet ./...` / `GOOS=windows go vet ./...` | exit 0 / exit 0 |
| `go test ./... -count=1` | exit 0, 22 packages ok, no FAIL |
| `go test -cover` provider / provhost / terminalbackend | 97.8% / 85.8% / 94.2% — reproduces the outcome exactly |
| `gofmt -l internal/` | clean |
| `go run ./internal/traceability/cmd/tracecheck` | ok — contracts=60, acceptance_cases=98 — reproduces |
| provhost refusal-arm bijection | 167/167 derived arms witnessed |
| candidate tree OID after all mutants | `a8570e2f…` unchanged |

## G-A — are the two classes pinned apart, or only the parse arm? **Closed**

1. **Named `DecodeResponse` rows exist for each malformed giant.**
   `protocol_test.go:645-668` `TestParseMajorSaturationStillValidatesTheRest`
   drives five rows (`<giant>.abc.def`, `<giant>.0.x`, `<giant>..0`,
   `18446744073709551618..0`, `18446744073709551618.0.x`) through
   `DecodeResponse` and asserts `provider_protocol_error` **and** exit 13, next
   to `TestParseMajorNeverWraps`'s exit-6 row for the all-numeric giant. Both
   use the same frame builder, so the classes differ only by input.

2. **A row that migrates between them reddens.** Planted the round-2 shape back
   (`major = math.MaxInt; continue` → `return math.MaxInt, true`): build exit 0,
   `--- FAIL: TestParseMajorSaturationStillValidatesTheRest`, and that is the
   only failure. The migration is pinned at the classification level, not only
   at the parse level.

3. **The `foreignMajor` peek.** The divergence round 2 found is *gone*, and I
   measured it rather than inferring it. Probing bare frames (`protocol`,
   `protocol_version`, `request_id`, `body` — no `ok`) against full frames:

   | version | bare frame | full frame |
   | --- | --- | --- |
   | `18446744073709551618.0.0` | `incompatible_protocol` 6 | `incompatible_protocol` 6 |
   | `99999999999999999999.abc.def` | `provider_protocol_error` 13, member `ok` | 13, member `protocol_version` |
   | `2.abc.def` | `provider_protocol_error` 13, member `ok` | 13, member `protocol_version` |
   | `a.0.0` | `provider_protocol_error` 13, member `ok` | 13, member `protocol_version` |

   The giant-malformed class now behaves identically to the ordinary malformed
   class in both frame shapes. The peek's *dangerous* direction is pinned:
   narrowing it to `major > ProtocolMajor` is KILLED by
   `TestDecodeResponseForeignMajorPrecedesMemberRules/major_one_with_v3_member`.
   (Its widening direction survives — note N4.)

## G-B — the evidence sentence. **Closed**

The round-2 "no gate was weakened" sentence is withdrawn *as classification
evidence* in `TASK-260906-vmzk0y_outcome-round3.md` ("Correction of the round-2
… sentence") and in LOGBOOK entry 2252. Both state the intended arm shift
**as measured** (all-numeric giants 13 → 6, witnessed by the exit-6 row) and
the unintended one, and confine the census claim to inventory bookkeeping
("saturation adds no `return 0, false` branch, so the inventory needs no new
row") — which I verified is true and is the only thing it now supports.

The `parseMajor` doc comment carries the same scoped form. I checked the other
prose touched by this delta (`provhost/doc.go`, `provider/doc.go`,
`terminalbackend/descriptor.go`, the §7.A registry gap) for a remaining
enumeration-standing-in-for-an-effect and found none.

## G-C — the notes

- **N1 closed.** The outcome heading reads "AC row coverage: 14 of 15 driven +
  1 stated bound", and row 14 is marked STATED BOUND in the table.
- **N2 partially closed** — see note N6.
- **N3 closed.** `mutG1_round3.py` and `mutsG1_round3.json` are under
  repo-root `.temp/TASK-260906-vmzk0y/`, not `/tmp`.

## G-D — battery and provenance

Denominator **re-derived for this revision** from production, not carried
forward: the nine version-classification obligations reachable from
`DecodeResponse` — `parseMajor`'s five rejection branches (P1 `len(parts) != 3`,
P2 major-digit range, P4 `len(parts[0]) == 0`, P5 `len(rest) == 0`, P6
rest-digit range) plus P3 the saturation guard, C1 the failure-error contract
major selection, K1 the `foreignMajor` peek predicate, and V1 the version-gate
mismatch predicate.

| Class | Applied | Killed | Survived |
| --- | ---: | ---: | ---: |
| narrowing | 12 | 10 | **2** |
| narrowing / token-preserving | 2 | 0 | **2** |
| arm-deletion | 2 | 2 | 0 |
| census-only | 1 | 1 | 0 |
| equivalence-probe | 1 | 0 | 1 (declared equivalent) |
| **total applied** | **18** | **13** | **5** |

Distinct rows, NOT counted as applied mutants:

| Control row | Disposition |
| --- | --- |
| X1 anchor `>=` absent from source (0 sites) | NOT_APPLIED |
| X2 saturation guard deleted outright (`math` unused) | COMPILE_FAIL |
| M13/M14 first anchoring: peek predicate text occurs twice | NOT_APPLIED — **re-anchored** as M13b/M14b on the indented occurrence and measured. Not passes. |

Per-mutant detail is in `.temp/TASK-260906-vmzk0y/review-rev3/battery-*.json`;
the harness restores each target byte-identically and asserts the SHA-256 after
every run.

**Provenance.** `git status` shows exactly the 15 recorded paths. Leaf 1's own
packages are untouched from `1d97474`: `git diff --name-only 1d97474 --
internal/canonicaljson internal/environ internal/terminalbackend/manifest.go
internal/terminalbackend/identity_ownership_test.go README.md` returns 0 files.
Of the files shared with leaf 1, `identity.go` (+28/-0) and
`refusal_arm_inventory_test.go` (+12/-0) are pure additions.

---

## F1 (blocking) — the parse gate's rejected class is pinned only far from its edges, and its arms are pinned by source text rather than behavior

`parseMajor` decides which refusal reaches the host: a recognized foreign major
is `incompatible_protocol` exit 6, everything else is `provider_protocol_error`
exit 13. Round 2 established that a plugin choosing between those two is the
harm. Three separate one-member narrowings of that gate migrate a real input
between the two classes and **the entire provhost suite passes**.

### F1a — the saturation guard's arithmetic edge is unwitnessed

Mutant M3, one token: `major > (math.MaxInt-step)/10` → `major > math.MaxInt/10`
(drops the `step` term — the classic off-by-one on this bound). It differs from
production only when `major == MaxInt/10` and the next digit is 8 or 9, i.e. at
the exact edge the guard exists to defend.

Measured through `DecodeResponse`:

| `protocol_version` | shipped | under M3 |
| --- | --- | --- |
| `2.0.0` (control) | ADMITTED | ADMITTED |
| `18446744073709551618.0.0` | `parseMajor`=(MaxInt,true), `incompatible_protocol` 6 | identical |
| `9223372036854775808.0.0` | (MaxInt,true), 6 | (-9223372036854775808,true), 6 |
| **`922337203685477580802.0.0`** | **(MaxInt,true) → `incompatible_protocol` exit 6** | **(2,true) → `provider_protocol_error` exit 13** |

The last row is a foreign major that **aliases native major 2**, which is
verbatim the invariant this leaf's own doc comment asserts — "never aliases a
small native major (18446744073709551618 must not read as 2)". `go test
./internal/provhost/ -count=1` passes under M3.

The witness is constructed, not searched: `922337203685477580` walks the
accumulator to exactly `MaxInt/10`; the next `8` overflows to `-2^63`; the
following `0` multiplies by 10, which is `≡ 0 (mod 2^64)`, resetting the
accumulator to 0; the final `2` lands on 2.

Why the existing rows do not see it: every witness in
`TestParseMajorNeverWraps` (`2^64+1`, `2^64+2`, `2^64`, 41 nines) sits far
above the guard's edge, where saturation is sticky and any narrowing still
yields `MaxInt`. The guard's only other mutant across all three rounds —
rev2/RV10 and rev3/G1-N4, `major > math.MaxInt` — never fires, so it is a
deletion wearing a narrowing label. Under the DoD row that is not accepted as
evidence, and here it demonstrably is not: the true narrowing survives.

### F1b — the digit-range arms are killed by the source-text census, not by behavior

`armParseBranches` keys each arm by the *source text* of its enclosing
condition, so a mutant that edits the condition orphans a declared witness and
dies in the inventory even when nothing behavioral notices. Two mutants that
PRESERVE the condition text and change behavior:

| Mutant | Production edit | Disposition |
| --- | --- | --- |
| M17 | insert `if digit == ':' { continue }` **above** the unchanged `if digit < '0' \|\| digit > '9' {` | **SURVIVED** |
| M18 | insert `if rest[i] == '/' { continue }` **above** the unchanged `if rest[i] < '0' \|\| rest[i] > '9' {` | **SURVIVED** |

Neither adds a `return 0, false`, so the derived arm set is byte-identical and
the bijection stays 167/167. Measured effect at the production entry:

| `protocol_version` | shipped | under M17+M18 |
| --- | --- | --- |
| `3:.0.0` | (0,false) → `provider_protocol_error` 13 | (3,true) → `incompatible_protocol` 6 |
| `3./.0` | (0,false) → 13 | (3,true) → 6 |
| `3.0./` | (0,false) → 13 | (3,true) → 6 |
| `a.0.0` (control) | 13 | 13 |
| `3.b.c` (control) | 13 | 13 |

The corpus reason is the same as F1a: the declared witnesses use `a`, `2a`,
`-1`, `+3`, `b`, `c` — none adjacent to the digit range. `/` (0x2F) and `:`
(0x3A) are the two characters that sit on the boundary the condition computes,
and neither appears anywhere in the package's test corpus.

For confirmation that the census is doing the killing today: my M5
(`rest[i] < '/'`) and M6 (`digit > ':'`) — the same behavioral weakenings,
written by editing the condition — die **only** to
`TestDerivedRefusalArmsAreAllWitnessed` and `TestWitnessedArmsAreAllDerived`.
No behavioral test appears in their kill lists.

### What round 4 has to produce

1. Behavioral rows that pin the rejected class **at its edges**, driven through
   `DecodeResponse`: the saturation guard's arithmetic edge (at minimum
   `9223372036854775808.0.0` and the aliasing witness
   `922337203685477580802.0.0`, asserting `incompatible_protocol` / exit 6 and
   `parseMajor(...) != ProtocolMajor`), and `/` and `:` in both the major and
   the rest components (`3:.0.0`, `3./.0`, `3.0./`, asserting
   `provider_protocol_error` / exit 13). Verify the fix one step away from the
   finding vector, not only on the literals named here.
2. A battery that carries, for the saturation guard, a **true narrowing** —
   the bound moved by one term, not a guard that never fires — and, for every
   arm whose killer today is the inventory, a **token-preserving production**
   mutant (condition text unchanged, behavior changed) with a behavioral test
   named as the killer. The current battery's token-preserving row (G1-C2)
   mutates the census itself, so it cannot detect this shape.
3. Report per arm which killer is behavioral and which is census-only. An arm
   whose only killer is `TestDerivedRefusalArmsAreAllWitnessed` /
   `TestWitnessedArmsAreAllDerived` is an unmeasured arm and should be stated
   as such if it stays that way.

---

## Notes (not blocking)

**N4 — the `foreignMajor` peek's widening direction survives.** M13b
(`recognized && major != ProtocolMajor` → `recognized || …`) arms the peek for
*unrecognized* versions too, which skips the whole v2 member-vocabulary block.
Measured: a bare frame with `2.abc.def` reports member `protocol_version` /
"unsupported protocol version" instead of member `ok` / "missing member".
Nothing is admitted, the code and exit stay `provider_protocol_error` / 13, and
the narrowing direction is killed (M14b), so this is a detail-level slide in
code outside this revision's delta. Worth a row if the arms are being tightened
anyway.

**N5 — M12 is a declared equivalent, not a defect.** Reverting
`Major: observedMajor` to a hardcoded `Major: 2` survives, because only `2.0.0`
reaches that line today. The `protocol.go` comment says exactly this ("the
contract below carries major 2 on every reachable path today"), and shifting
the major *off* the observed value (M11) is killed by
`TestDecodeResponseV2FailureWith130ErrorIsBoundTo100` and four others. Recorded
so it is not mistaken for a hole later.

**N6 — N2's stated bound landed in the outcome, not in the code.** The rev2
note asked for one sentence on the test. `TestNoProductionPathAttestsProvider
IdentityBinding`'s doc comment still does not mention that only `*ast.CallExpr`
is inspected; the bound exists only in `TASK-260906-vmzk0y_outcome-round3.md`.
The blind spot itself is real and correctly described (a `f := pkg.Verify…;
f(x)` binding would be missed; no such shape exists today).

**N7 (carry-forward, not raised this round) — the §7.A deferral names no board
item.** `provhost/doc.go` defers the v3 transport/Error-1.3.0 work to "this
task's outcome", `descriptor.go` defers to `provhost/doc.go`, and the registry
gap defers to "TASK-260906-vmzk0y's outcome" — a closed loop with no follow-up
task ID. The DoD wording is "an explicit stated bound names the deferring
task". This text is unchanged since rev2 and was accepted there, so it is
recorded, not re-litigated.

## What holds up (recorded so round 4 does not re-litigate it)

G-A, G-B, G-C(N1, N3) and G-D are closed. rev2's F1/F2/F4/F5 remain closed as
classes and their files are byte-identical to the accepted rev2 tree; F3's
bound reproduces; the §7.A ownership move is coherent and the registry digest
pin prevents self-minting; `Lift` fails closed on both sub-arms and its wire
message carries no local secret, with the causal-leak negative driving the real
`axerror.New` gate after asserting the control; the `(provider, 3) → 1.3.0`
binding is a real fail-closed table row, so the v3 premise test is not vacuous;
the candidate tree is exactly the worktree and leaf 1 is unregressed. Only F1
needs work, and it is one test-row group plus two battery rows.
