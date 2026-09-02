# BUG-260902-3dn8jd — Revision 3: response to the round-2 review (CR rev 2)

Round-2 verdict: CHANGES REQUESTED, F4 blocking, F5 recorded. Both are addressed.
Nothing in the `specdoc` design, the clause anchor, the F1 re-keying, or the
existing anti-vacuity suite was changed except to extend it.

Base for this revision: `c3cd51c` (the reviewed head). No production validator
changed. Changed paths: `internal/specdoc/specdoc.go`,
`internal/specdoc/specdoc_test.go`,
`internal/canonicaljson/constraint_excerpt_test.go`,
`internal/canonicaljson/testdata/constraint-enumeration.md`, `README.md`,
`LOGBOOK.md`.

---

## F4 — fixed, and the class is now gated rather than the seven instances

### 1. The seven cells are corrected

| Row | Was | Now | Declared by |
| --- | --- | --- | --- |
| `GitIndex.format` | L4789 `format:git_pack_v2` | L4790 `format:git_index` | GitIndex |
| `GitIndex.blob_id` | L4789 | L4790 `blob_id:digest` | GitIndex |
| `GitIndex.blob_descriptor_id` | L4789 | L4790 `blob_descriptor_id:digest` | GitIndex |
| `GitIndexEntry.mode` | L4788 `mode:branch\|detached\|unborn` | L4791 `mode:uint32` | GitIndexEntry |
| `GitIndexEntry.oid` | L4788 `oid:git-oid\|null` | L4791 `oid:git-oid` | GitIndexEntry |
| `GitSubmodule.path` | L4791 | L4792 `path:path` | GitSubmodule |
| `GitFeatures.object_format` | L4789 | L4793 `object_format:sha1\|sha256` | GitFeatures |

The three material ones now agree with production: `validateGitIndex` enforces
`git_index`, `validateGitIndexEntry` enforces `requireUint(object, "mode",
1<<32-1)` and `requireGitOID` (no null).

### 2. Class re-scan over all 347 rows — ratios, not prose

Independent scan (parses the artifact and the pinned document, classifies every
citation by whether it lands on a Markdown table body row, and compares the
row's first cell to the artifact row's shape and member). Run before and after
the fix:

| Measure | Before | After |
| ---: | ---: | ---: |
| Artifact rows | 347 | 347 |
| Citations | 357 | 357 |
| Citations landing on a table body row | 235 (65.8%) | 235 (65.8%) |
| — first-column header `Field` / `Type` / `Tag` / `Term` / `Member` | 126 / 51 / 55 / 2 / 1 | 126 / 51 / 55 / 2 / 1 |
| Citations landing outside any table row (prose, headings) | 122 | 122 |
| **Type/Tag citations naming another type** | **7** | **0** |
| Member-table citations naming another member | 2 | 2 (both exempted by name) |

My scan reproduced exactly the reviewer's seven and found no eighth. The two
member-table exceptions are the Section 2.1 Terms-table `Session name`
indirection already documented in round 1; they are now exempted by name, line,
and written reason rather than by silence.

### 3. The clause anchor is tightened — a declaring-row anchor

Judgement: not disproportionate. Implemented rather than argued away.

`specdoc.TableRowAt(line)` reports what a Markdown table body row declares: the
table's first-column header, the row's own first cell, and that cell with its
enclosing `<code>` element removed. Header and delimiter lines report nothing.

The gate: **a citation that lands on a table body row must land on the row that
declares what it cites.** Either the first cell names the member — the
per-member `Field` tables anchor themselves — or it names the identifier under
which the pinned document declares that shape, pinned per shape in
`constraintRowDeclaringIdentifiers` (15 shapes: the seven Git types, the four
`ManifestEntry` tags, the two `WorkspaceMember` tags, the two
`WorkspaceSnapshotMember` tags).

The rule is deliberately not restricted to the Section 10.4 multi-type table:
the same defect exists in per-member tables, and a mutant proves it (below).
Citations that land on prose or headings are unconstrained by this rule; that
residual is stated in `README.md` and in the artifact, and the `environment_id`
finding still sits inside it.

