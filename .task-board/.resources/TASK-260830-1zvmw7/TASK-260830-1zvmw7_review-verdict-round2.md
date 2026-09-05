# TASK-260830-1zvmw7 — Review Verdict (round 2, CR rev 2)

**Verdict: CHANGES REQUESTED.** Route to `to-dev`. One blocking finding (H1),
two non-blocking (H2, H3).

Scope reviewed: the uncommitted working tree at base `9481a20`. I recomputed the
working-tree tree OID with a scratch index before and after every probe: both
times `d99bf886589f38b06af25e8a381285d151a1817b`, byte-identical to the CR
candidate. `manifest.go` sha256 re-verified as
`97edf447a3c5a800321dd826bb55e9cc1a668af25472c9981ad6840199ea688b` after each of
the four batteries; all ten package/traceability source files re-checked against
a pre-review `shasum` baseline at the end.

Baseline re-run by me, not accepted from the run: `go build ./...`,
`go vet ./...`, `gofmt -l .` exit 0; `go test ./... -count=1` green across 14
packages; package coverage 88.6%; `tracecheck -root .` exit 0 reporting
`acceptance_cases=75`.

---

## Headline ratios

| Measure | Round 1 | Round 2 |
| --- | ---: | ---: |
| Refusal assertions that check which arm fired | 0 / 29 | **every executed arm identity, 76 / 76** |
| Executed arm **sites** whose identity is asserted | — | **85 / 85** |
| Arm sites executed by the suite | 71 / 121 | **85 / 119** |
| Mutants killed | 64 / 96 | **91 / 95** |
| Resurrections among round-1 kills | — | **0** |
| G-A gate deletions that fail by naming the right arm | 0 / 6 | **17 / 17** |
| G-C aliasing positive control | survives | **killed** |

---

## G-A — the arm-shift premise, tested rather than accepted

The rework's premise was that asserting `*Error.Detail` closes F2/F3/F5
structurally. I re-ran the battery over the four §4.B gates, F3's
untrusted-substitution gate at **both** its sites, F4's expiry boundary, and the
F5 closed-schema class on `ParseProbe`, `ParseEvidence` and `ParseManifest` —
17 gates, each grep-anchored (count == 1), `go vet`-checked, run, reverted.

**All 17 are killed, and every one of them fails with `error = <nil>`.** Not one
fixture slides onto a lower arm. That is the strongest outcome available: with
the gate removed the tuple reaches admission, so the test fails because the gate
admitted what it must reject, not because the refusal landed somewhere else.

| Gate deleted | Test that fails | Failure |
| --- | --- | --- |
| `checkClaimRelation` omission loop | `TestReconcileRefusals/omitted_manifest_claim` | `error = <nil>` want `probe omission of manifest claim` |
| probe static claim echo equality | `.../static_echo_drift` | `error = <nil>` want `probe static claim echo` |
| `checkProbeMembership` protocol half | `.../protocol_not_member` | `error = <nil>` want `probe protocol membership` |
| `checkEvidenceSet` `!claim.Value` half | `.../evidence_for_false_claim` | `error = <nil>` want `evidence claim binding` |
| `checkManifestRecordBinding` digest | `TestAdmitProbeExternalRealm` | `error = <nil>` want `executable substitution` |
| `checkProbeIdentity` digest | `TestReconcileRefusesExecutableSubstitution` | `error = <nil>` want `executable substitution` |
| liveness upper bound `!Before`→`After` | `.../evidence_expires_at_admission_instant` | `error = <nil>` want `evidence liveness` |
| `ParseProbe` schema / version / `ParseID` / kind / digest↔kind (5) | `TestProbeDocumentRefusals/*` | `error = <nil>` at the named arm |
| `ParseEvidence` schema / version / `ParseID` / protocol major (4) | `TestEvidenceDocumentRefusals/*` | `error = <nil>` at the named arm |
| `ParseManifest` kind | `TestManifestDocumentRefusals/unknown_implementation_kind` | `error = <nil>` want `manifest implementation kind` |

F3 is worth calling out: the fixture is driven through `Registry.AdmitProbe`,
the real production entry point, with the probe deliberately *tracking* the
foreign digest so probe↔manifest identity holds and only the manifest↔record
gate can refuse. That is the correct way to isolate two arms that share a code
and a detail string, and the test comment says so.

