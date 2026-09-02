# BUG-260902-3dn8jd revision 2 — response to review `CR-BUG-260902-3dn8jd-1` rev 1

Reviewer verdict was `changes_requested` for **F1 only**; **F2** and **F3** were
recorded for the producer's judgement. All three are addressed. No production
validator changed in this revision either.

| Finding | Severity | Disposition |
| --- | --- | --- |
| F1 — artifact rewrite silently narrowed a reachability gate | blocking | Fixed, and the loss turned out to be larger than the review measured: three of the eight dropped rows were the **only** thing covering their production path. |
| F2 — the excerpt gate is shape-blind | record | Reproduced, then **closed by a clause anchor**; the residual limit is documented. |
| F3 — normalization crosses block boundaries | record | Fixed. A whitespace run containing a blank line now collapses to an unmatchable separator. Zero shipped rows relied on the old behaviour. |

---

## F1 — fixed, and three of the dropped rows were load-bearing

### What was wrong

`sessionRecordGrammarFamily` classified a constraint-enumeration row into an
executable grammar family by substring-matching the row's `Pinned SPEC
declaration` prose. That column is the fidelity artifact revision 1 rewrote, so
the rewrite removed the words the classifier greps for and eight rows fell out of
`TestSessionRecordDeclaredGrammarRowsReachIdentityProductionEntries`. The only
completeness assertion was `familyCounts[family] == 0`; four is not zero, so a
64% narrowing of the `reverse-dns` family reported nothing.

### The fix

1. `sessionRecordGrammarRows` pins the row set by **shape and member** — the two
   fields a fidelity correction cannot rewrite. 19 rows. An absent pinned row is
   reported by name, never skipped.
2. `sessionRecordGrammarFamilyCounts` pins a **per-family count**, replacing the
   non-empty guard. A narrowing now reddens as loudly as an emptied family.
3. The prose classifier is deleted outright, not merely supplemented.
4. `Session Record Launch Plan.env_names` is back in `environment-name`.
5. All seven `extensions` rows are back in `reverse-dns` — see the decision below.
6. Incidental: `provider-id` is now shape-aware. The `Session Record 2.0.0 and
   3.0.0.provider_id` row that revision 1 *added* to the gate was driving a
   **v1** record, because the `provider-id` branch always built
   `validSessionRecordV1Object()`.

Row count 11 -> 19; `=== RUN` lines for this test 66 -> 113 (the review measured
107 on the pre-rewrite base, which had 18 rows and no shape-aware `provider-id`).

### Decision on the seven `extensions` rows: keep them, keyed on production

The review's fidelity analysis is right. `SPEC.md` states the reverse-DNS key
rule as a **local table row** only for Launch Plan (`:1493`), Task-board
Reference (`:1512`), and the two Session Record majors (`:1481`). For Board Goal,
Board Identity, Fork Provenance, and the four derivation-provenance variants it
names `extensions` in a prose member list and never restates the rule, so the
pre-fix cell "object; reverse-DNS extension keys only" was itself a cross-schema
inference.

They are still kept, because the two claims are different claims:

- The **artifact row** says what the pinned document declares. That is the
  fidelity axis this Bug added, and the row now says exactly what `SPEC.md` says
  and nothing more.
- The **reachability gate** says what production enforces here. Production routes
  all eleven `extensions` objects through one shared `validateExtensionsObject`
  (`internal/canonicaljson/closed_shapes.go:1784`), and `README.md` claims that
  for *every* open extensions map. Whether each shape's validator actually
  reaches it is a per-shape fact worth attacking whatever the document restates.

Conflating those two axes is what produced the defect in the first place. The
gate is keyed on the enforcement; the artifact records the document; `README.md`
and LOGBOOK 0050 state which is which.

### Evidence that this was not a cosmetic restoration

Mutants applied to `closed_shapes.go` in a scratch export of the reviewed head
(`git archive 5b8423a`) and to this tree, running the full
`go test ./internal/canonicaljson/ -count=1` both times. Each mutant deletes one
shape's `validateExtensionsObject` call — the shape's `extensions` grammar stops
being enforced on a real production path reachable from both public identity
entries.

| Mutant | rev 1 (reviewed) | rev 2 (this) |
| --- | --- | --- |
| `validateSessionBoardGoal` skips `validateExtensionsObject` | **SURVIVED — package green** | KILLED |
| `validateSessionBoardIdentity` skips `validateExtensionsObject` | **SURVIVED — package green** | KILLED |
| `validateSessionForkProvenance` skips `validateExtensionsObject` | **SURVIVED — package green** | KILLED |
| `validateSessionOriginProvenance` skips `validateExtensionsObject` | KILLED (`TestSessionRecordEveryProvenanceExtensionsGateReachesIdentityProductionEntries`) | KILLED |
| `validateSessionNativeAdoptionProvenance` skips `validateExtensionsObject` | KILLED (same test) | KILLED |
| `env_names` grammar narrowed to `name == ""` (admits `9NAME`, `-NAME`, `NAME-VALUE`, `ÉNAME`) | KILLED by 4 other tests | KILLED, +1 test |

On rev 2 the three survivors are killed by
`TestSessionRecordDeclaredGrammarRowsReachIdentityProductionEntries` and by
nothing else; `TestEveryProductionRefusalGuardIsExecuted` also fails, but only
because it re-runs the shipped suite under coverage and reports its failure.

So the review's "arguably correct, still a finding" reading was too generous to
revision 1. Three shapes could have silently stopped validating extension keys
with the whole package green. The four provenance variants were covered by a
dedicated test and `env_names` by four others — those five were disclosure-only,
as the review said.

### Anti-vacuity for the new pinning

- `TestSessionRecordGrammarRowSetRefusesASilentlyNarrowedEnumeration` removes one
  row at a time from the shipped enumeration — `env_names`, Board Goal
  `extensions`, native-adoption provenance `extensions` — and requires the report
  to name the dropped row and the family count to fall below its pinned value.
  These are **narrowings**: the gate stays present, reachable, and green on every
  row it still carries. That is exactly the mutant that reached review.
- `TestSessionRecordGrammarClassificationIgnoresPinnedSpecProse` replaces every
  row's declaration cell with `L1 “AX Specification”` and requires the
  classification, the row count, and every family count to be unchanged. This is
  the F1 defect executed as a test: a wholesale rewrite of the fidelity column
  must not move a single row.

---

## F2 — reproduced, then closed by a clause anchor

### Reproduced first

Retargeting `ManifestEntry.file.size` from its own `L4746` clause to BlobChunk's
`L4622` `size:uint53[1..4194304]` — a different schema, carrying a bound
`ManifestEntry.file` does not have — left `internal/canonicaljson` green
(`ok ... 18.989s`). Reproduced here, not taken on the review's word.

### Tractability assessment, as the rework brief asked

Tractable, and cheap. `SPEC.md` headings are well-formed ATX with numbered or
appendix identifiers, so a line resolves to its nearest enclosing numbered clause
mechanically. Before pinning anything I measured where each shape's citations
actually land: **all 40 shapes cluster in exactly one clause**, except the two
Session Record majors, which cite Section 2.1 in addition to 5.1 — the
Terms-table name grammar the document's own indirection points away from. No
shape's citations were scattered, so the pin is a 40-row table with no judgement
calls buried in it.

### Implemented

- `specdoc.SectionID(line)` resolves a 1-based line to its enclosing numbered
  clause. An **unnumbered** subheading (`#### Managed Replica Marker document`)
  opens no clause; its body stays attributed to the numbered clause above it,
  which is the clause a citation belongs to.
