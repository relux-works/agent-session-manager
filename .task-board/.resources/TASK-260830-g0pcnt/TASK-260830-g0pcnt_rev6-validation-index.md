# TASK-260830-g0pcnt revision 6 revalidation index

Run: `RUN-260924-80476b`  
Candidate checkpoint: `5a64077facb9f93357b9bbc8ebdba93069105715`  
Trunk: `0ca3e4c26e2b275212796657f785b9b450f6174e` (unchanged during this run)  
Environment: macOS Darwin/arm64, Go 1.25.5.

This revalidation changed the test-only transitive path-taint analysis in
`internal/tmuxserver/custody_walk_depth_unix_test.go` and added three mutant
rows in `internal/tmuxserver/mutant_harness.py`. No production Go source,
registry, README, or `task-board.config.json` changed. All commands ran as
standalone foreground processes with stdout/stderr redirected to the listed
task-scoped log. Exit codes below are the actual command exits, not inferred
from output text.

## Configured package tests

| Group | Packages | `go test -count=1 -v` | `go test -race -count=1 -timeout 25m` | `go test -cover -count=1` |
|---|---|---:|---:|---:|
| 01 | axerror, axpane, canonicaljson, catalog, catalog/cmd/cataloggen | 0 | 0 | 0 |
| 02 | cataloggen, cigate, cliresult, clonebundle, cloneproject | 0 | 0 | 0 |
| 03 | clonesnap, config, crashgate, dirnode, environ | 0 | 0 | 0 |
| 04 | fencing, hostchannel, hosttrust, invcore, localstore | 0 | 0 | 0 |
| 05 | matjournal, meshneg, peeridentity, provhost, provider | 0 | 0 | 0 |
| 06 | resumesmoke, rpcwire, scalar, secconftest, secprim | 0 | 0 | 0 |
| 07a | sessadapter, sessckpt, sessprofile, sessrepo | 0 | 0 | 0 |
| 07b | sessquery | 0 | interrupted; prior exact green reused | 0 |
| 08a | sessstate, specdoc, specpin, sshtransport | 0 | 0 | 0 |
| 08b | termbind | 0 | 0 | 0 |
| 09 | terminalbackend, terminstance, traceability, traceability/cmd/tracecheck | 0 | 0 | 0 |
| 10 | tmuxserver | 0 | 0 | 0 |

Current normal-test logs are `config-go-test-group-01.log` through
`config-go-test-group-10.log`; current race logs are `config-go-race-group-01`
through `-06`, `-07a`, `-08a`, `-08b`, `-09`, and `-10`; coverage logs are
`config-go-cover-group-01` through `-06`, `-07a`, `-07b`, `-08a`, `-08b`,
`-09`, and `-10`.

The current monolithic configured normal command
`go test ./... -count=1 -v` was interrupted at the shell-call window and exited
1 without a test failure. It is not counted green; all 45 packages passed in
the ten bounded current normal groups above.

The current isolated race command for `sessquery` was interrupted at 9:48 and
exited 1 without a test failure. That package and its production dependencies
are unchanged from the prior rev6 candidate, and the environment is the same.
The exact prior rev6 race log is
`previous-evidence/.temp/TASK-260830-g0pcnt/rev6-race-group-03.log`; it records
`sessquery` green in 409.416s. The changed `tmuxserver` race suite was rerun in
current group 10 and passed. Thus 44 package race suites are current and the
one unchanged package uses bound prior green evidence.

## Other configured gates

| Gate | Exit | Log |
|---|---:|---|
| 17 configured fuzz targets, 100 executions each | 0 each | `config-fuzz-01.log` through `config-fuzz-17.log` |
| Current custody test selection, `-count=3` | 0 | `config-determinism-count3.log` |
| Native build and vet | 0 each | `config-go-build.log`, `config-go-vet.log` |
| Windows vet | 0 | `config-go-vet-windows.log` |
| Linux and Windows builds | 0 each | `config-linux-build.log`, `config-windows-build.log` |
| Gofmt check | 0 | `config-gofmt-check.log` |
| Tracecheck after documentation edits | 0 | `final-tracecheck-after-docs.log` |
| README measured-coverage pin test | 0 | `final-readme-coverage-pin-test.log` |
| Adopted catalog check | 0 | `config-cataloggen-check.log` |
| JSON parse gate with `pipefail` | 0 | `config-json-parse.log` |
| Task-board validation | 0 | `config-task-board-validate.log`; 252 workspace issues, none names this task or Story |
| Final `git diff --check` | 0 | `final-diff-check-after-docs.log` |
| No tmux-process check | 0 | `no-tmux-process-check.log`; count is zero |

The previous rev6 digest and README wrong-figure plant remain applicable because
the registry, README, and pin test were unchanged. The digest is
`3664ab2fb166189545fea34a068a318af4f251689f29c92915fa185fefdedeea`; current
tracecheck prints `bindings=70 full=4 partial=10 sliver=11 unevidenced=41
unmeasured=4 unowned=7 clauses_discharged=81/585`. The previous rev6 importer
grid remains applicable because production implementation and runtime entries
were unchanged: 192 base keys, 223 candidate keys, 192 shared, 0 moved, 31
candidate-only.

## Isolated walk-length mutants

| Mutant | Named failing test | Run evidence | Result |
|---|---|---|---|
| `N-ancestor-walk-callee-depth-cap-24` | `TestCustodyAncestorWalkGeneratedDepthToPathMax` | `callee24-posttaint-run1/2-harness.log` and raw mutant logs | KILLED twice alone; raw test exit 1 |
| `N-ancestor-walk-filesystem-root-depth-cap` | `TestCustodyAncestorWalkHasNoLengthDependentControlFlow` | `root-cap-posttaint-run1/2-harness.log` and raw mutant logs | KILLED twice alone; raw test exit 1 |
| `N-ancestor-walk-split-component-count-cap` | `TestCustodyAncestorWalkGeneratedDepthToPathMax` and transitive guard | `split-cap-run1/2-harness.log` and raw mutant logs | KILLED twice alone; raw test exit 1 |
| `N-ancestor-walk-range-index-cap` | `TestCustodyAncestorWalkGeneratedDepthToPathMax` and transitive guard | `range-cap-run1-retry` and `range-cap-run2-harness.log` with raw mutant logs | KILLED twice alone; raw test exit 1. First link attempt is excluded as ERROR (`no space left on device`). |
| `N-ancestor-walk-byte-length-cap` | `TestCustodyAncestorWalkGeneratedDepthToPathMax` and transitive guard | `byte-cap-run1/2-harness.log` and raw mutant logs | KILLED twice alone; raw test exit 1 |
| `C-control` | None expected | `control-posttaint-run1-harness.log` and raw control log | Applied, SURVIVED; harness exit 0 |

The full 360-row harness was not run. The previous complete rev5 355-row
harness passes remain attached; the five rev6 narrowing rows above each have
two isolated raw runs.
