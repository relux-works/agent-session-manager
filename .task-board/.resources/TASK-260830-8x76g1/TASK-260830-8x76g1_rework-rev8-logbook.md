# TASK-260830-8x76g1 rework revision 8 logbook

## Recovery and source decision

The supplied archives were verified before use. The active Story worktree
already contained the combined accepted/carry-forward tree plus a later
revision 8 delta. Comparing the reconstructed archive tree under `.temp`
showed no missing file, so overwriting the nine newer candidate files would
have discarded valid rework. The current tree was therefore retained and the
archive comparison was recorded as source-binding evidence.

## Review finding closure

The pinned specification independently confirms the Session name grammar in
Section 2.1 and the Session Record `name` reference in Section 5.1. The public
identity gate validates it before self-field omission and hashing. The manifest
case-collision rule in Section 10.4 is preserved with Unicode simple-fold
equivalence while changing lookup complexity from quadratic to linear.

The member-enumeration artifact separates candidate-local type/member checks
from cross-object and external rules. README repeats that boundary and makes no
runtime migration, doctor, or capability claim.

## Validation anomaly

The prior revision 8 Change Request stopped at its first command because it ran
`gofmt -l .`, which traversed ignored task evidence under `.temp`. The candidate
validation command now enumerates only tracked and non-ignored `*.go` paths;
that exact gate exited 0. The board validator also exited 0 but printed 262
legacy `MISSING_ACTIVITY` diagnostics. They are inherited board-state
diagnostics outside this implementation scope and are preserved in the log.

## Negative-test decision

Two reversible mutants were run and restored before the final package/build
checks: a narrowed Session name maximum and a removed Session-name inventory
row. Both produced the expected non-zero test result, proving the acceptance
and structural drift guards rather than merely showing positive-path reachability.
