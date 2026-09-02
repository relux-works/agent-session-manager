# TASK-260902-24e45b — Object-store reapply onto current trunk

Branch `task-board/story/STORY-260902-227fzs`, commit `5e3e108`, tree
`149dbbaf25ec2fe6d065cd3c34ceae975b4dce80`, signed (ECDSA
`SHA256:V6JiKG7J29mjsvikcLoSVp0bLa77VTsFy12gnLO81cM`, `git verify-commit` good).

## Provenance of the input

The attached patch series was verified against the accepted commits before any
merge work. Applied with `git apply --index` at base `48db30b` in a scratch
worktree it produces tree `3d0f74df25b64bb182a0bec75c4691d7048d6c2a`, which is
byte-equal to `81305c8^{tree}`. The reapply was then performed as a single
cherry-pick of a squash commit over `48db30b..81305c8` onto trunk `010d114`,
so the two leaves land as one commit without an intermediate resolution pass.

## Byte-identity assertion

Machine-checked over the full trees, not sampled:

| Assertion | Result |
| --- | --- |
| Files changed by the accepted leaves vs base `48db30b` | 26 |
| Of those, conflicted | 8 |
| Of those, non-conflicted | 18 |
| Non-conflicted leaf files byte-identical to accepted tree `81305c8` | 18/18 |
| Every file the leaves did not touch byte-identical to trunk `010d114` | all |
| Files removed relative to trunk | none |

All 18 byte-identical files are under `internal/localstore/`. No file outside
the 26 the leaves touched differs from trunk.

## Conflict resolutions

Eight files conflicted. Every resolution is a union merge; every derived value
was recomputed from the gate that owns it.

| File | Shape | Resolution |
| --- | --- | --- |
| `internal/traceability/ownership.v0.5.0.json` | one text conflict on the `section:10.1` acceptance array, plus one **structural** collision git auto-merged without a marker | arrays unioned; the two `section:3.2` bindings merged into one — see below |
| `internal/traceability/traceability.go` | `reviewedOwnershipCanonicalSHA256` | recomputed from the `traceability.go:286` refusal against the merged registry: `b5882b265b29d0c286c62a0263a6a38444d972434a5c5d42569efe6fc3af0b2c` |
| `internal/traceability/traceability_test.go` | `AcceptanceCases` count | base 29 + trunk 9 + accepted 5 = **43**, confirmed by `tracecheck` output, not asserted |
| `internal/traceability/cmd/tracecheck/main_test.go` | two `acceptance_cases=` CLI strings, the scalar-section list, the scoped-owner mutant table | counts → 43; section list and mutant table union-merged (14 sections, 11 mutant rows) |
| `README.md` | acceptance-case sentence; Tools row | sentence → 43; Go toolchain row purpose/commands/outputs unioned, scoped `tracecheck` invocation now lists 30 sections; `go-toml`, `modernc.org/sqlite` and `golang.org/x/sys` rows all retained |
| `go.mod` | require blocks | union of both dependency sets, normalized by `go mod tidy` |
| `go.sum` | hash lines | union; result equals the accepted side plus exactly the two `go-toml v2.4.3` lines from trunk |
| `LOGBOOK.md` | add/add (the file did not exist at `48db30b`) | day blocks merged, `2026-09-01` entries interleaved by their own timestamps: 2340, 2320, 2300, 2245, 2129, 2113 |

## The one deviation that is reported rather than absorbed

Git auto-merged **both sides' `section:3.2` bindings** into the ownership
registry as two separate array entries. That produces no conflict marker but is
refused by `verifyOwnershipGroups` at `internal/traceability/traceability.go:404`
as `duplicate section_binding implementation owner`.

The underlying fact is not a merge artifact. Trunk (`cc89771`,
STORY-260830-jeaivu) and the accepted leaf independently implemented the **same
SPEC.md §3.2 precedence rule** — command flag → non-empty `AX_*` environment
override → platform default, over the same five path classes:

