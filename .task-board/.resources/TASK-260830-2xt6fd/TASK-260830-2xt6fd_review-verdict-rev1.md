# TASK-260830-2xt6fd review verdict — CR rev1 — CHANGES REQUESTED

Reviewer run RUN-260924-a1d410 (claude-opus-5-5). Candidate tree eea17c5d287c0b2700a3c939643692eab44737e0, base 6d3bff9. The live worktree equals the candidate tree (checked with a scratch-index write-tree).

## Verdict
changes_requested → to-dev. Production looks right. The held-quiescence coordinator (`internal/tmuxserver/capture_git_workspace.go`) has four admission gates. Only one of them, the state gate, is pinned by a narrowing that its named test kills when run alone. The tests for the other three either stop at an earlier arm or do not reach the gate. The decision brief asked for a narrowing killed by its named test alone, so this row is unproven.

## findings
```json
[
 {"id":"capture-proof-kind-gate-unmeasured","row":"held quiescence (decision-rev1)","invariant":"A checkpoint-only boundary (ax_checkpoint_boundary) must not count as provider quiescence (§12.3, lines 9076-9078)","mechanism":"capture_git_workspace.go: the `request.Runner == nil` refusal comes BEFORE the ProviderProofKind check. TestCaptureRejectsCheckpointBoundaryAsProviderProof passes no Runner, so it refuses with 'Git assembly runner is required' at the same code. TestCaptureProviderProofRequirements also omits Runner and BoundaryBody. Shape: driver reaches a different gate than it names.","reproductions":["plant: add `&& ProviderProofKind != \"ax_checkpoint_boundary\"` to the proof-kind condition; `go test -count=1 -run 'TestCaptureRejectsCheckpointBoundaryAsProviderProof$' ./internal/tmuxserver/` → PASS (SURVIVED); a logged run shows err='capability_unavailable ... Git assembly runner is required' (ev/plant-checkpointproof2.log)","the same plant under -run TestCapture (every capture test) → SURVIVED"],"severity":"bypass (unmeasured gate: the suite admits a checkpoint-only proof without reddening)","repeat-of":"none"},
 {"id":"capture-hold-recheck-unmeasured","row":"held quiescence (decision-rev1)","invariant":"Capture must refuse if the input-closure barrier stops proving the hold, or if the input-closure receipt belongs to another incarnation","mechanism":"capture_git_workspace.go: the final `QuiesceBarrierProven` check and the `quiesce.Incarnation != incarnation` arm of the quiesce-receipt check have no test that fails when either is narrowed","reproductions":["plant `if !proven && false {` → go test -run TestCapture ./internal/tmuxserver/ → SURVIVED (ev/plant-barrier.log)","plant: delete ` || quiesce.Incarnation != incarnation` → SURVIVED (ev/plant-quiesceinc.log)"],"severity":"bypass (unmeasured gate)","repeat-of":"none"},
 {"id":"capture-stopped-stays-stopped-unmeasured","row":"held quiescence","invariant":"A stopped-source capture must still be stopped at the closing read","mechanism":"narrowing the final check to also admit quiescing survives","reproductions":["plant stopstay → SURVIVED (ev/plant-stopstay.log)"],"severity":"bypass (unmeasured gate; low reachability because stopped→quiescing has no owner transition)","repeat-of":"none"}
]
```

## notes (non-blocking)
- TestCaptureStopPointStateDomain has no positive row. Its quiescing and stopped rows also refuse, and every row refuses at the same code with the same op count. A parked-admitting plant SURVIVES it (ev/plant-parked.log). The state gate is still pinned by TestCaptureRejectsParkedEvenWithPriorQuiescenceReceipt (KILLED alone), and an active-admitting plant is KILLED. Make the domain test distinguish arms: assert the detail per state and include admitted rows.
- The Story rewrote the semantics of the landed `tmuxserver/probe.go` Dial (ENOENT is stale only when the parent directory exists). No spec citation is given for this; justify it or cite one.
- `sameObservation` ignores json.Marshal errors (two failed marshals compare equal).
- The coordinator's harness ships 2 capture mutants. The producer mutant table under-covers the coordinator's gates.

## Mandatory checks (my own instruments)
| Check | Result |
|---|---|
| Worktree == candidate tree | equal (eea17c5) |
| Replayed checkpoints signed | 28ea2aa, 49d1d66 Good signature |
| task-board.config.json vs base | byte-identical |
| Trunk-only paths blob-equal | every path the Story does not change matches the base (candidate diff vs 6d3bff9 lists only Story paths); the 19 overlap paths are Story edits |
| go test ./... (archive, no .task-board) | green except specpin (needs .task-board: PASSES in the live worktree) and tmuxserver (hit the go default 10m timeout under load; not completed) |
| tmuxserver capture tests | green |
| gitsnap/canonicaljson/traceability/provhost | green |
| go vet / GOOS=windows vet / gofmt | clean |
| tracecheck | ok, clauses_discharged=106/597, equal to README |
| README coverage plant (106→107) | pin reddens (exit 1) |
| Registry merge | all 2107 base leaf values present except the intentionally replaced §10.4 gap/declaration row |
| Source-change gate (gitsnap assembly) | held: plants sameobs-json and sameobs-all are KILLED alone by TestAssemblySourceChangesAndRetry; control SURVIVED |
| Controls | ctrl, ctrl2, ctrl-g applied and SURVIVED (the harness can report a survivor) |
| Importer outcome grid, -count=3 determinism, digest re-derivation beyond tracecheck/registry tests | NOT rerun (budget); stated as not verified |

Measured: 1 of 4 coordinator admission gates pinned by a narrowing killed alone. The gitsnap source-change gate is pinned.

## Surface table
- held quiescence: broken (the three findings above)
- source change during capture: held
- registry/traceability: held
- manifest closure / path safety / byte identity: not-attacked (budget)
- composition/importer grid: not-attacked (budget)

## Rework scope (for the producer)
1. Give every coordinator gate a test that reaches it: supply Runner, Assembly and a valid BoundaryBody so each refusal comes from the arm under test, and assert the refusal detail, not only the code. Ship narrowing mutants for: the proof kind admitting ax_checkpoint_boundary, the quiesce receipt from another incarnation, the final barrier no longer proven, and stopped→non-stopped. Each must be killed by its named test run alone.
2. Make TestCaptureStopPointStateDomain distinguish arms, with admitted rows for quiescing (with a boundary) and stopped.
3. Justify the change to probe.go Dial with a spec citation or a separate task.
