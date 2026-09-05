# TASK-260830-2xdt8t — terminal backend registry: verification log

Task: implement-terminal-backend-registry (STORY-260830-3m2mw8).
Scope: spec v0.5.0 §4.B, §4.1, §6.5, §7.A. Manifest/Probe schemas, capability
evidence, lifecycle ops, and legacy v0.4.3 translation are sibling tasks and
were deliberately not implemented here.

## Changes

- NEW `internal/terminalbackend/terminalbackend.go`
  - `ParseID`: canonical ID grammar (1–128 ASCII bytes,
    `[a-z][a-z0-9]*(?:[.-][a-z0-9]+)*`) + `ax.` namespace reservation
    (only `ax.tmux`, `ax.conpty`).
  - `New`: admits canonical built-ins (`ax.tmux` on macos/linux/wsl2,
    `ax.conpty` on windows, `builtin_go`, null digest) at an explicit
    implementation semver + sorted-unique major-1 protocol list.
  - `RegisterExternal`: trust admission (enabled, absolute path only,
    digest equality before any probe, external kinds only, ID binding);
    duplicate ID → `terminal_backend_ambiguous`; any drift →
    `terminal_backend_implementation_drift`; trust failures →
    `terminal_backend_untrusted`. Refused writes never mutate the record.
  - `Resolve`: unknown/malformed → `terminal_backend_not_found` (read
    failure is never absence; no default fallback).
  - `DefaultForPlatform`, `RequireRestoreBinding` (exact prior binding;
    default substitution → `terminal_backend_restore_mismatch`).
  - `CheckVersionTuple`, `CheckProviderDescriptor` (§7.A; generation
    bound 1..256 bytes, 0/257 refused, 256 accepted).
  - `DigestFile`: symlink-resolving, regular-file-only sha256 trust digest.
  - All six wire codes verified against the pinned catalog error vocabulary
    in `TestWireCodesAreCatalogued`.
- EDIT `internal/config/validation.go`: `terminal.backend_id` and
  `terminal.external_trust[].backend_id` now validated by
  `terminalbackend.ParseID` (production call site); external trust entries
  claiming any `ax.` ID refused as ambiguous. Removed the superseded local
  `terminalBackendIDPattern`.
- NEW `internal/config/terminal_registry_wiring_test.go`: reserved-namespace
  refusals + built-in admission through `EncodeCurrent`.

## Gates run standalone (no pipes), real exit codes

- `go vet ./internal/terminalbackend/` → exit 0
- `go test ./internal/terminalbackend/ -v` → exit 0 (all pass)
- `go vet ./internal/config/ ./internal/terminalbackend/` → exit 0
- `go test ./internal/config/ ./internal/terminalbackend/` → exit 0
- `gofmt -l internal/` → clean (two files reformatted, then re-verified)
- `go vet ./...` → exit 0
- `go test ./...` → exit 0 (all 14 packages ok)
- `go test ./... -cover` → exit 0; terminalbackend 89.3%, config 94.7%

## Negative-evidence notes

- Every gate is narrowed, not just deleted: grammar cases widen exactly one
  character/separator/bound rule; reservation cases match the grammar but use
  the reserved namespace; drift cases change exactly one record member.
- `TestErrorPredicatesAreExclusive` proves each refusal carries exactly one
  wire code.
- Idempotency: identical re-registration is refused (`ambiguous`), and both
  duplicate and drift refusals assert the admitted record is byte-identical
  afterwards — a retried admission cannot mask a drifted one.
- Registry is process-local memory; no durable state is mutated, so no
  crash-recovery log applies. The duplicate/drift non-clobber assertions are
  the idempotency evidence.
- No new capability advertised: wire codes are a subset of the pinned catalog
  vocabulary; no CLI/doctor surface added.

## Board-disclosed findings decided

1. `required_capabilities` default vs "platform lane minimum" (SPEC.md:2585).
   DECISION: keep the current empty default; fixing it by inventing a set
   would violate the no-invented-constraints rule. Evidence: "platform lane
   minimum" occurs exactly once in the 12,665-line SPEC (line 2585) and no
   per-platform minimum set is enumerated anywhere. The empty default
   enables nothing, satisfying SPEC.md:2604-2605 ("Policy may further
   restrict capabilities/transports but cannot enable an unsupported
   claim"), while per-operation capability dependencies (§4.C/§4.D) still
   gate activation. No code change.
2. §6.2 "On native Windows, ... terminal backend MUST be conpty"
   (SPEC.md:2415-2418). DECISION: enforced, and now refusal-pinned on both
   arms. The v3 lane split was already refused in both directions
   (ax.tmux-on-Windows Decode case; ax.conpty-off-Windows refusal case).
   Added `TestDecodeRefusesNonConptyBackendOnNativeWindows`: a v1
   `backend="tmux"` document on Windows refuses with ErrConfigValidation
   through Decode, and `backend="conpty"` loads to `ax.conpty`. The rule
   was deliberately NOT extended to third-party registered IDs: §6.5
   admits them as "valid registered ID" with manifest-bound platforms, so
   a blanket Windows refusal would invent an undeclared constraint.
