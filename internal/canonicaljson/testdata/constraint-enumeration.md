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
artifact row, so a new, removed, or renamed member fails the suite. **Type-only** means the
pinned SPEC gives that member no additional constraint beyond the stated JSON/common type;
that row quotes the bare type text verbatim. **Presence-only** means the pinned SPEC names a
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

## How the Pinned SPEC declaration column is checked

The `Pinned SPEC declaration` column is compared against the specification
itself, not against this implementation. The document compared against is the
byte-exact `SPEC.md` embedded in `internal/specdoc`, accepted only when its
SHA-256 equals `internal/specpin.DocumentSHA256`; a substituted, edited,
truncated, or unreadable document is refused rather than compared against, so a
swapped specification cannot confirm a row.

Each cell is one or more entries joined by `; `. An entry has one of two forms:

- `L<line>` followed by the excerpt in curly quotes quotes the pinned document
  verbatim. The quoted text MUST occur in the pinned document and MUST begin on
  that exact 1-based `SPEC.md` line.
- `L<line> paraphrase: <text>` deliberately restates the pinned declaration in
  this artifact's words instead of quoting it. It MUST still name the exact
  line it paraphrases, and the raw text of that line MUST contain the member
  name, so a paraphrase cannot float free of the document.

Every row MUST anchor its own member: at least one entry either quotes text
containing the member name or paraphrases a line whose raw text contains it. A
row whose quote is absent from the pinned document, begins on a different line,
or never names its member reddens
`TestConstraintEnumerationSpecExcerptsQuoteThePinnedSpecification`.

A citation that lands on a body row of a Markdown table MUST land on the row
that declares what it cites. Either the row's first cell names the member — the
per-member `Field` tables — or it names the identifier under which the pinned
document declares this shape, which `constraintRowDeclaringIdentifiers` pins per
shape and `TestEveryConstraintEnumerationDeclaringIdentifierIsExercised` asserts
exactly. The clause anchor cannot do this on its own: Section 10.4 declares
seven Git types one per table row, so `GitIndex.format` could quote
GitObjectPack's `format:git_pack_v2` — verbatim, at the cited line, in the right
clause, containing the member name — while production enforces `git_index`.
Seven shipped rows did exactly that and the suite stayed green.
`TestDeclaringRowAnchorRefusesEverySiblingRowOfTheGitTable` plants all fourteen
sibling retargets the table admits, in both directions, and requires each to be
refused by the sibling's name. The two exemptions are listed in
`constraintRowTableAnchorExemptions` with their reason and are asserted used.

Two formatting rules, and no others, apply before comparison:

1. **Whitespace.** Every run of ASCII whitespace in both the quote and the
   pinned document collapses to one space, and leading/trailing whitespace is
   dropped. Letter case, punctuation, digits, and inline `<code>` markup are
   compared exactly, so a quote cannot drift into unrelated text. A whitespace
   run that crosses a **hard boundary** collapses to an unmatchable block
   separator instead of a space. Two boundaries are hard: a **blank line**, so a
   quote cannot stitch the tail of one block to the head of the next — the end
   of a table to the paragraph after it, or a heading to its body — and the
   newline between two **adjacent table rows**, because a table row is a
   complete line by construction, so no honest excerpt spans two of them while a
   stitched one imports the next member's constraint. Whitespace collapsing
   alone used to admit both, and the halves of such a stitch are individually
   verbatim, so nothing else would have caught them. What the rule still
   forgives, deliberately, is the newline inside one block: the specification's
   hard line wrapping, table indentation, and the newline between two adjacent
   list items or two adjacent lines of one paragraph.
2. **Escaped pipe.** Inside a quote, the two characters `\|` stand for one
   literal `|` in the pinned document, because an unescaped `|` would end this
   artifact's own Markdown table cell. Nothing else is decoded: the pinned
   document's own `&#124;` entities are compared literally, exactly as it
   writes them.

Curly quotation marks in this artifact are reserved for verbatim pinned
specification text. `TestArtifactQuotesAreVerbatimPinnedSpecificationText`
requires every curly-quoted span in this file, inside the row table and outside
it, to occur in the pinned document.

### The clause anchor, and what the check still does not prove

A quote can be verbatim, correctly located, and about a different schema.
Retargeting `ManifestEntry.file.size` from its own Section 10.4 row to
BlobChunk's Section 10.2 `size:uint53[1..4194304]` — real text, real line, and a
bound `ManifestEntry.file` does not carry — satisfied every rule above.

So each shape also pins the numbered `SPEC.md` clauses its citations may come
from, and every entry's line must resolve to one of them. The clause of a line
is the nearest enclosing numbered heading; an unnumbered subheading does not
open a clause of its own. The pinned shape set is asserted exactly against this
table, so a new shape has to declare its clause rather than inherit a free pass.
`TestClauseAnchorRefusesEveryForeignSectionForOneRow` plants a verbatim line
from each of the eleven other clauses into one row and requires all eleven to be
refused, while the shipped row still passes.

Two shapes pin two clauses. `Session Record 1.0.0` and `Session Record 2.0.0 and
3.0.0` cite Section 2.1 as well as 5.1, because the document's own name-grammar
pointer leads to a section containing no grammar and the grammar is written in
the 2.1 Terms table. Quoting both is what keeps that indirection visible.

**The residual limit.** The anchor is a clause, not a shape. Ten shapes are
declared in Section 10.4 and two in 10.2, so a citation retargeted *within* a
clause — `ManifestEntry.file` quoting a `GitIndexEntry` row — is still admitted.
The member anchor is a substring test, so a member name embedded in another
schema's identifier satisfies it too. "Quotes this shape's clause" is what the
gate now proves; "quotes this shape's declaration" is not. The `environment_id`
finding recorded below sits inside that gap: both of its lines are in Section
7.8, and it was reached by reading the document, not by this gate.

