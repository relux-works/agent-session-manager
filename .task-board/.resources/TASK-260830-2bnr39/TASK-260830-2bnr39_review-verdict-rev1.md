# TASK-260830-2bnr39 — review verdict, CR revision 1

Verdict: **changes_requested**. Route: **to-dev**. `repeat-of: none` for every finding below.

## Candidate and scope

- CR: `CR-TASK-260830-2bnr39-1`, revision `1`.
- Base: `7aa151a9c31071bfab190fd8ae8259f502f2ebbc`.
- Candidate tree: `2a7575a9b8b5983308fe8769c88a4006858580cf`.
- Supplied patch SHA-256: `12551886962860bd3049ef940e5da18292ce4bc1db02effa4adbdf9124cbf20c`.
- Delta: present; 16 paths. All 16 working files were byte-compared with the candidate archive before review. HEAD equals the base; `git rev-list --count HEAD..main` returned 0 against the existing local main ref. No claim is made about a freshly fetched remote main.
- Normative source: `relux-works/agent-session-manager-spec@v0.5.0`, commit `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`, §§10.2–10.4 and 12.1–12.3. Embedded SPEC.md SHA-256 verified against its lock: `562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a`.
- Review environment: macOS arm64, Go 1.25.5, Apple Git 2.50.1. Producer skill resolved from the main checkout's Curator-managed `.codex/skills/go-testing-tools/SKILL.md`; this isolated worktree has no installed adapter directory.
- Work ran sequentially without delegation. Product files were not modified. Baseline tests used an exact candidate archive; mutants and new probes used separate disposable archive copies under the task-scoped `.temp/` directory. Git fixtures never contacted a remote.

## AC coverage, checked before code inspection

**8 of 8 AC rows driven** by named candidate-resident tests through `Capture`; this ratio measures existing drivers, not acceptance of all behavior in each row. Tests are in the snapshotted candidate, intentionally uncommitted under this Story workflow.

| AC row | Production call site | Named candidate tests | Review result |
| --- | --- | --- | --- |
| Repository identity | Capture → readRemotes → deriveIdentity | TestCaptureLiveBranchRepository; TestRemoteFetchPushMapping | Baseline passes; remote domain incomplete (F6) |
| HEAD/ref, detached/unborn | Capture → readHead | TestCaptureLiveBranchRepository; TestCaptureLiveHeadModes | Baseline passes; read failure and ref race bypass (F2/F4) |
| Worktree metadata | Capture → readRepository | TestCaptureLiveBranchRepository | Baseline passes; target cwd/root mismatch (F3) |
| Index stages 0–3 | Capture → parseIndexEntries | TestCaptureLiveConflictStages; TestCompatConflictStages | Baseline passes; subtree omission (F3) |
| Index flags | Capture → parseIndexRecord/debugFlags | TestCaptureLiveIndexFlags; TestCompatFSMonitorBit | Baseline passes; fsmonitor is scripted only |
| Staged deltas | Capture → parseDeltas, cached stream | TestCaptureLiveBranchRepository | Baseline and added binary-rename control pass |
| Unstaged deltas | Capture → parseDeltas, worktree stream | TestCaptureLiveBranchRepository; TestCaptureLiveMissingTrackedPath | Baseline and added binary control pass |
| File modes | Capture → statIndexPaths/statDeltaPaths | TestCaptureLiveExecBitAndSymlinkKind | Baseline passes; wrong path stats from subdirectory (F3) |

Capture is an internal library entry at this leaf boundary. No other production package currently imports gitsnap; this review does not claim an end-to-end CLI checkpoint or completed workspace closure. Exact fixture entries and existing canonical wire-validation ownership remain separate, as stated in the producer outcome.

## Findings

### F1 — P1: inherited environment defeats read-only execution and changes the index

`repeat-of: none`

Location: `internal/gitsnap/runner.go:68`.

Shape: **bypass path around the check / capability claim that does not reproduce**.

The runner prepends `GIT_OPTIONAL_LOCKS=0` before `os.Environ()`. When the environment already contains `GIT_OPTIONAL_LOCKS=1`, the inherited value wins. `TestReviewRunnerLockOverride` invokes the production runner with a disposable executable and observes `1`. More seriously, `TestReviewLockOverrideMutatesIndex` uses real Git: replace a tracked file with identical bytes to invalidate the index stat cache, call Capture with inherited locks=1, and compare `.git/index` before/after. Capture rewrites the durable index and then returns `GateConsistency: index moved during capture`.

The README/doc.go/Capture no-durable-mutation claim, used to justify omitting write crash evidence, is therefore false. Enforce the runner-owned lock setting against inherited duplicates and add the real byte-preservation regression. Keep the implementation read-only; no new durable transaction is requested.

### F2 — P1: consistency checks admit changed HEAD/ref and changed index version

`repeat-of: none`

Location: `internal/gitsnap/snapshot.go:904` (`recheckConsistency`; called from Capture).

