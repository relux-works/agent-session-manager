# TASK-260830-21gygk results — producer run RUN-260909-f0fdb3

Run: RUN-260909-f0fdb3 (developer) · Story worktree `.temp/STORY-260830-3tq4ns/worktree`
Branch: `task-board/story/STORY-260830-3tq4ns` at `7208cc7427e1ebe95106d34f98214e2bbea4ae1c` (unmoved; candidate UNCOMMITTED, no commit by this run)
Authority: relux-works/agent-session-manager-spec v0.6.0 commit `0cbdf100dbf84df50c64f792b1f940e3a67859a6`
Scope: §§14.7/14.7.1 + shared SelectionPlan construction/revalidation in §14.7.2, refining §2.3 and §§5.7/14.7.3

This run kept the preserved candidate, corrected one evidence miscitation,
and re-derived validation on the final tree. No `.go` byte changed in this run.

## Change made in this run

`internal/sessquery/TRACEABILITY.md` (2 table cells, docs only): the AC row
and the exact-name gate row cited `TestResolveLocalNameAndUUID` as a driver
of the shared `Reader.Resolve` call site, but that test lives in `sessrepo`
and drives the retained `Repository.Resolve` entry. Row 1 now names only the
three sessquery drivers and notes the repository-layer pin separately; the
gate row now states each narrowing mutant with its real killing test and
entry point. Kill attributions below prove both sentences.

## AC coverage: 8 of 8 rows driven at the shared library boundary

| AC row | Production call site | Named test(s) |
| --- | --- | --- |
| Bare UUID/name selectors (§2.3 tiers preserved) | `Reader.Resolve` → `resolveBare` | `TestResolvePinnedPrecedence`, `TestResolveBareIdentityUnion`, `TestResolveAllowlistAndReadFailures` (+ repo-layer pin `TestResolveLocalNameAndUUID` at `Repository.Resolve`) |
| Qualified selectors (first-`@`, no fallback) | `Reader.Resolve` → `resolveExplicit` → `locateSource` | `TestResolveQualifiedSourcesSelectOneIndex`, `TestResolveQualifiedSourceNeverFallsBack`, `TestResolveQualifiedNameBeforeUUIDInSource`, `TestResolveExplicitSourceRefusals`, `TestResolveExplicitSourceReadFailures` |
| Ambiguity | `matchName`; `selectIdentity`/`checkRecordAgreement` | `TestResolveExactNamesAndASCIICollisions`, `TestResolvePeerOrderAndReplicatedIdentity`, `TestResolveQualifiedAmbiguityAndExclusions` |
| List summaries | `Reader.List` | `TestListStatusDerivedFactsAndStableOrder`, `TestListStatusCheckpointAndProjectionFailure`, `TestParkedReadRecoveryAndMissingVsMalformed` |
| Status summaries | `Reader.Status`, `Reader.InspectLocal` | Same list tests; `TestCreatingSummaryCannotClaimClosedCLIResult`; qualified status rows |
| Stable deterministic sorting | `Reader.read`, `peerCandidates`, `selectIdentity` | `TestListStatusDerivedFactsAndStableOrder`, `TestResolvePeerOrderAndReplicatedIdentity`, `TestResolveBareIdentityUnion` |
| SelectionPlan build + revalidate | `Reader.BuildPlan`, `Reader.Revalidate`, `ParsePlan`, `BoundariesFor` | `TestBuildPlanBindsCurrentFacts`, `TestBuildPlanPeerSourceFacts`, `TestBuildPlanRecordOnlySessionBindsEmptyLease`, `TestPlanDigestAndMarshalRoundTrip`, `TestRevalidateDetectsEachFactChange`, `TestRevalidateReadFailures`, `TestRevalidateMalformedPlans`, `TestActionBoundaryTable` |
| Negative/refusal cases, no unsupported capability | All gates in `internal/sessquery/TRACEABILITY.md` | Every refusal arm above; 29-mutant battery (prior re-derivation stands, code bytes unchanged); bounds documented |

0 of 8 rows delivered as public CLI: no `ax` session command exists in this
tree. Wire exits, transports, auth, observations, and effects stay with owning leaves.

## Validation log (real exits, this run, final tree)

