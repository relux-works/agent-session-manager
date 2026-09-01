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
contract rows, 36 pinned or catalog-referenced normative section keys, 29
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
| Go toolchain | Verify global and assigned-scope specification ownership, validate and fuzz common wire scalars and canonical identities, generate and check the typed catalogs, build, test, and measure the Go implementation | `go run ./internal/traceability/cmd/tracecheck`; `go run ./internal/traceability/cmd/tracecheck -section 1.6 -section 10.1 -section 10.2 -section 10.3 -section 10.4 -section 17.3`; `go test ./internal/scalar -cover -count=1`; `go test ./internal/scalar -run=^$ -fuzz=^FuzzScalarProductionEntries$ -fuzztime=100x -parallel=1`; `go test ./internal/canonicaljson -cover -count=1`; `go test ./internal/canonicaljson -run=^$ -fuzz=^FuzzCanonicalizeRoundTrip$ -fuzztime=100x -parallel=1`; `go test ./internal/canonicaljson -run=^$ -fuzz=^FuzzObjectIdentityRepresentationInvariant$ -fuzztime=100x -parallel=1`; `go test ./internal/canonicaljson -run=^$ -fuzz=^FuzzClosedIdentityShapeRefusal$ -fuzztime=100x -parallel=1`; `go generate ./internal/catalog`; `go run ./internal/catalog/cmd/cataloggen -metadata internal/catalog/catalog.v0.5.0.json -contracts internal/specpin/v0.5.0.lock.json -output internal/catalog/catalog_gen.go -check`; `go test ./... -v`; `go test ./... -cover`; `go build ./...` | Read-only traceability report; `internal/catalog/catalog_gen.go`; Go build/fuzz cache; test output captured under `.temp/<TASK-ID>/` when needed |
| `github.com/gowebpki/jcs` | RFC 8785 byte transformation after repository-owned strict I-JSON validation | Imported by `internal/canonicaljson.Canonicalize` at pinned module version `v1.0.1` | Canonical UTF-8 JSON bytes in memory; no durable output |
| GitHub Actions | Enforce traceability, generated-output, test, vet, and build gates on pull requests and `main` | `.github/workflows/ci.yml` | GitHub-hosted CI check results |
| Git | Branch, diff, and create signed commits/tags | `git status`; `git diff --check`; `git commit -S`; `git tag -s` | Git objects and refs under `.git/` |
| GitHub CLI | Inspect and open pull requests after bootstrap | `gh pr create`; `gh pr checks` | Pull requests and checks on GitHub |

Temporary validation logs and worktrees belong under `.temp/<TASK-ID>/` and
must not be committed. Diagrams, when implementation work introduces them,
belong under `diagrams/`.

## Contribution Baseline

All non-bootstrap changes are delivered automatically through pull requests as
author-signed commits, then landed by fast-forwarding the exact reviewed head
into `main`. Before creating a branch or task-scoped worktree, fetch
`origin/main`, fast-forward local `main`, and verify the refs are equal. Track
the work in `task-board`, use the managed `go-testing-tools` skill for Go
changes, use the global `project-management` skill for orchestration, run
relevant validation, and attach task-scoped evidence before review. See
[AGENTS.md](AGENTS.md) for the full contract.
