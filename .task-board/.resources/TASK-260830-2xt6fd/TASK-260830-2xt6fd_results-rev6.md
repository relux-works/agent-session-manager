# TASK-260830-2xt6fd developer handoff results

Status: **ready for review handoff**. The managed Story candidate remains
uncommitted at the recorded checkpoint for the handoff snapshot.

## Delivered behavior

- `gitsnap.AssembleProvisional` now drives real repository-local pack creation,
  isolated pack/index import, exact object inventories, raw/logical index
  agreement, blob descriptors, and workspace root/member/child closure.
- `(*tmuxserver.Lifecycle).CaptureGitWorkspace` is the production coordinator.
  It admits only a current `quiescing` instance with matching input/provider
  boundary receipts or `stopped`; it rechecks state and source identity after
  assembly and never releases, restores, or unlocks the instance.
- Source-change, path/partition, byte/digest, unsupported-input, crash/retry,
  and stop-point refusal cases are exercised through production entries.
- `UnixDialer.Dial` distinguishes a missing socket under an existing directory
  from a missing parent or failed parent read; unknown remains unknown.
- The rev2 gate matrix derives valid requests with a real `Runner`, `Assembly`,
  boundary body, current-incarnation quiesce receipt, and proven barrier. Each
  refusal test changes only its named gate input and checks a literal code and
  detail. A source AST census requires exactly one test row for each direct
  coordinator admission refusal site. Controls prove both allowed opening
  states are admitted. `N-capture-admits-parked`, all four gate narrowings, the
  stop-point transition narrowing, and the missing-parent dial narrowing were
  each killed alone twice; the census control was killed and the harmless
  comment control survived.
- `probe.go` keeps absent-parent `ENOENT` unknown under pinned SPEC v0.7.0 §4.C:
  only a successful status read proves absence. `TestUnixDialerMissingDirectoryIsUnknown`
  covers the production `UnixDialer.Dial` call site, and its one-case narrowing
  is killed alone.
- Transient `creating` is not persisted by the landed `InstanceStates` owner;
  its raw status observation projects to `unavailable`. The capture state test
  records this owner bound and asserts the owner's literal unavailable refusal
  instead of adding a second status path.
- The v0.7.0 ownership registry includes the three Story leaves. README's
  coverage pin matches the final tracecheck report. No CLI/doctor capture
  capability was added.

## Measured coverage

The six-row final-leaf acceptance matrix is driven **6 of 6 rows** through the
call sites listed in `coverage-map.md`. The brief supplied no surface table;
that is recorded as a brief gap, not as waived coverage. The handoff scope map
records the named out-of-contract clauses and each acceptance-criteria clause
that bounds them.

## Reviewer test retention

The tracked baseline has 3,347 distinct `Test*` names across Go files. The
candidate retains all of them; the candidate has 3,350 names and the measured
missing-name set is empty. This includes the hidden task-resource test files.
`TestContentNormativeWorkspaceBytes` moved between test files with its name
unchanged and passes when run alone. The two committed home-probe reviewer tests
remain in their original task-resource file; that exact file was materialized
temporarily into `internal/config` to run both original names, and both passed.
The temporary copy was removed. Audit counts, zero-missing output, and raw test
logs are included in the evidence archive.

Full tracecheck: exit 0; `contracts=64`, `normative_sections=36`,
`acceptance_cases=163`, `bindings=73`, `full=4`, `partial=11`, `sliver=13`,
`clauses_discharged=106/597`. README carries the same coverage pin.

## Mutant evidence

Across the recorded narrowing runs, 19 narrowing plants were killed by named
tests. The measured `N-assembly-closure-admits-one-missing-child` plant
survived: the malformed fixture is still rejected by the required repository
tree binding, so it proves no independent child-visitor gate. This bound is
not counted as independent evidence. The harmless comment-only `C-control`
survived as expected and is not a gate mutant. Full per-plant table, killer
names, survivor bound, and raw-log paths are in `coverage-map.md` and the
evidence archive.

Rev2 rework plants (each selected plant executed as an independent harness
row, twice):

