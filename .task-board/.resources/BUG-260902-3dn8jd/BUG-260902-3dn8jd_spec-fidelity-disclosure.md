# BUG-260902-3dn8jd — Every enumeration row measured against the pinned SPEC.md

Pinned document: `relux-works/agent-session-manager-spec` v0.5.0, commit
`28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`, `SPEC.md` SHA-256
`562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a`, verified
byte-for-byte against `internal/specpin.DocumentSHA256` before any comparison.

Nothing here was silently repaired. Every row below failed the comparison the
gate now performs, and every one is listed with its pre-fix cell and its
replacement.

## Result over the pre-fix artifact

| Class | Rows | Meaning |
| --- | ---: | --- |
| Quote absent from the pinned document | 149 | The cell attributed text to `SPEC.md` that `SPEC.md` does not contain, under whitespace-only normalization. |
| Quote present but unanchored | 161 | The text does occur, but at more than one place, so the cell identified no particular declaration. |
| Quote present at exactly one line | 37 | The only class the old contract could have gotten right by accident. |
| **Total rows** | **347** | |

All 347 rows were rewritten, because the new contract additionally requires a
declared `SPEC.md` line and an entry that names the member. 149 of them were
false attributions; the rest were unfalsifiable.

## What the old gate actually checked

`internal/canonicaljson/constraint_inventory_test.go` required only
`row.specExcerpt != ""`. The shape, member and call-site columns were derived
from the production `requireExactMembers` argument lists and compared both ways,
so the artifact resisted drift against the code — and proved nothing at all
about the specification. A row could say anything in the `Pinned SPEC
declaration` column and stay green.

## Findings that outlived the rewrite

These are not formatting defects. Each is a substantive disagreement between the
artifact and the pinned document, disclosed rather than accommodated.

### 1. `EnvironmentTuple.environment_id` grammar is a cross-schema inference

`validateEnvironmentTuple` enforces `[a-z][a-z0-9.-]{0,63}` on
`environment_id` (`internal/canonicaljson/closed_shapes.go:733`). The pinned
EnvironmentTuple clause (`SPEC.md` L3626-L3630) names the member and gives it no
type and no format, exactly like its `environment_version`,
`store_schema_fingerprint` and `adapter_version` siblings, for which this
package already refuses to infer a constraint. The only pinned statement of that
grammar is L3609, a row of the **Session Adapter Manifest** — a different
schema. The artifact previously quoted the bare grammar with no indication of
where it came from.

The row now quotes both lines and says plainly that the cross-schema step is an
open question. Production behaviour was left unchanged: deciding whether
`environment-id` is a common type inferable across schemas, or whether the tuple
member is presence-only like its siblings, is a specification/product question,
not a test-fidelity one. **Recommend a follow-up Bug.**

### 2. The Session Record `name` row cited the wrong section

The pre-fix cell read "Section 2.1 grammar `[A-Za-z0-9][A-Za-z0-9._-]{0,63}` and
1-64 characters", which appears nowhere in the pinned document. The Session
Record table row (L1471) says "Section 2.3 grammar"; the grammar itself is
written in the §2.1 Terms table (L363) and §2.3 contains no grammar. Both rows
now quote L1471 and L363, so the document's own indirection is visible instead of
being flattened into a claim it never makes.

`sessionRecordGrammarFamily` in `session_record_versions_test.go` classified that
row by matching the string "section 2.1 grammar". It now matches the quoted
grammar itself, which is a stricter key than a section reference — the same
production entries are still driven for the same six grammar families.

### 3. Adjacent gap left in scope for a follow-up

`internal/canonicaljson/testdata/closed-vocabularies.md` carries the same class
of column (`SPEC line` + `Pinned SPEC declaration`) and is still not compared to
the pinned document; its declarations paraphrase the pinned rows rather than
quoting them, and at least one row (`validateSessionForkProvenance`
`provider_fork_mode`) cites L1653, the same-provider-fork variant's line, rather
than the Fork Provenance prose at L1536-L1537 that its validator implements.
`internal/specdoc` now makes checking it cheap. Out of this Bug's stated scope;
**recommend a follow-up Bug**. The README claim that the artifact "cannot verify
itself against `SPEC.md`, which this repository does not vendor" was corrected,
since the document is now vendored.

## Class A — quote absent from the pinned document (149 rows)

