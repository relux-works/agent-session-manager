# TASK-260830-2xt6fd review verdict — CR rev5 — CHANGES REQUESTED

Reviewer run RUN-260925-d98623 (claude-opus-5-5, low). Candidate tree 60d988f86ee2097cdc41ee723768adf04471f02c, base 5b7876b. The live worktree equals the candidate tree (scratch-index write-tree). Commits 365fcd9..999d938 are all signed (%G?=G).

## Verdict
changes_requested → to-dev. The rev2 findings are FIXED. The census is now mechanical, and my planted extra refusal site reddens it. The three rev2 survivors (finalinc, boundaryinc, stoppedboundary) are KILLED, each by its named row run ALONE. The same class survives one level down, though. The census is keyed per refusal DETAIL, and two of the details guard 4-clause compound conditions (capture_git_workspace.go:121 and :146). Each matrix row breaks only one clause: Incarnation, plus InputClosedAt for the quiesce receipt. The other clauses can be narrowed and the whole `-run TestCapture` suite stays green. Shape: a rule pinned along one axis; a site census is not a clause census.

## findings
```json
[
 {"id":"capture-receipt-clause-unmeasured","row":"held quiescence","invariant":"The input-closure and provider-boundary receipts admit capture only when the Operation is exactly quiesce-input / wait-safe-boundary and the observation timestamp is present (§4.C operation table 1198-1215; §12.3 9076-9078)","mechanism":"internal/tmuxserver/capture_git_workspace.go:121 (quiesce.Operation clause) and :146 (boundaryReceipt.Operation and BoundaryObservedAt clauses). The matrix rows TestCaptureAdmission_CurrentInputClosureReceiptRequired / _CurrentProviderBoundaryReceiptRequired break only Incarnation","reproductions":["plant quiesceop: `quiesce.Operation != \"quiesce-input\"` → `(quiesce.Operation != \"quiesce-input\" && quiesce.Operation != \"wait-safe-boundary\")` (narrowing: admits one wrong operation). `go test -count=1 -run '^TestCaptureAdmission_CurrentInputClosureReceiptRequired$' ./internal/tmuxserver/` exit 0, and `-run TestCapture` exit 0 → SURVIVED (ev/plant-quiesceop.log, plant-quiesceop-all.log)","plant boundaryop: `boundaryReceipt.Operation != \"wait-safe-boundary\"` → also admits \"quiesce-input\"; `-run TestCapture` exit 0 SURVIVED (ev/plant-boundaryop.log)","plant observedat: `|| boundaryReceipt.BoundaryObservedAt == \"\" ||` → `|| false ||` (arm-delete); `-run TestCapture` exit 0 SURVIVED (ev/plant-observedat.log)"],"severity":"bypass","repeat-of":"rev2 capture-hold-recheck-unmeasured (same gates, adjacent clauses)"}
]
```

## notes
- By contrast, the closedat plant (the InputClosedAt clause at :121) is KILLED by `-run TestCapture` (TestCaptureProviderProofRequirements).
- Applied control (a comment-only change) SURVIVED, so the harness reports survivors.
- The planted extra `captureUnavailable("review planted site")` is KILLED by TestCaptureCoordinatorAdmissionGateCensus run alone. I did not plant a non-literal detail.
- Reachability: a wrong-Operation record under the receipt key needs a corrupted or foreign state-store record. The gate is still defensive production code that the brief requires to be measured, so it counts as a finding and not a note.

## Mandatory checks
| Check | Result |
|---|---|
| worktree == candidate tree | equal 60d988f |
| signatures | all G |
| task-board.config.json vs base | byte-identical |
| go test (all packages except tmuxserver) | exit 0 (ev/gotest-rest.log) |
| go test ./internal/tmuxserver -timeout 9m | exit 0, 140s (ev/gotest-tmuxserver.log) |
| go vet / GOOS=windows GOARCH=amd64 go vet ./... | exit 0 / exit 0 |
| gofmt -l internal | clean |
| tracecheck | ok, clauses_discharged=113/622, equal to README:3763 |
| README coverage plant | NOT run (low budget) |
| digest re-derivation | registry_rederivation_test green in the full suite; not re-derived independently |
| determinism -count=3 | NOT run |
| importer outcome grid | NOT rerun (budget); unverified |

Measured: 7 of 10 coordinator narrowings/arm-deletes I planted were KILLED (finalinc, boundaryinc, stoppedboundary, proofkind, stopstay, closedat, census site); 3 SURVIVED (quiesceop, boundaryop, observedat).

## Surface table
- held quiescence: broken (capture-receipt-clause-unmeasured)
- source change during capture (coordinator incarnation recheck): held (finalinc KILLED)
- census instrument: held (a planted site is caught)
- registry/traceability: held (tracecheck == README); plant not run
- manifest closure / path safety / byte identity: not-attacked (budget; held in earlier rounds)
- composition/importer grid: not-attacked (budget)

## Rework scope (for the producer)
1. Extend the matrix from per-site to per-CLAUSE. Every disjunct of the :121 and :146 conditions gets its own row: a wrong Operation (use the sibling operation name), an empty timestamp, and the other incarnation. Each row asserts the literal code+detail, and each has a narrowing KILLED by that row run alone.
2. Make the census count clauses: for every `captureUnavailable` site guarded by an `||` chain, require one row per disjunct.
3. Nothing else changes.
