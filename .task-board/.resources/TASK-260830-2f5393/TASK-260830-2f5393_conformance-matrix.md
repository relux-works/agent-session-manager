# TASK-260830-2f5393 conformance matrix — lease lifecycle vs pinned clauses

Authority: relux-works/agent-session-manager-spec v0.6.0
(`0cbdf100dbf84df50c64f792b1f940e3a67859a6`).

| Pinned clause | Owner (production) | Named test(s) | Verdict |
| --- | --- | --- | --- |
| 2.2 inv 1: exactly one winning lease per non-tombstoned session | `Repository.WinningLease` (greatest tuple, derived per read) | `TestCreateLeasePersistsEpochOneCreate`, `TestCompareAndSwapPersistsSuccessor`, `TestEpochMonotonicityAcrossSuccessors` | Driven |
| 2.2 inv 3: reject lower-epoch/losing same-epoch from authority | `Repository.VerifyFencingToken` (stale/conflict arms) | `TestVerifyFencingTokenRefusals/stale_epoch`, `/losing_lease` | Driven |
| 2.2 inv 4-5: no cross-host machine-local identity; no PIDs/locks replicated | Holder is host UUIDv7 only; `FencingGrant` never persisted | `TestVerifyFencingTokenRefusals/holder_mismatch`, `TestCheckFencingExpiry` | Driven (identity); grants non-persisted by construction |
| 2.2 inv 15: readiness before ownership; winning lease fences prior owner | CAS compare + fencing renewal (takeover orchestration in sibling leaves) | `TestCompareAndSwapRefusesUnknownExpectation`, `TestVerifyFencingTokenAcceptsWinner` | Shared behavior driven; takeover orchestration is a stated bound |
| 5.3 Lease Record 1.0.0 closed shape + couplings | `canonicaljson` (owner); minted via `mintLeaseRecord`, decoded via `decodeStoredLease` | `TestNormativeLeaseRecordExampleAttests`, create/successor shape assertions, `TestStoredLeaseCorruptionRefuses` | Driven (attested, never re-decoded here) |
| 5.3 winning tuple `(epoch, lease_id)`, bytewise tie-break | `CompareLeaseTuple` (+ `sessstate.Compare` owner) | `TestCompareLeaseTupleOrdersByGreatestTuple`, `TestLeaseTupleOrderAgreesWithSessstateCompare` | Driven |
| 5.3 epoch>1 names known predecessor at epoch+1 + validated checkpoint | CAS derivation (predecessor = head token, epoch = head+1, checkpoint digest required) | `TestCompareAndSwapPersistsSuccessor`, `TestEpochMonotonicityAcrossSuccessors`, checkpoint arms of `TestCompareAndSwapRefusesInvalidSuccessor` | Driven; full checkpoint-record admission stays with `sessquery` (bound) |
| 5.3 epoch-1 create: null predecessor, may null checkpoint | `CreateLease` (fixed reason, nulls) | `TestCreateLeasePersistsEpochOneCreate` | Driven |
| 5.3 `max_observed_epoch + 1` for new takeover leases | CAS head+1 derivation | `TestEpochMonotonicityAcrossSuccessors` | Driven within one store; cross-partition union is a sibling bound |
| 5.3 losing-lease history preserved, never applied | Immutable blobs; winner-only reads | `TestCompareAndSwapRefusesStaleExpectation` (losers persist, winner stands) | Driven |
| 5.3 fencing revalidation before input/turn/checkpoint/push/resume-after-gap | `VerifyFencingToken` (renewal) + `CheckFencingExpiry` (refresh interval) | `TestVerifyFencingTokenAcceptsWinner`, `TestVerifyFencingTokenRefusals`, `TestCheckFencingExpiry` | Driven; call-site wiring belongs to the owning leaves (bound) |
| 5.3 no time-expiring ownership lease; liveness ≠ authority | Non-expiry by construction; grants (not leases) lapse | `TestLeaseNeverExpiresWithAge` | Driven (pin) |
| 13.6 step 10: epoch source+1, predecessor source lease, checkpoint | `CompareAndSwapLease` semantics | `TestCompareAndSwapPersistsSuccessor` | Shared behavior driven; handoff orchestration is a bound |
| 13.6 post-commit: old source must not resume | Fencing renewal refuses superseded tokens | `TestVerifyFencingTokenRefusals/stale_epoch` | Driven (token half); process parking is a sibling bound |
| 13.7 step 6: `max_observed_epoch + 1`, random lease ID, reason `force_takeover` | CAS derivation + reason enum admission | `TestEpochMonotonicityAcrossSuccessors`, reason arms | Driven (mechanics); confirmation/receipt flow is a bound |
| 13.7 same-epoch tie → greater lease ID wins; loser stops input | Tuple order + losing-token refusal | `TestLeaseTupleOrderAgreesWithSessstateCompare`, `TestVerifyFencingTokenRefusals/losing_lease` | Driven (decision); process stop is a sibling bound |
| 13.7 `lease_conflict` refusal class | `ErrLeaseConflict` (`lease_conflict`) | `TestCompareAndSwapRefusesUnknownExpectation`, `TestVerifyFencingTokenRefusals/losing_lease` | Driven |
| 13.7/13.9/13.10 failure tables: never roll ownership back; no epoch without commit | No delete/rewrite entry; CAS only advances | Idempotency + monotonicity suites | Driven (no rollback entry exists) |
| 13.8 steps 1-3: epoch-1 lease for the new session | `CreateLease` | `TestCreateLeasePersistsEpochOneCreate` | Driven (record half); fork orchestration is a bound |
| 13.10 replica receives `not_owner`; owner verifies fencing token | Holder-mismatch refusal + renewal | `TestVerifyFencingTokenRefusals/holder_mismatch`, `TestVerifyFencingTokenAcceptsWinner` | Driven (token half; the `not_owner` resume surface is a bound) |
| 13.10 no null checkpoint past epoch 1 | Successor checkpoint-required gate | `TestCompareAndSwapRefusesInvalidSuccessor/checkpoint_absent` | Driven |
| 13.13 `safe_retry` / `recoverable_parked_state` outcome gate | Injector hooks + replay/idempotent retries | `TestCrashBeforeLeaseWriteIsSafeRetry`, `TestCrashAfterLeaseCommitCountsAsCommitted`, `TestLeaseKillAtRenameSeam/create`, `/cas` | Driven |

No unsupported capability is advertised: no CLI, no doctor result, no
checkpoint admission, no union convergence, no wall-clock lease expiry.
