# TASK-260830-2f5393: implementation evidence

Authority: relux-works/agent-session-manager-spec v0.6.0, commit
`0cbdf100dbf84df50c64f792b1f940e3a67859a6`. Primary scope is Sections
2.2, 5.3, and 13.6-13.10 (lease creation, renewal, expiry policy, epoch
monotonicity, holder identity, compare-and-swap writes). Historical
scope is retained without weakening: v0.5.0 (commit
`28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`), Sections 2.3, 5.1-5.2,
5.7, 14.4. This file records implementation evidence; it changes no
normative ownership and claims no public CLI delivery.

The lease store (`lease_store.go`) extends this package with the durable
Lease Record 1.0.0 lifecycle. It mints no second lease model: closed
shape, self identity, and the three Section 5.3 couplings stay owned by
`canonicaljson`; the winning-tuple rule stays owned by `sessstate`
(`CompareLeaseTuple` restates it because `sessstate` imports this
package, and a cross-package agreement test pins identical order); and
query-layer admission (`sessquery` `winningLeaseFor`) is unchanged — an
adoption test proves store-minted records admit there with the same
digest, epoch, lease ID, and holder.

## Acceptance rows

8 of 8 AC rows are driven at the production entries named below; 0 of 8
are delivered as a public CLI surface, which is an explicit
caller-integration bound (no `ax` session command exists in this tree),
not a gap in the assigned shared behavior. Takeover/fork/stop/resume
orchestration, checkpoint admission, and cross-partition union
convergence belong to their owning leaves, which invoke this API at
every required boundary.

