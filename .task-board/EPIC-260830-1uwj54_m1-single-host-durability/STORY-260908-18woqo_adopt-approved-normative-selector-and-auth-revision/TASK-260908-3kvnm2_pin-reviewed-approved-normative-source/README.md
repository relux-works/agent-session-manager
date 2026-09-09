# TASK-260908-3kvnm2: pin-reviewed-approved-normative-source

## Description
After the normative source delivery pool EPIC-260908-itxemt in agent-session-manager-spec lands, verify exact signed source/release provenance and adopt its immutable document and pin metadata. Preserve historical v0.5.0 provenance and avoid rewriting historical objects or inventing release tags. This task changes source authority only; new product contract support must remain truthfully tracked.

## Scope
internal/specpin, internal/specdoc and directly required pin/provenance documentation and tests.

## Acceptance Criteria
Pin references an actual verified signed landed source/release; embedded SPEC and section inventories match measured bytes; old pin provenance remains auditable; tests drive real pin readers and reject wrong source/hash/version evidence.