Exactness, both directions:
`TestEveryConstraintEnumerationDeclaringIdentifierIsExercised` fails on a pin no
citation lands on and on an exemption nothing consumes. A shape that needs a pin
and lacks one is failed by the gate itself, not skipped.

---

## F5 — fixed in code and in both documents

`specdoc` now has two hard boundaries, not one. The newline between two adjacent
Markdown table rows collapses to the unmatchable `BlockSeparator`, like a blank
line. A table row is a complete line by construction, so no honest excerpt spans
two of them; the whole shipped artifact stayed green under the change, which is
the evidence that none needed to.

The reviewer's exact planted stitch is now a permanent test case:
`TestPlantedConstraintEnumerationRowsRedden/quote_stitched_across_a_table_row_boundary`.

The wording is corrected in both places. `README.md` and the artifact no longer
claim whitespace collapsing forgives hard wrapping and table indentation "and
nothing else". They now name the two hard boundaries and state what remains
forgiven: the newline inside one block — hard line wrapping, table indentation,
and the newline between two adjacent list items or two adjacent lines of one
paragraph. `TestQuoteLinesStillForgivesHardWrappingInsideOneParagraph` pins the
other half by requiring 200 wrapped prose boundaries to stay quotable, so a
boundary rule that refused every newline would not pass.

---

## Evidence — mutants, all confirmed present before measurement

Run in `/tmp/axrev3`, a `git archive HEAD` copy with the revision's changed files
copied in. The worktree was never mutated; `git status --short` lists only the
six intended files.

| Mutant | Real exit | What reddened |
| --- | ---: | --- |
| M1 delete: `declaringRowFailure` result discarded at the call site | 1 | all 14 Git sibling subtests + the field-table plant |
| M2 narrow: anchor applied to `Type` tables only | 1 | `PlantedConstraintEnumerationRowsRedden/true_quote_from_a_sibling_member's_table_row` |
| M3 revert: `GitIndexEntry.mode` restored to its pre-fix L4788 citation | 1 | `SpecExcerptsQuote…` and `UnmodifiedConstraintEnumerationIsAdmitted`, naming the row and `GitHead` |
| M4 delete: table-row hard boundary removed from `specdoc` | 1 | `TestQuoteLinesRefusesTextThatSpansTwoTableRows` and the artifact stitch plant |
| M5 delete: `"GitIndexEntry"` pin removed | 1 | 8 shipped `GitIndexEntry` rows refused — fails closed, no free pass |
| M6 delete: the `Session Record 2.0.0 and 3.0.0` exemption removed | 1 | that shipped row refused — the exemption is load-bearing |
| M7 add: a phantom pin and a stale exemption | 1 | `EveryConstraintEnumerationDeclaringIdentifierIsExercised`, naming both |

New positive-bound tests, so the plants prove a bound rather than a broken
parser: `TestDeclaringRowAnchorRefusesEverySiblingRowOfTheGitTable` derives the
sibling pairs from the pinned document itself (14 retargets, both directions,
each refused by the sibling's name) while
`TestUnmodifiedConstraintEnumerationIsAdmitted` requires every shipped row to
pass; `TestTableRowAtNamesWhatEachRowDeclares` pins the index including the
header and delimiter lines that must declare nothing.

---

## Validation — real exit codes, run in this worktree

| Command | Exit |
| --- | ---: |
| `gofmt -l .` (empty output) | 0 |
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `go test ./... -count=1` (11 packages) | 0 |
| `go test ./... -cover` | 0 |
| `go run ./internal/traceability/cmd/tracecheck` | 0 |
| `go generate ./internal/catalog` + clean `git status` | 0 |

Coverage: `internal/canonicaljson` 97.2%, `internal/specdoc` 100.0%.
`go list -deps` over every package except `internal/specdoc` itself: no
non-test package reaches it, so the embedded document still never ships.

Planted-defect count in `TestPlantedConstraintEnumerationRowsRedden` is now 15
(was 13); `README.md` states the new count and the two new cases.
