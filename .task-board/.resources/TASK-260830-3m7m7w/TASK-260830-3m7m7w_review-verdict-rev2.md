# TASK-260830-3m7m7w — independent review, CR revision 2

Verdict: **accepted**. CR1 finding F1 (P1, failed-read-as-absence) is resolved; no new blocking finding or repeated finding was reproduced. This is acceptance of the content leaf, not Story landing.

## Authority and exact candidate

Reviewer RUN-260908-2e4871. Live board producer RUN-260908-5dde57, developer/implementer. CR-TASK-260830-3m7m7w-2 revision 2, repository_delta=present, 19 changed paths. Board state was ready on entry. Base/checkpoint d888cd576f5962eb258e40ca487f7711b2215610; exact candidate tree 4f1407cb98f19424db527d313a3d89d34e2fd231. Patch TASK-260830-3m7m7w_change-request_rev2.patch, SHA-256 04382425a19699cd65bcfa1139632d0dcef4693ba1372ad6e018250c7a89cec8, verified from the live resource.

Latest run goal GOAL-260908-f0f09a revision 1, reviewer_verdict/reviewer_verdict. Resolved scope TASK-260830-2bnr39 and TASK-260830-3m7m7w. The authoritative objective requires an evidence-backed verdict for each assigned leaf; full returned objective and scope are retained in goal-checkpoint.json. It was refreshed at directive checkpoints and before verdict. No directives, nested agents, parent goal writes, product edits, index changes, commits, or integration actions.

Pinned AX v0.5.0, commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c, §§10.2–10.4 and 12.1–12.3. Embedded SPEC.md SHA-256 verified: 562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a. Reviewed normative blob limits, entry representation, workspace selection, recursive repository boundaries and consistency requirements against the scoped implementation and exact corpus driver.

All 2934/2934 candidate files, modes and symlinks match the immutable tree. Tracked/nonignored path set also matches, without extra/missing paths. Original index SHA-256 remains bfb3eeb21c83daee474a42adb7b143c77f6ce0f8455fe3292ed6cc8d580e88ba. HEAD unchanged. See identity-final.json and reusable identity_audit.py. Additional reviewer probes ran only in a disposable archive of the exact candidate; mutation harness copied the candidate and restored each mutation. No candidate source was edited.

## F1 closure and independent attacks

Production chain is Capture → capture → contentState.capture → Runner.Run. The content census jointly examines process error, exit status and stderr before selecting files; every diagnostic refuses, without a warning-text or filename allowlist. Malformed NUL output still refuses. A nil snapshot is returned on failure. This rule is shared by root and recursive child capture; the existing private-index Runner is preserved.

Personally reran TestContentExcludePolicyReadCompleteness and TestContentIgnoreCensusWarningRefuses with TestContentReadFailureIsNotAbsence three times: exit 0. Both real policy sources (info/exclude, core.excludesFile) execute missing/readable/unreadable/restored controls. **6/6 actual exit-0 permission-warning observations refused**, no permission skips; nil Snapshot and original-index bytes checked. Empty/partial/unknown successful-exit diagnostics also refuse, and stable real-Git retry succeeds.

Replayed the prior reviewer probe source against CR2: the formerly failing real-policy and partial-success cases pass. Additional TestReviewerPolicyOtherFailedReads drives Capture on unreadable root .gitignore, nested .gitignore, inaccessible info directory, directory-valued external excludesFile and directory-valued info/exclude. **5/5 additional shapes refused**, nil snapshots and unchanged real index. Directory-valued policies fail earlier through the production staged-delta read (Git exit 128); they are not attributed to the new census check. Permission cases report Git exit 0 plus stderr and refuse at GateContentRead.

Prior probes also check tracked files inside ignored directories, unreadable included bytes, invalid initialized submodule markers and partial diagnostics. Two inherited probes are observations, not strong assertions: unreadable already-excluded directories remain excluded without failing capture; a symlink traversing a non-directory reports a read refusal. Their logged observations are not counted as additional gate proofs.

N-read-admits-success-diagnostic retains process-error/nonzero/malformed-output refusals but admits successful-exit diagnostics. It compiles and both TestContentIgnoreCensusWarningRefuses and TestContentExcludePolicyReadCompleteness fail (behavior exit 1); the existing fatal/partial-read driver remains green (positive-control exit 0). Thus the regression suite detects weakening the class boundary, not merely deleting a guard.

