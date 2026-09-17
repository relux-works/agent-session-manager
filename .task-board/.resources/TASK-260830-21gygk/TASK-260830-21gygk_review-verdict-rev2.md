# TASK-260830-21gygk — revision 2 review: changes requested

Reviewed immutable CR-TASK-260830-21gygk-2 revision 2, candidate tree
`fbff1e0a9f443b1378f04dd108a93c07449ba551`, base
`8cf4aaaa190e6a11dff2661aa6806ce476128653`, Story checkpoint
`83640d191f78f0cd685e5f709027819799efe1cf`.
Authority is the candidate's pinned SPEC.v0.6.0.md, spec commit: `0cbdf100dbf84df50c64f792b1f940e3a67859a6`.
Reviewed assigned shared-library scope, not future CLI invocation ownership.

## Blocking findings

1. **P1 — Revalidation ignores conflicting current authority outside the selected source.**
   `internal/sessquery/revalidate.go:62-123` reads only the plan-source repository;
   `plan.go:229-286` binds only its event tail. SPEC 14.7.2 lines 11965-11974
   requires the same immutable record, current tombstones, complete authority
   union and winning lease to be validated before proceeding. In
   `TestReviewerOtherSourceDivergenceMustRefuse`, BuildPlan selects a peer's
   UUID, then another allowed source (local) gains a different immutable
   record for that same UUID. Fresh Resolve refuses integrity_failure;
   Revalidate(oldPlan) incorrectly returns nil. The probe fails, exit 1.
   This is an exact-UUID authority check, not forbidden name re-resolution.
   Bind and revalidate the required complete authority, with class-preserving
   read/integrity refusals and changed-head/winner tests across sources.
   Keep explicit-source selection/no-fallback semantics intact.
   P1 in the producer's diagnostic (a different UUID gains the old name
   locally) alone does NOT prove a defect: no silent retargeting is allowed,
   and unrelated index invalidation is permissive. I do not require rerunning
   name resolution or treating every such gain as stale. Producer P2 does
   establish a different issue: contradictory authority for the pinned UUID.

2. **P1 — SelectionPlan does not implement the required plan members and lease authority.**
   `internal/sessquery/plan.go:122-165,249-266` and
   `revalidate.go:204-218,245-264` omit lease_record_id, permit empty source host,
   and accept the all-empty/zero lease tuple. SPEC 14.7.2 lines 11939-11959
   requires the exact 16-member plan, a source UUIDv7, a validated winning Lease
   Record digest, and positive epoch/UUID lease-owner facts from that same record.
   `TestReviewerPlanRequiresLeaseRecord` drives BuildPlan and Marshal and fails
   because lease_record_id is absent. Committed candidate tests
   TestBuildPlanBindsCurrentFacts and TestBuildPlanRecordOnlySessionBindsEmptyLease
   explicitly protect empty host/lease behavior. A missing Lease Record owner
   is a dependency gap, not equivalent authority or a scope exemption. Connect
   the shared builder/revalidator to validated lease facts; refuse unavailable
   required facts rather than returning a conforming-looking partial plan.
   Do not fabricate a digest. Action-specific invocation binding remains caller
   owned, but the required shared plan field does not.

3. **P1 — Assigned authoritative summary refusal is missing.**
   `internal/sessquery/query.go:94-155` exposes only an internal projection.
   `Reader.List` and `Reader.Status` succeed for record-only bootstrap with
   epoch 0/empty owner. Both subtests of TestReviewerBootstrapSummaryMustRefuse
   fail through these production entries. SPEC 14.7.3 lines 12354-12372 requires
   selector_bootstrap_incomplete for a complete record-only authority read,
   selector_observation_unavailable for unavailable required observations/host
   metadata, and whole-list refusal for an unrepresentable record. No shared
   summary API implements those classifications or accepts the needed validated
   observation facts. Preserving the accepted low-level projector is fine;
   it does not deliver the assigned authoritative summary layer. Add that layer
   and named negative tests, including a mixed healthy/incomplete list and
   unavailable host/observation facts. CLI envelope/exit rendering and the
   durable bootstrap recovery operation remain their existing caller owners.
   The current cliresult shape-rejection test does not prove these refusals.

4. **P2 — Narrowing evidence and completion claims overstate coverage.**
   `internal/sessquery/testdata/mutate.py:81-87` uses `&& false` for revocation,
   configuration, source binding, index, record, lease and heads comparisons.
   These are full clause disables, even though named N-*; preserving the text
   does not preserve the gate. Several only change a later refusal's reason.
   Cross-copy agreement is explicitly listed without a weakening in
   TRACEABILITY.md. Replace/supplement these with genuine single-class-member
   admission mutants and named behavioral failures, covering each required
   gate including the fixes above. Correct TRACEABILITY.md's “8 of 8 ... in
   full”, “every bound fact”, and “each killed by narrowing” claims and the
   corresponding README/outcome claims. Existing byte-equal mutant sources and
   raw logs support 29 behavioral KILLED results, not 26 qualifying narrowing
   proofs. Controls are correctly separate: CONTROL x2, NOT_APPLIED x1,
   COMPILE_OR_HARNESS_FAILURE x1. None is a SURVIVED measurement.
   I independently used the same run()/classifier with a harmless comment
   plant: SURVIVED exit 0. A true one-session index-gate narrowing was KILLED
   exit 1 by TestRevalidateDetectsEachFactChange/unrelated_session_change_changes_index.
   Thus the classifier works; the shipped battery's evidence coverage needs repair.

