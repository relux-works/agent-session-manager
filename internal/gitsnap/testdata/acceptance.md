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
