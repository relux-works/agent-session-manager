# Array order-constraint inventory

Normative source: `relux-works/agent-session-manager-spec` v0.5.0 at commit
`28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`, SPEC.md SHA-256
`562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a`.

Every row is one production comparison that decides the relative order of two
elements of one array, paired with the array member it orders. The `Source`,
`Function`, `Member` and `Enforces` columns are derived from the package
sources by `TestEveryArrayOrderConstraintMatchesItsPinnedPhrase`; nothing in
them is hand-written. `Enforces` is read off the comparison OPERATOR, which is
the only thing that decides whether a duplicate survives: refusing
`current <= previous` admits a strictly ascending array and therefore means
`sorted unique`, while refusing `current < previous` admits a non-descending
array and therefore means `sorted`.

This artifact exists because the specification uses a CONTROLLED VOCABULARY for
array constraints and the implementation read two of its phrases as synonyms.
Section 1.6 defines the compound phrase `sorted unique T[n..m]` as "such an
array with bytewise canonical ordering and no duplicate". The document applies
that phrase systematically - `event_heads`, `manifest_ids`, `evidence_ids`,
`object_ids`, `sanitized_remote_urls`, `agent_project_config_paths` - and uses
bare `sorted` where it declares ordering alone. Session Event `predecessors`
and Workspace Group Record `members` were both validated with the strict form,
so each refused a duplicate the pinned contract admits. Mapping phrase to
validator mechanically is what makes that class impossible rather than merely
absent: a strengthened validator cannot be written down consistently here,
because its row would have to cite a uniqueness declaration the document does
not contain.

Ordering and uniqueness are recorded as SEPARATE factors because the document
frequently declares them at different lines. Transfer Manifest `entries` are
declared "Sorted bytewise by normalized path" at one line and "MUST contain no
duplicate, overlapping, or destination-case-colliding path" at another; a row
whose `Enforces` is `sorted unique` must cite a uniqueness declaration, and a
row whose `Enforces` is `sorted` must cite none.

Rows are compared as an ordered sequence in production source order, so no
production line number appears here and an unrelated edit above a call site
does not churn the artifact.

`Member` is derived by tracing the ranged collection back to the string literal
that named it, through helper parameters and across call sites, so a reusable
ordering helper contributes one row per member it actually orders in
production. Adding a call site adds a row.

`Declared on` is the token the pinned document uses for the ordered array, and
it must appear as a whole token in both quoted declarations. It equals `Member`
except where the document states the rule on the ELEMENT TYPE instead of the
member - one row, `remotes`, whose ordering is declared on `GitRemote`. That
exception needs no list: a member name is lower_snake in this document and a
type name is CamelCase, so the checker admits a differing `Declared on` only
when it is capitalised.

The quoted declarations cannot be verified against SPEC.md here: this
repository does not vendor the pinned document, and no gate in this package can
open it. What is checked is internal consistency - the phrase each quote uses,
that it names the array it claims to declare, and that two rows quoting the
same declaration cite the same line.

Three ordering sites enforce uniqueness the pinned document does not declare.
They are recorded as `strengthens` rather than silently admitted, the set is
asserted exactly by `TestDeclaredOrderStrengtheningsAreExactlyTheDisclosedSet`,
and each names the leaf that owns the section in which it sits. They are listed
in `Strengthening` below rather than in a passing row.