Shape: **bypass path around the check; property inferred from a proxy signal**.

`TestReviewHeadRefRace` switches HEAD from `refs/heads/main` to `refs/heads/other` at the same commit immediately before the closing HEAD read. The fault fires exactly once; Capture succeeds with the stale main ref. `TestReviewIndexVersionRace` rewrites the real index from v2 to v4 before the closing ls-files read; Capture succeeds with stale version 2. Both drive Capture with a thin interception wrapper around ExecGitRunner; they mutate only a disposable fixture.

The sentinel compares the OID and `ls-files --debug` rendering, not the complete captured HEAD state or index state. §12.3 says to fail if HEAD or index changes; these are captured fields inside this leaf's scope, not omitted sibling blob/content features. Compare adequate state for HEAD mode/ref/OID and index version/identity, and cover these same-OID/same-logical-entry cases in committed tests. Preserve the sibling ownership boundary for full content capture.

### F3 — P1: subdirectory invocation silently drops index entries and records the wrong cwd

`repeat-of: none`

Locations: `internal/gitsnap/snapshot.go:186` (ls-files in caller dir), `:197` (root-relative stat), `:308` (os.Getwd).

Shape: **capability claim that does not reproduce**.

`TestReviewSubdirectoryCapture` creates three tracked paths, `AGENTS.md`, `README.md`, and `sub/inner`, then invokes Capture on `repo/sub`. It succeeds with only index path `inner`, losing the other two files and the `sub/` prefix. It then reports `inner` missing because stat interprets that spelling relative to the repository root. Worktree.CWD is the Go test process directory in the reviewer copy, unrelated to the captured repo/sub directory.

The API accepts a repository working directory without a root-only precondition. §12.1 requires the repository-relative cwd and staged index state. Resolve capture commands to a consistent repository root while preserving the actual requested cwd; ensure all index paths remain repo-relative. Add a subdirectory-versus-root comparison through Capture.

### F4 — P1: failed reads are silently converted to absent/false facts

`repeat-of: none`

Locations: `internal/gitsnap/snapshot.go:341` (readHead), `:388` (readUpstream), `:853` (feature bool helper), `:865` (required-filter list).

Shape: **absence and failure to read conflated**.

Through Capture, final probes inject fatal exit=128 responses: a symbolic-ref I/O failure is reported as detached HEAD (`TestReviewSymbolicRefReadFailure`); an upstream read I/O failure becomes null upstream (`TestReviewUpstreamReadFailure`); a failed core.symlinks read becomes false (`TestReviewFeatureReadFailure`); a failed required-filter census becomes an empty list (`TestReviewRequiredFilterReadFailure`). Scripted faults occur at the real production Runner seam; successful baseline reads and consistency rechecks remain present.

Handle the command-specific legitimate absence cases separately from fatal, transport, malformed, or partial reads. In particular, do not assume every upstream failure means no tracking branch: missing upstream and fatal reads can both use a nonzero code, so use enough authoritative evidence to distinguish them. Add negative tests for each fallback-bearing reader. A separate real malformed-config probe already refuses earlier in Git (`TestReviewMalformedConfig`); that passing control does not cover mid-read failures.

### F5 — P2: feature metadata misrepresents valid Git defaults and boolean spellings

`repeat-of: none`

Locations: `internal/gitsnap/snapshot.go:861`, `:873`.

Shape: **capability claim that does not reproduce**.

With system/global config explicitly disabled, `TestReviewSymlinkDefault` stages a real symlink in a fresh Git repo whose core.symlinks is unset. Capture reports `Features.Symlinks=false`, although Git is using its symlink behavior. `TestReviewRequiredFilterBoolean` sets `filter.demo.required=yes`, verifies `git config --bool --get` returns true, then observes Capture omit demo entirely. The current parser only recognizes literal `true`.

These fields are actively produced by this leaf and correspond to the closed GitFeatures type, even though filter blobs/materialization belong to siblings. Capture effective/default values and Git boolean syntax correctly, or refuse when the fact cannot be established. Do not encode unknown as false. Add tests that isolate ambient Git config.

### F6 — P2: the closed GitRemote name bound is not enforced

`repeat-of: none`

Location: `internal/gitsnap/snapshot.go:450`.

Shape: **absent validation treated as satisfied**.

`TestReviewRemoteNameBound` creates a real remote with a 129-character name and a valid sanitized URL. Git accepts it and Capture succeeds. The pinned §10.4 GitRemote requires name:string[1..128]; only the number of remotes and their URLs are currently validated. Add an exact name-bound check at Capture's read boundary, acceptance at 128, refusal at 129, and a narrowing mutant. Reconcile the broad “every string validated” claim with the complete declared output domain.

### F7 — P2: required narrowing proof is incomplete and one producer control was never executed

`repeat-of: none`

Locations: producer `TASK-260830-2bnr39_gitsnap-evidence.md`; captured producer harness `producer-mutant-harness.py`; `internal/gitsnap/census_test.go:151`.

