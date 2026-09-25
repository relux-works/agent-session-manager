# TASK-260830-2xt6fd — capture quiescence blocker

Status: **blocked** pending an owner/API decision. No developer handoff was run.

## Decision needed

May capture leave a live Terminal Instance durably quiescing until stop/restore, or should the existing lifecycle owner provide a temporary, generation-bound capture hold/release with crash recovery?

Recommendation: extend the existing terminal lifecycle owner with durable scoped hold/release and recovery, then have the capture coordinator hold it across assembly and release/recover through that owner. The current lifecycle cannot release a quiescence barrier for the same incarnation. Using restore as release rotates the incarnation after stop and changes session lifecycle; manually unlocking tmux or adding a caller flag would bypass the owner and would be a forced fit.

## Evidence for the missing contract

- `internal/terminalbackend/conformance.go:106-131` closes the operation vocabulary at ten operations. It includes quiesce and wait-safe-boundary but no release operation.
- `internal/terminalbackend/conformance.go:135-150` closes side effects without an input-reopened/released effect.
- `internal/tmuxserver/lifecycle.go:215-232` dispatches quiesce, boundary wait, stop, terminate and restore, but no same-instance release.
- `internal/tmuxserver/backend.go:197-216` implements input closure with lock-session and detach-clients.
- `internal/tmuxserver/ops.go:232-253` refuses writable attach while the current-incarnation quiescence proof exists.
- `internal/tmuxserver/state.go:302-317` makes the closure proof authoritative for the current incarnation; reopening is tied to a superseding incarnation.
- `internal/tmuxserver/lifecycle_rev5_unix_test.go:153`, test `TestExecuteQuiesceStopRestoreReopensBarrier`, demonstrates reopening only after stop/restore and incarnation rotation.
- `internal/gitsnap/assembly.go:43-45,65-69` explicitly says a provisional assembly is not proof of quiescence and requires a real runtime coordinator. No production gitsnap caller supplies one.
- `internal/axpane/doc.go:96-102` describes process control as modeled caller input, not a live capture coordinator.

The landed owners provide quiesce, safe-boundary, and durable current-incarnation refusal. They do not provide capture-scoped release/recovery. No capture integration was added because the missing owner contract is exactly the blocker specified by the assignment.

## Specification crosswalk

Source specification: v0.5.0 at `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`. Current trunk spec: `internal/specdoc/SPEC.v0.7.0.md`, based on refreshed trunk `6d3bff999f75ed8585a7ef32074ab108f364ec51`. Extracted bodies of every cited section are byte-identical; only their line offsets changed. No cited clause moved, split, or changed.

| Clause | v0.5.0 lines | v0.7.0 lines | Mapping |
| --- | ---: | ---: | --- |
| §4.C | 1046–1337 | 1082–1373 | Same text, +36 |
| §4.1 | 1338–1366 | 1374–1402 | Same text, +36 |
| §4.2 | 1367–1413 | 1403–1449 | Same text, +36 |
| §10.2 | 4600–4653 | 4851–4904 | Same text, +251 |
| §10.3 | 4654–4681 | 4905–4932 | Same text, +251 |
| §10.4 | 4682–5216 | 4933–5467 | Same text, +251 |
| §12.1 | 7939–7965 | 9020–9046 | Same text, +1081 |
| §12.2 | 7966–7992 | 9047–9073 | Same text, +1081 |
| §12.3 | 7993–8027 | 9074–9108 | Same text, +1081 |

The v0.7 quiesce and safe-boundary rows are at §4.C lines 1207–1208; the held-capture requirement is at §12.3 lines 9076–9078.

## Measured coverage and bounds

The preserved assembly acceptance matrix measures **5 of 6 final-leaf rows with component entry drivers** through `AssembleProvisional`; held input/provider quiescence is **0 of 1** through a coordinated production capture entry. This is not full capture acceptance.

