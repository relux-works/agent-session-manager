# TASK-260922-31qyyi: derivation-side-profile-source-gate

## Description
Implement the derivation-side authority and re-bind the SPEC 2.4 clause to it. Reproduce the hole first: disable sessrepo.checkWinningLease as an instrument and show probe 15 still yields {Profile: yolo, HasSource: true} at Projector.Project on the current trunk.

## Scope
internal/sessprofile and its callers; do not change the landed append-admission gate.

## Acceptance Criteria
With the append gate disabled as an instrument, a losing-lease profile.changed is refused as the effective profile source at every derivation entry; own tests, own narrowing mutant; the registry clause edge points at this owner.
