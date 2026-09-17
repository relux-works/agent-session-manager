# TASK-260830-21gygk — revision 4: changes requested

Reviewed immutable CR-TASK-260830-21gygk-4 revision 4, candidate tree
`8c74443973a76299cede4559895cf4ec10c8abf7`, base
`8cf4aaaa190e6a11dff2661aa6806ce476128653`, checkpoint
`83640d191f78f0cd685e5f709027819799efe1cf`.
Pinned authority: internal/specdoc/SPEC.v0.6.0.md, source commit
`0cbdf100dbf84df50c64f792b1f940e3a67859a6`.
Scope remains shared 14.7/14.7.1/14.7.2 and authoritative summaries
5.7/14.7.3; CLI integration and durable recovery remain caller-owned.

## Findings requiring rework

### P1 — Canonical identity is now real, but the winning lease is not semantically validated

`internal/sessquery/lease.go:153-185` selects the greatest tuple and checks
only whether any candidate has the named predecessor ID. That candidate can
be the winner itself. There is no predecessor epoch+1 check, and no validated
checkpoint/session/predecessor-lease binding. `sessrepo/lease.go:21` delegates
to canonical identity/shape verification; that verifier does not acquire or
validate the related records. SPEC 5.3:1957-1960 requires all these facts;
14.7.2:11953 requires a *validated* winning Lease Record.

Reviewer probes through supported repository creation/event APIs and
Reader.LeaseRecords -> Reader.BuildPlan -> Reader.Revalidate:

- `TestRev4SelfPredecessorMustRefuse`: an epoch-2 record names its own lease ID
  as predecessor, while the event chain is a normal epoch-1 -> epoch-2
  transfer. BuildPlan succeeds and Revalidate returns nil. **FAIL**.
- `TestRev4MissingCheckpointMustRefuse`: an epoch-2 record references the
  all-zero digest fixture; no validated checkpoint is supplied or acquired.
  BuildPlan succeeds and Revalidate returns nil. **FAIL**. This is a missing
  semantic acquisition/admission boundary, not proof that a checkpoint was
  passed through an API that does not exist.

The first is a demonstrated admission defect even within the new input model.
Calling raw records validated inputs does not implement the new linkage gate.
Validate complete predecessor legality and checkpoint relationships through
an owned, supported authority input/reader, then derive the winner from valid
records. Do not replace required related facts with digest-shape checks or
placeholder fixtures. Add positive valid-successor and narrowing negative
cases, including self/cyclic/wrong-epoch and unavailable/wrong-session or
wrong-predecessor checkpoint evidence. Preserve genuine canonical identity.

### P1 — Required parked authority is silently treated as absent

`internal/sessquery/revalidate.go:221-222` skips a parked copy of the selected
UUID; `collectAuthorityHeads` at 271-272 skips it again. Thus neither its
unknown authority nor its presence invalidates the prior plan. SPEC
14.7.1:11912-11916 separates partial/malformed authority from absence;
14.7.2:11966-11976 requires complete authority and preserves failed-read
classification. Being unable to prove a contradiction does not prove currency.

`TestRev4ParkedAuthorityMustRefuse` builds a peer plan, then uses production
CreateSession's existing crash hook after record persistence to expose a
conflicting same-UUID local record with a missing chain index. InspectLocal
confirms a parked failed observation; Revalidate nevertheless returns nil.
**FAIL**. The shipped `TestRevalidateUnionParkedCopyIgnored` explicitly
protects this permissive result and must change. Refuse at build/revalidation
when a required same-session source is incomplete, preserving the appropriate
integrity/read/unknown class; retain raw diagnostic inspection. A genuinely
absent source/session and a valid lagging copy remain distinct cases.

### P2 — Closed summaries admit invented capability names

`internal/sessquery/summary.go:208` checks only printable length of capability
keys, then `authorize` returns them. SPEC 14.2:11423 requires the closed
capability-name type; 7.3:2842-2860 fixes the seven-name provider registry
(native_resume, portable_store, managed_pty, appserver, task_board_primary,
prompt_spawn, native_goal_binding). The pinned catalog independently records
that same registry. This is not the distinct 15-name session-adapter registry.

