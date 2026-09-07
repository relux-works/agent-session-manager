# TASK-260906-3pln7q: establish-single-object-identity-owner

## Description
Close advisories A1 and A2: two identity/validation pipelines exist for the same schemas. canonicaljson is the documented owner of object identity and registers the three terminal schemas as refuse-everything placeholders (closed_shapes.go:115,130,132) while terminalbackend runs a private objectIdentity/checkIdentity pipeline (manifest.go:515,537); provhost.CheckIdentity (identity.go:90) re-implements canonicaljson.validateProviderIdentityRecord (core_records.go:207) rule-for-rule. Pick one owner per schema and make the other delegate, or formally transfer ownership and record it in the traceability registry. Follow the deduplication direction already set by internal/config importing terminalbackend.ParseID.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c). Resolve seams found by the consolidated cross-Story review (.temp/fable-consolidated-review.md). Work only inside the cross-story-identity-ownership-and-inventory-consistency story boundary. Preserve all stronger existing invariants; no gate may be weakened to close a seam.

## Acceptance Criteria
Production behavior demonstrates: exactly one owner validates each of urn:ax:schema:provider-identity 1.0.0 and the three terminal schemas, and the non-owner reaches it through a production call path rather than a copy. Where a copy is retained deliberately, a bidirectional agreement test drives both implementations over one fixture corpus and fails when they diverge. The safe-integer ordering defect in checkIdentity is closed: no jcs.Transform runs before member-type validation refuses a document. Negative tests prove each refusal arm, and the ownership registry names the owner.