| Shape | Member | Pre-fix cell | Text not found in SPEC.md | Replacement |
| --- | --- | --- | --- | --- |
| `Blob Descriptor` | `chunks` | “BlobChunk[0..32768]; empty exactly when size is zero and exact coverage otherwise” | BlobChunk[0..32768]; empty exactly when size is zero and exact coverage otherwise | L4619 “<code>chunks:BlobChunk[0..32768]</code>” |
| `Blob Descriptor` | `descriptor_id` | “digest; canonical object digest” | digest; canonical object digest | L4617 “<code>descriptor_id:digest</code>” |
| `Blob Descriptor` | `media_type` | “string[1..255]; lowercase ASCII type/subtype without parameters” | string[1..255]; lowercase ASCII type/subtype without parameters | L4618 “<code>media_type:string[1..255]</code>” |
| `Blob Descriptor` | `schema` | “string; exact urn:ax:schema:blob” | string; exact urn:ax:schema:blob | L4615 “The descriptor is closed and contains exactly <code>schema</code>”; L4610 “Every transferred blob has a Blob Descriptor with schema <code>urn:ax:schema:blob</code>” |
| `Blob Descriptor` | `schema_version` | “string; exact 1.0.0” | string; exact 1.0.0 | L4615 “contains exactly <code>schema</code>, <code>schema_version</code>”; L4611 “<code>urn:ax:schema:blob</code> version <code>1.0.0</code>” |
| `BlobChunk` | `index` | “uint32; starts at zero and increases by one” | uint32; starts at zero and increases by one | L4621 “<code>index:uint32</code>” |
| `BlobChunk` | `offset` | “uint53; contiguous from zero” | uint53; contiguous from zero | L4621 “<code>offset:uint53</code>” |
| `BlobChunk` | `size` | “uint53[1..4194304]; every non-final chunk is exactly 4194304” | uint53[1..4194304]; every non-final chunk is exactly 4194304 | L4622 “<code>size:uint53[1..4194304]</code>” |
| `EnvironmentTuple` | `adapter_version` | “store_schema_fingerprint, and adapter_version” | store_schema_fingerprint, and adapter_version | L3630 “and <code>adapter_version</code>; it never contains executable provenance” |
| `EnvironmentTuple` | `platform` | “linux, macos, windows, or wsl2” | linux, macos, windows, or wsl2 | L3628 “<code>platform=linux\&#124;macos\&#124;windows\&#124;wsl2</code>” |
| `GitFeatures` | `object_format` | “enum sha1 or sha256” | enum sha1 or sha256 | L4789 “<code>object_format:sha1&#124;sha256</code>” |
| `GitFeatures` | `sparse_checkout` | “boolean; tags sparse pattern digest pair” | boolean; tags sparse pattern digest pair | L4793 “<code>sparse_checkout:boolean</code>” |
| `GitHead` | `mode` | “enum branch, detached, or unborn” | enum branch, detached, or unborn | L4788 “<code>mode:branch&#124;detached&#124;unborn</code>” |
| `GitHead` | `oid` | “git-oid or null; tagged by mode and matching object format” | git-oid or null; tagged by mode and matching object format | L4788 “<code>oid:git-oid&#124;null</code>” |
| `GitHead` | `ref` | “git-ref or null; tagged by mode and unborn uses refs/heads/” | git-ref or null; tagged by mode and unborn uses refs/heads/ | L4788 “<code>ref:git-ref&#124;null</code>” |
| `GitIndex` | `entries` | “GitIndexEntry[0..65536]; sorted by path then stage” | GitIndexEntry[0..65536]; sorted by path then stage | L4790 “<code>entries:GitIndexEntry[0..65536]</code>” |
| `GitIndex` | `entry_count` | “uint53; equals entries length” | uint53; equals entries length | L4790 “<code>entry_count:uint53</code>” |
| `GitIndex` | `format` | “exact git_index” | exact git_index | L4789 “<code>format:git_pack_v2</code>” |
| `GitIndex` | `version` | “enum 2, 3, or 4” | enum 2, 3, or 4 | L4790 “<code>version:2&#124;3&#124;4</code>” |
| `GitIndexEntry` | `oid` | “git-oid; matches object format” | git-oid; matches object format | L4788 “<code>oid:git-oid&#124;null</code>” |
| `GitObjectPack` | `format` | “exact git_pack_v2” | exact git_pack_v2 | L4789 “<code>format:git_pack_v2</code>” |
| `GitObjectPack` | `object_format` | “enum sha1 or sha256” | enum sha1 or sha256 | L4789 “<code>object_format:sha1&#124;sha256</code>” |
| `GitRemote` | `name` | “string[1..128]; remotes sorted by name with no duplicate” | string[1..128]; remotes sorted by name with no duplicate | L4787 “<code>name:string[1..128]</code>” |
| `GitRemote` | `push_url` | “sanitized-git-URL or null” | sanitized-git-URL or null | L4787 “<code>push_url:sanitized-git-URL&#124;null</code>” |
| `GitSubmodule` | `agent_project_config_paths` | “sorted unique path[0..256] or null” | sorted unique path[0..256] or null | L4792 “<code>agent_project_config_paths:sorted unique path[0..256]&#124;null</code>” |
| `GitSubmodule` | `features` | “GitFeatures or null” | GitFeatures or null | L4792 “<code>features:GitFeatures&#124;null</code>” |
| `GitSubmodule` | `gitlink_oid` | “git-oid; equals containing stage-0 mode-160000 entry” | git-oid; equals containing stage-0 mode-160000 entry | L4792 “<code>gitlink_oid:git-oid</code>” |
| `GitSubmodule` | `head` | “GitHead or null” | GitHead or null | L4792 “<code>head:GitHead&#124;null</code>” |
| `GitSubmodule` | `index` | “GitIndex or null” | GitIndex or null | L4792 “<code>index:GitIndex&#124;null</code>” |
| `GitSubmodule` | `initialized` | “boolean; tags all following state members” | boolean; tags all following state members | L4792 “<code>initialized:boolean</code>” |
| `GitSubmodule` | `object_pack` | “GitObjectPack or null” | GitObjectPack or null | L4792 “<code>object_pack:GitObjectPack&#124;null</code>” |
| `GitSubmodule` | `path` | “path; no sibling destination-case collision” | path; no sibling destination-case collision | L4791 “<code>path:path</code>” |
| `GitSubmodule` | `repo_relative_cwd` | “dot or path or null” | dot or path or null | L4792 “<code>repo_relative_cwd:.&#124;path&#124;null</code>” |
| `GitSubmodule` | `repository_identity` | “string[1..256]; recursion acyclic by identity” | string[1..256]; recursion acyclic by identity | L4792 “<code>repository_identity:string[1..256]</code>” |
| `GitSubmodule` | `submodules` | “GitSubmodule[0..256] or null; depth at most 16 and total at most 256” | GitSubmodule[0..256] or null; depth at most 16 and total at most 256 | L4792 “<code>submodules:GitSubmodule[0..256]&#124;null</code>” |
| `GitSubmodule` | `upstream_ref` | “git-ref or null” | git-ref or null | L4792 “<code>upstream_ref:git-ref&#124;null</code>” |
| `ManifestEntry.file` | `type` | “exact file” | exact file | L4746 “<code>type = file</code>” |
| `ManifestEntry.hardlink` | `target_path` | “path; names an earlier file entry with the same mode” | path; names an earlier file entry with the same mode | L4748 “<code>target_path:path</code>” |
| `ManifestEntry.hardlink` | `type` | “exact hardlink” | exact hardlink | L4748 “<code>type = hardlink</code>” |
| `ManifestEntry.symlink` | `target` | “string[1..4096]; lexically remains within materialization root” | string[1..4096]; lexically remains within materialization root | L4747 “<code>target:string[1..4096]</code>” |
| `ManifestEntry.symlink` | `type` | “exact symlink” | exact symlink | L4747 “<code>type = symlink</code>” |
| `MigrationProvenance` | `schema_version` | “canonical semver” | canonical semver | L11461 “<code>schema_version:semver</code>” |
| `Observation Event` | `counts` | “ObservationCounts or null &#124; Closed aggregate below” | ObservationCounts or null &#124; Closed aggregate below | L11603 “\&#124; <code>counts</code> \&#124; ObservationCounts or null \&#124; Closed aggregate below \&#124;” |
| `Observation Event` | `duration_ms` | “uint53 or null &#124; Null for a point/start event; otherwise elapsed milliseconds” | uint53 or null &#124; Null for a point/start event; otherwise elapsed milliseconds | L11602 “\&#124; <code>duration_ms</code> \&#124; uint53 or null \&#124; Null for a point/start event; otherwise elapsed milliseconds \&#124;” |
| `Observation Event` | `error_code` | “string[1..128] or null &#124; Stable Section 15 code when result is partial/failure” | string[1..128] or null &#124; Stable Section 15 code when result is partial/failure | L11605 “\&#124; <code>error_code</code> \&#124; string[1..128] or null \&#124; Stable Section 15 code when result is partial/failure \&#124;” |
| `Observation Event` | `event` | “observation-name &#124; [a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*){1,7}, 3–128 characters” | observation-name &#124; [a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*){1,7}, 3–128 characters | L11595 “\&#124; <code>event</code> \&#124; observation-name \&#124; <code>[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*){1,7}</code>, 3–128 characters \&#124;” |
| `Observation Event` | `extensions` | “object &#124; Reverse-DNS extension keys only; no payload content” | object &#124; Reverse-DNS extension keys only; no payload content | L11606 “\&#124; <code>extensions</code> \&#124; object \&#124; Reverse-DNS extension keys only; no payload content \&#124;” |
| `Observation Event` | `host_id` | “UUIDv7 &#124; Emitting host” | UUIDv7 &#124; Emitting host | L11598 “\&#124; <code>host_id</code> \&#124; UUIDv7 \&#124; Emitting host \&#124;” |
| `Observation Event` | `level` | “enum &#124; debug, info, warn, or error” | enum &#124; debug, info, warn, or error | L11594 “\&#124; <code>level</code> \&#124; enum \&#124; <code>debug</code>, <code>info</code>, <code>warn</code>, or <code>error</code> \&#124;” |
| `Observation Event` | `object_ids` | “sorted unique digest[0..4096] &#124; Redacted object identities only” | sorted unique digest[0..4096] &#124; Redacted object identities only | L11604 “\&#124; <code>object_ids</code> \&#124; sorted unique digest[0..4096] \&#124; Redacted object identities only \&#124;” |
| `Observation Event` | `operation_id` | “UUIDv7 or null &#124; Required null when no operation exists” | UUIDv7 or null &#124; Required null when no operation exists | L11596 “\&#124; <code>operation_id</code> \&#124; UUIDv7 or null \&#124; Required null when no operation exists \&#124;” |
| `Observation Event` | `peer_host_id` | “UUIDv7 or null &#124; Required null when no peer participates” | UUIDv7 or null &#124; Required null when no peer participates | L11599 “\&#124; <code>peer_host_id</code> \&#124; UUIDv7 or null \&#124; Required null when no peer participates \&#124;” |
| `Observation Event` | `phase` | “string[1..128] or null &#124; Stable lower-snake-case phase or null” | string[1..128] or null &#124; Stable lower-snake-case phase or null | L11600 “\&#124; <code>phase</code> \&#124; string[1..128] or null \&#124; Stable lower-snake-case phase or null \&#124;” |
| `Observation Event` | `result` | “enum &#124; started, success, partial, failure, or cancelled” | enum &#124; started, success, partial, failure, or cancelled | L11601 “\&#124; <code>result</code> \&#124; enum \&#124; <code>started</code>, <code>success</code>, <code>partial</code>, <code>failure</code>, or <code>cancelled</code> \&#124;” |
| `Observation Event` | `schema` | “string &#124; Exact Observation schema identifier” | string &#124; Exact Observation schema identifier | L11589 “\&#124; <code>schema</code> \&#124; string \&#124; Exact Observation schema identifier \&#124;” |
| `Observation Event` | `schema_version` | “semver &#124; Exact 1.0.0” | semver &#124; Exact 1.0.0 | L11590 “\&#124; <code>schema_version</code> \&#124; semver \&#124; Exact <code>1.0.0</code> \&#124;” |
| `Observation Event` | `sequence` | “uint53 &#124; Starts at 1 and increases by exactly one before each durable append” | uint53 &#124; Starts at 1 and increases by exactly one before each durable append | L11592 “\&#124; <code>sequence</code> \&#124; uint53 \&#124; Starts at 1 and increases by exactly one before each durable append \&#124;” |
| `Observation Event` | `session_id` | “UUIDv7 or null &#124; Required null for non-session events” | UUIDv7 or null &#124; Required null for non-session events | L11597 “\&#124; <code>session_id</code> \&#124; UUIDv7 or null \&#124; Required null for non-session events \&#124;” |
| `Observation Event` | `stream_id` | “UUIDv7 &#124; Stable per host installation; changing it starts a new explicitly separate stream” | UUIDv7 &#124; Stable per host installation; changing it starts a new explicitly separate stream | L11591 “\&#124; <code>stream_id</code> \&#124; UUIDv7 \&#124; Stable per host installation; changing it starts a new explicitly separate stream \&#124;” |
| `Observation Event` | `timestamp` | “timestamp &#124; Observation time; not authority” | timestamp &#124; Observation time; not authority | L11593 “\&#124; <code>timestamp</code> \&#124; timestamp \&#124; Observation time; not authority \&#124;” |
| `Session Record 1.0.0` | `created_at` | “timestamp; diagnostic time” | timestamp; diagnostic time | L1473 “\&#124; <code>created_at</code> \&#124; timestamp \&#124; Diagnostic time \&#124;” |
| `Session Record 1.0.0` | `created_by_host_id` | “UUIDv7; allowlisted host at creation” | UUIDv7; allowlisted host at creation | L1474 “\&#124; <code>created_by_host_id</code> \&#124; UUIDv7 \&#124; Allowlisted host at creation \&#124;” |
| `Session Record 1.0.0` | `execution_profile` | “enum standard or yolo” | enum standard or yolo | L1477 “\&#124; <code>execution_profile</code> \&#124; enum \&#124; <code>standard</code> or <code>yolo</code> \&#124;” |
| `Session Record 1.0.0` | `extensions` | “object; required, reverse-DNS keys only” | object; required, reverse-DNS keys only | L1481 “\&#124; <code>extensions</code> \&#124; object \&#124; Required; may be empty; reverse-DNS keys only \&#124;” |
| `Session Record 1.0.0` | `fork_provenance` | “Fork Provenance or null; object exactly when created by fork” | Fork Provenance or null; object exactly when created by fork | L1480 “\&#124; <code>fork_provenance</code> \&#124; Fork Provenance or null \&#124; Required object exactly when this record was created by fork \&#124;” |
| `Session Record 1.0.0` | `kind` | “enum direct or task_board” | enum direct or task_board | L1472 “\&#124; <code>kind</code> \&#124; enum \&#124; <code>direct</code> or <code>task_board</code> \&#124;” |
| `Session Record 1.0.0` | `launch_plan` | “Launch Plan; closed, sanitized and secret-free” | Launch Plan; closed, sanitized and secret-free | L1478 “\&#124; <code>launch_plan</code> \&#124; Launch Plan \&#124; Closed shape below; sanitized and secret-free \&#124;” |
| `Session Record 1.0.0` | `name` | “string; Section 2.1 grammar [A-Za-z0-9][A-Za-z0-9._-]{0,63} and 1–64 characters” | string; Section 2.1 grammar [A-Za-z0-9][A-Za-z0-9._-]{0,63} and 1–64 characters | L1471 “\&#124; <code>name</code> \&#124; string \&#124; Section 2.3 grammar \&#124;”; L363 “\&#124; Session name \&#124; A mesh-unique human alias of 1–64 characters matching <code>[A-Za-z0-9][A-Za-z0-9._-]{0,63}</code>. \&#124;” |
| `Session Record 1.0.0` | `provider_id` | “string; lowercase plugin ID” | string; lowercase plugin ID | L1475 “\&#124; <code>provider_id</code> \&#124; string \&#124; Lowercase plugin ID \&#124;” |
| `Session Record 1.0.0` | `record_id` | “digest; canonical object digest” | digest; canonical object digest | L1468 “\&#124; <code>record_id</code> \&#124; digest \&#124; Canonical object digest \&#124;” |
| `Session Record 1.0.0` | `schema` | “string; exact schema identifier” | string; exact schema identifier | L1466 “\&#124; <code>schema</code> \&#124; string \&#124; Exact schema identifier \&#124;” |
| `Session Record 1.0.0` | `schema_version` | “semver; exact 1.0.0” | semver; exact 1.0.0 | L1467 “\&#124; <code>schema_version</code> \&#124; semver \&#124; <code>1.0.0</code> \&#124;” |
| `Session Record 1.0.0` | `session_id` | “UUIDv7; globally unique” | UUIDv7; globally unique | L1470 “\&#124; <code>session_id</code> \&#124; UUIDv7 \&#124; Globally unique \&#124;” |
| `Session Record 1.0.0` | `subject_id` | “UUIDv7; equal to session_id” | UUIDv7; equal to session_id | L1469 “\&#124; <code>subject_id</code> \&#124; UUIDv7 \&#124; Equal to <code>session_id</code> \&#124;” |
| `Session Record 1.0.0` | `task_board` | “Task-board Reference or null; object exactly when kind is task_board” | Task-board Reference or null; object exactly when kind is task_board | L1479 “\&#124; <code>task_board</code> \&#124; Task-board Reference or null \&#124; Required object exactly when <code>kind = task_board</code> \&#124;” |
| `Session Record 1.0.0` | `workspace_group_id` | “UUIDv7; required” | UUIDv7; required | L1476 “\&#124; <code>workspace_group_id</code> \&#124; UUIDv7 \&#124; Required \&#124;” |
| `Session Record 2.0.0 and 3.0.0` | `created_at` | “timestamp; diagnostic time” | timestamp; diagnostic time | L1473 “\&#124; <code>created_at</code> \&#124; timestamp \&#124; Diagnostic time \&#124;” |
| `Session Record 2.0.0 and 3.0.0` | `created_by_host_id` | “UUIDv7; allowlisted host at creation” | UUIDv7; allowlisted host at creation | L1474 “\&#124; <code>created_by_host_id</code> \&#124; UUIDv7 \&#124; Allowlisted host at creation \&#124;” |
| `Session Record 2.0.0 and 3.0.0` | `derivation_provenance` | “required closed derivation provenance; v3 closed creation union” | required closed derivation provenance; v3 closed creation union | L1623 “It retains every major-1 field except <code>fork_provenance</code>, which is replaced by required closed <code>derivation_provenance</code>” |
| `Session Record 2.0.0 and 3.0.0` | `execution_profile` | “enum standard or yolo” | enum standard or yolo | L1477 “\&#124; <code>execution_profile</code> \&#124; enum \&#124; <code>standard</code> or <code>yolo</code> \&#124;” |
| `Session Record 2.0.0 and 3.0.0` | `extensions` | “object; required, reverse-DNS keys only” | object; required, reverse-DNS keys only | L1481 “\&#124; <code>extensions</code> \&#124; object \&#124; Required; may be empty; reverse-DNS keys only \&#124;” |
| `Session Record 2.0.0 and 3.0.0` | `kind` | “enum direct or task_board” | enum direct or task_board | L1472 “\&#124; <code>kind</code> \&#124; enum \&#124; <code>direct</code> or <code>task_board</code> \&#124;” |
| `Session Record 2.0.0 and 3.0.0` | `launch_plan` | “Launch Plan; closed, sanitized and secret-free” | Launch Plan; closed, sanitized and secret-free | L1478 “\&#124; <code>launch_plan</code> \&#124; Launch Plan \&#124; Closed shape below; sanitized and secret-free \&#124;” |
| `Session Record 2.0.0 and 3.0.0` | `name` | “string; Section 2.1 grammar [A-Za-z0-9][A-Za-z0-9._-]{0,63} and 1–64 characters” | string; Section 2.1 grammar [A-Za-z0-9][A-Za-z0-9._-]{0,63} and 1–64 characters | L1471 “\&#124; <code>name</code> \&#124; string \&#124; Section 2.3 grammar \&#124;”; L363 “\&#124; Session name \&#124; A mesh-unique human alias of 1–64 characters matching <code>[A-Za-z0-9][A-Za-z0-9._-]{0,63}</code>. \&#124;” |
| `Session Record 2.0.0 and 3.0.0` | `provider_id` | “provider ID allocated at creation and immutable” | provider ID allocated at creation and immutable | L1475 “\&#124; <code>provider_id</code> \&#124; string \&#124; Lowercase plugin ID \&#124;”; L1676 “The new target Session ID and target <code>provider_id</code> are allocated at creation and never reuse or mutate the source Session or source provider ID.” |
| `Session Record 2.0.0 and 3.0.0` | `record_id` | “digest; canonical object digest” | digest; canonical object digest | L1468 “\&#124; <code>record_id</code> \&#124; digest \&#124; Canonical object digest \&#124;” |
| `Session Record 2.0.0 and 3.0.0` | `schema` | “string; exact Session Record schema identifier” | string; exact Session Record schema identifier | L1466 “\&#124; <code>schema</code> \&#124; string \&#124; Exact schema identifier \&#124;” |
| `Session Record 2.0.0 and 3.0.0` | `schema_version` | “independently closed 2.0.0 and 3.0.0 variants” | independently closed 2.0.0 and 3.0.0 variants | L1467 “\&#124; <code>schema_version</code> \&#124; semver \&#124; <code>1.0.0</code> \&#124;”; L1622 “Session Record 2.0.0 is emitted in v0.3.0 only for a cross-environment clone target.”; L1685 “Session Record 3.0.0 is the v0.4 creation contract.” |
| `Session Record 2.0.0 and 3.0.0` | `session_id` | “UUIDv7; newly allocated and globally unique” | UUIDv7; newly allocated and globally unique | L1470 “\&#124; <code>session_id</code> \&#124; UUIDv7 \&#124; Globally unique \&#124;”; L1676 “The new target Session ID and target <code>provider_id</code> are allocated at creation” |
| `Session Record 2.0.0 and 3.0.0` | `subject_id` | “UUIDv7; equal to session_id” | UUIDv7; equal to session_id | L1469 “\&#124; <code>subject_id</code> \&#124; UUIDv7 \&#124; Equal to <code>session_id</code> \&#124;” |
| `Session Record 2.0.0 and 3.0.0` | `task_board` | “Task-board Reference or null; orthogonal authority” | Task-board Reference or null; orthogonal authority | L1479 “\&#124; <code>task_board</code> \&#124; Task-board Reference or null \&#124; Required object exactly when <code>kind = task_board</code> \&#124;”; L1681 “Task-board references remain orthogonal authority in the existing <code>task_board</code> field” |
| `Session Record 2.0.0 and 3.0.0` | `workspace_group_id` | “UUIDv7; required” | UUIDv7; required | L1476 “\&#124; <code>workspace_group_id</code> \&#124; UUIDv7 \&#124; Required \&#124;” |
| `Session Record Board Goal` | `extensions` | “object; reverse-DNS extension keys only” | object; reverse-DNS extension keys only | L1521 “greater than zero, and <code>extensions</code>” |
| `Session Record Board Goal` | `goal_id` | “string[1..128]; public goal reference” | string[1..128]; public goal reference | L1520 “<code>goal_id</code> as a 1–128 character public goal reference” |
| `Session Record Board Goal` | `schema` | “string; exact board-goal-v2” | string; exact board-goal-v2 | L1520 “<code>schema = "board-goal-v2"</code>” |
| `Session Record Board Identity` | `extensions` | “object; reverse-DNS extension keys only” | object; reverse-DNS extension keys only | L1517 “and <code>extensions</code>. A local board requires null <code>remote_url</code>” |
| `Session Record Board Identity` | `kind` | “enum local or remote” | enum local or remote | L1514 “Board Identity has exactly <code>kind</code> (<code>local</code> or <code>remote</code>)” |
| `Session Record Board Identity` | `logical_id` | “string[1..128]; [A-Za-z0-9][A-Za-z0-9._:-]{0,127}” | string[1..128]; [A-Za-z0-9][A-Za-z0-9._:-]{0,127} | L1515 “<code>logical_id</code> (1–128 characters matching <code>[A-Za-z0-9][A-Za-z0-9._:-]{0,127}</code>)” |
| `Session Record Board Identity` | `remote_url` | “absolute HTTPS URL or null; tagged by kind and no userinfo, query, or fragment” | absolute HTTPS URL or null; tagged by kind and no userinfo, query, or fragment | L1517 “<code>remote_url</code> (absolute <code>https</code> URL or null)” |
| `Session Record Fork Provenance` | `extensions` | “object; reverse-DNS extension keys only” | object; reverse-DNS extension keys only | L1537 “or <code>task_board_clone</code>, and <code>extensions</code>” |
| `Session Record Fork Provenance` | `provider_fork_mode` | “enum native, supported_import, or task_board_clone” | enum native, supported_import, or task_board_clone | L1536 “<code>provider_fork_mode</code> as <code>native</code>, <code>supported_import</code>, or <code>task_board_clone</code>” |
| `Session Record Launch Plan` | `argv` | “array<string>[1..128]; each 1–4096 UTF-8 bytes and total encoded argv at most 65536 bytes” | array<string>[1..128]; each 1–4096 UTF-8 bytes and total encoded argv at most 65536 bytes | L1487 “\&#124; <code>argv</code> \&#124; array&lt;string&gt;[1..128] \&#124; Each element is 1–4,096 UTF-8 bytes; total encoded argv is at most 65,536 bytes; never a shell command string \&#124;” |
| `Session Record Launch Plan` | `contains_secrets` | “boolean; MUST be false” | boolean; MUST be false | L1492 “\&#124; <code>contains_secrets</code> \&#124; boolean \&#124; MUST be false \&#124;” |
| `Session Record Launch Plan` | `cwd_relative` | “string; dot or Section 1.6 path” | string; dot or Section 1.6 path | L1489 “\&#124; <code>cwd_relative</code> \&#124; string \&#124; <code>.</code> for the workspace root or a path satisfying Section 1.6 \&#124;” |
| `Session Record Launch Plan` | `cwd_workspace_id` | “UUIDv7; names one workspace in the record workspace group” | UUIDv7; names one workspace in the record workspace group | L1488 “\&#124; <code>cwd_workspace_id</code> \&#124; UUIDv7 \&#124; Names one workspace in the Session Record's workspace group \&#124;” |
| `Session Record Launch Plan` | `env_literals` | “map(environment-name,string)[0..64]; values at most 4096 UTF-8 bytes and keys disjoint from env_names” | map(environment-name,string)[0..64]; values at most 4096 UTF-8 bytes and keys disjoint from env_names | L1491 “\&#124; <code>env_literals</code> \&#124; map(environment-name,string)[0..64] \&#124; Non-secret literals of at most 4,096 UTF-8 bytes each; keys sorted in canonical form and disjoint from <code>env_names</code> \&#124;” |
| `Session Record Launch Plan` | `env_names` | “array<string>[0..64]; sorted unique environment names” | array<string>[0..64]; sorted unique environment names | L1490 “\&#124; <code>env_names</code> \&#124; array&lt;string&gt;[0..64] \&#124; Sorted, unique names matching <code>[A-Za-z_][A-Za-z0-9_]{0,127}</code>; values resolve only from destination-local state \&#124;” |
| `Session Record Launch Plan` | `extensions` | “object; reverse-DNS extension keys only” | object; reverse-DNS extension keys only | L1493 “\&#124; <code>extensions</code> \&#124; object \&#124; Reverse-DNS extension keys only \&#124;” |
| `Session Record Task-board Reference` | `board` | “Board Identity; closed shape” | Board Identity; closed shape | L1506 “\&#124; <code>board</code> \&#124; Board Identity \&#124; Closed shape below \&#124;” |
| `Session Record Task-board Reference` | `board_goal` | “Board Goal or null; non-null for primary_owner” | Board Goal or null; non-null for primary_owner | L1510 “\&#124; <code>board_goal</code> \&#124; Board Goal or null \&#124; Required non-null for <code>primary_owner</code> \&#124;” |
| `Session Record Task-board Reference` | `bridge_protocol_version` | “semver; exact 1.0.0” | semver; exact 1.0.0 | L1505 “\&#124; <code>bridge_protocol_version</code> \&#124; semver \&#124; Exact <code>1.0.0</code> \&#124;” |
| `Session Record Task-board Reference` | `extensions` | “object; reverse-DNS extension keys only” | object; reverse-DNS extension keys only | L1512 “\&#124; <code>extensions</code> \&#124; object \&#124; Reverse-DNS extension keys only \&#124;” |
| `Session Record Task-board Reference` | `launch_mode` | “enum primary_owner or tracked_prompt” | enum primary_owner or tracked_prompt | L1508 “\&#124; <code>launch_mode</code> \&#124; enum \&#124; <code>primary_owner</code> or <code>tracked_prompt</code> \&#124;” |
| `Session Record Task-board Reference` | `manager_session_ref` | “string or null; MUST be null in immutable creation record” | string or null; MUST be null in immutable creation record | L1509 “\&#124; <code>manager_session_ref</code> \&#124; string or null \&#124; MUST be null in the immutable creation record; the public reference is established by <code>task_board.launched</code> and may later change through <code>task_board.adopted</code> \&#124;” |
| `Session Record Task-board Reference` | `native_goal_binding` | “enum bound, prompt, or none; bound exactly for primary_owner” | enum bound, prompt, or none; bound exactly for primary_owner | L1511 “\&#124; <code>native_goal_binding</code> \&#124; enum \&#124; <code>bound</code>, <code>prompt</code>, or <code>none</code> \&#124;” |
| `Session Record Task-board Reference` | `task_element_id` | “string; 1–128 printable non-control UTF-8 bytes” | string; 1–128 printable non-control UTF-8 bytes | L1507 “\&#124; <code>task_element_id</code> \&#124; string \&#124; 1–128 printable non-control UTF-8 bytes \&#124;” |
| `Session Record cross-environment-clone provenance` | `kind` | “exact cross_environment_clone” | exact cross_environment_clone | L1656 “The <code>cross_environment_clone</code> variant's exact typed members are <code>kind</code>” |
| `Session Record cross-environment-clone provenance` | `source_checkpoint_id` | “digest or null; non-null exactly for ax_session” | digest or null; non-null exactly for ax_session | L1662 “<code>source_checkpoint_id:digest\&#124;null</code>” |
| `Session Record cross-environment-clone provenance` | `source_kind` | “ax_session or external_native” | ax_session or external_native | L1659 “<code>source_kind:ax_session\&#124;external_native</code>” |
| `Session Record cross-environment-clone provenance` | `source_native_session_id` | “sanitized non-authoritative source native session ID; string[1..512]” | sanitized non-authoritative source native session ID; string[1..512] | L1664 “<code>source_native_session_id:string[1..512]</code>” |
| `Session Record cross-environment-clone provenance` | `source_provider_identity_record_id` | “digest or null; non-null exactly for ax_session” | digest or null; non-null exactly for ax_session | L1663 “<code>source_provider_identity_record_id:digest\&#124;null</code>” |
| `Session Record cross-environment-clone provenance` | `source_session_id` | “UUIDv7 or null; non-null exactly for ax_session” | UUIDv7 or null; non-null exactly for ax_session | L1660 “<code>source_session_id:UUIDv7\&#124;null</code>” |
| `Session Record cross-environment-clone provenance` | `source_session_record_id` | “digest or null; non-null exactly for ax_session” | digest or null; non-null exactly for ax_session | L1661 “<code>source_session_record_id:digest\&#124;null</code>” |
| `Session Record native-adoption provenance` | `kind` | “exact native_adoption” | exact native_adoption | L1691 “<code>kind=native_adoption</code>” |
| `Session Record native-adoption provenance` | `target_provider_id` | “provider-id; target provider allocated at creation” | provider-id; target provider allocated at creation | L1696 “<code>target_provider_id:provider-id</code>” |
| `Session Record origin provenance` | `kind` | “exact origin” | exact origin | L1646 “<code>kind=origin</code>” |
| `Session Record same-provider-fork provenance` | `kind` | “exact same_provider_fork” | exact same_provider_fork | L1648 “<code>kind=same_provider_fork</code>” |
| `Session Record same-provider-fork provenance` | `provider_fork_mode` | “native, supported_import, or task_board_clone” | native, supported_import, or task_board_clone | L1653 “<code>provider_fork_mode:native\&#124;supported_import\&#124;task_board_clone</code>” |
| `Session Record same-provider-fork provenance` | `source_session_id` | “source_session_id UUIDv7; fork creates a new logical session” | source_session_id UUIDv7; fork creates a new logical session | L1649 “<code>source_session_id:UUIDv7</code>” |
| `Transfer Manifest` | `entries` | “ManifestEntry[0..65536]; strictly bytewise sorted with no destination-case collision” | ManifestEntry[0..65536]; strictly bytewise sorted with no destination-case collision | L4696 “\&#124; <code>entries</code> \&#124; ManifestEntry[0..65536] \&#124; Sorted bytewise by normalized path \&#124;” |
| `Transfer Manifest` | `extensions` | “object; reverse-DNS extension keys only” | object; reverse-DNS extension keys only | L4704 “\&#124; <code>extensions</code> \&#124; object \&#124; Reverse-DNS extension keys only \&#124;” |
| `Transfer Manifest` | `kind` | “enum workspace_group, workspace_tree, provider, task_board, or composite” | enum workspace_group, workspace_tree, provider, task_board, or composite | L4693 “\&#124; <code>kind</code> \&#124; enum \&#124; <code>workspace_group</code>, <code>workspace_tree</code>, <code>provider</code>, <code>task_board</code>, or <code>composite</code> \&#124;” |
| `Transfer Manifest` | `manifest_id` | “digest; canonical object digest” | digest; canonical object digest | L4692 “\&#124; <code>manifest_id</code> \&#124; digest \&#124; Canonical object digest \&#124;” |
| `Transfer Manifest` | `provider_identity_record_id` | “digest or null; non-null only for provider” | digest or null; non-null only for provider | L4699 “\&#124; <code>provider_identity_record_id</code> \&#124; digest or null \&#124; Non-null only for <code>provider</code> \&#124;” |
| `Transfer Manifest` | `schema` | “string; exact Transfer Manifest schema identifier” | string; exact Transfer Manifest schema identifier | L4690 “\&#124; <code>schema</code> \&#124; string \&#124; Exact Transfer Manifest schema identifier \&#124;” |
| `Transfer Manifest` | `schema_version` | “semver; exact 1.0.0” | semver; exact 1.0.0 | L4691 “\&#124; <code>schema_version</code> \&#124; semver \&#124; Exact <code>1.0.0</code> \&#124;” |
| `Transfer Manifest` | `subject_id` | “UUIDv7; scope selected by kind” | UUIDv7; scope selected by kind | L4694 “\&#124; <code>subject_id</code> \&#124; UUIDv7 \&#124; Group, workspace, or session selected by kind \&#124;” |
| `Transfer Manifest` | `task_board_bundle_id` | “digest or null; non-null only for task_board” | digest or null; non-null only for task_board | L4700 “\&#124; <code>task_board_bundle_id</code> \&#124; digest or null \&#124; Non-null only for <code>task_board</code> \&#124;” |
| `Transfer Manifest` | `workspace_snapshot` | “WorkspaceSnapshot or null; non-null only for workspace_group” | WorkspaceSnapshot or null; non-null only for workspace_group | L4698 “\&#124; <code>workspace_snapshot</code> \&#124; WorkspaceSnapshot or null \&#124; Non-null only for <code>workspace_group</code> \&#124;” |
| `WorkspaceSnapshot` | `members` | “WorkspaceSnapshotMember[1..256]; strict workspace-ID order and no destination-case-colliding group paths” | WorkspaceSnapshotMember[1..256]; strict workspace-ID order and no destination-case-colliding group paths | L4773 “<code>members:WorkspaceSnapshotMember[1..256]</code>” |
| `WorkspaceSnapshot` | `workspace_group_id` | “UUIDv7; equals manifest subject” | UUIDv7; equals manifest subject | L4772 “<code>workspace_group_id:UUIDv7</code>” |
| `WorkspaceSnapshotMember.git` | `kind` | “exact git” | exact git | L4780 “<code>kind = git</code>” |
| `WorkspaceSnapshotMember.git` | `materialization_policy` | “enum shared_checkout or separate_worktree” | enum shared_checkout or separate_worktree | L4780 “<code>materialization_policy:shared_checkout&#124;separate_worktree</code>” |
| `WorkspaceSnapshotMember.git` | `remotes` | “GitRemote[1..16]; sorted by name with no duplicate” | GitRemote[1..16]; sorted by name with no duplicate | L4780 “<code>remotes:GitRemote[1..16]</code>” |
| `WorkspaceSnapshotMember.git` | `repo_relative_cwd` | “dot or path” | dot or path | L4780 “<code>repo_relative_cwd:.&#124;path</code>” |
| `WorkspaceSnapshotMember.git` | `upstream_ref` | “git-ref or null” | git-ref or null | L4780 “<code>upstream_ref:git-ref&#124;null</code>” |
| `WorkspaceSnapshotMember.managed_tree` | `kind` | “exact managed_tree” | exact managed_tree | L4781 “<code>kind = managed_tree</code>” |
| `WorkspaceSnapshotMember.managed_tree` | `materialization_policy` | “enum shared_tree or separate_copy” | enum shared_tree or separate_copy | L4781 “<code>materialization_policy:shared_tree&#124;separate_copy</code>” |
| `WorkspaceSnapshotMember.managed_tree` | `repo_relative_cwd` | “dot or path” | dot or path | L4781 “<code>repo_relative_cwd:.&#124;path</code>” |