`TestRev4UnknownCapabilityMustRefuse` supplies
`invented_capability:{status:available,enabled:true}` and drives both
AuthoritativeStatus and AuthoritativeList. Both succeed and publish that
entry. **FAIL**. Use the authoritative capability vocabulary and its existing
observation ownership contract; test each admitted name and an unknown name
through both entries. Correct the existing positive fixture using `exec`
and `net` if those are not members of the bound registry. Keep empty
non-nil maps valid and missing observations unavailable.

## What this revision does close

- Actual epoch-1 canonical Lease Record identity now flows through
  sessrepo.AttestLeaseRecord; the four-field substitute is gone.
  TestRev3RealLeaseRecordIdentity passes in the exact candidate.
- Reviewer `TestRev4DifferentLeaseDigestRefuses` changes only created_at in
  the supplied canonical record, re-identifies it, and preserves its tuple.
  Revalidate refuses stale. **PASS**. This independently tests digest
  substitution; the candidate test with the similar name changes LeaseID.
- Unknown workspace/capabilities/process, missing host metadata, 65-character
  host name and bootstrap/mixed-list cases are covered and green.
- TestBuildWithLaggingLeaseCopy passes for its selected epoch-2 source and
  lagging epoch-1 peer. This proves that direction under the supplied-record
  model, not complete normative successor admission (finding 1).
- Existing record divergence, tombstone and unreadable-peer tests pass.
  No earlier settled predecessor crash/reducer/closed-enum review is reopened.

## AC coverage ratio

**8 of 8 AC rows driven at shared production entries; 4 of 8 established,
4 of 8 partial/failing.** The denominator retains the full task's existing
8-row decomposition, including SelectionPlan and negative/refusal obligations;
the producer's 6-row regrouping drops their explicit accounting. This is not
an exhaustive count of every normative fixture. **0 of 8 public CLI rows
implemented** remains the accepted caller-integration bound, not a finding.

| Row | Production entry and named candidate tests | Assessment |
| --- | --- | --- |
| UUID/name | Reader.Resolve; TestResolvePinnedPrecedence, TestResolveBareIdentityUnion | Established within supplied config/repository boundary |
| Qualified selector | Reader.Resolve -> resolveExplicit; TestResolveQualifiedSourcesSelectOneIndex, TestResolveQualifiedSourceNeverFallsBack, TestResolveExplicitSourceReadFailures | Established |
| Ambiguity | Reader.Resolve -> matchName/checkRecordAgreement; TestResolveExactNamesAndASCIICollisions, TestResolveBareIdentityUnion | Established within reached tiers |
| List summaries | Reader.List/AuthoritativeList; TestSummaryBootstrapRefusals, TestSummaryMixedListRefusesWhole, TestAuthoritativeUnknownObservationsRefuse | Partial: lease semantic authority and capability-name refusal |
| Status summaries | Reader.AuthoritativeStatus; TestAuthoritativeStatusHealthy, TestAuthoritativeMissingProcessRefusesStatus, TestAuthoritativeHostNameBound64 | Partial: same findings |
| Stable sorting | Reader.List/Resolve; TestListStatusDerivedFactsAndStableOrder, TestResolvePeerOrderAndReplicatedIdentity | Established |
| SelectionPlan | Reader.BuildPlan/Revalidate/ParsePlan; TestBuildPlanBindsCurrentFacts, TestRevalidateDetectsEachFactChange, TestBuildWithLaggingLeaseCopy, TestRev3RealLeaseRecordIdentity | Partial: findings 1–2 |
| Negative/refusal | Resolve/List/Status/BuildPlan/Revalidate; candidate refusal suites plus reviewer_rev4_test.go and mutation probes | Partial: demonstrated missing gates above |

The new reviewer tests are attached evidence, not committed candidate tests.
Producer must carry corrected semantic regressions into the next candidate.

## Mutation evidence and record-guard subsumption

Retained mutants-rev3 manifest has exactly **40 N + 3 B = 43 KILLED**, plus
2 CONTROL exits 0, 1 NOT_APPLIED and 1 COMPILE_OR_HARNESS_FAILURE. All 43 raw
logs' named failures match their manifest entries. No producer SURVIVED control
was delivered in that manifest. The retained source copy is mostly identical,
but lease.go still directly calls canonicaljson.VerifyObjectIdentity,
provhost's count gate is older, and TRACEABILITY differs; the new forwarding
sessrepo/lease.go is missing there. This is pre-wrapper evidence, not literally
an exact-CR4 battery. The wrapper forwards unchanged arguments/results, so the
old behavioral runs remain useful with that explicit equivalence bound.
The exact scripts, baseline differences, source hashes and raw logs are bundled.

