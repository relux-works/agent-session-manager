# TASK-260830-8x76g1 rev14 mutation summary

## Harness

The reviewer-supplied `genmutants.py` derived 71 mutants from the whole current
production file, including `maxBlobChunks`, `maxChunkSize`, every discovered
bound expression, and equality/zero guards. Four independently supplied
symlink-clause mutants were run separately. Each mutant invoked uncached
`go test ./internal/canonicaljson/ -count=1` and restored production source in a
`finally` block.

Final result: 60 killed, 15 raw survivors, zero actionable survivors. The
target `line614 drop chunk zero-size guard` is killed by
`TestTrailingBlobChunkSizeMinimumReachesBothIdentityEntries`.

## Raw survivor mechanisms

| Survivor(s) | Mechanism and public-entry proof |
| --- | --- |
| `line32 boardLogicalIDPattern {0,127}->{0,128}` and `line404 logical_id max 128->129` | Mutually redundant when applied alone. The existing 128/129-character case drives both public identity entries and kills the combined widening. |
| `line404 logical_id min 1->0` | `requireBoundedString` first calls `requireString`, which refuses the empty string. The public empty logical-ID case remains a refusal. |
| `line195 {0,63}->{0,64}` | The match is in diagnostic text, not the compiled Session Record name pattern; it cannot change acceptance. |
| `line582 totalSize > 0 -> > 1` | A one-byte descriptor without chunks still fails exact coverage (`covered != totalSize`) through both public entries. |
| `line596 <32 -> <31` | Mechanical match inside the `1<<32-1` literal narrows the uint32 ceiling; it cannot admit an invalid value. |
| `line596 uint32 max -> 1<<32` | `maxBlobChunks` and `chunkIndex == index` make index `2^32` publicly unreachable; the independent chunk-count/index boundaries remain pinned. |
| `line1479 extension-key len <3 -> <2` | `reverseDNSPattern` still requires at least `a.b`; malformed extension keys remain refused through both public entries. |
| `goal_id`, `tree_identity`, both `repository_identity` sites, and `GitRemote.name` minimum `1->0` | Each path first executes `requireString`, whose non-empty check subsumes the character-count minimum. The corresponding empty-value public-entry cases remain refusals. |
| Both `submodules 256->257` array bounds | The shared traversal counter refuses total count greater than 256. `TestTransferManifestSubmoduleTotalCountBoundary` drives both public entries and kills a widening of that shared cap. |

These are not declared as killed. They are reported honestly as surviving
mechanical mutants whose widened local expression cannot widen the composed
production gate, or as non-behavioral/narrowing mutations. The actionable
survivor count is zero.

## Evidence anomaly and remediation

An initial attempt split the sweep into four concurrent processes. Because the
reviewer harness mutates one shared production file, concurrency invalidates
causal attribution even though each process exited 0 and the final SHA matched.
Those logs are excluded from outcome archives. The entire 71-mutant set was
then rerun in one sequential foreground process against the final test tree,
followed by the four symlink mutants. Only the sequential logs are evidence.
