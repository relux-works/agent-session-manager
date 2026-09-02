# BUG-260902-2luo5h — refuse entry-local Transfer Manifest path overlap

Branch `task-board/story/STORY-260902-1evi33`, commit `b16266a`, one commit past
checkpoint `a031128`.

## Defect

`validateManifestEntries` (`internal/canonicaljson/closed_shapes.go`) enforced
the duplicate and destination-case-collision halves of SPEC.md:4768-4769 and
dropped the overlapping half entirely. On the checkpoint these were ACCEPTED:

- file `a` together with file `a/b`
- symlink `s` together with file `s/x`
- file `a` together with directory `a/b`

A symlink or file parent over a real child is a materialization-escape
primitive, so this is a path-safety hole, not a cosmetic gap. Mutation testing
could not surface it by construction: there was no clause to mutate.

## Fix

The loop already proves the entries strictly bytewise sorted. Every ancestor of
a path is a proper prefix ending at a `/`, so it sorts before that path, and the
still-open non-directory owners form a stack. Detection reuses that order rather
than re-deriving each path's ancestor set:

- refuse when the current path starts with `owner + "/"`, naming the owner tag;
- otherwise close the owner only once the path sorts strictly above
  `owner + "/"`, which is exactly the point past its whole descendant range;
- push every non-directory entry as an owner.

The closing rule is the non-obvious half. `.` (0x2E) sorts below `/` (0x2F), so
`a.txt` sits between `a` and `a/b`. An implementation that closes an owner on
the first non-descendant path loses the `a` owner before its escaping child
arrives and admits the overlap. Mutant M5 is exactly that implementation and is
killed by a dedicated case.

Overlap is compared bytewise, not case-folded: a case-only overlap such as `A`
over `a/b` is refused only where the existing duplicate/case-collision clause
already reaches it. That is a disclosed narrowing, recorded in the artifact row.

Parent directories are NOT required to be declared. SPEC.md §10.4/§13.14 states
no such rule, and the shipped fixture `transferManifestWithEntries(count)`
already declares `p/00000` entries with no `p` entry. Inventing that refusal
would have been an over-refusal beyond the spec.

## Artifact correction

`testdata/constraint-enumeration.md:415` narrowed "overlapping" to the external
child-partition case, so the row read as enforced while the entry-local half was
absent. The row now quotes the whole sentence, states that all three properties
are entry-local, names the production enforcement and both pinning tests, and
keeps only the child-partition closure external.

## Negative evidence

`TestManifestEntryOverlapRefusalReachesBothIdentityEntries` drives
`CalculateObjectIdentity` and `VerifyObjectIdentity` — the two public identity
entries — and pins the exact overlap message per case, so a neighbouring refusal
reporting a different problem cannot stand in for it:

| Case | Entries | Pinned refusal |
| --- | --- | --- |
| file over file | `a` file, `a/b` file | `ManifestEntry[1] path "a/b" overlaps earlier file entry "a"` |
| symlink over file | `s` symlink, `s/x` file | `... overlaps earlier symlink entry "s"` |
| file over directory | `a` file, `a/b` directory | `... overlaps earlier file entry "a"` |
| hardlink over file | `a.bin` file, `h` hardlink, `h/x` file | `... overlaps earlier hardlink entry "h"` |
| intervening sibling | `a` file, `a.txt` file, `a/b` file | `ManifestEntry[2] ... overlaps earlier file entry "a"` |
| nested owner | `a` dir, `a/b` file, `a/b/c` file | `... overlaps earlier file entry "a/b"` |

`TestManifestEntryOverlapAdmitsDirectoryParents` keeps the refusal narrow:
declared directory parents carry children, and a shared byte prefix (`a` /
`a.txt`, `s` / `s.txt`) is not an ancestor. Without it the clause could be
"proven" by a validator that refuses every nested manifest.

## Each fixture is refused by the clause it names, and by nothing else

Under mutant M1, which deletes only the overlap clause, all six negative
fixtures are ACCEPTED with `error = <nil>` at `CalculateObjectIdentity`:

```
clause_refusal_test.go:635: CalculateObjectIdentity(manifest_id malformed shape) error = <nil>, want identity refusal containing "ManifestEntry[1] path \"a/b\" overlaps earlier file entry \"a\""
... one per case, all six <nil>
```

That is the proof the strict-sort and simple-fold checks already present PASS on
every overlap fixture: nothing but the new clause can reject them. A fixture
that also tripped an earlier check would have produced that earlier refusal here
instead of `<nil>`.

The enumeration row quotes SPEC.md verbatim. At the pinned revision
`28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`, SPEC.md:4768-4769 reads "Entries and
child partitions MUST contain\nno duplicate, overlapping, or
destination-case-colliding path."; the row collapses that hard wrap to a space,
exactly as the sibling `array-order-constraints.md` row already does. No new
bound was introduced, so there is no limit/one-past pair to prove.

## Mutants — 7 generated, 7 killed, 0 survivors

Full log: `BUG-260902-2luo5h_overlap-mutants.log`. Every mutant weakens ONLY the
overlap clause; each acceptance case stays green throughout.

| Mutant | Weakening | Cases reddened |
| --- | --- | --- |
| M1 | delete the clause | all six |
| M2 | refuse only when the owner is a `file` | symlink-over-file, hardlink-over-file |
| M3 | skip when the descendant is a directory | file-over-directory |
| M4 | skip when the descendant is a file | all but file-over-directory |
| M5 | close an owner on the first non-descendant path | intervening sibling |
| M7 | refuse only when the owner is not a `file` | file-over-file, file-over-directory, intervening sibling, nested owner |
| M6 | narrow overlap to an exact duplicate path | all six |

M2 and M7 are the pair that proves no tag arm is redundant; M3 and M4 prove the
descendant side is unconstrained; M5 proves the closing rule.

## Gates — all exit 0

Full log with per-command exit codes: `BUG-260902-2luo5h_gates.log`.

| Command | Exit |
| --- | ---: |
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `gofmt -l .` | 0 (no output) |
| `go test ./... -count=1` | 0 |
| `go test ./... -cover -count=1` | 0 |
| `FuzzCanonicalizeRoundTrip` 100x | 0 |
| `FuzzObjectIdentityRepresentationInvariant` 100x | 0 |
| `FuzzClosedIdentityShapeRefusal` 100x | 0 |
| `FuzzObservationEventRefusal` 100x | 0 |
| `FuzzScalarProductionEntries` 100x | 0 |
| `tracecheck` | 0 |
| `cataloggen -check` | 0 |

No fuzz corpus file was added by any run; `git status --short` after the sweep
shows only the four changed sources.

## Scope

`internal/canonicaljson/closed_shapes.go`,
`internal/canonicaljson/clause_refusal_test.go`,
`internal/canonicaljson/boundary_constraints_test.go`,
`internal/canonicaljson/testdata/constraint-enumeration.md`, `README.md`
(the package enforcement claim now states the overlap refusal), `LOGBOOK.md`.
`gofmt -l .`, `go test ./... -count=1`, `tracecheck` and `cataloggen -check`
were re-run after the README and LOGBOOK edits and all exit 0; that rerun is
appended to the gates log.

## Coverage

| Package | Checkpoint `a031128` | This head | Delta |
| --- | ---: | ---: | ---: |
| `internal/canonicaljson` | 97.2% | 97.2% | 0.0 |

The baseline was measured in a throwaway worktree at `a031128`, not inherited
from a note.
