# TASK-260906-vmzk0y — review verdict, CR revision 2

Verdict: **changes requested** (one blocking finding, two notes).
repeat-of: `rev1/F3` — class only: *a claim in shipped text asserted from a
proxy signal rather than measured, which measurement contradicts*. It is a new
instance at a new site (`provhost.parseMajor`), introduced by the round-2 F2
fix. rev1/F3 itself is measurably closed (see G-D). rev1/F4 shares the
sub-shape "the code's own doc says otherwise".

Candidate tree `be5b2844b8996b4764e1cd7a0dfccf1facf478d2` — verified equal to
the worktree by a detached-index `read-tree HEAD` + `add -A` + `write-tree`,
and re-verified byte-identical after every mutant below (`TREE INTACT`).
`ls-tree` shows the four new files (`provider/lift.go`, `provider/lift_test.go`,
`terminalbackend/descriptor.go`, `terminalbackend/descriptor_test.go`) are
in-tree. 15 changed paths, matching the record.

## Verdict summary

Round 2 closes F1, F2, F4, F5 measurably and rewrites F3's bound around facts
that reproduce. 23 reviewer mutants applied, 23 killed, 0 survivors. The one
blocking finding is a **new** defect introduced by the F2 fix itself: the
saturating `return` in `parseMajor` short-circuits the minor/patch validation,
so a malformed version string is now reported as a *recognized* protocol major.
No admission results — but the shipped classification contradicts two doc
comments in this same diff, bypasses the v2 member-vocabulary gate for that
class, and **nothing in the suite pins it**: the corrected behaviour passes the
whole provhost suite unchanged.

## What I ran myself (nothing accepted from the outcome document)

| Gate | Result |
| --- | --- |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `go test ./...` | exit 0, no FAIL |
| `go test -cover` provider / provhost / terminalbackend | 97.8% / 85.8% / 94.2% — reproduces the outcome exactly |
| `gofmt -l internal/` | clean |
| `go run ./internal/traceability/cmd/tracecheck` | exit 0, contracts=60 acceptance_cases=98 — reproduces |
| provhost refusal-arm bijection | 167/167 derived arms witnessed |
| terminalbackend declared rows | 209 table rows + 1 funnel row = **210**, bijection green — the outcome's "210/210" is exact |
| candidate tree OID after all mutants | `be5b2844…` unchanged |

## G-A (blocking gate) — overflow class: **closed**

Census of every digit→bounded-number site **in this leaf** (3 sites; the
`strconv`-based sites elsewhere in the repo are out of leaf and untouched):

| Site | Shape | Refuses before accumulating? |
| --- | --- | --- |
| `terminalbackend/descriptor.go:243` `descriptorGeometry` | pre-check `value > 100` inside the loop, before the multiply | yes — accumulator is capped at 1009, cannot wrap on any platform |
| `terminalbackend/terminalbackend.go:259` `semverMajor` | `major > (MaxInt-digit)/10` → saturate | yes |
| `provhost/protocol.go:379` `parseMajor` | `major > (MaxInt-step)/10` → saturate | yes (but see G1) |

Driven through the real production entry points, **with the control first**:

| Vector | `AdmitProviderDescriptor` result |
| --- | --- |
| `columns = 80` (control) | ADMITTED as 80 |
| `columns = 1000` / `rows = 1000` (boundary) | ADMITTED as 1000 |
| `columns = 1001` / `rows = 1001` | REFUSED `descriptor geometry bound` |
| `0`, `-1`, `80.0`, `8e1` | REFUSED (`bound` / `digits` arms, correctly split) |
| `0080` | REFUSED at `document syntax` (strict decoder) |
| `2^63`, `2^64+1`, `2^64+500`, `2^64+1000` | REFUSED `descriptor geometry bound` |
| `55340232221128655348` (double overflow → 500) | REFUSED same arm |
| 100 nines, `1000000000000000000000000000000080` | REFUSED same arm |

Both members (`columns` *and* `rows`) driven separately for every vector, so the
two call sites of the shared helper are measured independently.

