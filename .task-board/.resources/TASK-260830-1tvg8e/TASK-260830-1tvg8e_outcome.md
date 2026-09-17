# TASK-260830-1tvg8e — developer review handoff

Base/checkpoint: `43c0e2b9a33f0b08ee9a6a6f8740cc0a68dd8792` in managed
`STORY-260830-1kiyj6`. HEAD..main was 0 at initial and handoff preparation reads.
All 13 changed paths are task-owned; SHA-256 identities are in
`candidate-files.json`. The Story branch and real index were not committed,
switched, rebased, integrated or published manually. Handoff owns the snapshot.

## Implementation and AC drivers

**5 of 5 scoped AC rows driven.** This ratio is the five deliverables named in
this leaf, not full normative section conformance. Named tests are in the
uncommitted review candidate; managed checkpoint/integration owns their eventual
signed commit, as the assignment requires.

| Scoped AC row | Production call site | Named candidate tests | Explicit evidence bound |
| --- | --- | --- | --- |
| Structured SSH argv | `sshtransport.New -> Client.Open -> Target.RPCArgv -> exec.Command` | `TestOpenStructuredCommand`, `TestOpenRefusesBeforeStart`, `TestOpenComposedArgvBound`, `TestOpenConfigurationCompatibility` | Native process with four fixed remote tokens; config 1/2/3, alias/host target, endpoint port, metacharacter identity selector, exact 65536/65537 composed-byte boundary. Config still forbids whitespace IdentityFile values; this leaf does not widen it. |
| Host-key verification responsibility | `Client.Open -> argv -> external OpenSSH` | `TestOpenNativeSSHPolicy`, `TestReceiveFailureAndLimits/exit255`, `TestOpenRefusesBeforeStart` | Real OpenSSH `-G` evaluates strict effective options against a conflicting hermetic config. OpenSSH owns crypto verification and user auth. No network, actual key exchange, real peer, key reads or invented verified-key assertion; hostile-network conformance is not claimed. |
| Bounded streams | `Session.Send`, `Session.Receive`, production stdout/stderr pumps | `TestDuplexAndSendBoundaries`, `TestReceiveFailureAndLimits`, `TestReadFailureAndRecovery` | 8 MiB excluding LF, exact-limit acceptance and +1 refusal both directions; one queued frame plus active reader, 64 KiB counted/discarded stderr; partial/empty/read/exit failure never clean EOF. Send-negative fixture is a sink so receive refusal cannot mask a missing send gate. |
| Cancellation and deterministic cleanup | `Client.Open -> Session.supervise`, `Send`, `Close`, `Wait` | `TestCancellationAndCleanup`, `TestCancelExitRace`, `TestInheritedPipeCleanup`, `TestReadFailureAndRecovery` | Config connect/RPC deadlines, explicit cancel/close, blocked write, full receive queue, failed OS pipe, repeat Close and cancellation/exit race. Reaps direct process and joins pumps; Unix ordinary descendants share a kill group. Escaped setsid child is outside termination scope but cannot retain AX waits. Windows direct-child cleanup only, cross-built rather than natively executed. |
| No permanent public listener | `Client.Open -> argv -> fixed foreground OpenSSH command` | `TestOpenNativeSSHPolicy`, `TestOpenStructuredCommand`, `TestInheritedPipeCleanup` | Effective no forwarding, no control socket/persist, no local/remote command replacement, no backgrounding and no PTY. No AX listener is created. Privileged local SSH executable/config and deliberately escaping descendants are outside the trusted-local-operator threat boundary. |

The accepted peeridentity API remains a configuration/identity plan. `Open`
means process start, not an authenticated or negotiated RPC session. Frames are
untrusted until the separate Mesh RPC owner validates hello, nonce, contract map,
limits and `Session.Target().CheckProtocolHost`. The transport does not invent a
hello subset, remote identity, capability, key registry, operation or responder
admission rule. Full §11.2 hello/operations and sibling hostile-network testing
remain explicit bounds, not permissive successful defaults.

Exit 255 is an SSH failure whose authentication/network subclass is unknown;
raw stderr is neither parsed as authority nor persisted. Earlier delivered
frames remain provisional protocol input; the RPC operation layer owns commit
semantics. No durable AX state is mutated: crash/write-idempotency injection is
N/A. Failed-read recovery, repeated closure and process/pipe failure are driven.

