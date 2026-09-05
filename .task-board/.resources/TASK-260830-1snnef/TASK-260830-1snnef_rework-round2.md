# Round-2 rework evidence (G1-G4, tooling note, P11)

## Fixes (test-only, no production or arm-table change)
- G2: `TestPlainErrorsLiveOnlyInTheirTwoFunctions` gained the GenDecl/ValueSpec loop (shared `isPlainErrorConstruction`, receiver-qualified names). Twin-hunt over `internal/terminalbackend` AST walks found no other twin.
- G1: attachability complement derived from the production state enum (`derivedParserVocabulary`); parked/active admitted, other six refused.
- G3: new internal `TestReplicationMembersAreExactlyTheClosedTable` (live map, both directions, size=64). External comment restated to its half with reverse cited.
- G4: new `TestDerivedVocabulariesAreExactlyTheAdmittedSets` (four vocabularies, both directions, size; pins `conformanceOperations` — the citation now holds). Refused samples restated as sampling bounds + extended with detached/detach/input_reopened/ssh_tunnel.
- Bounds recorded in Stated bounds block: W11 (DigestFile function-wide), W13 (bespoke error types), mismatchf single-arg assertion (derivation fatals on !=1 arg). P11 stale bound dropped in LOGBOOK.
- Tooling: `.temp/TASK-260830-1snnef-r3/probe3.py` restores every touched file (any extension) + JSON self-test + porcelain tree-identity check.

## Replay: 21/21 RED, 0 survived, 0 anchor-missing
- G1: quiescing/creating/absent/unavailable/stale_fenced/stopped widenings all RED via TestCheckStatusResultEnforcesLookupRules.
- G2: var errors.New / var fmt.Errorf / var func literal / FuncDecl control / unused var all RED via TestPlainErrorsLiveOnlyInTheirTwoFunctions.
- G3: add ax_pane_pid / init injection / native_pid reclassify / attach_credential promote all RED via TestReplicationMembersAreExactlyTheClosedTable.
- G4: detached / running / detach / input_reopened / ssh_tunnel all RED via TestDerivedVocabulariesAreExactlyTheAdmittedSets (+ sampling tests).
- F2 guard (admit unavailable for all ops) still RED. JSON-restore self-test OK. Tree porcelain+hashes identical before/after.

## Gates (all exit 0)
- `go test ./... -count=1`: 14/14 ok
- `go test ./internal/terminalbackend/ -race -count=1`: ok
- `go vet ./...`: exit 0; `gofmt -l .`: clean
- `tracecheck`: exit 0, acceptance_cases=76, clauses_discharged=17/403
- coverage: terminalbackend 90.0% statements

## Not re-run by me
- Inherited leaf-1/leaf-2 batteries: no production change and no test relaxation since round 2 (strictly stronger assertions), so no resurrection is constructible; accepted from round-2 attached evidence. Next review re-runs them.

## Handoff state
- Working tree holds the rework uncommitted per the brief (board checkpoints). Delta present: 9 modified paths + 3 untracked leaf files (conformance.go, conformance_test.go, refusal_arm_inventory_test.go).