| AC row | Production call site | Named test(s) | Evidence and bound |
| --- | --- | --- | --- |
| Lease creation | `Repository.CreateLease` | `TestCreateLeasePersistsEpochOneCreate`, `TestCreateLeaseRequiresExistingSession`, `TestCreateLeaseRefusesInvalidIdentity`, `TestCreateLeaseIsIdempotentOnByteIdenticalRetry`, `TestCreateLeaseRefusesSecondCreateWithDifferingBytes`, `TestNormativeLeaseRecordExampleAttests` | Driven: epoch-1 `create` minted from the durable bootstrap inputs (existing session record plus caller fencing token, holder, issuer, timestamp), attested through the canonical owner with null predecessor/checkpoint; unknown session refuses; malformed members refuse `invalid lease record`; byte-identical retry replays the same reference; a differing second create refuses `lease already exists` with the persisted lease undisturbed. The verbatim Section 5.3 example attests to its pinned digest. |
| Renewal | `Repository.VerifyFencingToken` | `TestVerifyFencingTokenAcceptsWinner`, `TestVerifyFencingTokenRefusals` | Driven: a token matching the winning tuple and holder revalidates (nil); epoch 0, malformed lease/holder refuse `invalid lease record`; a superseded epoch refuses `stale lease epoch`; a beyond-head epoch (including any token on an empty store) refuses `unknown lease`; a same-epoch foreign lease refuses `lease_conflict`; a foreign holder refuses `lease_holder_mismatch`. Renewal mints no lease and mutates no durable state. |
| Expiry policy | `CheckFencingExpiry`; `Repository.VerifyFencingToken` + `Repository.WinningLease` | `TestCheckFencingExpiry`, `TestLeaseNeverExpiresWithAge` | Driven: a grant older than the refresh interval refuses `fencing_grant_expired` with revalidation as the remedy; a grant at exactly the boundary stays current; a non-positive interval refuses `invalid lease record`. The lease itself never expires: an ancient `created_at` still verifies and still wins (Section 5.3 "no time-expiring ownership lease"; liveness is not authority). Grants are process-local and never persisted. |
| Epoch monotonicity | `Repository.CompareAndSwapLease` → head+1 derivation; `Repository.ListLeases` order | `TestCompareAndSwapPersistsSuccessor`, `TestEpochMonotonicityAcrossSuccessors`, `TestCompareAndSwapIsIdempotentOnByteIdenticalRetry` | Driven: every successor lands at exactly head epoch plus one with the head fencing token as predecessor — the epoch is never taken from input, so it can neither skip nor decrease; three successions pin epochs 2-4 with a linked predecessor chain; list order is ascending tuple order. A superseded expectation refuses stale (or replays the persisted successor on a byte-identical retry). Epoch gaps and cycles cannot be minted here; out-of-band ones still refuse at query admission. |
| Holder identity | `Repository.CreateLease`, `Repository.CompareAndSwapLease`, `Repository.VerifyFencingToken` | Holder arms of `TestCreateLeaseRefusesInvalidIdentity`, `TestCompareAndSwapRefusesInvalidSuccessor`, `TestVerifyFencingTokenRefusals` (`holder mismatch`), `TestCompareAndSwapPersistsSuccessor` | Driven: holder and issuer must be UUIDv7 host identities; the successor binds its holder; a token from a non-holder refuses `lease_holder_mismatch`. Process binding is machine-local and never persisted cross-host (Section 2.2 invariants 4-5). |
| Compare-and-swap writes | `Repository.CompareAndSwapLease` | `TestCompareAndSwapPersistsSuccessor`, `TestCompareAndSwapRequiresExistingLease`, `TestCompareAndSwapRefusesInvalidSuccessor`, `TestCompareAndSwapRefusesFencingTokenReuse`, `TestCompareAndSwapRefusesStaleExpectation`, `TestCompareAndSwapRefusesUnknownExpectation`, `TestCompareAndSwapIsIdempotentOnByteIdenticalRetry` | Driven: an expectation naming the head persists the successor; an expectation naming nothing known (malformed, empty, foreign) refuses `lease_conflict`; a superseded expectation refuses `stale lease epoch`; CAS on an empty store refuses `unknown lease`; a reused fencing token refuses `invalid lease record`; a non-enum reason or absent/malformed checkpoint refuses `invalid lease record`; a byte-identical retry re-mints against its pre-commit basis and replays the persisted successor. Full checkpoint-record admission stays with the query layer (stated bound). |
| Contract fixtures and negative/refusal cases | All gates below; `AttestLeaseRecord` | Every `Test*` arm above plus `TestStoredLeaseCorruptionRefuses`, `TestInstallDisagreeingBytesRefuses`, `TestGetLeaseRefusals`, `TestWinningLeaseRefusesEmptyStore`, `TestLeaseStageTempsAreIgnored`, `TestLeaseTupleOrderAgreesWithSessstateCompare`, `TestLeaseStoreRecordsAdmitWithoutBehavioralChange` | Driven: the normative example attests; torn blobs (unparseable names, undecodable/truncated bytes, cross-session plants, digest-name disagreement) refuse chain corruption through both funnels; disagreeing bytes at install refuse (same- and different-length arms); malformed/unknown reads refuse their classes; orphaned stage temps are ignored; the tuple rule agrees with `sessstate.Compare` on a pinned corpus; store-minted records admit through unchanged `winningLeaseFor`/`BuildPlan`/`Revalidate`. |
| Crash/idempotency evidence | `Repository.CreateLease`, `Repository.CompareAndSwapLease` + `BeforeWrite`/`AfterCommit`/`AfterLeaseStage` | `TestCrashBeforeLeaseWriteIsSafeRetry`, `TestCrashAfterLeaseCommitCountsAsCommitted`, `TestLeaseKillAtRenameSeam/create`, `TestLeaseKillAtRenameSeam/cas` | Driven: an injected pre-write fault carries `safe_retry` with nothing mutated and the retry succeeding; an injected post-commit fault carries `recoverable_parked_state` with the commit durable and the byte-identical retry replaying the same reference. A real SIGKILLed child at the stage-then-rename seam leaves no torn final (create: zero finals plus one ignored temp; CAS: the prior final plus one ignored temp); a fresh handle reopens cleanly and the byte-identical retry lands the exact epoch. See the recovery table below. |

## Refusal and recovery coverage

Every `N-` row is a genuine narrowing mutant: the gate stays present
and is weakened to admit exactly one member of the class it must
reject, and the named behavioral test fails through the delivered
harness (`internal/sessrepo/testdata/mutate.py`, run as
`python3 internal/sessrepo/testdata/mutate.py <evidence-dir>`).
Whole-clause disables are not accepted as narrowing. The `T-` row
preserves the searched-for timestamp token while changing behavior.
Controls are reported separately and never counted as kills.

