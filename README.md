# Agent Session Manager (`ax`)

Agent Session Manager is the implementation repository for the local-first,
cross-host `ax` coding-agent session orchestrator.

## Project Status

The Curator-managed repository foundation is complete and the published AX
v0.5.0 specification has been decomposed into an implementation board. The
Go contract foundation pins the immutable normative source and generates typed
contract, operation, capability-vocabulary, event, and error catalogs from
reviewed implementation metadata. The local storage foundation also resolves
the five Section 3.2 path classes and installs verified immutable blobs on
native macOS/Linux filesystems. Its local SQLite index projects those supported
raw blobs without becoming authority. Session, provider, terminal, mesh, and
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

## Pinned Specification Document

[`internal/specdoc`](internal/specdoc) embeds the byte-exact `SPEC.md` of that
same pinned release so repository fidelity gates can compare their artifacts
against the specification text instead of against the implementation those
artifacts are supposed to constrain. The upstream specification remains
normative; this copy is a verification input and amends nothing.

`Load` accepts the document only when its SHA-256 equals
`specpin.DocumentSHA256`, so a substituted, edited, truncated, or unreadable
document is refused rather than silently compared against — otherwise a swapped
specification would confirm whatever it happened to contain. `QuoteLines`
resolves an excerpt to the 1-based `SPEC.md` lines it begins on, `SectionID`
resolves a line to the numbered clause that contains it, `TableRowAt` reports
what a Markdown table body row declares, and all three work under one
normalization rule: runs of ASCII whitespace collapse to a single space. Case,
punctuation, digits, and inline `<code>` markup are compared exactly. A
whitespace run that crosses a hard boundary collapses instead to a block
separator that no normalized excerpt can contain, so a quote cannot stitch
across one; both halves of such a stitch are individually verbatim, so
whitespace collapsing alone admitted it. Two boundaries are hard: a blank line,
which separates a block from the next, and the newline between two adjacent
table rows, because a table row is a complete line by construction. Inside one
block the newline is still forgiven, deliberately — the document's hard line
wrapping, its table indentation, and the newline between two adjacent list items
or two adjacent lines of one paragraph.

Only repository gates import this package - the test binaries that check
enumeration artifacts, and `internal/traceability`, which measures the clause
inventory of a bound section from it. The embedded document never reaches the
`ax` command, which does not exist yet, and
`TestEmbeddedDocumentNeverReachesAProductBinary` is what keeps that true once it
does: it reads the module import graph from source, refuses any `main` package
outside `tracecheck` that can reach this package, and proves the detector by
planting a `cmd/ax` that imports `internal/traceability`. It is read-only,
mutates no durable state, and advertises no runtime capability.

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
go run ./internal/traceability/cmd/tracecheck
```

The assigned-scope gate refuses Sections 1.6, 10.1, 10.2, 10.3 and 10.4: each
binding validates a corner of its section and the gate now says so with the
measured ratio. See [Specification-to-Code Ownership
Gate](#specification-to-code-ownership-gate).

```bash
go run ./internal/traceability/cmd/tracecheck \
  -section 1.6 -section 10.1 -section 10.2 -section 10.3 -section 10.4
# exits non-zero: 0/31, 0/3, 0/5, 1/3 and 0/25 normative clauses discharged
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

That gate reads `ssh_args` only, and Section 6.2 hands `ssh(1)` the peer
`endpoint` as an atomic argv value too. An atomic argv value is not by itself a
destination: `ssh(1)` reads its destination positionally through getopt, so an
endpoint beginning with `-` is parsed as an option. `-oStrictHostKeyChecking=no`
written into `endpoint` was therefore the same Section 6.3 bypass reached
through a field the `ssh_args` gate never inspects, and an endpoint carrying a
space is that injection one word-split away. `internal/config/endpoint.go`
closes it the same derived way: the field is admitted against a closed
`[user@]host[:port]` grammar — a 1-64 character login name, LDH DNS labels or a
bracketed IPv6 literal, and a decimal port in 1-65535 — and refused otherwise,
naming which clause it violated. The grammar is narrower than every destination
`ssh(1)` would accept, so widening it is a reviewed change; the 1-1024 character
endpoint bound stays reachable through the grammar and is still proved at both
its limits.

Every negative case for that grammar carries an isolating neighbour: the same
endpoint with only the named violation removed, asserted to be admitted. A case
that names the `host` clause but would also be refused by the `port` clause
pins nothing, so the neighbour is what proves the named clause is the one
deciding. The leading-hyphen clause needs it most — the reported shapes such as
`-oStrictHostKeyChecking=no` are refused by the host grammar too once the
hyphen clause is gone, while `-ivan@peer.example` is not, because `-` is a legal
login-name byte and `ssh(1)` reads that value as `-i` with identity file
`van@peer.example`.

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
    go run ./internal/traceability/cmd/tracecheck -section 6.2