`semverMajor` through every production entry that consumes it, with the refusal
naming the major arm and not a downstream membership/ordering bound:

| Entry | `18446744073709551617.0.0` (wraps to 1) | `36893488147419103233.0.0` (double overflow → 1) |
| --- | --- | --- |
| `ParseProviderDescriptor` | REFUSED `descriptor protocol version` | REFUSED same |
| `New(…)` | REFUSED `protocol_versions major 1` | REFUSED same |
| `CheckVersionTuple(…)` | REFUSED `protocol_version major 1` | REFUSED same |

**Accumulation planted back at each site** (the fix is not a bypassable
pre-check):

| Mutant | Class | Verdict | Killing test |
| --- | --- | --- | --- |
| RV1 geometry pre-check deleted | arm-deletion | KILLED | `TestParseProviderDescriptorGeometryOverflowVectors` — `want refusal "descriptor geometry bound", got admission` |
| RV2 final bound `1000`→`1001` | narrowing | KILLED | `…GeometryOverflowVectors` + `ValueRefusals/columns_over_bound` |
| RV3 lower bound `1`→`0` | narrowing | KILLED | `ValueRefusals/columns_zero`, `/rows_zero` |
| RV5 `semverMajor` guard neutered to `major > math.MaxInt` (never fires ⇒ accumulation restored) | narrowing | KILLED | `TestParseProviderDescriptorProtocolVersionOverflowVectors` |
| RV10 `parseMajor` guard neutered the same way | narrowing | KILLED | `TestParseMajorNeverWraps` — `parseMajor("18446744073709551617.0.0") = 1` |
| RV4 `semverMajor` saturation deleted outright | — | COMPILE_FAIL (unused `math`) — control row, **not** a kill; re-anchored as RV5 |

Note on mutant class for `descriptorGeometry`: narrowing the *pre-check*
constant upward cannot create a hole (with threshold T the accumulator peaks at
`T*10+9`, so a wrap needs T ≈ MaxInt/10, i.e. deletion). Arm-deletion is the
correct class for that guard; the narrowing obligation is discharged by RV2/RV3
against the *final* bound, which is a separate rule. Both arms carry the same
`Detail`, and the inventory declares them as occurrences #1 and #2 — RV7
confirms the bijection notices when either disappears.

## G-B (blocking gate) — F4's registered-but-unlifted case: **closed**

Which answer was chosen: **fail closed** — `Lift` returns `(nil, failure)` and
mints no wire object. Independently confirmed that `not_found` really is
registered (so the round-1 hole was reachable) and that every registered code
outside the three lift arms takes the same closed path:

| `Lift(Error{code: …})` | registry | result |
| --- | --- | --- |
| `invalid_config` | registered, exit 3 | WIRE, static message, exit 3 |
| `local_precondition_failed` | registered, exit 3 | WIRE, static message, exit 3 |
| `integrity_failure` | registered, exit 9 | WIRE, static message, exit 9 |
| **`not_found`** | **registered, exit 4** | **REFUSED, wire object nil, NOT the unregistered arm, input handed back** |
| `capability_unavailable` | registered, exit 6 | REFUSED, same closed sub-arm |
| `incompatible_protocol` | registered, exit 6 | REFUSED, same closed sub-arm |
| `timeout`, `internal_error`, `""`, `bogus_code` | unregistered | REFUSED with `ErrUnregisteredCode` |

Mutants:

| Mutant | Class | Verdict | Killing test |
| --- | --- | --- | --- |
| RV13 default arm fails open again (registered code mints a generic-message wire object) | narrowing | KILLED | `TestLiftRefusesRegisteredCodeWithoutLiftArm` |
| RV14 registry check deleted from the default arm | arm-deletion | KILLED | `TestLiftRefusesUnregisteredCode` |
| RV15 lift appends the local detail to the wire message | narrowing | KILLED | `TestLiftCarriesDiscoveryFailuresToTheWire` — `lifted message = "…: cannot list plugin_dirs[0] directory \"/plugins\""` |

