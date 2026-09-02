# STORY-260902-2ni6i3: object-store-fault-boundaries

## Description
Fault-injection and filesystem-boundary proofs for the object store and SQLite projection, now that the package is on trunk. Split from STORY-260830-2dy8oj, whose two implementation leaves were reapplied and landed separately after trunk moved past its branch.

## Scope
Normative scope: §3, §10.1-10.2, §18.4.

## Acceptance Criteria
Object writes, SQLite rebuilds, permissions, symlinks, special files, short writes and full-disk boundaries are fault-injected at the production entries, with crash and idempotency evidence for every operation that mutates durable state.
