# BUG-260902-lyhvkw: cap-canonical-decode-recursion-depth

## Description
Uncapped recursion in decodeValue kills the process at the identity gate. internal/canonicaljson/canonical.go:333 decodeValue recurses once per nesting level with no depth cap, so a deeply nested document exhausts the goroutine stack and aborts the process with a fatal runtime error that recover() cannot catch.

Reproduced independently on clean main 48db30b through the public Canonicalize entry: depth 1000 accepted, depth 100000 accepted, depth 1000000 (a 2 MB input of nested open brackets) produces runtime: goroutine stack exceeds 1000000000-byte limit followed by fatal error: stack overflow and exit status 2.

2 MB sits well under the 5,242,880-byte identity cap that CalculateObjectIdentity and VerifyObjectIdentity apply, so that cap does not protect them; Canonicalize applies no size cap at all. This is the exact boundary where untrusted peer objects arrive, so a malformed Transfer Manifest from a peer terminates the host process. No test covers nesting depth.

Found by an adversarial audit of landed main and the three open story branches, then reproduced by the orchestrator before filing.

## Scope
Normative scope: SS1.6 canonicalization and the SS10.1 identity entries. Apply a bounded nesting depth at the shared decode path so every public entry inherits it.

## Acceptance Criteria
A declared maximum nesting depth is enforced in decodeValue and refuses past the limit with a typed error rather than a runtime abort, proven at both public identity entries and at Canonicalize, accept-at-limit and refuse-past-limit. The bound is a declared constant with a stated rationale, a negative case drives each public entry one level past the limit, and a widening mutant of the bound reddens the suite. Coverage must not regress.
