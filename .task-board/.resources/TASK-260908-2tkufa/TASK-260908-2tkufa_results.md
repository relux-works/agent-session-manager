# TASK-260908-2tkufa recovery outcome

Ready for independent review, not accepted or delivered. Candidate remains
UNCOMMITTED on `task-board/story/STORY-260908-18woqo`, checkpoint
`b4c43495b1824148b2ce6e2bb80e10bf4dccfccf` (0 commits behind local main).
No commit, rebase, branch switch, integration or hosted CI was performed.
This outcome supersedes the prior results' 22-file and successful-CR claims.
The attached evidence archive preserves that prior outcome for comparison.

## Changes and scope

Recovered the preserved candidate rather than restarting. Catalogue generator,
current registry, CI contract roots and traceability consume v0.6.0, with exact
V050/V043 projections and explicit CurrentV050 localstore binding. The v0.6.0
catalogue carries 63 contract rows and RPC 5 vocabulary. Error 1.4 is census-only;
its runtime error vocabulary and CLI5 support remain pending. Metadata canonical
digest: `5788d45d89a7f7a39f91130e4758d86697d22097a39e0a86cb32a0e9de87b963`.
Ownership canonical digest:
`a1ab29139bbf53259f6b76b4095a71531be6c2e7197979a598ffb6d54f3b130d`.

Fixed the actual CR command in task-board.config.json: only cataloggen's
metadata/lock version changed from v0.5.0 to v0.6.0. Applied the same expressly
authorized one-line change at the control root. Its pre-existing config diff
was empty; candidate and control configs now match byte-for-byte. Every other
field is unchanged. `config-consistency.json` includes complete before/after
hashes and exact commands; `control-config-before.json` and the empty before
diff are preserved. Models, signing, required gates and hosted-trigger policy
were not changed.

All 13 added sections name pending task owners in reviewed registry gaps.
`adoption-v0.6.0.md` and TASK-260908-2tkufa_ownership.md define non-duplicated
shared-API/caller responsibilities. Six existing task scopes were appended with
v0.6.0 obligations while their old scope and AC were preserved. New
TASK-260909-2ez769 owns credentials/trust lifecycle and Config-4 migration;
RPC admission depends on it. All seven owners are linked to Story landing.
The original blocked selector/RPC leaves remain blocked; no dependent was
resumed. Other Story partial implementations and historical accepted owners
were not edited.

Corrected the prior gap interpretation: Sections 11.10.5 and 14.7.5 define
upstream synthetic publication validators. Their source ownership remains in
the spec repository; local conformance owners must prove runtime behavior.
No AX runtime SPEC.md policy parser is invented.

Removed the prior candidate's hypothetical Config-4 reader admission from
AssessCompatibility. No config production code changes remain. Test fixtures
now distinguish catalogue census from implemented readers: 6 read-only pairs,
6 compatible pairs, 4 refused Config-4 reader pairs, plus explicit Config-4
Load and Migrate refusals. Historical 1/2/3 production behavior is retained.

## AC coverage: 4 of 5 AC rows driven; fifth is a stated delivery bound

| AC row | Production entry and executable evidence | Result / bound |
| --- | --- | --- |
| Full census and adopted source | catalog.Current / cataloggen.Generate: TestCurrentMatchesReviewedV060Catalog, TestGenerateMatchesCommittedTypedCatalog, TestGenerateRefusesStaleV050Authority; cigate.PinnedReleases / PinContractSets / CheckContractRoots: TestPinnedReleasesAgreeAcrossAuthorities, TestVerifyContractPreservationLive | V060 current, 63 rows, 3 release projections; stale inputs refused |
| Real owners for new/refined obligations | traceability.VerifyRepository / VerifyAssignedSections: TestAdoptedSectionsHavePendingOwnersAndRefuseRuntimeAdmission; board set_details/create/link operations captured in owner-mutations-01.log and owners-final.json | 13 of 13 source-derived added headings have pending owners and refuse runtime admission; board scopes assign actual responsibilities, not runtime support |
| Generators and gates detect omissions/stale authority | cataloggen CLI run: TestConfiguredCRCatalogGateConsumesAdoptedAuthority, TestGenerateDirectiveConsumesAdoptedAuthority; traceability.VerifyRepository: TestVerifyRepositoryRefusesStaleV050Lock, TestVerifyRepositoryRefusesMissingSelectorContract, TestVerifyRepositoryRejectsNarrowedOwnership | Actual configured command and directive are executed through CLI entry; missing ownership and stale source refused |
| Truthful legacy claims | catalog.ForRelease: TestV050ProjectionMatchesHistoricalLock; specpin.CurrentV050: TestCurrentV050IsTheExplicitHistoricalBaseline; config.Load/Migrate/AssessCompatibility: TestLoadSupportsEveryPinnedConfigurationVersionAndTranslatesLegacyAtProductionEntry, TestMigrateRefusesEveryTargetOutsideTheUpgradeVocabulary, TestAssessCompatibilityPinsEveryPinnedVersionPairAtTheProductionEntry | Historical authority artifacts byte-identical to checkpoint; 17 of 17 clause IDs/excerpts/acceptance owners preserved with migrated source line numbers; unimplemented v4 reader/document/target refused |
| Reviewed signed Story before dependent activation | Managed handoff and Story dependency links | STATED BOUND: independent acceptance and signed Story landing are orchestrator-owned and not performed by this developer. Links remain intact; no early activation |

