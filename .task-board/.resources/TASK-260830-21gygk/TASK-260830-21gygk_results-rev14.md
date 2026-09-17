# TASK-260830-21gygk rev14 producer outcome

## Candidate and authority

- Task: `TASK-260830-21gygk`, Story: `STORY-260830-3tq4ns`.
- Candidate source: `e119aaeede327580a470238fdb1ccf340a785b39`; Story checkpoint/base: `8626fb361c1d6b62d5f95c7be4cb33c85d4aadf3`.
- Normative authority: `relux-works/agent-session-manager-spec@v0.6.0`, commit `0cbdf100dbf84df50c64f792b1f940e3a67859a6`; historical scope retains v0.5.0 commit `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`.
- The candidate is intentionally uncommitted in the managed Story worktree. Foreign `.task-board` state was preserved and is not part of this product scope.

## AC coverage

Shared-library coverage is **8 of 8 AC rows driven**. Each row is exercised through a production entry point; the named tests are not validator-only tests.

| AC row | Production call site | Named evidence |
| --- | --- | --- |
| UUID selectors | `Reader.Resolve` → `resolveBare`/`selectIdentity` | `TestResolveLocalNameAndUUID`, `TestParseSelectorLiteralGrammar`, `TestResolveBareIdentityUnion` |
| Name selectors and exact precedence | `Reader.Resolve` → `matchName`; retained `Repository.Resolve` exact-name gate | `TestResolveExactNamesAndASCIICollisions`, `TestResolveQualifiedNameBeforeUUIDInSource`, `TestResolveLocalNameAndUUID` |
| Qualified selectors | `Reader.Resolve` → `parseSelector` → `resolveExplicit` → `locateSource` | `TestParseSelectorLiteralGrammar`, `TestResolveQualifiedSourcesSelectOneIndex`, `TestResolveQualifiedSourceNeverFallsBack`, `TestResolveExplicitSourceRefusals`, `TestResolveExplicitSourceReadFailures` |
| Ambiguity and cross-copy agreement | `matchName`, `selectIdentity`, `checkRecordAgreement` | `TestResolveExactNamesAndASCIICollisions`, `TestResolveQualifiedAmbiguityAndExclusions`, `TestResolveBareIdentityUnion` |
| List summaries | `Reader.List` → repository read/projector → `checkSummaryRepresentable` | `TestListStatusDerivedFactsAndStableOrder`, `TestListStatusCheckpointAndProjectionFailure`, `TestSummaryMixedListRefusesWhole`, `TestSummaryBootstrapRefusals` |
| Status summaries | `Reader.Status`, `Reader.AuthoritativeStatus`, `Reader.AuthoritativeList` | `TestAuthoritativeStatusHealthy`, `TestAuthoritativeMissingHostMetadataRefuses`, `TestAuthoritativeMissingProcessRefusesStatus`, `TestAuthoritativeMissingWorkspaceRefuses`, `TestAuthoritativeMissingCapabilitiesRefuses`, `TestRev4UnknownCapabilityMustRefuse` |
| Stable deterministic sorting | `Reader.read`, `peerCandidates`, `selectIdentity` | `TestListStatusDerivedFactsAndStableOrder`, `TestResolvePeerOrderAndReplicatedIdentity`, `TestResolveBareIdentityUnion` |
| Negative/refusal, plan immutability/revalidation, and capability bound | `Reader.BuildPlan`, `Reader.Revalidate`, shared `winningLeaseFor`/`admitCheckpoint`, `checkCapabilities` | `TestRevalidateDetectsEachFactChange`, `TestReviewerOtherSourceDivergenceMustRefuse`, `TestRev5AncestorCheckpointAuthority`, `TestRev5CheckpointCreatorMustBeHolder`, `TestRev7CheckpointHeadAuthority`, `TestRev14CapabilitySeal`, `TestRev14RecordConsumptionCensusRejectsAlternatePaths`, mutation battery |

Public CLI/lifecycle coverage is **0 of 8** by stated ownership bound: this package exposes the shared API, not an `ax` executable, CLI Result renderer, fencing, transport, or lifecycle mutation caller. The shared callers are the accepted implementation boundary. The read entries perform no durable mutation, so crash/idempotency evidence is not applicable to this operation; repository recovery tests remain green.

