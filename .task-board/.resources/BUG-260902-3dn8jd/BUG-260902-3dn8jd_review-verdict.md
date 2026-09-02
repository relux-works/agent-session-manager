# BUG-260902-3dn8jd — review verdict: CHANGES REQUESTED

Reviewer run `RUN-260902-f3e5b6`. Change Request `CR-BUG-260902-3dn8jd-1` revision 1.
Base `422786cc5b4303f03d3971caa509ac12b49a00c6` -> candidate tree
`9e9fd281a3c5629759f28f493bf50b415daa1be7` (worktree HEAD `5b8423a`, verified).

The acceptance criteria of this Bug are met and the gate survives attack. It is
refused on a collateral defect the change introduces: an undisclosed 38%
narrowing of a different, pre-existing reachability gate, caused by the artifact
rewrite and hidden by a non-empty-style guard — the same weakness this Bug
exists to remove, one level up.

## What was attacked, and what held

Every claim below was reproduced by mutating a scratch copy of the tree
(`/tmp/axmut`, `/tmp/axbase`) and running the real production entry points. The
reviewed worktree was never modified (`git status --short` clean throughout).

| Attack | Result |
| --- | --- |
| Plant an invented quote into the **shipped** artifact (`Blob Descriptor.media_type` -> "a lowercase ASCII digest of the blob") | REDDENS. `TestConstraintEnumerationSpecExcerptsQuoteThePinnedSpecification`, `TestArtifactQuotesAreVerbatimPinnedSpecificationText`, `TestUnmodifiedConstraintEnumerationIsAdmitted` all fail and name the row and the absent text. |
| Perturb one character of the embedded `internal/specdoc/SPEC.md` (`[1..255]` -> `[1..256]`) | REFUSED, fails closed. Every excerpt gate `t.Fatalf`s on `pinned specification document mismatch`; none skips or degrades to a pass. |
| Mint the document to fit the artifact | IMPOSSIBLE. `specpin.DocumentSHA256` is unchanged by this diff (`internal/specpin/pin.go:30`, also in `v0.5.0.lock.json`) and the embedded `SPEC.md` hashes to exactly it. The digest predates the change; the vendored document cannot have been written to match a fabricated excerpt. |
| Escape verbatim comparison through the `paraphrase:` hatch | Available but unused: 0 of 347 shipped rows use it. Refusals for out-of-document and member-less paraphrases are proven by the producer's plants and reproduce. |
| Delete-only vs narrowing mutants | The producer's 11 plants include two genuine narrowings (real text moved to the wrong line; real line stripped of the member anchor), paired with `TestUnmodifiedConstraintEnumerationIsAdmitted`. Not a delete-only suite. |
| Repository validation | `gofmt -l` empty, `go build ./...`, `go vet ./...`, `go test ./... -count=1` all green (11 packages, including `tracecheck`). |

Independent audit of the pre-fix artifact confirms the Bug's premise at scale:
of the 347 base rows, 139 quoted text absent from the pinned document under
whitespace-only normalization (the producer measured 149 with its own decoding;
same order, same conclusion). LOGBOOK 0022 discloses that count, the 161
ambiguous rows, the `environment_id` cross-schema finding, the fabricated
"Section 2.1 grammar" citation, and the out-of-scope `closed-vocabularies.md`
gap. The `Enforced constraint` column changed for exactly 1 of 347 rows, and
that change is a disclosure, not an accommodation. Provenance discipline here is
good.

## F1 — BLOCKING: the artifact rewrite silently narrowed a reachability gate

`sessionRecordGrammarFamily` (`internal/canonicaljson/session_record_versions_test.go:588`)
classifies a row into an executable grammar family by **substring-matching the
row's `specExcerpt` prose**. Rewriting that column from informative prose to
verbatim SPEC fragments removed the words it matches on, so rows fell out of
`TestSessionRecordDeclaredGrammarRowsReachIdentityProductionEntries`.

Measured, not inferred — same command on both trees:

```
go test ./internal/canonicaljson/ -count=1 \
  -run TestSessionRecordDeclaredGrammarRowsReachIdentityProductionEntries -v
```

| | base `422786cc` | candidate | delta |
| --- | ---: | ---: | ---: |
| rows entering the loop | 18 | 11 | -7 |
| subtests executed | 107 | 66 | -41 |
| `reverse-dns` family rows | 11 | 4 | -7 |
| `environment-name` family rows | 2 | 1 | -1 |
| `provider-id` family rows | 1 | 2 | +1 |

Eight rows stopped being attacked with invalid grammar values through both
public identity entries:

- `Session Record Board Goal.extensions` — "object; reverse-DNS extension keys only" -> `L1521 "greater than zero, and <code>extensions</code>"`
- `Session Record Board Identity.extensions`
- `Session Record Fork Provenance.extensions`
- `Session Record origin provenance.extensions`
- `Session Record same-provider-fork provenance.extensions`
- `Session Record cross-environment-clone provenance.extensions`
- `Session Record native-adoption provenance.extensions`
- `Session Record Launch Plan.env_names` — "sorted unique environment names" -> a verbatim row that spells the grammar but never says "environment names"

