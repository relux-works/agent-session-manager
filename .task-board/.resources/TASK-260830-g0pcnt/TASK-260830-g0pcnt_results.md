# TASK-260830-g0pcnt Results — revision 6

## Measured acceptance coverage

**4 of 4 acceptance-criteria rows are driven** through named production entries.

| Acceptance row | Production entry | Named behavioral evidence | Narrowing evidence |
|---|---|---|---|
| Lost-response idempotency | Lifecycle.Execute → executeAttach | TestExecuteAttachLostResponseReplaysRecordedOutcome verifies the recorded vector, receipt, effect evidence, and original timestamp; retry performs no second stage/install or receipt write. | N-attach-lost-response-replay |
| Reconnect and multi-attach policy | Lifecycle.Execute(attach) → executeAttach / checkAttachOverlap | TestAttachSameClientRetryWithPeerPresent, TestAttachOverlapRequiresMultiAttach, TestAttachOverlapRequiresMultipleInputClients, TestAttachOverlapAdmissionAtomicAcrossLifecycleInstances, and TestExecuteSerializesSameClientAttachBeforeConcurrentWinnerArm. | N-attach-overlap-multi, N-attach-overlap-input, N-attach-overlap-replay-unproven, N-attach-postlock-generation, and N-attach-same-client-concurrent-winner |
| Socket substitution refusal | ServerProber.Probe, ServerSpawner.Spawn, and all eight Lifecycle.Execute operation dispatches | Probe, Spawn, and Execute substitution tests, plus exhaustive custody mode/kind/owner oracles. Refusal precedes Dial, dispatch, unlink, and spawn. | N-socket-identity-swap, N-socket-symlink, N-socket-ownership, N-socket-permissions, and the whole-path custody narrowings |
| Ownership-neutral attach | Lifecycle.Execute(attach) → executeAttach | TestExecuteAttachDoesNotReadOrChangeSessionLease snapshots the durable lease and event chain before/after. | N-attach-ownership-neutral |

The production witnesses use fakes and synthetic metadata; no tmux process or
bound socket is needed. Every admission reaches a fake next stage; every
refusal checks the literal specification token/detail and zero effects.

## Revision 6 finding answered — physical walk length and transitive callees

The rev5 generated-depth finding was a path-length branch in the callee
`custodyModeForPath` beyond the fixture range. The new test uses
`golang.org/x/sys/unix.PathMax` (Darwin: 1024), builds one-byte nested directory
names until the runtime-root pathname is 1023 bytes, and sets a real non-sticky
0777 mode at both the deepest non-root ancestor (1021-byte path) and a middle
ancestor. `TestCustodyAncestorWalkGeneratedDepthToPathMax` drives the shared
`checkCustodyAncestors` predicate at both positions, asserts literal
`tmux_unsafe_socket_path at socket ancestor`, and confirms that each target was
visited exactly once. Reverse-order `t.Cleanup` removes every generated
directory. The derived socket is beyond Darwin `sun_path`; it is rejected by
the existing socket-length gate before a production filesystem walk.

`TestCustodyAncestorWalkReachableFromProductionEntries` parses the production
call graph and proves ServerProber.Probe, ServerSpawner.Spawn, and
Lifecycle.Execute each reach CheckSocketCustody and the same
`checkCustodyAncestors` predicate. Thus the physical-limit walk is tested at
the shared predicate seam, with production-entry wiring separately pinned.

`TestCustodyAncestorWalkHasNoLengthDependentControlFlow` discovers the
transitive in-module call graph from that predicate in the active tmuxserver
and secprim production files. It includes `custodyModeForPath`,
`writableByOthers`, `isFilesystemRoot`, `secprim.OpenNoFollowDir`, and its
local callees. It rejects path-derived length/depth comparisons, counter
loops, and counter updates. The mechanically identified
`secprim.memberErrorTarget` helper is reached only under `failPath` detail
construction; its existing message truncation cannot affect the gate.

