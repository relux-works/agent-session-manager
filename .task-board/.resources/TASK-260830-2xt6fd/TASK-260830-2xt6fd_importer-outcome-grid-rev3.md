# Importer outcome grid — TASK-260830-2xt6fd rev3 addendum

The base-to-Story outcome grid is preserved in `TASK-260830-2xt6fd_importer-outcome-grid-rev2.md`. It compares base checkpoint `49d1d660c11c4e1f1564d72ab7e28b3f6cd63203` with the refreshed Story candidate and names the existing-content, new assembly/coordinator, dialer, `provhost`, and traceability outcomes.

## Rev3 importer identity

`go list -f '{{.ImportPath}}|{{join .Imports ","}}' ./...` was rerun on the rev3 candidate (exit 0). Its complete importer graph is byte-identical to the rev2 candidate graph (`importers-candidate-rev3.log`; `importer-census-rev3-delta.log`, empty, diff exit 0). Thus rev3 adds no importer, removes no importer, and changes no package boundary. The new coordinator remains the candidate's first production caller of `gitsnap.AssembleProvisional`; its production entry remains absent at the base.

## `(package, production entry, input)` movement

| Package, entry, input | Base outcome | Rev3 candidate outcome | Movement |
| --- | --- | --- | --- |
| `internal/gitsnap`, `Capture`, shared content, symlink and digest-race inputs | Existing entry; all four shared fixtures green | Same four fixtures remain green in the attached rev2 comparison; no rev3 files changed in `gitsnap` | No moved class. The current full `tmuxserver` test does not substitute for component tests; the exact unchanged `gitsnap` outcome evidence remains attached. |
| `internal/gitsnap`, `AssembleProvisional`, packs, raw/logical indexes, descriptors and workspace closure | Production entry absent | Complete provisional assembly is exercised by the existing candidate assembly suite and narrowing evidence | New class; no baseline result is inferred from an absent entry. |
| `internal/tmuxserver`, `(*Lifecycle).CaptureGitWorkspace`, held state and same-incarnation closure | Production entry absent | Full current `internal/tmuxserver` suite passes on rev3 (`tmuxserver-full-rev3-01.log`, exit 0, 454.500s); the new 22-row matrix and each named narrowing are green/killed | New production entry. The gate matrix was extended in rev3; this did not change the importer graph. |
| `internal/tmuxserver`, capture invalid states, proof kind, receipt incarnation and closing recheck | No baseline entry | Each of the 22 refusal inputs reaches its literal code/detail through `CaptureGitWorkspace`; every matching one-input narrowing is killed alone twice | New refusal classes at the new production entry; no old importer outcome moved. |
| `internal/tmuxserver`, closing `wait-safe-boundary` owner result shape | No baseline entry | The landed `Lifecycle.Execute` typed owner supplies this result; capture validates its digest and persisted same-incarnation boundary receipt. The removed metadata-shape refusal cannot be forged through the production capture request | Explicit bound: no synthetic owner seam was added to manufacture an impossible typed result. |
| `internal/tmuxserver`, `UnixDialer.Dial`, ENOENT beneath a missing parent | Baseline classified every `ENOENT` as stale | Rev2 moved only this exact input to `unknown`; rev3 did not touch the dialer | Existing moved class retained; proof remains in the attached rev2 importer grid and dialer test evidence. |
| `internal/provhost`, provider identity admission | Existing entry and suite | Rev3 does not change `provhost`; its rev2 base/candidate package suites pass | No moved class. |
| `internal/traceability`, ownership registry / `tracecheck` | Earlier registry counts 160 cases / 70 bindings / 81 clauses | Rev2 story-final registry counts 163 cases / 73 bindings / 106 clauses; rev3 registry bytes are unchanged and current tracecheck remains green | Expected story registry movement only; no rev3 movement. |

The rev2 importer grid provides the base/candidate logs and runtime outcome evidence for unchanged classes. Rev3-specific evidence is the fresh importer census identity comparison, full affected-package run, 22-site production-entry matrix, narrowing logs and current full tracecheck. No absence is treated as a refusal outcome.