- `go build ./...` → exit 0
- `go test ./... -count=1` → every package ok, exit 0; second full pass grepped for FAIL/panic → none
- `go test ./internal/sessquery/ -count=1 -cover` → ok, 91.0% of statements
- All 27 named AC tests above run by name → PASS (incl. subtests of `TestRevalidateDetectsEachFactChange`, `TestRevalidateReadFailures`, `TestRevalidateMalformedPlans`, `TestSelectorConfigurationValidation`)
- `go test ./internal/sessquery ./internal/sessrepo ./internal/sessstate -race -count=1` → all three ok
- `go vet ./...` → clean; `gofmt -l internal` → empty (clean)
- Targeted narrowing proofs for the corrected citations, run in an isolated
  `/tmp/verify-cite` source copy (managed worktree never mutated; scratch copy
  restored to faithful bytes afterwards):
  - N-case-variant (admit `az-09._` at shared tier) → `TestResolveExactNamesAndASCIICollisions` FAILS:
    `query_test.go:64: "az-09._": <nil>` (local + peers subtests)
  - N-repository-case-variant (admit `Payments-API` at retained repo entry) → sessrepo `TestResolveLocalNameAndUUID` FAILS:
    `sessrepo_test.go:868: Resolve(non-exact case variant) error = <nil>, want errors.Is session name not found`
  - Full 29-mutant battery stands on identical code bytes from RUN-260909-754c69 (29 KILLED, 0 survivors, controls distinct); no `.go` byte changed since.

## Trunk-combination re-verification (this run, byte-level)

- Merge-base with origin/main: `7654d7ca`; origin/main at `8cf4aaaa` (unchanged since prior runs).
- All 22 trunk-owned tracked files byte-identical to `origin/main` (verified with `git show origin/main:<path> | cmp -`; catalog, cataloggen, cigate, config tests, localstore, specdoc, specpin, traceability, task-board.config.json).
- All 8 untracked adoption files byte-identical to `origin/main` (catalog.v0.6.0.json, SPEC.v0.6.0.md, specdoc060_test.go, pin060_test.go, sections060_test.go, v0.6.0.lock.json, adoption-v0.6.0.md, ownership.v0.6.0.json).
- Only README.md and LOGBOOK.md differ from trunk (merged additions); `internal/provhost/identity.go` delta vs trunk is story-committed predecessor content (identical vs fork point `7654d7ca`), not missing trunk content. sessrepo/sessstate/sessquery are story-only and intact.
- Predecessor checkpoints (1r9wrr rev3, wbpf1v rev4) untouched; no destructive checkout/reset/clean; no commit on the Story branch.

## Base-recovery refusal (recorded, not bypassed)

`task-board worktree refresh-candidate TASK-260830-21gygk` (run with my live
producer run RUN-260909-f0fdb3 holding the exclusive STORY-260830-3tq4ns lease,
no staged changes, no pending review for this leaf) refused with exact output:
`change_request_not_found: TASK-260830-21gygk has no Change Request record`.
Same refusal as RUN-260909-c41bb6. The recovery needs an existing CR while CR
construction needs the refreshed base (`change_request_base_authority_mismatch`:
checkpoint `7208cc7` does not descend from authority `8cf4aaa`); resolving that
circle is orchestrator/board mechanics, outside the developer role. The working
candidate already combines all reviewed trunk content (proven above), so no
producer-side merge remains.

## Stated bounds (unchanged)

- `lease_record_id` unbound (no Lease Record objects in this stack; winning envelope lease triple bound instead; nothing fabricated).
- Record-only chains keep the accepted internal projection with empty owner/lease — an interrupted persistence prefix for the bootstrap-recovery leaf (14.7.4), never a claimed public creating state; no nullable or placeholder owner/lease field (see `TestCreatingSummaryCannotClaimClosedCLIResult` and the pinned prefix clarification).
- Wire exits, CLI Result 5, transports, peer auth, host metadata, and all observation contracts stay with owning leaves; configuration and learned indexes are trusted caller inputs; the allowlist proof covers filtering only.
- Plans authorize read projection only; effect boundaries belong to callers via `BoundariesFor` + `Revalidate`.

## Candidate files

New: internal/sessquery/{selector,plan,revalidate,digest}.go,
selector_test.go, resolve_test.go, plan_test.go; updated query/bounds tests,
mutate.py, TRACEABILITY.md (this run: citation correction), README.md,
LOGBOOK.md; sessrepo exact-name rule retained. Full evidence map:
internal/sessquery/TRACEABILITY.md.

Ready for review (producer handoff; independent Astra review and orchestrator
integration own the remaining path, including the CR base-authority circle).