### The ratio that matters

Method: enumerate every `mismatchf(...)`/`integrityFailure(...)`/`Detail: "..."`
site in `manifest.go` by walking the file; map each to its narrowest containing
block in a `-covermode=count` profile of the shipped suite; extract the asserted
detail from every `requireRefusal`/`requirePinRefusal` call and table `detail:`
field.

* **119 arm sites**, 85 executed, 34 never executed.
* **85 / 85 executed sites have their arm identity asserted** somewhere in the
  suite. 76 / 76 executed distinct arm identities are named.
* Round 1's figure was 0 / 29. The structural change is real.

Stated bound on that ratio: 19 of the 85 executed sites share a detail string
with another site (`document members`, `document member type`, `document
syntax`, `document digest`, `document timestamp`, `document string bound`,
`capability vocabulary`, `operation vocabulary`, `evidence realm binding`,
`executable substitution`). For those the assertion names the **rule**, not the
**site**. 66 executed sites carry a file-unique name and are pinned exactly.

---

## H1 (blocking) — a fifth `checkClaimRelation` arm still deletes cleanly, and it admits an undeclared capability

`checkClaimRelation` has five arms. The rework gave four of them narrowed
fixtures. The first one has none:

```go
case proved.Origin == OriginStatic && !exists:
    return mismatchf("probe static claim without manifest")
```

Neuter it (`_ = static`) and **the whole shipped suite stays green**. The arm is
never executed by any test — coverage hit count 0.

This is not a cosmetic gap. I built the bypass through the real entry points
(`ParseManifest` → `ParseProbe` → `Reconcile`, pin fixtures, real signatures): a
probe that declares `graceful_stop` with `origin: static` while the manifest
declares only `headless_creation`.

| Code | `Reconcile` result |
| --- | --- |
| HEAD | `terminal_backend_manifest_probe_mismatch at probe static claim without manifest`, `admitted=[]` |
| gate neutered | `err=<nil>`, **`admitted=[graceful_stop headless_creation]`** |

A backend self-asserts static origin for a capability its Manifest never
declared, and the capability is admitted. The manifest→probe omission loop
cannot catch it — it checks the other direction. This is the self-minted-claim
shape, on the exact object this leaf exists to gate.

The round-2 premise was that arm assertions close the class "structurally rather
than case by case". Four of five arms in one `switch` got cases; the fifth did
not. Round 1 already scored this function as a `sliver` (area 11, 1/3). The
class is not closed.

**Required:** one `TestReconcileRefusals` row whose probe carries a static-origin
claim absent from the manifest, asserting `probe static claim without manifest`.

## H2 (non-blocking) — `ParseEvidence`'s platform vocabulary is the F5 shape, unclosed

`platform, err := scalar.ParsePlatform(platformName)` → `evidence platform`.
Narrow to `platform, _ :=` and the suite stays green; the arm has 0 coverage
hits. `ParseProbe` has `probe platform`, `ParseManifest` has `platforms
vocabulary`, both tested. This is precisely the "clause present in one context
profile and absent from the other" shape F5 named, in the one parser the rework
did not extend for platform. A bad platform would probably still be refused
downstream by `evidence tuple binding` or the signature, so the impact is
defence-in-depth — but the gate's own class is unmeasured.

## H3 (non-blocking) — `claim shape` unmeasured

`parseClaim`'s `raw.(map[string]any)` guard narrows to `object, _ :=` with a
green suite; 0 coverage hits. A non-object claim element then falls to
`checkExactMembers` and refuses as `document members`, so it stays fail-closed —
but the refusal is arm-blind, which is the property this round was about.

---

## G-B — resurrections: none

Both round-1 batteries re-run in full, each mutant grep-anchored (count == 1),
`go vet`-checked, run, reverted, with the file hash re-verified at the end.

| Battery | Killed | Survived | Other |
| --- | ---: | ---: | --- |
| Battery 1 (60) | 46 | **0** | 12 COMPILE-FAIL (formulation artifacts: deleting the code leaves an unused binding), 2 ANCHOR-BAD |
| Battery 2 (47) | 43 | 4 | 0 |
| Re-anchored M37, M58 | 2 | 0 | anchors moved by the F1/F8 edits |
| **Unique measured 95** | **91** | **4** | |

