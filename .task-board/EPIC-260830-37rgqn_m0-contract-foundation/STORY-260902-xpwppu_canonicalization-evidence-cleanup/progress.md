## Status
done

## Review
required

## Task Class
code

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
(empty)

## Notes
Two independent mechanical gaps, batched because both are small, both touch internal/canonicaljson, and parallel Stories would conflict on the same files. Each is independently verifiable and must be independently proven.

PART 1 - PIN THE TWO UNPINNED NORMATIVE DIGESTS (BUG-260902-1874eo)
SPEC.md:303-304 publishes two SS1.6 digest fixtures that no .go or .json file in this repository recomputes:
  bb80eb37... NUM-U64-STRING  = sha256 of {'n':'9007199254740992'}
  b0ec84c6... NUM-U64-MAX     = sha256 of {'n':'18446744073709551615'}
Only NUM-SAFE-MAX (e1da48c6...) is pinned today, at internal/canonicaljson/canonical_test.go:513.
Required: assert both digests by recomputing them from the canonical encoder output, next to the existing NUM-SAFE-MAX assertion and in the same shape. The decimal_uint64 SYNTAX is already tested; what is missing is the normative DIGEST claim.
Take the expected values from the PINNED spec, not from this note and not from a live fetch of the spec repository default branch. Extract SPEC.md at the pinned commit, verify its SHA-256 against internal/specpin, then read lines 303-304 and quote them in the test comment.
Anti-vacuity: prove each new assertion fails if the encoder output changes by one byte. A digest assertion that passes because both sides are computed the same wrong way proves nothing.

PART 2 - DOCUMENT OR DELETE THE SUBSUMED UTF-8 RE-CHECKS (BUG-260902-9uwnm7)
Four utf8.ValidString re-checks are unreachable and undocumented, at internal/canonicaljson/closed_shapes.go:258, :1506, :1525, :1760.
decodeStrict validates UTF-8 once at internal/canonicaljson/canonical.go:311 and is the only producer of the maps that reach those four sites; the package has no external caller on any branch.
The repository already has the convention: internal/localstore/paths.go:234-236 and internal/canonicaljson/projection.go:300 document their subsumed re-checks with a comment naming the check that subsumes them. Follow that existing convention exactly rather than inventing a new comment style.
Choose per site, with the reason stated: keep it with a subsumption comment naming decodeStrict and canonical.go:311 as the subsuming validator, or delete it. Deleting is acceptable ONLY where you can show no reachable caller can deliver invalid UTF-8; if reachability is uncertain, keep and document.
Do not add a test that fakes reachability to make a dead branch look covered. That is the exact anti-pattern this board refuses.

WHY THIS MATTERS
Both parts are about a claim nobody checks. A published digest that nothing recomputes is a contract nobody verifies, and an unreachable branch that is neither killable nor declared is indistinguishable from a real gap until someone re-derives reachability by hand.

STANDING CRITERIA FOR THIS BOARD
- Drive real production entry points; no test-only helper as the subject.
- Kill mutants, and report the decisive mutant for each change, not just a count.
- Narrow a mutant rather than deleting it when it is equivalent; state equivalence explicitly.
- Never weaken or delete an existing assertion to make new work pass. If an existing test blocks you, say so and stop.
- Quote the pinned spec verbatim for any normative claim, with file and line.
- Disclose anything you found but did not fix.


## Precondition Resources
(none)

## Outcome Resources
(none)

## Created
2026-09-02T19:49:43Z

## Last Update
2026-09-02T21:41:35Z
