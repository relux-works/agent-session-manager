# TASK-260830-17suox review verdict — ACCEPTED

- Change Request: `CR-TASK-260830-17suox-5` revision 5
- Base OID: `020d0b6c68c587b6463add58330050ceff71b87f`
- Candidate tree OID: `d6f1511a78991c5f3a0f9b8d83bfe610eb52f698`
- Reviewer run: `RUN-260901-1ffcc5`
- Reviewed at: 2026-09-01

## Candidate integrity

Every untracked new file on disk was hashed and compared against the candidate
tree blob. All seven match exactly, so the review below was performed on the
exact revision under review, not on a drifted working tree:

    internal/config/schema.go, validation.go, writer.go,
    schema_test.go, refusal_test.go, refusal_inventory_test.go,
    compatibility_regression_test.go   -> git hash-object == candidate blob

Tracked deltas were read from
`git diff 020d0b6c d6f1511a` (16 files, +3580/-40).

## Specification fidelity

The pinned normative source was checked out at the exact pinned commit and
document hash before comparison:

    relux-works/agent-session-manager-spec @ 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c
    SPEC.md sha256 562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a
    == internal/specpin/v0.5.0.lock.json source.document.sha256

§6.1–§6.5 and §17.1–§17.2 were read line-by-line against the implementation.
Checked and correct: closed root/table registries per version; the exact v1
field/default table; `sync.chunk_bytes` fixed constant; `mesh.payload_encryption`
= none only; `mesh.transport` = ssh only; unique peer host_id/name; peer
`ssh_args` [0..64] with 1–4096-byte members and 65,536-byte total argv including
the endpoint; `logical_root` grammar and uniqueness at root and peer scope;
runtime-probe platform match; the complete v2 `directory` table with its
relational freshness ordering; the exact `directory_installations`,
`directory_enrichment_profiles`, and `directory_peer_disclosure` member sets;
the v3 `terminal` replacement with `backend_id`, sorted-unique closed
capabilities, `multiple_input_policy`, `transport_policy` subset,
`external_trust`, and `backend_config`; §4.E legacy `tmux`->`ax.tmux` /
`conpty`->`ax.conpty` mapping with no inferred capabilities; §17.2 writer emits
exactly the current version and the reader rejects unsupported majors and minors.

## Gates attacked, not read

16 source mutants were applied in an isolated copy (`/tmp/ax-mutants`) and the
package suite was run against each. **16/16 killed** — including narrowing
mutants, not only deletions:

| # | Mutant | Result |
| --- | --- | --- |
| 1 | delete `providers.require_explicit_trust` gate | killed |
| 2 | widen `mesh.sync_interval_seconds` max 86,400 -> 86,401 | killed |
| 3 | narrow SSH bypass detection: drop `off` alias | killed |
| 4 | narrow forbidden extension names: drop `endpoint` | killed |
| 5 | delete `terminal.backend_id is not registered` gate | killed |
| 6 | accept `backend_config` when no settings schema is registered | killed |
| 7 | widen extension nesting depth 4 -> 5 | killed |
| 8 | allow duplicates in sorted-unique lists (`>=` -> `>`) | killed |
| 9 | delete platform-matches-runtime-probe gate | killed |
| 10 | remove `DisallowUnknownFields` (open the TOML shape) | killed |
| 11 | remove the present-empty-map repair | killed |
| 12 | accept any `schema_version` by falling through to v3 | killed |
| 13 | delete the schema identifier gate | killed |
| 14 | open the legacy `terminal.backend` vocabulary | killed |
| 15 | delete the `required_capabilities` default-provenance gate | killed |
| 16 | add a new, never-exercised `configError` call site | killed |

Mutant 16 is the important one: it proves the derived refusal inventory in
`refusal_inventory_test.go` is a real gate. That test parses every non-test
`.go` file in the package, derives the set of `configError(` call sites, and
fails the package when any site has no exercised negative path. Completeness is
derived from source, not enumerated by hand, so a new refusal cannot ship
without a negative test. It correctly restricts itself to full-package runs and
exempts exactly one site with an inline `config-refusal-subsumed:` justification.

