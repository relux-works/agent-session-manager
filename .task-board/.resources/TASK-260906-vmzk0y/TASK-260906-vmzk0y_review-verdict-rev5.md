# TASK-260906-vmzk0y — review verdict, CR revision 5

**Verdict: changes requested → `to-dev`.**

**repeat-of:** rev4 F1/F2 (and rev3 F1a, rev1 F1) — *a digit-accumulation bound
witnessed away from its own arithmetic edge*. This is the **fourth consecutive
round** of that class, now at the one accumulation bound in this leaf that no
census row has ever named.

Reviewer run `RUN-260907-ee0ed6`. Candidate tree
`d6253f58fe72a84c0900329c686300f232374874` — verified, see Provenance.

---

## What is right, and is not being re-litigated

I re-ran every gate myself rather than accepting the round-5 report.

| Check | Result |
| --- | --- |
| `go build ./...`, `go vet` (4 touched pkgs) | exit 0 |
| `gofmt -l internal/` | clean |
| `go test ./... -count=1` | exit 0, 22 packages ok, 0 FAIL |
| Candidate tree OID recomputed from the worktree via a detached index | `d6253f58…` — equals the CR record |
| `git ls-tree` for the six new files | all present in the recorded tree (not an index-tree artefact) |
| Leaf-1 packages (`internal/canonicaljson`, `internal/environ`) vs `1d97474` | untouched — absent from the rev5 diffstat |
| Production files byte-identical after my 28-mutant battery | SHA-256 asserted per file; tree re-derived to `d6253f58…` |

Substance verified by attack, not by reading:

- **§7.A ownership (AC 1, 6, 11).** `ownership.v0.5.0.json` `section:7.A` now
  binds `internal/terminalbackend/descriptor.go`→`AdmitProviderDescriptor`
  (was `catalog.ForRelease`); `terminal-provider-descriptor-7a` is a registered
  acceptance case; `reviewedOwnershipCanonicalSHA256` re-pinned; `tracecheck`
  ok (60/36/98). provhost `doc.go` names terminalbackend as the owner.
- **v3 both directions (AC 3).** `TestDecodeResponseWellFormedV3SuccessIsMismatch`
  and `TestDecodeResponseV3FailureWithValid130ErrorIsMismatch` refuse with the
  named arm; `TestDecodeResponseV2FailureWith130ErrorIsBoundTo100` is the
  opposite direction. Narrowing the mismatch arm (`major != ProtocolMajor` →
  `major > ProtocolMajor`, PH-04) is **killed**.
- **Error 1.3.0 (AC 4).** The doc claim that the binding "admits 1.3.0 by
  construction" is true: `axerror.staticBindings` carries
  `{provider, 3} → Version130`. The `observedMajor` refactor is behaviourally
  inert today (PH-03 survives, as the producer states in the comment) — that is
  a declared bound, not a defect.
- **Lift seam (AC 8, 9, 10).** The naive shape is genuinely refused by the real
  gate, not by a test double: `Message: failure.Error()` (PR-02b) fails with
  *"message reproduces the local cause verbatim"* from `axerror.New` on the
  production path. Leaking `failure.detail` into `Details` (PR-01) and dropping
  `Cause` (PR-04) are both killed. The unregistered/unlifted code sub-arms
  (PR-03) are killed.
- **Round-4 regressions closed.** `semverMajor` narrowing `> (MaxInt-digit)/10`
  → `> MaxInt/10` (TB-01) is now **killed** by `TestSemverMajorSaturationEdge`.
  `digit > 'E'` (TB-03) is **killed at both `columns` and `rows`**, and the
  token-preserving variants (TB-08 `&& digit != 'E'`, TB-09 `&& digit != 'e'`)
  are killed behaviourally with **no census failure** — the split is real.
- **G-C split.** Confirmed. For every provhost mutant killed by BOTH
  (PH-02/05/06) I re-ran the behavioural mask alone: `-run
  TestParseMajorDigitBoundariesRefuseAtEntry` is non-empty (1 RUN) and FAILS on
  its own (`parseMajor("3:.0.0") = (40, true)`). No production arm in this delta
  is census-only.

