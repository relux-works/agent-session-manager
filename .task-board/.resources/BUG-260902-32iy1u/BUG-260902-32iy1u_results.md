# BUG-260902-32iy1u — stop-inferring-semver-on-environment-tuple

Branch `task-board/story/STORY-260902-1khg33`, worktree
`.temp/STORY-260902-1khg33/worktree`.

## The decision

One compiled `semverPattern` in `internal/canonicaljson` was reached from more
than one schema, so two separate questions had to be answered. Both are now
recorded explicitly in
`internal/canonicaljson/testdata/constraint-enumeration.md` under
"Recorded decision: where `semver` applies, and what it means", and mirrored in
`README.md`.

**1. Does the constraint apply at this member?** It applies exactly where the
pinned document declares it, and nowhere else.

**2. Where it does apply, what does `semver` mean?** SPEC v0.5.0 types members
`semver` and names Semantic Version but spells out no grammar. The named
standard is Semantic Versioning 2.0.0, whose valid versions include optional
prerelease and build metadata. `semver` is adopted as that standard **in full**.

| Site | Pinned declaration | Decision |
| --- | --- | --- |
| `EnvironmentTuple.adapter_version` | "Environment Tuple contains exactly `environment_id`, `environment_version`, `platform=linux\|macos\|windows\|wsl2`, `architecture=amd64\|arm64`, `store_schema_fingerprint`, and `adapter_version`" — no type, no format | **Constraint removed.** Presence-only, exactly like its untyped `environment_version` and `store_schema_fingerprint` siblings in the same clause. |
| `works.relux.ax.migrated-from.schema_version` (§17.3) | "That extension value is a closed object containing exactly `schema_id:string`, `schema_version:semver`, and `object_id:digest`." | **Constraint kept and corrected** to SemVer 2.0.0 in full. This is the authorising line, quoted verbatim. |
| Session Event `terminal.*` `implementation_version` / `protocol_version` | "`implementation_version:semver`", "`protocol_version:semver`" | Same as above, through the shared widened grammar. |

The SemVer word that used to be read onto the tuple member comes from the
Session Adapter Manifest row "`display_name` / `adapter_version` | UTF-8
string[1..128] / SemVer" (SPEC.md:3610), which closes a different object, and
from the Probe sentence "Provider ID, manifest digest, and adapter version
equal the verified Manifest and host values" (SPEC.md:3625-3626), which names
the Probe's own top-level members and not the nested tuple member. Neither
reaches the tuple.

Pinned spec verified locally before quoting: `SPEC.md` SHA-256
`562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a` at commit
`28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`, matching `.spec/README.md`.

## Changes

Production, `internal/canonicaljson/closed_shapes.go`:

- `validateEnvironmentTuple` no longer reads or matches `adapter_version`. Its
  presence is still required by the existing `requireExactMembers` call. The
  in-function comment now covers all three untyped members and says why.
- `semverPattern` widened from `^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`
  to SemVer 2.0.0 in full, matching the spelling already used in
  `internal/config/validation.go:21`. This corrects the migration-provenance
  site at `validateMigrationExtensionObject` and the `terminal.*` versions at
  `validateTerminalV4Payload` in `core_records.go`.

Tests and pins:

- New `TestEnvironmentTupleAdapterVersionCarriesNoInferredSemVerConstraint`:
  `1.2.3-rc.1` is accepted at both public identity entries, as are build
  metadata, `01.2.3`, `1.2`, `v1.2.3`, free text, empty string, number, boolean
  and null. Absence is still refused, attributed by name, with the member
  deleted from **both** tuples of the record so a sibling's extra member cannot
  keep the case green.
- New `TestMigrationProvenanceSchemaVersionIsSemVer200InFull`: seven accepted
  forms including `1.2.3-rc.1` and `1.2.3-rc.1+exp.sha.5114f85`, and twelve
  still refused including `01.2.3`, `1.2`, `1.2.3-`, `1.2.3-01`, `1.2.3-a..b`,
  `1.2.3+` and a fullwidth digit. Widening the gate is not the same as deleting
  it, and only these cases tell the two apart.
