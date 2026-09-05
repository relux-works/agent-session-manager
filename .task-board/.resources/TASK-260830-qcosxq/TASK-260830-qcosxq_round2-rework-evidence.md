# TASK-260830-qcosxq — Round-2 rework evidence (rev2)

Base: `de51363` + uncommitted round-1 candidate (working tree). Production
untouched by this round; test-only changes in `internal/provhost` plus
README/LOGBOOK honesty items. Nothing committed, per the Change Request shape.

## Finding closures

- F25 (structural): every refusal row names its arm via `requireFrameRefusal`
  (code + `member` detail + rule detail + non-retryable) or
  `requireLocalRefusal` (code + rule detail + non-retryable); `observed`
  asserted on both mismatch paths, `status_state` on unknown. 0 code-only
  refusal rows remain outside the two helpers, the bound-child surfacing
  check, and mismatch checks paired with identity assertions.
- F21: `3.0`, `3.0.0.0`, `.0.0`, `3..0`, `3.b.c` assert
  `provider_protocol_error` with arm identity.
- F22: 31 decoded bytes refused / 32 accepted (`tokenWithDecodedBytes`).
- F23: `TestReadCappedEnforcesCap` (cut + prefix + passthrough) and
  `TestReadCappedStopsConsumingAtTheCap` (`errAfterReader` fails past 4x cap).
- F24: `TestEncodeRequestRefusesOversizeFrame` — limit+1 refused, limit framed.
- F26: `TestExecRunnerReportsCrashExit` — exit-3/exit-0 silent plugins through
  production `ExecRunner` + `Host.Call` (scripts drain stdin; see flake note).
- F27: `TestExecRunnerWritesOneTerminatedFrame` — capture file == frame + `\n`.
- F28: orphaned discovery paragraph moved back above the JSONL section.
- F29: narrowing-mutants sentence restated to the measured pins.
- Flake note: a no-read `exit 3` script races stdin writes into EPIPE
  (`runner reported no result`); fixtures now `cat > /dev/null` first.

## Mutant sweep (`/tmp/rev2-verify.sh`; cp-aside/cp-back, presence asserted, restore byte-verified)

| Mutant | Result | Killed by |
| --- | --- | --- |
| M19 del `len(parts)!=3` | KILLED | foreign_two/four-part versions |
| M20 del empty-major | KILLED | foreign_empty_major |
| M21 del empty minor/patch | KILLED | foreign_empty_minor |
| M22 del non-digit rest | KILLED | foreign_non-numeric_rest |
| M09 del response required list | KILLED | missing_protocol/version/request_id |
| M09a narrow to `{"ok"}` | KILLED | missing_protocol/version/request_id |
| M09b drop `"ok"` | EQUIVALENT | unreachable: missing-ok pre-check fires first; declared, not chased |
| S02 del status required list | KILLED | missing_plan/missing_state |
| S02a narrow to `{materialization_id}` | KILLED | missing_plan/missing_state |
| M27 del value-decode fault | KILLED | truncated + broad parse failures |
| R09 remove LimitReader | KILLED | TestReadCappedStopsConsumingAtTheCap |
| R07 remove truncation | KILLED | TestReadCappedEnforcesCap |
| M06c halve request bound | KILLED | TestEncodeRequestRefusesOversizeFrame |
| T08 token floor `< 8` | KILLED | TokenEntropyFloor (31-byte row) |
| T16 token floor `< 16` | KILLED | TokenEntropyFloor (31-byte row) |
| T48 token floor `< 48` | KILLED | TokenEntropyFloor (32-byte row) + valid fixture |
| R17 single-site exit clear | EQUIVALENT | no-op: Result literal already carries ProcessState.ExitCode() |
| R17-full both sites cleared | KILLED | crash_exit reproduces the exact process_failed→protocol_error flip |
| R14 omit `\n` terminator | KILLED | TestExecRunnerWritesOneTerminatedFrame |

18 killed, 2 declared equivalent with unreachability proofs. Per-mutant logs:
`/tmp/rev2logs/*.log`. Sweep ends `RESTORE_OK` (tree byte-identical).

## Gates (each run directly, exit codes observed)

- `go test ./... -count=1` exit 0 — 15/15 packages ok
- `go test -race ./internal/provhost/ -count=1` exit 0
- `go vet ./...` exit 0; `GOOS=windows go vet ./...` exit 0
- gofmt gate (`test -z "$(gofmt -l ...)"`) clean
- `go build ./...`, `GOOS=linux go build`, `GOOS=windows go build` exit 0
- `go run ./internal/traceability/cmd/tracecheck` exit 0, unchanged 17/403, 74 cases
- provhost coverage 88.1% (was 86.1%)

## Standing bounds (plain)

`provhost` does not import `internal/provider` (comment-only reference at
`runner.go:44`); `provhost` has no production caller and there is no `cmd/ax`.
Two orphaned packages; every `Host.Call` assertion is made from a test.
F18/F19/F20 closers untouched (production and provider tests unmodified).
