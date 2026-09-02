# TASK-260902-24e45b: reapply-object-store-onto-current-trunk

## Description
Reapply the two accepted leaves of the object-store Story onto current trunk. Attached as a precondition patch series of two commits from base 48db30b: e3e5ebb safe local layout and object sink, 81305c8 sqlite projection and rebuild.

WHY THIS EXISTS. The Story branch forked from 48db30b and trunk is now at 010d114, so it no longer descends from trunk. More pressingly, internal/localstore exists ONLY on that branch, so every later change to the package is unlandable: the quarantine classification fix was produced and fully verified against checkpoint 81305c8 and had to stop rather than import four thousand lines of an unaccepted package into a different Story branch.

WHAT THE TWO LEAVES CONTAIN, reviewed across fourteen rounds. Leaf one: safe local layout and the object sink, six rounds. Leaf two: the SQLite projection and rebuild, eight rounds, which closed two confirmed integrity bypasses - a TRIGGER escaping the closed-schema gate because the check enumerated the kinds it inspected instead of refusing the complement, and a second identical-shape bypass in the very next predicate whose excluded set was attacker-chosen. It also closed an availability defect: with both blob-leaf shape clauses disabled, OpenProjection HUNG in os.Open on a writerless FIFO with no refusal and no timeout.

Expect conflicts in the same structural cross-Story lines every reapply has hit: the README tools row and the ownership registry arrays want a union merge, and reviewedOwnershipCanonicalSHA256 is DERIVED - union-merge the registry, run tracecheck, and read the computed digest out of its refusal rather than choosing either side.

## Scope
Normative scope: §3.2, §3.3, §10.1, §10.2, §18.4.

## Acceptance Criteria
Both commits are reapplied onto current trunk as ONE signed commit. Byte-identity to the accepted trees is asserted for every file no conflict touched, and any deviation is reported rather than absorbed. Derived values are recomputed, never copied. Every gate exits 0 and coverage does not regress against the accepted trees.
