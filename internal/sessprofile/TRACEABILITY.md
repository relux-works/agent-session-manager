# TASK-260922-31qyyi: derivation-side profile source authority

Authority: `internal/specdoc/SPEC.v0.7.0.md` §2.4, especially the rule that
losing-lease or ambiguous events must not change the effective profile. The
Section 2.4 ownership edge and its clause acceptance-case edge are decoded and
checked by `TestSection24LosingLeaseClausePointsAtDerivationAuthority`; the
reviewed ownership digest is re-derived by the traceability test gate.

`Derivation.Derive` owns the repository-bound source rule. It consumes lease
rows from `sessrepo.Repository.ListLeases`, which validates the complete store
and returns the winning row from the same snapshot, and asks `sessckpt.Store`
to re-attest the winning handoff closure when present. A known previous-lease
source is effective only inside that captured closure. Current-winner events
must match its exact epoch and lease ID. An unminted higher epoch, an unminted
same-epoch tuple, and any profile event when the lease store is empty are
ignored; with no lease row the Session Record creation profile remains the
source. If a winning checkpoint closure cannot be read, a previous-lease
source is refused closed rather than treating the failed read as an empty
closure.

## Task acceptance rows

5 of 5 task-specific acceptance rows are driven by named candidate tests and
the captured outcome artifacts. This ratio counts the five task acceptance
rows in the assignment; the separate board checklist is reported in the
handoff results.

| Acceptance row | Production call site | Named test or evidence | Result |
| --- | --- | --- | --- |
| Probe 15 is no longer yolo with the append gate disabled | `Projector.Project` | `TestProbe15DisabledAppendGateProjector`; instrumented trunk baseline in `reproduction/probe-15-instrumented-baseline.log` | Baseline was `{Profile:yolo, HasSource:true}`; candidate returns `standard` with no source. |
| Independent derivation rule covers each required entry and has its own narrowing evidence | `LoadProfile`, `Projector.Project`, `Projector.ProjectForHeads`, `Transactor.SetProfile` from-end, `axpane.deriveProfile` | The gate × entry matrix below; `mutate_profile_authority.py` re-runs all five lower-epoch path witnesses with `checkWinningLease` disabled and kills M1 at each path; M2 is killed by the same-epoch `deriveProfile` witness | Five non-skipped path tests also run with the real append gate on. Both M1 and M2 make the plain full suite fail. The original four task acceptance entry categories are covered, with `Projector.ProjectForHeads` separately exercised as its own public path. |
| Higher-epoch and empty-store behavior is decided and pinned | `Projector.Project`, `Projector.ProjectForHeads`, `Derivation.DeriveForHeads`, `LoadProfile` | `TestProjectorRejectsNeverMintedHigherEpochProfileSource`, `TestProjectorForHeadsRejectsUnmintedHigherEpochProfileSource`, `TestDerivationForHeadsRejectsUnmintedHigherEpochSource`, `TestProjectorEmptyLeaseStoreKeepsRecordAuthority`, `TestLoadProfileEmptyLeaseStoreKeepsSessionRecordAuthority` | Unminted higher epochs and all profile events with an empty lease store are not sources; the creation profile remains effective. A source in the winning handoff closure remains effective in `TestProjectorKeepsPriorLeaseSourceFromWinningHandoffClosure`. |
| Section 2.4 clause 2 is rebound to the derivation owner and acceptance edge | `sessprofile.Derivation.Derive` | `TestSection24LosingLeaseClausePointsAtDerivationAuthority`, registry re-derivation tests, `tracecheck` | The decoded registry assigns `internal/sessprofile/authority.go:Derive`; `story-260922-derivation-side-profile-source` occurs in both the section and clause acceptance lists. |
| The importer census and writer composition are compared by runtime outcome | Complete `internal/sessprofile` importer set | `testdata/compare_importers.py` and its `outcome-comparison.json` | Every keyed input class is named; valid handoff derivation and the current-winner `SetProfile` write remain unchanged. |

## Gate × entry matrix

Each losing-lease witness creates a profile change while lease A is the
winner, after a checkpoint C, then transfers ownership to B naming C. The
real append gate admits the event under A; source authority must still keep it
out of the effective profile after B wins. The harness reruns each witness in
a scratch copy with only `sessrepo.checkWinningLease` replaced by a no-op. M1
admits exactly the fixture event while preserving every other source rule.
The M2 test supplies a same-epoch event with a lease ID absent from the
authority rows directly to `deriveProfile`; no append mutation is needed for
that projection-level case. The harness runs both mutants against the plain
full suite with the append gate intact and records each raw log and exit code.
Its harmless comment control survives.

