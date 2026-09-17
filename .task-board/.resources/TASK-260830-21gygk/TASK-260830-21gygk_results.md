# TASK-260830-21gygk — results (RUN-260909-fe37d7, final verification)

Scope: v0.6.0 selector contract (§§14.7/14.7.1, shared SelectionPlan
construction/revalidation §14.7.2, refining §2.3, authoritative summaries
§§5.7/14.7.3). Authority: spec v0.6.0 commit
`0cbdf100dbf84df50c64f792b1f940e3a67859a6` via STORY-260908-18woqo.
Historical v0.5.0 scope retained without weakening. No `ax` session surface
exists in this tree, so CLI/Result-5 delivery, wire exits, lifecycle effects,
and the §14.7.4 BootstrapIntent recovery flow are caller/recovery-leaf owned;
every assigned shared-library behavior exists and is driven below.

## Acceptance: 8 of 8 AC rows driven at the shared library boundary

Production call sites and named tests (full matrix in
`internal/sessquery/TRACEABILITY.md`):

1. Bare UUID/name selectors — `Reader.Resolve → resolveBare`
   (`TestResolvePinnedPrecedence`, `TestResolveBareIdentityUnion`,
   `TestResolveAllowlistAndReadFailures`; retained repository entry
   `Repository.Resolve` pinned by `TestResolveLocalNameAndUUID` in sessrepo).
2. Qualified selectors — `Reader.Resolve → resolveExplicit → locateSource`
   (`TestResolveQualifiedSourcesSelectOneIndex`,
   `TestResolveQualifiedSourceNeverFallsBack`,
   `TestResolveQualifiedNameBeforeUUIDInSource`,
   `TestResolveExplicitSourceRefusals`,
   `TestResolveExplicitSourceReadFailures`).
3. Ambiguity — `matchName`, `selectIdentity`/`checkRecordAgreement`
   (`TestResolveExactNamesAndASCIICollisions`,
   `TestResolvePeerOrderAndReplicatedIdentity`,
   `TestResolveQualifiedAmbiguityAndExclusions`).
4. List summaries — `Reader.List → Repository.ListSessions → Projector.Project`
   (`TestListStatusDerivedFactsAndStableOrder`,
   `TestListStatusCheckpointAndProjectionFailure`,
   `TestParkedReadRecoveryAndMissingVsMalformed`). Whole-read refusal on
   projection failure is driven; no silent omit.
5. Status summaries — `Reader.Status → Resolve → Projector.Project`,
   `Reader.InspectLocal`
   (`TestCreatingSummaryCannotClaimClosedCLIResult` plus list tests and
   qualified status rows). Record-only chains keep the accepted internal
   projection with empty owner/lease; no public creating state claimed.
6. Stable deterministic sorting — `Reader.read`, `peerCandidates`,
   `selectIdentity` (order tests above plus `TestResolveBareIdentityUnion`).
7. SelectionPlan build/revalidate — `Reader.BuildPlan → bindPlan`,
   `Reader.Revalidate`, `ParsePlan`, `BoundariesFor` (8 plan tests incl.
   `TestRevalidateDetectsEachFactChange` per-member arms).
8. Negative/refusal cases — every gate below, each with a narrowing mutant.

0 of 8 rows delivered as a public CLI surface: stated bound, not a gap in
assigned shared behavior. No unsupported capability advertised.

## Validation rerun by this run (all on the final candidate, real exits)

| Command | Exit | Log |
| --- | --- | --- |
| `go test ./... -count=1` | 0 | `.temp/TASK-260830-21gygk/full-tests.log` (all packages ok) |
| `go vet ./...` | 0 | `.temp/TASK-260830-21gygk/vet.log` |
| `go build ./...` | 0 | `.temp/TASK-260830-21gygk/build.log` |
| `gofmt -l internal/sessquery/ internal/sessrepo/` | 0, empty | `.temp/TASK-260830-21gygk/gofmt.log` |
| `git diff --check` | 0 | `.temp/TASK-260830-21gygk/diff-check.log` |
| `go test ./internal/sessquery/ -count=1 -cover` | 0 | 91.0% statements |
| `go test ./internal/sessquery/ ./internal/sessrepo/ ./internal/sessstate/ -race -count=1` | 0 | `.temp/TASK-260830-21gygk/race.log` |

