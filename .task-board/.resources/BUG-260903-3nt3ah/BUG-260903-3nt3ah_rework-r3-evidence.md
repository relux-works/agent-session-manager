# BUG-260903-3nt3ah — rev3 rework evidence

Rework of reviewer Finding 1 on CR rev2. Reviewed tree `f9c468ba` (`dc69559`);
delivered tree `2b3cefec7a31952105703d0be04934bad0c03e0b` (`4c6ad14`, signed,
amend of `dc69559`, still exactly one commit past `6c690da`).

## Finding 1 — what was wrong

`canonical.go` intro said `Canonicalize` rejects "any document that **opens**
more than 256 containers". `maxNestingDepth` bounds containers **open at once**.
Driven through the real exported entry point:

| Document | Containers opened | Open at once | `Canonicalize` |
| --- | ---: | ---: | --- |
| `[[],[],…×400]` | 401 | 2 | `err=<nil>`, 1201 canonical bytes |
| `{"k0":{},…×400}` | 401 | 2 | `err=<nil>`, 3891 canonical bytes |
| 256 nested arrays | 256 | 256 | `err=<nil>` |
| 257 nested arrays | 257 | 257 | `nesting depth 257 exceeds maximum 256` |

The reviewer's measurement reproduced independently before changing anything.

## Both required parts

**1. The clause states the enforced bound.** "holds more than 256 containers
open at once", matching `maxNestingDepth`'s own wording at `canonical.go:22-23`.

**2. The noun is anchored, not just the digit.** The claim moved out of free
prose into the row form this file already uses for the rounding claim. Rows in
the doc comment:

```
400 sibling arrays open 401 containers, 2 at once: accepted
256 nested arrays open 256 containers, 256 at once: accepted
257 nested arrays open 257 containers, 257 at once: refused
```

`TestCanonicalizeDocumentedContainerLimitIsADepthNotACount` parses each row,
builds the document it names, **re-measures both quantities from the document
bytes** rather than recomputing them from the shape that produced them, and
drives the verdict through `Canonicalize`. Three coverage requirements must stay
satisfiable by the rows, each failing separately:

- an accepted row with exactly `maxNestingDepth` open at once — the limit is reachable;
- a refused row with exactly `maxNestingDepth+1` open at once — the limit is enforced;
- an accepted row opening **more than** `maxNestingDepth` containers while
  staying shallow — the fact that makes a count reading false.

The `"open at once"` qualifier is required at **every** occurrence of
`"256 containers"` in the free prose, not once. A presence check is satisfied by
a comment that states the clause correctly here and incorrectly there — the same
hole that let rev2's mutant N8 add a false SPEC clause beside the true one. Rows
are excluded from that scan because they are not prose: a row that misstates the
quantity fails on behaviour instead.

## Mutants — 10, each confirmed present in the source before measuring, all exit 1

| # | Mutant | Reddens | Exit |
|---|---|---|---:|
| R1 | doc reverts to the rev2 count wording | `…ContainerLimitIsADepthNotACount` claim check | 1 |
| R2 | sibling row verdict flipped to `refused` | row subtest + shallow-beyond-limit requirement | 1 |
| R3 | sibling row **deleted** (delete-only) | shallow-beyond-limit requirement | 1 |
| R4 | row lies about its own arithmetic, 401 → 402 | byte re-measurement | 1 |
| R5 | documented digit drifts 256 → 512 | claim check | 1 |
| R6 | **production narrowed into a genuine count bound**: refuse when `[`+`{` exceed `maxNestingDepth` | sibling row subtest | 1 |
| R7 | false count claim **added** beside the true clause | every-occurrence qualifier scan | 1 |
| R8 | rounding row drifts `1.0 -> 1.0` (parser regression control for the new row form) | `…RoundingIsWhatCanonicalizeActuallyDoes/1.0` | 1 |
| R9 | depth guard narrowed to `maxNestingDepth-1` | 256-nested row subtest | 1 |
| R10 | doc renames the test it cites | `…DocCommentNamesOnlyTestsThatExist` | 1 |

No survivors. R6 is the one that proves the noun in **behaviour** rather than in
wording: it keeps the documented number true under a count reading and reddens
only because the shallow row is driven. R7 is the added-lie shape a presence
check cannot see. R3 is delete-only and still killed, by a coverage requirement
rather than by an assertion about the deleted row.

Harness: `.temp/BUG-260903-3nt3ah/r3/mutate.py`. Each mutant is applied by
string replacement, its presence re-read from the file before the test runs, and
reverted from a backup copy taken before the sweep — never `git checkout`.
Post-sweep `diff` against the backups reports the sources byte-identical.

## Validation at the delivered tree, real exit codes

| Command | Exit |
| --- | ---: |
| `gofmt -l .` | 0 (empty) |
| `go vet ./...` | 0 |
| `go test ./... -count=1` | 0 — 11 packages ok, no FAIL |
| `go test ./internal/canonicaljson/ -count=1 -cover` | 0 — coverage 97.2%, unchanged |
| `go run ./internal/catalog/cmd/cataloggen … -check` | 0 |
| `go run ./internal/traceability/cmd/tracecheck` | 0 |
| `go test … -fuzz=^FuzzCanonicalizeRoundTrip$ -fuzztime=200x` | 0 |
| `go test … -fuzz=^FuzzObjectIdentityRepresentationInvariant$ -fuzztime=200x` | 0 |
| `curator status --check` | **1** — `worktree: go-testing-tools not-installed` |

`curator status --check` fails for the same pre-existing Story-worktree
provisioning reason reported in rev1 and rev2 and accepted by the rev2 review.
This diff touches no Curator-managed path. Reported failing, not explained away.

## Untouched

`internal/canonicaljson/canonical_test.go` is byte-for-byte unchanged, including
the Appendix B pin at lines 47-70. `internal/specdoc/` is unchanged. No
production logic changed: the `canonical.go` diff is comments only.

## Residual

Unchanged in kind and still real:

- Claims about RFC 8785 itself stay unmeasured. The RFC is not vendored, so
  `TestCanonicalizeMatchesEveryFiniteRFC8785AppendixBNumberSample` catches drift
  between this implementation and a **transcription** of Appendix B, not between
  the transcription and the RFC.
- Free prose outside the enumerated rows, entry-point names, citations and the
  container clause is still unchecked. Finding 1 was a false claim living in
  exactly that gap, which is now the second demonstration that the bound has a
  cost rather than being hypothetical. The treatment applied both times is to
  move the specific claim into a driven row, not to widen the disclaimer.
- The qualifier scan is anchored to the string `"256 containers"`. A rewrite
  that states a count claim without that exact substring — "opens more than 256
  sibling containers" — passes the prose scan. It cannot pass the rows: the
  shallow accepted row is still driven and still true, so the comment would
  contradict its own machine-checked block, which the reviewer can see. Naming
  the gap rather than claiming the class is closed.
- The call-graph residual is inherited from `productionCallGraph` and unchanged.