The `callee24` narrowing and an `isFilesystemRoot` cap were each run alone
twice. Their named tests failed with subprocess exit 1 on both runs. The
line-count-preserving C-control survived as expected. The current harness has
360 rows: the 355-row rev5 harness plus five rev6 walk-length narrowings. The
full 360-row harness was not run as one complete pass; all five additions have
two isolated raw runs each.

| Mutant | Narrowing or control | Named failing test | Result / bound |
|---|---|---|---|
| `N-ancestor-walk-callee-depth-cap-24` | In `custodyModeForPath`, clear group/other write bits after 24 separators, admitting a 0777 ancestor past rev5's generator range | `TestCustodyAncestorWalkGeneratedDepthToPathMax` | KILLED twice alone; both deepest and middle placements fail in raw Go test exit 1 |
| `N-ancestor-walk-filesystem-root-depth-cap` | Make `isFilesystemRoot` stop the walk after 24 separators | `TestCustodyAncestorWalkHasNoLengthDependentControlFlow` | KILLED twice alone; the transitive AST guard fails in raw Go test exit 1 |
| `C-control` | Applied comment-only, line-count-preserving neutral edit | None expected | SURVIVED; expected harness control, not counted as a gate narrowing |

The previous supplementary survivor `D-attach-entry-deadline-shadowed` keeps
its rev5 bound: the earlier wait guard refuses the same deadline instant
before admission; owner is the `Lifecycle.Execute` wait/admission boundary,
measured by `N-attach-postlock-deadline` and `N-attach-wait-deadline`.

## Revision 4 finding answered

The rev4 repeat-of finding was custody walk length pinned to fixture depth.
The whole-path mode oracle now runs at the current path depth and at a fixture
with eight additional nested levels. It covers 22 concrete component positions
across those fixtures through Probe, Spawn, and Execute: **270,336 of 270,336
mode-position-entry classifications**, **330 of 330 kind-position-entry
cases**, and **54 of 54 owner-position-entry cases**. The owner sweep covers
the socket leaf, runtime tmux directory, and runtime root; ancestor ownership
remains the stated N2 bound.

A generated depth test varies 1–16 extra nested levels and plants a
non-sticky 0777 mode at the nearest, middle, and deepest non-root ancestor
through all three production entries: **144 of 144 named refusal cases**.
Each asserts literal tmux_unsafe_socket_path at socket ancestor and zero
effects. The AST test requires an unconditional parent walk with no counter,
limit, or early nil return outside the filesystem-root guards.

The six rev5 narrowings (depth caps 4/6/10, deepest-ancestor skip, AST
counter/cap, and AST early-return) each ran alone twice and were KILLED by
their named test. The harmless line-count-preserving C-control survived twice
as expected. In addition, the complete current 355-row shipped harness was
rerun twice in bounded slices. Each pass had 355 unique rows and raw
subprocess logs: 353 KILLED, two expected SURVIVED, zero NOT_APPLIED,
MISMATCH, or ERROR rows, and 24/24 candidate-file hashes restored. Pass 1
group logs and audit are rev5/harness-pass-1-group-*.log and
rev5/harness-pass-1-audit.json; pass 2 evidence is
rev5/harness-pass-2-group-*.log, rev5/harness-pass-2-audit.json, and
rev5/mutants/current-pass-2/. The two survivors are the expected harmless
C-control and the supplementary shadowed D-attach-entry-deadline-shadowed;
the latter has no failing test because the earlier wait guard rejects the same
instant before admission, owned by the Lifecycle.Execute wait/admission
boundary and measured by N-attach-postlock-deadline and
N-attach-wait-deadline. An initial pass-1 slice 300:350 exceeded the bounded
run window and was interrupted (exit 130); it is excluded. The complete
replacement slices all exited 0.

