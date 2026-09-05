# TASK-260830-1zvmw7 — Review Verdict (round 3, CR rev 3)

**Verdict: ACCEPTED.** `accept_cr(TASK-260830-1zvmw7, revision=3)`. Zero blocking
findings. One non-blocking finding (G-A) recorded with a concrete follow-up, and
one accuracy note on the producer's own enumeration report.

## Scope and identity

Reviewed the uncommitted working tree at base `9481a20`. Recomputed the
working-tree tree OID with a scratch index before and after every probe: both
times `9e3fb5998ea812f3ccada9d01d12f877b5369083`, byte-identical to the CR
candidate. All twelve package/traceability source files re-checked against a
pre-review `shasum -a 256` baseline at the end — all `OK`.

`manifest.go` sha256 is `97edf447a3c5a800321dd826bb55e9cc1a668af25472c9981ad6840199ea688b`,
**byte-identical to round 2**. The round-2→round-3 delta is exactly:

```
 LOGBOOK.md                                |  5 +++++
 internal/terminalbackend/manifest_test.go | 35 +++++++++++++++++++++++++++++++
```

Three table rows and a logbook entry. No production change, no test-helper
change, no fixture-shape change. Everything round 2 attacked in
`terminalbackend.go`, `traceability.go`, `ownership.v0.5.0.json` and `README.md`
is bit-for-bit the artifact round 2 verified; I re-state that as byte-identity
evidence rather than re-attacking it, and note it explicitly as evidence carried
forward, not re-measured.

Baseline re-run by me in standalone processes, not accepted from the run:
`go build ./...`, `go vet ./...`, `gofmt -l .` exit 0; `go test ./... -count=1`
green across 14 packages; package coverage 88.9%; `tracecheck -root .` exit 0
reporting `acceptance_cases=75`, `clauses_discharged=17/403`.

---

## Headline ratios

| Measure | Round 1 | Round 2 | Round 3 |
| --- | ---: | ---: | ---: |
| Mutants killed / measured | 64 / 96 | 91 / 95 | **91 / 95** |
| Resurrections among prior kills | — | 0 | **0** |
| Arm sites executed by the suite | 71 / 121 | 85 / 119 | **88 / 119** |
| Executed arm sites whose identity is asserted | — | 85 / 85 | **88 / 88** |
| Distinct executed arm identities asserted | 0 / 29 | 76 / 76 | **79 / 79** |
| Executed sites carrying a file-unique detail | — | 66 / 85 | **69 / 88** |
| Executed sites sharing a detail (rule-named, not site-named) | — | 19 | **19** |
| Gate deletions failing by naming the right arm | 0 / 6 | 17 / 17 | **20 / 20** |
| Refusal arms with zero assertion anywhere | — | 6 details / 8 sites | **3 details / 5 sites** |

Method for the arm ratios: enumerate every `mismatchf("…")`,
`integrityFailure("…")` and literal `Detail: "…"` site in `manifest.go` (119
sites, 82 distinct details — the two helper definitions take variables and are
excluded); map each site to its narrowest containing block in a `-covermode=count`
profile of the shipped suite; count the quoted detail literal across all four
package test files. Script attached as `…_review-round3-armratio.py`.

---

## G-A — the mechanism question, answered by attacking it

**Did the three rows land?** Yes, and they are live, not cosmetic. Each is
killed by neutering exactly the arm it names, and each mutant makes exactly that
subtest fail:

| Mutant | Result | Failure |
| --- | --- | --- |
| `H1` `case proved.Origin == OriginStatic && !exists:` → `_ = static` | KILLED | `TestReconcileRefusals/static_claim_without_manifest`: **`error = <nil>`**, want `probe static claim without manifest` |
| `H2` `platform, err := scalar.ParsePlatform(…)` → `platform, _ :=` | KILLED | `TestEvidenceDocumentRefusals/bad_platform`: **`error = <nil>`**, want `evidence platform` |
| `H3` `object, ok := raw.(map[string]any)` → `object, _ :=` | KILLED | `TestManifestDocumentRefusals/non-object_claim`: `refusal = … at "document members"`, want `claim shape` |

H1 and H2 fail at `error = <nil>` — the strongest outcome: with the arm removed
the tuple reaches admission, so nothing else in the pipeline over-determines the
fixture. H3 fails arm-blind at `document members`, exactly as the producer
predicted and disclosed; fail-closed but now arm-pinned, so narrowing is caught.

