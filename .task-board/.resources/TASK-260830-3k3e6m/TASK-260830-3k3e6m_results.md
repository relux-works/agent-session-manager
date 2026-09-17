# TASK-260830-3k3e6m results: implement-durable-operation-journal

Status: ready for review. The candidate is left UNCOMMITTED in the
managed Story worktree for the handoff to snapshot.

## Outcome

`internal/matjournal` implements the Section 10.6 Materialization
Journal 2.0.0 recovery contract: prepare-bound IDs and canonical
request digests, the phase machine with token/bridge/authority rules,
the no-replace prepare receipt, the managed-replica marker validator
and destination classifier, and the status-first `Recover` evaluator
that classifies every CR-MAT-01..08 boundary into exactly one Section
13.13 outcome with durable parked and rollback evidence.

## AC coverage: 12 of 12 rows driven

Every row is driven through the named production entry by a named
committed test.

| # | AC row | Production call site | Test |
|---|---|---|---|
| 1 | Journal document created, read, transitioned, persisted | `Store.Create`, `Store.Get`, `Store.Transition` (`internal/matjournal/store.go`) | `TestCreatePersistsJournalAndReceipt`, `TestSpecExampleShapeDrivesThroughProduction` |
| 2 | Operation receipts persisted with ID binding | `Store.Create` receipt install (`store.go`) | `TestCreatePersistsJournalAndReceipt`, `TestCrashBetweenJournalAndReceiptReplays` |
| 3 | Phase transitions enforced | `Store.Transition` (`store.go`, `journal.go`) | `TestTransitionRefusesIllegalEdges`, `TestTransitionGuards`, `TestFullDormantWalkCommits` |
| 4 | Idempotency input digests; byte-unequal retry refuses | `Store.Create` (`replayCreateLocked`) | `TestCreateReplayAndConflict` |
| 5 | Status-first recovery classification | `Store.Recover` (`internal/matjournal/recover.go`) | `TestRecoverGatesPark`, `TestRecoverResumes`, `TestCrash*` |
| 6 | Parked/rollback outcomes durable with lease, IDs, identity, error | `Store.Recover` (`park`, `completeRollback`), `Store.Rollback` | `TestParkedEvidenceIsDurable`, `TestRecoverRollbackRequired`, `TestRollbackConverges` |
| 7 | Exact contract fixtures pass | `Update*`, `Transition`, `ValidateMarker`, `Recover` | `TestSpecExampleShapeDrivesThroughProduction`, `TestOpenedFixtureMatchesProduction`, `TestMarkerExampleValidates`, `TestClassifyDestination`, `TestRecoverFailureMatrixRows` |
| 8 | Negative/refusal cases pass | gates in `journal.go`, `store.go`, `recover.go` | `TestCreateRefusesMalformedClosure`, `TestProviderTokenAndIDRules`, `TestBoardNullabilityRefusals`, `TestAuthorityGuards`, `TestTerminalFreezes`, `TestGetRefusals`, `TestRecoverRefusesBadBoundary` |
| 9 | Crash/idempotency evidence | hooks + `Recover` | 8 `TestCrash*` boundaries with hook crashes + reopen + `Recover`, `TestCreateCrashChildSelfTerminates` (real SIGKILL), replay tests |
| 10 | No unsupported capability advertised | (no CLI/doctor/claim surface) | `cigate` capability-claim gate green; README sections use the no-claim closer |
| 11 | Receipt reconciliation | prepare receipt adopts the sessckpt discipline; distinct record documented | `internal/matjournal/TRACEABILITY.md` (Receipt reconciliation) |
| 12 | Story-close items: forged-Admit negative, README sections, ownership bindings + re-pin | `sessckpt.Admit`, README, `ownership.v0.6.0.json` | `TestAdmitRefusesSameEpochForeignLeaseHead`, `TestREADMEOwnershipFiguresAreDerivedFromTheMeasuredReport`, `tracecheck` |

## Gating behavior and killers

Every refusing gate ships a narrowing mutant (gate stays present,
weakened to admit exactly one member of the rejected class) plus the
named killer. Full battery: `python3
internal/matjournal/mutant_harness.py` — 11 KILLED, 1 control
SURVIVED, exit 0 (log: `logs/mutant-harness.log`).