Battery: **28 applied / 23 killed / 5 survived / 1 COMPILE_FAIL control / 0
NOT_APPLIED**. Four survivors are equivalent and bounded below (TB-04, TB-05,
TB-10, PH-03). One is a hole.

---

## F1 (blocking) — `descriptorGeometry`'s overflow guard is unwitnessed at its own edge

`internal/terminalbackend/descriptor.go:238`

```go
if value > 100 {
    return 0, &Error{Code: CodeProtocolError, Detail: "descriptor geometry bound"}
}
```

Narrow it to its **own arithmetic edge** — `value > 922337203685477580`
(= `floor(math.MaxInt/10)`, the largest threshold that still lets
`value*10+digit` overflow int64) — and the whole suite stays green:

```
--- ADMITTED at the production entry point ---
columns: 9223372036854775808000000000000000001  ->  err=<nil>, Columns = 1
rows:    9223372036854775808000000000000000001  ->  err=<nil>, Rows    = 1

TestParseProviderDescriptorGeometryOverflowVectors   PASS
TestParseProviderDescriptorGeometryBounds            PASS
TestParseProviderDescriptorValueRefusals             PASS
```

A 37-digit literal is accepted as a 1-cell terminal. This is exactly the
fabricated-in-range value the guard exists to prevent, and it is exactly what
rev1's F1 was filed against — the guard fixes the behaviour, and four rounds of
evidence never pinned where its edge is.

**Why the existing corpus misses it.** `TestParseProviderDescriptorGeometryOverflowVectors`
is the test whose doc comment says it proves "the 1..1000 bound is decided on the
digit string, never on a wrapped accumulator". Measured against the threshold
window:

| Mutant threshold | Wrap possible? | Killed by the shipped corpus? |
| --- | --- | --- |
| `1000`, `10000`, `1e9`, `92233720368547758` | no — equivalent | n/a |
| **`922337203685477580` (the true edge)** | **yes** | **NO — suite green** |
| `2^62` | yes | yes (`18446744073709551617` admitted) |
| `MaxInt` (guard never fires) | yes | yes |

Every shipped vector is 19, 20 or 101 digits and refuses under the edge mutant
too — verbatim the round-4 sentence *"every prior witness saturates under the
weakened guard too"*, one site over.

**Fix vector, verified one step away.** A witness must walk the accumulator to
exactly `floor(MaxInt/10)` and then re-enter `1..1000`. I checked
`9223372036854775808000000000000000001` across the whole threshold family: it is
admitted at `922337203685477580`, `2^62` and `MaxInt` (so one row reddens the
entire dangerous window) and correctly refused at `100`, `1000`, `10000`, `1e9`
and `92233720368547758` (so it does not redden the equivalents). Add it on
**both** `columns` and `rows` asserting `descriptor geometry bound`, and prove
it by applying the `> 922337203685477580` mutant, not only by deleting the guard.

---

## F2 (blocking) — the digit-range census does not derive and does not fail closed

`internal/terminalbackend/digit_range_census_test.go`

G-A asked whether this artifact derives its site set from production or lists
it. It **lists** it: the census is a 60-line file header. The file contains no
`go/ast` scan, no site enumeration, no fail-closed check — only one test
(`TestDescriptorGeometryAdjacentsNeverReachTheGate`) pinning one *claim* from
the prose. Nothing in the repository derives the digit-range comparison set.

**Control plant (the test G-A named).** I added a fourth digit-range comparison
to production, deliberately inert so that only a census could see it:

```go
// in ParseProviderDescriptor, internal/terminalbackend/descriptor.go
nonDigitInGeneration := false
for i := 0; i < len(generation); i++ {
    if generation[i] < '0' || generation[i] > '9' {
        nonDigitInGeneration = true
    }
}
_ = nonDigitInGeneration
```

```
go test ./internal/terminalbackend/ ./internal/provhost/
ok  .../internal/terminalbackend  1.147s
ok  .../internal/provhost        11.390s
```

