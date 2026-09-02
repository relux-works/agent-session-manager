# TASK-260830-8x76g1 — closed-shape per-member constraint enumeration

Normative source: `relux-works/agent-session-manager-spec` v0.5.0 Sections 1.6,
2.1, 5.1, 10.1–10.4, and 17.3 at commit
`28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`, SPEC.md SHA-256
`562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a`.

Every row below is invoked through `CalculateObjectIdentity` and
`VerifyObjectIdentity` via `prepareObjectIdentity -> validateImmutableObjectShape`,
except when the earlier 5,242,880-byte encoded-object limit refuses an object before
shape validation. The `GitIndex.entries` declared maximum is the one enumerated bound
whose minimum closed representation cannot fit below that outer limit; its validator
boundary is pinned directly and both public entries are proven to refuse the oversized
representation before attestation rather than being reported as accepting it.
`TestConstraintEnumerationMatchesRequireExactMembers` derives the member set from
the production `requireExactMembers` argument lists and requires an exact one-to-one
artifact row, so a new, removed, or renamed member fails the suite. “Type-only” means the
pinned SPEC gives that member no additional constraint beyond the stated JSON/common type;
that row quotes the bare type text verbatim. The row table covers each member's declared
type/local constraint. Reachable tagged, ordering, recursive, cross-member, and external
rules are enumerated separately after the table so a local check is not misreported as a
cross-object proof. `TestEveryFixtureClosedShapeMemberIsRequiredAtBothIdentityEntries`
recursively deletes every member from valid fixtures spanning all 24 closed shapes and
requires both public identity entries to refuse, so requiredness is executable rather than
an artifact-only claim.

