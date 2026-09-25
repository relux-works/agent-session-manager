# TASK-260830-3m7m7w — CR1 F1 rework, revision-2 producer evidence

Ready for review. RUN-260908-5dde57; GOAL-260908-deaf77 revision 1,
role_handoff; resolved scope TASK-260830-2bnr39 and TASK-260830-3m7m7w.
The exact objective/scope are saved in goal-checkpoint.json. No directives were
recorded at the latest checkpoint; no sibling runs or parent-goal mutations.

## Review and ownership

Prior reviewer RUN-260907-68f937 requested changes on CR1, F1 P1,
failed-read-as-absence, repeat-of:none. Read its verdict and actual probes first.
Original candidate f219517ac93cf446bd6ae70c7ebd3a86eb2c85fa was reproduced as
17/17 matching initial task files. Current delta changes six existing files and
adds content_policy_test.go; all other 2927/2927 prior tree files, modes and
symlink targets match. source-manifest.json binds all 19 task candidate files;
rework-only.patch isolates the rework from CR1. No manual commit/index/branch
mutation. HEAD remains d888cd576f5962eb258e40ca487f7711b2215610; real index
SHA-256 bfb3eeb21c83daee474a42adb7b143c77f6ce0f8455fe3292ed6cc8d580e88ba.

Scoped predecessor TASK-260830-2bnr39 is already integrating with 18/18 checked
items and CR-TASK-260830-2bnr39-2 checkpointed. Its accepted tree is
935f624e45088a064de04945cbe4bd86d1b37d1b; reviewer RUN-260907-428f33.
Current worktree-status.json, predecessor-board.json and prior reviewer verdict
plus checkpoint establish this advanced role state. Preserve that acceptance;
no regression to to-review or new predecessor delta is requested. Prior full
predecessor review/mutation evidence is accepted, not independently recreated;
its candidate-resident production tests were rerun here with the full suite.

## Correction and measured F1 evidence

Capture → capture → contentState.capture now calls Runner.Run for
`git ls-files --others --ignored --exclude-standard --directory -z` and assesses
error, exit status and stderr together. Any diagnostic refuses GateContentRead
with nil Snapshot before selecting files. This owns completeness at the content
policy boundary, without parsing paths/warning text or changing shared runOK
and private-index execution. Missing optional policy files retain Git semantics.
All diagnostics are conservatively unproven; no benign-warning allowlist is
claimed. README and the scoped specification/AC mapping state this limit.

- Baseline regression command: exit 1 (expected failure), 2/2 real policy sources
  and 3/3 injected diagnostics expose the original missing refusal.
- Corrected focused command: exit 0, three repetitions; 6/6 actual successful-exit
  permission warnings refused. Both info/exclude and core.excludesFile execute
  missing/readable/unreadable/restored controls, nil Snapshot assertions and
  original-index byte preservation. TestContentExcludePolicyReadCompleteness
  drives real ExecGitRunner through Capture. No permission witness skipped here.
- TestContentIgnoreCensusWarningRefuses drives Capture with empty and partial
  successful output plus diagnostics and unclassified diagnostic text, checks
  that the production census is reached, refuses, then retries with real Git.
- N-read-admits-success-diagnostic compiles (exit 0), preserves the gate while
  admitting exit-0 diagnostics, and fails both named tests (behavior exit 1).
  TestContentReadFailureIsNotAbsence still passes (positive-control exit 0).

## AC and gates

**7 of 7 scoped AC rows driven**; ac-driver-audit.json checks every named driver
in the source mapping against actual PASS lines, not just presence. Exact
production call sites and test names follow:

| AC row | Production call site | Named passing drivers |
| --- | --- | --- |
| Untracked bytes, with staged/working bytes distinct | Capture → contentState.capture → Guard.Open → blob → ObjectStore.PutBlob | TestContentCaptureSelectionAndBytes; TestContentNormativeWorkspaceBytes |
| Ignored files only through classified project/provider includes; complete ignore census | Capture → contentState.validate/capture → Runner.Run | TestContentIgnoredDefaultAndPolicyRefusal; TestContentNestedDetachedSubmodule; TestContentIgnoreCensusWarningRefuses; TestContentExcludePolicyReadCompleteness |
| Symlink targets and containment | Capture → contentState.capture → safeSymlink; canonicaljson.CheckManifestEntries | TestContentSymlinks; TestContentSymlinkChainsAndExcludedTargets |
| Submodule state and repository boundaries | Capture → contentState.submodules → capture (recursive) | TestContentSubmodulePointerStates; TestContentSubmoduleUninitializedAndRefusals; TestContentNestedDetachedSubmodule; TestContentSubmoduleCountBounds; TestContentRelativeSubmoduleURL |
| Large blobs and protocol size refusal | Capture → contentState.blob → installBlob | TestContentLargeBlobChunksAndLimit; TestContentBlobInstallFailureAndRetry |
| Sparse/linked worktrees | Capture → readRepository/readFeatures → contentState.capture | TestContentSparseLinkedWorktree; TestContentSparseReadFailure; TestCaptureLinkedWorktreeSubdirectory |
| Mandatory exclusions and explicit owner policy | Capture → contentState.validate/excluded; submodules exclusion recheck | TestContentCaptureSelectionAndBytes; TestContentLocalPathOwnerExclusions; TestContentExcludedSubmoduleCannotBypassPolicy |

