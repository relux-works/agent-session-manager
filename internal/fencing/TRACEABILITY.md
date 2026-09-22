# TASK-260830-3g12yp: implementation evidence

Authority: relux-works/agent-session-manager-spec v0.6.0, commit
`0cbdf100dbf84df50c64f792b1f940e3a67859a6`. Primary scope is Sections
2.2, 5.3, and 13.6-13.10 (winning lease and exact epoch before
provider activation, input, mutation, checkpoint, or terminal
restore), the Section 4 terminal wrapper rule, and the Section 7.5
`LeaseToken`. This file records implementation evidence; it changes no
normative ownership and claims no public CLI delivery.

The fencing gates (`internal/fencing`) require the winning lease and
the exact epoch before every activation-class action. They mint no
second lease model: the Lease Record lifecycle stays owned by
`sessrepo` (`Observe` loads the winner through `ListLeases`, the
expiry arm calls `CheckFencingExpiry`); the winning-tuple rule stays
owned by `sessstate` (the gates compare one presented token for exact
equality with the one winning summary and never order two leases);
takeover, fork, stop, and resume transactions, terminal backends, and
provider plugin processes are modeled as callers, not members.

## Acceptance rows

18 of 18 AC rows are driven at the production entries named below; 0
of 18 are delivered as a public CLI surface, which is an explicit
caller-integration bound (no `ax` session command exists in this
tree), not a gap in the assigned shared behavior. Takeover/fork/stop/
resume orchestration belongs to its owning leaves, which invoke these
entries at every required boundary.

