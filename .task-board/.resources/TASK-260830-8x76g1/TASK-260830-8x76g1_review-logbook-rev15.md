# TASK-260830-8x76g1 — CR rev 15 reviewer logbook

## The generalization is where triage rots

The rev14 review asked for every clause-sweep survivor to be triaged "with proof
that it is dead code or genuinely subsumed, never with plausibility". Twenty-four of
the twenty-five rows meet that bar, and they meet it well — each names an input class,
has an executable case that drives it through both public identity entries, and the
sweep shows the clause's disablement leaves that case still refused. That is a real
mechanized subsumption proof, not prose.

The one row that fails is the one written as a **universal quantifier over callers**:
"Every caller applies exact grammar, a scalar validator, or a minimum length." Nine
`requireString` call sites; eight satisfy it; the ninth
(`validateManifestEntries`, symlink `target`) does not. The lesson worth keeping: a
triage row scoped to *one input the suite already drives* is checkable and usually
right; a triage row scoped to *all callers of a helper* is a claim about a set the
author did not enumerate, and it is exactly where a false one hides. When a survivor's
subsumption depends on the caller rather than on the same function, the enumeration
has to be written out and each member probed, not asserted.

## How to tell the difference cheaply

Apply the survivor's own mutant, then drive every member that reaches that clause
through both public entries in one probe and print ATTESTED/REFUSED per member. Nine
members, one test file, two runs. That converts a universal claim into a table:

```
PROBE symlink target           ATTESTED calc=<nil> verify=<nil>
PROBE hardlink target_path     REFUSED  ... path: must be non-empty valid UTF-8
PROBE board logical_id         REFUSED  ... must contain 1..128 Unicode characters
... six more REFUSED
```

The green suite under that same mutant (`exit 0`) is the point: nothing in the
repository notices. A clause-sweep survivor is not evidence of redundancy, it is
evidence of *silence*, and the two are only distinguishable by driving the clause's
input class deliberately.

## Bound coverage is asymmetric by construction

`ManifestEntry.symlink.target` is declared `string[1..4096]`. The `4096` end is a
numeric literal in `closed_shapes.go:797`, so the rev12 `genmutants.py` generates a
mutant for it and it is KILLED. The `1` end is not a literal anywhere — it is the
implicit consequence of `requireString`'s `text == ""` in a shared helper. A
numeric-bound generator structurally cannot see it. This is the same class of blind
spot rev14 found (the harness could not express a tagged-union arm or a grammar
predicate), reappearing for *implicit* bounds carried by a shared helper rather than
by a literal at the constraint's own site. When auditing declared `[m..n]` bounds,
check where `m` physically lives before believing the sweep covered it.

## Anomaly: disk exhaustion mid-review

The root volume hit 100% (148 MiB free of 926 GiB) partway through this review, with
54 GB in `~/Library/Caches/go-build` accumulated across fifteen revisions of mutation
sweeps. `go clean -cache` restored 54 GiB and the review continued. Worth knowing for
any future leaf that runs sweeps at this scale: a 110-mutant uncached sweep plus a
71-mutant sweep, repeated per revision, is a multi-GB-per-cycle cache cost. No
repository or board state was affected; the candidate tree hashed to
`1617e6afc56110589f6f0f524ceb94de192b43b5` before and after.

## Integrity discipline that paid off

Hashing the worktree into a throwaway `GIT_INDEX_FILE` (never the real index) before
review, after the sweep, after each probe, and after the gates caught nothing wrong
this time — but it is what makes "I mutated production 111 times and the candidate is
byte-identical" a statement rather than a hope. Restoring from a pre-hashed
`closed_shapes.go.orig` and re-verifying its SHA-256 after every mutant is the other
half.