| Rejected event class | Derivation entry | Production path exercised | Named test | Narrowing row |
| --- | --- | --- | --- | --- |
| Known lower-epoch event outside the winning handoff closure | `LoadProfile` | `LoadProfile` → `Derivation.Derive` | `TestLoadProfileDoesNotExposeLosingLeaseAsEffectiveSource` | M1 exact-event narrowing |
| Known lower-epoch event outside the winning handoff closure | `Projector.Project` | `Projector.Project` → `Derivation.Derive` | `TestProbe15DisabledAppendGateProjector` | M1 exact-event narrowing |
| Known lower-epoch event outside the winning handoff closure | `Projector.ProjectForHeads` | `Projector.ProjectForHeads` → `Derivation.DeriveForHeads` | `TestProjectorForHeadsDoesNotExposeLosingLeaseAsEffectiveSource` | M1 exact-event narrowing |
| Known lower-epoch event outside the winning handoff closure | `SetProfile` from-end | `Transactor.SetProfile` → `Derivation.Derive` | `TestSetProfileFromEndDoesNotReplayLosingLeaseChange` | M1 exact-event narrowing |
| Known lower-epoch event outside the winning handoff closure | `axpane.deriveProfile` | `LoadProfile` → `deriveProfile` → `Derivation.Derive` | `TestAxpaneDeriveProfileDoesNotUseLosingLeaseSource` | M1 exact-event narrowing |
| Unminted same-epoch tuple with a mismatched lease ID in the supplied closure | `axpane.deriveProfile` | `deriveProfile` → `Derivation.Derive` | `TestAxpaneDeriveProfileRejectsUnmintedSameEpochLeaseTuple` | M2 epoch-only `knownLease` narrowing |
| Same-epoch tuple rejected before durable loading | `LoadProfile` | `LoadProfile` → `sessrepo.AppendEvent` admission boundary | `TestEmitReachesSameEpochAppendAdmissionGate` | Unreachable after the append refusal; source authority's shared M2 predicate is tested through `axpane.deriveProfile` above |
| Same-epoch tuple rejected before durable projection | `Projector.Project` | `Projector.Project` → `sessrepo.AppendEvent` admission boundary | `TestEmitReachesSameEpochAppendAdmissionGate` | Unreachable after the append refusal; source authority's shared M2 predicate is tested through `axpane.deriveProfile` above |
| Same-epoch tuple rejected before durable from-end loading | `SetProfile` from-end | `Transactor.SetProfile` → `sessrepo.AppendEvent` admission boundary | `TestEmitReachesSameEpochAppendAdmissionGate` | Unreachable after the append refusal; the writer derives from the same authority before its own append |

`Projector.ProjectForHeads` has a losing-lease M1 witness above and an
unminted-higher-epoch witness,
`TestProjectorForHeadsRejectsUnmintedHigherEpochProfileSource`;
`Derivation.DeriveForHeads` is covered by
`TestDerivationForHeadsRejectsUnmintedHigherEpochSource`. The checkpoint reader
re-attests the closure at `sessckpt.Store.EventHeads`; its named witness is
`TestEventHeadsReattestsCheckpointClosureForSession` in `internal/sessckpt`.
An empty lease store has no winning lease to supply a handoff closure; its
record-authority behavior is pinned by the empty-store tests in both
`sessprofile` and `axpane`.

## Importer outcome comparison

The complete direct Go importer census contains only
`github.com/relux-works/agent-session-manager/internal/axpane` (including its
test build variant). With the append gate disabled in both isolated trees, the
outcome comparison contains 21 keys on the base and candidate trees.
`LoadProfile`, `Projector.Project`, `Projector.ProjectForHeads`, and
`axpane.deriveProfile` each move for the lower-epoch, unminted higher-epoch,
empty lease-store, and unminted same-epoch inputs. `SetProfile` from-end also
moves for the lower-epoch and same-epoch inputs because its previous profile
is now derived from the authorized source. A source inside the winning
captured handoff closure and a normal current-winner `SetProfile` write do not
move.
Raw inputs, exact outcomes, importer rows, tree hashes, and exit codes are in
the attached task outcome artifact.

## Boundaries

The pure reducers `sessprofile.Derive` and `DeriveForHeads` remain authority-
agnostic helpers for already supplied data. Repository consumers pass through
`Derivation` or `Projector`; `Projector.ProjectForHeads` and
`Derivation.DeriveForHeads` retain the same lease-source checks. A wrong
same-epoch lease tuple cannot reach the durable loader while the append gate is
on; M2 tests the source predicate with an already supplied `Derivation`. This
task does not change `sessrepo.checkWinningLease` or append admission.
