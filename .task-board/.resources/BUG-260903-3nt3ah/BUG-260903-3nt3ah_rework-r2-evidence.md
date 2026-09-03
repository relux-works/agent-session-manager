# BUG-260903-3nt3ah rev2 — rework evidence

Scope: rework of the single BLOCKING finding B1 plus secondaries S1/S2/S3.
The (a)/(b) decision was accepted in rev1 review and is not reopened.

## B1 — the false SPEC attribution

`canonical.go` attributed `"before identity calculation"` to normative fixture
`NUM-UNSAFE-ROUND`. Measured against the digest-verified `internal/specdoc`:

| Fragment | Real SPEC.md line | Row that declares it |
| --- | ---: | --- |
| `A decoder MUST reject a numeric literal at or beyond <code>2^53</code> even if its host language can represent it` | 292 | prose, Section 1.6 |
| `Implementations MUST NOT round a value and continue.` | 295 | prose, Section 1.6 |
| `Reject before identity calculation with <code>incompatible_schema</code>` | 301 | `NUM-UNSAFE-NUMBER` |
| `Reject from the JSON number token before conversion to a host double; an implementation that first rounds it to 9007199254740992 is nonconforming` | 302 | `NUM-UNSAFE-ROUND` |

Fixed by citing **both** fixture rows, each from its own line, in a parseable
form. `NUM-UNSAFE-ROUND`'s real text is the stronger citation for this
implementation: `validateAXNumbers` refuses on the `json.Number` token before
any conversion to a host double. `LOGBOOK.md` and `README.md` carry the same
correction.

## The bound that let it through

`canonical_number_contract_test.go` declared the specification quotations
"which no test can verify against their sources from inside this package".
False: `constraint_excerpt_test.go` in the same package already imports
`internal/specdoc` and does exactly that. That half is now a gate. The RFC 8785
half is genuinely unverifiable in-repo and is restated more precisely (see
RESIDUAL).

## The gate

`TestCanonicalizeSpecificationCitationsQuoteThePinnedSpecification` parses
`SPEC.md:<line> [FIXTURE] "text"` rows out of the `Canonicalize` doc comment
with `go/parser` — the comment is the test's input, not a restatement of it —
and requires, per citation:

1. the quote occurs in the digest-pinned `SPEC.md`;
2. **exactly once** — this is what stops a citation being weakened instead of
   corrected;
3. beginning at the declared line;
4. when a fixture is named, `TableRowAt(line).Identifier` is that fixture; and
   when no fixture is named, the line is not a table body row at all.

Plus `requiredSpecCitationLines` / `requiredSpecCitationFixtures`, so a citation
must be corrected rather than deleted, and set equality between the SPEC clauses
the comment names and the clauses its citations land in.

## Mutants — 9, each confirmed PRESENT in the source before measuring, all exit 1

| ID | Mutant | Gate that reddened |
| --- | --- | --- |
| N1 | `NUM-UNSAFE-ROUND` attribution re-pointed to the neighbouring row | row identity + required-fixture set |
| N2 | declared line re-pointed 302 -> 301, fixture unchanged | quote-begins-at-declared-line |
| N3 | quote weakened to `"Reject"` | uniqueness (matches 10 lines) |
| N4 | fixture citation deleted rather than corrected | required-citation set |
| N5 | fixture row cited without naming the fixture | unattributed-table-row |
| N6 | documented container limit 256 -> 255 | container-limit pin |
| N7 | Appendix B pin renamed in prose | test-name pin |
| N8 | false clause claim **added beside** the true one | clause set equality |
| N9 | clause claim substituted 1.6 -> 1.5 | clause set equality |

N8 is the one that matters methodologically: the first draft of the clause gate
used `strings.Contains` and N8 **survived green**. A presence check confirms the
truth is stated; it cannot see a lie added next to it. Rewritten as set
equality. Representative assertion text is in `mutation-detail.txt`.

Verbatim gate output for N8 after the rewrite:

    doc comment names SPEC.md clauses [1.5 1.6], but every citation lands in [1.6]

## Secondaries

- **S1** `LOGBOOK.md` cited `validateAXNumbers` at `canonical.go:199`; the call
  site had already moved to 256 and 199 was inside the new doc comment. Now
  names `prepareObjectIdentity` instead of a line number, so it cannot rot again.
- **S2** the comment names `TestCanonicalizeMatchesEveryFiniteRFC8785AppendixBNumberSample`.
  `TestCanonicalizeDocCommentNamesOnlyTestsThatExist` requires every `Test*`
  identifier in the comment to be a declared test in the package, reusing
  `packageTestFunctionNames` from `declared_bounds_proofs_test.go` rather than
  walking the sources a second time. Non-vacuous: at least one name required.
- **S3** the comment said `maxNestingDepth`, unexported and unresolvable from
  the exported docs. Now `256`, tied to the constant and driven through
  `Canonicalize` at accept-256 / refuse-257.

## Validation — real exit codes

| Command | Exit | Note |
| --- | ---: | --- |
| `go build ./...` | 0 | |
| `go vet ./...` | 0 | |
| `gofmt -l .` | 0 | empty output |
| `go test ./... -cover -count=1` | 0 | 11 ok, canonicaljson 97.2% (unchanged) |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | |
| `go generate ./internal/catalog` | 0 | no generated file in `git status`; only the 4 edited files are modified |
| 4x `go test -fuzz -fuzztime=15s` (canonicaljson) | 0 | all four targets |
| `curator status --check` | **1** | `worktree: go-testing-tools not-installed` — pre-existing Story-worktree provisioning state, unchanged from rev1; `.claude/skills/` does not exist in this worktree and the diff touches no Curator-managed path. Reported failing, not explained away. |

`internal/canonicaljson/canonical_test.go` byte-for-byte unchanged
(`git diff --numstat` returns no row for it).

Changed files: `internal/canonicaljson/canonical.go` (comments only, no logic),
`internal/canonicaljson/canonical_number_contract_test.go`, `README.md`,
`LOGBOOK.md`.

## Residual — named, not closed

- **Claims about RFC 8785 itself remain unmeasured.** RFC 8785 is not vendored,
  so no test here compares a claim about Section 3.2.2.3 or Appendix B against
  its source. What is measured is one step downstream:
  `TestCanonicalizeMatchesEveryFiniteRFC8785AppendixBNumberSample` catches drift
  between this implementation and a *vendored transcription* of the Appendix B
  samples; drift between that transcription and the RFC is not detectable here.
  Stated in the file as a bound.
- **Prose outside the enumerated rows is still unchecked.** The pin covers the
  `literal -> canonical` rows, the entry-point names, the `SPEC.md` citations,
  the clause names, the container limit and the test names. Free prose between
  them is not derived from anything. An indented line that is none of the three
  known forms is a hard failure, so documented-but-unchecked *structured* text
  cannot accumulate, but a sentence can.
- **The call-graph residual is inherited** from `productionCallGraph`:
  reflection, a function value handed to another package and invoked there, and
  a func-typed struct field are not modelled.
- **A citation could quote a true line that does not support the surrounding
  argument.** The gate proves a quotation is real, at the right line, and from
  the right fixture row; it does not prove the inference drawn from it.