| Shape | Member | Enforced constraint | Production call site | Pinned SPEC declaration |
| --- | --- | --- | --- | --- |
| `Blob Descriptor` | `blob_id` | Enforced exactly as declared before identity calculation or verification. | `validateBlobDescriptor` | “digest” |
| `Blob Descriptor` | `chunks` | Enforced exactly as declared before identity calculation or verification. | `validateBlobDescriptor` | “BlobChunk[0..32768]; empty exactly when size is zero and exact coverage otherwise” |
| `Blob Descriptor` | `descriptor_id` | Enforced exactly as declared before identity calculation or verification. | `validateBlobDescriptor` | “digest; canonical object digest” |
| `Blob Descriptor` | `media_type` | Enforced exactly as declared before identity calculation or verification. | `validateBlobDescriptor` | “string[1..255]; lowercase ASCII type/subtype without parameters” |
| `Blob Descriptor` | `schema` | Enforced exactly as declared before identity calculation or verification. | `validateBlobDescriptor` | “string; exact urn:ax:schema:blob” |
| `Blob Descriptor` | `schema_version` | Enforced exactly as declared before identity calculation or verification. | `validateBlobDescriptor` | “string; exact 1.0.0” |
| `Blob Descriptor` | `size` | Enforced exactly as declared before identity calculation or verification. | `validateBlobDescriptor` | “uint53” |
| `BlobChunk` | `chunk_id` | Enforced exactly as declared before identity calculation or verification. | `validateBlobDescriptor` | “digest” |
| `BlobChunk` | `index` | Enforced exactly as declared before identity calculation or verification. | `validateBlobDescriptor` | “uint32; starts at zero and increases by one” |
| `BlobChunk` | `offset` | Enforced exactly as declared before identity calculation or verification. | `validateBlobDescriptor` | “uint53; contiguous from zero” |
| `BlobChunk` | `size` | Enforced exactly as declared before identity calculation or verification; `TestTrailingBlobChunkSizeMinimumReachesBothIdentityEntries` pins a trailing zero-size chunk at the exact refusal clause through both public entries. | `validateBlobDescriptor` | “uint53[1..4194304]; every non-final chunk is exactly 4194304” |
| `GitFeatures` | `case_sensitive` | Enforced exactly as declared before identity calculation or verification. | `validateGitFeatures` | “boolean” |
| `GitFeatures` | `filemode` | Enforced exactly as declared before identity calculation or verification. | `validateGitFeatures` | “boolean” |
| `GitFeatures` | `lfs_required` | Enforced exactly as declared before identity calculation or verification. | `validateGitFeatures` | “boolean” |
| `GitFeatures` | `object_format` | Enforced exactly as declared before identity calculation or verification. | `validateGitFeatures` | “enum sha1 or sha256” |
| `GitFeatures` | `precompose_unicode` | Enforced exactly as declared before identity calculation or verification. | `validateGitFeatures` | “boolean” |
| `GitFeatures` | `required_filter_names` | Enforced exactly as declared before identity calculation or verification. | `validateGitFeatures` | “sorted unique string[0..64]” |
| `GitFeatures` | `sparse_checkout` | Enforced exactly as declared before identity calculation or verification. | `validateGitFeatures` | “boolean; tags sparse pattern digest pair” |
| `GitFeatures` | `sparse_patterns_blob_descriptor_id` | Enforced exactly as declared before identity calculation or verification. | `validateGitFeatures` | “digest or null” |
| `GitFeatures` | `sparse_patterns_blob_id` | Enforced exactly as declared before identity calculation or verification. | `validateGitFeatures` | “digest or null” |
| `GitFeatures` | `symlinks` | Enforced exactly as declared before identity calculation or verification. | `validateGitFeatures` | “boolean” |
| `GitHead` | `mode` | Enforced exactly as declared before identity calculation or verification. | `validateGitHead` | “enum branch, detached, or unborn” |
| `GitHead` | `oid` | Enforced exactly as declared before identity calculation or verification. | `validateGitHead` | “git-oid or null; tagged by mode and matching object format” |
| `GitHead` | `ref` | Enforced exactly as declared before identity calculation or verification. | `validateGitHead` | “git-ref or null; tagged by mode and unborn uses refs/heads/” |
| `GitIndex` | `blob_descriptor_id` | Enforced exactly as declared before identity calculation or verification. | `validateGitIndex` | “digest” |
| `GitIndex` | `blob_id` | Enforced exactly as declared before identity calculation or verification. | `validateGitIndex` | “digest” |
| `GitIndex` | `entries` | Production enforces 0..65536 and a direct boundary test pins accept-at-65536/refuse-at-65537. Public-entry acceptance at 65536 is not claimed: the required closed entries encode above 5,242,880 bytes and `prepareObjectIdentity` refuses the object first. | `validateGitIndex` | “GitIndexEntry[0..65536]; sorted by path then stage” |
| `GitIndex` | `entry_count` | Enforced exactly as declared before identity calculation or verification. | `validateGitIndex` | “uint53; equals entries length” |
| `GitIndex` | `format` | Enforced exactly as declared before identity calculation or verification. | `validateGitIndex` | “exact git_index” |
| `GitIndex` | `version` | Enforced exactly as declared before identity calculation or verification. | `validateGitIndex` | “enum 2, 3, or 4” |
| `GitIndexEntry` | `assume_unchanged` | Enforced exactly as declared before identity calculation or verification. | `validateGitIndexEntry` | “boolean” |
| `GitIndexEntry` | `fsmonitor_valid` | Enforced exactly as declared before identity calculation or verification. | `validateGitIndexEntry` | “boolean” |
| `GitIndexEntry` | `intent_to_add` | Enforced exactly as declared before identity calculation or verification. | `validateGitIndexEntry` | “boolean” |
| `GitIndexEntry` | `mode` | Enforced exactly as declared before identity calculation or verification. | `validateGitIndexEntry` | “uint32” |
| `GitIndexEntry` | `oid` | Enforced exactly as declared before identity calculation or verification. | `validateGitIndexEntry` | “git-oid; matches object format” |
| `GitIndexEntry` | `path` | Enforced exactly as declared before identity calculation or verification. | `validateGitIndexEntry` | “path” |
| `GitIndexEntry` | `skip_worktree` | Enforced exactly as declared before identity calculation or verification. | `validateGitIndexEntry` | “boolean” |
| `GitIndexEntry` | `stage` | Enforced exactly as declared before identity calculation or verification. | `validateGitIndexEntry` | “uint8[0..3]” |
| `GitObjectPack` | `blob_descriptor_id` | Enforced exactly as declared before identity calculation or verification. | `validateGitObjectPack` | “digest” |
| `GitObjectPack` | `blob_id` | Enforced exactly as declared before identity calculation or verification. | `validateGitObjectPack` | “digest” |
| `GitObjectPack` | `format` | Enforced exactly as declared before identity calculation or verification. | `validateGitObjectPack` | “exact git_pack_v2” |
| `GitObjectPack` | `inventory_blob_descriptor_id` | Enforced exactly as declared before identity calculation or verification. | `validateGitObjectPack` | “digest” |
| `GitObjectPack` | `inventory_blob_id` | Enforced exactly as declared before identity calculation or verification. | `validateGitObjectPack` | “digest” |
| `GitObjectPack` | `object_count` | Enforced exactly as declared before identity calculation or verification. | `validateGitObjectPack` | “uint53” |
| `GitObjectPack` | `object_format` | Enforced exactly as declared before identity calculation or verification. | `validateGitObjectPack` | “enum sha1 or sha256” |
| `GitRemote` | `fetch_url` | Enforced exactly as declared before identity calculation or verification. | `validateGitRemote` | “sanitized-git-URL” |
| `GitRemote` | `name` | Enforced exactly as declared before identity calculation or verification. | `validateGitRemote` | “string[1..128]; remotes sorted by name with no duplicate” |
| `GitRemote` | `push_url` | Enforced exactly as declared before identity calculation or verification. | `validateGitRemote` | “sanitized-git-URL or null” |
| `GitSubmodule` | `agent_project_config_paths` | Enforced exactly as declared before identity calculation or verification. | `validateGitSubmodule` | “sorted unique path[0..256] or null” |
| `GitSubmodule` | `features` | Enforced exactly as declared before identity calculation or verification. | `validateGitSubmodule` | “GitFeatures or null” |
| `GitSubmodule` | `gitlink_oid` | Enforced exactly as declared before identity calculation or verification. | `validateGitSubmodule` | “git-oid; equals containing stage-0 mode-160000 entry” |
| `GitSubmodule` | `head` | Enforced exactly as declared before identity calculation or verification. | `validateGitSubmodule` | “GitHead or null” |
| `GitSubmodule` | `index` | Enforced exactly as declared before identity calculation or verification. | `validateGitSubmodule` | “GitIndex or null” |
| `GitSubmodule` | `initialized` | Enforced exactly as declared before identity calculation or verification. | `validateGitSubmodule` | “boolean; tags all following state members” |
| `GitSubmodule` | `object_pack` | Enforced exactly as declared before identity calculation or verification. | `validateGitSubmodule` | “GitObjectPack or null” |
| `GitSubmodule` | `path` | Enforced exactly as declared before identity calculation or verification. | `validateGitSubmodule` | “path; no sibling destination-case collision” |
| `GitSubmodule` | `repo_relative_cwd` | Enforced exactly as declared before identity calculation or verification. | `validateGitSubmodule` | “dot or path or null” |
| `GitSubmodule` | `repository_identity` | Enforced exactly as declared before identity calculation or verification. | `validateGitSubmodule` | “string[1..256]; recursion acyclic by identity” |
| `GitSubmodule` | `sanitized_url` | Enforced exactly as declared before identity calculation or verification. | `validateGitSubmodule` | “sanitized-git-URL” |
| `GitSubmodule` | `submodules` | Enforced exactly as declared before identity calculation or verification. | `validateGitSubmodule` | “GitSubmodule[0..256] or null; depth at most 16 and total at most 256” |
| `GitSubmodule` | `upstream_ref` | Enforced exactly as declared before identity calculation or verification. | `validateGitSubmodule` | “git-ref or null” |
| `GitSubmodule` | `working_tree_manifest_id` | Enforced exactly as declared before identity calculation or verification. | `validateGitSubmodule` | “digest or null” |
| `ManifestEntry.directory` | `mode` | Enforced exactly as declared before identity calculation or verification. | `validateManifestEntries` | “uint32[0..4095]” |
| `ManifestEntry.directory` | `path` | Enforced exactly as declared before identity calculation or verification. | `validateManifestEntries` | “path” |
| `ManifestEntry.directory` | `type` | Enforced exactly as declared before identity calculation or verification. | `validateManifestEntries` | “exact directory” |
| `ManifestEntry.file` | `blob_descriptor_id` | Enforced exactly as declared before identity calculation or verification. | `validateManifestEntries` | “digest” |
| `ManifestEntry.file` | `blob_id` | Enforced exactly as declared before identity calculation or verification. | `validateManifestEntries` | “digest” |
| `ManifestEntry.file` | `mode` | Enforced exactly as declared before identity calculation or verification. | `validateManifestEntries` | “uint32[0..4095]” |
| `ManifestEntry.file` | `path` | Enforced exactly as declared before identity calculation or verification. | `validateManifestEntries` | “path” |
| `ManifestEntry.file` | `size` | Enforced exactly as declared before identity calculation or verification. | `validateManifestEntries` | “uint53” |
| `ManifestEntry.file` | `type` | Enforced exactly as declared before identity calculation or verification. | `validateManifestEntries` | “exact file” |
| `ManifestEntry.hardlink` | `mode` | Enforced exactly as declared before identity calculation or verification. | `validateManifestEntries` | “uint32[0..4095]” |
| `ManifestEntry.hardlink` | `path` | Enforced exactly as declared before identity calculation or verification. | `validateManifestEntries` | “path” |
| `ManifestEntry.hardlink` | `target_path` | Enforced exactly as declared before identity calculation or verification. | `validateManifestEntries` | “path; names an earlier file entry with the same mode” |
| `ManifestEntry.hardlink` | `type` | Enforced exactly as declared before identity calculation or verification. | `validateManifestEntries` | “exact hardlink” |
| `ManifestEntry.symlink` | `mode` | Enforced exactly as declared before identity calculation or verification. | `validateManifestEntries` | “uint32[0..4095]” |
| `ManifestEntry.symlink` | `path` | Enforced exactly as declared before identity calculation or verification. | `validateManifestEntries` | “path” |
| `ManifestEntry.symlink` | `target` | Enforced exactly as declared before identity calculation or verification; `TestSymlinkTargetLowerBoundReachesBothIdentityEntries` pins accept-at-one and refuse-below-one through both public entries. | `validateManifestEntries` | “string[1..4096]; lexically remains within materialization root” |
| `ManifestEntry.symlink` | `type` | Enforced exactly as declared before identity calculation or verification. | `validateManifestEntries` | “exact symlink” |
| `MigrationProvenance` | `object_id` | Enforced exactly as declared before identity calculation or verification. | `validateMigrationExtensionObject` | “digest” |
| `MigrationProvenance` | `schema_id` | Type-only: requires a valid UTF-8 JSON string and deliberately adds no non-empty or length rule. | `validateMigrationExtensionObject` | “string” |
| `MigrationProvenance` | `schema_version` | Enforced exactly as declared before identity calculation or verification. | `validateMigrationExtensionObject` | “canonical semver” |
| `Session Record 1.0.0` | `created_at` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | “timestamp; diagnostic time” |
| `Session Record 1.0.0` | `created_by_host_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | “UUIDv7; allowlisted host at creation” |
| `Session Record 1.0.0` | `execution_profile` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | “enum standard or yolo” |
| `Session Record 1.0.0` | `extensions` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | “object; required, reverse-DNS keys only” |
| `Session Record 1.0.0` | `fork_provenance` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | “Fork Provenance or null; object exactly when created by fork” |
| `Session Record 1.0.0` | `kind` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | “enum direct or task_board” |
| `Session Record 1.0.0` | `launch_plan` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | “Launch Plan; closed, sanitized and secret-free” |
| `Session Record 1.0.0` | `name` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | “string; Section 2.3 grammar [A-Za-z0-9][A-Za-z0-9._-]{0,63} and 1–64 characters” |
| `Session Record 1.0.0` | `provider_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | “string; lowercase plugin ID” |
| `Session Record 1.0.0` | `record_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | “digest; canonical object digest” |
| `Session Record 1.0.0` | `schema` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | “string; exact schema identifier” |
| `Session Record 1.0.0` | `schema_version` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | “semver; exact 1.0.0” |
| `Session Record 1.0.0` | `session_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | “UUIDv7; globally unique” |
| `Session Record 1.0.0` | `subject_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | “UUIDv7; equal to session_id” |
| `Session Record 1.0.0` | `task_board` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | “Task-board Reference or null; object exactly when kind is task_board” |
| `Session Record 1.0.0` | `workspace_group_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | “UUIDv7; required” |
| `Session Record Board Goal` | `extensions` | Enforced exactly as declared before identity calculation or verification. | `validateSessionBoardGoal` | “object; reverse-DNS extension keys only” |
| `Session Record Board Goal` | `goal_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionBoardGoal` | “string[1..128]; public goal reference” |
| `Session Record Board Goal` | `revision` | Enforced exactly as declared before identity calculation or verification. | `validateSessionBoardGoal` | “uint53 greater than zero” |
| `Session Record Board Goal` | `schema` | Enforced exactly as declared before identity calculation or verification. | `validateSessionBoardGoal` | “string; exact board-goal-v2” |
| `Session Record Board Identity` | `extensions` | Enforced exactly as declared before identity calculation or verification. | `validateSessionBoardIdentity` | “object; reverse-DNS extension keys only” |
| `Session Record Board Identity` | `kind` | Enforced exactly as declared before identity calculation or verification. | `validateSessionBoardIdentity` | “enum local or remote” |
| `Session Record Board Identity` | `logical_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionBoardIdentity` | “string[1..128]; [A-Za-z0-9][A-Za-z0-9._:-]{0,127}” |
| `Session Record Board Identity` | `remote_url` | Enforced exactly as declared before identity calculation or verification. | `validateSessionBoardIdentity` | “absolute HTTPS URL or null; tagged by kind and no userinfo, query, or fragment” |
| `Session Record Fork Provenance` | `extensions` | Enforced exactly as declared before identity calculation or verification. | `validateSessionForkProvenance` | “object; reverse-DNS extension keys only” |
| `Session Record Fork Provenance` | `operation_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionForkProvenance` | “UUIDv7” |
| `Session Record Fork Provenance` | `provider_fork_mode` | Enforced exactly as declared before identity calculation or verification. | `validateSessionForkProvenance` | “enum native, supported_import, or task_board_clone” |
| `Session Record Fork Provenance` | `source_checkpoint_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionForkProvenance` | “digest” |
| `Session Record Fork Provenance` | `source_session_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionForkProvenance` | “UUIDv7” |
| `Session Record Fork Provenance` | `source_workspace_group_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionForkProvenance` | “UUIDv7” |
| `Session Record Launch Plan` | `argv` | Enforced exactly as declared before identity calculation or verification. | `validateSessionLaunchPlan` | “array<string>[1..128]; each 1–4096 UTF-8 bytes and total encoded argv at most 65536 bytes” |
| `Session Record Launch Plan` | `contains_secrets` | Enforced exactly as declared before identity calculation or verification. | `validateSessionLaunchPlan` | “boolean; MUST be false” |
| `Session Record Launch Plan` | `cwd_relative` | Enforced exactly as declared before identity calculation or verification. | `validateSessionLaunchPlan` | “string; dot or Section 1.6 path” |
| `Session Record Launch Plan` | `cwd_workspace_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionLaunchPlan` | “UUIDv7; names one workspace in the record workspace group” |
| `Session Record Launch Plan` | `env_literals` | Enforced exactly as declared before identity calculation or verification. | `validateSessionLaunchPlan` | “map(environment-name,string)[0..64]; values at most 4096 UTF-8 bytes and keys disjoint from env_names” |
| `Session Record Launch Plan` | `env_names` | Enforced exactly as declared before identity calculation or verification. | `validateSessionLaunchPlan` | “array<string>[0..64]; sorted unique environment names” |
| `Session Record Launch Plan` | `extensions` | Enforced exactly as declared before identity calculation or verification. | `validateSessionLaunchPlan` | “object; reverse-DNS extension keys only” |
| `Session Record Task-board Reference` | `board` | Enforced exactly as declared before identity calculation or verification. | `validateSessionTaskBoardReference` | “Board Identity; closed shape” |
| `Session Record Task-board Reference` | `board_goal` | Enforced exactly as declared before identity calculation or verification. | `validateSessionTaskBoardReference` | “Board Goal or null; non-null for primary_owner” |
| `Session Record Task-board Reference` | `bridge_protocol_version` | Enforced exactly as declared before identity calculation or verification. | `validateSessionTaskBoardReference` | “semver; exact 1.0.0” |
| `Session Record Task-board Reference` | `extensions` | Enforced exactly as declared before identity calculation or verification. | `validateSessionTaskBoardReference` | “object; reverse-DNS extension keys only” |
| `Session Record Task-board Reference` | `launch_mode` | Enforced exactly as declared before identity calculation or verification. | `validateSessionTaskBoardReference` | “enum primary_owner or tracked_prompt” |
| `Session Record Task-board Reference` | `manager_session_ref` | Enforced exactly as declared before identity calculation or verification. | `validateSessionTaskBoardReference` | “string or null; MUST be null in immutable creation record” |
| `Session Record Task-board Reference` | `native_goal_binding` | Enforced exactly as declared before identity calculation or verification. | `validateSessionTaskBoardReference` | “enum bound, prompt, or none; bound exactly for primary_owner” |
| `Session Record Task-board Reference` | `task_element_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionTaskBoardReference` | “string; 1–128 printable non-control UTF-8 bytes” |
| `Transfer Manifest` | `base_checkpoint_id` | Enforced exactly as declared before identity calculation or verification. | `validateTransferManifest` | “digest or null” |
| `Transfer Manifest` | `child_manifest_ids` | Enforced exactly as declared before identity calculation or verification. | `validateTransferManifest` | “sorted unique digest[0..1024]” |
| `Transfer Manifest` | `created_at` | Enforced exactly as declared before identity calculation or verification. | `validateTransferManifest` | “timestamp” |
| `Transfer Manifest` | `created_by_host_id` | Enforced exactly as declared before identity calculation or verification. | `validateTransferManifest` | “UUIDv7” |
| `Transfer Manifest` | `entries` | Enforced exactly as declared before identity calculation or verification. | `validateTransferManifest` | “ManifestEntry[0..65536]; strictly bytewise sorted with no destination-case collision” |
| `Transfer Manifest` | `excluded_classes` | Enforced exactly as declared before identity calculation or verification. | `validateTransferManifest` | “sorted unique string[0..128]” |
| `Transfer Manifest` | `extensions` | Enforced exactly as declared before identity calculation or verification. | `validateTransferManifest` | “object; reverse-DNS extension keys only” |
| `Transfer Manifest` | `kind` | Enforced exactly as declared before identity calculation or verification. | `validateTransferManifest` | “enum workspace_group, workspace_tree, provider, task_board, or composite” |
| `Transfer Manifest` | `manifest_id` | Enforced exactly as declared before identity calculation or verification. | `validateTransferManifest` | “digest; canonical object digest” |
| `Transfer Manifest` | `provider_identity_record_id` | Enforced exactly as declared before identity calculation or verification. | `validateTransferManifest` | “digest or null; non-null only for provider” |
| `Transfer Manifest` | `schema` | Enforced exactly as declared before identity calculation or verification. | `validateTransferManifest` | “string; exact Transfer Manifest schema identifier” |
| `Transfer Manifest` | `schema_version` | Enforced exactly as declared before identity calculation or verification. | `validateTransferManifest` | “semver; exact 1.0.0” |
| `Transfer Manifest` | `subject_id` | Enforced exactly as declared before identity calculation or verification. | `validateTransferManifest` | “UUIDv7; scope selected by kind” |
| `Transfer Manifest` | `task_board_bundle_id` | Enforced exactly as declared before identity calculation or verification. | `validateTransferManifest` | “digest or null; non-null only for task_board” |
| `Transfer Manifest` | `workspace_snapshot` | Enforced exactly as declared before identity calculation or verification. | `validateTransferManifest` | “WorkspaceSnapshot or null; non-null only for workspace_group” |
| `WorkspaceSnapshot` | `members` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshot` | “WorkspaceSnapshotMember[1..256]; strict workspace-ID order and no destination-case-colliding group paths” |
| `WorkspaceSnapshot` | `workspace_group_id` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshot` | “UUIDv7; equals manifest subject” |
| `WorkspaceSnapshotMember.git` | `agent_project_config_paths` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | “sorted unique path[0..256]” |
| `WorkspaceSnapshotMember.git` | `features` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | “GitFeatures” |
| `WorkspaceSnapshotMember.git` | `group_relative_path` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | “path” |
| `WorkspaceSnapshotMember.git` | `head` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | “GitHead” |
| `WorkspaceSnapshotMember.git` | `index` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | “GitIndex” |
| `WorkspaceSnapshotMember.git` | `kind` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | “exact git” |
| `WorkspaceSnapshotMember.git` | `materialization_policy` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | “enum shared_checkout or separate_worktree” |
| `WorkspaceSnapshotMember.git` | `object_pack` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | “GitObjectPack” |
| `WorkspaceSnapshotMember.git` | `remotes` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | “GitRemote[1..16]; sorted by name with no duplicate” |
| `WorkspaceSnapshotMember.git` | `repo_relative_cwd` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | “dot or path” |
| `WorkspaceSnapshotMember.git` | `repository_identity` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | “string[1..256]” |
| `WorkspaceSnapshotMember.git` | `submodules` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | “GitSubmodule[0..256]” |
| `WorkspaceSnapshotMember.git` | `upstream_ref` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | “git-ref or null” |
| `WorkspaceSnapshotMember.git` | `working_tree_manifest_id` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | “digest” |
| `WorkspaceSnapshotMember.git` | `workspace_id` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | “UUIDv7” |
| `WorkspaceSnapshotMember.managed_tree` | `agent_project_config_paths` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | “sorted unique path[0..256]” |
| `WorkspaceSnapshotMember.managed_tree` | `group_relative_path` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | “path” |
| `WorkspaceSnapshotMember.managed_tree` | `kind` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | “exact managed_tree” |
| `WorkspaceSnapshotMember.managed_tree` | `materialization_policy` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | “enum shared_tree or separate_copy” |
| `WorkspaceSnapshotMember.managed_tree` | `repo_relative_cwd` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | “dot or path” |
| `WorkspaceSnapshotMember.managed_tree` | `tree_identity` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | “string[1..256]” |
| `WorkspaceSnapshotMember.managed_tree` | `tree_manifest_id` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | “digest” |
| `WorkspaceSnapshotMember.managed_tree` | `workspace_id` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | “UUIDv7” |