## AC coverage

**7 of 7 scoped AC rows driven.** Checked the named-driver map before code inspection and matched every named driver to fresh PASS output. These tests are in the immutable candidate; the assigned managed workflow reserves signing/checkpointing for the producer after acceptance. Internal Capture is the production entry for this leaf. CLI capture, doctor capability and full closure are explicitly not advertised.

| AC | Production call site | Named passing drivers |
| --- | --- | --- |
| Untracked bytes, with staged/working bytes distinct | Capture → contentState.capture → Guard.Open → blob → ObjectStore.PutBlob | TestContentCaptureSelectionAndBytes; TestContentNormativeWorkspaceBytes |
| Ignored files only through classified project/provider includes; complete ignore census | Capture → contentState.validate/capture → Runner.Run | TestContentIgnoredDefaultAndPolicyRefusal; TestContentNestedDetachedSubmodule; TestContentIgnoreCensusWarningRefuses; TestContentExcludePolicyReadCompleteness |
| Symlink targets and containment | Capture → contentState.capture → safeSymlink; canonicaljson.CheckManifestEntries | TestContentSymlinks; TestContentSymlinkChainsAndExcludedTargets |
| Submodule state and repository boundaries | Capture → contentState.submodules → capture (recursive) | TestContentSubmodulePointerStates; TestContentSubmoduleUninitializedAndRefusals; TestContentNestedDetachedSubmodule; TestContentSubmoduleCountBounds; TestContentRelativeSubmoduleURL |
| Large blobs and protocol size refusal | Capture → contentState.blob → installBlob | TestContentLargeBlobChunksAndLimit; TestContentBlobInstallFailureAndRetry |
| Sparse/linked worktrees | Capture → readRepository/readFeatures → contentState.capture | TestContentSparseLinkedWorktree; TestContentSparseReadFailure; TestCaptureLinkedWorktreeSubdirectory |
| Mandatory exclusions and explicit owner policy | Capture → contentState.validate/excluded; submodules exclusion recheck | TestContentCaptureSelectionAndBytes; TestContentLocalPathOwnerExclusions; TestContentExcludedSubmoduleCannotBypassPolicy |

## Independently executed mutation evidence

**12/12 narrowing vectors killed; known-bad control killed; neutral control passed.** Compile exits are all 0; behavioral exits are 1 for kills and 0 for neutral. Six of six content registry gates have selected-member witnesses, with additional exclusion/consistency and shared entry-count vectors. This is not every interior predicate. Source-token-preserving symlink weakening executes the complete behavioral suite, not only the source census. The only survivor is the comment-only neutral control with its explicit bound.

| Mutant | Narrowing/control | Actual named failure | Result |
| --- | --- | --- | --- |
| C-content-neutral-comment | Comment only (neutral control) | None; neutral full suite | NEUTRAL_PASS |
| C-content-bad-clear-entries | Erases collected entries (known-bad control) | TestContentCaptureSelectionAndBytes | BEHAVIOR_KILL |
| N-policy-admits-unclassified-allow | Allows one unclassified ignored include | TestContentIgnoredDefaultAndPolicyRefusal | BEHAVIOR_KILL |
| N-read-admits-quiet-fatal128 | Admits quiet exit 128 while retaining other read refusals | TestContentReadFailureIsNotAbsence | BEHAVIOR_KILL |
| N-read-admits-success-diagnostic | Admits successful-exit diagnostics; nonzero, process-error and malformed-output gates retained | TestContentIgnoreCensusWarningRefuses; TestContentExcludePolicyReadCompleteness | BEHAVIOR_KILL |
| N-path-admits-escaping-chain | Allows escape-chain while retaining original source tokens; full behavioral suite | TestContentSymlinkChainsAndExcludedTargets | BEHAVIOR_KILL |
| N-submodule-admits-stray-uninitialized | Allows exactly one stray entry in an uninitialized child | TestContentSubmoduleUninitializedAndRefusals | BEHAVIOR_KILL |
| N-blob-limit-admits-one-byte-over | Allows exactly max blob size + 1 | TestContentLargeBlobChunksAndLimit | BEHAVIOR_KILL |
| N-install-admits-shard-failure | Allows shard creation failures | TestContentBlobInstallFailureAndRetry | BEHAVIOR_KILL |
| N-exclusion-admits-auth-json | Removes auth.json from the credential exclusion set | TestContentCaptureSelectionAndBytes | BEHAVIOR_KILL |
| N-ignored-admits-deny | Allows ignored/deny without an include rule | TestContentIgnoredDefaultAndPolicyRefusal | BEHAVIOR_KILL |
| N-consistency-admits-five-byte-digest-change | Allows changed digest when size is 5 | TestContentDigestRaceAndRetry | BEHAVIOR_KILL |
| N-submodule-exclusion-bypass | Allows excluded vendor/lib through recursion | TestContentExcludedSubmoduleCannotBypassPolicy | BEHAVIOR_KILL |
| N-entry-set-admits-65537 | Allows exactly 65,537 entries at shared owner | TestCheckManifestEntriesCountBounds | BEHAVIOR_KILL |