| Mutant | What it narrows | Named test that fails | Result |
|---|---|---|---|
| N-ancestor-walk-depth-cap-4 | Admits an unsafe ancestor after four checked parents | TestCustodyAncestorWalkGeneratedDepthAtProductionEntries | KILLED twice |
| N-ancestor-walk-depth-cap-6 | Admits an unsafe ancestor after six checked parents | TestCustodyAncestorWalkGeneratedDepthAtProductionEntries | KILLED twice |
| N-ancestor-walk-depth-cap-10 | Admits an unsafe ancestor after ten checked parents | TestCustodyAncestorWalkGeneratedDepthAtProductionEntries | KILLED twice |
| N-ancestor-walk-skips-deepest-generated-ancestor | Admits the generated deepest non-root unsafe ancestor | TestCustodyAncestorWalkGeneratedDepthAtProductionEntries | KILLED twice |
| N-ancestor-walk-depth-cap-ast-control | Adds an ancestor counter/cap | TestCustodyAncestorWalkHasNoLengthDependentControlFlow | KILLED twice |
| N-ancestor-walk-early-return-ast-control | Adds return nil outside a root exit | TestCustodyAncestorWalkHasNoLengthDependentControlFlow | KILLED twice |
| C-control | Harmless line-count-preserving comment | None; control must survive | SURVIVED twice |

## Remaining bounds and handoff declarations

- B44 remains with a future authoritative attach-receipt/protocol owner: a receipt alone does not prove that a disconnected client stopped.
- B46 remains with CheckSocketCustody: a pathname replacement after final lstat and before OS connect cannot be portably pinned on Unix.
- N2 remains with the future tmux path-custody owner at checkCustodyAncestors: ancestors above the AX runtime root are not owner-checked. SPEC v0.7.0 §3.2 lines 810–813 does not define a trusted owner set for system ancestors; a foreign non-root owner could rename one.
- Out-of-contract acceptance rows: none. These are explicit bounds within the measured criteria, not waived rows.
- Brief gap: the producer brief supplied no catalog-derived surface table. The supplemental coverage map is explicitly not represented as that missing table.
- Rework diff stays within the named tmux backend Story boundary: tmuxserver implementation/tests, traceability registry/code/tests needed for story-final ownership, README, and newest-first LOGBOOK entry. The rev5 extension adds custody tests, mutant definitions, and evidence documentation.

Read-only→writable escalation remains refused by the literal idempotency_mismatch token with no writable vector when a read-only receipt is replayed with InputAuthorized=true; the planted-away store refusal is killed. Foreground Probe → Acquire → Execute attach is exercised end to end, and N-foreground-fresh-decoy-wiring is killed. No unsupported capability is advertised.

## Registry and traceability

The decoded ownership registry retains all four Story leaf bindings and the
trunk case story-260922-derivation-side-profile-source in clause 2.4#2.
Every acceptance case has a clause edge and named production owner. The
registry and cited test names did not change in rev5; the current canonical
digest remains:

reviewedOwnershipCanonicalSHA256=3664ab2fb166189545fea34a068a318af4f251689f29c92915fa185fefdedeea

Exact-tree tracecheck exited 0 and printed:

    traceability ok: contracts=64 normative_sections=36 acceptance_cases=160 fixtures=33 compatibility_contracts=55 assigned_scopes=0
    section coverage: bindings=70 full=4 partial=10 sliver=11 unevidenced=41 unmeasured=4 unowned=7 clauses_discharged=81/585

The README Measured coverage subsection equals that output. In an isolated
README plant, changing the figure 70 to 71 caused
TestREADMEMeasuredCoverageMatchesTracecheckReport to fail (exit 1); the
restored README hash matches its pre-plant bytes. The traceability package
tests and tracecheck passed on the candidate. Section-scoped diagnostics from
the unchanged registry snapshot: §2.4 exit 0; §3.2 exit 1 (1/13 sliver);
§4.2 exit 1 (5/11 sliver); §4.C exit 1 (5/7 partial); §4.D exit 1
(1/3 sliver); §4.E exit 1 (no scoped owner). These nonzero scoped exits are
the measured undercoverage diagnostics, not green gates.