The new excerpts are true; that is the point. Checked each drop against the
pinned document rather than assuming a regression:

- The seven `extensions` drops are **fidelity-justified**. `SPEC.md` states the
  reverse-DNS rule as a local table row for exactly the two shapes that
  survived — Launch Plan (`:1493`) and Task-board Reference (`:1512`). For Board
  Goal, Board Identity, Fork Provenance, and the four provenance variants the
  document states `extensions` only inside a prose member list and never
  restates the rule, so the pre-fix cell "object; reverse-DNS extension keys
  only" was itself the cross-schema inference this Bug exists to remove. Losing
  those seven from the loop is arguably *correct*. It is still a finding — the
  work has learned that seven shapes' `extensions` grammar was being tested on
  the strength of an invented declaration — and this Bug's own disclosure duty
  says a finding is reported, not silently absorbed.
- `Session Record Launch Plan.env_names` is a **pure, unjustified regression**.
  Its new excerpt at `L1490` quotes the grammar verbatim, including
  `[A-Za-z_][A-Za-z0-9_]{0,127}`. Nothing about its fidelity changed. It left
  the loop only because the classifier greps for the words "environment names",
  which the true quote does not contain. That row should still be attacked with
  invalid values and no longer is.

So the rewrite traded excerpts that were *informative but partly fabricated* for
excerpts that are *true*, and that prose was load-bearing for a second gate. Six
of the eight drops are defensible; one is not; and none of the eight is
disclosed.

Severity: the underlying reverse-DNS rule is one shared validator
(`closed_shapes.go:1790`), so production enforcement is still exercised by the 4
surviving rows. What was lost is per-shape **reachability** — precisely what the
test's name promises. LOGBOOK 0022 discloses the `name` row reclassification and
calls it "a stricter key"; it says nothing about the eight rows that left.

Fix direction (producer's call, this is not a prescription):
1. Restore `Session Record Launch Plan.env_names` to the `environment-name`
   family. Key `sessionRecordGrammarFamily` on shape+member, or on the quoted
   grammar itself, so classification cannot depend on prose the fidelity gate is
   actively rewriting.
2. Replace `familyCounts[family] == 0` with a pinned per-family expectation (or
   a pinned row set) so a future rewrite that drops rows reddens instead of
   passing.
3. Disclose the seven `extensions` drops in LOGBOOK as what they are: seven
   shapes whose reverse-DNS grammar was being tested on the strength of a
   declaration the pinned document does not make for them. That is a finding of
   the same class as the two already recorded, and it is the disclosure duty
   this Bug wrote for itself.

## F2 — record, non-blocking: the gate is shape-blind

Proven. Retargeting `ManifestEntry.file.size` from `L4746 "<code>size:uint53</code>"`
to `L4622 "<code>size:uint53[1..4194304]</code>"` — BlobChunk's clause, a
different schema, a bound `ManifestEntry.file.size` does not carry — leaves the
**entire** `internal/canonicaljson` package green (`ok ... 19.081s`).

The gate proves the characters exist at the cited line and that the member name
appears in them. It cannot tell whether the cited clause belongs to the row's
shape. `strings.Contains(text, row.member)` also matches a member name embedded
in another schema's identifier. This is the same cross-schema class as the
`environment_id` finding the producer surfaced *by hand* — the gate would not
have caught that one either.

No overclaim was made: README states the rule as implemented. But neither README
nor the artifact's own "How the Pinned SPEC declaration column is checked"
section states this limit, and a reader is entitled to know that "quotes the
pinned specification" does not mean "quotes this shape's declaration".

## F3 — record, non-blocking: normalization crosses block boundaries

`specdoc.Normalize` collapses every whitespace run including blank lines, so a
"verbatim" quote can stitch the tail of one paragraph to the head of the next —
or a heading to its body. Admitted, verified:

```
L4613 "<code>size</code> bytes: The descriptor is closed and contains exactly <code>schema</code>"
```

Those are two separate blocks in `SPEC.md` separated by a blank line. The
documented rule (`specdoc.go` doc comment, README, artifact) says it forgives
"hard line wrapping and table indentation". It also forgives paragraph and
section boundaries. The DoD asks for the normalization rule to be stated
explicitly and to be no looser than benign formatting; either state this case or
treat a blank line as a non-collapsible separator.

`TestNormalizeForgivesOnlyWhitespace` pins the rule well from the other side —
case, punctuation, and backtick-for-`<code>` variants all stay unmatched. This
is a documentation-vs-behavior gap, not a broken rule.

## Verdict

`changes_requested` -> `to-dev`, for F1 only. F2 and F3 are recorded for the
producer's judgement and do not by themselves require rework. The specdoc design
(vendor + digest-verify, refuse rather than compare) is the right call and its
negative coverage is real.