| Row | Production call site and named tests | Result |
| --- | --- | --- |
| Source change during capture | `AssembleProvisional → Capture → objects → Capture`; `TestAssemblySourceChangesAndRetry` | Component interval changes refuse. Change-and-revert and held runtime lifetime remain unproven. |
| Repository-local packs, inventories, raw/logical indexes | `AssembleProvisional → objects/rawIndex → RunInput/RunIsolated → contentState.blob → ObjectStore.PutBlob`; `TestAssemblyNormativeCorpus`, `TestAssemblyPackIndexAndBytes`, `TestAssemblyRecursivePacks`, `TestAssemblyFailedReadsAndCorruptObjects`, `TestAssemblyIndexCompatibility` | Component assembly covered; no runtime coordinator. |
| Workspace/root/child and blob manifest closure; safe paths | `AssembleProvisional → repository/seal → ValidateProvisional/checkGroupRecord/checkCapturedPaths`; `TestValidateProvisionalClosureAndDescriptors`, `TestAssemblyGroupRecordAgreement`, `TestAssemblyOptionsAndUnsupportedSources`, `TestAssemblyMultipleMembers`, `TestAssemblyRecursiveCWDAndConfigClosure`, `TestValidateProvisionalParentChildOverlap` | Component closure covered; group-history authority/conflict resolution remains an external owner input. |
| Byte identity, recursive dirty state, unsupported Git sources | `AssembleProvisional → existing Capture/content owner → per-repository object assembly`; `TestAssemblyNormativeCorpus`, `TestAssemblyRecursivePacks`, `TestAssemblyIndexCompatibility`, `TestAssemblyOptionsAndUnsupportedSources` | Component tests cover named cases. Shallow, partial-clone, and replacement-history sources refuse. |
| Durable mutation crash/retry | `AssembleProvisional/ObjectStore`; `TestAssemblyCrashAfterObjectsAndRetry`, `TestAssemblyPackIndexAndBytes` | Covered by the package suite; package command exit 0. Immutable unreferenced objects may remain after a crash. |
| Held provider/input quiescence over production capture | No coordinated production entry exists | **0 of 1**; no release or same-incarnation crash recovery path. |

These component results do not authorize a complete-capture capability claim. No doctor/capability evidence was rederived for the refreshed v0.7 registry. Materialization is also not claimed: the named deliverable is capture-time transfer-object assembly and workspace manifest closure; §12.3 materialization behavior is an explicit unproven boundary.

The brief supplies no surface table. This is a brief gap; no coverage-map handoff artifact is claimed. The project acceptance file is component evidence, not a substitute for that missing surface table or a full handoff map.

## Validation actually run

Each command was run as a standalone process with output redirected to a task-scoped log; the reported code is the real process exit code.

| Command | Exit | Evidence |
| --- | ---: | --- |
| `go test ./internal/gitsnap -count=1` | 0 | `go-test-gitsnap-01.log`; package summary `ok .../internal/gitsnap 189.652s` |
| `go test ./internal/terminstance -run '^TestExecuteQuiesceSuccess$|^TestExecuteOnlyQuiesceEntersQuiescing$|^TestExecuteWaitSuccess$' -count=1` | 0 | `go-test-terminstance-quiescence-01.log` |
| `go test ./internal/tmuxserver -run '^TestExecuteQuiesce$|^TestExecuteBoundary$|^TestExecuteQuiesceStopRestoreReopensBarrier$|^TestExecuteReadOnlyAttachPreservesQuiescedInput$|^TestObserveBoundaryBindsProviderRows$' -count=1` | 0 | `go-test-tmuxserver-quiescence-01.log` |
| `git diff --check` | 0 | `git-diff-check-01.log` |

Not run after refresh: full `go test ./...`, the mutation harness, `GOOS=windows GOARCH=amd64 go vet ./...`, importer outcome grid, lint, README/doctor and v0.7 ownership-registry update, `reviewedOwnershipCanonicalSHA256` rederivation, tracecheck and section-scoped runs. They remain pending because the required capture lifecycle integration cannot be implemented without the owner decision/API above. No checklist item for those commands should be checked.

The prior attached mutant report is historical pre-refresh evidence, not a rerun on this candidate. It measured 13 of 14 narrowing mutants killed. The known survivor is:

| Mutant | What it narrows | Named failing test | Bound |
| --- | --- | --- | --- |
| `N-assembly-closure-admits-one-missing-child` | Allows one missing-child visit through the closure visitor. | None; survivor | The unchanged repository-tree binding rejects the fixture, so this does not independently prove the visitor gate. |

The complete historical plant table and its batch exit codes are included as `prior-mutants.md` in the evidence archive. No mutant count is represented as current refreshed-candidate evidence.

## Refresh and candidate state

- Refresh target was trunk `6d3bff999f75ed8585a7ef32074ab108f364ec51`. Refreshed branch head is `49d1d660c11c4e1f1564d72ab7e28b3f6cd63203`.
- A pre-refresh candidate backup was created at `refs/backup/2xt6fd-pre-refresh-20260924T072226Z`, commit `d0e974b5451a028de596630e4bd553d634faa7a1`.
- Mechanical invariant audit `trunk-only-invariant-02.json` confirms all 2,116 trunk-only paths are blob-equal across worktree, index and candidate tree; preserved candidate paths were audited separately.
- The refresh tool's final invocation returned a session handle whose terminal shell exit was not collected. The resulting replay commits, signatures, and invariant audit are present; the refresh command itself is therefore reported as **exit unobserved**, not as exit 0.
- Candidate remains uncommitted; the index is clean. No handoff was run because the lifecycle contract prevents the assigned production acceptance criteria from being met.

## Handoff gates not met

The held-quiescence row is open, required suites/mutation checks and v0.7 registry/traceability runs remain unverified, and the surface-table brief gap is unresolved. This report records the current blocker and evidence; it does not claim review readiness.
