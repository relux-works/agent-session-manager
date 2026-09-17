# TASK-260830-21gygk — rev3 review: changes requested

Immutable CR-TASK-260830-21gygk-3 revision 3; tree
`ad077090f354e633a6a7a3eed48ef35dc60c0ad4`, base
`8cf4aaaa190e6a11dff2661aa6806ce476128653`, Story checkpoint
`83640d191f78f0cd685e5f709027819799efe1cf`.
Authority: candidate internal/specdoc/SPEC.v0.6.0.md, pinned spec commit
`0cbdf100dbf84df50c64f792b1f940e3a67859a6`. This reviews the assigned shared
library, with CLI invocation/envelopes and durable recovery left to their owners.

## Findings requiring rework

### 1. P1 — Required Lease Record identity is still replaced with another object

`internal/sessquery/digest.go:136` fingerprints only session_id, lease_epoch,
lease_id and owner_host_id. `plan.go:266` binds that result as LeaseRecordID;
`revalidate.go:105` verifies the same substitute. No actual Lease Record is
read or admitted by these production paths. SPEC 14.7.2:11953 requires the
validated winning Lease Record digest, with all lease facts from that same
record; 5.3 defines its closed schema and identity. A four-field observation
is a different object, even when all four values agree. Calling it locally
attested does not make it that Lease Record.

Reviewer production probes, both FAIL (test command exit 1):
- `TestRev3RealLeaseRecordIdentity`: constructs a real epoch-1 create Lease
  Record, validates it through `canonicaljson.VerifyObjectIdentity`, then
  drives `Reader.BuildPlan`. Expected actual record digest starts `28730fb6`;
  returned envelope-observation digest starts `5f6f848f`. Full values and
  validated record bytes are in regression.log.
- `TestRev3DifferentAttestationMustNotAuthorize`: BuildPlan → Marshal →
  ParsePlan → Revalidate accepts the different observation attestation with
  no Lease Record ever provided.

Exact probe bound: the candidate exposes **no supported Lease Record input**
on Reader/Repository. The real validated fixture is an independent identity
oracle, not a claim to have supplied it through an invented storage path.
This absent integration is itself the remaining requirement. Add real shared
Lease Record acquisition/admission, bind its canonical identity and validated
heads, and test same-record facts, changed/missing record and substitution.
Do not relabel an envelope digest or defer this shared requirement to CLI.
The added nonempty source host and positive tuple checks are useful fixes;
the old field-presence reviewer test is insufficient for semantic identity.

### 2. P1 — The authoritative summary still succeeds without required facts

`summary.go:50-90,239-241` treats unknown process/workspace/capability
observations as successful nil fields. It exposes WorkspacePresent rather
than the required workspace_status, and a string slice rather than validated
CapabilitySummary values. `summary.go:38` also admits host names up to 128
characters. SPEC 14.7.3:12348-12372 preserves the closed summary shape and
requires unavailable required observations to refuse. The inherited table at
11423 requires owner_host_name[1..64], workspace_status, capability summaries
and warnings; status at 11437 requires a boolean process_present.

`TestRev3UnknownRequiredObservationsRefuse` fails for both
Reader.AuthoritativeStatus and Reader.AuthoritativeList: an empty observation
map returns successful output instead of selector_observation_unavailable.
`TestRev3HostNameBound64` fails because a 65-character host name succeeds.
The diagnostic JSON records these missing facts; JSON member casing itself
is NOT a finding because wire serialization is caller-owned.

Connect the shared authoritative layer to the required validated observation
facts, distinguish unknown from established absence (including a legitimately
empty capability map), enforce the actual bounds, and refuse the whole list
when a required row is unrepresentable. Caller-owned formatting cannot recover
missing semantics without reimplementing this layer. Owner/lease facts also
need the actual Lease Record integration in finding 1. New bootstrap,
mixed-list, host-metadata and checkpoint-timestamp refusals do work; preserve
them and the diagnostic low-level projector.

### 3. P1 — Copy equality is not complete authority union/winner derivation

`revalidate.go:201-233` compares every nonzero copy's lease for equality with
the selected source. `bindPlan` calls it for a new plan as well. A known older
lease is not a contradictory winning lease: SPEC 5.3 selects the greatest
(epoch, lease_id) in the union and preserves losing histories. 14.7.2 requires
that winner, current tombstones and the complete authority union.

`TestRev3BuildWithLaggingLeaseCopy` drives production repository event entry
points and then fresh `BuildPlan("beta@local")`: local has epoch 2, peer has
the same immutable record at epoch 1. BuildPlan returns selector_plan_stale
although no prior plan exists and the union has a deterministic epoch-2 winner.
The test FAILS. Bound: these are the candidate's supported event-chain facts,
not proof of actual Lease Record storage (finding 1). It exposes that equality
alone cannot implement the required union even within that supported model.

Derive and validate the actual union/winner, while retaining exact selected
source provenance and never re-resolving by name. Bind all lease/event/tombstone
heads used in the decision; current code stores only the selected event tail.
Also audit the unconditional parked-copy skip at revalidate.go:206: failed
observations cannot establish a complete authority read. I did not rerun a
separate parked-copy regression; do not count that audit question as another
measured failure. Preserve the new same-UUID record-conflict refusal and
read-failure classes, which pass. Different-UUID same-name gain is still not
itself a defect or a reason to retarget the plan.

## Negative evidence and claim corrections

The same delivered run()/classifier was independently executed on three
isolated plants against the candidate's unmodified behavioral suites:

| Plant | Result | Exit | Meaning |
| --- | --- | ---: | --- |
| Harmless applied comment | SURVIVED | 0 | Actual same-instrument survivor control |
| N-record-nonempty | SURVIVED | 0 | Admit a changed nonempty record digest for idA at the selected-source comparison; parked/empty records still refuse |
| N-observation-host | KILLED | 1 | TestAuthoritativeMissingHostMetadataRefuses fails for one owner's missing metadata |

