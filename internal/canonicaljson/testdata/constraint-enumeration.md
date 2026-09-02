# Closed-shape per-member constraint enumeration

Normative source: `relux-works/agent-session-manager-spec` v0.5.0 Sections 1.6,
2.1–2.4, 5.1–5.6, 7.8, 10.1–10.4, 13.14.5, 13.15, and 17.3 at commit
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
that row quotes the bare type text verbatim. “Presence-only” means the pinned SPEC names a
required member without declaring its JSON type or a local constraint; the identity gate
therefore proves presence through the exact-member check and deliberately infers nothing
from its name or from a similarly named member in another schema. The row table covers each member's declared
type/local constraint. Reachable tagged, ordering, recursive, cross-member, and external
rules are enumerated separately after the table so a local check is not misreported as a
cross-object proof. `TestEveryFixtureClosedShapeMemberIsRequiredAtBothIdentityEntries`
recursively deletes every member from valid fixtures spanning all 30 closed shapes and
requires both public identity entries to refuse, so requiredness is executable rather than
an artifact-only claim. The Session Record 2.0.0 and 3.0.0 provenance fixture inventory is
also reused to derive every object-valued member path; replacing any such required nested
object with either `null` or a scalar must be refused by both public entries. The shared
Record Envelope's `extensions` object check is defensively redundant for supported Session
Records: their exact top-level member gate already refuses absence, and
`validateMigrationProvenance` refuses a present non-object. Unsupported record schemas are
refused unconditionally after the common envelope check, so relaxing only the shared
required-object lookup cannot create an attested identity.