The 12 battery-1 compile-fails are exactly the 12 whose compilable variants
battery 2 carries, and all 12 are killed there. **Every mutant round 1 recorded
as KILLED is still KILLED.** Nothing was loosened to close a finding.

The 4 survivors are the four round 1 declared, unchanged:

* `M71`/`M72` — JCS replaced by raw `json.Marshal`. **The rework did not change
  this and I am not claiming it did.** The three closed schemas still carry only
  ASCII keys and no numeric members, so Go's marshalling and RFC 8785 agree byte
  for byte and no fixture can distinguish them. Reported as **unknown**, not as
  covered.
* `M92` (nil-registry guard in `AdmitProbe`) and `M95` (absent-vs-null in
  `digestOrNullMember`) — near-equivalent; `Resolve` has its own nil guard and
  `checkExactMembers` already guarantees presence. Counted as survivors for
  honesty, per the round-1 convention.

Round 1's protected areas all hold: `UnsignedEvidenceBytes` narrowing 20/20 this
round (M16-M22, M61-M70, M15, M75 — every member drop, the domain separator and
its NUL), `GenerationDigest` 4/4 including domain truncation, signature 2/2
including forged key and nil verifier, tuple/coverage/ID-set 12/12.

### Round-1 findings

| Finding | State | Evidence |
| --- | --- | --- |
| F1 `CheckOperation` refused `manifest`/`probe` | **closed** | fix pinned; widening the bypass to `create` is killed (`error = <nil>` at `operation capability dependency`); deleting the vocabulary gate that precedes it is killed by name |
| F2 four over-determined fixtures | **closed** | 4/4 delete to `error = <nil>` |
| F3 executable substitution | **closed** | both sites, one through `Registry.AdmitProbe` |
| F4 expiry upper boundary | **closed** | `expires_at == now` row; `!Before`→`After` killed |
| F5 Probe/Evidence closed schemas | **closed** | 10/10 |
| F6 six unevidenced classes | **closed** | M38/M39/M41/M60/M79/M81 all killed |
| F7 four unmeasured bounds | **closed** | M45/M89/M90/M91 all killed |
| F8 dead arms and inaccurate doc | **closed** | both arms gone, doc corrected |
| F9 README + traceability | **closed** — see below | |
| G-A-1 claim-slice aliasing | **closed** | see G-C |

---

## G-C — the aliasing pin

I re-ran round 1's exact positive control: `parseClaim` returning
`row.dependentOperations` / `row.evidenceRequirements` instead of the parsed
slices. Round 1: all 52 tests passed. Round 2:

```
manifest_pin_test.go:166: capabilityRegistry["headless_creation"] =
  {generationVariable:true dependentOperations:[REVIEW-POISON]
   evidenceRequirements:[REVIEW-POISON REVIEW-POISON]} after claim mutation,
  want {... dependentOperations:[create] evidenceRequirements:[conformance_fixture runtime_probe]}
```

`TestParsedClaimSlicesShareNoRegistryBacking` catches it, and five other tests
fall over behind it. The pin re-reads the interior (row snapshot,
`CapabilitiesForOperation`, a fresh admission) rather than only re-comparing
`Admitted`, which is what round 1 asked for. **The claim is pinned.**

Stated bound: the pin poisons only the two capabilities the fixture claims, so it
directly holds 2 of the 16 registry rows. The aliasing it guards is structural in
`parseClaim`'s single return statement, so one path covers all rows — a bound on
the fixture, not a gap in the guard.

---

## F9 — verified by attacking the gate, not by reading the diff

`terminal-manifest-probe-admission` is registered against production
`internal/terminalbackend/manifest.go` `AdmitProbe` with 11 tests; the projection
digest is re-pinned to `892bdb6a…bf806a035`; README and the golden figure move
74→75. The gate has teeth — all five attacks killed:

| Attack | Result |
| --- | --- |
| acceptance case names a test that does not exist | KILLED — `declaration "TestNoSuchReviewTest" is absent from …` |
| acceptance case names a production declaration that does not exist | KILLED — same shape, caught by the README figures pin too |
| `gap` prose edited without re-pinning the digest | KILLED — projection digest differs from reviewed |
| the whole new acceptance case removed | KILLED — projection digest differs |
| reviewed digest reverted to leaf 1's value | KILLED |

The README `15.3#3` sentence is now true: the clause forbids Structured Error
from appearing as an RPC hello key or a TerminalBackend capability, and the
re-scoped text claims only that the admission registry is not on a hello path
and carries no Structured Error code, with the clause left **undischarged**.
I checked the substantive half independently — the 16 capability names are
disjoint from `internal/axerror`'s code registry — so the prose is accurate.

Two observations, neither blocking:

* That disjointness is asserted in prose in the ownership `gap` field and held by
  no test. It supports a *non*-discharge, so it is not an unsupported discharge
  claim, but it is checkable and unchecked.
* The `section:4.B`, `4.C`, `4.D`, `6.5` and `7.A` bindings still name
  `internal/catalog/catalog.go` `ForRelease` with `coverage: unevidenced` and
  `clauses: 0`, and `unevidenced=41` / `clauses_discharged=17/403` are unchanged
  from leaf 1. Round 1 cited those numbers as context for F9; its stated
  requirement was to re-scope the README sentence and register this leaf's
  evidence "as leaf 1 did", and both are done. Leaving the section binding
  unevidenced under-claims rather than over-claims, which is the correct
  direction — self-minting a coverage upgrade would be the worse error. Recorded
  so the next leaf can decide whether §4.B's owner should move.

---

## What is good, stated so the next rework does not undo it

* The four narrowed F2 fixtures are the right kind of narrow: each was tightened
  until only the named rule can reject it, and the deletion probe proves it by
  reaching admission rather than by landing elsewhere. Do not re-widen them.
* The F3 fixture's probe deliberately tracks the foreign digest. That is the only
  way to separate two arms sharing both code and detail. The comment above it
  explains why — keep it.
* `requireRefusal`/`requirePinRefusal` taking `(code, detail)` is the structural
  fix. Every new refusal case must go through them.
* `TestParsedClaimSlicesShareNoRegistryBacking` is a real interior pin, not a
  restatement of `TestAdmittedSharesNoBackingArrays`. Keep both.
* The logbook entry names the mis-targeted expiry test, the deterministic-RSA
  dedup surprise and the nil-vs-empty production bug. That honesty is the reason
  this round had a real baseline to measure against.

## Stated bounds — what this method cannot see

* **JCS remains unprovable** with these fixtures (`M71`/`M72`). Unchanged by the
  rework, and unfixable without a fixture carrying a non-ASCII key or a number,
  which the closed schemas forbid. Reported as unknown.
* `M92`/`M95` are near-equivalent mutants counted as survivors for honesty.
* 34 of 119 arm sites are never executed. Beyond H1-H3 the remainder are
  `document identity` (2), `evidence canonical bytes` (2) — `json.Marshal`/
  `jcs.Transform` failures on already-validated maps, practically unreachable —
  `registry unavailable` (= `M92`), and repeated `document members`/`document
  member type`/`document syntax`/`document timestamp` sites whose rule is
  asserted at another site.
* Arm→coverage mapping uses the narrowest containing block, so the 85/119
  executed figure is an upper bound on sites reached.
* Mutation bounds the suite, not the code. A killed mutant means the arm is
  exercised, not that it matches the spec. Round 1's F1 was the standing proof.
* Nothing measured under `-race`; concurrency is out of scope for this leaf.

---

## Required for acceptance

1. **H1** — add the `TestReconcileRefusals` row for a static-origin probe claim
   absent from the manifest, asserting `probe static claim without manifest`.
   Verify by neutering that `case` arm and confirming the new row fails.
2. **H2** — add an `evidence platform` case to `TestEvidenceDocumentRefusals`.
3. **H3** — add a non-object claim element case asserting `claim shape`.

All three are single table rows. Nothing else in this change needs to move — do
not re-run the fixtures through a wider refactor. Re-run both batteries plus the
17-gate arm-shift probe afterwards; the acceptance target is 91/95 held with the
same four declared survivors, and 17/17 still failing with `error = <nil>`.
