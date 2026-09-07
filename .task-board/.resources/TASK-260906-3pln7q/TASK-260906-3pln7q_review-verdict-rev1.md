# TASK-260906-3pln7q — review verdict, CR revision 1

**Verdict: ACCEPTED** (`accept_cr`, revision 1). The delegation is real, not a
second path that happens to agree today; every gate the leaf ships was attacked
and killed. Four findings are recorded below — all are claim/documentation
accuracy, none is a behavior defect, and three of the four understate the
change's own coverage.

Reviewer: RUN-260906-0924fb (claude / reviewer archetype). Every number below
was produced by this run in the story worktree at candidate tree
`764fabeb4e3853756b61fe7297d48a61d2861c17`; nothing is carried over from the
producer's outcome document.

---

## 1. Provenance and frozen ground (G-E) — confirmed

| Check | Method | Result |
| --- | --- | --- |
| Candidate tree OID equals the record's | temp-index `read-tree HEAD` + `add -A` + `write-tree` | `764fabeb4e3853756b61fe7297d48a61d2861c17` — **MATCH** |
| The new untracked test file is in that tree | `git ls-tree 764fabeb internal/terminalbackend/` | `identity_ownership_test.go` blob `002a9bed…` present |
| Nothing outside the 20 declared paths moved | `git status --short \| wc -l` | 20 |
| Packages this leaf does not own | path filter | `internal/secprim`, `internal/secconftest` **untouched**. `internal/environ` is touched **test-only** (`identity_agreement_test.go`), extending the pre-existing provider-identity agreement battery — that is the AC's own artifact, not scope creep. |
| Tree after my full mutation battery | recomputed | `764fabeb…` — **RESTORED-EXACT**, no reviewer drift |

## 2. Gates green (re-run by this reviewer, not accepted from the artifact)

| Command | Result |
| --- | --- |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `gofmt -l internal cmd` | empty |
| `go run ./internal/traceability/cmd/tracecheck` | exit 0, `acceptance_cases=98` |
| `go test ./... -count=1` | **22 of 22 packages `ok`** |

## 3. G-A (blocking) — is the delegation real, or a second path that agrees?

**Answer: real, in both directions, proven by mutation rather than by reading.**

The structural claim (`canonicaljson` holds no shape logic of its own for the
three terminal schemas) is not taken on the code's word. Two mutants settle it:

| Mutant | What it does to the **owner** | Where it must redden if delegation is real | Result |
| --- | --- | --- | --- |
| **MU-2 (NARROWING)** `terminalbackend.checkExtensions`: `len(extensions) != 0` → `len(extensions) > 1` — the gate stays present, admits exactly one member of the class it must reject | owner now admits a one-member `extensions` object | in **`internal/canonicaljson`** | **KILLED** — `TestUnknownTopLevelMemberIsRefusedWhileTheSameKeyIsAdmittedUnderExtensions/{terminal_backend_manifest,terminal_backend_probe,terminal_capability_evidence}`: "error = `<nil>`, want shape refusal". A local copy in canonicaljson would have kept refusing. |
| **MU-3 (TIGHTENING)** same gate: `!= 0` → `!= 1` — the owner now refuses every valid terminal document | owner refuses alone | in **`internal/canonicaljson`** | **KILLED** — `TestEveryValidIdentityFixtureIsAcceptedAtItsProductionEntry` fails on **all three** schemas, and the rendering carries the owner's own arm (`terminal backend manifest 1.0.0: terminal backend refused: … at document extensions`). |

MU-2 is the decisive one: narrowing the owner made the **delegate** admit. There
is exactly one decision.

Reverse direction, `provider-identity`:

| Mutant | Result |
| --- | --- |
| **MU-5 (NARROWING the owner)** `canonicaljson.validateExtensionsObject`: guard rewritten to `key != "x" && (…)`, admitting exactly the one key `"x"` | **KILLED** on the provhost side — `TestIdentityAgreementAcrossValidators/extensions_open_key_refused` ("provhost admitted …") and `TestEveryArmWitnessRefusesAtTheProductionEntry/…identity_owner-gate_extensions_content` ("want a failure, got nil"). `provhost` reads the owner's verdict; it holds no copy of the extensions rule. |
| **MU-4 (arm deletion)** the conjoined `canonicaljson.CalculateObjectIdentity` block removed from `CheckIdentity` | **KILLED** — 5 environ rows plus **two** provhost inventory tests: `TestEveryArmWitnessRefusesAtTheProductionEntry` *and* `TestWitnessedArmsAreAllDerived` ("witness naming no derived production arm"). The owner can refuse alone. |