## Narrowing-mutant battery rerun by this run

Runner `internal/sessquery/testdata/mutate.py` into
`.temp/TASK-260830-21gygk/mutants/` (isolated source copy; managed checkout
untouched; console log `mutants-console.log`, machine table
`mutants/mutants.json`):

- 29 of 29 behavioral mutants KILLED (26 narrowing N-*, 3 ordering B-*),
  every kill names failing behavioral tests (see console log).
- `control-before` and `control-after` exit 0 (green controls, not kills).
- `C-not-applied` NOT_APPLIED and `C-compile-failure`
  COMPILE_OR_HARNESS_FAILURE: correctly not kills.
- Token-preserving grammar mutants included and killed through the
  behavioral suites (not only a static check): `N-first-at-split` (keeps
  `@`, splits last), `N-local-prefix` (keeps `local`, admits prefix),
  `N-id-case` (keeps `id:`, folds prefix), `N-peer-alias-fold` (keeps
  mapping, folds comparison).
- Corroboration: the earlier `.temp/TASK-260830-21gygk/sel-mutants-final/`
  battery classifies all 33 entries identically (same keys, same classes).

## Candidate integrity

- Worktree `task-board/story/STORY-260830-3tq4ns` at `7208cc7427e1`, dirty
  with the uncommitted candidate; no commit made on the Story branch.
- Story lease held by this run (RUN-260909-fe37d7). No CR exists yet for
  this leaf (initial candidate); predecessors 1r9wrr-rev3 and wbpf1v-rev4
  remain checkpointed and untouched.
- Trunk combination verified byte-identical with `origin/main`
  (`8cf4aaaa190e`): `SPEC.v0.6.0.md`, `catalog.v0.6.0.json`,
  `v0.6.0.lock.json`, `ownership.v0.6.0.json`, `adoption-v0.6.0.md`.
  Story-only delta: `internal/sessquery/` (new shared API + tests +
  TRACEABILITY + mutate runner), exact-name fix in `sessrepo/store.go`
  with corrected `sessrepo_test.go` arm, README selector section,
  LOGBOOK entries. `internal/sessquery/` is absent from `origin/main`,
  as expected for this leaf.
- Prefix clarification applied: `TestCreatingSummaryCannotClaimClosedCLIResult`
  documents the record-only chain as an interrupted persistence prefix for
  the §14.7.4 recovery leaf, not a public creating state; no nullable or
  placeholder owner/lease field added.

## Stated bounds (declared, not implemented)

- `lease_record_id` unbound: the stack carries no Lease Record objects
  (verified: no `LeaseRecord` type in sessstate/sessrepo; winner is the
  envelope `LeaseHead`), so the plan binds the validated winning envelope
  triple (epoch/lease-ID/owner) and documents the delta in `plan.go` and
  TRACEABILITY. No digest fabricated.
- `authority_heads` binds validated event-head digests of the plan source
  only; union precedence facts at build time are not re-checked.
- Wire exit codes (CLI Result 5 / Structured Error 1.4.0), the §14.7.3
  public-document list refusal, §14.7.4 recovery, effect authorization, and
  the composed attach/logs invocation-to-plan bindings are caller-leaf
  owned; plans built here authorize read projection only, and
  `BoundariesFor` names every boundary callers must revalidate at.
- Reads across calls are not an atomic snapshot; transports, peer
  authentication, freshness, host metadata, and observation contracts
  remain upstream: configuration and learned indexes are trusted caller
  inputs, and the allowlist proof covers filtering, never attestation.
