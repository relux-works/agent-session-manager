# TASK-260830-2bnr39 — CR rev2 developer outcome

Disposition: ready for review; managed developer handoff publishes the candidate. Work remains uncommitted. The orchestrator owns review, signed checkpoint and integration.

## Preserved candidate and scope

Worktree: `.temp/STORY-260830-35dbcs/worktree`; HEAD/checkpoint `7aa151a9c31071bfab190fd8ae8259f502f2ebbc`; preserved prior tree `2a7575a9b8b5983308fe8769c88a4006858580cf`. No commit, switch, rebase, reset or merge was performed. At inspection `HEAD..main` was 2 commits; these measurements concern this managed candidate, not current main. The manager's new CR records its authoritative candidate tree. `candidate-source-sha256.json` pins the exact delivered source/docs independently.

Normative source: embedded v0.5.0, commit `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`, §§10.2–10.4 and 12.1–12.3. This remains an internal library leaf; no raw-index/object blob delivery, untracked/ignored content, symlink targets, recursive submodules, closure manifests, CLI orchestration or upstream #176/#177 work was added.

## Seven findings addressed

1. Runner-owned environment values override inherited duplicates. Real Git showed an additional porcelain refresh path independent of optional locks, so diff uses temporary index copies with preserved mtime and explicit content-comparison behavior. Real-byte preservation covers stat invalidation, alternate index, success and refusal; cleanup is tested. External Git/filter configuration remains trusted, not sandboxed.
2. Consistency compares HEAD mode/ref/OID and actual index path/file identity/digest/version, plus original logical/delta sentinels. Real same-OID branch/mode changes, v2→v4 rewrites and identical-byte index replacements refuse; stable retry succeeds. This is bounded observation, not atomic capture or a full content digest.
3. Reads/stat paths use repository root; requested relative cwd survives. Root/sub captures agree apart from cwd; linked worktrees are driven.
4. Symbolic HEAD, OID, upstream, every feature config reader and required-filter census have individual fatal/transport/partial-read attacks through Capture. Upstream absence comes from a successful field read. Closing HEAD failures, post-census filter failures, absent-index config failures and successful empty HEAD output refuse.
5. Symlink/filemode defaults match Git config behavior; --bool parses required-filter spellings, dotted names and last-value precedence. TestMain and fixture subprocesses isolate ambient Git config/identity/routing without printing environment values.
6. Remote names enforce 1..128 characters with ASCII/multibyte 128/129 tests and N23. The full string-domain audit additionally found and closed required-filter count 64/65 with N24 and GateFeatures.
7. Narrowed NotRepository/HeadRef/UpstreamRef vectors now fail named production-entry tests. Neutral controls execute full suites. Registration census and behavioral evidence have separate denominators; all supplied controls and deletion/supporting vectors remain available.

## Executed validation

All commands ran as standalone subprocesses with stdout/stderr retained, never piped through tee. No earlier producer/reviewer passing result substitutes for the reruns below. The host is macOS arm64, Go 1.25.5, Apple Git 2.50.1 (exact version logs included).

| Command | Actual exit | Evidence |
| --- | ---: | --- |
| go test ./internal/gitsnap -v -cover -count=1 | 0 | package-final-01.log; 97 top-level tests, 86.5% statements |
| go test ./... -v -count=1 | 0 | all-tests-final-03.log; all 24 packages |
| go test ./... -cover -count=1 | 0 | coverage-final-02.log |
| go test ./... -race -count=1 | 0 | race-final-02.log |
| go build ./... | 0 | build-final-01.log |
| go vet ./... | 0 | vet-final-01.log |
| go run ./internal/traceability/cmd/tracecheck | 0 | traceability-final-01.log; existing ownership unchanged, 17/428 clauses discharged |
| cataloggen with pinned metadata/contracts and -check | 0 | catalog-01.log; exact argv in log |
| repository gofmt empty-output gate | 0 | gofmt-final-01.log + observed-exit-codes.json |
| git diff --check | 0 | diff-check-final-02.log |
| binary/rename regression, -count=10 | 0 | binary-repeat-01.log |
| python3 .scripts/gitsnap-mutations.py --out .temp/TASK-260830-2bnr39/mutations-final-04 | 0 | mutations-final-04.log and per-vector logs/results |

