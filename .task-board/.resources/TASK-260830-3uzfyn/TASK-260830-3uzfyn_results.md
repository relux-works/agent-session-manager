# TASK-260830-3uzfyn results — implement profile resolution and persistence

## Verdict

**Ready for review.** AC coverage **16 of 16 rows driven** through
the named production entry by the named committed test (matrix:
`TASK-260830-3uzfyn_conformance-matrix.md`). Full suite green (27
packages, `go test ./...`, `-race`, `-cover`), lint/build clean
both platforms, mutant battery 20/20 narrowing kills with the
harmless control survived. Candidate left **uncommitted** in the
Story worktree for the handoff snapshot.

## What shipped

New package `internal/sessprofile` (Section 2.4 derivation and
persistence) plus the provhost mapping resolver and launch
projection:

- `Derive` (session head) and `DeriveForHeads` (checkpoint
  event-head closure) — pure reducer over the validated record and
  the authoritative chain; newest `profile.changed` wins, closure
  events only, continuity re-checked, losing/ambiguous inputs
  refuse. `Projector` binds it to `*sessrepo.Repository` through
  read entries only.
- `MintChangeEvent` — pure `profile.changed` factory computing the
  omit-self digest via `canonicaljson.CalculateObjectIdentity`
  (never `Verify`, so the no-attestation-outside-the-leaf bound
  holds); enforces the vocabulary, from≠to, and the yolo
  confirmation rule.
- `Transactor.SetProfile` — the set-profile transaction: derives
  from, fences the acting lease against the chain head (stale,
  divergent, and greater-epoch acting leases refuse), mints, and
  appends through `sessrepo.AppendEvent`. Retries replay the
  committed change by request envelope
  (target/lease/author/instant).
- Pair projections and checks — `BundlePair`, `ForkProjection`,
  `FinalizePair`, `CheckLaunchPair`, `CheckResumedPair`,
  `CheckForkPair`, `CheckBundlePair`, `CheckResumeRequestProfile`,
  `CheckFinalizeParams`, `CheckBridgeProfile`,
  `ResolveCreationProfile` — the reducer-level pairs the fixtures
  require, each divergence refusing `integrity_failure`.
- `provhost.ResolveMapping` — Section 7.7 mapping against the exact
  probed `BuildTuple` (tuple gate reused, caller argument
  classified first); unmappable provider/version refuses the new
  `profile_mapping_unavailable` Structured Error (exit 6,
  provider/version/profile details); Pi pins 0.73.1 with the
  both-profiles equivalence disclosed.
- `provhost.ProjectLaunchArgv` — sanitized launch argv: the exact
  table flag appended once for yolo (Pi appends nothing, its
  mapping is report-only), base unchanged for standard; duplicate,
  changed, foreign, and standard-mode tokens (including the codex
  alias) refuse; Section 5.1 numeric bounds enforced as the
  host-side twin of the SpawnPlan wire bounds (agreement test at
  every limit).

Gate updates owned by this change: 12 new derived refusal arms,
all witnessed (floor 217 → 229, constructor list extended to the
seventh constructor); runtime audit extended (seventh wrapper,
closed code set + `profile_mapping_unavailable`); secprim
secret-site census gained the reviewed `unrestrictedTokens`
argv-words row. No traceability row touched (final-leaf owned).

## Validation observed (this session, real exits)

- `gofmt -l internal`: clean (433 files), exit 0.
- `go vet ./...` and `GOOS=windows go vet ./...`: exit 0.
- `go build ./...` and `GOOS=windows go build ./...`: exit 0.
- `go test ./... -count=1`: exit 0, 27 ok, 0 fail.
- `go test -race ./... -count=1`: exit 0, 27 ok.
- `go test ./... -cover -count=1`: exit 0
  (`internal/sessprofile` 92.6% of statements).
- Mutant battery `internal/sessprofile/testdata/mutate_profile.py`:
  exit 0; 20/20 applied N plants KILLED, each by its named
  behavioral test; `C-harmless-comment` SURVIVED (applied
  control); `C-not-applied` NOT_APPLIED and `C-compile-failure`
  COMPILE_OR_HARNESS_FAILURE classified outside the numerator;
  control-before/after green; sources restored byte-identical
  (asserted in-harness; digests below).
- Re-ran myself: everything above. Accepted from attached
  evidence: nothing (no prior evidence exists for this task).

## Specification traceability notes

- Implemented as the pinned v0.6.0 headings define (§2.4, §7.7,
  with §§5.5/8 as tuple authority); verified by diff that §2.4 and
  §7.7 are textually identical in v0.5.0, so the task's v0.5.0
  scope citation and the producer's v0.6.0 pin agree on this
  deliverable.
- The task text's "named profiles, CLI overrides, task-board
  profiles" predate the pin: in v0.6.0 they are the `standard|yolo`
  enum named by `ax start --profile` (creation), `ax session
  set-profile` (change), and the bridge launch/resume profile
  (derived pair). No named registry exists (stated bound B1).
- There is no time-expiring ownership lease in §5.3, so R3/R9
  implement "losing or expired" as chain-head mismatch (stale
  epoch, divergent token); see the matrix bounds B1–B8 for the
  full declared list.

## Files changed (candidate, uncommitted)

- `internal/sessprofile/`: `doc.go`, `profile.go`, `decode.go`,
  `project.go`, `mint.go`, `setprofile.go`, `pairs.go`, plus
  `fixtures_test.go`, `profile_test.go`, `mint_test.go`,
  `setprofile_test.go`, `pairs_test.go`, `fixtures_e2e_test.go`,
  `crash_test.go`, `testdata/mutate_profile.py`.
- `internal/provhost/profile_resolve.go`,
  `internal/provhost/profile_resolve_test.go`,
  `internal/provhost/refusal_arm_operations_e_test.go`,
  `internal/provhost/protocol.go` (constructor only),
  `internal/provhost/refusal_arm_inventory_test.go` (list, floor,
  aggregation), `internal/provhost/inventory_test.go` (audit).
- `internal/secprim/census_test.go` (one reviewed row).
- `README.md` (package section, no positive claim),
  `LOGBOOK.md` (entry).

## Evidence attached

- `TASK-260830-3uzfyn_results.md` (this file).
- `TASK-260830-3uzfyn_conformance-matrix.md` (16/16 + mutant
  census + bounds).
- `TASK-260830-3uzfyn_producer-evidence.tar.gz` (mutant logs +
  `mutants.json`, digests, full/race/cover/lint/build logs).
