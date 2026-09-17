# TASK-260830-21gygk rev15 producer outcome

## Candidate and authority

- Task: `TASK-260830-21gygk`; Story: `STORY-260830-3tq4ns`.
- Candidate is the uncommitted working tree at `HEAD=e119aaeede327580a470238fdb1ccf340a785b39`; the recorded Story checkpoint/base remains `8626fb361c1d6b62d5f95c7be4cb33c85d4aadf3`.
- Normative authority: `relux-works/agent-session-manager-spec@v0.6.0`, commit `0cbdf100dbf84df50c64f792b1f940e3a67859a6`; retained historical scope is v0.5.0, commit `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`.
- No commit was created. Foreign `.task-board` changes were preserved and are not in the product candidate.

## CR14 rework closed

The package-wide record-consumption census now checks semantic identity rather
than syntax alone. It rejects non-composite construction and mutation paths for
`checkpointSeal` and `checkpointSealToken`, including `new(T)`, generic
instantiation, explicit field writes, `any` copies, `reflect.TypeFor`, and
`unsafe.Pointer`. It resolves callback callees to exact `go/types` objects and
admits only the exact callback parameter object recorded for the approved
profile entry. A package callback variable and a struct-field callback both
type-check but are refused.

The temporal fixture has separate admitted controls. `good_prechange` points
the checkpoint at the pre-change head and uses the yolo/null creation pair;
`good_divergent` preserves a losing-lease `profile.changed` blob outside the
authoritative index while admitting the winning standard/changed pair. Both
are still driven through the shared entry points, including old-plan
revalidation and both authoritative summaries.

## Acceptance-criteria coverage

Shared-library coverage is **8 of 8 AC rows driven**. Each row reaches a
production entry point; validator-only calls are not counted.

| AC row | Production call site | Named evidence |
| --- | --- | --- |
| UUID selectors | `Reader.Resolve` → `resolveBare`/`selectIdentity` | `TestResolveLocalNameAndUUID`, `TestParseSelectorLiteralGrammar`, `TestResolveBareIdentityUnion` |
| Name selectors, exact precedence, and ambiguity | `Reader.Resolve` → `matchName`/`selectIdentity` | `TestResolveExactNamesAndASCIICollisions`, `TestResolveQualifiedNameBeforeUUIDInSource`, `TestResolveQualifiedAmbiguityAndExclusions`, `TestResolveBareIdentityUnion` |
| Qualified selectors and exact source mapping | `Reader.Resolve` → `parseSelector` → `resolveExplicit` → `locateSource` | `TestParseSelectorLiteralGrammar`, `TestResolveQualifiedSourcesSelectOneIndex`, `TestResolveQualifiedSourceNeverFallsBack`, `TestResolveExplicitSourceRefusals`, `TestResolveExplicitSourceReadFailures` |
| Tier, collision, tombstone, and cross-copy agreement | `matchName`, `selectIdentity`, `checkRecordAgreement` | `TestResolveExcludesTombstonedButListsIt`, `TestResolveBareIdentityUnion`, `N-identity-diverge`, `N-agreement-name`, `N-parked-id`, `N-tombstoned-id` |
| List summaries | `Reader.List` → repository read/projector → `checkSummaryRepresentable` | `TestListStatusDerivedFactsAndStableOrder`, `TestListStatusCheckpointAndProjectionFailure`, `TestSummaryMixedListRefusesWhole`, `TestSummaryBootstrapRefusals` |
| Status summaries and closed capabilities | `Reader.Status`, `Reader.AuthoritativeStatus`, `Reader.AuthoritativeList` | `TestAuthoritativeStatusHealthy`, `TestAuthoritativeMissingHostMetadataRefuses`, `TestAuthoritativeMissingProcessRefusesStatus`, `TestAuthoritativeMissingWorkspaceRefuses`, `TestAuthoritativeMissingCapabilitiesRefuses`, `TestRev4UnknownCapabilityMustRefuse` |
| Stable deterministic sorting | `Reader.read`, `peerCandidates`, `selectIdentity` | `TestListStatusDerivedFactsAndStableOrder`, `TestResolvePeerOrderAndReplicatedIdentity`, `TestResolveBareIdentityUnion`, `B-peer-order`, `B-session-order`, `B-tie-break` |
| Immutable plan, current-fact revalidation, semantic ancestry, and refusal behavior | `Reader.BuildPlan`, `Reader.Revalidate`, `winningLeaseFor`, `admitCheckpoint`, summary owners | `TestRevalidateDetectsEachFactChange`, `TestRev5AncestorCheckpointAuthority`, `TestRev5AncestorCheckpointBinding`, `TestRev5CheckpointCreatorMustBeHolder`, `TestReview11ReferencedTemporalAuthority`, `TestRev14CapabilitySeal`, census and mutation evidence |