| Source | Function | Member | Declared on | Enforces | Order SPEC line | Pinned order declaration | Unique SPEC line | Pinned uniqueness declaration |
| --- | --- | --- | --- | --- | ---: | --- | ---: | --- |
| `closed_shapes.go` | `validateSessionLaunchPlan` | `env_names` | `env_names` | sorted unique | 1490 | `env_names &#124; array&lt;string&gt;[0..64] &#124; Sorted, unique names matching [A-Za-z_][A-Za-z0-9_]{0,127}` | 1490 | `env_names &#124; array&lt;string&gt;[0..64] &#124; Sorted, unique names matching [A-Za-z_][A-Za-z0-9_]{0,127}` |
| `closed_shapes.go` | `validateManifestEntries` | `entries` | `entries` | sorted unique | 4696 | `entries &#124; ManifestEntry[0..65536] &#124; Sorted bytewise by normalized path` | 4769 | `Entries and child partitions MUST contain no duplicate, overlapping, or destination-case-colliding path.` |
| `closed_shapes.go` | `validateWorkspaceSnapshot` | `members` | `members` | sorted unique | 4774 | `its members correspond one-for-one, in workspace-ID order, with the Workspace Group Record` | strengthens | `strengthens` |
| `closed_shapes.go` | `validateGitRemotes` | `remotes` | `GitRemote` | sorted unique | 4787 | `GitRemote &#124; name:string[1..128], fetch_url:sanitized-git-URL, push_url:sanitized-git-URL&#124;null; sorted by name, no duplicate` | 4787 | `GitRemote &#124; name:string[1..128], fetch_url:sanitized-git-URL, push_url:sanitized-git-URL&#124;null; sorted by name, no duplicate` |
| `closed_shapes.go` | `validateGitIndex` | `entries` | `entries` | sorted | 4790 | `entries sorted by path then stage, count equal to length` | - | - |
| `closed_shapes.go` | `validateGitIndex` | `entries` | `entries` | sorted unique | 4790 | `entries sorted by path then stage, count equal to length` | strengthens | `strengthens` |
| `closed_shapes.go` | `requireSortedUniquePaths` | `agent_project_config_paths` | `agent_project_config_paths` | sorted unique | 2162 | `agent_project_config_paths:sorted unique path[0..256]` | 2162 | `agent_project_config_paths:sorted unique path[0..256]` |
| `closed_shapes.go` | `validateSortedDigests` | `predecessors` | `predecessors` | sorted | 1728 | `predecessors as a sorted array of one or more record/event digests` | - | - |
| `closed_shapes.go` | `validateSortedUniqueDigests` | `child_manifest_ids` | `child_manifest_ids` | sorted unique | 4697 | `child_manifest_ids &#124; sorted unique digest[0..1024] &#124; Path-disjoint child/partition closure` | 4697 | `child_manifest_ids &#124; sorted unique digest[0..1024]` |
| `closed_shapes.go` | `validateSortedUniqueDigests` | `event_heads` | `event_heads` | sorted unique | 1982 | `event_heads &#124; sorted unique digest[1..64] &#124; Authoritative event DAG heads immediately before this object` | 1982 | `event_heads &#124; sorted unique digest[1..64]` |
| `closed_shapes.go` | `validateSortedUniqueDigests` | `evidence_ids` | `evidence_ids` | sorted unique | 1871 | `evidence_ids:sorted unique digest[1..256]` | 1871 | `evidence_ids:sorted unique digest[1..256]` |
| `closed_shapes.go` | `validateSortedUniqueDigests` | `manifest_ids` | `manifest_ids` | sorted unique | 1777 | `manifest_ids:sorted unique digest[1..1024]` | 1777 | `manifest_ids:sorted unique digest[1..1024]` |
| `closed_shapes.go` | `validateSortedUniqueDigests` | `object_ids` | `object_ids` | sorted unique | 11604 | `object_ids &#124; sorted unique digest[0..4096] &#124; Redacted object identities only` | 11604 | `object_ids &#124; sorted unique digest[0..4096]` |
| `closed_shapes.go` | `validateSortedUniqueStrings` | `excluded_classes` | `excluded_classes` | sorted unique | 4701 | `excluded_classes &#124; sorted unique string[0..128] &#124; Applied exclusion-policy classes` | 4701 | `excluded_classes &#124; sorted unique string[0..128]` |
| `closed_shapes.go` | `validateSortedUniqueStrings` | `required_filter_names` | `required_filter_names` | sorted unique | 4793 | `required_filter_names:sorted unique string[0..64]` | 4793 | `required_filter_names:sorted unique string[0..64]` |
| `core_records.go` | `validateWorkspaceGroupRecord` | `members` | `members` | sorted | 2146 | `Members are sorted by workspace_id, and no two members may have an equal or case-colliding group_relative_path.` | - | - |
| `core_records.go` | `validateSortedUniqueSanitizedGitURLs` | `sanitized_remote_urls` | `sanitized_remote_urls` | sorted unique | 2162 | `sanitized_remote_urls:sorted unique sanitized-git-URL[1..16]` | 2162 | `sanitized_remote_urls:sorted unique sanitized-git-URL[1..16]` |

## Strengthening

Each row below enforces uniqueness the pinned document does not declare at any
line. None is in this leaf's sections; each is routed to the leaf that owns the
section it sits in, and each is reported rather than repaired here, because
changing it is a behaviour change in a reviewed and accepted candidate.

| Source | Function | Member | Undeclared refusal | Owning leaf |
| --- | --- | --- | --- | --- |
| `closed_shapes.go` | `validateWorkspaceSnapshot` | `members` | Two members with an equal `workspace_id` are refused. Section 10.1 declares only "in workspace-ID order"; the one-for-one correspondence it names is with the Workspace Group Record, whose own member ordering declares no `workspace_id` uniqueness either. | TASK-260830-uqnwmi |
| `closed_shapes.go` | `validateGitIndex` | `entries` | Two entries with an equal `path` AND an equal `stage` are refused. Section 18.1 declares only "entries sorted by path then stage". The path factor alone is correctly non-strict. | TASK-260830-uqnwmi |

`core_records.go` `validateWorkspaceGroupRecord.members` was a third instance of
the same shape and IS repaired here, because Section 2 is a section this leaf
owns: the declared uniqueness at line 2146 is on `group_relative_path`, which
the same loop enforces separately, and never on `workspace_id`.
