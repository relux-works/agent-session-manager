# TASK-260830-3m7m7w — review verdict, revision 1

Verdict: **changes_requested**. Route: **to-dev**. Finding **F1**, priority **P1**.
Finding class: **Absent evidence treated as satisfied / failure to read reported as absence**.
repeat-of: **none**. This is the first review of this content candidate; it does not reopen an accepted predecessor finding.

## Candidate and authority

Reviewer: RUN-260907-68f937. Producer read from live board: RUN-260907-d14c2f (developer/implementer).
CR-TASK-260830-3m7m7w-1, revision 1, ready at review; repository_delta=present.
Base/checkpoint: d888cd576f5962eb258e40ca487f7711b2215610.
Exact candidate tree: f219517ac93cf446bd6ae70c7ebd3a86eb2c85fa.
Patch: TASK-260830-3m7m7w_change-request_rev1.patch, SHA-256 d5a2a67af14022da5cefd206c523fa547b30588e2174e044964aa7950e16b888.
Producer outcomes inspected: TASK-260830-3m7m7w_capture-evidence.md and TASK-260830-3m7m7w_capture-evidence.tar.gz. Archive hash verified: 0cdd79565b2b487a3d63b11d512811d56d28d5967fd3ff63e82e3e170069b239. All 17/17 source-manifest files match this candidate.
Pinned AX specification: v0.5.0, commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c, §§10.2–10.4 and 12.1–12.3; embedded SPEC.md hash verified as 562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a.

Live goal: GOAL-260907-0ef231 revision 1, kind/success_predicate reviewer_verdict/reviewer_verdict. Scope: TASK-260830-2bnr39 and TASK-260830-3m7m7w. Exact objective is attached in goal.json. It requires an evidence-backed verdict for each assigned item. No parent goal writes or integration actions were performed.
Predecessor CR-TASK-260830-2bnr39-2 is checkpointed, independently accepted by RUN-260907-428f33, then checkpointed by RUN-260907-d6fa16 at the base above, exact tree 935f624e45088a064de04945cbe4bd86d1b37d1b. Its task remains integrating. The persisted accepted CR and attached predecessor verdict/checkpoint prove its review branch; this does not assert Story delivery or done.

## F1: failed ignore-policy reads become permission to capture excluded bytes

Production path: `Capture` → `capture` → `contentState.capture`, internal/gitsnap/content.go:222–226, calls `runOK` for `git ls-files --others --ignored --exclude-standard --directory -z`. `runOK`, internal/gitsnap/snapshot.go:284–292, accepts exit 0 and discards stderr. An empty stdout is then a legitimate empty census to `nulPaths`; the walker includes files that the unreadable policy was supposed to exclude.

Real-Git witness: `TestReviewerUnreadableExcludePolicy` in the attached reviewer_probe_test.go (disposable exact-candidate copy; product sources untouched):

1. Create a real repository with an untracked `ignored-private` file.
2. Put `ignored-private` in either `.git/info/exclude` or an external file selected by repository `core.excludesFile`.
3. Confirm ordinary `Capture` excludes the file while the policy is readable.
4. Set the policy file to mode 000, then invoke real ExecGitRunner and real Capture.
5. Git returns exit 0, empty stdout, and `warning: unable to access ...: Permission denied` on stderr. Capture returns success and an entry for `ignored-private`.

Measured **3 of 3 reproductions for each of two policy sources (6/6)** on macOS arm64, Go 1.25.5, Apple Git 2.50.1. All six adversarial assertions fail as intended against this candidate; ignore-failure-repeat.log records the actual Git outputs and successful inclusion. This is a stationary unreadable-policy failure, not a quiescence/final-closure race delegated to TASK-260830-2xt6fd.

Additional injected production witness `TestReviewerPartialIgnoredSuccess` also fails: successful exit plus census warning and absent output bypass the leaf's GateContentRead. Existing `TestContentReadFailureIsNotAbsence` covers exit 128 and unterminated output only. The narrowing vector N-read-admits-quiet-fatal128 is killed, yet this distinct class member is admitted by the unchanged implementation.

Required rework: preserve and assess the completeness of the ignore census at this content boundary; a failed policy read must yield a refusal and nil Snapshot, not empty-policy semantics. Add committed-candidate real-Git tests for both policy sources, a missing-versus-unreadable control, restoration/retry, and a narrowing mutant that admits this successful-exit warning class. Preserve benign/legitimate absence behavior and the predecessor's private-index ownership. This is routine implementation rework; no human-only platform decision is needed.

## Coverage and validation

**7 of 7 scoped AC rows driven** by named candidate-resident tests at the internal Capture entry. This is driver presence/execution coverage, not proof that all classes work: F1 contradicts ignored-policy correctness and the claimed missing-versus-unreadable handling.

