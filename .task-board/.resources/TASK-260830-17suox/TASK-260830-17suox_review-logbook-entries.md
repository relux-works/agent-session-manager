# Flight Logbook

> Institutional memory. Concise, factual, high-signal.
> Newest entries first. One block per insight.

## 2026-09-01

### 1902 — The fix for a present-vs-absent ambiguity left the absent half untested
- CONTEXT: CR rev4 closes the rev3 round-trip defect by adding `restorePresentEmptyMaps` (internal/config/schema.go:480), a reflect walk that re-initializes a nil map member when the generic document shape shows the key was present. Correct, and the *present* half is pinned by two new tests.
- FINDING: The *absent* half is pinned by nothing. A narrowing mutant that initializes any nil map member before the source-presence check (schema.go:499) leaves `go test ./internal/config` fully green while making all four required closed map members admissible when omitted: `directory_installations[].extensions`, `directory_enrichment_profiles[].extensions`, `directory_peer_disclosure[].extensions`, `terminal.backend_config[].settings`. Probed through `Load`: refused on the candidate tree, `err=<nil>` under the mutant.
- ROOT CAUSE: The four `... required member` cases (internal/config/refusal_test.go:153-190) each omit six or seven members at once, so they refuse on an earlier disjunct and can never observe the closed-map disjunct being neutralized. A multi-omission negative case does not pin any single member.
- NOTE: General rule — when a change relaxes a presence check, the relaxation's *boundary* is what needs the new negative test, not the newly-accepted value.
- SCOPE: internal/config/schema.go:480-511, internal/config/validation.go:109-135
- STATUS: Pending — TASK-260830-17suox routed to `to-dev` on CR rev 4.

### 1901 — A bound fixture derived from the implementation constant proves only self-consistency
- FINDING: `schema_test.go:555-568` builds both the at-limit and past-limit extension payloads from `maxConfigExtensionBytes`, and self-checks against the same symbol. Mutating the constant 65_536 -> 655_360 left the package green: the fixture scales with the mutation.
- NOTE: SPEC.md §6 (pinned 28bf96d7, lines 350 and 2496) declares 65,536 bytes. Every neighbouring bound in the same suite uses a spec literal — 64/65, 1024/1025, 4096/4097, 32_766/32_767, 253/254 — and all of those kill their widening mutants. This one is the outlier.
- NOTE: Rule of thumb — a bound test may derive the *fixture shape* from the implementation, never the *bound value*.
- SCOPE: internal/config/schema_test.go:555, internal/config/validation.go:16
- STATUS: Pending — TASK-260830-17suox routed to `to-dev` on CR rev 4.

### 1900 — Deriving the refusal-site inventory from source is a gate that actually holds
- DECISION: rev4 makes `configError` a package var, wraps it in `TestMain` to record every call site reached, and after a full package run parses the non-test sources with go/ast to require that every `configError(...)` site was exercised — waivable only by an in-line `config-refusal-subsumed:` comment.
- FINDING: Attacked it by injecting an unreachable `configError` into `validateSync`; the package failed with `configuration refusal call sites without an exercised negative path: validation.go:509`. The one existing waiver (schema.go:468) is genuinely unreachable.
- NOTE: This closes the rev2 gap logged at 1814 (population and granularity) structurally rather than by enumerating clauses in a script. Caveat it does not close: reaching a refusal site is not the same as pinning the specific disjunct that fired — see 1902.
- SCOPE: internal/config/refusal_inventory_test.go, internal/config/schema.go:536
- STATUS: Method retained for future reviews.

### 1814 — A mutation sweep over error returns is not a sweep over clauses
- FINDING: `TASK-260830-17suox_clause-mutation-sweep-rework.log` reports "clauses found: 99 ... total survivors: 0 / 99". Every entry is a `return configError(...)` statement.
- ROOT CAUSE: Two gaps. (1) Population — the 19 bare `return ErrConfigValidation` sites in `validateExtensions`, `validateExtensionValue`, `validateSortedUniqueDigests`, `validateSortedUniqueClosed`, `validatePrintableCharacters` (internal/config/validation.go:735-836) are not enumerated at all. (2) Granularity — neutralizing a return kills the mutant when *any* test reaches it, so a composite guard `if A || B || C { return ... }` is one mutant and survives with two clauses unexercised.
- FINDING: An independent per-clause sweep over the same tree found 14 survivors, incl. `hasForbiddenConfigName` deletable entirely with the suite green (the §6.4 secret/credential/endpoint refusal), the printable/non-control rule, both `utf8.ValidString` checks, NUL-in-ssh-arg, extension reverse-DNS/key-length/float/array-depth, and the uniqueness half of both sorted-unique helpers (narrowing `>=` to `>`).
- NOTE: "0 survivors" was accurate for what it measured and was then carried into the task notes as satisfying the DoD clause gate. Proxy signal reported as the property.
- SCOPE: internal/config/validation.go:735-836
- STATUS: Pending — TASK-260830-17suox routed to `to-dev` on CR rev 2.