- `grammar_inventory_test.go`: the pinned reference for `semverPattern` is
  updated and its `implementationDefined` note now states the full-standard
  reading. The widened grammar declares 22 dimensions (2 anchors, 18 character
  classes, 2 one-or-more quantifiers); all 22 carry a witness that production
  refuses and that exactly its own one-dimension widening admits.
- `member_shape_refusal_test.go`: `adapter_version` added to
  `untypedFixtureMembers` and `unenforcedStructuredMembers`. Both sets are
  asserted exactly in both directions, so if production ever refuses a wrong
  JSON type or a malformed form there again, the exemption fails as obsolete.
- `session_record_versions_test.go`: the obsolete `canonical-semver` Session
  Record grammar family and the `adapter_version` typed-provenance case are
  removed; no Session Record member row declares `semver` any more.
- `constraint-enumeration.md`: the `EnvironmentTuple.adapter_version` row is now
  Presence-only; the §17.3 row records SemVer 2.0.0 in full; the decision
  section is appended.
- `README.md`: the "Probe-to-Manifest SemVer link for `adapter_version`" claim
  is replaced by the recorded decision.

## Mutation evidence

Each mutant was applied to production, the named case run, then reverted.

| Mutant | Result |
| --- | --- |
| A. Re-add a SemVer gate on `EnvironmentTuple.adapter_version` (widened pattern) | RED — 8 cases fail |
| A2. Re-add the **original** core-triple gate, i.e. the exact reported bug | RED — `.../prerelease` fails: "EnvironmentTuple adapter_version must be canonical semver" |
| B. Narrow `semverPattern` back to the core triple | RED — 5 `accepts_*` cases of the migration test fail |
| C. Delete the migration-provenance semver gate entirely | RED — 8 `refuses_*` cases fail |
| D. Drop `adapter_version` from `requireExactMembers` | RED — `.../absent` fails |

Mutant D was initially GREEN because only the source tuple was mutated and the
target tuple's surviving member produced an unrelated refusal; the test was
strengthened to delete from both tuples and to attribute the refusal by name,
after which D reddens.

## Validation

All commands run from the Story worktree as standalone processes; real exit
codes below.

| Gate | Exit | Result |
| --- | ---: | --- |
| `go build ./...` | 0 | pass |
| `GOOS=linux GOARCH=amd64 go build ./...` | 0 | pass |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 | pass |
| `go vet ./...` | 0 | pass |
| `gofmt -l internal/` | 0 | no output |
| `go test ./... -count=1` | 0 | 10/10 packages ok |
| `go test ./... -cover -count=1` | 0 | see below |
| `go test ./... -race -count=1` | 0 | 10/10 packages ok |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | contracts=60 sections=36 cases=43 fixtures=30 |
| `cataloggen ... -check` | 0 | pass |
| `FuzzCanonicalizeRoundTrip` 100x | 0 | pass |
| `FuzzClosedIdentityShapeRefusal` 100x | 0 | pass |
| `FuzzObjectIdentityRepresentationInvariant` 100x | 0 | pass |
| `FuzzObservationEventRefusal` 100x | 0 | pass |
| `FuzzScalarProductionEntries` 100x | 0 | pass |

Coverage, `internal/canonicaljson`, baseline taken from a `git archive HEAD`
extraction of the same worktree:

| Package | Before | After | Delta |
| --- | ---: | ---: | ---: |
| `internal/canonicaljson` | 97.2% | 97.2% | 0.0pp |

Other packages after the change: catalog 97.6%, cataloggen cmd 79.3%,
cataloggen 83.9%, config 94.7%, localstore 86.5%, scalar 90.1%, specpin 85.1%,
traceability 85.0%, tracecheck 87.5%. No package regressed; only
`internal/canonicaljson` was touched by this change.

Logs: `.temp/BUG-260902-32iy1u/gotest-01.log`, `gocover-01.log`,
`gorace-01.log`, `tracecheck-01.log`, `catalog-check-01.log`,
`fuzz-*.log`.

## Not claimed

- No commit, push, PR, or landing was performed; integration into trunk is the
  orchestrator's step.
- The Session Adapter Manifest and Session Adapter Probe schemas are not
  validated by this package, so the Manifest's own declared-SemVer
  `adapter_version` and the Probe's declared equality to it are untouched and
  remain outside this identity gate.