Rev6 re-derived the registry projection independently in a copy by planting a
zero digest pin. The resulting refusal printed
`3664ab2fb166189545fea34a068a318af4f251689f29c92915fa185fefdedeea`, equal to
the candidate's reviewed pin. The rev6 README plant changed the fenced figure
from 70 to 71; `TestREADMEMeasuredCoverageMatchesTracecheckReport` failed with
exit 1 and named the mismatched figure. The candidate README itself remains
unchanged and tracecheck passed on it.

## Importer outcome comparison

The runtime outcome grid keyed by (package, entry, input) was rerun on base
0ca3e4c26e2b275212796657f785b9b450f6174e against the rev6 candidate over the
complete importer set. The new depth test and AST graph test add no runtime
entry, so the measured counts remain:

| Measure | Base | Candidate | Result |
|---|---:|---:|---|
| Outcome rows | 192 | 223 | 192 shared |
| Packages | 35 | 36 | 31 candidate-only keys |
| Moved shared outcome classes | — | — | 0 |

The 31 candidate-only keys are four termbind Peers inputs and 27 tmuxserver
inputs; their names are listed above. No shared outcome class moved. The rev6
comparison JSON and both grids are under
`closuregrid-rev6-0ca3e4c/` and are included in the evidence archive.

## Historical rev5 validation commands and actual exits

The historical commands below ran on the refreshed rev5 source/test tree
(checkpoint 5a64077facb9f93357b9bbc8ebdba93069105715 plus uncommitted
candidate changes), before the rev6 physical-depth test and two harness rows
were added. Each command was a standalone process. Long race validation ran in
bounded sequential package groups, not in a background process.

| Command / gate | Exit | Evidence |
|---|---:|---|
| go test ./... | 0 | rev5/go-test-all-exact-01.log; all 45 packages passed |
| go test ./... -count=1 -v | 0 | rev5/go-test-all-01.log; all 45 packages passed |
| Full shipped mutant harness, 355 rows, pass 1 and pass 2 | 0 each | rev5/harness-pass-1-audit.json and harness-pass-2-audit.json; 353 KILLED, two named SURVIVED each pass; every KILLED row has two full-harness observations plus the separately repeated rev5 depth mutants |
| go test ./... -cover -count=1 | 0 | rev5/go-test-cover-01.log; 45 packages |
| Four configured race groups, go test -race -count=1 -timeout 25m | 0 each | rev5/race-group-01.log through race-group-04.log; 12+12+12+9 packages |
| 17 configured fuzz invocations with -fuzztime=100x -parallel=1 | 0 each | rev5/fuzz-*.log |
| New custody depth/oracle/name/symlink tests with -count=3 -v | 0 | rev5/determinism-count3-01.log |
| Focused generated-depth test | 0 | rev5/generated-depth-01.log |
| Focused mode/kind/owner oracles | 0 | rev5/mode-kind-oracles-02.log |
| AST structure test | 0 | rev5/ast-walk-structure-02.log |
| Component-name byte-length test | 0 | rev5/component-name-boundary-04.log |
| Component-name content test | 0 | rev5/component-name-axes-01.log |
| Symlink-chain test | 0 | rev5/symlink-chain-02.log |
| go run ./internal/traceability/cmd/tracecheck | 0 | rev5/tracecheck-01.log |
| go test ./internal/traceability/... -count=1 | 0 | rev5/traceability-tests-01.log |
| go build ./...; go vet ./... | 0 each | rev5/go-build-01.log; rev5/go-vet-01.log |
| GOOS=linux GOARCH=amd64 go build ./... | 0 | rev5/go-build-linux-01.log |
| GOOS=windows GOARCH=amd64 go build ./... | 0 | rev5/go-build-windows-01.log |
| GOOS=windows GOARCH=amd64 go vet ./... | 0 | rev5/go-vet-windows-01.log |
| Configured gofmt gate | 0 | rev5/gofmt-configured-01.log |
| cataloggen adopted catalog check | 0 | rev5/cataloggen-check-01.log |
| JSON parse gate with pipefail | 0 | rev5/json-parse-01.log |
| git diff --check | 0 before evidence-doc edits | rev5/diff-check-01.log |
| README wrong-figure plant / restored suite | 1 expected red / 0 | rev5/readme-coverage-plant-negative.log; rev5/traceability-tests-01.log |
| Importer outcome grid versus 0ca3e4c | 0 | rev5/importer-grid-01.log and closuregrid-rev5-0ca3e4c |
| task-board validate | 0 | rev5/task-board-validate-01.log; diagnostics are workspace-wide and none names this Task/Story |

