# TASK-260906-vmzk0y — review verdict, CR revision 6

**Verdict: ACCEPTED → `accept_cr` (element routes to `integrating`).**

**repeat-of:** `none`. Rev5's F1 and F2 are both closed, measured by my own
plants and mutants rather than accepted from the round-6 report.

Reviewer run `RUN-260907-e5e330` (`TASK_BOARD_RUN_ID`). Candidate tree
`817b1a68d844f7f5b6734197526a6661682df164`.

---

## Provenance (G-E)

| Check | Result |
| --- | --- |
| `git rev-parse HEAD` vs CR base OID | equal (`1d97474…`) — no commit past the checkpoint |
| `git rev-list --count 1d97474..HEAD` | `0` |
| Tree recomputed from the worktree via a detached index (`GIT_INDEX_FILE` copy + `read-tree HEAD` + `add -A` + `write-tree`) | `817b1a68d844f7f5b6734197526a6661682df164` — equals the CR record |
| `git ls-tree -r` for the six untracked new files (`descriptor.go`, `descriptor_test.go`, `digit_guard_census_test.go`, `semver_major_edge_test.go`, `lift.go`, `lift_test.go`) | all present in the recorded tree — not an index-tree artefact |
| Leaf-1 packages (`internal/canonicaljson`, `internal/environ`) vs `1d97474` | untouched — absent from the rev6 diffstat, and `internal/terminalbackend/manifest.go` (leaf 1's file) is unmodified |
| Tree re-derived **after** my whole 18-mutant battery and 9 control plants | `817b1a68…` again — every production and test file restored byte-for-byte |

Gates re-run by me, not accepted from the report:

| Gate | Result |
| --- | --- |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `gofmt -l internal/` | clean |
| `go test ./... -count=1` | exit 0, 22 `ok`, 0 `FAIL` |
| `go run ./internal/traceability/cmd/tracecheck` | ok — contracts=60, normative_sections=36, acceptance_cases=98 |
| `go test ./internal/terminalbackend/ -run TestDigitCensus -v` | `digit-guard census: 9/9 derived sites rowed across 19 production files` |

---

## G-A (blocking) — the fail-closed directions, all control-planted by me

The brief required three directions beyond the unregistered-site one it had
already driven, plus the two identifier-keyed bypass shapes. I planted all of
them. Every plant was applied, run, and reverted under a SHA-256 guard.

| Plant | Direction | Result |
| --- | --- | --- |
| P1 new production file, gate reached through a **var binding** (`octet := raw[i]`) | unregistered site | **RED**, exit 1 — `unregistered char site terminalbackend\|zzplant_p1.go\|zzPlantVarBinding\|char\|octet < '0' \|\| octet > '9'` |
| P2 new production file with an **aliased import** (`import js "encoding/json"`), gate on the aliased type's value | unregistered site | **RED** — `…\|zzPlantImportAlias\|char\|raw[i] < '0' \|\| raw[i] > '9'` |
| P3 new `*10` accumulator with a threshold guard | unregistered **bound** site | **RED** — `unregistered bound site …\|zzPlantAccumulator\|bound\|total > (1<<62-9)/10` |
| P4 `raw[i] == '5'` — equality against a digit rune | **cannot classify** | **RED**, and it refuses rather than skipping: `unclassifiable digit-guard shapes (1) … unclassifiable guard in zzplant_p4.go (zzPlantEquality): raw[i] == '5'` |
| P8 declared row edited to `value > 101`, production untouched | **orphan row** | **RED**, both directions at once — `unregistered bound site …descriptor.go\|descriptorGeometry\|bound\|value > 100` **and** `orphan row …\|bound\|value > 101` |
| Rev5's own inline plant (inert `generation[i] < '0' \|\| generation[i] > '9'` inside the existing `ParseProviderDescriptor`) | unregistered site | **RED** — `unregistered char site terminalbackend\|descriptor.go\|ParseProviderDescriptor\|char\|generation[i] < '0' \|\| generation[i] > '9'`. This is the exact plant the rev5 census did not see. |

In every case only `TestDigitCensusCoversEveryLeafGuard` failed, and the message
named package, file, function, shape and printer-normalized expression. The
denominator is directory-derived (19 production files across both packages), so
a new file is scanned with no registration step — P1/P2/P3 prove that, not the
prose.

## G-B (blocking) — the rows carry real obligations

**Char rows / reachable rejected class.** `descriptorGeometry`'s row claims the
reachable class is `{-, ., e, E}` (`+` legal only after `e`/`E`, which refuse
first — stated in the row) and that `/`/`:` are unreachable. Both halves are
measured, not asserted: `TestParseProviderDescriptorValueRefusals` drives `-80`,
`80.0`, `1e2`, `1E2` on `columns` and `24.5`/`1E2` on `rows`;
`TestDigitCensusAdjacentsNeverReachGeometryGate` drives `1/2` and `1:2` on both
members and pins them to the *decoder's* `document syntax` arm, so the
unreachability claim reddens if it ever stops being true. Independent narrowings
I ran: dropping the upper half (`digit < '0'` only) and shifting the lower bound
(`digit < '-'`) are both killed by `TestParseProviderDescriptorValueRefusals`.

**Accumulation bounds — are all three present?** There are exactly three `* 10`
accumulators in the two packages (`grep`: `descriptor.go:246`,
`terminalbackend.go:261`, `protocol.go:387`) and all three are guarded; the
census rows four bound sites (`descriptorGeometry` carries both the pre-multiply
guard and the post-loop range). Nothing is scoped out. The rev5 mechanism — a
prose census naming two of three — cannot recur, because the set is derived and
P3 proves a fourth accumulator fails closed.

**Witnesses redden at the edge and stay green at the equivalents.** My own
mutants, run with the census mask and the behavioural mask separately:

| My mutant | Site | Behavioural result |
| --- | --- | --- |
| `value > 100` → `> 922337203685477580` (the F1 edge) | S6 | **KILLED** by `TestParseProviderDescriptorGeometryOverflowVectors` |
| `value > 100` → `> 1000` | S6 | behaviourally **green**, census red — equivalent, as declared |
| `value > 100` → `> 92233720368547758` | S6 | behaviourally **green**, census red — equivalent, as declared |
| `major > (MaxInt-digit)/10` → `> MaxInt/10` | S8 | **KILLED** by `TestSemverMajorSaturationEdge` |
| `major > (MaxInt-step)/10` → `> MaxInt/10` | S9 | **KILLED** by `TestParseMajorSaturationEdge` |
| `value < 1 \|\| value > 1000` → `> 1002` | S7 | **KILLED** by `…GeometryOverflowVectors` + `…ValueRefusals` |
| `value < 1` → `value < 0` | S7 | **KILLED** by `…ValueRefusals` |
| hex `digit <= '9'` → `<= ':'` | S2 | **KILLED** by `TestDocumentHexEscapeDigitBoundaryRefused` |
| hex `b <= '9'` → `<= ':'` | S5 | **KILLED** by `TestProductionEntriesRefuseHexDigitBoundaryEscapes` |
| `parseMajor` major `digit > '9'` → `> ':'` | S3 | **KILLED** by `TestParseMajorDigitBoundariesRefuseAtEntry` (+ both arm-inventory bijections) |
| `parseMajor` rest `rest[i] > '9'` → `> ':'` | S4 | **KILLED** by the same |

The two hex boundary tests are new this round and they are the reason S2/S5 are
now pinned; under rev5's corpus the equivalent edits survived.

## G-C — F1 and the doc claim

`TestParseProviderDescriptorGeometryOverflowVectors` carries the 37-digit
witness `9223372036854775808000000000000000001` on **both** `columns` and
`rows`, and its doc comment now states the specific edge it pins
(`floor(MaxInt/10)`) instead of the bare "never on a wrapped accumulator"
claim rev5 falsified. I verified the claim is now true by applying the mutant
that made it false: the witness reddens at the edge and is verdict-equivalent at
`1000` and `92233720368547758`, so the row pins the dangerous window and nothing
wider.

I also shipped a **token-preserving** mutant on S6 that the producer's battery
does not contain — `if value > 100 && len(digits) < 3` keeps the literal `100`
and every token, and disables the guard for exactly the literals that can wrap.
**KILLED** behaviourally by `…GeometryOverflowVectors`.

## G-D — battery, denominator and split

The producer's `battery-round6.json` is 37 rows over the 9 derived sites with
`applied` / `NOT_APPLIED` (R6-X1, absent `>=` anchor) / `COMPILE_FAIL` (R6-X2,
unused `math` import) as distinct dispositions, `narrowing` (16),
`narrowing/token-preserving` (8), `arm-deletion` (9) and `census-only` (1) as
distinct classes, and a per-mutant `BOTH` / `BEHAVIORAL-ONLY` / `CENSUS-ONLY`
split derived from real exit codes on two separately calibrated masks (census =
2 RUN lines; behavioural = 460 tb + 671 ph RUN lines, green pristine).

I did not accept it. My independent battery, same two masks:

**18 applied / 17 killed / 1 survived (equivalent) / 1 COMPILE_FAIL / 0 NOT_APPLIED.**

- 15 killed behaviourally with the named test, 2 census-only (both declared
  equivalence probes).
- The one survivor is `ContainingContract{ID: ProtocolID}` → `"urn:ax:protocol:rpc"`.
  It is a **provably equivalent mutant**, not a hole: `axerror.staticBindings`
  maps `{rpc, 2}` to `Version100` exactly as it maps `{provider, 2}`, so no
  input can distinguish them. Hardcoding the *major* instead (`Major: 3`) is
  **KILLED** by four tests including `TestDecodeResponseV2FailureWith130ErrorIsBoundTo100`.
- One further attack worth recording, from a shape that has bitten this repo
  before (a saturating return skipping later validation): rewriting `parseMajor`'s
  `major = math.MaxInt; continue` into `return math.MaxInt, true` is **KILLED**
  by `TestParseMajorSaturationStillValidatesTheRest` and
  `TestDecodeResponseUnrecognizedVersionRunsMemberRules`. The doc claim that
  `99999999999999999999.abc.def` stays unrecognized is pinned, not asserted.
- Arm-deletion on the foreign-major peek (`&& false`) is **KILLED** by
  `TestDecodeResponseForeignMajorPrecedesMemberRules`, `BEHAVIORAL-ONLY` — the
  split is real, not a census artefact.

## AC substance

Rev5 verified §7.A ownership, the v3 both-directions tests, the Error 1.3.0
binding claim, and the Lift seam by attack, and I am not re-litigating them; I
re-ran the whole suite and the two mutants that would falsify the parts this
round touches. Spot-checks I did repeat: `axerror.staticBindings` really carries
`{provider, 3} → Version130`, so "admits 1.3.0 by construction" is true;
`Lift`'s tests drive the real `Discover`/`Verify` entry points, assert each
secret **is** present in the local rendering before asserting its absence from
the marshalled wire object (so the absence is a measured removal), and drive the
naive shape through the production `axerror.New` gate in both the message and
the details direction.

`AdmitProviderDescriptor` and `Lift` have no production caller. That is
structural, not concealed: there is no `cmd/` in this repository yet, and both
files say so in their headers and name the call order the future caller must
use.

---

## Non-blocking finding N-R1 — the census's stated bounds are incomplete

Recorded because the brief asked for exactly this to be named rather than left
for the next round. The census's shape definition is "an ordering comparison
against a decimal **digit rune** literal", so `digitGuardDigitRune` requires
`token.CHAR`. Three spellings of a semantically identical digit gate therefore
enumerate to nothing, and I measured each (census stayed green at 9/9, 20 files
scanned):

| Probe planted in production | Census |
| --- | --- |
| `raw[i] < 0x30 \|\| raw[i] > 0x39` (integer code points) | **green — invisible** |
| `const lo = '0'; const hi = '9'; raw[i] < lo \|\| raw[i] > hi` (named rune constants) | **green — invisible** |
| `strconv.Atoi` delegation with no comparison | **green — invisible** |

Additionally, `semverPattern`'s `[0-9]`/`[1-9]` classes in both
`terminalbackend.go:81` and `provhost/manifest.go:47` are live digit-admission
rules that the enumerator does not see and the stated-bounds paragraph does not
name.

**Why this is not blocking.** I grepped both packages: **no such site exists
today** — there is no integer-spelled or named-constant digit comparison
anywhere in `internal/terminalbackend` or `internal/provhost`, and all three
`* 10` accumulators are rowed. So this is an incomplete stated-bounds paragraph,
not a live hole, and it is the opposite of the rev1–rev5 class, where each round
had a real unpinned production site. The two identifier-keyed bypass shapes that
actually walk through gates in this repository — import alias and var binding —
are both caught, which I proved rather than read.

**What the next leaf should do:** add these three spellings and the regexp digit
classes to the stated-bounds block of `digit_guard_census_test.go`, or widen
`digitGuardDigitRune` to accept an integer literal whose value is in `'0'..'9'`
and a constant identifier bound to a digit rune. It is a comment or a ten-line
classifier change, not a revision of this delta.

---

## Verdict

Rev5's two blockers are closed and the closure is measured. The instrument
replacement the brief asked for landed: the denominator is derived from
production, it fails closed in every direction the brief named plus the orphan
and unclassifiable directions, and the class that survived five rounds is now
pinned behaviourally at every one of its nine sites by mutants I wrote myself.
Accepted.
