# TASK-260830-8x76g1 rev14 logbook

## Recovery and source decision

Both supplied archives matched their declared SHA-256 values. The worktree was
already at the rev13 candidate, which contains the accepted-leaf restoration,
the audit carry-forward, and later reviewed corrections. Re-extracting the old
carry-forward over this tree would have reverted accepted later work. I kept the
newer candidate and independently compared the accepted catalog, cataloggen,
specpin, and seven scalar files with the accepted-leaves archive; all were
byte-identical.

## Rework decision

CR revision 13 explicitly prohibited production changes. The defect is missing
test causality, not incorrect production behavior. Rev14 therefore changes only
the boundary test and its per-member enumeration row. The malformed case uses a
full first chunk and a trailing zero-size second chunk so neither empty-object,
ordering, offset, non-final-size, nor coverage checks can refuse before the
`size == 0` clause.

The former `blob chunk size non-zero` case set descriptor size and chunk size to
zero together, so it actually exercised the empty-descriptor/chunks cross-field
rule. It was renamed and reshaped to prove that rule honestly.

## Expected-red result

Disabling only the zero-size guard made
`TestTrailingBlobChunkSizeMinimumReachesBothIdentityEntries` fail with exit 1:
`CalculateObjectIdentity` returned no error. This is the required evidence that
the new negative case fails when the attestation gate admits the forbidden
shape. The source was restored from a task-scoped byte copy, not from Git.

## Execution anomaly

The first full-sweep attempt used four concurrent range processes. That was an
evidence error: all processes mutate `closed_shapes.go`, so overlapping mutants
could create false kills. The production hash happened to restore correctly,
but those results were discarded. I reran all 71 derived mutants sequentially
against the final test tree, then ran the four symlink mutants. The valid final
result is 60 killed, 15 mechanistically non-actionable survivors, zero
actionable survivors, with byte-identical source restoration.

## Validation anomaly

`task-board validate` exited 0 while printing 262 inherited
`MISSING_ACTIVITY` diagnostics. None names `TASK-260830-8x76g1`; this is the same
board-wide inherited condition recorded in prior revisions and does not affect
the scoped candidate.
