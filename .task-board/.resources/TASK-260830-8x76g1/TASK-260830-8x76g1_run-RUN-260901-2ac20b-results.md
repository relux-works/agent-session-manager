# TASK-260830-8x76g1 — developer handoff results (RUN-260901-2ac20b)

## Outcome

The authorized option-1 boundary contract is ready for review without changing
the production 5,242,880-byte identity cap. The carried rework contains:

- `TestDeclaredBoundaryConstraintsReachBothIdentityEntries`, which proves the 22
  publicly representable boundary constraints with accept-at-N and refuse-at-N+1
  cases through both `CalculateObjectIdentity` and `VerifyObjectIdentity`;
- `TestGitIndexEntryCountBoundaryIsBoundBelowThePublicObjectSizeGate`, which
  drives the production `validateGitIndex` call site directly at 65,536/65,537
  entries and then proves that both public identity entries refuse the smallest
  valid 65,536-entry parent at the earlier encoded-object cap;
- `constraint-enumeration.md`, which reports that exception and the measured
  12,323,001-byte representation instead of claiming unreachable public
  acceptance.

The shared production path remains
`CalculateObjectIdentity`/`VerifyObjectIdentity -> prepareObjectIdentity ->
validateImmutableObjectShape`. No production validator, schema limit, or public
resource-exhaustion behavior was widened to force the unreachable positive case.

## Source binding

- Current complete working-tree snapshot written through a temporary Git index:
  `cc16fd3d443927894a368558ec167e3a9e4e9851`.
- Accepted-leaves archive SHA-256:
  `9ae5e624addc7d954c391d96cab7c9f7aac3e6bcee58c76dd0cc95533d0eac9c`.
- Audit carryforward archive SHA-256:
  `df8d8fa712e0ce85bde4776dc1f83cef9d373593e76fd1145ce9d85b593ce4ee`.
- The archives were inspected in task-scoped scratch rather than overlaid onto
  the workspace. The prior attached recovery evidence binds the workspace to CR
  revision 10 and identifies archive differences as later owned task changes.

## Negative evidence

This run directly reran the focused production-entry boundary suite after the
23-mutant work and it exited 0. The already-attached
`TASK-260830-8x76g1_run-RUN-260901-6ce278-validation-logs.tar.gz` supplies the
expected-red narrowing evidence: each of the 23 widening mutants was run with
`-count=1`, each exited 1 because its named boundary test failed, and
`closed_shapes.go` was restored byte-for-byte afterward. I accepted that
already-attached mutation evidence and did not mutate production source again in
this recovery run.

## Validation run directly by this developer

Every command below was run as a standalone foreground process without `tee`.

| Command | Exit | Evidence |
| --- | ---: | --- |
| `go test ./internal/canonicaljson -count=1 -v` | 0 | `validation-canonicaljson-01.log` |
| configured `gofmt -l` gate | 0 | `validation-format-01.log` |
| `go build ./...` | 0 | `validation-build-01.log` |
| `go vet ./...` | 0 | `validation-vet-01.log` |
| `go test ./... -count=1 -v` | 0 | `validation-full-01.log` |
| `go test ./... -race -count=1` | 0 | `validation-race-01.log` |
| `go test ./... -cover -count=1` | 0 | `validation-coverage-01.log` |
| `FuzzScalarProductionEntries`, fixed `100x`, one worker | 0 | `validation-fuzz-scalar-01.log` |
| `FuzzCanonicalizeRoundTrip`, fixed `100x`, one worker | 0 | `validation-fuzz-canonicalize-01.log` |
| `FuzzObjectIdentityRepresentationInvariant`, fixed `100x`, one worker | 0 | `validation-fuzz-identity-01.log` |
| `FuzzClosedIdentityShapeRefusal`, fixed `100x`, one worker | 0 | `validation-fuzz-closed-shape-01.log` |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | `validation-trace-01.log` |
| `go run ./internal/traceability/cmd/tracecheck -section 17.3` | 0 | `validation-trace-17.3-01.log` |
| catalog generated-output freshness check | 0 | `validation-catalog-01.log` |
| Linux amd64 build | 0 | `validation-linux-build-01.log` |
| Windows amd64 build | 0 | `validation-windows-build-01.log` |
| tracked/untracked JSON parse | 0 | `validation-json-01.log` |
| `git diff --check` | 0 | `validation-diff-check-01.log` |
| `task-board validate` | 0 | `validation-board-01.log` |

Coverage was 82.5% for `internal/canonicaljson` and 90.1% for
`internal/scalar`. The scoped Section 17.3 trace gate reported one assigned
scope. Board validation retained 262 inherited `MISSING_ACTIVITY` diagnostics;
the current task was not among them.

## Scope boundary

The package remains read-only and deterministic. This task does not claim
migration publication, atomic reference advancement, rollback retention,
durable-state recovery, `ax migrate`, `ax doctor`, or a runtime capability.