Final battery (`mutants-03`, on the exact final source): 26 narrowing
(one per `refuse` site) + 1 token-preserving, all `KILLED`; the applied
harmless control is `SURVIVED`, while the not-applied and
compile-failure controls are classified separately. Battery exit 0.

| Gate | Real entry test | Narrowing attack (all killed) |
| --- | --- | --- |
| Create fencing-token grammar | `TestCreateLeaseRefusesInvalidIdentity/lease_token` | N-create-token admits exactly `not-a-uuid`; mint then fails closed with a plain error and the refusal class changes. |
| Create holder grammar | `TestCreateLeaseRefusesInvalidIdentity/holder` | N-create-holder admits exactly `not-a-uuid`; same closed mint failure. |
| Create issuer grammar | `TestCreateLeaseRefusesInvalidIdentity/issuer` | N-create-issuer admits exactly the Q-suffixed issuer; same closed mint failure. |
| Create timestamp grammar | `TestCreateLeaseRefusesInvalidIdentity/created_at`, `/created_at_padded` | N-create-time admits exactly `not-a-timestamp` (mint succeeds: no downstream check); T-create-time-pad parses the same token whitespace-tolerantly and admits the padded value. |
| Successor reason enum | `TestCompareAndSwapRefusesInvalidSuccessor/reason` | N-successor-reason admits exactly `bogus_reason`; mint fails closed and the refusal class changes. |
| Successor checkpoint grammar | `TestCompareAndSwapRefusesInvalidSuccessor/checkpoint_malformed` | N-successor-checkpoint admits exactly `not-a-digest`; absent and other malformed values still refuse. |
| Create-once | `TestCreateLeaseRefusesSecondCreateWithDifferingBytes` | N-create-once admits exactly a depth-one second create; a third create past a succession still refuses. |
| Fencing-token uniqueness | `TestCompareAndSwapRefusesFencingTokenReuse` | N-cas-token-reuse admits exactly the epoch-1 token reuse; the CAS succeeds and the negative fails. |
| CAS conflict | `TestCompareAndSwapRefusesUnknownExpectation/foreign` | N-cas-conflict admits exactly the foreign digest; replay minting fails closed and the refusal class changes. |
| CAS stale basis | `TestCompareAndSwapRefusesStaleExpectation` | N-cas-stale admits exactly epoch-1 stale bases; the epoch-2 arm still refuses. |
| CAS empty store | `TestCompareAndSwapRequiresExistingLease` | N-cas-empty admits exactly the empty-expectation CAS; a named expectation on an empty store still refuses. |
| Fencing token grammar (epoch/lease/holder) | `TestVerifyFencingTokenRefusals/epoch_zero`, `/lease_malformed`, `/holder_malformed` | N-fence-token-epoch/lease/holder admit exactly the suite vectors; each refusal degrades to its downstream class (stale/conflict/mismatch). |
| Fencing stale epoch | `TestVerifyFencingTokenRefusals/stale_epoch` | N-fence-stale admits exactly the epoch-1 token; the refusal degrades to conflict. |
| Fencing future epoch | `TestVerifyFencingTokenRefusals/future_epoch` | N-fence-future admits exactly epoch 9; the empty-store epoch-1 token still refuses unknown. |
| Fencing same-epoch tie | `TestVerifyFencingTokenRefusals/losing_lease` | N-fence-loser admits exactly the leaseC loser; the token verifies and the negative fails by success. |
| Fencing holder binding | `TestVerifyFencingTokenRefusals/holder_mismatch` | N-fence-holder admits exactly the hostA non-holder; the token verifies and the negative fails by success. |
| Grant-expiry policy shape | `TestCheckFencingExpiry` | N-expiry-policy admits exactly the zero interval; a negative interval still refuses. |
| Grant-expiry lapse | `TestCheckFencingExpiry` | N-expiry-lapsed admits exactly the suite lapsed grant; the grant verifies and the negative fails by success. |
| Lease load funnel | `TestStoredLeaseCorruptionRefuses/unparseable_name` | N-load-funnel admits exactly the garbage-name vector; truncated, cross-session, misnamed, and undecodable vectors still refuse. |
| GetLease funnel | `TestStoredLeaseCorruptionRefuses/undecodable_bytes` | N-get-funnel admits exactly the undecodable vector; the misnamed vector still refuses. |
| Install content equality | `TestInstallDisagreeingBytesRefuses/same_length` | N-install-length compares length only at the ledgered site; the same-length garbage arm reuses while the different-length arm still refuses. |
| GetLease malformed reference | `TestGetLeaseRefusals` | N-get-malformed admits exactly `not-a-digest`; the refusal degrades to unknown. |
| GetLease unknown reference | `TestGetLeaseRefusals` | N-get-unknown admits exactly the foreign digest; the refusal degrades to corruption. |
| Winning-lease absence | `TestWinningLeaseRefusesEmptyStore` | N-winning-empty admits exactly the suite empty store with a zero summary; the negative fails by success. |

