# Materialization Journal Traceability

Authority: relux-works/agent-session-manager-spec@v0.6.0, Section 10.6
(Materialization Journal), Section 10.5 (Materialization Plan,
referenced immutable input), Section 13.12 (failure matrix), and
Section 13.13 (crash/restart outcome gate). The task scope cites the
same section numbers of v0.5.0; the headings are retained in v0.6.0.

## Production entries

| Entry | File | Behavior |
|---|---|---|
| `Open` | `store.go` | Binds the owner-only materializations root; the root must be absolute. |
| `Create` | `store.go` | Binds both caller-stable IDs and the canonical request digest before any staging or bridge mutation; installs plan view, journal, and prepare receipt; byte-identical retries replay, moved bodies conflict. |
| `Get` | `store.go` | Returns byte-identical journal bytes, re-validated through the closed shape on every read. |
| `UpdateProvider` | `store.go` | Records one provider transaction state with the token rule and ID binding. |
| `UpdateTaskBoard` | `store.go` | Records one task-board state along the sub-state table with the nullability rules and bridge-key binding. |
| `UpdateAuthority` | `store.go` | Records one authority state along the sub-state table with the rollback-root rules. |
| `RecordProgress` | `store.go` | Merges per-blob chunk sets and whole-blob verifications in staging or validating. |
| `RecordError` | `store.go` | Records the redacted Structured Error without moving the phase. |
| `Transition` | `store.go` | Moves the phase along the table with the entry guards. |
| `Rollback` | `store.go` | Executes the allowed abort: rolling_back first, then converged rolled_back. |
| `Recover` | `recover.go` | Classifies one crashed materialization into exactly one Section 13.13 outcome, executing rollback or provable completions. |
| `ValidateMarker` | `marker.go` | Validates the closed marker shape and attests the omit-self identity. |
| `ClassifyDestination` | `marker.go` | Classifies one destination from its durable facts. |

## Conformance matrix

`n of m AC rows driven`: every row below is driven through the named
production entry by the named committed test.

### Journal shape and prepare (Section 10.6)

| Spec row | Production | Test |
|---|---|---|
| Journal closed 22-member shape, literals, null legs | `Create`, `Get` (`store.go`, `journal.go`) | `TestCreatePersistsJournalAndReceipt` |
| Normative prepared example members | `UpdateProvider`, `UpdateAuthority`, `UpdateTaskBoard`, `Transition` | `TestSpecExampleShapeDrivesThroughProduction` |
| MJ-TASK-BOARD-OPENED-POS verbatim fixture | `UpdateTaskBoard` | `TestOpenedFixtureMatchesProduction` |
| Import/open/adopt/resume operation-ID allocation with the materialization ID; no second manager | `Create` | `TestCreatePersistsJournalAndReceipt`, `TestCreateReplayAndConflict` |
| Same IDs plus byte-identical body replay; moved body refuses idempotency_mismatch (MJ-RPC-PREPARE-LOST) | `Create` (`replayCreateLocked`) | `TestCreateReplayAndConflict` |
| Second prepare for one materialization refuses | `Create` | `TestCreateReplayAndConflict` |
| Malformed closure members refuse the invalid class | `Create` (`buildCreate`) | `TestCreateRefusesMalformedClosure` |
| Unknown/absent journal and torn bytes | `Get` | `TestGetRefusals` |

### Phase machine (Section 10.6, derived table)

The pinned text names the phase enum and the coordinator order but no
explicit edge table; the table in `legalTransitions` derives one edge
per order step (prepare creates staging, transfer stages, validation
validates, commit returns prepared, finalize commits), one abort edge
per non-terminal phase (a valid staging transaction resumes or rolls
back), and one failure edge per non-terminal phase. The
committing-to-rolling_back edge additionally requires pre-activation
(byte rollback is forbidden after adopt). Each edge is proved by the
walk tests and the full illegal-pair battery.

| Spec row | Production | Test |
|---|---|---|
| Every legal edge admits | `Transition` | `TestSpecExampleShapeDrivesThroughProduction`, `TestFullDormantWalkCommits`, `TestRollbackConverges` |
| Every illegal pair refuses with the table message | `Transition` | `TestTransitionRefusesIllegalEdges` |
| Prepared requires provider prepared and opened bridge durable | `Transition` (`checkTransitionGuards`) | `TestTransitionGuards/prepared_requires_opened`, `prepared_requires_provider_prepared` |
| Committed requires converged sub-states, resumed/dormant bridge, authorities committed, exact destination marker for workspace | `Transition` | `TestTransitionGuards/committed_requires_convergence`, `committed_workspace_requires_marker`, `committed_rejects_drifting_marker` |
| Failed requires the explaining last_error | `Transition` | `TestTransitionGuards/failed_requires_error`, `TestLastErrorIsRedactedStructuredError` |
| Terminal phases freeze every entry | `update` | `TestTerminalFreezes`, `TestTransitionRefusesIllegalEdges` |