| AC row | Production call site | Named candidate drivers independently rerun |
| --- | --- | --- |
| Untracked bytes | Capture → contentState.capture → Guard.Open → blob → ObjectStore.PutBlob | TestContentCaptureSelectionAndBytes; TestContentNormativeWorkspaceBytes |
| Ignored policy | Capture → contentState.validate/capture | TestContentIgnoredDefaultAndPolicyRefusal; TestContentNestedDetachedSubmodule |
| Symlink targets/containment | Capture → contentState.capture → safeSymlink → CheckManifestEntries | TestContentSymlinks; TestContentSymlinkChainsAndExcludedTargets |
| Submodules | Capture → contentState.submodules → recursive capture | TestContentSubmodulePointerStates; TestContentSubmoduleUninitializedAndRefusals; TestContentNestedDetachedSubmodule; TestContentSubmoduleCountBounds; TestContentRelativeSubmoduleURL |
| Large blobs | Capture → contentState.blob → installBlob | TestContentLargeBlobChunksAndLimit; TestContentBlobInstallFailureAndRetry |
| Sparse/linked worktrees | Capture → readRepository/readFeatures → contentState.capture | TestContentSparseLinkedWorktree; TestContentSparseReadFailure |
| Exclusions | Capture → contentState.validate/excluded and recursive exclusion recheck | TestContentCaptureSelectionAndBytes; TestContentLocalPathOwnerExclusions; TestContentExcludedSubmoduleCannotBypassPolicy |

Independently executed on this exact candidate:
- `go test ./internal/gitsnap ./internal/canonicaljson -count=1 -v`: exit 0.
- `go test ./... -v`: exit 0, 24/24 packages; unchanged package cache use is visible in the log.
- `go test ./... -cover`: exit 0, 24/24 packages; gitsnap 83.4%, canonicaljson 97.1%.
- `go build ./...`, `go vet ./...`, formatting and `git diff --check`: exit 0; formatting output empty.
- Full content-mutations.json replay via .scripts/gitsnap-mutations.py: exit 0, **11/11 narrowing vectors killed**, known-bad control killed, **1/1 neutral control passed**; 13 total rows, all compile. The source-token-preserving path mutant runs the full behavioral suite; the entry-array mutant runs the shared behavioral bound plus a positive control. The 6/6 registered-gate witness ratio is selected-member coverage and cannot establish absence of F1.
- Reviewer probes: tracked bytes inside an ignored directory are retained; included unreadable files refuse; an invalid initialized .git marker refuses. An unreadable excluded directory remains excluded (permitted; this is not an unreadable policy source). A symlink through a non-directory refuses. Injected warning census and both actual unreadable policy sources expose F1.

Accepted existing evidence, explicitly not rerun: producer race checks, Windows build/vet (compile-only), catalog and tracecheck. Relevant archive logs and actual recorded exits inspected against the 17/17 matching source manifest. Global traceability remains 17/428 discharged clauses. These checks do not erase F1.

The initial reviewer probe command wrote to an incorrect path and ran zero tests; its exit 0 is not validation. The initial git-info fixture omitted directory creation and failed its baseline; it is superseded by the checked-setup 3-repeat log, not counted as a reproduction. A launch with a mistyped cwd never started. Logs retain these failed attempts. An initial file comparator followed CLAUDE.md and reported a mismatch; the corrected symlink-aware comparison verifies all 2933/2933 candidate files, including the relative AGENTS.md link, with zero mismatches. Index SHA-256 remains bfb3eeb21c83daee474a42adb7b143c77f6ce0f8455fe3292ed6cc8d580e88ba; HEAD unchanged.

## Scope and residual bounds

The optional internal Capture API is the real entry for this leaf; no CLI or doctor capture capability is advertised. Shared canonicaljson, secprim and localstore reuse fits the architecture. The scoped positive/recovery evidence, including real crash exit 73 and stable retry, remains useful but insufficient for acceptance due to F1.
Object packs, raw-index delivery, root/child manifest publication, final closure/race integration, provider quiescence and end-to-end materialization remain TASK-260830-2xt6fd/Story work. Do not treat this verdict as approval of those surfaces or of an uncalled final manifest helper. The predecessor's sharedindex auxiliary timestamp and hard-kill disposable-file bounds remain accepted. Native Linux/Windows execution, full successful 128 GiB transfer, all recursion/grammar combinations and adversarial filesystem substitution schedules are not newly proven here.

No product edits, manual commits, index changes, branch operations, accept_cr, or commit_ack. Detailed task-scoped evidence and a logbook handoff are attached before lifecycle end. Delivery is not accepted; owner routes F1 to a producer and another independent reviewer cycle.
