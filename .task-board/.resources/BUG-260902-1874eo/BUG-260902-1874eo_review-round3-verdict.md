# BUG-260902-1874eo — review round 3 verdict: ACCEPTED

Reviewer run against `CR-BUG-260902-1874eo-2` revision 2.
Worktree `.temp/STORY-260902-xpwppu/worktree`, branch `task-board/story/STORY-260902-xpwppu`.
Full mutant sweep: `BUG-260902-1874eo_review-round3-mutants.log`.

## First: the Change Request says `repository_delta=empty`, and that is a base-construction artifact, not an empty delivery

The spawn prompt asked me to state why no repository change was the right outcome.
Answering that question on its own terms would have accepted a fiction. The producer
changed 732 lines.

| Value | OID |
| --- | --- |
| `git rev-parse HEAD` | `8a0dcedf7d1d4a932eae335df2772effe3923cee` |
| CR `base_oid` | `8a0dcedf7d1d4a932eae335df2772effe3923cee` |
| `git rev-parse HEAD^{tree}` | `6b1134ed361b7617fda0f6a48cbc9ab3aba0458a` |
| CR `candidate_tree_oid` | `6b1134ed361b7617fda0f6a48cbc9ab3aba0458a` |
| workspace `WS-6f2050d0ba66` `checkpoint_oid` | `422786cc5b4303f03d3971caa509ac12b49a00c6` |
| `git rev-parse HEAD^` | `422786cc5b4303f03d3971caa509ac12b49a00c6` |

The base OID **is** the branch head, so `base..candidate` is empty by construction. The
branch is exactly one signed single-parent commit past the checkpoint — the correct leaf
shape. The reviewable window is `422786cc..8a0dced`, and that is the delta I reviewed:

```
LOGBOOK.md                                      |  27 ++
README.md                                       |  26 ++
internal/canonicaljson/canonical_test.go        |  35 ++
internal/canonicaljson/closed_shapes.go         |  42 ++
internal/canonicaljson/utf8_subsumption_test.go | 602 ++++++++++++++
5 files changed, 732 insertions(+), 0 deletions(-)
```

Zero deletions across the whole commit. This is the known snapshot-ordering trip: the
producer committed before the CR snapshot was taken. **Orchestrator action**, not producer
rework: when checkpointing this accepted revision, advance `checkpoint_oid` to `8a0dced`,
or every later leaf on this branch reads as two-plus commits past checkpoint.

## Part 1 — the two normative digests. Accepted.

The AC's load-bearing clause is "expected values taken from the pinned SPEC rather than
recomputed from the implementation, **so the test can fail**". I verified the whole
provenance chain myself rather than accepting the producer's account:

- `shasum -a 256 .temp/BUG-260902-1874eo/SPEC.pinned.md` = `562546d2…ac484a`, equal to
  `internal/specpin.DocumentSHA256` at `internal/specpin/pin.go:30`, pinned to commit
  `28bf96d7…` at `pin.go:28`.
- `SPEC.pinned.md:303` and `:304` are quoted verbatim in the test comments and carry
  `bb80eb37…` and `b0ec84c6…`.
- Recomputed independently of the encoder, straight from the fixture bytes with `shasum`:
  both match. The published contract claims reproduce.

Non-vacuity, four mutants, all killed:

| Mutant | Change | Result |
| --- | --- | --- |
| A | `Canonicalize` appends one byte | all four subtests redden |
| B2 | NUM-U64-STRING fixture, one byte | only its own subtest reddens |
| C2 | NUM-U64-MAX fixture, one byte | only its own subtest reddens |
| R1 (reviewer) | swap the two `want` digests | both redden |

B2/C2 prove each pin is individually decisive rather than carried by its sibling. R1 is my
addition: it proves each pin binds *its own* published value, not "some digest from the
table". Style matches the pre-existing NUM-SAFE-MAX assertion at `canonical_test.go:498`.

Attacked the README's capability claim too. `grep -n "SHA-256 <code>"` over the entire
pinned SPEC returns exactly four rows — `:300`, `:303`, `:304`, `:307`. README names
exactly those four as pinned, and all four are asserted. The claim reproduces.

One property I want on the record because the AC's phrase "or the canonicalization
reddens" is looser than it sounds: both new fixtures are already in JCS form, so
`Canonicalize` is byte-identity on them and an identity-function mutant would keep these
two subtests green. That is a property of the SPEC fixtures — they are published in
canonical form — and it is identical to the pre-existing NUM-SAFE-MAX pin. Canonicalization
semantics are covered by the sibling `UTF-16 order` and Appendix B assertions, which this
change did not touch. Not a defect in this leaf; recorded so nobody re-derives it later.

## Part 2 — the subsumed UTF-8 re-checks. Accepted.

Round 1 rejected this on a specific failure: an AST call graph that only saw `f(x)` covered
3 of 7 guards while presenting as complete, and mutant H survived it. I did not take the
producer's claim that this is fixed — I rebuilt H and six more mutants from scratch.