- `internal/config/loader.go:ResolvePaths`, exercised by `AC-PATH-001` (20 tests)
- `internal/localstore/paths.go:ResolvePaths`, exercised by
  `localstore-path-registry` (7 tests)

Neither package imports the other; neither has a caller outside its own package
(no `cmd/ax` exists yet). §3.2 "Platform paths" genuinely spans both halves —
precedence *and* the durable data layout / `digest_path_v1` / owner-only modes —
so both Stories were locally correct. **This needs a product/architecture
ownership decision and is not resolved here.**

What was done, which is a registry merge and not a code decision: the two
bindings were merged into one, keeping `internal/localstore/paths.go:ResolvePaths`
as `production` and union-merging `AC-PATH-001` into its `acceptance_cases`.
Both production implementations remain in the tree untouched.

Rationale for that direction, and why it loses no evidence:

- Nothing else in the registry pins the localstore `ResolvePaths` declaration —
  `localstore-path-registry` pins `PathDefinitions`, `localstore-layout-owner-only`
  pins `InitializeLayout`. The accepted leaf's `-section 3.2` mutant row therefore
  depends on the binding's `production` field alone.
- Trunk's §3.2 evidence survives through `AC-PATH-001`, whose own `production` is
  `internal/config/loader.go:Load` and which is now verified inside scope 3.2.
- Trunk's `TestRunAssignedConfigPathSectionUsesScopedImplementationOwner` asserts
  only `assigned_scopes=1` for `-section 3.2`, which is unaffected.
- Net effect: scoped `tracecheck -section 3.2` now verifies **both** packages,
  where before the merge each side verified only its own.

## Gates

Every command below was run as a standalone process on the committed tree and
its real exit code is reported.

| Gate | Command | Exit |
| --- | --- | ---: |
| Format | `gofmt -l .` (0 files) | 0 |
| Build | `go build ./...` | 0 |
| Vet | `go vet ./...` | 0 |
| Tests | `go test ./... -count=1` | 0 |
| Coverage | `go test ./... -cover -count=1` | 0 |
| Traceability, global | `go run ./internal/traceability/cmd/tracecheck` | 0 |
| Traceability, scoped | same with 30 `-section` flags | 0 |
| Catalog | `cataloggen ... -check` | 0 |
| Cross-build | `GOOS=linux GOARCH=amd64 go build ./...` | 0 |
| Cross-build | `GOOS=windows GOARCH=amd64 go build ./...` | 0 |
| Cross-vet | `GOOS=linux GOARCH=amd64 go vet ./...` | 0 |
| Cross-vet | `GOOS=windows GOARCH=amd64 go vet ./...` | 0 |
| Fuzz | `FuzzScalarProductionEntries` `-fuzztime=100x` | 0 |
| Fuzz | `FuzzCanonicalizeRoundTrip` `-fuzztime=100x` | 0 |
| Fuzz | `FuzzObjectIdentityRepresentationInvariant` `-fuzztime=100x` | 0 |
| Fuzz | `FuzzClosedIdentityShapeRefusal` `-fuzztime=100x` | 0 |
| Fuzz | `FuzzObservationEventRefusal` `-fuzztime=100x` | 0 |

Global tracecheck output on the merged tree:

```
traceability ok: contracts=60 normative_sections=36 acceptance_cases=43 fixtures=30 compatibility_contracts=55 assigned_scopes=0
```

One gate was deliberately run red and is reported as red: with the digest
stubbed to all-zeroes, `tracecheck` exits **1** with
`ownership registry projection digest b5882b26... differs from reviewed 0000...`.
That refusal is where the reviewed digest was read from; it is not a passing run.

## Coverage, measured on all three trees

