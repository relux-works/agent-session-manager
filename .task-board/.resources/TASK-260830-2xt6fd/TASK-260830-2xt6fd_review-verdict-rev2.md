# TASK-260830-2xt6fd review verdict — CR rev2 — CHANGES REQUESTED

Reviewer run RUN-260924-0ecc5a (claude-opus-5-5). Candidate tree f87de972ba2524a9c28b0040e770e58e8255909b, base 6d3bff9. I confirmed the live worktree equals the candidate tree with a scratch-index write-tree.

## Verdict
changes_requested → to-dev. All four rev1 findings are FIXED: my plants for proofkind, quiesceinc, barrier and stopstay are KILLED under `-run TestCapture`, and admitparked is KILLED too. The same class still survives at three adjacent refusal arms of `CaptureGitWorkspace`, however. The rework asked for a census enumerated MECHANICALLY from the coordinator's refusal sites, where any refusal site without a matrix row fails. `TestCaptureCoordinatorAdmissionGateCensus` instead classifies conditions through a hand-written 4-case switch (`captureAdmissionGateKind`), and returns "" (skip) for every other `captureUnavailable` site. The census denominator is therefore the four sites rev1 named, not the coordinator. Shape: census scoped by hand / finding closed on one path recurs on its twin.

## findings
```json
[
 {"id":"capture-hold-recheck-unmeasured","row":"held quiescence / source change during capture","invariant":"Capture must refuse when the Terminal Instance incarnation changes between the opening and closing reads (§12.3 9076-9078; a stop-point capture must hold one incarnation)","mechanism":"internal/tmuxserver/capture_git_workspace.go:159 `if !found || currentIncarnation != incarnation` — no test fails when this is narrowed","reproductions":["plant finalinc: `currentIncarnation != incarnation` → `currentIncarnation == \"\"`; `go test -count=1 -run TestCapture ./internal/tmuxserver/` → exit 0 SURVIVED (ev/plant-finalinc.log)"],"severity":"bypass","repeat-of":"rev1 capture-hold-recheck-unmeasured (adjacent site)"},
 {"id":"capture-hold-recheck-unmeasured","row":"held quiescence","invariant":"A provider-boundary receipt from another incarnation must not admit capture","mechanism":"capture_git_workspace.go:136 `boundaryReceipt.Incarnation != incarnation` arm is unmeasured","reproductions":["plant boundaryinc: `boundaryReceipt.Incarnation != incarnation` → `boundaryReceipt.Incarnation == \"\"`; same command → exit 0 SURVIVED (ev/plant-boundaryinc.log)"],"severity":"bypass","repeat-of":"rev1 capture-hold-recheck-unmeasured (adjacent site)"},
 {"id":"capture-stopped-boundary-unmeasured","row":"held quiescence","invariant":"A stopped capture must not carry a boundary operation","mechanism":"capture_git_workspace.go:67 `len(request.BoundaryBody) != 0` refusal is unmeasured","reproductions":["plant stoppedboundary: condition `&& false`; same command → exit 0 SURVIVED (ev/plant-stoppedboundary.log). This is an arm-delete, so it proves only that no test reaches the arm"],"severity":"bypass (unmeasured gate)","repeat-of":"rev1 capture-stopped-stays-stopped-unmeasured (class: unmeasured coordinator gate)"},
 {"id":"capture-gate-census-hand-scoped","row":"held quiescence (census)","invariant":"Every captureUnavailable refusal site in CaptureGitWorkspace needs a matrix row (rework-rev2 item 1)","mechanism":"capture_git_workspace_test.go captureAdmissionGateKind returns \"\" for unlisted conditions, and the census skips them; about 15 captureUnavailable sites exist and 4 are censused","reproductions":["the three survivors above pass the census unchanged; `go test -run TestCaptureCoordinatorAdmissionGateCensus` is green on each plant (included in the -run TestCapture runs)"],"severity":"bypass (instrument)","repeat-of":"none"}
]
```

## notes
- Control (a comment-only plant) SURVIVED as expected, so the harness can report a survivor.
- Held rev1 fixes: proofkind, quiesceinc, barrier, stopstay and admitparked are all KILLED (logs in ev/).
- The probe.go Dial change is now justified by §4.C and pinned by TestUnixDialerMissingDirectoryIsUnknown (per the producer; not re-planted).
- Importer outcome grid: NOT rerun (low budget). Stated as unverified. The full suite is green on the exact tree.

## Mandatory checks
| Check | Result |
|---|---|
| worktree == candidate tree | equal f87de97 |
| checkpoints 28ea2aa / 49d1d66 | Good signature |
| task-board.config.json vs base | byte-identical |
| go test ./... (live worktree), tmuxserver run separately with -timeout 9m | all green (ev/gotest-rest.log, ev/gotest-tmuxserver.log 294s) |
| go vet / GOOS=windows GOARCH=amd64 go vet / gofmt -l internal | clean |
| tracecheck | ok, clauses_discharged=106/597, equal to the README |
| README plant 106→107 | tracecheck tests red, exit 1 (ev/readme-plant.log) |
| digest re-derivation | registry_rederivation_test green in the full run (not re-derived independently) |
| determinism -count=3 | TestCapture* and TestAssembl* green |
| importer outcome grid | not rerun |

Measured: 5 of 8 coordinator refusal arms I planted are KILLED; 3 survive.

## Surface table
- held quiescence: broken (findings above)
- source change during capture: broken at the coordinator (finalinc). The gitsnap source-change gate held in rev1 and was not re-attacked.
- registry/traceability: held (tracecheck, README plant)
- manifest closure / path safety / byte identity: not-attacked (budget)
- composition/importer grid: not-attacked (budget)

## Rework scope (for the producer)
1. Make the census mechanical: every `captureUnavailable(...)` call inside `CaptureGitWorkspace` needs a matrix row (detail literal → test name); an unlisted detail fails the census. Control-plant it by adding a new refusal site.
2. Add reaching tests, with a narrowing killed ALONE, for: the closing incarnation change (line 159), the boundary receipt from another incarnation (line 136), and a stopped capture with a boundary body (line 67). Then sweep the remaining sites: identity mismatch, missing incarnation record, owner safe-boundary proof, and the closing status mismatch.
3. Nothing else changes; the rev1 fixes hold.
