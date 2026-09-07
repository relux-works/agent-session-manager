# TASK-260906-vmzk0y — reviewer verdict, CR revision 4

**Verdict: CHANGES REQUESTED → `to-dev`.**
**repeat-of: `rev3/F1a`** (class only: a saturation guard witnessed only far
from its own edge, plus the same "bound witnessed far out of range" shape at a
second digit-range comparison). New sites, both inside this leaf's delta, both
in `internal/terminalbackend`. This is the **third consecutive round** of this
class, so the two-same-class routing rule applies.

Reviewer run `RUN-260906-aceb29`. Every number below was produced by this run;
nothing is accepted from the round-4 outcome document.

---

## Provenance (clean)

| Check | Result |
|---|---|
| Candidate tree OID recomputed from the worktree (detached index: `read-tree HEAD` + `add -A` + `write-tree`) | `6c8e29de0765747dc93c8565a756a888c6609ca6` — equals the CR record |
| `git ls-tree <tree> -- <path>` for the 4 new files | all four present with blobs |
| Leaf 1 packages vs `1d97474` (`canonicaljson`, `environ`, `dirnode`, `sessadapter`) | 0 files changed |
| Worktree tree OID re-verified after all 14 reviewer mutants | `6c8e29de…` unchanged; every mutated file restored byte-identically (SHA-256 asserted, `ALL RESTORED`) |

Mutants were applied to a scratch copy at `/tmp/vmzk0y-rev4-mut`, never to the
candidate worktree.

## Gates re-run by this reviewer

| Gate | Result |
|---|---|
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `gofmt -l internal/` | clean |
| `go test ./... -count=1` | exit 0, 22 packages `ok`, no FAIL |
| `go run ./internal/traceability/cmd/tracecheck` | ok — contracts=60, normative_sections=36, acceptance_cases=98 |

Not re-run by me (stated, not implied): `GOOS=windows` build/vet, `-race`,
coverage. The round-4 outcome states the same; no gate is claimed on their
behalf.

---

## G-A — the behavioural-versus-census split: CLOSED

The report exists and is **derived, not asserted**: `battery-round4.json`
records a per-mutant `killed_by` list, and the BEHAVIOURAL/CENSUS category is
recomputable from whether `TestDerivedRefusalArmsAreAllWitnessed` /
`TestWitnessedArmsAreAllDerived` appear in it. Recomputing from the JSON
reproduces the outcome table exactly: 24 rows, 22 applied / 21 KILLED /
1 SURVIVED(equivalent) + 1 NOT_APPLIED + 1 COMPILE_FAIL as distinct rows;
narrowing 14, narrowing/token-preserving 4, arm-deletion 2, census-only 1.

