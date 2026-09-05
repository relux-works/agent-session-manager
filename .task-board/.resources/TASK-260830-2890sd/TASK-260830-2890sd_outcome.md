# TASK-260830-2890sd outcome: implement-provider-discovery-and-trust

## What was built
New `internal/provider` package implementing SPEC Section 7.1 (production entry points `Discover`, `Trust`, `Verify` over an injectable `System` seam, production `OSSystem`):
- Deterministic discovery in section order: `providers.plugin_dirs` listed order (bytewise-sorted entries), six built-ins (codex/claude/gemini/muse/antigravity/pi in document order), PATH only when `allow_path_plugins`.
- Trust recording per accepted external `ax-provider-<id>`: canonical absolute path, SHA-256 digest, approving owner (`uid:` identity under an explicit `OwnerPolicy`; no implied superuser).
- Substitution detection: `Verify` re-resolves the undisguised discovery path; changed target/digest/owner/shape or any re-read failure fails with `integrity_failure` (renewed trust required). Forged receipts fail.
- Duplicate refusal: any two candidates sharing one ID fail with `invalid_config` naming both sources, before probe/execution (no probe/exec dependency exists; no source imports `os/exec`; duplicate fixture with a live script leaves its sentinel unexecuted).
- Fail-closed reads: malformed prefixed names, non-regular targets, unapproved owners, relative dirs refuse; partial reads abort with `local_precondition_failed`, never a narrowed set.
- Codes pinned to registry with exits: invalid_config=3, local_precondition_failed=3, integrity_failure=9 (verified through `axerror.ExitCodeFor` at 1.0.0).
- No capability/availability advertisement (AST-scanned); internal M0 boundary, not a public SDK (exported-symbol scan); no durable-state mutation (host persists `TrustRecord`).

## Evidence (all exit 0, run directly)
- `go test ./internal/provider/ -count=1 -v`: 34 top-level PASS (attached log).
- `go test ./internal/provider/ -cover`: 93.4% statements.
- `go vet ./...`, `go build ./...`, full `go test ./... -count=1` (all 14 packages ok).
- `GOOS=windows go build ./internal/provider/`: ok.
- `go run ./internal/traceability/cmd/tracecheck`: exit 0, unchanged 17/403 (ownership JSON deliberately untouched: hash-pinned, another scope owns it).
- Refusal-inventory gate (AST-derived, in-package): all constructor sites exercised, no stray `Error` construction, closed 3-code set observed. It caught one real gap pre-commit (unexercised Verify re-read path).
- Trust test caught a real double-hash bug pre-commit (fixed to single `SHA256Digest`).

## Bounds and disclosures
- Native Windows owner attestation unimplemented: externals undiscoverable there; builtins unaffected.
- Same file observed via two directories still refuses (literal MUST, no precedence override).
- `ax-provider-<id>` with invalid suffix refuses (intent must be loud); bare non-prefixed entries ignored.
- Executable bit is not a trust fact (undeclared by the section); trust-record persistence belongs to the host (future leaf).
- Manifest/protocol/matrices (7.2-7.7, 8) belong to sibling leaves TASK-260830-qcosxq / TASK-260830-32jeti.

## Commit
- Story branch `task-board/story/STORY-260830-3jqsx1`, one commit past checkpoint: `292fc6d` (signed, `git verify-commit` ok), worktree clean, nothing pushed (integration is the orchestrator step).