README and `.spec/README.md` describe API use, operator prerequisites, timeouts,
OpenSSH responsibility and platform bounds. The registry adds five executable
acceptance links (105 to 110); its exact digest and output/count fixtures agree.
Measured section coverage remains **17 of 428 keyword clauses**, with no new
full-section claim. No `ax` executable, doctor success, provider/board credential
success, encryption-at-rest or hostile-network capability is advertised.

## Verification and provenance

All commands below were run by this developer as standalone processes with
captured real exits; no `tee` pipeline. Pipeline gates used zsh `pipefail`.
No passing result is inferred from absence of output. Full local logs and exact
commands/exits are in the attached evidence archive.

- macOS arm64, Go 1.25.5; no iOS target. OpenSSH version is in its readiness log.
- Full `go test ./... -v` and `go test ./... -cover`: exit 0, 25 packages.
- Full `go test ./... -race -count=1`: exit 0, 25 packages, 238.983 seconds.
- Final `go test ./internal/sshtransport -race -count=1 -cover`: exit 0,
  **98.3%** statement coverage. This includes the final test-only sink, stderr
  OS-fault and exact deadline refinements added after the full-suite run; the
  production tree was unchanged. Handoff replays configured gates against its
  exact candidate and attaches that validation log separately.
- Native build/vet and Linux/Windows amd64 builds: exit 0. Cross-builds are not
  native runtime evidence. The external SSH must support the enforced options;
  unavailable/unsupported executables or native OS argv limits fail rather than
  retry with weakened policy.
- All 13 configured 100x fuzz smokes, tracecheck, catalog generation check,
  tracked-JSON parse, formatting and diff check: exit 0. A 100x smoke does not
  claim exhaustive fuzzing or that a large baseline corpus reached mutation.
- `task-board validate`: exit 0, **237 shared activity/history/mirror issues**;
  none names this leaf or its Story. This is not an issue-free-board claim, and
  no upstream #176/#177 or tool-source repair was attempted.

Initial red results are retained, not relabeled as green:

| Command/iteration | Real exit | Explanation |
| --- | ---: | --- |
| First focused `go test ./internal/sshtransport -count=1 -v` | 1 | IdentityFile fixture used whitespace forbidden by accepted config; fixture corrected without changing admission. |
| First combined focused race/coverage | 1 | Instrumented child emitted a missing-GOCOVERDIR warning into exact-budget stderr. Hermetic fixture coverage directory fixed; no budget widening. |
| First mutation harness launch | 2 | Harness generation had a quoting error; no applied mutant or valid kill claimed. |
| First `tracecheck` after registry update | 1 | Expected stale canonical ownership digest; exact new digest pinned for review. |
| First full `go test ./... -v` | 1 | Line-wrapped README still published 105 acceptance cases; corrected to 110. |
| Initial 35-probe corpus | 1 | Two subsumed survivors were not yet declared; retained as survivors with explicit bounds below. |

## Narrowing evidence

**33 of 35 narrowing probes killed** across 37 applied compiling probes:
33 narrowing kills, one known-bad control kill, one green neutral control and two
subsumed survivors. Every kill is a real `go test` exit 1 with a named failing
behavioral test. Compile errors, absent tests, bad plants and timeouts are not
counted. The complete 35-probe cohort is `mutations-04`; two additional timeout
probes are `mutations-timeouts`. Both harness commands exit 0 because all expected
outcomes match. The shipped default harness runs all 37; `--only` supports the
bounded named follow-up. Original production files remain byte-identical under
Go overlays.

There is no new source-text-only enforcement gate. Existing traceability checks
still only bind source declarations; these mutants preserve those declarations
and run the behavioral package, including native effective-policy evaluation.
The two redundant-policy survivors do not claim independent protection: the
forwarding gate and fixed `-T` dominate them. Their precise bounds appear in the
table. The known-bad and neutral controls validate the instrument independently.

