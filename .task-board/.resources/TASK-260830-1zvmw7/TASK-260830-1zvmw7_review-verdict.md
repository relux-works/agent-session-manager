# TASK-260830-1zvmw7 — Review Verdict (round 1, CR rev 1)

**Verdict: CHANGES REQUESTED.** Route to `to-dev`.

Scope reviewed: `git diff 9481a20..f81e2ea`, 6 paths, +3896/-2. Working tree at
review time byte-identical to the CR candidate before and after every probe
(`shasum -a 256` over all six package `.go` files, diffed against a pre-review
baseline). Every mutant below was grep-confirmed present, compiled, run, and
reverted; `manifest.go` sha256 re-checked as
`d5a7b8cccf9670f98eb6e85cdbe76226fc8a293685177f685e0111fa74c7ab85` after each
battery.

Baseline accepted from the run, not re-derived: `go build ./...`, `go vet ./...`,
`gofmt -l .` all exit 0; `go test ./...` green across 14 packages; package
coverage 87.1%.

---

## Headline ratios

| Measure | Ratio |
| --- | ---: |
| Mutants killed by the shipped suite | **64 / 96** |
| Production refusal arms executed by the shipped suite | **71 / 121** |
| Leaf-2 refusal assertions that check which arm fired | **0 / 29** |
| G-A: exported paths probed for interior aliasing | 12 / 12 entry points, **1 injected aliasing bug survives** |

---

## G-A — leaf 1's pinned claim over the widened surface

Method (same as leaf 1: enumerate, then probe — not read). `go doc -all` gives
leaf 2 a new exported surface of 9 funcs, 5 types, 26 constants. I wrote a
white-box probe that:

1. snapshots the package interior — both built-in `Registration` records from a
   fresh `New("2.1.0", …)` and every package-level closed table
   (`capabilityRegistry`, `operationVocabulary`, `evidenceRequirementVocabulary`,
   `evidenceFactVocabulary`, `requirementFact`, the three `*Members` lists);
2. calls **every** leaf-2 exported entry point that returns composite data —
   `CapabilitiesForOperation` over all 10 operations, `ParseManifest`,
   `ParseProbe`, `ParseEvidence`, `Reconcile`, `Admitted.Has`,
   `Admitted.HasOperation`, `CheckOperation`, `UnsignedEvidenceBytes`,
   `Registry.AdmitProbe`, `Registry.Resolve`;
3. reflectively scribbles `REVIEW-POISON` into every reachable slice element,
   map value, and settable string/bool/byte field of every returned value;
4. re-snapshots and re-admits.

**At HEAD the probe passes: no exported path reaches the built-ins or the closed
tables.** The claim holds for the production code as written.

**Finding G-A-1 (blocking, evidence).** The probe is not vacuous — but the
shipped suite is. Positive control: change `parseClaim`'s return from the parsed
slices to the registry row's slices

```go
DependentOperations:  row.dependentOperations,
EvidenceRequirements: row.evidenceRequirements,
```

My probe kills it and shows `REVIEW-POISON` landing inside
`capabilityRegistry["graceful_stop"]` and `["headless_creation"]`. **The shipped
suite (52 test functions) passes with that mutant live.**
`TestAdmittedSharesNoBackingArrays` mutates `probe.EvidenceIDs[0]`,
`manifest.ProtocolVersions[0]` and `evidence[0].Facts[0]` — three fields, none of
which can alias a package table — and then only asserts that `admitted` did not
change; it never re-reads the interior and never re-admits.
`Claim.DependentOperations` and `Claim.EvidenceRequirements` are the two exported
slice fields on the whole new surface that *could* alias the closed registry, and
neither is touched. Leaf 1's claim is therefore **re-established for 3 of the
reachable composite fields and unestablished for the 2 that matter**.

Failure scenario if it ever regresses: any caller doing
`claim.DependentOperations[0] = "attach"` permanently rewrites the process-wide
capability registry, changing which operations every later admission confers and
which evidence proves them — and nothing in the suite notices.

---

## G-B — the traversal

Rows discovered by walking `manifest.go` top to bottom, not by following the test
file. 121 refusal arms; 96 mutants across every one of them; each row gets its
mutant, its kill/survive, and a bucket.

