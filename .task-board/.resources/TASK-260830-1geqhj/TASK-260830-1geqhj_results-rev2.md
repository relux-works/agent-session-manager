# TASK-260830-1geqhj rev2 — rework results (ax pane enforcement wrapper)

Revision 2 of the implementation deliverable, reworked on trunk
`888ae3d` after `TASK-260830-1geqhj_review-verdict-rev1.md`
(changes_requested, 18 findings: 4 P1, 5 P2, 5 P3, 4 process).
Candidate UNCOMMITTED in the Story worktree
(`task-board/story/STORY-260830-ptxkqe`); no review acceptance or
main integration claimed.

## What changed

- P1-1 event authority: every `Emit` gates through
  `fencing.AuthorizeMutation` presenting the winning tuple;
  `EmitParked` additionally folds `session.parked` through
  `sessstate.Reduce` (§5.7: only materializing/stopped/failed/parked
  park); `Run` parks without chain evidence for remote,
  unverifiable, or unfolderable winners instead of manufacturing a
  §5.2 collision. Losing-lease profile events never drive launch
  (winner-closure derivation).
- P1-2 realm: the `AttestedServer` boolean is removed; background
  callers authorize only through the admitted
  `credential_capable_execution_realm` row bound to the host
  binding, probed build, and current generation via `Reconcile`;
  smoke targets bind build/generation/OS version. Cached-evidence
  rule is behavioral (`TestRealmCarriesNoCachedEvidence`), not a
  field census.
- P1-3 window: the §13.1 window closes on the winner's
  `HasCheckpoint`; in-window changed ops refuse in both modes,
  post-window new ops supersede atomically (`Store.Supersede`);
  restore-mode idempotency covered (RM4 killer). Stale
  `binding-*.tmp` swept age-gated (1 min).
- P1-4 checkpoint: admission requires the winning lease's
  checkpoint and binds the checkpoint's own `session_id`.
- P2: profile from the checkpoint closure (`DeriveForHeads`) on
  resume; journal source bound to the lease checkpoint;
  non-interactive create requires `headless_creation`; remote
  non-interactive parks per §4.2 step 5; RM1/RM2 killed by
  `TestRunTornLeasePropagates` and
  `TestDecideRemoteOwnerPrecedesMaterialization`.
- P3: staging sweep, TRACEABILITY mutant table fixed (duplicate
  `N-discovery` removed, 36 rows), `doc.go` division corrected
  (`AuthorizeMutation` composed, `AuthorizeInput` removed,
  fail-fast guards justified).

## AC coverage

41 of 41 AC rows driven through production entries by named
committed tests (`internal/axpane/TRACEABILITY.md`); 0 of 41 as a
public CLI surface (explicit caller-integration bound). Six
previously unfaithful rows re-pinned:

- Bootstrap idempotency → `TestDecidePostWindowNewOperationLaunches`,
  `TestRunPostWindowSupersedes`, `TestDecideRestoreChangedOperationRefuses`
- Checkpoint admission → `TestDecideCheckpointNotLeaseParks`,
  `TestDecideForeignCheckpointParks`
- Profile source → `TestDecideResumeProfileUsesClosure`
  (resume mode with checkpoint), `TestLosingLeaseProfileEventIgnored`
- Background realm → `TestDecideRealmBindingMismatchRefuses`,
  `TestRealmCarriesNoCachedEvidence` (behavioral)
- Cached evidence → `TestRealmCarriesNoCachedEvidence`
  (cached smoke without realm refuses; stale refuses; admitted launches)
- `session.parked` authorship → `TestEmitUnderLosingLeaseRefuses`,
  `TestEmitParkedUnfoldableRefuses`,
  `TestRunParkedUnfoldableSkipsEmission`,
  `TestRunTakeoverOfferEmitsParked` (no remote emission)

Plus the step-5 label fix (`takeover_offer_noninteractive` now
parks) and `TestDecideNonInteractiveRequiresHeadless`,
`TestDecideMaterializationStaleParks`,
`TestDecideSmokeTargetBindingRefuses`, `TestIsFencingRefusal`,
`TestBindSweepsStaleStaging`, `TestRunTornLeasePropagates`,
`TestDecideRemoteOwnerPrecedesMaterialization`,
`TestEmitParkedVocabularyLoop/refuses_*`.

## Validation (27 commands, all green)

- `go test ./internal/axpane/ -count=1` → ok (~7s)
- Coverage → 79.3% statements
- Package subsets (decide/realm/launch, bind/status, observe/chain/load,
  run, events, conformance) → ok
- `go vet ./internal/axpane/` → clean; `gofmt -l` → clean
- `CGO_ENABLED=0` + `AXT_*` env → ok
- 8 owner packages → ok
- `go test ./...` → all green; `go vet ./...` → clean
- Narrowing battery → 36/36 (35 KILLED + control)
- Crash/hook/concurrent → green (real SIGKILL)
- Conformance matrix → 27 clauses mapped
- Reviewer RM1/RM2/RM4 → all KILLED by the named tests above

Logs, coverage report, and mutant log are in
`TASK-260830-1geqhj_producer-evidence-rev2.tar.gz`
(sha256 `c3254121a2a5b36dcedd979b4a0fab67601e7c71d1e61513ed1f9895f616567a`).
No new `ax` command, `doctor` result, or runtime capability is
claimed; bounds live in TRACEABILITY.md and the conformance matrix.