Section 6.2 is the only configuration section the assigned-scope gate admits:
its single normative clause is enumerated and discharged. Sections 3.2, 6.1,
6.3, 6.4, 6.5, 17.1, 17.2 and 17.4 are refused with their measured ratio, and
Section 6.5 additionally names the `required_capabilities` default defect. See
[Specification-to-Code Ownership Gate](#specification-to-code-ownership-gate).

## Provider Plugin Discovery and Trust

[`internal/provider`](internal/provider) implements the Section 7.1 trusted
provider plugin host boundary: deterministic executable discovery, trust
recording, and substitution detection. `Discover` is the production entry
point. It enumerates candidates in the section's source order — configured
`providers.plugin_dirs` in listed order (entry names sorted bytewise within
each directory), then the six built-in adapters (`codex`, `claude`, `gemini`,
`muse`, `antigravity`, `pi`), then `PATH` only when `allow_path_plugins` is
true — and records the canonical absolute path, SHA-256 digest, and approving
owner for every accepted external `ax-provider-<id>` executable. Symlinks are
resolved before comparison, the target must be a regular file owned by the
operator or an administrator-approved identity, and `Trust`/`Verify` carry the
receipt a changed path target or digest must renew.

If two candidates declare the same provider ID, discovery fails with
`invalid_config` before either is probed or executed; the package has no probe
or execution dependency, and no source imports a process-execution facility, so
the ordering holds structurally. Filesystem read failures abort discovery with
`local_precondition_failed` instead of yielding a partial set, and a receipt
that no longer matches freshly read facts fails verification with
`integrity_failure`. All three codes are pinned against the contract registry
with their exact exits (3, 3, 9). A discovered candidate carries no
availability, status, or capability claim, and the package is an internal M0
host boundary, not a public stable plugin SDK, which the section defers. The
package mutates no durable state: the host persists the `TrustRecord`, and
`Verify` rechecks it. Native Windows owner attestation is unimplemented, so
external executables are undiscoverable there until a Windows owner model
lands; built-in adapters are unaffected.

Run the focused tests and coverage with:

```bash
go test ./internal/provider -count=1
go test ./internal/provider -cover -count=1
```

The package derives its refusal inventory from its own source: every refusal
constructor call site must have an exercised negative path, no `Error` value
may be built outside the four constructors, and the observed code set must
equal the closed three-code set. Section 7.1 citations in its tests resolve
against the digest-pinned document in `internal/specdoc`, quoting the exact
line each claim begins on.

## Provider Plugin JSONL Protocol

[`internal/provhost`](internal/provhost) implements the ax host side of
the Section 7.2 provider plugin JSON-over-stdio protocol: one-frame JSONL
transport, deadlines, size limits, stdout/stderr separation, operation
dispatch, and status recovery. `Host.Call` is the production entry point.
It frames one request envelope (`protocol`, `protocol_version`,
`request_id`, `operation`, `deadline`, `body`), starts one plugin process
per operation, enforces the request deadline, and interprets the single
response frame. Unknown operations, stale deadlines, and unframeable
bodies are refused with `invalid_config` before any process starts.

Each stdout line must be one complete UTF-8 JSON object no larger than 8
MiB; stdout carries the single response frame while diagnostics stay on
stderr and never enter failure text. A recognizable foreign major yields
`incompatible_protocol` without trusting the payload; every other
unusable frame yields `provider_protocol_error`, both as local Structured
Error 1.0.0 objects built without adopting any child-supplied code,
retryable bit, details, or authority. A plugin that exceeds its deadline
is terminated (as a process group on unix; direct-child kill on Windows)
and reported as `provider_timeout`; a
crash without a response is `provider_process_failed`. On success the raw
body is returned for the operation layer; on failure the bound child
error is returned itself. All six codes are pinned against the contract
registry with their exact exits (3, 13, 6, 13, 13, 9).

Dispatch covers exactly the 15 Section 7.5 operations in manifest order,
derived from the pinned specification text, and refuses any other name
without spawning. Status recovery interprets `materialize-status` bodies
through the Section 7.5 state rules: unknown fails closed with
`integrity_failure` for quarantine, prepared requires plan, token, and
discovery, terminal states require plan and discovery with no token, and
identity members must equal the requested transaction. Reads evolve —
prepared under one envelope may be committed under the next — and each
call starts a fresh process, so recovery observes durable state through
the passed authority with no host-side mutation cache.

The operation layer validates what a plugin returned: the closed
Section 7.3 manifest (registries derived from the pinned example),
the Section 7.4 probe with its seven capability values (only
`available` may be `enabled`), the host-side capability gate
(`RequireCapability` refuses unproven surfaces before any process
starts), the exact Section 7.7 profile mapping (six providers times
two profiles swept, standard always the empty omission), the
Section 7.6 quiescence proof (a `safe` claim over any unproven fact
is refused; honest `unsafe` validates), the launch/resume SpawnPlan
(argv, destination-native cwd, sorted unique environment names,
literals disjoint from inherited names, and a `profile_mapping`
equal to the Section 7.7 mapping), the Section 5.5 Provider Identity
Record with the identify-session wrapper, and the single
`(operation, operation_id)` mutation key (keyed operations derived
per-row from the Section 7.5 table). A lost mutation retried with
identical bytes returns byte-identical results across fresh
processes; a changed body surfaces the bound `idempotency_mismatch`
child with no further frame sent. Mutation receipts themselves live
in the transaction document and the plugin, not the host. Full
Section 7.5 request-body vocabularies beyond these surfaces, and the
materialize commit/rollback result bodies, still cross the package
opaquely behind the object-shape check.

Run the focused tests and coverage with:

```bash
go test ./internal/provhost -count=1
go test ./internal/provhost -cover -count=1
```

The package derives its refusal inventory from its own source: every
refusal constructor call site must have an exercised negative path, no
Structured Error may be built outside the six constructors, no raw error
may be minted, and the observed code set must equal the closed six-code
set. Every refusal assertion names the arm it must reach (the refused
member and the rule detail), so a gate deleted from a required-member list
slides to a lower arm and reddens instead of passing silently. The
rollback-token entropy floor is pinned at exactly 31 refused / 32 accepted
decoded bytes, both frame-size bounds are pinned at exactly limit refused /
limit accepted from both directions, and the operation registry is pinned
against the Section 7.5 table in order. Refusal arms are derived from
production source rather than listed: every frameFault literal,
integrity detail, and parseMajor classification branch must carry a
witness at the production entry, and every witness must name a derived
arm, so a planted arm or a deleted branch reddens in either direction. This inventory is independent of the per-package refusal inventories in [`internal/provider`](internal/provider) above and [`internal/terminalbackend`](internal/terminalbackend) below (see [Specification-to-Code Ownership Gate](#specification-to-code-ownership-gate)): each package derives its own refusal arms from its own source in its own test file, and the three share no inventory code or table.
stdoutCap is pinned at MaxFrameBytes+2 in both directions, with a
maximal-frame-plus-junk probe through the production runner proving the
probe byte is load-bearing.

## Canonical JSON and Immutable Object Identities

[`internal/canonicaljson`](internal/canonicaljson) exposes the production RFC
8785 JCS entry point and the AX immutable-object identity calculation. The JCS
path validates I-JSON input before canonicalization, including duplicate-name,
UTF-8, surrogate-pair, ECMAScript number-formatting, string-escaping, and
UTF-16 property-ordering rules. Its number behavior is checked against every
finite RFC 8785 Appendix B sample through the production entry point.

All four Section 1.6 boundary fixtures that publish a digest are pinned:
`NUM-SAFE-MAX`, `NUM-U64-STRING`, `NUM-U64-MAX`, and `JCS-UTF16-ORDER` are each
recomputed from the production encoder and compared against the value quoted
from `SPEC.md` at the pinned commit, not against a value the implementation
derived for itself.

The `utf8.ValidString` re-checks in `closed_shapes.go` are kept and documented
rather than deleted; each names `decodeStrict` as the validator that subsumes it.
That subsumption is machine-checked, not only asserted in prose: no exported
function or method may hand an already-decoded value to one of those re-checks,
and `decodeStrict` must remain the only place in the package where bytes become
Go values. Adding a second decoder reddens the pin, as does a map-taking
exported entry point that reaches a re-check by calling it, through the
package's own `immutableObjectShapeValidators` dispatch table, or as a method on
an exported type. All 7 of the 7 derived re-check functions are reachable from
an exported entry point, so none of them is pinned vacuously, and that ratio is
itself asserted rather than left to prose.

The check is bounded, and the bound is published with it. Its call graph is
derived from the AST, so it models direct calls, calls to methods declared in
the package, and dispatch through a function value; it models nothing about a
callee reached by reflection, by a function value handed to another package, or
through a func-typed struct field. An entry point built one of those ways would
leave the guards live while the pin stayed green — which is why the guards are
kept rather than deleted.

The AX identity path additionally enforces the Section 1.6 common model: JSON
numbers are integral safe integers, and the exact `schema`/`schema_version`
pair selects the schema-defined self field from the generated catalog bound to
the pinned v0.5.0 source. The catalog contains the reviewed terminal, clone,
registry, and common-object omit-self set, but catalog membership is not a
runtime shape-support claim. A separate explicit shape-validator registry is
checked for exact completeness against every catalog schema/version: a newly
registered identity cannot silently fall through to extension-only
attestation. The public calculation and verification entries currently accept
Session Record `1.0.0`–`3.0.0`, Session Event `1.0.0`–`4.0.0`, Lease,
Checkpoint, Provider Identity, Workspace Group, Blob Descriptor, and Transfer
Manifest `1.0.0`, whose complete shapes are validated here. Every other catalog
identity is recognized for self-field resolution but is explicitly refused by
the public entries until its schema owner supplies a complete validator. Other
registered ID names may remain ordinary references inside a supported object;
only the selected schema's own field is removed before JCS and SHA-256.

Callers cannot choose an omit field. Verification recomputes the identity and
refuses unsupported or malformed schema contracts, a missing or malformed
selected self field, raw-byte `chunk_id` objects, mutable journal variants,
self-included digests, duplicate keys, unsafe integers, and floating-point
literals. Before either identity entry attests a value, the composed path
enforces exact top-level and nested shapes for every supported record. Session
Record validation covers its Section 10.1 common
envelope, the Section 2.1 ASCII name grammar and 1–64 character bound, plus the
closed Launch Plan, Task-board Reference, Board Identity, Board Goal, and Fork
Provenance objects. Major 2 replaces the v1 fork field with the required
`origin`, `same_provider_fork`, or `cross_environment_clone` derivation union;
major 3 keeps that exact top-level wire field and adds the `native_adoption`
creation tag. Each Environment Tuple is independently validated as a closed
object with the declared environment-ID grammar plus the platform and
architecture vocabularies. That grammar is not stated by the pinned
EnvironmentTuple clause, which only names `environment_id`; the sole pinned
statement of it is the Session Adapter Manifest row of a different schema, and
the enumeration row now quotes both so the cross-schema step is visible rather
than implied. The pinned tuple declaration assigns no type or
bound to `environment_version`, `store_schema_fingerprint`, or
`adapter_version`, so identity validation requires their presence without
inferring a constraint from another schema or a member name; in particular the
SemVer word on `adapter_version` belongs to the Session Adapter Manifest row of
a different schema and is not carried across. Where the document does type a
member `semver` — Section 17.3 migration provenance and the `terminal.*` Session
Event versions — the constraint is Semantic Versioning 2.0.0 in full, so
prerelease and build metadata are accepted. Both halves of that decision are
recorded in
[`internal/canonicaljson/testdata/constraint-enumeration.md`](internal/canonicaljson/testdata/constraint-enumeration.md).

Every row of that enumeration is compared against the pinned specification, not
only against this package. `TestConstraintEnumerationMatchesRequireExactMembers`
still derives the member set and call site from the production
`requireExactMembers` argument lists, and
`TestConstraintEnumerationSpecExcerptsQuoteThePinnedSpecification` additionally
requires each row's `Pinned SPEC declaration` to quote the hash-verified
[`internal/specdoc`](internal/specdoc) document verbatim, beginning on the exact
`SPEC.md` line the row declares, with at least one entry naming the member. A row
may instead mark an entry `paraphrase:`, which still has to name a line whose raw
text contains that member. `TestArtifactQuotesAreVerbatimPinnedSpecificationText`
extends the same requirement to every curly-quoted span elsewhere in the file.
`TestPlantedConstraintEnumerationRowsRedden` plants fifteen defects into a copy
of the shipped artifact and requires each to fail — an invented quote, a true
quote at the wrong line, a true quote that never names its member, a true quote
from another shape's clause, a true quote from a sibling member's table row, a
quote stitched across a blank line, a quote stitched across a table row
boundary, an out-of-document paraphrase, and the previous bare-prose cell among
them — while `TestUnmodifiedConstraintEnumerationIsAdmitted` requires the
shipped artifact to pass. This exists because the column used only to be required non-empty: the
artifact and the code could be wrong about the contract together and stay green,
which is how a quoted word reached a column for a member the pinned document
never types that way.

A verbatim, correctly located quote can still be about a different schema, so
each shape additionally pins the numbered `SPEC.md` clauses its citations may
come from, and `specdoc.SectionID` resolves every cited line to its nearest
enclosing numbered heading. Retargeting `ManifestEntry.file.size` from its
Section 10.4 row to BlobChunk's bounded Section 10.2 clause used to leave the
package green; it is now refused by clause number.
`TestEveryConstraintEnumerationShapePinsItsSpecificationClause` asserts the
pinned shape set exactly against the artifact, and
`TestClauseAnchorRefusesEveryForeignSectionForOneRow` plants a verbatim line
from each of the eleven other clauses into one row, requires all eleven to be
refused, and requires the shipped row to still pass.

A clause anchor alone is still not a shape anchor. Ten shapes are declared in
Section 10.4, and the member anchor is a substring test another schema's
identifier can satisfy, so a citation retargeted within one clause used to be
admitted — seven shipped rows were exactly that, quoting a sibling Git type's
declaration row while production enforced something else. Citations that land on
a Markdown table body row are therefore additionally held to the row that
declares what they cite: the first cell must name the member, or the identifier
under which the document declares that shape, which
`constraintRowDeclaringIdentifiers` pins per shape and
`TestEveryConstraintEnumerationDeclaringIdentifierIsExercised` asserts exactly
against the artifact. `TestDeclaringRowAnchorRefusesEverySiblingRowOfTheGitTable`
derives every shared-member pair of the Section 10.4 Git table from the document
itself, plants all fourteen retargets in both directions, and requires each to be
refused by the sibling's name. Two citations are exempted by name and reason —
both Session Record majors quote the Section 2.1 Terms row that carries the
`name` grammar the Session Record clause defers to — and an unused exemption
reddens. The residual gap is a citation that lands outside every table row: the
cross-schema `environment_id` finding below sits there, both of its lines are in
Section 7.8 prose, and it was reached by reading the document, not by this gate.

Membership in the Session Record grammar reachability gate is keyed on shape and
member, never on that column. `TestSessionRecordDeclaredGrammarRowsReachIdentityProductionEntries`
resolves a pinned row set with a pinned per-family row count, so correcting a
row's quote cannot move a row out of the gate and a narrowed family reddens as
loudly as an emptied one. It is keyed that way because it was not: classification
used to substring-match the declaration prose, and rewriting that column dropped
eight rows out of the gate while it stayed green, because its only completeness
check was that no family was empty. `TestSessionRecordGrammarRowSetRefusesASilentlyNarrowedEnumeration`
drops one row at a time and requires the report to name it, and
`TestSessionRecordGrammarClassificationIgnoresPinnedSpecProse` replaces every
row's declaration cell with unrelated text and requires the classification not to
move.

Seven of the eleven reverse-DNS rows in that set — the `extensions` of Board
Goal, Board Identity, Fork Provenance, and the four derivation-provenance
variants — are kept on production enforcement rather than on a local pinned
declaration. `SPEC.md` states the reverse-DNS key rule as a local table row only
for Launch Plan, Task-board Reference, and the two Session Record majors; for the
other seven it names `extensions` in a prose member list and never restates the
rule. Production routes all eleven through one shared `validateExtensionsObject`,
so each shape's reachability is still worth attacking, and the enumeration row
records what the document does and does not say separately from that. The Session
Record contract does not require its source and target `environment_id` values
to differ. AX-source nullability, source Session ID
separation, and immutable target-provider equality are enforced before identity
calculation or verification. Other Section 10.1 record schemas have
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
checks; this identity package does not claim to perform them. Its Section 2.2
identity contribution is limited to validating invariant 10's persisted
`execution_profile` member; it does not claim initial-launch/resume, lease,
replica, event-fencing,
replication, materialization, tombstone, capability, bridge, sync, or takeover
runtime enforcement. Session Event validation derives its complete
version/type registry from the generated catalog, selects the exact closed
payload for every registered v1–v4 event, retains unknown v1 event types as
inert immutable history, and refuses cross-major payload leakage. Lease,
Checkpoint, Provider Identity, and Workspace Group validation covers their
candidate-local scalar, closed-union, ordering, bound, literal, and cross-field
requirements, including Safe Boundary Evidence and direct/task-board
persistence exclusivity. Checks requiring a referenced predecessor, winning
lease, Session Record kind/profile, event DAG history, provider-specific secret
classification, materialized filesystem, or converged workspace-group history
remain external integrity/publication gates; this package reports no claim for
those facts.

Session Event 4.0.0 replaces exactly the `terminal.created` and
`session.resumed` payloads, and both declare `terminal_backend_id` as the
Section 4.B `terminal-backend-id` scalar type rather than a bare string.
`requireTerminalBackendID` enforces the declared 1–128 ASCII byte bound and the
declared `[a-z][a-z0-9]*(?:[.-][a-z0-9]+)*` grammar before either identity entry
attests the event; the declared minimum is subsumed by the existing non-empty
string check. Section 4.B also reserves the `ax.` namespace and names `ax.tmux`
and `ax.conpty` as canonical built-ins, but that is a registry and trust rule
rather than a payload constraint, so a vendor-namespaced backend ID that matches
the grammar is accepted here and no backend registry, discovery, probe, or trust
capability is claimed by this package.

`ValidateObservationEvent` validates one closed Section 18.1 Observation Event,
including the observation-name grammar, nullable fields, the result/error
relationship in both directions — `partial` and `failure` require a non-null
`error_code` and every other admitted result requires null, each half driven for
every result the production vocabulary admits — exact counts object, object-ID
ordering, and extension boundary.
`ValidateObservationStream` additionally requires one stream UUID, a first
sequence of 1, and exact `+1` continuity. It validates a supplied read-only
snapshot; it does not append, rotate, authorize, or retrieve logs. Whether a
terminal observation belongs to a measured operation requires operation-history
context and is intentionally not inferred from a single candidate.

The per-member inventory is checked against the production `requireExactMembers`
argument lists, so adding, removing, or renaming a member without updating its
pinned constraint row fails the focused suite. That walk is only complete while
`requireExactMembers` is the sole closed-member gate, so a separate assertion
derives every function in the package that emits a closed-member refusal and
fails unless that set is exactly `requireExactMembers`; a duplicate helper
previously carried whole member sets outside the inventory. The scanned file set
is derived from the package directory rather than a hand-written list, and every
inventoried validator must be reachable from an exported entry point. The one
production call whose member slice is computed rather than literal is the
Session Event payload gate; its members are pinned separately against
`testdata/session-event-payload-members.md`, mechanically extracted from the
pinned specification's `Exact payload members` tables, and any other computed
call site fails until it is declared and given its own pin. The Session Record provenance
fixture inventory also derives every required nested-object path and proves that
both `null` and scalar substitutions are refused by both identity entries.
Manifest entry destination-case
collision detection uses a linear Unicode simple-fold set; a production-entry
regression covers the declared 65,536-entry maximum inside the encoded identity
size cap. Entry-local path OVERLAP is refused by the same scan: an entry whose
ancestor is a declared `file`, `symlink` or `hardlink` is rejected, reusing the
strict bytewise order the scan already proves rather than re-deriving each
path's ancestor set, so a symlink or file parent cannot carry children into
materialization.
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
The AX safe-integer refusal is not part of `Canonicalize` and must not be: RFC
8785 Section 3.2.2.3 serializes numbers through the ECMAScript double-to-string
algorithm, so `Canonicalize` rounds `9007199254740993` to `9007199254740992`,
`18446744073709551615` to `18446744073709552000`, and `1.0` to `1`, exactly as
Appendix B publishes. The specification puts that refusal on the AX decoder
instead: Section 1.6 fixture `NUM-UNSAFE-NUMBER` (SPEC.md:301) requires
rejection "before identity calculation", and `NUM-UNSAFE-ROUND` (SPEC.md:302)
requires rejection "from the JSON number token before conversion to a host
double". So `CalculateObjectIdentity`, `VerifyObjectIdentity`,
`ValidateObservationEvent` and `ValidateObservationStream` reject all three
literals with a typed error while `Canonicalize` documents the rounding as
intended behaviour. The split is pinned rather than asserted in prose: the
`literal -> canonical` rows, the entry-point names and the `SPEC.md:<line>`
citations are parsed out of the `Canonicalize` doc comment and checked against
real behaviour, against the entry-point set derived from the production call
graph, and against the digest-pinned document in `internal/specdoc` — a quoted
fragment must occur there exactly once, begin at the declared line, and, when a
fixture is named, be the row that declares it. The same comment's container
clause is pinned the same way: it bounds containers open at once, not containers
opened, so `Canonicalize` accepts a shallow array of 400 empty arrays that opens
401 containers and refuses 257 nested ones, and each of those rows is built,
measured and driven through the exported entry point. A doc edit without a code
edit, a code edit without a doc edit, a widened or deleted safe-integer bound, a
depth bound restated as a count of containers opened, and a quotation re-pointed
at a neighbouring fixture row each redden the suite.
Every open `extensions` map is validated against the Section 1.6 reverse-DNS
key, member-count, nesting-depth, and canonical-size rules before either entry
attests it. Declared `string[n..m]` bounds count Unicode characters rather than
UTF-8 bytes, including Blob media type, Manifest symlink target, workspace
identity, remote name, and recursive submodule identity boundaries.

Every declared range bound is proven at both of its limits, and the inventory
that requires those proofs derives its own subject rather than listing it. The
bound-helper set is derived transitively from the package sources — a function
declaring a bounded `name` with an int `minimum` and/or `maximum`, plus any
wrapper forwarding that name to one with literal bounds — and every call site of
every derived helper becomes an obligation carrying the member and the literal
bounds. A proof discharges an obligation only by accepting at the limit and
refusing past it through `CalculateObjectIdentity` and `VerifyObjectIdentity`,
or through `ValidateObservationEvent` for the Observation Event bounds, and the
obligation set and the proof set are asserted equal in both directions. Adding a
bound call site, widening or narrowing a bound, or deleting a proof therefore
fails the suite; a review sweep previously found roughly twenty bounds that
widened silently while every configured gate stayed green.

Two obligations cannot be discharged that way and are declared rather than
waved through. The 65,536-entry `GitIndex` bound is refused by the outer 5 MiB
identity-size gate before the declared bound is reached, and a 256-entry nested
submodule array is refused by the whole-tree submodule count first. Each names
its subsuming refusal and the test that pins it, both sets are asserted exactly,
and each named test must exist in the package. Bounds written inline rather than
through a helper — the `opaque_identity` value length and the
`terminal-backend-id` byte count — are proven by their own named tests instead.
No behavioural claim here depends on a mutation sweep; the sweep is an audit of
these gates, not a substitute for them.

Every closed vocabulary the package admits is bound to a reviewed pin in
[`testdata/closed-vocabularies.md`](internal/canonicaljson/testdata/closed-vocabularies.md).
The production side is derived: the vocabulary-admitting helpers are derived
from the sources rather than named — a function that yields the value it
admitted and decides admission from its own variadic set, plus any wrapper
forwarding that set to one — and every call to one of them becomes a row
carrying the member and the admitted values in declaration order. The pinned
side records where the specification declares that vocabulary, by SPEC line and
by quoted declaration, so a row is checkable against the pinned document rather
than against this repository. Those two columns are read and checked rather than
parsed and discarded: the line must be a positive line number, the quoted
declaration must name the member and every admitted value as a whole token, and
two rows quoting the same declaration must cite the same line — so a row widened
with an extra member also has to carry that member into the text a reviewer
reads. This artifact's declarations are still not compared against `SPEC.md`
itself; unlike the constraint enumeration below, they paraphrase the pinned rows
rather than quoting them, so the vendored document in
[`internal/specdoc`](internal/specdoc) does not yet gate them. Admitting an extra member, dropping one, reordering
them, adding an unpinned call site, or deleting a pinned one all fail; so does
adding a second admitting helper, because the derivation asserts its own helper
set is complete.

This exists because a "refuses one outside value" test proves a vocabulary gate
is *reachable*, not that its admitted set is the *declared* set, and coverage
cannot tell the difference — a widened gate still executes its refusal for
whichever outside value a test happens to pick. A derived sweep that admitted one
extra member at each of the 47 call sites in turn survived all 47 times against
the full configured gate set. Binding the admitted set to a reviewed pin is what
makes widening fail, at derivation, before any case runs.

The pin covers the member LIST. The admitted set is that list intersected with
the COMPARISON that decides membership, and a pinned argument list says nothing
about the comparison: case-folding `requireEnum` made `CalculateObjectIdentity`
attest a Lease Record whose `reason` was `CREATE`, trimming made it attest
`" create "`, and relaxing `requireExactString` from equality to a prefix test
made it attest a Checkpoint Record whose `status` was
`validated_but_not_really` — each with the whole configured gate set green. The
second factor is pinned separately. Every function that decides a string
member's admission from a caller-supplied parameter is derived from the sources
rather than named, its call sites are resolved to literals through package
constants and through forwarding wrappers up the real call graph, and each of
the 74 resulting sites is located in the valid fixtures by behaviour: a site
binds to a position only when every one of its declared members is admitted
there. Each bound position is then attacked at `CalculateObjectIdentity` and
`VerifyObjectIdentity`, or at `ValidateObservationEvent`, with a family built
from the admitted value itself — case variants, leading and trailing whitespace,
a proper prefix, a proper suffix, a proper superstring, and an embedded NUL. A
site that binds nowhere is reported, which is also the signature of a narrowed
matcher. What this does not prove is a matcher that admits one arbitrary extra
string unrelated to any declared member; that space is unbounded and no finite
family reaches it.

Every regular expression the package compiles is pinned the same way, because a
grammar has the identical failure mode: widening `observationNamePattern` from
`{1,7}` to `{1,8}` segments, to `{0,7}` so a bare single-segment name is
admitted, or adding a hyphen to its character class each left the whole gate set
green. Each production pattern carries a reference written independently of the
source, its pinned SPEC declaration or an explicit statement that the document
declares none, and the refusal it emits; the reference is what every oracle in
that file reads, and production must equal it exactly. Anchors, character
classes, counted quantifier bounds and one-or-more quantifiers are then derived
from the reference as obligations, each discharged by a witness that the
production entry must refuse and that the mechanically widened grammar must
admit — so a witness which could not fail against a widened pattern is rejected
before it is trusted. Counted bounds are asserted against the number the pinned
specification declares, never the implementation constant, and the one bound
that cannot be reached — `boardLogicalIDPattern`'s repetition, refused first by
the declared `logical_id` string bound — names its subsuming refusal and the
test that pins it. On top of that, every one-character neighbour of the value
each fixture already carries at a located position — each position substituted
by each of twelve character families, each family inserted at each boundary, and
each position deleted — is driven through the production entry, so widening any
class by any family moves a neighbour into the admitted set and fails.

Every refusal this package can emit at runtime must have been executed by the
shipped suite, and that is observed rather than claimed. The obligation set is
derived from production the same way the bounds inventory is: every
refusal-emitting return in the package's sources, with the refusal constructors
themselves derived rather than named. The gate then runs the shipped suite in a
child process under a statement-coverage profile and requires each derived site
to carry a non-zero execution count. A guard added without a negative case, a
deleted negative case, an unreachability claim that becomes reachable, and a
declaration naming a guard that no longer exists all fail the gate.

This exists because a percentage is not evidence. At 87.7% package coverage,
eight normative gates could be deleted from the core-record validators
simultaneously with the whole configured gate set — `go test ./...`, both seeded
fuzz corpora, and `tracecheck` — still green, because 91 runtime refusal
branches in that one file were never executed once. The cause is structural:
`requireExactMembers` runs first in every validator, so a negative fixture that
omits a member stops at the closed-member sweep and never reaches the type,
format or coupling refusal it claims to pin. Every negative case here therefore
supplies a complete valid member set and violates exactly one clause.

Reaching those refusals is itself derived rather than hand-listed. Two sweeps
walk every value position of every valid fixture: one substitutes a value of the
wrong JSON type, and one corrupts every structurally-shaped value — digest,
UUIDv7, UUIDv4, timestamp, git OID, sanitized git URL, semver — classified by
the production `internal/scalar` parsers rather than by member name. The fixture
set is itself an obligation: every schema/version registered to a validator that
can accept anything must have a valid fixture, with the exempt total-refusal
validators derived from the sources rather than excused by name.

Members the pinned specification declares by name only are declared exemptions
quoting the clause, not silent acceptances. `EnvironmentTuple`
`environment_version` and `store_schema_fingerprint` carry neither a type nor a
format at the pinned commit while their siblings carry explicit ones, so
requiring a refusal there would invent a constraint. Both exemption sets are
asserted exactly in both directions, so an exemption that stops being true fails
as loudly as a missing case. 55 obligations that no candidate can reach through
any production entry — overwhelmingly the "requires member" branches that
`requireExactMembers` short-circuits — are declared with a reason, the refusal
that subsumes them, and a test that must exist in the package.

Two clause shapes are invisible to a deletion or operator-rewrite sweep, and
this leaf shipped one live instance of each. A coupling written as a single
boolean comparison — `requiresError != errorPresent` — was proven in one lexical
direction only: narrowing it to `requiresError && !errorPresent` left the whole
configured gate set green while `ValidateObservationEvent` attested a `success`
Observation Event carrying a non-null `error_code`. There is no operator to
rewrite there, only a boolean identity to split. An integer comparison against a
literal — `epoch != 1` — was proven only at epoch 4: narrowing it to
`epoch >= 3` left the gate set green while `CalculateObjectIdentity` attested an
epoch-2 Lease Record with a null predecessor, and `epoch != 1` narrowed to
`epoch > 1` is an equivalent mutant, so no deletion or operator sweep could ever
have surfaced it.

Both classes are now standing gates with derived subjects. Every `==`/`!=`
whose two operands are boolean-valued is derived from the package sources —
boolean-valued decided from the source, a negation, a nested comparison, a
short-circuit, or an identifier the enclosing function binds from a package
function whose result at that position is declared `bool` — and each site
obliges two proofs, one per single-sided violation, with the pair chosen by the
operator rather than by hand. Every comparison between an integer literal and
anything else in `core_records.go` obliges a proof at each value where the
comparison flips, read off the operator, with values below zero dropped for a
length, a range index, or an unsigned local. The obligation key carries the
literal, so moving a literal fails the gate before any case runs. Each proof
drives `CalculateObjectIdentity` and `VerifyObjectIdentity`, or
`ValidateObservationEvent`/`ValidateObservationStream` for the Observation
Event. Three boundary values cannot be driven — a zero `sequence` and a zero
`epoch` are refused by the positive-integer gate first — and each names its
subsuming refusal and a test that must exist. The one derived function whose
comparisons run at package initialization over the pinned catalog rather than
over a candidate is exempted by name, and the exemption asserts from the sources
that every caller of it is a package-level initializer that panics, so a new
caller from a validator reddens the gate.

An audit sweep generates its mutants from that same derivation rather than from
a hand-picked list: `TestDumpSweepSites` writes each derived site with its exact
source byte range when `AX_SWEEP_DUMP` is set, and skips otherwise, so an
external harness can only mutate what the gates themselves derive. Forty mutants
in the two grammars — both single-direction splits of every derived coupling,
and literal shifts an operator-rewrite grammar cannot express — were killed with
no survivors. `x >= K+1` is deliberately not generated for `x != K`: in a
non-negative domain whose minimum is `K` it is an equivalent mutant, and
reporting an equivalent mutant as a kill is how an earlier sweep on this leaf
reported strength it did not have.

The literal-boundary gate is scoped to `core_records.go`, this leaf's
deliverable. `canonical.go` and `closed_shapes.go` belong to the preceding
leaves; their comparisons were measured and disclosed rather than claimed. The
presence-coupling gate is package-wide, because a boolean-valued equality is
rare enough — eight sites — that scoping it would have left a known instance of
the same class unproven in a file this package ships.

Section 5.3 declares three Lease Record couplings and this package enforces
exactly those three: a lease after epoch one carries a non-null predecessor, an
epoch-one `create` lease carries a null predecessor, and every lease other than
an epoch-one `create` carries a non-null checkpoint. Section 5.3 declares no
coupling from the epoch to the `reason`, so an epoch-one `recovery` lease and a
`create` lease above epoch one are both admitted; the executable suite pins that
permissiveness so the inference cannot return.

Array order constraints are mapped from the specification's phrase to the
validator that enforces it, mechanically, in
[`testdata/array-order-constraints.md`](internal/canonicaljson/testdata/array-order-constraints.md).
Section 1.6 defines `sorted unique T[n..m]` as the compound phrase meaning
"bytewise canonical ordering and no duplicate", and the pinned document uses
bare `sorted` where it declares ordering alone. Reading the two as synonyms
shipped two live refusals the contract does not declare: Session Event
`predecessors` and Workspace Group Record `members` were both validated with the
strict comparison, so each refused a duplicate the contract admits. Both are
repaired, and both directions are now driven at `CalculateObjectIdentity` and
`VerifyObjectIdentity` — a descending pair refused, a duplicate admitted — with
the Session Event version set taken from the pinned catalog.

The production side is derived, not listed. An ordering site is found by tracing
the loop's element dataflow: a comparison between a value derived from the
ranged element and a value bound to an earlier element, either carried forward
or indexed at an offset from the loop key. Its strength is read off the
comparison OPERATOR, which is the only thing that decides whether a duplicate
survives. The array member is derived too, by tracing the ranged collection back
to the string literal that named it, through helper parameters and across call
sites, so a reusable ordering helper contributes one row per member it actually
orders. A row whose `Enforces` is `sorted unique` must cite a uniqueness
declaration; a row whose `Enforces` is `sorted` must cite none. A strengthened
validator therefore cannot be written down consistently: its row would have to
quote a uniqueness clause the document does not contain.

The gate proves its own coverage three ways. An ordering site whose member
cannot be traced is reported rather than silently contributing no row, which is
what a new ordering helper with no call site looks like. Every production
refusal whose message speaks about order must belong to a function carrying a
derived site, so a check written in a shape the tracer does not model reddens
instead of passing unpinned. And the phrase-to-validator mapping is a pure
function with its own negative proof, driven with rows wrong in each way it
claims to detect and asserted against the exact problem each must produce, so
narrowing one clause cannot pass on a neighbouring clause's refusal.

Three ordering sites enforce uniqueness the pinned document declares nowhere.
They are recorded as disclosed strengthenings rather than admitted silently, the
disclosed set is asserted exactly against the artifact in both directions, and
each names the leaf that owns the section it sits in. One — Workspace Group
Record `members` — is in this leaf's Section 2 and was repaired here. The other
two are `WorkspaceSnapshot.members` and `GitIndex.entries`, in sections owned by
a later leaf; they are reported with their exact undeclared refusal rather than
changed inside a reviewed and accepted candidate.

Six record-level conformance dimensions are swept across every registered
shape rather than one record at a time, in
[`record_conformance_test.go`](internal/canonicaljson/record_conformance_test.go).

Round trip drives `Canonicalize` as the storage form for every identity fixture:
canonicalization is idempotent, reading the canonical bytes back through
`CalculateObjectIdentity` reproduces the digest, and `VerifyObjectIdentity`
accepts the canonical claimed record. Its negative half is the Section 1.5
sentence "a malformed value remains invalid after a caller recomputes the
containing object's self-ID", driven through both the recomputation and the
canonicalization a writer could use to launder it; its anti-degenerate bound is
that the fixture digests are pairwise distinct, so a calculation that ignored
content would fail rather than satisfy every comparison.

Unknown-field closure binds both halves of the Section 1.5 extension boundary at
one call. The same reverse-DNS key with the same value is refused at the top
level and accepted under `extensions`, so neither a validator that rejects it
everywhere — which would break the only declared forward-compatibility channel —
nor one that accepts it everywhere passes.

Historical-major retention compares two independently generated release
projections: every contract version the v0.4.3 Section 1.5 registry declares must
still be declared by v0.5.0. The detector is a pure function with its own
negative proof, driven with a dropped version, a dropped contract, and a
purely additive registry, and asserted against the exact problem each must
produce. The opposite direction is checked too, because a historical projection
that silently returned the current registry would make every retention assertion
vacuous. Every historical-major fixture is then accepted through the identity
entries under its own registered version, and one major above the highest
registered version of each schema is refused.

Union closure walks every closed-vocabulary position of every fixture and feeds
it the union of every other pinned vocabulary. The pinned inventory kills a
widening mutant at the source; it does not establish that a JSON path in a real
record routes to the vocabulary its row names. A validator reading a Lease Record
`reason` through the `session.quiescing` reason gate would leave every pinned row
unchanged and admit `stop` on a lease, and that mutant is refused here. Each
position's declared set is derived from the single pinned row whose member and
values contain the fixture value; the four positions with two genuinely
different candidate rows carry a reviewed resolution that must remain one of the
candidates, so a widened production vocabulary invalidates the resolution rather
than being absorbed by it.

Provenance is swept per Section 10.1 family rather than per record. The family
list is the section's own sentence, and the families without a complete shape
validator are derived out by walking the production sources rather than excused
by name, so a Tombstone validator landing later enters the sweep automatically.
Each of `subject_id`, `created_by_host_id`, `created_at`, and `extensions` is
driven absent, null, wrongly typed, empty, and — the one that matters — carrying
a value of the right JSON type in the wrong identity grammar: a UUIDv4 stamped
into a UUIDv7 member is the shape a forged or copied provenance stamp takes, and
a check that only asks whether the member is a string admits it.

Cross-record references are resolved rather than pattern-matched. One coherent
lineage is built — session record, provider identity, provider and workspace
manifests, an epoch-1 create lease, the `session.created` event, the checkpoint
closing over that event head and both manifests, and the epoch-2 graceful
takeover whose handoff base is that checkpoint — with every cross-record member
holding the recomputed omit-self digest of the record it names. Each reference is
then resolved through a content-addressed store keyed by recomputed digest and
must land on the record it names. The negative half replaces one referenced
record with a different, individually valid record of the same schema, kind, and
subject, differing only in a diagnostic timestamp: under the "untrusted display
name" storage path Section 10.1 forbids, the substitute would keep the same
address, and here it must not resolve and swapping the reference must invalidate
the referring record's own claim. The in-object couplings those records carry —
subject/session scope, the lease issuer, the checkpoint's exactly-one manifest
rule — are already pinned by the derived coupling inventory and the clause
refusal proofs and are not restated.

These sweeps are audited by a recorded mutation run over the production sources.
Five mutants — a lease vocabulary widened with another record's value, a
`created_by_host_id` admitted as any string, an unknown reverse-DNS top-level
member retained, the self field included in its own digest, and a historical
release projection serving the current registry — are each refused by these
tests. A deleted checkpoint scope coupling survives them and is refused by the
existing coupling inventory, which owns that clause. A seventh mutant, skipping
the `created_by_host_id` check when the member is absent, survives both these
tests and the whole package suite: it is an equivalent mutant, because every
Section 10.1 validator calls `requireExactMembers` with its closed member list
before the envelope runs, so the absent-member path is unreachable at the
production entry. That subsumption is pinned rather than assumed — the sweep's
absent case asserts the refusal names the missing member — and the reading is
recorded rather than presented as a pass.

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

This package is read-only and deterministic; it does not mutate durable state,
so crash recovery and mutation idempotency are not applicable to these entry
points.
It supplies identity calculation for a new schema-versioned object under
Section 17.3, but does not claim to implement migration publication, atomic
reference advancement, rollback retention, `ax migrate`, `ax doctor`, Session
creation, clone, adoption, or any runtime capability. Schema acceptance alone
does not advertise those operations. Those surfaces remain unavailable until their owning
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
go test ./internal/canonicaljson -run=^$ \
  -fuzz=^FuzzObservationEventRefusal$ -fuzztime=100x -parallel=1
go run ./internal/traceability/cmd/tracecheck
```

Section 13.14.5 is refused at 0/0: the obligation scanner measures no clause
line under it, because the section states its obligations as required-member and
variant tables, so the gate cannot measure how much of it
`validateSessionEventV2` discharges. Sections 2.1, 2.3, 2.4, 5.1-5.6, 10.1, 13.15, 17.3 and 18.1 are
refused with their measured ratio, and Section 2.2 is refused as recorded
unowned. See [Specification-to-Code Ownership
Gate](#specification-to-code-ownership-gate).

## Safe Local Layout, Immutable Blob Sink, and SQLite Projection

[`internal/localstore`](internal/localstore) loads the reviewed five-row Path
Registry 1.0.0 bound to the pinned v0.5.0 Section 3.2 source. `ResolvePaths`
applies flag, non-empty `AX_*` environment, then platform-default precedence;
unknown `AX_*` variables remain ordinary environment. It resolves macOS,
Linux/WSL2, and native-Windows path syntax without consulting ambient process
state; Windows environment names are matched case-insensitively and unavailable
`APPDATA`/`LOCALAPPDATA` defaults fail closed. The future CLI/config owner must
capture those inputs once and retain the resulting roots for the process
lifetime.

`InitializeLayout` creates or verifies AX-owned configuration, data, state,
cache, and runtime roots as mode `0700` directories on native macOS/Linux, and
accepts only owner mode `0600` for an existing configuration file. Native
Windows path resolution and golden `digest_path_v1` projection are available,
but filesystem initialization refuses until the user-only DACL implementation
and Windows conformance lane land; lexical path evidence does not advertise
Windows storage support.

`OpenObjectStore` keeps the live root host-local. `PutBlob` writes a mode `0600`
same-directory temporary file, fsyncs it, verifies the declared uint53 size and
SHA-256, performs an OS-native atomic no-replace rename, then fsyncs the shard
directory. An identical retry verifies and reuses the existing inode. Size or
digest mismatches of the source bytes are installed create-new under the exact
quarantined digest path with a UUIDv7 leaf. Oversized input is read and
quarantined only through the bounded declared-size-plus-one diagnostic prefix. A
corrupt existing digest path causes both candidates to be quarantined and the
immutable namespace to refuse the write.

Quarantining an entry that already occupies a digest path requires a proof.
`verifyBlobContent` is the one classifier both `PutBlob` and the projection
source scan consult, and only a completed read that disagrees with the declared
size and digest authorizes a move. An inspection whose read did not complete —
descriptor exhaustion, media failure, a failed close — is reported as a
durability failure with the existing object left exactly where it is, because a
read that never finished proves neither a hash mismatch nor a representation
disagreement. `storeOperations.openExisting` and `projectionHooks.openBlob` are
the injected read seams that drive that path in tests, and the two paths are
asserted to reach the same classification rather than assumed to.

The same rule applies to the write side, and it is a separate refusal rather
than a variant of the mismatch. When the length the filesystem kept disagrees
with the byte count the staged copy accepted, the write did not complete and
therefore proves nothing about the source bytes: `PutBlob` reports a durability
failure and the partial candidate is discarded rather than quarantined as
disagreeing evidence. Quarantining it would misattribute the fault to a source
that may have been valid, and would consume more of the space whose exhaustion
typically caused it. That comparison is the only thing between a partially
written file and the immutable namespace when the declared digest and the copied
length still agree with the caller, so it is pinned in both directions: a
fixture that keeps less than the copy accepted and one that keeps more.

Full and failing volumes are driven at every boundary of the durable write. The
source that stops producing, the volume that refuses the accepting write, the
volume that discovers exhaustion at writeback, the staged file whose mode
drifted before installation, and the shard directory that is group-accessible, a
symlink, or not a directory at all are each refused with the immutable digest
path left absent, no staged residue beside live objects, nothing manufactured
into quarantine, and a retry after the fault clears installing the exact
declared bytes.

The quarantine move that cannot itself be written is a separate outcome, stated
separately because the artifact the move was refused for is precisely the one
that must not vanish. When the mismatched candidate cannot be moved, nothing is
installed and nothing is quarantined, and once the volume has room that same
candidate is preserved as evidence exactly as it would have been without the
fault — recovering from the fault does not relax the verdict that produced it.
When the candidate was preserved but the corrupt object already occupying the
digest path could not be moved, that object deliberately stays where it is and
the caller is told which artifact was preserved and which was not. A later run
then reads it, proves the disagreement, quarantines both artifacts, and installs
the declared bytes, so the denied move defers the cleanup rather than stranding
the digest path. A quarantine that failed is never reported as one that
succeeded.

A digest path replaced by a special file bounds availability as well as safety.
The regularity clause is driven with a FIFO created at exactly mode `0600`,
which is not a symlink, belongs to the effective user and already carries
owner-only bits, so every preceding check passes and that clause is the only
thing that can refuse; with it removed `os.Open` blocks indefinitely on a FIFO
that has no writer, so the case asserts a bounded return rather than hanging.
The refusal moves the entry nowhere, because an unsafe shape is not a proven
mismatch.

`OpenProjection` scans the immutable raw-blob namespace, re-verifies every byte
against its digest path, and rebuilds `<state>/index.sqlite` in deterministic
digest order. That order is owned by one comparator and asserted against the
scan itself rather than through a read query that re-sorts, and the recorded
`source_fingerprint` is checked against an expectation derived outside the
implementation, so neither a constant fingerprint nor a re-ordered scan passes.
The schema and its indexes come from one closed inventory.
The integrity gate derives SQLite-created internal indexes by applying that
catalog to a clean in-memory database, then compares every `sqlite_master` row;
no object kind or `sqlite_` name prefix is filtered out of verification.
Migrations and full row replacement run in transactions, with injected
pre-commit failures proving rollback to the prior usable index. A full volume is
driven as the engine failure it actually is rather than as a substituted error
value: `PRAGMA max_page_count` pinned to the database's current size makes
SQLite raise `SQLITE_FULL` from inside the migration and the rebuild
transactions. Exhaustion is classified as a migration or rebuild failure and
never as corruption, so the index is not quarantined and rebuilt on a volume
that just ran out of room; the prior usable index keeps its inode, its rows and
its recorded object count, no `index-recovery` directory appears, and the next
open on a volume with room migrates and rebuilds as if the refused one had never
run. Connections use
WAL journal mode, `synchronous=FULL`, foreign keys, defensive mode that refuses
`sqlite_master` writes on the live connection, and a bounded busy timeout;
an owner-only `index.sqlite.lock` serializes cold open, migration, rebuild, and
recovery across local processes, and is itself refused when it is not a regular
owner-only file. The authoritative scan runs before that lock is taken, so
concurrent opens may scan different source snapshots and the later committer may
write the older one; the index is a derived cache with no advertised freshness
guarantee and the next open converges on the current source. The database, lock, and any WAL/SHM/journal
sidecars are checked before SQLite opens either a new or existing index, remain
mode `0600` under the owner-only state root, and never appear beneath
transferable durable data.

SQLite integrity or closed-schema drift moves the database and its sidecars
create-new into `<state>/index-recovery/<uuid>/`, then reconstructs the cache
from authoritative blobs. A newer implementation schema is refused without
rewriting it. Missing object roots mean an empty source, while malformed,
unreadable, unsafe, or hash-mismatched source entries fail closed before SQLite
is touched. Projection recovery never rewrites or deletes an authoritative
blob.

The package still does not install JSON/CBOR records or manifests, publish an
`ax doctor` result, or advertise a runtime capability. Its immutable sink and
projection isolation are prerequisites for Section 18.4 retention, not a claim
that Session Event, Tombstone, Acknowledgement, or audit-chain retention
semantics are implemented.

Run the focused storage and assigned-scope gates with:

```bash
go test ./internal/localstore -count=1
go test ./internal/localstore -cover -count=1
go run ./internal/traceability/cmd/tracecheck
```

The assigned-scope gate refuses Sections 3.2, 3.3, 10.1 and 10.2 with their
measured ratio, and Section 18.4 as recorded unowned: no retention or
garbage-collection rule is implemented. See [Specification-to-Code Ownership
Gate](#specification-to-code-ownership-gate).

The unfiltered localstore package run derives the projection refusal inventory
from the production source, and it checks completeness in both halves.

Every refusal raised through a refusal funnel must have a negative path that is
reached beneath `openProjection`, so a unit test calling a refusal helper
directly no longer counts as proof that the production entry still consults it.
A site is exempt only when its line carries an inline statement naming the
exact subsuming check, or naming its production call site together with the
reason `OpenProjection` cannot reach it.

Because that first half is only as complete as the funnels are exhaustive, the
same gate also refuses refusals written outside them: a projection-owned source
may not wrap `ErrUnsafeOwnership` with a bare `fmt.Errorf`, and an owner-only
guard around `verifyOwnerFileInfo` or `verifyOwnerDirectory` must re-raise
through a funnel. Only files the current build context compiles are considered,
so a platform-specific refusal is never demanded from a platform that cannot
execute it.

Reaching a refusal clause and pinning it are different facts, and the derived
inventory can only prove the first. A fixture that also trips an earlier check
exercises the site while some other clause decides the outcome, so the shape
clauses that refuse a non-regular index, sidecar, blob shard, staged blob, or
blob leaf are driven with a FIFO created at exactly mode `0600`. Such a fixture
is not a symlink, is owned by the effective user, and already carries owner-only
permission bits, so every preceding check passes and the shape clause is the
only thing that can refuse; each case asserts that isolation before it drives
`OpenProjection`. Those clauses bound availability as well as safety: with one
removed, `os.Open` and SQLite block indefinitely on a FIFO that has no writer,
so the cases assert a bounded return rather than hanging. Mutual exclusion of
`index.sqlite.lock` is pinned separately from its ownership by holding one open
file description while a second `openProjection` must not reach its rebuild
transaction.

The same isolation requirement applies to a multi-term comparison, where the
earlier check that decides instead of the target one is a sibling term rather
than a preceding line. Each identity term of the closed-schema comparison is
therefore driven by a drift that no other term can observe: a definition-only
rewrite for the SQL comparison, and a `writable_schema` move of an autoindex
row's `tbl_name` for the table comparison, which SQLite accepts and reopens
while leaving the `(type, name)` key set and every SQL text unchanged. Each
case asserts those other terms still match before it drives `OpenProjection`.
Two terms carry no refusal of their own and say so at the call site instead:
`kind` and `name` are subsumed by the inventory key lookup, and the
presence-of-SQL term by SQLite's own schema parser, whose two reachable drift
directions make the database unopenable and route to the pinned corruption
path. The exact-name closure over the objects root is driven with a near-miss
sibling name as well as a foreign one, because a closure narrowed from equality
to a prefix admits exactly the near miss.

## Structured Errors, Stable Codes, and Causal Redaction

[`internal/axerror`](internal/axerror) implements the Section 15 Structured
Error contract: the closed versioned failure object, its stable code-to-exit
registry, its retryability rule, its typed diagnostic details, and the redaction
that keeps a local cause off the wire. The package holds no state, opens no
file, starts no process, and mutates nothing durable, so it has no crash or
idempotency surface and advertises no provider, platform, backend, or CLI
capability.

| Element | Where it is decided |
| --- | --- |
| Versions `1.0.0`, `1.1.0`, `1.2.0`, `1.3.0` | `Version` constants; any other value, including major 2, is refused rather than coerced |
| Code-to-exit registry | `ExitCodeFor`, projected from the reviewed catalog rather than retyped, and admitting a code only for the versions that register it |
| Exit-status registry | `ExitStatusMeaning` and `IsFailureExitStatus`, the closed Section 15.2 table, whose eighteen rows are measured from [`internal/specdoc`](internal/specdoc) rather than restated in prose; success is never a failure class |
| Containing-contract binding | `BindingFor`, a static table with no negotiation path; `DecodeBound` takes the version from the container, never from the payload |
| Retryability | `RetryabilityRefusal`, quoting the clause that disqualifies each exit class and each individually named code |
| Typed details | `TargetAuth` and `RealmEvidence`, plus a presence requirement on `target_auth_missing` that the generic constructor cannot bypass |
| Redaction | `ValidateDetails` exact-key scanner over the four Section 15.1 detail classes, and `refuseCausalLeak` over the local cause chain |
| Detail ownership | `New` deep-copies the caller's diagnostic graph and `Detail` deep-copies on the way out, so the graph `ValidateDetails` checked is the graph the object encodes |
| Untrusted boundaries | `LocalFromUntrusted`, which takes no part of a child or peer payload |

The exit status is never a constructor argument. `New` takes a `Spec` with no
`exit_code` field and resolves the status through `ExitCodeFor`, so a call site
cannot mint a mapping the specification does not assign. The reader is the
mirror image: a code the registry does not carry is admitted, because Section
15.3 permits a code added in a compatible minor, but it keeps its envelope's
exit class, is reported by `CodeRegistered` as unrecognized, and can never carry
the success status.

Redaction is two decidable gates and one structural property, and none of them
is a confidentiality guarantee:

- a detail key that **exactly** names one of the four classes Section 15.1
  forbids - credential, raw transcript, environment secret, opaque bundle
  content - is refused wherever it appears, including inside nested diagnostic
  objects;
- human text or a diagnostic value that reproduces the rendered local cause
  verbatim is refused, which is the `fmt.Errorf("...: %v", err)` accident;
- the cause itself is unexported and unreachable from the only encoder in the
  package, so it cannot be serialized by any future field addition.

Matching is exact and never by substring. A substring rule refusing every key
containing `token`, `socket` or `credential` would refuse `token_count`,
`socket_timeout_ms` and `credential_profile` - ordinary diagnostics - while
still admitting a secret written under an innocuous name: false-positive surface
with no true-positive capability, which is the defect `BUG-260902-2faftr`
removed from the Configuration extension-key validator. The scanner is likewise
scoped to Section 15.1's four classes rather than to the whole Section 16.2
exclusion table, because a diagnostic key naming a PID, a tmux socket or a
Terminal Instance Binding is none of those four.

A validated failure owns its diagnostic graph outright, and that is part of the
gate rather than a style choice. The first revision of this package copied only
the top level of the details map and handed the live nested container back from
`Detail`; every bound above - the four forbidden classes, the 16 KiB canonical
size, the depth-4 nesting limit, the admitted value types - could then be
violated after `ValidateDetails` had already run and passed, and the package
emitted objects its own `Decode` refused. A validator that checks a graph the
value does not own is a bypass path, not a gate, so `New` deep-copies on the way
in and `Detail` deep-copies on the way out.
`TestConstructionDoesNotAliasTheCallerDetailGraph` and
`TestDetailAccessorDoesNotHandOutTheLiveContainer` attack containers at three
depths and on both sides of an array, so a copy narrowed to one level fails them
rather than only a copy deleted outright.

`RedactionBound` states the limit in the code, and the tests assert that the
constant still says it. Section 16.2 is explicit that v0.5.0 "does not claim
reliable content-level secret scrubbing" while an implementation "SHOULD offer a
best-effort scanner"; this is that scanner and nothing more. A secret placed in
an innocuously named string value is admitted, and that is a stated bound rather
than a discovered one.

Two mappings the pinned document leaves open are reported as unknown rather than
guessed. `LocalFromUntrusted` refuses the Directory Node surface, because
Section 15.3 says its local code is `incompatible_protocol` or
`adapter_protocol_violation`/`transport_failure` "as applicable" without fixing
which; and `RetryabilityRefusal` never reports that a retry is safe, only where
the document forbids the claim.

`CodesByExitStatus` projects the registry the other way round, from exit
status to the codes that map to it, because that is the direction a machine
client reads it in. The measured fan-in is the reason an exit status is never a
failure identity:

| Structured Error version | Failure statuses | Registered codes | Statuses carrying more than one code | Largest class |
| --- | ---: | ---: | ---: | ---: |
| `1.0.0` | 17 | 47 | 14 | 6 codes at exit 6 |
| `1.1.0` | 17 | 66 | 14 | 9 codes at exit 6 |
| `1.2.0` | 17 | 94 | 15 | 14 codes at exit 16 |
| `1.3.0` | 17 | 109 | 15 | 17 codes at exit 6 |

Every cell of that table is re-derived from `CodesByExitStatus` by
`TestREADMEFanInTableIsDerivedFromTheMeasuredProjection`, which parses these rows
out of this file, requires every registered version to appear exactly once, and
refuses a version whose maximum is tied across two statuses because "the largest
class" would then name no single status. It exists because two of the four rows
were wrong when they were first published - `1.2.0` said 12 codes at exit 6
against a measured 14 at exit 16, `1.3.0` said 15 against a measured 17 - and
survived review because the largest class had been asserted for `1.0.0` and
`1.1.0` only. The same derivation covers the two figures Logbook entry 1003
states in prose.

`testdata/historical` holds one frozen envelope per bound version, each read
through a containing contract that binds it, with its bytes pinned by SHA-256.
They are checked-in bytes rather than objects a test builds: an envelope
regenerated by today's constructor would only show that this package agrees with
itself. `TestABoundReaderNeverAdoptsThePayloadDeclaredVersion` offers every
frozen envelope to every containing contract that binds a different version and
requires each cross pair to be refused, so a peer cannot select its own error
version by writing one into the payload.

Every gate above has a negative test that narrows it rather than deleting it:
each forbidden retryability class and each individually disqualified code is
exercised separately, each excluded detail class is refused on its own row, each
declared bound is refused one step past the limit and admitted exactly at it,
and each bootstrap mapping is pinned to its literal code.

That sentence was published before it was true of the reader. The
code-to-exit-status agreement in `decodeBody` and the registered-status
admission in `decodeExitStatus` both had sampled coverage that a narrowing
survived, and the first of the two was a reachable bypass of the exit-keyed
retryability refusal. Both are now swept over their whole domain - every
registered code of every registered version at every registered failure status,
and every exit status in 0..255 - and the mutants are in the harness. The
measurement, the bypass, and the sweeps are described under [CLI Result
Envelopes](#cli-result-envelopes-rendering-boundaries-and-exit-status), because
the reading path they protect is the one a machine client calls. Run the suite
with:

```bash
go test ./internal/axerror -count=1 -v
```

## CLI Result Envelopes, Rendering Boundaries, and Exit Status

[`internal/cliresult`](internal/cliresult) implements the Section 14.2 CLI
Result contract: the independently versioned closed success envelope, the
static per-command version selection, the closed embedded types and tagged
command bodies, the Section 17.2 reader rules, the stdout/stderr rendering
boundary, and the exact Section 15.2 status a failure carries. The package
holds no state, opens no file, starts no process, and writes only to the
streams a caller hands it, so it has no crash or idempotency surface and
advertises no provider, platform, backend, or CLI capability.

Success and failure are different objects rather than two shapes of one object.
Section 14.2 says "failure output is one Structured Error object from Section
15.1, not a CLI Result with `ok = false`", so `Result` can only hold an object
whose `ok` is the literal true, and every failure path takes an
[`axerror.Error`](internal/axerror) instead.

| Element | Where it is decided |
| --- | --- |
| Versions `1.0.0` and `2.0.0` | `VersionForCommand`, a static per-tag table; `3.0.0` and `4.0.0` are registered and refused with `ErrUnimplementedVersion` |
| Closed envelope | `New` and `Decode` over exactly the eight declared members; a ninth is refused, and a required `T\|null` identifier must be present |
| Tagged bodies | `validateBody`, one closed validator per Section 14.2 command tag, plus the stop tuple, materialization, and takeover cross-member rules |
| Identifier nullability | the reviewed `commandRegistry` columns; `idUnconstrained` where Section 14.2 names neither set |
| Session scope | `validateNestedSessionScope`, comparing the top-level `session_id` to every nested Session Summary and to a bare body `session_id` |
| Reader rules | `Decode` settles schema and major before anything else, the closed member set included; `acceptsVersion` implements the same-or-lower-minor rule |
| Common flags | `ParseCommonFlags`, the exact ten flags plus the two refusals Section 14.2 states |
| Confirmation | `RequireConfirmation`, which checks expectation flags before and independently of `--yes` |
| Rendering boundary | `Emitter`, one JSON document on stdout, logs and prompts on stderr, progress only on a TTY |
| Exit status | `ExitStatus` and `Emit`, returning the Structured Error's own `exit_code` unchanged |

One design decision removes a defect class rather than documenting it: the
writer and the reader share one validator over one value model. `New` marshals
the caller's body, runs it through the strict
[`canonicaljson`](internal/canonicaljson) model, and re-parses it, so the graph
the validators checked is the graph the object encodes and the writer cannot
emit an object its own `Decode` refuses.
`TestEveryImplementedCommandRoundTripsThroughItsOwnReader` drives that for all
eighteen tags. The caller's value is walked first, because `encoding/json`
replaces invalid UTF-8 with U+FFFD instead of failing and encodes a Go float as
a JSON number Section 1.6 forbids; substituting a replacement character for a
byte the caller supplied would silently change data the specification requires
to be valid UTF-8.

The vocabularies are projected rather than retyped where the repository already
reviewed them. The SessionSummary `capability-name` set comes from the
generated catalog's provider family and is asserted to be exactly the seven
names the Section 14.2 `[0..7]` bound implies; the `peer.probe` contract map is
restricted to the fourteen Section 11.2 hello keys, which is how that section's
"Structured Error, Observation Event, and CLI Result MUST NOT appear in this
map" is enforced rather than merely intended.

Where Section 14.2 states no rule, this package states none. `attach` and
`pane` appear in neither of the two `operation_id` sets the section names and
neither is a pure read, so their nullability is unconstrained; PathDiff and
CLIFinding carry no ID, so the "sorted bytewise by that ID" rule does not reach
them and no ordering is imposed on them. Inventing a sort or a nullability rule
would refuse conforming documents just as surely as omitting one admits
non-conforming ones.

`ContractBound` states the limits in the code, and the tests assert that the
constant still says them:

- CLI Result 3.0.0 and 4.0.0 are registered by the pinned Section 1.5 row and
  not built here; their tags are refused, never emitted with an unchecked body.
- The eight `session.clone.*` tags select CLI Result 2.0.0 - that selection is
  the Section 14.2 rule this package implements - but their Section 14.1 closed
  bodies over the Section 13.14 clone types are not built, so the version has a
  selection and no producer.
- The takeover adoption rule needs a session kind the body does not carry.
  `New` requires it as a constructor argument; `Decode` cannot have it and says
  so through `VerifyTakeoverAdoption` rather than skipping a MUST silently.
- An absolute-path member is admitted when it is absolute on any supported
  platform, because a CLI Result names none. `VerifyDestinationPlatform` is the
  narrowing hook for a caller that knows the emitting host.
- The Section 18.1 total order of a `logs` event array is not checked: it is a
  property of a durable stream rather than of one array.

The measured inventory is 18 of 44 registered command tags, 2 of 4 registered
versions, and 29 user command surfaces of 31, all asserted as ratios rather
than described in prose. Section 14.2 coverage is 8 of its 9 normative clauses;
clause `14.2#6` requires the process exit status to equal the failure's
`exit_code`, and while `Emit` returns exactly that status, no `ax` binary
exists to call `os.Exit` with it, so the process-level half is disclosed as
unowned rather than claimed.

Every gate has a negative test that narrows it rather than deleting it: each
stop-tuple condition is flipped on its own row, each materialization success
rule is violated separately, the takeover adoption rule is exercised for both
session kinds in both directions, every body member is driven with every wrong
JSON type, and each expectation flag is withheld individually. A 50-mutant
harness over the package's gates kills 48 of 48 non-subsumed mutants, each
mutant verified applied and compiled before it is measured; the two survivors
are declared subsumed with the guard that refuses the same input earlier.

### Which refusal wins when two apply

A document of a major this reader is not bound to, carrying a top-level member
this reader does not know, satisfies two refusals at once. Both orders refuse -
there was never a bypass here - but only one of them answers the question a
compatibility caller is actually asking, which is whether its `ax` is too old
for this output. That question cannot be answered from a member name.

The identity is therefore settled first, in `cliresult.Decode` and in
`axerror.Decode` alike, and the closed member set is checked against a document
whose major is known. The pinned document scopes the member rule to the object
it governs three times over: Section 1.6 requires a reader to "reject an unknown
top-level field in a major version 1 object", Section 17.1 scopes the same rule
to "within any negotiated major version", and Section 17.2 lists "rejects an
unsupported major" as the reader's first rule. Section 15.1 adds the posture:
"receivers MUST NOT parse a different major's payload far enough to trust its
error code, retryable bit, details, or authority fields".

| Document | Refusal |
| --- | --- |
| Unbound major, unknown member | `ErrUnsupportedMajor` |
| Unbound major alone | `ErrUnsupportedMajor` |
| Bound major, unknown member | unknown top-level member |
| Bound major, missing member | missing required member |
| Foreign schema, unknown member | schema refusal |

`TestUnsupportedMajorIsSettledBeforeTheClosedMemberRule` pins each row in both
packages and
`TestReorderingTheIdentityCheckAdmitsNothingItUsedToRefuse` re-drives every
shape the old order refused, so the reorder changed which fact a caller reads
and admitted nothing. Two harness mutants restore the previous order and are
killed.

### What a machine client is allowed to depend on

`Read` is the consuming half of the same contract: it takes one completed
`ax --json` invocation and returns the machine-actionable classification, or a
refusal that names which fact is missing.

Its input type is `InvocationOutput`, which has exactly `Stdout` and
`ExitStatus`. There is no stderr member, so a reading structurally cannot depend
on a diagnostic stream; `TestMachineReadingCannotSeeStderr` asserts the member
set and fails the moment one appears. The behavioural half drives the real
emitter with logs and progress on stderr and classifies the invocation from
stdout alone, and its mutant swaps the two streams so the document lands on
stderr - the machine client is then left with `ErrAbsentDocument` rather than a
classification recovered from the status.

Neither remaining signal is trusted alone either. The document on stdout decides
the outcome and the exit status only corroborates it:

| Observation | Answer |
| --- | --- |
| No bytes on stdout | `ErrAbsentDocument` - never resolved from the exit status |
| Bytes that are not one readable JSON document | `ErrUnreadableDocument` - a failure to read is not an absence |
| One document of another schema | `ErrForeignDocument` |
| Success object at a failure status, failure object at exit 0, or `exit_code` unequal to the observed status | `ErrOutcomeDisagreement` |
| A status Section 15.2 assigns no meaning, `1` included | `ErrUnregisteredExitStatus` |
| A document whose members repeat | `ErrUnreadableDocument` - bytes with two readings are not one document |

The last row is the reason both readers run the same Section 1.6
common-data-model gate. `encoding/json` does not refuse a repeated member: it
resolves the repeat, and it resolves it differently per decode target, so a
Structured Error declaring `"retryable": false` and repeating the member as
`true` was read as retryable - a forged retry claim assembled from two members
no conforming writer could emit - a repeated `code` inside one exit class
answered with the second occurrence, and a repeated `details` resolved to the
union of both occurrences, which is neither. The exit-status corroboration does
not catch any of it, because both occurrences can share the exit class.

`cliresult.Decode` always canonicalized. `axerror.Decode` did not, so the
failure branch of the same reading admitted the shape, and `axerror.Decode` is
also the reader for peer-supplied provider, bridge, RPC, session-adapter and
terminal-backend envelopes, which made it reachable from a remote peer.
`requireCommonDataModel` closes it at `Decode`, so every one of those surfaces
is gated in one place; `documentSchema` runs the same gate before it reads the
`schema` member, because a repeated `schema` would otherwise select the branch.

The gate uses the canonicalizer as a check and **discards its output**. RFC 8785
serializes numbers through the ECMAScript algorithm, so the transform rewrites
`1e1` to `10`; `decodeExitStatus` reads `exit_code` from its raw bytes precisely
so that form is refused, and a gate that adopted its own canonical bytes would
be a widening wearing a validator's clothes.
`TestTheCanonicalGateDoesNotLaunderTheExitStatusToken` proves both halves - that
the transform really does rewrite the literal, and that `Decode` still refuses
it - and the harness carries the adopt-the-canonical-bytes mutant.

The exit status cannot stand in for the document, and the refusals say so with a
measured count rather than as advice: `exitStatusIsNotEnough` reports how many
registered codes the bound error version assigns to that status. Nothing on
`Reading` is computed from `message`; `HumanMessage` exists for display and is
the only message-derived value. `TestMessageTextChangesNoMachineAnswer` replays
each envelope with the message replaced - by a different code's name, by JSON,
by 4,096 characters - and requires every machine answer to be identical.

`testdata/historical` holds nine frozen invocations - the Section 14.2 and 15.1
normative examples verbatim, an envelope carrying a namespaced extension this
reader knows nothing about, unknown detail keys, a code a later compatible minor
added, that code with `retryable` forged to true, and a `session.clone.*`
failure. Each file's bytes are pinned by SHA-256 and each row records the
machine answers a client must still get, so regenerating a fixture to make a
test pass changes the compatibility claim instead of restating it. The
`session.clone.*` row is the compatibility fact the two CLI bindings produce:
this repository builds no clone success body, and its failure is still fully
classified, because "this build cannot construct that success" and "this failure
is unreadable" are different facts.

An 86-mutant harness over these gates kills 85 and declares 1 subsumed, each
mutant verified applied and compiled before it is measured, and each verified to
carry the MUTATED text on disk rather than only to differ from the original. The
subsumed one removes the exit-0 guard in `readFailure`; the `exit_code` equality
check refuses the same input one guard later, because a Structured Error can
never carry `exit_code` 0.

That count is a STATED BOUND, not a figure this repository measures. The harness
mutates the working tree, so it is task-scoped evidence under `.temp/` and
attached to its board item rather than run by `go test ./...`; no committed
artifact derives it, and a reader should treat it as reproducible-on-demand
rather than gated. Every figure in this file that CAN be derived from a
committed measurement is derived from one -
`TestREADMEFanInTableIsDerivedFromTheMeasuredProjection` re-derives the fan-in
table above, and `TestREADMEOwnershipFiguresAreDerivedFromTheMeasuredReport`
re-derives every figure of the ownership paragraph below.

Sixty-four of those mutants narrow rather than delete. The class is defined
once in the harness and counted from that definition rather than by hand; the
same definition applied to the 28-, 44-, 53- and 63-mutant harnesses selects
exactly the fourteen, twenty-seven, thirty-six and forty-four those revisions
published, so this figure extends that class instead of redefining it. The common-data-model gate
is restricted to one of the four registered versions, moved behind the identity
check, and made to adopt its own canonical bytes; the discriminator's copy is
disabled on its own; `ErrAbsentDocument` is aliased onto `ErrUnreadableDocument`;
and the two published fan-in figures are restored to the wrong values they were
first published with, plus one row deleted outright. A gate whose only evidence
is that deleting it reddens something has been shown to exist, not to hold.

The three `ErrOutcomeDisagreement` guards are therefore measured over their
whole domain rather than at one sample each, because each was first published
with a narrowing that the suite could not tell from the real thing.

The Section 14.2 equality is narrowed to `failure.CodeRegistered() &&
failure.ExitCode() != output.ExitStatus`. For a registered code the equality is
bound twice - `axerror.Decode` already cross-checks `exit_code` against the
pinned registry - but for a code a later compatible minor added, that lookup
takes the unregistered branch and runs no exit check at all, so `readFailure` is
the only thing binding the document to the status the process exited with.
`TestTheExitStatusEqualityBindsACodeTheRegistryDoesNotCarry` drives the frozen
later-minor envelope at all sixteen other registered failure statuses and first
asserts that its code is still unregistered, so the row cannot quietly stop
covering the class it exists for. A second mutant spares one arbitrary status
instead, which a single-sample row would have survived.

The success-at-a-failure-status guard is narrowed past exit 130, the Section
15.2 row "interrupted by operator signal before a clean response" and the one
status at which a success document plausibly does reach stdout before the
process dies. Both `ErrOutcomeDisagreement` rows now sweep every registered
failure status, drawn from `axerror.IsFailureExitStatus` and asserted to be
seventeen, and a fifth mutant narrows that enumeration below 130 to show the
sweep fails closed rather than shrinking in silence.

Two further claims that read as measurements are measured as such. The refusal
that explains why the exit status alone is not enough states a count, and
`TestTheRefusalStatesTheMeasuredFanInRatherThanAdvice` drives every registered
command at every registered failure status through `Read` and re-derives that
count from `CodesByExitStatus`; the harness replaces the count with a constant,
replaces the whole clause with the word "many", and narrows it to be correct at
exit 6 only - the last being exactly what a single-row assertion would have
survived. The retry bit is likewise read from the document at every failure
status on both polarities by
`TestTheRetryDecisionFollowsTheDocumentAtEveryFailureStatus`, because a branch
answering `true` from the status alone survived while the only exit-15 envelope
any test drove already declared `retryable: true`. The true arm covers fourteen
of the seventeen statuses - the pinned document forbids the claim for every code
of the other three - and that figure is asserted rather than left implicit.

The third guard is the command-tag agreement: `Read` refuses a document whose
`command` is not the command the invocation ran. It was published proven at one
ordered pair, invoked `doctor` against a document reporting `list`, and two
narrowings survived the whole repository suite - one sparing a different invoked
command, and one admitting any document *claiming* to be a `takeover` result,
which is the one body in this contract carrying adoption and authority
semantics. The agreement is now driven over the whole implemented vocabulary in
both directions: every implemented tag as the invocation against every other
implemented tag's own emitted document, 306 ordered pairs, each required to
refuse with this guard's own sentence naming both tags so a pair settled by the
version binding does not count as coverage of it, plus the 18 agreeing pairs
required to be admitted.

The share of the vocabulary that guard actually owns is measured rather than
described. `TestTheCommandAgreementGuardOwnsAMeasuredShareOfTheTagVocabulary`
drives all 1,936 ordered pairs over the 44 registered tags and classifies every
answer: 18 admitted, 306 refused by the command-tag agreement, 1,144 refused
before stdout is read because the invoked tag selects a version this repository
does not build, and 468 refused by the version binding because the document
claims a tag bound to another CLI Result major. Nothing is admitted that should
not be, and the guard is reached for exactly the 306. That is the stated bound:
for a pair involving one of the 26 unimplemented tags the guard is never the
enforcement, and the reason is an earlier named refusal rather than a bypass.

A fourth guard sits on the same reading path and was published with delete-only
evidence: the code-to-exit-status agreement in `axerror.decodeBody`, which
refuses a Structured Error whose `code` and `exit_code` name different Section
15.2 classes. Its whole coverage was one row - `observation_gap` driven at exit
5 instead of its own exit 9 - against 752 ordered (code, wrong-status) pairs at
1.0.0 alone, rising to 1,744 at 1.3.0. Deleting the guard reddened that row;
narrowing it to `code == "observation_gap"`, or sparing `policy_refused` or
`authentication_failed` individually, passed all thirteen packages.

That was a bypass and not only an unmeasured ratio. Nothing else refuses the
pairing, and the retryability refusal this package publishes keys on the exit
status for three whole classes - 7 authorization, 16 policy refusal, 130
operator interrupt - so relabelling a code out of its own class asks that
refusal about the wrong class. Under the narrowing, an `authentication_failed`
document rewritten to `exit_code` 9 with `retryable: true` was ADMITTED by
`Read` at exit status 9, and the caller received a `Reading` whose `Code()`
names an authorization failure and whose `Retryable()` is true. The Section 14.2
exit-status corroboration cannot see it, because the process really did exit
with the relabelled status.

`TestTheCodeToExitStatusAgreementIsMeasuredOverTheWholeCodeRegistry` drives every
registered code of every registered version at every registered failure status:
316 codes over 4 versions, 5,372 rows. The agreeing status is required to be
admitted and each of the other sixteen to be refused with this guard's own
sentence naming both numbers, so a pair settled by `decodeExitStatus`, by the
closed member set, or by the retryability gate does not count as coverage of it.
Every figure is re-derived from `CodesFor` and `CodesByExitStatus` and the
per-version totals are pinned at 752, 1,056, 1,504 and 1,744, so the loop cannot
go vacuous or quietly shrink; the harness narrows the code loop to one entry to
prove that.

The bypass itself is closed at the production entry point rather than at the
decoder. `TestRelabellingACodeOutOfItsExitClassCannotForgeARetryClaimThroughRead`
takes every 1.0.0 code whose own class carries an exit-keyed retryability refusal
- 7 of them - relabels each into all 14 failure statuses that carry none, and
requires `Read` to refuse all 98 with the agreement guard's sentence. The control
in the other direction is required too: the same code at its own exit status with
`retryable: true` must be refused by the retryability gate, which is what makes
the relabelling a bypass of something rather than a rewrite of nothing.

The registered-status admission in `decodeExitStatus` was sampled the same way,
at `{0, 1, 18, 99}`, so `status != 42` survived the whole suite.
`TestTheExitStatusAdmissionIsSweptOverEveryByteValue` sweeps 0..255 and requires
admission if and only if the Section 15.2 table registers the status as a
failure, plus six values outside the byte range. The document carries a code the
registry does not register, which takes the Section 15.3 unknown-code branch and
skips the agreement guard entirely, so the only decision left in the row is the
one being measured. The oracle is `ExitStatusMeaning` rather than
`IsFailureExitStatus`: an oracle built from the predicate under test would move
with the mutant, and the asserted row count of seventeen is what catches a
mutation that moves both. This path needs no process at all - `DecodeBound`
reads peer-supplied provider, bridge, RPC, session-adapter and terminal-backend
envelopes, where the document's own number is the only evidence there is.

Both common-data-model gates were also proven only on hand-built documents, so a
narrowing by payload SIZE - `len(data) < 4096` - refused every fixture and
admitted a large one, in `axerror.Decode` and in `documentSchema` alike. Byte
length is not a dimension of the Section 1.6 contract, but it is the one
predicate a peer chooses: `axerror.Decode` reads provider, bridge, RPC,
session-adapter and terminal-backend envelopes, and `stdout` is whatever the
invoked process wrote. Each gate now also drives a conforming document several
times that size - a Structured Error padded through its 64-key `details` bound,
and a CLI Result carrying twelve Session Summaries - and the harness carries
both size narrowings plus a narrowing of each fixture's own padding, so a
fixture that stopped being large reddens on its length assertion instead of
shrinking in silence.

### Refusal guard inventory

Six rounds of review closed one guard at a time, and the seventh found the twin
of the sixth one file away: `decodeExitStatus` was swept over 0..255 while the
reader-side gate in `Read` stayed at four sampled values. Working from memory
closes the guard you were told about. This table is the forced traversal that
replaces memory, and it is derived rather than written: `TestTheRefusalGuardInventoryIsDerivedFromTheReaderSource`
parses every refusal site out of `internal/cliresult/client.go` and requires a
bijection with the rows below, so a guard added without a row - or a row whose
site no longer exists - is a red before a reviewer has to find it.

Every row states what its domain evidence actually is. `measured` means the
guard was driven across its whole domain and the admitted and refused counts are
asserted against a production enumeration. `stated bound` means part of the
domain is not driven and names the contract reason, never effort.

| # | Guard | Sentinel | Refusal marker | Domain evidence | Measured by |
| --- | --- | --- | --- | --- | --- |
| 1 | `Read` refuses an exit status Section 15.2 assigns no meaning | `ErrUnregisteredExitStatus` | `%w: %d` | measured over 0..255 and eight values outside the byte range, admission required to agree with `ExitStatusMeaning` at every point | `TestTheReadLevelExitStatusAdmissionIsSweptOverTheWholeDomain`, `TestTheReadLevelExitStatusAdmissionAdmitsARealReadingAtEveryRegisteredStatus` |
| 2 | `Read` refuses a document belonging to some other contract | `ErrForeignDocument` | `schema %q` | measured over every contract identifier the pinned catalog registers plus twelve near-miss neighbours of the two admitted ones; stated bound - the member's domain is every JSON string and a discriminator fails on a neighbour of what it admits, not on an arbitrary string | `TestTheForeignSchemaRefusalIsMeasuredOverTheRegisteredContractVocabulary` |
| 3 | `documentSchema` reports stdout that carries no document | `ErrAbsentDocument` | `%w: %s` | measured at every registered exit status, and required at each to be distinct from the read-failure and foreign-contract facts | `TestReadDistinguishesAbsenceFromAReadFailure`, `TestTheReadLevelExitStatusAdmissionIsSweptOverTheWholeDomain` |
| 4 | `documentSchema` reports stdout that is not readable JSON | `ErrUnreadableDocument` | `decode: %v` | measured on a truncated envelope and on non-JSON bytes, each required to be distinct from an absence | `TestReadDistinguishesAbsenceFromAReadFailure` |
| 5 | `documentSchema` refuses a second document on stdout | `ErrUnreadableDocument` | `more than one document` | measured on two concatenated conforming envelopes, required to be answered by this guard and not by the common-data-model gate that also refuses those bytes | `TestReadDistinguishesAbsenceFromAReadFailure` |
| 6 | `documentSchema` refuses bytes with more than one reading | `ErrUnreadableDocument` | `outside the Section 1.6 common logical data model` | measured over every declared member of both schemas and over a conforming document several thousand bytes long; stated bound - the number half of Section 1.6 is not enforced on this path, disclosed below and pinned | `TestADuplicateSchemaMemberCannotSelectTheBranch`, `TestTheCommonDataModelGateDoesNotCoverTheSection16NumberRule` |
| 7 | `documentSchema` refuses a document with no `schema` member | `ErrUnreadableDocument` | `document has no schema member` | measured over its two-valued presence domain, both values driven; stated bound - a predicate keyed on the exit status survives, and the exit status is not a dimension of Section 14.2's exactly-one-document rule | `TestReadDistinguishesAbsenceFromAReadFailure` |
| 8 | `documentSchema` refuses a `schema` member that is not a string | `ErrUnreadableDocument` | `schema member is not a string` | measured over every JSON value form, JSON `null` included, with a string control required to reach the discriminator instead | `TestTheSchemaMemberTypeGuardIsMeasuredOverEveryJSONValueForm` |
| 9 | `readSuccess` refuses a success object at a failure status | `ErrOutcomeDisagreement` | `carries a CLI Result, which reports success` | measured over all seventeen registered failure statuses, the count asserted against `axerror.IsFailureExitStatus` | `TestReadRefusesEveryDisagreementBetweenStdoutAndTheExitStatus` |
| 10 | `readSuccess` refuses a document claiming a command the invocation did not run | `ErrOutcomeDisagreement` | `stdout reports command %q` | measured over all 1,936 ordered pairs of the 44 registered tags, classified 18 admitted / 306 by this guard / 1,144 before stdout is read / 468 by the version binding; stated bound - for a pair involving one of the 26 unimplemented tags an earlier named refusal settles it | `TestTheCommandAgreementGuardOwnsAMeasuredShareOfTheTagVocabulary` |
| 11 | `readFailure` refuses a failure object at exit 0 | `ErrOutcomeDisagreement` | `carries a Structured Error, which reports failure` | declared subsumed - `decodeExitStatus` never admits `exit_code` 0, so row 12 refuses the same input one guard later; the harness carries the mutant and records it as subsumed rather than killed | `TestReadRefusesEveryDisagreementBetweenStdoutAndTheExitStatus` |
| 12 | `readFailure` refuses an `exit_code` unequal to the observed status | `ErrOutcomeDisagreement` | `must equal that error's exit_code` | measured over all seventeen registered failure statuses for a code the pinned registry does not carry, which is the class where this guard is the sole enforcement | `TestTheExitStatusEqualityBindsACodeTheRegistryDoesNotCarry` |

Row 8 changed production code when it was measured. `encoding/json` admits JSON
`null` into a `string` and yields `""`, so a document whose `schema` member was
`null` had been answered as a foreign contract carrying the schema `""` - a
claim that some other contract owns the document, when what is true is that this
one is not readable. The guard now checks the raw token alongside the unmarshal.

The guards this leaf added or reordered outside that file are listed with it,
because the twin one file away is exactly what a single-file table would have
missed again:

| Guard | File | Function | Measured by |
| --- | --- | --- | --- |
| The registered-status admission inside the document - the twin of row 1 | `internal/axerror/decode.go` | `decodeExitStatus` | `TestTheExitStatusAdmissionIsSweptOverEveryByteValue` |
| The common-data-model gate added to the failure branch | `internal/axerror/decode.go` | `requireCommonDataModel` | `TestDecodeRefusesADuplicateMemberOnEveryDeclaredMember` |
| The code-to-exit-status agreement | `internal/axerror/decode.go` | `decodeBody` | `TestTheCodeToExitStatusAgreementIsMeasuredOverTheWholeCodeRegistry` |
| The envelope identity settled before the closed member set | `internal/axerror/decode.go` | `Decode` | `TestUnsupportedMajorIsSettledBeforeTheClosedMemberRule` |
| The same reordering on the success branch | `internal/cliresult/decode.go` | `Decode` | `TestUnsupportedMajorIsSettledBeforeTheClosedMemberRule` |
| The unsupported-major comparison reached through `Read` | `internal/cliresult/decode.go` | `verifyEnvelopeIdentity` | `TestTheUnsupportedMajorGuardIsSweptOverAMeasuredMajorRange` |
| The fan-in projection's unregistered-version refusal | `internal/axerror/registry.go` | `CodesByExitStatus` | `TestCodesByExitStatusRefusesAnUnregisteredVersion` |

STATED BOUND of the inventory itself: the derivation is exhaustive over
`internal/cliresult/client.go`, which this leaf added in full. The second table
is enumerated rather than derived, and the refusals `internal/axerror/decode.go`,
`internal/axerror/registry.go` and `internal/cliresult/decode.go` carry from
earlier leaves are those leaves' inventories, not this one's. Both tables are
pinned: every named function must be declared in its named file and every named
test must exist as a declaration, so a rename is a red rather than a row
pointing at nothing.

Two disclosures the traversal produced, stated here because neither is visible
from a passing suite.

The Section 1.6 common-data-model gate does NOT enforce the number half of that
section. `canonicaljson.Canonicalize` refuses duplicate members, non-UTF-8, and
non-canonical encoding, and the doc comment on `documentSchema` enumerates what
it owns; the opening sentence "inside the Section 1.6 common logical data model"
overstated it. `Read` admits a Structured Error whose `details` carry `1.5`,
which SPEC.md forbids as "integers only, inside the IEEE 754 double
safe-integer range". Section 1.6's number rule is bound to `internal/scalar` and
left unevidenced there, so nothing false is claimed in the traceability
projection; the admission is now asserted by
`TestTheCommonDataModelGateDoesNotCoverTheSection16NumberRule`, so closing the
gap reddens the pin and forces this paragraph to be updated with it.

A reason recorded in an earlier round is wrong although its conclusion holds.
Three mutants were dismissed on the grounds that "none is a registered code";
`terminal_backend_capability_unproven` IS registered at Structured Error 1.3.0
exit 6, and sparing it in `ExitCodeFor`'s version scoping survives on a clean
tree. The conclusion survives for a different reason: the same narrowing on
`observation_gap`, which is also only partially registered, is killed by
`TestExitCodeForRefusesVersionAndCodeDrift`, so the class has a covered
representative. The dismissal stands; the sentence behind it did not.


```bash
go test ./internal/cliresult -count=1 -v
go test ./internal/cliresult -cover -count=1
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
contract rows, 36 pinned or catalog-referenced normative section keys, 81
executable acceptance cases, 53 exact section bindings with their declared
coverage, 2 disclosed unowned sections, and 30 exact fixture identities or
Appendix D anchors. The v0.4.3 projection is checked as an owned 55-contract subset.
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
go run ./internal/traceability/cmd/tracecheck -section 6.2
```

The repeated `-section` form is the Story-scope production gate. It resolves
each assigned subsection, and every heading in a same-top-level range, against
the immutable v0.5.0 inventory. Every exact `section_binding` must name its own
production declaration and executable acceptance case, and must additionally
discharge the whole section it claims. A generic top-level source pin is not a
scoped implementation owner. Malformed, nonexistent, unpinned, or otherwise
unowned assignments fail closed.

### Declared coverage and what the gate can decide

A binding that names a real Go declaration and a real acceptance case used to be
enough. It is not: `section:2.2` named `validateSessionRecordCommon`, which
validates one enum and a name grammar, while Section 2.2 is the twenty-two
unconditional lease and replica invariants; `section:18.4` named
`OpenProjection` while Section 18.4 is audit retention and no retention code
exists. A Story assigned either section would have found it already owned and
could have done nothing.

Every `section_binding` now declares a `coverage` level, and the gate recomputes
that level rather than believing it. The denominator is measured from
[`internal/specdoc`](internal/specdoc): the RFC 2119 clause lines the pinned
`SPEC.md` carries under the section's own heading and its subheadings. The
numerator is the `clauses` the binding enumerates, each of which must name a
clause of that inventory, quote it verbatim beginning on the exact `SPEC.md`
line it occupies, and be discharged by an acceptance case the binding itself
owns.

| Level | Meaning | Admitted as an assigned scope |
| --- | --- | --- |
| `full` | every normative clause of the section is enumerated and discharged | yes |
| `partial` | at least half are | no |
| `sliver` | fewer than half are | no |
| `unevidenced` | none are; the registry makes no clause-level claim | no |
| `unmeasured` | the obligation scanner finds no clause line under the section at all | no |

`unmeasured` is the honest name for what used to be called `declarative` and
used to be admitted. The scanner matches uppercase RFC 2119 keywords, so a
section that states its obligations as a table scores zero and was read as
carrying no obligation. That is false, and the first revision of this gate
reproduced the very bug it was built to remove: `-section 15.2`, the eighteen-row
normative exit-code registry, exited 0 with nothing in the tree implementing it,
and so did `-section 7.3`, the closed Provider Manifest. Nineteen of the 157
pinned headings are in that class - 7.3, 10.8.1, 13.5, 13.12, 13.14.1-13.14.5,
14.3, 14.6, 15.2, 16.6, 16.7, 18.2, 19.4 and Appendices A, B and C - and
`TestUnmeasuredCoverageIsAScannerBlindSpotNotAnAbsenceOfObligation` measures
that every one of them has a substantive body, so not one is a heading with
nothing to discharge. Keyword absence is now a failure to measure, never a
coverage claim.

A binding below `full` must also name its gap, `unmeasured` included, and the
gap has to name the section as a whole identifier - so a sentence about 6.55 is
not a gap about 6.5 - and name the production declaration the binding is
registered to. `unowned_sections` records a section this repository does not
implement at all, with a gap and the evidence for it; an unowned entry may not
cover a section the generated catalog requires an owner for, so it is a
disclosure and never an exemption. Every one of these fields is inside the
reviewed projection digest, so none of them can be self-minted.

This is what the gate decides, and it is less than semantic coverage. It decides
that a claimed clause is a real obligation of the claimed section, that it is
quoted verbatim at the line it occupies, that the discharging acceptance case is
registered and owned by the binding, and that the declared level equals the
measured ratio. **It cannot decide that the named acceptance case exercises the
clause's meaning.** A binding could enumerate every clause of a section and
point all of them at one weak test, and the gate would admit it. That residual
class - a complete enumeration discharged by inadequate tests - is not covered
here and is not claimed to be.

It also cannot see an obligation stated without an RFC 2119 keyword, which is
why `unmeasured` is refused rather than admitted: the gate reports that it could
not measure the section instead of inferring that there was nothing to measure.
And the gap-quality check is a tightening rather than a proof - a sentence that
names both its section and its production declaration and still says nothing
useful is admitted, and the gate cannot decide otherwise.

### Measured coverage of this repository

`tracecheck` prints the ratio it measured rather than a sentence about it:

```text
section coverage: bindings=53 full=1 partial=3 sliver=1 unevidenced=45 unmeasured=3 unowned=2 clauses_discharged=17/428
```

Fifty-three section bindings discharge 17 of the 428 normative clauses their
sections carry. One binding is `full` (Section 6.2, whose single clause is the
native-Windows `conpty` requirement, discharged by the positive
`TestEveryPinnedReaderHasPositiveNativeWindowsAndWSL2Lanes` lanes together
with the negative `TestDecodeRefusesNonConptyBackendOnNativeWindows` legacy
refusal arm), three are
`partial` (Section 14.2 at 8/9, bound to
[`internal/cliresult`](internal/cliresult), whose undischarged clause `14.2#6`
is the process exit status this repository has no binary to produce; and
Section 15.1 at 5/7 and Section 15.3 at 2/3, both bound to
[`internal/axerror`](internal/axerror); the three undischarged clauses there are
the RPC hello obligation `15.1#5`, the bootstrap-row sentence `15.1#6` that
binds the provider plugin rather than the host, and the hello-key and
TerminalBackend-capability prohibition `15.3#3` - this repository builds no RPC
hello frame and no provider plugin, and advertises no Structured Error as a
TerminalBackend capability. The closed 16-capability admission registry in
[`internal/terminalbackend`](internal/terminalbackend) gates which operations
an admitted probe may confer; it is never advertised on an RPC hello path and
carries no Structured Error code, so the prohibition holds vacuously and the
clause stays undischarged. The same package carries the Section 4 lifecycle,
attach, entrypoint, replication, and historical-translation conformance
harness (`conformance.go`, exercised by `conformance_test.go`) and an
AST-derived refusal-arm inventory (`refusal_arm_inventory_test.go`) that
requires every production refusal arm to be declared with a resolving named
asserting test in both directions; behavioral proof stays with each row's
named test, which the inventory resolves textually),
one is
`sliver` (Section 10.3, whose chunk offset invariant is
enforced by `validateBlobDescriptor` while its two receiver clauses have no
implementation), three are `unmeasured` (Sections 7.3, 13.14.5 and 15.2, each of
which carries a gap saying why the scanner measures zero and what is missing),
and forty-one are `unevidenced`. Two sections are recorded unowned.
Assigned-scope admission therefore succeeds today for `-section 6.2` and
nothing else; every other assignment is refused with its ratio and its gap.
A `partial` binding is refused by assigned-scope admission exactly like an
`unevidenced` one: admission requires `full`.

One admitted binding out of forty-nine covers a single clause, and it is
disclosed here rather than hidden: without Section 6.2 the admit path would only
ever be exercised synthetically. Its discharge is no longer positive-only: the
native-Windows lanes carry the positive arm and
`TestDecodeRefusesNonConptyBackendOnNativeWindows` the legacy refusal arm,
both registered against the `config-versioned-readers` acceptance case.

That is a disclosure of the shipped state, not a target that was met.
`TestRunRefusesEveryAssignedSectionThatOnlySlivers` and
`TestVerifyAssignedSectionsRefusesEveryBindingThatOnlySlivers` pin every failing
binding with its exact measured ratio, so a section that becomes covered has to
leave those tables deliberately. Five further gaps the disclosure surfaced:
Section 15.2's code-to-exit registry is implemented in
[`internal/axerror`](internal/axerror) and checked row by row, and
[`internal/cliresult`](internal/cliresult) now maps a failure to that exact
process status through `Emit` and `ExitStatus`, but nothing in the tree calls
`os.Exit` with it - the only `os.Exit` calls are the `exit(1)` failure paths of
`cataloggen` and `tracecheck` - so clause `14.2#6` stays undischarged until an
`ax` binary exists; Section 7.3's closed Provider Manifest exists only as a catalog
row naming its URN; Section 6.5 requires the `required_capabilities` default to
be the platform lane minimum while `internal/config/validation.go` accepts only
an empty default; Section 17.2's single clause is an unknown-event reader
obligation while its binding now names the CLI Result machine reader `Read`,
which classifies one completed invocation rather than retaining any event, so it
does not discharge that clause either - an unknown error code and an inert
unknown extension are not an unknown session event, and the four numbered reader
rules that binding does implement carry no RFC 2119 keyword and are invisible to
the clause scanner; and Section 2.1's single clause is a replica runtime
obligation that no code here implements.

Successful output reports ownership inventory counts and the measured coverage
ratio only. The gate does not mutate repository or product state, add an `ax`
command or `doctor` result, or claim that any catalog capability is available,
enabled, or supported.

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
| Go toolchain | Verify global and assigned-scope specification ownership, validate versioned Configuration readers/current writer, validate owner-local storage, immutable installs, and SQLite rebuild/recovery, validate and fuzz common wire scalars, canonical identities, core records, Session Events, and Observation Events, validate the Structured Error registry, its static containing-contract bindings, and its detail redaction, validate the CLI Result envelopes, command bodies, common flags, rendering boundary, and exit-status mapping, classify one completed `ax --json` invocation from stdout and its exit status through the machine reader and replay the frozen historical envelope corpora, generate and check the typed catalogs, build, test, and measure the Go implementation | `go run ./internal/traceability/cmd/tracecheck`; `go run ./internal/traceability/cmd/tracecheck -section 6.2` (every other assigned section is refused with its measured coverage ratio); `go test ./internal/config -cover -count=1`; `go test ./internal/localstore -cover -count=1`; `go test ./internal/scalar -cover -count=1`; `go test ./internal/scalar -run=^$ -fuzz=^FuzzScalarProductionEntries$ -fuzztime=100x -parallel=1`; `go test ./internal/canonicaljson -cover -count=1`; `go test ./internal/axerror -cover -count=1`; `go test ./internal/cliresult -cover -count=1`; `go test ./internal/canonicaljson -run=^$ -fuzz=^FuzzCanonicalizeRoundTrip$ -fuzztime=100x -parallel=1`; `go test ./internal/canonicaljson -run=^$ -fuzz=^FuzzObjectIdentityRepresentationInvariant$ -fuzztime=100x -parallel=1`; `go test ./internal/canonicaljson -run=^$ -fuzz=^FuzzClosedIdentityShapeRefusal$ -fuzztime=100x -parallel=1`; `go test ./internal/canonicaljson -run=^$ -fuzz=^FuzzObservationEventRefusal$ -fuzztime=100x -parallel=1`; `go generate ./internal/catalog`; `go run ./internal/catalog/cmd/cataloggen -metadata internal/catalog/catalog.v0.5.0.json -contracts internal/specpin/v0.5.0.lock.json -output internal/catalog/catalog_gen.go -check`; `go test ./... -v`; `go test ./... -cover`; `go build ./...` | Read-only traceability report; owner-only roots, immutable blob/quarantine data, and `<state>/index.sqlite` plus recovery evidence only when storage entries are called; `internal/catalog/catalog_gen.go`; Go build/fuzz cache; test output captured under `.temp/<TASK-ID>/` when needed |
| `github.com/gowebpki/jcs` | RFC 8785 byte transformation after repository-owned strict I-JSON validation | Imported by `internal/canonicaljson.Canonicalize` at pinned module version `v1.0.1` | Canonical UTF-8 JSON bytes in memory; no durable output |
| `github.com/pelletier/go-toml/v2` | Parse and emit TOML while the repository-owned Configuration layer enforces exact versioned closed schemas | Imported by `internal/config.Decode`, `internal/config.EncodeCurrent`, and explicit `internal/config.Migrate` at pinned module version `v2.4.3` | Validated Configuration values/TOML bytes in memory; explicit migration writes a same-directory replacement plus an owner-only versioned backup |
| `modernc.org/sqlite` | Provide the pure-Go SQLite driver for the local derived index without a CGO platform dependency | Imported by `internal/localstore.OpenProjection` at pinned module version `v1.57.0` | `<state>/index.sqlite`, its owner-only lock and WAL/SHM/journal sidecars, and `<state>/index-recovery/<uuid>/` corruption evidence |
| `golang.org/x/sys` | Invoke OS-native no-replace rename primitives so concurrent processes cannot overwrite an immutable digest path | Imported by `internal/localstore.atomicRenameNoReplace` at pinned module version `v0.47.0` | Same-filesystem atomic rename only; no standalone artifact |
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