| # | Gate area (production site) | Mutants | Killed | Bucket |
| --- | --- | ---: | ---: | --- |
| 1 | Manifest closed schema — `parseManifestObject` | 7 | 4 | `partial` |
| 2 | **Probe closed schema — `ParseProbe`** | 7 | **1** | **`sliver`** |
| 3 | **Evidence closed schema — `ParseEvidence`** | 11 | 5 | **`partial`** |
| 4 | Shared decode/member helpers | 10 | 5 | `partial` |
| 5 | Identity recompute / JCS — `objectIdentity`, `checkIdentity` | 3 | 2 | `partial` |
| 6 | Claim shape / registry-row binding — `parseClaim` | 3 | 2 | `partial` |
| 7 | Generation digest — `GenerationDigest`, `checkProbeGeneration` | 4 | 4 | `full` |
| 8 | Signed bytes — `UnsignedEvidenceBytes` | 20 | 19 | `full` (one stated bound) |
| 9 | Signature verification — `checkEvidenceSignature`, nil verifier | 2 | 2 | `full` |
| 10 | **Liveness / expiry — `checkEvidenceLiveness`** | 3 | 2 | **`partial`** |
| 11 | **§4.B keyed claim relation — `checkClaimRelation`** | 3 | **1** | **`sliver`** |
| 12 | Probe↔Manifest identity & membership | 4 | 3 | `partial` |
| 13 | Evidence set / tuple / coverage / ID set | 12 | 11 | `partial` |
| 14 | Operation gate — `CheckOperation`, `HasOperation` | 2 | 2 | `full` (but see F1 — the gate itself is wrong) |
| 15 | **Registry-bound `AdmitProbe`** | 3 | **1** | **`sliver`** |
| 16 | **Aliasing across the exported boundary (G-A)** | 1 | **0** | **`unevidenced`** |
| | **Total** | **96** | **64** | |

Cross-check from the other direction: a coverage profile over the shipped suite
shows **50 of the 121 refusal arms are never executed at all**, and the
never-executed set matches the survivor set area for area — the `sliver` rows
above are not an artifact of mutant selection.

### F1 (blocking, correctness — live spec violation)

`CheckOperation("manifest", …)` and `CheckOperation("probe", …)` refuse
unconditionally with `terminal_backend_capability_unproven`, **even when all 16
registry capabilities are admitted** (verified by direct probe against a
fully-populated `Admitted`).

Two independent authorities say that is wrong:

* the pinned spec's §4.D operation table gives both operations
  `Capability dependencies: none`, and neither row's allowed-error list contains
  `terminal_backend_capability_unproven`
  (`manifest`: `protocol_incompatible, protocol_error, process_failed, timeout,
  integrity_failure`; `probe`: `untrusted, manifest_probe_mismatch,
  implementation_drift, protocol_error, process_failed, timeout,
  integrity_failure`);
* this file's own sibling function documents the opposite model —
  `CapabilitiesForOperation`: *"manifest and probe confer through no capability
  and return an empty set: they carry no capability dependency."*

Failure scenario: the lifecycle owner gates an adapter `manifest` call through
`CheckOperation` — the operation this package admits into its own vocabulary —
and the Manifest read can never proceed, refused with an error code the spec does
not permit for that operation. No test covers either operation through
`CheckOperation`; `TestCapabilitiesForOperation` checks both through the *other*
function, where they behave as the spec says.

### F2 (blocking, evidence) — four §4.B/§4.D rules have a named test that lands on the right arm but does not hold the gate

At HEAD I confirmed each case reaches its intended clause. Under a mutant that
removes the gate the case is named for, **the fixture is caught by a downstream
arm instead and the suite stays green** — so the test does not fail when the gate
admits what it must reject.

| Named case in `TestReconcileRefusals` | Gate removed | Arm the case shifts to | Suite |
| --- | --- | --- | --- |
| `omitted manifest claim` | `checkClaimRelation` omission loop | `evidence claim binding` | green |
| `static echo drift` | `probe static claim echo` equality | `evidence claim binding` | green |
| `protocol not member` | `checkProbeMembership` protocol half | `evidence tuple binding` | green |
| `evidence for false claim` | `!claim.Value` half of `checkEvidenceSet` | `evidence id set binding` | green |

Root enabler: **0 of the 29 refusal assertions in `manifest_test.go` and
`manifest_pin_test.go` check which arm fired.** All 29 assert only
`IsMismatch(err)` / `IsStaleGeneration(err)` — and ~110 of the 121 arms return the
same `CodeMismatch`. Leaf 1 did assert `refusal.Detail` where it mattered; leaf 2
dropped that. Every one of these four fixtures over-determines the failure; each
needs to be narrowed so only the named rule can reject it, and the assertion needs
to name the clause.

### F3 (blocking, evidence) — the untrusted-executable gate at the production entry point is unevidenced

`checkManifestRecordBinding`'s executable-digest half

```go
if manifest.ExecutableDigest != record.ExecutableDigest {
    return &Error{Code: CodeUntrusted, …, Detail: "executable substitution"}
}
```

can be deleted whole and the suite stays green — no shift, no downstream arm, no
test reaches it. This is the §6.5 "trust established before any probe" gate on
`Registry.AdmitProbe`, the only registry-bound production entry point this leaf
ships. `TestAdmitProbeExternalRealm` drives an external-kind backend end to end on
the positive path; `TestAdmitProbeRefusesRecordDrift` covers the *version/kind/
platform* half (`M36` is killed), leaving the digest half — the substitution case —
positive-path-only.