**No production arm is reported as census-only** — the single census-only row
(M15) mutates the census itself. So the spot-check asked for ("two it calls
census-only") has no candidate; I spot-checked two BEHAVIOURAL-ONLY, two BOTH
and the true saturation narrowing instead, running the census mask and the
behavioural mask as separate processes:

| Reviewer mutant | Reported | Census mask (census pair) | Behavioural mask | Verdict |
|---|---|---|---|---|
| RM17 `if digit == ':' { continue }` inserted above the unchanged major condition | BEHAVIOURAL ONLY | exit 0, 2 RUN / 2 PASS | exit 1 — `TestParseMajorDigitBoundariesRefuseAtEntry` | matches |
| RM18 `if rest[i] == '/' { continue }` inserted above the unchanged rest condition | BEHAVIOURAL ONLY | exit 0, 2 RUN / 2 PASS | exit 1 — same test | matches |
| RM5 `rest[i] < '/'` (condition edit) | BOTH | exit 1 — both census tests | exit 1 — same test | matches |
| RM6 `digit > ':'` (condition edit) | BOTH | exit 1 — both census tests | exit 1 — same test | matches |
| RM3 `major > (MaxInt-step)/10` -> `major > MaxInt/10` | BEHAVIOURAL | exit 0 | exit 1 — `TestParseMajorSaturationEdge` alone | matches |

Masks verified non-empty (2 and 236 `=== RUN` lines respectively), so no row is
a silently-empty selection.

## G-B — the edges: two of four sites closed

Closed and re-verified:

- **Saturation guard at `provhost.parseMajor`.** Largest admitted
  (`9223372036854775807`), first saturated (`…808`), the aliasing witness
  `922337203685477580802.0.0` and its ±1 neighbours are all driven through
  `DecodeResponse` with `incompatible_protocol` / exit 6 / `observed` asserted.
  RM3 confirms the guard literal cannot move by one term (killed by
  `TestParseMajorSaturationEdge` alone, zero census kill).
- **`/` (0x2F) and `:` (0x3A) at `parseMajor`,** in the major and in *both*
  rest positions — 8 rows through `DecodeResponse` asserting
  `provider_protocol_error` / exit 13 / member `protocol_version`. RM17/RM18
  confirm these kills are behavioural with the condition text byte-identical.

**Not done: the census of every remaining digit-range comparison in this leaf.**
The round-4 battery denominator is stated as "the nine version-classification
obligations reachable from `DecodeResponse`" — `provhost` only. Two further
sites inside this leaf's delta were never censused, and both carry the shape.

---

## F1 (BLOCKING) — `semverMajor`'s saturation guard is pinned only far from its edge

`internal/terminalbackend/terminalbackend.go:257`. The guard added by this leaf
in round 2:

```go
if major > (math.MaxInt-digit)/10 {
    return math.MaxInt
}
```

The true narrowing — drop the `digit` term, exactly the M3/F1a shape round 4
was tasked with closing at the other guard — **SURVIVES the entire
terminalbackend suite**:

| Reviewer mutant | Disposition | Suite |
|---|---|---|
| S1 `major > (math.MaxInt-digit)/10` -> `major > math.MaxInt/10` | **SURVIVED** | `go test ./internal/terminalbackend/ -count=1` exit 0, 457 RUN / 457 PASS, 0 FAIL |
| S2 `major > math.MaxInt` (never fires — delete-shaped) | KILLED | `TestParseProviderDescriptorProtocolVersionOverflowVectors` |

So the guard's only mutant evidence is the delete-shaped one the DoD explicitly
refuses ("A delete-only mutant proves only that the gate exists").

**Measured effect at the production entry point.** Under S1,
`ParseProviderDescriptor`:

```
PROBE semverMajor(922337203685477580801.0.0) = 1
PROBE Parse(protocol_version=922337203685477580801.0.0) -> ProtocolVersion="922337203685477580801.0.0" err=<nil>
```

A foreign protocol major is **admitted** as native major 1 — verbatim the
invariant the shipped `semverMajor` doc comment asserts ("a huge foreign major
compares foreign at every `!= 1` call site rather than aliasing a small native
one"). Shipped code saturates and refuses with `descriptor protocol version`.
The same aliasing reaches every `semverMajor(...) != 1` site:
`terminalbackend.go:235` (`New` / `RegisterExternal`), `:556`,
`manifest.go:1072`, `:1155`, `:1301`, `descriptor.go:168`.

**Why the corpus misses it.** All three existing witnesses saturate under the
weakened guard too — measured under S1:
`semverMajor("18446744073709551617.0.0") = 9223372036854775807`,
`semverMajor("9223372036854775808.0.0") = -9223372036854775808` (still != 1, so
still refused). The guard is only ever exercised deep inside the sticky region;
its edge (accumulator exactly `922337203685477580`, next digit 8 or 9) is
never reached. `9223372036854775807.0.0` (largest admitted) and
`9223372036854775808.0.0` (first saturated) have no rows at all.

**Next round:** carry the `TestParseMajorSaturationEdge` family to
`semverMajor` — largest admitted, that value plus one, and the aliasing witness
`922337203685477580801.0.0` (and its ±1 neighbours) — driven through
`ParseProviderDescriptor` and through one `New`/`CheckVersionTuple` call site,
asserting the named refusal arm, so S1 reddens.

## F2 (BLOCKING) — `descriptorGeometry`'s digit gate has no witness in the upper half of its rejected class

`internal/terminalbackend/descriptor.go:235`, `if digit < '0' || digit > '9'`.
This is the leaf's third digit-range comparison and was not censused.

| Reviewer mutant | Disposition | Suite |
|---|---|---|
| D0 (control) `if digit == '.' { continue }`, token-preserving | KILLED | `TestParseProviderDescriptorValueRefusals/columns_fraction`, `/rows_fraction` |
| D1 `if digit == 'E' { continue }`, token-preserving | **SURVIVED** | exit 0, 457/457 PASS |
| D2 `digit > '9'` -> `digit > 'E'` (bound shift) | **SURVIVED** | exit 0, 457/457 PASS |
| D3 `if digit == '+' { continue }`, token-preserving | SURVIVED | equivalent — `+` is only reachable after `e`/`E`, which is refused first |

The control dies, so the instrument is live.

**Measured effect.** Under D2, through the production entry:

```
shipped: Parse(columns=1E2) -> err=...descriptor geometry digits    (correct)
shipped: Parse(columns=1e2) -> err=...descriptor geometry digits    (correct)
D2:      Parse(columns=1E2) -> Columns:312, err=<nil>               (admitted, fabricated)
D2:      Parse(columns=1e2) -> err=...descriptor geometry digits
```

`1E2` is a valid JSON number naming 100; the gate admits it as **312** — a
fabricated in-range value the caller cannot distinguish from a real one. That
is the exact consequence class of rev1/F1, in the gate built to close rev1/F1.

**Why.** `/` (0x2F) and `:` (0x3A) are unreachable through `json.Number`, so the
reachable rejected class is `{-, +, ., e, E}`. The corpus witnesses `-` (`-80`),
`.` (`80.0`, `24.5`) and `e` (`1e2`) — but `e` is 0x65, **44 bytes above** the
true edge `'9'` (0x39). Every mutant upper bound `X` with `'E' <= X < 'e'`
admits `E` and survives; `E` is the one reachable member of that 32-value
window and has no row.

**Next round:** add `1E2` rows on `columns` and `rows` asserting
`descriptor geometry digits`, and record the census of all three digit-range
comparisons in this leaf with the reachable rejected class named per site
(including the declared-equivalent `/`, `:`, `+` cases), so the denominator
covers the gate under review rather than provhost alone.

---

## Confirmed sound (re-verified by this reviewer, not read)

- **Censuses fail closed**, control-planted:

  | Plant | Result |
  |---|---|
  | Undeclared refusal arm added to `descriptor.go` production | `TestDerivedRefusalArmsAreAllDeclared` **and** `TestDeclaredRefusalArmsAreAllDerived` both FAIL |
  | `var codePlantedByReviewer = "…"` in `internal/provider/lift.go` | `TestLiftCoversTheClosedCodeSet` FAILS |
  | `const codePlantedConst = "…"` in a second production file of the package | `TestLiftCoversTheClosedCodeSet` FAILS |

- **§7.A ownership**: `AdmitProviderDescriptor` named in `provhost/doc.go`
  stated bounds and in `ownership.v0.5.0.json`; `traceability.go` review digest
  re-pinned; `tracecheck` green. Both directions driven
  (`TestAdmitProviderDescriptorAdmitsMatchingBinding` /
  `…RefusesMismatch`, including the case-folding narrowing on generation).
- **v3 envelope at provhost**: `TestDecodeResponseWellFormedV3SuccessIsMismatch`
  and `TestDecodeResponseV3FailureWithValid130ErrorIsMismatch` — the latter
  first decodes the embedded error through the production `(provider, 3)` ->
  1.3.0 binding, so refusal is of a genuinely well-formed foreign error.
  `TestDecodeResponseV2FailureWith130ErrorIsBoundTo100` pins that the contract
  is selected by the observed major, not the document.
- **The lift seam**: `Lift` builds the wire message from the stable code alone,
  Details empty, cause local-only; four production-derived vectors (real
  `Discover`/`Verify` refusals) assert the secrets are present in `Error()` and
  absent from the marshalled wire object — a measured removal, not a vacuous
  non-mention. `TestNaiveLiftShapeIsRefusedByTheWireGate` refuses the naive
  shape in both the message and the details direction.
  `TestLiftRefusesRegisteredCodeWithoutLiftArm` pins the fail-closed default
  arm and discriminates it from the unregistered-code arm.
- **N6 closed**: the CallExpr-only bound now sits on
  `TestNoProductionPathAttestsProviderIdentityBinding`'s doc comment.
- **AC rows 1–13 and 15** re-run green in the full suite; row 14 remains a
  declared stated bound (14 driven + 1 bound, honestly reported).

## Notes (non-blocking)

- **N7 carried**: the §7.A deferral still names no follow-up board item. Per the
  round-4 brief this is the orchestrator's board-element freeze, not a producer
  omission.
- **N8**: `AdmitProviderDescriptor` still has zero production callers, disclosed
  in its own doc comment with the intended call order. Systemic and honest;
  restated so it is not lost.
- **N9**: D3 (`+`) is an equivalent mutant by reachability. Worth one declared
  line in the census rather than a test row.

## Reviewer battery

14 mutants, all applied to a scratch copy and restored byte-identically.
11 KILLED / 3 SURVIVED (D1, D2, S1 — the two blocking findings; D3 declared
equivalent). Full rows: `TASK-260906-vmzk0y_review-rev4-battery.json`, harness
`TASK-260906-vmzk0y_review-rev4-battery.py`.