## Implementation result

- `sessrepo.Repository.Resolve` now requires exact name equality after folded uniqueness; UUID lookup remains the later identity tier.
- `internal/sessquery` supplies one shared selector API for UUID/name/qualified selectors, source mapping and allowlist refusals, ambiguity and tombstone exclusion, list/status summaries, stable bytewise ordering, immutable `SelectionPlan` construction, and current-fact `Revalidate`.
- Winning lease admission binds the canonical Lease Record digest and the same record's epoch, lease ID, and holder. Every necessary ancestry checkpoint is admitted through the shared semantic owner with session, owning lease, holder, persistence variant, event-head closure, and profile authority checks.
- Profile derivation consumes only the private `admittedCheckpoint` sealed capability. The exact free `admitCheckpoint` constructor creates its private token; all nine profile entries verify the seal before consuming authority facts.
- The package-wide `go/types` census resolves exact function objects, rejects alternate raw/constructed/copy paths and unresolved callees, and covers method/name collisions with an applied narrowing mutant.
- No unsupported capability is advertised; the seven-name Section 7.3 registry is the only admitted capability vocabulary.

## Validation and evidence

All listed commands completed with exit 0 unless explicitly classified as a mutation control:

- Focused final census: `go test ./internal/sessquery -run 'TestRev14CapabilitySeal|TestRev14RecordConsumptionCensusRejectsAlternatePaths|TestRev11RecordConsumptionCensus|TestRev12RecordConsumptionCensusRejectsAlternatePaths|TestRev12AdmissionPrecisionCensus' -count=1 -v`; `.temp/TASK-260830-21gygk/final-census-01.log`.
- Full package suite: `go test ./... -v`; `.temp/TASK-260830-21gygk/go-test-all-v-01.log`.
- Full coverage: `go test ./... -cover`; `.temp/TASK-260830-21gygk/go-test-all-cover-01.log`; `sessquery` reports 86.9% statement coverage.
- Full race suite: `go test ./... -race`; `.temp/TASK-260830-21gygk/go-test-all-race-01.log`.
- Static/build checks: `go vet ./...` (`go-vet-01.log`), `go build ./...` (`go-build-01.log`), `git diff --check`, and `go run ./internal/traceability/cmd/tracecheck` (`tracecheck-01.log`).
- Curator: `curator install` and `curator status --check` exit 0 (`curator-install-01.log`, `curator-status-02.log`). The earlier missing-install diagnostic is retained in `curator-status-01.log` and superseded by the successful recheck.

The final-source hashes match the mutation copy for the key rev14 files:

| File | SHA-256 |
| --- | --- |
| `internal/sessquery/lease.go` | `3ce1956872db85b39105bab1970eaf757b19d05d0753482bdba449d52d68ec2f` |
| `internal/sessquery/rev11_regression_test.go` | `d4b04bc0af2b5fb0b6c3242ad2c23c784d6095d78d110557324c460b39103ab4` |
| `internal/sessquery/testdata/mutate.py` | `703a342932f92ff3264275eef4c9fc8c8dfd7cc29da180ab3f7b3d759592f53c` |

## Mutation evidence

Raw evidence: `.temp/TASK-260830-21gygk-mutants-rev14-02/`.

- `mutants.json` contains 64 applied N/B plants: 61 narrowing and 3 ordering. All 64 are `KILLED` with a named failed behavioral test; no N/B plant is `SURVIVED` or `NOT_APPLIED`.
- `N-census-raw-owner-object` preserves the `admitCheckpoint` token, weakens exact raw-owner identity to a name check, and is killed by the `method_raw_only.go` subtest of `TestRev14RecordConsumptionCensusRejectsAlternatePaths`.
- `N-lease-missing`, `N-chain-self`, `N-checkpoint-holder`, `N-profile-change-direction`, `N-profile-first-value`, and `N-capability-name` were initially under-mapped or mis-anchored; the final run uses corrected exact production anchors and fails under each narrowed behavior.
- Controls are separate: `control-before` and `control-after` are `CONTROL` exit 0; applied `C-harmless-comment` is `SURVIVED` exit 0; `C-not-applied` is `NOT_APPLIED`; `C-compile-failure` is `COMPILE_OR_HARNESS_FAILURE` exit 1. No control is counted as a behavioral kill.