Failure scenario: a manifest whose `executable_digest` differs from the trusted
registry record admits, and its probe's capabilities are activated, because the
probe agrees with its own manifest (`checkProbeIdentity` compares probe↔manifest,
not manifest↔record).

### F4 (blocking, evidence) — expiry boundary proven on one side only

`checkEvidenceLiveness` implements `observed_at <= now < expires_at`. At HEAD I
confirmed all four boundary points behave correctly. But:

* the lower bound has both sides covered (`future observed evidence` case, and
  `M23` future-half deletion is killed);
* the upper bound has only the far side. Narrowing `!now.Before(expires)` to
  `now.After(expires)` — i.e. admitting evidence at exactly `expires_at` —
  **survives**. `expired evidence` sets `expires_at` two months into the past; no
  case sits at the instant.

This is the shape this Story has already been burned by three times. It is one
table row in the existing case list.

### F5 (blocking, evidence) — the Probe and Evidence closed-schema gates are proven only for the Manifest

The three parsers are independent code paths with their own literals. The Manifest
parser's shape gates are well covered (`M85`–`M88` all killed).
`TestProbeDocumentRefusals` (10 cases) and `TestEvidenceDocumentRefusals`
(16 cases) omit the entire class the manifest test covers, and every one of these
survives deletion:

| Mutant | Gate deleted | Result |
| --- | --- | --- |
| M01 | `ParseProbe` schema URN | survives |
| M02 | `ParseProbe` schema_version | survives |
| M03 | `ParseProbe` `ParseID(backendID)` | survives |
| M04 | `ParseProbe` `implementation_kind` | survives |
| M05 | `ParseProbe` digest↔kind consistency | survives |
| M06 | `ParseEvidence` schema URN | survives |
| M07 | `ParseEvidence` schema_version | survives |
| M08 | `ParseEvidence` `ParseID(backendID)` | survives |
| M09 | `ParseEvidence` protocol major 1 | survives |
| M10 | `ParseManifest` `implementation_kind` | survives |

Failure scenario for M03/M08 specifically: `ParseID`'s reserved-namespace rule is
the gate that stops a third party minting `ax.evil`. `TestManifestDocumentRefusals`
has the `reserved backend id` case; Probe and Evidence have none, so the same rule
could be dropped from two of three parsers unnoticed. This is the
"clause present in one context profile, absent from the other" shape.

### F6 (non-blocking, evidence) — six gates whose class is unevidenced

Each survives; each is a one-line addition to an existing table-driven case list.

| Mutant | Gate | Why the existing case does not hold it |
| --- | --- | --- |
| M41 | `checkSortedUnique` duplicate half (`>=`→`>`) | `duplicate evidence ids` appends the *smallest* ID, producing a descending pair caught by the ordering half. An adjacent duplicate is never built. |
| M60 | `parseClaimList` duplicate half (`>=`→`>`) | `duplicate claims` appends `claims[0]`, same reason. |
| M79 | `decodeCappedValue` depth cap | the `deep` case is 40 nested arrays; with the cap removed it is refused by `document shape` instead (top-level not an object). Assertion is arm-blind, so nothing changes. |
| M81 | `decodeStrictObject` UTF-8 validity | Go's decoder substitutes U+FFFD, so identity recompute diverges and refuses anyway — the read-failure-vs-absence distinction this file's package doc claims is untested. |
| M39 | `parseAttestationSignature` scheme prefix | `bad signature scheme` uses `"sha256:abcd"`, which fails base64 under the mutant. An unprefixed valid Base64 (`"AA=="`) is admitted by the mutant and refused at HEAD; that input is never built. |
| M38 | `parseRealmMembers` required-members half for `credential_capable_execution_realm` | only the inverse (realm members on a plain claim) has a case. |

### F7 (non-blocking, evidence) — four declared bounds never measured

`M45` (`os_version` upper 256), `M89` (`platforms` non-empty), `M90` (16-claim
cap), `M91` (256-evidence-ID cap) all survive. Leaf 1 measured its bounds at
33-distinct/32-admission; leaf 2 declares four and measures none.

### F8 (non-blocking) — two unreachable arms and one inaccurate doc

* `checkProbeIdentity` L1550 `return mismatchf("probe manifest binding")` is dead:
  the enclosing `if` is reached only when the backend IDs are already equal (the
  first `if` returned otherwise), so the inner `if` is always true. Never executed
  in coverage.
* `parsePlatformList` L950 `platforms[index-1] == platforms[index]` is subsumed by
  the `>=` comparison two lines above. Never executed.
* `checkEvidenceSet`'s doc says *"Expired, wrong-generation, or dangling evidence
  disables the claim it names"*. The code refuses the whole reconciliation. Not a
  behaviour bug — the code matches §4.B — but the comment describes a different
  model.