Crash/idempotency matrix (Section 13.13 outcomes):

| Boundary | Required recovery | Evidence |
| --- | --- | --- |
| Before first lease byte (`CR-MAT-prepare-enter`) | `safe_retry`: nothing mutated, identical retry succeeds | Injector arms on create and CAS; kill drill shows zero premature finals. |
| After staged fsync, before rename (`AfterLeaseStage`) | `safe_retry`: only an orphaned ignored temp, identical retry lands the exact epoch | Real SIGKILL child drills for create and CAS with fresh-handle reopen and temp-tolerance assertions. |
| After install (`CR-MAT-commit-apply`) | Committed: the lease reads back, the byte-identical retry replays the same reference (CAS re-mints against its pre-commit basis) | Injector arms on create and CAS. |
| Disagreeing bytes at a digest path | Torn store: no retry heals; operator removes the session directory | Same- and different-length plant arms; post-plant reads funnel chain corruption. |

## Bounds (not silent gaps)

- No public CLI: 0 of 8 rows are delivered as an `ax` command. The
  shared-library entries above are the deliverable; orchestration leaves
  invoke them.
- No checkpoint admission on the write path: CAS requires a well-formed
  checkpoint digest, but validating the referenced Checkpoint Record
  stays with `sessquery` admission.
- No cross-partition union convergence: same-epoch concurrent leases
  from partitioned writers reconcile in the sibling ownership leaves;
  this store orders whatever blobs exist by the shared tuple rule.
- No wall-clock lease expiry and no persisted grants: expiry retires
  only the process-local authorization; the lease stays authoritative.
- Mint failures are plain operational errors, never refusals: entry
  validation already admitted the inputs, so a mint failure is an
  internal inconsistency. The narrowing batteries rely on this: a
  weakened grammar gate fails closed at mint with a changed class.

## Story-final BUG-260917-3lddu0: winning-lease append admission

This leaf adds the owner-side admission gate in
`Repository.AppendEvent`: once a session has a lease store, a new event
must be at the winning epoch and, at that epoch, must name the winning
lease. The gate is `checkWinningLease` in `chain.go`; historical replay
does not call it, because an earlier winner's already-authoritative event
remains valid history after a successor wins. A refused lower-epoch or
same-epoch losing event is still installed as an immutable blob, while
the authoritative chain is unchanged.

The two leaf acceptance rows are fully driven: **2 of 2**. The profile-source
row is route (b): it is closed by the append gate at
`internal/sessrepo/sessrepo.go:AppendEvent`. `sessprofile.Derive` is
unchanged, cannot see the lease store, and is recorded as a bound rather than
an independent losing-lease guarantee. Independent lease-aware derivation-side
authority is owned by `STORY-260922-cpkajd` / `TASK-260922-31qyyi`
(`lease-aware-profile-source-authority` /
`derivation-side-profile-source-gate`), including the higher-epoch and
empty-lease-store classes disclosed below.

