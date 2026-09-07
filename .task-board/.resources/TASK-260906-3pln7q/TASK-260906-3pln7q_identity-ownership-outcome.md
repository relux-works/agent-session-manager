# TASK-260906-3pln7q — establish-single-object-identity-owner: outcome

Status: ready for review (role handoff `developer` → review).

## 1. Ownership decision

| Schema | Owner (validates) | Non-owner path (production call, not a copy) |
|---|---|---|
| `urn:ax:schema:provider-identity` 1.0.0 (§5.5) | `canonicaljson.validateProviderIdentityRecord` (unchanged; already registered for `section:5.5`) | `provhost.CheckIdentity` keeps its member-attributed dialect AND conjoins the verdict with `canonicaljson.CalculateObjectIdentity(body)` as its last arm (`internal/provhost/identity.go`). Deliberate copy retained for the §15.1 error surface, pinned by the bidirectional battery. |
| `urn:ax:schema:terminal-backend-manifest` 1.0.0 (§4.B) | `terminalbackend.ParseManifest` | `canonicaljson` validates via `validateTerminalBackendManifest`, which re-encodes the decoded object and drives the owner's production `ParseManifest` (`internal/canonicaljson/closed_shapes.go`). |
| `urn:ax:schema:terminal-backend-probe` 1.0.0 (§4.B) | `terminalbackend.ParseProbe` | Same via `validateTerminalBackendProbe` → `ParseProbe`. |
| `urn:ax:schema:terminal-capability-evidence` 1.0.0 (§4.B) | `terminalbackend.ParseEvidence` | Same via `validateTerminalCapabilityEvidence` → `ParseEvidence`. |

The transfer is recorded in the traceability registry as four new acceptance
cases (`terminal-manifest-identity-ownership`,
`terminal-probe-identity-ownership`, `terminal-evidence-identity-ownership`,
`provider-identity-ownership`), naming the owner entry point as production
and the agreement tests as tests. Registry digest, report count (94→98),
CLI expectation, and README figure updated with it.

Direction note: the repo already deduplicates toward domain owners
(`config` imports `terminalbackend.ParseID`; `sessadapter`/`dirnode`
delegate strict decoding to `environ`). This leaf follows it: the generic
identity package calls the domain owner; the domain package calls nothing
new. No import cycle (`terminalbackend` production imports only `scalar`).

## 2. AC coverage: 7 of 7 rows driven through production entries

1. One owner validates provider-identity; non-owner reaches it through a
   production call: `provhost.CheckIdentity` → `canonicaljson.CalculateObjectIdentity`
   (`internal/provhost/identity.go`). Driven by `TestIdentityAgreementAcrossValidators`
   (34 rows incl. 5 owner-gate rows) and the inventory witness
   `identity owner-gate extensions content` (exercises the new arm at
   `CheckIdentity`).
2. One owner validates each terminal schema; placeholders implemented-or-transferred:
   transfer to `terminalbackend`, recorded (see §1). Driven by
   `TestTerminalManifestIdentityAgreement`, `TestTerminalProbeIdentityAgreement`,
   `TestTerminalEvidenceIdentityAgreement`, `TestTerminalRealmEvidenceIdentityAgreement`.
3. Bidirectional agreement over one corpus, failing on divergence either way:
   terminal battery (valid + drop-each-member + wrong-type-each-member over
   AST-derived member lists + tampered claim + pre-delegation gates +
   surrogate sweep) and the extended environ battery. Deleting either side's
   gate fails rows on that side (proven: D1/D2/D3/N4/N5).
4. `ParseManifest` ↔ `CalculateObjectIdentity` agree on every valid terminal
   manifest incl. digest equality: `valid manifest agrees with equal digests`
   (owner ID == test-side recipe == canonical digest). Realm and probe/evidence
   halves identical.
5. `checkIdentity` ordering closed: identity binding now verified after all
   member-type rules in all three parse entries, plus a structural number
   walk inside `objectIdentity` that refuses before any JCS transform.
   Proven by `TestTerminalIdentityOrderingRefusesMemberTypeBeforeBinding`
   (numeric member + stale claim → member-type arm, never binding) and the
   white-box `TestObjectIdentityRefusesNumbersBeforeCanonicalization`.
