# TASK-260830-21gygk results (RUN-260909-ffb76b, producer recovery + handoff)

Scope: v0.6.0 selector contract (§§14.7/14.7.1, shared SelectionPlan
construction/revalidation §14.7.2, refining §2.3, authoritative summaries
§§5.7/14.7.3). Authority: spec v0.6.0 commit
`0cbdf100dbf84df50c64f792b1f940e3a67859a6` (STORY-260908-18woqo landed).
Branch `task-board/story/STORY-260830-3tq4ns` at
`83640d191f78f0cd685e5f709027819799efe1cf`, descending from
`origin/main` `8cf4aaaa190e6a11dff2661aa6806ce476128653` (re-verified
with `git merge-base --is-ancestor`). Candidate left UNCOMMITTED.
No `ax` session surface exists in this tree; CLI delivery, wire exits,
lifecycle effects, and the §14.7.4 recovery flow are caller/recovery-leaf
owned. No product-code change was made in this run; see §1.

## 1. Publication recovery record (this run)

- `task-board worktree refresh-candidate TASK-260830-21gygk` returned
  `refresh_already_current` (TrunkOID = ReviewedTrunkOID = `8cf4aaa…`,
  BranchOID = `83640d1…`). The installed source fix restored the
  previously missing immutable resource files: before this run the
  worktree missed ~100 tracked `.task-board` files; after, 19
  deletions + 3 modifications remain, all non-JSON board-checkout
  artifacts (activity streams, progress/README markdown), none of them
  product files and none covered by a validation gate.
- Exact failed gate re-run on the final candidate:
  `git ls-files -z '*.json' | xargs -0 -n1 python3 -c
  'import json,sys;json.load(open(sys.argv[1]))'` → exit 0
  (122 tracked JSON files, all parse).
- Product bytes untouched: zero product edits in this run. The working
  candidate is exactly the RUN-260909-bd27d9 candidate: tracked delta
  `LOGBOOK.md`, `README.md`, `internal/sessrepo/store.go`,
  `internal/sessrepo/sessrepo_test.go`; untracked `internal/sessquery/`
  (13 files). sha256 (abbrev): LOGBOOK `da0823d6`, README `86af5b32`,
  store.go `28d64e24`, sessrepo_test.go `a5618be2`, query.go `bde3c9c8`,
  selector.go `6769455f`, plan.go `eaa82f3a`, revalidate.go `818df82e`,
  digest.go `b6cc6a27`, TRACEABILITY.md `0fde2779`.
- No stray `--help/` tree at the worktree root (verified absent); the
  preserved prior-run artifact stays under gitignored
  `.temp/TASK-260830-21gygk/stray/` and is not in the candidate diff.

## 2. Validation re-derived by this run on the final candidate (real exits)

| Command | Exit |
| --- | --- |
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `gofmt -l internal/sessquery/ internal/sessrepo/` (empty) | 0 |
| `git diff --check -- internal/ LOGBOOK.md README.md` | 0 |
| `go test ./... -count=1` (all 26 packages ok) | 0 |
| `go test ./internal/sessquery/ ./internal/sessrepo/ ./internal/sessstate/ -cover` (91.0% / 87.6% / 92.2%) | 0 |
| `go test ./internal/sessquery/ ./internal/sessrepo/ ./internal/sessstate/ -race -count=1` | 0 |
| tracked-JSON gate above | 0 |

Raw logs: `TASK-260830-21gygk_validation-ffb76b.log` (attached).

## 3. Mutation evidence (reused exact-candidate battery, machine manifest attached)

No product byte changed since RUN-260909-bd27d9, so the battery was
not repeated merely because a new run began. Machine manifest
`TASK-260830-21gygk_mutants-bd27d9.json` (attached; 14 KiB) records,
per mutant, the executed behavioral command
(`go test ./internal/sessquery ./internal/sessrepo -count=1 -v`) and
the killing test names: 29 KILLED (26 narrowing `N-*`, 3 ordering
`B-*`), 2 CONTROL green (`control-before`, `control-after`),
`C-not-applied` NOT_APPLIED and `C-compile-failure`
COMPILE_OR_HARNESS_FAILURE — correctly distinguished, never counted
as kills. Token-preserving grammar mutants (`N-first-at-split`,
`N-local-prefix`, `N-id-case`, `N-peer-alias-fold`) were killed
through the behavioral suites. Per-mutant logs
(`mutants-bd27d9/*.log`, 11 MB) remain in worktree scratch
`.temp/TASK-260830-21gygk/mutants-bd27d9/` and are cited, not
re-attached.

## 4. Acceptance: 8 of 8 AC rows driven at the shared library boundary

Unchanged code; matrix per `internal/sessquery/TRACEABILITY.md`:

1. Bare UUID/name selectors — `Reader.Resolve → resolveBare`
   (`TestResolvePinnedPrecedence`, `TestResolveBareIdentityUnion`,
   `TestResolveAllowlistAndReadFailures`; repository entry
   `Repository.Resolve` pinned by `TestResolveLocalNameAndUUID`).
