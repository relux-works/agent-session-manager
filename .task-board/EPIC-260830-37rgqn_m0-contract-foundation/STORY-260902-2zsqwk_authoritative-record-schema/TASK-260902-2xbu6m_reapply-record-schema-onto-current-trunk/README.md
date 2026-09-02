# TASK-260902-2xbu6m: reapply-record-schema-onto-current-trunk

## Description
Reapply the three accepted leaves of the record-schema Story onto current trunk. The work is attached as a precondition patch series of three commits from base 48db30b.

WHY THIS EXISTS. Trunk advanced thirteen commits (48db30b to 10aaa16) while the Story was in flight, so its checkpoint no longer descends from trunk and Change Request construction refuses. The automatic base refresh aborts on a conflict located inside the ALREADY-CHECKPOINTED FIRST leaf, which no producer can resolve: a hand rebase rewrites reviewed commits and orphans the recorded checkpoint OID, moving the workspace from stale to unrecoverable. There is no worktree subcommand that re-runs a base refresh with operator conflict resolution. Filed as skill-project-management#104. Nothing is wrong with the work.

THE CONFLICT IS FOUR HUNKS, all structural cross-Story lines, and the previous producer resolved them in a throwaway probe and recorded how:
  README.md Go-toolchain tools row - union merge
  internal/traceability/ownership.v0.5.0.json, two arrays - union merge
  reviewedOwnershipCanonicalSHA256 in internal/traceability/traceability.go - DERIVED, must be RECOMPUTED not chosen: union-merge the registry, run tracecheck, and read the computed digest out of the traceability.go:286 refusal.

WHAT THE THREE LEAVES CONTAIN, all reviewed to acceptance across fifteen rounds: closed Session Record 1.0.0/2.0.0/3.0.0 validators driving both public identity entries; derived inventories that prove their own coverage for member sets, declared bounds, refusal guards, grammar rows and the array phrase-to-validator mapping; and the closure work of the final leaf. The last review killed 20 of 20 mutants with every derived ordering site flipped individually, and the mapping itself found a live invented constraint nobody had looked for.

## Scope
Normative scope: §2, §5, §10.1, §18.1.

## Acceptance Criteria
All three commits are reapplied onto current trunk as ONE signed commit. Byte-identity to the accepted trees is asserted for every file the conflict did not touch, and any deviation is reported rather than absorbed. The four conflict hunks are resolved as recorded, with the derived digest RECOMPUTED from tracecheck rather than copied. Every gate exits 0 and coverage does not regress against the accepted trees.