The census did not redden. A listed set rots on the next comparison, and this
one already has.

**And the omission has a live consequence.** The census header scopes out the
accumulation bounds like this:

> *"the semverMajor/parseMajor saturation guards are accumulation bounds, pinned
> at their arithmetic edges by TestSemverMajorSaturationEdge (here) and
> TestParseMajorSaturationEdge (provhost)"*

It names **two** accumulation bounds. This leaf has **three** — the third is
`descriptorGeometry`'s `value > 100`, which is *not* pinned at its arithmetic
edge and is F1 above. A derived census over `<production digit-accumulation
bound>` would have listed it and forced the row. The prose one silently omitted
the only unpinned member of its own class. That is the mechanism by which this
class has now survived four rounds, and it is why F2 is not a style point.

The round-5 battery inherits the same denominator defect (G-D): "the three
digit-range comparisons plus both saturation guards" — five gates, nine mutants
— with the third accumulation bound absent. An unmeasured gate is not a passing
gate.

**Not in scope of this finding:** the *code* censuses in this delta are fine and
I am not asking for them again. `loadProviderCodes` is derived and fails closed
(var-binding, grouped-const, every-file and unclassifiable-binding plants all
present); both refusal-arm inventories are AST-derived bijections — PH-02/05/06
co-fired them, which is how I know they are live.

---

## Recommendation to the orchestrator — this should be a gate, not a fifth revision

Four rounds, one class, four different sites. Rounds 2–5 each closed the sites
the previous reviewer named and each shipped a hand-written artifact
(hand-table → hand-table → prose census) that could not name the next site. The
production code has been correct since round 2; only the instrument keeps
failing.

Asking for a sixth hand-written row will close F1 and will not close the class.
The thing that ends it is one derived, fail-closed instrument over both shapes
in one denominator:

1. AST-enumerate, per package, every `X < '0' || X > '9'` character comparison
   **and** every `value = value*10 + …` accumulation bound in non-test sources.
2. Require a declared row per site: the reachable rejected class (checked against
   what the entry point can actually spell) for the character gates, and the
   arithmetic-edge threshold plus its witness for the accumulation bounds.
3. Fail closed on a site with no row, a row with no site, and a site it cannot
   classify — each control-planted, including an import alias and a `var`
   binding.

That is a scoped piece of work and it is the only version of item 3 that
survives the next comparison anyone writes.

---

## Notes carried, unchanged

- **N4** — `foreignMajor` peek widening placement outside this delta: accepted
  in round 3, no production file changed in round 5, still holds.
- **N6** — the N2 CallExpr-only bound sits on
  `TestNoProductionPathAttestsProviderIdentityBinding`'s doc comment; untouched.
- **N7** — the §7.A deferral names no follow-up board item. Orchestrator's
  board-element freeze, not a producer omission; attach when the Story's final
  leaf closes.
- **F3 correction stands.** The withdrawn justification in `provhost/identity.go`
  is now measured, not asserted: `TestSpecIdentityExampleVerifiesAgainstItsClaimedDigest`
  shows the pinned Section 5.5 example *does* verify against its own record_id,
  so it never justified skipping attestation. Good correction; keep it.

## Provenance

| Item | Value |
| --- | --- |
| Base OID | `1d9747433ac1d9fcdd74331c8cbf1db13f9ac2af` (= worktree HEAD, 0 commits behind `main`) |
| Candidate tree recomputed | `d6253f58fe72a84c0900329c686300f232374874` — matches the CR record |
| New files present in that tree | `provider/lift.go`, `provider/lift_test.go`, `terminalbackend/descriptor.go`, `descriptor_test.go`, `digit_range_census_test.go`, `semver_major_edge_test.go` |
| Worktree after the battery | byte-identical; tree re-derived to `d6253f58…` |
| Battery evidence | `TASK-260906-vmzk0y_review-rev5-battery.json`, harness `TASK-260906-vmzk0y_review-rev5-battery.py` |
