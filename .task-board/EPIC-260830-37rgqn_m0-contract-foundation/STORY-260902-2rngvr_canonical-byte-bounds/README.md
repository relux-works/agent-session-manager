# STORY-260902-2rngvr: canonical-byte-bounds

## Description
Declared byte bounds are measured on Go HTML-escaped JSON rather than canonical bytes, so the effective limit is implementation-defined and diverges from the pinned one by up to six times. The same defect exists in two packages, and one call site already does it correctly.

## Scope
Normative scope: §6 and §10.1 declared byte bounds.

## Acceptance Criteria
Every declared byte bound is measured on canonical bytes, the two sites that disagreed share one helper, and a document at the declared limit is accepted regardless of how far JSON escaping would expand it.