- `constraintRowSpecSections` pins, per shape, the clauses its citations may use.
  Every verbatim and paraphrase entry's line must resolve into that set.
- `TestEveryConstraintEnumerationShapePinsItsSpecificationClause` asserts the
  pinned shape set **exactly** against the artifact, in both directions, so a new
  shape must declare its clause rather than inherit a free pass and a stale pin
  cannot linger.

### Anti-vacuity

- The F2 mutant is now a twelfth planted row: `ManifestEntry.file.size` retargeted
  to `L4622` fails with `cites Section 10.2, but ManifestEntry.file is declared in
  Section 10.4`.
- `TestClauseAnchorRefusesEveryForeignSectionForOneRow` proves a **bound**, not
  one lucky case. It derives every other clause from the pin, plants a verbatim,
  uniquely locatable line from each into one row, and requires each to be refused
  by clause number: **11 of 11 foreign clauses refused**. It then requires the
  shipped row to still pass, so a check that refused everything would fail.
- `TestSectionIDResolvesTheEnclosingNumberedClause` pins the resolver itself: five
  known line/clause pairs, out-of-range and title lines failing closed, an
  unnumbered subheading not opening a clause, and at most 32 of 12,665 lines
  resolving to no clause — a resolver that silently answered `""` would disable
  the anchor without failing anything.