### 1812 — SSH bypass gate closed the separator spellings but not the value vocabulary
- FINDING: Rev 2 fixed the `=`/whitespace/quote separator gap from rev 1. The value vocabulary is still partial: `StrictHostKeyChecking=off`, `UserKnownHostsFile=none` and `GlobalKnownHostsFile=none` all load with `err=nil` through `config.Load`.
- ROOT CAUSE: The matcher compares against a hand-listed value set (`no`, ``, `/dev/null`, `nul`) rather than the OpenSSH value grammar. ssh_config(5) documents `off` as an alias of `no`, and `none` as the value that makes ssh ignore user-specific known hosts files.
- NOTE: The test is named `TestLoadRefusesEveryOpenSSHHostAuthenticationBypassSpelling` and enumerates nine cases — a name asserting a completeness the fixture list cannot carry. Same failure shape as rev 1, one layer in: separator was derived, value set was not.
- SCOPE: internal/config/validation.go `sshHostAuthenticationBypass`
- STATUS: Pending — TASK-260830-17suox routed to `to-dev` on CR rev 2.

### 1810 — A required config field that nothing reads gates nothing
- FINDING: `terminal.external_trust[].enabled` is decoded, presence-checked in `validateRawTerminalPresence`, cloned and re-emitted by `EncodeCurrent` — and never read by any production path.
- FINDING: `validateTerminal` does `registered[entry.BackendID] = struct{}{}` for every trust entry regardless of `Enabled`, so `backend_id = "com.example.term"` with that backend's only trust entry at `enabled = false` loads with `err=nil`.
- NOTE: §6.5 lists "untrusted executable" among the conditions that fail configuration/activation. Whether `enabled=false` is a config-time or activation-time refusal is a real design question — but the field currently reads as enforced and is inert.
- SCOPE: internal/config/validation.go `validateTerminal`
- STATUS: Pending — TASK-260830-17suox routed to `to-dev` on CR rev 2.

### 1741 — Self-referential completeness test hides a dropped capability
- FINDING: `terminalCapabilityRegistry` (internal/config/validation.go:24) hand-copies the 16-name terminal_backend capability set that is already pinned in `internal/catalog/catalog.v0.5.0.json` → `capability_families[family=terminal_backend].capabilities`.
- FINDING: The completeness test derives its expected set from that same variable, so it asserts the implementation equals itself.
- ANOMALY: Deleting `multiple_input_clients` from the registry left `go test ./internal/config` green — a closed capability name silently stops being accepted with no red test.
- NOTE: The test file already imports `internal/catalog` for the reader-version check, so the pinned source was available and unused.
- SCOPE: internal/config/validation.go:24, internal/config/schema_test.go
- STATUS: Pending — TASK-260830-17suox routed to `to-dev`.

### 1740 — SSH host-key bypass gate matched only the `=` spelling
- ROOT CAUSE: `sshHostAuthenticationBypass` (internal/config/validation.go:845) splits each `-o` option on `=` only. OpenSSH parses `-o` with the ssh_config line parser, which also accepts whitespace separation and quoted values.
- FINDING: Verified live with `ssh -G localhost` — `-o "StrictHostKeyChecking no"` and `-o 'StrictHostKeyChecking="no"'` both yield `stricthostkeychecking false`; `-o "UserKnownHostsFile /dev/null"` yields `userknownhostsfile /dev/null`. All admitted by `config.Decode`; only the `=` form is refused.
- NOTE: Spec §6.3 requires these to fail configuration validation. The single negative test covered one spelling of the gate — positive-path evidence for a refusal.
- NOTE: Normalize on the first `=` **or** whitespace and strip quotes before comparing; one negative case per admitted shape.
- SCOPE: internal/config/validation.go:845
- STATUS: Pending — TASK-260830-17suox routed to `to-dev`.

### 1739 — Per-clause disablement is worth running as a sweep, not per-gate
- DECISION: Neutralizing each `return configError(...)` one at a time (statement → `_, _ = <args>`, guard falls through) and re-running the package suite scales to a whole validator cheaply.
- FINDING: 95 clauses in `internal/config`; 43 reddened, 52 survived. Survivors were behaviorally correct when probed directly — the sweep isolates missing evidence, not broken code.
- NOTE: Untracked files do not appear in `git diff <tree>`; verify mutant restoration with `git hash-object <file>` against `<tree>:<file>` instead.
- SCOPE: internal/config/{validation,schema,writer}.go
- STATUS: Method retained for future reviews.
