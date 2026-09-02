# Flight Logbook

> Institutional memory. Concise, factual, high-signal.
> Newest entries first. One block per insight.

## 2026-09-02

### 0900 — BUG-260902-lyhvkw rev-2 reapplied onto advanced trunk a351afd
- FINDING: the accepted revision-2 patch reapplied cleanly except `internal/traceability/traceability.go`, where trunk had independently repinned `reviewedOwnershipCanonicalSHA256`; resolved to the trunk value, then repinned to the recomputed projection digest `9f7737cb07f853012fcc9e2359981e20e9b65df622f9da7fa4935a2180cd04b0` reported by tracecheck.
- STATUS: canonicaljson coverage 87.1% (trunk a351afd) -> 87.2%; widen-512, narrow-128, and delete-gate mutants all redden the suite; the delete-gate mutant still reproduces `fatal error: stack overflow`.

### 0420 — Canonical decode nesting depth capped at 256
- ROOT CAUSE: `internal/canonicaljson/canonical.go` `decodeValue` recursed once per nesting level with no bound; a 2,000,000-byte document of one million nested arrays exhausted the goroutine stack and killed the process with uncatchable `fatal error: stack overflow` at `Canonicalize`, `CalculateObjectIdentity`, and `VerifyObjectIdentity`, well under the 5,242,880-byte identity size cap.
- FIX: `maxNestingDepth = 256` (`internal/canonicaljson/canonical.go:37`) enforced inside `decodeValue` before it opens a container (`canonical.go:362`), typed `ErrInvalidJSON`; every public entry inherits it through the shared `decodeStrict`.
- DECISION: 256 chosen because the deepest pinned closed shape (16-level submodule tree) decodes at roughly depth 40 and open extensions stop at depth 4, giving more than sixfold headroom; the bound stays far below encoding/json's 10,000-level cap and jcs's unbounded re-parse, so the typed refusal is always the first gate a deep document meets.
- FINDING: the delete-gate mutant reproduces the exact crash (`runtime: goroutine stack exceeds 1000000000-byte limit`, exit 1 with fatal error) through the new regression test; widen-512 and narrow-128 mutants both redden the suite because tests pin the literal 256/257 rather than reading the constant.
- FINDING: registering new evidence tests in `internal/traceability/ownership.v0.5.0.json` requires repinning `reviewedOwnershipCanonicalSHA256` in `internal/traceability/traceability.go:42`; tracecheck fails closed on the projection digest until the repin is reviewed.
- SCOPE: `internal/canonicaljson/canonical.go`, `internal/canonicaljson/nesting_depth_test.go`, `internal/canonicaljson/testdata/constraint-enumeration.md`, `internal/canonicaljson/testdata/fuzz/FuzzCanonicalizeRoundTrip/nesting-depth-over-limit`, `internal/traceability/ownership.v0.5.0.json`, `internal/traceability/traceability.go`, `README.md`.
- STATUS: fixed in STORY-260902-3os1kh workspace (trunk 48db30b descendant); canonicaljson coverage 87.1% -> 87.2%; handed off to review.
