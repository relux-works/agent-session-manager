# STORY-260903-1h7ybr: canonicalize-number-contract

## Description
The exported Canonicalize doc promises refusal of non-I-JSON input; it silently rounds instead. Finding N5 of the second adversarial audit, the only one of thirteen still open.

## Scope
Normative scope: RFC 8785 I-JSON surface; internal/canonicaljson.

## Acceptance Criteria
Either Canonicalize refuses the numbers its documentation says it refuses, or the documentation states what it actually does and the rounding is pinned as intended behaviour with the reason.