The tests reside in the managed candidate and await the orchestrator's signed
checkpoint after independent review; the explicit no-manual-commit assignment
controls the checklist's “committed test” wording. Full scoped mapping and bounds
remain internal/gitsnap/testdata/content-acceptance.md. 6/6 content registry gates
have selected-member narrowing witnesses, plus exclusion/consistency call sites
and the shared entry-count bound; not every interior predicate is proven.
12/12 narrowing vectors killed; known-bad control killed; one neutral survivor
has its explicit bound. See TASK-260830-3m7m7w_mutant-table-rev2.md for the required
mutant/narrowing/named-failure/survivor-bound table and mutations/results.json for
all compile and behavior exits. Token-preserving path mutant runs the full
behavioral suite. No mutant is counted as killed without named test failure.

## Validation executed by this run

macOS arm64; Go 1.25.5; Apple Git 2.50.1. Direct standalone processes with real
exit codes; no tee pipeline. validation-exits.json and publication-quick-exits.json
record each command and log. The configured JSON pipeline used pipefail.

| Command | Exit | Log |
| --- | --- | --- |
| `go test ./internal/gitsnap -run 'TestContent(IgnoreCensusWarningRefuses\|ExcludePolicyReadCompleteness)$' -count=1 -v` | 1 | baseline-red.log |
| `go test ./internal/gitsnap -run 'TestContent(IgnoreCensusWarningRefuses\|ExcludePolicyReadCompleteness\|ReadFailureIsNotAbsence)$' -count=3 -v` | 0 | f1-green.log |
| `go test ./internal/gitsnap ./internal/canonicaljson -count=1 -v` | 0 | targeted.log |
| `go test ./... -v` | 0 | all-tests.log |
| `go test ./... -cover` | 0 | coverage.log |
| `python3 .scripts/gitsnap-mutations.py --vectors internal/gitsnap/testdata/content-mutations.json --out /Users/iv/Developer/ReluxWorks/agent-session-manager/.temp/TASK-260830-3m7m7w/rework-rev2/mutations` | 0 | mutations.log |
| `go build ./...` | 0 | build.log |
| `go vet ./...` | 0 | vet.log |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | tracecheck.log |
| `GOOS=windows GOARCH=amd64 go vet ./...` | 0 | windows-vet.log |
| `GOOS=linux GOARCH=amd64 go build ./...` | 0 | linux-build.log |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 | windows-build.log |
| `test -z "$(gofmt -l $(git ls-files --cached --others --exclude-standard -- '*.go'))"` | 0 | format.log |
| `go run ./internal/catalog/cmd/cataloggen -metadata internal/catalog/catalog.v0.5.0.json -contracts internal/specpin/v0.5.0.lock.json -output internal/catalog/catalog_gen.go -check` | 0 | catalog-check.log |
| `git diff --check` | 0 | diff-check.log |
| `python3 /Users/iv/Developer/ReluxWorks/agent-session-manager/.temp/TASK-260830-3m7m7w/rework-rev2/publication-quick.py` | 0 | publication-quick.log |
| `go test ./... -race -count=1` | 0 | race.log |

Full test/coverage/race logs each show 24/24 packages passing. Coverage: gitsnap
83.4%, canonicaljson 97.1%. Full no-count test/coverage commands can reuse
unchanged-package cache; relevant package tests and race explicitly disable it.
The managed runner, after this producer session returns, constructs the CR
and records candidate-bound publication checks. The handoff command itself
checks role status/checklist/outcomes; its output does not prove CR publication.
The exact configured no-cache test and coverage commands are additionally run
by this producer, with their own logs and exits recorded below.
13/13 configured fuzz smoke commands exit 0 with observed fuzz-engine output;
this does not prove that every target reached mutation beyond baseline seeds.
All configured short checks ran (including board validation), plus Windows vet.
Linux and Windows builds are compile-only evidence, not native behavior.
The verbose suite records five pre-existing skip rows (one debug census helper,
four malformed-member examples); no new F1 test skips. Coverage output alone
is not a skip census. tracecheck reports 17/428 discharged clauses, not full
specification implementation. Exact pinned v0.5.0 SPEC hash is verified.

## Durable state, limits and role acceptance

The existing TestContentCrashAfterInstall ran in the full suite: intentional
child exit 73 before result, then retry with verified object reuse. No new
publication/storage algorithm is introduced. A refused content capture can
leave inert verified objects, but returns no snapshot/manifest. Broad raw-index,
object-pack, root/child publication, end-to-end materialization and quiescence /
final race closure stay TASK-260830-2xt6fd. Shared-index auxiliary timestamp and
hard-kill cleanup bounds remain accepted predecessor limits. Full successful
128 GiB transfer, all grammar/depth combinations and arbitrary adversarial
filesystem schedules are not claimed. No CLI/doctor capture capability is added.
Permission witnesses explicitly skip if mode 000 remains readable or on Windows;
this run proves the two actual sources on the stated native host only.

The task-scoped outcome is attached before managed developer handoff. The
handoff result owns the to-review transition. CR tree/revision and its
publication transcript are produced by the managed runner after session return;
no published CR2 tree is asserted from the handoff output alone. Independent review is still
required. No accept_cr, checkpoint, commit, push or landing by this producer.

## Publication-command audit and handoff result

All 26/26 configured publication commands executed by this producer with exit 0;
publication-command-audit.json maps the exact command slots to logs/exits.
`go test ./... -count=1 -v` and `go test ./... -cover -count=1` each additionally
exit 0 with 24/24 packages, independent of the earlier cache-eligible commands.
The local board validation slot uses the explicitly required control board path.
The source manifest still matches 19/19 files after verification.

The developer handoff command exited 0 and returned to-review, 19/19 checklist
items, and the three new task-scoped outcomes. handoff.json contains the exact
receipt. TASK-260830-2bnr39 remains accepted/checkpointed, integrating, 18/18;
handoff-checklists.json and worktree-status.json carry the scoped evidence.
This establishes the producer role handoff; publication of the managed CR is
the runner's post-session action, not an already-observed CR2 tree.
