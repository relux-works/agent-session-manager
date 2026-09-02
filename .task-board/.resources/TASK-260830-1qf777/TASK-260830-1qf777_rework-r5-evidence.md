# TASK-260830-1qf777 — Round 5: closing CR rev3 F1, F2, F3

Producer round answering the `to-dev` routing of review verdict rev3
(`RUN-260901-8970a0`). That verdict found **no production defect** — every
probe passed on the shipped behavior. What it found was missing negative and
preservation evidence: three mutants that change durable operator state while
`go test ./internal/config` stays green.

All three are now dead, and so are four of the five survivors the same verdict
listed as non-blocking. Full sweep: **73 mutants — 71 RED, 1 declared-subsumed
survivor with its subsuming check named, 1 positive control GREEN, 0 unexplained
survivors, 0 invalid anchors**, restored tree green.

The one survivor is R13 and it is reported as a survivor, not folded into the
RED count. `encodeVersion2`'s directory-collection passthrough cannot be killed
through the production entry, so the harness declares it in `EXPECT_SUBSUMED`
alongside the check that subsumes it, and reports
`SUBSUMPTION CLAIM NO LONGER NEEDED` if it ever starts dying.

## F2 (the serious one) — migration never carried a non-default member

The verdict was explicit that three cases for the three cited members would not
be an answer: *"a fixture that only carries defaults cannot observe a dropped
member, because the default is what a dropped member decays to."* The property
built instead:

1. **The member set is derived, not listed.** `wireLeafTOMLPaths` walks
   `reflect.TypeOf(rawV1{})` and returns every leaf TOML path, recursing into
   arrays of tables under a `[]` segment. A member added to `rawV1` later joins
   the set without editing the test.
2. **A saturated fixture.** `saturatedVersion1Document()` carries every derived
   path, each at a value distinguishable from the one it would decay to.
   `TestSaturatedVersion1FixtureCoversEveryConfiguration1Member` proves that
   mechanically: every derived path must be present in the fixture, every
   scalar path's value must differ from the same path in a defaults-only
   document (produced by migrating a minimal v1 through the real `Migrate`),
   and every collection path must be absent from that defaults document so a
   populated entry is itself the distinguishing value.
3. **Exemptions are falsifiable, not asserted.** Ten Configuration 1.0.0
   members cannot be set away from their default and still load — the
   validators admit exactly one value (`mesh.transport`, `mesh.payload_encryption`,
   `sync.chunk_bytes`, `providers.require_explicit_trust`, `terminal.backend`
   under its platform coupling, `platform` under the runtime probe) or the
   member is required and has no default (`host_id`, `host_name`, `schema`,
   `schema_version`). Each exemption declares an alternate value, and
   `assertVersion1ExemptionIsReal` drives that alternate through the real
   `Load` entry and requires a refusal. Mutants N18–N23 widen each of those
   validators in turn; every one reddens the exemption self-check.
4. **Whole-struct comparison.** `TestMigrationRetainsEveryConfiguration1MemberOnEveryTarget`
   loads the saturated source, migrates it through the real `Migrate` entry, and
   compares the whole loaded `Configuration` before and after on three paths:
   `v1 → 2.0.0`, then `2.0.0 → 3.0.0` continuing from that migrated document,
   and a fresh direct `v1 → 3.0.0`. The only declared difference is
   `Directory.GeneratedSummaryUpgradeChoice`, which the migration records by
   contract. `configurationDifferences` reports the exact member path that moved
   rather than dumping a struct, so a dropped member is named.

The three F2 survivors and twelve further drop-a-member mutants across
`encodeVersion2` and `currentWire` (mesh sync interval, service health interval,
path-plugin policy, auto-resume, yolo confirmation, workspace roots, mesh peers,
peer SSH args) are all RED.

**Subsumption named, per the DoD.** `encodeVersion2`'s three directory-collection
passthroughs cannot be killed: `Migrate` reaches that encoder only with a
Configuration 1.0.0 source, and `rawV1` declares no directory collection, so
they are always empty. `TestVersion1SourcesNeverCarryDirectoryCollections` pins
that precondition from the derived member set and from a real load, and reddens
if Configuration 1.0.0 ever gains one. That is the subsuming check for
reviewer mutants R13 and R14.

## F1 — the read-only downgrade bound at the one-major step

`TestAssessCompatibilityPinsEveryPinnedVersionPairAtTheProductionEntry` drives
the full cross-product of pinned Configuration versions through the real
`AssessCompatibility` entry. The version list comes from the pinned catalog
(`catalog.Current()`), not from a literal, and the expected mode is derived from
a test-local major-version parse rather than from the production `compareSemver`
the assertion is meant to bound. Both directions are required: the test fails if
the cross-product exercised zero read-only or zero compatible pairs, so the gate
cannot pass by answering a single mode.

- Narrowing to `source.major > reader.major+1` (the rev3 survivor) reddens on
  `document_2.0.0_reader_1.0.0` and `document_3.0.0_reader_2.0.0`.
- Widening to `>=` reddens on all three equal-version pairs.
- Deleting the mode gate outright reddens.

## F3 — the temp-file half of "fsyncs it and the directory"

