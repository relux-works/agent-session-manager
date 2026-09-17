# TASK-260830-21gygk results — resume re-validation (RUN-260909 resume)

Branch: `task-board/story/STORY-260830-3tq4ns` at `7208cc7427e1ebe95106d34f98214e2bbea4ae1c` (unmoved; candidate UNCOMMITTED)
Authority: relux-works/agent-session-manager-spec v0.6.0 commit `0cbdf100dbf84df50c64f792b1f940e3a67859a6`
Scope: §§14.7/14.7.1 + shared SelectionPlan construction/revalidation §14.7.2, refining §2.3 and §§5.7/14.7.3
Prior evidence: RUN-260909-c41bb6 partial outcome (preserved implementation, this run re-validates it)

## This run's delta

- `internal/sessquery/plan.go` header: corrected "two members stay unbound" to
  exactly one (`lease_record_id`); authority heads bind validated event-head
  digests of the plan source. Comment-only; matches implementation,
  `internal/sessquery/TRACEABILITY.md` bounds, and the §14.7.2 table.
- `LOGBOOK.md`: resume entry recording scope check, fix, and re-validation.
- No behavior change; no state/error invented; record-only prefix routing
  unchanged (creating = record AND initial lease per §§5.7/13.1 step 2;
  recovery belongs to the §14.7.4 leaf; no nullable/placeholder owner/lease).

## Trunk combination (re-verified this run)

- `git fetch origin`; `origin/main` = `8cf4aaa`; merge-base with HEAD = `7654d7c`.
- Every shared path byte-identical with `origin/main`: catalog, cataloggen,
  cigate, config tests, localstore, specdoc, specpin (+`v0.6.0.lock.json`,
  `SPEC.v0.6.0.md`, `specdoc060_test.go`, `pin060_test.go`,
  `sections060_test.go`), traceability (+`adoption-v0.6.0.md`,
  `ownership.v0.6.0.json`), `catalog.v0.6.0.json`, `task-board.config.json`.
- Intentional deltas only: README/LOGBOOK story additions; provhost comment
  adapted to the story stack (trunk deleted sessrepo/sessstate, this story
  retains them); story-only `internal/sessrepo/`, `internal/sessstate/`,
  `internal/sessquery/` preserved. No destructive checkout/reset/clean.

## AC coverage: 8 of 8 rows driven at the shared library boundary

| AC row | Production call site | Named test(s) |
| --- | --- | --- |
| Bare UUID/name selectors (§2.3 tiers preserved) | `Reader.Resolve` → `resolveBare` | `TestResolvePinnedPrecedence`, `TestResolveLocalNameAndUUID`, `TestResolveBareIdentityUnion`, `TestResolveAllowlistAndReadFailures` |
| Qualified selectors (first-`@`, no fallback) | `Reader.Resolve` → `resolveExplicit` → `locateSource` | `TestResolveQualifiedSourcesSelectOneIndex`, `TestResolveQualifiedSourceNeverFallsBack`, `TestResolveQualifiedNameBeforeUUIDInSource`, `TestResolveExplicitSourceRefusals`, `TestResolveExplicitSourceReadFailures` |
| Ambiguity | `matchName`; `selectIdentity`/`checkRecordAgreement` | `TestResolveExactNamesAndASCIICollisions`, `TestResolvePeerOrderAndReplicatedIdentity`, `TestResolveQualifiedAmbiguityAndExclusions` |
| List summaries | `Reader.List` | `TestListStatusDerivedFactsAndStableOrder`, `TestListStatusCheckpointAndProjectionFailure`, `TestParkedReadRecoveryAndMissingVsMalformed` |
| Status summaries | `Reader.Status`, `Reader.InspectLocal` | Same list tests; `TestCreatingSummaryCannotClaimClosedCLIResult`; qualified status rows |
| Stable deterministic sorting | `Reader.read`, `peerCandidates`, `selectIdentity` | `TestListStatusDerivedFactsAndStableOrder`, `TestResolvePeerOrderAndReplicatedIdentity`, `TestResolveBareIdentityUnion` |
| SelectionPlan build + revalidate | `Reader.BuildPlan`, `Reader.Revalidate`, `ParsePlan`, `BoundariesFor` | `TestBuildPlanBindsCurrentFacts`, `TestBuildPlanPeerSourceFacts`, `TestBuildPlanRecordOnlySessionBindsEmptyLease`, `TestPlanDigestAndMarshalRoundTrip`, `TestRevalidateDetectsEachFactChange`, `TestRevalidateReadFailures`, `TestRevalidateMalformedPlans`, `TestActionBoundaryTable` |
| Negative/refusal cases, no unsupported capability | All gates in TRACEABILITY.md | Every refusal arm above; 29-mutant battery below; bounds documented |

0 of 8 rows delivered as public CLI: no `ax` session command exists. CLI Result 5,
wire exits, transports, auth, observations, and effects stay with owning leaves.

## Validation log (real exits, final candidate, this run)

- `go build ./...` → exit 0
- `go test ./... -count=1` → all 26 packages ok, exit 0
- `go test ./internal/sessquery/ -count=1 -cover` → ok, 91.0% of statements
- `go vet ./...` → clean, exit 0
- `gofmt -l internal` (+ `internal/sessquery/`) → empty (clean)
- `python3 internal/sessquery/testdata/mutate.py .temp/TASK-260830-21gygk/sel-mutants-final` → 29 KILLED, 0 survivors;
  controls: 2 CONTROL (before/after green), 1 NOT_APPLIED distinct, 1 COMPILE_OR_HARNESS_FAILURE distinct

## Mutant summary (all KILLED; token-preserving grammar mutants marked *)

N-collision-pair, N-case-variant, N-parked-id, N-tombstoned-id,
N-unlisted-peer, N-duplicate-peer, N-read-failure-fallback,
N-projection-failure-fallback, N-repository-case-variant, N-first-at-split *,
N-local-prefix *, N-id-case *, N-peer-alias-fold, N-id-through-names,
N-explicit-fallback, N-unknown-alias-local, N-disallowed-admit, N-dup-alias,
N-rev-revocation, N-rev-config, N-rev-binding, N-rev-index, N-rev-record,
N-rev-lease, N-rev-heads, N-rev-absent-substitute, B-peer-order,
B-session-order, B-tie-break. Per-mutant logs + mutants.json in
`.temp/TASK-260830-21gygk/sel-mutants-final/` (git-ignored scratch; names,
exits, and failed tests re-derived above).

## Stated bounds (unchanged)

- `lease_record_id` unbound (no Lease Record objects in this stack; winning
  envelope lease triple bound instead; nothing fabricated).
- Record-only chains keep the accepted internal projection with empty
  owner/lease; §14.7.3 list-refusal and §14.7.4 recovery belong to owning leaves.
- Wire exits, CLI Result 5, transports, peer auth, host metadata, and all
  observation contracts stay with owning leaves; configuration and learned
  indexes are trusted caller inputs; the allowlist proof covers filtering only.
- Plans authorize read projection only; effect boundaries belong to callers via
  `BoundariesFor` + `Revalidate`.

## Candidate files

New: internal/sessquery/{selector,plan,revalidate,digest}.go,
selector_test.go, resolve_test.go, plan_test.go; updated query/bounds tests,
mutate.py, TRACEABILITY.md, README.md, LOGBOOK.md; sessrepo exact-name rule
retained. Full evidence map: internal/sessquery/TRACEABILITY.md.

Ready for review (producer handoff; independent Astra review and orchestrator
integration own the remaining path).