**Does provhost's retained dialect change any machine answer, or only the error
surface?** Structurally the answer is now closed: admission is
`dialect ∧ owner`, so no document the owner refuses can be admitted — there is
no admit-hole by construction. I probed 21 directed vectors through
`CheckIdentity`, `CalculateObjectIdentity` and `VerifyObjectIdentity` (probe
written into `internal/environ`, run, deleted; tree re-verified `764fabeb…`).
Every divergence found is the dialect being *stricter* or attributing a
different arm — never admitting more. See **Finding 2** for one class the
producer's stated bound got wrong.

## 4. G-B (blocking) — the corpus behind the agreement

**Terminal battery, `internal/terminalbackend/identity_ownership_test.go` —
derived, not hand-built.** Measured by `-v` subtest count: **137 subtests, all
PASS.**

| Sweep | Denominator | Source |
| --- | --- | --- |
| drop-each-member | 12 + 16 + 24 = 52 | `deriveTerminalMembers` reads the **production** `manifestMembers` / `probeMembers` / `evidenceMembers` composite literals out of `manifest.go` via `go/ast`; those same vars are the argument to `checkExactMembers` in production, so the roster and the gate are one table |
| wrong-type (safe number) per member | 52 | same derivation |
| valid + tampered-claim + realm halves | 8 | |
| pre-delegation gates (both arms named per row) | 10 | |
| oversize / ordering / member-list pins | 9 | |
| surrogate agreement sweep | **~4 100 vectors** (0xD800–0xDFFF lone ×2048, high+each ×2048, plus corners) | exhaustive enumeration, both entries per vector |

**Does it contain the class that used to diverge?** Yes, and the test catches
its removal. Before this leaf `ParseManifest` admitted every valid terminal
manifest that `CalculateObjectIdentity` refused as unsupported. **MU-1**
(registration reverted to `rejectUnsupportedImmutableObjectShape`) is **KILLED**
by exactly that row: `TestEveryValidIdentityFixtureIsAcceptedAtItsProductionEntry/terminal_backend_manifest`
→ "complete immutable-object shape validation is unavailable for
urn:ax:schema:terminal-backend-manifest@1.0.0". The previously-broken class is
the first row of the corpus, not an afterthought.

**Bidirectionality is genuine** in the surrogate sweep: `rejectVector` requires
**both** sides to refuse *at the surrogate arm*; `admitVector` requires both to
pass the surrogate question and requires canonicaljson's error to carry the
owner's exact member arm. This is not the one-directional shape that hid a wire
MUST violation for four rounds in the sibling story.

**Stated bound I am recording, not a defect:** the `provider-identity` corpus in
`internal/environ` is **hand-built (34 rows)**, not derived. Its member table is
pinned derived-equal on both sides by `TestIdentityMemberTablesAreOneTable`
(AST over `provhost.identityRequired` and canonicaljson's
`requireExactMembers("Provider Identity Record", …)`), which is the strongest
derivation available for two differently-shaped validators.

## 5. G-C (blocking) — the ordering fix

**Closed, and measured on all three schemas** — though not by the test that
claims to measure it (Finding 1/4).

| Mutant | Result |
| --- | --- |
| **MU-6** manifest `checkIdentity` restored to its exact pre-leaf position (before member validation), number walk kept | **KILLED**, deterministic over 2/2 runs — `TestTerminalManifestPreDelegationGatesAgree/non-empty_extensions`: refusal reported at `document identity binding`, want `document extensions`, and the same on the canonicaljson side |
| **MU-6b** the same revert for **probe and evidence** | **KILLED** — `TestProbeDocumentRefusals/non-empty_extensions` and `TestTerminalRealmEvidenceIdentityAgreement/realm_members_on_a_non-realm_claim_refused_by_both` |
| **MU-7** `refuseIdentityNumbers(object)` call removed from `objectIdentity` (planting its removal, as briefed) | **KILLED** — `TestObjectIdentityRefusesNumbersBeforeCanonicalization`, all 4 placements |
| **MU-8 (NARROWING)** the walk keeps its `map[string]any` recursion but loses the `[]any` case | **KILLED** — the `nested array number` and `decoded float` placements. The walk refuses unsafe integers itself; it does not lean on `Transform` having moved. |

**No path reaches `jcs.Transform` before member-type validation.** Verified by
call-site census, not inference: `jcs.Transform` has exactly two production
sites in the package. `manifest.go:650` is inside `objectIdentity`, whose only
production caller is `checkIdentity`, whose only three production callers are
the tail of `parseManifestObject` / `ParseProbe` / `ParseEvidence` — after every
member rule — and which is additionally fronted by the structural number walk.
`manifest.go:1544` operates on an already-parsed `Evidence` **struct** (typed Go
fields, reconstructed for the signature domain), so no undecoded document
reaches it. Error paths included: `objectIdentity` returns before `Transform` on
the walk's refusal.

## 6. G-D — the battery, re-derived

