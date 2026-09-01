# TASK-260830-8x76g1 mutation triage (RUN-260901-885c85)

## Refusal-clause sweep

The CR revision 14 harness enumerated 110 `return refusal(...)` sites in `internal/canonicaljson/closed_shapes.go`. On the exact current test tree, 85 site deletions are killed and the 25 raw survivors below are non-actionable because the deleted clause is unreachable through decoded public JSON or an independent production gate rejects the same malformed value.

| Site | Clause | Classification and proof |
| ---: | --- | --- |
| 13 / line 327 | missing `task_board` | Subsumed by the Session Record exact-member gate; recursive member deletion drives both public entries. |
| 17 / line 370 | missing `board_goal` | Subsumed by the Task Board Binding exact-member gate. |
| 21 / line 412 | missing `remote_url` | Subsumed by the Board Identity exact-member gate. |
| 24 / line 421 | remote URL type/UTF-8 | A non-string defaults to empty and is refused by URL grammar; invalid UTF-8 cannot pass `decodeStrict`. |
| 26 / line 462 | missing `fork_provenance` | Subsumed by the Session Record exact-member gate. |
| 30 / line 579 | empty blob has chunks | Final chunk coverage refuses the shape. |
| 31 / line 582 | non-empty blob has no chunks | Final exact coverage refuses the shape. |
| 36 / line 620 | uint53 addition overflow | Unreachable: maximum covered bytes are `32768 * 4194304 = 137438953472` (`2^37`), below `MaxUint53`. |
| 37 / line 624 | chunk exceeds total | The final `covered != totalSize` gate refuses it. |
| 74 / line 1435 | migration extension is not object | The schema validator's `requireObject` runs first. |
| 79 / line 1506 | invalid UTF-8 string value | Unreachable through public JSON because `decodeStrict` rejects invalid UTF-8 and unpaired surrogates. |
| 82 / line 1525 | invalid UTF-8 object key | Same public-decoder invariant. |
| 85 / line 1593 | path is not string | Empty default is refused by the production relative-path parser. |
| 87 / line 1609 | bool member missing | Subsumed by the enclosing exact-member gate. |
| 89 / line 1621 | nullable string member missing | Subsumed by the enclosing exact-member gate. |
| 90 / line 1628 | nullable string type/empty/UTF-8 | Downstream Git OID/ref grammar or tagged-union consistency refuses it; invalid UTF-8 is decoder-unreachable. |
| 93 / line 1714 | object member missing | Subsumed by the enclosing exact-member gate. |
| 95 / line 1730 | array member missing | Subsumed by the enclosing exact-member gate. |
| 98 / line 1748 | required string empty | Every caller applies exact grammar, a scalar validator, or a minimum length; the public-entry media-type case pins this path. |
| 99 / line 1756 | UTF-8 string member missing | Subsumed by the enclosing exact-member gate. |
| 102 / line 1790 | unsigned member missing | Subsumed by the enclosing exact-member gate. |
| 103 / line 1794 | unsigned member not number | The empty literal fails `strconv.ParseUint`. |
| 104 / line 1798 | unsigned literal fractional/negative | `strconv.ParseUint` independently refuses the literal. |
| 105 / line 1821 | nullable digest missing | Subsumed by the enclosing exact-member gate. |
| 106 / line 1845 | sorted digest element not string | Empty default fails the production digest parser. |

Site 94 (`requireObjectValue` type guard) appeared in the first post-change raw set and was killed by the exact-current rerun after the systematic subsumption test was added. The attached `final-survivor-rerun.tsv` is therefore the authoritative 25-row raw-survivor list.

## Numeric and bound sweep

The pinned rev12 `genmutants.py` generated 71 mutants. The regenerated JSON is byte-identical to the reviewer corpus (SHA-256 `f55bb35f5374357e9ee23e1b5a6b06df72c1293aaac73e336d8563684c7cf7bd`). Fifty-six mutants are killed. All fifteen raw survivors are independently constrained or non-behavioral:

| Mutant | Classification and proof |
| ---: | --- |
| 2 | Board logical-ID regex max +1 remains blocked by `boundedString(..., 128)`. |
| 4 | Changes only the regex text embedded in an error message; matching behavior is unchanged. |
| 11 | Raising the no-chunk precondition from size 0 to 1 is blocked by final exact coverage. |
| 12 | `< 32 -> < 31` narrows, rather than widens, an intermediate uint32 check and remains constrained by ordered indices and the chunk-count cap. |
| 24 | Extension-key minimum 3 -> 2 remains blocked by required reverse-DNS `a.b` grammar. |
| 35 | Board logical-ID bounded max 128 -> 129 remains blocked by the 128-character regex. |
| 36 | Board logical-ID bounded min 1 -> 0 remains blocked by `requireString` and regex grammar. |
| 38 | Goal-ID bounded min 1 -> 0 remains blocked by `requireString`. |
| 39 | BlobChunk index upper bound widened to `2^32`; the index must equal its slice position and the array has at most 32768 chunks. |
| 46 | Empty `tree_identity` remains blocked by `requireString`. |
| 49 | Empty workspace `repository_identity` remains blocked by `requireString`. |
| 50 | Workspace submodule array max 256 -> 257 remains blocked by the recursive total-submodule cap of 256. |
| 54 | Empty remote name remains blocked by `requireString`. |
| 60 | Empty submodule `repository_identity` remains blocked by `requireString`. |
| 62 | Nested submodule array max 256 -> 257 remains blocked by the recursive total-submodule cap of 256. |

No actionable mutant survived either sweep.
