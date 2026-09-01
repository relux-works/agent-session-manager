# TASK-260830-8x76g1 republish results

## Outcome

The revision 6 common-types/canonical-identity candidate remains intact on every non-Finder path. The production gate is still `prepareObjectIdentity -> validateImmutableObjectShape`, shared by `CalculateObjectIdentity` and `VerifyObjectIdentity`. The validator registry remains total over the generated self-identity catalog: supported Section 10.1-10.4 shapes validate recursively, while registered schemas without a complete in-scope validator refuse explicitly.

This run removed a recurring root `.DS_Store` that had drifted the Change Request candidate after validation. Because Finder recreated it during this run, `.DS_Store` was added to the existing project-local `.gitignore`; `.temp/.DS_Store` was already covered by `.temp/`. No production Go, test, fixture, catalog, traceability, or README bytes changed in this run.

## Source binding

- Accepted leaves archive: `sha256:9ae5e624addc7d954c391d96cab7c9f7aac3e6bcee58c76dd0cc95533d0eac9c` (matches the attached digest).
- Audit carry-forward archive: `sha256:df8d8fa712e0ce85bde4776dc1f83cef9d373593e76fd1145ce9d85b593ce4ee` (matches the attached digest).
- `git apply --reverse --check` against recorded CR revision 6 truthfully exited 1 because that recorded patch contains the drifted `.DS_Store`.
- Repeating the binding check with only `.DS_Store` excluded exited 0, proving every non-Finder path is byte-compatible with recorded CR revision 6 before the one-line ignore rule.

## Validation

All commands were run directly as standalone foreground processes. Exit codes:

| Gate | Exit |
| --- | ---: |
| configured gofmt check | 0 |
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `go test ./... -count=1 -v` | 0 |
| `go test ./... -race -count=1` | 0 |
| `go test ./... -cover -count=1` | 0 |
| scalar fuzz, fixed `100x`, `parallel=1` | 0 |
| canonical round-trip fuzz, fixed `100x`, `parallel=1` | 0 |
| identity representation-invariance fuzz, fixed `100x`, `parallel=1` | 0 |
| closed-shape refusal fuzz, fixed `100x`, `parallel=1` | 0 |
| global tracecheck | 0 |
| generated catalog check | 0 |
| Linux amd64 build | 0 |
| Windows amd64 build | 0 |
| tracked JSON parse check (`pipefail` enabled) | 0 |
| `task-board validate` | 0 |
| `git diff --check` | 0 |
| `tracecheck -section 17.3` | 0 |
| focused scalar/canonicaljson tests | 0 |
| post-ignore gofmt check | 0 |
| post-ignore full tests | 0 |
| post-ignore build | 0 |
| post-ignore diff check | 0 |
| CR revision 6 reverse check excluding Finder artifact | 0 |

Coverage from the uncached full suite: `internal/scalar` 90.1%, `internal/canonicaljson` 80.6%, `internal/traceability` 85.0%, and `tracecheck` 87.5%. The four fuzz targets seeded 36, 21, 21, and 63 baseline cases respectively and all reported PASS.

`task-board validate` exited 0 while retaining inherited missing-activity diagnostics outside this task; the diagnostic output is preserved verbatim in the attached log.

## Capability boundary

This remains a read-only identity validation/canonicalization contribution. It makes no migration-publication, atomic-reference, durable-state mutation, or crash-recovery capability claim.
