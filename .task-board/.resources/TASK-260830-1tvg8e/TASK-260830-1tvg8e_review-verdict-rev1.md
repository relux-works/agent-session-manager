# TASK-260830-1tvg8e — independent review, revision 1

Verdict: **accepted**. No blocking implementation finding. `repeat-of: none`.
This accepts the five scoped SSH transport/command behaviors, not full Mesh RPC
or authenticated protocol-session conformance. The named acceptance mutation
routes this exact candidate to integrating; this reviewer does not checkpoint,
commit, publish, or integrate it.

## Candidate and authority

- CR: `CR-TASK-260830-1tvg8e-1`, revision `1`, repository_delta `present`.
- Base: `43c0e2b9a33f0b08ee9a6a6f8740cc0a68dd8792`.
- Candidate tree: `f1dbf8f2dcd80ec8fdee9c0e7e1822cd921d41ac`.
- Patch SHA-256: `1641def4bc3b210fc9f534a68982d22a3553d847d96ad09d73a3472dca24adb6` (independently verified).
- Normative authority: AX v0.5.0, `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`, §§6, 11.1, 16.1; §11.2 supplies the 8 MiB LF framing bound. Read embedded pinned SPEC.md; specification fidelity gates passed.
- Reviewed producer outcome/logbook and current merged 18-row checklist before acceptance. AC driver table was read before implementation code.
- Archived exact candidate into `.temp/TASK-260830-1tvg8e/review-r1/candidate`; independently checked all 2,932 file/symlink Git blob identities and all 13 changed paths against the managed workspace. Managed HEAD remains the base above; HEAD..main count was 0. No source/index/branch edits in the managed workspace.
- `task-board spawn goal` reported this run is not goal-bound. No operator directives were pending.

## AC coverage

**5 of 5 AC rows driven**, with the explicit bounds below. Tests named here are
in the exact CR candidate, awaiting managed signed checkpoint; reviewer-only
neighbor probes are supplemental and do not substitute for candidate tests.

| AC row | Production entry / actual path | Named candidate drivers | Result and bound |
| --- | --- | --- | --- |
| Structured SSH argv | `New → Client.Open → peeridentity.Target.RPCArgv → exec.Command/Start` | `TestOpenStructuredCommand`, `TestOpenRefusesBeforeStart`, `TestOpenComposedArgvBound`, `TestOpenConfigurationCompatibility` | Fixed remote tokens, allowlist refusal, pre-cancel/start failure and 65536/65537 boundary pass; no shell-generated command. |
| Host-key verification | `Client.Open → argv → external OpenSSH` | `TestOpenNativeSSHPolicy`, `TestReceiveFailureAndLimits/exit255` | Native OpenSSH resolves strict policy despite conflicting isolated config; exit255 is a failure, never absence/authentication success. Crypto/key exchange is delegated to external SSH and not executed by these hermetic fixtures. |
| Bounded streams | `Session.Send/Receive → stdin.Write/readLines/readStderr` | `TestDuplexAndSendBoundaries`, `TestReceiveFailureAndLimits`, `TestReadFailureAndRecovery` | Exact 8 MiB acceptance, +1 refusal, empty/newline/partial/read errors, stderr exact 64 KiB/+1 and duplex flow pass. Frames are untrusted; no authorized result is published by this API. |
| Cancellation and cleanup | `Client.Open → Session.supervise`, `Close`, `Wait`, `Send` | `TestCancellationAndCleanup`, `TestCancelExitRace`, `TestInheritedPipeCleanup`, `TestReadFailureAndRecovery` | Before-launch cancel, configured deadline, blocked write/full queue, direct reap, inherited pipes and retry paths pass. One-second post-exit backpressure/drain bound is documented. Unix process-group scope excludes setsid escape; Windows direct-child scope is explicit. |
| No permanent public listener | `Client.Open → fixed foreground SSH invocation`, `Session.supervise` | `TestOpenNativeSSHPolicy`, `TestOpenStructuredCommand`, `TestInheritedPipeCleanup` | Native effective policy disables forwarding/control sockets/persistence/TTY/backgrounding; AX creates only owned pipes and a child. Local SSH binary/config are trusted operator inputs, not hostile caller-controlled configuration. |

The production call search found no existing product binary importing this new
library. That is consistent with this repository's pre-existing absence of an
`ax` executable, explicitly documented in README. This is a real production
transport API rather than a fake Runner: it owns native process start, six pipe
endpoints, stream pumps, waits and termination. The supplemental default-route
probe exercises `New → Open → ssh from PATH` without changing any Client fields.

