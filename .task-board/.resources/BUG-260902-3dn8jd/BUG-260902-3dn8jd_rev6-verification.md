# BUG-260902-3dn8jd — revision 6 republication run: independent re-verification

Run: `RUN-260902-0d963e`. Worktree `.temp/STORY-260902-2xwq38/worktree`,
branch `task-board/story/STORY-260902-2xwq38`.

Nothing in the repository was changed by this run. The change was ACCEPTED at
review round 3, rebased onto trunk `facbd9a8` in revision 4, and re-verified in
revision 5. This run re-verifies it again from scratch on the same tree, adds a
measurement the previous run only reasoned about, and attempts publication.

## Leaf shape

```
HEAD        6466943e53de001817901079cf9cd15872afd716
parent      facbd9a8ca3bcf385a0b5f9c646c81282a495758  (== origin/main)
ahead 1, behind 0, single parent, git status --short empty
```

## Validation — all 18 configured gates, real exit codes, this tree

Every command from `task-board.config.json -> spawn.worktree_isolation.validation.commands`,
each run as a standalone process, exit code as returned. Log:
`BUG-260902-3dn8jd_rev6-validation.log`.

| # | Command | Exit |
| --- | --- | ---: |
| 1 | `test -z "$(gofmt -l $(git ls-files ... '*.go'))"` | 0 |
| 2 | `go build ./...` | 0 |
| 3 | `go vet ./...` | 0 |
| 4 | `go test ./... -count=1` | 0 |
| 5 | `go test ./... -race -count=1` | 0 |
| 6 | `go test ./... -cover -count=1` | 0 |
| 7 | `go test ./internal/scalar -fuzz=^FuzzScalarProductionEntries$ -fuzztime=100x` | 0 |
| 8 | `go test ./internal/canonicaljson -fuzz=^FuzzCanonicalizeRoundTrip$ -fuzztime=100x` | 0 |
| 9 | `go test ./internal/canonicaljson -fuzz=^FuzzObjectIdentityRepresentationInvariant$ -fuzztime=100x` | 0 |
| 10 | `go test ./internal/canonicaljson -fuzz=^FuzzClosedIdentityShapeRefusal$ -fuzztime=100x` | 0 |
| 11 | `go test ./internal/canonicaljson -fuzz=^FuzzObservationEventRefusal$ -fuzztime=100x` | 0 |
| 12 | `go run ./internal/traceability/cmd/tracecheck` | 0 |
| 13 | `cataloggen ... -check` | 0 |
| 14 | `GOOS=linux GOARCH=amd64 go build ./...` | 0 |
| 15 | `GOOS=windows GOARCH=amd64 go build ./...` | 0 |
| 16 | `git ls-files -z '*.json' \| xargs -0 -n1 python3 -c 'json.load(...)'` | 0 |
| 17 | `task-board validate` | 0 (prints 250 pre-existing board-hygiene advisories on unrelated elements; the gate's own status is 0) |
| 18 | `git diff --check` | 0 |

## Anti-vacuity — re-proved on this tree, three mutants

Each mutation's presence was confirmed in the file before its result was
believed. Each was reverted from a byte backup under
`.temp/BUG-260902-3dn8jd/backup/`, never with `git checkout`, and the restored
SHA-256 was compared against the pre-mutation SHA-256 every time.
`git status --short` was empty after each. Log: `BUG-260902-3dn8jd_rev6-mutants.log`.

**Mutant A — the original incident defect replayed.** Row 375
`EnvironmentTuple.store_schema_fingerprint`, whose fabricated `digest` claim is
why this Bug exists, re-planted as
`L3630 "<code>store_schema_fingerprint</code> is a lowercase hex digest"`.
Presence confirmed (`grep -c` = 1). **Real exit 1**, three tests red, naming the
row, the absent text, and the lost member anchor:

```
TestConstraintEnumerationSpecExcerptsQuoteThePinnedSpecification
  EnvironmentTuple.store_schema_fingerprint: entry 1 (L3630) quotes text absent
  from the pinned SPEC.md: "<code>store_schema_fingerprint</code> is a lowercase hex digest"
TestArtifactQuotesAreVerbatimPinnedSpecificationText
  line 375 quotes text absent from the pinned SPEC.md: ...
TestUnmodifiedConstraintEnumerationIsAdmitted
  shipped row ... refused: [... no entry names the member; the row could quote
  any line of the specification]
```

**Mutant B — a NARROWING of the document, not a deletion.** One trailing space
appended to `SPEC.md` line 3630. That perturbation is benign under the package's
own whitespace-collapse rule, so the text comparison alone would still admit the
row; only the digest gate can catch it. **Real exit 1**, refused by both digests:

```
Load() error = pinned specification document mismatch:
SHA-256 is b7ffc9ea04b0b38e38a7092bf052ad2ece47f4f7f3f8c1113f34694d0b362c2a,
want    562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a
```

Every excerpt test failed CLOSED on the load error instead of comparing against
the substituted document. This separates the two axes: a document that would
still satisfy the text comparison is refused anyway, so the gate is anchored on
the pin, not on the text.

**Mutant C — a NARROWING of a citation, not a deletion.** `GitIndexEntry.mode`
retargeted from its own declaring line to sibling `GitHead`'s verbatim enum at
`L4788 "<code>mode:branch&#124;detached&#124;unborn</code>"` — a true, verbatim
quote, in the same clause 10.4, of a different type's declaration. This is the F4
class the round-2 review found seven live instances of. **Real exit 1**, refused
by the sibling's own name:

```
GitIndexEntry.mode: entry 1 (L4788) cites the "Type" table row that declares
"GitHead", but "GitIndexEntry" is declared as "GitIndexEntry"
```

**Bound, not a parser that reddens on everything.** With the tree restored,
`go test ./internal/canonicaljson/ ./internal/specdoc/ -count=1` is **exit 0**
and `TestUnmodifiedConstraintEnumerationIsAdmitted` is green under none of the
three mutants and green here.

## Pin and embed isolation — verified, not asserted

```
sha256(internal/specdoc/SPEC.md) = 562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a
internal/specpin/pin.go:30 DocumentSHA256 = 562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a
```

`internal/specpin` is absent from this branch's diff against trunk, so the digest
the fidelity gate anchors on is the pre-existing pin and could not have been
minted to fit.

`go list -f '{{.Imports}} / {{.TestImports}}{{.XTestImports}}'` over every
package: `internal/specdoc` appears in `canonicaljson`'s **TestImports** and in
its own test imports, and in **no** package's non-test `Imports`. The embedded
12,665-line document never reaches a shipped command.

## Carried forward

Unchanged from revision 5, none regressed, each still needing its own board item:
`environment_id` cross-schema inference (both lines inside clause 7.8, so the
clause anchor genuinely cannot catch it — the stated residual limit is honest);
`closed-vocabularies.md` has the same unchecked-column class; the residual
clause-not-shape gap and the fenced-JSON-example landing, both stated in README
and in the artifact, with zero shipped rows using either; the pre-existing
LOGBOOK 1810/2240 ordering inversion owned by trunk.

No production validator was changed at any revision of this Bug.