| Shape | Member | Enforced constraint | Production call site | Pinned SPEC declaration |
| --- | --- | --- | --- | --- |
| `Lease Record` | `schema` | Exact Lease schema identifier. | `validateLeaseRecord` | L1899 “\| <code>schema</code> \| string \| Exact Lease Record schema identifier \|” |
| `Lease Record` | `schema_version` | Exact version 1.0.0. | `validateLeaseRecord` | L1900 “\| <code>schema_version</code> \| semver \| Exact <code>1.0.0</code> \|” |
| `Lease Record` | `record_id` | Canonical digest self identity. | `validateLeaseRecord` | L1901 “\| <code>record_id</code> \| digest \| Canonical Lease Record digest \|” |
| `Lease Record` | `subject_id` | UUIDv7 equal to session ID. | `validateLeaseRecord` | L1902 “\| <code>subject_id</code> \| UUIDv7 \| Equal to <code>session_id</code> \|” |
| `Lease Record` | `lease_id` | UUIDv4 fencing token. | `validateLeaseRecord` | L1903 “\| <code>lease_id</code> \| UUIDv4 \| Cryptographically random unique fencing token \|” |
| `Lease Record` | `session_id` | UUIDv7 lease scope. | `validateLeaseRecord` | L1904 “\| <code>session_id</code> \| UUIDv7 \| Lease scope \|” |
| `Lease Record` | `epoch` | Positive uint53; the declared epoch-one and successor nullability rules are enforced, and no reason is inferred from the epoch. | `validateLeaseRecord` | L1905 “\| <code>epoch</code> \| uint53 \| Starts at 1; never decreases \|” |
| `Lease Record` | `holder_host_id` | UUIDv7 proposed owner. | `validateLeaseRecord` | L1906 “\| <code>holder_host_id</code> \| UUIDv7 \| Proposed owner \|” |
| `Lease Record` | `predecessor_lease_id` | UUIDv4 or null; non-null is required after epoch one, and an epoch-one `create` lease must carry null. | `validateLeaseRecord` | L1907 “\| <code>predecessor_lease_id</code> \| UUIDv4 or null \| Null only at epoch 1 \|” |
| `Lease Record` | `reason` | Closed four-member reason enum. Section 5.3 declares no coupling from the epoch to the reason, so no epoch-one-implies-`create` or `create`-implies-epoch-one rule is enforced. | `validateLeaseRecord` | L1908 “\| <code>reason</code> \| enum \| <code>create</code>, <code>graceful_takeover</code>, <code>force_takeover</code>, <code>recovery</code> \|” |
| `Lease Record` | `checkpoint_id` | Digest, and null only for an epoch-one `create` lease; every other epoch and reason combination requires a non-null checkpoint. | `validateLeaseRecord` | L1909 “\| <code>checkpoint_id</code> \| digest or null \| Null only for epoch-1 <code>create</code>; otherwise the validated materialized handoff base \|” |
| `Lease Record` | `issued_by_host_id` | UUIDv7 initiator. | `validateLeaseRecord` | L1910 “\| <code>issued_by_host_id</code> \| UUIDv7 \| Initiator \|” |
| `Lease Record` | `created_by_host_id` | UUIDv7 equal to issuer. | `validateLeaseRecord` | L1911 “\| <code>created_by_host_id</code> \| UUIDv7 \| MUST equal <code>issued_by_host_id</code> \|” |
| `Lease Record` | `created_at` | Timestamp, diagnostic only. | `validateLeaseRecord` | L1912 “\| <code>created_at</code> \| timestamp \| Diagnostic only \|” |
| `Lease Record` | `extensions` | Required reverse-DNS extension object. | `validateLeaseRecord` | L1913 “\| <code>extensions</code> \| object \| Reverse-DNS extension keys only \|” |
| `Checkpoint Record` | `schema` | Exact Checkpoint schema identifier. | `validateCheckpointRecord` | L1974 “\| <code>schema</code> \| string \| Exact Checkpoint schema identifier \|” |
| `Checkpoint Record` | `schema_version` | Exact version 1.0.0. | `validateCheckpointRecord` | L1975 “\| <code>schema_version</code> \| semver \| Exact <code>1.0.0</code> \|” |
| `Checkpoint Record` | `checkpoint_id` | Canonical digest self identity. | `validateCheckpointRecord` | L1976 “\| <code>checkpoint_id</code> \| digest \| Canonical object digest \|” |
| `Checkpoint Record` | `subject_id` | UUIDv7 equal to session ID. | `validateCheckpointRecord` | L1977 “\| <code>subject_id</code> \| UUIDv7 \| Equal to <code>session_id</code> \|” |
| `Checkpoint Record` | `session_id` | UUIDv7 existing Session Record reference. | `validateCheckpointRecord` | L1978 “\| <code>session_id</code> \| UUIDv7 \| Existing Session Record \|” |
| `Checkpoint Record` | `lease_epoch` | Positive uint53. | `validateCheckpointRecord` | L1979 “\| <code>lease_epoch</code> \| uint53 \| Greater than zero and equal to the referenced winning lease \|” |
| `Checkpoint Record` | `lease_id` | UUIDv4 fencing token. | `validateCheckpointRecord` | L1980 “\| <code>lease_id</code> \| UUIDv4 \| Equal to that lease's fencing token \|” |
| `Checkpoint Record` | `safe_boundary` | Required closed Safe Boundary Evidence. | `validateCheckpointRecord` | L1981 “\| <code>safe_boundary</code> \| Safe Boundary Evidence \| Closed shape below \|” |
| `Checkpoint Record` | `event_heads` | Sorted unique digest array with 1..64 entries. | `validateCheckpointRecord` | L1982 “\| <code>event_heads</code> \| sorted unique digest[1..64] \| Authoritative event DAG heads immediately before this object \|” |
| `Checkpoint Record` | `workspace_manifest_id` | Required digest. | `validateCheckpointRecord` | L1983 “\| <code>workspace_manifest_id</code> \| digest \| Workspace-group Transfer Manifest root \|” |
| `Checkpoint Record` | `provider_manifest_id` | Nullable digest; exactly one persistence reference is non-null. | `validateCheckpointRecord` | L1984 “\| <code>provider_manifest_id</code> \| digest or null \| Direct native-store/provider snapshot only \|” |
| `Checkpoint Record` | `task_board_bundle_id` | Nullable digest; exactly one persistence reference is non-null. | `validateCheckpointRecord` | L1985 “\| <code>task_board_bundle_id</code> \| digest or null \| Task-board path only \|” |
| `Checkpoint Record` | `created_by_host_id` | UUIDv7 current holder. | `validateCheckpointRecord` | L1986 “\| <code>created_by_host_id</code> \| UUIDv7 \| Current lease holder \|” |
| `Checkpoint Record` | `created_at` | Timestamp, diagnostic only. | `validateCheckpointRecord` | L1987 “\| <code>created_at</code> \| timestamp \| Diagnostic only \|” |
| `Checkpoint Record` | `status` | Exact validated literal. | `validateCheckpointRecord` | L1988 “\| <code>status</code> \| enum \| Literal <code>validated</code> \|” |
| `Checkpoint Record` | `extensions` | Required reverse-DNS extension object. | `validateCheckpointRecord` | L1989 “\| <code>extensions</code> \| object \| Reverse-DNS extension keys only \|” |
| `Safe Boundary Evidence` | `provider_id` | Provider ID grammar. | `validateSafeBoundaryEvidence` | L1992 “<code>provider_id:provider-id</code>” |
| `Safe Boundary Evidence` | `provider_version` | String of 1..128 characters. | `validateSafeBoundaryEvidence` | L1993 “<code>provider_version:string[1..128]</code>” |
| `Safe Boundary Evidence` | `evidence` | Closed five-member evidence enum. | `validateSafeBoundaryEvidence` | L1994 “<code>evidence:provider_api\|provider_event\|managed_pty\|task_board_bridge\|accepted_test</code>” |
| `Safe Boundary Evidence` | `input_blocked` | Boolean required true for publication. | `validateSafeBoundaryEvidence` | L1995 “<code>input_blocked:boolean</code>” |
| `Safe Boundary Evidence` | `foreground_idle` | Boolean required true for publication. | `validateSafeBoundaryEvidence` | L1995 “<code>foreground_idle:boolean</code>” |
| `Safe Boundary Evidence` | `background_idle` | Boolean required true for publication. | `validateSafeBoundaryEvidence` | L1996 “<code>background_idle:boolean</code>” |
| `Safe Boundary Evidence` | `open_processes` | uint53 required zero for publication. | `validateSafeBoundaryEvidence` | L1996 “<code>open_processes:uint53</code>” |
| `Safe Boundary Evidence` | `open_database_handles` | uint53 required zero for publication. | `validateSafeBoundaryEvidence` | L1997 “<code>open_database_handles:uint53</code>” |
| `Provider Identity Record` | `schema` | Exact Provider Identity schema identifier. | `validateProviderIdentityRecord` | L2077 “\| <code>schema</code> \| string \| Exact Provider Identity schema identifier \|” |
| `Provider Identity Record` | `schema_version` | Exact version 1.0.0. | `validateProviderIdentityRecord` | L2078 “\| <code>schema_version</code> \| semver \| Exact <code>1.0.0</code> \|” |
| `Provider Identity Record` | `record_id` | Canonical digest self identity. | `validateProviderIdentityRecord` | L2079 “\| <code>record_id</code> \| digest \| Canonical object digest \|” |
| `Provider Identity Record` | `subject_id` | UUIDv7 equal to session ID. | `validateProviderIdentityRecord` | L2080 “\| <code>subject_id</code> \| UUIDv7 \| Equal to <code>session_id</code> \|” |
| `Provider Identity Record` | `session_id` | UUIDv7 logical session. | `validateProviderIdentityRecord` | L2081 “\| <code>session_id</code> \| UUIDv7 \| Existing logical session \|” |
| `Provider Identity Record` | `provider_id` | Provider ID grammar. | `validateProviderIdentityRecord` | L2082 “\| <code>provider_id</code> \| provider-id \| Must equal the Session Record provider \|” |
| `Provider Identity Record` | `provider_version` | String of 1..128 characters. | `validateProviderIdentityRecord` | L2083 “\| <code>provider_version</code> \| string[1..128] \| Exact probed version \|” |
| `Provider Identity Record` | `provider_version_range` | String of 1..256 characters. | `validateProviderIdentityRecord` | L2084 “\| <code>provider_version_range</code> \| string[1..256] \| Adapter compatibility range used for this identity \|” |
| `Provider Identity Record` | `native_session_id` | String of 1..512 characters. | `validateProviderIdentityRecord` | L2085 “\| <code>native_session_id</code> \| string[1..512] \| Opaque provider handle; never interpreted by core \|” |
| `Provider Identity Record` | `identity_kind` | Closed five-member identity enum. | `validateProviderIdentityRecord` | L2086 “\| <code>identity_kind</code> \| enum \| <code>session_uuid</code>, <code>session_path_or_id</code>, <code>backend_conversation_uuid</code>, <code>task_board_managed</code>, or <code>provider_defined</code> \|” |
| `Provider Identity Record` | `logical_workspace_id` | UUIDv7 workspace reference. | `validateProviderIdentityRecord` | L2087 “\| <code>logical_workspace_id</code> \| UUIDv7 \| Member of the Session Record workspace group \|” |
| `Provider Identity Record` | `backend_realm_fingerprint` | Nullable digest; Antigravity backend conversation requires non-null. | `validateProviderIdentityRecord` | L2088 “\| <code>backend_realm_fingerprint</code> \| digest or null \| Non-secret fingerprint; non-null when backend/account realm is a resume precondition \|” |
| `Provider Identity Record` | `opaque_identity` | Closed provider-data map of 0..32 bounded string values. | `validateProviderIdentityRecord` | L2089 “\| <code>opaque_identity</code> \| map(provider-identity-key,string[1..1024])[0..32] \| Explicit adapter data map defined below \|” |
| `Provider Identity Record` | `created_by_host_id` | UUIDv7 identifying host. | `validateProviderIdentityRecord` | L2090 “\| <code>created_by_host_id</code> \| UUIDv7 \| Identifying owner host \|” |
| `Provider Identity Record` | `created_at` | Timestamp, diagnostic only. | `validateProviderIdentityRecord` | L2091 “\| <code>created_at</code> \| timestamp \| Diagnostic only \|” |
| `Provider Identity Record` | `extensions` | Required reverse-DNS extension object. | `validateProviderIdentityRecord` | L2092 “\| <code>extensions</code> \| object \| Reverse-DNS extension keys only \|” |
| `Workspace Group Record` | `schema` | Exact Workspace Group schema identifier. | `validateWorkspaceGroupRecord` | L2138 “Its closed top-level shape contains exactly <code>schema</code>, <code>schema_version</code>” |
| `Workspace Group Record` | `schema_version` | Exact version 1.0.0. | `validateWorkspaceGroupRecord` | L2139 “<code>schema_version</code>, <code>record_id:digest</code>” |
| `Workspace Group Record` | `record_id` | Canonical digest self identity. | `validateWorkspaceGroupRecord` | L2140 “<code>record_id:digest</code>” |
| `Workspace Group Record` | `subject_id` | UUIDv7 equal to workspace group ID. | `validateWorkspaceGroupRecord` | L2140 “<code>subject_id:UUIDv7</code>” |
| `Workspace Group Record` | `workspace_group_id` | UUIDv7 equal to subject ID. | `validateWorkspaceGroupRecord` | L2141 “<code>workspace_group_id:UUIDv7</code>” |
| `Workspace Group Record` | `display_name` | String of 1..128 characters. | `validateWorkspaceGroupRecord` | L2142 “<code>display_name:string[1..128]</code>” |
| `Workspace Group Record` | `members` | Closed WorkspaceMember array of 1..256 entries sorted by ID. | `validateWorkspaceGroupRecord` | L2143 “<code>members:WorkspaceMember[1..256]</code>” |
| `Workspace Group Record` | `created_by_host_id` | UUIDv7 creator. | `validateWorkspaceGroupRecord` | L2144 “<code>created_by_host_id:UUIDv7</code>” |
| `Workspace Group Record` | `created_at` | Timestamp. | `validateWorkspaceGroupRecord` | L2144 “<code>created_at:timestamp</code>” |
| `Workspace Group Record` | `extensions` | Required reverse-DNS extension object. | `validateWorkspaceGroupRecord` | L2145 “<code>extensions:object</code>” |
| `Session Event` | `schema` | Exact Session Event schema identifier. | `validateSessionEvent` | L1733 “The exact top-level shape also requires <code>schema</code>, <code>schema_version</code>, and <code>extensions</code>” |
| `Session Event` | `schema_version` | Exact selected version 1.0.0 through 4.0.0. | `validateSessionEvent` | L1734 “<code>schema_version</code>, and <code>extensions</code>; no other top-level member is permitted” |
| `Session Event` | `event_id` | Canonical digest self identity. | `validateSessionEvent` | L1722 “Required fields are <code>event_id</code> digest” |
| `Session Event` | `subject_id` | UUIDv7 equal to session ID. | `validateSessionEvent` | L1722 “<code>subject_id</code> and <code>session_id</code> with the same UUID” |
| `Session Event` | `session_id` | UUIDv7 equal to subject ID. | `validateSessionEvent` | L1723 “<code>session_id</code> with the same UUID” |
| `Session Event` | `event_type` | Version-selected catalog event name; v1 unknown types remain inert and retainable. | `validateSessionEvent` | L1724 “<code>event_type</code>, <code>created_by_host_id</code>” |
| `Session Event` | `created_by_host_id` | UUIDv7 author host. | `validateSessionEvent` | L1724 “<code>created_by_host_id</code>, <code>lease_epoch</code>” |
| `Session Event` | `lease_epoch` | Positive uint53. | `validateSessionEvent` | L1725 “<code>lease_epoch</code>, <code>lease_id</code>” |
| `Session Event` | `lease_id` | UUIDv4 winning lease token. | `validateSessionEvent` | L1725 “<code>lease_id</code>, and <code>lease_sequence</code>” |
| `Session Event` | `lease_sequence` | Positive uint53 starting at one. | `validateSessionEvent` | L1726 “<code>lease_sequence</code> as a uint53 starting at 1 for each lease” |
| `Session Event` | `predecessors` | Non-empty sorted digest array. | `validateSessionEvent` | L1728 “<code>predecessors</code> as a sorted array of one or more record/event digests” |
| `Session Event` | `created_at` | Timestamp. | `validateSessionEvent` | L1729 “<code>created_at</code>, and <code>payload</code>” |
| `Session Event` | `payload` | Closed version-selected tagged union for registered types. | `validateSessionEvent` | L1764 “The <code>payload</code> object is a closed tagged union selected by <code>event_type</code>” |
| `Session Event` | `extensions` | Required reverse-DNS extension object. | `validateSessionEvent` | L1734 “and <code>extensions</code>; no other top-level member is permitted” |
| `Blob Descriptor` | `blob_id` | Enforced exactly as declared before identity calculation or verification. | `validateBlobDescriptor` | L4617 “<code>blob_id:digest</code>” |
| `Blob Descriptor` | `chunks` | Enforced exactly as declared before identity calculation or verification. | `validateBlobDescriptor` | L4619 “<code>chunks:BlobChunk[0..32768]</code>” |
| `Blob Descriptor` | `descriptor_id` | Enforced exactly as declared before identity calculation or verification. | `validateBlobDescriptor` | L4617 “<code>descriptor_id:digest</code>” |
| `Blob Descriptor` | `media_type` | Enforced exactly as declared before identity calculation or verification. | `validateBlobDescriptor` | L4618 “<code>media_type:string[1..255]</code>” |
| `Blob Descriptor` | `schema` | Enforced exactly as declared before identity calculation or verification. | `validateBlobDescriptor` | L4615 “The descriptor is closed and contains exactly <code>schema</code>”; L4610 “Every transferred blob has a Blob Descriptor with schema <code>urn:ax:schema:blob</code>” |
| `Blob Descriptor` | `schema_version` | Enforced exactly as declared before identity calculation or verification. | `validateBlobDescriptor` | L4615 “contains exactly <code>schema</code>, <code>schema_version</code>”; L4611 “<code>urn:ax:schema:blob</code> version <code>1.0.0</code>” |
| `Blob Descriptor` | `size` | Enforced exactly as declared before identity calculation or verification. | `validateBlobDescriptor` | L4618 “<code>size:uint53</code>” |
| `BlobChunk` | `chunk_id` | Enforced exactly as declared before identity calculation or verification. | `validateBlobDescriptor` | L4622 “<code>chunk_id:digest</code>” |
| `BlobChunk` | `index` | Enforced exactly as declared before identity calculation or verification. | `validateBlobDescriptor` | L4621 “<code>index:uint32</code>” |
| `BlobChunk` | `offset` | Enforced exactly as declared before identity calculation or verification. | `validateBlobDescriptor` | L4621 “<code>offset:uint53</code>” |
| `BlobChunk` | `size` | Enforced exactly as declared before identity calculation or verification; `TestTrailingBlobChunkSizeMinimumReachesBothIdentityEntries` pins a trailing zero-size chunk at the exact refusal clause through both public entries. | `validateBlobDescriptor` | L4622 “<code>size:uint53[1..4194304]</code>” |
| `GitFeatures` | `case_sensitive` | Enforced exactly as declared before identity calculation or verification. | `validateGitFeatures` | L4793 “<code>case_sensitive:boolean</code>” |
| `GitFeatures` | `filemode` | Enforced exactly as declared before identity calculation or verification. | `validateGitFeatures` | L4793 “<code>filemode:boolean</code>” |
| `GitFeatures` | `lfs_required` | Enforced exactly as declared before identity calculation or verification. | `validateGitFeatures` | L4793 “<code>lfs_required:boolean</code>” |
| `GitFeatures` | `object_format` | Enforced exactly as declared before identity calculation or verification. | `validateGitFeatures` | L4793 “<code>object_format:sha1&#124;sha256</code>” |
| `GitFeatures` | `precompose_unicode` | Enforced exactly as declared before identity calculation or verification. | `validateGitFeatures` | L4793 “<code>precompose_unicode:boolean</code>” |
| `GitFeatures` | `required_filter_names` | Enforced exactly as declared before identity calculation or verification. | `validateGitFeatures` | L4793 “<code>required_filter_names:sorted unique string[0..64]</code>” |
| `GitFeatures` | `sparse_checkout` | Enforced exactly as declared before identity calculation or verification. | `validateGitFeatures` | L4793 “<code>sparse_checkout:boolean</code>” |
| `GitFeatures` | `sparse_patterns_blob_descriptor_id` | Enforced exactly as declared before identity calculation or verification. | `validateGitFeatures` | L4793 “<code>sparse_patterns_blob_descriptor_id:digest&#124;null</code>” |
| `GitFeatures` | `sparse_patterns_blob_id` | Enforced exactly as declared before identity calculation or verification. | `validateGitFeatures` | L4793 “<code>sparse_patterns_blob_id:digest&#124;null</code>” |
| `GitFeatures` | `symlinks` | Enforced exactly as declared before identity calculation or verification. | `validateGitFeatures` | L4793 “<code>symlinks:boolean</code>” |
| `GitHead` | `mode` | Enforced exactly as declared before identity calculation or verification. | `validateGitHead` | L4788 “<code>mode:branch&#124;detached&#124;unborn</code>” |
| `GitHead` | `oid` | Enforced exactly as declared before identity calculation or verification. | `validateGitHead` | L4788 “<code>oid:git-oid&#124;null</code>” |
| `GitHead` | `ref` | Enforced exactly as declared before identity calculation or verification. | `validateGitHead` | L4788 “<code>ref:git-ref&#124;null</code>” |
| `GitIndex` | `blob_descriptor_id` | Enforced exactly as declared before identity calculation or verification. | `validateGitIndex` | L4790 “<code>blob_descriptor_id:digest</code>” |
| `GitIndex` | `blob_id` | Enforced exactly as declared before identity calculation or verification. | `validateGitIndex` | L4790 “<code>blob_id:digest</code>” |
| `GitIndex` | `entries` | Production enforces 0..65536 and a direct boundary test pins accept-at-65536/refuse-at-65537. Public-entry acceptance at 65536 is not claimed: the required closed entries encode above 5,242,880 bytes and `prepareObjectIdentity` refuses the object first. | `validateGitIndex` | L4790 “<code>entries:GitIndexEntry[0..65536]</code>” |
| `GitIndex` | `entry_count` | Enforced exactly as declared before identity calculation or verification. | `validateGitIndex` | L4790 “<code>entry_count:uint53</code>” |
| `GitIndex` | `format` | Enforced exactly as declared before identity calculation or verification. | `validateGitIndex` | L4790 “<code>format:git_index</code>” |
| `GitIndex` | `version` | Enforced exactly as declared before identity calculation or verification. | `validateGitIndex` | L4790 “<code>version:2&#124;3&#124;4</code>” |
| `GitIndexEntry` | `assume_unchanged` | Enforced exactly as declared before identity calculation or verification. | `validateGitIndexEntry` | L4791 “<code>assume_unchanged:boolean</code>” |
| `GitIndexEntry` | `fsmonitor_valid` | Enforced exactly as declared before identity calculation or verification. | `validateGitIndexEntry` | L4791 “<code>fsmonitor_valid:boolean</code>” |
| `GitIndexEntry` | `intent_to_add` | Enforced exactly as declared before identity calculation or verification. | `validateGitIndexEntry` | L4791 “<code>intent_to_add:boolean</code>” |
| `GitIndexEntry` | `mode` | Enforced exactly as declared before identity calculation or verification. | `validateGitIndexEntry` | L4791 “<code>mode:uint32</code>” |
| `GitIndexEntry` | `oid` | Enforced exactly as declared before identity calculation or verification. | `validateGitIndexEntry` | L4791 “<code>oid:git-oid</code>” |
| `GitIndexEntry` | `path` | Enforced exactly as declared before identity calculation or verification. | `validateGitIndexEntry` | L4791 “<code>path:path</code>” |
| `GitIndexEntry` | `skip_worktree` | Enforced exactly as declared before identity calculation or verification. | `validateGitIndexEntry` | L4791 “<code>skip_worktree:boolean</code>” |
| `GitIndexEntry` | `stage` | Enforced exactly as declared before identity calculation or verification. | `validateGitIndexEntry` | L4791 “<code>stage:uint8[0..3]</code>” |
| `GitObjectPack` | `blob_descriptor_id` | Enforced exactly as declared before identity calculation or verification. | `validateGitObjectPack` | L4789 “<code>blob_descriptor_id:digest</code>” |
| `GitObjectPack` | `blob_id` | Enforced exactly as declared before identity calculation or verification. | `validateGitObjectPack` | L4789 “<code>blob_id:digest</code>” |
| `GitObjectPack` | `format` | Enforced exactly as declared before identity calculation or verification. | `validateGitObjectPack` | L4789 “<code>format:git_pack_v2</code>” |
| `GitObjectPack` | `inventory_blob_descriptor_id` | Enforced exactly as declared before identity calculation or verification. | `validateGitObjectPack` | L4789 “<code>inventory_blob_descriptor_id:digest</code>” |
| `GitObjectPack` | `inventory_blob_id` | Enforced exactly as declared before identity calculation or verification. | `validateGitObjectPack` | L4789 “<code>inventory_blob_id:digest</code>” |
| `GitObjectPack` | `object_count` | Enforced exactly as declared before identity calculation or verification. | `validateGitObjectPack` | L4789 “<code>object_count:uint53</code>” |
| `GitObjectPack` | `object_format` | Enforced exactly as declared before identity calculation or verification. | `validateGitObjectPack` | L4789 “<code>object_format:sha1&#124;sha256</code>” |
| `GitRemote` | `fetch_url` | Enforced exactly as declared before identity calculation or verification. | `validateGitRemote` | L4787 “<code>fetch_url:sanitized-git-URL</code>” |
| `GitRemote` | `name` | Enforced exactly as declared before identity calculation or verification. | `validateGitRemote` | L4787 “<code>name:string[1..128]</code>” |
| `GitRemote` | `push_url` | Enforced exactly as declared before identity calculation or verification. | `validateGitRemote` | L4787 “<code>push_url:sanitized-git-URL&#124;null</code>” |
| `GitSubmodule` | `agent_project_config_paths` | Enforced exactly as declared before identity calculation or verification. | `validateGitSubmodule` | L4792 “<code>agent_project_config_paths:sorted unique path[0..256]&#124;null</code>” |
| `GitSubmodule` | `features` | Enforced exactly as declared before identity calculation or verification. | `validateGitSubmodule` | L4792 “<code>features:GitFeatures&#124;null</code>” |
| `GitSubmodule` | `gitlink_oid` | Enforced exactly as declared before identity calculation or verification. | `validateGitSubmodule` | L4792 “<code>gitlink_oid:git-oid</code>” |
| `GitSubmodule` | `head` | Enforced exactly as declared before identity calculation or verification. | `validateGitSubmodule` | L4792 “<code>head:GitHead&#124;null</code>” |
| `GitSubmodule` | `index` | Enforced exactly as declared before identity calculation or verification. | `validateGitSubmodule` | L4792 “<code>index:GitIndex&#124;null</code>” |
| `GitSubmodule` | `initialized` | Enforced exactly as declared before identity calculation or verification. | `validateGitSubmodule` | L4792 “<code>initialized:boolean</code>” |
| `GitSubmodule` | `object_pack` | Enforced exactly as declared before identity calculation or verification. | `validateGitSubmodule` | L4792 “<code>object_pack:GitObjectPack&#124;null</code>” |
| `GitSubmodule` | `path` | Enforced exactly as declared before identity calculation or verification. | `validateGitSubmodule` | L4792 “<code>path:path</code>” |
| `GitSubmodule` | `repo_relative_cwd` | Enforced exactly as declared before identity calculation or verification. | `validateGitSubmodule` | L4792 “<code>repo_relative_cwd:.&#124;path&#124;null</code>” |
| `GitSubmodule` | `repository_identity` | Enforced exactly as declared before identity calculation or verification. | `validateGitSubmodule` | L4792 “<code>repository_identity:string[1..256]</code>” |
| `GitSubmodule` | `sanitized_url` | Enforced exactly as declared before identity calculation or verification. | `validateGitSubmodule` | L4792 “<code>sanitized_url:sanitized-git-URL</code>” |
| `GitSubmodule` | `submodules` | Enforced exactly as declared before identity calculation or verification. | `validateGitSubmodule` | L4792 “<code>submodules:GitSubmodule[0..256]&#124;null</code>” |
| `GitSubmodule` | `upstream_ref` | Enforced exactly as declared before identity calculation or verification. | `validateGitSubmodule` | L4792 “<code>upstream_ref:git-ref&#124;null</code>” |
| `GitSubmodule` | `working_tree_manifest_id` | Enforced exactly as declared before identity calculation or verification. | `validateGitSubmodule` | L4792 “<code>working_tree_manifest_id:digest&#124;null</code>” |
| `ManifestEntry.directory` | `mode` | Enforced exactly as declared before identity calculation or verification. | `validateManifestEntries` | L4745 “<code>mode:uint32[0..4095]</code>” |
| `ManifestEntry.directory` | `path` | Enforced exactly as declared before identity calculation or verification. | `validateManifestEntries` | L4745 “<code>path:path</code>” |
| `ManifestEntry.directory` | `type` | Enforced exactly as declared before identity calculation or verification. | `validateManifestEntries` | L4745 “<code>type = directory</code>” |
| `ManifestEntry.file` | `blob_descriptor_id` | Enforced exactly as declared before identity calculation or verification. | `validateManifestEntries` | L4746 “<code>blob_descriptor_id:digest</code>” |
| `ManifestEntry.file` | `blob_id` | Enforced exactly as declared before identity calculation or verification. | `validateManifestEntries` | L4746 “<code>blob_id:digest</code>” |
| `ManifestEntry.file` | `mode` | Enforced exactly as declared before identity calculation or verification. | `validateManifestEntries` | L4746 “<code>mode:uint32[0..4095]</code>” |
| `ManifestEntry.file` | `path` | Enforced exactly as declared before identity calculation or verification. | `validateManifestEntries` | L4746 “<code>path:path</code>” |
| `ManifestEntry.file` | `size` | Enforced exactly as declared before identity calculation or verification. | `validateManifestEntries` | L4746 “<code>size:uint53</code>” |
| `ManifestEntry.file` | `type` | Enforced exactly as declared before identity calculation or verification. | `validateManifestEntries` | L4746 “<code>type = file</code>” |
| `ManifestEntry.hardlink` | `mode` | Enforced exactly as declared before identity calculation or verification. | `validateManifestEntries` | L4748 “<code>mode:uint32[0..4095]</code>” |
| `ManifestEntry.hardlink` | `path` | Enforced exactly as declared before identity calculation or verification. | `validateManifestEntries` | L4748 “<code>path:path</code>” |
| `ManifestEntry.hardlink` | `target_path` | Enforced exactly as declared before identity calculation or verification. | `validateManifestEntries` | L4748 “<code>target_path:path</code>” |
| `ManifestEntry.hardlink` | `type` | Enforced exactly as declared before identity calculation or verification. | `validateManifestEntries` | L4748 “<code>type = hardlink</code>” |
| `ManifestEntry.symlink` | `mode` | Enforced exactly as declared before identity calculation or verification. | `validateManifestEntries` | L4747 “<code>mode:uint32[0..4095]</code>” |
| `ManifestEntry.symlink` | `path` | Enforced exactly as declared before identity calculation or verification. | `validateManifestEntries` | L4747 “<code>path:path</code>” |
| `ManifestEntry.symlink` | `target` | Enforced exactly as declared before identity calculation or verification; `TestSymlinkTargetLowerBoundReachesBothIdentityEntries` pins accept-at-one and refuse-below-one through both public entries. | `validateManifestEntries` | L4747 “<code>target:string[1..4096]</code>” |
| `ManifestEntry.symlink` | `type` | Enforced exactly as declared before identity calculation or verification. | `validateManifestEntries` | L4747 “<code>type = symlink</code>” |
| `MigrationProvenance` | `object_id` | Enforced exactly as declared before identity calculation or verification. | `validateMigrationExtensionObject` | L11461 “<code>object_id:digest</code>” |
| `MigrationProvenance` | `schema_id` | Type-only: requires a valid UTF-8 JSON string and deliberately adds no non-empty or length rule. | `validateMigrationExtensionObject` | L11460 “<code>schema_id:string</code>” |
| `MigrationProvenance` | `schema_version` | Enforced exactly as declared before identity calculation or verification. | `validateMigrationExtensionObject` | L11461 “<code>schema_version:semver</code>” |
| `Session Record 1.0.0` | `created_at` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | L1473 “\| <code>created_at</code> \| timestamp \| Diagnostic time \|” |
| `Session Record 1.0.0` | `created_by_host_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | L1474 “\| <code>created_by_host_id</code> \| UUIDv7 \| Allowlisted host at creation \|” |
| `Session Record 1.0.0` | `execution_profile` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | L1477 “\| <code>execution_profile</code> \| enum \| <code>standard</code> or <code>yolo</code> \|” |
| `Session Record 1.0.0` | `extensions` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | L1481 “\| <code>extensions</code> \| object \| Required; may be empty; reverse-DNS keys only \|” |
| `Session Record 1.0.0` | `fork_provenance` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | L1480 “\| <code>fork_provenance</code> \| Fork Provenance or null \| Required object exactly when this record was created by fork \|” |
| `Session Record 1.0.0` | `kind` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | L1472 “\| <code>kind</code> \| enum \| <code>direct</code> or <code>task_board</code> \|” |
| `Session Record 1.0.0` | `launch_plan` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | L1478 “\| <code>launch_plan</code> \| Launch Plan \| Closed shape below; sanitized and secret-free \|” |
| `Session Record 1.0.0` | `name` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | L1471 “\| <code>name</code> \| string \| Section 2.3 grammar \|”; L363 “\| Session name \| A mesh-unique human alias of 1–64 characters matching <code>[A-Za-z0-9][A-Za-z0-9._-]{0,63}</code>. \|” |
| `Session Record 1.0.0` | `provider_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | L1475 “\| <code>provider_id</code> \| string \| Lowercase plugin ID \|” |
| `Session Record 1.0.0` | `record_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | L1468 “\| <code>record_id</code> \| digest \| Canonical object digest \|” |
| `Session Record 1.0.0` | `schema` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | L1466 “\| <code>schema</code> \| string \| Exact schema identifier \|” |
| `Session Record 1.0.0` | `schema_version` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | L1467 “\| <code>schema_version</code> \| semver \| <code>1.0.0</code> \|” |
| `Session Record 1.0.0` | `session_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | L1470 “\| <code>session_id</code> \| UUIDv7 \| Globally unique \|” |
| `Session Record 1.0.0` | `subject_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | L1469 “\| <code>subject_id</code> \| UUIDv7 \| Equal to <code>session_id</code> \|” |
| `Session Record 1.0.0` | `task_board` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | L1479 “\| <code>task_board</code> \| Task-board Reference or null \| Required object exactly when <code>kind = task_board</code> \|” |
| `Session Record 1.0.0` | `workspace_group_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordV1` | L1476 “\| <code>workspace_group_id</code> \| UUIDv7 \| Required \|” |
| `Session Record 2.0.0 and 3.0.0` | `created_at` | Enforced through the common immutable Record Envelope before identity calculation or verification. | `validateSessionRecordWithDerivation` | L1473 “\| <code>created_at</code> \| timestamp \| Diagnostic time \|” |
| `Session Record 2.0.0 and 3.0.0` | `created_by_host_id` | Enforced through the common immutable Record Envelope before identity calculation or verification. | `validateSessionRecordWithDerivation` | L1474 “\| <code>created_by_host_id</code> \| UUIDv7 \| Allowlisted host at creation \|” |
| `Session Record 2.0.0 and 3.0.0` | `derivation_provenance` | Required closed provenance union; v2 admits three tags and v3 admits those three plus native adoption. | `validateSessionRecordWithDerivation` | L1623 “It retains every major-1 field except <code>fork_provenance</code>, which is replaced by required closed <code>derivation_provenance</code>” |
| `Session Record 2.0.0 and 3.0.0` | `execution_profile` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordWithDerivation` | L1477 “\| <code>execution_profile</code> \| enum \| <code>standard</code> or <code>yolo</code> \|” |
| `Session Record 2.0.0 and 3.0.0` | `extensions` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordWithDerivation` | L1481 “\| <code>extensions</code> \| object \| Required; may be empty; reverse-DNS keys only \|” |
| `Session Record 2.0.0 and 3.0.0` | `kind` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordWithDerivation` | L1472 “\| <code>kind</code> \| enum \| <code>direct</code> or <code>task_board</code> \|” |
| `Session Record 2.0.0 and 3.0.0` | `launch_plan` | Enforced by the shared immutable Session Record creation shape. | `validateSessionRecordWithDerivation` | L1478 “\| <code>launch_plan</code> \| Launch Plan \| Closed shape below; sanitized and secret-free \|” |
| `Session Record 2.0.0 and 3.0.0` | `name` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordWithDerivation` | L1471 “\| <code>name</code> \| string \| Section 2.3 grammar \|”; L363 “\| Session name \| A mesh-unique human alias of 1–64 characters matching <code>[A-Za-z0-9][A-Za-z0-9._-]{0,63}</code>. \|” |
| `Session Record 2.0.0 and 3.0.0` | `provider_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordWithDerivation` | L1475 “\| <code>provider_id</code> \| string \| Lowercase plugin ID \|”; L1676 “The new target Session ID and target <code>provider_id</code> are allocated at creation and never reuse or mutate the source Session or source provider ID.” |
| `Session Record 2.0.0 and 3.0.0` | `record_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordWithDerivation` | L1468 “\| <code>record_id</code> \| digest \| Canonical object digest \|” |
| `Session Record 2.0.0 and 3.0.0` | `schema` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordWithDerivation` | L1466 “\| <code>schema</code> \| string \| Exact schema identifier \|” |
| `Session Record 2.0.0 and 3.0.0` | `schema_version` | Enforced as exact 2.0.0 or exact 3.0.0 at the version-selected validator. | `validateSessionRecordWithDerivation` | L1467 “\| <code>schema_version</code> \| semver \| <code>1.0.0</code> \|”; L1622 “Session Record 2.0.0 is emitted in v0.3.0 only for a cross-environment clone target.”; L1685 “Session Record 3.0.0 is the v0.4 creation contract.” |
| `Session Record 2.0.0 and 3.0.0` | `session_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordWithDerivation` | L1470 “\| <code>session_id</code> \| UUIDv7 \| Globally unique \|”; L1676 “The new target Session ID and target <code>provider_id</code> are allocated at creation” |
| `Session Record 2.0.0 and 3.0.0` | `subject_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordWithDerivation` | L1469 “\| <code>subject_id</code> \| UUIDv7 \| Equal to <code>session_id</code> \|” |
| `Session Record 2.0.0 and 3.0.0` | `task_board` | Enforced by the shared immutable Session Record creation shape. | `validateSessionRecordWithDerivation` | L1479 “\| <code>task_board</code> \| Task-board Reference or null \| Required object exactly when <code>kind = task_board</code> \|”; L1681 “Task-board references remain orthogonal authority in the existing <code>task_board</code> field” |
| `Session Record 2.0.0 and 3.0.0` | `workspace_group_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionRecordWithDerivation` | L1476 “\| <code>workspace_group_id</code> \| UUIDv7 \| Required \|” |
| `Session Record origin provenance` | `creation_operation_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionOriginProvenance` | L1646 “<code>creation_operation_id:UUIDv7</code>” |
| `Session Record origin provenance` | `extensions` | Enforced exactly as declared before identity calculation or verification. | `validateSessionOriginProvenance` | L1646 “<code>creation_operation_id:UUIDv7</code>, and <code>extensions</code>” |
| `Session Record origin provenance` | `kind` | Enforced exactly as declared before identity calculation or verification; the per-variant exact-string check is defensively redundant with the `validateSessionDerivationProvenance` switch dispatch that exclusively selects this validator. | `validateSessionOriginProvenance` | L1646 “<code>kind=origin</code>” |
| `Session Record same-provider-fork provenance` | `extensions` | Enforced exactly as declared before identity calculation or verification. | `validateSessionSameProviderForkProvenance` | L1654 “<code>source_profile_event_id:digest\|null</code>, and <code>extensions</code>” |
| `Session Record same-provider-fork provenance` | `kind` | Enforced exactly as declared before identity calculation or verification; the per-variant exact-string check is defensively redundant with the `validateSessionDerivationProvenance` switch dispatch that exclusively selects this validator. | `validateSessionSameProviderForkProvenance` | L1648 “<code>kind=same_provider_fork</code>” |
| `Session Record same-provider-fork provenance` | `operation_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionSameProviderForkProvenance` | L1652 “<code>operation_id:UUIDv7</code>” |
| `Session Record same-provider-fork provenance` | `provider_fork_mode` | Enforced exactly as declared before identity calculation or verification. | `validateSessionSameProviderForkProvenance` | L1653 “<code>provider_fork_mode:native\|supported_import\|task_board_clone</code>” |
| `Session Record same-provider-fork provenance` | `source_checkpoint_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionSameProviderForkProvenance` | L1650 “<code>source_checkpoint_id:digest</code>” |
| `Session Record same-provider-fork provenance` | `source_profile_event_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionSameProviderForkProvenance` | L1654 “<code>source_profile_event_id:digest\|null</code>” |
| `Session Record same-provider-fork provenance` | `source_session_id` | Enforced as UUIDv7 distinct from the new target Session ID. | `validateSessionSameProviderForkProvenance` | L1649 “<code>source_session_id:UUIDv7</code>” |
| `Session Record same-provider-fork provenance` | `source_workspace_group_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionSameProviderForkProvenance` | L1651 “<code>source_workspace_group_id:UUIDv7</code>” |
| `Session Record cross-environment-clone provenance` | `bundle_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionCrossEnvironmentCloneProvenance` | L1658 “<code>bundle_id:UUIDv7</code>” |
| `Session Record cross-environment-clone provenance` | `canonical_session_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionCrossEnvironmentCloneProvenance` | L1669 “<code>canonical_session_id:digest</code>” |
| `Session Record cross-environment-clone provenance` | `capture_manifest_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionCrossEnvironmentCloneProvenance` | L1668 “<code>capture_manifest_id:digest</code>” |
| `Session Record cross-environment-clone provenance` | `extensions` | Enforced exactly as declared before identity calculation or verification. | `validateSessionCrossEnvironmentCloneProvenance` | L1673 “<code>source_profile_event_id:digest\|null</code>, and <code>extensions</code>” |
| `Session Record cross-environment-clone provenance` | `kind` | Enforced exactly as declared before identity calculation or verification; the per-variant exact-string check is defensively redundant with the `validateSessionDerivationProvenance` switch dispatch that exclusively selects this validator. | `validateSessionCrossEnvironmentCloneProvenance` | L1656 “The <code>cross_environment_clone</code> variant's exact typed members are <code>kind</code>” |
| `Session Record cross-environment-clone provenance` | `migration_checkpoint_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionCrossEnvironmentCloneProvenance` | L1671 “<code>migration_checkpoint_id:digest</code>” |
| `Session Record cross-environment-clone provenance` | `operation_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionCrossEnvironmentCloneProvenance` | L1657 “<code>operation_id:UUIDv7</code>” |
| `Session Record cross-environment-clone provenance` | `previous_lineage_receipt_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionCrossEnvironmentCloneProvenance` | L1672 “<code>previous_lineage_receipt_id:digest\|null</code>” |
| `Session Record cross-environment-clone provenance` | `projection_plan_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionCrossEnvironmentCloneProvenance` | L1670 “<code>projection_plan_id:digest</code>” |
| `Session Record cross-environment-clone provenance` | `source_checkpoint_id` | Enforced as digest or null with the four-way AX-source nullability rule. | `validateSessionCrossEnvironmentCloneProvenance` | L1662 “<code>source_checkpoint_id:digest\|null</code>” |
| `Session Record cross-environment-clone provenance` | `source_environment` | Enforced as an exact closed EnvironmentTuple. | `validateSessionCrossEnvironmentCloneProvenance` | L1665 “<code>source_environment:EnvironmentTuple</code>” |
| `Session Record cross-environment-clone provenance` | `source_kind` | Enforced exactly as declared before identity calculation or verification. | `validateSessionCrossEnvironmentCloneProvenance` | L1659 “<code>source_kind:ax_session\|external_native</code>” |
| `Session Record cross-environment-clone provenance` | `source_native_session_id` | Enforced as 1–512 printable non-control Unicode characters, matching the pinned sanitization requirement; accept-at and refuse-past boundaries drive both public entries. The minimum-one check is subsumed by `requireString`, which refuses the empty string before `requirePrintableBoundedString` counts characters. | `validateSessionCrossEnvironmentCloneProvenance` | L1664 “<code>source_native_session_id:string[1..512]</code>” |
| `Session Record cross-environment-clone provenance` | `source_profile_event_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionCrossEnvironmentCloneProvenance` | L1673 “<code>source_profile_event_id:digest\|null</code>” |
| `Session Record cross-environment-clone provenance` | `source_provider_identity_record_id` | Enforced as digest or null with the four-way AX-source nullability rule. | `validateSessionCrossEnvironmentCloneProvenance` | L1663 “<code>source_provider_identity_record_id:digest\|null</code>” |
| `Session Record cross-environment-clone provenance` | `source_session_id` | Enforced as UUIDv7 or null, distinct from target, with the four-way AX-source nullability rule. | `validateSessionCrossEnvironmentCloneProvenance` | L1660 “<code>source_session_id:UUIDv7\|null</code>” |
| `Session Record cross-environment-clone provenance` | `source_session_record_id` | Enforced as digest or null with the four-way AX-source nullability rule. | `validateSessionCrossEnvironmentCloneProvenance` | L1661 “<code>source_session_record_id:digest\|null</code>” |
| `Session Record cross-environment-clone provenance` | `source_snapshot_digest` | Enforced exactly as declared before identity calculation or verification. | `validateSessionCrossEnvironmentCloneProvenance` | L1667 “<code>source_snapshot_digest:digest</code>” |
| `Session Record cross-environment-clone provenance` | `target_environment` | Enforced as an exact closed EnvironmentTuple. | `validateSessionCrossEnvironmentCloneProvenance` | L1666 “<code>target_environment:EnvironmentTuple</code>” |
| `Session Record native-adoption provenance` | `extensions` | Enforced exactly as declared before identity calculation or verification. | `validateSessionNativeAdoptionProvenance` | L1696 “<code>target_provider_id:provider-id</code>, and <code>extensions</code>” |
| `Session Record native-adoption provenance` | `kind` | Enforced exactly as declared before identity calculation or verification; the per-variant exact-string check is defensively redundant with the `validateSessionDerivationProvenance` switch dispatch that exclusively selects this validator. | `validateSessionNativeAdoptionProvenance` | L1691 “<code>kind=native_adoption</code>” |
| `Session Record native-adoption provenance` | `operation_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionNativeAdoptionProvenance` | L1691 “<code>operation_id:UUIDv7</code>” |
| `Session Record native-adoption provenance` | `source_environment` | Enforced as an exact closed EnvironmentTuple. | `validateSessionNativeAdoptionProvenance` | L1695 “<code>source_environment:EnvironmentTuple</code>” |
| `Session Record native-adoption provenance` | `source_head_digest` | Enforced exactly as declared before identity calculation or verification. | `validateSessionNativeAdoptionProvenance` | L1694 “<code>source_head_digest:digest</code>” |
| `Session Record native-adoption provenance` | `source_host_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionNativeAdoptionProvenance` | L1692 “<code>source_host_id:UUIDv7</code>” |
| `Session Record native-adoption provenance` | `source_instance_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionNativeAdoptionProvenance` | L1692 “<code>source_instance_id:digest</code>” |
| `Session Record native-adoption provenance` | `source_observation_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionNativeAdoptionProvenance` | L1693 “<code>source_observation_id:digest</code>” |
| `Session Record native-adoption provenance` | `target_provider_id` | Enforced equal to the immutable target Session Record provider. Provider-ID grammar is subsumed by `validateSessionRecordCommon`, which rejects a malformed record `provider_id` before this equality gate; a different malformed target is refused by the equality gate. | `validateSessionNativeAdoptionProvenance` | L1696 “<code>target_provider_id:provider-id</code>” |
| `EnvironmentTuple` | `adapter_version` | Presence-only, by the recorded SemVer decision below. The pinned EnvironmentTuple declaration supplies no JSON type and no format; the SemVer word belongs to the Session Adapter Manifest row of a different schema and is not inferred across schemas by field-name similarity. | `validateEnvironmentTuple` | L3630 “and <code>adapter_version</code>; it never contains executable provenance” |
| `EnvironmentTuple` | `architecture` | Enforced exactly as declared before identity calculation or verification. | `validateEnvironmentTuple` | L3629 “<code>architecture=amd64\|arm64</code>” |
| `EnvironmentTuple` | `environment_id` | Enforced against the exact environment ID grammar. The pinned EnvironmentTuple clause names the member and gives it no type or format; the only pinned statement of that grammar is the Session Adapter Manifest row of a different schema, quoted beside it. Whether the grammar is inferable across those two schemas is an open question recorded against this row, not a settled reading. | `validateEnvironmentTuple` | L3626 “Environment Tuple contains exactly <code>environment_id</code>”; L3609 “\| <code>environment_id</code> \| <code>[a-z][a-z0-9.-]{0,63}</code>; one semantic native environment \|” |
| `EnvironmentTuple` | `environment_version` | Presence-only. The pinned EnvironmentTuple declaration supplies no JSON type or bound; the `string[1..128]` bound belongs to the distinct Environment Observation schema and is not inferred here. | `validateEnvironmentTuple` | L3627 “<code>environment_id</code>, <code>environment_version</code>” |
| `EnvironmentTuple` | `platform` | Enforced against the complete generated AX platform scalar vocabulary. | `validateEnvironmentTuple` | L3628 “<code>platform=linux\|macos\|windows\|wsl2</code>” |
| `EnvironmentTuple` | `store_schema_fingerprint` | Presence-only. The pinned EnvironmentTuple declaration supplies no JSON type or format; in particular, identity validation does not infer a digest from the member name. | `validateEnvironmentTuple` | L3630 “<code>store_schema_fingerprint</code>, and <code>adapter_version</code>” |
| `Session Record Board Goal` | `extensions` | Enforced exactly as declared before identity calculation or verification. | `validateSessionBoardGoal` | L1521 “greater than zero, and <code>extensions</code>” |
| `Session Record Board Goal` | `goal_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionBoardGoal` | L1520 “<code>goal_id</code> as a 1–128 character public goal reference” |
| `Session Record Board Goal` | `revision` | Enforced exactly as declared before identity calculation or verification. | `validateSessionBoardGoal` | L1521 “<code>revision</code> as uint53 greater than zero” |
| `Session Record Board Goal` | `schema` | Enforced exactly as declared before identity calculation or verification. | `validateSessionBoardGoal` | L1520 “<code>schema = "board-goal-v2"</code>” |
| `Session Record Board Identity` | `extensions` | Enforced exactly as declared before identity calculation or verification. | `validateSessionBoardIdentity` | L1517 “and <code>extensions</code>. A local board requires null <code>remote_url</code>” |
| `Session Record Board Identity` | `kind` | Enforced exactly as declared before identity calculation or verification. | `validateSessionBoardIdentity` | L1514 “Board Identity has exactly <code>kind</code> (<code>local</code> or <code>remote</code>)” |
| `Session Record Board Identity` | `logical_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionBoardIdentity` | L1515 “<code>logical_id</code> (1–128 characters matching <code>[A-Za-z0-9][A-Za-z0-9._:-]{0,127}</code>)” |
| `Session Record Board Identity` | `remote_url` | Enforced exactly as declared before identity calculation or verification. | `validateSessionBoardIdentity` | L1517 “<code>remote_url</code> (absolute <code>https</code> URL or null)” |
| `Session Record Fork Provenance` | `extensions` | Enforced exactly as declared before identity calculation or verification. | `validateSessionForkProvenance` | L1537 “or <code>task_board_clone</code>, and <code>extensions</code>” |
| `Session Record Fork Provenance` | `operation_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionForkProvenance` | L1535 “<code>operation_id</code> UUIDv7” |
| `Session Record Fork Provenance` | `provider_fork_mode` | Enforced exactly as declared before identity calculation or verification. | `validateSessionForkProvenance` | L1536 “<code>provider_fork_mode</code> as <code>native</code>, <code>supported_import</code>, or <code>task_board_clone</code>” |
| `Session Record Fork Provenance` | `source_checkpoint_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionForkProvenance` | L1534 “<code>source_checkpoint_id</code> digest” |
| `Session Record Fork Provenance` | `source_session_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionForkProvenance` | L1534 “<code>source_session_id</code> UUIDv7” |
| `Session Record Fork Provenance` | `source_workspace_group_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionForkProvenance` | L1535 “<code>source_workspace_group_id</code> UUIDv7” |
| `Session Record Launch Plan` | `argv` | Enforced exactly as declared before identity calculation or verification. | `validateSessionLaunchPlan` | L1487 “\| <code>argv</code> \| array&lt;string&gt;[1..128] \| Each element is 1–4,096 UTF-8 bytes; total encoded argv is at most 65,536 bytes; never a shell command string \|” |
| `Session Record Launch Plan` | `contains_secrets` | Enforced exactly as declared before identity calculation or verification. | `validateSessionLaunchPlan` | L1492 “\| <code>contains_secrets</code> \| boolean \| MUST be false \|” |
| `Session Record Launch Plan` | `cwd_relative` | Enforced exactly as declared before identity calculation or verification. | `validateSessionLaunchPlan` | L1489 “\| <code>cwd_relative</code> \| string \| <code>.</code> for the workspace root or a path satisfying Section 1.6 \|” |
| `Session Record Launch Plan` | `cwd_workspace_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionLaunchPlan` | L1488 “\| <code>cwd_workspace_id</code> \| UUIDv7 \| Names one workspace in the Session Record's workspace group \|” |
| `Session Record Launch Plan` | `env_literals` | Enforced exactly as declared before identity calculation or verification. | `validateSessionLaunchPlan` | L1491 “\| <code>env_literals</code> \| map(environment-name,string)[0..64] \| Non-secret literals of at most 4,096 UTF-8 bytes each; keys sorted in canonical form and disjoint from <code>env_names</code> \|” |
| `Session Record Launch Plan` | `env_names` | Enforced exactly as declared before identity calculation or verification. | `validateSessionLaunchPlan` | L1490 “\| <code>env_names</code> \| array&lt;string&gt;[0..64] \| Sorted, unique names matching <code>[A-Za-z_][A-Za-z0-9_]{0,127}</code>; values resolve only from destination-local state \|” |
| `Session Record Launch Plan` | `extensions` | Enforced exactly as declared before identity calculation or verification. | `validateSessionLaunchPlan` | L1493 “\| <code>extensions</code> \| object \| Reverse-DNS extension keys only \|” |
| `Session Record Task-board Reference` | `board` | Enforced exactly as declared before identity calculation or verification. | `validateSessionTaskBoardReference` | L1506 “\| <code>board</code> \| Board Identity \| Closed shape below \|” |
| `Session Record Task-board Reference` | `board_goal` | Enforced exactly as declared before identity calculation or verification. | `validateSessionTaskBoardReference` | L1510 “\| <code>board_goal</code> \| Board Goal or null \| Required non-null for <code>primary_owner</code> \|” |
| `Session Record Task-board Reference` | `bridge_protocol_version` | Enforced exactly as declared before identity calculation or verification. | `validateSessionTaskBoardReference` | L1505 “\| <code>bridge_protocol_version</code> \| semver \| Exact <code>1.0.0</code> \|” |
| `Session Record Task-board Reference` | `extensions` | Enforced exactly as declared before identity calculation or verification. | `validateSessionTaskBoardReference` | L1512 “\| <code>extensions</code> \| object \| Reverse-DNS extension keys only \|” |
| `Session Record Task-board Reference` | `launch_mode` | Enforced exactly as declared before identity calculation or verification. | `validateSessionTaskBoardReference` | L1508 “\| <code>launch_mode</code> \| enum \| <code>primary_owner</code> or <code>tracked_prompt</code> \|” |
| `Session Record Task-board Reference` | `manager_session_ref` | Enforced exactly as declared before identity calculation or verification. | `validateSessionTaskBoardReference` | L1509 “\| <code>manager_session_ref</code> \| string or null \| MUST be null in the immutable creation record; the public reference is established by <code>task_board.launched</code> and may later change through <code>task_board.adopted</code> \|” |
| `Session Record Task-board Reference` | `native_goal_binding` | Enforced exactly as declared before identity calculation or verification. | `validateSessionTaskBoardReference` | L1511 “\| <code>native_goal_binding</code> \| enum \| <code>bound</code>, <code>prompt</code>, or <code>none</code> \|” |
| `Session Record Task-board Reference` | `task_element_id` | Enforced exactly as declared before identity calculation or verification. | `validateSessionTaskBoardReference` | L1507 “\| <code>task_element_id</code> \| string \| 1–128 printable non-control UTF-8 bytes \|” |
| `Transfer Manifest` | `base_checkpoint_id` | Enforced exactly as declared before identity calculation or verification. | `validateTransferManifest` | L4695 “\| <code>base_checkpoint_id</code> \| digest or null \| Null only for an initial capture with no predecessor checkpoint \|” |
| `Transfer Manifest` | `child_manifest_ids` | Enforced exactly as declared before identity calculation or verification. | `validateTransferManifest` | L4697 “\| <code>child_manifest_ids</code> \| sorted unique digest[0..1024] \| Path-disjoint child/partition closure \|” |
| `Transfer Manifest` | `created_at` | Enforced exactly as declared before identity calculation or verification. | `validateTransferManifest` | L4703 “\| <code>created_at</code> \| timestamp \| Diagnostic only \|” |
| `Transfer Manifest` | `created_by_host_id` | Enforced exactly as declared before identity calculation or verification. | `validateTransferManifest` | L4702 “\| <code>created_by_host_id</code> \| UUIDv7 \| Capturing host \|” |
| `Transfer Manifest` | `entries` | Enforced exactly as declared before identity calculation or verification. | `validateTransferManifest` | L4696 “\| <code>entries</code> \| ManifestEntry[0..65536] \| Sorted bytewise by normalized path \|” |
| `Transfer Manifest` | `excluded_classes` | Enforced exactly as declared before identity calculation or verification. | `validateTransferManifest` | L4701 “\| <code>excluded_classes</code> \| sorted unique string[0..128] \| Applied exclusion-policy classes \|” |
| `Transfer Manifest` | `extensions` | Enforced exactly as declared before identity calculation or verification. | `validateTransferManifest` | L4704 “\| <code>extensions</code> \| object \| Reverse-DNS extension keys only \|” |
| `Transfer Manifest` | `kind` | Enforced exactly as declared before identity calculation or verification. | `validateTransferManifest` | L4693 “\| <code>kind</code> \| enum \| <code>workspace_group</code>, <code>workspace_tree</code>, <code>provider</code>, <code>task_board</code>, or <code>composite</code> \|” |
| `Transfer Manifest` | `manifest_id` | Enforced exactly as declared before identity calculation or verification. | `validateTransferManifest` | L4692 “\| <code>manifest_id</code> \| digest \| Canonical object digest \|” |
| `Transfer Manifest` | `provider_identity_record_id` | Enforced exactly as declared before identity calculation or verification. | `validateTransferManifest` | L4699 “\| <code>provider_identity_record_id</code> \| digest or null \| Non-null only for <code>provider</code> \|” |
| `Transfer Manifest` | `schema` | Enforced exactly as declared before identity calculation or verification. | `validateTransferManifest` | L4690 “\| <code>schema</code> \| string \| Exact Transfer Manifest schema identifier \|” |
| `Transfer Manifest` | `schema_version` | Enforced exactly as declared before identity calculation or verification. | `validateTransferManifest` | L4691 “\| <code>schema_version</code> \| semver \| Exact <code>1.0.0</code> \|” |
| `Transfer Manifest` | `subject_id` | Enforced exactly as declared before identity calculation or verification. | `validateTransferManifest` | L4694 “\| <code>subject_id</code> \| UUIDv7 \| Group, workspace, or session selected by kind \|” |
| `Transfer Manifest` | `task_board_bundle_id` | Enforced exactly as declared before identity calculation or verification. | `validateTransferManifest` | L4700 “\| <code>task_board_bundle_id</code> \| digest or null \| Non-null only for <code>task_board</code> \|” |
| `Transfer Manifest` | `workspace_snapshot` | Enforced exactly as declared before identity calculation or verification. | `validateTransferManifest` | L4698 “\| <code>workspace_snapshot</code> \| WorkspaceSnapshot or null \| Non-null only for <code>workspace_group</code> \|” |
| `WorkspaceSnapshot` | `members` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshot` | L4773 “<code>members:WorkspaceSnapshotMember[1..256]</code>” |
| `WorkspaceSnapshot` | `workspace_group_id` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshot` | L4772 “<code>workspace_group_id:UUIDv7</code>” |
| `WorkspaceSnapshotMember.git` | `agent_project_config_paths` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | L4780 “<code>agent_project_config_paths:sorted unique path[0..256]</code>” |
| `WorkspaceSnapshotMember.git` | `features` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | L4780 “<code>features:GitFeatures</code>” |
| `WorkspaceSnapshotMember.git` | `group_relative_path` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | L4780 “<code>group_relative_path:path</code>” |
| `WorkspaceSnapshotMember.git` | `head` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | L4780 “<code>head:GitHead</code>” |
| `WorkspaceSnapshotMember.git` | `index` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | L4780 “<code>index:GitIndex</code>” |
| `WorkspaceSnapshotMember.git` | `kind` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | L4780 “<code>kind = git</code>” |
| `WorkspaceSnapshotMember.git` | `materialization_policy` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | L4780 “<code>materialization_policy:shared_checkout&#124;separate_worktree</code>” |
| `WorkspaceSnapshotMember.git` | `object_pack` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | L4780 “<code>object_pack:GitObjectPack</code>” |
| `WorkspaceSnapshotMember.git` | `remotes` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | L4780 “<code>remotes:GitRemote[1..16]</code>” |
| `WorkspaceSnapshotMember.git` | `repo_relative_cwd` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | L4780 “<code>repo_relative_cwd:.&#124;path</code>” |
| `WorkspaceSnapshotMember.git` | `repository_identity` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | L4780 “<code>repository_identity:string[1..256]</code>” |
| `WorkspaceSnapshotMember.git` | `submodules` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | L4780 “<code>submodules:GitSubmodule[0..256]</code>” |
| `WorkspaceSnapshotMember.git` | `upstream_ref` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | L4780 “<code>upstream_ref:git-ref&#124;null</code>” |
| `WorkspaceSnapshotMember.git` | `working_tree_manifest_id` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | L4780 “<code>working_tree_manifest_id:digest</code>” |
| `WorkspaceSnapshotMember.git` | `workspace_id` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | L4780 “<code>workspace_id:UUIDv7</code>” |
| `WorkspaceSnapshotMember.managed_tree` | `agent_project_config_paths` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | L4781 “<code>agent_project_config_paths:sorted unique path[0..256]</code>” |
| `WorkspaceSnapshotMember.managed_tree` | `group_relative_path` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | L4781 “<code>group_relative_path:path</code>” |
| `WorkspaceSnapshotMember.managed_tree` | `kind` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | L4781 “<code>kind = managed_tree</code>” |
| `WorkspaceSnapshotMember.managed_tree` | `materialization_policy` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | L4781 “<code>materialization_policy:shared_tree&#124;separate_copy</code>” |
| `WorkspaceSnapshotMember.managed_tree` | `repo_relative_cwd` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | L4781 “<code>repo_relative_cwd:.&#124;path</code>” |
| `WorkspaceSnapshotMember.managed_tree` | `tree_identity` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | L4781 “<code>tree_identity:string[1..256]</code>” |
| `WorkspaceSnapshotMember.managed_tree` | `tree_manifest_id` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | L4781 “<code>tree_manifest_id:digest</code>” |
| `WorkspaceSnapshotMember.managed_tree` | `workspace_id` | Enforced exactly as declared before identity calculation or verification. | `validateWorkspaceSnapshotMember` | L4781 “<code>workspace_id:UUIDv7</code>” |
| `WorkspaceMember.git` | `workspace_id` | UUIDv7. | `validateWorkspaceMember` | L2162 “<code>workspace_id:UUIDv7</code>” |
| `WorkspaceMember.managed_tree` | `workspace_id` | UUIDv7. | `validateWorkspaceMember` | L2163 “<code>workspace_id:UUIDv7</code>” |
| `WorkspaceMember.git` | `group_relative_path` | Relative path; absolute, parent-escaping, and non-canonical forms are refused. | `validateWorkspaceMember` | L2162 “<code>group_relative_path:path</code>” |
| `WorkspaceMember.managed_tree` | `group_relative_path` | Relative path; absolute, parent-escaping, and non-canonical forms are refused. | `validateWorkspaceMember` | L2163 “<code>group_relative_path:path</code>” |
| `WorkspaceMember.git` | `repo_relative_cwd` | Literal `.` or a relative path. | `validateWorkspaceMember` | L2162 “<code>repo_relative_cwd:.&#124;path</code>” |
| `WorkspaceMember.managed_tree` | `repo_relative_cwd` | Literal `.` or a relative path. | `validateWorkspaceMember` | L2163 “<code>repo_relative_cwd:.&#124;path</code>” |
| `WorkspaceMember.git` | `agent_project_config_paths` | Sorted unique relative paths, 0..256 entries. | `validateWorkspaceMember` | L2162 “<code>agent_project_config_paths:sorted unique path[0..256]</code>” |
| `WorkspaceMember.managed_tree` | `agent_project_config_paths` | Sorted unique relative paths, 0..256 entries. | `validateWorkspaceMember` | L2163 “<code>agent_project_config_paths:sorted unique path[0..256]</code>” |
| `WorkspaceMember.git` | `kind` | Exact tag selecting the git member set. | `validateWorkspaceMember` | L2162 “<code>kind = git</code>” |
| `WorkspaceMember.git` | `repository_identity` | 1..256 Unicode characters and refused when it is an absolute path. | `validateWorkspaceMember` | L2162 “<code>repository_identity:string[1..256]</code>” |
| `WorkspaceMember.git` | `sanitized_remote_urls` | Sorted unique sanitized Git URLs, 1..16 entries; password, token, query, fragment, and machine-local file forms are refused. | `validateWorkspaceMember` | L2162 “<code>sanitized_remote_urls:sorted unique sanitized-git-URL[1..16]</code>” |
| `WorkspaceMember.git` | `materialization_policy` | Enum shared_checkout or separate_worktree. | `validateWorkspaceMember` | L2162 “<code>materialization_policy:shared_checkout&#124;separate_worktree</code>” |
| `WorkspaceMember.managed_tree` | `kind` | Exact tag selecting the managed_tree member set. | `validateWorkspaceMember` | L2163 “<code>kind = managed_tree</code>” |
| `WorkspaceMember.managed_tree` | `tree_identity` | 1..256 Unicode characters and refused when it is an absolute path. | `validateWorkspaceMember` | L2163 “<code>tree_identity:string[1..256]</code>” |
| `WorkspaceMember.managed_tree` | `materialization_policy` | Enum shared_tree or separate_copy. | `validateWorkspaceMember` | L2163 “<code>materialization_policy:shared_tree&#124;separate_copy</code>” |
| `Observation Event` | `schema` | Exact Observation schema identifier. | `validateObservationEvent` | L11589 “\| <code>schema</code> \| string \| Exact Observation schema identifier \|” |
| `Observation Event` | `schema_version` | Exact version 1.0.0. | `validateObservationEvent` | L11590 “\| <code>schema_version</code> \| semver \| Exact <code>1.0.0</code> \|” |
| `Observation Event` | `stream_id` | UUIDv7. | `validateObservationEvent` | L11591 “\| <code>stream_id</code> \| UUIDv7 \| Stable per host installation; changing it starts a new explicitly separate stream \|” |
| `Observation Event` | `sequence` | uint53 greater than zero. | `validateObservationEvent` | L11592 “\| <code>sequence</code> \| uint53 \| Starts at 1 and increases by exactly one before each durable append \|” |
| `Observation Event` | `timestamp` | Canonical AX timestamp. | `validateObservationEvent` | L11593 “\| <code>timestamp</code> \| timestamp \| Observation time; not authority \|” |
| `Observation Event` | `level` | Enum debug, info, warn, or error. | `validateObservationEvent` | L11594 “\| <code>level</code> \| enum \| <code>debug</code>, <code>info</code>, <code>warn</code>, or <code>error</code> \|” |
| `Observation Event` | `event` | 3..128 Unicode characters matching the declared observation-name grammar. | `validateObservationEvent` | L11595 “\| <code>event</code> \| observation-name \| <code>[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*){1,7}</code>, 3–128 characters \|” |
| `Observation Event` | `operation_id` | UUIDv7 or null. | `validateObservationEvent` | L11596 “\| <code>operation_id</code> \| UUIDv7 or null \| Required null when no operation exists \|” |
| `Observation Event` | `session_id` | UUIDv7 or null. | `validateObservationEvent` | L11597 “\| <code>session_id</code> \| UUIDv7 or null \| Required null for non-session events \|” |
| `Observation Event` | `host_id` | UUIDv7. | `validateObservationEvent` | L11598 “\| <code>host_id</code> \| UUIDv7 \| Emitting host \|” |
| `Observation Event` | `peer_host_id` | UUIDv7 or null. | `validateObservationEvent` | L11599 “\| <code>peer_host_id</code> \| UUIDv7 or null \| Required null when no peer participates \|” |
| `Observation Event` | `phase` | 1..128 Unicode characters in lower_snake_case, or null. | `validateObservationEvent` | L11600 “\| <code>phase</code> \| string[1..128] or null \| Stable lower-snake-case phase or null \|” |
| `Observation Event` | `result` | Enum started, success, partial, failure, or cancelled. | `validateObservationEvent` | L11601 “\| <code>result</code> \| enum \| <code>started</code>, <code>success</code>, <code>partial</code>, <code>failure</code>, or <code>cancelled</code> \|” |
| `Observation Event` | `duration_ms` | uint53 or null; a started result requires null. | `validateObservationEvent` | L11602 “\| <code>duration_ms</code> \| uint53 or null \| Null for a point/start event; otherwise elapsed milliseconds \|” |
| `Observation Event` | `counts` | Closed ObservationCounts object or null. | `validateObservationEvent` | L11603 “\| <code>counts</code> \| ObservationCounts or null \| Closed aggregate below \|” |
| `Observation Event` | `object_ids` | Sorted unique digests, 0..4096 entries. | `validateObservationEvent` | L11604 “\| <code>object_ids</code> \| sorted unique digest[0..4096] \| Redacted object identities only \|” |
| `Observation Event` | `error_code` | 1..128 Unicode characters or null; non-null exactly for partial and failure. | `validateObservationEvent` | L11605 “\| <code>error_code</code> \| string[1..128] or null \| Stable Section 15 code when result is partial/failure \|” |
| `Observation Event` | `extensions` | Present object member. | `validateObservationEvent` | L11606 “\| <code>extensions</code> \| object \| Reverse-DNS extension keys only; no payload content \|” |
| `ObservationCounts` | `records` | uint53 within the safe-integer bound. | `validateObservationCountsMember` | L11609 “<code>records:uint53</code>” |
| `ObservationCounts` | `events` | uint53 within the safe-integer bound. | `validateObservationCountsMember` | L11609 “<code>events:uint53</code>” |
| `ObservationCounts` | `manifests` | uint53 within the safe-integer bound. | `validateObservationCountsMember` | L11610 “<code>manifests:uint53</code>” |
| `ObservationCounts` | `blobs` | uint53 within the safe-integer bound. | `validateObservationCountsMember` | L11610 “<code>blobs:uint53</code>” |
| `ObservationCounts` | `chunks` | uint53 within the safe-integer bound. | `validateObservationCountsMember` | L11611 “<code>chunks:uint53</code>” |
| `ObservationCounts` | `bytes` | uint53 within the safe-integer bound. | `validateObservationCountsMember` | L11611 “<code>bytes:uint53</code>” |
| `ObservationCounts` | `retries` | uint53 within the safe-integer bound. | `validateObservationCountsMember` | L11612 “<code>retries:uint53</code>” |

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
| Section 17.3 migration contribution | `works.relux.ax.migrated-from` is a closed three-member object; `schema_id` is bare `string`, `schema_version` is Semantic Versioning 2.0.0 in full including optional prerelease and build metadata, and `object_id` is a digest. | Enforced by `validateMigrationExtensionObject`; `TestMigrationProvenanceSchemaVersionIsSemVer200InFull` pins both the admitted prerelease/build forms and the forms still refused. Publication, atomic-reference advancement, rollback retention, and runtime migration remain explicitly unclaimed. |