The other derived-completeness properties were verified as real, not decorative:
the supported reader set is derived from the pinned Configuration catalog
contract versions; the terminal capability vocabulary is derived from the pinned
catalog `terminal_backend` family and asserted member-by-member against the
production set; the closed-map member inventory is derived by reflection over
`rawV3` and compared with `reflect.DeepEqual` against the declared case table,
so a new closed map member fails the suite until it has both an empty-object
round trip and an absence refusal.

## Adversarial acceptance probes

19 hand-crafted documents were driven through the production `Load` entry.
Correctly refused: empty document; v1 carrying `[directory]` or
`directory_installations`; v2 carrying `terminal.backend_id`; v3 carrying legacy
`terminal.backend`; v1 carrying `terminal.required_capabilities`; a TOML
datetime as an extension value; a UUIDv4 `host_id`; a duplicated `[mesh]` table;
a peer workspace root missing `path`.

Accepted-and-correct: `StrictHostKeyChecking=accept-new` (not a bypass — a
changed key is still refused); an explicit empty `transport_policy` (the valid
deny-all subset, distinct from omission and separately pinned); an external
backend with no config-layer platform gate (platform support is Manifest/Probe
evidence, not a configuration fact).

Bound directions were confirmed in both directions at the production entry, and
character bounds count characters: `host_name` accepts 64 CJK characters and
refuses 65, `endpoint` accepts 1024 characters and refuses 1025, SSH arguments
accept 4096 bytes and refuse 4097, the 65,536-byte extension object is pinned
against a spec-derived constant rather than the implementation constant.

## Validation run by the reviewer

    go build ./...                    ok
    go vet ./...                      ok
    go test ./...                     all 9 packages ok
    go test ./internal/config -cover  93.2% of statements
    go run ./internal/traceability/cmd/tracecheck \
      -section 3.2 -section 6.1 -section 6.2 -section 6.3 -section 6.4 \
      -section 6.5 -section 17.1 -section 17.2
      -> traceability ok: contracts=60 normative_sections=36
         acceptance_cases=32 fixtures=30 compatibility_contracts=55
         assigned_scopes=8

The seven new ownership section bindings were read individually. Each names a
real production declaration (`Load`, `Decode`, `validateConfiguration`,
`translateV2`, `translateV3`, `EncodeCurrent`) and real test declarations that
exist and execute. The reviewed ownership canonical SHA-256 was advanced with
those bindings, which is exactly the review this constant is designed to force.

## Unsupported-claim check

`Decode` is reached from the production `Load` path, not only from tests.
`EncodeCurrent` is the package's exported writer entry point; the repository has
no `ax` CLI yet, so this is not a guard that production never invokes. README
scope statements are accurate: no migration, backup, atomic replacement, doctor,
CLI, or backend-availability claim is made, and no such behavior is implemented.
The AC clause about crash/idempotency evidence does not apply — `EncodeCurrent`
returns bytes and writes no file, and that non-mutation is stated rather than
papered over.

## Non-blocking notes for later scope

1. **`terminal.backend_config[].settings` receives no config-layer structural
   screen.** Unlike every `extensions` map, `settings` gets no depth, byte-size,
   value-type, or forbidden-name check — the shape is fully delegated to the
   caller's registered validator. This is spec-faithful (§6.5 assigns the closed
   settings shape to the exact backend implementation version) and it fails
   closed with no validator (mutant 6). Worth revisiting as defence in depth
   when the terminal backend settings registry story lands.
2. **`required_capabilities` default.** §6.5 states the default is the "platform
   lane minimum", a phrase that appears exactly once in the specification and is
   never defined by any normative table. The implementation leaves the default
   empty with explicit provenance instead of inventing a set. That is the
   correct reading: §4.E requires "no inferred capabilities" and §17.2 makes an
   enabled capability valid only against exact evidence binding. The choice is
   stated plainly in the README rather than silently taken.
