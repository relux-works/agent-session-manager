# BUG-260902-3dn8jd — Base refresh onto trunk facbd9a (revision 4)

Accepted work was NOT redesigned. The declaring-row anchor, table-row boundary,
seven corrected Git citations, two named exemptions, per-family counts, the
specdoc digest gate and the anti-vacuity suite are unchanged. This run rebased
the leaf, composed the two overlapping documents, and revalidated.

## Leaf shape

| Property | Value |
| --- | --- |
| Head | `6466943e53de001817901079cf9cd15872afd716` |
| Parent | `facbd9a` (single parent, == `origin/main`) |
| Commits past checkpoint | 1 |
| Behind trunk | 0 |
| Signature | Good, ECDSA `SHA256:V6JiKG7J29mjsvikcLoSVp0bLa77VTsFy12gnLO81cM` |
| Author | Ivan Oparin <oparin@me.com> |
| Files vs trunk | 10 — exactly the CR paths |

`internal/specpin/` is untouched (`scope_test.go` diff = 0 lines), as instructed.

## LOGBOOK.md composition — mechanical union, one defect found and fixed

Entry identity = (date, timestamp). Verified as sets, then body-by-body.

| Side | Entries |
| ---: | --- |
| Base `422786c` | 25 |
| Trunk added | 2 — `2026-09-03 0210`, `0035` |
| This Story added | 3 — `2026-09-03 0135`, `0050`, `0022` |
| Merged | 30 |

`merged == trunk ∪ story` exactly: zero entries in merged absent from the union,
zero union entries absent from merged, zero duplicates. All 30 entry bodies are
**byte-identical** to their originating side.

**DEFECT FOUND AND FIXED.** The rebase conflict resolution dropped the blank-line
separator before 4 of the 5 interleaved `2026-09-03` entries (`0135`, `0050`,
`0035`, `0022`), leaving a `- bullet` line immediately followed by a `### ` heading.
Base, trunk and the pre-rebase story file each have **0** such cases; the merged
file had exactly 4, all inside the interleaved region — so this was merge damage,
not inherited. Restored; the file now has 0 headings without a preceding blank,
and all 30 bodies are byte-exact against their sources *including* blank lines.
This is the class the refresh brief predicted: a merge that resolves cleanly and
is still wrong.

**Pre-existing, NOT introduced, NOT fixed (out of scope):** LOGBOOK has one
ordering inversion — `2026-09-02 1810` sits before `2026-09-02 2240`. It is present
identically in base, trunk, the pre-rebase story file and the merged file. The
newly interleaved `2026-09-03` region is strictly descending (0210, 0135, 0050,
0035, 0022). Reported rather than silently repaired, since trunk owns it.

## README.md composition — mechanical, provably lossless

Line counts are exactly additive: 1172 base + 26 trunk + 115 story = 1313 merged.
Headings: merged set == union of both sides, no extra, no missing.

Three-way verification:

- added lines in `diff(base → trunk)` == added lines in `diff(story → merged)` — **identical sets**, so trunk's contribution survived whole.
- added lines in `diff(base → story)` == added lines in `diff(trunk → merged)` — **identical sets**, so this Story's contribution survived whole.
- removals in `diff(trunk → merged)` == removals in `diff(base → story)` — **identical**: the merge deletes exactly the 4 lines this Story deleted and nothing of trunk's.
- removals in `diff(story → merged)` = **0**.

Contradiction sweep (the brief's explicit concern): the stale claims
`does not vendor` / `cannot verify itself against SPEC.md` are absent from
`README.md`, `.spec/README.md` and the enumeration artifact. The false F5 claim
`and nothing else` survives only inside LOGBOOK historical entries — once as the
title of entry 0022, once inside 0135 where it is quoted precisely to record that
it *was* false, and once in an unrelated 2026-09-02 entry. Not a contradiction.
The two sides' additions land far apart (story section at L91, trunk content at
L371-395) and describe different mechanisms.

Judgement required: none for README (purely additive on both sides). The only
judgement call was the LOGBOOK separator restoration described above.

