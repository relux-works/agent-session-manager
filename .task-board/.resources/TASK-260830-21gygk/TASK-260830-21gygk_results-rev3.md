# TASK-260830-21gygk rev3 rework outcome

## Verdict request
Ready for review. Candidate left UNCOMMITTED in STORY-260830-3tq4ns worktree.

## What changed (rev3 findings)
1. Real Lease Record admission: new Reader.LeaseRecords input; sessrepo.AttestLeaseRecord owns canonical identity (4th allowlisted site with provhost bound rewrite); sessquery selects greatest (epoch, lease_id) winner with predecessor linkage; plan/summary bind digest+triple+holder from same record. Deleted four-field fingerprint substitution. Tests: TestRev3RealLeaseRecordIdentity, TestRev3DifferentAttestationMustNotAuthorize, TestReviewerPlanRequiresLeaseRecord (presence+absence).
2. Greatest-winner union: checkUnionCopy uses compareLeaseTuple; lagging smaller tuples stay current (TestBuildWithLaggingLeaseCopy), greater tuples refuse stale; agree/tombstone/record-divergence preserved. AuthorityHeads bind winning lease digest plus every non-parked source tail, sorted unique; Revalidate checks union before heads so specific lease/tombstone reasons are preserved.
3. Closed summaries: workspace_status enum (absent|current|staged|conflict|unsupported), CapabilitySummary map (0..7, only available enables, detail 0..2048), warnings from projection, owner_host_name 1..64, process_present boolean required for status (carried for lists). Unknown required observations refuse selector_observation_unavailable; established absence (workspace absent, empty non-nil capabilities) stays valid; whole mixed lists refuse. Tests: UnknownObservationsRefuse, MissingWorkspace/Capabilities/Process, HostNameBound64.
4. Narrowing evidence repaired: fixed 3 NOT_APPLIED (N-rev-lease, N-union-lease, N-observation-host) to new code; added 5 new narrowing mutants (workspace, capabilities, process, host-bound64, lease-missing); removed 2 SURVIVED extras. Battery: 40N+3B=43 ALL KILLED, controls green. Prior claim 33N+3B corrected to 35N+3B baseline and 40N+3B final.

## AC coverage ratio: 6 of 6 rows driven
- Resolve UUID selectors via Reader.Resolve/BuildPlan: resolve_test TestResolveUUIDVariants, plan_test mustBuild alpha.
- Resolve name selectors via Reader.Resolve: TestResolveExactNamesAndASCIICollisions.
- Resolve qualified selectors via Reader.Resolve/BuildPlan: TestResolveQualifiedAmbiguityAndExclusions, TestBuildPlanPeerSourceFacts (beta@peer:workstation).
- Ambiguity refusal via Reader.Resolve: qualified ambiguity tests (same-name local+peer refuses, no fallback).
- List/status summaries via Reader.List/Status/AuthoritativeStatus/AuthoritativeList: summary_test healthy/bootstrap/mixed/observation/host-bound/checkpoint tests.
- Stable deterministic sorting via Reader.List/fingerprintIndex/collectAuthorityHeads: peer-order/session-order/tie-break behavioral mutants (B-*) KILLED; List bytewise session-ID order asserted.
Prose replaced by ratio per DoD; no row left to prose.

## Negative/refusal gates (production call sites named)
- Reader.BuildPlan/Revalidate: bootstrap_incomplete (record-only), observation_unavailable (missing lease/host/observations), invalid_config (malformed lease/host/observations, unknown local host), plan_stale per-member (record/lease/heads/index/union/action/destination/expectations), integrity_failure (cross-source record/owner divergence), revocation (peer_not_allowlisted), read failures (local/peer chain, repository path).
- Reader.AuthoritativeStatus/List via authorize: bootstrap_incomplete, observation_unavailable (host/lease/workspace/capabilities/process/timestamp), invalid_config (malformed inputs, 65-char host).
- Each gate ships a narrowing mutant (admit exactly one member) KILLED through the same instrument; delete-only and control green/compile-failure also recorded. No token-search gate lacks its token-preserving mutant.

## Validation (all observed in-session)
- go test ./... -count=1: PASS all packages (provhost bound updated to 4 sites, 0 elsewhere).
- go test ./internal/sessquery/ ./internal/sessrepo/ ./internal/provhost/ -cover: 87.9%/87.4%/86.0%.
- gofmt -l internal/: clean. go vet ./internal/sessquery/ ./internal/sessrepo/: clean.
- Mutant battery .temp/TASK-260830-21gygk/mutants-rev3/mutants.json: 40N+3B KILLED, 0 BAD, control-before/after green.
- sessquery verbose: 55 top-level PASS, 0 FAIL.

## Docs/traceability
- internal/sessquery/TRACEABILITY.md: acceptance rows (lease/union/summary tests), refusal table (+5 new gates), bounds (real lease digest, lease+tails heads, greatest winner).
- README.md: immutable-plans and authoritative-layer paragraphs rewritten (no envelope substitution, no standalone-objects claim).
- internal/sessrepo/lease.go: new attestation owner; internal/provhost/identity_test.go bound 3->4 with lease justification.

##曹操Notes
- Disk ENOSPC during battery: removed extracted worktree scratch review-rev2/review-rev3 dirs (520M each, gitignored) and /tmp mutant dirs; board resources and new mutants-rev3 evidence retained. Candidate diff excludes .task-board and .temp scratch.
- Changed-nonempty live-record case: selected-source filesystem surgery parks with empty digest (first-event predecessor mismatch), so the parked N-rev-record plus union live-divergent N-union-record plus index subsumption cover the gate; no separate nonempty mutant shipped.
- No durable mutation in scope (read projection only); crash/idempotency N/A with justification.