| Mutant | Narrowed admission | Named test / result |
| --- | --- | --- |
| `N-capture-admits-parked` | Admits only `parked` at capture admission | `TestCaptureStopPointStateDomain` alone — killed; behavior exit 1, harness exit 0 |
| `N-capture-admits-checkpoint-proof` | Admits exactly `ax_checkpoint_boundary` at the provider-proof gate | `TestCaptureRejectsCheckpointBoundaryAsProviderProof` alone — killed; behavior exit 1, harness exit 0 |
| `N-capture-admits-foreign-quiesce-receipt` | Admits only an input-closure receipt from another incarnation while the provider boundary receipt stays current | `TestCaptureRejectsQuiesceReceiptFromAnotherIncarnation` alone — killed; behavior exit 1, harness exit 0 |
| `N-capture-admits-unproven-barrier-for-one-group` | Admits one unproven close-time quiescence barrier | `TestCaptureRejectsBarrierLostAfterAssembly` alone — killed; behavior exit 1, harness exit 0 |
| `N-capture-admits-stopped-to-quiescing-for-one-group` | Admits a stopped instance that becomes quiescing at the closing read | `TestCaptureStoppedMustRemainStoppedAtClose` alone — killed; behavior exit 1, harness exit 0 |
| `C-capture-census-unlisted-refusal-site` | Adds a second refusal beneath a gate that lacks a second matrix row | `TestCaptureCoordinatorAdmissionGateCensus` alone — killed; census control exit 1, harness exit 0 |
| `N-capture-restores-with-boundary-token-preserved` | Uses `restore` instead of the owner's `wait-safe-boundary` while retaining receipt/parser tokens | `TestCaptureGitWorkspaceQuiescedAssemblesAndRechecks` and `TestCaptureNeverTransitionsOutsideStopPointOwnerSet` — killed; behavior exit 1, harness exit 0 |
| `N-unix-dialer-missing-parent-admits-stale` | Maps only ENOENT with an absent containing directory to `stale` | `TestUnixDialerMissingDirectoryIsUnknown` alone — killed; behavior exit 1, harness exit 0 |
| `C-control` | Comment-only behavior-neutral source edit | `TestSocketPathDerivesOnlyFromRuntimeDir` passes — expected survivor, proving the harness reports a harmless survivor |

Each of the four reviewer-identified gates starts from the shared valid-request
builder and asserts both the literal `capability_unavailable` token and its
literal detail. Each narrowing changes only the gate predicate while leaving
the refusal branch and code/detail tokens in place; the harness runs the
named production-behavior test, not just the AST census. This attacks the
source census with token-preserving behavior changes. `TestCaptureAdmissionGateValidControls` and
`TestCaptureStopPointStateDomain` provide admitted controls for quiescing with
a current boundary and stopped. `TestCaptureCoordinatorAdmissionGateCensus`
enumerates refusal sites from the production AST and checks one row/test per
site. Exact per-run output and raw plant logs are in the two
`mutants-capture-rework-final-*` directories.

## Validation

All commands below ran as standalone processes. Logs are in
`.temp/TASK-260830-2xt6fd/` and included in the evidence archive.

