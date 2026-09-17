# TASK-260830-21gygk — Profile-pair relation census (rev11)

Rev11 change (CR10 P1): the R5 resume row now admits its REFERENCED
Checkpoint Record through the shared semantic owner
(`checkReferencedCheckpointBinding` in `internal/sessquery/lease.go`)
before deriving from its closure. The new record-consumption table
below is the class-closing instrument for the fifth C6 member:
every record the shared owner consumes names its admission owner,
call site, positive control, negatives, and narrowing plant. The
six R1-R6 rows are retained with R5 extended; 6 of 6 rows remain
DRIVEN, 0 undriven.

Authority: AX v0.6.0 (`0cbdf100dbf84df50c64f792b1f940e3a67859a6`),
pinned sentences:

- SPEC 5.2: "The profile and profile-source pair in every launch,
  resume, and fork event MUST equal the Section 2.4 effective profile
  at the referenced checkpoint or, for the first launch, the Session
  Record and null; a non-null source MUST name the newest
  authoritative `profile.changed` event in that event-head closure;
  for `fork.created` the effective new-session authority is its newly
  persisted Session Record so `profile_source_event_id` is null;
  `source_profile_event_id` never participates" in new-session
  derivation.
- SPEC 2.4: effective profile is the Session Record value followed by
  the newest authoritative `profile.changed` event in lease/sequence
  order; a checkpoint's event-head closure fixes both values; fork
  creates a new authority boundary (new Session Record stores the
  source-checkpoint effective profile, new-session source is null,
  `source_profile_event_id` is provenance only).
- SPEC 5.4: the transitive event-head closure fixes the effective
  profile/source for its checkpoint; missing, losing-lease, or
  non-newest references are `integrity_failure`; consumers MUST NOT
  consult a later local-only event or fall back to creation.

One row per (event kind x relationship). "Entries" = the four shared
production entries `BuildPlan`, `Revalidate` (fresh + old-plan),
`AuthoritativeStatus`, `AuthoritativeList`, all through
`winningLeaseFor` -> `checkWinnerCheckpoint`/`checkCheckpointBinding`
-> `checkCheckpointProfileAuthority` in
`internal/sessquery/lease.go`. Every row is DRIVEN (named committed
tests through the entries); nothing below is an undriven bound.
Adjacent caller bounds are named with owning clauses at the end.

## Census rows

### R1 — provider.launched, first launch