| Gate (production site) | Mutant | Killer |
|---|---|---|
| Phase table (`legalTransitions`) | N-phase-table (admits staging-to-committed; literals preserved) | `TestTransitionRefusesIllegalEdges` |
| Bridge sub-table (`boardTransitions`) | N-board-subtable (admits not_started-to-opened; literals preserved) | `TestBoardNullabilityRefusals/subtable_refuses_skip` |
| Provider token rule (`checkProviderTransaction`) | N-provider-token (admits prepared-without-token) | `TestProviderTokenAndIDRules/prepared_requires_token` |
| Import triple (`requireImportTriple`) | N-board-import (admits expiry-less imports) | `TestBoardNullabilityRefusals/imported_triple_incomplete` |
| Bridge ID drift (`checkTaskBoardDrift`) | N-id-drift-adopt (admits adopt-ID drift) | `TestBoardNullabilityRefusals/operation_drift_refuses` |
| First-prepare conflict (`replayCreateLocked`) | N-create-first-prepare (admits different-operation same-body replay) | `TestCreateReplayAndConflict` |
| Pre-activation (`checkPreActivation`) | N-rollback-adopt (admits adopted rollback) | `TestTransitionGuards/rollback_refused_after_adopt` |
| Failed evidence (`checkTransitionGuards`) | N-failed-null (admits explicit-null error) | `TestTransitionGuards/failed_requires_error` |
| Commit marker (`checkTransitionGuards`) | N-commit-marker (admits marker-less commits with authorities) | `TestTransitionGuards/committed_workspace_requires_marker` |
| Crash-window uncertainty (`providerEffectUncertain`) | N-recover-uncertain-floor (admits CR-MAT-05..07 unknown probes) | `TestParkedEvidenceIsDurable` |
| Token grammar (`checkToken`) | N-token-length (admits exactly 3-byte tokens) | `TestProviderTokenAndIDRules/short_token_refuses` |

No gate inspects source text: every gate is a value, table, or grammar
comparison, so the token-preserving requirement holds structurally via
the table mutants. The byte-equality and digest checks are redundant
defenses (either refuses a moved body alone), so the idempotency
mutant targets the first-prepare check with a message-specific killer.

## Validation (all exit 0)

- `go test ./... -count=1`: 28 packages ok (`logs/suite.log`)
- `go test ./internal/matjournal/ -count=1 -v` (`logs/matjournal-tests.log`), `-cover`: 77.0%
- `go test ./internal/sessckpt/ -count=1` (`logs/sessckpt-tests.log`), `-cover`: 82.1%
- `go vet ./...`, `go build ./...`, `GOOS=windows go build ./...`, `gofmt -l` clean (`logs/gates.log`)
- Catalog `-check` and `tracecheck` pass: acceptance_cases=105, bindings=59, clauses 17/489 (`logs/gates.log`)

## Files changed (uncommitted in the Story worktree)

- `internal/matjournal/`: `doc.go`, `journal.go`, `marker.go`, `store.go`, `recover.go`, `journal_test.go`, `refusal_test.go`, `recover_test.go`, `crash_test.go`, `crash_unix_test.go`, `mutant_harness.py`, `TRACEABILITY.md` (new package)
- `internal/sessckpt/refusal_test.go`: `TestAdmitRefusesSameEpochForeignLeaseHead` (verdict section 4)
- `internal/traceability/ownership.v0.6.0.json`: 4 acceptance cases; section:5.4 rebound to `sessckpt.Capture`; new section:10.6, 13.12 (unmeasured: the scanner finds no clause line in the table-shaped section), 13.13 bindings
- `internal/traceability/traceability.go`: re-pinned `reviewedOwnershipCanonicalSHA256` to the reviewed projection
- `internal/traceability/traceability_test.go`, `internal/traceability/cmd/tracecheck/main_test.go`: measured counts
- `README.md`: Workspace Checkpoint Records + Materialization Journal sections; ownership figures 101/56 to 105/59
- `LOGBOOK.md`: task entry

## Stated bounds

- Journal/marker shapes validate in `matjournal`; the canonicaljson
  shape owner still refuses `materialization-journal` 2.0.0 and its
  census pins that registration. Replacing it is outside this leaf.
- The plan is an opaque digest input; only its authority routing view
  is cross-checked. Operation-predecessor and prior-classification
  rules stay with the plan owner.
- Provider/bridge/marker/lease/native states are modeled `Recover`
  inputs; staging removal and predecessor restoration are coordinator
  work; marker files belong to the replica owner.
- last_error encodes Structured Error 1.2.0, the earliest version
  registering every emitted code (`operation_uncertain` registers at
  1.2.0, not below).
- The phase and sub-state tables derive from the coordinator order,
  fixtures, and abort rules; each derived edge is documented in
  `TRACEABILITY.md` and the code comments name their grounding.