The two sub-arms are pinned by two different named tests with different
`errors.Is` assertions, so neither can pass on the other's behaviour. The
causal-leak negative drives the real `axerror.New` gate and asserts
`ErrCausalLeak` by name, with the control (the secret is present in the local
rendering) asserted first in both directions.

## G-C (blocking gate) — F5's derivation and the four required plants: **closed**

All four plants present and each one is real — I neutered the corresponding
production behaviour and watched the named plant die:

| Mutant | Class | Verdict | Killing test |
| --- | --- | --- | --- |
| RV16 census drops `token.VAR` (the `code` token preserved) | census-only, token-preserving | KILLED | `TestDerivedCodesIncludesVarBinding` — `derived codes = map[held_code:true], want "planted_var_code"` |
| RV17 census reads only `provider.go` | census-only, token-preserving | KILLED | `TestDerivedCodesScansEveryPackageFile` (+ two others) |
| RV19 code-prefixed **type alias** no longer fails closed | census-only | KILLED | `TestDerivedCodesFailsClosedOnUnclassifiableBinding` |
| RV18b non-literal value `continue`s instead of erroring | census-only | KILLED | `…FailsClosedOnUnclassifiableBinding` — `identifier-valued derivation = map[base_code:true]` |

Grouped consts are covered by `TestDerivedCodesIncludesGroupedConsts`; the
inherited-empty-value-list and shared-value-list shapes fail closed by
construction. The derivation is unclassifiable-fails-closed, not skip.
Denominator is production-derived (3 codes) and `TestLiftCoversTheClosedCodeSet`
runs it in both directions (unwitnessed code ⇄ orphan vector).

## G-D (blocking gate) — F3's bound: **rests on reproducible facts**

Every sub-claim of the rewritten bound checked against source, not against the
outcome:

| Claim in `identity.go` | Verified |
| --- | --- |
| `VerifyObjectIdentity` on the §5.5 example returns the example's own `record_id`, nil error | yes — `sha256:c879d766…e220c2` is literally the example's `record_id`; the test asserts field *and* digest, not nil-error-only |
| `VerifyObjectIdentity` has **zero** production call sites repo-wide | yes — independent grep: only the definition (`canonicaljson/canonical.go:309`) and two comments |
| `CheckIdentity` requires `record_id` to be a well-formed digest but never recomputes it | yes — `identity.go:169`, `isDigest` only |
| the conjoined `CalculateObjectIdentity` entry discards its digest | yes — `identity.go:288`, `_, _, err :=` |
| the three terminal schemas *do* recompute the omit-self digest | yes — `terminalbackend/manifest.go:682` `checkIdentity` |
| the old "merely illustrative" justification is withdrawn *in the bound text* | yes — stated explicitly, not paraphrased away |

**Control plant:** I added a production call to `VerifyObjectIdentity` in
`internal/provhost/` and `TestNoProductionPathAttestsProviderIdentityBinding`
reddened naming the site; removed, it passes and logs `125 production files, 2
parsed, 0 production call sites` — and 125 is exactly `find internal cmd -name
'*.go' -not -name '*_test.go' | wc -l`. The scan is not vacuous.

## G-E — F6 / F7 notes: **corrected**

- **F6** — `AdmitProviderDescriptor` still has zero production callers, and
  `descriptor.go:20-31` says so in the code, naming the future call path, the
  required order (parse, match, then launch) and the two miswirings that would
  admit a stale binding. Independently confirmed: no non-comment caller exists.
  (This is not unusual here — the repository has no application entry point at
  all yet; `cmd/` does not exist.)
- **F7** — the outcome now lists all 15 changed paths and they match the record.
  See note N1 below for the one count that is still slightly generous.

## G-F — battery and provenance

Denominator re-derived independently rather than accepted. My own battery,
reported with the classes split and the controls as distinct rows:

| Class | Applied | Killed | Survived |
| --- | ---: | ---: | ---: |
| narrowing | 13 | 13 | 0 |
| arm-deletion | 6 | 6 | 0 |
| census-only | 4 | 4 | 0 |
| **total applied** | **23** | **23** | **0** |

