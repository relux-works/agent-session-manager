# BUG-260902-3dn8jd — review verdict, CR rev3

**Verdict: accepted.** Change Request `CR-BUG-260902-3dn8jd-3` revision 3,
base `422786cc`, candidate tree `faa92b69`, `repository_delta=present`.
Worktree `HEAD` tree verified equal to the candidate tree OID and `HEAD~1`
equal to the base OID — one commit past the checkpoint, clean tree.

This review re-derived the producer's central claims independently and attacked
the gate with narrowing mutants rather than reading it. Every attack below was
executed in a throwaway `git archive` copy of the candidate tree; the reviewed
worktree was not modified.

## The pin is real, and it is the pre-existing pin

| Fact | Evidence |
| --- | --- |
| Embedded document digest | `shasum -a 256 internal/specdoc/SPEC.md` = `562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a` |
| Pinned constant | `internal/specpin/pin.go:30 DocumentSHA256` = same value |
| Lockfile | `internal/specpin/v0.5.0.lock.json` `source.document.sha256` = same value |
| `internal/specpin` touched by this CR? | No — not in the 10 changed paths |

The producer could not have moved the goalposts: the digest the comparison is
held to was already in the tree before this change.

No live fetch: `specdoc` uses `//go:embed` and imports no network package.
Import scope: `grep -rl "internal/specdoc" --include="*.go"` outside the package
returns only `_test.go` files, so the 883 KiB document never reaches `ax`.

## The gate reddens on the shipped artifact, not only on copies

The producer's plants mutate a `t.TempDir()` copy. I planted the incident defect
into the **real** `testdata/constraint-enumeration.md` and ran the shipped gate:

```
--- FAIL: TestConstraintEnumerationSpecExcerptsQuoteThePinnedSpecification
    Lease Record.reason: entry 1 (L1908) quotes text absent from the pinned SPEC.md:
      "... a digest of the lease reason ..."
    Lease Record.reason: no entry names the member; ...
```

Row named, absent text named. AC "planting a row with an invented quote reddens,
proven by doing it" is satisfied against the production artifact path.

## Narrowing mutants — nine executed, nine red

| # | Mutant (narrowing, not deletion) | Result |
| --- | --- | --- |
| M2 | digest comparison short-circuited to `false && …` | RED — `specdoc` 7 perturbation cases + `TestSpecExcerptComparisonRefusesASwappedSpecification` |
| M3 | `containsLine` returns true (line anchor neutered, presence kept) | RED — `…/true_quote_at_the_wrong_line` |
| M4 | `containsSection` returns true (clause anchor neutered) | RED — `TestClauseAnchorRefusesEveryForeignSectionForOneRow` + `…/true_quote_from_another_shape's_clause` |
| M5 | `declaringRowFailure` returns no failure | RED — `…/true_quote_from_a_sibling_member's_table_row` + 8 `TestDeclaringRowAnchorRefusesEverySiblingRowOfTheGitTable` subtests |
| M6 | `anchored := true` (member anchor pre-satisfied) | RED — `…/true_quote_that_never_names_the_member` |
| M7 | table-row hard boundary removed, blank-line boundary kept | RED — `TestQuoteLinesRefusesTextThatSpansTwoTableRows` + `…/quote_stitched_across_a_table_row_boundary` |
| M8 | blank-line hard boundary removed, table-row boundary kept | RED — `TestQuoteLinesRefusesTextThatSpansABlankLine` + `…/quote_stitched_across_a_blank_line` |
| M10 | one whole row deleted from the artifact (bypass by omission) | RED — `TestConstraintEnumerationMatchesRequireExactMembers`; the pre-existing artifact↔code derivation is preserved and still load-bearing |
| M11 | `constraintRowDeclaringIdentifiers["GitIndex"]` removed | RED, and **fails closed**: refuses 6 shipped `GitIndex` rows rather than granting a free pass |

M7 and M8 are the pair that matters: each keeps one boundary and removes the
other, so neither is proven by the other's test.

