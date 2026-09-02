# BUG-260902-1qsdag: quarantine-only-on-proven-mismatch

## Description
PutBlob quarantines an existing object on ANY inspection error, including a transient read failure, so a flaky read can permanently sideline a valid immutable object.

internal/localstore/object_store.go:216: any non-nil existingErr that is not ErrUnsafeOwnership reaches quarantineExisting at :229. inspectExisting returns raw os.Open, io.Copy and Close errors at :282-293, so an EMFILE or a transient I/O error is indistinguishable from a hash mismatch at the decision point.

SPEC.md:819-820 mandates quarantine only for a hash mismatch or representation disagreement. A read that did not complete proves neither.

The projection path already gets this right: projection.go:704-774 classifies a read failure as sourceFailure and refuses rather than quarantining. The two paths disagree with each other, and the store side is the one that moves data.

Established by code reading. The auditor did not build the injection: reproducing it needs file-descriptor exhaustion or an injectable reader, which storeOperations does not currently expose.

## Scope
Normative scope: §3.2 object store, SPEC.md:819-820.

## Acceptance Criteria
A read failure during inspection is reported as a durability error with nothing moved, and quarantine happens only on a proven digest mismatch or representation disagreement. storeOperations gains an injectable reader so the transient-failure path is driven by a test rather than argued. A negative case proves a transient read error leaves the object in place, and reddens when the classification is collapsed back. The projection and store paths agree, and the agreement is asserted rather than assumed.
