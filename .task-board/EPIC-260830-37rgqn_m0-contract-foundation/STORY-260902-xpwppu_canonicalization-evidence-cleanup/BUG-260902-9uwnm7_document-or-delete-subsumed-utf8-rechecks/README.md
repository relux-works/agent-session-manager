# BUG-260902-9uwnm7: document-or-delete-subsumed-utf8-rechecks

## Description
Four utf8.ValidString re-checks are unreachable and undocumented, so they are neither killable by a mutant nor declared as subsumed - which the acceptance bar this board applies forbids in both directions.

decodeStrict validates UTF-8 once at internal/canonicaljson/canonical.go:311 and is the only producer of the maps that reach closed_shapes.go:258, :1506, :1525 and :1760. There is no external caller of the package on any branch, so no other path can deliver invalid UTF-8 to those sites.

The repository already has the convention this needs: internal/localstore/paths.go:234-236 and projection.go:300 document their subsumed re-checks in a comment naming the check that subsumes them. closed_shapes.go carries no such comment.

Consequence today: a clause sweep reports these as survivors that cannot be killed, which is indistinguishable from a real gap until someone re-derives the reachability argument by hand.

## Scope
Normative scope: §1.6 canonicalization.

## Acceptance Criteria
Each re-check either carries a comment naming decodeStrict as the subsuming validator, following the localstore convention, or is deleted. The clause sweep no longer reports them as unexplained survivors. If the package ever gains an entry point that bypasses decodeStrict, the argument breaks - so whichever resolution is chosen states that dependency explicitly.
