# BUG-260902-2luo5h — review verdict: ACCEPTED

Reviewer run `RUN-260902-729ab0`. Change Request `CR-BUG-260902-2luo5h-1` revision 1,
base `a031128`, candidate tree `716683fda12c8bb060743d0b404b8d6b4e80a3f4`
(= worktree HEAD `b16266a`), repository delta `present`, 6 files.
Full attack evidence: `BUG-260902-2luo5h_review-attack.log`.

## Verdict

Accepted. The absent third of SPEC.md:4768-4769 is now enforced at both public
identity entries, every acceptance-criteria case is individually load-bearing
under reviewer-authored narrowing mutants, and the artifact row that hid the
gap now quotes the whole sentence including the narrowing that remains.

## What was verified, not read

**The gate is reached from production.** `validateManifestEntries` is called
only from `validateTransferManifest` (`closed_shapes.go:948`), which is the
registered validator for the Transfer Manifest schema (`closed_shapes.go:65`),
and it runs unconditionally for all five `kind` arms before the tagged-field
switch. The tests drive `CalculateObjectIdentity` and `VerifyObjectIdentity`
through `assertIdentityEntriesRefuseWithReason`; nothing calls the validator
directly. No second copy of the validator exists in the tree.

**The bug reproduces at base and the new tests fail there.** The two new test
files overlaid on a pristine `a031128` tree produce six failing subtests —
including the three the bug reported — while
`TestManifestEntryOverlapAdmitsDirectoryParents` already passed at base. The
acceptance cases are therefore not wins manufactured by the fix.

**Eight reviewer mutants, eight killed, no survivors.** Six of them are
narrowings rather than deletions, which is what the AC clause "each reddens when
only the overlap clause is weakened" actually asks for:

| Mutant | Result | Cases that died |
| --- | --- | --- |
| Delete the whole clause | KILLED | all six refusal cases |
| Owners restricted to `file` | KILLED | symlink-over-file, hardlink-over-file |
| Owners restricted to non-`file` | KILLED | the four file-owner cases |
| Close an owner on the first non-descendant | KILLED | intervening `a.txt` sibling |
| Single pop instead of stack drain | KILLED | intervening `a.txt` sibling |
| Ancestor match without the `/` separator | KILLED | both prefix-sibling acceptance cases |
| Directories also own subtrees | KILLED | two acceptance cases + nested owner |
| Refuse only strict grandchildren | KILLED | all six refusal cases |

The `file`-only / non-`file`-only pair is the important one: it proves no
entry-type arm of the refusal is redundant. The two over-refusal mutants prove
the acceptance cases are load-bearing, so the clause cannot be "satisfied" by a
validator that refuses every nested manifest.

**The order-reuse algorithm is correct, not merely plausible.** Paths are
strictly bytewise increasing and `scalar.ParseRelativePath` rejects empty, `.`
and `..` segments, so no path carries a trailing or doubled separator. For an
open owner `p` with `S = p + "/"`: every descendant of `p` has `S` as a prefix
and so exceeds `S`. If the current path `q` exceeds `S` without that prefix,
they first differ at some index `i < len(S)` with `q[i] > S[i]`, so every string
prefixed by `S` is strictly below `q` and cannot appear later in a sorted list —
popping is safe. If `q <= S`, later paths may still descend into `p`, and the
loop breaks without popping — keeping is safe. An owner pushed above `p` is
strictly below `S`, so its own subtree bound is below `S` too, and an early
break above it cannot skip a descendant of `p`. Cost is one push and one pop per
entry, so the 65,536-entry production guard is unaffected; the suite's timing
gate stayed green.

**Nine additional shapes probed through the production entry** beyond the
producer's suite — symlink-over-directory-over-file, symlink over a deep
grandchild, hardlink over a directory, a file owning its own subtree under a
declared directory, two open owners with an escape into the deeper one — all
refused with the correct owner named. Legitimate shapes, including the shipped
`p/00000`-with-no-`p` fixture pattern and a mixed deep tree, stay accepted.

**Every declared gate rerun by the reviewer**: build, vet, gofmt, `go test ./...
-count=1 -cover` (9/9 packages ok), tracecheck, cataloggen `-check`, and all
five seeded fuzz targets, each rc=0. `internal/canonicaljson` coverage is 97.2%
at the candidate and 97.2% measured independently at base `a031128` — no
regression, verified rather than accepted from the report.

