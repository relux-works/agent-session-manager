# TASK-260830-3m7m7w revision-2 mutation evidence

12/12 narrowing vectors killed; known-bad control killed; neutral control passed. Every compile exits 0; every killed behavioral process exits 1. The neutral behavioral process exits 0. Successful-exit warning positive control exits 0. Full per-command exits and executed names are in mutations/results.json.

| Mutant | What the gate is narrowed to | Named failing test(s) | Result / survivor bound |
| --- | --- | --- | --- |
| C-content-neutral-comment | Comment only (neutral control) | None | NEUTRAL_PASS: Comment-only change has no runtime effect; the complete gitsnap suite executes. |
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

Gate coverage is 6 of 6 content registry gates with selected rejected-member witnesses, plus exclusion and consistency call sites and the shared entry-count bound. This is not exhaustive coverage of every predicate or warning source. No killed row lacks a named failing test. The source-token-preserving path mutant executes the full gitsnap behavioral suite. The newly added warning mutant drives real Capture with BOTH actual policy sources and injected empty/partial/unknown diagnostics. Native permission checks can skip on privileged/Windows hosts; they did not skip here.
