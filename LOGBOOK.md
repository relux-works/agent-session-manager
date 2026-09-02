# Flight Logbook

> Institutional memory. Concise, factual, high-signal.
> Newest entries first. One block per insight.

## 2026-09-02

### 1353 — ssh_args admission derived from a declared grammar, not a name blacklist
- ROOT CAUSE: `sshHostAuthenticationBypass` (`internal/config/validation.go:846`, called from `:465`) matched three option names, which cannot satisfy SPEC.md 6.3's "or an equivalent host-authentication bypass"; it inspected only whole `-o` tokens and matched only `no|off`.
- FINDING: reproduced against OpenSSH 10.2p1 on this host — `ssh -G -vo StrictHostKeyChecking=no` prints `stricthostkeychecking false` (grouped short flags were never seen), `-o StrictHostKeyChecking=false` and `=NO` also print `false` (live aliases, case-folded), and `-F`, `ProxyCommand`, `KnownHostsCommand`, `Include`, `PermitLocalCommand`+`LocalCommand` were unknown to the filter entirely.
- FIX: `internal/config/sshargs.go` declares the ssh(1) short-option arity transcribed from the OpenSSH usage text, the short options AX admits (`4 6 C T a q v`, `i l p o`), and an `-o` registry with per-option value rules; `admitSSHArguments` walks argv with getopt semantics and refuses the complement, so an undeclared option fails closed. One refusal call site remains at `validation.go:465`, now carrying a closed reason vocabulary in the clause.
- DECISION: `StrictHostKeyChecking` declared with only the enforcing spelling `yes`, so `no`/`off`/`false`/`accept-new` and anything OpenSSH adds later are refused without listing aliases. Host-authentication option names are still declared in the registry — not to widen admission, but so their refusal reports the 6.3 clause instead of an unknown-name clause.
- FINDING: `-i` consuming the next argv as a filename (`ssh -G -i -oProxyCommand=id` sets no proxycommand) is faithful OpenSSH behavior and is admitted deliberately; a token scanner that "found" `-oProxyCommand=` there would diverge from what ssh actually does.
- FINDING: a derived per-letter loop goes quiet under the mutant that permits that letter — the subtest simply stops existing. Cited bypasses need an explicitly named assertion alongside the derivation; M1 (permit `-F`) reddens only because of it.
- REGRESSION: two pre-existing bound tests in `internal/config/schema_test.go` used `"xxxx…"` filler as `ssh_args`; that is a bare operand and is now refused, so they were rewritten to `-i<path>` argv of identical byte lengths.
- SCOPE: `internal/config/sshargs.go` (new), `internal/config/ssh_admission_test.go` (new), `internal/config/validation.go`, `internal/config/refusal_test.go`, `internal/config/schema_test.go`, `README.md`.
- REAPPLY: this leaf was reprovisioned onto a fresh Story workspace after an empty-OID checkpoint; the accepted patch was reapplied rather than reimplemented, and every gate and mutant below was rerun in the new workspace instead of inherited from the earlier run. The five cited OpenSSH behaviours were reproduced again against the live OpenSSH 10.2p1 binary on this host.
- STATUS: BUG-260902-beqfwr on `task-board/story/STORY-260902-2230n7`, one commit past checkpoint `6f77719`; `internal/config` coverage 93.7% (measured at the checkpoint) -> 93.9%; `go build ./...`, `go vet ./...`, `gofmt -l .`, `go test ./... -count=1`, `go test ./... -cover`, tracecheck, and the catalog `-check` gate all exit 0; ten single-clause mutants each redden their own cases; handed off to review.

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
