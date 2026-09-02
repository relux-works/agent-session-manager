# Review verdict: CR-BUG-260902-lyhvkw-3 revision 3 — ACCEPTED

Reviewer run: RUN-260902-70cdd1 (reviewer archetype, bound to revision 3).
Candidate: base a351afd842a2, tree d75db085e284 (verified equal to `HEAD^{tree}` of
`task-board/story/STORY-260902-3os1kh` at 05b7d6f in the story workspace).
Revision 3 is the accepted revision-2 work reapplied onto advanced trunk; the
only non-mechanical delta versus rev 2 is the `reviewedOwnershipCanonicalSHA256`
repin to `9f7737cb…` (trunk had independently repinned) and a LOGBOOK entry.

## Gate attacked, not read

Production call site: `decodeValue` (`internal/canonicaljson/canonical.go:353`)
checks `depth >= maxNestingDepth` (line 362) before opening any container and
propagates `depth+1` on both the object (381) and array (395) branches. It is
reached only through `decodeStrict` (337), which is the sole decode path for
`Canonicalize` (160) and `prepareObjectIdentity` (195), i.e. both identity
entries. `grep` confirms no other `json.Unmarshal`/`json.NewDecoder` inside the
package; downstream `json.Marshal` + `jcs.Transform` and the recursive value
walks operate on the already-bounded decoded value, so no bypass path exists
around the gate for these entries. encoding/json's Token() mode has no depth cap
of its own (hence the original crash), so this gate is the only bound.

Mutant matrix (scratch copy, see `BUG-260902-lyhvkw_review-mutants-rev3.log`):
widen-512, narrow-128, widen-by-one (`>`), narrow-by-one (`-1`),
array-no-propagate, object-no-propagate, delete-gate — all seven exit 1.
Delete-gate and array-no-propagate reproduce the original
`fatal error: stack overflow` through the regression test, proving the test
drives the real crash shape. Tests pin the literal 256/257 locally, so widening
the constant reddens the suite (AC).

Adversarial probe through the three public entries (probe-01.log): whitespace
padded 257, deep container as a later sibling, deep value under a second object
member, 100,000-deep formerly-accepted input, truncated 300 opens, unicode keys,
5,000 brackets inside a string (must not count) and a 1 MB fan-out of 2,000
siblings at depth 256. Every refusal is typed `ErrInvalidJSON` at all three
entries; every at-limit shape is accepted by `Canonicalize` and reaches the
later identity shape gates (not the decoder) at the identity entries.

## Definition of Done check

- Declared constant with rationale: `maxNestingDepth = 256`, rationale in the
  doc comment; submodule bound 16 confirmed at `closed_shapes.go:1241`; deepest
  checked-in fixture nests 2, so headroom claim is not contradicted. ✔
- Refuse one past limit at all three entries with typed error: proven by
  `TestCanonicalizeRefusesDocumentPastMaxNestingDepth`,
  `TestIdentityEntriesRefuseDocumentPastMaxNestingDepth` (arrays/objects/mixed). ✔
- Accept at limit at every entry: `TestCanonicalizeAcceptsDocumentAtMaxNestingDepth`,
  `TestIdentityEntriesAcceptDocumentAtMaxNestingDepth` (decode passes, refusal
  comes from the later Section 1.6 depth-4 extension gate with `ErrInvalidIdentity`). ✔
- Regression: `TestNestingDepthRegressionTwoMegabyteArrayReturnsTypedError`
  replays the exact 2,000,000-byte one-million-level input at all entries. ✔
- Widening mutant reddens the suite: yes (and six more). ✔
- Fuzz seed `nesting-depth-over-limit` (257 deep) added and passes. ✔
- Ownership registration + digest repin: tracecheck full and sections exit 0. ✔
- Docs: README, constraint-enumeration row, LOGBOOK updated. ✔

## Gates rerun by this reviewer (worktree, candidate head 05b7d6f)

| gate | result |
| --- | --- |
| go build ./... | exit 0 |
| go vet ./... | exit 0 |
| go test ./... -cover -count=1 | exit 0 |
| go test ./internal/canonicaljson -cover | 87.2% (base a351afd: 87.1%) |
| fuzz seed FuzzCanonicalizeRoundTrip 100x | PASS |
| tracecheck -section 1.6 -section 10.1 / -section 17.3 / full | exit 0 / 0 / 0 |
| cataloggen -check | exit 0 |

Per-package coverage vs base a351afd: canonicaljson 87.1% -> 87.2%; every other
package identical (catalog 97.6, cataloggen 83.9, cmd/cataloggen 79.3, config
93.7, scalar 90.1, specpin 85.1, traceability 85.0, cmd/tracecheck 87.5). No
regression.

Logs: `.temp/BUG-260902-lyhvkw/review/{cand-canonicaljson-cover-01,base-canonicaljson-cover-01,repo-gates-01,base-repo-cover-01,tracecheck-catalog-01,probe-01}.log`.

## Notes for the orchestrator

- No findings. Accepted as-is; hand to the commit-owning mover for the `done`
  transition with `commit_ack=scope_committed`.
- Out of scope, observed only: other packages (`internal/scalar`, `specpin`,
  `cataloggen`, `traceability`) use `json.Unmarshal` on inputs that are not
  untrusted peer objects; the bug's stated boundary (SS1.6/SS10.1) is fully
  covered by this shared decode gate.