6. Negative tests for each new refusal arm: `document size`
   (`TestTerminalDocumentsRefuseOversizeInput`, all three schemas),
   `document member type` via the walk (white-box, 4 placements),
   owner-gate conjunction (inventory witness + 5 environ rows). New arms
   declared in the terminalbackend arm inventory (196→198, bijection green)
   and the provhost inventory (166→167, witnessed).
7. Ownership registry names the owner: 4 new acceptance cases (see §1);
   `tracecheck`, `TestVerifyRepositoryAcceptsExactOwnership`, README pin green.

## 3. Mutation battery (production-derived denominator: changed gates)

Applied 13, killed 10, survived 2 (bounded), compile-fail 1, not-applied 1.
Killed-over-applied: 10/13. Each mutant was planted, probed, and reverted
(files restored from pre-battery copies; final diff is feature-only).

| Mutant | Narrows the gate to | Named failing test (exit 1) | Outcome |
|---|---|---|---|
| N1c tb: number walk blinded + identity-first restored (exact original defect) | JCS rounds before refusal | `TestTerminalIdentityOrderingRefusesMemberTypeBeforeBinding` (reports binding, want member type) | KILLED |
| N2 tb: number walk blinded (`refuseIdentityNumbers(nil)`) | numbers enter JCS | `TestObjectIdentityRefusesNumbersBeforeCanonicalization` | KILLED |
| N3 tb: `document size` check removed | oversize admitted | `TestTerminalDocumentsRefuseOversizeInput` (all 3 schemas) | KILLED |
| N4 cj: delegate only when extensions empty | extension-bearing docs admitted | `TestTerminalManifestPreDelegationGatesAgree/non-empty_extensions` (+ drop/number extensions rows) | KILLED |
| N5 provhost: owner gate only when extensions empty | extensions-content classes admitted | `TestIdentityAgreementAcrossValidators` (exactly the 5 owner-gate rows) | KILLED |
| N6 tb: lone-low range `0xdc00..0xdfff` → `0xdc00..0xdbff` (`\u` token preserved) | 1024 lone lows admitted; behavioral suite (`ParseManifest` + `Canonicalize`) executed, not only the static gate | `TestSurrogateGateVerdicts`, `TestSurrogateGateAgreesWithCanonicalJSON` | KILLED |
| D1 tb: manifest `checkIdentity` deleted | binding never verified | `TestTerminalManifestIdentityAgreement/tampered_claim…` + pre-existing `TestManifestIdentityMismatch` | KILLED |
| D2 cj: manifest validator back to refuse-everything | valid manifests refused | `TestEveryValidIdentityFixtureIsAcceptedAtItsProductionEntry/terminal_backend_manifest` | KILLED |
| D3 provhost: owner-gate block deleted | dialect-only verdicts | `TestIdentityAgreementAcrossValidators` (5 owner-gate rows) | KILLED |
| C1 tb: dead second `document size` arm (`if false`) | — (no behavior change) | `TestDerivedRefusalArmsAreAllDeclared` only; behavioral oversize test still green | KILLED-BY-CENSUS |
| S0 tb: identity-first restored, walk kept | — | none (suite green) | SURVIVED — bound: with the structural walk present, `checkIdentity` position has no observable verdict; the walk subsumes the ordering. The move-last stays as the readable enforcement of the fixture wording. |
| S1 provhost: owner gate only when extensions non-empty | — | none (provhost + environ suites green) | SURVIVED — bound: the conjoined verdict is load-bearing exactly for non-empty extensions; every other divergence class is refused identically by the dialect arms (all 16 members type-checked, shared decoder, shared scalar rules, rune-count bounds). |
| F1 cj: validator definition renamed, call site kept | — | `go build ./...` (`undefined: validateTerminalBackendProbe`) | COMPILE_FAIL |
| NA1 tb: "delete the duplicate probe `checkIdentity`" | — | n/a: exactly 1 call site exists | NOT_APPLIED |

Every shipped gate has a narrowing kill: ordering/number (N1c/N2), size (N3),
delegation (N4), owner conjunction (N5), surrogate incl. token-preserving
behavioral attack (N6). Delete-only evidence was never accepted alone.