## Recorded decision: where `semver` applies, and what it means

This package compiles one `semverPattern` and reaches it from more than one
schema. Two separate questions therefore have to be answered, and both answers
are recorded here rather than left implied by the code.

**Does the constraint apply at all at this member?** It applies exactly where
the pinned document declares it, and nowhere else. A type or format written on
a similarly named member of a different schema is not evidence about this one.

**Where it does apply, what does `semver` mean?** The pinned document names
Semantic Version and types members `semver` but spells out no grammar for one.
The named standard is Semantic Versioning 2.0.0, whose valid versions include
optional prerelease and build metadata. `semver` is therefore adopted as that
standard **in full**; a core-triple-only reading would refuse `1.2.3-rc.1`, a
version neither the standard nor the pinned document excludes.

Applying the two answers to the sites that reach the grammar:

| Site | Pinned declaration | Decision | Pinned by |
| --- | --- | --- | --- |
| EnvironmentTuple `adapter_version` | “Environment Tuple contains exactly <code>environment_id</code>, <code>environment_version</code>, <code>platform=linux\|macos\|windows\|wsl2</code>, <code>architecture=amd64\|arm64</code>, <code>store_schema_fingerprint</code>, and <code>adapter_version</code>” — no type, no format | **No constraint.** Presence-only, like its untyped `environment_version` and `store_schema_fingerprint` siblings in the same clause. `1.2.3-rc.1` is accepted. | `TestEnvironmentTupleAdapterVersionCarriesNoInferredSemVerConstraint` |
| Migration provenance `schema_version` | “That extension value is a closed object containing exactly <code>schema_id:string</code>, <code>schema_version:semver</code>, and <code>object_id:digest</code>.” | **SemVer 2.0.0 in full.** The type is declared here, so the constraint stands; prerelease and build metadata are accepted because the standard admits them. `1.2.3-rc.1` is accepted; `01.2.3`, `1.2`, `1.2.3-`, `1.2.3-01` and `1.2.3+` are still refused. | `TestMigrationProvenanceSchemaVersionIsSemVer200InFull` |
| Session Event `terminal.*` `implementation_version` / `protocol_version` | “<code>implementation_version:semver</code>”, “<code>protocol_version:semver</code>” | **SemVer 2.0.0 in full**, for the same reason as the row above. | `validateTerminalV4Payload`, through the shared `semverPattern` grammar inventory |

The word SemVer that used to be read onto `EnvironmentTuple.adapter_version`
comes from the Session Adapter Manifest row “<code>display_name</code> /
<code>adapter_version</code> | UTF-8 string[1..128] / SemVer”, which closes a
different object, and from the Probe sentence “Provider ID, manifest digest, and
adapter version equal the verified Manifest and host values”, which names the
Probe's own top-level members and not this nested tuple member. Neither reaches
the tuple, so neither authorises a constraint on it.

`semverPattern` itself is carried in the grammar inventory
(`grammar_inventory_test.go`) as an implementation-defined grammar with a
witness for every anchor, character class and one-or-more quantifier it
declares, so widening any one of its dimensions reddens the suite.
