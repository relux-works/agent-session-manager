# STORY-260902-2zsqwk: authoritative-record-schema

## Description
Reapply of STORY-260830-4n0fo8, whose three leaves all reached acceptance but whose workspace could not be rebased: trunk advanced thirteen commits while the Story was in flight and the automatic base refresh conflicts inside its already-checkpointed first leaf, a state with no operator surface (tracked as skill-project-management#104).

## Scope
Normative scope: §2, §5, §10.1, §18.1. Preserve all stronger global AX invariants and historical contract versions.

## Acceptance Criteria
The three accepted leaves are reapplied faithfully onto current trunk and the four cross-Story conflict hunks are resolved correctly, with byte-identity to the accepted trees asserted everywhere the conflict did not touch.