### F9 (blocking, DoD) — README carries a claim this leaf falsified, and no traceability binding was registered

`README.md:1977` discharges nothing for clause `15.3#3` on the stated grounds that
*"this repository builds no RPC hello frame, no provider plugin, and **no
TerminalBackend capability set**"*. Leaf 2 ships exactly a TerminalBackend
capability set: the closed 16-row `capabilityRegistry`, `Claim`, `Admitted`, and
the emission of `terminal_backend_capability_unproven`. That sentence is now
false.

Leaf 1 updated `README.md` and registered its new negative arm in
`internal/traceability/ownership.v0.5.0.json` with a re-pinned
`reviewedOwnershipCanonicalSHA256`. Leaf 2 touched neither; `tracecheck` still
reports `assigned_scopes=0`, `unevidenced=41`, `clauses_discharged=17/403` — no
change for §4.B/§4.D/§6.5/§7.A. DoD item *"README/doctor/capability evidence and
specification traceability are updated without unsupported claims"* is not met.

---

## What is good, stated so the rework does not undo it

* **Signed-bytes canonicalization is the strongest area in the change: 19/20
  narrowing mutants killed.** Dropping any single member from the
  `UnsignedEvidenceBytes` object — `capability`, `backend_generation_digest`,
  `expires_at`, `facts`, `issuer_id`, `platform`, `os_version`,
  `conformance_fixture_id`, `issuer`, `protocol_version`, `terminal_backend_id`,
  `observed_at`, and all five realm members — is caught, as is removing the domain
  separator or its NUL. That is a genuine narrowing result, not a delete-only one.
* Generation digest: 4/4, including domain removal *and* domain truncation to the
  non-NUL prefix, plus the empty-generation bound.
* Signature verification: forged-key and nil-verifier arms both held.
* Evidence tuple/coverage/ID-set: 11/12, including the two narrowing mutants that
  drop only the generation or only the fixture from the tuple comparison, and the
  `factsCover` first-requirement-only mutant.
* The logbook entry is honest about the three things the mutant protocol caught
  during development, including a real production bug (nil-vs-empty slice
  round-trip). That is the right instinct.

## Stated bounds — what this method cannot see

* **JCS is unprovable with these fixtures.** `M71` and `M72` (replace
  `jcs.Transform` with the raw `json.Marshal` output) both survive. They survive
  because these three schemas have ASCII-only keys and no numeric members, so
  Go's map marshalling and RFC 8785 agree byte for byte. This is a real property
  of the fixtures, not a defect, and not something a new test can fix without a
  fixture carrying a non-ASCII key or a number — which the closed schemas forbid.
  Report as unknown, not as covered.
* **`M92` (nil-registry check in `AdmitProbe`) and `M95` (absent-vs-null in
  `digestOrNullMember`) are near-equivalent mutants**: `Registry.Resolve` has its
  own nil guard and `checkExactMembers` already guarantees presence, so removing
  either changes the arm but not the outcome. Counted as survivors above for
  honesty; they are not defects.
* Mutation testing bounds the suite, not the code. A gate whose mutant is killed
  is exercised; it is not thereby proven correct against the spec. F1 is the case
  in point — area 14 scores `full` on mutants and the gate is still wrong.
* Coverage-arm mapping uses the max hit count of any block containing the return
  statement, so the 71/121 figure is an upper bound on arms executed.
* Concurrency is out of scope: nothing here is measured under `-race` beyond the
  default suite run.

## Required for acceptance

1. Fix F1 — decide `manifest`/`probe` against §4.D (`Capability dependencies:
   none`) and make `CheckOperation` and `CapabilitiesForOperation` agree; add the
   negative test for the operation whose gate must not refuse.
2. Narrow the four F2 fixtures so each is rejected only by the rule its case is
   named for, and assert `*Error.Detail` in every refusal case, not just the wire
   code.
3. Add the F3 executable-substitution case against `Registry.AdmitProbe`.
4. Add the F4 `now == expires_at` row.
5. Close the F5 class: give `ParseProbe` and `ParseEvidence` the same
   schema/schema_version/backend-identity/kind/digest cases the manifest parser
   has.
6. Re-establish G-A over the two claim slice fields, with a probe that re-reads
   the package interior rather than only re-comparing `Admitted`.
7. F9 — correct or re-scope the README `15.3#3` sentence and register this leaf's
   evidence in the ownership projection, as leaf 1 did.

F6, F7 and F8 are non-blocking but cheap; closing them with the rest is the
sensible move.

Re-run the whole mutant battery after rework: the survivors above are the
acceptance target, and the two stated-bound survivors (`M71`, `M72`) plus the two
near-equivalents (`M92`, `M95`) are expected to remain.
