# TASK-260830-21gygk — results (RUN-260909-bd27d9, producer handoff)

Scope: v0.6.0 selector contract (§§14.7/14.7.1, shared SelectionPlan
construction/revalidation §14.7.2, refining §2.3, authoritative summaries
§§5.7/14.7.3). Authority: spec v0.6.0 commit
`0cbdf100dbf84df50c64f792b1f940e3a67859a6` via STORY-260908-18woqo
(landed on origin/main as 5d98b08). Historical v0.5.0 scope retained
without weakening. No `ax` session surface exists in this tree, so CLI
delivery, wire exits, lifecycle effects, and the §14.7.4 recovery flow
are caller/recovery-leaf owned; every assigned shared-library behavior
exists and is driven below.

## Branch and candidate integrity (this run)

- Source-owned replay completed: `refresh_advanced`, trunk
  `8cf4aaaa190e6a11dff2661aa6806ce476128653`, new branch OID
  `83640d191f78f0cd685e5f709027819799efe1cf`, which descends from
  `origin/main` (verified with `git merge-base --is-ancestor`).
- Both replayed checkpoints signed (`git verify-commit` exit 0, ECDSA
  `SHA256:V6JiKG7J29mjsvikcLoSVp0bLa77VTsFy12gnLO81cM`).
- Single LOGBOOK.md conflict resolved by packet
  `.temp/TASK-260830-21gygk/replay-resolutions-7208cc7.json`
  (checkpoint `7208cc7...`, sha256
  `4f2dcf42…b84b7f3c`): both additive blocks preserved newest-first
  (2026-09-09 adoption entries, then 2026-09-08 1r9wrr entry plus its two
  2026-09-07 entries). Retained replay worktree never edited; no manual
  commit, reset, or registry edit.
- Candidate left UNCOMMITTED: tracked delta LOGBOOK.md, README.md,
  `internal/sessrepo/store.go`, `internal/sessrepo/sessrepo_test.go`;
  untracked `internal/sessquery/` (shared API + tests + TRACEABILITY +
  mutate runner). Predecessor checkpoints untouched.
- Live-candidate logs for this run: `.temp/TASK-260830-21gygk/` in the
  Story worktree (replay packet, mutant battery `mutants-bd27d9/`).

## Acceptance: 8 of 8 AC rows driven at the shared library boundary

Production call sites and named tests (full matrix in
`internal/sessquery/TRACEABILITY.md`):

1. Bare UUID/name selectors — `Reader.Resolve → resolveBare`
   (`TestResolvePinnedPrecedence`, `TestResolveBareIdentityUnion`,
   `TestResolveAllowlistAndReadFailures`; retained repository entry
   `Repository.Resolve` pinned by `TestResolveLocalNameAndUUID`).
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
4. List summaries — `Reader.List → Repository.ListSessions →
   Projector.Project` (`TestListStatusDerivedFactsAndStableOrder`,
   `TestListStatusCheckpointAndProjectionFailure`,
   `TestParkedReadRecoveryAndMissingVsMalformed`). Whole-read refusal on
   projection failure; no silent omit.
5. Status summaries — `Reader.Status → Resolve → Projector.Project`,
   `Reader.InspectLocal` (`TestCreatingSummaryCannotClaimClosedCLIResult`
   plus list tests and qualified status rows). Record-only chains keep
   the accepted internal projection with empty owner/lease; per the
   pinned prefix clarification this is an interrupted persistence prefix
   for the §14.7.4 recovery leaf, not a public creating state. No
   nullable/placeholder owner or lease field added.
6. Stable deterministic sorting — `Reader.read`, `peerCandidates`,
   `selectIdentity` (order tests above plus
   `TestResolveBareIdentityUnion`).
7. SelectionPlan build/revalidate — `Reader.BuildPlan → bindPlan`,
   `Reader.Revalidate`, `ParsePlan`, `BoundariesFor`
   (`TestRevalidateDetectsEachFactChange` per-member arms and 7 further
   plan tests).
8. Negative/refusal cases — every gate below, each with a narrowing
   mutant killed by a named behavioral test.

0 of 8 rows delivered as a public CLI surface: stated bound, not a gap
in assigned shared behavior. No unsupported capability advertised.

## Validation re-derived by this run on the final candidate (real exits)

| Command | Exit |
| --- | --- |
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `gofmt -l internal/sessquery/ internal/sessrepo/` (empty) | 0 |
| `git diff --check` | 0 |
| `go test ./... -count=1` (all packages ok) | 0 |
| `go test ./... -cover` (sessquery 91.0%, sessrepo 87.6%, sessstate 92.2%) | 0 |
| `go test ./internal/sessquery/ ./internal/sessrepo/ ./internal/sessstate/ -race -count=1` | 0 |
| `python3 internal/sessquery/testdata/mutate.py .temp/TASK-260830-21gygk/mutants-bd27d9` | 0 |

Mutant battery (isolated source copy; managed checkout untouched;
machine table `mutants.json` + per-mutant logs in `mutants-bd27d9/`):
29 of 29 behavioral mutants KILLED, every kill naming failing
behavioral tests (26 narrowing N-*, 3 ordering B-*), including
token-preserving grammar mutants (`N-first-at-split`, `N-local-prefix`,
`N-id-case`, `N-peer-alias-fold`) killed through the behavioral suites.
`control-before`/`control-after` exit 0; `C-not-applied` NOT_APPLIED and
`C-compile-failure` COMPILE_OR_HARNESS_FAILURE correctly not kills.

## Stated bounds (declared, not implemented)

- `lease_record_id` unbound: the stack carries no Lease Record objects,
  so the plan binds the validated winning envelope triple
  (epoch/lease-ID/owner); documented in `plan.go` and TRACEABILITY.
- `authority_heads` binds validated event-head digests of the plan
  source only; union precedence facts at build time are not re-checked.
- Wire exit codes, the §14.7.3 public-document list refusal, §14.7.4
  recovery, effect authorization, and composed invocation-to-plan
  bindings are caller-leaf owned; `BoundariesFor` names every boundary
  callers must revalidate at. Plans authorize read projection only.
- Reads across calls are not an atomic snapshot; transports, peer
  authentication, freshness, host metadata, and observation contracts
  remain upstream caller inputs.

## Anomaly recorded: stray prior-run artifact relocated (bytes preserved)

The first full-suite run in this session failed in exactly two packages
(`canonicaljson` census, `specdoc` embed-reach) naming only
`--help/mutation-source/...` packages: a prior run had left an
untracked 11 MB `--help/` directory (471 files: mutant logs plus a full
`mutation-source` tree copy) at the worktree root, inside the Go module
package pattern. Both packages passed after relocation (18.5s / 0.3s),
and the final full suite is green, proving the failures were
environmental, not product. The directory was moved byte-intact (not
deleted) to gitignored scratch
`.temp/TASK-260830-21gygk/stray/preserved-prior-run-help-artifact/`;
restore with `mv .temp/TASK-260830-21gygk/stray/preserved-prior-run-help-artifact ./--help`.
No board evidence was affected (durable evidence lives in board
resources). No product file was added, removed, or edited for this.
