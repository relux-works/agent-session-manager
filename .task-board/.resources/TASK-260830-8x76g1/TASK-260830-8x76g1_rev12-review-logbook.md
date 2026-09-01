# Flight Logbook — TASK-260830-8x76g1 rev12 review (RUN-260901-34271e)

> Attached as a board resource rather than a repository file: this is a
> read-only reviewer run and the candidate tree must stay byte-identical.

## 2026-09-01

### 0624 — Symbol-referenced boundary tests are tautological
- FINDING: A boundary case written as `blobDescriptorWithChunks(maxBlobChunks)` /
  `blobDescriptorWithChunks(maxBlobChunks + 1)`
  (`internal/canonicaljson/boundary_constraints_test.go:137-138`) pins only the
  relationship between the production call site and the constant. It never pins
  the declared value. Mutating the constant declaration
  `maxBlobChunks = 32_768` -> `32_769` at `internal/canonicaljson/closed_shapes.go:23`
  leaves the entire repository suite green.
- ROOT CAUSE: Gate and expectation move together, so the assertion is
  `production == production` and cannot fail.
- FINDING: The sibling constant is safe by accident, not by design.
  `maxChunkSize` is independently pinned by the literal `4194304` in
  `internal/canonicaljson/canonical_test.go:693-694`, so its constant mutant
  dies. Nothing pins `32768` anywhere.
- DECISION: A mutation sweep that only edits call sites cannot detect this
  class. The rev11 harness mutated `"chunks", maxBlobChunks` ->
  `maxBlobChunks+1` and correctly reported KILLED, which is why two cycles
  passed over the constant. Sweeps must enumerate constant *declarations* as
  first-class mutation targets.
- NOTE: General rule for boundary tests — express the expected bound as an
  independent literal in the test. Referencing the production symbol converts a
  boundary proof into a tautology.
- SCOPE: `internal/canonicaljson/closed_shapes.go:23`,
  `internal/canonicaljson/boundary_constraints_test.go:137-138`,
  `internal/canonicaljson/testdata/constraint-enumeration.md:27`
- STATUS: pending — routed to `to-dev` as test-only rework.

### 0618 — A multi-clause refusal needs one negative case per clause
- FINDING: `symlinkTargetEscapes` (`internal/canonicaljson/closed_shapes.go:844-849`)
  refuses a materialization-root escape on four independent grounds: embedded
  NUL, `/`-absolute target, backslash in target, and a Windows drive-relative
  `X:` prefix. The only symlink-escape negative case in the whole repository is
  `"../escape"`. Only the backslash clause was additionally covered.
- FINDING: Applied together, mutants that neutralize the NUL clause, the
  absolute clause, and widen `len(target) >= 2` to `>= 3` make
  `CalculateObjectIdentity` **attest** a Transfer Manifest whose symlink target
  is `/etc/passwd`, contains a NUL, or is a bare `C:` — while
  `go test ./...` exits 0 across all eight packages.
- NOTE: Coverage percentage hides this completely. The function is executed by
  the `../escape` case, so every line reports covered while three of four
  refusal grounds are unproven. Per-clause negative cases, not line coverage,
  are what pin a disjunctive guard.
- SCOPE: `internal/canonicaljson/closed_shapes.go:844-849`,
  `internal/canonicaljson/canonical_test.go:1119`,
  `internal/canonicaljson/fuzz_test.go:256`
- STATUS: pending — routed to `to-dev` as test-only rework.

### 0610 — Loop-guard off-by-one leaves two-element ordering unproven
- FINDING: Strict-sortedness guards written `if index > 0 && value <= previous`
  survive widening to `index > 1` at `closed_shapes.go:762` (Transfer Manifest
  entry paths) and `:882` (WorkspaceSnapshot member IDs): the first comparison
  is skipped, so a two-element out-of-order array is accepted silently. The five
  equivalent guards at lines 1034, 1172, 1599, 1851 and 1866 are all killed.
- NOTE: Ordering corpora built from three-or-more-element arrays never exercise
  the first comparison. A two-element unsorted case is the minimum that does.
- STATUS: pending — routed to `to-dev` as test-only rework.

### 0559 — Production is correct on all three findings
- NOTE: Every finding above is a negative-evidence gap, not a defect. Unmutated
  production refuses all of it correctly: 32769 chunks with
  `member chunks exceeds maximum length 32768`, and each escaping symlink target
  with `symlink ManifestEntry[0] target escapes the materialization root`.
- DECISION: Verdict is `changes_requested` -> `to-dev`, test-only. Production
  `closed_shapes.go` and the pinned v0.5.0 specification must not be edited.
- MILESTONE: All configured validation gates re-run in the foreground at
  candidate tree `af4f5e32` and green, including all four `-fuzztime=100x` fuzz
  gates and `tracecheck -section 17.3` (`assigned_scopes=1`).
