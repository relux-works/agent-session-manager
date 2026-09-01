# Flight Logbook — TASK-260830-8x76g1 reviewer, CR rev 14

> Repository has no `LOGBOOK.md` and this run is read-only against candidate
> tree `0ca65322a7bf171b8e51bef353250a35d4f094f5`; creating one would have moved
> the tree. Recorded as a board resource, matching the rev2/rev3/rev7/rev13
> reviewer-logbook convention on this task.

## 2026-09-01

### 0817 — Numeric-bound mutation harnesses are blind to tagged-union and predicate gates
- FINDING: The rev12 `genmutants.py` harness this task has used for four cycles
  derives mutants only from regex quantifiers, numeric comparisons,
  helper-call limits, two constants and three named zero/empty guards. It
  cannot express a mutant for a tagged-union arm, a cross-field equality, a
  grammar predicate, or a case-collision predicate. Four consecutive cycles of
  "zero actionable survivors" therefore measured one axis of the gate.
- FINDING: A complementary sweep that disables every refusal clause
  individually (110 sites, `if <cond> {` -> `if false && (<cond>) {`, uncached
  `go test` each) returns 65 KILLED / 45 SURVIVED on the same tree.
- SCOPE: `internal/canonicaljson/closed_shapes.go`
- STATUS: pending — routed to `to-dev` with the twelve confirmed sites.

### 0810 — Both public identity entries attest case-colliding submodule siblings under one narrowing mutant
- FINDING: `closed_shapes.go:1231` refuses duplicate-or-case-colliding sibling
  `GitSubmodule` paths. Narrowing it from `strings.EqualFold(pathValue,
  previous)` to `pathValue == previous` keeps exact-duplicate refusal and drops
  only the case-collision half. `CalculateObjectIdentity` and
  `VerifyObjectIdentity` then both attest a Transfer Manifest carrying sibling
  submodules `vendor/lib` and `vendor/LIB`, and the whole
  `internal/canonicaljson` suite still exits 0.
- FINDING: The equivalent site for WorkspaceSnapshot group paths
  (`closed_shapes.go:888`) IS pinned — the same mutant there is KILLED — and
  `simpleFoldKey -> identity` is KILLED. The mechanism is covered; the
  per-site case is not.
- NOTE: `constraint-enumeration.md:220` claims "sibling paths do not
  case-collide | Enforced by `validateGitSubmodules`". Production is correct;
  the claim outruns the suite.
- SCOPE: `internal/canonicaljson/closed_shapes.go:1231`
- STATUS: pending

### 0755 — rev13 trailing zero-size BlobChunk finding closed and independently confirmed
- FIX: `TestTrailingBlobChunkSizeMinimumReachesBothIdentityEntries`
  (`boundary_constraints_test.go:336`) reaches `closed_shapes.go:614` instead of
  being short-circuited by the empty-descriptor rule.
- FINDING: Under `if size == 0 {` -> `if false && size == 0 {` it is the SOLE
  failing test in the package, so the clause is pinned by exactly the case that
  claims to pin it.
- FINDING: The 71-mutant derived set regenerates byte-identically from the
  current production file, and an independent sequential runner reproduces the
  producer's 60 KILLED / 15 SURVIVED split over 75 mutants.
- SCOPE: `internal/canonicaljson/boundary_constraints_test.go`,
  `internal/canonicaljson/testdata/constraint-enumeration.md`
- STATUS: resolved

### 0742 — AST-derived enumeration test masks six unpinned per-site closed shapes
- FINDING: Renaming `requireExactMembers` to an open variant at one call site
  is KILLED at all 24 sites in the real suite, but 6 of them
  (`:340`, `:437`, `:780`, `:792`, `:804`, `:933`) survive once
  `TestConstraintEnumerationMatchesRequireExactMembers` is skipped.
- DECISION: Not a finding. The AST test derives the call set and member lists
  from production source and requires an exact one-to-one artifact mapping, so
  it kills those mutants for the right reason, and the closedness mechanism is
  separately pinned by 9 fuzz seeds that fail when the unknown-member refusal in
  `requireExactMembers` itself is disabled.
- SCOPE: `internal/canonicaljson/constraint_inventory_test.go`
- STATUS: resolved
