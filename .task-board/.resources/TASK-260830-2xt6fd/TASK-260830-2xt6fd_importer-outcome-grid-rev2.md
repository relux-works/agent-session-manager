# Importer outcome grid — TASK-260830-2xt6fd

## Compared revisions and environment

Base is the Story worktree's recorded checkpoint `49d1d660c11c4e1f1564d72ab7e28b3f6cd63203` (the source archive at `.temp/TASK-260830-2xt6fd/base`). Candidate is that same `HEAD` plus the current uncommitted worktree delta. Shared-input commands used the same task-scoped `HOME` and `TMPDIR`; their exact standalone command and full output are in the paired logs below.

The rev2 comparison was rerun by this producer after the reviewer requested
independent importer evidence. The host `go test ./... -count=1` candidate run
used default `t.TempDir` after a first run with a deep `TMPDIR` failed unrelated
path-sensitive fixtures. The dedicated base/candidate outcome processes below
used identical source, task-scoped home/temp roots, and test source where
instrumentation was needed.

## Key: `(package, production entry, input)`

| Package, entry, input | Base outcome | Candidate outcome | Class movement |
| --- | --- | --- | --- |
| `internal/gitsnap`, `Capture`, ordinary blob/chunk limit boundary (`TestContentLargeBlobChunksAndLimit`) | Exit 0; pass | Exit 0; pass | Existing content-byte and chunk path retained. |
| `internal/gitsnap`, `Capture`, uninitialized submodule with stray state (`TestContentSubmoduleUninitializedAndRefusals`) | Exit 0; pass | Exit 0; pass | Existing refusal class retained; candidate narrowing `N-submodule-admits-stray-uninitialized` is killed by this named test. |
| `internal/gitsnap`, `Capture`, symlink chain/excluded-target cases (`TestContentSymlinkChainsAndExcludedTargets`) | Exit 0; pass | Exit 0; pass | Existing path rules retained; candidate chain-boundary test adds generated 40/41 hop coverage at `Capture`. |
| `internal/gitsnap`, `Capture`, included content digest change and retry (`TestContentDigestRaceAndRetry`) | Exit 0; pass | Exit 0; pass | Persistent digest-change refusal retained; candidate `N-consistency-admits-five-byte-digest-change` is killed by this test. Source change-and-revert between reads remains an explicit bound protected by held quiescence. |
| `internal/gitsnap`, `AssembleProvisional`, repository-local object pack and exact reachable inventory | Entry absent at base; no baseline outcome to infer | Candidate behavior and corruption cases pass in the repository suite; `N-assembly-inventory-admits-four-object-metadata-mismatch` is killed by `TestAssemblyFailedReadsAndCorruptObjects` | New class: packs are produced, imported into an isolated fresh repository, checked against exact reachable object/type/size inventory, and stored as blob descriptors. |
| `internal/gitsnap`, `AssembleProvisional`, raw Git index bytes vs logical entries | Entry absent at base | Candidate pack/index, format, raw checksum and disagreement tests pass; `N-assembly-raw-index-admits-two-entry-disagreement` and `N-assembly-checksum-admits-last-bit-flip` are killed by their named tests | New class: preserve raw index bytes and prove they describe the exact logical entry inventory. |
| `internal/gitsnap`, `ValidateProvisional` through `AssembleProvisional`, descriptors and root/member/child closure | Entry absent at base | Candidate missing/wrong descriptor, unbound tree, group mismatch, config and parent/child overlap cases pass; associated narrowings are killed except the explicit missing-child survivor | New class: immutable manifests/descriptors are connected through root and recursively captured member closure. Exact survivor limitation is recorded in `coverage-map.md`. |
| `internal/tmuxserver`, `(*Lifecycle).CaptureGitWorkspace`, state × phase, provider/input receipts, final observation and crash/retry | Production entry absent at base | Candidate `^TestCapture` suite passes, exit 0 (`capture-tests-final-04.log`); parked-state and restore-transition plants are killed | New class: production lifecycle boundary holds capture at `quiescing` with same-incarnation receipts or `stopped`; no release transition is introduced. |
| `internal/tmuxserver`, `(*Lifecycle).CaptureGitWorkspace`, proof-kind, quiesce-receipt incarnation, close-time barrier and stopped-state change | Production entry absent at base | Candidate valid-control requests admitted from both allowed states; each invalid-input row refuses with literal code and detail in its own test, each exit 0 (`capture-rework-determinism-02.log`, repeated three times) | New class: four refusal gates are witnessed independently after all other request inputs are valid; each corresponding narrowing dies when its named test runs alone. |
| `internal/tmuxserver`, `UnixDialer.Dial`, ENOENT with absent containing directory | Base source classifies every `ENOENT` as stale (`git show 49d1d660:internal/tmuxserver/probe.go`); a separate baseline process run for this one input is not claimed | `TestUnixDialerMissingDirectoryIsUnknown` passes in the final full tmuxserver package run; `N-unix-dialer-missing-parent-admits-stale` is killed by that test alone | Moved class: a missing parent or failed parent read remains unknown; only a missing socket under an existing directory is stale. |