`faultMigrationFileSystem.CreateTemp` now wraps its `*os.File` in
`faultMigrationFile`, which counts and can fail the per-file `Sync`. Two cases
joined `TestMigrateLeavesTheSourceIntactAtEveryDurableFailurePoint`, so they
inherit its whole invariant set — typed error, source untouched, no
`.ax-config-*` staging leak, and a clean retry completes:

| Injected fault | Expected | Asserted |
| --- | --- | --- |
| file sync 1 (backup staging) | `ErrMigrationBackup` + `fs.ErrInvalid` | source untouched, no leak, retry green |
| file sync 2 (replacement) | `ErrMigrationWrite` + `fs.ErrInvalid` | source untouched, no leak, retry green |

The narrowing direction is pinned separately:
`TestMigrateFsyncsEveryStagedFileBeforeItBecomesVisible` asserts a successful
`v1 → 3.0.0` stages exactly two files and that `fileSyncCalls == createTempCalls`
— *every* staged file is fsynced, not just the last. Deleting `file.Sync()`
(N04) and narrowing it to the replacement only (N17) both redden.

## Non-blocking observations from rev3, also closed

| # | Observation | Action |
| --- | --- | --- |
| 1 | `encodeVersion2`'s round-trip `Decode` re-validation deletable, invisible to the derived inventory | Carries a `config-refusal-subsumed: v2 re-read` marker naming why it is defence in depth; the `encode target` propagation comment now names it; pinned in `TestRefusalSubsumptionInventoryIsPinned` |
| 2 | `TestRefusalSubsumptionInventoryIsPinned` compared only `len()` | Now resolves each `<file>: <clause>` pin to exactly one marked site whose source line names that clause, refuses duplicate claims, and refuses an unclaimed marked site. No production marker-format change |
| 3 | `writeAll`'s short-write guard deletable | `faultMigrationFile.Write` can report zero progress without an error; new `short write` case asserts `ErrMigrationBackup` + `io.ErrShortWrite` |
| 4 | Tightening the replacement mode survived | The migrated file's mode is now asserted **equal** to the source's `0640`, pinning both directions |
| 5 | Non-canonical semver core (`01.0.0`) survived, claimed subsumed | Not subsumed on the read-only branch, which never consults the known-version set: `AssessCompatibility("04.0.0", reader 3.0.0)` returns a successful read-only assessment under the mutant. Two cases added; R07 now RED |
| 6 | `MigrationResult.BackupPath` populated before `replaceDurably` | Production behavior unchanged deliberately — clearing it would also hide a backup that *was* published on the staging-remove failure path. The contract is now stated on the type: `BackupPath` is only guaranteed to exist when `Migrate` returned nil |

## Gates — real exit codes, standalone processes, no `tee`, no pipes

| Gate | Exit |
| --- | ---: |
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `gofmt -l .` | 0 (empty) |
| `go test ./... -cover -count=1` | 0 |
| `go test ./internal/config -race -count=1` | 0 |
| `go run ./internal/traceability/cmd/tracecheck` | 0 |
| `cataloggen -check` | 0 |
| `go generate ./internal/catalog` + `git diff --exit-code internal/catalog/catalog_gen.go` | 0 |
| `go mod verify` | 0 |
| `git diff --check` | 0 |
| `mutants-r5.py` (73 mutants, 4 bounded batches) | 0 |
| batch exits: `[0:18]`, `[18:36]`, `[36:55]`, `[55:73]` | 0, 0, 0, 0 |
| `go test ./internal/config/ -count=1` under `env -i`, empty `HOME`, `GOPROXY=off` | 0 |

`internal/config` coverage 92.3% → **93.1%**.

## Traceability

Four new tests joined `config-durable-migration` and one joined
`config-read-only-downgrade` in `internal/traceability/ownership.v0.5.0.json`.
That registry is protected by `reviewedOwnershipCanonicalSHA256`, which is a
deliberate review gate rather than a formatting checksum, so it moved from
`1abb92b1…` to `8885305e…` in the same change. `tracecheck` reports
`contracts=60 normative_sections=36 acceptance_cases=35 fixtures=30
compatibility_contracts=55`. No new section binding, no new acceptance case ID,
and no CLI, `doctor`, or backend capability is advertised.

## Incident worth recording

While updating the ownership registry I ran `git checkout --` on
`internal/traceability/ownership.v0.5.0.json` to undo a formatting-churning
rewrite, which also discarded ~100 lines of *uncommitted* prior-round work on
that file. Recovered byte-identically from the pre-edit tree copy under
`.temp/TASK-260830-1qf777/r5/tree/`, verified identical to
`d583fe8:internal/traceability/ownership.v0.5.0.json`. In an uncommitted
story-final candidate, a single-path `git checkout --` is as destructive as a
repo-wide one; the surgical text edit that followed is the pattern to keep.

## Artifacts

- `.temp/TASK-260830-1qf777/r5/mutants-r5.py` — 73-mutant harness (29 producer +
  18 reviewer rev3, both re-run against the *whole* package rather than a
  per-mutant `-run` mask, + 26 new)
- `.temp/TASK-260830-1qf777/r5/final-p{1,2,3,4}.log` — per-batch sweep transcripts
- `.temp/TASK-260830-1qf777/r5/full-cover.log` — repository-wide coverage run
- `.temp/TASK-260830-1qf777/r5/clean-env.log` — clean-environment determinism run
