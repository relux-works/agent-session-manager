# Working-copy content capture acceptance

Task TASK-260830-3m7m7w; parent STORY-260830-35dbcs. Normative source is
agent-session-manager-spec v0.5.0, commit
28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c, §§10.2–10.4 and 12.1–12.3.
The embedded document hash is checked by TestContentNormativeWorkspaceBytes.

**7 of 7 scoped AC rows driven.** These are the seven content classes in this
leaf's assignment, not seven discharged specification sections or complete
workspace closure. Tests reside in the managed candidate; the owner commits
only after independent review. Production entry is the internal
`Capture(ctx, runner, directory, ContentOptions)` API. No CLI capture or doctor
capability is advertised; omitting options returns only predecessor metadata.

| AC row | Production call site | Named drivers |
| --- | --- | --- |
| Untracked bytes, with staged/working bytes distinct | Capture → contentState.capture → Guard.Open → blob → ObjectStore.PutBlob | TestContentCaptureSelectionAndBytes; TestContentNormativeWorkspaceBytes |
| Ignored files only through classified project/provider includes; complete ignore census | Capture → contentState.validate/capture → Runner.Run | TestContentIgnoredDefaultAndPolicyRefusal; TestContentNestedDetachedSubmodule; TestContentIgnoreCensusWarningRefuses; TestContentExcludePolicyReadCompleteness |
| Symlink targets and containment | Capture → contentState.capture → safeSymlink; canonicaljson.CheckManifestEntries | TestContentSymlinks; TestContentSymlinkChainsAndExcludedTargets |
| Submodule state and repository boundaries | Capture → contentState.submodules → capture (recursive) | TestContentSubmodulePointerStates (clean/staged/unstaged/combined); TestContentSubmoduleUninitializedAndRefusals; TestContentNestedDetachedSubmodule; TestContentSubmoduleCountBounds; TestContentRelativeSubmoduleURL |
| Large blobs and protocol size refusal | Capture → contentState.blob → installBlob | TestContentLargeBlobChunksAndLimit; TestContentBlobInstallFailureAndRetry |
| Sparse/linked worktrees | Capture → readRepository/readFeatures → contentState.capture | TestContentSparseLinkedWorktree; TestContentSparseReadFailure; predecessor TestCaptureLinkedWorktreeSubdirectory |
| Mandatory exclusions and explicit owner policy | Capture → contentState.validate/excluded; submodules exclusion recheck | TestContentCaptureSelectionAndBytes; TestContentLocalPathOwnerExclusions; TestContentExcludedSubmoduleCannotBypassPolicy |

Additional production refusal/recovery drivers:
TestContentReadFailureIsNotAbsence (fatal and partial ignored census),
TestContentIgnoreCensusWarningRefuses (exit-0 diagnostics with empty/partial
output and an unclassified diagnostic),
TestContentExcludePolicyReadCompleteness (real Git info/exclude and
core.excludesFile: missing, readable, unreadable, restoration/retry; nil snapshot
on refusal and unchanged real index),
TestContentDigestRaceAndRetry (same-size edit hidden from ordinary Git diff),
TestContentBlobInstallFailureAndRetry (unsafe object shard ownership), and
TestContentCrashAfterInstall (real child exit 73 after installs before result;
fresh capture reuses verified objects). The child exit is expected failure,
not a passing capture. No durable snapshot/manifest publication is performed.
The existing localstore fault suite remains the owner of fsync/install crashes.

CR1 F1 (failed-read-as-absence, repeat-of:none) is addressed at the content
census boundary. Exit status and stderr are retained together; any diagnostic
leaves policy completeness unproven and refuses without a filename or message
allowlist. This conservative contract does not classify benign warnings as
safe. Missing optional policies emit no diagnostic in the tested Git version.
Mode-000 permission witnesses explicitly skip where the process can still read
the file or Windows permission semantics apply; those skips prove no refusal.
This stationary read check does not establish final race closure.

## Shared boundaries and strings

- Content entry maps have the exact directory/file/symlink Section 10.4 shape.
  They use the same canonicaljson validator as Transfer Manifest identity.
  CheckManifestEntries's 65536/65537 boundary is executed by its named test and
  the declared-bound proof census. The shared owner checks sorting, overlap,
  case folding, paths, modes, targets and field presence; the capture walker
  checks live filesystem containment in addition.
- Blob descriptors use exact schema/version, SHA-256 identities and 4 MiB chunks.
  Data streams with fixed memory per file; no truncation or large-file omission.
  Full byte-stream success is measured at 8 MiB + 4 bytes, and refusal is
  measured on a real sparse file of 128 GiB + 1 byte. A full successful 128 GiB
  transfer is not measured. Descriptor identity is verified through the shared
  canonicaljson owner; a descriptor's raw stored representation has its own
  byte digest, distinct from its omit-self descriptor identity.
- New strings are normalized relative paths (scalar), literal entry/schema/MIME
  tags, SHA-256 IDs (scalar plus canonicaljson), validated UTF-8 symlink targets
  (canonicaljson plus live containment), and sanitized submodule URL/identity
  (scalar and existing deriveIdentity). Native paths stay internal metadata.
  Exclusion classes and include classifications are trusted policy configuration,
  not evidence from the captured files. Policy root reads distinguish absence
  from errors and normalize macOS aliases before comparison with Git roots.
- The single selection owner handles all child prefixes. Built-in conventional
  filename exclusions are bounded, not a comprehensive content scanner.
  Arbitrarily named auth/runtime/control artifacts require explicit Exclude or
  LocalPaths policy. A non-secret include cannot override an exclusion.
  secprim.Redact is deliberately not used to attest payload confidentiality.
- Relative submodule mappings follow the documented Git default-remote directory
  rule (https://git-scm.com/docs/git-submodule). A missing/machine-local default
  remote refuses instead of copying local config. Initialized child objects are
  never queried via the parent object database. The pinned corpus test imports
  its supplied packs into distinct databases; pack production is not claimed.

## Evidence bounds

Object-pack production, raw-index blob delivery, root/child manifest assembly,
quiescence, end-to-end materialization, and full race/closure strengthening remain
separate Story work. These types intentionally are not complete Git wire members.
The existing private-index algorithm is unchanged. Its auxiliary shared-index
mtime and hard-kill temp cleanup bounds remain in the predecessor review.
Included bytes are checked before returning, but this is not an atomic snapshot
and cannot detect change-and-revert between observations. A full adversarial
filesystem substitution matrix, sparse-pattern changes after reading, every
submodule grammar clause/depth combination, and native Linux/Windows execution
are not established by this leaf's test count.

Mutation vectors in content-mutations.json retain the gates while admitting
selected rejected members. The task-scoped outcome contains actual compile and
behavior exits, named failing tests, controls and every survivor bound. A source
census count is only syntactic; it is not behavioral gate coverage. The neutral
and known-bad controls must succeed before interpreting the battery.