| Command | Exit | Evidence |
| --- | ---: | --- |
| `go test ./internal/tmuxserver -count=1 -v` | 0 | `tmuxserver-full-03.log`; package reports 337.331s |
| `go test ./internal/gitsnap -count=1 -v` | 0 | `gitsnap-final-04.log` |
| `go test ./internal/tmuxserver -run 'TestCapture' -count=1 -v` | 0 | `tmuxserver-capture-final-01.log` |
| `go test ./internal/config -run '^TestProbeOSInputs(PreservesRealHomeFailure|CapturesRealHomeForPlatformDefaults)$' -count=1 -v` | 0 | Both original reviewer names pass from the exact attached test source; `reviewer-tests-materialized-01.log` |
| `go test ./internal/gitsnap -run '^TestContentNormativeWorkspaceBytes$' -count=1 -v` | 0 | Original name preserved after file move; `reviewer-test-preserved-01.log` |
| `env -u TMPDIR go test ./... -count=1` | 0 | All packages green on the same production/test/config identity before a subsequent README link sentence edit; `go-test-all-rework-final-02.log`. The post-edit README pin test is listed separately below. First run with task-root `TMPDIR` exited 1 on two temp-path-sensitive fixtures; the default-temp rerun passed (`go-test-all-rework-final-01.log`, `config-tempdir-recheck-01.log`, `tmux-socket-tempdir-recheck-01.log`). |
| `env -u TMPDIR go test ./... -count=1` (fresh run after README link edit) | 1 | `go-test-all-rework-final-03.log`; four packages (`resumesmoke`, `sessquery`, `tmuxserver`, `tracecheck`) hit Go's 10-minute package timeout while unrelated repository test/harness runs were concurrent. All other listed packages completed. This retry is reported as red. The only source delta since the earlier green exact functional suite was README wording/link text; the focused README pin test below passed on this tree. |
| `env -u TMPDIR go test ./... -cover` | 1, timed out | Coverage instrumentation hit Go's 10m test timeout in `tmuxserver.TestCustodyModeOracleAtProductionEntries` while other repo work was concurrent; other packages, including tracecheck, reported coverage. This is not counted green (`go-test-all-cover-rework-final-01.log`). |
| `env -u TMPDIR go vet ./...` | 0 | `go-vet-rework-final-01.log` |
| `env -u TMPDIR GOOS=windows GOARCH=amd64 go vet ./...` | 0 | `go-vet-windows-rework-final-01.log` |
| `env -u TMPDIR go build ./...` | 0 | `go-build-rework-final-01.log` |
| `gofmt -l internal` | 0 | Empty output, `gofmt-rework-final-01.log` |
| `git diff --check` | 0 | Empty output, `git-diff-check-rework-final-01.log` |
| `golangci-lint run --new-from-rev=HEAD` | 0 | `golangci-lint-rework-final-01.log`; no new delta findings |
| `golangci-lint run ./...` | 1 | `golangci-lint-01.log`; 136 repository-wide findings, including unchanged paths. Delta-specific lint is green; the full repository command remains red. |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | Exact current report: 64 contracts, 36 normative sections, 163 acceptance cases, 73 bindings, 106/597 clauses discharged; `tracecheck-rework-final-01.log` |
| `go run .../tracecheck -section 10.4` | 1, expected scope refusal | 21/25; clauses #3, #5, #15, #25 are out of contract |
| `go run .../tracecheck -section 12.1` | 1, expected scope refusal | 0/1; checkpoint publisher/linkage is out of contract |
| `go run .../tracecheck -section 12.2` | 1, expected scope refusal | 2/6; clauses #3–#6 are destination-materialization gates |
| `go run .../tracecheck -section 12.3` | 1, expected scope refusal | 2/5; materialization steps/matrix/network rule are out of contract |
| Two rev2 shipped-harness selections, each with nine selected rows | 0 each | Seven narrowing plants (parked-state, four admission gates, restore transition, missing-parent Dial), the census control, and harmless `C-control`; seven narrowings plus the census control were killed, and `C-control` survived. Summary and raw per-plant logs are in `mutants-capture-rework-final-01/` and `mutants-capture-rework-final-02/`. |
| Focused capture/dialer tests with `-count=3` | 0 | Nine named production-entry tests pass three times; `capture-rework-determinism-02.log` |
| README measured coverage plant | 1 expected-red | Scratch README pin `107/597` refused against tracecheck's `106/597` by `TestREADMEMeasuredCoverageMatchesTracecheckReport`; README restored byte-identically (`readme-coverage-plant-01.log`). |
| `go test ./internal/traceability/cmd/tracecheck -run '^TestREADMEMeasuredCoverageMatchesTracecheckReport$' -count=1 -v` after README edit | 0 | `readme-coverage-final-02.log`; exact working-tree README pin agrees with the candidate tracecheck report. |
| Registry digest re-derivation | 0 | JSON decoded and canonical digest independently re-derived as `bc46c7705670bc331811a32eb25a5712331fd11248c207a61c7544023f84107b`; `registry-digest-rederive-01.log` |
| Three-leaf registry edge audit | 0 | Decoded the v0.7.0 registry and checked all three acceptance case IDs, their section-clause references, and each referenced production path/declaration; 3/3, `registry-three-leaf-audit-01.log`. |
| Importer outcome comparison | 0 for matched cases; one expected-red base Dial observation (exit 1) / candidate exit 0; two malformed temporary test probes exited 1 at compile time and were corrected before comparison | Independent base and candidate comparisons for gitsnap, provhost, traceability, mismatched-self-digest `CheckIdentity`, and missing-parent `UnixDialer.Dial`; tracecheck CLI base/candidate also ran 0. Exact package/entry/input outcomes and all logs are in `TASK-260830-2xt6fd_importer-outcome-grid.md`. |
| `task-board.config.json` byte comparison | 0 | Compared with `HEAD` bytes; identical. `config-base.json` is in task scratch. |
| Scratch-index hygiene | 0 after retry | Separate `.temp/TASK-260830-2xt6fd/hygiene.index`; full candidate staged only into this scratch index, `diff --cached --check` clean, `write-tree=affce3fa49043fd53db6a8d67c181afd2bce20c9`. The real index remained unchanged (`git diff --cached --quiet` exit 0). One premature `write-tree` during the still-running scratch `git add` hit the scratch lock (exit 128); the completed add was followed by successful clean check and tree write. |

