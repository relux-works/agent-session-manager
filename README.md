# Agent Session Manager (`ax`)

Agent Session Manager is the implementation repository for the local-first,
cross-host `ax` coding-agent session orchestrator.

## Project Status

The Curator-managed repository foundation is complete and the published AX
v0.5.0 specification has been decomposed into an implementation board. The
Go contract foundation pins the immutable normative source and generates typed
contract, operation, capability-vocabulary, event, and error catalogs from
reviewed implementation metadata. Session, provider, terminal, mesh, and
operator command behavior is not implemented or advertised by this slice.

Product behavior is defined by the
[normative AX specification](https://github.com/relux-works/agent-session-manager-spec/blob/main/SPEC.md).
When this repository and the specification disagree, the specification is
authoritative.

## Bootstrap

The machine used for repository bootstrap runs Curator
`v0.14.0-rc.3-106-g665dd13f`, built from `relux-works/curator` main revision
`665dd13f7b6ab0408300070ef68d895f2d03b79f`.

From a fresh checkout, verify Curator, install the exact project-local skill
revisions and agent adapters pinned in `Skillfile.json`, and check the managed
state:

```bash
curator --version
curator install
curator status --check
```

Configure the required repository-local Git identity and SSH signing policy:

```bash
git config --local user.name "Ivan Oparin"
git config --local user.email "oparin@me.com"
git config --local gpg.format ssh
git config --local user.signingkey /Users/iv/.ssh/ivanopcode.pub
git config --local commit.gpgsign true
git config --local tag.gpgsign true
```

Contributor and agent workflow rules live in [AGENTS.md](AGENTS.md).
`CLAUDE.md` is a relative symlink to that canonical file.

## Implementation Plan

The live `task-board` contains five implementation milestones (`M0` through
`M4`) with 62 Stories and 186 atomic agent-executable Tasks. The dependency
closure of the final release Task contains every implementation Task, and the
machine-derived milestone critical path is:

```text
M0 contract foundation
  -> M1 single-host durability
  -> M2 multi-host preview
  -> M3 daily-driver tmux
  -> M4 cloning, Directory, platforms, and product release
```

All optional human UX, pilot, fidelity, and product go/no-go work is isolated
in a separate advisory Epic with no hard dependency into the implementation
DAG. It cannot block autonomous agent execution.

See [.spec/README.md](.spec/README.md) for the pinned specification source,
section coverage, board IDs, counts, and execution rules. Generated planning
snapshots live in [.planning/](.planning/); the live task-board remains the
planning authority.

## Normative Source Pin

[`internal/specpin`](internal/specpin) embeds one implementation lock for
`agent-session-manager-spec@v0.5.0`. Its production `Current` and `Verify`
entry points fail closed on a partial read, unknown member, source substitution,
contract or fixture drift, and any byte-level lock change. It also projects the
exact historical v0.4.3 registry without widening that immutable baseline.

This package is read-only and does not mutate durable state. Repeated reads are
idempotent and isolated from caller mutation. It deliberately adds no `ax`
command, `doctor` result, conformance-target declaration, or runtime capability
claim; those surfaces remain unavailable until their owning implementation and
acceptance tasks land.

## Common Wire Scalar Types

[`internal/scalar`](internal/scalar) implements the validated AX value types
used by later schema owners: canonical lowercase UUIDv4 and UUIDv7 values, real
UTC RFC3339 timestamps with millisecond-or-finer precision, SHA-256 digest
identifiers, the current four-member AX platform vocabulary, bounded provider
IDs, platform-neutral relative paths, platform-bound absolute paths, safe and
bounded JSON integers, `decimal_uint64`, and exact lower-snake-case closed
enums. Embedded Git values use the same production scalar layer for exact
SHA-1/SHA-256 OIDs, fully qualified `git check-ref-format` references, and
sanitized HTTPS/SSH/git remote URLs that contain no password or token, query,
fragment, local path, or unsupported scheme.

Timestamp validation accepts published UTC leap seconds, including
`1990-12-31T23:59:60.000Z`, while refusing a fabricated `:60` in an ordinary
minute or on an unpublished date. Because Go's `time.Time` has no leap-second
representation, `Timestamp.Time` maps an accepted leap second to the following
representable UTC second while the scalar retains the original wire text.

Constructors and JSON/text decoders fail closed on malformed, null,
out-of-range, wrong-version, traversal, encoded-separator, wrong-platform,
unknown-enum, or fabricated leap-second inputs. `AbsolutePath` decoding
requires the containing platform, and generic closed-enum decoding requires
the exact negotiated vocabulary; neither guesses that context from a field
name or value prefix. Native Windows paths additionally refuse Win32-reserved
component punctuation, control characters, and DOS device names even when a
device name has an extension; those Windows-only component rules do not alter
POSIX path validation. These types are read-only value validation. They do not
mutate durable state and add no `ax doctor` result, provider/platform
availability claim, or runtime capability advertisement.

`FuzzScalarProductionEntries` drives the production constructors and wire
decoders with a checked-in corpus covering published and fabricated leap
seconds, lowercase RFC3339, impossible dates, malformed identifiers, unsafe
integer boundaries, and Windows reserved-device and wildcard components. Any
accepted value must publish and read back byte-for-byte without a later
refusal.

Run the focused tests and assigned-scope gate with:

```bash
go test ./internal/scalar -count=1
go test ./internal/scalar -cover -count=1
go test ./internal/scalar -run=^$ \
  -fuzz=^FuzzScalarProductionEntries$ -fuzztime=100x -parallel=1
go run ./internal/traceability/cmd/tracecheck \
  -section 1.6 -section 10.1 -section 10.2 -section 10.3 -section 10.4
```

## Configuration Loading and Versioned Schemas

The internal/config package implements the read-only Section 3.2 path-selection
boundary used by Section 6.1 configuration loading. LoadOS captures the
caller-supplied runtime-probed AX platform and current OS inputs once; Load is
the dependency-injected production entry used by deterministic tests and
alternate launchers.

One five-row registry drives --config, --data-dir, --state-dir, --cache-dir,
and --runtime-dir together with their exact AX_* counterparts. Resolution
applies flag, non-empty documented environment override, then platform default
independently to every path class. Relative flag and override values are made
absolute against the captured working directory before use. XDG, macOS,
WSL2/Linux, and native Windows defaults use native path grammar; missing
required Linux runtime or Windows profile inputs fail closed.

A failed user-home lookup is deferred rather than refused at capture time,
because Windows derives no path default from the user home and several classes
are satisfied by an XDG override. The captured cause travels into the
platform-default refusal of every home-derived class, so an operator still sees
why the default was unavailable. Both halves are proven at the real LoadOS and
MigrateOS entries against a home the case makes and unmakes through the exact
environment variable os.UserHomeDir reads; no case assigns the captured value or
its error directly.

Load returns isolated TOML bytes, one immutable resolved-root snapshot, and an
isolated current-model Configuration when the selected file exists. It dispatches
strict closed-table readers for Configuration 1.0.0, 2.0.0, and 3.0.0 from the
pinned catalog; v1/v2 are translated in memory to v3 without changing their
source bytes. `EncodeCurrent` validates and emits only Configuration 3.0.0.
Backend-specific v3 settings require an exact caller-supplied settings validator;
without one, a `backend_config` entry fails closed rather than accepting an
arbitrary blob. Only an enabled external-trust entry registers its backend for
selection or backend-specific configuration; disabled entries remain
round-trippable but cannot authorize activation. An explicit empty
`transport_policy` is the valid deny-all subset, distinct from omission, whose
default is both closed transport names.

A not-yet-created configuration file is accepted only when its parent directory
exists, and ConfigPresent distinguishes that state from an existing empty
file. Versioned validation applies closed members, scalar and relational bounds,
unique/sorted registries, SSH host-authentication refusals, directory disclosure
constraints, and the v3 TerminalBackend trust/config shape from Sections 6 and
17. Required closed map members preserve the distinction between absence and an
explicit empty object in both inline (`extensions = {}`) and multiline table
syntax, and current-version writer output round-trips through the reader. Legacy
`terminal.backend` accepts exactly `tmux|conpty` before translation.
The v3 terminal capability vocabulary is derived from the pinned catalog's
`terminal_backend` family, and the supported reader set is tested as a derived
property of the pinned Configuration catalog, so neither a new capability nor a
new Configuration version can silently lack enforcement. Full package tests also
derive the production refusal-site inventory from source and fail when a new
configuration refusal has no exercised negative path. That derivation proves
its own coverage: it enumerates every error type the package declares, requires
exactly one instrumented constructor per type, and reports any refusal built
outside that constructor, so a second refusal form or a duplicated constructor
cannot report green. Sites excluded as subsumed are pinned by an explicit
inventory test and each names the check that covers it. An omitted
`required_capabilities` remains marked as contextual default provenance; this
package does not invent a platform lane minimum that only activation evidence
can establish.

`Migrate` and `MigrateOS` are the explicit Section 6.4/6.5 durable migration
entries; ordinary `Load` and startup never call them. Migration validates the
selected source first, requires the v1 generated-summary disclosure choice,
re-inspects the selected file's kind and permission bits before anything durable
is written, creates an owner-only versioned backup containing the exact old
bytes, writes
and fsyncs a same-directory temporary file, atomically replaces the source,
and fsyncs the directory. Exact backups are safely reusable on retry, a
pre-existing backup whose bytes differ from the source, whose mode is readable
beyond the owner, or that cannot be read at all is refused rather than treated
as satisfied, an already-target-version source is a no-op, and a post-replace
directory-sync failure restores the old bytes before returning an error. A
failure injected at each durable step leaves the selected file exactly as it
was, and a rollback that itself fails is reported as a recovery failure rather
than an ordinary sync failure. Every file the migration stages is fsynced before
it becomes visible, not only the last one, and the replacement inherits the
selected file's exact permission bits rather than widening or tightening them.
Migration retains every Configuration 1.0.0 member: the member set is derived
from the versioned wire type and a source that carries every member at a
non-default value is compared as a whole loaded `Configuration` before and after
each supported target, so a member an encoder drops cannot decay silently to its
default. `AssessCompatibility`
reads only the schema envelope and reports `read-only-diagnostic` when a reader
is older than the document; it exposes no decoded configuration or writer path,
and the mode is pinned across the full cross-product of pinned Configuration
versions, so the one-major step is asserted alongside the two-major one.

These are package entry points, not an advertised `ax migrate config` command
or `doctor` result. The package still does not enumerate environment variables,
copy ambient credentials, interpret unknown AX_* names, mutate roots, or claim
runtime capability/backend availability. Existing selected roots are checked
for directory kind, while absent roots remain eligible for
the later owner-only initialization boundary. An absent config with a missing
or non-directory parent, a wrong existing file kind, a read failure, and a root
inspection failure remain distinct through `errors.Is`; rendered loader errors
omit wrapped OS/filesystem details so selected machine-local paths cannot leak.

The two seams take deliberately different symlink stances, and both are pinned
in both directions at their real `LoadOS` and `MigrateOS` entries. The read seam
resolves symlinks and then applies the Section 3.2 value kind to the resolved
target: a configuration file symlinked onto a regular file loads, a root
symlinked onto a directory loads, and either link pointed at the wrong kind
still fails closed with the same typed refusal a direct path would produce. The
pinned specification states no no-follow requirement for these five classes and
resolves symlinks before comparison where it speaks about them at all, so
refusing an ordinary dotfiles or Application Support symlink would be an
invented constraint; a dangling configuration symlink accordingly resolves to a
not-yet-created regular-file path whose parent exists, which `Load` admits
without creating anything. The durable migration seam does not follow symlinks,
because its atomic rename would replace the operator's link rather than the
document the link points at, silently detaching it and leaving the real document
holding stale bytes. The specification is silent about that mutation, so the
mutating path fails closed: a selected file that is not itself a regular file is
refused with `ErrConfigNotRegular` before any backup, staging file, or
replacement is written, and the refusal leaves the directory exactly as it was.

Section 6.3 requires refusing `StrictHostKeyChecking=no`, an empty
`UserKnownHostsFile`, "or an equivalent host-authentication bypass" in a mesh
peer's `ssh_args`. An equivalence class has no enumeration, so admission is
derived rather than blacklisted: `internal/config/sshargs.go` declares the
`ssh(1)` short-option arity transcribed from the OpenSSH usage text, the short
options AX admits, and the `-o` option registry with the values each option
permits. Every argument outside that declaration is refused, so an option name
the parser has never heard of — `ProxyCommand`, `KnownHostsCommand`, `Include`,
`LocalCommand`, or anything OpenSSH adds later — fails closed, as do `-F`, a
bare destination or remote command, and any grouped short flag carrying them.
Short options are walked with getopt semantics, so `-vo StrictHostKeyChecking=no`
and `-4o UserKnownHostsFile=/dev/null` are seen as the options OpenSSH resolves
them to. `StrictHostKeyChecking` is declared with only its enforcing spelling,
which refuses the live `false` alias without listing aliases. The refusal clause
distinguishes a host-authentication option from an undeclared name, an
unpermitted flag, and an unpermitted value; the negative suite derives its cases
from those same tables, so a newly declared option is covered the moment it is
added.

Section 6 states one rule for an `extensions` key: it is a reverse-DNS key of
3-253 lowercase ASCII characters with at least one dot and dot-separated labels
matching `[a-z][a-z0-9-]{0,62}`. No label is reserved, so `validateExtensions`
admits or refuses a key by that grammar and the 253-byte bound alone, and an
`ExtensionValue` object is constrained only to string keys within nesting depth
4. A reverse-DNS namespace this organisation owns — `works.relux.env-tools`,
`com.example.auth-manager`, `io.example.endpoint-list` — therefore loads, and a
nested key named `endpoint` or `token` is preserved as data. Reading a name as
evidence about a value is not secret detection: the only "secret, token,
endpoint credential" clause in Section 6 governs a terminal backend-config
`settings` object, and it is enforced where it applies, by the closed schema a
backend implementation version registers, which refuses any settings object it
did not declare. Section 6.4's "no v2 table accepts a secret, endpoint
credential, model token, auth root, or arbitrary environment passthrough" is
likewise a field-declaration rule, enforced by the closed table shape
`decodeStrict` decodes with `DisallowUnknownFields`; Section 6.1 states the
value rule and its permission together, forbidding secret *values* in config
fields while allowing a provider to name a machine-local environment variable or
credential profile.

Run the focused tests and assigned-scope traceability gate with:

    go test ./internal/config -count=1 -v
    go test ./internal/config -cover -count=1
    go run ./internal/traceability/cmd/tracecheck \
      -section 3.2 \
      -section 6.1 -section 6.2 -section 6.3 -section 6.4 -section 6.5 \
      -section 17.1 -section 17.2 -section 17.4

## Canonical JSON and Immutable Object Identities

[`internal/canonicaljson`](internal/canonicaljson) exposes the production RFC
8785 JCS entry point and the AX immutable-object identity calculation. The JCS
path validates I-JSON input before canonicalization, including duplicate-name,
UTF-8, surrogate-pair, ECMAScript number-formatting, string-escaping, and
UTF-16 property-ordering rules. Its number behavior is checked against every
finite RFC 8785 Appendix B sample through the production entry point.

The AX identity path additionally enforces the Section 1.6 common model: JSON
numbers are integral safe integers, and the exact `schema`/`schema_version`
pair selects the schema-defined self field from the generated catalog bound to
the pinned v0.5.0 source. The catalog contains the reviewed terminal, clone,
registry, and common-object omit-self set, but catalog membership is not a
runtime shape-support claim. A separate explicit shape-validator registry is
checked for exact completeness against every catalog schema/version: a newly
registered identity cannot silently fall through to extension-only
attestation. The public calculation and verification entries currently accept
only Session Record `1.0.0`, Blob Descriptor `1.0.0`, and Transfer Manifest
`1.0.0`, whose complete shapes are validated here. Every other catalog
identity is recognized for self-field resolution but is explicitly refused by
the public entries until its schema owner supplies a complete validator. Other
registered ID names may remain ordinary references inside a supported object;
only the selected schema's own field is removed before JCS and SHA-256.

Callers cannot choose an omit field. Verification recomputes the identity and
refuses unsupported or malformed schema contracts, a missing or malformed
selected self field, raw-byte `chunk_id` objects, mutable journal variants,
self-included digests, duplicate keys, unsafe integers, and floating-point
literals. Before either identity entry attests a value, the composed path
enforces the exact Session Record `1.0.0`, Blob Descriptor, and Transfer
Manifest member sets. Session Record validation covers its Section 10.1 common
envelope, the Section 2.1 ASCII name grammar and 1–64 character bound, plus the
closed Launch Plan, Task-board Reference, Board Identity, Board Goal, and Fork
Provenance objects; other Section 10.1 record schemas have
their common envelope scalar grammar checked before the explicit unsupported
shape refusal. Blob and Manifest validation covers every self-contained closed
nested object and rejects BlobChunk index, offset, size, bounds, ordering, and
coverage violations. The Manifest nested gate includes common scalar types,
Unicode character bounds, tagged nullability, array count/order/uniqueness, Git
object-format consistency, index count/stage rules, hardlink and lexical
symlink rules, and recursive submodule state/depth/count constraints that can
be proven from the candidate object alone. Rules that need referenced Blob
Descriptors, child manifests, raw Git pack/index bytes, an isolated Git object
database, or filesystem resolution remain external integrity/materialization
checks; this identity package does not claim to perform them. The complete
per-member inventory is checked against the production `requireExactMembers`
argument lists, so adding, removing, or renaming a member without updating its
pinned constraint row fails the focused suite. Manifest entry destination-case
collision detection uses a linear Unicode simple-fold set; a production-entry
regression covers the declared 65,536-entry maximum inside the encoded identity
size cap.
The shared strict decoder behind `Canonicalize` and both identity entries
enforces a declared 256-container nesting bound: a document that opens a 257th
nested object or array is refused with the typed `ErrInvalidJSON` error before
any recursion can exhaust the goroutine stack, which Go otherwise reports as an
uncatchable fatal runtime error. The bound leaves more than sixfold headroom
over the deepest normative closed shape (the 16-level submodule tree) and stays
far below encoding/json's 10,000-level cap, so it is always the first gate a
deep peer object meets. The suite pins the literal value, proves accept-at-limit
and refuse-past-limit at every public entry in array, object, and mixed
container shapes, and replays the original 2,000,000-byte nested-array crash
input as a typed refusal.
Every open `extensions` map is validated against the Section 1.6 reverse-DNS
key, member-count, nesting-depth, and canonical-size rules before either entry
attests it. Declared `string[n..m]` bounds count Unicode characters rather than
UTF-8 bytes, including Blob media type, Manifest symlink target, workspace
identity, remote name, and recursive submodule identity boundaries.

Native fuzz targets recursively attack malformed member syntax, duplicate
names, unpaired surrogates, and UTF-16 ordering through `Canonicalize`, then
prove canonical read-back, outer-whitespace invariance, omit-self identity
order independence, and successful verification of every accepted generated
identity. A separate refusal target injects unknown top-level and nested
members plus Session Record envelope/member mutations, chunk rules, malformed
nested digests/refs/URLs, Unicode over-bounds, TM-GIT-N2 stage 4, impossible
submodule state, sparse-pair drift, and hardlink/symlink/count violations
through both identity production entries. The checked-in cross-platform golden
runs equivalent LF and CRLF representations with different property orders
against one language-neutral SHA-256 identity. Validation uses fixed `100x`
budgets and one
worker per fuzz target so the gate is bounded by an input count rather than
wall-clock timing.

This package is read-only and deterministic; it does not mutate durable state.
It supplies identity calculation for a new schema-versioned object under
Section 17.3, but does not claim to implement migration publication, atomic
reference advancement, rollback retention, `ax migrate`, `ax doctor`, or any
runtime capability. Those surfaces remain unavailable until their owning
implementation tasks land.

Run its focused tests with:

```bash
go test ./internal/canonicaljson -count=1
go test ./internal/canonicaljson -cover -count=1
go test ./internal/canonicaljson -run=^$ \
  -fuzz=^FuzzCanonicalizeRoundTrip$ -fuzztime=100x -parallel=1
go test ./internal/canonicaljson -run=^$ \
  -fuzz=^FuzzObjectIdentityRepresentationInvariant$ -fuzztime=100x -parallel=1
go test ./internal/canonicaljson -run=^$ \
  -fuzz=^FuzzClosedIdentityShapeRefusal$ -fuzztime=100x -parallel=1
go run ./internal/traceability/cmd/tracecheck -section 17.3
```

## Generated Contract Catalogs

[`internal/catalog`](internal/catalog) exposes typed records through
`catalog.Current()` for v0.5.0 and `catalog.ForRelease()` for the exact pinned
v0.4.3 compatibility projection. The reviewed input is
[`catalog.v0.5.0.json`](internal/catalog/catalog.v0.5.0.json); generation first
verifies the exact [`specpin`](internal/specpin) lock, strictly rejects partial,
unknown, substituted, duplicate, release-incompatible, or unreviewed semantic
metadata through a canonical projection digest, then writes
[`catalog_gen.go`](internal/catalog/catalog_gen.go) atomically and
deterministically. JSON whitespace and formatting do not alter that reviewed
semantic identity.

| Release projection | Contracts | Operations | Capability names | Events | Error codes |
| --- | ---: | ---: | ---: | ---: | ---: |
| v0.5.0 | 60 | 99 | 46 | 112 | 109 |
| v0.4.3 | 55 | 89 | 30 | 112 | 94 |

The source pin and catalog metadata cover every normative top-level section
claimed by the implementation board: Sections 1-20 and Appendices A-D. Each
generated family retains its exact defining section and named
Appendix D fixture anchors. Durable operation entries include
their reviewed idempotency scope and crash/lost-result recovery evidence;
Terminal Backend entries reproduce the distinct Section 4.C canonical keys.
Capability entries are vocabulary members only: the public type has no
availability, enabled, supported, or status field, so catalog generation cannot
advertise runtime capability.

Regenerate and verify the committed output with:

```bash
go generate ./internal/catalog
go run ./internal/catalog/cmd/cataloggen -metadata internal/catalog/catalog.v0.5.0.json -contracts internal/specpin/v0.5.0.lock.json -output internal/catalog/catalog_gen.go -check
go test ./internal/catalog ./internal/cataloggen ./internal/catalog/cmd/cataloggen -count=1
```

## Specification-to-Code Ownership Gate

[`internal/traceability`](internal/traceability) provides the read-only
repository gate used by CI. Its reviewed
[`ownership.v0.5.0.json`](internal/traceability/ownership.v0.5.0.json)
registry independently enumerates implementation owners for all 60 current
contract rows, 36 pinned or catalog-referenced normative section keys, 30
executable acceptance cases, and 30 exact fixture identities or Appendix D
anchors. The v0.4.3 projection is checked as an owned 55-contract subset.
The generated v0.5.0 catalog also carries the reviewed schema/version/self-field
contracts used by canonical object identity calculation; generator validation
binds each row to a pinned contract and rejects duplicate, unsupported,
malformed, or digest-drifted metadata.

The production validator re-verifies the exact source lock and reviewed catalog
metadata, refuses stale generated output, requires every production and test
owner to resolve to a real Go declaration, and rejects gaps, duplicates,
self-minted keys, partial or malformed reads, and semantic ownership drift.
GitHub Actions invokes the gate directly before generation, tests, vet, and
build:

```bash
go run ./internal/traceability/cmd/tracecheck
go run ./internal/traceability/cmd/tracecheck -section 9.2 -section 7.9
```

The repeated `-section` form is the Story-scope production gate. It resolves
each assigned subsection, and every heading in a same-top-level range, against
the immutable v0.5.0 inventory. Every exact `section_binding` must name its own
production declaration and executable acceptance case. A generic top-level
source pin is not a scoped implementation owner. The common-types and canonical
identity implementations now own the assigned Section 1.6, Sections 10.1-10.4,
and Section 17.3 identity-contribution bindings; malformed, nonexistent,
unpinned, or otherwise unowned assignments fail closed.

Successful output reports ownership inventory counts only. The gate does not
mutate repository or product state, add an `ax` command or `doctor` result,
or claim that any catalog capability is available, enabled, or supported.

## Managed Skills

`Skillfile.json` targets the Curator-standard `claude_code` and `codex_cli`
agent adapters and pins this project-local skill to an exact revision:

| Skill | Revision |
| --- | --- |
| `relux-works/skill-go-testing-tools` | `90c1515239eed9321068f3bafbeb5d0a0c2aa26a` |

Project management is intentionally global. The repository does not declare or
install `project-management`; development uses the globally installed skill
and `task-board` CLI.

Curator owns `.agents/`, `.claude/skills/`, and `.codex/skills/`. Do not edit
their generated contents directly; change `Skillfile.json` and rerun Curator.

## Tools and Outputs

| Tool | Purpose | Command or entry point | Outputs |
| --- | --- | --- | --- |
| Curator | Pin, install, and validate project skills | `curator install`; `curator status --check` | `.agents/`, `.claude/skills/`, `.codex/skills/` |
| `task-board` | Track scope, lifecycle, checklists, evidence, dependency waves, and the critical path through the global `project-management` installation | `task-board q 'plan()'`; `task-board q 'plan(TASK-260830-55kcni, mode=related)'`; `task-board plan --save` | `.task-board/`; `.planning/`; task outcome resources |
| Go toolchain | Verify global and assigned-scope specification ownership, validate versioned Configuration readers/current writer, validate and fuzz common wire scalars and canonical identities, generate and check the typed catalogs, build, test, and measure the Go implementation | `go run ./internal/traceability/cmd/tracecheck`; `go run ./internal/traceability/cmd/tracecheck -section 6.1 -section 6.2 -section 6.3 -section 6.4 -section 6.5 -section 17.1 -section 17.2 -section 17.4`; `go test ./internal/config -cover -count=1`; `go test ./internal/scalar -cover -count=1`; `go test ./internal/scalar -run=^$ -fuzz=^FuzzScalarProductionEntries$ -fuzztime=100x -parallel=1`; `go test ./internal/canonicaljson -cover -count=1`; `go test ./internal/canonicaljson -run=^$ -fuzz=^FuzzCanonicalizeRoundTrip$ -fuzztime=100x -parallel=1`; `go test ./internal/canonicaljson -run=^$ -fuzz=^FuzzObjectIdentityRepresentationInvariant$ -fuzztime=100x -parallel=1`; `go test ./internal/canonicaljson -run=^$ -fuzz=^FuzzClosedIdentityShapeRefusal$ -fuzztime=100x -parallel=1`; `go generate ./internal/catalog`; `go run ./internal/catalog/cmd/cataloggen -metadata internal/catalog/catalog.v0.5.0.json -contracts internal/specpin/v0.5.0.lock.json -output internal/catalog/catalog_gen.go -check`; `go test ./... -v`; `go test ./... -cover`; `go build ./...` | Read-only traceability report; `internal/catalog/catalog_gen.go`; Go build/fuzz cache; test output captured under `.temp/<TASK-ID>/` when needed |
| `github.com/gowebpki/jcs` | RFC 8785 byte transformation after repository-owned strict I-JSON validation | Imported by `internal/canonicaljson.Canonicalize` at pinned module version `v1.0.1` | Canonical UTF-8 JSON bytes in memory; no durable output |
| `github.com/pelletier/go-toml/v2` | Parse and emit TOML while the repository-owned Configuration layer enforces exact versioned closed schemas | Imported by `internal/config.Decode`, `internal/config.EncodeCurrent`, and explicit `internal/config.Migrate` at pinned module version `v2.4.3` | Validated Configuration values/TOML bytes in memory; explicit migration writes a same-directory replacement plus an owner-only versioned backup |
| GitHub Actions | Enforce traceability, generated-output, test, vet, and build gates on pull requests and `main` | `.github/workflows/ci.yml` | GitHub-hosted CI check results |
| Git | Branch, diff, and create signed commits/tags | `git status`; `git diff --check`; `git commit -S`; `git tag -s` | Git objects and refs under `.git/` |
| GitHub CLI | Inspect and open pull requests after bootstrap | `gh pr create`; `gh pr checks` | Pull requests and checks on GitHub |

Temporary validation logs and worktrees belong under `.temp/<TASK-ID>/` and
must not be committed. Diagrams, when implementation work introduces them,
belong under `diagrams/`.

Configuration loading, migration, and downgrade assessment use the existing Go
toolchain entry in the tools table. Focused outputs are Go test/coverage results
and the read-only traceability report; migration writes only when its explicit
production entry is called, and task-scoped captured logs belong under
`.temp/<TASK-ID>/`.

## Contribution Baseline

All non-bootstrap changes are delivered automatically through pull requests as
author-signed commits, then landed by fast-forwarding the exact reviewed head
into `main`. Before creating a branch or task-scoped worktree, fetch
`origin/main`, fast-forward local `main`, and verify the refs are equal. Track
the work in `task-board`, use the managed `go-testing-tools` skill for Go
changes, use the global `project-management` skill for orchestration, run
relevant validation, and attach task-scoped evidence before review. See
[AGENTS.md](AGENTS.md) for the full contract.