### The residual limit, stated rather than implied

The anchor is a **clause, not a shape**. Ten shapes are declared in Section 10.4
and two in 10.2, so a citation retargeted *within* a clause —
`ManifestEntry.file` quoting a `GitIndexEntry` row — is still admitted, and the
member anchor is still a substring test another schema's identifier can satisfy.
"Quotes this shape's clause" is what the gate now proves; "quotes this shape's
declaration" is not. The `environment_id` finding sits inside exactly that gap:
both of its lines are in Section 7.8.

Stated in `internal/canonicaljson/testdata/constraint-enumeration.md`
("The clause anchor, and what the check still does not prove") and in `README.md`.

## F3 — fixed

`specdoc.Parse` collapsed blank lines along with every other whitespace run, so a
"verbatim" quote could stitch the tail of one block to the head of the next.
Confirmed admitted on the reviewed head:

```
rev1 QuoteLines("sorted, contiguous, non-overlapping, start at offset zero, and cover
                 exactly <code>size</code> bytes: The descriptor is closed and
                 contains exactly <code>schema</code>") = [4612]
```

Those are two blocks separated by the blank line at `:4614`. Both halves are
individually verbatim, so no other rule would have caught it.

A whitespace run containing a blank line now collapses to `specdoc.BlockSeparator`
(`'\n'`), which `Normalize` can never emit, so the refusal is structural rather
than a special case in the caller. **Zero of the 347 shipped rows relied on the
old behaviour**, measured before changing anything, so the tightening cost
nothing.

Pinned by `TestQuoteLinesRefusesTextThatSpansABlankLine` (each half matches, the
stitch does not), `TestNormalizeNeverEmitsTheBlockSeparator`, and a twelfth
planted row in `TestPlantedConstraintEnumerationRowsRedden` that stitches the
Lease Record table's last row to the paragraph after it. Documented in the
artifact's normalization rule 1 and in `README.md`.

---

## Validation

Run in this worktree, each as a standalone process, real exit codes:

| Command | Exit | Result |
| --- | ---: | --- |
| `gofmt -l .` | 0 | no output |
| `go build ./...` | 0 | — |
| `go vet ./...` | 0 | — |
| `go test ./... -count=1` | 0 | 11 packages ok |
| `go test ./... -cover` | 0 | `canonicaljson` 97.2%, `specdoc` 100.0% |

Planted-defect suite: 13 cases, each required to redden against a copy of the
shipped artifact, paired with `TestUnmodifiedConstraintEnumerationIsAdmitted`.
Three are narrowings rather than deletions (true quote at the wrong line, true
quote stripped of the member anchor, true quote from another shape's clause).

Logs: `.temp/BUG-260902-3dn8jd/rev2-go-test.log`,
`.temp/BUG-260902-3dn8jd/rev2-go-cover.log`.

## Scope

`internal/canonicaljson/session_record_versions_test.go`,
`internal/canonicaljson/constraint_excerpt_test.go`,
`internal/canonicaljson/testdata/constraint-enumeration.md` (methodology section
only — no row's shape, member, constraint, call site, or excerpt changed),
`internal/specdoc/specdoc.go` (block separator, `SectionID`),
`internal/specdoc/specdoc_test.go`, `README.md`,
`LOGBOOK.md` (entry 0050). No production source changed.