The manager additionally runs its configured validation at handoff and attaches that log to the new CR. It is not preclaimed here. Native Linux/Windows runtime and explicit fuzz mutation-budget coverage were not rerun independently in this developer packet; configured cross-build/fuzz handoff gates retain their own real statuses. No unsupported capability is advertised.

## Mutation measurement and prior failures

Final battery: **35 of 35 vectors applied and compiled; 33 behavioral kills (including known-bad control), 2 measured neutral survivors; 0 unexplained survivors, compile failures, census-only kills or unexecuted selections.** Every kill has named failing tests below. The two survivors have explicit bounds. **17 of 17 registry gates have a narrowing witness** (table below). This does not establish all interior clauses: the final source census has **77 literal sites over 17 registered gates**. Deletion-only rows are supporting evidence and are not used for the gate-narrowing ratio.

Earlier runs are preserved honestly: imported pre-fix review probes exit 1; optional-lock-only fix still failed index bytes (exit 1); disabling autoRefreshIndex failed stat-only delta correctness (exit 1). First mutation battery exit 1 includes a real neutral failure from private-index timestamp drift, subsequently fixed. Its N26 mode fixture also let another sentinel catch the checkout's index change; the corrected mode-only update-ref fixture has a targeted green rerun and is included in the final full battery. The intermediate second battery's actual status is retained separately. No early run is relabeled as passing.

## Narrowing coverage by registry gate

| Registry gate | Narrowing vector |
| --- | --- |
| GateNotRepository | N20 |
| GateHeadCorrupt | M6 |
| GateHeadRef | N21 |
| GateHeadOIDFormat | M8b |
| GateRemotesRange | M3 |
| GateRemoteURL | N23 |
| GateIdentityLength | M5 |
| GateIndexVersion | M2 |
| GateIndexStage | M1 |
| GateIndexEntry | M18 |
| GateIndexSort | M9 |
| GateIndexEntriesRange | M4 |
| GateUpstreamRef | N22 |
| GateDeltaStatus | M7 |
| GateWorktreeKind | M19 |
| GateConsistency | N25 |
| GateFeatures | N24 |

## Per-mutant evidence