## Reachable cross-member, recursive, and external rules

| Scope | Pinned rule | Disposition and production call site |
| --- | --- | --- |
| Session Record | `subject_id` equals `session_id`. | Enforced by `validateSessionRecordV1`. |
| Session Record | `task_board` is non-null exactly for `kind=task_board`. | Enforced by `validateSessionTaskBoardReference`; direct and task-board positive variants plus both refusal directions drive both public entries. |
| Session Record | `fork_provenance` is null unless the record was created by fork. | The nullable closed shape is enforced by `validateSessionForkProvenance`; whether an external creation operation actually was a fork requires that operation evidence and is outside one identity candidate. |
| Launch Plan | `argv` is non-empty, each element is 1–4096 UTF-8 bytes, and encoded argv is at most 65,536 bytes. | Enforced by `validateSessionLaunchPlan`. The broader semantic prohibition on secret-bearing or shell-fragment arguments requires provider/secret classification and is external to syntax-only identity validation. |
| Launch Plan | `env_names` is sorted and unique; `env_literals` keys use the same grammar and are disjoint. | Enforced by `validateSessionLaunchPlan`; `TestSpecDerivedRefusalClausesReachBothIdentityEntries` pins the literal-key grammar through both public entries. Whether a literal is semantically secret requires secret classification and remains external. |
| Task-board Reference | Creation-time `manager_session_ref` is null; `primary_owner` requires a non-null goal and `bound`; `tracked_prompt` permits only `prompt` or `none`. | Enforced by `validateSessionTaskBoardReference`. |
| Board Identity | Local requires null URL; remote requires absolute HTTPS with host and no userinfo, query, or fragment. | Enforced by `validateSessionBoardIdentity`; `TestSpecDerivedRefusalClausesReachBothIdentityEntries` pins the local-null arm through both public entries. |
| Blob Descriptor | Empty size has no chunks; non-empty has at least one; indexes start at zero, offsets are contiguous, non-final chunks are fixed 4 MiB, and chunks cover size exactly without overflow. | Enforced by `validateBlobDescriptor`. |
| Blob Descriptor | Descriptor size/digest and chunk digests match referenced raw bytes. | External: raw bytes are not members of the descriptor identity candidate. |
| Transfer Manifest | The five `kind` arms impose exact nullability plus empty/non-empty entry and child rules. | Enforced by `validateTransferManifest`. |
| Transfer Manifest | Entries are strictly bytewise path-sorted and have no destination-case collisions. | Enforced by `validateManifestEntries`; collision membership is O(total path characters) through `simpleFoldKey`, and the 65,536-entry production calculation guard is below 2 seconds. |
| Transfer Manifest | Entry and child partitions have no duplicate, overlapping, or destination-case-colliding paths. | Entry-local duplicate/case collision is enforced. Child partition paths require the referenced child manifests and are external to one candidate. |
| Transfer Manifest | Every file entry agrees with its referenced Blob Descriptor on blob ID and size. | External: referenced descriptors are not members of this candidate. |
| Transfer Manifest | A hardlink names an earlier file with the same mode; a symlink target stays lexically within the materialization root. | Enforced by `validateManifestEntries`; `TestSpecDerivedRefusalClausesReachBothIdentityEntries` pins both target type guards through both public entries, while filesystem resolution is explicitly external. |
| Transfer Manifest | `excluded_classes` diagnoses every applicable capture exclusion. | Syntax/order/count are enforced; capture-selection completeness requires capture evidence and is external. |
| WorkspaceSnapshot | Group ID equals manifest subject; members are non-empty, workspace-ID ordered, and group paths do not case-collide. | Enforced by `validateWorkspaceSnapshot`; `TestSpecDerivedRefusalClausesReachBothIdentityEntries` pins group/subject equality through both public entries. One-for-one equality with the referenced Workspace Group Record is external. |
| Git remotes | Non-empty, name-sorted, and unique. | Enforced by `validateGitRemotes`; URL grammar reuses `scalar.ParseSanitizedGitURL`. |
| GitHead | Branch has OID and ref, detached has only OID, unborn has only a `refs/heads/` ref. | Enforced by `validateGitHead`; `TestSpecDerivedRefusalClausesReachBothIdentityEntries` pins malformed-ref, branch, and unborn refusals through both public entries, and `TestHelperLevelRefusalsAndUnbornHeadReachBothIdentityEntries` pins valid unborn acceptance. OID format is checked against `GitFeatures` by `validateGitHeadObjectFormat`. |
| GitIndex | Version is 2–4; entries are path/stage ordered; count equals length; stage is 0–3. | Enforced by `validateGitIndex` and `validateGitIndexEntry`, including TM-GIT-N2 stage-4 refusal. Raw-index equality and required extension interpretation need referenced bytes and are external. |
| Git object formats | Pack, head, index, feature, and submodule OIDs agree on SHA-1 or SHA-256. | Enforced by `validateWorkspaceSnapshotMember`, `validateGitSubmodule`, `validateGitHeadObjectFormat`, and `validateGitIndexObjectFormat`; `TestSpecDerivedRefusalClausesReachBothIdentityEntries` pins the submodule pack/features mismatch through both public entries. Pack inventory/object reachability and repository isolation require referenced Git bytes and are external. |
| GitFeatures | Sparse pattern IDs are both non-null exactly when sparse checkout is true. | Enforced by `validateGitFeatures`. |
| GitSubmodule | `gitlink_oid` equals a containing stage-0 mode-160000 index entry; initialized state is complete and not unborn; uninitialized state is all null. | Enforced recursively by `validateGitSubmodule`; `TestSpecDerivedRefusalClausesReachBothIdentityEntries` pins missing initialized state and unborn initialized head through both public entries. Resolving parent HEAD-tree and child HEAD commits requires isolated object databases and is external. |
| GitSubmodule | Tree depth is at most 16, total count at most 256, repository identities are ancestor-acyclic, and sibling paths do not case-collide. | Enforced by `validateGitSubmodules` and `validateGitSubmodule`; `TestSpecDerivedRefusalClausesReachBothIdentityEntries` pins sibling case collision through both public entries and is mutation-checked with the narrowing `strings.EqualFold` to exact-equality substitution. |
| Manifest closure | Every referenced working-tree/managed-tree/submodule manifest, cwd, and project config path occurs in the transitive child closure. | External: child manifests and materialized filesystem are outside one identity candidate. |
| Decoder robustness bound (implementation, not a pinned SPEC clause) | Every public entry refuses a document nested deeper than `maxNestingDepth` (256 containers) with a typed `ErrInvalidJSON` error instead of a fatal stack overflow; the deepest pinned closed shape, the 16-level submodule tree, decodes at roughly depth 40. | Enforced by `decodeValue` inside `decodeStrict`, shared by `Canonicalize`, `CalculateObjectIdentity`, and `VerifyObjectIdentity` before number, schema, or shape validation; `TestMaxNestingDepthIsDeclaredAndPinned` pins the literal, `TestCanonicalizeAcceptsDocumentAtMaxNestingDepth`, `TestCanonicalizeRefusesDocumentPastMaxNestingDepth`, `TestIdentityEntriesAcceptDocumentAtMaxNestingDepth`, and `TestIdentityEntriesRefuseDocumentPastMaxNestingDepth` prove accept-at-256 and refuse-at-257 from a test-local literal in array, object, and mixed shapes so widening or narrowing the constant reddens the suite, and `TestNestingDepthRegressionTwoMegabyteArrayReturnsTypedError` replays the 2,000,000-byte one-million-level crash input through every entry. |
| Section 1.6 extensions | Every open extension point has 0–64 lowercase reverse-DNS keys, depth at most 4, and canonical size at most 65,536 bytes. | Enforced by `validateExtensionsObject` before either public entry attests. |
| Section 17.3 migration contribution | `works.relux.ax.migrated-from` is a closed three-member object; `schema_id` is bare `string`, `schema_version` canonical semver, and `object_id` digest. | Enforced by `validateMigrationExtensionObject`; publication, atomic-reference advancement, rollback retention, and runtime migration remain explicitly unclaimed. |