Fresh probes execute the *unchanged run() function extracted from CR4's
mutate.py* against the exact candidate suites, with real exits and restoration:

| Plant | Classification | Exit | Named behavior |
| --- | --- | ---: | --- |
| Harmless applied comment | SURVIVED | 0 | Actual same-instrument classifier control |
| N-record-nonempty | SURVIVED | 0 | Existing candidate suite does not kill this narrowed comparison |
| N-observation-host | KILLED | 1 | TestAuthoritativeMissingHostMetadataRefuses |

The nonempty-record survivor is **not demonstrated unauthorized acceptance**.
`TestRev4ChangedLiveRecordRefuses` constructs a replacement repository using
CreateSession/AppendEvent for the same UUID and a changed immutable record,
then swaps the supported Reader.Local handle. The copy is live, non-parked,
and its RecordID is nonempty and different. Baseline refuses `session record
changed`; with just the narrowed record guard it still refuses `authority
heads changed` (test exit 0). This closes the earlier inability to produce a
live witness. The new first event references the new record digest, changing
its event identity and hence heads. Independently fingerprintIndex directly
includes every row's RecordID (`digest.go:90-141`), so an admitted nonempty
change cannot retain that fingerprint absent a hash collision. No claim that
the parked replacement fixture covers live records is needed. Preserve this
explicit subsumption test/evidence instead of counting a refusal-order mutant
as complete semantic coverage.

## Validation, preservation and evidence bounds

- Go library target: darwin/arm64, Go 1.25.5; no iOS build target applies.
  Managed project skill links are absent in this Story checkout; read the
  existing global Go testing skill, plus the assigned project-management and
  architecture-diagrams skills. No install or infrastructure change.
- Exported immutable Git tree without .task-board, avoiding a large stale
  board copy. 44/44 changed paths match live bytes; after probes all are still
  unchanged. No commit, index mutation, branch operation or product edit.
  HEAD stays the checkpoint above; HEAD..main count 0 is recorded, not used
  as a substitute for immutable-tree identity. No --help or board path enters
  the CR delta.
- Independently reran `go test ./internal/sessquery ./internal/sessrepo
  ./internal/sessstate ./internal/provhost -count=1 -v`: **exit 0** on the
  exact archive, before injecting probes. This includes the changed static
  attestation-site bound and predecessor regression suites.
- Reviewer `go test ./internal/sessquery -run '^TestRev4' -count=1 -v`:
  **exit 1**, four failed top-level tests and two passing controls. Source
  and full output attached. An initial wrong-relative-path write produced
  an empty test selection; it is explicitly excluded from passing evidence.
- Fresh mutation driver exit 0 means collection completed, not all mutants
  killed. Per-plant exits above. Live-record subsumption probe exit 0.
- `git diff --check BASE TREE`: exit 0. Reused CR4 publication log's five
  exit-0 shell markers for gofmt/build/vet/full-test/diff gates, not an
  independently rerun 26-command battery. That bounded log also contains
  366 board-copy validation issues; it is not a clean board validation
  claim. Full coverage/race results are producer reports, not fresh reviewer
  measurements. None overrides the failing semantic regressions.
- Read previous review verdict/evidence inventory, rework/result/addendum,
  recovery outcome and current traceability. Accepted predecessor code is
  preserved apart from the scoped exact-name fix and new attestation wrapper;
  package rerun passes. Current README overstates complete authority and must
  be corrected with the fixes; append current outcome to the task logbook.
- Shared reads introduce no durable mutation; existing creation crash hooks
  exercise composition, not a new crash protocol. No unsupported runtime or
  external capability is independently attested by this review.

## Disposition / task logbook

**Changes requested; route TASK-260830-21gygk to to-dev.** Findings are concrete
shared-layer implementation gaps, not a new human-only product decision.
Keep the full assigned objective. Attach this verdict and review-evidence-rev4
before routing. Incomplete implementation/conformance/negative-evidence
checklist claims are unchecked; completed checks retain their actual bounds.
No accept_cr, commit, checkpoint, integration, landing or commit_ack performed.