The shipped N-rev-record now narrows the gate but exercises a parked empty
record and fails on a later refusal reason. That proves the tested error
ordering for that case; it does not establish behavior for a validated changed
record. The new survivor is a coverage gap at this comparison, **not proof of
an end-to-end unauthorized acceptance**: later heads/index checks may subsume
it. Add a meaningful changed-record witness or document and test exact
subsumption, rather than calling all record cases covered.

The producer manifest actually contains **35 N + 3 B = 38 KILLED**, not the
33 N + 3 B described in its resource description/outcome. Its other results
are CONTROL x2, NOT_APPLIED x1, COMPILE_OR_HARNESS_FAILURE x1; none is SURVIVED.
I located 41 raw logs (the NOT_APPLIED plant has none), checked named failures
against every manifest entry, and preserved those logs in this bundle. They
match. The original mutation source copy is absent at the inspected run path;
therefore exact source attribution of all old plants is not independently
established by a retained copy. I did not rerun the full battery or convert
that absence into a failing behavioral result. Fresh three-plant evidence is
exact-candidate-bound. Correct counts and “all four addressed”, “8/8 complete”
and actual-Lease-equivalence claims in outcome/README/TRACEABILITY after fixes.

## AC coverage ratio

**8 of 8 AC rows driven through shared production entries; 4 of 8 established
within the supplied-repository boundary; 4 of 8 partial/failing.** This uses the
existing eight-row AC decomposition, not a claim to enumerate all normative
fixture clauses. **0 of 8 public CLI rows delivered** remains an explicitly
accepted caller-integration bound, not a rejection reason.

| Row | Production call site and named candidate tests | Result |
| --- | --- | --- |
| UUID/name | Reader.Resolve; TestResolvePinnedPrecedence, TestResolveBareIdentityUnion | Established within supplied config/repository boundary |
| Qualified selectors | Reader.Resolve → resolveExplicit; TestResolveQualifiedSourcesSelectOneIndex, TestResolveQualifiedSourceNeverFallsBack, TestResolveExplicitSourceReadFailures | Established |
| Ambiguity | Reader.Resolve → matchName/checkRecordAgreement; TestResolveExactNamesAndASCIICollisions, TestResolveBareIdentityUnion | Established for resolution; plans separately below |
| List summaries | Reader.List/AuthoritativeList; TestSummaryBootstrapRefusals, TestSummaryMixedListRefusesWhole, TestReviewerBootstrapSummaryMustRefuse | Partial: finding 2, plus actual lease authority |
| Status summaries | Reader.AuthoritativeStatus; TestAuthoritativeMissingHostMetadataRefuses, TestAuthoritativeCheckpointRequiresTimestamp; reviewer observation/name tests | Partial: findings 1–2 |
| Deterministic sorting | Reader.List/Resolve; TestListStatusDerivedFactsAndStableOrder, TestResolvePeerOrderAndReplicatedIdentity | Established |
| SelectionPlan | Reader.BuildPlan/Revalidate/ParsePlan; TestRevalidateDetectsEachFactChange, TestReviewerOtherSourceDivergenceMustRefuse; reviewer lease/lagging-copy probes | Partial: findings 1 and 3 |
| Negative/refusal | Reader.Resolve/List/Status/BuildPlan/Revalidate; candidate refusal suites and reviewer mutation/probe logs | Partial: missing semantic admission/refusal evidence above |

## Validation and preservation

- Exported exact tree using git archive to task scratch. All 42 candidate paths
  match live worktree bytes (candidate-manifest.json). No product, branch,
  index or board checkout file edited. No stray --help tree or .task-board path
  in the immutable delta. HEAD remains the supplied Story checkpoint;
  `git rev-list --count HEAD..main` was 0, not used as a substitute for tree identity.
- Independently reran `go test ./internal/sessquery ./internal/sessrepo ./internal/sessstate -count=1 -v`
  on the exact archive: **exit 0**. Full log attached. Target is Go on
  darwin/arm64, Go 1.25.5; no iOS target exists for this library work.
- Reviewer-only probes in archive: `go test ./internal/sessquery -run '^TestRev3' -count=1 -v`:
  **exit 1**, five failed top-level tests (six failing assertions/subcases),
  one diagnostic PASS. Source and full log attached, then probe source removed
  from archive; live candidate never modified.
- Fresh three-plant driver **exit 0** means evidence collection succeeded,
  not that all plants were killed. Per-plant exits above. Runner restores bytes.
- `git diff --check BASE TREE`: **exit 0**. Reused attached producer
  validation-rework.log for full tests, coverage, race, build, vet, gofmt and
  JSON gate (all report exit=0); did not independently rerun those full gates.
  Actual rev3 publication validation resource has five `[exit 0]` markers.
  These green results do not override reviewer failures.
- Read full prior rev2 verdict and its evidence archive inventory, current
  rework outcome, mutation manifest and validation resources. No need to reopen
  settled predecessor crash/reducer/canonical reviews. provhost and sessstate
  match checkpoint; the only predecessor production delta is store.go's exact
  case-sensitive name match and its regression, covered by the package rerun.
- Missing Story skill symlinks were handled by reading the existing
  Curator-managed Go skill from main; no install or infrastructure edits.

## Disposition / task logbook

**Changes requested; route TASK-260830-21gygk (implement-name-resolution-and-summary)
to to-dev.** This is implementable shared-layer rework, not blocked on a new
human product decision. Preserve full assigned scope. This outcome records
review findings in the task logbook; product LOGBOOK remains read-only.
No accept_cr, commit, checkpoint, integration or landing performed.
