# TASK-260908-snfd1u: propose-qualified-selector-and-peer-identity-contracts

## Description
User-requested decision proposal for the demonstrated gaps in pinned AX Sections 2.3 and 11.1: qualified selector grammar/semantics and independent inbound peer identity binding. Compare elegant, stable, safe designs; recommend precise contracts and migration/conformance requirements without approving or implementing a spec change. Justified gap: existing TASK-260830-21gygk requires undefined qualification and TASK-260830-z1yxg9 requires an inbound expected peer that current transport/config cannot independently supply. Existing stop packets verify the normative paragraphs and out-of-scope boundary. This bounded research supports those existing owners, does not duplicate their implementations, and does not touch deferred upstream issues 176/177.

## Scope
Read-only design analysis and task-scoped proposal for existing selector and peer identity gaps; no product code or approved specification changes

## Acceptance Criteria
Concrete selector grammar and identity semantics; supported inbound identity trust boundary with platform limits; safe enrollment and migration; failure scenarios; explicit decisions; primary-source citations; preservation of all existing implementation work