| Mutant | Narrowed rejection / behavioral change | Named failing test(s) | Outcome / survivor bound |
| --- | --- | --- | --- |
| C-neutral-comment | Comment-only neutral instrument control | None — survivor | Neutral survivor, actual behavior exit 0; Comment-only change cannot affect runtime behavior; full suite must execute. |
| M1-stage-admits-4 | Allows stage 4 (stage limit remains) | TestRefuseIndexStageFour | Behavior kill; compile 0, behavior 1; positive control 0 |
| M2-version-admits-5 | Allows index version 5 (closed list otherwise retained) | TestRefuseIndexVersionFive | Behavior kill; compile 0, behavior 1; positive control 0 |
| M3-remotes-admits-17 | Allows the 17th remote | TestEdgeRemotesSeventeenRefuses | Behavior kill; compile 0, behavior 1; positive control 0 |
| M4-entries-admits-65537 | Allows index entry 65537 | TestEdgeIndexMaxPlusOneRefuses | Behavior kill; compile 0, behavior 1; positive control 0 |
| M5-identity-admits-257 | Allows identity character 257 | TestEdgeIdentity257CharsRefuses | Behavior kill; compile 0, behavior 1; positive control 0 |
| M6-unborn-admits-tags | Narrows unborn namespace refusal to refs outside refs/ (includes tags) | TestRefuseUnbornNonBranchRef | Behavior kill; compile 0, behavior 1 |
| M7-delta-admits-X | Allows unknown raw-diff status X | TestRefuseUnknownDeltaStatus | Behavior kill; compile 0, behavior 1 |
| M8b-branch-admits-40hex-in-sha256-repo | Allows a 40-hex HEAD in a sha256 repository | TestRefuseCrossFormatOID | Behavior kill; compile 0, behavior 1; positive control 0 |
| C-neutral-oid-swap-survives | Equivalent OID scalar-parser substitution; full behavioral suite | None — survivor | Neutral survivor, actual behavior exit 0; Both scalar parsers enforce prefix/length; the supplied objectFormat is also the constructed prefix. This substitution is behavior-neutral at this call site; it does not cover cross-format acceptance elsewhere. |
| M9-sort-admits-duplicate | Allows duplicate path/stage pairs | TestRefuseDuplicateIndexEntry | Behavior kill; compile 0, behavior 1; positive control 0 |
| M10-fetch-push-swapped | Swaps fetch/push behavior while keeping searched tokens; full behavioral suite | TestRemoteFetchPushMapping | Behavior kill; compile 0, behavior 1; positive control 0 |
| M11-assume-bit-swapped | Swaps assume-unchanged flag behavior while retaining flag token; full behavioral suite | TestCaptureLiveIndexFlags, TestCaptureLiveBranchRepository | Behavior kill; compile 0, behavior 1 |
| M17-push-credential-admission | Deletes credential-bearing push URL refusal; supporting deletion evidence only | TestRefuseCredentialPushURL | Behavior kill; compile 0, behavior 1; positive control 0 |
| M18-index-admits-8hex-oid | Allows an 8-hex index OID | TestRefuseIndexBadOID | Behavior kill; compile 0, behavior 1; positive control 0 |
| M19-worktree-admits-fifo | Allows FIFO worktree kind | TestRefuseSpecialWorktreeFile | Behavior kill; compile 0, behavior 1; positive control 0 |
| M12-remotes-min-disabled | Deletes empty-remotes minimum; supporting deletion evidence only | TestRefuseNoRemotes | Behavior kill; compile 0, behavior 1 |
| M13-branch-ref-clause-disabled | Deletes branch-ref refusal; supporting deletion evidence only | TestRefuseBranchWithLiteralHeadRef | Behavior kill; compile 0, behavior 1 |
| M14-consistency-index-clause-disabled | Deletes logical-index consistency clause; supporting deletion evidence only | TestConsistencyRefusesIndexMutationArmedOnce | Behavior kill; compile 0, behavior 1; positive control 0 |
| M15-corrupt-head-clause-disabled | Deletes corrupt-HEAD clause; supporting deletion evidence only | TestRefuseCorruptHead | Behavior kill; compile 0, behavior 1 |
| M16-upstream-clause-disabled | Deletes upstream-ref refusal; supporting deletion evidence only | TestRefuseBadUpstream | Behavior kill; compile 0, behavior 1 |
| C-bad-inverted-sort | Known-bad instrument control: rejects a valid sorted index | TestAcceptBaselineSnapshot | Behavior kill; compile 0, behavior 1 |
| N20-not-repository-admits-fatal-128 | Allows quiet fatal exit 128 as legitimate absent result | TestReviewFeatureReadFailure | Behavior kill; compile 0, behavior 1 |
| N21-head-ref-admits-HEAD | Allows literal HEAD at branch-ref read boundary | TestCaptureRefGrammarAtEachRead | Behavior kill; compile 0, behavior 1 |
| N22-upstream-admits-HEAD | Allows literal HEAD at upstream-ref read boundary | TestCaptureRefGrammarAtEachRead | Behavior kill; compile 0, behavior 1 |
| N23-remote-name-admits-129 | Allows remote name character 129 | TestCaptureRemoteNameBounds | Behavior kill; compile 0, behavior 1 |
| N24-filters-admits-65 | Allows required filter 65 | TestCaptureRequiredFilterCountBounds | Behavior kill; compile 0, behavior 1 |
| N25-consistency-admits-same-length-index-change | Allows same-length logical index mutations | TestConsistencyRefusesIndexMutationArmedOnce | Behavior kill; compile 0, behavior 1 |
| N26-consistency-admits-same-oid-branch-switch | Allows same-OID HEAD ref/mode changes | TestReviewHeadRefRace, TestCaptureSameOIDHeadModeRace | Behavior kill; compile 0, behavior 1 |
| N27-consistency-admits-index-version-only-change | Allows actual index changes when version differs | TestReviewIndexVersionRace | Behavior kill; compile 0, behavior 1 |
| N28-lock-env-inherited-one-wins | Inherited optional-lock setting can override runner setting | TestReviewRunnerLockOverride | Behavior kill; compile 0, behavior 1 |
| N29-diff-bypasses-private-index | Unstaged diff bypasses private index (cached diff remains isolated) | TestReviewLockOverrideMutatesIndex | Behavior kill; compile 0, behavior 1 |
| N30-feature-default-symlinks-false | Unset symlinks incorrectly defaults false; behavior regression | TestReviewSymlinkDefault | Behavior kill; compile 0, behavior 1 |
| N31-filter-boolean-raw-yes | Required-filter read bypasses Git boolean parsing; behavior regression | TestReviewRequiredFilterBoolean | Behavior kill; compile 0, behavior 1 |
| N32-subdir-keeps-caller-command-scope | Retains caller command directory and root token; full behavioral suite | TestReviewSubdirectoryCapture, TestCaptureRootAndSubdirectoryAgree, TestCaptureLinkedWorktreeSubdirectory | Behavior kill; compile 0, behavior 1 |