`Target.CheckProtocolHost` remains a plan-level comparator. Runtime hello and
responder host-ID admission belong to the Mesh RPC owner and are **not** claimed
implemented here. `Open` reports only local process start, `Target` reports only
configured identity, and delivered frames remain provisional/untrusted. No
verified-host flag, authorized operation result, doctor success, provider-auth
success, at-rest encryption, or full-section conformance is introduced. This is
why absent RPC integration is a stated downstream bound rather than invented
authentication evidence. Existing predecessor admission semantics are preserved.

## Attacks and independent evidence

All 37 shipped probes were rerun through compiling Go overlays against the exact
candidate, in two bounded batches. **33 of 35 narrowing witnesses killed**; the
known-bad command mutation also killed, the neutral control passed, and two
subsumed policy survivors reproduced. `Tunnel` is dominated by
`ClearAllForwardings=yes`; `RequestTTY` is dominated by fixed `-T`. Those stronger
gates were independently exercised. Full named-failure tables, actual exits,
overlays and JSON test events are in `mutants/`, `mutants-remaining/`, and
`mutants-combined.json`. No compile failure, timeout, empty selection, or
not-applied probe was counted as a kill. No new source-text guard is introduced;
these mutations preserve production declarations and execute behavior, including
native SSH parsing, rather than running the trace checker alone.

Additional disposable-copy probes, all passing five times under race detection:

- `TestReviewSimultaneousPressureAndPartialWrites`: 16 ordered 256 KiB stdout frames written in 4 KiB pieces concurrently with exactly 64 KiB stderr, then clean exit/EOF.
- `TestReviewNativeConfiguredDestination`: `Open` into native isolated `ssh -G`, conflicting endpoint port 2222, `-p 3333`, `Port=4444` and config port 5555; effective port remains 2222, user alice and host ::1.
- `TestReviewRetryAfterRefusalAndCancel`: unknown-peer and pre-cancel refusals followed by ten successful clean sessions per run.
- `TestReviewDefaultNewOpenUsesPATHSSH`: synthetic executable named ssh in a disposable PATH; public `New/Open` uses it without private-field injection and retains the fixed destination/command.

The probe source is included as `reviewer_probe_test.go`. These are fixture
processes, not real peers. Native `-G` probes use explicit isolated configuration
or `/dev/null`; no private credential read, SSH setting mutation, network
connection, or listener exposure was used. Raw transport stderr canaries do not
appear in returned failures.

## Validation rerun by this reviewer

All commands below exited 0. Full logs and actual exit files are attached.

| Command | Evidence |
| --- | --- |
| `go test ./... -v -count=1` | `tests-01.log`: 25 packages; 5 explicit pre-existing skips (one diagnostic dump and four canonicaljson corpus subcases), no sshtransport skips |
| `go test ./... -cover -count=1` | `coverage.log`: 25 packages, sshtransport 98.3% |
| `go test ./... -race -count=1` | `race.log`: 25 packages |
| `go test ./internal/sshtransport -race -count=1 -cover` | `focused-race.log`: 98.3% |
| `go build ./...`, `go vet ./...` | `gates.json`, `gates-01/02.log` |
| Windows build and vet; Linux amd64 build | `gates-03/04/05.log`; cross-checks only, not native runtime tests |
| `go run ./internal/traceability/cmd/tracecheck` | `gates-06.log`; 110 linked cases, full-section coverage remains explicitly limited |
| Catalog generator `-check`; `gofmt -l internal` | `gates-07/08.log`; exact commands in `gates.json` |
| All 13 derived fuzz targets, `-fuzztime=100x -parallel=1 -count=1` | `fuzz.json` and per-target full logs; engine execution checked; smoke budget is not exhaustive fuzzing |
| Exact candidate JSON parsing, blob identity comparison, `git diff --check BASE TREE` | `identity.json`, `diff-check.log` |
| `task-board validate` | `board-validate.log`: exit 0 with 237 shared issues, none naming this task/Story; not an issue-free-board claim |

Environment: macOS arm64, Go 1.25.5, OpenSSH 10.3p1 / LibreSSL 3.3.6. These are
local review gates, not claims of hosted CI or native Linux/Windows execution.
Producer logs were read for context; every core gate and all 37 mutation results
above were independently rerun, rather than accepted solely from its report.

## Findings and checklist disposition

No blocking finding; severity/production consequence/reproduction/repeat-of:
not applicable. The observed limits are reported above rather than inferred
as successful crypto/RPC/platform coverage. No upstream #176/#177 change is
needed or implemented. No durable AX state is mutated by this transport, so
crash/write-idempotency tests are inapplicable; process/read failure recovery,
closure idempotence and retries are exercised.

Reviewer checklist rows 14–17 are satisfied by this analysis and attached reruns.
Row 18 is conditional on rejection and is satisfied as N/A for acceptance.
All work products are persisted before the named acceptance mutation.
