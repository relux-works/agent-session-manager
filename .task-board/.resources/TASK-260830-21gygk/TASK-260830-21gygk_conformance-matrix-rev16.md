# TASK-260830-21gygk conformance matrix — rev16

Authority: `agent-session-manager-spec` v0.6.0, commit
`0cbdf100dbf84df50c64f792b1f940e3a67859a6`; historical compatibility scope is
v0.5.0, commit `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`.

Candidate source checkpoint: `e119aaeede327580a470238fdb1ccf340a785b39`.
Expected base: `8626fb361c1d6b62d5f95c7be4cb33c85d4aadf3`.
All rows below are shared-library evidence. Public CLI/lifecycle delivery is
outside this leaf: `0 of 8` rows are claimed as an `ax` command surface.

| AC row | Production entry and decision path | Named positive/negative tests | rev16 evidence and bound |
| --- | --- | --- | --- |
| 1. Bare UUID/name selectors | `Reader.Resolve` → `resolveBare`; local repository tier delegates to `sessrepo.Repository.Resolve` | `TestResolvePinnedPrecedence`, `TestResolveLocalNameAndUUID`, `TestResolveBareIdentityUnion`, `TestResolveAllowlistAndReadFailures` | Name-first and `id:` identity tiers, replicated identity agreement, and allowlist/read refusal are driven. `N-case-variant`, `N-repository-case-variant`, `N-id-through-names`, `N-identity-diverge`, and `N-agreement-name` are killed. |
| 2. Qualified selectors | `Reader.Resolve` → `resolveExplicit` → `locateSource` | `TestParseSelectorLiteralGrammar`, `TestResolveQualifiedSourcesSelectOneIndex`, `TestResolveQualifiedNameBeforeUUIDInSource`, `TestResolveQualifiedSourceNeverFallsBack`, `TestResolveExplicitSourceRefusals`, `TestResolveExplicitSourceReadFailures` | First-`@` grammar, exact `local`/`peer:`/`id:` source forms, name-before-UUID within one source, no fallback, and distinct source/read refusals are driven. The literal and mapping narrowing mutants are killed. |
| 3. Ambiguity and collision semantics | `matchName`, `selectIdentity`, `checkRecordAgreement` | `TestResolveExactNamesAndASCIICollisions`, `TestResolveQualifiedAmbiguityAndExclusions`, `TestResolveBareIdentityUnion`, `TestResolvePeerOrderAndReplicatedIdentity` | ASCII-fold collisions refuse; same-record replicas deduplicate; divergent records refuse `integrity_failure`. Both identity/name agreement narrowing mutants are killed. |
| 4. List summaries | `Reader.List` → repository read → `Projector.Project` → `checkSummaryRepresentable` | `TestListStatusDerivedFactsAndStableOrder`, `TestListStatusCheckpointAndProjectionFailure`, `TestParkedReadRecoveryAndMissingVsMalformed`, `TestSummaryBootstrapRefusals`, `TestSummaryMixedListRefusesWhole`, `TestReviewerBootstrapSummaryMustRefuse`, `TestCreatingSummaryCannotClaimClosedCLIResult` | Derived identity/state/lease/owner/role/checkpoint and parked diagnostics are represented with stable bytewise session-ID order. Record-only bootstrap prefixes refuse the whole list; projection/read failures propagate. |
| 5. Status and authoritative summaries | `Reader.Status`; `Reader.AuthoritativeStatus`; `Reader.AuthoritativeList` → `authorize` | `TestAuthoritativeStatusHealthy`, `TestAuthoritativeMissingHostMetadataRefuses`, `TestAuthoritativeUnknownLocalHostRefuses`, `TestAuthoritativeCheckpointRequiresTimestamp`, `TestAuthoritativeInputValidation`, `TestAuthoritativeOptionalObservations`, `TestAuthoritativeUnknownObservationsRefuse`, `TestAuthoritativeMissingCapabilitiesRefuses`, `TestAuthoritativeMissingProcessRefusesStatus`, `TestAuthoritativeMissingWorkspaceRefuses`, `TestRev4UnknownCapabilityMustRefuse` | Required owner/lease/role/observation facts, seven-name capability vocabulary, established absence, host-name bounds, and refusal classes are driven. Observation and capability narrowing mutants are killed; no capability is advertised. |
| 6. Stable deterministic sorting | `Reader.read`, `peerCandidates`, `selectIdentity` | `TestListStatusDerivedFactsAndStableOrder`, `TestResolvePeerOrderAndReplicatedIdentity`, `TestResolveBareIdentityUnion` | Session IDs, peer-source candidates, and identical-copy host tie-breaks are deterministic. `B-peer-order`, `B-session-order`, and `B-tie-break` all fail their named behavioral tests. |
| 7. Immutable SelectionPlan build/revalidation | `Reader.BuildPlan` → `bindPlan`; `Reader.Revalidate` → `validateAuthorityUnion`/`collectAuthorityHeads`; `Reader.winningLeaseFor` → `checkLeaseChain`/shared checkpoint admission | `TestBuildPlanBindsCurrentFacts`, `TestBuildPlanPeerSourceFacts`, `TestBuildPlanValidatesUnionAtBuild`, `TestPlanDigestAndMarshalRoundTrip`, `TestRevalidateDetectsEachFactChange`, `TestRevalidateUnionLeaseDivergenceRefusesStale`, `TestRevalidateUnionTombstoneRefusesStale`, `TestRevalidateUnionParkedCopyRefuses`, `TestRevalidateReadFailures`, `TestReviewerPlanRequiresLeaseRecord`, `TestRev5AncestorCheckpointAuthority`, `TestRev5CheckpointCreatorMustBeHolder`, `TestRev5AncestorCheckpointBinding`, `TestRev6CheckpointPersistenceMustMatchSession`, `TestRev7CheckpointHeadAuthority`, `TestReview11ReferencedTemporalAuthority`, `TestRev12ReferencedCheckpointLaterLease`, `TestRev15AliasAwareRecordConsumptionCensus` | The plan binds the selector, UUID, record/lease facts, source/config/index digests, authority heads, action, destination, and expectation. Build and old-plan revalidation preserve pinned identity, lagging copies, parked-source refusal, ancestry checkpoint/session/lease/creator/variant/head relations, and no name re-resolution. |
| 8. Negative/refusal/recovery boundary | Shared resolver, plan, summary, and admission gates above; repository recovery remains owned by `sessrepo` | All refusal subtests in the named suites; `TestCreateSessionResumesInterruptedCreate`, `TestCreateSessionRefusesResumeWithDifferingBytes`, `TestListSessionsReportsTornHintOnUnreadableRecord` | Validation, authorization, attestation, ambiguity, not-found, source/read, integrity, bootstrap, observation, and stale-plan behavior is attacked by the applied N/B battery. Reads/revalidation perform no durable writes, so crash/idempotency evidence is not applicable to these entry points; retained repository recovery tests stay green. |

## Ratio and validation

- Shared API: **8 of 8 AC rows driven** by named candidate tests and the
  production call sites above.
- CLI/lifecycle: **0 of 8 delivered here**, an explicit ownership bound; this
  package has no `ax` executable, Result-5 renderer, transport, or lifecycle
  mutation entry point.
- Full local `go test ./... -count=1 -v`: exit 0.
- Full local `go test ./... -count=1 -cover`: exit 0.
- Full local `go test ./... -count=1 -race`: exit 0.
- `go vet ./...`, `go build ./...`, `gofmt -l internal`, `git diff --check`,
  and `go run ./internal/traceability/cmd/tracecheck`: exit 0.
- Evidence logs are in `.temp/TASK-260830-21gygk/`; the raw rev16 mutation
  output is `mutants-rev16b/`.