The following early attempts failed and are not counted as passing evidence:
AST test compile attempt (exit 1, fixed import), first symlink-chain expectation
(exit 1, corrected to the specified refusal detail), and component byte-boundary
attempts 01–03 (exit 1, fixture construction corrected). The final reruns above
are the counted evidence. The first mode-oracle attempt had no captured process
exit and is not counted; mode-kind-oracles-02.log is the counted standalone
run. The optional harness --help invocation returned exit 2 because --help is
not supported; named mutant invocations ran successfully. All rev5 KILLED and
SURVIVED mutant outcomes have raw logs and process exits.

The full suite includes the previously committed reviewer tests, kept under
their original names; all passed in the exact-tree full run. No tmux process was
running during the no-tmux tests; the final process check found none.

## Initial Revision 6 validation before AST-taint extension

The candidate source/test tree is checkpoint `5a64077facb9f93357b9bbc8ebdba93069105715`
plus the uncommitted Story changes and rev6 additions. All commands were
foreground standalone processes with real exit codes. The configured full
Go test and coverage commands were split into sequential package groups to
stay below the shell-call limit; all 45 packages were covered. The configured
race command ran in four bounded groups (12+12+12+9 packages).

| Command / gate | Exit | Evidence |
|---|---:|---|
| `go test ./...` | 0 | `rev6-go-test-all-exact-01.log`; all 45 packages passed |
| Configured `go test -count=1 -v`, six sequential groups | 0 each | `rev6-config-go-test-group-01.log` through `-05b.log`; all 45 packages passed |
| Configured `go test -race -count=1 -timeout 25m`, four sequential groups | 0 each | `rev6-race-group-01.log` through `-04.log`; all 45 packages passed |
| Configured `go test -cover -count=1`, six sequential groups | 0 each | `rev6-config-cover-group-01.log` through `-05b.log`; all 45 packages passed |
| 17 configured fuzz commands, each `-fuzztime=100x -parallel=1` | 0 each | `rev6-fuzz-01.log` through `rev6-fuzz-17.log` |
| New PATH_MAX/wiring/transitive-guard tests, `-count=3 -v` | 0 | `rev6-determinism-count3-01.log` |
| Final focused PATH_MAX, production-entry wiring, and transitive guard rerun | 0 | rev6-focused-final-01.log |
| Registry re-derivation, Story acceptance-case edges, and §2.4 authority tests | 0 | rev6-registry-tests-final-01.log |
| `go test ./internal/tmuxserver ./internal/termbind -count=1 -v` | 0 | `rev6-package-tests-01.log` |
| `go build ./...`; `go vet ./...` | 0 each | `rev6-go-build.log`; `rev6-go-vet.log` |
| `GOOS=linux GOARCH=amd64 go build ./...` | 0 | `rev6-go-build-linux.log` |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 | `rev6-go-build-windows.log` |
| `GOOS=windows GOARCH=amd64 go vet ./...` | 0 | `rev6-go-vet-windows.log` |
| Configured gofmt gate | 0 | `rev6-gofmt-configured.log` |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | `rev6-tracecheck.log`; output is printed above and matches README |
| cataloggen adopted catalog check | 0 | `rev6-cataloggen-check.log` |
| JSON parse gate with `pipefail` | 0 | `rev6-json-parse.log` |
| `task-board validate` | 0 | `rev6-task-board-validate.log`; 253 workspace-wide diagnostics, none names this Task or Story |
| `git diff --check` | 0 | `rev6-diff-check.log` |
| README figure 70→71 plant; digest pin zero plant | 1 expected red each | `rev6-readme-coverage-plant-negative.log`; `rev6-digest-rederive-plant.log` |
| Importer outcome comparison versus trunk `0ca3e4c` | 0 | `rev6-importer-grid-01.log`; `closuregrid-rev6-0ca3e4c/`; 192 base rows, 223 candidate rows, 0 moved |

