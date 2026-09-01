# Flight Logbook — TASK-260830-8x76g1 review, CR revision 7

> Reviewer RUN-260831-a35299 (claude/opus). Candidate tree 2101a330 over base ad727518.
> Recorded as a board resource rather than repo LOGBOOK.md: the reviewer role is read-only
> on repository content, and prior rounds used the same convention (review-logbook-rev2..4).

## 2026-09-01

### 0350 — Identity attests a Session Record name with no grammar at all
- FINDING: `closed_shapes.go:191` validates Session Record 1.0.0 `name` with a bare
  `requireString`. SPEC 5.1 declares it as the Section 2.3 grammar; SPEC.md:363 defines
  that term as 1-64 characters matching `[A-Za-z0-9][A-Za-z0-9._-]{0,63}`.
- FINDING: both `CalculateObjectIdentity` and `VerifyObjectIdentity` attest names holding
  a newline, a tab, `../../etc/passwd`, `$(rm -rf /)`, 1024 characters, 65 characters,
  a space, a leading hyphen, and non-ASCII text. Ten cases, all attested.
- ROOT CAUSE: not a stray field. `name` is absent from the code and absent from
  `TASK-260830-8x76g1_constraint-enumeration.md` in exactly the same place. The
  enumeration artifact that was added to prevent a hand-picked subset is itself a
  hand-picked subset for the one schema whose complete shape the candidate claims.
- SCOPE: internal/canonicaljson/closed_shapes.go; README.md; constraint-enumeration artifact.
- STATUS: pending — routed to `to-dev`.

### 0345 — Structural gates from prior rounds are genuinely real
- FINDING: removing one `register(...)` row panics at package init
  (`missing immutable-object shape validator for urn:ax:schema:migration-checkpoint@1.0.0`).
  The registry-derived totality fix from the rev4 review holds under mutation.
- FINDING: the fuzz-wiring check is AST-derived over every `_test.go`; an unwired
  `FuzzReviewMutantUnwiredTarget` fails `TestConfiguredValidationRunsEveryFuzzTargetWithFixedBudget`.
- FINDING: four NARROWING mutants (byte-vs-character bounds, BlobChunk index equality,
  extensions reverse-DNS grammar, symlink target bound) are each killed by a named test.
  The negative evidence is not delete-only.
- NOTE: recorded so rework does not churn the parts that already hold.

### 0340 — Byte-vs-character audit is correct in both directions
- FINDING: the rev2 F2 class is genuinely closed. Members SPEC declares in BYTES
  (Launch Plan argv elements, env_literals values, task_element_id) are implemented in
  bytes; members declared in CHARACTERS (media_type, symlink target, logical_id, goal_id,
  repository_identity, tree_identity, GitRemote name) count runes and accept multibyte
  values exactly at the bound. Bare `string` (17.3 schema_id) correctly gains no bound.
- NOTE: this is the first round where a previously-found class survived an independent
  cross-provider re-attack unchanged.

### 0335 — O(n^2) case-collision scan stalls the identity gate on a valid input
- FINDING: `validateManifestEntries` compares each entry path against every earlier one
  with `strings.EqualFold`. A contract-valid 65,536-entry `workspace_tree` manifest of
  3,211,786 bytes — inside the 5,242,880-byte cap — spends 6.61s in
  `CalculateObjectIdentity`. Scaling: 2000/17ms, 8000/162ms, 20000/875ms.
- NOTE: Transfer Manifests arrive from peers over Mesh RPC, so this is CPU amplification
  on untrusted input. Entries are already strictly sorted and unique, so a lowercased
  set makes the check linear with no loss of coverage.
- STATUS: pending.

### 0330 — Accepted-leaf content survived the workspace re-provision intact
- FINDING: `internal/scalar/{scalar,names,path,integer,time_digest,uuid,scalar_test}.go`
  are byte-identical to `TASK-260830-8x76g1_accepted-leaves-tree.tar.gz` (checkpoint
  828d633). This leaf only adds git.go, git_test.go, fuzz_test.go and testdata.
- NOTE: the disclosed workspace discard did not silently rewrite already-accepted work.
