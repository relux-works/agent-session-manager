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

## 2026-09-02 — BUG-260902-2faftr: a name blacklist is not secret detection

`hasForbiddenConfigName` refused any `extensions` key, and any nested extension
object key at any depth, containing a `secret`/`token`/`password`/`credential`/
`auth`/`environment`/`env`/`endpoint` part. The pinned v0.5.0 spec states no
such rule: `SPEC.md:345-347` defines an extension key as the reverse-DNS grammar
and reserves no label, and `SPEC.md:347-349` constrains an ExtensionValue object
only to string keys within nesting depth 4, so the nested-key arm had no source
at all.

The finding worth keeping is the failure shape, not the deletion. The rule read
like secret exclusion and was presumably written for `SPEC.md:2596-2597`
("secret, token, endpoint credential ... is forbidden") — but that clause is
scoped to a terminal backend-config `settings` object two sentences earlier, and
the implementation applied it to key *names* in a different section. It
inspected zero values, so `com.example.deploy = "AKIA..."` passed while
`works.relux.env-tools` and `com.example.auth-manager` — namespaces this
organisation owns — were refused. It was pure false-positive surface with no
true-positive capability.

It survived a review loop because `refusal_test.go` pinned it: the loop
hardened the invented constraint instead of asking which spec line declared it.
A refusal test asserts a gate is reachable; it says nothing about whether the
gate should exist. Reviewing a refusal now means quoting its declaring spec line
before accepting its test.

The two clauses that do exist were already enforced where they apply, and both
are now pinned by a test rather than left implicit. `SPEC.md:2596-2597` is
enforced by the closed `BackendSettingsValidator` schema a backend
implementation version registers. `SPEC.md:2562-2563` ("No v2 table accepts a
secret, endpoint credential, model token, auth root, or arbitrary environment
passthrough") — the §6.4 clause an earlier review cited when it demanded
coverage for the blacklist — is enforced by the closed table shape
`decodeStrict` decodes with `DisallowUnknownFields`, which is a field-declaration
rule, not a key-spelling rule. `SPEC.md:2344-2345` draws the same line for
values and goes further: "Secret values MUST NOT be accepted in config fields; a
provider MAY name a machine-local environment variable or credential profile."
The specification explicitly permits a config field to *name* a credential
profile, which is exactly what the removed rule refused.

Same class as BUG-260902-beqfwr (`ssh_args` name blacklist -> derived
admission): where the pinned spec is silent, be permissive; where it names an
equivalence class, derive admission rather than enumerate refusals.

A second finding came out of the review of the reapply, and it is the more
transferable one. README and the commit message both claimed that a nested key
named `endpoint` or `token` is "preserved as data". The tests asserted only that
such a document is *admitted*. Those are different properties: the reviewer
built a mutant that admits the document and then silently deletes every nested
`secret`/`token`/`password`/`credential`/`auth`/`env`/`environment`/`endpoint`
key from the loaded map, and it passed the entire repository suite, `go vet`,
full and scoped `tracecheck` and `cataloggen -check` — reproduced here at exit
0. Nothing in the tree read a nested value back, so nothing could notice.

This is the same shape as the defect the change itself removed, mirrored: there,
a test pinned a rule the specification never declared; here, an artifact
declared a property no test pinned. An admission assertion is evidence about a
gate, never about what survives the gate. The fix is cheap and general — assert
the round trip, not the verdict: `TestExtensionValueObjectKeysArePreservedAsData`
loads each previously blacklisted label through the production
`loadConfigDocument` entry and compares the re-encoded extensions map
byte-for-byte, so a dropped, renamed, retyped or rewritten nested key fails.
`encoding/json` sorts object keys, which is what makes byte identity a usable
assertion here rather than a brittle one.

- PROVENANCE: this change was produced and reviewed to acceptance on BUG-260902-3ru6vw, whose element was discarded when its parent Story was deleted while a move chain still referenced it. The accepted patch was reapplied here, not reimplemented. It did not apply to current trunk unchanged: `BUG-260902-beqfwr` had since landed and removed `parseSSHConfigOption`, so the helper-removal hunk and the README insertion point both had to be relocated by hand. The code and test deltas are byte-identical to the accepted patch.
- STATUS: BUG-260902-2faftr, on `task-board/story/STORY-260902-2oiugz` (forked from trunk `67aed0b`, zero commits behind `main` at measurement time); `internal/config` coverage 94.4%, against a 93.9% pre-change measurement taken from a `git archive HEAD` copy (the preservation tests raise it; no statement lost coverage); `go build ./...`, `go vet ./...`, `gofmt -l ./internal`, `go test ./... -count=1`, `go test ./... -cover -count=1`, full and assigned-scope `tracecheck`, and `cataloggen -check` all exit 0; the seven single-clause mutants were re-run on this tree and each exited 1, counting `--- FAIL` subtest lines only (re-added key blacklist reddens 17 of the 20 admission subtests, reproducing the accepted review's measurement exactly; re-added nested-key blacklist 10; dropped grammar gate 11; widened 253-byte key bound 2; widened depth bound 2; dropped registered backend-settings schema call 2; dropped `DisallowUnknownFields` 5), with the tree restored green after each. An eighth mutant closes the review's non-blocking finding: deleting every blacklisted-label nested key from the loaded map passes the whole suite on the previous revision (exit 0) and reddens all 11 preservation subtests here (exit 1). With the preservation tests present the two required narrowing mutants redden more, not less: re-added key blacklist 18, re-added nested-key blacklist 21. Handed off to review. No commit hash is named here: this entry ships in the commit it would describe, so the hash cannot be known when it is written.