The exact `go test ./...` and the configured tests both pass, so no failed or
interrupted full-suite run is excluded. The rev6 candidate config remains
byte-equal to trunk `0ca3e4c`. All previously committed reviewer tests remain
under their original names and pass in the full suite. No tmux process is
required by the custody witnesses.

## Evidence and worktree

The current `TASK-260830-g0pcnt_producer-evidence.tar.gz` (684 KiB) contains the
updated results, conformance matrix, coverage map, mutation table, current
source snapshots, current delta logs, five isolated narrowing plants, neutral
control, prior sessquery race evidence, and the applicable importer/digest
evidence. The separate
`TASK-260830-g0pcnt_rev6-current-test-groups.tar.gz` (646 KiB) contains current
package-group logs for the configured normal, race, and coverage runs, all 17
fuzz logs, and the revalidation index. The earlier producer bundle and rev6
configured-group bundle remain separately attached. Both current archives are
below the 1 MiB limit.

Current trunk remains 0ca3e4c26e2b275212796657f785b9b450f6174e;
candidate remains UNCOMMITTED at Story checkpoint
5a64077facb9f93357b9bbc8ebdba93069105715. task-board.config.json is unchanged
from current trunk. No real Git index writes or task-board file edits were made.

## Current rev6 revalidation after path-derived-length guard extension

This is the latest validation state. Relative to the prior rev6 candidate, the
revalidation changed only `internal/tmuxserver/custody_walk_depth_unix_test.go`
and `internal/tmuxserver/mutant_harness.py`, then updated this task evidence,
`internal/tmuxserver/TRACEABILITY.md`, and the newest-first `LOGBOOK.md` entry.
No production Go implementation, registry, README, or task-board config changed.
The candidate remains uncommitted at checkpoint
`5a64077facb9f93357b9bbc8ebdba93069105715` on trunk
`0ca3e4c26e2b275212796657f785b9b450f6174e`.

The structural guard now carries path-derived values through `strings.Split`,
range indices over path components, numeric component counts, and
`[]byte(path)`. Three corresponding narrowings were added to the harness, on
top of rev6's `callee24` and filesystem-root cap. All five are measured below.