| Gate arm | `Repository.AppendEvent` | `sessprofile.Transactor.SetProfile` | `axpane.Emit` | `axpane.EmitParked` | `Run → EmitParked` |
| --- | --- | --- | --- | --- | --- |
| Lower epoch → `ErrStaleLease` | `TestAppendEventRefusesSupersededLeaseWhileTailStillMatches` | `TestSetProfileRefusesSupersededLeaseWhileTailStillMatches/lower epoch` | `TestEmitReachesAppendAdmissionGateAfterStaleObservation` | `TestEmitParkedReachesAppendAdmissionGateAfterStaleObservation` | Bound: unreachable except by an interleaving between `Observe` and `AppendEvent`, closed by the durable gate. Owner `internal/axpane/run.go:Run` + `internal/sessrepo/sessrepo.go:AppendEvent`. |
| Same-epoch loser → `ErrDivergentBranch` | `TestAppendEventRefusesSameEpochLosingLeaseWhileTailStillMatches` (both loser IDs) | `TestSetProfileRefusesSupersededLeaseWhileTailStillMatches` (both loser-ID subtests) | `TestEmitReachesSameEpochAppendAdmissionGate` (both loser IDs) | `TestEmitParkedReachesSameEpochAppendAdmissionGate` (both loser IDs) | Bound: unreachable except by an interleaving between `Observe` and `AppendEvent`, closed by the durable gate. Owner `internal/axpane/run.go:Run` + `internal/sessrepo/sessrepo.go:AppendEvent`. |

The gate-entry census is **8 of 10 cells driven** and **2 of 10 cells
bounded**. Measured row symmetry is lower epoch **4 of 4** and same-epoch
loser **4 of 4** across the four reachable writer entries; the Run row is
bounded on both arms. Every reachable entry has a named test and a matching
narrowing row. The same-epoch tests cover both loser-ID directions,
`cccccccc-dddd-4eee-8fff-111111111111` and
`11111111-2222-4333-8444-555555555555`, and direct plus axpane paths
read back the preserved blob.

The complete importer comparison uses the exact mask
`./internal/axpane ./internal/crashgate ./internal/fencing
./internal/sessckpt ./internal/sessprofile ./internal/sessquery
./internal/sessstate ./internal/termbind ./internal/terminstance`:
all 9 packages pass, with 2,018 before and 2,030 after test-status keys
(including 9 package rows, or 2,009 / 2,021 actual test rows), 12 additions,
0 removals, and 0 changed statuses. The adopted reviewer runtime grid over
the same importer set records 130 outcome keys / 2,926 calls before versus
137 / 2,955 after, with 10 added and 3 removed outcome keys. Stable-name
fixture meaning changes are recorded in the task matrix and axpane
TRACEABILITY; the status grid cannot report those semantic moves.

The final append-admission narrowing battery was run twice from the exact
candidate source, under `mutants-append-rev3-pass1/` and
`mutants-append-rev3-pass2/`. Both true mutants retain the gate and
admit exactly one reachable rejected class; both are **KILLED** on both runs.
`N-stale-winning-admission` retains the gate but admits only the
reachable lower-epoch A member; `N-same-epoch-winning-admission` retains
the gate but admits only reachable loser C. The direct, SetProfile, Emit,
and EmitParked tests kill both true mutants. The harmless comment control is
`SURVIVED`, the missing-token control is `NOT_APPLIED`, and the
syntax control is `COMPILE_OR_HARNESS_FAILURE`; controls are not kills.
No source-text-inspecting gate exists, so a token-preserving source-text
mutant is not applicable.

Bounds: an empty lease store skips the winner gate and can admit valid
`(epoch 7, lease B)` sequence 1; owner
`internal/sessrepo/sessrepo.go:AppendEvent` / `checkWinningLease` plus
the lease lifecycle caller/store. An unknown higher epoch remains admitted
by `checkWinningLease` and may become a profile source; owner
`internal/sessrepo/chain.go:checkWinningLease` composed by
`AppendEvent`. That is not independent Section 2.4 ambiguity closure.
The sibling profile test `TestLosingLeaseProfileEventIgnored` drives the
same append gate and then checks `sessprofile.Derive` and `Run`:
the losing event is refused, `standard` with no source remains effective,
and the yolo/bypass mapping cannot launch. No public `ax` CLI entry
exists in this repository.
