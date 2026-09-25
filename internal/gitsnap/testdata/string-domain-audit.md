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