Public CLI/lifecycle coverage is **0 of 8 AC rows**, by stated ownership
bound. This package owns the shared API, not an `ax` executable, CLI Result
renderer, transport, fencing, or lifecycle mutation caller. No read/plan/
summary operation mutates durable state, so crash/idempotency evidence is not
applicable here; repository recovery/idempotency tests remain in the full
suite.

## Gate and mutant evidence

- `TestRev14RecordConsumptionCensusRejectsAlternatePaths` rejects interface,
  closure, unresolved, method-name collision, field assembly, embedding, raw
  copy, map, non-composite seal, generic, `any`, reflect, and unsafe plants.
- `TestRev14CallbackProvenanceRejectsUnknownFunctions` rejects both the
  package-level callback variable and struct-field callback. The errors are
  produced by the admitted-result/copy flow, not by a type-check failure.
- The final `mutate.py` run exited 0. `mutants.json` contains **65 of 65
  applied N/B plants killed**: 62 narrowing and 3 ordering. The new
  `N-census-unknown-callback` preserves the callback shape and is killed by
  the exact `unknown_callback.go` behavioral subtest. Controls are separate:
  `control-before=CONTROL/0`, `control-after=CONTROL/0`,
  `C-harmless-comment=SURVIVED/0`, `C-not-applied=NOT_APPLIED`, and
  `C-compile-failure=COMPILE_OR_HARNESS_FAILURE/1`. No control is counted as
  a behavioral kill.
- The mutation harness restores every copied source and verifies its SHA-256
  against the copied baseline before writing `mutants.json`.

## Validation

All commands below ran on this final uncommitted candidate and exited 0 unless
explicitly classified above.

| Command | Exit | Evidence |
| --- | ---: | --- |
| `go test ./internal/sessquery -run '^(TestRev14RecordConsumptionCensusRejectsAlternatePaths|TestRev14CallbackProvenanceRejectsUnknownFunctions|TestReview11ReferencedTemporalAuthority|TestRev12ReferencedCheckpointLaterLease)$' -count=1 -v` | 0 | `census-temporal-green.log` |
| `python3 -m py_compile internal/sessquery/testdata/mutate.py` | 0 | command result |
| `python3 internal/sessquery/testdata/mutate.py .temp/TASK-260830-21gygk/rework-rev15/mutants` | 0 | `mutants/mutants.json` and per-plant logs |
| `go test ./... -v` | 0 | `01-go-test-all-v.log` |
| `go test ./... -cover` | 0 | `02-go-test-all-cover.log`; sessquery 86.9%, sessrepo 87.2%, tracecheck command package 88.5% |
| `go test ./... -race` | 0 | `07-go-test-all-race.log` |
| `go vet ./...` | 0 | `03-go-vet.log` |
| `go build ./...` | 0 | `04-go-build.log` |
| `gofmt -l internal/sessquery internal/sessrepo internal/provhost` | 0, empty output | `05-gofmt-check.log` |
| `git diff --check` | 0 | `06-git-diff-check.log` |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | `08-tracecheck.log` |
| `curator status --check` | 0 | `09-curator-status.log` |

After the README/LOGBOOK evidence update, `gofmt -l` remained empty and
`git diff --check` remained clean (`10-gofmt-check-after-docs.log`,
`11-git-diff-check-after-docs.log`).

The full suite also ran the traceability tests, including the production
tracecheck command package. No unsupported provider, platform, doctor, CLI,
transport, or durable-write capability is advertised.

## Final-source provenance

| File | SHA-256 |
| --- | --- |
| `internal/sessquery/lease.go` | `3ce1956872db85b39105bab1970eaf757b19d05d0753482bdba449d52d68ec2f` |
| `internal/sessquery/rev11_regression_test.go` | `cdeb0a9e9708018d2504a22144e26f95e25441892727b8fcd5327acacbcd611d` |
| `internal/sessquery/rev14_regression_test.go` | `f9532d8e939fb09a3377a9418d96426fb6e8e55184f900b4cec0661cd2cb42a1` |
| `internal/sessquery/testdata/mutate.py` | `5225bfa8e582fa7398b6e3f195450dab3fc3a0f3a406f7b52fa3d6ae89452da2` |
| `internal/sessquery/TRACEABILITY.md` | `36ee630e3b8a020da594c4c1f3aa139e18c16a2f461cb4fcd1d914bcdf7568cc` |

The Story worktree remains uncommitted for the normal task-board snapshot and
independent review handoff.