## Verification and evidence boundaries

Personally reran `go test ./... -count=1 -v` and `go test ./... -cover -count=1`: each exit 0, **24/24 packages**. Coverage gitsnap 83.4%, canonicaljson 97.1%. Five pre-existing skipped rows are listed in acceptance-audit.json; none is a content-policy permission witness. Build, vet, formatting, catalog check, tracecheck and diff check exit 0. Tracecheck reports 17/428 discharged clauses, not full specification coverage. Full suite executes the real child exit-73 crash/retry and digest-race/retry tests. Every launched verification process completed before verdict attachment.

Read prior verdict/probes before producer rework evidence. Producer archive SHA-256 ec5b8eaa153ad820d688d4ba5fcfa05503966c195da51b689380dca0188b396b and prior reviewer archive ae707103b861f35dd043ecc2c1b681ff2e79695b009654eeaa82a40b8968f573 verified. Producer source manifest matches **19/19** candidate files. Accept existing producer race evidence (24/24 packages), cross-compilation and Windows vet, fuzz smoke and remaining publication-command logs; those commands were not rerun by this reviewer. Native evidence here is macOS arm64, Go 1.25.5, Apple Git 2.50.1.

The managed publication transcript is explicitly truncated (5,280,886 bytes omitted), with only five visible exit markers; it does not independently establish all 26 checks. Producer's separate command audit/logs supply that evidence. A review audit initially asserted 26 visible markers, failed, then inspected the truncation and corrected the evidence attribution. Board validate reports 1729 issues despite exit 0; no clean-board claim is made, and unrelated control-plane repairs are outside this content review.

README, logbook and content-acceptance.md keep the following bounds explicit: no complete manifest assembly, object-pack production, raw-index blob delivery, quiescence or destination materialization; final closure/race integration remains TASK-260830-2xt6fd. Included-byte rechecks are not atomic filesystem snapshots and cannot detect change-and-revert. Sparse-pattern final races, all hostile filesystem schedules, all grammar/depth combinations and native Linux/Windows behavior remain unproven. Successful streaming is measured at 8 MiB + 4 bytes; a real sparse 128 GiB + 1 file refuses, while full 128 GiB success is unmeasured. Conventional filename exclusions are not confidentiality attestation; arbitrary runtime paths require trusted owner policy. Conservative diagnostics can refuse benign warnings. Missing optional policy is distinct from failed reads on the tested platform.

## Scoped predecessor and routing

TASK-260830-2bnr39 remains integrating with CR2 checkpointed, 18/18 checklist items, reviewer RUN-260907-428f33. Existing accepted verdict and RUN-260907-d6fa16 checkpoint resources were read. Its signed checkpoint d888cd576f5962eb258e40ca487f7711b2215610 is this candidate's base; signature independently reverified. Accepted tree 935f624e45088a064de04945cbe4bd86d1b37d1b. No new contradictory evidence reopens it. Preserve its sharedindex auxiliary timestamp and hard-kill disposable-file bounds. Its acceptance is inherited scoped evidence, not a fresh review of unchanged work.

Record this leaf using `accept_cr(TASK-260830-3m7m7w, revision=2, evidence=TASK-260830-3m7m7w_review-verdict-rev2.md)`. The atomic accepted/integrating receipt is the reviewer verdict evidence. A new tracked developer/implementer owns checkpoint/integration; this reviewer does not set done or supply commit_ack. Both scoped leaves are accepted but not landed; parent delivery remains separate.
