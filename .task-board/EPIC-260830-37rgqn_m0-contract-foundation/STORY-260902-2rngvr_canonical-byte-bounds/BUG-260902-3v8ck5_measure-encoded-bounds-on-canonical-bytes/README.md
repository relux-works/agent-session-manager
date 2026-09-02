# BUG-260902-3v8ck5: measure-encoded-bounds-on-canonical-bytes

## Description
Declared byte bounds are measured on Go HTML-escaped JSON rather than canonical bytes, so the effective limit is implementation-defined and diverges from the pinned one by up to six times.

Probed with the full Launch Plan shape: an argv of 24 x 1000 less-than characters is 24,073 canonical bytes but 144,073 through json.Marshal, and is refused with the message argv encodes to 144073 bytes (internal/canonicaljson/closed_shapes.go:262-267). Plain control characters at the same canonical size are accepted. U+2028 doubles as well: 65,569 canonical becomes 131,089 escaped.

The same defect exists in the configuration package on the extension bound (validation.go:752).

The extensions path in canonicaljson already does it correctly at :1487-1495 by measuring canonical bytes, so the repository contains both the defect and its fix.

It fails closed, so no oversized document is admitted. The harm is that a conforming document can be refused, the declared 65,536 bound is not the bound in force, and two call sites in one repository disagree about what a byte is.

## Scope
Normative scope: §6 and §10.1 declared byte bounds.

## Acceptance Criteria
Every declared byte bound is measured on canonical bytes, either by reusing Canonicalize or by an encoder with SetEscapeHTML(false). A property test asserts that a document at the declared limit is accepted regardless of how many characters JSON escaping would expand, using the SPEC literal rather than the implementation constant. The two sites that disagreed now share one helper, and a mutant restoring escaped measurement reddens.