Distinct rows, NOT counted as applied mutants:

| Control row | Count | Disposition |
| --- | ---: | --- |
| NOT_APPLIED (anchor absent) | 1 | RV18 — indentation drift; re-anchored as RV18b, then KILLED. Not a pass. |
| COMPILE_FAIL | 2 | RV4 (unused `math`), RV12 (unused `observedMajor`) — re-anchored as RV5 and RV20, then KILLED. Not kills. |

RV10 and RV11 are the same mutation measured twice (targeted `-run` mask, then
the whole package) and are counted once.

Additional kills beyond the gates above: RV6 generation match dropped; RV7
member-set count dropped (killed by the *inventory bijection*, not only the
parse test); RV8 version match narrowed to implementation-only; RV9 descriptor
major gate dropped; RV20 failure-error contract major shifted off the observed
major; RV22/RV23 ownership-registry 7.A claim self-minted / pointed at a
non-existent declaration — both killed by `traceability`, so the registry
cannot mint an ownership claim; RV24 instance-id UUIDv7 gate degraded to a
length check; RV25 binding-digest match dropped; RV26 backend-id match dropped;
RV27 generation match folds case.

Provenance: `git status` shows exactly the 15 paths, unchanged. Leaf 1's own
packages (`canonicaljson/*`, `environ/*`, `terminalbackend/manifest.go`) are
untouched from `1d97474`. Of the four files this leaf shares with leaf 1,
`identity.go` and `refusal_arm_inventory_test.go` are pure additions (0 deleted
lines), and the `ownership.v0.5.0.json` deletions are confined to the §7.A
`section_binding` row, which leaf 1 never touched. The referenced acceptance
case `terminal-provider-descriptor-7a` exists in the registry.

---

## G1 (blocking) — the F2 fix reclassifies malformed versions as a recognized foreign major

`parseMajor` saturates with an early **`return`** (`protocol.go:380`), which
exits the function before the minor/patch validation at `protocol.go:386-395`
ever runs. A version string whose first component overflows is therefore
reported as *recognized* no matter what follows the first dot.

Measured through the production entry point:

| `protocol_version` in a frame | `parseMajor` | `DecodeResponse` |
| --- | --- | --- |
| `2.0.0` (control) | `(2, true)` | ADMITTED |
| `3.0.0` | `(3, true)` | `incompatible_protocol`, exit 6 |
| `18446744073709551618.0.0` | `(MaxInt, true)` | `incompatible_protocol`, exit 6 — correct, this is F2's fix working |
| `2.abc.def` | `(0, false)` | `provider_protocol_error`, exit 13 |
| **`99999999999999999999.abc.def`** | **`(MaxInt, true)`** | **`incompatible_protocol`, exit 6** |
| **`99999999999999999999.0.x`** | **`(MaxInt, true)`** | **`incompatible_protocol`, exit 6** |
| **`18446744073709551618..0`** (empty minor) | **`(MaxInt, true)`** | **`incompatible_protocol`, exit 6** |

The three bold rows are not versions at all. Two shipped doc comments say they
must not take this arm:

- `protocol.go:359-360` — "*parseMajor extracts the major from a strict numeric
  X.Y.Z version. **Anything else is not a recognizable major, so the frame is
  unusable rather than a mismatch.***"
- `protocol.go:403-404` — "*A recognizable foreign major yields
  incompatible_protocol; **every other unusable frame yields
  provider_protocol_error**.*"
- and the new paragraph justifying the change asserts the narrower class it does
  not enforce: "*an **all-numeric** giant is observably foreign*".
  `99999999999999999999.abc.def` is not all-numeric.

### Why it matters beyond the exit code

The `foreignMajor` peek (`protocol.go:437-451`) arms on the same predicate, and
arming it **skips the entire v2 member-vocabulary check** — the `ok` member read
and `checkResponseMembers` both live inside `if foreignMajor == ""`
(`protocol.go:454`). Measured on a frame carrying nothing but `protocol` and
`protocol_version`:

| bare frame version | refusal |
| --- | --- |
| `2.abc.def` | `provider_protocol_error` — `missing member` |
| `99999999999999999999.abc.def` | `incompatible_protocol` — member checks never ran |

That is a bypass path around a gate for a class of frames. It ends in a refusal
today, so nothing is admitted — but the class that reaches the peek is now
wider than the class the peek was written for, and a plugin chooses which
refusal (and which exit status, 6 vs 13, i.e. version-negotiation vs
malformed-frame) the host reports.

### Nothing measures this

I applied the candidate fix — replace the early `return math.MaxInt, true` with
`major = math.MaxInt; continue`, so the loop still validates every digit and
still falls through to the minor/patch checks — and:

- all three bold rows return to `(0, false)` / `provider_protocol_error` exit 13;
- `18446744073709551618.0.0` still saturates to `incompatible_protocol` exit 6,
  so F2 stays closed;
- **`go test ./internal/provhost/ -count=1` passes unchanged.**

The whole suite is indifferent between the shipped behaviour and the corrected
one. The shipped classification was never chosen — it fell out of the `return`.

### Why the round-2 evidence did not catch it

The outcome states, as the proof that nothing moved: "*the provhost parse-arm
census derives the identical set (saturation adds no `return 0, false`
branch)*". That is true and it is exactly the blind spot: `armParseBranches`
enumerates **rejection branches**, so a change that moves inputs *between*
existing branches is invisible to it. A branch census is not an effect census.
This is a property inferred from a proxy signal and reported as fact — the same
shape as rev1/F3, which is why `repeat-of` names it.

### What the next round has to produce

1. Make `parseMajor` validate the whole `X.Y.Z` shape before it reports a
   recognized major (the `major = MaxInt; continue` form above is sufficient and
   is already proven compatible with the existing suite).
2. Add a named test row driving at least `<giant>.abc.def` and `<giant>..0`
   through `DecodeResponse` and asserting `provider_protocol_error` / exit 13,
   alongside the existing `TestParseMajorNeverWraps` rows asserting exit 6 for
   the all-numeric giants — so the two classes are pinned apart and a future
   short-circuit reddens.
3. Correct the outcome's "no gate was weakened" sentence: state the arm-shift
   that *was* intended (all-numeric giants: 13 → 6) as measured, and drop the
   branch-census sentence as evidence for classification stability, or add the
   effect measurement that would actually support it.

---

## Notes (not blocking)

**N1 — the AC ratio is 14 driven + 1 stated bound, reported as "15 of 15
driven".** Row 14 (F6, future caller recorded) says "prose by nature" in its own
Driving-test cell. The DoD explicitly permits a stated bound in place of a
driving test, so the row is *fine* — but counting it inside a "15 of 15 driven"
ratio is the same generosity F7 flagged in round 1. Report it as
`14 of 15 driven, 1 stated bound`.

**N2 — the attestation tripwire matches call expressions only.** `f :=
canonicaljson.VerifyObjectIdentity; f(x)` would not be seen, because only
`CallExpr.Fun` is inspected. No such shape exists today (0 files reference the
symbol outside its definition), and the scan is otherwise strong — AST-based,
comment-immune, test-file-excluded, fails closed on `scanned == 0`, and
control-planted. Worth one sentence as a stated bound on the test rather than a
code change.

**N3 — battery script provenance.** The outcome names `/tmp/mutA.py` and
`/tmp/mutB.py` as the battery scripts. They happen to still exist, but `/tmp` is
not a task-scoped artifact path; project convention is `.temp/<TASK-ID>/`. Not a
correctness issue, only reproducibility.

## What holds up (recorded so the next round does not re-litigate it)

F1, F2, F4, F5 are closed as *classes*, not as sites; F3's bound reproduces on
every sub-claim; the §7.A ownership move is coherent and the registry cannot
mint it without the digest pin noticing; the descriptor match gate refuses on
every one of its five dimensions under mutation; the lift's wire message carries
no local secret and the causal-leak negative drives the real gate; the candidate
tree is exactly the worktree and leaf 1 is unregressed. Only G1 needs work.
