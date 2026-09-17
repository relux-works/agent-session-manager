# TASK-260830-2atgj4 conformance matrix: ownership reducer properties

Authority: relux-works/agent-session-manager-spec v0.6.0, commit
`0cbdf100dbf84df50c64f792b1f940e3a67859a6` (pinned
`internal/specdoc/SPEC.v0.6.0.md`), Sections 2.2, 5.3, 13.6-13.10.
Each row maps one invariant or clause to the executable property, its
generator domain, its falsifying mutant, and the observed exits.
"No product behavior change" was the expectation; the union-order
property exposed one defect and fixed it minimally with a RED-first
regression (row P1, note 1).

## Invariant rows

| # | Invariant / clause | Property (production call site) | Generator domain | Falsifying mutant (harness exit) |
| --- | --- | --- | --- | --- |
| P1a | Union-order independence: permutations project byte-identically (§2.2#7 set union plus deterministic derivation; §5.3 greatest-tuple rule) | `TestOwnershipUnionOrderIndependent` (`sessstate.Reduce`) | Exhaustive: 969 union multisets of size 0..3 over 16 atoms (epochs 1..4 x leases R/A/B/C) x 3 chain shapes (empty, bootstrap A@1, succession A@1->B@2), all distinct permutations; seeded-random: 2000 multisets of size 4..6 (seed 260830, <=24 sampled permutations each). 63,050 `Reduce` calls. Winner also checked against the independent builtin-operator maximum. | `N-compare-tiebreak` KILLED (exit 1, `--- FAIL: TestOwnershipUnionOrderIndependent`): inverted tie-break still permutes identically but disagrees with the independent maximum. `N-union-detail-first` KILLED (exit 1): first-arrival evidence target permutes the conflicts while the winner stands. |
| P1b | Successive-union partitions project byte-identically (reconnect narrative) | `TestOwnershipUnionPartitionStable` (`sessstate.Reduce`) | Exhaustive: all ordered 2-partitions (2^n concatenations) of every size 0..3 multiset x 3 chain shapes. 21,315 partition concatenations, 24,222 `Reduce` calls. | Same `N-` pair as P1a (every concatenation is a permutation of the multiset, so permutation-invariance entails partition-invariance; executed distinctly). |
| P2a | Stale loser preservation: a lower-epoch lease is never dropped, rewritten, or promoted (§5.3#6/#7; §2.2#4) | `TestOwnershipLoserHistoryPreserved` (`sessstate.Reduce`); `TestOwnershipStoreLoserBytesPreserved` (`sessrepo.GetLease`, `ListLeases`) | Exhaustive: 2,907 (multiset, chain) pairs; every below-winner union lease named in a `losing_branch_preserved` detail, every shared epoch named in a `same_epoch_tie` detail, winner equals the independent maximum, authoritative derivation equals the union-free projection. Store: epoch-1 blob byte-identical across two successions, 3 leases listed, sibling session isolated. | `N-loser-drop-lower-epoch` KILLED (exit 1, `--- FAIL: TestOwnershipLoserHistoryPreserved`): lower-epoch losers vanish from the evidence while same-epoch losers still report. |
| P2b | Same-epoch loser preservation incl. tie evidence (§5.3#6/#7; §2.2#4) | Same tests as P2a | Same domain as P2a (multisets include same-epoch rivals R/A/B/C and duplicates; ties checked per shared epoch). | `N-loser-drop-same-epoch` KILLED (exit 1): same-epoch losers vanish while lower-epoch losers still report. |
| P2c | Winner is the tuple maximum; `Compare` is the total order the properties rely on (§5.3 greatest `(epoch, lease_id)`, bytewise UUID order) | `TestOwnershipWinnerIsTupleMaximum` (`sessstate.Compare`) | Exhaustive: 17 atoms (closed alphabet plus zero head), 289 pairs (sign agreement with the independent oracle, reflexivity, antisymmetry), 4,913 triples (transitivity). | `N-compare-tiebreak` (same plant; direction arm). |
| P3a | Clock non-authority over the store: `created_at` never influences epochs, tokens, holders, winner, or fencing verdicts (§5.3 `created_at` diagnostic-only; "no time-expiring ownership lease") | `TestOwnershipStoreClockNonAuthority` (`sessrepo.CreateLease`, `CompareAndSwapLease`, `WinningLease`, `VerifyFencingToken`) | 4 uniform shifts (2020 backwards, suite baseline, equal repeat, 2030 future) x create+2 CAS, plus one mixed non-monotonic chain. Authority projections identical; record digests vary (identity varies, authority does not); winner/future/stale verdict classes pinned per shift. | `N-create-clock` KILLED (exit 1, `--- FAIL: TestOwnershipStoreClockNonAuthority`): creation before 2026 refused; the backwards shift errors while baseline shifts still mint. Landed tests use 2026-08-19 dates and stay green under the plant. |
| P3b | Clock non-authority over the gates: absolute wall-clock position never influences the verdict (§5.3 revalidation list; grant age stays authoritative by design) | `TestOwnershipGateClockNonAuthority` (`fencing.Authorize*`, all five entries) | 4 ages (0, 30s fresh, 60s exact boundary current, 61s lapsed) x 4 translations (-1h, 0, +1h, +365d) x 5 operations = 80 authorizations. Verdict (authorized / refused class / parked reason+cause+lease) identical per (age, operation); age classes pinned (current authorizes, lapsed refuses `lease_conflict` on every entry). | `N-expiry-absolute` KILLED (exit 1, `--- FAIL: TestOwnershipGateClockNonAuthority`): only 2026-validated grants expire, so a lapsed grant translated to 2027 authorizes while the baseline refuses. Landed `TestCheckFencingExpiry` (all-2026) stays green: only the translation property kills this plant. |
| P4a | Zero duplicates over the store: at most one authoritative lease per session and epoch; the winner is the head (§2.2#1; §5.3 epoch-plus-one) | `TestOwnershipStoreSingleAuthorityPerEpoch` (`sessrepo.WinningLease`, `ListLeases`, `CompareAndSwapLease`, `GetLease`) | Exhaustive: reason sequences of length 0..2 over the 4 Section 5.3 reasons (21 sequences); seeded-random lengths 3..5 (seed 260831, 200 sequences). Epochs exactly 1..n in tuple order, per-epoch count 1, winner is head and compared maximum, every blob verifies. | Covered by the P4b gate plants for authorization; store single-head persistence is additionally pinned by `TestOwnershipStoreTwoHandlesOneHead` (a lagging CAS refuses stale instead of forking a second epoch-2 head; the refused CAS persists nothing). |
| P4b | Zero duplicates over the gates: exactly the winner authorizes; no two hosts, processes, or presenters both pass (§2.2#1 one winning owner; §2.2#2 replica restraint) | `TestOwnershipGateSingleAuthorizedOwner` (`fencing.Authorize*`, all five entries) | Exhaustive: 6 presenters (exact, stale, below-/above-winner same-epoch losers, future, foreign) x 5 observations (local, remote, ambiguous, failed-handoff, unverified) x 5 operations = 150 authorizations. Local table pinned cell by cell; exactly one authorization per local operation; zero elsewhere; pairwise mutual exclusion asserted over the recorded table in both framings (two presenters, two hosts). | `N-gate-same-epoch-loser` KILLED (exit 1): the above-winner loser authorizes and exactly-one fails. `N-gate-remote-holder` KILLED (exit 1): the remote exact token authorizes and the two-host exclusion fails. |
| P5 | P3 vector from the 3g12yp verdict: epoch-zero with a well-formed third lease refuses `invalid_arguments` | `TestAuthorizeRefusesMalformedPresented/epoch_zero_other_lease` (`fencing.Authorize*`, all five entries) | Fixed vector `{session A, epoch 0, fenceLeaseC}` x 5 entries. | Existing `N-presented-epoch` plant (landed harness) still killed by the winner-lease vector; the new vector refuses under that plant (proves the epoch gate fires regardless of lease identity). |

## Notes

1. Defect and minimal fix (RED-first). The P1 property failed on the
   landed reducer: `resolveWinner` reported each losing union lease
   against the transient arrival-order winner, so a loser's evidence
   (even its presence) depended on union order. `internal/sessstate/
   sessstate.go` now selects the greatest tuple in a first pass and
   reports every entry below it in ascending tuple order naming the
   final winner, with the off-chain flag recomputed from the final
   winner. RED log: `.temp/TASK-260830-2atgj4/red-ownership-
   properties.log` (4 of 5 fail); GREEN log: `green-ownership-
   properties.log` (5 of 5 pass). All landed union assertions were
   order-free; only the refusal-census line pins moved
   (sessstate.go:924/927 to :929/:932).
2. Harness: `internal/sessstate/testdata/mutate_properties.py`
   (isolated tree copy, per-plant raw logs, real exits). Full run:
   8 `N-` plants KILLED, `C-harmless-comment` SURVIVED (applied),
   `C-not-applied` NOT_APPLIED, `C-compile-failure`
   COMPILE_OR_HARNESS_FAILURE, both `TestOwnership` controls exit 0,
   harness exit 0. Logs and `mutants-full.json` under
   `.temp/TASK-260830-2atgj4/mutants/`.
3. No source-text gate is shipped by this task, so no new
   token-preserving `T-` mutant is owed: the only token-searching
   gate in scope (the `LeaseToken` constructor census) keeps its
   landed `T-` mutant. The P2 property asserts both the searched-for
   token (loser named in the detail) and the behavior (winner is the
   maximum, state unchanged), and the harness executes the full
   behavioral property, not a static check.
4. Traceability: `lease-ownership-union-properties` (production
   `sessstate.Reduce`), `lease-ownership-store-properties`
   (production `sessrepo.WinningLease`), and
   `lease-ownership-gate-properties` (production `fencing.Authorize`)
   are registered with the 11 property tests above and attached to
   the discharged 5.3#6, 5.3#7, 2.2#1, 2.2#2, and 2.2#4 clause
   evidence; coverage ratios unchanged (5.3 partial 7/8 with 5.3#5
   open, 2.2 sliver 4/22); section:17.2 gap reworded to its narrower
   reader-enum meaning; registry digest re-pinned to
   `63c8379637c5ab72dbc979eed8dcd92b7149969deb3690fa7109a64789e0e121`;
   acceptance count 110 to 113.
5. Stated bounds: grant age relative to the refresh policy is
   authoritative by design (P3b translates with age fixed);
   malformed-union refusal evidence names the first malformed entry
   in arrival order (P1 covers well-formed inputs; malformed vectors
   stay with the landed negatives); the suite adds no `ax` command,
   no `doctor` result, and no runtime capability claim
   (`TestRealREADMECarriesNoPositiveClaim` green).
