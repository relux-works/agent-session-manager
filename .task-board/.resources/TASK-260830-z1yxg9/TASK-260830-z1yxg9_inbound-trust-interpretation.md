# Inbound trust interpretation withdrawn \u2014 explicit decision pending

Primary correction, 2026-09-08. The provisional membership-only interpretation previously attached under this resource name is withdrawn. Do not treat it as approved product behavior or as permission to relax the spoofed-host acceptance case.

Section16.1 excludes payload-confidentiality protection from an authorized peer, trusted plugin or compromised local user. That limited statement does not itself waive identity or integrity obligations. The prior primary inference broadened the exclusion and therefore cannot establish that membership-only responder admission satisfies section11.1. The exact published section11.1 requirement and missing inbound association described in TASK-260830-z1yxg9_blocked-outcome.md remain the authoritative unresolved constraint.

RPC run RUN-260908-cc2b6a was launched under that provisional interpretation and immediately given an operator cancellation directive after this error was identified. Preserve any work and the previously accepted checkpoints; no result from this run is accepted or authorized for landing. The original inference remains in ignored scratch history only to explain the correction, not as active instructions.

The decision remains: provide an independently authenticated SSH principal/invocation-to-configured-host association with an approved contract, or explicitly accept membership-only inbound admission and its bound that a trusted authenticated SSH user may assert any listed host ID. The latter cannot simultaneously claim cross-listed-host spoofing resistance. No option is assumed, and a default UI choice is not approval.

All seven peer-auth conformance cases and all RPC acceptance criteria remain open. The earlier dependency-order repair still stands; it did not change product behavior. Native changed-key/replay tests and disclosure enforcement still require real production evidence. Qualified selectors remain a separate pending decision. Independent Git work continues while this security decision is pending.