## AC coverage ratio and bounds

**8 of 8 producer AC rows are exercised at some shared-library production entry;
4 of 8 are established within the supplied-repository test boundary; 4 of 8
remain partial or failing.** Exercised is not implemented in full. This is the
producer's eight-row breakdown of the assigned AC, not a claim to enumerate
all normative fixture rows. **0 of 8 public CLI rows delivered** is an explicit
caller integration bound and is not itself a rejection reason.

| Row | Production entry / named candidate tests | Review result |
| --- | --- | --- |
| UUID/name | Reader.Resolve; TestResolvePinnedPrecedence, TestResolveBareIdentityUnion | Established for supplied repository/config inputs |
| Qualified | Reader.Resolve; TestResolveQualifiedSourcesSelectOneIndex, TestResolveQualifiedSourceNeverFallsBack, TestResolveExplicitSourceReadFailures | Established for supplied inputs; does not attest live peer transport/config acquisition |
| Ambiguity | Reader.Resolve → matchName/checkRecordAgreement; TestResolveExactNamesAndASCIICollisions, TestResolveBareIdentityUnion | Resolution tests established; current-plan authority remains finding 1 |
| List summaries | Reader.List; TestListStatusDerivedFactsAndStableOrder, TestListStatusCheckpointAndProjectionFailure | Partial: finding 3 |
| Status summaries | Reader.Status/InspectLocal; TestCreatingSummaryCannotClaimClosedCLIResult and qualified status tests | Partial: finding 3 |
| Deterministic order | Reader.List/Resolve; TestResolvePeerOrderAndReplicatedIdentity, TestResolveBareIdentityUnion | Established within supplied-input boundary |
| Plan | Reader.BuildPlan/Revalidate/ParsePlan; TestBuildPlanBindsCurrentFacts, TestRevalidateDetectsEachFactChange, TestPlanDigestAndMarshalRoundTrip | Partial: findings 1–2 |
| Negative/refusal | Reader.Resolve/List/Status/BuildPlan/Revalidate; named refusal tests and mutation battery | Partial: findings 1–4 |

Freshness/transport authentication are not proven by local learned repository
handles, as SPEC 14.7.1 expressly says. Current tests must not be cited as such
proof. Shared APIs should expose the required validated facts/refusals without
requiring callers to implement another resolver.

## Verification and preservation

- Exported the exact immutable tree to task scratch with git archive. All 38
  changed paths matched both archive and live worktree bytes after review;
  candidate-manifest.json records SHA-256 values. No product or branch edit.
- Independently ran `go test ./internal/sessquery ./internal/sessrepo ./internal/sessstate -count=1 -v`
  on that archive before probes: exit 0. Raw review-tests.log attached in bundle.
- Added reviewer-only regression tests to the scratch archive, ran
  `go test ./internal/sessquery -run '^TestReviewer' -count=1 -v`: exit 1,
  three failed top-level tests (four failed assertions/subcases). Saved their
  source and log, then removed them from the archive; no candidate test changed.
- Independent classifier/narrowing check: driver exit 0, SURVIVED plant exit 0,
  KILLED narrowing exit 1, full named behavioral tests recorded. This is a
  two-plant check, not a rerun of the producer's whole battery.
- Reused attached ffb76b evidence for build, vet, formatting, full suite,
  coverage (91.0% query / 87.6% repo / 92.2% state), race and JSON validation;
  these report exit 0. I did not rerun those full gates. Read the actual rev2
  CR validation resource: recorded command exit markers are 0; board validation
  warning counts are not product acceptance. No failing reviewer probe is
  converted into green by that earlier log.
- Retained predecessor provhost/sessstate bytes match checkpoint exactly.
  Only predecessor repo code change is exact-case name matching at store.go:135
  plus its test, consistent with the normative exact-name rule. Narrow tests
  passed. No reason to reopen settled crash/reducer/canonical admission reviews.
- LOGBOOK replay preserves predecessor and new-leaf entries; no stray --help
  tree or board checkout path is in the 38-path CR delta. The adopted spec pin
  and catalog remain at the supplied base, not replaced with historical v0.5.
- Failed skill-path probe: this Story worktree has no .codex/skills or
  .agents/bin. Read the existing Curator-managed go-testing-tools skill from
  the main checkout; no installation or shared-infra modification performed.

## Disposition

Changes requested; route TASK-260830-21gygk (implement-name-resolution-and-summary)
to **to-dev**, not blocked. This is implementable rework/dependency integration,
not a new product decision. Preserve full assigned scope. No accept_cr,
commit, checkpoint, branch switch, or landing performed. The reviewer logbook
entry is this durable task outcome; product LOGBOOK is left read-only.