Earlier monolithic `go test ./... -v` attempts were not green: two pre-fix
runs exited 1 on the dialer classification, and a third attempt was interrupted
at the call limit (exit 143). A pre-fix full `tmuxserver` attempt also exited 1
because long test temp paths exhausted `sun_path` before custody checks. The
fixture now gets its directory from `t.TempDir()` and relocates that owned
directory under canonical `/var/tmp` with `t.Cleanup`; the final full
`tmuxserver` run and all coverage shards pass. Four broad mutation controls
exceeded their 240-second harness limit and are diagnostics only.

## Specification boundary and evidence map

The clause crosswalk and line ranges are in `spec-crosswalk.md`. Section bodies
for v0.5.0 → pinned v0.7.0 are byte-identical where compared. The final leaf
does not implement checkpoint publication (§12.1#1), Git destination
materialization (§12.2#3–#6; §12.3 steps 2–8), external provider transfer
response admission (§10.4#15), cross-platform runtime proof (§10.4#25), or
capture returning to `active`; the §4.C lifecycle vocabulary is closed. The
acceptance clauses bounding these rows are listed in `coverage-map.md`.

## Rev2 rework diff scope

Production capture/assembly behavior and its narrowing tests stay in the Story's
`internal/gitsnap` and `internal/tmuxserver` boundary. `internal/tmuxserver/probe.go`
is the explicit rev2 exception: the rework asks to justify or revert this status
read change; the pinned §4.C successful-status-only absence rule, direct
`UnixDialer.Dial` test, and its killed narrowing are cited above. The
`internal/provhost` delta changes comments/tests only: it replaces an
over-broad generic-verifier absence census with an input/output assertion at
`CheckIdentity`, and the same mismatched-digest input was rerun at base and
candidate. Production behavior is unchanged. Registry/traceability files,
README, and `LOGBOOK.md` are the explicit story-final documentation and registry
work required by the resume brief; mutant harness edits are in `tmuxserver`.

Supporting artifacts: `coverage-map.md`, `axis-inventory.md`,
`importer-outcome-grid.md`, and `spec-crosswalk.md`. The archive includes these
documents, the final logs, selected historical failure logs, mutation result
JSON, and one raw behavior log per plant. Archive size is kept below 1 MiB.

## Revision 2 exact-tree follow-up — 2026-09-24

- Current `internal/tmuxserver/mutant_harness.py` defines 368 rows. The broad pass ran once in bounded slices; each slice exited 0, with the corrected `120:140` slice used and the two superseded failed attempts retained as diagnostics. Raw per-plant logs, named test commands, type (narrowing vs arm-delete), observed subprocess exit, and survivor bounds are in `TASK-260830-2xt6fd_mutant-harness-results.md`. It reconciles 368/368 logs: 366 KILLED, 2 SURVIVED, no missing or NOT_APPLIED rows. The `D-attach-entry-deadline-shadowed` survivor is a supplementary shadowed narrowing, not gate evidence; its bound and the independent deadline mutants are recorded in the table. `C-control` is the harmless expected survivor. The four admission-gate narrowings, parked-state gate, restore transition, census control, and Dial classification selection were separately run twice with their named tests; this targeted repeat does not mean the entire 368-row broad manifest ran twice.
- The broad slices 0:320 ran in the Story worktree; 320:368 ran in a copied candidate. All 13 changed paths in `internal/tmuxserver`, `internal/termbind`, `internal/terminstance`, and `internal/axpane` were byte-identical between the live worktree and copy. The candidate snapshot tree recorded in `full-mutant-run/candidate-tree.oid` is `5966b53498062b83193e1ff996be3e38b485bed6`.
- Exact-tree package reruns: `go test ./internal/resumesmoke -count=1` exit 0 (324.229s); `go test ./internal/sessquery -count=1` exit 0 (470.884s); `go test ./internal/traceability/cmd/tracecheck -count=1` exit 0 (337.781s); `go test ./internal/tmuxserver -count=1` exit 1 at 600.341s in `TestCustodyPathComponentByteLengthsAtProductionEntries` cleanup while an unrelated package suite was active; the isolated named test then passed in 90.473s, and a clean full package rerun after that suite exited 0 in 320.735s. Direct logs are `go-test-{resumesmoke,sessquery,tracecheck}-rerun-01.log`, `go-test-tmuxserver-rerun-01.log`, `go-test-tmuxserver-path-component-rerun-01.log`, and `go-test-tmuxserver-rerun-02.log`.
- The full `env -u TMPDIR go test ./... -count=1` on this exact tree exited 1: four packages timed out during concurrent repository-wide work. It is reported red, not converted to green by the sequential package reruns. The coverage-instrumented full command also timed out. Earlier full-suite green evidence remains separately recorded and is not claimed as this exact-tree rerun.
- Measured final-leaf acceptance ratio remains 6 of 6 rows driven through their named production capture call sites; `coverage-map.md` lists each test and narrowing mutant. The brief supplied no surface table; that gap is explicit. Out-of-contract rows and acceptance clauses remain enumerated there.

## Rev3 refreshed candidate handoff — 2026-09-25

This section supersedes earlier refreshed-tree figures in this document. The
importer source/test snapshot is scratch-index tree
7bebe7213a4c3c71410c591b14b2761d2e8d4bdc on trunk
5b7876be29cf578ff02583660d77fc10944e855c. The final handoff snapshot tree,
which also contains this LOGBOOK entry, is recorded in
handoff-candidate-tree-final.oid (60d988f86ee2097cdc41ee723768adf04471f02c). The candidate is
uncommitted and exactly two checkpoint commits ahead of origin/main.

- Refresh invariant: comparing the pre-refresh candidate tree
  6040c3815b61c344ebbe821f29a947d35292a877 against the old base yields 2,162
  Story-changed paths. Of 2,263 paths changed by trunk since 7654d7c, 142 are
  trunk-only. Every one is present with the same blob as origin/main in the
  candidate tree; mismatches=0 and missing=0. Machine result:
  trunk-only-invariant-handoff-02.json. The task-board config blob remains
  61bf40ce095916f26199f4373f2eb5d9b6403517. Both introduced checkpoint commits
  verify with Ivan Oparin's configured SSH signing key.
- Registry: the v0.7.0 registry digest was re-derived as
  b7a3861fa8693ee451ae6742d9dd5de85268497191eed6c305887bec7f086737 over the
  201,812-byte canonical payload. Current tracecheck exits 0:
  contracts=64, normative_sections=36, acceptance_cases=166, fixtures=33,
  compatibility_contracts=55; bindings=75, full=4, partial=12, sliver=14,
  unevidenced=41, unmeasured=4, unowned=7, clauses_discharged=113/622.
  README pin matches. The wrong-figure README plant is expected-red (exit 1)
  and the restored/current README pin test passes (exit 0). All three Story
  leaf rows decode to their clause edges and production owners.
- Rev3 gate census: the AST census finds 22 direct captureUnavailable refusal
  sites in the coordinator. Every site has exactly one literal-detail matrix
  row; the census rejects an added unlisted site, an orphan row and a
  non-literal detail. Both source control plants were killed alone twice on the
  current tree; the harmless comment control survived as expected. The 22 site-specific narrowing mutants were each run alone twice on the current tree: 22/22 killed in pass 1 and 22/22 killed in pass 2, with no survivor or NOT_APPLIED result. This includes fresh, current-tree reruns of TestCaptureAdmission_StoppedBoundaryBodyForbidden and TestCaptureAdmission_IncarnationStableAcrossCapture. The focused current-tree admission/domain tests pass three times (see determinism-capture-admission-count3-handoff-01.log).
- Exact-tree standard Go suite: the direct go test ./... -v -count=1 process
  ended with exit 1 when interrupted at the single-call time limit after 44 of
  47 package outputs. This is not reported as a green monolithic command.
  The three remaining package suites were run directly against the same
  candidate tree and passed; gitsnap had also passed its focused exact-tree
  suite. Together, all 47 candidate packages have green direct package-suite
  evidence, including the current traceability and tracecheck reruns. The
  refreshed exact-tree all-package coverage command was not run. A changed-
  package coverage shard passed (exit 0): gitsnap 84.4%, tmuxserver 88.7%,
  provhost 85.7%, traceability 87.2%, and tracecheck 88.5%.
- Exact-tree validators rerun after the final source/test edits: native
  go vet ./... exit 0, Windows/amd64 go vet ./... exit 0, go build ./... exit
  0, tracecheck exit 0, traceability tests exit 0, README coverage-pin test
  exit 0, formatting and diff checks exit 0. The final capture admission/
  census/domain set with -count=3 exits 0. A broader TestCapture -count=3
  attempt timed out in crash-retry under unrelated host test load; the full
  tmuxserver package suite itself passed in a separate bounded run. A broad
  gitsnap -count=3 attempt was stopped at the command bound; the assembly and
  content/capture families each passed -count=3 separately.
- Mutant scope: the earlier 368-row shipped-harness result is attached and
  reconciles 366 KILLED plus the documented two survivors, but the entire
  368-row manifest was not repeated after refresh. This rev3 rework instead
  reran all 22 capture refusal-site narrowings twice on the current tree, with
  per-plant raw behavior logs in the two pass directories. The unrelated prior 368-row
  results are not represented as a rev3 full-manifest rerun.
- Importer comparison on base 5b7876b and candidate tree 7bebe721: 7 package
  suites, 4,098 shared named-test keys, and one shared-key movement:
  tmuxserver TestUnixDialerMissingDirectoryIsUnknown fail -> pass. That move
  is the intended §4.C unknown-on-failed-parent-read correction. All other
  shared keys retained their status. The newly added gitsnap package and new
  production capture/assembly entrypoints are reported as candidate-only;
  the base package did not exist. The refreshed importer comparison and grid
  carry every moved class. The tracecheck owner-renaming test kept its original
  name while its input changed to the merged AssembleProvisional binding.
- Current scope ratio is 6 of 6 final-leaf acceptance rows driven through the
  production entries named in coverage-map.md. The brief supplied no surface
  table; this remains an explicit brief gap. Non-disruptive return-to-active
  capture, checkpoint publication, destination materialization, external
  provider-transfer response admission, Windows runtime DACL proof, and a
  future export/replication consumer are listed with their limiting clauses
  in coverage-map.md as out of contract.

### Run write-boundary incident

At 2026-09-25 03:16:46–03:16:48 +0400, four importer inventory text outputs
were initially created in the sibling main-checkout scratch directory
/Users/iv/Developer/ReluxWorks/agent-session-manager/.temp/TASK-260830-2xt6fd/
instead of this Story run worktree. The files were
importer-refresh-base-imports.txt, importer-refresh-base-packages.txt,
importer-refresh-candidate-imports.txt, and
importer-refresh-candidate-packages.txt. They were moved into this run's
.temp/TASK-260830-2xt6fd/; at 04:27:23 +0400 the sibling source paths were
absent and the worktree copies were present with the original creation times.
This was an agent boundary error and is disclosed as run_wrote_outside_worktree.
No tracked source file or control-root/board file was written outside the
worktree; board mutations were made only through task-board CLI.

## Rev6 rework evidence

This rework answers `repeat-of: capture-gate-disjunct-unmeasured` from rev5. The AST census now splits every nested `||` guard into clauses, rejects a non-literal detail or an unlisted/extra clause, and requires a bijection with the 27-row matrix in `TASK-260830-2xt6fd_capture-admission-matrix-rev6.md`. Each of the 27 clause narrowings was applied and killed by its named row test alone twice. The unlisted-site, non-literal-detail, and added-disjunct census controls were also killed twice. `C-control` is an applied harmless control and survived. The additional provider-boundary empty-timestamp check is outside the `captureUnavailable` clause set: `TestCaptureOwnerBoundaryEmptyTimestampRepeatOf` retains the reviewer test `TestCapturePropagatesOwnerBoundaryTimestampIntegrity`, and `N-capture-admits-empty-provider-boundary-timestamp` was killed alone twice. This is a repeat of rev5's `capture-receipt-clause-unmeasured` class at the provider owner boundary.

Current refresh evidence: the Story was refreshed onto trunk `eb12183056eb86dba663540518c6ac8f79f56a87`, producing signed replay commits `1ed5318c85047d494426cb14be6678e76d2e8098` and `1121b4ab54709a9966fbbcc359230d15b0e7592b`; both passed `git verify-commit`. The mechanical old-base invariant compared all 2,547 trunk-only paths and found zero mismatches. `task-board.config.json` is byte-identical to refreshed `origin/main`; the candidate remains uncommitted. A fresh fetch immediately before handoff still reported `origin/main=eb121830...`, so no second refresh was needed.

The merged v0.7.0 ownership registry has canonical SHA-256 `b3c19a372b72d48f01757bc209162e8da7d3fd4a56908169fc18c7d9009314d2` and includes the three Story leaves. Current-tree `tracecheck` exited 0; the README coverage pin test exited 0, while its wrong-number scratch-copy plant exited 1 as expected. Measured final-leaf AC coverage remains **6 of 6 rows** through the production call sites in `coverage-map.md`. The assigned brief contains no surface table; that remains an explicit brief gap. Out-of-contract rows and their bounding AC clauses are listed in `coverage-map.md`.

Validation on the refreshed candidate: `go vet ./...` exit 0, `GOOS=windows GOARCH=amd64 go vet ./...` exit 0, `go build ./...` exit 0, and `git diff --check` exit 0. The requested full `go test ./... -v` did not complete: after the bounded run exceeded ten minutes it was terminated with SIGTERM (observed exit 143). This is incomplete evidence, not a passing suite. The importer comparison artifact is from base `5b7876b` to the pre-refresh candidate tree `087ac14`; because the candidate was subsequently refreshed, that comparison is not claimed as an exact post-refresh importer rerun.

Determinism follow-up: the first broad admission `-count=3` exited 1 because the new owner-timestamp regression test name matched the census's matrix-row prefix without being a `captureUnavailable` clause. I renamed that separate test to `TestCaptureOwnerBoundaryEmptyTimestampRepeatOf` (the original reviewer test remains intact); the census plus renamed owner test then passed together at `-count=3` (exit 0, `rev6-capture-census-rename-count3.log`). The provider timestamp test itself also passed at `-count=3` before the rename. The first broad log is retained as a diagnostic and not presented as green.
