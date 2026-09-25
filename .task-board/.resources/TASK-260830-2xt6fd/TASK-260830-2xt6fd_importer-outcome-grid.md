# Importer outcome grid — TASK-260830-2xt6fd

## Compared revisions and environment

Base is the Story worktree's recorded checkpoint `49d1d660c11c4e1f1564d72ab7e28b3f6cd63203` (the source archive at `.temp/TASK-260830-2xt6fd/base`). Candidate is that same `HEAD` plus the current uncommitted worktree delta. Shared-input commands used the same task-scoped `HOME` and `TMPDIR`; their exact standalone command and full output are in the paired logs below.

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