## Validation — all 18 configured gates, real exit codes

Every gate below ran against the final merged tree WITH the LOGBOOK fix applied.

| # | Gate | Exit |
| ---: | --- | ---: |
| 1 | `gofmt -l` empty | 0 |
| 2 | `go build ./...` | 0 |
| 3 | `go vet ./...` | 0 |
| 4 | `go test ./... -count=1 -v` | 0 |
| 5 | `go test ./... -race -count=1` | 0 |
| 6 | `go test ./... -cover -count=1` | 0 |
| 7 | fuzz `FuzzScalarProductionEntries` 100x | 0 |
| 8 | fuzz `FuzzCanonicalizeRoundTrip` 100x | 0 |
| 9 | fuzz `FuzzObjectIdentityRepresentationInvariant` 100x | 0 |
| 10 | fuzz `FuzzClosedIdentityShapeRefusal` 100x | 0 |
| 11 | fuzz `FuzzObservationEventRefusal` 100x | 0 |
| 12 | `tracecheck` | 0 |
| 13 | `cataloggen -check` (no drift) | 0 |
| 14 | `GOOS=linux GOARCH=amd64 go build ./...` | 0 |
| 15 | `GOOS=windows GOARCH=amd64 go build ./...` | 0 |
| 16 | every tracked `*.json` parses | 0 |
| 17 | `task-board validate` | 0 |
| 18 | `git diff --check` | 0 |

Coverage: `internal/canonicaljson` 97.2%, `internal/specdoc` 100.0%.

## Anti-vacuity re-proved ON THE MERGED TREE

A green suite on a composition proves nothing unless something in it still fails.
Both mutants were confirmed PRESENT before the gate was believed, and both were
reverted from a file backup rather than `git checkout`.

**Mutant A — invented quote (the original incident defect).** Replaced the
`EnvironmentTuple.store_schema_fingerprint` excerpt with
`"<code>store_schema_fingerprint</code> is a digest of the store schema"`, text
absent from the pinned document. Presence confirmed by grep. **Real exit 1.** The
gate named both the row and the absent text, and separately flagged that no entry
names the member:

```
EnvironmentTuple.store_schema_fingerprint: entry 1 (L3630) quotes text absent
from the pinned SPEC.md: "<code>store_schema_fingerprint</code> is a digest of
the store schema"
EnvironmentTuple.store_schema_fingerprint: no entry names the member; the row
could quote any line of the specification
```

**Mutant B — one-byte SPEC perturbation, a NARROWING not a deletion.** Appended a
single trailing space to `SPEC.md` line 3630. That edit is *benign under the
package's own normalization rule* (whitespace runs collapse), so the excerpt
comparison alone would still have passed it — only the digest gate can catch it.
Presence confirmed by recomputed SHA-256. **Real exit 1**, refused by
`TestLoadAcceptsOnlyThePinnedDocumentDigest` and by every excerpt test, which
fail closed on the load error rather than comparing against the swapped document.

Unmodified tree passes (gates above). Tree restored to a clean digest match.

## Pin and isolation re-verified after the rebase

- `internal/specdoc/SPEC.md` SHA-256 == `pin.go` `DocumentSHA256` == lockfile
  `source.document.sha256` == `562546d2…484a`. The pin is the pre-existing one;
  `internal/specpin/` was not touched by this CR.
- `internal/specdoc` has **no non-test importer**. It appears only in
  `canonicaljson`'s `TestImports` and its own external test, so the embedded
  12,665-line document never reaches a shipped command. (Verified with
  `go list -f {{.Imports}}` vs `{{.TestImports}}`; a bare `go list -deps ./...`
  grep returns 1 only because `./...` enumerates the package itself.)

## Findings the combination surfaced that neither side would have caught alone

1. The dropped LOGBOOK blank-line separators — only visible once both sides'
   entries were interleaved into the same day block. Fixed.
2. The pre-existing `1810`/`2240` ordering inversion, confirmed as trunk's and
   left alone. Reported, not repaired.

Nothing was weakened or deleted to make the merged tree pass.