| Mutant | What the gate is narrowed to admit | Named failing test | Exit | Result / surviving bound |
| --- | --- | --- | ---: | --- |
| neutral | Equivalent size accumulator | none | 0 | neutral passed:  |
| known-bad | Adds a remote command token | TestOpenStructuredCommand | 1 | killed:  |
| argv-plus-one | Admits exactly 65537 composed bytes | TestOpenComposedArgvBound/65537 | 1 | killed:  |
| send-plus-one | Admits exactly one excess outbound byte | TestDuplexAndSendBoundaries/oversize | 1 | killed:  |
| read-plus-one | Admits exactly one excess inbound byte | TestReceiveFailureAndLimits/oversize | 1 | killed:  |
| send-empty | Admits a non-nil empty outbound line | TestDuplexAndSendBoundaries/empty | 1 | killed:  |
| send-two-lines | Admits the two-frame outbound fixture | TestDuplexAndSendBoundaries/newline | 1 | killed:  |
| read-empty | Admits the initial empty inbound frame | TestReceiveFailureAndLimits/blank | 1 | killed:  |
| read-partial | Treats one unterminated success-shaped frame as EOF | TestReceiveFailureAndLimits/partial | 1 | killed:  |
| stderr-plus-one | Admits exactly one excess stderr byte | TestReceiveFailureAndLimits/stderr-over | 1 | killed:  |
| exit255 | Treats SSH exit 255 alone as successful EOF | TestReceiveFailureAndLimits/exit255 | 1 | killed:  |
| cancel-as-success | Suppresses explicit cancellation but keeps other failures | TestCancellationAndCleanup/stall | 1 | killed:  |
| drain-as-success | Suppresses inherited-pipe failure only | TestInheritedPipeCleanup/descendant | 1 | killed:  |
| read-as-absence | Suppresses OS stream errors only | TestReadFailureAndRecovery | 1 | killed:  |
| direct-child-only | Kills direct child but admits inherited pipe descendants | TestInheritedPipeCleanup/descendant | 1 | killed:  |
| policy-StrictHostKeyChecking | Admits StrictHostKeyChecking=accept-new while preserving all other SSH policy | TestOpenNativeSSHPolicy | 1 | killed:  |
| policy-NoHostAuthenticationForLocalhost | Admits NoHostAuthenticationForLocalhost=yes while preserving all other SSH policy | TestOpenNativeSSHPolicy | 1 | killed:  |
| policy-VerifyHostKeyDNS | Admits VerifyHostKeyDNS=yes while preserving all other SSH policy | TestOpenNativeSSHPolicy | 1 | killed:  |
| policy-UpdateHostKeys | Admits UpdateHostKeys=yes while preserving all other SSH policy | TestOpenNativeSSHPolicy | 1 | killed:  |
| policy-BatchMode | Admits BatchMode=no while preserving all other SSH policy | TestOpenNativeSSHPolicy | 1 | killed:  |
| policy-NumberOfPasswordPrompts | Admits NumberOfPasswordPrompts=1 while preserving all other SSH policy | TestOpenNativeSSHPolicy | 1 | killed:  |
| policy-ControlMaster | Admits ControlMaster=auto while preserving all other SSH policy | TestOpenNativeSSHPolicy | 1 | killed:  |
| policy-ControlPath | Admits ControlPath=/fixture/control while preserving all other SSH policy | TestOpenNativeSSHPolicy | 1 | killed:  |
| policy-ControlPersist | Admits ControlPersist=1 while preserving all other SSH policy | TestOpenNativeSSHPolicy | 1 | killed:  |
| policy-ClearAllForwardings | Admits ClearAllForwardings=no while preserving all other SSH policy | TestOpenNativeSSHPolicy | 1 | killed:  |
| policy-ForwardAgent | Admits ForwardAgent=yes while preserving all other SSH policy | TestOpenNativeSSHPolicy | 1 | killed:  |
| policy-ForwardX11 | Admits ForwardX11=yes while preserving all other SSH policy | TestOpenNativeSSHPolicy | 1 | killed:  |
| policy-Tunnel | Admits Tunnel=point-to-point while preserving all other SSH policy | none | 0 | survived: Native ClearAllForwardings=yes suppresses tunnels too; the ClearAllForwardings mutant is killed. No independent Tunnel-policy kill claimed. |
| policy-PermitLocalCommand | Admits PermitLocalCommand=yes while preserving all other SSH policy | TestOpenNativeSSHPolicy | 1 | killed:  |
| policy-RemoteCommand | Admits RemoteCommand=false while preserving all other SSH policy | TestOpenNativeSSHPolicy | 1 | killed:  |
| policy-RequestTTY | Admits RequestTTY=force while preserving all other SSH policy | none | 0 | survived: The fixed trailing -T from peeridentity overrides RequestTTY; no independent RequestTTY-policy kill claimed. |
| policy-SessionType | Admits SessionType=none while preserving all other SSH policy | TestOpenNativeSSHPolicy | 1 | killed:  |
| policy-ForkAfterAuthentication | Admits ForkAfterAuthentication=yes while preserving all other SSH policy | TestOpenNativeSSHPolicy | 1 | killed:  |
| policy-StdinNull | Admits StdinNull=yes while preserving all other SSH policy | TestOpenNativeSSHPolicy | 1 | killed:  |
| policy-ConnectionAttempts | Admits ConnectionAttempts=2 while preserving all other SSH policy | TestOpenNativeSSHPolicy | 1 | killed:  |
| connect-timeout | Allows one extra second before connection timeout | TestOpenNativeSSHPolicy | 1 | killed:  |
| rpc-timeout | Allows one extra second of process lifetime | TestCancellationAndCleanup/deadline | 1 | killed:  |