## Class B — quote present but unanchored (161 rows)

The text occurs in the pinned document, but at several places, so the cell
named no particular declaration. `Occurrences` is how many places the quoted
text appears.

| Shape | Member | Pre-fix cell | Occurrences | Replacement |
| --- | --- | --- | ---: | --- |
| `Blob Descriptor` | `blob_id` | “digest” | 976 | L4617 “<code>blob_id:digest</code>” |
| `Blob Descriptor` | `size` | “uint53” | 195 | L4618 “<code>size:uint53</code>” |
| `BlobChunk` | `chunk_id` | “digest” | 976 | L4622 “<code>chunk_id:digest</code>” |
| `Checkpoint Record` | `checkpoint_id` | “Canonical object digest” | 7 | L1976 “\&#124; <code>checkpoint_id</code> \&#124; digest \&#124; Canonical object digest \&#124;” |
| `Checkpoint Record` | `created_at` | “Diagnostic only” | 6 | L1987 “\&#124; <code>created_at</code> \&#124; timestamp \&#124; Diagnostic only \&#124;” |
| `Checkpoint Record` | `event_heads` | “sorted unique digest[1..64]” | 5 | L1982 “\&#124; <code>event_heads</code> \&#124; sorted unique digest[1..64] \&#124; Authoritative event DAG heads immediately before this object \&#124;” |
| `Checkpoint Record` | `extensions` | “Reverse-DNS extension keys only” | 9 | L1989 “\&#124; <code>extensions</code> \&#124; object \&#124; Reverse-DNS extension keys only \&#124;” |
| `Checkpoint Record` | `lease_id` | “UUIDv4” | 29 | L1980 “\&#124; <code>lease_id</code> \&#124; UUIDv4 \&#124; Equal to that lease's fencing token \&#124;” |
| `Checkpoint Record` | `safe_boundary` | “Safe Boundary Evidence” | 5 | L1981 “\&#124; <code>safe_boundary</code> \&#124; Safe Boundary Evidence \&#124; Closed shape below \&#124;” |
| `Checkpoint Record` | `schema_version` | “1.0.0” | 336 | L1975 “\&#124; <code>schema_version</code> \&#124; semver \&#124; Exact <code>1.0.0</code> \&#124;” |
| `Checkpoint Record` | `status` | “Literal” | 4 | L1988 “\&#124; <code>status</code> \&#124; enum \&#124; Literal <code>validated</code> \&#124;” |
| `Checkpoint Record` | `subject_id` | “Equal to” | 8 | L1977 “\&#124; <code>subject_id</code> \&#124; UUIDv7 \&#124; Equal to <code>session_id</code> \&#124;” |
| `EnvironmentTuple` | `architecture` | “amd64 or arm64” | 4 | L3629 “<code>architecture=amd64\&#124;arm64</code>” |
| `EnvironmentTuple` | `environment_version` | “environment_version” | 4 | L3627 “<code>environment_id</code>, <code>environment_version</code>” |
| `GitFeatures` | `case_sensitive` | “boolean” | 202 | L4793 “<code>case_sensitive:boolean</code>” |
| `GitFeatures` | `filemode` | “boolean” | 202 | L4793 “<code>filemode:boolean</code>” |
| `GitFeatures` | `lfs_required` | “boolean” | 202 | L4793 “<code>lfs_required:boolean</code>” |
| `GitFeatures` | `precompose_unicode` | “boolean” | 202 | L4793 “<code>precompose_unicode:boolean</code>” |
| `GitFeatures` | `required_filter_names` | “sorted unique string[0..64]” | 6 | L4793 “<code>required_filter_names:sorted unique string[0..64]</code>” |
| `GitFeatures` | `sparse_patterns_blob_descriptor_id` | “digest or null” | 28 | L4793 “<code>sparse_patterns_blob_descriptor_id:digest&#124;null</code>” |
| `GitFeatures` | `sparse_patterns_blob_id` | “digest or null” | 28 | L4793 “<code>sparse_patterns_blob_id:digest&#124;null</code>” |
| `GitFeatures` | `symlinks` | “boolean” | 202 | L4793 “<code>symlinks:boolean</code>” |
| `GitIndex` | `blob_descriptor_id` | “digest” | 976 | L4789 “<code>blob_descriptor_id:digest</code>” |
| `GitIndex` | `blob_id` | “digest” | 976 | L4789 “<code>blob_id:digest</code>” |
| `GitIndexEntry` | `assume_unchanged` | “boolean” | 202 | L4791 “<code>assume_unchanged:boolean</code>” |
| `GitIndexEntry` | `fsmonitor_valid` | “boolean” | 202 | L4791 “<code>fsmonitor_valid:boolean</code>” |
| `GitIndexEntry` | `intent_to_add` | “boolean” | 202 | L4791 “<code>intent_to_add:boolean</code>” |
| `GitIndexEntry` | `mode` | “uint32” | 12 | L4788 “<code>mode:branch&#124;detached&#124;unborn</code>” |
| `GitIndexEntry` | `path` | “path” | 491 | L4791 “<code>path:path</code>” |
| `GitIndexEntry` | `skip_worktree` | “boolean” | 202 | L4791 “<code>skip_worktree:boolean</code>” |
| `GitObjectPack` | `blob_descriptor_id` | “digest” | 976 | L4789 “<code>blob_descriptor_id:digest</code>” |
| `GitObjectPack` | `blob_id` | “digest” | 976 | L4789 “<code>blob_id:digest</code>” |
| `GitObjectPack` | `inventory_blob_descriptor_id` | “digest” | 976 | L4789 “<code>inventory_blob_descriptor_id:digest</code>” |
| `GitObjectPack` | `inventory_blob_id` | “digest” | 976 | L4789 “<code>inventory_blob_id:digest</code>” |
| `GitObjectPack` | `object_count` | “uint53” | 195 | L4789 “<code>object_count:uint53</code>” |
| `GitRemote` | `fetch_url` | “sanitized-git-URL” | 4 | L4787 “<code>fetch_url:sanitized-git-URL</code>” |
| `GitSubmodule` | `sanitized_url` | “sanitized-git-URL” | 4 | L4792 “<code>sanitized_url:sanitized-git-URL</code>” |
| `GitSubmodule` | `working_tree_manifest_id` | “digest or null” | 28 | L4792 “<code>working_tree_manifest_id:digest&#124;null</code>” |
| `Lease Record` | `created_at` | “Diagnostic only” | 6 | L1912 “\&#124; <code>created_at</code> \&#124; timestamp \&#124; Diagnostic only \&#124;” |
| `Lease Record` | `created_by_host_id` | “MUST equal” | 36 | L1911 “\&#124; <code>created_by_host_id</code> \&#124; UUIDv7 \&#124; MUST equal <code>issued_by_host_id</code> \&#124;” |
| `Lease Record` | `extensions` | “Reverse-DNS extension keys only” | 9 | L1913 “\&#124; <code>extensions</code> \&#124; object \&#124; Reverse-DNS extension keys only \&#124;” |
| `Lease Record` | `lease_id` | “UUIDv4” | 29 | L1903 “\&#124; <code>lease_id</code> \&#124; UUIDv4 \&#124; Cryptographically random unique fencing token \&#124;” |
| `Lease Record` | `schema_version` | “1.0.0” | 336 | L1900 “\&#124; <code>schema_version</code> \&#124; semver \&#124; Exact <code>1.0.0</code> \&#124;” |
| `Lease Record` | `subject_id` | “Equal to” | 8 | L1902 “\&#124; <code>subject_id</code> \&#124; UUIDv7 \&#124; Equal to <code>session_id</code> \&#124;” |
| `ManifestEntry.directory` | `mode` | “uint32[0..4095]” | 6 | L4745 “<code>mode:uint32[0..4095]</code>” |
| `ManifestEntry.directory` | `path` | “path” | 491 | L4745 “<code>path:path</code>” |
| `ManifestEntry.directory` | `type` | “exact directory” | 2 | L4745 “<code>type = directory</code>” |
| `ManifestEntry.file` | `blob_descriptor_id` | “digest” | 976 | L4746 “<code>blob_descriptor_id:digest</code>” |
| `ManifestEntry.file` | `blob_id` | “digest” | 976 | L4746 “<code>blob_id:digest</code>” |
| `ManifestEntry.file` | `mode` | “uint32[0..4095]” | 6 | L4746 “<code>mode:uint32[0..4095]</code>” |
| `ManifestEntry.file` | `path` | “path” | 491 | L4746 “<code>path:path</code>” |
| `ManifestEntry.file` | `size` | “uint53” | 195 | L4746 “<code>size:uint53</code>” |
| `ManifestEntry.hardlink` | `mode` | “uint32[0..4095]” | 6 | L4748 “<code>mode:uint32[0..4095]</code>” |
| `ManifestEntry.hardlink` | `path` | “path” | 491 | L4748 “<code>path:path</code>” |
| `ManifestEntry.symlink` | `mode` | “uint32[0..4095]” | 6 | L4747 “<code>mode:uint32[0..4095]</code>” |
| `ManifestEntry.symlink` | `path` | “path” | 491 | L4747 “<code>path:path</code>” |
| `MigrationProvenance` | `object_id` | “digest” | 976 | L11461 “<code>object_id:digest</code>” |
| `MigrationProvenance` | `schema_id` | “string” | 373 | L11460 “<code>schema_id:string</code>” |
| `ObservationCounts` | `blobs` | “blobs:uint53” | 3 | L11610 “<code>blobs:uint53</code>” |
| `ObservationCounts` | `bytes` | “bytes:uint53” | 17 | L11611 “<code>bytes:uint53</code>” |
| `ObservationCounts` | `events` | “events:uint53” | 6 | L11609 “<code>events:uint53</code>” |
| `Provider Identity Record` | `backend_realm_fingerprint` | “digest or null” | 28 | L2088 “\&#124; <code>backend_realm_fingerprint</code> \&#124; digest or null \&#124; Non-secret fingerprint; non-null when backend/account realm is a resume precondition \&#124;” |
| `Provider Identity Record` | `created_at` | “Diagnostic only” | 6 | L2091 “\&#124; <code>created_at</code> \&#124; timestamp \&#124; Diagnostic only \&#124;” |
| `Provider Identity Record` | `extensions` | “Reverse-DNS extension keys only” | 9 | L2092 “\&#124; <code>extensions</code> \&#124; object \&#124; Reverse-DNS extension keys only \&#124;” |
| `Provider Identity Record` | `identity_kind` | “enum” | 61 | L2086 “\&#124; <code>identity_kind</code> \&#124; enum \&#124; <code>session_uuid</code>, <code>session_path_or_id</code>, <code>backend_conversation_uuid</code>, <code>task_board_managed</code>, or <code>provider_defined</code> \&#124;” |
| `Provider Identity Record` | `logical_workspace_id` | “UUIDv7” | 480 | L2087 “\&#124; <code>logical_workspace_id</code> \&#124; UUIDv7 \&#124; Member of the Session Record workspace group \&#124;” |
| `Provider Identity Record` | `native_session_id` | “string[1..512]” | 103 | L2085 “\&#124; <code>native_session_id</code> \&#124; string[1..512] \&#124; Opaque provider handle; never interpreted by core \&#124;” |
| `Provider Identity Record` | `provider_id` | “provider-id” | 46 | L2082 “\&#124; <code>provider_id</code> \&#124; provider-id \&#124; Must equal the Session Record provider \&#124;” |
| `Provider Identity Record` | `provider_version` | “string[1..128]” | 59 | L2083 “\&#124; <code>provider_version</code> \&#124; string[1..128] \&#124; Exact probed version \&#124;” |
| `Provider Identity Record` | `provider_version_range` | “string[1..256]” | 32 | L2084 “\&#124; <code>provider_version_range</code> \&#124; string[1..256] \&#124; Adapter compatibility range used for this identity \&#124;” |
| `Provider Identity Record` | `record_id` | “Canonical object digest” | 7 | L2079 “\&#124; <code>record_id</code> \&#124; digest \&#124; Canonical object digest \&#124;” |
| `Provider Identity Record` | `schema_version` | “1.0.0” | 336 | L2078 “\&#124; <code>schema_version</code> \&#124; semver \&#124; Exact <code>1.0.0</code> \&#124;” |
| `Provider Identity Record` | `subject_id` | “Equal to” | 8 | L2080 “\&#124; <code>subject_id</code> \&#124; UUIDv7 \&#124; Equal to <code>session_id</code> \&#124;” |
| `Safe Boundary Evidence` | `background_idle` | “background_idle:boolean” | 6 | L1996 “<code>background_idle:boolean</code>” |
| `Safe Boundary Evidence` | `evidence` | “accepted_test” | 7 | L1994 “<code>evidence:provider_api\&#124;provider_event\&#124;managed_pty\&#124;task_board_bridge\&#124;accepted_test</code>” |
| `Safe Boundary Evidence` | `foreground_idle` | “foreground_idle:boolean” | 6 | L1995 “<code>foreground_idle:boolean</code>” |
| `Safe Boundary Evidence` | `input_blocked` | “input_blocked:boolean” | 6 | L1995 “<code>input_blocked:boolean</code>” |
| `Safe Boundary Evidence` | `open_database_handles` | “open_database_handles:uint53” | 3 | L1997 “<code>open_database_handles:uint53</code>” |
| `Safe Boundary Evidence` | `open_processes` | “open_processes:uint53” | 3 | L1996 “<code>open_processes:uint53</code>” |
| `Safe Boundary Evidence` | `provider_id` | “provider-id” | 46 | L1992 “<code>provider_id:provider-id</code>” |
| `Safe Boundary Evidence` | `provider_version` | “string[1..128]” | 59 | L1993 “<code>provider_version:string[1..128]</code>” |
| `Session Event` | `created_at` | “created_at” | 61 | L1729 “<code>created_at</code>, and <code>payload</code>” |
| `Session Event` | `created_by_host_id` | “created_by_host_id” | 44 | L1724 “<code>created_by_host_id</code>, <code>lease_epoch</code>” |
| `Session Event` | `event_id` | “event_id” | 57 | L1722 “Required fields are <code>event_id</code> digest” |
| `Session Event` | `event_type` | “event_type” | 3 | L1724 “<code>event_type</code>, <code>created_by_host_id</code>” |
| `Session Event` | `extensions` | “extensions” | 291 | L1734 “and <code>extensions</code>; no other top-level member is permitted” |
| `Session Event` | `lease_epoch` | “lease_epoch” | 37 | L1725 “<code>lease_epoch</code>, <code>lease_id</code>” |
| `Session Event` | `lease_id` | “lease_id” | 53 | L1725 “<code>lease_id</code>, and <code>lease_sequence</code>” |
| `Session Event` | `payload` | “closed tagged union” | 5 | L1764 “The <code>payload</code> object is a closed tagged union selected by <code>event_type</code>” |
| `Session Event` | `schema` | “urn:ax:schema:session-event” | 3 | L1733 “The exact top-level shape also requires <code>schema</code>, <code>schema_version</code>, and <code>extensions</code>” |
| `Session Event` | `schema_version` | “schema_version” | 99 | L1734 “<code>schema_version</code>, and <code>extensions</code>; no other top-level member is permitted” |
| `Session Event` | `session_id` | “session_id” | 187 | L1723 “<code>session_id</code> with the same UUID” |
| `Session Event` | `subject_id` | “subject_id” | 66 | L1722 “<code>subject_id</code> and <code>session_id</code> with the same UUID” |
| `Session Record Fork Provenance` | `operation_id` | “UUIDv7” | 480 | L1535 “<code>operation_id</code> UUIDv7” |
| `Session Record Fork Provenance` | `source_checkpoint_id` | “digest” | 976 | L1534 “<code>source_checkpoint_id</code> digest” |
| `Session Record Fork Provenance` | `source_session_id` | “UUIDv7” | 480 | L1534 “<code>source_session_id</code> UUIDv7” |
| `Session Record Fork Provenance` | `source_workspace_group_id` | “UUIDv7” | 480 | L1535 “<code>source_workspace_group_id</code> UUIDv7” |
| `Session Record cross-environment-clone provenance` | `bundle_id` | “UUIDv7” | 480 | L1658 “<code>bundle_id:UUIDv7</code>” |
| `Session Record cross-environment-clone provenance` | `canonical_session_id` | “digest” | 976 | L1669 “<code>canonical_session_id:digest</code>” |
| `Session Record cross-environment-clone provenance` | `capture_manifest_id` | “digest” | 976 | L1668 “<code>capture_manifest_id:digest</code>” |
| `Session Record cross-environment-clone provenance` | `migration_checkpoint_id` | “digest” | 976 | L1671 “<code>migration_checkpoint_id:digest</code>” |
| `Session Record cross-environment-clone provenance` | `operation_id` | “UUIDv7” | 480 | L1657 “<code>operation_id:UUIDv7</code>” |
| `Session Record cross-environment-clone provenance` | `previous_lineage_receipt_id` | “digest or null” | 28 | L1672 “<code>previous_lineage_receipt_id:digest\&#124;null</code>” |
| `Session Record cross-environment-clone provenance` | `projection_plan_id` | “digest” | 976 | L1670 “<code>projection_plan_id:digest</code>” |
| `Session Record cross-environment-clone provenance` | `source_environment` | “EnvironmentTuple” | 33 | L1665 “<code>source_environment:EnvironmentTuple</code>” |
| `Session Record cross-environment-clone provenance` | `source_profile_event_id` | “digest or null” | 28 | L1673 “<code>source_profile_event_id:digest\&#124;null</code>” |
| `Session Record cross-environment-clone provenance` | `source_snapshot_digest` | “digest” | 976 | L1667 “<code>source_snapshot_digest:digest</code>” |
| `Session Record cross-environment-clone provenance` | `target_environment` | “EnvironmentTuple” | 33 | L1666 “<code>target_environment:EnvironmentTuple</code>” |
| `Session Record native-adoption provenance` | `operation_id` | “UUIDv7” | 480 | L1691 “<code>operation_id:UUIDv7</code>” |
| `Session Record native-adoption provenance` | `source_environment` | “EnvironmentTuple” | 33 | L1695 “<code>source_environment:EnvironmentTuple</code>” |
| `Session Record native-adoption provenance` | `source_head_digest` | “digest” | 976 | L1694 “<code>source_head_digest:digest</code>” |
| `Session Record native-adoption provenance` | `source_host_id` | “UUIDv7” | 480 | L1692 “<code>source_host_id:UUIDv7</code>” |
| `Session Record native-adoption provenance` | `source_instance_id` | “digest” | 976 | L1692 “<code>source_instance_id:digest</code>” |
| `Session Record native-adoption provenance` | `source_observation_id` | “digest” | 976 | L1693 “<code>source_observation_id:digest</code>” |
| `Session Record origin provenance` | `creation_operation_id` | “UUIDv7” | 480 | L1646 “<code>creation_operation_id:UUIDv7</code>” |
| `Session Record same-provider-fork provenance` | `operation_id` | “UUIDv7” | 480 | L1652 “<code>operation_id:UUIDv7</code>” |
| `Session Record same-provider-fork provenance` | `source_checkpoint_id` | “digest” | 976 | L1650 “<code>source_checkpoint_id:digest</code>” |
| `Session Record same-provider-fork provenance` | `source_profile_event_id` | “digest or null” | 28 | L1654 “<code>source_profile_event_id:digest\&#124;null</code>” |
| `Session Record same-provider-fork provenance` | `source_workspace_group_id` | “UUIDv7” | 480 | L1651 “<code>source_workspace_group_id:UUIDv7</code>” |
| `Transfer Manifest` | `base_checkpoint_id` | “digest or null” | 28 | L4695 “\&#124; <code>base_checkpoint_id</code> \&#124; digest or null \&#124; Null only for an initial capture with no predecessor checkpoint \&#124;” |
| `Transfer Manifest` | `child_manifest_ids` | “sorted unique digest[0..1024]” | 8 | L4697 “\&#124; <code>child_manifest_ids</code> \&#124; sorted unique digest[0..1024] \&#124; Path-disjoint child/partition closure \&#124;” |
| `Transfer Manifest` | `created_at` | “timestamp” | 117 | L4703 “\&#124; <code>created_at</code> \&#124; timestamp \&#124; Diagnostic only \&#124;” |
| `Transfer Manifest` | `created_by_host_id` | “UUIDv7” | 480 | L4702 “\&#124; <code>created_by_host_id</code> \&#124; UUIDv7 \&#124; Capturing host \&#124;” |
| `Transfer Manifest` | `excluded_classes` | “sorted unique string[0..128]” | 13 | L4701 “\&#124; <code>excluded_classes</code> \&#124; sorted unique string[0..128] \&#124; Applied exclusion-policy classes \&#124;” |
| `Workspace Group Record` | `created_at` | “created_at:timestamp” | 10 | L2144 “<code>created_at:timestamp</code>” |
| `Workspace Group Record` | `display_name` | “display_name:string[1..128]” | 2 | L2142 “<code>display_name:string[1..128]</code>” |
| `Workspace Group Record` | `extensions` | “extensions:object” | 2 | L2145 “<code>extensions:object</code>” |
| `Workspace Group Record` | `record_id` | “record_id:digest” | 32 | L2140 “<code>record_id:digest</code>” |
| `Workspace Group Record` | `schema` | “urn:ax:schema:workspace-group” | 3 | L2138 “Its closed top-level shape contains exactly <code>schema</code>, <code>schema_version</code>” |
| `Workspace Group Record` | `schema_version` | “1.0.0” | 336 | L2139 “<code>schema_version</code>, <code>record_id:digest</code>” |
| `Workspace Group Record` | `subject_id` | “subject_id:UUIDv7” | 9 | L2140 “<code>subject_id:UUIDv7</code>” |
| `Workspace Group Record` | `workspace_group_id` | “workspace_group_id:UUIDv7” | 10 | L2141 “<code>workspace_group_id:UUIDv7</code>” |
| `WorkspaceMember.git` | `agent_project_config_paths` | “agent_project_config_paths:sorted unique path[0..256]” | 5 | L2162 “<code>agent_project_config_paths:sorted unique path[0..256]</code>” |
| `WorkspaceMember.git` | `group_relative_path` | “group_relative_path:path” | 5 | L2162 “<code>group_relative_path:path</code>” |
| `WorkspaceMember.git` | `kind` | “kind:git” | 2 | L2162 “<code>kind = git</code>” |
| `WorkspaceMember.git` | `materialization_policy` | “materialization_policy:shared_checkout&#124;separate_worktree” | 2 | L2162 “<code>materialization_policy:shared_checkout&#124;separate_worktree</code>” |
| `WorkspaceMember.git` | `repo_relative_cwd` | “repo_relative_cwd:.&#124;path” | 5 | L2162 “<code>repo_relative_cwd:.&#124;path</code>” |
| `WorkspaceMember.git` | `repository_identity` | “repository_identity:string[1..256]” | 4 | L2162 “<code>repository_identity:string[1..256]</code>” |
| `WorkspaceMember.git` | `workspace_id` | “workspace_id:UUIDv7” | 13 | L2162 “<code>workspace_id:UUIDv7</code>” |
| `WorkspaceMember.managed_tree` | `agent_project_config_paths` | “agent_project_config_paths:sorted unique path[0..256]” | 5 | L2163 “<code>agent_project_config_paths:sorted unique path[0..256]</code>” |
| `WorkspaceMember.managed_tree` | `group_relative_path` | “group_relative_path:path” | 5 | L2163 “<code>group_relative_path:path</code>” |
| `WorkspaceMember.managed_tree` | `kind` | “kind:managed_tree” | 2 | L2163 “<code>kind = managed_tree</code>” |
| `WorkspaceMember.managed_tree` | `materialization_policy` | “materialization_policy:shared_tree&#124;separate_copy” | 2 | L2163 “<code>materialization_policy:shared_tree&#124;separate_copy</code>” |
| `WorkspaceMember.managed_tree` | `repo_relative_cwd` | “repo_relative_cwd:.&#124;path” | 5 | L2163 “<code>repo_relative_cwd:.&#124;path</code>” |
| `WorkspaceMember.managed_tree` | `tree_identity` | “tree_identity:string[1..256]” | 2 | L2163 “<code>tree_identity:string[1..256]</code>” |
| `WorkspaceMember.managed_tree` | `workspace_id` | “workspace_id:UUIDv7” | 13 | L2163 “<code>workspace_id:UUIDv7</code>” |
| `WorkspaceSnapshotMember.git` | `agent_project_config_paths` | “sorted unique path[0..256]” | 5 | L4780 “<code>agent_project_config_paths:sorted unique path[0..256]</code>” |
| `WorkspaceSnapshotMember.git` | `features` | “GitFeatures” | 3 | L4780 “<code>features:GitFeatures</code>” |
| `WorkspaceSnapshotMember.git` | `group_relative_path` | “path” | 491 | L4780 “<code>group_relative_path:path</code>” |
| `WorkspaceSnapshotMember.git` | `head` | “GitHead” | 3 | L4780 “<code>head:GitHead</code>” |
| `WorkspaceSnapshotMember.git` | `index` | “GitIndex” | 7 | L4780 “<code>index:GitIndex</code>” |
| `WorkspaceSnapshotMember.git` | `object_pack` | “GitObjectPack” | 4 | L4780 “<code>object_pack:GitObjectPack</code>” |
| `WorkspaceSnapshotMember.git` | `repository_identity` | “string[1..256]” | 32 | L4780 “<code>repository_identity:string[1..256]</code>” |
| `WorkspaceSnapshotMember.git` | `submodules` | “GitSubmodule[0..256]” | 2 | L4780 “<code>submodules:GitSubmodule[0..256]</code>” |
| `WorkspaceSnapshotMember.git` | `working_tree_manifest_id` | “digest” | 976 | L4780 “<code>working_tree_manifest_id:digest</code>” |
| `WorkspaceSnapshotMember.git` | `workspace_id` | “UUIDv7” | 480 | L4780 “<code>workspace_id:UUIDv7</code>” |
| `WorkspaceSnapshotMember.managed_tree` | `agent_project_config_paths` | “sorted unique path[0..256]” | 5 | L4781 “<code>agent_project_config_paths:sorted unique path[0..256]</code>” |
| `WorkspaceSnapshotMember.managed_tree` | `group_relative_path` | “path” | 491 | L4781 “<code>group_relative_path:path</code>” |
| `WorkspaceSnapshotMember.managed_tree` | `tree_identity` | “string[1..256]” | 32 | L4781 “<code>tree_identity:string[1..256]</code>” |
| `WorkspaceSnapshotMember.managed_tree` | `tree_manifest_id` | “digest” | 976 | L4781 “<code>tree_manifest_id:digest</code>” |
| `WorkspaceSnapshotMember.managed_tree` | `workspace_id` | “UUIDv7” | 480 | L4781 “<code>workspace_id:UUIDv7</code>” |