### Provider, bridge, authority rules (Sections 10.6, 7.5)

| Spec row | Production | Test |
|---|---|---|
| Prepared requires a token; every other state requires null | `UpdateProvider` | `TestProviderTokenAndIDRules` |
| Token base64url-256+ grammar (fixpoint, 32..512 bytes) | `checkToken` | `TestProviderTokenAndIDRules`, `TestTokenFixturesAreWellFormed` |
| Transaction authority closed shape and mat/tx/plan binding | `UpdateProvider` | `TestProviderTokenAndIDRules/authority_ids_bind` |
| Transaction ID stable; per-call operation ID may advance | `UpdateProvider` | `TestProviderTokenAndIDRules/transaction_drift_refuses`, `operation_id_may_advance` |
| Bridge null/state invariants per state | `UpdateTaskBoard` | `TestBoardNullabilityRefusals` |
| Bridge sub-state table incl. no adopted rollback | `UpdateTaskBoard` | `TestBoardNullabilityRefusals/subtable_*` |
| Dormant cleanup_after equals the consumed open expiry | `UpdateTaskBoard` | `TestBoardNullabilityRefusals/dormant_expiry_binds_consumed`, `TestFullDormantWalkCommits` |
| Failed preserves only one still-valid pair | `UpdateTaskBoard` | `TestBoardNullabilityRefusals/failed_*` |
| Bridge operation-ID drift refuses | `UpdateTaskBoard` (`checkTaskBoardDrift`) | `TestBoardNullabilityRefusals/operation_drift_refuses` |
| Authority sub-table, rollback-root rules, single provider backup | `UpdateAuthority` | `TestAuthorityGuards` |
| Per-blob chunk sets merge by union (MJ-MULTIBLOB-POS) | `RecordProgress` | `TestRecordProgressMergesPerBlob` |
| Progress closes after validation | `RecordProgress` | `TestRecordProgressMergesPerBlob` |
| Rollback converges tokens, states, and the terminal failure; refuses past activation and without a failure | `Rollback` | `TestRollbackConverges`, `TestTransitionGuards/rollback_refused_*` |

### Marker and destination (Section 10.6)

| Spec row | Production | Test |
|---|---|---|
| Normative marker example validates; marker_id recomputes | `ValidateMarker` | `TestMarkerExampleValidates` |
| Marker binding (replica, plan, checkpoint, materialization, predecessor) | `Transition`, `Recover` (`checkMarkerBinding`) | `TestTransitionGuards/committed_rejects_drifting_marker`, `TestRecoverCompletesCommit` |
| Destination classes incl. MARKER-* rows | `ClassifyDestination` | `TestClassifyDestination` |

### Recovery (Sections 13.12, 13.13)

| Spec row | Production | Test |
|---|---|---|
| CR-MAT-01..08, each with a real crash injection followed by `Recover` | `Recover` | `TestCrash*` (8 boundaries), `TestCreateCrashChildSelfTerminates` (SIGKILL) |
| MJ-CRASH-STAGE/PREPARE-LOST/COMMIT-LOST/ROLLBACK-LOST/MARKER-MISMATCH | `Recover` | `TestCrashAfterTransferResumes`, `TestCrashAfterProviderPreparePersistsToken`, `TestRecoverResumes/committing_retries_commit`, `TestRecoverCompletesCommit`, `TestRecoverRollbackRequired`, `TestRecoverGatesPark/marker_mismatch` |
| TB-TXN IMPORT/OPEN/ADOPT/RESUME-LOST, BINDING-MISMATCH, TOKEN-EXPIRED, PASSIVE | `Recover` | `TestRecoverResumes/bridge_*`, `TestRecoverGatesPark/binding_mismatch`, `TestRecoverRollbackRequired/token_expired_before_lease`, `TestFullDormantWalkCommits`, `TestRecoverCompletesCommit/dormant_finalize_closes` |
| 13.12 disk full / rename blocked park with remediation | `Recover` | `TestRecoverGatesPark/disk_full`, `rename_blocked` |
| 13.12 owner-resume lost finalizes under the same IDs | `Recover` | `TestRecoverFailureMatrixRows/owner_resume_lost_finalizes` |
| 13.12 operator interrupt retries the same operation | `Recover` | `TestRecoverFailureMatrixRows/operator_interrupt_retries_same_operation` |
| 13.12 import/open/adopt fail before the lease rolls back | `Recover` | `TestRecoverRollbackRequired/bridge_failed_before_lease`, `token_expired_before_lease` |
| 13.12 adopt fails after the lease stays the stopped owner | `Recover` | `TestRecoverFailureMatrixRows/adopt_fails_after_lease_stays_stopped_owner` |
| Rejection gates: moved/unknown lease, substitution, two live authorities, unfenced continuation, marker integrity, unknown decisive status, token/binding contradiction, torn progression | `Recover` | `TestRecoverGatesPark`, `TestRecoverTornJournalParks`, `TestRecoverContradictoryProgressionParks` |
| Terminal journals replay; park leaves them frozen | `Recover` | `TestRecoverTerminalReplays` |
| Parked/rollback evidence durable with lease, IDs, identity, error; never a fourth outcome | `Recover` (`park`, `completeRollback`) | `TestParkedEvidenceIsDurable`, `mustOutcome` (every recovery test) |