| Mutant | Shape | Result |
| --- | --- | --- |
| H | exported `ValidateDecodedRecord(map[string]any)` via the dispatch table | **KILLED**, names all 7 guards |
| I2 | exported method, `[]byte` param, decoded receiver | **KILLED**, names all 7 |
| D2 | exported map-taking entry point, direct call | **KILLED** |
| G2 | second `json.Unmarshal` outside `decodeStrict` | **KILLED**, file/line/function named |
| E3 | delete one subsumption comment | **KILLED** |
| F3 | narrowing — comment kept, names `requireExactMembers` | **KILLED** |
| N1 | a brand-new undocumented re-check | **KILLED** |
| V1 | orphan guarded function, reachable from nothing exported | **KILLED**: reports `7/8`, names it |
| B1 | dispatch through a func-typed struct field | **SURVIVED — as published** |

H is the round-1 survivor and it is dead. F3 is the narrowing mutant that matters: the
check binds the *named* subsuming validator, not merely the presence of a comment, so this
is not a delete-only proof. N1 confirms the guarded set is derived from the AST rather than
a hardcoded list. V1 is the one I care about most — it is the check that would have caught
round 1's silent 3/7, and it reddens with `coverage is 7/8; [reviewerOrphanRecheck] are
reachable from no exported entry point`. The ratio is measured, not asserted as a constant.

B1 survives, and that is the right outcome rather than a finding. `closed_shapes.go:57-61`
and README both name "a func-typed struct field" among the constructions the graph does not
model, and give the reason (`x.M(...)` resolves only for methods declared in this package).
I built the mutant specifically to test whether the published bound is honest. It is: the
survivor is exactly the shape the code says will survive, and it is the stated reason the
guards stay live rather than being deleted. A pin that states its bound and then reproduces
that bound under attack is the opposite of the false guarantee round 1 rejected.

Subsumption argument verified at source, not read off the comments: `canonical.go:328`
`!utf8.Valid(input)` refuses before `canonical.go:335` `json.NewDecoder`, inside
`decodeStrict` at `:324`. The board note's "canonical.go:311 is now :328" is accurate. The
keep-rather-than-delete rationale also reproduces: `refusal_guards_test.go:194-195` registers
two of the eight sites as `invalidUTF8Refusal`, so deleting them would have rewritten pinned
refusal statements.

DoD item on comment style: the new comments are `//` blocks immediately above the guarded
statement naming the subsuming validator, plus one shared file-level note. That is the
`internal/localstore/paths.go:233-236` convention (`// It is independently subsumed by
scalar.ParseAbsolutePath below…`), not a new style.

Scope correction accepted: the audit's four line numbers (`:258 :1506 :1525 :1760`) match
commit `7b94c9ad`, which carries eight sites with no property separating the cited subset.
Resolving eight rather than four is the correct reading — "the sweep reports no unexplained
survivor" is false while four identical ones remain.

## Validation I ran myself

`go build ./...`, `gofmt -l internal/` (clean), `go vet ./...`, `go test ./... -count=1`
(all 10 packages ok), `go run ./internal/traceability/cmd/tracecheck` (ok: contracts=60
normative_sections=36 acceptance_cases=43 fixtures=30 compatibility_contracts=55),
`go test ./internal/canonicaljson/ -cover -count=1` → **97.2%**, matching the commit's
claim. `git verify-commit 8a0dced`: good ECDSA signature for `oparin@me.com`, author
Ivan Oparin, `%G?` = `G`. Tree clean before and after the sweep.

I did not re-run the 100x fuzz; that evidence stands as attached by the producer.

## Disclosed, accepted, and carried forward

1. **`internal/canonicaljson/core_records.go:278`** (`validateProviderIdentityRecord`) is a
   ninth re-check of the same class with no subsumption comment.
   `TestEveryUTF8RecheckDeclaresItsSubsumption` only scans `closed_shapes.go`, so it is not
   covered. It *is* covered by the entry-point and coverage pins (it appears in the 7/7 and
   in every H/I2 failure message). Outside this bug's stated `closed_shapes.go` scope and
   disclosed by the producer as FOUND-NOT-FIXED. **Wants its own board item.**
2. **Nothing derives the set of digest-publishing §1.6 fixtures from the pinned document**,
   because `SPEC.md` is not vendored — so a fifth published fixture added upstream cannot be
   caught as unpinned by a test. Correctly disclosed in LOGBOOK. Worth a board item.
3. **`BUG-260902-9uwnm7` is in `backlog`** while its part-2 work ships in this commit, and
   its board note names commit `454c2db`, which is not reachable from `HEAD` (it is the
   abandoned revision-1 commit). The delivered commit is `8a0dced`. Bookkeeping only — the
   note's substantive claim, "delivered together with BUG-260902-1874eo in one Change
   Request", is true. **Orchestrator action:** correct the hash and route 9uwnm7 out of
   `backlog` alongside this acceptance.

## Verdict

**ACCEPTED.** Both parts hold under attack. Part 1's expected values come from the pinned
SPEC with its digest verified, reproduce against an independent `shasum`, and are each
individually decisive. Part 2 kills the round-1 survivor, measures its own coverage as a
ratio instead of narrating it, and publishes a bound that reproduces exactly under a mutant
built to break it. No existing assertion was weakened or deleted and no production logic
changed.

Acceptance recorded with `accept_cr`; the element parks at `to-review` for the orchestrator
to checkpoint and make the `done` transition with `commit_ack=scope_committed`. Reviewer
runs do not supply `commit_ack`.