- Authority input: Session Record creation profile + null source. No
  checkpoint input (5.2 "for the first launch, the Session Record and
  null"; 13.1 step 5). Read via `repo.GetRecord` ->
  `sessionCreationProfile`.
- Production owner arm: `checkClosureProfilePairs`
  provider/task_board.launched case, `firstLaunch` branch; presence
  and value bound by `checkProfilePairSource`.
- Positive control: `TestRev8CheckpointProfileAuthority`
  (missing_source=false, direct/task_board x winner/ancestor);
  `TestRev10ForkLocalProfile` good (launch after fork is still the
  first launch); `TestRev10ResumeReferencedCheckpoint` good.
- Negatives:
  - contradictory value: `TestRev8ProfileSourceRefusals/first_wrong_profile`
    (integrity).
  - contradictory presence: `TestRev8CheckpointProfileAuthority`
    missing_source=true, all-zero source (integrity).
  - missing Session Record bytes (`GetRecord` failure): unavailable
    with cause (shared helper; defensive, see read-failed bound).
  - undecodable Session Record: integrity via `sessionCreationProfile`.
  - non-newest / losing-lease / wrong-session / later-local-only:
    not applicable — a first launch cites no change event by
    construction; any source at all is already contradictory.
  - read-failed mid-derivation: defensive bound (shared helpers wrap
    the cause in unavailable; unreachable without a storage race
    because the repository re-verifies fully per load and parks
    corrupt sessions before the gate runs).
- Narrowing plants: `N-profile-first-source` (admits exactly the
  all-zero source where null is wanted; killed by the rev8
  missing-source negatives by admission);
  `N-profile-first-value` (admits exactly a standard-valued
  sourceless pair where (yolo, null) is wanted; killed by
  first_wrong_profile and task_board bad_first_profile by admission).

### R2 — provider.launched, later launch

- Authority input: newest authoritative `profile.changed` at or
  before the event in the admitting closure with its target profile;
  creation pair while none exists. Launch payloads carry no
  `checkpoint_id` member (5.2 payload table), so the admitting
  closure evaluated in 2.4 lease/sequence order is the only
  referenced authority a launch can name; a launch cannot cite a
  change after it (causal necessity, retained rev8 positive
  controls).
- Production owner arm: same launch case, `!firstLaunch && hasNewest`
  branch; source resolved by `checkProfilePairSource` against the
  admitting closure.
- Positive control: `TestRev8ProfileChangedPositiveControl`,
  `TestRev8ProfileTwoGenerationControl` (direct/task_board,
  winner/ancestor, fresh-plan revalidation); rev10 ancestor launch2
  positives in the resume and task-board fixtures.
- Negatives (all integrity, all four entries):
  - wrong-type source: `TestRev8ProfileSourceRefusals/wrong_type`.
  - non-newest source: `/non_newest`.
  - stale value: `/stale_value`.
  - cross-session source: `/cross_session`.
  - losing-lease source (preserved divergent blob outside the
    index): `/losing_branch`.
  - dangling closure predecessor: `/dangling_predecessor`.
  - missing source digest (unknown to the index): same unknown-source
    arm (dangling_resume_source covers the shared helper for resumes;
    launches share `checkProfilePairSource`).
  - later-local-only: `TestRev8ProfileHistoricalClosureAdmits`
    (post-head change never consulted; admission control). A launch
    citing a well-formed post-head digest has no fixture by causal
    necessity (chain continuity forces every prior event into the
    closure); stated bound with reason, retained from rev8.
  - read-failed: defensive bound as in R1.
- Narrowing plants: `N-profile-newest` (first change wins; killed by
  non_newest + two-generation control by admission);
  `N-profile-value` (admits exactly a yolo-valued mismatch; killed by
  stale_value by admission); `N-profile-change-direction`
  (token-preserving: still matches `profile.changed` and reads `to`
  but derives from `from`; killed by the P1/E1 positive controls).

### R3 — task_board.launched, first launch

- Authority input: Session Record creation profile + null source
  (5.2; 13.2 step 3). Same read path as R1.
- Production owner arm: shared launch case, `firstLaunch` branch
  (same arm as R1; the event-type case covers both launch types).
- Positive control:
  `TestRev10TaskBoardLaunchProfileAuthority/good_first` and
  `/good_historical` (first launch yolo/null, winner/ancestor).
- Negatives (all integrity, all four entries):
  - contradictory value: `/bad_first_profile`.
  - contradictory presence: `/bad_first_source`.
  - Session Record missing/undecodable: shared path with R1.
  - non-newest / losing-lease / wrong-session / later-local-only:
    not applicable (same reason as R1).
  - read-failed: defensive bound as in R1.
- Narrowing plants: shared-arm `N-profile-first-source` (presence)
  and `N-profile-first-value` (value; killed by bad_first_profile by
  admission). No separate task-board-only plant: the arm is shared
  and the plants kill through task-board fixtures in the same suite.

### R4 — task_board.launched, later launch

- Authority input: same prefix rule as R2 (no `checkpoint_id`
  member; admitting closure in lease/sequence order).
- Production owner arm: shared launch case, later branch.
- Positive control: good_first/good_historical second and third
  launches cite the newest change (winner/ancestor); good_historical
  is the later-local-only control (post-head change never consulted,
  winner path; ancestor path cites it once inside the ancestor
  closure).
- Negatives (all integrity, all four entries):
  - stale value: `/bad_later_stale`.
  - non-newest: `/bad_later_non_newest`.
  - wrong-type / cross-session / losing-lease / dangling: shared
    `checkProfilePairSource`/closure helpers with R2 (driven there);
    the task-board rows drive value + newest, the two members the
    later-launch derivation adds over the shared resolution.
  - later-local-only citation: no fixture by causal necessity (same
    bound as R2).
  - read-failed: defensive bound as in R1.
- Narrowing plants: shared-arm `N-profile-newest` (killed through
  bad_later_non_newest by admission alongside the rev8 non_newest),
  `N-profile-value`, `N-profile-change-direction`.

### R5 — session.resumed

- Authority input: effective pair of the event's OWN referenced
  checkpoint (SPEC 5.2; 13.10 "derive the exact effective
  profile/source from its event-head closure"; 2.4
  PROFILE-*-RESUME fixtures): `checkpoint_id` member ->
  semantically admitted Checkpoint Record (`Reader.CheckpointRecords`
  through `checkReferencedCheckpointBinding`: owning lease tuple in
  the winning ancestry, creator-holder, Session.kind variant, head
  closure via the shared helpers) -> transitive predecessor closure
  over the winning-source chain index -> newest `profile.changed`
  with its target, or the creation pair while the referenced closure
  holds no change. Never the outer-walk prefix, never a canonically
  parsed but unadmitted record.
- Production owner arm: `checkClosureProfilePairs` session.resumed
  case + `referencedCheckpointProfile` + `checkReferencedCheckpointBinding`
  (`lease.go`); source resolved by `checkProfilePairSource` against
  the REFERENCED closure. The winning ancestry chain and Session.kind
  are threaded from `winningLeaseFor` through `checkWinnerCheckpoint`/
  `checkCheckpointBinding` -> `checkCheckpointProfileAuthority` ->
  `checkClosureProfilePairs`; no new store, transport, publication,
  or materialization.
- Positive control: ported `TestReview9ResumeReferencedCheckpoint`
  good; `TestRev10ResumeReferencedCheckpoint` good (E1 closure),
  good_prechange (pre-change closure cites yolo/null while E1 sits
  later in the outer closure — the historical control),
  good_divergent (preserved losing-lease change never consulted);
  updated `TestRev8ProfileResumePositiveControl` (both checkpoints
  supplied); `TestReview10ReferencedCheckpointAdmission` good
  (direct/task_board x winner/ancestor, 4 controls); fresh-plan
  Revalidate in every good mode. Historical, branching, and lagging
  references stay admitted because their owning leases, creators,
  variants, and at-or-before heads all pass the shared owner.
- Negatives:
  - contradictory pair vs referenced closure (later-local-only
    citation): bad_stale (integrity, all four entries) — the P1-B
    probe negative across direct/task_board x winner/ancestor.
  - non-newest within referenced closure: bad_non_newest
    (integrity).
  - losing-lease source: bad_losing (integrity).
  - creation fallback past a referenced change: bad_creation_fallback
    (integrity).
  - dangling source with valid reference: rev8
    `dangling_resume_source` (integrity; reference repaired to a
    real checkpoint so the test isolates the source with a valid
    reference).
  - missing referenced record: bad_missing, bad_missing_stale
    (observation_unavailable, all four entries).
  - wrong-session reference: bad_wrong_session (unavailable).
  - unresolvable referenced head: bad_unknown_head (unavailable).
  - wrong-creator referenced record (creator host B while its owning
    lease holder is A): wrong_creator (unavailable, all four entries
    + old-plan, direct/task_board x winner/ancestor) — the CR10 P1
    probe negative.
  - wrong-variant referenced record (task-board persistence under a
    direct Session Record, and the reverse): wrong_variant
    (unavailable, all four entries + old-plan) — the CR10 P1 probe
    negative.
  - wrong-lease referenced record (fencing token without an admitted
    owning lease at that epoch): wrong_lease (unavailable, all four
    entries + old-plan) — the CR10 P1 probe negative.
  - old-plan substitution covering a stale resume pair:
    `TestRev10ResumeOldPlanSubstitution` (integrity); referenced
    withdrawal: `TestRev10ResumeReferencedWithdrawal` (unavailable);
    referenced-admission old-plan: `TestRev11ReferencedCheckpointAdmissionOldPlan`
    (unavailable, 12 negatives).
  - read-failed mid-derivation: defensive bound as in R1 (same
    helpers; missing vs read-failed stay distinct errors).
- Narrowing plants: `N-profile-resume-missing` (admits exactly a
  missing-reference resume carrying the creation pair; killed by
  bad_missing by admission while bad_missing_stale still refuses);
  `N-profile-resume-newest` (first referenced change wins; killed by
  bad_non_newest by admission); `N-referenced-creator` (admits exactly
  a hostB-created referenced record; killed by the wrong_creator
  negatives by admission while the census gate passes);
  `N-referenced-variant` (admits exactly the idA wrong variant on the
  referenced path; killed by the wrong_variant negatives by admission
  while the census gate passes); `N-referenced-lease` (admits exactly
  the leaseCycleB fencing token past the tuple match; killed by the
  wrong_lease negatives by admission while the census gate passes).
  All five kills are behavioral admission failures (see per-plant raw
  logs with subprocess exits), not class-label failures. The three
  N-referenced plants preserve every census call token, so the static
  `TestRev11RecordConsumptionCensus` gate passes and only the
  behavioral suites fail.

### R6 — fork.created

- Authority input: newly persisted Session Record creation profile
  + null source (5.2 fork sentence; 2.4 fork boundary; 13.8 step 8;
  2.4 PROFILE-*-FORK fixtures). The fork event lives in the new
  session chain, so the admitting Session Record IS the new record;
  no cross-session input is needed.
- Production owner arm: `checkClosureProfilePairs` fork.created
  case; `source_profile_event_id` is never read.
- Positive control: ported `TestReview9ForkLocalProfile` good
  (direct/task_board); `TestRev10ForkLocalProfile` good and
  good_provenance (real `source_profile_event_id` digest plus a
  preserved divergent blob: admitted, proving non-participation);
  old-plan narrow baselines.
- Negatives (all integrity, all four entries + old-plan):
  - contradictory value: bad_profile (ported probe + rev10 matrix
    direct/task_board x winner/ancestor +
    `TestRev10ForkOldPlanSubstitution` with the integrity class
    pinned).
  - non-null source: NO chained fixture — the canonical shape owner
    pins fork `profile_source_event_id` to null at append time
    (observed: "member profile_source_event_id must be null"), so
    the shared presence check is unreachable for forks and stays
    covered through launches. Stated bound with owning package
    named (canonicaljson/session-event), not an undriven row.
  - Session Record missing/undecodable: shared path with R1.
  - non-newest / losing-lease / wrong-session / later-local-only:
    not applicable — a fork cites no change event (null source).
  - read-failed: defensive bound as in R1.
- Narrowing plant: `N-profile-fork-value` (admits exactly a
  standard-valued fork; killed by the ported probe, the rev10 fork
  matrix, and the fork old-plan substitution by admission).
- Caller bound: `source_profile_event_id` provenance (2.4: "never
  participates in new-session derivation"; cross-session input
  unresolvable from the winning-source closure). The
  good_provenance mode proves the gate ignores it.


## Record-consumption table (rev11, class-closing for C6 referenced admission)

Every record the shared profile owner consumes must pass its semantic
admission owner before its bytes are derived. "Entries" = the four shared
production entries. The call-graph gate `TestRev11RecordConsumptionCensus`
requires each consumption site to call its owner before deriving; the three
N-referenced narrowing plants preserve those call tokens while admitting one
invalid member each, so only the behavioral suites fail.

| # | Consumed record | Semantic admission owner (must admit before use) | Production call site | Positive control | Negatives (wrong creator, wrong variant, wrong lease/fencing, missing, read-failed) | Narrowing plant (behavioral kill) |
|---|---|---|---|---|---|---|
| K1 | Session Record (creation profile + kind + record digest) | Canonical shape owner (`canonicaljson` + `sessrepo` attestation) + `sessionCreationProfile` grammar gate (`lease.go`); kind selects variant via `checkCheckpointPersistence` | `checkCheckpointProfileAuthority` via `repo.GetRecord` (`lease.go`); kind threaded from `winningLeaseFor(sessionID, kind, repo)` | `TestRev8CheckpointProfileAuthority` (missing_source=false), `TestRev10ForkLocalProfile` good, `TestReview10ReferencedCheckpointAdmission` good (4 controls) | Undecodable record / invalid creation profile: integrity via `sessionCreationProfile`; missing record bytes (`GetRecord` failure): unavailable with cause; wrong kind for variant: see K2-K4 wrong_variant | `N-profile-first-value` (value), `N-profile-first-source` (presence); referenced variant via `N-referenced-variant` |
| K2 | Winner Checkpoint Record | `checkWinnerCheckpoint` (`lease.go`): session + predecessor/self lease tuple + creator-holder + `checkCheckpointPersistence` (Session.kind) + `checkCheckpointEventHeads` (at-or-before owning lease) + `checkCheckpointProfileAuthority` | `winningLeaseFor` (`lease.go:359`) from `bindPlan` (`plan.go:274`), `Revalidate` (`revalidate.go:101`), `authorize` (`summary.go:366`) | `TestValidSuccessorBuildsAndRevalidates`, `TestRev7CheckpointHeadAuthority` controls, `TestReview10ReferencedCheckpointAdmission` good | Wrong creator: `TestRev5CheckpointCreatorMustBeHolder` (unavailable); wrong variant: `TestRev6CheckpointPersistenceMustMatchSession` + `TestRev6PersistenceMismatchIsObservationUnavailable` (unavailable); wrong lease: `TestWrongPredecessorCheckpointMustRefuse` (unavailable); missing: `TestRev4MissingCheckpointMustRefuse` (unavailable); read-failed: defensive unavailable with cause | `N-checkpoint-holder` (winner creator), `N-checkpoint-persistence` (variant), `N-checkpoint-placeholder` (missing), `N-checkpoint-heads` (heads precision) |
| K3 | Necessary-ancestor Checkpoint Record (every non-winner lease in winning ancestry) | `checkCheckpointBinding` (`lease.go`): same five gates as K2 for (current, owner) | `checkAncestorCheckpoints` (`lease.go`) from `winningLeaseFor` | `TestRev5AncestorCheckpointAuthority` complete control, `TestRev7CheckpointHeadAuthority` ancestor controls | Wrong creator: `TestRev5AncestorCheckpointBinding/wrong_creator` (unavailable); wrong variant: `TestRev6CheckpointPersistenceMustMatchSession` ancestor arm (unavailable); wrong lease: `TestRev5AncestorCheckpointBinding/wrong_lease` (unavailable); missing: `TestRev5AncestorCheckpointAuthority/missing_ancestor` (unavailable); read-failed: defensive unavailable | `N-ancestor-holder` (ancestor creator), `N-ancestor-checkpoint` (ancestry gate), shared `N-checkpoint-persistence` / `N-checkpoint-heads` |
| K4 | REFERENCED resume Checkpoint Record (`session.resumed.checkpoint_id`) | `checkReferencedCheckpointBinding` (`lease.go`, rev11): owning lease tuple resolved in winning ancestry chain + creator-holder + `checkCheckpointPersistence` (Session.kind) + `checkCheckpointEventHeads` (at-or-before owning lease); called by `referencedCheckpointProfile` before `profileClosure` | `checkClosureProfilePairs` session.resumed case -> `referencedCheckpointProfile` -> `checkReferencedCheckpointBinding` (`lease.go`), with chain + Session.kind threaded from `winningLeaseFor` | `TestReview10ReferencedCheckpointAdmission` good (4 controls), `TestRev10ResumeReferencedCheckpoint` good/good_prechange/good_divergent, `TestRev8ProfileResumePositiveControl` | Wrong creator: wrong_creator (unavailable, 4 controls + 12 negatives via `TestReview10ReferencedCheckpointAdmission`, `TestRev11ReferencedCheckpointAdmissionRefusalClass`, `TestRev11ReferencedCheckpointAdmissionOldPlan`); wrong variant: wrong_variant (unavailable, same suites); wrong lease/fencing (no admitted owning lease at epoch): wrong_lease (unavailable, same suites); missing: bad_missing/bad_missing_stale (unavailable); wrong-session: bad_wrong_session (unavailable); unresolvable head: bad_unknown_head (unavailable); read-failed: defensive unavailable with cause (same helpers) | `N-referenced-creator` (hostB creator; census tokens preserved, behavioral fails), `N-referenced-variant` (idA variant; census passes, behavioral fails), `N-referenced-lease` (leaseCycleB tuple; census passes, behavioral fails) |
| K5 | Lease Records (winning ancestry + owning leases for K2-K4) | `parseLeaseRecord` (canonical shape + identity, `invalid_config` on grammar failure) + `checkLeaseChain` (succession: missing name/record unavailable, contradictory/self/cyclic/skipped integrity) + winner tuple rule via `sessstate.Compare` | `validatedLeasesBySession` + `winningLeaseFor` (`lease.go`) | `TestValidSuccessorBuildsAndRevalidates`, `TestRev5AncestorCheckpointAuthority` complete, all R5 good modes | Missing winning record: unavailable (`TestReviewerPlanRequiresLeaseRecord`); contradictory/duplicate/self/cyclic/skipped: integrity (`TestSuccessorCycleMustRefuse`, `TestSelfPredecessorWithCheckpointMustRefuse`, `TestRev4SelfPredecessorMustRefuse`); fencing without owning lease (referenced): wrong_lease via K4 (unavailable) | `N-lease-missing` (absence), `N-chain-self` (self link) |
| K6 | `profile.changed` events (authority inputs: newest + target in governing closure) | Index membership = authority (`profileClosure` over `repo.ListEvents` predecessors; divergent blobs outside index never enter) + `eventProfileTarget` (`to` member, integrity on grammar failure) + `checkProfilePairSource` (newest + type + closure membership, integrity) | `checkClosureProfilePairs` profile.changed case + launch/resume derivation (`lease.go`) | `TestRev8ProfileChangedPositiveControl`, `TestRev8ProfileTwoGenerationControl`, R5 good (E1 closure) | Non-newest: `TestRev8ProfileSourceRefusals/non_newest`, `TestRev10ResumeReferencedCheckpoint/bad_non_newest` (integrity); losing-lease: `/losing_branch`, `/bad_losing` (integrity); wrong-type/cross-session/dangling: integrity; missing source digest: integrity; read-failed (`GetEvent` I/O): unavailable with cause; undecodable payload: integrity | `N-profile-newest` (first-wins), `N-profile-value` (yolo mismatch), `N-profile-change-direction` (token-preserving from-for-to), `N-profile-resume-newest` (referenced first-wins) |
| K7 | Other events (launches, resumes, forks, idle/transfer/stop/create for closure routing) | Chain index + closed payload shape at append (`canonicaljson`) + `eventProfilePair` / `eventPayloadMembers` (integrity on undecodable, unavailable with cause on I/O failure) + per-kind pair binding via `checkProfilePairSource` | `checkClosureProfilePairs` launch/resume/fork cases (`lease.go`) | R1-R6 good modes (first/later launches, pre-change/divergent resumes, fork good/good_provenance) | Contradictory value/presence: integrity (`first_wrong_profile`, `bad_first_profile/source`, `bad_later_stale/non_newest`, `bad_stale`, `bad_profile`, `bad_creation_fallback`); dangling closure predecessor: integrity (`dangling_predecessor`); missing pair members: integrity; read-failed: unavailable with cause | `N-profile-first-source` (presence), `N-profile-first-value` (first value), `N-profile-fork-value` (fork value), `N-profile-resume-missing` (missing-reference creation pair) |

Status: 7 of 7 consumption rows DRIVEN through the shared entries with named positives, negatives, and narrowing plants. 0 undriven rows. 0 scope exemptions by declaration. The gate `TestRev11RecordConsumptionCensus` fails if any K4 derivation reaches `profileClosure` without calling `checkReferencedCheckpointBinding` first (source-order check), and the three N-referenced plants prove the gate alone never suffices.

## Read-failed bound (all rows)

`eventPayloadMembers` and `eventProfileTarget` wrap every chain-read
failure in `observation_unavailable` with the cause kept, and the
nil-repo handle refuses the same class. These branches are defensive:
the repository fully re-verifies every load and parks corrupt
sessions before the shared gate runs, so an isolated mid-derivation
read failure is unreachable without a storage race. Missing
(supplied-set lookup miss, no I/O) and read-failed (I/O error) stay
distinct errors in code; missing is driven (R5), read-failed is
covered by construction through the retained admitting-path helpers.

## Caller-owned boundaries (not rows)

- `fork.created.source_profile_event_id` provenance: caller-owned
  cross-session input (SPEC 2.4); R6 proves non-participation.
- `session.resumed.checkpoint_id` existence as a generic storage
  service: outside this leaf; access to the already-supplied
  checkpoint closure is owned (R5).
- `task_board.launched` creation-lease repeat and provider/launch-mode
  binding: 5.2 non-pair relationship; the reducer checks provider and
  creation lease, launch_mode has no consumer here (retained verdict
  boundary, no new finding).
- `profile.changed.from` continuity and confirmation gating:
  publication-flow concerns (2.4); canonical shape checks from != to.
- CLI invocation, wire Result/exit mapping, lifecycle effects,
  publication/storage/transport, materialization: accepted
  caller-owned boundary (0 of 8 CLI rows).

## Status

6 of 6 pair rows (R1-R6) DRIVEN plus 7 of 7 record-consumption rows
(K1-K7) DRIVEN through the shared production entries. 0 undriven
rows. 0 scope exemptions by declaration.