H1 is checked as a real bypass, not a table entry: the fixture flips
`capability_claims[1]` (`graceful_stop`, `probed`, `true`) to `origin: static`
while the manifest declares only `durable_disconnect` / `headless_creation` /
`local_attach`. Value stays `true` with evidence intact and `probe_id` re-stamped,
so only the fifth arm can reject. The row drives `reconcileFixture`
(`ParseManifest`/`ParseProbe`/`ParseEvidence`) into the real
`terminalbackend.Reconcile`. That is the self-minted-claim shape, on the
production entry point.

**Is the enumeration a mechanism, or three named rows?** It is manual. I ran the
positive control the question deserves rather than reading the diff: I inserted a
brand-new production refusal arm carrying a brand-new detail string that no test
asserts —

```go
// in parsePlatformList, after the existing bound (which already caps at 4)
if len(names) > 8 {
    return nil, mismatchf("platforms review probe bound")
}
```

— and ran the whole tree:

| Gate | Result with the unasserted arm present |
| --- | --- |
| `gofmt -l .` | exit 0 |
| `go vet ./...` | exit 0 |
| `go test ./... -count=1` (14 packages) | **exit 0 — entire suite green** |
| `tracecheck -root .` | **exit 0** |

**Finding G-A (non-blocking): nothing in the tree enumerates refusal arms from
production and fails when one is unasserted.** A future arm can ship unwitnessed
and every gate stays green. This repo already has the mechanism in a sibling
package — `internal/canonicaljson/grammar_inventory_test.go` and
`constraint_inventory_test.go` derive their obligation from the production AST
and fail at derivation when a construct has no witness — and the shared logbook
already records the general property from TASK-260830-1tax26: *"an inventory must
assert that the set of constructs it scans is complete for the package."*
`internal/terminalbackend` does not adopt it.

**Why this does not block acceptance:**

1. **There is no sixth row to add.** I ran the enumeration independently, with
   my own script over the shipped tree, and it reproduces the producer's result:
   after these three rows, **zero** file-unique arms are unasserted. The entire
   residue is the set round 2 already declared as stated bounds —
   `document identity` (2 sites, `json.Marshal`/`jcs.Transform` failure on an
   already-validated map, practically unreachable), `evidence canonical bytes`
   (2 sites, same class), `registry unavailable` (1 site, = `M92`
   near-equivalent). The specific defect the round-3 brief warned about — a
   fourth round adding a sixth named row — cannot recur on this leaf.
2. **The producer found the arms without me.** They did not take the round-2
   list on trust; they derived it by the same extraction-and-count method I used
   and reached substantively the same answer, and documented the method. The
   *capability* is demonstrated; what is missing is the standing *gate*.
3. **The gate is new test infrastructure, and round 2 foreclosed it.** An
   AST-derived arm inventory is the size of `grammar_inventory_test.go`, which in
   this repo was its own task. Round 2's required list was exactly three table
   rows and said "nothing else in this change needs to move — do not re-run the
   fixtures through a wider refactor". Reopening now for work the previous
   verdict forbade would be the process failure, not the fix.

**Recommendation (for the orchestrator, not a condition of this acceptance):**
open a follow-up leaf for a `refusal_arm_inventory_test.go` in
`internal/terminalbackend` modelled on `internal/canonicaljson/grammar_inventory_test.go`:
walk the package production AST for `mismatchf`/`integrityFailure`/`Detail:`
literals, index the asserted details across the package test files, and fail
with the arm's file:line when a detail has no witness — with the three
practically-unreachable details carried as an explicitly enumerated, justified
allowlist so that shrinking the allowlist is also a reviewable event. Until that
exists, "the arm set is closed" is a measurement, not an invariant.

### Accuracy note on the producer's enumeration report

`TASK-260830-1zvmw7_round3-three-rows.md` states "Exactly five details had zero
hits: the three above plus `document identity` (x2) and `registry unavailable`".
The measured zero-hit set was **six** details across eight sites — the note omits
`evidence canonical bytes` (2 sites). The omitted item was already inside round
2's stated bounds and required no action, so nothing follows from it for this
leaf. It is recorded because it is the concrete cost of G-A: the manual
enumeration's own report drifted by one item on its first outing, which is the
exact failure a derived gate makes impossible.

---

## G-B — the full battery, re-run: no resurrections

Both round-1 batteries plus the arm-shift battery and the supplementary battery
re-run in full, each mutant grep-anchored (`count == 1`), `go vet`-checked, run,
reverted, with `manifest.go`'s hash re-verified against the original after every
battery.

| Battery | Killed | Survived | Other |
| --- | ---: | ---: | --- |
| Battery 1 (60) | 46 | **0** | 12 COMPILE-FAIL, 2 ANCHOR-BAD |
| Battery 2 (47) | 43 | 4 | 0 |
| Re-anchored `M37` (= `S2`), `M58` (= `F3b`) | 2 | 0 | |
| **Unique measured 95** | **91** | **4** | |
| Arm-shift 17 gates + `H1`/`H2`/`H3` | **20 / 20** | 0 | all 17 gate deletions at `error = <nil>` |
| Supplementary (`S1`–`S3`, `S5r`) | 4 | 0 | `S4` COMPILE-FAIL (unreachable-code artifact) |

