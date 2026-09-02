# STORY-260902-1evi33: manifest-entry-path-overlap

## Description
Transfer Manifest entries are not checked for path overlap, so a symlink or file entry may have children. On landed main, and a materialization-escape primitive.

## Scope
Normative scope: §13.14 Transfer Manifest entries, SPEC.md:4766-4769.

## Acceptance Criteria
Entry-local overlap is refused at the production identity entries, and the enumeration row states the whole rule rather than the child-partition half.