# Git repository/index capture acceptance drivers

TASK-260830-2bnr39; normative v0.5.0 commit
28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c, §§10.2–10.4 and 12.1–12.3.

**8 of 8 AC rows driven** through the internal production `Capture` entry.
This measures entry drivers, not complete coverage of every interior clause.
Tests are included in the managed Change Request; the orchestrator owns their
signed checkpoint and integration. No CLI orchestration is claimed.

| AC row | Production call site | Named drivers |
| --- | --- | --- |
| Repository identity | Capture → readRemotes → deriveIdentity | TestCaptureLiveBranchRepository; TestRemoteFetchPushMapping; TestCaptureRemoteNameBounds |
| HEAD/ref | Capture → readHead → recheckConsistency | TestCaptureLiveHeadModes; TestReviewHeadRefRace; TestCaptureSameOIDHeadModeRace; TestCaptureRefGrammarAtEachRead; TestCaptureSuccessfulEmptyHeadReadRefuses |
| Worktree metadata | Capture → readRepository | TestCaptureRootAndSubdirectoryAgree; TestCaptureLinkedWorktreeSubdirectory |
| Index stages | Capture → parseIndexEntries | TestCaptureLiveConflictStages; TestCompatConflictStages; TestRefuseIndexStageFour |
| Index flags | Capture → parseIndexRecord/debugFlags | TestCaptureLiveIndexFlags; TestCaptureLiveBranchRepository; TestCompatFSMonitorBit |
| Staged deltas | Capture → parseDeltas (cached stream) | TestCaptureLiveBranchRepository; TestReviewBinaryRenameControl |
| Unstaged deltas | Capture → parseDeltas (worktree stream) | TestCaptureLiveBranchRepository; TestCaptureLiveMissingTrackedPath; TestReviewLockOverrideMutatesIndex |
| File modes | Capture → statIndexPaths/statDeltaPaths | TestCaptureLiveExecBitAndSymlinkKind; TestRefuseSpecialWorktreeFile; TestCaptureRootAndSubdirectoryAgree |

Additional negative/recovery drivers cover failed reads (per reader and per
failure shape), actual-index version/replacement races, exact bounds,
repeat capture, and temporary-index cleanup. The task outcome contains the
executed narrowing-mutant table and command exit codes. The source census is
only a count of literal refusal sites; its denominator is not behavioral
proof. Native Windows/Linux runtime, live fsmonitor-valid flags, arbitrary Git
extensions and full workspace content closure remain stated bounds.


# Capture output string-domain audit

Source of expected wire fields: pinned `internal/specdoc/SPEC.md` v0.5.0,
commit `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`, §10.4 Git member and
embedded types, and §12.1 repository-relative cwd. This audit starts with
those declarations and the complete exported `Snapshot` type graph, not the
implementation's refusal census. `Snapshot` is not a serialized closed Git
WorkspaceSnapshotMember; sibling-owned fields are intentionally absent.