**Zero resurrections.** Every mutant rounds 1 and 2 recorded as KILLED is still
KILLED, and the 12 battery-1 compile-fails are the same 12 whose compilable
variants battery 2 carries and kills. Round 2's protected areas hold unchanged:
`UnsignedEvidenceBytes` 20/20 member-drop and domain-separator narrowings,
`GenerationDigest` 4/4, signature 2/2 (forged key, nil verifier),
tuple/coverage/ID-set 12/12. `G-C`'s aliasing positive control (`S1`) is still
killed by `TestParsedClaimSlicesShareNoRegistryBacking` with five other tests
behind it.

Two supplementary notes:

* `S5` (F1 divergence) reported `ANCHOR 0` — its round-2 formulation targeted an
  early return that this design does not have. I re-anchored it rather than
  dropping it: adding `"manifest"` to a registry row's `dependentOperations`
  creates the real divergence (`CheckOperation` hardcodes the manifest/probe
  bypass while `CapabilitiesForOperation` derives from the registry). **KILLED** —
  `TestCapabilitiesForOperation` fails with `CapabilitiesForOperation("manifest") = [remote_attach], want []`.
  The divergence class is pinned by the full expected map over the operation
  vocabulary, not by the removed early return.
* `H2`'s kill is `ParseEvidence`-local. In the reconcile path a bad platform is
  still refused downstream by `checkEvidenceTuple`'s `object.Platform != probe.Platform`,
  so `evidence platform` is defence-in-depth, as round 2 predicted. The parser's
  own class is now closed; the production consequence was never open.

### The 19 detail-string collisions

**All 19 remain, unchanged.** The three new rows all pin file-unique details, so
they move the file-unique figure (66/85 → 69/88) without touching the collision
set. For these 19 executed sites the assertion names the **rule**, not the
**site**: `capability vocabulary` (2), `document digest` (3), `document member type` (2),
`document members` (1), `document string bound` (1), `document syntax` (2),
`document timestamp` (1), `evidence realm binding` (3), `executable substitution` (2),
`operation vocabulary` (2). Unchanged stated bound, not a regression.

---

## Bounds, kept honest

* **`M71`/`M72` (JCS) remain unprovable and are reported as unknown, not
  covered.** The rework did **not** change this, and I am not claiming it did:
  `manifest.go` is byte-identical to round 2, so by construction the JCS path is
  the same code. The three closed schemas still carry only ASCII keys and no
  numeric members, so `json.Marshal` and RFC 8785 agree byte for byte and no
  fixture these schemas admit can distinguish them.
* **`M92` (nil-registry guard in `AdmitProbe`) and `M95` (absent-vs-null in
  `digestOrNullMember`) remain counted as survivors**, unchanged by the rework,
  near-equivalent by the round-1 convention.
* 31 of 119 arm sites are never executed. 26 of them share a detail asserted at
  another site; the remaining 5 are the three declared-unreachable/near-equivalent
  details above.
* Arm→coverage mapping uses the narrowest containing block, so 88/119 executed is
  an **upper** bound on sites reached.
* Mutation bounds the suite, not the code: a killed mutant means the arm is
  exercised, not that it matches the spec.
* Nothing measured under `-race`; concurrency is out of scope for this leaf.
* The traceability, README and `terminalbackend.go` surfaces were **not**
  re-attacked this round. Their evidence is round 2's, carried forward on
  byte-identity of the tree diff, not re-measured.

---

## Why this is accepted

The leaf has had three rounds and the numbers moved in one direction throughout:
64/96 → 91/95 killed with zero resurrections; 0/29 → 79/79 executed arm
identities asserted; 0/6 → 20/20 gate deletions failing by naming the right arm,
17 of them at `error = <nil>`. The one blocking finding from round 2 — a fifth
`checkClaimRelation` arm that deleted cleanly and admitted a self-minted
`graceful_stop` claim — is closed with a minimal, non-over-determined fixture
driving the real `Reconcile`, and its two non-blocking siblings with it. The
enumeration that found them is manual, and that is worth a follow-up leaf, but
the arm set it enumerates is closed today and I verified that independently
rather than accepting the report.

Handoff: this is a reviewer-archetype acceptance. No `commit_ack` supplied. The
orchestrator checkpoints or integrates the accepted revision and makes the `done`
transition with `commit_ack=scope_committed`.