2. Qualified selectors — `Reader.Resolve → resolveExplicit →
   locateSource` (`TestResolveQualifiedSourcesSelectOneIndex`,
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
   `TestParkedReadRecoveryAndMissingVsMalformed`). Whole-read refusal
   on projection failure; no silent omit.
5. Status summaries — `Reader.Status → Resolve → Projector.Project`,
   `Reader.InspectLocal` (`TestCreatingSummaryCannotClaimClosedCLIResult`
   plus list tests and qualified status rows). Record-only chains keep
   the accepted internal projection with empty owner/lease: per the
   pinned prefix clarification this is an interrupted persistence
   prefix for the §14.7.4 recovery leaf, not a public creating state
   (SPEC §5.7 requires Session Record AND initial lease for creating;
   §13.1 step 2 persists the initial Lease Record before process
   creation). No nullable/placeholder owner or lease field added.
6. Stable deterministic sorting — `Reader.read`, `peerCandidates`,
   `selectIdentity` (order tests above plus
   `TestResolveBareIdentityUnion`).
7. SelectionPlan build/revalidate — `Reader.BuildPlan → bindPlan`,
   `Reader.Revalidate`, `ParsePlan`, `BoundariesFor`
   (`TestRevalidateDetectsEachFactChange` per-member arms and 7
   further plan tests).
8. Negative/refusal cases — every gate above, each with a narrowing
   mutant killed by a named behavioral test (§3).

0 of 8 rows delivered as a public CLI surface: stated bound, not a gap
in assigned shared behavior. No unsupported capability advertised.

## 5. Reviewer-addendum evidence (observed this run, scratch probe deleted after)

A temporary in-package probe (created, run, deleted in this run; not
in the candidate) recorded actual production behavior; full log
attached as `TASK-260830-21gygk_revalidation-probe-ffb76b.log`:

- P1: bare-NAME plan won at a peer tier (source hostA, session …90ac).
  After local gained the same live name, fresh `Resolve("beta")`
  moved to the local copy (…90ad, peer `""`) — the resolution-drift
  premise holds — while `Revalidate(plan)` returned nil.
- P2: bare-`id:` plan on one copy; after another allowed source gained
  a disagreeing record copy, fresh `Resolve` refused
  `integrity_failure: … disagreeing record digests across sources`,
  while `Revalidate(plan)` returned nil.
- P3 control: appending an event to the plan-source chain made
  `Revalidate` return `selector_plan_stale` — revalidation watches
  the plan source, so the P1/P2 nils are scope, not blindness.

Normative comparison (SPEC.v0.6.0.md): the plan binds one provenance
source (`source_host_id`: "for a bare union match, provenance host of
the selected entry", l11948) and one complete validated read of that
source (`source_index_digest`: "Digest of the complete validated
index read used for selection", l11950; completeness contrasts with
partial/cached reads per §14.7.1, no union-digest construction is
defined anywhere). Revalidation compares every bound fact; the
invalidation list (l11967–11971) names bound-member changes only, and
"An unrelated index change may conservatively require a fresh plan"
(l11973–11974) is permissive. Detecting name-tier drift inside
`Revalidate` would be re-resolving by name, which the spec forbids
(l11972); the plan pins the UUID, and invocation drift is owned by
the caller-side composed binding (`SEL-CASE-…-session/source-mismatch`
witnesses: "Plan/current agreement alone authorizes nothing",
l12122–12125; receiver validates "against its independently read
exact session, winning lease, complete authority", l12049–12052).
All revalidation members stay `endpoint-local … never cross-compared`
(l12127–12128, l12143–12152). No corrective product change follows
from this comparison; the ruling is the reviewer's.

- `lease_record_id`: reported as a declared delta, not equivalence.
  The 16-member table requires it (l11952); this stack contains no
  Lease Record objects, so there is no validated digest to bind and
  the wire format carries no such field (a future member without a
  disposition row fails the publication gate closed, l12130–12133).
  The winning lease triple (epoch/lease-ID/owner) binds from the
  validated envelope derivation, satisfying its own row (l11953).
  CLI 5 attach `lease_record_id` admission is caller/receiver-leaf
  owned. Fabricating a digest would be an unsupported claim.
- §14.7.3: shared-library scope delivered is authoritative
  read-model summaries with derived facts only and whole-read refusal
  (row 4/5); wire CLI Result 5 / public-document refusal is
  caller-leaf owned.

## 6. Stated bounds (unchanged)

`authority_heads` binds validated event-head digests of the plan
source only; union precedence facts are not re-checked (§5 evidence).
Wire exit codes, §14.7.3 public-document list refusal, §14.7.4
recovery, effect authorization, and composed invocation-to-plan
bindings are caller-leaf owned; `BoundariesFor` names every boundary
callers must revalidate at. Plans authorize read projection only.
Reads across calls are not an atomic snapshot; transports, peer
authentication, freshness, host metadata, and observation contracts
remain upstream caller inputs.

Ready for review (producer handoff; independent Astra review and
orchestrator integration own the remaining path).