Shape: **unmeasured evidence reported as measured / existence census substituted for behavior coverage**.

The live checklist demands a narrowing mutant for every gate. The producer explicitly reports **13 of 16 gates**, exempting GateNotRepository, GateHeadRef, and GateUpstreamRef. An open environment class can still be narrowed to one error case, and scalar-helper coverage does not by itself prove each Capture call site's narrowed behavior. The exceptions do not satisfy the checked checklist row.

The 59 literal refuse sites over 16 registry names are a syntactic census, not 59 clause-level behavioral proofs. This review verifies that stated syntax denominator and the 22 supplied mutation/control vectors; it does not claim all 59 clauses are behavior-covered. The new remote-name miss (F6) also demonstrates that a registry derived from implementation cannot establish completeness against the normative output domain.

Additionally, producer C-neutral-oid-swap passes empty test lists with full=false. run_tests returns None without running Go, and the harness labels that SURVIVED-NEUTRAL-OK. The evidence claims exit 0, which was not measured by that row. The reviewer reran this neutral control with the full behavioral suite and observed a real exit 0, but the shipped evidence/harness needs correction. Add the missing narrowing tests, retain separate applied/compiling/behavior/census/neutral classifications, and update checklist/README/logbook evidence accurately.

## Validation and mutation evidence

All commands, untruncated stdout/stderr, and exit codes are in `TASK-260830-2bnr39_review-evidence-rev1.tar.gz`.

| Check | Reviewer execution | Result |
| --- | --- | --- |
| Candidate leaf suite | go test ./internal/gitsnap -v -cover -count=1 | exit 0; 61 top-level tests; 83.5% statement coverage |
| Vet | go vet ./internal/gitsnap | exit 0 |
| Build | go build ./... | exit 0 |
| Traceability | go run ./internal/traceability/cmd/tracecheck | exit 0; existing global ownership result preserved |
| Whole repository tests + coverage | go list ./... gives 24 packages; go test on packages 1–15 and 16–24 with -v -cover -count=1 -timeout=4m | both exit 0; all 24 packages included |
| Formatting | gofmt -l internal/gitsnap in exact candidate | exit 0, no paths |
| Final adversarial probes | go test ./internal/gitsnap -run '^TestReview' -v -count=1 in disposable probe copy | exit 1; concrete defects described above; positive binary/rename and malformed-config controls pass |
| Mutation instrument controls | compiling comment-neutral full suite, then known-bad inverted sort | neutral exit 0; known-bad exit 1 from TestAcceptBaselineSnapshot |
| Producer mutation vectors, independently replayed | compile via go test -run '^$' -count=1, then named behavioral tests/full suites and positive controls; all -count=1 | 20 behavioral kills including known-bad; 2 actual neutral full-suite passes; 0 census-only kills, compile failures, or non-applied mutants |

The review harness executes full behavioral suites for M10/M11 token-preserving behavior swaps and both neutral controls, not only the source-text checker. M12's empty-remotes mutant is a behavioral failure (it can panic after the guard is removed), not evidence for a class-narrowing proof. Each mutant is restored from exact candidate bytes in the disposable copy. See mutation-results.json and mutation-logs/ for per-test failures and compile/control exit codes.

Whole-repository tests and final probes set GIT_CONFIG_GLOBAL=/dev/null and GIT_CONFIG_SYSTEM=/dev/null. The initial package suite used the current environment; both executions pass. This separation matters: the original live fixture helper prepends overrides before os.Environ, while ExecGitRunner inherits host config. The first review probe saw an ambient lfs filter; that observation was not used as evidence of absence. Final findings use isolated configuration. The fixture filesystem stat seam is real; scripted failure tests do not claim real device I/O fault injection.

## Bounds and disposition

- This is ordinary implementation/evidence rework, not an external blocker or human-only decision. No blocked status or approval request is warranted.
- No implementation of upstream #176/#177, no changes to shared reviewer rules, no commits, no branch operations, and no integration or landing were performed.
- Object packs, raw-index blob delivery, untracked/ignored content, symlink targets, recursive submodules, full working-tree digests/manifests, CLI envelopes and orchestration/quiescence are sibling/integration bounds. They are not being demanded in this leaf. F2 concerns fields already captured here.
- Binary content/rename, leading-dash filename, detached/unborn, conflict stages, normal executable/symlink kinds, and repeat capture were driven. Native Windows/Linux, real fsmonitor-valid flags, live SHA-256 repositories, every possible Git extension and all 59 refusal clauses were not independently exhausted. Scripted compatibility evidence is not represented as a live platform result.
- A task-scoped logbook handoff is attached as TASK-260830-2bnr39_review-logbook-rev1.md. The reviewer preserves the accepted candidate identity by leaving LOGBOOK.md untouched; the producer should incorporate the correction while doing rework.
- Required lifecycle: attach this verdict/evidence, route to-dev, then producer rework and another reviewer cycle. Do not accept or checkpoint revision 1.