Tests are repository candidate files awaiting the managed signed snapshot;
they are deliberately not hand-committed on the Story branch. No upstream
cryptographic/network provenance was re-proven here: predecessor's attached
independent review and signed checkpoint are accepted inputs. This run
personally reran the local consumer tests, gates and mutations below.

## Measurements and validation

Darwin arm64, Go 1.25.5; Go CLI/library supports desktop/server platforms,
not iOS. No unrelated Apple build was run.

Tracecheck: contracts=63, normative_sections=36, acceptance_cases=101,
fixtures=32, compatibility_contracts=55; bindings=56, full=1, partial=3,
sliver=1, unevidenced=48, unmeasured=3, unowned=12,
clauses_discharged=17/463. This is inventory/clause evidence, not product
capability coverage. README's stale 53/428 example is corrected.

All 26 configured local commands passed (table below). The full test/race
commands in this first sweep started before the final assessment-refusal
correction; final-focused tests reran every touched package after it, exit 0.
The final handoff will run all configured gates against its exact captured
candidate and attach its own bounded validation log. Only that successful
handoff establishes the final candidate's CR validation; no success is presumed.

Final focused commands (all exit 0):
- `go generate ./internal/catalog` (byte-idempotent, generated file SHA256 d9e23d2f1532e860efe09367be7e01e25315ee07e6719689ba919b1f8718b556)
- `go test ./internal/catalog/... ./internal/cataloggen ./internal/cigate ./internal/traceability/... ./internal/config -v -count=1`
- `git diff --check`

Coverage in the full sweep: catalog 97.6%, cataloggen 83.9%, cigate 90.5%,
traceability 86.2%, config 94.7%, localstore 86.6%, specpin 87.3%, specdoc 100%;
cataloggen CLI 79.3%. Cross-builds are compilation evidence only, not Windows
or Linux runtime execution. No test, race, coverage or hosted result is forged.

## Narrowing evidence: 11 of 11 killed, 11 instrument controls passed

Each mutant runs the full relevant behavioral package suite with `-v -count=1`.
An instrument-only control emits a unique marker and exits 0; the mutated suite
must emit the same marker, exit 1 and contain a named semantic test failure.
Applicator/compile failures and empty test selections are rejected by the harness.

| Mutant | Weakened production bound | Named failure | Instrument / mutant exits |
| --- | --- | --- | --- |
| M1 | Generate falls back only to the exact legacy lock | TestGenerateRefusesStaleV050Authority | 0 / 1 |
| M2 | CI roots admit stale v0.5.0 current pin | TestCheckReleaseRootsRefusesDrift | 0 / 1 |
| M3 | VerifyRepository falls back to exact legacy lock | TestVerifyRepositoryRefusesStaleV050Lock | 0 / 1 |
| M4 | Current selects only the stale V050 projection | TestCurrentMatchesReviewedV060Catalog | 0 / 1 |
| M5 | Required ownership skips exactly Session selector | TestVerifyRepositoryRefusesMissingSelectorContract | 0 / 1 |
| M6 | Active go:generate token retained, inputs changed to V050 | TestGenerateDirectiveConsumesAdoptedAuthority | 0 / 1 |
| M7 | Decode admits only Config-4 through the V3 reader | TestLoadSupportsEveryPinnedConfigurationVersionAndTranslatesLegacyAtProductionEntry | 0 / 1 |
| M8 | CR cataloggen command token retained, inputs changed to V050 | TestConfiguredCRCatalogGateConsumesAdoptedAuthority | 0 / 1 |
| M9 | Assigned-scope disclosure admits only section 14.7.1 | TestAdoptedSectionsHavePendingOwnersAndRefuseRuntimeAdmission | 0 / 1 |
| M10 | Compatibility reader gate admits only unimplemented reader 4 | TestAssessCompatibilityPinsEveryPinnedVersionPairAtTheProductionEntry | 0 / 1 |
| M11 | Migration target gate admits only unsupported target 4 | TestMigrateRefusesEveryTargetOutsideTheUpgradeVocabulary/adopted_but_unimplemented_target | 0 / 1 |

M6's static TestGenerateDirectiveStaysRecognizedForm still exits 0, proving
that preserving the searched token does not preserve behavior. The actual CLI
behavioral suite fails. Byte-exact source restoration is asserted in finally
blocks; pristine and restored suite controls pass. The later M11 test row was
copied into the same isolated candidate and has its own instrument/suite log.
This is targeted adoption-gate coverage, not an exhaustive historical mutation
score or proof of selector/TLS runtime implementation.

