# TASK-260830-8x76g1 recovery developer outcome

## Outcome

The fresh Story workspace now contains, in order, the exact supplied accepted-leaf tree and the exact supplied audit carry-forward overlay. The archive SHA-256 values reproduce the assignment values:

- accepted leaves: `9ae5e624addc7d954c391d96cab7c9f7aac3e6bcee58c76dd0cc95533d0eac9c`
- audit carry-forward: `df8d8fa712e0ce85bde4776dc1f83cef9d373593e76fd1145ce9d85b593ce4ee`

Checkpoint `828d63321a74bb6be1b322eebc5ea64794d7d914` verifies as signed (exit 0), and its tree is the disclosed `6b400c3c6b0718d4d08dc905ac46673bd4a87aa6`.

The restored implementation drives the composed production gate:

`CalculateObjectIdentity|VerifyObjectIdentity -> prepareObjectIdentity -> validateImmutableObjectShape`

Before identity calculation or attestation, that path enforces the candidate-local closed Blob Descriptor, BlobChunk, Transfer Manifest, ManifestEntry, WorkspaceSnapshot, Git embedded-object, extension-key/value, Unicode character-bound, ordering, uniqueness, nullability, tagged-union, chunk-coverage, and recursive submodule rules derived from pinned SPEC.md v0.5.0 Sections 1.6 and 10.1-10.4. Section 17.3 ownership remains limited to the immutable `works.relux.ax.migrated-from` contribution; README does not advertise migration publication, atomic reference advancement, rollback retention, doctor output, an `ax` command, or another unavailable runtime capability.

The pinned normative source was independently inspected at exact local checkout commit `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`. The existing task-scoped constraint enumeration matches the literal field tables and normative refusal cases, including TM-GIT-N2 stage 4.

## Negative and boundary evidence

The focused suite drives both public identity entries. A malformed candidate is also supplied with a correctly recomputed omit-self claim, so verification cannot pass merely because the claim is stale. Tests refuse unknown top-level and nested members, forged extension namespaces, invalid BlobChunk order/bounds/coverage, malformed digest/UUID/timestamp/path/ref/URL values, impossible calendar dates, stage 4, invalid tagged nullability, recursive submodule state/depth/count/cycle violations, sorting/case collisions, hardlink/symlink violations, and count drift.

Positive precision tests accept exact multibyte Unicode character limits and valid bare strings/host-only sanitized remote URLs, proving the validator does not replace under-validation with over-refusal. The cross-platform golden assigns one SHA-256 identity to LF and CRLF/property-reordered representations.

## Direct validation

Every command below ran as a standalone foreground process in this recovered workspace. Exit codes are the actual process results.

| Gate | Exit | Evidence |
| --- | ---: | --- |
| Supplied archive SHA-256 verification | 0 | `00-archive-hashes.log` |
| Accepted checkpoint signature and tree binding | 0 / 0 | `00-accepted-checkpoint-signature.log`, `00-accepted-checkpoint-tree.log` |
| Focused scalar/canonical production-entry tests | 0 | `01-focused-tests.log` |
| `go test ./... -count=1 -v` | 0 | `02-go-test-all.log` |
| `go test ./... -race -count=1` | 0 | `03-go-test-race.log` |
| `go test ./... -cover -count=1` | 0 | `04-go-test-cover.log` |
| Scalar fuzz, fixed `100x`, `parallel=1` | 0 | 34 baseline entries; `05-fuzz-scalar.log` |
| Canonical round-trip fuzz, fixed `100x`, `parallel=1` | 0 | 19 baseline entries; `06-fuzz-canonical-roundtrip.log` |
| Identity representation fuzz, fixed `100x`, `parallel=1` | 0 | 17 baseline entries; `07-fuzz-identity-invariant.log` |
| Closed-shape refusal fuzz, fixed `100x`, `parallel=1` | 0 | 50 baseline entries; `08-fuzz-closed-shapes.log` |
| Scoped tracecheck: 1.6, 10.1-10.4, 17.3 | 0 | `assigned_scopes=6`; `09-tracecheck-scoped.log` |
| Exact Section 17.3 tracecheck | 0 | `assigned_scopes=1`; `10-tracecheck-17.3.log` |
| Generated catalog consistency | 0 | `11-catalog-check.log` |
| `go build ./...` | 0 | `12-go-build.log` |
| `go vet ./...` | 0 | `13-go-vet.log` |
| Linux canonicaljson test cross-compile | 0 | `14-linux-test-compile.log` |
| Windows canonicaljson test cross-compile | 0 | `15-windows-test-compile.log` |
| Linux full build | 0 | `16-linux-build.log` |
| Windows full build | 0 | `17-windows-build.log` |
| Repository `gofmt` gate | 0 | `18-gofmt-check.log` |
| Tracked JSON parse gate (`pipefail` enabled) | 0 | `19-json-check.log` |
| `task-board validate` | 0 | `20-task-board-validate.log` |
| `git diff --check` | 0 | `21-git-diff-check.log` |

Coverage was 90.1% for `internal/scalar` and 81.4% for `internal/canonicaljson`; all other package results are retained in the coverage log. The AST-backed configuration test in the full suite requires every repository fuzz target to appear exactly once in `task-board.config.json` with `-fuzztime=100x -parallel=1`.

`task-board validate` reported 262 inherited `MISSING_ACTIVITY` diagnostics while returning exit 0. The current task is not named by those diagnostics.

## Scope boundary

The implementation is read-only and deterministic, so durable-state crash recovery is not applicable. Rules requiring referenced blob bytes, child manifests, raw Git pack/index bytes, isolated Git object databases, filesystem resolution, or publication/reference mutation remain external to one immutable identity candidate and are not claimed here.
