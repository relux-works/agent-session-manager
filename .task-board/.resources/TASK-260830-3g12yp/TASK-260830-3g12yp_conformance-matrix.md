# TASK-260830-3g12yp conformance matrix

Authority: `relux-works/agent-session-manager-spec` v0.6.0, commit
`0cbdf100dbf84df50c64f792b1f940e3a67859a6`
(`internal/specdoc/SPEC.v0.6.0.md`). Each row maps one fencing clause
to the production entry and named test that discharges it, or to a
stated bound. Line numbers are 1-based into `SPEC.v0.6.0.md`.

## Section 2.2 Global invariants (lines 404-592)

| Clause (ID) | Requirement | Disposition | Production call site | Named test(s) |
| --- | --- | --- | --- | --- |
| 2.2#1 (408) | Exactly one winning lease, hence one logical owner | Driven | `fencing.Observe` + `fencing.Authorize` | `TestObserveLoadsWinningLease`, `TestAuthorizeAdmitsWinnerExactEpochAllOperations` |
| 2.2#2 (410) | A replica MUST NOT launch, resume, accept input, or publish authoritative events | Driven | `fencing.Authorize*` (remote arm) | `TestAuthorizeParksAndRefusesRemote`, `TestLifecycleCallersPassTheGate/owner_resume_and_replica_park` |
| 2.2#3 (412) | Every owner-authored event carries the winning epoch + lease ID | Driven | `fencing.AuthorizeMutation` | `TestAuthorizeAdmitsWinnerExactEpochAllOperations`, `TestLifecycleCallersPassTheGate/graceful_takeover_commit` |
| 2.2#4 (413) | Peers reject lower-epoch / losing same-epoch events, preserve divergent branch | Driven | `sessrepo.AppendEvent`, `sessstate.Projector.Project` | `TestAppendEventPreservesDivergentBranchWithoutApplying`, `TestAppendEventRefusesStaleLeaseEpoch`, `TestOldOwnerReconnectRejectedAfterForceTakeover/losing_events_preserved_without_application`, `/union_preserves_losing_branch` |
| 2.2#5-#22 | Replication, secret, SQLite, materialization, tombstone, profile, capability, bridge, sync, continuation, readiness, exclusion, directory-merge preamble | Stated bound | — | No implementation in this ownership story; registry gap on `section:2.2` discloses each family. |

## Section 5.3 Lease Record and ownership (lines 1908-1982)

| Clause (ID) | Requirement | Disposition | Production call site | Named test(s) |
| --- | --- | --- | --- | --- |
| 5.3#1 (1927) | `created_by_host_id` MUST equal `issued_by_host_id` | Driven (predecessor) | `canonicaljson.validateLeaseRecord` | `core-record-identity-validation` case tests |
| 5.3#2 (1957) | Epoch > 1 MUST name a known predecessor | Driven (predecessor + query) | `sessrepo.CompareAndSwapLease`, `sessquery.winningLeaseFor` | `TestCompareAndSwapPersistsSuccessor`, `TestSuccessorCycleMustRefuse`, `TestDanglingPredecessorMustRefuse`, `TestValidSuccessorBuildsAndRevalidates` |
| 5.3#3 (1958) | Epoch MUST equal predecessor + 1 and reference a validated checkpoint | Driven (predecessor + query) | `sessrepo.CompareAndSwapLease`, `sessquery.winningLeaseFor` | `TestEpochMonotonicityAcrossSuccessors`, `TestSkippedEpochMustRefuse`, `TestWrongPredecessorCheckpointMustRefuse`, `TestWrongSessionCheckpointMustRefuse` |
| 5.3#4 (1960) | Epoch-1 `create` MUST have a null predecessor | Driven (predecessor) | `sessrepo.CreateLease` | `TestCreateLeasePersistsEpochOneCreate`, `TestNormativeLeaseRecordExampleAttests` |
| 5.3#5 (1962) | New takeover lease MUST use `max_observed_epoch + 1` | Stated bound | — | The union maximum is caller-side input; the store mints head + 1 and no takeover flow exists yet. Registry gap on `section:5.3` discloses it. |
| 5.3#6 (1967), 5.3#7 (1968) | Losing/lower events preserved, never applied | Driven | `sessrepo.AppendEvent`, `sessstate.Reduce` | Same as 2.2#4, plus `TestWinnerResolutionLosingBranchPreserved`, `TestWinnerResolutionUnionSupersedesChain` |
| 5.3#8 (1970) | Owner MUST revalidate before input, turn, checkpoint, push, resume | Driven | `sessrepo.VerifyFencingToken`, `sessrepo.CheckFencingExpiry`, `fencing.Authorize*` | `TestVerifyFencingTokenAcceptsWinner`, `TestVerifyFencingTokenRefusals`, `TestCheckFencingExpiry`, `TestAuthorizeRefusesExpiredGrant`, all `TestAuthorize*` refusal tables |

## Sections 13.6-13.10 lifecycle fencing

