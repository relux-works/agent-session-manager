# TASK-260908-2tkufa: bind-approved-contract-catalog-and-implementation-owners

## Description
Regenerate and reconcile contract catalog, traceability ownership and source-version validation after the approved normative pin is adopted. Every added or refined selector/auth/migration obligation must map to its real implementation owner; unsupported new runtime behavior must not be reported as implemented. Preserve acceptance evidence for unchanged legacy contracts and resume the existing dependent leaves after this Story lands.

## Scope
internal/catalog, internal/traceability, directly required generator checks and board scope bindings; no edits to another Story partial implementation.

## Acceptance Criteria
Full contract census and source references match the adopted normative source; existing and added contract obligations have non-duplicated real owners; generator/traceability gates pass and detect omissions; legacy support claims remain truthful; exact reviewed signed Story delivered before unblocking dependent code.