## Acceptance criteria

| Criterion | Status |
| --- | --- |
| Entry-local overlap refused at both public identity entries | Met; refusal reaches `CalculateObjectIdentity` and `VerifyObjectIdentity`, file/symlink/hardlink ancestors all refused |
| Negative cases cover file-over-file, symlink-over-file, file-over-directory | Met, plus hardlink-over-file, an ordering-trap sibling and a nested owner |
| Each case reddens when only the overlap clause is weakened | Met; proven by reviewer mutants M2, M3, M4, M5, M8, not only by deletion |
| Sorted order reused rather than re-derived | Met; a stack over the existing single scan, no per-path ancestor enumeration |
| Enumeration row states the whole rule | Met; the row now quotes the full sentence, states that all three properties are entry-local, names the enforcer and the pinning tests |
| Parent that is not a declared directory refused "where the spec requires one" | Correctly declined; §13.14 and §10.4 impose no declared-parent rule, and the shipped `p/00000` fixture has no `p` entry. Adding it would have been an over-refusal, and the decision is disclosed in the logbook rather than silently assumed |
| Repository gates and fuzz targets green, no coverage regression | Met, rerun independently |

## Front-loaded DoD items checked explicitly

**"Never invent what you add" / the quote must be literal.** The enumeration row
quotes `Entries and child partitions MUST contain no duplicate, overlapping, or
destination-case-colliding path.` That string is literally present in the pinned
SPEC at `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c` (the commit named in
`internal/specpin`), spanning the wrap between SPEC.md:4768 and :4769 — checked
by extracting the row's quoted span and matching it against the whitespace-
normalized pinned blob, not by eye. Nothing in the implementation extends past
the sentence: no declared-parent rule was invented, and the bytewise-only
comparison is stated in the row rather than assumed.

**"A negative case must be refused by the clause it names."** Proven by
construction rather than by inspection: on a pristine base tree `a031128`, with
only the two new test files overlaid, every one of the six overlap fixtures is
ACCEPTED. The pre-existing sort, duplicate and case-fold checks therefore pass on
all of them, so the new clause is the only thing that can reject them on the
candidate.

**Bounds and adjacency.** The change introduces no new bound, so there is no
limit to pin. The existing 65,536-entry guard is unaffected: the scan adds one
push and one pop per entry, and the package suite (which contains that timing
guard) stays green at 19.4s.

## Residual, disclosed by the producer, recommended as a follow-up

Overlap is compared bytewise, so an overlap that exists only after destination
case folding is still accepted. Confirmed by probe against the production entry
on the candidate tree:

- symlink `S` together with file `s/x` — ACCEPTED
- file `A` together with file `a/b` — ACCEPTED
- file `s` together with file `S/x` — ACCEPTED
- file `k` together with file `K` (U+212A Kelvin) `/x` — ACCEPTED

On a case-insensitive destination this is the same materialization-escape
primitive the bug names: `s/x` opens through symlink `S`. It is *not* grounds to
reject this change — it is outside the reported repro set and outside the AC's
named cases, the change is a strict safety improvement, and the producer
disclosed the narrowing in both the enumeration row and the logbook instead of
letting a half-quoted row read as complete, which is exactly the failure mode
this bug was about.

The producer's justification is accurate but narrower than stated: folding does
not preserve byte order, so *this* scan cannot run over folded paths. It does
not follow that the property is unenforceable — a second pass over fold-keyed
paths would reach it, and the existing `foldedPaths` set is not sufficient on
its own because a folded ancestor does not always sort before its folded
descendant (`s` vs `S/x`). Recommend a follow-up Bug for entry-local
case-folded overlap; the evidence is in the probe section of the attack log.

Second, smaller observation, no action required here: the constraint table this
change corrected is not machine-checked. `constraint_inventory_test.go` parses
`constraint-enumeration.md` only for the `requireExactMembers` member rows; the
normative-constraint narrative rows are verified by reading alone. That is
structurally why a half-quoted row could pass review for as long as it did.

## Handoff

Reviewer-archetype run, so no `commit_ack`. Acceptance recorded with
`accept_cr(BUG-260902-2luo5h, revision=1, evidence=BUG-260902-2luo5h_review-verdict.md)`,
which parks the element at `to-review` for the orchestrator to checkpoint or
integrate and to make the `done` transition with `commit_ack=scope_committed`.