I did not accept the producer's 13 rows. I derived the denominator from the
production delta myself — 9 changed/added production gate sites (3 delegating
registrations, 1 provhost conjunction, 1 document-size bound, 1 number walk with
3 type arms, 3 identity-binding reorderings) plus 4 census/roster gates the leaf
relies on — and planted **13 mutants of my own**.

**Reviewer battery: 13 applied / 13 KILLED. 0 survivors, 0 COMPILE_FAIL,
0 NOT_APPLIED.** Categories counted separately:

| Category | Mutants | Result |
| --- | --- | --- |
| NARROWING (gate present, admits exactly one member of its rejected class) | MU-2, MU-5, MU-8, MU-9, MU-11 | 5/5 killed |
| TIGHTENING (owner refuses alone) | MU-3 | 1/1 killed |
| ARM DELETION | MU-1, MU-4, MU-7 | 3/3 killed |
| ORDERING REVERT | MU-6, MU-6b | 2/2 killed |
| CENSUS / source-text gate, **token-preserving** | MU-10, MU-11, MU-12a | 3/3 killed |

Edge-exactness (memory of "bound witnessed far out of range"): **MU-9** moves
the 5 MiB bound by **one byte** (`> max` → `> max+1`) against a document padded
to exactly `max+1`. **KILLED** on all three schemas. The bound is witnessed at
its edge, not far outside it.

**Token-preserving attacks on the source-text gates** (the DoD row a static
checker cannot satisfy):

- **MU-10** — the manifest registration wrapped in a `FuncLit` that *still
  spells* `validateTerminalBackendManifest` in source. The AST derivation
  **fails closed**: `register call in mustBuildImmutableObjectShapeValidators
  does not spell a literal schema and a named validator`. It does not silently
  rescope the delegated-schema set.
- **MU-11** — `manifestMembers` literal left byte-identical (the derivation
  still reads 12 members and still passes its size tripwire) while production is
  widened to `append(manifestMembers, "works_extra_member")`. **KILLED** by the
  behavioral suite, not the checker: `TestManifestDocumentRefusals` reports
  `document members` where each row wants its own arm.
- **MU-12a** — the ownership registry names a test declaration that does not
  exist. `tracecheck` **fails closed on the declaration itself**, ahead of the
  canonical SHA pin: `acceptance case "terminal-manifest-identity-ownership"
  test owner: declaration "…XYZ_DOES_NOT_EXIST" is absent from …`. The registry
  rows are real bindings, not a census of names.

Every mutant harness run executed the **behavioral** package suite, never the
static checker alone.

## 7. AC coverage: 4 of 4 rows driven through production entries

| AC row | Production call site | Named driving test | Attacked by |
| --- | --- | --- | --- |
| Exactly one owner per schema; non-owner reaches it by production call | `canonicaljson.CalculateObjectIdentity` → `closed_shapes.go:validateTerminalBackend{Manifest,Probe,Evidence}` → `terminalbackend.Parse{Manifest,Probe,Evidence}`; `provhost.CheckIdentity` → `canonicaljson.CalculateObjectIdentity` | `TestTerminal{Manifest,Probe,Evidence,RealmEvidence}IdentityAgreement`, `TestIdentityAgreementAcrossValidators` | MU-1, MU-2, MU-3, MU-4, MU-5 |
| Bidirectional agreement over one corpus, failing on divergence either way | same entries | `TestSurrogateGateAgreesWithCanonicalJSON` (`rejectVector`/`admitVector` both directions), `TestTerminalManifestPreDelegationGatesAgree` (both arms per row) | MU-2, MU-3 |
| Safe-integer ordering defect closed: no `jcs.Transform` before member-type refusal | `parseManifestObject`/`ParseProbe`/`ParseEvidence` tail → `checkIdentity` → `objectIdentity` → `refuseIdentityNumbers` | `TestTerminalManifestPreDelegationGatesAgree`, `TestProbeDocumentRefusals`, `TestTerminalRealmEvidenceIdentityAgreement`, `TestObjectIdentityRefusesNumbersBeforeCanonicalization` | MU-6, MU-6b, MU-7, MU-8 |
| Negative tests prove each refusal arm; the registry names the owner | `tracecheck` over `ownership.v0.5.0.json` | `TestVerifyRepositoryAcceptsExactOwnership`, `TestRunReportsExactCoverageAndFailsClosed`, both arm inventories | MU-12a |

## 8. The ownership split itself — judged, not only its implementation

The leaf split ownership **per schema** rather than forcing one owner. I judge
that correct for this codebase:

- `terminalbackend` production imports **only `internal/scalar`** (verified with
  `go list -deps`). It is a low-level leaf despite its domain name, so
  `canonicaljson → terminalbackend` is not a layering inversion and creates no
  cycle risk beyond `scalar`.