| Clause | Requirement | Disposition | Production call site | Named test(s) |
| --- | --- | --- | --- | --- |
| 13.6 step 10 | Destination creates epoch `source+1` after stop/prepare; MUST NOT adopt/resume under a concurrent winner | Driven (gate half) | `fencing.AuthorizeMutation`, `fencing.AuthorizeActivation` | `TestLifecycleCallersPassTheGate/graceful_takeover_commit`; orchestration itself is a stated bound (modeled caller) |
| 13.6 failure table | No failure before step 10 advances ownership; none after lets the source resume | Driven (gate half) | `fencing.Authorize*` | `TestAuthorizeParksAndRefusesFailedHandoff`; orchestration is a stated bound |
| 13.7 force lease | Only the winning committed force lease authorizes runtime creation; its token rejects the prior owner | Driven | `fencing.AuthorizeActivation`, `fencing.AuthorizeInput` | `TestOldOwnerReconnectRejectedAfterForceTakeover`, `TestAuthorizeEndToEndOverRepository` |
| 13.7 old owner | Prior owner becomes stale; losing history preserved in divergent branches | Driven | `sessstate.Projector.Project`, `sessrepo.AppendEvent` | `TestOldOwnerReconnectRejectedAfterForceTakeover/union_preserves_losing_branch`, `/losing_events_preserved_without_application` |
| 13.7 same-epoch tie | Bytewise-greater lease ID wins; loser stops accepting input | Driven | `sessrepo.CompareLeaseTuple`, `sessstate.Compare`, `fencing.AuthorizeInput` | `TestConcurrentForceTakeoversDeterministicWinner`, `TestLeaseTupleOrderAgreesWithSessstateCompare` (144-pair enumeration) |
| 13.7 failure table | Losing lease before activation parks; post-commit failure never rolls ownership back | Driven (gate half) | `fencing.AuthorizeActivation` (`HandoffFailed` arm) | `TestAuthorizeParksAndRefusesFailedHandoff`; orchestration is a stated bound |
| 13.8 fork | Epoch-1 lease owned by the destination; source untouched | Driven (gate half) | `fencing.AuthorizeActivation` | `TestLifecycleCallersPassTheGate/fork_epoch_one_activation`; projection/materialization orchestration is a stated bound |
| 13.9 stop | Winning owner lease; checkpoint under the current lease | Driven (gate half) | `fencing.AuthorizeCheckpoint` | `TestLifecycleCallersPassTheGate/stop_checkpoint`; quiesce/stop orchestration is a stated bound |
| 13.10 resume | Allowed only on the winning owner; replica receives `not_owner`; lease revalidated first | Driven | `fencing.AuthorizeRestore` | `TestLifecycleCallersPassTheGate/owner_resume_and_replica_park`; materialization paths are a stated bound |
| 13.12 matrix | Old owner reconnect rejected with its lower/losing token | Driven | `fencing.AuthorizeInput`, `fencing.AuthorizeMutation` | `TestOldOwnerReconnectRejectedAfterForceTakeover/new_owner_rejects_old_token_as_stale` |

## Section 4 terminal wrapper rule

| Clause | Requirement | Disposition | Production call site | Named test(s) |
| --- | --- | --- | --- | --- |
| 4.1 wrapper (1369-1372) | Compare the local fencing token before launching; park when remote, ambiguous, or unverified | Driven | `fencing.AuthorizeRestore`, `fencing.AuthorizeActivation` | `TestAuthorizeParksAndRefusesRemote`, `TestAuthorizeParksAndRefusesUnverified`, `TestAuthorizeParksAndRefusesAmbiguous`, `TestAuthorizeParksAndRefusesAbsentWinner` |
| 4.1 terminate-stale (1366) | Terminate only for explicit force recovery after preserving diagnostics | Driven | `fencing.AuthorizeTerminateStale` | `TestTerminateStaleAuthorizesFencedTarget`, `TestTerminateStaleRefusesWithoutForce`, `TestTerminateStaleRefusesWithoutDiagnostics`, `TestTerminateStaleRefusesWithoutWinner`, `TestTerminateStaleRefusesLiveOwner` |
| 4.C row (1190) | `terminate-stale` error vocabulary incl. `local_precondition_failed` | Driven | `fencing.AuthorizeTerminateStale` | Same as above; class proven registered via `TestGateRefusalsUseRegisteredCodes` |
| 4.2/4.3/4.4 backends | tmux/ConPTY/user-service wrapper behavior | Stated bound | — | No terminal backend exists in this tree; the gates own the wrapper's ownership entry points only. |

## Section 7.5 LeaseToken

| Clause | Requirement | Disposition | Production call site | Named test(s) |
| --- | --- | --- | --- | --- |
| LeaseToken triple (2952) + quiesce/capture/materialize carriage | `{session_id, lease_epoch, lease_id}` minted from a passed gate only | Driven | `fencing.LeaseToken.Bind` | `TestBindProjectsMintedToken`, `TestBindRefusesForgedToken`, `TestMintedTokenIsStableAcrossEntries`, census tests |
| resume/stop/materialize-commit/native-store-plan carriage | Same triple on other operations | Stated bound | — | Identical wire shape, but binding without the operations' callers would be untestable. |

## Section 15 error classes

| Clause | Requirement | Disposition | Production call site | Named test(s) |
| --- | --- | --- | --- | --- |
| 15.3 exit 10 | `not_owner`, `stale_owner`, `lease_conflict` | Driven | All `fencing` refusals | `TestGateRefusalsUseRegisteredCodes` (constructs each through `axerror` with exit 10) |
| 15.3 exit 2/3 | `invalid_arguments`, `local_precondition_failed` | Driven | Malformed calls, terminate-stale misuse | Same test (exits 2 and 3); no new class is minted anywhere. |

## Crash/idempotency

The gates perform no durable writes (pure over durable inputs), so no
Section 13.13 seam exists. `TestGatesPerformNoDurableWrites` hashes the
repository tree, evaluates every entry passing and failing, and requires
byte-identical trees plus identical verdicts across two runs.