| Shape | Member | Enforced constraint | Production call site | Pinned SPEC declaration |
| --- | --- | --- | --- | --- |
| `Lease Record` | `schema` | Exact Lease schema identifier. | `validateLeaseRecord` | “Exact Lease Record schema identifier” |
| `Lease Record` | `schema_version` | Exact version 1.0.0. | `validateLeaseRecord` | “1.0.0” |
| `Lease Record` | `record_id` | Canonical digest self identity. | `validateLeaseRecord` | “Canonical Lease Record digest” |
| `Lease Record` | `subject_id` | UUIDv7 equal to session ID. | `validateLeaseRecord` | “Equal to” |
| `Lease Record` | `lease_id` | UUIDv4 fencing token. | `validateLeaseRecord` | “UUIDv4” |
| `Lease Record` | `session_id` | UUIDv7 lease scope. | `validateLeaseRecord` | “Lease scope” |
| `Lease Record` | `epoch` | Positive uint53; the declared epoch-one and successor nullability rules are enforced, and no reason is inferred from the epoch. | `validateLeaseRecord` | “Starts at 1; never decreases” |
| `Lease Record` | `holder_host_id` | UUIDv7 proposed owner. | `validateLeaseRecord` | “Proposed owner” |
| `Lease Record` | `predecessor_lease_id` | UUIDv4 or null; non-null is required after epoch one, and an epoch-one `create` lease must carry null. | `validateLeaseRecord` | “Null only at epoch 1”; “An epoch-1 <code>create</code> lease MUST have a null predecessor” |
| `Lease Record` | `reason` | Closed four-member reason enum. Section 5.3 declares no coupling from the epoch to the reason, so no epoch-one-implies-`create` or `create`-implies-epoch-one rule is enforced. | `validateLeaseRecord` | “<code>create</code>, <code>graceful_takeover</code>, <code>force_takeover</code>, <code>recovery</code>” |
| `Lease Record` | `checkpoint_id` | Digest, and null only for an epoch-one `create` lease; every other epoch and reason combination requires a non-null checkpoint. | `validateLeaseRecord` | “Null only for epoch-1 <code>create</code>; otherwise the validated materialized handoff base” |
| `Lease Record` | `issued_by_host_id` | UUIDv7 initiator. | `validateLeaseRecord` | “Initiator” |
| `Lease Record` | `created_by_host_id` | UUIDv7 equal to issuer. | `validateLeaseRecord` | “MUST equal” |
| `Lease Record` | `created_at` | Timestamp, diagnostic only. | `validateLeaseRecord` | “Diagnostic only” |
| `Lease Record` | `extensions` | Required reverse-DNS extension object. | `validateLeaseRecord` | “Reverse-DNS extension keys only” |
| `Checkpoint Record` | `schema` | Exact Checkpoint schema identifier. | `validateCheckpointRecord` | “Exact Checkpoint schema identifier” |
| `Checkpoint Record` | `schema_version` | Exact version 1.0.0. | `validateCheckpointRecord` | “1.0.0” |
| `Checkpoint Record` | `checkpoint_id` | Canonical digest self identity. | `validateCheckpointRecord` | “Canonical object digest” |
| `Checkpoint Record` | `subject_id` | UUIDv7 equal to session ID. | `validateCheckpointRecord` | “Equal to” |
| `Checkpoint Record` | `session_id` | UUIDv7 existing Session Record reference. | `validateCheckpointRecord` | “Existing Session Record” |
| `Checkpoint Record` | `lease_epoch` | Positive uint53. | `validateCheckpointRecord` | “Greater than zero and equal to the referenced winning lease” |
| `Checkpoint Record` | `lease_id` | UUIDv4 fencing token. | `validateCheckpointRecord` | “UUIDv4” |
| `Checkpoint Record` | `safe_boundary` | Required closed Safe Boundary Evidence. | `validateCheckpointRecord` | “Safe Boundary Evidence” |
| `Checkpoint Record` | `event_heads` | Sorted unique digest array with 1..64 entries. | `validateCheckpointRecord` | “sorted unique digest[1..64]” |
| `Checkpoint Record` | `workspace_manifest_id` | Required digest. | `validateCheckpointRecord` | “Workspace-group Transfer Manifest root” |
| `Checkpoint Record` | `provider_manifest_id` | Nullable digest; exactly one persistence reference is non-null. | `validateCheckpointRecord` | “Direct native-store/provider snapshot only” |
| `Checkpoint Record` | `task_board_bundle_id` | Nullable digest; exactly one persistence reference is non-null. | `validateCheckpointRecord` | “Task-board path only” |
| `Checkpoint Record` | `created_by_host_id` | UUIDv7 current holder. | `validateCheckpointRecord` | “Current lease holder” |
| `Checkpoint Record` | `created_at` | Timestamp, diagnostic only. | `validateCheckpointRecord` | “Diagnostic only” |
| `Checkpoint Record` | `status` | Exact validated literal. | `validateCheckpointRecord` | “Literal” |
| `Checkpoint Record` | `extensions` | Required reverse-DNS extension object. | `validateCheckpointRecord` | “Reverse-DNS extension keys only” |
| `Safe Boundary Evidence` | `provider_id` | Provider ID grammar. | `validateSafeBoundaryEvidence` | “provider-id” |
| `Safe Boundary Evidence` | `provider_version` | String of 1..128 characters. | `validateSafeBoundaryEvidence` | “string[1..128]” |
| `Safe Boundary Evidence` | `evidence` | Closed five-member evidence enum. | `validateSafeBoundaryEvidence` | “accepted_test” |
| `Safe Boundary Evidence` | `input_blocked` | Boolean required true for publication. | `validateSafeBoundaryEvidence` | “input_blocked:boolean” |
| `Safe Boundary Evidence` | `foreground_idle` | Boolean required true for publication. | `validateSafeBoundaryEvidence` | “foreground_idle:boolean” |
| `Safe Boundary Evidence` | `background_idle` | Boolean required true for publication. | `validateSafeBoundaryEvidence` | “background_idle:boolean” |
| `Safe Boundary Evidence` | `open_processes` | uint53 required zero for publication. | `validateSafeBoundaryEvidence` | “open_processes:uint53” |
| `Safe Boundary Evidence` | `open_database_handles` | uint53 required zero for publication. | `validateSafeBoundaryEvidence` | “open_database_handles:uint53” |
| `Provider Identity Record` | `schema` | Exact Provider Identity schema identifier. | `validateProviderIdentityRecord` | “Exact Provider Identity schema identifier” |
| `Provider Identity Record` | `schema_version` | Exact version 1.0.0. | `validateProviderIdentityRecord` | “1.0.0” |
| `Provider Identity Record` | `record_id` | Canonical digest self identity. | `validateProviderIdentityRecord` | “Canonical object digest” |
| `Provider Identity Record` | `subject_id` | UUIDv7 equal to session ID. | `validateProviderIdentityRecord` | “Equal to” |
| `Provider Identity Record` | `session_id` | UUIDv7 logical session. | `validateProviderIdentityRecord` | “Existing logical session” |
| `Provider Identity Record` | `provider_id` | Provider ID grammar. | `validateProviderIdentityRecord` | “provider-id” |
| `Provider Identity Record` | `provider_version` | String of 1..128 characters. | `validateProviderIdentityRecord` | “string[1..128]” |
| `Provider Identity Record` | `provider_version_range` | String of 1..256 characters. | `validateProviderIdentityRecord` | “string[1..256]” |
| `Provider Identity Record` | `native_session_id` | String of 1..512 characters. | `validateProviderIdentityRecord` | “string[1..512]” |
| `Provider Identity Record` | `identity_kind` | Closed five-member identity enum. | `validateProviderIdentityRecord` | “enum” |
| `Provider Identity Record` | `logical_workspace_id` | UUIDv7 workspace reference. | `validateProviderIdentityRecord` | “UUIDv7” |
| `Provider Identity Record` | `backend_realm_fingerprint` | Nullable digest; Antigravity backend conversation requires non-null. | `validateProviderIdentityRecord` | “digest or null” |
| `Provider Identity Record` | `opaque_identity` | Closed provider-data map of 0..32 bounded string values. | `validateProviderIdentityRecord` | “map(provider-identity-key,string[1..1024])[0..32]” |
| `Provider Identity Record` | `created_by_host_id` | UUIDv7 identifying host. | `validateProviderIdentityRecord` | “Identifying owner host” |
| `Provider Identity Record` | `created_at` | Timestamp, diagnostic only. | `validateProviderIdentityRecord` | “Diagnostic only” |
| `Provider Identity Record` | `extensions` | Required reverse-DNS extension object. | `validateProviderIdentityRecord` | “Reverse-DNS extension keys only” |
| `Workspace Group Record` | `schema` | Exact Workspace Group schema identifier. | `validateWorkspaceGroupRecord` | “urn:ax:schema:workspace-group” |
| `Workspace Group Record` | `schema_version` | Exact version 1.0.0. | `validateWorkspaceGroupRecord` | “1.0.0” |
| `Workspace Group Record` | `record_id` | Canonical digest self identity. | `validateWorkspaceGroupRecord` | “record_id:digest” |
| `Workspace Group Record` | `subject_id` | UUIDv7 equal to workspace group ID. | `validateWorkspaceGroupRecord` | “subject_id:UUIDv7” |
| `Workspace Group Record` | `workspace_group_id` | UUIDv7 equal to subject ID. | `validateWorkspaceGroupRecord` | “workspace_group_id:UUIDv7” |
| `Workspace Group Record` | `display_name` | String of 1..128 characters. | `validateWorkspaceGroupRecord` | “display_name:string[1..128]” |
| `Workspace Group Record` | `members` | Closed WorkspaceMember array of 1..256 entries sorted by ID. | `validateWorkspaceGroupRecord` | “members:WorkspaceMember[1..256]” |
| `Workspace Group Record` | `created_by_host_id` | UUIDv7 creator. | `validateWorkspaceGroupRecord` | “created_by_host_id:UUIDv7” |
| `Workspace Group Record` | `created_at` | Timestamp. | `validateWorkspaceGroupRecord` | “created_at:timestamp” |
| `Workspace Group Record` | `extensions` | Required reverse-DNS extension object. | `validateWorkspaceGroupRecord` | “extensions:object” |
| `Session Event` | `schema` | Exact Session Event schema identifier. | `validateSessionEvent` | “urn:ax:schema:session-event” |
| `Session Event` | `schema_version` | Exact selected version 1.0.0 through 4.0.0. | `validateSessionEvent` | “schema_version” |
| `Session Event` | `event_id` | Canonical digest self identity. | `validateSessionEvent` | “event_id” |
| `Session Event` | `subject_id` | UUIDv7 equal to session ID. | `validateSessionEvent` | “subject_id” |
| `Session Event` | `session_id` | UUIDv7 equal to subject ID. | `validateSessionEvent` | “session_id” |
| `Session Event` | `event_type` | Version-selected catalog event name; v1 unknown types remain inert and retainable. | `validateSessionEvent` | “event_type” |
| `Session Event` | `created_by_host_id` | UUIDv7 author host. | `validateSessionEvent` | “created_by_host_id” |
| `Session Event` | `lease_epoch` | Positive uint53. | `validateSessionEvent` | “lease_epoch” |
| `Session Event` | `lease_id` | UUIDv4 winning lease token. | `validateSessionEvent` | “lease_id” |
| `Session Event` | `lease_sequence` | Positive uint53 starting at one. | `validateSessionEvent` | “uint53 starting at 1” |
| `Session Event` | `predecessors` | Non-empty sorted digest array. | `validateSessionEvent` | “sorted array of one or more record/event digests” |
| `Session Event` | `created_at` | Timestamp. | `validateSessionEvent` | “created_at” |
| `Session Event` | `payload` | Closed version-selected tagged union for registered types. | `validateSessionEvent` | “closed tagged union” |
| `Session Event` | `extensions` | Required reverse-DNS extension object. | `validateSessionEvent` | “extensions” |
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
| `Session Record 1.0.0` | `name` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | “string; Section 2.1 grammar [A-Za-z0-9][A-Za-z0-9._-]{0,63} and 1–64 characters” |
| `Session Record 1.0.0` | `provider_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | “string; lowercase plugin ID” |
| `Session Record 1.0.0` | `record_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | “digest; canonical object digest” |
| `Session Record 1.0.0` | `schema` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | “string; exact schema identifier” |
| `Session Record 1.0.0` | `schema_version` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | “semver; exact 1.0.0” |
| `Session Record 1.0.0` | `session_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | “UUIDv7; globally unique” |
| `Session Record 1.0.0` | `subject_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | “UUIDv7; equal to session_id” |
| `Session Record 1.0.0` | `task_board` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | “Task-board Reference or null; object exactly when kind is task_board” |
| `Session Record 1.0.0` | `workspace_group_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | “UUIDv7; required” |
| `Session Record 2.0.0 and 3.0.0` | `created_at` | Enforced through the common immutable Record Envelope before identity calculation or verification. | `validateSessionRecordWithDerivation` | “timestamp; diagnostic time” |
| `Session Record 2.0.0 and 3.0.0` | `created_by_host_id` | Enforced through the common immutable Record Envelope before identity calculation or verification. | `validateSessionRecordWithDerivation` | “UUIDv7; allowlisted host at creation” |
| `Session Record 2.0.0 and 3.0.0` | `derivation_provenance` | Required closed provenance union; v2 admits three tags and v3 admits those three plus native adoption. | `validateSessionRecordWithDerivation` | “required closed derivation provenance; v3 closed creation union” |
| `Session Record 2.0.0 and 3.0.0` | `execution_profile` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordWithDerivation` | “enum standard or yolo” |
| `Session Record 2.0.0 and 3.0.0` | `extensions` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordWithDerivation` | “object; required, reverse-DNS keys only” |
| `Session Record 2.0.0 and 3.0.0` | `kind` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordWithDerivation` | “enum direct or task_board” |
| `Session Record 2.0.0 and 3.0.0` | `launch_plan` | Enforced by the shared immutable Session Record creation shape. | `validateSessionRecordWithDerivation` | “Launch Plan; closed, sanitized and secret-free” |
| `Session Record 2.0.0 and 3.0.0` | `name` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordWithDerivation` | “string; Section 2.1 grammar [A-Za-z0-9][A-Za-z0-9._-]{0,63} and 1–64 characters” |
| `Session Record 2.0.0 and 3.0.0` | `provider_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordWithDerivation` | “provider ID allocated at creation and immutable” |
| `Session Record 2.0.0 and 3.0.0` | `record_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordWithDerivation` | “digest; canonical object digest” |
| `Session Record 2.0.0 and 3.0.0` | `schema` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordWithDerivation` | “string; exact Session Record schema identifier” |
| `Session Record 2.0.0 and 3.0.0` | `schema_version` | Enforced as exact 2.0.0 or exact 3.0.0 at the version-selected validator. | `validateSessionRecordWithDerivation` | “independently closed 2.0.0 and 3.0.0 variants” |
| `Session Record 2.0.0 and 3.0.0` | `session_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordWithDerivation` | “UUIDv7; newly allocated and globally unique” |
| `Session Record 2.0.0 and 3.0.0` | `subject_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordWithDerivation` | “UUIDv7; equal to session_id” |
| `Session Record 2.0.0 and 3.0.0` | `task_board` | Enforced by the shared immutable Session Record creation shape. | `validateSessionRecordWithDerivation` | “Task-board Reference or null; orthogonal authority” |
| `Session Record 2.0.0 and 3.0.0` | `workspace_group_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordWithDerivation` | “UUIDv7; required” |
| `Session Record origin provenance` | `creation_operation_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionOriginProvenance` | “UUIDv7” |
| `Session Record origin provenance` | `extensions` | Enforced exactly as declared before identity calculation or verification. | `validateSessionOriginProvenance` | “reverse-DNS extensions” |
| `Session Record origin provenance` | `kind` | Enforced exactly as declared before identity calculation or verification; the per-variant exact-string check is defensively redundant with the `validateSessionDerivationProvenance` switch dispatch that exclusively selects this validator. | `validateSessionOriginProvenance` | “exact origin” |
| `Session Record same-provider-fork provenance` | `extensions` | Enforced exactly as declared before identity calculation or verification. | `validateSessionSameProviderForkProvenance` | “reverse-DNS extensions” |
| `Session Record same-provider-fork provenance` | `kind` | Enforced exactly as declared before identity calculation or verification; the per-variant exact-string check is defensively redundant with the `validateSessionDerivationProvenance` switch dispatch that exclusively selects this validator. | `validateSessionSameProviderForkProvenance` | “exact same_provider_fork” |
| `Session Record same-provider-fork provenance` | `operation_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionSameProviderForkProvenance` | “UUIDv7” |
| `Session Record same-provider-fork provenance` | `provider_fork_mode` | Enforced exactly as declared before identity calculation or verification. | `validateSessionSameProviderForkProvenance` | “native, supported_import, or task_board_clone” |
| `Session Record same-provider-fork provenance` | `source_checkpoint_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionSameProviderForkProvenance` | “digest” |
| `Session Record same-provider-fork provenance` | `source_profile_event_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionSameProviderForkProvenance` | “digest or null” |
| `Session Record same-provider-fork provenance` | `source_session_id` | Enforced as UUIDv7 distinct from the new target Session ID. | `validateSessionSameProviderForkProvenance` | “source_session_id UUIDv7; fork creates a new logical session” |
| `Session Record same-provider-fork provenance` | `source_workspace_group_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionSameProviderForkProvenance` | “UUIDv7” |
| `Session Record cross-environment-clone provenance` | `bundle_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionCrossEnvironmentCloneProvenance` | “UUIDv7” |
| `Session Record cross-environment-clone provenance` | `canonical_session_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionCrossEnvironmentCloneProvenance` | “digest” |
| `Session Record cross-environment-clone provenance` | `capture_manifest_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionCrossEnvironmentCloneProvenance` | “digest” |
| `Session Record cross-environment-clone provenance` | `extensions` | Enforced exactly as declared before identity calculation or verification. | `validateSessionCrossEnvironmentCloneProvenance` | “reverse-DNS extensions” |
| `Session Record cross-environment-clone provenance` | `kind` | Enforced exactly as declared before identity calculation or verification; the per-variant exact-string check is defensively redundant with the `validateSessionDerivationProvenance` switch dispatch that exclusively selects this validator. | `validateSessionCrossEnvironmentCloneProvenance` | “exact cross_environment_clone” |
| `Session Record cross-environment-clone provenance` | `migration_checkpoint_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionCrossEnvironmentCloneProvenance` | “digest” |
| `Session Record cross-environment-clone provenance` | `operation_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionCrossEnvironmentCloneProvenance` | “UUIDv7” |
| `Session Record cross-environment-clone provenance` | `previous_lineage_receipt_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionCrossEnvironmentCloneProvenance` | “digest or null” |
| `Session Record cross-environment-clone provenance` | `projection_plan_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionCrossEnvironmentCloneProvenance` | “digest” |
| `Session Record cross-environment-clone provenance` | `source_checkpoint_id` | Enforced as digest or null with the four-way AX-source nullability rule. | `validateSessionCrossEnvironmentCloneProvenance` | “digest or null; non-null exactly for ax_session” |
| `Session Record cross-environment-clone provenance` | `source_environment` | Enforced as an exact closed EnvironmentTuple. | `validateSessionCrossEnvironmentCloneProvenance` | “EnvironmentTuple” |
| `Session Record cross-environment-clone provenance` | `source_kind` | Enforced exactly as declared before identity calculation or verification. | `validateSessionCrossEnvironmentCloneProvenance` | “ax_session or external_native” |
| `Session Record cross-environment-clone provenance` | `source_native_session_id` | Enforced as 1–512 printable non-control Unicode characters, matching the pinned sanitization requirement; accept-at and refuse-past boundaries drive both public entries. The minimum-one check is subsumed by `requireString`, which refuses the empty string before `requirePrintableBoundedString` counts characters. | `validateSessionCrossEnvironmentCloneProvenance` | “sanitized non-authoritative source native session ID; string[1..512]” |
| `Session Record cross-environment-clone provenance` | `source_profile_event_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionCrossEnvironmentCloneProvenance` | “digest or null” |
| `Session Record cross-environment-clone provenance` | `source_provider_identity_record_id` | Enforced as digest or null with the four-way AX-source nullability rule. | `validateSessionCrossEnvironmentCloneProvenance` | “digest or null; non-null exactly for ax_session” |
| `Session Record cross-environment-clone provenance` | `source_session_id` | Enforced as UUIDv7 or null, distinct from target, with the four-way AX-source nullability rule. | `validateSessionCrossEnvironmentCloneProvenance` | “UUIDv7 or null; non-null exactly for ax_session” |
| `Session Record cross-environment-clone provenance` | `source_session_record_id` | Enforced as digest or null with the four-way AX-source nullability rule. | `validateSessionCrossEnvironmentCloneProvenance` | “digest or null; non-null exactly for ax_session” |
| `Session Record cross-environment-clone provenance` | `source_snapshot_digest` | Enforced exactly as declared before identity calculation or verification. | `validateSessionCrossEnvironmentCloneProvenance` | “digest” |
| `Session Record cross-environment-clone provenance` | `target_environment` | Enforced as an exact closed EnvironmentTuple. | `validateSessionCrossEnvironmentCloneProvenance` | “EnvironmentTuple” |
| `Session Record native-adoption provenance` | `extensions` | Enforced exactly as declared before identity calculation or verification. | `validateSessionNativeAdoptionProvenance` | “reverse-DNS extensions” |
| `Session Record native-adoption provenance` | `kind` | Enforced exactly as declared before identity calculation or verification; the per-variant exact-string check is defensively redundant with the `validateSessionDerivationProvenance` switch dispatch that exclusively selects this validator. | `validateSessionNativeAdoptionProvenance` | “exact native_adoption” |
| `Session Record native-adoption provenance` | `operation_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionNativeAdoptionProvenance` | “UUIDv7” |
| `Session Record native-adoption provenance` | `source_environment` | Enforced as an exact closed EnvironmentTuple. | `validateSessionNativeAdoptionProvenance` | “EnvironmentTuple” |
| `Session Record native-adoption provenance` | `source_head_digest` | Enforced exactly as declared before identity calculation or verification. | `validateSessionNativeAdoptionProvenance` | “digest” |
| `Session Record native-adoption provenance` | `source_host_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionNativeAdoptionProvenance` | “UUIDv7” |
| `Session Record native-adoption provenance` | `source_instance_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionNativeAdoptionProvenance` | “digest” |
| `Session Record native-adoption provenance` | `source_observation_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionNativeAdoptionProvenance` | “digest” |
| `Session Record native-adoption provenance` | `target_provider_id` | Enforced equal to the immutable target Session Record provider. Provider-ID grammar is subsumed by `validateSessionRecordCommon`, which rejects a malformed record `provider_id` before this equality gate; a different malformed target is refused by the equality gate. | `validateSessionNativeAdoptionProvenance` | “provider-id; target provider allocated at creation” |
| `EnvironmentTuple` | `adapter_version` | Enforced as canonical SemVer. The Probe requires its adapter version to equal the verified Manifest value, whose declared type is SemVer. | `validateEnvironmentTuple` | “display_name / adapter_version — UTF-8 string[1..128] / SemVer”; “adapter version equal the verified Manifest and host values” |
| `EnvironmentTuple` | `architecture` | Enforced exactly as declared before identity calculation or verification. | `validateEnvironmentTuple` | “amd64 or arm64” |
| `EnvironmentTuple` | `environment_id` | Enforced against the exact environment ID grammar. | `validateEnvironmentTuple` | “[a-z][a-z0-9.-]{0,63}” |
| `EnvironmentTuple` | `environment_version` | Presence-only. The pinned EnvironmentTuple declaration supplies no JSON type or bound; the `string[1..128]` bound belongs to the distinct Environment Observation schema and is not inferred here. | `validateEnvironmentTuple` | “environment_version” |
| `EnvironmentTuple` | `platform` | Enforced against the complete generated AX platform scalar vocabulary. | `validateEnvironmentTuple` | “linux, macos, windows, or wsl2” |
| `EnvironmentTuple` | `store_schema_fingerprint` | Presence-only. The pinned EnvironmentTuple declaration supplies no JSON type or format; in particular, identity validation does not infer a digest from the member name. | `validateEnvironmentTuple` | “store_schema_fingerprint” |
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
| `WorkspaceMember.git` | `workspace_id` | UUIDv7. | `validateWorkspaceMember` | “workspace_id:UUIDv7” |
| `WorkspaceMember.managed_tree` | `workspace_id` | UUIDv7. | `validateWorkspaceMember` | “workspace_id:UUIDv7” |
| `WorkspaceMember.git` | `group_relative_path` | Relative path; absolute, parent-escaping, and non-canonical forms are refused. | `validateWorkspaceMember` | “group_relative_path:path” |
| `WorkspaceMember.managed_tree` | `group_relative_path` | Relative path; absolute, parent-escaping, and non-canonical forms are refused. | `validateWorkspaceMember` | “group_relative_path:path” |
| `WorkspaceMember.git` | `repo_relative_cwd` | Literal `.` or a relative path. | `validateWorkspaceMember` | “repo_relative_cwd:.&#124;path” |
| `WorkspaceMember.managed_tree` | `repo_relative_cwd` | Literal `.` or a relative path. | `validateWorkspaceMember` | “repo_relative_cwd:.&#124;path” |
| `WorkspaceMember.git` | `agent_project_config_paths` | Sorted unique relative paths, 0..256 entries. | `validateWorkspaceMember` | “agent_project_config_paths:sorted unique path[0..256]” |
| `WorkspaceMember.managed_tree` | `agent_project_config_paths` | Sorted unique relative paths, 0..256 entries. | `validateWorkspaceMember` | “agent_project_config_paths:sorted unique path[0..256]” |
| `WorkspaceMember.git` | `kind` | Exact tag selecting the git member set. | `validateWorkspaceMember` | “kind:git” |
| `WorkspaceMember.git` | `repository_identity` | 1..256 Unicode characters and refused when it is an absolute path. | `validateWorkspaceMember` | “repository_identity:string[1..256]” |
| `WorkspaceMember.git` | `sanitized_remote_urls` | Sorted unique sanitized Git URLs, 1..16 entries; password, token, query, fragment, and machine-local file forms are refused. | `validateWorkspaceMember` | “sanitized_remote_urls:sorted unique sanitized-git-URL[1..16]” |
| `WorkspaceMember.git` | `materialization_policy` | Enum shared_checkout or separate_worktree. | `validateWorkspaceMember` | “materialization_policy:shared_checkout&#124;separate_worktree” |
| `WorkspaceMember.managed_tree` | `kind` | Exact tag selecting the managed_tree member set. | `validateWorkspaceMember` | “kind:managed_tree” |
| `WorkspaceMember.managed_tree` | `tree_identity` | 1..256 Unicode characters and refused when it is an absolute path. | `validateWorkspaceMember` | “tree_identity:string[1..256]” |
| `WorkspaceMember.managed_tree` | `materialization_policy` | Enum shared_tree or separate_copy. | `validateWorkspaceMember` | “materialization_policy:shared_tree&#124;separate_copy” |
| `Observation Event` | `schema` | Exact Observation schema identifier. | `validateObservationEvent` | “string &#124; Exact Observation schema identifier” |
| `Observation Event` | `schema_version` | Exact version 1.0.0. | `validateObservationEvent` | “semver &#124; Exact 1.0.0” |
| `Observation Event` | `stream_id` | UUIDv7. | `validateObservationEvent` | “UUIDv7 &#124; Stable per host installation; changing it starts a new explicitly separate stream” |
| `Observation Event` | `sequence` | uint53 greater than zero. | `validateObservationEvent` | “uint53 &#124; Starts at 1 and increases by exactly one before each durable append” |
| `Observation Event` | `timestamp` | Canonical AX timestamp. | `validateObservationEvent` | “timestamp &#124; Observation time; not authority” |
| `Observation Event` | `level` | Enum debug, info, warn, or error. | `validateObservationEvent` | “enum &#124; debug, info, warn, or error” |
| `Observation Event` | `event` | 3..128 Unicode characters matching the declared observation-name grammar. | `validateObservationEvent` | “observation-name &#124; [a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*){1,7}, 3–128 characters” |
| `Observation Event` | `operation_id` | UUIDv7 or null. | `validateObservationEvent` | “UUIDv7 or null &#124; Required null when no operation exists” |
| `Observation Event` | `session_id` | UUIDv7 or null. | `validateObservationEvent` | “UUIDv7 or null &#124; Required null for non-session events” |
| `Observation Event` | `host_id` | UUIDv7. | `validateObservationEvent` | “UUIDv7 &#124; Emitting host” |
| `Observation Event` | `peer_host_id` | UUIDv7 or null. | `validateObservationEvent` | “UUIDv7 or null &#124; Required null when no peer participates” |
| `Observation Event` | `phase` | 1..128 Unicode characters in lower_snake_case, or null. | `validateObservationEvent` | “string[1..128] or null &#124; Stable lower-snake-case phase or null” |
| `Observation Event` | `result` | Enum started, success, partial, failure, or cancelled. | `validateObservationEvent` | “enum &#124; started, success, partial, failure, or cancelled” |
| `Observation Event` | `duration_ms` | uint53 or null; a started result requires null. | `validateObservationEvent` | “uint53 or null &#124; Null for a point/start event; otherwise elapsed milliseconds” |
| `Observation Event` | `counts` | Closed ObservationCounts object or null. | `validateObservationEvent` | “ObservationCounts or null &#124; Closed aggregate below” |
| `Observation Event` | `object_ids` | Sorted unique digests, 0..4096 entries. | `validateObservationEvent` | “sorted unique digest[0..4096] &#124; Redacted object identities only” |
| `Observation Event` | `error_code` | 1..128 Unicode characters or null; non-null exactly for partial and failure. | `validateObservationEvent` | “string[1..128] or null &#124; Stable Section 15 code when result is partial/failure” |
| `Observation Event` | `extensions` | Present object member. | `validateObservationEvent` | “object &#124; Reverse-DNS extension keys only; no payload content” |
| `ObservationCounts` | `records` | uint53 within the safe-integer bound. | `validateObservationCountsMember` | “records:uint53” |
| `ObservationCounts` | `events` | uint53 within the safe-integer bound. | `validateObservationCountsMember` | “events:uint53” |
| `ObservationCounts` | `manifests` | uint53 within the safe-integer bound. | `validateObservationCountsMember` | “manifests:uint53” |
| `ObservationCounts` | `blobs` | uint53 within the safe-integer bound. | `validateObservationCountsMember` | “blobs:uint53” |
| `ObservationCounts` | `chunks` | uint53 within the safe-integer bound. | `validateObservationCountsMember` | “chunks:uint53” |
| `ObservationCounts` | `bytes` | uint53 within the safe-integer bound. | `validateObservationCountsMember` | “bytes:uint53” |
| `ObservationCounts` | `retries` | uint53 within the safe-integer bound. | `validateObservationCountsMember` | “retries:uint53” |

