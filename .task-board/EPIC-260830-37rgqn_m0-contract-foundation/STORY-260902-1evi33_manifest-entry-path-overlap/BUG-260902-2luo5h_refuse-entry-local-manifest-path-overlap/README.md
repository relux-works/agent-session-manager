# BUG-260902-2luo5h: refuse-entry-local-manifest-path-overlap

## Description
Transfer Manifest entries are not checked for path overlap, so a symlink or file entry may have children. This is a path-safety hole at materialization time and it is on landed main.

Reproduced by probe against validateManifestEntries (main internal/canonicaljson/closed_shapes.go:742; the record-schema branch carries the same code at :1018). All of these are ACCEPTED today:
  file "a" together with file "a/b"
  symlink "s" together with file "s/x"
  file "a" together with directory "a/b"
Only a case-fold duplicate is refused.

SPEC.md:4768-4769 requires entries to contain no duplicate, OVERLAPPING, or destination-case-colliding path. The implementation covers duplicate and case-collision and drops overlap.

The artifact hides it: testdata/constraint-enumeration.md:212 on main (and :290 on the record-schema branch) narrows overlapping to the external child-partition case, so the row reads as enforced while the entry-local half is absent.

Mutation testing cannot surface this by construction - there is no clause to mutate. It was found by reading the spec sentence against the code, which is the only method that finds an absent clause.

A symlink parent over a real child is a materialization-escape primitive, so this is not cosmetic.

## Scope
Normative scope: §13.14 Transfer Manifest entries, SPEC.md:4766-4769.

## Acceptance Criteria
Entry-local overlap is refused at the production identity entries: an entry whose ancestor is a file, symlink or hardlink is rejected, as is a parent that is not a declared directory where the spec requires one. The enumeration row is corrected to state the whole rule rather than the child-partition half. Negative cases cover file-over-file, symlink-over-file and file-over-directory, and each reddens when only the overlap clause is weakened. The sorted order already computed is reused rather than re-deriving it.