## Mutants

Run: `python3 internal/matjournal/mutant_harness.py`. All narrowing
mutants are KILLED; the harmless control SURVIVES. No gate inspects
source text: every gate is a value, table, or grammar comparison, so
the token-preserving requirement holds structurally — the table
mutants (N-phase-table, N-board-subtable) preserve every literal and
change only the edge set.

| Mutant | Weakening | Killer |
|---|---|---|
| N-phase-table | Admits staging-to-committed | `TestTransitionRefusesIllegalEdges` |
| N-board-subtable | Admits not_started-to-opened | `TestBoardNullabilityRefusals/subtable_refuses_skip` |
| N-provider-token | Admits prepared-without-token | `TestProviderTokenAndIDRules/prepared_requires_token` |
| N-board-import | Admits expiry-less imports | `TestBoardNullabilityRefusals/imported_triple_incomplete` |
| N-id-drift-adopt | Admits adopt-ID drift | `TestBoardNullabilityRefusals/operation_drift_refuses` |
| N-create-first-prepare | Admits different-operation same-body replay | `TestCreateReplayAndConflict` |
| N-rollback-adopt | Admits adopted rollback | `TestTransitionGuards/rollback_refused_after_adopt` |
| N-failed-null | Admits explicit-null explaining error | `TestTransitionGuards/failed_requires_error` |
| N-commit-marker | Admits marker-less commits on journals with authorities | `TestTransitionGuards/committed_workspace_requires_marker` |
| N-recover-uncertain-floor | Admits unrecorded-prepare unknown probes at CR-MAT-05..07 | `TestParkedEvidenceIsDurable` |
| N-token-length | Admits exactly 3-byte tokens | `TestProviderTokenAndIDRules/short_token_refuses` |
| C-doc-comment | Comment-only control | `TestTokenFixturesAreWellFormed` (SURVIVES) |

## Receipt reconciliation

The prepare receipt adopts the sessckpt operation-receipt discipline —
an operation-keyed, input-digest-compared, no-replace receipt installed
after the bytes it names — and is a distinct record because it binds
the materialize.prepare namespace (prepare operation, canonical
request body, journal) rather than a checkpoint capture (capture
operation, record bytes, checkpoint). The two receipts are never
interchangeable: different operations, different idempotency keys,
different digests, different result bindings. Sharing one shape would
merge the namespaces the idempotency rules keep apart.

## Stated bounds

- The journal and marker shapes validate in this package; the
  canonicaljson owner still registers materialization-journal 2.0.0
  (and materialization-plan 1.0.0) with
  rejectUnsupportedImmutableObjectShape, and its census and inventory
  suites pin that registration. Replacing it is outside this leaf.
- The plan is an opaque digest input; only its authority routing view
  (kind, platform, root path) is cross-checked. Operation-predecessor
  and prior-checkpoint classification rules stay with the plan owner.
- Status probes (provider, bridge, marker, lease, native) are modeled
  inputs. No Mesh RPC, provider plugin, bridge process, or lease
  arbitration is driven here.
- Filesystem staging removal and predecessor restoration are
  coordinator work; the journal records the converged states and the
  probe-verified closure facts.
- Marker history/current files belong to the replica owner; this
  package validates marker bytes and classifies destinations.
- The package adds no `ax` command, no `doctor` result, and no runtime
  capability claim.
