# TASK-260830-8x76g1 — rework revision 4 logbook

- Pinned source read locally from spec commit
  `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`; no unpinned specification text
  was used.
- CR revision 3's failure class was broader than its seven examples: member
  names were closed, but most nested values, unions, ordering and cross-field
  rules were not validated.
- Expected-red evidence was captured before implementation. The production
  entry test exited 1 because 21/21 malformed nested candidates were attested.
- Reused `internal/scalar` for path, UUID, timestamp and digest rules; added
  production scalar types for Git OIDs, refs and sanitized URLs rather than
  embedding looser regular expressions in `closed_shapes.go`.
- Kept candidate-local validation separate from facts that require referenced
  bytes: Blob Descriptor equality, child-manifest closure, raw Git pack/index
  agreement, isolated object-database resolution, and filesystem resolution.
  README now states that boundary explicitly.
- The recursive submodule validator now checks initialized/uninitialized state,
  TM-GIT parent index linkage, object formats, depth 16, total 256, ancestor
  cycles and sibling path collisions. Exact depth/count positives and narrowed
  negatives drive both identity entries.
- Every post-fix test/fuzz/trace/build/vet/cross-compile gate exited 0. Package
  coverage is 90.1% scalar and 81.5% canonicaljson.
- `task-board validate` exited 0 while printing 262 inherited
  `MISSING_ACTIVITY` diagnostics. `TASK-260830-8x76g1` is not among them.
- No external blocker, product decision, durable mutation, doctor output, or
  capability advertisement was introduced.