Recorded anomalies: first M4 apply failed because the instrument interrupted
its source needle (exit 1, NOT a kill). Corrected ordering then re-proved the
entire battery. The initial ownership repin probe intentionally failed with the
new semantic digest (go test exit 1); after review-bound repinning it passed.
Initial optional local skill-path search found no Curator install inside this
worktree; the required control-root Go skill was read. One board query used an
unknown acceptanceCriteria field and was corrected to ac without mutations.
Prior unarchived mutation prose is not used as validation evidence.

## Per-command configured gate exits (first sweep)

| # | Command | Exit |
| ---: | --- | ---: |
| 1 | `test -z "$(gofmt -l $(git ls-files --cached --others --exclude-standard -- '*.go'))"` | 0 |
| 2 | `go build ./...` | 0 |
| 3 | `go vet ./...` | 0 |
| 4 | `go test ./... -count=1 -v` | 0 |
| 5 | `go test ./... -race -count=1` | 0 |
| 6 | `go test ./... -cover -count=1` | 0 |
| 7 | `go test ./internal/scalar -run=^$ -fuzz=^FuzzScalarProductionEntries$ -fuzztime=100x -parallel=1` | 0 |
| 8 | `go test ./internal/canonicaljson -run=^$ -fuzz=^FuzzCanonicalizeRoundTrip$ -fuzztime=100x -parallel=1` | 0 |
| 9 | `go test ./internal/canonicaljson -run=^$ -fuzz=^FuzzObjectIdentityRepresentationInvariant$ -fuzztime=100x -parallel=1` | 0 |
| 10 | `go test ./internal/canonicaljson -run=^$ -fuzz=^FuzzClosedIdentityShapeRefusal$ -fuzztime=100x -parallel=1` | 0 |
| 11 | `go test ./internal/canonicaljson -run=^$ -fuzz=^FuzzObservationEventRefusal$ -fuzztime=100x -parallel=1` | 0 |
| 12 | `go test ./internal/secconftest -run=^$ -fuzz=^FuzzCheckArgv$ -fuzztime=100x -parallel=1` | 0 |
| 13 | `go test ./internal/secconftest -run=^$ -fuzz=^FuzzCheckMemberPath$ -fuzztime=100x -parallel=1` | 0 |
| 14 | `go test ./internal/secconftest -run=^$ -fuzz=^FuzzIsEnvName$ -fuzztime=100x -parallel=1` | 0 |
| 15 | `go test ./internal/secconftest -run=^$ -fuzz=^FuzzRedactCorpus$ -fuzztime=100x -parallel=1` | 0 |
| 16 | `go test ./internal/secconftest -run=^$ -fuzz=^FuzzEscapeForTerminal$ -fuzztime=100x -parallel=1` | 0 |
| 17 | `go test ./internal/secconftest -run=^$ -fuzz=^FuzzRenderForTerminal$ -fuzztime=100x -parallel=1` | 0 |
| 18 | `go test ./internal/secconftest -run=^$ -fuzz=^FuzzGuardResolve$ -fuzztime=100x -parallel=1` | 0 |
| 19 | `go test ./internal/secconftest -run=^$ -fuzz=^FuzzDetectCaseCollision$ -fuzztime=100x -parallel=1` | 0 |
| 20 | `go run ./internal/traceability/cmd/tracecheck` | 0 |
| 21 | `go run ./internal/catalog/cmd/cataloggen -metadata internal/catalog/catalog.v0.6.0.json -contracts internal/specpin/v0.6.0.lock.json -output internal/catalog/catalog_gen.go -check` | 0 |
| 22 | `GOOS=linux GOARCH=amd64 go build ./...` | 0 |
| 23 | `GOOS=windows GOARCH=amd64 go build ./...` | 0 |
| 24 | `git ls-files -z '*.json' | xargs -0 -n1 python3 -c 'import json,sys;json.load(open(sys.argv[1]))'` | 0 |
| 25 | `task-board validate` | 0 |
| 26 | `git diff --check` | 0 |

## Final repository scope: 25 paths

- `LOGBOOK.md`
- `README.md`
- `internal/catalog/catalog.go`
- `internal/catalog/catalog_gen.go`
- `internal/catalog/catalog_test.go`
- `internal/catalog/cmd/cataloggen/main_test.go`
- `internal/cataloggen/generate.go`
- `internal/cataloggen/generate_test.go`
- `internal/cigate/contracts.go`
- `internal/cigate/contracts_test.go`
- `internal/config/compatibility_regression_test.go`
- `internal/config/migration_refusal_test.go`
- `internal/config/migration_test.go`
- `internal/config/schema_test.go`
- `internal/localstore/paths.go`
- `internal/specdoc/specdoc.go`
- `internal/specpin/pin.go`
- `internal/specpin/pin_test.go`
- `internal/traceability/cmd/tracecheck/main_test.go`
- `internal/traceability/traceability.go`
- `internal/traceability/traceability_test.go`
- `task-board.config.json`
- `internal/catalog/catalog.v0.6.0.json`
- `internal/traceability/adoption-v0.6.0.md`
- `internal/traceability/ownership.v0.6.0.json`

These are paths relative to the checkpoint, including untracked artifacts.
CR path counts may additionally contain task-board snapshot paths; the managed
CR record is authoritative for that broader count. The control-root config
edit is the same path/change, not an additional product file.