## Importer census

`go list` exits 0 at both revisions. The baseline importer output is
`imports-base-01.log`: `internal/tmuxserver` has no `gitsnap` package import.
Candidate output is `imports-candidate-01.log`: the production package now
imports `internal/gitsnap` through
`internal/tmuxserver/capture_git_workspace.go`. `AssembleProvisional` is called
by `(*Lifecycle).CaptureGitWorkspace`; direct component tests remain separate
and do not stand in for coordinator evidence.

The shared-input test command was:

```text
HOME=<task-scoped HOME> TMPDIR=<task-scoped TMPDIR> go test ./internal/gitsnap -run '^(TestContentDigestRaceAndRetry|TestContentSymlinkChainsAndExcludedTargets|TestContentSubmoduleUninitializedAndRefusals|TestContentLargeBlobChunksAndLimit)$' -count=1 -v
```

Base exit: **0**, log `base-content-tests-02.log`. Candidate exit: **0**, log
`candidate-content-tests-02.log`. Four of four shared content tests passed on
both revisions. New assembly/coordinator inputs are marked “entry absent” at
base because the checkpoint has neither production entry; this is not treated
as evidence of a baseline refusal.

## Rev2 independent importer rerun

The direct import census was regenerated from `go list -f
'{{.ImportPath}}|{{join .Imports ","}}' ./...` in both trees. The complete
per-package outputs are `importers-base-02.log` and
`importers-candidate-02.log`; their scoped comparison is
`importer-census-02.txt`. Direct importers did not move for `provhost`,
`tmuxserver`, or `traceability`. `internal/gitsnap` gained its first direct
production importer, `internal/tmuxserver`, through
`capture_git_workspace.go`; this is the new coordinator edge, not an existing
consumer whose behavior changed.

| `(package, entry, input)` | Base result | Candidate result | Moved class / evidence |
| --- | --- | --- | --- |
| `internal/gitsnap`, `Capture`, four shared content fixtures | Exit 0; 4/4 pass | Exit 0; 4/4 pass | No moved class; `importer-shared-content-base-01.log` and `importer-shared-content-candidate-01.log`. |
| `internal/provhost`, full package suite | Exit 0 | Exit 0 | No moved classes among existing provider-host importers; logs `importer-provhost-base-01.log` / `importer-provhost-candidate-01.log`. |
| `internal/traceability`, full registry and verifier suite | Exit 0 | Exit 0 | No moved classes for the existing tracecheck importer; logs `importer-traceability-base-01.log` / `importer-traceability-candidate-01.log`. Candidate includes the three-leaf registry edge cases. |
| `internal/tmuxserver`, `UnixDialer.Dial`, ENOENT under missing parent | Exit 1 expected-red: observed `stale`, literal `want unknown` failed | Exit 0: observed `unknown` | Exact moved class: failed parent observation is unknown instead of stale. Identical temporary test source was injected in both trees and removed afterwards; logs `importer-dial-baseline-01.log` / `importer-dial-candidate-01.log`. |
| `internal/provhost`, `CheckIdentity`, spec Section 5.5 record with a valid-format but mismatched `record_id` | Exit 0; generic `VerifyObjectIdentity` rejects the self-digest and shape-only `CheckIdentity` admits | Exit 0; same assertions and result | No outcome movement. Identical temporary production-entry test source was injected in both trees and removed; logs `importer-checkidentity-base-04.log` / `importer-checkidentity-candidate-04.log`. The earlier malformed temporary source attempts exited 1 at package setup (`importer-checkidentity-*-03.log`); they did not run tests and are not counted as outcomes. |
| `internal/traceability/cmd/tracecheck`, `go run`, complete registry | Exit 0; 64 contracts, 160 acceptance cases, 70 bindings, 81/585 clauses discharged | Exit 0; 64 contracts, 163 acceptance cases, 73 bindings, 106/597 clauses discharged | Expected registry and evidence movement for the three story leaves: +3 acceptance cases, +3 bindings and +25 discharged clauses; no tracecheck refusal/output-class regression. Logs `tracecheck-importer-base-03.log` / `tracecheck-importer-candidate-03.log`. |

This comparison is keyed by production package/entry/input; absent baseline
entries are not assigned inferred outcomes. The complete candidate suite is
recorded separately in `go-test-all-rework-final-02.log` (exit 0).
