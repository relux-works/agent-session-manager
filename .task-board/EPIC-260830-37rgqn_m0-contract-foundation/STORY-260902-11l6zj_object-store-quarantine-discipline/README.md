# STORY-260902-11l6zj: object-store-quarantine-discipline

## Description
PutBlob quarantines an existing object on any inspection error, so a transient read failure can permanently sideline a valid immutable object. The projection path already classifies read failures correctly; the two paths disagree and the store side is the one that moves data.

## Scope
Normative scope: §3.2 object store, SPEC.md:819-820.

## Acceptance Criteria
Quarantine happens only on a proven digest mismatch or representation disagreement; a read failure is reported as a durability error with nothing moved, and the transient path is driven by a test rather than argued.