3. **`hasForbiddenConfigName` is a label-split heuristic**, so an extension key
   such as `works.relux.mytoken` passes while `works.relux.token` is refused.
   The specification forbids secret-bearing fields, not substrings, so this is
   not a defect; recorded only so the boundary is on the record.

## Verdict

Accepted. Production entry points implement the scoped deliverable, every
refusal clause is individually pinned and survives targeted narrowing, the
completeness properties are derived from pinned sources rather than hand-listed,
the traceability binding is real, and no unsupported capability is advertised.

## Round-4 directive: CLASS 3 verification

The round-4 verdict left one class — "tests that pass for the wrong reason" —
with four concretely named requirements. Each was verified by running the exact
mutant the directive describes, not by reading the diff.

| Round-4 requirement | Mutant applied | Result |
| --- | --- | --- |
| Four negative Load cases omitting exactly one closed map member each | `restorePresentEmptyMaps` narrowed to initialize nil map members before the source-presence check (the rev4 survivor) | **killed** — `TestLoadRefusesEveryAbsentRequiredClosedMapMemberAtProductionEntry` fails on all four members |
| Cases must isolate their clause | drop only the `Extensions == nil` disjunct from the `directory_installations` presence check | **killed** — fails on exactly `.../directory_installations[].extensions` |
| Same, for the fourth presence check | drop only the `Settings == nil` disjunct from `terminal.backend_config` | **killed** — fails on exactly `.../terminal.backend_config[].settings` |
| Byte bound asserted against the SPEC literal, not the implementation constant | `maxConfigExtensionBytes` 65,536 -> 655,360 | **killed** — `.../extension_entry_depth_and_canonical_byte_bounds` |
| Same bound, narrowing direction | `maxConfigExtensionBytes` 65,536 -> 65,535 | **killed** — same subtest |

The first mutant is the one that mattered: it is precisely the rev4 blocker,
and it now reddens. The disjunct mutants each fail on the single subtest that
claims to pin them, which is the isolation property the directive demanded —
the case cannot be refusing on an earlier disjunct, because there is only one
violation in the document. `TestLoadRefusesEveryAbsentRequiredClosedMapMemberAtProductionEntry`
first asserts the complete fixture loads, then omits exactly one member, so a
false pass from an unrelated defect is excluded.

The byte bound is now declared in the test as
`specificationExtensionObjectMaxBytes = 65_536` with the SPEC §6 citation, plus
a self-check that the fixture's marshalled JSON is exactly that many bytes. It
no longer references `maxConfigExtensionBytes`, so the assertion has an
independent reference and fails in both directions.

README line 171 claims the absence/explicit-empty-object distinction is
preserved in both inline and multiline spelling. Both sides are now evidenced:
`TestEveryClosedMapMemberRoundTripsAnExplicitEmptyObjectAtProductionEntry` and
`TestLoadTreatsInlineAndMultilineEmptyClosedMapsAsPresent` for presence,
`TestLoadRefusesEveryAbsentRequiredClosedMapMemberAtProductionEntry` for
absence. The claim is no longer one-sided.

An invalid mutant is recorded for honesty: the first attempt at the
`restorePresentEmptyMaps` narrowing left an unused variable and failed to
compile. A build failure is not a kill, so it was rewritten to compile cleanly
and re-run; the result above is from the compiling version. Every mutant in
both sweeps was build-checked before its verdict was recorded.

**Fourth non-blocking note.** `TestLoadRefusesEveryAbsentRequiredClosedMapMemberAtProductionEntry`
is not listed among the `config-versioned-readers` test declarations in
`ownership.v0.5.0.json`, even though it is the test that closes the rev4
blocker. Traceability passes and the test executes in every full-package run,
so no gate is weakened; the registry simply under-cites its own strongest
evidence. Worth adding when that file is next touched.