| Package | Trunk `010d114` | Accepted `81305c8` | Merged `5e3e108` | Delta |
| --- | ---: | ---: | ---: | --- |
| `internal/canonicaljson` | 97.2% | 87.1% | 97.2% | = trunk (max) |
| `internal/catalog` | 97.6% | 97.6% | 97.6% | = |
| `internal/catalog/cmd/cataloggen` | 79.3% | 79.3% | 79.3% | = |
| `internal/cataloggen` | 83.9% | 83.9% | 83.9% | = |
| `internal/config` | 94.7% | n/a | 94.7% | = |
| `internal/localstore` | n/a | 83.8% | 83.8% | = |
| `internal/scalar` | 90.1% | 90.1% | 90.1% | = |
| `internal/specpin` | 85.1% | 85.1% | 85.1% | = |
| `internal/traceability` | 85.0% | 85.0% | 85.0% | = |
| `internal/traceability/cmd/tracecheck` | 87.5% | 87.5% | 87.5% | = |

No package regresses against either baseline. Baselines were re-measured in a
scratch worktree at each commit, not quoted from the leaves' own records.

## Negative evidence for the merge resolutions

`merge-mutants.sh`, 3 baseline controls plus 14 mutants. Restores are taken
from the git index; the tree hash before and after the sweep is identical
(`5ff09ef`).

| Mutant | What it narrows or perturbs | Gate | Expected | Observed |
| --- | --- | --- | --- | --- |
| m00 ×3 | nothing (baseline must be green first) | tracecheck, traceability pkg, scoped-owner suite | GREEN | GREEN |
| m01 | one hex digit of the recomputed digest | `tracecheck` | RED | RED (1) |
| m02 | same flip | `go test ./internal/traceability` | RED | RED (1) |
| m03 | digest copied from the trunk side | `tracecheck` | RED | RED (1) |
| m04 | digest copied from the accepted side | `tracecheck` | RED | RED (1) |
| m05 | `AcceptanceCases` pinned to trunk's 38 | traceability pkg | RED | RED (1) |
| m06 | `AcceptanceCases` pinned to accepted's 34 | traceability pkg | RED | RED (1) |
| m07 | CLI string pinned to 38 | `TestRunReportsExactCoverageAndFailsClosed` | RED | RED (1) |
| m08 | localstore `ResolvePaths` renamed | `tracecheck -section 3.2` | RED | RED (1) |
| m09 | config `Load` renamed (the union-merged `AC-PATH-001` owner) | `tracecheck -section 3.2` | RED | RED (1) |
| m10 | `AC-PATH-001` dropped from the merged 3.2 binding | `tracecheck` | RED | RED (1) |
| m11 | the duplicate `section:3.2` binding reintroduced | `tracecheck` | RED | RED (1) |
| m12 | scalar-section list narrowed to the accepted side (control) | scalar-sections suite | GREEN | GREEN |
| m13 | §5.1 owner renamed, merged list in place | scalar-sections suite | RED | RED (1) |
| m14 | §5.1 owner renamed **and** list narrowed (anti-vacuity) | scalar-sections suite | GREEN | GREEN |

m12/m13/m14 are the pairing that matters: m13 alone would only prove the mutant
row exists. m14 shows the same rename goes **undetected** under the narrowed
list, so m13's kill is attributable to the union merge of the section list and
not to some other check in the suite.

m03/m04 are the anti-copy controls for the derived digest: neither side's
pre-merge value passes over the merged registry, so the recomputation was
necessary rather than cosmetic.

m11 is the control for the §3.2 resolution itself: it reproduces the exact
duplicate-owner refusal that forced the merge, proving the one-owner rule is
enforced and the merge was not an arbitrary choice between two acceptable
registries.

## Not gated, stated honestly

The README acceptance-case sentence is prose. `tracecheck` pins
`AcceptanceCases` and the two CLI strings; nothing reads the README. It was set
to 43 to match `len(acceptance_cases)`, and it can drift again without any gate
noticing. Same for the README tools row and the scoped `tracecheck` command line
recorded in it.

## Follow-up owed

SPEC.md §3.2 has two independent production implementations of the same
precedence rule. The registry can name only one owner and now names the
localstore one. Someone needs to decide whether `internal/config` should consume
`internal/localstore.ResolvePaths`, whether the localstore resolver should be
dropped in favour of the config one, or whether the two roles are genuinely
distinct and the section should be split. Nothing in this task's scope makes
that call.
