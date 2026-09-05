# TASK-260830-qcosxq outcome — provider JSONL protocol host

Leaf 2 of STORY-260830-3jqsx1. Built on checkpoint `de51363`; leaf 1
untouched except its three handed-forward findings. Work left
**uncommitted** in the worktree per the Change Request shape.

## Deliverable

New package `internal/provhost` — the ax host side of the §7.2
JSON-over-stdio protocol:

- One-frame JSONL transport: exact 6-member request envelopes,
  strict duplicate-detecting response decode, 8 MiB line limit both
  directions (absolute literal fixtures), UTF-8 gate, single-frame
  stdout rule, stdout/stderr separation with redaction (stderr never
  enters failure text, proven with a planted secret).
- Deadlines: request deadline enforced via context cut; `provider_timeout`
  only when the instant actually passed; parent-cancel and early transport
  failures are `provider_process_failed`. Deadline kill is process-group
  on unix (direct child on Windows, stated) with `WaitDelay` and stream
  abandonment; elapsed-time bound pins the group kill.
- Operation dispatch: exactly the 15 §7.5 operations in manifest order,
  derived from the pinned table; unknown names refused without spawning.
- Status recovery: `DecodeStatusOutcome` enforces state/nullability/ID
  rules; unknown fails closed with `integrity_failure` for quarantine;
  reads evolve with no replay rule; every call is a fresh process, so
  cross-process recovery observes durable state through the passed
  authority. Lost-mutation test proves no blind mutation retry.
- Failures are `*axerror.Error` 1.0.0: six constructor sites audited
  bidirectionally with a closed code set; child failures via the static
  provider-major-2 `DecodeBound`; first-frame faults via
  `LocalFromUntrusted` (§15.1 row, never trusting a foreign payload).

Inherited findings: F18 pinned (`ownerAttester` pointer-equals
`fileOwnerUID`, SEAM1 verified red); F19 planted distinct-ID candidates
so the `return out` mutant returns 1 candidate and reddens; F20 added
`GOOS=windows go vet ./...` to CI (planted type error verified it sees
`os_windows_test.go`).

Notable: the registry counts 15, not 16 — manifest JSON and §7.5 table
agree; the ownership gap prose miscounts. Transport placed in
`internal/provhost` (not `internal/provider`) to preserve leaf 1's
no-`os/exec` structural scan. Full body vocabularies and mutation
idempotency are the conformance leaf's; no §7.2 clause newly discharged.

## Evidence (exit codes observed directly)

- `go test ./... -count=1` → exit 0, 15/15 packages ok
- `go test -race` → exit 0, all 15 packages in bounded batches
- `go vet ./...` → exit 0; `GOOS=windows go vet ./...` → exit 0
- `gofmt -l .` → clean; `go generate ./internal/catalog` + diff → current
- `go run ./internal/traceability/cmd/tracecheck` → exit 0,
  unchanged at 17/403 clauses, 74 acceptance cases
- provhost coverage 86.1% statements; blind spot stated: defensive
  constructor-error branches (constructors proven total) and ExecRunner
  stdin-failure races
- 25 narrowing mutants, each verified present, compiling, and restored:
  25 killed, 0 survivors (`TASK-260830-qcosxq_mutants.log`)
- Focused suite: 232 PASS lines, 0 FAIL
  (`TASK-260830-qcosxq_tests.log`)

## Files

- `internal/provhost/{doc,protocol,runner,runner_unix,runner_windows,status}.go`
- `internal/provhost/{protocol,runner,status,inventory}_test.go`
- `internal/provider/{os,discovery}_test.go` (F18 pin, F19 fixture)
- `.github/workflows/ci.yml` (F20 step), `README.md` (protocol section),
  `LOGBOOK.md` (entry 0429)