| Validation | Real exit | Evidence / disposition |
|---|---:|---|
| Configured `go test -count=1 -v`, package groups 01–10, all 45 packages | 0 each | `.temp/TASK-260830-g0pcnt/rev6-current/config-go-test-group-01.log` through `-10.log`; every package group passed on the current tree. |
| Monolithic `go test ./... -count=1 -v` attempt | 1 | `full-go-test-count1-verbose.log`; manually interrupted at the shell window with no test failure output. It is not counted green; the ten package groups above cover the full package set. |
| Configured `go test -race -count=1 -timeout 25m`, current package groups | 0 each | Current logs `config-go-race-group-01` through `-06`, `-07a`, `-08a`, `-08b`, `-09`, and `-10`; 44 packages were rerun, including changed `tmuxserver` tests. |
| `sessquery` race package | 1 current attempt; prior exact evidence 0 | The current isolated attempt reached the 10-minute shell limit and was interrupted at 9:48, with no test failure output. Reused unchanged-package evidence from `previous-evidence/.temp/TASK-260830-g0pcnt/rev6-race-group-03.log`: `sessquery` passed in 409.416s. This revalidation changed no `sessquery` source/test or production dependency, on the same Darwin/arm64 Go 1.25.5 environment. |
| Configured `go test -cover -count=1`, package groups 01–10, all 45 packages | 0 each | Current logs `config-go-cover-group-01` through `-06`, `-07a`, `-07b`, `-08a`, `-08b`, `-09`, and `-10`. |
| 17 configured fuzz commands (`-fuzztime=100x -parallel=1`) | 0 each | Current logs `config-fuzz-01.log` through `config-fuzz-17.log`. |
| Custody determinism rerun at `-count=3` | 0 | `config-determinism-count3.log`; PATH_MAX, entry wiring, transitive guard, mode/kind/owner, name, symlink, and permission tests. |
| `go build ./...`, `go vet ./...`, `GOOS=windows GOARCH=amd64 go vet ./...` | 0 each | `config-go-build.log`, `config-go-vet.log`, `config-go-vet-windows.log`. |
| Configured gofmt check; Linux and Windows builds | 0 each | `config-gofmt-check.log`, `config-linux-build.log`, `config-windows-build.log`. |
| Exact-tree `go run ./internal/traceability/cmd/tracecheck`; adopted catalog check; JSON parse with `pipefail`; `task-board validate`; `git diff --check` | 0 each | `config-tracecheck.log`, `config-cataloggen-check.log`, `config-json-parse.log`, `config-task-board-validate.log`, `config-diff-check.log`. Task-board validation found 252 workspace-wide issues; none names this task or Story. |
| No-tmux-process check | 0 | `no-tmux-process-check.log` reports `tmux_process_count=0`. |
| README measured-coverage pin test | 0 current; wrong-figure plant prior rev6 expected-red | Current `final-readme-coverage-pin-test.log` passes against current tracecheck output. The README plant log `rev6-readme-coverage-plant-negative.log` still demonstrates that a wrong figure fails. README and registry were unchanged in this revalidation; digest evidence remains `rev6-digest-rederive-plant.log`, pin `3664ab2fb166189545fea34a068a318af4f251689f29c92915fa185fefdedeea`. |
| Importer outcome grid keyed by `(package, entry, input)` | Prior rev6 evidence reused | Production files and runtime entries were unchanged. `rev6-importer-grid-01.log` remains applicable: base 192 rows, candidate 223, 192 shared, 0 moved, 31 candidate-only. |

### Rev6 path-length mutant evidence

Each harness mutant ran alone against `tmuxserver` and `termbind`; the listed
raw Go test process exited 1 because its named test failed under the narrowing.
Each narrowing has two counted runs. The range-index plant's first attempt
failed during linking with `no space left on device`; it is excluded, then two
isolated retries killed the plant. The applied line-count-preserving neutral
control survived with harness exit 0.

| Mutant | What the narrowing admits | Named failing test | Raw exits / harness outcome |
|---|---|---|---|
| `N-ancestor-walk-callee-depth-cap-24` | `custodyModeForPath` clears 0o022 after 24 separators | `TestCustodyAncestorWalkGeneratedDepthToPathMax` | 1, 1 / KILLED twice alone |
| `N-ancestor-walk-filesystem-root-depth-cap` | `isFilesystemRoot` stops after 24 separators | `TestCustodyAncestorWalkHasNoLengthDependentControlFlow` | 1, 1 / KILLED twice alone |
| `N-ancestor-walk-split-component-count-cap` | `len(strings.Split(path, "/")) > 24` admits the path | `TestCustodyAncestorWalkGeneratedDepthToPathMax`; structural test also fails | 1, 1 / KILLED twice alone |
| `N-ancestor-walk-range-index-cap` | A range index over split path components greater than 24 admits the path | `TestCustodyAncestorWalkGeneratedDepthToPathMax`; structural test also fails | 1, 1 / KILLED twice alone; first link attempt excluded as ERROR |
| `N-ancestor-walk-byte-length-cap` | `len([]byte(path)) > 24` admits the path | `TestCustodyAncestorWalkGeneratedDepthToPathMax`; structural test also fails | 1, 1 / KILLED twice alone |
| `C-control` | Applied comment-only, line-count-preserving control | None expected | SURVIVED; harness exit 0 |

The unchanged 355-row rev5 harness was run twice previously. The full 360-row
rev6 harness was not run; the five added path-length narrowings above each
have two isolated raw executions. This is the exact mutant evidence claimed.