## Reachable cross-member, recursive, and external rules

| Scope | Pinned rule | Disposition and production call site |
| --- | --- | --- |
| Session Record | `subject_id` equals `session_id`. | Enforced by `validateSessionRecordV1`. |
| Session Record | `task_board` is non-null exactly for `kind=task_board`. | Enforced by `validateSessionTaskBoardReference`; direct and task-board positive variants plus both refusal directions drive both public entries. |
| Session Record | `fork_provenance` is null unless the record was created by fork. | The nullable closed shape is enforced by `validateSessionForkProvenance`; whether an external creation operation actually was a fork requires that operation evidence and is outside one identity candidate. |
| Session Record 2 and 3 | `derivation_provenance` is a required closed union, with native adoption admitted only in major 3. | Enforced by `validateSessionDerivationProvenance`; positive fixtures cover every tag through both identity entries and cross-tag member leakage is refused. The major-3 union is the specification's creation union but retains the v2 wire member name. |
| Same-provider fork provenance | A fork creates a new logical Session rather than reusing the source Session ID. | Enforced by `validateSessionSameProviderForkProvenance`; the v1 equivalent is enforced by `validateSessionForkProvenance`. Both reuse attempts are production-entry refusal cases. |
| Cross-environment clone provenance | All four AX-source IDs are non-null exactly for `ax_session`, null exactly for `external_native`, and a source AX Session ID never equals the target. Source and target Environment Tuples are closed and validated independently against every constraint the pinned declaration states; the Session Record contract states no tuple- or environment-ID inequality rule. | Enforced by `validateSessionCrossEnvironmentCloneProvenance`; both source-kind arms, both nullability refusal directions, every derived AX-source member's UUID/digest format, and Session ID reuse drive both public entries. Equal environment IDs and identical tuples are positive compatibility cases. |
| Native-adoption provenance | The creation provenance target provider equals the immutable Session Record provider; later Provider Identity, Checkpoint, event, receipt, lease, or runtime facts are not members. | Equality is enforced by `validateSessionNativeAdoptionProvenance`; exact-member validation refuses final-fact or cross-tag leakage. `source_observation_id` is a Directory observation digest and does not claim Section 18.1 Observation Event runtime support. |
| Section 2.1 identity contribution | A Session name is 1–64 characters matching `[A-Za-z0-9][A-Za-z0-9._-]{0,63}`. | `validateSessionRecordCommon` enforces the exact ASCII grammar for every supported Session Record major at identity calculation and verification. Mesh-wide uniqueness requires the derived index and peer state and remains explicitly unclaimed by this identity package. |
| Section 2.2 identity contribution | The immutable Session Record persists `execution_profile`, and initial launch and every resume must use it without silently downgrading `yolo`. | `validateSessionRecordCommon` enforces the closed `standard|yolo` value at identity calculation and verification. Lease, replica, event-fencing, replication, materialization, tombstone, capability, bridge, sync, and takeover runtime invariants remain explicitly unclaimed by this identity package. |
| Section 2.3 identity contribution | Name resolution consumes a Session Record name, but resolution order and action selection require derived runtime state. | `validateSessionRecordCommon` supplies only a syntactically valid immutable name. Local/peer lookup order, exact UUID fallback, ASCII-fold collision detection, `name_ambiguous`, safe auto-attach/resume, pure planning, and `interactive_choice_required` remain explicitly unclaimed by this identity package. |
| Session Event | Every generated catalog v1–v4 event selects one exact closed payload; v1 unknown types remain inert, while later majors refuse unknown types. | Enforced by `validateSessionEvent` and the catalog-derived `sessionEventPayloadShapes`; executable tests enumerate every catalog version/type, every required payload member, wrong-type and unknown-member refusals, closed-vocabulary complements, literal opposites, and cross-major leakage through both public identity entries. |
| Session Event | Lease-sequence continuity, immediate authoritative predecessors, winning-lease membership, referenced checkpoint/profile equality, and operation/event causal order. | External: those facts require the event DAG, lease union, Session Record, Checkpoints, and operation receipts, none of which are contained in one immutable event candidate. |
| Lease Record | An epoch-one `create` lease has a null predecessor; any lease after epoch one has a non-null predecessor; every lease other than an epoch-one `create` has a non-null Checkpoint; creator equals issuer. No reason is inferred from the epoch in either direction, because Section 5.3 declares no such clause. | Enforced by `validateLeaseRecord`. Equality to the known predecessor session, exact predecessor epoch +1, max-observed epoch, and winning tuple require the converged record set and remain external. |
| Checkpoint Record | Safe Boundary booleans are true, counters are zero, status is `validated`, and exactly one persistence reference is non-null. | Enforced by `validateCheckpointRecord` and `validateSafeBoundaryEvidence`. Matching the selected Session Record kind, winning lease, authoritative event heads, referenced objects, and actual provider quiescence requires external records/runtime evidence and remains a publication gate. |
| Provider Identity Record | Subject equals Session ID, provider identity data has the declared closed grammar/bounds, an `opaque_identity` value that BEGINS with a POSIX or Windows absolute path is refused, and Antigravity backend-conversation identities require a realm fingerprint. | Enforced by `validateProviderIdentityRecord`. The pinned clause is “It MUST NOT contain an absolute path, credential, environment value, PID, socket, terminal ID, or mutable cache selector”; only the leading-absolute-path form of it is decidable from one candidate, so a value such as `cwd=/Users/x` embeds an absolute path and is admitted. That remainder joins the credential, environment-value, PID, socket, terminal-ID and mutable-selector classes of the same sentence as external: each needs provider/session context to classify. Equality to the referenced Session provider/workspace is external for the same reason. |
| Workspace Group Record | Subject equals group ID; members are non-empty, workspace-ID ordered, closed by tag, path-safe, and have no equal/case-colliding group paths. | Enforced by `validateWorkspaceGroupRecord` and `validateWorkspaceMember`, including absolute logical-identity refusal. Conflicting records for one group, membership convergence, snapshot existence, and whole-cohort runtime migration require the converged record/filesystem/runtime set and remain external. |
| Launch Plan | `argv` is non-empty, each element is 1–4096 UTF-8 bytes, and encoded argv is at most 65,536 bytes. | Enforced by `validateSessionLaunchPlan`. The broader semantic prohibition on secret-bearing or shell-fragment arguments requires provider/secret classification and is external to syntax-only identity validation. |
| Launch Plan | `env_names` is sorted and unique; `env_literals` keys use the same grammar and are disjoint. | Enforced by `validateSessionLaunchPlan`; `TestSpecDerivedRefusalClausesReachBothIdentityEntries` pins the literal-key grammar through both public entries. Whether a literal is semantically secret requires secret classification and remains external. |
| Task-board Reference | Creation-time `manager_session_ref` is null; `primary_owner` requires a non-null goal and `bound`; `tracked_prompt` permits only `prompt` or `none`. | Enforced by `validateSessionTaskBoardReference`. |
| Board Identity | Local requires null URL; remote requires absolute HTTPS with host and no userinfo, query, or fragment. | Enforced by `validateSessionBoardIdentity`; `TestSpecDerivedRefusalClausesReachBothIdentityEntries` pins the local-null arm through both public entries. |
| Blob Descriptor | Empty size has no chunks; non-empty has at least one; indexes start at zero, offsets are contiguous, non-final chunks are fixed 4 MiB, and chunks cover size exactly without overflow. | Enforced by `validateBlobDescriptor`. |
| Blob Descriptor | Descriptor size/digest and chunk digests match referenced raw bytes. | External: raw bytes are not members of the descriptor identity candidate. |
| Transfer Manifest | The five `kind` arms impose exact nullability plus empty/non-empty entry and child rules. | Enforced by `validateTransferManifest`. |
| Transfer Manifest | Entries are strictly bytewise path-sorted and have no destination-case collisions. | Enforced by `validateManifestEntries`; collision membership is O(total path characters) through `simpleFoldKey`, and the 65,536-entry production calculation guard is below 2 seconds. |
| Transfer Manifest | “Entries and child partitions MUST contain no duplicate, overlapping, or destination-case-colliding path.” All three properties are entry-local, and the whole sentence is the row. | Entry-local duplicate and destination-case collision are enforced by `simpleFoldKey` membership. Entry-local OVERLAP is enforced by `validateManifestEntries`: an entry whose ancestor is a declared `file`, `symlink` or `hardlink` is refused, reusing the strict bytewise order the same loop already proves rather than re-deriving each path's ancestor set. `TestManifestEntryOverlapRefusalReachesBothIdentityEntries` pins file-over-file, symlink-over-file, file-over-directory, hardlink-over-file, an intervening `.`-sorted sibling and a nested owner through both public entries, and `TestManifestEntryOverlapAdmitsDirectoryParents` keeps declared directory parents and shared byte prefixes admitted. Overlap is compared bytewise, not case-folded, so a case-only overlap such as `A` over `a/b` is refused only where the duplicate/case-collision clause already reaches it. Child partition paths require the referenced child manifests and remain external to one candidate. |
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