| Produced field(s) | Domain / read-boundary enforcement | Driver / stated bound |
| --- | --- | --- |
| RepositoryIdentity | UTF-8, 1..256 characters, derived sanitized remote path | TestEdgeIdentity256CharsAccepts, TestEdgeIdentity257CharsRefuses |
| Remotes[].Name | UTF-8, 1..128 characters; outputs sorted and unique by name | TestCaptureRemoteNameBounds (ASCII/multibyte), TestRemoteFetchPushMapping |
| Remotes[].FetchURL, PushURL | scalar.ParseSanitizedGitURL; nullable push URL | TestRefuseCredentialBearingRemote, TestRefuseCredentialPushURL |
| Head.Mode | branch/detached/unborn generated from validated reads | TestCaptureLiveHeadModes |
| Head.OID | object-format-matching scalar GitOID or nil | TestRefuseCrossFormatOID, TestCompatSHA256ObjectFormat |
| Head.Ref | scalar GitRef or nil; unborn requires refs/heads/ | TestCaptureRefGrammarAtEachRead, TestRefuseUnbornNonBranchRef |
| UpstreamRef | scalar GitRef or nil from a successful upstream field read | TestCaptureRefGrammarAtEachRead, TestCaptureReadFailuresNeverBecomeAbsence, TestCaptureUpstreamConfiguredMissingTarget |
| ObjectFormat, Features.ObjectFormat | sha1/sha256 from repository probe | TestRefuseCrossFormatOID, TestCompatSHA256ObjectFormat |
| Index.Format | generated git_index constant | TestCompatIndexVersions |
| Index.Entries[].Path | scalar RelativePath | TestRefuseIndexEscapingPath; exact parent/child fixtures |
| Index.Entries[].OID | scalar GitOID matching repository format | TestRefuseIndexBadOID, TestCompatSHA256ObjectFormat |
| Worktree.CWD | dot or scalar RelativePath, based on requested capture directory | TestCaptureRootAndSubdirectoryAgree, TestCaptureLinkedWorktreeSubdirectory |
| Features.RequiredFilters[] | nonempty UTF-8 Git subsection names; sorted unique array, 0..64 members | TestCaptureRequiredFilterCountBounds, TestCaptureRequiredFilterBooleanSpellings; no invented per-name maximum |
| Worktree.RepoRoot, GitDir, CommonDir, Worktrees[] | Supplementary native Git/OS path metadata, not normative wire paths | Root/sub/linked tests; no general wire-scalar or arbitrary newline-path fidelity claim |
| Staged[].Path, Target; Unstaged[].Path, Target | Supplementary raw-diff paths, scalar RelativePath; target nullable | TestReviewBinaryRenameControl; relative-path scalar validation (no separate invalid-delta-path driver claimed) |
| Staged[].Status, Unstaged[].Status | A/C/D/M/R/T/U; supplementary delta vocabulary | TestRefuseUnknownDeltaStatus |
| Staged[].OldOID, NewOID; Unstaged[].OldOID, NewOID | Supplementary raw Git hex OIDs, zero sides become nil | No independent GitOID wire validation at these fields; they are not serialized as normative git-oid members. Content/object closure is sibling-owned. |
| PathModes[].Path | Derived from already-validated index/delta paths | TestCaptureRootAndSubdirectoryAgree, TestCaptureLiveMissingTrackedPath |
| PathModes[].Kind | generated filesystem-kind vocabulary; special files refused | TestRefuseSpecialWorktreeFile, TestCaptureLiveExecBitAndSymlinkKind |

All exported string and pointer/slice-of-string fields in the Snapshot type
graph are accounted for above. This is an explicit audit, not an automated
proof of complete schema conformance. The normative member's IDs, object
pack/blob descriptors, manifests, submodules, sparse-pattern blobs, project
config paths and materialization policy are absent and remain sibling-owned.

The registry census measures literal sites and registration only. Mutation
results measure selected narrowing vectors per registry gate; neither proves
every interior refusal clause or every possible Git extension/platform.

## Operational notes and bounds

No delegates were spawned. Required project-management and Curator-managed Go testing skills were read. This managed worktree lacks installed .claude/.agents adapters; the Go skill was read from the main checkout's Curator-managed adapter as assigned. The failed local skill-path read and corrected path are recorded in tooling notes; no tool install or global shared-contract edit was performed.

Full logs and the final harness/vector source are included. Real Git tests do not contact remotes. Scripted fsmonitor-valid/SHA256 evidence does not claim live platform coverage. Split-index compatibility here is logical entries and real-index byte preservation; auxiliary shared-index timestamps and every Git extension are not exhaustively attested. Supplementary raw-diff OIDs/native paths have the explicit bounds in the schema audit. Quiescence, change-and-revert races and all included-file digests remain integration/sibling responsibilities.