| AC row | Production call site | Named test(s) | Evidence and bound |
| --- | --- | --- | --- |
| Provider activation requires winner + exact epoch | `AuthorizeActivation` | `TestAuthorizeAdmitsWinnerExactEpochAllOperations`, `TestAuthorizeEndToEndOverRepository`, `TestLifecycleCallersPassTheGate/fork_epoch_one_activation` | Driven: the winning lease with the exact epoch mints through the activation entry over pure and repository-backed observations, including the fork epoch-1 lease; a rival succession parks or refuses the old token. |
| Provider input requires winner + exact epoch | `AuthorizeInput` | `TestAuthorizeAdmitsWinnerExactEpochAllOperations`, `TestOldOwnerReconnectRejectedAfterForceTakeover/new_owner_rejects_old_token_as_stale`, `TestConcurrentForceTakeoversDeterministicWinner` | Driven: exact input authorizes; the old token after a force takeover refuses `stale_owner`; the losing same-epoch token refuses `lease_conflict` once the union winner is known. |
| Owner-authored mutation requires winner + exact epoch | `AuthorizeMutation` | `TestAuthorizeAdmitsWinnerExactEpochAllOperations`, `TestLifecycleCallersPassTheGate/graceful_takeover_commit`, `TestAuthorizeEndToEndOverRepository` | Driven: the modeled graceful-takeover commit authorizes its mutation with the new winning tuple; the source token refuses after the succession. |
| Checkpoint capture requires winner + exact epoch | `AuthorizeCheckpoint` | `TestAuthorizeAdmitsWinnerExactEpochAllOperations`, `TestLifecycleCallersPassTheGate/stop_checkpoint` | Driven: the modeled stop checkpoint authorizes with the exact tuple and refuses `stale_owner` for a superseded token. No second checkpoint admission exists in this tree (the brief's `sessckpt` package is absent), so the gate is the capture admission (stated bound). |
| Terminal restore requires winner + exact epoch | `AuthorizeRestore` | `TestAuthorizeAdmitsWinnerExactEpochAllOperations`, `TestLifecycleCallersPassTheGate/owner_resume_and_replica_park` | Driven: the owner resume authorizes; the replica restore parks `remote_owner` with the `not_owner` cause instead of starting a runtime. |
| Lower epoch refused | `Authorize` via all five entries | `TestAuthorizeRefusesLowerEpoch` | Driven: a superseded epoch refuses `stale_owner` on input/mutation/checkpoint and parks `stale_owner` on activation/restore, for the standard old-lease vector and the winner-lease-at-lower-epoch vector. |
| Same-epoch loser refused | `Authorize` via all five entries | `TestAuthorizeRefusesSameEpochLoser` | Driven: a losing lease at the winning epoch refuses `lease_conflict` on input/mutation/checkpoint and parks `stale_owner` on activation/restore, for the standard and prefix-sharing losers. |
| Foreign session refused | `Authorize` via all five entries | `TestAuthorizeRefusesForeignSession` | Driven: a token for another session refuses `lease_conflict` on every entry, for the prefix-sharing and fully foreign sessions. |
| Expired grant refused | local-owner `Authorize` via all five entries → `sessrepo.CheckFencingExpiry` | `TestAuthorizeRefusesExpiredGrant`, `TestAuthorizeLapsedLocalOwnerStaleTokenRemainsLeaseConflict`, `TestAuthorizeRemoteOwnerInvalidPolicyRemainsInvalidArguments` | Driven: a local lapsed grant refuses `lease_conflict` on every entry so the holder revalidates; a grant at exactly the boundary stays current; an unusable policy refuses `invalid_arguments` even with a remote winner. For a remote winner, the lapsed result is deferred until after ownership direction so `remote_owner` remains reachable. Expiry logic is reused from the `sessrepo` owner, never restated. |
| Absent lease parks or refuses | `Authorize` via all five entries; `Observe` | `TestAuthorizeParksAndRefusesAbsentWinner`, `TestObserveLoadsWinningLease/no_leases_yields_no_winner` | Driven: with no winning lease, launch entries park `restore_policy` with an empty winning lease ID while the other entries refuse `lease_conflict`; `Observe` reports absence as data, never an error. |
| Ambiguous ownership parks or refuses | `Authorize` via all five entries | `TestAuthorizeParksAndRefusesAmbiguous` | Driven: conflicting union observations park `restore_policy` under the winner on launch entries and refuse `lease_conflict` elsewhere. |
| Remote ownership parks or refuses | `Authorize` via all five entries | `TestAuthorizeParksAndRefusesRemote` | Driven: a remote winning holder parks `remote_owner` under the winner on launch entries and refuses `not_owner` elsewhere, for prefix-sharing and fully remote holders. |
| Unverified sync parks or refuses | `Authorize` via all five entries | `TestAuthorizeParksAndRefusesUnverified` | Driven: without a proving lease refresh, launch entries park `restore_policy` under the winner and the other entries refuse `lease_conflict`. |
| Old-owner reconnect rejected, divergent history preserved | `AuthorizeInput`, `AuthorizeMutation`, `AuthorizeRestore`, `AuthorizeActivation`, `Repository.AppendEvent`, `sessstate.Projector.Project` | `TestOldOwnerReconnectRejectedAfterForceTakeover` | Driven: after a force takeover the old token refuses `stale_owner` from input/mutation/checkpoint on the new owner and parks `remote_owner` on the old wrapper; the losing event refuses stale with its bytes preserved and the chain untouched; the union projection reports `losing_branch_preserved` and `union_supersedes_chain` with authoritative state standing and the old-owner view stale. |
| Concurrent force takeovers resolve deterministically | `sessrepo.CompareLeaseTuple`, `sessstate.Compare`, `AuthorizeInput`, `AuthorizeMutation`, `AuthorizeActivation` | `TestConcurrentForceTakeoversDeterministicWinner` | Driven: two same-epoch force leases from one base resolve to the greater lease ID in either union order under both tuple rules; the winner authorizes and the loser stops accepting input with `lease_conflict`. |
| Token forgery refused | `LeaseToken.Bind`; constructor census | `TestBindRefusesForgedToken`, `TestLeaseTokenConstructorCensus`, `TestLeaseTokenCensusPlants`, `TestLeaseTokenUnforgeableOutsidePackage` | Driven: zero and field-assembled tokens refuse `lease_conflict` on every provider operation; only `token.go` names the seal types or assembles a non-empty token; comment/string mentions are not references while alias, direct, and shadowing forgeries are rejected; an outside package assembling the token by field name fails to build while the public-surface fixture builds. |
| Terminate-stale outside force recovery refused | `AuthorizeTerminateStale` | `TestTerminateStaleAuthorizesFencedTarget`, `TestTerminateStaleRefusesWithoutForce`, `TestTerminateStaleRefusesWithoutDiagnostics`, `TestTerminateStaleRefusesWithoutWinner`, `TestTerminateStaleRefusesLiveOwner`, `TestTerminateStaleRefusesForeignSession`, `TestTerminateStaleRefusesMalformedTarget`, `TestTerminateStaleRefusesGarbageWinner` | Driven: a fenced target (lower epoch, divergent, or beyond the winner) authorizes under explicit force recovery with preserved diagnostics; missing force, diagnostics, or winner, a live-owner target, a foreign session, and malformed members refuse their classes. |
| LeaseToken bound to provider operations from a passed gate only | `LeaseToken.Bind` | `TestBindProjectsMintedToken`, `TestBindRefusesUnknownOperation`, `TestMintedTokenIsStableAcrossEntries` | Driven: a token minted by a passed gate projects the exact triple onto `quiesce`, `capture`, and `materialize` identically from every entry; unknown operations refuse `invalid_arguments`. Resume, stop, materialize-commit, and native-store-plan carry the same triple on the wire but have no gate entry here (stated bound). |

## BUG-260917-2fwf8e arm-precedence matrix

The v0.7.0 §4.2 after-restore sequence requires the wrapper to offer
remote attach/takeover at step 4 when another host owns the session in an
interactive terminal. `Authorize` validates refresh-policy shape first,
then resolves ownership direction before consuming a lapsed-grant result:
a remote winner returns the existing `remote_owner` park/cause, allowing
the caller to select `attach_remote` or `takeover_offer`; a local winner
still reaches the expiry arm. An unusable policy remains the existing
`invalid_arguments` refusal even for a remote winner. The lease chain and
the Section 5.2 park vocabulary are unchanged.

| Vector | Production call site | Named test | Expected result |
| --- | --- | --- | --- |
| Remote interactive owner + lapsed local grant | `Authorize(OperationRestore, ...)` directly; composed by `axpane.Decide` | `TestAuthorizeRemoteInteractiveOwnerLapsedGrantUsesRemoteOfferArm`; `TestDecideLapsedRemoteOwnerOffersOrParksByInteractiveContext/{restore_interactive_attach,restore_interactive_takeover,launch_interactive_attach}` | `ParkDetails = remote_owner/<winning lease>` with `not_owner` cause, never `lease_conflict`; wrapper emits `attach_remote` or `takeover_offer`. |
| Live local grant | `Authorize(OperationRestore, ...)` directly | `TestAuthorizeArmOrderingNeighbourVectors/live_local_grant` | Authorized token for the exact winning tuple. |
| Remote non-interactive owner | `axpane.Decide` → `fencing.Authorize` | `TestAuthorizeArmOrderingNeighbourVectors/remote_owner_live_grant`; `TestDecideLapsedRemoteOwnerOffersOrParksByInteractiveContext/restore_noninteractive_park` | The direct neighbour uses a live grant; the wrapper row uses the lapsed grant and remains `remote_owner`; a non-interactive caller parks without an offer/runtime. |
| Local owner + lapsed grant | `Authorize(OperationRestore, ...)` directly | `TestAuthorizeArmOrderingNeighbourVectors/local_owner_lapsed_grant` | Literal `lease_conflict` refusal remains local-only. |
| Arm precedence narrowing | `Authorize` direct regression | `N-arm-order-expiry-before-direction` in `internal/fencing/testdata/mutate.py` | Swapping expiry before direction is applied in isolation and kills `TestAuthorizeRemoteInteractiveOwnerLapsedGrantUsesRemoteOfferArm`. |

Direct fencing vector coverage for this leaf is 4 of 4 named neighbour rows,
plus 5 of 5 `Authorize` operation entries in the lapsed-remote census and 5
of 5 in the lapsed-local stale-token refusal census. The three wrapper-context
rows are additionally driven through `Decide`, including `ModeLaunch` through
the `AuthorizeActivation` call site. The full importer-set before/after
per-test outcome diff is recorded in the task-scoped handoff results: 1,073
baseline rows versus 1,103 candidate rows, with 30 new named rows and no
existing row changing status.

The report-only fencing grid names the moved inputs rather than hiding them in
an unchanged pass total. V01 (probe 10), V07 (stale epoch), V08 (same-epoch
loser), V09 (future epoch), and V14 (empty `LocalHostID`) were each
`REFUSE(lease_conflict)` on all five entries before the fix. After the fix,
each is `PARK(remote_owner,cause=not_owner)` on activation/restore and
`REFUSE(not_owner)` on input/mutation/checkpoint. That move is intentional:
the ownership-direction arm is now reachable before a lapsed-grant result;
the local-owner lapsed-grant stale-token row remains `lease_conflict` on all
five entries.

## Refusal and recovery coverage

Every `N-` row is a genuine narrowing mutant: the gate stays present
and is weakened to admit exactly one member of the class it must
reject, and the named behavioral test fails through the delivered
harness (`internal/fencing/testdata/mutate.py`, run as
`python3 internal/fencing/testdata/mutate.py <evidence-dir>`).
Whole-clause disables are not accepted as narrowing. The `T-` row
preserves the searched-for `leaseSeal` token while constructing
through an alias, and its harness run executes the full behavioral
suite (263 passing assertions alongside the census failure), not only
the static checker. Controls: `C-harmless-comment` is applied and
survives, `C-not-applied` is not a kill, and `C-compile-failure`
produces no named behavioral failure.

| Gate | Named test | Narrowing mutant |
| --- | --- | --- |
| Operation enum | `TestAuthorizeRefusesUnknownOperation` | N-op admits exactly the bogus operation; the token authorizes and the negative fails by success. |
| Presented session grammar | `TestAuthorizeRefusesMalformedPresented/session_not_uuid` | N-presented-session admits exactly `not-a-uuid`; the refusal degrades to the foreign-session class. |
| Presented epoch grammar | `TestAuthorizeRefusesMalformedPresented/epoch_zero` | N-presented-epoch admits exactly the epoch-zero winner-lease vector; the refusal degrades to the stale class. |
| Presented lease grammar | `TestAuthorizeRefusesMalformedPresented/lease_not_uuid` | N-presented-lease admits exactly `zzz`; the refusal degrades to the loser class. |
| Winner grammar | `TestAuthorizeRefusesGarbageWinner/epoch_zero_lease` | N-winner admits exactly the epoch-zero `zzz` winner; the refusal degrades to the ahead class. |
| Session binding | `TestAuthorizeRefusesForeignSession/prefix_sharing` | N-session-prefix compares first UUID segments only; the prefix-sharing session authorizes while the fully foreign session still refuses. |
| Winner absence | `TestAuthorizeParksAndRefusesAbsentWinner/verified` | N-absent admits exactly the verified absence; the refusal degrades to the malformed-observation class. |
| Sync verification | `TestAuthorizeParksAndRefusesUnverified/unambiguous` | N-verified admits exactly the unambiguous unverified observation; the token authorizes and the negative fails by success. |
| Ambiguity | `TestAuthorizeParksAndRefusesAmbiguous/verified` | N-ambiguous admits exactly the verified ambiguity; the token authorizes and the negative fails by success. |
| Handoff outcome | `TestAuthorizeParksAndRefusesFailedHandoff/verified` | N-handoff admits exactly the verified failed handoff; the token authorizes and the negative fails by success. |
| Grant presence | `TestAuthorizeRefusesMissingGrant/verified` | N-grant admits exactly the verified grantless call; the refusal degrades to the lapsed-grant class. |
| Clock presence | `TestAuthorizeRefusesMissingClock/zero_validated` | N-clock admits exactly the clockless never-validated grant; the token authorizes and the negative fails by success. |
| Grant expiry | `TestAuthorizeRefusesExpiredGrant/lapsed` | N-expiry extends the interval by an hour; the lapsed-by-seconds grant authorizes while the never-validated grant still refuses. |
| Expiry classification | `TestAuthorizeRefusesExpiredGrant` | N-expiry-map swaps the lapsed/unusable classes; both vectors change class. |
| Arm precedence: ownership direction before grant expiry | `TestAuthorizeRemoteInteractiveOwnerLapsedGrantUsesRemoteOfferArm` | N-arm-order-expiry-before-direction swaps the two arms back to expiry-first; the remote lapsed-grant vector returns `lease_conflict` and the named test fails. |
| Ownership direction | `TestAuthorizeParksAndRefusesRemote/prefix_sharing` | N-remote-prefix compares first UUID segments only; the prefix-sharing remote holder authorizes while the fully remote holder still refuses. |
| Epoch equality, ahead side | `TestAuthorizeRefusesEpochBeyondWinner/exact_lease` | N-epoch-gte weakens `==` to `>=`; the ahead token authorizes while the lower-epoch vectors still refuse stale in the same run. |
| Epoch equality, lower side | `TestAuthorizeRefusesLowerEpoch/exact_lease` | N-epoch-lte weakens `==` to `<=`; the lower token authorizes past the epoch arm while the ahead vectors still refuse in the same run. |
| Lease equality | `TestAuthorizeRefusesSameEpochLoser/standard` | N-lease-vector admits exactly the standard loser; the prefix-sharing loser still refuses. |
| Lease equality, prefix confusion | `TestAuthorizeRefusesSameEpochLoser/prefix_sharing` | N-lease-prefix compares first UUID segments only; the prefix-sharing loser authorizes while the standard loser still refuses. |
| Launch classification, input side | `TestAuthorizeParksAndRefusesRemote/input` | N-launch-add-input parks input instead of refusing; the hard-refusal assertion fails. |
| Launch classification, restore side | `TestAuthorizeParksAndRefusesRemote/restore` | N-launch-drop-restore refuses restore instead of parking; the park assertion fails. |
| Capability seal | `TestBindRefusesForgedToken` | N-bind-seal admits exactly the field-assembled seal; the zero token still refuses. |
| Provider operation enum | `TestBindRefusesUnknownOperation` | N-bind-op admits exactly the bogus operation; the projection mints and the negative fails by success. |
| Terminate target grammar | `TestTerminateStaleRefusesMalformedTarget/lease_not_uuid` | N-term-presented admits exactly `zzz`; the termination authorizes and the negative fails by success. |
| Terminate winner grammar | `TestTerminateStaleRefusesGarbageWinner/epoch_zero_lease` | N-term-winner admits exactly the epoch-zero `zzz` winner; the termination authorizes and the negative fails by success. |
| Terminate winner presence | `TestTerminateStaleRefusesWithoutWinner/force` | N-term-absent admits exactly the forced winnerless call; the refusal degrades to the malformed-observation class. |
| Terminate force confirmation | `TestTerminateStaleRefusesWithoutForce/diagnostics` | N-term-force admits exactly the unforced diagnosed call; the termination authorizes and the negative fails by success. |
| Terminate diagnostics | `TestTerminateStaleRefusesWithoutDiagnostics/force` | N-term-diag admits exactly the undiagnosed forced call; the termination authorizes and the negative fails by success. |
| Terminate session binding | `TestTerminateStaleRefusesForeignSession/prefix_sharing` | N-term-session admits exactly the prefix-sharing foreign target; the refusal degrades to the live-owner class. |
| Terminate live-owner guard | `TestTerminateStaleRefusesLiveOwner/epoch_one` | N-term-live admits exactly the epoch-1 live owner; the epoch-2 owner still refuses. |
| Seal construction census | `TestLeaseTokenConstructorCensus` | T-census-alias adds an alias-backdoor plant preserving `leaseSeal`; the census rejects it while the full suite runs. |

Crash/idempotency matrix: the gates perform no durable writes, so
there is no crash seam. `TestGatesPerformNoDurableWrites` hashes the
repository tree, evaluates every entry (passing and failing) with
`Bind` and `AuthorizeTerminateStale`, and requires byte-identical
trees and identical verdicts across two runs.

Refusal classes are proven registered, never invented:
`TestGateRefusalsUseRegisteredCodes` constructs each sentinel through
`internal/axerror` with its exact exit (`lease_conflict`, `not_owner`,
`stale_owner` at 10; `invalid_arguments` at 2;
`local_precondition_failed` at 3), and every park decision unwraps to
its underlying Section 15 cause.

## Bounds (not silent gaps)

- No public CLI: 0 of 18 rows are delivered as an `ax` command. The
  shared-library entries above are the deliverable; orchestration
  leaves invoke them.
- No takeover/fork/stop/resume transactions: lifecycle flows are
  modeled as gate callers in `TestLifecycleCallersPassTheGate`; their
  orchestration belongs to the owning leaves.
- No terminal backend or provider plugin process: the gates own the
  wrapper and provider-operation entry points only.
- No `sessckpt` capture admission to reuse: no such package exists in
  this tree, so `AuthorizeCheckpoint` is the capture admission and its
  tests drive it directly.
- No `resume`, `stop`, `materialize-commit`, or `native-store-plan`
  provider binding: the wire triple is identical, but binding those
  operations here without their callers would be an untestable claim.
- No wall-clock lease expiry and no persisted grants: expiry retires
  only the process-local authorization through the `sessrepo` owner;
  the lease stays authoritative.
- Clause 5.3#5 (`max_observed_epoch + 1`) is caller-side input: the
  store mints head plus one, and the initiator's union maximum arrives
  with the synced store the gate reads.
- Precedence is decided and pinned: malformed calls die first, then
  foreign sessions, then presence and verifiability, then the grant,
  then direction, then the exact tuple match (`TestAuthorizePrecedence`).