## 4. Verification (real exit codes, standalone processes, no pipes)

- `go build ./...` → 0
- `go vet ./...` → 0; `GOOS=windows go vet ./...` → 0; `gofmt -l internal` → empty
- `go run ./internal/traceability/cmd/tracecheck` → 0 (acceptance_cases=98)
- `go generate ./internal/catalog` → no diff
- `go test ./... -count=1` → 0 (all 22 packages; one interim failure in
  `tracecheck/main_test.go` stale 94-pin, updated to 98, re-green)
- `go test -race` all packages in 4 groups → 0 each
- `go test ./... -cover -count=1` → 0 (terminalbackend 94.1%, canonicaljson 97.1%,
  provhost 85.8%, environ 89.8%)
- cigate claims + contract + target gates → 0; fuzz targets derive 13/13;
  fuzz smoke 13/13 `ok`
- Fixture matrix packages (`secprim`, `secconftest`) green on macOS; ubuntu
  matrix and Windows execution are CI's; `GOOS=windows go vet` covers compile.

## 5. Candidate tree

OID `764fabeb4e3853756b61fe7297d48a61d2861c17` (`git write-tree` on the
staged worktree, then `git reset`; no commits made on the story branch).
Verified by listing: `git ls-tree -r <OID>` contains
`internal/terminalbackend/identity_ownership_test.go`,
`internal/provhost/identity.go`, `internal/canonicaljson/closed_shapes.go`,
`internal/environ/identity_agreement_test.go`, and the listed blob hash for
the new test file equals `git hash-object` of the worktree file.
(A first `write-tree` without staging missed the untracked test file; the
recorded OID is the staged one, verified to hold it.)

## 6. Stated bounds and notes for review / siblings

- `terminal-instance-binding` stays a refuse-everything placeholder in
  `canonicaljson` (out of scope: no second validator exists for it, so no
  duopoly to close; §7.A untouched).
- `section:4.B` binding still points at `catalog.ForRelease` (unevidenced,
  as before): the transfer is recorded as acceptance cases, following the
  existing pattern (all other `terminalbackend` cases are likewise
  group-unreferenced). Re-pointing the coarse binding is a coverage-claim
  change for the story, not this leaf.
- Canonicaljson terminal fixtures carry correct self digests up front
  (unlike other fixtures' zero placeholders), because delegation verifies
  the binding inside `CalculateObjectIdentity`. Documented at the fixture
  list; `withCorrectIdentityClaimForTest` recomputation agrees.
- `TestUnknownTopLevelMemberIsRefusedWhileTheSameKeyIsAdmittedUnderExtensions`
  admission half is scoped to open-extensions schemas via a
  production-derived delegation set: the pinned §4.B tables declare
  `extensions` as "exact empty object `{}`", so the owner refuses
  non-empty extensions and the 1.5 sweep's universal admission claim was
  over-broad. The closed rule is proven at the owner (checkExtensions arms
  + agreement extensions rows).
- Sibling touch points: `internal_pin_test.go` lost its `canonicaljson`
  import (import cycle after delegation); the surrogate sweep now runs
  externally in `identity_ownership_test.go` with branch pins kept
  white-box. `provhost.CheckIdentity` gained a `canonicaljson` import.
  `TASK-260906-33xcnc` (environment copies) may want the same conjunction
  shape for `decodeStrictObject`; `TASK-260906-vmzk0y` (protocol v3) touches
  adjacent provhost errors but not `CheckIdentity`'s verdict.

Files changed: `internal/terminalbackend/manifest.go`,
`internal/terminalbackend/internal_pin_test.go`,
`internal/terminalbackend/refusal_arm_inventory_test.go`,
`internal/terminalbackend/identity_ownership_test.go` (new),
`internal/canonicaljson/closed_shapes.go` (+ fixtures/sweeps/guards tests),
`internal/provhost/identity.go`,
`internal/provhost/refusal_arm_operations_c_test.go`,
`internal/environ/identity_agreement_test.go`,
`internal/traceability/{ownership.v0.5.0.json,traceability.go,traceability_test.go,cmd/tracecheck/main_test.go}`,
`README.md` (94→98 figure only).