## Class C — quote present at exactly one line (37 rows)

These were the only rows the pre-fix contract could have had right, and it had
no way to say so. They were rewritten to declare that line explicitly.

| Shape | Member | Pre-fix cell | Resolved line | Replacement |
| --- | --- | --- | ---: | --- |
| `Checkpoint Record` | `created_by_host_id` | “Current lease holder” | 1986 | L1986 “\&#124; <code>created_by_host_id</code> \&#124; UUIDv7 \&#124; Current lease holder \&#124;” |
| `Checkpoint Record` | `lease_epoch` | “Greater than zero and equal to the referenced winning lease” | 1979 | L1979 “\&#124; <code>lease_epoch</code> \&#124; uint53 \&#124; Greater than zero and equal to the referenced winning lease \&#124;” |
| `Checkpoint Record` | `provider_manifest_id` | “Direct native-store/provider snapshot only” | 1984 | L1984 “\&#124; <code>provider_manifest_id</code> \&#124; digest or null \&#124; Direct native-store/provider snapshot only \&#124;” |
| `Checkpoint Record` | `schema` | “Exact Checkpoint schema identifier” | 1974 | L1974 “\&#124; <code>schema</code> \&#124; string \&#124; Exact Checkpoint schema identifier \&#124;” |
| `Checkpoint Record` | `session_id` | “Existing Session Record” | 1978 | L1978 “\&#124; <code>session_id</code> \&#124; UUIDv7 \&#124; Existing Session Record \&#124;” |
| `Checkpoint Record` | `task_board_bundle_id` | “Task-board path only” | 1985 | L1985 “\&#124; <code>task_board_bundle_id</code> \&#124; digest or null \&#124; Task-board path only \&#124;” |
| `Checkpoint Record` | `workspace_manifest_id` | “Workspace-group Transfer Manifest root” | 1983 | L1983 “\&#124; <code>workspace_manifest_id</code> \&#124; digest \&#124; Workspace-group Transfer Manifest root \&#124;” |
| `EnvironmentTuple` | `environment_id` | “[a-z][a-z0-9.-]{0,63}” | 3609 | L3626 “Environment Tuple contains exactly <code>environment_id</code>”; L3609 “\&#124; <code>environment_id</code> \&#124; <code>[a-z][a-z0-9.-]{0,63}</code>; one semantic native environment \&#124;” |
| `EnvironmentTuple` | `store_schema_fingerprint` | “store_schema_fingerprint” | 3630 | L3630 “<code>store_schema_fingerprint</code>, and <code>adapter_version</code>” |
| `GitIndexEntry` | `stage` | “uint8[0..3]” | 4791 | L4791 “<code>stage:uint8[0..3]</code>” |
| `Lease Record` | `checkpoint_id` | “Null only for epoch-1 <code>create</code>; otherwise the validated materialized handoff base” | 1909 | L1909 “\&#124; <code>checkpoint_id</code> \&#124; digest or null \&#124; Null only for epoch-1 <code>create</code>; otherwise the validated materialized handoff base \&#124;” |
| `Lease Record` | `epoch` | “Starts at 1; never decreases” | 1905 | L1905 “\&#124; <code>epoch</code> \&#124; uint53 \&#124; Starts at 1; never decreases \&#124;” |
| `Lease Record` | `holder_host_id` | “Proposed owner” | 1906 | L1906 “\&#124; <code>holder_host_id</code> \&#124; UUIDv7 \&#124; Proposed owner \&#124;” |
| `Lease Record` | `issued_by_host_id` | “Initiator” | 1910 | L1910 “\&#124; <code>issued_by_host_id</code> \&#124; UUIDv7 \&#124; Initiator \&#124;” |
| `Lease Record` | `predecessor_lease_id` | “Null only at epoch 1”; “An epoch-1 <code>create</code> lease MUST have a null predecessor” | 1907 | L1907 “\&#124; <code>predecessor_lease_id</code> \&#124; UUIDv4 or null \&#124; Null only at epoch 1 \&#124;” |
| `Lease Record` | `reason` | “<code>create</code>, <code>graceful_takeover</code>, <code>force_takeover</code>, <code>recovery</code>” | 1908 | L1908 “\&#124; <code>reason</code> \&#124; enum \&#124; <code>create</code>, <code>graceful_takeover</code>, <code>force_takeover</code>, <code>recovery</code> \&#124;” |
| `Lease Record` | `record_id` | “Canonical Lease Record digest” | 1901 | L1901 “\&#124; <code>record_id</code> \&#124; digest \&#124; Canonical Lease Record digest \&#124;” |
| `Lease Record` | `schema` | “Exact Lease Record schema identifier” | 1899 | L1899 “\&#124; <code>schema</code> \&#124; string \&#124; Exact Lease Record schema identifier \&#124;” |
| `Lease Record` | `session_id` | “Lease scope” | 1904 | L1904 “\&#124; <code>session_id</code> \&#124; UUIDv7 \&#124; Lease scope \&#124;” |
| `ObservationCounts` | `chunks` | “chunks:uint53” | 11611 | L11611 “<code>chunks:uint53</code>” |
| `ObservationCounts` | `manifests` | “manifests:uint53” | 11610 | L11610 “<code>manifests:uint53</code>” |
| `ObservationCounts` | `records` | “records:uint53” | 11609 | L11609 “<code>records:uint53</code>” |
| `ObservationCounts` | `retries` | “retries:uint53” | 11612 | L11612 “<code>retries:uint53</code>” |
| `Provider Identity Record` | `created_by_host_id` | “Identifying owner host” | 2090 | L2090 “\&#124; <code>created_by_host_id</code> \&#124; UUIDv7 \&#124; Identifying owner host \&#124;” |
| `Provider Identity Record` | `opaque_identity` | “map(provider-identity-key,string[1..1024])[0..32]” | 2089 | L2089 “\&#124; <code>opaque_identity</code> \&#124; map(provider-identity-key,string[1..1024])[0..32] \&#124; Explicit adapter data map defined below \&#124;” |
| `Provider Identity Record` | `schema` | “Exact Provider Identity schema identifier” | 2077 | L2077 “\&#124; <code>schema</code> \&#124; string \&#124; Exact Provider Identity schema identifier \&#124;” |
| `Provider Identity Record` | `session_id` | “Existing logical session” | 2081 | L2081 “\&#124; <code>session_id</code> \&#124; UUIDv7 \&#124; Existing logical session \&#124;” |
| `Session Event` | `lease_sequence` | “uint53 starting at 1” | 1726 | L1726 “<code>lease_sequence</code> as a uint53 starting at 1 for each lease” |
| `Session Event` | `predecessors` | “sorted array of one or more record/event digests” | 1728 | L1728 “<code>predecessors</code> as a sorted array of one or more record/event digests” |
| `Session Record Board Goal` | `revision` | “uint53 greater than zero” | 1521 | L1521 “<code>revision</code> as uint53 greater than zero” |
| `Session Record cross-environment-clone provenance` | `extensions` | “reverse-DNS extensions” | 1643 | L1673 “<code>source_profile_event_id:digest\&#124;null</code>, and <code>extensions</code>” |
| `Session Record native-adoption provenance` | `extensions` | “reverse-DNS extensions” | 1643 | L1696 “<code>target_provider_id:provider-id</code>, and <code>extensions</code>” |
| `Session Record origin provenance` | `extensions` | “reverse-DNS extensions” | 1643 | L1646 “<code>creation_operation_id:UUIDv7</code>, and <code>extensions</code>” |
| `Session Record same-provider-fork provenance` | `extensions` | “reverse-DNS extensions” | 1643 | L1654 “<code>source_profile_event_id:digest\&#124;null</code>, and <code>extensions</code>” |
| `Workspace Group Record` | `created_by_host_id` | “created_by_host_id:UUIDv7” | 2144 | L2144 “<code>created_by_host_id:UUIDv7</code>” |
| `Workspace Group Record` | `members` | “members:WorkspaceMember[1..256]” | 2143 | L2143 “<code>members:WorkspaceMember[1..256]</code>” |
| `WorkspaceMember.git` | `sanitized_remote_urls` | “sanitized_remote_urls:sorted unique sanitized-git-URL[1..16]” | 2162 | L2162 “<code>sanitized_remote_urls:sorted unique sanitized-git-URL[1..16]</code>” |