## Independently re-derived census

I wrote a throwaway probe over the shipped artifact and the verified document:

| Metric | Producer's LOGBOOK claim | Reviewer measurement |
| --- | ---: | ---: |
| Rows | 347 | 347 |
| Citations | 357 | 357 |
| Citations on table body rows | 235 | 235 |
| Citations naming another type | 0 | 0 |
| Citations naming another member (exempted) | 2 | 2 |

Doc/artifact numbers verified against actual `-v` runs: 15 planted defects
(15 subtests), 14 Git sibling retargets (14 subtests), "11 of 11 foreign
clauses" (test log), 15 declaring identifiers + 2 exemptions (test log).

## The seven corrected Git rows, verified by hand

Read `internal/specdoc/SPEC.md:4785-4793` directly and checked the corrections:

- `GitIndex.format` → L4790 `format:git_index` (was GitObjectPack's `git_pack_v2`) ✓
- `GitIndexEntry.mode` → L4791 `mode:uint32` (was GitHead's `branch|detached|unborn`) ✓
- `GitIndexEntry.oid` → L4791 `oid:git-oid` (was GitHead's nullable `oid`) ✓

The row from the Bug description — `EnvironmentTuple.store_schema_fingerprint`,
which used to claim "digest" — now quotes real text at L3630 and states
presence-only with the digest inference explicitly refused.

## Validation, run by the reviewer on the candidate tree

| Command | Result |
| --- | --- |
| `gofmt -l .` | clean |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `go test ./... -count=1` | all packages ok |
| `go test ./... -cover` | `canonicaljson` 97.2%, `specdoc` 100.0% |
| `tracecheck` | `contracts=60 normative_sections=36 acceptance_cases=43 fixtures=30 compatibility_contracts=55` exit 0 |
| `go generate ./internal/catalog` + status | no diff |
| `golangci-lint run` (no repo config) | only pre-existing `errcheck`/`ST1005`/`unused` in untouched files; the one hit in `constraint_excerpt_test.go` is `defer file.Close()`, the identical pre-existing pattern at `constraint_inventory_test.go:414` |

## Acceptance criteria

- Every `specExcerpt` compared verbatim at the declared line, absent quote reddens — **met**, proven on the real artifact.
- Comparison uses the `internal/specpin` digest so a swapped spec cannot pass — **met**, proven by M2 and by the untouched pin.
- Planting an invented quote reddens, proven by doing it — **met**, by the producer on copies and by the reviewer on the shipped file.
- Deliberate paraphrases marked and held to naming their line — **met**; the form is supported (`TestMarkedParaphraseRowIsAdmittedWhenItNamesItsLine`), refused when the line does not name the member, and refused when the line is outside the document. Zero shipped rows use it, which is honest rather than a gap.

## Non-blocking observations

1. **Fenced-example citations are admitted.** I confirmed by probe that a row can
   satisfy every rule by citing a line inside a clause's non-normative
   `~~~json` example instead of its declaration — e.g. `Lease Record.reason`
   citing `L1929 "reason": "graceful_takeover",` returns zero failures. The
   line is verbatim, correctly located, in clause 5.3, names the member, and is
   not a table body row, so the declaring-row anchor does not reach it. **Zero
   shipped rows do this**, and the class is already disclosed in `README.md` and
   the artifact as "a citation that lands outside every table row" being the
   residual gap. Recorded so the next tightening has a named target, not as a
   defect in this delta.
2. LOGBOOK 0135 says removing one declaring pin "reddens 8 shipped rows"; the
   `GitIndex` pin I removed reddens 6. Different pin, different count — the
   claim's substance (fails closed, no free pass) is confirmed.

## Handoff

Accepted via `accept_cr(BUG-260902-3dn8jd, revision=3, …)`, which parks the
element at `to-review`. This is a reviewer-archetype run and supplies no
`commit_ack`; the orchestrator checkpoints or integrates the accepted revision
and makes the `done` transition with `commit_ack=scope_committed`.