- The direction matches the deduplication already set in this repo
  (`internal/config` imports `terminalbackend.ParseID`).
- The §4.B closed shapes genuinely live in `terminalbackend` — it already owned
  the member tables, the vocabulary registries and the realm rules; moving them
  into `canonicaljson` would have been a much larger, riskier transfer for the
  same guarantee.
- The `provider-identity` direction is the opposite because the shape rule
  genuinely lives in `canonicaljson` and `provhost` only needs a different
  **error surface**. Conjoining rather than replacing keeps §15.1 member
  attribution while making admission `dialect ∧ owner`, which is the correct
  fail-closed composition.

The one real cost is acknowledged and paid: `internal_pin_test.go` lost its
`canonicaljson` import to the new cycle, and the sweep moved to an external test
package **without losing a vector** (I counted the moved sweep: the full
0xD800–0xDFFF enumeration survives, and the branch-level pins stayed white-box in
`TestSurrogateGateVerdicts`).

---

## Findings (recorded, none blocking)

**F1 — the mutation table reports a KILL as a survivor (conservative error).**
`internal/…` outcome §3, row **S0** ("tb: identity-first restored, walk kept →
none (suite green) → SURVIVED"). Reproduced exactly — manifest `checkIdentity`
moved back to its pre-leaf position with the number walk kept — the full
`internal/terminalbackend` suite is **RED**, deterministically over 2/2 runs, at
`TestTerminalManifestPreDelegationGatesAgree/non-empty_extensions`. The row is
KILLED, the ratio is **11/13** rather than 10/13, and the attached bound ("the
walk subsumes the ordering; the move-last stays as readable enforcement") is
wrong — the move-last is independently measured. Most likely a `-run` mask
narrower than the claim ("suite green" appears to have been read off the
ordering test alone). Direction is safe: the change is better covered than its
own document says.

**F2 — a stated bound that is factually wrong.** Outcome §3, row **S1**: "every
other divergence class is refused identically by the dialect arms". Measured
false. A `provider-identity` body padded past 5 MiB is admitted by **every**
provhost dialect arm and refused **only** by the conjoined owner gate
(`identity is not a valid provider identity` ← `encoded identity object is
6291457 bytes … maximum is 5242880`). The owner conjunction is therefore
load-bearing for at least two classes — extensions content **and** document size
— and the environ corpus carries no oversize row. The S1 *verdict* (survives the
current suite) is correct; the *bound* is not. No behavior defect: the gate is
present and refuses correctly, and MU-4 shows a full deletion is caught by 5
rows plus 2 inventory tests, so the wrong bound cannot lead to an undetected
removal. **Suggested for a sibling leaf:** one oversize row in
`identityCorpus`, which would convert S1 into a kill.

**F3 — observation, outside this leaf's AC.** `CheckIdentity` conjoins with
`CalculateObjectIdentity`, which *computes* the omit-self digest but does not
verify the claimed `record_id`. Measured: a record with `record_id` set to
all-zeros is admitted by `provhost.CheckIdentity`. Note this is not a
regression and not a cheap fix — the shipped valid fixture *itself* fails
`VerifyObjectIdentity`, so its `record_id` is a placeholder and switching the
conjunction to `VerifyObjectIdentity` would require a real fixture digest and a
§5.5 binding decision. Contrast: the three terminal schemas **do** get binding
verification through delegation, because `Parse*` checks it. Recording the
asymmetry for the story; not rework here.

**F4 — an inaccurate claim in a test comment.**
`TestTerminalIdentityOrderingRefusesMemberTypeBeforeBinding` states "Under the
old order (identity first) every row below reported 'document identity binding'
instead." Measured false: with the number walk present, that test passes
unchanged under the old order (it is what made MU-6 look like a survivor). The
test measures the **walk**, not the ordering. The ordering is really measured by
`…PreDelegationGatesAgree/non-empty_extensions`,
`TestProbeDocumentRefusals/non-empty_extensions` and
`TestTerminalRealmEvidenceIdentityAgreement/realm_members…`. Comment-only.

---

## Why accepted rather than `changes_requested`

All four findings are claim accuracy, three of them in the direction that
*understates* the change. No gate is missing, weakened, uncalled or bypassable:
every gate the leaf ships was attacked with a narrowing mutant and killed, the
two source-text gates were attacked with token-preserving mutants and fail
closed, the previously-divergent class is the corpus's first row and its
delegation removal reddens, and the repository suite is green at the exact
candidate tree. Requesting a revision to fix two sentences in an evidence
document and one test comment would cost a full producer/reviewer cycle for no
change in the shipped guarantee; F2 and F3 are carried to the story instead,
where the sibling leaves (`TASK-260906-33xcnc` in particular) touch the same
composition shape.

`repeat-of: none`.