## Direct gate command exits

| Command | Exit | Log |
| --- | ---: | --- |
| `go build ./...` | 0 | `core-01.log` |
| `go vet ./...` | 0 | `core-02.log` |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | `metadata-01.log` |
| `go run ./internal/catalog/cmd/cataloggen -metadata internal/catalog/catalog.v0.5.0.json -contracts internal/specpin/v0.5.0.lock.json -output internal/catalog/catalog_gen.go -check` | 0 | `metadata-02.log` |
| `go test ./... -v` | 1 | `core-03.log` |
| `GOOS=linux GOARCH=amd64 go build ./...` | 0 | `metadata-03.log` |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 | `metadata-04.log` |
| `git ls-files -z '*.json' \| xargs -0 -n1 python3 -c 'import json,sys;json.load(open(sys.argv[1]))'` | 0 | `metadata-05.log` |
| `git diff --check` | 0 | `metadata-06.log` |
| `go test ./internal/scalar -run=^$ -fuzz=^FuzzScalarProductionEntries$ -fuzztime=100x -parallel=1` | 0 | `fuzz-01.log` |
| `go test ./internal/canonicaljson -run=^$ -fuzz=^FuzzCanonicalizeRoundTrip$ -fuzztime=100x -parallel=1` | 0 | `fuzz-02.log` |
| `go test ./internal/canonicaljson -run=^$ -fuzz=^FuzzObjectIdentityRepresentationInvariant$ -fuzztime=100x -parallel=1` | 0 | `fuzz-03.log` |
| `go test ./internal/canonicaljson -run=^$ -fuzz=^FuzzClosedIdentityShapeRefusal$ -fuzztime=100x -parallel=1` | 0 | `fuzz-04.log` |
| `go test ./internal/canonicaljson -run=^$ -fuzz=^FuzzObservationEventRefusal$ -fuzztime=100x -parallel=1` | 0 | `fuzz-05.log` |
| `go test ./internal/secconftest -run=^$ -fuzz=^FuzzCheckArgv$ -fuzztime=100x -parallel=1` | 0 | `fuzz-06.log` |
| `go test ./internal/secconftest -run=^$ -fuzz=^FuzzCheckMemberPath$ -fuzztime=100x -parallel=1` | 0 | `fuzz-07.log` |
| `go test ./internal/secconftest -run=^$ -fuzz=^FuzzIsEnvName$ -fuzztime=100x -parallel=1` | 0 | `fuzz-08.log` |
| `go test ./internal/secconftest -run=^$ -fuzz=^FuzzRedactCorpus$ -fuzztime=100x -parallel=1` | 0 | `fuzz-09.log` |
| `go test ./internal/secconftest -run=^$ -fuzz=^FuzzEscapeForTerminal$ -fuzztime=100x -parallel=1` | 0 | `fuzz-10.log` |
| `go test ./internal/secconftest -run=^$ -fuzz=^FuzzRenderForTerminal$ -fuzztime=100x -parallel=1` | 0 | `fuzz-11.log` |
| `go test ./internal/secconftest -run=^$ -fuzz=^FuzzGuardResolve$ -fuzztime=100x -parallel=1` | 0 | `fuzz-12.log` |
| `go test ./internal/secconftest -run=^$ -fuzz=^FuzzDetectCaseCollision$ -fuzztime=100x -parallel=1` | 0 | `fuzz-13.log` |
| `go test ./... -v` | 0 | `tests-01.log` |
| `test -z "$(gofmt -l $(git ls-files --cached --others --exclude-standard -- '*.go'))"` | 0 | `format-01.log` |
| `go test ./... -cover` | 0 | `tests-02.log` |
| `test -z "$(gofmt -l $(git ls-files --cached --others --exclude-standard -- '*.go'))"` | 0 | `format-01.log` |
| `go test ./... -race -count=1` | 0 | `race-01.log` |

Separate focused/mutation/build/vet/board logs have the exits stated above; their subprocess exits were observed directly. Contract research, anomalies and ownership decisions are in the attached task logbook. No external research is held only in chat.
