# TASK-260830-2890sd verification evidence (developer re-verification, 2026-09-05)

HEAD verified: d8fc669 (`TASK-260830-2890sd: implement-provider-discovery-and-trust`), branch
task-board/story/STORY-260830-3jqsx1, worktree clean before and after. All commands below
were run directly in this session as standalone processes (no pipes through tee).

## Gates run in this session (all exit 0)

- `go test ./internal/provider/ -v` — all tests PASS, coverage 94.1% of statements.
- `go test ./...` — all 14 packages ok (axerror, canonicaljson, catalog, cataloggen,
  catalog/cmd/cataloggen, cliresult, config, localstore, provider, scalar, specdoc,
  specpin, traceability, traceability/cmd/tracecheck).
- `go vet ./...` — exit 0.
- `go build ./...` — exit 0.
- `go run ./internal/traceability/cmd/tracecheck` — exit 0;
  `contracts=60 normative_sections=36 acceptance_cases=74 fixtures=30
  compatibility_contracts=55 assigned_scopes=0`,
  `clauses_discharged=17/403` (unchanged: S7.3 manifest remains unmeasured, sibling scope).
- `go generate ./internal/catalog` + `git diff --exit-code` — exit 0, no diff.

## Independent attack probe (temporary in-package test, removed afterwards)

Drove the real production entry points Discover/Trust/Verify through an independently
written System implementation (not the repo's fake). 13/13 checks passed:

- discover-ok, foo-found, digest-present
- single-hash-not-double: recorded digest == sha256(bytes) hex, != sha256(sha256(bytes)) —
  confirms the double-hash fix end to end (trustCandidate calls scalar.SHA256Digest once;
  Verify calls sha256.Sum256 once + subtle.ConstantTimeCompare).
- trust-ok, trust-pure (identical inputs, identical receipts), trust-refuses-builtin
  (invalid_config)
- verify-unchanged accepts; verify-refuses-changed-bytes (last-byte flip, integrity_failure)
- verify-refuses-owner-change-to-approved-admin: same bytes, owner moved to an approved
  administrator UID — still integrity_failure, narrowing the owner dimension (not just digest)
- verify-refuses-retarget: same bytes at a new canonical path — integrity_failure
- duplicate-refuses-before-probe: two plugin dirs, error is invalid_config naming both
  plugin_dirs[0] and plugin_dirs[1]; Discover takes no probe/execution callback so refusal
  is structurally before any probe or execution
- duplicate-refuses-byte-identical: same file observed through two dirs still refused

## Reviewer-context items checked

1. Re-read path test (TestVerifyTreatsReadFailureAsIntegrityFailure/unreadable_target,
   trust_test.go:233): correctly scripts contentErr AFTER trust, so it drives the Verify
   re-read at provider.go:467, not the digest-mismatch path. The previously caught
   nil-content variant is gone.
2. Ownership JSON deliberately untouched (hash-pinned, sibling scope owns it); gate
   unchanged at 17/403 — disclosed, not hidden.
3. Stated bound: Windows owner attestation seam (os_windows.go) untested on this host;
   named in code/README, not claimed.

## Crash/idempotency

Package holds no durable state of its own (Trust returns a value the host persists;
Verify is a pure recheck; Discover is deterministic over an unchanged tree), so there is
no crash/idempotency surface inside this package — stated in doc.go and README.
