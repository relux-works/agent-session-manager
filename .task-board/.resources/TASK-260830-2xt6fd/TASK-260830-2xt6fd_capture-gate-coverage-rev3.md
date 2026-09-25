# Capture refusal-site coverage — TASK-260830-2xt6fd rev3

The assignment supplied no surface table; this is a brief gap, not a coverage waiver. The six-row acceptance map remains in `TASK-260830-2xt6fd_coverage-map-rev2.md` and measures 6 of 6 acceptance rows through their production call sites. This supplement covers the rework's coordinator refusal-site matrix.

## Measured census

`TestCaptureCoordinatorAdmissionGateCensus` parses all non-test Go files in `internal/tmuxserver`, starts at `(*Lifecycle).CaptureGitWorkspace`, follows package-local calls, and collects every reachable `captureUnavailable` call. It rejects non-literal detail arguments, duplicate details, production details without matrix rows, and matrix rows without production sites. The census uses a shared AST file set so declaration positions remain distinct. A control plant inserts an unlisted reachable refusal; another wraps an existing detail in `fmt.Sprint` so it ceases to be a literal. Both controls are expected to make the census fail.

Measured: **22 of 22** reachable literal refusal sites have exactly one production-entry test row, and **22 of 22** have an individual narrowing mutant killed by that row when run alone. Every row asserts literal `capability_unavailable` and its literal detail. Allowed controls are tested separately by `TestCaptureAdmissionGateValidControls` and the stop-point domain tests.

| Production refusal detail | Production-entry test | Narrowing mutant |
| --- | --- | --- |
| `lifecycle is required` | `TestCaptureAdmission_LifecycleRequired` | `N-capture-admits-nil-lifecycle` |
| `capture requires an exact Terminal Instance identity` | `TestCaptureAdmission_ExactStatusIdentityRequired` | `N-capture-admits-session-scoped-status` |
| `exact Terminal Instance status did not match` | `TestCaptureAdmission_OpeningStatusIdentityMatch` | `N-capture-admits-opening-status-nonmatch` |
| `capture is unsupported from Terminal Instance state "absent"` | `TestCaptureAdmission_InitialAbsent` | `N-capture-admits-absent-opening-state` |
| `capture is unsupported from Terminal Instance state "parked"` | `TestCaptureAdmission_InitialParked` | `N-capture-admits-parked` |
| `capture is unsupported from Terminal Instance state "active"` | `TestCaptureAdmission_InitialActive` | `N-capture-admits-active-opening-state` |
| `capture is unsupported from Terminal Instance state "stale_fenced"` | `TestCaptureAdmission_InitialStaleFenced` | `N-capture-admits-stale-fenced-opening-state` |
| `capture is unsupported from Terminal Instance state "unavailable"` | `TestCaptureAdmission_InitialUnavailable` | `N-capture-admits-unavailable-opening-state` |
| `Git assembly runner is required` | `TestCaptureAdmission_AssemblyRunnerRequired` | `N-capture-admits-nil-assembly-runner` |
| `stopped capture must not carry a boundary operation` | `TestCaptureAdmission_StoppedBoundaryBodyForbidden` | `N-capture-admits-stopped-boundary-body` |
| `stopped capture has no current incarnation record` | `TestCaptureAdmission_StoppedIncarnationRequired` | `N-capture-admits-stopped-without-incarnation` |
| `quiescing capture requires a provider safe-boundary operation` | `TestCaptureAdmission_QuiescingBoundaryRequired` | `N-capture-admits-quiescing-without-boundary` |
| `boundary operation identity differs from exact status` | `TestCaptureAdmission_BoundaryIdentityMatchesStatus` | `N-capture-admits-boundary-identity-drift` |
| `checkpoint boundary alone does not prove provider quiescence` | `TestCaptureAdmission_CheckpointOnlyProviderProof` | `N-capture-admits-checkpoint-proof` |
| `quiescing capture has no current incarnation record` | `TestCaptureAdmission_QuiescingIncarnationRequired` | `N-capture-admits-quiescing-without-incarnation` |
| `current-incarnation input-closure receipt is missing` | `TestCaptureAdmission_CurrentInputClosureReceiptRequired` | `N-capture-admits-foreign-quiesce-receipt` |
| `current-incarnation provider-boundary receipt is missing` | `TestCaptureAdmission_CurrentProviderBoundaryReceiptRequired` | `N-capture-admits-foreign-provider-boundary-receipt` |
| `closing Terminal Instance status did not match` | `TestCaptureAdmission_ClosingStatusIdentityMatch` | `N-capture-admits-closing-status-nonmatch` |
| `Terminal Instance incarnation changed during capture` | `TestCaptureAdmission_IncarnationStableAcrossCapture` | `N-capture-admits-changed-closing-incarnation` |
| `held capture left quiescing or stopped state` | `TestCaptureAdmission_FinalStateRemainsHeld` | `N-capture-admits-final-parked-state` |
| `stopped capture did not remain stopped` | `TestCaptureAdmission_StoppedRemainsStopped` | `N-capture-admits-stopped-to-quiescing-for-one-group` |
| `input-closure barrier no longer proves capture hold` | `TestCaptureAdmission_ClosureBarrierStillProven` | `N-capture-admits-unproven-barrier-for-one-group` |

## Test and mutant evidence

- Census baseline: `capture-census-01.log`, exit 0.
- Production-entry matrix plus stop-point state/transition checks: `capture-matrix-01.log`, exit 0.
- Each of the 22 site plants was run alone through the shipped harness and killed on the named row. First-pass logs are in `mutants-sites-01/`, `mutants-sites-02/`, and the retry folder `mutants-sites-retry-01/`; second-pass logs are in `mutants-sites-repeat-01/` and `mutants-sites-repeat-02/`. The first group-2 attempt hit a disk-full Go compiler failure for the closing-status plant; that is recorded as an error, not a kill. The single-plant retry killed it, and the repeat pass killed it again.
- `C-capture-census-unlisted-refusal-site` and `C-capture-census-nonliteral-detail` are control plants, not gate evidence. The harmless `C-control` is an applied survivor proving the harness reports survival rather than assuming every applied patch is killed.
- `N-capture-restores-with-boundary-token-preserved` attacks the stop-point owner-transition set while retaining its operation tokens; its named behavioral test is run separately.

Every harness verdict records the harness exit (0 for a valid verdict) and each raw behavior process exit. For KILLED rows, the behavior process exits 1 with a named test failure. A compile failure is ERROR, never KILLED.

## Bound and out-of-contract scope

The coordinator's metadata-shape checks on an owner-returned `wait-safe-boundary` `Mutation` are an internal typed-owner invariant: `Lifecycle.Execute` constructs this result from the landed owner. The capture entry still validates operation-body identity, proof kind, the stored current-incarnation quiesce receipt, provider-boundary receipt, parsed safe-boundary digest, closing status/incarnation, and held barrier. No synthetic owner seam was introduced to forge an impossible typed result.

The capture operation is a stop point under pinned SPEC v0.7.0 §4.C: it accepts only `quiescing` with current boundary evidence or `stopped` and never releases the instance. Returning to `active` in the same incarnation is out of contract because the lifecycle vocabulary is closed. The remaining out-of-contract rows and their acceptance-criteria bounds are enumerated in `TASK-260830-2xt6fd_coverage-map-rev2.md`.
