# TASK-260830-2z3se0 — round-3 rework evidence (answers review-verdict-rev2)

Tree: worktree `task-board/story/STORY-260830-3drr2m`. Production changes in
`internal/sessadapter/operations.go` (B1 only); all other changes are tests in
`internal/sessadapter/` plus one LOGBOOK entry. No commit per the CR-shape
instruction.

## Fixes per finding

### B1 (blocking) — archive-mode validate success refused unconditionally
`checkValidateResult` no longer calls `checkValidateNullability`. The pinned
§7.8 validate success member list names no target members, so the
request-body nullability rule cannot apply to successes — option (a) from the
verdict, resolved by the document, not by a product decision.
`checkValidateNullability` and `checkValidateResult` comments updated to state
the request-only scope. New `TestValidateSuccessAdmitsEveryMode` drives
staged/live/archive successes through `CheckSuccessBody`; the archive subtest
is the previously unreachable mode (904 tests green over it).

### B2 (blocking) — four member maps without a content pin
`TestClosedMemberSetsAreDerivedFromSpec` now compares the Members gate map as
a set against the derived list for all four tail bodies (success, failure,
manifest, doctor). The `manifestMembers :=` local is renamed
`derivedManifest` so it cannot shadow the production map. The log now reports
member-map comparisons: `20 member maps pinned across 21 bodies`
(SupportedContractVersions has no production gate map by design).

### B3 (blocking) — constructor-closure sentence false for additive arms
New `TestRefusalConstructorsMatchProduction` derives the denominator from
production: every `axerror.New` site must sit inside a registered constructor
body, and the registered set must equal the error-constructing vars exactly
(both directions). The `refusalConstructors` comment now states what both
gates do (census for new sites, alias gate for aliased uses).

### N4 — CheckCallBinding zero-fact gates
New `TestCheckCallBindingZeroFactsRefuse`: all five zero-fact exemptions
refuse through their own arms (role, provider, manifest digest, executable
digest, zero admitted tuple).

### N5 — census sentences ahead of the walk
`vocabularyTablesInFile` handles shared and paired multi-name initializers
(three synthetic vectors added); the tagless-switch exemption is confined to
`readUTF16Escape` by function name (`classifySwitch`, proven by
`TestSwitchClassifierNamesItsExemption`); the census stated bound now names
the five function-local obligation lists (pinned behaviourally in both
directions) and the const/regexp shapes (absent from production).

## Mutant battery (narrowing; 14 killed / 14)

Method: each mutant applied by exact-string replacement, grep-confirmed
PRESENT and `go vet`-clean before its verdict; production restored after
every mutant (script `/tmp/mutant-battery-r3.py`). The named test passes on
the clean tree in every row (checked by the harness).

| Mutant | Narrows the gate to | Named test that fails |
|---|---|---|
| M-B1 success nullability restored | refuse archive success with no target members | TestValidateSuccessAdmitsEveryMode |
| M-B2a successMembers +1 | success envelope admits an extra member | TestClosedMemberSetsAreDerivedFromSpec |
| M-B2b failureMembers +1 | failure envelope admits an extra member | TestClosedMemberSetsAreDerivedFromSpec |
| M-B2c manifestMembers +1 | manifest admits an extra member | TestClosedMemberSetsAreDerivedFromSpec |
| M-B2d doctorResultMembers +1 | doctor result admits an extra member | TestClosedMemberSetsAreDerivedFromSpec |
| M-B3a 8th ctor failQuota + additive arm | unregistered constructor ships an unwitnessed arm | TestRefusalConstructorsMatchProduction |
| M-B3b inline axerror.New + additive arm | bare axerror.New ships an unwitnessed arm | TestRefusalConstructorsMatchProduction |
| M-N4a role gate `&& binding.Role != ""` | admit a call under a roleless binding | TestCheckCallBindingZeroFactsRefuse/zero_binding_role |
| M-N4b provider gate `&& binding.ProviderID != ""` | admit a call under a providerless binding | TestCheckCallBindingZeroFactsRefuse/zero_binding_provider |
| M-N4c manifest gate `&& binding.AdapterManifestDigest != ""` | admit a call under a digestless binding | TestCheckCallBindingZeroFactsRefuse/zero_binding_manifest_digest |
| M-N4d executable gate `&& binding.ExecutableSHA256 != ""` | admit a call under an exeless binding | TestCheckCallBindingZeroFactsRefuse/zero_binding_executable_digest |
| M-N4e tuple gate `&& admitted != (Tuple{})` | admit a call against a zero tuple | TestCheckCallBindingZeroFactsRefuse/zero_admitted_tuple |
| M-N5a paired var table planted | paired table hides from the scan | TestClosedVocabularyTablesAreRegistered |
| M-N5b tagless switch planted elsewhere | tagless switch outside readUTF16Escape hides | TestAllProductionSwitchesAreClassified |

Survivors: none. No stated-bound exemptions were exercised by this battery.

Anti-inflation check (the rev2 B3 lesson): with M-B3a / M-B3b applied, the
FULL suite fails on exactly one test — `TestRefusalConstructorsMatchProduction`
— while the 323-arm inventory stays green. The kill comes from the new
census, not the count floor.

Method bounds: battery is scoped to this rework (B1–B3, N4, N5). It does not
re-traverse the rev2 93-site surface; rev2 kills were not re-measured here.
Delegated packages (`canonicaljson`, `scalar`, `axerror`) not attacked. The
package has no in-repo importers; gates are measured at its exported API.

## AC coverage delta (this round; prior 10-row coverage stands)

| AC row | Production call site | Named committed test |
|---|---|---|
| closed operations, validate modes | CheckSuccessBody (operations.go) | TestValidateSuccessAdmitsEveryMode (staged/live/archive) |
| negative/refusal, member gates | unknownMember via CheckSuccessEnvelope / DecodeManifest / DecodeDoctorResult | TestClosedMemberSetsAreDerivedFromSpec (20 map pins) |
| negative/refusal, construction closure | Discover / CheckSuccessBody | TestRefusalConstructorsMatchProduction |
| tuple/binding gates | CheckCallBinding (discovery.go) | TestCheckCallBindingZeroFactsRefuse (5 of 5) |
| census scope | scanVocabularyTables / switch walk | multi-name vectors, TestSwitchClassifierNamesItsExemption |

## Gates (exit codes observed directly, no pipes)

- `go build ./...`: exit 0
- `go test ./... -count=1`: 17/17 packages ok, exit 0
- `go test ./internal/sessadapter/ -count=1 -v`: 923 PASS, 0 FAIL, exit 0
- `go test ./internal/sessadapter/ -cover`: 81.5% of statements, exit 0
- `go test -race ./internal/sessadapter/ -count=1`: exit 0
- `go vet ./...`: exit 0; `GOOS=windows go vet ./internal/sessadapter/`: exit 0
- `gofmt -l internal/`: empty
- `go run ./internal/traceability/cmd/tracecheck`: ok (acceptance_cases=88), exit 0
