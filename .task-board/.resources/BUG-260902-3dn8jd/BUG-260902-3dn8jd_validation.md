# BUG-260902-3dn8jd — validation evidence

Worktree: `.temp/STORY-260902-2xwq38/worktree`, branch
`task-board/story/STORY-260902-2xwq38`. Every command below was run directly as
a standalone process from the worktree root; the exit code shown is the real
status of that command.

| Command | Exit | Log |
| --- | ---: | --- |
| `go build ./...` | 0 | `.temp/BUG-260902-3dn8jd/go-build-01.log` |
| `go vet ./...` | 0 | `.temp/BUG-260902-3dn8jd/go-vet-01.log` |
| `go test ./... -count=1` | 0 | `.temp/BUG-260902-3dn8jd/go-test-01.log` |
| `go test ./... -cover -count=1` | 0 | `.temp/BUG-260902-3dn8jd/go-test-cover-01.log` |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | `.temp/BUG-260902-3dn8jd/tracecheck-01.log` |
| `go generate ./internal/catalog` | 0 | `.temp/BUG-260902-3dn8jd/go-generate-01.log` |
| `git diff --exit-code -- internal/catalog/catalog_gen.go` | 0 | generated catalog unchanged |
| `gofmt -l internal/` | 0 | no output; no file needs formatting |

`tracecheck` reported
`traceability ok: contracts=60 normative_sections=36 acceptance_cases=43 fixtures=30 compatibility_contracts=55 assigned_scopes=0`.

## Coverage

| Package | Coverage |
| --- | ---: |
| `internal/canonicaljson` | 97.2% |
| `internal/specdoc` (new) | 100.0% |
| `internal/specpin` | 85.1% |
| `internal/traceability` | 85.0% |

`internal/canonicaljson` coverage is unchanged from before this work; the
package's own `TestEveryProductionRefusalGuardIsExecuted` still passes, which
runs the shipped suite under coverage before measuring any refusal guard.

## Negative cases that ran

`go test ./internal/canonicaljson -run 'Planted|Paraphrase|Unmodified|SwappedSpecification' -v`
executed and passed:

- `TestPlantedConstraintEnumerationRowsRedden/invented_quote`
- `TestPlantedConstraintEnumerationRowsRedden/invented_quote_among_true_ones`
- `TestPlantedConstraintEnumerationRowsRedden/true_quote_at_the_wrong_line`
- `TestPlantedConstraintEnumerationRowsRedden/true_quote_that_never_names_the_member`
- `TestPlantedConstraintEnumerationRowsRedden/paraphrase_of_a_line_that_never_names_the_member`
- `TestPlantedConstraintEnumerationRowsRedden/paraphrase_outside_the_document`
- `TestPlantedConstraintEnumerationRowsRedden/empty_quote`
- `TestPlantedConstraintEnumerationRowsRedden/unanchored_prose_with_no_line_reference`
- `TestPlantedConstraintEnumerationRowsRedden/bare_prose,_the_pre-fix_contract`
- `TestPlantedConstraintEnumerationRowsRedden/empty_cell`
- `TestPlantedConstraintEnumerationRowsRedden/trailing_text_after_a_valid_entry`
- `TestMarkedParaphraseRowIsAdmittedWhenItNamesItsLine`
- `TestUnmodifiedConstraintEnumerationIsAdmitted`
- `TestSpecExcerptComparisonRefusesASwappedSpecification`
- `TestParseRefusesEveryNonPinnedDocument` (7 sub-cases: absent, empty,
  truncated read, appended byte, single-character substitution,
  whitespace-only substitution, unrelated document)

Each planted case mutates a copy of the shipped artifact in `t.TempDir()`, feeds
it through the same `readConstraintRows` +
`verifyRowAgainstPinnedSpecification` path the real gate uses, and fails if the
defect is admitted. `TestUnmodifiedConstraintEnumerationIsAdmitted` is the other
half of the pair: the gate that reddens all eleven plants still accepts the
shipped artifact, so the plants prove a bound rather than a broken parser.

The bound is proven by narrowing, not only by deletion:
`true_quote_at_the_wrong_line` keeps real specification text and moves only the
declared line; `true_quote_that_never_names_the_member` keeps a real line and a
real quote and removes only the member anchor.
`TestNormalizeForgivesOnlyWhitespace` pins the normalization rule from the other
side, requiring case-folded, backtick-for-`<code>`, and markup-stripped variants
of real specification text to stay unmatched.

## What was not run

Nothing in the repository validation set was skipped. The seeded fuzz targets
(`FuzzCanonicalizeRoundTrip`, `FuzzObjectIdentityRepresentationInvariant`,
`FuzzClosedIdentityShapeRefusal`, `FuzzObservationEventRefusal`) run as ordinary
seed-corpus tests inside `go test ./...` above; no extended `-fuzz` campaign was
run, because this change touches no decoder, canonicalizer, or validator
production code.
